// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// RejectedIssue captures why an issue was rejected for spawning.
type RejectedIssue struct {
	Issue  Issue  // The rejected issue
	Reason string // Human-readable rejection reason
}

// PreviewResult contains the result of a preview operation.
type PreviewResult struct {
	Issue           *Issue
	Skill           string
	Model           string           // Inferred model alias (e.g., "opus", "sonnet")
	Message         string
	RateLimited     bool             // True if rate limit would prevent spawning
	RateStatus      string           // Rate limit status message (e.g., "5/20 spawns in last hour")
	HotspotWarnings []HotspotWarning // Warnings about hotspot areas this issue may touch
	RejectedIssues  []RejectedIssue  // Issues that were rejected with reasons
}

// HasHotspotWarnings returns true if there are any hotspot warnings.
func (r *PreviewResult) HasHotspotWarnings() bool {
	return len(r.HotspotWarnings) > 0
}

// HasCriticalHotspots returns true if any hotspot warning is critical (score >= 10).
func (r *PreviewResult) HasCriticalHotspots() bool {
	for _, w := range r.HotspotWarnings {
		if w.IsCritical() {
			return true
		}
	}
	return false
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
	issues, epicChildIDs, err := d.expandTriageReadyEpics(issues)
	if err != nil {
		return nil, fmt.Errorf("failed to expand epics: %w", err)
	}

	// Sort by priority (lower number = higher priority)
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Priority < issues[j].Priority
	})

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
	result.Model = InferModelFromSkill(skill)

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
		return fmt.Sprintf("type '%s' not spawnable (must be bug/feature/task/investigation)", issue.IssueType)
	}

	// Check for blocked status
	if issue.Status == "blocked" {
		return "status is blocked"
	}

	// Check for in_progress status
	if issue.Status == "in_progress" {
		return "status is in_progress (already being worked on)"
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
