package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	harnessAuditDays int
	harnessAuditJSON bool
)

var harnessAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit gate invocations, fire rates, costs, and anomalies",
	Long: `Analyze spawn.gate_decision events to produce per-gate metrics.

Shows per-gate table with:
  - Invocation count
  - Fire rate (blocked + bypassed / total)
  - Mean cost (ms) from pipeline_timing in agent.completed events
  - Coverage (% of spawns that passed through this gate)

Flags anomalies:
  - Gates with zero fires in the analysis period
  - Gates costing >1s mean
  - Gates with <50% coverage (not evaluating most spawns)

Examples:
  orch harness audit              # Last 30 days
  orch harness audit --days 7     # Last 7 days
  orch harness audit --json       # Machine-readable output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		emitCommandInvoked("harness audit", flagsFromCmd(cmd)...)
		return runHarnessAudit()
	},
}

func init() {
	harnessAuditCmd.Flags().IntVar(&harnessAuditDays, "days", 30, "Number of days to analyze")
	harnessAuditCmd.Flags().BoolVar(&harnessAuditJSON, "json", false, "Output as JSON")
	harnessCmd.AddCommand(harnessAuditCmd)
}

// GateAuditResult is the top-level result for harness audit.
type GateAuditResult struct {
	GeneratedAt  string            `json:"generated_at"`
	DaysAnalyzed int               `json:"days_analyzed"`
	TotalSpawns  int               `json:"total_spawns"`
	Gates        []GateAuditEntry  `json:"gates"`
	DeadSignals  []DeadSignalEntry `json:"dead_signals,omitempty"`
	Anomalies    []GateAnomaly     `json:"anomalies"`
}

// DeadSignalEntry flags a feedback channel with zero events over N completions.
// Zero events + many completions = channel is likely dead (broken/disconnected),
// not clean (nothing to report).
type DeadSignalEntry struct {
	Channel     string `json:"channel"`     // e.g. "verification", "accretion", "duplication"
	Events      int    `json:"events"`      // event count in window (always 0 for dead signals)
	Completions int    `json:"completions"` // completions in window
	Description string `json:"description"` // what event types feed this channel
}

// GateAuditEntry contains per-gate metrics.
type GateAuditEntry struct {
	Gate        string  `json:"gate"`
	Invocations int     `json:"invocations"`
	Blocks      int     `json:"blocks"`
	Bypasses    int     `json:"bypasses"`
	Allows      int     `json:"allows"`
	FireRatePct float64 `json:"fire_rate_pct"` // (blocks+bypasses)/invocations * 100
	CoveragePct float64 `json:"coverage_pct"`  // invocations/totalSpawns * 100
	MeanCostMs  int     `json:"mean_cost_ms"`  // from pipeline_timing in agent.completed
}

// GateAnomaly flags a gate with unusual behavior.
type GateAnomaly struct {
	Gate    string `json:"gate"`
	Type    string `json:"type"`    // zero_fires, high_cost, low_coverage
	Message string `json:"message"`
}

func runHarnessAudit() error {
	eventsPath := getEventsPath()
	events, err := parseEvents(eventsPath, eventsSince(harnessAuditDays))
	if err != nil {
		// No events — show empty report
		result := buildGateAudit(nil, harnessAuditDays)
		return outputGateAudit(result)
	}
	result := buildGateAudit(events, harnessAuditDays)
	return outputGateAudit(result)
}

func outputGateAudit(result *GateAuditResult) error {
	if harnessAuditJSON {
		output, err := formatGateAuditJSON(result)
		if err != nil {
			return err
		}
		fmt.Print(output)
		return nil
	}
	fmt.Print(formatGateAuditText(result))
	return nil
}

// removedGates lists gates that were removed from the spawn pipeline.
// Historical events for these gates are ignored in the audit to avoid
// false zero-fire anomaly warnings.
// See pkg/orch/spawn_preflight.go line 9.
var removedGates = map[string]bool{
	"verification": true,
	"drain":        true,
	"concurrency":  true,
	"ratelimit":    true,
}

func buildGateAudit(events []StatsEvent, days int) *GateAuditResult {
	cutoff := time.Now().Unix() - int64(days)*86400

	// Per-gate counters
	type gateCounts struct {
		blocks   int
		bypasses int
		allows   int
	}
	gateMap := make(map[string]*gateCounts)

	// Pipeline timing: gate_name → []duration_ms
	costSamples := make(map[string][]int)

	// Count spawns and completions in window
	totalSpawns := 0
	totalCompletions := 0

	for _, e := range events {
		if e.Timestamp < cutoff {
			continue
		}

		switch e.Type {
		case "session.spawned":
			totalSpawns++

		case "agent.completed":
			totalCompletions++
			if e.Data != nil {
				timings, ok := e.Data["pipeline_timing"].([]interface{})
				if ok {
					for _, t := range timings {
						tm, ok := t.(map[string]interface{})
						if !ok {
							continue
						}
						name, _ := tm["name"].(string)
						durationMs, _ := tm["duration_ms"].(float64)
						skipped, _ := tm["skipped"].(bool)
						if name != "" && !skipped && durationMs > 0 {
							costSamples[name] = append(costSamples[name], int(durationMs))
						}
					}
				}
			}

		case "spawn.gate_decision":
			if e.Data == nil {
				continue
			}
			gateName, _ := e.Data["gate_name"].(string)
			decision, _ := e.Data["decision"].(string)
			if gateName == "" || decision == "" || removedGates[gateName] {
				continue
			}
			gc, ok := gateMap[gateName]
			if !ok {
				gc = &gateCounts{}
				gateMap[gateName] = gc
			}
			switch decision {
			case "block":
				gc.blocks++
			case "bypass":
				gc.bypasses++
			case "allow", "advisory", "warn":
				gc.allows++
			}
		}
	}

	// Build entries sorted by invocation count descending
	var entries []GateAuditEntry
	for gate, gc := range gateMap {
		invocations := gc.blocks + gc.bypasses + gc.allows
		fireRate := 0.0
		if invocations > 0 {
			fireRate = float64(gc.blocks+gc.bypasses) / float64(invocations) * 100
		}
		coverage := 0.0
		if totalSpawns > 0 {
			coverage = float64(invocations) / float64(totalSpawns) * 100
		}
		meanCost := 0
		if samples, ok := costSamples[gate]; ok && len(samples) > 0 {
			sum := 0
			for _, s := range samples {
				sum += s
			}
			meanCost = sum / len(samples)
		}
		entries = append(entries, GateAuditEntry{
			Gate:        gate,
			Invocations: invocations,
			Blocks:      gc.blocks,
			Bypasses:    gc.bypasses,
			Allows:      gc.allows,
			FireRatePct: fireRate,
			CoveragePct: coverage,
			MeanCostMs:  meanCost,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Invocations > entries[j].Invocations
	})

	// Detect anomalies
	var anomalies []GateAnomaly
	for _, e := range entries {
		fires := e.Blocks + e.Bypasses
		if fires == 0 {
			anomalies = append(anomalies, GateAnomaly{
				Gate:    e.Gate,
				Type:    "zero_fires",
				Message: fmt.Sprintf("0 fires in %dd — gate may be inert", days),
			})
		}
		if e.MeanCostMs > 1000 {
			anomalies = append(anomalies, GateAnomaly{
				Gate:    e.Gate,
				Type:    "high_cost",
				Message: fmt.Sprintf("mean cost %dms > 1000ms threshold", e.MeanCostMs),
			})
		}
		if totalSpawns > 0 && e.CoveragePct < 50 {
			anomalies = append(anomalies, GateAnomaly{
				Gate:    e.Gate,
				Type:    "low_coverage",
				Message: fmt.Sprintf("%.0f%% coverage < 50%% threshold (%d/%d spawns)", e.CoveragePct, e.Invocations, totalSpawns),
			})
		}
	}

	// Dead signal detection: flag feedback channels with zero events over N completions.
	deadSignals, deadAnomalies := detectDeadSignals(events, cutoff, totalCompletions, days)
	anomalies = append(anomalies, deadAnomalies...)

	return &GateAuditResult{
		GeneratedAt:  time.Now().Format(time.RFC3339),
		DaysAnalyzed: days,
		TotalSpawns:  totalSpawns,
		Gates:        entries,
		DeadSignals:  deadSignals,
		Anomalies:    anomalies,
	}
}

// deadSignalMinCompletions is the minimum number of completions required
// before flagging a channel as dead. Below this threshold, zero events
// could simply mean not enough data.
const deadSignalMinCompletions = 10

// feedbackChannel defines a feedback channel and the event types that feed it.
type feedbackChannel struct {
	name        string
	eventTypes  []string
	description string
}

// knownFeedbackChannels lists the channels we expect to produce events
// during normal operation. If any of these have zero events over N completions,
// they're likely dead (broken/disconnected), not clean (nothing to report).
var knownFeedbackChannels = []feedbackChannel{
	{
		name:        "verification",
		eventTypes:  []string{"verification.failed", "verification.bypassed", "verification.auto_skipped"},
		description: "verification.failed + verification.bypassed + verification.auto_skipped",
	},
	{
		name:        "accretion",
		eventTypes:  []string{"accretion.delta"},
		description: "accretion.delta (file growth/shrinkage per agent)",
	},
	{
		name:        "duplication",
		eventTypes:  []string{"duplication.detected", "duplication.suppressed"},
		description: "duplication.detected + duplication.suppressed",
	},
	{
		name:        "spawn_gates",
		eventTypes:  []string{"spawn.gate_decision"},
		description: "spawn.gate_decision (gate evaluations during spawn)",
	},
	{
		name:        "trigger_outcomes",
		eventTypes:  []string{"trigger.outcome"},
		description: "trigger.outcome (daemon detector false positive tracking)",
	},
}

// detectDeadSignals checks known feedback channels for zero events over N completions.
func detectDeadSignals(events []StatsEvent, cutoff int64, completions int, days int) ([]DeadSignalEntry, []GateAnomaly) {
	if completions < deadSignalMinCompletions {
		return nil, nil
	}

	// Count events per channel
	channelCounts := make(map[string]int)
	for _, ch := range knownFeedbackChannels {
		channelCounts[ch.name] = 0
	}

	// Build event-type-to-channel lookup
	eventToChannel := make(map[string]string)
	for _, ch := range knownFeedbackChannels {
		for _, et := range ch.eventTypes {
			eventToChannel[et] = ch.name
		}
	}

	for _, e := range events {
		if e.Timestamp < cutoff {
			continue
		}
		if ch, ok := eventToChannel[e.Type]; ok {
			channelCounts[ch]++
		}
	}

	var deadSignals []DeadSignalEntry
	var anomalies []GateAnomaly

	for _, ch := range knownFeedbackChannels {
		count := channelCounts[ch.name]
		if count == 0 {
			deadSignals = append(deadSignals, DeadSignalEntry{
				Channel:     ch.name,
				Events:      0,
				Completions: completions,
				Description: ch.description,
			})
			anomalies = append(anomalies, GateAnomaly{
				Gate:    ch.name,
				Type:    "dead_signal",
				Message: fmt.Sprintf("0 events over %d completions in %dd — channel may be dead, not clean", completions, days),
			})
		}
	}

	return deadSignals, anomalies
}

func formatGateAuditText(result *GateAuditResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "═══ GATE AUDIT (%dd, %d spawns) ═══\n\n", result.DaysAnalyzed, result.TotalSpawns)

	if len(result.Gates) == 0 {
		fmt.Fprintln(&b, "  No gate decision events found.")
		fmt.Fprintln(&b)
	} else {
		// Table header
		fmt.Fprintf(&b, "  %-22s %5s  %5s  %5s  %5s  %7s  %7s  %7s\n",
			"GATE", "INV", "BLOCK", "BYPAS", "ALLOW", "FIRE%", "COV%", "COST")
		fmt.Fprintf(&b, "  %s\n", strings.Repeat("─", 78))

		for _, g := range result.Gates {
			costStr := "  —"
			if g.MeanCostMs > 0 {
				costStr = fmt.Sprintf("%4dms", g.MeanCostMs)
			}
			fmt.Fprintf(&b, "  %-22s %5d  %5d  %5d  %5d  %6.1f%%  %6.1f%%  %s\n",
				g.Gate, g.Invocations, g.Blocks, g.Bypasses, g.Allows,
				g.FireRatePct, g.CoveragePct, costStr)
		}
		fmt.Fprintln(&b)
	}

	// Dead signals
	if len(result.DeadSignals) > 0 {
		fmt.Fprintf(&b, "DEAD SIGNALS (%d completions, 0 events)\n", result.DeadSignals[0].Completions)
		fmt.Fprintf(&b, "  %-22s %6s  %s\n", "CHANNEL", "EVENTS", "DESCRIPTION")
		fmt.Fprintf(&b, "  %s\n", strings.Repeat("─", 60))
		for _, ds := range result.DeadSignals {
			fmt.Fprintf(&b, "  %-22s %6d  %s\n", ds.Channel, ds.Events, ds.Description)
		}
		fmt.Fprintln(&b)
	}

	// Anomalies
	if len(result.Anomalies) > 0 {
		fmt.Fprintln(&b, "ANOMALIES")
		for _, a := range result.Anomalies {
			fmt.Fprintf(&b, "  ⚠ %-22s [%s] %s\n", a.Gate, a.Type, a.Message)
		}
		fmt.Fprintln(&b)
	}

	return b.String()
}

func formatGateAuditJSON(result *GateAuditResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling gate audit: %w", err)
	}
	return string(data) + "\n", nil
}
