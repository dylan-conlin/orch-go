package daemon

import (
	"testing"
)

// TestDigestGate_ReExports verifies that the re-exported types and functions
// from pkg/digest are accessible through pkg/daemon for backward compatibility.
func TestDigestGate_ReExports(t *testing.T) {
	// Verify constructors work through re-exports
	f := NewDigestFeedbackState()
	if f == nil {
		t.Fatal("NewDigestFeedbackState should return non-nil")
	}

	f.RecordProduct("thread_progression")
	if f.TotalProduced() != 1 {
		t.Errorf("TotalProduced = %d, want 1", f.TotalProduced())
	}

	// Verify constants are accessible
	if MinProductsForAdaptation != 10 {
		t.Errorf("MinProductsForAdaptation = %d, want 10", MinProductsForAdaptation)
	}
	if MaturityWindowDays != 14 {
		t.Errorf("MaturityWindowDays = %d, want 14", MaturityWindowDays)
	}
}
