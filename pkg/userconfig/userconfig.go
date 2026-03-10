// Package userconfig provides user-level configuration management for orch-go.
//
// Configuration file: ~/.orch/config.yaml
//
// Example config:
//
//	backend: opencode
//	allow_anthropic_opencode: true  # Allow Anthropic models via OpenCode backend (unsupported)
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

// SessionConfig holds settings for orchestrator session management.
type SessionConfig struct {
	// OrchestratorCheckpoints holds checkpoint thresholds for orchestrator sessions.
	// Orchestrator sessions coordinate work and don't accumulate implementation context,
	// so they can safely run longer than agent sessions.
	OrchestratorCheckpoints *CheckpointThresholds `yaml:"orchestrator_checkpoints,omitempty"`

	// AgentCheckpoints holds checkpoint thresholds for agent sessions.
	// Agents accumulate implementation context which degrades over time,
	// so they need shorter checkpoint thresholds.
	AgentCheckpoints *CheckpointThresholds `yaml:"agent_checkpoints,omitempty"`
}

// CheckpointThresholds defines the checkpoint duration thresholds for sessions.
type CheckpointThresholds struct {
	// WarningMinutes is when to start suggesting checkpoints.
	WarningMinutes *int `yaml:"warning_minutes,omitempty"`

	// StrongMinutes is when to strongly recommend handoff.
	StrongMinutes *int `yaml:"strong_minutes,omitempty"`

	// MaxMinutes is the maximum recommended session duration.
	MaxMinutes *int `yaml:"max_minutes,omitempty"`
}

// Config represents the user-level orch configuration.
type Config struct {
	// Backend specifies the orchestration backend (e.g., "opencode").
	Backend string `yaml:"backend,omitempty"`
	// AllowAnthropicOpenCode allows Anthropic models to run on the OpenCode backend.
	// Defaults to false; only set to true to override the compatibility guard.
	AllowAnthropicOpenCode bool `yaml:"allow_anthropic_opencode,omitempty"`
	// Models holds custom model aliases (e.g., "opus": "anthropic/claude-opus-4-6").
	// These override built-in aliases in pkg/model/model.go.
	Models map[string]string `yaml:"models,omitempty"`
	// AutoExportTranscript enables automatic transcript export.
	AutoExportTranscript bool `yaml:"auto_export_transcript,omitempty"`
	// Notifications holds notification-related settings.
	Notifications NotificationConfig `yaml:"notifications,omitempty"`
	// Reflect holds settings for periodic kb reflect analysis.
	Reflect ReflectConfig `yaml:"reflect,omitempty"`
	// DefaultModel specifies the default model alias for worker spawns.
	// When set, spawns without an explicit --model flag will use this model.
	// Accepts aliases (e.g., "gpt4o", "opus", "sonnet") or provider/model format.
	// When empty, uses the hardcoded DefaultModel in pkg/model.
	DefaultModel string `yaml:"default_model,omitempty"`
	// DefaultTier specifies the default spawn tier: "light" or "full".
	// When set to "full", all spawns (including light-tier skills) will require SYNTHESIS.md.
	// When set to "light" or empty, skill defaults are used.
	// Explicit --light or --full flags still override this setting.
	DefaultTier string `yaml:"default_tier,omitempty"`
	// Daemon holds settings for the orch daemon plist generation.
	Daemon DaemonConfig `yaml:"daemon,omitempty"`
	// Session holds settings for orchestrator session management.
	Session SessionConfig `yaml:"session,omitempty"`
}

// ConfigMeta tracks which YAML keys were explicitly set.
type ConfigMeta struct {
	Explicit                            map[string]bool
	ExplicitNotifications               map[string]bool
	ExplicitReflect                     map[string]bool
	ExplicitDaemon                      map[string]bool
	ExplicitSession                     map[string]bool
	ExplicitSessionOrchestratorCheckpts map[string]bool
	ExplicitSessionAgentCheckpts        map[string]bool
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

// LoadWithMeta loads the user configuration and tracks explicit YAML keys.
func LoadWithMeta() (*Config, *ConfigMeta, error) {
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), &ConfigMeta{
				Explicit:                            map[string]bool{},
				ExplicitNotifications:               map[string]bool{},
				ExplicitReflect:                     map[string]bool{},
				ExplicitDaemon:                      map[string]bool{},
				ExplicitSession:                     map[string]bool{},
				ExplicitSessionOrchestratorCheckpts: map[string]bool{},
				ExplicitSessionAgentCheckpts:        map[string]bool{},
			}, nil
		}
		return nil, nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, nil, err
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, nil, err
	}

	sessionRaw := mapValue(raw["session"])

	meta := &ConfigMeta{
		Explicit:                            explicitKeys(raw),
		ExplicitNotifications:               explicitKeys(raw["notifications"]),
		ExplicitReflect:                     explicitKeys(raw["reflect"]),
		ExplicitDaemon:                      explicitKeys(raw["daemon"]),
		ExplicitSession:                     explicitKeys(raw["session"]),
		ExplicitSessionOrchestratorCheckpts: explicitKeys(sessionRaw["orchestrator_checkpoints"]),
		ExplicitSessionAgentCheckpts:        explicitKeys(sessionRaw["agent_checkpoints"]),
	}

	return &cfg, meta, nil
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

// Default checkpoint thresholds (in minutes).
// Agent sessions use shorter thresholds because implementation context degrades.
// Orchestrator sessions use longer thresholds because coordination context persists better.
const (
	// Agent session defaults (implementation work)
	DefaultAgentWarningMinutes = 120 // 2 hours
	DefaultAgentStrongMinutes  = 180 // 3 hours
	DefaultAgentMaxMinutes     = 240 // 4 hours

	// Orchestrator session defaults (coordination work)
	DefaultOrchestratorWarningMinutes = 240 // 4 hours
	DefaultOrchestratorStrongMinutes  = 360 // 6 hours
	DefaultOrchestratorMaxMinutes     = 480 // 8 hours
)

// OrchestratorCheckpointWarning returns the warning threshold for orchestrator sessions.
// Defaults to 4 hours if not configured.
func (c *Config) OrchestratorCheckpointWarning() int {
	if c.Session.OrchestratorCheckpoints != nil && c.Session.OrchestratorCheckpoints.WarningMinutes != nil {
		return *c.Session.OrchestratorCheckpoints.WarningMinutes
	}
	return DefaultOrchestratorWarningMinutes
}

// OrchestratorCheckpointStrong returns the strong threshold for orchestrator sessions.
// Defaults to 6 hours if not configured.
func (c *Config) OrchestratorCheckpointStrong() int {
	if c.Session.OrchestratorCheckpoints != nil && c.Session.OrchestratorCheckpoints.StrongMinutes != nil {
		return *c.Session.OrchestratorCheckpoints.StrongMinutes
	}
	return DefaultOrchestratorStrongMinutes
}

// OrchestratorCheckpointMax returns the max threshold for orchestrator sessions.
// Defaults to 8 hours if not configured.
func (c *Config) OrchestratorCheckpointMax() int {
	if c.Session.OrchestratorCheckpoints != nil && c.Session.OrchestratorCheckpoints.MaxMinutes != nil {
		return *c.Session.OrchestratorCheckpoints.MaxMinutes
	}
	return DefaultOrchestratorMaxMinutes
}

// AgentCheckpointWarning returns the warning threshold for agent sessions.
// Defaults to 2 hours if not configured.
func (c *Config) AgentCheckpointWarning() int {
	if c.Session.AgentCheckpoints != nil && c.Session.AgentCheckpoints.WarningMinutes != nil {
		return *c.Session.AgentCheckpoints.WarningMinutes
	}
	return DefaultAgentWarningMinutes
}

// AgentCheckpointStrong returns the strong threshold for agent sessions.
// Defaults to 3 hours if not configured.
func (c *Config) AgentCheckpointStrong() int {
	if c.Session.AgentCheckpoints != nil && c.Session.AgentCheckpoints.StrongMinutes != nil {
		return *c.Session.AgentCheckpoints.StrongMinutes
	}
	return DefaultAgentStrongMinutes
}

// AgentCheckpointMax returns the max threshold for agent sessions.
// Defaults to 4 hours if not configured.
func (c *Config) AgentCheckpointMax() int {
	if c.Session.AgentCheckpoints != nil && c.Session.AgentCheckpoints.MaxMinutes != nil {
		return *c.Session.AgentCheckpoints.MaxMinutes
	}
	return DefaultAgentMaxMinutes
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

func mapValue(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case map[interface{}]interface{}:
		result := map[string]any{}
		for key, val := range typed {
			keyName, ok := key.(string)
			if !ok {
				continue
			}
			result[keyName] = val
		}
		return result
	default:
		return nil
	}
}
