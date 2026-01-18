// Package activity handles activity feed persistence for agent workspaces.
// It exports session activity to ACTIVITY.json files for archival purposes.
package activity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

// ActivityFile represents the structure of ACTIVITY.json.
type ActivityFile struct {
	Version    int                    `json:"version"`
	SessionID  string                 `json:"session_id"`
	ExportedAt string                 `json:"exported_at"`
	Events     []MessagePartResponse  `json:"events"`
}

// MessagePartResponse represents a single activity event in SSE-compatible format.
// This format matches what the dashboard activity feed expects.
type MessagePartResponse struct {
	ID         string                `json:"id"`
	Type       string                `json:"type"` // Always "message.part" to match SSE event type
	Properties MessagePartProperties `json:"properties"`
	Timestamp  int64                 `json:"timestamp,omitempty"`
}

// MessagePartProperties contains the part data in SSE-compatible format.
type MessagePartProperties struct {
	SessionID string      `json:"sessionID"`
	MessageID string      `json:"messageID"`
	Part      PartDetails `json:"part"`
}

// PartDetails contains the actual part content.
type PartDetails struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"` // "text", "tool", "reasoning", "step-start", "step-finish"
	Text      string     `json:"text,omitempty"`
	SessionID string     `json:"sessionID"`
	Tool      string     `json:"tool,omitempty"`
	State     *ToolState `json:"state,omitempty"`
}

// ToolState contains tool invocation state for tool parts.
type ToolState struct {
	Title  string                 `json:"title,omitempty"`
	Status string                 `json:"status,omitempty"`
	Input  map[string]interface{} `json:"input,omitempty"`
	Output string                 `json:"output,omitempty"`
}

// ExportToWorkspace exports session activity to ACTIVITY.json in the workspace.
// It fetches messages from the OpenCode API and transforms them to SSE-compatible format.
// Returns the path to the exported file, or empty string if no messages found.
func ExportToWorkspace(sessionID, workspacePath, serverURL string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("session ID is required")
	}
	if workspacePath == "" {
		return "", fmt.Errorf("workspace path is required")
	}

	// Fetch messages from OpenCode API
	client := opencode.NewClient(serverURL)
	messages, err := client.GetMessages(sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch messages: %w", err)
	}

	// Skip if no messages
	if len(messages) == 0 {
		return "", nil
	}

	// Transform to SSE-compatible format
	events := TransformMessages(sessionID, messages)

	// Create activity file structure
	activityFile := ActivityFile{
		Version:    1,
		SessionID:  sessionID,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Events:     events,
	}

	// Write to ACTIVITY.json
	outputPath := filepath.Join(workspacePath, "ACTIVITY.json")
	data, err := json.MarshalIndent(activityFile, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal activity: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write activity file: %w", err)
	}

	return outputPath, nil
}

// TransformMessages converts OpenCode messages to SSE-compatible activity events.
// This matches the transformation done in serve_agents.go handleSessionMessages.
func TransformMessages(sessionID string, messages []opencode.Message) []MessagePartResponse {
	var parts []MessagePartResponse

	for _, msg := range messages {
		for _, part := range msg.Parts {
			// Map OpenCode part types to activity feed types
			partType := part.Type
			switch part.Type {
			case "tool-invocation":
				partType = "tool"
			}

			// Only include types that the activity feed displays
			if partType != "text" && partType != "tool" && partType != "reasoning" &&
				partType != "step-start" && partType != "step-finish" {
				continue
			}

			// Transform tool state if present
			var state *ToolState
			if part.State != nil {
				state = &ToolState{
					Title:  part.State.Title,
					Status: part.State.Status,
					Input:  part.State.Input,
					Output: part.State.Output,
				}
			}

			response := MessagePartResponse{
				ID:   part.ID,
				Type: "message.part", // Match SSE event type
				Properties: MessagePartProperties{
					SessionID: sessionID,
					MessageID: msg.Info.ID,
					Part: PartDetails{
						ID:        part.ID,
						Type:      partType,
						Text:      part.Text,
						SessionID: sessionID,
						Tool:      part.Tool,
						State:     state,
					},
				},
				Timestamp: msg.Info.Time.Created,
			}
			parts = append(parts, response)
		}
	}

	return parts
}

// LoadFromWorkspace loads activity events from ACTIVITY.json in a workspace.
// Returns the events and nil if the file exists and is valid.
// Returns nil, nil if the file doesn't exist (not an error).
// Returns nil, error if the file exists but is invalid.
func LoadFromWorkspace(workspacePath string) ([]MessagePartResponse, error) {
	activityPath := filepath.Join(workspacePath, "ACTIVITY.json")

	data, err := os.ReadFile(activityPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // File doesn't exist - not an error
		}
		return nil, fmt.Errorf("failed to read activity file: %w", err)
	}

	var activityFile ActivityFile
	if err := json.Unmarshal(data, &activityFile); err != nil {
		return nil, fmt.Errorf("failed to parse activity file: %w", err)
	}

	return activityFile.Events, nil
}
