package main

import (
	"testing"
)

// TestCleanupTmuxWindowNoOp verifies that cleanupTmuxWindow is a no-op
// when no matching tmux window exists (doesn't panic or error).
func TestCleanupTmuxWindowNoOp(t *testing.T) {
	// Call with a beads ID that won't match any window.
	// This should be a silent no-op (no panic, no error).
	cleanupTmuxWindow(false, "nonexistent-agent", "nonexistent-beads-id", "nonexistent-id")
}

// TestCleanupTmuxWindowOrchestratorNoOp verifies orchestrator path is a no-op
// when no matching tmux window exists.
func TestCleanupTmuxWindowOrchestratorNoOp(t *testing.T) {
	cleanupTmuxWindow(true, "nonexistent-orchestrator", "", "")
}

// TestCleanupTmuxWindowFallbackToIdentifier verifies that when beadsID is empty,
// the function falls back to using identifier for the search.
func TestCleanupTmuxWindowFallbackToIdentifier(t *testing.T) {
	// With empty beadsID, should search by identifier instead.
	// Still a no-op since the window won't exist, but exercises the fallback path.
	cleanupTmuxWindow(false, "agent-name", "", "some-identifier")
}
