// Package main provides the CLI entry point for orch-go.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/spf13/cobra"
)

var (
	// Sync command flags
	syncDryRun  bool
	syncVerbose bool
	syncDays    int
	syncJSON    bool
)

// SyncResult represents the result of a sync operation.
type SyncResult struct {
	IssueID   string `json:"issue_id"`
	Title     string `json:"title"`
	CommitRef string `json:"commit_ref"`
	Closed    bool   `json:"closed"`
	Error     string `json:"error,omitempty"`
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Auto-close beads issues mentioned in recent commits",
	Long: `Scan recent git commits for beads issue references and auto-close matching issues.

This command parses commit messages looking for beads issue IDs (e.g., orch-go-abc123,
project-xyz) and closes any matching open issues with a reference to the commit.

The purpose is to prevent orphaned issues that stay open after fixes are committed,
which would otherwise cause the daemon to respawn work that's already done.

Issue ID patterns detected:
  - Full IDs: project-name-hash (e.g., orch-go-f9l5, kb-cli-abc1)
  - Short refs in conventional commits: fix(issue-id): message
  - Explicit markers: closes #id, fixes #id, resolves #id

Examples:
  orch sync                     # Check last 7 days of commits
  orch sync --days 30           # Check last 30 days
  orch sync --dry-run           # Preview what would be closed
  orch sync --verbose           # Show detailed output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSync()
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "Preview what would be closed without closing")
	syncCmd.Flags().BoolVarP(&syncVerbose, "verbose", "v", false, "Show detailed output")
	syncCmd.Flags().IntVar(&syncDays, "days", 7, "Number of days of commit history to scan")
	syncCmd.Flags().BoolVar(&syncJSON, "json", false, "Output results as JSON")

	rootCmd.AddCommand(syncCmd)
}

func runSync() error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Get beads client
	client, err := getBeadsClient()
	if err != nil {
		return fmt.Errorf("failed to get beads client: %w", err)
	}

	// Get all open and in_progress issues
	openIssues, err := client.List(&beads.ListArgs{Status: "open"})
	if err != nil {
		return fmt.Errorf("failed to list open issues: %w", err)
	}

	inProgressIssues, err := client.List(&beads.ListArgs{Status: "in_progress"})
	if err != nil {
		return fmt.Errorf("failed to list in_progress issues: %w", err)
	}

	allIssues := append(openIssues, inProgressIssues...)
	if len(allIssues) == 0 {
		if syncVerbose {
			fmt.Println("No open or in_progress issues to sync")
		}
		return nil
	}

	// Build map of issue IDs to issues
	issueMap := make(map[string]beads.Issue)
	for _, issue := range allIssues {
		issueMap[issue.ID] = issue
		// Also map by short ID (last 4 chars)
		if len(issue.ID) >= 4 {
			shortID := issue.ID[len(issue.ID)-4:]
			// Only add short ID if it doesn't collide
			if _, exists := issueMap[shortID]; !exists {
				issueMap[shortID] = issue
			}
		}
	}

	// Get commits from the last N days
	commits, err := getRecentCommits(projectDir, syncDays)
	if err != nil {
		return fmt.Errorf("failed to get commits: %w", err)
	}

	if syncVerbose {
		fmt.Printf("Scanning %d commits from last %d days...\n", len(commits), syncDays)
		fmt.Printf("Checking against %d open/in_progress issues...\n", len(allIssues))
	}

	// Find issue references in commits
	var results []SyncResult
	closedIDs := make(map[string]bool) // Track already closed to avoid duplicates

	for _, commit := range commits {
		refs := extractIssueRefs(commit.Subject, issueMap)
		for _, issueID := range refs {
			if closedIDs[issueID] {
				continue // Already processed
			}

			issue, exists := issueMap[issueID]
			if !exists {
				continue
			}

			result := SyncResult{
				IssueID:   issue.ID,
				Title:     issue.Title,
				CommitRef: commit.Hash,
			}

			if syncDryRun {
				result.Closed = false
				if syncVerbose {
					fmt.Printf("Would close: %s (%s) - referenced in %s\n", issue.ID, issue.Title, commit.Hash)
				}
			} else {
				reason := fmt.Sprintf("Auto-closed: referenced in commit %s (%s)", commit.Hash, truncateSyncString(commit.Subject, 50))
				if err := client.CloseIssue(issue.ID, reason); err != nil {
					result.Error = err.Error()
					if syncVerbose {
						fmt.Printf("Failed to close %s: %v\n", issue.ID, err)
					}
				} else {
					result.Closed = true
					closedIDs[issue.ID] = true
					if syncVerbose {
						fmt.Printf("Closed: %s (%s) - referenced in %s\n", issue.ID, issue.Title, commit.Hash)
					}
				}
			}

			results = append(results, result)
		}
	}

	// Output results
	if syncJSON {
		return outputJSON(results)
	}

	if len(results) == 0 {
		fmt.Println("No issues found referenced in recent commits")
		return nil
	}

	// Summary
	closedCount := 0
	for _, r := range results {
		if r.Closed {
			closedCount++
		}
	}

	if syncDryRun {
		fmt.Printf("\n[DRY-RUN] Would close %d issue(s)\n", len(results))
	} else {
		fmt.Printf("\nClosed %d issue(s)\n", closedCount)
	}

	return nil
}

// syncCommit holds basic commit information for sync operations.
type syncCommit struct {
	Hash    string
	Subject string
	Date    time.Time
}

// getRecentCommits retrieves commits from the last N days.
func getRecentCommits(projectDir string, days int) ([]syncCommit, error) {
	sinceDate := time.Now().AddDate(0, 0, -days).Format(time.RFC3339)

	cmd := exec.Command("git", "log", "--oneline", "--since="+sinceDate, "--format=%h|%s|%aI")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	var commits []syncCommit
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			continue
		}

		date, _ := time.Parse(time.RFC3339, parts[2])
		commits = append(commits, syncCommit{
			Hash:    parts[0],
			Subject: parts[1],
			Date:    date,
		})
	}

	return commits, nil
}

// extractIssueRefs extracts beads issue references from a commit message.
// It only matches commits that explicitly indicate issue completion:
// 1. Explicit close markers: closes #id, fixes #id, resolves #id
// 2. Fix-type commits mentioning issue IDs: "fix: ... issue-id" or "fix(issue-id): ..."
//
// This excludes commits that merely mention issues without fixing them,
// such as "Create epic orch-go-xyz" which references but doesn't close the issue.
func extractIssueRefs(message string, issueMap map[string]beads.Issue) []string {
	var refs []string
	seen := make(map[string]bool)
	lowerMessage := strings.ToLower(message)

	// Exclusion pattern: commits that create/update issues shouldn't close them
	createPattern := regexp.MustCompile(`(?i)^(feat|chore):\s*(create|add|update|edit)\s+(epic|issue|task)`)
	if createPattern.MatchString(message) {
		return nil
	}

	// Pattern 1: Explicit close markers (closes #id, fixes #id, resolves #id)
	// These are unambiguous intent to close
	closePattern := regexp.MustCompile(`(?i)(closes?|fixes?|resolves?)\s+#?([a-z0-9]+-[a-z0-9]+-[a-z0-9]+|[a-z0-9]{4,})`)
	closeMatches := closePattern.FindAllStringSubmatch(message, -1)
	for _, match := range closeMatches {
		id := strings.ToLower(match[2])
		if issue, exists := issueMap[id]; exists && !seen[issue.ID] {
			refs = append(refs, issue.ID)
			seen[issue.ID] = true
		}
	}

	// Pattern 2: Fix commits with issue ID in scope: fix(issue-id): message
	// The scope placement indicates the fix is specifically for that issue
	scopePattern := regexp.MustCompile(`^fix\(([a-z0-9]+-[a-z0-9]+-[a-z0-9]+|[a-z0-9]{4,})\):`)
	scopeMatches := scopePattern.FindAllStringSubmatch(lowerMessage, -1)
	for _, match := range scopeMatches {
		id := match[1]
		if issue, exists := issueMap[id]; exists && !seen[issue.ID] {
			refs = append(refs, issue.ID)
			seen[issue.ID] = true
		}
	}

	// Pattern 3: Fix commits with issue ID as suffix: fix: description issue-id
	// Common pattern where the issue ID appears at the end
	if strings.HasPrefix(lowerMessage, "fix:") || strings.HasPrefix(lowerMessage, "fix(") {
		fullIDPattern := regexp.MustCompile(`\b([a-z]+-[a-z]+-[a-z0-9]{4,})\b`)
		matches := fullIDPattern.FindAllStringSubmatch(lowerMessage, -1)
		for _, match := range matches {
			id := match[1]
			if _, exists := issueMap[id]; exists && !seen[id] {
				refs = append(refs, id)
				seen[id] = true
			}
		}
	}

	return refs
}

// getBeadsClient returns a beads client, preferring RPC over CLI.
func getBeadsClient() (beads.BeadsClient, error) {
	socketPath, err := beads.FindSocketPath("")
	if err != nil {
		// Fall back to CLI client
		return beads.NewCLIClient(), nil
	}

	client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
	if err := client.Connect(); err != nil {
		// Fall back to CLI client
		return beads.NewCLIClient(), nil
	}

	return client, nil
}

func truncateSyncString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func outputJSON(results []SyncResult) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
