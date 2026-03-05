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

	"github.com/dylan-conlin/orch-go/pkg/focus"
	"github.com/dylan-conlin/orch-go/pkg/orient"
	"github.com/spf13/cobra"
)

var (
	orientDays      int
	orientJSON      bool
	orientSkipReady bool
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
}

func runOrient() error {
	now := time.Now()
	projectDir, _ := os.Getwd()

	data := &orient.OrientationData{}

	// 1. Throughput from events.jsonl
	data.Throughput = collectThroughput(now)

	// 2. Previous session from latest debrief
	sessionsDir := filepath.Join(projectDir, ".kb", "sessions")
	data.PreviousSession = collectPreviousSession(sessionsDir)

	// 3. Ready issues from bd ready (skippable when frontier provides them)
	if !orientSkipReady {
		data.ReadyIssues = collectReadyIssues()

		// 3b. Decision context per ready issue from kb context
		enrichIssuesWithKBContext(data.ReadyIssues)
	}

	// 4. Active plans from .kb/plans/
	plansDir := filepath.Join(projectDir, ".kb", "plans")
	activePlans, err := orient.ScanActivePlans(plansDir)
	if err == nil && len(activePlans) > 0 {
		data.ActivePlans = activePlans
	}

	// 5. Model freshness from .kb/models/
	modelsDir := filepath.Join(projectDir, ".kb", "models")
	allModels, err := orient.ScanModelFreshness(modelsDir)
	if err == nil {
		// Relevant models: top 3 freshest non-stale models
		data.RelevantModels = selectRelevantModels(allModels, 3)

		// Stale models: up to 2
		data.StaleModels = orient.FilterStaleModels(allModels, 2)
	}

	// 6. Focus
	data.FocusGoal = collectFocus()

	// 7. In-progress count from bd
	data.Throughput.InProgress = collectInProgressCount()

	if orientJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}

	fmt.Print(orient.FormatOrientation(data))
	return nil
}

// collectThroughput reads events.jsonl and computes throughput metrics.
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

	return orient.ComputeThroughput(events, now, orientDays)
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

// collectInProgressCount runs `bd list --status=in_progress` and counts issue lines.
func collectInProgressCount() int {
	cmd := exec.Command("bd", "list", "--status=in_progress")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	return parseInProgressCount(string(output))
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
