// Package main provides OpenCode session cleanup for the clean command.
// Extracted from clean_cmd.go for per-concern file organization.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/cleanup"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// cleanOrphanedDiskSessions finds and deletes OpenCode disk sessions that aren't tracked via workspace files.
func cleanOrphanedDiskSessions(serverURL, projectDir string, dryRun bool, preserveOrchestrator bool) (int, error) {
	return cleanOrphanedDiskSessionsWithClient(opencode.NewClient(serverURL), projectDir, dryRun, preserveOrchestrator)
}

func cleanOrphanedDiskSessionsWithClient(client opencode.ClientInterface, projectDir string, dryRun bool, preserveOrchestrator bool) (int, error) {

	fmt.Printf("\nVerifying OpenCode disk sessions for %s...\n", projectDir)

	diskSessions, err := client.ListDiskSessions(projectDir)
	if err != nil {
		return 0, fmt.Errorf("failed to list disk sessions: %w", err)
	}

	fmt.Printf("  Found %d disk sessions\n", len(diskSessions))

	trackedSessionIDs := make(map[string]bool)
	orchestratorSessionIDs := make(map[string]bool)
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	if entries, err := os.ReadDir(workspaceDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				wsPath := filepath.Join(workspaceDir, entry.Name())
				sessionID := spawn.ReadSessionID(wsPath)
				if sessionID != "" {
					trackedSessionIDs[sessionID] = true
					if isOrchestratorWorkspace(wsPath) {
						orchestratorSessionIDs[sessionID] = true
					}
				}
			}
		}
	}

	fmt.Printf("  Workspaces track %d session IDs\n", len(trackedSessionIDs))
	if preserveOrchestrator && len(orchestratorSessionIDs) > 0 {
		fmt.Printf("  Found %d orchestrator session IDs to preserve\n", len(orchestratorSessionIDs))
	}

	var orphanedSessions []opencode.Session
	var skippedActive int
	now := time.Now()
	const recentActivityThreshold = 5 * time.Minute

	for _, session := range diskSessions {
		if !trackedSessionIDs[session.ID] {
			updatedAt := time.Unix(session.Time.Updated/1000, 0)
			isRecentlyActive := now.Sub(updatedAt) <= recentActivityThreshold

			if isRecentlyActive {
				if client.IsSessionProcessing(session.ID) {
					skippedActive++
					continue
				}
			}
			orphanedSessions = append(orphanedSessions, session)
		}
	}

	if skippedActive > 0 {
		fmt.Printf("  Skipped %d active sessions (currently processing)\n", skippedActive)
	}

	if len(orphanedSessions) == 0 {
		fmt.Println("  No orphaned disk sessions found")
		return 0, nil
	}

	fmt.Printf("  Found %d orphaned disk sessions:\n", len(orphanedSessions))

	deleted := 0
	skippedOrch := 0
	for _, session := range orphanedSessions {
		title := session.Title
		if title == "" {
			title = "(untitled)"
		}

		if preserveOrchestrator && orchestratorSessionIDs[session.ID] {
			skippedOrch++
			continue
		}

		if preserveOrchestrator && cleanup.IsOrchestratorSessionTitle(title) {
			skippedOrch++
			continue
		}

		if dryRun {
			fmt.Printf("    [DRY-RUN] Would delete: %s (%s)\n", session.ID[:12], title)
			deleted++
			continue
		}

		if err := client.DeleteSession(session.ID); err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: failed to delete %s: %v\n", session.ID[:12], err)
			continue
		}

		fmt.Printf("    Deleted: %s (%s)\n", session.ID[:12], title)
		deleted++
	}

	if skippedOrch > 0 {
		fmt.Printf("  Skipped %d orchestrator sessions (--preserve-orchestrator)\n", skippedOrch)
	}

	return deleted, nil
}
