// Package daemon provides autonomous overnight processing capabilities.
// This file contains phase timeout detection: finds agents with an active session
// but no phase comment update within the configured threshold, and flags them
// as "unresponsive" for surfacing in orch status.
package daemon

import (
	"fmt"
	"strings"
	"time"
)

// PhaseTimeoutResult contains the result of a phase timeout detection operation.
type PhaseTimeoutResult struct {
	// UnresponsiveCount is the number of agents flagged as unresponsive.
	UnresponsiveCount int

	// SkippedCount is the number of agents skipped (recent phase, completed, no session).
	SkippedCount int

	// Agents lists the agents detected as unresponsive.
	Agents []UnresponsiveAgent

	// Error is set if the detection failed.
	Error error

	// Message is a human-readable summary.
	Message string
}

// UnresponsiveAgent represents an agent that has an active session but hasn't
// reported a phase update within the configured threshold.
type UnresponsiveAgent struct {
	BeadsID      string
	Title        string
	Phase        string
	IdleDuration time.Duration
}

// PhaseTimeoutSnapshot is a point-in-time snapshot for the daemon status file.
type PhaseTimeoutSnapshot struct {
	UnresponsiveCount int       `json:"unresponsive_count"`
	LastCheck         time.Time `json:"last_check"`
}

// Snapshot converts a PhaseTimeoutResult to a dashboard-ready snapshot.
func (r *PhaseTimeoutResult) Snapshot() PhaseTimeoutSnapshot {
	return PhaseTimeoutSnapshot{
		UnresponsiveCount: r.UnresponsiveCount,
		LastCheck:         time.Now(),
	}
}

// ShouldRunPhaseTimeout returns true if periodic phase timeout detection should run.
func (d *Daemon) ShouldRunPhaseTimeout() bool {
	return d.Scheduler.IsDue(TaskPhaseTimeout)
}

// RunPeriodicPhaseTimeout runs phase timeout detection if due.
// Returns the result if detection was run, or nil if it wasn't due.
//
// An unresponsive agent is one that:
// 1. Is in_progress (not completed)
// 2. Has an active session (OpenCode or tmux)
// 3. Hasn't reported a phase update in PhaseTimeoutThreshold duration
//
// Unlike orphan detection (which resets issues to open), phase timeout is
// advisory-only: it flags agents for visibility in orch status without
// taking corrective action. Recovery (resume prompts) handles the corrective path.
func (d *Daemon) RunPeriodicPhaseTimeout() *PhaseTimeoutResult {
	if !d.ShouldRunPhaseTimeout() {
		return nil
	}

	agentDiscoverer := d.Agents
	if agentDiscoverer == nil {
		agentDiscoverer = &defaultAgentDiscoverer{}
	}

	agents, err := agentDiscoverer.GetActiveAgents()
	if err != nil {
		return &PhaseTimeoutResult{
			Error:   err,
			Message: fmt.Sprintf("Phase timeout detection failed to list agents: %v", err),
		}
	}

	unresponsiveCount := 0
	skipped := 0
	var unresponsiveAgents []UnresponsiveAgent
	now := time.Now()

	for _, agent := range agents {
		if agent.BeadsID == "" {
			skipped++
			continue
		}

		// Skip agents that reported Phase: Complete (waiting for review)
		if strings.EqualFold(agent.Phase, "complete") {
			skipped++
			continue
		}

		// Check age threshold
		idleTime := now.Sub(agent.UpdatedAt)
		if idleTime < d.Config.PhaseTimeoutThreshold {
			skipped++
			continue
		}

		// Only flag agents that HAVE a session — agents without sessions
		// are handled by orphan detection (which resets them to open).
		if !agentDiscoverer.HasExistingSession(agent.BeadsID) {
			skipped++
			continue
		}

		// Agent has a session but hasn't reported a phase update — unresponsive.
		if d.Config.Verbose {
			fmt.Printf("  Unresponsive: %s (phase: %s, idle %v)\n",
				agent.BeadsID, agent.Phase, idleTime.Round(time.Minute))
		}

		unresponsiveAgents = append(unresponsiveAgents, UnresponsiveAgent{
			BeadsID:      agent.BeadsID,
			Title:        agent.Title,
			Phase:        agent.Phase,
			IdleDuration: idleTime,
		})
		unresponsiveCount++
	}

	d.Scheduler.MarkRun(TaskPhaseTimeout)

	return &PhaseTimeoutResult{
		UnresponsiveCount: unresponsiveCount,
		SkippedCount:      skipped,
		Agents:            unresponsiveAgents,
		Message:           fmt.Sprintf("Phase timeout: %d unresponsive, %d ok", unresponsiveCount, skipped),
	}
}

// LastPhaseTimeoutTime returns when phase timeout detection was last run.
// Returns zero time if it has never run.
func (d *Daemon) LastPhaseTimeoutTime() time.Time {
	return d.Scheduler.LastRunTime(TaskPhaseTimeout)
}

// NextPhaseTimeoutTime returns when the next phase timeout detection is scheduled.
// Returns zero time if disabled.
func (d *Daemon) NextPhaseTimeoutTime() time.Time {
	return d.Scheduler.NextRunTime(TaskPhaseTimeout)
}
