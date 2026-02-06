// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/cleanup"
	"github.com/dylan-conlin/orch-go/pkg/registry"
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

// runRegistryCleanup removes stale registry entries older than the given age.
// Uses Registry.Purge + Save to ensure proper file locking and merge semantics,
// preventing race conditions with concurrent agent registrations.
// Returns the number of entries removed and any error encountered.
func runRegistryCleanup(ageDays int) (int, error) {
	reg, err := registry.New("")
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().AddDate(0, 0, -ageDays)

	removed := reg.Purge(func(agent *registry.Agent) bool {
		spawnTime, err := time.Parse(registry.TimeFormat, agent.SpawnedAt)
		if err != nil {
			return false // Can't parse spawn time - keep the agent (safer)
		}
		return spawnTime.Before(cutoff)
	})

	if removed == 0 {
		return 0, nil
	}

	// Use SaveSkipMerge since Purge modifies in-memory state;
	// regular Save would re-merge purged entries from disk.
	if err := reg.SaveSkipMerge(); err != nil {
		return 0, err
	}

	return removed, nil
}

// runUntrackedAgentExpiry removes idle untracked agents from the registry.
// An agent is considered for removal if:
// 1. It has no beads_id OR beads_id contains "untracked"
// 2. It has been idle for more than the specified duration (idleHours)
// Uses Registry.Purge + Save to ensure proper file locking and merge semantics,
// preventing race conditions with concurrent agent registrations.
// Returns the number of agents removed and any error encountered.
func runUntrackedAgentExpiry(idleHours int) (int, error) {
	reg, err := registry.New("")
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().Add(-time.Duration(idleHours) * time.Hour)

	removed := reg.Purge(func(agent *registry.Agent) bool {
		// Only purge untracked agents
		isUntracked := agent.BeadsID == "" || strings.Contains(agent.BeadsID, "untracked")
		if !isUntracked {
			return false
		}

		// Only purge if idle past cutoff
		updatedTime, err := time.Parse(registry.TimeFormat, agent.UpdatedAt)
		if err != nil {
			return false // Can't parse time - keep the agent (safer)
		}
		return updatedTime.Before(cutoff)
	})

	if removed == 0 {
		return 0, nil
	}

	// Use SaveSkipMerge since Purge modifies in-memory state;
	// regular Save would re-merge purged entries from disk.
	if err := reg.SaveSkipMerge(); err != nil {
		return 0, err
	}

	return removed, nil
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
