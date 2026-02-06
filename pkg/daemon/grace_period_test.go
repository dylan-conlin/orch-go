package daemon

import (
	"strings"
	"testing"
	"time"
)

func TestGracePeriod_SkipsNewlySeenIssue(t *testing.T) {
	d := &Daemon{
		Config: Config{
			GracePeriod: 5 * time.Second,
		},
		firstSeen: make(map[string]time.Time),
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "New Issue", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
	}

	// First call should skip the issue (grace period active)
	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() unexpected error: %v", err)
	}
	if issue != nil {
		t.Errorf("NextIssue() expected nil (grace period active), got %v", issue.ID)
	}

	// Verify issue was recorded in firstSeen
	if _, exists := d.firstSeen["proj-1"]; !exists {
		t.Error("Expected issue to be recorded in firstSeen")
	}
}

func TestGracePeriod_AllowsAfterExpiry(t *testing.T) {
	d := &Daemon{
		Config: Config{
			GracePeriod: 10 * time.Millisecond, // Very short for testing
		},
		firstSeen: make(map[string]time.Time),
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Issue", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
	}

	// First call records issue, skips due to grace period
	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("First NextIssue() unexpected error: %v", err)
	}
	if issue != nil {
		t.Error("First NextIssue() expected nil (grace period active)")
	}

	// Wait for grace period to expire
	time.Sleep(15 * time.Millisecond)

	// Second call should return the issue
	issue, err = d.NextIssue()
	if err != nil {
		t.Fatalf("Second NextIssue() unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("Second NextIssue() expected issue after grace period expired, got nil")
	}
	if issue.ID != "proj-1" {
		t.Errorf("Second NextIssue() = %q, want 'proj-1'", issue.ID)
	}
}

func TestGracePeriod_DisabledWhenZero(t *testing.T) {
	d := &Daemon{
		Config: Config{
			GracePeriod: 0, // Disabled
		},
		firstSeen: make(map[string]time.Time),
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Issue", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
	}

	// Should return immediately since grace period is disabled
	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssue() expected issue (grace period disabled), got nil")
	}
	if issue.ID != "proj-1" {
		t.Errorf("NextIssue() = %q, want 'proj-1'", issue.ID)
	}
}

func TestGracePeriod_SkipsFirstReturnsSecondPastGrace(t *testing.T) {
	// Test that if the first issue is in grace period but a second issue
	// has been seen before and is past grace period, the second one is returned.
	d := &Daemon{
		Config: Config{
			GracePeriod: 5 * time.Second,
		},
		firstSeen: map[string]time.Time{
			// proj-2 was seen 10 seconds ago (past grace period)
			"proj-2": time.Now().Add(-10 * time.Second),
		},
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "New Issue", Priority: 0, IssueType: "feature", Status: "open"},
				{ID: "proj-2", Title: "Old Issue", Priority: 1, IssueType: "feature", Status: "open"},
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
	// proj-1 is higher priority but in grace period, so proj-2 should be returned
	if issue.ID != "proj-2" {
		t.Errorf("NextIssue() = %q, want 'proj-2' (proj-1 in grace period)", issue.ID)
	}
}

func TestGracePeriod_Preview_ShowsGracePeriodRejection(t *testing.T) {
	d := &Daemon{
		Config: Config{
			GracePeriod: 5 * time.Second,
		},
		firstSeen: make(map[string]time.Time),
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "New Issue", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
	}

	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}

	// Issue should be rejected due to grace period
	if result.Issue != nil {
		t.Errorf("Preview() expected no spawnable issue, got %v", result.Issue.ID)
	}

	// Check rejection reason
	if len(result.RejectedIssues) != 1 {
		t.Fatalf("Preview() expected 1 rejected issue, got %d", len(result.RejectedIssues))
	}
	if !strings.Contains(result.RejectedIssues[0].Reason, "grace period") {
		t.Errorf("Preview() rejection reason = %q, want to contain 'grace period'", result.RejectedIssues[0].Reason)
	}
}

func TestRecordFirstSeen_OnlyRecordsOnce(t *testing.T) {
	d := &Daemon{
		firstSeen: make(map[string]time.Time),
	}

	// First call should record
	first := d.RecordFirstSeen("proj-1")
	if !first {
		t.Error("RecordFirstSeen() first call should return true")
	}

	firstTime := d.firstSeen["proj-1"]

	// Small delay
	time.Sleep(5 * time.Millisecond)

	// Second call should not overwrite
	second := d.RecordFirstSeen("proj-1")
	if second {
		t.Error("RecordFirstSeen() second call should return false")
	}

	if d.firstSeen["proj-1"] != firstTime {
		t.Error("RecordFirstSeen() should not overwrite existing timestamp")
	}
}

func TestInGracePeriod_ReturnsFalseWhenDisabled(t *testing.T) {
	d := &Daemon{
		Config:    Config{GracePeriod: 0},
		firstSeen: make(map[string]time.Time),
	}

	if d.InGracePeriod("proj-1") {
		t.Error("InGracePeriod() should return false when grace period is 0")
	}
}

func TestCleanFirstSeen_RemovesStaleEntries(t *testing.T) {
	d := &Daemon{
		firstSeen: map[string]time.Time{
			"proj-1": time.Now(),
			"proj-2": time.Now(),
			"proj-3": time.Now(),
		},
	}

	// Only proj-1 and proj-3 are still active
	active := map[string]bool{
		"proj-1": true,
		"proj-3": true,
	}

	d.CleanFirstSeen(active)

	if _, exists := d.firstSeen["proj-1"]; !exists {
		t.Error("CleanFirstSeen() should keep active entries")
	}
	if _, exists := d.firstSeen["proj-2"]; exists {
		t.Error("CleanFirstSeen() should remove inactive entries")
	}
	if _, exists := d.firstSeen["proj-3"]; !exists {
		t.Error("CleanFirstSeen() should keep active entries")
	}
}

func TestGracePeriod_OnceExcluding_SkipsDuringGrace(t *testing.T) {
	// Test that OnceExcluding also respects grace period
	d := &Daemon{
		Config: Config{
			GracePeriod: 5 * time.Second,
		},
		firstSeen: make(map[string]time.Time),
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Issue", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		spawnFunc: func(beadsID string) error {
			return nil
		},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if result.Processed {
		t.Error("OnceExcluding() expected not processed (grace period active)")
	}
}

func TestGracePeriod_DefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config.GracePeriod != 30*time.Second {
		t.Errorf("DefaultConfig().GracePeriod = %v, want 30s", config.GracePeriod)
	}
}
