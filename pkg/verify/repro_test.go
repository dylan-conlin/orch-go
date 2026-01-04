package verify

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

func TestIsBugType(t *testing.T) {
	tests := []struct {
		name      string
		issueType string
		want      bool
	}{
		{name: "bug type", issueType: "bug", want: true},
		{name: "Bug uppercase", issueType: "Bug", want: true},
		{name: "BUG all caps", issueType: "BUG", want: true},
		{name: "defect type", issueType: "defect", want: true},
		{name: "bugfix type", issueType: "bugfix", want: true},
		{name: "feature type", issueType: "feature", want: false},
		{name: "task type", issueType: "task", want: false},
		{name: "investigation type", issueType: "investigation", want: false},
		{name: "epic type", issueType: "epic", want: false},
		{name: "empty type", issueType: "", want: false},
		{name: "with whitespace", issueType: "  bug  ", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsBugType(tt.issueType)
			if got != tt.want {
				t.Errorf("IsBugType(%q) = %v, want %v", tt.issueType, got, tt.want)
			}
		})
	}
}

func TestExtractReproFromIssue(t *testing.T) {
	tests := []struct {
		name        string
		issue       *beads.Issue
		wantRepro   string
		wantHasRepo bool
	}{
		{
			name: "repro in markdown heading",
			issue: &beads.Issue{
				Description: `## Problem
Something is broken.

## Reproduction
1. Run command X
2. See error Y

## Expected
It should work.`,
			},
			wantRepro:   "1. Run command X\n2. See error Y",
			wantHasRepo: true,
		},
		{
			name: "repro steps heading",
			issue: &beads.Issue{
				Description: `## Steps to Reproduce
- Click button
- Wait for loading
- See crash`,
			},
			wantRepro:   "- Click button\n- Wait for loading\n- See crash",
			wantHasRepo: true,
		},
		{
			name: "bold repro marker",
			issue: &beads.Issue{
				Description: `**Problem:** App crashes

**Reproduction:** Run 'orch status' and observe 27 active agents shown when only 4 are running.

**Expected:** Count should match actual running agents.`,
			},
			wantRepro:   "Run 'orch status' and observe 27 active agents shown when only 4 are running.",
			wantHasRepo: true,
		},
		{
			name: "code block repro extracts command",
			issue: &beads.Issue{
				Description: "To reproduce:\n```bash\norch status\n```\nShows wrong count.",
			},
			// The "To reproduce:" pattern matches first and extracts everything after
			wantRepro:   "```bash\norch status\n```\nShows wrong count.",
			wantHasRepo: true,
		},
		{
			name: "to reproduce pattern",
			issue: &beads.Issue{
				Description: `To reproduce: run 'make test' and observe flaky failures on TestX.

Expected: consistent passes.`,
			},
			wantRepro:   "run 'make test' and observe flaky failures on TestX.",
			wantHasRepo: true,
		},
		{
			name: "no explicit repro - uses description",
			issue: &beads.Issue{
				Title:       "Dashboard shows wrong count",
				Description: "The active agent count is incorrect.",
			},
			wantRepro:   "The active agent count is incorrect.",
			wantHasRepo: true,
		},
		{
			name:        "nil issue",
			issue:       nil,
			wantRepro:   "",
			wantHasRepo: false,
		},
		{
			name: "empty description",
			issue: &beads.Issue{
				Title:       "Some bug",
				Description: "",
			},
			wantRepro:   "",
			wantHasRepo: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRepro, gotHasRepro := ExtractReproFromIssue(tt.issue)
			if gotHasRepro != tt.wantHasRepo {
				t.Errorf("ExtractReproFromIssue() hasRepro = %v, want %v", gotHasRepro, tt.wantHasRepo)
			}
			if tt.wantHasRepo && gotRepro != tt.wantRepro {
				t.Errorf("ExtractReproFromIssue() repro = %q, want %q", gotRepro, tt.wantRepro)
			}
		})
	}
}
