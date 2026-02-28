package agent

import (
	"time"
)

// LifecycleManager coordinates state transitions across all four authoritative layers
// (beads, workspace, OpenCode, tmux). It does NOT store agent state — it reads from
// and writes to authoritative sources. After any method returns, the manager holds
// no agent state.
//
// Architectural constraint: This is a coordinator, not a cache.
// See: .kb/models/agent-lifecycle-state-model/model.md (Invariant #7)
type LifecycleManager interface {
	// BeginSpawn performs Phase 1 of the spawn transition (Spawning state).
	// Tags beads issue with orch:agent label (if tracked).
	// Returns a SpawnHandle with rollback capability. The caller is responsible for:
	//   1. Workspace content generation (via pkg/spawn)
	//   2. Session/window creation (via backend)
	//   3. Calling ActivateSpawn on success, or handle.Rollback() on failure
	//
	// The handle accumulates EffectResults so the full Spawning → Active
	// transition is captured in a single TransitionEvent.
	BeginSpawn(input SpawnInput) (*SpawnHandle, error)

	// ActivateSpawn performs Phase 2 of the spawn transition (Spawning → Active).
	// Records session ID in workspace metadata and finalizes the TransitionEvent.
	// The sessionID may be empty for Claude-mode agents (tmux window ID is used instead).
	ActivateSpawn(handle *SpawnHandle, sessionID string) (*TransitionEvent, error)

	// Complete performs all side effects for the Complete transition.
	// Precondition: verification gates have already passed (caller's responsibility).
	// The lifecycle manager owns cleanup, not verification.
	Complete(agent AgentRef, reason string) (*TransitionEvent, error)

	// Abandon performs all side effects for the Abandon transition.
	// Removes orch:agent label and clears assignee (fixes ghost agent bug).
	Abandon(agent AgentRef, reason string) (*TransitionEvent, error)

	// ForceComplete performs GC-initiated completion for orphaned agents.
	ForceComplete(agent AgentRef, reason string) (*TransitionEvent, error)

	// ForceAbandon performs GC-initiated abandonment for orphaned agents
	// that should be retried via respawn.
	ForceAbandon(agent AgentRef) (*TransitionEvent, error)

	// DetectOrphans scans for agents in Active state with no live execution.
	// threshold determines how long an agent must be inactive before being considered orphaned.
	DetectOrphans(projectDirs []string, threshold time.Duration) (*OrphanDetectionResult, error)

	// CurrentState determines an agent's current lifecycle state by querying
	// authoritative sources (beads, workspace, OpenCode, tmux).
	CurrentState(agent AgentRef) (State, error)
}

// BeadsClient abstracts beads issue operations for lifecycle transitions.
type BeadsClient interface {
	AddLabel(beadsID, label string) error
	RemoveLabel(beadsID, label string) error
	UpdateStatus(beadsID, status string) error
	SetAssignee(beadsID, assignee string) error
	ClearAssignee(beadsID string) error
	CloseIssue(beadsID, reason string) error
	GetComments(beadsID string) ([]string, error)
}

// OpenCodeClient abstracts OpenCode session operations for lifecycle transitions.
type OpenCodeClient interface {
	SessionExists(sessionID string) (bool, error)
	DeleteSession(sessionID string) error
	ExportActivity(sessionID, outputPath string) error
}

// TmuxClient abstracts tmux window operations for lifecycle transitions.
type TmuxClient interface {
	WindowExists(name string) (bool, error)
	KillWindow(name string) error
}

// EventLogger abstracts event logging for lifecycle transitions.
type EventLogger interface {
	Log(eventType string, data map[string]interface{}) error
}

// WorkspaceManager abstracts workspace file operations for lifecycle transitions.
type WorkspaceManager interface {
	Archive(workspacePath string) error
	WriteFailureReport(workspacePath, reason string) error
	Exists(workspacePath string) bool

	// WriteSessionID writes the session ID to the workspace dotfile.
	// Used during ActivateSpawn (Phase 2) after session creation.
	WriteSessionID(workspacePath, sessionID string) error

	// Remove deletes the workspace directory.
	// Used during spawn rollback when session creation fails.
	Remove(workspacePath string) error
}
