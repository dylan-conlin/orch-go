package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

func TestRunEmit_AgentCompleted(t *testing.T) {
	// Create temp directory for events.jsonl
	tmpDir := t.TempDir()
	eventsPath := filepath.Join(tmpDir, "events.jsonl")

	// Override the default log path by using the events package directly
	// We'll test the event creation logic
	logger := events.NewLogger(eventsPath)

	// Test data
	beadsID := "test-abc123"
	reason := "Test reason"

	// Build event data as runEmit does
	eventData := map[string]interface{}{
		"beads_id": beadsID,
		"source":   "bd_close_hook",
		"reason":   reason,
	}

	event := events.Event{
		Type:      "agent.completed",
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}

	// Log the event
	if err := logger.Log(event); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}

	// Verify file was created and contains the event
	data, err := os.ReadFile(events.RotatedLogPath(eventsPath))
	if err != nil {
		t.Fatalf("failed to read events file: %v", err)
	}

	var readEvent events.Event
	if err := json.Unmarshal(data[:len(data)-1], &readEvent); err != nil { // -1 to remove newline
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if readEvent.Type != "agent.completed" {
		t.Errorf("expected type agent.completed, got %s", readEvent.Type)
	}

	if readEvent.Data["beads_id"] != beadsID {
		t.Errorf("expected beads_id %s, got %v", beadsID, readEvent.Data["beads_id"])
	}

	if readEvent.Data["source"] != "bd_close_hook" {
		t.Errorf("expected source bd_close_hook, got %v", readEvent.Data["source"])
	}

	if readEvent.Data["reason"] != reason {
		t.Errorf("expected reason %s, got %v", reason, readEvent.Data["reason"])
	}
}

func TestRunEmit_MissingBeadsID(t *testing.T) {
	// Test that agent.completed requires beads-id
	err := runEmit("agent.completed", "", "", "", false)
	if err == nil {
		t.Error("expected error for missing beads-id")
	}

	expectedMsg := "--beads-id is required for agent.completed events"
	if err.Error() != expectedMsg {
		t.Errorf("expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestRunEmit_UnsupportedEventType(t *testing.T) {
	// Test that unsupported event types are rejected
	err := runEmit("unsupported.event", "test-123", "", "", false)
	if err == nil {
		t.Error("expected error for unsupported event type")
	}

	expectedPrefix := "unsupported event type: unsupported.event"
	if err.Error()[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("expected error starting with %q, got %q", expectedPrefix, err.Error())
	}
}

func TestRunEmit_AdditionalData(t *testing.T) {
	// Create temp directory for events.jsonl
	tmpDir := t.TempDir()
	eventsPath := filepath.Join(tmpDir, "events.jsonl")

	logger := events.NewLogger(eventsPath)

	// Test merging additional data
	eventData := map[string]interface{}{
		"beads_id": "test-456",
		"source":   "bd_close_hook",
	}

	// Parse additional data
	additionalJSON := `{"custom_field":"custom_value","source":"override"}`
	var additionalData map[string]interface{}
	if err := json.Unmarshal([]byte(additionalJSON), &additionalData); err != nil {
		t.Fatalf("failed to parse additional data: %v", err)
	}

	// Merge (additional data takes precedence)
	for k, v := range additionalData {
		eventData[k] = v
	}

	event := events.Event{
		Type:      "agent.completed",
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}

	if err := logger.Log(event); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}

	// Verify
	data, err := os.ReadFile(events.RotatedLogPath(eventsPath))
	if err != nil {
		t.Fatalf("failed to read events file: %v", err)
	}

	var readEvent events.Event
	if err := json.Unmarshal(data[:len(data)-1], &readEvent); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	// Check that additional data was merged and takes precedence
	if readEvent.Data["custom_field"] != "custom_value" {
		t.Errorf("expected custom_field to be custom_value, got %v", readEvent.Data["custom_field"])
	}

	if readEvent.Data["source"] != "override" {
		t.Errorf("expected source to be override (from additional data), got %v", readEvent.Data["source"])
	}
}

func TestRunEmit_InvalidDataJSON(t *testing.T) {
	// Test that invalid JSON in --data is rejected
	err := runEmit("agent.completed", "test-123", "", "invalid-json", false)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	expectedPrefix := "invalid --data JSON"
	if len(err.Error()) < len(expectedPrefix) || err.Error()[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("expected error starting with %q, got %q", expectedPrefix, err.Error())
	}
}
