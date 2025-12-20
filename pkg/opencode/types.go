// Package opencode provides a client for interacting with OpenCode sessions.
package opencode

import (
	"encoding/json"
	"errors"
)

// ErrNoSessionID is returned when no session ID is found in output.
var ErrNoSessionID = errors.New("no session ID found in output")

// Event represents an event from opencode's JSON output.
type Event struct {
	Type      string          `json:"type"`
	SessionID string          `json:"sessionID,omitempty"`
	Session   *SessionInfo    `json:"session,omitempty"`
	Step      *StepInfo       `json:"step,omitempty"`
	Content   string          `json:"content,omitempty"`
	Timestamp int64           `json:"timestamp,omitempty"`
	Raw       json.RawMessage `json:"-"`
}

// SessionInfo contains session details.
type SessionInfo struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}

// StepInfo contains step details.
type StepInfo struct {
	ID string `json:"id"`
}

// SSEEvent represents a Server-Sent Event.
type SSEEvent struct {
	Event string
	Data  string
}

// SessionStatus represents a session status from SSE data.
type SessionStatus struct {
	Status    string `json:"status"`
	SessionID string `json:"session_id"`
}

// Result holds the result of processing opencode output.
type Result struct {
	SessionID string
	Events    []Event
}
