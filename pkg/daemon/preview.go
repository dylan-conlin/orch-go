// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/daemonconfig"
)

// RejectedIssue captures why an issue was rejected for spawning.
type RejectedIssue struct {
	Issue  Issue  // The rejected issue
	Reason string // Human-readable rejection reason
}

// PreviewResult contains the result of a preview operation.
type PreviewResult struct {
	Issue              *Issue
	Skill              string
	Model              string           // Inferred model alias (e.g., "opus", "sonnet", "gpt-5.4")
	ModelRouteReason   string           // Explains why this model was chosen
	Message            string
	RateLimited        bool             // True if rate limit would prevent spawning
	RateStatus         string           // Rate limit status message (e.g., "5/20 spawns in last hour")
	HotspotWarnings       []HotspotWarning       // Warnings about hotspot areas this issue may touch
	ChannelHealthWarnings []ChannelHealthWarning // Skills with rework=0 + high completions (silent channel)
	RejectedIssues        []RejectedIssue        // Issues that were rejected with reasons
	SpawnableCount        int                    // Total issues passing compliance (not just the displayed one)
	ArchitectEscalated    bool                   // True if skill would be escalated from impl to architect
	FocusBoosted          bool                   // True if selected issue was boosted by focus
	FocusGoal             string                 // Current focus goal (if any)
	ModelRoute            *daemonconfig.ModelRouteResult // Model routing decision details (nil if no routing config)
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
	issues, err := d.resolveIssueQuerier().ListReadyIssues()
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	// Prioritize: epic expansion, focus boost, allocation scoring, interleaving.
	// Uses the same PrioritizeIssues() as the poll loop for consistent ordering.
	issues, epicChildIDs, err := d.PrioritizeIssues(issues)
	if err != nil {
		return nil, fmt.Errorf("failed to prioritize issues: %w", err)
	}

	if d.FocusGoal != "" {
		result.FocusGoal = d.FocusGoal
	}

	// Check for silent rework channels
	result.ChannelHealthWarnings = CheckChannelHealth(d.Learning)

	// Build sibling validator matching Decide() in ooda.go.
	siblingCache := make(map[string]bool)
	siblingExists := func(id string) bool {
		if cached, ok := siblingCache[id]; ok {
			return cached
		}
		_, err := d.resolveIssueQuerier().GetIssueStatus(id)
		exists := err == nil
		siblingCache[id] = exists
		return exists
	}

	var spawnable *Issue
	for _, issue := range issues {
		// Coordination: defer test issues when implementation siblings are pending (epic children only).
		// Matches the ShouldDeferTestIssue check in Decide() (ooda.go).
		if shouldDefer, reason := ShouldDeferTestIssue(issue, issues, siblingExists, epicChildIDs); shouldDefer {
			result.RejectedIssues = append(result.RejectedIssues, RejectedIssue{
				Issue:  issue,
				Reason: reason,
			})
			continue
		}

		// Use CheckIssueCompliance — the same filter as the poll loop.
		// This ensures Preview accurately reflects what the daemon will spawn.
		filter := d.CheckIssueCompliance(issue, nil, epicChildIDs)
		if !filter.Passed {
			result.RejectedIssues = append(result.RejectedIssues, RejectedIssue{
				Issue:  issue,
				Reason: filter.Reason,
			})
			continue
		}

		// Count all spawnable issues for accurate reporting.
		result.SpawnableCount++

		// Take the first one (highest priority) as the preview candidate.
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
	modelRoute := RouteModel(skill, spawnable)
	result.Model = modelRoute.Model
	result.ModelRouteReason = modelRoute.Reason

	// Resolve model routing config if available
	if d.Config.ModelRouting != nil && d.Config.ModelRouting.IsConfigured() {
		route := d.Config.ModelRouting.Resolve(skill, result.Model)
		result.ModelRoute = &route
		result.Model = route.EffectiveModel
		result.ModelRouteReason = route.Reason
	}

	// Check if selected issue was focus-boosted
	if d.FocusGoal != "" && d.FocusBoostAmount > 0 {
		prefix := projectFromIssueID(spawnable.ID)
		result.FocusBoosted = matchFocusToProject(d.FocusGoal, prefix, d.ProjectDirNames)
	}

	// Check for hotspot warnings if checker is configured
	if d.HotspotChecker != nil {
		result.HotspotWarnings = CheckHotspotsForIssue(spawnable, d.HotspotChecker)

		// Check if architect escalation would apply
		escalation := CheckArchitectEscalation(spawnable, skill, d.HotspotChecker, d.PriorArchitectFinder)
		if escalation != nil && escalation.Escalated {
			result.Skill = "architect"
			result.Model = InferModelFromSkill("architect")
			result.ArchitectEscalated = true
		}
	}

	return result, nil
}

// CountSpawnable returns how many of the given issues would actually be spawned
// by the daemon (i.e., pass all rejection filters). This is the number that should
// be shown in status displays — not the raw bd ready count.
func (d *Daemon) CountSpawnable(issues []Issue) int {
	count := 0
	for _, issue := range issues {
		filter := d.CheckIssueCompliance(issue, nil, nil)
		if filter.Passed {
			count++
		}
	}
	return count
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

// FormatChannelHealthWarnings formats channel health warnings for display.
func FormatChannelHealthWarnings(warnings []ChannelHealthWarning) string {
	if len(warnings) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\nChannel Health Warnings:\n")
	for _, w := range warnings {
		sb.WriteString(fmt.Sprintf("  [!] %s\n", w.Message))
	}
	return sb.String()
}

// FormatRejectedIssues formats rejected issues as a grouped summary by reason.
// Shows counts per rejection reason rather than individual issue IDs,
// preventing agents from misreading the rejected list as queue depth.
func FormatRejectedIssues(rejected []RejectedIssue) string {
	if len(rejected) == 0 {
		return ""
	}

	// Group by reason
	reasonCounts := make(map[string]int)
	var reasonOrder []string
	for _, r := range rejected {
		if reasonCounts[r.Reason] == 0 {
			reasonOrder = append(reasonOrder, r.Reason)
		}
		reasonCounts[r.Reason]++
	}

	// Sort by count descending for readability
	sort.Slice(reasonOrder, func(i, j int) bool {
		return reasonCounts[reasonOrder[i]] > reasonCounts[reasonOrder[j]]
	})

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\nRejected (%d issues):\n", len(rejected)))
	for _, reason := range reasonOrder {
		sb.WriteString(fmt.Sprintf("  %s: %d\n", reason, reasonCounts[reason]))
	}
	return sb.String()
}
