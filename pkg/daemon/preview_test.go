package daemon

import (
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

func TestDaemon_Preview_ShowsRejectionReasons(t *testing.T) {
	// Test that Preview returns rejection reasons for non-spawnable issues
	d := &Daemon{
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Missing type", Priority: 0, IssueType: "", Status: "open"},
				{ID: "proj-2", Title: "Epic type", Priority: 1, IssueType: "epic", Status: "open"},
				{ID: "proj-3", Title: "Blocked", Priority: 2, IssueType: "feature", Status: "blocked"},
				{ID: "proj-4", Title: "In progress", Priority: 3, IssueType: "feature", Status: "in_progress"},
				{ID: "proj-5", Title: "Spawnable", Priority: 4, IssueType: "bug", Status: "open"},
			}, nil
		},
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
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "No label", Priority: 0, IssueType: "feature", Status: "open", Labels: []string{}},
				{ID: "proj-2", Title: "Has label", Priority: 1, IssueType: "feature", Status: "open", Labels: []string{"triage:ready"}},
			}, nil
		},
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

	if !strings.Contains(output, "Rejected issues:") {
		t.Error("FormatRejectedIssues() missing header")
	}
	if !strings.Contains(output, "proj-1: missing type") {
		t.Error("FormatRejectedIssues() missing proj-1 entry")
	}
	if !strings.Contains(output, "proj-2: status is in_progress") {
		t.Error("FormatRejectedIssues() missing proj-2 entry")
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
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
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

func TestPreview_EpicWithTriageReadyShowsHelpfulMessage(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready"},
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-epic", Title: "Epic", IssueType: "epic", Status: "open", Labels: []string{"triage:ready"}},
			}, nil
		},
		// Mock to return no children, isolating the test from real data
		listEpicChildrenFunc: func(epicID string) ([]Issue, error) {
			return []Issue{}, nil
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
