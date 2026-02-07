// Package main provides verification and interactive gate stages for completion.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	statedb "github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"golang.org/x/term"
)

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
		title = strings.TrimSpace(title)
		if title == "" {
			fmt.Fprintln(os.Stderr, "  Skipping discovered work item with empty title")
			continue
		}

		description := strings.TrimSpace(item.Description)
		if description == "" {
			description = title
		}
		description = fmt.Sprintf("Follow-up discovered during completion of %s.\n\nSource item:\n%s", target.BeadsID, description)

		labels := []string{"triage:review"}
		if suggestedArea := beads.SuggestAreaLabel(title, description); suggestedArea != "" {
			labels = append(labels, suggestedArea)
			fmt.Printf("  Auto-applying area label: %s\n", suggestedArea)
		}

		issue, err := beads.FallbackCreate(title, description, "task", 2, labels)
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
