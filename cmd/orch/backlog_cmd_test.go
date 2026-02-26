package main

import (
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

func TestFilterStaleBacklogIssues(t *testing.T) {
	now := time.Now()
	old := now.AddDate(0, 0, -20).Format(time.RFC3339) // 20 days ago
	recent := now.AddDate(0, 0, -5).Format(time.RFC3339) // 5 days ago
	veryOld := now.AddDate(0, 0, -60).Format(time.RFC3339) // 60 days ago

	issues := []beads.Issue{
		{ID: "og-1", Title: "Old P3 bug", Priority: 3, Status: "open", IssueType: "bug", CreatedAt: old},
		{ID: "og-2", Title: "Recent P3 task", Priority: 3, Status: "open", IssueType: "task", CreatedAt: recent},
		{ID: "og-3", Title: "Old P4 feature", Priority: 4, Status: "open", IssueType: "feature", CreatedAt: veryOld},
		{ID: "og-4", Title: "Old P2 bug", Priority: 2, Status: "open", IssueType: "bug", CreatedAt: old},
		{ID: "og-5", Title: "Old P3 in progress", Priority: 3, Status: "in_progress", IssueType: "task", CreatedAt: old},
		{ID: "og-6", Title: "Old P3 closed", Priority: 3, Status: "closed", IssueType: "task", CreatedAt: old},
	}

	result := filterStaleBacklogIssues(issues, 14, now)

	// Should include: og-1 (P3, 20d old), og-3 (P4, 60d old), og-5 (P3 in_progress, 20d old)
	// Should exclude: og-2 (too recent), og-4 (P2, not low-priority), og-6 (closed)
	if len(result) != 3 {
		t.Fatalf("expected 3 stale issues, got %d", len(result))
	}

	ids := make(map[string]bool)
	for _, si := range result {
		ids[si.Issue.ID] = true
	}

	if !ids["og-1"] {
		t.Error("expected og-1 (old P3 bug) to be included")
	}
	if !ids["og-3"] {
		t.Error("expected og-3 (old P4 feature) to be included")
	}
	if !ids["og-5"] {
		t.Error("expected og-5 (old P3 in_progress) to be included")
	}
	if ids["og-2"] {
		t.Error("expected og-2 (recent P3) to be excluded")
	}
	if ids["og-4"] {
		t.Error("expected og-4 (P2) to be excluded")
	}
	if ids["og-6"] {
		t.Error("expected og-6 (closed) to be excluded")
	}
}

func TestFilterStaleBacklogIssues_UsesUpdatedAt(t *testing.T) {
	now := time.Now()
	oldCreated := now.AddDate(0, 0, -30).Format(time.RFC3339)
	recentUpdated := now.AddDate(0, 0, -3).Format(time.RFC3339)

	issues := []beads.Issue{
		{
			ID:        "og-1",
			Title:     "Old but recently updated",
			Priority:  3,
			Status:    "open",
			IssueType: "task",
			CreatedAt: oldCreated,
			UpdatedAt: recentUpdated,
		},
	}

	result := filterStaleBacklogIssues(issues, 14, now)

	if len(result) != 0 {
		t.Fatalf("expected 0 stale issues (updated recently), got %d", len(result))
	}
}

func TestFilterStaleBacklogIssues_SortsByAge(t *testing.T) {
	now := time.Now()
	d20 := now.AddDate(0, 0, -20).Format(time.RFC3339)
	d60 := now.AddDate(0, 0, -60).Format(time.RFC3339)
	d30 := now.AddDate(0, 0, -30).Format(time.RFC3339)

	issues := []beads.Issue{
		{ID: "og-1", Title: "20 days", Priority: 3, Status: "open", IssueType: "bug", CreatedAt: d20},
		{ID: "og-2", Title: "60 days", Priority: 4, Status: "open", IssueType: "bug", CreatedAt: d60},
		{ID: "og-3", Title: "30 days", Priority: 3, Status: "open", IssueType: "task", CreatedAt: d30},
	}

	result := filterStaleBacklogIssues(issues, 14, now)

	if len(result) != 3 {
		t.Fatalf("expected 3 stale issues, got %d", len(result))
	}

	// Should be sorted oldest first
	if result[0].Issue.ID != "og-2" {
		t.Errorf("expected oldest (og-2) first, got %s", result[0].Issue.ID)
	}
	if result[1].Issue.ID != "og-3" {
		t.Errorf("expected second oldest (og-3) second, got %s", result[1].Issue.ID)
	}
	if result[2].Issue.ID != "og-1" {
		t.Errorf("expected newest (og-1) third, got %s", result[2].Issue.ID)
	}
}

func TestFilterStaleBacklogIssues_CustomDays(t *testing.T) {
	now := time.Now()
	d10 := now.AddDate(0, 0, -10).Format(time.RFC3339)

	issues := []beads.Issue{
		{ID: "og-1", Title: "10 days old", Priority: 3, Status: "open", IssueType: "bug", CreatedAt: d10},
	}

	// With 14-day threshold, should NOT be included
	result14 := filterStaleBacklogIssues(issues, 14, now)
	if len(result14) != 0 {
		t.Errorf("expected 0 with 14-day threshold, got %d", len(result14))
	}

	// With 7-day threshold, should be included
	result7 := filterStaleBacklogIssues(issues, 7, now)
	if len(result7) != 1 {
		t.Errorf("expected 1 with 7-day threshold, got %d", len(result7))
	}
}

func TestFilterStaleBacklogIssues_DateOnlyFormat(t *testing.T) {
	now := time.Now()
	old := now.AddDate(0, 0, -20).Format("2006-01-02")

	issues := []beads.Issue{
		{ID: "og-1", Title: "Date only format", Priority: 3, Status: "open", IssueType: "bug", CreatedAt: old},
	}

	result := filterStaleBacklogIssues(issues, 14, now)
	if len(result) != 1 {
		t.Errorf("expected 1 issue with date-only format, got %d", len(result))
	}
}
