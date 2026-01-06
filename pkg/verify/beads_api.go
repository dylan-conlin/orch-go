// Package verify provides verification helpers for agent completion.
// This file contains the beads API wrapper functions that handle RPC-first
// with CLI fallback pattern for beads operations.
package verify

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// Pre-compiled regex patterns for beads API operations
var (
	regexPhaseComment = regexp.MustCompile(`(?i)Phase:\s*(\w+)(?:\s*[-–—]\s*(.*))?`)
)

// Comment is an alias for beads.Comment for compatibility.
type Comment = beads.Comment

// Issue represents a beads issue with comments.
type Issue struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	IssueType   string    `json:"issue_type"`
	CloseReason string    `json:"close_reason,omitempty"`
	Comments    []Comment `json:"comments"`
}

// PhaseStatus represents the current phase of an agent.
type PhaseStatus struct {
	Phase   string // Current phase (e.g., "Complete", "Implementing", "Planning")
	Summary string // Optional summary from the phase comment
	Found   bool   // Whether a Phase: comment was found
}

// GetComments retrieves comments for a beads issue.
// It uses the beads RPC client when available, falling back to the bd CLI.
func GetComments(beadsID string) ([]Comment, error) {
	return GetCommentsWithDir(beadsID, "")
}

// GetCommentsWithDir retrieves comments for a beads issue from a specific project directory.
// This is used for cross-project agent visibility where the beads issue is in a different
// project than the current working directory.
// If projectDir is empty, uses beads.DefaultDir if set, otherwise the current working directory.
func GetCommentsWithDir(beadsID, projectDir string) ([]Comment, error) {
	// Use DefaultDir if projectDir is empty
	if projectDir == "" && beads.DefaultDir != "" {
		projectDir = beads.DefaultDir
	}

	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath(projectDir)
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if projectDir != "" {
			opts = append(opts, beads.WithCwd(projectDir))
		}
		client := beads.NewClient(socketPath, opts...)
		comments, err := client.Comments(beadsID)
		if err == nil {
			return comments, nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI with project directory
	return FallbackCommentsWithDir(beadsID, projectDir)
}

// FallbackCommentsWithDir retrieves comments via bd CLI in a specific directory.
func FallbackCommentsWithDir(beadsID, projectDir string) ([]Comment, error) {
	cmd := exec.Command("bd", "comments", beadsID, "--json")
	if projectDir != "" {
		cmd.Dir = projectDir
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd comments failed: %w", err)
	}

	var comments []Comment
	if err := json.Unmarshal(output, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse bd comments output: %w", err)
	}

	return comments, nil
}

// HasBeadsComment checks if a beads issue has any comments.
// Returns true if the issue has at least one comment, false otherwise.
// This is useful for detecting stalled sessions that never reported progress.
func HasBeadsComment(beadsID string) (bool, error) {
	comments, err := GetComments(beadsID)
	if err != nil {
		return false, err
	}
	return len(comments) > 0, nil
}

// ParsePhaseFromComments extracts the latest Phase status from comments.
// Looks for comments matching "Phase: <phase> - <summary>" pattern.
func ParsePhaseFromComments(comments []Comment) PhaseStatus {
	var latestPhase PhaseStatus

	for _, comment := range comments {
		matches := regexPhaseComment.FindStringSubmatch(comment.Text)
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

// CloseIssue closes a beads issue with the given reason.
// It uses the beads RPC client with auto-reconnect when available, falling back to the bd CLI.
// Uses beads.DefaultDir if set to ensure cross-project operations work correctly.
func CloseIssue(beadsID, reason string) error {
	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if beads.DefaultDir != "" {
			opts = append(opts, beads.WithCwd(beads.DefaultDir))
		}
		client := beads.NewClient(socketPath, opts...)
		if err := client.CloseIssue(beadsID, reason); err == nil {
			return nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	return beads.FallbackClose(beadsID, reason)
}

// UpdateIssueStatus updates the status of a beads issue.
// It uses the beads RPC client with auto-reconnect when available, falling back to the bd CLI.
// Uses beads.DefaultDir if set to ensure cross-project operations work correctly.
func UpdateIssueStatus(beadsID, status string) error {
	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if beads.DefaultDir != "" {
			opts = append(opts, beads.WithCwd(beads.DefaultDir))
		}
		client := beads.NewClient(socketPath, opts...)
		statusPtr := &status
		_, err := client.Update(&beads.UpdateArgs{
			ID:     beadsID,
			Status: statusPtr,
		})
		if err == nil {
			return nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	return beads.FallbackUpdate(beadsID, status)
}

// RemoveTriageReadyLabel removes the triage:ready label from a beads issue.
// It uses the beads RPC client with auto-reconnect when available, falling back to the bd CLI.
// This should be called after orch complete successfully closes the issue, not at spawn time.
// This ensures failed/abandoned agents leave issues in the ready queue for daemon retry.
// Uses beads.DefaultDir if set to ensure cross-project operations work correctly.
func RemoveTriageReadyLabel(beadsID string) error {
	const triageReadyLabel = "triage:ready"

	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if beads.DefaultDir != "" {
			opts = append(opts, beads.WithCwd(beads.DefaultDir))
		}
		client := beads.NewClient(socketPath, opts...)
		if connErr := client.Connect(); connErr == nil {
			defer client.Close()
			err := client.RemoveLabel(beadsID, triageReadyLabel)
			if err == nil {
				return nil
			}
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	return beads.FallbackRemoveLabel(beadsID, triageReadyLabel)
}

// GetIssue retrieves issue details from beads.
// It uses the beads RPC client with auto-reconnect when available, falling back to the bd CLI.
// Uses beads.DefaultDir if set to ensure cross-project operations work correctly.
func GetIssue(beadsID string) (*Issue, error) {
	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if beads.DefaultDir != "" {
			opts = append(opts, beads.WithCwd(beads.DefaultDir))
		}
		client := beads.NewClient(socketPath, opts...)
		issue, err := client.Show(beadsID)
		if err == nil {
			// Convert beads.Issue to verify.Issue
			return &Issue{
				ID:          issue.ID,
				Title:       issue.Title,
				Description: issue.Description,
				Status:      issue.Status,
				IssueType:   issue.IssueType,
				CloseReason: issue.CloseReason,
				// Comments are not populated via Show() - use GetComments() if needed
			}, nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	issue, err := beads.FallbackShow(beadsID)
	if err != nil {
		return nil, err
	}

	return &Issue{
		ID:          issue.ID,
		Title:       issue.Title,
		Description: issue.Description,
		Status:      issue.Status,
		IssueType:   issue.IssueType,
		CloseReason: issue.CloseReason,
	}, nil
}

// GetIssuesBatch retrieves multiple issues efficiently.
// Returns a map from beadsID to Issue. Missing/invalid IDs are silently skipped.
// Uses individual Show() calls which handle short ID resolution (e.g., '51jz' -> 'orch-go-51jz').
// This includes closed issues, unlike List(nil) which only returns open issues.
// Uses beads.DefaultDir if set to ensure cross-project operations work correctly.
func GetIssuesBatch(beadsIDs []string) (map[string]*Issue, error) {
	if len(beadsIDs) == 0 {
		return make(map[string]*Issue), nil
	}

	// Use mutex-protected map for thread-safe writes
	var mu sync.Mutex
	result := make(map[string]*Issue, len(beadsIDs))

	// Limit concurrent RPC calls to avoid overwhelming the server
	const maxConcurrent = 20
	sem := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup

	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if beads.DefaultDir != "" {
			opts = append(opts, beads.WithCwd(beads.DefaultDir))
		}
		client := beads.NewClient(socketPath, opts...)

		// Fetch each issue via Show() which handles short ID resolution
		for _, beadsID := range beadsIDs {
			wg.Add(1)
			go func(id string, c *beads.Client) {
				defer wg.Done()
				sem <- struct{}{}        // Acquire semaphore
				defer func() { <-sem }() // Release semaphore

				issue, err := c.Show(id)
				if err == nil && issue != nil {
					mu.Lock()
					// Store by the original ID passed in, so callers can find their result
					result[id] = &Issue{
						ID:          issue.ID,
						Title:       issue.Title,
						Description: issue.Description,
						Status:      issue.Status,
						IssueType:   issue.IssueType,
						CloseReason: issue.CloseReason,
					}
					mu.Unlock()
				}
			}(beadsID, client)
		}

		wg.Wait()
		if len(result) > 0 {
			return result, nil
		}
		// Fall through to CLI if RPC returned no results
	}

	// Fallback to CLI - fetch each issue via bd show which handles short ID resolution
	for _, beadsID := range beadsIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			issue, err := beads.FallbackShow(id)
			if err == nil && issue != nil {
				mu.Lock()
				// Store by the original ID passed in, so callers can find their result
				result[id] = &Issue{
					ID:          issue.ID,
					Title:       issue.Title,
					Description: issue.Description,
					Status:      issue.Status,
					IssueType:   issue.IssueType,
					CloseReason: issue.CloseReason,
				}
				mu.Unlock()
			}
		}(beadsID)
	}

	wg.Wait()
	return result, nil
}

// ListOpenIssues retrieves all open issues in a single call.
// Returns a map from beadsID to Issue.
// It uses the beads RPC client with auto-reconnect when available, falling back to the bd CLI.
// Uses beads.DefaultDir if set to ensure cross-project operations work correctly.
func ListOpenIssues() (map[string]*Issue, error) {
	result := make(map[string]*Issue)

	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if beads.DefaultDir != "" {
			opts = append(opts, beads.WithCwd(beads.DefaultDir))
		}
		client := beads.NewClient(socketPath, opts...)

		// List all issues via RPC
		issues, err := client.List(nil)
		if err == nil {
			// Filter to open/in_progress/blocked statuses
			for i := range issues {
				status := strings.ToLower(issues[i].Status)
				if status == "open" || status == "in_progress" || status == "blocked" {
					result[issues[i].ID] = &Issue{
						ID:          issues[i].ID,
						Title:       issues[i].Title,
						Description: issues[i].Description,
						Status:      issues[i].Status,
						IssueType:   issues[i].IssueType,
						CloseReason: issues[i].CloseReason,
					}
				}
			}
			return result, nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	issues, err := beads.FallbackList("")
	if err != nil {
		return nil, err
	}

	// Filter to open/in_progress/blocked statuses
	for i := range issues {
		status := strings.ToLower(issues[i].Status)
		if status == "open" || status == "in_progress" || status == "blocked" {
			result[issues[i].ID] = &Issue{
				ID:          issues[i].ID,
				Title:       issues[i].Title,
				Description: issues[i].Description,
				Status:      issues[i].Status,
				IssueType:   issues[i].IssueType,
				CloseReason: issues[i].CloseReason,
			}
		}
	}

	return result, nil
}

// GetCommentsBatch fetches comments for multiple issues in parallel.
// Returns a map from beadsID to comments. Errors are silently skipped.
// Uses goroutines with semaphore to parallelize fetching (much faster than sequential).
// Uses beads.DefaultDir if set to ensure cross-project operations work correctly.
func GetCommentsBatch(beadsIDs []string) map[string][]Comment {
	// Delegate to the parallel implementation with empty projectDirs
	// All issues will use the default directory (current working directory or beads.DefaultDir)
	return GetCommentsBatchWithProjectDirs(beadsIDs, nil)
}

// GetCommentsBatchWithProjectDirs fetches comments for multiple issues in parallel.
// The projectDirs map should contain beadsID -> projectDir mappings.
// For beads IDs not in projectDirs, the current working directory is used.
// Returns a map from beadsID to comments. Errors are silently skipped.
// This is used for cross-project agent visibility where agents may be from different projects.
// Uses goroutines with semaphore to parallelize fetching (much faster than sequential).
func GetCommentsBatchWithProjectDirs(beadsIDs []string, projectDirs map[string]string) map[string][]Comment {
	if len(beadsIDs) == 0 {
		return make(map[string][]Comment)
	}

	// Use mutex-protected map for thread-safe writes
	var mu sync.Mutex
	commentMap := make(map[string][]Comment, len(beadsIDs))

	// Group beads IDs by project directory for efficient RPC client reuse
	byProjectDir := make(map[string][]string)
	for _, beadsID := range beadsIDs {
		dir := projectDirs[beadsID]
		byProjectDir[dir] = append(byProjectDir[dir], beadsID)
	}

	// Limit concurrent RPC calls to avoid overwhelming the server
	const maxConcurrent = 20
	sem := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup

	// Process each project directory group in parallel
	for projectDir, ids := range byProjectDir {
		// Determine effective directory (use DefaultDir if projectDir is empty)
		effectiveDir := projectDir
		if effectiveDir == "" && beads.DefaultDir != "" {
			effectiveDir = beads.DefaultDir
		}

		// Try RPC client first
		socketPath, err := beads.FindSocketPath(effectiveDir)
		if err == nil {
			opts := []beads.Option{beads.WithAutoReconnect(3)}
			if effectiveDir != "" {
				opts = append(opts, beads.WithCwd(effectiveDir))
			}
			client := beads.NewClient(socketPath, opts...)

			// Fetch comments in parallel via RPC
			for _, beadsID := range ids {
				wg.Add(1)
				go func(id string, c *beads.Client) {
					defer wg.Done()
					sem <- struct{}{}        // Acquire semaphore
					defer func() { <-sem }() // Release semaphore

					comments, err := c.Comments(id)
					if err == nil {
						mu.Lock()
						commentMap[id] = comments
						mu.Unlock()
					}
				}(beadsID, client)
			}
		} else {
			// Fallback to CLI for this project dir in parallel
			for _, beadsID := range ids {
				wg.Add(1)
				go func(id string, dir string) {
					defer wg.Done()
					sem <- struct{}{}        // Acquire semaphore
					defer func() { <-sem }() // Release semaphore

					comments, err := FallbackCommentsWithDir(id, dir)
					if err == nil {
						mu.Lock()
						commentMap[id] = comments
						mu.Unlock()
					}
				}(beadsID, effectiveDir)
			}
		}
	}

	wg.Wait()
	return commentMap
}
