package agent

import (
	"fmt"
	"time"
)

// State represents an agent's lifecycle state.
type State string

const (
	StateSpawning      State = "spawning"       // Transient: during orch spawn execution
	StateActive        State = "active"          // Agent is working
	StatePhaseComplete State = "phase_complete"  // Agent declared done, awaiting orch complete
	StateCompleting    State = "completing"      // Transient: during orch complete execution
	StateCompleted     State = "completed"       // Terminal: beads issue closed
	StateAbandoned     State = "abandoned"       // Terminal: beads reset to open, respawnable
	StateOrphaned      State = "orphaned"        // Detected by GC: in_progress but no live execution
)

// AllStates returns all valid lifecycle states.
func AllStates() []State {
	return []State{
		StateSpawning,
		StateActive,
		StatePhaseComplete,
		StateCompleting,
		StateCompleted,
		StateAbandoned,
		StateOrphaned,
	}
}

// IsTerminal returns true if the state is a terminal state (no further transitions).
func (s State) IsTerminal() bool {
	return s == StateCompleted || s == StateAbandoned
}

// IsTransient returns true if the state is transient (should be short-lived).
func (s State) IsTransient() bool {
	return s == StateSpawning || s == StateCompleting
}

// String returns the string representation of the state.
func (s State) String() string {
	return string(s)
}

// Transition represents a lifecycle state change.
type Transition string

const (
	TransitionSpawn         Transition = "spawn"          // Spawning → Active
	TransitionPhaseComplete Transition = "phase_complete"  // Active → PhaseComplete
	TransitionComplete      Transition = "complete"        // PhaseComplete → Completed (via Completing)
	TransitionAbandon       Transition = "abandon"         // Active → Abandoned
	TransitionForceComplete Transition = "force_complete"  // Orphaned → Completed
	TransitionForceAbandon  Transition = "force_abandon"   // Orphaned → Abandoned
)

// AllTransitions returns all valid lifecycle transitions.
func AllTransitions() []Transition {
	return []Transition{
		TransitionSpawn,
		TransitionPhaseComplete,
		TransitionComplete,
		TransitionAbandon,
		TransitionForceComplete,
		TransitionForceAbandon,
	}
}

// String returns the string representation of the transition.
func (t Transition) String() string {
	return string(t)
}

// validTransitions maps each transition to its valid (from → to) state pair.
var validTransitions = map[Transition]struct {
	From State
	To   State
}{
	TransitionSpawn:         {From: StateSpawning, To: StateActive},
	TransitionPhaseComplete: {From: StateActive, To: StatePhaseComplete},
	TransitionComplete:      {From: StatePhaseComplete, To: StateCompleted},
	TransitionAbandon:       {From: StateActive, To: StateAbandoned},
	TransitionForceComplete: {From: StateOrphaned, To: StateCompleted},
	TransitionForceAbandon:  {From: StateOrphaned, To: StateAbandoned},
}

// ValidateTransition checks whether a transition is valid from the given state.
// Returns the target state if valid, or an error if the transition is not allowed.
func ValidateTransition(from State, t Transition) (State, error) {
	rule, ok := validTransitions[t]
	if !ok {
		return "", fmt.Errorf("unknown transition: %s", t)
	}
	if rule.From != from {
		return "", fmt.Errorf("invalid transition %s from state %s (expected %s)", t, from, rule.From)
	}
	return rule.To, nil
}

// AgentRef identifies an agent for lifecycle operations.
// This is NOT stored state — it's a query handle assembled from authoritative sources.
type AgentRef struct {
	// BeadsID is the beads issue ID for lifecycle tracking.
	BeadsID string

	// WorkspaceName is the workspace directory name (canonical agent identifier).
	WorkspaceName string

	// WorkspacePath is the full path to the workspace directory.
	WorkspacePath string

	// SessionID is the OpenCode session ID. Empty for Claude-mode agents.
	SessionID string

	// ProjectDir is the absolute path to the project directory.
	ProjectDir string

	// SpawnMode is the spawn backend: "opencode" or "claude".
	SpawnMode string
}

// TransitionEvent records a state transition with its side effects.
type TransitionEvent struct {
	// Transition is the type of state change.
	Transition Transition

	// Agent is the agent that transitioned.
	Agent AgentRef

	// FromState is the state before the transition.
	FromState State

	// ToState is the state after the transition.
	ToState State

	// Effects tracks individual side effect results.
	Effects []EffectResult

	// Warnings collects non-fatal issues encountered during the transition.
	Warnings []string

	// Success indicates whether all critical effects succeeded.
	Success bool

	// Timestamp is when the transition occurred.
	Timestamp time.Time

	// Reason explains why the transition was triggered.
	Reason string
}

// HasCriticalFailure returns true if any critical effect failed.
func (te *TransitionEvent) HasCriticalFailure() bool {
	for _, e := range te.Effects {
		if e.Critical && !e.Success {
			return true
		}
	}
	return false
}

// AddEffect appends an effect result and updates warnings if non-critical failure.
func (te *TransitionEvent) AddEffect(e EffectResult) {
	te.Effects = append(te.Effects, e)
	if !e.Success && !e.Critical && e.Error != nil {
		te.Warnings = append(te.Warnings, fmt.Sprintf("%s/%s: %v", e.Subsystem, e.Operation, e.Error))
	}
}

// EffectResult tracks the outcome of a single side effect within a transition.
type EffectResult struct {
	// Subsystem identifies which layer this effect targets.
	// One of: "beads", "opencode", "tmux", "workspace", "events"
	Subsystem string

	// Operation describes the specific action taken.
	// Examples: "close_issue", "remove_label", "archive_workspace", "kill_window"
	Operation string

	// Success indicates whether the effect completed without error.
	Success bool

	// Error captures the error if the effect failed.
	Error error

	// Critical indicates whether failure of this effect should fail the entire transition.
	Critical bool

	// Duration records how long the effect took.
	Duration time.Duration
}

// OrphanDetectionResult from a periodic GC scan.
type OrphanDetectionResult struct {
	// Orphans is the list of detected orphaned agents.
	Orphans []OrphanedAgent

	// Scanned is the total number of agents examined.
	Scanned int

	// Elapsed is how long the scan took.
	Elapsed time.Duration
}

// TrackedIssue represents a beads issue returned by ListByLabel.
// Used by DetectOrphans to find agents tagged with orch:agent.
type TrackedIssue struct {
	// BeadsID is the issue identifier.
	BeadsID string

	// Status is the issue status (e.g., "open", "in_progress", "closed").
	Status string

	// Labels are the issue's labels (e.g., ["orch:agent", "triage:ready"]).
	Labels []string
}

// WorkspaceInfo represents metadata about an agent workspace.
// Used by DetectOrphans to join workspace data with beads issues.
type WorkspaceInfo struct {
	// Name is the workspace directory name.
	Name string

	// Path is the full path to the workspace directory.
	Path string

	// BeadsID is the beads issue ID from the manifest.
	BeadsID string

	// SessionID is the OpenCode session ID (empty for Claude-mode agents).
	SessionID string

	// SpawnMode is the spawn backend: "opencode" or "claude".
	SpawnMode string

	// SpawnTime is when the agent was spawned.
	SpawnTime time.Time
}

// OrphanedAgent represents an agent detected as orphaned by GC.
type OrphanedAgent struct {
	// Agent is the reference to the orphaned agent.
	Agent AgentRef

	// Reason explains why the agent was classified as orphaned.
	// Examples: "no_session_no_phase", "session_dead_no_phase", "no_activity_2h"
	Reason string

	// LastPhase is the last known phase from beads comments (may be empty).
	LastPhase string

	// StaleFor is how long since the agent's last detected activity.
	StaleFor time.Duration

	// ShouldRetry indicates whether the issue should be respawned (based on triage:ready label).
	ShouldRetry bool
}
