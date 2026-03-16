package daemon

import (
	"testing"
	"time"
)

func TestRunPeriodicInvestigationOrphan_NotDue(t *testing.T) {
	d := NewWithConfig(DefaultConfig())
	// Mark as just run so it's not due
	d.Scheduler.MarkRun(TaskInvestigationOrphan)

	result := d.RunPeriodicInvestigationOrphan()
	if result != nil {
		t.Fatal("expected nil when not due")
	}
}

func TestRunPeriodicInvestigationOrphan_Disabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.InvestigationOrphanEnabled = false
	d := NewWithConfig(cfg)

	result := d.RunPeriodicInvestigationOrphan()
	if result != nil {
		t.Fatal("expected nil when disabled")
	}
}

func TestIsInvestigation(t *testing.T) {
	tests := []struct {
		name     string
		issue    Issue
		expected bool
	}{
		{
			name:     "investigation by type",
			issue:    Issue{IssueType: "investigation"},
			expected: true,
		},
		{
			name:     "investigation by label",
			issue:    Issue{IssueType: "task", Labels: []string{"skill:investigation"}},
			expected: true,
		},
		{
			name:     "not investigation",
			issue:    Issue{IssueType: "feature", Labels: []string{"triage:ready"}},
			expected: false,
		},
		{
			name:     "investigation label case insensitive",
			issue:    Issue{IssueType: "task", Labels: []string{"Skill:Investigation"}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInvestigation(tt.issue)
			if got != tt.expected {
				t.Errorf("isInvestigation() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDeduplicateIssues(t *testing.T) {
	a := []Issue{
		{ID: "orch-go-001", Title: "first"},
		{ID: "orch-go-002", Title: "second"},
	}
	b := []Issue{
		{ID: "orch-go-002", Title: "second duplicate"},
		{ID: "orch-go-003", Title: "third"},
	}

	result := deduplicateIssues(a, b)
	if len(result) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(result))
	}

	// Verify dedup kept the original from slice a
	for _, issue := range result {
		if issue.ID == "orch-go-002" && issue.Title != "second" {
			t.Error("dedup should keep the first occurrence")
		}
	}
}

func TestInvestigationOrphanSnapshot(t *testing.T) {
	result := &InvestigationOrphanResult{
		OrphanCount:  3,
		ScannedCount: 10,
	}

	snapshot := result.Snapshot()
	if snapshot.OrphanCount != 3 {
		t.Errorf("snapshot.OrphanCount = %d, want 3", snapshot.OrphanCount)
	}
	if snapshot.ScannedCount != 10 {
		t.Errorf("snapshot.ScannedCount = %d, want 10", snapshot.ScannedCount)
	}
	if snapshot.LastCheck.IsZero() {
		t.Error("snapshot.LastCheck should be set")
	}
}

func TestShouldRunInvestigationOrphan(t *testing.T) {
	cfg := DefaultConfig()
	cfg.InvestigationOrphanEnabled = true
	cfg.InvestigationOrphanInterval = time.Minute
	d := NewWithConfig(cfg)

	// Should be due on first run (never run before)
	if !d.ShouldRunInvestigationOrphan() {
		t.Fatal("should be due on first run")
	}

	// Mark as run, should no longer be due
	d.Scheduler.MarkRun(TaskInvestigationOrphan)
	if d.ShouldRunInvestigationOrphan() {
		t.Fatal("should not be due immediately after running")
	}

	// Set last run to past threshold
	d.Scheduler.SetLastRun(TaskInvestigationOrphan, time.Now().Add(-2*time.Minute))
	if !d.ShouldRunInvestigationOrphan() {
		t.Fatal("should be due after interval elapsed")
	}
}
