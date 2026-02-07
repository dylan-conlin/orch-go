package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// MessagePartResponse is the JSON structure for message parts in the activity feed.
// This mirrors the SSEEvent structure used by the frontend for real-time events,
// enabling seamless merging of historical API data with live SSE data.
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

// ActivityJSONFile is the structure of ACTIVITY.json exported on agent completion.
// This file serves as archival storage for session activity, loaded when
// the OpenCode session no longer exists (deleted/cleaned up).
type ActivityJSONFile struct {
	Version    int                   `json:"version"`
	SessionID  string                `json:"session_id"`
	ExportedAt string                `json:"exported_at"`
	Events     []MessagePartResponse `json:"events"`
}

// extractLastActivityFromMessages extracts the last meaningful activity from messages.
// It looks for the most recent assistant message and extracts a summary of what
// the agent is doing (tool use, text generation, etc.).
// Returns nil if no activity can be extracted.
func extractLastActivityFromMessages(messages []opencode.Message) *opencode.LastActivity {
	if len(messages) == 0 {
		return nil
	}

	// Find the last assistant message (most relevant for activity)
	var lastAssistantMsg *opencode.Message
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Info.Role == "assistant" {
			lastAssistantMsg = &messages[i]
			break
		}
	}

	if lastAssistantMsg == nil {
		return nil
	}

	// Extract activity from message parts
	// Priority: tool invocation > text > reasoning
	var activityText string
	for _, part := range lastAssistantMsg.Parts {
		switch part.Type {
		case "tool-invocation", "tool":
			// Tool use is the most informative activity
			activityText = "Using tool"
			if part.Text != "" {
				// Truncate tool text for display
				toolText := part.Text
				if len(toolText) > 40 {
					toolText = toolText[:40] + "..."
				}
				activityText = "Using tool: " + toolText
			}
		case "text":
			if part.Text != "" && activityText == "" {
				// Truncate long text
				text := part.Text
				if len(text) > 80 {
					// Find last space before 80 chars
					cutoff := 77
					for i := cutoff; i > 0; i-- {
						if text[i] == ' ' {
							cutoff = i
							break
						}
					}
					text = text[:cutoff] + "..."
				}
				activityText = text
			}
		case "reasoning":
			if activityText == "" {
				activityText = "Thinking..."
			}
		}
	}

	if activityText == "" {
		return nil
	}

	// Use message completion time if available, otherwise created time
	timestamp := lastAssistantMsg.Info.Time.Completed
	if timestamp == 0 {
		timestamp = lastAssistantMsg.Info.Time.Created
	}

	return &opencode.LastActivity{
		Text:      activityText,
		Timestamp: timestamp,
	}
}

// findWorkspaceBySessionID searches for a workspace directory with a matching .session_id file.
// This is used to find archived activity when the OpenCode session has been deleted.
// Returns the workspace path if found, or empty string if not found.
func findWorkspaceBySessionID(projectDir, sessionID string) string {
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		workspacePath := filepath.Join(workspaceDir, entry.Name())
		storedSessionID := spawn.ReadSessionID(workspacePath)
		if storedSessionID == sessionID {
			return workspacePath
		}
	}

	// Also check archived workspaces
	archivedDir := filepath.Join(workspaceDir, "archived")
	archivedEntries, err := os.ReadDir(archivedDir)
	if err != nil {
		return ""
	}

	for _, entry := range archivedEntries {
		if !entry.IsDir() {
			continue
		}
		workspacePath := filepath.Join(archivedDir, entry.Name())
		storedSessionID := spawn.ReadSessionID(workspacePath)
		if storedSessionID == sessionID {
			return workspacePath
		}
	}

	return ""
}

// loadActivityFromWorkspace loads activity events from ACTIVITY.json in a workspace.
// Returns the events if found and valid, or nil if not available.
func loadActivityFromWorkspace(workspacePath string) []MessagePartResponse {
	activityPath := filepath.Join(workspacePath, "ACTIVITY.json")
	data, err := os.ReadFile(activityPath)
	if err != nil {
		return nil
	}

	var activityFile ActivityJSONFile
	if err := json.Unmarshal(data, &activityFile); err != nil {
		return nil
	}

	return activityFile.Events
}

// handleSessionMessages proxies OpenCode's /session/:sessionID/message API.
// This endpoint enables the dashboard to fetch historical session messages
// for the activity feed, complementing real-time SSE updates.
//
// GET /api/session/:sessionID/messages
// Response: Array of MessagePartResponse in SSE-compatible format
func (s *Server) handleSessionMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract sessionID from URL path: /api/session/{sessionID}/messages
	path := r.URL.Path
	prefix := "/api/session/"
	suffix := "/messages"

	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		http.Error(w, "Invalid path format. Expected: /api/session/{sessionID}/messages", http.StatusBadRequest)
		return
	}

	sessionID := path[len(prefix) : len(path)-len(suffix)]
	if sessionID == "" {
		http.Error(w, "Session ID is required", http.StatusBadRequest)
		return
	}

	client := opencode.NewClient(s.ServerURL)
	messages, err := client.GetMessages(sessionID)
	if err != nil {
		// OpenCode API failed (session may be deleted/cleaned up).
		// Fall back to ACTIVITY.json if available in the workspace.
		projectDir, _ := s.currentProjectDir()
		workspacePath := findWorkspaceBySessionID(projectDir, sessionID)
		if workspacePath != "" {
			if events := loadActivityFromWorkspace(workspacePath); events != nil {
				// Successfully loaded from ACTIVITY.json
				w.Header().Set("Content-Type", "application/json")
				if encErr := json.NewEncoder(w).Encode(events); encErr != nil {
					http.Error(w, fmt.Sprintf("Failed to encode events: %v", encErr), http.StatusInternalServerError)
				}
				return
			}
		}
		// No fallback available, return original error
		http.Error(w, fmt.Sprintf("Failed to fetch messages: %v", err), http.StatusInternalServerError)
		return
	}

	// Transform OpenCode messages to SSE-compatible format for the activity feed.
	// This enables seamless merging with real-time SSE events in the frontend.
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
						Tool:      part.Tool, // Add tool name for tool invocations
						State:     state,     // Add tool state (input/output)
					},
				},
				Timestamp: msg.Info.Time.Created,
			}
			parts = append(parts, response)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(parts); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode messages: %v", err), http.StatusInternalServerError)
		return
	}
}
