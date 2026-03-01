// Package main provides the completion pipeline phases for decomposing runComplete()
// into typed, testable phase functions.
//
// Pipeline: resolveCompletionTarget → executeVerificationGates → runCompletionAdvisories → executeLifecycleTransition
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/activity"
	"github.com/dylan-conlin/orch-go/pkg/agent"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/checkpoint"
	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/orch"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"golang.org/x/term"
)

// CompletionTarget holds the resolved target of a completion operation.
type CompletionTarget struct {
	Identifier            string // Original command-line identifier
	WorkspacePath         string
	AgentName             string
	BeadsID               string
	BeadsProjectDir       string
	IsOrchestratorSession bool
	IsUntracked           bool
	Issue                 *verify.Issue
	IsClosed              bool
}

// VerificationOutcome holds the results of verification gate execution.
type VerificationOutcome struct {
	Passed      bool
	GatesFailed []string
	SkillName   string
	Result      verify.VerificationResult
	ResultSet   bool
}

// AdvisoryResults holds the results of running completion advisories.
// Advisories are primarily side-effects (printing), but this struct exists
// for pipeline type consistency and future extensibility.
type AdvisoryResults struct{}

// resolveCompletionTarget resolves the identifier to a completion target,
// finding workspace paths, beads IDs, and project directories.
func resolveCompletionTarget(identifier, workdir string) (CompletionTarget, error) {
	var target CompletionTarget
	target.Identifier = identifier

	currentDir, err := os.Getwd()
	if err != nil {
		return target, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Step 1: Try direct workspace name lookup in current directory
	directWorkspacePath := findWorkspaceByName(currentDir, identifier)
	if directWorkspacePath != "" {
		target.WorkspacePath = directWorkspacePath
		target.AgentName = identifier
		if isOrchestratorWorkspace(directWorkspacePath) {
			target.IsOrchestratorSession = true
			fmt.Printf("Orchestrator session: %s\n", target.AgentName)
		} else {
			manifest := spawn.ReadAgentManifestWithFallback(directWorkspacePath)
			target.BeadsID = manifest.BeadsID
		}
	}

	// Step 2: Search for workspace across known projects if identifier looks like a workspace name
	if target.WorkspacePath == "" && looksLikeWorkspaceName(identifier) {
		if foundPath := findWorkspaceByNameAcrossProjects(identifier); foundPath != "" {
			target.WorkspacePath = foundPath
			target.AgentName = identifier
			if isOrchestratorWorkspace(foundPath) {
				target.IsOrchestratorSession = true
				fmt.Printf("Orchestrator session (cross-project): %s\n", target.AgentName)
			} else {
				manifest := spawn.ReadAgentManifestWithFallback(foundPath)
				target.BeadsID = manifest.BeadsID
			}
		}
	}

	// Step 3: If no workspace match and not an orchestrator session, treat identifier as beads ID
	if target.WorkspacePath == "" && !target.IsOrchestratorSession {
		// Auto-detect cross-project agents BEFORE resolution
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

		// Resolve short beads ID to full ID
		resolvedID, err := resolveShortBeadsID(identifier)
		if err != nil {
			return target, fmt.Errorf("failed to resolve beads ID: %w", err)
		}
		target.BeadsID = resolvedID

		// Find workspace by beads ID
		searchDir := currentDir
		if crossProjectDir != "" {
			searchDir = crossProjectDir
		}
		target.WorkspacePath, target.AgentName = findWorkspaceByBeadsID(searchDir, target.BeadsID)
	}

	// Determine beads project directory:
	// 1. If --workdir provided, use that
	// 2. Otherwise, try to auto-detect from workspace SPAWN_CONTEXT.md
	// 3. Fall back to current directory
	if workdir != "" {
		target.BeadsProjectDir, err = filepath.Abs(workdir)
		if err != nil {
			return target, fmt.Errorf("failed to resolve workdir path: %w", err)
		}
		if stat, err := os.Stat(target.BeadsProjectDir); err != nil {
			return target, fmt.Errorf("workdir does not exist: %s", target.BeadsProjectDir)
		} else if !stat.IsDir() {
			return target, fmt.Errorf("workdir is not a directory: %s", target.BeadsProjectDir)
		}
		fmt.Printf("Using explicit workdir: %s\n", target.BeadsProjectDir)
	} else if target.WorkspacePath != "" {
		projectDirFromWorkspace := extractProjectDirFromWorkspace(target.WorkspacePath)
		if projectDirFromWorkspace != "" && projectDirFromWorkspace != currentDir {
			target.BeadsProjectDir = projectDirFromWorkspace
			fmt.Printf("Auto-detected cross-project: %s\n", filepath.Base(target.BeadsProjectDir))
		} else {
			target.BeadsProjectDir = currentDir
		}
	} else {
		target.BeadsProjectDir = currentDir
	}

	// Set beads.DefaultDir for cross-project operations BEFORE any beads operations
	if target.BeadsProjectDir != currentDir {
		beads.DefaultDir = target.BeadsProjectDir
	}

	// Check if this is an untracked agent
	target.IsUntracked = target.IsOrchestratorSession ||
		(target.BeadsID != "" && isUntrackedBeadsID(target.BeadsID)) ||
		target.BeadsID == ""

	// For tracked agents, verify the beads issue exists
	if !target.IsUntracked {
		issue, err := verify.GetIssue(target.BeadsID)
		if err != nil {
			projectName := filepath.Base(target.BeadsProjectDir)
			issuePrefix := strings.Split(target.BeadsID, "-")[0]
			if len(strings.Split(target.BeadsID, "-")) > 1 {
				issuePrefix = strings.Join(strings.Split(target.BeadsID, "-")[:len(strings.Split(target.BeadsID, "-"))-1], "-")
			}
			if issuePrefix != projectName {
				return target, fmt.Errorf("failed to get beads issue %s: %w\n\nHint: The issue ID suggests it belongs to project '%s', but you're in '%s'.\nTry: orch complete %s --workdir ~/path/to/%s", target.BeadsID, err, issuePrefix, projectName, target.BeadsID, issuePrefix)
			}
			return target, fmt.Errorf("failed to get beads issue: %w", err)
		}
		target.Issue = issue
		target.IsClosed = issue.Status == "closed"
		if target.IsClosed {
			fmt.Printf("Issue %s is already closed in beads\n", target.BeadsID)
		}
	} else if target.IsOrchestratorSession {
		fmt.Printf("Note: %s is an orchestrator session (no beads tracking)\n", target.AgentName)
		target.IsClosed = false
	} else {
		fmt.Printf("Note: %s is an untracked agent (no beads issue)\n", identifier)
		target.IsClosed = false
	}

	return target, nil
}

// executeVerificationGates runs checkpoint gates, completion verification, skip
// filtering, and liveness checks. Returns a VerificationOutcome with results.
func executeVerificationGates(target CompletionTarget, skipConfig verify.SkipConfig) (VerificationOutcome, error) {
	var outcome VerificationOutcome
	outcome.Passed = true

	// Checkpoint verification gate (Verifiability-first enforcement)
	var checkpointErrors []string
	var checkpointGatesFailed []string
	if !target.IsUntracked && !completeForce && target.Issue != nil {
		tier := checkpoint.TierForIssueType(target.Issue.IssueType)

		if checkpoint.RequiresCheckpoint(target.Issue.IssueType) {
			hasGate1, err := checkpoint.HasGate1Checkpoint(target.BeadsID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to check verification checkpoint: %v\n", err)
			} else if !hasGate1 && !skipConfig.ExplainBack && completeExplain == "" {
				checkpointGatesFailed = append(checkpointGatesFailed, verify.GateExplainBack)
				checkpointErrors = append(checkpointErrors,
					fmt.Sprintf("comprehension gate (gate1) missing for Tier %d work (%s) — use --explain 'Built X because Y'", tier, target.Issue.IssueType))
			} else if hasGate1 {
				fmt.Println("✓ Comprehension gate (gate1) passed")
			}
		}

		if checkpoint.RequiresGate2(target.Issue.IssueType) {
			hasGate2, err := checkpoint.HasGate2Checkpoint(target.BeadsID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to check gate2 checkpoint: %v\n", err)
			} else if !hasGate2 && !completeVerified {
				checkpointGatesFailed = append(checkpointGatesFailed, "verified")
				checkpointErrors = append(checkpointErrors,
					fmt.Sprintf("behavioral verification (gate2) missing for Tier 1 work (%s) — use --verified", target.Issue.IssueType))
			} else if hasGate2 {
				fmt.Println("✓ Behavioral verification (gate2) passed")
			}
		}
	}

	// If --approve flag is set, add approval comment BEFORE verification
	if completeApprove && !target.IsUntracked {
		approvalComment := "✅ APPROVED - Visual changes reviewed and approved by orchestrator"
		if err := addApprovalComment(target.BeadsID, approvalComment); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to add approval comment: %v\n", err)
		} else {
			fmt.Printf("Added approval: %s\n", approvalComment)
		}
	}

	// Verify completion status
	if !completeForce {
		if target.IsOrchestratorSession {
			// Orchestrator sessions use SESSION_HANDOFF.md as completion signal
			if target.WorkspacePath != "" {
				fmt.Printf("Workspace: %s\n", target.AgentName)
			}

			result := verify.VerifyOrchestratorCompletion(target.WorkspacePath)
			outcome.SkillName = result.Skill
			outcome.Result = result
			outcome.ResultSet = true

			// Apply skip config to filter out bypassed gates
			if skipConfig.HasAnySkip() && !result.Passed {
				applySkipFilters(&result, skipConfig, "", target.AgentName, outcome.SkillName)
			}

			if !result.Passed {
				outcome.Passed = false
				outcome.GatesFailed = result.GatesFailed

				logger := events.NewLogger(events.DefaultLogPath())
				if err := logger.LogVerificationFailed(events.VerificationFailedData{
					Workspace:   target.AgentName,
					GatesFailed: result.GatesFailed,
					Errors:      result.Errors,
					Skill:       outcome.SkillName,
				}); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log verification failure event: %v\n", err)
				}

				fmt.Fprintf(os.Stderr, "Cannot complete orchestrator session - %d gate(s) failed:\n", len(result.GatesFailed))
				for i, e := range result.Errors {
					gate := ""
					if i < len(result.GatesFailed) {
						gate = result.GatesFailed[i]
					}
					if gate != "" {
						fmt.Fprintf(os.Stderr, "  ❌ %s: %s\n", gate, e)
					} else {
						fmt.Fprintf(os.Stderr, "  ❌ %s\n", e)
					}
				}
				fmt.Fprintf(os.Stderr, "\nOrchestrator must fill SESSION_HANDOFF.md with:\n")
				fmt.Fprintf(os.Stderr, "  - TLDR section (actual content, not placeholder)\n")
				fmt.Fprintf(os.Stderr, "  - Outcome field (success, partial, blocked, or failed)\n")
				fmt.Fprintf(os.Stderr, "Or use --skip-handoff-content --skip-reason \"...\" to bypass\n")
				return outcome, fmt.Errorf("verification failed: %d gate(s)", len(result.GatesFailed))
			}
			fmt.Println("Completion signal: SESSION_HANDOFF.md verified (content validated)")
		} else if !target.IsUntracked {
			// Regular agents use beads phase verification
			if target.WorkspacePath != "" {
				fmt.Printf("Workspace: %s\n", target.AgentName)
			}

			result, err := verify.VerifyCompletionFull(target.BeadsID, target.WorkspacePath, target.BeadsProjectDir, "", serverURL)
			if err != nil {
				return outcome, fmt.Errorf("verification failed: %w", err)
			}
			outcome.Result = result
			outcome.ResultSet = true
			outcome.SkillName = result.Skill

			// If skip flags are set, filter out the skipped gates from failures
			if skipConfig.HasAnySkip() && !result.Passed {
				applySkipFilters(&result, skipConfig, target.BeadsID, target.AgentName, outcome.SkillName)
			}

			// Surface model references for modified files (informational only)
			if target.WorkspacePath != "" && target.BeadsProjectDir != "" {
				matches, err := verify.FindModelReferencesForModifiedFiles(target.WorkspacePath, target.BeadsProjectDir)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to check model code references: %v\n", err)
				} else if note := verify.FormatModelReferenceNote(matches); note != "" {
					fmt.Println(note)
				}
			}

			// Merge checkpoint gate failures into verification result
			if len(checkpointGatesFailed) > 0 {
				result.GatesFailed = append(result.GatesFailed, checkpointGatesFailed...)
				result.Errors = append(result.Errors, checkpointErrors...)
				result.Passed = false
			}

			if !result.Passed {
				outcome.Passed = false
				outcome.GatesFailed = result.GatesFailed

				logger := events.NewLogger(events.DefaultLogPath())
				if err := logger.LogVerificationFailed(events.VerificationFailedData{
					BeadsID:     target.BeadsID,
					Workspace:   target.AgentName,
					GatesFailed: result.GatesFailed,
					Errors:      result.Errors,
					Skill:       outcome.SkillName,
				}); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log verification failure event: %v\n", err)
				}

				fmt.Fprintf(os.Stderr, "Cannot complete agent - %d gate(s) failed:\n", len(result.GatesFailed))
				for i, e := range result.Errors {
					gate := ""
					if i < len(result.GatesFailed) {
						gate = result.GatesFailed[i]
					}
					if gate != "" {
						fmt.Fprintf(os.Stderr, "  ❌ %s: %s\n", gate, e)
					} else {
						fmt.Fprintf(os.Stderr, "  ❌ %s\n", e)
					}
				}
				// Print fix hints
				fmt.Fprintf(os.Stderr, "\nTo fix:\n")
				fmt.Fprintf(os.Stderr, "  orch complete %s", target.BeadsID)
				for _, g := range result.GatesFailed {
					if g == verify.GateExplainBack {
						fmt.Fprintf(os.Stderr, " --explain '...'")
					} else if g == "verified" {
						fmt.Fprintf(os.Stderr, " --verified")
					}
				}
				fmt.Fprintln(os.Stderr)
				fmt.Fprintf(os.Stderr, "  Use --skip-<gate> --skip-reason \"...\" to bypass specific gates\n")
				return outcome, fmt.Errorf("verification failed: %d gate(s)", len(result.GatesFailed))
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
		} else {
			fmt.Println("Skipping phase verification (untracked agent)")
		}
	} else {
		// --force was used, run verification anyway to capture which gates would have failed
		if !target.IsOrchestratorSession && !target.IsUntracked {
			result, err := verify.VerifyCompletionFull(target.BeadsID, target.WorkspacePath, target.BeadsProjectDir, "", serverURL)
			if err == nil {
				outcome.SkillName = result.Skill
				outcome.Result = result
				outcome.ResultSet = true
				if !result.Passed {
					outcome.Passed = false
					outcome.GatesFailed = result.GatesFailed
				}
			}
		} else if target.IsOrchestratorSession {
			result := verify.VerifyOrchestratorCompletion(target.WorkspacePath)
			outcome.SkillName = result.Skill
			outcome.Result = result
			outcome.ResultSet = true
			if !result.Passed {
				outcome.Passed = false
				outcome.GatesFailed = result.GatesFailed
			}
		}
		fmt.Println("Skipping all verification (--force) - DEPRECATED: use targeted --skip-* flags")
	}

	// Check liveness before closing - warn if agent appears still running
	if !completeForce && !target.IsUntracked {
		phaseComplete := false
		if !target.IsOrchestratorSession && target.BeadsID != "" {
			phaseComplete, _ = verify.IsPhaseComplete(target.BeadsID)
		}

		if !phaseComplete {
			liveness := state.GetLiveness(target.BeadsID, serverURL, target.BeadsProjectDir)
			if liveness.IsAlive() {
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
						detail += " (" + shortID(liveness.SessionID) + ")"
					}
					runningDetails = append(runningDetails, detail)
				}

				fmt.Fprintf(os.Stderr, "⚠️  Agent appears still running: %s\n", strings.Join(runningDetails, ", "))

				if !term.IsTerminal(int(os.Stdin.Fd())) {
					return outcome, fmt.Errorf("agent still running and stdin is not a terminal; use --force to complete anyway")
				}

				fmt.Fprint(os.Stderr, "Proceed anyway? [y/N]: ")
				reader := bufio.NewReader(os.Stdin)
				response, err := reader.ReadString('\n')
				if err != nil {
					return outcome, fmt.Errorf("failed to read response: %w", err)
				}

				response = strings.TrimSpace(strings.ToLower(response))
				if response != "y" && response != "yes" {
					return outcome, fmt.Errorf("aborted: agent still running")
				}

				fmt.Println("Proceeding with completion despite liveness warning...")
			}
		}
	}

	// DISABLED: Reproduction verification gate (Jan 4, 2026)
	_ = completeSkipReproCheck
	_ = completeSkipReproReason

	return outcome, nil
}

// runCompletionAdvisories runs advisory gates: discovered work, probe verdicts,
// architectural choices, knowledge maintenance, explain-back, and verification checklist.
func runCompletionAdvisories(target CompletionTarget, outcome VerificationOutcome, skipConfig verify.SkipConfig) (AdvisoryResults, error) {
	var advisories AdvisoryResults

	// Gate completion on discovered work disposition
	if target.WorkspacePath != "" && !completeForce {
		synthesis, err := verify.ParseSynthesis(target.WorkspacePath)
		if err == nil && synthesis != nil {
			items := verify.CollectDiscoveredWork(synthesis)

			if len(items) > 0 {
				fmt.Println("\n--- Discovered Work Gate ---")

				if synthesis.Recommendation != "" && synthesis.Recommendation != "close" {
					fmt.Printf("Recommendation: %s\n", synthesis.Recommendation)
				}

				fmt.Printf("%d discovered work item(s) require disposition:\n", len(items))

				if !term.IsTerminal(int(os.Stdin.Fd())) {
					fmt.Println("(Skipping interactive prompts - stdin is not a terminal)")
					fmt.Println("Use --force to complete without disposition, or run interactively")
				} else {
					result, err := verify.PromptDiscoveredWorkDisposition(items, os.Stdin, os.Stdout)
					if err != nil {
						return advisories, fmt.Errorf("discovered work disposition failed: %w\n\nCompletion blocked. Run again to disposition all items, or use --force to skip", err)
					}

					if !result.AllDispositioned {
						return advisories, fmt.Errorf("not all discovered work items were dispositioned\n\nCompletion blocked. Run again to disposition all items, or use --force to skip")
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

						issue, err := beads.FallbackCreate(title, "", "task", 2, []string{"triage:review"})
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
				}

				fmt.Println("---------------------------------")
			}
		}
	}

	// Surface probe verdicts for orchestrator review
	if target.WorkspacePath != "" {
		probeVerdicts := verify.FindProbesForWorkspace(target.WorkspacePath, target.BeadsProjectDir)
		if len(probeVerdicts) > 0 {
			fmt.Print(verify.FormatProbeVerdicts(probeVerdicts))
		}
	}

	// Surface architectural choices for orchestrator review
	if target.WorkspacePath != "" && !target.IsOrchestratorSession {
		if choicesOutput := verify.FormatArchitecturalChoicesForCompletion(target.WorkspacePath); choicesOutput != "" {
			fmt.Print(choicesOutput)
		}
	}

	// Knowledge maintenance step (Touchpoint 1: Completion Review)
	if !target.IsOrchestratorSession && !target.IsUntracked && !completeForce {
		var issueTitle string
		if target.Issue != nil {
			issueTitle = target.Issue.Title
		}
		var phaseSummary string
		if outcome.ResultSet && outcome.Result.Phase.Found {
			phaseSummary = outcome.Result.Phase.Summary
		}
		if err := RunKnowledgeMaintenance(outcome.SkillName, issueTitle, phaseSummary, os.Stdout, os.Stdin); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: knowledge maintenance failed: %v\n", err)
		}
	}

	// Explain-back verification gate
	priorGate1, _ := checkpoint.HasGate1Checkpoint(target.BeadsID)
	if completeExplain != "" || !priorGate1 {
		if err := orch.RunExplainBackGate(
			target.BeadsID,
			completeForce,
			skipConfig.ExplainBack,
			skipConfig.Reason,
			target.IsOrchestratorSession,
			target.IsUntracked,
			completeExplain,
			completeVerified,
			os.Stdout,
		); err != nil {
			return advisories, err
		}
	}

	// Record gate2 checkpoint if --verified flag is set and explain-back gate didn't run
	if completeVerified && !target.IsUntracked && !target.IsOrchestratorSession && target.BeadsID != "" {
		hasGate2, _ := checkpoint.HasGate2Checkpoint(target.BeadsID)
		if !hasGate2 && priorGate1 && completeExplain == "" {
			if err := orch.RecordGate2Checkpoint(target.BeadsID, os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to record gate2 checkpoint: %v\n", err)
			}
		}
	}

	// Surface verification checklist before closing
	if outcome.ResultSet && !target.IsUntracked {
		gate1Complete := false
		gate2Complete := false
		if target.BeadsID != "" && !target.IsOrchestratorSession {
			gate1Complete, _ = checkpoint.HasGate1Checkpoint(target.BeadsID)
			gate2Complete, _ = checkpoint.HasGate2Checkpoint(target.BeadsID)
		}
		tier := ""
		if target.WorkspacePath != "" && !target.IsOrchestratorSession {
			tier = verify.ReadTierFromWorkspace(target.WorkspacePath)
		}
		issueType := ""
		if target.Issue != nil {
			issueType = target.Issue.IssueType
		}
		checklist := buildVerificationChecklist(outcome.Result, issueType, tier, target.IsOrchestratorSession, skipConfig, gate1Complete, gate2Complete)

		// Compute trust calibration tier from review tier + bypass signals
		reviewTier := ""
		if target.WorkspacePath != "" && !target.IsOrchestratorSession {
			reviewTier = verify.ReadReviewTierFromWorkspace(target.WorkspacePath)
		}
		trustTier := ComputeTrustTier(reviewTier, skipConfig, completeForce)
		printVerificationChecklist(checklist, trustTier)
	}

	// Surface hotspot advisory for modified files (informational, not a gate)
	if target.BeadsProjectDir != "" && !target.IsOrchestratorSession {
		if advisory := RunHotspotAdvisoryForCompletion(target.BeadsProjectDir); advisory != "" {
			fmt.Print(advisory)
		}
	}

	// Surface model-impact advisory (informational, not a gate)
	if target.BeadsProjectDir != "" && target.WorkspacePath != "" && !target.IsOrchestratorSession {
		if advisory := RunModelImpactAdvisory(target.BeadsProjectDir, target.WorkspacePath); advisory != "" {
			fmt.Print(advisory)
		}
	}

	// Surface synthesis checkpoint (informational, not a gate)
	if !target.IsOrchestratorSession && !target.IsUntracked {
		var issueTitle string
		if target.Issue != nil {
			issueTitle = target.Issue.Title
		}
		var phaseSummary string
		if outcome.ResultSet && outcome.Result.Phase.Found {
			phaseSummary = outcome.Result.Phase.Summary
		}
		if advisory := RunSynthesisCheckpoint(outcome.SkillName, issueTitle, phaseSummary); advisory != "" {
			fmt.Print(advisory)
		}
	}

	// Update session handoff with spawn completion info
	if !target.IsOrchestratorSession && target.AgentName != "" && target.BeadsID != "" {
		if err := UpdateHandoffAfterComplete(target.BeadsProjectDir, target.AgentName, target.BeadsID, outcome.SkillName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update session handoff: %v\n", err)
		}
	}

	return advisories, nil
}

// executeLifecycleTransition handles close reason determination, pre-lifecycle exports,
// LifecycleManager.Complete execution, and all post-lifecycle operations (triage label,
// daemon signal, auto-rebuild, changelog, telemetry, event logging, cache invalidation).
// Returns true if LifecycleManager handled tmux cleanup (so deferred cleanup can be skipped).
func executeLifecycleTransition(target CompletionTarget, outcome VerificationOutcome, _ AdvisoryResults) (lifecycleCleanedUp bool, err error) {
	// Determine close reason
	reason := completeReason
	if reason == "" {
		if !target.IsUntracked && target.BeadsID != "" {
			status, _ := verify.GetPhaseStatus(target.BeadsID)
			if status.Summary != "" {
				reason = status.Summary
			}
		}
		if reason == "" {
			if target.IsOrchestratorSession {
				reason = "Orchestrator session completed"
			} else {
				reason = "Completed via orch complete"
			}
		}
	}

	// Auto-create implementation issue for architect completions
	if !target.IsUntracked && !target.IsOrchestratorSession && target.WorkspacePath != "" {
		maybeAutoCreateImplementationIssue(outcome.SkillName, target.BeadsID, target.WorkspacePath)
	}

	// --- Pre-lifecycle operations (need session/workspace alive) ---

	// Collect telemetry BEFORE lifecycle transition, because lm.Complete()
	// archives the workspace (moves it to archived/), making the manifest
	// unreadable at the original path.
	var durationSecs, tokensIn, tokensOut int
	var telemetryOutcome string
	if target.WorkspacePath != "" {
		durationSecs, tokensIn, tokensOut, telemetryOutcome = collectCompletionTelemetry(target.WorkspacePath, completeForce, outcome.Passed)
	}

	// Collect accretion delta before archival (same reason as telemetry above)
	var accretionData *events.AccretionDeltaData
	if target.WorkspacePath != "" && target.BeadsProjectDir != "" {
		accretionData = collectAccretionDelta(target.BeadsProjectDir, target.WorkspacePath)
	}

	// Export activity to ACTIVITY.json for archival
	if target.WorkspacePath != "" && !target.IsOrchestratorSession {
		sid := spawn.ReadSessionID(target.WorkspacePath)
		if sid != "" {
			if activityPath, err := activity.ExportToWorkspace(sid, target.WorkspacePath, serverURL); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to export activity: %v\n", err)
			} else if activityPath != "" {
				fmt.Printf("Exported activity: %s\n", filepath.Base(activityPath))
			}
		}
	}

	// For orchestrator sessions, export transcript before lifecycle transition
	if target.WorkspacePath != "" && target.IsOrchestratorSession {
		if err := exportOrchestratorTranscript(target.WorkspacePath, target.BeadsProjectDir, target.AgentName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to export orchestrator transcript: %v\n", err)
		}
	}

	// --- Execute lifecycle transition via LifecycleManager ---
	sessionID := spawn.ReadSessionID(target.WorkspacePath)

	lifecycleWorkspacePath := target.WorkspacePath
	if completeNoArchive {
		lifecycleWorkspacePath = ""
	}

	agentRef := agent.AgentRef{
		BeadsID:       target.BeadsID,
		WorkspaceName: target.AgentName,
		WorkspacePath: lifecycleWorkspacePath,
		SessionID:     sessionID,
		ProjectDir:    target.BeadsProjectDir,
	}

	if !target.IsClosed || target.IsOrchestratorSession || target.IsUntracked {
		if target.IsOrchestratorSession || target.IsUntracked {
			agentRef.BeadsID = ""
		}

		lm := buildLifecycleManager(target.BeadsProjectDir, serverURL, target.AgentName, target.BeadsID)
		event, err := lm.Complete(agentRef, reason)
		if err != nil {
			return false, fmt.Errorf("complete transition failed: %w", err)
		}

		// Report lifecycle effects
		for _, e := range event.Effects {
			if e.Success {
				switch e.Operation {
				case "close_issue":
					fmt.Printf("Closed beads issue: %s\n", target.BeadsID)
				case "remove_label":
					fmt.Printf("Removed orch:agent label\n")
				case "kill_window":
					fmt.Printf("Killed tmux window\n")
				case "delete_session":
					fmt.Printf("Deleted OpenCode session: %s\n", shortID(sessionID))
				case "archive":
					fmt.Printf("Archived workspace: %s\n", target.AgentName)
				}
			}
		}

		// Report warnings (non-critical failures)
		for _, w := range event.Warnings {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
		}

		// Report critical failures
		if !event.Success {
			for _, e := range event.Effects {
				if e.Critical && !e.Success {
					return false, fmt.Errorf("failed to close issue: %v", e.Error)
				}
			}
		}

		lifecycleCleanedUp = true
	}

	if target.IsOrchestratorSession {
		fmt.Printf("Completed orchestrator session: %s\n", target.AgentName)
	} else if target.IsUntracked {
		fmt.Printf("Cleaned up untracked agent: %s\n", target.Identifier)
	}
	fmt.Printf("Reason: %s\n", reason)

	// Post-lifecycle operations

	// Remove triage:ready label on successful completion
	if !target.IsClosed && !target.IsUntracked && target.BeadsID != "" {
		if err := verify.RemoveTriageReadyLabel(target.BeadsID); err != nil {
			// Non-critical
		}

		// Signal human verification to daemon
		if err := daemon.WriteVerificationSignal(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to signal human verification to daemon: %v\n", err)
		}
	}

	if completeNoArchive && target.WorkspacePath != "" {
		fmt.Println("Skipped workspace archival (--no-archive)")
	}

	// Auto-rebuild if agent committed Go changes
	if hasGoChangesInRecentCommits(target.BeadsProjectDir) {
		fmt.Println("Detected Go file changes in recent commits")
		if err := runAutoRebuild(target.BeadsProjectDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: auto-rebuild failed: %v\n", err)
		} else {
			fmt.Println("Auto-rebuild completed: make install")
			if restarted, err := restartOrchServe(target.BeadsProjectDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to restart orch serve: %v\n", err)
			} else if restarted {
				fmt.Println("Restarted orch serve")
			}
		}

		// Check for new CLI commands that may need skill documentation
		newCommands := detectNewCLICommands(target.BeadsProjectDir)
		if len(newCommands) > 0 {
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
	}

	// Check for notable changelog entries
	if !completeNoChangelogCheck {
		var agentSkill string
		if target.WorkspacePath != "" {
			agentSkill, _ = verify.ExtractSkillNameFromSpawnContext(target.WorkspacePath)
		}

		notableEntries := detectNotableChangelogEntries(target.BeadsProjectDir, agentSkill)
		if len(notableEntries) > 0 {
			fmt.Println()
			fmt.Println("┌─────────────────────────────────────────────────────────────┐")
			fmt.Println("│  ⚠️  NOTABLE ECOSYSTEM CHANGES DETECTED                      │")
			fmt.Println("├─────────────────────────────────────────────────────────────┤")
			for _, entry := range notableEntries {
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
	}

	// Log the completion with verification metadata (telemetry collected pre-lifecycle above)
	logger := events.NewLogger(events.DefaultLogPath())
	completedData := events.AgentCompletedData{
		Reason:             reason,
		Forced:             completeForce,
		Untracked:          target.IsUntracked,
		Orchestrator:       target.IsOrchestratorSession,
		VerificationPassed: outcome.Passed,
		Skill:              outcome.SkillName,
		DurationSeconds:    durationSecs,
		TokensInput:        tokensIn,
		TokensOutput:       tokensOut,
		Outcome:            telemetryOutcome,
	}
	if target.BeadsID != "" {
		completedData.BeadsID = target.BeadsID
	}
	if target.AgentName != "" {
		completedData.Workspace = target.AgentName
	}
	if completeForce && len(outcome.GatesFailed) > 0 {
		completedData.GatesBypassed = outcome.GatesFailed
	}
	if completeForce && completeReason != "" {
		completedData.ForceReason = completeReason
	}
	if err := logger.LogAgentCompleted(completedData); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Log accretion delta metrics (collected pre-lifecycle above)
	if accretionData != nil {
		accretionData.BeadsID = target.BeadsID
		accretionData.Workspace = target.AgentName
		accretionData.Skill = outcome.SkillName

		if err := logger.LogAccretionDelta(*accretionData); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log accretion delta: %v\n", err)
		}
	}

	// Invalidate orch serve cache
	invalidateServeCache()

	return lifecycleCleanedUp, nil
}

// applySkipFilters applies skip configuration to filter out bypassed gates from a
// verification result. Deduplicates the identical skip-filter logic used in both
// orchestrator and worker verification paths.
func applySkipFilters(result *verify.VerificationResult, skipConfig verify.SkipConfig, beadsID, agentName, skillName string) {
	var filteredErrors []string
	var filteredGates []string
	var skippedGatesFound []string

	for _, gate := range result.GatesFailed {
		if skipConfig.ShouldSkipGate(gate) {
			skippedGatesFound = append(skippedGatesFound, gate)
			fmt.Printf("⚠️  Bypassing gate: %s (reason: %s)\n", gate, skipConfig.Reason)
		} else {
			filteredGates = append(filteredGates, gate)
		}
	}

	// Filter errors - keep only those not related to skipped gates
	for _, e := range result.Errors {
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

	// Log bypass events for skipped gates
	if len(skippedGatesFound) > 0 {
		logSkipEvents(skipConfig, beadsID, agentName, skillName)
	}

	// Update result with filtered data
	result.GatesFailed = filteredGates
	result.Errors = filteredErrors
	result.Passed = len(filteredGates) == 0
}
