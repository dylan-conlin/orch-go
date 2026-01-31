package activity

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

func TestTransformMessages(t *testing.T) {
	sessionID := "ses_test123"
	messages := []opencode.Message{
		{
			Info: opencode.MessageInfo{
				ID:        "msg_1",
				SessionID: sessionID,
				Role:      "assistant",
				Time:      opencode.MessageTime{Created: 1737146700000},
			},
			Parts: []opencode.MessagePart{
				{
					ID:        "part_1",
					SessionID: sessionID,
					MessageID: "msg_1",
					Type:      "text",
					Text:      "Hello, I'll help you with that.",
				},
				{
					ID:        "part_2",
					SessionID: sessionID,
					MessageID: "msg_1",
					Type:      "tool-invocation",
					Tool:      "Bash",
					State: &opencode.ToolState{
						Status: "completed",
						Title:  "List files",
						Input:  map[string]interface{}{"command": "ls -la"},
						Output: "file1.txt\nfile2.txt",
					},
				},
				{
					ID:        "part_3",
					SessionID: sessionID,
					MessageID: "msg_1",
					Type:      "reasoning",
					Text:      "Thinking about the next step...",
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
	messages := []opencode.Message{
		{
			Info: opencode.MessageInfo{
				ID:        "msg_1",
				SessionID: sessionID,
				Role:      "assistant",
				Time:      opencode.MessageTime{Created: 1737146700000},
			},
			Parts: []opencode.MessagePart{
				{
					ID:        "part_1",
					SessionID: sessionID,
					MessageID: "msg_1",
					Type:      "text",
					Text:      "Valid text",
				},
				{
					ID:        "part_2",
					SessionID: sessionID,
					MessageID: "msg_1",
					Type:      "unknown-type", // Should be filtered out
					Text:      "Should not appear",
				},
				{
					ID:        "part_3",
					SessionID: sessionID,
					MessageID: "msg_1",
					Type:      "step-start", // Valid type
					Text:      "Starting step",
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

func TestDetectPhaseCompleteAttemptFromEvents(t *testing.T) {
	tests := []struct {
		name            string
		events          []MessagePartResponse
		wantFound       bool
		wantSuccess     bool
		wantInputSubstr string
	}{
		{
			name:      "no events",
			events:    nil,
			wantFound: false,
		},
		{
			name: "no Phase: Complete attempt",
			events: []MessagePartResponse{
				{
					Properties: MessagePartProperties{
						Part: PartDetails{
							Type: "tool",
							Tool: "bash",
							State: &ToolState{
								Input:  map[string]interface{}{"command": "ls -la"},
								Output: "files...",
							},
						},
					},
				},
			},
			wantFound: false,
		},
		{
			name: "Phase: Complete with success",
			events: []MessagePartResponse{
				{
					Timestamp: 1737146700000,
					Properties: MessagePartProperties{
						Part: PartDetails{
							Type: "tool",
							Tool: "bash",
							State: &ToolState{
								Input:  map[string]interface{}{"command": `bd comment orch-go-123 "Phase: Complete - All done"`},
								Output: "Command \"comment\" is deprecated, use 'bd comments add' instead\nComment added to orch-go-123\n",
							},
						},
					},
				},
			},
			wantFound:       true,
			wantSuccess:     true,
			wantInputSubstr: "Phase: Complete",
		},
		{
			name: "Phase: Complete with bd comments add",
			events: []MessagePartResponse{
				{
					Timestamp: 1737146700000,
					Properties: MessagePartProperties{
						Part: PartDetails{
							Type: "tool",
							Tool: "bash",
							State: &ToolState{
								Input:  map[string]interface{}{"command": `bd comments add orch-go-123 "Phase: Complete - All done"`},
								Output: "Comment added to orch-go-123\n",
							},
						},
					},
				},
			},
			wantFound:       true,
			wantSuccess:     true,
			wantInputSubstr: "Phase: Complete",
		},
		{
			name: "Phase: Complete with failure (no Comment added)",
			events: []MessagePartResponse{
				{
					Timestamp: 1737146700000,
					Properties: MessagePartProperties{
						Part: PartDetails{
							Type: "tool",
							Tool: "bash",
							State: &ToolState{
								Input:  map[string]interface{}{"command": `bd comment orch-go-123 "Phase: Complete - All done"`},
								Output: "error: database locked\n",
							},
						},
					},
				},
			},
			wantFound:       true,
			wantSuccess:     false,
			wantInputSubstr: "Phase: Complete",
		},
		{
			name: "does not match Phase: Complete mentioned in description",
			events: []MessagePartResponse{
				{
					Properties: MessagePartProperties{
						Part: PartDetails{
							Type: "tool",
							Tool: "bash",
							State: &ToolState{
								// This command mentions Phase: Complete but it's in a description, not reporting it
								Input:  map[string]interface{}{"command": `bd comment orch-go-123 "Scope: 1. Detect sessions without Phase: Complete 2. Mark as failed"`},
								Output: "Comment added to orch-go-123\n",
							},
						},
					},
				},
			},
			wantFound: false, // Should NOT match - Phase: Complete is mentioned but not being reported
		},
		{
			name: "does not match bd comments list command",
			events: []MessagePartResponse{
				{
					Properties: MessagePartProperties{
						Part: PartDetails{
							Type: "tool",
							Tool: "bash",
							State: &ToolState{
								// This is listing comments, not adding one
								Input:  map[string]interface{}{"command": `bd comments orch-go-123 | grep "Phase: Complete"`},
								Output: "",
							},
						},
					},
				},
			},
			wantFound: false,
		},
		{
			name: "case insensitive Phase: complete",
			events: []MessagePartResponse{
				{
					Properties: MessagePartProperties{
						Part: PartDetails{
							Type: "tool",
							Tool: "bash",
							State: &ToolState{
								Input:  map[string]interface{}{"command": `bd comment orch-go-123 "phase: complete - done"`},
								Output: "Comment added to orch-go-123\n",
							},
						},
					},
				},
			},
			wantFound:   true,
			wantSuccess: true,
		},
		{
			name: "not a bash tool",
			events: []MessagePartResponse{
				{
					Properties: MessagePartProperties{
						Part: PartDetails{
							Type: "tool",
							Tool: "read", // not bash
							State: &ToolState{
								Input:  map[string]interface{}{"command": `Phase: Complete`},
								Output: "some output",
							},
						},
					},
				},
			},
			wantFound: false,
		},
		{
			name: "nil state",
			events: []MessagePartResponse{
				{
					Properties: MessagePartProperties{
						Part: PartDetails{
							Type:  "tool",
							Tool:  "bash",
							State: nil,
						},
					},
				},
			},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectPhaseCompleteAttemptFromEvents(tt.events)

			if result.Found != tt.wantFound {
				t.Errorf("Found = %v, want %v", result.Found, tt.wantFound)
			}

			if tt.wantFound {
				if result.ReportedSuccess != tt.wantSuccess {
					t.Errorf("ReportedSuccess = %v, want %v", result.ReportedSuccess, tt.wantSuccess)
				}
				if tt.wantInputSubstr != "" && !containsPhaseComplete(result.CommandInput) {
					t.Errorf("CommandInput should contain Phase: Complete, got: %s", result.CommandInput)
				}
			}
		})
	}
}
