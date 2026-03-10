// window_search.go contains window listing, searching, and querying functions
// across tmux sessions. Includes single-session and all-sessions search variants.

package tmux

import (
	"fmt"
	"strings"
)

// ListWorkersSessions returns all tmux sessions starting with "workers-".
func ListWorkersSessions() ([]string, error) {
	cmd, err := tmuxCommand("list-sessions", "-F", "#{session_name}")
	if err != nil {
		return nil, fmt.Errorf("failed to create tmux command: %w", err)
	}
	output, err := cmd.Output()
	if err != nil {
		// If no sessions exist, tmux returns error 1
		return nil, nil
	}

	var sessions []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "workers-") {
			sessions = append(sessions, line)
		}
	}
	return sessions, nil
}

// ListWindowIDs returns all window IDs in a session.
func ListWindowIDs(sessionName string) ([]string, error) {
	cmd, err := tmuxCommand("list-windows", "-t", sessionName, "-F", "#{window_id}")
	if err != nil {
		return nil, fmt.Errorf("failed to create tmux command: %w", err)
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list windows: %w", err)
	}

	var ids []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			ids = append(ids, line)
		}
	}
	return ids, nil
}

// WindowInfo holds information about a tmux window.
type WindowInfo struct {
	Index  string // Window index (e.g., "5")
	ID     string // Window ID (e.g., "@1234")
	Name   string // Window name
	Target string // session:index format
}

// ListWindows returns all windows in a session with their details.
func ListWindows(sessionName string) ([]WindowInfo, error) {
	// Format: index:id:name
	cmd, err := tmuxCommand("list-windows", "-t", sessionName, "-F", "#{window_index}:#{window_id}:#{window_name}")
	if err != nil {
		return nil, fmt.Errorf("failed to create tmux command: %w", err)
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list windows: %w", err)
	}

	var windows []WindowInfo
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Parse "index:id:name"
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}
		windows = append(windows, WindowInfo{
			Index:  parts[0],
			ID:     parts[1],
			Name:   parts[2],
			Target: fmt.Sprintf("%s:%s", sessionName, parts[0]),
		})
	}
	return windows, nil
}

// WindowMatch pairs a window with the session it was found in.
// Used by FindAll* functions that search across multiple sessions.
type WindowMatch struct {
	Window      WindowInfo
	SessionName string
}

// FindWindowByBeadsID finds a window by searching for beads ID in window name.
// Returns nil if not found (no error).
func FindWindowByBeadsID(sessionName, beadsID string) (*WindowInfo, error) {
	windows, err := ListWindows(sessionName)
	if err != nil {
		return nil, err
	}

	// Look for window with beads ID in name (format: "[beadsID]")
	searchPattern := fmt.Sprintf("[%s]", beadsID)
	for i := range windows {
		if strings.Contains(windows[i].Name, searchPattern) {
			return &windows[i], nil
		}
	}
	return nil, nil
}

// FindWindowByWorkspaceName finds a window by searching for workspace name in window name.
// Window names follow the pattern: "🔬 og-inv-topic-date [beads-id]" or "⚙️ og-feat-topic-date"
// Returns nil if not found (no error).
func FindWindowByWorkspaceName(sessionName, workspaceName string) (*WindowInfo, error) {
	windows, err := ListWindows(sessionName)
	if err != nil {
		return nil, err
	}

	// Look for window containing the workspace name
	// Workspace name is typically after the emoji and before the beads ID (if present)
	for i := range windows {
		if strings.Contains(windows[i].Name, workspaceName) {
			return &windows[i], nil
		}
	}
	return nil, nil
}

// allSessions returns all worker, orchestrator, and meta-orchestrator sessions.
// Used by cross-session search functions to avoid duplicating session enumeration.
func allSessions() ([]string, error) {
	sessions, err := ListWorkersSessions()
	if err != nil {
		return nil, err
	}

	if SessionExists(OrchestratorSessionName) {
		sessions = append(sessions, OrchestratorSessionName)
	}
	if SessionExists(MetaOrchestratorSessionName) {
		sessions = append(sessions, MetaOrchestratorSessionName)
	}

	return sessions, nil
}

// FindWindowByWorkspaceNameAllSessions searches all workers sessions, the orchestrator session,
// and the meta-orchestrator session for a window with the given workspace name.
// This is useful when we don't know which project session the window is in.
// Returns the window info and session name, or nil if not found.
func FindWindowByWorkspaceNameAllSessions(workspaceName string) (*WindowInfo, string, error) {
	sessions, err := allSessions()
	if err != nil {
		return nil, "", err
	}

	for _, sessionName := range sessions {
		window, err := FindWindowByWorkspaceName(sessionName, workspaceName)
		if err != nil {
			continue // Skip sessions that fail
		}
		if window != nil {
			return window, sessionName, nil
		}
	}
	return nil, "", nil
}

// FindWindowByBeadsIDAllSessions searches all workers sessions, the orchestrator session,
// and the meta-orchestrator session for a window with the given beads ID.
// This is useful when we don't know which project session the window is in.
// Returns the window info and session name, or nil if not found.
func FindWindowByBeadsIDAllSessions(beadsID string) (*WindowInfo, string, error) {
	sessions, err := allSessions()
	if err != nil {
		return nil, "", err
	}

	for _, sessionName := range sessions {
		window, err := FindWindowByBeadsID(sessionName, beadsID)
		if err != nil {
			continue // Skip sessions that fail
		}
		if window != nil {
			return window, sessionName, nil
		}
	}
	return nil, "", nil
}

// FindAllWindowsByBeadsID finds ALL windows matching a beads ID in a single session.
// Unlike FindWindowByBeadsID which returns only the first match, this returns all matches.
func FindAllWindowsByBeadsID(sessionName, beadsID string) ([]WindowInfo, error) {
	windows, err := ListWindows(sessionName)
	if err != nil {
		return nil, err
	}

	searchPattern := fmt.Sprintf("[%s]", beadsID)
	var matches []WindowInfo
	for _, w := range windows {
		if strings.Contains(w.Name, searchPattern) {
			matches = append(matches, w)
		}
	}
	return matches, nil
}

// FindAllWindowsByWorkspaceName finds ALL windows matching a workspace name in a single session.
// Unlike FindWindowByWorkspaceName which returns only the first match, this returns all matches.
func FindAllWindowsByWorkspaceName(sessionName, workspaceName string) ([]WindowInfo, error) {
	windows, err := ListWindows(sessionName)
	if err != nil {
		return nil, err
	}

	var matches []WindowInfo
	for _, w := range windows {
		if strings.Contains(w.Name, workspaceName) {
			matches = append(matches, w)
		}
	}
	return matches, nil
}

// FindAllWindowsByBeadsIDAllSessions searches all workers sessions, the orchestrator session,
// and the meta-orchestrator session for ALL windows with the given beads ID.
// Unlike FindWindowByBeadsIDAllSessions which returns only the first match,
// this returns every matching window across all sessions.
func FindAllWindowsByBeadsIDAllSessions(beadsID string) ([]WindowMatch, error) {
	sessions, err := allSessions()
	if err != nil {
		return nil, err
	}

	var allMatches []WindowMatch
	for _, sessionName := range sessions {
		windows, err := FindAllWindowsByBeadsID(sessionName, beadsID)
		if err != nil {
			continue // Skip sessions that fail
		}
		for _, w := range windows {
			allMatches = append(allMatches, WindowMatch{Window: w, SessionName: sessionName})
		}
	}
	return allMatches, nil
}

// FindAllWindowsByWorkspaceNameAllSessions searches all workers sessions, the orchestrator session,
// and the meta-orchestrator session for ALL windows with the given workspace name.
// Unlike FindWindowByWorkspaceNameAllSessions which returns only the first match,
// this returns every matching window across all sessions.
func FindAllWindowsByWorkspaceNameAllSessions(workspaceName string) ([]WindowMatch, error) {
	sessions, err := allSessions()
	if err != nil {
		return nil, err
	}

	var allMatches []WindowMatch
	for _, sessionName := range sessions {
		windows, err := FindAllWindowsByWorkspaceName(sessionName, workspaceName)
		if err != nil {
			continue // Skip sessions that fail
		}
		for _, w := range windows {
			allMatches = append(allMatches, WindowMatch{Window: w, SessionName: sessionName})
		}
	}
	return allMatches, nil
}

// WindowExistsByID checks if a tmux window exists by its unique ID (e.g., "@1234").
// Returns true if the window is still present in any tmux session.
func WindowExistsByID(windowID string) bool {
	// Query all tmux windows to find this ID
	cmd, err := tmuxCommand("list-windows", "-a", "-F", "#{window_id}")
	if err != nil {
		return false
	}
	output, err := cmd.Output()
	if err != nil {
		// If tmux isn't running or no sessions exist, window doesn't exist
		return false
	}

	// Check if this window ID is in the output
	for _, line := range strings.Split(string(output), "\n") {
		if strings.TrimSpace(line) == windowID {
			return true
		}
	}
	return false
}
