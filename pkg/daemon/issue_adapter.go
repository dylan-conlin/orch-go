// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/checkpoint"
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
				Limit:     0,
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
	cmd := exec.Command("bd", "list", "--json", "--limit", "0", "-l", label)
	cmd.Env = os.Environ()
	output, err := cmd.Output()
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

// CountUnverifiedCompletions counts open/in_progress issues with
// the daemon:ready-review label that don't have verification checkpoint entries.
// This represents the backlog of daemon-completed work awaiting human review.
func CountUnverifiedCompletions() (int, error) {
	readyForReview, err := ListIssuesWithLabel("daemon:ready-review")
	if err != nil {
		return 0, fmt.Errorf("failed to list ready-for-review issues: %w", err)
	}

	if len(readyForReview) == 0 {
		return 0, nil
	}

	// Read checkpoint file
	checkpoints, err := checkpoint.ReadCheckpoints()
	if err != nil {
		// Checkpoint file missing/corrupt - count all as unverified
		return len(readyForReview), nil
	}

	// Build set of checkpoint beads IDs (only gate1 completed = verified)
	checkpointIDs := make(map[string]bool)
	for _, cp := range checkpoints {
		if cp.Gate1Complete {
			checkpointIDs[cp.BeadsID] = true
		}
	}

	// Count issues without checkpoints
	unverified := 0
	for _, issue := range readyForReview {
		if !checkpointIDs[issue.ID] {
			unverified++
		}
	}

	return unverified, nil
}

// SpawnWork spawns work on a beads issue using orch work command.
// This is the default implementation that shells out to orch.
func SpawnWork(beadsID string) error {
	cmd := exec.Command("orch", "work", beadsID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to spawn work: %w: %s", err, string(output))
	}
	return nil
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
	return beads.FallbackUpdate(beadsID, status)
}
