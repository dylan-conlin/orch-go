// Package main provides the orient command for session start orientation.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/claims"
	"github.com/dylan-conlin/orch-go/pkg/compose"
	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/focus"
	"github.com/dylan-conlin/orch-go/pkg/health"
	"github.com/dylan-conlin/orch-go/pkg/kbmetrics"
	"github.com/dylan-conlin/orch-go/pkg/orient"
	"github.com/dylan-conlin/orch-go/pkg/thread"
	"github.com/spf13/cobra"
)

var (
	orientDays      int
	orientJSON      bool
	orientSkipReady bool
	orientHook      bool
)

var (
	orientComposeTimeout  = 2 * time.Second
	orientComposeFunc     = compose.Compose
	orientWriteDigestFunc = compose.WriteDigest
)

var orientCmd = &cobra.Command{
	Use:   "orient",
	Short: "Session start orientation with throughput baseline and model surfacing",
	Long: `Produce structured session orientation for the orchestrator to present
conversationally at session start. Surfaces:

  - Recent throughput (completions, abandonments, avg duration)
  - Previous session summary (from latest debrief in .kb/sessions/)
  - Ready work from beads (bd ready)
  - Active coordination plans from .kb/plans/
  - Active living threads from .kb/threads/ (open, updated within 7 days)
  - Relevant models matching ready work
  - Stale model warnings (>14 days without probes)
  - Current focus (if set)

Designed for orchestrator consumption, not direct human use.

Examples:
  orch orient              # Default orientation (last 1 day)
  orch orient --days 3     # Throughput from last 3 days
  orch orient --json       # JSON output for programmatic consumption
  orch orient --skip-ready # Skip ready issues (when frontier covers them)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runOrient()
	},
}

func init() {
	orientCmd.Flags().IntVar(&orientDays, "days", 1, "Number of days for throughput analysis")
	orientCmd.Flags().BoolVar(&orientJSON, "json", false, "Output as JSON for programmatic consumption")
	orientCmd.Flags().BoolVar(&orientSkipReady, "skip-ready", false, "Skip ready issues collection (use when frontier provides them)")
	orientCmd.Flags().BoolVar(&orientHook, "hook", false, "Wrap output in SessionStart hook JSON envelope")
}

func runOrient() error {
	now := time.Now()
	projectDir, _ := os.Getwd()

	data := &orient.OrientationData{}

	// === THINKING SURFACE DISPLAY (threads, briefs, tensions) ===
	// Additional fields below are still collected for --json and other consumers.

	// Previous session (needed for insight and changelog date)
	sessionsDir := filepath.Join(projectDir, ".kb", "sessions")
	data.PreviousSession = collectPreviousSession(sessionsDir)

	// Element 1: Active threads
	data.ActiveThreads = collectActiveThreads(projectDir)

	// Element 1b: Promotion-ready threads
	data.PromotionReady = collectPromotionReady(projectDir)

	// Element 2: Recent briefs
	briefsDir := filepath.Join(projectDir, ".kb", "briefs")
	digestsDir := orientDigestsDir(projectDir)
	readState := loadBriefReadStateForOrient(projectDir)
	data.RecentBriefs, data.UnreadBriefCount = orient.ScanRecentBriefs(briefsDir, readState, 5)
	data.ComposeSummary = collectComposeSummary(briefsDir, orientThreadsDir(projectDir), digestsDir, readState)

	// Between-session digests
	var prevSessionDate time.Time
	if data.PreviousSession != nil && data.PreviousSession.Date != "" {
		prevSessionDate, _ = time.Parse("2006-01-02", data.PreviousSession.Date)
	}
	data.DigestSummary = orient.ScanRecentDigests(digestsDir, prevSessionDate)
	if data.DigestSummary != nil {
		data.DigestSummary.MaintenanceCount = orient.CountMaintenanceBriefs(briefsDir, prevSessionDate)
	}

	// Element 3: Active tensions (claim edges from models)
	modelsDir := filepath.Join(projectDir, ".kb", "models")
	data.ClaimEdges = collectClaimEdges(modelsDir, now)

	// Ready work
	if !orientSkipReady {
		data.ReadyIssues = collectReadyIssues()
		enrichIssuesWithKBContext(data.ReadyIssues)
	}

	// Active plans
	plansDir := filepath.Join(projectDir, ".kb", "plans")
	activePlans, err := orient.ScanActivePlans(plansDir)
	if err == nil && len(activePlans) > 0 {
		if statusMap := queryPlanBeadsStatuses(activePlans); len(statusMap) > 0 {
			orient.ApplyBeadsProgress(activePlans, statusMap)
		}
		data.ActivePlans = activePlans
	}

	// Focus
	data.FocusGoal = collectFocus()

	// Context
	data.ConfigDrift = collectConfigDrift()
	data.SessionResume = collectSessionResume()

	// === OPERATIONAL (collected for --json backward compat, rendered by FormatHealth) ===

	data.Throughput = collectThroughput(now)
	data.Throughput.InProgress = collectInProgressCount()
	enrichThroughputWithGitGroundTruth(&data.Throughput)

	allModels, err := orient.ScanModelFreshness(modelsDir)
	if err == nil {
		data.RelevantModels = selectRelevantModels(allModels, 3)
		data.StaleModels = orient.FilterStaleModels(allModels, 2)
	}

	data.Changelog = collectChangelog(data.PreviousSession)
	data.HealthSummary = collectHealthSummary()
	data.DaemonHealth = collectDaemonHealth(now)
	data.ReflectSummary = collectReflectSuggestions()
	enrichReflectWithSessionOrphans(data.ReflectSummary, data.PreviousSession, projectDir)
	data.DivergenceAlerts = computeDivergenceAlerts(data)
	data.ExploreCandidates = collectExploreCandidates(projectDir, modelsDir, now)
	data.AdoptionDrift = collectAdoptionDrift(projectDir)

	if orientHook {
		// Wrap in SessionStart hook JSON envelope
		text := orient.FormatOrientation(data)

		// Inject orchestrator skill content when CLAUDE_CONTEXT=orchestrator
		if os.Getenv("CLAUDE_CONTEXT") == "orchestrator" {
			skillContent := loadOrchSkillContent()
			if skillContent != "" {
				text = skillContent + "\n\n" + text
			}
		}

		envelope := map[string]interface{}{
			"hookSpecificOutput": map[string]interface{}{
				"hookEventName":     "SessionStart",
				"additionalContext": text,
			},
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(envelope)
	}

	if orientJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}

	fmt.Print(orient.FormatOrientation(data))
	return nil
}

// collectThroughput reads events.jsonl and computes throughput metrics,
// scoped to the current project via .beads/config.yaml issue-prefix.
func collectThroughput(now time.Time) orient.Throughput {
	home, err := os.UserHomeDir()
	if err != nil {
		return orient.Throughput{}
	}

	eventsPath := filepath.Join(home, ".orch", "events.jsonl")
	events, err := parseOrientEvents(eventsPath)
	if err != nil {
		return orient.Throughput{}
	}

	prefix := readBeadsPrefix()
	return orient.ComputeThroughput(events, now, orientDays, prefix)
}

// readBeadsPrefix reads the issue-prefix from .beads/config.yaml in the working directory.
// Returns empty string if not found (no filtering will be applied).
func readBeadsPrefix() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(dir, ".beads", "config.yaml"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "issue-prefix:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "issue-prefix:"))
		}
	}
	return ""
}

// parseOrientEvents reads events.jsonl into orient.Event slice.
func parseOrientEvents(path string) ([]orient.Event, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var events []orient.Event
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var event orient.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		events = append(events, event)
	}

	return events, scanner.Err()
}

// collectReadyIssues runs `bd ready` and parses the output.
func collectReadyIssues() []orient.ReadyIssue {
	cmd := exec.Command("bd", "ready")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	return parseBdReadyForOrient(string(output), 3)
}

// parseBdReadyForOrient parses bd ready output into ReadyIssue slice, limited to maxCount.
func parseBdReadyForOrient(output string, maxCount int) []orient.ReadyIssue {
	var issues []orient.ReadyIssue
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if len(issues) >= maxCount {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "\U0001F4CB") || strings.HasPrefix(line, "No ") {
			continue
		}
		// Match numbered lines like: "1. [P2] [feature] orch-go-xwh: Title here"
		if len(line) < 3 || line[0] < '0' || line[0] > '9' {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}

		priority := strings.Trim(parts[1], "[]")
		var beadsID, title string
		for i := 2; i < len(parts); i++ {
			if strings.HasSuffix(parts[i], ":") {
				beadsID = strings.TrimSuffix(parts[i], ":")
				if i+1 < len(parts) {
					title = strings.Join(parts[i+1:], " ")
				}
				break
			}
		}
		if beadsID != "" {
			issues = append(issues, orient.ReadyIssue{
				ID:       beadsID,
				Title:    title,
				Priority: priority,
			})
		}
	}

	return issues
}

// selectRelevantModels picks the top N freshest non-stale models with summaries.
func selectRelevantModels(models []orient.ModelFreshness, maxCount int) []orient.ModelFreshness {
	var candidates []orient.ModelFreshness
	for _, m := range models {
		if m.Summary != "" && !m.IsStale() {
			candidates = append(candidates, m)
		}
	}

	// Sort by freshness (most recently updated first)
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].AgeDays < candidates[i].AgeDays {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	if len(candidates) > maxCount {
		candidates = candidates[:maxCount]
	}

	return candidates
}

// collectClaimEdges reads claims.yaml files from model directories and collects
// claim status summaries, recent disconfirmations, and notable edges.
// Returns pre-formatted text for orient output, or empty string if no edges found.
func collectClaimEdges(modelsDir string, now time.Time) string {
	files, err := claims.ScanAll(modelsDir)
	if err != nil || len(files) == 0 {
		return ""
	}

	// Claim status summaries (models with untested claims)
	statuses := claims.CollectClaimStatus(files, now)

	// Recently disconfirmed claims (contradicts verdict in last 7 days)
	disconfirmations := claims.CollectRecentDisconfirmations(files, now, 7)

	// Extract active keywords from recent spawn events (last 7 days), scoped to current project
	activeKeywords := extractRecentSpawnKeywords(now, readBeadsPrefix())

	// Notable edges (tensions, stale-in-active, unconfirmed core)
	edges := claims.CollectEdges(files, now, activeKeywords, 5)

	return claims.FormatClaimSurface(statuses, disconfirmations, edges)
}

// extractRecentSpawnKeywords extracts domain-relevant keywords from recent
// spawn events in events.jsonl (last 7 days), scoped to the given project prefix.
func extractRecentSpawnKeywords(now time.Time, projectPrefix string) []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	eventsPath := filepath.Join(home, ".orch", "events.jsonl")
	evts, err := parseOrientEvents(eventsPath)
	if err != nil {
		return nil
	}

	return spawnKeywordsFromEvents(evts, now, projectPrefix)
}

// spawnKeywordsFromEvents extracts domain-relevant keywords from spawn events
// within the last 7 days. Extracts skill names and significant task words.
// If projectPrefix is non-empty, only events matching that project are included.
func spawnKeywordsFromEvents(evts []orient.Event, now time.Time, projectPrefix string) []string {
	cutoff := now.Add(-7 * 24 * time.Hour).Unix()
	keywordSet := make(map[string]bool)

	for _, e := range evts {
		if e.Timestamp < cutoff {
			continue
		}
		if e.Type != "session.spawned" {
			continue
		}
		if e.Data == nil {
			continue
		}
		if projectPrefix != "" {
			beadsID, _ := e.Data["beads_id"].(string)
			if !strings.HasPrefix(beadsID, projectPrefix+"-") {
				continue
			}
		}
		// Extract skill name as keyword
		if skill, ok := e.Data["skill"].(string); ok && skill != "" {
			keywordSet[skill] = true
		}
		// Extract significant words from task description
		if task, ok := e.Data["task"].(string); ok {
			for _, word := range strings.Fields(strings.ToLower(task)) {
				// Strip surrounding punctuation (commas, periods, parens)
				word = strings.Trim(word, ".,;:!?()[]{}\"'")
				if len(word) > 3 && !isStopWord(word) {
					keywordSet[word] = true
				}
			}
		}
	}

	keywords := make([]string, 0, len(keywordSet))
	for kw := range keywordSet {
		keywords = append(keywords, kw)
	}
	return keywords
}

// spawnKeywordStopWords are common English words that add noise to keyword matching.
var spawnKeywordStopWords = map[string]bool{
	"that": true, "this": true, "with": true, "from": true, "when": true,
	"have": true, "been": true, "will": true, "should": true, "would": true,
	"could": true, "into": true, "also": true, "each": true, "then": true,
	"than": true, "them": true, "they": true, "their": true, "there": true,
	"were": true, "what": true, "which": true, "where": true, "does": true,
	"about": true, "after": true, "before": true, "between": true,
	"only": true, "other": true, "some": true, "such": true, "more": true,
	"most": true, "very": true, "just": true, "over": true,
}

func isStopWord(word string) bool {
	return spawnKeywordStopWords[word]
}

// collectPreviousSession finds and parses the most recent session debrief.
func collectPreviousSession(sessionsDir string) *orient.DebriefSummary {
	path, err := orient.FindLatestDebrief(sessionsDir)
	if err != nil {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	return orient.ParseDebriefSummary(string(content))
}

// collectFocus reads the current focus goal.
func collectFocus() string {
	store, err := focus.New("")
	if err != nil {
		return ""
	}
	f := store.Get()
	if f == nil {
		return ""
	}
	return f.Goal
}

// enrichIssuesWithKBContext queries `kb context` for each ready issue and attaches
// relevant decisions, constraints, and failed attempts.
func enrichIssuesWithKBContext(issues []orient.ReadyIssue) {
	for i := range issues {
		entries := queryKBContextForIssue(issues[i].Title)
		issues[i].KBContext = orient.SelectTopEntries(entries, 2)
	}
}

// queryKBContextForIssue calls `kb context "<title>" --format json` with a timeout
// and parses the result.
func queryKBContextForIssue(title string) []orient.KBEntry {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kb", "context", title, "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	return orient.ParseKBContext(output, 1)
}

// collectActiveThreads returns open threads updated within the last 7 days.
func collectActiveThreads(projectDir string) []orient.ActiveThread {
	threadsDir := filepath.Join(projectDir, ".kb", "threads")
	summaries, err := thread.ActiveThreads(threadsDir, 7)
	if err != nil || len(summaries) == 0 {
		return nil
	}

	// Limit to top 5 most recently updated
	limit := 5
	if len(summaries) < limit {
		limit = len(summaries)
	}

	result := make([]orient.ActiveThread, limit)
	for i := 0; i < limit; i++ {
		result[i] = orient.ActiveThread{
			Name:        summaries[i].Name,
			Title:       summaries[i].Title,
			Updated:     summaries[i].Updated,
			EntryCount:  summaries[i].EntryCount,
			LatestEntry: summaries[i].LatestEntry,
		}
	}
	return result
}

// collectPromotionReady returns converged threads without promoted_to.
func collectPromotionReady(projectDir string) []orient.PromotionCandidate {
	threadsDir := filepath.Join(projectDir, ".kb", "threads")
	candidates, err := thread.PromotionReady(threadsDir)
	if err != nil || len(candidates) == 0 {
		return nil
	}

	result := make([]orient.PromotionCandidate, len(candidates))
	for i, c := range candidates {
		result[i] = orient.PromotionCandidate{
			Slug:       c.Slug,
			Title:      c.Title,
			Updated:    c.Updated,
			EntryCount: c.EntryCount,
		}
	}
	return result
}

// collectInProgressCount runs `bd list --status=in_progress` and counts issue lines.
func collectInProgressCount() int {
	cmd := exec.Command("bd", "list", "--status=in_progress")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	return parseInProgressCount(string(output))
}

// collectHealthSummary reads the most recent health snapshot and generates alerts.
func collectHealthSummary() *orient.HealthSummary {
	store := getHealthStore()
	recent, err := store.ReadRecent(30)
	if err != nil || len(recent) == 0 {
		return nil
	}

	report := health.GenerateReport(recent)
	c := report.Current

	summary := &orient.HealthSummary{
		OpenIssues:    c.OpenIssues,
		BlockedIssues: c.BlockedIssues,
		StaleIssues:   c.StaleIssues,
		BloatedFiles:  c.BloatedFiles,
		FixFeatRatio:  c.FixFeatRatio,
	}

	for _, a := range report.Alerts {
		summary.Alerts = append(summary.Alerts, orient.HealthAlert{
			Message: a.Message,
			Level:   a.Level,
		})
	}

	return summary
}

// collectDaemonHealth reads daemon-status.json and computes 6 health signals.
func collectDaemonHealth(now time.Time) *orient.DaemonHealthView {
	status, err := daemon.ReadValidatedStatusFile()
	if err != nil || status == nil {
		return nil
	}

	summary := daemon.ComputeDaemonHealth(status, now)
	if summary == nil {
		return nil
	}

	view := &orient.DaemonHealthView{}
	for _, sig := range summary.Signals {
		view.Signals = append(view.Signals, orient.DaemonHealthSignalView{
			Name:   sig.Name,
			Level:  sig.Level,
			Detail: sig.Detail,
		})
	}
	return view
}

// collectChangelog runs `git log` since the last session date and returns changelog entries.
func collectChangelog(prevSession *orient.DebriefSummary) []orient.ChangelogEntry {
	var args []string
	if prevSession != nil && prevSession.Date != "" {
		args = []string{"log", "--format=%h|%s", "--since=" + prevSession.Date + "T00:00:00", "--no-merges"}
	} else {
		// Fallback: last 20 commits
		args = []string{"log", "--format=%h|%s", "--no-merges", "-20"}
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	return orient.ParseGitLog(string(output), 15)
}

// collectReflectSuggestions reads ~/.orch/reflect-suggestions.json.
func collectReflectSuggestions() *orient.ReflectSummary {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	path := filepath.Join(home, ".orch", "reflect-suggestions.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	return parseReflectSuggestions(data)
}

// parseReflectSuggestions parses reflect-suggestions.json into ReflectSummary.
func parseReflectSuggestions(data []byte) *orient.ReflectSummary {
	var raw struct {
		Timestamp string `json:"timestamp"`
		Synthesis []struct {
			Topic string `json:"topic"`
			Count int    `json:"count"`
		} `json:"synthesis"`
		Promote    []json.RawMessage `json:"promote"`
		Stale      []json.RawMessage `json:"stale"`
		Drift      []json.RawMessage `json:"drift"`
		Agreements []json.RawMessage `json:"agreements"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	synthCount := len(raw.Synthesis)
	promoteCount := len(raw.Promote)
	staleCount := len(raw.Stale)
	driftCount := len(raw.Drift)
	agreeCount := len(raw.Agreements)
	total := synthCount + promoteCount + staleCount + driftCount + agreeCount

	if total == 0 {
		return nil
	}

	summary := &orient.ReflectSummary{
		Total:      total,
		Synthesis:  synthCount,
		Stale:      staleCount,
		Promote:    promoteCount,
		Drift:      driftCount,
		Agreements: agreeCount,
	}

	// Top 3 synthesis clusters
	limit := 3
	if len(raw.Synthesis) < limit {
		limit = len(raw.Synthesis)
	}
	for i := 0; i < limit; i++ {
		summary.TopClusters = append(summary.TopClusters, orient.ReflectCluster{
			Topic: raw.Synthesis[i].Topic,
			Count: raw.Synthesis[i].Count,
		})
	}

	// Compute age from timestamp
	if raw.Timestamp != "" {
		summary.Age = computeReflectAge(raw.Timestamp)
	}

	return summary
}

// enrichReflectWithSessionOrphans computes session-scoped orphan counts
// and injects them into the ReflectSummary. Uses the previous session date
// as the cutoff — investigations created since then are counted.
func enrichReflectWithSessionOrphans(summary *orient.ReflectSummary, prevSession *orient.DebriefSummary, projectDir string) {
	if summary == nil {
		return
	}

	// Determine cutoff: previous session date, or 24h ago as fallback
	var since time.Time
	if prevSession != nil && prevSession.Date != "" {
		parsed, err := time.Parse("2006-01-02", prevSession.Date)
		if err == nil {
			since = parsed
		}
	}
	if since.IsZero() {
		since = time.Now().Add(-24 * time.Hour)
	}

	kbDir := filepath.Join(projectDir, ".kb")
	report, err := kbmetrics.ComputeSessionOrphans(kbDir, since)
	if err != nil {
		return
	}

	summary.SessionOrphans = report.Orphaned
	summary.SessionInvestigations = report.Investigations
}

// computeReflectAge computes a human-readable age from an ISO timestamp.
func computeReflectAge(timestamp string) string {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		// Try alternate format
		t, err = time.Parse("2006-01-02T15:04:05.999999Z", timestamp)
		if err != nil {
			return ""
		}
	}

	age := time.Since(t)
	hours := int(age.Hours())
	if hours < 1 {
		return "just now"
	}
	if hours < 24 {
		return fmt.Sprintf("%dh ago", hours)
	}
	days := hours / 24
	return fmt.Sprintf("%dd ago", days)
}

// collectConfigDrift checks expected symlinks in ~/.claude-personal/.
func collectConfigDrift() []orient.ConfigDriftItem {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	personalDir := filepath.Join(home, ".claude-personal")
	primaryDir := filepath.Join(home, ".claude")

	// Skip if personal config dir doesn't exist
	if _, err := os.Stat(personalDir); err != nil {
		return nil
	}

	expectedSymlinks := []string{
		"settings.json",
		"CLAUDE.md",
		"skills",
		"hooks",
		"statusline.sh",
	}

	var drifted []orient.ConfigDriftItem
	for _, file := range expectedSymlinks {
		target := filepath.Join(primaryDir, file)
		link := filepath.Join(personalDir, file)

		// Skip if source doesn't exist in primary
		if _, err := os.Stat(target); err != nil {
			continue
		}

		info, err := os.Lstat(link)
		if err != nil {
			continue
		}

		if info.Mode()&os.ModeSymlink == 0 {
			drifted = append(drifted, orient.ConfigDriftItem{
				File:   file,
				Reason: "not a symlink",
			})
		} else {
			linkTarget, err := os.Readlink(link)
			if err != nil {
				continue
			}
			if linkTarget != filepath.Join(primaryDir, file) && linkTarget != target {
				drifted = append(drifted, orient.ConfigDriftItem{
					File:   file,
					Reason: fmt.Sprintf("points to %s", linkTarget),
				})
			}
		}
	}

	return drifted
}

// collectSessionResume runs `orch session resume --for-injection` and captures output.
func collectSessionResume() *orient.SessionResume {
	// Skip for spawned agents (they have SPAWN_CONTEXT.md)
	if os.Getenv("ORCH_SPAWNED") == "1" {
		return nil
	}
	// Skip for non-interactive contexts
	ctx := os.Getenv("CLAUDE_CONTEXT")
	switch ctx {
	case "bare", "worker", "orchestrator", "meta-orchestrator":
		return nil
	}

	// Check if handoff exists
	checkCmd := exec.Command("orch", "session", "resume", "--check")
	if err := checkCmd.Run(); err != nil {
		return nil
	}

	// Get the handoff content
	contentCmd := exec.Command("orch", "session", "resume", "--for-injection")
	output, err := contentCmd.Output()
	if err != nil {
		return nil
	}

	content := strings.TrimSpace(string(output))
	if content == "" {
		return nil
	}

	return &orient.SessionResume{
		Content: content,
	}
}

// loadOrchSkillContent reads the orchestrator skill from ~/.claude/skills/meta/orchestrator/SKILL.md.
func loadOrchSkillContent() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(home, ".claude", "skills", "meta", "orchestrator", "SKILL.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(content)
}

// queryPlanBeadsStatuses collects all beads IDs from hydrated plans and queries their statuses.
func queryPlanBeadsStatuses(plans []orient.PlanSummary) map[string]string {
	ids := orient.CollectPlanBeadsIDs(plans)
	if len(ids) == 0 {
		return nil
	}

	client := beads.NewCLIClient()
	statusMap := make(map[string]string)
	for _, id := range ids {
		issue, err := client.Show(id)
		if err != nil {
			statusMap[id] = "unknown"
			continue
		}
		statusMap[id] = issue.Status
	}
	return statusMap
}

// enrichThroughputWithGitGroundTruth populates net code impact
// by querying git numstat for all commits in the throughput window.
func enrichThroughputWithGitGroundTruth(tp *orient.Throughput) {
	sinceArg := fmt.Sprintf("--since=%dd", tp.Days)

	numstatCmd := exec.Command("git", "log", "--format=", "--numstat", sinceArg, "--no-merges")
	numstatOutput, err := numstatCmd.Output()
	if err != nil {
		return
	}

	tp.NetLinesAdded, tp.NetLinesRemoved = orient.ParseGitNumstat(string(numstatOutput))
}

// parseInProgressCount counts issue lines from `bd list --status=in_progress` output.
// Lines start with issue IDs (e.g., "orch-go-abc1 [P2] [feature] in_progress ...").
func parseInProgressCount(output string) int {
	count := 0
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, " in_progress ") {
			count++
		}
	}
	return count
}

// computeDivergenceAlerts compares activity metrics against impact metrics
// and returns alerts for sustained gaps. Uses data already collected by orient:
// throughput (completion rate, merge rate) and reflect summary (session orphans, stale decisions).
func computeDivergenceAlerts(data *orient.OrientationData) []orient.DivergenceAlert {
	tp := data.Throughput

	input := orient.DivergenceInput{
		Days: tp.Days,
	}

	// Completion rate from throughput
	if tp.Spawns > 0 {
		input.CompletionRate = float64(tp.Completions) / float64(tp.Spawns)
	}

	// Session orphans and stale decisions from reflect summary
	if data.ReflectSummary != nil {
		input.SessionOrphans = data.ReflectSummary.SessionOrphans
		input.SessionInvestigations = data.ReflectSummary.SessionInvestigations
		input.StaleDecisions = data.ReflectSummary.Stale
		// Total decisions = stale + non-stale; approximate from reflect data
		// Stale count comes from reflect; total is not directly available,
		// so we use a heuristic: if stale > 0, estimate total from the ratio
		// For now, query decision count from filesystem
		input.TotalDecisions = countDecisions()
	}

	// Per-skill rework rate from learning store (Phase 2)
	// Aggregate across all skills for the overall rework signal
	home, _ := os.UserHomeDir()
	eventsPath := filepath.Join(home, ".orch", "events.jsonl")
	if store, err := events.ComputeLearning(eventsPath); err == nil {
		var totalCompletions, totalRework int
		var totalCompletionRate float64
		var skillCount int
		for _, sl := range store.Skills {
			totalCompletions += sl.TotalCompletions
			totalRework += sl.ReworkCount
			if sl.TotalCompletions > 0 {
				totalCompletionRate += sl.SuccessRate
				skillCount++
			}
		}
		if totalCompletions > 0 {
			input.ReworkRate = float64(totalRework) / float64(totalCompletions)
		}
		if skillCount > 0 {
			input.SelfReportedCompletion = totalCompletionRate / float64(skillCount)
		}
	}

	return orient.ComputeDivergence(input)
}

// collectAdoptionDrift runs the adoption measurement and returns items
// where signals have drifted below target. Only surfaces non-ok signals
// to keep orient output concise.
func collectAdoptionDrift(projectDir string) []orient.AdoptionDriftItem {
	result := measureAdoption(projectDir)
	var items []orient.AdoptionDriftItem
	for _, sig := range result.Signals {
		if sig.Status != "ok" {
			items = append(items, orient.AdoptionDriftItem{
				Signal:    sig.Name,
				RatePct:   sig.RatePct,
				TargetPct: sig.TargetPct,
				Level:     sig.Status,
			})
		}
	}
	return items
}

// loadBriefReadStateForOrient loads the persistent brief read state and returns
// a map of beadsID → true for briefs that have been read in the given project.
func loadBriefReadStateForOrient(projectDir string) map[string]bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	statePath := filepath.Join(home, ".orch", "briefs-read-state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil
	}
	var fullState map[string]bool
	if err := json.Unmarshal(data, &fullState); err != nil {
		return nil
	}
	// Filter to current project — keys are "projectDir:beadsID"
	result := make(map[string]bool)
	prefix := projectDir + ":"
	for key, read := range fullState {
		if read && strings.HasPrefix(key, prefix) {
			beadsID := strings.TrimPrefix(key, prefix)
			result[beadsID] = true
		}
	}
	return result
}

type orientComposeResult struct {
	digest *compose.Digest
	err    error
}

func collectComposeSummary(briefsDir, threadsDir, digestsDir string, readState map[string]bool) *orient.ComposeSummary {
	unprocessedBriefs := countUnprocessedBriefs(briefsDir, digestsDir, readState)
	if unprocessedBriefs < 5 {
		return nil
	}

	resultCh := make(chan orientComposeResult, 1)
	go func() {
		digest, err := orientComposeFunc(briefsDir, threadsDir)
		resultCh <- orientComposeResult{digest: digest, err: err}
	}()

	timer := time.NewTimer(orientComposeTimeout)
	defer timer.Stop()

	select {
	case result := <-resultCh:
		if result.err != nil || result.digest == nil {
			return nil
		}
		path, err := orientWriteDigestFunc(result.digest, digestsDir)
		if err != nil {
			return &orient.ComposeSummary{
				UnprocessedBriefs: unprocessedBriefs,
				BriefsComposed:    result.digest.BriefsComposed,
				ClustersFound:     result.digest.ClustersFound,
				Clusters:          summarizeComposeClusters(result.digest, 3),
				Note:              fmt.Sprintf("digest write skipped: %v", err),
			}
		}
		return &orient.ComposeSummary{
			UnprocessedBriefs: unprocessedBriefs,
			BriefsComposed:    result.digest.BriefsComposed,
			ClustersFound:     result.digest.ClustersFound,
			DigestPath:        path,
			Clusters:          summarizeComposeClusters(result.digest, 3),
		}
	case <-timer.C:
		return &orient.ComposeSummary{
			UnprocessedBriefs: unprocessedBriefs,
			Note:              "skipped: compose exceeded 2s budget",
		}
	}
}

func countUnprocessedBriefs(briefsDir, digestsDir string, readState map[string]bool) int {
	briefs, err := compose.LoadBriefs(briefsDir)
	if err != nil {
		return 0
	}

	digested := orient.DigestedBriefIDs(digestsDir)
	count := 0
	for _, brief := range briefs {
		if !readState[brief.ID] || !digested[brief.ID] {
			count++
		}
	}
	return count
}

func summarizeComposeClusters(digest *compose.Digest, limit int) []orient.ComposeClusterSummary {
	if digest == nil || len(digest.Clusters) == 0 {
		return nil
	}
	if len(digest.Clusters) < limit {
		limit = len(digest.Clusters)
	}
	clusters := make([]orient.ComposeClusterSummary, 0, limit)
	for _, cluster := range digest.Clusters[:limit] {
		clusters = append(clusters, orient.ComposeClusterSummary{
			Name:       cluster.Name,
			BriefCount: len(cluster.Briefs),
		})
	}
	return clusters
}

func orientThreadsDir(projectDir string) string {
	return filepath.Join(projectDir, ".kb", "threads")
}

func orientDigestsDir(projectDir string) string {
	return filepath.Join(projectDir, ".kb", "digests")
}

// countDecisions counts .md files in .kb/decisions/ for stale rate computation.
func countDecisions() int {
	dir, err := os.Getwd()
	if err != nil {
		return 0
	}
	decisionsDir := filepath.Join(dir, ".kb", "decisions")
	entries, err := os.ReadDir(decisionsDir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".md" {
			count++
		}
	}
	return count
}
