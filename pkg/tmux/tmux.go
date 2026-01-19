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
	Model      string
	SessionID  string // optional: continue existing session
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
	cmd := fmt.Sprintf("ORCH_WORKER=1 %s attach %q --dir %q", opencodeBin, cfg.ServerURL, cfg.ProjectDir)

	// Add model if provided
	if cfg.Model != "" {
		cmd += fmt.Sprintf(" --model %q", cfg.Model)
	}

	// Continue existing session if provided
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
	// Set ORCH_WORKER=1 so agents know they are orch-managed workers
	cmd.Env = append(os.Environ(), "ORCH_WORKER=1")
	return cmd
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
		return "", err
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
		return "", err
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
		return "", err
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

// KillSession kills a tmux session.
func KillSession(sessionName string) error {
	cmd, err := tmuxCommand("kill-session", "-t", sessionName)
	if err != nil {
		return err
	}
	return cmd.Run()
}

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
		return err
	}

	// Connect stdin/stdout/stderr so tmux can take over the terminal
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ListWorkersSessions returns all tmux sessions starting with "workers-".
func ListWorkersSessions() ([]string, error) {
	cmd, err := tmuxCommand("list-sessions", "-F", "#{session_name}")
	if err != nil {
		return nil, nil
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
