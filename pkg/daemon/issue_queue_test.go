package daemon

import (
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

func TestNextIssue_SkipsInProgressIssues(t *testing.T) {
	// This test verifies that in_progress issues are SKIPPED to prevent duplicate spawns.
	// Even though bd ready returns both open and in_progress issues, we only spawn for open ones.
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
	// Should skip in_progress and return the open issue
	if issue.ID != "proj-2" {
		t.Errorf("NextIssue() = %q, want 'proj-2' (should skip in_progress)", issue.ID)
	}
	if issue.Status != "open" {
		t.Errorf("NextIssue() status = %q, want 'open'", issue.Status)
	}
}

func TestNextIssueExcluding_SkipsExcludedIssues(t *testing.T) {
	// Test that NextIssueExcluding skips issues in the skip set.
	// This is critical for the daemon to skip issues that failed to spawn
	// (e.g., due to failure report gate) and continue with other issues.
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
				{ID: "proj-2", Title: "Second", Priority: 1, IssueType: "feature", Status: "open"},
				{ID: "proj-3", Title: "Third", Priority: 2, IssueType: "feature", Status: "open"},
			}, nil
		},
	}

	// Skip the first issue (simulating failure report gate blocked it)
	skip := map[string]bool{"proj-1": true}

	issue, err := d.NextIssueExcluding(skip)
	if err != nil {
		t.Fatalf("NextIssueExcluding() unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssueExcluding() expected issue, got nil")
	}
	// Should skip proj-1 and return proj-2
	if issue.ID != "proj-2" {
		t.Errorf("NextIssueExcluding() = %q, want 'proj-2' (should skip excluded issue)", issue.ID)
	}
}

func TestNextIssueExcluding_SkipsMultipleExcludedIssues(t *testing.T) {
	// Test that multiple excluded issues are all skipped.
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
				{ID: "proj-2", Title: "Second", Priority: 1, IssueType: "feature", Status: "open"},
				{ID: "proj-3", Title: "Third", Priority: 2, IssueType: "feature", Status: "open"},
			}, nil
		},
	}

	// Skip multiple issues
	skip := map[string]bool{"proj-1": true, "proj-2": true}

	issue, err := d.NextIssueExcluding(skip)
	if err != nil {
		t.Fatalf("NextIssueExcluding() unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssueExcluding() expected issue, got nil")
	}
	// Should skip proj-1 and proj-2, return proj-3
	if issue.ID != "proj-3" {
		t.Errorf("NextIssueExcluding() = %q, want 'proj-3' (should skip excluded issues)", issue.ID)
	}
}

func TestNextIssueExcluding_ReturnsNilWhenAllExcluded(t *testing.T) {
	// Test that NextIssueExcluding returns nil when all issues are excluded.
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
				{ID: "proj-2", Title: "Second", Priority: 1, IssueType: "feature", Status: "open"},
			}, nil
		},
	}

	// Skip all issues
	skip := map[string]bool{"proj-1": true, "proj-2": true}

	issue, err := d.NextIssueExcluding(skip)
	if err != nil {
		t.Fatalf("NextIssueExcluding() unexpected error: %v", err)
	}
	// Should return nil when all issues are excluded
	if issue != nil {
		t.Errorf("NextIssueExcluding() = %v, want nil (all issues excluded)", issue)
	}
}

func TestNextIssueExcluding_NilSkipWorksLikeNextIssue(t *testing.T) {
	// Test that passing nil skip set works like NextIssue (returns first issue).
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
				{ID: "proj-2", Title: "Second", Priority: 1, IssueType: "feature", Status: "open"},
			}, nil
		},
	}

	issue, err := d.NextIssueExcluding(nil)
	if err != nil {
		t.Fatalf("NextIssueExcluding(nil) unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssueExcluding(nil) expected issue, got nil")
	}
	// Should return first issue (no exclusions)
	if issue.ID != "proj-1" {
		t.Errorf("NextIssueExcluding(nil) = %q, want 'proj-1'", issue.ID)
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
