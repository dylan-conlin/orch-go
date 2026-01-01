package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

func TestExtractIssueRefs(t *testing.T) {
	// Create a mock issue map
	issueMap := map[string]beads.Issue{
		"orch-go-f9l5": {ID: "orch-go-f9l5", Title: "Test issue 1"},
		"f9l5":         {ID: "orch-go-f9l5", Title: "Test issue 1"}, // short ID mapping
		"kb-cli-abc1":  {ID: "kb-cli-abc1", Title: "Test issue 2"},
		"abc1":         {ID: "kb-cli-abc1", Title: "Test issue 2"}, // short ID mapping
		"orch-go-gxwu": {ID: "orch-go-gxwu", Title: "Test issue 3"},
		"gxwu":         {ID: "orch-go-gxwu", Title: "Test issue 3"}, // short ID mapping
	}

	tests := []struct {
		name     string
		message  string
		expected []string
	}{
		{
			name:     "fix commit with full issue ID",
			message:  "fix: resolve auth bug orch-go-f9l5",
			expected: []string{"orch-go-f9l5"},
		},
		{
			name:     "closes marker with full ID",
			message:  "fix: resolve bug closes #orch-go-f9l5",
			expected: []string{"orch-go-f9l5"},
		},
		{
			name:     "fixes marker",
			message:  "chore: cleanup fixes orch-go-gxwu",
			expected: []string{"orch-go-gxwu"},
		},
		{
			name:     "resolves marker",
			message:  "docs: update readme resolves kb-cli-abc1",
			expected: []string{"kb-cli-abc1"},
		},
		{
			name:     "no issue reference",
			message:  "chore: update dependencies",
			expected: nil,
		},
		{
			name:     "non-existent issue ID",
			message:  "fix: something orch-go-xxxx",
			expected: nil,
		},
		{
			name:     "case insensitive closes",
			message:  "fix: resolve Closes ORCH-GO-F9L5",
			expected: []string{"orch-go-f9l5"},
		},
		{
			name:     "issue ID in conventional commit scope",
			message:  "fix(orch-go-f9l5): resolve the bug",
			expected: []string{"orch-go-f9l5"},
		},
		{
			name:     "feat commit without explicit close marker should NOT match",
			message:  "feat: implement new feature (orch-go-f9l5)",
			expected: nil,
		},
		{
			name:     "create epic should NOT close the epic",
			message:  "feat: Create epic orch-go-f9l5 for consolidation",
			expected: nil,
		},
		{
			name:     "chore update issue should NOT close",
			message:  "chore: update issue orch-go-f9l5 priority",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractIssueRefs(tt.message, issueMap)

			if len(got) != len(tt.expected) {
				t.Errorf("extractIssueRefs() returned %d refs, want %d\ngot: %v\nwant: %v",
					len(got), len(tt.expected), got, tt.expected)
				return
			}

			for i, ref := range got {
				if ref != tt.expected[i] {
					t.Errorf("extractIssueRefs()[%d] = %s, want %s", i, ref, tt.expected[i])
				}
			}
		})
	}
}

func TestExtractIssueRefs_NoDuplicates(t *testing.T) {
	issueMap := map[string]beads.Issue{
		"orch-go-f9l5": {ID: "orch-go-f9l5", Title: "Test issue"},
		"f9l5":         {ID: "orch-go-f9l5", Title: "Test issue"}, // short ID mapping
	}

	// Message with explicit close marker AND issue ID should only return once
	message := "fix: resolve issue closes orch-go-f9l5 and fixes #orch-go-f9l5"
	got := extractIssueRefs(message, issueMap)

	if len(got) != 1 {
		t.Errorf("Expected 1 unique issue ref, got %d: %v", len(got), got)
	}

	if len(got) > 0 && got[0] != "orch-go-f9l5" {
		t.Errorf("Expected orch-go-f9l5, got %s", got[0])
	}
}

func TestExtractIssueRefs_ExcludesCreation(t *testing.T) {
	issueMap := map[string]beads.Issue{
		"orch-go-6uli": {ID: "orch-go-6uli", Title: "Epic: Test"},
	}

	// Commits that create or update issues should NOT trigger close
	excludeMessages := []string{
		"feat: Create epic orch-go-6uli for consolidation",
		"chore: create issue orch-go-6uli",
		"feat: add epic orch-go-6uli",
		"chore: update epic orch-go-6uli status",
	}

	for _, msg := range excludeMessages {
		got := extractIssueRefs(msg, issueMap)
		if len(got) > 0 {
			t.Errorf("Message %q should NOT match, but got: %v", msg, got)
		}
	}
}

func TestTruncateSyncString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly ten", 11, "exactly ten"},
		{"this is a long string that needs truncation", 20, "this is a long st..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		got := truncateSyncString(tt.input, tt.maxLen)
		if got != tt.expected {
			t.Errorf("truncateSyncString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
		}
	}
}
