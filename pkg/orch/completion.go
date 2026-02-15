// Package orch provides orchestration-level utilities for agent management.
// This includes completion backlog detection and related metrics.
package orch

import (
	"strings"
	"time"
)

// AgentInfo represents the minimal agent information needed for completion backlog detection.
// This is populated from serve_agents.go's agent data structures.
type AgentInfo struct {
	BeadsID         string    // Beads issue ID (e.g., "orch-go-e6o")
	SessionID       string    // OpenCode session ID
	Phase           string    // Current phase from beads comments (e.g., "Planning", "Complete")
	PhaseReportedAt time.Time // Timestamp when the phase was reported
	Status          string    // Agent status (e.g., "active", "idle", "dead", "completed")
}

// DetectCompletionBacklog checks for agents that have reported Phase: Complete
// but haven't been closed by orch complete for longer than the threshold duration.
//
// This is used to detect completion backlog - agents that are done but waiting for
// orchestrator action. These agents should be surfaced to the orchestrator for review.
//
// The threshold is typically 10 minutes, based on the coaching metrics design:
// "Detect agents at Phase:Complete for >10 minutes without orch complete being run."
//
// Parameters:
//   - agents: slice of AgentInfo structs containing agent phase and timing information
//   - threshold: duration after which a completed agent is considered backlogged
//
// Returns:
//   - slice of beads IDs for agents in completion backlog
//
// Example usage:
//
//	agents := []orch.AgentInfo{
//	    {BeadsID: "orch-go-abc", Phase: "Complete", PhaseReportedAt: time.Now().Add(-15 * time.Minute)},
//	    {BeadsID: "orch-go-xyz", Phase: "Planning", PhaseReportedAt: time.Now().Add(-5 * time.Minute)},
//	}
//	backlog := orch.DetectCompletionBacklog(agents, 10 * time.Minute)
//	// backlog = ["orch-go-abc"]
func DetectCompletionBacklog(agents []AgentInfo, threshold time.Duration) []string {
	now := time.Now()
	var backlog []string
	for _, a := range agents {
		// Skip agents not at Phase: Complete
		if !strings.EqualFold(a.Phase, "Complete") {
			continue
		}
		// Skip agents already closed by orch complete
		if a.Status == "completed" {
			continue
		}
		// Skip agents with zero PhaseReportedAt (no timestamp available)
		if a.PhaseReportedAt.IsZero() {
			continue
		}
		// Check if agent has been in Complete phase longer than threshold
		if now.Sub(a.PhaseReportedAt) > threshold {
			backlog = append(backlog, a.BeadsID)
		}
	}
	return backlog
}
