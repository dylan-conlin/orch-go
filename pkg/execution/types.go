// Package execution provides backend-agnostic types and interfaces for session
// management. This abstraction layer decouples the orchestration system from any
// specific execution backend (OpenCode, OpenClaw, Claude CLI).
package execution

import "time"

// SessionHandle is an opaque identifier for a session. It could be an OpenCode
// UUID, an OpenClaw runId, or any other backend-specific identifier.
type SessionHandle string

// String returns the string representation of the handle.
func (h SessionHandle) String() string { return string(h) }

// SessionInfo contains backend-agnostic session metadata.
type SessionInfo struct {
	ID        string
	Directory string
	Title     string
	ParentID  string
	Created   time.Time
	Updated   time.Time
	Metadata  map[string]string
	Summary   ChangeSummary
}

// ChangeSummary tracks file changes made during a session.
type ChangeSummary struct {
	Additions int
	Deletions int
	Files     int
}

// CompletionStatus represents the result of waiting for session completion.
type CompletionStatus struct {
	Status   string        // "completed", "error", "timeout"
	Error    string        // Error message if Status == "error"
	Duration time.Duration // How long the session ran
}

// IsComplete returns true if the session completed successfully.
func (s CompletionStatus) IsComplete() bool { return s.Status == "completed" }

// IsError returns true if the session ended with an error.
func (s CompletionStatus) IsError() bool { return s.Status == "error" }

// Message represents a backend-agnostic conversation message.
type Message struct {
	ID        string
	SessionID string
	Role      string // "user" or "assistant"
	Created   time.Time
	Completed time.Time
	Finish    string // "stop", "error", etc.
	Parts     []MessagePart
	Tokens    *TokenCount
	Cost      float64
}

// MessagePart represents a segment of a message (text, tool call, etc.).
type MessagePart struct {
	Type   string // "text", "tool-invocation", "tool", "reasoning", etc.
	Text   string
	Tool   string
	CallID string
	State  *ToolState
}

// ToolState represents the execution state of a tool invocation.
type ToolState struct {
	Status   string
	Input    map[string]interface{}
	Output   string
	Title    string
	Metadata map[string]interface{}
}

// TokenCount contains token usage for a message.
type TokenCount struct {
	Input     int
	Output    int
	Reasoning int
	CacheRead int
}

// TokenStats contains aggregated token usage for a session.
type TokenStats struct {
	InputTokens     int `json:"input_tokens"`
	OutputTokens    int `json:"output_tokens"`
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
	CacheReadTokens int `json:"cache_read_tokens,omitempty"`
	TotalTokens     int `json:"total_tokens"` // Input + Output + Reasoning
}

// AggregateTokens calculates total token usage from a slice of messages.
func AggregateTokens(messages []Message) TokenStats {
	var stats TokenStats
	for _, msg := range messages {
		if msg.Tokens == nil {
			continue
		}
		stats.InputTokens += msg.Tokens.Input
		stats.OutputTokens += msg.Tokens.Output
		stats.ReasoningTokens += msg.Tokens.Reasoning
		stats.CacheReadTokens += msg.Tokens.CacheRead
	}
	stats.TotalTokens = stats.InputTokens + stats.OutputTokens + stats.ReasoningTokens
	return stats
}

// SessionStatusInfo represents the current processing status of a session.
type SessionStatusInfo struct {
	Type    string // "idle", "busy", "retry"
	Message string
}

// IsIdle returns true if the session is idle.
func (s *SessionStatusInfo) IsIdle() bool { return s.Type == "idle" }

// IsBusy returns true if the session is busy.
func (s *SessionStatusInfo) IsBusy() bool { return s.Type == "busy" }

// SessionRequest contains parameters for creating a new session.
type SessionRequest struct {
	Title     string
	Directory string
	Model     string
	Prompt    string
	Metadata  map[string]string
	TimeTTL   int // Session TTL in seconds (0 = no expiration)
}

// LastActivity represents the most recent activity from a session.
type LastActivity struct {
	Text      string
	Timestamp time.Time
}

// ExtractRecentText extracts the most recent text content from messages.
// It returns up to `lines` worth of text from the most recent messages.
func ExtractRecentText(messages []Message, lines int) []string {
	var result []string
	for i := len(messages) - 1; i >= 0 && len(result) < lines; i-- {
		msg := messages[i]
		for j := len(msg.Parts) - 1; j >= 0 && len(result) < lines; j-- {
			part := msg.Parts[j]
			if part.Type == "text" && part.Text != "" {
				textLines := splitLines(part.Text)
				for k := len(textLines) - 1; k >= 0 && len(result) < lines; k-- {
					line := textLines[k]
					if line != "" || len(result) > 0 {
						result = append([]string{line}, result...)
					}
				}
			}
		}
	}
	if len(result) > lines {
		result = result[len(result)-lines:]
	}
	return result
}

// splitLines splits a string into lines without importing strings.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
