// Package main provides the pipeline decomposition of the complete command.
// Each phase is a standalone function with typed input/output, enabling isolated testing.
//
// Pipeline phases:
//  1. ResolveTarget:    identifier → CompletionTarget
//  2. VerifyCompletion: target + skipConfig → VerificationOutcome
//  3. CheckLiveness:    target → (prompt or continue)
//  4. ProcessGates:     target → (discovered work, knowledge gaps, etc.)
//  5. IntegrateBranch:  target → (cherry-pick onto base branch)
//  6. CloseIssue:       target + reason → CloseOutcome (includes epic handling)
//  7. Cleanup:          target → CleanupOutcome (session, archive, docker, tmux)
//  8. PostComplete:     target + outcomes → (telemetry, events, cache)
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	statedb "github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// ──────────────────────────────────────────────────────────────
// Pipeline types
// ──────────────────────────────────────────────────────────────

// CompletionTarget is the resolved identity of the agent being completed.
// Produced by resolveTarget(), consumed by all subsequent phases.
type CompletionTarget struct {
	Identifier            string // Original identifier from CLI arg
	BeadsID               string // Resolved beads ID (empty for orchestrators)
	WorkspacePath         string // Path to .orch/workspace/...
	AgentName             string // Workspace directory name
	BeadsProjectDir       string // Directory containing the beads database
	SourceProjectDir      string // Canonical source repo for beads/project identity
	GitWorktreeDir        string // Worktree dir for verification and integration
	GitBranch             string // Agent branch to rebase + merge
	IsOrchestratorSession bool
	IsUntracked           bool
	IsQuestion            bool
	IsClosed              bool
	Issue                 *verify.Issue // nil for untracked/orchestrator
}

func (t *CompletionTarget) sourceDir() string {
	if t == nil {
		return ""
	}
	if strings.TrimSpace(t.SourceProjectDir) != "" {
		return t.SourceProjectDir
	}
	return t.BeadsProjectDir
}

func (t *CompletionTarget) gitDir() string {
	if t == nil {
		return ""
	}
	if strings.TrimSpace(t.GitWorktreeDir) != "" {
		return t.GitWorktreeDir
	}
	if strings.TrimSpace(t.SourceProjectDir) != "" {
		return t.SourceProjectDir
	}
	return t.BeadsProjectDir
}

// artifactsDir returns the directory where agent-created artifacts (like SYNTHESIS.md) should be found.
// Prefers worktree directory (where agents actually work), with workspace as fallback.
// This fixes the workspace/ vs worktrees/ path confusion: spawn artifacts go to workspace/,
// but agents write their artifacts in worktrees/.
func (t *CompletionTarget) artifactsDir() string {
	if t == nil {
		return ""
	}
	// Prefer worktree directory where agents actually work
	if strings.TrimSpace(t.GitWorktreeDir) != "" {
		return t.GitWorktreeDir
	}
	// Fall back to workspace directory for non-worktree spawns
	if strings.TrimSpace(t.WorkspacePath) != "" {
		return t.WorkspacePath
	}
	return t.BeadsProjectDir
}

// VerificationOutcome is the result of the verification phase.
type VerificationOutcome struct {
	Passed      bool
	SkillName   string
	GatesFailed []string
	Warnings    []string
	PhaseInfo   *verify.PhaseStatus // non-nil if phase info was found
}

// CloseOutcome is the result of closing the beads issue.
type CloseOutcome struct {
	Closed       bool
	Reason       string
	ArchivedPath string // Set after archival in cleanup
}

// CleanupOutcome is the result of the cleanup phase.
type CleanupOutcome struct {
	ArchivedPath       string
	SessionDeleted     bool
	ProcessTerminated  bool
	TmuxWindowClosed   bool
	DockerCleaned      bool
	TranscriptExported bool
	GitWorktreeRemoved bool
	GitBranchDeleted   bool
}

// CompletionTelemetry holds pre-collected telemetry data.
// Must be collected BEFORE cleanup because session deletion and workspace archival
// make the underlying data sources unavailable.
type CompletionTelemetry struct {
	DurationSecs int
	TokensIn     int
	TokensOut    int
	Outcome      string
}

// ──────────────────────────────────────────────────────────────
// Phase 1: Resolve Target
// ──────────────────────────────────────────────────────────────

// resolveTarget resolves the CLI identifier into a CompletionTarget.
// It handles orchestrator registry lookup, workspace name lookup, beads ID resolution,
// and cross-project detection.
func resolveTarget(identifier, workdir string) (*CompletionTarget, error) {
	currentDir, err := currentProjectDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	target := &CompletionTarget{Identifier: identifier}

	// Step 1: Check orchestrator session registry FIRST
	registry := session.NewRegistry("")
	if orchSession, err := registry.Get(identifier); err == nil {
		target.IsOrchestratorSession = true
		target.AgentName = orchSession.WorkspaceName
		fmt.Printf("Orchestrator session (from registry): %s\n", target.AgentName)
		target.WorkspacePath = findWorkspaceByName(orchSession.ProjectDir, target.AgentName)
		if target.WorkspacePath == "" {
			fmt.Fprintf(os.Stderr, "Warning: Workspace %s not found in %s\n", target.AgentName, orchSession.ProjectDir)
		}
	}

	// Step 2: Try direct workspace name lookup (if not found in registry)
	if target.WorkspacePath == "" && !target.IsOrchestratorSession {
		directWorkspacePath := findWorkspaceByName(currentDir, identifier)
		if directWorkspacePath != "" {
			target.WorkspacePath = directWorkspacePath
			target.AgentName = identifier
			if isOrchestratorWorkspace(target.WorkspacePath) {
				target.IsOrchestratorSession = true
				fmt.Printf("Orchestrator session: %s\n", target.AgentName)
			} else {
				beadsIDPath := filepath.Join(target.WorkspacePath, ".beads_id")
				if content, err := os.ReadFile(beadsIDPath); err == nil {
					target.BeadsID = strings.TrimSpace(string(content))
				}
			}
		}
	}

	// Step 3: Treat identifier as beads ID (fallback for worker sessions)
	if target.WorkspacePath == "" && !target.IsOrchestratorSession {
		var crossProjectDir string
		if !strings.Contains(identifier, filepath.Base(currentDir)) {
			projectName := extractProjectFromBeadsID(identifier)
			if projectName != "" && projectName != filepath.Base(currentDir) {
				if foundDir := findProjectDirByName(projectName); foundDir != "" {
					crossProjectDir = foundDir
					beads.DefaultDir = crossProjectDir
					fmt.Printf("Auto-detected cross-project from beads ID: %s\n", filepath.Base(crossProjectDir))
				}
			}
		}

		resolvedID, err := resolveShortBeadsID(identifier)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve beads ID: %w", err)
		}
		target.BeadsID = resolvedID

		searchDir := currentDir
		if crossProjectDir != "" {
			searchDir = crossProjectDir
		}
		target.WorkspacePath, target.AgentName = findWorkspaceByBeadsID(searchDir, target.BeadsID)
	}

	// Resolve beads project directory
	projectResult, err := resolveProjectDir(workdir, target.WorkspacePath, currentDir)
	if err != nil {
		return nil, err
	}
	target.BeadsProjectDir = projectResult.ProjectDir

	switch projectResult.Source {
	case "workdir":
		fmt.Printf("Using explicit workdir: %s\n", target.BeadsProjectDir)
	case "workspace":
		fmt.Printf("Auto-detected cross-project: %s\n", filepath.Base(target.BeadsProjectDir))
	}

	projectResult.SetBeadsDefaultDir()
	target.SourceProjectDir = target.BeadsProjectDir
	target.GitWorktreeDir = target.BeadsProjectDir
	enrichGitTarget(target)

	// Determine tracked/untracked status
	target.IsUntracked = target.IsOrchestratorSession ||
		(target.BeadsID != "" && isUntrackedBeadsID(target.BeadsID)) ||
		target.BeadsID == ""

	// Fetch beads issue for tracked agents
	if !target.IsUntracked {
		issue, err := verify.GetIssue(target.BeadsID)
		if err != nil {
			projectName := filepath.Base(target.BeadsProjectDir)
			issuePrefix := strings.Split(target.BeadsID, "-")[0]
			if len(strings.Split(target.BeadsID, "-")) > 1 {
				issuePrefix = strings.Join(strings.Split(target.BeadsID, "-")[:len(strings.Split(target.BeadsID, "-"))-1], "-")
			}
			if issuePrefix != projectName {
				return nil, fmt.Errorf("failed to get beads issue %s: %w\n\nHint: The issue ID suggests it belongs to project '%s', but you're in '%s'.\nTry: orch complete %s --workdir ~/path/to/%s",
					target.BeadsID, err, issuePrefix, projectName, target.BeadsID, issuePrefix)
			}
			return nil, fmt.Errorf("failed to get beads issue: %w", err)
		}
		target.Issue = issue
		target.IsClosed = issue.Status == "closed"
		target.IsQuestion = issue.IssueType == "question"

		if target.IsClosed {
			fmt.Printf("Issue %s is already closed in beads\n", target.BeadsID)
		}
	} else if target.IsOrchestratorSession {
		fmt.Printf("Note: %s is an orchestrator session (no beads tracking)\n", target.AgentName)
	} else {
		fmt.Printf("Note: %s is an untracked agent (no beads issue)\n", identifier)
	}

	return target, nil
}

func enrichGitTarget(target *CompletionTarget) {
	if target == nil {
		return
	}

	if target.WorkspacePath != "" {
		source, worktree, branch := readGitMetadataFromManifest(target.WorkspacePath)
		if source != "" {
			target.SourceProjectDir = source
		}
		if worktree != "" {
			target.GitWorktreeDir = worktree
		}
		if branch != "" {
			target.GitBranch = branch
		}
	}

	if target.SourceProjectDir == "" {
		target.SourceProjectDir = target.BeadsProjectDir
	}
	if target.GitWorktreeDir == "" {
		target.GitWorktreeDir = target.SourceProjectDir
	}
	if target.GitWorktreeDir == "" {
		target.GitWorktreeDir = target.BeadsProjectDir
	}
	if target.GitBranch != "" {
		return
	}

	branch, err := readBranchName(target.gitDir())
	if err != nil {
		return
	}
	target.GitBranch = branch
}

func readGitMetadataFromManifest(workspacePath string) (string, string, string) {
	manifest, err := spawn.ReadAgentManifest(workspacePath)
	if err != nil || manifest == nil {
		return "", "", ""
	}

	source := strings.TrimSpace(manifest.SourceProjectDir)
	if source == "" {
		source = strings.TrimSpace(manifest.ProjectDir)
	}

	worktree := strings.TrimSpace(manifest.GitWorktreeDir)
	if worktree == "" {
		worktree = strings.TrimSpace(manifest.ProjectDir)
	}

	branch := strings.TrimSpace(manifest.GitBranch)
	return source, worktree, branch
}

// ──────────────────────────────────────────────────────────────
// Phase 2: Verify Completion
// ──────────────────────────────────────────────────────────────

// verifyCompletion checks all verification gates for the completion target.
// It handles orchestrator, regular agent, question entity, and force-mode paths,
// with a unified skip-gate-filtering implementation.

// verifyOrchestratorSession runs verification for orchestrator sessions.

// verifyRegularAgent runs verification for regular (non-orchestrator) agents.

// applySkipFiltering is the unified skip-gate-filtering implementation.
// It replaces the duplicated logic that was in both orchestrator and regular agent paths.

// emitVerificationFailedEvent logs a verification.failed event.

// ──────────────────────────────────────────────────────────────
// Phase 3: Check Liveness
// ──────────────────────────────────────────────────────────────

// checkLiveness warns if the agent appears still running and prompts for confirmation.
// Returns an error if the user declines to proceed.

// ──────────────────────────────────────────────────────────────
// Phase 4: Process Gates (discovered work, knowledge gaps)
// ──────────────────────────────────────────────────────────────

// processGates handles discovered work disposition and knowledge gap detection.
// These are interactive/informational gates that run after verification passes.

// processDiscoveredWork handles the discovered work disposition gate.

// processKnowledgeGaps detects and logs knowledge gaps (informational, non-blocking).

// ──────────────────────────────────────────────────────────────
// Phase 5: Close Issue
// ──────────────────────────────────────────────────────────────

// closeIssue closes the beads issue and handles epic protection/auto-close.
// Returns the resolved close reason.
func closeIssue(target *CompletionTarget, skipConfig SkipConfig) (string, error) {
	reason := resolveCloseReason(target)

	if target.IsClosed || target.IsUntracked || target.BeadsID == "" {
		// Handle orchestrator session completion
		if target.IsOrchestratorSession {
			fmt.Printf("Completed orchestrator session: %s\n", target.AgentName)
			updateOrchestratorSessionStatus(target.AgentName, "completed")
		} else if target.IsUntracked {
			fmt.Printf("Cleaned up untracked agent: %s\n", target.Identifier)
		}
		fmt.Printf("Reason: %s\n", reason)
		return reason, nil
	}

	// Epic protection: check for open children
	if target.Issue != nil && target.Issue.IssueType == "epic" {
		if err := handleEpicProtection(target, reason); err != nil {
			return reason, err
		}
	}

	// Close the issue
	if err := verify.CloseIssueForce(target.BeadsID, reason, skipConfig.PhaseComplete); err != nil {
		return reason, fmt.Errorf("failed to close issue: %w", err)
	}

	// Verify close actually succeeded by reading back the issue status.
	// This catches silent failures where bd close returns success but the issue
	// remains open (e.g., due to beads daemon bugs or missing Phase: Complete proof).
	verifiedIssue, verifyErr := verify.GetIssue(target.BeadsID)
	if verifyErr != nil {
		// Can't verify - warn but continue (issue might be closed, just unreadable)
		fmt.Fprintf(os.Stderr, "Warning: Could not verify issue closure: %v\n", verifyErr)
	} else if strings.ToLower(verifiedIssue.Status) != "closed" {
		// Close command succeeded but issue is NOT closed - this is a silent failure.
		// This can happen when:
		// 1. Beads daemon returns success incorrectly
		// 2. bd close CLI exits 0 but didn't actually close
		// 3. Phase: Complete comment is missing and bd close rejected (but no error returned)
		return reason, fmt.Errorf(
			"bd close reported success but issue %s is still '%s' (expected: closed)\n\n"+
				"This typically happens when:\n"+
				"  - 'Phase: Complete' comment is missing from beads (check: bd comments %s)\n"+
				"  - The agent wrote to state.db but bd comment failed\n\n"+
				"To fix:\n"+
				"  - If Phase: Complete is in state.db: orch complete %s --skip-phase-complete --skip-reason \"Phase in state.db\"\n"+
				"  - Or force close: bd close %s --force",
			target.BeadsID, verifiedIssue.Status, target.BeadsID, target.BeadsID, target.BeadsID)
	}

	fmt.Printf("Closed beads issue: %s\n", target.BeadsID)

	// Epic auto-close: check parent
	if !target.IsOrchestratorSession {
		handleParentEpicAutoClose(target)
	}

	// Remove triage:ready label
	_ = verify.RemoveTriageReadyLabel(target.BeadsID)

	fmt.Printf("Reason: %s\n", reason)
	return reason, nil
}

// resolveCloseReason determines the reason string for closing the issue.
func resolveCloseReason(target *CompletionTarget) string {
	if completeReason != "" {
		return completeReason
	}

	if !target.IsUntracked && target.BeadsID != "" {
		status, _ := verify.GetPhaseStatus(target.BeadsID)
		if status.Summary != "" {
			return status.Summary
		}
	}

	if target.IsOrchestratorSession {
		return "Orchestrator session completed"
	}
	return "Completed via orch complete"
}

// handleEpicProtection checks for open children before closing an epic.
func handleEpicProtection(target *CompletionTarget, reason string) error {
	if !completeForceCloseEpic {
		openChildren, err := verify.GetOpenEpicChildren(target.BeadsID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to check epic children: %v\n", err)
			return nil // Don't block on failure to check
		}
		if len(openChildren) > 0 {
			fmt.Fprintf(os.Stderr, "Cannot complete epic %s - has %d open children:\n", target.BeadsID, len(openChildren))
			showCount := len(openChildren)
			if showCount > 5 {
				showCount = 5
			}
			for i := 0; i < showCount; i++ {
				child := openChildren[i]
				fmt.Fprintf(os.Stderr, "  - %s (%s): %s\n", child.ID, child.Status, child.Title)
			}
			if len(openChildren) > 5 {
				fmt.Fprintf(os.Stderr, "  ... and %d more\n", len(openChildren)-5)
			}
			fmt.Fprintf(os.Stderr, "\nUse --force-close-epic to close anyway\n")
			return fmt.Errorf("epic has open children")
		}
		return nil
	}

	// Force-closing epic: log orphaned children
	openChildren, err := verify.GetOpenEpicChildren(target.BeadsID)
	if err == nil && len(openChildren) > 0 {
		orphanIDs := make([]string, len(openChildren))
		for i, child := range openChildren {
			orphanIDs[i] = child.ID
		}
		logger := events.NewLogger(events.DefaultLogPath())
		if err := logger.LogEpicOrphaned(events.EpicOrphanedData{
			EpicID:           target.BeadsID,
			EpicTitle:        target.Issue.Title,
			OrphanedChildren: orphanIDs,
			Reason:           reason,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log epic orphan event: %v\n", err)
		}
		fmt.Fprintf(os.Stderr, "\033[1;33mWarning: Force-closing epic with %d open children (orphaned)\033[0m\n", len(openChildren))
		for _, child := range openChildren {
			fmt.Fprintf(os.Stderr, "  - %s (%s): %s\n", child.ID, child.Status, child.Title)
		}
	}
	return nil
}

// handleParentEpicAutoClose checks if this was the last open child of a parent epic.
func handleParentEpicAutoClose(target *CompletionTarget) {
	parentInfo, err := verify.GetParentEpicInfo(target.BeadsID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to check parent epic: %v\n", err)
		return
	}
	if parentInfo == nil || parentInfo.Status == "closed" || parentInfo.OpenChildrenLeft != 0 {
		return
	}

	if completeAutoCloseParent {
		if err := verify.CloseIssue(parentInfo.ID, "All children completed"); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to auto-close parent epic %s: %v\n", parentInfo.ID, err)
		} else {
			fmt.Printf("Auto-closed parent epic: %s (%s)\n", parentInfo.ID, parentInfo.Title)
		}
		return
	}

	// Interactive prompt
	fmt.Printf("\n\033[1;33mAll children of epic %s complete.\033[0m\n", parentInfo.ID)
	fmt.Printf("  Epic: %s\n", parentInfo.Title)
	fmt.Printf("\nClose parent epic? [y/N]: ")
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
		if err := verify.CloseIssue(parentInfo.ID, "All children completed"); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to close parent epic: %v\n", err)
		} else {
			fmt.Printf("Closed parent epic: %s\n", parentInfo.ID)
		}
	} else {
		fmt.Printf("Parent epic left open. Use 'orch complete %s' to close later.\n", parentInfo.ID)
	}
}

// updateOrchestratorSessionStatus updates the orchestrator session status in the registry.
func updateOrchestratorSessionStatus(agentName, status string) {
	registry := session.NewRegistry("")
	if err := registry.Update(agentName, func(s *session.OrchestratorSession) {
		s.Status = status
	}); err != nil {
		if err == session.ErrSessionNotFound {
			fmt.Printf("Note: Session %s was not in registry (legacy workspace)\n", agentName)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: failed to update session status in registry: %v\n", err)
		}
	} else {
		fmt.Printf("Updated session registry: status → %s\n", status)
	}
}

// ──────────────────────────────────────────────────────────────
// Phase 6: Cleanup
// ──────────────────────────────────────────────────────────────

// runCleanup handles session deletion, activity export, archival, docker, and tmux cleanup.

// exportActivity exports agent activity to the workspace.

// deleteSessionAndProcess deletes the OpenCode session and terminates the process.
// Returns (sessionDeleted, processTerminated).

// cleanupTmuxWindow finds and kills the tmux window for the agent.

// ──────────────────────────────────────────────────────────────
// Phase 7: Post-Complete
// ──────────────────────────────────────────────────────────────

// postComplete handles CLI command detection, changelog checks,
// state recording, event logging, and cache invalidation.
// Telemetry must be pre-collected because cleanup deletes the session and archives the workspace.
func postComplete(target *CompletionTarget, vOutcome *VerificationOutcome, reason string, telemetry CompletionTelemetry) {
	// Check for new CLI commands
	if hasGoChangesInRecentCommits(target.sourceDir()) {
		newCommands := detectNewCLICommands(target.sourceDir())
		if len(newCommands) > 0 {
			printNewCLICommandsNotice(newCommands)
		}
	}

	// Check for notable changelog entries
	if !completeNoChangelogCheck {
		var agentSkill string
		if target.WorkspacePath != "" {
			agentSkill, _ = verify.ExtractSkillNameFromSpawnContext(target.WorkspacePath)
		}
		notableEntries := detectNotableChangelogEntries(target.sourceDir(), agentSkill)
		if len(notableEntries) > 0 {
			printNotableChangelogEntries(notableEntries)
		}
	}

	// Record completion in state database
	if err := statedb.RecordComplete(target.AgentName, target.BeadsID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to record completion in state db: %v\n", err)
	}

	// Log completion event with pre-collected telemetry
	logger := events.NewLogger(events.DefaultLogPath())
	completedData := events.AgentCompletedData{
		Reason:             reason,
		Forced:             completeForce,
		Untracked:          target.IsUntracked,
		Orchestrator:       target.IsOrchestratorSession,
		VerificationPassed: vOutcome.Passed,
		Skill:              vOutcome.SkillName,
		DurationSeconds:    telemetry.DurationSecs,
		TokensInput:        telemetry.TokensIn,
		TokensOutput:       telemetry.TokensOut,
		Outcome:            telemetry.Outcome,
	}
	if target.BeadsID != "" {
		completedData.BeadsID = target.BeadsID
	}
	if target.AgentName != "" {
		completedData.Workspace = target.AgentName
	}
	if target.WorkspacePath != "" {
		completedData.AttemptID = spawn.ReadAttemptID(target.WorkspacePath)
	}
	if completeForce && len(vOutcome.GatesFailed) > 0 {
		completedData.GatesBypassed = vOutcome.GatesFailed
	}
	_ = completeBatch // TODO: Record batch mode metadata
	if err := logger.LogAgentCompleted(completedData); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Invalidate serve cache
	invalidateServeCache()
}

// printNewCLICommandsNotice prints the notice about detected new CLI commands.
func printNewCLICommandsNotice(newCommands []string) {
	newlyTracked := trackDocDebt(newCommands)
	fmt.Println()
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│  📚 NEW CLI COMMANDS DETECTED                               │")
	fmt.Println("├─────────────────────────────────────────────────────────────┤")
	for _, cmd := range newCommands {
		fmt.Printf("│  • %s\n", cmd)
	}
	fmt.Println("├─────────────────────────────────────────────────────────────┤")
	fmt.Println("│  Consider updating skill documentation:                     │")
	fmt.Println("│  - ~/.claude/skills/meta/orchestrator/SKILL.md              │")
	fmt.Println("│  - docs/orch-commands-reference.md                          │")
	fmt.Println("├─────────────────────────────────────────────────────────────┤")
	if newlyTracked > 0 {
		fmt.Printf("│  📝 Added %d command(s) to doc debt tracker                  │\n", newlyTracked)
	}
	fmt.Println("│  Run 'orch doctor --docs' to see all undocumented commands  │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
}

// printNotableChangelogEntries prints notable changelog entries.
func printNotableChangelogEntries(entries []string) {
	fmt.Println()
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│  ⚠️  NOTABLE ECOSYSTEM CHANGES DETECTED                      │")
	fmt.Println("├─────────────────────────────────────────────────────────────┤")
	for _, entry := range entries {
		if len(entry) > 55 {
			fmt.Printf("│  %s\n", entry[:55])
			fmt.Printf("│    %s\n", entry[55:])
		} else {
			fmt.Printf("│  %s\n", entry)
		}
	}
	fmt.Println("├─────────────────────────────────────────────────────────────┤")
	fmt.Println("│  Review recent changes that may affect agent behavior       │")
	fmt.Println("│  Run: orch changelog --days 3                               │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
}

// ──────────────────────────────────────────────────────────────
// Dead Session Recovery
// ──────────────────────────────────────────────────────────────

// DeadSessionRecovery holds the result of dead session detection.
type DeadSessionRecovery struct {
	IsDeadSession  bool
	HasCommits     bool
	Reason         string
	CommitCount    int
	LastCommitHash string
}

// detectDeadSessionWithCommits checks if this is a dead session that has commits
// and can be recovered automatically.
//
// A dead session is detected when:
//  1. Issue status is in_progress
//  2. No active OpenCode session exists
//  3. No "Phase: Complete" comment found
//  4. Agent branch has at least one commit
//
// This enables automatic recovery: orch complete can bypass the phase_complete gate
// while still running all other verification gates (build, test, commit_evidence).
func detectDeadSessionWithCommits(target *CompletionTarget) DeadSessionRecovery {
	result := DeadSessionRecovery{}

	// Only applies to tracked agents with beads issues
	if target == nil || target.BeadsID == "" || target.IsUntracked || target.IsOrchestratorSession {
		return result
	}

	// Check 1: Issue must be in_progress
	if target.Issue == nil || strings.ToLower(target.Issue.Status) != "in_progress" {
		return result
	}

	// Check 2: No active OpenCode session
	if hasSession := daemon.HasExistingSessionForBeadsID(target.BeadsID); hasSession {
		return result
	}

	// Check 3: No "Phase: Complete" comment
	// Use verify package which already checks both beads comments and state.db
	if hasComplete, _ := verify.IsPhaseComplete(target.BeadsID); hasComplete {
		return result
	}

	// Check 4: Agent branch must have commits
	if strings.TrimSpace(target.GitBranch) == "" || strings.TrimSpace(target.gitDir()) == "" {
		return result
	}

	// Count commits on agent branch
	worktree := target.gitDir()
	sourceDir := target.sourceDir()
	if sourceDir == "" {
		sourceDir = worktree
	}

	baseBranch, err := readBranchName(sourceDir)
	if err != nil || baseBranch == "" || baseBranch == target.GitBranch {
		return result
	}

	// Get merge-base and count commits
	mergeBase, err := runGitMerge(worktree, "merge-base", baseBranch, target.GitBranch)
	if err != nil {
		return result
	}

	countOut, err := runGitMerge(worktree, "rev-list", "--count", mergeBase+".."+target.GitBranch)
	if err != nil {
		return result
	}

	count := strings.TrimSpace(countOut)
	if count == "0" {
		return result
	}

	// Get last commit hash for logging
	lastCommit, _ := runGitMerge(worktree, "rev-parse", "--short", target.GitBranch)

	// All conditions met - this is a recoverable dead session
	result.IsDeadSession = true
	result.HasCommits = true
	result.CommitCount, _ = fmt.Sscanf(count, "%d", &result.CommitCount)
	result.LastCommitHash = strings.TrimSpace(lastCommit)
	result.Reason = fmt.Sprintf("Dead session recovery: agent died after committing %s commit(s) (%s)", count, result.LastCommitHash)

	return result
}
