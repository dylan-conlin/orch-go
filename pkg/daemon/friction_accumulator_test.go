package daemon

import (
	"errors"
	"testing"
	"time"
)

var errTestFailure = errors.New("test failure")

func TestShouldRunFrictionAccumulation(t *testing.T) {
	t.Run("disabled returns false", func(t *testing.T) {
		cfg := Config{FrictionAccumulationEnabled: false, FrictionAccumulationInterval: time.Hour}
		d := &Daemon{Config: cfg, Scheduler: NewSchedulerFromConfig(cfg)}
		if d.ShouldRunFrictionAccumulation() {
			t.Error("expected false when disabled")
		}
	})

	t.Run("zero interval returns false", func(t *testing.T) {
		cfg := Config{FrictionAccumulationEnabled: true, FrictionAccumulationInterval: 0}
		d := &Daemon{Config: cfg, Scheduler: NewSchedulerFromConfig(cfg)}
		if d.ShouldRunFrictionAccumulation() {
			t.Error("expected false when interval is zero")
		}
	})

	t.Run("first run returns true", func(t *testing.T) {
		cfg := Config{FrictionAccumulationEnabled: true, FrictionAccumulationInterval: time.Hour}
		d := &Daemon{Config: cfg, Scheduler: NewSchedulerFromConfig(cfg)}
		if !d.ShouldRunFrictionAccumulation() {
			t.Error("expected true on first run")
		}
	})

	t.Run("not due returns false", func(t *testing.T) {
		cfg := Config{FrictionAccumulationEnabled: true, FrictionAccumulationInterval: time.Hour}
		d := &Daemon{
			Config:    cfg,
			Scheduler: NewSchedulerFromConfig(cfg),
		}
		d.Scheduler.SetLastRun(TaskFrictionAccumulation, time.Now())
		if d.ShouldRunFrictionAccumulation() {
			t.Error("expected false when not due")
		}
	})

	t.Run("past interval returns true", func(t *testing.T) {
		cfg := Config{FrictionAccumulationEnabled: true, FrictionAccumulationInterval: time.Hour}
		d := &Daemon{
			Config:    cfg,
			Scheduler: NewSchedulerFromConfig(cfg),
		}
		d.Scheduler.SetLastRun(TaskFrictionAccumulation, time.Now().Add(-2*time.Hour))
		if !d.ShouldRunFrictionAccumulation() {
			t.Error("expected true when past interval")
		}
	})
}

func TestRunPeriodicFrictionAccumulation(t *testing.T) {
	t.Run("returns nil when not due", func(t *testing.T) {
		cfg := Config{FrictionAccumulationEnabled: false}
		d := &Daemon{Config: cfg, Scheduler: NewSchedulerFromConfig(cfg)}
		result := d.RunPeriodicFrictionAccumulation()
		if result != nil {
			t.Error("expected nil when not due")
		}
	})

	t.Run("scans and accumulates friction items", func(t *testing.T) {
		scanned := false
		stored := false

		cfg := Config{FrictionAccumulationEnabled: true, FrictionAccumulationInterval: time.Hour}
		d := &Daemon{
			Config:    cfg,
			Scheduler: NewSchedulerFromConfig(cfg),
			FrictionAccumulator: &mockFrictionAccumulatorService{
				scanFn: func() ([]FrictionEntry, error) {
					scanned = true
					return []FrictionEntry{
						{BeadsID: "orch-go-abc1", Category: "bug", Description: "beads dir resolution fails"},
						{BeadsID: "orch-go-abc1", Category: "tooling", Description: "bd sync noise"},
						{BeadsID: "orch-go-def2", Category: "ceremony", Description: "process overhead"},
					}, nil
				},
				storeFn: func(entries []FrictionEntry) error {
					stored = true
					if len(entries) != 3 {
						t.Errorf("expected 3 entries to store, got %d", len(entries))
					}
					return nil
				},
			},
		}

		result := d.RunPeriodicFrictionAccumulation()
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Error != nil {
			t.Errorf("unexpected error: %v", result.Error)
		}
		if !scanned {
			t.Error("scan was not called")
		}
		if !stored {
			t.Error("store was not called")
		}
		if result.NewItems != 3 {
			t.Errorf("NewItems = %d, want 3", result.NewItems)
		}
		if result.ByCategoryCount["bug"] != 1 {
			t.Errorf("bug count = %d, want 1", result.ByCategoryCount["bug"])
		}
		if d.Scheduler.LastRunTime(TaskFrictionAccumulation).IsZero() {
			t.Error("lastFrictionAccumulation was not updated")
		}
	})

	t.Run("no items found", func(t *testing.T) {
		cfg := Config{FrictionAccumulationEnabled: true, FrictionAccumulationInterval: time.Hour}
		d := &Daemon{
			Config:    cfg,
			Scheduler: NewSchedulerFromConfig(cfg),
			FrictionAccumulator: &mockFrictionAccumulatorService{
				scanFn: func() ([]FrictionEntry, error) {
					return nil, nil
				},
			},
		}

		result := d.RunPeriodicFrictionAccumulation()
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.NewItems != 0 {
			t.Errorf("NewItems = %d, want 0", result.NewItems)
		}
	})

	t.Run("handles scan error", func(t *testing.T) {
		cfg := Config{FrictionAccumulationEnabled: true, FrictionAccumulationInterval: time.Hour}
		d := &Daemon{
			Config:    cfg,
			Scheduler: NewSchedulerFromConfig(cfg),
			FrictionAccumulator: &mockFrictionAccumulatorService{
				scanFn: func() ([]FrictionEntry, error) {
					return nil, errTestFailure
				},
			},
		}

		result := d.RunPeriodicFrictionAccumulation()
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Error == nil {
			t.Error("expected error in result")
		}
	})
}

func TestFrictionAccumulationSnapshot(t *testing.T) {
	result := &FrictionAccumulationResult{
		NewItems: 5,
		ByCategoryCount: map[string]int{
			"bug":      2,
			"ceremony": 1,
			"tooling":  1,
			"gap":      1,
		},
	}

	snap := result.Snapshot()
	if snap.NewItems != 5 {
		t.Errorf("NewItems = %d, want 5", snap.NewItems)
	}
	if snap.ByCategoryCount["bug"] != 2 {
		t.Errorf("bug count = %d, want 2", snap.ByCategoryCount["bug"])
	}
	if snap.LastCheck.IsZero() {
		t.Error("LastCheck should be set")
	}
}

// mockFrictionAccumulatorService provides a testable implementation.
type mockFrictionAccumulatorService struct {
	scanFn  func() ([]FrictionEntry, error)
	storeFn func(entries []FrictionEntry) error
}

func (m *mockFrictionAccumulatorService) Scan() ([]FrictionEntry, error) {
	if m.scanFn != nil {
		return m.scanFn()
	}
	return nil, nil
}

func (m *mockFrictionAccumulatorService) Store(entries []FrictionEntry) error {
	if m.storeFn != nil {
		return m.storeFn(entries)
	}
	return nil
}
