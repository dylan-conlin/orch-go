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

// E2E tracked test: pipeline validation Feb 10 2026

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const legacyFlatConfigMigrationGuidePath = "docs/project-config-migration.md"

// Config represents the project configuration.
type Config struct {
	SpawnMode  string           `yaml:"spawn_mode"`         // "claude" | "opencode"
	Domain     string           `yaml:"domain,omitempty"`   // "personal" | "work" - overrides auto-detection
	Claude     ClaudeConfig     `yaml:"claude,omitempty"`   // Claude mode settings
	OpenCode   OpenCodeConfig   `yaml:"opencode,omitempty"` // OpenCode mode settings
	Servers    map[string]int   `yaml:"servers,omitempty"`
	Daemon     DaemonConfig     `yaml:"daemon,omitempty"`
	Dashboard  DashboardConfig  `yaml:"dashboard,omitempty"`
	Spawn      SpawnConfig      `yaml:"spawn,omitempty"`
	Completion CompletionConfig `yaml:"completion,omitempty"`
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

// DaemonConfig holds project-level daemon policy overrides.
type DaemonConfig struct {
	Cleanup           DaemonCleanupConfig           `yaml:"cleanup,omitempty"`
	DeadSession       DaemonDeadSessionConfig       `yaml:"dead_session,omitempty"`
	OrphanReap        DaemonOrphanReapConfig        `yaml:"orphan_reap,omitempty"`
	DashboardWatchdog DaemonDashboardWatchdogConfig `yaml:"dashboard_watchdog,omitempty"`
}

// DaemonCleanupConfig holds daemon cleanup policy values.
type DaemonCleanupConfig struct {
	IntervalMinutes   int `yaml:"interval_minutes,omitempty"`
	SessionsAgeDays   int `yaml:"sessions_age_days,omitempty"`
	WorkspacesAgeDays int `yaml:"workspaces_age_days,omitempty"`
}

// DaemonDeadSessionConfig holds dead-session detection policy values.
type DaemonDeadSessionConfig struct {
	IntervalMinutes int `yaml:"interval_minutes,omitempty"`
	MaxRetries      int `yaml:"max_retries,omitempty"`
}

// DaemonOrphanReapConfig holds orphan reaper policy values.
type DaemonOrphanReapConfig struct {
	IntervalMinutes int `yaml:"interval_minutes,omitempty"`
}

// DaemonDashboardWatchdogConfig holds dashboard watchdog policy values.
type DaemonDashboardWatchdogConfig struct {
	IntervalSeconds        int `yaml:"interval_seconds,omitempty"`
	FailuresBeforeRestart  int `yaml:"failures_before_restart,omitempty"`
	RestartCooldownMinutes int `yaml:"restart_cooldown_minutes,omitempty"`
}

// DashboardConfig holds project dashboard policy overrides.
type DashboardConfig struct {
	Agents DashboardAgentsConfig `yaml:"agents,omitempty"`
}

// DashboardAgentsConfig holds status timing thresholds for /api/agents.
type DashboardAgentsConfig struct {
	ActiveMinutes     int `yaml:"active_minutes,omitempty"`
	GhostDisplayHours int `yaml:"ghost_display_hours,omitempty"`
	DeadMinutes       int `yaml:"dead_minutes,omitempty"`
	StalledMinutes    int `yaml:"stalled_minutes,omitempty"`
	BeadsFetchHours   int `yaml:"beads_fetch_hours,omitempty"`
}

// SpawnConfig holds spawn policy overrides.
type SpawnConfig struct {
	ContextQuality  SpawnContextQualityConfig `yaml:"context_quality,omitempty"`
	AllowAPIBilling bool                      `yaml:"allow_api_billing,omitempty"` // Opt-in for pay-per-token API billing
}

// SpawnContextQualityConfig holds gap gate policy values.
type SpawnContextQualityConfig struct {
	Threshold int `yaml:"threshold,omitempty"`
}

// CompletionConfig holds completion policy overrides.
type CompletionConfig struct {
	AutoRebuild      CompletionAutoRebuildConfig      `yaml:"auto_rebuild,omitempty"`
	TranscriptExport CompletionTranscriptExportConfig `yaml:"transcript_export,omitempty"`
	CacheInvalidate  CompletionCacheInvalidateConfig  `yaml:"cache_invalidate,omitempty"`
}

// CompletionAutoRebuildConfig holds timeout for auto-rebuild.
type CompletionAutoRebuildConfig struct {
	TimeoutSeconds int `yaml:"timeout_seconds,omitempty"`
}

// CompletionTranscriptExportConfig holds timeout for transcript export.
type CompletionTranscriptExportConfig struct {
	TimeoutSeconds int `yaml:"timeout_seconds,omitempty"`
}

// CompletionCacheInvalidateConfig holds timeout for dashboard cache invalidation.
type CompletionCacheInvalidateConfig struct {
	TimeoutSeconds int `yaml:"timeout_seconds,omitempty"`
}

// legacyFlatConfig captures deprecated top-level policy keys that predate
// the typed nested schema.
//
// Deprecated keys are accepted during the migration window and automatically
// rewritten into nested keys in .orch/config.yaml.
type legacyFlatConfig struct {
	DaemonCleanupIntervalMinutes                  *int `yaml:"daemon_cleanup_interval_minutes,omitempty"`
	DaemonCleanupSessionsAgeDays                  *int `yaml:"daemon_cleanup_sessions_age_days,omitempty"`
	DaemonCleanupWorkspacesAgeDays                *int `yaml:"daemon_cleanup_workspaces_age_days,omitempty"`
	DaemonDeadSessionIntervalMinutes              *int `yaml:"daemon_dead_session_interval_minutes,omitempty"`
	DaemonMaxDeadSessionRetries                   *int `yaml:"daemon_max_dead_session_retries,omitempty"`
	DaemonOrphanReapIntervalMinutes               *int `yaml:"daemon_orphan_reap_interval_minutes,omitempty"`
	DaemonDashboardWatchdogIntervalSeconds        *int `yaml:"daemon_dashboard_watchdog_interval_seconds,omitempty"`
	DaemonDashboardWatchdogFailuresBeforeRestart  *int `yaml:"daemon_dashboard_watchdog_failures_before_restart,omitempty"`
	DaemonDashboardWatchdogRestartCooldownMinutes *int `yaml:"daemon_dashboard_watchdog_restart_cooldown_minutes,omitempty"`
	DashboardAgentsActiveMinutes                  *int `yaml:"dashboard_agents_active_minutes,omitempty"`
	DashboardAgentsGhostDisplayHours              *int `yaml:"dashboard_agents_ghost_display_hours,omitempty"`
	DashboardAgentsDeadMinutes                    *int `yaml:"dashboard_agents_dead_minutes,omitempty"`
	DashboardAgentsStalledMinutes                 *int `yaml:"dashboard_agents_stalled_minutes,omitempty"`
	DashboardAgentsBeadsFetchHours                *int `yaml:"dashboard_agents_beads_fetch_hours,omitempty"`
	SpawnContextQualityThreshold                  *int `yaml:"spawn_context_quality_threshold,omitempty"`
	CompletionAutoRebuildTimeoutSeconds           *int `yaml:"completion_auto_rebuild_timeout_seconds,omitempty"`
	CompletionTranscriptExportTimeoutSeconds      *int `yaml:"completion_transcript_export_timeout_seconds,omitempty"`
	CompletionCacheInvalidateTimeoutSeconds       *int `yaml:"completion_cache_invalidate_timeout_seconds,omitempty"`
}

func migrateLegacyInt(target *int, legacy *int, legacyKey string, nestedPath string, notices *[]string) bool {
	if legacy == nil {
		return false
	}

	*notices = append(*notices, fmt.Sprintf("%s -> %s", legacyKey, nestedPath))
	if *target == 0 {
		*target = *legacy
	}

	return true
}

func (c *Config) applyLegacyFlatConfig(legacy legacyFlatConfig) (bool, []string) {
	notices := make([]string, 0)
	legacyFound := false

	legacyFound = migrateLegacyInt(&c.Daemon.Cleanup.IntervalMinutes, legacy.DaemonCleanupIntervalMinutes, "daemon_cleanup_interval_minutes", "daemon.cleanup.interval_minutes", &notices) || legacyFound
	legacyFound = migrateLegacyInt(&c.Daemon.Cleanup.SessionsAgeDays, legacy.DaemonCleanupSessionsAgeDays, "daemon_cleanup_sessions_age_days", "daemon.cleanup.sessions_age_days", &notices) || legacyFound
	legacyFound = migrateLegacyInt(&c.Daemon.Cleanup.WorkspacesAgeDays, legacy.DaemonCleanupWorkspacesAgeDays, "daemon_cleanup_workspaces_age_days", "daemon.cleanup.workspaces_age_days", &notices) || legacyFound
	legacyFound = migrateLegacyInt(&c.Daemon.DeadSession.IntervalMinutes, legacy.DaemonDeadSessionIntervalMinutes, "daemon_dead_session_interval_minutes", "daemon.dead_session.interval_minutes", &notices) || legacyFound
	legacyFound = migrateLegacyInt(&c.Daemon.DeadSession.MaxRetries, legacy.DaemonMaxDeadSessionRetries, "daemon_max_dead_session_retries", "daemon.dead_session.max_retries", &notices) || legacyFound
	legacyFound = migrateLegacyInt(&c.Daemon.OrphanReap.IntervalMinutes, legacy.DaemonOrphanReapIntervalMinutes, "daemon_orphan_reap_interval_minutes", "daemon.orphan_reap.interval_minutes", &notices) || legacyFound
	legacyFound = migrateLegacyInt(&c.Daemon.DashboardWatchdog.IntervalSeconds, legacy.DaemonDashboardWatchdogIntervalSeconds, "daemon_dashboard_watchdog_interval_seconds", "daemon.dashboard_watchdog.interval_seconds", &notices) || legacyFound
	legacyFound = migrateLegacyInt(&c.Daemon.DashboardWatchdog.FailuresBeforeRestart, legacy.DaemonDashboardWatchdogFailuresBeforeRestart, "daemon_dashboard_watchdog_failures_before_restart", "daemon.dashboard_watchdog.failures_before_restart", &notices) || legacyFound
	legacyFound = migrateLegacyInt(&c.Daemon.DashboardWatchdog.RestartCooldownMinutes, legacy.DaemonDashboardWatchdogRestartCooldownMinutes, "daemon_dashboard_watchdog_restart_cooldown_minutes", "daemon.dashboard_watchdog.restart_cooldown_minutes", &notices) || legacyFound
	legacyFound = migrateLegacyInt(&c.Dashboard.Agents.ActiveMinutes, legacy.DashboardAgentsActiveMinutes, "dashboard_agents_active_minutes", "dashboard.agents.active_minutes", &notices) || legacyFound
	legacyFound = migrateLegacyInt(&c.Dashboard.Agents.GhostDisplayHours, legacy.DashboardAgentsGhostDisplayHours, "dashboard_agents_ghost_display_hours", "dashboard.agents.ghost_display_hours", &notices) || legacyFound
	legacyFound = migrateLegacyInt(&c.Dashboard.Agents.DeadMinutes, legacy.DashboardAgentsDeadMinutes, "dashboard_agents_dead_minutes", "dashboard.agents.dead_minutes", &notices) || legacyFound
	legacyFound = migrateLegacyInt(&c.Dashboard.Agents.StalledMinutes, legacy.DashboardAgentsStalledMinutes, "dashboard_agents_stalled_minutes", "dashboard.agents.stalled_minutes", &notices) || legacyFound
	legacyFound = migrateLegacyInt(&c.Dashboard.Agents.BeadsFetchHours, legacy.DashboardAgentsBeadsFetchHours, "dashboard_agents_beads_fetch_hours", "dashboard.agents.beads_fetch_hours", &notices) || legacyFound
	legacyFound = migrateLegacyInt(&c.Spawn.ContextQuality.Threshold, legacy.SpawnContextQualityThreshold, "spawn_context_quality_threshold", "spawn.context_quality.threshold", &notices) || legacyFound
	legacyFound = migrateLegacyInt(&c.Completion.AutoRebuild.TimeoutSeconds, legacy.CompletionAutoRebuildTimeoutSeconds, "completion_auto_rebuild_timeout_seconds", "completion.auto_rebuild.timeout_seconds", &notices) || legacyFound
	legacyFound = migrateLegacyInt(&c.Completion.TranscriptExport.TimeoutSeconds, legacy.CompletionTranscriptExportTimeoutSeconds, "completion_transcript_export_timeout_seconds", "completion.transcript_export.timeout_seconds", &notices) || legacyFound
	legacyFound = migrateLegacyInt(&c.Completion.CacheInvalidate.TimeoutSeconds, legacy.CompletionCacheInvalidateTimeoutSeconds, "completion_cache_invalidate_timeout_seconds", "completion.cache_invalidate.timeout_seconds", &notices) || legacyFound

	return legacyFound, notices
}

// Policy defaults for project-level config knobs.
const (
	DefaultDaemonCleanupIntervalMinutes             = 30
	DefaultDaemonCleanupSessionsAgeDays             = 7
	DefaultDaemonCleanupWorkspacesAgeDays           = 7
	DefaultDaemonDeadSessionIntervalMinutes         = 10
	DefaultDaemonMaxDeadSessionRetries              = 2
	DefaultDaemonOrphanReapIntervalMinutes          = 5
	DefaultDaemonDashboardWatchdogIntervalSeconds   = 30
	DefaultDaemonWatchdogFailuresBeforeRestart      = 2
	DefaultDaemonWatchdogRestartCooldownMinutes     = 5
	DefaultDashboardAgentsActiveMinutes             = 10
	DefaultDashboardAgentsGhostDisplayHours         = 4
	DefaultDashboardAgentsDeadMinutes               = 3
	DefaultDashboardAgentsStalledMinutes            = 15
	DefaultDashboardAgentsBeadsFetchHours           = 2
	DefaultSpawnContextQualityThreshold             = 20
	DefaultCompletionAutoRebuildTimeoutSeconds      = 120
	DefaultCompletionTranscriptExportTimeoutSeconds = 10
	DefaultCompletionCacheInvalidateTimeoutSeconds  = 2
)

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

	var legacy legacyFlatConfig
	if err := yaml.Unmarshal(data, &legacy); err != nil {
		return nil, fmt.Errorf("failed to parse legacy config keys: %w", err)
	}

	legacyFound, notices := cfg.applyLegacyFlatConfig(legacy)
	if legacyFound {
		if err := Save(projectDir, &cfg); err != nil {
			fmt.Fprintf(os.Stderr, "DEPRECATED: legacy flat config keys detected in %s but auto-migration failed: %v\n", configPath, err)
		} else {
			fmt.Fprintf(os.Stderr, "DEPRECATED: legacy flat config keys detected in %s\n", configPath)
			for _, notice := range notices {
				fmt.Fprintf(os.Stderr, "  - %s\n", notice)
			}
			fmt.Fprintf(os.Stderr, "Auto-migrated to nested keys. Migration guide: %s\n", legacyFlatConfigMigrationGuidePath)
		}
	}

	// Apply defaults for backward compatibility
	cfg.ApplyDefaults()

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

// ApplyDefaults sets default values for unspecified config fields.
func (c *Config) ApplyDefaults() {
	// NOTE: Do NOT default SpawnMode here - let it stay empty so global config is respected
	// The backend priority chain is: --backend flag > project config > global config > code default
	// Setting a default here would prevent global config from being used

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

// DaemonCleanupIntervalMinutes returns daemon cleanup interval in minutes.
func (c *Config) DaemonCleanupIntervalMinutes() int {
	if c.Daemon.Cleanup.IntervalMinutes > 0 {
		return c.Daemon.Cleanup.IntervalMinutes
	}
	return DefaultDaemonCleanupIntervalMinutes
}

// DaemonCleanupSessionsAgeDays returns session cleanup age threshold in days.
func (c *Config) DaemonCleanupSessionsAgeDays() int {
	if c.Daemon.Cleanup.SessionsAgeDays > 0 {
		return c.Daemon.Cleanup.SessionsAgeDays
	}
	return DefaultDaemonCleanupSessionsAgeDays
}

// DaemonCleanupWorkspacesAgeDays returns workspace cleanup age threshold in days.
func (c *Config) DaemonCleanupWorkspacesAgeDays() int {
	if c.Daemon.Cleanup.WorkspacesAgeDays > 0 {
		return c.Daemon.Cleanup.WorkspacesAgeDays
	}
	return DefaultDaemonCleanupWorkspacesAgeDays
}

// DaemonDeadSessionIntervalMinutes returns dead-session detection interval in minutes.
func (c *Config) DaemonDeadSessionIntervalMinutes() int {
	if c.Daemon.DeadSession.IntervalMinutes > 0 {
		return c.Daemon.DeadSession.IntervalMinutes
	}
	return DefaultDaemonDeadSessionIntervalMinutes
}

// DaemonMaxDeadSessionRetries returns maximum dead-session retries before escalation.
func (c *Config) DaemonMaxDeadSessionRetries() int {
	if c.Daemon.DeadSession.MaxRetries > 0 {
		return c.Daemon.DeadSession.MaxRetries
	}
	return DefaultDaemonMaxDeadSessionRetries
}

// DaemonOrphanReapIntervalMinutes returns orphan reaper interval in minutes.
func (c *Config) DaemonOrphanReapIntervalMinutes() int {
	if c.Daemon.OrphanReap.IntervalMinutes > 0 {
		return c.Daemon.OrphanReap.IntervalMinutes
	}
	return DefaultDaemonOrphanReapIntervalMinutes
}

// DaemonDashboardWatchdogIntervalSeconds returns dashboard watchdog interval in seconds.
func (c *Config) DaemonDashboardWatchdogIntervalSeconds() int {
	if c.Daemon.DashboardWatchdog.IntervalSeconds > 0 {
		return c.Daemon.DashboardWatchdog.IntervalSeconds
	}
	return DefaultDaemonDashboardWatchdogIntervalSeconds
}

// DaemonDashboardWatchdogFailuresBeforeRestart returns required consecutive failures before restart.
func (c *Config) DaemonDashboardWatchdogFailuresBeforeRestart() int {
	if c.Daemon.DashboardWatchdog.FailuresBeforeRestart > 0 {
		return c.Daemon.DashboardWatchdog.FailuresBeforeRestart
	}
	return DefaultDaemonWatchdogFailuresBeforeRestart
}

// DaemonDashboardWatchdogRestartCooldownMinutes returns restart cooldown in minutes.
func (c *Config) DaemonDashboardWatchdogRestartCooldownMinutes() int {
	if c.Daemon.DashboardWatchdog.RestartCooldownMinutes > 0 {
		return c.Daemon.DashboardWatchdog.RestartCooldownMinutes
	}
	return DefaultDaemonWatchdogRestartCooldownMinutes
}

// DashboardAgentsActiveMinutes returns active threshold for dashboard agent status in minutes.
func (c *Config) DashboardAgentsActiveMinutes() int {
	if c.Dashboard.Agents.ActiveMinutes > 0 {
		return c.Dashboard.Agents.ActiveMinutes
	}
	return DefaultDashboardAgentsActiveMinutes
}

// DashboardAgentsGhostDisplayHours returns ghost display threshold in hours.
func (c *Config) DashboardAgentsGhostDisplayHours() int {
	if c.Dashboard.Agents.GhostDisplayHours > 0 {
		return c.Dashboard.Agents.GhostDisplayHours
	}
	return DefaultDashboardAgentsGhostDisplayHours
}

// DashboardAgentsDeadMinutes returns dead threshold for dashboard agent status in minutes.
func (c *Config) DashboardAgentsDeadMinutes() int {
	if c.Dashboard.Agents.DeadMinutes > 0 {
		return c.Dashboard.Agents.DeadMinutes
	}
	return DefaultDashboardAgentsDeadMinutes
}

// DashboardAgentsStalledMinutes returns stalled threshold in minutes.
func (c *Config) DashboardAgentsStalledMinutes() int {
	if c.Dashboard.Agents.StalledMinutes > 0 {
		return c.Dashboard.Agents.StalledMinutes
	}
	return DefaultDashboardAgentsStalledMinutes
}

// DashboardAgentsBeadsFetchHours returns beads fetch threshold in hours.
func (c *Config) DashboardAgentsBeadsFetchHours() int {
	if c.Dashboard.Agents.BeadsFetchHours > 0 {
		return c.Dashboard.Agents.BeadsFetchHours
	}
	return DefaultDashboardAgentsBeadsFetchHours
}

// SpawnContextQualityThreshold returns context gate threshold.
func (c *Config) SpawnContextQualityThreshold() int {
	if c.Spawn.ContextQuality.Threshold > 0 {
		return c.Spawn.ContextQuality.Threshold
	}
	return DefaultSpawnContextQualityThreshold
}

// CompletionAutoRebuildTimeoutSeconds returns auto-rebuild timeout in seconds.
func (c *Config) CompletionAutoRebuildTimeoutSeconds() int {
	if c.Completion.AutoRebuild.TimeoutSeconds > 0 {
		return c.Completion.AutoRebuild.TimeoutSeconds
	}
	return DefaultCompletionAutoRebuildTimeoutSeconds
}

// CompletionTranscriptExportTimeoutSeconds returns transcript export timeout in seconds.
func (c *Config) CompletionTranscriptExportTimeoutSeconds() int {
	if c.Completion.TranscriptExport.TimeoutSeconds > 0 {
		return c.Completion.TranscriptExport.TimeoutSeconds
	}
	return DefaultCompletionTranscriptExportTimeoutSeconds
}

// CompletionCacheInvalidateTimeoutSeconds returns cache invalidation timeout in seconds.
func (c *Config) CompletionCacheInvalidateTimeoutSeconds() int {
	if c.Completion.CacheInvalidate.TimeoutSeconds > 0 {
		return c.Completion.CacheInvalidate.TimeoutSeconds
	}
	return DefaultCompletionCacheInvalidateTimeoutSeconds
}
