// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
)

// DefaultMaxDeadSessionRetries is the default number of times a dead session
// can be reset to open before escalating to needs:human.
const DefaultMaxDeadSessionRetries = 2

// DeadSessionDetectionConfig holds configuration for dead session detection.
type DeadSessionDetectionConfig struct {
	// Verbose enables debug logging.
	Verbose bool

	// MaxRetries is the maximum number of times a dead session can be reset
	// to open before escalating to needs:human. 0 means use DefaultMaxDeadSessionRetries.
	MaxRetries int
}

// DefaultDeadSessionDetectionConfig returns default configuration.
func DefaultDeadSessionDetectionConfig() DeadSessionDetectionConfig {
	return DeadSessionDetectionConfig{
		Verbose:    false,
		MaxRetries: DefaultMaxDeadSessionRetries,
	}
}

// maxRetries returns the effective max retries, using the default if not set.
func (c DeadSessionDetectionConfig) maxRetries() int {
	if c.MaxRetries <= 0 {
		return DefaultMaxDeadSessionRetries
	}
	return c.MaxRetries
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

// CountDeadSessionComments counts how many "DEAD SESSION:" comments exist on an issue.
// The comments themselves are the source of truth for retry count - no external state needed.
func CountDeadSessionComments(beadsID string) (int, error) {
	comments, err := getIssueComments(beadsID)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, c := range comments {
		if strings.HasPrefix(c.Text, "DEAD SESSION:") {
			count++
		}
	}
	return count, nil
}

// GetLastPhaseComment extracts the last "Phase:" comment from an issue.
// Returns empty string if no phase comment found.
func GetLastPhaseComment(beadsID string) string {
	comments, err := getIssueComments(beadsID)
	if err != nil {
		return ""
	}
	for i := len(comments) - 1; i >= 0; i-- {
		text := comments[i].Text
		if strings.Contains(text, "Phase:") && !strings.HasPrefix(text, "DEAD SESSION:") {
			return text
		}
	}
	return ""
}

// getIssueComments retrieves all comments for a beads issue.
func getIssueComments(beadsID string) ([]beads.Comment, error) {
	var comments []beads.Comment
	err := beads.Do("", func(client *beads.Client) error {
		if connErr := client.Connect(); connErr != nil {
			return connErr
		}
		defer client.Close()

		var rpcErr error
		comments, rpcErr = client.Comments(beadsID)
		return rpcErr
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return comments, nil
	}

	cmd := exec.Command("bd", "--sandbox", "--quiet", "comments", beadsID, "--json")
	output, err := bdCombinedOutput(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w: %s", err, string(output))
	}
	comments = nil
	if err := json.Unmarshal(output, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse comments: %w", err)
	}
	return comments, nil
}

// MarkSessionAsDead adds a comment to the beads issue indicating the session died.
// Includes the last phase comment from the dying agent for context preservation.
// Also logs a crash event to agentlog for dashboard visibility.
func MarkSessionAsDead(beadsID string, reason string) error {
	lastPhase := GetLastPhaseComment(beadsID)

	// Log crash event to agentlog for dashboard visibility
	logger := events.NewDefaultLogger()
	if err := logger.LogSessionDied(beadsID, reason, lastPhase); err != nil {
		// Log error but don't fail - comment and status update are more critical
		fmt.Fprintf(os.Stderr, "Warning: failed to log crash event: %v\n", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("DEAD SESSION: %s\n\n", reason))
	sb.WriteString("The spawned agent session died without completing. This can happen due to:\n")
	sb.WriteString("- Agent crash or context exhaustion\n")
	sb.WriteString("- OpenCode server restart\n")
	sb.WriteString("- Manual session termination\n")
	if lastPhase != "" {
		sb.WriteString(fmt.Sprintf("\nLast agent progress:\n%s\n", lastPhase))
	}
	sb.WriteString("\nStatus reset to 'open' for respawning.")

	if err := AddCommentToIssue(beadsID, sb.String()); err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	if err := UpdateIssueStatus(beadsID, "open"); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// EscalateDeadSession marks an issue as needing human intervention after
// exceeding the retry threshold. Labels with needs:human to stop daemon respawning.
func EscalateDeadSession(beadsID string, retryCount int, reason string) error {
	lastPhase := GetLastPhaseComment(beadsID)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("DEAD SESSION ESCALATED: %s\n\n", reason))
	sb.WriteString(fmt.Sprintf("This issue has died %d time(s) without completing.\n", retryCount))
	sb.WriteString("Retry limit reached - escalating to human review.\n")
	sb.WriteString("The daemon will NOT respawn this issue automatically.\n")
	if lastPhase != "" {
		sb.WriteString(fmt.Sprintf("\nLast agent progress:\n%s\n", lastPhase))
	}
	sb.WriteString("\nTo retry manually: bd update <id> --status=open && bd label <id> triage:ready")

	if err := AddCommentToIssue(beadsID, sb.String()); err != nil {
		return fmt.Errorf("failed to add escalation comment: %w", err)
	}

	if err := addNeedsHumanLabel(beadsID); err != nil {
		return fmt.Errorf("failed to add needs:human label: %w", err)
	}

	if err := UpdateIssueStatus(beadsID, "open"); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// AddCommentToIssue adds a comment to a beads issue.
// Uses the beads RPC client with auto-reconnect when available, falling back to CLI.
func AddCommentToIssue(beadsID, comment string) error {
	err := beads.Do("", func(client *beads.Client) error {
		if connErr := client.Connect(); connErr != nil {
			return connErr
		}
		defer client.Close()
		// AddComment expects (id, author, text)
		// Use "daemon" as the author for automated comments
		return client.AddComment(beadsID, "daemon", comment)
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return nil
	}

	// Fallback to CLI if daemon unavailable
	cmd := exec.Command("bd", "--sandbox", "--quiet", "comments", "add", beadsID, comment)
	output, err := bdCombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w: %s", err, string(output))
	}
	return nil
}

// UpdateIssueStatus updates the status of a beads issue.
// Uses the beads RPC client with auto-reconnect when available, falling back to CLI.
func UpdateIssueStatus(beadsID, status string) error {
	err := beads.Do("", func(client *beads.Client) error {
		if connErr := client.Connect(); connErr != nil {
			return connErr
		}
		defer client.Close()

		// Update expects UpdateArgs with Status field as pointer
		args := &beads.UpdateArgs{
			ID:     beadsID,
			Status: &status,
		}
		_, rpcErr := client.Update(args)
		return rpcErr
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return nil
	}

	// Fallback to CLI if daemon unavailable
	cmd := exec.Command("bd", "--sandbox", "--quiet", "update", beadsID, "--status", status)
	output, err := bdCombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("failed to update status: %w: %s", err, string(output))
	}
	return nil
}

// ListIssuesWithStatus returns all issues with the given status.
// Uses beads RPC client with auto-reconnect when available, falling back to CLI.
func ListIssuesWithStatus(status string) ([]Issue, error) {
	var issues []Issue
	err := beads.Do("", func(client *beads.Client) error {
		if connErr := client.Connect(); connErr != nil {
			return connErr
		}
		defer client.Close()

		// Use List with Status filter
		beadsIssues, rpcErr := client.List(&beads.ListArgs{
			Status: status,
			Limit:  0, // Get all issues
		})
		if rpcErr != nil {
			return rpcErr
		}
		issues = convertBeadsIssues(beadsIssues)
		return nil
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return issues, nil
	}

	// Fallback to CLI if daemon unavailable
	return listIssuesWithStatusCLI(status)
}

// listIssuesWithStatusCLI retrieves issues by status using bd CLI.
func listIssuesWithStatusCLI(status string) ([]Issue, error) {
	// Use bd list --status <status> --json
	// Note: bd list may not support --status filter in all versions
	// Try filtered query first, fall back to manual filtering
	cmd := exec.Command("bd", "--sandbox", "--quiet", "list", "--status", status, "--json", "--limit", "0")
	output, err := bdCombinedOutput(cmd)

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
	cmd := exec.Command("bd", "--sandbox", "--quiet", "list", "--json", "--limit", "0")
	output, err := bdCombinedOutput(cmd)
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
