// logs.go - View orch command logs
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/spf13/cobra"
)

var (
	// Logs command flags
	logsLimit         int
	logsTypeFilter    string
	logsBeadsIDFilter string
	logsJSONOutput    bool
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View orch event logs",
	Long: `View orch event logs with optional filtering.

Events are stored in ~/.orch/events.jsonl and include:
- session.spawned    - Agent spawn events
- session.completed  - Agent completion events
- session.error      - Agent error events
- agent.abandoned    - Agent abandonment events
- agent.completed    - Agent completion via orch complete
- agents.cleaned     - Cleanup events

Examples:
  orch logs                           # Show last 50 events
  orch logs --limit 100               # Show last 100 events
  orch logs --type session.spawned    # Filter by event type
  orch logs --beads-id orch-go-xxxx   # Filter by beads ID
  orch logs --json                    # Output as JSON`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLogs()
	},
}

func init() {
	logsCmd.Flags().IntVar(&logsLimit, "limit", 50, "Number of log entries to show")
	logsCmd.Flags().StringVar(&logsTypeFilter, "type", "", "Filter by event type")
	logsCmd.Flags().StringVar(&logsBeadsIDFilter, "beads-id", "", "Filter by beads ID")
	logsCmd.Flags().BoolVar(&logsJSONOutput, "json", false, "Output as JSON")

	rootCmd.AddCommand(logsCmd)
}

// LogEntry represents a parsed log entry.
type LogEntry struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Time      string                 `json:"time,omitempty"` // Formatted time for display
}

func runLogs() error {
	logPath := events.DefaultLogPath()

	// Check if log file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		fmt.Println("No log entries found.")
		return nil
	}

	// Read log file
	file, err := os.Open(logPath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Parse all entries and filter
	var entries []LogEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip malformed lines
		}

		// Apply type filter
		if logsTypeFilter != "" && !strings.Contains(entry.Type, logsTypeFilter) {
			continue
		}

		// Apply beads ID filter
		if logsBeadsIDFilter != "" {
			beadsID := ""
			if entry.Data != nil {
				if id, ok := entry.Data["beads_id"].(string); ok {
					beadsID = id
				}
			}
			if beadsID == "" || !strings.Contains(beadsID, logsBeadsIDFilter) {
				// Also check session ID
				if !strings.Contains(entry.SessionID, logsBeadsIDFilter) {
					continue
				}
			}
		}

		// Format timestamp for display
		entry.Time = time.Unix(entry.Timestamp, 0).Format("2006-01-02 15:04:05")

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading log file: %w", err)
	}

	// Take last N entries
	if len(entries) > logsLimit {
		entries = entries[len(entries)-logsLimit:]
	}

	// Output
	if logsJSONOutput {
		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	if len(entries) == 0 {
		fmt.Println("No log entries found matching criteria.")
		return nil
	}

	fmt.Printf("\n📋 Orch Logs (showing %d entries)\n\n", len(entries))

	for _, entry := range entries {
		// Emoji based on event type
		emoji := getEventEmoji(entry.Type)

		fmt.Printf("%s %s [%s]", emoji, entry.Time, entry.Type)

		// Add key details
		if entry.Data != nil {
			if beadsID, ok := entry.Data["beads_id"].(string); ok {
				fmt.Printf(" beads:%s", beadsID)
			} else if entry.SessionID != "" && len(entry.SessionID) > 12 {
				fmt.Printf(" session:%s", entry.SessionID[:12])
			}
			if skill, ok := entry.Data["skill"].(string); ok {
				fmt.Printf(" skill:%s", skill)
			}
		}
		fmt.Println()
	}

	fmt.Println()
	return nil
}

func getEventEmoji(eventType string) string {
	switch {
	case strings.Contains(eventType, "spawned"):
		return "🚀"
	case strings.Contains(eventType, "completed"):
		return "✅"
	case strings.Contains(eventType, "error"):
		return "❌"
	case strings.Contains(eventType, "abandoned"):
		return "🚫"
	case strings.Contains(eventType, "cleaned"):
		return "🧹"
	case strings.Contains(eventType, "send"):
		return "💬"
	case strings.Contains(eventType, "switched"):
		return "🔄"
	case strings.Contains(eventType, "gap"):
		return "⚠️"
	default:
		return "•"
	}
}
