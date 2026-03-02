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
	"golang.org/x/term"
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
		modTime      time.Time
		isUntracked  bool
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

			// Check if this is a light-tier workspace (tier in manifest, fallback to dotfiles)
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
				projectDir:   scanDir,
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
				beadsIDProjectDirs[beadsID] = scanDir
			}

			// Also collect light-tier beads IDs for Phase: Complete check
			if isLightTier && beadsID != "" && !isUntracked && !lightTierBeadsIDSet[beadsID] {
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

// extractBeadsIDFromWorkspace extracts the beads ID from workspace files.
// Checks sources in order of reliability:
// 1. .beads_id file (written directly by spawn code)
// 2. AGENT_MANIFEST.json (has beads_id field)
// 3. SPAWN_CONTEXT.md "beads issue:" pattern (legacy fallback)
func extractBeadsIDFromWorkspace(workspacePath string) string {
	// Source 1: .beads_id file (most reliable - written directly by spawn code)
	beadsIDPath := filepath.Join(workspacePath, ".beads_id")
	if data, err := os.ReadFile(beadsIDPath); err == nil {
		id := strings.TrimSpace(string(data))
		if id != "" {
			return id
		}
	}

	// Source 2: AGENT_MANIFEST.json
	manifestPath := filepath.Join(workspacePath, "AGENT_MANIFEST.json")
	if data, err := os.ReadFile(manifestPath); err == nil {
		// Simple extraction - avoid importing encoding/json just for one field
		// Look for "beads_id": "value" in the JSON
		content := string(data)
		if idx := strings.Index(content, `"beads_id"`); idx != -1 {
			// Find the value after the colon
			rest := content[idx+len(`"beads_id"`):]
			// Skip whitespace and colon
			rest = strings.TrimLeft(rest, " \t\n:")
			// Extract quoted value
			if len(rest) > 0 && rest[0] == '"' {
				end := strings.Index(rest[1:], `"`)
				if end > 0 {
					id := rest[1 : end+1]
					if id != "" {
						return id
					}
				}
			}
		}
	}

	// Source 3: SPAWN_CONTEXT.md (legacy fallback for older workspaces)
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

	// Hygiene nudge: show triage:review count
	if triageCount := getTriageReviewCount(); triageCount > 0 {
		fmt.Printf("\n%s", formatTriageSummary(triageCount))
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

	// Confirmation prompt unless --yes flag is set or stdin is not a terminal
	if !reviewDoneYes {
		// Auto-skip confirmation when stdin is not a terminal (e.g., daemon, scripts)
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Printf("\nThis will close %d beads issues and clean up resources.\n", len(canComplete))
			fmt.Println("(Skipping confirmation - stdin is not a terminal)")
		} else {
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
	}

	// Process each completion
	completed := 0
	var completionErrors []string
	// Auto-skip prompts when stdin is not a terminal (e.g., daemon, scripts)
	skipAllPrompts := reviewNoPrompt || !term.IsTerminal(int(os.Stdin.Fd()))
	if !reviewNoPrompt && skipAllPrompts {
		fmt.Println("(Skipping recommendation prompts - stdin is not a terminal)")
	}

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
			if err := tmux.KillWindowByID(window.ID); err != nil {
				fmt.Printf("  Warning: failed to close tmux window: %v\n", err)
			} else {
				fmt.Printf("  Closed tmux window: %s:%s (%s)\n", sessionName, window.Name, window.ID)
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

	// Run bd create with triage:ready label — discovered work filed during orchestrator
	// review has already been reviewed; it's ready for daemon pickup, not further triage.
	args := []string{"create", title, "-d", description, "-l", "triage:ready"}
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
