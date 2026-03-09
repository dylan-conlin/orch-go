package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dylan-conlin/orch-go/pkg/dupdetect"
	"github.com/dylan-conlin/orch-go/pkg/entropy"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/spf13/cobra"
)

var (
	entropyDays       int
	entropyJSON       bool
	entropySkipDupdet bool
	entropySkipLint   bool
	entropyOutputDir  string
)

var entropyCmd = &cobra.Command{
	Use:   "entropy",
	Short: "Analyze codebase entropy — growth trends, duplication, health signals",
	Long: `Aggregate signals from git history, events.jsonl, bloat analysis, duplication
detection, and architecture lint into an entropy health report.

This is Harness Layer 3 — combining signals from:
  - Fix:Feat commit ratio (entropy spiral indicator)
  - Commit velocity (verification bandwidth check)
  - File bloat scan (>800 line files)
  - Duplicate function detection (code clone accumulation)
  - Architecture lint results (structural test health)
  - Override/bypass trends (gate effectiveness)
  - Agent event stats (abandonment, rework rates)

Health levels (from entropy spiral model):
  healthy:   Fix:Feat < 0.5:1, velocity manageable
  degrading: Fix:Feat 0.5-0.9:1, signals weakening
  spiral:    Fix:Feat > 0.9:1, agents fixing agents' work

Examples:
  orch entropy                  # Full analysis, 28-day window
  orch entropy --days 7         # Last 7 days
  orch entropy --json           # Machine-readable output
  orch entropy --skip-dupdetect # Skip expensive AST scan
  orch entropy --output-dir .kb/entropy  # Save JSON to dated file`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEntropy()
	},
}

func init() {
	entropyCmd.Flags().IntVar(&entropyDays, "days", 28, "Analysis window in days")
	entropyCmd.Flags().BoolVar(&entropyJSON, "json", false, "Output JSON format")
	entropyCmd.Flags().BoolVar(&entropySkipDupdet, "skip-dupdetect", false, "Skip duplicate detection (faster)")
	entropyCmd.Flags().BoolVar(&entropySkipLint, "skip-lint", false, "Skip architecture lint tests (faster)")
	entropyCmd.Flags().StringVar(&entropyOutputDir, "output-dir", "", "Save JSON report to dated file in this directory")
	rootCmd.AddCommand(entropyCmd)
}

func runEntropy() error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	eventsPath := events.DefaultLogPath()

	report, err := entropy.Analyze(projectDir, entropyDays, eventsPath)
	if err != nil {
		return fmt.Errorf("entropy analysis: %w", err)
	}

	// Duplicate detection (optional, expensive)
	if !entropySkipDupdet {
		det := dupdetect.NewDetector()
		pairs, scanErr := det.ScanProject(projectDir)
		if scanErr == nil {
			report.DuplicatePairCount = len(pairs)
		}
	}

	// Architecture lint (optional)
	if !entropySkipLint {
		passed, failed, lintErr := entropy.RunArchLintTests(projectDir)
		if lintErr == nil {
			if failed > 0 {
				report.Recommendations = append(report.Recommendations, entropy.Recommendation{
					Severity: "critical",
					Signal:   "arch_lint",
					Message:  fmt.Sprintf("Architecture lint: %d passed, %d FAILED. Structural invariants broken.", passed, failed),
				})
			}
		}
	}

	// Re-generate recommendations after dupdetect count is available
	// (the initial Analyze call didn't have it)
	// Only needed if dupdetect found pairs worth recommending
	if report.DuplicatePairCount > 5 && !entropySkipDupdet {
		// Check if duplication recommendation already exists
		hasDupRec := false
		for _, r := range report.Recommendations {
			if r.Signal == "duplication" {
				hasDupRec = true
				break
			}
		}
		if !hasDupRec {
			report.Recommendations = append(report.Recommendations, entropy.Recommendation{
				Severity: "warning",
				Signal:   "duplication",
				Message:  fmt.Sprintf("%d duplicate function pairs detected. Run 'orch dupdetect' for extraction candidates.", report.DuplicatePairCount),
			})
		}
	}

	// Save to file if --output-dir specified
	if entropyOutputDir != "" {
		path, err := entropy.SaveReport(report, entropyOutputDir)
		if err != nil {
			return fmt.Errorf("save report: %w", err)
		}
		fmt.Printf("Report saved to %s\n", path)
		return nil
	}

	if entropyJSON {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("json marshal: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Print(entropy.FormatText(report))
	return nil
}
