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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/activity"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/process"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	statedb "github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"golang.org/x/term"
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
	currentDir, err := os.Getwd()
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
func verifyCompletion(target *CompletionTarget, skipConfig SkipConfig) (*VerificationOutcome, error) {
	outcome := &VerificationOutcome{Passed: true}

	if completeForce {
		// --force: run verification to capture which gates would have failed, but don't block
		if !target.IsOrchestratorSession && !target.IsUntracked {
			result, err := verify.VerifyCompletionFull(target.BeadsID, target.WorkspacePath, target.BeadsProjectDir, "", serverURL)
			if err == nil {
				outcome.SkillName = result.Skill
				if !result.Passed {
					outcome.Passed = false
					outcome.GatesFailed = result.GatesFailed
				}
			}
		} else if target.IsOrchestratorSession {
			result := verify.VerifyOrchestratorCompletion(target.WorkspacePath)
			outcome.SkillName = result.Skill
			if !result.Passed {
				outcome.Passed = false
				outcome.GatesFailed = result.GatesFailed
			}
		}
		fmt.Println("Skipping all verification (--force) - DEPRECATED: use targeted --skip-* flags")
		return outcome, nil
	}

	if target.IsQuestion {
		fmt.Printf("Question entity: %s (skipping Phase: Complete - strategic node)\n", target.BeadsID)
		return outcome, nil
	}

	if target.IsOrchestratorSession {
		return verifyOrchestratorSession(target, skipConfig, outcome)
	}

	if !target.IsUntracked {
		return verifyRegularAgent(target, skipConfig, outcome)
	}

	fmt.Println("Skipping phase verification (untracked agent)")
	return outcome, nil
}

// verifyOrchestratorSession runs verification for orchestrator sessions.
func verifyOrchestratorSession(target *CompletionTarget, skipConfig SkipConfig, outcome *VerificationOutcome) (*VerificationOutcome, error) {
	if target.WorkspacePath != "" {
		fmt.Printf("Workspace: %s\n", target.AgentName)
	}

	result := verify.VerifyOrchestratorCompletion(target.WorkspacePath)
	outcome.SkillName = result.Skill

	// Apply skip-gate filtering (unified implementation)
	if skipConfig.hasAnySkip() && !result.Passed {
		applySkipFiltering(&result.GatesFailed, &result.Errors, skipConfig, target)
		result.Passed = len(result.GatesFailed) == 0
	}

	if !result.Passed {
		outcome.Passed = false
		outcome.GatesFailed = result.GatesFailed
		emitVerificationFailedEvent(target.BeadsID, target.AgentName, outcome.SkillName, result.GatesFailed, result.Errors)
		fmt.Fprintf(os.Stderr, "Cannot complete orchestrator session - verification failed:\n\n")
		printGateResults(result.GateResults, result.GatesFailed)
		return outcome, fmt.Errorf("verification failed")
	}

	fmt.Println("Completion signal: SYNTHESIS.md verified (content validated)")
	return outcome, nil
}

// verifyRegularAgent runs verification for regular (non-orchestrator) agents.
func verifyRegularAgent(target *CompletionTarget, skipConfig SkipConfig, outcome *VerificationOutcome) (*VerificationOutcome, error) {
	if target.WorkspacePath != "" {
		fmt.Printf("Workspace: %s\n", target.AgentName)
	}

	result, err := verify.VerifyCompletionFull(target.BeadsID, target.WorkspacePath, target.BeadsProjectDir, "", serverURL)
	if err != nil {
		return nil, fmt.Errorf("verification failed: %w", err)
	}
	outcome.SkillName = result.Skill

	// Apply skip-gate filtering (unified implementation)
	if skipConfig.hasAnySkip() && !result.Passed {
		applySkipFiltering(&result.GatesFailed, &result.Errors, skipConfig, target)
		result.Passed = len(result.GatesFailed) == 0
	}

	if !result.Passed {
		outcome.Passed = false
		outcome.GatesFailed = result.GatesFailed
		emitVerificationFailedEvent(target.BeadsID, target.AgentName, outcome.SkillName, result.GatesFailed, result.Errors)
		fmt.Fprintf(os.Stderr, "Cannot complete agent - verification failed:\n\n")
		printGateResults(result.GateResults, result.GatesFailed)
		return outcome, fmt.Errorf("verification failed")
	}

	// Print constraint warnings
	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "⚠️  %s\n", w)
	}

	// Print phase info
	if result.Phase.Found {
		fmt.Printf("Phase: %s\n", result.Phase.Phase)
		if result.Phase.Summary != "" {
			fmt.Printf("Summary: %s\n", result.Phase.Summary)
		}
	}

	// Behavioral validation checkpoint (informational, not blocking)
	if target.BeadsID != "" && target.BeadsProjectDir != "" {
		comments, _ := verify.GetComments(target.BeadsID)
		behavioralResult := verify.CheckBehavioralValidationForCompletion(target.BeadsID, target.WorkspacePath, target.BeadsProjectDir, comments)
		if behavioralResult != nil && behavioralResult.BehavioralValidationSuggested {
			printBehavioralValidationInfo(behavioralResult)
		}
	}

	return outcome, nil
}

// applySkipFiltering is the unified skip-gate-filtering implementation.
// It replaces the duplicated logic that was in both orchestrator and regular agent paths.
func applySkipFiltering(gatesFailed *[]string, errors *[]string, skipConfig SkipConfig, target *CompletionTarget) {
	var filteredGates []string
	var filteredErrors []string
	var skippedGatesFound []string

	for _, gate := range *gatesFailed {
		if skipConfig.shouldSkipGate(gate) {
			skippedGatesFound = append(skippedGatesFound, gate)
			fmt.Printf("⚠️  Bypassing gate: %s (reason: %s)\n", gate, skipConfig.Reason)
		} else {
			filteredGates = append(filteredGates, gate)
		}
	}

	// Filter errors - keep only those not related to skipped gates
	for _, e := range *errors {
		isSkippedError := false
		for _, gate := range skippedGatesFound {
			if strings.Contains(strings.ToLower(e), strings.ReplaceAll(gate, "_", " ")) ||
				strings.Contains(strings.ToLower(e), strings.ReplaceAll(gate, "_", "-")) ||
				(gate == verify.GateHandoffContent && (strings.Contains(e, "TLDR") || strings.Contains(e, "Outcome"))) {
				isSkippedError = true
				break
			}
		}
		if !isSkippedError {
			filteredErrors = append(filteredErrors, e)
		}
	}

	// Log bypass events
	if len(skippedGatesFound) > 0 {
		logSkipEvents(skipConfig, target.BeadsID, target.AgentName, "")
	}

	// Persist gate skip memory
	if len(skippedGatesFound) > 0 {
		skippedByID := target.BeadsID
		if skippedByID == "" {
			skippedByID = target.AgentName
		}
		persistGateSkipMemory(skippedGatesFound, skipConfig.Reason, target.BeadsProjectDir, skippedByID)
	}

	*gatesFailed = filteredGates
	*errors = filteredErrors
}

// emitVerificationFailedEvent logs a verification.failed event.
func emitVerificationFailedEvent(beadsID, workspace, skill string, gatesFailed, errors []string) {
	logger := events.NewLogger(events.DefaultLogPath())
	if err := logger.LogVerificationFailed(events.VerificationFailedData{
		BeadsID:     beadsID,
		Workspace:   workspace,
		GatesFailed: gatesFailed,
		Errors:      errors,
		Skill:       skill,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log verification failure event: %v\n", err)
	}
}

// ──────────────────────────────────────────────────────────────
// Phase 3: Check Liveness
// ──────────────────────────────────────────────────────────────

// checkLiveness warns if the agent appears still running and prompts for confirmation.
// Returns an error if the user declines to proceed.
func checkLiveness(target *CompletionTarget) error {
	if completeForce || target.IsUntracked {
		return nil
	}

	// If Phase: Complete was reported, skip liveness check
	phaseComplete := false
	if !target.IsOrchestratorSession && target.BeadsID != "" {
		phaseComplete, _ = verify.IsPhaseComplete(target.BeadsID)
	}
	if phaseComplete {
		return nil
	}

	liveness := statedb.GetLiveness(target.BeadsID, serverURL, target.BeadsProjectDir)
	if !liveness.IsAlive() {
		return nil
	}

	// Build warning message
	var runningDetails []string
	if liveness.TmuxLive {
		detail := "tmux window"
		if liveness.WindowID != "" {
			detail += " (" + liveness.WindowID + ")"
		}
		runningDetails = append(runningDetails, detail)
	}
	if liveness.OpencodeLive {
		detail := "OpenCode session"
		if liveness.SessionID != "" {
			detail += " (" + liveness.SessionID[:12] + ")"
		}
		runningDetails = append(runningDetails, detail)
	}

	fmt.Fprintf(os.Stderr, "⚠️  Agent appears still running: %s\n", strings.Join(runningDetails, ", "))

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("agent still running and stdin is not a terminal; use --force to complete anyway")
	}

	fmt.Fprint(os.Stderr, "Proceed anyway? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		return fmt.Errorf("aborted: agent still running")
	}

	fmt.Println("Proceeding with completion despite liveness warning...")
	return nil
}

// ──────────────────────────────────────────────────────────────
// Phase 4: Process Gates (discovered work, knowledge gaps)
// ──────────────────────────────────────────────────────────────

// processGates handles discovered work disposition and knowledge gap detection.
// These are interactive/informational gates that run after verification passes.
func processGates(target *CompletionTarget, skillName string) error {
	if target.WorkspacePath == "" || completeForce {
		return nil
	}

	// Discovered work disposition gate
	if err := processDiscoveredWork(target); err != nil {
		return err
	}

	// Knowledge gap detection (informational, non-blocking)
	processKnowledgeGaps(target, skillName)

	return nil
}

// processDiscoveredWork handles the discovered work disposition gate.
func processDiscoveredWork(target *CompletionTarget) error {
	synthesis, err := verify.ParseSynthesis(target.WorkspacePath)
	if err != nil || synthesis == nil {
		return nil
	}

	items := verify.CollectDiscoveredWork(synthesis)
	if len(items) == 0 {
		return nil
	}

	fmt.Println("\n--- Discovered Work Gate ---")
	if synthesis.Recommendation != "" && synthesis.Recommendation != "close" {
		fmt.Printf("Recommendation: %s\n", synthesis.Recommendation)
	}
	fmt.Printf("%d discovered work item(s) require disposition:\n", len(items))

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Println("(Skipping interactive prompts - stdin is not a terminal)")
		fmt.Println("Use --force to complete without disposition, or run interactively")
		fmt.Println("---------------------------------")
		return nil
	}

	result, err := verify.PromptDiscoveredWorkDisposition(items, os.Stdin, os.Stdout)
	if err != nil {
		return fmt.Errorf("discovered work disposition failed: %w\n\nCompletion blocked. Run again to disposition all items, or use --force to skip", err)
	}

	if !result.AllDispositioned {
		return fmt.Errorf("not all discovered work items were dispositioned\n\nCompletion blocked. Run again to disposition all items, or use --force to skip")
	}

	// File issues for items marked 'y'
	filedItems := result.FiledItems()
	createdCount := 0
	for _, item := range filedItems {
		title := strings.TrimPrefix(item.Description, "- ")
		title = strings.TrimPrefix(title, "* ")
		if len(title) > 3 && title[0] >= '0' && title[0] <= '9' && (title[1] == '.' || (title[1] >= '0' && title[1] <= '9' && title[2] == '.')) {
			if idx := strings.Index(title, ". "); idx != -1 && idx < 4 {
				title = title[idx+2:]
			}
		}

		labels := []string{"triage:review"}
		if suggestedArea := beads.SuggestAreaLabel(title, ""); suggestedArea != "" {
			labels = append(labels, suggestedArea)
			fmt.Printf("  Auto-applying area label: %s\n", suggestedArea)
		}

		issue, err := beads.FallbackCreate(title, "", "task", 2, labels)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Failed to create issue: %v\n", err)
		} else {
			fmt.Printf("  Created: %s - %s\n", issue.ID, title)
			createdCount++
		}
	}

	if createdCount > 0 {
		fmt.Printf("\n✓ Created %d follow-up issue(s)\n", createdCount)
	}

	if result.SkipAllReason != "" {
		fmt.Printf("Skip-all reason: %s\n", result.SkipAllReason)
	}

	skippedItems := result.SkippedItems()
	if len(skippedItems) > 0 {
		fmt.Printf("Skipped %d item(s)\n", len(skippedItems))
	}

	fmt.Println("---------------------------------")
	return nil
}

// processKnowledgeGaps detects and logs knowledge gaps (informational, non-blocking).
func processKnowledgeGaps(target *CompletionTarget, skillName string) {
	gapResult, err := verify.DetectKnowledgeGaps(target.WorkspacePath, target.BeadsID, skillName, target.BeadsProjectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to detect knowledge gaps: %v\n", err)
		return
	}
	if gapResult == nil || gapResult.GapsDetected == 0 {
		return
	}

	if err := verify.LogKnowledgeGaps(gapResult.Gaps); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log knowledge gaps: %v\n", err)
	} else {
		fmt.Printf("\nℹ️  Knowledge Gap Detection: %d gap(s) detected and logged\n", gapResult.GapsDetected)
		fmt.Printf("   Agent surfaced questions that kb already answers.\n")
		fmt.Printf("   Review: cat ~/.orch/knowledge-gaps.jsonl | jq 'select(.workspace==\"%s\")'\n", target.AgentName)
	}
}

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
func runCleanup(target *CompletionTarget) *CleanupOutcome {
	outcome := &CleanupOutcome{}

	if target.WorkspacePath == "" {
		return outcome
	}

	// Export activity (before session deletion)
	if !target.IsOrchestratorSession {
		exportActivity(target)
	}

	// Delete OpenCode session and terminate process
	outcome.SessionDeleted, outcome.ProcessTerminated = deleteSessionAndProcess(target)

	// Export orchestrator transcript (before archival)
	if target.IsOrchestratorSession {
		if err := exportOrchestratorTranscript(target.WorkspacePath, target.BeadsProjectDir, target.AgentName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to export orchestrator transcript: %v\n", err)
		} else {
			outcome.TranscriptExported = true
		}
	}

	// Archive workspace
	if !completeNoArchive {
		archivedPath, err := archiveWorkspace(target.WorkspacePath, target.BeadsProjectDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to archive workspace: %v\n", err)
		} else {
			outcome.ArchivedPath = archivedPath
			fmt.Printf("Archived workspace: %s\n", filepath.Base(archivedPath))

			if target.IsOrchestratorSession && archivedPath != "" {
				registry := session.NewRegistry("")
				if err := registry.Update(target.AgentName, func(s *session.OrchestratorSession) {
					s.ArchivedPath = archivedPath
				}); err != nil {
					if err != session.ErrSessionNotFound {
						fmt.Fprintf(os.Stderr, "Warning: failed to update archived path in registry: %v\n", err)
					}
				}
			}
		}
	} else {
		fmt.Println("Skipped workspace archival (--no-archive)")
	}

	// Clean up Docker container
	containerName := spawn.ReadContainerID(target.WorkspacePath)
	if containerName != "" {
		if err := spawn.CleanupDockerContainer(containerName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean up Docker container %s: %v\n", containerName, err)
		} else {
			outcome.DockerCleaned = true
			fmt.Printf("Cleaned up Docker container: %s\n", containerName)
		}
	}

	// Clean up tmux window
	outcome.TmuxWindowClosed = cleanupTmuxWindow(target)

	return outcome
}

// exportActivity exports agent activity to the workspace.
func exportActivity(target *CompletionTarget) {
	sessionFile := filepath.Join(target.WorkspacePath, ".session_id")
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return
	}
	sessionID := strings.TrimSpace(string(data))
	if sessionID == "" {
		return
	}

	activityPath, err := activity.ExportToWorkspace(sessionID, target.WorkspacePath, serverURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to export activity: %v\n", err)
	} else if activityPath != "" {
		fmt.Printf("Exported activity: %s\n", filepath.Base(activityPath))
	}
}

// deleteSessionAndProcess deletes the OpenCode session and terminates the process.
// Returns (sessionDeleted, processTerminated).
func deleteSessionAndProcess(target *CompletionTarget) (bool, bool) {
	return deleteSessionAndProcessWithClient(opencode.NewClient(serverURL), target)
}

func deleteSessionAndProcessWithClient(client opencode.ClientInterface, target *CompletionTarget) (bool, bool) {
	var sessionDeleted, processTerminated bool

	sessionFile := filepath.Join(target.WorkspacePath, ".session_id")
	data, err := os.ReadFile(sessionFile)
	if err == nil {
		sessionID := strings.TrimSpace(string(data))
		if sessionID != "" {
			if err := client.DeleteSession(sessionID); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to delete OpenCode session %s: %v\n", sessionID[:12], err)
			} else {
				sessionDeleted = true
				fmt.Printf("Deleted OpenCode session: %s\n", sessionID[:12])
			}
		}
	}

	pid := spawn.ReadProcessID(target.WorkspacePath)
	if pid > 0 {
		if process.Terminate(pid, "opencode") {
			processTerminated = true
		}
	}

	return sessionDeleted, processTerminated
}

// cleanupTmuxWindow finds and kills the tmux window for the agent.
func cleanupTmuxWindow(target *CompletionTarget) bool {
	var window *tmux.WindowInfo
	var tmuxSessionName string
	var findErr error

	if target.IsOrchestratorSession {
		window, tmuxSessionName, findErr = tmux.FindWindowByWorkspaceNameAllSessions(target.AgentName)
	} else {
		windowSearchID := target.BeadsID
		if windowSearchID == "" {
			windowSearchID = target.Identifier
		}
		window, tmuxSessionName, findErr = tmux.FindWindowByBeadsIDAllSessions(windowSearchID)
	}

	if findErr != nil || window == nil {
		return false
	}

	if err := tmux.KillWindow(window.Target); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to close tmux window %s: %v\n", window.Target, err)
		return false
	}

	fmt.Printf("Closed tmux window: %s:%s\n", tmuxSessionName, window.Name)
	return true
}

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
