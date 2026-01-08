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
//	default_tier: full  # Force all spawns to produce SYNTHESIS.md (or "light" for skill defaults)
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

// DaemonConfig holds settings for the orch daemon plist generation.
// This is the declarative source of truth for ~/Library/LaunchAgents/com.orch.daemon.plist.
type DaemonConfig struct {
	// PollInterval is how often the daemon polls for ready issues (in seconds).
	// Defaults to 60 if not specified.
	PollInterval *int `yaml:"poll_interval,omitempty"`

	// MaxAgents is the maximum number of concurrent agents the daemon will spawn.
	// Defaults to 3 if not specified.
	MaxAgents *int `yaml:"max_agents,omitempty"`

	// Label is the beads label to filter for when finding ready issues.
	// Defaults to "triage:ready" if not specified.
	Label string `yaml:"label,omitempty"`

	// Verbose enables verbose logging in the daemon.
	// Defaults to true if not specified.
	Verbose *bool `yaml:"verbose,omitempty"`

	// ReflectIssues controls whether the daemon creates issues from kb reflect findings.
	// Defaults to false if not specified.
	ReflectIssues *bool `yaml:"reflect_issues,omitempty"`

	// WorkingDirectory is the directory the daemon runs from.
	// Defaults to ~/Documents/personal/orch-go if not specified.
	WorkingDirectory string `yaml:"working_directory,omitempty"`

	// Path is a list of directories to add to the daemon's PATH environment variable.
	// These are prepended to the system PATH.
	Path []string `yaml:"path,omitempty"`
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
	// DefaultTier specifies the default spawn tier: "light" or "full".
	// When set to "full", all spawns (including light-tier skills) will require SYNTHESIS.md.
	// When set to "light" or empty, skill defaults are used.
	// Explicit --light or --full flags still override this setting.
	DefaultTier string `yaml:"default_tier,omitempty"`
	// Daemon holds settings for the orch daemon plist generation.
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

// GetDefaultTier returns the default spawn tier from config.
// Returns "full" if configured as "full", empty string otherwise (use skill defaults).
// Valid values are "light", "full", or empty (skill defaults).
func (c *Config) GetDefaultTier() string {
	// Only return "full" if explicitly configured - this forces all spawns to full tier
	// "light" or empty means use skill defaults
	if c.DefaultTier == "full" {
		return "full"
	}
	return "" // Use skill defaults
}

// DaemonPollInterval returns the daemon poll interval in seconds.
// Defaults to 60 seconds if not configured.
func (c *Config) DaemonPollInterval() int {
	if c.Daemon.PollInterval == nil {
		return 60 // Default to 60 seconds
	}
	return *c.Daemon.PollInterval
}

// DaemonMaxAgents returns the maximum number of concurrent agents.
// Defaults to 3 if not configured.
func (c *Config) DaemonMaxAgents() int {
	if c.Daemon.MaxAgents == nil {
		return 3 // Default to 3 agents
	}
	return *c.Daemon.MaxAgents
}

// DaemonLabel returns the beads label filter for ready issues.
// Defaults to "triage:ready" if not configured.
func (c *Config) DaemonLabel() string {
	if c.Daemon.Label == "" {
		return "triage:ready" // Default label
	}
	return c.Daemon.Label
}

// DaemonVerbose returns whether verbose logging is enabled.
// Defaults to true if not configured.
func (c *Config) DaemonVerbose() bool {
	if c.Daemon.Verbose == nil {
		return true // Default to verbose
	}
	return *c.Daemon.Verbose
}

// DaemonReflectIssues returns whether to create issues from kb reflect findings.
// Defaults to false if not configured.
func (c *Config) DaemonReflectIssues() bool {
	if c.Daemon.ReflectIssues == nil {
		return false // Default to false (the flag that caused the bug!)
	}
	return *c.Daemon.ReflectIssues
}

// DaemonWorkingDirectory returns the daemon's working directory.
// Defaults to ~/Documents/personal/orch-go if not configured.
func (c *Config) DaemonWorkingDirectory() string {
	if c.Daemon.WorkingDirectory == "" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Documents", "personal", "orch-go")
	}
	// Expand ~ in path
	if len(c.Daemon.WorkingDirectory) > 0 && c.Daemon.WorkingDirectory[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, c.Daemon.WorkingDirectory[1:])
	}
	return c.Daemon.WorkingDirectory
}

// DaemonPath returns the PATH directories to add to the daemon environment.
// Defaults to common orch tool locations if not configured.
func (c *Config) DaemonPath() []string {
	if len(c.Daemon.Path) == 0 {
		home, _ := os.UserHomeDir()
		return []string{
			filepath.Join(home, ".bun", "bin"),
			filepath.Join(home, "bin"),
			filepath.Join(home, "go", "bin"),
			"/opt/homebrew/bin",
			filepath.Join(home, ".local", "bin"),
		}
	}
	// Expand ~ in each path
	result := make([]string, len(c.Daemon.Path))
	for i, p := range c.Daemon.Path {
		if len(p) > 0 && p[0] == '~' {
			home, _ := os.UserHomeDir()
			result[i] = filepath.Join(home, p[1:])
		} else {
			result[i] = p
		}
	}
	return result
}
