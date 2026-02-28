package main

import (
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// TestOrphanItemFromIssue verifies conversion from beads issue to OrphanItem.
func TestOrphanItemFromIssue(t *testing.T) {
	issue := beads.Issue{
		ID:        "orch-go-abc1",
		Title:     "architect: design new caching layer",
		Priority:  2,
		IssueType: "task",
		Status:    "closed",
		Labels:    []string{"skill:architect"},
		CreatedAt: "2026-02-20T10:00:00-08:00",
		ClosedAt:  "2026-02-21T14:00:00-08:00",
	}

	item := orphanItemFromIssue(issue)

	if item.ID != "orch-go-abc1" {
		t.Errorf("expected ID orch-go-abc1, got %s", item.ID)
	}
	if item.Title != "architect: design new caching layer" {
		t.Errorf("expected correct title, got %s", item.Title)
	}
	if item.Age == "" {
		t.Error("expected age to be populated")
	}
}

// TestHasImplementationFollowUp verifies detection of follow-up issues.
func TestHasImplementationFollowUp(t *testing.T) {
	tests := []struct {
		name         string
		architectID  string
		allIssues    []beads.Issue
		expectFollow bool
	}{
		{
			name:        "title pattern match",
			architectID: "orch-go-abc1",
			allIssues: []beads.Issue{
				{ID: "orch-go-def2", Title: "Implement caching layer (from architect orch-go-abc1)"},
			},
			expectFollow: true,
		},
		{
			name:        "no match",
			architectID: "orch-go-abc1",
			allIssues: []beads.Issue{
				{ID: "orch-go-def2", Title: "Some unrelated task"},
			},
			expectFollow: false,
		},
		{
			name:        "case insensitive match",
			architectID: "orch-go-abc1",
			allIssues: []beads.Issue{
				{ID: "orch-go-def2", Title: "Implement caching layer (From Architect orch-go-abc1)"},
			},
			expectFollow: true,
		},
		{
			name:        "partial ID should not match different issue",
			architectID: "orch-go-abc1",
			allIssues: []beads.Issue{
				{ID: "orch-go-def2", Title: "Fix something (from architect orch-go-xyz9)"},
			},
			expectFollow: false,
		},
		{
			name:        "empty issues list",
			architectID: "orch-go-abc1",
			allIssues:   nil,
			expectFollow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasImplementationFollowUp(tt.architectID, tt.allIssues)
			if result != tt.expectFollow {
				t.Errorf("hasImplementationFollowUp(%q) = %v, want %v", tt.architectID, result, tt.expectFollow)
			}
		})
	}
}

// TestFormatOrphansList verifies the display output for orphan items.
func TestFormatOrphansList(t *testing.T) {
	t.Run("no orphans", func(t *testing.T) {
		output := formatOrphansList(nil)
		if !containsStr(output, "No orphaned architect") {
			t.Errorf("expected no-orphans message, got %q", output)
		}
	})

	t.Run("with orphans", func(t *testing.T) {
		items := []OrphanItem{
			{
				ID:    "orch-go-abc1",
				Title: "architect: design caching layer",
				Age:   "7d",
			},
			{
				ID:            "orch-go-def2",
				Title:         "architect: redesign spawn flow",
				Age:           "3d",
				SynthesisTLDR: "Designed new spawn pipeline with retry logic",
			},
		}

		output := formatOrphansList(items)

		// Should contain both IDs
		if !containsStr(output, "orch-go-abc1") {
			t.Error("expected output to contain orch-go-abc1")
		}
		if !containsStr(output, "orch-go-def2") {
			t.Error("expected output to contain orch-go-def2")
		}

		// Should contain the TLDR for the second item
		if !containsStr(output, "Designed new spawn pipeline") {
			t.Error("expected output to contain synthesis TLDR")
		}

		// Should contain the no-implementation marker
		if !containsStr(output, "No implementation issues found") {
			t.Error("expected output to contain orphan marker")
		}

		// Should contain age
		if !containsStr(output, "7d") {
			t.Error("expected output to contain age")
		}
	})
}

// TestHumanAgeSinceTime verifies age formatting from time values.
func TestHumanAgeSinceTime(t *testing.T) {
	// humanAge is already tested in review_triage_test.go
	// Just verify it works with durations we'll use
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Minute, "30m"},
		{5 * time.Hour, "5h"},
		{3 * 24 * time.Hour, "3d"},
		{14 * 24 * time.Hour, "14d"},
	}

	for _, tt := range tests {
		result := humanAge(tt.duration)
		if result != tt.expected {
			t.Errorf("humanAge(%v) = %q, want %q", tt.duration, result, tt.expected)
		}
	}
}

// TestIsArchitectBeadsIssue verifies detection of architect issues.
func TestIsArchitectBeadsIssue(t *testing.T) {
	tests := []struct {
		name  string
		issue beads.Issue
		want  bool
	}{
		{
			name:  "skill:architect label",
			issue: beads.Issue{Title: "some task", Labels: []string{"skill:architect"}},
			want:  true,
		},
		{
			name:  "architect: in title",
			issue: beads.Issue{Title: "architect: design caching layer"},
			want:  true,
		},
		{
			name:  "Architect: capitalized in title",
			issue: beads.Issue{Title: "Architect: review spawn flow"},
			want:  true,
		},
		{
			name:  "no architect marker",
			issue: beads.Issue{Title: "implement caching layer", Labels: []string{"skill:feature-impl"}},
			want:  false,
		},
		{
			name:  "label among others",
			issue: beads.Issue{Title: "review hotspot", Labels: []string{"area:spawn", "skill:architect"}},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isArchitectBeadsIssue(tt.issue)
			if got != tt.want {
				t.Errorf("isArchitectBeadsIssue() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestReviewOrphansCommandExists verifies the orphans subcommand is registered.
func TestReviewOrphansCommandExists(t *testing.T) {
	orphansCmd, _, err := reviewCmd.Find([]string{"orphans"})
	if err != nil || orphansCmd == nil {
		t.Fatal("Expected 'orphans' subcommand to exist on review")
	}

	// Check --create-follow-up flag exists
	cfFlag := orphansCmd.Flags().Lookup("create-follow-up")
	if cfFlag == nil {
		t.Error("Expected --create-follow-up flag on review orphans command")
	}
}
