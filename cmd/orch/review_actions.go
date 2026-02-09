package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"golang.org/x/term"
)

// confirmReviewDone prompts the user to confirm batch completion.
// Respects the --yes flag and auto-skips when stdin is not a terminal.
func confirmReviewDone(count int) error {
	if reviewDoneYes {
		return nil
	}

	fmt.Printf("\nThis will close %d beads issues and clean up resources.\n", count)

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Println("(Skipping confirmation - stdin is not a terminal)")
		return nil
	}

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
	return nil
}

// processCompletions iterates over ready completions, handling recommendations,
// closing beads issues, cleaning up tmux windows, and logging events.
// Returns the count of successfully completed agents and any error messages.
func processCompletions(canComplete []CompletionInfo, projectDir string) (int, []string) {
	completed := 0
	var completionErrors []string

	skipAllPrompts := reviewNoPrompt || !term.IsTerminal(int(os.Stdin.Fd()))
	if !reviewNoPrompt && skipAllPrompts {
		fmt.Println("(Skipping recommendation prompts - stdin is not a terminal)")
	}

	logger := events.NewLogger(events.DefaultLogPath())
	reader := bufio.NewReader(os.Stdin)

	for _, c := range canComplete {
		fmt.Printf("\nCompleting: %s (%s)\n", c.WorkspaceID, c.BeadsID)

		skipAllPrompts = processAgentRecommendations(c, reader, skipAllPrompts)

		if err := closeBeadsIssue(c); err != nil {
			completionErrors = append(completionErrors, fmt.Sprintf("%s: %v", c.BeadsID, err))
			continue
		}

		cleanupAgentTmuxWindow(c)
		logCompletionEvent(logger, c, projectDir)
		completed++
	}

	return completed, completionErrors
}

// processAgentRecommendations handles the recommendation prompt workflow for a single agent.
// Returns the (potentially updated) skipAllPrompts flag.
func processAgentRecommendations(c CompletionInfo, reader *bufio.Reader, skipAllPrompts bool) bool {
	if c.Synthesis == nil || len(c.Synthesis.NextActions) == 0 {
		return skipAllPrompts
	}

	totalRecommendations := len(c.Synthesis.NextActions)
	var actedOnIndices []int
	var dismissedIndices []int

	if !skipAllPrompts {
		actedOnIndices, dismissedIndices, skipAllPrompts = promptForRecommendations(c, reader, totalRecommendations)
	} else {
		dismissedIndices = makeIndexRange(totalRecommendations)
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

	return skipAllPrompts
}

// promptForRecommendations displays recommendations and prompts the user for action.
// Returns acted-on indices, dismissed indices, and updated skipAllPrompts flag.
func promptForRecommendations(c CompletionInfo, reader *bufio.Reader, total int) (acted, dismissed []int, skipAll bool) {
	fmt.Printf("\n  Has %d recommendations:\n", total)
	for i, action := range c.Synthesis.NextActions {
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
		return nil, makeIndexRange(total), true
	}

	response = strings.TrimSpace(strings.ToLower(response))
	switch response {
	case "y", "yes":
		for i, action := range c.Synthesis.NextActions {
			title := action
			if len(title) > 80 {
				title = title[:77] + "..."
			}
			fmt.Printf("  Creating issue: %s\n", title)
			if err := createFollowUpIssue(title, c.WorkspaceID); err != nil {
				fmt.Printf("    Warning: failed to create issue: %v\n", err)
			}
			acted = append(acted, i)
		}
	case "skip-all", "s":
		fmt.Println("  Skipping prompts for remaining agents")
		dismissed = makeIndexRange(total)
		skipAll = true
	case "n", "no", "":
		fmt.Println("  Skipping recommendations")
		dismissed = makeIndexRange(total)
	default:
		fmt.Printf("  Unknown response '%s', skipping recommendations\n", response)
		dismissed = makeIndexRange(total)
	}
	return
}

// makeIndexRange returns a slice [0, 1, ..., n-1].
func makeIndexRange(n int) []int {
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	return indices
}

// closeBeadsIssue checks if the issue is already closed and closes it if needed.
func closeBeadsIssue(c CompletionInfo) error {
	issue, err := verify.GetIssue(c.BeadsID)
	if err != nil {
		return fmt.Errorf("failed to get issue: %v", err)
	}
	if issue.Status == "closed" {
		fmt.Printf("  Already closed, skipping beads close\n")
		return nil
	}

	reason := "Completed via orch review done"
	if c.Summary != "" {
		reason = c.Summary
	}

	if err := verify.CloseIssueForce(c.BeadsID, reason, true); err != nil {
		return fmt.Errorf("failed to close: %v", err)
	}
	fmt.Printf("  Closed beads issue\n")
	return nil
}

// cleanupAgentTmuxWindow kills the tmux window associated with a completion's beads ID.
func cleanupAgentTmuxWindow(c CompletionInfo) {
	if window, sessionName, err := tmux.FindWindowByBeadsIDAllSessions(c.BeadsID); err == nil && window != nil {
		if err := tmux.KillWindow(window.Target); err != nil {
			fmt.Printf("  Warning: failed to close tmux window: %v\n", err)
		} else {
			fmt.Printf("  Closed tmux window: %s:%s\n", sessionName, window.Name)
		}
	}
}

// logCompletionEvent records an agent.completed event for the given completion.
func logCompletionEvent(logger *events.Logger, c CompletionInfo, projectDir string) {
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
}

// createFollowUpIssue creates a beads issue for a synthesis recommendation.
// Uses bd create command to create the issue with appropriate labels.
// Automatically suggests an area: label based on the title.
func createFollowUpIssue(title string, sourceWorkspace string) error {
	// Clean up the title - remove leading bullet markers
	title = strings.TrimPrefix(title, "- ")
	title = strings.TrimPrefix(title, "* ")
	title = strings.TrimSpace(title)

	// Create description linking back to source
	description := fmt.Sprintf("Follow-up from synthesis review of %s", sourceWorkspace)

	labels := []string{"triage:review"}

	// Suggest and add area label based on title/description
	// This enables label discipline for follow-up issues.
	// See: .kb/investigations/2026-02-05-inv-design-label-based-issue-grouping.md
	suggestedArea := beads.SuggestAreaLabel(title, description)
	if suggestedArea != "" {
		labels = append(labels, suggestedArea)
		fmt.Printf("Auto-applying area label: %s\n", suggestedArea)
	}

	_, err := beads.FallbackCreate(title, description, "task", 2, labels)
	if err != nil {
		return fmt.Errorf("bd create failed: %w", err)
	}

	return nil
}
