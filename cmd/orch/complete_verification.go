// Package main provides the verification gate phase of the completion pipeline.
// Extracted from complete_pipeline.go for cohesion: verification gates + skip filters.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/checkpoint"
	"github.com/dylan-conlin/orch-go/pkg/completion"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"golang.org/x/term"
)

// executeVerificationGates runs checkpoint gates, completion verification, skip
// filtering, and liveness checks. Returns a VerificationOutcome with results.
func executeVerificationGates(target CompletionTarget, skipConfig verify.SkipConfig) (VerificationOutcome, error) {
	var outcome VerificationOutcome
	outcome.Passed = true

	// Checkpoint verification gate (Verifiability-first enforcement)
	// auto/scan review tiers skip checkpoint gates entirely
	var checkpointErrors []string
	var checkpointGatesFailed []string
	skipCheckpoints := target.ReviewTier == "auto" || target.ReviewTier == "scan"
	if !completeForce && target.Issue != nil && !skipCheckpoints {
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
	} else if skipCheckpoints && !completeForce && target.Issue != nil {
		fmt.Printf("Checkpoint gates skipped (review tier: %s)\n", target.ReviewTier)
	}

	// Auto-create implementation issue for architect completions BEFORE gates.
	// This must run before VerifyCompletionFull so the architect_handoff gate
	// can verify the implementation issue exists. Runs regardless of --force
	// since implementation issues should always be created for actionable recommendations.
	if !target.IsOrchestratorSession && target.WorkspacePath != "" && target.BeadsID != "" {
		skillName, _ := verify.ExtractSkillNameFromSpawnContext(target.WorkspacePath)
		maybeAutoCreateImplementationIssue(skillName, target.BeadsID, target.WorkspacePath)
	}

	// If --approve flag is set, add approval comment BEFORE verification
	if completeApprove && target.BeadsID != "" {
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
					Workspace:         target.AgentName,
					GatesFailed:       result.GatesFailed,
					Errors:            result.Errors,
					Skill:             outcome.SkillName,
					VerificationLevel: result.VerifyLevel,
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
		} else if target.BeadsID != "" {
			// Regular agents use beads phase verification
			if target.WorkspacePath != "" {
				fmt.Printf("Workspace: %s\n", target.AgentName)
			}

			result, err := verify.VerifyCompletionFull(target.BeadsID, target.WorkspacePath, target.WorkProjectDir, "", serverURL)
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
			if target.WorkspacePath != "" && target.WorkProjectDir != "" {
				matches, err := verify.FindModelReferencesForModifiedFiles(target.WorkspacePath, target.WorkProjectDir)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to check model code references: %v\n", err)
				} else if note := verify.FormatModelReferenceNote(matches); note != "" {
					fmt.Println(note)
				}
			}

			// Artifact gate: validate COMPLETION.yaml per work type (V1+)
			// Scan-tier skills (investigation, probe, research, audit) are exempt —
			// they produce knowledge artifacts, not COMPLETION.yaml.
			isScanTierForArtifact := target.ReviewTier == spawn.ReviewScan || target.ReviewTier == spawn.ReviewAuto
			if target.WorkspacePath != "" && target.Issue != nil &&
				!isScanTierForArtifact &&
				verify.ShouldRunGate(result.VerifyLevel, verify.GateArtifact) &&
				!skipConfig.ShouldSkipGate(verify.GateArtifact) {
				result.GatesRun = append(result.GatesRun, verify.GateArtifact)
				artResult := completion.CheckArtifact(target.WorkspacePath, target.Issue.IssueType)
				if !artResult.Passed {
					result.Passed = false
					result.Errors = append(result.Errors, artResult.Errors...)
					result.GatesFailed = append(result.GatesFailed, verify.GateArtifact)
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
					BeadsID:           target.BeadsID,
					Workspace:         target.AgentName,
					GatesFailed:       result.GatesFailed,
					Errors:            result.Errors,
					Skill:             outcome.SkillName,
					VerificationLevel: result.VerifyLevel,
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
			fmt.Println("Skipping phase verification (no beads ID)")
		}
	} else {
		// --force was used, run verification anyway to capture which gates would have failed
		if !target.IsOrchestratorSession && target.BeadsID != "" {
			result, err := verify.VerifyCompletionFull(target.BeadsID, target.WorkspacePath, target.WorkProjectDir, "", serverURL)
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

	// Check liveness before closing - warn if agent appears still running.
	// Uses phase-based liveness (decision: .kb/decisions/2026-02-26-phase-based-liveness-over-tmux-as-state.md)
	if !completeForce && target.BeadsID != "" && !target.IsOrchestratorSession {
		comments, _ := verify.GetComments(target.BeadsID, target.BeadsProjectDir)
		spawnTime := readSpawnTimeFromWorkspace(target.WorkspacePath)

		liveness := verify.VerifyLiveness(verify.LivenessInput{
			Comments:  comments,
			SpawnTime: spawnTime,
			Now:       time.Now(),
		})

		if liveness.IsAlive() {
			fmt.Fprintf(os.Stderr, "⚠️  %s\n", liveness.Warning())

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

	// DISABLED: Reproduction verification gate (Jan 4, 2026)
	_ = completeSkipReproCheck
	_ = completeSkipReproReason

	return outcome, nil
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
		logSkipEvents(skipConfig, beadsID, agentName, skillName, result.VerifyLevel)
	}

	// Update result with filtered data
	result.GatesFailed = filteredGates
	result.Errors = filteredErrors
	result.Passed = len(filteredGates) == 0
}

// readSpawnTimeFromWorkspace reads the spawn time from the agent manifest.
// Returns zero time if workspace is empty or manifest is unreadable.
func readSpawnTimeFromWorkspace(workspacePath string) time.Time {
	if workspacePath == "" {
		return time.Time{}
	}
	manifest := spawn.ReadAgentManifestWithFallback(workspacePath)
	return manifest.ParseSpawnTime()
}
