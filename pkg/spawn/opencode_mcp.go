package spawn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// OpenCodeMCPServerConfig defines an MCP server entry in opencode.json format.
// This differs from Claude's format: command is a single array (not command+args),
// and includes type and enabled fields.
type OpenCodeMCPServerConfig struct {
	Type    string   `json:"type"`
	Command []string `json:"command"`
	Enabled bool     `json:"enabled"`
}

// opencodeMCPPresets maps known MCP preset names to their OpenCode server configurations.
// Format matches OpenCode's opencode.json mcp config format.
//
// The default browser automation path is playwright-cli (standalone CLI tool),
// configured via BrowserTool field. The "playwright" MCP preset below is an
// opt-in override for interactive exploration via --mcp playwright.
var opencodeMCPPresets = map[string]OpenCodeMCPServerConfig{
	"playwright": {
		Type:    "stdio",
		Command: []string{"npx", "@anthropic-ai/mcp-server-playwright"},
		Enabled: true,
	},
}

// EnsureOpenCodeMCP reads (or creates) opencode.json in projectDir and merges
// the named MCP preset into the "mcp" key. Preserves all existing config.
// Returns an error if the preset is unknown.
func EnsureOpenCodeMCP(projectDir, preset string) error {
	server, ok := opencodeMCPPresets[preset]
	if !ok {
		return fmt.Errorf("unknown MCP preset %q for OpenCode backend", preset)
	}

	configPath := filepath.Join(projectDir, "opencode.json")

	// Read existing config or start with empty object
	config := make(map[string]interface{})
	data, err := os.ReadFile(configPath)
	if err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing opencode.json: %w", err)
		}
	}
	// If file doesn't exist, config stays as empty map — that's fine

	// Get or create the "mcp" section
	mcpSection, ok := config["mcp"].(map[string]interface{})
	if !ok {
		mcpSection = make(map[string]interface{})
	}

	// Add the preset (overwrites if already present — ensures latest config)
	mcpSection[preset] = server

	config["mcp"] = mcpSection

	// Write back with indentation for readability
	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal opencode.json: %w", err)
	}
	out = append(out, '\n')

	if err := os.WriteFile(configPath, out, 0644); err != nil {
		return fmt.Errorf("failed to write opencode.json: %w", err)
	}

	return nil
}
