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
	Short: "Manage comprehension queue (pending review items)",
	Long: `List and manage issues with comprehension:pending label.

The daemon adds comprehension:pending after auto-completing agents.
The orchestrator removes it during completion review (orch complete).
When the queue exceeds the threshold (default 5), the daemon pauses spawning.`,
}

var comprehensionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pending comprehension items",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		output, err := daemon.RunBdListComprehensionPending()
		if err != nil {
			return fmt.Errorf("failed to list comprehension queue: %w", err)
		}

		items := parseComprehensionItems(output)
		if len(items) == 0 {
			fmt.Println("Comprehension queue is empty.")
			return nil
		}

		fmt.Printf("Comprehension queue: %d pending items\n\n", len(items))
		for _, item := range items {
			age := ""
			if !item.ClosedAt.IsZero() {
				age = fmt.Sprintf(" (closed %s ago)", time.Since(item.ClosedAt).Truncate(time.Minute))
			}
			fmt.Printf("  %s  %s%s\n", item.ID, item.Title, age)
		}
		return nil
	},
}

var comprehensionCountCmd = &cobra.Command{
	Use:   "count",
	Short: "Show count of pending comprehension items",
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
	Short: "Mark an issue as comprehended (remove comprehension:pending)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		if err := daemon.RemoveComprehensionPending(beadsID); err != nil {
			return fmt.Errorf("failed to mark %s as comprehended: %w", beadsID, err)
		}
		fmt.Printf("Marked %s as comprehended (removed comprehension:pending)\n", beadsID)
		return nil
	},
}

type comprehensionItem struct {
	ID       string
	Title    string
	ClosedAt time.Time
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

	// Default to list when no subcommand given
	comprehensionCmd.RunE = comprehensionListCmd.RunE
}
