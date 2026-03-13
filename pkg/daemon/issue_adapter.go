// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/checkpoint"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// isBdTimeoutError returns true if the error is a bd command timeout.
// Useful for logging and circuit breaker tracking.
func isBdTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	// exec.CommandContext kills the process on timeout, producing an ExitError
	// with context.DeadlineExceeded as the underlying cause
	if exitErr, ok := err.(*exec.ExitError); ok {
		_ = exitErr // The process was killed by context cancellation
	}
	return err.Error() == "signal: killed" || err.Error() == "context deadline exceeded"
}

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
			// Use IntPtr(0) to get ALL ready issues (bd ready defaults to limit 10)
			beadsIssues, err := client.Ready(&beads.ReadyArgs{Limit: beads.IntPtr(0)})
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

// listReadyIssuesCLI retrieves ready issues by shelling out to bd CLI.
func listReadyIssuesCLI() ([]Issue, error) {
	// Use --limit 0 to get ALL ready issues (bd ready defaults to limit 10)
	output, err := runBdCommand("ready", "--json", "--limit", "0")
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
			beadsIssues, err := client.List(&beads.ListArgs{Parent: epicID, Limit: beads.IntPtr(0)})
			if err == nil {
				return convertBeadsIssues(beadsIssues), nil
			}
			// Fall through to CLI fallback on List() error
		}
		// Fall through to CLI fallback on Connect() error
	}

	// Fallback to CLI if daemon unavailable
	beadsIssues, err := beads.FallbackListByParent(epicID, "")
	if err != nil {
		return nil, err
	}
	return convertBeadsIssues(beadsIssues), nil
}

// ListIssuesWithLabel lists open/in_progress issues with a specific label.
// Uses the beads RPC client if available, falling back to the bd CLI.
func ListIssuesWithLabel(label string) ([]Issue, error) {
	if label == "" {
		return []Issue{}, nil
	}

	// Try to use the beads RPC client first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			beadsIssues, err := client.List(&beads.ListArgs{
				LabelsAny: []string{label},
				Limit:     beads.IntPtr(0),
			})
			if err == nil {
				// Filter to open/in_progress only (RPC list returns all statuses)
				var filtered []Issue
				for _, issue := range convertBeadsIssues(beadsIssues) {
					if issue.Status == "open" || issue.Status == "in_progress" {
						filtered = append(filtered, issue)
					}
				}
				return filtered, nil
			}
			// Fall through to CLI fallback on List() error
		}
		// Fall through to CLI fallback on Connect() error
	}

	// Fallback to CLI
	return listIssuesWithLabelCLI(label)
}

// listIssuesWithLabelCLI retrieves issues with a label by shelling out to bd CLI.
func listIssuesWithLabelCLI(label string) ([]Issue, error) {
	output, err := runBdCommand("list", "--json", "--limit", "0", "-l", label)
	if err != nil {
		return nil, fmt.Errorf("failed to run bd list -l %s: %w", label, err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	// Filter to open/in_progress only
	var filtered []Issue
	for _, issue := range issues {
		if issue.Status == "open" || issue.Status == "in_progress" {
			filtered = append(filtered, issue)
		}
	}
	return filtered, nil
}

// CountUnverifiedCompletions counts unverified completions using the canonical
// verify.ListUnverifiedWork() function. This ensures the daemon, spawn gate,
// and review command all agree on what constitutes "unverified work."
//
// Only counts checkpoints for OPEN issues (open/in_progress/blocked).
// Closed/deferred/tombstone issues are excluded.
//
// Falls back to the legacy counting logic if the canonical function fails
// (e.g., beads RPC unavailable). Better to overcount than undercount.
func CountUnverifiedCompletions() (int, error) {
	count, err := verify.CountUnverifiedWork()
	if err != nil {
		// Fall back to legacy counting if canonical function fails
		checkpoints, cpErr := checkpoint.ReadCheckpoints()
		if cpErr != nil {
			return 0, fmt.Errorf("failed to read checkpoints: %w", cpErr)
		}
		if len(checkpoints) == 0 {
			return 0, nil
		}
		return countUnverifiedWithoutFiltering(checkpoints)
	}
	return count, nil
}

// countUnverifiedWithoutFiltering is a fallback when open issues can't be fetched.
// Uses the old logic that attempts to look up each issue individually.
func countUnverifiedWithoutFiltering(checkpoints []checkpoint.Checkpoint) (int, error) {
	// Try RPC client first for issue lookups
	socketPath, err := beads.FindSocketPath("")
	var client *beads.Client
	useRPC := false
	if err == nil {
		client = beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			useRPC = true
			defer client.Close()
		}
	}

	// Count unverified completions based on tier
	unverified := 0
	for _, cp := range checkpoints {
		// Look up issue to determine tier
		var issueType string
		if useRPC {
			issue, err := client.Show(cp.BeadsID)
			if err != nil {
				// Issue may be deleted or inaccessible - skip
				continue
			}
			issueType = issue.IssueType
		} else {
			// Fallback to CLI
			issue, err := showIssueCLI(cp.BeadsID)
			if err != nil {
				// Issue may be deleted or inaccessible - skip
				continue
			}
			issueType = issue.IssueType
		}

		// Determine tier and check verification status
		tier := checkpoint.TierForIssueType(issueType)
		switch tier {
		case 1:
			// Tier 1 (feature/bug/decision): requires both gates
			if !cp.Gate2Complete {
				unverified++
			}
		case 2:
			// Tier 2 (investigation/probe): requires gate1 only
			if !cp.Gate1Complete {
				unverified++
			}
			// case 3: Tier 3 (task/question/other) never requires verification - skip
		}
	}

	return unverified, nil
}

// showIssueCLI fetches a single issue using bd CLI (fallback when RPC unavailable).
func showIssueCLI(beadsID string) (*beads.Issue, error) {
	output, err := runBdCommand("show", beadsID, "--json")
	if err != nil {
		return nil, fmt.Errorf("failed to run bd show: %w", err)
	}

	var issues []beads.Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse bd show output: %w", err)
	}

	if len(issues) == 0 {
		return nil, fmt.Errorf("issue not found: %s", beadsID)
	}

	return &issues[0], nil
}

// GetBeadsIssueStatus fetches the current status of a beads issue directly from beads.
// This is used for fresh status checks before spawning to prevent the TOCTOU race
// condition where the cached status from ListReadyIssues() is stale because another
// daemon process has already marked the issue as in_progress.
//
// Returns the issue status string ("open", "in_progress", "closed", etc.) or error.
func GetBeadsIssueStatus(beadsID string) (string, error) {
	// Try RPC first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			issue, err := client.Show(beadsID)
			if err == nil {
				return issue.Status, nil
			}
			// Fall through to CLI fallback
		}
	}

	// Fallback to CLI
	issue, err := beads.FallbackShow(beadsID, "")
	if err != nil {
		return "", fmt.Errorf("failed to get issue status: %w", err)
	}
	return issue.Status, nil
}

// FindInProgressByTitle returns the first in_progress issue with a matching title.
// Uses case-insensitive substring matching via beads List API.
// Returns nil if no match found. Fails open (returns nil on error) to avoid blocking work.
// This is the persistent layer of content-aware dedup - it survives daemon restarts
// because it queries the beads database directly.
func FindInProgressByTitle(title string) *Issue {
	if title == "" {
		return nil
	}

	// Try RPC first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			beadsIssues, err := client.List(&beads.ListArgs{
				Status: "in_progress",
				Title:  title,
				Limit:  beads.IntPtr(1),
			})
			if err == nil && len(beadsIssues) > 0 {
				issues := convertBeadsIssues(beadsIssues)
				return &issues[0]
			}
			// Fall through to CLI fallback
		}
	}

	// Fallback to CLI
	beadsIssues, err := beads.FallbackList("in_progress", "")
	if err != nil {
		return nil // fail-open
	}

	normalizedTarget := normalizeTitle(title)
	for _, bi := range beadsIssues {
		if normalizeTitle(bi.Title) == normalizedTarget {
			issue := Issue{
				ID:        bi.ID,
				Title:     bi.Title,
				Status:    bi.Status,
				IssueType: bi.IssueType,
			}
			return &issue
		}
	}

	return nil
}

// SpawnWork spawns work on a beads issue using orch work command.
// This is the default implementation that shells out to orch.
// If model is non-empty, it passes --model to orch work for model-aware routing.
// If workdir is non-empty, it passes --workdir for cross-project spawning.
// If account is non-empty, it passes --account for group-based account routing.
func SpawnWork(beadsID, model, workdir, account string) error {
	args := []string{"work"}
	if model != "" {
		args = append(args, "--model", model)
	}
	if workdir != "" {
		args = append(args, "--workdir", workdir)
	}
	if account != "" {
		args = append(args, "--account", account)
	}
	args = append(args, beadsID)
	cmd := exec.Command("orch", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to spawn work: %w: %s", err, string(output))
	}
	return nil
}

// ListReadyIssuesForProject retrieves ready issues from a specific project directory.
// Tags each returned issue with ProjectDir so the caller knows which project it came from.
func ListReadyIssuesForProject(projectDir string) ([]Issue, error) {
	// Try RPC first
	socketPath, err := beads.FindSocketPath(projectDir)
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			beadsIssues, err := client.Ready(&beads.ReadyArgs{Limit: beads.IntPtr(0)})
			if err == nil {
				issues := convertBeadsIssues(beadsIssues)
				for i := range issues {
					issues[i].ProjectDir = projectDir
				}
				return issues, nil
			}
		}
	}

	// Fallback to CLI with Dir set
	output, err := runBdCommandInDir(projectDir, "ready", "--json", "--limit", "0")
	if err != nil {
		return nil, fmt.Errorf("failed to run bd ready in %s: %w", projectDir, err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues from %s: %w", projectDir, err)
	}
	for i := range issues {
		issues[i].ProjectDir = projectDir
	}
	return issues, nil
}

// ListReadyIssuesMultiProject retrieves ready issues from all registered projects.
// Deduplicates by issue ID. Clears ProjectDir for issues from the current project
// (no --workdir needed). Falls back to ListReadyIssues when registry is nil.
func ListReadyIssuesMultiProject(registry *ProjectRegistry) ([]Issue, error) {
	if registry == nil {
		return ListReadyIssues()
	}

	seen := make(map[string]bool)
	var allIssues []Issue

	for _, proj := range registry.Projects() {
		issues, err := ListReadyIssuesForProject(proj.Dir)
		if err != nil {
			// Log warning but continue with other projects
			fmt.Fprintf(os.Stderr, "Warning: failed to list issues for %s: %v\n", proj.Dir, err)
			continue
		}
		for _, issue := range issues {
			if seen[issue.ID] {
				continue
			}
			seen[issue.ID] = true
			// Clear ProjectDir for current-project issues (no workdir needed)
			if issue.ProjectDir == registry.CurrentDir() {
				issue.ProjectDir = ""
			}
			allIssues = append(allIssues, issue)
		}
	}

	// If no projects returned any issues, fall back to local query
	// This handles the case where the registry has projects but none have beads sockets
	if len(allIssues) == 0 && len(registry.Projects()) > 0 {
		return ListReadyIssues()
	}

	return allIssues, nil
}

// UpdateBeadsStatusForProject updates the status of a beads issue in a specific project.
// When projectDir is empty, delegates to UpdateBeadsStatus (current project).
func UpdateBeadsStatusForProject(beadsID, status, projectDir string) error {
	if projectDir == "" {
		return UpdateBeadsStatus(beadsID, status)
	}

	// Try RPC first
	socketPath, err := beads.FindSocketPath(projectDir)
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			statusPtr := &status
			_, err := client.Update(&beads.UpdateArgs{
				ID:     beadsID,
				Status: statusPtr,
			})
			if err == nil {
				return nil
			}
		}
	}

	// Fallback to CLI with Dir set
	args := []string{"update", beadsID}
	if status != "" {
		args = append(args, "--status", status)
	}
	output, err := runBdCommandCombinedInDir(projectDir, args...)
	if err != nil {
		return fmt.Errorf("bd update failed in %s: %w: %s", projectDir, err, string(output))
	}
	return nil
}

// GetBeadsIssueStatusForProject fetches the current status of a beads issue in a specific project.
// When projectDir is empty, delegates to GetBeadsIssueStatus (current project).
func GetBeadsIssueStatusForProject(beadsID, projectDir string) (string, error) {
	if projectDir == "" {
		return GetBeadsIssueStatus(beadsID)
	}

	// Try RPC first
	socketPath, err := beads.FindSocketPath(projectDir)
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			issue, err := client.Show(beadsID)
			if err == nil {
				return issue.Status, nil
			}
		}
	}

	// Fallback to CLI with Dir set
	output, err := runBdCommandInDir(projectDir, "show", beadsID, "--json")
	if err != nil {
		return "", fmt.Errorf("bd show failed in %s: %w", projectDir, err)
	}

	var issues []beads.Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return "", fmt.Errorf("failed to parse bd show output from %s: %w", projectDir, err)
	}
	if len(issues) == 0 {
		return "", fmt.Errorf("issue not found in %s: %s", projectDir, beadsID)
	}
	return issues[0].Status, nil
}

// UpdateBeadsStatus updates the status of a beads issue.
// Uses the beads RPC client if available, falling back to CLI.
// This is called by the daemon to mark issues as in_progress before spawning.
func UpdateBeadsStatus(beadsID, status string) error {
	// Try to use the beads RPC client first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			statusPtr := &status
			_, err := client.Update(&beads.UpdateArgs{
				ID:     beadsID,
				Status: statusPtr,
			})
			if err == nil {
				return nil
			}
			// Fall through to CLI fallback on Update() error
		}
		// Fall through to CLI fallback on Connect() error
	}

	// Fallback to CLI if daemon unavailable
	return beads.FallbackUpdate(beadsID, status, "")
}

// ListIssuesWithLabelForProject lists open/in_progress issues with a label in a specific project.
func ListIssuesWithLabelForProject(label, projectDir string) ([]Issue, error) {
	if projectDir == "" {
		return ListIssuesWithLabel(label)
	}

	// Try RPC first
	socketPath, err := beads.FindSocketPath(projectDir)
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			beadsIssues, err := client.List(&beads.ListArgs{
				LabelsAny: []string{label},
				Limit:     beads.IntPtr(0),
			})
			if err == nil {
				var filtered []Issue
				for _, issue := range convertBeadsIssues(beadsIssues) {
					if issue.Status == "open" || issue.Status == "in_progress" {
						filtered = append(filtered, issue)
					}
				}
				return filtered, nil
			}
		}
	}

	// Fallback to CLI
	output, err := runBdCommandInDir(projectDir, "list", "--json", "--limit", "0", "-l", label)
	if err != nil {
		return nil, fmt.Errorf("failed to run bd list -l %s in %s: %w", label, projectDir, err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	var filtered []Issue
	for _, issue := range issues {
		if issue.Status == "open" || issue.Status == "in_progress" {
			filtered = append(filtered, issue)
		}
	}
	return filtered, nil
}

// CloseIssueForProject closes a beads issue in a specific project directory.
func CloseIssueForProject(beadsID, projectDir, reason string) error {
	args := []string{"close", beadsID}
	if reason != "" {
		args = append(args, "--reason", reason)
	}

	if projectDir == "" {
		output, err := runBdCommandCombined(args...)
		if err != nil {
			return fmt.Errorf("bd close failed: %w: %s", err, string(output))
		}
		return nil
	}

	output, err := runBdCommandCombinedInDir(projectDir, args...)
	if err != nil {
		return fmt.Errorf("bd close failed in %s: %w: %s", projectDir, err, string(output))
	}
	return nil
}
