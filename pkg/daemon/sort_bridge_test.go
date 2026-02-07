package daemon

import (
	"testing"

	daemonsort "github.com/dylan-conlin/orch-go/pkg/daemon/sort"
	"github.com/dylan-conlin/orch-go/pkg/frontier"
)

func TestToSortIssues(t *testing.T) {
	issues := []Issue{
		{ID: "a", Title: "Title A", Priority: 0, IssueType: "task", Labels: []string{"triage:ready"}},
		{ID: "b", Title: "Title B", Priority: 1, IssueType: "bug"},
	}

	result := toSortIssues(issues)

	if len(result) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(result))
	}
	if result[0].ID != "a" || result[0].Title != "Title A" || result[0].Priority != 0 {
		t.Errorf("first issue not converted correctly: %+v", result[0])
	}
	if len(result[0].Labels) != 1 || result[0].Labels[0] != "triage:ready" {
		t.Errorf("labels not converted correctly: %v", result[0].Labels)
	}
}

func TestFromSortIssues(t *testing.T) {
	issues := []daemonsort.Issue{
		{ID: "x", Title: "X", Priority: 2, IssueType: "feature"},
	}

	result := fromSortIssues(issues)

	if len(result) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result))
	}
	if result[0].ID != "x" || result[0].IssueType != "feature" {
		t.Errorf("issue not converted correctly: %+v", result[0])
	}
}

func TestBuildSortContextFromFrontier_Nil(t *testing.T) {
	ctx := buildSortContextFromFrontier(nil)
	if ctx != nil {
		t.Error("expected nil context from nil frontier")
	}
}

func TestBuildSortContextFromFrontier_Empty(t *testing.T) {
	fs := &frontier.FrontierState{}
	ctx := buildSortContextFromFrontier(fs)

	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	if len(ctx.Leverage) != 0 {
		t.Errorf("expected empty leverage map, got %d entries", len(ctx.Leverage))
	}
}

func TestBuildSortContextFromFrontier_MapsLeverage(t *testing.T) {
	fs := &frontier.FrontierState{
		Blocked: []*frontier.BlockedIssue{
			{
				Issue: &frontier.Issue{
					ID:        "blocked-1",
					BlockedBy: []string{"ready-a"},
				},
				TotalLeverage: 2,
			},
			{
				Issue: &frontier.Issue{
					ID:        "blocked-2",
					BlockedBy: []string{"ready-a", "ready-b"},
				},
				TotalLeverage: 1,
			},
		},
	}

	ctx := buildSortContextFromFrontier(fs)

	if ctx == nil {
		t.Fatal("expected non-nil context")
	}

	// ready-a blocks both blocked-1 and blocked-2
	leverageA, ok := ctx.Leverage["ready-a"]
	if !ok {
		t.Fatal("expected leverage for 'ready-a'")
	}
	// blocked-1 (leverage 2) + 1 for the blocked issue itself = 3
	// blocked-2 (leverage 1) + 1 for the blocked issue itself = 2
	// Total for ready-a: 3 + 2 = 5
	if leverageA.TotalLeverage != 5 {
		t.Errorf("expected ready-a leverage 5, got %d", leverageA.TotalLeverage)
	}
	if len(leverageA.WouldUnblock) != 2 {
		t.Errorf("expected ready-a to unblock 2, got %d", len(leverageA.WouldUnblock))
	}

	// ready-b blocks only blocked-2
	leverageB, ok := ctx.Leverage["ready-b"]
	if !ok {
		t.Fatal("expected leverage for 'ready-b'")
	}
	if leverageB.TotalLeverage != 2 {
		t.Errorf("expected ready-b leverage 2, got %d", leverageB.TotalLeverage)
	}
}

func TestSortIssues_DefaultPriority(t *testing.T) {
	d := NewWithConfig(Config{})
	issues := []Issue{
		{ID: "c", Priority: 2},
		{ID: "a", Priority: 0},
		{ID: "b", Priority: 1},
	}

	result := d.SortIssues(issues)

	if result[0].ID != "a" || result[1].ID != "b" || result[2].ID != "c" {
		t.Errorf("expected priority order [a, b, c], got [%s, %s, %s]",
			result[0].ID, result[1].ID, result[2].ID)
	}
}

func TestSortIssues_UnblockMode(t *testing.T) {
	d := NewWithConfig(Config{SortMode: "unblock"})
	// Pre-populate frontier cache
	d.CachedFrontier = &frontier.FrontierState{
		Blocked: []*frontier.BlockedIssue{
			{
				Issue: &frontier.Issue{
					ID:        "blocked-1",
					BlockedBy: []string{"low-pri"},
				},
				TotalLeverage: 5,
			},
		},
	}

	issues := []Issue{
		{ID: "high-pri", Priority: 0, IssueType: "task"},
		{ID: "low-pri", Priority: 2, IssueType: "task"},
	}

	result := d.SortIssues(issues)

	// low-pri has leverage (blocks blocked-1 with leverage 5), should come first
	if result[0].ID != "low-pri" {
		t.Errorf("expected 'low-pri' first (has leverage), got %q", result[0].ID)
	}
}

func TestSortIssues_UnblockModeWithoutFrontier(t *testing.T) {
	d := NewWithConfig(Config{SortMode: "unblock"})
	// No frontier cache — should degrade to priority sort
	issues := []Issue{
		{ID: "c", Priority: 2, IssueType: "task"},
		{ID: "a", Priority: 0, IssueType: "task"},
	}

	result := d.SortIssues(issues)

	if result[0].ID != "a" {
		t.Errorf("expected priority fallback, got %q first", result[0].ID)
	}
}

func TestSortCrossProjectIssues(t *testing.T) {
	d := NewWithConfig(Config{})
	issues := []CrossProjectIssue{
		{Issue: Issue{ID: "c", Priority: 2}, Project: Project{Name: "proj1"}},
		{Issue: Issue{ID: "a", Priority: 0}, Project: Project{Name: "proj2"}},
		{Issue: Issue{ID: "b", Priority: 1}, Project: Project{Name: "proj1"}},
	}

	result := d.SortCrossProjectIssues(issues)

	if result[0].Issue.ID != "a" || result[1].Issue.ID != "b" || result[2].Issue.ID != "c" {
		t.Errorf("expected priority order [a, b, c], got [%s, %s, %s]",
			result[0].Issue.ID, result[1].Issue.ID, result[2].Issue.ID)
	}
	// Verify project association preserved
	if result[0].Project.Name != "proj2" {
		t.Errorf("expected project 'proj2' for issue 'a', got %q", result[0].Project.Name)
	}
}

func TestSortCrossProjectIssues_Empty(t *testing.T) {
	d := NewWithConfig(Config{})
	result := d.SortCrossProjectIssues(nil)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d", len(result))
	}
}

func TestDaemon_SortStrategyName(t *testing.T) {
	tests := []struct {
		mode     string
		expected string
	}{
		{"", "priority"},
		{"priority", "priority"},
		{"unblock", "unblock"},
	}

	for _, tt := range tests {
		d := NewWithConfig(Config{SortMode: tt.mode})
		if d.SortStrategy.Name() != tt.expected {
			t.Errorf("SortMode=%q: expected strategy name %q, got %q",
				tt.mode, tt.expected, d.SortStrategy.Name())
		}
	}
}

func TestDaemon_InvalidSortModeFallsBack(t *testing.T) {
	d := NewWithConfig(Config{SortMode: "invalid"})
	// Should fall back to priority
	if d.SortStrategy.Name() != "priority" {
		t.Errorf("expected fallback to 'priority', got %q", d.SortStrategy.Name())
	}
}
