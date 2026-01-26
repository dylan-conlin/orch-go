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
	// EventTypeAgentAbandoned indicates an agent was abandoned via orch abandon.
	EventTypeAgentAbandoned = "agent.abandoned"
	// EventTypeDedupBlocked indicates a spawn was blocked by a deduplication layer.
	EventTypeDedupBlocked = "daemon.dedup_blocked"
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
func DefaultLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orch/events.jsonl"
	}
	return filepath.Join(home, ".orch", "events.jsonl")
}

// Log appends an event to the JSONL log file.
func (l *Logger) Log(event Event) error {
	// Ensure directory exists
	dir := filepath.Dir(l.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file for appending
	f, err := os.OpenFile(l.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	// Encode and write
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
	BeadsID     string   `json:"beads_id,omitempty"`
	Workspace   string   `json:"workspace,omitempty"`
	GatesFailed []string `json:"gates_failed"` // Which gates failed (e.g., "test_evidence", "git_diff")
	Errors      []string `json:"errors"`       // Human-readable error messages
	Skill       string   `json:"skill,omitempty"`
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

	return l.Log(Event{
		Type:      EventTypeVerificationFailed,
		SessionID: data.BeadsID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// AgentCompletedData contains the data for an agent.completed event.
type AgentCompletedData struct {
	BeadsID            string   `json:"beads_id,omitempty"`
	Workspace          string   `json:"workspace,omitempty"`
	Reason             string   `json:"reason,omitempty"`
	Forced             bool     `json:"forced"`
	Untracked          bool     `json:"untracked"`
	Orchestrator       bool     `json:"orchestrator"`
	VerificationPassed bool     `json:"verification_passed"`      // Did verification pass on first try?
	GatesBypassed      []string `json:"gates_bypassed,omitempty"` // Which gates were skipped (if forced)
	Skill              string   `json:"skill,omitempty"`
	DurationSeconds    int      `json:"duration_seconds,omitempty"` // Spawn to completion duration
	TokensInput        int      `json:"tokens_input,omitempty"`     // Total input tokens
	TokensOutput       int      `json:"tokens_output,omitempty"`    // Total output tokens
	Outcome            string   `json:"outcome,omitempty"`          // success|forced|failed
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
	if len(data.GatesBypassed) > 0 {
		eventData["gates_bypassed"] = data.GatesBypassed
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
	if data.Outcome != "" {
		eventData["outcome"] = data.Outcome
	}

	return l.Log(Event{
		Type:      EventTypeAgentCompleted,
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

// LogAgentAbandoned logs an agent abandonment event with telemetry.
func (l *Logger) LogAgentAbandoned(data AgentAbandonedData) error {
	eventData := map[string]interface{}{
		"reason":  data.Reason,
		"outcome": "abandoned", // Always abandoned for this event type
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
		Type:      EventTypeAgentAbandoned,
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
	BeadsID   string `json:"beads_id,omitempty"`
	Workspace string `json:"workspace,omitempty"`
	Gate      string `json:"gate"`   // Which gate was bypassed (e.g., "test_evidence", "git_diff")
	Reason    string `json:"reason"` // User-provided reason for bypass
	Skill     string `json:"skill,omitempty"`
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

	return l.Log(Event{
		Type:      EventTypeVerificationBypassed,
		SessionID: data.BeadsID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}

// DedupBlockedData contains the data for a daemon.dedup_blocked event.
type DedupBlockedData struct {
	BeadsID    string `json:"beads_id"`
	DedupLayer string `json:"dedup_layer"` // Which layer blocked: "spawned_tracker", "session_dedup", "phase_complete", "beads_status"
	Reason     string `json:"reason"`      // Human-readable reason
}

// LogDedupBlocked logs a deduplication blocking event.
// This is emitted when the daemon skips spawning an issue because a dedup layer detected it already exists.
// Accepts either DedupBlockedData struct or map[string]interface{} for flexibility.
func (l *Logger) LogDedupBlocked(data interface{}) error {
	var eventData map[string]interface{}

	switch d := data.(type) {
	case DedupBlockedData:
		eventData = map[string]interface{}{
			"beads_id":    d.BeadsID,
			"dedup_layer": d.DedupLayer,
			"reason":      d.Reason,
		}
	case map[string]interface{}:
		// Already a map, use directly
		eventData = d
	default:
		return fmt.Errorf("unexpected data type for LogDedupBlocked: %T", data)
	}

	beadsID, _ := eventData["beads_id"].(string)

	return l.Log(Event{
		Type:      EventTypeDedupBlocked,
		SessionID: beadsID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}
