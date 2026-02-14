// Package verify provides verification helpers for agent completion.
package verify

import (
	"fmt"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// BackendResult represents the result of verifying deliverables from a backend (opencode/tmux).
type BackendResult struct {
	Passed   bool
	Errors   []string
	Warnings []string
}

// VerifyBackendDeliverables verifies that the agent has reported completion in its backend.
// For opencode mode, it checks the HTTP API transcript.
// For claude/tmux mode, it checks the tmux window capture.
func VerifyBackendDeliverables(workspacePath, beadsID, serverURL, backend string) *BackendResult {
	result := &BackendResult{Passed: true}

	if backend == "" {
		// Try to read from workspace
		backend = ReadSpawnModeFromWorkspace(workspacePath)
	}

	switch strings.ToLower(backend) {
	case "opencode", "headless":
		verifyOpencodeDeliverables(workspacePath, serverURL, result)
	case "claude", "tmux":
		verifyTmuxDeliverables(beadsID, result)
	}

	return result
}

// verifyOpencodeDeliverables checks the opencode transcript for completion signals.
func verifyOpencodeDeliverables(workspacePath, serverURL string, result *BackendResult) {
	if serverURL == "" {
		return
	}

	// Read session ID from workspace (.session_id stays separate - infrastructure handle)
	sessionID := spawn.ReadSessionID(workspacePath)
	if sessionID == "" {
		result.Warnings = append(result.Warnings, "could not read .session_id")
		return
	}

	client := opencode.NewClient(serverURL)
	messages, err := client.GetMessages(sessionID)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to fetch opencode messages: %v", err))
		return
	}

	// Check for "Phase: Complete" in any assistant message
	found := false
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Info.Role != "assistant" {
			continue
		}
		for _, part := range msg.Parts {
			if part.Type == "text" && strings.Contains(part.Text, "Phase: Complete") {
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		result.Warnings = append(result.Warnings, "Phase: Complete not found in opencode transcript")
		// We don't fail yet, because beads comment is the authoritative source.
		// But Dylan wants to "verify deliverables".
	}
}

// verifyTmuxDeliverables checks the tmux window capture for completion signals.
func verifyTmuxDeliverables(beadsID string, result *BackendResult) {
	if beadsID == "" {
		return
	}

	window, _, err := tmux.FindWindowByBeadsIDAllSessions(beadsID)
	if err != nil || window == nil {
		result.Warnings = append(result.Warnings, "could not find tmux window for agent")
		return
	}

	// Capture pane output
	content, err := tmux.GetPaneContent(window.Target)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to capture tmux pane: %v", err))
		return
	}

	if !strings.Contains(content, "Phase: Complete") {
		result.Warnings = append(result.Warnings, "Phase: Complete not found in tmux window output")
	}
}

// ReadSpawnModeFromWorkspace reads the spawn mode from the workspace.
// Reads AGENT_MANIFEST.json first, falls back to .spawn_mode dotfile.
func ReadSpawnModeFromWorkspace(workspacePath string) string {
	manifest := spawn.ReadAgentManifestWithFallback(workspacePath)
	return manifest.SpawnMode
}
