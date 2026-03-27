package daemon

import (
	"testing"
)

func TestInvariantChecker_DisabledWhenThresholdZero(t *testing.T) {
	ic := NewInvariantChecker(0, 5)
	result := ic.Check(&InvariantInput{
		ActiveCount: -1, // Would normally violate
		MaxAgents:   5,
	})
	if result.HasViolations() {
		t.Error("disabled checker should not report violations")
	}
	if ic.IsPaused() {
		t.Error("disabled checker should never pause")
	}
}

func TestInvariantChecker_NilInputFailsOpen(t *testing.T) {
	ic := NewInvariantChecker(3, 5)
	result := ic.Check(nil)
	if result.Error == nil {
		t.Error("nil input should set Error (fail-open)")
	}
	if result.HasViolations() {
		t.Error("nil input should not produce violations")
	}
	// Fail-open: should NOT increment violation count
	if ic.ViolationCount() != 0 {
		t.Errorf("nil input should not count as violation, got count %d", ic.ViolationCount())
	}
}

func TestInvariantChecker_ActiveCountNegative(t *testing.T) {
	ic := NewInvariantChecker(3, 5)
	result := ic.Check(&InvariantInput{
		ActiveCount: -1,
		MaxAgents:   5,
	})
	if !result.HasViolations() {
		t.Fatal("negative active count should be a violation")
	}
	if result.Violations[0].Name != "active-count-negative" {
		t.Errorf("expected active-count-negative, got %s", result.Violations[0].Name)
	}
	if result.Violations[0].Severity != "critical" {
		t.Errorf("expected critical severity, got %s", result.Violations[0].Severity)
	}
}

func TestInvariantChecker_ActiveCountExceedsCap(t *testing.T) {
	ic := NewInvariantChecker(3, 5)
	// 2x cap = 10, so 11 should violate
	result := ic.Check(&InvariantInput{
		ActiveCount: 11,
		MaxAgents:   5,
	})
	if !result.HasViolations() {
		t.Fatal("active count > 2x cap should be a violation")
	}
	if result.Violations[0].Name != "active-count-exceeds-cap" {
		t.Errorf("expected active-count-exceeds-cap, got %s", result.Violations[0].Name)
	}
}

func TestInvariantChecker_ActiveCountWithinRange(t *testing.T) {
	ic := NewInvariantChecker(3, 5)
	// Normal range: 0 to 10 (2x cap)
	for _, count := range []int{0, 1, 5, 10} {
		result := ic.Check(&InvariantInput{
			ActiveCount: count,
			MaxAgents:   5,
		})
		if result.HasViolations() {
			t.Errorf("active count %d should be valid (max=5, 2x cap=10)", count)
		}
	}
}

func TestInvariantChecker_CompletionMissingProjectDir(t *testing.T) {
	ic := NewInvariantChecker(3, 5)
	result := ic.Check(&InvariantInput{
		MaxAgents: 5,
		CompletedAgents: []CompletedAgent{
			{
				BeadsID:       "orch-go-abc1",
				WorkspacePath: "/path/to/workspace",
				ProjectDir:    "", // Missing!
			},
		},
	})
	if !result.HasViolations() {
		t.Fatal("cross-project agent with empty ProjectDir should be a violation")
	}
	if result.Violations[0].Name != "completion-missing-project-dir" {
		t.Errorf("expected completion-missing-project-dir, got %s", result.Violations[0].Name)
	}
	if result.Violations[0].Severity != "warning" {
		t.Errorf("expected warning severity, got %s", result.Violations[0].Severity)
	}
}

func TestInvariantChecker_CompletionWithProjectDirIsValid(t *testing.T) {
	ic := NewInvariantChecker(3, 5)
	result := ic.Check(&InvariantInput{
		MaxAgents: 5,
		CompletedAgents: []CompletedAgent{
			{
				BeadsID:       "orch-go-abc1",
				WorkspacePath: "/path/to/workspace",
				ProjectDir:    "/path/to/project",
			},
		},
	})
	if result.HasViolations() {
		t.Error("agent with ProjectDir set should not violate")
	}
}

func TestInvariantChecker_CompletionSyntheticBeadsID(t *testing.T) {
	ic := NewInvariantChecker(3, 5)
	result := ic.Check(&InvariantInput{
		MaxAgents: 5,
		CompletedAgents: []CompletedAgent{
			{
				BeadsID: "workspace-untracked-abc",
			},
		},
	})
	if !result.HasViolations() {
		t.Fatal("synthetic beads ID should be a violation")
	}
	if result.Violations[0].Name != "completion-synthetic-beads-id" {
		t.Errorf("expected completion-synthetic-beads-id, got %s", result.Violations[0].Name)
	}
}

func TestInvariantChecker_CompletionEmptyBeadsID(t *testing.T) {
	ic := NewInvariantChecker(3, 5)
	result := ic.Check(&InvariantInput{
		MaxAgents: 5,
		CompletedAgents: []CompletedAgent{
			{BeadsID: ""},
		},
	})
	if !result.HasViolations() {
		t.Fatal("empty beads ID should be a violation")
	}
}

func TestInvariantChecker_PauseAfterThreshold(t *testing.T) {
	ic := NewInvariantChecker(3, 5)

	// 3 consecutive violation cycles should trigger pause
	for i := 0; i < 3; i++ {
		ic.Check(&InvariantInput{
			ActiveCount: -1, // Always violates
			MaxAgents:   5,
		})
	}

	if !ic.IsPaused() {
		t.Error("should be paused after 3 consecutive violation cycles")
	}
	if ic.ViolationCount() != 3 {
		t.Errorf("expected violation count 3, got %d", ic.ViolationCount())
	}
}

func TestInvariantChecker_CleanCycleResetsCount(t *testing.T) {
	ic := NewInvariantChecker(3, 5)

	// 2 violation cycles
	for i := 0; i < 2; i++ {
		ic.Check(&InvariantInput{
			ActiveCount: -1,
			MaxAgents:   5,
		})
	}
	if ic.ViolationCount() != 2 {
		t.Fatalf("expected 2 violations, got %d", ic.ViolationCount())
	}

	// Clean cycle resets
	ic.Check(&InvariantInput{
		ActiveCount: 3,
		MaxAgents:   5,
	})
	if ic.ViolationCount() != 0 {
		t.Errorf("clean cycle should reset count to 0, got %d", ic.ViolationCount())
	}
	if ic.IsPaused() {
		t.Error("should not be paused after clean cycle")
	}
}

func TestInvariantChecker_Resume(t *testing.T) {
	ic := NewInvariantChecker(1, 5)

	// Trigger pause
	ic.Check(&InvariantInput{
		ActiveCount: -1,
		MaxAgents:   5,
	})
	if !ic.IsPaused() {
		t.Fatal("should be paused")
	}

	// Resume
	ic.Resume()
	if ic.IsPaused() {
		t.Error("should not be paused after Resume")
	}
	if ic.ViolationCount() != 0 {
		t.Errorf("violation count should be 0 after Resume, got %d", ic.ViolationCount())
	}
}

func TestInvariantChecker_CriticalCount(t *testing.T) {
	ic := NewInvariantChecker(3, 5)
	result := ic.Check(&InvariantInput{
		ActiveCount: -1, // critical
		MaxAgents:   5,
		CompletedAgents: []CompletedAgent{
			{BeadsID: "test-untracked-abc"}, // warning
		},
	})
	if result.CriticalCount() != 1 {
		t.Errorf("expected 1 critical violation (negative active count), got %d", result.CriticalCount())
	}
}

func TestInvariantChecker_MultipleViolationsInOneCycle(t *testing.T) {
	ic := NewInvariantChecker(3, 5)
	result := ic.Check(&InvariantInput{
		ActiveCount: -1,
		MaxAgents:   5,
	})
	if len(result.Violations) != 1 {
		t.Errorf("expected 1 violation (negative active count), got %d", len(result.Violations))
	}
}

func TestInvariantChecker_MaxAgentsZeroSkipsRangeCheck(t *testing.T) {
	ic := NewInvariantChecker(3, 0)
	// When MaxAgents is 0 (unlimited), active count range check is skipped
	result := ic.Check(&InvariantInput{
		ActiveCount: 100,
		MaxAgents:   0,
	})
	if result.HasViolations() {
		t.Error("unlimited concurrency should skip active count range check")
	}
}

func TestIsUntrackedOrSyntheticBeadsID(t *testing.T) {
	tests := []struct {
		id       string
		expected bool
	}{
		{"", true},
		{"orch-go-abc1", false},
		{"workspace-untracked-xyz", true},
		{"some-untracked-thing", true},
		{"price-watch-defg", false},
	}
	for _, tt := range tests {
		got := isUntrackedOrSyntheticBeadsID(tt.id)
		if got != tt.expected {
			t.Errorf("isUntrackedOrSyntheticBeadsID(%q) = %v, want %v", tt.id, got, tt.expected)
		}
	}
}
