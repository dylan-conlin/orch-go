// Package main provides the rework command for handling post-completion failures.
// When a "completed" feature doesn't actually work, this command:
// 1. Adds a POST-COMPLETION-FAILURE comment to the beads issue
// 2. Reopens the issue via bd reopen
// 3. Offers to spawn a new attempt with failure context injected
//
// See: .kb/investigations/2026-02-04-inv-design-retry-rework-pattern-completed.md
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	// Rework command flags
	reworkFailureType string
	reworkDescription string
	reworkWorkdir     string
	reworkNoSpawn     bool // Skip the offer to spawn a new attempt
)

var reworkCmd = &cobra.Command{
	Use:   "rework [beads-id]",
	Short: "Rework a completed-but-broken feature",
	Long: `Rework a feature that was marked complete but doesn't actually work.

This command handles the full rework flow:
  1. Adds a POST-COMPLETION-FAILURE comment with failure details
  2. Reopens the beads issue via bd reopen
  3. Offers to spawn a new agent with failure context injected

The failure type determines which skill is suggested for the new attempt:
  verification:    Agent didn't properly verify → reliability-testing
  implementation:  Code has a bug → systematic-debugging
  spec:            Spec was wrong/incomplete → investigation
  integration:     Works in isolation, fails in context → reliability-testing

Examples:
  orch rework orch-go-1234 --type implementation --description "Button click doesn't close dialog"
  orch rework orch-go-1234 --type verification --description "Agent claimed tests pass but feature broken"
  orch rework orch-go-1234 --type spec --description "API format was wrong" --no-spawn`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runRework(beadsID, reworkFailureType, reworkDescription, reworkWorkdir, reworkNoSpawn)
	},
}

func init() {
	reworkCmd.Flags().StringVar(&reworkFailureType, "type", "", "Failure type: verification, implementation, spec, integration (required)")
	reworkCmd.Flags().StringVar(&reworkDescription, "description", "", "Description of what failed (required)")
	reworkCmd.Flags().StringVar(&reworkWorkdir, "workdir", "", "Target project directory (for cross-project rework)")
	reworkCmd.Flags().BoolVar(&reworkNoSpawn, "no-spawn", false, "Skip the offer to spawn a new attempt")
	_ = reworkCmd.MarkFlagRequired("type")
	_ = reworkCmd.MarkFlagRequired("description")
}

// runRework executes the rework flow for a completed-but-broken feature.
func runRework(beadsID, failureType, description, workdir string, noSpawn bool) error {
	// Validate failure type
	if !isValidFailureType(failureType) {
		return fmt.Errorf("invalid failure type %q: must be one of: verification, implementation, spec, integration", failureType)
	}

	// Determine project directory
	var projectDir string
	var err error
	if workdir != "" {
		projectDir, err = filepath.Abs(workdir)
		if err != nil {
			return fmt.Errorf("failed to resolve workdir path: %w", err)
		}
		if stat, err := os.Stat(projectDir); err != nil {
			return fmt.Errorf("workdir does not exist: %s", projectDir)
		} else if !stat.IsDir() {
			return fmt.Errorf("workdir is not a directory: %s", projectDir)
		}
		beads.DefaultDir = projectDir
	} else {
		projectDir, err = currentProjectDir()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Resolve beads ID (handle short IDs)
	resolvedID, err := resolveShortBeadsID(beadsID)
	if err != nil {
		return fmt.Errorf("failed to resolve beads ID: %w", err)
	}
	beadsID = resolvedID

	// Verify issue exists
	issue, err := verify.GetIssue(beadsID)
	if err != nil {
		return fmt.Errorf("failed to get beads issue: %w", err)
	}

	// Display rework summary
	fmt.Println(buildReworkSummary(beadsID, failureType, description))

	// Step 1: Add POST-COMPLETION-FAILURE comment
	fmt.Printf("\nStep 1: Adding failure comment to %s...\n", beadsID)
	comment := buildPostCompletionFailureComment(failureType, description)
	if err := addFailureComment(beadsID, comment); err != nil {
		return fmt.Errorf("failed to add failure comment: %w", err)
	}
	fmt.Printf("  Added: %s\n", comment)

	// Step 2: Reopen the issue
	fmt.Printf("\nStep 2: Reopening issue %s...\n", beadsID)
	reopenReason := fmt.Sprintf("Post-completion failure: %s - %s", failureType, truncate(description, 80))
	if err := reopenIssue(beadsID, reopenReason); err != nil {
		return fmt.Errorf("failed to reopen issue: %w", err)
	}
	fmt.Printf("  Reopened: %s (was: %s)\n", beadsID, issue.Status)

	// Log the rework event
	logReworkEvent(beadsID, failureType, description, issue.Title)

	// Step 3: Offer to spawn new attempt
	suggestedSkill := spawn.SuggestSkillForFailure(failureType)
	if noSpawn {
		fmt.Printf("\nRework complete. Issue %s is reopened.\n", beadsID)
		fmt.Printf("Suggested skill for retry: %s\n", suggestedSkill)
		fmt.Printf("To spawn manually:\n")
		fmt.Printf("  orch spawn --bypass-triage --issue %s %s %q\n", beadsID, suggestedSkill, issue.Title)
		return nil
	}

	fmt.Printf("\nStep 3: Spawn new attempt?\n")
	fmt.Printf("  Suggested skill: %s\n", suggestedSkill)
	fmt.Printf("  Task: %s\n", issue.Title)
	fmt.Printf("\nSpawn new agent with failure context? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer == "y" || answer == "yes" {
		fmt.Printf("\nSpawning new attempt with %s skill...\n", suggestedSkill)
		if err := spawnReworkAttempt(beadsID, suggestedSkill, issue.Title, projectDir); err != nil {
			return fmt.Errorf("failed to spawn rework attempt: %w", err)
		}
	} else {
		fmt.Printf("\nRework complete. Issue %s is reopened.\n", beadsID)
		fmt.Printf("To spawn manually:\n")
		fmt.Printf("  orch spawn --bypass-triage --issue %s %s %q\n", beadsID, suggestedSkill, issue.Title)
	}

	return nil
}

// buildPostCompletionFailureComment creates the POST-COMPLETION-FAILURE comment text.
// Format: POST-COMPLETION-FAILURE: [type] - [description]
func buildPostCompletionFailureComment(failureType, description string) string {
	return fmt.Sprintf("%s %s - %s", PostCompletionFailurePrefix, failureType, description)
}

// buildBdReopenArgs constructs the arguments for the bd reopen command.
func buildBdReopenArgs(beadsID, reason string) []string {
	args := []string{"reopen", beadsID}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	return args
}

// isValidFailureType checks if the failure type is one of the recognized types.
func isValidFailureType(failureType string) bool {
	switch failureType {
	case spawn.FailureTypeVerification,
		spawn.FailureTypeImplementation,
		spawn.FailureTypeSpec,
		spawn.FailureTypeIntegration:
		return true
	default:
		return false
	}
}

// buildReworkSummary creates a human-readable summary of the rework action.
func buildReworkSummary(beadsID, failureType, description string) string {
	suggestedSkill := spawn.SuggestSkillForFailure(failureType)
	var sb strings.Builder
	sb.WriteString("┌─────────────────────────────────────────────────────────────────┐\n")
	sb.WriteString("│  REWORK: Post-Completion Failure                                │\n")
	sb.WriteString("├─────────────────────────────────────────────────────────────────┤\n")
	sb.WriteString(fmt.Sprintf("│  Issue:       %s\n", beadsID))
	sb.WriteString(fmt.Sprintf("│  Failure:     %s\n", failureType))
	sb.WriteString(fmt.Sprintf("│  Description: %s\n", truncate(description, 50)))
	sb.WriteString(fmt.Sprintf("│  Suggested:   %s\n", suggestedSkill))
	sb.WriteString("└─────────────────────────────────────────────────────────────────┘")
	return sb.String()
}

// addFailureComment adds a POST-COMPLETION-FAILURE comment to the beads issue.
// Uses the beads RPC client with fallback to CLI.
func addFailureComment(beadsID, comment string) error {
	return withBeadsFallback("", func(client *beads.Client) error {
		return client.AddComment(beadsID, "orchestrator", comment)
	}, func() error {
		return beads.FallbackAddComment(beadsID, comment)
	}, beads.WithAutoReconnect(3))
}

// reopenIssue reopens a beads issue via bd reopen.
// This is preferred over UpdateIssueStatus because it emits a Reopened event
// which integrates with attempt tracking in pkg/verify/attempts.go.
func reopenIssue(beadsID, reason string) error {
	return beads.FallbackReopen(beadsID, reason)
}

// spawnReworkAttempt spawns a new agent for the rework attempt.
// Uses orch spawn with the suggested skill and --issue flag to inject failure context.
func spawnReworkAttempt(beadsID, skill, task, projectDir string) error {
	// Build the orch spawn command
	orchPath, err := os.Executable()
	if err != nil {
		orchPath = "orch" // fallback
	}

	args := []string{
		"spawn",
		"--bypass-triage",
		"--issue", beadsID,
		skill,
		task,
	}

	cmd := exec.Command(orchPath, args...)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// logReworkEvent logs a rework event to events.jsonl.
func logReworkEvent(beadsID, failureType, description, issueTitle string) {
	logger := events.NewLogger(events.DefaultLogPath())

	// Log the reopen event
	reopenData := events.IssueReopenedData{
		BeadsID:        beadsID,
		Title:          issueTitle,
		PreviousStatus: "closed",
		Reason:         fmt.Sprintf("Post-completion failure: %s - %s", failureType, description),
	}
	if err := logger.LogIssueReopened(reopenData); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log rework event: %v\n", err)
	}

	// Also log a specific rework event for tracking
	event := events.Event{
		Type:      "issue.reworked",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"beads_id":        beadsID,
			"failure_type":    failureType,
			"description":     description,
			"title":           issueTitle,
			"suggested_skill": spawn.SuggestSkillForFailure(failureType),
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log rework tracking event: %v\n", err)
	}
}
