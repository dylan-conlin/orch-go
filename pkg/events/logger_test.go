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

func TestDefaultLogPath_EnvOverride(t *testing.T) {
	// ORCH_EVENTS_PATH overrides the default log path, preventing test leakage
	// into production events.jsonl.
	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "custom-events.jsonl")

	original := os.Getenv("ORCH_EVENTS_PATH")
	os.Setenv("ORCH_EVENTS_PATH", customPath)
	defer os.Setenv("ORCH_EVENTS_PATH", original)

	got := DefaultLogPath()
	if got != customPath {
		t.Errorf("DefaultLogPath() = %q, want %q (ORCH_EVENTS_PATH override)", got, customPath)
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
	data, err := os.ReadFile(logger.CurrentPath())
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
	data, err := os.ReadFile(logger.CurrentPath())
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

	// Verify rotated file exists in nested directory
	if _, err := os.Stat(logger.CurrentPath()); os.IsNotExist(err) {
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

	data, err := os.ReadFile(logger.CurrentPath())
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

	data, err := os.ReadFile(logger.CurrentPath())
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

	data, err := os.ReadFile(logger.CurrentPath())
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

	data, err := os.ReadFile(logger.CurrentPath())
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

	data, err := os.ReadFile(logger.CurrentPath())
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

	data, err := os.ReadFile(logger.CurrentPath())
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

	data, err := os.ReadFile(logger.CurrentPath())
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

// Test LogVerificationBypassed helper
func TestLogVerificationBypassed(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogVerificationBypassed(VerificationBypassedData{
		BeadsID:   "orch-go-abc1",
		Workspace: "og-feat-test-14jan",
		Gate:      "test_evidence",
		Reason:    "Tests run in CI pipeline",
		Skill:     "feature-impl",
	})
	if err != nil {
		t.Fatalf("LogVerificationBypassed() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Verify event type
	if !strings.Contains(string(data), EventTypeVerificationBypassed) {
		t.Error("LogVerificationBypassed() should log verification.bypassed event type")
	}
	// Verify gate
	if !strings.Contains(string(data), "test_evidence") {
		t.Error("LogVerificationBypassed() should include gate in data")
	}
	// Verify reason
	if !strings.Contains(string(data), "Tests run in CI pipeline") {
		t.Error("LogVerificationBypassed() should include reason in data")
	}
	// Verify beads_id
	if !strings.Contains(string(data), "orch-go-abc1") {
		t.Error("LogVerificationBypassed() should include beads_id in data")
	}
	// Verify workspace
	if !strings.Contains(string(data), "og-feat-test-14jan") {
		t.Error("LogVerificationBypassed() should include workspace in data")
	}
	// Verify skill
	if !strings.Contains(string(data), "feature-impl") {
		t.Error("LogVerificationBypassed() should include skill in data")
	}
}

// Test LogVerificationBypassed with minimal data
func TestLogVerificationBypassed_Minimal(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogVerificationBypassed(VerificationBypassedData{
		Gate:   "git_diff",
		Reason: "Docs-only change",
	})
	if err != nil {
		t.Fatalf("LogVerificationBypassed() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Verify required fields
	if !strings.Contains(string(data), "git_diff") {
		t.Error("LogVerificationBypassed() should include gate")
	}
	if !strings.Contains(string(data), "Docs-only change") {
		t.Error("LogVerificationBypassed() should include reason")
	}
}

// Test VerificationBypassedData serialization
func TestVerificationBypassedDataSerialization(t *testing.T) {
	data := VerificationBypassedData{
		BeadsID:   "orch-go-abc1",
		Workspace: "og-feat-test",
		Gate:      "test_evidence",
		Reason:    "Test reason",
		Skill:     "feature-impl",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var parsed VerificationBypassedData
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if parsed.Gate != data.Gate {
		t.Errorf("Gate = %v, want %v", parsed.Gate, data.Gate)
	}
	if parsed.Reason != data.Reason {
		t.Errorf("Reason = %v, want %v", parsed.Reason, data.Reason)
	}
	if parsed.BeadsID != data.BeadsID {
		t.Errorf("BeadsID = %v, want %v", parsed.BeadsID, data.BeadsID)
	}
}

// Test PipelineStepTiming serialization
func TestPipelineStepTimingSerialization(t *testing.T) {
	steps := []PipelineStepTiming{
		{Name: "hotspot", DurationMs: 42},
		{Name: "duplication", Skipped: true, SkipReason: "orchestrator"},
		{Name: "model_impact", DurationMs: 15},
		{Name: "auto_rebuild", Skipped: true, SkipReason: "no_go_changes"},
	}

	jsonData, err := json.Marshal(steps)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var parsed []PipelineStepTiming
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(parsed) != 4 {
		t.Fatalf("Expected 4 steps, got %d", len(parsed))
	}

	// Verify executed step
	if parsed[0].Name != "hotspot" || parsed[0].DurationMs != 42 || parsed[0].Skipped {
		t.Errorf("Step 0: got %+v, want hotspot/42ms/not-skipped", parsed[0])
	}

	// Verify skipped step has reason and no duration
	if parsed[1].Name != "duplication" || !parsed[1].Skipped || parsed[1].SkipReason != "orchestrator" {
		t.Errorf("Step 1: got %+v, want duplication/skipped/orchestrator", parsed[1])
	}

	// Verify skip_reason is omitted when not skipped
	raw := string(jsonData)
	if !strings.Contains(raw, `"skip_reason":"orchestrator"`) {
		t.Error("Expected skip_reason for skipped step")
	}
}

// Test LogAgentCompleted includes pipeline timing in event
func TestLogAgentCompleted_PipelineTiming(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogAgentCompleted(AgentCompletedData{
		BeadsID:   "orch-go-test1",
		Workspace: "og-feat-test",
		Reason:    "Completed",
		Outcome:   "success",
		PipelineTiming: []PipelineStepTiming{
			{Name: "hotspot", DurationMs: 120},
			{Name: "duplication", Skipped: true, SkipReason: "no_project_dir"},
			{Name: "model_impact", DurationMs: 30},
			{Name: "auto_rebuild", DurationMs: 5400},
		},
		PipelineTotalMs: 200,
	})
	if err != nil {
		t.Fatalf("LogAgentCompleted() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	raw := string(data)

	// Verify pipeline_timing is present
	if !strings.Contains(raw, "pipeline_timing") {
		t.Error("Expected pipeline_timing in event data")
	}
	if !strings.Contains(raw, "pipeline_total_ms") {
		t.Error("Expected pipeline_total_ms in event data")
	}

	// Verify step names are present
	for _, name := range []string{"hotspot", "duplication", "model_impact", "auto_rebuild"} {
		if !strings.Contains(raw, name) {
			t.Errorf("Expected step name %q in event data", name)
		}
	}

	// Verify skip_reason is present for skipped step
	if !strings.Contains(raw, "no_project_dir") {
		t.Error("Expected skip_reason 'no_project_dir' in event data")
	}

	// Parse and verify structure
	var event Event
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &event); err != nil {
		t.Fatalf("Failed to parse event JSON: %v", err)
	}
	if event.Type != EventTypeAgentCompleted {
		t.Errorf("Event type = %v, want %v", event.Type, EventTypeAgentCompleted)
	}
}

// Test LogDuplicationDetected logs event with match details
func TestLogDuplicationDetected(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogDuplicationDetected(DuplicationDetectedData{
		BeadsID:   "orch-go-abc1",
		Workspace: "og-feat-test",
		Count:     2,
		Matches: []DuplicationMatch{
			{
				FileA:      "cmd/orch/spawn.go",
				FuncA:      "spawnAgent",
				FileB:      "cmd/orch/work.go",
				FuncB:      "workAgent",
				Similarity: 0.92,
			},
			{
				FileA:      "pkg/verify/check.go",
				FuncA:      "validatePhase",
				FileB:      "pkg/verify/update.go",
				FuncB:      "checkPhase",
				Similarity: 0.87,
			},
		},
	})
	if err != nil {
		t.Fatalf("LogDuplicationDetected() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	raw := string(data)

	// Verify event type
	if !strings.Contains(raw, EventTypeDuplicationDetected) {
		t.Error("Expected event type 'duplication.detected'")
	}

	// Verify match data
	if !strings.Contains(raw, "spawnAgent") {
		t.Error("Expected function name 'spawnAgent' in matches")
	}
	if !strings.Contains(raw, "workAgent") {
		t.Error("Expected function name 'workAgent' in matches")
	}
	if !strings.Contains(raw, "0.92") {
		t.Error("Expected similarity score 0.92 in matches")
	}
	if !strings.Contains(raw, "cmd/orch/spawn.go") {
		t.Error("Expected file path in matches")
	}

	// Verify beads_id and workspace
	if !strings.Contains(raw, "orch-go-abc1") {
		t.Error("Expected beads_id in event")
	}
	if !strings.Contains(raw, "og-feat-test") {
		t.Error("Expected workspace in event")
	}

	// Parse and verify structure
	var event Event
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &event); err != nil {
		t.Fatalf("Failed to parse event JSON: %v", err)
	}
	if event.Type != EventTypeDuplicationDetected {
		t.Errorf("Event type = %v, want %v", event.Type, EventTypeDuplicationDetected)
	}
	count, ok := event.Data["count"].(float64)
	if !ok || int(count) != 2 {
		t.Errorf("Expected count=2, got %v", event.Data["count"])
	}
}

// Test LogDuplicationDetected with minimal data (no beads/workspace)
func TestLogDuplicationDetected_Minimal(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogDuplicationDetected(DuplicationDetectedData{
		Count: 1,
		Matches: []DuplicationMatch{
			{
				FileA:      "a.go",
				FuncA:      "foo",
				FileB:      "b.go",
				FuncB:      "bar",
				Similarity: 0.95,
			},
		},
	})
	if err != nil {
		t.Fatalf("LogDuplicationDetected() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	raw := string(data)
	// beads_id and workspace should be omitted
	if strings.Contains(raw, "beads_id") {
		t.Error("Expected beads_id to be omitted when empty")
	}
	if strings.Contains(raw, "workspace") {
		t.Error("Expected workspace to be omitted when empty")
	}
	// matches should be present
	if !strings.Contains(raw, "foo") {
		t.Error("Expected function name in matches")
	}
}

// Test LogDuplicationSuppressed logs allowlist-suppressed pairs
func TestLogDuplicationSuppressed(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogDuplicationSuppressed(DuplicationSuppressedData{
		BeadsID:   "orch-go-abc1",
		Workspace: "og-feat-test",
		Count:     2,
		Matches: []DuplicationSuppressedMatch{
			{
				FuncA:      "(Logger).LogSpawn",
				FuncB:      "(Logger).LogCompleted",
				Similarity: 0.95,
				Pattern:    "(Logger).Log*",
			},
			{
				FuncA:      "handleOrphanResult",
				FuncB:      "handleRecoveryResult",
				Similarity: 0.88,
				Pattern:    "handle*Result",
			},
		},
	})
	if err != nil {
		t.Fatalf("LogDuplicationSuppressed() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	raw := string(data)

	// Verify event type
	if !strings.Contains(raw, EventTypeDuplicationSuppressed) {
		t.Error("Expected event type 'duplication.suppressed'")
	}

	// Verify match data includes pattern
	if !strings.Contains(raw, "(Logger).Log*") {
		t.Error("Expected pattern '(Logger).Log*' in matches")
	}
	if !strings.Contains(raw, "handle*Result") {
		t.Error("Expected pattern 'handle*Result' in matches")
	}
	if !strings.Contains(raw, "(Logger).LogSpawn") {
		t.Error("Expected function name in matches")
	}

	// Parse and verify structure
	var event Event
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &event); err != nil {
		t.Fatalf("Failed to parse event JSON: %v", err)
	}
	if event.Type != EventTypeDuplicationSuppressed {
		t.Errorf("Event type = %v, want %v", event.Type, EventTypeDuplicationSuppressed)
	}
	count, ok := event.Data["count"].(float64)
	if !ok || int(count) != 2 {
		t.Errorf("Expected count=2, got %v", event.Data["count"])
	}
}

// Test LogDuplicationSuppressed with minimal data
func TestLogDuplicationSuppressed_Minimal(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogDuplicationSuppressed(DuplicationSuppressedData{
		Count: 1,
		Matches: []DuplicationSuppressedMatch{
			{
				FuncA:      "foo",
				FuncB:      "bar",
				Similarity: 0.90,
				Pattern:    "*",
			},
		},
	})
	if err != nil {
		t.Fatalf("LogDuplicationSuppressed() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	raw := string(data)
	// beads_id and workspace should be omitted
	if strings.Contains(raw, "beads_id") {
		t.Error("Expected beads_id to be omitted when empty")
	}
	if strings.Contains(raw, "workspace") {
		t.Error("Expected workspace to be omitted when empty")
	}
}

// Test LogAgentCompleted omits pipeline timing when empty
func TestLogAgentCompleted_NoPipelineTiming(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogAgentCompleted(AgentCompletedData{
		BeadsID: "orch-go-test2",
		Reason:  "Completed",
		Outcome: "success",
	})
	if err != nil {
		t.Fatalf("LogAgentCompleted() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	raw := string(data)
	if strings.Contains(raw, "pipeline_timing") {
		t.Error("Expected pipeline_timing to be omitted when empty")
	}
	if strings.Contains(raw, "pipeline_total_ms") {
		t.Error("Expected pipeline_total_ms to be omitted when zero")
	}
}

// Test LogAgentCompleted includes verification_level when set
func TestLogAgentCompleted_VerificationLevel(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogAgentCompleted(AgentCompletedData{
		BeadsID:           "orch-go-test-vlevel",
		Workspace:         "og-feat-test",
		Reason:            "Completed",
		Outcome:           "success",
		Skill:             "feature-impl",
		VerificationLevel: "V2",
	})
	if err != nil {
		t.Fatalf("LogAgentCompleted() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	raw := string(data)
	if !strings.Contains(raw, `"verification_level":"V2"`) {
		t.Errorf("Expected verification_level:V2 in event data, got: %s", raw)
	}
}

// Test LogAgentCompleted omits verification_level when empty
func TestLogAgentCompleted_NoVerificationLevel(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogAgentCompleted(AgentCompletedData{
		BeadsID: "orch-go-test-no-vlevel",
		Reason:  "Completed",
		Outcome: "success",
	})
	if err != nil {
		t.Fatalf("LogAgentCompleted() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	raw := string(data)
	if strings.Contains(raw, "verification_level") {
		t.Error("Expected verification_level to be omitted when empty")
	}
}

// Test LogVerificationFailed includes verification_level
func TestLogVerificationFailed_VerificationLevel(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogVerificationFailed(VerificationFailedData{
		BeadsID:           "orch-go-vfail",
		GatesFailed:       []string{"test_evidence"},
		Errors:            []string{"no test evidence"},
		Skill:             "feature-impl",
		VerificationLevel: "V2",
	})
	if err != nil {
		t.Fatalf("LogVerificationFailed() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	raw := string(data)
	if !strings.Contains(raw, `"verification_level":"V2"`) {
		t.Errorf("Expected verification_level:V2 in event data, got: %s", raw)
	}
}

// Test LogVerificationBypassed includes verification_level
func TestLogVerificationBypassed_VerificationLevel(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogVerificationBypassed(VerificationBypassedData{
		BeadsID:           "orch-go-vbypass",
		Gate:              "test_evidence",
		Reason:            "no tests needed",
		Skill:             "investigation",
		VerificationLevel: "V1",
	})
	if err != nil {
		t.Fatalf("LogVerificationBypassed() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	raw := string(data)
	if !strings.Contains(raw, `"verification_level":"V1"`) {
		t.Errorf("Expected verification_level:V1 in event data, got: %s", raw)
	}
}

func TestLogArchitectEscalation_Escalated(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogArchitectEscalation(ArchitectEscalationData{
		IssueID:     "proj-123",
		HotspotFile: "pkg/daemon/daemon.go",
		HotspotType: "fix-density",
		Escalated:   true,
	})
	if err != nil {
		t.Fatalf("LogArchitectEscalation() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if event.Type != EventTypeDaemonArchitectEscalation {
		t.Errorf("event.Type = %q, want %q", event.Type, EventTypeDaemonArchitectEscalation)
	}
	if event.SessionID != "proj-123" {
		t.Errorf("event.SessionID = %q, want %q", event.SessionID, "proj-123")
	}
	if event.Data["issue_id"] != "proj-123" {
		t.Errorf("data.issue_id = %v, want %q", event.Data["issue_id"], "proj-123")
	}
	if event.Data["hotspot_file"] != "pkg/daemon/daemon.go" {
		t.Errorf("data.hotspot_file = %v, want %q", event.Data["hotspot_file"], "pkg/daemon/daemon.go")
	}
	if event.Data["hotspot_type"] != "fix-density" {
		t.Errorf("data.hotspot_type = %v, want %q", event.Data["hotspot_type"], "fix-density")
	}
	if event.Data["escalated"] != true {
		t.Errorf("data.escalated = %v, want true", event.Data["escalated"])
	}
	if _, ok := event.Data["prior_architect_ref"]; ok {
		t.Error("data.prior_architect_ref should be omitted when empty")
	}
}

func TestLogArchitectEscalation_SkippedWithPriorRef(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogArchitectEscalation(ArchitectEscalationData{
		IssueID:           "proj-456",
		HotspotFile:       "cmd/orch/spawn_cmd.go",
		HotspotType:       "bloat-size",
		Escalated:         false,
		PriorArchitectRef: "orch-go-1119",
	})
	if err != nil {
		t.Fatalf("LogArchitectEscalation() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if event.Data["escalated"] != false {
		t.Errorf("data.escalated = %v, want false", event.Data["escalated"])
	}
	if event.Data["prior_architect_ref"] != "orch-go-1119" {
		t.Errorf("data.prior_architect_ref = %v, want %q", event.Data["prior_architect_ref"], "orch-go-1119")
	}
}

func TestLogGateDecision_Block(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogGateDecision(GateDecisionData{
		GateName:    "hotspot",
		Decision:    "block",
		Skill:       "feature-impl",
		BeadsID:     "orch-go-abc1",
		TargetFiles: []string{"cmd/orch/spawn_cmd.go", "cmd/orch/daemon.go"},
		Reason:      "CRITICAL hotspot: cmd/orch/spawn_cmd.go exceeds 1500 lines",
	})
	if err != nil {
		t.Fatalf("LogGateDecision() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if event.Type != EventTypeSpawnGateDecision {
		t.Errorf("event.Type = %q, want %q", event.Type, EventTypeSpawnGateDecision)
	}
	if event.Data["gate_name"] != "hotspot" {
		t.Errorf("data.gate_name = %v, want %q", event.Data["gate_name"], "hotspot")
	}
	if event.Data["decision"] != "block" {
		t.Errorf("data.decision = %v, want %q", event.Data["decision"], "block")
	}
	if event.Data["skill"] != "feature-impl" {
		t.Errorf("data.skill = %v, want %q", event.Data["skill"], "feature-impl")
	}
	if event.SessionID != "orch-go-abc1" {
		t.Errorf("event.SessionID = %q, want %q", event.SessionID, "orch-go-abc1")
	}
}

func TestLogGateDecision_Bypass(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogGateDecision(GateDecisionData{
		GateName: "triage",
		Decision: "bypass",
		Skill:    "investigation",
		Reason:   "urgent one-off exploration",
	})
	if err != nil {
		t.Fatalf("LogGateDecision() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if event.Data["gate_name"] != "triage" {
		t.Errorf("data.gate_name = %v, want %q", event.Data["gate_name"], "triage")
	}
	if event.Data["decision"] != "bypass" {
		t.Errorf("data.decision = %v, want %q", event.Data["decision"], "bypass")
	}
	// beads_id should be omitted when empty
	if _, ok := event.Data["beads_id"]; ok {
		t.Error("data.beads_id should be omitted when empty")
	}
	// target_files should be omitted when empty
	if _, ok := event.Data["target_files"]; ok {
		t.Error("data.target_files should be omitted when empty")
	}
}

func TestLogExplorationDecomposed(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogExplorationDecomposed(ExplorationDecomposedData{
		BeadsID:       "orch-go-abc1",
		ParentSkill:   "investigation",
		Question:      "How does daemon handle concurrent spawns?",
		Subproblems:   []string{"mutex locking", "queue ordering", "rate limiting"},
		Breadth:       3,
	})
	if err != nil {
		t.Fatalf("LogExplorationDecomposed() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if event.Type != EventTypeExplorationDecomposed {
		t.Errorf("event.Type = %q, want %q", event.Type, EventTypeExplorationDecomposed)
	}
	if event.SessionID != "orch-go-abc1" {
		t.Errorf("event.SessionID = %q, want %q", event.SessionID, "orch-go-abc1")
	}
	if event.Data["parent_skill"] != "investigation" {
		t.Errorf("data.parent_skill = %v, want %q", event.Data["parent_skill"], "investigation")
	}
	if event.Data["breadth"] != float64(3) {
		t.Errorf("data.breadth = %v, want 3", event.Data["breadth"])
	}
}

func TestLogExplorationJudged(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogExplorationJudged(ExplorationJudgedData{
		BeadsID:           "orch-go-abc1",
		ParentSkill:       "investigation",
		TotalFindings:     8,
		Accepted:          5,
		Contested:         2,
		Rejected:          1,
		CoverageGaps:      1,
	})
	if err != nil {
		t.Fatalf("LogExplorationJudged() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if event.Type != EventTypeExplorationJudged {
		t.Errorf("event.Type = %q, want %q", event.Type, EventTypeExplorationJudged)
	}
	if event.Data["total_findings"] != float64(8) {
		t.Errorf("data.total_findings = %v, want 8", event.Data["total_findings"])
	}
	if event.Data["accepted"] != float64(5) {
		t.Errorf("data.accepted = %v, want 5", event.Data["accepted"])
	}
	if event.Data["contested"] != float64(2) {
		t.Errorf("data.contested = %v, want 2", event.Data["contested"])
	}
	if event.Data["rejected"] != float64(1) {
		t.Errorf("data.rejected = %v, want 1", event.Data["rejected"])
	}
}

func TestLogExplorationSynthesized(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogExplorationSynthesized(ExplorationSynthesizedData{
		BeadsID:          "orch-go-abc1",
		ParentSkill:      "investigation",
		WorkerCount:      3,
		DurationSeconds:  450,
		SynthesisPath:    ".orch/workspace/test/SYNTHESIS.md",
	})
	if err != nil {
		t.Fatalf("LogExplorationSynthesized() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if event.Type != EventTypeExplorationSynthesized {
		t.Errorf("event.Type = %q, want %q", event.Type, EventTypeExplorationSynthesized)
	}
	if event.Data["worker_count"] != float64(3) {
		t.Errorf("data.worker_count = %v, want 3", event.Data["worker_count"])
	}
	if event.Data["duration_seconds"] != float64(450) {
		t.Errorf("data.duration_seconds = %v, want 450", event.Data["duration_seconds"])
	}
}

func TestLogExplorationIterated(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogExplorationIterated(ExplorationIteratedData{
		BeadsID:       "orch-go-iter1",
		ParentSkill:   "investigation",
		Iteration:     2,
		GapsAddressed: 2,
		NewWorkers:    2,
	})
	if err != nil {
		t.Fatalf("LogExplorationIterated() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if event.Type != EventTypeExplorationIterated {
		t.Errorf("event.Type = %q, want %q", event.Type, EventTypeExplorationIterated)
	}
	if event.Data["iteration"] != float64(2) {
		t.Errorf("data.iteration = %v, want 2", event.Data["iteration"])
	}
	if event.Data["gaps_addressed"] != float64(2) {
		t.Errorf("data.gaps_addressed = %v, want 2", event.Data["gaps_addressed"])
	}
	if event.Data["new_workers"] != float64(2) {
		t.Errorf("data.new_workers = %v, want 2", event.Data["new_workers"])
	}
	if event.Data["beads_id"] != "orch-go-iter1" {
		t.Errorf("data.beads_id = %v, want orch-go-iter1", event.Data["beads_id"])
	}
}

func TestLogDecisionMade(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogDecisionMade(DecisionMadeData{
		DecisionID:      "dec-abc123",
		Class:           "auto_complete_light",
		Category:        "completion",
		Tier:            "propose-and-act",
		BaseTier:        "propose-and-act",
		ComplianceLevel: "standard",
		Target:          "orch-go-xyz1",
		Reason:          "Light-tier agent reported Phase: Complete",
		Outcome:         "executed",
	})
	if err != nil {
		t.Fatalf("LogDecisionMade() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if event.Type != EventTypeDecisionMade {
		t.Errorf("event.Type = %q, want %q", event.Type, EventTypeDecisionMade)
	}
	if event.SessionID != "orch-go-xyz1" {
		t.Errorf("event.SessionID = %q, want %q", event.SessionID, "orch-go-xyz1")
	}
	if event.Data["decision_id"] != "dec-abc123" {
		t.Errorf("data.decision_id = %v, want %q", event.Data["decision_id"], "dec-abc123")
	}
	if event.Data["class"] != "auto_complete_light" {
		t.Errorf("data.class = %v, want %q", event.Data["class"], "auto_complete_light")
	}
	if event.Data["category"] != "completion" {
		t.Errorf("data.category = %v, want %q", event.Data["category"], "completion")
	}
	if event.Data["tier"] != "propose-and-act" {
		t.Errorf("data.tier = %v, want %q", event.Data["tier"], "propose-and-act")
	}
	if event.Data["base_tier"] != "propose-and-act" {
		t.Errorf("data.base_tier = %v, want %q", event.Data["base_tier"], "propose-and-act")
	}
	if event.Data["compliance_level"] != "standard" {
		t.Errorf("data.compliance_level = %v, want %q", event.Data["compliance_level"], "standard")
	}
	if event.Data["target"] != "orch-go-xyz1" {
		t.Errorf("data.target = %v, want %q", event.Data["target"], "orch-go-xyz1")
	}
	if event.Data["reason"] != "Light-tier agent reported Phase: Complete" {
		t.Errorf("data.reason = %v, want expected string", event.Data["reason"])
	}
	if event.Data["outcome"] != "executed" {
		t.Errorf("data.outcome = %v, want %q", event.Data["outcome"], "executed")
	}
}

func TestLogDecisionMade_Minimal(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogDecisionMade(DecisionMadeData{
		DecisionID:      "dec-min1",
		Class:           "select_issue",
		Category:        "spawn",
		Tier:            "autonomous",
		BaseTier:        "autonomous",
		ComplianceLevel: "standard",
	})
	if err != nil {
		t.Fatalf("LogDecisionMade() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	raw := string(data)
	// Optional fields should be omitted when empty
	if strings.Contains(raw, `"target"`) {
		t.Error("Expected target to be omitted when empty")
	}
	if strings.Contains(raw, `"reason"`) {
		t.Error("Expected reason to be omitted when empty")
	}
	if strings.Contains(raw, `"outcome"`) {
		t.Error("Expected outcome to be omitted when empty")
	}
	// Required fields should always be present
	if !strings.Contains(raw, "decision.made") {
		t.Error("Expected event type 'decision.made'")
	}
	if !strings.Contains(raw, "dec-min1") {
		t.Error("Expected decision_id in event")
	}
}

func TestLogDecisionMade_ComplianceModulation(t *testing.T) {
	// Test that strict compliance shows different tier than base
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogDecisionMade(DecisionMadeData{
		DecisionID:      "dec-strict1",
		Class:           "select_issue",
		Category:        "spawn",
		Tier:            "propose-and-act",  // promoted from autonomous by strict
		BaseTier:        "autonomous",
		ComplianceLevel: "strict",
		Target:          "orch-go-test1",
	})
	if err != nil {
		t.Fatalf("LogDecisionMade() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	// Verify tier differs from base_tier (compliance modulation visible in event)
	if event.Data["tier"] == event.Data["base_tier"] {
		t.Error("Expected tier to differ from base_tier under strict compliance")
	}
	if event.Data["tier"] != "propose-and-act" {
		t.Errorf("data.tier = %v, want propose-and-act", event.Data["tier"])
	}
	if event.Data["base_tier"] != "autonomous" {
		t.Errorf("data.base_tier = %v, want autonomous", event.Data["base_tier"])
	}
}

func TestLogGateDecision_AccretionPrecommit(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogGateDecision(GateDecisionData{
		GateName:    "accretion_precommit",
		Decision:    "block",
		TargetFiles: []string{"cmd/orch/stats_cmd.go"},
		Reason:      "file exceeds accretion threshold",
	})
	if err != nil {
		t.Fatalf("LogGateDecision() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if event.Data["gate_name"] != "accretion_precommit" {
		t.Errorf("data.gate_name = %v, want %q", event.Data["gate_name"], "accretion_precommit")
	}
	// skill should be omitted for precommit (no spawn context)
	if _, ok := event.Data["skill"]; ok {
		t.Error("data.skill should be omitted when empty")
	}
}

func TestLogAgentRejected(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogAgentRejected(AgentRejectedData{
		BeadsID:       "orch-go-test1",
		Reason:        "Tests missing edge cases",
		Category:      "quality",
		OriginalSkill: "feature-impl",
		OriginalModel: "claude-opus-4-5-20251101",
	})
	if err != nil {
		t.Fatalf("LogAgentRejected() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	raw := string(data)

	if !strings.Contains(raw, "agent.rejected") {
		t.Error("Expected event type agent.rejected")
	}
	if !strings.Contains(raw, "orch-go-test1") {
		t.Error("Expected beads_id in event")
	}
	if !strings.Contains(raw, "quality") {
		t.Error("Expected category in event")
	}
	if !strings.Contains(raw, "feature-impl") {
		t.Error("Expected original_skill in event")
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to parse event: %v", err)
	}
	if event.Type != EventTypeAgentRejected {
		t.Errorf("Expected type %q, got %q", EventTypeAgentRejected, event.Type)
	}
}

func TestLogAgentRejected_OmitsEmptyOptionals(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogAgentRejected(AgentRejectedData{
		BeadsID:  "orch-go-test2",
		Reason:   "Bad approach",
		Category: "approach",
	})
	if err != nil {
		t.Fatalf("LogAgentRejected() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to parse event: %v", err)
	}

	if _, ok := event.Data["original_skill"]; ok {
		t.Error("original_skill should be omitted when empty")
	}
	if _, ok := event.Data["original_model"]; ok {
		t.Error("original_model should be omitted when empty")
	}
}

func TestLogEmptyExecutionRetry(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogEmptyExecutionRetry("ses_456", EmptyExecutionRetryData{
		BeadsID:        "orch-go-abc12",
		Attempt:        3,
		Classification: "empty-execution",
		Reason:         "zero output tokens and no substantive content",
		RecoveryResult: "retrying",
	})
	if err != nil {
		t.Fatalf("LogEmptyExecutionRetry() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	raw := string(data)
	if !strings.Contains(raw, EventTypeEmptyExecutionRetry) {
		t.Error("Expected event type 'session.empty_execution_retry'")
	}

	var event Event
	if err := json.Unmarshal(data[:len(data)-1], &event); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if event.SessionID != "ses_456" {
		t.Errorf("SessionID = %q, want %q", event.SessionID, "ses_456")
	}
	if event.Data["beads_id"] != "orch-go-abc12" {
		t.Errorf("beads_id = %v, want %q", event.Data["beads_id"], "orch-go-abc12")
	}
	if event.Data["attempt"] != float64(3) {
		t.Errorf("attempt = %v, want 3", event.Data["attempt"])
	}
	if event.Data["classification"] != "empty-execution" {
		t.Errorf("classification = %v, want %q", event.Data["classification"], "empty-execution")
	}
	if event.Data["recovery"] != "retrying" {
		t.Errorf("recovery = %v, want %q", event.Data["recovery"], "retrying")
	}
}

func TestLogEmptyExecutionRetry_WithoutBeadsID(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogEmptyExecutionRetry("ses_789", EmptyExecutionRetryData{
		Attempt:        1,
		Classification: "empty-execution",
		Reason:         "no assistant messages",
		RecoveryResult: "escalated",
	})
	if err != nil {
		t.Fatalf("LogEmptyExecutionRetry() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	raw := string(data)
	if strings.Contains(raw, `"beads_id"`) {
		t.Error("Expected beads_id to be omitted when empty")
	}
	if !strings.Contains(raw, `"recovery":"escalated"`) {
		t.Error("Expected recovery field 'escalated'")
	}
}
