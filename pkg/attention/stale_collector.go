package attention

import (
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// StaleIssueCollector implements the Collector interface for issues with no activity
// beyond a configurable threshold (default 30 days). These are Observability signals
// that help orchestrators identify forgotten or abandoned work.
type StaleIssueCollector struct {
	client    beads.BeadsClient
	staleDays int
}

// NewStaleIssueCollector creates a new StaleIssueCollector.
// staleDays is the number of days of inactivity after which an issue is considered stale (default: 30).
func NewStaleIssueCollector(client beads.BeadsClient, staleDays int) *StaleIssueCollector {
	if staleDays <= 0 {
		staleDays = 30
	}
	return &StaleIssueCollector{
		client:    client,
		staleDays: staleDays,
	}
}

// Collect gathers attention items for stale open issues.
func (c *StaleIssueCollector) Collect(role string) ([]AttentionItem, error) {
	// Query all open issues
	issues, err := c.client.List(&beads.ListArgs{
		Status: "open",
		Limit:  0, // All open issues
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query open issues: %w", err)
	}

	now := time.Now()
	threshold := now.AddDate(0, 0, -c.staleDays)
	items := make([]AttentionItem, 0)

	for _, issue := range issues {
		// Use UpdatedAt if available, fall back to CreatedAt
		timestamp := issue.UpdatedAt
		if timestamp == "" {
			timestamp = issue.CreatedAt
		}
		if timestamp == "" {
			continue
		}

		lastActivity, err := time.Parse(time.RFC3339, timestamp)
		if err != nil {
			// Try date-only format
			lastActivity, err = time.Parse("2006-01-02", timestamp)
			if err != nil {
				continue
			}
		}

		if lastActivity.After(threshold) {
			continue // Not stale
		}

		staleDays := int(now.Sub(lastActivity).Hours() / 24)
		priority := calculateStalePriority(staleDays, role)

		item := AttentionItem{
			ID:          fmt.Sprintf("stale-%s", issue.ID),
			Source:      "beads",
			Concern:     Observability,
			Signal:      "stale",
			Subject:     issue.ID,
			Summary:     fmt.Sprintf("Stale %dd: %s", staleDays, issue.Title),
			Priority:    priority,
			Role:        role,
			ActionHint:  fmt.Sprintf("Review: bd show %s (close, update, or re-prioritize)", issue.ID),
			CollectedAt: now,
			Metadata: map[string]any{
				"stale_days":     staleDays,
				"last_activity":  timestamp,
				"issue_type":     issue.IssueType,
				"beads_priority": issue.Priority,
			},
		}
		items = append(items, item)
	}

	return items, nil
}

// calculateStalePriority determines priority based on how stale and the role.
// Lower numbers = higher priority.
func calculateStalePriority(staleDays int, role string) int {
	// Base priority for stale issues - informational, not urgent
	base := 200

	// Staler = higher priority (more urgent to clean up)
	if staleDays > 90 {
		base -= 40 // Very stale (>3 months)
	} else if staleDays > 60 {
		base -= 25 // Moderately stale (>2 months)
	} else if staleDays > 30 {
		base -= 10 // Just crossed threshold
	}

	switch role {
	case "human":
		return base - 10 // Humans should review stale work
	case "orchestrator":
		return base
	case "daemon":
		return base + 100 // Daemon can't act on stale issues
	default:
		return base
	}
}
