// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"os"

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

// runWorkspaceCleanup is a helper that wraps the cleanup package's ArchiveStaleWorkspaces function.
// This allows the daemon to call cleanup without circular imports.
func runWorkspaceCleanup(projectDir string, ageDays int, preserveOrchestrator bool) (int, error) {
	return cleanup.ArchiveStaleWorkspaces(cleanup.ArchiveStaleWorkspacesOptions{
		ProjectDir:           projectDir,
		StaleDays:            ageDays,
		DryRun:               false,
		PreserveOrchestrator: preserveOrchestrator,
		Quiet:                true, // Daemon runs in background - suppress output
	})
}

// runInvestigationCleanup is a helper that wraps the cleanup package's ArchiveEmptyInvestigations function.
// This allows the daemon to call cleanup without circular imports.
func runInvestigationCleanup(projectDir string) (int, error) {
	return cleanup.ArchiveEmptyInvestigations(cleanup.ArchiveEmptyInvestigationsOptions{
		ProjectDir: projectDir,
		DryRun:     false,
		Quiet:      true, // Daemon runs in background - suppress output
	})
}

// getProjectDir returns the current project directory.
// Falls back to current working directory if not in a project.
func getProjectDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return cwd
}
