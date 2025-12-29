// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"encoding/json"
	"fmt"
	"os/user"
	"path/filepath"
)

// MCPServerType defines the type of MCP server connection.
type MCPServerType string

const (
	MCPServerTypeLocal  MCPServerType = "local"
	MCPServerTypeRemote MCPServerType = "remote"
)

// MCPServer represents a single MCP server configuration.
// This matches the OpenCode config format for MCP servers.
type MCPServer struct {
	Type    MCPServerType `json:"type"`
	Command []string      `json:"command,omitempty"` // For local servers
	URL     string        `json:"url,omitempty"`     // For remote servers
	Enabled bool          `json:"enabled"`
}

// MCPConfig represents the MCP portion of an OpenCode config.
// Used to enable/disable specific MCP servers for a spawn.
type MCPConfig struct {
	MCP map[string]MCPServer `json:"mcp,omitempty"`
}

// KnownMCPServers defines the predefined MCP server configurations.
// These are the servers that can be enabled via --mcp flag.
var KnownMCPServers = map[string]MCPServer{
	"glass": {
		Type:    MCPServerTypeLocal,
		Command: []string{glassBinPath(), "mcp"},
		Enabled: true,
	},
	"playwright": {
		Type:    MCPServerTypeLocal,
		Command: []string{"npx", "@playwright/mcp@latest", "--viewport-size=1440x900"},
		Enabled: true,
	},
}

// glassBinPath returns the path to the glass binary.
// Uses ~/bin/glass as the default location.
func glassBinPath() string {
	usr, err := user.Current()
	if err != nil {
		return "glass" // Fallback to PATH lookup
	}
	return filepath.Join(usr.HomeDir, "bin", "glass")
}

// GenerateMCPConfig generates an OpenCode config that enables the specified MCP server.
// Returns the JSON config string suitable for OPENCODE_CONFIG_CONTENT env var.
// Returns empty string if mcpName is empty or unknown.
func GenerateMCPConfig(mcpName string) (string, error) {
	if mcpName == "" {
		return "", nil
	}

	server, ok := KnownMCPServers[mcpName]
	if !ok {
		return "", fmt.Errorf("unknown MCP server: %s (known servers: glass, playwright)", mcpName)
	}

	config := MCPConfig{
		MCP: map[string]MCPServer{
			mcpName: server,
		},
	}

	data, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal MCP config: %w", err)
	}

	return string(data), nil
}

// ValidateMCPName checks if the provided MCP name is valid.
// Returns nil if valid, error if unknown.
func ValidateMCPName(mcpName string) error {
	if mcpName == "" {
		return nil
	}
	if _, ok := KnownMCPServers[mcpName]; !ok {
		return fmt.Errorf("unknown MCP server: %s (known servers: glass, playwright)", mcpName)
	}
	return nil
}
