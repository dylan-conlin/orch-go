package daemon

import (
	"fmt"
	"testing"
)

func TestSpawnFailureTracker_PerIssueCircuitBreaker(t *testing.T) {
	tracker := NewSpawnFailureTracker()

	// Issue should not be circuit-broken initially
	broken, count, _ := tracker.IsIssueCircuitBroken("issue-1")
	if broken {
		t.Error("issue should not be circuit-broken with 0 failures")
	}
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}

	// Record failures below threshold (default 3)
	tracker.RecordIssueFailure("issue-1", "spawn failed: exit 1")
	tracker.RecordIssueFailure("issue-1", "spawn failed: exit 1")

	broken, count, _ = tracker.IsIssueCircuitBroken("issue-1")
	if broken {
		t.Error("issue should not be circuit-broken with 2 failures (threshold 3)")
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}

	// Third failure should trigger circuit breaker
	tracker.RecordIssueFailure("issue-1", "spawn failed: workdir not found")

	broken, count, reason := tracker.IsIssueCircuitBroken("issue-1")
	if !broken {
		t.Error("issue should be circuit-broken after 3 failures")
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}
	if reason != "spawn failed: workdir not found" {
		t.Errorf("expected last reason, got %q", reason)
	}
}

func TestSpawnFailureTracker_ClearIssueFailures(t *testing.T) {
	tracker := NewSpawnFailureTracker()

	// Record failures to trigger circuit breaker
	for i := 0; i < 3; i++ {
		tracker.RecordIssueFailure("issue-1", "fail")
	}

	broken, _, _ := tracker.IsIssueCircuitBroken("issue-1")
	if !broken {
		t.Fatal("issue should be circuit-broken")
	}

	// Clear should reset
	tracker.ClearIssueFailures("issue-1")

	broken, count, _ := tracker.IsIssueCircuitBroken("issue-1")
	if broken {
		t.Error("issue should not be circuit-broken after clear")
	}
	if count != 0 {
		t.Errorf("expected count 0 after clear, got %d", count)
	}
}

func TestSpawnFailureTracker_PerIssueIndependent(t *testing.T) {
	tracker := NewSpawnFailureTracker()

	// Circuit-break issue-1
	for i := 0; i < 3; i++ {
		tracker.RecordIssueFailure("issue-1", "fail")
	}

	// issue-2 should be unaffected
	broken, _, _ := tracker.IsIssueCircuitBroken("issue-2")
	if broken {
		t.Error("issue-2 should not be circuit-broken (no failures recorded)")
	}

	// issue-1 should be circuit-broken
	broken, _, _ = tracker.IsIssueCircuitBroken("issue-1")
	if !broken {
		t.Error("issue-1 should be circuit-broken")
	}
}

func TestSpawnFailureTracker_CustomThreshold(t *testing.T) {
	tracker := NewSpawnFailureTrackerWithThreshold(5)

	// Should not be broken at 3 (default) since threshold is 5
	for i := 0; i < 3; i++ {
		tracker.RecordIssueFailure("issue-1", "fail")
	}

	broken, _, _ := tracker.IsIssueCircuitBroken("issue-1")
	if broken {
		t.Error("issue should not be circuit-broken at 3 failures with threshold 5")
	}

	// Should be broken at 5
	tracker.RecordIssueFailure("issue-1", "fail")
	tracker.RecordIssueFailure("issue-1", "fail")

	broken, count, _ := tracker.IsIssueCircuitBroken("issue-1")
	if !broken {
		t.Error("issue should be circuit-broken at 5 failures with threshold 5")
	}
	if count != 5 {
		t.Errorf("expected count 5, got %d", count)
	}
}

func TestSpawnFailureTracker_CircuitBrokenIssues(t *testing.T) {
	tracker := NewSpawnFailureTracker()

	// Circuit-break two issues
	for i := 0; i < 3; i++ {
		tracker.RecordIssueFailure("issue-1", "cross-repo fail")
		tracker.RecordIssueFailure("issue-2", "workdir missing")
	}

	// One non-broken issue
	tracker.RecordIssueFailure("issue-3", "transient")

	broken := tracker.CircuitBrokenIssues()
	if len(broken) != 2 {
		t.Errorf("expected 2 circuit-broken issues, got %d", len(broken))
	}
	if _, ok := broken["issue-1"]; !ok {
		t.Error("issue-1 should be in circuit-broken set")
	}
	if _, ok := broken["issue-2"]; !ok {
		t.Error("issue-2 should be in circuit-broken set")
	}
	if _, ok := broken["issue-3"]; !ok {
		// issue-3 only has 1 failure, should NOT be circuit-broken
	} else {
		t.Error("issue-3 should not be in circuit-broken set")
	}
}

func TestSpawnFailureTracker_SnapshotIncludesCircuitBroken(t *testing.T) {
	tracker := NewSpawnFailureTracker()

	// Circuit-break one issue
	for i := 0; i < 3; i++ {
		tracker.RecordIssueFailure("issue-1", "fail")
	}

	snapshot := tracker.Snapshot()
	if snapshot.CircuitBrokenIssues != 1 {
		t.Errorf("expected 1 circuit-broken issue in snapshot, got %d", snapshot.CircuitBrokenIssues)
	}
}

func TestSpawnFailureTracker_IssueFailureCount(t *testing.T) {
	tracker := NewSpawnFailureTracker()

	if count := tracker.IssueFailureCount("issue-1"); count != 0 {
		t.Errorf("expected 0 for unknown issue, got %d", count)
	}

	tracker.RecordIssueFailure("issue-1", "fail")
	tracker.RecordIssueFailure("issue-1", "fail")

	if count := tracker.IssueFailureCount("issue-1"); count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

// TestDaemon_NextIssueExcluding_SkipsCircuitBrokenIssues verifies that
// the daemon skips issues that have exceeded the per-issue failure threshold.
// This is the key integration test for the cross-repo queue poisoning fix.
func TestDaemon_NextIssueExcluding_SkipsCircuitBrokenIssues(t *testing.T) {
	tracker := NewSpawnFailureTracker()

	// Circuit-break issue-1
	for i := 0; i < 3; i++ {
		tracker.RecordIssueFailure("orch-go-bad1", "cross-repo spawn failed")
	}

	d := &Daemon{
		Config: Config{Label: "triage:ready"},
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "orch-go-bad1", Title: "Cross-repo issue", IssueType: "task", Status: "open", Labels: []string{"triage:ready"}},
					{ID: "orch-go-good", Title: "Local issue", IssueType: "task", Status: "open", Labels: []string{"triage:ready"}},
				}, nil
			},
		},
		SpawnFailureTracker: tracker,
	}

	issue, err := d.NextIssueExcluding(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("expected an issue, got nil")
	}
	if issue.ID != "orch-go-good" {
		t.Errorf("expected orch-go-good (circuit-broken issue should be skipped), got %s", issue.ID)
	}
}

// TestDaemon_NextIssueExcluding_NoIssuesWhenAllCircuitBroken verifies that
// when all issues are circuit-broken, NextIssueExcluding returns nil.
func TestDaemon_NextIssueExcluding_NoIssuesWhenAllCircuitBroken(t *testing.T) {
	tracker := NewSpawnFailureTracker()

	// Circuit-break all issues
	for i := 0; i < 3; i++ {
		tracker.RecordIssueFailure("issue-1", "fail")
		tracker.RecordIssueFailure("issue-2", "fail")
	}

	d := &Daemon{
		Config: Config{Label: "triage:ready"},
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "issue-1", Title: "Issue 1", IssueType: "task", Status: "open", Labels: []string{"triage:ready"}},
					{ID: "issue-2", Title: "Issue 2", IssueType: "task", Status: "open", Labels: []string{"triage:ready"}},
				}, nil
			},
		},
		SpawnFailureTracker: tracker,
	}

	issue, err := d.NextIssueExcluding(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue != nil {
		t.Errorf("expected nil (all circuit-broken), got %s", issue.ID)
	}
}

// TestDaemon_spawnIssue_RecordsPerIssueFailure verifies that spawn failures
// are recorded per-issue, eventually circuit-breaking the issue.
// Uses spawnIssue directly to avoid control plane halt checks in OnceExcluding.
func TestDaemon_spawnIssue_RecordsPerIssueFailure(t *testing.T) {
	tracker := NewSpawnFailureTracker()
	spawnCallCount := 0

	d := &Daemon{
		Config: Config{Label: "triage:ready"},
		Issues: &mockIssueQuerier{
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				return "open", nil
			},
		},
		StatusUpdater:       &mockIssueUpdater{},
		SpawnFailureTracker: tracker,
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir string) error {
				spawnCallCount++
				return fmt.Errorf("spawn failed: workdir not found")
			},
		},
	}

	issue := &Issue{ID: "cross-repo-1", Title: "Cross-repo issue", IssueType: "task", Status: "open"}

	// First spawn attempt - should fail and record per-issue failure
	result, _, err := d.spawnIssue(issue, "feature-impl", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Processed {
		t.Error("expected spawn failure, got processed=true")
	}
	if tracker.IssueFailureCount("cross-repo-1") != 1 {
		t.Errorf("expected 1 per-issue failure, got %d", tracker.IssueFailureCount("cross-repo-1"))
	}

	// Second and third attempts
	d.spawnIssue(issue, "feature-impl", "")
	d.spawnIssue(issue, "feature-impl", "")

	if tracker.IssueFailureCount("cross-repo-1") != 3 {
		t.Errorf("expected 3 per-issue failures, got %d", tracker.IssueFailureCount("cross-repo-1"))
	}

	// Verify issue is now circuit-broken
	broken, count, _ := tracker.IsIssueCircuitBroken("cross-repo-1")
	if !broken {
		t.Error("issue should be circuit-broken after 3 failures")
	}
	if count != 3 {
		t.Errorf("expected 3 failures, got %d", count)
	}
}

// TestDaemon_spawnIssue_ClearsFailuresOnSuccess verifies that a successful
// spawn clears the per-issue failure count.
func TestDaemon_spawnIssue_ClearsFailuresOnSuccess(t *testing.T) {
	tracker := NewSpawnFailureTracker()

	d := &Daemon{
		Config: Config{Label: "triage:ready"},
		Issues: &mockIssueQuerier{
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				return "open", nil
			},
		},
		StatusUpdater:       &mockIssueUpdater{},
		SpawnFailureTracker: tracker,
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir string) error {
				return nil // success
			},
		},
	}

	// Record some failures first
	tracker.RecordIssueFailure("issue-1", "fail 1")
	tracker.RecordIssueFailure("issue-1", "fail 2")

	if tracker.IssueFailureCount("issue-1") != 2 {
		t.Fatalf("expected 2 failures before spawn, got %d", tracker.IssueFailureCount("issue-1"))
	}

	issue := &Issue{ID: "issue-1", Title: "Test issue", IssueType: "task", Status: "open"}
	result, _, err := d.spawnIssue(issue, "feature-impl", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Processed {
		t.Error("expected successful spawn")
	}

	// Failures should be cleared after successful spawn
	if tracker.IssueFailureCount("issue-1") != 0 {
		t.Errorf("expected 0 failures after success, got %d", tracker.IssueFailureCount("issue-1"))
	}
}
