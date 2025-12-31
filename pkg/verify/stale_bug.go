// Package verify provides stale bug detection before spawning.
// This helps prevent wasted agent time investigating bugs that were already fixed.
package verify

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// RelatedCommit represents a git commit that may be related to an issue.
type RelatedCommit struct {
	Hash    string    // Short commit hash
	Subject string    // Commit subject line
	Author  string    // Commit author
	Date    time.Time // Commit date
}

// StaleBugResult represents the result of checking if a bug issue might be stale.
type StaleBugResult struct {
	IssueID        string          // The beads issue ID being checked
	IssueTitle     string          // The issue title (if available)
	Keywords       []string        // Keywords searched for
	RelatedCommits []RelatedCommit // Commits that may have fixed the bug
	CheckedSince   time.Time       // Time from which commits were checked
}

// IsPotentiallyStale returns true if there are commits that might have fixed the bug.
func (r *StaleBugResult) IsPotentiallyStale() bool {
	return len(r.RelatedCommits) > 0
}

// CheckStaleBug checks git history for commits that might have fixed a bug issue.
// It searches for:
// 1. Commits mentioning the issue ID
// 2. Commits mentioning keywords from the issue title/description
//
// Parameters:
//   - projectDir: The git repository directory
//   - issueID: The beads issue ID to search for
//   - keywords: Space-separated keywords to search for (extracted from issue title)
//   - since: Only check commits after this time (typically issue creation time)
//
// Returns a StaleBugResult with any matching commits found.
func CheckStaleBug(projectDir, issueID, keywords string, since time.Time) (*StaleBugResult, error) {
	result := &StaleBugResult{
		IssueID:      issueID,
		CheckedSince: since,
	}

	// If no search criteria, return empty result
	if issueID == "" && keywords == "" {
		return result, nil
	}

	// Build search patterns
	var searchPatterns []string
	if issueID != "" {
		searchPatterns = append(searchPatterns, issueID)
	}

	// Extract keywords if provided
	keywordList := strings.Fields(keywords)
	result.Keywords = keywordList

	// Search for commits containing the issue ID or keywords
	commits, err := searchCommits(projectDir, issueID, keywordList, since)
	if err != nil {
		return result, err
	}

	result.RelatedCommits = commits
	return result, nil
}

// searchCommits searches git log for commits matching the issue ID or keywords.
func searchCommits(projectDir string, issueID string, keywords []string, since time.Time) ([]RelatedCommit, error) {
	var commits []RelatedCommit

	// Format time for git --since flag
	sinceStr := since.Format(time.RFC3339)

	// Get all commits since the time
	cmd := exec.Command("git", "log", "--oneline", "--since="+sinceStr, "--format=%h|%s|%an|%aI")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// If git command fails (e.g., not a git repo), return empty result
		return commits, nil
	}

	// Parse commits and filter by search patterns
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}

		hash := parts[0]
		subject := parts[1]
		author := parts[2]
		dateStr := parts[3]

		// Check if commit matches any search pattern
		if matchesSearchCriteria(subject, issueID, keywords) {
			date, _ := time.Parse(time.RFC3339, dateStr)
			commits = append(commits, RelatedCommit{
				Hash:    hash,
				Subject: subject,
				Author:  author,
				Date:    date,
			})
		}
	}

	return commits, nil
}

// matchesSearchCriteria checks if a commit subject matches the issue ID or keywords.
func matchesSearchCriteria(subject, issueID string, keywords []string) bool {
	subjectLower := strings.ToLower(subject)

	// Check for issue ID match (case-insensitive)
	if issueID != "" && strings.Contains(subjectLower, strings.ToLower(issueID)) {
		return true
	}

	// Check for keyword matches - require at least 2 keywords to match
	// to reduce false positives
	if len(keywords) > 0 {
		matchCount := 0
		for _, keyword := range keywords {
			if len(keyword) >= 3 && strings.Contains(subjectLower, strings.ToLower(keyword)) {
				matchCount++
			}
		}
		// For single keyword, require exact match
		// For multiple keywords, require at least 2 matches
		if len(keywords) == 1 && matchCount == 1 {
			return true
		}
		if len(keywords) >= 2 && matchCount >= 2 {
			return true
		}
	}

	return false
}

// ExtractKeywordsFromTitle extracts meaningful keywords from an issue title.
// It filters out common stop words and short words.
func ExtractKeywordsFromTitle(title string) []string {
	// Common stop words to filter out
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "must": true, "can": true,
		"for": true, "of": true, "to": true, "in": true, "on": true,
		"at": true, "by": true, "with": true, "from": true, "into": true,
		"this": true, "that": true, "these": true, "those": true,
		"it": true, "its": true, "and": true, "or": true, "but": true,
		"not": true, "no": true, "if": true, "when": true, "where": true,
		"how": true, "what": true, "which": true, "who": true, "whom": true,
		"fix": true, "bug": true, "issue": true, "error": true, // Common issue words
		"some": true, "any": true, "all": true, "each": true,
	}

	var keywords []string
	words := strings.Fields(strings.ToLower(title))

	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,;:!?\"'()[]{}#@")

		// Skip short words and stop words
		if len(word) < 4 || stopWords[word] {
			continue
		}

		keywords = append(keywords, word)
	}

	return keywords
}

// FormatStaleBugWarning formats a warning message for potentially stale bugs.
// Used when spawning an agent for a bug that may have already been fixed.
func FormatStaleBugWarning(result *StaleBugResult) string {
	if result == nil || !result.IsPotentiallyStale() {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("⚠️  POTENTIALLY STALE BUG DETECTED\n")
	sb.WriteString("┌─────────────────────────────────────────────────────────────┐\n")
	sb.WriteString("│  This bug may have already been fixed. Found commits that   │\n")
	sb.WriteString("│  mention the issue ID or related keywords since creation.   │\n")
	sb.WriteString("└─────────────────────────────────────────────────────────────┘\n")

	sb.WriteString("\n")
	sb.WriteString("Related commits:\n")

	for i, commit := range result.RelatedCommits {
		if i >= 3 {
			sb.WriteString(fmt.Sprintf("  ... and %d more\n", len(result.RelatedCommits)-3))
			break
		}
		dateStr := ""
		if !commit.Date.IsZero() {
			dateStr = commit.Date.Format("2006-01-02")
		}
		sb.WriteString(fmt.Sprintf("  %s %s (%s)\n", commit.Hash, commit.Subject, dateStr))
	}

	sb.WriteString("\n")
	sb.WriteString("Consider verifying the bug still exists before spawning.\n")
	sb.WriteString("Use --skip-stale-check to bypass this warning.\n")

	return sb.String()
}

// CheckStaleBugForIssue is a convenience function that retrieves issue details
// and performs the stale bug check. It handles getting the issue creation time
// and title from beads.
func CheckStaleBugForIssue(projectDir, beadsID string) (*StaleBugResult, error) {
	// Get issue details via the existing GetIssue helper (which handles RPC/CLI fallback)
	issue, err := GetIssue(beadsID)
	if err != nil {
		// Can't get issue details - return empty result
		return &StaleBugResult{IssueID: beadsID}, nil
	}

	// Only check for bug-type issues
	if issue.IssueType != "" && issue.IssueType != "bug" {
		return &StaleBugResult{IssueID: beadsID}, nil
	}

	// We need to get the CreatedAt from the beads issue
	// For now, default to checking last 7 days if we can't get creation time
	since := time.Now().Add(-7 * 24 * time.Hour)

	// Try to get issue from beads with creation time via RPC
	if createdAt, err := getIssueCreatedAt(beadsID); err == nil && !createdAt.IsZero() {
		since = createdAt
	}

	// Extract keywords from issue title
	keywords := strings.Join(ExtractKeywordsFromTitle(issue.Title), " ")

	result, err := CheckStaleBug(projectDir, beadsID, keywords, since)
	if err != nil {
		return result, err
	}

	result.IssueTitle = issue.Title
	return result, nil
}

// getIssueCreatedAt retrieves the issue creation time from beads.
// Returns zero time if unable to retrieve.
func getIssueCreatedAt(beadsID string) (time.Time, error) {
	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err != nil {
		return time.Time{}, err
	}

	client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
	issue, err := client.Show(beadsID)
	if err != nil {
		return time.Time{}, err
	}

	if issue.CreatedAt == "" {
		return time.Time{}, nil
	}

	return time.Parse(time.RFC3339, issue.CreatedAt)
}
