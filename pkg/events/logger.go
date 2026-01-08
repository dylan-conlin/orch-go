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

	return l.Log(Event{
		Type:      EventTypeAgentCompleted,
		SessionID: data.BeadsID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}
