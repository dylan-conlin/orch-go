package daemonconfig

import (
	"time"

	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)

// FromUserConfig converts a userconfig.Config into a runtime daemon Config.
// Fields with userconfig backing are mapped from the config's accessor methods
// (which apply defaults for unset values). Fields without userconfig backing
// use DefaultConfig() values.
func FromUserConfig(cfg *userconfig.Config) Config {
	defaults := DefaultConfig()

	return Config{
		PollInterval:     time.Duration(cfg.DaemonPollInterval()) * time.Second,
		MaxAgents:        cfg.DaemonMaxAgents(),
		MaxSpawnsPerHour: defaults.MaxSpawnsPerHour,
		Label:            cfg.DaemonLabel(),
		SpawnDelay:       defaults.SpawnDelay,
		DryRun:           false,
		Verbose:          cfg.DaemonVerbose(),

		ReflectEnabled:      cfg.ReflectEnabled(),
		ReflectInterval:     time.Duration(cfg.ReflectIntervalMinutes()) * time.Minute,
		ReflectCreateIssues: cfg.DaemonReflectIssues(),
		ReflectOpenEnabled:  cfg.DaemonReflectOpen(),

		ReflectModelDriftEnabled:  defaults.ReflectModelDriftEnabled,
		ReflectModelDriftInterval: defaults.ReflectModelDriftInterval,

		CleanupEnabled:              defaults.CleanupEnabled,
		CleanupInterval:             defaults.CleanupInterval,
		CleanupAgeDays:              defaults.CleanupAgeDays,
		CleanupPreserveOrchestrator: defaults.CleanupPreserveOrchestrator,
		CleanupServerURL:            defaults.CleanupServerURL,

		RecoveryEnabled:       defaults.RecoveryEnabled,
		RecoveryInterval:      defaults.RecoveryInterval,
		RecoveryIdleThreshold: defaults.RecoveryIdleThreshold,
		RecoveryRateLimit:     defaults.RecoveryRateLimit,

		VerificationPauseThreshold: defaults.VerificationPauseThreshold,
	}
}
