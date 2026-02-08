// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

var (
	ensureOrchestratorSession = tmux.EnsureOrchestratorSession
	ensureWorkersSession      = tmux.EnsureWorkersSession
	buildTmuxWindowName       = tmux.BuildWindowName
	createTmuxWindow          = tmux.CreateWindow
	sendTmuxKeys              = tmux.SendKeys
	sendTmuxKeysLiteral       = tmux.SendKeysLiteral
	sendTmuxEnter             = tmux.SendEnter
	killTmuxWindow            = tmux.KillWindow
	getTmuxPaneContent        = tmux.GetPaneContent
)

// SpawnClaude launches a Claude Code agent in a tmux window.
// It uses the SPAWN_CONTEXT.md file approach: claude --file SPAWN_CONTEXT.md
func SpawnClaude(cfg *Config) (*tmux.SpawnResult, error) {
	// 1. Ensure appropriate tmux session exists
	// Meta-orchestrators and orchestrators go into 'orchestrator' session
	// Workers go into 'workers-{project}' session
	var sessionName string
	var err error
	if cfg.IsMetaOrchestrator || cfg.IsOrchestrator {
		sessionName, err = ensureOrchestratorSession()
	} else {
		sessionName, err = ensureWorkersSession(cfg.Project, cfg.ProjectDir)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to ensure tmux session: %w", err)
	}

	// 2. Build window name with emoji and beads ID
	windowName := buildTmuxWindowName(cfg.WorkspaceName, cfg.SkillName, cfg.BeadsID)

	// 3. Create detached window and get its target and ID
	windowTarget, windowID, err := createTmuxWindow(sessionName, windowName, cfg.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create tmux window: %w", err)
	}

	// 4. Launch claude using the context file
	contextPath := cfg.ContextFilePath()

	// Determine CLAUDE_CONTEXT env var to signal hooks to skip duplicate injection
	var claudeContext string
	switch {
	case cfg.IsMetaOrchestrator:
		claudeContext = "meta-orchestrator"
	case cfg.IsOrchestrator:
		claudeContext = "orchestrator"
	default:
		claudeContext = "worker"
	}

	// Command: export CLAUDE_CONTEXT=X [ORCH_WORKER=1]; cat CONTEXT.md | claude --dangerously-skip-permissions
	// - Export env vars so SessionStart hooks skip duplicate context injection
	//   (must export, not inline, so claude inherits it through the pipe)
	// - For workers, also export ORCH_WORKER=1 so OpenCode can detect worker sessions
	// - Pipe the file content to claude (no --file flag exists)
	// - Use --dangerously-skip-permissions to avoid blocking on edit prompts
	var launchCmd string
	if claudeContext == "worker" {
		launchCmd = fmt.Sprintf("export CLAUDE_CONTEXT=%s ORCH_WORKER=1; cat %q | claude --dangerously-skip-permissions", claudeContext, contextPath)
	} else {
		launchCmd = fmt.Sprintf("export CLAUDE_CONTEXT=%s; cat %q | claude --dangerously-skip-permissions", claudeContext, contextPath)
	}

	if err := sendTmuxKeys(windowTarget, launchCmd); err != nil {
		return nil, fmt.Errorf("failed to send launch command: %w", err)
	}
	if err := sendTmuxEnter(windowTarget); err != nil {
		return nil, fmt.Errorf("failed to send enter: %w", err)
	}

	return &tmux.SpawnResult{
		Window:        windowTarget,
		WindowID:      windowID,
		WindowName:    windowName,
		WorkspaceName: cfg.WorkspaceName,
	}, nil
}

// MonitorClaude captures the current content of the Claude agent's tmux pane.
func MonitorClaude(windowTarget string) (string, error) {
	return getTmuxPaneContent(windowTarget)
}

// SendClaude sends keys to the Claude agent's tmux pane, followed by Enter.
func SendClaude(windowTarget, keys string) error {
	// Use literal mode to handle special characters in the message
	if err := sendTmuxKeysLiteral(windowTarget, keys); err != nil {
		return fmt.Errorf("failed to send keys: %w", err)
	}
	// Send Enter to submit the message
	if err := sendTmuxEnter(windowTarget); err != nil {
		return fmt.Errorf("failed to send enter: %w", err)
	}
	return nil
}

// AbandonClaude kills the tmux window running the Claude agent.
func AbandonClaude(windowTarget string) error {
	if err := killTmuxWindow(windowTarget); err != nil {
		return fmt.Errorf("failed to kill tmux window: %w", err)
	}
	return nil
}

// SpawnClaudeInline launches a Claude Code agent inline (blocking) in the current terminal.
// Unlike SpawnClaude, this runs directly without tmux, blocking until the session completes.
// This is useful for interactive orchestrator sessions where Dylan wants to collaborate
// directly with the agent in the current terminal.
func SpawnClaudeInline(cfg *Config) error {
	// Get context file path
	contextPath := cfg.ContextFilePath()

	// Determine CLAUDE_CONTEXT env var to signal hooks to skip duplicate injection
	var claudeContext string
	switch {
	case cfg.IsMetaOrchestrator:
		claudeContext = "meta-orchestrator"
	case cfg.IsOrchestrator:
		claudeContext = "orchestrator"
	default:
		claudeContext = "worker"
	}

	// Read the context file content
	contextContent, err := os.ReadFile(contextPath)
	if err != nil {
		return fmt.Errorf("failed to read context file: %w", err)
	}

	// Build claude command
	// Use --dangerously-skip-permissions to avoid blocking on edit prompts
	cmd := exec.Command("claude", "--dangerously-skip-permissions")
	cmd.Dir = cfg.ProjectDir
	cmd.Env = append(os.Environ(),
		"CLAUDE_CONTEXT="+claudeContext,
		"ORCH_WORKER=1",
	)

	// Connect stdin to the context content, then inherit from terminal
	// We pipe the context first, then claude continues with interactive input
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Connect stdout and stderr to terminal for interactive use
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start claude: %w", err)
	}

	// Write context to stdin and close to signal end of initial input
	if _, err := stdinPipe.Write(contextContent); err != nil {
		_ = stdinPipe.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
		return fmt.Errorf("failed to write context to stdin: %w", err)
	}
	stdinPipe.Close()

	// Wait for command to complete (blocking)
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("claude exited with error: %w", err)
	}

	return nil
}
