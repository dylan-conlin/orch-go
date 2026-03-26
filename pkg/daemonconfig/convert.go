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


		Compliance:   complianceFromYAML(cfg.DaemonComplianceConfig()),
		ModelRouting: modelRoutingFromYAML(cfg.DaemonModelRoutingConfig()),
	}
}

// modelRoutingFromYAML converts a userconfig YAML model routing config to runtime ModelRoutingConfig.
func modelRoutingFromYAML(yamlCfg *userconfig.ModelRoutingYAMLConfig) *ModelRoutingConfig {
	if yamlCfg == nil {
		return nil
	}

	cfg := &ModelRoutingConfig{
		Default: yamlCfg.Default,
	}

	if len(yamlCfg.Skills) > 0 {
		cfg.Skills = make(map[string]string, len(yamlCfg.Skills))
		for k, v := range yamlCfg.Skills {
			cfg.Skills[k] = v
		}
	}

	if len(yamlCfg.Models) > 0 {
		cfg.Models = make(map[string]string, len(yamlCfg.Models))
		for k, v := range yamlCfg.Models {
			cfg.Models[k] = v
		}
	}

	if len(yamlCfg.Combos) > 0 {
		cfg.Combos = make(map[string]string, len(yamlCfg.Combos))
		for k, v := range yamlCfg.Combos {
			cfg.Combos[k] = v
		}
	}

	return cfg
}

// complianceFromYAML converts a userconfig YAML compliance config to runtime ComplianceConfig.
func complianceFromYAML(yamlCfg *userconfig.ComplianceYAMLConfig) ComplianceConfig {
	if yamlCfg == nil {
		return ComplianceConfig{} // Default = ComplianceStrict (zero value)
	}

	cc := ComplianceConfig{}

	if level, ok := ParseComplianceLevel(yamlCfg.Default); ok {
		cc.Default = level
	}

	if len(yamlCfg.Skills) > 0 {
		cc.Skills = make(map[string]ComplianceLevel, len(yamlCfg.Skills))
		for k, v := range yamlCfg.Skills {
			if level, ok := ParseComplianceLevel(v); ok {
				cc.Skills[k] = level
			}
		}
	}

	if len(yamlCfg.Models) > 0 {
		cc.Models = make(map[string]ComplianceLevel, len(yamlCfg.Models))
		for k, v := range yamlCfg.Models {
			if level, ok := ParseComplianceLevel(v); ok {
				cc.Models[k] = level
			}
		}
	}

	if len(yamlCfg.Combos) > 0 {
		cc.Combos = make(map[string]ComplianceLevel, len(yamlCfg.Combos))
		for k, v := range yamlCfg.Combos {
			if level, ok := ParseComplianceLevel(v); ok {
				cc.Combos[k] = level
			}
		}
	}

	return cc
}
