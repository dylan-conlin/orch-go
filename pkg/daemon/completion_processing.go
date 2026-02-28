// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	if d.Completions != nil {
		return d.Completions.ListCompletedAgents(config)
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

	var candidates []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip archived directory - only scan active workspaces
		if entry.Name() == "archived" {
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
			candidates = append(candidates, wsPath)
		}
	}

	if len(candidates) == 0 {
		return ""
	}
	if len(candidates) == 1 {
		return candidates[0]
	}

	// Multiple candidates: prefer workspace with SYNTHESIS.md, then most recent spawn time
	return pickBestWorkspacePath(candidates)
}

// pickBestWorkspacePath selects the best workspace path from multiple candidates.
// Priority: 1) has SYNTHESIS.md (completed work), 2) most recent .spawn_time.
func pickBestWorkspacePath(paths []string) string {
	best := 0
	bestHasSynthesis := false
	bestSpawnTime := readSpawnTime(paths[0])
	if _, err := os.Stat(filepath.Join(paths[0], "SYNTHESIS.md")); err == nil {
		bestHasSynthesis = true
	}

	for i := 1; i < len(paths); i++ {
		hasSynthesis := false
		if _, err := os.Stat(filepath.Join(paths[i], "SYNTHESIS.md")); err == nil {
			hasSynthesis = true
		}

		if hasSynthesis && !bestHasSynthesis {
			best = i
			bestHasSynthesis = hasSynthesis
			bestSpawnTime = readSpawnTime(paths[i])
			continue
		}
		if !hasSynthesis && bestHasSynthesis {
			continue
		}

		spawnTime := readSpawnTime(paths[i])
		if spawnTime > bestSpawnTime {
			best = i
			bestHasSynthesis = hasSynthesis
			bestSpawnTime = spawnTime
		}
	}

	return paths[best]
}

// readSpawnTime reads the .spawn_time file from a workspace directory.
// Returns the Unix nanosecond timestamp, or 0 if not found.
func readSpawnTime(wsPath string) int64 {
	data, err := os.ReadFile(filepath.Join(wsPath, ".spawn_time"))
	if err != nil {
		return 0
	}
	t, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0
	}
	return t
}

// ProcessCompletion verifies and marks a single completed agent as ready-for-review.
// It runs the same verification as `orch complete` but does NOT close the beads issue.
// Instead, it adds a "daemon:ready-review" label for orchestrator review.
// Uses the escalation model to determine whether to mark ready-for-review:
//   - EscalationNone/Info/Review: Mark ready-for-review (labeled, not closed)
//   - EscalationBlock/Failed: Requires human review (no label, remains in_progress)
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
		return result
	}

	// Build completion summary from phase summary
	completionSummary := "Phase: Complete"
	if agent.PhaseSummary != "" {
		completionSummary = fmt.Sprintf("Phase: Complete - %s", agent.PhaseSummary)
	}

	// Mark issue as ready for review (unless dry run)
	// Instead of auto-closing, add a label so Dylan can review via orchestrator
	if !config.DryRun {
		if err := verify.AddLabel(agent.BeadsID, "daemon:ready-review"); err != nil {
			result.Error = fmt.Errorf("failed to mark ready for review: %w", err)
			return result
		}

		// Record auto-completion for verification tracking.
		// This increments the counter and may trigger pause if threshold reached.
		if d.VerificationTracker != nil {
			shouldPause := d.VerificationTracker.RecordCompletion()
			if shouldPause && config.Verbose {
				status := d.VerificationTracker.Status()
				fmt.Printf("    Verification pause triggered: %d/%d auto-completions. Resume with: orch daemon resume\n",
					status.CompletionsSinceVerification, status.Threshold)
			}
		}
	}

	result.Processed = true
	result.CloseReason = completionSummary // Still used for logging/display
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
