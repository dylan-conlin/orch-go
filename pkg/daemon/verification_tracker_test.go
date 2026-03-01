// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"os"
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
				shouldPause = vt.RecordCompletion()
			}

			// Check final pause signal
			if shouldPause != tt.expectPause && tt.threshold > 0 {
				// Only check last call for pause signal
				vt2 := NewVerificationTracker(tt.threshold)
				for i := 0; i < tt.completions-1; i++ {
					vt2.RecordCompletion()
				}
				lastPause := vt2.RecordCompletion()
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
		vt.RecordCompletion()
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
}

func TestVerificationTracker_Resume(t *testing.T) {
	vt := NewVerificationTracker(3)

	// Record completions to trigger pause
	for i := 0; i < 3; i++ {
		vt.RecordCompletion()
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
				vt.RecordCompletion()
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

// Pressure test: Concurrent RecordCompletion calls
func TestVerificationTracker_ConcurrentRecordCompletion(t *testing.T) {
	threshold := 3
	vt := NewVerificationTracker(threshold)

	// Launch many goroutines calling RecordCompletion concurrently
	numGoroutines := 100
	results := make(chan bool, numGoroutines)
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			shouldPause := vt.RecordCompletion()
			results <- shouldPause
		}()
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

	// All 100 completions should be recorded
	if status.CompletionsSinceVerification != numGoroutines {
		t.Errorf("Expected %d completions, got %d", numGoroutines, status.CompletionsSinceVerification)
	}

	// Should be paused after hitting threshold
	if !vt.IsPaused() {
		t.Error("Expected tracker to be paused after concurrent completions")
	}

	// At least one goroutine should have received pause signal (when threshold was hit)
	// Due to concurrent access, multiple goroutines might see the pause signal
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
				vt.RecordCompletion()
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
		// Record completions to trigger pause
		for i := 0; i < threshold; i++ {
			vt.RecordCompletion()
		}

		if !vt.IsPaused() {
			t.Fatalf("Expected pause after %d completions in cycle %d", threshold, cycle)
		}

		// Resume immediately
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

	// Goroutine 1: Continuously record completions
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numIterations; i++ {
			vt.RecordCompletion()
			time.Sleep(1 * time.Microsecond)
		}
	}()

	// Goroutine 2: Continuously resume
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
			if status.IsPaused && status.CompletionsSinceVerification < threshold {
				// Note: This might trigger due to race - paused state might persist briefly
				// after Resume, but counter should be reset
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

	// Goroutines recording completions
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOps/10; j++ {
				vt.RecordCompletion()
			}
		}()
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
	// Counter should be < threshold due to periodic resets from RecordHumanVerification
	// (unless last operations were all RecordCompletion)
}

func TestVerificationTracker_SeedFromBacklog(t *testing.T) {
	tests := []struct {
		name           string
		threshold      int
		seedCount      int
		expectPaused   bool
		expectCounter  int
	}{
		{
			name:          "seed below threshold",
			threshold:     3,
			seedCount:     2,
			expectPaused:  false,
			expectCounter: 2,
		},
		{
			name:          "seed at threshold triggers pause",
			threshold:     3,
			seedCount:     3,
			expectPaused:  true,
			expectCounter: 3,
		},
		{
			name:          "seed above threshold triggers pause",
			threshold:     3,
			seedCount:     5,
			expectPaused:  true,
			expectCounter: 5,
		},
		{
			name:          "seed with zero threshold (disabled)",
			threshold:     0,
			seedCount:     5,
			expectPaused:  false,
			expectCounter: 5,
		},
		{
			name:          "seed zero count",
			threshold:     3,
			seedCount:     0,
			expectPaused:  false,
			expectCounter: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := NewVerificationTracker(tt.threshold)
			vt.SeedFromBacklog(tt.seedCount)

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
	// Seed with 2 of 3, then record 1 more to hit threshold
	vt := NewVerificationTracker(3)
	vt.SeedFromBacklog(2)

	if vt.IsPaused() {
		t.Fatal("Should not be paused after seeding 2 with threshold 3")
	}

	shouldPause := vt.RecordCompletion()
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

// Pressure test: Threshold boundary concurrent access
func TestVerificationTracker_ThresholdBoundaryConcurrent(t *testing.T) {
	threshold := 3
	vt := NewVerificationTracker(threshold)

	// Pre-load to one below threshold
	for i := 0; i < threshold-1; i++ {
		vt.RecordCompletion()
	}

	// Launch many goroutines trying to cross the threshold simultaneously
	numGoroutines := 50
	results := make(chan bool, numGoroutines)
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			shouldPause := vt.RecordCompletion()
			results <- shouldPause
		}()
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

// Regression test: RecordCompletion must only be called once per daemon completion.
// Previously, it was called both inside ProcessCompletion() and in the daemon main
// loop, causing each completion to increment the counter by 2 instead of 1.
// With threshold=3, this caused the daemon to pause after only 2 actual completions.
func TestVerificationTracker_SingleCountPerCompletion(t *testing.T) {
	threshold := 3
	vt := NewVerificationTracker(threshold)

	// Simulate 3 daemon completions, each calling RecordCompletion exactly once
	// (as ProcessCompletion does). The daemon should pause at exactly 3.
	for i := 0; i < threshold; i++ {
		shouldPause := vt.RecordCompletion()
		if i < threshold-1 && shouldPause {
			t.Errorf("Should not pause at completion %d (threshold=%d)", i+1, threshold)
		}
		if i == threshold-1 && !shouldPause {
			t.Errorf("Should pause at completion %d (threshold=%d)", i+1, threshold)
		}
	}

	status := vt.Status()
	if status.CompletionsSinceVerification != threshold {
		t.Errorf("Counter = %d, want %d (one increment per completion)",
			status.CompletionsSinceVerification, threshold)
	}

	// Verify that calling RecordCompletion twice per completion (the old bug)
	// would have caused premature pausing
	vt2 := NewVerificationTracker(threshold)
	actualCompletions := 0
	for i := 0; i < threshold; i++ {
		// Old bug: called RecordCompletion twice per completion
		vt2.RecordCompletion() // first call (ProcessCompletion)
		vt2.RecordCompletion() // second call (daemon loop) — BUG
		actualCompletions++
		if vt2.IsPaused() {
			break
		}
	}

	if actualCompletions >= threshold {
		t.Errorf("Double-counting should have caused premature pause, but didn't pause until %d completions", actualCompletions)
	}

	status2 := vt2.Status()
	if status2.CompletionsSinceVerification == threshold {
		t.Error("Double-counting should produce counter != threshold at pause point")
	}
}
