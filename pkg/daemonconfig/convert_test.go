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

	// Fields from DefaultConfig (no userconfig backing)
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
