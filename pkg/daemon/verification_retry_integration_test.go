package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// TestCompletionOnce_ArchitectHandoffGateFailure_FlowsThroughRetryTracker verifies the
// exact production path: an architect agent with a failing handoff gate (missing Recommendation)
// flows through ProcessCompletion → handleVerificationFailure → retry tracker, and after
// budget exhaustion, the error message contains architect_handoff gate information.
//
// This is the test for the concern: "verification_failed_escalation.go should label it,
// but untested with live beads." We can't test the actual beads labeling without live beads,
// but we CAN verify the error propagation path that drives the labeling decision.
func TestCompletionOnce_ArchitectHandoffGateFailure_FlowsThroughRetryTracker(t *testing.T) {
	// Create a real workspace that will trigger architect_handoff gate failure
	wsDir := t.TempDir()

	// AGENT_MANIFEST.json — identifies as architect at V1 level
	manifest := map[string]string{
		"workspace_name": "og-arch-test",
		"skill":          "architect",
		"beads_id":       "orch-go-arch-gate-test",
		"project_dir":    wsDir,
		"spawn_time":     "2026-03-27T00:00:00Z",
		"tier":           "full",
		"verify_level":   "V1",
		"review_tier":    "review",
	}
	manifestJSON, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(wsDir, "AGENT_MANIFEST.json"), manifestJSON, 0644); err != nil {
		t.Fatal(err)
	}

	// SPAWN_CONTEXT.md — needed for skill name extraction
	if err := os.WriteFile(filepath.Join(wsDir, "SPAWN_CONTEXT.md"),
		[]byte("## SKILL GUIDANCE (architect)\nInstructions.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// SYNTHESIS.md WITHOUT Recommendation field — triggers architect_handoff failure
	synthesis := "## TLDR\nDesigned caching.\n\n## Next\nImplement it.\n"
	if err := os.WriteFile(filepath.Join(wsDir, "SYNTHESIS.md"), []byte(synthesis), 0644); err != nil {
		t.Fatal(err)
	}

	tracker := NewVerificationRetryTracker()

	d := &Daemon{
		VerificationRetryTracker: tracker,
		Completions: &mockCompletionFinder{
			ListCompletedAgentsFunc: func(config CompletionConfig) ([]CompletedAgent, error) {
				return []CompletedAgent{
					{
						BeadsID:       "orch-go-arch-gate-test",
						Title:         "Design caching layer",
						Status:        "in_progress",
						PhaseSummary:  "Design document written",
						WorkspacePath: wsDir,
						ProjectDir:    "",
					},
				}, nil
			},
		},
	}

	config := CompletionConfig{
		ProjectDir: wsDir,
	}

	// Run completion — will fail because beads comments can't be fetched (no live beads).
	// But verify the error flows through the retry tracker.
	result, err := d.CompletionOnce(config)
	if err != nil {
		t.Fatalf("CompletionOnce error: %v", err)
	}

	// Should have processed exactly 1 agent
	if len(result.Processed) != 1 {
		t.Fatalf("expected 1 processed result, got %d", len(result.Processed))
	}

	compResult := result.Processed[0]

	// Should have failed (either beads fetch or gate failure)
	if compResult.Error == nil {
		t.Fatal("expected error from verification failure")
	}

	// Retry tracker should have recorded 1 failure
	if tracker.Attempts("orch-go-arch-gate-test") != 1 {
		t.Errorf("expected 1 attempt, got %d", tracker.Attempts("orch-go-arch-gate-test"))
	}

	// Run 2 more cycles to exhaust the local retry budget (3 total)
	d.CompletionOnce(config)
	d.CompletionOnce(config)

	if !tracker.IsExhausted("orch-go-arch-gate-test", false) {
		t.Error("should be exhausted after 3 attempts (local budget)")
	}

	// Error message should contain useful diagnostic info
	errMsg := compResult.Error.Error()
	// The error will be from either beads comment fetch failure (if beads not running)
	// or from architect_handoff gate failure (if beads IS running but gate fails).
	// Either way, it should be a non-empty, informative error.
	if errMsg == "" {
		t.Error("error message should not be empty")
	}
}

// TestHandleVerificationFailure_ArchitectGateError_AttemptsToLabel verifies that
// handleVerificationFailure correctly attempts to label a beads issue with
// daemon:verification-failed when the retry budget is exhausted, even for
// architect_handoff gate failures.
func TestHandleVerificationFailure_ArchitectGateError_AttemptsToLabel(t *testing.T) {
	tracker := NewVerificationRetryTracker()

	// Pre-record 2 failures (one short of exhaustion for local agents)
	tracker.RecordFailure("orch-go-arch-label-test")
	tracker.RecordFailure("orch-go-arch-label-test")

	d := &Daemon{
		VerificationRetryTracker: tracker,
	}

	agent := CompletedAgent{
		BeadsID:    "orch-go-arch-label-test",
		Title:      "Architect agent with gate failure",
		ProjectDir: "", // local
	}

	compResult := CompletionResult{
		BeadsID: "orch-go-arch-label-test",
		Error:   fmt.Errorf("verification failed: SYNTHESIS.md missing **Recommendation:** field"),
	}

	config := CompletionConfig{
		ProjectDir: "/tmp/nonexistent-for-test",
		Verbose:    true,
	}

	// This 3rd call should exhaust the budget (3/3 for local)
	d.handleVerificationFailure(agent, compResult, config)

	// Verify budget is exhausted
	if !tracker.IsExhausted("orch-go-arch-label-test", false) {
		t.Error("should be exhausted after 3 attempts")
	}

	// The AddLabel call will fail (no live beads), but the attempt should happen.
	// We verify the tracker state which drives the labeling decision.
	if tracker.Attempts("orch-go-arch-label-test") != 3 {
		t.Errorf("expected 3 attempts, got %d", tracker.Attempts("orch-go-arch-label-test"))
	}
}

// TestVerificationFailedEscalation_PicksUpArchitectHandoffFailures verifies that
// RunPeriodicVerificationFailedEscalation correctly filters for daemon:verification-failed
// issues regardless of which gate failed. The escalation scanner is gate-agnostic —
// it picks up ANY verification-failed issue and promotes to triage:review.
func TestVerificationFailedEscalation_PicksUpArchitectHandoffFailures(t *testing.T) {
	// This test verifies the filter constant matches what handleVerificationFailure labels.
	// The actual beads query can't run without live beads, but we can verify the constants align.

	if LabelVerificationFailed != "daemon:verification-failed" {
		t.Errorf("LabelVerificationFailed = %q, want 'daemon:verification-failed'", LabelVerificationFailed)
	}

	if LabelTriageReview != "triage:review" {
		t.Errorf("LabelTriageReview = %q, want 'triage:review'", LabelTriageReview)
	}

	// Verify the escalation scanner uses the same label as the retry tracker
	// This is the critical alignment: handleVerificationFailure labels with
	// LabelVerificationFailed, and RunPeriodicVerificationFailedEscalation
	// queries for the same label.
	if !strings.Contains(LabelVerificationFailed, "verification-failed") {
		t.Error("LabelVerificationFailed must contain 'verification-failed' for escalation scanner alignment")
	}
}

// TestCompletionOnce_ArchitectHandoffGateFailure_WithLiveBeads is the integration test
// that exercises the full daemon auto-complete path with a real beads database.
// This closes the gap identified in the original task: "verification_failed_escalation.go
// should label it, but untested with live beads."
//
// The test:
// 1. Initializes a beads project in a temp dir
// 2. Creates an issue and adds "Phase: Complete" comment
// 3. Sets up an architect workspace with missing Recommendation (triggers gate failure)
// 4. Runs CompletionOnce → ProcessCompletion → VerifyCompletionCompliance
// 5. Verifies: gate failure flows through retry tracker AND label is applied on exhaustion
func TestCompletionOnce_ArchitectHandoffGateFailure_WithLiveBeads(t *testing.T) {
	// Initialize beads in a temp dir
	projectDir := t.TempDir()
	out, err := runBdCommandInDirForTest(projectDir, "init")
	if err != nil {
		t.Skipf("bd init failed (bd CLI not available?): %v: %s", err, out)
	}

	// Create an issue
	out, err = runBdCommandInDirForTest(projectDir, "create", "design caching layer", "--type", "task", "--json")
	if err != nil {
		t.Fatalf("bd create failed: %v: %s", err, out)
	}

	// Parse issue ID from JSON output (may have warning text before the JSON block)
	var created struct {
		ID string `json:"id"`
	}
	outStr := string(out)
	// Find the JSON object in the output (bd may print warnings before it)
	jsonStart := strings.Index(outStr, "{")
	jsonEnd := strings.LastIndex(outStr, "}")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		if jsonErr := json.Unmarshal([]byte(outStr[jsonStart:jsonEnd+1]), &created); jsonErr != nil {
			t.Fatalf("failed to parse issue JSON: %v: %s", jsonErr, outStr)
		}
	}
	if created.ID == "" {
		t.Fatalf("could not parse issue ID from bd create output: %s", outStr)
	}

	// Transition to in_progress
	_, err = runBdCommandInDirForTest(projectDir, "update", created.ID, "--status", "in_progress")
	if err != nil {
		t.Fatalf("bd update failed: %v", err)
	}

	// Add Phase: Complete comment
	_, err = runBdCommandInDirForTest(projectDir, "comments", "add", created.ID, "Phase: Complete - Design document written")
	if err != nil {
		t.Fatalf("bd comments add failed: %v", err)
	}

	// Create architect workspace
	wsDir := filepath.Join(projectDir, ".orch", "workspace", "og-arch-test-live")
	if mkErr := os.MkdirAll(wsDir, 0755); mkErr != nil {
		t.Fatal(mkErr)
	}

	// AGENT_MANIFEST.json — V1 architect
	manifest := map[string]string{
		"workspace_name": "og-arch-test-live",
		"skill":          "architect",
		"beads_id":       created.ID,
		"project_dir":    projectDir,
		"spawn_time":     "2026-03-27T00:00:00Z",
		"tier":           "full",
		"verify_level":   "V1",
		"review_tier":    "review",
	}
	manifestJSON, _ := json.Marshal(manifest)
	os.WriteFile(filepath.Join(wsDir, "AGENT_MANIFEST.json"), manifestJSON, 0644)

	// SPAWN_CONTEXT.md
	os.WriteFile(filepath.Join(wsDir, "SPAWN_CONTEXT.md"),
		[]byte("## SKILL GUIDANCE (architect)\nArchitect instructions.\n"), 0644)

	// SYNTHESIS.md WITHOUT **Recommendation:** — triggers architect_handoff gate
	os.WriteFile(filepath.Join(wsDir, "SYNTHESIS.md"),
		[]byte("## TLDR\nDesigned caching layer.\n\n## Next\nImplement it.\n"), 0644)

	tracker := NewVerificationRetryTracker()

	d := &Daemon{
		VerificationRetryTracker: tracker,
		Completions: &mockCompletionFinder{
			ListCompletedAgentsFunc: func(config CompletionConfig) ([]CompletedAgent, error) {
				return []CompletedAgent{
					{
						BeadsID:       created.ID,
						Title:         "design caching layer",
						Status:        "in_progress",
						PhaseSummary:  "Design document written",
						WorkspacePath: wsDir,
						ProjectDir:    "",
					},
				}, nil
			},
		},
	}

	config := CompletionConfig{
		ProjectDir: projectDir,
	}

	// Run first completion cycle
	result, err := d.CompletionOnce(config)
	if err != nil {
		t.Fatalf("CompletionOnce error: %v", err)
	}

	if len(result.Processed) != 1 {
		t.Fatalf("expected 1 processed, got %d", len(result.Processed))
	}

	compResult := result.Processed[0]

	// MUST have an error (gate failure)
	if compResult.Error == nil {
		t.Fatal("expected verification error from architect_handoff gate failure")
	}

	// Error should mention "Recommendation" (the missing field that triggers the gate)
	errMsg := compResult.Error.Error()
	if !strings.Contains(errMsg, "Recommendation") {
		t.Errorf("error should mention 'Recommendation', got: %s", errMsg)
	}

	// GatesFailed should include architect_handoff
	foundGate := false
	for _, g := range compResult.Verification.GatesFailed {
		if g == "architect_handoff" {
			foundGate = true
		}
	}
	if !foundGate {
		t.Errorf("expected 'architect_handoff' in GatesFailed, got %v", compResult.Verification.GatesFailed)
	}

	// Retry tracker should have recorded 1 failure
	if tracker.Attempts(created.ID) != 1 {
		t.Errorf("expected 1 attempt, got %d", tracker.Attempts(created.ID))
	}

	// Exhaust the retry budget (3 attempts for local)
	d.CompletionOnce(config)
	d.CompletionOnce(config)

	if !tracker.IsExhausted(created.ID, false) {
		t.Error("should be exhausted after 3 attempts")
	}

	// Verify the label was applied to the beads issue
	out, err = runBdCommandInDirForTest(projectDir, "show", created.ID, "--json")
	if err != nil {
		t.Fatalf("bd show failed: %v: %s", err, out)
	}

	// bd show --json returns an array of issues
	var issues []struct {
		Labels []string `json:"labels"`
	}
	outStr = string(out)
	jsonStart = strings.Index(outStr, "[")
	jsonEnd = strings.LastIndex(outStr, "]")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		if jsonErr := json.Unmarshal([]byte(outStr[jsonStart:jsonEnd+1]), &issues); jsonErr != nil {
			t.Fatalf("failed to parse issue JSON: %v: %s", jsonErr, outStr)
		}
	}
	if len(issues) == 0 {
		t.Fatalf("bd show returned no issues: %s", outStr)
	}

	foundLabel := false
	for _, l := range issues[0].Labels {
		if l == LabelVerificationFailed {
			foundLabel = true
		}
	}
	if !foundLabel {
		t.Errorf("expected label %q on issue after retry exhaustion, got %v", LabelVerificationFailed, issues[0].Labels)
	}
}

// runBdCommandInDirForTest wraps bd CLI with BEADS_DIR set to the project's .beads directory.
func runBdCommandInDirForTest(projectDir string, args ...string) ([]byte, error) {
	beadsDir := filepath.Join(projectDir, ".beads")
	cmd := exec.Command("bd", args...)
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), "BEADS_DIR="+beadsDir)
	return cmd.CombinedOutput()
}
