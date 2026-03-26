package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLogThinIssueDetected(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogThinIssueDetected(ThinIssueDetectedData{
		IssueID: "proj-123",
		Title:   "Fix auth bug",
	})
	if err != nil {
		t.Fatalf("LogThinIssueDetected() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if event.Type != EventTypeThinIssueDetected {
		t.Errorf("Type = %q, want %q", event.Type, EventTypeThinIssueDetected)
	}
	if event.SessionID != "proj-123" {
		t.Errorf("SessionID = %q, want %q", event.SessionID, "proj-123")
	}
	if event.Data["issue_id"] != "proj-123" {
		t.Errorf("issue_id = %q, want %q", event.Data["issue_id"], "proj-123")
	}
	if event.Data["title"] != "Fix auth bug" {
		t.Errorf("title = %q, want %q", event.Data["title"], "Fix auth bug")
	}
}
