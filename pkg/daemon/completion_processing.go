// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/attention"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// CompletionConfig holds configuration for the completion processing loop.
type CompletionConfig struct {
	// PollInterval is the time between polling cycles.
	PollInterval time.Duration

	// DryRun shows what would be processed without actually closing issues.
	DryRun bool

	// Verbose enables detailed output.
	Verbose bool

	// WorkspaceDir is the base directory for agent workspaces.
	// Defaults to .orch/workspace/ relative to project root.
	WorkspaceDir string

	// ProjectDir is the project root directory.
	// Used to locate workspaces and verify constraints.
	ProjectDir string

	// ServerURL is the OpenCode server URL.
	// Used for mode-aware backend verification.
	ServerURL string
}

// DefaultCompletionConfig returns sensible defaults for completion configuration.
func DefaultCompletionConfig() CompletionConfig {
	return CompletionConfig{
		PollInterval: 60 * time.Second,
		DryRun:       false,
		Verbose:      false,
	}
}

// CompletedAgent represents an agent that has reported Phase: Complete
// but whose beads issue is still open/in_progress.
type CompletedAgent struct {
	BeadsID       string
	Title         string
	Status        string // open or in_progress
	PhaseSummary  string // Summary from "Phase: Complete - <summary>"
	WorkspacePath string // Path to agent workspace (if found)
}

// CompletionResult contains the result of processing a completion.
type CompletionResult struct {
	BeadsID      string
	Processed    bool
	CloseReason  string
	Error        error
	Verification verify.VerificationResult
	Escalation   verify.EscalationLevel // Escalation level for this completion
}

// CompletionLoopResult contains the results of a completion loop iteration.
type CompletionLoopResult struct {
	Processed []CompletionResult
	Errors    []error
}

// ListCompletedAgents finds all agents that have reported Phase: Complete
// but whose beads issues are still open or in_progress.
func (d *Daemon) ListCompletedAgents(config CompletionConfig) ([]CompletedAgent, error) {
	if d.listCompletedAgentsFunc != nil {
		return d.listCompletedAgentsFunc(config)
	}
	return ListCompletedAgentsDefault(config)
}

// ListCompletedAgentsDefault is the default implementation that queries beads.
func ListCompletedAgentsDefault(config CompletionConfig) ([]CompletedAgent, error) {
	// Get all open/in_progress issues
	openIssues, err := verify.ListOpenIssues()
	if err != nil {
		return nil, fmt.Errorf("failed to list open issues: %w", err)
	}

	if len(openIssues) == 0 {
		return nil, nil
	}

	// Collect beads IDs for batch comment fetch
	var beadsIDs []string
	for id := range openIssues {
		beadsIDs = append(beadsIDs, id)
	}

	// Fetch comments for all issues in batch
	commentMap := verify.GetCommentsBatch(beadsIDs)

	var completed []CompletedAgent

	for id, issue := range openIssues {
		comments, ok := commentMap[id]
		if !ok {
			continue
		}

		// Parse phase from comments
		phaseStatus := verify.ParsePhaseFromComments(comments)
		if !phaseStatus.Found {
			continue
		}

		// Check if Phase: Complete
		if !strings.EqualFold(phaseStatus.Phase, "Complete") {
			continue
		}

		// Found a completed agent - look for its workspace
		workspacePath := findWorkspaceForIssue(id, config.WorkspaceDir, config.ProjectDir)

		completed = append(completed, CompletedAgent{
			BeadsID:       id,
			Title:         issue.Title,
			Status:        issue.Status,
			PhaseSummary:  phaseStatus.Summary,
			WorkspacePath: workspacePath,
		})
	}

	return completed, nil
}

// findWorkspaceForIssue tries to find the workspace directory for a beads issue.
// It scans .orch/workspace/ for directories that might match the issue.
func findWorkspaceForIssue(beadsID, workspaceDir, projectDir string) string {
	if workspaceDir == "" && projectDir != "" {
		workspaceDir = filepath.Join(projectDir, ".orch", "workspace")
	}
	if workspaceDir == "" {
		// Try current directory
		cwd, _ := os.Getwd()
		workspaceDir = filepath.Join(cwd, ".orch", "workspace")
	}

	// Check if workspace directory exists
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		return ""
	}

	// Scan workspace directories for SPAWN_CONTEXT.md that references this beads ID
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		wsPath := filepath.Join(workspaceDir, entry.Name())
		spawnContext := filepath.Join(wsPath, "SPAWN_CONTEXT.md")

		// Check if SPAWN_CONTEXT.md exists and references this beads ID
		data, err := os.ReadFile(spawnContext)
		if err != nil {
			continue
		}

		// Look for beads ID in spawn context (e.g., "bd comment <id>" or "--issue <id>")
		if strings.Contains(string(data), beadsID) {
			return wsPath
		}
	}

	return ""
}

// ProcessCompletion verifies and closes a single completed agent.
// It runs the same verification as `orch complete` and closes the beads issue.
// Uses the escalation model to determine whether to auto-complete:
//   - EscalationNone/Info/Review: Auto-complete (issue closed)
//   - EscalationBlock/Failed: Do not auto-complete (issue remains open)
func (d *Daemon) ProcessCompletion(agent CompletedAgent, config CompletionConfig) CompletionResult {
	result := CompletionResult{
		BeadsID: agent.BeadsID,
	}

	// Determine tier from workspace if available
	tier := ""
	if agent.WorkspacePath != "" {
		tier = verify.ReadTierFromWorkspace(agent.WorkspacePath)
	}

	// Run full verification
	verificationResult, err := verify.VerifyCompletionFull(
		agent.BeadsID,
		agent.WorkspacePath,
		config.ProjectDir,
		tier,
		config.ServerURL,
	)
	if err != nil {
		result.Error = fmt.Errorf("verification failed: %w", err)
		result.Verification = verificationResult
		result.Escalation = verify.EscalationFailed
		return result
	}

	result.Verification = verificationResult

	// Try to parse synthesis for escalation signals
	var synthesis *verify.Synthesis
	if agent.WorkspacePath != "" {
		synthesis, _ = verify.ParseSynthesis(agent.WorkspacePath)
	}

	// Determine escalation level
	escalation := verify.DetermineEscalationFromCompletion(
		verificationResult,
		synthesis,
		agent.BeadsID,
		agent.WorkspacePath,
		config.ProjectDir,
	)
	result.Escalation = escalation

	// Check if verification passed
	if !verificationResult.Passed {
		result.Error = fmt.Errorf("verification failed: %s", strings.Join(verificationResult.Errors, "; "))
		// Emit verify_failed attention signal for visibility
		emitVerifyFailedSignal(agent, verificationResult.GatesFailed, verificationResult.Errors)

		return result
	}

	// Check if escalation allows auto-completion
	if !escalation.ShouldAutoComplete() {
		reason := verify.ExplainEscalation(verify.EscalationInput{
			VerificationPassed:  verificationResult.Passed,
			VerificationErrors:  verificationResult.Errors,
			NeedsVisualApproval: escalation == verify.EscalationBlock,
		})
		result.Error = fmt.Errorf("requires human review: %s", reason.Reason)
		// Emit verify_failed attention signal (escalation blocked auto-completion)
		emitVerifyFailedSignal(agent, []string{"escalation_blocked"}, []string{result.Error.Error()})

		return result
	}

	// Build close reason from phase summary
	closeReason := "Phase: Complete"
	if agent.PhaseSummary != "" {
		closeReason = fmt.Sprintf("Phase: Complete - %s", agent.PhaseSummary)
	}

	// Close the issue (unless dry run), using force to bypass bd's redundant
	// Phase: Complete gate since we already verified it via ListCompletedAgents
	if !config.DryRun {
		if err := verify.CloseIssueForce(agent.BeadsID, closeReason, true); err != nil {
			result.Error = fmt.Errorf("failed to close issue: %w", err)
			return result
		}
	}

	result.Processed = true
	result.CloseReason = closeReason
	return result
}

// CompletionOnce runs a single iteration of the completion loop.
// It finds all Phase: Complete agents and processes their completions.
func (d *Daemon) CompletionOnce(config CompletionConfig) (*CompletionLoopResult, error) {
	result := &CompletionLoopResult{}

	// Find completed agents
	completed, err := d.ListCompletedAgents(config)
	if err != nil {
		return nil, fmt.Errorf("failed to list completed agents: %w", err)
	}

	if len(completed) == 0 {
		return result, nil
	}

	// Process each completed agent
	logger := events.NewDefaultLogger()

	for _, agent := range completed {
		if config.Verbose {
			fmt.Printf("  Processing completion for %s: %s\n", agent.BeadsID, agent.Title)
		}

		compResult := d.ProcessCompletion(agent, config)
		result.Processed = append(result.Processed, compResult)

		if compResult.Error != nil {
			result.Errors = append(result.Errors, compResult.Error)
			if config.Verbose {
				fmt.Printf("    Error: %v (escalation=%s)\n", compResult.Error, compResult.Escalation)
			}
		} else if compResult.Processed {
			// Log successful auto-completion with escalation level
			if err := logger.LogAutoCompletedWithEscalation(agent.BeadsID, compResult.CloseReason, compResult.Escalation.String()); err != nil && config.Verbose {
				fmt.Printf("    Warning: failed to log completion event: %v\n", err)
			}
			if config.Verbose {
				fmt.Printf("    Closed: %s (escalation=%s)\n", compResult.CloseReason, compResult.Escalation)
			}
		}
	}

	return result, nil
}

// CompletionLoop runs the completion processing loop continuously.
// It polls for Phase: Complete agents and closes their issues.
// The loop continues until the context is cancelled.
func (d *Daemon) CompletionLoop(ctx context.Context, config CompletionConfig) error {
	ticker := time.NewTicker(config.PollInterval)
	defer ticker.Stop()

	// Run immediately on first call
	if _, err := d.CompletionOnce(config); err != nil && config.Verbose {
		fmt.Printf("Completion loop error: %v\n", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := d.CompletionOnce(config); err != nil && config.Verbose {
				fmt.Printf("Completion loop error: %v\n", err)
			}
		}
	}
}

// PreviewCompletions shows what agents would be completed without actually closing them.
func (d *Daemon) PreviewCompletions(config CompletionConfig) ([]CompletedAgent, error) {
	return d.ListCompletedAgents(config)
}

// emitVerifyFailedSignal stores a verification failure signal for attention system visibility.
// This enables the Work Graph to show issues stuck in "verification purgatory".
func emitVerifyFailedSignal(agent CompletedAgent, failedGates, errors []string) {
	entry := attention.VerifyFailedEntry{
		BeadsID:      agent.BeadsID,
		Title:        agent.Title,
		FailedGates:  failedGates,
		Errors:       errors,
		PhaseSummary: agent.PhaseSummary,
	}

	// Store the failure - errors are logged but don't block completion processing
	if err := attention.StoreVerifyFailed(entry); err != nil {
		// Log but don't fail - this is observability, not critical path
		fmt.Printf("Warning: failed to store verify_failed signal for %s: %v\n", agent.BeadsID, err)
	}
}
