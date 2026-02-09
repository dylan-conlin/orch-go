package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

func TestDetectBacklogHygieneViolations(t *testing.T) {
	tests := []struct {
		name      string
		issue     beads.Issue
		wantRules []string
	}{
		{
			name: "empty description",
			issue: beads.Issue{
				ID:          "orch-go-1",
				Title:       "Implement recurring check",
				Description: "",
			},
			wantRules: []string{backlogRuleEmptyDescription},
		},
		{
			name: "placeholder description",
			issue: beads.Issue{
				ID:          "orch-go-2",
				Title:       "Implement recurring check",
				Description: "TBD",
			},
			wantRules: []string{backlogRulePlaceholderDesc},
		},
		{
			name: "metadata prefixed truncated title",
			issue: beads.Issue{
				ID:          "orch-go-3",
				Title:       "[orch-go] feature-impl: Enrich in_progress items in work-graph tree wit...",
				Description: "Needs a real description",
			},
			wantRules: []string{backlogRuleMetadataTruncated},
		},
		{
			name: "placeholder title",
			issue: beads.Issue{
				ID:          "orch-go-4",
				Title:       "TODO",
				Description: "Implement this later",
			},
			wantRules: []string{backlogRulePlaceholderTitle},
		},
		{
			name: "clean issue",
			issue: beads.Issue{
				ID:          "orch-go-5",
				Title:       "Add recurring backlog hygiene command",
				Description: "Add doctor mode to scan backlog issue quality.",
			},
			wantRules: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := detectBacklogHygieneViolations(tt.issue)

			if len(violations) != len(tt.wantRules) {
				t.Fatalf("len(violations) = %d, want %d", len(violations), len(tt.wantRules))
			}

			for _, rule := range tt.wantRules {
				if !containsRule(violations, rule) {
					t.Fatalf("missing rule %q in %+v", rule, violations)
				}
			}
		})
	}
}

func TestEvaluateBacklogHygiene(t *testing.T) {
	issues := []beads.Issue{
		{
			ID:          "orch-go-10",
			Title:       "Clean title",
			Description: "Clear description",
			Status:      "open",
		},
		{
			ID:          "orch-go-11",
			Title:       "TODO",
			Description: "",
			Status:      "open",
		},
		{
			ID:          "orch-go-12",
			Title:       "Closed issue should be ignored",
			Description: "",
			Status:      "closed",
		},
	}

	report := evaluateBacklogHygiene(issues)

	if report.CheckedCount != 2 {
		t.Fatalf("CheckedCount = %d, want 2", report.CheckedCount)
	}
	if report.IssueCount != 1 {
		t.Fatalf("IssueCount = %d, want 1", report.IssueCount)
	}
	if report.Healthy {
		t.Fatal("Healthy = true, want false")
	}

	if report.RuleCounts[backlogRuleEmptyDescription] != 1 {
		t.Fatalf("empty-description count = %d, want 1", report.RuleCounts[backlogRuleEmptyDescription])
	}
	if report.RuleCounts[backlogRulePlaceholderTitle] != 1 {
		t.Fatalf("placeholder-title count = %d, want 1", report.RuleCounts[backlogRulePlaceholderTitle])
	}
}

func TestHasMetadataTruncatedTitle(t *testing.T) {
	tests := []struct {
		title string
		want  bool
	}{
		{title: "[orch-go] feature-impl: Implement backlog scanner...", want: true},
		{title: "[orch-go] feature-impl: Implement backlog scanner", want: false},
		{title: "Implement backlog scanner...", want: false},
		{title: "[orch-go] feature-impl: Implement backlog scanner…", want: true},
	}

	for _, tt := range tests {
		if got := hasMetadataTruncatedTitle(tt.title); got != tt.want {
			t.Fatalf("hasMetadataTruncatedTitle(%q) = %v, want %v", tt.title, got, tt.want)
		}
	}
}

func containsRule(violations []BacklogHygieneViolation, rule string) bool {
	for _, violation := range violations {
		if violation.Rule == rule {
			return true
		}
	}
	return false
}
