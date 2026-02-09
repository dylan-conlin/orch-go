package episodic

import (
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/activity"
	"github.com/dylan-conlin/orch-go/pkg/events"
)

func TestExtractEventSpawn(t *testing.T) {
	now := time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	event := events.Event{
		Type:      events.EventTypeSessionSpawned,
		SessionID: "ses_abc",
		Timestamp: now.Unix(),
		Data: map[string]interface{}{
			"spawn_mode": "headless",
			"beads_id":   "orch-go-123",
			"workspace":  "og-feat-123",
		},
	}

	memory, err := ExtractEvent(event, Context{
		Project:   "orch-go",
		Workspace: "og-feat-123",
		SessionID: "ses_abc",
		BeadsID:   "orch-go-123",
		Now:       now,
	})
	if err != nil {
		t.Fatalf("ExtractEvent failed: %v", err)
	}
	if memory == nil {
		t.Fatal("expected memory")
	}
	if memory.Boundary != BoundarySpawn {
		t.Fatalf("boundary = %s, want %s", memory.Boundary, BoundarySpawn)
	}
	if memory.Action.Name != events.EventTypeSessionSpawned {
		t.Fatalf("action.name = %s", memory.Action.Name)
	}
	if memory.Outcome.Status != OutcomeSuccess {
		t.Fatalf("outcome.status = %s", memory.Outcome.Status)
	}
}

func TestExtractEventVerificationBypassed(t *testing.T) {
	now := time.Now().UTC()
	event := events.Event{
		Type:      events.EventTypeVerificationBypassed,
		Timestamp: now.Unix(),
		Data: map[string]interface{}{
			"gate":     "test_evidence",
			"reason":   "tests run in CI",
			"beads_id": "orch-go-123",
		},
	}

	memory, err := ExtractEvent(event, Context{Project: "orch-go", Workspace: "og-feat-123", SessionID: "ses_123", BeadsID: "orch-go-123", Now: now})
	if err != nil {
		t.Fatalf("ExtractEvent failed: %v", err)
	}
	if memory == nil {
		t.Fatal("expected memory")
	}
	if memory.Boundary != BoundaryVerification {
		t.Fatalf("boundary = %s, want %s", memory.Boundary, BoundaryVerification)
	}
	if memory.Outcome.Status != OutcomeBypassed {
		t.Fatalf("outcome.status = %s, want %s", memory.Outcome.Status, OutcomeBypassed)
	}
}

func TestExtractActivityParts(t *testing.T) {
	now := time.Now().UTC()
	parts := []activity.MessagePartResponse{
		{
			ID:        "part_1",
			Type:      "message.part",
			Timestamp: now.Unix(),
			Properties: activity.MessagePartProperties{
				SessionID: "ses_123",
				MessageID: "msg_1",
				Part: activity.PartDetails{
					ID:        "part_1",
					Type:      "tool",
					SessionID: "ses_123",
					Tool:      "bash",
					State: &activity.ToolState{
						Status: "completed",
						Input: map[string]interface{}{
							"command": "go test ./...",
						},
						Output: "ok",
						Title:  "Run tests",
					},
				},
			},
		},
	}

	memories, err := ExtractActivityParts(parts, Context{
		Boundary:        BoundaryCompletion,
		Project:         "orch-go",
		Workspace:       "og-feat-123",
		SessionID:       "ses_123",
		BeadsID:         "orch-go-123",
		EvidencePointer: ".orch/workspace/og-feat-123/ACTIVITY.json",
		Now:             now,
	})
	if err != nil {
		t.Fatalf("ExtractActivityParts failed: %v", err)
	}
	if len(memories) != 1 {
		t.Fatalf("len(memories) = %d, want 1", len(memories))
	}
	if memories[0].Action.Name != "bash" {
		t.Fatalf("action.name = %s, want bash", memories[0].Action.Name)
	}
	if memories[0].Outcome.Status != OutcomeSuccess {
		t.Fatalf("outcome.status = %s, want %s", memories[0].Outcome.Status, OutcomeSuccess)
	}
}
