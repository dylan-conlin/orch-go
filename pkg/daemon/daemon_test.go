// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"testing"
)

func TestNextIssue_EmptyQueue(t *testing.T) {
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{}, nil
		},
	}

	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() unexpected error: %v", err)
	}
	if issue != nil {
		t.Errorf("NextIssue() expected nil for empty queue, got %v", issue)
	}
}

func TestNextIssue_ReturnsHighestPriority(t *testing.T) {
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-3", Title: "Low priority", Priority: 2, IssueType: "feature"},
				{ID: "proj-1", Title: "High priority", Priority: 0, IssueType: "bug"},
				{ID: "proj-2", Title: "Medium priority", Priority: 1, IssueType: "task"},
			}, nil
		},
	}

	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssue() expected issue, got nil")
	}
	if issue.ID != "proj-1" {
		t.Errorf("NextIssue() = %q, want highest priority 'proj-1'", issue.ID)
	}
}

func TestNextIssue_SkipsNonSpawnableTypes(t *testing.T) {
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Epic", Priority: 0, IssueType: "epic"},
				{ID: "proj-2", Title: "Feature", Priority: 1, IssueType: "feature"},
			}, nil
		},
	}

	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssue() expected issue, got nil")
	}
	if issue.ID != "proj-2" {
		t.Errorf("NextIssue() = %q, want 'proj-2' (skipping non-spawnable epic)", issue.ID)
	}
}

func TestNextIssue_SkipsBlockedIssues(t *testing.T) {
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Blocked", Priority: 0, IssueType: "feature", Status: "blocked"},
				{ID: "proj-2", Title: "Open", Priority: 1, IssueType: "feature", Status: "open"},
			}, nil
		},
	}

	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssue() expected issue, got nil")
	}
	if issue.ID != "proj-2" {
		t.Errorf("NextIssue() = %q, want 'proj-2' (skipping blocked)", issue.ID)
	}
}

func TestNextIssue_IncludesInProgressIssues(t *testing.T) {
	// This test verifies that in_progress issues are included in processing.
	// The daemon should use bd ready which returns both open and in_progress issues.
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "In Progress", Priority: 0, IssueType: "feature", Status: "in_progress", Labels: []string{"triage:ready"}},
				{ID: "proj-2", Title: "Open", Priority: 1, IssueType: "feature", Status: "open", Labels: []string{"triage:ready"}},
			}, nil
		},
		Config: Config{Label: "triage:ready"},
	}

	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssue() expected issue, got nil")
	}
	// Should return the in_progress issue since it has higher priority (0 vs 1)
	if issue.ID != "proj-1" {
		t.Errorf("NextIssue() = %q, want 'proj-1' (in_progress with higher priority)", issue.ID)
	}
	if issue.Status != "in_progress" {
		t.Errorf("NextIssue() status = %q, want 'in_progress'", issue.Status)
	}
}

func TestIsSpawnableType(t *testing.T) {
	tests := []struct {
		issueType string
		want      bool
	}{
		{"bug", true},
		{"feature", true},
		{"task", true},
		{"investigation", true},
		{"epic", false},
		{"chore", false},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.issueType, func(t *testing.T) {
			got := IsSpawnableType(tt.issueType)
			if got != tt.want {
				t.Errorf("IsSpawnableType(%q) = %v, want %v", tt.issueType, got, tt.want)
			}
		})
	}
}

func TestFormatPreview(t *testing.T) {
	issue := &Issue{
		ID:          "proj-123",
		Title:       "Fix critical bug",
		Priority:    0,
		IssueType:   "bug",
		Status:      "open",
		Description: "This is a detailed description",
	}

	preview := FormatPreview(issue)

	// Check that key information is present
	if preview == "" {
		t.Error("FormatPreview() returned empty string")
	}
	if !contains(preview, "proj-123") {
		t.Error("FormatPreview() missing issue ID")
	}
	if !contains(preview, "Fix critical bug") {
		t.Error("FormatPreview() missing title")
	}
	if !contains(preview, "bug") {
		t.Error("FormatPreview() missing issue type")
	}
}

func TestDaemon_Preview_NoIssues(t *testing.T) {
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{}, nil
		},
	}

	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}
	if result.Issue != nil {
		t.Errorf("Preview() expected nil issue for empty queue, got %v", result.Issue)
	}
	if result.Message == "" {
		t.Error("Preview() expected message for empty queue")
	}
}

func TestDaemon_Preview_HasIssues(t *testing.T) {
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test issue", Priority: 1, IssueType: "feature", Status: "open"},
			}, nil
		},
	}

	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}
	if result.Issue == nil {
		t.Fatal("Preview() expected issue, got nil")
	}
	if result.Issue.ID != "proj-1" {
		t.Errorf("Preview() issue ID = %q, want 'proj-1'", result.Issue.ID)
	}
	if result.Skill == "" {
		t.Error("Preview() expected skill to be inferred")
	}
}

func TestInferSkill(t *testing.T) {
	tests := []struct {
		issueType string
		wantSkill string
		wantErr   bool
	}{
		{"bug", "systematic-debugging", false},
		{"feature", "feature-impl", false},
		{"task", "feature-impl", false},
		{"investigation", "investigation", false},
		{"epic", "", true},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.issueType, func(t *testing.T) {
			got, err := InferSkill(tt.issueType)
			if (err != nil) != tt.wantErr {
				t.Errorf("InferSkill(%q) error = %v, wantErr %v", tt.issueType, err, tt.wantErr)
				return
			}
			if got != tt.wantSkill {
				t.Errorf("InferSkill(%q) = %q, want %q", tt.issueType, got, tt.wantSkill)
			}
		})
	}
}

func TestDaemon_Once_NoIssues(t *testing.T) {
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{}, nil
		},
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
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string) error {
			spawnCalled = true
			if beadsID != "proj-1" {
				t.Errorf("spawnFunc called with %q, want 'proj-1'", beadsID)
			}
			return nil
		},
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

func TestDaemon_Run_EmptyQueue(t *testing.T) {
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{}, nil
		},
	}

	results, err := d.Run(10) // Max 10 iterations
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
		listIssuesFunc: func() ([]Issue, error) {
			// Return remaining issues
			if callCount >= len(issues) {
				return []Issue{}, nil
			}
			remaining := issues[callCount:]
			return remaining, nil
		},
		spawnFunc: func(beadsID string) error {
			callCount++
			return nil
		},
	}

	results, err := d.Run(10) // Max 10 iterations
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
	// Infinite queue (always returns same issue)
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Infinite", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string) error {
			callCount++
			return nil
		},
	}

	results, err := d.Run(5) // Max 5 iterations
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

// Tests for new features: label filtering, capacity, config

func TestIssue_HasLabel(t *testing.T) {
	tests := []struct {
		name   string
		labels []string
		query  string
		want   bool
	}{
		{"has exact label", []string{"triage:ready", "P1"}, "triage:ready", true},
		{"has label case insensitive", []string{"TRIAGE:ready", "P1"}, "triage:ready", true},
		{"does not have label", []string{"P1", "P2"}, "triage:ready", false},
		{"empty labels", []string{}, "triage:ready", false},
		{"nil labels", nil, "triage:ready", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := &Issue{Labels: tt.labels}
			got := issue.HasLabel(tt.query)
			if got != tt.want {
				t.Errorf("Issue.HasLabel(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestNextIssue_FiltersbyLabel(t *testing.T) {
	config := Config{Label: "triage:ready"}
	d := &Daemon{
		Config: config,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "No label", Priority: 0, IssueType: "feature", Labels: []string{}},
				{ID: "proj-2", Title: "With label", Priority: 1, IssueType: "feature", Labels: []string{"triage:ready"}},
			}, nil
		},
	}

	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssue() expected issue, got nil")
	}
	if issue.ID != "proj-2" {
		t.Errorf("NextIssue() = %q, want 'proj-2' (with triage:ready label)", issue.ID)
	}
}

func TestNextIssue_NoLabelFilter(t *testing.T) {
	// Empty label means no filtering
	config := Config{Label: ""}
	d := &Daemon{
		Config: config,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "No label", Priority: 0, IssueType: "feature", Labels: []string{}},
				{ID: "proj-2", Title: "With label", Priority: 1, IssueType: "feature", Labels: []string{"triage:ready"}},
			}, nil
		},
	}

	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssue() expected issue, got nil")
	}
	// Should return highest priority regardless of labels
	if issue.ID != "proj-1" {
		t.Errorf("NextIssue() = %q, want 'proj-1' (no label filter)", issue.ID)
	}
}

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
			d := &Daemon{
				Config:          Config{MaxAgents: tt.maxAgents},
				activeCountFunc: tt.activeFunc,
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
		{"no limit", 0, func() int { return 100 }, 100}, // Returns high number when no limit
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Daemon{
				Config:          Config{MaxAgents: tt.maxAgents},
				activeCountFunc: tt.activeFunc,
			}
			got := d.AvailableSlots()
			if got != tt.want {
				t.Errorf("AvailableSlots() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Check sensible defaults
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
		MaxAgents: 0, // No limit
	}
	d := NewWithConfig(config)

	if d.Pool != nil {
		t.Error("NewWithConfig() should not create pool when MaxAgents = 0")
	}
}

func TestNewWithPool(t *testing.T) {
	pool := NewWorkerPool(5)
	config := Config{
		MaxAgents: 10, // This should be ignored when pool is provided
	}
	d := NewWithPool(config, pool)

	if d.Pool != pool {
		t.Error("NewWithPool() should use provided pool")
	}
	// The pool's max should be 5, not 10
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

	// Acquire slots
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
	d := &Daemon{} // No pool

	status := d.PoolStatus()
	if status != nil {
		t.Error("PoolStatus() should be nil when no pool is configured")
	}
}

func TestDaemon_Once_WithPool_AcquiresSlot(t *testing.T) {
	pool := NewWorkerPool(2)
	spawnCount := 0
	d := &Daemon{
		Pool: pool,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string) error {
			spawnCount++
			return nil
		},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() error = %v", err)
	}
	if !result.Processed {
		t.Error("Once() expected Processed=true")
	}

	// Pool should have one active slot
	if pool.Active() != 1 {
		t.Errorf("Pool.Active() = %d, want 1", pool.Active())
	}
}

func TestDaemon_Once_WithPool_AtCapacity(t *testing.T) {
	pool := NewWorkerPool(1)
	// Fill the pool
	pool.TryAcquire()

	d := &Daemon{
		Pool: pool,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string) error {
			t.Error("spawnFunc should not be called when at capacity")
			return nil
		},
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
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string) error {
			return fmt.Errorf("spawn failed")
		},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() error = %v", err)
	}
	if result.Processed {
		t.Error("Once() expected Processed=false on spawn error")
	}

	// Pool should have released the slot on error
	if pool.Active() != 0 {
		t.Errorf("Pool.Active() = %d, want 0 (slot should be released on error)", pool.Active())
	}
}

func TestDaemon_OnceWithSlot_ReturnsSlot(t *testing.T) {
	pool := NewWorkerPool(2)
	d := &Daemon{
		Pool: pool,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string) error {
			return nil
		},
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

	// Release the slot
	d.ReleaseSlot(slot)
	if pool.Active() != 0 {
		t.Errorf("Pool.Active() = %d after release, want 0", pool.Active())
	}
}

func TestDaemon_OnceWithSlot_NoPool(t *testing.T) {
	d := &Daemon{
		Pool: nil, // No pool
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string) error {
			return nil
		},
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
