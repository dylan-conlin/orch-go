// stats_cmd.go - Aggregate events.jsonl metrics for orchestration observability
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	statsDays       int
	statsJSONOutput bool
	statsVerbose    bool
	statsSnapshot   bool
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show aggregated agent statistics from events.jsonl",
	Long: `Aggregate events.jsonl to surface orchestration metrics.

Shows:
  - Spawn and completion counts
  - Completion and abandonment rates
  - Average session duration
  - Skill effectiveness breakdown
  - Daemon health metrics

Examples:
  orch stats                    # Show last 7 days
  orch stats --days 1           # Show last 24 hours
  orch stats --days 30          # Show last 30 days
  orch stats --json             # Output as JSON for scripting
  orch stats --verbose          # Show additional metrics
  orch stats --snapshot         # Record gate accuracy baseline`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStats()
	},
}

func init() {
	statsCmd.Flags().IntVar(&statsDays, "days", 7, "Number of days to analyze")
	statsCmd.Flags().BoolVar(&statsJSONOutput, "json", false, "Output as JSON")
	statsCmd.Flags().BoolVar(&statsVerbose, "verbose", false, "Show additional metrics")
	statsCmd.Flags().BoolVar(&statsSnapshot, "snapshot", false, "Record gate accuracy baseline snapshot")
	rootCmd.AddCommand(statsCmd)
}

func runStats() error {
	// Get events file path
	eventsPath := getEventsPath()

	// Parse all events (time filtering happens in aggregateStats)
	events, err := parseEvents(eventsPath)
	if err != nil {
		return fmt.Errorf("failed to parse events: %w", err)
	}

	// Aggregate statistics
	report := aggregateStats(events, statsDays)

	// Snapshot mode: record gate accuracy baseline
	if statsSnapshot {
		return recordGateBaseline(report)
	}

	// Output
	if statsJSONOutput {
		return outputStatsJSON(report)
	}
	return outputStatsText(report)
}

func getEventsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orch/events.jsonl"
	}
	return filepath.Join(home, ".orch", "events.jsonl")
}

// parseEvents reads events from events.jsonl, returning all events.
// Time window filtering is done in aggregateStats to support multi-window metrics.
func parseEvents(path string) ([]StatsEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("events.jsonl not found at %s - no events recorded yet", path)
		}
		return nil, fmt.Errorf("failed to open events file: %w", err)
	}
	defer file.Close()

	var events []StatsEvent
	scanner := bufio.NewScanner(file)
	// Increase buffer size for potentially long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event StatsEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Skip malformed lines
			continue
		}

		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading events: %w", err)
	}

	return events, nil
}

func aggregateStats(events []StatsEvent, days int) *StatsReport {
	a := newStatsAggregator(days)

	for _, event := range events {
		if event.Timestamp >= a.cutoffDays {
			a.eventsInWindow++
		}

		switch event.Type {
		case "session.spawned":
			a.processSessionSpawned(event)
		case "session.completed":
			a.processSessionCompleted(event)
		case "agent.completed":
			a.processAgentCompleted(event)
		case "agent.abandoned":
			a.processAgentAbandoned(event)
		case "daemon.spawn":
			a.processDaemonSpawn(event)
		case "session.auto_completed":
			a.processAutoCompleted(event)
		case "spawn.triage_bypassed":
			a.processTriageBypassed(event)
		case "spawn.hotspot_bypassed":
			a.processHotspotBypassed(event)
		case "spawn.verification_bypassed":
			a.processVerificationBypassed(event)
		case "agent.wait.complete":
			a.processWaitComplete(event)
		case "agent.wait.timeout":
			a.processWaitTimeout(event)
		case "session.orchestrator.started", "session.started":
			a.processSessionStarted(event)
		case "session.orchestrator.ended", "session.ended":
			a.processSessionEnded(event)
		case "verification.failed":
			a.processVerificationFailed(event)
		case "verification.bypassed":
			a.processVerificationBypassedEvent(event)
		case "verification.auto_skipped":
			a.processVerificationAutoSkipped(event)
		case "spawn.gate_decision":
			a.processGateDecision(event)
		case "daemon.architect_escalation":
			a.processArchitectEscalation(event)
		case "spawn.skill_inferred":
			a.processSkillInferred(event)
		}
	}

	a.report.EventsAnalyzed = a.eventsInWindow

	a.calcEscapeHatchStats()
	a.calcVerificationStats()
	a.calcSpawnGateStats()
	a.calcOverrideStats()
	a.calcRatesAndDuration()
	a.calcGateDecisionStats()
	a.calcGateEffectiveness(events)
	a.calcSkillInferenceStats()
	a.calcCoachingStats()

	return a.report
}

// extractGateAccuracyBaseline extracts a point-in-time snapshot of gate accuracy metrics
// from a stats report. Used for prospective measurement: compare baselines over time
// to answer "do gates improve agent quality?"
func extractGateAccuracyBaseline(report *StatsReport) GateAccuracyBaseline {
	ge := report.GateEffectivenessStats
	return GateAccuracyBaseline{
		SnapshotTime:             report.GeneratedAt,
		DaysAnalyzed:             report.DaysAnalyzed,
		TotalSpawns:              report.Summary.TotalSpawns,
		TotalCompletions:         report.Summary.TotalCompletions,
		GatedSpawns:              ge.GatedCompletion.TotalSpawns,
		UngatedSpawns:            ge.UngatedCompletion.TotalSpawns,
		GateDecisions:            report.GateDecisionStats.TotalDecisions,
		TotalBlocks:              ge.TotalBlocks,
		TotalBypasses:            ge.TotalBypasses,
		GatedCompletionRate:      ge.GatedCompletion.CompletionRate,
		UngatedCompletionRate:    ge.UngatedCompletion.CompletionRate,
		GatedVerificationRate:    ge.GatedCompletion.VerificationRate,
		UngatedVerificationRate:  ge.UngatedCompletion.VerificationRate,
		GatedAvgDuration:         ge.GatedCompletion.AvgDurationMinutes,
		UngatedAvgDuration:       ge.UngatedCompletion.AvgDurationMinutes,
		SpawnGateBypassRate:      report.SpawnGateStats.BypassRate,
		VerificationFirstTryRate: report.VerificationStats.PassRate,
	}
}

// recordGateBaseline records a gate accuracy baseline snapshot to ~/.orch/gate-baselines.jsonl.
// Each line is a JSON object with the baseline metrics at the time of recording.
func recordGateBaseline(report *StatsReport) error {
	baseline := extractGateAccuracyBaseline(report)

	data, err := json.Marshal(baseline)
	if err != nil {
		return fmt.Errorf("failed to marshal baseline: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	baselinePath := filepath.Join(home, ".orch", "gate-baselines.jsonl")
	f, err := os.OpenFile(baselinePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open baseline file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write baseline: %w", err)
	}

	// Print summary
	fmt.Printf("Gate accuracy baseline recorded to %s\n\n", baselinePath)
	fmt.Printf("  Period:           %s (%d days)\n", baseline.SnapshotTime, baseline.DaysAnalyzed)
	fmt.Printf("  Total spawns:     %d (%d gated, %d ungated)\n", baseline.TotalSpawns, baseline.GatedSpawns, baseline.UngatedSpawns)
	fmt.Printf("  Gate decisions:   %d (blocks: %d, bypasses: %d)\n", baseline.GateDecisions, baseline.TotalBlocks, baseline.TotalBypasses)
	fmt.Println()
	fmt.Printf("  Gated completion rate:      %.1f%%\n", baseline.GatedCompletionRate)
	fmt.Printf("  Ungated completion rate:    %.1f%%\n", baseline.UngatedCompletionRate)
	fmt.Printf("  Gated verification rate:    %.1f%%\n", baseline.GatedVerificationRate)
	fmt.Printf("  Ungated verification rate:  %.1f%%\n", baseline.UngatedVerificationRate)
	fmt.Printf("  Gated avg duration:         %.0fm\n", baseline.GatedAvgDuration)
	fmt.Printf("  Ungated avg duration:       %.0fm\n", baseline.UngatedAvgDuration)
	fmt.Printf("  Spawn gate bypass rate:     %.1f%%\n", baseline.SpawnGateBypassRate)
	fmt.Printf("  Verification 1st try rate:  %.1f%%\n", baseline.VerificationFirstTryRate)

	// Load and compare with previous baseline
	baselines, err := loadGateBaselines(baselinePath)
	if err == nil && len(baselines) > 1 {
		prev := baselines[len(baselines)-2]
		fmt.Println()
		fmt.Println("  Delta from previous baseline:")
		printDelta("  Gated verification rate", prev.GatedVerificationRate, baseline.GatedVerificationRate)
		printDelta("  Ungated verification rate", prev.UngatedVerificationRate, baseline.UngatedVerificationRate)
		printDelta("  Spawn gate bypass rate", prev.SpawnGateBypassRate, baseline.SpawnGateBypassRate)
		printDelta("  Verification 1st try rate", prev.VerificationFirstTryRate, baseline.VerificationFirstTryRate)
	}

	return nil
}

func printDelta(label string, prev, curr float64) {
	delta := curr - prev
	direction := "="
	if delta > 0.5 {
		direction = "+"
	} else if delta < -0.5 {
		direction = "-"
	}
	fmt.Printf("    %-30s %6.1f%% -> %6.1f%% (%s%.1f%%)\n", label, prev, curr, direction, delta)
}

func loadGateBaselines(path string) ([]GateAccuracyBaseline, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var baselines []GateAccuracyBaseline
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var b GateAccuracyBaseline
		if err := json.Unmarshal([]byte(line), &b); err != nil {
			continue
		}
		baselines = append(baselines, b)
	}
	return baselines, scanner.Err()
}
