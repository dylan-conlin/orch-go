// Package config provides project configuration management for orch-go.
//
// The config file is stored at .orch/config.yaml in the project directory.
//
// Example config:
//
//	servers:
//	  web: 5173
//	  api: 3000
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the project configuration.
type Config struct {
	SpawnMode string            `yaml:"spawn_mode"`         // "claude" | "opencode"
	Models    map[string]string `yaml:"models,omitempty"`   // Model aliases (e.g., "opus": "anthropic/claude-opus-4-6")
	Claude    ClaudeConfig      `yaml:"claude,omitempty"`   // Claude mode settings
	OpenCode  OpenCodeConfig    `yaml:"opencode,omitempty"` // OpenCode mode settings
	Servers   map[string]int    `yaml:"servers,omitempty"`
	Opsec     OpsecConfig       `yaml:"opsec,omitempty"`    // OPSEC sandbox settings
}

// OpsecConfig holds settings for network sandbox enforcement.
type OpsecConfig struct {
	Sandbox        bool     `yaml:"sandbox"`                    // Enable sandbox-exec wrapping for spawned agents
	ProxyPort      int      `yaml:"proxy_port,omitempty"`       // Local proxy port (default 8199)
	BlockedDomains []string `yaml:"blocked_domains,omitempty"`  // Additional blocked domains (merged with global)
}

// ConfigMeta tracks which YAML keys were explicitly set.
type ConfigMeta struct {
	Explicit         map[string]bool
	ExplicitClaude   map[string]bool
	ExplicitOpenCode map[string]bool
}

// ClaudeConfig holds settings for Claude mode spawning.
type ClaudeConfig struct {
	Model       string `yaml:"model"`        // "opus" | "sonnet" | "haiku"
	TmuxSession string `yaml:"tmux_session"` // tmux session name
}

// OpenCodeConfig holds settings for OpenCode mode spawning.
type OpenCodeConfig struct {
	Model  string `yaml:"model"`  // default model for spawns
	Server string `yaml:"server"` // HTTP server URL
}

// DefaultPath returns the default config file path for a project directory.
func DefaultPath(projectDir string) string {
	return filepath.Join(projectDir, ".orch", "config.yaml")
}

// Load loads the project configuration from .orch/config.yaml.
func Load(projectDir string) (*Config, error) {
	configPath := DefaultPath(projectDir)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults for backward compatibility
	cfg.ApplyDefaults()

	return &cfg, nil
}

// LoadWithMeta loads the project configuration and tracks explicit YAML keys.
func LoadWithMeta(projectDir string) (*Config, *ConfigMeta, error) {
	configPath := DefaultPath(projectDir)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, nil, fmt.Errorf("failed to parse config metadata: %w", err)
	}

	meta := &ConfigMeta{
		Explicit:         explicitKeys(raw),
		ExplicitClaude:   explicitKeys(raw["claude"]),
		ExplicitOpenCode: explicitKeys(raw["opencode"]),
	}

	// Apply defaults for backward compatibility
	cfg.ApplyDefaults()

	return &cfg, meta, nil
}

// Save saves the project configuration to .orch/config.yaml.
func Save(projectDir string, cfg *Config) error {
	configPath := DefaultPath(projectDir)

	// Ensure .orch directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create .orch directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ApplyDefaults sets default values for unspecified config fields.
func (c *Config) ApplyDefaults() {
	// Default spawn mode to opencode for backward compatibility
	if c.SpawnMode == "" {
		c.SpawnMode = "opencode"
	}

	// Default Claude settings
	if c.Claude.Model == "" {
		c.Claude.Model = "opus"
	}
	if c.Claude.TmuxSession == "" {
		c.Claude.TmuxSession = "workers-orch-go"
	}

	// Default OpenCode settings
	if c.OpenCode.Model == "" {
		c.OpenCode.Model = "flash"
	}
	if c.OpenCode.Server == "" {
		c.OpenCode.Server = "http://127.0.0.1:4096"
	}

	// Initialize servers map if nil
	if c.Servers == nil {
		c.Servers = make(map[string]int)
	}
}

// GetServerPort returns the port for a service, or 0 and false if not found.
func (c *Config) GetServerPort(service string) (int, bool) {
	if c.Servers == nil {
		return 0, false
	}
	port, ok := c.Servers[service]
	return port, ok
}

func explicitKeys(value any) map[string]bool {
	keys := map[string]bool{}

	switch typed := value.(type) {
	case map[string]any:
		for key := range typed {
			keys[key] = true
		}
	case map[interface{}]interface{}:
		for key := range typed {
			if keyName, ok := key.(string); ok {
				keys[keyName] = true
			}
		}
	}

	return keys
}
