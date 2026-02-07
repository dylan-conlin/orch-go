package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

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

	// Verify tmux is available before attempting session creation
	tmuxPath, err := findTmux()
	if err != nil {
		return "", fmt.Errorf("tmux not available: %w (install tmux or ensure it's in PATH)", err)
	}

	// Verify project directory exists (tmux will fail silently otherwise)
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return "", fmt.Errorf("project directory does not exist: %s", projectDir)
	}

	// Create new session with a "servers" window (matching Python behavior)
	// -d: detached mode
	// -s: session name
	// -n: initial window name (servers)
	// -c: working directory
	cmd, err := tmuxCommand("new-session", "-d", "-s", sessionName, "-n", "servers", "-c", projectDir)
	if err != nil {
		return "", fmt.Errorf("failed to build tmux command: %w", err)
	}

	// Use CombinedOutput to capture both stdout and stderr for better error messages
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Include tmux output in error message for debugging
		outputStr := strings.TrimSpace(string(output))
		if outputStr != "" {
			return "", fmt.Errorf("failed to create tmux session '%s': %s (tmux: %s)", sessionName, err, outputStr)
		}
		return "", fmt.Errorf("failed to create tmux session '%s': %w (tmux path: %s)", sessionName, err, tmuxPath)
	}

	// Verify session was created with helpful error message
	if !SessionExists(sessionName) {
		// Try to diagnose why session wasn't created
		listCmd, listErr := tmuxCommand("list-sessions")
		var diagInfo string
		if listErr == nil {
			listOutput, _ := listCmd.CombinedOutput()
			diagInfo = fmt.Sprintf(", existing sessions: %s", strings.TrimSpace(string(listOutput)))
		}
		return "", fmt.Errorf("tmux session '%s' was not created (projectDir: %s%s). Try manually: tmux new-session -d -s %s -n servers -c %s",
			sessionName, projectDir, diagInfo, sessionName, projectDir)
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

	// Verify tmux is available before attempting session creation
	tmuxPath, err := findTmux()
	if err != nil {
		return "", fmt.Errorf("tmux not available: %w (install tmux or ensure it's in PATH)", err)
	}

	// Create new orchestrator session
	// Note: We don't specify a working directory - orchestrators work across projects
	cmd, err := tmuxCommand("new-session", "-d", "-s", OrchestratorSessionName, "-n", "main")
	if err != nil {
		return "", fmt.Errorf("failed to build tmux command: %w", err)
	}

	// Use CombinedOutput to capture both stdout and stderr for better error messages
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		if outputStr != "" {
			return "", fmt.Errorf("failed to create orchestrator session: %s (tmux: %s)", err, outputStr)
		}
		return "", fmt.Errorf("failed to create orchestrator session: %w (tmux path: %s)", err, tmuxPath)
	}

	// Verify session was created with helpful error message
	if !SessionExists(OrchestratorSessionName) {
		return "", fmt.Errorf("orchestrator session was not created. Try manually: tmux new-session -d -s %s -n main", OrchestratorSessionName)
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

	// Verify tmux is available before attempting session creation
	tmuxPath, err := findTmux()
	if err != nil {
		return "", fmt.Errorf("tmux not available: %w (install tmux or ensure it's in PATH)", err)
	}

	// Create new meta-orchestrator session
	// Note: We don't specify a working directory - meta-orchestrators work across projects
	cmd, err := tmuxCommand("new-session", "-d", "-s", MetaOrchestratorSessionName, "-n", "main")
	if err != nil {
		return "", fmt.Errorf("failed to build tmux command: %w", err)
	}

	// Use CombinedOutput to capture both stdout and stderr for better error messages
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		if outputStr != "" {
			return "", fmt.Errorf("failed to create meta-orchestrator session: %s (tmux: %s)", err, outputStr)
		}
		return "", fmt.Errorf("failed to create meta-orchestrator session: %w (tmux path: %s)", err, tmuxPath)
	}

	// Verify session was created with helpful error message
	if !SessionExists(MetaOrchestratorSessionName) {
		return "", fmt.Errorf("meta-orchestrator session was not created. Try manually: tmux new-session -d -s %s -n main", MetaOrchestratorSessionName)
	}

	return MetaOrchestratorSessionName, nil
}

// KillSession kills a tmux session.
func KillSession(sessionName string) error {
	cmd, err := tmuxCommand("kill-session", "-t", sessionName)
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
	// Outside tmux: select the window in the existing session
	// This makes the window active without taking over the current terminal
	return tmuxCommand("select-window", "-t", windowTarget)
}

// Attach attaches the current terminal to a tmux window.
// If already inside tmux, it switches the client to the target window.
// If outside tmux, it selects the window without taking over the current terminal.
func Attach(windowTarget string) error {
	cmd, err := BuildAttachCommand(windowTarget, os.Getenv("TMUX") != "")
	if err != nil {
		return err
	}

	// When inside tmux, connect stdin/stdout/stderr so tmux can take over the terminal
	// When outside tmux, just run select-window (doesn't need terminal takeover)
	if os.Getenv("TMUX") != "" {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

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
