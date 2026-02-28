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
	if !d.Config.OrphanDetectionEnabled || d.Config.OrphanDetectionInterval <= 0 {
		return false
	}
	if d.lastOrphanDetection.IsZero() {
		return true
	}
	return time.Since(d.lastOrphanDetection) >= d.Config.OrphanDetectionInterval
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
	getAgents := d.getActiveAgentsFunc
	if getAgents == nil {
		getAgents = GetActiveAgents
	}

	agents, err := getAgents()
	if err != nil {
		return &OrphanDetectionResult{
			Error:   err,
			Message: fmt.Sprintf("Orphan detection failed to list agents: %v", err),
		}
	}

	hasSession := d.hasExistingSessionFunc
	if hasSession == nil {
		hasSession = HasExistingSessionForBeadsID
	}

	updateStatus := d.updateBeadsStatusForOrphanFunc
	if updateStatus == nil {
		updateStatus = UpdateBeadsStatus
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
		if hasSession(agent.BeadsID) {
			// Agent exists - not an orphan (recovery handles idle agents)
			skipped++
			continue
		}

		// No session and no tmux window - this is an orphan.
		// Reset to open so daemon can respawn it.
		if d.Config.Verbose {
			fmt.Printf("  Orphan detected: %s (idle %v, no session/window)\n",
				agent.BeadsID, idleTime.Round(time.Minute))
		}

		if err := updateStatus(agent.BeadsID, "open"); err != nil {
			if d.Config.Verbose {
				fmt.Printf("  Failed to reset orphan %s: %v\n", agent.BeadsID, err)
			}
			skipped++
			continue
		}

		// Unmark from SpawnedIssues tracker so it can be respawned
		if d.SpawnedIssues != nil {
			d.SpawnedIssues.Unmark(agent.BeadsID)
		}

		orphans = append(orphans, OrphanedIssue{
			BeadsID:  agent.BeadsID,
			Title:    agent.Title,
			IdleTime: idleTime,
		})
		reset++
	}

	d.lastOrphanDetection = time.Now()

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
	return d.lastOrphanDetection
}

// NextOrphanDetectionTime returns when the next orphan detection is scheduled.
// Returns zero time if orphan detection is disabled.
func (d *Daemon) NextOrphanDetectionTime() time.Time {
	if !d.Config.OrphanDetectionEnabled || d.Config.OrphanDetectionInterval <= 0 {
		return time.Time{}
	}
	if d.lastOrphanDetection.IsZero() {
		return time.Now()
	}
	return d.lastOrphanDetection.Add(d.Config.OrphanDetectionInterval)
}
