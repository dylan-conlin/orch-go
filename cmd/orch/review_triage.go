// Package main provides the triage:review pass for orch review.
// This surfaces items stuck in triage:review and forces a decision:
// promote to triage:ready, close, or defer with reason.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	triageNonInteractive bool
)

// TriageItem represents a beads issue in triage:review state.
type TriageItem struct {
	ID             string
	Title          string
	Priority       int
	IssueType      string
	Age            string // Human-readable age (e.g., "5d", "2h")
	CreatedAt      time.Time
	IsCompletedWork bool // True if daemon:ready-review label present (agent completed work on this)
}

var reviewTriageCmd = &cobra.Command{
	Use:   "triage",
	Short: "Process triage:review items — promote, close, or defer",
	Long: `Process items stuck in triage:review state.

For each item, you decide:
  [r] ready    - Promote to triage:ready (daemon will spawn it)
  [c] close    - Close the issue (not worth doing)
  [d] defer    - Keep in triage:review with a reason comment
  [s] skip     - Skip this item (no change)

Use --non-interactive to just list items without prompting.

Examples:
  orch review triage                  # Interactive triage pass
  orch review triage --non-interactive  # Just list items`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReviewTriage(triageNonInteractive)
	},
}

func init() {
	reviewTriageCmd.Flags().BoolVar(&triageNonInteractive, "non-interactive", false, "List items without prompting for decisions")
	reviewCmd.AddCommand(reviewTriageCmd)
}

// triageItemFromIssue converts a beads Issue to a TriageItem.
func triageItemFromIssue(issue beads.Issue) TriageItem {
	item := TriageItem{
		ID:        issue.ID,
		Title:     issue.Title,
		Priority:  issue.Priority,
		IssueType: issue.IssueType,
	}

	// Check if this is completed agent work (has daemon:ready-review label)
	for _, label := range issue.Labels {
		if label == "daemon:ready-review" {
			item.IsCompletedWork = true
			break
		}
	}

	// Parse creation time for age calculation
	if issue.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, issue.CreatedAt); err == nil {
			item.CreatedAt = t
			item.Age = humanAge(time.Since(t))
		} else {
			// Try alternate format
			if t, err := time.Parse(time.RFC3339, issue.CreatedAt); err == nil {
				item.CreatedAt = t
				item.Age = humanAge(time.Since(t))
			}
		}
	}

	return item
}

// humanAge formats a duration as a human-readable age string.
func humanAge(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// formatTriageList formats triage items for display.
func formatTriageList(items []TriageItem) string {
	if len(items) == 0 {
		return "No triage:review items found.\n"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n## Triage Review (%d items)\n\n", len(items)))

	for i, item := range items {
		typeTag := item.IssueType
		ageStr := ""
		if item.Age != "" {
			ageStr = fmt.Sprintf(" (%s old)", item.Age)
		}
		originTag := "new"
		if item.IsCompletedWork {
			originTag = "completed"
		}
		b.WriteString(fmt.Sprintf("  %2d. [P%d] [%s] [%s] %s%s\n", i+1, item.Priority, typeTag, originTag, item.ID, ageStr))
		// Truncate long titles
		title := item.Title
		if len(title) > 90 {
			title = title[:87] + "..."
		}
		b.WriteString(fmt.Sprintf("      %s\n", title))
	}

	return b.String()
}

// formatTriageSummary returns a one-line hygiene nudge for the main review output.
// Returns empty string if count is 0.
func formatTriageSummary(count int) string {
	if count == 0 {
		return ""
	}
	return fmt.Sprintf("Triage: %d items in triage:review awaiting decision (run: orch review triage)\n", count)
}

// getTriageReviewItems fetches open issues with the triage:review label.
func getTriageReviewItems() ([]TriageItem, error) {
	// Try RPC client first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath)
		if err := client.Connect(); err == nil {
			defer client.Close()

			issues, err := client.List(&beads.ListArgs{
				Status: "open",
				Labels: []string{"triage:review"},
			})
			if err == nil {
				var items []TriageItem
				for _, issue := range issues {
					items = append(items, triageItemFromIssue(issue))
				}
				return items, nil
			}
			// Fall through to CLI fallback
		}
	}

	// Fallback: use CLI client
	cliClient := beads.NewCLIClient()
	issues, err := cliClient.List(&beads.ListArgs{
		Status: "open",
		Labels: []string{"triage:review"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list triage:review items: %w", err)
	}

	var items []TriageItem
	for _, issue := range issues {
		items = append(items, triageItemFromIssue(issue))
	}
	return items, nil
}

// getTriageReviewCount returns the count of open triage:review items.
// Optimized for the hygiene nudge in regular review output.
func getTriageReviewCount() int {
	items, err := getTriageReviewItems()
	if err != nil {
		return 0
	}
	return len(items)
}

// runReviewTriage implements the interactive triage pass.
func runReviewTriage(nonInteractive bool) error {
	items, err := getTriageReviewItems()
	if err != nil {
		return err
	}

	// Display the list
	fmt.Print(formatTriageList(items))

	if len(items) == 0 {
		return nil
	}

	if nonInteractive {
		fmt.Printf("\nUse 'orch review triage' (without --non-interactive) to process these items.\n")
		return nil
	}

	// Check if stdin is a terminal
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Println("\n(Non-interactive mode - stdin is not a terminal)")
		return nil
	}

	// Interactive processing
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\nFor each item: [r]eady  [c]lose  [d]efer  [s]kip  [q]uit")
	fmt.Println()

	promoted := 0
	closed := 0
	deferred := 0
	skipped := 0

	for _, item := range items {
		originTag := "new"
		if item.IsCompletedWork {
			originTag = "completed"
		}
		fmt.Printf("  [P%d] [%s] [%s] %s\n", item.Priority, item.IssueType, originTag, item.ID)
		fmt.Printf("  %s\n", truncateTitle(item.Title, 90))
		fmt.Print("  Decision [r/c/d/s/q]: ")

		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))

		switch response {
		case "r", "ready":
			if err := promoteToReady(item.ID); err != nil {
				fmt.Printf("  Error promoting: %v\n", err)
			} else {
				fmt.Printf("  -> Promoted to triage:ready\n")
				promoted++
			}

		case "c", "close":
			fmt.Print("  Close reason (optional): ")
			reason, _ := reader.ReadString('\n')
			reason = strings.TrimSpace(reason)
			if reason == "" {
				reason = "Closed during triage review"
			}
			if err := closeTriageItem(item.ID, reason); err != nil {
				fmt.Printf("  Error closing: %v\n", err)
			} else {
				fmt.Printf("  -> Closed: %s\n", reason)
				closed++
			}

		case "d", "defer":
			fmt.Print("  Defer reason: ")
			reason, _ := reader.ReadString('\n')
			reason = strings.TrimSpace(reason)
			if reason == "" {
				reason = "Deferred during triage review"
			}
			if err := deferTriageItem(item.ID, reason); err != nil {
				fmt.Printf("  Error deferring: %v\n", err)
			} else {
				fmt.Printf("  -> Deferred: %s\n", reason)
				deferred++
			}

		case "q", "quit":
			fmt.Println("  Quitting triage review.")
			goto summary

		case "s", "skip", "":
			fmt.Println("  -> Skipped")
			skipped++

		default:
			fmt.Printf("  Unknown option '%s', skipping\n", response)
			skipped++
		}

		fmt.Println()
	}

summary:
	fmt.Println("---")
	fmt.Printf("Triage summary: %d promoted, %d closed, %d deferred, %d skipped\n",
		promoted, closed, deferred, skipped)

	return nil
}

// promoteToReady promotes an issue from triage:review to triage:ready.
func promoteToReady(id string) error {
	client, cleanup, err := getBeadsClient()
	if err != nil {
		return err
	}
	defer cleanup()

	// Remove triage:review, add triage:ready
	if err := client.RemoveLabel(id, "triage:review"); err != nil {
		return fmt.Errorf("failed to remove triage:review: %w", err)
	}
	if err := client.AddLabel(id, "triage:ready"); err != nil {
		return fmt.Errorf("failed to add triage:ready: %w", err)
	}
	return nil
}

// closeTriageItem closes a triage:review issue.
func closeTriageItem(id string, reason string) error {
	client, cleanup, err := getBeadsClient()
	if err != nil {
		return err
	}
	defer cleanup()

	return client.CloseIssue(id, reason)
}

// deferTriageItem adds a comment with the defer reason but keeps the issue open.
func deferTriageItem(id string, reason string) error {
	client, cleanup, err := getBeadsClient()
	if err != nil {
		return err
	}
	defer cleanup()

	return client.AddComment(id, "orch-triage", fmt.Sprintf("Deferred during triage review: %s", reason))
}

// getBeadsClient returns a beads client with cleanup function.
// Tries RPC first, falls back to CLI.
func getBeadsClient() (beads.BeadsClient, func(), error) {
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath)
		if err := client.Connect(); err == nil {
			return client, func() { client.Close() }, nil
		}
	}

	// Fallback to CLI
	cliClient := beads.NewCLIClient()
	return cliClient, func() {}, nil
}

// truncateTitle truncates a title string for display.
func truncateTitle(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
