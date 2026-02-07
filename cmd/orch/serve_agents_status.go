package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/port"
)

// determineDeathReason analyzes why an agent died and returns a specific reason code.
// Reasons: "server_restart", "context_exhausted", "auth_failed", "error", "timeout", "unknown"
func determineDeathReason(sessionID string, sessionCreatedAt time.Time, client opencode.ClientInterface) string {
	// Check if session existed before server restart
	if !serverStartTime.IsZero() && sessionCreatedAt.Before(serverStartTime) {
		return "server_restart"
	}

	// Check last message for specific error patterns
	lastMsg, err := client.GetLastMessage(sessionID)
	if err != nil || lastMsg == nil {
		// No messages available, likely timeout
		return "timeout"
	}

	// Check for context exhaustion (token limit errors)
	// OpenCode reports token limits in error messages
	for _, part := range lastMsg.Parts {
		if part.Type == "text" && part.Text != "" {
			text := strings.ToLower(part.Text)
			if strings.Contains(text, "token limit") ||
				strings.Contains(text, "context length") ||
				strings.Contains(text, "maximum context") ||
				strings.Contains(text, "too many tokens") {
				return "context_exhausted"
			}
		}
	}

	// Check for auth failures (401/403 errors)
	// Look at message finish reason or error info
	if lastMsg.Info.Finish != "" {
		finish := strings.ToLower(lastMsg.Info.Finish)
		if strings.Contains(finish, "unauthorized") ||
			strings.Contains(finish, "forbidden") ||
			strings.Contains(finish, "authentication") ||
			strings.Contains(finish, "auth") {
			return "auth_failed"
		}
	}

	// Check for general errors in the last message
	// If the last message had an error, it's an error death
	if lastMsg.Info.Finish == "error" || lastMsg.Info.Finish == "tool_error" {
		return "error"
	}

	// Check message parts for error indicators
	for _, part := range lastMsg.Parts {
		if part.Type == "error" || (part.Type == "text" && strings.HasPrefix(strings.ToLower(part.Text), "error:")) {
			return "error"
		}
	}

	// If we got here, it's likely a timeout (no activity for threshold period)
	// This is the most common death reason for agents that just stop responding
	return "timeout"
}

// getProjectAPIPort returns the allocated API port for the current project.
// Returns 0 if no allocation exists or on error.
func getProjectAPIPort() int {
	projectDir, err := currentProjectDir()
	if err != nil {
		return 0
	}
	projectName := filepath.Base(projectDir)

	registry, err := port.New("")
	if err != nil {
		return 0
	}

	alloc := registry.Find(projectName, "api")
	if alloc == nil {
		return 0
	}

	return alloc.Port
}

// checkWorkspaceSynthesis checks if a workspace has a non-empty SYNTHESIS.md file.
// This is used to detect completion for untracked agents (--no-track) where
// there's no beads issue to check Phase: Complete.
func checkWorkspaceSynthesis(workspacePath string) bool {
	if workspacePath == "" {
		return false
	}
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	info, err := os.Stat(synthesisPath)
	if err != nil {
		return false
	}
	// SYNTHESIS.md must exist and be non-empty
	return info.Size() > 0
}

// determineAgentStatus implements the Priority Cascade model for agent status.
// This is the single source of truth for determining agent status.
//
// Priority order (highest to lowest):
//  1. Beads issue closed -> "completed" (orchestrator verified completion)
//  2. Phase: Complete reported AND session dead -> "awaiting-cleanup" (agent done, needs orch complete)
//  3. Phase: Complete reported -> "completed" (agent declared done, still has active session)
//  4. SYNTHESIS.md exists AND session dead -> "awaiting-cleanup" (artifact proves completion, needs cleanup)
//  5. SYNTHESIS.md exists -> "completed" (artifact proves completion)
//  6. Session activity -> sessionStatus ("active", "idle", or "dead")
//
// The "awaiting-cleanup" status distinguishes completed-but-orphaned agents from crashed agents.
// This helps orchestrators prioritize: awaiting-cleanup needs orch complete, dead needs investigation.
// See .kb/investigations/2026-01-04-design-dashboard-agent-status-model.md for design rationale.
// See .kb/investigations/2026-01-08-inv-handle-multiple-agents-same-beads.md for awaiting-cleanup addition.
func determineAgentStatus(issueClosed bool, phaseComplete bool, workspacePath string, sessionStatus string) string {
	// Priority 1: Beads issue closed -> completed (orchestrator verified completion)
	if issueClosed {
		return "completed"
	}

	hasSynthesis := checkWorkspaceSynthesis(workspacePath)
	isDead := sessionStatus == "dead"

	// Priority 2: Phase: Complete reported AND session dead -> awaiting-cleanup
	// Agent finished work and reported completion, but orchestrator hasn't run orch complete.
	// This is NOT an error state - the agent did its job, just needs cleanup.
	if phaseComplete && isDead {
		return "awaiting-cleanup"
	}

	// Priority 3: Phase: Complete reported (session still active/idle) -> completed
	if phaseComplete {
		return "completed"
	}

	// Priority 4: SYNTHESIS.md exists AND session dead -> awaiting-cleanup
	// Agent wrote synthesis artifact (proof of completion) but session is dead.
	// Similar to Phase: Complete case - needs cleanup, not investigation.
	if hasSynthesis && isDead {
		return "awaiting-cleanup"
	}

	// Priority 5: SYNTHESIS.md exists (session still active/idle) -> completed
	if hasSynthesis {
		return "completed"
	}

	// Priority 6: Session activity (fallback)
	// "dead" agents without completion signals truly need attention (crashed/stuck)
	return sessionStatus
}

// getWorkspaceLastActivity returns the most recent file modification time in a workspace.
// This is used for activity detection in tmux agents (Claude CLI escape hatch) where
// we don't have session timestamps from OpenCode API.
// Returns zero time if workspace doesn't exist or has no activity files.
func getWorkspaceLastActivity(workspacePath string) time.Time {
	if workspacePath == "" {
		return time.Time{}
	}

	// Files that indicate agent activity (ordered by relevance)
	// Investigation files and SYNTHESIS.md are the most relevant indicators
	activityFiles := []string{
		"SYNTHESIS.md",
		"SPAWN_CONTEXT.md",
		".session_id",
		".spawn_time",
	}

	var lastMod time.Time

	// Check known activity files first
	for _, filename := range activityFiles {
		filePath := filepath.Join(workspacePath, filename)
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}
		if info.ModTime().After(lastMod) {
			lastMod = info.ModTime()
		}
	}

	// Also check for any .md files in the workspace (investigation files)
	entries, err := os.ReadDir(workspacePath)
	if err != nil {
		return lastMod
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// Check markdown files (investigation files, notes, etc.)
		if strings.HasSuffix(entry.Name(), ".md") {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.ModTime().After(lastMod) {
				lastMod = info.ModTime()
			}
		}
	}

	return lastMod
}
