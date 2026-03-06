package daemon

import (
	"fmt"
	"testing"
	"time"
)

// --- ShouldRunOrphanDetection tests ---

func TestDaemon_ShouldRunOrphanDetection_Disabled(t *testing.T) {
	cfg := Config{
		OrphanDetectionEnabled:  false,
		OrphanDetectionInterval: 30 * time.Minute,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}
	if d.ShouldRunOrphanDetection() {
		t.Error("ShouldRunOrphanDetection() should return false when disabled")
	}
}

func TestDaemon_ShouldRunOrphanDetection_ZeroInterval(t *testing.T) {
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 0,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}
	if d.ShouldRunOrphanDetection() {
		t.Error("ShouldRunOrphanDetection() should return false when interval is 0")
	}
}

func TestDaemon_ShouldRunOrphanDetection_NeverRun(t *testing.T) {
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}
	if !d.ShouldRunOrphanDetection() {
		t.Error("ShouldRunOrphanDetection() should return true when never run before")
	}
}

func TestDaemon_ShouldRunOrphanDetection_IntervalElapsed(t *testing.T) {
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}
	d.Scheduler.SetLastRun(TaskOrphanDetection, time.Now().Add(-45*time.Minute))
	if !d.ShouldRunOrphanDetection() {
		t.Error("ShouldRunOrphanDetection() should return true when interval has elapsed")
	}
}

func TestDaemon_ShouldRunOrphanDetection_IntervalNotElapsed(t *testing.T) {
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}
	d.Scheduler.SetLastRun(TaskOrphanDetection, time.Now().Add(-15*time.Minute))
	if d.ShouldRunOrphanDetection() {
		t.Error("ShouldRunOrphanDetection() should return false when interval has not elapsed")
	}
}

// --- RunPeriodicOrphanDetection tests ---

func TestDaemon_RunPeriodicOrphanDetection_NotDue(t *testing.T) {
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}
	d.Scheduler.SetLastRun(TaskOrphanDetection, time.Now())
	result := d.RunPeriodicOrphanDetection()
	if result != nil {
		t.Error("RunPeriodicOrphanDetection() should return nil when not due")
	}
}

func TestDaemon_RunPeriodicOrphanDetection_GetAgentsError(t *testing.T) {
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return nil, fmt.Errorf("beads unavailable")
			},
		},
	}
	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result on error")
	}
	if result.Error == nil {
		t.Error("Should have error")
	}
}

func TestDaemon_RunPeriodicOrphanDetection_DetectsOrphan(t *testing.T) {
	resetCalled := map[string]bool{}
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:        cfg,
		Scheduler:     NewSchedulerFromConfig(cfg),
		SpawnedIssues: NewSpawnedIssueTracker(),
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{
						BeadsID:   "orphan-001",
						Phase:     "Planning",
						UpdatedAt: time.Now().Add(-2 * time.Hour),
						Title:     "Orphaned task",
					},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool {
				return false
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error {
				resetCalled[beadsID] = true
				if status != "open" {
					t.Errorf("Expected status 'open', got '%s'", status)
				}
				return nil
			},
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}
	if result.ResetCount != 1 {
		t.Errorf("ResetCount = %d, want 1", result.ResetCount)
	}
	if !resetCalled["orphan-001"] {
		t.Error("Should have reset orphan-001")
	}
	if len(result.Orphans) != 1 {
		t.Errorf("Orphans count = %d, want 1", len(result.Orphans))
	}
	if result.Orphans[0].BeadsID != "orphan-001" {
		t.Errorf("Orphan BeadsID = %s, want orphan-001", result.Orphans[0].BeadsID)
	}
}

func TestDaemon_RunPeriodicOrphanDetection_SkipsActiveAgent(t *testing.T) {
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "active-001", Phase: "Implementing", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Active task"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return true },
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}
	if result.ResetCount != 0 {
		t.Errorf("ResetCount = %d, want 0 (agent has session)", result.ResetCount)
	}
	if result.SkippedCount != 1 {
		t.Errorf("SkippedCount = %d, want 1", result.SkippedCount)
	}
}

func TestDaemon_RunPeriodicOrphanDetection_SkipsTooNew(t *testing.T) {
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "new-001", Phase: "Planning", UpdatedAt: time.Now().Add(-30 * time.Minute), Title: "New task"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return false },
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}
	if result.ResetCount != 0 {
		t.Errorf("ResetCount = %d, want 0 (too new)", result.ResetCount)
	}
}

func TestDaemon_RunPeriodicOrphanDetection_SkipsPhaseComplete(t *testing.T) {
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "complete-001", Phase: "Complete", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Completed task"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return false },
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}
	if result.ResetCount != 0 {
		t.Errorf("ResetCount = %d, want 0 (Phase: Complete)", result.ResetCount)
	}
}

func TestDaemon_RunPeriodicOrphanDetection_UpdateStatusError(t *testing.T) {
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "orphan-001", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Orphaned task"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return false },
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error {
				return fmt.Errorf("beads update failed")
			},
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}
	if result.ResetCount != 0 {
		t.Errorf("ResetCount = %d, want 0 (update failed)", result.ResetCount)
	}
	if result.SkippedCount != 1 {
		t.Errorf("SkippedCount = %d, want 1", result.SkippedCount)
	}
}

func TestDaemon_RunPeriodicOrphanDetection_MultipleAgentsMixed(t *testing.T) {
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:        cfg,
		Scheduler:     NewSchedulerFromConfig(cfg),
		SpawnedIssues: NewSpawnedIssueTracker(),
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "orphan-1", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Orphan 1"},
					{BeadsID: "active-1", Phase: "Implementing", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Active 1"},
					{BeadsID: "new-1", Phase: "Planning", UpdatedAt: time.Now().Add(-30 * time.Minute), Title: "New 1"},
					{BeadsID: "orphan-2", Phase: "Testing", UpdatedAt: time.Now().Add(-3 * time.Hour), Title: "Orphan 2"},
					{BeadsID: "complete-1", Phase: "Complete", UpdatedAt: time.Now().Add(-5 * time.Hour), Title: "Complete 1"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return beadsID == "active-1" },
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error { return nil },
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}
	if result.ResetCount != 2 {
		t.Errorf("ResetCount = %d, want 2 (orphan-1, orphan-2)", result.ResetCount)
	}
	if result.SkippedCount != 3 {
		t.Errorf("SkippedCount = %d, want 3 (active-1, new-1, complete-1)", result.SkippedCount)
	}
}

func TestDaemon_RunPeriodicOrphanDetection_EmptyBeadsID(t *testing.T) {
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "No ID"},
				}, nil
			},
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}
	if result.ResetCount != 0 {
		t.Errorf("ResetCount = %d, want 0 (no beads ID)", result.ResetCount)
	}
	if result.SkippedCount != 1 {
		t.Errorf("SkippedCount = %d, want 1", result.SkippedCount)
	}
}

func TestDaemon_RunPeriodicOrphanDetection_NoAgents(t *testing.T) {
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) { return nil, nil },
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}
	if result.ResetCount != 0 {
		t.Errorf("ResetCount = %d, want 0", result.ResetCount)
	}
	if result.SkippedCount != 0 {
		t.Errorf("SkippedCount = %d, want 0", result.SkippedCount)
	}
}

func TestDaemon_RunPeriodicOrphanDetection_RetainsSpawnCacheEntry(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	tracker.MarkSpawned("orphan-001")
	if !tracker.IsSpawned("orphan-001") {
		t.Fatal("Issue should be in tracker before orphan detection")
	}

	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:        cfg,
		Scheduler:     NewSchedulerFromConfig(cfg),
		SpawnedIssues: tracker,
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "orphan-001", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Orphan"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return false },
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error { return nil },
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil || result.ResetCount != 1 {
		t.Fatal("Should have reset 1 orphan")
	}
	if !tracker.IsSpawned("orphan-001") {
		t.Error("Orphan-001 should STILL be in SpawnedIssues tracker (cooldown prevents immediate respawn)")
	}
}

func TestDaemon_RunPeriodicOrphanDetection_FailClosedOnSessionCheckError(t *testing.T) {
	resetCalled := false
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "running-agent-001", Phase: "Implementing", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Running task with infrastructure down"},
				}, nil
			},
			HasExistingSessionOrErrorFunc: func(beadsID string) (bool, error) {
				return false, fmt.Errorf("opencode session check failed: connection refused")
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error {
				resetCalled = true
				return nil
			},
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}
	if resetCalled {
		t.Error("Should NOT have called UpdateStatus — fail-closed means we skip on session check errors")
	}
	if result.ResetCount != 0 {
		t.Errorf("ResetCount = %d, want 0 (session check errored, should skip)", result.ResetCount)
	}
	if result.SkippedCount != 1 {
		t.Errorf("SkippedCount = %d, want 1 (errored agent should be counted as skipped)", result.SkippedCount)
	}
}

func TestDaemon_RunPeriodicOrphanDetection_MixedErrorAndOrphan(t *testing.T) {
	resetIDs := map[string]bool{}
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:        cfg,
		Scheduler:     NewSchedulerFromConfig(cfg),
		SpawnedIssues: NewSpawnedIssueTracker(),
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "errored-001", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Errored agent"},
					{BeadsID: "orphan-001", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Real orphan"},
					{BeadsID: "active-001", Phase: "Implementing", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Active agent"},
				}, nil
			},
			HasExistingSessionOrErrorFunc: func(beadsID string) (bool, error) {
				switch beadsID {
				case "errored-001":
					return false, fmt.Errorf("tmux session check failed: command not found")
				case "orphan-001":
					return false, nil
				case "active-001":
					return true, nil
				}
				return false, nil
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error {
				resetIDs[beadsID] = true
				return nil
			},
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}
	if result.ResetCount != 1 {
		t.Errorf("ResetCount = %d, want 1 (only orphan-001)", result.ResetCount)
	}
	if result.SkippedCount != 2 {
		t.Errorf("SkippedCount = %d, want 2 (errored-001 + active-001)", result.SkippedCount)
	}
	if resetIDs["errored-001"] {
		t.Error("Should NOT reset errored-001 — session check errored, fail-closed")
	}
	if !resetIDs["orphan-001"] {
		t.Error("Should reset orphan-001 — confirmed orphan (no session, no error)")
	}
	if resetIDs["active-001"] {
		t.Error("Should NOT reset active-001 — has active session")
	}
}

func TestDefaultConfig_IncludesOrphanDetection(t *testing.T) {
	config := DefaultConfig()
	if !config.OrphanDetectionEnabled {
		t.Error("OrphanDetectionEnabled should be true by default")
	}
	if config.OrphanDetectionInterval != 30*time.Minute {
		t.Errorf("OrphanDetectionInterval = %v, want 30m", config.OrphanDetectionInterval)
	}
	if config.OrphanAgeThreshold != time.Hour {
		t.Errorf("OrphanAgeThreshold = %v, want 1h", config.OrphanAgeThreshold)
	}
}

func TestLastOrphanDetectionTime_InitiallyZero(t *testing.T) {
	d := New()
	if !d.LastOrphanDetectionTime().IsZero() {
		t.Error("LastOrphanDetectionTime should be zero initially")
	}
}

func TestNextOrphanDetectionTime_DisabledWhenOff(t *testing.T) {
	cfg := Config{OrphanDetectionEnabled: false}
	d := &Daemon{Config: cfg, Scheduler: NewSchedulerFromConfig(cfg)}
	if !d.NextOrphanDetectionTime().IsZero() {
		t.Error("NextOrphanDetectionTime should be zero when disabled")
	}
}

func TestNextOrphanDetectionTime_ImmediateWhenNeverRun(t *testing.T) {
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
	}
	d := &Daemon{Config: cfg, Scheduler: NewSchedulerFromConfig(cfg)}
	next := d.NextOrphanDetectionTime()
	if time.Until(next) > 5*time.Second {
		t.Error("NextOrphanDetectionTime should be immediate when never run")
	}
}

func TestOrphanDetectionResult_Snapshot(t *testing.T) {
	result := &OrphanDetectionResult{
		ResetCount:   2,
		SkippedCount: 5,
	}
	snapshot := result.Snapshot()
	if snapshot.ResetCount != 2 {
		t.Errorf("Snapshot.ResetCount = %d, want 2", snapshot.ResetCount)
	}
	if snapshot.SkippedCount != 5 {
		t.Errorf("Snapshot.SkippedCount = %d, want 5", snapshot.SkippedCount)
	}
}

func TestDaemon_RunPeriodicOrphanDetection_UpdatesTimestamp(t *testing.T) {
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) { return nil, nil },
		},
	}

	if !d.Scheduler.LastRunTime(TaskOrphanDetection).IsZero() {
		t.Error("lastOrphanDetection should be zero initially")
	}

	d.RunPeriodicOrphanDetection()

	if d.Scheduler.LastRunTime(TaskOrphanDetection).IsZero() {
		t.Error("lastOrphanDetection should be updated after running")
	}
}
