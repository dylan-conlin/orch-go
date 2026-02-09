// sort_bridge.go provides conversion between daemon.Issue and sort.Issue,
// and builds SortContext from daemon state for use by sort strategies.
package daemon

import (
	"fmt"
	gosort "sort"

	daemonsort "github.com/dylan-conlin/orch-go/pkg/daemon/sort"
	"github.com/dylan-conlin/orch-go/pkg/frontier"
)

// toSortIssues converts a slice of daemon.Issue to sort.Issue.
func toSortIssues(issues []Issue) []daemonsort.Issue {
	result := make([]daemonsort.Issue, len(issues))
	for i, issue := range issues {
		result[i] = daemonsort.Issue{
			ID:          issue.ID,
			Title:       issue.Title,
			Description: issue.Description,
			Priority:    issue.Priority,
			Status:      issue.Status,
			IssueType:   issue.IssueType,
			Labels:      issue.Labels,
		}
	}
	return result
}

// fromSortIssues converts a slice of sort.Issue back to daemon.Issue.
func fromSortIssues(issues []daemonsort.Issue) []Issue {
	result := make([]Issue, len(issues))
	for i, issue := range issues {
		result[i] = Issue{
			ID:          issue.ID,
			Title:       issue.Title,
			Description: issue.Description,
			Priority:    issue.Priority,
			Status:      issue.Status,
			IssueType:   issue.IssueType,
			Labels:      issue.Labels,
		}
	}
	return result
}

// buildSortContext creates a SortContext from the daemon's cached frontier state.
// Returns nil if no frontier data is available (strategies will degrade gracefully).
func (d *Daemon) buildSortContext() *daemonsort.SortContext {
	if d.CachedFrontier == nil {
		return nil
	}
	return buildSortContextFromFrontier(d.CachedFrontier)
}

// buildSortContextFromFrontier creates a SortContext from a FrontierState.
func buildSortContextFromFrontier(fs *frontier.FrontierState) *daemonsort.SortContext {
	if fs == nil {
		return nil
	}

	ctx := &daemonsort.SortContext{
		Leverage: make(map[string]*daemonsort.LeverageInfo),
	}

	// Map blocked issues to leverage info.
	// The frontier computes leverage for blocked issues (what completing their
	// blockers would unblock). For sort purposes, we need leverage on the
	// ready/spawnable issues — specifically, which ready issues would unblock
	// the most downstream work if completed.
	//
	// Frontier's blocked issues have leverage computed per-blocker. We need
	// to map this back: for each ready issue, sum up how many blocked issues
	// it would unblock if completed.
	readyLeverage := make(map[string]*daemonsort.LeverageInfo)

	for _, blocked := range fs.Blocked {
		if blocked.Issue == nil {
			continue
		}
		// Each blocked issue's dependencies are the blockers.
		// The leverage (WouldUnblock, TotalLeverage) is about what the blocked
		// issue would unblock if IT were completed. But we need the inverse:
		// what would completing a ready issue unblock?
		//
		// Approach: For each blocked issue, check its BlockedBy list.
		// Each blocker gets credit for potentially unblocking this chain.
		for _, blockerID := range blocked.Issue.BlockedBy {
			info, ok := readyLeverage[blockerID]
			if !ok {
				info = &daemonsort.LeverageInfo{}
				readyLeverage[blockerID] = info
			}
			info.WouldUnblock = append(info.WouldUnblock, blocked.Issue.ID)
			info.TotalLeverage += 1 + blocked.TotalLeverage
		}
	}

	ctx.Leverage = readyLeverage
	return ctx
}

// SortIssues applies the daemon's active sort strategy to a list of issues.
// This is the main entry point used by NextIssueExcluding and Preview.
// If no strategy is set (e.g., in tests that construct Daemon directly),
// falls back to priority-based sorting to maintain backward compatibility.
func (d *Daemon) SortIssues(issues []Issue) []Issue {
	strategy := d.SortStrategy
	if strategy == nil {
		strategy = &daemonsort.PriorityStrategy{}
	}

	sortIssues := toSortIssues(issues)
	ctx := d.buildSortContext()
	sorted := strategy.Sort(sortIssues, ctx)

	// Preserve all fields from the original issues (including metadata like
	// UpdatedAt) while applying the sorted order returned by the strategy.
	byID := make(map[string]Issue, len(issues))
	for _, issue := range issues {
		byID[issue.ID] = issue
	}

	result := make([]Issue, 0, len(sorted))
	for _, sortedIssue := range sorted {
		if original, ok := byID[sortedIssue.ID]; ok {
			result = append(result, original)
			continue
		}

		result = append(result, Issue{
			ID:          sortedIssue.ID,
			Title:       sortedIssue.Title,
			Description: sortedIssue.Description,
			Priority:    sortedIssue.Priority,
			Status:      sortedIssue.Status,
			IssueType:   sortedIssue.IssueType,
			Labels:      sortedIssue.Labels,
		})
	}

	return result
}

// SortCrossProjectIssues applies the daemon's active sort strategy to
// a list of CrossProjectIssue, sorting by the Issue field.
func (d *Daemon) SortCrossProjectIssues(issues []CrossProjectIssue) []CrossProjectIssue {
	if len(issues) == 0 {
		return issues
	}

	strategy := d.SortStrategy
	if strategy == nil {
		strategy = &daemonsort.PriorityStrategy{}
	}

	// Extract issues, sort them, then build an order map
	plainIssues := make([]Issue, len(issues))
	for i, cpi := range issues {
		plainIssues[i] = cpi.Issue
	}

	sortIssues := toSortIssues(plainIssues)
	ctx := d.buildSortContext()
	sorted := strategy.Sort(sortIssues, ctx)

	// Build position map: issue ID -> sorted position
	sortMap := make(map[string]int, len(sorted))
	for i, s := range sorted {
		sortMap[s.ID] = i
	}

	// Reorder cross-project issues to match sort order
	result := make([]CrossProjectIssue, len(issues))
	copy(result, issues)

	gosort.SliceStable(result, func(i, j int) bool {
		oi, okI := sortMap[result[i].Issue.ID]
		oj, okJ := sortMap[result[j].Issue.ID]
		if !okI {
			oi = len(sorted)
		}
		if !okJ {
			oj = len(sorted)
		}
		return oi < oj
	})

	return result
}

// RefreshFrontierCache updates the cached frontier state.
// Called once per poll cycle before issue selection.
// Errors are logged but don't prevent operation — sort strategies
// degrade gracefully when frontier data is unavailable.
func (d *Daemon) RefreshFrontierCache() {
	fs, err := frontier.CalculateFrontier()
	if err != nil {
		if d.Config.Verbose {
			fmt.Printf("  DEBUG: Failed to refresh frontier cache: %v\n", err)
		}
		// Keep stale cache rather than clearing it
		return
	}
	d.CachedFrontier = fs
}
