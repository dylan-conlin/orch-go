// Package main provides the CLI entry point for orch-go.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	// Review command flags
	reviewProject     string
	reviewNeedsReview bool
	reviewStale       bool
	reviewAll         bool
	reviewLimit       int
)

// StaleThreshold defines how long an agent must be in a non-Complete phase to be considered stale.
const StaleThreshold = 24 * time.Hour

var reviewCmd = &cobra.Command{
	Use:   "review [beads-id]",
	Short: "Review agent work before completing",
	Long: `Review agent work before completing.

Without arguments: Shows actionable pending completions grouped by project.
With beads-id: Shows detailed review for a single agent.

By default, stale agents (in non-Complete phase for >24h) are excluded from
the output. Use --stale to see them, or --all to see everything.

Single-agent review shows:
  - SYNTHESIS.md summary (TLDR, outcome, recommendation)
  - Recent commits with stats
  - Beads comments history
  - Artifacts produced (investigations, design docs)

Examples:
  orch review                       # Actionable completions only (excludes stale)
  orch review --limit 5             # Show at most 5 completions
  orch review --all                 # Show everything including stale
  orch review --stale               # Show only stale agents
  orch review orch-go-3anf          # Single agent: detailed review
  orch review -p orch-cli           # Filter by project
  orch review --needs               # Show failures only (shorthand for --needs-review)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Single-agent mode if beads ID provided
		if len(args) > 0 {
			return runReviewSingle(args[0])
		}
		// Batch mode
		return runReview(reviewProject, reviewNeedsReview, reviewStale, reviewAll, reviewLimit)
	},
}

func init() {
	reviewCmd.Flags().StringVarP(&reviewProject, "project", "p", "", "Filter by project")
	reviewCmd.Flags().BoolVar(&reviewNeedsReview, "needs-review", false, "Show failures only")
	reviewCmd.Flags().BoolVar(&reviewNeedsReview, "needs", false, "Show failures only (shorthand for --needs-review)")
	reviewCmd.Flags().BoolVar(&reviewStale, "stale", false, "Show stale agents only")
	reviewCmd.Flags().BoolVar(&reviewAll, "all", false, "Show all agents including stale")
	reviewCmd.Flags().IntVarP(&reviewLimit, "limit", "l", 0, "Maximum number of completions to show (0 = no limit)")
	reviewCmd.AddCommand(reviewDoneCmd)
}

// CompletionInfo holds information about a completed agent for review.
type CompletionInfo struct {
	WorkspaceID     string // Workspace directory name
	WorkspacePath   string // Full path to workspace directory
	BeadsID         string // Beads issue ID
	Project         string
	VerifyOK        bool
	VerifyError     string
	Phase           string
	Summary         string
	Skill           string
	Synthesis       *verify.Synthesis
	ModTime         time.Time // Workspace modification time
	IsStale         bool      // True if agent is in non-Complete phase for >24h
	IsLightTier     bool      // True if agent was spawned as light tier (no SYNTHESIS.md by design)
	ReviewTier      string    // Review tier from manifest (auto/scan/review/deep)
	IsAutoCompleted       bool   // True if this was auto-completed by daemon (from events.jsonl)
	EmptyExecutionRetries int    // Count of empty-execution retries from events
	RecoveryOutcome       string // Recovery result: "recovered", "escalated", or ""
}

// getCompletionsForReview retrieves completed agents with verification status.
// Scans .orch/workspace/ for completed workspaces. Detects both:
//   - Full-tier agents: those with SYNTHESIS.md
//   - Light-tier agents: those with Tier "light" in the agent manifest (fallback to dotfiles)
//     AND Phase: Complete in beads comments
//
// Filters out completions whose beads issues are already closed (closed/deferred/tombstone).
//
// Performance optimization: Uses batch comment fetching to avoid O(n) beads API calls.
// Previously, each workspace with SYNTHESIS.md would call VerifyCompletionFull which called
// GetComments multiple times (for phase verification, test evidence, visual verification, etc.).
// Now, all comments are fetched in a single batch call, then passed to verification functions.
func getCompletionsForReview() ([]CompletionInfo, error) {
	projectDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Phase 1: Scan workspaces and collect candidates with beads IDs
	// Scans across all kb-registered project directories for cross-project visibility.
	type candidateWorkspace struct {
		dirName      string
		dirPath      string
		projectDir   string // Project directory this workspace belongs to
		hasSynthesis bool
		isLightTier  bool
		lightBeadsID string
		beadsID      string
		skill        string
		reviewTier   string
		modTime      time.Time
	}

	var workspaceCandidates []candidateWorkspace
	var beadsIDsToFetch []string
	beadsIDSet := make(map[string]bool) // Deduplicate beads IDs

	// Track light-tier beads IDs separately for batch fetching
	var lightTierBeadsIDs []string
	lightTierBeadsIDSet := make(map[string]bool)

	// Build list of project directories to scan.
	// Start with cwd, then add all kb-registered projects (deduplicated).
	projectDirsToScan := []string{projectDir}
	seenProjectDirs := map[string]bool{projectDir: true}
	for _, proj := range getKBProjectsWithNames() {
		if proj.Path != "" && !seenProjectDirs[proj.Path] {
			seenProjectDirs[proj.Path] = true
			projectDirsToScan = append(projectDirsToScan, proj.Path)
		}
	}

	// Track beads ID -> project dir for cross-project comment fetching
	beadsIDProjectDirs := make(map[string]string)

	for _, scanDir := range projectDirsToScan {
		workspaceDir := filepath.Join(scanDir, ".orch", "workspace")
		entries, _ := os.ReadDir(workspaceDir)

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

			// Read manifest once to extract tier info (avoids double-read)
			manifest := spawn.ReadAgentManifestWithFallback(dirPath)
			isLightTier := strings.TrimSpace(manifest.Tier) == spawn.TierLight
			reviewTier := manifest.ReviewTier
			if reviewTier == "" {
				// Infer from skill if not set in manifest
				reviewTier = spawn.DefaultReviewTier(extractSkillFromTitle(dirName), "")
			}

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

			workspaceCandidates = append(workspaceCandidates, candidateWorkspace{
				dirName:      dirName,
				dirPath:      dirPath,
				projectDir:   scanDir,
				hasSynthesis: hasSynthesis,
				isLightTier:  isLightTier, // Note: this is now the workspace tier, not completion status
				lightBeadsID: beadsID,     // Store beads ID for light-tier workspaces
				beadsID:      beadsID,
				skill:        skill,
				reviewTier:   reviewTier,
				modTime:      modTime,
			})

			// Collect beads IDs for batch fetching
			if beadsID != "" && !beadsIDSet[beadsID] {
				beadsIDsToFetch = append(beadsIDsToFetch, beadsID)
				beadsIDSet[beadsID] = true
				beadsIDProjectDirs[beadsID] = scanDir
			}

			// Also collect light-tier beads IDs for Phase: Complete check
			if isLightTier && beadsID != "" && !lightTierBeadsIDSet[beadsID] {
				lightTierBeadsIDs = append(lightTierBeadsIDs, beadsID)
				lightTierBeadsIDSet[beadsID] = true
			}
		}
	}

	// Phase 2: Batch fetch all comments for tracked workspaces
	// Uses project-dir-aware fetching for cross-project beads lookups.
	// Note: beadsIDsToFetch includes both full-tier and light-tier beads IDs
	commentsMap := verify.GetCommentsBatchWithProjectDirs(beadsIDsToFetch, beadsIDProjectDirs)

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
			Project:       extractProject(ws.projectDir),
			Skill:         ws.skill,
			ReviewTier:    ws.reviewTier,
			ModTime:       ws.modTime,
			IsLightTier:   isLightComplete, // Now reflects actual completion status, not just tier
		}

		// Handle full-tier agents (with SYNTHESIS.md)
		if ws.hasSynthesis {
			// Check verification status if we have a beads ID
			if ws.beadsID != "" {
				// Use lightweight verification for review (avoids O(n) git/build commands)
				// Full verification with git diff, build checks, etc. is done in orch complete
				comments := commentsMap[ws.beadsID]
				result, err := verify.VerifyCompletionForReview(ws.beadsID, ws.dirPath, "", comments)
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

	// Enrich candidates with empty-execution retry counts from events
	enrichRetryCountsFromEvents(candidates)

	// Filter out completions whose beads issues are already closed
	// This prevents showing NEEDS_REVIEW for issues that were force-closed
	return filterClosedIssues(candidates), nil
}

// enrichRetryCountsFromEvents scans recent events for empty-execution retries
// and populates retry counts on matching CompletionInfo entries.
func enrichRetryCountsFromEvents(candidates []CompletionInfo) {
	if len(candidates) == 0 {
		return
	}

	// Build lookup map of beads IDs to candidate indices
	beadsToIdx := make(map[string][]int)
	for i, c := range candidates {
		if c.BeadsID != "" {
			beadsToIdx[c.BeadsID] = append(beadsToIdx[c.BeadsID], i)
		}
	}
	if len(beadsToIdx) == 0 {
		return
	}

	eventsPath := events.DefaultLogPath()
	after := time.Now().Add(-72 * time.Hour)

	events.ScanEventsFromPath(eventsPath, after, time.Time{}, func(event events.Event) {
		if event.Type != events.EventTypeEmptyExecutionRetry {
			return
		}
		beadsID, _ := event.Data["beads_id"].(string)
		indices, ok := beadsToIdx[beadsID]
		if !ok {
			return
		}
		for _, idx := range indices {
			candidates[idx].EmptyExecutionRetries++
			if recovery, ok := event.Data["recovery"].(string); ok {
				candidates[idx].RecoveryOutcome = recovery
			}
		}
	})
}

// runReviewSingle displays detailed review information for a single agent.
func runReviewSingle(beadsID string) error {
	// Try to find workspace from current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	projectDir := cwd

	// Find workspace by beads ID (searches SPAWN_CONTEXT.md, not just directory name)
	workspacePath, _ := findWorkspaceByBeadsID(projectDir, beadsID)

	// Get review data
	review, err := verify.GetAgentReview(beadsID, workspacePath, projectDir)
	if err != nil {
		return fmt.Errorf("failed to get agent review: %w", err)
	}

	// Derive skill from workspace name if available
	if workspacePath != "" {
		review.Skill = extractSkillFromTitle(filepath.Base(workspacePath))
	}

	// Check if this is a light tier agent
	if workspacePath != "" {
		review.IsLightTier = isLightTierWorkspace(workspacePath)
	}

	// Display the review
	fmt.Print(verify.FormatAgentReview(review))

	// Print next steps
	fmt.Println("---")
	if review.Status == "Phase: Complete" {
		// Light tier agents are ready without SYNTHESIS.md
		if review.SynthesisExists || review.IsLightTier {
			fmt.Printf("Ready to complete: orch complete %s\n", beadsID)
		} else {
			fmt.Println("Missing: SYNTHESIS.md - agent should create this before completing")
			fmt.Printf("\nTo force completion: orch complete %s --force\n", beadsID)
		}
	} else {
		if !review.SynthesisExists && !review.IsLightTier {
			fmt.Println("Missing: SYNTHESIS.md - agent should create this before completing")
		}
		fmt.Println("Missing: Phase: Complete - agent should report via bd comment")
		fmt.Printf("\nTo force completion: orch complete %s --force\n", beadsID)
	}

	return nil
}

func runReview(projectFilter string, needsReviewOnly bool, staleOnly bool, showAll bool, limit int) error {
	completions, err := getCompletionsForReview()
	if err != nil {
		return err
	}

	// Merge recent auto-completed events (last 24h) so they appear in the review output.
	// These are agents the daemon closed automatically — shown for awareness.
	autoCompleted := getRecentAutoCompletions(24 * time.Hour)
	existingBeadsIDs := make(map[string]bool)
	for _, c := range completions {
		if c.BeadsID != "" {
			existingBeadsIDs[c.BeadsID] = true
		}
	}
	for _, ac := range autoCompleted {
		// Skip if this beads ID is already in the pending completions list
		if ac.BeadsID != "" && existingBeadsIDs[ac.BeadsID] {
			continue
		}
		project := "unknown"
		if ac.ProjectDir != "" {
			project = extractProject(ac.ProjectDir)
		}
		workspaceID := ac.Workspace
		if workspaceID == "" && ac.BeadsID != "" {
			workspaceID = ac.BeadsID
		}
		completions = append(completions, CompletionInfo{
			WorkspaceID:     workspaceID,
			BeadsID:         ac.BeadsID,
			Project:         project,
			VerifyOK:        true,
			Phase:           "Complete",
			Summary:         ac.Summary,
			ReviewTier:      ac.ReviewTier,
			ModTime:         ac.Timestamp,
			IsAutoCompleted: true,
		})
	}

	// Track counts before filtering for summary
	totalCount := len(completions)
	staleCount := 0
	for _, c := range completions {
		if c.IsStale {
			staleCount++
		}
	}

	// Filter by stale status
	// Default: exclude stale
	// --stale: show only stale
	// --all: show everything
	if !showAll {
		var filtered []CompletionInfo
		for _, c := range completions {
			if staleOnly {
				// Show only stale
				if c.IsStale {
					filtered = append(filtered, c)
				}
			} else {
				// Default: exclude stale
				if !c.IsStale {
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
		// Still show triage nudge even when no completions pending
		if triageCount := getTriageReviewCount(); triageCount > 0 {
			fmt.Printf("\n%s", formatTriageSummary(triageCount))
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

			// Add stale/untracked/light-tier/auto-completed indicators
			if c.IsAutoCompleted {
				status = "auto-completed"
			} else if c.IsStale {
				status = "STALE"
			} else if c.BeadsID == "" {
				status = "UNTRACKED"
			} else if c.IsLightTier {
				status = "LIGHT"
			}

			// Build tier badge
			tierBadge := ""
			if c.ReviewTier != "" {
				tierBadge = fmt.Sprintf(" {%s}", c.ReviewTier)
			}

			beadsInfo := ""
			if c.BeadsID != "" {
				beadsInfo = fmt.Sprintf(" (%s)", c.BeadsID)
			}

			fmt.Printf("  [%s]%s %s%s\n", status, tierBadge, c.WorkspaceID, beadsInfo)

			// Auto-completed: show one-line summary only
			if c.IsAutoCompleted {
				if c.Summary != "" {
					fmt.Printf("         %s\n", c.Summary)
				}
				continue
			}

			if c.VerifyOK && c.Summary != "" {
				fmt.Printf("         Phase: %s - %s\n", c.Phase, c.Summary)
			}

			// Scan-tier items: show SYNTHESIS TLDR inline (compact view)
			if c.ReviewTier == spawn.ReviewScan && c.Synthesis != nil && c.Synthesis.TLDR != "" {
				tldr := c.Synthesis.TLDR
				if len(tldr) > 120 {
					tldr = tldr[:117] + "..."
				}
				tldr = strings.ReplaceAll(tldr, "\n", " ")
				fmt.Printf("         TLDR: %s\n", tldr)
			} else if c.Synthesis != nil {
				// Display full Synthesis Card for review/deep tier
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

			// Show empty-execution retry telemetry if present
			if c.EmptyExecutionRetries > 0 {
				retryInfo := fmt.Sprintf("         Retries: %d empty-execution", c.EmptyExecutionRetries)
				if c.RecoveryOutcome != "" {
					retryInfo += fmt.Sprintf(" (recovery: %s)", c.RecoveryOutcome)
				}
				fmt.Println(retryInfo)
			}
		}
	}

	// Count auto-completed and untracked for summary
	autoCompletedCount := 0
	untrackedCount := 0
	for _, c := range completions {
		if c.IsAutoCompleted {
			autoCompletedCount++
		} else if c.BeadsID == "" && !c.IsStale {
			untrackedCount++
		}
	}

	// Print summary
	fmt.Printf("\n---\n")
	summaryParts := []string{fmt.Sprintf("%d OK", totalOK), fmt.Sprintf("%d need review", totalFailed)}
	if untrackedCount > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("%d untracked", untrackedCount))
	}
	if autoCompletedCount > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("%d auto-completed", autoCompletedCount))
	}
	fmt.Printf("Total: %d completions (%s)\n", totalOK+totalFailed, strings.Join(summaryParts, ", "))

	// Show truncation notice if limit was applied
	if limit > 0 && totalAfterFilters > limit {
		fmt.Printf("Showing: %d of %d (use --limit 0 or remove --limit to see all)\n", limit, totalAfterFilters)
	}

	// Show hidden counts if not showing all
	if !showAll && !staleOnly {
		if staleCount > 0 {
			fmt.Printf("Hidden: %d stale (use --stale to view, --all to include)\n", staleCount)
		}
	}

	// Show total breakdown if viewing stale only
	if staleOnly {
		fmt.Printf("Showing: %d stale\n", staleCount)
		actionableCount := totalCount - staleCount
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

	// Hygiene nudge: show triage:review count
	if triageCount := getTriageReviewCount(); triageCount > 0 {
		fmt.Printf("\n%s", formatTriageSummary(triageCount))
	}

	return nil
}

// AutoCompletedInfo holds information about an auto-completed agent from events.jsonl.
type AutoCompletedInfo struct {
	BeadsID    string
	Summary    string
	Timestamp  time.Time
	Workspace  string
	ReviewTier string
	ProjectDir string
}

// getRecentAutoCompletions reads recent auto-completed events from event files.
// Returns events within the given duration (e.g., 24h).
func getRecentAutoCompletions(since time.Duration) []AutoCompletedInfo {
	eventsPath := events.DefaultLogPath()
	after := time.Now().Add(-since)
	var results []AutoCompletedInfo

	events.ScanEventsFromPath(eventsPath, after, time.Time{}, func(event events.Event) {
		if event.Type != events.EventTypeAutoCompleted {
			return
		}

		info := AutoCompletedInfo{
			Timestamp: time.Unix(event.Timestamp, 0),
		}
		if v, ok := event.Data["beads_id"].(string); ok {
			info.BeadsID = v
		}
		if v, ok := event.Data["close_reason"].(string); ok {
			info.Summary = v
		}
		if v, ok := event.Data["workspace"].(string); ok {
			info.Workspace = v
		}
		if v, ok := event.Data["review_tier"].(string); ok {
			info.ReviewTier = v
		}
		if v, ok := event.Data["project_dir"].(string); ok {
			info.ProjectDir = v
		}

		results = append(results, info)
	})

	return results
}

// printSynthesisCard displays a condensed Synthesis Card for an agent.
// Shows the D.E.K.N. sections (Delta, Evidence, Knowledge, Next) in a compact format.
func printSynthesisCard(s *verify.Synthesis) {
	indent := "         "

	// TLDR is always shown if available
	if s.TLDR != "" {
		// Truncate TLDR if too long (single line display)
		tldr := s.TLDR
		if len(tldr) > 100 {
			tldr = tldr[:97] + "..."
		}
		// Replace newlines with spaces for single-line display
		tldr = strings.ReplaceAll(tldr, "\n", " ")
		fmt.Printf("%sTLDR:  %s\n", indent, tldr)
	}

	// Outcome and Recommendation (condensed line)
	if s.Outcome != "" || s.Recommendation != "" {
		var meta []string
		if s.Outcome != "" {
			meta = append(meta, fmt.Sprintf("outcome=%s", s.Outcome))
		}
		if s.Recommendation != "" {
			meta = append(meta, fmt.Sprintf("rec=%s", s.Recommendation))
		}
		fmt.Printf("%sStatus: %s\n", indent, strings.Join(meta, ", "))
	}

	// Delta summary (files changed, commits)
	if s.Delta != "" {
		deltaSummary := summarizeDelta(s.Delta)
		if deltaSummary != "" {
			fmt.Printf("%sDelta: %s\n", indent, deltaSummary)
		}
	}

	// Next Actions
	if len(s.NextActions) > 0 {
		fmt.Printf("%sNext:\n", indent)
		// Show at most 3 actions to keep it condensed
		maxActions := 3
		for i, action := range s.NextActions {
			if i >= maxActions {
				fmt.Printf("%s  ... +%d more\n", indent, len(s.NextActions)-maxActions)
				break
			}
			// Truncate long actions
			if len(action) > 80 {
				action = action[:77] + "..."
			}
			fmt.Printf("%s  %s\n", indent, action)
		}
	}
}

// summarizeDelta creates a one-line summary of the Delta section.
// Extracts file counts and commit info.
func summarizeDelta(delta string) string {
	var parts []string

	// Count files created
	createdCount := strings.Count(delta, "### Files Created")
	if createdCount > 0 {
		// Count bullet points in the section
		fileCount := countBulletPoints(delta, "### Files Created")
		if fileCount > 0 {
			parts = append(parts, fmt.Sprintf("%d files created", fileCount))
		}
	}

	// Count files modified
	modifiedCount := strings.Count(delta, "### Files Modified")
	if modifiedCount > 0 {
		fileCount := countBulletPoints(delta, "### Files Modified")
		if fileCount > 0 {
			parts = append(parts, fmt.Sprintf("%d files modified", fileCount))
		}
	}

	// Count commits
	commitsCount := strings.Count(delta, "### Commits")
	if commitsCount > 0 {
		commitCount := countBulletPoints(delta, "### Commits")
		if commitCount > 0 {
			parts = append(parts, fmt.Sprintf("%d commits", commitCount))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, ", ")
}

// countBulletPoints counts bullet points (-) after a section header.
func countBulletPoints(content, sectionHeader string) int {
	idx := strings.Index(content, sectionHeader)
	if idx == -1 {
		return 0
	}

	// Find content after header
	afterHeader := content[idx+len(sectionHeader):]

	// Find end (next ### or end of content)
	endIdx := strings.Index(afterHeader, "\n###")
	if endIdx == -1 {
		endIdx = len(afterHeader)
	}

	section := afterHeader[:endIdx]

	// Count lines starting with -
	count := 0
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
			count++
		}
	}

	return count
}
