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

// ReflectConfig holds settings for periodic kb reflect analysis.
type ReflectConfig struct {
	// Enabled controls whether periodic reflection is enabled in the daemon.
	// Defaults to true if not specified.
	Enabled *bool `yaml:"enabled,omitempty"`

	// IntervalMinutes is how often to run reflection analysis (in minutes).
	// Defaults to 60 (hourly) if not specified.
	IntervalMinutes *int `yaml:"interval_minutes,omitempty"`

	// CreateIssues controls whether to automatically create beads issues
	// for synthesis opportunities (topics with 10+ investigations).
	// Defaults to true if not specified.
	CreateIssues *bool `yaml:"create_issues,omitempty"`
}

// Config represents the user-level orch configuration.
type Config struct {
	// Backend specifies the orchestration backend (e.g., "opencode").
	Backend string `yaml:"backend,omitempty"`
	// AutoExportTranscript enables automatic transcript export.
	AutoExportTranscript bool `yaml:"auto_export_transcript,omitempty"`
	// Notifications holds notification-related settings.
	Notifications NotificationConfig `yaml:"notifications,omitempty"`
	// Reflect holds settings for periodic kb reflect analysis.
	Reflect ReflectConfig `yaml:"reflect,omitempty"`
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

// ReflectEnabled returns whether periodic reflection is enabled in the daemon.
// Defaults to true if not configured.
func (c *Config) ReflectEnabled() bool {
	if c.Reflect.Enabled == nil {
		return true // Default to enabled
	}
	return *c.Reflect.Enabled
}

// ReflectIntervalMinutes returns how often to run reflection analysis.
// Defaults to 60 minutes (hourly) if not configured.
func (c *Config) ReflectIntervalMinutes() int {
	if c.Reflect.IntervalMinutes == nil {
		return 60 // Default to hourly
	}
	return *c.Reflect.IntervalMinutes
}

// ReflectCreateIssues returns whether to create beads issues for synthesis opportunities.
// Defaults to true if not configured.
func (c *Config) ReflectCreateIssues() bool {
	if c.Reflect.CreateIssues == nil {
		return true // Default to creating issues
	}
	return *c.Reflect.CreateIssues
}
