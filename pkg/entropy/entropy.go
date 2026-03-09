// Package entropy analyzes codebase growth trends, duplication, and structural
// health to detect entropy spiral conditions. This is Harness Layer 3 —
// aggregating signals from git history, events.jsonl, bloat analysis, and
// duplication detection into actionable recommendations.
//
// Key thresholds (from .kb/models/entropy-spiral/model.md):
//   - Fix:Feat ratio healthy < 0.5, degrading 0.5-0.9, spiral > 0.9
//   - Velocity > 45 commits/day = unverifiable
//   - Override/bypass trending up = weakening gates
package entropy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

// Report is the top-level entropy analysis result.
type Report struct {
	GeneratedAt          time.Time            `json:"generated_at"`
	WindowDays           int                  `json:"window_days"`
	ProjectDir           string               `json:"project_dir"`
	CommitClassification CommitClassification `json:"commit_classification"`
	Velocity             Velocity             `json:"velocity"`
	HealthLevel          string               `json:"health_level"` // "healthy", "degrading", "spiral"
	BloatedFileCount     int                  `json:"bloated_file_count"`
	BloatedFiles         []BloatedFile        `json:"bloated_files,omitempty"`
	DuplicatePairCount   int                  `json:"duplicate_pair_count"`
	EventStats           EventStats           `json:"event_stats"`
	OverrideTrend        OverrideTrend        `json:"override_trend"`
	Recommendations      []Recommendation     `json:"recommendations"`
}

// CommitClassification breaks down commits by conventional prefix.
type CommitClassification struct {
	Fixes    int `json:"fixes"`
	Features int `json:"features"`
	Other    int `json:"other"`
	Total    int `json:"total"`
}

// FixFeatRatio returns the fix-to-feature ratio. Returns 0 if no features.
func (cc *CommitClassification) FixFeatRatio() float64 {
	if cc.Features == 0 {
		return 0
	}
	return float64(cc.Fixes) / float64(cc.Features)
}

// Velocity tracks commit rate over a time window.
type Velocity struct {
	CommitsPerDay float64 `json:"commits_per_day"`
	TotalCommits  int     `json:"total_commits"`
	WindowDays    int     `json:"window_days"`
}

// BloatedFile represents a file exceeding the line threshold.
type BloatedFile struct {
	Path  string `json:"path"`
	Lines int    `json:"lines"`
}

// EventStats aggregates events.jsonl counts within the analysis window.
type EventStats struct {
	Spawns       int `json:"spawns"`
	Completions  int `json:"completions"`
	Abandonments int `json:"abandonments"`
	Bypasses     int `json:"bypasses"`
	Reworks      int `json:"reworks"`
}

// OverrideTrend tracks verification bypass direction.
type OverrideTrend struct {
	WindowDays    int    `json:"window_days"`
	CurrentCount  int    `json:"current_count"`
	PreviousCount int    `json:"previous_count"`
	Delta         int    `json:"delta"`
	Direction     string `json:"direction"` // "up", "down", "flat"
}

// Recommendation is a specific action item based on the analysis.
type Recommendation struct {
	Severity string `json:"severity"` // "critical", "warning", "info"
	Signal   string `json:"signal"`   // which metric triggered this
	Message  string `json:"message"`
}

// commitEntry is an internal representation of a git commit.
type commitEntry struct {
	message   string
	timestamp time.Time
}

// Analyze runs the full entropy analysis for the given project directory.
func Analyze(projectDir string, windowDays int, eventsPath string) (*Report, error) {
	if windowDays <= 0 {
		windowDays = 28
	}

	commits, err := gitLogCommits(projectDir, windowDays)
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	cc := classifyCommits(commits)
	vel := calculateVelocity(commits, windowDays)
	bloatCount, bloatedFiles := countBloatedFiles(projectDir, 800)
	es := aggregateEvents(eventsPath, windowDays)
	ot := calculateOverrideTrend(eventsPath, 7)

	ratio := cc.FixFeatRatio()

	report := &Report{
		GeneratedAt:          time.Now(),
		WindowDays:           windowDays,
		ProjectDir:           projectDir,
		CommitClassification: cc,
		Velocity:             vel,
		HealthLevel:          healthLevel(ratio),
		BloatedFileCount:     bloatCount,
		BloatedFiles:         bloatedFiles,
		EventStats:           es,
		OverrideTrend:        ot,
	}

	report.Recommendations = generateRecommendations(report)
	return report, nil
}

// classifyCommits categorizes commits by conventional commit prefix.
func classifyCommits(commits []commitEntry) CommitClassification {
	var cc CommitClassification
	cc.Total = len(commits)

	for _, c := range commits {
		msg := strings.ToLower(c.message)
		switch {
		case strings.HasPrefix(msg, "fix:") || strings.HasPrefix(msg, "fix("):
			cc.Fixes++
		case strings.HasPrefix(msg, "feat:") || strings.HasPrefix(msg, "feat("):
			cc.Features++
		default:
			cc.Other++
		}
	}
	return cc
}

// calculateVelocity computes commits per day over the window.
func calculateVelocity(commits []commitEntry, windowDays int) Velocity {
	if windowDays <= 0 {
		windowDays = 1
	}
	return Velocity{
		CommitsPerDay: float64(len(commits)) / float64(windowDays),
		TotalCommits:  len(commits),
		WindowDays:    windowDays,
	}
}

// healthLevel maps fix:feat ratio to a health label.
// Thresholds from .kb/models/entropy-spiral/model.md:
//   - healthy: < 0.5
//   - degrading: 0.5 - 0.9
//   - spiral: >= 0.9
func healthLevel(fixFeatRatio float64) string {
	switch {
	case fixFeatRatio >= 0.9:
		return "spiral"
	case fixFeatRatio >= 0.5:
		return "degrading"
	default:
		return "healthy"
	}
}

// countBloatedFiles walks the project and counts files exceeding the threshold.
func countBloatedFiles(dir string, threshold int) (int, []BloatedFile) {
	var bloated []BloatedFile

	skipDirs := map[string]bool{
		".git": true, "node_modules": true, "vendor": true,
		".svelte-kit": true, "dist": true, "build": true,
		"__pycache__": true, ".next": true, ".nuxt": true,
		".output": true, ".opencode": true, ".orch": true,
		".beads": true, ".claude": true, ".playwright-mcp": true,
	}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".ts" && ext != ".svelte" && ext != ".js" {
			return nil
		}

		lines := countLines(path)
		if lines > threshold {
			rel, _ := filepath.Rel(dir, path)
			if rel == "" {
				rel = path
			}
			bloated = append(bloated, BloatedFile{Path: rel, Lines: lines})
		}
		return nil
	})

	sort.Slice(bloated, func(i, j int) bool {
		return bloated[i].Lines > bloated[j].Lines
	})
	return len(bloated), bloated
}

func countLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}
	return count
}

// aggregateEvents counts key event types from events.jsonl within the window.
func aggregateEvents(logPath string, windowDays int) EventStats {
	var stats EventStats
	if logPath == "" {
		logPath = events.DefaultLogPath()
	}

	f, err := os.Open(logPath)
	if err != nil {
		return stats
	}
	defer f.Close()

	cutoff := time.Now().AddDate(0, 0, -windowDays).Unix()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event events.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		if event.Timestamp < cutoff {
			continue
		}

		switch event.Type {
		case events.EventTypeSessionSpawned:
			stats.Spawns++
		case events.EventTypeAgentCompleted:
			stats.Completions++
		case events.EventTypeAgentAbandonedTelemetry:
			stats.Abandonments++
		case events.EventTypeVerificationBypassed:
			stats.Bypasses++
		case events.EventTypeAgentReworked:
			stats.Reworks++
		}
	}
	return stats
}

// calculateOverrideTrend compares bypass counts between current and previous windows.
func calculateOverrideTrend(logPath string, windowDays int) OverrideTrend {
	if windowDays <= 0 {
		windowDays = 7
	}
	if logPath == "" {
		logPath = events.DefaultLogPath()
	}

	f, err := os.Open(logPath)
	if err != nil {
		return OverrideTrend{WindowDays: windowDays, Direction: "flat"}
	}
	defer f.Close()

	now := time.Now()
	windowStart := now.AddDate(0, 0, -windowDays).Unix()
	previousStart := now.AddDate(0, 0, -2*windowDays).Unix()

	current, previous := 0, 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var event events.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event.Type != events.EventTypeVerificationBypassed {
			continue
		}
		if event.Timestamp >= windowStart {
			current++
		} else if event.Timestamp >= previousStart {
			previous++
		}
	}

	delta := current - previous
	direction := "flat"
	if delta > 0 {
		direction = "up"
	} else if delta < 0 {
		direction = "down"
	}

	return OverrideTrend{
		WindowDays:    windowDays,
		CurrentCount:  current,
		PreviousCount: previous,
		Delta:         delta,
		Direction:     direction,
	}
}

// generateRecommendations produces actionable items based on analysis.
func generateRecommendations(report *Report) []Recommendation {
	var recs []Recommendation

	ratio := report.CommitClassification.FixFeatRatio()

	// Fix:Feat ratio
	switch {
	case ratio >= 0.9:
		recs = append(recs, Recommendation{
			Severity: "critical",
			Signal:   "fix_feat_ratio",
			Message:  fmt.Sprintf("Fix:Feat ratio %.2f:1 indicates entropy spiral. Agents are primarily fixing other agents' work. Pause autonomous spawns and audit recent changes.", ratio),
		})
	case ratio >= 0.5:
		recs = append(recs, Recommendation{
			Severity: "warning",
			Signal:   "fix_feat_ratio",
			Message:  fmt.Sprintf("Fix:Feat ratio %.2f:1 is degrading. Consider reducing concurrent agents or increasing verification gates.", ratio),
		})
	}

	// Velocity
	if report.Velocity.CommitsPerDay > 45 {
		recs = append(recs, Recommendation{
			Severity: "critical",
			Signal:   "velocity",
			Message:  fmt.Sprintf("Velocity %.1f commits/day exceeds verification bandwidth (>45/day). Human review cannot keep pace.", report.Velocity.CommitsPerDay),
		})
	} else if report.Velocity.CommitsPerDay > 30 {
		recs = append(recs, Recommendation{
			Severity: "warning",
			Signal:   "velocity",
			Message:  fmt.Sprintf("Velocity %.1f commits/day is approaching verification bandwidth limit.", report.Velocity.CommitsPerDay),
		})
	}

	// Bloated files
	if report.BloatedFileCount > 0 {
		severity := "warning"
		if report.BloatedFileCount > 5 {
			severity = "critical"
		}
		recs = append(recs, Recommendation{
			Severity: severity,
			Signal:   "bloat",
			Message:  fmt.Sprintf("%d files exceed 800-line bloat threshold. Run 'orch hotspot' for extraction targets.", report.BloatedFileCount),
		})
	}

	// Override trend
	if report.OverrideTrend.Direction == "up" && report.OverrideTrend.Delta > 2 {
		recs = append(recs, Recommendation{
			Severity: "warning",
			Signal:   "override_trend",
			Message:  fmt.Sprintf("Verification bypasses trending up (+%d in last %d days). Gates may be losing effectiveness.", report.OverrideTrend.Delta, report.OverrideTrend.WindowDays),
		})
	}

	// Abandonment rate
	if report.EventStats.Spawns > 0 {
		abandonRate := float64(report.EventStats.Abandonments) / float64(report.EventStats.Spawns)
		if abandonRate > 0.5 {
			recs = append(recs, Recommendation{
				Severity: "critical",
				Signal:   "abandonment_rate",
				Message:  fmt.Sprintf("%.0f%% abandonment rate (%d/%d spawns). Agents are failing more than succeeding.", abandonRate*100, report.EventStats.Abandonments, report.EventStats.Spawns),
			})
		} else if abandonRate > 0.3 {
			recs = append(recs, Recommendation{
				Severity: "warning",
				Signal:   "abandonment_rate",
				Message:  fmt.Sprintf("%.0f%% abandonment rate. Investigate common failure patterns.", abandonRate*100),
			})
		}
	}

	// Rework rate
	if report.EventStats.Completions > 0 {
		reworkRate := float64(report.EventStats.Reworks) / float64(report.EventStats.Completions)
		if reworkRate > 0.3 {
			recs = append(recs, Recommendation{
				Severity: "warning",
				Signal:   "rework_rate",
				Message:  fmt.Sprintf("%.0f%% rework rate. Quality issues may be slipping through verification.", reworkRate*100),
			})
		}
	}

	// Duplicate detection hint
	if report.DuplicatePairCount > 5 {
		recs = append(recs, Recommendation{
			Severity: "warning",
			Signal:   "duplication",
			Message:  fmt.Sprintf("%d duplicate function pairs detected. Run 'orch dupdetect' for extraction candidates.", report.DuplicatePairCount),
		})
	}

	return recs
}

// gitLogCommits retrieves commits from the last N days.
func gitLogCommits(dir string, days int) ([]commitEntry, error) {
	since := fmt.Sprintf("--since=%d days ago", days)
	cmd := exec.Command("git", "log", since, "--pretty=format:%s\t%aI")
	cmd.Dir = dir

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var commits []commitEntry
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) < 2 {
			continue
		}
		t, err := time.Parse(time.RFC3339, parts[1])
		if err != nil {
			t = time.Now()
		}
		commits = append(commits, commitEntry{message: parts[0], timestamp: t})
	}
	return commits, nil
}

// FormatText produces a human-readable summary of the entropy report.
func FormatText(report *Report) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Entropy Report — %s (%d-day window)\n", report.ProjectDir, report.WindowDays))
	b.WriteString(strings.Repeat("=", 60) + "\n\n")

	// Health banner
	switch report.HealthLevel {
	case "spiral":
		b.WriteString("!! ENTROPY SPIRAL DETECTED !!\n\n")
	case "degrading":
		b.WriteString("~~ Degrading Health ~~\n\n")
	default:
		b.WriteString("Healthy\n\n")
	}

	// Commit classification
	cc := report.CommitClassification
	b.WriteString(fmt.Sprintf("Commits: %d total (fix: %d, feat: %d, other: %d)\n", cc.Total, cc.Fixes, cc.Features, cc.Other))
	b.WriteString(fmt.Sprintf("Fix:Feat Ratio: %.2f:1\n", cc.FixFeatRatio()))
	b.WriteString(fmt.Sprintf("Velocity: %.1f commits/day\n", report.Velocity.CommitsPerDay))
	b.WriteString("\n")

	// Bloat
	b.WriteString(fmt.Sprintf("Bloated Files (>800 lines): %d\n", report.BloatedFileCount))
	for _, f := range report.BloatedFiles {
		b.WriteString(fmt.Sprintf("  %s (%d lines)\n", f.Path, f.Lines))
	}
	b.WriteString("\n")

	// Events
	es := report.EventStats
	b.WriteString(fmt.Sprintf("Agent Activity: %d spawns, %d completions, %d abandonments, %d bypasses, %d reworks\n",
		es.Spawns, es.Completions, es.Abandonments, es.Bypasses, es.Reworks))

	// Override trend
	ot := report.OverrideTrend
	b.WriteString(fmt.Sprintf("Override Trend (%dd): %d current, %d previous (%s)\n",
		ot.WindowDays, ot.CurrentCount, ot.PreviousCount, ot.Direction))
	b.WriteString("\n")

	// Duplicates
	if report.DuplicatePairCount > 0 {
		b.WriteString(fmt.Sprintf("Duplicate Function Pairs: %d\n\n", report.DuplicatePairCount))
	}

	// Recommendations
	if len(report.Recommendations) > 0 {
		b.WriteString("Recommendations\n")
		b.WriteString(strings.Repeat("-", 40) + "\n")
		for _, r := range report.Recommendations {
			prefix := "  "
			switch r.Severity {
			case "critical":
				prefix = "!! "
			case "warning":
				prefix = "~  "
			}
			b.WriteString(fmt.Sprintf("%s[%s] %s\n", prefix, r.Signal, r.Message))
		}
	} else {
		b.WriteString("No recommendations — system looks healthy.\n")
	}

	return b.String()
}

// CountDuplicatePairs runs dupdetect and returns the count of duplicate pairs.
// This is separated from Analyze to allow callers to skip the expensive AST scan.
func CountDuplicatePairs(projectDir string) int {
	// Import dupdetect at runtime to avoid circular dependency concerns.
	// We shell out to keep the entropy package lightweight.
	cmd := exec.Command("go", "run", ".", "dupdetect", "--json", "--dry-run")
	cmd.Dir = projectDir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return 0
	}

	// Parse the JSON output to count pairs
	var result struct {
		Pairs []interface{} `json:"pairs"`
		Count int           `json:"count"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		// Try counting lines as fallback
		count := 0
		for _, b := range out.Bytes() {
			if b == '\n' {
				count++
			}
		}
		return count
	}
	if result.Count > 0 {
		return result.Count
	}
	return len(result.Pairs)
}

// RunArchLintTests runs the architecture lint test suite and returns pass/fail count.
func RunArchLintTests(projectDir string) (passed, failed int, err error) {
	cmd := exec.Command("go", "test", "-run", "TestArchitectureLint", "-v", "-count=1", "./cmd/orch/")
	cmd.Dir = projectDir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	runErr := cmd.Run()

	// Parse test output
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "--- PASS:") {
			passed++
		} else if strings.HasPrefix(line, "--- FAIL:") {
			failed++
		}
	}

	if runErr != nil && failed == 0 {
		return passed, 0, runErr
	}
	return passed, failed, nil
}

// parseCount is a helper to parse integers from strings.
func parseCount(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}
