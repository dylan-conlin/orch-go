// Package tmux provides tmux session and window management for agent spawning.
package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// tmuxPath caches the resolved path to the tmux binary.
// mainSocket caches the main tmux socket path when running inside overmind.
// This is set once by findTmux() and reused for all subsequent calls.
var (
	tmuxPath     string
	mainSocket   string
	tmuxPathOnce sync.Once
	tmuxPathErr  error
)

// detectMainSocket determines the main tmux socket path.
// When running inside overmind's tmux, we need to explicitly target the main tmux socket
// because the default behavior connects to overmind's tmux server instead.
// Returns empty string if not running inside tmux, or if running in main tmux already.
func detectMainSocket() string {
	// Check if we're in a tmux session
	tmuxEnv := os.Getenv("TMUX")
	if tmuxEnv == "" {
		// Not in tmux - no socket override needed
		return ""
	}

	// Check if we're inside overmind's tmux (socket path contains "overmind")
	if !strings.Contains(tmuxEnv, "overmind") {
		// Inside regular tmux - no socket override needed
		return ""
	}

	// Inside overmind's tmux - need to target main socket
	// Extract socket path from $TMUX (format: socket_path,server_pid,session_id)
	parts := strings.Split(tmuxEnv, ",")
	if len(parts) < 1 {
		return ""
	}
	overmindSocket := parts[0]

	// Construct main socket path based on socket directory
	// Example: /private/tmp/tmux-501/overmind-orch-go-xyz -> /tmp/tmux-501/default
	socketDir := filepath.Dir(overmindSocket)
	mainSocketPath := filepath.Join(socketDir, "default")

	// Verify main socket exists
	if _, err := os.Stat(mainSocketPath); err != nil {
		// Main socket doesn't exist - can't override
		return ""
	}

	return mainSocketPath
}

// findTmux locates the tmux binary, checking common locations first.
// This handles cases where PATH doesn't include tmux (e.g., launchd-spawned processes).
// Also detects the main tmux socket when running inside overmind.
// The results are cached for subsequent calls.
func findTmux() (string, error) {
	tmuxPathOnce.Do(func() {
		// Common tmux locations in order of preference
		commonPaths := []string{
			"/opt/homebrew/bin/tmux", // macOS ARM (Homebrew)
			"/usr/local/bin/tmux",    // macOS Intel, Linux (Homebrew/manual)
			"/usr/bin/tmux",          // Linux system package
		}

		// Check common locations first
		for _, path := range commonPaths {
			if _, err := os.Stat(path); err == nil {
				tmuxPath = path
				// Also detect main socket when inside overmind
				mainSocket = detectMainSocket()
				return
			}
		}

		// Fall back to PATH lookup
		path, err := exec.LookPath("tmux")
		if err != nil {
			tmuxPathErr = fmt.Errorf("tmux not found in common locations or PATH: %w", err)
			return
		}
		tmuxPath = path
		// Also detect main socket when inside overmind
		mainSocket = detectMainSocket()
	})

	return tmuxPath, tmuxPathErr
}

// tmuxCommand creates an exec.Cmd for tmux with the given arguments.
// Uses the cached tmux path from findTmux().
// When running inside overmind's tmux, automatically adds -S flag to target main socket.
func tmuxCommand(args ...string) (*exec.Cmd, error) {
	path, err := findTmux()
	if err != nil {
		return nil, err
	}

	// If we detected a main socket (running inside overmind), prepend -S flag
	if mainSocket != "" {
		args = append([]string{"-S", mainSocket}, args...)
	}

	return exec.Command(path, args...), nil
}

// tmuxCommandCurrent creates an exec.Cmd for tmux targeting the current tmux context.
// This explicitly does NOT add the -S flag, even when inside overmind.
// Use this for operations on the current window (GetCurrentWindowName, RenameCurrentWindow).
func tmuxCommandCurrent(args ...string) (*exec.Cmd, error) {
	path, err := findTmux()
	if err != nil {
		return nil, err
	}
	// No socket override - use current tmux context
	return exec.Command(path, args...), nil
}

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

// IsAvailable checks if tmux is installed and available.
func IsAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// SessionExists checks if a tmux session exists.
func SessionExists(sessionName string) bool {
	cmd, err := tmuxCommand("has-session", "-t", sessionName)
	if err != nil {
		return false // tmux not found
	}
	return cmd.Run() == nil
}

// GetWorkersSessionName derives the per-project workers session name.
func GetWorkersSessionName(projectName string) string {
	return fmt.Sprintf("workers-%s", projectName)
}

// EnsureWorkersSession ensures the workers session exists, creating if needed.
// Also updates the tmuxinator config with current port allocations.
// Returns the session name.
func EnsureWorkersSession(projectName, projectDir string) (string, error) {
	sessionName := GetWorkersSessionName(projectName)

	// Check if session already exists
	if SessionExists(sessionName) {
		// Update tmuxinator config with current port allocations (non-blocking)
		go func() {
			_, _ = EnsureTmuxinatorConfig(projectName, projectDir)
		}()
		return sessionName, nil
	}

	// Update tmuxinator config before creating session
	// This ensures the config exists for future tmuxinator start commands
	if _, err := EnsureTmuxinatorConfig(projectName, projectDir); err != nil {
		// Log warning but continue - tmuxinator config is nice-to-have
		fmt.Fprintf(os.Stderr, "Warning: failed to update tmuxinator config: %v\n", err)
	}

	// Create new session with a "servers" window (matching Python behavior)
	// -d: detached mode
	// -s: session name
	// -n: initial window name (servers)
	// -c: working directory
	cmd, err := tmuxCommand("new-session", "-d", "-s", sessionName, "-n", "servers", "-c", projectDir)
	if err != nil {
		return "", fmt.Errorf("failed to create tmux command: %w", err)
	}
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	// Verify session was created
	if !SessionExists(sessionName) {
		return "", fmt.Errorf("session %s was not created", sessionName)
	}

	return sessionName, nil
}

// OrchestratorSessionName is the fixed name for the orchestrator tmux session.
const OrchestratorSessionName = "orchestrator"

// MetaOrchestratorSessionName is the fixed name for the meta-orchestrator tmux session.
// Meta-orchestrators get their own session to distinguish them from regular orchestrators.
const MetaOrchestratorSessionName = "meta-orchestrator"

// EnsureOrchestratorSession ensures the orchestrator session exists.
// Unlike workers sessions (per-project), there's a single orchestrator session.
// Returns the session name ("orchestrator").
func EnsureOrchestratorSession() (string, error) {
	// Check if session already exists
	if SessionExists(OrchestratorSessionName) {
		return OrchestratorSessionName, nil
	}

	// Create new orchestrator session
	// Note: We don't specify a working directory - orchestrators work across projects
	cmd, err := tmuxCommand("new-session", "-d", "-s", OrchestratorSessionName, "-n", "main")
	if err != nil {
		return "", fmt.Errorf("failed to create tmux command: %w", err)
	}
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create orchestrator session: %w", err)
	}

	// Verify session was created
	if !SessionExists(OrchestratorSessionName) {
		return "", fmt.Errorf("orchestrator session was not created")
	}

	return OrchestratorSessionName, nil
}

// EnsureMetaOrchestratorSession ensures the meta-orchestrator session exists.
// Meta-orchestrators get their own session to distinguish them from regular orchestrators
// when looking at tmux sessions.
// Returns the session name ("meta-orchestrator").
func EnsureMetaOrchestratorSession() (string, error) {
	// Check if session already exists
	if SessionExists(MetaOrchestratorSessionName) {
		return MetaOrchestratorSessionName, nil
	}

	// Create new meta-orchestrator session
	// Note: We don't specify a working directory - meta-orchestrators work across projects
	cmd, err := tmuxCommand("new-session", "-d", "-s", MetaOrchestratorSessionName, "-n", "main")
	if err != nil {
		return "", fmt.Errorf("failed to create tmux command: %w", err)
	}
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create meta-orchestrator session: %w", err)
	}

	// Verify session was created
	if !SessionExists(MetaOrchestratorSessionName) {
		return "", fmt.Errorf("meta-orchestrator session was not created")
	}

	return MetaOrchestratorSessionName, nil
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
		return "", "", fmt.Errorf("failed to create tmux command: %w", err)
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
		return fmt.Errorf("failed to create tmux command: %w", err)
	}
	return cmd.Run()
}

// SendKeysLiteral sends keystrokes in literal mode (no special char interpretation).
func SendKeysLiteral(windowTarget, keys string) error {
	cmd, err := tmuxCommand("send-keys", "-t", windowTarget, "-l", keys)
	if err != nil {
		return fmt.Errorf("failed to create tmux command: %w", err)
	}
	return cmd.Run()
}

// SendEnter sends an Enter keystroke to a tmux window.
func SendEnter(windowTarget string) error {
	return SendKeys(windowTarget, "Enter")
}

// SendTextAndSubmit sends literal text to a tmux pane, waits for the TUI to process it,
// then sends Enter to submit. The delay between text and Enter is critical — without it,
// Enter gets processed before the TUI has fully ingested the pasted text, causing the
// message to sit in the input area without submitting.
//
// This matches the Python orch-cli's proven pattern (send.py:101-110).
func SendTextAndSubmit(windowTarget, text string, delay time.Duration) error {
	// Send text in literal mode (handles special characters safely)
	if err := SendKeysLiteral(windowTarget, text); err != nil {
		return fmt.Errorf("failed to send text: %w", err)
	}

	// Wait for TUI to process the pasted text before sending Enter.
	// Without this delay, Enter arrives before the TUI event loop has finished
	// processing the literal characters, causing the submit to be missed.
	time.Sleep(delay)

	// Send Enter to submit
	if err := SendEnter(windowTarget); err != nil {
		return fmt.Errorf("failed to send enter: %w", err)
	}

	return nil
}

// DefaultSendDelay is the delay between typing text and pressing Enter in a TUI pane.
// 500ms is reliable for most cases; the Python orch-cli uses 1s for extra safety.
const DefaultSendDelay = 500 * time.Millisecond

// SelectWindow selects (focuses) a window.
func SelectWindow(windowTarget string) error {
	cmd, err := tmuxCommand("select-window", "-t", windowTarget)
	if err != nil {
		return fmt.Errorf("failed to create tmux command: %w", err)
	}
	return cmd.Run()
}

// KillSession kills a tmux session.
func KillSession(sessionName string) error {
	cmd, err := tmuxCommand("kill-session", "-t", sessionName)
	if err != nil {
		return fmt.Errorf("failed to create tmux command: %w", err)
	}
	return cmd.Run()
}

// GetPaneContent captures the content of a tmux pane.
func GetPaneContent(windowTarget string) (string, error) {
	cmd, err := tmuxCommand("capture-pane", "-t", windowTarget, "-p")
	if err != nil {
		return "", fmt.Errorf("failed to create tmux command: %w", err)
	}
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
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

// KillWindow closes a tmux window by target (session:window format).
func KillWindow(windowTarget string) error {
	cmd, err := tmuxCommand("kill-window", "-t", windowTarget)
	if err != nil {
		return fmt.Errorf("failed to create tmux command: %w", err)
	}
	return cmd.Run()
}

// KillWindowByID closes a tmux window by its unique ID (e.g., "@1234").
func KillWindowByID(windowID string) error {
	cmd, err := tmuxCommand("kill-window", "-t", windowID)
	if err != nil {
		return fmt.Errorf("failed to create tmux command: %w", err)
	}
	return cmd.Run()
}

// BuildAttachCommand creates the tmux command for attaching to a window.
// insideTmux should be true if the current process is running inside a tmux session.
func BuildAttachCommand(windowTarget string, insideTmux bool) (*exec.Cmd, error) {
	if insideTmux {
		// Inside tmux: switch client to the new window
		return tmuxCommand("switch-client", "-t", windowTarget)
	}
	// Outside tmux: attach to the session/window
	return tmuxCommand("attach-session", "-t", windowTarget)
}

// Attach attaches the current terminal to a tmux window.
// If already inside tmux, it switches the client to the target window.
// If outside tmux, it attaches to the session/window.
func Attach(windowTarget string) error {
	cmd, err := BuildAttachCommand(windowTarget, os.Getenv("TMUX") != "")
	if err != nil {
		return fmt.Errorf("failed to create tmux command: %w", err)
	}

	// Connect stdin/stdout/stderr so tmux can take over the terminal
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
