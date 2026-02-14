// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

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
