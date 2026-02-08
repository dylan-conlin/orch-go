// daemon_queue.go contains issue queue management, filtering, rejection checks, and preview.
package daemon

import (
	"fmt"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

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
	issues, err := d.listIssuesFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	if d.Config.Verbose {
		fmt.Printf("  DEBUG: Found %d open issues\n", len(issues))
	}

	// Expand triage:ready epics by including their children.
	// This allows "label the epic" to mean "process the entire epic".
	issues, epicChildIDs := d.expandTriageReadyEpics(issues)

	// Apply the active sort strategy (priority, unblock, etc.)
	issues = d.SortIssues(issues)

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
			// Emit telemetry event when SpawnedIssueTracker blocks spawn
			if d.EventLogger != nil {
				_ = d.EventLogger.LogDedupBlocked(map[string]interface{}{
					"beads_id":    issue.ID,
					"dedup_layer": "spawned_tracker",
					"reason":      "Issue recently spawned, awaiting status update (6h TTL)",
				})
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
		// Skip in_progress issues ONLY if there's an active session working on them.
		// If no active session exists, the user may have marked it in_progress to release it TO the daemon.
		if issue.Status == "in_progress" {
			if HasExistingSessionForBeadsID(issue.ID) {
				if d.Config.Verbose {
					fmt.Printf("  DEBUG: Skipping %s (in_progress with active session)\n", issue.ID)
				}
				continue
			}
			// No active session - issue was likely released to daemon, proceed with spawn
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Including %s (in_progress but no active session)\n", issue.ID)
			}
		}
		// Skip issues without required label (if filter is set)
		// BUT: Children of triage:ready epics are exempt from this check
		// (they inherit triage-ready status from their parent)
		if d.Config.Label != "" && !issue.HasLabel(d.Config.Label) {
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
		// Grace period check: skip issues recently seen for the first time.
		if d.Config.GracePeriod > 0 && d.InGracePeriod(issue.ID) {
			if d.Config.Verbose {
				remaining := d.Config.GracePeriod - time.Since(d.firstSeen[issue.ID])
				fmt.Printf("  DEBUG: Skipping %s (in grace period, %s remaining)\n", issue.ID, remaining.Round(time.Second))
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

// QueueDiagnosticsForIssues computes queued-work diagnostics for dashboard status.
// It explains why currently ready issues are not spawning yet.
func (d *Daemon) QueueDiagnosticsForIssues(readyIssues []Issue) QueueDiagnostics {
	diagnostics := QueueDiagnostics{
		Queued: len(readyIssues),
	}

	if len(readyIssues) == 0 {
		return diagnostics
	}

	spawnable := 0
	for _, issue := range readyIssues {
		if d.ProcessedCache != nil && !d.ProcessedCache.ShouldProcess(issue.ID) {
			diagnostics.ProcessedCache++
			continue
		}

		if d.InGracePeriodWithoutRecording(issue.ID) {
			diagnostics.GracePeriod++
			continue
		}

		spawnable++
	}

	diagnostics.Spawnable = spawnable
	availableSlots := d.AvailableSlots()
	if spawnable > availableSlots {
		diagnostics.WaitingForSlots = spawnable - availableSlots
	}

	return diagnostics
}

// expandTriageReadyEpics finds epics with the required label and includes their children.
// Returns the expanded issue list and a map of issue IDs that are epic children
// (for label exemption in NextIssueExcluding).
func (d *Daemon) expandTriageReadyEpics(issues []Issue) ([]Issue, map[string]bool) {
	epicChildIDs := make(map[string]bool)

	// If no label filter is set, no expansion needed
	if d.Config.Label == "" {
		return issues, epicChildIDs
	}

	// Find epics with the required label
	var epicsToExpand []string
	existingIDs := make(map[string]bool)
	for _, issue := range issues {
		existingIDs[issue.ID] = true
		if issue.IssueType == "epic" && issue.HasLabel(d.Config.Label) {
			epicsToExpand = append(epicsToExpand, issue.ID)
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Found triage:ready epic %s, will include children\n", issue.ID)
			}
		}
	}

	// No epics to expand
	if len(epicsToExpand) == 0 {
		return issues, epicChildIDs
	}

	// Expand each epic by fetching its children
	listChildren := d.listEpicChildrenFunc
	if listChildren == nil {
		listChildren = ListEpicChildren
	}
	for _, epicID := range epicsToExpand {
		children, err := listChildren(epicID)
		if err != nil {
			if d.Config.Verbose {
				fmt.Printf("  DEBUG: Warning: could not list children of epic %s: %v\n", epicID, err)
			}
			continue
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

	return issues, epicChildIDs
}

// Preview shows what would be processed next without actually processing.
// It also collects all rejected issues with their rejection reasons.
func (d *Daemon) Preview() (*PreviewResult, error) {
	result := &PreviewResult{}

	// Check rate limit status
	if d.RateLimiter != nil {
		canSpawn, count, msg := d.RateLimiter.CanSpawn()
		result.RateLimited = !canSpawn
		if d.RateLimiter.MaxPerHour > 0 {
			result.RateStatus = fmt.Sprintf("%d/%d spawns in last hour", count, d.RateLimiter.MaxPerHour)
		}
		if !canSpawn {
			result.Message = msg
			// Still collect rejected issues even if rate limited
		}
	}

	// Get all issues and categorize them
	issues, err := d.listIssuesFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	// Expand triage:ready epics by including their children
	issues, epicChildIDs := d.expandTriageReadyEpics(issues)

	// Apply the active sort strategy
	issues = d.SortIssues(issues)

	var spawnable *Issue
	for _, issue := range issues {
		// Check each rejection reason in order and collect all rejected issues
		reason := d.checkRejectionReasonWithEpicChildren(issue, epicChildIDs)
		if reason != "" {
			result.RejectedIssues = append(result.RejectedIssues, RejectedIssue{
				Issue:  issue,
				Reason: reason,
			})
			continue
		}

		// Found a spawnable issue - take the first one (highest priority)
		if spawnable == nil {
			issueCopy := issue
			spawnable = &issueCopy
		}
	}

	// If rate limited, we still collected rejected issues but can't spawn
	if result.RateLimited {
		return result, nil
	}

	if spawnable == nil {
		result.Message = "No spawnable issues in queue"
		return result, nil
	}

	skill, err := InferSkillFromIssue(spawnable)
	if err != nil {
		return nil, fmt.Errorf("failed to infer skill: %w", err)
	}

	result.Issue = spawnable
	result.Skill = skill

	// Check for hotspot warnings if checker is configured
	if d.HotspotChecker != nil {
		result.HotspotWarnings = CheckHotspotsForIssue(spawnable, d.HotspotChecker)
	}

	return result, nil
}

// checkRejectionReason checks if an issue should be rejected and returns the reason.
// Returns empty string if the issue is spawnable.
// This is the legacy version that doesn't consider epic children.
func (d *Daemon) checkRejectionReason(issue Issue) string {
	return d.checkRejectionReasonWithEpicChildren(issue, nil)
}

// checkRejectionReasonWithEpicChildren checks if an issue should be rejected and returns the reason.
// The epicChildIDs map contains IDs of issues that are children of triage:ready epics.
// These children are exempt from the label requirement check.
// Returns empty string if the issue is spawnable.
func (d *Daemon) checkRejectionReasonWithEpicChildren(issue Issue, epicChildIDs map[string]bool) string {
	// Check for empty/missing type first (the main problem case from the bug report)
	if issue.IssueType == "" {
		return "missing type (required for skill inference)"
	}

	// Check for non-spawnable type
	// Note: Epics with triage:ready are not spawnable themselves, but their children are.
	// The message is informative to explain why epics are rejected.
	if !IsSpawnableType(issue.IssueType) {
		if issue.IssueType == "epic" && issue.HasLabel(d.Config.Label) {
			return fmt.Sprintf("type 'epic' not spawnable (children will be processed instead)")
		}
		return fmt.Sprintf("type '%s' not spawnable (must be bug/feature/task/investigation/question)", issue.IssueType)
	}

	// Check for blocked status
	if issue.Status == "blocked" {
		return "status is blocked"
	}

	// Check for in_progress status - only reject if there's an active session
	if issue.Status == "in_progress" {
		if HasExistingSessionForBeadsID(issue.ID) {
			return "status is in_progress (active session found)"
		}
		// No active session - issue was likely released to daemon, spawnable
	}

	// Check for missing required label
	// Epic children are exempt from this check - they inherit triage status from parent
	if d.Config.Label != "" && !issue.HasLabel(d.Config.Label) {
		if epicChildIDs == nil || !epicChildIDs[issue.ID] {
			return fmt.Sprintf("missing label '%s'", d.Config.Label)
		}
		// Epic child - exempt from label requirement
	}

	// Check for blocking dependencies
	blockers, err := beads.CheckBlockingDependencies(issue.ID)
	if err == nil && len(blockers) > 0 {
		var blockerIDs []string
		for _, b := range blockers {
			blockerIDs = append(blockerIDs, fmt.Sprintf("%s (%s)", b.ID, b.Status))
		}
		return fmt.Sprintf("blocked by dependencies: %s", strings.Join(blockerIDs, ", "))
	}

	// Grace period check: issue was recently added to the queue
	if d.Config.GracePeriod > 0 && d.InGracePeriod(issue.ID) {
		remaining := d.Config.GracePeriod - time.Since(d.firstSeen[issue.ID])
		return fmt.Sprintf("in grace period (%s remaining)", remaining.Round(time.Second))
	}
	return "" // Spawnable
}

// FormatPreview formats an issue for preview display.
func FormatPreview(issue *Issue) string {
	return fmt.Sprintf(`Issue:    %s
Title:    %s
Type:     %s
Priority: P%d
Status:   %s
Description: %s`,
		issue.ID,
		issue.Title,
		issue.IssueType,
		issue.Priority,
		issue.Status,
		truncate(issue.Description, 100),
	)
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// FormatRejectedIssues formats rejected issues for display.
func FormatRejectedIssues(rejected []RejectedIssue) string {
	if len(rejected) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\nRejected issues:\n")
	for _, r := range rejected {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", r.Issue.ID, r.Reason))
	}
	return sb.String()
}
