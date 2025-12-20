// Package events provides event logging functionality.
package events

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// NewLogger creates a new event logger.
func NewLogger(path string) *Logger {
	return &Logger{Path: path}
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
