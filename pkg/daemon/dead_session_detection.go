// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// DeadSessionDetectionConfig holds configuration for dead session detection.
type DeadSessionDetectionConfig struct {
	// Verbose enables debug logging.
	Verbose bool
}

// DefaultDeadSessionDetectionConfig returns default configuration.
func DefaultDeadSessionDetectionConfig() DeadSessionDetectionConfig {
	return DeadSessionDetectionConfig{
		Verbose: false,
	}
}

// DeadSession represents a detected dead session.
type DeadSession struct {
	BeadsID string
	Title   string
	Reason  string // Why it's considered dead
}

// FindDeadSessions finds all issues with in_progress status that have no active
// session and no Phase: Complete comment. These are "zombie" issues where the
// agent died without completing.
//
// Returns a list of dead sessions and any error encountered.
func FindDeadSessions(config DeadSessionDetectionConfig) ([]DeadSession, error) {
	// Get all issues with in_progress status
	inProgressIssues, err := ListIssuesWithStatus("in_progress")
	if err != nil {
		return nil, fmt.Errorf("failed to list in_progress issues: %w", err)
	}

	if config.Verbose {
		fmt.Printf("  DEBUG: Found %d in_progress issues\n", len(inProgressIssues))
	}

	var deadSessions []DeadSession

	for _, issue := range inProgressIssues {
		// Check if agent reported Phase: Complete
		// If yes, this is waiting for orchestrator review, not dead
		hasComplete, err := HasPhaseComplete(issue.ID)
		if err != nil && config.Verbose {
			fmt.Printf("  DEBUG: Warning: failed to check Phase: Complete for %s: %v\n", issue.ID, err)
		}
		if hasComplete {
			if config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (Phase: Complete found)\n", issue.ID)
			}
			continue
		}

		// Check if there's an active OpenCode session
		// If yes, the agent is still working, not dead
		if HasExistingSessionForBeadsID(issue.ID) {
			if config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (active session found)\n", issue.ID)
			}
			continue
		}

		// No active session AND no Phase: Complete → dead session
		reason := "Session died without completing (no active session, no Phase: Complete)"
		deadSessions = append(deadSessions, DeadSession{
			BeadsID: issue.ID,
			Title:   issue.Title,
			Reason:  reason,
		})

		if config.Verbose {
			fmt.Printf("  DEBUG: Detected dead session: %s\n", issue.ID)
		}
	}

	return deadSessions, nil
}

// MarkSessionAsDead adds a comment to the beads issue indicating the session died.
// This updates the issue status back to "open" so it can be respawned.
func MarkSessionAsDead(beadsID string, reason string) error {
	// Add comment explaining what happened
	comment := fmt.Sprintf("DEAD SESSION: %s\n\nThe spawned agent session died without completing. This can happen due to:\n- Agent crash or context exhaustion\n- OpenCode server restart\n- Manual session termination\n\nStatus reset to 'open' for respawning.", reason)

	if err := AddCommentToIssue(beadsID, comment); err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	// Reset status to "open" so daemon can respawn
	if err := UpdateIssueStatus(beadsID, "open"); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// AddCommentToIssue adds a comment to a beads issue.
// Uses the beads RPC client with auto-reconnect when available, falling back to CLI.
func AddCommentToIssue(beadsID, comment string) error {
	// Try to use the beads RPC client first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			// AddComment expects (id, author, text)
			// Use "daemon" as the author for automated comments
			if err := client.AddComment(beadsID, "daemon", comment); err == nil {
				return nil
			}
			// Fall through to CLI fallback on error
		}
	}

	// Fallback to CLI if daemon unavailable
	cmd := exec.Command("bd", "comments", "add", beadsID, comment)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add comment: %w: %s", err, string(output))
	}
	return nil
}

// UpdateIssueStatus updates the status of a beads issue.
// Uses the beads RPC client with auto-reconnect when available, falling back to CLI.
func UpdateIssueStatus(beadsID, status string) error {
	// Try to use the beads RPC client first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			// Update expects UpdateArgs with Status field as pointer
			args := &beads.UpdateArgs{
				ID:     beadsID,
				Status: &status,
			}
			if _, err := client.Update(args); err == nil {
				return nil
			}
			// Fall through to CLI fallback on error
		}
	}

	// Fallback to CLI if daemon unavailable
	cmd := exec.Command("bd", "update", beadsID, "--status", status)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update status: %w: %s", err, string(output))
	}
	return nil
}

// ListIssuesWithStatus returns all issues with the given status.
// Uses beads RPC client with auto-reconnect when available, falling back to CLI.
func ListIssuesWithStatus(status string) ([]Issue, error) {
	// Try to use the beads RPC client first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			// Use List with Status filter
			beadsIssues, err := client.List(&beads.ListArgs{
				Status: status,
				Limit:  0, // Get all issues
			})
			if err == nil {
				return convertBeadsIssues(beadsIssues), nil
			}
			// Fall through to CLI fallback on error
		}
	}

	// Fallback to CLI if daemon unavailable
	return listIssuesWithStatusCLI(status)
}

// listIssuesWithStatusCLI retrieves issues by status using bd CLI.
func listIssuesWithStatusCLI(status string) ([]Issue, error) {
	// Use bd list --status <status> --json
	// Note: bd list may not support --status filter in all versions
	// Try filtered query first, fall back to manual filtering
	cmd := exec.Command("bd", "list", "--status", status, "--json", "--limit", "0")
	output, err := cmd.CombinedOutput()

	// If --status flag not supported, fall back to unfiltered list + manual filter
	if err != nil && strings.Contains(string(output), "unknown flag") {
		return listAndFilter(status)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w: %s", err, string(output))
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	return issues, nil
}

// listAndFilter gets all issues and filters by status manually.
func listAndFilter(status string) ([]Issue, error) {
	cmd := exec.Command("bd", "list", "--json", "--limit", "0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w: %s", err, string(output))
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	// Filter by status (case-insensitive)
	var filtered []Issue
	for _, issue := range issues {
		if strings.EqualFold(issue.Status, status) {
			filtered = append(filtered, issue)
		}
	}

	return filtered, nil
}
