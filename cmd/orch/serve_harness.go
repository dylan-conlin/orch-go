package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"
)

// HarnessResponse is the top-level response for GET /api/harness.
type HarnessResponse struct {
	GeneratedAt           string                          `json:"generated_at"`
	AnalysisPeriod        string                          `json:"analysis_period"`
	TotalSpawns           int                             `json:"total_spawns"`
	Pipeline              []PipelineStage                 `json:"pipeline"`
	AccretionVelocity     *AccretionVelocity              `json:"accretion_velocity,omitempty"`
	CompletionCoverage    CompletionCoverage              `json:"completion_coverage"`
	FalsificationVerdicts map[string]FalsificationVerdict `json:"falsification_verdicts"`
	MeasurementCoverage   MeasurementCoverage             `json:"measurement_coverage"`
	ExplorationMetrics    *ExplorationMetrics             `json:"exploration_metrics,omitempty"`
}

// ExplorationMetrics aggregates exploration mode run data.
type ExplorationMetrics struct {
	TotalRuns        int     `json:"total_runs"`
	CompletedRuns    int     `json:"completed_runs"`
	TotalFindings    int     `json:"total_findings"`
	TotalAccepted    int     `json:"total_accepted"`
	TotalContested   int     `json:"total_contested"`
	TotalRejected    int     `json:"total_rejected"`
	TotalGaps        int     `json:"total_gaps"`
	AvgWorkersPerRun float64 `json:"avg_workers_per_run"`
	TotalIterations  int     `json:"total_iterations"`  // Total re-exploration rounds across all runs
	IteratedRuns     int     `json:"iterated_runs"`     // Number of runs that used iteration (depth > 1)
}

// PipelineStage represents one stage in the harness pipeline.
type PipelineStage struct {
	Stage      string              `json:"stage"`
	Components []PipelineComponent `json:"components"`
}

// PipelineComponent represents a single harness component within a stage.
type PipelineComponent struct {
	Name              string   `json:"name"`
	Type              string   `json:"type"`               // "hard", "soft", "human"
	MeasurementStatus string   `json:"measurement_status"` // "flowing", "proxy_only", "unmeasured", "collecting"
	FireRate          *float64 `json:"fire_rate,omitempty"`
	BlockRate         *float64 `json:"block_rate,omitempty"`
	BypassRate        *float64 `json:"bypass_rate,omitempty"`
	FailRate          *float64 `json:"fail_rate,omitempty"`
	PassRate          *float64 `json:"pass_rate,omitempty"`
	Bypassed          int      `json:"bypassed,omitempty"`
	Blocked           int      `json:"blocked,omitempty"`
	LastFired         string   `json:"last_fired,omitempty"`
	ProxyMetric       string   `json:"proxy_metric,omitempty"`
	CollectingSince   string   `json:"collecting_since,omitempty"`
}

// AccretionVelocity tracks code growth rate over time.
type AccretionVelocity struct {
	CurrentWeeklyLines  int     `json:"current_weekly_lines"`
	BaselineWeeklyLines int     `json:"baseline_weekly_lines"`
	VelocityChangePct   float64 `json:"velocity_change_pct"`
	Trend               string  `json:"trend"` // "declining", "stable", "increasing"
}

// CompletionCoverage tracks field coverage in completion events.
type CompletionCoverage struct {
	TotalCompletions int     `json:"total_completions"`
	WithSkill        int     `json:"with_skill"`
	WithOutcome      int     `json:"with_outcome"`
	WithDuration     int     `json:"with_duration"`
	CoveragePct      float64 `json:"coverage_pct"`
}

// FalsificationVerdict represents the status of one falsification criterion.
type FalsificationVerdict struct {
	Criterion string `json:"criterion"`
	Status    string `json:"status"` // "falsified", "confirmed", "insufficient_data", "not_measurable"
	Evidence  string `json:"evidence"`
	Threshold string `json:"threshold"`
}

// MeasurementCoverage summarizes how many components are measured.
type MeasurementCoverage struct {
	TotalComponents int `json:"total_components"`
	WithMeasurement int `json:"with_measurement"`
	ProxyOnly       int `json:"proxy_only"`
	Unmeasured      int `json:"unmeasured"`
}

// accretionSnapshot is an internal type for tracking code size over time.
type accretionSnapshot struct {
	Timestamp  int64
	TotalLines int
	Directory  string
}

// handleHarnessReport serves GET /api/harness with harness pipeline data.
func handleHarnessReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse days parameter (default 7)
	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	// Parse events
	eventsPath := getEventsPath()
	events, err := parseEvents(eventsPath, eventsSince(days))
	if err != nil {
		resp := buildEmptyHarnessResponse(days)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := buildHarnessResponse(events, days)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode harness report: %v", err), http.StatusInternalServerError)
		return
	}
}

func buildEmptyHarnessResponse(days int) *HarnessResponse {
	return &HarnessResponse{
		GeneratedAt:           time.Now().Format(time.RFC3339),
		AnalysisPeriod:        fmt.Sprintf("Last %d days", days),
		Pipeline:              buildDefaultPipeline(0, nil),
		FalsificationVerdicts: buildDefaultVerdicts(),
		MeasurementCoverage: MeasurementCoverage{
			TotalComponents: 12,
			WithMeasurement: 0,
			ProxyOnly:       3,
			Unmeasured:      9,
		},
	}
}

func buildHarnessResponse(events []StatsEvent, days int) *HarnessResponse {
	now := time.Now().Unix()
	cutoff := now - int64(days*86400)

	var totalSpawns int
	triageBypassed := 0
	hotspotBypassed := 0

	gateDecisions := make(map[string]map[string]int)
	lastGateFired := make(map[string]int64)

	var totalCompletions int
	var withSkill, withOutcome, withDuration int

	// Exploration metrics
	explorationDecomposed := make(map[string]float64) // beads_id -> breadth
	explorationJudged := make(map[string]struct{})
	var explorationTotalFindings, explorationAccepted, explorationContested, explorationRejected, explorationGaps int
	explorationSynthesized := make(map[string]struct{})
	var explorationTotalIterations int
	explorationIteratedRuns := make(map[string]struct{}) // beads_ids that used iteration

	// Accretion snapshots collected from ALL events (not just in-window)
	var snapshots []accretionSnapshot
	var firstAccretionEvent int64

	for _, event := range events {
		// Accretion snapshots need full history for baseline comparison
		if event.Type == "accretion.snapshot" && event.Data != nil {
			if firstAccretionEvent == 0 || event.Timestamp < firstAccretionEvent {
				firstAccretionEvent = event.Timestamp
			}
			totalLines := 0
			// Support both formats:
			// 1. Flat: {"total_lines": N, "directory": "pkg/foo/"}
			// 2. Directories array: {"directories": [{"total_lines": N, ...}, ...]}
			if tl, ok := event.Data["total_lines"].(float64); ok {
				totalLines = int(tl)
			} else if dirs, ok := event.Data["directories"].([]interface{}); ok {
				for _, d := range dirs {
					if dm, ok := d.(map[string]interface{}); ok {
						if tl, ok := dm["total_lines"].(float64); ok {
							totalLines += int(tl)
						}
					}
				}
			}
			if totalLines > 0 {
				snapshots = append(snapshots, accretionSnapshot{
					Timestamp:  event.Timestamp,
					TotalLines: totalLines,
				})
			}
			continue
		}

		// Other events filtered by time window
		if event.Timestamp < cutoff {
			continue
		}

		switch event.Type {
		case "session.spawned":
			totalSpawns++

		case "spawn.triage_bypassed":
			triageBypassed++

		case "spawn.hotspot_bypassed":
			hotspotBypassed++

		case "spawn.gate_decision":
			if event.Data != nil {
				gate, _ := event.Data["gate_name"].(string)
				decision, _ := event.Data["decision"].(string)
				if gate != "" && decision != "" && !removedGates[gate] {
					if gateDecisions[gate] == nil {
						gateDecisions[gate] = make(map[string]int)
					}
					gateDecisions[gate][decision]++
					if decision == "block" || decision == "bypass" {
						lastGateFired[gate] = event.Timestamp
					}
				}
			}

		case "agent.completed":
			totalCompletions++
			if event.Data != nil {
				if _, ok := event.Data["skill"]; ok {
					withSkill++
				}
				if _, ok := event.Data["outcome"]; ok {
					withOutcome++
				}
				if _, ok := event.Data["duration_seconds"]; ok {
					withDuration++
				} else if _, ok := event.Data["duration_minutes"]; ok {
					withDuration++
				}
			}

		case "exploration.decomposed":
			if event.Data != nil {
				bid, _ := event.Data["beads_id"].(string)
				breadth, _ := event.Data["breadth"].(float64)
				if bid != "" {
					explorationDecomposed[bid] = breadth
				}
			}

		case "exploration.judged":
			if event.Data != nil {
				bid, _ := event.Data["beads_id"].(string)
				if bid != "" {
					explorationJudged[bid] = struct{}{}
				}
				if tf, ok := event.Data["total_findings"].(float64); ok {
					explorationTotalFindings += int(tf)
				}
				if a, ok := event.Data["accepted"].(float64); ok {
					explorationAccepted += int(a)
				}
				if c, ok := event.Data["contested"].(float64); ok {
					explorationContested += int(c)
				}
				if r, ok := event.Data["rejected"].(float64); ok {
					explorationRejected += int(r)
				}
				if g, ok := event.Data["coverage_gaps"].(float64); ok {
					explorationGaps += int(g)
				}
			}

		case "exploration.synthesized":
			if event.Data != nil {
				bid, _ := event.Data["beads_id"].(string)
				if bid != "" {
					explorationSynthesized[bid] = struct{}{}
				}
			}

		case "exploration.iterated":
			if event.Data != nil {
				bid, _ := event.Data["beads_id"].(string)
				if bid != "" {
					explorationIteratedRuns[bid] = struct{}{}
				}
				explorationTotalIterations++
			}
		}
	}

	pipeline := buildPipeline(totalSpawns, triageBypassed, hotspotBypassed, gateDecisions, lastGateFired, totalCompletions, firstAccretionEvent)

	// Compute accretion velocity from snapshots
	var accVelocity *AccretionVelocity
	if len(snapshots) >= 2 {
		accVelocity = computeAccretionVelocity(snapshots)
	}

	// Completion coverage
	coveragePct := 0.0
	if totalCompletions > 0 {
		minField := withSkill
		if withOutcome < minField {
			minField = withOutcome
		}
		if withDuration < minField {
			minField = withDuration
		}
		coveragePct = float64(minField) / float64(totalCompletions) * 100
	}

	verdicts := buildVerdicts(totalSpawns, triageBypassed, hotspotBypassed, gateDecisions)

	measured := 0
	collecting := 0
	for _, stage := range pipeline {
		for _, comp := range stage.Components {
			if comp.MeasurementStatus == "flowing" {
				measured++
			} else if comp.MeasurementStatus == "collecting" {
				collecting++
			}
		}
	}

	// Build exploration metrics if any exploration events exist
	var explMetrics *ExplorationMetrics
	if len(explorationDecomposed) > 0 {
		totalBreadth := 0.0
		for _, b := range explorationDecomposed {
			totalBreadth += b
		}
		explMetrics = &ExplorationMetrics{
			TotalRuns:        len(explorationDecomposed),
			CompletedRuns:    len(explorationSynthesized),
			TotalFindings:    explorationTotalFindings,
			TotalAccepted:    explorationAccepted,
			TotalContested:   explorationContested,
			TotalRejected:    explorationRejected,
			TotalGaps:        explorationGaps,
			AvgWorkersPerRun: totalBreadth / float64(len(explorationDecomposed)),
			TotalIterations:  explorationTotalIterations,
			IteratedRuns:     len(explorationIteratedRuns),
		}
	}

	return &HarnessResponse{
		GeneratedAt:       time.Now().Format(time.RFC3339),
		AnalysisPeriod:    fmt.Sprintf("Last %d days", days),
		TotalSpawns:       totalSpawns,
		Pipeline:          pipeline,
		AccretionVelocity: accVelocity,
		CompletionCoverage: CompletionCoverage{
			TotalCompletions: totalCompletions,
			WithSkill:        withSkill,
			WithOutcome:      withOutcome,
			WithDuration:     withDuration,
			CoveragePct:      coveragePct,
		},
		FalsificationVerdicts: verdicts,
		MeasurementCoverage: MeasurementCoverage{
			TotalComponents: 12,
			WithMeasurement: measured + collecting,
			ProxyOnly:       3,
			Unmeasured:      12 - measured - collecting - 3,
		},
		ExplorationMetrics: explMetrics,
	}
}

// computeAccretionVelocity calculates weekly velocity from snapshots.
func computeAccretionVelocity(snapshots []accretionSnapshot) *AccretionVelocity {
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp < snapshots[j].Timestamp
	})

	n := len(snapshots)
	if n < 2 {
		return nil
	}

	// Baseline: first two snapshots
	baselineDelta := snapshots[1].TotalLines - snapshots[0].TotalLines
	baselineDays := float64(snapshots[1].Timestamp-snapshots[0].Timestamp) / 86400.0
	baselineWeekly := 0
	if baselineDays > 0 {
		baselineWeekly = int(float64(baselineDelta) / baselineDays * 7)
	}

	// Current: last two snapshots
	currentDelta := snapshots[n-1].TotalLines - snapshots[n-2].TotalLines
	currentDays := float64(snapshots[n-1].Timestamp-snapshots[n-2].Timestamp) / 86400.0
	currentWeekly := 0
	if currentDays > 0 {
		currentWeekly = int(float64(currentDelta) / currentDays * 7)
	}

	changePct := 0.0
	if baselineWeekly > 0 {
		changePct = (float64(currentWeekly) - float64(baselineWeekly)) / float64(baselineWeekly) * 100
	}

	trend := "stable"
	if changePct < -20 {
		trend = "declining"
	} else if changePct > 20 {
		trend = "increasing"
	}

	return &AccretionVelocity{
		CurrentWeeklyLines:  currentWeekly,
		BaselineWeeklyLines: baselineWeekly,
		VelocityChangePct:   changePct,
		Trend:               trend,
	}
}

func buildDefaultPipeline(totalSpawns int, lastGateFired map[string]int64) []PipelineStage {
	return buildPipeline(totalSpawns, 0, 0, nil, lastGateFired, 0, 0)
}

func buildPipeline(totalSpawns, triageBypassed, hotspotBypassed int, gateDecisions map[string]map[string]int, lastGateFired map[string]int64, totalCompletions int, firstAccretionEvent int64) []PipelineStage {
	ptrFloat := func(f float64) *float64 { return &f }
	safeRate := func(count, total int) *float64 {
		if total == 0 {
			return nil
		}
		return ptrFloat(float64(count) / float64(total))
	}
	formatTimestamp := func(ts int64) string {
		if ts == 0 {
			return ""
		}
		return time.Unix(ts, 0).Format(time.RFC3339)
	}

	triageTotal := triageBypassed
	hotspotTotal := hotspotBypassed
	hotspotBlocked := 0

	if gateDecisions != nil {
		if d, ok := gateDecisions["hotspot"]; ok {
			hotspotBlocked += d["block"]
			hotspotTotal += d["bypass"]
		}
		if d, ok := gateDecisions["triage"]; ok {
			triageTotal += d["bypass"]
		}
	}

	return []PipelineStage{
		{
			Stage: "spawn",
			Components: []PipelineComponent{
				{
					Name:              "triage_gate",
					Type:              "hard",
					MeasurementStatus: statusFromCount(triageTotal),
					FireRate:          safeRate(triageTotal, totalSpawns),
					BlockRate:         ptrFloat(0),
					BypassRate:        safeRate(triageTotal, totalSpawns),
					Bypassed:          triageTotal,
				},
				{
					Name:              "hotspot_gate",
					Type:              "hard",
					MeasurementStatus: statusFromCount(hotspotTotal + hotspotBlocked),
					FireRate:          safeRate(hotspotTotal+hotspotBlocked, totalSpawns),
					BlockRate:         safeRate(hotspotBlocked, totalSpawns),
					BypassRate:        safeRate(hotspotTotal, totalSpawns),
					Bypassed:          hotspotTotal,
					Blocked:           hotspotBlocked,
					LastFired:         formatTimestamp(lastGateFired["hotspot"]),
				},
			},
		},
		{
			Stage: "authoring",
			Components: []PipelineComponent{
				{
					Name:              "claude_md",
					Type:              "soft",
					MeasurementStatus: "proxy_only",
					ProxyMetric:       "convention violation rate via accretion gate",
				},
				{
					Name:              "spawn_context",
					Type:              "soft",
					MeasurementStatus: "proxy_only",
					ProxyMetric:       "hotspot advisory compliance",
				},
				{
					Name:              "kb_knowledge",
					Type:              "soft",
					MeasurementStatus: "proxy_only",
					ProxyMetric:       "re-investigation rate",
				},
			},
		},
		{
			Stage: "pre_commit",
			Components: []PipelineComponent{
				accretionGateComponent(firstAccretionEvent, formatTimestamp),

				{
					Name:              "build_gate",
					Type:              "hard",
					MeasurementStatus: "flowing",
				},
			},
		},
		{
			Stage: "completion",
			Components: []PipelineComponent{
				{
					Name:              "verification_pipeline",
					Type:              "hard",
					MeasurementStatus: statusFromCount(totalCompletions),
				},
				{
					Name:              "explain_back",
					Type:              "human",
					MeasurementStatus: statusFromCount(totalCompletions),
				},
			},
		},
	}
}

func accretionGateComponent(firstAccretionEvent int64, formatTimestamp func(int64) string) PipelineComponent {
	comp := PipelineComponent{
		Name: "accretion_gate",
		Type: "hard",
	}
	if firstAccretionEvent > 0 {
		comp.MeasurementStatus = "collecting"
		comp.CollectingSince = time.Unix(firstAccretionEvent, 0).Format("Jan 2")
	} else {
		comp.MeasurementStatus = "unmeasured"
	}
	return comp
}

func statusFromCount(count int) string {
	if count > 0 {
		return "flowing"
	}
	return "unmeasured"
}

func buildDefaultVerdicts() map[string]FalsificationVerdict {
	return buildVerdicts(0, 0, 0, nil)
}

func buildVerdicts(totalSpawns, triageBypassed, hotspotBypassed int, gateDecisions map[string]map[string]int) map[string]FalsificationVerdict {
	totalFireCount := triageBypassed + hotspotBypassed
	if gateDecisions != nil {
		for _, decisions := range gateDecisions {
			for _, count := range decisions {
				totalFireCount += count
			}
		}
	}

	gatesIrrelevantStatus := "insufficient_data"
	gatesIrrelevantEvidence := "No spawn data available."
	if totalSpawns > 0 {
		fireRate := float64(totalFireCount) / float64(totalSpawns)
		if fireRate > 0.05 {
			gatesIrrelevantStatus = "falsified"
			gatesIrrelevantEvidence = fmt.Sprintf("%.1f%% combined fire rate (%d events across %d spawns). Gates fire on a significant portion of spawns.", fireRate*100, totalFireCount, totalSpawns)
		} else {
			gatesIrrelevantStatus = "confirmed"
			gatesIrrelevantEvidence = fmt.Sprintf("%.1f%% combined fire rate — gates rarely fire.", fireRate*100)
		}
	}

	return map[string]FalsificationVerdict{
		"gates_are_ceremony": {
			Criterion: "Gates ship, accretion doesn't slow",
			Status:    "insufficient_data",
			Evidence:  "Need 2+ weeks of post-gate accretion velocity data. Checkpoint: Mar 24.",
			Threshold: "Post-gate velocity must be <50% of pre-gate for 2+ consecutive weeks",
		},
		"gates_are_irrelevant": {
			Criterion: "Gate deflection rate near-zero",
			Status:    gatesIrrelevantStatus,
			Evidence:  gatesIrrelevantEvidence,
			Threshold: "Fire rate <5% would indicate irrelevance",
		},
		"soft_harness_is_inert": {
			Criterion: "Soft harness removal causes no behavior change",
			Status:    "not_measurable",
			Evidence:  "No controlled experiments. Proxy metrics available but not definitive.",
			Threshold: "Requires A/B test: spawn with vs without soft harness component",
		},
		"framework_is_anecdotal": {
			Criterion: "Second system, no benefit",
			Status:    "not_measurable",
			Evidence:  "No second system instrumented.",
			Threshold: "Deploy harness tooling to second project, compare outcomes",
		},
	}
}
