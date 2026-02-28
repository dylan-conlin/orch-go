package agent

import (
	"fmt"
	"time"
)

// lifecycleManager is the concrete implementation of LifecycleManager.
// It coordinates side effects across authoritative layers but holds no state.
type lifecycleManager struct {
	beads     BeadsClient
	opencode  OpenCodeClient
	tmux      TmuxClient
	events    EventLogger
	workspace WorkspaceManager
}

// NewLifecycleManager creates a new lifecycle manager with the given clients.
func NewLifecycleManager(
	beads BeadsClient,
	opencode OpenCodeClient,
	tmux TmuxClient,
	events EventLogger,
	workspace WorkspaceManager,
) LifecycleManager {
	return &lifecycleManager{
		beads:     beads,
		opencode:  opencode,
		tmux:      tmux,
		events:    events,
		workspace: workspace,
	}
}

// BeginSpawn performs Phase 1 of the spawn transition.
// Tags the beads issue with orch:agent (if tracked) and returns a SpawnHandle
// with rollback capability. The caller must:
//   1. Generate workspace content (via pkg/spawn.WriteContext)
//   2. Create session/window (via backend)
//   3. Call ActivateSpawn on success, or handle.Rollback() on failure
//
// Effect ordering:
//  1. [critical if tracked] beads: add orch:agent label
func (m *lifecycleManager) BeginSpawn(input SpawnInput) (*SpawnHandle, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	agent := input.ToAgentRef()
	var cleanups []func()

	handle := NewSpawnHandle(agent, func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	})

	// 1. Tag beads issue with orch:agent (skip if NoTrack)
	if !input.NoTrack && input.BeadsID != "" {
		m.runEffect(handle.Event(), "beads", "add_label", true, func() error {
			return m.beads.AddLabel(input.BeadsID, "orch:agent")
		})
		cleanups = append(cleanups, func() {
			_ = m.beads.RemoveLabel(input.BeadsID, "orch:agent")
		})

		if handle.Event().HasCriticalFailure() {
			handle.SafeRollback()
			return nil, handle.Event().Effects[len(handle.Event().Effects)-1].Error
		}
	}

	return handle, nil
}

// ActivateSpawn performs Phase 2 of the spawn transition (Spawning → Active).
// Records the session ID in workspace metadata and logs the spawn event.
//
// Effect ordering:
//  1. [non-critical] workspace: write session ID
//  2. [non-critical] events: log session.spawned
func (m *lifecycleManager) ActivateSpawn(handle *SpawnHandle, sessionID string) (*TransitionEvent, error) {
	if handle == nil {
		return nil, fmt.Errorf("nil SpawnHandle")
	}

	// 1. Write session ID to workspace (non-critical — session already exists)
	if sessionID != "" && handle.Agent.WorkspacePath != "" {
		m.runEffect(handle.Event(), "workspace", "write_session_id", false, func() error {
			return m.workspace.WriteSessionID(handle.Agent.WorkspacePath, sessionID)
		})
	}

	// 2. Log spawn event
	m.runEffect(handle.Event(), "events", "log_spawned", false, func() error {
		return m.events.Log("session.spawned", map[string]interface{}{
			"beads_id":   handle.Agent.BeadsID,
			"workspace":  handle.Agent.WorkspaceName,
			"session_id": sessionID,
			"spawn_mode": handle.Agent.SpawnMode,
		})
	})

	return handle.Finalize(sessionID), nil
}

// Abandon performs all side effects for the Abandon transition.
// Critical effects (beads label removal, assignee clear, status reset) determine Success.
// Non-critical effects (tmux kill, session delete, event log) produce warnings on failure.
//
// Effect ordering:
//  1. [critical]     beads: remove orch:agent label (fixes ghost agent bug)
//  2. [critical]     beads: clear assignee
//  3. [critical]     beads: reset status to open (enables respawn)
//  4. [non-critical] tmux: kill window (if exists)
//  5. [non-critical] opencode: delete session (if exists)
//  6. [non-critical] workspace: write failure report (if reason provided)
//  7. [non-critical] events: log agent.abandoned
func (m *lifecycleManager) Abandon(agent AgentRef, reason string) (*TransitionEvent, error) {
	event := &TransitionEvent{
		Transition: TransitionAbandon,
		Agent:      agent,
		FromState:  StateActive,
		ToState:    StateAbandoned,
		Timestamp:  time.Now(),
		Reason:     reason,
	}

	// --- Critical effects (beads state) ---

	// 1. Remove orch:agent label — THE fix for ghost agent bug.
	// Without this, abandoned agents still appear in `bd list -l orch:agent`.
	m.runEffect(event, "beads", "remove_label", true, func() error {
		return m.beads.RemoveLabel(agent.BeadsID, "orch:agent")
	})

	// 2. Clear assignee so the issue doesn't appear "owned" by dead workspace.
	m.runEffect(event, "beads", "clear_assignee", true, func() error {
		return m.beads.ClearAssignee(agent.BeadsID)
	})

	// 3. Reset status to open for respawn via `orch work`.
	m.runEffect(event, "beads", "update_status", true, func() error {
		return m.beads.UpdateStatus(agent.BeadsID, "open")
	})

	// --- Non-critical effects (cleanup) ---

	// 4. Kill tmux window if it exists.
	if agent.WorkspaceName != "" {
		m.runEffect(event, "tmux", "kill_window", false, func() error {
			exists, err := m.tmux.WindowExists(agent.WorkspaceName)
			if err != nil {
				return err
			}
			if !exists {
				return nil
			}
			return m.tmux.KillWindow(agent.WorkspaceName)
		})
	}

	// 5. Delete OpenCode session if it exists.
	if agent.SessionID != "" {
		m.runEffect(event, "opencode", "delete_session", false, func() error {
			exists, err := m.opencode.SessionExists(agent.SessionID)
			if err != nil {
				return err
			}
			if !exists {
				return nil
			}
			return m.opencode.DeleteSession(agent.SessionID)
		})
	}

	// 6. Write failure report if reason provided and workspace exists.
	if reason != "" && agent.WorkspacePath != "" && m.workspace.Exists(agent.WorkspacePath) {
		m.runEffect(event, "workspace", "write_failure_report", false, func() error {
			return m.workspace.WriteFailureReport(agent.WorkspacePath, reason)
		})
	}

	// 7. Log abandonment event.
	m.runEffect(event, "events", "log_abandoned", false, func() error {
		return m.events.Log("agent.abandoned", map[string]interface{}{
			"beads_id":  agent.BeadsID,
			"workspace": agent.WorkspaceName,
			"reason":    reason,
		})
	})

	// Set overall success based on critical effects
	event.Success = !event.HasCriticalFailure()

	return event, nil
}

// Complete performs all side effects for the Complete transition.
// Precondition: verification gates have already passed (caller's responsibility).
// The lifecycle manager owns cleanup, not verification.
//
// Effect ordering:
//  1. [critical]     beads: close issue (the primary lifecycle operation)
//  2. [non-critical] beads: remove orch:agent label (prevents ghost agent visibility)
//  3. [non-critical] tmux: kill window (if exists)
//  4. [non-critical] opencode: delete session (if exists)
//  5. [non-critical] workspace: archive (move to archived/)
//  6. [non-critical] events: log agent.completed
func (m *lifecycleManager) Complete(agent AgentRef, reason string) (*TransitionEvent, error) {
	event := &TransitionEvent{
		Transition: TransitionComplete,
		Agent:      agent,
		FromState:  StatePhaseComplete,
		ToState:    StateCompleted,
		Timestamp:  time.Now(),
		Reason:     reason,
	}

	// --- Critical effects (beads state) ---

	// 1. Close the beads issue — THE primary Complete operation.
	if agent.BeadsID != "" {
		m.runEffect(event, "beads", "close_issue", true, func() error {
			return m.beads.CloseIssue(agent.BeadsID, reason)
		})
	}

	// --- Non-critical effects (cleanup) ---

	// 2. Remove orch:agent label so bd list -l orch:agent returns only active agents.
	if agent.BeadsID != "" {
		m.runEffect(event, "beads", "remove_label", false, func() error {
			return m.beads.RemoveLabel(agent.BeadsID, "orch:agent")
		})
	}

	// 3. Kill tmux window if it exists.
	if agent.WorkspaceName != "" {
		m.runEffect(event, "tmux", "kill_window", false, func() error {
			exists, err := m.tmux.WindowExists(agent.WorkspaceName)
			if err != nil {
				return err
			}
			if !exists {
				return nil
			}
			return m.tmux.KillWindow(agent.WorkspaceName)
		})
	}

	// 4. Delete OpenCode session if it exists.
	if agent.SessionID != "" {
		m.runEffect(event, "opencode", "delete_session", false, func() error {
			exists, err := m.opencode.SessionExists(agent.SessionID)
			if err != nil {
				return err
			}
			if !exists {
				return nil
			}
			return m.opencode.DeleteSession(agent.SessionID)
		})
	}

	// 5. Archive workspace if it exists.
	if agent.WorkspacePath != "" {
		m.runEffect(event, "workspace", "archive", false, func() error {
			return m.workspace.Archive(agent.WorkspacePath)
		})
	}

	// 6. Log completion event.
	m.runEffect(event, "events", "log_completed", false, func() error {
		return m.events.Log("agent.completed", map[string]interface{}{
			"beads_id":  agent.BeadsID,
			"workspace": agent.WorkspaceName,
			"reason":    reason,
		})
	})

	// Set overall success based on critical effects
	event.Success = !event.HasCriticalFailure()

	return event, nil
}

// ForceComplete performs GC-initiated completion for orphaned agents.
func (m *lifecycleManager) ForceComplete(agent AgentRef, reason string) (*TransitionEvent, error) {
	// TODO: Implement in orch-go-vp6u
	return nil, nil
}

// ForceAbandon performs GC-initiated abandonment for orphaned agents.
func (m *lifecycleManager) ForceAbandon(agent AgentRef) (*TransitionEvent, error) {
	// TODO: Implement in orch-go-vp6u
	return nil, nil
}

// DetectOrphans scans for agents in Active state with no live execution.
func (m *lifecycleManager) DetectOrphans(projectDirs []string, threshold time.Duration) (*OrphanDetectionResult, error) {
	// TODO: Implement in orch-go-vp6u
	return nil, nil
}

// CurrentState determines an agent's current lifecycle state.
func (m *lifecycleManager) CurrentState(agent AgentRef) (State, error) {
	// TODO: Implement in future issue
	return "", nil
}

// runEffect executes a side effect and records it in the transition event.
func (m *lifecycleManager) runEffect(event *TransitionEvent, subsystem, operation string, critical bool, fn func() error) {
	start := time.Now()
	err := fn()
	event.AddEffect(EffectResult{
		Subsystem: subsystem,
		Operation: operation,
		Success:   err == nil,
		Error:     err,
		Critical:  critical,
		Duration:  time.Since(start),
	})
}
