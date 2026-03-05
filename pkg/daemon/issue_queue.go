// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"strings"
)

// Issue represents a beads issue for processing.
type Issue struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    int      `json:"priority"`
	Status      string   `json:"status"`
	IssueType   string   `json:"issue_type"`
	Labels      []string `json:"labels"`
	ProjectDir  string   `json:"-"` // Source project dir (empty = current project)
}

// HasLabel checks if an issue has a specific label.
func (i *Issue) HasLabel(label string) bool {
	for _, l := range i.Labels {
		if strings.EqualFold(l, label) {
			return true
		}
	}
	return false
}

// projectFromIssueID extracts the project prefix from a beads issue ID.
// Issue IDs follow the format "prefix-hash" (e.g., "orch-go-1169", "pw-5678").
// Returns the text before the last hyphen segment.
func projectFromIssueID(id string) string {
	lastDash := strings.LastIndex(id, "-")
	if lastDash <= 0 {
		return id
	}
	return id[:lastDash]
}

// interleaveByProject reorders a priority-sorted issue slice so that issues
// from different projects alternate within each priority level (round-robin).
// Higher priority groups still come before lower priority groups.
func interleaveByProject(issues []Issue) []Issue {
	if len(issues) <= 1 {
		return issues
	}

	// Group issues by priority, preserving order within each group.
	type priorityGroup struct {
		priority int
		issues   []Issue
	}
	var groups []priorityGroup
	groupIdx := -1
	lastPriority := -1

	for _, iss := range issues {
		if iss.Priority != lastPriority {
			groups = append(groups, priorityGroup{priority: iss.Priority})
			groupIdx++
			lastPriority = iss.Priority
		}
		groups[groupIdx].issues = append(groups[groupIdx].issues, iss)
	}

	// Interleave within each priority group by project.
	var result []Issue
	for _, g := range groups {
		result = append(result, roundRobinByProject(g.issues)...)
	}
	return result
}

// roundRobinByProject interleaves issues from different projects.
// Issues from each project are placed into per-project queues (preserving
// their relative order), then drawn round-robin.
func roundRobinByProject(issues []Issue) []Issue {
	if len(issues) <= 1 {
		return issues
	}

	// Build per-project queues, preserving insertion order of projects.
	var projectOrder []string
	projectQueues := make(map[string][]Issue)
	for _, iss := range issues {
		proj := projectFromIssueID(iss.ID)
		if _, exists := projectQueues[proj]; !exists {
			projectOrder = append(projectOrder, proj)
		}
		projectQueues[proj] = append(projectQueues[proj], iss)
	}

	// Single project: no interleaving needed.
	if len(projectOrder) <= 1 {
		return issues
	}

	// Round-robin across projects.
	result := make([]Issue, 0, len(issues))
	idx := make(map[string]int) // current index per project
	for len(result) < len(issues) {
		for _, proj := range projectOrder {
			qi := idx[proj]
			if qi < len(projectQueues[proj]) {
				result = append(result, projectQueues[proj][qi])
				idx[proj] = qi + 1
			}
		}
	}
	return result
}

// FilterIssues returns issues that pass all filter criteria.
// This is a helper for applying consistent filtering logic.
type IssueFilter struct {
	// Label filters to issues with this label (empty = no filter).
	Label string
	// IncludeBlocked includes issues with status "blocked" (default: false).
	IncludeBlocked bool
	// IncludeInProgress includes issues with status "in_progress" (default: false).
	IncludeInProgress bool
	// Skip is a set of issue IDs to skip.
	Skip map[string]bool
}

// Filter returns true if the issue passes all filter criteria.
func (f *IssueFilter) Filter(issue Issue) bool {
	// Skip issues in the skip set
	if f.Skip != nil && f.Skip[issue.ID] {
		return false
	}

	// Skip non-spawnable types
	if !IsSpawnableType(issue.IssueType) {
		return false
	}

	// Skip blocked issues (unless included)
	if !f.IncludeBlocked && issue.Status == "blocked" {
		return false
	}

	// Skip in_progress issues (unless included)
	if !f.IncludeInProgress && issue.Status == "in_progress" {
		return false
	}

	// Skip issues without required label (if filter is set)
	if f.Label != "" && !issue.HasLabel(f.Label) {
		return false
	}

	return true
}
