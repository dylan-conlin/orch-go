package userconfig

import (
	"os"
	"path/filepath"
)

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

	// PlanStalenessEnabled controls whether periodic plan staleness detection is enabled.
	// Defaults to true if not specified.
	PlanStalenessEnabled *bool `yaml:"plan_staleness_enabled,omitempty"`

	// PlanStalenessIntervalMinutes is how often to check for stale plans (in minutes).
	// Defaults to 30 if not specified.
	PlanStalenessIntervalMinutes *int `yaml:"plan_staleness_interval_minutes,omitempty"`

	// Compliance holds per-spawn compliance level configuration.
	// Levels: strict (default), standard, relaxed, autonomous.
	Compliance *ComplianceYAMLConfig `yaml:"compliance,omitempty"`
}

// ComplianceYAMLConfig is the YAML representation of compliance configuration.
// All levels are strings ("strict", "standard", "relaxed", "autonomous").
type ComplianceYAMLConfig struct {
	// Default is the global compliance level. Defaults to "strict".
	Default string `yaml:"default,omitempty"`
	// Skills maps skill names to compliance levels.
	Skills map[string]string `yaml:"skills,omitempty"`
	// Models maps model names to compliance levels.
	Models map[string]string `yaml:"models,omitempty"`
	// Combos maps "model+skill" keys to compliance levels (highest precedence).
	Combos map[string]string `yaml:"combos,omitempty"`
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

// DaemonPlanStalenessEnabled returns whether periodic plan staleness detection is enabled.
// Defaults to true if not configured.
func (c *Config) DaemonPlanStalenessEnabled() bool {
	if c.Daemon.PlanStalenessEnabled == nil {
		return true
	}
	return *c.Daemon.PlanStalenessEnabled
}

// DaemonPlanStalenessIntervalMinutes returns how often to check for stale plans (in minutes).
// Defaults to 30 if not configured.
func (c *Config) DaemonPlanStalenessIntervalMinutes() int {
	if c.Daemon.PlanStalenessIntervalMinutes == nil {
		return 30
	}
	return *c.Daemon.PlanStalenessIntervalMinutes
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

// DaemonComplianceConfig returns the parsed compliance YAML config.
// Returns nil if no compliance section is configured.
func (c *Config) DaemonComplianceConfig() *ComplianceYAMLConfig {
	return c.Daemon.Compliance
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
