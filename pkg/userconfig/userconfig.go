// Package userconfig provides user-level configuration management for orch-go.
//
// Configuration file: ~/.orch/config.yaml
//
// Example config:
//
//	backend: opencode
//	auto_export_transcript: true
//	notifications:
//	  enabled: true
package userconfig

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// NotificationConfig holds notification-related settings.
type NotificationConfig struct {
	// Enabled controls whether desktop notifications are sent.
	// Defaults to true if not specified.
	Enabled *bool `yaml:"enabled,omitempty"`
}

// DaemonConfig holds daemon-related settings.
type DaemonConfig struct {
	// MaxAgents is the maximum number of concurrent agents (0 = no limit).
	// Defaults to 3 if not specified.
	MaxAgents *int `yaml:"max_agents,omitempty"`
	// MaxSpawnsPerHour is the maximum number of spawns allowed per hour (0 = no limit).
	// This prevents runaway spawning when many issues are batch-labeled as triage:ready.
	// Defaults to 20 if not specified.
	MaxSpawnsPerHour *int `yaml:"max_spawns_per_hour,omitempty"`
}

// Config represents the user-level orch configuration.
type Config struct {
	// Backend specifies the orchestration backend (e.g., "opencode").
	Backend string `yaml:"backend,omitempty"`
	// AutoExportTranscript enables automatic transcript export.
	AutoExportTranscript bool `yaml:"auto_export_transcript,omitempty"`
	// Notifications holds notification-related settings.
	Notifications NotificationConfig `yaml:"notifications,omitempty"`
	// Daemon holds daemon-related settings.
	Daemon DaemonConfig `yaml:"daemon,omitempty"`
}

// ConfigPath returns the path to the user config file.
func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orch", "config.yaml")
}

// Load loads the user configuration from ~/.orch/config.yaml.
// Returns a default config if the file doesn't exist.
func Load() (*Config, error) {
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return DefaultConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *Config {
	enabled := true
	return &Config{
		Backend: "opencode",
		Notifications: NotificationConfig{
			Enabled: &enabled,
		},
	}
}

// Save saves the user configuration to ~/.orch/config.yaml.
func Save(cfg *Config) error {
	// Ensure directory exists
	dir := filepath.Dir(ConfigPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigPath(), data, 0644)
}

// NotificationsEnabled returns whether desktop notifications are enabled.
// Defaults to true if not configured.
func (c *Config) NotificationsEnabled() bool {
	if c.Notifications.Enabled == nil {
		return true // Default to enabled
	}
	return *c.Notifications.Enabled
}

// DaemonMaxAgents returns the max concurrent agents for the daemon.
// Defaults to 3 if not configured.
func (c *Config) DaemonMaxAgents() int {
	if c.Daemon.MaxAgents == nil {
		return 3 // Default
	}
	return *c.Daemon.MaxAgents
}

// DaemonMaxSpawnsPerHour returns the max spawns per hour for the daemon.
// Defaults to 20 if not configured.
func (c *Config) DaemonMaxSpawnsPerHour() int {
	if c.Daemon.MaxSpawnsPerHour == nil {
		return 20 // Default
	}
	return *c.Daemon.MaxSpawnsPerHour
}
