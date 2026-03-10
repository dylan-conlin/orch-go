package main

import (
	"github.com/spf13/cobra"
)

var (
	healthJSON bool
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Show harness health score with trend analysis",
	Long: `Display composite harness health score (0-100) with longitudinal trend tracking.

The health score is computed from 5 dimensions:
  - Gate coverage: fraction of enforcement gates active (pre-commit, spawn, completion, etc.)
  - Accretion control: bloated file count (files >800 lines)
  - Fix:feat balance: ratio of fix: to feat: commits (28-day window)
  - Hotspot control: active hotspot count (fix-density + bloat)
  - Bloat percentage: structural health via bloated file distribution

Each snapshot is stored for longitudinal tracking. Trends are computed via
linear regression over the last 30 snapshots.

Grades: A (90+), B (80-89), C (65-79), D (50-64), F (<50)

Examples:
  orch health          # Show health score with trends
  orch health --json   # Machine-readable JSON output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Reuse the doctor health flags for JSON output
		doctorHealthJSON = healthJSON
		return runDoctorHealth()
	},
}

func init() {
	healthCmd.Flags().BoolVar(&healthJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(healthCmd)
}
