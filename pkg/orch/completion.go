// Package orch provides orchestration-level utilities for agent management.
// This includes completion backlog detection and related metrics.
package orch

import (
	"time"
)

// AgentInfo represents the minimal agent information needed for completion backlog detection.
// This is populated from serve_agents.go's agent data structures.
type AgentInfo struct {
	BeadsID         string    // Beads issue ID (e.g., "orch-go-e6o")
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
// Example usage (to be implemented in orch-go-k5v):
//
//	agents := []orch.AgentInfo{
//	    {BeadsID: "orch-go-abc", Phase: "Complete", PhaseReportedAt: time.Now().Add(-15 * time.Minute)},
//	    {BeadsID: "orch-go-xyz", Phase: "Planning", PhaseReportedAt: time.Now().Add(-5 * time.Minute)},
//	}
//	backlog := orch.DetectCompletionBacklog(agents, 10 * time.Minute)
//	// backlog = ["orch-go-abc"]
//
// NOTE: This is a structural extraction placeholder. The actual implementation
// will be added in issue orch-go-k5v when the completion_backlog metric detection
// is implemented in serve_agents.go.
func DetectCompletionBacklog(agents []AgentInfo, threshold time.Duration) []string {
	// Placeholder implementation - will be filled in orch-go-k5v
	// Expected logic:
	// 1. Filter agents where Phase == "Complete" (case-insensitive)
	// 2. Check if PhaseReportedAt + threshold < now
	// 3. Exclude agents with Status == "completed" (already closed by orch complete)
	// 4. Return beads IDs of backlogged agents
	return nil
}
