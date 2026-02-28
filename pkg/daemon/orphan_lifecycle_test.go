package daemon

import (
	"fmt"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/agent"
)

// mockLifecycleManager implements agent.LifecycleManager for tests.
type mockLifecycleManager struct {
	DetectOrphansFunc  func(projectDirs []string, threshold time.Duration) (*agent.OrphanDetectionResult, error)
	ForceCompleteFunc  func(a agent.AgentRef, reason string) (*agent.TransitionEvent, error)
	ForceAbandonFunc   func(a agent.AgentRef) (*agent.TransitionEvent, error)
}

func (m *mockLifecycleManager) BeginSpawn(input agent.SpawnInput) (*agent.SpawnHandle, error) {
	return nil, nil
}
func (m *mockLifecycleManager) ActivateSpawn(handle *agent.SpawnHandle, sessionID string) (*agent.TransitionEvent, error) {
	return nil, nil
}
func (m *mockLifecycleManager) Complete(a agent.AgentRef, reason string) (*agent.TransitionEvent, error) {
	return nil, nil
}
func (m *mockLifecycleManager) Abandon(a agent.AgentRef, reason string) (*agent.TransitionEvent, error) {
	return nil, nil
}
func (m *mockLifecycleManager) CurrentState(a agent.AgentRef) (agent.State, error) {
	return "", nil
}

func (m *mockLifecycleManager) DetectOrphans(projectDirs []string, threshold time.Duration) (*agent.OrphanDetectionResult, error) {
	if m.DetectOrphansFunc != nil {
		return m.DetectOrphansFunc(projectDirs, threshold)
	}
	return &agent.OrphanDetectionResult{}, nil
}

func (m *mockLifecycleManager) ForceComplete(a agent.AgentRef, reason string) (*agent.TransitionEvent, error) {
	if m.ForceCompleteFunc != nil {
		return m.ForceCompleteFunc(a, reason)
	}
	return &agent.TransitionEvent{Success: true}, nil
}

func (m *mockLifecycleManager) ForceAbandon(a agent.AgentRef) (*agent.TransitionEvent, error) {
	if m.ForceAbandonFunc != nil {
		return m.ForceAbandonFunc(a)
	}
	return &agent.TransitionEvent{Success: true}, nil
}

func TestRunLifecycleOrphanRecovery_DetectionError(t *testing.T) {
	lm := &mockLifecycleManager{
		DetectOrphansFunc: func(projectDirs []string, threshold time.Duration) (*agent.OrphanDetectionResult, error) {
			return nil, fmt.Errorf("beads unavailable")
		},
	}

	result := RunLifecycleOrphanRecovery(lm, []string{"/tmp"}, time.Hour, false)
	if result.Error == nil {
		t.Error("Should return error when detection fails")
	}
	if result.ForceCompleted != 0 || result.ForceAbandoned != 0 {
		t.Error("Should not have recovered any agents on error")
	}
}

func TestRunLifecycleOrphanRecovery_NoOrphans(t *testing.T) {
	lm := &mockLifecycleManager{
		DetectOrphansFunc: func(projectDirs []string, threshold time.Duration) (*agent.OrphanDetectionResult, error) {
			return &agent.OrphanDetectionResult{
				Scanned: 5,
				Orphans: nil,
			}, nil
		},
	}

	result := RunLifecycleOrphanRecovery(lm, []string{"/tmp"}, time.Hour, false)
	if result.Error != nil {
		t.Errorf("Unexpected error: %v", result.Error)
	}
	if result.Scanned != 5 {
		t.Errorf("Scanned = %d, want 5", result.Scanned)
	}
	if result.ForceCompleted != 0 || result.ForceAbandoned != 0 {
		t.Error("Should not have recovered any agents when no orphans")
	}
}

func TestRunLifecycleOrphanRecovery_ForceCompletesPhaseComplete(t *testing.T) {
	forceCompleteCalled := map[string]bool{}
	lm := &mockLifecycleManager{
		DetectOrphansFunc: func(projectDirs []string, threshold time.Duration) (*agent.OrphanDetectionResult, error) {
			return &agent.OrphanDetectionResult{
				Scanned: 3,
				Orphans: []agent.OrphanedAgent{
					{
						Agent:     agent.AgentRef{BeadsID: "orphan-complete"},
						LastPhase: "Complete",
						StaleFor:  2 * time.Hour,
					},
				},
			}, nil
		},
		ForceCompleteFunc: func(a agent.AgentRef, reason string) (*agent.TransitionEvent, error) {
			forceCompleteCalled[a.BeadsID] = true
			return &agent.TransitionEvent{Success: true}, nil
		},
	}

	result := RunLifecycleOrphanRecovery(lm, []string{"/tmp"}, time.Hour, false)
	if result.Error != nil {
		t.Errorf("Unexpected error: %v", result.Error)
	}
	if result.ForceCompleted != 1 {
		t.Errorf("ForceCompleted = %d, want 1", result.ForceCompleted)
	}
	if !forceCompleteCalled["orphan-complete"] {
		t.Error("ForceComplete should have been called for Phase: Complete orphan")
	}
}

func TestRunLifecycleOrphanRecovery_ForceAbandonsNonComplete(t *testing.T) {
	forceAbandonCalled := map[string]bool{}
	lm := &mockLifecycleManager{
		DetectOrphansFunc: func(projectDirs []string, threshold time.Duration) (*agent.OrphanDetectionResult, error) {
			return &agent.OrphanDetectionResult{
				Scanned: 3,
				Orphans: []agent.OrphanedAgent{
					{
						Agent:     agent.AgentRef{BeadsID: "orphan-planning"},
						LastPhase: "Planning",
						StaleFor:  3 * time.Hour,
					},
				},
			}, nil
		},
		ForceAbandonFunc: func(a agent.AgentRef) (*agent.TransitionEvent, error) {
			forceAbandonCalled[a.BeadsID] = true
			return &agent.TransitionEvent{Success: true}, nil
		},
	}

	result := RunLifecycleOrphanRecovery(lm, []string{"/tmp"}, time.Hour, false)
	if result.Error != nil {
		t.Errorf("Unexpected error: %v", result.Error)
	}
	if result.ForceAbandoned != 1 {
		t.Errorf("ForceAbandoned = %d, want 1", result.ForceAbandoned)
	}
	if !forceAbandonCalled["orphan-planning"] {
		t.Error("ForceAbandon should have been called for non-Complete orphan")
	}
}

func TestRunLifecycleOrphanRecovery_MixedOrphans(t *testing.T) {
	lm := &mockLifecycleManager{
		DetectOrphansFunc: func(projectDirs []string, threshold time.Duration) (*agent.OrphanDetectionResult, error) {
			return &agent.OrphanDetectionResult{
				Scanned: 10,
				Orphans: []agent.OrphanedAgent{
					{Agent: agent.AgentRef{BeadsID: "a1"}, LastPhase: "Complete", StaleFor: 2 * time.Hour},
					{Agent: agent.AgentRef{BeadsID: "a2"}, LastPhase: "Planning", StaleFor: 3 * time.Hour},
					{Agent: agent.AgentRef{BeadsID: "a3"}, LastPhase: "Implementing", StaleFor: 4 * time.Hour},
					{Agent: agent.AgentRef{BeadsID: "a4"}, LastPhase: "Complete", StaleFor: 5 * time.Hour},
					{Agent: agent.AgentRef{BeadsID: ""}, LastPhase: "", StaleFor: 6 * time.Hour}, // No BeadsID → skip
				},
			}, nil
		},
		ForceCompleteFunc: func(a agent.AgentRef, reason string) (*agent.TransitionEvent, error) {
			return &agent.TransitionEvent{Success: true}, nil
		},
		ForceAbandonFunc: func(a agent.AgentRef) (*agent.TransitionEvent, error) {
			return &agent.TransitionEvent{Success: true}, nil
		},
	}

	result := RunLifecycleOrphanRecovery(lm, []string{"/tmp"}, time.Hour, false)
	if result.Error != nil {
		t.Errorf("Unexpected error: %v", result.Error)
	}
	if result.ForceCompleted != 2 {
		t.Errorf("ForceCompleted = %d, want 2", result.ForceCompleted)
	}
	if result.ForceAbandoned != 2 {
		t.Errorf("ForceAbandoned = %d, want 2", result.ForceAbandoned)
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1 (empty BeadsID)", result.Skipped)
	}
}

func TestRunLifecycleOrphanRecovery_ForceCompleteError(t *testing.T) {
	lm := &mockLifecycleManager{
		DetectOrphansFunc: func(projectDirs []string, threshold time.Duration) (*agent.OrphanDetectionResult, error) {
			return &agent.OrphanDetectionResult{
				Scanned: 1,
				Orphans: []agent.OrphanedAgent{
					{Agent: agent.AgentRef{BeadsID: "fail-1"}, LastPhase: "Complete"},
				},
			}, nil
		},
		ForceCompleteFunc: func(a agent.AgentRef, reason string) (*agent.TransitionEvent, error) {
			return nil, fmt.Errorf("beads unavailable")
		},
	}

	result := RunLifecycleOrphanRecovery(lm, []string{"/tmp"}, time.Hour, false)
	if result.ForceCompleted != 0 {
		t.Errorf("ForceCompleted = %d, want 0 (error)", result.ForceCompleted)
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Skipped)
	}
}

func TestRunLifecycleOrphanRecovery_ForceAbandonError(t *testing.T) {
	lm := &mockLifecycleManager{
		DetectOrphansFunc: func(projectDirs []string, threshold time.Duration) (*agent.OrphanDetectionResult, error) {
			return &agent.OrphanDetectionResult{
				Scanned: 1,
				Orphans: []agent.OrphanedAgent{
					{Agent: agent.AgentRef{BeadsID: "fail-2"}, LastPhase: "Planning"},
				},
			}, nil
		},
		ForceAbandonFunc: func(a agent.AgentRef) (*agent.TransitionEvent, error) {
			return nil, fmt.Errorf("beads unavailable")
		},
	}

	result := RunLifecycleOrphanRecovery(lm, []string{"/tmp"}, time.Hour, false)
	if result.ForceAbandoned != 0 {
		t.Errorf("ForceAbandoned = %d, want 0 (error)", result.ForceAbandoned)
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Skipped)
	}
}

func TestRunLifecycleOrphanRecovery_Snapshot(t *testing.T) {
	lm := &mockLifecycleManager{
		DetectOrphansFunc: func(projectDirs []string, threshold time.Duration) (*agent.OrphanDetectionResult, error) {
			return &agent.OrphanDetectionResult{
				Scanned: 5,
				Orphans: []agent.OrphanedAgent{
					{Agent: agent.AgentRef{BeadsID: "a1"}, LastPhase: "Complete"},
					{Agent: agent.AgentRef{BeadsID: "a2"}, LastPhase: "Planning"},
				},
			}, nil
		},
	}

	result := RunLifecycleOrphanRecovery(lm, []string{"/tmp"}, time.Hour, false)
	snapshot := result.Snapshot()
	if snapshot.ResetCount != 2 {
		t.Errorf("Snapshot.ResetCount = %d, want 2", snapshot.ResetCount)
	}
}

func TestIsPhaseCompleteStr(t *testing.T) {
	tests := []struct {
		phase string
		want  bool
	}{
		{"Complete", true},
		{"complete", true},
		{"COMPLETE", true},
		{"Planning", false},
		{"Implementing", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isPhaseCompleteStr(tt.phase)
		if got != tt.want {
			t.Errorf("isPhaseCompleteStr(%q) = %v, want %v", tt.phase, got, tt.want)
		}
	}
}
