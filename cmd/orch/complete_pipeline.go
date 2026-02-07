// Package main provides the pipeline decomposition of the complete command.
// Each phase is a standalone function with typed input/output, enabling isolated testing.
//
// Pipeline phases:
//  1. ResolveTarget:    identifier → CompletionTarget
//  2. VerifyCompletion: target + skipConfig → VerificationOutcome
//  3. CheckLiveness:    target → (prompt or continue)
//  4. ProcessGates:     target → (discovered work, knowledge gaps, etc.)
//  5. CloseIssue:       target + reason → CloseOutcome (includes epic handling)
//  6. Cleanup:          target → CleanupOutcome (session, archive, docker, tmux)
//  7. PostComplete:     target + outcomes → (telemetry, events, cache)
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/session"
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
	IsOrchestratorSession bool
	IsUntracked           bool
	IsQuestion            bool
	IsClosed              bool
	Issue                 *verify.Issue // nil for untracked/orchestrator
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
	if hasGoChangesInRecentCommits(target.BeadsProjectDir) {
		newCommands := detectNewCLICommands(target.BeadsProjectDir)
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
		notableEntries := detectNotableChangelogEntries(target.BeadsProjectDir, agentSkill)
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
