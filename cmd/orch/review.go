// Package main provides the CLI entry point for orch-go.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/registry"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	// Review command flags
	reviewProject     string
	reviewNeedsReview bool
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
	Short: "Mark project completions as reviewed",
	Long: `Mark all completed agents for a project as reviewed (deleted from registry).

Use this after reviewing completions and verifying the work is acceptable.

Examples:
  orch-go review done orch-cli     # Mark orch-cli completions as reviewed`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReviewDone(args[0])
	},
}

func init() {
	reviewCmd.Flags().StringVarP(&reviewProject, "project", "p", "", "Filter by project")
	reviewCmd.Flags().BoolVar(&reviewNeedsReview, "needs-review", false, "Show failures only")
	reviewCmd.AddCommand(reviewDoneCmd)
}

// CompletionInfo holds information about a completed agent for review.
type CompletionInfo struct {
	Agent       *registry.Agent
	Project     string
	VerifyOK    bool
	VerifyError string
	Phase       string
	Summary     string
	Synthesis   *verify.Synthesis
}

// getCompletionsForReview retrieves completed agents with verification status.
func getCompletionsForReview() ([]CompletionInfo, error) {
	reg, err := registry.New("")
	if err != nil {
		return nil, fmt.Errorf("failed to open registry: %w", err)
	}

	completed := reg.ListCompleted()
	var results []CompletionInfo

	for _, agent := range completed {
		info := CompletionInfo{
			Agent:   agent,
			Project: extractProject(agent.ProjectDir),
		}

		workspacePath := filepath.Join(agent.ProjectDir, ".orch", "workspace", agent.ID)

		// Check verification status
		if agent.BeadsID != "" {
			result, err := verify.VerifyCompletion(agent.BeadsID, workspacePath)
			if err != nil {
				info.VerifyError = fmt.Sprintf("verification error: %v", err)
				info.VerifyOK = false
			} else if result.Passed {
				info.VerifyOK = true
				info.Phase = result.Phase.Phase
				info.Summary = result.Phase.Summary

				// Try to parse synthesis
				s, err := verify.ParseSynthesis(workspacePath)
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
			info.VerifyError = "no beads ID"
			info.VerifyOK = false
		}

		results = append(results, info)
	}

	return results, nil
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
	// Find agent in registry
	reg, err := registry.New("")
	if err != nil {
		return fmt.Errorf("failed to open registry: %w", err)
	}

	agent := reg.Find(beadsID)

	// Build workspace path
	var workspacePath string
	var projectDir string

	if agent != nil {
		projectDir = agent.ProjectDir
		workspacePath = filepath.Join(agent.ProjectDir, ".orch", "workspace", agent.ID)
	} else {
		// Try to find from current directory
		cwd, err := os.Getwd()
		if err == nil {
			projectDir = cwd
			// Look for workspaces matching the beads ID
			workspaceDir := filepath.Join(cwd, ".orch", "workspace")
			entries, err := os.ReadDir(workspaceDir)
			if err == nil {
				for _, entry := range entries {
					if entry.IsDir() && strings.Contains(entry.Name(), beadsID) {
						workspacePath = filepath.Join(workspaceDir, entry.Name())
						break
					}
				}
			}
		}
	}

	// Get review data
	review, err := verify.GetAgentReview(beadsID, workspacePath, projectDir)
	if err != nil {
		return fmt.Errorf("failed to get agent review: %w", err)
	}

	// Enrich with registry data if available
	if agent != nil {
		review.Skill = agent.Skill
		if review.WorkspaceName == "" {
			review.WorkspaceName = agent.ID
		}
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
			if c.Agent.BeadsID != "" {
				beadsInfo = fmt.Sprintf(" (%s)", c.Agent.BeadsID)
			}

			fmt.Printf("  [%s] %s%s\n", status, c.Agent.ID, beadsInfo)

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

			if c.Agent.Skill != "" {
				fmt.Printf("         Skill: %s\n", c.Agent.Skill)
			}
		}
	}

	// Print summary
	fmt.Printf("\n---\n")
	fmt.Printf("Total: %d completions (%d OK, %d need review)\n", totalOK+totalFailed, totalOK, totalFailed)

	if totalOK > 0 {
		fmt.Printf("\nTo mark completions as reviewed:\n")
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

	// Open registry for modifications
	reg, err := registry.New("")
	if err != nil {
		return fmt.Errorf("failed to open registry: %w", err)
	}

	// Mark each as deleted (reviewed)
	marked := 0
	for _, c := range projectCompletions {
		if reg.Remove(c.Agent.ID) {
			marked++
			fmt.Printf("Marked as reviewed: %s", c.Agent.ID)
			if c.Agent.BeadsID != "" {
				fmt.Printf(" (%s)", c.Agent.BeadsID)
			}
			fmt.Println()
		}
	}

	// Save registry
	if err := reg.SaveSkipMerge(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	fmt.Printf("\nMarked %d completions as reviewed for %s\n", marked, project)

	// Check if there are failures that weren't addressed
	failedCount := 0
	for _, c := range projectCompletions {
		if !c.VerifyOK {
			failedCount++
		}
	}

	if failedCount > 0 {
		fmt.Fprintf(os.Stderr, "\nWarning: %d completions had verification failures.\n", failedCount)
		fmt.Fprintf(os.Stderr, "Consider reviewing these issues manually.\n")
	}

	return nil
}

// FormatCompletionStatus returns a formatted status string for a completion.
func FormatCompletionStatus(c CompletionInfo) string {
	var parts []string

	if c.VerifyOK {
		parts = append(parts, "OK")
	} else {
		parts = append(parts, "NEEDS_REVIEW")
	}

	parts = append(parts, c.Agent.ID)

	if c.Agent.BeadsID != "" {
		parts = append(parts, fmt.Sprintf("(%s)", c.Agent.BeadsID))
	}

	return strings.Join(parts, " ")
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
