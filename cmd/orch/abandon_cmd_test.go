package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/state"
)

func TestResolveAbandonIdentifierWorkspaceName(t *testing.T) {
	projectDir := t.TempDir()
	workspace := filepath.Join(projectDir, ".orch", "workspace", "og-debug-stuck-09feb-ab12")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".beads_id"), []byte("orch-go-abc123\n"), 0644); err != nil {
		t.Fatalf("write .beads_id failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".session_id"), []byte("ses_abc123xyz\n"), 0644); err != nil {
		t.Fatalf("write .session_id failed: %v", err)
	}

	r, err := resolveAbandonIdentifier(projectDir, "og-debug-stuck-09feb-ab12")
	if err != nil {
		t.Fatalf("resolveAbandonIdentifier returned error: %v", err)
	}
	if r.BeadsID != "orch-go-abc123" {
		t.Fatalf("BeadsID = %q, want %q", r.BeadsID, "orch-go-abc123")
	}
	if r.AgentName != "og-debug-stuck-09feb-ab12" {
		t.Fatalf("AgentName = %q, want %q", r.AgentName, "og-debug-stuck-09feb-ab12")
	}
	if r.SessionID != "ses_abc123xyz" {
		t.Fatalf("SessionID = %q, want %q", r.SessionID, "ses_abc123xyz")
	}
}

func TestResolveAbandonIdentifierSessionIDFromWorkspaceScan(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projectDir := t.TempDir()
	workspace := filepath.Join(projectDir, ".orch", "workspace", "og-debug-stuck-09feb-ab12")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".beads_id"), []byte("orch-go-abc123\n"), 0644); err != nil {
		t.Fatalf("write .beads_id failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".session_id"), []byte("ses_abc123xyz\n"), 0644); err != nil {
		t.Fatalf("write .session_id failed: %v", err)
	}

	r, err := resolveAbandonIdentifier(projectDir, "ses_abc123xyz")
	if err != nil {
		t.Fatalf("resolveAbandonIdentifier returned error: %v", err)
	}
	if r.BeadsID != "orch-go-abc123" {
		t.Fatalf("BeadsID = %q, want %q", r.BeadsID, "orch-go-abc123")
	}
	if r.AgentName != "og-debug-stuck-09feb-ab12" {
		t.Fatalf("AgentName = %q, want %q", r.AgentName, "og-debug-stuck-09feb-ab12")
	}
}

func TestResolveAbandonIdentifierUntrackedDisplayHandle(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	db, err := state.OpenDefault()
	if err != nil {
		t.Fatalf("OpenDefault failed: %v", err)
	}
	if db == nil {
		t.Fatal("OpenDefault returned nil db")
	}
	defer db.Close()

	beadsID := "orch-go-untracked-1768090360"
	err = db.UpsertAgent(&state.Agent{
		WorkspaceName: "og-debug-untracked-09feb",
		BeadsID:       beadsID,
		ProjectDir:    filepath.Join(home, "Documents", "personal", "orch-go"),
		ProjectName:   "orch-go",
		Mode:          "opencode",
		SpawnTime:     time.Now().UnixMilli(),
	})
	if err != nil {
		t.Fatalf("UpsertAgent failed: %v", err)
	}

	r, err := resolveAbandonIdentifier(t.TempDir(), formatBeadsIDForDisplay(beadsID))
	if err != nil {
		t.Fatalf("resolveAbandonIdentifier returned error: %v", err)
	}
	if r.BeadsID != beadsID {
		t.Fatalf("BeadsID = %q, want %q", r.BeadsID, beadsID)
	}
}

func TestLogStructuredEventIncludesAttemptID(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "ws")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	attemptID := "123e4567-e89b-42d3-a456-426614174000"
	if err := os.WriteFile(filepath.Join(workspace, ".attempt_id"), []byte(attemptID+"\n"), 0644); err != nil {
		t.Fatalf("write .attempt_id failed: %v", err)
	}

	logger := events.NewLogger(filepath.Join(root, "events.jsonl"))
	ctx := &abandonContext{
		BeadsID:       "orch-go-abc123",
		AgentName:     "og-debug-stuck-09feb-ab12",
		WorkspacePath: workspace,
	}

	logStructuredEvent(logger, ctx)

	content, err := os.ReadFile(filepath.Join(root, "events.jsonl"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var event map[string]interface{}
	if err := json.Unmarshal(content[:len(content)-1], &event); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	data, ok := event["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("event data missing or malformed: %v", event)
	}
	if got, _ := data["attempt_id"].(string); got != attemptID {
		t.Fatalf("attempt_id = %q, want %q", got, attemptID)
	}
}

func TestLogAbandonTelemetryIncludesAttemptID(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "ws")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	attemptID := "123e4567-e89b-42d3-a456-426614174000"
	if err := os.WriteFile(filepath.Join(workspace, ".attempt_id"), []byte(attemptID+"\n"), 0644); err != nil {
		t.Fatalf("write .attempt_id failed: %v", err)
	}

	logger := events.NewLogger(filepath.Join(root, "events.jsonl"))
	ctx := &abandonContext{
		BeadsID:       "orch-go-abc123",
		AgentName:     "og-debug-stuck-09feb-ab12",
		WorkspacePath: workspace,
	}

	logAbandonTelemetry(logger, ctx)

	content, err := os.ReadFile(filepath.Join(root, "events.jsonl"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var event map[string]interface{}
	if err := json.Unmarshal(content[:len(content)-1], &event); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	data, ok := event["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("event data missing or malformed: %v", event)
	}
	if got, _ := data["attempt_id"].(string); got != attemptID {
		t.Fatalf("attempt_id = %q, want %q", got, attemptID)
	}
}
