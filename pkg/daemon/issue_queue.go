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
