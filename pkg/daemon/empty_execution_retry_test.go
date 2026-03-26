package daemon

import (
	"fmt"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

// --- EmptyExecutionRetryTracker tests ---

func TestEmptyExecutionRetryTracker_FirstAttempt(t *testing.T) {
	tracker := NewEmptyExecutionRetryTracker()
	if tracker.HasRetried("issue-001") {
		t.Error("HasRetried should be false for unseen issue")
	}
}

func TestEmptyExecutionRetryTracker_MarkAndCheck(t *testing.T) {
	tracker := NewEmptyExecutionRetryTracker()
	tracker.MarkRetried("issue-001")
	if !tracker.HasRetried("issue-001") {
		t.Error("HasRetried should be true after MarkRetried")
	}
}

func TestEmptyExecutionRetryTracker_IndependentIssues(t *testing.T) {
	tracker := NewEmptyExecutionRetryTracker()
	tracker.MarkRetried("issue-001")
	if tracker.HasRetried("issue-002") {
		t.Error("Different issue should not be marked as retried")
	}
}

func TestEmptyExecutionRetryTracker_Clear(t *testing.T) {
	tracker := NewEmptyExecutionRetryTracker()
	tracker.MarkRetried("issue-001")
	tracker.Clear("issue-001")
	if tracker.HasRetried("issue-001") {
		t.Error("HasRetried should be false after Clear")
	}
}

func TestEmptyExecutionRetryTracker_ClearUnknown(t *testing.T) {
	tracker := NewEmptyExecutionRetryTracker()
	// Should not panic
	tracker.Clear("nonexistent")
}

// --- Orphan detection with empty-execution classification tests ---

func TestOrphanDetection_EmptyExecution_FirstAttemptRetries(t *testing.T) {
	resetCalled := map[string]string{}
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
				return &opencode.OutcomeDetail{
					Outcome: opencode.OutcomeEmptyExecution,
					Reason:  "zero output tokens and no substantive content",
				}, nil
			},
		},
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "empty-001", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Empty exec task"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return false },
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error {
				resetCalled[beadsID] = status
				return nil
			},
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}

	// First empty-execution should reset to open (allows respawn)
	if resetCalled["empty-001"] != "open" {
		t.Errorf("Should reset empty-001 to open for retry, got status=%q", resetCalled["empty-001"])
	}
	if result.ResetCount != 1 {
		t.Errorf("ResetCount = %d, want 1", result.ResetCount)
	}
	// Tracker should now mark this issue as retried
	if !retryTracker.HasRetried("empty-001") {
		t.Error("Should mark empty-001 as retried after first empty-execution")
	}
	// Should have empty-execution retry metadata in result
	if len(result.EmptyExecutionRetries) != 1 {
		t.Fatalf("EmptyExecutionRetries = %d, want 1", len(result.EmptyExecutionRetries))
	}
	if result.EmptyExecutionRetries[0].BeadsID != "empty-001" {
		t.Errorf("Retry BeadsID = %s, want empty-001", result.EmptyExecutionRetries[0].BeadsID)
	}
	if result.EmptyExecutionRetries[0].Attempt != 1 {
		t.Errorf("Retry Attempt = %d, want 1", result.EmptyExecutionRetries[0].Attempt)
	}
}

func TestOrphanDetection_EmptyExecution_SecondAttemptEscalates(t *testing.T) {
	resetCalled := map[string]string{}
	retryTracker := NewEmptyExecutionRetryTracker()
	retryTracker.MarkRetried("empty-001") // Already retried once

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
					Reason:  "zero output tokens and no substantive content",
				}, nil
			},
		},
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "empty-001", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Empty exec task"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return false },
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error {
				resetCalled[beadsID] = status
				return nil
			},
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}

	// Second empty-execution should NOT reset to open — escalate instead
	if _, wasReset := resetCalled["empty-001"]; wasReset {
		t.Error("Should NOT reset empty-001 to open on second empty-execution (should escalate)")
	}
	if result.ResetCount != 0 {
		t.Errorf("ResetCount = %d, want 0 (escalated, not reset)", result.ResetCount)
	}
	// Should have escalation metadata
	if len(result.EmptyExecutionEscalations) != 1 {
		t.Fatalf("EmptyExecutionEscalations = %d, want 1", len(result.EmptyExecutionEscalations))
	}
	if result.EmptyExecutionEscalations[0].BeadsID != "empty-001" {
		t.Errorf("Escalation BeadsID = %s, want empty-001", result.EmptyExecutionEscalations[0].BeadsID)
	}
}

func TestOrphanDetection_NormalOrphan_NoClassifier(t *testing.T) {
	// When no classifier is set, existing behavior is preserved
	resetCalled := map[string]string{}
	cfg := Config{
		OrphanDetectionEnabled:  true,
		OrphanDetectionInterval: 30 * time.Minute,
		OrphanAgeThreshold:      time.Hour,
	}
	d := &Daemon{
		Config:        cfg,
		Scheduler:     NewSchedulerFromConfig(cfg),
		SpawnedIssues: NewSpawnedIssueTracker(),
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "orphan-001", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Normal orphan"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return false },
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error {
				resetCalled[beadsID] = status
				return nil
			},
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}
	if resetCalled["orphan-001"] != "open" {
		t.Errorf("Should reset orphan-001 to open (no classifier = existing behavior)")
	}
	if result.ResetCount != 1 {
		t.Errorf("ResetCount = %d, want 1", result.ResetCount)
	}
}

func TestOrphanDetection_NormalCompletion_ClassifiedAsNonEmpty(t *testing.T) {
	// When classifier returns normal-completion, treat as normal orphan (reset to open)
	resetCalled := map[string]string{}
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
				return &opencode.OutcomeDetail{
					Outcome:      opencode.OutcomeNormalCompletion,
					Reason:       "assistant produced output",
					OutputTokens: 1500,
				}, nil
			},
		},
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "orphan-001", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Normal orphan"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return false },
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error {
				resetCalled[beadsID] = status
				return nil
			},
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}
	// Non-empty outcome → normal orphan reset, no retry tracking
	if resetCalled["orphan-001"] != "open" {
		t.Errorf("Should reset orphan-001 to open (normal completion)")
	}
	if retryTracker.HasRetried("orphan-001") {
		t.Error("Should NOT mark non-empty-execution as retried")
	}
	if len(result.EmptyExecutionRetries) != 0 {
		t.Errorf("EmptyExecutionRetries = %d, want 0", len(result.EmptyExecutionRetries))
	}
}

func TestOrphanDetection_ClassifierError_FallsBackToNormalBehavior(t *testing.T) {
	// When classifier errors, fall back to existing behavior (reset to open)
	resetCalled := map[string]string{}
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
				return nil, fmt.Errorf("opencode unavailable")
			},
		},
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "orphan-001", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Orphan"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return false },
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error {
				resetCalled[beadsID] = status
				return nil
			},
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}
	if resetCalled["orphan-001"] != "open" {
		t.Errorf("Should fall back to normal reset on classifier error")
	}
	if result.ResetCount != 1 {
		t.Errorf("ResetCount = %d, want 1", result.ResetCount)
	}
}

func TestOrphanDetection_EmptyExecution_MixedWithNormalOrphans(t *testing.T) {
	retryTracker := NewEmptyExecutionRetryTracker()
	retryTracker.MarkRetried("empty-already-retried") // This one already retried

	resetCalled := map[string]string{}
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
				switch beadsID {
				case "empty-first-time":
					return &opencode.OutcomeDetail{
						Outcome: opencode.OutcomeEmptyExecution,
						Reason:  "no messages in session",
					}, nil
				case "empty-already-retried":
					return &opencode.OutcomeDetail{
						Outcome: opencode.OutcomeEmptyExecution,
						Reason:  "zero output tokens",
					}, nil
				case "normal-orphan":
					return &opencode.OutcomeDetail{
						Outcome:      opencode.OutcomeNormalCompletion,
						Reason:       "assistant produced output",
						OutputTokens: 500,
					}, nil
				}
				return nil, nil
			},
		},
		Agents: &mockAgentDiscoverer{
			GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
				return []ActiveAgent{
					{BeadsID: "empty-first-time", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Empty first"},
					{BeadsID: "empty-already-retried", Phase: "Planning", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Empty second"},
					{BeadsID: "normal-orphan", Phase: "Implementing", UpdatedAt: time.Now().Add(-2 * time.Hour), Title: "Normal orphan"},
				}, nil
			},
			HasExistingSessionFunc: func(beadsID string) bool { return false },
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID, status string) error {
				resetCalled[beadsID] = status
				return nil
			},
		},
	}

	result := d.RunPeriodicOrphanDetection()
	if result == nil {
		t.Fatal("Should return result")
	}

	// empty-first-time: classified as empty, first attempt → reset to open (retry)
	if resetCalled["empty-first-time"] != "open" {
		t.Error("empty-first-time should be reset to open (first retry)")
	}
	// empty-already-retried: classified as empty, second attempt → escalate (no reset)
	if _, wasReset := resetCalled["empty-already-retried"]; wasReset {
		t.Error("empty-already-retried should NOT be reset (should escalate)")
	}
	// normal-orphan: classified as normal → reset to open
	if resetCalled["normal-orphan"] != "open" {
		t.Error("normal-orphan should be reset to open (normal orphan)")
	}

	// Counts: 2 reset (empty-first-time + normal-orphan), 1 skipped (escalated)
	if result.ResetCount != 2 {
		t.Errorf("ResetCount = %d, want 2", result.ResetCount)
	}
	if len(result.EmptyExecutionRetries) != 1 {
		t.Errorf("EmptyExecutionRetries = %d, want 1", len(result.EmptyExecutionRetries))
	}
	if len(result.EmptyExecutionEscalations) != 1 {
		t.Errorf("EmptyExecutionEscalations = %d, want 1", len(result.EmptyExecutionEscalations))
	}
}

// --- mock ---

// mockEmptyExecutionClassifier implements EmptyExecutionClassifier for tests.
type mockEmptyExecutionClassifier struct {
	ClassifyFunc func(beadsID string) (*opencode.OutcomeDetail, error)
}

func (m *mockEmptyExecutionClassifier) ClassifyLastSession(beadsID string) (*opencode.OutcomeDetail, error) {
	if m.ClassifyFunc != nil {
		return m.ClassifyFunc(beadsID)
	}
	return nil, nil
}
