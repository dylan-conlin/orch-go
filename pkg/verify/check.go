// Package verify provides verification helpers for agent completion.
package verify

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Comment represents a beads issue comment.
type Comment struct {
	ID        int64  `json:"id"`
	Content   string `json:"content"`
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
}

// Issue represents a beads issue with comments.
type Issue struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Status   string    `json:"status"`
	Comments []Comment `json:"comments"`
}

// PhaseStatus represents the current phase of an agent.
type PhaseStatus struct {
	Phase   string // Current phase (e.g., "Complete", "Implementing", "Planning")
	Summary string // Optional summary from the phase comment
	Found   bool   // Whether a Phase: comment was found
}

// GetComments retrieves comments for a beads issue using the bd CLI.
func GetComments(beadsID string) ([]Comment, error) {
	cmd := exec.Command("bd", "comments", beadsID, "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}

	// Handle null response (no comments)
	if strings.TrimSpace(string(output)) == "null" {
		return []Comment{}, nil
	}

	var comments []Comment
	if err := json.Unmarshal(output, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse comments: %w", err)
	}

	return comments, nil
}

// ParsePhaseFromComments extracts the latest Phase status from comments.
// Looks for comments matching "Phase: <phase> - <summary>" pattern.
func ParsePhaseFromComments(comments []Comment) PhaseStatus {
	// Pattern: "Phase: <phase>" optionally followed by " - <summary>"
	phasePattern := regexp.MustCompile(`(?i)Phase:\s*(\w+)(?:\s*[-–—]\s*(.*))?`)

	var latestPhase PhaseStatus

	for _, comment := range comments {
		matches := phasePattern.FindStringSubmatch(comment.Content)
		if len(matches) >= 2 {
			latestPhase = PhaseStatus{
				Phase: matches[1],
				Found: true,
			}
			if len(matches) >= 3 && matches[2] != "" {
				latestPhase.Summary = strings.TrimSpace(matches[2])
			}
		}
	}

	return latestPhase
}

// GetPhaseStatus retrieves the current phase status for a beads issue.
func GetPhaseStatus(beadsID string) (PhaseStatus, error) {
	comments, err := GetComments(beadsID)
	if err != nil {
		return PhaseStatus{}, err
	}

	return ParsePhaseFromComments(comments), nil
}

// IsPhaseComplete returns true if the agent has reported "Phase: Complete".
func IsPhaseComplete(beadsID string) (bool, error) {
	status, err := GetPhaseStatus(beadsID)
	if err != nil {
		return false, err
	}

	if !status.Found {
		return false, nil
	}

	return strings.EqualFold(status.Phase, "Complete"), nil
}

// VerificationResult represents the result of a completion verification.
type VerificationResult struct {
	Passed   bool     // Whether all checks passed
	Errors   []string // Errors that prevent completion
	Warnings []string // Warnings that don't block completion
	Phase    PhaseStatus
}

// VerifyCompletion checks if an agent is ready for completion.
// Returns a VerificationResult with any errors or warnings.
func VerifyCompletion(beadsID string) (VerificationResult, error) {
	result := VerificationResult{
		Passed: true,
	}

	// Get phase status
	status, err := GetPhaseStatus(beadsID)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("failed to get phase status: %v", err))
		return result, nil
	}

	result.Phase = status

	// Check if Phase: Complete was reported
	if !status.Found {
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("agent has not reported any Phase status for %s", beadsID))
		return result, nil
	}

	if !strings.EqualFold(status.Phase, "Complete") {
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("agent phase is '%s', not 'Complete' (beads: %s)", status.Phase, beadsID))
		return result, nil
	}

	return result, nil
}

// CloseIssue closes a beads issue with the given reason.
func CloseIssue(beadsID, reason string) error {
	args := []string{"close", beadsID}
	if reason != "" {
		args = append(args, "--reason", reason)
	}

	cmd := exec.Command("bd", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to close issue: %w: %s", err, string(output))
	}

	return nil
}

// UpdateIssueStatus updates the status of a beads issue.
func UpdateIssueStatus(beadsID, status string) error {
	args := []string{"update", beadsID, "--status", status}
	cmd := exec.Command("bd", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update issue status: %w: %s", err, string(output))
	}
	return nil
}

// GetIssue retrieves issue details from beads.
func GetIssue(beadsID string) (*Issue, error) {
	cmd := exec.Command("bd", "show", beadsID, "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	// bd show returns an array with one element
	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issue: %w", err)
	}

	if len(issues) == 0 {
		return nil, fmt.Errorf("issue not found: %s", beadsID)
	}

	return &issues[0], nil
}
