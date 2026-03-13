package beads

import (
	"os"
	"testing"
)

// Integration tests for the beads RPC client.
// These tests require a running beads daemon and are skipped when unavailable.
//
// To run integration tests:
//   cd to a directory with .beads/ initialized
//   ensure beads daemon is running (bd daemon or bd commands auto-start it)
//   go test -v ./pkg/beads/... -run Integration
//
// Note: These tests use the real beads daemon but are read-only where possible
// to avoid side effects. Tests that create issues clean up after themselves.

// skipIfNoDaemon skips the test if beads daemon is not available.
// Uses the current working directory to find the socket.
func skipIfNoDaemon(t *testing.T) *Client {
	t.Helper()

	// First check if we're in a beads project
	socketPath, err := FindSocketPath("")
	if err != nil {
		t.Skipf("Skipping integration test: %v", err)
	}

	client := NewClient(socketPath)
	if err := client.Connect(); err != nil {
		t.Skipf("Skipping integration test: daemon not available: %v", err)
	}

	return client
}

// TestIntegration_Health tests the health check endpoint.
func TestIntegration_Health(t *testing.T) {
	client := skipIfNoDaemon(t)
	defer client.Close()

	health, err := client.Health()
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("Health status = %q, want %q", health.Status, "healthy")
	}

	if health.Version == "" {
		t.Error("Health version should not be empty")
	}
}

// TestIntegration_Stats tests retrieving beads statistics.
func TestIntegration_Stats(t *testing.T) {
	client := skipIfNoDaemon(t)
	defer client.Close()

	stats, err := client.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	// Just verify we got a valid response - actual counts will vary
	if stats.Summary.TotalIssues < 0 {
		t.Error("TotalIssues should be non-negative")
	}
}

// TestIntegration_List tests listing issues.
func TestIntegration_List(t *testing.T) {
	client := skipIfNoDaemon(t)
	defer client.Close()

	issues, err := client.List(nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Verify we got a valid slice (may be empty)
	if issues == nil {
		t.Error("List should return non-nil slice")
	}
}

// TestIntegration_Ready tests retrieving ready issues.
func TestIntegration_Ready(t *testing.T) {
	client := skipIfNoDaemon(t)
	defer client.Close()

	issues, err := client.Ready(nil)
	if err != nil {
		t.Fatalf("Ready failed: %v", err)
	}

	// Verify we got a valid slice (may be empty)
	if issues == nil {
		t.Error("Ready should return non-nil slice")
	}
}

// TestIntegration_ChildID_Show tests that child IDs (dot notation) are correctly
// handled by the Show method. This is a critical test case because child ID
// parsing failures previously blocked the daemon.
//
// Background: Epic children use IDs like "proj-epic.1", "proj-epic.2.1" etc.
// The dot notation distinguishes them from regular IDs like "proj-abc".
func TestIntegration_ChildID_Show(t *testing.T) {
	client := skipIfNoDaemon(t)
	defer client.Close()

	// First find a child ID from the list
	issues, err := client.List(nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	var childID string
	for _, issue := range issues {
		// Child IDs contain dots after the hash portion
		// Pattern: prefix-hash.number or prefix-hash.number.number
		for i := len(issue.ID) - 1; i >= 0; i-- {
			if issue.ID[i] == '.' {
				childID = issue.ID
				break
			}
			if issue.ID[i] == '-' {
				// No dot found before the last hyphen, not a child ID
				break
			}
		}
		if childID != "" {
			break
		}
	}

	if childID == "" {
		t.Skip("No child ID found in beads - create an epic with children to run this test")
	}

	t.Logf("Testing Show with child ID: %s", childID)

	// This is the critical test - Show should work with child IDs
	issue, err := client.Show(childID)
	if err != nil {
		t.Fatalf("Show failed for child ID %q: %v", childID, err)
	}

	if issue.ID != childID {
		t.Errorf("Show returned ID = %q, want %q", issue.ID, childID)
	}

	// Verify title is non-empty
	if issue.Title == "" {
		t.Error("Show returned empty title for child ID")
	}

	// Log some info for debugging
	t.Logf("Child issue: %s - %s (status: %s, type: %s)",
		issue.ID, issue.Title, issue.Status, issue.IssueType)
}

// TestIntegration_ChildID_CreateAndClose tests creating and closing
// a child issue (epic child). This is a write test that cleans up after itself.
func TestIntegration_ChildID_CreateAndClose(t *testing.T) {
	// Skip if TEST_BEADS_WRITE is not set to avoid accidental writes
	if os.Getenv("TEST_BEADS_WRITE") == "" {
		t.Skip("Set TEST_BEADS_WRITE=1 to run write tests")
	}

	client := skipIfNoDaemon(t)
	defer client.Close()

	// First, find or create an epic to be the parent
	issues, err := client.List(&ListArgs{IssueType: "epic"})
	if err != nil {
		t.Fatalf("List epics failed: %v", err)
	}

	var parentID string
	for _, issue := range issues {
		if issue.Status == "open" {
			parentID = issue.ID
			break
		}
	}

	if parentID == "" {
		t.Skip("No open epic found - create an epic to run this test")
	}

	t.Logf("Creating child of epic: %s", parentID)

	// Create a child issue
	child, err := client.Create(&CreateArgs{
		Parent:    parentID,
		Title:     "Integration test child - will be deleted",
		IssueType: "task",
		Priority:  3,
		Labels:    []string{"test"},
	})
	if err != nil {
		t.Fatalf("Create child failed: %v", err)
	}

	t.Logf("Created child issue: %s", child.ID)

	// Verify the child ID has dot notation
	hasDot := false
	for i := len(child.ID) - 1; i >= 0; i-- {
		if child.ID[i] == '.' {
			hasDot = true
			break
		}
		if child.ID[i] == '-' {
			break
		}
	}
	if !hasDot {
		t.Errorf("Child ID %q should have dot notation", child.ID)
	}

	// Clean up: close the child issue
	err = client.CloseIssue(child.ID, "Integration test cleanup")
	if err != nil {
		t.Errorf("Failed to close child issue %q: %v", child.ID, err)
	}

	t.Logf("Closed child issue: %s", child.ID)
}

// TestIntegration_ChildID_Comments tests adding and reading comments
// on a child ID issue.
func TestIntegration_ChildID_Comments(t *testing.T) {
	client := skipIfNoDaemon(t)
	defer client.Close()

	// Find a child ID
	issues, err := client.List(nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	var childID string
	for _, issue := range issues {
		for i := len(issue.ID) - 1; i >= 0; i-- {
			if issue.ID[i] == '.' {
				childID = issue.ID
				break
			}
			if issue.ID[i] == '-' {
				break
			}
		}
		if childID != "" {
			break
		}
	}

	if childID == "" {
		t.Skip("No child ID found in beads")
	}

	t.Logf("Testing Comments with child ID: %s", childID)

	// Get comments for the child ID
	comments, err := client.Comments(childID)
	if err != nil {
		t.Fatalf("Comments failed for child ID %q: %v", childID, err)
	}

	// Verify we got a valid slice (may be empty)
	if comments == nil {
		t.Error("Comments should return non-nil slice")
	}

	t.Logf("Child issue %s has %d comments", childID, len(comments))
}

// TestIntegration_Fallback_Show tests the CLI fallback for Show.
// This verifies that FallbackShow correctly handles the array response
// from bd show --json, including child IDs.
func TestIntegration_Fallback_Show(t *testing.T) {
	// Check if bd command is available
	socketPath, err := FindSocketPath("")
	if err != nil {
		t.Skipf("Skipping: %v", err)
	}

	// Use the socket path to determine a valid issue ID
	client := NewClient(socketPath)
	if err := client.Connect(); err != nil {
		t.Skipf("Skipping: daemon not available: %v", err)
	}
	defer client.Close()

	issues, err := client.List(&ListArgs{Limit: IntPtr(1)})
	if err != nil || len(issues) == 0 {
		t.Skip("No issues available for fallback test")
	}

	testID := issues[0].ID
	t.Logf("Testing FallbackShow with ID: %s", testID)

	// Test the fallback
	issue, err := FallbackShow(testID, "")
	if err != nil {
		t.Fatalf("FallbackShow failed: %v", err)
	}

	if issue.ID != testID {
		t.Errorf("FallbackShow returned ID = %q, want %q", issue.ID, testID)
	}
}

// TestIntegration_Fallback_Show_ChildID specifically tests FallbackShow
// with a child ID to ensure the CLI fallback handles dot notation correctly.
func TestIntegration_Fallback_Show_ChildID(t *testing.T) {
	socketPath, err := FindSocketPath("")
	if err != nil {
		t.Skipf("Skipping: %v", err)
	}

	client := NewClient(socketPath)
	if err := client.Connect(); err != nil {
		t.Skipf("Skipping: daemon not available: %v", err)
	}
	defer client.Close()

	// Find a child ID
	issues, err := client.List(nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	var childID string
	for _, issue := range issues {
		for i := len(issue.ID) - 1; i >= 0; i-- {
			if issue.ID[i] == '.' {
				childID = issue.ID
				break
			}
			if issue.ID[i] == '-' {
				break
			}
		}
		if childID != "" {
			break
		}
	}

	if childID == "" {
		t.Skip("No child ID found in beads")
	}

	t.Logf("Testing FallbackShow with child ID: %s", childID)

	// This is the critical fallback test for child IDs
	issue, err := FallbackShow(childID, "")
	if err != nil {
		t.Fatalf("FallbackShow failed for child ID %q: %v", childID, err)
	}

	if issue.ID != childID {
		t.Errorf("FallbackShow returned ID = %q, want %q", issue.ID, childID)
	}

	t.Logf("FallbackShow returned: %s - %s", issue.ID, issue.Title)
}

// TestIntegration_MultiLevelChildID tests handling of deeply nested child IDs
// (e.g., "proj-epic.1.2" for grandchildren).
func TestIntegration_MultiLevelChildID(t *testing.T) {
	client := skipIfNoDaemon(t)
	defer client.Close()

	// Search for multi-level child IDs
	issues, err := client.List(nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	var multiLevelID string
	for _, issue := range issues {
		// Count dots after the last hyphen
		dotCount := 0
		foundHyphen := false
		for i := len(issue.ID) - 1; i >= 0; i-- {
			if issue.ID[i] == '.' {
				dotCount++
			} else if issue.ID[i] == '-' {
				foundHyphen = true
				break
			}
		}
		if foundHyphen && dotCount >= 2 {
			multiLevelID = issue.ID
			break
		}
	}

	if multiLevelID == "" {
		t.Skip("No multi-level child ID (e.g., proj.1.2) found in beads")
	}

	t.Logf("Testing Show with multi-level child ID: %s", multiLevelID)

	issue, err := client.Show(multiLevelID)
	if err != nil {
		t.Fatalf("Show failed for multi-level child ID %q: %v", multiLevelID, err)
	}

	if issue.ID != multiLevelID {
		t.Errorf("Show returned ID = %q, want %q", issue.ID, multiLevelID)
	}

	t.Logf("Multi-level child: %s - %s", issue.ID, issue.Title)
}
