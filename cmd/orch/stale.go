// stale.go - Show stale beads issues
package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/spf13/cobra"
)

var (
	// Stale command flags
	staleDays       int
	staleStatus     string
	staleLimit      int
	staleJSONOutput bool
)

var staleCmd = &cobra.Command{
	Use:   "stale",
	Short: "Show stale beads issues (not updated recently)",
	Long: `Show stale beads issues that haven't been updated recently.

Detects issues that may need attention based on last update time.
Useful for backlog review during session start.

Examples:
  orch stale                          # Issues not updated in 14 days
  orch stale --days 7                 # Issues not updated in 7 days
  orch stale --status in_progress     # Only in-progress issues
  orch stale --json                   # Output as JSON`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStale()
	},
}

func init() {
	staleCmd.Flags().IntVar(&staleDays, "days", 14, "Days since last update to consider stale")
	staleCmd.Flags().StringVar(&staleStatus, "status", "", "Filter by issue status")
	staleCmd.Flags().IntVar(&staleLimit, "limit", 20, "Maximum number of issues to show")
	staleCmd.Flags().BoolVar(&staleJSONOutput, "json", false, "Output as JSON")

	rootCmd.AddCommand(staleCmd)
}

// StaleIssue represents a stale issue for display.
type StaleIssue struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Priority  int    `json:"priority"`
	UpdatedAt string `json:"updated_at"`
	DaysSince int    `json:"days_since_update"`
}

func runStale() error {
	// Find socket path
	socketPath, err := beads.FindSocketPath("")
	if err != nil {
		return fmt.Errorf("beads socket not found: %w\nMake sure beads daemon is running (bd daemon start)", err)
	}

	// Connect to beads
	client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to beads: %w", err)
	}
	defer client.Close()

	// Use the Stale RPC method directly
	issues, err := client.Stale(&beads.StaleArgs{
		Days:   staleDays,
		Status: staleStatus,
		Limit:  staleLimit,
	})
	if err != nil {
		// Fall back to manual filtering
		return runStaleManual()
	}

	// Convert to display format
	now := time.Now()
	var staleIssues []StaleIssue
	for _, issue := range issues {
		daysSince := 0
		if issue.UpdatedAt != "" {
			if t, err := time.Parse(time.RFC3339, issue.UpdatedAt); err == nil {
				daysSince = int(now.Sub(t).Hours() / 24)
			}
		}

		staleIssues = append(staleIssues, StaleIssue{
			ID:        issue.ID,
			Title:     issue.Title,
			Status:    issue.Status,
			Priority:  issue.Priority,
			UpdatedAt: issue.UpdatedAt,
			DaysSince: daysSince,
		})
	}

	// Output
	if staleJSONOutput {
		data, err := json.MarshalIndent(staleIssues, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	if len(staleIssues) == 0 {
		fmt.Printf("✨ No stale issues found (all updated within %d days)\n", staleDays)
		return nil
	}

	statusFilterStr := ""
	if staleStatus != "" {
		statusFilterStr = fmt.Sprintf(" with status '%s'", staleStatus)
	}
	fmt.Printf("⏰ %d stale issue(s) not updated in %d+ days%s:\n\n", len(staleIssues), staleDays, statusFilterStr)

	for _, issue := range staleIssues {
		priorityStr := fmt.Sprintf("P%d", issue.Priority)
		if issue.Priority == 0 {
			priorityStr = "P?"
		}

		fmt.Printf("  [%s] %s: %s\n", priorityStr, issue.ID, issue.Title)
		fmt.Printf("       Status: %s, Updated: %s (%d days ago)\n\n", issue.Status, issue.UpdatedAt, issue.DaysSince)
	}

	return nil
}

// runStaleManual is a fallback that manually filters issues by date.
func runStaleManual() error {
	issues, err := beads.FallbackList("")
	if err != nil {
		return fmt.Errorf("failed to list issues: %w", err)
	}

	// Filter stale issues
	now := time.Now()
	cutoff := now.AddDate(0, 0, -staleDays)
	var staleIssues []StaleIssue

	for _, issue := range issues {
		// Apply status filter if specified
		if staleStatus != "" && issue.Status != staleStatus {
			continue
		}

		// Skip closed issues
		if issue.Status == "closed" {
			continue
		}

		// Parse updated_at
		var updatedAt time.Time
		if issue.UpdatedAt != "" {
			t, err := time.Parse(time.RFC3339, issue.UpdatedAt)
			if err != nil {
				t, err = time.Parse("2006-01-02T15:04:05", issue.UpdatedAt)
			}
			if err == nil {
				updatedAt = t
			}
		}

		// Skip if updated within the threshold
		if !updatedAt.IsZero() && updatedAt.After(cutoff) {
			continue
		}

		daysSince := 0
		if !updatedAt.IsZero() {
			daysSince = int(now.Sub(updatedAt).Hours() / 24)
		}

		staleIssues = append(staleIssues, StaleIssue{
			ID:        issue.ID,
			Title:     issue.Title,
			Status:    issue.Status,
			Priority:  issue.Priority,
			UpdatedAt: issue.UpdatedAt,
			DaysSince: daysSince,
		})

		if staleLimit > 0 && len(staleIssues) >= staleLimit {
			break
		}
	}

	// Output using shared function
	return outputStaleIssues(staleIssues)
}

// outputStaleIssues handles the output formatting for stale issues.
func outputStaleIssues(staleIssues []StaleIssue) error {
	if staleJSONOutput {
		data, err := json.MarshalIndent(staleIssues, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	if len(staleIssues) == 0 {
		fmt.Printf("✨ No stale issues found (all updated within %d days)\n", staleDays)
		return nil
	}

	statusFilterStr := ""
	if staleStatus != "" {
		statusFilterStr = fmt.Sprintf(" with status '%s'", staleStatus)
	}
	fmt.Printf("⏰ %d stale issue(s) not updated in %d+ days%s:\n\n", len(staleIssues), staleDays, statusFilterStr)

	for _, issue := range staleIssues {
		priorityStr := fmt.Sprintf("P%d", issue.Priority)
		if issue.Priority == 0 {
			priorityStr = "P?"
		}

		fmt.Printf("  [%s] %s: %s\n", priorityStr, issue.ID, issue.Title)
		fmt.Printf("       Status: %s, Updated: %s (%d days ago)\n\n", issue.Status, issue.UpdatedAt, issue.DaysSince)
	}

	return nil
}
