package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogCommandInvoked(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogCommandInvoked(CommandInvokedData{
		Command: "harness audit",
		Caller:  "human",
		Flags:   "--days=7 --json=true",
	})
	if err != nil {
		t.Fatalf("LogCommandInvoked() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if event.Type != EventTypeCommandInvoked {
		t.Errorf("event.Type = %q, want %q", event.Type, EventTypeCommandInvoked)
	}
	if event.Data["command"] != "harness audit" {
		t.Errorf("data.command = %v, want %q", event.Data["command"], "harness audit")
	}
	if event.Data["caller"] != "human" {
		t.Errorf("data.caller = %v, want %q", event.Data["caller"], "human")
	}
	if event.Data["flags"] != "--days=7 --json=true" {
		t.Errorf("data.flags = %v, want %q", event.Data["flags"], "--days=7 --json=true")
	}
}

func TestLogCommandInvoked_NoFlags(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogCommandInvoked(CommandInvokedData{
		Command: "stats",
		Caller:  "worker",
	})
	if err != nil {
		t.Fatalf("LogCommandInvoked() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	raw := string(data)
	var event Event
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &event); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if event.Data["command"] != "stats" {
		t.Errorf("data.command = %v, want %q", event.Data["command"], "stats")
	}
	if event.Data["caller"] != "worker" {
		t.Errorf("data.caller = %v, want %q", event.Data["caller"], "worker")
	}
	// flags should be omitted when empty
	if _, ok := event.Data["flags"]; ok {
		t.Error("Expected flags to be omitted when empty")
	}
}
