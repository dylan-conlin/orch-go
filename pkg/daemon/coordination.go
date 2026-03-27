// Package daemon provides autonomous overnight processing capabilities.
// coordination.go contains coordination logic — routing, scheduling, and
// prioritization decisions that consume compliance signals (compliance.go).
// Coordination value increases as parallelism grows.
package daemon

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/daemonconfig"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// SkillRoute contains the effective skill, model, and any routing metadata
// after applying hotspot extraction and architect escalation.
type SkillRoute struct {
	Skill string
	Model string

	// ModelRouteReason explains why this model was selected.
	// Set by RouteModel and updated if routing overrides the model.
	ModelRouteReason string

	// ExtractionSpawned indicates an extraction agent was spawned instead.
	ExtractionSpawned bool
	// ExtractionIssueID is the ID of the created extraction issue.
	ExtractionIssueID string
	// OriginalIssueID is the original issue (blocked by extraction).
	OriginalIssueID string
	// ReplacementIssue replaces the original issue for spawning (set when extraction happens).
	ReplacementIssue *Issue

	// ArchitectEscalated indicates skill was escalated to architect.
	ArchitectEscalated bool
	// ArchitectEscalationDetail contains full escalation decision.
	ArchitectEscalationDetail *ArchitectEscalation
}

// RouteIssueForSpawn determines the effective skill and model for an issue,
// applying hotspot extraction and architect escalation.
// This is a coordination decision that consumes hotspot compliance signals.
func (d *Daemon) RouteIssueForSpawn(issue *Issue, skill, inferredModel string, routeReason string) (SkillRoute, error) {
	route := SkillRoute{
		Skill:            skill,
		Model:            inferredModel,
		ModelRouteReason: routeReason,
	}

	// Check for critical hotspots requiring pre-extraction.
	if d.HotspotChecker != nil {
		extraction := CheckExtractionNeeded(issue, d.HotspotChecker)
		if extraction != nil && extraction.Needed {
			extractionID, err := d.resolveIssueQuerier().CreateExtractionIssue(extraction.ExtractionTask, issue.ID)
			if err != nil {
				// Extraction gate is non-negotiable: if setup fails, skip the issue
				if d.Config.Verbose {
					fmt.Printf("  Extraction setup failed for %s: %v (skipping issue)\n", issue.ID, err)
				}
				return route, fmt.Errorf("extraction setup failed for %s: %v (issue skipped, will retry on next poll)", issue.ID, err)
			}

			if d.Config.Verbose {
				fmt.Printf("  Auto-extraction: created %s blocking %s for %s (%d lines)\n",
					extractionID, issue.ID, extraction.CriticalFile, extraction.Hotspot.Score)
			}

			// Replace issue and skill with extraction work
			route.ExtractionSpawned = true
			route.OriginalIssueID = issue.ID
			route.ExtractionIssueID = extractionID
			route.ReplacementIssue = &Issue{
				ID:        extractionID,
				Title:     extraction.ExtractionTask,
				IssueType: "task",
				Priority:  1,
			}
			route.Skill = "feature-impl"
			route.Model = InferModelFromSkill(route.Skill)
			route.ModelRouteReason = "extraction task → feature-impl default"
			return route, nil
		}
	}

	// Layer 2: Architect escalation for hotspot areas.
	// Only when extraction didn't happen (extraction handles the most critical case).
	// Gated by compliance level — relaxed/autonomous levels skip escalation.
	complianceLevel := d.Config.Compliance.Resolve(skill, inferredModel)
	if d.HotspotChecker != nil && daemonconfig.DeriveArchitectEscalationEnabled(complianceLevel) {
		escalationDetail := CheckArchitectEscalation(issue, skill, d.HotspotChecker, d.PriorArchitectFinder)
		if escalationDetail != nil {
			route.ArchitectEscalationDetail = escalationDetail
			if escalationDetail.Escalated {
				if d.Config.Verbose {
					fmt.Printf("  Architect escalation: %s targets hotspot %s (%s, score=%d)\n",
						issue.ID, escalationDetail.HotspotFile, escalationDetail.HotspotType, escalationDetail.HotspotScore)
				}
				route.Skill = "architect"
				route.Model = InferModelFromSkill(route.Skill)
				route.ModelRouteReason = fmt.Sprintf("architect escalation (hotspot %s) → opus", escalationDetail.HotspotFile)
				route.ArchitectEscalated = true
			}
		}
	}

	return route, nil
}

// PrioritizeIssues applies coordination logic to order issues for selection:
// epic expansion, focus boost, allocation scoring (or priority sort), project interleaving.
// When learning data is available, uses skill-aware scoring instead of pure priority sort.
// Returns the ordered issues and a map of epic child IDs for label exemption.
func (d *Daemon) PrioritizeIssues(issues []Issue) ([]Issue, map[string]bool, error) {
	// Expand triage:ready epics by including their children
	issues, epicChildIDs, err := d.expandTriageReadyEpics(issues)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to expand epics: %w", err)
	}

	// Apply focus boost: issues from focused projects get priority boost
	if d.FocusGoal != "" && d.FocusBoostAmount > 0 {
		issues = applyFocusBoost(issues, d.FocusGoal, d.FocusBoostAmount, d.ProjectDirNames)
	}

	if d.Learning != nil && len(d.Learning.Skills) > 0 {
		// Allocation scoring: blend priority with skill success rate
		scored := ScoreIssues(issues, d.Learning)
		issues = make([]Issue, len(scored))
		for i, s := range scored {
			issues[i] = s.Issue
		}
	} else {
		// Fallback: sort by priority (lower number = higher priority)
		sort.Slice(issues, func(i, j int) bool {
			return issues[i].Priority < issues[j].Priority
		})
	}

	// Round-robin across projects within each priority level
	issues = interleaveByProject(issues)

	return issues, epicChildIDs, nil
}

// CompletionRoute describes how a completed agent should be processed.
type CompletionRoute struct {
	// Action is the routing action: "auto-complete-light", "auto-complete", "label-ready-review".
	Action string
	// ReviewTier is the agent's review tier from workspace manifest.
	ReviewTier string
}

// RouteCompletion determines how to process a completed agent based on
// compliance signals and effort/tier metadata. This is a coordination
// decision — it decides the action, but does not execute it.
func RouteCompletion(agent CompletedAgent) CompletionRoute {
	// Determine review tier from workspace manifest
	reviewTier := ""
	if agent.WorkspacePath != "" {
		reviewTier = verify.ReadReviewTierFromWorkspace(agent.WorkspacePath)
	}

	route := CompletionRoute{ReviewTier: reviewTier}

	// Effort-based routing: effort:small → light auto-complete
	if IsEffortSmall(agent.Labels) {
		route.Action = "auto-complete-light"
		return route
	}

	// Auto-tier routing: review-tier=auto → full auto-complete
	if reviewTier == "auto" {
		route.Action = "auto-complete"
		return route
	}

	// Scan-tier routing: review-tier=scan → auto-complete
	// Scan-tier work (investigations, probes, research, audits) produces
	// knowledge artifacts, not code changes. Low risk of breaking production.
	if reviewTier == "scan" {
		route.Action = "auto-complete"
		return route
	}

	// Default: label for orchestrator review
	route.Action = "label-ready-review"
	return route
}

// ExecuteCompletionRoute executes a completion routing decision.
// Separated from RouteCompletion to keep the routing decision pure/testable.
func (d *Daemon) ExecuteCompletionRoute(
	agent CompletedAgent,
	route CompletionRoute,
	signal CompletionVerifySignal,
	config CompletionConfig,
) CompletionResult {
	result := CompletionResult{
		BeadsID:      agent.BeadsID,
		Verification: signal.Verification,
		Escalation:   signal.Escalation,
	}

	effectiveProjectDir := agent.ProjectDir
	if effectiveProjectDir == "" {
		effectiveProjectDir = config.ProjectDir
	}

	// Build completion summary
	completionSummary := "Phase: Complete"
	if agent.PhaseSummary != "" {
		completionSummary = fmt.Sprintf("Phase: Complete - %s", agent.PhaseSummary)
	}

	switch route.Action {
	case "auto-complete-light":
		if d.AutoCompleter != nil && !config.DryRun {
			var completeErr error
			if lightCompleter, ok := d.AutoCompleter.(LightAutoCompleter); ok {
				completeErr = lightCompleter.CompleteLight(agent.BeadsID, effectiveProjectDir)
			} else {
				completeErr = d.AutoCompleter.Complete(agent.BeadsID, effectiveProjectDir)
			}
			if completeErr != nil {
				result.Error = fmt.Errorf("light auto-complete failed for effort:small agent: %w", completeErr)
				return result
			}
			result.Processed = true
			result.AutoCompleted = true
			result.CloseReason = completionSummary
			// Queue for orchestrator comprehension (fire-and-forget)
			d.addComprehensionUnread(agent.BeadsID, effectiveProjectDir, config)
			// Brief already generated by CompleteLight (--headless mode)
			return result
		}
		// Fall through to label if no auto-completer
		return d.labelReadyReview(agent, completionSummary, effectiveProjectDir, config)

	case "auto-complete":
		if d.AutoCompleter != nil && !config.DryRun {
			if err := d.AutoCompleter.Complete(agent.BeadsID, effectiveProjectDir); err != nil {
				result.Error = fmt.Errorf("auto-complete failed for auto-tier agent: %w", err)
				return result
			}
			result.Processed = true
			result.AutoCompleted = true
			result.CloseReason = completionSummary
			// Queue for orchestrator comprehension (fire-and-forget)
			d.addComprehensionUnread(agent.BeadsID, effectiveProjectDir, config)
			// Fire-and-forget: pre-generate brief via headless completion
			d.fireHeadlessCompletion(agent.BeadsID, effectiveProjectDir, config)
			return result
		}
		// Fall through to label if no auto-completer
		return d.labelReadyReview(agent, completionSummary, effectiveProjectDir, config)

	default: // "label-ready-review"
		return d.labelReadyReview(agent, completionSummary, effectiveProjectDir, config)
	}
}

// labelReadyReview marks an issue as ready for orchestrator review.
// Also fires headless completion to pre-generate the brief (same as auto-complete paths).
func (d *Daemon) labelReadyReview(agent CompletedAgent, completionSummary, effectiveProjectDir string, config CompletionConfig) CompletionResult {
	result := CompletionResult{
		BeadsID: agent.BeadsID,
	}

	if !config.DryRun {
		if err := verify.AddLabel(agent.BeadsID, LabelReadyReview, effectiveProjectDir); err != nil {
			result.Error = fmt.Errorf("failed to mark ready for review: %w", err)
			return result
		}

		// Remove triage:ready to prevent re-spawn
		verify.RemoveTriageLabels(agent.BeadsID, effectiveProjectDir)

		// Queue for orchestrator comprehension (fire-and-forget)
		d.addComprehensionUnread(agent.BeadsID, effectiveProjectDir, config)

		// Fire-and-forget: pre-generate brief via headless completion.
		// This runs `orch complete --headless` which skips interactive gates
		// and writes a brief to .kb/briefs/. Failure doesn't block labeling.
		d.fireHeadlessCompletion(agent.BeadsID, effectiveProjectDir, config)
	}

	result.Processed = true
	result.CloseReason = completionSummary
	return result
}

// fireHeadlessCompletion triggers headless completion to pre-generate a brief.
// Fire-and-forget: errors are logged but don't block the completion pipeline.
// Only runs when AutoCompleter implements HeadlessAutoCompleter.
func (d *Daemon) fireHeadlessCompletion(beadsID, projectDir string, config CompletionConfig) {
	headless, ok := d.AutoCompleter.(HeadlessAutoCompleter)
	if !ok || headless == nil {
		return
	}

	go func() {
		if err := headless.CompleteHeadless(beadsID, projectDir); err != nil {
			if config.Verbose {
				fmt.Printf("    Warning: headless brief generation failed for %s: %v\n", beadsID, err)
			}
		} else if config.Verbose {
			fmt.Printf("    Headless brief generated for %s\n", beadsID)
		}
	}()
}

// addComprehensionUnread adds the comprehension:unread label to a completed issue.
// Fire-and-forget: label failure is logged but doesn't block completion processing.
func (d *Daemon) addComprehensionUnread(beadsID, projectDir string, config CompletionConfig) {
	var err error
	if projectDir != "" {
		err = AddComprehensionUnreadInDir(beadsID, projectDir)
	} else {
		err = AddComprehensionUnread(beadsID)
	}
	if err != nil {
		if config.Verbose {
			fmt.Printf("    Warning: failed to add comprehension:unread label to %s: %v\n", beadsID, err)
		}
	}
}

// formatBlockerIDs formats blocker IDs for debug output.
func formatBlockerIDs(blockers []struct{ ID, Status string }) string {
	var ids []string
	for _, b := range blockers {
		ids = append(ids, fmt.Sprintf("%s (%s)", b.ID, b.Status))
	}
	return strings.Join(ids, ", ")
}
