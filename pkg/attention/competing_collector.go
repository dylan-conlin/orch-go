package attention

import (
	"fmt"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// CompetingCollector implements the Collector interface for detecting issues
// in the same work area with similar scope that may compete for the same solution
// space. This is distinct from duplicates - competing issues address overlapping
// concerns from different angles but may conflict if worked simultaneously.
type CompetingCollector struct {
	client    beads.BeadsClient
	threshold float64 // Title similarity threshold for competing detection (default 0.4)
}

// NewCompetingCollector creates a new CompetingCollector.
// threshold is the minimum title similarity for issues in the same area to be flagged (default: 0.4).
// This is lower than duplicate threshold because area overlap already indicates relatedness.
func NewCompetingCollector(client beads.BeadsClient, threshold float64) *CompetingCollector {
	if threshold <= 0 || threshold > 1.0 {
		threshold = 0.4
	}
	return &CompetingCollector{
		client:    client,
		threshold: threshold,
	}
}

// Collect gathers attention items for competing issues (same area + similar scope).
func (c *CompetingCollector) Collect(role string) ([]AttentionItem, error) {
	issues, err := c.client.List(&beads.ListArgs{
		Status: "open",
		Limit:  0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query open issues: %w", err)
	}

	if len(issues) < 2 {
		return nil, nil
	}

	// Group issues by area label
	groups := make(map[string][]beads.Issue)
	for _, issue := range issues {
		area := extractArea(issue.Labels)
		if area == "" {
			continue // Only detect competing within labeled issues
		}
		groups[area] = append(groups[area], issue)
	}

	now := time.Now()
	items := make([]AttentionItem, 0)
	seen := make(map[string]bool)

	for area, group := range groups {
		if len(group) < 2 {
			continue
		}

		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				score := titleSimilarity(group[i].Title, group[j].Title)
				if score < c.threshold {
					continue
				}

				key := group[i].ID + ":" + group[j].ID
				if seen[key] {
					continue
				}
				seen[key] = true

				priority := calculateCompetingPriority(score, role)

				item := AttentionItem{
					ID:          fmt.Sprintf("competing-%s-%s", group[i].ID, group[j].ID),
					Source:      "beads",
					Concern:     Observability,
					Signal:      "competing",
					Subject:     group[j].ID,
					Summary:     fmt.Sprintf("Competing in %s (%.0f%%): %s", area, score*100, group[i].Title),
					Priority:    priority,
					Role:        role,
					ActionHint:  fmt.Sprintf("Review overlap: bd show %s && bd show %s", group[i].ID, group[j].ID),
					CollectedAt: now,
					Metadata: map[string]any{
						"area":            area,
						"competing_id":    group[i].ID,
						"competing_title": group[i].Title,
						"this_title":      group[j].Title,
						"score":           score,
					},
				}
				items = append(items, item)
			}
		}
	}

	return items, nil
}

// extractArea returns the first area:* label value from labels, or empty string.
func extractArea(labels []string) string {
	for _, label := range labels {
		if strings.HasPrefix(label, "area:") {
			return strings.TrimPrefix(label, "area:")
		}
	}
	return ""
}

// calculateCompetingPriority determines priority based on similarity and role.
func calculateCompetingPriority(score float64, role string) int {
	base := 190

	if score > 0.8 {
		base -= 30 // Very similar within same area
	} else if score > 0.6 {
		base -= 20
	} else if score > 0.4 {
		base -= 10
	}

	switch role {
	case "human":
		return base - 10
	case "orchestrator":
		return base - 5 // Orchestrator should coordinate competing work
	case "daemon":
		return base + 100
	default:
		return base
	}
}
