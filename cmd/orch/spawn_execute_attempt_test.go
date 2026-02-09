package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestAddAttemptIDToEventDataUsesConfigAttemptID(t *testing.T) {
	eventData := map[string]interface{}{}
	cfg := &spawn.Config{AttemptID: "123e4567-e89b-42d3-a456-426614174000"}

	addAttemptIDToEventData(eventData, cfg)

	if got, _ := eventData["attempt_id"].(string); got != cfg.AttemptID {
		t.Fatalf("attempt_id = %q, want %q", got, cfg.AttemptID)
	}
}

func TestAddAttemptIDToEventDataFallsBackToWorkspaceFile(t *testing.T) {
	projectDir := t.TempDir()
	workspaceName := "og-feat-test-attempt-09feb-ab12"
	workspacePath := filepath.Join(projectDir, ".orch", "workspace", workspaceName)
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	attemptID := "123e4567-e89b-42d3-a456-426614174001"
	if err := os.WriteFile(filepath.Join(workspacePath, ".attempt_id"), []byte(attemptID+"\n"), 0644); err != nil {
		t.Fatalf("write .attempt_id failed: %v", err)
	}

	eventData := map[string]interface{}{}
	cfg := &spawn.Config{
		ProjectDir:    projectDir,
		WorkspaceName: workspaceName,
	}

	addAttemptIDToEventData(eventData, cfg)

	if got, _ := eventData["attempt_id"].(string); got != attemptID {
		t.Fatalf("attempt_id = %q, want %q", got, attemptID)
	}
}
