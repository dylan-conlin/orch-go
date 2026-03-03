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

// SkillCategory represents the type of work a skill does
type SkillCategory string

const (
	// TaskSkill represents skills that complete discrete tasks
	TaskSkill SkillCategory = "task"
	// CoordinationSkill represents skills that coordinate other agents (not meant to complete)
	CoordinationSkill SkillCategory = "coordination"
)

// coordinationSkills lists skills that are coordination roles, not completable tasks.
// These are excluded from the completion rate warning because they're interactive sessions
// designed to run until context exhaustion, not complete discrete tasks.
var coordinationSkills = map[string]bool{
	"orchestrator":      true,
	"meta-orchestrator": true,
}

// getSkillCategory returns the category of a skill
func getSkillCategory(skill string) SkillCategory {
	if coordinationSkills[skill] {
		return CoordinationSkill
	}
	return TaskSkill
}

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

// Event represents a parsed event from events.jsonl
type StatsEvent struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// StatsReport contains all aggregated statistics
type StatsReport struct {
	GeneratedAt       string                            `json:"generated_at"`
	AnalysisPeriod    string                            `json:"analysis_period"`
	DaysAnalyzed      int                               `json:"days_analyzed"`
	EventsAnalyzed    int                               `json:"events_analyzed"`
	Summary           StatsSummary                      `json:"summary"`
	SkillStats        []SkillStatsSummary               `json:"skill_stats"`
	DaemonStats       DaemonStatsSummary                `json:"daemon_stats"`
	WaitStats         WaitStatsSummary                  `json:"wait_stats,omitempty"`
	SessionStats      SessionStatsSummary               `json:"session_stats,omitempty"`
	EscapeHatchStats  EscapeHatchStats                  `json:"escape_hatch_stats,omitempty"`
	VerificationStats VerificationStats                 `json:"verification_stats,omitempty"`
	OverrideStats     OverrideStats                    `json:"override_stats,omitempty"`
	CoachingStats     map[string]coaching.MetricSummary `json:"coaching_stats,omitempty"`
}

// StatsSummary contains core metrics
type StatsSummary struct {
	TotalSpawns        int     `json:"total_spawns"`
	TotalCompletions   int     `json:"total_completions"`
	TotalAbandonments  int     `json:"total_abandonments"`
	CompletionRate     float64 `json:"completion_rate"`
	AbandonmentRate    float64 `json:"abandonment_rate"`
	AvgDurationMinutes float64 `json:"avg_duration_minutes,omitempty"`
	// Task skill specific metrics (excludes coordination skills like orchestrator)
	TaskSpawns         int     `json:"task_spawns"`
	TaskCompletions    int     `json:"task_completions"`
	TaskCompletionRate float64 `json:"task_completion_rate"`
	// Coordination skill metrics (orchestrator, meta-orchestrator)
	CoordinationSpawns         int     `json:"coordination_spawns"`
	CoordinationCompletions    int     `json:"coordination_completions"`
	CoordinationCompletionRate float64 `json:"coordination_completion_rate"`
}

// SkillStatsSummary contains per-skill metrics
type SkillStatsSummary struct {
	Skill          string        `json:"skill"`
	Category       SkillCategory `json:"category"`
	Spawns         int           `json:"spawns"`
	Completions    int           `json:"completions"`
	Abandonments   int           `json:"abandonments"`
	CompletionRate float64       `json:"completion_rate"`
}

// DaemonStatsSummary contains daemon-specific metrics
type DaemonStatsSummary struct {
	DaemonSpawns    int     `json:"daemon_spawns"`
	AutoCompletions int     `json:"auto_completions"`
	TriageBypassed  int     `json:"triage_bypassed"`
	DaemonSpawnRate float64 `json:"daemon_spawn_rate"`
}

// WaitStatsSummary contains wait operation metrics
type WaitStatsSummary struct {
	WaitCompleted int     `json:"wait_completed"`
	WaitTimeouts  int     `json:"wait_timeouts"`
	TimeoutRate   float64 `json:"timeout_rate"`
}

// SessionStatsSummary contains orchestrator session metrics
type SessionStatsSummary struct {
	SessionsStarted int `json:"sessions_started"`
	SessionsEnded   int `json:"sessions_ended"`
	ActiveSessions  int `json:"active_sessions"`
}

// EscapeHatchStats tracks escape hatch spawn usage (--backend claude)
// Escape hatch provides resilience when OpenCode server is unstable
type EscapeHatchStats struct {
	TotalSpawns     int                     `json:"total_spawns"`      // All-time escape hatch spawns
	Last7DaySpawns  int                     `json:"last_7d_spawns"`    // Last 7 days
	Last30DaySpawns int                     `json:"last_30d_spawns"`   // Last 30 days
	ByAccount       []AccountSpawnBreakdown `json:"by_account"`        // Breakdown by Claude Max account
	EscapeHatchRate float64                 `json:"escape_hatch_rate"` // % of spawns using escape hatch (in analysis window)
}

// VerificationStats tracks completion verification metrics
// Enables identifying miscalibrated gates (high fail + high force = false positive pattern)
type VerificationStats struct {
	TotalAttempts  int                      `json:"total_attempts"`              // Total completion attempts
	PassedFirstTry int                      `json:"passed_first_try"`            // Passed verification on first try
	Bypassed       int                      `json:"bypassed"`                    // Used --force to bypass failures
	SkipBypassed   int                      `json:"skip_bypassed"`               // Bypassed via --skip-* flags
	AutoSkipped    int                      `json:"auto_skipped"`                // Auto-skipped by skill-class/file exemption
	PassRate       float64                  `json:"pass_rate"`                   // % passed first try
	BypassRate     float64                  `json:"bypass_rate"`                 // % bypassed with --force
	FailuresByGate []GateFailureStats       `json:"failures_by_gate"`            // Breakdown by gate type
	BySkill        []SkillVerificationStats `json:"by_skill,omitempty"`          // Optional: breakdown by skill
	BypassReasons  []BypassReasonEntry      `json:"bypass_reasons,omitempty"`    // Reasons given for --skip-* bypasses
}

// GateFailureStats tracks failure count for a specific verification gate
type GateFailureStats struct {
	Gate          string  `json:"gate"`            // Gate name (test_evidence, git_diff, visual_verification, phase_complete)
	FailCount     int     `json:"fail_count"`      // Times this gate failed
	BypassCount   int     `json:"bypass_count"`    // Times this gate was bypassed (--force or --skip-*)
	AutoSkipCount int     `json:"auto_skip_count"` // Times this gate was auto-skipped by exemption
	FailRate      float64 `json:"fail_rate"`       // % of attempts that failed this gate
}

// BypassReasonEntry tracks a specific gate+reason combination for --skip-* bypasses
type BypassReasonEntry struct {
	Gate   string `json:"gate"`
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

// SkillVerificationStats tracks verification metrics per skill
type SkillVerificationStats struct {
	Skill          string  `json:"skill"`
	TotalAttempts  int     `json:"total_attempts"`
	PassedFirstTry int     `json:"passed_first_try"`
	Bypassed       int     `json:"bypassed"`
	PassRate       float64 `json:"pass_rate"`
}

// OverrideStats tracks reasons given for safety-override flags
// Surfaces patterns in why operators bypass gates (e.g., "daemon unreliable" → systemic issue vs "urgent fix")
type OverrideStats struct {
	TotalOverrides int                   `json:"total_overrides"`
	ByType         []OverrideTypeEntry   `json:"by_type,omitempty"`
	TopReasons     []OverrideReasonEntry `json:"top_reasons,omitempty"`
}

// OverrideTypeEntry tracks override count per type
type OverrideTypeEntry struct {
	Type    string                `json:"type"`    // e.g., "triage_bypassed", "force_complete", "hotspot_bypassed", "no_track"
	Count   int                   `json:"count"`
	Reasons []OverrideReasonEntry `json:"reasons,omitempty"`
}

// OverrideReasonEntry tracks a specific reason and its frequency
type OverrideReasonEntry struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

// AccountSpawnBreakdown tracks spawns per Claude Max account
type AccountSpawnBreakdown struct {
	Account     string `json:"account"`
	TotalSpawns int    `json:"total_spawns"`
	Last7Days   int    `json:"last_7d"`
	Last30Days  int    `json:"last_30d"`
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
	spawnTimes := make(map[string]int64)       // session_id -> timestamp
	spawnSkills := make(map[string]string)     // session_id -> skill
	spawnBeadsIDs := make(map[string]string)   // session_id -> beads_id
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

	// Track override reasons across all override types
	// key: "type|reason", value: count
	overrideReasons := make(map[string]int)

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
			// Track override reason
			if data := event.Data; data != nil {
				if reason, ok := data["reason"].(string); ok && reason != "" {
					overrideReasons["triage_bypassed|"+reason]++
				}
			}

		case "spawn.hotspot_bypassed":
			// Skip events outside the --days window
			if event.Timestamp < cutoffDays {
				continue
			}
			// Track override reason
			if data := event.Data; data != nil {
				reason, _ := data["reason"].(string)
				if reason != "" {
					overrideReasons["hotspot_bypassed|"+reason]++
				} else {
					overrideReasons["hotspot_bypassed|"]++
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

func outputStatsJSON(report *StatsReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func outputStatsText(report *StatsReport) error {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("📊 ORCHESTRATION STATISTICS")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("Period: %s  |  Events analyzed: %d\n", report.AnalysisPeriod, report.EventsAnalyzed)
	fmt.Println(strings.Repeat("-", 70))

	// Core metrics
	fmt.Println()
	fmt.Println("🎯 CORE METRICS")
	fmt.Printf("  Spawns:        %d\n", report.Summary.TotalSpawns)
	fmt.Printf("  Completions:   %d (%.1f%%)\n", report.Summary.TotalCompletions, report.Summary.CompletionRate)
	fmt.Printf("  Abandonments:  %d (%.1f%%)\n", report.Summary.TotalAbandonments, report.Summary.AbandonmentRate)
	if report.Summary.AvgDurationMinutes > 0 {
		fmt.Printf("  Avg Duration:  %.0f minutes\n", report.Summary.AvgDurationMinutes)
	}

	// Task vs Coordination breakdown
	fmt.Println()
	fmt.Println("📋 COMPLETION BY CATEGORY")
	fmt.Printf("  Task Skills:         %d/%d spawns (%.1f%%) ← main metric\n",
		report.Summary.TaskCompletions, report.Summary.TaskSpawns, report.Summary.TaskCompletionRate)
	fmt.Printf("  Coordination Skills: %d/%d spawns (%.1f%%) [interactive sessions]\n",
		report.Summary.CoordinationCompletions, report.Summary.CoordinationSpawns, report.Summary.CoordinationCompletionRate)

	// Daemon metrics
	fmt.Println()
	fmt.Println("🤖 DAEMON HEALTH")
	fmt.Printf("  Daemon spawns:    %d (%.1f%% of all spawns)\n", report.DaemonStats.DaemonSpawns, report.DaemonStats.DaemonSpawnRate)
	fmt.Printf("  Auto-completions: %d\n", report.DaemonStats.AutoCompletions)
	fmt.Printf("  Triage bypassed:  %d\n", report.DaemonStats.TriageBypassed)

	// Wait metrics (if any)
	if report.WaitStats.WaitCompleted > 0 || report.WaitStats.WaitTimeouts > 0 {
		fmt.Println()
		fmt.Println("⏱️  WAIT OPERATIONS")
		fmt.Printf("  Completed: %d\n", report.WaitStats.WaitCompleted)
		fmt.Printf("  Timeouts:  %d (%.1f%% timeout rate)\n", report.WaitStats.WaitTimeouts, report.WaitStats.TimeoutRate)
	}

	// Session metrics (if verbose or has activity)
	if statsVerbose || report.SessionStats.SessionsStarted > 0 {
		fmt.Println()
		fmt.Println("📝 ORCHESTRATOR SESSIONS")
		fmt.Printf("  Started:  %d\n", report.SessionStats.SessionsStarted)
		fmt.Printf("  Ended:    %d\n", report.SessionStats.SessionsEnded)
		fmt.Printf("  Active:   %d\n", report.SessionStats.ActiveSessions)
	}

	// Escape hatch metrics (if any escape hatch spawns exist)
	if report.EscapeHatchStats.TotalSpawns > 0 {
		fmt.Println()
		fmt.Println("🚪 ESCAPE HATCH (--backend claude)")
		fmt.Printf("  Total:     %d spawns (all time)\n", report.EscapeHatchStats.TotalSpawns)
		fmt.Printf("  Last 7d:   %d spawns\n", report.EscapeHatchStats.Last7DaySpawns)
		fmt.Printf("  Last 30d:  %d spawns\n", report.EscapeHatchStats.Last30DaySpawns)
		if report.EscapeHatchStats.EscapeHatchRate > 0 {
			fmt.Printf("  Rate:      %.1f%% of spawns (in analysis window)\n", report.EscapeHatchStats.EscapeHatchRate)
		}

		// Show account breakdown (if more than one account or verbose)
		if len(report.EscapeHatchStats.ByAccount) > 1 || statsVerbose {
			fmt.Println()
			fmt.Println("  By Account:")
			for _, acct := range report.EscapeHatchStats.ByAccount {
				if acct.Account == "unknown" {
					fmt.Printf("    %-35s %4d total (%d 7d, %d 30d)\n", "(no account info)", acct.TotalSpawns, acct.Last7Days, acct.Last30Days)
				} else {
					// Truncate long email addresses
					displayAcct := acct.Account
					if len(displayAcct) > 35 {
						displayAcct = displayAcct[:32] + "..."
					}
					fmt.Printf("    %-35s %4d total (%d 7d, %d 30d)\n", displayAcct, acct.TotalSpawns, acct.Last7Days, acct.Last30Days)
				}
			}
		}
	}

	// Skill breakdown
	if len(report.SkillStats) > 0 {
		fmt.Println()
		fmt.Println("🎭 SKILL BREAKDOWN")
		fmt.Println("  (C) = Coordination skill (excluded from completion rate warning)")
		fmt.Printf("  %-25s %8s %8s %8s %10s\n", "Skill", "Spawns", "Complete", "Abandon", "Rate")
		fmt.Println("  " + strings.Repeat("-", 62))

		// Show top 10 skills by default, all if verbose
		limit := 10
		if statsVerbose {
			limit = len(report.SkillStats)
		}

		for i, skill := range report.SkillStats {
			if i >= limit {
				remaining := len(report.SkillStats) - limit
				fmt.Printf("  ... and %d more skills (use --verbose to show all)\n", remaining)
				break
			}
			// Mark coordination skills with (C) indicator
			skillName := truncateSkill(skill.Skill, 22)
			if skill.Category == CoordinationSkill {
				skillName = skillName + " (C)"
			}
			fmt.Printf("  %-25s %8d %8d %8d %9.1f%%\n",
				skillName,
				skill.Spawns,
				skill.Completions,
				skill.Abandonments,
				skill.CompletionRate,
			)
		}
	}

	// Verification stats (if any completion attempts or bypass events exist)
	hasVerificationData := report.VerificationStats.TotalAttempts > 0 ||
		report.VerificationStats.SkipBypassed > 0 ||
		report.VerificationStats.AutoSkipped > 0
	if hasVerificationData {
		fmt.Println()
		fmt.Println("✅ VERIFICATION GATES")
		if report.VerificationStats.TotalAttempts > 0 {
			fmt.Printf("  Total attempts:     %d\n", report.VerificationStats.TotalAttempts)
			fmt.Printf("  Passed 1st try:     %d (%.1f%%)\n", report.VerificationStats.PassedFirstTry, report.VerificationStats.PassRate)
			fmt.Printf("  Bypassed (--force): %d (%.1f%%)\n", report.VerificationStats.Bypassed, report.VerificationStats.BypassRate)
		}
		if report.VerificationStats.SkipBypassed > 0 {
			fmt.Printf("  Skipped (--skip-*): %d gate bypass events\n", report.VerificationStats.SkipBypassed)
		}
		if report.VerificationStats.AutoSkipped > 0 {
			fmt.Printf("  Auto-skipped:       %d (skill-class/file exemptions)\n", report.VerificationStats.AutoSkipped)
		}

		// Gate breakdown (if there are any gate-level stats)
		if len(report.VerificationStats.FailuresByGate) > 0 {
			fmt.Println()
			fmt.Println("  Gate Breakdown:")
			fmt.Printf("  %-25s %8s %8s %10s %10s\n", "Gate", "Failed", "Bypassed", "AutoSkip", "Fail Rate")
			fmt.Println("  " + strings.Repeat("-", 65))
			for _, gate := range report.VerificationStats.FailuresByGate {
				fmt.Printf("  %-25s %8d %8d %10d %9.1f%%\n",
					gate.Gate,
					gate.FailCount,
					gate.BypassCount,
					gate.AutoSkipCount,
					gate.FailRate,
				)
			}
		}

		// Bypass reasons (if any --skip-* bypasses with reasons exist)
		if len(report.VerificationStats.BypassReasons) > 0 {
			fmt.Println()
			fmt.Println("  Bypass Reasons (--skip-*):")
			for _, br := range report.VerificationStats.BypassReasons {
				reason := br.Reason
				if reason == "" {
					reason = "(no reason)"
				}
				if len(reason) > 50 {
					reason = reason[:47] + "..."
				}
				fmt.Printf("    %-20s %dx  %s\n", br.Gate, br.Count, reason)
			}
		}

		// Skill breakdown (if verbose and there's skill-level data)
		if statsVerbose && len(report.VerificationStats.BySkill) > 0 {
			fmt.Println()
			fmt.Println("  By Skill:")
			fmt.Printf("  %-25s %8s %8s %8s %10s\n", "Skill", "Attempts", "Passed", "Bypassed", "Pass Rate")
			fmt.Println("  " + strings.Repeat("-", 62))
			for _, sv := range report.VerificationStats.BySkill {
				fmt.Printf("  %-25s %8d %8d %8d %9.1f%%\n",
					truncateSkill(sv.Skill, 22),
					sv.TotalAttempts,
					sv.PassedFirstTry,
					sv.Bypassed,
					sv.PassRate,
				)
			}
		}
	}

	// Override reasons (if any override events with reasons exist)
	if report.OverrideStats.TotalOverrides > 0 {
		fmt.Println()
		fmt.Println("🔓 OVERRIDE REASONS")
		fmt.Printf("  Total overrides with reasons: %d\n", report.OverrideStats.TotalOverrides)
		fmt.Println()
		for _, entry := range report.OverrideStats.ByType {
			fmt.Printf("  %s (%d):\n", entry.Type, entry.Count)
			if len(entry.Reasons) > 0 {
				for _, reason := range entry.Reasons {
					r := reason.Reason
					if len(r) > 55 {
						r = r[:52] + "..."
					}
					fmt.Printf("    %dx  %s\n", reason.Count, r)
				}
			} else {
				fmt.Println("    (no reasons recorded)")
			}
		}
	}

	// Behavioral health (coaching metrics)
	if len(report.CoachingStats) > 0 {
		fmt.Println()
		fmt.Println("🧠 BEHAVIORAL HEALTH (coaching metrics)")

		// Separate orchestrator and worker metrics
		orchestratorMetrics := []string{"frame_collapse", "completion_backlog", "behavioral_variation", "circular_pattern"}
		workerMetrics := []string{"tool_failure_rate", "context_usage", "session_timeout", "spawn_depth_exceeded"}

		// Display orchestrator metrics
		hasOrchestratorMetrics := false
		for _, metricType := range orchestratorMetrics {
			if _, exists := report.CoachingStats[metricType]; exists {
				hasOrchestratorMetrics = true
				break
			}
		}

		if hasOrchestratorMetrics {
			fmt.Println("  Orchestrator:")
			for _, metricType := range orchestratorMetrics {
				if summary, exists := report.CoachingStats[metricType]; exists {
					timeSince := time.Since(summary.LastSeen)
					var timeStr string
					if timeSince < time.Minute {
						timeStr = "just now"
					} else if timeSince < time.Hour {
						timeStr = fmt.Sprintf("%dm ago", int(timeSince.Minutes()))
					} else if timeSince < 24*time.Hour {
						timeStr = fmt.Sprintf("%dh ago", int(timeSince.Hours()))
					} else {
						timeStr = fmt.Sprintf("%dd ago", int(timeSince.Hours()/24))
					}

					// Add warning emoji for recent events
					warningEmoji := ""
					if timeSince < 30*time.Minute {
						warningEmoji = " ⚠️"
					} else if summary.Count == 0 {
						warningEmoji = " ✅"
					}

					fmt.Printf("    %-25s %d events (last: %s)%s\n",
						metricType+":", summary.Count, timeStr, warningEmoji)
				}
			}
		}

		// Display worker metrics
		hasWorkerMetrics := false
		for _, metricType := range workerMetrics {
			if _, exists := report.CoachingStats[metricType]; exists {
				hasWorkerMetrics = true
				break
			}
		}

		if hasWorkerMetrics {
			if hasOrchestratorMetrics {
				fmt.Println()
			}
			fmt.Println("  Workers:")
			for _, metricType := range workerMetrics {
				if summary, exists := report.CoachingStats[metricType]; exists {
					timeSince := time.Since(summary.LastSeen)
					var timeStr string
					if timeSince < time.Minute {
						timeStr = "just now"
					} else if timeSince < time.Hour {
						timeStr = fmt.Sprintf("%dm ago", int(timeSince.Minutes()))
					} else if timeSince < 24*time.Hour {
						timeStr = fmt.Sprintf("%dh ago", int(timeSince.Hours()))
					} else {
						timeStr = fmt.Sprintf("%dd ago", int(timeSince.Hours()/24))
					}

					fmt.Printf("    %-25s %d events (last: %s)\n",
						metricType+":", summary.Count, timeStr)
				}
			}
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))

	// Quick health assessment (based on task skill rate, not overall)
	// Coordination skills (orchestrator, meta-orchestrator) are interactive sessions,
	// not completable tasks, so they're excluded from the health check.
	if report.Summary.TaskSpawns > 0 && report.Summary.TaskCompletionRate < 80 {
		fmt.Println("⚠️  WARNING: Task skill completion rate below 80% - investigate failure patterns")
	} else if report.Summary.TaskSpawns > 0 && report.Summary.TaskCompletionRate >= 95 {
		fmt.Println("✅ HEALTHY: Task skill completion rate at 95%+")
	}

	// Verification health check
	if report.VerificationStats.TotalAttempts > 0 && report.VerificationStats.BypassRate > 50 {
		fmt.Println("⚠️  WARNING: >50% of completions bypassed verification - gates may be miscalibrated")
	}

	return nil
}

func truncateSkill(skill string, maxLen int) string {
	if len(skill) <= maxLen {
		return skill
	}
	return skill[:maxLen-3] + "..."
}
