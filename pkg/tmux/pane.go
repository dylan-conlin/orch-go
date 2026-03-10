// pane.go contains pane inspection, liveness detection, and content capture functions.

package tmux

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetPaneCurrentCommand returns the current foreground command running in a window's pane.
// Uses the window's unique ID (e.g., "@1234") for targeting.
// Returns the command name (e.g., "claude", "zsh", "node").
//
// Note: On macOS with tmux 3.5+, this may report "zsh" even when a child process
// (like claude) is running in the foreground. Use GetPanePID + child process
// checking for more reliable liveness detection.
func GetPaneCurrentCommand(windowID string) (string, error) {
	cmd, err := tmuxCommand("list-panes", "-t", windowID, "-F", "#{pane_current_command}")
	if err != nil {
		return "", fmt.Errorf("failed to create tmux command: %w", err)
	}
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get pane command for %s: %w", windowID, err)
	}
	// Take first line (first pane — windows typically have one pane)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return "", fmt.Errorf("no pane found for window %s", windowID)
	}
	return lines[0], nil
}

// GetPanePID returns the PID of the process running in a window's first pane.
// Uses the window's unique ID (e.g., "@1234") for targeting.
func GetPanePID(windowID string) (string, error) {
	cmd, err := tmuxCommand("list-panes", "-t", windowID, "-F", "#{pane_pid}")
	if err != nil {
		return "", fmt.Errorf("failed to create tmux command: %w", err)
	}
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get pane PID for %s: %w", windowID, err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return "", fmt.Errorf("no pane found for window %s", windowID)
	}
	return lines[0], nil
}

// idleShellCommands are process names that indicate a pane is idle (no agent running).
var idleShellCommands = map[string]bool{
	"zsh": true, "bash": true, "sh": true, "fish": true,
	"-zsh": true, "-bash": true, "-sh": true, "login": true,
}

// IsPaneActive checks if a tmux window's pane has an active (non-shell) process.
// Uses two signals for reliability:
//  1. pane_current_command — if a non-shell process is in the foreground, the pane is active
//  2. child process detection — if the pane's shell has child processes, an agent is running
//
// The dual approach is needed because macOS tmux may report "zsh" for pane_current_command
// even when a child process (like claude) is actively running in the foreground.
//
// Returns true (conservative) if the check fails, to avoid counting active agents as dead.
func IsPaneActive(windowID string) bool {
	// Signal 1: Check pane_current_command — if it's not a shell, something is running.
	cmd, err := GetPaneCurrentCommand(windowID)
	if err != nil {
		// Can't determine — be conservative, assume alive
		return true
	}
	if !idleShellCommands[cmd] {
		return true
	}

	// Signal 2: pane_current_command shows a shell, but on macOS this can be
	// unreliable when child processes (claude, opencode) are running.
	// Check if the pane's shell PID has any child processes.
	pid, err := GetPanePID(windowID)
	if err != nil {
		// Can't determine — be conservative, assume alive
		return true
	}
	return hasChildProcesses(pid)
}

// hasChildProcesses checks if a process has any child processes.
// Uses pgrep -P which returns exit 0 if children found, 1 if not.
func hasChildProcesses(pid string) bool {
	cmd := exec.Command("pgrep", "-P", pid)
	err := cmd.Run()
	return err == nil // exit 0 = children found
}

// CaptureLines captures the last N lines from a tmux pane.
// If lines is 0, captures all visible content.
func CaptureLines(windowTarget string, lines int) ([]string, error) {
	var cmd *exec.Cmd
	var cmdErr error
	if lines > 0 {
		// Capture last N lines using negative start offset
		startLine := fmt.Sprintf("-%d", lines)
		cmd, cmdErr = tmuxCommand("capture-pane", "-t", windowTarget, "-p", "-S", startLine)
	} else {
		// Capture visible pane
		cmd, cmdErr = tmuxCommand("capture-pane", "-t", windowTarget, "-p")
	}
	if cmdErr != nil {
		return nil, fmt.Errorf("failed to create tmux command: %w", cmdErr)
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
