// daemon_crossproject.go contains cross-project polling, spawning, and preview.
package daemon

import (
	"fmt"
	"strings"
)

// CrossProjectIssue represents an issue with its associated project context.
// Used for cross-project polling where issues need to track their source project.
type CrossProjectIssue struct {
	Issue   Issue
	Project Project
}

// CrossProjectOnceResult contains the result of processing one cross-project issue.
type CrossProjectOnceResult struct {
	Processed   bool
	Issue       *Issue
	Project     *Project
	Skill       string
	Message     string
	Error       error
	ProjectName string // Convenience field for logging: "[project-name]"
}

// ListCrossProjectIssues returns all triage:ready issues across all kb-registered projects.
// Issues are sorted by priority (0 = highest priority).
// Errors in individual projects are logged but don't stop processing of other projects.
func (d *Daemon) ListCrossProjectIssues() ([]CrossProjectIssue, error) {
	projects, err := d.listProjectsFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	var allIssues []CrossProjectIssue

	for _, project := range projects {
		issues, err := d.listIssuesForProjectFunc(project.Path)
		if err != nil {
			// Log error but continue to next project (per acceptance criteria)
			if d.Config.Verbose {
				fmt.Printf("  [%s] Failed to list issues: %v\n", project.Name, err)
			}
			continue
		}

		for _, issue := range issues {
			allIssues = append(allIssues, CrossProjectIssue{
				Issue:   issue,
				Project: project,
			})
		}
	}

	// Apply the active sort strategy
	allIssues = d.SortCrossProjectIssues(allIssues)

	return allIssues, nil
}

// CrossProjectOnce processes a single issue from any kb-registered project.
// If cross-project mode is not enabled in config, this behaves like Once().
// Returns a result indicating what was processed and from which project.
//
// Key behaviors:
// - Iterates over all kb-registered projects
// - Respects global capacity limit (shared across all projects)
// - Error in one project doesn't block other projects
// - Includes project name in result for logging visibility
func (d *Daemon) CrossProjectOnce() (*CrossProjectOnceResult, error) {
	return d.CrossProjectOnceExcluding(nil)
}

// CrossProjectOnceExcluding processes a single issue from any kb-registered project,
// excluding any issues in the skip set. The skip map keys should be "projectPath:issueID".
func (d *Daemon) CrossProjectOnceExcluding(skip map[string]bool) (*CrossProjectOnceResult, error) {
	// Check rate limit first (before fetching issues)
	if d.RateLimiter != nil {
		canSpawn, count, msg := d.RateLimiter.CanSpawn()
		if !canSpawn {
			if d.Config.Verbose {
				fmt.Printf("  Rate limited: %s\n", msg)
			}
			return &CrossProjectOnceResult{
				Processed: false,
				Message:   fmt.Sprintf("Rate limited: %d/%d spawns in the last hour", count, d.RateLimiter.MaxPerHour),
			}, nil
		}
	}

	// Get all projects
	projects, err := d.listProjectsFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		return &CrossProjectOnceResult{
			Processed: false,
			Message:   "No kb-registered projects found",
		}, nil
	}

	// Collect spawnable issues across all projects
	allIssues := d.collectSpawnableIssues(projects, skip)

	if len(allIssues) == 0 {
		return &CrossProjectOnceResult{
			Processed: false,
			Message:   "No spawnable issues in any project",
		}, nil
	}

	// Apply the active sort strategy
	allIssues = d.SortCrossProjectIssues(allIssues)

	// Select the best candidate that passes dedup and completion checks
	selected, skill := d.selectCrossProjectCandidate(allIssues)

	if selected == nil {
		return &CrossProjectOnceResult{
			Processed: false,
			Message:   "No spawnable issues (all skipped due to existing sessions or Phase: Complete)",
		}, nil
	}

	// Execute spawn: acquire pool slot, mark processed, spawn, handle failures
	return d.executeCrossProjectSpawn(selected, skill)
}

// projectSkipCounts tracks reasons issues were skipped during collection for a single project.
// Used for summary logging to reduce verbosity compared to per-issue logging.
type projectSkipCounts struct {
	failedSpawn   int
	recentSpawn   int
	typeNotSpawn  int
	statusBlocked int
	missingLabel  int
}

// collectSpawnableIssues gathers issues from all projects, applying pre-spawn filters
// (skip set, recently spawned, type checks, status checks, label checks).
// Per-project skip summaries are logged in verbose mode.
func (d *Daemon) collectSpawnableIssues(projects []Project, skip map[string]bool) []CrossProjectIssue {
	var allIssues []CrossProjectIssue

	for _, project := range projects {
		issues, err := d.listIssuesForProjectFunc(project.Path)
		if err != nil {
			if d.Config.Verbose {
				fmt.Printf("  [%s] Failed to list issues: %v\n", project.Name, err)
			}
			continue
		}

		var counts projectSkipCounts
		spawnable := 0

		for _, issue := range issues {
			if d.shouldSkipIssue(issue, project.Path, skip, &counts) {
				continue
			}

			spawnable++
			allIssues = append(allIssues, CrossProjectIssue{
				Issue:   issue,
				Project: project,
			})
		}

		d.logProjectSkipSummary(project.Name, spawnable, &counts)
	}

	return allIssues
}

// shouldSkipIssue checks whether an issue should be filtered out during collection.
// It updates counts for each skip reason to enable summary logging.
// Returns true if the issue should be skipped.
func (d *Daemon) shouldSkipIssue(issue Issue, projectPath string, skip map[string]bool, counts *projectSkipCounts) bool {
	// Skip issues in the skip set
	skipKey := fmt.Sprintf("%s:%s", projectPath, issue.ID)
	if skip != nil && skip[skipKey] {
		counts.failedSpawn++
		return true
	}

	// Skip issues that have been recently spawned
	if d.SpawnedIssues != nil && d.SpawnedIssues.IsSpawned(issue.ID) {
		counts.recentSpawn++
		if d.EventLogger != nil {
			_ = d.EventLogger.LogDedupBlocked(map[string]interface{}{
				"beads_id":    issue.ID,
				"dedup_layer": "spawned_tracker",
				"reason":      "Issue recently spawned, awaiting status update (6h TTL)",
			})
		}
		return true
	}

	// Skip non-spawnable types
	if !IsSpawnableType(issue.IssueType) {
		counts.typeNotSpawn++
		return true
	}

	// Skip blocked or in_progress issues
	if issue.Status == "blocked" || issue.Status == "in_progress" {
		counts.statusBlocked++
		return true
	}

	// Skip issues without required label (if filter is set)
	if d.Config.Label != "" && !issue.HasLabel(d.Config.Label) {
		counts.missingLabel++
		return true
	}

	return false
}

// logProjectSkipSummary logs a summary of skip reasons for a project in verbose mode.
// This is much less verbose than logging each skipped issue individually.
func (d *Daemon) logProjectSkipSummary(projectName string, spawnable int, counts *projectSkipCounts) {
	if !d.Config.Verbose {
		return
	}

	totalSkipped := counts.failedSpawn + counts.recentSpawn +
		counts.typeNotSpawn + counts.statusBlocked + counts.missingLabel
	if totalSkipped == 0 && spawnable == 0 {
		return
	}

	var parts []string
	if spawnable > 0 {
		parts = append(parts, fmt.Sprintf("%d spawnable", spawnable))
	}
	if counts.missingLabel > 0 {
		parts = append(parts, fmt.Sprintf("%d missing label", counts.missingLabel))
	}
	if counts.statusBlocked > 0 {
		parts = append(parts, fmt.Sprintf("%d blocked/in_progress", counts.statusBlocked))
	}
	if counts.typeNotSpawn > 0 {
		parts = append(parts, fmt.Sprintf("%d non-spawnable type", counts.typeNotSpawn))
	}
	if counts.recentSpawn > 0 {
		parts = append(parts, fmt.Sprintf("%d recently spawned", counts.recentSpawn))
	}
	if counts.failedSpawn > 0 {
		parts = append(parts, fmt.Sprintf("%d failed this cycle", counts.failedSpawn))
	}
	fmt.Printf("  [%s] %s\n", projectName, strings.Join(parts, ", "))
}

// selectCrossProjectCandidate iterates candidates in priority order, applying dedup checks
// (ProcessedCache, synthesis completion) to find the first viable candidate.
// Returns nil if no candidate passes all checks.
func (d *Daemon) selectCrossProjectCandidate(candidates []CrossProjectIssue) (*CrossProjectIssue, string) {
	for i := range candidates {
		candidate := &candidates[i]

		// Infer skill for this candidate
		skill, err := InferSkillFromIssue(&candidate.Issue)
		if err != nil {
			if d.Config.Verbose {
				fmt.Printf("  [%s] Skipping %s (failed to infer skill: %v)\n",
					candidate.Project.Name, candidate.Issue.ID, err)
			}
			continue
		}

		// Unified dedup check: Use ProcessedCache to consolidate three checks
		if d.ProcessedCache != nil && !d.ProcessedCache.ShouldProcess(candidate.Issue.ID) {
			if d.Config.Verbose {
				fmt.Printf("  [%s] Skipping %s (blocked by ProcessedCache)\n",
					candidate.Project.Name, candidate.Issue.ID)
			}
			if d.EventLogger != nil {
				_ = d.EventLogger.LogDedupBlocked(map[string]interface{}{
					"beads_id":    candidate.Issue.ID,
					"dedup_layer": "processed_cache",
					"reason":      "Issue blocked by unified ProcessedCache",
				})
			}
			continue
		}

		// Synthesis completion check (cross-project: use project path)
		if reason := CheckSynthesisCompletion(&candidate.Issue, candidate.Project.Path); reason != "" {
			if d.Config.Verbose {
				fmt.Printf("  [%s] Skipping %s (%s)\n",
					candidate.Project.Name, candidate.Issue.ID, reason)
			}
			if d.EventLogger != nil {
				_ = d.EventLogger.LogDedupBlocked(map[string]interface{}{
					"beads_id":    candidate.Issue.ID,
					"dedup_layer": "synthesis_completion",
					"reason":      reason,
				})
			}
			continue
		}

		return candidate, skill
	}

	return nil, ""
}

// executeCrossProjectSpawn handles pool slot acquisition, marking the issue as processed,
// spawning the work, and cleanup on failure (unmarking, releasing slot).
func (d *Daemon) executeCrossProjectSpawn(selected *CrossProjectIssue, skill string) (*CrossProjectOnceResult, error) {
	// If pool is configured, acquire a slot first
	var slot *Slot
	if d.Pool != nil {
		slot = d.Pool.TryAcquire()
		if slot == nil {
			return &CrossProjectOnceResult{
				Processed:   false,
				Issue:       &selected.Issue,
				Project:     &selected.Project,
				Skill:       skill,
				ProjectName: selected.Project.Name,
				Message:     "At capacity - no slots available",
			}, nil
		}
		slot.BeadsID = selected.Issue.ID
	}

	// Mark in legacy tracker before spawn to preserve the race-window dedup behavior.
	// ProcessedCache is marked only after confirmed successful spawn.
	if d.SpawnedIssues != nil {
		d.SpawnedIssues.MarkSpawned(selected.Issue.ID)
	}

	// Spawn the work with project context
	if err := d.spawnForProjectFunc(selected.Issue.ID, selected.Project.Path); err != nil {
		if d.SpawnedIssues != nil {
			d.SpawnedIssues.Unmark(selected.Issue.ID)
		}
		// Release slot on spawn failure
		if d.Pool != nil && slot != nil {
			d.Pool.Release(slot)
		}
		return &CrossProjectOnceResult{
			Processed:   false,
			Issue:       &selected.Issue,
			Project:     &selected.Project,
			Skill:       skill,
			ProjectName: selected.Project.Name,
			Error:       err,
			Message:     fmt.Sprintf("[%s] Failed to spawn: %v", selected.Project.Name, err),
		}, nil
	}

	// Mark in persistent processed cache only after successful spawn.
	if d.ProcessedCache != nil {
		if err := d.ProcessedCache.MarkProcessed(selected.Issue.ID); err != nil {
			fmt.Printf("warning: failed to mark issue as processed: %v\n", err)
		}
	}

	// Record successful spawn for rate limiting
	if d.RateLimiter != nil {
		d.RateLimiter.RecordSpawn()
	}

	return &CrossProjectOnceResult{
		Processed:   true,
		Issue:       &selected.Issue,
		Project:     &selected.Project,
		Skill:       skill,
		ProjectName: selected.Project.Name,
		Message:     fmt.Sprintf("[%s] Spawned work on %s", selected.Project.Name, selected.Issue.ID),
	}, nil
}

// CrossProjectPreview shows what would be processed next without actually processing.
// Returns issues from all kb-registered projects, sorted by priority.
func (d *Daemon) CrossProjectPreview() (*CrossProjectPreviewResult, error) {
	result := &CrossProjectPreviewResult{}

	// Check rate limit status
	if d.RateLimiter != nil {
		canSpawn, count, msg := d.RateLimiter.CanSpawn()
		result.RateLimited = !canSpawn
		if d.RateLimiter.MaxPerHour > 0 {
			result.RateStatus = fmt.Sprintf("%d/%d spawns in last hour", count, d.RateLimiter.MaxPerHour)
		}
		if !canSpawn {
			result.Message = msg
		}
	}

	// Get all projects
	projects, err := d.listProjectsFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	result.Projects = projects

	if len(projects) == 0 {
		result.Message = "No kb-registered projects found"
		return result, nil
	}

	// Collect spawnable and rejected issues from all projects
	for _, project := range projects {
		issues, err := d.listIssuesForProjectFunc(project.Path)
		if err != nil {
			result.ProjectErrors = append(result.ProjectErrors, ProjectError{
				Project: project,
				Error:   err,
			})
			continue
		}

		for _, issue := range issues {
			reason := d.checkRejectionReason(issue)
			if reason != "" {
				result.RejectedIssues = append(result.RejectedIssues, CrossProjectRejected{
					Issue:   issue,
					Project: project,
					Reason:  reason,
				})
				continue
			}

			result.SpawnableIssues = append(result.SpawnableIssues, CrossProjectIssue{
				Issue:   issue,
				Project: project,
			})
		}
	}

	// Apply the active sort strategy to spawnable issues
	result.SpawnableIssues = d.SortCrossProjectIssues(result.SpawnableIssues)

	// Select the first spawnable issue (if any) for preview
	if len(result.SpawnableIssues) > 0 {
		first := result.SpawnableIssues[0]
		result.NextIssue = &first.Issue
		result.NextProject = &first.Project

		skill, err := InferSkillFromIssue(&first.Issue)
		if err == nil {
			result.Skill = skill
		}

		// Check for hotspot warnings if checker is configured
		if d.HotspotChecker != nil {
			result.HotspotWarnings = CheckHotspotsForIssue(&first.Issue, d.HotspotChecker)
		}
	} else if result.Message == "" {
		result.Message = "No spawnable issues in any project"
	}

	return result, nil
}

// CrossProjectPreviewResult contains the result of a cross-project preview operation.
type CrossProjectPreviewResult struct {
	NextIssue       *Issue
	NextProject     *Project
	Skill           string
	Message         string
	RateLimited     bool
	RateStatus      string
	HotspotWarnings []HotspotWarning
	Projects        []Project
	SpawnableIssues []CrossProjectIssue
	RejectedIssues  []CrossProjectRejected
	ProjectErrors   []ProjectError
}

// CrossProjectRejected captures a rejected issue with its project context.
type CrossProjectRejected struct {
	Issue   Issue
	Project Project
	Reason  string
}

// ProjectError captures an error that occurred while processing a project.
type ProjectError struct {
	Project Project
	Error   error
}

// FormatCrossProjectPreview formats cross-project preview results for display.
func FormatCrossProjectPreview(result *CrossProjectPreviewResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Projects scanned: %d\n", len(result.Projects)))

	if result.RateLimited {
		sb.WriteString(fmt.Sprintf("Rate limited: %s\n", result.Message))
	}

	if len(result.ProjectErrors) > 0 {
		sb.WriteString("\nProject errors:\n")
		for _, pe := range result.ProjectErrors {
			sb.WriteString(fmt.Sprintf("  [%s] %v\n", pe.Project.Name, pe.Error))
		}
	}

	if result.NextIssue != nil && result.NextProject != nil {
		sb.WriteString("\nNext to spawn:\n")
		sb.WriteString(fmt.Sprintf("  Project:  %s\n", result.NextProject.Name))
		sb.WriteString(FormatPreview(result.NextIssue))
		sb.WriteString(fmt.Sprintf("\nInferred skill: %s\n", result.Skill))
	} else {
		sb.WriteString(fmt.Sprintf("\n%s\n", result.Message))
	}

	if len(result.SpawnableIssues) > 1 {
		sb.WriteString(fmt.Sprintf("\nOther spawnable issues: %d\n", len(result.SpawnableIssues)-1))
		for i, cpi := range result.SpawnableIssues[1:] {
			if i >= 5 {
				sb.WriteString(fmt.Sprintf("  ... and %d more\n", len(result.SpawnableIssues)-6))
				break
			}
			sb.WriteString(fmt.Sprintf("  [%s] %s: %s\n", cpi.Project.Name, cpi.Issue.ID, cpi.Issue.Title))
		}
	}

	if len(result.RejectedIssues) > 0 {
		sb.WriteString(fmt.Sprintf("\nRejected issues: %d\n", len(result.RejectedIssues)))
		for i, cpr := range result.RejectedIssues {
			if i >= 10 {
				sb.WriteString(fmt.Sprintf("  ... and %d more\n", len(result.RejectedIssues)-10))
				break
			}
			sb.WriteString(fmt.Sprintf("  [%s] %s: %s\n", cpr.Project.Name, cpr.Issue.ID, cpr.Reason))
		}
	}

	return sb.String()
}
