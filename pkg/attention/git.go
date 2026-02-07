// Package attention provides attention signals for work graph monitoring.
// This includes detecting issues that may be complete but not yet closed.
package attention

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// LikelyDoneSignal represents an issue that appears to be complete
// based on git commit history but hasn't been formally closed.
type LikelyDoneSignal struct {
	IssueID      string   `json:"issue_id"`
	IssueTitle   string   `json:"issue_title"`
	IssueStatus  string   `json:"issue_status"`
	CommitCount  int      `json:"commit_count"`
	LastCommitAt string   `json:"last_commit_at"`
	CommitHashes []string `json:"commit_hashes"`
	Reason       string   `json:"reason"`
}

// LikelyDoneResponse is the response structure for the API endpoint.
type LikelyDoneResponse struct {
	Signals     []LikelyDoneSignal `json:"signals"`
	Total       int                `json:"total"`
	LastUpdated string             `json:"last_updated"`
	Error       string             `json:"error,omitempty"`
}

// LikelyDoneCache provides TTL-based caching for LIKELY_DONE signals.
// Git log scanning and workspace checks can be slow, so we cache with 5-minute TTL.
type LikelyDoneCache struct {
	mu sync.RWMutex

	data      *LikelyDoneResponse
	fetchedAt time.Time
	ttl       time.Duration
}

// NewLikelyDoneCache creates a new cache with 5-minute TTL.
func NewLikelyDoneCache() *LikelyDoneCache {
	return &LikelyDoneCache{
		ttl: 5 * time.Minute, // Git commits change slowly
	}
}

// Get returns cached data or fetches fresh if stale.
func (c *LikelyDoneCache) Get(projectDir string, client beads.BeadsClient) (*LikelyDoneResponse, error) {
	c.mu.RLock()
	if c.data != nil && time.Since(c.fetchedAt) < c.ttl {
		result := c.data
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	// Fetch fresh data
	data, err := CollectLikelyDoneSignals(projectDir, client)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.data = data
	c.fetchedAt = time.Now()
	c.mu.Unlock()

	return data, nil
}

// Invalidate clears the cache, forcing fresh fetch on next request.
func (c *LikelyDoneCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = nil
}

// commitInfo holds information about a commit.
type commitInfo struct {
	Hash      string
	Message   string
	Timestamp string
}

// issueIDPattern matches common issue ID formats in commit messages.
// Matches: orch-go-12345, og-12345, bd-a3f8, specs-platform-10, etc.
// Pattern breakdown: word boundary, letters, hyphen, letters/digits/hyphens, word boundary
var issueIDPattern = regexp.MustCompile(`\b([a-z]+(?:-[a-z0-9]+)+)\b`)

// CollectLikelyDoneSignals scans git commits for issue mentions and cross-references
// with open beads issues to identify work that appears complete but not closed.
func CollectLikelyDoneSignals(projectDir string, client beads.BeadsClient) (*LikelyDoneResponse, error) {
	response := &LikelyDoneResponse{
		Signals:     []LikelyDoneSignal{},
		Total:       0,
		LastUpdated: time.Now().Format(time.RFC3339),
	}

	// Get recent commits (last 30 days)
	commits, err := getRecentCommits(projectDir, 30)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent commits: %w", err)
	}

	// Extract issue IDs from commit messages
	issueCommits := extractIssueIDs(commits)
	if len(issueCommits) == 0 {
		return response, nil
	}

	// Get open issues from beads
	openIssues, err := client.List(&beads.ListArgs{
		Status: "open,in_progress",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list open issues: %w", err)
	}

	// Check for active workspaces
	activeWorkspaces, err := getActiveWorkspaces(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get active workspaces: %w", err)
	}

	// Build signals for issues with commits but no active workspace
	for _, issue := range openIssues {
		commits, hasCommits := issueCommits[issue.ID]
		if !hasCommits {
			continue
		}

		// Skip if issue has an active workspace
		if _, hasWorkspace := activeWorkspaces[issue.ID]; hasWorkspace {
			continue
		}

		// Get last commit timestamp
		lastCommitAt := ""
		if len(commits) > 0 {
			lastCommitAt = commits[0].Timestamp
		}

		// Build commit hashes list
		hashes := make([]string, 0, len(commits))
		for _, c := range commits {
			hashes = append(hashes, c.Hash)
		}

		signal := LikelyDoneSignal{
			IssueID:      issue.ID,
			IssueTitle:   issue.Title,
			IssueStatus:  issue.Status,
			CommitCount:  len(commits),
			LastCommitAt: lastCommitAt,
			CommitHashes: hashes,
			Reason:       fmt.Sprintf("%d commits found, no active workspace", len(commits)),
		}

		response.Signals = append(response.Signals, signal)
	}

	response.Total = len(response.Signals)
	return response, nil
}

// getRecentCommits fetches git commits from the last N days.
func getRecentCommits(projectDir string, days int) ([]commitInfo, error) {
	since := fmt.Sprintf("--since=%d.days.ago", days)

	cmd := exec.Command("git", "log", since, "--pretty=format:%H|%s|%ai")
	if projectDir != "" {
		cmd.Dir = projectDir
	}

	output, err := cmd.Output()
	if err != nil {
		// Return empty list if not a git repository or git command fails
		return []commitInfo{}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	commits := make([]commitInfo, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}

		commits = append(commits, commitInfo{
			Hash:      parts[0],
			Message:   parts[1],
			Timestamp: parts[2],
		})
	}

	return commits, nil
}

// extractIssueIDs extracts issue IDs from commit messages and groups by issue ID.
func extractIssueIDs(commits []commitInfo) map[string][]commitInfo {
	issueCommits := make(map[string][]commitInfo)

	for _, commit := range commits {
		matches := issueIDPattern.FindAllString(commit.Message, -1)
		for _, match := range matches {
			issueID := strings.ToLower(match)
			issueCommits[issueID] = append(issueCommits[issueID], commit)
		}
	}

	return issueCommits
}

// getActiveWorkspaces scans .orch/workspace for active workspaces and returns
// a map of issue IDs to workspace paths.
func getActiveWorkspaces(projectDir string) (map[string]string, error) {
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")

	// Check if workspace directory exists
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		return map[string]string{}, nil
	}

	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace directory: %w", err)
	}

	activeWorkspaces := make(map[string]string)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip archived directory
		if entry.Name() == "archived" {
			continue
		}

		dirPath := filepath.Join(workspaceDir, entry.Name())

		// Check for .beads_id file
		beadsIDPath := filepath.Join(dirPath, ".beads_id")
		beadsIDData, err := os.ReadFile(beadsIDPath)
		if err != nil {
			continue
		}

		beadsID := strings.TrimSpace(string(beadsIDData))
		if beadsID != "" {
			activeWorkspaces[beadsID] = dirPath
		}
	}

	return activeWorkspaces, nil
}
