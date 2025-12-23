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
	Servers map[string]int `yaml:"servers"`
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

	// Initialize servers map if nil
	if cfg.Servers == nil {
		cfg.Servers = make(map[string]int)
	}

	return &cfg, nil
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

// GetServerPort returns the port for a service, or 0 and false if not found.
func (c *Config) GetServerPort(service string) (int, bool) {
	if c.Servers == nil {
		return 0, false
	}
	port, ok := c.Servers[service]
	return port, ok
}
