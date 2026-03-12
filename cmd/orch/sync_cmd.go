package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/artifactsync"
	"github.com/spf13/cobra"
)

var (
	syncDryRun bool
	syncFix    bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Check and fix artifact drift against ARTIFACT_MANIFEST.yaml",
	Long: `Analyze recent drift events to find artifacts that may be stale.

Drift events are captured at completion time (Phase 1) and logged to
~/.orch/artifact-drift.jsonl. This command cross-references those events
against ARTIFACT_MANIFEST.yaml to identify which artifacts need updating.

Examples:
  orch sync              # Show drift report (same as --dry-run)
  orch sync --dry-run    # Show what's drifted without fixing
  orch sync --fix        # Spawn artifact-sync agent to update drifted artifacts`,
	RunE: runSync,
}

func init() {
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "Show drift report without spawning sync agent")
	syncCmd.Flags().BoolVar(&syncFix, "fix", false, "Spawn artifact-sync agent to update drifted artifacts")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	// Determine project directory
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Load manifest
	manifest, err := artifactsync.LoadManifest(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Read drift events
	driftLogPath := artifactsync.DefaultDriftLogPath()
	events, err := artifactsync.ReadDriftEvents(driftLogPath)
	if err != nil {
		return fmt.Errorf("failed to read drift events: %w", err)
	}

	if len(events) == 0 {
		fmt.Println("No drift events found. Artifacts are up to date.")
		return nil
	}

	// Analyze drift
	report := artifactsync.AnalyzeDrift(manifest, events)

	if len(report.Entries) == 0 {
		fmt.Printf("Found %d drift events but none match manifest triggers. Artifacts are up to date.\n", len(events))
		return nil
	}

	// Print report
	printDriftReport(report, events)

	// If --fix, spawn the sync agent
	if syncFix {
		return spawnSyncAgent(report)
	}

	return nil
}

func printDriftReport(report *artifactsync.DriftReport, allEvents []artifactsync.DriftEvent) {
	fmt.Printf("Artifact Drift Report (%d events, %d affected artifacts/sections)\n", len(allEvents), len(report.Entries))
	fmt.Println(strings.Repeat("─", 60))

	for _, entry := range report.Entries {
		label := entry.ArtifactPath
		if entry.SectionName != "" {
			label = fmt.Sprintf("%s:%s", entry.ArtifactPath, entry.SectionName)
		}

		// Collect unique beads IDs from events
		beadsIDs := make([]string, 0)
		seen := make(map[string]bool)
		for _, ev := range entry.Events {
			if ev.BeadsID != "" && !seen[ev.BeadsID] {
				seen[ev.BeadsID] = true
				beadsIDs = append(beadsIDs, ev.BeadsID)
			}
		}

		fmt.Printf("\n  %s\n", label)
		fmt.Printf("    Triggers: %s\n", strings.Join(entry.Triggers, ", "))
		fmt.Printf("    Events:   %d (%s)\n", len(entry.Events), strings.Join(beadsIDs, ", "))
	}

	fmt.Println()
}

func buildSyncTask(report *artifactsync.DriftReport) string {
	var lines []string
	lines = append(lines, "Update the following drifted artifacts based on recent code changes:")
	lines = append(lines, "")

	for _, entry := range report.Entries {
		label := entry.ArtifactPath
		if entry.SectionName != "" {
			label = fmt.Sprintf("%s:%s", entry.ArtifactPath, entry.SectionName)
		}

		// Collect commit ranges
		var commitRanges []string
		seen := make(map[string]bool)
		for _, ev := range entry.Events {
			if ev.CommitRange != "" && !seen[ev.CommitRange] {
				seen[ev.CommitRange] = true
				commitRanges = append(commitRanges, ev.CommitRange)
			}
		}

		line := fmt.Sprintf("- %s (triggers: %s)", label, strings.Join(entry.Triggers, ", "))
		if len(commitRanges) > 0 {
			line += fmt.Sprintf(" [commits: %s]", strings.Join(commitRanges, ", "))
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func spawnSyncAgent(report *artifactsync.DriftReport) error {
	task := buildSyncTask(report)

	fmt.Println("Spawning artifact-sync agent...")

	// Spawn with artifact-sync skill, light tier, bypass triage
	spawnLight = true
	spawnBypassTriage = true
	return runSpawnWithSkill(serverURL, "artifact-sync", task, false, false, true, false)
}
