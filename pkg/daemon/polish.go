package daemon

import (
	"fmt"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

const (
	defaultPolishIssueType = "task"
	defaultPolishPriority  = 3
)

// PolishIssueSpec describes a polish follow-up issue to create.
type PolishIssueSpec struct {
	Audit       string
	DedupLabel  string
	Title       string
	Description string
	IssueType   string
	Priority    int
	Labels      []string
}

// PolishResult contains the result of one polish-mode run.
type PolishResult struct {
	Candidates      int
	Created         int
	Skipped         int
	CreatedIssueIDs []string
	DailyRemaining  int
	Message         string
	Error           error
}

// ShouldRunPolish returns true if polish mode should run based on configuration
// and interval state.
func (d *Daemon) ShouldRunPolish() bool {
	if !d.Config.PolishEnabled || d.Config.PolishInterval <= 0 {
		return false
	}
	if d.lastPolish.IsZero() {
		return true
	}
	return time.Since(d.lastPolish) >= d.Config.PolishInterval
}

// LastPolishTime returns when polish mode last ran.
func (d *Daemon) LastPolishTime() time.Time {
	return d.lastPolish
}

// RunPolish runs idle-time polish audits and creates low-priority follow-up
// issues in triage:review.
// Returns nil if polish is not due.
func (d *Daemon) RunPolish(projectDir string) *PolishResult {
	if !d.ShouldRunPolish() {
		return nil
	}

	now := time.Now().UTC()
	maxPerCycle := d.Config.PolishMaxIssuesPerCycle
	if maxPerCycle <= 0 {
		maxPerCycle = 3
	}
	maxPerDay := d.Config.PolishMaxIssuesPerDay
	if maxPerDay <= 0 {
		maxPerDay = 10
	}

	d.resetPolishDailyWindow(now)

	remainingDaily := maxPerDay - d.polishCreatedToday
	if remainingDaily <= 0 {
		d.lastPolish = now
		return &PolishResult{
			DailyRemaining: 0,
			Message:        fmt.Sprintf("Polish skipped: daily cap reached (%d/%d)", d.polishCreatedToday, maxPerDay),
		}
	}

	collect := d.collectPolishCandidatesFunc
	if collect == nil {
		collect = d.collectPolishCandidates
	}

	candidates, err := collect(projectDir)
	if err != nil {
		d.lastPolish = now
		return &PolishResult{
			DailyRemaining: remainingDaily,
			Error:          err,
			Message:        fmt.Sprintf("Polish failed while collecting candidates: %v", err),
		}
	}

	result := &PolishResult{
		Candidates:     len(candidates),
		DailyRemaining: remainingDaily,
	}

	if len(candidates) == 0 {
		d.lastPolish = now
		result.Message = "Polish found no follow-up work"
		return result
	}

	listAll := d.listAllIssuesFunc
	if listAll == nil {
		listAll = ListOpenAndInProgressIssues
	}
	activeIssues, err := listAll()
	if err != nil {
		d.lastPolish = now
		result.Error = err
		result.Message = fmt.Sprintf("Polish failed while checking existing issues: %v", err)
		return result
	}

	openDedupLabels := make(map[string]bool)
	for _, issue := range activeIssues {
		for _, label := range issue.Labels {
			if strings.HasPrefix(label, "polish:") {
				openDedupLabels[label] = true
			}
		}
	}

	createIssue := d.createPolishIssueFunc
	if createIssue == nil {
		createIssue = d.createPolishIssue
	}

	cycleLimit := maxPerCycle
	if remainingDaily < cycleLimit {
		cycleLimit = remainingDaily
	}

	for _, candidate := range candidates {
		if result.Created >= cycleLimit {
			break
		}

		if candidate.DedupLabel != "" && openDedupLabels[candidate.DedupLabel] {
			result.Skipped++
			continue
		}

		issueID, createErr := createIssue(candidate)
		if createErr != nil {
			result.Skipped++
			if d.Config.Verbose {
				fmt.Printf("  Polish create failed for %s: %v\n", candidate.Audit, createErr)
			}
			continue
		}

		result.Created++
		result.CreatedIssueIDs = append(result.CreatedIssueIDs, issueID)
		if candidate.DedupLabel != "" {
			openDedupLabels[candidate.DedupLabel] = true
		}
	}

	d.polishCreatedToday += result.Created
	result.DailyRemaining = maxPerDay - d.polishCreatedToday
	d.lastPolish = now

	if result.Created == 0 {
		if result.Skipped > 0 {
			result.Message = "Polish found work but skipped issue creation (dedup/caps)"
		} else {
			result.Message = "Polish found no actionable follow-up work"
		}
		return result
	}

	result.Message = fmt.Sprintf("Polish created %d issue(s)", result.Created)
	return result
}

func (d *Daemon) collectPolishCandidates(projectDir string) ([]PolishIssueSpec, error) {
	var specs []PolishIssueSpec

	metadataSpec, err := d.buildMetadataPolishIssue()
	if err != nil {
		if d.Config.Verbose {
			fmt.Printf("  Polish metadata audit failed: %v\n", err)
		}
	} else if metadataSpec != nil {
		specs = append(specs, *metadataSpec)
	}

	knowledgeSpecs, err := d.buildKnowledgePolishIssues()
	if err != nil {
		if d.Config.Verbose {
			fmt.Printf("  Polish knowledge audit failed: %v\n", err)
		}
	} else {
		specs = append(specs, knowledgeSpecs...)
	}

	qualitySpec, err := d.buildQualityPolishIssue(projectDir)
	if err != nil {
		if d.Config.Verbose {
			fmt.Printf("  Polish quality audit failed: %v\n", err)
		}
	} else if qualitySpec != nil {
		specs = append(specs, *qualitySpec)
	}

	return specs, nil
}

func (d *Daemon) buildMetadataPolishIssue() (*PolishIssueSpec, error) {
	listAll := d.listAllIssuesFunc
	if listAll == nil {
		listAll = ListOpenAndInProgressIssues
	}

	issues, err := listAll()
	if err != nil {
		return nil, err
	}

	missingArea := make([]string, 0)
	for _, issue := range issues {
		if hasLabelPrefix(issue.Labels, "polish:") {
			continue
		}
		if !beads.HasAreaLabel(issue.Labels) {
			missingArea = append(missingArea, issue.ID)
		}
	}

	if len(missingArea) == 0 {
		return nil, nil
	}

	sample := strings.Join(missingArea[:minInt(len(missingArea), 5)], ", ")
	description := fmt.Sprintf(
		"Daemon polish metadata audit found %d active issues without area labels.\n\nSample issue IDs: %s\n\nAdd appropriate area:* labels and note any taxonomy gaps.",
		len(missingArea),
		sample,
	)

	return &PolishIssueSpec{
		Audit:       "metadata",
		DedupLabel:  "polish:metadata",
		Title:       "Polish: add missing area labels on active issues",
		Description: description,
		IssueType:   defaultPolishIssueType,
		Priority:    defaultPolishPriority,
		Labels: []string{
			"polish:metadata",
		},
	}, nil
}

func (d *Daemon) buildKnowledgePolishIssues() ([]PolishIssueSpec, error) {
	runReflect := d.reflectFunc
	if runReflect == nil {
		runReflect = DefaultRunReflection
	}

	result, err := runReflect(false)
	if err != nil {
		return nil, err
	}
	if result == nil || result.Suggestions == nil {
		return nil, nil
	}

	var specs []PolishIssueSpec

	if len(result.Suggestions.Synthesis) > 0 {
		topics := make([]string, 0, minInt(len(result.Suggestions.Synthesis), 3))
		for _, s := range result.Suggestions.Synthesis[:minInt(len(result.Suggestions.Synthesis), 3)] {
			topics = append(topics, fmt.Sprintf("%s (%d investigations)", s.Topic, s.Count))
		}
		description := fmt.Sprintf(
			"kb reflect found %d synthesis opportunity clusters.\n\nTop clusters: %s\n\nConsolidate these clusters into model/decision updates and close redundant investigations.",
			len(result.Suggestions.Synthesis),
			strings.Join(topics, "; "),
		)
		specs = append(specs, PolishIssueSpec{
			Audit:       "knowledge-synthesis",
			DedupLabel:  "polish:knowledge-synthesis",
			Title:       "Polish: consolidate investigation clusters from kb reflect",
			Description: description,
			IssueType:   defaultPolishIssueType,
			Priority:    defaultPolishPriority,
			Labels: []string{
				"polish:knowledge",
				"polish:knowledge-synthesis",
			},
		})
	}

	if len(result.Suggestions.Stale) > 0 {
		samples := make([]string, 0, minInt(len(result.Suggestions.Stale), 5))
		for _, stale := range result.Suggestions.Stale[:minInt(len(result.Suggestions.Stale), 5)] {
			samples = append(samples, stale.Path)
		}
		description := fmt.Sprintf(
			"kb reflect flagged %d stale decisions with low citation activity.\n\nSample files: %s\n\nReview for relevance, supersede outdated decisions, and link surviving decisions in active docs.",
			len(result.Suggestions.Stale),
			strings.Join(samples, "; "),
		)
		specs = append(specs, PolishIssueSpec{
			Audit:       "knowledge-stale",
			DedupLabel:  "polish:knowledge-stale",
			Title:       "Polish: review stale decisions from kb reflect",
			Description: description,
			IssueType:   defaultPolishIssueType,
			Priority:    defaultPolishPriority,
			Labels: []string{
				"polish:knowledge",
				"polish:knowledge-stale",
			},
		})
	}

	if len(result.Suggestions.Drift) > 0 {
		description := fmt.Sprintf(
			"kb reflect detected %d potential constraint drifts.\n\nValidate whether current code/workflows still match documented constraints and update the stale artifact or implementation.",
			len(result.Suggestions.Drift),
		)
		specs = append(specs, PolishIssueSpec{
			Audit:       "knowledge-drift",
			DedupLabel:  "polish:knowledge-drift",
			Title:       "Polish: reconcile drifted constraints from kb reflect",
			Description: description,
			IssueType:   defaultPolishIssueType,
			Priority:    defaultPolishPriority,
			Labels: []string{
				"polish:knowledge",
				"polish:knowledge-drift",
			},
		})
	}

	return specs, nil
}

func (d *Daemon) buildQualityPolishIssue(projectDir string) (*PolishIssueSpec, error) {
	checker := d.HotspotChecker
	if checker == nil {
		checker = NewGitHotspotChecker()
	}

	hotspots, err := checker.CheckHotspots(projectDir)
	if err != nil {
		return nil, err
	}
	if len(hotspots) == 0 {
		return nil, nil
	}

	parts := make([]string, 0, minInt(len(hotspots), 5))
	for _, hotspot := range hotspots[:minInt(len(hotspots), 5)] {
		parts = append(parts, fmt.Sprintf("%s [%d] (%s)", hotspot.Path, hotspot.Score, hotspot.Type))
	}

	description := fmt.Sprintf(
		"Hotspot analysis found %d code quality hotspots.\n\nTop hotspots: %s\n\nRun a focused codebase audit and file targeted follow-ups for the highest-risk areas.",
		len(hotspots),
		strings.Join(parts, "; "),
	)

	return &PolishIssueSpec{
		Audit:       "quality",
		DedupLabel:  "polish:quality",
		Title:       "Polish: run codebase audit for hotspot areas",
		Description: description,
		IssueType:   defaultPolishIssueType,
		Priority:    defaultPolishPriority,
		Labels: []string{
			"polish:quality",
		},
	}, nil
}

func (d *Daemon) createPolishIssue(spec PolishIssueSpec) (string, error) {
	issueType := spec.IssueType
	if issueType == "" {
		issueType = defaultPolishIssueType
	}
	priority := spec.Priority
	if priority <= 0 {
		priority = defaultPolishPriority
	}

	labels := normalizePolishLabels(spec)

	var createdID string
	err := beads.Do("", func(client *beads.Client) error {
		if connErr := client.Connect(); connErr != nil {
			return connErr
		}
		defer client.Close()

		issue, createErr := client.Create(&beads.CreateArgs{
			Title:       spec.Title,
			Description: spec.Description,
			IssueType:   issueType,
			Priority:    priority,
			Labels:      labels,
		})
		if createErr != nil {
			return createErr
		}
		createdID = issue.ID
		return nil
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return createdID, nil
	}

	issue, fallbackErr := beads.FallbackCreate(spec.Title, spec.Description, issueType, priority, labels)
	if fallbackErr != nil {
		return "", fallbackErr
	}
	return issue.ID, nil
}

func normalizePolishLabels(spec PolishIssueSpec) []string {
	labels := []string{"triage:review", "area:daemon"}
	if spec.DedupLabel != "" {
		labels = append(labels, spec.DedupLabel)
	}
	labels = append(labels, spec.Labels...)

	seen := make(map[string]struct{}, len(labels))
	unique := make([]string, 0, len(labels))
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		if _, exists := seen[label]; exists {
			continue
		}
		seen[label] = struct{}{}
		unique = append(unique, label)
	}

	return unique
}

// ListOpenAndInProgressIssues lists active issues used by polish audits.
func ListOpenAndInProgressIssues() ([]Issue, error) {
	statuses := []string{"open", "in_progress"}
	combined := make([]Issue, 0)
	seen := make(map[string]struct{})

	for _, status := range statuses {
		issues, err := listIssuesByStatus(status)
		if err != nil {
			return nil, err
		}
		for _, issue := range issues {
			if _, exists := seen[issue.ID]; exists {
				continue
			}
			seen[issue.ID] = struct{}{}
			combined = append(combined, issue)
		}
	}

	return combined, nil
}

func listIssuesByStatus(status string) ([]Issue, error) {
	var issues []Issue
	err := beads.Do("", func(client *beads.Client) error {
		if connErr := client.Connect(); connErr != nil {
			return connErr
		}
		defer client.Close()

		listed, listErr := client.List(&beads.ListArgs{Status: status, Limit: 0})
		if listErr != nil {
			return listErr
		}
		issues = convertBeadsIssues(listed)
		return nil
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return issues, nil
	}

	fallbackIssues, fallbackErr := beads.FallbackList(status)
	if fallbackErr != nil {
		return nil, fallbackErr
	}
	return convertBeadsIssues(fallbackIssues), nil
}

func (d *Daemon) resetPolishDailyWindow(now time.Time) {
	windowStart := now.UTC().Truncate(24 * time.Hour)
	if d.polishWindowStart.IsZero() || !d.polishWindowStart.Equal(windowStart) {
		d.polishWindowStart = windowStart
		d.polishCreatedToday = 0
	}
}

func hasLabelPrefix(labels []string, prefix string) bool {
	for _, label := range labels {
		if strings.HasPrefix(label, prefix) {
			return true
		}
	}
	return false
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
