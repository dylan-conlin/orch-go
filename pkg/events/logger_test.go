package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger("/tmp/test.jsonl")
	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}
	if logger.Path != "/tmp/test.jsonl" {
		t.Errorf("NewLogger() path = %v, want /tmp/test.jsonl", logger.Path)
	}
}

func TestDefaultLogPath(t *testing.T) {
	path := DefaultLogPath()
	if path == "" {
		t.Error("DefaultLogPath() returned empty string")
	}
	if !strings.Contains(path, ".orch") {
		t.Errorf("DefaultLogPath() = %v, should contain .orch", path)
	}
	if !strings.HasSuffix(path, "events.jsonl") {
		t.Errorf("DefaultLogPath() = %v, should end with events.jsonl", path)
	}
}

func TestLogEvent_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	event := Event{
		Type:      "session.spawned",
		SessionID: "ses_123",
		Timestamp: time.Now().Unix(),
		Data:      map[string]interface{}{"prompt": "test prompt"},
	}

	err := logger.Log(event)
	if err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	// Verify file was created and contains expected content
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(data), "session.spawned") {
		t.Error("Log file doesn't contain event type")
	}
	if !strings.Contains(string(data), "ses_123") {
		t.Error("Log file doesn't contain session ID")
	}
}

func TestLogEvent_MultipleEvents(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	events := []Event{
		{Type: "session.spawned", SessionID: "ses_1", Timestamp: 1703001600},
		{Type: "session.completed", SessionID: "ses_1", Timestamp: 1703001700},
		{Type: "session.error", SessionID: "ses_2", Timestamp: 1703001800},
	}

	for _, event := range events {
		if err := logger.Log(event); err != nil {
			t.Fatalf("Log() error = %v", err)
		}
	}

	// Read and verify each line is valid JSON
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	for i, line := range lines {
		var parsed Event
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i, err)
		}
	}
}

func TestLogEvent_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "subdir", "nested", "events.jsonl")
	logger := NewLogger(logPath)

	event := Event{
		Type:      "session.spawned",
		SessionID: "ses_123",
		Timestamp: time.Now().Unix(),
	}

	err := logger.Log(event)
	if err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log() did not create nested directories")
	}
}

func TestEventSerialization(t *testing.T) {
	event := Event{
		Type:      "session.spawned",
		SessionID: "ses_abc",
		Timestamp: 1703001600,
		Data: map[string]interface{}{
			"prompt": "test prompt",
			"title":  "test title",
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var parsed Event
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if parsed.Type != event.Type {
		t.Errorf("Type = %v, want %v", parsed.Type, event.Type)
	}
	if parsed.SessionID != event.SessionID {
		t.Errorf("SessionID = %v, want %v", parsed.SessionID, event.SessionID)
	}
	if parsed.Timestamp != event.Timestamp {
		t.Errorf("Timestamp = %v, want %v", parsed.Timestamp, event.Timestamp)
	}
}

func TestEventSerialization_OmitsEmptyFields(t *testing.T) {
	event := Event{
		Type:      "session.spawned",
		Timestamp: 1703001600,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// session_id should be omitted when empty
	if strings.Contains(string(data), "session_id") {
		t.Error("Expected session_id to be omitted when empty")
	}

	// data should be omitted when nil
	if strings.Contains(string(data), `"data"`) {
		t.Error("Expected data to be omitted when nil")
	}
}

// Test event types per schema
func TestEventTypes_SessionSpawned(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	event := Event{
		Type:      EventTypeSessionSpawned,
		SessionID: "ses_123",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"prompt": "say hello",
			"title":  "orch-go-test",
		},
	}

	if err := logger.Log(event); err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(data), "session.spawned") {
		t.Error("Expected event type 'session.spawned'")
	}
}

func TestEventTypes_SessionCompleted(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	event := Event{
		Type:      EventTypeSessionCompleted,
		SessionID: "ses_123",
		Timestamp: time.Now().Unix(),
	}

	if err := logger.Log(event); err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(data), "session.completed") {
		t.Error("Expected event type 'session.completed'")
	}
}

func TestEventTypes_SessionError(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	event := Event{
		Type:      EventTypeSessionError,
		SessionID: "ses_123",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"error": "connection timeout",
		},
	}

	if err := logger.Log(event); err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(data), "session.error") {
		t.Error("Expected event type 'session.error'")
	}
}

// Test NewDefaultLogger helper
func TestNewDefaultLogger(t *testing.T) {
	logger := NewDefaultLogger()
	if logger == nil {
		t.Fatal("NewDefaultLogger() returned nil")
	}
	if !strings.Contains(logger.Path, ".orch") {
		t.Errorf("NewDefaultLogger() path = %v, should contain .orch", logger.Path)
	}
}

// Test LogSpawn helper
func TestLogSpawn(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogSpawn("ses_123", "test prompt", "test title")
	if err != nil {
		t.Fatalf("LogSpawn() error = %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(data), EventTypeSessionSpawned) {
		t.Error("LogSpawn() should log session.spawned event type")
	}
	if !strings.Contains(string(data), "ses_123") {
		t.Error("LogSpawn() should include session ID")
	}
	if !strings.Contains(string(data), "test prompt") {
		t.Error("LogSpawn() should include prompt in data")
	}
}

// Test LogCompleted helper
func TestLogCompleted(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogCompleted("ses_123")
	if err != nil {
		t.Fatalf("LogCompleted() error = %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(data), EventTypeSessionCompleted) {
		t.Error("LogCompleted() should log session.completed event type")
	}
	if !strings.Contains(string(data), "ses_123") {
		t.Error("LogCompleted() should include session ID")
	}
}

// Test LogError helper
func TestLogError(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogError("ses_123", "connection timeout")
	if err != nil {
		t.Fatalf("LogError() error = %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(data), EventTypeSessionError) {
		t.Error("LogError() should log session.error event type")
	}
	if !strings.Contains(string(data), "connection timeout") {
		t.Error("LogError() should include error message in data")
	}
}

// Test LogStatusChange helper
func TestLogStatusChange(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogStatusChange("ses_123", "idle")
	if err != nil {
		t.Fatalf("LogStatusChange() error = %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(data), EventTypeSessionStatus) {
		t.Error("LogStatusChange() should log session.status event type")
	}
	if !strings.Contains(string(data), "idle") {
		t.Error("LogStatusChange() should include status in data")
	}
}
