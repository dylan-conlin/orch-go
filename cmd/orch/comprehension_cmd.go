package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/spf13/cobra"
)

var comprehensionCmd = &cobra.Command{
	Use:   "comprehension",
	Short: "Manage comprehension queue (two-state: unread → processed)",
	Long: `List and manage the comprehension queue.

Two-state lifecycle:
  comprehension:unread     — daemon completed work, orchestrator hasn't reviewed yet
  comprehension:processed  — orchestrator reviewed (orch complete), Dylan hasn't read brief

The daemon throttles spawning based on comprehension:unread count.
orch complete transitions unread → processed.
Reading the brief removes comprehension:processed.`,
}

var comprehensionListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List comprehension items by state",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Show unread items (need orchestrator review)
		unreadOutput, err := daemon.RunBdListComprehensionUnread()
		if err != nil {
			return fmt.Errorf("failed to list unread queue: %w", err)
		}
		unreadItems := parseComprehensionItems(unreadOutput)

		// Show processed items (need Dylan to read brief)
		processedOutput, err := daemon.RunBdListComprehensionProcessed()
		if err != nil {
			return fmt.Errorf("failed to list processed queue: %w", err)
		}
		processedItems := parseComprehensionItems(processedOutput)

		// Check for legacy pending items
		legacyOutput, _ := daemon.RunBdListComprehensionPending()
		legacyItems := parseComprehensionItems(legacyOutput)

		if len(unreadItems) == 0 && len(processedItems) == 0 && len(legacyItems) == 0 {
			fmt.Println("Comprehension queue is empty.")
			return nil
		}

		if len(unreadItems) > 0 {
			fmt.Printf("Unread (%d) — needs orchestrator review:\n", len(unreadItems))
			for _, item := range unreadItems {
				fmt.Printf("  %s  %s%s\n", item.ID, item.Title, formatAge(item.ClosedAt))
			}
			fmt.Println()
		}

		if len(processedItems) > 0 {
			fmt.Printf("Processed (%d) — needs brief read:\n", len(processedItems))
			for _, item := range processedItems {
				feedback := ""
				if rating, _ := daemon.ReadBriefFeedback(item.ID, sourceDir); rating != "" {
					feedback = fmt.Sprintf(" [%s]", rating)
				}
				fmt.Printf("  %s  %s%s%s\n", item.ID, item.Title, formatAge(item.ClosedAt), feedback)
			}
			fmt.Println()
		}

		if len(legacyItems) > 0 {
			fmt.Printf("Legacy pending (%d) — migrate with 'orch complete':\n", len(legacyItems))
			for _, item := range legacyItems {
				fmt.Printf("  %s  %s%s\n", item.ID, item.Title, formatAge(item.ClosedAt))
			}
			fmt.Println()
		}

		fmt.Printf("Throttle: %d unread (threshold: %d)\n", len(unreadItems)+len(legacyItems), daemon.DefaultComprehensionThreshold)
		return nil
	},
}

var comprehensionCountCmd = &cobra.Command{
	Use:   "count",
	Short: "Show count of items needing orchestrator review",
	RunE: func(cmd *cobra.Command, args []string) error {
		q := &daemon.BeadsComprehensionQuerier{}
		count, err := q.CountPending()
		if err != nil {
			return fmt.Errorf("failed to count: %w", err)
		}
		fmt.Printf("%d\n", count)
		return nil
	},
}

var comprehensionReviewCmd = &cobra.Command{
	Use:   "review <beads-id>",
	Short: "Mark an issue as reviewed (transition unread → processed)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		if err := daemon.TransitionToProcessed(beadsID); err != nil {
			return fmt.Errorf("failed to transition %s to processed: %w", beadsID, err)
		}
		fmt.Printf("Transitioned %s: unread → processed\n", beadsID)
		return nil
	},
}

var comprehensionReadCmd = &cobra.Command{
	Use:   "read <beads-id>",
	Short: "Mark a processed issue as read (remove comprehension:processed)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		if err := daemon.RemoveComprehensionProcessed(beadsID); err != nil {
			return fmt.Errorf("failed to mark %s as read: %w", beadsID, err)
		}
		fmt.Printf("Marked %s as read (removed comprehension:processed)\n", beadsID)
		return nil
	},
}

var comprehensionFeedbackCmd = &cobra.Command{
	Use:   "feedback <beads-id> <shallow|good>",
	Short: "Rate brief quality for a completed issue",
	Long: `Record quality feedback on a comprehension brief.

Ratings:
  shallow  — brief lacks depth, missing key insights
  good     — brief is useful and comprehensive

Feedback is stored in .kb/briefs/feedback/ and used to improve brief generation.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		rating := args[1]

		projectDir := sourceDir
		if err := daemon.RecordBriefFeedback(beadsID, rating, projectDir); err != nil {
			return fmt.Errorf("failed to record feedback: %w", err)
		}
		fmt.Printf("Recorded feedback for %s: %s\n", beadsID, rating)
		return nil
	},
}

type comprehensionItem struct {
	ID       string
	Title    string
	ClosedAt time.Time
}

func formatAge(closedAt time.Time) string {
	if closedAt.IsZero() {
		return ""
	}
	return fmt.Sprintf(" (closed %s ago)", time.Since(closedAt).Truncate(time.Minute))
}

func parseComprehensionItems(output []byte) []comprehensionItem {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" || trimmed == "[]" {
		return nil
	}

	var items []comprehensionItem
	for _, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		item := comprehensionItem{}
		if id, ok := raw["id"].(string); ok {
			item.ID = id
		}
		if title, ok := raw["title"].(string); ok {
			item.Title = title
		}
		if closedAt, ok := raw["closed_at"].(string); ok && closedAt != "" {
			if t, err := time.Parse(time.RFC3339, closedAt); err == nil {
				item.ClosedAt = t
			}
		}
		items = append(items, item)
	}
	return items
}

func init() {
	comprehensionCmd.AddCommand(comprehensionListCmd)
	comprehensionCmd.AddCommand(comprehensionCountCmd)
	comprehensionCmd.AddCommand(comprehensionReviewCmd)
	comprehensionCmd.AddCommand(comprehensionReadCmd)
	comprehensionCmd.AddCommand(comprehensionFeedbackCmd)

	// Default to list when no subcommand given
	comprehensionCmd.RunE = comprehensionListCmd.RunE
}
