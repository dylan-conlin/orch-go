package daemon

import (
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestDaemon_ShouldRunModelDriftReflection_Disabled(t *testing.T) {
	d := &Daemon{
		Config: Config{
			ReflectModelDriftEnabled:  false,
			ReflectModelDriftInterval: time.Hour,
		},
	}

	if d.ShouldRunModelDriftReflection() {
		t.Error("ShouldRunModelDriftReflection() should return false when disabled")
	}
}

func TestDaemon_RunPeriodicModelDriftReflection_NotDue(t *testing.T) {
	d := &Daemon{
		Config: Config{
			ReflectModelDriftEnabled:  true,
			ReflectModelDriftInterval: time.Hour,
		},
		lastModelDriftReflect: time.Now(),
		ModelDrift: &mockModelDriftStore{
			ReadStalenessEventsFunc: func(path string) ([]spawn.StalenessEvent, error) {
				return nil, nil
			},
		},
	}

	result := d.RunPeriodicModelDriftReflection()
	if result != nil {
		t.Error("RunPeriodicModelDriftReflection() should return nil when not due")
	}
}

func TestDaemon_RunPeriodicModelDriftReflection_Due(t *testing.T) {
	d := &Daemon{
		Config: Config{
			ReflectModelDriftEnabled:  true,
			ReflectModelDriftInterval: time.Hour,
		},
		lastModelDriftReflect: time.Now().Add(-2 * time.Hour),
		ModelDrift: &mockModelDriftStore{
			ReadStalenessEventsFunc: func(path string) ([]spawn.StalenessEvent, error) {
				return []spawn.StalenessEvent{}, nil
			},
		},
	}

	result := d.RunPeriodicModelDriftReflection()
	if result == nil {
		t.Fatal("RunPeriodicModelDriftReflection() should return result when due")
	}
	if d.lastModelDriftReflect.IsZero() {
		t.Error("lastModelDriftReflect should be updated after running")
	}
}
