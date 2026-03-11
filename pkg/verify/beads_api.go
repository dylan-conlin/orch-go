// Package verify provides verification helpers for agent completion.
// This file contains the beads API wrapper functions that handle RPC-first
// with CLI fallback pattern for beads operations.
package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// Pre-compiled regex patterns for beads API operations
var (
	regexPhaseComment             = regexp.MustCompile(`(?i)Phase:\s*(\w+)(?:\s*[-–—]\s*(.*))?`)
	regexInvestigationPathComment = regexp.MustCompile(`(?i)investigation_path:\s*(.+)`)
	regexProbePathComment         = regexp.MustCompile(`(?i)probe_path:\s*(.+)`)
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
	Labels      []string  `json:"labels,omitempty"`
	CloseReason string    `json:"close_reason,omitempty"`
	Comments    []Comment `json:"comments"`
}

// PhaseStatus represents the current phase of an agent.
type PhaseStatus struct {
	Phase           string     // Current phase (e.g., "Complete", "Implementing", "Planning")
	Summary         string     // Optional summary from the phase comment
	Found           bool       // Whether a Phase: comment was found
	PhaseReportedAt *time.Time // When the latest phase comment was posted (nil if not parseable)
}

// GetComments retrieves comments for a beads issue.
// If projectDir is non-empty, uses that directory for beads operations.
func GetComments(beadsID, projectDir string) ([]Comment, error) {
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

	// Fallback to CLI
	return beads.FallbackComments(beadsID, projectDir)
}

// HasBeadsComment checks if a beads issue has any comments.
// Returns true if the issue has at least one comment, false otherwise.
// This is useful for detecting stalled sessions that never reported progress.
func HasBeadsComment(beadsID, projectDir string) (bool, error) {
	comments, err := GetComments(beadsID, projectDir)
	if err != nil {
		return false, err
	}
	return len(comments) > 0, nil
}

// ParsePhaseFromComments extracts the latest Phase status from comments.
// Looks for comments matching "Phase: <phase> - <summary>" pattern.
// Also captures the timestamp of when the phase was reported for stall detection.
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
			// Parse the comment timestamp for stall detection
			// Beads comments use RFC3339 format: "2026-01-08T10:30:00Z"
			if comment.CreatedAt != "" {
				if t, err := time.Parse(time.RFC3339, comment.CreatedAt); err == nil {
					latestPhase.PhaseReportedAt = &t
				}
			}
		}
	}

	return latestPhase
}

// ParseInvestigationPathFromComments extracts the investigation file path from comments.
// Looks for comments matching "investigation_path: <path>" pattern.
// Returns empty string if no investigation_path comment is found.
func ParseInvestigationPathFromComments(comments []Comment) string {
	var latestPath string

	for _, comment := range comments {
		matches := regexInvestigationPathComment.FindStringSubmatch(comment.Text)
		if len(matches) >= 2 {
			latestPath = strings.TrimSpace(matches[1])
		}
	}

	return latestPath
}

// ParseProbePathFromComments extracts the probe file path from comments.
// Looks for comments matching "probe_path: <path>" pattern.
// Returns empty string if no probe_path comment is found.
func ParseProbePathFromComments(comments []Comment) string {
	var latestPath string

	for _, comment := range comments {
		matches := regexProbePathComment.FindStringSubmatch(comment.Text)
		if len(matches) >= 2 {
			latestPath = strings.TrimSpace(matches[1])
		}
	}

	return latestPath
}

// CheckCrossRepoDeliverable checks if a reported probe or investigation path
// is outside the agent's project directory, indicating a cross-repo deliverable.
// Returns the path if cross-repo, empty string otherwise.
func CheckCrossRepoDeliverable(comments []Comment, projectDir string) string {
	probePath := ParseProbePathFromComments(comments)
	if probePath != "" && !strings.HasPrefix(probePath, projectDir+string(filepath.Separator)) {
		return probePath
	}

	invPath := ParseInvestigationPathFromComments(comments)
	if invPath != "" && !strings.HasPrefix(invPath, projectDir+string(filepath.Separator)) {
		return invPath
	}

	return ""
}

// GetPhaseStatus retrieves the current phase status for a beads issue.
func GetPhaseStatus(beadsID, projectDir string) (PhaseStatus, error) {
	comments, err := GetComments(beadsID, projectDir)
	if err != nil {
		return PhaseStatus{}, err
	}

	return ParsePhaseFromComments(comments), nil
}

// IsPhaseComplete returns true if the agent has reported "Phase: Complete".
func IsPhaseComplete(beadsID, projectDir string) (bool, error) {
	status, err := GetPhaseStatus(beadsID, projectDir)
	if err != nil {
		return false, err
	}

	if !status.Found {
		return false, nil
	}

	return strings.EqualFold(status.Phase, "Complete"), nil
}

// CloseIssue closes a beads issue with the given reason.
// If projectDir is non-empty, uses that directory for beads operations.
func CloseIssue(beadsID, reason, projectDir string) error {
	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath(projectDir)
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if projectDir != "" {
			opts = append(opts, beads.WithCwd(projectDir))
		}
		client := beads.NewClient(socketPath, opts...)
		if err := client.CloseIssue(beadsID, reason); err == nil {
			return nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	return beads.FallbackClose(beadsID, reason, projectDir)
}

// ForceCloseIssue closes a beads issue with --force, bypassing bd's Phase: Complete check.
// If projectDir is non-empty, uses that directory for beads operations.
func ForceCloseIssue(beadsID, reason, projectDir string) error {
	return beads.FallbackForceClose(beadsID, reason, projectDir)
}

// UpdateIssueStatus updates the status of a beads issue.
// If projectDir is non-empty, uses that directory for beads operations.
func UpdateIssueStatus(beadsID, status, projectDir string) error {
	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath(projectDir)
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if projectDir != "" {
			opts = append(opts, beads.WithCwd(projectDir))
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
	return beads.FallbackUpdate(beadsID, status, projectDir)
}

// UpdateIssueAssignee updates the assignee of a beads issue.
// If projectDir is non-empty, uses that directory for beads operations.
func UpdateIssueAssignee(beadsID, assignee, projectDir string) error {
	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath(projectDir)
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if projectDir != "" {
			opts = append(opts, beads.WithCwd(projectDir))
		}
		client := beads.NewClient(socketPath, opts...)
		assigneePtr := &assignee
		_, err := client.Update(&beads.UpdateArgs{
			ID:       beadsID,
			Assignee: assigneePtr,
		})
		if err == nil {
			return nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	return beads.FallbackUpdateAssignee(beadsID, assignee, projectDir)
}

// AddLabel adds a label to a beads issue.
// If projectDir is non-empty, uses that directory for beads operations.
func AddLabel(beadsID, label, projectDir string) error {
	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath(projectDir)
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if projectDir != "" {
			opts = append(opts, beads.WithCwd(projectDir))
		}
		client := beads.NewClient(socketPath, opts...)
		if connErr := client.Connect(); connErr == nil {
			defer client.Close()
			err := client.AddLabel(beadsID, label)
			if err == nil {
				return nil
			}
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	return beads.FallbackAddLabel(beadsID, label, projectDir)
}

// RemoveTriageReadyLabel removes the triage:ready label from a beads issue.
// If projectDir is non-empty, uses that directory for beads operations.
func RemoveTriageReadyLabel(beadsID, projectDir string) error {
	const triageReadyLabel = "triage:ready"

	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath(projectDir)
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if projectDir != "" {
			opts = append(opts, beads.WithCwd(projectDir))
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
	return beads.FallbackRemoveLabel(beadsID, triageReadyLabel, projectDir)
}

// RemoveTriageLabels removes all daemon-spawnable triage labels (triage:ready
// and triage:approved) from a beads issue. Used during manual spawn to prevent
// the daemon from picking up the same issue (race condition between manual
// spawn pipeline and daemon poll).
func RemoveTriageLabels(beadsID, projectDir string) {
	if err := RemoveTriageReadyLabel(beadsID, projectDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to remove triage:ready label from %s: %v\n", beadsID, err)
	}
	// Also remove triage:approved (daemon treats it as equivalent to triage:ready)
	if err := removeLabel(beadsID, "triage:approved", projectDir); err != nil {
		// Silently ignore - label may not exist
		_ = err
	}
}

// removeLabel removes a specific label from a beads issue.
func removeLabel(beadsID, label, projectDir string) error {
	socketPath, err := beads.FindSocketPath(projectDir)
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if projectDir != "" {
			opts = append(opts, beads.WithCwd(projectDir))
		}
		client := beads.NewClient(socketPath, opts...)
		if connErr := client.Connect(); connErr == nil {
			defer client.Close()
			if err := client.RemoveLabel(beadsID, label); err == nil {
				return nil
			}
		}
	}
	return beads.FallbackRemoveLabel(beadsID, label, projectDir)
}

// RemoveOrchAgentLabel removes the orch:agent label from a beads issue.
// If projectDir is non-empty, uses that directory for beads operations.
func RemoveOrchAgentLabel(beadsID, projectDir string) error {
	const orchAgentLabel = "orch:agent"

	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath(projectDir)
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if projectDir != "" {
			opts = append(opts, beads.WithCwd(projectDir))
		}
		client := beads.NewClient(socketPath, opts...)
		if connErr := client.Connect(); connErr == nil {
			defer client.Close()
			err := client.RemoveLabel(beadsID, orchAgentLabel)
			if err == nil {
				return nil
			}
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	return beads.FallbackRemoveLabel(beadsID, orchAgentLabel, projectDir)
}

// GetIssue retrieves issue details from beads.
// If projectDir is non-empty, uses that directory for beads operations.
func GetIssue(beadsID, projectDir string) (*Issue, error) {
	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath(projectDir)
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if projectDir != "" {
			opts = append(opts, beads.WithCwd(projectDir))
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
				Labels:      issue.Labels,
				CloseReason: issue.CloseReason,
				// Comments are not populated via Show() - use GetComments() if needed
			}, nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	issue, err := beads.FallbackShow(beadsID, projectDir)
	if err != nil {
		return nil, err
	}

	return &Issue{
		ID:          issue.ID,
		Title:       issue.Title,
		Description: issue.Description,
		Status:      issue.Status,
		IssueType:   issue.IssueType,
		Labels:      issue.Labels,
		CloseReason: issue.CloseReason,
	}, nil
}

// GetIssuesBatch retrieves multiple issues efficiently.
// projectDirs should contain beadsID -> projectDir mappings for cross-project lookups.
// Returns a map from beadsID to Issue. Missing/invalid IDs are silently skipped.
func GetIssuesBatch(beadsIDs []string, projectDirs map[string]string) (map[string]*Issue, error) {
	if len(beadsIDs) == 0 {
		return make(map[string]*Issue), nil
	}

	// Use mutex-protected map for thread-safe writes
	var mu sync.Mutex
	result := make(map[string]*Issue, len(beadsIDs))

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
		effectiveDir := projectDir

		// Try RPC client first with auto-reconnect
		socketPath, err := beads.FindSocketPath(effectiveDir)
		if err == nil {
			opts := []beads.Option{beads.WithAutoReconnect(3)}
			if effectiveDir != "" {
				opts = append(opts, beads.WithCwd(effectiveDir))
			}
			client := beads.NewClient(socketPath, opts...)

			// Fetch each issue via Show() which handles short ID resolution
			for _, id := range ids {
				wg.Add(1)
				go func(beadsID string, c *beads.Client) {
					defer wg.Done()
					sem <- struct{}{}        // Acquire semaphore
					defer func() { <-sem }() // Release semaphore

					issue, err := c.Show(beadsID)
					if err == nil && issue != nil {
						mu.Lock()
						// Store by the original ID passed in, so callers can find their result
						result[beadsID] = &Issue{
							ID:          issue.ID,
							Title:       issue.Title,
							Description: issue.Description,
							Status:      issue.Status,
							IssueType:   issue.IssueType,
							Labels:      issue.Labels,
							CloseReason: issue.CloseReason,
						}
						mu.Unlock()
					}
				}(id, client)
			}
		} else {
			// Fallback to CLI for this project dir in parallel
			for _, id := range ids {
				wg.Add(1)
				go func(beadsID string, dir string) {
					defer wg.Done()
					sem <- struct{}{}        // Acquire semaphore
					defer func() { <-sem }() // Release semaphore

					issue, err := beads.FallbackShow(beadsID, dir)
					if err == nil && issue != nil {
						mu.Lock()
						// Store by the original ID passed in, so callers can find their result
						result[beadsID] = &Issue{
							ID:          issue.ID,
							Title:       issue.Title,
							Description: issue.Description,
							Status:      issue.Status,
							IssueType:   issue.IssueType,
							Labels:      issue.Labels,
							CloseReason: issue.CloseReason,
						}
						mu.Unlock()
					}
				}(id, effectiveDir)
			}
		}
	}

	wg.Wait()
	return result, nil
}

// ListOpenIssues retrieves all open issues in a single call.
// If projectDir is non-empty, uses that directory for beads operations.
func ListOpenIssues(projectDir string) (map[string]*Issue, error) {
	result := make(map[string]*Issue)

	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath(projectDir)
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if projectDir != "" {
			opts = append(opts, beads.WithCwd(projectDir))
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
						Labels:      issues[i].Labels,
						CloseReason: issues[i].CloseReason,
					}
				}
			}
			return result, nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	issues, err := beads.FallbackList("", projectDir)
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
				Labels:      issues[i].Labels,
				CloseReason: issues[i].CloseReason,
			}
		}
	}

	return result, nil
}

// ListOpenIssuesWithDir retrieves all open issues scoped to a project directory.
// Deprecated: Use ListOpenIssues(projectDir) directly.
func ListOpenIssuesWithDir(projectDir string) (map[string]*Issue, error) {
	return ListOpenIssues(projectDir)
}

// GetCommentsBatch fetches comments for multiple issues in parallel.
// Returns a map from beadsID to comments. Errors are silently skipped.
// Uses goroutines with semaphore to parallelize fetching (much faster than sequential).
func GetCommentsBatch(beadsIDs []string, projectDirs map[string]string) map[string][]Comment {
	return GetCommentsBatchWithProjectDirs(beadsIDs, projectDirs)
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
		// Try RPC client first
		socketPath, err := beads.FindSocketPath(projectDir)
		if err == nil {
			opts := []beads.Option{beads.WithAutoReconnect(3)}
			if projectDir != "" {
				opts = append(opts, beads.WithCwd(projectDir))
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

					comments, err := beads.FallbackComments(id, dir)
					if err == nil {
						mu.Lock()
						commentMap[id] = comments
						mu.Unlock()
					}
				}(beadsID, projectDir)
			}
		}
	}

	wg.Wait()
	return commentMap
}
