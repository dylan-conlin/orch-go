// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestVerificationTracker_RecordCompletion(t *testing.T) {
	tests := []struct {
		name           string
		threshold      int
		completions    int
		expectPause    bool
		expectIsPaused bool
	}{
		{
			name:           "below threshold",
			threshold:      3,
			completions:    2,
			expectPause:    false,
			expectIsPaused: false,
		},
		{
			name:           "at threshold",
			threshold:      3,
			completions:    3,
			expectPause:    true,
			expectIsPaused: true,
		},
		{
			name:           "above threshold",
			threshold:      3,
			completions:    5,
			expectPause:    true, // First pause at 3, subsequent calls don't unpause
			expectIsPaused: true,
		},
		{
			name:           "threshold zero (disabled) - counter should not increment",
			threshold:      0,
			completions:    10,
			expectPause:    false,
			expectIsPaused: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := NewVerificationTracker(tt.threshold)

			var shouldPause bool
			for i := 0; i < tt.completions; i++ {
				shouldPause = vt.RecordCompletion(fmt.Sprintf("agent-%d", i))
			}

			// Check final pause signal
			if shouldPause != tt.expectPause && tt.threshold > 0 {
				// Only check last call for pause signal
				vt2 := NewVerificationTracker(tt.threshold)
				for i := 0; i < tt.completions-1; i++ {
					vt2.RecordCompletion(fmt.Sprintf("agent-%d", i))
				}
				lastPause := vt2.RecordCompletion(fmt.Sprintf("agent-%d", tt.completions-1))
				if lastPause != tt.expectPause {
					t.Errorf("RecordCompletion() pause signal = %v, want %v", lastPause, tt.expectPause)
				}
			}

			// Check paused state
			if vt.IsPaused() != tt.expectIsPaused {
				t.Errorf("IsPaused() = %v, want %v", vt.IsPaused(), tt.expectIsPaused)
			}

			// Check status
			status := vt.Status()
			if status.IsPaused != tt.expectIsPaused {
				t.Errorf("Status().IsPaused = %v, want %v", status.IsPaused, tt.expectIsPaused)
			}

			// When threshold is 0 (disabled), counter should not increment
			expectedCount := tt.completions
			if tt.threshold == 0 {
				expectedCount = 0
			}
			if status.CompletionsSinceVerification != expectedCount {
				t.Errorf("Status().CompletionsSinceVerification = %v, want %v",
					status.CompletionsSinceVerification, expectedCount)
			}
		})
	}
}

func TestVerificationTracker_RecordHumanVerification(t *testing.T) {
	vt := NewVerificationTracker(3)

	// Record some completions to trigger pause
	for i := 0; i < 3; i++ {
		vt.RecordCompletion(fmt.Sprintf("agent-%d", i))
	}

	if !vt.IsPaused() {
		t.Fatal("Expected tracker to be paused after 3 completions")
	}

	beforeVerification := time.Now()
	vt.RecordHumanVerification()
	afterVerification := time.Now()

	// Check that counter was reset
	status := vt.Status()
	if status.CompletionsSinceVerification != 0 {
		t.Errorf("After RecordHumanVerification, counter = %v, want 0",
			status.CompletionsSinceVerification)
	}

	// Check that pause was cleared
	if vt.IsPaused() {
		t.Error("Expected tracker to be unpaused after RecordHumanVerification")
	}

	// Check that last verification timestamp was updated
	if status.LastVerification.Before(beforeVerification) ||
		status.LastVerification.After(afterVerification) {
		t.Errorf("LastVerification timestamp not updated correctly: %v", status.LastVerification)
	}

	// After RecordHumanVerification, same IDs should count again (seenIDs cleared)
	shouldPause := vt.RecordCompletion("agent-0")
	if shouldPause {
		t.Error("Should not pause on first completion after RecordHumanVerification")
	}
	if vt.Status().CompletionsSinceVerification != 1 {
		t.Errorf("Expected 1 completion after re-recording previously seen ID, got %d",
			vt.Status().CompletionsSinceVerification)
	}
}

func TestVerificationTracker_Resume(t *testing.T) {
	vt := NewVerificationTracker(3)

	// Record completions to trigger pause
	for i := 0; i < 3; i++ {
		vt.RecordCompletion(fmt.Sprintf("agent-%d", i))
	}

	if !vt.IsPaused() {
		t.Fatal("Expected tracker to be paused")
	}

	beforeResume := time.Now()
	vt.Resume()
	afterResume := time.Now()

	// Check that pause was cleared
	if vt.IsPaused() {
		t.Error("Expected tracker to be unpaused after Resume")
	}

	// Check that counter was reset
	status := vt.Status()
	if status.CompletionsSinceVerification != 0 {
		t.Errorf("After Resume, counter = %v, want 0", status.CompletionsSinceVerification)
	}

	// Check that last verification was updated
	if status.LastVerification.Before(beforeResume) ||
		status.LastVerification.After(afterResume) {
		t.Errorf("LastVerification not updated on Resume: %v", status.LastVerification)
	}
}

func TestVerificationStatus_RemainingBeforePause(t *testing.T) {
	tests := []struct {
		name          string
		threshold     int
		completions   int
		wantRemaining int
	}{
		{
			name:          "threshold 3, no completions",
			threshold:     3,
			completions:   0,
			wantRemaining: 3,
		},
		{
			name:          "threshold 3, 1 completion",
			threshold:     3,
			completions:   1,
			wantRemaining: 2,
		},
		{
			name:          "threshold 3, 2 completions",
			threshold:     3,
			completions:   2,
			wantRemaining: 1,
		},
		{
			name:          "threshold 3, at threshold",
			threshold:     3,
			completions:   3,
			wantRemaining: 0,
		},
		{
			name:          "threshold 0 (disabled)",
			threshold:     0,
			completions:   5,
			wantRemaining: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := NewVerificationTracker(tt.threshold)
			for i := 0; i < tt.completions; i++ {
				vt.RecordCompletion(fmt.Sprintf("agent-%d", i))
			}

			status := vt.Status()
			remaining := status.RemainingBeforePause()

			if remaining != tt.wantRemaining {
				t.Errorf("RemainingBeforePause() = %v, want %v", remaining, tt.wantRemaining)
			}
		})
	}
}

func TestResumeSignal_WriteAndCheck(t *testing.T) {
	// Set up temp directory for test
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	// Initially, no signal should exist
	exists, err := CheckAndClearResumeSignal()
	if err != nil {
		t.Fatalf("CheckAndClearResumeSignal failed: %v", err)
	}
	if exists {
		t.Error("Expected no resume signal initially")
	}

	// Write resume signal
	if err := WriteResumeSignal(); err != nil {
		t.Fatalf("WriteResumeSignal failed: %v", err)
	}

	// Check that signal exists and is cleared
	exists, err = CheckAndClearResumeSignal()
	if err != nil {
		t.Fatalf("CheckAndClearResumeSignal failed: %v", err)
	}
	if !exists {
		t.Error("Expected resume signal to exist")
	}

	// Check that signal was cleared
	exists, err = CheckAndClearResumeSignal()
	if err != nil {
		t.Fatalf("CheckAndClearResumeSignal failed after clear: %v", err)
	}
	if exists {
		t.Error("Expected resume signal to be cleared after first check")
	}
}

func TestResumeSignal_MultipleWrites(t *testing.T) {
	// Set up temp directory for test
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	// Write signal multiple times
	for i := 0; i < 3; i++ {
		if err := WriteResumeSignal(); err != nil {
			t.Fatalf("WriteResumeSignal failed on iteration %d: %v", i, err)
		}
	}

	// Signal should still exist and be checkable
	exists, err := CheckAndClearResumeSignal()
	if err != nil {
		t.Fatalf("CheckAndClearResumeSignal failed: %v", err)
	}
	if !exists {
		t.Error("Expected resume signal to exist after multiple writes")
	}
}

// Pressure test: Concurrent RecordCompletion calls with unique IDs
func TestVerificationTracker_ConcurrentRecordCompletion(t *testing.T) {
	threshold := 3
	vt := NewVerificationTracker(threshold)

	// Launch many goroutines calling RecordCompletion with unique IDs concurrently
	numGoroutines := 100
	results := make(chan bool, numGoroutines)
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			shouldPause := vt.RecordCompletion(fmt.Sprintf("agent-%d", idx))
			results <- shouldPause
		}(i)
	}

	wg.Wait()
	close(results)

	// Count how many goroutines received the pause signal
	pauseCount := 0
	for result := range results {
		if result {
			pauseCount++
		}
	}

	// Verify tracker state
	status := vt.Status()

	// All 100 unique completions should be recorded
	if status.CompletionsSinceVerification != numGoroutines {
		t.Errorf("Expected %d completions, got %d", numGoroutines, status.CompletionsSinceVerification)
	}

	// Should be paused after hitting threshold
	if !vt.IsPaused() {
		t.Error("Expected tracker to be paused after concurrent completions")
	}

	// At least one goroutine should have received pause signal (when threshold was hit)
	if pauseCount == 0 {
		t.Error("Expected at least one goroutine to receive pause signal")
	}
}

// Pressure test: Concurrent mixed operations
func TestVerificationTracker_ConcurrentMixedOperations(t *testing.T) {
	threshold := 10
	vt := NewVerificationTracker(threshold)

	numGoroutines := 50
	var wg sync.WaitGroup

	// Launch goroutines performing different operations concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			switch idx % 4 {
			case 0:
				vt.RecordCompletion(fmt.Sprintf("agent-%d", idx))
			case 1:
				vt.IsPaused()
			case 2:
				vt.Status()
			case 3:
				// Every 10th operation does resume/verification
				if idx%10 == 3 {
					vt.Resume()
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify tracker is in a valid state
	status := vt.Status()
	if status.CompletionsSinceVerification < 0 {
		t.Errorf("Invalid completion count: %d", status.CompletionsSinceVerification)
	}
}

// Pressure test: Rapid pause/resume cycles
func TestVerificationTracker_RapidPauseResumeCycles(t *testing.T) {
	threshold := 3
	vt := NewVerificationTracker(threshold)

	// Perform rapid pause/resume cycles
	for cycle := 0; cycle < 100; cycle++ {
		// Record completions to trigger pause (unique IDs per cycle)
		for i := 0; i < threshold; i++ {
			vt.RecordCompletion(fmt.Sprintf("cycle%d-agent%d", cycle, i))
		}

		if !vt.IsPaused() {
			t.Fatalf("Expected pause after %d completions in cycle %d", threshold, cycle)
		}

		// Resume immediately (clears seenIDs)
		vt.Resume()

		if vt.IsPaused() {
			t.Fatalf("Expected unpause after Resume in cycle %d", cycle)
		}

		// Verify counter was reset
		status := vt.Status()
		if status.CompletionsSinceVerification != 0 {
			t.Errorf("Expected counter reset in cycle %d, got %d", cycle, status.CompletionsSinceVerification)
		}
	}
}

// Pressure test: State consistency under concurrent pause/resume
func TestVerificationTracker_ConcurrentPauseResume(t *testing.T) {
	threshold := 5
	vt := NewVerificationTracker(threshold)

	var wg sync.WaitGroup
	numIterations := 1000

	// Goroutine 1: Continuously record completions with unique IDs
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numIterations; i++ {
			vt.RecordCompletion(fmt.Sprintf("agent-%d", i))
			time.Sleep(1 * time.Microsecond)
		}
	}()

	// Goroutine 2: Continuously resume (clears seenIDs, allowing re-counting)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numIterations/2; i++ {
			vt.Resume()
			time.Sleep(2 * time.Microsecond)
		}
	}()

	// Goroutine 3: Continuously check status
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numIterations; i++ {
			status := vt.Status()
			// Verify invariants
			if status.CompletionsSinceVerification < 0 {
				t.Errorf("Invalid completion count: %d", status.CompletionsSinceVerification)
			}
		}
	}()

	wg.Wait()

	// Final state should be consistent
	status := vt.Status()
	if status.CompletionsSinceVerification < 0 {
		t.Errorf("Final state invalid: completions = %d", status.CompletionsSinceVerification)
	}
}

// Pressure test: Concurrent RecordHumanVerification and RecordCompletion
func TestVerificationTracker_ConcurrentVerificationAndCompletion(t *testing.T) {
	threshold := 5
	vt := NewVerificationTracker(threshold)

	var wg sync.WaitGroup
	numOps := 1000

	// Goroutines recording completions with unique IDs
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(goroutineIdx int) {
			defer wg.Done()
			for j := 0; j < numOps/10; j++ {
				vt.RecordCompletion(fmt.Sprintf("g%d-agent-%d", goroutineIdx, j))
			}
		}(i)
	}

	// Goroutines recording human verifications
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOps/20; j++ {
				vt.RecordHumanVerification()
				time.Sleep(1 * time.Microsecond)
			}
		}()
	}

	wg.Wait()

	// Verify final state is consistent
	status := vt.Status()
	if status.CompletionsSinceVerification < 0 {
		t.Errorf("Invalid final completion count: %d", status.CompletionsSinceVerification)
	}
}

func TestVerificationTracker_SeedFromBacklog(t *testing.T) {
	tests := []struct {
		name          string
		threshold     int
		seedIDs       []string
		expectPaused  bool
		expectCounter int
	}{
		{
			name:          "seed below threshold",
			threshold:     3,
			seedIDs:       []string{"agent-1", "agent-2"},
			expectPaused:  false,
			expectCounter: 2,
		},
		{
			name:          "seed at threshold triggers pause",
			threshold:     3,
			seedIDs:       []string{"agent-1", "agent-2", "agent-3"},
			expectPaused:  true,
			expectCounter: 3,
		},
		{
			name:          "seed above threshold triggers pause",
			threshold:     3,
			seedIDs:       []string{"agent-1", "agent-2", "agent-3", "agent-4", "agent-5"},
			expectPaused:  true,
			expectCounter: 5,
		},
		{
			name:          "seed with zero threshold (disabled)",
			threshold:     0,
			seedIDs:       []string{"agent-1", "agent-2", "agent-3", "agent-4", "agent-5"},
			expectPaused:  false,
			expectCounter: 5,
		},
		{
			name:          "seed empty",
			threshold:     3,
			seedIDs:       []string{},
			expectPaused:  false,
			expectCounter: 0,
		},
		{
			name:          "seed with duplicate IDs deduplicates",
			threshold:     3,
			seedIDs:       []string{"agent-1", "agent-1", "agent-2"},
			expectPaused:  false,
			expectCounter: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := NewVerificationTracker(tt.threshold)
			vt.SeedFromBacklog(tt.seedIDs)

			if vt.IsPaused() != tt.expectPaused {
				t.Errorf("IsPaused() = %v, want %v", vt.IsPaused(), tt.expectPaused)
			}

			status := vt.Status()
			if status.CompletionsSinceVerification != tt.expectCounter {
				t.Errorf("CompletionsSinceVerification = %v, want %v",
					status.CompletionsSinceVerification, tt.expectCounter)
			}
		})
	}
}

func TestVerificationTracker_SeedThenRecord(t *testing.T) {
	// Seed with 2 IDs of 3 threshold, then record 1 new to hit threshold
	vt := NewVerificationTracker(3)
	vt.SeedFromBacklog([]string{"agent-1", "agent-2"})

	if vt.IsPaused() {
		t.Fatal("Should not be paused after seeding 2 with threshold 3")
	}

	shouldPause := vt.RecordCompletion("agent-3")
	if !shouldPause {
		t.Error("RecordCompletion should signal pause at threshold")
	}
	if !vt.IsPaused() {
		t.Error("Should be paused after reaching threshold")
	}

	status := vt.Status()
	if status.CompletionsSinceVerification != 3 {
		t.Errorf("Expected 3 completions (2 seeded + 1 recorded), got %d",
			status.CompletionsSinceVerification)
	}
}

// Test that seeded IDs are deduplicated when subsequently recorded
func TestVerificationTracker_SeedThenRecordDuplicate(t *testing.T) {
	vt := NewVerificationTracker(3)
	vt.SeedFromBacklog([]string{"agent-1", "agent-2"})

	// Recording a seeded ID should NOT increment counter
	shouldPause := vt.RecordCompletion("agent-1")
	if shouldPause {
		t.Error("Recording seeded ID should not cause pause")
	}

	status := vt.Status()
	if status.CompletionsSinceVerification != 2 {
		t.Errorf("Expected 2 completions (seeded IDs only, duplicate ignored), got %d",
			status.CompletionsSinceVerification)
	}
}

// Pressure test: Threshold boundary concurrent access
func TestVerificationTracker_ThresholdBoundaryConcurrent(t *testing.T) {
	threshold := 3
	vt := NewVerificationTracker(threshold)

	// Pre-load to one below threshold with unique IDs
	for i := 0; i < threshold-1; i++ {
		vt.RecordCompletion(fmt.Sprintf("preload-%d", i))
	}

	// Launch many goroutines trying to cross the threshold simultaneously
	numGoroutines := 50
	results := make(chan bool, numGoroutines)
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			shouldPause := vt.RecordCompletion(fmt.Sprintf("concurrent-%d", idx))
			results <- shouldPause
		}(i)
	}

	wg.Wait()
	close(results)

	// Count pause signals
	pauseCount := 0
	for result := range results {
		if result {
			pauseCount++
		}
	}

	// Should be paused
	if !vt.IsPaused() {
		t.Error("Expected tracker to be paused")
	}

	// At least one should have triggered pause
	if pauseCount == 0 {
		t.Error("Expected at least one pause signal")
	}

	// Total completions should be threshold-1 (preload) + numGoroutines
	status := vt.Status()
	expectedTotal := (threshold - 1) + numGoroutines
	if status.CompletionsSinceVerification != expectedTotal {
		t.Errorf("Expected %d total completions, got %d", expectedTotal, status.CompletionsSinceVerification)
	}
}

// Regression test: RecordCompletion must only count unique beads IDs.
// Previously, RecordCompletion used a plain counter without deduplication.
// The same agent appearing across multiple poll cycles would increment the
// counter each time, causing premature pause (1 agent × 3 cycles = threshold 3).
func TestVerificationTracker_DeduplicatesByBeadsID(t *testing.T) {
	threshold := 3
	vt := NewVerificationTracker(threshold)

	// Simulate the bug scenario: 1 agent across 3 poll cycles
	// Each poll cycle finds the same agent and calls RecordCompletion
	for cycle := 0; cycle < 3; cycle++ {
		shouldPause := vt.RecordCompletion("orch-go-smha") // Same ID every cycle
		if shouldPause {
			t.Fatalf("Same agent across %d cycles should NOT trigger pause (threshold=%d)",
				cycle+1, threshold)
		}
	}

	// Counter should be 1 (one unique agent), not 3
	status := vt.Status()
	if status.CompletionsSinceVerification != 1 {
		t.Errorf("Expected 1 unique completion, got %d (dedup failed)",
			status.CompletionsSinceVerification)
	}

	// Now add 2 more UNIQUE agents to actually hit the threshold
	vt.RecordCompletion("orch-go-abc1")
	shouldPause := vt.RecordCompletion("orch-go-xyz2")
	if !shouldPause {
		t.Error("Should pause after 3 unique agents")
	}

	status = vt.Status()
	if status.CompletionsSinceVerification != 3 {
		t.Errorf("Expected 3 unique completions, got %d",
			status.CompletionsSinceVerification)
	}
}

// Regression test: RecordCompletion with duplicate ID returns current pause state.
// When an already-seen ID is recorded while paused, it should still return true.
func TestVerificationTracker_DuplicateReturnsPausedState(t *testing.T) {
	threshold := 2
	vt := NewVerificationTracker(threshold)

	vt.RecordCompletion("agent-1")
	shouldPause := vt.RecordCompletion("agent-2") // Hits threshold
	if !shouldPause {
		t.Fatal("Should pause at threshold")
	}

	// Recording a duplicate while paused should return true (still paused)
	stillPaused := vt.RecordCompletion("agent-1")
	if !stillPaused {
		t.Error("Duplicate record while paused should return true (paused state)")
	}

	// Counter should still be 2 (not incremented)
	status := vt.Status()
	if status.CompletionsSinceVerification != 2 {
		t.Errorf("Expected 2 (duplicate not counted), got %d",
			status.CompletionsSinceVerification)
	}
}

// TestVerificationTracker_ResyncWithBacklog verifies that the resync mechanism
// correctly removes stale entries and unpauses when the backlog drops below threshold.
// This is the fix for the daemon verification pause disagreeing with orch review:
// the tracker was seeded at startup but never re-checked when issues were closed
// through non-interactive paths (headless completion, bd close).
func TestVerificationTracker_ResyncWithBacklog(t *testing.T) {
	t.Run("resync_unpauses_when_backlog_drops", func(t *testing.T) {
		vt := NewVerificationTracker(3)

		// Seed with 3 items → paused
		vt.SeedFromBacklog([]string{"a", "b", "c"})
		if !vt.IsPaused() {
			t.Fatal("Expected paused after seeding 3 items with threshold 3")
		}

		// Resync: only 1 item remains in backlog
		unpaused := vt.ResyncWithBacklog([]string{"a"})
		if !unpaused {
			t.Error("Expected ResyncWithBacklog to return true (was paused, now unpaused)")
		}
		if vt.IsPaused() {
			t.Error("Expected unpaused after resync dropped count below threshold")
		}
		status := vt.Status()
		if status.CompletionsSinceVerification != 1 {
			t.Errorf("Expected 1 completion after resync, got %d", status.CompletionsSinceVerification)
		}
	})

	t.Run("resync_stays_paused_when_backlog_at_threshold", func(t *testing.T) {
		vt := NewVerificationTracker(3)

		// Seed with 4 items → paused
		vt.SeedFromBacklog([]string{"a", "b", "c", "d"})
		if !vt.IsPaused() {
			t.Fatal("Expected paused")
		}

		// Resync: 3 items remain (still at threshold)
		unpaused := vt.ResyncWithBacklog([]string{"a", "b", "c"})
		if unpaused {
			t.Error("Expected ResyncWithBacklog to return false (still at threshold)")
		}
		if !vt.IsPaused() {
			t.Error("Expected still paused at threshold")
		}
	})

	t.Run("resync_empty_backlog_unpauses", func(t *testing.T) {
		vt := NewVerificationTracker(3)

		vt.SeedFromBacklog([]string{"a", "b", "c"})
		if !vt.IsPaused() {
			t.Fatal("Expected paused")
		}

		// All issues closed
		unpaused := vt.ResyncWithBacklog([]string{})
		if !unpaused {
			t.Error("Expected unpaused with empty backlog")
		}
		if vt.IsPaused() {
			t.Error("Expected unpaused")
		}
		status := vt.Status()
		if status.CompletionsSinceVerification != 0 {
			t.Errorf("Expected 0 completions after empty resync, got %d", status.CompletionsSinceVerification)
		}
	})

	t.Run("resync_noop_when_not_paused", func(t *testing.T) {
		vt := NewVerificationTracker(3)

		vt.RecordCompletion("a")
		if vt.IsPaused() {
			t.Fatal("Should not be paused yet")
		}

		// Resync with same items — should return false (wasn't paused)
		unpaused := vt.ResyncWithBacklog([]string{"a"})
		if unpaused {
			t.Error("Expected false — was not paused before resync")
		}
	})

	t.Run("resync_adds_new_backlog_items_to_seen", func(t *testing.T) {
		vt := NewVerificationTracker(5)

		// Start with 2 items
		vt.SeedFromBacklog([]string{"a", "b"})

		// Resync with 3 items (1 new)
		vt.ResyncWithBacklog([]string{"a", "b", "c"})

		// The new item "c" should NOT be added to seenIDs — resync only
		// prunes stale entries, it doesn't add new ones. New completions
		// come through RecordCompletion().
		status := vt.Status()
		if status.CompletionsSinceVerification != 2 {
			t.Errorf("Expected 2 (resync should only prune, not add), got %d", status.CompletionsSinceVerification)
		}
	})

	t.Run("resync_disabled_tracker", func(t *testing.T) {
		vt := NewVerificationTracker(0) // disabled

		unpaused := vt.ResyncWithBacklog([]string{"a", "b"})
		if unpaused {
			t.Error("Disabled tracker should return false")
		}
	})
}

// Structural test: WriteVerificationSignal must not be called from serve_daemon_actions.go.
// Dashboard API endpoints are used by both humans and automated orchestrator sessions.
// Calling WriteVerificationSignal from those paths resets the daemon's
// completionsSinceVerification counter, defeating the comprehension throttle.
// See: orch-go-hry8a
func TestWriteVerificationSignal_NotCalledFromDashboardAPI(t *testing.T) {
	// Read serve_daemon_actions.go and verify it does not call WriteVerificationSignal
	target := filepath.Join("..", "..", "cmd", "orch", "serve_daemon_actions.go")
	data, err := os.ReadFile(target)
	if err != nil {
		t.Skipf("Cannot read %s: %v (running outside repo root?)", target, err)
	}

	content := string(data)
	if strings.Contains(content, "WriteVerificationSignal()") {
		t.Errorf("serve_daemon_actions.go must NOT call WriteVerificationSignal().\n" +
			"Dashboard API endpoints are used by automated orchestrator sessions,\n" +
			"not exclusively by humans. Calling it here defeats the comprehension throttle.\n" +
			"Human verification signal should only come from interactive `orch complete`.")
	}
}

// Structural test: WriteVerificationSignal in complete_lifecycle.go must be gated
// on both !completeHeadless and !target.IsOrchestratorSession.
// See: orch-go-hry8a
func TestWriteVerificationSignal_GatedInCompleteLifecycle(t *testing.T) {
	target := filepath.Join("..", "..", "cmd", "orch", "complete_lifecycle.go")
	data, err := os.ReadFile(target)
	if err != nil {
		t.Skipf("Cannot read %s: %v (running outside repo root?)", target, err)
	}

	content := string(data)

	// Must have headless guard
	if !strings.Contains(content, "!completeHeadless") {
		t.Error("complete_lifecycle.go: WriteVerificationSignal must be gated on !completeHeadless")
	}

	// Must have orchestrator session guard
	if !strings.Contains(content, "!target.IsOrchestratorSession") {
		t.Error("complete_lifecycle.go: WriteVerificationSignal must be gated on !target.IsOrchestratorSession")
	}
}
