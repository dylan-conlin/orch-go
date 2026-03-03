package main

import (
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

func TestCryptoRandSelection_FewerThanN(t *testing.T) {
	issues := []beads.Issue{
		{ID: "a", Title: "one"},
	}
	selected, err := cryptoRandSelection(issues, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selected) != 1 {
		t.Errorf("expected 1 issue, got %d", len(selected))
	}
}

func TestCryptoRandSelection_ExactlyN(t *testing.T) {
	issues := []beads.Issue{
		{ID: "a", Title: "one"},
		{ID: "b", Title: "two"},
	}
	selected, err := cryptoRandSelection(issues, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selected) != 2 {
		t.Errorf("expected 2 issues, got %d", len(selected))
	}
}

func TestCryptoRandSelection_MoreThanN(t *testing.T) {
	issues := make([]beads.Issue, 20)
	for i := range issues {
		issues[i] = beads.Issue{ID: string(rune('a' + i))}
	}
	selected, err := cryptoRandSelection(issues, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selected) != 2 {
		t.Errorf("expected 2 issues, got %d", len(selected))
	}
	// Verify selected are from original pool
	idSet := make(map[string]bool)
	for _, issue := range issues {
		idSet[issue.ID] = true
	}
	for _, s := range selected {
		if !idSet[s.ID] {
			t.Errorf("selected issue %s not in original pool", s.ID)
		}
	}
	// Verify no duplicates
	if selected[0].ID == selected[1].ID {
		t.Error("selected duplicate issues")
	}
}

func TestCryptoRandSelection_Empty(t *testing.T) {
	selected, err := cryptoRandSelection(nil, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selected) != 0 {
		t.Errorf("expected 0 issues, got %d", len(selected))
	}
}

func TestHasLabel(t *testing.T) {
	labels := []string{"area:cli", "audit:deep-review", "effort:medium"}
	if !hasLabel(labels, "audit:deep-review") {
		t.Error("expected to find audit:deep-review")
	}
	if hasLabel(labels, "missing-label") {
		t.Error("should not find missing-label")
	}
	if hasLabel(nil, "anything") {
		t.Error("should not find in nil slice")
	}
}

func TestAuditPlistContent(t *testing.T) {
	content := auditPlistContent("/usr/local/bin/orch", "/home/user/project", "/home/user/.orch/audit.log")
	if content == "" {
		t.Fatal("plist content should not be empty")
	}
	// Verify key elements
	checks := []string{
		"com.orch.audit-select",
		"/usr/local/bin/orch",
		"<string>audit</string>",
		"<string>select</string>",
		"<key>Weekday</key>",
		"<integer>1</integer>",        // Monday
		"/home/user/project",          // WorkingDirectory
		"/home/user/.orch/audit.log",  // Log path
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("plist missing: %s", check)
		}
	}
}

// TestRecentClosedFiltering tests the time-window and label filtering logic
// without calling bd (by testing the filter logic directly).
func TestRecentClosedFiltering(t *testing.T) {
	now := time.Now()
	recent := now.Add(-24 * time.Hour).Format(time.RFC3339Nano)
	old := now.Add(-14 * 24 * time.Hour).Format(time.RFC3339Nano)
	window := 7 * 24 * time.Hour
	cutoff := now.Add(-window)

	issues := []beads.Issue{
		{ID: "recent-1", ClosedAt: recent, Labels: nil},
		{ID: "recent-2", ClosedAt: recent, Labels: []string{"audit:deep-review"}},
		{ID: "old-1", ClosedAt: old, Labels: nil},
		{ID: "no-date", ClosedAt: "", Labels: nil},
	}

	// Simulate the filtering logic from recentClosedIssues
	var filtered []beads.Issue
	for _, issue := range issues {
		if issue.ClosedAt == "" {
			continue
		}
		closedAt, err := time.Parse(time.RFC3339Nano, issue.ClosedAt)
		if err != nil {
			closedAt, err = time.Parse(time.RFC3339, issue.ClosedAt)
			if err != nil {
				continue
			}
		}
		if closedAt.After(cutoff) && !hasLabel(issue.Labels, auditLabel) {
			filtered = append(filtered, issue)
		}
	}

	if len(filtered) != 1 {
		t.Fatalf("expected 1 eligible issue, got %d", len(filtered))
	}
	if filtered[0].ID != "recent-1" {
		t.Errorf("expected recent-1, got %s", filtered[0].ID)
	}
}
