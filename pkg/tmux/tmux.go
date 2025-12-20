// Package tmux provides tmux session and window management for agent spawning.
package tmux

import (
	"fmt"
	"os/exec"
	"strings"
)

// SKILL_EMOJIS maps skill names to their display emojis.
var SKILL_EMOJIS = map[string]string{
	"investigation":        "🔬",
	"feature-impl":         "🏗️",
	"systematic-debugging": "🐛",
	"architect":            "📐",
	"codebase-audit":       "📋",
	"research":             "🔍",
}

// SpawnConfig holds configuration for spawning an agent in tmux.
type SpawnConfig struct {
	ServerURL     string
	Prompt        string
	Title         string
	ProjectDir    string
	WorkspaceName string
}

// SpawnResult holds the result of spawning an agent.
type SpawnResult struct {
	SessionID     string
	Window        string // e.g., "workers-orch-go:5"
	WindowID      string // e.g., "@1234"
	WindowName    string // e.g., "🔬 og-inv-test-19dec"
	WorkspaceName string
}

// IsAvailable checks if tmux is installed and available.
func IsAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// SessionExists checks if a tmux session exists.
func SessionExists(sessionName string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", sessionName)
	err := cmd.Run()
	return err == nil
}

// GetWorkersSessionName derives the per-project workers session name.
func GetWorkersSessionName(projectName string) string {
	return fmt.Sprintf("workers-%s", projectName)
}

// BuildWindowName creates a window name with emoji and optional beads ID.
func BuildWindowName(workspaceName, skillName, beadsID string) string {
	// Get emoji for skill
	emoji := "⚙️" // Default
	if e, ok := SKILL_EMOJIS[skillName]; ok {
		emoji = e
	}

	// Build window name
	name := fmt.Sprintf("%s %s", emoji, workspaceName)

	// Append beads ID if present
	if beadsID != "" {
		name = fmt.Sprintf("%s [%s]", name, beadsID)
	}

	return name
}

// BuildSpawnCommand creates the opencode command for spawning.
// Note: Does NOT include --format json because tmux spawn should show the TUI.
// Inline spawn uses --format json separately to parse session ID.
func BuildSpawnCommand(cfg *SpawnConfig) *exec.Cmd {
	args := []string{
		"run",
		"--attach", cfg.ServerURL,
		"--title", cfg.Title,
		cfg.Prompt,
	}
	cmd := exec.Command("opencode", args...)
	cmd.Dir = cfg.ProjectDir
	return cmd
}

// EnsureWorkersSession ensures the workers session exists, creating if needed.
// Returns the session name.
func EnsureWorkersSession(projectName, projectDir string) (string, error) {
	sessionName := GetWorkersSessionName(projectName)

	// Check if session already exists
	if SessionExists(sessionName) {
		return sessionName, nil
	}

	// Create new session with a "servers" window (matching Python behavior)
	// -d: detached mode
	// -s: session name
	// -n: initial window name (servers)
	// -c: working directory
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-n", "servers", "-c", projectDir)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	// Verify session was created
	if !SessionExists(sessionName) {
		return "", fmt.Errorf("session %s was not created", sessionName)
	}

	return sessionName, nil
}

// CreateWindow creates a new detached window in the session and returns window info.
func CreateWindow(sessionName, windowName, workDir string) (windowTarget string, windowID string, err error) {
	// Create detached window and get its index and ID
	// -d: detached
	// -P: print info
	// -F: format output
	cmd := exec.Command("tmux", "new-window",
		"-t", sessionName,
		"-n", windowName,
		"-c", workDir,
		"-d", "-P", "-F", "#{window_index}:#{window_id}")

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
	cmd := exec.Command("tmux", "send-keys", "-t", windowTarget, keys)
	return cmd.Run()
}

// SendKeysLiteral sends keystrokes in literal mode (no special char interpretation).
func SendKeysLiteral(windowTarget, keys string) error {
	cmd := exec.Command("tmux", "send-keys", "-t", windowTarget, "-l", keys)
	return cmd.Run()
}

// SendEnter sends an Enter keystroke to a tmux window.
func SendEnter(windowTarget string) error {
	return SendKeys(windowTarget, "Enter")
}

// SelectWindow selects (focuses) a window.
func SelectWindow(windowTarget string) error {
	cmd := exec.Command("tmux", "select-window", "-t", windowTarget)
	return cmd.Run()
}

// KillSession kills a tmux session.
func KillSession(sessionName string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	return cmd.Run()
}

// GetPaneContent captures the content of a tmux pane.
func GetPaneContent(windowTarget string) (string, error) {
	cmd := exec.Command("tmux", "capture-pane", "-t", windowTarget, "-p")
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

	cmd := exec.Command("tmux", "list-windows", "-t", sessionName, "-F", "#{window_index}")
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
