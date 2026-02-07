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
	Model         string
}

// RunConfig holds configuration for spawning an agent using 'opencode run'.
// DEPRECATED: Use StandaloneConfig instead for TUI spawning.
type RunConfig struct {
	ProjectDir string
	Model      string
	Title      string
	Prompt     string
}

// BuildRunCommand creates the opencode run command.
// DEPRECATED: Use BuildStandaloneCommand instead for TUI spawning.
func BuildRunCommand(cfg *RunConfig) *exec.Cmd {
	opencodeBin := "opencode"
	if bin := os.Getenv("OPENCODE_BIN"); bin != "" {
		opencodeBin = bin
	}

	args := []string{
		"run",
		"--model", cfg.Model,
		"--title", cfg.Title,
		cfg.Prompt,
	}
	cmd := exec.Command(opencodeBin, args...)
	cmd.Dir = cfg.ProjectDir
	// Set ORCH_WORKER=1 so agents know they are orch-managed workers
	cmd.Env = append(os.Environ(), "ORCH_WORKER=1")
	return cmd
}

// StandaloneConfig holds configuration for spawning an agent in standalone mode.
// DEPRECATED: Use OpencodeAttachConfig instead for dual TUI+API access.
type StandaloneConfig struct {
	ProjectDir string // Project directory (passed as first arg to opencode)
	Model      string // Model in format "provider/model"
}

// BuildStandaloneCommand creates the opencode standalone mode command string.
// DEPRECATED: Use BuildOpencodeAttachCommand instead for dual TUI+API access.
func BuildStandaloneCommand(cfg *StandaloneConfig) string {
	opencodeBin := "opencode"
	if bin := os.Getenv("OPENCODE_BIN"); bin != "" {
		opencodeBin = bin
	}

	// Build command: ORCH_WORKER=1 opencode {project_dir} --model {model}
	// Quote project dir in case it has spaces
	// Prefix with ORCH_WORKER=1 so the spawned agent knows it's an orch-managed worker
	return fmt.Sprintf("ORCH_WORKER=1 %s %q --model %q", opencodeBin, cfg.ProjectDir, cfg.Model)
}

// OpencodeAttachConfig holds configuration for spawning an agent in attach mode.
// This is the preferred approach for TUI spawning - it connects to a shared
// server, making sessions visible via API while still showing the TUI.
type OpencodeAttachConfig struct {
	ServerURL  string // http://127.0.0.1:4096
	ProjectDir string
	SessionID  string // optional: attach to pre-created session (e.g., created via API with specific model)
}

// BuildOpencodeAttachCommand creates the opencode command string for tmux spawning.
// Uses "opencode attach <url> --dir <project>" to connect to shared server, making
// sessions visible via API (enabling session ID capture, orch status, resume).
// OpenCode commit 18b26856a fixed Session.create to respect the directory parameter.
// Sets ORCH_WORKER=1 so agents know they are orch-managed workers.
func BuildOpencodeAttachCommand(cfg *OpencodeAttachConfig) string {
	opencodeBin := "opencode"
	if bin := os.Getenv("OPENCODE_BIN"); bin != "" {
		opencodeBin = bin
	}

	// Use attach mode with --dir to connect to shared server
	// This makes sessions visible via API for session ID capture
	// Note: opencode attach does NOT accept --model. To spawn with a specific model,
	// create the session via API first (client.CreateSession), then attach with --session.
	cmd := fmt.Sprintf("ORCH_WORKER=1 %s attach %q --dir %q", opencodeBin, cfg.ServerURL, cfg.ProjectDir)

	// Continue existing session if provided (used when session pre-created via API with model)
	if cfg.SessionID != "" {
		cmd += fmt.Sprintf(" --session %q", cfg.SessionID)
	}
	return cmd
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
	opencodeBin := "opencode"
	if bin := os.Getenv("OPENCODE_BIN"); bin != "" {
		opencodeBin = bin
	}

	args := []string{
		"run",
		"--attach", cfg.ServerURL,
		"--title", cfg.Title,
		cfg.Prompt,
	}
	cmd := exec.Command(opencodeBin, args...)
	cmd.Dir = cfg.ProjectDir
	// Set ORCH_WORKER=1 so agents know they are orch-managed workers
	cmd.Env = append(os.Environ(), "ORCH_WORKER=1")
	return cmd
}
