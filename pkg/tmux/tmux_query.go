package tmux

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GetPaneContent captures the content of a tmux pane.
func GetPaneContent(windowTarget string) (string, error) {
	cmd, err := tmuxCommand("capture-pane", "-t", windowTarget, "-p")
	if err != nil {
		return "", err
	}
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// IsOpenCodeReady checks if OpenCode TUI is ready based on pane content.
// Returns true when the TUI displays the prompt box AND either agent selector or command hints.
func IsOpenCodeReady(content string) bool {
	contentLower := strings.ToLower(content)

	// OpenCode TUI indicators - need BOTH visual box AND agent selector
	// The agent selector (showing "Build" or agent name) indicates the
	// TUI is fully initialized and ready for input
	hasPromptBox := strings.Contains(content, "┃") // Thick vertical bar used by OpenCode
	hasAgentSelector := strings.Contains(contentLower, "build") || strings.Contains(contentLower, "agent")
	hasCommandHint := strings.Contains(contentLower, "alt+x") || strings.Contains(contentLower, "commands")

	// TUI is ready when we see the prompt box AND either agent selector or command hints
	return hasPromptBox && (hasAgentSelector || hasCommandHint)
}

// WaitForOpenCodeReady waits for OpenCode TUI to be ready in the tmux window.
// Polls the pane content until TUI indicators are present or timeout is reached.
func WaitForOpenCodeReady(windowTarget string, cfg WaitConfig) error {
	start := time.Now()

	for time.Since(start) < cfg.Timeout {
		content, err := GetPaneContent(windowTarget)
		if err != nil {
			// Pane capture failed - window may have closed
			return fmt.Errorf("failed to capture pane content: %w", err)
		}

		if IsOpenCodeReady(content) {
			return nil
		}

		time.Sleep(cfg.PollInterval)
	}

	return fmt.Errorf("timeout waiting for OpenCode TUI to be ready after %v", cfg.Timeout)
}

// SendPromptAfterReady waits for OpenCode to be ready, then types the prompt.
// This is the high-level function that orchestrates:
// 1. Wait for TUI ready
// 2. Sleep for post-ready delay (letting input focus settle)
// 3. Send prompt via send-keys -l (literal mode)
// 4. Send Enter to submit
func SendPromptAfterReady(windowTarget, prompt string, waitCfg WaitConfig, sendCfg SendPromptConfig) error {
	// Wait for TUI to be ready
	if err := WaitForOpenCodeReady(windowTarget, waitCfg); err != nil {
		return err
	}

	// Wait for input focus to settle (TUI needs time after visual render)
	time.Sleep(sendCfg.PostReadyDelay)

	// Type the prompt in literal mode (handles special characters)
	if err := SendKeysLiteral(windowTarget, prompt); err != nil {
		return fmt.Errorf("failed to send prompt: %w", err)
	}

	// Send Enter to submit
	if err := SendEnter(windowTarget); err != nil {
		return fmt.Errorf("failed to send enter: %w", err)
	}

	return nil
}

// WindowExists checks if a window exists.
func WindowExists(windowTarget string) bool {
	// Use list-windows to check if window exists
	parts := strings.SplitN(windowTarget, ":", 2)
	if len(parts) != 2 {
		return false
	}
	sessionName := parts[0]
	windowIndex := parts[1]

	cmd, err := tmuxCommand("list-windows", "-t", sessionName, "-F", "#{window_index}")
	if err != nil {
		return false
	}
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	for _, line := range strings.Split(string(output), "\n") {
		if strings.TrimSpace(line) == windowIndex {
			return true
		}
	}
	return false
}

// ListAllWindowTargets returns a set of all existing window targets across all sessions.
// Returns targets in "session:index" format for fast O(1) existence checks.
// This is much more efficient than calling WindowExists() multiple times.
func ListAllWindowTargets() map[string]bool {
	result := make(map[string]bool)

	cmd, err := tmuxCommand("list-windows", "-a", "-F", "#{session_name}:#{window_index}")
	if err != nil {
		return result
	}
	output, err := cmd.Output()
	if err != nil {
		return result
	}

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result[line] = true
		}
	}
	return result
}

// ListWindowIDs returns all window IDs in a session.
func ListWindowIDs(sessionName string) ([]string, error) {
	cmd, err := tmuxCommand("list-windows", "-t", sessionName, "-F", "#{window_id}")
	if err != nil {
		return nil, err
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
		return nil, err
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

// FindWindowByWorkspaceNameAllSessions searches all workers sessions, the orchestrator session,
// and the meta-orchestrator session for a window with the given workspace name.
// This is useful when we don't know which project session the window is in.
// Returns the window info and session name, or nil if not found.
func FindWindowByWorkspaceNameAllSessions(workspaceName string) (*WindowInfo, string, error) {
	sessions, err := ListWorkersSessions()
	if err != nil {
		return nil, "", err
	}

	// Also search the orchestrator session (orchestrator spawns go there)
	if SessionExists(OrchestratorSessionName) {
		sessions = append(sessions, OrchestratorSessionName)
	}

	// Also search the meta-orchestrator session (meta-orchestrator spawns go there)
	if SessionExists(MetaOrchestratorSessionName) {
		sessions = append(sessions, MetaOrchestratorSessionName)
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
	sessions, err := ListWorkersSessions()
	if err != nil {
		return nil, "", err
	}

	// Also search the orchestrator session (orchestrator spawns go there)
	if SessionExists(OrchestratorSessionName) {
		sessions = append(sessions, OrchestratorSessionName)
	}

	// Also search the meta-orchestrator session (meta-orchestrator spawns go there)
	if SessionExists(MetaOrchestratorSessionName) {
		sessions = append(sessions, MetaOrchestratorSessionName)
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

// PaneActivity holds information about activity in a tmux pane.
type PaneActivity struct {
	CurrentCommand string    // The command currently running in the pane (e.g., "claude", "opencode", "zsh")
	PanePID        string    // The PID of the shell process in the pane
	ActivityTime   time.Time // Last activity timestamp
	IsActive       bool      // True if there's a non-shell process running
}

// GetPaneActivity returns activity information for a tmux pane.
// This is useful for detecting if an agent is actively running in a tmux window.
// windowTarget can be "session:window" format or a window ID like "@1234".
func GetPaneActivity(windowTarget string) (*PaneActivity, error) {
	// Query pane info: current_command, pane_pid, pane_activity
	cmd, err := tmuxCommand("display-message", "-t", windowTarget, "-p",
		"#{pane_current_command}:#{pane_pid}:#{pane_activity}")
	if err != nil {
		return nil, err
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get pane activity: %w", err)
	}

	// Parse "command:pid:activity_timestamp"
	outputStr := strings.TrimSpace(string(output))
	parts := strings.SplitN(outputStr, ":", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("unexpected pane activity format: %s", outputStr)
	}

	activity := &PaneActivity{
		CurrentCommand: parts[0],
		PanePID:        parts[1],
	}

	// Parse activity timestamp (Unix timestamp)
	if activityTS := parts[2]; activityTS != "" {
		if ts, err := parseUnixTimestamp(activityTS); err == nil {
			activity.ActivityTime = ts
		}
	}

	// Determine if the pane is active (running a non-shell process)
	// Common shells: bash, zsh, sh, fish, dash
	// If current command is NOT a shell, the pane is actively running something
	shellCommands := map[string]bool{
		"bash": true, "zsh": true, "sh": true, "fish": true, "dash": true,
		"tcsh": true, "csh": true, "ksh": true,
	}
	activity.IsActive = !shellCommands[activity.CurrentCommand]

	return activity, nil
}

// parseUnixTimestamp parses a Unix timestamp string to time.Time.
func parseUnixTimestamp(ts string) (time.Time, error) {
	var timestamp int64
	_, err := fmt.Sscanf(ts, "%d", &timestamp)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(timestamp, 0), nil
}

// IsPaneProcessRunning checks if a tmux pane has an active process running.
// Returns true if the pane is running a command (not just a shell prompt).
// This is useful for detecting if a claude/opencode agent is actively working.
func IsPaneProcessRunning(windowTarget string) bool {
	activity, err := GetPaneActivity(windowTarget)
	if err != nil {
		return false
	}
	return activity.IsActive
}

// CaptureLines captures the last N lines from a tmux pane.
// If lines is 0, captures all visible content.
func CaptureLines(windowTarget string, lines int) ([]string, error) {
	var cmd *exec.Cmd
	var err error
	if lines > 0 {
		// Capture last N lines using negative start offset
		startLine := fmt.Sprintf("-%d", lines)
		cmd, err = tmuxCommand("capture-pane", "-t", windowTarget, "-p", "-S", startLine)
	} else {
		// Capture visible pane
		cmd, err = tmuxCommand("capture-pane", "-t", windowTarget, "-p")
	}
	if err != nil {
		return nil, err
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to capture pane: %w", err)
	}

	var result []string
	for _, line := range strings.Split(string(output), "\n") {
		result = append(result, line)
	}
	return result, nil
}
