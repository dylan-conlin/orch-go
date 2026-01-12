// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"github.com/dylan-conlin/orch-go/pkg/cleanup"
)

// runSessionCleanup is a helper that wraps the cleanup package's CleanStaleSessions function.
// This allows the daemon to call cleanup without circular imports.
func runSessionCleanup(serverURL string, ageDays int, preserveOrchestrator bool) (int, error) {
	return cleanup.CleanStaleSessions(cleanup.CleanStaleSessionsOptions{
		ServerURL:            serverURL,
		StaleDays:            ageDays,
		DryRun:               false,
		PreserveOrchestrator: preserveOrchestrator,
		Quiet:                true, // Daemon runs in background - suppress output
	})
}
