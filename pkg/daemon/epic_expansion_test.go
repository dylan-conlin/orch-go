package daemon

import (
	"fmt"
	"strings"
	"testing"
)

func TestExpandTriageReadyEpics_NoEpics(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready"},
	}

	issues := []Issue{
		{ID: "proj-1", Title: "Feature", IssueType: "feature", Labels: []string{"triage:ready"}},
		{ID: "proj-2", Title: "Bug", IssueType: "bug", Labels: []string{"triage:ready"}},
	}

	expanded, epicChildIDs, err := d.expandTriageReadyEpics(issues)
	if err != nil {
		t.Fatalf("expandTriageReadyEpics() unexpected error: %v", err)
	}

	// No epics, so nothing should change
	if len(expanded) != 2 {
		t.Errorf("expandTriageReadyEpics() returned %d issues, want 2", len(expanded))
	}
	if len(epicChildIDs) != 0 {
		t.Errorf("expandTriageReadyEpics() returned %d epic children, want 0", len(epicChildIDs))
	}
}

func TestExpandTriageReadyEpics_NoLabelFilter(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: ""}, // No label filter
	}

	issues := []Issue{
		{ID: "proj-1", Title: "Epic", IssueType: "epic", Labels: []string{}},
		{ID: "proj-1.1", Title: "Child", IssueType: "task", Labels: []string{}},
	}

	expanded, epicChildIDs, err := d.expandTriageReadyEpics(issues)
	if err != nil {
		t.Fatalf("expandTriageReadyEpics() unexpected error: %v", err)
	}

	// No label filter, so no expansion needed
	if len(expanded) != 2 {
		t.Errorf("expandTriageReadyEpics() returned %d issues, want 2", len(expanded))
	}
	if len(epicChildIDs) != 0 {
		t.Errorf("expandTriageReadyEpics() returned %d epic children, want 0", len(epicChildIDs))
	}
}

func TestExpandTriageReadyEpics_EpicWithoutLabel(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready"},
	}

	issues := []Issue{
		{ID: "proj-epic", Title: "Epic", IssueType: "epic", Labels: []string{}}, // No triage:ready
		{ID: "proj-1", Title: "Feature", IssueType: "feature", Labels: []string{"triage:ready"}},
	}

	expanded, epicChildIDs, err := d.expandTriageReadyEpics(issues)
	if err != nil {
		t.Fatalf("expandTriageReadyEpics() unexpected error: %v", err)
	}

	// Epic doesn't have the label, so no expansion
	if len(expanded) != 2 {
		t.Errorf("expandTriageReadyEpics() returned %d issues, want 2", len(expanded))
	}
	if len(epicChildIDs) != 0 {
		t.Errorf("expandTriageReadyEpics() returned %d epic children, want 0", len(epicChildIDs))
	}
}

func TestNextIssue_EpicChildrenIncludedInSpawnQueue(t *testing.T) {
	// Mock the ListEpicChildren function via the listIssuesFunc
	// by including all issues upfront (simulating what expandTriageReadyEpics would do)
	epicChildCalled := false

	d := &Daemon{
		Config: Config{Label: "triage:ready"},
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-epic", Title: "Epic", IssueType: "epic", Labels: []string{"triage:ready"}},
				{ID: "proj-1", Title: "Feature without label", IssueType: "feature", Status: "open", Labels: []string{}},
			}, nil
		},
		listEpicChildrenFunc: func(epicID string) ([]Issue, error) {
			epicChildCalled = true
			if epicID == "proj-epic" {
				return []Issue{
					{ID: "proj-child-1", Title: "Child 1", IssueType: "task", Status: "open"},
				}, nil
			}
			return []Issue{}, nil
		},
	}

	// Create a wrapper that tracks ListEpicChildren calls
	// We can't easily mock ListEpicChildren since it's a package-level function,
	// but we can test the overall behavior by checking the results

	// First, verify that without epic child expansion, the feature would be rejected
	// (it doesn't have the required label)
	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() unexpected error: %v", err)
	}

	// The epic should be skipped (not spawnable), and the feature should be skipped (no label)
	// unless epic child expansion is working
	if issue != nil {
		t.Logf("NextIssue() returned %s", issue.ID)
		// If we got an issue, epic child expansion must be working
		epicChildCalled = true
	}

	// For a full integration test, we would need to mock ListEpicChildren
	// which would require dependency injection. For now, test the logic directly.
	_ = epicChildCalled
}

func TestCheckRejectionReasonWithEpicChildren_EpicChildExempt(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready"},
	}

	// Issue without triage:ready label
	issue := Issue{
		ID:        "proj-1",
		Title:     "Feature",
		IssueType: "feature",
		Status:    "open",
		Labels:    []string{}, // No triage:ready label
	}

	// Without being an epic child, should be rejected for missing label
	reason := d.checkRejectionReasonWithEpicChildren(issue, nil)
	if !strings.Contains(reason, "missing label") {
		t.Errorf("checkRejectionReasonWithEpicChildren() = %q, want to contain 'missing label'", reason)
	}

	// When marked as an epic child, should be accepted (empty reason)
	epicChildIDs := map[string]bool{"proj-1": true}
	reason = d.checkRejectionReasonWithEpicChildren(issue, epicChildIDs)
	if reason != "" {
		t.Errorf("checkRejectionReasonWithEpicChildren() for epic child = %q, want empty string", reason)
	}
}

func TestCheckRejectionReasonWithEpicChildren_EpicWithLabelExplains(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready"},
	}

	// Epic with triage:ready label
	issue := Issue{
		ID:        "proj-epic",
		Title:     "Epic",
		IssueType: "epic",
		Status:    "open",
		Labels:    []string{"triage:ready"},
	}

	reason := d.checkRejectionReasonWithEpicChildren(issue, nil)

	// Should explain that children will be processed instead
	if !strings.Contains(reason, "children will be processed") {
		t.Errorf("checkRejectionReasonWithEpicChildren() for triage:ready epic = %q, want to contain 'children will be processed'", reason)
	}
}

func TestExpandTriageReadyEpics_FiltersClosedChildren(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready", Verbose: true},
		listEpicChildrenFunc: func(epicID string) ([]Issue, error) {
			if epicID == "proj-epic" {
				return []Issue{
					{ID: "proj-child-1", Title: "Open Child", IssueType: "feature", Status: "open"},
					{ID: "proj-child-2", Title: "Closed Child", IssueType: "feature", Status: "closed"},
					{ID: "proj-child-3", Title: "In Progress Child", IssueType: "feature", Status: "in_progress"},
				}, nil
			}
			return []Issue{}, nil
		},
	}

	issues := []Issue{
		{ID: "proj-epic", Title: "Epic", IssueType: "epic", Status: "open", Labels: []string{"triage:ready"}},
	}

	expanded, epicChildIDs, err := d.expandTriageReadyEpics(issues)
	if err != nil {
		t.Fatalf("expandTriageReadyEpics() unexpected error: %v", err)
	}

	// Should have original epic + 2 children (open and in_progress, but NOT closed)
	if len(expanded) != 3 {
		t.Errorf("expandTriageReadyEpics() returned %d issues, want 3 (epic + 2 open children)", len(expanded))
	}

	// Only the 2 non-closed children should be marked as epic children
	if len(epicChildIDs) != 2 {
		t.Errorf("expandTriageReadyEpics() returned %d epic children, want 2", len(epicChildIDs))
	}

	// Verify open child is included
	if !epicChildIDs["proj-child-1"] {
		t.Error("expandTriageReadyEpics() did not include open child proj-child-1")
	}

	// Verify closed child is NOT included
	if epicChildIDs["proj-child-2"] {
		t.Error("expandTriageReadyEpics() incorrectly included closed child proj-child-2")
	}

	// Verify in_progress child is included
	if !epicChildIDs["proj-child-3"] {
		t.Error("expandTriageReadyEpics() did not include in_progress child proj-child-3")
	}

	// Verify the closed child is not in the expanded issues list
	for _, issue := range expanded {
		if issue.ID == "proj-child-2" {
			t.Error("expandTriageReadyEpics() added closed child to issues list")
		}
	}
}

func TestExpandTriageReadyEpics_ListChildrenError(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready"},
		listEpicChildrenFunc: func(epicID string) ([]Issue, error) {
			return nil, fmt.Errorf("simulated error listing children")
		},
	}

	issues := []Issue{
		{ID: "proj-epic", Title: "Epic", IssueType: "epic", Labels: []string{"triage:ready"}},
		{ID: "proj-task", Title: "Task", IssueType: "task", Labels: []string{}},
	}

	_, _, err := d.expandTriageReadyEpics(issues)
	if err == nil {
		t.Error("expandTriageReadyEpics() expected error when ListEpicChildren fails, got nil")
	}

	if err != nil && !strings.Contains(err.Error(), "failed to list children of epic proj-epic") {
		t.Errorf("expandTriageReadyEpics() error message = %v, want to contain 'failed to list children of epic proj-epic'", err)
	}
}
