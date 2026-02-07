// Package activity handles activity feed persistence for agent workspaces.
// It exports session activity to ACTIVITY.json files for archival purposes.
package activity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

// ActivityFile represents the structure of ACTIVITY.json.
type ActivityFile struct {
	Version    int                   `json:"version"`
	SessionID  string                `json:"session_id"`
	ExportedAt string                `json:"exported_at"`
	Events     []MessagePartResponse `json:"events"`
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
	return ExportToWorkspaceWithClient(nil, sessionID, workspacePath, serverURL)
}

// ExportToWorkspaceWithClient exports session activity to a workspace using a provided client.
// If client is nil, a new client is created using serverURL.
func ExportToWorkspaceWithClient(client opencode.ClientInterface, sessionID, workspacePath, serverURL string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("session ID is required")
	}
	if workspacePath == "" {
		return "", fmt.Errorf("workspace path is required")
	}

	// Fetch messages from OpenCode API
	if client == nil {
		client = opencode.NewClient(serverURL)
	}
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

// PhaseCompleteAttempt represents evidence of an agent attempting to report Phase: Complete.
type PhaseCompleteAttempt struct {
	Found           bool   // Whether a Phase: Complete attempt was found
	CommandOutput   string // The output of the bd comment command
	CommandInput    string // The command that was executed
	ReportedSuccess bool   // Whether bd reported success ("Comment added")
	Timestamp       int64  // When the attempt occurred
}

// DetectPhaseCompleteAttempt scans ACTIVITY.json for evidence that the agent
// attempted to report "Phase: Complete" via bd comment.
//
// This is used as a fallback when Phase: Complete is not found in beads comments.
// It handles the case where bd comment reports success but the comment fails to
// persist (a known beads bug).
//
// Returns the attempt details if found, or an empty struct with Found=false if not.
func DetectPhaseCompleteAttempt(workspacePath string) PhaseCompleteAttempt {
	events, err := LoadFromWorkspace(workspacePath)
	if err != nil || events == nil {
		return PhaseCompleteAttempt{}
	}

	return DetectPhaseCompleteAttemptFromEvents(events)
}

// DetectPhaseCompleteAttemptFromEvents scans activity events for Phase: Complete attempts.
// This is separated from DetectPhaseCompleteAttempt for easier testing.
func DetectPhaseCompleteAttemptFromEvents(events []MessagePartResponse) PhaseCompleteAttempt {
	// Scan for bash tool calls that contain "Phase: Complete" in the command
	for _, event := range events {
		part := event.Properties.Part
		if part.Type != "tool" || part.Tool != "bash" {
			continue
		}
		if part.State == nil {
			continue
		}

		// Check if the command contains "bd comment" and "Phase: Complete"
		command, ok := part.State.Input["command"].(string)
		if !ok {
			continue
		}

		// Look for bd comment commands with Phase: Complete
		if !containsPhaseComplete(command) {
			continue
		}

		// Check if bd reported success
		output := part.State.Output
		reportedSuccess := strings.Contains(output, "Comment added")

		return PhaseCompleteAttempt{
			Found:           true,
			CommandInput:    command,
			CommandOutput:   output,
			ReportedSuccess: reportedSuccess,
			Timestamp:       event.Timestamp,
		}
	}

	return PhaseCompleteAttempt{}
}

// containsPhaseComplete checks if a command string appears to be reporting Phase: Complete.
// Matches patterns like:
//   - bd comment <id> "Phase: Complete - ..."
//   - bd comments add <id> "Phase: Complete - ..."
//
// Returns true only when "Phase: Complete" appears at the start of the quoted comment,
// not when it's mentioned within a larger description or in a grep/list command.
func containsPhaseComplete(command string) bool {
	lowerCmd := strings.ToLower(command)

	// Must be a bd comment or bd comments add command
	// Exclude: bd comments <id> (list), bd comments <id> | grep (piped list)
	isBdCommentAdd := false
	if strings.Contains(lowerCmd, "bd comment ") && !strings.Contains(lowerCmd, "bd comments") {
		// Simple "bd comment <id> <text>" - this is an add operation
		isBdCommentAdd = true
	} else if strings.Contains(lowerCmd, "bd comments add") {
		// Explicit "bd comments add <id> <text>"
		isBdCommentAdd = true
	}

	if !isBdCommentAdd {
		return false
	}

	// Look for Phase: Complete at the start of the quoted text
	// Matches: "Phase: Complete" or 'Phase: Complete'
	return strings.Contains(lowerCmd, `"phase: complete`) || strings.Contains(lowerCmd, `'phase: complete`)
}
