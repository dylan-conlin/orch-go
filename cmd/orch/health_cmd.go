package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/orient"
	"github.com/spf13/cobra"
)

var (
	healthJSON bool
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Show harness health score with operational metrics",
	Long: `Display composite harness health score (0-100) with operational metrics.

Includes harness health dimensions (gate coverage, accretion, fix:feat, hotspots)
plus operational metrics previously in orient:
  - Throughput (completions, abandonments, avg duration)
  - Changelog since last session
  - Relevant and stale models
  - Daemon health signals
  - Divergence alerts
  - Adoption drift
  - Explore candidates
  - Reflection suggestions

Examples:
  orch health          # Show health score + operational metrics
  orch health --json   # Machine-readable JSON output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		emitCommandInvoked("health", flagsFromCmd(cmd)...)
		// Reuse the doctor health flags for JSON output
		doctorHealthJSON = healthJSON
		if err := runDoctorHealth(); err != nil {
			return err
		}

		// Append operational metrics (moved from orient thinking surface)
		if !healthJSON {
			fmt.Println()
			fmt.Print(collectAndFormatOperationalHealth())
		}
		return nil
	},
}

func init() {
	healthCmd.Flags().BoolVar(&healthJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(healthCmd)
}

// collectAndFormatOperationalHealth collects operational metrics and renders them
// using FormatHealth. These are the sections that were moved out of orient.
func collectAndFormatOperationalHealth() string {
	now := time.Now()
	projectDir, _ := os.Getwd()

	data := &orient.OrientationData{}

	// Previous session (needed for changelog date)
	sessionsDir := filepath.Join(projectDir, ".kb", "sessions")
	data.PreviousSession = collectPreviousSession(sessionsDir)

	// Throughput
	data.Throughput = collectThroughput(now)
	data.Throughput.InProgress = collectInProgressCount()
	enrichThroughputWithGitGroundTruth(&data.Throughput)

	// Models
	modelsDir := filepath.Join(projectDir, ".kb", "models")
	allModels, err := orient.ScanModelFreshness(modelsDir)
	if err == nil {
		data.RelevantModels = selectRelevantModels(allModels, 3)
		data.StaleModels = orient.FilterStaleModels(allModels, 2)
	}

	// Changelog
	data.Changelog = collectChangelog(data.PreviousSession)

	// Health summary
	data.HealthSummary = collectHealthSummary()

	// Daemon health
	data.DaemonHealth = collectDaemonHealth(now)

	// Reflect suggestions
	data.ReflectSummary = collectReflectSuggestions()
	enrichReflectWithSessionOrphans(data.ReflectSummary, data.PreviousSession, projectDir)

	// Divergence
	data.DivergenceAlerts = computeDivergenceAlerts(data)

	// Explore candidates
	data.ExploreCandidates = collectExploreCandidates(projectDir, modelsDir, now)

	// Adoption drift
	data.AdoptionDrift = collectAdoptionDrift(projectDir)

	return orient.FormatHealth(data)
}
