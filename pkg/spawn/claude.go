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
//
// Note: The default browser automation path is playwright-cli (standalone CLI tool),
// configured via BrowserTool field and needs:playwright label. The "playwright" MCP
// preset below is an opt-in override for interactive exploration via --mcp playwright.
var mcpPresets = map[string]MCPServerConfig{
	"playwright": {
		Command: "npx",
		Args:    []string{"@anthropic-ai/mcp-server-playwright"},
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
// - When beadsDir is set (cross-repo spawn), injects BEADS_DIR env var so
//   bd comment/show commands reach the source project's beads database
// - When effort is set, adds --effort flag for reasoning effort optimization
// - When maxTurns > 0, adds --max-turns flag to prevent runaway agents
// - When settings is set, adds --settings flag for worker hook isolation
func BuildClaudeLaunchCommand(contextPath, claudeContext, mcp, configDir, beadsDir, beadsID, effort string, maxTurns int, settings string) string {
	// Account isolation prefix: when configDir is set and non-default,
	// unset the OAuth token and set CLAUDE_CONFIG_DIR so the Claude CLI
	// uses the correct account's config directory.
	accountPrefix := ""
	if configDir != "" && configDir != "~/.claude" {
		accountPrefix = fmt.Sprintf("unset CLAUDE_CODE_OAUTH_TOKEN; export CLAUDE_CONFIG_DIR=%s; ", configDir)
	}

	// Cross-repo beads prefix: when beadsDir is set, inject BEADS_DIR so
	// bd commands (comment, show) reach the source project's beads database.
	// Without this, cross-repo agents can't report Phase: Complete because
	// bd defaults to the .beads/ in the current working directory.
	beadsDirPrefix := ""
	if beadsDir != "" {
		beadsDirPrefix = fmt.Sprintf("export BEADS_DIR=%s; ", beadsDir)
	}

	// Base command: export CLAUDE_CONTEXT=X; cat CONTEXT.md | claude --dangerously-skip-permissions
	mcpFlag := ""
	if mcp != "" {
		// MCP values are treated as MCP server presets or raw config.
		// Browser automation via playwright-cli is handled separately via BrowserTool field.
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

	// Beads ID prefix: when beadsID is set, export ORCH_BEADS_ID so Stop hooks
	// can reliably find the beads issue without parsing messages.
	beadsIDPrefix := ""
	if beadsID != "" {
		beadsIDPrefix = fmt.Sprintf("export ORCH_BEADS_ID=%s; ", beadsID)
	}

	// Effort flag: controls reasoning effort level for cost/speed optimization.
	effortFlag := ""
	if effort != "" {
		effortFlag = fmt.Sprintf(" --effort %s", effort)
	}

	// Max turns flag: prevents runaway agents by limiting agentic turns.
	maxTurnsFlag := ""
	if maxTurns > 0 {
		maxTurnsFlag = fmt.Sprintf(" --max-turns %d", maxTurns)
	}

	// Settings flag: path to settings.json for worker hook isolation.
	settingsFlag := ""
	if settings != "" {
		settingsFlag = fmt.Sprintf(" --settings %q", settings)
	}

	return fmt.Sprintf("%s%s%sexport ORCH_SPAWNED=1; export CLAUDE_CONTEXT=%s; cat %q | claude --dangerously-skip-permissions%s%s%s%s%s", accountPrefix, beadsDirPrefix, beadsIDPrefix, claudeContext, contextPath, effortFlag, mcpFlag, disallowFlag, maxTurnsFlag, settingsFlag)
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

	launchCmd := BuildClaudeLaunchCommand(contextPath, cfg.ClaudeContext(), cfg.MCP, cfg.AccountConfigDir, cfg.BeadsDir, cfg.BeadsID, cfg.Effort, cfg.MaxTurns, cfg.Settings)

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
// Uses SendTextAndSubmit with a delay between text and Enter to ensure
// the TUI processes the pasted text before the submit keystroke arrives.
func SendClaude(windowTarget, keys string) error {
	return tmux.SendTextAndSubmit(windowTarget, keys, tmux.DefaultSendDelay)
}

// AbandonClaude kills the tmux window running the Claude agent.
func AbandonClaude(windowTarget string) error {
	if err := tmux.KillWindow(windowTarget); err != nil {
		return fmt.Errorf("failed to kill tmux window: %w", err)
	}
	return nil
}
