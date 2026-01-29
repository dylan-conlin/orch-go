// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// ListReadyIssues retrieves ready issues from beads (open or in_progress, no blockers).
// It uses the beads RPC daemon if available, falling back to the bd CLI if not.
// Uses WithAutoReconnect for resilience against transient connection issues.
func ListReadyIssues() ([]Issue, error) {
	// Try to use the beads RPC client first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		// Use WithAutoReconnect for resilience against daemon restarts/transient issues
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			// Use Limit: 0 to get ALL ready issues (bd ready defaults to limit 10)
			beadsIssues, err := client.Ready(&beads.ReadyArgs{Limit: 0})
			if err == nil {
				return convertBeadsIssues(beadsIssues), nil
			}
			// Fall through to CLI fallback on Ready() error
		}
		// Fall through to CLI fallback on Connect() error
	}

	// Fallback to CLI if daemon unavailable
	return listReadyIssuesCLI()
}

// ListReadyIssuesForProject returns triage:ready issues for a specific project.
// Uses the beads RPC daemon if available at that project path, falling back to CLI.
// On error, returns empty list with logged warning (does not crash).
// Returns empty list (no error) for projects without .beads/ directory.
func ListReadyIssuesForProject(projectPath string) ([]Issue, error) {
	if projectPath == "" {
		return nil, fmt.Errorf("projectPath is required")
	}

	// Skip projects without beads initialized (avoids noisy warnings)
	beadsDir := filepath.Join(projectPath, ".beads")
	if _, err := os.Stat(beadsDir); os.IsNotExist(err) {
		return []Issue{}, nil
	}

	// Try to use the beads RPC client first
	socketPath, err := beads.FindSocketPath(projectPath)
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			// Use Limit: 0 to get ALL ready issues (bd ready defaults to limit 10)
			beadsIssues, err := client.Ready(&beads.ReadyArgs{Limit: 0})
			if err == nil {
				return convertBeadsIssues(beadsIssues), nil
			}
			// Fall through to CLI fallback on Ready() error
		}
		// Fall through to CLI fallback on Connect() error
	}

	// Fallback to CLI if daemon unavailable
	return listReadyIssuesForProjectCLI(projectPath)
}

// listReadyIssuesForProjectCLI retrieves ready issues for a project by shelling out to bd CLI.
func listReadyIssuesForProjectCLI(projectPath string) ([]Issue, error) {
	// Use --limit 0 to get ALL ready issues (bd ready defaults to limit 10)
	cmd := exec.Command("bd", "ready", "--json", "--limit", "0")
	cmd.Dir = projectPath
	cmd.Env = os.Environ() // Inherit env (including BEADS_NO_DAEMON)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("warning: failed to get ready issues for project %s: %v", projectPath, err)
		return []Issue{}, nil // Return empty list, not error (per acceptance criteria)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		log.Printf("warning: failed to parse ready issues for project %s: %v", projectPath, err)
		return []Issue{}, nil // Return empty list, not error
	}

	return issues, nil
}

// listReadyIssuesCLI retrieves ready issues by shelling out to bd CLI.
func listReadyIssuesCLI() ([]Issue, error) {
	// Use --limit 0 to get ALL ready issues (bd ready defaults to limit 10)
	cmd := exec.Command("bd", "ready", "--json", "--limit", "0")
	cmd.Env = os.Environ() // Inherit env (including BEADS_NO_DAEMON)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run bd ready: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	return issues, nil
}

// convertBeadsIssues converts beads.Issue slice to daemon.Issue slice.
func convertBeadsIssues(beadsIssues []beads.Issue) []Issue {
	issues := make([]Issue, len(beadsIssues))
	for i, bi := range beadsIssues {
		issues[i] = Issue{
			ID:          bi.ID,
			Title:       bi.Title,
			Description: bi.Description,
			Priority:    bi.Priority,
			Status:      bi.Status,
			IssueType:   bi.IssueType,
			Labels:      bi.Labels,
		}
	}
	return issues
}

// ListOpenIssues is an alias for ListReadyIssues for backward compatibility.
// Deprecated: Use ListReadyIssues instead.
func ListOpenIssues() ([]Issue, error) {
	return ListReadyIssues()
}

// ListEpicChildren retrieves children of an epic by its ID.
// Uses the beads RPC client if available, falling back to CLI.
func ListEpicChildren(epicID string) ([]Issue, error) {
	if epicID == "" {
		return []Issue{}, nil
	}

	// Try to use the beads RPC client first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			beadsIssues, err := client.List(&beads.ListArgs{Parent: epicID, Limit: 0})
			if err == nil {
				return convertBeadsIssues(beadsIssues), nil
			}
			// Fall through to CLI fallback on List() error
		}
		// Fall through to CLI fallback on Connect() error
	}

	// Fallback to CLI if daemon unavailable
	beadsIssues, err := beads.FallbackListByParent(epicID)
	if err != nil {
		return nil, err
	}
	return convertBeadsIssues(beadsIssues), nil
}

// HasPhaseComplete checks if an issue has "Phase: Complete" in its comments.
// This prevents respawning work that an agent has already completed but the
// orchestrator hasn't closed yet (e.g., waiting for review).
// Uses the beads RPC daemon if available, falling back to the bd CLI if not.
func HasPhaseComplete(beadsID string) (bool, error) {
	return HasPhaseCompleteForProject(beadsID, "")
}

// HasPhaseCompleteForProject checks if an issue has "Phase: Complete" in its
// comments, using a specific project path for beads socket lookup.
// If projectPath is empty, uses the current working directory.
func HasPhaseCompleteForProject(beadsID, projectPath string) (bool, error) {
	if beadsID == "" {
		return false, nil
	}

	// Try to use the beads RPC client first
	socketPath, err := beads.FindSocketPath(projectPath)
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			comments, err := client.Comments(beadsID)
			if err == nil {
				return checkCommentsForPhaseComplete(comments), nil
			}
			// Fall through to CLI fallback on Comments() error
		}
		// Fall through to CLI fallback on Connect() error
	}

	// Fallback to CLI if daemon unavailable
	return hasPhaseCompleteCLI(beadsID, projectPath)
}

// hasPhaseCompleteCLI checks for Phase: Complete via bd CLI.
func hasPhaseCompleteCLI(beadsID, projectPath string) (bool, error) {
	cmd := exec.Command("bd", "comments", beadsID, "--json")
	if projectPath != "" {
		cmd.Dir = projectPath
	}
	cmd.Env = os.Environ()
	output, err := cmd.Output()
	if err != nil {
		// If the command fails (e.g., invalid beads ID), treat as not complete
		// rather than failing the spawn flow entirely
		log.Printf("warning: failed to check Phase: Complete for %s: %v", beadsID, err)
		return false, nil
	}

	var comments []beads.Comment
	if err := json.Unmarshal(output, &comments); err != nil {
		log.Printf("warning: failed to parse comments for %s: %v", beadsID, err)
		return false, nil
	}

	return checkCommentsForPhaseComplete(comments), nil
}

// checkCommentsForPhaseComplete checks if any comment contains "Phase: Complete".
// The check is case-insensitive for the phase name portion.
func checkCommentsForPhaseComplete(comments []beads.Comment) bool {
	for _, c := range comments {
		// Check for "Phase: Complete" case-insensitively
		// Common formats: "Phase: Complete", "Phase: Complete - summary", etc.
		text := strings.ToLower(c.Text)
		if strings.Contains(text, "phase: complete") {
			return true
		}
	}
	return false
}

// SpawnWork spawns work on a beads issue using orch work command.
// This is the default implementation that spawns in the current working directory.
// Delegates to SpawnWorkForProject for consistency.
//
// IMPORTANT: Sets status to in_progress BEFORE spawning to prevent duplicate
// spawns. This is critical because:
// 1. The in-memory SpawnedIssueTracker doesn't survive daemon restarts
// 2. SessionDedupChecker only works for OpenCode sessions, not Claude CLI
// 3. Without this, daemon restart + Claude CLI backend = respawn loop
func SpawnWork(beadsID string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	return SpawnWorkForProject(beadsID, cwd)
}

// SpawnWorkForProject spawns work on a beads issue in a specific project directory.
// Uses --workdir flag to ensure the agent operates in the correct project context.
//
// IMPORTANT: Sets status to in_progress BEFORE spawning (same dedup mechanism as SpawnWork).
// If spawn fails, rolls back status to open so the issue can be retried.
func SpawnWorkForProject(beadsID, projectPath string) error {
	if projectPath == "" {
		return fmt.Errorf("projectPath is required")
	}

	// Extract project name for logging
	projectName := filepath.Base(projectPath)
	log.Printf("[%s] Spawning work for issue %s", projectName, beadsID)

	// Set status to in_progress BEFORE spawning to prevent duplicate spawns.
	updateCmd := exec.Command("bd", "update", beadsID, "--status=in_progress")
	updateCmd.Dir = projectPath
	updateCmd.Env = os.Environ()
	if output, err := updateCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("[%s] failed to set status to in_progress: %w: %s", projectName, err, string(output))
	}

	cmd := exec.Command("orch", "work", beadsID, "--workdir", projectPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Rollback: set status back to open so the issue can be retried
		rollbackCmd := exec.Command("bd", "update", beadsID, "--status=open")
		rollbackCmd.Dir = projectPath
		rollbackCmd.Env = os.Environ()
		if rollbackOutput, rollbackErr := rollbackCmd.CombinedOutput(); rollbackErr != nil {
			log.Printf("[%s] WARNING: failed to rollback status to open for %s: %v: %s", projectName, beadsID, rollbackErr, string(rollbackOutput))
		} else {
			log.Printf("[%s] Rolled back status to open for %s after spawn failure", projectName, beadsID)
		}
		return fmt.Errorf("[%s] failed to spawn work: %w: %s", projectName, err, string(output))
	}
	log.Printf("[%s] Successfully spawned work for issue %s", projectName, beadsID)
	return nil
}
