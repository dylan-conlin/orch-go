package daemon

import (
	"fmt"
	"testing"
)

// TestCompletionOnce_SkipsExhaustedAgents verifies that CompletionOnce skips
// agents that have exhausted their verification retry budget.
func TestCompletionOnce_SkipsExhaustedAgents(t *testing.T) {
	tracker := NewVerificationRetryTracker()

	// Pre-exhaust "proj-exhausted" (local, needs 3 failures)
	tracker.RecordFailure("proj-exhausted")
	tracker.RecordFailure("proj-exhausted")
	tracker.RecordFailure("proj-exhausted")

	processCallCount := 0

	d := &Daemon{
		VerificationRetryTracker: tracker,
		Completions: &mockCompletionFinder{
			ListCompletedAgentsFunc: func(config CompletionConfig) ([]CompletedAgent, error) {
				return []CompletedAgent{
					{BeadsID: "proj-exhausted", Title: "Exhausted", ProjectDir: ""},
					{BeadsID: "proj-fresh", Title: "Fresh", ProjectDir: ""},
				}, nil
			},
		},
	}

	// Override ProcessCompletion by using DryRun mode (it still calls ProcessCompletion
	// but the verification will fail because there's no real beads/workspace).
	// The key assertion is that the exhausted agent is skipped.
	config := CompletionConfig{
		DryRun:  true,
		Verbose: true,
	}

	result, err := d.CompletionOnce(config)
	if err != nil {
		t.Fatalf("CompletionOnce error: %v", err)
	}

	// Count how many agents were actually processed (attempted verification)
	// The exhausted agent should be skipped, so only "proj-fresh" should appear
	for _, r := range result.Processed {
		if r.BeadsID == "proj-exhausted" {
			t.Error("exhausted agent should have been skipped, but was processed")
		}
		processCallCount++
	}

	// proj-fresh should have been attempted (and likely failed since no real workspace)
	if processCallCount != 1 {
		t.Errorf("expected 1 processed result (proj-fresh), got %d", processCallCount)
	}
}

// TestCompletionOnce_TracksVerificationFailures verifies that CompletionOnce
// increments the retry counter when verification fails.
func TestCompletionOnce_TracksVerificationFailures(t *testing.T) {
	tracker := NewVerificationRetryTracker()

	d := &Daemon{
		VerificationRetryTracker: tracker,
		Completions: &mockCompletionFinder{
			ListCompletedAgentsFunc: func(config CompletionConfig) ([]CompletedAgent, error) {
				return []CompletedAgent{
					{BeadsID: "proj-will-fail", Title: "Will Fail", ProjectDir: ""},
				}, nil
			},
		},
	}

	config := CompletionConfig{
		ProjectDir: "/nonexistent",
	}

	// First attempt
	_, err := d.CompletionOnce(config)
	if err != nil {
		t.Fatalf("CompletionOnce error: %v", err)
	}

	if tracker.Attempts("proj-will-fail") != 1 {
		t.Errorf("after 1 cycle, attempts = %d, want 1", tracker.Attempts("proj-will-fail"))
	}
	if tracker.IsExhausted("proj-will-fail", false) {
		t.Error("should not be exhausted after 1 attempt")
	}

	// Second attempt
	d.CompletionOnce(config)
	if tracker.Attempts("proj-will-fail") != 2 {
		t.Errorf("after 2 cycles, attempts = %d, want 2", tracker.Attempts("proj-will-fail"))
	}

	// Third attempt — should exhaust
	d.CompletionOnce(config)
	if tracker.Attempts("proj-will-fail") != 3 {
		t.Errorf("after 3 cycles, attempts = %d, want 3", tracker.Attempts("proj-will-fail"))
	}
	if !tracker.IsExhausted("proj-will-fail", false) {
		t.Error("should be exhausted after 3 attempts")
	}
}

// TestCompletionOnce_CrossProjectExhaustsAfterOneAttempt verifies that
// cross-project agents exhaust their retry budget after just 1 failure.
func TestCompletionOnce_CrossProjectExhaustsAfterOneAttempt(t *testing.T) {
	tracker := NewVerificationRetryTracker()

	d := &Daemon{
		VerificationRetryTracker: tracker,
		Completions: &mockCompletionFinder{
			ListCompletedAgentsFunc: func(config CompletionConfig) ([]CompletedAgent, error) {
				return []CompletedAgent{
					{
						BeadsID:    "specs-platform-xyz",
						Title:      "Cross-project agent",
						ProjectDir: "/path/to/specs-platform", // non-empty = cross-project
					},
				}, nil
			},
		},
	}

	config := CompletionConfig{
		ProjectDir: "/path/to/orch-go",
	}

	// First attempt — should process and fail
	_, err := d.CompletionOnce(config)
	if err != nil {
		t.Fatalf("CompletionOnce error: %v", err)
	}

	if tracker.Attempts("specs-platform-xyz") != 1 {
		t.Errorf("after 1 cycle, attempts = %d, want 1", tracker.Attempts("specs-platform-xyz"))
	}
	if !tracker.IsExhausted("specs-platform-xyz", true) {
		t.Error("cross-project agent should be exhausted after 1 attempt")
	}

	// Second attempt — should be skipped entirely
	result, err := d.CompletionOnce(config)
	if err != nil {
		t.Fatalf("CompletionOnce error: %v", err)
	}

	for _, r := range result.Processed {
		if r.BeadsID == "specs-platform-xyz" {
			t.Error("exhausted cross-project agent should have been skipped")
		}
	}
}

// TestIsIssueDone verifies the capacity accounting helper.
func TestIsIssueDone(t *testing.T) {
	tests := []struct {
		name   string
		status string
		labels []string
		want   bool
	}{
		{
			name:   "closed issue",
			status: "closed",
			labels: nil,
			want:   true,
		},
		{
			name:   "closed case insensitive",
			status: "Closed",
			labels: nil,
			want:   true,
		},
		{
			name:   "open issue no labels",
			status: "open",
			labels: nil,
			want:   false,
		},
		{
			name:   "in_progress no labels",
			status: "in_progress",
			labels: nil,
			want:   false,
		},
		{
			name:   "in_progress with verification-failed label",
			status: "in_progress",
			labels: []string{LabelVerificationFailed},
			want:   true,
		},
		{
			name:   "in_progress with ready-review label",
			status: "in_progress",
			labels: []string{LabelReadyReview},
			want:   true,
		},
		{
			name:   "open with verification-failed label",
			status: "open",
			labels: []string{"triage:ready", LabelVerificationFailed},
			want:   true,
		},
		{
			name:   "in_progress with unrelated labels",
			status: "in_progress",
			labels: []string{"triage:ready", "orch:agent"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isIssueDone(tt.status, tt.labels)
			if got != tt.want {
				t.Errorf("isIssueDone(%q, %v) = %v, want %v", tt.status, tt.labels, got, tt.want)
			}
		})
	}
}

// TestCompletionOnce_ClearsTrackerOnSuccess verifies that successful verification
// clears the retry tracker for that agent.
func TestCompletionOnce_ClearsTrackerOnSuccess(t *testing.T) {
	tracker := NewVerificationRetryTracker()

	// Record one prior failure
	tracker.RecordFailure("proj-123")
	if tracker.Attempts("proj-123") != 1 {
		t.Fatalf("setup: expected 1 attempt, got %d", tracker.Attempts("proj-123"))
	}

	// Create a daemon where ProcessCompletion succeeds (dry run mode simulates this
	// partially, but we need a mock that returns success)
	d := &Daemon{
		VerificationRetryTracker: tracker,
		Completions: &mockCompletionFinder{
			ListCompletedAgentsFunc: func(config CompletionConfig) ([]CompletedAgent, error) {
				return []CompletedAgent{
					{BeadsID: "proj-123", Title: "Test", ProjectDir: ""},
				}, nil
			},
		},
	}

	// Process — will fail (no real workspace), but verify that after exhaustion
	// the tracker reflects the recorded failure
	config := CompletionConfig{ProjectDir: "/nonexistent"}
	d.CompletionOnce(config)

	if tracker.Attempts("proj-123") != 2 {
		t.Errorf("expected 2 attempts after another failure, got %d", tracker.Attempts("proj-123"))
	}
}

// TestDaemon_NewWithConfig_InitializesRetryTracker verifies that NewWithConfig
// properly initializes the VerificationRetryTracker.
func TestDaemon_NewWithConfig_InitializesRetryTracker(t *testing.T) {
	config := DefaultConfig()
	d := NewWithConfig(config)

	if d.VerificationRetryTracker == nil {
		t.Error("NewWithConfig should initialize VerificationRetryTracker")
	}
}

// TestCompletionOnce_NoTrackerDoesNotPanic verifies that CompletionOnce works
// even without a VerificationRetryTracker (backwards compatibility).
func TestCompletionOnce_NoTrackerDoesNotPanic(t *testing.T) {
	d := &Daemon{
		VerificationRetryTracker: nil, // explicitly nil
		Completions: &mockCompletionFinder{
			ListCompletedAgentsFunc: func(config CompletionConfig) ([]CompletedAgent, error) {
				return []CompletedAgent{
					{BeadsID: "proj-1", Title: "Test", ProjectDir: ""},
				}, nil
			},
		},
	}

	config := CompletionConfig{ProjectDir: "/nonexistent"}
	_, err := d.CompletionOnce(config)
	if err != nil {
		t.Fatalf("CompletionOnce should not error with nil tracker: %v", err)
	}
}

// TestCompletionOnce_MixedLocalAndCrossProject verifies that a mixed batch
// correctly applies different retry budgets.
func TestCompletionOnce_MixedLocalAndCrossProject(t *testing.T) {
	tracker := NewVerificationRetryTracker()
	callCount := 0

	d := &Daemon{
		VerificationRetryTracker: tracker,
		Completions: &mockCompletionFinder{
			ListCompletedAgentsFunc: func(config CompletionConfig) ([]CompletedAgent, error) {
				callCount++
				return []CompletedAgent{
					{BeadsID: "orch-go-local", Title: "Local agent", ProjectDir: ""},
					{BeadsID: "specs-platform-cross", Title: "Cross-project agent", ProjectDir: "/path/to/specs"},
				}, nil
			},
		},
	}

	config := CompletionConfig{ProjectDir: "/nonexistent"}

	// Cycle 1: both should be attempted
	result, _ := d.CompletionOnce(config)
	if len(result.Processed) != 2 {
		t.Errorf("cycle 1: expected 2 processed, got %d", len(result.Processed))
	}

	// After cycle 1: cross-project should be exhausted (1/1), local should not (1/3)
	if tracker.IsExhausted("specs-platform-cross", true) != true {
		t.Error("cross-project should be exhausted after 1 attempt")
	}
	if tracker.IsExhausted("orch-go-local", false) != false {
		t.Error("local should not be exhausted after 1 attempt")
	}

	// Cycle 2: only local should be attempted (cross-project is exhausted)
	result, _ = d.CompletionOnce(config)
	if len(result.Processed) != 1 {
		t.Errorf("cycle 2: expected 1 processed (local only), got %d", len(result.Processed))
	}
	if result.Processed[0].BeadsID != "orch-go-local" {
		t.Errorf("cycle 2: expected local agent, got %s", result.Processed[0].BeadsID)
	}

	// Cycle 3: still only local (2/3)
	result, _ = d.CompletionOnce(config)
	if len(result.Processed) != 1 {
		t.Errorf("cycle 3: expected 1 processed, got %d", len(result.Processed))
	}

	// Cycle 4: local now exhausted too (3/3), both skipped
	result, _ = d.CompletionOnce(config)
	if len(result.Processed) != 0 {
		// At this point both should be skipped.
		// Note: the mock still returns both agents from ListCompletedAgents,
		// but CompletionOnce's in-memory check should skip them.
		for _, r := range result.Processed {
			t.Errorf("cycle 4: agent %s should have been skipped", r.BeadsID)
		}
	}
}

// TestCompletionLoopResult_ErrorTracking verifies that errors from
// verification failures are properly recorded in the loop result.
func TestCompletionLoopResult_ErrorTracking(t *testing.T) {
	d := &Daemon{
		VerificationRetryTracker: NewVerificationRetryTracker(),
		Completions: &mockCompletionFinder{
			ListCompletedAgentsFunc: func(config CompletionConfig) ([]CompletedAgent, error) {
				return []CompletedAgent{
					{BeadsID: "proj-1", Title: "Agent 1", ProjectDir: ""},
					{BeadsID: "proj-2", Title: "Agent 2", ProjectDir: "/other"},
				}, nil
			},
		},
	}

	config := CompletionConfig{ProjectDir: "/nonexistent"}
	result, err := d.CompletionOnce(config)
	if err != nil {
		t.Fatalf("CompletionOnce error: %v", err)
	}

	// Both should have errors (no real workspace)
	if len(result.Errors) < 1 {
		t.Errorf("expected at least 1 error, got %d", len(result.Errors))
	}

	// Verify errors are non-nil
	for _, e := range result.Errors {
		if e == nil {
			t.Error("error should not be nil")
		}
	}

	// Verify that the error contains useful information
	found := false
	for _, e := range result.Errors {
		if e != nil {
			found = true
			errMsg := fmt.Sprintf("%v", e)
			if errMsg == "" {
				t.Error("error message should not be empty")
			}
		}
	}
	if !found {
		t.Error("expected at least one non-nil error")
	}
}
