package daemon

import (
	"testing"
	"time"
)

func TestRunPeriodicLearningRefresh_NotDue(t *testing.T) {
	cfg := Config{
		LearningRefreshEnabled:  true,
		LearningRefreshInterval: time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}
	// Mark as just run
	d.Scheduler.SetLastRun(TaskLearningRefresh, time.Now())

	result := d.RunPeriodicLearningRefresh()
	if result != nil {
		t.Error("expected nil when not due")
	}
}

func TestRunPeriodicLearningRefresh_Disabled(t *testing.T) {
	cfg := Config{
		LearningRefreshEnabled:  false,
		LearningRefreshInterval: time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}

	result := d.RunPeriodicLearningRefresh()
	if result != nil {
		t.Error("expected nil when disabled")
	}
}

func TestRunPeriodicLearningRefresh_Due_Runs(t *testing.T) {
	cfg := Config{
		LearningRefreshEnabled:  true,
		LearningRefreshInterval: time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}

	// Verify Learning is nil initially
	if d.Learning != nil {
		t.Error("expected Learning to be nil initially")
	}

	result := d.RunPeriodicLearningRefresh()
	if result == nil {
		t.Fatal("expected non-nil result when due")
	}
	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}

	// After refresh, Learning should be set (even if empty)
	if d.Learning == nil {
		t.Error("expected Learning to be set after refresh")
	}
}
