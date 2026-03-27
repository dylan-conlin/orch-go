package daemon

import (
	"testing"
	"time"
)

func TestNewShutdownBudget_Defaults(t *testing.T) {
	b := NewShutdownBudget()

	if b.Total != 4*time.Second {
		t.Errorf("Total = %v, want 4s", b.Total)
	}
	if b.Reflection != 2500*time.Millisecond {
		t.Errorf("Reflection = %v, want 2.5s", b.Reflection)
	}
	if b.StatusCleanup != 500*time.Millisecond {
		t.Errorf("StatusCleanup = %v, want 500ms", b.StatusCleanup)
	}
	if b.LogFlush != 500*time.Millisecond {
		t.Errorf("LogFlush = %v, want 500ms", b.LogFlush)
	}
}

func TestShutdownBudget_AllocationsWithinTotal(t *testing.T) {
	b := NewShutdownBudget()
	sum := b.Reflection + b.StatusCleanup + b.LogFlush
	if sum > b.Total {
		t.Errorf("allocations sum %v exceeds total %v", sum, b.Total)
	}
}

func TestShutdownBudget_SafetyMargin(t *testing.T) {
	b := NewShutdownBudget()
	// launchd ExitTimeOut is 5s, budget must leave >= 1s safety margin
	launchdTimeout := 5 * time.Second
	margin := launchdTimeout - b.Total
	if margin < 1*time.Second {
		t.Errorf("safety margin %v < 1s (launchd=%v, budget=%v)", margin, launchdTimeout, b.Total)
	}
}

func TestShutdownBudget_Remaining(t *testing.T) {
	b := NewShutdownBudget()
	b.start = time.Now().Add(-2 * time.Second)
	remaining := b.Remaining()
	// Should be approximately 2s (4s total - 2s elapsed)
	if remaining < 1500*time.Millisecond || remaining > 2500*time.Millisecond {
		t.Errorf("Remaining() = %v, want ~2s", remaining)
	}
}

func TestShutdownBudget_Remaining_Expired(t *testing.T) {
	b := NewShutdownBudget()
	b.start = time.Now().Add(-5 * time.Second) // past total budget
	remaining := b.Remaining()
	if remaining != 0 {
		t.Errorf("Remaining() = %v, want 0 (budget expired)", remaining)
	}
}

func TestShutdownBudget_Begin(t *testing.T) {
	b := NewShutdownBudget()
	if !b.start.IsZero() {
		t.Fatal("start should be zero before Begin()")
	}
	b.Begin()
	if b.start.IsZero() {
		t.Fatal("start should be set after Begin()")
	}
	if time.Since(b.start) > 100*time.Millisecond {
		t.Errorf("start should be ~now, got %v ago", time.Since(b.start))
	}
}
