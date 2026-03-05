package verify

import (
	"os"
	"testing"
)

func TestUpdateIssueStatus(t *testing.T) {
	// Use test issue ID from environment or skip
	issueID := os.Getenv("TEST_BEADS_ISSUE")
	if issueID == "" {
		t.Skip("Set TEST_BEADS_ISSUE environment variable to run this test")
	}

	// Test updating to in_progress
	err := UpdateIssueStatus(issueID, "in_progress", "")
	if err != nil {
		t.Errorf("UpdateIssueStatus to in_progress failed: %v", err)
	}

	// Verify status changed
	issue, err := GetIssue(issueID, "")
	if err != nil {
		t.Errorf("GetIssue failed: %v", err)
	}
	if issue.Status != "in_progress" {
		t.Errorf("Expected status in_progress, got %s", issue.Status)
	}

	// Clean up: revert to open
	err = UpdateIssueStatus(issueID, "open", "")
	if err != nil {
		t.Errorf("UpdateIssueStatus to open failed: %v", err)
	}
}
