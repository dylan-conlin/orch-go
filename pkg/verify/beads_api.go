// Package verify provides verification helpers for agent completion.
// This file contains the beads API wrapper functions that handle RPC-first
// with CLI fallback pattern for beads operations.
package verify

import (
	"fmt"
	"os"
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
// It uses the beads RPC client when available, falling back to the bd CLI.
func GetComments(beadsID string) ([]Comment, error) {
	return GetCommentsWithDir(beadsID, "")
}

// GetCommentsWithDir retrieves comments for a beads issue from a specific project directory.
// This is used for cross-project agent visibility where the beads issue is in a different
// project than the current working directory.
// If projectDir is empty, uses beads.DefaultDir if set, otherwise the current working directory.
func GetCommentsWithDir(beadsID, projectDir string) ([]Comment, error) {
	effectiveDir := projectDir
	if effectiveDir == "" && beads.DefaultDir != "" {
		effectiveDir = beads.DefaultDir
	}

	var comments []Comment
	err := beads.Do(effectiveDir, func(client *beads.Client) error {
		var rpcErr error
		comments, rpcErr = client.Comments(beadsID)
		return rpcErr
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return comments, nil
	}

	// Fallback to CLI with project directory
	return FallbackCommentsWithDir(beadsID, effectiveDir)
}

// FallbackCommentsWithDir retrieves comments via bd CLI in a specific directory.
// Sets BEADS_NO_DAEMON=1 to skip daemon connection attempts, avoiding 5s timeout
// in launchd/minimal environments.
func FallbackCommentsWithDir(beadsID, projectDir string) ([]Comment, error) {
	cliClient := beads.NewCLIClient(
		beads.WithWorkDir(projectDir),
		beads.WithEnv(append(os.Environ(), "BEADS_NO_DAEMON=1")),
	)

	comments, err := cliClient.Comments(beadsID)
	if err != nil {
		if beads.IsCLITimeout(err) {
			return nil, fmt.Errorf("bd comments timed out after %v", beads.DefaultCLITimeout)
		}
		return nil, fmt.Errorf("bd comments failed: %w", err)
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
// Also captures the timestamp of when the phase was reported for stall detection.
func ParsePhaseFromComments(comments []Comment) PhaseStatus {
	for _, comment := range comments {
		matches := regexPhaseComment.FindStringSubmatch(comment.Text)
		if len(matches) >= 2 {
			phaseStatus := PhaseStatus{
				Phase: matches[1],
				Found: true,
			}
			if len(matches) >= 3 && matches[2] != "" {
				phaseStatus.Summary = strings.TrimSpace(matches[2])
			}
			// Parse the comment timestamp for stall detection
			// Beads comments use RFC3339 format: "2026-01-08T10:30:00Z"
			if comment.CreatedAt != "" {
				if t, err := time.Parse(time.RFC3339, comment.CreatedAt); err == nil {
					phaseStatus.PhaseReportedAt = &t
				}
			}

			// Beads returns comments in reverse-chronological order (newest first),
			// so the first phase match is the current phase.
			return phaseStatus
		}
	}

	return PhaseStatus{}
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
	return CloseIssueForce(beadsID, reason, false)
}

// CloseIssueForce closes a beads issue with optional force flag.
// When force=true, passes --force to bd close to bypass Phase: Complete checks.
// It uses the beads RPC client with auto-reconnect when available, falling back to the bd CLI.
// Uses beads.DefaultDir if set to ensure cross-project operations work correctly.
func CloseIssueForce(beadsID, reason string, force bool) error {
	err := beads.Do("", func(client *beads.Client) error {
		return client.CloseIssueForce(beadsID, reason, force)
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return nil
	}

	// Fallback to CLI (with force support)
	return beads.FallbackCloseForce(beadsID, reason, force)
}

// UpdateIssueStatus updates the status of a beads issue.
// It uses the beads RPC client with auto-reconnect when available, falling back to the bd CLI.
// Uses beads.DefaultDir if set to ensure cross-project operations work correctly.
func UpdateIssueStatus(beadsID, status string) error {
	err := beads.Do("", func(client *beads.Client) error {
		statusPtr := &status
		_, rpcErr := client.Update(&beads.UpdateArgs{
			ID:     beadsID,
			Status: statusPtr,
		})
		return rpcErr
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return nil
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

	err := beads.Do("", func(client *beads.Client) error {
		if connErr := client.Connect(); connErr != nil {
			return connErr
		}
		defer client.Close()
		return client.RemoveLabel(beadsID, triageReadyLabel)
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return nil
	}

	// Fallback to CLI
	return beads.FallbackRemoveLabel(beadsID, triageReadyLabel)
}

// GetIssue retrieves issue details from beads.
// It uses the beads RPC client with auto-reconnect when available, falling back to the bd CLI.
// Uses beads.DefaultDir if set to ensure cross-project operations work correctly.
func GetIssue(beadsID string) (*Issue, error) {
	var rpcIssue *beads.Issue
	err := beads.Do("", func(client *beads.Client) error {
		var rpcErr error
		rpcIssue, rpcErr = client.Show(beadsID)
		return rpcErr
	}, beads.WithAutoReconnect(3))
	if err == nil {
		// Convert beads.Issue to verify.Issue
		return &Issue{
			ID:          rpcIssue.ID,
			Title:       rpcIssue.Title,
			Description: rpcIssue.Description,
			Status:      rpcIssue.Status,
			IssueType:   rpcIssue.IssueType,
			Labels:      rpcIssue.Labels,
			CloseReason: rpcIssue.CloseReason,
			// Comments are not populated via Show() - use GetComments() if needed
		}, nil
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
		Labels:      issue.Labels,
		CloseReason: issue.CloseReason,
	}, nil
}

// GetIssueWithDir retrieves a beads issue from a specific project directory.
// This is used for cross-project operations where the issue belongs to a different project
// than the current working directory.
func GetIssueWithDir(beadsID string, projectDir string) (*Issue, error) {
	var rpcIssue *beads.Issue
	err := beads.Do(projectDir, func(client *beads.Client) error {
		var rpcErr error
		rpcIssue, rpcErr = client.Show(beadsID)
		return rpcErr
	}, beads.WithAutoReconnect(3))
	if err == nil {
		// Convert beads.Issue to verify.Issue
		return &Issue{
			ID:          rpcIssue.ID,
			Title:       rpcIssue.Title,
			Description: rpcIssue.Description,
			Status:      rpcIssue.Status,
			IssueType:   rpcIssue.IssueType,
			Labels:      rpcIssue.Labels,
			CloseReason: rpcIssue.CloseReason,
		}, nil
	}

	// Fallback to CLI
	issue, err := beads.FallbackShowWithDir(beadsID, projectDir)
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
// Uses individual Show() calls which handle short ID resolution (e.g., '51jz' -> 'orch-go-51jz').
// This includes closed issues, unlike List(nil) which only returns open issues.
// Uses beads.DefaultDir if set to ensure cross-project operations work correctly.
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
		// Determine effective directory (use DefaultDir if projectDir is empty)
		effectiveDir := projectDir
		if effectiveDir == "" && beads.DefaultDir != "" {
			effectiveDir = beads.DefaultDir
		}

		var client *beads.Client
		err := beads.Do(effectiveDir, func(c *beads.Client) error {
			client = c
			return nil
		}, beads.WithAutoReconnect(3))
		if err == nil {
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

					issue, err := beads.FallbackShowWithDir(beadsID, dir)
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
// Returns a map from beadsID to Issue.
// It uses the beads RPC client with auto-reconnect when available, falling back to the bd CLI.
// Uses beads.DefaultDir if set to ensure cross-project operations work correctly.
func ListOpenIssues() (map[string]*Issue, error) {
	result := make(map[string]*Issue)

	err := beads.Do("", func(client *beads.Client) error {
		issues, rpcErr := client.List(nil)
		if rpcErr != nil {
			return rpcErr
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
		return nil
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return result, nil
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
				Labels:      issues[i].Labels,
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

		var client *beads.Client
		err := beads.Do(effectiveDir, func(c *beads.Client) error {
			client = c
			return nil
		}, beads.WithAutoReconnect(3))
		if err == nil {
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

// EpicChildInfo represents information about an epic's child issue.
type EpicChildInfo struct {
	ID     string
	Title  string
	Status string
}

// GetOpenEpicChildren retrieves any open (non-closed) children of an epic.
// Returns a slice of open children and any error encountered.
// If the issue is not an epic or has no children, returns empty slice.
// Uses beads.DefaultDir if set to ensure cross-project operations work correctly.
func GetOpenEpicChildren(beadsID string) ([]EpicChildInfo, error) {
	// Get all children of the epic (including closed)
	children, err := beads.FallbackListByParent(beadsID)
	if err != nil {
		return nil, fmt.Errorf("failed to list epic children: %w", err)
	}

	// Filter to only open children (not closed, deferred, or tombstone)
	var openChildren []EpicChildInfo
	for _, child := range children {
		status := strings.ToLower(child.Status)
		// Exclude closed, deferred, and tombstone statuses
		if status != "closed" && status != "deferred" && status != "tombstone" {
			openChildren = append(openChildren, EpicChildInfo{
				ID:     child.ID,
				Title:  child.Title,
				Status: child.Status,
			})
		}
	}

	return openChildren, nil
}

// ExtractParentID extracts the parent issue ID from a child issue ID.
// Child IDs follow the format: parentID.N (e.g., "orch-go-erdw.4" has parent "orch-go-erdw").
// Returns empty string if the issue ID doesn't appear to be a child (no dot separator).
func ExtractParentID(issueID string) string {
	// Find the last dot in the ID
	lastDotIdx := strings.LastIndex(issueID, ".")
	if lastDotIdx == -1 {
		return "" // No dot means not a child issue
	}

	// Check if what follows the dot is numeric (child number)
	suffix := issueID[lastDotIdx+1:]
	for _, c := range suffix {
		if c < '0' || c > '9' {
			return "" // Not numeric, so not a child ID pattern
		}
	}

	return issueID[:lastDotIdx]
}

// GetParentEpicInfo retrieves information about a child issue's parent epic.
// Returns nil if the issue has no parent or if the parent is not an epic.
// Also returns remaining open children count for the parent epic.
type ParentEpicInfo struct {
	ID               string
	Title            string
	Status           string
	IssueType        string
	OpenChildrenLeft int // Number of open children EXCLUDING the current issue
}

// GetParentEpicInfo retrieves information about a child issue's parent epic.
// The currentIssueID is used to exclude the current issue from the open children count.
// Returns nil if the issue has no parent or if the parent is not an epic.
func GetParentEpicInfo(currentIssueID string) (*ParentEpicInfo, error) {
	parentID := ExtractParentID(currentIssueID)
	if parentID == "" {
		return nil, nil // Not a child issue
	}

	// Get parent issue details
	parentIssue, err := GetIssue(parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent issue: %w", err)
	}

	// Only proceed if parent is an epic
	if parentIssue.IssueType != "epic" {
		return nil, nil
	}

	// Count remaining open children (excluding current issue)
	openChildren, err := GetOpenEpicChildren(parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic children: %w", err)
	}

	// Count children excluding the current one
	openCount := 0
	for _, child := range openChildren {
		if child.ID != currentIssueID {
			openCount++
		}
	}

	return &ParentEpicInfo{
		ID:               parentID,
		Title:            parentIssue.Title,
		Status:           parentIssue.Status,
		IssueType:        parentIssue.IssueType,
		OpenChildrenLeft: openCount,
	}, nil
}

// IsEpicClosed checks if an epic is closed (useful for pre-flight spawn checks).
func IsEpicClosed(epicID string) (bool, error) {
	issue, err := GetIssue(epicID)
	if err != nil {
		return false, fmt.Errorf("failed to get epic: %w", err)
	}

	status := strings.ToLower(issue.Status)
	return status == "closed" || status == "deferred" || status == "tombstone", nil
}
