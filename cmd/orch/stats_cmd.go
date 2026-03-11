// stats_cmd.go - Aggregate events.jsonl metrics for orchestration observability
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/coaching"
	"github.com/spf13/cobra"
)

var (
	statsDays       int
	statsJSONOutput bool
	statsVerbose    bool
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
  orch stats --verbose          # Show additional metrics`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStats()
	},
}

func init() {
	statsCmd.Flags().IntVar(&statsDays, "days", 7, "Number of days to analyze")
	statsCmd.Flags().BoolVar(&statsJSONOutput, "json", false, "Output as JSON")
	statsCmd.Flags().BoolVar(&statsVerbose, "verbose", false, "Show additional metrics")
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
	report := &StatsReport{
		GeneratedAt:    time.Now().Format(time.RFC3339),
		AnalysisPeriod: fmt.Sprintf("Last %d days", days),
		DaysAnalyzed:   days,
		SkillStats:     []SkillStatsSummary{},
	}

	// Time window cutoffs
	now := time.Now().Unix()
	cutoffDays := now - int64(days*86400) // --days window for main stats
	cutoff7d := now - int64(7*86400)      // 7 days for escape hatch
	cutoff30d := now - int64(30*86400)    // 30 days for escape hatch

	// Track spawn times for duration calculation
	spawnTimes := make(map[string]int64)     // session_id -> timestamp
	spawnSkills := make(map[string]string)   // session_id -> skill
	spawnBeadsIDs := make(map[string]string) // session_id -> beads_id
	skillCounts := make(map[string]*SkillStatsSummary)
	var durations []float64

	// Track beads completions to correlate with spawns
	beadsCompletions := make(map[string]int64) // beads_id -> completion timestamp

	// Track unique completions by beads_id to avoid double-counting
	// When multiple agent.completed events exist for same beads_id, we:
	// 1. Only count the completion once
	// 2. Use the latest event for duration calculation
	completedBeadsIDs := make(map[string]bool) // beads_id -> true if already counted

	// Track unique abandonments by beads_id to avoid double-counting.
	// orch abandon emits two agent.abandoned events per abandonment:
	// one from LifecycleManager (basic data) and one from telemetry (enriched with skill/tokens).
	// Additionally, the same issue may be abandoned multiple times (retries).
	// We only count each unique beads_id once.
	abandonedBeadsIDs := make(map[string]bool) // beads_id -> true if already counted

	// Track workspace -> session mapping for orchestrator completions
	// (orchestrators don't have beads_id, they use workspace for correlation)
	workspaceToSession := make(map[string]string) // workspace -> session_id (pseudo)

	// Track escape hatch spawns (spawn_mode = "claude")
	// This is tracked separately to support multi-window analysis (total, 7d, 30d)
	type escapeHatchSpawn struct {
		timestamp int64
		account   string
	}
	var escapeHatchSpawns []escapeHatchSpawn
	escapeHatchInWindow := 0 // count of escape hatch spawns within --days window

	// Track verification stats
	// gateFailures tracks how many times each gate failed across all verification.failed events
	gateFailures := make(map[string]int)                          // gate -> failure count
	gatesBypassed := make(map[string]int)                         // gate -> bypass count (from agent.completed with forced=true, or verification.bypassed)
	gatesAutoSkipped := make(map[string]int)                      // gate -> auto-skip count (from verification.auto_skipped)
	bypassReasons := make(map[string]int)                         // "gate|reason" -> count (from verification.bypassed)
	skillVerification := make(map[string]*SkillVerificationStats) // skill -> verification stats

	// Track spawn gate bypasses (triage, hotspot, verification) for unified view
	spawnGateBypasses := make(map[string]int) // gate -> bypass count
	spawnGateReasons := make(map[string]int)  // "gate|reason" -> count

	// Track override reasons across all override types
	// key: "type|reason", value: count
	overrideReasons := make(map[string]int)

	// Track gate decision events (spawn.gate_decision)
	// key: "gate_name|decision", value: count
	gateDecisionCounts := make(map[string]int)
	// key: "gate_name|skill", value: count (blocks only)
	gateBlockedSkills := make(map[string]int)

	// Gate effectiveness: track which beads_ids had gate decisions
	// Gate effectiveness: track which beads_ids had gate decisions
	gatedBeadsIDs := make(map[string]bool)        // beads_id -> true if any gate_decision exists
	blockedBeadsIDs := make(map[string]bool)       // beads_id -> true if blocked by a gate
	architectEscalatedIDs := make(map[string]bool) // issue_id -> true if escalated to architect

	// Count events within analysis window for EventsAnalyzed
	eventsInWindow := 0

	for _, event := range events {
		// Track events in analysis window
		if event.Timestamp >= cutoffDays {
			eventsInWindow++
		}

		switch event.Type {
		case "session.spawned":
			// Extract skill, beads_id, workspace, spawn_mode, and account from data
			var beadsID string
			var skill string
			var workspace string
			var spawnMode string
			var account string
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
				if a, ok := data["usage_account"].(string); ok && a != "" {
					account = a
				}
			}

			// Track no_track override reason (within analysis window)
			if event.Timestamp >= cutoffDays {
				if data := event.Data; data != nil {
					if reason, ok := data["no_track_reason"].(string); ok && reason != "" {
						overrideReasons["no_track|"+reason]++
					}
				}
			}

			// Track escape hatch spawns (spawn_mode = "claude")
			// This is tracked across all time windows for comprehensive metrics
			if spawnMode == "claude" {
				escapeHatchSpawns = append(escapeHatchSpawns, escapeHatchSpawn{
					timestamp: event.Timestamp,
					account:   account,
				})
				// Track if within --days window for escape hatch rate calculation
				if event.Timestamp >= cutoffDays {
					escapeHatchInWindow++
				}
			}

			// Determine the effective session ID
			// For orchestrators (empty SessionID), use workspace as the key
			effectiveSessionID := event.SessionID
			if effectiveSessionID == "" && workspace != "" {
				effectiveSessionID = "ws:" + workspace // prefix to avoid collision
			}

			spawnTimes[effectiveSessionID] = event.Timestamp
			if skill != "" {
				spawnSkills[effectiveSessionID] = skill
			}
			if beadsID != "" {
				spawnBeadsIDs[effectiveSessionID] = beadsID
			}
			if workspace != "" {
				workspaceToSession[workspace] = effectiveSessionID
			}

			// Skip events outside the --days window for main stats
			if event.Timestamp < cutoffDays {
				continue
			}

			report.Summary.TotalSpawns++
			if skill != "" {
				if _, exists := skillCounts[skill]; !exists {
					skillCounts[skill] = &SkillStatsSummary{
						Skill:    skill,
						Category: getSkillCategory(skill),
					}
				}
				skillCounts[skill].Spawns++
			}

		case "session.completed":
			// Skip events outside the --days window
			if event.Timestamp < cutoffDays {
				continue
			}

			// Extract beads_id for deduplication
			var sessionBeadsID string
			if data := event.Data; data != nil {
				if b, ok := data["beads_id"].(string); ok && b != "" {
					sessionBeadsID = b
				}
			}

			// Deduplicate by beads_id (same as agent.completed)
			deduplicationKey := sessionBeadsID
			if sessionBeadsID == "" {
				// Fall back to session_id if no beads_id
				deduplicationKey = "session:" + event.SessionID
			}

			alreadyCounted := completedBeadsIDs[deduplicationKey]

			if !alreadyCounted {
				completedBeadsIDs[deduplicationKey] = true
				report.Summary.TotalCompletions++
				// Calculate duration if we have spawn time
				if spawnTime, ok := spawnTimes[event.SessionID]; ok {
					duration := float64(event.Timestamp-spawnTime) / 60.0 // minutes
					if duration > 0 && duration < 480 {                   // Sanity check: < 8 hours
						durations = append(durations, duration)
					}
				}
				// Update skill completions
				if skill, ok := spawnSkills[event.SessionID]; ok {
					if stats, exists := skillCounts[skill]; exists {
						stats.Completions++
					}
				}
			}

		case "agent.completed":
			// Skip events outside the --days window
			if event.Timestamp < cutoffDays {
				continue
			}

			// Extract beads_id, workspace, and flags for correlation
			var beadsID string
			var workspace string
			var sessionID string
			// Verification metrics
			var verificationPassed bool
			var wasForced bool
			var eventGatesBypassed []string
			var eventSkill string
			if data := event.Data; data != nil {
				if b, ok := data["beads_id"].(string); ok && b != "" {
					beadsID = b
					beadsCompletions[beadsID] = event.Timestamp
					// Find session with matching beads_id
					for sid, spawnBeadsID := range spawnBeadsIDs {
						if spawnBeadsID == beadsID {
							sessionID = sid
							break
						}
					}
				}
				if w, ok := data["workspace"].(string); ok && w != "" {
					workspace = w
					// For orchestrators, correlate via workspace if beads_id didn't work
					if sessionID == "" && workspace != "" {
						if sid, ok := workspaceToSession[workspace]; ok {
							sessionID = sid
						}
					}
				}
				// Extract verification metrics
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
				// Track force_reason for override stats
				if wasForced {
					if reason, ok := data["force_reason"].(string); ok && reason != "" {
						overrideReasons["force_complete|"+reason]++
					}
				}
			}

			// Deduplicate completions by beads_id:
			// Multiple agent.completed events can exist for the same beads_id (e.g., from retries).
			// We only count unique completions, but always update to use the latest event's
			// timestamp for duration calculation.
			deduplicationKey := beadsID
			if beadsID == "" {
				// For completions without beads_id (e.g., some orchestrators), use workspace
				deduplicationKey = "ws:" + workspace
			}

			alreadyCounted := completedBeadsIDs[deduplicationKey]

			// Only count this completion if we haven't already counted this beads_id
			if !alreadyCounted {
				completedBeadsIDs[deduplicationKey] = true
				report.Summary.TotalCompletions++
				// Update skill completions
				if sessionID != "" {
					if skill, ok := spawnSkills[sessionID]; ok {
						if stats, exists := skillCounts[skill]; exists {
							stats.Completions++
						}
					}
				}
			}

			// Always calculate/update duration using the latest event for this beads_id
			// (even if we've already counted the completion, we want the latest timestamp)
			if sessionID != "" {
				if spawnTime, ok := spawnTimes[sessionID]; ok {
					duration := float64(event.Timestamp-spawnTime) / 60.0
					if duration > 0 && duration < 480 {
						// For deduplicated completions, we need to track durations per beads_id
						// and only add to the durations slice once (using the latest)
						// For simplicity, we add duration on first completion only
						// The "latest" logic would require more complex tracking
						if !alreadyCounted {
							durations = append(durations, duration)
						}
					}
				}
			}

			// Track verification metrics (only for unique completions)
			if !alreadyCounted {
				report.VerificationStats.TotalAttempts++
				if verificationPassed {
					report.VerificationStats.PassedFirstTry++
				}
				if wasForced {
					report.VerificationStats.Bypassed++
					// Track gates that were bypassed
					for _, gate := range eventGatesBypassed {
						gatesBypassed[gate]++
					}
				}
				// Track per-skill verification stats
				skill := eventSkill
				if skill == "" && sessionID != "" {
					// Fall back to skill from spawn if not in event
					skill = spawnSkills[sessionID]
				}
				if skill != "" {
					if _, exists := skillVerification[skill]; !exists {
						skillVerification[skill] = &SkillVerificationStats{Skill: skill}
					}
					skillVerification[skill].TotalAttempts++
					if verificationPassed {
						skillVerification[skill].PassedFirstTry++
					}
					if wasForced {
						skillVerification[skill].Bypassed++
					}
				}
			}

		case "agent.abandoned":
			// Skip events outside the --days window
			if event.Timestamp < cutoffDays {
				continue
			}

			// Extract beads_id and check if untracked
			var beadsID string
			var sessionID string
			if data := event.Data; data != nil {
				if b, ok := data["beads_id"].(string); ok && b != "" {
					beadsID = b
					for sid, spawnBeadsID := range spawnBeadsIDs {
						if spawnBeadsID == beadsID {
							sessionID = sid
							break
						}
					}
				}
			}

			// Deduplicate abandonments by beads_id:
			// orch abandon emits two agent.abandoned events per abandonment
			// (LifecycleManager + telemetry). We only count unique beads_ids.
			deduplicationKey := beadsID
			if beadsID == "" {
				// Fall back to timestamp-based key for events without beads_id
				deduplicationKey = fmt.Sprintf("ts:%d", event.Timestamp)
			}
			alreadyCounted := abandonedBeadsIDs[deduplicationKey]

			if !alreadyCounted {
				abandonedBeadsIDs[deduplicationKey] = true
				report.Summary.TotalAbandonments++
				// Update skill abandonments
				if sessionID != "" {
					if skill, ok := spawnSkills[sessionID]; ok {
						if stats, exists := skillCounts[skill]; exists {
							stats.Abandonments++
						}
					}
				}
			}

		case "daemon.spawn":
			// Skip events outside the --days window
			if event.Timestamp < cutoffDays {
				continue
			}
			report.DaemonStats.DaemonSpawns++

		case "session.auto_completed":
			// Skip events outside the --days window
			if event.Timestamp < cutoffDays {
				continue
			}
			report.DaemonStats.AutoCompletions++

		case "spawn.triage_bypassed":
			// Skip events outside the --days window
			if event.Timestamp < cutoffDays {
				continue
			}
			report.DaemonStats.TriageBypassed++
			spawnGateBypasses["triage"]++
			// Track override reason
			if data := event.Data; data != nil {
				if reason, ok := data["reason"].(string); ok && reason != "" {
					overrideReasons["triage_bypassed|"+reason]++
					spawnGateReasons["triage|"+reason]++
				}
			}

		case "spawn.hotspot_bypassed":
			// Skip events outside the --days window
			if event.Timestamp < cutoffDays {
				continue
			}
			spawnGateBypasses["hotspot"]++
			// Track override reason
			if data := event.Data; data != nil {
				reason, _ := data["reason"].(string)
				if reason != "" {
					overrideReasons["hotspot_bypassed|"+reason]++
					spawnGateReasons["hotspot|"+reason]++
				} else {
					overrideReasons["hotspot_bypassed|"]++
				}
			}

		case "spawn.verification_bypassed":
			// Skip events outside the --days window
			if event.Timestamp < cutoffDays {
				continue
			}
			spawnGateBypasses["verification"]++
			// Track override reason
			if data := event.Data; data != nil {
				if reason, ok := data["reason"].(string); ok && reason != "" {
					overrideReasons["verification_bypassed|"+reason]++
					spawnGateReasons["verification|"+reason]++
				}
			}

		case "agent.wait.complete":
			// Skip events outside the --days window
			if event.Timestamp < cutoffDays {
				continue
			}
			report.WaitStats.WaitCompleted++

		case "agent.wait.timeout":
			// Skip events outside the --days window
			if event.Timestamp < cutoffDays {
				continue
			}
			report.WaitStats.WaitTimeouts++

		case "session.orchestrator.started", "session.started":
			// Skip events outside the --days window
			if event.Timestamp < cutoffDays {
				continue
			}
			report.SessionStats.SessionsStarted++

		case "session.orchestrator.ended", "session.ended":
			// Skip events outside the --days window
			if event.Timestamp < cutoffDays {
				continue
			}
			report.SessionStats.SessionsEnded++

		case "verification.failed":
			// Skip events outside the --days window
			if event.Timestamp < cutoffDays {
				continue
			}
			// Extract gates_failed and skill from event data
			if data := event.Data; data != nil {
				// Track failed gates
				if gatesFailed, ok := data["gates_failed"].([]interface{}); ok {
					for _, g := range gatesFailed {
						if gate, ok := g.(string); ok {
							gateFailures[gate]++
						}
					}
				}
				// Track by skill for optional breakdown
				if skill, ok := data["skill"].(string); ok && skill != "" {
					if _, exists := skillVerification[skill]; !exists {
						skillVerification[skill] = &SkillVerificationStats{Skill: skill}
					}
					// Don't count here - failures don't directly map to attempts
					// Attempts are counted via agent.completed events
				}
			}

		case "verification.bypassed":
			// Skip events outside the --days window
			if event.Timestamp < cutoffDays {
				continue
			}
			report.VerificationStats.SkipBypassed++
			if data := event.Data; data != nil {
				if gate, ok := data["gate"].(string); ok && gate != "" {
					gatesBypassed[gate]++
					// Track reason for this gate bypass
					reason, _ := data["reason"].(string)
					bypassReasons[gate+"|"+reason]++
				}
			}

		case "verification.auto_skipped":
			// Skip events outside the --days window
			if event.Timestamp < cutoffDays {
				continue
			}
			report.VerificationStats.AutoSkipped++
			if data := event.Data; data != nil {
				if gate, ok := data["gate"].(string); ok && gate != "" {
					gatesAutoSkipped[gate]++
				}
			}

		case "spawn.gate_decision":
			// Skip events outside the --days window
			if event.Timestamp < cutoffDays {
				continue
			}
			if data := event.Data; data != nil {
				gateName, _ := data["gate_name"].(string)
				decision, _ := data["decision"].(string)
				skill, _ := data["skill"].(string)
				beadsID, _ := data["beads_id"].(string)
				if gateName != "" && decision != "" {
					gateDecisionCounts[gateName+"|"+decision]++
					if decision == "block" && skill != "" {
						gateBlockedSkills[gateName+"|"+skill]++
					}
					// Track for gate effectiveness
					if beadsID != "" {
						gatedBeadsIDs[beadsID] = true
						if decision == "block" {
							blockedBeadsIDs[beadsID] = true
						}
					}
				}
			}

		case "daemon.architect_escalation":
			// Skip events outside the --days window
			if event.Timestamp < cutoffDays {
				continue
			}
			if data := event.Data; data != nil {
				issueID, _ := data["issue_id"].(string)
				escalated, _ := data["escalated"].(bool)
				if issueID != "" && escalated {
					architectEscalatedIDs[issueID] = true
				}
			}
		}
	}

	// Set EventsAnalyzed to events within the analysis window
	report.EventsAnalyzed = eventsInWindow

	// Calculate escape hatch stats (multi-window: total, 7d, 30d)
	// Also calculate breakdown by account
	accountTotals := make(map[string]*AccountSpawnBreakdown)
	for _, eh := range escapeHatchSpawns {
		report.EscapeHatchStats.TotalSpawns++
		if eh.timestamp >= cutoff7d {
			report.EscapeHatchStats.Last7DaySpawns++
		}
		if eh.timestamp >= cutoff30d {
			report.EscapeHatchStats.Last30DaySpawns++
		}

		// Track by account (use "unknown" for spawns without account info)
		acct := eh.account
		if acct == "" {
			acct = "unknown"
		}
		if _, exists := accountTotals[acct]; !exists {
			accountTotals[acct] = &AccountSpawnBreakdown{Account: acct}
		}
		accountTotals[acct].TotalSpawns++
		if eh.timestamp >= cutoff7d {
			accountTotals[acct].Last7Days++
		}
		if eh.timestamp >= cutoff30d {
			accountTotals[acct].Last30Days++
		}
	}

	// Convert account map to slice and sort by total spawns descending
	for _, breakdown := range accountTotals {
		report.EscapeHatchStats.ByAccount = append(report.EscapeHatchStats.ByAccount, *breakdown)
	}
	sort.Slice(report.EscapeHatchStats.ByAccount, func(i, j int) bool {
		return report.EscapeHatchStats.ByAccount[i].TotalSpawns > report.EscapeHatchStats.ByAccount[j].TotalSpawns
	})

	// Calculate escape hatch rate (% of spawns within --days window using escape hatch)
	if report.Summary.TotalSpawns > 0 {
		report.EscapeHatchStats.EscapeHatchRate = float64(escapeHatchInWindow) / float64(report.Summary.TotalSpawns) * 100
	}

	// Calculate verification stats
	if report.VerificationStats.TotalAttempts > 0 {
		report.VerificationStats.PassRate = float64(report.VerificationStats.PassedFirstTry) / float64(report.VerificationStats.TotalAttempts) * 100
		report.VerificationStats.BypassRate = float64(report.VerificationStats.Bypassed) / float64(report.VerificationStats.TotalAttempts) * 100
	}

	// Build gate failure stats - combine failures, bypasses, and auto-skips
	// Collect all unique gates from all maps
	allGates := make(map[string]bool)
	for gate := range gateFailures {
		allGates[gate] = true
	}
	for gate := range gatesBypassed {
		allGates[gate] = true
	}
	for gate := range gatesAutoSkipped {
		allGates[gate] = true
	}
	for gate := range allGates {
		gateStats := GateFailureStats{
			Gate:          gate,
			FailCount:     gateFailures[gate],
			BypassCount:   gatesBypassed[gate],
			AutoSkipCount: gatesAutoSkipped[gate],
		}
		if report.VerificationStats.TotalAttempts > 0 {
			gateStats.FailRate = float64(gateStats.FailCount) / float64(report.VerificationStats.TotalAttempts) * 100
		}
		report.VerificationStats.FailuresByGate = append(report.VerificationStats.FailuresByGate, gateStats)
	}
	// Sort gates by fail count descending
	sort.Slice(report.VerificationStats.FailuresByGate, func(i, j int) bool {
		return report.VerificationStats.FailuresByGate[i].FailCount > report.VerificationStats.FailuresByGate[j].FailCount
	})

	// Build bypass reasons list from tracked gate|reason pairs
	for key, count := range bypassReasons {
		parts := strings.SplitN(key, "|", 2)
		gate := parts[0]
		reason := ""
		if len(parts) > 1 {
			reason = parts[1]
		}
		report.VerificationStats.BypassReasons = append(report.VerificationStats.BypassReasons, BypassReasonEntry{
			Gate:   gate,
			Reason: reason,
			Count:  count,
		})
	}
	// Sort bypass reasons by count descending
	sort.Slice(report.VerificationStats.BypassReasons, func(i, j int) bool {
		return report.VerificationStats.BypassReasons[i].Count > report.VerificationStats.BypassReasons[j].Count
	})

	// Build skill verification stats
	for _, sv := range skillVerification {
		if sv.TotalAttempts > 0 {
			sv.PassRate = float64(sv.PassedFirstTry) / float64(sv.TotalAttempts) * 100
		}
		report.VerificationStats.BySkill = append(report.VerificationStats.BySkill, *sv)
	}
	// Sort by total attempts descending
	sort.Slice(report.VerificationStats.BySkill, func(i, j int) bool {
		return report.VerificationStats.BySkill[i].TotalAttempts > report.VerificationStats.BySkill[j].TotalAttempts
	})

	// Build spawn gate stats from tracked bypasses
	for _, count := range spawnGateBypasses {
		report.SpawnGateStats.TotalBypasses += count
	}
	report.SpawnGateStats.TotalSpawns = report.Summary.TotalSpawns
	if report.SpawnGateStats.TotalSpawns > 0 {
		report.SpawnGateStats.BypassRate = float64(report.SpawnGateStats.TotalBypasses) / float64(report.SpawnGateStats.TotalSpawns) * 100
	}
	for gate, count := range spawnGateBypasses {
		entry := SpawnGateEntry{
			Gate:     gate,
			Bypassed: count,
		}
		if report.SpawnGateStats.TotalSpawns > 0 {
			entry.BypassRate = float64(count) / float64(report.SpawnGateStats.TotalSpawns) * 100
		}
		// Flag as miscalibrated if >50% of spawns bypass this gate
		if entry.BypassRate > 50 {
			entry.Miscalibrated = true
		}
		report.SpawnGateStats.ByGate = append(report.SpawnGateStats.ByGate, entry)
	}
	sort.Slice(report.SpawnGateStats.ByGate, func(i, j int) bool {
		return report.SpawnGateStats.ByGate[i].Bypassed > report.SpawnGateStats.ByGate[j].Bypassed
	})
	// Build spawn gate top reasons
	for key, count := range spawnGateReasons {
		parts := strings.SplitN(key, "|", 2)
		gate := parts[0]
		reason := ""
		if len(parts) > 1 {
			reason = parts[1]
		}
		if reason != "" {
			report.SpawnGateStats.TopReasons = append(report.SpawnGateStats.TopReasons, SpawnGateReasonEntry{
				Gate:   gate,
				Reason: reason,
				Count:  count,
			})
		}
	}
	sort.Slice(report.SpawnGateStats.TopReasons, func(i, j int) bool {
		return report.SpawnGateStats.TopReasons[i].Count > report.SpawnGateStats.TopReasons[j].Count
	})

	// Build override stats from tracked override reasons
	overrideByType := make(map[string]map[string]int) // type -> reason -> count
	for key, count := range overrideReasons {
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
		report.OverrideStats.TotalOverrides += count
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
		report.OverrideStats.ByType = append(report.OverrideStats.ByType, entry)
	}
	sort.Slice(report.OverrideStats.ByType, func(i, j int) bool {
		return report.OverrideStats.ByType[i].Count > report.OverrideStats.ByType[j].Count
	})
	// Build flattened top reasons list
	for _, entry := range report.OverrideStats.ByType {
		for _, reason := range entry.Reasons {
			report.OverrideStats.TopReasons = append(report.OverrideStats.TopReasons, reason)
		}
	}
	sort.Slice(report.OverrideStats.TopReasons, func(i, j int) bool {
		return report.OverrideStats.TopReasons[i].Count > report.OverrideStats.TopReasons[j].Count
	})

	// Calculate rates
	if report.Summary.TotalSpawns > 0 {
		report.Summary.CompletionRate = float64(report.Summary.TotalCompletions) / float64(report.Summary.TotalSpawns) * 100
		report.Summary.AbandonmentRate = float64(report.Summary.TotalAbandonments) / float64(report.Summary.TotalSpawns) * 100
		report.DaemonStats.DaemonSpawnRate = float64(report.DaemonStats.DaemonSpawns) / float64(report.Summary.TotalSpawns) * 100
	}

	// Calculate average duration
	if len(durations) > 0 {
		var total float64
		for _, d := range durations {
			total += d
		}
		report.Summary.AvgDurationMinutes = total / float64(len(durations))
	}

	// Calculate wait timeout rate
	totalWaits := report.WaitStats.WaitCompleted + report.WaitStats.WaitTimeouts
	if totalWaits > 0 {
		report.WaitStats.TimeoutRate = float64(report.WaitStats.WaitTimeouts) / float64(totalWaits) * 100
	}

	// Calculate active sessions
	report.SessionStats.ActiveSessions = report.SessionStats.SessionsStarted - report.SessionStats.SessionsEnded

	// Calculate per-skill completion rates and aggregate by category
	for _, stats := range skillCounts {
		if stats.Spawns > 0 {
			stats.CompletionRate = float64(stats.Completions) / float64(stats.Spawns) * 100
		}
		report.SkillStats = append(report.SkillStats, *stats)

		// Aggregate by category
		if stats.Category == TaskSkill {
			report.Summary.TaskSpawns += stats.Spawns
			report.Summary.TaskCompletions += stats.Completions
		} else if stats.Category == CoordinationSkill {
			report.Summary.CoordinationSpawns += stats.Spawns
			report.Summary.CoordinationCompletions += stats.Completions
		}
	}

	// Calculate task skill completion rate
	if report.Summary.TaskSpawns > 0 {
		report.Summary.TaskCompletionRate = float64(report.Summary.TaskCompletions) / float64(report.Summary.TaskSpawns) * 100
	}

	// Calculate coordination skill completion rate
	if report.Summary.CoordinationSpawns > 0 {
		report.Summary.CoordinationCompletionRate = float64(report.Summary.CoordinationCompletions) / float64(report.Summary.CoordinationSpawns) * 100
	}

	// Sort skills by spawn count descending
	sort.Slice(report.SkillStats, func(i, j int) bool {
		return report.SkillStats[i].Spawns > report.SkillStats[j].Spawns
	})

	// Build gate decision stats from tracked gate_decision events
	gateBlocks := make(map[string]int)
	gateBypasses := make(map[string]int)
	gateAllows := make(map[string]int)
	for key, count := range gateDecisionCounts {
		parts := strings.SplitN(key, "|", 2)
		gateName := parts[0]
		decision := ""
		if len(parts) > 1 {
			decision = parts[1]
		}
		report.GateDecisionStats.TotalDecisions += count
		if decision == "block" {
			report.GateDecisionStats.TotalBlocks += count
			gateBlocks[gateName] += count
		} else if decision == "bypass" {
			report.GateDecisionStats.TotalBypasses += count
			gateBypasses[gateName] += count
		} else if decision == "allow" {
			report.GateDecisionStats.TotalAllows += count
			gateAllows[gateName] += count
		}
	}
	// Build per-gate entries
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
		report.GateDecisionStats.ByGate = append(report.GateDecisionStats.ByGate, GateDecisionEntry{
			Gate:     g,
			Blocks:   gateBlocks[g],
			Bypasses: gateBypasses[g],
			Allows:   gateAllows[g],
		})
	}
	sort.Slice(report.GateDecisionStats.ByGate, func(i, j int) bool {
		totalI := report.GateDecisionStats.ByGate[i].Blocks + report.GateDecisionStats.ByGate[i].Bypasses + report.GateDecisionStats.ByGate[i].Allows
		totalJ := report.GateDecisionStats.ByGate[j].Blocks + report.GateDecisionStats.ByGate[j].Bypasses + report.GateDecisionStats.ByGate[j].Allows
		return totalI > totalJ
	})
	// Build top blocked skills
	for key, count := range gateBlockedSkills {
		parts := strings.SplitN(key, "|", 2)
		gateName := parts[0]
		skill := ""
		if len(parts) > 1 {
			skill = parts[1]
		}
		report.GateDecisionStats.TopBlockedSkills = append(report.GateDecisionStats.TopBlockedSkills, GateSkillEntry{
			Gate:  gateName,
			Skill: skill,
			Count: count,
		})
	}
	sort.Slice(report.GateDecisionStats.TopBlockedSkills, func(i, j int) bool {
		return report.GateDecisionStats.TopBlockedSkills[i].Count > report.GateDecisionStats.TopBlockedSkills[j].Count
	})

	// Build gate effectiveness stats by correlating gate decisions with outcomes
	if len(gatedBeadsIDs) > 0 || len(architectEscalatedIDs) > 0 {
		ge := &report.GateEffectivenessStats
		ge.TotalEvaluations = report.GateDecisionStats.TotalDecisions
		ge.TotalBlocks = report.GateDecisionStats.TotalBlocks
		ge.TotalBypasses = report.GateDecisionStats.TotalBypasses
		ge.TotalAllows = ge.TotalEvaluations - ge.TotalBlocks - ge.TotalBypasses
		if ge.TotalEvaluations > 0 {
			ge.BlockRate = float64(ge.TotalBlocks) / float64(ge.TotalEvaluations) * 100
		}

		// Architect escalation count
		ge.ArchitectEscalations = len(architectEscalatedIDs)

		// Blocked outcome correlation
		for beadsID := range blockedBeadsIDs {
			if architectEscalatedIDs[beadsID] {
				ge.BlockedOutcomes.EscalatedToArchitect++
			}
			if completedBeadsIDs[beadsID] {
				ge.BlockedOutcomes.EventuallyCompleted++
			} else if !abandonedBeadsIDs[beadsID] {
				ge.BlockedOutcomes.StillPending++
			}
		}

		// Quality metrics: iterate all spawns and classify as gated vs ungated
		var gatedDurations, ungatedDurations []float64
		for sid, spawnTime := range spawnTimes {
			// Only consider spawns within the analysis window
			if spawnTime < cutoffDays {
				continue
			}
			beadsID := spawnBeadsIDs[sid]
			if beadsID == "" {
				continue
			}

			isGated := gatedBeadsIDs[beadsID]
			var metrics *QualityMetrics
			if isGated {
				metrics = &ge.GatedCompletion
			} else {
				metrics = &ge.UngatedCompletion
			}
			metrics.TotalSpawns++

			if completedBeadsIDs[beadsID] {
				metrics.Completions++
				// Check verification pass from completion events
				// We need to find the completion event for this beads_id
				for _, event := range events {
					if event.Type != "agent.completed" || event.Timestamp < cutoffDays {
						continue
					}
					if data := event.Data; data != nil {
						if b, ok := data["beads_id"].(string); ok && b == beadsID {
							if vp, ok := data["verification_passed"].(bool); ok && vp {
								metrics.VerificationPassed++
							}
							// Calculate duration
							duration := float64(event.Timestamp-spawnTime) / 60.0
							if duration > 0 && duration < 480 {
								if isGated {
									gatedDurations = append(gatedDurations, duration)
								} else {
									ungatedDurations = append(ungatedDurations, duration)
								}
							}
							break // Only count first completion event per beads_id
						}
					}
				}
			} else if abandonedBeadsIDs[beadsID] {
				metrics.Abandonments++
			}
		}

		// Calculate rates
		for _, metrics := range []*QualityMetrics{&ge.GatedCompletion, &ge.UngatedCompletion} {
			if metrics.TotalSpawns > 0 {
				metrics.CompletionRate = float64(metrics.Completions) / float64(metrics.TotalSpawns) * 100
			}
			if metrics.Completions > 0 {
				metrics.VerificationRate = float64(metrics.VerificationPassed) / float64(metrics.Completions) * 100
			}
		}

		// Calculate average durations
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

	// Read coaching metrics
	home, err := os.UserHomeDir()
	if err == nil {
		coachingPath := filepath.Join(home, ".orch", "coaching-metrics.jsonl")
		since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
		coachingMetrics, err := coaching.ReadMetricsSince(coachingPath, since)
		if err == nil && len(coachingMetrics) > 0 {
			report.CoachingStats = coaching.AggregateByType(coachingMetrics, since)
		}
	}

	return report
}
