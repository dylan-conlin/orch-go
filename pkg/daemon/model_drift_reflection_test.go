package daemon

import (
	"testing"
	"time"
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
	called := false
	d := &Daemon{
		Config: Config{
			ReflectModelDriftEnabled:  true,
			ReflectModelDriftInterval: time.Hour,
		},
		lastModelDriftReflect: time.Now(),
		modelDriftReflectFunc: func() (*ModelDriftResult, error) {
			called = true
			return &ModelDriftResult{Message: "ok"}, nil
		},
	}

	result := d.RunPeriodicModelDriftReflection()
	if result != nil {
		t.Error("RunPeriodicModelDriftReflection() should return nil when not due")
	}
	if called {
		t.Error("modelDriftReflectFunc should not be called when not due")
	}
}

func TestDaemon_RunPeriodicModelDriftReflection_Due(t *testing.T) {
	called := false
	d := &Daemon{
		Config: Config{
			ReflectModelDriftEnabled:  true,
			ReflectModelDriftInterval: time.Hour,
		},
		lastModelDriftReflect: time.Now().Add(-2 * time.Hour),
		modelDriftReflectFunc: func() (*ModelDriftResult, error) {
			called = true
			return &ModelDriftResult{Message: "ok"}, nil
		},
	}

	result := d.RunPeriodicModelDriftReflection()
	if result == nil {
		t.Fatal("RunPeriodicModelDriftReflection() should return result when due")
	}
	if !called {
		t.Error("modelDriftReflectFunc should be called when due")
	}
	if d.lastModelDriftReflect.IsZero() {
		t.Error("lastModelDriftReflect should be updated after running")
	}
}
