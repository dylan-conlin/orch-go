package sort

import (
	gosort "sort"
)

// UnblockStrategy sorts issues to maximize throughput by clearing bottlenecks.
// Issues that would unblock the most downstream work are prioritized.
//
// Sort order:
//  1. dependency_leverage descending (highest leverage first)
//  2. authority_level: daemon-traversable types before orchestrator/human types
//  3. priority ascending (tiebreaker)
//
// When frontier data is unavailable (ctx is nil or has no leverage data),
// falls back to priority-only sorting (graceful degradation).
type UnblockStrategy struct{}

// Name returns "unblock".
func (s *UnblockStrategy) Name() string {
	return "unblock"
}

// Sort orders issues by leverage (descending), then authority level, then priority.
func (s *UnblockStrategy) Sort(issues []Issue, ctx *SortContext) []Issue {
	result := make([]Issue, len(issues))
	copy(result, issues)

	// If no context or no leverage data, fall back to priority sort
	if ctx == nil || ctx.Leverage == nil || len(ctx.Leverage) == 0 {
		gosort.SliceStable(result, func(i, j int) bool {
			return result[i].Priority < result[j].Priority
		})
		return result
	}

	gosort.SliceStable(result, func(i, j int) bool {
		li := getLeverage(result[i].ID, ctx)
		lj := getLeverage(result[j].ID, ctx)

		// Primary: leverage descending (higher leverage = more unblocking potential)
		if li != lj {
			return li > lj
		}

		// Secondary: authority level (daemon-traversable first)
		ai := authorityLevel(result[i])
		aj := authorityLevel(result[j])
		if ai != aj {
			return ai < aj
		}

		// Tertiary: priority ascending (lower number = higher priority)
		return result[i].Priority < result[j].Priority
	})

	return result
}

// getLeverage returns the total leverage for an issue from the sort context.
// Returns 0 if no leverage data is available for the issue (neutral score).
func getLeverage(issueID string, ctx *SortContext) int {
	if ctx == nil || ctx.Leverage == nil {
		return 0
	}
	info, ok := ctx.Leverage[issueID]
	if !ok || info == nil {
		return 0
	}
	return info.TotalLeverage
}

// authorityLevel returns a numeric authority level for sorting.
// Lower values are "easier" for the daemon to process autonomously.
//
//	0 = daemon-traversable (task, bug, factual questions)
//	1 = needs some judgment (feature, investigation)
//	2 = needs human/orchestrator (judgment questions)
func authorityLevel(issue Issue) int {
	switch issue.IssueType {
	case "task", "bug":
		return 0
	case "feature", "investigation":
		return 1
	case "question":
		// Factual questions are daemon-traversable
		if issue.HasLabel("subtype:factual") {
			return 0
		}
		// Judgment questions need orchestrator/human
		return 2
	default:
		return 1 // neutral for unknown types
	}
}
