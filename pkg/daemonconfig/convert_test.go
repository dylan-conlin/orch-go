package daemonconfig

import (
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)


func TestFromUserConfig_Defaults(t *testing.T) {
	cfg := userconfig.DefaultConfig()
	result := FromUserConfig(cfg)

	// Fields mapped from userconfig accessor defaults
	if result.PollInterval != 60*time.Second {
		t.Errorf("PollInterval = %v, want 60s", result.PollInterval)
	}
	if result.MaxAgents != 5 {
		t.Errorf("MaxAgents = %d, want 5", result.MaxAgents)
	}
	if result.Label != "triage:ready" {
		t.Errorf("Label = %q, want %q", result.Label, "triage:ready")
	}
	if result.Verbose != true {
		t.Errorf("Verbose = %v, want true", result.Verbose)
	}
	if result.ReflectEnabled != true {
		t.Errorf("ReflectEnabled = %v, want true", result.ReflectEnabled)
	}
	if result.ReflectInterval != 60*time.Minute {
		t.Errorf("ReflectInterval = %v, want 60m", result.ReflectInterval)
	}
	if result.ReflectCreateIssues != false {
		t.Errorf("ReflectCreateIssues = %v, want false", result.ReflectCreateIssues)
	}
	if result.ReflectOpenEnabled != false {
		t.Errorf("ReflectOpenEnabled = %v, want false", result.ReflectOpenEnabled)
	}
	if result.DryRun != false {
		t.Errorf("DryRun = %v, want false", result.DryRun)
	}

	// All fields should now be backed by userconfig accessors with matching defaults
	defaults := DefaultConfig()
	if result.MaxSpawnsPerHour != defaults.MaxSpawnsPerHour {
		t.Errorf("MaxSpawnsPerHour = %d, want %d", result.MaxSpawnsPerHour, defaults.MaxSpawnsPerHour)
	}
	if result.SpawnDelay != defaults.SpawnDelay {
		t.Errorf("SpawnDelay = %v, want %v", result.SpawnDelay, defaults.SpawnDelay)
	}
	if result.RecoveryEnabled != defaults.RecoveryEnabled {
		t.Errorf("RecoveryEnabled = %v, want %v", result.RecoveryEnabled, defaults.RecoveryEnabled)
	}
	if result.CleanupEnabled != defaults.CleanupEnabled {
		t.Errorf("CleanupEnabled = %v, want %v", result.CleanupEnabled, defaults.CleanupEnabled)
	}
	if result.VerificationPauseThreshold != defaults.VerificationPauseThreshold {
		t.Errorf("VerificationPauseThreshold = %d, want %d", result.VerificationPauseThreshold, defaults.VerificationPauseThreshold)
	}

	// Verify previously-missing fields are now present and match defaults
	if result.OrphanDetectionEnabled != defaults.OrphanDetectionEnabled {
		t.Errorf("OrphanDetectionEnabled = %v, want %v", result.OrphanDetectionEnabled, defaults.OrphanDetectionEnabled)
	}
	if result.OrphanDetectionInterval != defaults.OrphanDetectionInterval {
		t.Errorf("OrphanDetectionInterval = %v, want %v", result.OrphanDetectionInterval, defaults.OrphanDetectionInterval)
	}
	if result.OrphanAgeThreshold != defaults.OrphanAgeThreshold {
		t.Errorf("OrphanAgeThreshold = %v, want %v", result.OrphanAgeThreshold, defaults.OrphanAgeThreshold)
	}
	if result.PhaseTimeoutEnabled != defaults.PhaseTimeoutEnabled {
		t.Errorf("PhaseTimeoutEnabled = %v, want %v", result.PhaseTimeoutEnabled, defaults.PhaseTimeoutEnabled)
	}
	if result.PhaseTimeoutInterval != defaults.PhaseTimeoutInterval {
		t.Errorf("PhaseTimeoutInterval = %v, want %v", result.PhaseTimeoutInterval, defaults.PhaseTimeoutInterval)
	}
	if result.PhaseTimeoutThreshold != defaults.PhaseTimeoutThreshold {
		t.Errorf("PhaseTimeoutThreshold = %v, want %v", result.PhaseTimeoutThreshold, defaults.PhaseTimeoutThreshold)
	}
	if result.AgreementCheckEnabled != defaults.AgreementCheckEnabled {
		t.Errorf("AgreementCheckEnabled = %v, want %v", result.AgreementCheckEnabled, defaults.AgreementCheckEnabled)
	}
	if result.AgreementCheckInterval != defaults.AgreementCheckInterval {
		t.Errorf("AgreementCheckInterval = %v, want %v", result.AgreementCheckInterval, defaults.AgreementCheckInterval)
	}
	if result.InvariantCheckEnabled != defaults.InvariantCheckEnabled {
		t.Errorf("InvariantCheckEnabled = %v, want %v", result.InvariantCheckEnabled, defaults.InvariantCheckEnabled)
	}
	if result.InvariantViolationThreshold != defaults.InvariantViolationThreshold {
		t.Errorf("InvariantViolationThreshold = %d, want %d", result.InvariantViolationThreshold, defaults.InvariantViolationThreshold)
	}
	if result.CleanupArchivedTTLDays != defaults.CleanupArchivedTTLDays {
		t.Errorf("CleanupArchivedTTLDays = %d, want %d", result.CleanupArchivedTTLDays, defaults.CleanupArchivedTTLDays)
	}
}


func TestFromUserConfig_CustomValues(t *testing.T) {
	pollInterval := 30
	maxAgents := 5
	verbose := false
	reflectIssues := true
	reflectOpen := true

	cfg := &userconfig.Config{
		Daemon: userconfig.DaemonConfig{
			PollInterval:  &pollInterval,
			MaxAgents:     &maxAgents,
			Label:         "custom:label",
			Verbose:       &verbose,
			ReflectIssues: &reflectIssues,
			ReflectOpen:   &reflectOpen,
		},
	}

	result := FromUserConfig(cfg)

	if result.PollInterval != 30*time.Second {
		t.Errorf("PollInterval = %v, want 30s", result.PollInterval)
	}
	if result.MaxAgents != 5 {
		t.Errorf("MaxAgents = %d, want 5", result.MaxAgents)
	}
	if result.Label != "custom:label" {
		t.Errorf("Label = %q, want %q", result.Label, "custom:label")
	}
	if result.Verbose != false {
		t.Errorf("Verbose = %v, want false", result.Verbose)
	}
	if result.ReflectCreateIssues != true {
		t.Errorf("ReflectCreateIssues = %v, want true", result.ReflectCreateIssues)
	}
	if result.ReflectOpenEnabled != true {
		t.Errorf("ReflectOpenEnabled = %v, want true", result.ReflectOpenEnabled)
	}
}


func TestFromUserConfig_AllFieldsCustom(t *testing.T) {
	// Verify all fields can be overridden via userconfig
	maxSpawns := 10
	spawnDelay := 5
	cleanupEnabled := false
	cleanupHours := 12
	cleanupDays := 14
	cleanupPreserve := false
	cleanupTTL := 60
	recoveryEnabled := false
	recoveryInterval := 15
	recoveryIdle := 20
	recoveryRate := 120
	orphanEnabled := false
	orphanInterval := 60
	orphanAge := 120
	ptEnabled := false
	ptInterval := 10
	ptThreshold := 60
	agreeEnabled := false
	agreeInterval := 60
	invEnabled := false
	invThreshold := 5

	cfg := &userconfig.Config{
		Daemon: userconfig.DaemonConfig{
			MaxSpawnsPerHour:               &maxSpawns,
			SpawnDelaySeconds:              &spawnDelay,
			CleanupEnabled:                 &cleanupEnabled,
			CleanupIntervalHours:           &cleanupHours,
			CleanupAgeDays:                 &cleanupDays,
			CleanupPreserveOrchestrator:    &cleanupPreserve,
			CleanupServerURL:               "http://custom:9999",
			CleanupArchivedTTLDays:         &cleanupTTL,
			RecoveryEnabled:                &recoveryEnabled,
			RecoveryIntervalMinutes:        &recoveryInterval,
			RecoveryIdleThresholdMinutes:   &recoveryIdle,
			RecoveryRateLimitMinutes:       &recoveryRate,
			OrphanDetectionEnabled:         &orphanEnabled,
			OrphanDetectionIntervalMinutes: &orphanInterval,
			OrphanAgeThresholdMinutes:      &orphanAge,
			PhaseTimeoutEnabled:            &ptEnabled,
			PhaseTimeoutIntervalMinutes:    &ptInterval,
			PhaseTimeoutThresholdMinutes:   &ptThreshold,
			AgreementCheckEnabled:          &agreeEnabled,
			AgreementCheckIntervalMinutes:  &agreeInterval,
			InvariantCheckEnabled:          &invEnabled,
			InvariantViolationThreshold:    &invThreshold,
		},
	}

	result := FromUserConfig(cfg)

	if result.MaxSpawnsPerHour != 10 {
		t.Errorf("MaxSpawnsPerHour = %d, want 10", result.MaxSpawnsPerHour)
	}
	if result.SpawnDelay != 5*time.Second {
		t.Errorf("SpawnDelay = %v, want 5s", result.SpawnDelay)
	}
	if result.CleanupEnabled != false {
		t.Errorf("CleanupEnabled = %v, want false", result.CleanupEnabled)
	}
	if result.CleanupInterval != 12*time.Hour {
		t.Errorf("CleanupInterval = %v, want 12h", result.CleanupInterval)
	}
	if result.CleanupAgeDays != 14 {
		t.Errorf("CleanupAgeDays = %d, want 14", result.CleanupAgeDays)
	}
	if result.CleanupPreserveOrchestrator != false {
		t.Errorf("CleanupPreserveOrchestrator = %v, want false", result.CleanupPreserveOrchestrator)
	}
	if result.CleanupServerURL != "http://custom:9999" {
		t.Errorf("CleanupServerURL = %q, want %q", result.CleanupServerURL, "http://custom:9999")
	}
	if result.CleanupArchivedTTLDays != 60 {
		t.Errorf("CleanupArchivedTTLDays = %d, want 60", result.CleanupArchivedTTLDays)
	}
	if result.RecoveryEnabled != false {
		t.Errorf("RecoveryEnabled = %v, want false", result.RecoveryEnabled)
	}
	if result.RecoveryInterval != 15*time.Minute {
		t.Errorf("RecoveryInterval = %v, want 15m", result.RecoveryInterval)
	}
	if result.RecoveryIdleThreshold != 20*time.Minute {
		t.Errorf("RecoveryIdleThreshold = %v, want 20m", result.RecoveryIdleThreshold)
	}
	if result.RecoveryRateLimit != 120*time.Minute {
		t.Errorf("RecoveryRateLimit = %v, want 120m", result.RecoveryRateLimit)
	}
	if result.OrphanDetectionEnabled != false {
		t.Errorf("OrphanDetectionEnabled = %v, want false", result.OrphanDetectionEnabled)
	}
	if result.OrphanDetectionInterval != 60*time.Minute {
		t.Errorf("OrphanDetectionInterval = %v, want 60m", result.OrphanDetectionInterval)
	}
	if result.OrphanAgeThreshold != 120*time.Minute {
		t.Errorf("OrphanAgeThreshold = %v, want 120m", result.OrphanAgeThreshold)
	}
	if result.PhaseTimeoutEnabled != false {
		t.Errorf("PhaseTimeoutEnabled = %v, want false", result.PhaseTimeoutEnabled)
	}
	if result.PhaseTimeoutInterval != 10*time.Minute {
		t.Errorf("PhaseTimeoutInterval = %v, want 10m", result.PhaseTimeoutInterval)
	}
	if result.PhaseTimeoutThreshold != 60*time.Minute {
		t.Errorf("PhaseTimeoutThreshold = %v, want 60m", result.PhaseTimeoutThreshold)
	}
	if result.AgreementCheckEnabled != false {
		t.Errorf("AgreementCheckEnabled = %v, want false", result.AgreementCheckEnabled)
	}
	if result.AgreementCheckInterval != 60*time.Minute {
		t.Errorf("AgreementCheckInterval = %v, want 60m", result.AgreementCheckInterval)
	}
	if result.InvariantCheckEnabled != false {
		t.Errorf("InvariantCheckEnabled = %v, want false", result.InvariantCheckEnabled)
	}
	if result.InvariantViolationThreshold != 5 {
		t.Errorf("InvariantViolationThreshold = %d, want 5", result.InvariantViolationThreshold)
	}
}


func TestFromUserConfig_ComplianceNil(t *testing.T) {
	cfg := userconfig.DefaultConfig()
	result := FromUserConfig(cfg)

	// No compliance config = default to Strict
	if result.Compliance.Default != ComplianceStrict {
		t.Errorf("Compliance.Default = %v, want strict", result.Compliance.Default)
	}
	if got := result.Compliance.Resolve("feature-impl", "opus"); got != ComplianceStrict {
		t.Errorf("Compliance.Resolve() = %v, want strict", got)
	}
}


func TestFromUserConfig_ComplianceWithOverrides(t *testing.T) {
	cfg := &userconfig.Config{
		Daemon: userconfig.DaemonConfig{
			Compliance: &userconfig.ComplianceYAMLConfig{
				Default: "standard",
				Skills: map[string]string{
					"architect": "strict",
					"issue-creation": "autonomous",
				},
				Models: map[string]string{
					"opus": "relaxed",
				},
				Combos: map[string]string{
					"opus+feature-impl": "standard",
				},
			},
		},
	}

	result := FromUserConfig(cfg)

	if result.Compliance.Default != ComplianceStandard {
		t.Errorf("Compliance.Default = %v, want standard", result.Compliance.Default)
	}

	// Combo overrides everything
	if got := result.Compliance.Resolve("feature-impl", "opus"); got != ComplianceStandard {
		t.Errorf("Resolve combo = %v, want standard", got)
	}
	// Skill override
	if got := result.Compliance.Resolve("architect", "sonnet"); got != ComplianceStrict {
		t.Errorf("Resolve skill = %v, want strict", got)
	}
	// Model override
	if got := result.Compliance.Resolve("investigation", "opus"); got != ComplianceRelaxed {
		t.Errorf("Resolve model = %v, want relaxed", got)
	}
	// Default fallthrough
	if got := result.Compliance.Resolve("investigation", "sonnet"); got != ComplianceStandard {
		t.Errorf("Resolve default = %v, want standard", got)
	}
}


func TestFromUserConfig_ComplianceInvalidLevel(t *testing.T) {
	cfg := &userconfig.Config{
		Daemon: userconfig.DaemonConfig{
			Compliance: &userconfig.ComplianceYAMLConfig{
				Default: "invalid_level",
				Skills: map[string]string{
					"feature-impl": "also_invalid",
					"architect":    "strict",
				},
			},
		},
	}

	result := FromUserConfig(cfg)

	// Invalid default falls back to strict (zero value)
	if result.Compliance.Default != ComplianceStrict {
		t.Errorf("Compliance.Default = %v, want strict (invalid ignored)", result.Compliance.Default)
	}
	// Invalid skill level should be skipped
	if _, ok := result.Compliance.Skills["feature-impl"]; ok {
		t.Error("Invalid skill level should not be in Skills map")
	}
	// Valid skill level should be present
	if got, ok := result.Compliance.Skills["architect"]; !ok || got != ComplianceStrict {
		t.Errorf("Skills[architect] = %v, want strict", got)
	}
}


func TestFromUserConfig_ReflectConfig(t *testing.T) {
	enabled := false
	interval := 120
	createIssues := false

	cfg := &userconfig.Config{
		Reflect: userconfig.ReflectConfig{
			Enabled:         &enabled,
			IntervalMinutes: &interval,
			CreateIssues:    &createIssues,
		},
	}

	result := FromUserConfig(cfg)

	if result.ReflectEnabled != false {
		t.Errorf("ReflectEnabled = %v, want false", result.ReflectEnabled)
	}
	if result.ReflectInterval != 120*time.Minute {
		t.Errorf("ReflectInterval = %v, want 120m", result.ReflectInterval)
	}
}
