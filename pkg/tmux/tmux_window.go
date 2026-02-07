package tmux

import (
	"fmt"
	"os"
	"strings"
)

// GetCurrentWindowName returns the name of the current tmux window, or an error if not in tmux.
// Returns "default" as fallback if not in a tmux session.
// Window names are sanitized to be filesystem-safe (removes emojis, special chars, and spaces).
// Note: This uses tmuxCommandCurrent() to target the current tmux context (not main socket).
func GetCurrentWindowName() (string, error) {
	// Check if we're in a tmux session
	if os.Getenv("TMUX") == "" {
		return "default", nil
	}

	// Get the current window name using tmux display-message
	// Use tmuxCommandCurrent() because we want the window name where THIS process is running
	cmd, err := tmuxCommandCurrent("display-message", "-p", "#{window_name}")
	if err != nil {
		return "", fmt.Errorf("failed to create tmux command: %w", err)
	}

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get window name: %w", err)
	}

	windowName := strings.TrimSpace(string(output))
	if windowName == "" {
		return "default", nil
	}

	// Sanitize window name for filesystem safety
	// Remove emojis and special characters, replace spaces with hyphens
	sanitized := sanitizeWindowName(windowName)
	if sanitized == "" {
		return "default", nil
	}

	return sanitized, nil
}

// RenameCurrentWindow renames the current tmux window.
// Returns nil if not in a tmux session (no-op).
// Note: This uses tmuxCommandCurrent() to target the current tmux context (not main socket).
func RenameCurrentWindow(newName string) error {
	// Check if we're in a tmux session
	if os.Getenv("TMUX") == "" {
		return nil // Not in tmux, nothing to rename
	}

	// Get current window index to target the rename
	// Use tmuxCommandCurrent() because we want to rename the window where THIS process is running
	cmd, err := tmuxCommandCurrent("display-message", "-p", "#{window_index}")
	if err != nil {
		return fmt.Errorf("failed to create tmux command: %w", err)
	}

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get window index: %w", err)
	}

	windowIndex := strings.TrimSpace(string(output))
	if windowIndex == "" {
		return fmt.Errorf("failed to get window index: empty output")
	}

	// Rename the window using tmux rename-window
	renameCmd, err := tmuxCommandCurrent("rename-window", "-t", windowIndex, newName)
	if err != nil {
		return fmt.Errorf("failed to create rename command: %w", err)
	}

	if err := renameCmd.Run(); err != nil {
		return fmt.Errorf("failed to rename window: %w", err)
	}

	return nil
}

// sanitizeWindowName converts a tmux window name to a filesystem-safe string.
// Removes emojis, special characters, and replaces spaces with hyphens.
func sanitizeWindowName(name string) string {
	// Build result by filtering characters
	var result strings.Builder
	for _, r := range name {
		// Keep alphanumeric, dash, and underscore
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		} else if r == ' ' {
			// Replace spaces with hyphens
			result.WriteRune('-')
		}
		// Skip all other characters (emojis, brackets, special chars)
	}

	sanitized := result.String()

	// Remove leading/trailing hyphens and collapse multiple hyphens
	sanitized = strings.Trim(sanitized, "-")
	for strings.Contains(sanitized, "--") {
		sanitized = strings.ReplaceAll(sanitized, "--", "-")
	}

	return sanitized
}

// CreateWindow creates a new detached window in the session and returns window info.
func CreateWindow(sessionName, windowName, workDir string) (windowTarget string, windowID string, err error) {
	// Create detached window and get its index and ID
	// -d: detached
	// -P: print info
	// -F: format output
	cmd, err := tmuxCommand("new-window",
		"-t", sessionName,
		"-n", windowName,
		"-c", workDir,
		"-d", "-P", "-F", "#{window_index}:#{window_id}")
	if err != nil {
		return "", "", err
	}

	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to create window: %w", err)
	}

	// Parse "index:id" output (e.g., "5:@1234")
	outputStr := strings.TrimSpace(string(output))
	parts := strings.SplitN(outputStr, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected output format: %s", outputStr)
	}

	windowIndex := parts[0]
	windowID = parts[1]
	windowTarget = fmt.Sprintf("%s:%s", sessionName, windowIndex)

	return windowTarget, windowID, nil
}

// SendKeys sends keystrokes to a tmux window.
func SendKeys(windowTarget string, keys string) error {
	cmd, err := tmuxCommand("send-keys", "-t", windowTarget, keys)
	if err != nil {
		return err
	}
	return cmd.Run()
}

// SendKeysLiteral sends keystrokes in literal mode (no special char interpretation).
func SendKeysLiteral(windowTarget, keys string) error {
	cmd, err := tmuxCommand("send-keys", "-t", windowTarget, "-l", keys)
	if err != nil {
		return err
	}
	return cmd.Run()
}

// SendEnter sends an Enter keystroke to a tmux window.
func SendEnter(windowTarget string) error {
	return SendKeys(windowTarget, "Enter")
}

// SelectWindow selects (focuses) a window.
func SelectWindow(windowTarget string) error {
	cmd, err := tmuxCommand("select-window", "-t", windowTarget)
	if err != nil {
		return err
	}
	return cmd.Run()
}

// KillWindow closes a tmux window by target (session:window format).
func KillWindow(windowTarget string) error {
	cmd, err := tmuxCommand("kill-window", "-t", windowTarget)
	if err != nil {
		return err
	}
	return cmd.Run()
}

// KillWindowByID closes a tmux window by its unique ID (e.g., "@1234").
func KillWindowByID(windowID string) error {
	cmd, err := tmuxCommand("kill-window", "-t", windowID)
	if err != nil {
		return err
	}
	return cmd.Run()
}
