// stats_aggregation.go - statsAggregator struct and methods for event processing and post-aggregation calculations.
// Decomposed from the monolithic aggregateStats() function in stats_cmd.go.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/coaching"
)

// statsAggregator holds intermediate state during event aggregation.
// The 20+ maps are shared between event processors and post-aggregation calculators.
type statsAggregator struct {
	report     *StatsReport
	days       int
	cutoffDays int64
	cutoff7d   int64
	cutoff30d  int64

	// Spawn correlation
	spawnTimes         map[string]int64  // session_id -> timestamp
	spawnSkills        map[string]string // session_id -> skill
	spawnBeadsIDs      map[string]string // session_id -> beads_id
	workspaceToSession map[string]string // workspace -> session_id (pseudo)

	// Deduplication
	completedBeadsIDs map[string]bool // beads_id -> true if already counted
	abandonedBeadsIDs map[string]bool // beads_id -> true if already counted

	// Duration tracking
	durations []float64

	// Beads completions
	beadsCompletions map[string]int64 // beads_id -> completion timestamp

	// Skill tracking
	skillCounts map[string]*SkillStatsSummary

	// Escape hatch
	escapeHatchSpawns   []escapeHatchSpawn
	escapeHatchInWindow int

	// Bypass tracking (direct orch spawn, not via daemon)
	bypassSpawns int

	// Verification
	gateFailures      map[string]int                    // gate -> failure count
	gatesBypassed     map[string]int                    // gate -> bypass count
	gatesAutoSkipped  map[string]int                    // gate -> auto-skip count
	bypassReasons     map[string]int                    // "gate|reason" -> count
	skillVerification map[string]*SkillVerificationStats // skill -> verification stats

	// Spawn gates
	spawnGateBypasses map[string]int // gate -> bypass count
	spawnGateReasons  map[string]int // "gate|reason" -> count

	// Overrides
	overrideReasons map[string]int // "type|reason" -> count

	// Gate decisions
	gateDecisionCounts    map[string]int  // "gate_name|decision" -> count
	gateBlockedSkills     map[string]int  // "gate_name|skill" -> count
	gatedBeadsIDs         map[string]bool // beads_id -> true if any gate_decision exists
	blockedBeadsIDs       map[string]bool // beads_id -> true if blocked by a gate
	architectEscalatedIDs map[string]bool // issue_id -> true if escalated to architect

	// Skill inference tracking
	skillInferences map[string]*skillInferenceRecord // issue_id -> inference record

	// Event window counter
	eventsInWindow int
}

func newStatsAggregator(days int) *statsAggregator {
	now := time.Now().Unix()
	return &statsAggregator{
		report: &StatsReport{
			GeneratedAt:    time.Now().Format(time.RFC3339),
			AnalysisPeriod: fmt.Sprintf("Last %d days", days),
			DaysAnalyzed:   days,
			SkillStats:     []SkillStatsSummary{},
		},
		days:       days,
		cutoffDays: now - int64(days*86400),
		cutoff7d:   now - int64(7*86400),
		cutoff30d:  now - int64(30*86400),

		spawnTimes:         make(map[string]int64),
		spawnSkills:        make(map[string]string),
		spawnBeadsIDs:      make(map[string]string),
		workspaceToSession: make(map[string]string),
		completedBeadsIDs:  make(map[string]bool),
		abandonedBeadsIDs:  make(map[string]bool),
		beadsCompletions:   make(map[string]int64),
		skillCounts:        make(map[string]*SkillStatsSummary),

		gateFailures:      make(map[string]int),
		gatesBypassed:     make(map[string]int),
		gatesAutoSkipped:  make(map[string]int),
		bypassReasons:     make(map[string]int),
		skillVerification: make(map[string]*SkillVerificationStats),

		spawnGateBypasses: make(map[string]int),
		spawnGateReasons:  make(map[string]int),
		overrideReasons:   make(map[string]int),

		gateDecisionCounts:    make(map[string]int),
		gateBlockedSkills:     make(map[string]int),
		gatedBeadsIDs:         make(map[string]bool),
		blockedBeadsIDs:       make(map[string]bool),
		architectEscalatedIDs: make(map[string]bool),

		skillInferences: make(map[string]*skillInferenceRecord),
	}
}

// --- Event Processors ---

func (a *statsAggregator) processSessionSpawned(event StatsEvent) {
	var beadsID, skill, workspace, spawnMode, account string
	if data := event.Data; data != nil {
		if s, ok := data["skill"].(string); ok && s != "" {
			skill = s
		}
		if b, ok := data["beads_id"].(string); ok && b != "" {
			beadsID = b
		}
		if w, ok := data["workspace"].(string); ok && w != "" {
			workspace = w
		}
		if m, ok := data["spawn_mode"].(string); ok && m != "" {
			spawnMode = m
		}
		if ac, ok := data["usage_account"].(string); ok && ac != "" {
			account = ac
		}
	}

	// Track no_track override reason (within analysis window)
	if event.Timestamp >= a.cutoffDays {
		if data := event.Data; data != nil {
			if reason, ok := data["no_track_reason"].(string); ok && reason != "" {
				a.overrideReasons["no_track|"+reason]++
			}
		}
	}

	// Track escape hatch spawns (spawn_mode = "claude")
	if spawnMode == "claude" {
		a.escapeHatchSpawns = append(a.escapeHatchSpawns, escapeHatchSpawn{
			timestamp: event.Timestamp,
			account:   account,
		})
		if event.Timestamp >= a.cutoffDays {
			a.escapeHatchInWindow++
		}
	}

	// Determine the effective session ID
	effectiveSessionID := event.SessionID
	if effectiveSessionID == "" && workspace != "" {
		effectiveSessionID = "ws:" + workspace
	}

	a.spawnTimes[effectiveSessionID] = event.Timestamp
	if skill != "" {
		a.spawnSkills[effectiveSessionID] = skill
	}
	if beadsID != "" {
		a.spawnBeadsIDs[effectiveSessionID] = beadsID
	}
	if workspace != "" {
		a.workspaceToSession[workspace] = effectiveSessionID
	}

	// Skip events outside the --days window for main stats
	if event.Timestamp < a.cutoffDays {
		return
	}

	a.report.Summary.TotalSpawns++
	if skill != "" {
		if _, exists := a.skillCounts[skill]; !exists {
			a.skillCounts[skill] = &SkillStatsSummary{
				Skill:    skill,
				Category: getSkillCategory(skill),
			}
		}
		a.skillCounts[skill].Spawns++
	}
}

func (a *statsAggregator) processSessionCompleted(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}

	var sessionBeadsID string
	if data := event.Data; data != nil {
		if b, ok := data["beads_id"].(string); ok && b != "" {
			sessionBeadsID = b
		}
	}

	deduplicationKey := sessionBeadsID
	if sessionBeadsID == "" {
		deduplicationKey = "session:" + event.SessionID
	}

	if !a.completedBeadsIDs[deduplicationKey] {
		a.completedBeadsIDs[deduplicationKey] = true
		a.report.Summary.TotalCompletions++
		if spawnTime, ok := a.spawnTimes[event.SessionID]; ok {
			duration := float64(event.Timestamp-spawnTime) / 60.0
			if duration > 0 && duration < 480 {
				a.durations = append(a.durations, duration)
			}
		}
		if skill, ok := a.spawnSkills[event.SessionID]; ok {
			if stats, exists := a.skillCounts[skill]; exists {
				stats.Completions++
			}
		}
	}
}

func (a *statsAggregator) processAgentCompleted(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}

	var beadsID, workspace, sessionID string
	var verificationPassed, wasForced bool
	var eventGatesBypassed []string
	var eventSkill string

	if data := event.Data; data != nil {
		if b, ok := data["beads_id"].(string); ok && b != "" {
			beadsID = b
			a.beadsCompletions[beadsID] = event.Timestamp
			for sid, spawnBeadsID := range a.spawnBeadsIDs {
				if spawnBeadsID == beadsID {
					sessionID = sid
					break
				}
			}
		}
		if w, ok := data["workspace"].(string); ok && w != "" {
			workspace = w
			if sessionID == "" && workspace != "" {
				if sid, ok := a.workspaceToSession[workspace]; ok {
					sessionID = sid
				}
			}
		}
		if vp, ok := data["verification_passed"].(bool); ok {
			verificationPassed = vp
		}
		if f, ok := data["forced"].(bool); ok {
			wasForced = f
		}
		if gb, ok := data["gates_bypassed"].([]interface{}); ok {
			for _, g := range gb {
				if gate, ok := g.(string); ok {
					eventGatesBypassed = append(eventGatesBypassed, gate)
				}
			}
		}
		if s, ok := data["skill"].(string); ok && s != "" {
			eventSkill = s
		}
		if wasForced {
			if reason, ok := data["force_reason"].(string); ok && reason != "" {
				a.overrideReasons["force_complete|"+reason]++
			}
		}
	}

	deduplicationKey := beadsID
	if beadsID == "" {
		deduplicationKey = "ws:" + workspace
	}

	alreadyCounted := a.completedBeadsIDs[deduplicationKey]

	if !alreadyCounted {
		a.completedBeadsIDs[deduplicationKey] = true
		a.report.Summary.TotalCompletions++
		if sessionID != "" {
			if skill, ok := a.spawnSkills[sessionID]; ok {
				if stats, exists := a.skillCounts[skill]; exists {
					stats.Completions++
				}
			}
		}
	}

	// Always calculate duration (but only add on first count)
	if sessionID != "" {
		if spawnTime, ok := a.spawnTimes[sessionID]; ok {
			duration := float64(event.Timestamp-spawnTime) / 60.0
			if duration > 0 && duration < 480 {
				if !alreadyCounted {
					a.durations = append(a.durations, duration)
				}
			}
		}
	}

	// Track verification metrics (only for unique completions)
	if !alreadyCounted {
		a.report.VerificationStats.TotalAttempts++
		if verificationPassed {
			a.report.VerificationStats.PassedFirstTry++
		}
		if wasForced {
			a.report.VerificationStats.Bypassed++
			for _, gate := range eventGatesBypassed {
				a.gatesBypassed[gate]++
			}
		}
		skill := eventSkill
		if skill == "" && sessionID != "" {
			skill = a.spawnSkills[sessionID]
		}
		if skill != "" {
			if _, exists := a.skillVerification[skill]; !exists {
				a.skillVerification[skill] = &SkillVerificationStats{Skill: skill}
			}
			a.skillVerification[skill].TotalAttempts++
			if verificationPassed {
				a.skillVerification[skill].PassedFirstTry++
			}
			if wasForced {
				a.skillVerification[skill].Bypassed++
			}
		}
	}
}

func (a *statsAggregator) processAgentAbandoned(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}

	var beadsID, sessionID string
	if data := event.Data; data != nil {
		if b, ok := data["beads_id"].(string); ok && b != "" {
			beadsID = b
			for sid, spawnBeadsID := range a.spawnBeadsIDs {
				if spawnBeadsID == beadsID {
					sessionID = sid
					break
				}
			}
		}
	}

	deduplicationKey := beadsID
	if beadsID == "" {
		deduplicationKey = fmt.Sprintf("ts:%d", event.Timestamp)
	}

	if !a.abandonedBeadsIDs[deduplicationKey] {
		a.abandonedBeadsIDs[deduplicationKey] = true
		a.report.Summary.TotalAbandonments++
		if sessionID != "" {
			if skill, ok := a.spawnSkills[sessionID]; ok {
				if stats, exists := a.skillCounts[skill]; exists {
					stats.Abandonments++
				}
			}
		}
	}
}

func (a *statsAggregator) processDaemonSpawn(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}
	a.report.DaemonStats.DaemonSpawns++
}

func (a *statsAggregator) processSpawnBypass(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}
	a.bypassSpawns++
}

func (a *statsAggregator) processAutoCompleted(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}
	a.report.DaemonStats.AutoCompletions++
}

func (a *statsAggregator) processTriageBypassed(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}
	a.report.DaemonStats.TriageBypassed++
	a.spawnGateBypasses["triage"]++
	if data := event.Data; data != nil {
		if reason, ok := data["reason"].(string); ok && reason != "" {
			a.overrideReasons["triage_bypassed|"+reason]++
			a.spawnGateReasons["triage|"+reason]++
		}
	}
}

func (a *statsAggregator) processHotspotBypassed(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}
	a.spawnGateBypasses["hotspot"]++
	if data := event.Data; data != nil {
		reason, _ := data["reason"].(string)
		if reason != "" {
			a.overrideReasons["hotspot_bypassed|"+reason]++
			a.spawnGateReasons["hotspot|"+reason]++
		} else {
			a.overrideReasons["hotspot_bypassed|"]++
		}
	}
}

func (a *statsAggregator) processVerificationBypassed(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}
	a.spawnGateBypasses["verification"]++
	if data := event.Data; data != nil {
		if reason, ok := data["reason"].(string); ok && reason != "" {
			a.overrideReasons["verification_bypassed|"+reason]++
			a.spawnGateReasons["verification|"+reason]++
		}
	}
}

func (a *statsAggregator) processWaitComplete(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}
	a.report.WaitStats.WaitCompleted++
}

func (a *statsAggregator) processWaitTimeout(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}
	a.report.WaitStats.WaitTimeouts++
}

func (a *statsAggregator) processSessionStarted(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}
	a.report.SessionStats.SessionsStarted++
}

func (a *statsAggregator) processSessionEnded(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}
	a.report.SessionStats.SessionsEnded++
}

func (a *statsAggregator) processVerificationFailed(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}
	if data := event.Data; data != nil {
		if gatesFailed, ok := data["gates_failed"].([]interface{}); ok {
			for _, g := range gatesFailed {
				if gate, ok := g.(string); ok {
					a.gateFailures[gate]++
				}
			}
		}
		if skill, ok := data["skill"].(string); ok && skill != "" {
			if _, exists := a.skillVerification[skill]; !exists {
				a.skillVerification[skill] = &SkillVerificationStats{Skill: skill}
			}
		}
	}
}

func (a *statsAggregator) processVerificationBypassedEvent(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}
	a.report.VerificationStats.SkipBypassed++
	if data := event.Data; data != nil {
		if gate, ok := data["gate"].(string); ok && gate != "" {
			a.gatesBypassed[gate]++
			reason, _ := data["reason"].(string)
			a.bypassReasons[gate+"|"+reason]++
		}
	}
}

func (a *statsAggregator) processVerificationAutoSkipped(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}
	a.report.VerificationStats.AutoSkipped++
	if data := event.Data; data != nil {
		if gate, ok := data["gate"].(string); ok && gate != "" {
			a.gatesAutoSkipped[gate]++
		}
	}
}

func (a *statsAggregator) processGateDecision(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}
	if data := event.Data; data != nil {
		gateName, _ := data["gate_name"].(string)
		decision, _ := data["decision"].(string)
		skill, _ := data["skill"].(string)
		beadsID, _ := data["beads_id"].(string)
		if gateName != "" && decision != "" {
			a.gateDecisionCounts[gateName+"|"+decision]++
			if decision == "block" && skill != "" {
				a.gateBlockedSkills[gateName+"|"+skill]++
			}
			if beadsID != "" {
				a.gatedBeadsIDs[beadsID] = true
				if decision == "block" {
					a.blockedBeadsIDs[beadsID] = true
				}
			}
		}
	}
}

func (a *statsAggregator) processArchitectEscalation(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}
	if data := event.Data; data != nil {
		issueID, _ := data["issue_id"].(string)
		escalated, _ := data["escalated"].(bool)
		if issueID != "" && escalated {
			a.architectEscalatedIDs[issueID] = true
		}
	}
}

// --- Post-Aggregation Calculators ---

func (a *statsAggregator) calcEscapeHatchStats() {
	accountTotals := make(map[string]*AccountSpawnBreakdown)
	for _, eh := range a.escapeHatchSpawns {
		a.report.EscapeHatchStats.TotalSpawns++
		if eh.timestamp >= a.cutoff7d {
			a.report.EscapeHatchStats.Last7DaySpawns++
		}
		if eh.timestamp >= a.cutoff30d {
			a.report.EscapeHatchStats.Last30DaySpawns++
		}

		acct := eh.account
		if acct == "" {
			acct = "unknown"
		}
		if _, exists := accountTotals[acct]; !exists {
			accountTotals[acct] = &AccountSpawnBreakdown{Account: acct}
		}
		accountTotals[acct].TotalSpawns++
		if eh.timestamp >= a.cutoff7d {
			accountTotals[acct].Last7Days++
		}
		if eh.timestamp >= a.cutoff30d {
			accountTotals[acct].Last30Days++
		}
	}

	for _, breakdown := range accountTotals {
		a.report.EscapeHatchStats.ByAccount = append(a.report.EscapeHatchStats.ByAccount, *breakdown)
	}
	sort.Slice(a.report.EscapeHatchStats.ByAccount, func(i, j int) bool {
		return a.report.EscapeHatchStats.ByAccount[i].TotalSpawns > a.report.EscapeHatchStats.ByAccount[j].TotalSpawns
	})

	if a.report.Summary.TotalSpawns > 0 {
		a.report.EscapeHatchStats.EscapeHatchRate = float64(a.escapeHatchInWindow) / float64(a.report.Summary.TotalSpawns) * 100
	}
}

func (a *statsAggregator) calcVerificationStats() {
	if a.report.VerificationStats.TotalAttempts > 0 {
		a.report.VerificationStats.PassRate = float64(a.report.VerificationStats.PassedFirstTry) / float64(a.report.VerificationStats.TotalAttempts) * 100
		a.report.VerificationStats.BypassRate = float64(a.report.VerificationStats.Bypassed) / float64(a.report.VerificationStats.TotalAttempts) * 100
	}

	// Build gate failure stats
	allGates := make(map[string]bool)
	for gate := range a.gateFailures {
		allGates[gate] = true
	}
	for gate := range a.gatesBypassed {
		allGates[gate] = true
	}
	for gate := range a.gatesAutoSkipped {
		allGates[gate] = true
	}
	for gate := range allGates {
		gateStats := GateFailureStats{
			Gate:          gate,
			FailCount:     a.gateFailures[gate],
			BypassCount:   a.gatesBypassed[gate],
			AutoSkipCount: a.gatesAutoSkipped[gate],
		}
		if a.report.VerificationStats.TotalAttempts > 0 {
			gateStats.FailRate = float64(gateStats.FailCount) / float64(a.report.VerificationStats.TotalAttempts) * 100
		}
		a.report.VerificationStats.FailuresByGate = append(a.report.VerificationStats.FailuresByGate, gateStats)
	}
	sort.Slice(a.report.VerificationStats.FailuresByGate, func(i, j int) bool {
		return a.report.VerificationStats.FailuresByGate[i].FailCount > a.report.VerificationStats.FailuresByGate[j].FailCount
	})

	// Build bypass reasons
	for key, count := range a.bypassReasons {
		parts := strings.SplitN(key, "|", 2)
		gate := parts[0]
		reason := ""
		if len(parts) > 1 {
			reason = parts[1]
		}
		a.report.VerificationStats.BypassReasons = append(a.report.VerificationStats.BypassReasons, BypassReasonEntry{
			Gate:   gate,
			Reason: reason,
			Count:  count,
		})
	}
	sort.Slice(a.report.VerificationStats.BypassReasons, func(i, j int) bool {
		return a.report.VerificationStats.BypassReasons[i].Count > a.report.VerificationStats.BypassReasons[j].Count
	})

	// Build skill verification stats
	for _, sv := range a.skillVerification {
		if sv.TotalAttempts > 0 {
			sv.PassRate = float64(sv.PassedFirstTry) / float64(sv.TotalAttempts) * 100
		}
		a.report.VerificationStats.BySkill = append(a.report.VerificationStats.BySkill, *sv)
	}
	sort.Slice(a.report.VerificationStats.BySkill, func(i, j int) bool {
		return a.report.VerificationStats.BySkill[i].TotalAttempts > a.report.VerificationStats.BySkill[j].TotalAttempts
	})
}

func (a *statsAggregator) calcSpawnGateStats() {
	for _, count := range a.spawnGateBypasses {
		a.report.SpawnGateStats.TotalBypasses += count
	}
	a.report.SpawnGateStats.TotalSpawns = a.report.Summary.TotalSpawns
	if a.report.SpawnGateStats.TotalSpawns > 0 {
		a.report.SpawnGateStats.BypassRate = float64(a.report.SpawnGateStats.TotalBypasses) / float64(a.report.SpawnGateStats.TotalSpawns) * 100
	}
	for gate, count := range a.spawnGateBypasses {
		entry := SpawnGateEntry{
			Gate:     gate,
			Bypassed: count,
		}
		if a.report.SpawnGateStats.TotalSpawns > 0 {
			entry.BypassRate = float64(count) / float64(a.report.SpawnGateStats.TotalSpawns) * 100
		}
		if entry.BypassRate > 50 {
			entry.Miscalibrated = true
		}
		a.report.SpawnGateStats.ByGate = append(a.report.SpawnGateStats.ByGate, entry)
	}
	sort.Slice(a.report.SpawnGateStats.ByGate, func(i, j int) bool {
		return a.report.SpawnGateStats.ByGate[i].Bypassed > a.report.SpawnGateStats.ByGate[j].Bypassed
	})
	// Build spawn gate top reasons
	for key, count := range a.spawnGateReasons {
		parts := strings.SplitN(key, "|", 2)
		gate := parts[0]
		reason := ""
		if len(parts) > 1 {
			reason = parts[1]
		}
		if reason != "" {
			a.report.SpawnGateStats.TopReasons = append(a.report.SpawnGateStats.TopReasons, SpawnGateReasonEntry{
				Gate:   gate,
				Reason: reason,
				Count:  count,
			})
		}
	}
	sort.Slice(a.report.SpawnGateStats.TopReasons, func(i, j int) bool {
		return a.report.SpawnGateStats.TopReasons[i].Count > a.report.SpawnGateStats.TopReasons[j].Count
	})
}

func (a *statsAggregator) calcOverrideStats() {
	overrideByType := make(map[string]map[string]int)
	for key, count := range a.overrideReasons {
		parts := strings.SplitN(key, "|", 2)
		overrideType := parts[0]
		reason := ""
		if len(parts) > 1 {
			reason = parts[1]
		}
		if _, exists := overrideByType[overrideType]; !exists {
			overrideByType[overrideType] = make(map[string]int)
		}
		overrideByType[overrideType][reason] += count
		a.report.OverrideStats.TotalOverrides += count
	}
	for overrideType, reasons := range overrideByType {
		entry := OverrideTypeEntry{Type: overrideType}
		for reason, count := range reasons {
			entry.Count += count
			if reason != "" {
				entry.Reasons = append(entry.Reasons, OverrideReasonEntry{
					Reason: reason,
					Count:  count,
				})
			}
		}
		sort.Slice(entry.Reasons, func(i, j int) bool {
			return entry.Reasons[i].Count > entry.Reasons[j].Count
		})
		a.report.OverrideStats.ByType = append(a.report.OverrideStats.ByType, entry)
	}
	sort.Slice(a.report.OverrideStats.ByType, func(i, j int) bool {
		return a.report.OverrideStats.ByType[i].Count > a.report.OverrideStats.ByType[j].Count
	})
	for _, entry := range a.report.OverrideStats.ByType {
		for _, reason := range entry.Reasons {
			a.report.OverrideStats.TopReasons = append(a.report.OverrideStats.TopReasons, reason)
		}
	}
	sort.Slice(a.report.OverrideStats.TopReasons, func(i, j int) bool {
		return a.report.OverrideStats.TopReasons[i].Count > a.report.OverrideStats.TopReasons[j].Count
	})
}

func (a *statsAggregator) calcRatesAndDuration() {
	if a.report.Summary.TotalSpawns > 0 {
		a.report.Summary.CompletionRate = float64(a.report.Summary.TotalCompletions) / float64(a.report.Summary.TotalSpawns) * 100
		a.report.Summary.AbandonmentRate = float64(a.report.Summary.TotalAbandonments) / float64(a.report.Summary.TotalSpawns) * 100
		a.report.DaemonStats.DaemonSpawnRate = float64(a.report.DaemonStats.DaemonSpawns) / float64(a.report.Summary.TotalSpawns) * 100
		a.report.DaemonStats.BypassSpawns = a.bypassSpawns
		totalSpawns := a.report.DaemonStats.DaemonSpawns + a.bypassSpawns
		if totalSpawns > 0 {
			a.report.DaemonStats.BypassRate = float64(a.bypassSpawns) / float64(totalSpawns) * 100
		}
	}

	if len(a.durations) > 0 {
		var total float64
		for _, d := range a.durations {
			total += d
		}
		a.report.Summary.AvgDurationMinutes = total / float64(len(a.durations))
	}

	totalWaits := a.report.WaitStats.WaitCompleted + a.report.WaitStats.WaitTimeouts
	if totalWaits > 0 {
		a.report.WaitStats.TimeoutRate = float64(a.report.WaitStats.WaitTimeouts) / float64(totalWaits) * 100
	}

	a.report.SessionStats.ActiveSessions = a.report.SessionStats.SessionsStarted - a.report.SessionStats.SessionsEnded

	for _, stats := range a.skillCounts {
		if stats.Spawns > 0 {
			stats.CompletionRate = float64(stats.Completions) / float64(stats.Spawns) * 100
		}
		a.report.SkillStats = append(a.report.SkillStats, *stats)

		if stats.Category == TaskSkill {
			a.report.Summary.TaskSpawns += stats.Spawns
			a.report.Summary.TaskCompletions += stats.Completions
		} else if stats.Category == CoordinationSkill {
			a.report.Summary.CoordinationSpawns += stats.Spawns
			a.report.Summary.CoordinationCompletions += stats.Completions
		}
	}

	if a.report.Summary.TaskSpawns > 0 {
		a.report.Summary.TaskCompletionRate = float64(a.report.Summary.TaskCompletions) / float64(a.report.Summary.TaskSpawns) * 100
	}
	if a.report.Summary.CoordinationSpawns > 0 {
		a.report.Summary.CoordinationCompletionRate = float64(a.report.Summary.CoordinationCompletions) / float64(a.report.Summary.CoordinationSpawns) * 100
	}

	sort.Slice(a.report.SkillStats, func(i, j int) bool {
		return a.report.SkillStats[i].Spawns > a.report.SkillStats[j].Spawns
	})
}

func (a *statsAggregator) calcGateDecisionStats() {
	gateBlocks := make(map[string]int)
	gateBypasses := make(map[string]int)
	gateAllows := make(map[string]int)
	for key, count := range a.gateDecisionCounts {
		parts := strings.SplitN(key, "|", 2)
		gateName := parts[0]
		decision := ""
		if len(parts) > 1 {
			decision = parts[1]
		}
		a.report.GateDecisionStats.TotalDecisions += count
		if decision == "block" {
			a.report.GateDecisionStats.TotalBlocks += count
			gateBlocks[gateName] += count
		} else if decision == "bypass" {
			a.report.GateDecisionStats.TotalBypasses += count
			gateBypasses[gateName] += count
		} else if decision == "allow" {
			a.report.GateDecisionStats.TotalAllows += count
			gateAllows[gateName] += count
		}
	}
	allGateNames := make(map[string]bool)
	for g := range gateBlocks {
		allGateNames[g] = true
	}
	for g := range gateBypasses {
		allGateNames[g] = true
	}
	for g := range gateAllows {
		allGateNames[g] = true
	}
	for g := range allGateNames {
		a.report.GateDecisionStats.ByGate = append(a.report.GateDecisionStats.ByGate, GateDecisionEntry{
			Gate:     g,
			Blocks:   gateBlocks[g],
			Bypasses: gateBypasses[g],
			Allows:   gateAllows[g],
		})
	}
	sort.Slice(a.report.GateDecisionStats.ByGate, func(i, j int) bool {
		totalI := a.report.GateDecisionStats.ByGate[i].Blocks + a.report.GateDecisionStats.ByGate[i].Bypasses + a.report.GateDecisionStats.ByGate[i].Allows
		totalJ := a.report.GateDecisionStats.ByGate[j].Blocks + a.report.GateDecisionStats.ByGate[j].Bypasses + a.report.GateDecisionStats.ByGate[j].Allows
		return totalI > totalJ
	})
	for key, count := range a.gateBlockedSkills {
		parts := strings.SplitN(key, "|", 2)
		gateName := parts[0]
		skill := ""
		if len(parts) > 1 {
			skill = parts[1]
		}
		a.report.GateDecisionStats.TopBlockedSkills = append(a.report.GateDecisionStats.TopBlockedSkills, GateSkillEntry{
			Gate:  gateName,
			Skill: skill,
			Count: count,
		})
	}
	sort.Slice(a.report.GateDecisionStats.TopBlockedSkills, func(i, j int) bool {
		return a.report.GateDecisionStats.TopBlockedSkills[i].Count > a.report.GateDecisionStats.TopBlockedSkills[j].Count
	})
}

func (a *statsAggregator) calcGateEffectiveness(events []StatsEvent) {
	if len(a.gatedBeadsIDs) == 0 && len(a.architectEscalatedIDs) == 0 {
		return
	}

	ge := &a.report.GateEffectivenessStats
	ge.TotalEvaluations = a.report.GateDecisionStats.TotalDecisions
	ge.TotalBlocks = a.report.GateDecisionStats.TotalBlocks
	ge.TotalBypasses = a.report.GateDecisionStats.TotalBypasses
	ge.TotalAllows = ge.TotalEvaluations - ge.TotalBlocks - ge.TotalBypasses
	if ge.TotalEvaluations > 0 {
		ge.BlockRate = float64(ge.TotalBlocks) / float64(ge.TotalEvaluations) * 100
	}

	ge.ArchitectEscalations = len(a.architectEscalatedIDs)

	// Blocked outcome correlation
	for beadsID := range a.blockedBeadsIDs {
		if a.architectEscalatedIDs[beadsID] {
			ge.BlockedOutcomes.EscalatedToArchitect++
		}
		if a.completedBeadsIDs[beadsID] {
			ge.BlockedOutcomes.EventuallyCompleted++
		} else if !a.abandonedBeadsIDs[beadsID] {
			ge.BlockedOutcomes.StillPending++
		}
	}

	// Quality metrics: iterate all spawns and classify as gated vs ungated
	var gatedDurations, ungatedDurations []float64
	for sid, spawnTime := range a.spawnTimes {
		if spawnTime < a.cutoffDays {
			continue
		}
		beadsID := a.spawnBeadsIDs[sid]
		if beadsID == "" {
			continue
		}

		isGated := a.gatedBeadsIDs[beadsID]
		var metrics *QualityMetrics
		if isGated {
			metrics = &ge.GatedCompletion
		} else {
			metrics = &ge.UngatedCompletion
		}
		metrics.TotalSpawns++

		if a.completedBeadsIDs[beadsID] {
			metrics.Completions++
			for _, event := range events {
				if event.Type != "agent.completed" || event.Timestamp < a.cutoffDays {
					continue
				}
				if data := event.Data; data != nil {
					if b, ok := data["beads_id"].(string); ok && b == beadsID {
						if vp, ok := data["verification_passed"].(bool); ok && vp {
							metrics.VerificationPassed++
						}
						duration := float64(event.Timestamp-spawnTime) / 60.0
						if duration > 0 && duration < 480 {
							if isGated {
								gatedDurations = append(gatedDurations, duration)
							} else {
								ungatedDurations = append(ungatedDurations, duration)
							}
						}
						break
					}
				}
			}
		} else if a.abandonedBeadsIDs[beadsID] {
			metrics.Abandonments++
		}
	}

	for _, metrics := range []*QualityMetrics{&ge.GatedCompletion, &ge.UngatedCompletion} {
		if metrics.TotalSpawns > 0 {
			metrics.CompletionRate = float64(metrics.Completions) / float64(metrics.TotalSpawns) * 100
		}
		if metrics.Completions > 0 {
			metrics.VerificationRate = float64(metrics.VerificationPassed) / float64(metrics.Completions) * 100
		}
	}

	if len(gatedDurations) > 0 {
		var total float64
		for _, d := range gatedDurations {
			total += d
		}
		ge.GatedCompletion.AvgDurationMinutes = total / float64(len(gatedDurations))
	}
	if len(ungatedDurations) > 0 {
		var total float64
		for _, d := range ungatedDurations {
			total += d
		}
		ge.UngatedCompletion.AvgDurationMinutes = total / float64(len(ungatedDurations))
	}
}

func (a *statsAggregator) calcCoachingStats() {
	home, err := os.UserHomeDir()
	if err == nil {
		coachingPath := filepath.Join(home, ".orch", "coaching-metrics.jsonl")
		since := time.Now().Add(-time.Duration(a.days) * 24 * time.Hour)
		coachingMetrics, err := coaching.ReadMetricsSince(coachingPath, since)
		if err == nil && len(coachingMetrics) > 0 {
			a.report.CoachingStats = coaching.AggregateByType(coachingMetrics, since)
		}
	}
}
