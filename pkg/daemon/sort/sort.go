// Package sort provides named sort strategies for daemon issue prioritization.
//
// The daemon processes issues from a queue, and the sort strategy determines
// which issue to process next. Each strategy optimizes for a different
// operational goal (e.g., unblocking throughput vs minimizing context switches).
//
// Sort strategies integrate cross-system data via SortContext, which provides
// cached frontier/leverage data and session context that beads alone cannot provide.
package sort

import (
	"fmt"
	"strings"
)

// Issue represents a beads issue for sorting.
// This mirrors daemon.Issue to avoid circular imports.
type Issue struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    int      `json:"priority"`
	Status      string   `json:"status"`
	IssueType   string   `json:"issue_type"`
	Labels      []string `json:"labels"`
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

// LeverageInfo contains pre-computed leverage data for an issue.
type LeverageInfo struct {
	// TotalLeverage is the count of transitively unblocked issues.
	TotalLeverage int
	// WouldUnblock is the list of directly unblocked issue IDs.
	WouldUnblock []string
}

// SortContext provides pre-computed cross-system data for sort strategies.
// Computed once per daemon poll cycle and shared across sort calls.
type SortContext struct {
	// Leverage maps issue IDs to their pre-computed leverage info.
	// Populated from frontier.CalculateFrontier() results.
	// May be nil if frontier data is unavailable.
	Leverage map[string]*LeverageInfo

	// ActiveAreas maps area labels to count of active agents working in that area.
	// Populated from current agent sessions.
	// May be nil if session data is unavailable.
	ActiveAreas map[string]int

	// LastCompletedArea is the area label of the most recently completed issue.
	// Used by Flow State strategy for context locality.
	LastCompletedArea string
}

// Strategy defines a named sort strategy for daemon issue prioritization.
type Strategy interface {
	// Name returns the strategy identifier (e.g., "priority", "unblock").
	Name() string

	// Sort orders issues by this strategy's criteria.
	// The ctx parameter provides cached cross-system data (frontier leverage,
	// session context). ctx may be nil, in which case the strategy should
	// degrade gracefully (e.g., fall back to priority-only sorting).
	//
	// Returns a new sorted slice; does not modify the input.
	Sort(issues []Issue, ctx *SortContext) []Issue
}

// Get returns the Strategy for the given mode name.
// Returns an error if the mode is not recognized.
func Get(mode string) (Strategy, error) {
	switch strings.ToLower(mode) {
	case "priority", "":
		return &PriorityStrategy{}, nil
	case "unblock":
		return &UnblockStrategy{}, nil
	default:
		return nil, fmt.Errorf("unknown sort mode %q (available: priority, unblock)", mode)
	}
}

// Modes returns the list of available sort mode names.
func Modes() []string {
	return []string{"priority", "unblock"}
}
