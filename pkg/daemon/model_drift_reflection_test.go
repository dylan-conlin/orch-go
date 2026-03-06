package daemon

import (
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/modeldrift"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestDaemon_ShouldRunModelDriftReflection_Disabled(t *testing.T) {
	cfg := Config{
		ReflectModelDriftEnabled:  false,
		ReflectModelDriftInterval: time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}

	if d.ShouldRunModelDriftReflection() {
		t.Error("ShouldRunModelDriftReflection() should return false when disabled")
	}
}

func TestDaemon_RunPeriodicModelDriftReflection_NotDue(t *testing.T) {
	cfg := Config{
		ReflectModelDriftEnabled:  true,
		ReflectModelDriftInterval: time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		ModelDrift: &mockModelDriftStore{
			ReadStalenessEventsFunc: func(path string) ([]spawn.StalenessEvent, error) {
				return nil, nil
			},
		},
	}
	d.Scheduler.SetLastRun(TaskModelDriftReflect, time.Now())

	result := d.RunPeriodicModelDriftReflection()
	if result != nil {
		t.Error("RunPeriodicModelDriftReflection() should return nil when not due")
	}
}

func TestDaemon_RunPeriodicModelDriftReflection_Due(t *testing.T) {
	cfg := Config{
		ReflectModelDriftEnabled:  true,
		ReflectModelDriftInterval: time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		ModelDrift: &mockModelDriftStore{
			ReadStalenessEventsFunc: func(path string) ([]spawn.StalenessEvent, error) {
				return []spawn.StalenessEvent{}, nil
			},
		},
	}
	d.Scheduler.SetLastRun(TaskModelDriftReflect, time.Now().Add(-2*time.Hour))

	result := d.RunPeriodicModelDriftReflection()
	if result == nil {
		t.Fatal("RunPeriodicModelDriftReflection() should return result when due")
	}
	if d.Scheduler.LastRunTime(TaskModelDriftReflect).IsZero() {
		t.Error("lastModelDriftReflect should be updated after running")
	}
}

// Ensure mockModelDriftStore satisfies modeldrift.Store at compile time.
var _ modeldrift.Store = (*mockModelDriftStore)(nil)
