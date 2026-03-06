// Package verify provides verification helpers for agent completion.
// This file implements composed liveness verification for destructive commands
// (complete, abandon). Uses phase-based liveness per the decision:
// .kb/decisions/2026-02-26-phase-based-liveness-over-tmux-as-state.md
package verify

import (
	"fmt"
	"strings"
	"time"
)

// Liveness status constants.
const (
	LivenessActive    = "active"
	LivenessCompleted = "completed"
	LivenessDead      = "dead"
)

// Liveness reason codes matching the decision document.
const (
	ReasonPhaseReported   = "phase_reported"
	ReasonPhaseComplete   = "phase_complete"
	ReasonRecentlySpawned = "recently_spawned"
	ReasonNoPhaseReported = "no_phase_reported"
)

// livenessGracePeriod is the window after spawn where agents without phase
// comments are still considered active (allowing time for first tool calls).
const livenessGracePeriod = 5 * time.Minute

// LivenessInput contains the data needed to determine agent liveness.
// All fields are pre-fetched — this function does no I/O.
type LivenessInput struct {
	// Comments from the beads issue (pre-fetched via GetComments).
	Comments []Comment

	// SpawnTime is when the agent was created. Zero value means unknown.
	SpawnTime time.Time

	// Now is the current time. Injected for testability.
	Now time.Time
}

// LivenessResult contains the outcome of a liveness check.
type LivenessResult struct {
	// Status is one of: "active", "completed", "dead".
	Status string

	// Reason is the reason code for the status determination.
	Reason string

	// Phase is the parsed phase status from comments (if any).
	Phase PhaseStatus
}

// IsAlive returns true if the agent appears to be actively running.
// Use this as the guard before destructive operations.
func (r *LivenessResult) IsAlive() bool {
	return r.Status == LivenessActive
}

// Warning returns a human-readable warning message if the agent is alive.
// Returns empty string if the agent is not alive (safe to proceed).
func (r *LivenessResult) Warning() string {
	if !r.IsAlive() {
		return ""
	}

	switch r.Reason {
	case ReasonPhaseReported:
		msg := fmt.Sprintf("agent appears still running (last phase: %s", r.Phase.Phase)
		if r.Phase.PhaseReportedAt != nil {
			elapsed := r.Phase.PhaseReportedAt.Sub(time.Time{})
			if !r.Phase.PhaseReportedAt.IsZero() {
				elapsed = time.Since(*r.Phase.PhaseReportedAt)
				msg += fmt.Sprintf(", %s ago", formatElapsed(elapsed))
			}
		}
		if r.Phase.Summary != "" {
			msg += fmt.Sprintf(": %s", r.Phase.Summary)
		}
		msg += ")"
		return msg
	case ReasonRecentlySpawned:
		return "agent was recently spawned and may not have reported its first phase yet"
	default:
		return ""
	}
}

// VerifyLiveness determines agent liveness using phase-based logic.
// This is a pure function — no I/O, no side effects. All data is pre-fetched
// and passed via LivenessInput.
//
// The four states follow the decision document:
//
//	Phase comment exists (not "Complete") → active (phase_reported)
//	Phase: Complete                       → completed (phase_complete)
//	Recently spawned (<5 min), no phase   → active (recently_spawned)
//	No phase comment, >5 min since spawn  → dead (no_phase_reported)
func VerifyLiveness(input LivenessInput) LivenessResult {
	phase := ParsePhaseFromComments(input.Comments)

	if phase.Found {
		if strings.EqualFold(phase.Phase, "Complete") {
			return LivenessResult{
				Status: LivenessCompleted,
				Reason: ReasonPhaseComplete,
				Phase:  phase,
			}
		}
		return LivenessResult{
			Status: LivenessActive,
			Reason: ReasonPhaseReported,
			Phase:  phase,
		}
	}

	// No phase found — check grace period
	if !input.SpawnTime.IsZero() && input.Now.Sub(input.SpawnTime) < livenessGracePeriod {
		return LivenessResult{
			Status: LivenessActive,
			Reason: ReasonRecentlySpawned,
			Phase:  phase,
		}
	}

	return LivenessResult{
		Status: LivenessDead,
		Reason: ReasonNoPhaseReported,
		Phase:  phase,
	}
}

// formatElapsed formats a duration into a human-readable string.
func formatElapsed(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
