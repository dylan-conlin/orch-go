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
	gateEffDays int
	gateEffJSON bool
)

var harnessGateEffectivenessCmd = &cobra.Command{
	Use:   "gate-effectiveness",
	Short: "Analyze whether gate enforcement improves agent quality",
	Long: `Correlates gate decisions with agent outcomes to answer:
"Does structural enforcement improve agent quality?"

For each gate, compares three cohorts:
  - Blocked: agents stopped by the gate (redirected/escalated)
  - Bypassed: agents that skipped the gate (--force, --bypass-triage, etc.)
  - Allowed: agents that passed through the gate cleanly

Quality signals per cohort:
  - Completion rate, verification pass rate
  - Abandonment rate
  - Mean duration (minutes)
  - Accretion impact (net line delta, risk files touched)

Also produces an overall verdict comparing "enforced" (blocked+allowed)
vs "bypassed" cohorts across all gates.

Examples:
  orch harness gate-effectiveness              # Last 30 days
  orch harness gate-effectiveness --days 14    # Last 14 days
  orch harness gate-effectiveness --json       # Machine-readable`,
	RunE: func(cmd *cobra.Command, args []string) error {
		emitCommandInvoked("harness gate-effectiveness", flagsFromCmd(cmd)...)
		return runGateEffectiveness()
	},
}

func init() {
	harnessGateEffectivenessCmd.Flags().IntVar(&gateEffDays, "days", 30, "Number of days to analyze")
	harnessGateEffectivenessCmd.Flags().BoolVar(&gateEffJSON, "json", false, "Output as JSON")
	harnessCmd.AddCommand(harnessGateEffectivenessCmd)
}

// GateEffResult is the top-level result for gate-effectiveness analysis.
type GateEffResult struct {
	GeneratedAt    string               `json:"generated_at"`
	DaysAnalyzed   int                  `json:"days_analyzed"`
	TotalSpawns    int                  `json:"total_spawns"`
	TotalWithGates int                  `json:"total_with_gates"` // spawns that hit any gate
	PerGate        []GateEffPerGate     `json:"per_gate"`
	Overall        GateEffOverall       `json:"overall"`
	BySkill        []GateEffSkillEntry  `json:"by_skill,omitempty"`
	Verdict        string               `json:"verdict"`
	Caveats        []string             `json:"caveats,omitempty"`
}

// GateEffPerGate shows cohort comparison for a single gate.
type GateEffPerGate struct {
	Gate     string          `json:"gate"`
	Total    int             `json:"total_evaluations"`
	Blocked  GateEffCohort   `json:"blocked"`
	Bypassed GateEffCohort   `json:"bypassed"`
	Allowed  GateEffCohort   `json:"allowed"`
}

// GateEffCohort holds quality metrics for one cohort of agents.
type GateEffCohort struct {
	Count              int     `json:"count"`
	Completions        int     `json:"completions"`
	Abandonments       int     `json:"abandonments"`
	CompletionRate     float64 `json:"completion_rate"`
	VerificationPassed int     `json:"verification_passed"`
	VerificationRate   float64 `json:"verification_rate"`
	AvgDurationMin     float64 `json:"avg_duration_min,omitempty"`
	NetAccretion       int     `json:"net_accretion"`       // total net line delta
	RiskFilesTouched   int     `json:"risk_files_touched"`  // files >800 lines modified
	AvgNetDelta        float64 `json:"avg_net_delta"`       // per-agent average
}

// GateEffOverall compares enforced (blocked+allowed) vs bypassed across all gates.
type GateEffOverall struct {
	Enforced GateEffCohort `json:"enforced"` // agents that went through gates normally
	Bypassed GateEffCohort `json:"bypassed"` // agents that bypassed at least one gate
}

// GateEffSkillEntry shows per-skill gate interaction.
type GateEffSkillEntry struct {
	Skill          string  `json:"skill"`
	GateEvents     int     `json:"gate_events"`
	Blocks         int     `json:"blocks"`
	Bypasses       int     `json:"bypasses"`
	CompletionRate float64 `json:"completion_rate"`
}

func runGateEffectiveness() error {
	eventsPath := getEventsPath()
	events, err := parseEvents(eventsPath, eventsSince(gateEffDays))
	if err != nil {
		events = nil
	}
	result := buildGateEffectiveness(events, gateEffDays)

	if gateEffJSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling result: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Print(formatGateEffText(result))
	return nil
}

// agentRecord holds correlated data for a single agent (by beads_id).
type agentRecord struct {
	beadsID            string
	skill              string
	spawnTime          int64
	completed          bool
	abandoned          bool
	verificationPassed bool
	completionTime     int64
	netAccretion       int
	riskFiles          int
	// per-gate decisions: gate_name -> decision
	gateDecisions map[string]string
}

func buildGateEffectiveness(events []StatsEvent, days int) *GateEffResult {
	cutoff := time.Now().Unix() - int64(days)*86400

	// Phase 1: Build per-agent records by correlating events
	agents := make(map[string]*agentRecord) // beads_id -> record
	sessionToBeads := make(map[string]string)
	workspaceToBeads := make(map[string]string)
	totalSpawns := 0

	// First pass: establish session-to-beads mapping and spawn data
	for _, e := range events {
		if e.Timestamp < cutoff || e.Data == nil {
			continue
		}
		switch e.Type {
		case "session.spawned":
			totalSpawns++
			beadsID, _ := e.Data["beads_id"].(string)
			skill, _ := e.Data["skill"].(string)
			workspace, _ := e.Data["workspace"].(string)
			if beadsID != "" {
				sessionToBeads[e.SessionID] = beadsID
				if workspace != "" {
					workspaceToBeads[workspace] = beadsID
				}
				if _, ok := agents[beadsID]; !ok {
					agents[beadsID] = &agentRecord{
						beadsID:       beadsID,
						skill:         skill,
						spawnTime:     e.Timestamp,
						gateDecisions: make(map[string]string),
					}
				}
			}
		}
	}

	// Second pass: enrich with gate decisions, completions, accretion
	for _, e := range events {
		if e.Timestamp < cutoff || e.Data == nil {
			continue
		}
		switch e.Type {
		case "spawn.gate_decision":
			beadsID, _ := e.Data["beads_id"].(string)
			gateName, _ := e.Data["gate_name"].(string)
			decision, _ := e.Data["decision"].(string)
			if beadsID == "" || gateName == "" || decision == "" {
				continue
			}
			rec, ok := agents[beadsID]
			if !ok {
				// gate fired for unknown beads_id — create stub
				rec = &agentRecord{
					beadsID:       beadsID,
					gateDecisions: make(map[string]string),
				}
				agents[beadsID] = rec
			}
			// Keep the most restrictive decision per gate
			existing := rec.gateDecisions[gateName]
			if decisionSeverity(decision) > decisionSeverity(existing) {
				rec.gateDecisions[gateName] = decision
			}

		case "agent.completed":
			beadsID, _ := e.Data["beads_id"].(string)
			if beadsID == "" {
				workspace, _ := e.Data["workspace"].(string)
				beadsID = workspaceToBeads[workspace]
			}
			if beadsID == "" {
				continue
			}
			rec, ok := agents[beadsID]
			if !ok {
				continue
			}
			rec.completed = true
			rec.completionTime = e.Timestamp
			if vp, ok := e.Data["verification_passed"].(bool); ok {
				rec.verificationPassed = vp
			}

		case "agent.abandoned":
			beadsID, _ := e.Data["beads_id"].(string)
			if beadsID == "" {
				continue
			}
			if rec, ok := agents[beadsID]; ok {
				rec.abandoned = true
			}

		case "accretion.delta":
			beadsID, _ := e.Data["beads_id"].(string)
			if beadsID == "" {
				continue
			}
			rec, ok := agents[beadsID]
			if !ok {
				continue
			}
			if nd, ok := e.Data["net_delta"].(float64); ok {
				rec.netAccretion += int(nd)
			}
			if rf, ok := e.Data["risk_files"].(float64); ok {
				rec.riskFiles += int(rf)
			}
		}
	}

	// Phase 2: Build per-gate cohort analysis
	// Collect all gate names that appeared
	gateNames := make(map[string]bool)
	for _, rec := range agents {
		for g := range rec.gateDecisions {
			gateNames[g] = true
		}
	}

	var perGate []GateEffPerGate
	for gateName := range gateNames {
		pg := GateEffPerGate{Gate: gateName}
		var blockedAgents, bypassedAgents, allowedAgents []*agentRecord

		for _, rec := range agents {
			decision, hasGate := rec.gateDecisions[gateName]
			if !hasGate {
				continue
			}
			pg.Total++
			switch decision {
			case "block":
				blockedAgents = append(blockedAgents, rec)
			case "bypass":
				bypassedAgents = append(bypassedAgents, rec)
			case "allow", "advisory":
				allowedAgents = append(allowedAgents, rec)
			}
		}

		pg.Blocked = buildCohort(blockedAgents)
		pg.Bypassed = buildCohort(bypassedAgents)
		pg.Allowed = buildCohort(allowedAgents)
		perGate = append(perGate, pg)
	}

	sort.Slice(perGate, func(i, j int) bool {
		return perGate[i].Total > perGate[j].Total
	})

	// Phase 3: Overall cohort — enforced vs bypassed
	var enforcedAgents, bypassedAgents []*agentRecord
	bypassedSet := make(map[string]bool) // beads_ids that bypassed any gate
	for _, rec := range agents {
		if len(rec.gateDecisions) == 0 {
			continue // no gate interaction — skip for overall comparison
		}
		bypassed := false
		for _, decision := range rec.gateDecisions {
			if decision == "bypass" {
				bypassed = true
				break
			}
		}
		if bypassed {
			bypassedAgents = append(bypassedAgents, rec)
			bypassedSet[rec.beadsID] = true
		} else {
			enforcedAgents = append(enforcedAgents, rec)
		}
	}

	overall := GateEffOverall{
		Enforced: buildCohort(enforcedAgents),
		Bypassed: buildCohort(bypassedAgents),
	}

	// Phase 4: Per-skill breakdown
	skillData := make(map[string]*struct {
		gateEvents  int
		blocks      int
		bypasses    int
		completions int
		total       int
	})
	for _, rec := range agents {
		if rec.skill == "" || len(rec.gateDecisions) == 0 {
			continue
		}
		sd, ok := skillData[rec.skill]
		if !ok {
			sd = &struct {
				gateEvents  int
				blocks      int
				bypasses    int
				completions int
				total       int
			}{}
			skillData[rec.skill] = sd
		}
		sd.total++
		for _, decision := range rec.gateDecisions {
			sd.gateEvents++
			switch decision {
			case "block":
				sd.blocks++
			case "bypass":
				sd.bypasses++
			}
		}
		if rec.completed {
			sd.completions++
		}
	}

	var bySkill []GateEffSkillEntry
	for skill, sd := range skillData {
		completionRate := 0.0
		if sd.total > 0 {
			completionRate = float64(sd.completions) / float64(sd.total) * 100
		}
		bySkill = append(bySkill, GateEffSkillEntry{
			Skill:          skill,
			GateEvents:     sd.gateEvents,
			Blocks:         sd.blocks,
			Bypasses:       sd.bypasses,
			CompletionRate: completionRate,
		})
	}
	sort.Slice(bySkill, func(i, j int) bool {
		return bySkill[i].GateEvents > bySkill[j].GateEvents
	})

	// Phase 5: Generate verdict
	verdict, caveats := generateVerdict(overall, perGate, totalSpawns)

	totalWithGates := len(enforcedAgents) + len(bypassedAgents)

	return &GateEffResult{
		GeneratedAt:    time.Now().Format(time.RFC3339),
		DaysAnalyzed:   days,
		TotalSpawns:    totalSpawns,
		TotalWithGates: totalWithGates,
		PerGate:        perGate,
		Overall:        overall,
		BySkill:        bySkill,
		Verdict:        verdict,
		Caveats:        caveats,
	}
}

func buildCohort(agents []*agentRecord) GateEffCohort {
	c := GateEffCohort{Count: len(agents)}
	if c.Count == 0 {
		return c
	}

	var durations []float64
	for _, rec := range agents {
		if rec.completed {
			c.Completions++
			if rec.verificationPassed {
				c.VerificationPassed++
			}
			if rec.spawnTime > 0 && rec.completionTime > rec.spawnTime {
				dur := float64(rec.completionTime-rec.spawnTime) / 60.0
				if dur > 0 && dur < 480 { // cap at 8 hours
					durations = append(durations, dur)
				}
			}
		}
		if rec.abandoned {
			c.Abandonments++
		}
		c.NetAccretion += rec.netAccretion
		c.RiskFilesTouched += rec.riskFiles
	}

	c.CompletionRate = float64(c.Completions) / float64(c.Count) * 100
	if c.Completions > 0 {
		c.VerificationRate = float64(c.VerificationPassed) / float64(c.Completions) * 100
	}
	if len(durations) > 0 {
		total := 0.0
		for _, d := range durations {
			total += d
		}
		c.AvgDurationMin = total / float64(len(durations))
	}
	if c.Count > 0 {
		c.AvgNetDelta = float64(c.NetAccretion) / float64(c.Count)
	}

	return c
}

func decisionSeverity(decision string) int {
	switch decision {
	case "block":
		return 3
	case "bypass":
		return 2
	case "allow", "advisory":
		return 1
	default:
		return 0
	}
}

func generateVerdict(overall GateEffOverall, perGate []GateEffPerGate, totalSpawns int) (string, []string) {
	var caveats []string

	// Check sample sizes
	minCohort := 20
	if overall.Enforced.Count < minCohort || overall.Bypassed.Count < minCohort {
		caveats = append(caveats, fmt.Sprintf(
			"Small sample: enforced=%d, bypassed=%d (need >=%d each for confidence)",
			overall.Enforced.Count, overall.Bypassed.Count, minCohort))
	}

	// Check if gate coverage is meaningful
	totalGated := overall.Enforced.Count + overall.Bypassed.Count
	if totalSpawns > 0 && totalGated < totalSpawns/2 {
		caveats = append(caveats, fmt.Sprintf(
			"Low gate coverage: %d/%d spawns (%.0f%%) had gate interactions",
			totalGated, totalSpawns, float64(totalGated)/float64(totalSpawns)*100))
	}

	// Check for confounding: are bypassed agents using different skills?
	for _, pg := range perGate {
		if pg.Gate == "triage" && pg.Bypassed.Count > 0 && pg.Allowed.Count > 0 {
			// Triage bypass is operator choice — compare directly
			diff := pg.Allowed.CompletionRate - pg.Bypassed.CompletionRate
			if diff < -10 {
				caveats = append(caveats, fmt.Sprintf(
					"Triage-bypassed agents complete %.0f%% more often than triage-allowed — "+
						"possible selection bias (operators bypass triage for high-confidence work)",
					-diff))
			}
		}
	}

	// Generate verdict
	enfRate := overall.Enforced.CompletionRate
	bypRate := overall.Bypassed.CompletionRate
	enfVerif := overall.Enforced.VerificationRate
	bypVerif := overall.Bypassed.VerificationRate

	if overall.Enforced.Count < minCohort || overall.Bypassed.Count < minCohort {
		return "INSUFFICIENT DATA — need more gate interactions for reliable comparison", caveats
	}

	// Compute quality delta (weighted: 60% verification rate, 40% completion rate)
	enfScore := enfVerif*0.6 + enfRate*0.4
	bypScore := bypVerif*0.6 + bypRate*0.4
	delta := enfScore - bypScore

	var verdict string
	switch {
	case delta > 10:
		verdict = fmt.Sprintf("GATES IMPROVE QUALITY — enforced agents score %.0f points higher "+
			"(completion: %.0f%% vs %.0f%%, verification: %.0f%% vs %.0f%%)",
			delta, enfRate, bypRate, enfVerif, bypVerif)
	case delta > 3:
		verdict = fmt.Sprintf("WEAK POSITIVE — enforced agents score %.0f points higher, "+
			"but delta is small (completion: %.0f%% vs %.0f%%, verification: %.0f%% vs %.0f%%)",
			delta, enfRate, bypRate, enfVerif, bypVerif)
	case delta > -3:
		verdict = fmt.Sprintf("NO MEASURABLE DIFFERENCE — enforced vs bypassed within noise "+
			"(completion: %.0f%% vs %.0f%%, verification: %.0f%% vs %.0f%%)",
			enfRate, bypRate, enfVerif, bypVerif)
	case delta > -10:
		verdict = fmt.Sprintf("WEAK NEGATIVE — bypassed agents score %.0f points higher, "+
			"likely selection bias (completion: %.0f%% vs %.0f%%, verification: %.0f%% vs %.0f%%)",
			-delta, enfRate, bypRate, enfVerif, bypVerif)
	default:
		verdict = fmt.Sprintf("GATES MAY HURT — bypassed agents score %.0f points higher "+
			"(completion: %.0f%% vs %.0f%%, verification: %.0f%% vs %.0f%%). "+
			"Investigate: are gates blocking good work?",
			-delta, enfRate, bypRate, enfVerif, bypVerif)
	}

	// Add accretion insight
	enfAccretion := overall.Enforced.AvgNetDelta
	bypAccretion := overall.Bypassed.AvgNetDelta
	if enfAccretion > 0 || bypAccretion > 0 {
		verdict += fmt.Sprintf(" Accretion: enforced avg %.0f lines/agent vs bypassed %.0f lines/agent.",
			enfAccretion, bypAccretion)
	}

	return verdict, caveats
}

func formatGateEffText(result *GateEffResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "═══ GATE EFFECTIVENESS (%dd, %d spawns, %d with gates) ═══\n\n",
		result.DaysAnalyzed, result.TotalSpawns, result.TotalWithGates)

	// Per-gate tables
	for _, pg := range result.PerGate {
		fmt.Fprintf(&b, "── %s (%d evaluations) ──\n", strings.ToUpper(pg.Gate), pg.Total)
		fmt.Fprintf(&b, "  %-10s %6s %6s %6s %8s %8s %8s %10s\n",
			"Cohort", "N", "Comp", "Aband", "Comp%", "Verif%", "AvgDur", "AvgDelta")
		fmt.Fprintf(&b, "  %s\n", strings.Repeat("─", 72))

		for _, entry := range []struct {
			label string
			c     GateEffCohort
		}{
			{"Blocked", pg.Blocked},
			{"Bypassed", pg.Bypassed},
			{"Allowed", pg.Allowed},
		} {
			if entry.c.Count == 0 {
				continue
			}
			durStr := "—"
			if entry.c.AvgDurationMin > 0 {
				durStr = fmt.Sprintf("%.0fm", entry.c.AvgDurationMin)
			}
			fmt.Fprintf(&b, "  %-10s %6d %6d %6d %7.1f%% %7.1f%% %8s %+10.0f\n",
				entry.label, entry.c.Count, entry.c.Completions, entry.c.Abandonments,
				entry.c.CompletionRate, entry.c.VerificationRate, durStr, entry.c.AvgNetDelta)
		}
		fmt.Fprintln(&b)
	}

	// Overall comparison
	fmt.Fprintln(&b, "── OVERALL: ENFORCED vs BYPASSED ──")
	fmt.Fprintf(&b, "  %-10s %6s %6s %6s %8s %8s %8s %10s %8s\n",
		"Cohort", "N", "Comp", "Aband", "Comp%", "Verif%", "AvgDur", "AvgDelta", "RiskFiles")
	fmt.Fprintf(&b, "  %s\n", strings.Repeat("─", 82))

	for _, entry := range []struct {
		label string
		c     GateEffCohort
	}{
		{"Enforced", result.Overall.Enforced},
		{"Bypassed", result.Overall.Bypassed},
	} {
		durStr := "—"
		if entry.c.AvgDurationMin > 0 {
			durStr = fmt.Sprintf("%.0fm", entry.c.AvgDurationMin)
		}
		fmt.Fprintf(&b, "  %-10s %6d %6d %6d %7.1f%% %7.1f%% %8s %+10.0f %8d\n",
			entry.label, entry.c.Count, entry.c.Completions, entry.c.Abandonments,
			entry.c.CompletionRate, entry.c.VerificationRate, durStr,
			entry.c.AvgNetDelta, entry.c.RiskFilesTouched)
	}
	fmt.Fprintln(&b)

	// Skill breakdown
	if len(result.BySkill) > 0 {
		fmt.Fprintln(&b, "── BY SKILL ──")
		fmt.Fprintf(&b, "  %-20s %6s %6s %6s %8s\n",
			"Skill", "Gates", "Block", "Bypass", "Comp%")
		fmt.Fprintf(&b, "  %s\n", strings.Repeat("─", 52))
		for _, s := range result.BySkill {
			fmt.Fprintf(&b, "  %-20s %6d %6d %6d %7.1f%%\n",
				truncateSkill(s.Skill, 20), s.GateEvents, s.Blocks, s.Bypasses, s.CompletionRate)
		}
		fmt.Fprintln(&b)
	}

	// Verdict
	fmt.Fprintln(&b, "── VERDICT ──")
	fmt.Fprintf(&b, "  %s\n", result.Verdict)

	// Caveats
	if len(result.Caveats) > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "  Caveats:")
		for _, c := range result.Caveats {
			fmt.Fprintf(&b, "    • %s\n", c)
		}
	}
	fmt.Fprintln(&b)

	return b.String()
}
