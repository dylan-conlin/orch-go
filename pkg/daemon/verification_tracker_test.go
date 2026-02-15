// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"os"
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
