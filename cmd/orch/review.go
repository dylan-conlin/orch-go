// Package main provides the CLI entry point for orch-go.
package main

import (
	"bufio"
	"fmt"
	"os"
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
)

var reviewCmd = &cobra.Command{
	Use:   "review [beads-id]",
	Short: "Review agent work before completing",
	Long: `Review agent work before completing.

Without arguments: Shows all pending completions grouped by project (batch mode).
With beads-id: Shows detailed review for a single agent.

Single-agent review shows:
  - SYNTHESIS.md summary (TLDR, outcome, recommendation)
  - Recent commits with stats
  - Beads comments history
  - Artifacts produced (investigations, design docs)

Examples:
  orch-go review                    # Batch mode: show all pending completions
  orch-go review orch-go-3anf       # Single agent: detailed review
  orch-go review -p orch-cli        # Batch mode: filter by project
  orch-go review --needs-review     # Batch mode: show failures only`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Single-agent mode if beads ID provided
		if len(args) > 0 {
			return runReviewSingle(args[0])
		}
		// Batch mode
		return runReview(reviewProject, reviewNeedsReview)
	},
}

var reviewDoneCmd = &cobra.Command{
	Use:   "done [project]",
	Short: "Complete all agents for a project",
	Long: `Complete all agents for a project by closing their beads issues.

This runs the completion workflow for each agent with Phase: Complete status,
closing the beads issue and cleaning up resources.

Agents that fail verification (no Phase: Complete) will be skipped.

Examples:
  orch-go review done orch-cli     # Complete all orch-cli agents (with confirmation)
  orch-go review done orch-cli -y  # Skip confirmation`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReviewDone(args[0])
	},
}

func init() {
	reviewCmd.Flags().StringVarP(&reviewProject, "project", "p", "", "Filter by project")
	reviewCmd.Flags().BoolVar(&reviewNeedsReview, "needs-review", false, "Show failures only")
	reviewDoneCmd.Flags().BoolVarP(&reviewDoneYes, "yes", "y", false, "Skip confirmation prompt")
	reviewCmd.AddCommand(reviewDoneCmd)
}

// CompletionInfo holds information about a completed agent for review.
type CompletionInfo struct {
	WorkspaceID string // Workspace directory name
	BeadsID     string // Beads issue ID
	Project     string
	VerifyOK    bool
	VerifyError string
	Phase       string
	Summary     string
	Skill       string
	Synthesis   *verify.Synthesis
}

// getCompletionsForReview retrieves completed agents with verification status.
// Scans .orch/workspace/ for completed workspaces (those with SYNTHESIS.md).
func getCompletionsForReview() ([]CompletionInfo, error) {
	projectDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	var results []CompletionInfo

	// Scan workspaces for SYNTHESIS.md (completion indicator)
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	entries, _ := os.ReadDir(workspaceDir)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		dirPath := filepath.Join(workspaceDir, dirName)

		// Check for SYNTHESIS.md
		synthesisPath := filepath.Join(dirPath, "SYNTHESIS.md")
		if _, err := os.Stat(synthesisPath); os.IsNotExist(err) {
			continue // No synthesis = not complete
		}

		// Extract beads ID from SPAWN_CONTEXT.md
		beadsID := extractBeadsIDFromWorkspace(dirPath)

		// Extract skill from workspace name
		skill := extractSkillFromTitle(dirName)

		info := CompletionInfo{
			WorkspaceID: dirName,
			BeadsID:     beadsID,
			Project:     extractProject(projectDir),
			Skill:       skill,
		}

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

		results = append(results, info)
	}

	return results, nil
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

	// Display the review
	fmt.Print(verify.FormatAgentReview(review))

	// Print next steps
	fmt.Println("---")
	if review.Status == "Phase: Complete" && review.SynthesisExists {
		fmt.Printf("Ready to complete: orch complete %s\n", beadsID)
	} else {
		if !review.SynthesisExists {
			fmt.Println("Missing: SYNTHESIS.md - agent should create this before completing")
		}
		if review.Status != "Phase: Complete" {
			fmt.Println("Missing: Phase: Complete - agent should report via bd comment")
		}
		fmt.Printf("\nTo force completion: orch complete %s --force\n", beadsID)
	}

	return nil
}

func runReview(projectFilter string, needsReviewOnly bool) error {
	completions, err := getCompletionsForReview()
	if err != nil {
		return err
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

			beadsInfo := ""
			if c.BeadsID != "" {
				beadsInfo = fmt.Sprintf(" (%s)", c.BeadsID)
			}

			fmt.Printf("  [%s] %s%s\n", status, c.WorkspaceID, beadsInfo)

			if c.VerifyOK && c.Summary != "" {
				fmt.Printf("         Phase: %s - %s\n", c.Phase, c.Summary)
			}

			// Display Synthesis Card if available
			if c.Synthesis != nil {
				printSynthesisCard(c.Synthesis)
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

	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	logger := events.NewLogger(events.DefaultLogPath())

	for _, c := range canComplete {
		fmt.Printf("\nCompleting: %s (%s)\n", c.WorkspaceID, c.BeadsID)

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
