package daemon

import (
	"testing"
	"time"
)

func TestRunPeriodicRegistryRefresh_NotDue(t *testing.T) {
	cfg := Config{
		RegistryRefreshEnabled:  true,
		RegistryRefreshInterval: time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}
	// Mark as just run so it's not due
	d.Scheduler.MarkRun(TaskRegistryRefresh)

	result := d.RunPeriodicRegistryRefresh()
	if result != nil {
		t.Error("RunPeriodicRegistryRefresh should return nil when not due")
	}
}

func TestRunPeriodicRegistryRefresh_Disabled(t *testing.T) {
	cfg := Config{
		RegistryRefreshEnabled:  false,
		RegistryRefreshInterval: time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}

	result := d.RunPeriodicRegistryRefresh()
	if result != nil {
		t.Error("RunPeriodicRegistryRefresh should return nil when disabled")
	}
}

func TestRunPeriodicRegistryRefresh_NoChange(t *testing.T) {
	cfg := Config{
		RegistryRefreshEnabled:  true,
		RegistryRefreshInterval: time.Minute,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}

	// Set up identical registries - the refresh function will call
	// NewProjectRegistryWithGroups which shells out to kb projects list.
	// In test environment, we can't control that, so we test the
	// unchanged detection path by setting a registry first.
	d.ProjectRegistry = NewProjectRegistryFromMap(map[string]string{}, "/tmp/test")

	// The actual refresh will call kb projects list which may fail or
	// return different results in CI. We just verify the method runs
	// without panicking and returns a result.
	result := d.RunPeriodicRegistryRefresh()
	if result == nil {
		t.Fatal("RunPeriodicRegistryRefresh should return non-nil when due")
	}
	// Should either succeed or return an error (depending on kb availability)
	// but should never panic
}

func TestRegistryRefreshResult_Fields(t *testing.T) {
	r := &RegistryRefreshResult{
		Changed: true,
		Added:   []string{"new-project"},
		Removed: []string{"old-project"},
		Message: "Registry updated: +1 -1 projects",
	}

	if !r.Changed {
		t.Error("Changed should be true")
	}
	if len(r.Added) != 1 || r.Added[0] != "new-project" {
		t.Errorf("Added = %v, want [new-project]", r.Added)
	}
	if len(r.Removed) != 1 || r.Removed[0] != "old-project" {
		t.Errorf("Removed = %v, want [old-project]", r.Removed)
	}
}

func TestTaskRegistryRefresh_Constant(t *testing.T) {
	if TaskRegistryRefresh != "registry_refresh" {
		t.Errorf("TaskRegistryRefresh = %q, want 'registry_refresh'", TaskRegistryRefresh)
	}
}

func TestSchedulerRegistersRegistryRefresh(t *testing.T) {
	cfg := Config{
		RegistryRefreshEnabled:  true,
		RegistryRefreshInterval: 5 * time.Minute,
	}
	s := NewSchedulerFromConfig(cfg)

	// Should be due immediately (never run before)
	if !s.IsDue(TaskRegistryRefresh) {
		t.Error("TaskRegistryRefresh should be due immediately after registration")
	}

	// After marking, should not be due
	s.MarkRun(TaskRegistryRefresh)
	if s.IsDue(TaskRegistryRefresh) {
		t.Error("TaskRegistryRefresh should not be due immediately after marking")
	}
}
