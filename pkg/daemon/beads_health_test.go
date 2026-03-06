package daemon

import (
	"testing"
	"time"
)

func TestShouldRunBeadsHealth(t *testing.T) {
	t.Run("disabled returns false", func(t *testing.T) {
		cfg := Config{BeadsHealthEnabled: false, BeadsHealthInterval: time.Hour}
		d := &Daemon{Config: cfg, Scheduler: NewSchedulerFromConfig(cfg)}
		if d.ShouldRunBeadsHealth() {
			t.Error("expected false when disabled")
		}
	})

	t.Run("zero interval returns false", func(t *testing.T) {
		cfg := Config{BeadsHealthEnabled: true, BeadsHealthInterval: 0}
		d := &Daemon{Config: cfg, Scheduler: NewSchedulerFromConfig(cfg)}
		if d.ShouldRunBeadsHealth() {
			t.Error("expected false when interval is zero")
		}
	})

	t.Run("first run returns true", func(t *testing.T) {
		cfg := Config{BeadsHealthEnabled: true, BeadsHealthInterval: time.Hour}
		d := &Daemon{Config: cfg, Scheduler: NewSchedulerFromConfig(cfg)}
		if !d.ShouldRunBeadsHealth() {
			t.Error("expected true on first run")
		}
	})

	t.Run("not due returns false", func(t *testing.T) {
		cfg := Config{BeadsHealthEnabled: true, BeadsHealthInterval: time.Hour}
		d := &Daemon{
			Config:    cfg,
			Scheduler: NewSchedulerFromConfig(cfg),
		}
		d.Scheduler.SetLastRun(TaskBeadsHealth, time.Now())
		if d.ShouldRunBeadsHealth() {
			t.Error("expected false when not due")
		}
	})

	t.Run("past interval returns true", func(t *testing.T) {
		cfg := Config{BeadsHealthEnabled: true, BeadsHealthInterval: time.Hour}
		d := &Daemon{
			Config:    cfg,
			Scheduler: NewSchedulerFromConfig(cfg),
		}
		d.Scheduler.SetLastRun(TaskBeadsHealth, time.Now().Add(-2*time.Hour))
		if !d.ShouldRunBeadsHealth() {
			t.Error("expected true when past interval")
		}
	})
}

func TestRunPeriodicBeadsHealth(t *testing.T) {
	t.Run("returns nil when not due", func(t *testing.T) {
		cfg := Config{BeadsHealthEnabled: false}
		d := &Daemon{Config: cfg, Scheduler: NewSchedulerFromConfig(cfg)}
		result := d.RunPeriodicBeadsHealth()
		if result != nil {
			t.Error("expected nil when not due")
		}
	})

	t.Run("runs collector and stores snapshot", func(t *testing.T) {
		collected := false
		stored := false

		cfg := Config{BeadsHealthEnabled: true, BeadsHealthInterval: time.Hour}
		d := &Daemon{
			Config:    cfg,
			Scheduler: NewSchedulerFromConfig(cfg),
			BeadsHealth: &mockBeadsHealthService{
				collectFn: func() (*BeadsHealthResult, error) {
					collected = true
					return &BeadsHealthResult{
						OpenIssues:    10,
						BlockedIssues: 2,
						StaleIssues:   3,
						BloatedFiles:  1,
						FixFeatRatio:  0.5,
					}, nil
				},
				storeFn: func(result *BeadsHealthResult) error {
					stored = true
					return nil
				},
			},
		}

		result := d.RunPeriodicBeadsHealth()
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Error != nil {
			t.Errorf("unexpected error: %v", result.Error)
		}
		if !collected {
			t.Error("collector was not called")
		}
		if !stored {
			t.Error("store was not called")
		}
		if result.OpenIssues != 10 {
			t.Errorf("OpenIssues = %d, want 10", result.OpenIssues)
		}
		if d.Scheduler.LastRunTime(TaskBeadsHealth).IsZero() {
			t.Error("lastBeadsHealth was not updated")
		}
	})

	t.Run("handles collector error", func(t *testing.T) {
		cfg := Config{BeadsHealthEnabled: true, BeadsHealthInterval: time.Hour}
		d := &Daemon{
			Config:    cfg,
			Scheduler: NewSchedulerFromConfig(cfg),
			BeadsHealth: &mockBeadsHealthService{
				collectFn: func() (*BeadsHealthResult, error) {
					return nil, errTestFailure
				},
			},
		}

		result := d.RunPeriodicBeadsHealth()
		if result == nil {
			t.Fatal("expected non-nil result on error")
		}
		if result.Error == nil {
			t.Error("expected error in result")
		}
	})
}

func TestBeadsHealthSnapshot(t *testing.T) {
	result := &BeadsHealthResult{
		OpenIssues:    15,
		BlockedIssues: 3,
		StaleIssues:   5,
		BloatedFiles:  2,
		FixFeatRatio:  1.2,
	}

	snap := result.Snapshot()
	if snap.OpenIssues != 15 {
		t.Errorf("OpenIssues = %d, want 15", snap.OpenIssues)
	}
	if snap.BlockedIssues != 3 {
		t.Errorf("BlockedIssues = %d, want 3", snap.BlockedIssues)
	}
	if snap.StaleIssues != 5 {
		t.Errorf("StaleIssues = %d, want 5", snap.StaleIssues)
	}
	if snap.BloatedFiles != 2 {
		t.Errorf("BloatedFiles = %d, want 2", snap.BloatedFiles)
	}
	if snap.FixFeatRatio != 1.2 {
		t.Errorf("FixFeatRatio = %f, want 1.2", snap.FixFeatRatio)
	}
	if snap.LastCheck.IsZero() {
		t.Error("LastCheck should be set")
	}
}

// mockBeadsHealthService provides a testable implementation.
type mockBeadsHealthService struct {
	collectFn func() (*BeadsHealthResult, error)
	storeFn   func(result *BeadsHealthResult) error
}

func (m *mockBeadsHealthService) Collect() (*BeadsHealthResult, error) {
	if m.collectFn != nil {
		return m.collectFn()
	}
	return &BeadsHealthResult{}, nil
}

func (m *mockBeadsHealthService) Store(result *BeadsHealthResult) error {
	if m.storeFn != nil {
		return m.storeFn(result)
	}
	return nil
}
