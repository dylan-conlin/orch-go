package daemonconfig

import (
	"time"

	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)

// FromUserConfig converts a userconfig.Config into a runtime daemon Config.
// All fields are mapped from userconfig accessor methods, which apply defaults
// for unset values. No field uses DefaultConfig() directly.
func FromUserConfig(cfg *userconfig.Config) Config {
	return Config{
		PollInterval:     time.Duration(cfg.DaemonPollInterval()) * time.Second,
		MaxAgents:        cfg.DaemonMaxAgents(),
		MaxSpawnsPerHour: cfg.DaemonMaxSpawnsPerHour(),
		Label:            cfg.DaemonLabel(),
		SpawnDelay:       time.Duration(cfg.DaemonSpawnDelaySeconds()) * time.Second,
		DryRun:           false,
		Verbose:          cfg.DaemonVerbose(),

		ReflectEnabled:      cfg.ReflectEnabled(),
		ReflectInterval:     time.Duration(cfg.ReflectIntervalMinutes()) * time.Minute,
		ReflectCreateIssues: cfg.DaemonReflectIssues(),
		ReflectOpenEnabled:  cfg.DaemonReflectOpen(),

		ReflectModelDriftEnabled:  cfg.DaemonReflectModelDriftEnabled(),
		ReflectModelDriftInterval: time.Duration(cfg.DaemonReflectModelDriftIntervalHours()) * time.Hour,

		CleanupEnabled:              cfg.DaemonCleanupEnabled(),
		CleanupInterval:             time.Duration(cfg.DaemonCleanupIntervalHours()) * time.Hour,
		CleanupAgeDays:              cfg.DaemonCleanupAgeDays(),
		CleanupPreserveOrchestrator: cfg.DaemonCleanupPreserveOrchestrator(),
		CleanupServerURL:            cfg.DaemonCleanupServerURL(),
		CleanupArchivedTTLDays:      cfg.DaemonCleanupArchivedTTLDays(),

		RecoveryEnabled:       cfg.DaemonRecoveryEnabled(),
		RecoveryInterval:      time.Duration(cfg.DaemonRecoveryIntervalMinutes()) * time.Minute,
		RecoveryIdleThreshold: time.Duration(cfg.DaemonRecoveryIdleThresholdMinutes()) * time.Minute,
		RecoveryRateLimit:     time.Duration(cfg.DaemonRecoveryRateLimitMinutes()) * time.Minute,

		VerificationPauseThreshold: cfg.DaemonVerificationPauseThreshold(),

		KnowledgeHealthEnabled:   cfg.DaemonKnowledgeHealthEnabled(),
		KnowledgeHealthInterval:  time.Duration(cfg.DaemonKnowledgeHealthIntervalHours()) * time.Hour,
		KnowledgeHealthThreshold: cfg.DaemonKnowledgeHealthThreshold(),

		OrphanDetectionEnabled:  cfg.DaemonOrphanDetectionEnabled(),
		OrphanDetectionInterval: time.Duration(cfg.DaemonOrphanDetectionIntervalMinutes()) * time.Minute,
		OrphanAgeThreshold:      time.Duration(cfg.DaemonOrphanAgeThresholdMinutes()) * time.Minute,

		PhaseTimeoutEnabled:   cfg.DaemonPhaseTimeoutEnabled(),
		PhaseTimeoutInterval:  time.Duration(cfg.DaemonPhaseTimeoutIntervalMinutes()) * time.Minute,
		PhaseTimeoutThreshold: time.Duration(cfg.DaemonPhaseTimeoutThresholdMinutes()) * time.Minute,

		AgreementCheckEnabled:  cfg.DaemonAgreementCheckEnabled(),
		AgreementCheckInterval: time.Duration(cfg.DaemonAgreementCheckIntervalMinutes()) * time.Minute,

		InvariantCheckEnabled:       cfg.DaemonInvariantCheckEnabled(),
		InvariantViolationThreshold: cfg.DaemonInvariantViolationThreshold(),
	}
}
