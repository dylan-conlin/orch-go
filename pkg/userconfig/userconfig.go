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

	// ReflectOpen controls whether the daemon creates issues for open investigation actions.
	// Defaults to false if not specified.
	ReflectOpen *bool `yaml:"reflect_open,omitempty"`

	// VerificationPauseThreshold is the max unverified completions before daemon pauses.
	// Defaults to 5 if not specified. Set to 0 to disable verification pause.
	VerificationPauseThreshold *int `yaml:"verification_pause_threshold,omitempty"`

	// MaxSpawnsPerHour is the maximum number of spawns per hour (0 = no limit).
	// Defaults to 20 if not specified.
	MaxSpawnsPerHour *int `yaml:"max_spawns_per_hour,omitempty"`

	// SpawnDelaySeconds is the delay between spawns in seconds.
	// Defaults to 3 if not specified.
	SpawnDelaySeconds *int `yaml:"spawn_delay_seconds,omitempty"`

	// ReflectModelDriftEnabled controls whether model drift reflection is enabled.
	// Defaults to true if not specified.
	ReflectModelDriftEnabled *bool `yaml:"reflect_model_drift_enabled,omitempty"`

	// ReflectModelDriftIntervalHours is how often to run model drift reflection (in hours).
	// Defaults to 4 if not specified.
	ReflectModelDriftIntervalHours *int `yaml:"reflect_model_drift_interval_hours,omitempty"`

	// CleanupEnabled controls whether periodic session cleanup is enabled.
	// Defaults to true if not specified.
	CleanupEnabled *bool `yaml:"cleanup_enabled,omitempty"`

	// CleanupIntervalHours is how often to run cleanup (in hours).
	// Defaults to 6 if not specified.
	CleanupIntervalHours *int `yaml:"cleanup_interval_hours,omitempty"`

	// CleanupAgeDays is the age threshold in days for session cleanup.
	// Defaults to 7 if not specified.
	CleanupAgeDays *int `yaml:"cleanup_age_days,omitempty"`

	// CleanupPreserveOrchestrator if true, skips orchestrator sessions during cleanup.
	// Defaults to true if not specified.
	CleanupPreserveOrchestrator *bool `yaml:"cleanup_preserve_orchestrator,omitempty"`

	// CleanupServerURL is the OpenCode server URL for cleanup operations.
	// Defaults to "http://127.0.0.1:4096" if not specified.
	CleanupServerURL string `yaml:"cleanup_server_url,omitempty"`

	// CleanupArchivedTTLDays is the TTL in days for archived workspace expiry.
	// Defaults to 30 if not specified.
	CleanupArchivedTTLDays *int `yaml:"cleanup_archived_ttl_days,omitempty"`

	// RecoveryEnabled controls whether stuck agent recovery is enabled.
	// Defaults to true if not specified.
	RecoveryEnabled *bool `yaml:"recovery_enabled,omitempty"`

	// RecoveryIntervalMinutes is how often to check for stuck agents (in minutes).
	// Defaults to 5 if not specified.
	RecoveryIntervalMinutes *int `yaml:"recovery_interval_minutes,omitempty"`

	// RecoveryIdleThresholdMinutes is how long an agent must be idle before recovery (in minutes).
	// Defaults to 10 if not specified.
	RecoveryIdleThresholdMinutes *int `yaml:"recovery_idle_threshold_minutes,omitempty"`

	// RecoveryRateLimitMinutes is minimum time between resume attempts per agent (in minutes).
	// Defaults to 60 if not specified.
	RecoveryRateLimitMinutes *int `yaml:"recovery_rate_limit_minutes,omitempty"`

	// KnowledgeHealthEnabled controls whether periodic knowledge health checks are enabled.
	// Defaults to true if not specified.
	KnowledgeHealthEnabled *bool `yaml:"knowledge_health_enabled,omitempty"`

	// KnowledgeHealthIntervalHours is how often to run knowledge health checks (in hours).
	// Defaults to 2 if not specified.
	KnowledgeHealthIntervalHours *int `yaml:"knowledge_health_interval_hours,omitempty"`

	// KnowledgeHealthThreshold is the number of active quick entries that triggers a maintenance issue.
	// Defaults to 50 if not specified.
	KnowledgeHealthThreshold *int `yaml:"knowledge_health_threshold,omitempty"`

	// OrphanDetectionEnabled controls whether periodic orphan detection is enabled.
	// Defaults to true if not specified.
	OrphanDetectionEnabled *bool `yaml:"orphan_detection_enabled,omitempty"`

	// OrphanDetectionIntervalMinutes is how often to check for orphaned issues (in minutes).
	// Defaults to 30 if not specified.
	OrphanDetectionIntervalMinutes *int `yaml:"orphan_detection_interval_minutes,omitempty"`

	// OrphanAgeThresholdMinutes is how long an issue must be in_progress with no agent
	// before it's considered orphaned (in minutes). Defaults to 60 if not specified.
	OrphanAgeThresholdMinutes *int `yaml:"orphan_age_threshold_minutes,omitempty"`

	// PhaseTimeoutEnabled controls whether periodic phase timeout detection is enabled.
	// Defaults to true if not specified.
	PhaseTimeoutEnabled *bool `yaml:"phase_timeout_enabled,omitempty"`

	// PhaseTimeoutIntervalMinutes is how often to check for unresponsive agents (in minutes).
	// Defaults to 5 if not specified.
	PhaseTimeoutIntervalMinutes *int `yaml:"phase_timeout_interval_minutes,omitempty"`

	// PhaseTimeoutThresholdMinutes is how long an agent can go without a phase update
	// before being flagged (in minutes). Defaults to 30 if not specified.
	PhaseTimeoutThresholdMinutes *int `yaml:"phase_timeout_threshold_minutes,omitempty"`

	// AgreementCheckEnabled controls whether periodic agreement checking is enabled.
	// Defaults to true if not specified.
	AgreementCheckEnabled *bool `yaml:"agreement_check_enabled,omitempty"`

	// AgreementCheckIntervalMinutes is how often to run agreement checks (in minutes).
	// Defaults to 30 if not specified.
	AgreementCheckIntervalMinutes *int `yaml:"agreement_check_interval_minutes,omitempty"`

	// InvariantCheckEnabled controls whether daemon self-check invariants run each poll cycle.
	// Defaults to true if not specified.
	InvariantCheckEnabled *bool `yaml:"invariant_check_enabled,omitempty"`

	// InvariantViolationThreshold is the number of consecutive poll cycles with invariant
	// violations before the daemon pauses. Defaults to 3 if not specified.
	InvariantViolationThreshold *int `yaml:"invariant_violation_threshold,omitempty"`

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

// DaemonPollInterval returns the daemon poll interval in seconds.
// Defaults to 60 seconds if not configured.
func (c *Config) DaemonPollInterval() int {
	if c.Daemon.PollInterval == nil {
		return 60 // Default to 60 seconds
	}
	return *c.Daemon.PollInterval
}

// DaemonMaxAgents returns the maximum number of concurrent agents.
// Defaults to 5 if not configured.
func (c *Config) DaemonMaxAgents() int {
	if c.Daemon.MaxAgents == nil {
		return 5 // Default to 5 agents
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

// DaemonVerificationPauseThreshold returns the max unverified completions before daemon pauses.
// Defaults to 5 if not configured. Returns 0 to disable.
func (c *Config) DaemonVerificationPauseThreshold() int {
	if c.Daemon.VerificationPauseThreshold == nil {
		return 5
	}
	return *c.Daemon.VerificationPauseThreshold
}

// DaemonReflectIssues returns whether to create issues from kb reflect findings.
// Defaults to false if not configured.
func (c *Config) DaemonReflectIssues() bool {
	if c.Daemon.ReflectIssues == nil {
		return false // Default to false (the flag that caused the bug!)
	}
	return *c.Daemon.ReflectIssues
}

// DaemonReflectOpen returns whether to create issues from kb reflect open findings.
// Defaults to false if not configured.
func (c *Config) DaemonReflectOpen() bool {
	if c.Daemon.ReflectOpen == nil {
		return false
	}
	return *c.Daemon.ReflectOpen
}

// DaemonMaxSpawnsPerHour returns the maximum spawns per hour.
// Defaults to 20 if not configured.
func (c *Config) DaemonMaxSpawnsPerHour() int {
	if c.Daemon.MaxSpawnsPerHour == nil {
		return 20
	}
	return *c.Daemon.MaxSpawnsPerHour
}

// DaemonSpawnDelaySeconds returns the delay between spawns in seconds.
// Defaults to 3 if not configured.
func (c *Config) DaemonSpawnDelaySeconds() int {
	if c.Daemon.SpawnDelaySeconds == nil {
		return 3
	}
	return *c.Daemon.SpawnDelaySeconds
}

// DaemonReflectModelDriftEnabled returns whether model drift reflection is enabled.
// Defaults to true if not configured.
func (c *Config) DaemonReflectModelDriftEnabled() bool {
	if c.Daemon.ReflectModelDriftEnabled == nil {
		return true
	}
	return *c.Daemon.ReflectModelDriftEnabled
}

// DaemonReflectModelDriftIntervalHours returns the model drift reflection interval in hours.
// Defaults to 4 if not configured.
func (c *Config) DaemonReflectModelDriftIntervalHours() int {
	if c.Daemon.ReflectModelDriftIntervalHours == nil {
		return 4
	}
	return *c.Daemon.ReflectModelDriftIntervalHours
}

// DaemonCleanupEnabled returns whether periodic session cleanup is enabled.
// Defaults to true if not configured.
func (c *Config) DaemonCleanupEnabled() bool {
	if c.Daemon.CleanupEnabled == nil {
		return true
	}
	return *c.Daemon.CleanupEnabled
}

// DaemonCleanupIntervalHours returns the cleanup interval in hours.
// Defaults to 6 if not configured.
func (c *Config) DaemonCleanupIntervalHours() int {
	if c.Daemon.CleanupIntervalHours == nil {
		return 6
	}
	return *c.Daemon.CleanupIntervalHours
}

// DaemonCleanupAgeDays returns the age threshold in days for session cleanup.
// Defaults to 7 if not configured.
func (c *Config) DaemonCleanupAgeDays() int {
	if c.Daemon.CleanupAgeDays == nil {
		return 7
	}
	return *c.Daemon.CleanupAgeDays
}

// DaemonCleanupPreserveOrchestrator returns whether to skip orchestrator sessions during cleanup.
// Defaults to true if not configured.
func (c *Config) DaemonCleanupPreserveOrchestrator() bool {
	if c.Daemon.CleanupPreserveOrchestrator == nil {
		return true
	}
	return *c.Daemon.CleanupPreserveOrchestrator
}

// DaemonCleanupServerURL returns the OpenCode server URL for cleanup.
// Defaults to "http://127.0.0.1:4096" if not configured.
func (c *Config) DaemonCleanupServerURL() string {
	if c.Daemon.CleanupServerURL == "" {
		return "http://127.0.0.1:4096"
	}
	return c.Daemon.CleanupServerURL
}

// DaemonCleanupArchivedTTLDays returns the TTL in days for archived workspace expiry.
// Defaults to 30 if not configured.
func (c *Config) DaemonCleanupArchivedTTLDays() int {
	if c.Daemon.CleanupArchivedTTLDays == nil {
		return 30
	}
	return *c.Daemon.CleanupArchivedTTLDays
}

// DaemonRecoveryEnabled returns whether stuck agent recovery is enabled.
// Defaults to true if not configured.
func (c *Config) DaemonRecoveryEnabled() bool {
	if c.Daemon.RecoveryEnabled == nil {
		return true
	}
	return *c.Daemon.RecoveryEnabled
}

// DaemonRecoveryIntervalMinutes returns how often to check for stuck agents.
// Defaults to 5 if not configured.
func (c *Config) DaemonRecoveryIntervalMinutes() int {
	if c.Daemon.RecoveryIntervalMinutes == nil {
		return 5
	}
	return *c.Daemon.RecoveryIntervalMinutes
}

// DaemonRecoveryIdleThresholdMinutes returns how long an agent must be idle before recovery.
// Defaults to 10 if not configured.
func (c *Config) DaemonRecoveryIdleThresholdMinutes() int {
	if c.Daemon.RecoveryIdleThresholdMinutes == nil {
		return 10
	}
	return *c.Daemon.RecoveryIdleThresholdMinutes
}

// DaemonRecoveryRateLimitMinutes returns the minimum time between resume attempts per agent.
// Defaults to 60 if not configured.
func (c *Config) DaemonRecoveryRateLimitMinutes() int {
	if c.Daemon.RecoveryRateLimitMinutes == nil {
		return 60
	}
	return *c.Daemon.RecoveryRateLimitMinutes
}

// DaemonKnowledgeHealthEnabled returns whether periodic knowledge health checks are enabled.
// Defaults to true if not configured.
func (c *Config) DaemonKnowledgeHealthEnabled() bool {
	if c.Daemon.KnowledgeHealthEnabled == nil {
		return true
	}
	return *c.Daemon.KnowledgeHealthEnabled
}

// DaemonKnowledgeHealthIntervalHours returns how often to run knowledge health checks.
// Defaults to 2 if not configured.
func (c *Config) DaemonKnowledgeHealthIntervalHours() int {
	if c.Daemon.KnowledgeHealthIntervalHours == nil {
		return 2
	}
	return *c.Daemon.KnowledgeHealthIntervalHours
}

// DaemonKnowledgeHealthThreshold returns the number of active quick entries that triggers maintenance.
// Defaults to 50 if not configured.
func (c *Config) DaemonKnowledgeHealthThreshold() int {
	if c.Daemon.KnowledgeHealthThreshold == nil {
		return 50
	}
	return *c.Daemon.KnowledgeHealthThreshold
}

// DaemonOrphanDetectionEnabled returns whether periodic orphan detection is enabled.
// Defaults to true if not configured.
func (c *Config) DaemonOrphanDetectionEnabled() bool {
	if c.Daemon.OrphanDetectionEnabled == nil {
		return true
	}
	return *c.Daemon.OrphanDetectionEnabled
}

// DaemonOrphanDetectionIntervalMinutes returns how often to check for orphaned issues.
// Defaults to 30 if not configured.
func (c *Config) DaemonOrphanDetectionIntervalMinutes() int {
	if c.Daemon.OrphanDetectionIntervalMinutes == nil {
		return 30
	}
	return *c.Daemon.OrphanDetectionIntervalMinutes
}

// DaemonOrphanAgeThresholdMinutes returns how long before an issue is considered orphaned.
// Defaults to 60 if not configured.
func (c *Config) DaemonOrphanAgeThresholdMinutes() int {
	if c.Daemon.OrphanAgeThresholdMinutes == nil {
		return 60
	}
	return *c.Daemon.OrphanAgeThresholdMinutes
}

// DaemonPhaseTimeoutEnabled returns whether periodic phase timeout detection is enabled.
// Defaults to true if not configured.
func (c *Config) DaemonPhaseTimeoutEnabled() bool {
	if c.Daemon.PhaseTimeoutEnabled == nil {
		return true
	}
	return *c.Daemon.PhaseTimeoutEnabled
}

// DaemonPhaseTimeoutIntervalMinutes returns how often to check for unresponsive agents.
// Defaults to 5 if not configured.
func (c *Config) DaemonPhaseTimeoutIntervalMinutes() int {
	if c.Daemon.PhaseTimeoutIntervalMinutes == nil {
		return 5
	}
	return *c.Daemon.PhaseTimeoutIntervalMinutes
}

// DaemonPhaseTimeoutThresholdMinutes returns how long before an agent is flagged as unresponsive.
// Defaults to 30 if not configured.
func (c *Config) DaemonPhaseTimeoutThresholdMinutes() int {
	if c.Daemon.PhaseTimeoutThresholdMinutes == nil {
		return 30
	}
	return *c.Daemon.PhaseTimeoutThresholdMinutes
}

// DaemonAgreementCheckEnabled returns whether periodic agreement checking is enabled.
// Defaults to true if not configured.
func (c *Config) DaemonAgreementCheckEnabled() bool {
	if c.Daemon.AgreementCheckEnabled == nil {
		return true
	}
	return *c.Daemon.AgreementCheckEnabled
}

// DaemonAgreementCheckIntervalMinutes returns how often to run agreement checks.
// Defaults to 30 if not configured.
func (c *Config) DaemonAgreementCheckIntervalMinutes() int {
	if c.Daemon.AgreementCheckIntervalMinutes == nil {
		return 30
	}
	return *c.Daemon.AgreementCheckIntervalMinutes
}

// DaemonInvariantCheckEnabled returns whether daemon self-check invariants are enabled.
// Defaults to true if not configured.
func (c *Config) DaemonInvariantCheckEnabled() bool {
	if c.Daemon.InvariantCheckEnabled == nil {
		return true
	}
	return *c.Daemon.InvariantCheckEnabled
}

// DaemonInvariantViolationThreshold returns the consecutive violation cycles before daemon pauses.
// Defaults to 3 if not configured.
func (c *Config) DaemonInvariantViolationThreshold() int {
	if c.Daemon.InvariantViolationThreshold == nil {
		return 3
	}
	return *c.Daemon.InvariantViolationThreshold
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
