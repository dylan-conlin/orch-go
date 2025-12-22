// Package state provides agent state reconciliation across multiple sources.
package state

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// LivenessResult contains the liveness status from multiple sources.
type LivenessResult struct {
	// TmuxLive indicates if a tmux window exists for this agent.
	TmuxLive bool

	// OpencodeLive indicates if an OpenCode session is active.
	OpencodeLive bool

	// BeadsOpen indicates if the beads issue is open (not closed).
	BeadsOpen bool

	// WorkspaceExists indicates if the workspace directory exists.
	WorkspaceExists bool

	// SessionID is the OpenCode session ID if found.
	SessionID string

	// WindowID is the tmux window ID if found (e.g., "@1234").
	WindowID string

	// WorkspacePath is the path to the workspace directory if found.
	WorkspacePath string

	// AgentName is the workspace name (directory name) if found.
	AgentName string
}

// IsAlive returns true if the agent is live in any source (tmux or OpenCode).
func (r *LivenessResult) IsAlive() bool {
	return r.TmuxLive || r.OpencodeLive
}

// IsPhantom returns true if the beads issue is open but no live sources exist.
// This indicates a "phantom" agent that appears in tracking but isn't actually running.
func (r *LivenessResult) IsPhantom() bool {
	return r.BeadsOpen && !r.TmuxLive && !r.OpencodeLive
}

// IsLive cross-references all 4 state sources to determine if an agent is actually running.
// Returns (tmuxLive, opencodeLive bool) indicating which sources show the agent as active.
//
// The 4 state sources checked:
// 1. tmux windows - via FindWindowByBeadsID or workspace session lookup
// 2. OpenCode sessions - via API session lookup
// 3. beads issues - checked for open status
// 4. workspaces - checked for existence and session ID file
//
// This function is designed for use in status commands and agent cleanup.
func IsLive(beadsID, serverURL, projectDir string) (tmuxLive, opencodeLive bool) {
	result := GetLiveness(beadsID, serverURL, projectDir)
	return result.TmuxLive, result.OpencodeLive
}

// GetLiveness returns detailed liveness information for an agent.
// This is the comprehensive version that returns all state information.
func GetLiveness(beadsID, serverURL, projectDir string) LivenessResult {
	result := LivenessResult{}

	if beadsID == "" {
		return result
	}

	// 1. Check workspace exists (fast, local file check)
	workspacePath, agentName := FindWorkspaceByBeadsID(projectDir, beadsID)
	if workspacePath != "" {
		result.WorkspaceExists = true
		result.WorkspacePath = workspacePath
		result.AgentName = agentName
	}

	// 2. Check beads issue status (shells out to bd)
	issue, err := verify.GetIssue(beadsID)
	if err == nil && issue != nil {
		result.BeadsOpen = issue.Status != "closed"
	}

	// 3. Check OpenCode session
	if serverURL != "" {
		result.OpencodeLive, result.SessionID = checkOpenCodeSession(serverURL, projectDir, beadsID, workspacePath)
	}

	// 4. Check tmux window
	result.TmuxLive, result.WindowID = checkTmuxWindow(beadsID)

	return result
}

// checkOpenCodeSession checks if an OpenCode session is active for the agent.
// It tries multiple approaches:
// 1. Read session ID from workspace .session_id file
// 2. Search sessions by title containing beads ID
func checkOpenCodeSession(serverURL, projectDir, beadsID, workspacePath string) (bool, string) {
	client := opencode.NewClient(serverURL)

	// Try 1: Read session ID from workspace file
	if workspacePath != "" {
		sessionFile := filepath.Join(workspacePath, ".session_id")
		if data, err := os.ReadFile(sessionFile); err == nil {
			sessionID := strings.TrimSpace(string(data))
			if sessionID != "" && client.SessionExists(sessionID) {
				return true, sessionID
			}
		}
	}

	// Try 2: Search sessions by title/beads ID match
	sessions, err := client.ListSessions(projectDir)
	if err != nil {
		return false, ""
	}

	for _, s := range sessions {
		// Match by beads ID in title (common pattern: "... [beadsID]" or "og-feat-X-beadsID-date")
		if strings.Contains(s.Title, beadsID) || extractBeadsIDFromTitle(s.Title) == beadsID {
			return true, s.ID
		}
	}

	return false, ""
}

// checkTmuxWindow checks if a tmux window exists for the agent.
// It searches all workers-* sessions for a window with the beads ID.
func checkTmuxWindow(beadsID string) (bool, string) {
	sessions, err := tmux.ListWorkersSessions()
	if err != nil {
		return false, ""
	}

	for _, session := range sessions {
		window, err := tmux.FindWindowByBeadsID(session, beadsID)
		if err == nil && window != nil {
			return true, window.ID
		}
	}

	return false, ""
}

// FindWorkspaceByBeadsID finds a workspace directory by beads ID.
// It searches .orch/workspace/ for directories that:
// 1. Contain the beads ID in the directory name
// 2. Have a SPAWN_CONTEXT.md that references the beads ID
//
// Returns (workspacePath, agentName) or ("", "") if not found.
func FindWorkspaceByBeadsID(projectDir, beadsID string) (string, string) {
	if projectDir == "" || beadsID == "" {
		return "", ""
	}

	workspaceBase := filepath.Join(projectDir, ".orch", "workspace")
	entries, err := os.ReadDir(workspaceBase)
	if err != nil {
		return "", ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		workspacePath := filepath.Join(workspaceBase, entry.Name())

		// Check 1: beads ID in directory name
		if strings.Contains(entry.Name(), beadsID) {
			return workspacePath, entry.Name()
		}

		// Check 2: beads ID in SPAWN_CONTEXT.md
		spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
		if content, err := os.ReadFile(spawnContextPath); err == nil {
			// Look for "spawned from beads issue: **beadsID**" pattern
			if containsBeadsIssueReference(string(content), beadsID) {
				return workspacePath, entry.Name()
			}
		}
	}

	return "", ""
}

// containsBeadsIssueReference checks if content contains the authoritative beads issue reference.
// The authoritative format is: "You were spawned from beads issue: **beadsID**"
func containsBeadsIssueReference(content, beadsID string) bool {
	// Pattern: "spawned from beads issue: **beadsID**"
	// This is the authoritative source - other mentions don't count
	pattern := regexp.MustCompile(`spawned from beads issue:\s*\*\*` + regexp.QuoteMeta(beadsID) + `\*\*`)
	return pattern.MatchString(content)
}

// extractBeadsIDFromTitle extracts a beads ID from an OpenCode session title.
// Common patterns:
// - "og-feat-X-beadsID-22dec"
// - "workspace [beadsID]"
// - "Task description [beadsID]"
func extractBeadsIDFromTitle(title string) string {
	// Try bracket pattern first: "[beadsID]"
	bracketPattern := regexp.MustCompile(`\[([a-zA-Z0-9-]+)\]`)
	matches := bracketPattern.FindStringSubmatch(title)
	if len(matches) >= 2 {
		return matches[1]
	}

	// Try workspace name pattern: "og-{skill}-{desc}-{beadsID}-{date}"
	// beads IDs are typically "project-hash" format
	parts := strings.Split(title, "-")
	if len(parts) >= 3 {
		// Look for a part that looks like a beads ID (contains alphanumeric hash)
		for i := len(parts) - 2; i >= 0; i-- {
			part := parts[i]
			// Skip date-like parts (22dec, etc.)
			if len(part) >= 5 && !isDatePart(part) {
				// Check if it looks like a beads ID component
				if looksLikeBeadsIDPart(part) {
					// Reconstruct potential beads ID
					if i > 0 {
						potentialID := parts[i-1] + "-" + part
						if looksLikeBeadsID(potentialID) {
							return potentialID
						}
					}
				}
			}
		}
	}

	return ""
}

// isDatePart checks if a string looks like a date suffix (e.g., "22dec")
func isDatePart(s string) bool {
	months := []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"}
	sLower := strings.ToLower(s)
	for _, m := range months {
		if strings.HasSuffix(sLower, m) {
			return true
		}
	}
	return false
}

// looksLikeBeadsIDPart checks if a string looks like part of a beads ID
func looksLikeBeadsIDPart(s string) bool {
	// Beads ID parts are typically 4+ alphanumeric characters
	if len(s) < 4 {
		return false
	}
	for _, c := range s {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// looksLikeBeadsID checks if a string looks like a beads ID (e.g., "proj-abc12")
func looksLikeBeadsID(s string) bool {
	// Beads IDs are typically "project-hash" format
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return false
	}
	// First part is project name (letters, possibly with hyphens within)
	// Second part is hash (alphanumeric, 4+ chars)
	return len(parts[0]) >= 2 && looksLikeBeadsIDPart(parts[1])
}
