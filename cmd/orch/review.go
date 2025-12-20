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
	Use:   "review",
	Short: "Review pending agent completions",
	Long: `Review pending agent completions grouped by project.

Shows completed agents that need orchestrator review. Use after overnight
daemon runs or when multiple agents have finished work.

Examples:
  orch-go review                    # Show all pending completions
  orch-go review -p orch-cli        # Filter by project
  orch-go review --needs-review     # Show failures only (need attention)`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		// Check verification status
		if agent.BeadsID != "" {
			result, err := verify.VerifyCompletion(agent.BeadsID)
			if err != nil {
				info.VerifyError = fmt.Sprintf("verification error: %v", err)
				info.VerifyOK = false
			} else if result.Passed {
				info.VerifyOK = true
				info.Phase = result.Phase.Phase
				info.Summary = result.Phase.Summary
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
