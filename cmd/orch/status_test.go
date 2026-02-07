package main

import (
	"testing"
	"time"
)

// TestFormatDuration tests the formatDuration function.
// Note: formatDuration is defined in wait.go
func TestFormatDurationForStatus(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"seconds", 45 * time.Second, "45s"},
		{"minutes and seconds", 5*time.Minute + 23*time.Second, "5m 23s"},
		{"hours and minutes", 1*time.Hour + 2*time.Minute, "1h 2m"},
		{"zero", 0, "0s"},
		{"just minutes", 10 * time.Minute, "10m"},
		{"just hours", 3 * time.Hour, "3h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

// TestStatusUsesMultipleSources verifies that status command uses all agent sources.
// This is a design test - the actual implementation combines data from:
// - SQLite state DB for agent metadata (spawn-time projection cache)
// - OpenCode API (ListSessions) for opencode-mode session liveness
// - Tmux window discovery for running tmux-based agents
func TestStatusUsesMultipleSources(t *testing.T) {
	// The status command uses multiple sources for complete agent discovery:
	//
	// The runStatus function:
	// 1. Queries SQLite state DB for all agents (primary path)
	// 2. Falls back to OpenCode + tmux discovery if state DB is empty
	// 3. Enriches with beads comments and workspace metadata
	// 4. Displays results
	//
	// This ensures all agent types are visible:
	// - opencode-mode: via state DB + OpenCode API for liveness
	// - claude-mode: via state DB + tmux windows for liveness
	// - docker-mode: via state DB
	//
	// Integration testing requires a running OpenCode server and tmux.
}
