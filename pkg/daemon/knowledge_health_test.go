package daemon

import (
	"fmt"
	"testing"
	"time"
)

func TestDaemon_ShouldRunKnowledgeHealth_Disabled(t *testing.T) {
	cfg := Config{
		KnowledgeHealthEnabled:  false,
		KnowledgeHealthInterval: 2 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}

	if d.ShouldRunKnowledgeHealth() {
		t.Error("ShouldRunKnowledgeHealth() should return false when disabled")
	}
}

func TestDaemon_ShouldRunKnowledgeHealth_ZeroInterval(t *testing.T) {
	cfg := Config{
		KnowledgeHealthEnabled:  true,
		KnowledgeHealthInterval: 0,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}

	if d.ShouldRunKnowledgeHealth() {
		t.Error("ShouldRunKnowledgeHealth() should return false when interval is 0")
	}
}

func TestDaemon_ShouldRunKnowledgeHealth_NeverRun(t *testing.T) {
	cfg := Config{
		KnowledgeHealthEnabled:  true,
		KnowledgeHealthInterval: 2 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}

	if !d.ShouldRunKnowledgeHealth() {
		t.Error("ShouldRunKnowledgeHealth() should return true when never run before")
	}
}

func TestDaemon_ShouldRunKnowledgeHealth_IntervalElapsed(t *testing.T) {
	cfg := Config{
		KnowledgeHealthEnabled:  true,
		KnowledgeHealthInterval: 2 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}
	d.Scheduler.SetLastRun(TaskKnowledgeHealth, time.Now().Add(-3*time.Hour))

	if !d.ShouldRunKnowledgeHealth() {
		t.Error("ShouldRunKnowledgeHealth() should return true when interval has elapsed")
	}
}

func TestDaemon_ShouldRunKnowledgeHealth_IntervalNotElapsed(t *testing.T) {
	cfg := Config{
		KnowledgeHealthEnabled:  true,
		KnowledgeHealthInterval: 2 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}
	d.Scheduler.SetLastRun(TaskKnowledgeHealth, time.Now().Add(-1*time.Hour))

	if d.ShouldRunKnowledgeHealth() {
		t.Error("ShouldRunKnowledgeHealth() should return false when interval has not elapsed")
	}
}

func TestDaemon_RunPeriodicKnowledgeHealth_NotDue(t *testing.T) {
	called := false
	cfg := Config{
		KnowledgeHealthEnabled:  true,
		KnowledgeHealthInterval: 2 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		KnowledgeHealth: &mockKnowledgeHealthService{CheckFunc: func() (*KnowledgeHealthResult, error) {
			called = true
			return &KnowledgeHealthResult{}, nil
		}},
	}
	d.Scheduler.SetLastRun(TaskKnowledgeHealth, time.Now())

	result := d.RunPeriodicKnowledgeHealth()
	if result != nil {
		t.Error("RunPeriodicKnowledgeHealth() should return nil when not due")
	}
	if called {
		t.Error("KnowledgeHealth.Check should not be called when not due")
	}
}

func TestDaemon_RunPeriodicKnowledgeHealth_Due(t *testing.T) {
	called := false
	cfg := Config{
		KnowledgeHealthEnabled:   true,
		KnowledgeHealthInterval:  2 * time.Hour,
		KnowledgeHealthThreshold: 20,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		KnowledgeHealth: &mockKnowledgeHealthService{CheckFunc: func() (*KnowledgeHealthResult, error) {
			called = true
			return &KnowledgeHealthResult{
				TotalActive: 15,
				ByType: map[string]int{
					"decision":   10,
					"constraint": 5,
				},
			}, nil
		}},
	}
	d.Scheduler.SetLastRun(TaskKnowledgeHealth, time.Now().Add(-3*time.Hour))

	result := d.RunPeriodicKnowledgeHealth()
	if result == nil {
		t.Fatal("RunPeriodicKnowledgeHealth() should return result when due")
	}
	if !called {
		t.Error("KnowledgeHealth.Check should be called when due")
	}
	if result.TotalActive != 15 {
		t.Errorf("TotalActive = %d, want 15", result.TotalActive)
	}
	if d.Scheduler.LastRunTime(TaskKnowledgeHealth).IsZero() {
		t.Error("lastKnowledgeHealth should be updated after running")
	}
}

func TestDaemon_RunPeriodicKnowledgeHealth_Error(t *testing.T) {
	cfg := Config{
		KnowledgeHealthEnabled:  true,
		KnowledgeHealthInterval: 2 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		KnowledgeHealth: &mockKnowledgeHealthService{CheckFunc: func() (*KnowledgeHealthResult, error) {
			return nil, fmt.Errorf("kb quick list failed")
		}},
	}

	result := d.RunPeriodicKnowledgeHealth()
	if result == nil {
		t.Fatal("RunPeriodicKnowledgeHealth() should return result on error")
	}
	if result.Error == nil {
		t.Error("Result should have error")
	}
}

func TestDaemon_RunPeriodicKnowledgeHealth_ThresholdExceeded(t *testing.T) {
	issueCalled := false
	cfg := Config{
		KnowledgeHealthEnabled:   true,
		KnowledgeHealthInterval:  2 * time.Hour,
		KnowledgeHealthThreshold: 20,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		KnowledgeHealth: &mockKnowledgeHealthService{
			CheckFunc: func() (*KnowledgeHealthResult, error) {
				return &KnowledgeHealthResult{
					TotalActive: 50,
					ByType: map[string]int{
						"decision":   35,
						"constraint": 15,
					},
				}, nil
			},
			CreateIssueFunc: func(result *KnowledgeHealthResult) error {
				issueCalled = true
				return nil
			},
		},
	}

	result := d.RunPeriodicKnowledgeHealth()
	if result == nil {
		t.Fatal("RunPeriodicKnowledgeHealth() should return result")
	}
	if !result.ThresholdExceeded {
		t.Error("ThresholdExceeded should be true when total exceeds threshold")
	}
	if !issueCalled {
		t.Error("KnowledgeHealth.CreateIssue should be called when threshold exceeded")
	}
}

func TestDaemon_RunPeriodicKnowledgeHealth_ThresholdNotExceeded(t *testing.T) {
	issueCalled := false
	cfg := Config{
		KnowledgeHealthEnabled:   true,
		KnowledgeHealthInterval:  2 * time.Hour,
		KnowledgeHealthThreshold: 50,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		KnowledgeHealth: &mockKnowledgeHealthService{
			CheckFunc: func() (*KnowledgeHealthResult, error) {
				return &KnowledgeHealthResult{
					TotalActive: 15,
					ByType: map[string]int{
						"decision":   10,
						"constraint": 5,
					},
				}, nil
			},
			CreateIssueFunc: func(result *KnowledgeHealthResult) error {
				issueCalled = true
				return nil
			},
		},
	}

	result := d.RunPeriodicKnowledgeHealth()
	if result == nil {
		t.Fatal("RunPeriodicKnowledgeHealth() should return result")
	}
	if result.ThresholdExceeded {
		t.Error("ThresholdExceeded should be false when total below threshold")
	}
	if issueCalled {
		t.Error("KnowledgeHealth.CreateIssue should NOT be called when threshold not exceeded")
	}
}

func TestDefaultConfig_IncludesKnowledgeHealth(t *testing.T) {
	config := DefaultConfig()

	if !config.KnowledgeHealthEnabled {
		t.Error("DefaultConfig().KnowledgeHealthEnabled should be true")
	}
	if config.KnowledgeHealthInterval != 2*time.Hour {
		t.Errorf("DefaultConfig().KnowledgeHealthInterval = %v, want 2h", config.KnowledgeHealthInterval)
	}
	if config.KnowledgeHealthThreshold != 50 {
		t.Errorf("DefaultConfig().KnowledgeHealthThreshold = %d, want 50", config.KnowledgeHealthThreshold)
	}
}

func TestKnowledgeHealthResult_Snapshot(t *testing.T) {
	result := &KnowledgeHealthResult{
		TotalActive: 30,
		ByType: map[string]int{
			"decision":   20,
			"constraint": 10,
		},
		ThresholdExceeded: true,
	}

	snapshot := result.Snapshot()
	if snapshot.TotalActive != 30 {
		t.Errorf("Snapshot.TotalActive = %d, want 30", snapshot.TotalActive)
	}
	if snapshot.Decisions != 20 {
		t.Errorf("Snapshot.Decisions = %d, want 20", snapshot.Decisions)
	}
	if snapshot.Constraints != 10 {
		t.Errorf("Snapshot.Constraints = %d, want 10", snapshot.Constraints)
	}
	if !snapshot.ThresholdExceeded {
		t.Error("Snapshot.ThresholdExceeded should be true")
	}
}
