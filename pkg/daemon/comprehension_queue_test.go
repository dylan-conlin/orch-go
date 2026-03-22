package daemon

import (
	"fmt"
	"testing"
)

// mockComprehensionQuerier is a test double for ComprehensionQuerier.
type mockComprehensionQuerier struct {
	count int
	err   error
}

func (m *mockComprehensionQuerier) CountPending() (int, error) {
	return m.count, m.err
}

func TestCheckComprehensionThrottle_NilQuerier(t *testing.T) {
	allowed, count, threshold := CheckComprehensionThrottle(nil, 5)
	if !allowed {
		t.Error("nil querier should allow spawning")
	}
	if count != 0 {
		t.Errorf("nil querier count = %d, want 0", count)
	}
	if threshold != 5 {
		t.Errorf("threshold = %d, want 5", threshold)
	}
}

func TestCheckComprehensionThrottle_BelowThreshold(t *testing.T) {
	q := &mockComprehensionQuerier{count: 3}
	allowed, count, threshold := CheckComprehensionThrottle(q, 5)
	if !allowed {
		t.Error("should allow when below threshold")
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
	if threshold != 5 {
		t.Errorf("threshold = %d, want 5", threshold)
	}
}

func TestCheckComprehensionThrottle_AtThreshold(t *testing.T) {
	q := &mockComprehensionQuerier{count: 5}
	allowed, _, _ := CheckComprehensionThrottle(q, 5)
	if allowed {
		t.Error("should block when at threshold")
	}
}

func TestCheckComprehensionThrottle_AboveThreshold(t *testing.T) {
	q := &mockComprehensionQuerier{count: 8}
	allowed, _, _ := CheckComprehensionThrottle(q, 5)
	if allowed {
		t.Error("should block when above threshold")
	}
}

func TestCheckComprehensionThrottle_ErrorFailsOpen(t *testing.T) {
	q := &mockComprehensionQuerier{count: 0, err: fmt.Errorf("bd failed")}
	allowed, _, _ := CheckComprehensionThrottle(q, 5)
	if !allowed {
		t.Error("should fail-open on error")
	}
}

func TestCheckComprehensionThrottle_DefaultThreshold(t *testing.T) {
	q := &mockComprehensionQuerier{count: 3}
	_, _, threshold := CheckComprehensionThrottle(q, 0)
	if threshold != DefaultComprehensionThreshold {
		t.Errorf("default threshold = %d, want %d", threshold, DefaultComprehensionThreshold)
	}
}

func TestComprehensionLabelConstant(t *testing.T) {
	if LabelComprehensionPending != "comprehension:pending" {
		t.Errorf("LabelComprehensionPending = %q, want %q", LabelComprehensionPending, "comprehension:pending")
	}
}
