// stats_cmd.go - Aggregate events.jsonl metrics for orchestration observability
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/action"
	"github.com/spf13/cobra"
)

var (
	statsDays             int
	statsJSONOutput       bool
	statsVerbose          bool
	statsIncludeUntracked bool
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

// isUntrackedSpawn returns true if the beads_id indicates an untracked spawn.
// Untracked spawns have beads_ids containing "untracked" (e.g., "orch-go-untracked-abc123").
// These are test/ad-hoc spawns that should be excluded from production metrics by default.
func isUntrackedSpawn(beadsID string) bool {
	return strings.Contains(beadsID, "untracked")
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

By default, untracked spawns (test/ad-hoc work via --no-track) are excluded
from completion rate calculations to show production metrics only.
Use --include-untracked to include them.

Examples:
  orch stats                    # Show last 7 days (tracked spawns only)
  orch stats --include-untracked  # Include test/ad-hoc spawns
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
	statsCmd.Flags().BoolVar(&statsIncludeUntracked, "include-untracked", false, "Include untracked spawns in completion rate calculation")

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
	GeneratedAt       string              `json:"generated_at"`
	AnalysisPeriod    string              `json:"analysis_period"`
	DaysAnalyzed      int                 `json:"days_analyzed"`
	EventsAnalyzed    int                 `json:"events_analyzed"`
	Summary           StatsSummary        `json:"summary"`
	SkillStats        []SkillStatsSummary `json:"skill_stats"`
	DaemonStats       DaemonStatsSummary  `json:"daemon_stats"`
	WaitStats         WaitStatsSummary    `json:"wait_stats,omitempty"`
	SessionStats      SessionStatsSummary `json:"session_stats,omitempty"`
	EscapeHatchStats  EscapeHatchStats    `json:"escape_hatch_stats,omitempty"`
	VerificationStats VerificationStats   `json:"verification_stats,omitempty"`
	AttemptStats      AttemptStats        `json:"attempt_stats,omitempty"`
	DiscoveredWork    DiscoveredWorkStats `json:"discovered_work_stats,omitempty"`
}

// DiscoveredWorkStats captures discovered-work issue creation behavior.
// It measures how often worker sessions create follow-up issues via bd create.
type DiscoveredWorkStats struct {
	WorkerSessions                  int     `json:"worker_sessions"`
	WorkerSessionsWithIssueCreation int     `json:"worker_sessions_with_issue_creation"`
	WorkerIssueCreationRate         float64 `json:"worker_issue_creation_rate"`
	WorkerIssuesCreated             int     `json:"worker_issues_created"`
	OrchestratorIssuesCreated       int     `json:"orchestrator_issues_created"`
	WorkerIssueShare                float64 `json:"worker_issue_share"`
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
	// Untracked spawn metrics (test/ad-hoc work excluded from production metrics)
	UntrackedSpawns      int  `json:"untracked_spawns"`
	UntrackedCompletions int  `json:"untracked_completions"`
	IncludesUntracked    bool `json:"includes_untracked"`
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
	TotalAttempts  int                      `json:"total_attempts"`     // Total completion attempts
	PassedFirstTry int                      `json:"passed_first_try"`   // Passed verification on first try
	Bypassed       int                      `json:"bypassed"`           // Used --force to bypass failures
	PassRate       float64                  `json:"pass_rate"`          // % passed first try
	BypassRate     float64                  `json:"bypass_rate"`        // % bypassed with --force
	FailuresByGate []GateFailureStats       `json:"failures_by_gate"`   // Breakdown by gate type
	BySkill        []SkillVerificationStats `json:"by_skill,omitempty"` // Optional: breakdown by skill
}

// GateFailureStats tracks failure count for a specific verification gate
type GateFailureStats struct {
	Gate        string  `json:"gate"`         // Gate name (test_evidence, git_diff, visual_verification, phase_complete)
	FailCount   int     `json:"fail_count"`   // Times this gate failed
	BypassCount int     `json:"bypass_count"` // Times this gate was bypassed
	FailRate    float64 `json:"fail_rate"`    // % of attempts that failed this gate
}

// SkillVerificationStats tracks verification metrics per skill
type SkillVerificationStats struct {
	Skill          string  `json:"skill"`
	TotalAttempts  int     `json:"total_attempts"`
	PassedFirstTry int     `json:"passed_first_try"`
	Bypassed       int     `json:"bypassed"`
	PassRate       float64 `json:"pass_rate"`
}

// AccountSpawnBreakdown tracks spawns per Claude Max account
type AccountSpawnBreakdown struct {
	Account     string `json:"account"`
	TotalSpawns int    `json:"total_spawns"`
	Last7Days   int    `json:"last_7d"`
	Last30Days  int    `json:"last_30d"`
}

// AttemptStats tracks attempt-related metrics across all issues.
// An "attempt" is a spawn for a beads issue. Multiple attempts occur when:
// - An issue is reopened after being closed
// - A worker fails/dies and the issue is retried
type AttemptStats struct {
	// ReopenedCount is the number of issues that were reopened in the analysis window.
	// Each reopen triggers a new attempt at completing the issue.
	ReopenedCount int `json:"reopened_count"`
	// MultiAttemptIssues is the count of unique issues that had more than one attempt.
	MultiAttemptIssues int `json:"multi_attempt_issues"`
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
	report := aggregateStats(events, statsDays, statsIncludeUntracked)

	// Enrich with discovered-work issue creation metrics from action-log.
	if actionEvents, err := loadActionEvents(); err == nil {
		report.DiscoveredWork = computeDiscoveredWorkStats(events, actionEvents, statsDays, statsIncludeUntracked)
	}

	// Output
	if statsJSONOutput {
		return outputStatsJSON(report)
	}
	return outputStatsText(report)
}

func loadActionEvents() ([]action.ActionEvent, error) {
	tracker, err := action.LoadTracker("")
	if err != nil {
		return nil, err
	}
	return tracker.Events, nil
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

// escapeHatchSpawn tracks a spawn that used the escape hatch (--backend claude)
type escapeHatchSpawn struct {
	timestamp int64
	account   string
}

// statsAggregator holds all intermediate tracking state needed during event processing.
// This separates the accumulation state from the final StatsReport output.
type statsAggregator struct {
	report           *StatsReport
	includeUntracked bool

	// Time window cutoffs
	cutoffDays int64 // --days window for main stats
	cutoff7d   int64 // 7 days for escape hatch
	cutoff30d  int64 // 30 days for escape hatch

	// Session tracking
	spawnTimes         map[string]int64  // session_id -> timestamp
	spawnSkills        map[string]string // session_id -> skill
	spawnBeadsIDs      map[string]string // session_id -> beads_id
	untrackedSessions  map[string]bool   // session_id -> true if untracked
	workspaceToSession map[string]string // workspace -> session_id (pseudo)

	// Completion tracking
	completedBeadsIDs map[string]bool // beads_id -> true if already counted
	durations         []float64

	// Skill tracking
	skillCounts map[string]*SkillStatsSummary

	// Escape hatch tracking (multi-window: total, 7d, 30d)
	escapeHatchSpawns   []escapeHatchSpawn
	escapeHatchInWindow int // count within --days window

	// Verification tracking
	gateFailures      map[string]int                     // gate -> failure count
	gatesBypassed     map[string]int                     // gate -> bypass count
	skillVerification map[string]*SkillVerificationStats // skill -> verification stats

	// Event counting
	eventsInWindow int
}

// newStatsAggregator creates a new aggregator with all tracking state initialized.
func newStatsAggregator(days int, includeUntracked bool) *statsAggregator {
	now := time.Now().Unix()
	report := &StatsReport{
		GeneratedAt:    time.Now().Format(time.RFC3339),
		AnalysisPeriod: fmt.Sprintf("Last %d days", days),
		DaysAnalyzed:   days,
		SkillStats:     []SkillStatsSummary{},
	}
	report.Summary.IncludesUntracked = includeUntracked

	return &statsAggregator{
		report:           report,
		includeUntracked: includeUntracked,

		cutoffDays: now - int64(days*86400),
		cutoff7d:   now - int64(7*86400),
		cutoff30d:  now - int64(30*86400),

		spawnTimes:         make(map[string]int64),
		spawnSkills:        make(map[string]string),
		spawnBeadsIDs:      make(map[string]string),
		untrackedSessions:  make(map[string]bool),
		workspaceToSession: make(map[string]string),

		completedBeadsIDs: make(map[string]bool),

		skillCounts: make(map[string]*SkillStatsSummary),

		gateFailures:      make(map[string]int),
		gatesBypassed:     make(map[string]int),
		skillVerification: make(map[string]*SkillVerificationStats),
	}
}

// findSessionByBeadsID looks up the session ID that was spawned with the given beads_id.
func (a *statsAggregator) findSessionByBeadsID(beadsID string) string {
	for sid, spawnBeadsID := range a.spawnBeadsIDs {
		if spawnBeadsID == beadsID {
			return sid
		}
	}
	return ""
}

// handleSpawned processes a session.spawned event.
// It tracks spawn metadata (always, regardless of time window) and counts spawns within the window.
func (a *statsAggregator) handleSpawned(event StatsEvent) {
	// Extract skill, beads_id, workspace, spawn_mode, and account from data
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

	// Track escape hatch spawns (spawn_mode = "claude")
	// This is tracked across all time windows for comprehensive metrics
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
	// For orchestrators (empty SessionID), use workspace as the key
	effectiveSessionID := event.SessionID
	if effectiveSessionID == "" && workspace != "" {
		effectiveSessionID = "ws:" + workspace // prefix to avoid collision
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

	// Check if this is an untracked spawn
	isUntracked := isUntrackedSpawn(beadsID)
	if isUntracked {
		a.untrackedSessions[effectiveSessionID] = true
		if event.Timestamp >= a.cutoffDays {
			a.report.Summary.UntrackedSpawns++
		}
	}

	// Skip events outside the --days window for main stats
	if event.Timestamp < a.cutoffDays {
		return
	}

	// Coordination skills (orchestrator, meta-orchestrator) are always counted
	// even when untracked, since they're interactive sessions not task work.
	isCoordinationSkill := coordinationSkills[skill]

	// Only count toward overall metrics if tracked OR includeUntracked OR coordination skill
	if !isUntracked || a.includeUntracked || isCoordinationSkill {
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
}

// handleSessionCompleted processes a session.completed event.
func (a *statsAggregator) handleSessionCompleted(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
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
		deduplicationKey = "session:" + event.SessionID
	}

	alreadyCounted := a.completedBeadsIDs[deduplicationKey]

	// Check if this is an untracked session
	isUntracked := a.untrackedSessions[event.SessionID] || isUntrackedSpawn(sessionBeadsID)
	if isUntracked && !alreadyCounted {
		a.report.Summary.UntrackedCompletions++
	}
	// Only count if tracked OR includeUntracked is set, AND not already counted
	if (!isUntracked || a.includeUntracked) && !alreadyCounted {
		a.completedBeadsIDs[deduplicationKey] = true
		a.report.Summary.TotalCompletions++
		// Calculate duration if we have spawn time
		if spawnTime, ok := a.spawnTimes[event.SessionID]; ok {
			duration := float64(event.Timestamp-spawnTime) / 60.0 // minutes
			if duration > 0 && duration < 480 {                   // Sanity check: < 8 hours
				a.durations = append(a.durations, duration)
			}
		}
		// Update skill completions
		if skill, ok := a.spawnSkills[event.SessionID]; ok {
			if stats, exists := a.skillCounts[skill]; exists {
				stats.Completions++
			}
		}
	}
}

// handleAgentCompleted processes an agent.completed event.
// This is the most complex handler due to orchestrator correlation, deduplication,
// duration tracking, and verification metrics.
func (a *statsAggregator) handleAgentCompleted(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}

	// Extract beads_id, workspace, and flags for correlation
	var beadsID, workspace, sessionID, eventSkill string
	var isOrchestrator, eventMarkedUntracked, verificationPassed, wasForced bool
	var eventGatesBypassed []string

	if data := event.Data; data != nil {
		if b, ok := data["beads_id"].(string); ok && b != "" {
			beadsID = b
			sessionID = a.findSessionByBeadsID(beadsID)
		}
		if orch, ok := data["orchestrator"].(bool); ok && orch {
			isOrchestrator = true
		}
		if unt, ok := data["untracked"].(bool); ok && unt {
			eventMarkedUntracked = true
		}
		if w, ok := data["workspace"].(string); ok && w != "" {
			workspace = w
			// For orchestrators, correlate via workspace if beads_id didn't work
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
	}

	// Check if untracked:
	// 1. Event explicitly marked untracked
	// 2. beads_id pattern contains "untracked"
	// 3. Session was marked untracked at spawn time
	// 4. Orchestrator completion that couldn't be correlated
	isUntracked := eventMarkedUntracked || isUntrackedSpawn(beadsID) || a.untrackedSessions[sessionID]
	if isOrchestrator && sessionID == "" && !isUntracked {
		isUntracked = true
	}

	// Deduplicate completions by beads_id
	deduplicationKey := beadsID
	if beadsID == "" {
		deduplicationKey = "ws:" + workspace
	}

	alreadyCounted := a.completedBeadsIDs[deduplicationKey]

	// Track untracked completions (don't double-count orchestrator completions)
	if isUntracked && !isOrchestrator && !alreadyCounted {
		a.report.Summary.UntrackedCompletions++
	}

	// Check if this is a coordination skill (via correlated session)
	var isCoordinationSkill bool
	if sessionID != "" {
		if skill, ok := a.spawnSkills[sessionID]; ok {
			isCoordinationSkill = coordinationSkills[skill]
		}
	}

	shouldCount := !isUntracked || a.includeUntracked || isCoordinationSkill

	// Count completion (deduplicated)
	if shouldCount && !alreadyCounted {
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

	// Calculate duration (only on first completion for this beads_id)
	if shouldCount && sessionID != "" && !alreadyCounted {
		if spawnTime, ok := a.spawnTimes[sessionID]; ok {
			duration := float64(event.Timestamp-spawnTime) / 60.0
			if duration > 0 && duration < 480 {
				a.durations = append(a.durations, duration)
			}
		}
	}

	// Track verification metrics (only for unique completions)
	if shouldCount && !alreadyCounted {
		a.trackVerification(eventSkill, sessionID, verificationPassed, wasForced, eventGatesBypassed)
	}
}

// trackVerification records verification metrics for a completion event.
func (a *statsAggregator) trackVerification(eventSkill, sessionID string, passed, forced bool, gatesBypassed []string) {
	a.report.VerificationStats.TotalAttempts++
	if passed {
		a.report.VerificationStats.PassedFirstTry++
	}
	if forced {
		a.report.VerificationStats.Bypassed++
		for _, gate := range gatesBypassed {
			a.gatesBypassed[gate]++
		}
	}

	// Track per-skill verification stats
	skill := eventSkill
	if skill == "" && sessionID != "" {
		skill = a.spawnSkills[sessionID]
	}
	if skill != "" {
		if _, exists := a.skillVerification[skill]; !exists {
			a.skillVerification[skill] = &SkillVerificationStats{Skill: skill}
		}
		a.skillVerification[skill].TotalAttempts++
		if passed {
			a.skillVerification[skill].PassedFirstTry++
		}
		if forced {
			a.skillVerification[skill].Bypassed++
		}
	}
}

// handleAbandoned processes an agent.abandoned event.
func (a *statsAggregator) handleAbandoned(event StatsEvent) {
	if event.Timestamp < a.cutoffDays {
		return
	}

	var beadsID, sessionID string
	if data := event.Data; data != nil {
		if b, ok := data["beads_id"].(string); ok && b != "" {
			beadsID = b
			sessionID = a.findSessionByBeadsID(beadsID)
		}
	}

	isUntracked := isUntrackedSpawn(beadsID) || a.untrackedSessions[sessionID]

	if !isUntracked || a.includeUntracked {
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

// handleVerificationFailed processes a verification.failed event.
func (a *statsAggregator) handleVerificationFailed(event StatsEvent) {
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
		// Track by skill for optional breakdown
		if skill, ok := data["skill"].(string); ok && skill != "" {
			if _, exists := a.skillVerification[skill]; !exists {
				a.skillVerification[skill] = &SkillVerificationStats{Skill: skill}
			}
		}
	}
}

// processEvents iterates through all events and dispatches to the appropriate handler.
func (a *statsAggregator) processEvents(events []StatsEvent) {
	for _, event := range events {
		if event.Timestamp >= a.cutoffDays {
			a.eventsInWindow++
		}

		switch event.Type {
		case "session.spawned":
			a.handleSpawned(event)
		case "session.completed":
			a.handleSessionCompleted(event)
		case "agent.completed":
			a.handleAgentCompleted(event)
		case "agent.abandoned":
			a.handleAbandoned(event)
		case "daemon.spawn":
			if event.Timestamp >= a.cutoffDays {
				a.report.DaemonStats.DaemonSpawns++
			}
		case "session.auto_completed":
			if event.Timestamp >= a.cutoffDays {
				a.report.DaemonStats.AutoCompletions++
			}
		case "spawn.triage_bypassed":
			if event.Timestamp >= a.cutoffDays {
				a.report.DaemonStats.TriageBypassed++
			}
		case "agent.wait.complete":
			if event.Timestamp >= a.cutoffDays {
				a.report.WaitStats.WaitCompleted++
			}
		case "agent.wait.timeout":
			if event.Timestamp >= a.cutoffDays {
				a.report.WaitStats.WaitTimeouts++
			}
		case "session.orchestrator.started", "session.started":
			if event.Timestamp >= a.cutoffDays {
				a.report.SessionStats.SessionsStarted++
			}
		case "session.orchestrator.ended", "session.ended":
			if event.Timestamp >= a.cutoffDays {
				a.report.SessionStats.SessionsEnded++
			}
		case "verification.failed":
			a.handleVerificationFailed(event)
		case "issue.reopened":
			if event.Timestamp >= a.cutoffDays {
				a.report.AttemptStats.ReopenedCount++
			}
		}
	}
}

// calculateEscapeHatchStats computes escape hatch metrics across multiple time windows.
func (a *statsAggregator) calculateEscapeHatchStats() {
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

// calculateVerificationStats computes verification pass/bypass rates and gate breakdowns.
func (a *statsAggregator) calculateVerificationStats() {
	if a.report.VerificationStats.TotalAttempts > 0 {
		a.report.VerificationStats.PassRate = float64(a.report.VerificationStats.PassedFirstTry) / float64(a.report.VerificationStats.TotalAttempts) * 100
		a.report.VerificationStats.BypassRate = float64(a.report.VerificationStats.Bypassed) / float64(a.report.VerificationStats.TotalAttempts) * 100
	}

	// Build gate failure stats - combine failures and bypasses
	allGates := make(map[string]bool)
	for gate := range a.gateFailures {
		allGates[gate] = true
	}
	for gate := range a.gatesBypassed {
		allGates[gate] = true
	}
	for gate := range allGates {
		gateStats := GateFailureStats{
			Gate:        gate,
			FailCount:   a.gateFailures[gate],
			BypassCount: a.gatesBypassed[gate],
		}
		if a.report.VerificationStats.TotalAttempts > 0 {
			gateStats.FailRate = float64(gateStats.FailCount) / float64(a.report.VerificationStats.TotalAttempts) * 100
		}
		a.report.VerificationStats.FailuresByGate = append(a.report.VerificationStats.FailuresByGate, gateStats)
	}
	sort.Slice(a.report.VerificationStats.FailuresByGate, func(i, j int) bool {
		return a.report.VerificationStats.FailuresByGate[i].FailCount > a.report.VerificationStats.FailuresByGate[j].FailCount
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

// calculateSummaryRates computes completion, abandonment, daemon, duration, wait, and session rates.
func (a *statsAggregator) calculateSummaryRates() {
	if a.report.Summary.TotalSpawns > 0 {
		a.report.Summary.CompletionRate = float64(a.report.Summary.TotalCompletions) / float64(a.report.Summary.TotalSpawns) * 100
		a.report.Summary.AbandonmentRate = float64(a.report.Summary.TotalAbandonments) / float64(a.report.Summary.TotalSpawns) * 100
		a.report.DaemonStats.DaemonSpawnRate = float64(a.report.DaemonStats.DaemonSpawns) / float64(a.report.Summary.TotalSpawns) * 100
	}

	// Average duration
	if len(a.durations) > 0 {
		var total float64
		for _, d := range a.durations {
			total += d
		}
		a.report.Summary.AvgDurationMinutes = total / float64(len(a.durations))
	}

	// Wait timeout rate
	totalWaits := a.report.WaitStats.WaitCompleted + a.report.WaitStats.WaitTimeouts
	if totalWaits > 0 {
		a.report.WaitStats.TimeoutRate = float64(a.report.WaitStats.WaitTimeouts) / float64(totalWaits) * 100
	}

	// Active sessions
	a.report.SessionStats.ActiveSessions = a.report.SessionStats.SessionsStarted - a.report.SessionStats.SessionsEnded
}

// calculateSkillStats computes per-skill completion rates and aggregates by category.
func (a *statsAggregator) calculateSkillStats() {
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

func aggregateStats(events []StatsEvent, days int, includeUntracked bool) *StatsReport {
	agg := newStatsAggregator(days, includeUntracked)

	agg.processEvents(events)

	agg.report.EventsAnalyzed = agg.eventsInWindow

	agg.calculateEscapeHatchStats()
	agg.calculateVerificationStats()
	agg.calculateSummaryRates()
	agg.calculateSkillStats()

	return agg.report
}

var bdCreateCommandPattern = regexp.MustCompile(`(?i)(?:^|&&|\|\||;)\s*bd\s+create(?:\s|$)`)

func isBDCreateAction(target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	return bdCreateCommandPattern.MatchString(target)
}

func computeDiscoveredWorkStats(events []StatsEvent, actionEvents []action.ActionEvent, days int, includeUntracked bool) DiscoveredWorkStats {
	stats := DiscoveredWorkStats{}
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)

	workerSessions := make(map[string]bool)

	for _, event := range events {
		if event.Type != "session.spawned" {
			continue
		}
		if time.Unix(event.Timestamp, 0).Before(cutoff) {
			continue
		}

		sessionID := event.SessionID
		if sessionID == "" {
			continue
		}

		skill, _ := event.Data["skill"].(string)
		if skill == "" || coordinationSkills[skill] {
			continue
		}

		beadsID, _ := event.Data["beads_id"].(string)
		if isUntrackedSpawn(beadsID) && !includeUntracked {
			continue
		}

		workerSessions[sessionID] = true
	}

	workerCreatorSessions := make(map[string]bool)

	for _, event := range actionEvents {
		if event.Timestamp.Before(cutoff) {
			continue
		}
		if !strings.EqualFold(event.Tool, "bash") {
			continue
		}
		if event.Outcome != action.OutcomeSuccess {
			continue
		}
		if !isBDCreateAction(event.Target) {
			continue
		}

		if event.SessionID != "" && workerSessions[event.SessionID] {
			stats.WorkerIssuesCreated++
			workerCreatorSessions[event.SessionID] = true
			continue
		}

		// Any successful bd create action not attributable to a worker session is
		// counted as orchestrator-side issue creation.
		stats.OrchestratorIssuesCreated++
	}

	stats.WorkerSessions = len(workerSessions)
	stats.WorkerSessionsWithIssueCreation = len(workerCreatorSessions)

	if stats.WorkerSessions > 0 {
		stats.WorkerIssueCreationRate = float64(stats.WorkerSessionsWithIssueCreation) / float64(stats.WorkerSessions) * 100
	}

	totalIssueCreates := stats.WorkerIssuesCreated + stats.OrchestratorIssuesCreated
	if totalIssueCreates > 0 {
		stats.WorkerIssueShare = float64(stats.WorkerIssuesCreated) / float64(totalIssueCreates) * 100
	}

	return stats
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
	fmt.Printf("  Spawns:        %d", report.Summary.TotalSpawns)
	if !report.Summary.IncludesUntracked && report.Summary.UntrackedSpawns > 0 {
		fmt.Printf(" (excluding %d untracked)", report.Summary.UntrackedSpawns)
	}
	fmt.Println()
	fmt.Printf("  Completions:   %d (%.1f%%)\n", report.Summary.TotalCompletions, report.Summary.CompletionRate)
	fmt.Printf("  Abandonments:  %d (%.1f%%)\n", report.Summary.TotalAbandonments, report.Summary.AbandonmentRate)
	if report.Summary.AvgDurationMinutes > 0 {
		fmt.Printf("  Avg Duration:  %.0f minutes\n", report.Summary.AvgDurationMinutes)
	}

	if report.DiscoveredWork.WorkerSessions > 0 || report.DiscoveredWork.WorkerIssuesCreated > 0 || report.DiscoveredWork.OrchestratorIssuesCreated > 0 {
		fmt.Println()
		fmt.Println("🔍 DISCOVERED WORK CAPTURE")
		fmt.Printf("  Worker sessions creating issues: %d/%d (%.1f%%)\n",
			report.DiscoveredWork.WorkerSessionsWithIssueCreation,
			report.DiscoveredWork.WorkerSessions,
			report.DiscoveredWork.WorkerIssueCreationRate,
		)
		fmt.Printf("  Worker issues created:          %d\n", report.DiscoveredWork.WorkerIssuesCreated)
		fmt.Printf("  Orchestrator issues created:    %d\n", report.DiscoveredWork.OrchestratorIssuesCreated)
		fmt.Printf("  Worker share of issue creation: %.1f%%\n", report.DiscoveredWork.WorkerIssueShare)
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

	// Verification stats (if any completion attempts exist)
	if report.VerificationStats.TotalAttempts > 0 {
		fmt.Println()
		fmt.Println("✅ VERIFICATION GATES")
		fmt.Printf("  Total attempts:   %d\n", report.VerificationStats.TotalAttempts)
		fmt.Printf("  Passed 1st try:   %d (%.1f%%)\n", report.VerificationStats.PassedFirstTry, report.VerificationStats.PassRate)
		fmt.Printf("  Bypassed (--force): %d (%.1f%%)\n", report.VerificationStats.Bypassed, report.VerificationStats.BypassRate)

		// Gate breakdown (if there are failures)
		if len(report.VerificationStats.FailuresByGate) > 0 {
			fmt.Println()
			fmt.Println("  Gate Breakdown:")
			fmt.Printf("  %-25s %8s %8s %10s\n", "Gate", "Failed", "Bypassed", "Fail Rate")
			fmt.Println("  " + strings.Repeat("-", 55))
			for _, gate := range report.VerificationStats.FailuresByGate {
				fmt.Printf("  %-25s %8d %8d %9.1f%%\n",
					gate.Gate,
					gate.FailCount,
					gate.BypassCount,
					gate.FailRate,
				)
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

	// Attempt stats (if any reopens exist)
	if report.AttemptStats.ReopenedCount > 0 {
		fmt.Println()
		fmt.Println("🔄 ATTEMPT TRACKING")
		fmt.Printf("  Issues reopened:  %d\n", report.AttemptStats.ReopenedCount)
		if report.AttemptStats.MultiAttemptIssues > 0 {
			fmt.Printf("  Multi-attempt:    %d issues required >1 attempt\n", report.AttemptStats.MultiAttemptIssues)
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
