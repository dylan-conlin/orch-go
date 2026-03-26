package activity

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/execution"
)

func TestTransformMessages(t *testing.T) {
	sessionID := "ses_test123"
	messages := []execution.Message{
		{
			ID:        "msg_1",
			SessionID: sessionID,
			Role:      "assistant",
			Created:   time.Unix(1737146700, 0),
			Parts: []execution.MessagePart{
				{
					CallID: "part_1",
					Type:   "text",
					Text:   "Hello, I'll help you with that.",
				},
				{
					CallID: "part_2",
					Type:   "tool-invocation",
					Tool:   "Bash",
					State: &execution.ToolState{
						Status: "completed",
						Title:  "List files",
						Input:  map[string]interface{}{"command": "ls -la"},
						Output: "file1.txt\nfile2.txt",
					},
				},
				{
					CallID: "part_3",
					Type:   "reasoning",
					Text:   "Thinking about the next step...",
				},
			},
		},
	}

	events := TransformMessages(sessionID, messages)

	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}

	// Check first event (text)
	if events[0].Type != "message.part" {
		t.Errorf("expected type 'message.part', got '%s'", events[0].Type)
	}
	if events[0].Properties.Part.Type != "text" {
		t.Errorf("expected part type 'text', got '%s'", events[0].Properties.Part.Type)
	}
	if events[0].Properties.Part.Text != "Hello, I'll help you with that." {
		t.Errorf("unexpected text: %s", events[0].Properties.Part.Text)
	}

	// Check second event (tool - converted from tool-invocation)
	if events[1].Properties.Part.Type != "tool" {
		t.Errorf("expected part type 'tool', got '%s'", events[1].Properties.Part.Type)
	}
	if events[1].Properties.Part.Tool != "Bash" {
		t.Errorf("expected tool 'Bash', got '%s'", events[1].Properties.Part.Tool)
	}
	if events[1].Properties.Part.State == nil {
		t.Error("expected state to be present for tool")
	} else {
		if events[1].Properties.Part.State.Status != "completed" {
			t.Errorf("expected status 'completed', got '%s'", events[1].Properties.Part.State.Status)
		}
	}

	// Check third event (reasoning)
	if events[2].Properties.Part.Type != "reasoning" {
		t.Errorf("expected part type 'reasoning', got '%s'", events[2].Properties.Part.Type)
	}
}

func TestTransformMessages_FiltersInvalidTypes(t *testing.T) {
	sessionID := "ses_test123"
	messages := []execution.Message{
		{
			ID:        "msg_1",
			SessionID: sessionID,
			Role:      "assistant",
			Created:   time.Unix(1737146700, 0),
			Parts: []execution.MessagePart{
				{
					CallID: "part_1",
					Type:   "text",
					Text:   "Valid text",
				},
				{
					CallID: "part_2",
					Type:   "unknown-type", // Should be filtered out
					Text:   "Should not appear",
				},
				{
					CallID: "part_3",
					Type:   "step-start", // Valid type
					Text:   "Starting step",
				},
			},
		},
	}

	events := TransformMessages(sessionID, messages)

	// Only text and step-start should be included (unknown-type filtered)
	if len(events) != 2 {
		t.Errorf("expected 2 events (filtered unknown-type), got %d", len(events))
	}
}

func TestLoadFromWorkspace_FileNotExists(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "activity-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	events, err := LoadFromWorkspace(tempDir)
	if err != nil {
		t.Errorf("expected no error for non-existent file, got: %v", err)
	}
	if events != nil {
		t.Errorf("expected nil events for non-existent file, got: %v", events)
	}
}

func TestLoadFromWorkspace_ValidFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "activity-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write a valid ACTIVITY.json
	activityJSON := `{
		"version": 1,
		"session_id": "ses_test",
		"exported_at": "2026-01-17T15:45:00Z",
		"events": [
			{
				"id": "part_1",
				"type": "message.part",
				"properties": {
					"sessionID": "ses_test",
					"messageID": "msg_1",
					"part": {
						"id": "part_1",
						"type": "text",
						"text": "Hello world",
						"sessionID": "ses_test"
					}
				},
				"timestamp": 1737146700000
			}
		]
	}`
	if err := os.WriteFile(filepath.Join(tempDir, "ACTIVITY.json"), []byte(activityJSON), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	events, err := LoadFromWorkspace(tempDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
	if events[0].Properties.Part.Text != "Hello world" {
		t.Errorf("unexpected text: %s", events[0].Properties.Part.Text)
	}
}

func TestLoadFromWorkspace_InvalidJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "activity-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write invalid JSON
	if err := os.WriteFile(filepath.Join(tempDir, "ACTIVITY.json"), []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	events, err := LoadFromWorkspace(tempDir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if events != nil {
		t.Errorf("expected nil events for invalid JSON, got: %v", events)
	}
}
