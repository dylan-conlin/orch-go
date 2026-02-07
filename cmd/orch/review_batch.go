package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// getCompletionsForReview retrieves completed agents with verification status.
// Scans .orch/workspace/ for completed workspaces. Detects both:
// - Full-tier agents: those with SYNTHESIS.md
// - Light-tier agents: those with .tier file containing "light" AND Phase: Complete in beads comments
// Filters out completions whose beads issues are already closed (closed/deferred/tombstone).
//
// Performance optimization: Uses batch comment fetching to avoid O(n) beads API calls.
// Previously, each workspace with SYNTHESIS.md would call VerifyCompletionFull which called
// GetComments multiple times (for phase verification, test evidence, visual verification, etc.).
// Now, all comments are fetched in a single batch call, then passed to verification functions.
//
// The workdir parameter allows cross-project review by specifying a different project directory.
func getCompletionsForReview(workdir string) ([]CompletionInfo, error) {
	currentDir, err := currentProjectDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Resolve project directory using shared helper
	projectResult, err := resolveProjectDir(workdir, "", currentDir)
	if err != nil {
		return nil, err
	}
	projectDir := projectResult.ProjectDir

	// Set beads.DefaultDir for cross-project operations
	projectResult.SetBeadsDefaultDir()

	// Phase 1: Scan workspaces and collect candidates with beads IDs
	type candidateWorkspace struct {
		dirName      string
		dirPath      string
		hasSynthesis bool
		isLightTier  bool
		lightBeadsID string
		beadsID      string
		skill        string
		modTime      time.Time
		isUntracked  bool
	}

	var workspaceCandidates []candidateWorkspace
	var beadsIDsToFetch []string
	beadsIDSet := make(map[string]bool) // Deduplicate beads IDs

	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	entries, _ := os.ReadDir(workspaceDir)

	// Track light-tier beads IDs separately for batch fetching
	var lightTierBeadsIDs []string
	lightTierBeadsIDSet := make(map[string]bool)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		dirPath := filepath.Join(workspaceDir, dirName)

		// Check for SYNTHESIS.md (full-tier completion)
		synthesisPath := filepath.Join(dirPath, "SYNTHESIS.md")
		hasSynthesis := false
		if _, err := os.Stat(synthesisPath); err == nil {
			hasSynthesis = true
		}

		// Check if this is a light-tier workspace (has .tier file with "light")
		// Note: We check isLightTierWorkspace here, NOT isLightTierComplete
		// The Phase: Complete check is deferred until after batch fetching
		isLightTier := isLightTierWorkspace(dirPath)

		// Skip workspaces that are neither full-tier with synthesis nor light-tier
		if !hasSynthesis && !isLightTier {
			continue
		}

		// Get workspace modification time from directory
		dirInfo, err := entry.Info()
		modTime := time.Now()
		if err == nil {
			modTime = dirInfo.ModTime()
		}

		// Early filter: Skip workspaces that are definitely stale (older than StaleThreshold)
		// This avoids fetching comments for old workspaces that will be filtered out anyway.
		// The stale check later refines this based on Phase status.
		// Note: We can't determine Phase yet (need comments), but we can skip very old workspaces.
		isDefinitelyStale := time.Since(modTime) > StaleThreshold
		if isDefinitelyStale && !hasSynthesis {
			// Skip light-tier workspaces that are definitely stale
			// Full-tier workspaces (with SYNTHESIS.md) are always processed for synthesis review
			continue
		}

		// Extract beads ID from SPAWN_CONTEXT.md
		beadsID := extractBeadsIDFromWorkspace(dirPath)

		// Extract skill from workspace name
		skill := extractSkillFromTitle(dirName)

		// Detect if agent is untracked
		isUntracked := isUntrackedBeadsID(beadsID)

		workspaceCandidates = append(workspaceCandidates, candidateWorkspace{
			dirName:      dirName,
			dirPath:      dirPath,
			hasSynthesis: hasSynthesis,
			isLightTier:  isLightTier, // Note: this is now the workspace tier, not completion status
			lightBeadsID: beadsID,     // Store beads ID for light-tier workspaces
			beadsID:      beadsID,
			skill:        skill,
			modTime:      modTime,
			isUntracked:  isUntracked,
		})

		// Collect beads IDs for batch fetching (skip untracked and empty)
		if beadsID != "" && !isUntracked && !beadsIDSet[beadsID] {
			beadsIDsToFetch = append(beadsIDsToFetch, beadsID)
			beadsIDSet[beadsID] = true
		}

		// Also collect light-tier beads IDs for Phase: Complete check
		if isLightTier && beadsID != "" && !isUntracked && !lightTierBeadsIDSet[beadsID] {
			lightTierBeadsIDs = append(lightTierBeadsIDs, beadsID)
			lightTierBeadsIDSet[beadsID] = true
		}
	}

	// Phase 2: Batch fetch all comments for tracked workspaces
	// This single batch call replaces O(n * 4+) individual GetComments calls
	// (each workspace potentially called GetComments for phase, test evidence,
	// visual verification, phase gates, AND light-tier completion checks).
	// Note: beadsIDsToFetch includes both full-tier and light-tier beads IDs
	commentsMap := verify.GetCommentsBatch(beadsIDsToFetch)

	// Phase 3: Verify each workspace using cached comments
	var candidates []CompletionInfo

	for _, ws := range workspaceCandidates {
		// For light-tier workspaces, check Phase: Complete using pre-fetched comments
		isLightComplete := false
		if ws.isLightTier && ws.beadsID != "" {
			if comments, ok := commentsMap[ws.beadsID]; ok {
				phaseStatus := verify.ParsePhaseFromComments(comments)
				isLightComplete = phaseStatus.Found && strings.EqualFold(phaseStatus.Phase, "Complete")
			}
		}

		// Skip light-tier workspaces that don't have Phase: Complete
		if ws.isLightTier && !ws.hasSynthesis && !isLightComplete {
			continue
		}

		info := CompletionInfo{
			WorkspaceID:   ws.dirName,
			WorkspacePath: ws.dirPath,
			BeadsID:       ws.beadsID,
			Project:       extractProject(projectDir),
			Skill:         ws.skill,
			ModTime:       ws.modTime,
			IsUntracked:   ws.isUntracked,
			IsLightTier:   isLightComplete, // Now reflects actual completion status, not just tier
		}

		// Handle full-tier agents (with SYNTHESIS.md)
		if ws.hasSynthesis {
			// Check verification status if we have a beads ID
			if ws.beadsID != "" {
				// Use lightweight verification for review (avoids O(n) git/build commands)
				// Full verification with git diff, build checks, etc. is done in orch complete
				comments := commentsMap[ws.beadsID]
				result, err := verify.VerifyCompletionForReview(ws.beadsID, ws.dirPath, "", serverURL, comments)
				if err != nil {
					info.VerifyError = fmt.Sprintf("verification error: %v", err)
					info.VerifyOK = false
				} else if result.Passed {
					info.VerifyOK = true
					info.Phase = result.Phase.Phase
					info.Summary = result.Phase.Summary

					// Try to parse synthesis
					s, err := verify.ParseSynthesis(ws.dirPath)
					if err == nil {
						info.Synthesis = s
					}
				} else {
					info.VerifyOK = false
					if len(result.Errors) > 0 {
						info.VerifyError = result.Errors[0]
					}
				}
			} else {
				// No beads ID but has SYNTHESIS.md - partially verifiable
				info.VerifyOK = true
				info.Phase = "Complete"
				info.Summary = "(no beads tracking)"

				// Try to parse synthesis
				s, err := verify.ParseSynthesis(ws.dirPath)
				if err == nil {
					info.Synthesis = s
				}
			}
		} else if isLightComplete {
			// Handle light-tier agents (no SYNTHESIS.md by design)
			// Light-tier agents are verified OK if they have Phase: Complete
			info.VerifyOK = true
			info.Phase = "Complete"
			info.Summary = "(light tier - no synthesis by design)"
		}

		// Determine if agent is stale (non-Complete phase for >24h)
		info.IsStale = isStaleAgent(info.Phase, info.ModTime)

		candidates = append(candidates, info)
	}

	// Filter out completions whose beads issues are already closed
	// This prevents showing NEEDS_REVIEW for issues that were force-closed
	return filterClosedIssues(candidates), nil
}

// filterClosedIssues removes completions whose beads issues are closed/deferred/tombstone.
// Uses ListOpenIssues for efficiency - a single call to get all open issues.
// If beads is unavailable, returns all candidates (better to show potential false positives than hide real issues).
func filterClosedIssues(candidates []CompletionInfo) []CompletionInfo {
	if len(candidates) == 0 {
		return candidates
	}

	// Use ListOpenIssues to get all open issues in a single call
	// This is much faster than individual Show() calls for each beads ID
	openIssueMap, err := verify.ListOpenIssues()
	if err != nil {
		// If beads is unavailable, return all candidates
		return candidates
	}

	// Filter out closed issues (keep only those that exist in openIssueMap)
	var results []CompletionInfo
	for _, c := range candidates {
		// Keep untracked agents (no beads issue to check)
		if c.IsUntracked || c.BeadsID == "" {
			results = append(results, c)
			continue
		}

		// Check if issue is open (exists in openIssueMap)
		if _, isOpen := openIssueMap[c.BeadsID]; isOpen {
			results = append(results, c)
		}
		// If not in openIssueMap, it's closed - skip it
	}

	return results
}

// extractBeadsIDFromWorkspace extracts the beads ID from SPAWN_CONTEXT.md
func extractBeadsIDFromWorkspace(workspacePath string) string {
	spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	content, err := os.ReadFile(spawnContextPath)
	if err != nil {
		return ""
	}

	// Look for "beads issue: **xxx**" pattern or "orch-go-pe5d.2" format
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, "beads issue:") || strings.Contains(lineLower, "spawned from beads issue:") {
			// Extract beads ID from the line
			// Patterns: "beads issue: **orch-go-pe5d.2**" or "orch-go-pe5d.2"
			for _, part := range strings.Fields(line) {
				part = strings.Trim(part, "*`[]")
				// Look for pattern like "project-xxxx" or "project-xxxx.n"
				if strings.Count(part, "-") >= 1 && len(part) > 5 {
					// Skip common non-ID words
					if strings.HasPrefix(part, "beads") || strings.HasPrefix(part, "BEADS") ||
						strings.HasPrefix(part, "issue") || strings.HasPrefix(part, "ISSUE") ||
						strings.HasPrefix(part, "bd") || strings.HasPrefix(part, "comment") {
						continue
					}
					return part
				}
			}
		}
	}
	return ""
}

// extractProject gets project name from project directory.
func extractProject(projectDir string) string {
	if projectDir == "" {
		return "unknown"
	}
	return filepath.Base(projectDir)
}

// isStaleAgent returns true if the agent is in a non-Complete phase and
// the workspace hasn't been modified in over 24 hours.
func isStaleAgent(phase string, modTime time.Time) bool {
	if phase == "Complete" {
		return false
	}
	return time.Since(modTime) > StaleThreshold
}

// groupByProject groups completions by project.
func groupByProject(completions []CompletionInfo) map[string][]CompletionInfo {
	grouped := make(map[string][]CompletionInfo)
	for _, c := range completions {
		grouped[c.Project] = append(grouped[c.Project], c)
	}
	return grouped
}

func runReview(projectFilter string, needsReviewOnly bool, staleOnly bool, showAll bool, limit int, workdir string) error {
	completions, err := getCompletionsForReview(workdir)
	if err != nil {
		return err
	}

	// Track counts before filtering for summary
	totalCount := len(completions)
	staleCount := 0
	untrackedCount := 0
	staleOrUntrackedCount := 0 // Stale OR untracked (no double-counting)
	for _, c := range completions {
		if c.IsStale {
			staleCount++
		}
		if c.IsUntracked {
			untrackedCount++
		}
		if c.IsStale || c.IsUntracked {
			staleOrUntrackedCount++
		}
	}

	// Filter by stale/untracked status
	// Default: exclude stale and untracked
	// --stale: show only stale and untracked
	// --all: show everything
	if !showAll {
		var filtered []CompletionInfo
		for _, c := range completions {
			if staleOnly {
				// Show only stale or untracked
				if c.IsStale || c.IsUntracked {
					filtered = append(filtered, c)
				}
			} else {
				// Default: exclude stale and untracked
				if !c.IsStale && !c.IsUntracked {
					filtered = append(filtered, c)
				}
			}
		}
		completions = filtered
	}

	// Filter by project if specified
	if projectFilter != "" {
		var filtered []CompletionInfo
		for _, c := range completions {
			if c.Project == projectFilter {
				filtered = append(filtered, c)
			}
		}
		completions = filtered
	}

	// Filter by needs-review if specified
	if needsReviewOnly {
		var filtered []CompletionInfo
		for _, c := range completions {
			if !c.VerifyOK {
				filtered = append(filtered, c)
			}
		}
		completions = filtered
	}

	// Track total after filters for limit messaging
	totalAfterFilters := len(completions)

	// Apply limit if specified (after all filters)
	if limit > 0 && len(completions) > limit {
		completions = completions[:limit]
	}

	if len(completions) == 0 {
		if projectFilter != "" {
			fmt.Printf("No pending completions for project: %s\n", projectFilter)
		} else if needsReviewOnly {
			fmt.Println("No completions need review")
		} else {
			fmt.Println("No pending completions")
		}
		return nil
	}

	// Group by project
	grouped := groupByProject(completions)

	// Get sorted project names
	var projects []string
	for p := range grouped {
		projects = append(projects, p)
	}
	sort.Strings(projects)

	// Print results
	totalOK := 0
	totalFailed := 0

	for _, project := range projects {
		items := grouped[project]
		fmt.Printf("\n## %s (%d completions)\n\n", project, len(items))

		for _, c := range items {
			status := "OK"
			if c.VerifyOK {
				totalOK++
				status = "OK"
			} else {
				totalFailed++
				status = "NEEDS_REVIEW"
			}

			// Add stale/untracked/light-tier indicators
			if c.IsStale {
				status = "STALE"
			}
			if c.IsUntracked {
				status = "UNTRACKED"
			}
			if c.IsLightTier {
				status = "LIGHT"
			}

			beadsInfo := ""
			if c.BeadsID != "" {
				beadsInfo = fmt.Sprintf(" (%s)", c.BeadsID)
			}

			fmt.Printf("  [%s] %s%s\n", status, c.WorkspaceID, beadsInfo)

			if c.VerifyOK && c.Summary != "" {
				fmt.Printf("         Phase: %s - %s\n", c.Phase, c.Summary)
			}

			// Display Synthesis Card if available (full-tier only)
			if c.Synthesis != nil {
				printSynthesisCard(c.Synthesis)
			}

			// Light tier note
			if c.IsLightTier {
				fmt.Println("         (Light tier - no synthesis by design)")
			}

			if !c.VerifyOK && c.VerifyError != "" {
				fmt.Printf("         Error: %s\n", c.VerifyError)
			}

			// Show skill if available
			if c.Skill != "" {
				fmt.Printf("         Skill: %s\n", c.Skill)
			}
		}
	}

	// Print summary
	fmt.Printf("\n---\n")
	fmt.Printf("Total: %d completions (%d OK, %d need review)\n", totalOK+totalFailed, totalOK, totalFailed)

	// Show truncation notice if limit was applied
	if limit > 0 && totalAfterFilters > limit {
		fmt.Printf("Showing: %d of %d (use --limit 0 or remove --limit to see all)\n", limit, totalAfterFilters)
	}

	// Show hidden counts if not showing all
	if !showAll && !staleOnly {
		if staleOrUntrackedCount > 0 {
			fmt.Printf("Hidden: %d stale/untracked (use --stale to view, --all to include)\n", staleOrUntrackedCount)
		}
	}

	// Show total breakdown if viewing stale only
	if staleOnly {
		fmt.Printf("Showing: %d stale, %d untracked (may overlap)\n", staleCount, untrackedCount)
		actionableCount := totalCount - staleOrUntrackedCount
		if actionableCount > 0 {
			fmt.Printf("Actionable (hidden): %d (run without --stale to view)\n", actionableCount)
		}
	}

	if totalOK > 0 {
		fmt.Printf("\nTo complete agents and close beads issues:\n")
		for _, project := range projects {
			fmt.Printf("  orch-go review done %s\n", project)
		}
	}

	if totalFailed > 0 {
		fmt.Printf("\nTo complete agents with issues:\n")
		fmt.Printf("  orch-go complete <beads-id>         # If Phase: Complete reported\n")
		fmt.Printf("  orch-go complete <beads-id> --force # Skip phase verification\n")
	}

	return nil
}

func runReviewDone(project, workdir string) error {
	completions, err := getCompletionsForReview(workdir)
	if err != nil {
		return err
	}

	projectCompletions := filterCompletionsByProject(completions, project)
	if len(projectCompletions) == 0 {
		fmt.Printf("No pending completions for project: %s\n", project)
		return nil
	}

	canComplete, needsReview := categorizeCompletions(projectCompletions)

	// Show summary before proceeding
	fmt.Printf("Project: %s\n", project)
	fmt.Printf("  Ready to complete: %d\n", len(canComplete))
	fmt.Printf("  Needs manual review: %d\n", len(needsReview))

	if len(canComplete) == 0 {
		printNoneReady(needsReview)
		return nil
	}

	if err := confirmReviewDone(len(canComplete)); err != nil {
		return err
	}

	currentDir, err := currentProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	projectResult, err := resolveProjectDir(workdir, "", currentDir)
	if err != nil {
		return err
	}
	projectResult.SetBeadsDefaultDir()

	completed, completionErrors := processCompletions(canComplete, projectResult.ProjectDir)

	printReviewDoneSummary(completed, len(canComplete), completionErrors, needsReview)

	return nil
}

// filterCompletionsByProject returns only completions matching the given project name.
func filterCompletionsByProject(completions []CompletionInfo, project string) []CompletionInfo {
	var result []CompletionInfo
	for _, c := range completions {
		if c.Project == project {
			result = append(result, c)
		}
	}
	return result
}

// categorizeCompletions splits completions into those ready to complete
// (verified OK with a beads ID) and those needing manual review.
func categorizeCompletions(completions []CompletionInfo) (canComplete, needsReview []CompletionInfo) {
	for _, c := range completions {
		if c.VerifyOK && c.BeadsID != "" {
			canComplete = append(canComplete, c)
		} else {
			needsReview = append(needsReview, c)
		}
	}
	return
}
