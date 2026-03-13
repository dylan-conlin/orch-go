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
	GeneratedAt  string           `json:"generated_at"`
	DaysAnalyzed int              `json:"days_analyzed"`
	TotalSpawns  int              `json:"total_spawns"`
	Gates        []GateAuditEntry `json:"gates"`
	Anomalies    []GateAnomaly    `json:"anomalies"`
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
	events, err := parseEvents(eventsPath)
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

	// Count spawns in window
	totalSpawns := 0

	for _, e := range events {
		if e.Timestamp < cutoff {
			continue
		}

		switch e.Type {
		case "session.spawned":
			totalSpawns++

		case "spawn.gate_decision":
			if e.Data == nil {
				continue
			}
			gateName, _ := e.Data["gate_name"].(string)
			decision, _ := e.Data["decision"].(string)
			if gateName == "" || decision == "" {
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
			case "allow", "advisory":
				gc.allows++
			}

		case "agent.completed":
			if e.Data == nil {
				continue
			}
			timings, ok := e.Data["pipeline_timing"].([]interface{})
			if !ok {
				continue
			}
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

	return &GateAuditResult{
		GeneratedAt:  time.Now().Format(time.RFC3339),
		DaysAnalyzed: days,
		TotalSpawns:  totalSpawns,
		Gates:        entries,
		Anomalies:    anomalies,
	}
}

func formatGateAuditText(result *GateAuditResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "═══ GATE AUDIT (%dd, %d spawns) ═══\n\n", result.DaysAnalyzed, result.TotalSpawns)

	if len(result.Gates) == 0 {
		fmt.Fprintln(&b, "  No gate decision events found.")
		fmt.Fprintln(&b)
		return b.String()
	}

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
