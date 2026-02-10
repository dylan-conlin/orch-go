// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/cleanup"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

const defaultUntrackedSessionIdleThreshold = 30 * time.Minute

type sessionReaperClient interface {
	ListSessions(directory string) ([]opencode.Session, error)
	IsSessionProcessing(sessionID string) bool
	DeleteSession(sessionID string) error
}

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

func runUntrackedSessionCleanup(serverURL string, preserveOrchestrator bool) (int, error) {
	client := opencode.NewClient(serverURL)
	return reapIdleUntrackedSessions(client, defaultUntrackedSessionIdleThreshold, preserveOrchestrator, time.Now())
}

func reapIdleUntrackedSessions(client sessionReaperClient, maxIdle time.Duration, preserveOrchestrator bool, now time.Time) (int, error) {
	if client == nil {
		return 0, nil
	}
	if maxIdle <= 0 {
		maxIdle = defaultUntrackedSessionIdleThreshold
	}

	sessions, err := client.ListSessions("")
	if err != nil {
		return 0, err
	}

	deleted := 0
	for _, session := range sessions {
		beadsID := extractBeadsIDFromSessionTitle(session.Title)
		if beadsID != "" && !isUntrackedBeadsID(beadsID) {
			continue
		}

		if preserveOrchestrator && cleanup.IsOrchestratorSessionTitle(session.Title) {
			continue
		}

		updatedAt := time.Unix(session.Time.Updated/1000, 0)
		if now.Sub(updatedAt) < maxIdle {
			continue
		}

		if client.IsSessionProcessing(session.ID) {
			continue
		}

		if err := client.DeleteSession(session.ID); err != nil {
			if strings.Contains(err.Error(), "status 404") {
				continue
			}
			return deleted, err
		}

		deleted++
	}

	return deleted, nil
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
