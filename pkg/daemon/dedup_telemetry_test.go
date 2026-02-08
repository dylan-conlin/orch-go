package daemon

import (
	"testing"
	"time"
)

// MockEventLogger captures logged events for testing
type MockEventLogger struct {
	Events []map[string]interface{}
}

func (m *MockEventLogger) LogDedupBlocked(data interface{}) error {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	m.Events = append(m.Events, dataMap)
	return nil
}

// TestSpawnedTrackerEmitsEvent verifies that dedup event is emitted when SpawnedIssueTracker blocks.
func TestSpawnedTrackerEmitsEvent(t *testing.T) {
	// Setup: Create daemon with mock event logger
	mockLogger := &MockEventLogger{Events: []map[string]interface{}{}}
	d := NewWithConfig(Config{
		Label:   "triage:ready",
		Verbose: false,
	})
	d.SetEventLogger(mockLogger)

	// Setup: Mark an issue as spawned
	testIssueID := "test-issue-123"
	d.SpawnedIssues.MarkSpawned(testIssueID)

	// Setup: Mock listIssuesFunc to return the spawned issue
	d.listIssuesFunc = func() ([]Issue, error) {
		return []Issue{
			{
				ID:          testIssueID,
				Title:       "Test Issue",
				Description: "Test description",
				Status:      "open",
				IssueType:   "bug",
				Priority:    2,
				Labels:      []string{"triage:ready"},
			},
		}, nil
	}

	// Execute: Call NextIssueExcluding which should skip the spawned issue
	issue, err := d.NextIssueExcluding(nil)

	// Verify: No issue returned (it was skipped)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue != nil {
		t.Fatalf("expected nil issue (should be skipped), got: %v", issue)
	}

	// Verify: Event was logged
	if len(mockLogger.Events) != 1 {
		t.Fatalf("expected 1 dedup event, got %d", len(mockLogger.Events))
	}

	event := mockLogger.Events[0]

	// Verify: Event contains correct data
	if event["beads_id"] != testIssueID {
		t.Errorf("expected beads_id=%s, got %v", testIssueID, event["beads_id"])
	}
	if event["dedup_layer"] != "spawned_tracker" {
		t.Errorf("expected dedup_layer=spawned_tracker, got %v", event["dedup_layer"])
	}
	if event["reason"] == nil || event["reason"] == "" {
		t.Errorf("expected non-empty reason, got %v", event["reason"])
	}
}

// TestSessionDedupEmitsEvent verifies that dedup event is emitted when session dedup blocks.
func TestSessionDedupEmitsEvent(t *testing.T) {
	// Setup: Create daemon with mock event logger
	mockLogger := &MockEventLogger{Events: []map[string]interface{}{}}
	d := NewWithConfig(Config{
		Label:   "triage:ready",
		Verbose: false,
	})
	d.SetEventLogger(mockLogger)

	testIssueID := "test-issue-456"

	// Setup: Mock listIssuesFunc to return a ready issue
	d.listIssuesFunc = func() ([]Issue, error) {
		return []Issue{
			{
				ID:          testIssueID,
				Title:       "Test Issue",
				Description: "Test description",
				Status:      "open",
				IssueType:   "feature",
				Priority:    2,
				Labels:      []string{"triage:ready"},
			},
		}, nil
	}

	// Setup: Mock HasExistingSessionForBeadsID to return true (session exists)
	// We can't easily mock this without refactoring, but we can test the code path
	// by checking that OnceExcluding calls HasExistingSessionForBeadsID.
	// For this test, we'll verify the integration by checking the skip behavior.

	// Note: This test is limited because HasExistingSessionForBeadsID is a package-level
	// function that calls the real OpenCode API. To properly test this, we'd need to
	// refactor HasExistingSessionForBeadsID to be injectable or create an integration test.

	// For now, we'll skip this test and document the limitation.
	t.Skip("Session dedup requires OpenCode API integration - tested manually")
}

// TestPhaseCompleteEmitsEvent verifies that dedup event is emitted when Phase:Complete blocks.
func TestPhaseCompleteEmitsEvent(t *testing.T) {
	// Similar to TestSessionDedupEmitsEvent, this requires integration with beads RPC.
	// We'll skip for now and document the limitation.
	t.Skip("Phase:Complete dedup requires beads RPC integration - tested manually")
}

// TestEventLoggerNilSafe verifies that nil event logger doesn't cause panics.
func TestEventLoggerNilSafe(t *testing.T) {
	// Setup: Create daemon WITHOUT event logger (nil)
	d := NewWithConfig(Config{
		Label:   "triage:ready",
		Verbose: false,
	})

	// EventLogger should be nil by default
	if d.EventLogger != nil {
		t.Fatal("expected EventLogger to be nil by default")
	}

	testIssueID := "test-issue-789"
	d.SpawnedIssues.MarkSpawned(testIssueID)

	d.listIssuesFunc = func() ([]Issue, error) {
		return []Issue{
			{
				ID:          testIssueID,
				Title:       "Test Issue",
				Description: "Test description",
				Status:      "open",
				IssueType:   "task",
				Priority:    2,
				Labels:      []string{"triage:ready"},
			},
		}, nil
	}

	// Execute: Call NextIssueExcluding - should NOT panic even with nil logger
	issue, err := d.NextIssueExcluding(nil)

	// Verify: No panic, issue was skipped
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue != nil {
		t.Fatalf("expected nil issue (should be skipped), got: %v", issue)
	}
}

// TestSetEventLogger verifies the SetEventLogger method works correctly.
func TestSetEventLogger(t *testing.T) {
	d := NewWithConfig(Config{})

	// Initially nil
	if d.EventLogger != nil {
		t.Fatal("expected EventLogger to be nil initially")
	}

	// Set logger
	mockLogger := &MockEventLogger{Events: []map[string]interface{}{}}
	d.SetEventLogger(mockLogger)

	// Now should be set
	if d.EventLogger == nil {
		t.Fatal("expected EventLogger to be set")
	}

	// Verify it's the same instance
	if d.EventLogger != mockLogger {
		t.Fatal("expected EventLogger to be the mock instance")
	}
}

// TestSpawnedTrackerCleanupStale verifies that stale entries don't trigger events.
func TestSpawnedTrackerCleanupStale(t *testing.T) {
	mockLogger := &MockEventLogger{Events: []map[string]interface{}{}}
	d := NewWithConfig(Config{
		Label:   "triage:ready",
		Verbose: false,
	})
	d.SetEventLogger(mockLogger)

	// Create a tracker with very short TTL for testing
	d.SpawnedIssues = NewSpawnedIssueTrackerWithTTL(DefaultSpawnedIssueTrackerMaxEntries, 100*time.Millisecond)

	testIssueID := "test-stale-issue"
	d.SpawnedIssues.MarkSpawned(testIssueID)

	// Wait for entry to become stale
	time.Sleep(150 * time.Millisecond)

	d.listIssuesFunc = func() ([]Issue, error) {
		return []Issue{
			{
				ID:          testIssueID,
				Title:       "Stale Test Issue",
				Description: "Should not be blocked after TTL",
				Status:      "open",
				IssueType:   "bug",
				Priority:    2,
				Labels:      []string{"triage:ready"},
			},
		}, nil
	}

	// Execute: Call NextIssueExcluding - stale entry should not block
	issue, err := d.NextIssueExcluding(nil)

	// Verify: Issue was NOT skipped (TTL expired)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("expected issue to be returned (stale entry shouldn't block)")
	}
	if issue.ID != testIssueID {
		t.Errorf("expected issue ID %s, got %s", testIssueID, issue.ID)
	}

	// Verify: No dedup event was logged (entry was stale)
	if len(mockLogger.Events) != 0 {
		t.Errorf("expected 0 dedup events for stale entry, got %d", len(mockLogger.Events))
	}
}
