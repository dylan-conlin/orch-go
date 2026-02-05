// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"os"
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
// Returns the number of entries removed and any error encountered.
func runRegistryCleanup(ageDays int) (int, error) {
	reg, err := registry.New("")
	if err != nil {
		return 0, err
	}

	agents := reg.ListAgents()
	if len(agents) == 0 {
		return 0, nil
	}

	cutoff := time.Now().AddDate(0, 0, -ageDays)

	var toKeep []*registry.Agent
	removed := 0
	for _, agent := range agents {
		spawnTime, err := time.Parse(registry.TimeFormat, agent.SpawnedAt)
		if err != nil {
			// Can't parse spawn time - keep the agent (safer)
			toKeep = append(toKeep, agent)
			continue
		}
		if spawnTime.Before(cutoff) {
			removed++
		} else {
			toKeep = append(toKeep, agent)
		}
	}

	if removed == 0 {
		return 0, nil
	}

	// Rebuild registry file with only entries to keep
	type registryData struct {
		Agents []*registry.Agent `json:"agents"`
	}
	data, err := json.MarshalIndent(registryData{Agents: toKeep}, "", "  ")
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(registry.DefaultPath(), data, 0644); err != nil {
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
