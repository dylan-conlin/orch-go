// Package daemon provides autonomous overnight processing capabilities.
// This file contains orphan detection: finds in_progress issues with no active agent
// and resets them to open for respawning.
package daemon

import (
	"fmt"
	"strings"
	"time"
)

// OrphanDetectionResult contains the result of an orphan detection operation.
type OrphanDetectionResult struct {
	// ResetCount is the number of orphaned issues reset to open.
	ResetCount int

	// SkippedCount is the number of in_progress issues skipped (have agent or too new).
	SkippedCount int

	// Orphans lists the beads IDs that were detected as orphaned and reset.
	Orphans []OrphanedIssue

	// Error is set if the detection failed.
	Error error

	// Message is a human-readable summary.
	Message string
}

// OrphanedIssue represents a single orphaned issue that was detected and reset.
type OrphanedIssue struct {
	BeadsID  string
	Title    string
	IdleTime time.Duration
}

// OrphanDetectionSnapshot is a point-in-time snapshot for the daemon status file.
type OrphanDetectionSnapshot struct {
	ResetCount   int       `json:"reset_count"`
	SkippedCount int       `json:"skipped_count"`
	LastCheck    time.Time `json:"last_check"`
}

// Snapshot converts an OrphanDetectionResult to a dashboard-ready snapshot.
func (r *OrphanDetectionResult) Snapshot() OrphanDetectionSnapshot {
	return OrphanDetectionSnapshot{
		ResetCount:   r.ResetCount,
		SkippedCount: r.SkippedCount,
		LastCheck:    time.Now(),
	}
}

// ShouldRunOrphanDetection returns true if periodic orphan detection should run.
func (d *Daemon) ShouldRunOrphanDetection() bool {
	return d.Scheduler.IsDue(TaskOrphanDetection)
}

// RunPeriodicOrphanDetection runs orphan detection if due.
// Returns the result if detection was run, or nil if it wasn't due.
//
// An orphaned issue is one that is in_progress but has no active agent:
// no OpenCode session, no tmux window. If an issue has been in this state
// for longer than OrphanAgeThreshold (default 1h), it is reset to open
// so the daemon can respawn it on the next poll cycle.
func (d *Daemon) RunPeriodicOrphanDetection() *OrphanDetectionResult {
	if !d.ShouldRunOrphanDetection() {
		return nil
	}

	// Get all in_progress agents
	agentDiscoverer := d.Agents
	if agentDiscoverer == nil {
		agentDiscoverer = &defaultAgentDiscoverer{}
	}

	agents, err := agentDiscoverer.GetActiveAgents()
	if err != nil {
		return &OrphanDetectionResult{
			Error:   err,
			Message: fmt.Sprintf("Orphan detection failed to list agents: %v", err),
		}
	}

	statusUpdater := d.StatusUpdater
	if statusUpdater == nil {
		statusUpdater = &defaultIssueUpdater{}
	}

	reset := 0
	skipped := 0
	var orphans []OrphanedIssue
	now := time.Now()

	for _, agent := range agents {
		if agent.BeadsID == "" {
			skipped++
			continue
		}

		// Skip agents that reported Phase: Complete (waiting for orchestrator review)
		if strings.EqualFold(agent.Phase, "complete") {
			skipped++
			continue
		}

		// Check age threshold: only consider issues orphaned for long enough
		idleTime := now.Sub(agent.UpdatedAt)
		if idleTime < d.Config.OrphanAgeThreshold {
			skipped++
			continue
		}

		// THE KEY CHECK: Does this issue have an actual agent working on it?
		// Uses error-aware version to fail-closed: if session checks error out
		// (OpenCode API down, tmux errors), we do NOT assume the agent is dead.
		// This prevents the orphan detector from incorrectly resetting running
		// agents to "open" during infrastructure instability, which was the root
		// cause of duplicate overnight spawns (orch-go-n20j).
		found, sessionErr := agentDiscoverer.HasExistingSessionOrError(agent.BeadsID)
		if sessionErr != nil {
			// Fail-closed: infrastructure error means we can't confirm agent is dead.
			// Skip this issue — retry on next orphan detection cycle.
			if d.Config.Verbose {
				fmt.Printf("  Skipping orphan check for %s: session check error: %v\n",
					agent.BeadsID, sessionErr)
			}
			skipped++
			continue
		}
		if found {
			// Agent exists - not an orphan (recovery handles idle agents)
			skipped++
			continue
		}

		// No session and no tmux window, AND session checks succeeded (no errors).
		// This is a confirmed orphan. Reset to open so daemon can respawn it.
		if d.Config.Verbose {
			fmt.Printf("  Orphan detected: %s (idle %v, no session/window)\n",
				agent.BeadsID, idleTime.Round(time.Minute))
		}

		if err := statusUpdater.UpdateStatus(agent.BeadsID, "open"); err != nil {
			if d.Config.Verbose {
				fmt.Printf("  Failed to reset orphan %s: %v\n", agent.BeadsID, err)
			}
			skipped++
			continue
		}

		// NOTE: We intentionally do NOT call d.SpawnedIssues.Unmark() here.
		// The spawn cache entry provides a natural cooldown (6h TTL) that prevents
		// thrash loops where an agent dies, gets orphan-detected, status resets to
		// "open", and the daemon immediately respawns it. The TTL expiry will
		// eventually allow respawn. This was the root cause of duplicate spawns
		// during overnight runs (orch-go-ahif).

		orphans = append(orphans, OrphanedIssue{
			BeadsID:  agent.BeadsID,
			Title:    agent.Title,
			IdleTime: idleTime,
		})
		reset++
	}

	d.Scheduler.MarkRun(TaskOrphanDetection)

	return &OrphanDetectionResult{
		ResetCount:   reset,
		SkippedCount: skipped,
		Orphans:      orphans,
		Message:      fmt.Sprintf("Orphan detection: %d reset to open, %d skipped", reset, skipped),
	}
}

// LastOrphanDetectionTime returns when orphan detection was last run.
// Returns zero time if orphan detection has never run.
func (d *Daemon) LastOrphanDetectionTime() time.Time {
	return d.Scheduler.LastRunTime(TaskOrphanDetection)
}

// NextOrphanDetectionTime returns when the next orphan detection is scheduled.
// Returns zero time if orphan detection is disabled.
func (d *Daemon) NextOrphanDetectionTime() time.Time {
	return d.Scheduler.NextRunTime(TaskOrphanDetection)
}
