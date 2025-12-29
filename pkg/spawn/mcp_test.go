package spawn

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateMCPConfig(t *testing.T) {
	tests := []struct {
		name       string
		mcpName    string
		wantErr    bool
		wantEmpty  bool
		wantServer string
		wantType   MCPServerType
	}{
		{
			name:      "empty mcp name returns empty string",
			mcpName:   "",
			wantEmpty: true,
		},
		{
			name:       "glass generates valid config",
			mcpName:    "glass",
			wantServer: "glass",
			wantType:   MCPServerTypeLocal,
		},
		{
			name:       "playwright generates valid config",
			mcpName:    "playwright",
			wantServer: "playwright",
			wantType:   MCPServerTypeLocal,
		},
		{
			name:    "unknown mcp name returns error",
			mcpName: "unknown-mcp",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateMCPConfig(tt.mcpName)

			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateMCPConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if tt.wantEmpty {
				if got != "" {
					t.Errorf("GenerateMCPConfig() = %q, want empty string", got)
				}
				return
			}

			// Parse the JSON to verify structure
			var config MCPConfig
			if err := json.Unmarshal([]byte(got), &config); err != nil {
				t.Errorf("GenerateMCPConfig() returned invalid JSON: %v", err)
				return
			}

			server, ok := config.MCP[tt.wantServer]
			if !ok {
				t.Errorf("GenerateMCPConfig() missing server %q in config", tt.wantServer)
				return
			}

			if server.Type != tt.wantType {
				t.Errorf("GenerateMCPConfig() server type = %v, want %v", server.Type, tt.wantType)
			}

			if !server.Enabled {
				t.Errorf("GenerateMCPConfig() server should be enabled")
			}
		})
	}
}

func TestGenerateMCPConfig_GlassPath(t *testing.T) {
	config, err := GenerateMCPConfig("glass")
	if err != nil {
		t.Fatalf("GenerateMCPConfig(glass) failed: %v", err)
	}

	var parsed MCPConfig
	if err := json.Unmarshal([]byte(config), &parsed); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	server := parsed.MCP["glass"]
	if len(server.Command) == 0 {
		t.Error("glass command should not be empty")
	}

	// Command should include "glass" and "mcp"
	cmdStr := strings.Join(server.Command, " ")
	if !strings.Contains(cmdStr, "glass") {
		t.Errorf("glass command should contain 'glass': %v", server.Command)
	}
	if !strings.Contains(cmdStr, "mcp") {
		t.Errorf("glass command should contain 'mcp': %v", server.Command)
	}
}

func TestGenerateMCPConfig_PlaywrightCommand(t *testing.T) {
	config, err := GenerateMCPConfig("playwright")
	if err != nil {
		t.Fatalf("GenerateMCPConfig(playwright) failed: %v", err)
	}

	var parsed MCPConfig
	if err := json.Unmarshal([]byte(config), &parsed); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	server := parsed.MCP["playwright"]
	if len(server.Command) == 0 {
		t.Error("playwright command should not be empty")
	}

	// Command should include npx and @playwright/mcp
	cmdStr := strings.Join(server.Command, " ")
	if !strings.Contains(cmdStr, "npx") {
		t.Errorf("playwright command should contain 'npx': %v", server.Command)
	}
	if !strings.Contains(cmdStr, "@playwright/mcp") {
		t.Errorf("playwright command should contain '@playwright/mcp': %v", server.Command)
	}
}

func TestValidateMCPName(t *testing.T) {
	tests := []struct {
		name    string
		mcpName string
		wantErr bool
	}{
		{
			name:    "empty is valid",
			mcpName: "",
			wantErr: false,
		},
		{
			name:    "glass is valid",
			mcpName: "glass",
			wantErr: false,
		},
		{
			name:    "playwright is valid",
			mcpName: "playwright",
			wantErr: false,
		},
		{
			name:    "unknown is invalid",
			mcpName: "unknown",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMCPName(tt.mcpName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMCPName(%q) error = %v, wantErr %v", tt.mcpName, err, tt.wantErr)
			}
		})
	}
}
