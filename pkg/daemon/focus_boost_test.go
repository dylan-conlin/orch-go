package daemon

import (
	"testing"
)

func TestMatchFocusToProject(t *testing.T) {
	tests := []struct {
		name      string
		goal      string
		prefix    string
		dirNames  map[string]string // prefix → dir basename
		wantMatch bool
	}{
		{
			name:      "direct prefix match",
			goal:      "orch-go reliability",
			prefix:    "orch-go",
			wantMatch: true,
		},
		{
			name:      "case insensitive prefix match",
			goal:      "Orch-Go reliability",
			prefix:    "orch-go",
			wantMatch: true,
		},
		{
			name:      "dir basename match via registry",
			goal:      "price-watch improvements",
			prefix:    "pw",
			dirNames:  map[string]string{"pw": "price-watch"},
			wantMatch: true,
		},
		{
			name:      "no match",
			goal:      "daemon reliability",
			prefix:    "pw",
			dirNames:  map[string]string{"pw": "price-watch"},
			wantMatch: false,
		},
		{
			name:      "empty goal",
			goal:      "",
			prefix:    "orch-go",
			wantMatch: false,
		},
		{
			name:      "empty prefix",
			goal:      "orch-go work",
			prefix:    "",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchFocusToProject(tt.goal, tt.prefix, tt.dirNames)
			if got != tt.wantMatch {
				t.Errorf("matchFocusToProject(%q, %q, %v) = %v, want %v",
					tt.goal, tt.prefix, tt.dirNames, got, tt.wantMatch)
			}
		})
	}
}

func TestApplyFocusBoost(t *testing.T) {
	t.Run("boosts matching project issues by 1 priority level", func(t *testing.T) {
		issues := []Issue{
			{ID: "pw-1", Title: "PW task", Priority: 2, IssueType: "feature"},
			{ID: "orch-go-1", Title: "Orch task", Priority: 2, IssueType: "feature"},
		}

		boosted := applyFocusBoost(issues, "price-watch", 1, map[string]string{"pw": "price-watch"})

		// pw-1 should have priority boosted from 2 to 1
		if boosted[0].Priority != 1 {
			t.Errorf("boosted pw-1 priority = %d, want 1", boosted[0].Priority)
		}
		// orch-go-1 should remain at 2
		if boosted[1].Priority != 2 {
			t.Errorf("non-boosted orch-go-1 priority = %d, want 2", boosted[1].Priority)
		}
	})

	t.Run("does not boost below 0", func(t *testing.T) {
		issues := []Issue{
			{ID: "pw-1", Title: "PW task", Priority: 0, IssueType: "feature"},
		}

		boosted := applyFocusBoost(issues, "price-watch", 1, map[string]string{"pw": "price-watch"})

		if boosted[0].Priority != 0 {
			t.Errorf("boosted P0 priority = %d, want 0 (should not go below 0)", boosted[0].Priority)
		}
	})

	t.Run("empty goal returns issues unchanged", func(t *testing.T) {
		issues := []Issue{
			{ID: "pw-1", Title: "PW task", Priority: 2, IssueType: "feature"},
		}

		boosted := applyFocusBoost(issues, "", 1, nil)

		if boosted[0].Priority != 2 {
			t.Errorf("priority = %d, want 2 (no boost with empty goal)", boosted[0].Priority)
		}
	})

	t.Run("configurable boost amount", func(t *testing.T) {
		issues := []Issue{
			{ID: "pw-1", Title: "PW task", Priority: 3, IssueType: "feature"},
		}

		boosted := applyFocusBoost(issues, "price-watch", 2, map[string]string{"pw": "price-watch"})

		if boosted[0].Priority != 1 {
			t.Errorf("boosted priority = %d, want 1 (boost of 2 from P3)", boosted[0].Priority)
		}
	})

	t.Run("does not modify original slice", func(t *testing.T) {
		issues := []Issue{
			{ID: "pw-1", Title: "PW task", Priority: 2, IssueType: "feature"},
		}

		_ = applyFocusBoost(issues, "price-watch", 1, map[string]string{"pw": "price-watch"})

		if issues[0].Priority != 2 {
			t.Errorf("original priority = %d, want 2 (should not modify original)", issues[0].Priority)
		}
	})
}

func TestBuildProjectDirNames(t *testing.T) {
	// Test with nil registry
	names := BuildProjectDirNames(nil)
	if len(names) != 0 {
		t.Errorf("nil registry should return empty map, got %v", names)
	}
}

func TestNextIssue_FocusBoost(t *testing.T) {
	// P2 issue from focused project should sort before P2 from other project
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "orch-go-1", Title: "Orch P2", Priority: 2, IssueType: "feature"},
				{ID: "pw-1", Title: "PW P2", Priority: 2, IssueType: "feature"},
			}, nil
		}},
		FocusGoal:          "price-watch",
		FocusBoostAmount:   1,
		ProjectDirNames:    map[string]string{"pw": "price-watch"},
	}

	issue, err := d.NextIssue()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("expected issue, got nil")
	}
	// PW P2 should be boosted to effective P1, so it comes first
	if issue.ID != "pw-1" {
		t.Errorf("expected pw-1 (focus boosted), got %s", issue.ID)
	}
}

func TestNextIssue_FocusBoost_NoFocus(t *testing.T) {
	// Without focus, normal priority ordering applies
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "orch-go-1", Title: "Orch P1", Priority: 1, IssueType: "feature"},
				{ID: "pw-1", Title: "PW P2", Priority: 2, IssueType: "feature"},
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
	if issue.ID != "orch-go-1" {
		t.Errorf("expected orch-go-1 (P1), got %s", issue.ID)
	}
}

func TestPreview_ShowsFocusBoost(t *testing.T) {
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "pw-1", Title: "PW task", Priority: 2, IssueType: "feature"},
			}, nil
		}},
		FocusGoal:        "price-watch",
		FocusBoostAmount: 1,
		ProjectDirNames:  map[string]string{"pw": "price-watch"},
	}

	result, err := d.Preview()
	if err != nil {
		t.Fatalf("Preview() error: %v", err)
	}
	if result.Issue == nil {
		t.Fatal("expected issue in preview")
	}
	if !result.FocusBoosted {
		t.Error("expected FocusBoosted=true for focused project issue")
	}
	if result.FocusGoal != "price-watch" {
		t.Errorf("FocusGoal = %q, want %q", result.FocusGoal, "price-watch")
	}
}
