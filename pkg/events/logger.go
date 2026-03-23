// Package events provides event logging functionality for agent lifecycle events.
// Events are appended to ~/.orch/events.jsonl in JSONL format.
package events

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Event types for agent lifecycle tracking.
const (
	// EventTypeSessionSpawned indicates a new session was created.
	EventTypeSessionSpawned = "session.spawned"
	// EventTypeSessionCompleted indicates a session finished successfully.
	EventTypeSessionCompleted = "session.completed"
	// EventTypeSessionError indicates a session encountered an error.
	EventTypeSessionError = "session.error"
	// EventTypeSessionStatus indicates a session status change (busy/idle).
	EventTypeSessionStatus = "session.status"
	// EventTypeAutoCompleted indicates a session was auto-completed by the daemon.
	EventTypeAutoCompleted = "session.auto_completed"
	// EventTypeAgentCompleted indicates an agent was completed via orch complete.
	EventTypeAgentCompleted = "agent.completed"
	// EventTypeAgentReworked indicates a rework spawn was created for an issue.
	EventTypeAgentReworked = "agent.reworked"
	// EventTypeVerificationFailed indicates verification failed before user decides to --force or fix.
	EventTypeVerificationFailed = "verification.failed"
	// EventTypeServiceCrashed indicates a service crashed (PID changed unexpectedly).
	EventTypeServiceCrashed = "service.crashed"
	// EventTypeServiceRestarted indicates a service was automatically restarted after a crash.
	EventTypeServiceRestarted = "service.restarted"
	// EventTypeServiceStarted indicates a service started (first time seen).
	EventTypeServiceStarted = "service.started"
	// EventTypeVerificationBypassed indicates a verification gate was bypassed via --skip-* flag.
	EventTypeVerificationBypassed = "verification.bypassed"
	// EventTypeVerificationAutoSkipped indicates a verification gate was auto-skipped due to skill-class or file type exemption.
	EventTypeVerificationAutoSkipped = "verification.auto_skipped"
	// EventTypeAgentAbandoned indicates an agent was abandoned via orch abandon.
	EventTypeAgentAbandoned = "agent.abandoned"
	// EventTypeAgentAbandonedTelemetry is enriched telemetry for abandonments (skill, tokens, duration).
	// Separate from agent.abandoned to avoid double-counting in stats.
	EventTypeAgentAbandonedTelemetry = "agent.abandoned.telemetry"
	// EventTypeSpawnSkillInferred indicates a skill was inferred for an issue spawn.
	EventTypeSpawnSkillInferred = "spawn.skill_inferred"
	// EventTypeHotspotBypassed indicates a CRITICAL hotspot blocking gate was bypassed via --force-hotspot.
	EventTypeHotspotBypassed = "spawn.hotspot_bypassed"
	// EventTypeReviewTierEscalated indicates a review tier was automatically escalated based on completion signals.
	EventTypeReviewTierEscalated = "review_tier.escalated"
	// EventTypeDuplicationDetected indicates the duplication detector found similar function pairs.
	EventTypeDuplicationDetected = "duplication.detected"
	// EventTypeDuplicationSuppressed indicates allowlist-matched pairs were suppressed.
	// Used for passive precision measurement: precision = detected / (detected + suppressed).
	EventTypeDuplicationSuppressed = "duplication.suppressed"
	// EventTypeSpawnGateDecision logs every gate evaluation (block, bypass, or allow).
	EventTypeSpawnGateDecision = "spawn.gate_decision"
	// EventTypeAccretionSnapshot logs periodic directory-level line count snapshots for velocity tracking.
	EventTypeAccretionSnapshot = "accretion.snapshot"
	// EventTypeDaemonArchitectEscalation logs daemon routing decisions for hotspot-targeting issues.
	EventTypeDaemonArchitectEscalation = "daemon.architect_escalation"
	// EventTypeExplorationDecomposed logs when an exploration orchestrator decomposes a question into subproblems.
	EventTypeExplorationDecomposed = "exploration.decomposed"
	// EventTypeExplorationJudged logs when an exploration judge produces verdicts on sub-findings.
	EventTypeExplorationJudged = "exploration.judged"
	// EventTypeExplorationSynthesized logs when an exploration run produces a final synthesis.
	EventTypeExplorationSynthesized = "exploration.synthesized"
	// EventTypeExplorationIterated logs when a judge-triggered re-exploration round occurs.
	EventTypeExplorationIterated = "exploration.iterated"
	// EventTypeDecisionMade logs a daemon decision with its classification tier.
	EventTypeDecisionMade = "decision.made"
	// EventTypeCommandInvoked logs when a measurement/diagnostic command is run,
	// with caller context (human, daemon, orchestrator, worker) to track actual usage.
	EventTypeCommandInvoked = "command.invoked"
	// EventTypeAgentRejected logs when an orchestrator rejects agent work quality.
	EventTypeAgentRejected = "agent.rejected"
)

// Event is a loggable event for events.jsonl.
type Event struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// Logger handles event logging to a JSONL file.
type Logger struct {
	Path string
}

// NewLogger creates a new event logger with a custom path.
func NewLogger(path string) *Logger {
	return &Logger{Path: path}
}

// NewDefaultLogger creates a new event logger with the default path (~/.orch/events.jsonl).
func NewDefaultLogger() *Logger {
	return &Logger{Path: DefaultLogPath()}
}

// DefaultLogPath returns the default path to events.jsonl.
// If ORCH_EVENTS_PATH is set, it overrides the default (~/.orch/events.jsonl).
// This is used by tests to prevent writing to the production log.
func DefaultLogPath() string {
	if p := os.Getenv("ORCH_EVENTS_PATH"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orch/events.jsonl"
	}
	return filepath.Join(home, ".orch", "events.jsonl")
}

// CurrentPath returns the path of the current month's rotated event file.
func (l *Logger) CurrentPath() string {
	return RotatedLogPath(l.Path)
}

// Log appends an event to the rotated JSONL log file (events-YYYY-MM.jsonl).
// The directory is derived from l.Path (the legacy events.jsonl path).
func (l *Logger) Log(event Event) error {
	// Write to rotated file based on current month
	target := RotatedLogPath(l.Path)
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	f, err := os.OpenFile(target, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	return nil
}

// LogSpawn logs a session spawn event with prompt and title metadata.
func (l *Logger) LogSpawn(sessionID, prompt, title string) error {
	return l.Log(Event{
		Type:      EventTypeSessionSpawned,
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"prompt": prompt,
			"title":  title,
		},
	})
}

// LogCompleted logs a session completion event.
func (l *Logger) LogCompleted(sessionID string) error {
	return l.Log(Event{
		Type:      EventTypeSessionCompleted,
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
	})
}

// LogError logs a session error event with error message.
func (l *Logger) LogError(sessionID, errMsg string) error {
	return l.Log(Event{
		Type:      EventTypeSessionError,
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"error": errMsg,
		},
	})
}

// LogStatusChange logs a session status change event.
func (l *Logger) LogStatusChange(sessionID, status string) error {
	return l.Log(Event{
		Type:      EventTypeSessionStatus,
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"status": status,
		},
	})
}

// LogAutoCompleted logs an auto-completion event (daemon closed the issue).
func (l *Logger) LogAutoCompleted(beadsID, closeReason string) error {
	return l.LogAutoCompletedWithEscalation(beadsID, closeReason, "")
}

// LogAutoCompletedWithEscalation logs an auto-completion event with escalation level.
func (l *Logger) LogAutoCompletedWithEscalation(beadsID, closeReason, escalationLevel string) error {
	data := map[string]interface{}{
		"beads_id":     beadsID,
		"close_reason": closeReason,
	}
	if escalationLevel != "" {
		data["escalation_level"] = escalationLevel
	}
	return l.Log(Event{
		Type:      EventTypeAutoCompleted,
		SessionID: beadsID, // Using beads ID as session identifier
		Timestamp: time.Now().Unix(),
		Data:      data,
	})
}

// VerificationFailedData contains the data for a verification.failed event.
type VerificationFailedData struct {
	BeadsID           string   `json:"beads_id,omitempty"`
	Workspace         string   `json:"workspace,omitempty"`
	GatesFailed       []string `json:"gates_failed"` // Which gates failed (e.g., "test_evidence", "git_diff")
	Errors            []string `json:"errors"`       // Human-readable error messages
	Skill             string   `json:"skill,omitempty"`
	VerificationLevel string   `json:"verification_level,omitempty"` // V0-V3 level that determined which gates fired
}

// LogVerificationFailed logs a verification failure event.
// This is emitted when verification fails before the user decides to --force or fix.
func (l *Logger) LogVerificationFailed(data VerificationFailedData) error {
	eventData := map[string]interface{}{
		"gates_failed": data.GatesFailed,
		"errors":       data.Errors,
	}
	if data.BeadsID != "" {
		eventData["beads_id"] = data.BeadsID
	}
	if data.Workspace != "" {
		eventData["workspace"] = data.Workspace
	}
	if data.Skill != "" {
		eventData["skill"] = data.Skill
	}
	if data.VerificationLevel != "" {
		eventData["verification_level"] = data.VerificationLevel
	}

	return l.Log(Event{
		Type:      EventTypeVerificationFailed,
		SessionID: data.BeadsID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// PipelineStepTiming records duration and skip status for a completion pipeline step.
type PipelineStepTiming struct {
	Name       string `json:"name"`                  // Step name: hotspot, duplication, build, model_impact
	DurationMs int    `json:"duration_ms"`           // Wall-clock milliseconds (0 if skipped)
	Skipped    bool   `json:"skipped"`               // Was this step skipped?
	SkipReason string `json:"skip_reason,omitempty"` // Why skipped: no_code_files, no_go_changes, orchestrator, no_project_dir
}

// AgentCompletedData contains the data for an agent.completed event.
type AgentCompletedData struct {
	BeadsID            string               `json:"beads_id,omitempty"`
	Workspace          string               `json:"workspace,omitempty"`
	Reason             string               `json:"reason,omitempty"`
	Forced             bool                 `json:"forced"`
	ForceReason        string               `json:"force_reason,omitempty"` // Reason for --force override (separate from close reason)
	Untracked          bool                 `json:"untracked"`
	Orchestrator       bool                 `json:"orchestrator"`
	VerificationPassed bool                 `json:"verification_passed"`      // Did verification pass on first try?
	GatesBypassed      []string             `json:"gates_bypassed,omitempty"` // Which gates were skipped (if forced)
	Skill              string               `json:"skill,omitempty"`
	VerificationLevel  string               `json:"verification_level,omitempty"` // V0-V3 level used at completion (measures what "verified" means)
	DurationSeconds    int                  `json:"duration_seconds,omitempty"`   // Spawn to completion duration
	TokensInput        int                  `json:"tokens_input,omitempty"`       // Total input tokens
	TokensOutput       int                  `json:"tokens_output,omitempty"`      // Total output tokens
	Outcome            string               `json:"outcome,omitempty"`            // success|forced|failed
	PipelineTiming     []PipelineStepTiming `json:"pipeline_timing,omitempty"`    // Per-step timing for completion pipeline
	PipelineTotalMs    int                  `json:"pipeline_total_ms,omitempty"`  // Total advisory pipeline wall-clock ms
}

// LogAgentCompleted logs an agent completion event with verification metadata.
func (l *Logger) LogAgentCompleted(data AgentCompletedData) error {
	eventData := map[string]interface{}{
		"reason":              data.Reason,
		"forced":              data.Forced,
		"untracked":           data.Untracked,
		"orchestrator":        data.Orchestrator,
		"verification_passed": data.VerificationPassed,
	}
	if data.BeadsID != "" {
		eventData["beads_id"] = data.BeadsID
	}
	if data.Workspace != "" {
		eventData["workspace"] = data.Workspace
	}
	if data.ForceReason != "" {
		eventData["force_reason"] = data.ForceReason
	}
	if len(data.GatesBypassed) > 0 {
		eventData["gates_bypassed"] = data.GatesBypassed
	}
	if data.Skill != "" {
		eventData["skill"] = data.Skill
	}
	if data.VerificationLevel != "" {
		eventData["verification_level"] = data.VerificationLevel
	}
	if data.DurationSeconds > 0 {
		eventData["duration_seconds"] = data.DurationSeconds
	}
	if data.TokensInput > 0 {
		eventData["tokens_input"] = data.TokensInput
	}
	if data.TokensOutput > 0 {
		eventData["tokens_output"] = data.TokensOutput
	}
	if data.Outcome != "" {
		eventData["outcome"] = data.Outcome
	}
	if len(data.PipelineTiming) > 0 {
		eventData["pipeline_timing"] = data.PipelineTiming
	}
	if data.PipelineTotalMs > 0 {
		eventData["pipeline_total_ms"] = data.PipelineTotalMs
	}

	return l.Log(Event{
		Type:      EventTypeAgentCompleted,
		SessionID: data.BeadsID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// AgentReworkedData contains the data for an agent.reworked event.
type AgentReworkedData struct {
	BeadsID        string `json:"beads_id,omitempty"`
	PriorWorkspace string `json:"prior_workspace,omitempty"`
	NewWorkspace   string `json:"new_workspace,omitempty"`
	ReworkNumber   int    `json:"rework_number,omitempty"`
	Feedback       string `json:"feedback,omitempty"`
	Skill          string `json:"skill,omitempty"`
	Model          string `json:"model,omitempty"`
}

// LogAgentReworked logs a rework event with metadata for quality tracking.
func (l *Logger) LogAgentReworked(data AgentReworkedData) error {
	eventData := map[string]interface{}{}
	if data.BeadsID != "" {
		eventData["beads_id"] = data.BeadsID
	}
	if data.PriorWorkspace != "" {
		eventData["prior_workspace"] = data.PriorWorkspace
	}
	if data.NewWorkspace != "" {
		eventData["new_workspace"] = data.NewWorkspace
	}
	if data.ReworkNumber > 0 {
		eventData["rework_number"] = data.ReworkNumber
	}
	if data.Feedback != "" {
		eventData["feedback"] = data.Feedback
	}
	if data.Skill != "" {
		eventData["skill"] = data.Skill
	}
	if data.Model != "" {
		eventData["model"] = data.Model
	}

	return l.Log(Event{
		Type:      EventTypeAgentReworked,
		SessionID: data.BeadsID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// AgentAbandonedData contains the data for an agent.abandoned event.
type AgentAbandonedData struct {
	BeadsID         string `json:"beads_id,omitempty"`
	Workspace       string `json:"workspace,omitempty"`
	Reason          string `json:"reason,omitempty"`
	Skill           string `json:"skill,omitempty"`
	DurationSeconds int    `json:"duration_seconds,omitempty"` // Spawn to abandonment duration
	TokensInput     int    `json:"tokens_input,omitempty"`     // Total input tokens
	TokensOutput    int    `json:"tokens_output,omitempty"`    // Total output tokens
	Outcome         string `json:"outcome,omitempty"`          // Always "abandoned"
}

// LogAgentAbandoned logs enriched abandonment telemetry (skill, tokens, duration).
// This uses EventTypeAgentAbandonedTelemetry ("agent.abandoned.telemetry") to avoid
// double-counting with the primary agent.abandoned event emitted by LifecycleManager
// or the direct emit in abandon_cmd.go for untracked agents.
func (l *Logger) LogAgentAbandoned(data AgentAbandonedData) error {
	eventData := map[string]interface{}{
		"reason":  data.Reason,
		"outcome": "abandoned",
	}
	if data.BeadsID != "" {
		eventData["beads_id"] = data.BeadsID
	}
	if data.Workspace != "" {
		eventData["workspace"] = data.Workspace
	}
	if data.Skill != "" {
		eventData["skill"] = data.Skill
	}
	if data.DurationSeconds > 0 {
		eventData["duration_seconds"] = data.DurationSeconds
	}
	if data.TokensInput > 0 {
		eventData["tokens_input"] = data.TokensInput
	}
	if data.TokensOutput > 0 {
		eventData["tokens_output"] = data.TokensOutput
	}

	return l.Log(Event{
		Type:      EventTypeAgentAbandonedTelemetry,
		SessionID: data.BeadsID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// AgentRejectedData contains the data for an agent.rejected event.
type AgentRejectedData struct {
	BeadsID       string `json:"beads_id"`
	Reason        string `json:"reason"`
	Category      string `json:"category"`                // quality, scope, approach, stale
	OriginalSkill string `json:"original_skill,omitempty"` // Skill from the rejected work
	OriginalModel string `json:"original_model,omitempty"` // Model from the rejected work
}

// LogAgentRejected logs a quality rejection event for negative feedback signal.
func (l *Logger) LogAgentRejected(data AgentRejectedData) error {
	eventData := map[string]interface{}{
		"beads_id": data.BeadsID,
		"reason":   data.Reason,
		"category": data.Category,
	}
	if data.OriginalSkill != "" {
		eventData["original_skill"] = data.OriginalSkill
	}
	if data.OriginalModel != "" {
		eventData["original_model"] = data.OriginalModel
	}

	return l.Log(Event{
		Type:      EventTypeAgentRejected,
		SessionID: data.BeadsID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// ServiceEventData contains the data for service lifecycle events.
type ServiceEventData struct {
	ServiceName  string `json:"service_name"`
	ProjectPath  string `json:"project_path"`
	OldPID       int    `json:"old_pid,omitempty"`
	NewPID       int    `json:"new_pid,omitempty"`
	RestartCount int    `json:"restart_count,omitempty"`
	AutoRestart  bool   `json:"auto_restart,omitempty"` // Was it auto-restarted by monitor?
}

// LogServiceCrashed logs a service crash event.
func (l *Logger) LogServiceCrashed(data ServiceEventData) error {
	eventData := map[string]interface{}{
		"service_name": data.ServiceName,
		"project_path": data.ProjectPath,
	}
	if data.OldPID != 0 {
		eventData["old_pid"] = data.OldPID
	}
	if data.NewPID != 0 {
		eventData["new_pid"] = data.NewPID
	}

	return l.Log(Event{
		Type:      EventTypeServiceCrashed,
		SessionID: data.ServiceName, // Use service name as session ID for grouping
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// LogServiceRestarted logs a service restart event (after crash).
func (l *Logger) LogServiceRestarted(data ServiceEventData) error {
	eventData := map[string]interface{}{
		"service_name":  data.ServiceName,
		"project_path":  data.ProjectPath,
		"restart_count": data.RestartCount,
		"auto_restart":  data.AutoRestart,
	}
	if data.NewPID != 0 {
		eventData["new_pid"] = data.NewPID
	}

	return l.Log(Event{
		Type:      EventTypeServiceRestarted,
		SessionID: data.ServiceName,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// LogServiceStarted logs a service start event (first time seen).
func (l *Logger) LogServiceStarted(data ServiceEventData) error {
	eventData := map[string]interface{}{
		"service_name": data.ServiceName,
		"project_path": data.ProjectPath,
	}
	if data.NewPID != 0 {
		eventData["pid"] = data.NewPID
	}

	return l.Log(Event{
		Type:      EventTypeServiceStarted,
		SessionID: data.ServiceName,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// VerificationBypassedData contains the data for a verification.bypassed event.
type VerificationBypassedData struct {
	BeadsID           string `json:"beads_id,omitempty"`
	Workspace         string `json:"workspace,omitempty"`
	Gate              string `json:"gate"`   // Which gate was bypassed (e.g., "test_evidence", "git_diff")
	Reason            string `json:"reason"` // User-provided reason for bypass
	Skill             string `json:"skill,omitempty"`
	VerificationLevel string `json:"verification_level,omitempty"` // V0-V3 level context for the bypass
}

// LogVerificationBypassed logs a verification gate bypass event.
// This is emitted when a user explicitly bypasses a gate via --skip-* flags.
func (l *Logger) LogVerificationBypassed(data VerificationBypassedData) error {
	eventData := map[string]interface{}{
		"gate":   data.Gate,
		"reason": data.Reason,
	}
	if data.BeadsID != "" {
		eventData["beads_id"] = data.BeadsID
	}
	if data.Workspace != "" {
		eventData["workspace"] = data.Workspace
	}
	if data.Skill != "" {
		eventData["skill"] = data.Skill
	}
	if data.VerificationLevel != "" {
		eventData["verification_level"] = data.VerificationLevel
	}

	return l.Log(Event{
		Type:      EventTypeVerificationBypassed,
		SessionID: data.BeadsID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// LogVerificationAutoSkipped logs a verification gate auto-skip event.
// This is emitted when a gate is automatically skipped due to skill-class or file type exemptions.
func (l *Logger) LogVerificationAutoSkipped(data VerificationBypassedData) error {
	eventData := map[string]interface{}{
		"gate":   data.Gate,
		"reason": data.Reason,
	}
	if data.BeadsID != "" {
		eventData["beads_id"] = data.BeadsID
	}
	if data.Workspace != "" {
		eventData["workspace"] = data.Workspace
	}
	if data.Skill != "" {
		eventData["skill"] = data.Skill
	}
	if data.VerificationLevel != "" {
		eventData["verification_level"] = data.VerificationLevel
	}

	return l.Log(Event{
		Type:      EventTypeVerificationAutoSkipped,
		SessionID: data.BeadsID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// ReviewTierEscalatedData contains the data for a review_tier.escalated event.
type ReviewTierEscalatedData struct {
	BeadsID      string   `json:"beads_id,omitempty"`
	Workspace    string   `json:"workspace,omitempty"`
	Skill        string   `json:"skill,omitempty"`
	OriginalTier string   `json:"original_tier"`
	EscalatedTo  string   `json:"escalated_to"`
	Reasons      []string `json:"reasons"`
}

// LogReviewTierEscalated logs a review tier escalation event.
func (l *Logger) LogReviewTierEscalated(data ReviewTierEscalatedData) error {
	eventData := map[string]interface{}{
		"original_tier": data.OriginalTier,
		"escalated_to":  data.EscalatedTo,
		"reasons":       data.Reasons,
	}
	if data.BeadsID != "" {
		eventData["beads_id"] = data.BeadsID
	}
	if data.Workspace != "" {
		eventData["workspace"] = data.Workspace
	}
	if data.Skill != "" {
		eventData["skill"] = data.Skill
	}

	return l.Log(Event{
		Type:      EventTypeReviewTierEscalated,
		SessionID: data.BeadsID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// DuplicationMatch represents a single duplicate function pair for event logging.
type DuplicationMatch struct {
	FileA       string  `json:"file_a"`
	FuncA       string  `json:"func_a"`
	FileB       string  `json:"file_b"`
	FuncB       string  `json:"func_b"`
	Similarity  float64 `json:"similarity"`
}

// DuplicationDetectedData contains the data for a duplication.detected event.
type DuplicationDetectedData struct {
	BeadsID   string               `json:"beads_id,omitempty"`
	Workspace string               `json:"workspace,omitempty"`
	Matches   []DuplicationMatch   `json:"matches"`
	Count     int                  `json:"count"`
}

// LogDuplicationDetected logs a duplication detection event with match details.
func (l *Logger) LogDuplicationDetected(data DuplicationDetectedData) error {
	eventData := map[string]interface{}{
		"matches": data.Matches,
		"count":   data.Count,
	}
	if data.BeadsID != "" {
		eventData["beads_id"] = data.BeadsID
	}
	if data.Workspace != "" {
		eventData["workspace"] = data.Workspace
	}

	return l.Log(Event{
		Type:      EventTypeDuplicationDetected,
		SessionID: data.BeadsID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// DuplicationSuppressedMatch represents a single allowlist-suppressed pair for event logging.
type DuplicationSuppressedMatch struct {
	FuncA      string  `json:"func_a"`
	FuncB      string  `json:"func_b"`
	Similarity float64 `json:"similarity"`
	Pattern    string  `json:"pattern"` // the allowlist pattern that matched
}

// DuplicationSuppressedData contains the data for a duplication.suppressed event.
type DuplicationSuppressedData struct {
	BeadsID   string                        `json:"beads_id,omitempty"`
	Workspace string                        `json:"workspace,omitempty"`
	Matches   []DuplicationSuppressedMatch  `json:"matches"`
	Count     int                           `json:"count"`
}

// LogDuplicationSuppressed logs pairs suppressed by the allowlist for precision tracking.
func (l *Logger) LogDuplicationSuppressed(data DuplicationSuppressedData) error {
	eventData := map[string]interface{}{
		"matches": data.Matches,
		"count":   data.Count,
	}
	if data.BeadsID != "" {
		eventData["beads_id"] = data.BeadsID
	}
	if data.Workspace != "" {
		eventData["workspace"] = data.Workspace
	}

	return l.Log(Event{
		Type:      EventTypeDuplicationSuppressed,
		SessionID: data.BeadsID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// SkillInferredData contains the data for a spawn.skill_inferred event.
type SkillInferredData struct {
	IssueID                  string `json:"issue_id"`
	InferredSkill            string `json:"inferred_skill"`
	IssueType                string `json:"issue_type"`
	Title                    string `json:"title"`
	HadSkillLabel            bool   `json:"had_skill_label"`
	HadTitleMatch            bool   `json:"had_title_match"`
	UsedDescriptionHeuristic bool   `json:"used_description_heuristic"`
}

// LogSkillInferred logs a skill inference event with metadata for accuracy tracking.
func (l *Logger) LogSkillInferred(data SkillInferredData) error {
	eventData := map[string]interface{}{
		"issue_id":                   data.IssueID,
		"inferred_skill":             data.InferredSkill,
		"issue_type":                 data.IssueType,
		"title":                      data.Title,
		"had_skill_label":            data.HadSkillLabel,
		"had_title_match":            data.HadTitleMatch,
		"used_description_heuristic": data.UsedDescriptionHeuristic,
	}

	return l.Log(Event{
		Type:      EventTypeSpawnSkillInferred,
		SessionID: data.IssueID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// ArchitectEscalationData contains the data for a daemon.architect_escalation event.
type ArchitectEscalationData struct {
	IssueID           string `json:"issue_id"`
	HotspotFile       string `json:"hotspot_file"`
	HotspotType       string `json:"hotspot_type"`
	Escalated         bool   `json:"escalated"`
	PriorArchitectRef string `json:"prior_architect_ref,omitempty"`
}

// LogArchitectEscalation logs a daemon architect escalation decision.
func (l *Logger) LogArchitectEscalation(data ArchitectEscalationData) error {
	eventData := map[string]interface{}{
		"issue_id":     data.IssueID,
		"hotspot_file": data.HotspotFile,
		"hotspot_type": data.HotspotType,
		"escalated":    data.Escalated,
	}
	if data.PriorArchitectRef != "" {
		eventData["prior_architect_ref"] = data.PriorArchitectRef
	}

	return l.Log(Event{
		Type:      EventTypeDaemonArchitectEscalation,
		SessionID: data.IssueID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// DirectorySnapshot represents line count metrics for a single directory.
type DirectorySnapshot struct {
	Directory     string `json:"directory"`
	TotalLines    int    `json:"total_lines"`
	FileCount     int    `json:"file_count"`
	FilesOver800  int    `json:"files_over_800"`
	FilesOver1500 int    `json:"files_over_1500"`
	LargestFile   string `json:"largest_file,omitempty"`
	LargestLines  int    `json:"largest_lines,omitempty"`
}

// AccretionSnapshotData contains the data for an accretion.snapshot event.
type AccretionSnapshotData struct {
	Directories  []DirectorySnapshot `json:"directories"`
	SnapshotType string              `json:"snapshot_type"` // "weekly", "manual", "baseline"
}

// LogAccretionSnapshot logs a periodic directory-level line count snapshot.
func (l *Logger) LogAccretionSnapshot(data AccretionSnapshotData) error {
	return l.Log(Event{
		Type:      EventTypeAccretionSnapshot,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"directories":   data.Directories,
			"snapshot_type": data.SnapshotType,
		},
	})
}

// ExplorationDecomposedData contains the data for an exploration.decomposed event.
type ExplorationDecomposedData struct {
	BeadsID     string   `json:"beads_id,omitempty"`
	ParentSkill string   `json:"parent_skill,omitempty"` // investigation or architect
	Question    string   `json:"question,omitempty"`     // Original question being explored
	Subproblems []string `json:"subproblems"`            // List of decomposed subproblem descriptions
	Breadth     int      `json:"breadth"`                // Number of parallel workers
}

// LogExplorationDecomposed logs when an exploration orchestrator decomposes a question into subproblems.
func (l *Logger) LogExplorationDecomposed(data ExplorationDecomposedData) error {
	eventData := map[string]interface{}{
		"subproblems": data.Subproblems,
		"breadth":     data.Breadth,
	}
	if data.BeadsID != "" {
		eventData["beads_id"] = data.BeadsID
	}
	if data.ParentSkill != "" {
		eventData["parent_skill"] = data.ParentSkill
	}
	if data.Question != "" {
		eventData["question"] = data.Question
	}

	return l.Log(Event{
		Type:      EventTypeExplorationDecomposed,
		SessionID: data.BeadsID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// ExplorationJudgedData contains the data for an exploration.judged event.
type ExplorationJudgedData struct {
	BeadsID       string `json:"beads_id,omitempty"`
	ParentSkill   string `json:"parent_skill,omitempty"`
	TotalFindings int    `json:"total_findings"`
	Accepted      int    `json:"accepted"`
	Contested     int    `json:"contested"`
	Rejected      int    `json:"rejected"`
	CoverageGaps  int    `json:"coverage_gaps"`
}

// LogExplorationJudged logs when an exploration judge produces verdicts on sub-findings.
func (l *Logger) LogExplorationJudged(data ExplorationJudgedData) error {
	eventData := map[string]interface{}{
		"total_findings": data.TotalFindings,
		"accepted":       data.Accepted,
		"contested":      data.Contested,
		"rejected":       data.Rejected,
		"coverage_gaps":  data.CoverageGaps,
	}
	if data.BeadsID != "" {
		eventData["beads_id"] = data.BeadsID
	}
	if data.ParentSkill != "" {
		eventData["parent_skill"] = data.ParentSkill
	}

	return l.Log(Event{
		Type:      EventTypeExplorationJudged,
		SessionID: data.BeadsID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// ExplorationSynthesizedData contains the data for an exploration.synthesized event.
type ExplorationSynthesizedData struct {
	BeadsID         string `json:"beads_id,omitempty"`
	ParentSkill     string `json:"parent_skill,omitempty"`
	WorkerCount     int    `json:"worker_count"`
	DurationSeconds int    `json:"duration_seconds,omitempty"` // Total exploration wall-clock time
	SynthesisPath   string `json:"synthesis_path,omitempty"`   // Path to synthesis output file
}

// LogExplorationSynthesized logs when an exploration run produces a final synthesis.
func (l *Logger) LogExplorationSynthesized(data ExplorationSynthesizedData) error {
	eventData := map[string]interface{}{
		"worker_count": data.WorkerCount,
	}
	if data.BeadsID != "" {
		eventData["beads_id"] = data.BeadsID
	}
	if data.ParentSkill != "" {
		eventData["parent_skill"] = data.ParentSkill
	}
	if data.DurationSeconds > 0 {
		eventData["duration_seconds"] = data.DurationSeconds
	}
	if data.SynthesisPath != "" {
		eventData["synthesis_path"] = data.SynthesisPath
	}

	return l.Log(Event{
		Type:      EventTypeExplorationSynthesized,
		SessionID: data.BeadsID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// ExplorationIteratedData contains the data for an exploration.iterated event.
type ExplorationIteratedData struct {
	BeadsID       string `json:"beads_id,omitempty"`
	ParentSkill   string `json:"parent_skill,omitempty"`
	Iteration     int    `json:"iteration"`      // Current iteration number (2 = first re-exploration)
	GapsAddressed int    `json:"gaps_addressed"`  // Number of critical gaps being addressed
	NewWorkers    int    `json:"new_workers"`     // Number of new workers spawned for gap-filling
}

// LogExplorationIterated logs when a judge-triggered re-exploration round occurs.
func (l *Logger) LogExplorationIterated(data ExplorationIteratedData) error {
	eventData := map[string]interface{}{
		"iteration":      data.Iteration,
		"gaps_addressed": data.GapsAddressed,
		"new_workers":    data.NewWorkers,
	}
	if data.BeadsID != "" {
		eventData["beads_id"] = data.BeadsID
	}
	if data.ParentSkill != "" {
		eventData["parent_skill"] = data.ParentSkill
	}

	return l.Log(Event{
		Type:      EventTypeExplorationIterated,
		SessionID: data.BeadsID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// GateDecisionData contains the data for a spawn.gate_decision event.
// Block, bypass, and allow decisions are all logged for true fire rate calculation.
type GateDecisionData struct {
	GateName    string   `json:"gate_name"`              // hotspot, triage, verification, accretion_precommit
	Decision    string   `json:"decision"`               // block, bypass, allow
	Skill       string   `json:"skill,omitempty"`        // Skill being spawned
	BeadsID     string   `json:"beads_id,omitempty"`     // Issue ID if available
	TargetFiles []string `json:"target_files,omitempty"` // Files that triggered the gate
	Reason      string   `json:"reason,omitempty"`       // Why the decision was made
}

// LogGateDecision logs a spawn gate evaluation event (block, bypass, or allow).
func (l *Logger) LogGateDecision(data GateDecisionData) error {
	eventData := map[string]interface{}{
		"gate_name": data.GateName,
		"decision":  data.Decision,
	}
	if data.Skill != "" {
		eventData["skill"] = data.Skill
	}
	if data.BeadsID != "" {
		eventData["beads_id"] = data.BeadsID
	}
	if len(data.TargetFiles) > 0 {
		eventData["target_files"] = data.TargetFiles
	}
	if data.Reason != "" {
		eventData["reason"] = data.Reason
	}

	return l.Log(Event{
		Type:      EventTypeSpawnGateDecision,
		SessionID: data.BeadsID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}
