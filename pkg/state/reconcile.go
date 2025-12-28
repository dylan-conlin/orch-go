// Package state provides agent state reconciliation across multiple sources.
package state

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// AgentStatus represents the unified status of an agent.
// This is the single source of truth for agent state used by both CLI and API.
type AgentStatus string

const (
	// StatusRunning indicates the agent is actively processing (generating a response).
	StatusRunning AgentStatus = "running"

	// StatusIdle indicates the agent has an active session but is not currently processing.
	StatusIdle AgentStatus = "idle"

	// StatusCompleted indicates the agent's beads issue is closed (Phase: Complete reported).
	StatusCompleted AgentStatus = "completed"

	// StatusStale indicates the session exists but hasn't been updated within the threshold.
	// This is a transitional state - the session may be dead or just paused.
	StatusStale AgentStatus = "stale"
)

// AgentStatusResult contains comprehensive status information for an agent.
// This unifies the status determination logic used by both CLI and API.
type AgentStatusResult struct {
	// Status is the unified agent status (running/idle/completed/stale).
	Status AgentStatus

	// IsProcessing indicates if the agent is actively generating a response.
	// This is determined by checking if the last assistant message is incomplete.
	IsProcessing bool

	// SessionID is the OpenCode session ID if found.
	SessionID string

	// BeadsID is the beads issue ID for this agent.
	BeadsID string

	// Phase is the last reported phase from beads comments (e.g., "Planning", "Implementing", "Complete").
	Phase string

	// IsCompleted indicates if the beads issue is closed.
	IsCompleted bool

	// TmuxWindow is the tmux window target if the agent has a tmux window.
	TmuxWindow string

	// Runtime is the duration since the session was created.
	Runtime time.Duration

	// IdleTime is the duration since the session was last updated.
	IdleTime time.Duration

	// UpdatedAt is the timestamp of the last session update.
	UpdatedAt time.Time

	// CreatedAt is the timestamp when the session was created.
	CreatedAt time.Time
}

// DetermineAgentStatus determines the unified status of an agent from multiple sources.
// This is the single source of truth for agent state, used by both CLI and API.
//
// The function uses the following priority order:
// 1. If beads issue is closed → StatusCompleted
// 2. If session is actively processing → StatusRunning
// 3. If session is recent (within maxIdleTime) → StatusIdle
// 4. If session is old but exists → StatusStale
//
// Parameters:
// - client: OpenCode client for session queries
// - session: The OpenCode session to evaluate (can be nil if only checking beads)
// - beadsID: The beads issue ID
// - projectDir: The project directory for workspace lookups
// - maxIdleTime: Maximum time since last update to consider session "active"
//
// If multiple sessions exist for the same beadsID, the caller should pass the most
// recently updated session.
func DetermineAgentStatus(
	client *opencode.Client,
	session *opencode.Session,
	beadsID string,
	projectDir string,
	maxIdleTime time.Duration,
) AgentStatusResult {
	result := AgentStatusResult{
		BeadsID: beadsID,
		Status:  StatusStale, // Default to stale
	}

	now := time.Now()

	// Check beads issue status first - closed issues are always "completed"
	if beadsID != "" {
		issue, err := verify.GetIssue(beadsID)
		if err == nil && issue != nil {
			result.IsCompleted = strings.EqualFold(issue.Status, "closed")
			if result.IsCompleted {
				result.Status = StatusCompleted
			}
		}
	}

	// If no session, try to find one by beads ID
	if session == nil && client != nil && beadsID != "" {
		session = findSessionByBeadsID(client, beadsID, projectDir, maxIdleTime)
	}

	// Populate session-related fields
	if session != nil {
		result.SessionID = session.ID
		result.CreatedAt = time.Unix(session.Time.Created/1000, 0)
		result.UpdatedAt = time.Unix(session.Time.Updated/1000, 0)
		result.Runtime = now.Sub(result.CreatedAt)
		result.IdleTime = now.Sub(result.UpdatedAt)

		// If issue is completed, we're done
		if result.IsCompleted {
			return result
		}

		// Check if session is actively processing
		if client != nil {
			result.IsProcessing = client.IsSessionProcessing(session.ID)
		}

		// Determine status based on activity
		if result.IsProcessing {
			result.Status = StatusRunning
		} else if result.IdleTime <= maxIdleTime {
			result.Status = StatusIdle
		} else {
			result.Status = StatusStale
		}
	}

	// Check for tmux window
	if beadsID != "" {
		if window := findTmuxWindowByBeadsID(beadsID); window != nil {
			result.TmuxWindow = window.Target
		}
	}

	// Get phase from beads comments
	if beadsID != "" {
		comments, err := verify.GetComments(beadsID)
		if err == nil {
			phaseStatus := verify.ParsePhaseFromComments(comments)
			if phaseStatus.Found {
				result.Phase = phaseStatus.Phase
			}
		}
	}

	return result
}

// findSessionByBeadsID finds the most recently updated OpenCode session for a beads ID.
// It checks both workspace file and session title matching.
func findSessionByBeadsID(client *opencode.Client, beadsID, projectDir string, maxIdleTime time.Duration) *opencode.Session {
	// Try workspace file lookup for session ID (fast path)
	workspacePath, _ := FindWorkspaceByBeadsID(projectDir, beadsID)
	if workspacePath != "" {
		sessionFile := filepath.Join(workspacePath, ".session_id")
		if data, err := os.ReadFile(sessionFile); err == nil {
			sessionID := strings.TrimSpace(string(data))
			if sessionID != "" {
				if session, err := client.GetSession(sessionID); err == nil {
					return session
				}
			}
		}
	}

	// Try session title matching
	sessions, err := client.ListSessions(projectDir)
	if err != nil {
		return nil
	}

	var mostRecent *opencode.Session
	for i := range sessions {
		s := &sessions[i]
		// Match by beads ID in title
		if strings.Contains(s.Title, beadsID) || extractBeadsIDFromTitle(s.Title) == beadsID {
			if mostRecent == nil || s.Time.Updated > mostRecent.Time.Updated {
				mostRecent = s
			}
		}
	}

	return mostRecent
}

// findTmuxWindowByBeadsID finds a tmux window for the given beads ID.
func findTmuxWindowByBeadsID(beadsID string) *tmux.WindowInfo {
	sessions, err := tmux.ListWorkersSessions()
	if err != nil {
		return nil
	}

	for _, session := range sessions {
		window, err := tmux.FindWindowByBeadsID(session, beadsID)
		if err == nil && window != nil {
			return window
		}
	}

	return nil
}

// DetermineAgentStatusBatch determines status for multiple agents efficiently.
// It batches beads lookups and session queries to minimize API calls.
//
// Parameters:
// - client: OpenCode client for session queries
// - sessions: Map of beadsID -> OpenCode session
// - beadsIDs: List of beads IDs to check
// - projectDir: The project directory
// - maxIdleTime: Maximum idle time threshold
//
// Returns a map of beadsID -> AgentStatusResult.
func DetermineAgentStatusBatch(
	client *opencode.Client,
	sessions map[string]*opencode.Session,
	beadsIDs []string,
	projectDir string,
	maxIdleTime time.Duration,
) map[string]AgentStatusResult {
	results := make(map[string]AgentStatusResult, len(beadsIDs))

	// Batch fetch beads issues
	issues, _ := verify.GetIssuesBatch(beadsIDs)

	// Build project dirs map for cross-project agents
	projectDirs := make(map[string]string)
	for _, beadsID := range beadsIDs {
		workspacePath, _ := FindWorkspaceByBeadsID(projectDir, beadsID)
		if workspacePath != "" {
			if spawnContext, err := os.ReadFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md")); err == nil {
				if dir := extractProjectDirFromSpawnContext(string(spawnContext)); dir != "" {
					projectDirs[beadsID] = dir
				}
			}
		}
	}

	// Batch fetch beads comments with project dirs
	commentsMap := verify.GetCommentsBatchWithProjectDirs(beadsIDs, projectDirs)

	now := time.Now()

	for _, beadsID := range beadsIDs {
		result := AgentStatusResult{
			BeadsID: beadsID,
			Status:  StatusStale,
		}

		// Check issue status
		if issue, ok := issues[beadsID]; ok && issue != nil {
			result.IsCompleted = strings.EqualFold(issue.Status, "closed")
			if result.IsCompleted {
				result.Status = StatusCompleted
			}
		}

		// Get phase from comments
		if comments, ok := commentsMap[beadsID]; ok {
			phaseStatus := verify.ParsePhaseFromComments(comments)
			if phaseStatus.Found {
				result.Phase = phaseStatus.Phase
			}
		}

		// Get session
		session := sessions[beadsID]
		if session != nil {
			result.SessionID = session.ID
			result.CreatedAt = time.Unix(session.Time.Created/1000, 0)
			result.UpdatedAt = time.Unix(session.Time.Updated/1000, 0)
			result.Runtime = now.Sub(result.CreatedAt)
			result.IdleTime = now.Sub(result.UpdatedAt)

			// If not completed, determine status from session
			if !result.IsCompleted {
				if client != nil {
					result.IsProcessing = client.IsSessionProcessing(session.ID)
				}

				if result.IsProcessing {
					result.Status = StatusRunning
				} else if result.IdleTime <= maxIdleTime {
					result.Status = StatusIdle
				} else {
					result.Status = StatusStale
				}
			}
		}

		// Check for tmux window
		if window := findTmuxWindowByBeadsID(beadsID); window != nil {
			result.TmuxWindow = window.Target
		}

		results[beadsID] = result
	}

	return results
}

// extractProjectDirFromSpawnContext extracts PROJECT_DIR from SPAWN_CONTEXT.md content.
func extractProjectDirFromSpawnContext(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "PROJECT_DIR:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "PROJECT_DIR:"))
		}
	}
	return ""
}

// DetermineStatusFromSession determines agent status from session data without making API calls.
// This is a fast-path for the API that avoids IsSessionProcessing() HTTP calls.
//
// Parameters:
// - isProcessing: whether the session is actively processing (from SSE or cached state)
// - idleTime: duration since the session was last updated
// - isCompleted: whether the beads issue is closed
// - maxIdleTime: maximum idle time to consider "active"
//
// Returns the unified AgentStatus.
func DetermineStatusFromSession(isProcessing bool, idleTime time.Duration, isCompleted bool, maxIdleTime time.Duration) AgentStatus {
	if isCompleted {
		return StatusCompleted
	}
	if isProcessing {
		return StatusRunning
	}
	if idleTime <= maxIdleTime {
		return StatusIdle
	}
	return StatusStale
}

// StatusToAPIString converts an AgentStatus to the string used in API responses.
// This ensures consistency between CLI and API status strings.
func StatusToAPIString(status AgentStatus) string {
	switch status {
	case StatusRunning:
		return "active" // API uses "active" for running agents
	case StatusIdle:
		return "idle"
	case StatusCompleted:
		return "completed"
	case StatusStale:
		return "stale"
	default:
		return "unknown"
	}
}

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

// DefaultMaxIdleTime is the maximum time since last update to consider a session "active".
// Sessions idle longer than this are considered stale and not actively running.
const DefaultMaxIdleTime = 30 * time.Minute

// checkOpenCodeSession checks if an OpenCode session is active for the agent.
// It tries multiple approaches:
// 1. Read session ID from workspace .session_id file and verify it's recently active
// 2. Search sessions by title containing beads ID and verify activity
//
// NOTE: OpenCode persists sessions to disk, so we check activity time rather than
// just existence. A session is considered "active" if updated within DefaultMaxIdleTime.
func checkOpenCodeSession(serverURL, projectDir, beadsID, workspacePath string) (bool, string) {
	client := opencode.NewClient(serverURL)

	// Try 1: Read session ID from workspace file
	if workspacePath != "" {
		sessionFile := filepath.Join(workspacePath, ".session_id")
		if data, err := os.ReadFile(sessionFile); err == nil {
			sessionID := strings.TrimSpace(string(data))
			if sessionID != "" && client.IsSessionActive(sessionID, DefaultMaxIdleTime) {
				return true, sessionID
			}
		}
	}

	// Try 2: Search sessions by title/beads ID match
	// Use ListSessions without directory to get in-memory sessions only
	// (With directory header, OpenCode returns ALL disk-persisted sessions)
	sessions, err := client.ListSessions("")
	if err != nil {
		return false, ""
	}

	now := time.Now()
	for _, s := range sessions {
		// Match by beads ID in title (common pattern: "... [beadsID]" or "og-feat-X-beadsID-date")
		if strings.Contains(s.Title, beadsID) || extractBeadsIDFromTitle(s.Title) == beadsID {
			// Verify the session is recently active
			updatedAt := time.Unix(s.Time.Updated/1000, 0)
			if now.Sub(updatedAt) <= DefaultMaxIdleTime {
				return true, s.ID
			}
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
