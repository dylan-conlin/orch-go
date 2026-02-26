package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// TestTriageItemFromIssue verifies conversion from beads issue to TriageItem.
func TestTriageItemFromIssue(t *testing.T) {
	issue := beads.Issue{
		ID:        "orch-go-1173",
		Title:     "opencode attach lacks --model flag",
		Priority:  2,
		IssueType: "bug",
		Status:    "open",
		Labels:    []string{"triage:review"},
		CreatedAt: "2026-02-21T07:14:04.47563-08:00",
	}

	item := triageItemFromIssue(issue)

	if item.ID != "orch-go-1173" {
		t.Errorf("expected ID orch-go-1173, got %s", item.ID)
	}
	if item.Title != "opencode attach lacks --model flag" {
		t.Errorf("expected correct title, got %s", item.Title)
	}
	if item.Priority != 2 {
		t.Errorf("expected priority 2, got %d", item.Priority)
	}
	if item.IssueType != "bug" {
		t.Errorf("expected type bug, got %s", item.IssueType)
	}
}

// TestFormatTriageList verifies the display output for triage items.
func TestFormatTriageList(t *testing.T) {
	items := []TriageItem{
		{ID: "orch-go-1173", Title: "opencode attach lacks --model flag", Priority: 2, IssueType: "bug", Age: "5d"},
		{ID: "orch-go-1260", Title: "skillc: fix test signature mismatches", Priority: 3, IssueType: "bug", Age: "1d"},
	}

	output := formatTriageList(items)

	if output == "" {
		t.Fatal("expected non-empty output")
	}

	// Should contain both IDs
	if !containsStr(output, "orch-go-1173") {
		t.Error("expected output to contain orch-go-1173")
	}
	if !containsStr(output, "orch-go-1260") {
		t.Error("expected output to contain orch-go-1260")
	}

	// Should contain priority indicators
	if !containsStr(output, "P2") {
		t.Error("expected output to contain P2")
	}
	if !containsStr(output, "P3") {
		t.Error("expected output to contain P3")
	}
}

// TestFormatTriageSummary verifies the hygiene nudge summary.
func TestFormatTriageSummary(t *testing.T) {
	t.Run("no items", func(t *testing.T) {
		summary := formatTriageSummary(0)
		if summary != "" {
			t.Errorf("expected empty summary for 0 items, got %q", summary)
		}
	})

	t.Run("with items", func(t *testing.T) {
		summary := formatTriageSummary(12)
		if !containsStr(summary, "12") {
			t.Error("expected summary to contain count 12")
		}
		if !containsStr(summary, "triage:review") {
			t.Error("expected summary to contain triage:review")
		}
		if !containsStr(summary, "orch review triage") {
			t.Error("expected summary to contain the command hint")
		}
	})
}

// TestReviewTriageCommandExists verifies the triage subcommand is registered.
func TestReviewTriageCommandExists(t *testing.T) {
	triageCmd, _, err := reviewCmd.Find([]string{"triage"})
	if err != nil || triageCmd == nil {
		t.Fatal("Expected 'triage' subcommand to exist on review")
	}

	// Check --non-interactive flag exists
	niFlag := triageCmd.Flags().Lookup("non-interactive")
	if niFlag == nil {
		t.Error("Expected --non-interactive flag on review triage command")
	}
}

// containsStr is a test helper (avoids importing strings in tests where it's trivially used).
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
