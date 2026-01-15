// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"fmt"

	"github.com/dylan-conlin/orch-go/pkg/tmux"
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
		sessionName, err = tmux.EnsureOrchestratorSession()
	} else {
		sessionName, err = tmux.EnsureWorkersSession(cfg.Project, cfg.ProjectDir)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to ensure tmux session: %w", err)
	}

	// 2. Build window name with emoji and beads ID
	windowName := tmux.BuildWindowName(cfg.WorkspaceName, cfg.SkillName, cfg.BeadsID)

	// 3. Create detached window and get its target and ID
	windowTarget, windowID, err := tmux.CreateWindow(sessionName, windowName, cfg.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create tmux window: %w", err)
	}

	// 4. Launch claude using the SPAWN_CONTEXT.md file
	contextPath := cfg.ContextFilePath()
	// Command: cat SPAWN_CONTEXT.md | claude --dangerously-skip-permissions
	// Pipe the file content to claude (no --file flag exists)
	// Use --dangerously-skip-permissions to avoid blocking on edit prompts
	launchCmd := fmt.Sprintf("cat %q | claude --dangerously-skip-permissions", contextPath)

	if err := tmux.SendKeys(windowTarget, launchCmd); err != nil {
		return nil, fmt.Errorf("failed to send launch command: %w", err)
	}
	if err := tmux.SendEnter(windowTarget); err != nil {
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
	return tmux.GetPaneContent(windowTarget)
}

// SendClaude sends keys to the Claude agent's tmux pane, followed by Enter.
func SendClaude(windowTarget, keys string) error {
	// Use literal mode to handle special characters in the message
	if err := tmux.SendKeysLiteral(windowTarget, keys); err != nil {
		return fmt.Errorf("failed to send keys: %w", err)
	}
	// Send Enter to submit the message
	if err := tmux.SendEnter(windowTarget); err != nil {
		return fmt.Errorf("failed to send enter: %w", err)
	}
	return nil
}

// AbandonClaude kills the tmux window running the Claude agent.
func AbandonClaude(windowTarget string) error {
	if err := tmux.KillWindow(windowTarget); err != nil {
		return fmt.Errorf("failed to kill tmux window: %w", err)
	}
	return nil
}
