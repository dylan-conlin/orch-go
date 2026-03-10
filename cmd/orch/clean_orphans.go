// Package main provides orphan and ghost agent cleanup functions for the clean command.
// Extracted from clean_cmd.go for cohesion (orphan GC, ghost agent label cleanup).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/agent"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// detectOrphansReport runs orphan detection and returns formatted report lines.
// Used in default mode (no flags) for reporting only — no GC actions taken.
func detectOrphansReport(projectDir string) ([]string, error) {
	lm := buildLifecycleManager(projectDir, serverURL, "", "")
	result, err := lm.DetectOrphans([]string{projectDir}, 30*time.Minute)
	if err != nil {
		return nil, err
	}
	if len(result.Orphans) == 0 {
		return nil, nil
	}

	var lines []string
	for _, orphan := range result.Orphans {
		action := "force-complete"
		if orphan.ShouldRetry {
			action = "force-abandon"
		}
		detail := orphan.Reason
		if orphan.LastPhase != "" {
			detail += fmt.Sprintf(", phase: %s", orphan.LastPhase)
		}
		if orphan.StaleFor > 0 {
			detail += fmt.Sprintf(", stale %v", orphan.StaleFor.Round(time.Minute))
		}
		lines = append(lines, fmt.Sprintf("%s → %s (%s)", orphan.Agent.BeadsID, action, detail))
	}
	return lines, nil
}

// runOrphanGC detects orphaned agents and performs lifecycle GC transitions.
// Uses LifecycleManager.DetectOrphans to find agents tagged orch:agent with no live
// execution, then applies ForceComplete (for completed orphans) or ForceAbandon
// (for retryable orphans) to clean up state consistently.
func runOrphanGC(projectDir string, dryRun bool, preserveOrchestrator bool) (forceCompleted int, forceAbandoned int, err error) {
	fmt.Println("\nScanning for orphaned agents...")

	// Build lifecycle manager for detection (agentName/beadsID not needed for DetectOrphans)
	lm := buildLifecycleManager(projectDir, serverURL, "", "")

	result, err := lm.DetectOrphans([]string{projectDir}, 30*time.Minute)
	if err != nil {
		return 0, 0, fmt.Errorf("orphan detection failed: %w", err)
	}

	fmt.Printf("  Scanned %d tracked agents in %v\n", result.Scanned, result.Elapsed.Round(time.Millisecond))

	if len(result.Orphans) == 0 {
		fmt.Println("  No orphaned agents found")
		return 0, 0, nil
	}

	fmt.Printf("  Found %d orphaned agents:\n", len(result.Orphans))

	for _, orphan := range result.Orphans {
		// Skip orchestrator workspaces if requested
		if preserveOrchestrator && orphan.Agent.WorkspacePath != "" && isOrchestratorWorkspace(orphan.Agent.WorkspacePath) {
			fmt.Printf("    Skipped (orchestrator): %s\n", orphan.Agent.BeadsID)
			continue
		}

		action := "force-complete"
		if orphan.ShouldRetry {
			action = "force-abandon"
		}

		// Format details for output
		detail := orphan.Reason
		if orphan.LastPhase != "" {
			detail += fmt.Sprintf(", phase: %s", orphan.LastPhase)
		}
		if orphan.StaleFor > 0 {
			detail += fmt.Sprintf(", stale %v", orphan.StaleFor.Round(time.Minute))
		}

		if dryRun {
			fmt.Printf("    [DRY-RUN] Would %s: %s (%s)\n", action, orphan.Agent.BeadsID, detail)
			if orphan.ShouldRetry {
				forceAbandoned++
			} else {
				forceCompleted++
			}
			continue
		}

		// Build per-agent lifecycle manager (workspace adapter needs agent-specific params)
		agentLM := buildLifecycleManager(projectDir, serverURL, orphan.Agent.WorkspaceName, orphan.Agent.BeadsID)

		var event *agent.TransitionEvent
		if orphan.ShouldRetry {
			event, err = agentLM.ForceAbandon(orphan.Agent)
			if err != nil {
				fmt.Fprintf(os.Stderr, "    Warning: force-abandon failed for %s: %v\n", orphan.Agent.BeadsID, err)
				continue
			}
			if event.Success {
				fmt.Printf("    Force-abandoned: %s (will retry via respawn)\n", orphan.Agent.BeadsID)
				forceAbandoned++
			}
		} else {
			reason := fmt.Sprintf("GC: orphaned agent (%s)", detail)
			event, err = agentLM.ForceComplete(orphan.Agent, reason)
			if err != nil {
				fmt.Fprintf(os.Stderr, "    Warning: force-complete failed for %s: %v\n", orphan.Agent.BeadsID, err)
				continue
			}
			if event.Success {
				fmt.Printf("    Force-completed: %s (%s)\n", orphan.Agent.BeadsID, detail)
				forceCompleted++
			}
		}

		// Report effect details
		for _, e := range event.Effects {
			if e.Critical && !e.Success {
				fmt.Fprintf(os.Stderr, "    Warning: %s/%s failed for %s: %v\n", e.Subsystem, e.Operation, orphan.Agent.BeadsID, e.Error)
			}
		}
		for _, w := range event.Warnings {
			fmt.Fprintf(os.Stderr, "    Warning: %s\n", w)
		}
	}

	return forceCompleted, forceAbandoned, nil
}

// NOTE: extractBeadsIDFromWorkspace is defined in review.go

// cleanGhostAgents finds cross-project beads issues with stale orch:agent labels
// and removes the label. A "ghost" is an issue that appears in orch status via
// cross-project beads query (orch:agent label + in_progress) but has no active
// agent working on it (no workspace, no session).
//
// Ghost agents are caused by agents that died without proper cleanup — the
// orch:agent label was never removed. This makes them permanently visible in
// orch status with no way to dismiss them.
func cleanGhostAgents(currentProjectDir string, dryRun bool) (int, error) {
	projectDirs := getKBProjectsFn()
	client := opencode.NewClient(opencode.DefaultServerURL)
	cleaned := 0

	// Phase 1: Clean stale orch:agent labels on closed issues in the local project.
	// These are missed by --orphans (which only looks at open/in_progress issues)
	// and by the cross-project loop below (which skips the current project).
	localIssues, err := beads.FallbackListWithLabel("orch:agent", "")
	if err == nil {
		for _, issue := range localIssues {
			if strings.EqualFold(issue.Status, "closed") {
				if dryRun {
					fmt.Printf("  Stale label on closed issue: %s (%s)\n", issue.ID, issue.Title)
					cleaned++
					continue
				}
				if err := beads.FallbackRemoveLabel(issue.ID, "orch:agent", ""); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to remove orch:agent from %s: %v\n", issue.ID, err)
				} else {
					fmt.Printf("  Cleaned stale label: %s (closed)\n", issue.ID)
					cleaned++
				}
			}
		}
	}

	// Phase 2: Clean cross-project ghost agents (open/in_progress with no live agent,
	// OR closed issues with stale labels).
	if len(projectDirs) == 0 {
		return cleaned, nil
	}

	for _, dir := range projectDirs {
		// Skip current project — local issues handled in Phase 1 and by --orphans
		if filepath.Clean(dir) == filepath.Clean(currentProjectDir) {
			continue
		}

		// Find orch:agent labeled issues in this project
		issues, err := beads.FallbackListWithLabel("orch:agent", dir)
		if err != nil {
			continue
		}

		for _, issue := range issues {
			isGhost := false
			reason := ""

			if strings.EqualFold(issue.Status, "closed") {
				// Closed issue with stale label — always a ghost
				isGhost = true
				reason = "closed"
			} else if issue.Status == "open" || issue.Status == "in_progress" {
				// Check if there's an active workspace for this issue
				wPath, _ := findWorkspaceByBeadsID(dir, issue.ID)

				// Check if there's an active OpenCode session
				hasSession := false
				if wPath != "" {
					sessionID := spawn.ReadSessionID(wPath)
					if sessionID != "" {
						hasSession = client.SessionExists(sessionID)
					}
				}

				// If no workspace and no session, this is a ghost
				if wPath == "" || !hasSession {
					isGhost = true
					reason = "no active agent"
				}
			}

			if isGhost {
				if dryRun {
					fmt.Printf("  Ghost: %s in %s (%s) [%s]\n", issue.ID, filepath.Base(dir), issue.Title, reason)
					cleaned++
					continue
				}

				removeLabelErr := beads.FallbackRemoveLabel(issue.ID, "orch:agent", dir)
				if removeLabelErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to remove orch:agent from %s: %v\n", issue.ID, removeLabelErr)
					continue
				}
				fmt.Printf("  Cleaned ghost: %s in %s [%s]\n", issue.ID, filepath.Base(dir), reason)
				cleaned++
			}
		}
	}

	return cleaned, nil
}
