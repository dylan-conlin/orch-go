// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

const (
	modelDriftLabel               = "model-maintenance"
	modelDriftIssueType           = "task"
	modelDriftSpawnThreshold      = 3
	modelDriftBackpressureLimit   = 3
	modelDriftCircuitBreakerLimit = 5
)

// ModelDriftResult contains the result of running model drift reflection.
type ModelDriftResult struct {
	Created []string
	Skipped int
	Message string
	Error   error
}

// ModelDriftIssueCreateArgs defines parameters for creating model drift issues.
type ModelDriftIssueCreateArgs struct {
	Title       string
	Description string
	Priority    int
	Labels      []string
}

// ModelDriftMetadata captures model attributes used for grouping and priority.
type ModelDriftMetadata struct {
	ModelPath   string
	Domain      string
	DomainKey   string
	LastUpdated string
	ProjectDir  string
}

type modelDriftAggregate struct {
	ModelPath  string
	Count      int
	ChangedSet map[string]struct{}
	DeletedSet map[string]struct{}
}

type modelDriftCandidate struct {
	ModelPath   string
	Count       int
	Changed     []string
	Deleted     []string
	CommitCount int
	Priority    int
	Domain      string
	DomainKey   string
}

type modelDriftGroup struct {
	Domain    string
	DomainKey string
	Priority  int
	Models    []modelDriftCandidate
}

// ShouldRunModelDriftReflection returns true if model drift reflection should run.
func (d *Daemon) ShouldRunModelDriftReflection() bool {
	if !d.Config.ReflectModelDriftEnabled || d.Config.ReflectModelDriftInterval <= 0 {
		return false
	}
	if d.lastModelDriftReflect.IsZero() {
		return true
	}
	return time.Since(d.lastModelDriftReflect) >= d.Config.ReflectModelDriftInterval
}

// RunPeriodicModelDriftReflection runs model drift reflection analysis if due.
// Returns the result if reflection was run, or nil if it wasn't due.
func (d *Daemon) RunPeriodicModelDriftReflection() *ModelDriftResult {
	if !d.ShouldRunModelDriftReflection() {
		return nil
	}

	reflectFunc := d.modelDriftReflectFunc
	if reflectFunc == nil {
		reflectFunc = d.RunModelDriftReflection
	}

	result, err := reflectFunc()
	if err != nil {
		if result == nil {
			return &ModelDriftResult{
				Error:   err,
				Message: fmt.Sprintf("Model drift reflection failed: %v", err),
			}
		}
		return result
	}

	if result != nil && result.Error == nil {
		d.lastModelDriftReflect = time.Now()
	}

	return result
}

// RunModelDriftReflection scans staleness events and creates model-maintenance issues.
func (d *Daemon) RunModelDriftReflection() (*ModelDriftResult, error) {
	reader := d.modelDriftEventReader
	if reader == nil {
		reader = readStalenessEvents
	}

	events, err := reader(spawn.DefaultStalenessEventPath())
	if err != nil {
		result := &ModelDriftResult{
			Error:   err,
			Message: fmt.Sprintf("Model drift reflection failed: %v", err),
		}
		return result, err
	}

	if len(events) == 0 {
		return &ModelDriftResult{Message: "Model drift reflection: no staleness events"}, nil
	}

	aggregates := map[string]*modelDriftAggregate{}
	for _, event := range events {
		modelPath := strings.TrimSpace(event.Model)
		if modelPath == "" {
			continue
		}
		agg := aggregates[modelPath]
		if agg == nil {
			agg = &modelDriftAggregate{
				ModelPath:  modelPath,
				ChangedSet: make(map[string]struct{}),
				DeletedSet: make(map[string]struct{}),
			}
			aggregates[modelPath] = agg
		}
		agg.Count++
		for _, file := range event.ChangedFiles {
			if file == "" {
				continue
			}
			agg.ChangedSet[file] = struct{}{}
		}
		for _, file := range event.DeletedFiles {
			if file == "" {
				continue
			}
			agg.DeletedSet[file] = struct{}{}
		}
	}

	var candidates []modelDriftCandidate
	for _, agg := range aggregates {
		if agg.Count < modelDriftSpawnThreshold {
			continue
		}
		changed := mapKeysSorted(agg.ChangedSet)
		deleted := mapKeysSorted(agg.DeletedSet)
		candidates = append(candidates, modelDriftCandidate{
			ModelPath: agg.ModelPath,
			Count:     agg.Count,
			Changed:   changed,
			Deleted:   deleted,
		})
	}

	if len(candidates) == 0 {
		return &ModelDriftResult{Message: "Model drift reflection: no models exceeded threshold"}, nil
	}

	listIssues := d.listIssuesWithLabelFunc
	if listIssues == nil {
		listIssues = ListIssuesWithLabel
	}
	openIssues, err := listIssues(modelDriftLabel)
	if err != nil {
		result := &ModelDriftResult{
			Error:   err,
			Message: fmt.Sprintf("Model drift reflection failed: %v", err),
		}
		return result, err
	}

	openCount := len(openIssues)
	if openCount >= modelDriftCircuitBreakerLimit {
		return &ModelDriftResult{
			Message: fmt.Sprintf("Model drift reflection halted: circuit breaker (%d open)", openCount),
		}, nil
	}
	if openCount >= modelDriftBackpressureLimit {
		return &ModelDriftResult{
			Message: fmt.Sprintf("Model drift reflection paused: %d open (max %d)", openCount, modelDriftBackpressureLimit),
		}, nil
	}

	existingModels := existingModelKeys(openIssues)
	existingDomains := existingDomainKeys(openIssues)

	loader := d.modelDriftMetadataLoader
	if loader == nil {
		loader = LoadModelDriftMetadata
	}
	commitCounter := d.modelDriftCommitCounter
	if commitCounter == nil {
		commitCounter = DefaultModelDriftCommitCounter
	}

	groups := map[string]*modelDriftGroup{}
	skipped := 0
	for _, candidate := range candidates {
		if isModelTracked(candidate.ModelPath, existingModels) {
			skipped++
			continue
		}

		metadata, metaErr := loader(candidate.ModelPath)
		if metaErr != nil {
			metadata = fallbackModelMetadata(candidate.ModelPath)
		}
		if metadata.Domain == "" {
			metadata.Domain = deriveDomainFromPath(candidate.ModelPath)
		}
		if metadata.DomainKey == "" {
			metadata.DomainKey = normalizeDomainKey(metadata.Domain)
		}

		commitCount, err := commitCounter(metadata.ProjectDir, metadata.LastUpdated, candidate.Changed)
		if err != nil {
			commitCount = len(candidate.Changed)
		}
		candidate.CommitCount = commitCount
		candidate.Domain = metadata.Domain
		candidate.DomainKey = metadata.DomainKey
		candidate.Priority = modelDriftPriority(candidate)

		group := groups[candidate.DomainKey]
		if group == nil {
			group = &modelDriftGroup{
				Domain:    metadata.Domain,
				DomainKey: metadata.DomainKey,
				Priority:  candidate.Priority,
			}
			groups[candidate.DomainKey] = group
		}
		if candidate.Priority < group.Priority {
			group.Priority = candidate.Priority
		}
		group.Models = append(group.Models, candidate)
	}

	if len(groups) == 0 {
		return &ModelDriftResult{Message: "Model drift reflection: no new models to file"}, nil
	}

	var ordered []*modelDriftGroup
	for _, group := range groups {
		ordered = append(ordered, group)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Priority != ordered[j].Priority {
			return ordered[i].Priority < ordered[j].Priority
		}
		return ordered[i].Domain < ordered[j].Domain
	})

	createIssue := d.createModelDriftIssueFunc
	if createIssue == nil {
		createIssue = DefaultCreateModelDriftIssue
	}

	allowed := modelDriftBackpressureLimit - openCount
	created := []string{}
	for _, group := range ordered {
		if len(created) >= allowed {
			break
		}
		if existingDomains[group.DomainKey] {
			skipped++
			continue
		}
		issueID, err := createIssue(ModelDriftIssueCreateArgs{
			Title:       fmt.Sprintf("Model drift: %s", group.Domain),
			Description: formatModelDriftIssueDescription(group),
			Priority:    group.Priority,
			Labels:      []string{modelDriftLabel},
		})
		if err != nil {
			result := &ModelDriftResult{
				Created: created,
				Skipped: skipped,
				Error:   err,
				Message: fmt.Sprintf("Model drift issue creation failed: %v", err),
			}
			return result, err
		}
		created = append(created, issueID)
	}

	message := fmt.Sprintf("Model drift reflection: created %d issue(s), skipped %d (open: %d)", len(created), skipped, openCount)
	return &ModelDriftResult{
		Created: created,
		Skipped: skipped,
		Message: message,
	}, nil
}

// DefaultCreateModelDriftIssue creates a model maintenance issue in beads.
func DefaultCreateModelDriftIssue(args ModelDriftIssueCreateArgs) (string, error) {
	if strings.TrimSpace(args.Title) == "" {
		return "", fmt.Errorf("model drift issue title is required")
	}
	labels := args.Labels
	if len(labels) == 0 {
		labels = []string{modelDriftLabel}
	}
	priority := args.Priority
	if priority == 0 {
		priority = 3
	}
	issue, err := createBeadsIssue(args.Title, args.Description, modelDriftIssueType, priority, labels)
	if err != nil {
		return "", err
	}
	return issue.ID, nil
}

func createBeadsIssue(title, description, issueType string, priority int, labels []string) (*beads.Issue, error) {
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			issue, err := client.Create(&beads.CreateArgs{
				Title:       title,
				Description: description,
				IssueType:   issueType,
				Priority:    priority,
				Labels:      labels,
			})
			if err == nil {
				return issue, nil
			}
		}
	}

	return beads.FallbackCreate(title, description, issueType, priority, labels)
}

func readStalenessEvents(path string) ([]spawn.StalenessEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []spawn.StalenessEvent{}, nil
		}
		return nil, fmt.Errorf("failed to open staleness events file: %w", err)
	}
	defer file.Close()

	var events []spawn.StalenessEvent
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event spawn.StalenessEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read staleness events file: %w", err)
	}

	return events, nil
}

// LoadModelDriftMetadata loads domain and last updated metadata from a model file.
func LoadModelDriftMetadata(modelPath string) (ModelDriftMetadata, error) {
	normalized, err := normalizeModelPath(modelPath)
	if err != nil {
		return fallbackModelMetadata(modelPath), err
	}

	content, err := os.ReadFile(normalized)
	if err != nil {
		return fallbackModelMetadata(modelPath), err
	}

	text := string(content)
	lastUpdated := extractModelField(text, "Last Updated")
	domain := normalizeDomainName(extractModelField(text, "Domain"))
	if domain == "" {
		domain = deriveDomainFromPath(normalized)
	}

	projectDir := projectDirFromModelPath(normalized)
	metadata := ModelDriftMetadata{
		ModelPath:   normalized,
		Domain:      domain,
		DomainKey:   normalizeDomainKey(domain),
		LastUpdated: lastUpdated,
		ProjectDir:  projectDir,
	}
	return metadata, nil
}

// DefaultModelDriftCommitCounter counts commits since last update across given files.
func DefaultModelDriftCommitCounter(projectDir, lastUpdated string, files []string) (int, error) {
	if lastUpdated == "" || len(files) == 0 {
		return 0, nil
	}
	if projectDir == "" {
		projectDir = "."
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	args := []string{"log", "--since=" + lastUpdated, "--oneline", "--"}
	args = append(args, files...)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count, nil
}

func modelDriftPriority(candidate modelDriftCandidate) int {
	if len(candidate.Deleted) > 0 || candidate.CommitCount >= 5 {
		return 2
	}
	return 3
}

func formatModelDriftIssueDescription(group *modelDriftGroup) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Model drift detected for domain \"%s\".\n\n", group.Domain))
	builder.WriteString(fmt.Sprintf("Models (>= %d stale spawns):\n", modelDriftSpawnThreshold))
	for _, model := range group.Models {
		builder.WriteString(fmt.Sprintf("- %s (stale spawns: %d, commits since update: %d", model.ModelPath, model.Count, model.CommitCount))
		if len(model.Deleted) > 0 {
			builder.WriteString(fmt.Sprintf(", deleted files: %d", len(model.Deleted)))
		}
		if len(model.Changed) > 0 {
			builder.WriteString(fmt.Sprintf(", changed files: %d", len(model.Changed)))
		}
		builder.WriteString(")\n")
		if len(model.Deleted) > 0 {
			builder.WriteString(fmt.Sprintf("  Deleted: %s\n", formatLimitedList(model.Deleted, 5)))
		}
	}
	builder.WriteString("\nSource: ")
	builder.WriteString(spawn.DefaultStalenessEventPath())
	builder.WriteString("\n")
	return builder.String()
}

func mapKeysSorted(input map[string]struct{}) []string {
	if len(input) == 0 {
		return []string{}
	}
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func formatLimitedList(values []string, limit int) string {
	if len(values) == 0 {
		return ""
	}
	if limit <= 0 || len(values) <= limit {
		return strings.Join(values, ", ")
	}
	return fmt.Sprintf("%s, +%d more", strings.Join(values[:limit], ", "), len(values)-limit)
}

func existingModelKeys(issues []Issue) map[string]struct{} {
	keys := make(map[string]struct{})
	for _, issue := range issues {
		for _, modelPath := range extractModelPathsFromText(issue.Title + "\n" + issue.Description) {
			for _, key := range modelKeyVariants(modelPath) {
				keys[key] = struct{}{}
			}
		}
	}
	return keys
}

func existingDomainKeys(issues []Issue) map[string]bool {
	keys := make(map[string]bool)
	for _, issue := range issues {
		domainKey := domainKeyFromIssue(issue)
		if domainKey != "" {
			keys[domainKey] = true
		}
	}
	return keys
}

func isModelTracked(modelPath string, existing map[string]struct{}) bool {
	for _, key := range modelKeyVariants(modelPath) {
		if _, ok := existing[key]; ok {
			return true
		}
	}
	return false
}

func modelKeyVariants(modelPath string) []string {
	cleaned := filepath.Clean(modelPath)
	base := filepath.Base(cleaned)
	base = strings.TrimSpace(base)
	baseNoExt := strings.TrimSuffix(base, filepath.Ext(base))

	rel := cleaned
	marker := string(filepath.Separator) + ".kb" + string(filepath.Separator)
	if idx := strings.Index(cleaned, marker); idx != -1 {
		rel = cleaned[idx+1:]
	}

	variants := []string{
		strings.ToLower(cleaned),
		strings.ToLower(rel),
		strings.ToLower(base),
		strings.ToLower(baseNoExt),
	}
	return variants
}

func extractModelPathsFromText(text string) []string {
	pattern := regexp.MustCompile(`(?i)(?:/[^\s]+/\.kb/models/[^\s]+\.md|\.kb/models/[^\s]+\.md)`)
	matches := pattern.FindAllString(text, -1)
	seen := make(map[string]struct{})
	var results []string
	for _, match := range matches {
		cleaned := strings.TrimSpace(match)
		if cleaned == "" {
			continue
		}
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		results = append(results, cleaned)
	}
	return results
}

func domainKeyFromIssue(issue Issue) string {
	trimmed := strings.TrimSpace(issue.Title)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "model drift:") {
		domain := strings.TrimSpace(trimmed[len("model drift:"):])
		return normalizeDomainKey(domain)
	}
	return ""
}

func normalizeModelPath(modelPath string) (string, error) {
	if modelPath == "" {
		return "", fmt.Errorf("empty model path")
	}
	if filepath.IsAbs(modelPath) {
		return filepath.Clean(modelPath), nil
	}
	return filepath.Abs(modelPath)
}

func normalizeDomainName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "/") {
		parts := strings.Split(value, "/")
		value = strings.TrimSpace(parts[0])
	}
	if strings.Contains(value, ",") {
		parts := strings.Split(value, ",")
		value = strings.TrimSpace(parts[0])
	}
	return value
}

func normalizeDomainKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "/", "-")
	re := regexp.MustCompile(`[^a-z0-9\-]`)
	value = re.ReplaceAllString(value, "")
	value = strings.Trim(value, "-")
	return value
}

func deriveDomainFromPath(modelPath string) string {
	base := filepath.Base(modelPath)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	return base
}

func fallbackModelMetadata(modelPath string) ModelDriftMetadata {
	domain := deriveDomainFromPath(modelPath)
	return ModelDriftMetadata{
		ModelPath:  modelPath,
		Domain:     domain,
		DomainKey:  normalizeDomainKey(domain),
		ProjectDir: projectDirFromModelPath(modelPath),
	}
}

func projectDirFromModelPath(modelPath string) string {
	cleaned := filepath.Clean(modelPath)
	marker := string(filepath.Separator) + ".kb" + string(filepath.Separator)
	if idx := strings.Index(cleaned, marker); idx != -1 {
		return cleaned[:idx]
	}
	return filepath.Dir(cleaned)
}

func extractModelField(content, field string) string {
	lines := strings.Split(content, "\n")
	plainPrefix := field + ":"
	boldPrefix := "**" + field + ":**"
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, boldPrefix):
			return strings.TrimSpace(strings.TrimPrefix(trimmed, boldPrefix))
		case strings.HasPrefix(trimmed, plainPrefix):
			return strings.TrimSpace(strings.TrimPrefix(trimmed, plainPrefix))
		}
	}
	return ""
}
