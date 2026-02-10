// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"errors"
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
	var ready []Issue
	err := beads.Do("", func(client *beads.Client) error {
		if connErr := client.Connect(); connErr != nil {
			return connErr
		}
		defer client.Close()

		// Use Limit: 0 to get ALL ready issues (bd ready defaults to limit 10)
		beadsIssues, rpcErr := client.Ready(&beads.ReadyArgs{Limit: 0})
		if rpcErr != nil {
			return rpcErr
		}
		ready = filterAccessibleReadyIssues(
			convertBeadsIssues(beadsIssues),
			func(id string) error {
				_, showErr := client.Show(id)
				return showErr
			},
			"",
		)
		return nil
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return ready, nil
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

	var ready []Issue
	err := beads.Do(projectPath, func(client *beads.Client) error {
		if connErr := client.Connect(); connErr != nil {
			return connErr
		}
		defer client.Close()

		// Use Limit: 0 to get ALL ready issues (bd ready defaults to limit 10)
		beadsIssues, rpcErr := client.Ready(&beads.ReadyArgs{Limit: 0})
		if rpcErr != nil {
			return rpcErr
		}
		ready = filterAccessibleReadyIssues(
			convertBeadsIssues(beadsIssues),
			func(id string) error {
				_, showErr := client.Show(id)
				return showErr
			},
			projectPath,
		)
		return nil
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return ready, nil
	}

	// Fallback to CLI if daemon unavailable
	return listReadyIssuesForProjectCLI(projectPath)
}

// listReadyIssuesForProjectCLI retrieves ready issues for a project by shelling out to bd CLI.
func listReadyIssuesForProjectCLI(projectPath string) ([]Issue, error) {
	// Use --limit 0 to get ALL ready issues (bd ready defaults to limit 10)
	cmd := exec.Command("bd", "--sandbox", "--quiet", "ready", "--json", "--limit", "0")
	cmd.Dir = projectPath
	cmd.Env = os.Environ() // Inherit env (including BEADS_NO_DAEMON)
	output, err := bdOutput(cmd)
	if err != nil {
		log.Printf("warning: failed to get ready issues for project %s: %v", projectPath, err)
		return []Issue{}, nil // Return empty list, not error (per acceptance criteria)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		log.Printf("warning: failed to parse ready issues for project %s: %v", projectPath, err)
		return []Issue{}, nil // Return empty list, not error
	}

	return filterAccessibleReadyIssues(
		issues,
		func(id string) error {
			_, showErr := beads.FallbackShowWithDir(id, projectPath)
			return showErr
		},
		projectPath,
	), nil
}

// listReadyIssuesCLI retrieves ready issues by shelling out to bd CLI.
func listReadyIssuesCLI() ([]Issue, error) {
	// Use --limit 0 to get ALL ready issues (bd ready defaults to limit 10)
	cmd := exec.Command("bd", "--sandbox", "--quiet", "ready", "--json", "--limit", "0")
	cmd.Env = os.Environ() // Inherit env (including BEADS_NO_DAEMON)
	output, err := bdOutput(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to run bd ready: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	return filterAccessibleReadyIssues(
		issues,
		func(id string) error {
			_, showErr := beads.FallbackShowWithDir(id, "")
			return showErr
		},
		"",
	), nil
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
			UpdatedAt:   bi.UpdatedAt,
		}
	}
	return issues
}

// ListOpenIssues is an alias for ListReadyIssues for backward compatibility.
// Deprecated: Use ListReadyIssues instead.
func ListOpenIssues() ([]Issue, error) {
	return ListReadyIssues()
}

// ListReadyIssuesWithLabel returns ready issues filtered by a specific label.
// Uses the beads RPC daemon if available, falling back to CLI if not.
// If label is empty, behaves like ListReadyIssues (no filter).
func ListReadyIssuesWithLabel(label string) ([]Issue, error) {
	if label == "" {
		return ListReadyIssues()
	}

	var ready []Issue
	err := beads.Do("", func(client *beads.Client) error {
		if connErr := client.Connect(); connErr != nil {
			return connErr
		}
		defer client.Close()

		// Use Labels filter to only get issues with the specified label
		beadsIssues, rpcErr := client.Ready(&beads.ReadyArgs{
			Limit:  0,
			Labels: []string{label},
		})
		if rpcErr != nil {
			return rpcErr
		}
		ready = convertBeadsIssues(beadsIssues)
		return nil
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return ready, nil
	}

	// Fallback to CLI if daemon unavailable
	return listReadyIssuesWithLabelCLI(label)
}

// listReadyIssuesWithLabelCLI retrieves ready issues with a label by shelling out to bd CLI.
func listReadyIssuesWithLabelCLI(label string) ([]Issue, error) {
	cmd := exec.Command("bd", "--sandbox", "--quiet", "ready", "--json", "--limit", "0", "--label", label)
	cmd.Env = os.Environ()
	output, err := bdOutput(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to run bd ready: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	return filterAccessibleReadyIssues(
		issues,
		func(id string) error {
			_, showErr := beads.FallbackShowWithDir(id, "")
			return showErr
		},
		"",
	), nil
}

// filterAccessibleReadyIssues removes ready issues that cannot be resolved by ID.
// A ready issue that cannot be shown will fail later at spawn time, so we drop it
// early to keep the queue actionable. Unknown/transient validation errors keep the
// issue in the queue to avoid false negatives during temporary outages.
func filterAccessibleReadyIssues(issues []Issue, checkAccessible func(id string) error, projectPath string) []Issue {
	if len(issues) == 0 || checkAccessible == nil {
		return issues
	}

	filtered := make([]Issue, 0, len(issues))
	for _, issue := range issues {
		err := checkAccessible(issue.ID)
		if err == nil {
			filtered = append(filtered, issue)
			continue
		}

		if errors.Is(err, beads.ErrIssueNotFound) {
			if projectPath == "" {
				log.Printf("warning: dropping unfindable issue from ready queue: %s", issue.ID)
			} else {
				log.Printf("warning: dropping unfindable issue from ready queue for project %s: %s", projectPath, issue.ID)
			}
			continue
		}

		// Keep issue on non-not-found errors to avoid hiding valid work due to
		// transient connectivity/process failures.
		if projectPath == "" {
			log.Printf("warning: failed ready accessibility check for %s (keeping issue): %v", issue.ID, err)
		} else {
			log.Printf("warning: failed ready accessibility check for %s in project %s (keeping issue): %v", issue.ID, projectPath, err)
		}
		filtered = append(filtered, issue)
	}

	return filtered
}

// ListEpicChildren retrieves children of an epic by its ID.
// Uses the beads RPC client if available, falling back to CLI.
func ListEpicChildren(epicID string) ([]Issue, error) {
	if epicID == "" {
		return []Issue{}, nil
	}

	var children []Issue
	err := beads.Do("", func(client *beads.Client) error {
		if connErr := client.Connect(); connErr != nil {
			return connErr
		}
		defer client.Close()

		beadsIssues, rpcErr := client.List(&beads.ListArgs{Parent: epicID, Limit: 0})
		if rpcErr != nil {
			return rpcErr
		}
		children = convertBeadsIssues(beadsIssues)
		return nil
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return children, nil
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

	var hasComplete bool
	err := beads.Do(projectPath, func(client *beads.Client) error {
		if connErr := client.Connect(); connErr != nil {
			return connErr
		}
		defer client.Close()

		comments, rpcErr := client.Comments(beadsID)
		if rpcErr != nil {
			return rpcErr
		}
		hasComplete = checkCommentsForPhaseComplete(comments)
		return nil
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return hasComplete, nil
	}

	// Fallback to CLI if daemon unavailable
	return hasPhaseCompleteCLI(beadsID, projectPath)
}

// hasPhaseCompleteCLI checks for Phase: Complete via bd CLI.
func hasPhaseCompleteCLI(beadsID, projectPath string) (bool, error) {
	cmd := exec.Command("bd", "--sandbox", "--quiet", "comments", beadsID, "--json")
	if projectPath != "" {
		cmd.Dir = projectPath
	}
	cmd.Env = os.Environ()
	output, err := bdOutput(cmd)
	if err != nil {
		// Fail-safe: on error, assume Phase: Complete exists to prevent duplicate spawns.
		// Better to skip one spawn cycle than create a duplicate agent.
		log.Printf("warning: failed to check Phase: Complete for %s (assuming complete to prevent duplicate): %v", beadsID, err)
		return true, fmt.Errorf("bd comments failed: %w", err)
	}

	var comments []beads.Comment
	if err := json.Unmarshal(output, &comments); err != nil {
		// Fail-safe: on parse error, assume Phase: Complete exists to prevent duplicate spawns.
		log.Printf("warning: failed to parse comments for %s (assuming complete to prevent duplicate): %v", beadsID, err)
		return true, fmt.Errorf("json parse failed: %w", err)
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
	updateCmd := exec.Command("bd", "--sandbox", "--quiet", "update", beadsID, "--status=in_progress")
	updateCmd.Dir = projectPath
	updateCmd.Env = os.Environ()
	if output, err := bdCombinedOutput(updateCmd); err != nil {
		return fmt.Errorf("[%s] failed to set status to in_progress: %w: %s", projectName, err, string(output))
	}

	cmd := exec.Command("orch", "work", beadsID, "--workdir", projectPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Rollback: set status back to open so the issue can be retried
		rollbackCmd := exec.Command("bd", "--sandbox", "--quiet", "update", beadsID, "--status=open")
		rollbackCmd.Dir = projectPath
		rollbackCmd.Env = os.Environ()
		if rollbackOutput, rollbackErr := bdCombinedOutput(rollbackCmd); rollbackErr != nil {
			log.Printf("[%s] WARNING: failed to rollback status to open for %s: %v: %s", projectName, beadsID, rollbackErr, string(rollbackOutput))
		} else {
			log.Printf("[%s] Rolled back status to open for %s after spawn failure", projectName, beadsID)
		}
		return fmt.Errorf("[%s] failed to spawn work: %w: %s", projectName, err, string(output))
	}
	log.Printf("[%s] Successfully spawned work for issue %s", projectName, beadsID)
	return nil
}
