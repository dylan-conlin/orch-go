// Package verify provides verification helpers for agent completion.
package verify

import (
	"testing"
)

// TestEpicChildInfo tests the EpicChildInfo struct.
func TestEpicChildInfo(t *testing.T) {
	// Basic struct test
	child := EpicChildInfo{
		ID:     "test-123",
		Title:  "Test task",
		Status: "open",
	}

	if child.ID != "test-123" {
		t.Errorf("ID = %q, want %q", child.ID, "test-123")
	}
	if child.Title != "Test task" {
		t.Errorf("Title = %q, want %q", child.Title, "Test task")
	}
	if child.Status != "open" {
		t.Errorf("Status = %q, want %q", child.Status, "open")
	}
}

// Note: GetOpenEpicChildren requires integration testing with real beads database
// as it relies on FallbackListByParent which makes CLI calls.
// The filtering logic is straightforward - it filters out "closed", "deferred", and "tombstone" statuses.
// Integration tests should be added when a mock beads client is available.

func TestExtractParentID(t *testing.T) {
	tests := []struct {
		name     string
		issueID  string
		expected string
	}{
		{
			name:     "simple child ID",
			issueID:  "orch-go-erdw.4",
			expected: "orch-go-erdw",
		},
		{
			name:     "double digit child number",
			issueID:  "proj-abc.12",
			expected: "proj-abc",
		},
		{
			name:     "triple digit child number",
			issueID:  "proj-xyz.123",
			expected: "proj-xyz",
		},
		{
			name:     "not a child - no dot",
			issueID:  "orch-go-21234",
			expected: "",
		},
		{
			name:     "not a child - non-numeric suffix",
			issueID:  "orch-go-erdw.abc",
			expected: "",
		},
		{
			name:     "not a child - mixed suffix",
			issueID:  "orch-go-erdw.4a",
			expected: "",
		},
		{
			name:     "grandchild ID",
			issueID:  "proj-epic.1.2",
			expected: "proj-epic.1",
		},
		{
			name:     "empty string",
			issueID:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractParentID(tt.issueID)
			if result != tt.expected {
				t.Errorf("ExtractParentID(%q) = %q, want %q", tt.issueID, result, tt.expected)
			}
		})
	}
}
