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
	"encoding/json"
	"os"
	"path/filepath"
	"time"

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

// DocDebt tracks documentation debt for CLI commands.
// Stored in ~/.orch/doc-debt.json.
type DocDebt struct {
	// Commands maps command file name to its debt entry.
	Commands map[string]DocDebtEntry `json:"commands"`
	// LastUpdated is when the doc debt file was last modified.
	LastUpdated string `json:"last_updated"`
}

// DocDebtEntry represents a single CLI command's documentation status.
type DocDebtEntry struct {
	// CommandFile is the file name (e.g., "reconcile.go").
	CommandFile string `json:"command_file"`
	// DateAdded is when the command was first detected (YYYY-MM-DD).
	DateAdded string `json:"date_added"`
	// Documented indicates if the command has been documented.
	Documented bool `json:"documented"`
	// DateDocumented is when the command was marked as documented (YYYY-MM-DD).
	DateDocumented string `json:"date_documented,omitempty"`
	// DocLocations lists where documentation should exist.
	DocLocations []string `json:"doc_locations,omitempty"`
}

// DocDebtPath returns the path to the doc debt file.
func DocDebtPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orch", "doc-debt.json")
}

// LoadDocDebt loads the doc debt file from ~/.orch/doc-debt.json.
// Returns an empty DocDebt if the file doesn't exist.
func LoadDocDebt() (*DocDebt, error) {
	data, err := os.ReadFile(DocDebtPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &DocDebt{
				Commands: make(map[string]DocDebtEntry),
			}, nil
		}
		return nil, err
	}

	var debt DocDebt
	if err := json.Unmarshal(data, &debt); err != nil {
		return nil, err
	}

	if debt.Commands == nil {
		debt.Commands = make(map[string]DocDebtEntry)
	}

	return &debt, nil
}

// SaveDocDebt saves the doc debt to ~/.orch/doc-debt.json.
func SaveDocDebt(debt *DocDebt) error {
	// Update timestamp
	debt.LastUpdated = time.Now().Format("2006-01-02T15:04:05")

	data, err := json.MarshalIndent(debt, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(DocDebtPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(DocDebtPath(), data, 0644)
}

// AddCommand adds a new command to the doc debt tracker.
// Returns true if the command was newly added, false if it already exists.
func (d *DocDebt) AddCommand(fileName string) bool {
	if _, exists := d.Commands[fileName]; exists {
		return false
	}

	d.Commands[fileName] = DocDebtEntry{
		CommandFile: fileName,
		DateAdded:   time.Now().Format("2006-01-02"),
		Documented:  false,
		DocLocations: []string{
			"~/.claude/skills/meta/orchestrator/SKILL.md",
			"docs/orch-commands-reference.md",
		},
	}
	return true
}

// MarkDocumented marks a command as documented.
func (d *DocDebt) MarkDocumented(fileName string) bool {
	entry, exists := d.Commands[fileName]
	if !exists {
		return false
	}

	entry.Documented = true
	entry.DateDocumented = time.Now().Format("2006-01-02")
	d.Commands[fileName] = entry
	return true
}

// UndocumentedCommands returns all commands that are not yet documented.
func (d *DocDebt) UndocumentedCommands() []DocDebtEntry {
	var result []DocDebtEntry
	for _, entry := range d.Commands {
		if !entry.Documented {
			result = append(result, entry)
		}
	}
	return result
}
