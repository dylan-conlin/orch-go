package agent

import (
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
func (m *lifecycleManager) Complete(agent AgentRef, reason string) (*TransitionEvent, error) {
	// TODO: Implement in orch-go-hbtr
	return nil, nil
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
