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

// Session represents a session from the OpenCode /session API.
type Session struct {
	ID        string         `json:"id"`
	Version   string         `json:"version,omitempty"`
	ProjectID string         `json:"projectID,omitempty"`
	Directory string         `json:"directory"`
	Title     string         `json:"title"`
	ParentID  string         `json:"parentID,omitempty"`
	Time      SessionTime    `json:"time"`
	Summary   SessionSummary `json:"summary,omitempty"`
}

// SessionTime contains session timing information.
type SessionTime struct {
	Created int64 `json:"created"`
	Updated int64 `json:"updated"`
}

// SessionSummary contains session change summary.
type SessionSummary struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Files     int `json:"files"`
}

// Message represents a message from the OpenCode /session/{id}/message API.
type Message struct {
	Info  MessageInfo   `json:"info"`
	Parts []MessagePart `json:"parts"`
}

// MessageInfo contains message metadata.
type MessageInfo struct {
	ID         string      `json:"id"`
	SessionID  string      `json:"sessionID"`
	Role       string      `json:"role"` // "user" or "assistant"
	Time       MessageTime `json:"time"`
	ParentID   string      `json:"parentID,omitempty"`
	ModelID    string      `json:"modelID,omitempty"`
	ProviderID string      `json:"providerID,omitempty"`
	Mode       string      `json:"mode,omitempty"`
	Finish     string      `json:"finish,omitempty"` // "stop", "error", etc.
}

// MessageTime contains message timing.
type MessageTime struct {
	Created   int64 `json:"created"`
	Completed int64 `json:"completed,omitempty"`
}

// MessagePart represents a part of a message (text, reasoning, tool call, etc.).
type MessagePart struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"`
	MessageID string `json:"messageID"`
	Type      string `json:"type"` // "text", "reasoning", "step-start", "step-finish", "tool-invocation", etc.
	Text      string `json:"text,omitempty"`
}
