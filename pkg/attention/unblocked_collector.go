package attention

import (
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// UnblockedCollector implements the Collector interface for issues that were blocked
// but are now unblocked (all blockers have been resolved). This helps orchestrators
// identify work that has become actionable after dependencies completed.
type UnblockedCollector struct {
	client beads.BeadsClient
}

// NewUnblockedCollector creates a new UnblockedCollector with the given beads client.
func NewUnblockedCollector(client beads.BeadsClient) *UnblockedCollector {
	return &UnblockedCollector{
		client: client,
	}
}

// Collect gathers attention items for issues that have dependencies but are now unblocked.
// These are actionability signals - the issue can now be worked on.
func (c *UnblockedCollector) Collect(role string) ([]AttentionItem, error) {
	// Query open/in_progress issues
	args := &beads.ListArgs{
		Status: "open,in_progress",
		Limit:  0, // Get all
	}

	issues, err := c.client.List(args)
	if err != nil {
		return nil, fmt.Errorf("failed to list open issues: %w", err)
	}

	// Filter for issues that have dependencies but are unblocked
	items := make([]AttentionItem, 0)
	now := time.Now()

	for _, issue := range issues {
		// Skip issues without dependencies
		deps := issue.ParseDependencies()
		if len(deps) == 0 {
			continue
		}

		// Check if any blocking dependencies remain
		blockers := issue.GetBlockingDependencies()
		if len(blockers) > 0 {
			// Still blocked, skip
			continue
		}

		// Issue has dependencies but none are blocking - it's unblocked!
		// Find which dependencies were resolved (closed or answered)
		resolvedDeps := findResolvedDependencies(deps)
		if len(resolvedDeps) == 0 {
			// No dependencies were blocking type, skip
			continue
		}

		priority := calculateUnblockedPriority(issue, role)

		summary := fmt.Sprintf("Unblocked: %s (was waiting on %d deps)", issue.Title, len(resolvedDeps))

		item := AttentionItem{
			ID:          fmt.Sprintf("unblocked-%s", issue.ID),
			Source:      "beads",
			Concern:     Actionability,
			Signal:      "unblocked",
			Subject:     issue.ID,
			Summary:     summary,
			Priority:    priority,
			Role:        role,
			ActionHint:  fmt.Sprintf("orch spawn %s", issue.ID),
			CollectedAt: now,
			Metadata: map[string]any{
				"issue_type":     issue.IssueType,
				"issue_status":   issue.Status,
				"resolved_deps":  resolvedDeps,
				"total_deps":     len(deps),
				"beads_priority": issue.Priority,
			},
		}
		items = append(items, item)
	}

	return items, nil
}

// findResolvedDependencies returns dependency IDs that were blocking-type and are now resolved.
// This distinguishes between issues that never had blockers vs issues that had blockers that resolved.
func findResolvedDependencies(deps []beads.Dependency) []string {
	var resolved []string
	for _, dep := range deps {
		// Skip parent-child relationships (they never block)
		if dep.EffectiveType() == "parent-child" {
			continue
		}

		// Check if this was a blocking dependency that is now resolved
		// Issues resolve when closed; questions also resolve when answered
		isResolved := dep.Status == "closed" || dep.Status == "answered"

		if isResolved {
			resolved = append(resolved, dep.EffectiveID())
		}
	}
	return resolved
}

// calculateUnblockedPriority determines priority based on role and issue characteristics.
// Lower numbers = higher priority.
func calculateUnblockedPriority(issue beads.Issue, role string) int {
	// Base priority for unblocked signals - high because this is actionable work
	basePriority := 40

	// Adjust based on beads priority (P0 = 0, P1 = 1, etc.)
	// Higher beads priority = lower attention priority value (more urgent)
	basePriority += issue.Priority * 10

	// Role-aware adjustments
	switch role {
	case "human":
		// Humans need to know about unblocked work
		return basePriority

	case "orchestrator":
		// Orchestrators actively spawn unblocked work - high priority
		return basePriority - 10

	case "daemon":
		// Daemon auto-spawns if labels match - high priority
		return basePriority - 5

	default:
		return basePriority
	}
}
