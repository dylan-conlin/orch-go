package daemon

import (
	"fmt"
)

// resolveIssueQuerier returns the effective IssueQuerier.
// If Issues is set, returns it directly.
// If only ProjectRegistry is set (no custom Issues), wraps it into a defaultIssueQuerier.
func (d *Daemon) resolveIssueQuerier() IssueQuerier {
	if d.Issues != nil {
		// If it's the default querier, update its registry pointer lazily
		if dq, ok := d.Issues.(*defaultIssueQuerier); ok {
			dq.registry = d.ProjectRegistry
		}
		return d.Issues
	}
	return &defaultIssueQuerier{registry: d.ProjectRegistry}
}

// issueMatchesLabel checks if an issue matches the daemon's configured label filter.
// Recognizes equivalent labels (e.g., triage:approved is equivalent to triage:ready)
// so that human-approved items are also spawnable by the daemon.
func (d *Daemon) issueMatchesLabel(issue Issue) bool {
	if d.Config.Label == "" {
		return true
	}
	return issue.HasAnyLabel(SpawnableLabelsFor(d.Config.Label)...)
}

// NextIssue returns the next spawnable issue from the queue.
// Returns nil if no spawnable issues are available.
// Issues are sorted by priority (0 = highest priority).
// If a label filter is configured, only issues with that label are considered.
func (d *Daemon) NextIssue() (*Issue, error) {
	return d.NextIssueExcluding(nil)
}

// NextIssueExcluding returns the next spawnable issue from the queue,
// excluding any issues in the skip set. This allows the daemon to skip
// issues that failed to spawn (e.g., due to failure report gate) and
// continue processing other issues in the queue.
//
// Returns nil if no spawnable issues are available after excluding skipped ones.
// Issues are sorted by priority (0 = highest priority).
// If a label filter is configured, only issues with that label are considered.
//
// Epic child expansion: When an epic has the required label (e.g., triage:ready),
// its children are automatically included in the spawn queue even if they don't
// have the label themselves. This implements the user mental model that labeling
// an epic means "process this entire epic".
func (d *Daemon) NextIssueExcluding(skip map[string]bool) (*Issue, error) {
	issues, err := d.resolveIssueQuerier().ListReadyIssues()
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	if d.Config.Verbose {
		fmt.Printf("  DEBUG: Found %d open issues\n", len(issues))
	}

	// Coordination: prioritize issues (epic expansion, focus boost, sort, interleave)
	issues, epicChildIDs, err := d.PrioritizeIssues(issues)
	if err != nil {
		return nil, err
	}

	// Compliance: filter each issue through compliance checks
	for _, issue := range issues {
		filter := d.CheckIssueCompliance(issue, skip, epicChildIDs)
		if !filter.Passed {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (%s)\n", issue.ID, filter.Reason)
			}
			continue
		}
		if d.Config.Verbose {
			fmt.Printf("  DEBUG: Selected %s (type=%s, labels=%v)\n", issue.ID, issue.IssueType, issue.Labels)
		}
		return &issue, nil
	}

	return nil, nil
}

// expandTriageReadyEpics finds epics with the required label and includes their children.
// Returns the expanded issue list and a map of issue IDs that are epic children
// (for label exemption in NextIssueExcluding).
func (d *Daemon) expandTriageReadyEpics(issues []Issue) ([]Issue, map[string]bool, error) {
	epicChildIDs := make(map[string]bool)

	// If no label filter is set, no expansion needed
	if d.Config.Label == "" {
		return issues, epicChildIDs, nil
	}

	// Find epics with the required label
	var epicsToExpand []string
	existingIDs := make(map[string]bool)
	for _, issue := range issues {
		existingIDs[issue.ID] = true
		if issue.IssueType == "epic" && d.issueMatchesLabel(issue) {
			epicsToExpand = append(epicsToExpand, issue.ID)
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Found triage:ready epic %s, will include children\n", issue.ID)
			}
		}
	}

	// No epics to expand
	if len(epicsToExpand) == 0 {
		return issues, epicChildIDs, nil
	}

	// Expand each epic by fetching its children
	querier := d.resolveIssueQuerier()
	for _, epicID := range epicsToExpand {
		children, err := querier.ListEpicChildren(epicID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to list children of epic %s: %w", epicID, err)
		}

		for _, child := range children {
			// Skip closed children - they shouldn't be spawned
			if child.Status == "closed" {
				if d.Config.Verbose {
					fmt.Printf("  DEBUG: Skipping closed epic child %s (from parent %s)\n", child.ID, epicID)
				}
				continue
			}
			// Only add if not already in the list
			if !existingIDs[child.ID] {
				issues = append(issues, child)
				existingIDs[child.ID] = true
				epicChildIDs[child.ID] = true
				if d.Config.Verbose {
					fmt.Printf("  DEBUG: Added epic child %s (from parent %s)\n", child.ID, epicID)
				}
			} else {
				// Already in list, but mark as epic child for label exemption
				epicChildIDs[child.ID] = true
			}
		}
	}

	return issues, epicChildIDs, nil
}
