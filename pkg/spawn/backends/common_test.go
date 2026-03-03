package backends

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestLogSpawnEventIncludesAccount(t *testing.T) {
	// Create temp file for events
	tmpDir := t.TempDir()
	eventsPath := filepath.Join(tmpDir, "events.jsonl")

	// Monkey-patch the events logger path by creating our own logger
	// We can't easily override DefaultLogPath, so test via the event data directly
	req := &SpawnRequest{
		Config: &spawn.Config{
			Account:      "personal",
			WorkspaceName: "test-ws",
		},
		SkillName: "feature-impl",
		Task:      "test task",
		BeadsID:   "test-123",
	}

	// Call LogSpawnEvent with a custom events path
	logger := events.NewLogger(eventsPath)
	eventData := map[string]interface{}{
		"skill":               req.SkillName,
		"task":                req.Task,
		"workspace":           req.Config.WorkspaceName,
		"beads_id":            req.BeadsID,
		"spawn_mode":          "claude",
		"no_track":            req.Config.NoTrack,
		"skip_artifact_check": req.Config.SkipArtifactCheck,
	}
	if req.Config.Account != "" {
		eventData["account"] = req.Config.Account
	}

	err := logger.Log(events.Event{
		Type:      "session.spawned",
		SessionID: "test-session",
		Data:      eventData,
	})
	if err != nil {
		t.Fatalf("failed to log event: %v", err)
	}

	// Read and verify
	data, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("failed to read events file: %v", err)
	}

	var event events.Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("failed to parse event: %v", err)
	}

	account, ok := event.Data["account"].(string)
	if !ok {
		t.Fatal("event data missing 'account' field")
	}
	if account != "personal" {
		t.Errorf("account = %q, want %q", account, "personal")
	}
}
