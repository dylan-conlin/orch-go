// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"encoding/json"
	"fmt"

	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// mcpPresets maps known MCP preset names to their server configurations.
// Each preset defines the command and args needed to launch the MCP server.
// Format matches Claude CLI's --mcp-config JSON format.
var mcpPresets = map[string]MCPServerConfig{
	"playwright": {
		Command: "npx",
		Args:    []string{"-y", "@playwright/mcp@latest", "--cdp-endpoint", "http://localhost:9222"},
	},
}

// MCPServerConfig defines the command to launch an MCP server.
type MCPServerConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// MCPConfigJSON returns the JSON string for --mcp-config given an MCP preset name.
// Returns the JSON config and true if the preset is known, or empty string and false if not.
// For unknown presets, the caller should treat the value as a raw JSON string or file path.
func MCPConfigJSON(preset string) (string, bool) {
	server, ok := mcpPresets[preset]
	if !ok {
		return "", false
	}

	config := map[string]map[string]MCPServerConfig{
		"mcpServers": {
			preset: server,
		},
	}

	data, err := json.Marshal(config)
	if err != nil {
		return "", false
	}
	return string(data), true
}

// BuildClaudeLaunchCommand constructs the shell command to launch Claude Code CLI.
// This is extracted from SpawnClaude for testability.
//
// The command:
// - Exports CLAUDE_CONTEXT env var so SessionStart hooks skip duplicate injection
// - Pipes the context file to claude (no --file flag exists)
// - Uses --dangerously-skip-permissions to avoid blocking on edit prompts
// - When MCP is set, adds --mcp-config with the appropriate JSON config
// - When configDir is set (and differs from default ~/.claude), injects
//   CLAUDE_CONFIG_DIR env var and unsets CLAUDE_CODE_OAUTH_TOKEN for account isolation
func BuildClaudeLaunchCommand(contextPath, claudeContext, mcp, configDir string) string {
	// Account isolation prefix: when configDir is set and non-default,
	// unset the OAuth token and set CLAUDE_CONFIG_DIR so the Claude CLI
	// uses the correct account's config directory.
	accountPrefix := ""
	if configDir != "" && configDir != "~/.claude" {
		accountPrefix = fmt.Sprintf("unset CLAUDE_CODE_OAUTH_TOKEN; export CLAUDE_CONFIG_DIR=%s; ", configDir)
	}

	// Base command: export CLAUDE_CONTEXT=X; cat CONTEXT.md | claude --dangerously-skip-permissions
	mcpFlag := ""
	if mcp != "" {
		// Check if it's a known preset
		if configJSON, ok := MCPConfigJSON(mcp); ok {
			// Use single quotes around JSON to avoid shell interpretation of double quotes
			mcpFlag = fmt.Sprintf(" --mcp-config '%s'", configJSON)
		} else {
			// Treat as raw value (could be a file path or custom JSON)
			mcpFlag = fmt.Sprintf(" --mcp-config '%s'", mcp)
		}
	}

	// Orchestrator tool restrictions: remove worker-level tools that orchestrators should not use
	disallowFlag := ""
	if claudeContext == "orchestrator" || claudeContext == "meta-orchestrator" {
		disallowFlag = " --disallowedTools 'Task,Edit,Write,NotebookEdit'"
	}

	return fmt.Sprintf("%sexport ORCH_SPAWNED=1; export CLAUDE_CONTEXT=%s; cat %q | claude --dangerously-skip-permissions%s%s", accountPrefix, claudeContext, contextPath, mcpFlag, disallowFlag)
}

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

	launchCmd := BuildClaudeLaunchCommand(contextPath, claudeContext, cfg.MCP, cfg.AccountConfigDir)

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
