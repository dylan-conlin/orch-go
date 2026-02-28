// Package agent provides agent-related utilities for orch-go.
// This includes filtering logic for determining which agents are "active"
// for different purposes (concurrency limits vs. status display).
package agent

import (
	"strings"
	"time"
)

// IsActiveForConcurrency determines if an agent should count toward the concurrency limit.
// Only running agents count — idle agents are not consuming resources and should not
// block new spawns.
//
// Parameters:
//   - status: agent status (e.g., "running", "idle", "dead")
//   - lastActivity: timestamp of the agent's last activity (unused, kept for API compat)
//   - phase: current phase from beads comments (e.g., "Planning", "Complete")
//
// Returns true if the agent should count toward the concurrency limit.
//
// Rules:
//  1. Phase: Complete agents never count (they're done, just need cleanup)
//  2. Only running agents count
//  3. Idle agents never count (not consuming resources)
func IsActiveForConcurrency(status string, lastActivity time.Time, phase string) bool {
	// Phase: Complete agents don't count (they're done, just need orchestrator action)
	if strings.EqualFold(phase, "Complete") {
		return false
	}

	// Only running agents count toward the concurrency limit.
	// Idle agents are not consuming resources and should not block new spawns.
	return status == "running"
}

// IsVisibleByDefault determines if an agent should be shown in the default status view.
// Uses a conservative 4-hour threshold with state-aware filtering.
//
// This balances visibility (don't hide agents needing action) with signal-to-noise
// (hide true ghosts that have been idle for hours).
//
// Parameters:
//   - status: agent status (e.g., "running", "idle", "dead")
//   - lastActivity: timestamp of the agent's last activity (zero time if unknown)
//   - phase: current phase from beads comments (e.g., "Planning", "Complete")
//
// Returns true if the agent should be visible by default (without --all flag).
//
// Rules:
//  1. Running agents always visible
//  2. Phase: Complete agents always visible (need review/cleanup)
//  3. Agents with unknown activity time (zero time) always visible (conservative)
//  4. Agents active within last 4 hours are visible
//  5. Older idle agents are hidden (shown with --all flag)
func IsVisibleByDefault(status string, lastActivity time.Time, phase string) bool {
	now := time.Now()
	fourHours := 4 * time.Hour

	// Always show running agents
	if status == "running" {
		return true
	}

	// Always show Phase: Complete agents (need review/cleanup)
	if strings.EqualFold(phase, "Complete") {
		return true
	}

	// If lastActivity is zero (unknown), show by default (conservative)
	// This handles agents where we can't determine last activity (e.g., tmux-only)
	if lastActivity.IsZero() {
		return true
	}

	// Show agents active within last 4 hours
	if now.Sub(lastActivity) < fourHours {
		return true
	}

	return false
}
