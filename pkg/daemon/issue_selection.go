package daemon

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
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

	// Expand triage:ready epics by including their children.
	// This allows "label the epic" to mean "process the entire epic".
	issues, epicChildIDs, err := d.expandTriageReadyEpics(issues)
	if err != nil {
		return nil, fmt.Errorf("failed to expand epics: %w", err)
	}

	// Apply focus boost: issues from focused projects get priority boost
	if d.FocusGoal != "" && d.FocusBoostAmount > 0 {
		issues = applyFocusBoost(issues, d.FocusGoal, d.FocusBoostAmount, d.ProjectDirNames)
	}

	// Sort by priority (lower number = higher priority)
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Priority < issues[j].Priority
	})

	// Round-robin across projects within each priority level.
	// This prevents one project from monopolizing all slots when
	// multiple projects have issues at the same priority.
	issues = interleaveByProject(issues)

	for _, issue := range issues {
		// Skip issues in the skip set (failed to spawn this cycle)
		if skip != nil && skip[issue.ID] {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (failed to spawn this cycle)\n", issue.ID)
			}
			continue
		}
		// Skip issues that have been recently spawned but status not yet updated.
		// This prevents the race condition where the daemon spawns duplicate agents
		// because beads status update hasn't propagated yet.
		if d.SpawnedIssues != nil && d.SpawnedIssues.IsSpawned(issue.ID) {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (recently spawned, awaiting status update)\n", issue.ID)
			}
			continue
		}
		// Skip non-spawnable types
		if !IsSpawnableType(issue.IssueType) {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (type %s not spawnable)\n", issue.ID, issue.IssueType)
			}
			continue
		}
		// Skip blocked issues
		if issue.Status == "blocked" {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (blocked)\n", issue.ID)
			}
			continue
		}
		// Skip in_progress issues (already being worked on)
		if issue.Status == "in_progress" {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (already in_progress)\n", issue.ID)
			}
			continue
		}
		// Skip issues already processed by the completion loop.
		// daemon:ready-review = waiting for orchestrator review (completion verified)
		// daemon:verification-failed = exhausted verification retries (deferred for human)
		// Without this check, completed issues with triage:ready still in their labels
		// re-enter the spawn queue, causing duplicate spawns on stale Phase: Complete.
		if issue.HasAnyLabel(LabelReadyReview, LabelVerificationFailed) {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Skipping %s (has daemon completion label)\n", issue.ID)
			}
			continue
		}
		// Skip issues without required label (if filter is set)
		// Recognizes equivalent labels (e.g., triage:approved ≈ triage:ready).
		// BUT: Children of triage:ready epics are exempt from this check
		// (they inherit triage-ready status from their parent)
		if !d.issueMatchesLabel(issue) {
			// Check if this issue is a child of a triage:ready epic
			if _, isEpicChild := epicChildIDs[issue.ID]; !isEpicChild {
				if d.Config.Verbose {
					fmt.Printf("  DEBUG: Skipping %s (missing label %s, has %v)\n", issue.ID, d.Config.Label, issue.Labels)
				}
				continue
			}
			// Epic child - proceed even without label
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Including %s (epic child, inherits triage status from parent)\n", issue.ID)
			}
		}
		// Skip issues with blocking dependencies (open/in_progress dependencies)
		blockers, err := beads.CheckBlockingDependencies(issue.ID)
		if err != nil {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Warning: could not check dependencies for %s: %v\n", issue.ID, err)
			}
			// Continue checking - don't skip issue just because we can't check dependencies
		} else if len(blockers) > 0 {
			if d.Config.Verbose {
				var blockerIDs []string
				for _, b := range blockers {
					blockerIDs = append(blockerIDs, fmt.Sprintf("%s (%s)", b.ID, b.Status))
				}
				fmt.Printf("  DEBUG: Skipping %s (blocked by dependencies: %s)\n", issue.ID, strings.Join(blockerIDs, ", "))
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
