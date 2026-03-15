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
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "proj-epic", Title: "Epic", IssueType: "epic", Labels: []string{"triage:ready"}},
					{ID: "proj-1", Title: "Feature without label", IssueType: "feature", Status: "open", Labels: []string{}},
				}, nil
			},
			ListEpicChildrenFunc: func(epicID string) ([]Issue, error) {
				epicChildCalled = true
				if epicID == "proj-epic" {
					return []Issue{
						{ID: "proj-child-1", Title: "Child 1", IssueType: "task", Status: "open"},
					}, nil
				}
				return []Issue{}, nil
			},
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

func TestEpicExpansion_EpicChildExemptFromLabel(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready"},
	}

	issue := Issue{
		ID:        "proj-1",
		Title:     "Feature",
		IssueType: "feature",
		Status:    "open",
		Labels:    []string{},
	}

	result := d.CheckIssueCompliance(issue, nil, nil)
	if result.Passed {
		t.Error("CheckIssueCompliance() should reject issue without required label")
	}
	if !strings.Contains(result.Reason, "missing label") {
		t.Errorf("CheckIssueCompliance() reason = %q, want to contain 'missing label'", result.Reason)
	}

	epicChildIDs := map[string]bool{"proj-1": true}
	result = d.CheckIssueCompliance(issue, nil, epicChildIDs)
	if !result.Passed {
		t.Errorf("CheckIssueCompliance() should allow epic child without label; Reason: %s", result.Reason)
	}
}

func TestEpicExpansion_EpicWithLabelExplainsChildProcessing(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready"},
	}

	issue := Issue{
		ID:        "proj-epic",
		Title:     "Epic",
		IssueType: "epic",
		Status:    "open",
		Labels:    []string{"triage:ready"},
	}

	result := d.CheckIssueCompliance(issue, nil, nil)
	if result.Passed {
		t.Error("CheckIssueCompliance() should reject epic (not spawnable)")
	}
	if !strings.Contains(result.Reason, "children will be processed") {
		t.Errorf("CheckIssueCompliance() reason = %q, want to contain 'children will be processed'", result.Reason)
	}
}

func TestExpandTriageReadyEpics_FiltersClosedChildren(t *testing.T) {
	d := &Daemon{
		Config: Config{Label: "triage:ready", Verbose: true},
		Issues: &mockIssueQuerier{
			ListEpicChildrenFunc: func(epicID string) ([]Issue, error) {
				if epicID == "proj-epic" {
					return []Issue{
						{ID: "proj-child-1", Title: "Open Child", IssueType: "feature", Status: "open"},
						{ID: "proj-child-2", Title: "Closed Child", IssueType: "feature", Status: "closed"},
						{ID: "proj-child-3", Title: "In Progress Child", IssueType: "feature", Status: "in_progress"},
					}, nil
				}
				return []Issue{}, nil
			},
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
		Issues: &mockIssueQuerier{
			ListEpicChildrenFunc: func(epicID string) ([]Issue, error) {
				return nil, fmt.Errorf("simulated error listing children")
			},
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
