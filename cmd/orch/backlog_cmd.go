package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/spf13/cobra"
)

var (
	backlogCullDays  int
	backlogCullClose bool
)

var backlogCmd = &cobra.Command{
	Use:   "backlog",
	Short: "Backlog maintenance commands",
	Long:  `Commands for maintaining the beads backlog (culling stale issues, etc).`,
}

var backlogCullCmd = &cobra.Command{
	Use:   "cull",
	Short: "Surface stale P3/P4 issues for keep-or-close decision",
	Long: `Find P3/P4 issues older than a threshold (default 14 days) with no recent activity.

By default, lists stale issues without making changes (dry-run).
Use --close to close all stale issues with a reason.

Examples:
  orch backlog cull                  # List stale P3/P4 issues (preview)
  orch backlog cull --days 7         # Use 7-day threshold
  orch backlog cull --close          # Close all stale issues`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBacklogCull()
	},
}

func init() {
	backlogCmd.AddCommand(backlogCullCmd)

	backlogCullCmd.Flags().IntVar(&backlogCullDays, "days", 14, "Age threshold in days for stale issues")
	backlogCullCmd.Flags().BoolVar(&backlogCullClose, "close", false, "Close all stale issues (default: preview only)")
}

// staleIssue wraps a beads issue with staleness metadata.
type staleIssue struct {
	Issue        beads.Issue
	AgeDays      int
	LastActivity time.Time
}

// filterStaleBacklogIssues finds P3/P4 issues older than the given threshold.
// Filters to open/in_progress status only. Returns sorted oldest-first.
func filterStaleBacklogIssues(issues []beads.Issue, thresholdDays int, now time.Time) []staleIssue {
	threshold := now.AddDate(0, 0, -thresholdDays)
	var result []staleIssue

	for _, issue := range issues {
		// Only P3 and P4
		if issue.Priority < 3 {
			continue
		}

		// Only open or in_progress
		if issue.Status != "open" && issue.Status != "in_progress" {
			continue
		}

		// Determine last activity time
		timestamp := issue.UpdatedAt
		if timestamp == "" {
			timestamp = issue.CreatedAt
		}
		if timestamp == "" {
			continue
		}

		lastActivity, err := time.Parse(time.RFC3339, timestamp)
		if err != nil {
			lastActivity, err = time.Parse("2006-01-02", timestamp)
			if err != nil {
				continue
			}
		}

		if lastActivity.After(threshold) {
			continue
		}

		ageDays := int(now.Sub(lastActivity).Hours() / 24)
		result = append(result, staleIssue{
			Issue:        issue,
			AgeDays:      ageDays,
			LastActivity: lastActivity,
		})
	}

	// Sort by age descending (oldest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].AgeDays > result[j].AgeDays
	})

	return result
}

func runBacklogCull() error {
	client := beads.NewCLIClient()

	// Query all open issues (P3 and P4 will be filtered in code)
	issues, err := client.List(&beads.ListArgs{
		Status: "open",
		Limit:  beads.IntPtr(0),
	})
	if err != nil {
		return fmt.Errorf("failed to list issues: %w", err)
	}

	// Also get in_progress issues
	inProgressIssues, err := client.List(&beads.ListArgs{
		Status: "in_progress",
		Limit:  beads.IntPtr(0),
	})
	if err != nil {
		return fmt.Errorf("failed to list in_progress issues: %w", err)
	}

	allIssues := append(issues, inProgressIssues...)

	now := time.Now()
	stale := filterStaleBacklogIssues(allIssues, backlogCullDays, now)

	if len(stale) == 0 {
		fmt.Printf("No stale P3/P4 issues older than %d days.\n", backlogCullDays)
		return nil
	}

	fmt.Printf("Found %d stale P3/P4 issues (older than %d days):\n\n", len(stale), backlogCullDays)

	for _, si := range stale {
		statusTag := ""
		if si.Issue.Status == "in_progress" {
			statusTag = " [in_progress]"
		}
		fmt.Printf("  %s [P%d] [%s]%s %dd old - %s\n",
			si.Issue.ID,
			si.Issue.Priority,
			si.Issue.IssueType,
			statusTag,
			si.AgeDays,
			si.Issue.Title,
		)
	}

	if !backlogCullClose {
		fmt.Printf("\nPreview only. Use --close to close these issues.\n")
		return nil
	}

	// Close stale issues
	fmt.Printf("\nClosing %d stale issues...\n", len(stale))

	closed := 0
	for _, si := range stale {
		reason := fmt.Sprintf("Backlog cull: P%d issue stale for %d days with no activity",
			si.Issue.Priority, si.AgeDays)

		if err := client.CloseIssue(si.Issue.ID, reason); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: failed to close %s: %v\n", si.Issue.ID, err)
			continue
		}

		fmt.Printf("  Closed: %s - %s\n", si.Issue.ID, si.Issue.Title)
		closed++
	}

	fmt.Printf("\nClosed %d/%d stale issues.\n", closed, len(stale))
	return nil
}
