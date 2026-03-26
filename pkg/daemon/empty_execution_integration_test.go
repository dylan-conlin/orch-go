package daemon

import (
	"fmt"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

// Integration tests: prove the full retry→escalate cycle works end-to-end.
// These tests simulate multiple orphan detection cycles to verify that:
// 1. First empty-execution → retry (reset to open)
// 2. Second empty-execution → escalate (no reset, no duplicate work)
// 3. Mixed agent populations are handled correctly across cycles

func TestIntegration_EmptyExecution_FullCycle_RetryThenEscalate(t *testing.T) {
	// This test simulates two consecutive orphan detection cycles for the same
	// issue. The first cycle should retry (reset to open), the second should
	// escalate (skip reset). This proves the tracker state persists across cycles.
	retryTracker := NewEmptyExecutionRetryTracker()
	statusUpdates := map[string][]string{} // beadsID → list of status updates

	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}

	makeOrphanAgent := func(beadsID, title string) []ActiveAgent {
		return []ActiveAgent{
			{BeadsID: beadsID, Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: title},
		}
	}

	makeDaemon := func(agents []ActiveAgent) *Daemon {
		return &Daemon{
			Config:                     cfg,
			Scheduler:                  NewSchedulerFromConfig(cfg),
			SpawnedIssues:              NewSpawnedIssueTracker(),
			EmptyExecutionRetryTracker: retryTracker, // Shared across cycles
			EmptyExecutionClassifier: &mockEmptyExecutionClassifier{
				ClassifyFunc: func(beadsID string) (*opencode.OutcomeDetail, error) {
					return &opencode.OutcomeDetail{
						Outcome: opencode.OutcomeEmptyExecution,
						Reason:  "zero output tokens and no substantive content",
					}, nil
				},
			},
			Agents: &mockAgentDiscoverer{
				GetActiveAgentsFunc:    func() ([]ActiveAgent, error) { return agents, nil },
				HasExistingSessionFunc: func(beadsID string) bool { return false },
			},
			StatusUpdater: &mockIssueUpdater{
				UpdateStatusFunc: func(beadsID, status string) error {
					statusUpdates[beadsID] = append(statusUpdates[beadsID], status)
					return nil
				},
			},
		}
	}

	// --- Cycle 1: First empty-execution → should retry ---
	d1 := makeDaemon(makeOrphanAgent("issue-retry-cycle", "GPT session died"))
	result1 := d1.RunPeriodicOrphanDetection()
	if result1 == nil {
		t.Fatal("Cycle 1: should return result")
	}
	if result1.ResetCount != 1 {
		t.Errorf("Cycle 1: ResetCount = %d, want 1", result1.ResetCount)
	}
	if len(result1.EmptyExecutionRetries) != 1 {
		t.Fatalf("Cycle 1: EmptyExecutionRetries = %d, want 1", len(result1.EmptyExecutionRetries))
	}
	if result1.EmptyExecutionRetries[0].Action != "retrying" {
		t.Errorf("Cycle 1: Action = %q, want %q", result1.EmptyExecutionRetries[0].Action, "retrying")
	}
	if result1.EmptyExecutionRetries[0].Attempt != 1 {
		t.Errorf("Cycle 1: Attempt = %d, want 1", result1.EmptyExecutionRetries[0].Attempt)
	}
	if len(result1.EmptyExecutionEscalations) != 0 {
		t.Errorf("Cycle 1: should have 0 escalations, got %d", len(result1.EmptyExecutionEscalations))
	}

	// Verify status was set to "open" exactly once
	if len(statusUpdates["issue-retry-cycle"]) != 1 {
		t.Fatalf("Cycle 1: status updates = %d, want 1", len(statusUpdates["issue-retry-cycle"]))
	}
	if statusUpdates["issue-retry-cycle"][0] != "open" {
		t.Errorf("Cycle 1: status = %q, want %q", statusUpdates["issue-retry-cycle"][0], "open")
	}

	// --- Cycle 2: Same issue fails again → should escalate, NOT retry ---
	d2 := makeDaemon(makeOrphanAgent("issue-retry-cycle", "GPT session died again"))
	result2 := d2.RunPeriodicOrphanDetection()
	if result2 == nil {
		t.Fatal("Cycle 2: should return result")
	}
	if result2.ResetCount != 0 {
		t.Errorf("Cycle 2: ResetCount = %d, want 0 (escalated, not reset)", result2.ResetCount)
	}
	if len(result2.EmptyExecutionEscalations) != 1 {
		t.Fatalf("Cycle 2: EmptyExecutionEscalations = %d, want 1", len(result2.EmptyExecutionEscalations))
	}
	if result2.EmptyExecutionEscalations[0].Action != "escalated" {
		t.Errorf("Cycle 2: Action = %q, want %q", result2.EmptyExecutionEscalations[0].Action, "escalated")
	}
	if result2.EmptyExecutionEscalations[0].Attempt != 2 {
		t.Errorf("Cycle 2: Attempt = %d, want 2", result2.EmptyExecutionEscalations[0].Attempt)
	}
	if len(result2.EmptyExecutionRetries) != 0 {
		t.Errorf("Cycle 2: should have 0 retries, got %d", len(result2.EmptyExecutionRetries))
	}

	// Verify no additional status update happened (escalation skips reset)
	if len(statusUpdates["issue-retry-cycle"]) != 1 {
		t.Errorf("Cycle 2: status updates = %d, want 1 (no new update on escalation)", len(statusUpdates["issue-retry-cycle"]))
	}
}

func TestIntegration_EmptyExecution_RecoveryAfterRetry_ClearTracker(t *testing.T) {
	// After a successful retry, Clear() removes the issue from the tracker.
	// If the issue fails again later, it should get another retry (not escalate).
	retryTracker := NewEmptyExecutionRetryTracker()

	// Simulate: first failure → retry → success → Clear
	retryTracker.MarkRetried("issue-recovered")
	if !retryTracker.HasRetried("issue-recovered") {
		t.Fatal("Should be marked as retried")
	}
	retryTracker.Clear("issue-recovered") // Successful completion clears tracker
	if retryTracker.HasRetried("issue-recovered") {
		t.Fatal("Should be cleared after recovery")
	}

	// Now if it fails again, it gets a fresh retry (attempt 1), not escalation
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:                     cfg,
		Scheduler:                  NewSchedulerFromConfig(cfg),
		SpawnedIssues:              NewSpawnedIssueTracker(),
		EmptyExecutionRetryTracker: retryTracker,
		EmptyExecutionClassifier: &mockEmptyExecutionClassifier{
			ClassifyFunc: func(beadsID string) (*opencode.OutcomeDetail, error) {
				return &opencode.OutcomeDetail{
					Outcome: opencode.OutcomeEmptyExecution,
					Reason:  "zero output tokens",
				}, nil
			},
		},
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "issue-recovered", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Re-failed"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return false },
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error { return nil },
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}
	// Should be attempt 1 again (retry), not attempt 2 (escalation)
	if len(result.EmptyExecutionRetries) != 1 {
		t.Fatalf("EmptyExecutionRetries = %d, want 1", len(result.EmptyExecutionRetries))
	}
	if result.EmptyExecutionRetries[0].Attempt != 1 {
		t.Errorf("Attempt = %d, want 1 (fresh retry after clear)", result.EmptyExecutionRetries[0].Attempt)
	}
	if result.ResetCount != 1 {
		t.Errorf("ResetCount = %d, want 1", result.ResetCount)
	}
}

func TestIntegration_EmptyExecution_NoDuplicateWork_ConcurrentIssues(t *testing.T) {
	// Multiple issues in the same cycle: each gets independent retry/escalate
	// tracking. No cross-contamination.
	retryTracker := NewEmptyExecutionRetryTracker()
	retryTracker.MarkRetried("issue-B") // B already retried once

	statusUpdates := map[string]string{}
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:                     cfg,
		Scheduler:                  NewSchedulerFromConfig(cfg),
		SpawnedIssues:              NewSpawnedIssueTracker(),
		EmptyExecutionRetryTracker: retryTracker,
		EmptyExecutionClassifier: &mockEmptyExecutionClassifier{
			ClassifyFunc: func(beadsID string) (*opencode.OutcomeDetail, error) {
				return &opencode.OutcomeDetail{
					Outcome: opencode.OutcomeEmptyExecution,
					Reason:  "zero output tokens",
				}, nil
			},
		},
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "issue-A", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Issue A (first fail)"},
					{BeadsID: "issue-B", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Issue B (second fail)"},
					{BeadsID: "issue-C", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Issue C (first fail)"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return false },
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error {
				statusUpdates[beadsID] = status
				return nil
			},
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}

	// A and C: first empty-execution → retry (reset)
	// B: second empty-execution → escalate (no reset)
	if result.ResetCount != 2 {
		t.Errorf("ResetCount = %d, want 2 (A + C)", result.ResetCount)
	}
	if len(result.EmptyExecutionRetries) != 2 {
		t.Errorf("EmptyExecutionRetries = %d, want 2 (A + C)", len(result.EmptyExecutionRetries))
	}
	if len(result.EmptyExecutionEscalations) != 1 {
		t.Errorf("EmptyExecutionEscalations = %d, want 1 (B)", len(result.EmptyExecutionEscalations))
	}

	// Verify: A and C got reset to open, B did not
	if statusUpdates["issue-A"] != "open" {
		t.Errorf("issue-A status = %q, want open", statusUpdates["issue-A"])
	}
	if _, wasReset := statusUpdates["issue-B"]; wasReset {
		t.Error("issue-B should NOT have been reset (escalated)")
	}
	if statusUpdates["issue-C"] != "open" {
		t.Errorf("issue-C status = %q, want open", statusUpdates["issue-C"])
	}

	// Verify tracker state: A and C now marked, B was already marked
	if !retryTracker.HasRetried("issue-A") {
		t.Error("issue-A should be marked as retried")
	}
	if !retryTracker.HasRetried("issue-B") {
		t.Error("issue-B should still be marked as retried")
	}
	if !retryTracker.HasRetried("issue-C") {
		t.Error("issue-C should be marked as retried")
	}
}

// --- GPT-5.4 / OpenCode session replay tests ---
// These simulate the specific empty-execution patterns observed with non-Anthropic
// models (GPT-5.4, Codex) routed through OpenCode. The key difference: these
// models have higher stall rates (67-87%) and their empty sessions have distinct
// signatures (zero assistant messages, or assistant messages with zero output tokens).

func TestIntegration_GPTSessionReplay_ZeroAssistantMessages(t *testing.T) {
	// GPT-5.4 pattern: session starts, user message sent, but model never responds.
	// OpenCode records the session with only user messages.
	session := opencode.Session{
		ID:    "gpt-stall-001",
		Time:  opencode.SessionTime{Created: time.Now().Add(-30 * time.Minute).UnixMilli()},
		Title: "GPT-5.4 stalled session",
	}
	messages := []opencode.Message{
		{
			Info: opencode.MessageInfo{
				ID:   "msg-user",
				Role: "user",
				Time: opencode.MessageTime{Created: time.Now().Add(-30 * time.Minute).UnixMilli()},
			},
			Parts: []opencode.MessagePart{
				{Type: "text", Text: "Implement the retry handler per the spawn context"},
			},
		},
	}

	detail := opencode.ClassifyTerminalOutcomeDetail(session, messages)
	if detail.Outcome != opencode.OutcomeEmptyExecution {
		t.Errorf("GPT stall: outcome = %v, want empty-execution", detail.Outcome)
	}
	if detail.Reason != "no assistant messages" {
		t.Errorf("GPT stall: reason = %q, want 'no assistant messages'", detail.Reason)
	}
	if detail.OutputTokens != 0 {
		t.Errorf("GPT stall: output tokens = %d, want 0", detail.OutputTokens)
	}

	// Now verify this classification triggers retry in the full orphan detection path
	retryTracker := NewEmptyExecutionRetryTracker()
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:                     cfg,
		Scheduler:                  NewSchedulerFromConfig(cfg),
		SpawnedIssues:              NewSpawnedIssueTracker(),
		EmptyExecutionRetryTracker: retryTracker,
		EmptyExecutionClassifier: &mockEmptyExecutionClassifier{
			ClassifyFunc: func(beadsID string) (*opencode.OutcomeDetail, error) {
				return &detail, nil
			},
		},
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "gpt-stall-001", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "GPT stalled"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return false },
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error { return nil },
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}
	if result.ResetCount != 1 {
		t.Errorf("ResetCount = %d, want 1 (retry GPT stall)", result.ResetCount)
	}
	if len(result.EmptyExecutionRetries) != 1 {
		t.Fatalf("EmptyExecutionRetries = %d, want 1", len(result.EmptyExecutionRetries))
	}
	if result.EmptyExecutionRetries[0].Reason != "no assistant messages" {
		t.Errorf("Reason = %q, want 'no assistant messages'", result.EmptyExecutionRetries[0].Reason)
	}
}

func TestIntegration_GPTSessionReplay_AssistantZeroTokens(t *testing.T) {
	// GPT-5.4/Codex pattern: model "responds" but with zero output tokens
	// and empty/whitespace-only text parts.
	session := opencode.Session{
		ID:    "gpt-empty-002",
		Time:  opencode.SessionTime{Created: time.Now().Add(-45 * time.Minute).UnixMilli()},
		Title: "GPT-5.4 empty response",
	}
	messages := []opencode.Message{
		{
			Info: opencode.MessageInfo{Role: "user"},
			Parts: []opencode.MessagePart{
				{Type: "text", Text: "Fix the authentication bug"},
			},
		},
		{
			Info: opencode.MessageInfo{
				Role:   "assistant",
				Tokens: &opencode.MessageToken{Input: 8000, Output: 0},
				Finish: "stop",
			},
			Parts: []opencode.MessagePart{
				{Type: "text", Text: "  \n  "}, // Whitespace-only "response"
			},
		},
	}

	detail := opencode.ClassifyTerminalOutcomeDetail(session, messages)
	if detail.Outcome != opencode.OutcomeEmptyExecution {
		t.Errorf("GPT empty response: outcome = %v, want empty-execution", detail.Outcome)
	}
	if detail.AssistantMessages != 1 {
		t.Errorf("GPT empty response: assistant messages = %d, want 1", detail.AssistantMessages)
	}

	// Verify full retry→escalate cycle with this classification
	retryTracker := NewEmptyExecutionRetryTracker()
	var resetCount int

	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	makeDaemon := func() *Daemon {
		return &Daemon{
			Config:                     cfg,
			Scheduler:                  NewSchedulerFromConfig(cfg),
			SpawnedIssues:              NewSpawnedIssueTracker(),
			EmptyExecutionRetryTracker: retryTracker,
			EmptyExecutionClassifier: &mockEmptyExecutionClassifier{
				ClassifyFunc: func(beadsID string) (*opencode.OutcomeDetail, error) {
					return &detail, nil
				},
			},
			Agents: &mockAgentDiscoverer{
				GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
					return []ActiveAgent{
						{BeadsID: "gpt-empty-002", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "GPT empty"},
					}, nil
				},
				HasExistingSessionFunc: func(beadsID string) bool { return false },
			},
			StatusUpdater: &mockIssueUpdater{
				UpdateStatusFunc: func(beadsID, status string) error {
					resetCount++
					return nil
				},
			},
		}
	}

	// Cycle 1: retry
	r1 := makeDaemon().RunPeriodicOrphanDetection()
	if r1.ResetCount != 1 {
		t.Errorf("Cycle 1: ResetCount = %d, want 1", r1.ResetCount)
	}

	// Cycle 2: escalate
	r2 := makeDaemon().RunPeriodicOrphanDetection()
	if r2.ResetCount != 0 {
		t.Errorf("Cycle 2: ResetCount = %d, want 0", r2.ResetCount)
	}
	if len(r2.EmptyExecutionEscalations) != 1 {
		t.Errorf("Cycle 2: escalations = %d, want 1", len(r2.EmptyExecutionEscalations))
	}

	// Total resets across both cycles: exactly 1 (no duplicate work)
	if resetCount != 1 {
		t.Errorf("Total status resets = %d, want 1 (retry once, then escalate)", resetCount)
	}
}

func TestIntegration_GPTSessionReplay_ErrorTermination_NotRetried(t *testing.T) {
	// GPT-5.4 error pattern: model hits a rate limit or error.
	// Error-terminated sessions should NOT be treated as empty-execution.
	session := opencode.Session{
		ID:    "gpt-error-003",
		Time:  opencode.SessionTime{Created: time.Now().Add(-20 * time.Minute).UnixMilli()},
		Title: "GPT-5.4 rate limited",
	}
	messages := []opencode.Message{
		{
			Info: opencode.MessageInfo{Role: "user"},
			Parts: []opencode.MessagePart{
				{Type: "text", Text: "Implement feature X"},
			},
		},
		{
			Info: opencode.MessageInfo{
				Role:   "assistant",
				Tokens: &opencode.MessageToken{Input: 500, Output: 50},
				Finish: "error",
			},
			Parts: []opencode.MessagePart{
				{Type: "text", Text: "Rate limit exceeded"},
			},
		},
	}

	detail := opencode.ClassifyTerminalOutcomeDetail(session, messages)
	if detail.Outcome != opencode.OutcomeErrorTermination {
		t.Errorf("GPT error: outcome = %v, want error-termination", detail.Outcome)
	}
	if detail.Outcome.IsEmpty() {
		t.Error("error-termination should NOT be classified as empty")
	}

	// When plugged into orphan detection, error-termination follows normal orphan path
	// (not the empty-execution retry path)
	retryTracker := NewEmptyExecutionRetryTracker()
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:                     cfg,
		Scheduler:                  NewSchedulerFromConfig(cfg),
		SpawnedIssues:              NewSpawnedIssueTracker(),
		EmptyExecutionRetryTracker: retryTracker,
		EmptyExecutionClassifier: &mockEmptyExecutionClassifier{
			ClassifyFunc: func(beadsID string) (*opencode.OutcomeDetail, error) {
				return &detail, nil
			},
		},
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "gpt-error-003", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "GPT error"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return false },
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error { return nil },
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}
	// Should be treated as normal orphan reset, NOT empty-execution retry
	if result.ResetCount != 1 {
		t.Errorf("ResetCount = %d, want 1 (normal orphan reset)", result.ResetCount)
	}
	if len(result.EmptyExecutionRetries) != 0 {
		t.Errorf("EmptyExecutionRetries = %d, want 0 (not an empty execution)", len(result.EmptyExecutionRetries))
	}
	if retryTracker.HasRetried("gpt-error-003") {
		t.Error("error-termination should NOT mark issue as retried")
	}
}

func TestIntegration_FaultInjection_ClassifierDown_FallsBack(t *testing.T) {
	// When the OpenCode API is down and classifier errors, the system should
	// fall back to normal orphan reset behavior — not block indefinitely.
	retryTracker := NewEmptyExecutionRetryTracker()
	var resetStatus string

	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:                     cfg,
		Scheduler:                  NewSchedulerFromConfig(cfg),
		SpawnedIssues:              NewSpawnedIssueTracker(),
		EmptyExecutionRetryTracker: retryTracker,
		EmptyExecutionClassifier: &mockEmptyExecutionClassifier{
			ClassifyFunc: func(beadsID string) (*opencode.OutcomeDetail, error) {
				return nil, fmt.Errorf("opencode API: connection refused")
			},
		},
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "fault-001", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Fault test"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return false },
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error {
				resetStatus = status
				return nil
			},
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}
	// Classifier error → graceful fallback to normal reset
	if result.ResetCount != 1 {
		t.Errorf("ResetCount = %d, want 1 (fallback to normal reset)", result.ResetCount)
	}
	if resetStatus != "open" {
		t.Errorf("Status = %q, want 'open'", resetStatus)
	}
	if len(result.EmptyExecutionRetries) != 0 {
		t.Errorf("EmptyExecutionRetries = %d, want 0 (classifier failed)", len(result.EmptyExecutionRetries))
	}
	// Should NOT contaminate the retry tracker
	if retryTracker.HasRetried("fault-001") {
		t.Error("Classifier failure should NOT mark issue as retried")
	}
}

func TestIntegration_FaultInjection_ClassifierReturnsNil(t *testing.T) {
	// Edge case: classifier returns (nil, nil) — no session data found.
	retryTracker := NewEmptyExecutionRetryTracker()

	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:                     cfg,
		Scheduler:                  NewSchedulerFromConfig(cfg),
		SpawnedIssues:              NewSpawnedIssueTracker(),
		EmptyExecutionRetryTracker: retryTracker,
		EmptyExecutionClassifier: &mockEmptyExecutionClassifier{
			ClassifyFunc: func(beadsID string) (*opencode.OutcomeDetail, error) {
				return nil, nil // No session data
			},
		},
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "nil-001", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Nil detail"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return false },
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error { return nil },
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}
	// nil detail → normal orphan handling (not treated as empty-execution)
	if result.ResetCount != 1 {
		t.Errorf("ResetCount = %d, want 1 (normal reset for nil detail)", result.ResetCount)
	}
	if len(result.EmptyExecutionRetries) != 0 {
		t.Errorf("EmptyExecutionRetries = %d, want 0", len(result.EmptyExecutionRetries))
	}
	if retryTracker.HasRetried("nil-001") {
		t.Error("Nil detail should NOT mark issue as retried")
	}
}
