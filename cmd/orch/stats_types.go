// stats_types.go - Type definitions for stats aggregation
package main

import "github.com/dylan-conlin/orch-go/pkg/coaching"

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
	SpawnGateStats    SpawnGateStats                   `json:"spawn_gate_stats,omitempty"`
	OverrideStats     OverrideStats                    `json:"override_stats,omitempty"`
	GateDecisionStats      GateDecisionStats               `json:"gate_decision_stats,omitempty"`
	GateEffectivenessStats GateEffectivenessStats          `json:"gate_effectiveness_stats,omitempty"`
	SkillInferenceStats    SkillInferenceStats              `json:"skill_inference_stats,omitempty"`
	CoachingStats          map[string]coaching.MetricSummary `json:"coaching_stats,omitempty"`
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

// SpawnGateStats tracks bypass frequency across all spawn-level gates.
// High bypass rate for a gate signals miscalibration (gate too strict → operators routinely bypass).
type SpawnGateStats struct {
	TotalBypasses int              `json:"total_bypasses"`   // Total spawn gate bypasses across all gates
	TotalSpawns   int              `json:"total_spawns"`     // Total spawns in window (for rate calculation)
	BypassRate    float64          `json:"bypass_rate"`      // % of spawns with at least one gate bypassed
	ByGate        []SpawnGateEntry `json:"by_gate,omitempty"`
	TopReasons    []SpawnGateReasonEntry `json:"top_reasons,omitempty"`
}

// SpawnGateEntry tracks bypass metrics for a single spawn gate.
type SpawnGateEntry struct {
	Gate          string  `json:"gate"`           // "triage", "hotspot", "verification"
	Bypassed      int     `json:"bypassed"`       // Times this gate was bypassed
	BypassRate    float64 `json:"bypass_rate"`    // % of spawns that bypassed this gate
	Miscalibrated bool    `json:"miscalibrated"`  // True if bypass rate > 50% (signal for review)
}

// SpawnGateReasonEntry tracks a reason given for a spawn gate bypass.
type SpawnGateReasonEntry struct {
	Gate   string `json:"gate"`
	Reason string `json:"reason"`
	Count  int    `json:"count"`
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

// GateDecisionStats tracks spawn.gate_decision events aggregated by gate and decision.
type GateDecisionStats struct {
	TotalDecisions int                    `json:"total_decisions"`
	TotalBlocks    int                    `json:"total_blocks"`
	TotalBypasses  int                    `json:"total_bypasses"`
	TotalAllows    int                    `json:"total_allows"`
	ByGate         []GateDecisionEntry    `json:"by_gate,omitempty"`
	TopBlockedSkills []GateSkillEntry     `json:"top_blocked_skills,omitempty"`
}

// GateEffectivenessStats correlates gate decisions with agent outcomes.
// Answers: "Do gates improve quality?" by comparing gated vs ungated work.
type GateEffectivenessStats struct {
	TotalEvaluations int     `json:"total_evaluations"`
	TotalBlocks      int     `json:"total_blocks"`
	TotalBypasses    int     `json:"total_bypasses"`
	TotalAllows      int     `json:"total_allows"`
	BlockRate        float64 `json:"block_rate"`

	// Outcome correlation for blocked work
	BlockedOutcomes BlockedOutcomeStats `json:"blocked_outcomes"`

	// Quality comparison: gated (went through a gate decision) vs ungated (no gate event)
	GatedCompletion   QualityMetrics `json:"gated_completion"`
	UngatedCompletion QualityMetrics `json:"ungated_completion"`

	// Architect escalation stats (daemon.architect_escalation)
	ArchitectEscalations int `json:"architect_escalations"`
}

// BlockedOutcomeStats tracks what happened to work blocked by gates.
type BlockedOutcomeStats struct {
	EscalatedToArchitect int `json:"escalated_to_architect"` // Redirected via architect
	EventuallyCompleted  int `json:"eventually_completed"`   // Completed after redirect
	StillPending         int `json:"still_pending"`          // No completion event found
}

// QualityMetrics tracks completion quality for a cohort of spawns.
type QualityMetrics struct {
	TotalSpawns        int     `json:"total_spawns"`
	Completions        int     `json:"completions"`
	Abandonments       int     `json:"abandonments"`
	CompletionRate     float64 `json:"completion_rate"`
	VerificationPassed int     `json:"verification_passed"`
	VerificationRate   float64 `json:"verification_rate"` // % of completions that passed verification
	AvgDurationMinutes float64 `json:"avg_duration_minutes,omitempty"`
}

// GateDecisionEntry tracks block/bypass/allow counts for a single gate.
type GateDecisionEntry struct {
	Gate     string `json:"gate"`
	Blocks   int    `json:"blocks"`
	Bypasses int    `json:"bypasses"`
	Allows   int    `json:"allows"`
}

// GateSkillEntry tracks how often a skill was blocked by a specific gate.
type GateSkillEntry struct {
	Gate  string `json:"gate"`
	Skill string `json:"skill"`
	Count int    `json:"count"`
}

// escapeHatchSpawn tracks a single escape hatch spawn for multi-window analysis.
type escapeHatchSpawn struct {
	timestamp int64
	account   string
}

// SkillInferenceStats tracks accuracy of daemon skill inference.
// Correlates spawn.skill_inferred events with agent outcomes to measure
// whether inferred skills lead to successful completions.
type SkillInferenceStats struct {
	TotalInferences int                         `json:"total_inferences"`
	Completed       int                         `json:"completed"`       // Inferred spawns that completed
	Abandoned       int                         `json:"abandoned"`       // Inferred spawns that were abandoned
	SuccessRate     float64                     `json:"success_rate"`    // % completed / (completed + abandoned)
	ByMethod        []InferenceMethodStats      `json:"by_method,omitempty"`
	BySkill         []InferenceSkillStats       `json:"by_skill,omitempty"`
}

// InferenceMethodStats tracks outcomes by inference method.
type InferenceMethodStats struct {
	Method      string  `json:"method"`       // "label", "title", "description", "type"
	Inferences  int     `json:"inferences"`
	Completed   int     `json:"completed"`
	Abandoned   int     `json:"abandoned"`
	SuccessRate float64 `json:"success_rate"`
}

// InferenceSkillStats tracks inference outcomes per inferred skill.
type InferenceSkillStats struct {
	Skill       string  `json:"skill"`
	Inferences  int     `json:"inferences"`
	Completed   int     `json:"completed"`
	Abandoned   int     `json:"abandoned"`
	SuccessRate float64 `json:"success_rate"`
}

// GateAccuracyBaseline is a point-in-time snapshot of gate accuracy metrics.
// Used for prospective measurement: compare baselines over time to determine
// whether gates improve agent quality.
type GateAccuracyBaseline struct {
	SnapshotTime             string  `json:"snapshot_time"`
	DaysAnalyzed             int     `json:"days_analyzed"`
	TotalSpawns              int     `json:"total_spawns"`
	TotalCompletions         int     `json:"total_completions"`
	GatedSpawns              int     `json:"gated_spawns"`
	UngatedSpawns            int     `json:"ungated_spawns"`
	GateDecisions            int     `json:"gate_decisions"`
	TotalBlocks              int     `json:"total_blocks"`
	TotalBypasses            int     `json:"total_bypasses"`
	GatedCompletionRate      float64 `json:"gated_completion_rate"`
	UngatedCompletionRate    float64 `json:"ungated_completion_rate"`
	GatedVerificationRate    float64 `json:"gated_verification_rate"`
	UngatedVerificationRate  float64 `json:"ungated_verification_rate"`
	GatedAvgDuration         float64 `json:"gated_avg_duration_minutes"`
	UngatedAvgDuration       float64 `json:"ungated_avg_duration_minutes"`
	SpawnGateBypassRate      float64 `json:"spawn_gate_bypass_rate"`
	VerificationFirstTryRate float64 `json:"verification_first_try_rate"`
}
