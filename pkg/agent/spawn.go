package agent

import (
	"fmt"
	"time"
)

// SpawnInput provides parameters for the spawn lifecycle transition.
// This captures the lifecycle-relevant subset of spawn configuration,
// decoupled from spawn-specific content generation (spawn.Config).
type SpawnInput struct {
	// BeadsID is the beads issue ID. Required when NoTrack is false.
	BeadsID string

	// WorkspaceName is the canonical agent identifier (directory name).
	WorkspaceName string

	// WorkspacePath is the full path to the workspace directory.
	WorkspacePath string

	// ProjectDir is the absolute path to the project directory.
	ProjectDir string

	// SpawnMode is the spawn backend: "opencode" or "claude".
	SpawnMode string

	// NoTrack disables beads integration when true.
	NoTrack bool
}

// Validate checks that all required fields are present.
func (si *SpawnInput) Validate() error {
	if si.WorkspaceName == "" {
		return fmt.Errorf("WorkspaceName is required")
	}
	if si.WorkspacePath == "" {
		return fmt.Errorf("WorkspacePath is required")
	}
	if si.ProjectDir == "" {
		return fmt.Errorf("ProjectDir is required")
	}
	if si.SpawnMode == "" {
		return fmt.Errorf("SpawnMode is required")
	}
	if !si.NoTrack && si.BeadsID == "" {
		return fmt.Errorf("BeadsID is required when tracking is enabled")
	}
	return nil
}

// ToAgentRef creates an AgentRef from the spawn input.
// The returned ref has no SessionID — that is set during ActivateSpawn.
func (si *SpawnInput) ToAgentRef() AgentRef {
	return AgentRef{
		BeadsID:       si.BeadsID,
		WorkspaceName: si.WorkspaceName,
		WorkspacePath: si.WorkspacePath,
		ProjectDir:    si.ProjectDir,
		SpawnMode:     si.SpawnMode,
	}
}

// SpawnHandle represents an in-progress spawn between Phase 1 (BeginSpawn)
// and Phase 2 (ActivateSpawn). The caller creates the session/window between
// phases and then calls ActivateSpawn (or Rollback on failure).
//
// The handle accumulates EffectResults from both phases into a single
// TransitionEvent for the complete Spawning → Active transition.
type SpawnHandle struct {
	// Agent is the reference to the spawning agent.
	Agent AgentRef

	// Rollback undoes Phase 1 side effects (beads untag, workspace removal).
	// Safe to call multiple times. Nil-safe via SafeRollback().
	Rollback func()

	// event accumulates effects across both spawn phases.
	event *TransitionEvent
}

// NewSpawnHandle creates a SpawnHandle with an initialized TransitionEvent.
func NewSpawnHandle(agent AgentRef, rollback func()) *SpawnHandle {
	return &SpawnHandle{
		Agent:    agent,
		Rollback: rollback,
		event: &TransitionEvent{
			Transition: TransitionSpawn,
			Agent:      agent,
			FromState:  StateSpawning,
			// ToState and Timestamp set in Finalize
		},
	}
}

// Event returns the in-progress TransitionEvent for adding effects.
func (h *SpawnHandle) Event() *TransitionEvent {
	return h.event
}

// SafeRollback calls Rollback if it is non-nil.
func (h *SpawnHandle) SafeRollback() {
	if h.Rollback != nil {
		h.Rollback()
	}
}

// Finalize completes the TransitionEvent with the session ID and timestamp.
// Sets Success based on whether any critical effects failed.
// Returns the finalized event (Spawning → Active).
func (h *SpawnHandle) Finalize(sessionID string) *TransitionEvent {
	h.event.Agent.SessionID = sessionID
	h.event.ToState = StateActive
	h.event.Timestamp = time.Now()
	h.event.Success = !h.event.HasCriticalFailure()
	return h.event
}
