package attention

import (
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// RecentlyClosedCollector implements the Collector interface for recently closed issues.
// It queries beads for issues closed within a configurable time window (default 24h)
// and surfaces them as observability signals for human verification.
type RecentlyClosedCollector struct {
	client beads.BeadsClient
	// LookbackHours is the number of hours to look back for closed issues.
	// Default: 24 hours
	LookbackHours int
}

// NewRecentlyClosedCollector creates a new RecentlyClosedCollector with the given beads client.
// lookbackHours specifies how far back to look for closed issues (default: 24h if <= 0).
func NewRecentlyClosedCollector(client beads.BeadsClient, lookbackHours int) *RecentlyClosedCollector {
	if lookbackHours <= 0 {
		lookbackHours = 24 // Default to 24 hours
	}
	return &RecentlyClosedCollector{
		client:        client,
		LookbackHours: lookbackHours,
	}
}

// Collect gathers attention items for recently closed issues.
// These are observability signals that help orchestrators verify completion.
func (c *RecentlyClosedCollector) Collect(role string) ([]AttentionItem, error) {
	// Calculate the timestamp for lookback window
	lookbackTime := time.Now().Add(-time.Duration(c.LookbackHours) * time.Hour)
	closedAfter := lookbackTime.Format(time.RFC3339)

	// Query beads for recently closed issues
	args := &beads.ListArgs{
		Status:      "closed",
		ClosedAfter: closedAfter,
		Limit:       0, // Get all closed issues in window
	}

	closedIssues, err := c.client.List(args)
	if err != nil {
		return nil, fmt.Errorf("failed to query recently closed issues: %w", err)
	}

	// Transform issues into attention items
	items := make([]AttentionItem, 0, len(closedIssues))
	now := time.Now()

	for _, issue := range closedIssues {
		// Parse closed_at timestamp
		closedAt, err := time.Parse(time.RFC3339, issue.ClosedAt)
		if err != nil {
			// Skip issues with invalid timestamps
			continue
		}

		// Calculate priority based on role and how recently it was closed
		priority := calculateRecentlyClosedPriority(closedAt, role)

		item := AttentionItem{
			ID:          fmt.Sprintf("recently-closed-%s", issue.ID),
			Source:      "beads",
			Concern:     Observability,
			Signal:      "recently-closed",
			Subject:     issue.ID,
			Summary:     fmt.Sprintf("Closed %s: %s", formatRelativeTime(closedAt), issue.Title),
			Priority:    priority,
			Role:        role,
			ActionHint:  fmt.Sprintf("Review completion: bd show %s", issue.ID),
			CollectedAt: now,
			Metadata: map[string]any{
				"closed_at":      issue.ClosedAt,
				"close_reason":   issue.CloseReason,
				"issue_type":     issue.IssueType,
				"beads_priority": issue.Priority,
				"status":         issue.Status,
			},
		}
		items = append(items, item)
	}

	return items, nil
}

// calculateRecentlyClosedPriority determines priority based on role and how recently closed.
// Lower numbers = higher priority.
func calculateRecentlyClosedPriority(closedAt time.Time, role string) int {
	// Base priority for recently closed (lower than actionable work, but higher than stale info)
	basePriority := 150

	// How many hours ago was it closed?
	hoursAgo := time.Since(closedAt).Hours()

	// Role-aware adjustments
	switch role {
	case "human":
		// Humans care about verification - prioritize more recently closed issues
		if hoursAgo < 1 {
			return basePriority - 30 // Very recently closed (< 1h) = high priority
		} else if hoursAgo < 6 {
			return basePriority - 20 // Recently closed (< 6h)
		} else if hoursAgo < 12 {
			return basePriority - 10 // Moderately recent (< 12h)
		}
		return basePriority

	case "orchestrator":
		// Orchestrators need to track recent completions
		if hoursAgo < 2 {
			return basePriority - 25 // Very recent for tracking
		} else if hoursAgo < 8 {
			return basePriority - 15
		}
		return basePriority

	case "daemon":
		// Daemons don't care about closed issues
		return basePriority + 100

	default:
		return basePriority
	}
}

// formatRelativeTime formats a timestamp as relative time (e.g., "2h ago", "just now").
func formatRelativeTime(t time.Time) string {
	duration := time.Since(t)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes())

	if minutes < 1 {
		return "just now"
	} else if minutes < 60 {
		return fmt.Sprintf("%dm ago", minutes)
	} else if hours < 24 {
		return fmt.Sprintf("%dh ago", hours)
	} else {
		days := hours / 24
		return fmt.Sprintf("%dd ago", days)
	}
}
