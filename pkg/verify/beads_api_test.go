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
