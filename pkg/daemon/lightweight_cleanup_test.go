package daemon

import (
	"testing"
	"time"
)

func TestRunPeriodicLightweightCleanup_NotDue(t *testing.T) {
	d := New()
	// Set last run to recent — should not be due
	d.Scheduler.SetLastRun(TaskLightweightCleanup, time.Now())

	result := d.RunPeriodicLightweightCleanup()
	if result != nil {
		t.Error("expected nil result when not due")
	}
}

func TestRunPeriodicLightweightCleanup_DueOnFirstRun(t *testing.T) {
	d := New()
	// Never run before — should be due, but will fail to connect to beads (no daemon)
	result := d.RunPeriodicLightweightCleanup()
	if result == nil {
		t.Fatal("expected non-nil result on first run")
	}
	// Should get an error since beads isn't running in test
	if result.Error == nil {
		if result.Message == "" {
			t.Error("expected non-empty message")
		}
	}
}

func TestLightweightCleanupResult_Snapshot(t *testing.T) {
	result := &LightweightCleanupResult{
		ClosedCount:  3,
		ScannedCount: 7,
	}
	snapshot := result.Snapshot()
	if snapshot.ClosedCount != 3 {
		t.Errorf("ClosedCount = %d, want 3", snapshot.ClosedCount)
	}
	if snapshot.ScannedCount != 7 {
		t.Errorf("ScannedCount = %d, want 7", snapshot.ScannedCount)
	}
	if snapshot.LastCheck.IsZero() {
		t.Error("LastCheck should not be zero")
	}
}

func TestLabelLightweight(t *testing.T) {
	if LabelLightweight != "tier:lightweight" {
		t.Errorf("LabelLightweight = %q, want 'tier:lightweight'", LabelLightweight)
	}
}

func TestDefaultConfig_LightweightCleanup(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.LightweightCleanupEnabled {
		t.Error("LightweightCleanupEnabled should default to true")
	}
	if cfg.LightweightCleanupInterval != 30*time.Minute {
		t.Errorf("LightweightCleanupInterval = %v, want 30m", cfg.LightweightCleanupInterval)
	}
	if cfg.LightweightCleanupTimeout != 2*time.Hour {
		t.Errorf("LightweightCleanupTimeout = %v, want 2h", cfg.LightweightCleanupTimeout)
	}
}

func TestDefaultConfig_VerificationFailedEscalation(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.VerificationFailedEscalationEnabled {
		t.Error("VerificationFailedEscalationEnabled should default to true")
	}
	if cfg.VerificationFailedEscalationInterval != 30*time.Minute {
		t.Errorf("VerificationFailedEscalationInterval = %v, want 30m", cfg.VerificationFailedEscalationInterval)
	}
	if cfg.VerificationFailedEscalationTimeout != time.Hour {
		t.Errorf("VerificationFailedEscalationTimeout = %v, want 1h", cfg.VerificationFailedEscalationTimeout)
	}
}
