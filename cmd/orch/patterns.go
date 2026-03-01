// Package main provides the patterns command for surfacing behavioral patterns to orchestrators.
package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/action"
	"github.com/dylan-conlin/orch-go/pkg/patterns"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	patternsJSON    bool
	patternsVerbose bool

	// suppress subcommand flags
	suppressReason   string
	suppressDuration string
)

var patternsCmd = &cobra.Command{
	Use:   "patterns",
	Short: "Surface behavioral patterns for orchestrator awareness",
	Long: `Show detected behavioral patterns that orchestrators should be aware of.

Surfaces patterns like:
- Retry patterns (issues with multiple spawn/abandon cycles)
- Empty context queries (kb context returned no results 3+ times)
- Recurring gaps (same knowledge gap detected repeatedly)
- Persistent failures (multiple attempts with no success)

These patterns help orchestrators avoid blind respawning and identify
systemic issues that need addressing.

Examples:
  orch patterns                    # Show all detected patterns
  orch patterns --json             # Output as JSON for scripting
  orch patterns --verbose          # Show detailed pattern information`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPatterns()
	},
}

var patternsSuppressCmd = &cobra.Command{
	Use:   "suppress <index>",
	Short: "Suppress a pattern by its index",
	Long: `Suppress a behavioral pattern so it no longer appears in pattern reports.

The index corresponds to the position in the 'orch patterns' output (0-based).
Suppressed patterns can optionally expire after a duration.

Examples:
  orch patterns suppress 0                    # Suppress first pattern permanently
  orch patterns suppress 2 --reason "known"   # Suppress with reason
  orch patterns suppress 1 --duration 7d      # Suppress for 7 days`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPatternsSuppress(args[0])
	},
}

func init() {
	patternsCmd.Flags().BoolVar(&patternsJSON, "json", false, "Output as JSON for scripting")
	patternsCmd.Flags().BoolVarP(&patternsVerbose, "verbose", "v", false, "Show detailed pattern information")

	// Add suppress subcommand
	patternsSuppressCmd.Flags().StringVar(&suppressReason, "reason", "", "Reason for suppressing this pattern")
	patternsSuppressCmd.Flags().StringVar(&suppressDuration, "duration", "", "Duration to suppress (e.g., 1h, 7d, 30d). Empty means permanent")
	patternsCmd.AddCommand(patternsSuppressCmd)

	rootCmd.AddCommand(patternsCmd)
}

// PatternType categorizes the type of behavioral pattern.
type PatternType string

const (
	// PatternTypeRetry indicates an issue with retry history (spawn/abandon cycles).
	PatternTypeRetry PatternType = "retry"

	// PatternTypePersistentFailure indicates multiple attempts with no success.
	PatternTypePersistentFailure PatternType = "persistent_failure"

	// PatternTypeEmptyContext indicates kb context queries returning no results.
	PatternTypeEmptyContext PatternType = "empty_context"

	// PatternTypeRecurringGap indicates the same knowledge gap detected repeatedly.
	PatternTypeRecurringGap PatternType = "recurring_gap"

	// PatternTypeContextDrift indicates context quality degrading over time.
	PatternTypeContextDrift PatternType = "context_drift"

	// PatternTypeFutileAction indicates repeated tool actions with unsuccessful outcomes.
	PatternTypeFutileAction PatternType = "futile_action"
)

// PatternSeverity indicates how significant the pattern is.
type PatternSeverity string

const (
	PatternSeverityCritical PatternSeverity = "critical"
	PatternSeverityWarning  PatternSeverity = "warning"
	PatternSeverityInfo     PatternSeverity = "info"
)

// DetectedPattern represents a single behavioral pattern detected.
type DetectedPattern struct {
	Type        PatternType     `json:"type"`
	Severity    PatternSeverity `json:"severity"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Count       int             `json:"count"`           // Number of occurrences
	Query       string          `json:"query,omitempty"` // For gap patterns
	BeadsID     string          `json:"beads_id,omitempty"`
	Suggestion  string          `json:"suggestion"`
	Details     []string        `json:"details,omitempty"` // Additional context
	FirstSeen   time.Time       `json:"first_seen,omitempty"`
	LastSeen    time.Time       `json:"last_seen,omitempty"`
}

// PatternsOutput represents the complete patterns analysis output.
type PatternsOutput struct {
	TotalPatterns int               `json:"total_patterns"`
	Critical      int               `json:"critical_count"`
	Warning       int               `json:"warning_count"`
	Info          int               `json:"info_count"`
	Patterns      []DetectedPattern `json:"patterns"`
	GeneratedAt   time.Time         `json:"generated_at"`
}

func runPatterns() error {
	output := PatternsOutput{
		Patterns:    []DetectedPattern{},
		GeneratedAt: time.Now(),
	}

	// 1. Collect retry patterns from events.jsonl
	if retryPatterns, err := collectRetryPatterns(); err == nil {
		output.Patterns = append(output.Patterns, retryPatterns...)
	}

	// 2. Collect recurring gap patterns from gap-tracker.json
	if gapPatterns, err := collectGapPatterns(); err == nil {
		output.Patterns = append(output.Patterns, gapPatterns...)
	}

	// 3. Collect action outcome patterns from action-log.jsonl
	if actionPatterns, err := collectActionPatterns(); err == nil {
		output.Patterns = append(output.Patterns, actionPatterns...)
	}

	// Sort patterns by severity (critical first), then by count
	sortPatterns(output.Patterns)

	// Count by severity
	for _, p := range output.Patterns {
		switch p.Severity {
		case PatternSeverityCritical:
			output.Critical++
		case PatternSeverityWarning:
			output.Warning++
		case PatternSeverityInfo:
			output.Info++
		}
	}
	output.TotalPatterns = len(output.Patterns)

	// Output
	if patternsJSON {
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Human-readable output
	printPatternsOutput(output)
	return nil
}

// collectRetryPatterns collects retry patterns from events.jsonl via verify package.
// Filters out closed issues to avoid flagging resolved work as failures.
func collectRetryPatterns() ([]DetectedPattern, error) {
	patterns := []DetectedPattern{}

	retryStats, err := verify.GetAllRetryPatterns()
	if err != nil {
		return nil, err
	}

	if len(retryStats) == 0 {
		return patterns, nil
	}

	// Collect all beads IDs to batch-fetch their status
	beadsIDs := make([]string, 0, len(retryStats))
	for _, stats := range retryStats {
		beadsIDs = append(beadsIDs, stats.BeadsID)
	}

	// Batch-fetch issue statuses from beads
	// This filters out closed issues that shouldn't be flagged as failures
	issueMap, _ := verify.GetIssuesBatch(beadsIDs, nil)
	// Ignore error - if beads is unavailable, we'll show all patterns
	// (better to show potential false positives than hide real issues)

	for _, stats := range retryStats {
		// Skip closed issues - they're resolved and shouldn't be flagged
		if issue, ok := issueMap[stats.BeadsID]; ok {
			status := strings.ToLower(issue.Status)
			if status == "closed" || status == "deferred" || status == "tombstone" {
				continue
			}
		}

		pattern := DetectedPattern{
			BeadsID: stats.BeadsID,
			Count:   stats.SpawnCount,
		}

		if stats.IsPersistentFailure() {
			pattern.Type = PatternTypePersistentFailure
			pattern.Severity = PatternSeverityCritical
			pattern.Title = fmt.Sprintf("Persistent failure: %s", stats.BeadsID)
			pattern.Description = fmt.Sprintf("Issue has failed %d times without success (%d abandoned, 0 completed)",
				stats.SpawnCount, stats.AbandonedCount)
			pattern.Suggestion = "Use 'orch spawn reliability-testing' to address underlying reliability issues"
		} else {
			pattern.Type = PatternTypeRetry
			pattern.Severity = PatternSeverityWarning
			pattern.Title = fmt.Sprintf("Retry pattern: %s", stats.BeadsID)
			pattern.Description = fmt.Sprintf("Issue has been respawned %d times (%d abandoned, %d completed)",
				stats.SpawnCount, stats.AbandonedCount, stats.CompletedCount)
			pattern.Suggestion = "Investigate root cause before spawning again"
		}

		if len(stats.Skills) > 0 {
			pattern.Details = append(pattern.Details, fmt.Sprintf("Skills used: %s", strings.Join(stats.Skills, ", ")))
		}
		if !stats.LastAttemptAt.IsZero() {
			pattern.LastSeen = stats.LastAttemptAt
			pattern.Details = append(pattern.Details, fmt.Sprintf("Last attempt: %s ago", formatDuration(time.Since(stats.LastAttemptAt))))
		}

		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

// collectGapPatterns collects recurring gap patterns from gap-tracker.json.
func collectGapPatterns() ([]DetectedPattern, error) {
	patterns := []DetectedPattern{}

	tracker, err := spawn.LoadTracker()
	if err != nil {
		return nil, err
	}

	// Get recurring gaps (3+ occurrences)
	suggestions := tracker.FindRecurringGaps()

	for _, s := range suggestions {
		pattern := DetectedPattern{
			Type:  PatternTypeRecurringGap,
			Query: s.Query,
			Count: s.Count,
		}

		// Determine severity based on gap priority and whether it's empty context
		hasNoContext := false
		for _, e := range s.Events {
			if e.GapType == string(spawn.GapTypeNoContext) {
				hasNoContext = true
				break
			}
		}

		if hasNoContext {
			pattern.Type = PatternTypeEmptyContext
			pattern.Severity = PatternSeverityCritical
			pattern.Title = fmt.Sprintf("Empty context: %q", s.Query)
			pattern.Description = fmt.Sprintf("Query %q has returned no results %d times", s.Query, s.Count)
			pattern.Suggestion = "Add knowledge via 'kb quick decide', 'kb quick constrain', or 'kb create investigation'"
		} else {
			if s.Priority == "high" {
				pattern.Severity = PatternSeverityWarning
			} else {
				pattern.Severity = PatternSeverityInfo
			}
			pattern.Title = fmt.Sprintf("Recurring gap: %q", s.Query)
			pattern.Description = s.Suggestion
			pattern.Suggestion = s.Command
		}

		// Extract time info from events
		for _, e := range s.Events {
			if pattern.FirstSeen.IsZero() || e.Timestamp.Before(pattern.FirstSeen) {
				pattern.FirstSeen = e.Timestamp
			}
			if e.Timestamp.After(pattern.LastSeen) {
				pattern.LastSeen = e.Timestamp
			}
		}

		// Add skill context
		skillSet := make(map[string]bool)
		for _, e := range s.Events {
			if e.Skill != "" {
				skillSet[e.Skill] = true
			}
		}
		if len(skillSet) > 0 {
			skills := make([]string, 0, len(skillSet))
			for skill := range skillSet {
				skills = append(skills, skill)
			}
			sort.Strings(skills)
			pattern.Details = append(pattern.Details, fmt.Sprintf("Affected skills: %s", strings.Join(skills, ", ")))
		}

		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

// sortPatterns sorts patterns by severity (critical first), then by count.
func sortPatterns(patterns []DetectedPattern) {
	sort.Slice(patterns, func(i, j int) bool {
		// Critical > Warning > Info
		severityOrder := map[PatternSeverity]int{
			PatternSeverityCritical: 0,
			PatternSeverityWarning:  1,
			PatternSeverityInfo:     2,
		}
		if severityOrder[patterns[i].Severity] != severityOrder[patterns[j].Severity] {
			return severityOrder[patterns[i].Severity] < severityOrder[patterns[j].Severity]
		}
		// Then by count (higher first)
		return patterns[i].Count > patterns[j].Count
	})
}

// printPatternsOutput prints the patterns output in human-readable format.
func printPatternsOutput(output PatternsOutput) {
	if output.TotalPatterns == 0 {
		fmt.Println("No behavioral patterns detected.")
		fmt.Println("This is good! The system is operating without notable friction.")
		return
	}

	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Println("  BEHAVIORAL PATTERNS - Orchestrator Awareness Report")
	fmt.Println("================================================================================")
	fmt.Println()

	// Summary
	fmt.Printf("  Total: %d patterns detected\n", output.TotalPatterns)
	if output.Critical > 0 {
		fmt.Printf("  Critical: %d (require immediate attention)\n", output.Critical)
	}
	if output.Warning > 0 {
		fmt.Printf("  Warning:  %d (should be addressed)\n", output.Warning)
	}
	if output.Info > 0 {
		fmt.Printf("  Info:     %d (for awareness)\n", output.Info)
	}
	fmt.Println()

	// Pattern sections by severity
	if output.Critical > 0 {
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println("  CRITICAL PATTERNS")
		fmt.Println("--------------------------------------------------------------------------------")
		for _, p := range output.Patterns {
			if p.Severity == PatternSeverityCritical {
				printPattern(p)
			}
		}
	}

	if output.Warning > 0 {
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println("  WARNING PATTERNS")
		fmt.Println("--------------------------------------------------------------------------------")
		for _, p := range output.Patterns {
			if p.Severity == PatternSeverityWarning {
				printPattern(p)
			}
		}
	}

	if output.Info > 0 && patternsVerbose {
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println("  INFO PATTERNS (verbose)")
		fmt.Println("--------------------------------------------------------------------------------")
		for _, p := range output.Patterns {
			if p.Severity == PatternSeverityInfo {
				printPattern(p)
			}
		}
	} else if output.Info > 0 {
		fmt.Printf("\n  (Use --verbose to see %d info-level patterns)\n", output.Info)
	}

	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Println("  Actions: Run 'orch learn' for suggestions, or address patterns directly.")
	fmt.Println("================================================================================")
}

// printPattern prints a single pattern with formatting.
func printPattern(p DetectedPattern) {
	// Icon based on severity
	icon := "   "
	switch p.Severity {
	case PatternSeverityCritical:
		icon = " ! "
	case PatternSeverityWarning:
		icon = " * "
	}

	fmt.Println()
	fmt.Printf("%s %s\n", icon, p.Title)
	fmt.Printf("      %s\n", p.Description)

	if patternsVerbose && len(p.Details) > 0 {
		for _, d := range p.Details {
			fmt.Printf("      - %s\n", d)
		}
	}

	if p.Suggestion != "" {
		fmt.Printf("      -> %s\n", p.Suggestion)
	}
}

// collectActionPatterns collects action outcome patterns from action-log.jsonl.
// These represent repeated futile actions (e.g., reading files that don't exist).
func collectActionPatterns() ([]DetectedPattern, error) {
	patterns := []DetectedPattern{}

	tracker, err := action.LoadTracker("")
	if err != nil {
		return nil, err
	}

	// Get action patterns (3+ occurrences of failed actions)
	actionPatterns := tracker.FindPatterns()

	for _, ap := range actionPatterns {
		pattern := DetectedPattern{
			Type:  PatternTypeFutileAction,
			Count: ap.Count,
		}

		// Determine severity based on outcome type AND count
		// Errors are more serious than empty results
		// Higher thresholds prevent noise from normal command patterns
		switch ap.Outcome {
		case action.OutcomeError:
			// Errors are actionable - they indicate real failures
			if ap.Count >= 10 {
				pattern.Severity = PatternSeverityCritical
			} else if ap.Count >= 5 {
				pattern.Severity = PatternSeverityWarning
			} else {
				pattern.Severity = PatternSeverityInfo
			}
		case action.OutcomeEmpty:
			// Empty results are often normal - be conservative with severity
			// Only flag as warning if truly excessive
			if ap.Count >= 15 {
				pattern.Severity = PatternSeverityWarning
			} else {
				pattern.Severity = PatternSeverityInfo
			}
		case action.OutcomeFallback:
			// Fallbacks indicate workarounds are needed
			if ap.Count >= 8 {
				pattern.Severity = PatternSeverityWarning
			} else {
				pattern.Severity = PatternSeverityInfo
			}
		default:
			pattern.Severity = PatternSeverityInfo
		}

		// Build title and description based on outcome type
		switch ap.Outcome {
		case action.OutcomeEmpty:
			pattern.Title = fmt.Sprintf("Empty result: %s on %s", ap.Tool, ap.Target)
			pattern.Description = fmt.Sprintf("Tool %s has returned empty results on %s %d times - target may not exist or has no content",
				ap.Tool, ap.Target, ap.Count)
		case action.OutcomeError:
			pattern.Title = fmt.Sprintf("Repeated error: %s on %s", ap.Tool, ap.Target)
			pattern.Description = fmt.Sprintf("Tool %s has failed on %s %d times - investigate underlying cause",
				ap.Tool, ap.Target, ap.Count)
		case action.OutcomeFallback:
			pattern.Title = fmt.Sprintf("Fallback pattern: %s on %s", ap.Tool, ap.Target)
			pattern.Description = fmt.Sprintf("Tool %s has required fallback on %s %d times - consider using alternative approach",
				ap.Tool, ap.Target, ap.Count)
		default:
			pattern.Title = fmt.Sprintf("Action pattern: %s on %s", ap.Tool, ap.Target)
			pattern.Description = fmt.Sprintf("Tool %s on %s has pattern with %d occurrences", ap.Tool, ap.Target, ap.Count)
		}

		// Add suggestion from action pattern
		if suggestion := ap.SuggestKbEntry(); suggestion != "" {
			pattern.Suggestion = suggestion
		}

		// Add workspace context if available
		if len(ap.Workspaces) > 0 {
			pattern.Details = append(pattern.Details, fmt.Sprintf("Workspaces: %s", strings.Join(ap.Workspaces, ", ")))
		}

		// Time range
		if !ap.FirstSeen.IsZero() && !ap.LastSeen.IsZero() {
			pattern.FirstSeen = ap.FirstSeen
			pattern.LastSeen = ap.LastSeen
			pattern.Details = append(pattern.Details,
				fmt.Sprintf("First: %s, Last: %s",
					ap.FirstSeen.Format("Jan 2 15:04"),
					ap.LastSeen.Format("Jan 2 15:04")))
		}

		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

// runPatternsSuppress suppresses a pattern by its index.
func runPatternsSuppress(indexArg string) error {
	// Parse index
	index, err := strconv.Atoi(indexArg)
	if err != nil {
		return fmt.Errorf("invalid index %q: must be a number", indexArg)
	}

	// Load action log and detect patterns
	log, err := patterns.LoadLog()
	if err != nil {
		return fmt.Errorf("failed to load action log: %w", err)
	}

	detected := log.DetectPatterns()
	if len(detected) == 0 {
		return fmt.Errorf("no patterns detected to suppress")
	}

	if index < 0 || index >= len(detected) {
		return fmt.Errorf("index %d out of range (0-%d)", index, len(detected)-1)
	}

	// Parse duration if provided
	var duration time.Duration
	if suppressDuration != "" {
		duration, err = parseSuppressDuration(suppressDuration)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", suppressDuration, err)
		}
	}

	// Suppress the pattern
	pattern := detected[index]
	log.SuppressPattern(pattern, suppressReason, duration)

	// Save the log
	if err := log.Save(); err != nil {
		return fmt.Errorf("failed to save action log: %w", err)
	}

	// Confirm to user
	fmt.Printf("Suppressed pattern: %s\n", pattern.Description)
	if suppressReason != "" {
		fmt.Printf("  Reason: %s\n", suppressReason)
	}
	if duration > 0 {
		fmt.Printf("  Expires: %s\n", time.Now().Add(duration).Format("2006-01-02 15:04"))
	} else {
		fmt.Println("  Duration: permanent")
	}

	return nil
}

// parseSuppressDuration parses a duration string with support for days (d).
// Standard Go durations like "1h", "30m" are supported, plus "d" for days.
func parseSuppressDuration(s string) (time.Duration, error) {
	// Handle days suffix
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid days value: %w", err)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	// Otherwise use standard duration parsing
	return time.ParseDuration(s)
}
