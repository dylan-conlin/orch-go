package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLogKBContextTimeout(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "events.jsonl")
	logger := NewLogger(logPath)

	err := logger.LogKBContextTimeout(KBContextTimeoutData{
		Query:       "wire context timeout detection",
		ProjectDir:  "/tmp/orch-go",
		Skill:       "feature-impl",
		BeadsID:     "orch-go-r0pu5",
		WorkspaceID: "og-feat-wire-kb-context-27mar-7ee3",
	})
	if err != nil {
		t.Fatalf("LogKBContextTimeout() error = %v", err)
	}

	data, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if event.Type != EventTypeKBContextTimeout {
		t.Errorf("Type = %q, want %q", event.Type, EventTypeKBContextTimeout)
	}
	if event.SessionID != "og-feat-wire-kb-context-27mar-7ee3" {
		t.Errorf("SessionID = %q, want %q", event.SessionID, "og-feat-wire-kb-context-27mar-7ee3")
	}
	if event.Data["query"] != "wire context timeout detection" {
		t.Errorf("query = %v, want %q", event.Data["query"], "wire context timeout detection")
	}
	if event.Data["project_dir"] != "/tmp/orch-go" {
		t.Errorf("project_dir = %v, want %q", event.Data["project_dir"], "/tmp/orch-go")
	}
	if event.Data["skill"] != "feature-impl" {
		t.Errorf("skill = %v, want %q", event.Data["skill"], "feature-impl")
	}
	if event.Data["beads_id"] != "orch-go-r0pu5" {
		t.Errorf("beads_id = %v, want %q", event.Data["beads_id"], "orch-go-r0pu5")
	}
	if event.Data["workspace_id"] != "og-feat-wire-kb-context-27mar-7ee3" {
		t.Errorf("workspace_id = %v, want %q", event.Data["workspace_id"], "og-feat-wire-kb-context-27mar-7ee3")
	}
}
