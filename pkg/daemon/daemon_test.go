// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"testing"
)

func TestDaemon_Once_NoIssues(t *testing.T) {
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{}, nil
		}},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() unexpected error: %v", err)
	}
	if result.Processed {
		t.Error("Once() expected Processed=false for empty queue")
	}
	if result.Message == "" {
		t.Error("Once() expected message for empty queue")
	}
}

func TestDaemon_Once_ProcessesOneIssue(t *testing.T) {
	spawnCalled := false
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		}},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
			spawnCalled = true
			if beadsID != "proj-1" {
				t.Errorf("spawnFunc called with %q, want 'proj-1'", beadsID)
			}
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() unexpected error: %v", err)
	}
	if !result.Processed {
		t.Error("Once() expected Processed=true")
	}
	if !spawnCalled {
		t.Error("Once() expected spawnFunc to be called")
	}
	if result.Issue == nil || result.Issue.ID != "proj-1" {
		t.Error("Once() expected result.Issue to be proj-1")
	}
}

func TestDaemon_SpawnIssue_StatusUpdateFailureReleasesSlot(t *testing.T) {
	pool := NewWorkerPool(1)
	spawnCalled := false
	d := &Daemon{
		Pool: pool,
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
			spawnCalled = true
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return fmt.Errorf("update failed")
		}},
	}

	issue := &Issue{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"}
	result, slot, err := d.spawnIssue(issue, "feature-impl", "sonnet")
	if err != nil {
		t.Fatalf("spawnIssue() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("spawnIssue() expected result on status update failure")
	}
	if result.Processed {
		t.Error("spawnIssue() expected Processed=false on status update failure")
	}
	if result.Error == nil {
		t.Error("spawnIssue() expected Error to be set on status update failure")
	}
	if spawnCalled {
		t.Error("spawnIssue() should not call spawnFunc when status update fails")
	}
	if slot != nil {
		t.Error("spawnIssue() expected nil slot on status update failure")
	}
	if pool.Active() != 0 {
		t.Errorf("Pool.Active() = %d, want 0 (slot should be released on error)", pool.Active())
	}
}

func TestDaemon_Run_EmptyQueue(t *testing.T) {
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{}, nil
		}},
	}

	results, err := d.Run(10)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Run() expected 0 results for empty queue, got %d", len(results))
	}
}

func TestDaemon_Run_ProcessesAllIssues(t *testing.T) {
	callCount := 0
	issues := []Issue{
		{ID: "proj-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
		{ID: "proj-2", Title: "Second", Priority: 1, IssueType: "bug", Status: "open"},
		{ID: "proj-3", Title: "Third", Priority: 2, IssueType: "task", Status: "open"},
	}

	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			if callCount >= len(issues) {
				return []Issue{}, nil
			}
			remaining := issues[callCount:]
			return remaining, nil
		}},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
			callCount++
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	results, err := d.Run(10)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Run() expected 3 results, got %d", len(results))
	}
	if callCount != 3 {
		t.Errorf("Run() expected 3 spawn calls, got %d", callCount)
	}
}

func TestDaemon_Run_RespectsMaxIterations(t *testing.T) {
	callCount := 0
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Infinite", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		}},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
			callCount++
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	results, err := d.Run(5)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("Run() expected 5 results (max), got %d", len(results))
	}
	if callCount != 5 {
		t.Errorf("Run() expected 5 spawn calls (max), got %d", callCount)
	}
}

// Test helpers

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Tests for capacity management

func TestDaemon_AtCapacity(t *testing.T) {
	tests := []struct {
		name       string
		maxAgents  int
		activeFunc func() int
		want       bool
	}{
		{"below limit", 3, func() int { return 1 }, false},
		{"at limit", 3, func() int { return 3 }, true},
		{"above limit", 3, func() int { return 5 }, true},
		{"no limit (0)", 0, func() int { return 100 }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{MaxAgents: tt.maxAgents}
			d := NewWithConfig(config)
			if d.Pool != nil {
				activeCount := tt.activeFunc()
				for i := 0; i < activeCount; i++ {
					d.Pool.TryAcquire()
				}
			}
			got := d.AtCapacity()
			if got != tt.want {
				t.Errorf("AtCapacity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDaemon_AvailableSlots(t *testing.T) {
	tests := []struct {
		name       string
		maxAgents  int
		activeFunc func() int
		want       int
	}{
		{"none active", 3, func() int { return 0 }, 3},
		{"some active", 3, func() int { return 1 }, 2},
		{"at capacity", 3, func() int { return 3 }, 0},
		{"over capacity", 3, func() int { return 5 }, 0},
		{"no limit", 0, func() int { return 100 }, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{MaxAgents: tt.maxAgents}
			d := NewWithConfig(config)
			if d.Pool != nil {
				activeCount := tt.activeFunc()
				for i := 0; i < activeCount; i++ {
					d.Pool.TryAcquire()
				}
			}
			got := d.AvailableSlots()
			if got != tt.want {
				t.Errorf("AvailableSlots() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Tests for config/constructor

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.PollInterval <= 0 {
		t.Error("DefaultConfig() PollInterval should be positive")
	}
	if config.MaxAgents <= 0 {
		t.Error("DefaultConfig() MaxAgents should be positive")
	}
	if config.Label == "" {
		t.Error("DefaultConfig() Label should not be empty")
	}
	if config.SpawnDelay <= 0 {
		t.Error("DefaultConfig() SpawnDelay should be positive")
	}
}

func TestNewWithConfig(t *testing.T) {
	config := Config{
		MaxAgents: 5,
		Label:     "custom:label",
	}
	d := NewWithConfig(config)

	if d.Config.MaxAgents != 5 {
		t.Errorf("NewWithConfig() MaxAgents = %d, want 5", d.Config.MaxAgents)
	}
	if d.Config.Label != "custom:label" {
		t.Errorf("NewWithConfig() Label = %q, want 'custom:label'", d.Config.Label)
	}
}

// Tests for WorkerPool integration

func TestNewWithConfig_CreatesPool(t *testing.T) {
	config := Config{
		MaxAgents: 3,
	}
	d := NewWithConfig(config)

	if d.Pool == nil {
		t.Fatal("NewWithConfig() should create pool when MaxAgents > 0")
	}
	if d.Pool.MaxWorkers() != 3 {
		t.Errorf("Pool.MaxWorkers() = %d, want 3", d.Pool.MaxWorkers())
	}
}

func TestNewWithConfig_NoPoolWhenNoLimit(t *testing.T) {
	config := Config{
		MaxAgents: 0,
	}
	d := NewWithConfig(config)

	if d.Pool != nil {
		t.Error("NewWithConfig() should not create pool when MaxAgents = 0")
	}
}

func TestNewWithPool(t *testing.T) {
	pool := NewWorkerPool(5)
	config := Config{
		MaxAgents: 10,
	}
	d := NewWithPool(config, pool)

	if d.Pool != pool {
		t.Error("NewWithPool() should use provided pool")
	}
	if d.Pool.MaxWorkers() != 5 {
		t.Errorf("Pool.MaxWorkers() = %d, want 5 (from provided pool)", d.Pool.MaxWorkers())
	}
}

func TestDaemon_AtCapacity_WithPool(t *testing.T) {
	pool := NewWorkerPool(2)
	d := NewWithPool(Config{}, pool)

	if d.AtCapacity() {
		t.Error("AtCapacity() should be false when pool is empty")
	}

	slot1 := pool.TryAcquire()
	slot2 := pool.TryAcquire()

	if !d.AtCapacity() {
		t.Error("AtCapacity() should be true when pool is full")
	}

	pool.Release(slot1)
	if d.AtCapacity() {
		t.Error("AtCapacity() should be false after release")
	}
	pool.Release(slot2)
}

func TestDaemon_AvailableSlots_WithPool(t *testing.T) {
	pool := NewWorkerPool(3)
	d := NewWithPool(Config{}, pool)

	if d.AvailableSlots() != 3 {
		t.Errorf("AvailableSlots() = %d, want 3", d.AvailableSlots())
	}

	slot := pool.TryAcquire()
	if d.AvailableSlots() != 2 {
		t.Errorf("AvailableSlots() = %d, want 2", d.AvailableSlots())
	}
	pool.Release(slot)
}

func TestDaemon_ActiveCount_WithPool(t *testing.T) {
	pool := NewWorkerPool(3)
	d := NewWithPool(Config{}, pool)

	if d.ActiveCount() != 0 {
		t.Errorf("ActiveCount() = %d, want 0", d.ActiveCount())
	}

	slot := pool.TryAcquire()
	if d.ActiveCount() != 1 {
		t.Errorf("ActiveCount() = %d, want 1", d.ActiveCount())
	}
	pool.Release(slot)
}

func TestDaemon_PoolStatus(t *testing.T) {
	pool := NewWorkerPool(3)
	d := NewWithPool(Config{}, pool)

	status := d.PoolStatus()
	if status == nil {
		t.Fatal("PoolStatus() should not be nil when pool is configured")
	}
	if status.MaxWorkers != 3 {
		t.Errorf("PoolStatus().MaxWorkers = %d, want 3", status.MaxWorkers)
	}
}

func TestDaemon_PoolStatus_NilPool(t *testing.T) {
	d := &Daemon{}

	status := d.PoolStatus()
	if status != nil {
		t.Error("PoolStatus() should be nil when no pool is configured")
	}
}

func TestDaemon_Once_WithPool_AcquiresSlot(t *testing.T) {
	pool := NewWorkerPool(3)

	d := &Daemon{
		Pool: pool,
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		}},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() error = %v", err)
	}
	if !result.Processed {
		t.Error("Once() expected Processed=true")
	}

	if pool.Active() != 1 {
		t.Errorf("Pool.Active() = %d, want 1", pool.Active())
	}
}

func TestDaemon_Once_WithPool_AtCapacity(t *testing.T) {
	pool := NewWorkerPool(1)
	pool.TryAcquire()

	d := &Daemon{
		Pool: pool,
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		}},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
			t.Error("spawnFunc should not be called when at capacity")
			return nil
		}},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() error = %v", err)
	}
	if result.Processed {
		t.Error("Once() should not process when at capacity")
	}
	if result.Message != "At capacity - no slots available" {
		t.Errorf("Once() message = %q, want 'At capacity - no slots available'", result.Message)
	}
}

func TestDaemon_Once_WithPool_ReleasesSlotOnError(t *testing.T) {
	pool := NewWorkerPool(2)
	d := &Daemon{
		Pool: pool,
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		}},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
			return fmt.Errorf("spawn failed")
		}},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() error = %v", err)
	}
	if result.Processed {
		t.Error("Once() expected Processed=false on spawn error")
	}

	if pool.Active() != 0 {
		t.Errorf("Pool.Active() = %d, want 0 (slot should be released on error)", pool.Active())
	}
}

func TestDaemon_OnceWithSlot_ReturnsSlot(t *testing.T) {
	pool := NewWorkerPool(2)
	spawnCount := 0
	d := &Daemon{
		Pool: pool,
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		}},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
			spawnCount++
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	result, slot, err := d.OnceWithSlot()
	if err != nil {
		t.Fatalf("OnceWithSlot() error = %v", err)
	}
	if !result.Processed {
		t.Error("OnceWithSlot() expected Processed=true")
	}
	if slot == nil {
		t.Error("OnceWithSlot() should return slot")
	}
	if slot.BeadsID != "proj-1" {
		t.Errorf("Slot.BeadsID = %q, want 'proj-1'", slot.BeadsID)
	}

	d.ReleaseSlot(slot)
	if pool.Active() != 0 {
		t.Errorf("Pool.Active() = %d after release, want 0", pool.Active())
	}
}

func TestDaemon_OnceWithSlot_NoPool(t *testing.T) {
	d := &Daemon{
		Pool: nil,
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		}},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	result, slot, err := d.OnceWithSlot()
	if err != nil {
		t.Fatalf("OnceWithSlot() error = %v", err)
	}
	if !result.Processed {
		t.Error("OnceWithSlot() expected Processed=true")
	}
	if slot != nil {
		t.Error("OnceWithSlot() should return nil slot when no pool configured")
	}
}

func TestDaemon_ReleaseSlot_Nil(t *testing.T) {
	pool := NewWorkerPool(2)
	d := NewWithPool(Config{}, pool)

	// Should not panic
	d.ReleaseSlot(nil)
}

func TestDaemon_ReleaseSlot_NoPool(t *testing.T) {
	d := &Daemon{Pool: nil}

	// Should not panic
	d.ReleaseSlot(&Slot{ID: 1})
}

// =============================================================================
// Tests for ReconcileWithOpenCode
// =============================================================================

func TestDaemon_ReconcileWithOpenCode_NoPool(t *testing.T) {
	d := &Daemon{Pool: nil}

	freed := d.ReconcileWithOpenCode()
	if freed != 0 {
		t.Errorf("ReconcileWithOpenCode() = %d, want 0 (no pool)", freed)
	}
}

func TestDaemon_ReconcileWithOpenCode_WithPool(t *testing.T) {
	pool := NewWorkerPool(3)
	pool.TryAcquire()
	pool.TryAcquire()
	pool.TryAcquire()

	d := &Daemon{
		Pool: pool,
	}

	freed := d.ReconcileWithOpenCode()

	if freed < 0 || freed > 3 {
		t.Errorf("ReconcileWithOpenCode() freed = %d, want 0-3", freed)
	}

	if pool.Active()+freed != 3 {
		t.Errorf("Pool.Active() + freed = %d + %d = %d, want 3",
			pool.Active(), freed, pool.Active()+freed)
	}
}

// =============================================================================
// Tests for Fresh Status Check (TOCTOU race prevention)
// =============================================================================

func TestDaemon_Once_FreshStatusCheck_SkipsInProgressIssue(t *testing.T) {
	spawnCalled := false
	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
				}, nil
			},
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				return "in_progress", nil
			},
		},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
			spawnCalled = true
			return nil
		}},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() unexpected error: %v", err)
	}
	if result.Processed {
		t.Error("Once() should not process an in_progress issue")
	}
	if spawnCalled {
		t.Error("spawnFunc should not be called when fresh status check shows in_progress")
	}
	if result.Issue == nil || result.Issue.ID != "proj-1" {
		t.Error("result.Issue should still reference the skipped issue")
	}
}

func TestDaemon_Once_FreshStatusCheck_AllowsOpenIssue(t *testing.T) {
	spawnCalled := false
	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
				}, nil
			},
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				return "open", nil
			},
		},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
			spawnCalled = true
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() unexpected error: %v", err)
	}
	if !result.Processed {
		t.Error("Once() should process an open issue")
	}
	if !spawnCalled {
		t.Error("spawnFunc should be called when fresh status check confirms open")
	}
}

func TestDaemon_Once_FreshStatusCheck_FailOpenOnError(t *testing.T) {
	spawnCalled := false
	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
				}, nil
			},
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				return "", fmt.Errorf("beads daemon unavailable")
			},
		},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
			spawnCalled = true
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() unexpected error: %v", err)
	}
	if !result.Processed {
		t.Error("Once() should still process when fresh status check fails (fail-open)")
	}
	if !spawnCalled {
		t.Error("spawnFunc should be called when fresh status check errors (fail-open)")
	}
}

func TestDaemon_Once_FreshStatusCheck_NilFunc(t *testing.T) {
	spawnCalled := false
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		}},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
			spawnCalled = true
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() unexpected error: %v", err)
	}
	if !result.Processed {
		t.Error("Once() should process when Issues has no GetIssueStatusFunc")
	}
	if !spawnCalled {
		t.Error("spawnFunc should be called when Issues has no GetIssueStatusFunc")
	}
}

// =============================================================================
// Tests for Concurrent Daemon Dedup
// =============================================================================

func TestDaemon_ConcurrentDaemonDedup(t *testing.T) {
	issueStatus := "open"
	spawnCount := 0

	makeDaemon := func() *Daemon {
		return &Daemon{
			Issues: &mockIssueQuerier{
				ListReadyIssuesFunc: func() ([]Issue, error) {
					return []Issue{
						{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
					}, nil
				},
				GetIssueStatusFunc: func(beadsID string) (string, error) {
					return issueStatus, nil
				},
			},
			Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
				spawnCount++
				return nil
			}},
			StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
				issueStatus = status
				return nil
			}},
		}
	}

	d1 := makeDaemon()
	result1, err := d1.Once()
	if err != nil {
		t.Fatalf("Daemon 1 Once() unexpected error: %v", err)
	}
	if !result1.Processed {
		t.Error("Daemon 1 should have processed the issue")
	}

	issueStatus = "in_progress"

	d2 := makeDaemon()
	result2, err := d2.Once()
	if err != nil {
		t.Fatalf("Daemon 2 Once() unexpected error: %v", err)
	}
	if result2.Processed {
		t.Error("Daemon 2 should NOT have processed the issue (fresh status check should catch in_progress)")
	}

	if spawnCount != 1 {
		t.Errorf("Expected exactly 1 spawn, got %d", spawnCount)
	}
}

// =============================================================================
// Tests for Cross-Project Support
// =============================================================================

func TestDaemon_Once_CrossProject_UsesProjectDir(t *testing.T) {
	var capturedWorkdir string
	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{
						ID:         "bd-123",
						Title:      "Fix beads bug",
						Priority:   0,
						IssueType:  "bug",
						Status:     "open",
						ProjectDir: "/home/user/beads",
					},
				}, nil
			},
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				return "open", nil
			},
		},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
			capturedWorkdir = workdir
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() unexpected error: %v", err)
	}
	if !result.Processed {
		t.Errorf("Once() expected Processed=true, got message: %s", result.Message)
	}
	if capturedWorkdir != "/home/user/beads" {
		t.Errorf("spawnFunc workdir = %q, want '/home/user/beads'", capturedWorkdir)
	}
}

func TestDaemon_Once_LocalProject_NoWorkdir(t *testing.T) {
	var capturedWorkdir string
	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{
						ID:        "orch-go-456",
						Title:     "Add feature",
						Priority:  0,
						IssueType: "feature",
						Status:    "open",
					},
				}, nil
			},
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				return "open", nil
			},
		},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
			capturedWorkdir = workdir
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() unexpected error: %v", err)
	}
	if !result.Processed {
		t.Errorf("Once() expected Processed=true, got message: %s", result.Message)
	}
	if capturedWorkdir != "" {
		t.Errorf("spawnFunc workdir = %q, want empty (local project)", capturedWorkdir)
	}
}

func TestDaemon_resolveIssueQuerier_MockTakesPrecedence(t *testing.T) {
	mockCalled := false
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			mockCalled = true
			return []Issue{}, nil
		}},
		ProjectRegistry: &ProjectRegistry{
			prefixToDir: map[string]string{"bd": "/home/user/beads"},
			currentDir:  "/home/user/orch-go",
		},
	}

	q := d.resolveIssueQuerier()
	_, _ = q.ListReadyIssues()
	if !mockCalled {
		t.Error("resolveIssueQuerier should prefer explicit mock over ProjectRegistry")
	}
}

func TestDaemon_resolveIssueQuerier_NilFallsToDefault(t *testing.T) {
	d := &Daemon{}
	q := d.resolveIssueQuerier()
	if q == nil {
		t.Fatal("resolveIssueQuerier should not return nil")
	}
}

// =============================================================================
// Tests for Sticky Spawn Failure Fix
// =============================================================================

// TestOnceExcluding_NonErrorSkip_ContinuesToNextIssue verifies that when
// OnceExcluding returns a non-error skip for an issue (e.g., status already
// in_progress), adding that issue to the skip map and calling again processes
// the next issue in the queue. This is the core fix for sticky spawn failures:
// non-error dedup returns must be skippable so lower-priority issues get tried.
func TestOnceExcluding_NonErrorSkip_ContinuesToNextIssue(t *testing.T) {
	spawnedIDs := []string{}
	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "issue-A", Title: "High priority", Priority: 0, IssueType: "feature", Status: "open"},
					{ID: "issue-B", Title: "Lower priority", Priority: 1, IssueType: "task", Status: "open"},
				}, nil
			},
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				// Issue A is already in_progress (dedup case)
				if beadsID == "issue-A" {
					return "in_progress", nil
				}
				return "open", nil
			},
		},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
			spawnedIDs = append(spawnedIDs, beadsID)
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	// First call: issue-A should be skipped (non-error: status is in_progress)
	skip := make(map[string]bool)
	result1, err := d.OnceExcluding(skip)
	if err != nil {
		t.Fatalf("OnceExcluding() error: %v", err)
	}
	if result1.Processed {
		t.Fatal("OnceExcluding() should not process issue-A (status is in_progress)")
	}
	if result1.Issue == nil || result1.Issue.ID != "issue-A" {
		t.Fatalf("OnceExcluding() should return issue-A, got %v", result1.Issue)
	}
	if result1.Error != nil {
		t.Fatalf("OnceExcluding() non-error skip should have nil Error, got %v", result1.Error)
	}

	// Add issue-A to skip map (simulating what the daemon loop now does)
	skip[result1.Issue.ID] = true

	// Second call: issue-B should be tried and spawned
	result2, err := d.OnceExcluding(skip)
	if err != nil {
		t.Fatalf("OnceExcluding() second call error: %v", err)
	}
	if !result2.Processed {
		t.Fatalf("OnceExcluding() should process issue-B, got message: %s", result2.Message)
	}
	if result2.Issue == nil || result2.Issue.ID != "issue-B" {
		t.Fatalf("OnceExcluding() should spawn issue-B, got %v", result2.Issue)
	}
	if len(spawnedIDs) != 1 || spawnedIDs[0] != "issue-B" {
		t.Errorf("expected spawn of issue-B only, got %v", spawnedIDs)
	}
}

// TestOnceExcluding_SpawnFailure_RetriedWithFreshSkipMap verifies that issues
// that fail to spawn in one cycle are retried when called with a fresh skip map
// (simulating the start of a new poll cycle).
func TestOnceExcluding_SpawnFailure_RetriedWithFreshSkipMap(t *testing.T) {
	callCount := 0
	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "issue-1", Title: "Retry test", Priority: 0, IssueType: "feature", Status: "open"},
				}, nil
			},
		},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
			callCount++
			if callCount == 1 {
				return fmt.Errorf("transient spawn failure")
			}
			return nil // succeeds on retry
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	// Cycle 1: spawn fails
	skip1 := make(map[string]bool)
	result1, err := d.OnceExcluding(skip1)
	if err != nil {
		t.Fatalf("Cycle 1: OnceExcluding() error: %v", err)
	}
	if result1.Processed {
		t.Fatal("Cycle 1: should not process (spawn failed)")
	}
	if result1.Error == nil {
		t.Fatal("Cycle 1: should have Error set")
	}

	// Cycle 2: fresh skip map (simulates new poll cycle), should retry
	skip2 := make(map[string]bool)
	result2, err := d.OnceExcluding(skip2)
	if err != nil {
		t.Fatalf("Cycle 2: OnceExcluding() error: %v", err)
	}
	if !result2.Processed {
		t.Fatalf("Cycle 2: should process on retry, got message: %s", result2.Message)
	}
	if callCount != 2 {
		t.Errorf("expected 2 spawn calls (fail + retry), got %d", callCount)
	}
}
