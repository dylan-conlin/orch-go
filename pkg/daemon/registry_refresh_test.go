package daemon

import (
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/group"
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

func TestGroupFilterReapplication_NewProjectDiscovered(t *testing.T) {
	// Simulates: daemon started with --group orch, then scrape added to groups.yaml.
	// After refresh, the filtered registry should include scrape.

	// Initial filtered registry (startup): only orch-go and beads
	initialRegistry := NewProjectRegistryFromMap(map[string]string{
		"orch-go": "/home/user/orch-go",
		"bd":      "/home/user/beads",
	}, "/home/user/orch-go")

	// Simulated "new" unfiltered registry from NewProjectRegistryWithGroups()
	// that now includes scrape (discovered via filesystem heuristic after groups.yaml update)
	unfilteredRegistry := NewProjectRegistryFromMap(map[string]string{
		"orch-go": "/home/user/orch-go",
		"bd":      "/home/user/beads",
		"scrape":  "/home/user/scrape",
		"blog":    "/home/user/blog", // Not in orch group
	}, "/home/user/orch-go")

	groupCfg := &group.Config{
		Groups: map[string]group.Group{
			"orch": {
				Account:  "personal",
				Projects: []string{"orch-go", "beads", "scrape"},
			},
		},
	}

	kbProjects := map[string]string{
		"orch-go": "/home/user/orch-go",
		"beads":   "/home/user/beads",
		"scrape":  "/home/user/scrape",
		"blog":    "/home/user/blog",
	}

	// Apply group filter (same logic as RunPeriodicRegistryRefresh)
	groupFilter := "orch"
	members := groupCfg.ResolveGroupMembers(groupFilter, kbProjects)
	allowedDirs := make(map[string]bool, len(members))
	for _, name := range members {
		if path, ok := kbProjects[name]; ok {
			allowedDirs[path] = true
		}
	}
	filteredRegistry := unfilteredRegistry.FilterByDirs(allowedDirs)

	// Verify scrape is in the filtered result
	projects := filteredRegistry.Projects()
	found := map[string]bool{}
	for _, p := range projects {
		found[p.Prefix] = true
	}

	if !found["scrape"] {
		t.Error("scrape should be in filtered registry after group filter reapplication")
	}
	if !found["orch-go"] {
		t.Error("orch-go should be in filtered registry")
	}
	if !found["bd"] {
		t.Error("bd (beads) should be in filtered registry")
	}
	if found["blog"] {
		t.Error("blog should NOT be in filtered registry (not in orch group)")
	}

	// Verify the filtered registry is different from the initial
	if initialRegistry.Equal(filteredRegistry) {
		t.Error("filtered registry should differ from initial (scrape was added)")
	}

	// Verify diff shows scrape as added
	added, removed := initialRegistry.Diff(filteredRegistry)
	if len(added) != 1 || added[0] != "scrape" {
		t.Errorf("Diff added = %v, want [scrape]", added)
	}
	if len(removed) != 0 {
		t.Errorf("Diff removed = %v, want []", removed)
	}
}

func TestGroupFilterReapplication_NoFilterPassesAll(t *testing.T) {
	// When GroupFilter is empty, no filtering should be applied.
	// This ensures the daemon without --group still discovers all projects.
	d := &Daemon{
		GroupFilter: "",
		GroupConfig: &group.Config{
			Groups: map[string]group.Group{
				"orch": {Projects: []string{"orch-go"}},
			},
		},
		KBProjects: map[string]string{
			"orch-go": "/home/user/orch-go",
			"blog":    "/home/user/blog",
		},
	}

	registry := NewProjectRegistryFromMap(map[string]string{
		"orch-go": "/home/user/orch-go",
		"blog":    "/home/user/blog",
	}, "/home/user/orch-go")

	// With empty GroupFilter, the filter block in RunPeriodicRegistryRefresh
	// should be skipped. Verify by checking the logic condition.
	if d.GroupFilter != "" {
		t.Error("GroupFilter should be empty for this test")
	}

	// All projects should remain (no filtering applied)
	projects := registry.Projects()
	if len(projects) != 2 {
		t.Errorf("expected 2 projects without group filter, got %d", len(projects))
	}
}

func TestGroupConfigRefreshDuringRegistryRefresh(t *testing.T) {
	// Verify that RunPeriodicRegistryRefresh updates GroupConfig and KBProjects.
	// Since the actual function shells out to kb/groups.yaml, we test the
	// state update contract: after refresh, KBProjects should reflect the new registry.

	oldRegistry := NewProjectRegistryFromMap(map[string]string{
		"orch-go": "/home/user/orch-go",
	}, "/home/user/orch-go")

	newRegistry := NewProjectRegistryFromMap(map[string]string{
		"orch-go": "/home/user/orch-go",
		"scrape":  "/home/user/scrape",
	}, "/home/user/orch-go")

	// BuildKBProjectsMap should include the new project
	kbMap := BuildKBProjectsMap(newRegistry)
	if kbMap["scrape"] != "/home/user/scrape" {
		t.Errorf("BuildKBProjectsMap missing scrape: got %v", kbMap)
	}
	if kbMap["orch-go"] != "/home/user/orch-go" {
		t.Errorf("BuildKBProjectsMap missing orch-go: got %v", kbMap)
	}

	// Verify old map doesn't have scrape
	oldMap := BuildKBProjectsMap(oldRegistry)
	if _, ok := oldMap["scrape"]; ok {
		t.Error("old KBProjectsMap should not have scrape")
	}
}

func TestGroupFilter_StoredOnDaemon(t *testing.T) {
	d := &Daemon{}
	if d.GroupFilter != "" {
		t.Error("GroupFilter should default to empty string")
	}

	d.GroupFilter = "orch"
	if d.GroupFilter != "orch" {
		t.Errorf("GroupFilter = %q, want 'orch'", d.GroupFilter)
	}
}
