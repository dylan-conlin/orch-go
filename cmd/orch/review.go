// Package main provides the CLI entry point for orch-go.
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	// Review command flags
	reviewProject     string
	reviewNeedsReview bool
	reviewDoneYes     bool
	reviewStale       bool
	reviewAll         bool
	reviewNoPrompt    bool
	reviewLimit       int
	reviewArchitects  bool
)

// StaleThreshold defines how long an agent must be in a non-Complete phase to be considered stale.
const StaleThreshold = 24 * time.Hour

var reviewCmd = &cobra.Command{
	Use:   "review [beads-id]",
	Short: "Review agent work before completing",
	Long: `Review agent work before completing.

Without arguments: Shows actionable pending completions grouped by project.
With beads-id: Shows detailed review for a single agent.

By default, stale agents (in non-Complete phase for >24h) and untracked agents
(spawned with --no-track) are excluded from the output. Use --stale to see them,
or --all to see everything.

Single-agent review shows:
  - SYNTHESIS.md summary (TLDR, outcome, recommendation)
  - Recent commits with stats
  - Beads comments history
  - Artifacts produced (investigations, design docs)

Examples:
  orch review                       # Actionable completions only (excludes stale/untracked)
  orch review --limit 5             # Show at most 5 completions
  orch review --all                 # Show everything including stale/untracked
  orch review --stale               # Show only stale/untracked agents
  orch review orch-go-3anf          # Single agent: detailed review
  orch review -p orch-cli           # Filter by project
  orch review --needs               # Show failures only (shorthand for --needs-review)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Single-agent mode if beads ID provided
		if len(args) > 0 {
			return runReviewSingle(args[0])
		}
		// Architect mode shows only architect recommendations
		if reviewArchitects {
			return runReviewArchitects(reviewProject, reviewLimit)
		}
		// Batch mode
		return runReview(reviewProject, reviewNeedsReview, reviewStale, reviewAll, reviewLimit)
	},
}

var reviewDoneCmd = &cobra.Command{
	Use:   "done [project]",
	Short: "Complete all agents for a project",
	Long: `Complete all agents for a project by closing their beads issues.

This runs the completion workflow for each agent with Phase: Complete status,
closing the beads issue and cleaning up resources.

For each agent with synthesis recommendations (NextActions in SYNTHESIS.md),
you'll be prompted to create follow-up issues:
  - y: Create beads issues for all recommendations
  - n: Skip this agent's recommendations
  - skip-all: Skip prompts for all remaining agents

Use --no-prompt to skip all recommendation prompts (for automation/scripting).

Agents that fail verification (no Phase: Complete) will be skipped.

Examples:
  orch-go review done orch-cli           # Complete with recommendation prompts
  orch-go review done orch-cli -y        # Skip initial confirmation
  orch-go review done orch-cli --no-prompt  # Skip recommendation prompts`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReviewDone(args[0])
	},
}

func init() {
	reviewCmd.Flags().StringVarP(&reviewProject, "project", "p", "", "Filter by project")
	reviewCmd.Flags().BoolVar(&reviewNeedsReview, "needs-review", false, "Show failures only")
	reviewCmd.Flags().BoolVar(&reviewNeedsReview, "needs", false, "Show failures only (shorthand for --needs-review)")
	reviewCmd.Flags().BoolVar(&reviewStale, "stale", false, "Show stale/untracked agents only")
	reviewCmd.Flags().BoolVar(&reviewAll, "all", false, "Show all agents including stale/untracked")
	reviewCmd.Flags().IntVarP(&reviewLimit, "limit", "l", 0, "Maximum number of completions to show (0 = no limit)")
	reviewCmd.Flags().BoolVar(&reviewArchitects, "architects", false, "Show only architect agents with unreviewed recommendations")
	reviewDoneCmd.Flags().BoolVarP(&reviewDoneYes, "yes", "y", false, "Skip confirmation prompt")
	reviewDoneCmd.Flags().BoolVar(&reviewNoPrompt, "no-prompt", false, "Skip recommendation prompts (auto-close without reviewing synthesis)")
	reviewCmd.AddCommand(reviewDoneCmd)
}

// CompletionInfo holds information about a completed agent for review.
type CompletionInfo struct {
	WorkspaceID   string // Workspace directory name
	WorkspacePath string // Full path to workspace directory
	BeadsID       string // Beads issue ID
	Project       string
	VerifyOK      bool
	VerifyError   string
	Phase         string
	Summary       string
	Skill         string
	Synthesis     *verify.Synthesis
	ModTime       time.Time // Workspace modification time
	IsUntracked   bool      // True if agent was spawned with --no-track
	IsStale       bool      // True if agent is in non-Complete phase for >24h
	IsLightTier   bool      // True if agent was spawned as light tier (no SYNTHESIS.md by design)
}

// getCompletionsForReview retrieves completed agents with verification status.
// Scans .orch/workspace/ for completed workspaces. Detects both:
// - Full-tier agents: those with SYNTHESIS.md
// - Light-tier agents: those with .tier file containing "light" AND Phase: Complete in beads comments
// Filters out completions whose beads issues are already closed (closed/deferred/tombstone).
func getCompletionsForReview() ([]CompletionInfo, error) {
	projectDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	var candidates []CompletionInfo

	// Scan workspaces for completions
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
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

		// Check for light-tier completion (no synthesis by design)
		isLightComplete, lightBeadsID := isLightTierComplete(dirPath)

		// Skip workspaces that are neither full-tier with synthesis nor light-tier complete
		if !hasSynthesis && !isLightComplete {
			continue
		}

		// Get workspace modification time from directory
		dirInfo, err := entry.Info()
		modTime := time.Now()
		if err == nil {
			modTime = dirInfo.ModTime()
		}

		// Extract beads ID from SPAWN_CONTEXT.md (or use lightBeadsID for light tier)
		beadsID := extractBeadsIDFromWorkspace(dirPath)
		if beadsID == "" && lightBeadsID != "" {
			beadsID = lightBeadsID
		}

		// Extract skill from workspace name
		skill := extractSkillFromTitle(dirName)

		// Detect if agent is untracked
		isUntracked := isUntrackedBeadsID(beadsID)

		info := CompletionInfo{
			WorkspaceID:   dirName,
			WorkspacePath: dirPath,
			BeadsID:       beadsID,
			Project:       extractProject(projectDir),
			Skill:         skill,
			ModTime:       modTime,
			IsUntracked:   isUntracked,
			IsLightTier:   isLightComplete,
		}

		// Handle full-tier agents (with SYNTHESIS.md)
		if hasSynthesis {
			// Check verification status if we have a beads ID
			if beadsID != "" {
				result, err := verify.VerifyCompletionFull(beadsID, dirPath, projectDir, "")
				if err != nil {
					info.VerifyError = fmt.Sprintf("verification error: %v", err)
					info.VerifyOK = false
				} else if result.Passed {
					info.VerifyOK = true
					info.Phase = result.Phase.Phase
					info.Summary = result.Phase.Summary

					// Try to parse synthesis
					s, err := verify.ParseSynthesis(dirPath)
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
				s, err := verify.ParseSynthesis(dirPath)
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
// Uses batch fetching for efficiency. If beads is unavailable, returns all candidates
// (better to show potential false positives than hide real issues).
func filterClosedIssues(candidates []CompletionInfo) []CompletionInfo {
	if len(candidates) == 0 {
		return candidates
	}

	// Collect all beads IDs for batch fetch
	beadsIDs := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if c.BeadsID != "" && !c.IsUntracked {
			beadsIDs = append(beadsIDs, c.BeadsID)
		}
	}

	if len(beadsIDs) == 0 {
		return candidates
	}

	// Batch fetch issue statuses
	issueMap, _ := verify.GetIssuesBatch(beadsIDs)
	// Ignore error - if beads is unavailable, return all candidates

	// Filter out closed issues
	var results []CompletionInfo
	for _, c := range candidates {
		// Keep untracked agents (no beads issue to check)
		if c.IsUntracked || c.BeadsID == "" {
			results = append(results, c)
			continue
		}

		// Check if issue is closed
		if issue, ok := issueMap[c.BeadsID]; ok {
			status := strings.ToLower(issue.Status)
			if status == "closed" || status == "deferred" || status == "tombstone" {
				// Skip closed issues - they're resolved and shouldn't appear in review
				continue
			}
		}

		results = append(results, c)
	}

	return results
}

// getCompletionsForSurfacing retrieves completed agents WITHOUT running expensive verification.
// This is a lightweight version of getCompletionsForReview designed for surfacing recommendations
// in orch status where we don't need full verification (constraint checks, phase gates, etc.).
//
// Parses SYNTHESIS.md and workspace metadata, then filters out closed beads issues.
// The beads filtering is necessary to avoid showing stale recommendations for resolved issues.
// Use getCompletionsForReview for actual review workflows that need verification status.
func getCompletionsForSurfacing() ([]CompletionInfo, error) {
	projectDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	var results []CompletionInfo

	// Scan workspaces for completions
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
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

		// Skip workspaces without SYNTHESIS.md (surfacing only needs synthesis data)
		if !hasSynthesis {
			continue
		}

		// Get workspace modification time from directory
		dirInfo, err := entry.Info()
		modTime := time.Now()
		if err == nil {
			modTime = dirInfo.ModTime()
		}

		// Extract beads ID from SPAWN_CONTEXT.md
		beadsID := extractBeadsIDFromWorkspace(dirPath)

		// Extract skill from workspace name
		skill := extractSkillFromTitle(dirName)

		// Detect if agent is untracked
		isUntracked := isUntrackedBeadsID(beadsID)

		info := CompletionInfo{
			WorkspaceID:   dirName,
			WorkspacePath: dirPath,
			BeadsID:       beadsID,
			Project:       extractProject(projectDir),
			Skill:         skill,
			ModTime:       modTime,
			IsUntracked:   isUntracked,
		}

		// Parse synthesis (the main thing we need for surfacing)
		s, err := verify.ParseSynthesis(dirPath)
		if err == nil {
			info.Synthesis = s
		}

		results = append(results, info)
	}

	// Filter out completions whose beads issues are already closed
	// This prevents showing stale recommendations for resolved issues
	return filterClosedIssues(results), nil
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

// extractProjectDirFromWorkspace extracts the PROJECT_DIR from SPAWN_CONTEXT.md
// This is used to determine which project's beads database to query for cross-project agents.
func extractProjectDirFromWorkspace(workspacePath string) string {
	spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	content, err := os.ReadFile(spawnContextPath)
	if err != nil {
		return ""
	}

	// Look for "PROJECT_DIR: /path/to/project" pattern
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "PROJECT_DIR:") {
			// Extract path after "PROJECT_DIR:"
			path := strings.TrimPrefix(line, "PROJECT_DIR:")
			path = strings.TrimSpace(path)
			return path
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

// isUntrackedBeadsID returns true if the beads ID indicates an untracked agent.
// Untracked agents have IDs like "orch-go-untracked-1766695797".
func isUntrackedBeadsID(beadsID string) bool {
	return strings.Contains(beadsID, "-untracked-")
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

func runReviewDone(project string) error {
	completions, err := getCompletionsForReview()
	if err != nil {
		return err
	}

	// Filter by project
	var projectCompletions []CompletionInfo
	for _, c := range completions {
		if c.Project == project {
			projectCompletions = append(projectCompletions, c)
		}
	}

	if len(projectCompletions) == 0 {
		fmt.Printf("No pending completions for project: %s\n", project)
		return nil
	}

	// Count by verification status
	var canComplete []CompletionInfo
	var needsReview []CompletionInfo
	for _, c := range projectCompletions {
		if c.VerifyOK && c.BeadsID != "" {
			canComplete = append(canComplete, c)
		} else {
			needsReview = append(needsReview, c)
		}
	}

	// Show summary before proceeding
	fmt.Printf("Project: %s\n", project)
	fmt.Printf("  Ready to complete: %d\n", len(canComplete))
	fmt.Printf("  Needs manual review: %d\n", len(needsReview))

	if len(canComplete) == 0 {
		fmt.Println("\nNo agents ready to complete (need Phase: Complete and valid beads ID)")
		if len(needsReview) > 0 {
			fmt.Println("\nAgents needing manual review:")
			for _, c := range needsReview {
				reason := "missing beads ID"
				if c.BeadsID != "" {
					reason = "verification failed"
					if c.VerifyError != "" {
						reason = c.VerifyError
					}
				}
				fmt.Printf("  - %s: %s\n", c.WorkspaceID, reason)
			}
		}
		return nil
	}

	// Confirmation prompt unless --yes flag is set
	if !reviewDoneYes {
		fmt.Printf("\nThis will close %d beads issues and clean up resources.\n", len(canComplete))
		fmt.Print("Continue? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			return fmt.Errorf("aborted")
		}
	}

	// Process each completion
	completed := 0
	var completionErrors []string
	skipAllPrompts := reviewNoPrompt // Start with flag value

	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	logger := events.NewLogger(events.DefaultLogPath())
	reader := bufio.NewReader(os.Stdin)

	for _, c := range canComplete {
		fmt.Printf("\nCompleting: %s (%s)\n", c.WorkspaceID, c.BeadsID)

		// Track which recommendations were acted on vs dismissed for review state
		var actedOnIndices []int
		var dismissedIndices []int
		totalRecommendations := 0

		// Prompt for recommendations unless --no-prompt or user chose skip-all
		if c.Synthesis != nil && len(c.Synthesis.NextActions) > 0 {
			totalRecommendations = len(c.Synthesis.NextActions)

			if !skipAllPrompts {
				fmt.Printf("\n  Has %d recommendations:\n", totalRecommendations)
				for i, action := range c.Synthesis.NextActions {
					// Truncate long actions for display
					display := action
					if len(display) > 100 {
						display = display[:97] + "..."
					}
					fmt.Printf("    %d. %s\n", i+1, display)
				}
				fmt.Print("\n  Create follow-up issues? [y/n/skip-all]: ")

				response, err := reader.ReadString('\n')
				if err != nil {
					fmt.Printf("  Warning: failed to read response, skipping prompts: %v\n", err)
					skipAllPrompts = true
					// Mark all as dismissed when skipping due to error
					for i := 0; i < totalRecommendations; i++ {
						dismissedIndices = append(dismissedIndices, i)
					}
				} else {
					response = strings.TrimSpace(strings.ToLower(response))
					switch response {
					case "y", "yes":
						// Create beads issues for each recommendation
						for i, action := range c.Synthesis.NextActions {
							title := action
							if len(title) > 80 {
								title = title[:77] + "..."
							}
							fmt.Printf("  Creating issue: %s\n", title)
							// Use bd create to create follow-up issue
							if err := createFollowUpIssue(title, c.WorkspaceID); err != nil {
								fmt.Printf("    Warning: failed to create issue: %v\n", err)
								// Still count as acted on even if creation failed
							}
							actedOnIndices = append(actedOnIndices, i)
						}
					case "skip-all", "s":
						fmt.Println("  Skipping prompts for remaining agents")
						skipAllPrompts = true
						// Mark all as dismissed
						for i := 0; i < totalRecommendations; i++ {
							dismissedIndices = append(dismissedIndices, i)
						}
					case "n", "no", "":
						// Skip this agent's recommendations, continue to close
						fmt.Println("  Skipping recommendations")
						// Mark all as dismissed
						for i := 0; i < totalRecommendations; i++ {
							dismissedIndices = append(dismissedIndices, i)
						}
					default:
						fmt.Printf("  Unknown response '%s', skipping recommendations\n", response)
						// Mark all as dismissed
						for i := 0; i < totalRecommendations; i++ {
							dismissedIndices = append(dismissedIndices, i)
						}
					}
				}
			} else {
				// --no-prompt flag: mark all as dismissed
				for i := 0; i < totalRecommendations; i++ {
					dismissedIndices = append(dismissedIndices, i)
				}
			}

			// Persist review state to workspace
			if c.WorkspacePath != "" {
				reviewState := verify.ReviewStateFromCompletion(
					c.WorkspaceID,
					c.BeadsID,
					totalRecommendations,
					actedOnIndices,
					dismissedIndices,
				)
				if err := verify.SaveReviewState(c.WorkspacePath, reviewState); err != nil {
					fmt.Printf("  Warning: failed to save review state: %v\n", err)
				}
			}
		}

		// Check if already closed
		issue, err := verify.GetIssue(c.BeadsID)
		if err != nil {
			completionErrors = append(completionErrors, fmt.Sprintf("%s: failed to get issue: %v", c.BeadsID, err))
			continue
		}
		if issue.Status == "closed" {
			fmt.Printf("  Already closed, skipping beads close\n")
		} else {
			// Determine close reason from phase summary
			reason := "Completed via orch review done"
			if c.Summary != "" {
				reason = c.Summary
			}

			// Close the beads issue
			if err := verify.CloseIssue(c.BeadsID, reason); err != nil {
				completionErrors = append(completionErrors, fmt.Sprintf("%s: failed to close: %v", c.BeadsID, err))
				continue
			}
			fmt.Printf("  Closed beads issue\n")
		}

		// Clean up tmux window if it exists
		if window, sessionName, err := tmux.FindWindowByBeadsIDAllSessions(c.BeadsID); err == nil && window != nil {
			if err := tmux.KillWindow(window.Target); err != nil {
				fmt.Printf("  Warning: failed to close tmux window: %v\n", err)
			} else {
				fmt.Printf("  Closed tmux window: %s:%s\n", sessionName, window.Name)
			}
		}

		// Log the completion
		event := events.Event{
			Type:      "agent.completed",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"beads_id":    c.BeadsID,
				"workspace":   c.WorkspaceID,
				"reason":      c.Summary,
				"batch":       true,
				"source":      "review_done",
				"project_dir": projectDir,
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Printf("  Warning: failed to log event: %v\n", err)
		}

		completed++
	}

	// Summary
	fmt.Printf("\n---\n")
	fmt.Printf("Completed: %d/%d agents\n", completed, len(canComplete))

	if len(completionErrors) > 0 {
		fmt.Fprintf(os.Stderr, "\nErrors (%d):\n", len(completionErrors))
		for _, e := range completionErrors {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
	}

	if len(needsReview) > 0 {
		fmt.Printf("\nAgents needing manual review (%d):\n", len(needsReview))
		for _, c := range needsReview {
			reason := "missing beads ID"
			if c.BeadsID != "" {
				reason = "verification failed"
			}
			fmt.Printf("  - %s: %s\n", c.WorkspaceID, reason)
		}
	}

	return nil
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

// createFollowUpIssue creates a beads issue for a synthesis recommendation.
// Uses bd create command to create the issue with appropriate labels.
func createFollowUpIssue(title string, sourceWorkspace string) error {
	// Clean up the title - remove leading bullet markers
	title = strings.TrimPrefix(title, "- ")
	title = strings.TrimPrefix(title, "* ")
	title = strings.TrimSpace(title)

	// Create description linking back to source
	description := fmt.Sprintf("Follow-up from synthesis review of %s", sourceWorkspace)

	// Find bd command
	bdPath, err := findBdCommand()
	if err != nil {
		return fmt.Errorf("bd command not found: %w", err)
	}

	// Run bd create with triage:review label (needs orchestrator review before spawning)
	args := []string{"create", title, "-d", description, "-l", "triage:review"}
	cmd := exec.Command(bdPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd create failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// findBdCommand locates the bd binary.
func findBdCommand() (string, error) {
	// Try common locations
	paths := []string{
		filepath.Join(os.Getenv("HOME"), "bin", "bd"),
		filepath.Join(os.Getenv("HOME"), "go", "bin", "bd"),
		filepath.Join(os.Getenv("HOME"), ".local", "bin", "bd"),
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// Try PATH
	if path, err := exec.LookPath("bd"); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("bd not found in common locations or PATH")
}

// ArchitectRecommendation represents an unreviewed recommendation from an architect agent.
type ArchitectRecommendation struct {
	WorkspaceID   string // Workspace directory name
	WorkspacePath string // Full path to workspace directory
	BeadsID       string // Beads issue ID
	Project       string
	Skill         string
	TLDR          string   // Brief summary of the design work
	Recommendation string  // Recommended action (spawn-follow-up, close, etc.)
	NextActions   []string // Specific follow-up items
	ModTime       time.Time
}

// runReviewArchitects displays architect agents with unreviewed recommendations.
// This surfaces strategic design work that needs orchestrator attention.
func runReviewArchitects(projectFilter string, limit int) error {
	completions, err := getCompletionsForReview()
	if err != nil {
		return err
	}

	// Filter for architect skill with recommendations
	var architects []ArchitectRecommendation
	for _, c := range completions {
		// Filter by architect skill
		if c.Skill != "architect" {
			continue
		}

		// Filter by project if specified
		if projectFilter != "" && c.Project != projectFilter {
			continue
		}

		// Skip untracked agents
		if c.IsUntracked {
			continue
		}

		// Must have synthesis with recommendations
		if c.Synthesis == nil {
			continue
		}

		// Count actionable items (NextActions, AreasToExplore, Uncertainties)
		totalItems := len(c.Synthesis.NextActions) + len(c.Synthesis.AreasToExplore) + len(c.Synthesis.Uncertainties)
		if totalItems == 0 {
			continue
		}

		architects = append(architects, ArchitectRecommendation{
			WorkspaceID:    c.WorkspaceID,
			WorkspacePath:  c.WorkspacePath,
			BeadsID:        c.BeadsID,
			Project:        c.Project,
			Skill:          c.Skill,
			TLDR:           c.Synthesis.TLDR,
			Recommendation: c.Synthesis.Recommendation,
			NextActions:    c.Synthesis.NextActions,
			ModTime:        c.ModTime,
		})
	}

	// Apply limit if specified
	totalFound := len(architects)
	if limit > 0 && len(architects) > limit {
		architects = architects[:limit]
	}

	if len(architects) == 0 {
		if projectFilter != "" {
			fmt.Printf("No architect recommendations awaiting review for project: %s\n", projectFilter)
		} else {
			fmt.Println("No architect recommendations awaiting review")
		}
		return nil
	}

	// Print prominent header
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Printf("│  📐 ARCHITECT RECOMMENDATIONS AWAITING REVIEW (%d)           │\n", totalFound)
	fmt.Println("├─────────────────────────────────────────────────────────────┤")

	// Group by project
	byProject := make(map[string][]ArchitectRecommendation)
	for _, a := range architects {
		byProject[a.Project] = append(byProject[a.Project], a)
	}

	// Get sorted project names
	var projects []string
	for p := range byProject {
		projects = append(projects, p)
	}
	sort.Strings(projects)

	for _, project := range projects {
		items := byProject[project]
		fmt.Printf("│                                                             │\n")
		fmt.Printf("│  ## %s (%d recommendations)                                  \n", project, len(items))

		for _, a := range items {
			// Truncate TLDR for display
			tldr := a.TLDR
			if len(tldr) > 50 {
				tldr = tldr[:47] + "..."
			}
			tldr = strings.ReplaceAll(tldr, "\n", " ")

			fmt.Printf("│    - %s", a.WorkspaceID)
			if a.BeadsID != "" {
				fmt.Printf(" (%s)", a.BeadsID)
			}
			fmt.Println()
			if tldr != "" {
				fmt.Printf("│      TLDR: %s\n", tldr)
			}
			fmt.Printf("│      Items: %d recommendations\n", len(a.NextActions))
		}
	}

	fmt.Println("│                                                             │")
	fmt.Println("├─────────────────────────────────────────────────────────────┤")
	fmt.Println("│  Review with: orch review <beads-id>                        │")
	fmt.Println("│  Complete:    orch complete <beads-id>                      │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	if limit > 0 && totalFound > limit {
		fmt.Printf("\nShowing %d of %d (use --limit 0 to see all)\n", limit, totalFound)
	}

	return nil
}

// GetArchitectRecommendationCount returns the count of architect agents with unreviewed recommendations.
// Used by orch status to surface pending architect work.
// Uses getCompletionsForSurfacing() to avoid expensive verification overhead.
func GetArchitectRecommendationCount() (int, error) {
	completions, err := getCompletionsForSurfacing()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, c := range completions {
		// Filter for architect skill
		if c.Skill != "architect" {
			continue
		}

		// Skip untracked agents
		if c.IsUntracked {
			continue
		}

		// Must have synthesis with recommendations
		if c.Synthesis == nil {
			continue
		}

		// Count actionable items (NextActions, AreasToExplore, Uncertainties)
		totalItems := len(c.Synthesis.NextActions) + len(c.Synthesis.AreasToExplore) + len(c.Synthesis.Uncertainties)
		if totalItems > 0 {
			count++
		}
	}

	return count, nil
}

// ArchitectRecommendationSummary contains a brief summary of a single architect recommendation
// suitable for display in orch status SessionStart surfacing.
type ArchitectRecommendationSummary struct {
	WorkspaceID string // Workspace directory name (e.g., "de-bloat-feature")
	TLDR        string // Brief summary (e.g., "Feature-impl skill 71% reduction")
	ItemCount   int    // Number of actionable items
}

// ArchitectRecommendationsSurface contains all information needed for SessionStart surfacing
// of unreviewed architect recommendations.
type ArchitectRecommendationsSurface struct {
	TotalCount int                              // Total number of architect recommendations awaiting review
	Summaries  []ArchitectRecommendationSummary // Individual recommendation summaries (limited to first 5)
}

// GetArchitectRecommendationsSurface returns structured data for surfacing architect recommendations
// in orch status. This provides the rich detail needed for SessionStart awareness of pending
// high-value design work.
//
// Uses getCompletionsForSurfacing() which skips expensive verification - we only need synthesis
// data for surfacing, not full verification status. This makes the function fast enough for
// use in orch status (< 1s vs 1m+ with full verification).
//
// Returns summaries of up to 5 recommendations to keep output manageable. Use orch review --architects
// for the full list.
func GetArchitectRecommendationsSurface() (*ArchitectRecommendationsSurface, error) {
	completions, err := getCompletionsForSurfacing()
	if err != nil {
		return nil, err
	}

	var recommendations []ArchitectRecommendationSummary
	for _, c := range completions {
		// Filter for architect skill
		if c.Skill != "architect" {
			continue
		}

		// Skip untracked agents
		if c.IsUntracked {
			continue
		}

		// Must have synthesis with recommendations
		if c.Synthesis == nil {
			continue
		}

		// Count actionable items (NextActions, AreasToExplore, Uncertainties)
		totalItems := len(c.Synthesis.NextActions) + len(c.Synthesis.AreasToExplore) + len(c.Synthesis.Uncertainties)
		if totalItems == 0 {
			continue
		}

		// Extract a concise workspace identifier from the full workspace ID
		// e.g., "og-arch-de-bloat-feature-27dec" -> "de-bloat-feature"
		workspaceShort := extractShortWorkspaceName(c.WorkspaceID)

		// Clean up TLDR for single-line display
		tldr := c.Synthesis.TLDR
		tldr = strings.ReplaceAll(tldr, "\n", " ")
		if len(tldr) > 60 {
			tldr = tldr[:57] + "..."
		}

		recommendations = append(recommendations, ArchitectRecommendationSummary{
			WorkspaceID: workspaceShort,
			TLDR:        tldr,
			ItemCount:   totalItems,
		})
	}

	// Limit to first 5 for display brevity
	displayed := recommendations
	if len(displayed) > 5 {
		displayed = displayed[:5]
	}

	return &ArchitectRecommendationsSurface{
		TotalCount: len(recommendations),
		Summaries:  displayed,
	}, nil
}

// extractShortWorkspaceName extracts a concise name from a full workspace ID.
// e.g., "og-arch-de-bloat-feature-27dec" -> "de-bloat-feature"
// e.g., "og-feat-beads-integration-27dec" -> "beads-integration"
func extractShortWorkspaceName(workspaceID string) string {
	// Remove common prefixes (og-, kb-, etc.) and date suffixes
	parts := strings.Split(workspaceID, "-")
	if len(parts) <= 2 {
		return workspaceID
	}

	// Skip first part if it's a project prefix (2-3 chars like "og", "kb")
	start := 0
	if len(parts[0]) <= 3 {
		start = 1
	}

	// Skip second part if it's a skill type (arch, feat, inv, debug)
	if start < len(parts) {
		skillTypes := map[string]bool{"arch": true, "feat": true, "inv": true, "debug": true, "research": true, "audit": true}
		if skillTypes[parts[start]] {
			start++
		}
	}

	// Skip last part if it looks like a date (e.g., "27dec", "21dec")
	end := len(parts)
	if end > 0 {
		lastPart := parts[end-1]
		// Check if it's a date pattern: digits followed by month abbreviation
		if len(lastPart) >= 3 && len(lastPart) <= 6 {
			hasDigit := false
			for _, c := range lastPart {
				if c >= '0' && c <= '9' {
					hasDigit = true
					break
				}
			}
			if hasDigit {
				end--
			}
		}
	}

	if start >= end {
		return workspaceID
	}

	return strings.Join(parts[start:end], "-")
}

// FormatArchitectRecommendationsSurface formats the architect recommendations surface
// for display in orch status. Returns an empty string if there are no recommendations.
func FormatArchitectRecommendationsSurface(surface *ArchitectRecommendationsSurface) string {
	if surface == nil || surface.TotalCount == 0 {
		return ""
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("⚠️  %d architect recommendation(s) awaiting review:\n", surface.TotalCount))

	for _, s := range surface.Summaries {
		sb.WriteString(fmt.Sprintf("  - %s (%d items)", s.WorkspaceID, s.ItemCount))
		if s.TLDR != "" {
			sb.WriteString(fmt.Sprintf(": %s", s.TLDR))
		}
		sb.WriteString("\n")
	}

	if surface.TotalCount > len(surface.Summaries) {
		sb.WriteString(fmt.Sprintf("  ... and %d more\n", surface.TotalCount-len(surface.Summaries)))
	}

	sb.WriteString("Run 'orch review --architects' to process\n")

	return sb.String()
}
