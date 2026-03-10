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

	// ProjectDirs lists all project directories to scan for completions.
	// When non-empty, the completion scanner queries each project's beads
	// database and workspace directory, enabling cross-project completion
	// detection. When empty, only the local project (ProjectDir/cwd) is scanned.
	ProjectDirs []string
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
	Status        string   // open or in_progress
	PhaseSummary  string   // Summary from "Phase: Complete - <summary>"
	WorkspacePath string   // Path to agent workspace (if found)
	ProjectDir    string   // Source project directory (for cross-project operations)
	Labels        []string // Issue labels (for effort-based completion routing)
}

// CompletionResult contains the result of processing a completion.
type CompletionResult struct {
	BeadsID       string
	Processed     bool
	AutoCompleted bool // True when daemon ran orch complete (auto-tier)
	CloseReason   string
	Error         error
	Verification  verify.VerificationResult
	Escalation    verify.EscalationLevel // Escalation level for this completion
}

// CompletionLoopResult contains the results of a completion loop iteration.
type CompletionLoopResult struct {
	Processed []CompletionResult
	Errors    []error
}

// ListCompletedAgents finds all agents that have reported Phase: Complete
// but whose beads issues are still open or in_progress.
// When ProjectRegistry is available, lazily wires it into the default
// completion finder for cross-project scanning.
func (d *Daemon) ListCompletedAgents(config CompletionConfig) ([]CompletedAgent, error) {
	if d.Completions != nil {
		// Lazily wire ProjectRegistry into default completion finder
		if dcf, ok := d.Completions.(*defaultCompletionFinder); ok {
			dcf.registry = d.ProjectRegistry
		}
		return d.Completions.ListCompletedAgents(config)
	}
	return ListCompletedAgentsDefault(config)
}

// ListCompletedAgentsDefault is the default implementation that queries beads.
// When config.ProjectDirs is non-empty, scans all listed projects for completions.
// Otherwise, scans only the local project (config.ProjectDir or cwd).
func ListCompletedAgentsDefault(config CompletionConfig) ([]CompletedAgent, error) {
	if len(config.ProjectDirs) > 0 {
		return listCompletedAgentsMultiProject(config)
	}
	return listCompletedAgentsSingleProject(config, "", "")
}

// listCompletedAgentsSingleProject scans a single project's beads database
// and workspace directory for completed agents.
// If projectDir is empty, uses config.ProjectDir or cwd.
// If workspaceDir is empty, uses config.WorkspaceDir or derives from projectDir.
func listCompletedAgentsSingleProject(config CompletionConfig, projectDir, workspaceDir string) ([]CompletedAgent, error) {
	// Get all open/in_progress issues from this project's beads database
	var openIssues map[string]*verify.Issue
	var err error
	if projectDir != "" {
		openIssues, err = verify.ListOpenIssuesWithDir(projectDir)
	} else {
		openIssues, err = verify.ListOpenIssues("")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list open issues: %w", err)
	}

	if len(openIssues) == 0 {
		return nil, nil
	}

	// Filter out issues already labeled daemon:ready-review or daemon:verification-failed.
	// - daemon:ready-review: already processed by a previous completion cycle
	// - daemon:verification-failed: exhausted verification retry budget, deferred for human review
	// Without this filter, resume→re-detect→re-pause loops occur because
	// the same cross-project Phase: Complete issues get re-counted after
	// resume clears the VerificationTracker's seenIDs.
	for id, issue := range openIssues {
		for _, label := range issue.Labels {
			if label == LabelReadyReview || label == LabelVerificationFailed {
				delete(openIssues, id)
				break
			}
		}
	}

	if len(openIssues) == 0 {
		return nil, nil
	}

	// Collect beads IDs for batch comment fetch
	var beadsIDs []string
	for id := range openIssues {
		beadsIDs = append(beadsIDs, id)
	}

	// Fetch comments — use project-dir-aware variant for cross-project
	var commentMap map[string][]verify.Comment
	if projectDir != "" {
		projectDirs := make(map[string]string, len(beadsIDs))
		for _, id := range beadsIDs {
			projectDirs[id] = projectDir
		}
		commentMap = verify.GetCommentsBatchWithProjectDirs(beadsIDs, projectDirs)
	} else {
		commentMap = verify.GetCommentsBatch(beadsIDs, nil)
	}

	// Resolve workspace dir for finding agent workspaces
	effectiveWorkspaceDir := workspaceDir
	if effectiveWorkspaceDir == "" {
		effectiveWorkspaceDir = config.WorkspaceDir
	}
	effectiveProjectDir := projectDir
	if effectiveProjectDir == "" {
		effectiveProjectDir = config.ProjectDir
	}

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
		workspacePath := findWorkspaceForIssue(id, effectiveWorkspaceDir, effectiveProjectDir)

		completed = append(completed, CompletedAgent{
			BeadsID:       id,
			Title:         issue.Title,
			Status:        issue.Status,
			PhaseSummary:  phaseStatus.Summary,
			WorkspacePath: workspacePath,
			ProjectDir:    projectDir,
			Labels:        issue.Labels,
		})
	}

	return completed, nil
}

// listCompletedAgentsMultiProject scans all project directories in config.ProjectDirs
// for completed agents. Each project's beads database and workspace directory are
// queried independently, and results are merged with deduplication by beads ID.
func listCompletedAgentsMultiProject(config CompletionConfig) ([]CompletedAgent, error) {
	seen := make(map[string]bool)
	var allCompleted []CompletedAgent

	for _, projectDir := range config.ProjectDirs {
		wsDir := filepath.Join(projectDir, ".orch", "workspace")
		completed, err := listCompletedAgentsSingleProject(config, projectDir, wsDir)
		if err != nil {
			// Log but continue — one project's beads failure shouldn't block others
			if config.Verbose {
				fmt.Printf("  Warning: failed to scan completions in %s: %v\n", projectDir, err)
			}
			continue
		}

		for _, agent := range completed {
			if !seen[agent.BeadsID] {
				seen[agent.BeadsID] = true
				allCompleted = append(allCompleted, agent)
			}
		}
	}

	return allCompleted, nil
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

	// Use agent's project dir for cross-project operations, fall back to config
	effectiveProjectDir := agent.ProjectDir
	if effectiveProjectDir == "" {
		effectiveProjectDir = config.ProjectDir
	}

	// Determine tier from workspace if available
	tier := ""
	if agent.WorkspacePath != "" {
		tier = verify.ReadTierFromWorkspace(agent.WorkspacePath)
	}

	// Pre-fetch comments using the correct project directory.
	// This avoids VerifyCompletionFullWithComments re-fetching from the wrong dir
	// for cross-project agents (the daemon's cwd != the agent's project).
	comments, err := verify.GetComments(agent.BeadsID, effectiveProjectDir)
	if err != nil {
		result.Error = fmt.Errorf("failed to fetch comments for %s (dir=%s): %w", agent.BeadsID, effectiveProjectDir, err)
		result.Escalation = verify.EscalationFailed
		return result
	}

	// Run full verification with pre-fetched comments
	verificationResult, err := verify.VerifyCompletionFullWithComments(
		agent.BeadsID,
		agent.WorkspacePath,
		effectiveProjectDir,
		tier,
		config.ServerURL,
		comments,
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
		effectiveProjectDir,
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

	// Determine review tier from workspace manifest
	reviewTier := ""
	if agent.WorkspacePath != "" {
		reviewTier = verify.ReadReviewTierFromWorkspace(agent.WorkspacePath)
	}

	// Effort-based auto-complete: effort:small issues skip explain-back and verified gates.
	// Uses LightAutoCompleter (targeted skips) rather than full --force.
	// effort:medium and effort:large follow the normal path (tag ready-review).
	if IsEffortSmall(agent.Labels) && d.AutoCompleter != nil && !config.DryRun {
		var completeErr error
		if lightCompleter, ok := d.AutoCompleter.(LightAutoCompleter); ok {
			completeErr = lightCompleter.CompleteLight(agent.BeadsID, effectiveProjectDir)
		} else {
			// Fallback to full auto-complete if LightAutoCompleter not available
			completeErr = d.AutoCompleter.Complete(agent.BeadsID, effectiveProjectDir)
		}

		if completeErr != nil {
			// Light auto-complete failed — fall back to orchestrator review
			result.Error = fmt.Errorf("light auto-complete failed for effort:small agent: %w", completeErr)
			return result
		}

		result.Processed = true
		result.AutoCompleted = true
		result.CloseReason = completionSummary

		// Record auto-completion for verification tracking
		if d.VerificationTracker != nil {
			shouldPause := d.VerificationTracker.RecordCompletion(agent.BeadsID)
			if shouldPause && config.Verbose {
				status := d.VerificationTracker.Status()
				fmt.Printf("    Verification pause triggered: %d/%d auto-completions. Resume with: orch daemon resume\n",
					status.CompletionsSinceVerification, status.Threshold)
			}
		}

		return result
	}

	// Auto-tier: run orch complete directly instead of labeling for review.
	// This closes the issue, archives workspace, and frees the slot automatically.
	// If orch complete fails (gate failure, escalation), fall back to orchestrator review.
	if reviewTier == "auto" && d.AutoCompleter != nil && !config.DryRun {
		if err := d.AutoCompleter.Complete(agent.BeadsID, effectiveProjectDir); err != nil {
			// Auto-complete failed — leave for orchestrator review
			result.Error = fmt.Errorf("auto-complete failed for auto-tier agent: %w", err)
			return result
		}

		result.Processed = true
		result.AutoCompleted = true
		result.CloseReason = completionSummary

		// Record auto-completion for verification tracking
		if d.VerificationTracker != nil {
			shouldPause := d.VerificationTracker.RecordCompletion(agent.BeadsID)
			if shouldPause && config.Verbose {
				status := d.VerificationTracker.Status()
				fmt.Printf("    Verification pause triggered: %d/%d auto-completions. Resume with: orch daemon resume\n",
					status.CompletionsSinceVerification, status.Threshold)
			}
		}

		return result
	}

	// Mark issue as ready for review (unless dry run)
	// Non-auto tiers: add a label so Dylan can review via orchestrator
	if !config.DryRun {
		if err := verify.AddLabel(agent.BeadsID, LabelReadyReview, effectiveProjectDir); err != nil {
			result.Error = fmt.Errorf("failed to mark ready for review: %w", err)
			return result
		}

		// Record auto-completion for verification tracking.
		// Only increments if this beads ID hasn't been counted yet (dedup across poll cycles).
		if d.VerificationTracker != nil {
			shouldPause := d.VerificationTracker.RecordCompletion(agent.BeadsID)
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
//
// Verification retry budget: Each agent gets a limited number of verification
// attempts (3 for local, 1 for cross-project). After exhausting the budget,
// the agent is labeled daemon:verification-failed and excluded from future scans.
// This prevents the infinite retry loop on verification failures.
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
		isCrossProject := agent.ProjectDir != ""

		// Check if this agent has exhausted its verification retry budget.
		// Skip it entirely — the label filter in ListCompletedAgents should
		// have caught it, but this is defense-in-depth for the race window
		// between labeling and the next scan.
		if d.VerificationRetryTracker != nil && d.VerificationRetryTracker.IsExhausted(agent.BeadsID, isCrossProject) {
			if config.Verbose {
				fmt.Printf("  Skipping %s (verification retry budget exhausted)\n", agent.BeadsID)
			}
			continue
		}

		if config.Verbose {
			fmt.Printf("  Processing completion for %s: %s\n", agent.BeadsID, agent.Title)
		}

		compResult := d.ProcessCompletion(agent, config)
		result.Processed = append(result.Processed, compResult)

		if compResult.Error != nil {
			result.Errors = append(result.Errors, compResult.Error)

			// Track verification failure and check retry budget
			d.handleVerificationFailure(agent, compResult, config)
		} else if compResult.Processed {
			// Successful completion — clear any prior retry tracking
			if d.VerificationRetryTracker != nil {
				d.VerificationRetryTracker.Clear(agent.BeadsID)
			}

			// Log completion event — differentiate auto-completed (closed by daemon)
			// from labeled (waiting for orchestrator review)
			if compResult.AutoCompleted {
				logReason := "auto-completed"
				tierLabel := "tier=auto"
				if IsEffortSmall(agent.Labels) {
					logReason = "light-auto-completed"
					tierLabel = "effort=small"
				}
				if err := logger.LogAutoCompletedWithEscalation(agent.BeadsID, compResult.CloseReason, logReason); err != nil && config.Verbose {
					fmt.Printf("    Warning: failed to log auto-completion event: %v\n", err)
				}
				if config.Verbose {
					fmt.Printf("    Auto-completed: %s (%s)\n", compResult.CloseReason, tierLabel)
				}
			} else {
				if err := logger.LogAutoCompletedWithEscalation(agent.BeadsID, compResult.CloseReason, compResult.Escalation.String()); err != nil && config.Verbose {
					fmt.Printf("    Warning: failed to log completion event: %v\n", err)
				}
				if config.Verbose {
					fmt.Printf("    Labeled ready-review: %s (escalation=%s)\n", compResult.CloseReason, compResult.Escalation)
				}
			}
		}
	}

	return result, nil
}

// handleVerificationFailure tracks a verification failure and applies the
// daemon:verification-failed label when the retry budget is exhausted.
func (d *Daemon) handleVerificationFailure(agent CompletedAgent, compResult CompletionResult, config CompletionConfig) {
	if d.VerificationRetryTracker == nil {
		if config.Verbose {
			fmt.Printf("    Error: %v (escalation=%s)\n", compResult.Error, compResult.Escalation)
		}
		return
	}

	isCrossProject := agent.ProjectDir != ""
	attempts := d.VerificationRetryTracker.RecordFailure(agent.BeadsID)
	maxAttempts := MaxAttemptsFor(isCrossProject)

	if attempts >= maxAttempts {
		// Budget exhausted — label the issue so it's filtered out of future scans
		effectiveProjectDir := agent.ProjectDir
		if effectiveProjectDir == "" {
			effectiveProjectDir = config.ProjectDir
		}

		if err := verify.AddLabel(agent.BeadsID, LabelVerificationFailed, effectiveProjectDir); err != nil {
			// Log but don't fail — worst case, the in-memory tracker still prevents retries
			fmt.Fprintf(os.Stderr, "    Warning: failed to label %s as %s: %v\n",
				agent.BeadsID, LabelVerificationFailed, err)
		}

		projectType := "local"
		if isCrossProject {
			projectType = "cross-project"
		}
		fmt.Printf("    Verification failed (%s, attempt %d/%d): %s - deferred for human review\n",
			projectType, attempts, maxAttempts, agent.BeadsID)
		fmt.Printf("    Last error: %v\n", compResult.Error)
	} else if config.Verbose {
		fmt.Printf("    Error: %v (escalation=%s, attempt %d/%d)\n",
			compResult.Error, compResult.Escalation, attempts, maxAttempts)
	}
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
