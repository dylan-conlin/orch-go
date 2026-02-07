package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/stability"
	"github.com/spf13/cobra"
)

var (
	stabilityDays int
	stabilityJSON bool
)

var stabilityCmd = &cobra.Command{
	Use:   "stability",
	Short: "Show system stability report (Phase 3 reliability tracking)",
	Long: `Report system stability for Phase 3 reliability tracking.

Shows:
  - Current clean-session streak (time since last manual recovery)
  - Progress toward 7-day target
  - Recent manual recovery interventions
  - Health snapshot statistics

Data comes from ~/.orch/stability.jsonl, recorded automatically by
the doctor daemon (orch doctor --daemon).

Examples:
  orch stability              # Show stability report
  orch stability --json       # JSON output for scripting
  orch stability --days 30    # Show last 30 days`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStability()
	},
}

func init() {
	stabilityCmd.Flags().IntVar(&stabilityDays, "days", 7, "Number of days to analyze")
	stabilityCmd.Flags().BoolVar(&stabilityJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(stabilityCmd)
}

func runStability() error {
	report, err := stability.ComputeReport(stability.DefaultPath(), stabilityDays)
	if err != nil {
		return fmt.Errorf("failed to compute stability report: %w", err)
	}

	if stabilityJSON {
		return printStabilityJSON(report)
	}
	return printStabilityHuman(report)
}

func printStabilityJSON(report *stability.Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func printStabilityHuman(report *stability.Report) error {
	fmt.Println("Stability Report (Phase 3 Reliability Tracking)")
	fmt.Println("================================================")
	fmt.Println()

	if !report.HasData {
		fmt.Println("No stability data found.")
		fmt.Println()
		fmt.Println("The doctor daemon records stability snapshots automatically.")
		fmt.Println("Start it with: orch doctor --daemon")
		return nil
	}

	// Current streak
	fmt.Printf("Current streak:     %s (target: %s)\n",
		stability.FormatDuration(report.CurrentStreak),
		stability.FormatDuration(report.TargetDuration))

	// Progress bar
	bar := stability.ProgressBar(report.ProgressPercent, 20)
	fmt.Printf("Phase 3 progress:   %s %.0f%%\n", bar, report.ProgressPercent)

	if report.ProgressPercent >= 100 {
		fmt.Println()
		fmt.Println("TARGET REACHED! System has been stable for 1 week.")
		fmt.Println("Reliability focus can be lifted; feature work may resume.")
	}

	fmt.Println()

	// Interventions
	if len(report.Interventions) == 0 {
		fmt.Printf("Last %d days: 0 interventions\n", stabilityDays)
	} else {
		fmt.Printf("Last %d days: %d intervention(s)\n", stabilityDays, len(report.Interventions))
		for _, iv := range report.Interventions {
			t := time.Unix(iv.Ts, 0)
			fmt.Printf("  %s  %-18s %s\n",
				t.Format("2006-01-02 15:04"),
				iv.Source,
				iv.Detail)
		}
	}

	fmt.Println()

	// Health snapshots
	if report.SnapshotsTotal > 0 {
		fmt.Printf("Health snapshots:   %d recorded\n", report.SnapshotsTotal)
		fmt.Printf("  Healthy:          %d/%d (%.1f%%)\n",
			report.SnapshotsHealthy, report.SnapshotsTotal, report.HealthPercent)
	}

	return nil
}
