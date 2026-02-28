package agent

import (
	"fmt"
	"testing"
	"time"
)

func TestSpawnInput_ToAgentRef(t *testing.T) {
	input := SpawnInput{
		BeadsID:       "proj-123",
		WorkspaceName: "og-feat-add-auth-27feb-abc1",
		WorkspacePath: "/tmp/proj/.orch/workspace/og-feat-add-auth-27feb-abc1",
		ProjectDir:    "/tmp/proj",
		SpawnMode:     "opencode",
	}

	ref := input.ToAgentRef()

	if ref.BeadsID != "proj-123" {
		t.Errorf("BeadsID: got %q, want %q", ref.BeadsID, "proj-123")
	}
	if ref.WorkspaceName != "og-feat-add-auth-27feb-abc1" {
		t.Errorf("WorkspaceName: got %q, want %q", ref.WorkspaceName, "og-feat-add-auth-27feb-abc1")
	}
	if ref.WorkspacePath != "/tmp/proj/.orch/workspace/og-feat-add-auth-27feb-abc1" {
		t.Errorf("WorkspacePath: got %q, want %q", ref.WorkspacePath, "/tmp/proj/.orch/workspace/og-feat-add-auth-27feb-abc1")
	}
	if ref.ProjectDir != "/tmp/proj" {
		t.Errorf("ProjectDir: got %q, want %q", ref.ProjectDir, "/tmp/proj")
	}
	if ref.SpawnMode != "opencode" {
		t.Errorf("SpawnMode: got %q, want %q", ref.SpawnMode, "opencode")
	}
	if ref.SessionID != "" {
		t.Errorf("SessionID should be empty before activation, got %q", ref.SessionID)
	}
}

func TestSpawnInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   SpawnInput
		wantErr bool
	}{
		{
			name: "valid input",
			input: SpawnInput{
				BeadsID:       "proj-123",
				WorkspaceName: "og-feat-test-27feb-abc1",
				WorkspacePath: "/tmp/.orch/workspace/og-feat-test-27feb-abc1",
				ProjectDir:    "/tmp/proj",
				SpawnMode:     "opencode",
			},
			wantErr: false,
		},
		{
			name: "valid no-track input (no beads ID required)",
			input: SpawnInput{
				WorkspaceName: "og-feat-test-27feb-abc1",
				WorkspacePath: "/tmp/.orch/workspace/og-feat-test-27feb-abc1",
				ProjectDir:    "/tmp/proj",
				SpawnMode:     "opencode",
				NoTrack:       true,
			},
			wantErr: false,
		},
		{
			name: "missing workspace name",
			input: SpawnInput{
				BeadsID:       "proj-123",
				WorkspacePath: "/tmp/.orch/workspace/og-feat-test-27feb-abc1",
				ProjectDir:    "/tmp/proj",
				SpawnMode:     "opencode",
			},
			wantErr: true,
		},
		{
			name: "missing workspace path",
			input: SpawnInput{
				BeadsID:       "proj-123",
				WorkspaceName: "og-feat-test-27feb-abc1",
				ProjectDir:    "/tmp/proj",
				SpawnMode:     "opencode",
			},
			wantErr: true,
		},
		{
			name: "missing project dir",
			input: SpawnInput{
				BeadsID:       "proj-123",
				WorkspaceName: "og-feat-test-27feb-abc1",
				WorkspacePath: "/tmp/.orch/workspace/og-feat-test-27feb-abc1",
				SpawnMode:     "opencode",
			},
			wantErr: true,
		},
		{
			name: "missing spawn mode",
			input: SpawnInput{
				BeadsID:       "proj-123",
				WorkspaceName: "og-feat-test-27feb-abc1",
				WorkspacePath: "/tmp/.orch/workspace/og-feat-test-27feb-abc1",
				ProjectDir:    "/tmp/proj",
			},
			wantErr: true,
		},
		{
			name: "tracked but missing beads ID",
			input: SpawnInput{
				WorkspaceName: "og-feat-test-27feb-abc1",
				WorkspacePath: "/tmp/.orch/workspace/og-feat-test-27feb-abc1",
				ProjectDir:    "/tmp/proj",
				SpawnMode:     "opencode",
				NoTrack:       false,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSpawnHandle_Rollback(t *testing.T) {
	rolledBack := false
	handle := &SpawnHandle{
		Agent: AgentRef{
			BeadsID:       "proj-123",
			WorkspaceName: "og-feat-test-27feb-abc1",
		},
		Rollback: func() { rolledBack = true },
	}

	handle.Rollback()
	if !rolledBack {
		t.Error("Rollback function was not called")
	}
}

func TestSpawnHandle_NilRollback(t *testing.T) {
	handle := &SpawnHandle{
		Agent: AgentRef{
			BeadsID:       "proj-123",
			WorkspaceName: "og-feat-test-27feb-abc1",
		},
	}

	// Should not panic with nil rollback
	handle.SafeRollback()
}

func TestSpawnHandle_Event(t *testing.T) {
	handle := NewSpawnHandle(
		AgentRef{BeadsID: "proj-123", WorkspaceName: "test"},
		func() {},
	)

	if handle.Event() == nil {
		t.Fatal("Event should be initialized")
	}
	if handle.Event().Transition != TransitionSpawn {
		t.Errorf("Transition: got %q, want %q", handle.Event().Transition, TransitionSpawn)
	}
	if handle.Event().FromState != StateSpawning {
		t.Errorf("FromState: got %q, want %q", handle.Event().FromState, StateSpawning)
	}
}

func TestSpawnHandle_AddEffect(t *testing.T) {
	handle := NewSpawnHandle(
		AgentRef{BeadsID: "proj-123"},
		func() {},
	)

	handle.Event().AddEffect(EffectResult{
		Subsystem: "beads",
		Operation: "add_label",
		Success:   true,
		Critical:  true,
	})

	if len(handle.Event().Effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(handle.Event().Effects))
	}
	if handle.Event().Effects[0].Subsystem != "beads" {
		t.Errorf("Subsystem: got %q, want %q", handle.Event().Effects[0].Subsystem, "beads")
	}
}

func TestSpawnHandle_FinalizeSuccess(t *testing.T) {
	handle := NewSpawnHandle(
		AgentRef{BeadsID: "proj-123", WorkspaceName: "test"},
		func() {},
	)

	// Add a successful critical effect
	handle.Event().AddEffect(EffectResult{
		Subsystem: "beads",
		Operation: "add_label",
		Success:   true,
		Critical:  true,
	})

	event := handle.Finalize("session-abc")
	if event.Agent.SessionID != "session-abc" {
		t.Errorf("SessionID: got %q, want %q", event.Agent.SessionID, "session-abc")
	}
	if event.ToState != StateActive {
		t.Errorf("ToState: got %q, want %q", event.ToState, StateActive)
	}
	if !event.Success {
		t.Error("event should be successful when no critical failures")
	}
	if event.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestSpawnHandle_FinalizeWithCriticalFailure(t *testing.T) {
	handle := NewSpawnHandle(
		AgentRef{BeadsID: "proj-123"},
		func() {},
	)

	handle.Event().AddEffect(EffectResult{
		Subsystem: "beads",
		Operation: "add_label",
		Success:   false,
		Critical:  true,
		Error:     fmt.Errorf("beads daemon unreachable"),
	})

	event := handle.Finalize("")
	if event.Success {
		t.Error("event should fail when critical effect failed")
	}
}

func TestValidateTransition_SpawnTransition(t *testing.T) {
	// Verify the spawn transition is valid from Spawning → Active
	target, err := ValidateTransition(StateSpawning, TransitionSpawn)
	if err != nil {
		t.Fatalf("ValidateTransition(Spawning, Spawn) error: %v", err)
	}
	if target != StateActive {
		t.Errorf("target state: got %q, want %q", target, StateActive)
	}

	// Verify spawn from Active is invalid
	_, err = ValidateTransition(StateActive, TransitionSpawn)
	if err == nil {
		t.Error("expected error for spawn from Active state")
	}
}

func TestNewSpawnHandle_TimestampNotSet(t *testing.T) {
	handle := NewSpawnHandle(
		AgentRef{BeadsID: "proj-123"},
		func() {},
	)

	// Timestamp should be zero until Finalize
	if !handle.Event().Timestamp.IsZero() {
		t.Error("Timestamp should be zero before Finalize")
	}
}

func TestSpawnHandle_Finalize_SetsTimestamp(t *testing.T) {
	handle := NewSpawnHandle(
		AgentRef{BeadsID: "proj-123"},
		func() {},
	)

	before := time.Now()
	event := handle.Finalize("session-123")
	after := time.Now()

	if event.Timestamp.Before(before) || event.Timestamp.After(after) {
		t.Errorf("Timestamp %v should be between %v and %v", event.Timestamp, before, after)
	}
}
