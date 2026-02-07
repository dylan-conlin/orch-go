package attention

import (
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// BeadsCollector implements the Collector interface for beads issues.
// It wraps the beads graph API and adds actionability signals for ready issues.
type BeadsCollector struct {
	client beads.BeadsClient
}

// NewBeadsCollector creates a new BeadsCollector with the given beads client.
func NewBeadsCollector(client beads.BeadsClient) *BeadsCollector {
	return &BeadsCollector{
		client: client,
	}
}

// Collect gathers attention items for ready beads issues.
// It queries the beads API for ready issues and transforms them into AttentionItems
// with actionability signals.
func (c *BeadsCollector) Collect(role string) ([]AttentionItem, error) {
	// Query beads for ready issues
	readyIssues, err := c.client.Ready(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query ready issues: %w", err)
	}

	// Transform issues into attention items
	items := make([]AttentionItem, 0, len(readyIssues))
	now := time.Now()

	for _, issue := range readyIssues {
		item := AttentionItem{
			ID:          fmt.Sprintf("beads-%s", issue.ID),
			Source:      "beads",
			Concern:     Actionability,
			Signal:      "issue-ready",
			Subject:     issue.ID,
			Summary:     fmt.Sprintf("%s: %s", issue.IssueType, issue.Title),
			Priority:    issue.Priority, // Direct mapping: beads priority = attention priority
			Role:        role,
			ActionHint:  fmt.Sprintf("orch spawn %s", issue.ID),
			CollectedAt: now,
			Metadata: map[string]any{
				"status":         issue.Status,
				"issue_type":     issue.IssueType,
				"beads_priority": issue.Priority,
			},
		}
		items = append(items, item)
	}

	return items, nil
}
