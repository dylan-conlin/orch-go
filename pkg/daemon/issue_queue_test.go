package daemon

import (
	"testing"
)

func TestNextIssue_EmptyQueue(t *testing.T) {
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{}, nil
		}},
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
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-3", Title: "Low priority", Priority: 2, IssueType: "feature"},
				{ID: "proj-1", Title: "High priority", Priority: 0, IssueType: "bug"},
				{ID: "proj-2", Title: "Medium priority", Priority: 1, IssueType: "task"},
			}, nil
		}},
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
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Epic", Priority: 0, IssueType: "epic"},
				{ID: "proj-2", Title: "Feature", Priority: 1, IssueType: "feature"},
			}, nil
		}},
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
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Blocked", Priority: 0, IssueType: "feature", Status: "blocked"},
				{ID: "proj-2", Title: "Open", Priority: 1, IssueType: "feature", Status: "open"},
			}, nil
		}},
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
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "In Progress", Priority: 0, IssueType: "feature", Status: "in_progress", Labels: []string{"triage:ready"}},
				{ID: "proj-2", Title: "Open", Priority: 1, IssueType: "feature", Status: "open", Labels: []string{"triage:ready"}},
			}, nil
		}},
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
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
				{ID: "proj-2", Title: "Second", Priority: 1, IssueType: "feature", Status: "open"},
				{ID: "proj-3", Title: "Third", Priority: 2, IssueType: "feature", Status: "open"},
			}, nil
		}},
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
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
				{ID: "proj-2", Title: "Second", Priority: 1, IssueType: "feature", Status: "open"},
				{ID: "proj-3", Title: "Third", Priority: 2, IssueType: "feature", Status: "open"},
			}, nil
		}},
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
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
				{ID: "proj-2", Title: "Second", Priority: 1, IssueType: "feature", Status: "open"},
			}, nil
		}},
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
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
				{ID: "proj-2", Title: "Second", Priority: 1, IssueType: "feature", Status: "open"},
			}, nil
		}},
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

func TestNextIssue_RoundRobinAcrossProjects(t *testing.T) {
	// When multiple projects have issues at the same priority,
	// daemon should alternate between projects (round-robin).
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "orch-go-1", Title: "Orch 1", Priority: 2, IssueType: "feature"},
				{ID: "orch-go-2", Title: "Orch 2", Priority: 2, IssueType: "feature"},
				{ID: "orch-go-3", Title: "Orch 3", Priority: 2, IssueType: "feature"},
				{ID: "pw-1", Title: "PW 1", Priority: 2, IssueType: "feature"},
				{ID: "pw-2", Title: "PW 2", Priority: 2, IssueType: "feature"},
			}, nil
		}},
	}

	// Collect all issues in order using NextIssueExcluding with skip sets
	var order []string
	skip := map[string]bool{}
	for i := 0; i < 5; i++ {
		issue, err := d.NextIssueExcluding(skip)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		if issue == nil {
			break
		}
		order = append(order, issue.ID)
		skip[issue.ID] = true
	}

	if len(order) != 5 {
		t.Fatalf("expected 5 issues, got %d: %v", len(order), order)
	}

	// Verify round-robin: projects should alternate, not drain one first.
	// Expected interleaved: orch-go, pw, orch-go, pw, orch-go
	// (or pw, orch-go, pw, orch-go, orch-go - order of first project doesn't matter)
	// Key assertion: the first two issues should be from different projects.
	proj0 := projectFromIssueID(order[0])
	proj1 := projectFromIssueID(order[1])
	if proj0 == proj1 {
		t.Errorf("first two issues are from same project %q: %v (expected round-robin)", proj0, order)
	}
}

func TestNextIssue_HigherPriorityBeatsRoundRobin(t *testing.T) {
	// Higher priority issues should still come first regardless of project.
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "orch-go-1", Title: "Orch P2", Priority: 2, IssueType: "feature"},
				{ID: "pw-1", Title: "PW P1", Priority: 1, IssueType: "feature"},
				{ID: "pw-2", Title: "PW P2", Priority: 2, IssueType: "feature"},
			}, nil
		}},
	}

	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("expected issue, got nil")
	}
	// P1 should beat P2 regardless of project
	if issue.ID != "pw-1" {
		t.Errorf("expected pw-1 (P1), got %s", issue.ID)
	}
}

func TestNextIssue_SingleProjectUnaffected(t *testing.T) {
	// Single-project case should work exactly as before.
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "orch-go-1", Title: "First", Priority: 2, IssueType: "feature"},
				{ID: "orch-go-2", Title: "Second", Priority: 2, IssueType: "feature"},
				{ID: "orch-go-3", Title: "Third", Priority: 2, IssueType: "feature"},
			}, nil
		}},
	}

	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("expected issue, got nil")
	}
	// Should return first issue (stable ordering within single project)
	if issue.ID != "orch-go-1" {
		t.Errorf("expected orch-go-1, got %s", issue.ID)
	}
}

func TestProjectFromIssueID(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"orch-go-1234", "orch-go"},
		{"pw-5678", "pw"},
		{"bd-123", "bd"},
		{"scs-special-abc", "scs-special"},
		{"single", "single"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := projectFromIssueID(tt.id)
			if got != tt.want {
				t.Errorf("projectFromIssueID(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func TestInterleaveByProject(t *testing.T) {
	issues := []Issue{
		{ID: "a-1", Priority: 2},
		{ID: "a-2", Priority: 2},
		{ID: "a-3", Priority: 2},
		{ID: "b-1", Priority: 2},
		{ID: "b-2", Priority: 2},
	}
	result := interleaveByProject(issues)

	if len(result) != 5 {
		t.Fatalf("expected 5 issues, got %d", len(result))
	}

	// First two should be from different projects
	proj0 := projectFromIssueID(result[0].ID)
	proj1 := projectFromIssueID(result[1].ID)
	if proj0 == proj1 {
		t.Errorf("first two issues from same project: %v", idsOf(result))
	}
}

func TestInterleaveByProject_PreservesPriorityGroups(t *testing.T) {
	issues := []Issue{
		{ID: "a-1", Priority: 1},
		{ID: "b-1", Priority: 1},
		{ID: "a-2", Priority: 2},
		{ID: "b-2", Priority: 2},
		{ID: "a-3", Priority: 2},
	}
	result := interleaveByProject(issues)

	// All P1 issues should come before P2 issues
	for i, iss := range result {
		if i > 0 && result[i-1].Priority > iss.Priority {
			t.Errorf("priority order violated at index %d: %v", i, idsOf(result))
		}
	}
}

func idsOf(issues []Issue) []string {
	ids := make([]string, len(issues))
	for i, iss := range issues {
		ids[i] = iss.ID
	}
	return ids
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
		{"experiment", true},
		{"question", true},
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
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "No label", Priority: 0, IssueType: "feature", Labels: []string{}},
				{ID: "proj-2", Title: "With label", Priority: 1, IssueType: "feature", Labels: []string{"triage:ready"}},
			}, nil
		}},
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
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "No label", Priority: 0, IssueType: "feature", Labels: []string{}},
				{ID: "proj-2", Title: "With label", Priority: 1, IssueType: "feature", Labels: []string{"triage:ready"}},
			}, nil
		}},
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

func TestNextIssue_TriageApprovedEquivalent(t *testing.T) {
	// triage:approved should be treated as equivalent to triage:ready
	config := Config{Label: "triage:ready"}
	d := &Daemon{
		Config: config,
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "No label", Priority: 0, IssueType: "feature", Labels: []string{}},
				{ID: "proj-2", Title: "Approved", Priority: 1, IssueType: "feature", Labels: []string{"triage:approved"}},
			}, nil
		}},
	}

	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("NextIssue() unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("NextIssue() expected issue with triage:approved to be spawnable, got nil")
	}
	if issue.ID != "proj-2" {
		t.Errorf("NextIssue() = %q, want 'proj-2' (triage:approved should be equivalent to triage:ready)", issue.ID)
	}
}

func TestHasAnyLabel(t *testing.T) {
	tests := []struct {
		name   string
		labels []string
		query  []string
		want   bool
	}{
		{"matches first", []string{"triage:ready"}, []string{"triage:ready", "triage:approved"}, true},
		{"matches second", []string{"triage:approved"}, []string{"triage:ready", "triage:approved"}, true},
		{"no match", []string{"P1"}, []string{"triage:ready", "triage:approved"}, false},
		{"empty labels", []string{}, []string{"triage:ready"}, false},
		{"nil labels", nil, []string{"triage:ready"}, false},
		{"empty query", []string{"triage:ready"}, []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := &Issue{Labels: tt.labels}
			got := issue.HasAnyLabel(tt.query...)
			if got != tt.want {
				t.Errorf("HasAnyLabel(%v) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestSpawnableLabelsFor(t *testing.T) {
	tests := []struct {
		label string
		want  []string
	}{
		{"triage:ready", []string{"triage:ready", "triage:approved"}},
		{"TRIAGE:READY", []string{"triage:ready", "triage:approved"}},
		{"custom:label", []string{"custom:label"}},
		{"", []string{""}},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			got := SpawnableLabelsFor(tt.label)
			if len(got) != len(tt.want) {
				t.Fatalf("SpawnableLabelsFor(%q) returned %d labels, want %d", tt.label, len(got), len(tt.want))
			}
			for i, g := range got {
				if g != tt.want[i] {
					t.Errorf("SpawnableLabelsFor(%q)[%d] = %q, want %q", tt.label, i, g, tt.want[i])
				}
			}
		})
	}
}

func TestIssueMatchesLabel(t *testing.T) {
	tests := []struct {
		name       string
		label      string
		issueLabel []string
		want       bool
	}{
		{"exact match triage:ready", "triage:ready", []string{"triage:ready"}, true},
		{"equivalent triage:approved", "triage:ready", []string{"triage:approved"}, true},
		{"no match", "triage:ready", []string{"P1"}, false},
		{"empty config label matches all", "", []string{"anything"}, true},
		{"custom label exact match", "custom:label", []string{"custom:label"}, true},
		{"custom label no equivalent", "custom:label", []string{"triage:approved"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Daemon{Config: Config{Label: tt.label}}
			issue := Issue{Labels: tt.issueLabel}
			got := d.issueMatchesLabel(issue)
			if got != tt.want {
				t.Errorf("issueMatchesLabel(label=%q, issue=%v) = %v, want %v", tt.label, tt.issueLabel, got, tt.want)
			}
		})
	}
}

// TestNextIssue_TriageLabelRemovedByManualSpawn verifies that when a manual
// spawn removes triage:ready/triage:approved labels from an issue, the daemon's
// NextIssue() correctly skips it. This is the fix for the race condition where
// both manual spawn and daemon pick up the same triage:ready issue.
func TestNextIssue_TriageLabelRemovedByManualSpawn(t *testing.T) {
	config := Config{Label: "triage:ready"}

	t.Run("issue without triage labels is skipped by daemon", func(t *testing.T) {
		d := &Daemon{
			Config: config,
			Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					// Simulates issue after manual spawn removed triage:ready label
					{ID: "proj-1", Title: "Claimed by manual spawn", Priority: 0, IssueType: "feature", Labels: []string{}, Status: "open"},
				}, nil
			}},
		}

		issue, err := d.NextIssue()
		if err != nil {
			t.Fatalf("NextIssue() unexpected error: %v", err)
		}
		if issue != nil {
			t.Errorf("NextIssue() = %q, want nil (issue without triage label should be skipped)", issue.ID)
		}
	})

	t.Run("daemon picks unclaimed issue over claimed one", func(t *testing.T) {
		d := &Daemon{
			Config: config,
			Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					// Manual spawn claimed this (removed triage:ready)
					{ID: "proj-1", Title: "Claimed", Priority: 0, IssueType: "feature", Labels: []string{}, Status: "open"},
					// This one is still available for daemon
					{ID: "proj-2", Title: "Unclaimed", Priority: 1, IssueType: "feature", Labels: []string{"triage:ready"}, Status: "open"},
				}, nil
			}},
		}

		issue, err := d.NextIssue()
		if err != nil {
			t.Fatalf("NextIssue() unexpected error: %v", err)
		}
		if issue == nil {
			t.Fatal("NextIssue() returned nil, expected proj-2")
		}
		if issue.ID != "proj-2" {
			t.Errorf("NextIssue() = %q, want 'proj-2' (should skip claimed proj-1)", issue.ID)
		}
	})

	t.Run("in_progress status also prevents daemon pickup", func(t *testing.T) {
		d := &Daemon{
			Config: config,
			Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					// Manual spawn set this to in_progress (defense in depth)
					{ID: "proj-1", Title: "In progress", Priority: 0, IssueType: "feature", Labels: []string{"triage:ready"}, Status: "in_progress"},
				}, nil
			}},
		}

		issue, err := d.NextIssue()
		if err != nil {
			t.Fatalf("NextIssue() unexpected error: %v", err)
		}
		if issue != nil {
			t.Errorf("NextIssue() = %q, want nil (in_progress issue should be skipped)", issue.ID)
		}
	})
}

// TestNextIssue_DaemonCompletionLabelSkipped verifies that issues with
// daemon:ready-review or daemon:verification-failed labels are skipped
// by the spawn loop. This prevents completed issues from re-entering
// the spawn queue when triage:ready label was not removed.
func TestNextIssue_DaemonCompletionLabelSkipped(t *testing.T) {
	config := Config{Label: "triage:ready"}

	t.Run("daemon:ready-review skipped", func(t *testing.T) {
		d := &Daemon{
			Config: config,
			Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "proj-1", Title: "Completed work", Priority: 0, IssueType: "task",
						Labels: []string{"triage:ready", "daemon:ready-review"}, Status: "open"},
				}, nil
			}},
		}

		issue, err := d.NextIssue()
		if err != nil {
			t.Fatalf("NextIssue() unexpected error: %v", err)
		}
		if issue != nil {
			t.Errorf("NextIssue() = %q, want nil (daemon:ready-review should be skipped)", issue.ID)
		}
	})

	t.Run("daemon:verification-failed skipped", func(t *testing.T) {
		d := &Daemon{
			Config: config,
			Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "proj-1", Title: "Failed verification", Priority: 0, IssueType: "task",
						Labels: []string{"triage:ready", "daemon:verification-failed"}, Status: "open"},
				}, nil
			}},
		}

		issue, err := d.NextIssue()
		if err != nil {
			t.Fatalf("NextIssue() unexpected error: %v", err)
		}
		if issue != nil {
			t.Errorf("NextIssue() = %q, want nil (daemon:verification-failed should be skipped)", issue.ID)
		}
	})

	t.Run("selects uncompleted issue over completed", func(t *testing.T) {
		d := &Daemon{
			Config: config,
			Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "proj-1", Title: "Completed", Priority: 0, IssueType: "task",
						Labels: []string{"triage:ready", "daemon:ready-review"}, Status: "open"},
					{ID: "proj-2", Title: "Ready to spawn", Priority: 1, IssueType: "task",
						Labels: []string{"triage:ready"}, Status: "open"},
				}, nil
			}},
		}

		issue, err := d.NextIssue()
		if err != nil {
			t.Fatalf("NextIssue() unexpected error: %v", err)
		}
		if issue == nil {
			t.Fatal("NextIssue() returned nil, expected proj-2")
		}
		if issue.ID != "proj-2" {
			t.Errorf("NextIssue() = %q, want 'proj-2'", issue.ID)
		}
	})
}
