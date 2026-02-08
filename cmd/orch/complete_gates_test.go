package main

import (
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/verify"
)

func TestRequiresDesignDecomposition(t *testing.T) {
	tests := []struct {
		name  string
		skill string
		want  bool
	}{
		{name: "design session", skill: "design-session", want: true},
		{name: "architect", skill: "architect", want: true},
		{name: "mixed case", skill: "  ArChItEcT  ", want: true},
		{name: "feature impl", skill: "feature-impl", want: false},
		{name: "empty", skill: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := requiresDesignDecomposition(tt.skill)
			if got != tt.want {
				t.Fatalf("requiresDesignDecomposition(%q) = %v, want %v", tt.skill, got, tt.want)
			}
		})
	}
}

func TestBuildDesignDecompositionIssueTitle(t *testing.T) {
	item := verify.DesignActionItem{Section: "Components to Build", Text: "`WorkInProgressSection`"}
	title := buildDesignDecompositionIssueTitle(item)

	if !strings.HasPrefix(title, "Design follow-up: ") {
		t.Fatalf("unexpected prefix in title: %q", title)
	}
	if strings.Contains(title, "`") {
		t.Fatalf("title should strip markdown backticks, got %q", title)
	}

	longText := strings.Repeat("a", 200)
	longTitle := buildDesignDecompositionIssueTitle(verify.DesignActionItem{Text: longText})
	if len(longTitle) > 140 {
		t.Fatalf("title length should be capped at 140, got %d", len(longTitle))
	}
}
