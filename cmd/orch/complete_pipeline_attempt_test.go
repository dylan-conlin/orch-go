package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPostCompleteIncludesAttemptID(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	workspace := filepath.Join(t.TempDir(), "ws")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	attemptID := "123e4567-e89b-42d3-a456-426614174002"
	if err := os.WriteFile(filepath.Join(workspace, ".attempt_id"), []byte(attemptID+"\n"), 0644); err != nil {
		t.Fatalf("write .attempt_id failed: %v", err)
	}

	origNoChangelogCheck := completeNoChangelogCheck
	origForce := completeForce
	completeNoChangelogCheck = true
	completeForce = false
	defer func() {
		completeNoChangelogCheck = origNoChangelogCheck
		completeForce = origForce
	}()

	target := &CompletionTarget{
		BeadsID:         "orch-go-abc123",
		WorkspacePath:   workspace,
		AgentName:       "og-feat-test-09feb-ab12",
		BeadsProjectDir: t.TempDir(),
		IsUntracked:     true,
	}
	vOutcome := &VerificationOutcome{Passed: true, SkillName: "feature-impl"}
	telemetry := CompletionTelemetry{Outcome: "success"}

	postComplete(target, vOutcome, "done", telemetry)

	content, err := os.ReadFile(filepath.Join(home, ".orch", "events.jsonl"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) == 0 {
		t.Fatal("expected events.jsonl to contain events")
	}

	var event map[string]interface{}
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &event); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if got, _ := event["type"].(string); got != "agent.completed" {
		t.Fatalf("event type = %q, want agent.completed", got)
	}

	data, ok := event["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("event data missing or malformed: %v", event)
	}
	if got, _ := data["attempt_id"].(string); got != attemptID {
		t.Fatalf("attempt_id = %q, want %q", got, attemptID)
	}
}
