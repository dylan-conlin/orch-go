package daemon

import (
	"fmt"
	"strings"
	"testing"
)

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
	if !strings.Contains(preview, "proj-123") {
		t.Error("FormatPreview() missing issue ID")
	}
	if !strings.Contains(preview, "Fix critical bug") {
		t.Error("FormatPreview() missing title")
	}
	if !strings.Contains(preview, "bug") {
		t.Error("FormatPreview() missing issue type")
	}
}

func TestDaemon_Preview_NilListIssuesFunc(t *testing.T) {
	// Regression test: Preview() must not panic when listIssuesFunc is nil.
	// This is the production path — resolveListIssuesFunc() falls back to ListReadyIssues.
	d := NewWithConfig(Config{})

	// This would panic with "nil pointer dereference" before the fix
	// because preview.go called d.listIssuesFunc() directly instead of
	// d.resolveListIssuesFunc(). We don't check the result since
	// ListReadyIssues calls the real bd CLI — we just verify no panic.
	result, err := d.Preview()
	// If bd is not available, we'll get an error — that's fine, no panic is the goal
	if err != nil {
		t.Logf("Preview() returned error (expected without bd): %v", err)
		return
	}
	if result == nil {
		t.Error("Preview() returned nil result without error")
	}
}

func TestDaemon_Preview_NoIssues(t *testing.T) {
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{}, nil
		}},
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
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test issue", Priority: 1, IssueType: "feature", Status: "open"},
			}, nil
		}},
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

func TestDaemon_Preview_ShowsRejectionReasons(t *testing.T) {
	// Test that Preview returns rejection reasons for non-spawnable issues
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Missing type", Priority: 0, IssueType: "", Status: "open"},
				{ID: "proj-2", Title: "Epic type", Priority: 1, IssueType: "epic", Status: "open"},
				{ID: "proj-3", Title: "Blocked", Priority: 2, IssueType: "feature", Status: "blocked"},
				{ID: "proj-4", Title: "In progress", Priority: 3, IssueType: "feature", Status: "in_progress"},
				{ID: "proj-5", Title: "Spawnable", Priority: 4, IssueType: "bug", Status: "open"},
			}, nil
		}},
	}

	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}

	// Should have one spawnable issue
	if result.Issue == nil {
		t.Fatal("Preview() expected spawnable issue, got nil")
	}
	if result.Issue.ID != "proj-5" {
		t.Errorf("Preview() spawnable issue ID = %q, want 'proj-5'", result.Issue.ID)
	}

	// Should have 4 rejected issues with reasons
	if len(result.RejectedIssues) != 4 {
		t.Errorf("Preview() rejected count = %d, want 4", len(result.RejectedIssues))
	}

	// Check rejection reasons
	rejectedByID := make(map[string]string)
	for _, r := range result.RejectedIssues {
		rejectedByID[r.Issue.ID] = r.Reason
	}

	if r, ok := rejectedByID["proj-1"]; !ok {
		t.Error("Preview() missing rejection for proj-1 (empty type)")
	} else if !strings.Contains(r, "missing type") {
		t.Errorf("Preview() proj-1 reason = %q, want to contain 'missing type'", r)
	}

	if r, ok := rejectedByID["proj-2"]; !ok {
		t.Error("Preview() missing rejection for proj-2 (epic type)")
	} else if !strings.Contains(r, "not spawnable") {
		t.Errorf("Preview() proj-2 reason = %q, want to contain 'not spawnable'", r)
	}

	if r, ok := rejectedByID["proj-3"]; !ok {
		t.Error("Preview() missing rejection for proj-3 (blocked status)")
	} else if !strings.Contains(r, "blocked") {
		t.Errorf("Preview() proj-3 reason = %q, want to contain 'blocked'", r)
	}

	if r, ok := rejectedByID["proj-4"]; !ok {
		t.Error("Preview() missing rejection for proj-4 (in_progress)")
	} else if !strings.Contains(r, "in_progress") {
		t.Errorf("Preview() proj-4 reason = %q, want to contain 'in_progress'", r)
	}
}

func TestDaemon_Preview_ShowsMissingLabelRejection(t *testing.T) {
	// Test that Preview shows rejection reason for missing label
	d := &Daemon{
		Config: Config{Label: "triage:ready"},
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "No label", Priority: 0, IssueType: "feature", Status: "open", Labels: []string{}},
				{ID: "proj-2", Title: "Has label", Priority: 1, IssueType: "feature", Status: "open", Labels: []string{"triage:ready"}},
			}, nil
		}},
	}

	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}

	// Should select proj-2 (has required label)
	if result.Issue == nil {
		t.Fatal("Preview() expected spawnable issue, got nil")
	}
	if result.Issue.ID != "proj-2" {
		t.Errorf("Preview() spawnable issue ID = %q, want 'proj-2'", result.Issue.ID)
	}

	// Should reject proj-1 for missing label
	if len(result.RejectedIssues) != 1 {
		t.Errorf("Preview() rejected count = %d, want 1", len(result.RejectedIssues))
	}
	if len(result.RejectedIssues) > 0 {
		r := result.RejectedIssues[0]
		if r.Issue.ID != "proj-1" {
			t.Errorf("Preview() rejected issue ID = %q, want 'proj-1'", r.Issue.ID)
		}
		if !strings.Contains(r.Reason, "missing label") {
			t.Errorf("Preview() rejection reason = %q, want to contain 'missing label'", r.Reason)
		}
	}
}

func TestFormatRejectedIssues(t *testing.T) {
	rejected := []RejectedIssue{
		{Issue: Issue{ID: "proj-1"}, Reason: "missing type (required for skill inference)"},
		{Issue: Issue{ID: "proj-2"}, Reason: "status is in_progress (already being worked on)"},
	}

	output := FormatRejectedIssues(rejected)

	if !strings.Contains(output, "Rejected (2 issues):") {
		t.Errorf("FormatRejectedIssues() missing grouped header, got: %s", output)
	}
	if !strings.Contains(output, "missing type (required for skill inference): 1") {
		t.Errorf("FormatRejectedIssues() missing grouped reason count, got: %s", output)
	}
	if !strings.Contains(output, "status is in_progress (already being worked on): 1") {
		t.Errorf("FormatRejectedIssues() missing grouped reason count, got: %s", output)
	}
	// Should NOT contain individual issue IDs
	if strings.Contains(output, "proj-1:") || strings.Contains(output, "proj-2:") {
		t.Errorf("FormatRejectedIssues() should not list individual issue IDs, got: %s", output)
	}
}

func TestFormatRejectedIssues_GroupsByReason(t *testing.T) {
	// Simulate many issues rejected for the same reason (the actual bug scenario)
	rejected := make([]RejectedIssue, 0, 205)
	for i := 0; i < 180; i++ {
		rejected = append(rejected, RejectedIssue{
			Issue:  Issue{ID: fmt.Sprintf("proj-%d", i)},
			Reason: "missing label 'triage:ready'",
		})
	}
	for i := 0; i < 15; i++ {
		rejected = append(rejected, RejectedIssue{
			Issue:  Issue{ID: fmt.Sprintf("prog-%d", i)},
			Reason: "status is in_progress (already being worked on)",
		})
	}
	for i := 0; i < 10; i++ {
		rejected = append(rejected, RejectedIssue{
			Issue:  Issue{ID: fmt.Sprintf("other-%d", i)},
			Reason: "status is blocked",
		})
	}

	output := FormatRejectedIssues(rejected)

	// Should show total count in header
	if !strings.Contains(output, "Rejected (205 issues):") {
		t.Errorf("expected 'Rejected (205 issues):' header, got: %s", output)
	}
	// Should group by reason with counts
	if !strings.Contains(output, "missing label 'triage:ready': 180") {
		t.Errorf("expected grouped count for missing label, got: %s", output)
	}
	if !strings.Contains(output, "status is in_progress (already being worked on): 15") {
		t.Errorf("expected grouped count for in_progress, got: %s", output)
	}
	if !strings.Contains(output, "status is blocked: 10") {
		t.Errorf("expected grouped count for blocked, got: %s", output)
	}
	// Output should be compact - only a few lines, not 205
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) > 6 {
		t.Errorf("expected compact output (<=6 lines), got %d lines", len(lines))
	}
}

func TestFormatRejectedIssues_Empty(t *testing.T) {
	output := FormatRejectedIssues(nil)
	if output != "" {
		t.Errorf("FormatRejectedIssues(nil) = %q, want empty string", output)
	}

	output = FormatRejectedIssues([]RejectedIssue{})
	if output != "" {
		t.Errorf("FormatRejectedIssues([]) = %q, want empty string", output)
	}
}

func TestDaemon_Preview_RateLimited(t *testing.T) {
	d := &Daemon{
		Config: Config{MaxSpawnsPerHour: 1},
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		}},
	}
	d.RateLimiter = NewRateLimiter(1)

	// First preview should show rate status
	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}
	if result.RateLimited {
		t.Error("First preview should not be rate limited")
	}
	if result.RateStatus == "" {
		t.Error("Rate status should be set")
	}

	// Record a spawn to hit limit
	d.RateLimiter.RecordSpawn()

	// Preview should now show rate limited
	result, err = d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}
	if !result.RateLimited {
		t.Error("Second preview should be rate limited")
	}
}

func TestPreview_TriageApprovedIsSpawnable(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready"},
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "proj-1", Title: "Approved item", IssueType: "feature", Status: "open", Labels: []string{"triage:approved"}},
				}, nil
			},
		},
	}

	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}

	if result.Issue == nil {
		t.Fatal("Preview() expected triage:approved issue to be spawnable, got nil")
	}
	if result.Issue.ID != "proj-1" {
		t.Errorf("Preview() issue ID = %q, want 'proj-1'", result.Issue.ID)
	}
}

func TestPreview_EpicWithTriageApprovedShowsHelpfulMessage(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready"},
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "proj-epic", Title: "Epic", IssueType: "epic", Status: "open", Labels: []string{"triage:approved"}},
				}, nil
			},
			ListEpicChildrenFunc: func(epicID string) ([]Issue, error) {
				return []Issue{}, nil
			},
		},
	}

	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}

	if result.Issue != nil {
		t.Errorf("Preview() expected nil issue (epic not spawnable), got %v", result.Issue)
	}

	if len(result.RejectedIssues) != 1 {
		t.Fatalf("Preview() rejected count = %d, want 1", len(result.RejectedIssues))
	}
	if !strings.Contains(result.RejectedIssues[0].Reason, "children will be processed") {
		t.Errorf("Preview() rejection reason = %q, want 'children will be processed'", result.RejectedIssues[0].Reason)
	}
}

func TestPreview_EpicWithTriageReadyShowsHelpfulMessage(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready"},
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "proj-epic", Title: "Epic", IssueType: "epic", Status: "open", Labels: []string{"triage:ready"}},
				}, nil
			},
			// Mock to return no children, isolating the test from real data
			ListEpicChildrenFunc: func(epicID string) ([]Issue, error) {
				return []Issue{}, nil
			},
		},
	}

	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}

	// No spawnable issue since only an epic in queue
	if result.Issue != nil {
		t.Errorf("Preview() expected nil issue (epic not spawnable), got %v", result.Issue)
	}

	// Should have the epic in rejected list with helpful message
	if len(result.RejectedIssues) != 1 {
		t.Fatalf("Preview() rejected count = %d, want 1", len(result.RejectedIssues))
	}

	rejected := result.RejectedIssues[0]
	if rejected.Issue.ID != "proj-epic" {
		t.Errorf("Preview() rejected issue ID = %q, want 'proj-epic'", rejected.Issue.ID)
	}
	if !strings.Contains(rejected.Reason, "children will be processed") {
		t.Errorf("Preview() rejection reason = %q, want to contain 'children will be processed'", rejected.Reason)
	}
}
