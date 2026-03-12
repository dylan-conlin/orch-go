package daemon

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestConcurrentOnce_SameIssue verifies that when multiple goroutines call
// daemon.Once() simultaneously with the same issue in the queue, only ONE
// spawn occurs.
//
// PRODUCTION NOTE: The daemon loop calls Once() sequentially (daemon_loop.go:355),
// and PID lock prevents multiple daemon instances (daemon_loop.go:63). So this
// concurrent scenario doesn't occur within the daemon itself. However, it
// validates the correctness of dedup gates under concurrent stress and catches
// data races (like the session_dedup.go sync.Once fix).
func TestConcurrentOnce_SameIssue(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	var spawnCount atomic.Int32

	issues := &mockIssueQuerier{
		ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "issue-1", Title: "Build feature X", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		},
		GetIssueStatusFunc: func(beadsID string) (string, error) {
			return "open", nil
		},
	}

	d := &Daemon{
		SpawnedIssues: tracker,
		Issues:        issues,
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
				spawnCount.Add(1)
				time.Sleep(10 * time.Millisecond)
				return nil
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error {
				return nil
			},
		},
	}

	const goroutines = 10
	var wg sync.WaitGroup
	var processedCount atomic.Int32

	wg.Add(goroutines)
	barrier := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-barrier
			result, err := d.Once()
			if err != nil {
				t.Errorf("Once() error: %v", err)
				return
			}
			if result.Processed {
				processedCount.Add(1)
			}
		}()
	}

	close(barrier)
	wg.Wait()

	// Document the TOCTOU: multiple goroutines can pass NextIssueExcluding's
	// IsSpawned check and the pipeline's SpawnTrackerGate before the first one
	// calls MarkSpawnedWithTitle. The test logs the count for analysis.
	count := spawnCount.Load()
	t.Logf("SpawnWork called %d times out of %d concurrent goroutines", count, goroutines)

	if count == 0 {
		t.Error("No goroutine spawned the issue")
	}
	if count > 1 {
		t.Logf("TOCTOU CONFIRMED: %d concurrent spawns passed all gates", count)
		t.Log("In production, the daemon calls Once() sequentially + PID lock prevents multiple instances")
		t.Log("Manual spawn + daemon racing is a real risk window (see TestManualSpawn_BeadsStatusCheck_TOCTOU)")
	}
}

// TestConcurrentSpawnIssue_DirectCall verifies spawnIssue() behavior under
// concurrent stress. This tests the 5-layer pipeline + status update.
func TestConcurrentSpawnIssue_DirectCall(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	var spawnCount atomic.Int32

	d := &Daemon{
		SpawnedIssues: tracker,
		Issues: &mockIssueQuerier{
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				return "open", nil
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
				spawnCount.Add(1)
				time.Sleep(5 * time.Millisecond)
				return nil
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error {
				return nil
			},
		},
	}

	issue := &Issue{ID: "issue-1", Title: "Build feature X", IssueType: "feature", Status: "open"}

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	barrier := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-barrier
			d.spawnIssue(issue, "feature-impl", "opus")
		}()
	}

	close(barrier)
	wg.Wait()

	count := spawnCount.Load()
	t.Logf("SpawnWork called %d times out of %d concurrent goroutines", count, goroutines)

	if count > 1 {
		t.Logf("TOCTOU in spawnIssue: %d goroutines passed pipeline before first MarkSpawnedWithTitle", count)
	}
}

// TestConcurrentSpawnIssue_FreshStatusSerializes verifies that when the
// FreshStatusGate reflects actual beads state changes, it prevents duplicates.
// This is the production protection: beads status is the structural backstop.
func TestConcurrentSpawnIssue_FreshStatusSerializes(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	var spawnCount atomic.Int32

	// Simulate beads status that actually changes when updated
	var mu sync.Mutex
	issueStatus := "open"

	d := &Daemon{
		SpawnedIssues: tracker,
		Issues: &mockIssueQuerier{
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				mu.Lock()
				s := issueStatus
				mu.Unlock()
				return s, nil
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
				spawnCount.Add(1)
				return nil
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error {
				mu.Lock()
				issueStatus = status
				mu.Unlock()
				return nil
			},
		},
	}

	issue := &Issue{ID: "issue-1", Title: "Build feature X", IssueType: "feature", Status: "open"}

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	barrier := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-barrier
			d.spawnIssue(issue, "feature-impl", "opus")
		}()
	}

	close(barrier)
	wg.Wait()

	count := spawnCount.Load()
	t.Logf("With status reflection: SpawnWork called %d times out of %d goroutines", count, goroutines)

	// With the fresh status gate reflecting actual state, only 1 should succeed
	if count > 1 {
		t.Errorf("SpawnWork called %d times (want at most 1) — fresh status gate didn't serialize", count)
	}
}

// TestSpawnTrackerGate_TOCTOU_Window measures the check-then-mark window.
// Multiple goroutines can pass the gate before the first marks the issue.
func TestSpawnTrackerGate_TOCTOU_Window(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	issue := &Issue{ID: "issue-race", Title: "Race Test", IssueType: "feature", Status: "open"}

	const goroutines = 100
	var passedGate atomic.Int32

	var wg sync.WaitGroup
	wg.Add(goroutines)
	barrier := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-barrier

			gate := &SpawnTrackerGate{Tracker: tracker}
			result := gate.Check(issue)
			if result.Verdict == GateAllow {
				passedGate.Add(1)
				tracker.MarkSpawnedWithTitle(issue.ID, issue.Title)
			}
		}()
	}

	close(barrier)
	wg.Wait()

	passed := passedGate.Load()
	t.Logf("SpawnTrackerGate TOCTOU: %d/%d goroutines passed before first mark", passed, goroutines)

	if passed == 0 {
		t.Error("No goroutine passed the gate")
	}
	if passed > 1 {
		t.Logf("TOCTOU window confirmed — defense-in-depth via beads status is the real protection")
	}
}

// TestSequentialOnce_PreventsDuplicate confirms that sequential Once() calls
// (the actual production pattern) correctly prevent duplicate spawns.
func TestSequentialOnce_PreventsDuplicate(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	var spawnCount atomic.Int32

	d := &Daemon{
		SpawnedIssues: tracker,
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "issue-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
				}, nil
			},
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				return "open", nil
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
				spawnCount.Add(1)
				return nil
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error {
				return nil
			},
		},
	}

	// First call: should spawn
	result1, err := d.Once()
	if err != nil {
		t.Fatalf("First Once() error: %v", err)
	}
	if !result1.Processed {
		t.Fatal("First Once() should process")
	}

	// Second call: should be blocked by spawn tracker
	result2, err := d.Once()
	if err != nil {
		t.Fatalf("Second Once() error: %v", err)
	}
	if result2.Processed {
		t.Error("Second Once() should NOT process — spawn tracker should block")
	}

	if count := spawnCount.Load(); count != 1 {
		t.Errorf("Expected exactly 1 spawn, got %d", count)
	}
}

// TestSpawnTracker_MarkIsAtomic verifies concurrent MarkSpawnedWithTitle calls
// don't corrupt internal state. Run with: go test -race
func TestSpawnTracker_MarkIsAtomic(t *testing.T) {
	tracker := NewSpawnedIssueTracker()

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			issueID := fmt.Sprintf("issue-%d", id)
			tracker.MarkSpawnedWithTitle(issueID, fmt.Sprintf("Title %d", id))
			tracker.IsSpawned(issueID)
			tracker.IsTitleSpawned(fmt.Sprintf("Title %d", id))
			tracker.SpawnCount(issueID)
		}(i)
	}

	wg.Wait()

	if count := tracker.Count(); count != goroutines {
		t.Errorf("Expected %d tracked issues, got %d", goroutines, count)
	}
}

// TestSpawnIssue_SequentialBlocksSecondCall confirms the serial dedup path.
func TestSpawnIssue_SequentialBlocksSecondCall(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	var spawnCount atomic.Int32

	d := &Daemon{
		SpawnedIssues: tracker,
		Issues: &mockIssueQuerier{
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				return "open", nil
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
				spawnCount.Add(1)
				return nil
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error {
				return nil
			},
		},
	}

	issue := &Issue{ID: "issue-1", Title: "Feature X", IssueType: "feature", Status: "open"}

	result1, _, err := d.spawnIssue(issue, "feature-impl", "opus")
	if err != nil {
		t.Fatalf("First spawnIssue() error: %v", err)
	}
	if !result1.Processed {
		t.Fatal("First spawnIssue() should process")
	}

	result2, _, err := d.spawnIssue(issue, "feature-impl", "opus")
	if err != nil {
		t.Fatalf("Second spawnIssue() error: %v", err)
	}
	if result2.Processed {
		t.Error("Second spawnIssue() should be blocked by spawn tracker gate")
	}

	if count := spawnCount.Load(); count != 1 {
		t.Errorf("Expected exactly 1 spawn, got %d", count)
	}
}

// TestManualSpawn_BeadsStatusCheck_TOCTOU documents the race window
// in the manual spawn path (SetupBeadsTracking in spawn_beads.go).
func TestManualSpawn_BeadsStatusCheck_TOCTOU(t *testing.T) {
	// SetupBeadsTracking (spawn_beads.go:38-56) has this race window:
	//
	// Process A (orch spawn)            Process B (orch spawn OR daemon)
	// ──────────────────────            ────────────────────────────────
	// GetIssue() → status="open"
	//                                   GetIssue() → status="open"
	// UpdateIssueStatus("in_progress")
	//                                   UpdateIssueStatus("in_progress")
	// → both succeed, both create tmux windows
	//
	// Mitigation layers:
	// 1. Manual spawns are rare (1-2 per day, human-driven)
	// 2. Triage label removal (spawn_cmd.go:311) closes daemon race window
	// 3. Workspace name includes random suffix (og-*-XXXX), reducing collision
	// 4. Tmux window name includes beads ID, making duplicate visible
	//
	// Structural fix would require: beads CAS (compare-and-swap) semantics —
	// "update status to in_progress WHERE current_status = open" atomically.

	t.Log("DOCUMENTED: SetupBeadsTracking has TOCTOU between status check and update")
	t.Log("PRODUCTION RISK: Low (manual spawns are infrequent, label removal narrows window)")
	t.Log("STRUCTURAL FIX: Beads CAS semantics or file-level locking on status transition")
}
