package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/spf13/cobra"
)

var (
	learnShowPatterns bool
	learnShowSkills   bool
	learnShowEffects  bool
)

var learnCmd = &cobra.Command{
	Use:   "learn",
	Short: "Review and act on system learning suggestions",
	Long: `Review recurring context gaps and get suggestions for improvement.

The learning loop tracks gaps detected during spawns and suggests:
- Creating beads issues for recurring gaps
- Adding kn entries (decide/constrain) for missing knowledge
- Spawning investigations for unclear areas

Subcommands:
  orch learn                         Show all suggestions (default)
  orch learn suggest                 Show suggestions with commands
  orch learn patterns                Analyze gap patterns by topic
  orch learn skills                  Show gap rates by skill
  orch learn effects                 Show effectiveness of past improvements
  orch learn act [index]             Run the suggested command for a gap
  orch learn resolve [index] [type]  Mark a gap as resolved manually
  orch learn clear                   Clear gap history (use sparingly)

Examples:
  orch learn                         # Show suggestions
  orch learn act 1                   # Run first suggestion's command
  orch learn resolve 2 added_knowledge  # Mark as resolved without running command
  orch learn patterns                # Analyze gap patterns
  orch learn effects                 # Check if improvements reduced gaps`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLearnSuggest()
	},
}

var learnSuggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Show suggestions for recurring gaps",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLearnSuggest()
	},
}

var learnPatternsCmd = &cobra.Command{
	Use:   "patterns",
	Short: "Analyze gap patterns by topic",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLearnPatterns()
	},
}

var learnSkillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Show gap rates by skill",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLearnSkills()
	},
}

var learnEffectsCmd = &cobra.Command{
	Use:   "effects",
	Short: "Show effectiveness of past improvements",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLearnEffects()
	},
}

var learnActCmd = &cobra.Command{
	Use:   "act [index]",
	Short: "Run the suggested command for a gap",
	Long: `Run the suggested command for a recurring gap.

The index corresponds to the suggestion number shown by 'orch learn'.
This will execute the suggested kn, bd, or orch command.

Example:
  orch learn           # Shows: [1] auth (3x) - suggest: kn decide...
  orch learn act 1     # Runs the kn decide command for auth`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLearnAct(args[0])
	},
}

var learnResolveCmd = &cobra.Command{
	Use:   "resolve [index] [resolution]",
	Short: "Mark a gap as resolved without running a command",
	Long: `Mark a recurring gap as resolved with a custom resolution.

This is useful when you've already addressed a gap through other means,
or when the suggested command isn't appropriate for your situation.

Resolution types:
  added_knowledge   - Added kn entry or documentation
  created_issue     - Created a beads issue to track
  investigated      - Completed investigation
  wont_fix          - Decided not to address
  custom            - Any other resolution (requires details)

Example:
  orch learn           # Shows: [1] auth (3x) - suggest: kn decide...
  orch learn resolve 1 added_knowledge
  orch learn resolve 2 wont_fix "Not relevant to current project"`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		details := ""
		if len(args) == 3 {
			details = args[2]
		}
		return runLearnResolve(args[0], args[1], details)
	},
}

var learnClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear gap history (use sparingly)",
	Long: `Clear all gap tracking history.

This resets the learning loop's memory of past gaps.
Use sparingly - the learning loop needs history to detect patterns.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLearnClear()
	},
}

func init() {
	learnCmd.AddCommand(learnSuggestCmd)
	learnCmd.AddCommand(learnPatternsCmd)
	learnCmd.AddCommand(learnSkillsCmd)
	learnCmd.AddCommand(learnEffectsCmd)
	learnCmd.AddCommand(learnActCmd)
	learnCmd.AddCommand(learnResolveCmd)
	learnCmd.AddCommand(learnClearCmd)
	rootCmd.AddCommand(learnCmd)
}

func runLearnSuggest() error {
	tracker, err := spawn.LoadTracker()
	if err != nil {
		return fmt.Errorf("failed to load gap tracker: %w", err)
	}

	if len(tracker.Events) == 0 {
		fmt.Println("No gaps tracked yet. Gaps are recorded when spawning with sparse context.")
		return nil
	}

	fmt.Printf("Gap tracker: %s\n\n", tracker.Summary())

	suggestions := tracker.FindRecurringGaps()
	if len(suggestions) == 0 {
		fmt.Println("No recurring patterns found. Gaps must occur 3+ times to trigger suggestions.")
		return nil
	}

	fmt.Println(spawn.FormatSuggestions(suggestions))

	// Also show numbered list for act command
	fmt.Println("\nTo act on a suggestion:")
	for i, s := range suggestions {
		if i >= 5 {
			fmt.Printf("  ... and %d more (use 'orch learn patterns' for full view)\n", len(suggestions)-5)
			break
		}
		fmt.Printf("  [%d] %s (%dx)\n", i+1, s.Query, s.Count)
		if s.Command != "" {
			fmt.Printf("      $ %s\n", s.Command)
		}
	}

	return nil
}

func runLearnPatterns() error {
	tracker, err := spawn.LoadTracker()
	if err != nil {
		return fmt.Errorf("failed to load gap tracker: %w", err)
	}

	if len(tracker.Events) == 0 {
		fmt.Println("No gaps tracked yet.")
		return nil
	}

	analyses := tracker.AnalyzePatterns()

	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  📊 GAP PATTERN ANALYSIS                                                     ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════════════════════╣")

	for i, a := range analyses {
		if i >= 10 {
			fmt.Printf("║  ... and %d more topics                                                    ║\n", len(analyses)-10)
			break
		}

		trendIcon := "→"
		switch a.Trend {
		case "increasing":
			trendIcon = "↗"
		case "decreasing":
			trendIcon = "↘"
		}

		fmt.Printf("║  %s %-30s Total: %3d  Recent: %3d  Critical: %2d     ║\n",
			trendIcon, truncateString(a.Topic, 30), a.TotalGaps, a.RecentGaps, a.CriticalGaps)

		if len(a.Skills) > 0 {
			skillStr := strings.Join(a.Skills, ", ")
			fmt.Printf("║     Skills: %-60s ║\n", truncateString(skillStr, 60))
		}
	}

	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")

	return nil
}

func runLearnSkills() error {
	tracker, err := spawn.LoadTracker()
	if err != nil {
		return fmt.Errorf("failed to load gap tracker: %w", err)
	}

	if len(tracker.Events) == 0 {
		fmt.Println("No gaps tracked yet.")
		return nil
	}

	rates := tracker.GetSkillGapRates()

	// Sort by count
	type skillRate struct {
		skill string
		count int
	}
	var sorted []skillRate
	for skill, count := range rates {
		sorted = append(sorted, skillRate{skill, count})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  📊 GAP RATES BY SKILL                                                       ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════════════════════╣")

	for _, sr := range sorted {
		bar := strings.Repeat("█", min(sr.count, 40))
		fmt.Printf("║  %-25s %3d  [%-40s] ║\n", sr.skill, sr.count, bar)
	}

	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")

	return nil
}

func runLearnEffects() error {
	tracker, err := spawn.LoadTracker()
	if err != nil {
		return fmt.Errorf("failed to load gap tracker: %w", err)
	}

	improvements := tracker.MeasureImprovementEffectiveness()

	if len(improvements) == 0 {
		fmt.Println("No improvements tracked yet.")
		fmt.Println("Improvements are recorded when you act on suggestions (orch learn act).")
		return nil
	}

	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  📈 IMPROVEMENT EFFECTIVENESS                                                ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════════════════════╣")

	for _, imp := range improvements {
		delta := imp.GapCountBefore - imp.GapCountAfter
		deltaStr := fmt.Sprintf("%+d", -delta) // Show reduction as negative
		if delta > 0 {
			deltaStr = fmt.Sprintf("✓ -%d", delta)
		} else if delta == 0 {
			deltaStr = "→ 0"
		}

		fmt.Printf("║  %-12s %-30s Before: %2d  After: %2d  (%s)   ║\n",
			imp.Type, truncateString(imp.Query, 30), imp.GapCountBefore, imp.GapCountAfter, deltaStr)
		fmt.Printf("║     Reference: %-60s ║\n", truncateString(imp.Reference, 60))
	}

	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")

	return nil
}

func runLearnAct(indexStr string) error {
	var index int
	if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
		return fmt.Errorf("invalid index: %s", indexStr)
	}

	tracker, err := spawn.LoadTracker()
	if err != nil {
		return fmt.Errorf("failed to load gap tracker: %w", err)
	}

	suggestions := tracker.FindRecurringGaps()
	if len(suggestions) == 0 {
		fmt.Println("No suggestions available.")
		return nil
	}

	if index < 1 || index > len(suggestions) {
		return fmt.Errorf("index %d out of range (1-%d)", index, len(suggestions))
	}

	s := suggestions[index-1]

	if s.Command == "" {
		fmt.Printf("Suggestion %d has no executable command.\n", index)
		return nil
	}

	fmt.Printf("Running: %s\n\n", s.Command)

	// Parse and execute the command
	parts := strings.Fields(s.Command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	// Record the improvement
	tracker.RecordImprovement(s.Type, s.Query, s.Command)
	tracker.RecordResolution(s.Query, "added_knowledge", s.Command)

	if err := tracker.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save improvement record: %v\n", err)
	}

	fmt.Printf("\n✓ Recorded improvement for %q\n", s.Query)
	fmt.Println("Run 'orch learn effects' later to see if this reduced gaps.")

	return nil
}

func runLearnResolve(indexStr, resolution, details string) error {
	var index int
	if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
		return fmt.Errorf("invalid index: %s", indexStr)
	}

	// Validate resolution type
	validResolutions := map[string]bool{
		"added_knowledge": true,
		"created_issue":   true,
		"investigated":    true,
		"wont_fix":        true,
		"custom":          true,
	}
	if !validResolutions[resolution] {
		return fmt.Errorf("invalid resolution type %q. Valid types: added_knowledge, created_issue, investigated, wont_fix, custom", resolution)
	}

	if resolution == "custom" && details == "" {
		return fmt.Errorf("custom resolution requires details as third argument")
	}

	tracker, err := spawn.LoadTracker()
	if err != nil {
		return fmt.Errorf("failed to load gap tracker: %w", err)
	}

	suggestions := tracker.FindRecurringGaps()
	if len(suggestions) == 0 {
		fmt.Println("No suggestions available.")
		return nil
	}

	if index < 1 || index > len(suggestions) {
		return fmt.Errorf("index %d out of range (1-%d)", index, len(suggestions))
	}

	s := suggestions[index-1]

	// Build resolution details
	resolutionDetails := resolution
	if details != "" {
		resolutionDetails = fmt.Sprintf("%s: %s", resolution, details)
	}

	// Record the resolution for all events in this gap pattern
	tracker.RecordResolution(s.Query, resolution, resolutionDetails)

	// Record as improvement to track effectiveness
	tracker.RecordImprovement(resolution, s.Query, resolutionDetails)

	if err := tracker.Save(); err != nil {
		return fmt.Errorf("failed to save resolution: %w", err)
	}

	fmt.Printf("✓ Marked %q as resolved (%s)\n", s.Query, resolution)
	if details != "" {
		fmt.Printf("  Details: %s\n", details)
	}
	fmt.Println("Run 'orch learn effects' later to see if this reduced gaps.")

	return nil
}

func runLearnClear() error {
	tracker := &spawn.GapTracker{Events: []spawn.GapEvent{}}

	if err := tracker.Save(); err != nil {
		return fmt.Errorf("failed to clear gap tracker: %w", err)
	}

	fmt.Println("Gap history cleared.")
	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
