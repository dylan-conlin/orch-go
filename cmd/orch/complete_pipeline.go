// Package main provides the completion pipeline phases for decomposing runComplete()
// into typed, testable phase functions.
//
// Pipeline: resolveCompletionTarget → executeVerificationGates → runCompletionAdvisories → executeLifecycleTransition
//
// File layout:
//   complete_pipeline.go      - Types, resolveCompletionTarget, runCompletionAdvisories
//   complete_verification.go  - executeVerificationGates, applySkipFilters
//   complete_lifecycle.go     - executeLifecycleTransition
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/checkpoint"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/identity"
	"github.com/dylan-conlin/orch-go/pkg/orch"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"golang.org/x/term"
)

// CompletionTarget holds the resolved target of a completion operation.
type CompletionTarget struct {
	Identifier            string // Original command-line identifier
	WorkspacePath         string
	AgentName             string
	BeadsID               string
	BeadsProjectDir       string // Where the beads issue lives (for bd show/close/comments)
	WorkProjectDir        string // Where the agent did its work (for build, verification, hotspot)
	IsOrchestratorSession bool
	Issue                 *verify.Issue
	IsClosed              bool
	ReviewTier            string // Effective review tier (auto/scan/review/deep)
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
// Includes pipeline timing for harness measurement.
type AdvisoryResults struct {
	PipelineTiming  []events.PipelineStepTiming // Per-step timing
	PipelineTotalMs int                         // Total advisory pipeline wall-clock ms
}

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

		// Fallback: search across all known projects for cross-repo workspaces.
		// This handles the case where an agent was spawned with --workdir targeting
		// a different project (e.g., orch-go issue but workspace in kb-cli).
		// The beads ID prefix matches CWD so crossProjectDir isn't set above,
		// but the workspace lives in the target project.
		if target.WorkspacePath == "" {
			if wsPath, name := findWorkspaceByBeadsIDAcrossProjects(target.BeadsID); wsPath != "" {
				target.WorkspacePath = wsPath
				target.AgentName = name
				fmt.Printf("Found cross-repo workspace in %s\n", filepath.Dir(filepath.Dir(filepath.Dir(wsPath))))
			}
		}
	}

	// Determine BeadsProjectDir — where the beads issue lives.
	// Derived from beads ID prefix, independent of workspace location.
	// This fixes cross-project completion where issue and workspace live in different repos.
	if target.BeadsID != "" {
		projectName := extractProjectFromBeadsID(target.BeadsID)
		if projectName != "" {
			if foundDir := findProjectDirByName(projectName); foundDir != "" {
				target.BeadsProjectDir = foundDir
				if foundDir != currentDir {
					fmt.Printf("Beads project (from ID prefix): %s\n", filepath.Base(foundDir))
				}
			} else {
				// ID prefix doesn't resolve to a known project, fall back to CWD
				target.BeadsProjectDir = currentDir
			}
		} else {
			target.BeadsProjectDir = currentDir
		}
	} else {
		target.BeadsProjectDir = currentDir
	}

	// Determine WorkProjectDir — where the agent did its work.
	// Priority: --workdir override > workspace manifest > SPAWN_CONTEXT.md > identity registry > BeadsProjectDir
	if workdir != "" {
		target.WorkProjectDir, err = identity.ResolveProject(target.BeadsID, workdir)
		if err != nil {
			return target, fmt.Errorf("failed to resolve workdir: %w", err)
		}
		fmt.Printf("Using explicit workdir: %s\n", target.WorkProjectDir)
	} else if target.WorkspacePath != "" {
		// Try workspace manifest first (ground truth for spawned agents)
		if manifest, mErr := spawn.ReadAgentManifest(target.WorkspacePath); mErr == nil && manifest.ProjectDir != "" {
			target.WorkProjectDir = filepath.Clean(manifest.ProjectDir)
			if target.WorkProjectDir != currentDir {
				fmt.Printf("Work project (from manifest): %s\n", filepath.Base(target.WorkProjectDir))
			}
		} else {
			// Fallback: extract from SPAWN_CONTEXT.md
			projectDirFromWorkspace := extractProjectDirFromWorkspace(target.WorkspacePath)
			if projectDirFromWorkspace != "" {
				target.WorkProjectDir = projectDirFromWorkspace
				if target.WorkProjectDir != currentDir {
					fmt.Printf("Work project (from SPAWN_CONTEXT): %s\n", filepath.Base(target.WorkProjectDir))
				}
			}
		}
	} else if target.BeadsID != "" {
		// No workspace and no --workdir — try identity registry
		resolved, rErr := identity.ResolveProject(target.BeadsID, "")
		if rErr == nil && resolved != currentDir {
			target.WorkProjectDir = resolved
			fmt.Printf("Auto-resolved work project: %s\n", filepath.Base(resolved))
		}
	}

	// Default: WorkProjectDir falls back to BeadsProjectDir (same-project case)
	if target.WorkProjectDir == "" {
		target.WorkProjectDir = target.BeadsProjectDir
	}

	// Log cross-project split when beads and work dirs differ
	if target.BeadsProjectDir != target.WorkProjectDir {
		fmt.Printf("Cross-project: beads in %s, work in %s\n",
			filepath.Base(target.BeadsProjectDir), filepath.Base(target.WorkProjectDir))
	}

	// Verify the beads issue exists (orchestrator sessions and agents without beads ID skip this)
	if target.IsOrchestratorSession {
		fmt.Printf("Note: %s is an orchestrator session (no beads tracking)\n", target.AgentName)
		target.IsClosed = false
	} else if target.BeadsID == "" {
		fmt.Printf("Note: %s has no beads ID\n", identifier)
		target.IsClosed = false
	} else {
		issue, err := verify.GetIssue(target.BeadsID, target.BeadsProjectDir)
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
	}

	return target, nil
}

// runCompletionAdvisories runs advisory gates: discovered work, probe verdicts,
// architectural choices, knowledge maintenance, explain-back, and verification checklist.
//
// Review tier controls which advisories run:
//   - auto:   auto-file discovered work (no prompt), skip knowledge maintenance + explain-back
//   - scan:   print SYNTHESIS TLDR, interactive discovered work, skip knowledge maintenance + explain-back
//   - review: full ceremony (current behavior)
//   - deep:   full ceremony (current behavior)
func runCompletionAdvisories(target CompletionTarget, outcome VerificationOutcome, skipConfig verify.SkipConfig) (AdvisoryResults, error) {
	var advisories AdvisoryResults

	isAutoTier := target.ReviewTier == "auto"
	isScanTier := target.ReviewTier == "scan"
	isLightReview := isAutoTier || isScanTier

	// For scan tier: print SYNTHESIS TLDR upfront for quick context
	if isScanTier && target.WorkspacePath != "" {
		synthesis, err := verify.ParseSynthesis(target.WorkspacePath)
		if err == nil && synthesis != nil && synthesis.TLDR != "" {
			fmt.Println("\n--- SYNTHESIS TLDR ---")
			fmt.Println(synthesis.TLDR)
			fmt.Println("---------------------")
		}
	}

	// Gate completion on discovered work disposition
	if target.WorkspacePath != "" && !completeForce {
		synthesis, err := verify.ParseSynthesis(target.WorkspacePath)
		if err == nil && synthesis != nil {
			items := verify.CollectDiscoveredWork(synthesis)

			if len(items) > 0 {
				if isAutoTier {
					// Auto tier: file all discovered work without prompting
					fmt.Printf("\n--- Discovered Work (auto-filed, review tier: auto) ---\n")
					createdCount := 0
					for _, item := range items {
						title := cleanDiscoveredWorkTitle(item.Description)
						issue, err := beads.FallbackCreate(title, "", "task", 2, []string{"triage:ready"}, target.BeadsProjectDir)
						if err != nil {
							fmt.Fprintf(os.Stderr, "  Failed to create issue: %v\n", err)
						} else {
							fmt.Printf("  Auto-filed: %s - %s\n", issue.ID, title)
							createdCount++
						}
					}
					if createdCount > 0 {
						fmt.Printf("✓ Auto-filed %d follow-up issue(s)\n", createdCount)
					}
					fmt.Println("---------------------------------")
				} else {
					// scan/review/deep: interactive disposition
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
							title := cleanDiscoveredWorkTitle(item.Description)

							issue, err := beads.FallbackCreate(title, "", "task", 2, []string{"triage:ready"}, target.BeadsProjectDir)
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
	}

	// Surface probe verdicts for orchestrator review
	if target.WorkspacePath != "" {
		probeVerdicts := verify.FindProbesForWorkspace(target.WorkspacePath, target.WorkProjectDir)
		if len(probeVerdicts) > 0 {
			fmt.Print(verify.FormatProbeVerdicts(probeVerdicts))
		}
	}

	// Surface friction reports for orchestrator review
	if target.BeadsID != "" && !target.IsOrchestratorSession {
		frictionItems := verify.FetchAndParseFriction(target.BeadsID, target.BeadsProjectDir)
		if len(frictionItems) > 0 {
			fmt.Print(verify.FormatFrictionAdvisory(frictionItems))
		} else if !isLightReview {
			fmt.Println("\n\u26A0\uFE0F  No friction reported (expected Friction: comment)")
		}
	}

	// Surface architectural choices for orchestrator review
	if target.WorkspacePath != "" && !target.IsOrchestratorSession {
		if choicesOutput := verify.FormatArchitecturalChoicesForCompletion(target.WorkspacePath); choicesOutput != "" {
			fmt.Print(choicesOutput)
		}
	}

	// Knowledge maintenance step (Touchpoint 1: Completion Review)
	// Skipped for auto/scan tiers — these are lightweight completions
	if !target.IsOrchestratorSession && target.BeadsID != "" && !completeForce && !isLightReview {
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
	// Skipped for auto/scan tiers — checkpoint gates handle this at the verification level
	if !isLightReview {
		priorGate1, _ := checkpoint.HasGate1Checkpoint(target.BeadsID)
		if completeExplain != "" || !priorGate1 {
			if err := orch.RunExplainBackGate(
				target.BeadsID,
				completeForce,
				skipConfig.ExplainBack,
				skipConfig.Reason,
				target.IsOrchestratorSession,
				completeExplain,
				completeVerified,
				os.Stdout,
			); err != nil {
				return advisories, err
			}
		}

		// Record gate2 checkpoint if --verified flag is set and explain-back gate didn't run
		if completeVerified && !target.IsOrchestratorSession && target.BeadsID != "" {
			hasGate2, _ := checkpoint.HasGate2Checkpoint(target.BeadsID)
			if !hasGate2 && priorGate1 && completeExplain == "" {
				if err := orch.RecordGate2Checkpoint(target.BeadsID, os.Stdout); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to record gate2 checkpoint: %v\n", err)
				}
			}
		}
	}

	// Surface verification checklist before closing
	if outcome.ResultSet && target.BeadsID != "" {
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

		// Use target.ReviewTier directly instead of re-reading from workspace
		trustTier := ComputeTrustTier(target.ReviewTier, skipConfig, completeForce)
		printVerificationChecklist(checklist, trustTier)
	}

	// Surface hotspot advisory for modified files (informational, not a gate)
	pipelineStart := time.Now()
	{
		step := events.PipelineStepTiming{Name: "hotspot"}
		if target.WorkProjectDir == "" || target.IsOrchestratorSession {
			step.Skipped = true
			if target.IsOrchestratorSession {
				step.SkipReason = "orchestrator"
			} else {
				step.SkipReason = "no_project_dir"
			}
		} else {
			t0 := time.Now()
			if advisory := RunHotspotAdvisoryForCompletion(target.WorkProjectDir, target.WorkspacePath); advisory != "" {
				fmt.Print(advisory)
			}
			step.DurationMs = int(time.Since(t0).Milliseconds())
		}
		advisories.PipelineTiming = append(advisories.PipelineTiming, step)
	}

	// Surface duplication advisory for modified files (informational, not a gate)
	{
		step := events.PipelineStepTiming{Name: "duplication"}
		if target.WorkProjectDir == "" || target.IsOrchestratorSession {
			step.Skipped = true
			if target.IsOrchestratorSession {
				step.SkipReason = "orchestrator"
			} else {
				step.SkipReason = "no_project_dir"
			}
		} else {
			t0 := time.Now()
			if advisory := RunDuplicationAdvisoryForCompletion(target.WorkProjectDir, target.WorkspacePath); advisory != "" {
				fmt.Print(advisory)
			}
			step.DurationMs = int(time.Since(t0).Milliseconds())
		}
		advisories.PipelineTiming = append(advisories.PipelineTiming, step)
	}

	// Surface model-impact advisory (informational, not a gate)
	{
		step := events.PipelineStepTiming{Name: "model_impact"}
		if target.WorkProjectDir == "" || target.WorkspacePath == "" || target.IsOrchestratorSession {
			step.Skipped = true
			if target.IsOrchestratorSession {
				step.SkipReason = "orchestrator"
			} else {
				step.SkipReason = "no_project_dir"
			}
		} else {
			t0 := time.Now()
			if advisory := RunModelImpactAdvisory(target.WorkProjectDir, target.WorkspacePath); advisory != "" {
				fmt.Print(advisory)
			}
			step.DurationMs = int(time.Since(t0).Milliseconds())
		}
		advisories.PipelineTiming = append(advisories.PipelineTiming, step)
	}
	advisories.PipelineTotalMs = int(time.Since(pipelineStart).Milliseconds())

	// Surface synthesis checkpoint (informational, not a gate)
	if !target.IsOrchestratorSession && target.BeadsID != "" {
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
		if err := UpdateHandoffAfterComplete(target.WorkProjectDir, target.AgentName, target.BeadsID, outcome.SkillName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update session handoff: %v\n", err)
		}
	}

	return advisories, nil
}

// cleanDiscoveredWorkTitle cleans up a discovered work item description for use as an issue title.
func cleanDiscoveredWorkTitle(description string) string {
	title := strings.TrimPrefix(description, "- ")
	title = strings.TrimPrefix(title, "* ")
	if len(title) > 3 && title[0] >= '0' && title[0] <= '9' && (title[1] == '.' || (title[1] >= '0' && title[1] <= '9' && title[2] == '.')) {
		if idx := strings.Index(title, ". "); idx != -1 && idx < 4 {
			title = title[idx+2:]
		}
	}
	return title
}
