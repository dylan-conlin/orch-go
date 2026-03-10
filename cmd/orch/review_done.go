package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	// Review done command flags
	reviewDoneYes bool
	reviewNoPrompt bool
)

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
	reviewDoneCmd.Flags().BoolVarP(&reviewDoneYes, "yes", "y", false, "Skip confirmation prompt")
	reviewDoneCmd.Flags().BoolVar(&reviewNoPrompt, "no-prompt", false, "Skip recommendation prompts (auto-close without reviewing synthesis)")
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
	var canComplete []CompletionInfo   // Has beads ID + verified OK
	var canArchive []CompletionInfo    // No beads ID but completed (untracked)
	var needsReview []CompletionInfo
	for _, c := range projectCompletions {
		if c.VerifyOK && c.BeadsID != "" {
			canComplete = append(canComplete, c)
		} else if c.VerifyOK && c.BeadsID == "" {
			canArchive = append(canArchive, c)
		} else {
			needsReview = append(needsReview, c)
		}
	}

	// Show summary before proceeding
	fmt.Printf("Project: %s\n", project)
	fmt.Printf("  Ready to complete: %d\n", len(canComplete))
	if len(canArchive) > 0 {
		fmt.Printf("  Untracked (will archive): %d\n", len(canArchive))
	}
	fmt.Printf("  Needs manual review: %d\n", len(needsReview))

	if len(canComplete) == 0 && len(canArchive) == 0 {
		fmt.Println("\nNo agents ready to complete")
		if len(needsReview) > 0 {
			fmt.Println("\nAgents needing manual review:")
			for _, c := range needsReview {
				reason := "verification failed"
				if c.VerifyError != "" {
					reason = c.VerifyError
				}
				fmt.Printf("  - %s: %s\n", c.WorkspaceID, reason)
			}
		}
		return nil
	}

	// Confirmation prompt unless --yes flag is set or stdin is not a terminal
	if !reviewDoneYes {
		actionSummary := fmt.Sprintf("close %d beads issues", len(canComplete))
		if len(canArchive) > 0 {
			actionSummary += fmt.Sprintf(" and archive %d untracked workspaces", len(canArchive))
		}

		// Auto-skip confirmation when stdin is not a terminal (e.g., daemon, scripts)
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Printf("\nThis will %s.\n", actionSummary)
			fmt.Println("(Skipping confirmation - stdin is not a terminal)")
		} else {
			fmt.Printf("\nThis will %s.\n", actionSummary)
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
		issue, err := verify.GetIssue(c.BeadsID, "")
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
			if err := verify.CloseIssue(c.BeadsID, reason, ""); err != nil {
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

	// Archive untracked workspaces (no beads ID, completed with SYNTHESIS.md)
	archived := 0
	for _, c := range canArchive {
		fmt.Printf("\nArchiving untracked: %s\n", c.WorkspaceID)

		if c.WorkspacePath != "" {
			archivedPath, err := archiveWorkspace(c.WorkspacePath, projectDir)
			if err != nil {
				completionErrors = append(completionErrors, fmt.Sprintf("%s: failed to archive: %v", c.WorkspaceID, err))
				continue
			}
			fmt.Printf("  Archived to: %s\n", filepath.Base(archivedPath))
		}

		// Log the archival event
		event := events.Event{
			Type:      "agent.completed",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"workspace":   c.WorkspaceID,
				"reason":      "Archived untracked workspace via review done",
				"batch":       true,
				"source":      "review_done",
				"untracked":   true,
				"project_dir": projectDir,
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Printf("  Warning: failed to log event: %v\n", err)
		}

		archived++
	}

	// Summary
	fmt.Printf("\n---\n")
	if len(canComplete) > 0 {
		fmt.Printf("Completed: %d/%d agents\n", completed, len(canComplete))
	}
	if len(canArchive) > 0 {
		fmt.Printf("Archived: %d/%d untracked workspaces\n", archived, len(canArchive))
	}

	if len(completionErrors) > 0 {
		fmt.Fprintf(os.Stderr, "\nErrors (%d):\n", len(completionErrors))
		for _, e := range completionErrors {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
	}

	if len(needsReview) > 0 {
		fmt.Printf("\nAgents needing manual review (%d):\n", len(needsReview))
		for _, c := range needsReview {
			reason := "verification failed"
			if c.VerifyError != "" {
				reason = c.VerifyError
			}
			fmt.Printf("  - %s: %s\n", c.WorkspaceID, reason)
		}
	}

	return nil
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
