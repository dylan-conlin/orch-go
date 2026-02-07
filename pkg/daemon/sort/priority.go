package sort

import (
	gosort "sort"
)

// PriorityStrategy sorts issues by priority (lower number = higher priority).
// This is the current default behavior of the daemon.
type PriorityStrategy struct{}

// Name returns "priority".
func (s *PriorityStrategy) Name() string {
	return "priority"
}

// Sort orders issues by priority ascending (P0 first, then P1, P2, etc.).
// This replicates the existing daemon_queue.go sort behavior exactly.
func (s *PriorityStrategy) Sort(issues []Issue, ctx *SortContext) []Issue {
	result := make([]Issue, len(issues))
	copy(result, issues)

	gosort.SliceStable(result, func(i, j int) bool {
		return result[i].Priority < result[j].Priority
	})

	return result
}
