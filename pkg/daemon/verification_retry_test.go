package daemon

import (
	"testing"
)

func TestVerificationRetryTracker_RecordFailure(t *testing.T) {
	tracker := NewVerificationRetryTracker()

	// First failure should return 1
	if n := tracker.RecordFailure("proj-123"); n != 1 {
		t.Errorf("RecordFailure first call = %d, want 1", n)
	}

	// Second failure should return 2
	if n := tracker.RecordFailure("proj-123"); n != 2 {
		t.Errorf("RecordFailure second call = %d, want 2", n)
	}

	// Different ID starts at 1
	if n := tracker.RecordFailure("proj-456"); n != 1 {
		t.Errorf("RecordFailure different ID = %d, want 1", n)
	}
}

func TestVerificationRetryTracker_IsExhausted_Local(t *testing.T) {
	tracker := NewVerificationRetryTracker()

	// Local agents get 3 attempts
	if tracker.IsExhausted("proj-123", false) {
		t.Error("should not be exhausted with 0 attempts")
	}

	tracker.RecordFailure("proj-123") // attempt 1
	if tracker.IsExhausted("proj-123", false) {
		t.Error("should not be exhausted with 1 attempt")
	}

	tracker.RecordFailure("proj-123") // attempt 2
	if tracker.IsExhausted("proj-123", false) {
		t.Error("should not be exhausted with 2 attempts")
	}

	tracker.RecordFailure("proj-123") // attempt 3
	if !tracker.IsExhausted("proj-123", false) {
		t.Error("should be exhausted with 3 attempts (DefaultMaxVerificationAttempts)")
	}
}

func TestVerificationRetryTracker_IsExhausted_CrossProject(t *testing.T) {
	tracker := NewVerificationRetryTracker()

	// Cross-project agents get 1 attempt
	if tracker.IsExhausted("specs-platform-123", true) {
		t.Error("should not be exhausted with 0 attempts")
	}

	tracker.RecordFailure("specs-platform-123") // attempt 1
	if !tracker.IsExhausted("specs-platform-123", true) {
		t.Error("should be exhausted with 1 attempt (CrossProjectMaxVerificationAttempts)")
	}
}

func TestVerificationRetryTracker_Clear(t *testing.T) {
	tracker := NewVerificationRetryTracker()

	tracker.RecordFailure("proj-123")
	tracker.RecordFailure("proj-123")
	tracker.RecordFailure("proj-123")

	if !tracker.IsExhausted("proj-123", false) {
		t.Error("should be exhausted before clear")
	}

	tracker.Clear("proj-123")

	if tracker.IsExhausted("proj-123", false) {
		t.Error("should not be exhausted after clear")
	}
	if tracker.Attempts("proj-123") != 0 {
		t.Errorf("Attempts after clear = %d, want 0", tracker.Attempts("proj-123"))
	}
}

func TestVerificationRetryTracker_Attempts(t *testing.T) {
	tracker := NewVerificationRetryTracker()

	if tracker.Attempts("proj-123") != 0 {
		t.Errorf("Attempts for unknown ID = %d, want 0", tracker.Attempts("proj-123"))
	}

	tracker.RecordFailure("proj-123")
	tracker.RecordFailure("proj-123")

	if tracker.Attempts("proj-123") != 2 {
		t.Errorf("Attempts = %d, want 2", tracker.Attempts("proj-123"))
	}
}

func TestMaxAttemptsFor(t *testing.T) {
	if MaxAttemptsFor(false) != DefaultMaxVerificationAttempts {
		t.Errorf("MaxAttemptsFor(local) = %d, want %d", MaxAttemptsFor(false), DefaultMaxVerificationAttempts)
	}
	if MaxAttemptsFor(true) != CrossProjectMaxVerificationAttempts {
		t.Errorf("MaxAttemptsFor(cross-project) = %d, want %d", MaxAttemptsFor(true), CrossProjectMaxVerificationAttempts)
	}
}

func TestVerificationRetryTracker_Constants(t *testing.T) {
	// Sanity check that constants are reasonable
	if DefaultMaxVerificationAttempts < 1 {
		t.Errorf("DefaultMaxVerificationAttempts = %d, must be >= 1", DefaultMaxVerificationAttempts)
	}
	if CrossProjectMaxVerificationAttempts < 1 {
		t.Errorf("CrossProjectMaxVerificationAttempts = %d, must be >= 1", CrossProjectMaxVerificationAttempts)
	}
	if CrossProjectMaxVerificationAttempts > DefaultMaxVerificationAttempts {
		t.Errorf("CrossProjectMaxVerificationAttempts (%d) should be <= DefaultMaxVerificationAttempts (%d)",
			CrossProjectMaxVerificationAttempts, DefaultMaxVerificationAttempts)
	}
}

func TestLabelConstants(t *testing.T) {
	if LabelVerificationFailed != "daemon:verification-failed" {
		t.Errorf("LabelVerificationFailed = %q, want 'daemon:verification-failed'", LabelVerificationFailed)
	}
	if LabelReadyReview != "daemon:ready-review" {
		t.Errorf("LabelReadyReview = %q, want 'daemon:ready-review'", LabelReadyReview)
	}
}
