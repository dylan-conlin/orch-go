// Package tmux provides tmux session and window management for agent spawning.
package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
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

// SpawnConfig holds configuration for spawning an agent in tmux (attach mode).
type SpawnConfig struct {
	ServerURL     string
	Prompt        string
	Title         string
	ProjectDir    string
	WorkspaceName string
}

// StandaloneConfig holds configuration for spawning an agent in standalone mode.
// Standalone mode launches opencode without attaching to a server - each agent
// gets its own independent opencode instance.
type StandaloneConfig struct {
	ProjectDir string            // Working directory for the agent
	Model      string            // Model to use (e.g., "anthropic/claude-sonnet-4-20250514")
	EnvVars    map[string]string // Environment variables to set (e.g., ORCH_WORKER=true)
}

// WaitConfig holds configuration for waiting for OpenCode to be ready.
type WaitConfig struct {
	Timeout      time.Duration // Maximum time to wait for TUI to be ready
	PollInterval time.Duration // How often to check pane content
}

// DefaultWaitConfig returns the default wait configuration.
// Timeout: 15s, PollInterval: 200ms (matching Python behavior)
func DefaultWaitConfig() WaitConfig {
	return WaitConfig{
		Timeout:      15 * time.Second,
		PollInterval: 200 * time.Millisecond,
	}
}

// SendPromptConfig holds configuration for sending a prompt after TUI is ready.
type SendPromptConfig struct {
	PostReadyDelay time.Duration // How long to wait after TUI ready before typing
}

// DefaultSendPromptConfig returns the default send prompt configuration.
// PostReadyDelay: 1s (matching Python behavior - TUI needs time for input focus)
func DefaultSendPromptConfig() SendPromptConfig {
	return SendPromptConfig{
		PostReadyDelay: 1 * time.Second,
	}
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

// BuildSpawnCommand creates the opencode command for spawning (attach mode).
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

// BuildStandaloneCommand creates the opencode command for standalone mode.
// Standalone mode: opencode {dir} --model {model}
// This launches a TUI that needs prompt typed after it's ready.
func BuildStandaloneCommand(cfg *StandaloneConfig) *exec.Cmd {
	// Get opencode binary path - allow override for dev builds
	opencodeBin := "opencode"
	if bin := os.Getenv("OPENCODE_BIN"); bin != "" {
		opencodeBin = bin
	}

	args := []string{
		cfg.ProjectDir,
		"--model", cfg.Model,
	}
	cmd := exec.Command(opencodeBin, args...)
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
