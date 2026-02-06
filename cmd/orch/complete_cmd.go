// Package main provides the complete command for completing agents and closing beads issues.
// Extracted from main.go as part of the main.go refactoring (Phase 3).
//
// Related files:
//   - complete_verify.go:  SkipConfig type and verification skip logic
//   - complete_actions.go: Post-completion actions (archival, transcript, telemetry, cache)
//   - complete_helpers.go: Changelog detection, auto-rebuild, CLI command detection, display
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
	"github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	// Complete command flags
	completeForce            bool
	completeReason           string
	completeApprove          bool
	completeWorkdir          string
	completeNoChangelogCheck bool
	completeSkipReproCheck   bool
	completeSkipReproReason  string
	completeNoArchive        bool
	completeForceCloseEpic   bool // Force close epic even with open children
	completeAutoCloseParent  bool // Auto-close parent epic when all children complete

	// Targeted skip flags (replace blanket --force)
	// Each requires completeSkipReason to be set (min 10 chars)
	completeSkipTestEvidence    bool
	completeSkipVisual          bool
	completeSkipGitDiff         bool
	completeSkipSynthesis       bool
	completeSkipBuild           bool
	completeSkipConstraint      bool
	completeSkipPhaseGate       bool
	completeSkipSkillOutput     bool
	completeSkipDecisionPatch   bool
	completeSkipPhaseComplete   bool
	completeSkipHandoffContent  bool
	completeSkipDashboardHealth bool
	completeSkipReason          string // Required for all --skip-* flags (min 10 chars)
)

var completeCmd = &cobra.Command{
	Use:   "complete [beads-id-or-workspace]",
	Short: "Complete an agent and close the beads issue",
	Long: `Complete an agent's work by verifying Phase: Complete and closing the beads issue.

Checks that the agent has reported "Phase: Complete" via beads comments before
closing the issue. After successful completion, the workspace is automatically
archived to .orch/workspace/archived/ for cleanup. Use --no-archive to opt out.

VERIFICATION GATES:
The following gates are checked before completion:
  - phase_complete:       Agent reported "Phase: Complete"
  - synthesis:            SYNTHESIS.md exists (full tier only)
  - test_evidence:        Test execution evidence in beads comments
  - visual_verification:  Visual verification for web/ changes
  - git_diff:             Git changes match SYNTHESIS.md claims
  - build:                Project builds successfully
  - constraint:           Skill constraints satisfied
  - phase_gate:           Required skill phases completed
  - skill_output:         Required skill outputs exist
  - decision_patch_limit: Decision patch count not exceeded
  - handoff_content:      SYNTHESIS.md has actual content (orchestrator only)
  - dashboard_health:     Dashboard API endpoints healthy for web/ or serve_*.go changes

TARGETED SKIP FLAGS:
Use --skip-{gate} with --skip-reason to bypass specific gates:
  --skip-test-evidence     Skip test evidence gate
  --skip-visual            Skip visual verification gate
  --skip-git-diff          Skip git diff verification gate
  --skip-synthesis         Skip SYNTHESIS.md gate
  --skip-build             Skip build verification gate
  --skip-constraint        Skip constraint verification gate
  --skip-phase-gate        Skip phase gate verification
  --skip-skill-output      Skip skill output verification gate
  --skip-decision-patch    Skip decision patch count gate
  --skip-phase-complete    Skip Phase: Complete gate
  --skip-handoff-content   Skip handoff content validation (orchestrator only)
  --skip-dashboard-health  Skip dashboard health check for web/ or serve_*.go changes

Each --skip-* flag requires --skip-reason with a minimum of 10 characters
explaining why the gate is being bypassed. Bypasses are logged for audit.

DEPRECATION: --force is deprecated. Use targeted --skip-* flags instead.
Using --force will show a deprecation warning.

For orchestrator sessions (spawned with orchestrator or meta-orchestrator skill),
the argument is the workspace name instead of beads ID. Orchestrators use
SYNTHESIS.md as completion signal instead of Phase: Complete.

For agents that modified web/ files (UI tasks), --approve is required to explicitly
confirm human review of the visual changes. This prevents agents from self-certifying
UI correctness.

For cross-project completion (agents spawned with --workdir in another project),
the command auto-detects the project from the workspace's SPAWN_CONTEXT.md.
Use --workdir as explicit override when auto-detection fails.

EPIC PROTECTION:
For epic issues, completion is blocked if the epic has open children (tasks, bugs, etc.)
that are not yet closed. This prevents accidentally closing an epic while work remains.
Use --force-close-epic to override this protection.

Examples:
  orch-go complete proj-123
  orch-go complete proj-123 --reason "All tests passing"
  orch-go complete proj-123 --approve       # Approve UI changes after visual review
  orch-go complete proj-123 --skip-test-evidence --skip-reason "Tests run in CI"
  orch-go complete proj-123 --skip-git-diff --skip-synthesis --skip-reason "Docs-only change"
  orch-go complete kb-cli-123 --workdir ~/projects/kb-cli  # Cross-project completion

  # Orchestrator session completion (by workspace name)
  orch-go complete og-orch-goal-04jan       # Complete orchestrator session

  # Epic with open children (will fail unless forced)
  orch-go complete proj-epic-123 --force-close-epic

  # Deprecated (shows warning):
  orch-go complete proj-123 --force         # Use targeted --skip-* flags instead`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		identifier := args[0]
		return runComplete(identifier, completeWorkdir)
	},
}

func init() {
	completeCmd.Flags().BoolVarP(&completeForce, "force", "f", false, "DEPRECATED: Skip all verification (use targeted --skip-* flags instead)")
	completeCmd.Flags().StringVarP(&completeReason, "reason", "r", "", "Reason for closing (default: uses phase summary)")
	completeCmd.Flags().BoolVar(&completeApprove, "approve", false, "Approve visual changes for UI tasks (adds approval comment)")
	completeCmd.Flags().StringVar(&completeWorkdir, "workdir", "", "Target project directory (for cross-project completion)")
	completeCmd.Flags().StringVar(&completeWorkdir, "project", "", "Alias for --workdir")
	completeCmd.Flags().MarkHidden("project")
	completeCmd.Flags().BoolVar(&completeNoChangelogCheck, "no-changelog-check", false, "Skip changelog detection for notable changes")
	completeCmd.Flags().BoolVar(&completeSkipReproCheck, "skip-repro-check", false, "Skip reproduction verification for bug issues (requires --reason)")
	completeCmd.Flags().StringVar(&completeSkipReproReason, "skip-repro-reason", "", "Reason for skipping reproduction verification")
	completeCmd.Flags().BoolVar(&completeNoArchive, "no-archive", false, "Skip automatic workspace archival after completion")
	completeCmd.Flags().BoolVar(&completeForceCloseEpic, "force-close-epic", false, "Force close epic even if it has open children (use with caution)")
	completeCmd.Flags().BoolVar(&completeAutoCloseParent, "auto-close-parent", false, "Automatically close parent epic when completing the last open child")

	// Targeted skip flags - each bypasses a specific verification gate
	completeCmd.Flags().BoolVar(&completeSkipTestEvidence, "skip-test-evidence", false, "Skip test execution evidence gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipVisual, "skip-visual", false, "Skip visual verification gate for web/ changes (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipGitDiff, "skip-git-diff", false, "Skip git diff verification gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipSynthesis, "skip-synthesis", false, "Skip SYNTHESIS.md verification gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipBuild, "skip-build", false, "Skip project build verification gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipConstraint, "skip-constraint", false, "Skip constraint verification gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipPhaseGate, "skip-phase-gate", false, "Skip phase gate verification (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipSkillOutput, "skip-skill-output", false, "Skip skill output verification gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipDecisionPatch, "skip-decision-patch", false, "Skip decision patch count verification gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipPhaseComplete, "skip-phase-complete", false, "Skip Phase: Complete verification gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipHandoffContent, "skip-handoff-content", false, "Skip handoff content validation gate for orchestrators (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipDashboardHealth, "skip-dashboard-health", false, "Skip dashboard health check gate for web/ or serve_*.go changes (requires --skip-reason)")
	completeCmd.Flags().StringVar(&completeSkipReason, "skip-reason", "", "Reason for skip (required for all --skip-* flags, min 10 chars)")
}

func runComplete(identifier, workdir string) error {
	// Validate skip flags before doing anything else
	skipConfig := getSkipConfig()
	if err := validateSkipFlags(skipConfig); err != nil {
		return err
	}

	// Show deprecation warning for --force
	if completeForce {
		fmt.Fprintln(os.Stderr, "⚠️  DEPRECATED: --force is deprecated. Use targeted --skip-* flags instead.")
		fmt.Fprintln(os.Stderr, "   Example: --skip-test-evidence --skip-reason \"Tests run in CI\"")
		fmt.Fprintln(os.Stderr, "   This flag will be removed in a future version.")
		fmt.Fprintln(os.Stderr)
	}

	// Get current directory as base project dir
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Determine if identifier is a workspace name or beads ID.
	// Workspace names: og-feat-description-21dec, og-orch-goal-04jan
	// Beads IDs: orch-go-kypi, kb-cli-abc1
	//
	// Strategy (priority order):
	// 1. Check orchestrator session registry FIRST (handles cross-project orchestrators)
	// 2. Try to find workspace by name in current directory
	// 3. Only fall back to beads ID lookup for worker sessions
	//
	// This prevents orchestrator workspace names from being misinterpreted as beads IDs.
	var workspacePath, agentName string
	var beadsID string
	var isOrchestratorSession bool

	// Step 1: Check orchestrator session registry FIRST
	// This is the authoritative source for orchestrator sessions and handles cross-project cases.
	registry := session.NewRegistry("")
	if orchSession, err := registry.Get(identifier); err == nil {
		// Found in registry - this is an orchestrator session
		isOrchestratorSession = true
		agentName = orchSession.WorkspaceName
		fmt.Printf("Orchestrator session (from registry): %s\n", agentName)

		// Use the registry's ProjectDir to find the workspace
		workspacePath = findWorkspaceByName(orchSession.ProjectDir, agentName)
		if workspacePath == "" {
			// Workspace not found in expected location - might have been moved or deleted
			fmt.Fprintf(os.Stderr, "Warning: Workspace %s not found in %s\n", agentName, orchSession.ProjectDir)
		}
	}

	// Step 2: Try direct workspace name lookup in current directory (if not found in registry)
	if workspacePath == "" && !isOrchestratorSession {
		directWorkspacePath := findWorkspaceByName(currentDir, identifier)
		if directWorkspacePath != "" {
			workspacePath = directWorkspacePath
			agentName = identifier
			// Check if this is an orchestrator workspace (no beads tracking)
			if isOrchestratorWorkspace(workspacePath) {
				isOrchestratorSession = true
				fmt.Printf("Orchestrator session: %s\n", agentName)
			} else {
				// Non-orchestrator workspace found by name - read beads ID from .beads_id file
				beadsIDPath := filepath.Join(workspacePath, ".beads_id")
				if content, err := os.ReadFile(beadsIDPath); err == nil {
					beadsID = strings.TrimSpace(string(content))
				}
			}
		}
	}

	// Step 3: If no workspace match and not an orchestrator session, treat identifier as beads ID
	// This is the fallback for worker sessions identified by beads ID.
	if workspacePath == "" && !isOrchestratorSession {
		// Auto-detect cross-project agents BEFORE resolution
		// If the identifier looks like a cross-project beads ID (e.g., "pw-ed7h" when in orch-go),
		// try to find that project's directory and set beads.DefaultDir before resolution.
		var crossProjectDir string
		if !strings.Contains(identifier, filepath.Base(currentDir)) {
			// Identifier might be from a different project - try to extract project name
			projectName := extractProjectFromBeadsID(identifier)
			if projectName != "" && projectName != filepath.Base(currentDir) {
				// Try to find the project directory
				if foundDir := findProjectDirByName(projectName); foundDir != "" {
					crossProjectDir = foundDir
					beads.DefaultDir = crossProjectDir
					fmt.Printf("Auto-detected cross-project from beads ID: %s\n", filepath.Base(crossProjectDir))
				}
			}
		}

		// Resolve short beads ID to full ID (e.g., "qdaa" -> "orch-go-qdaa")
		// Now uses correct project's beads database if cross-project detected
		resolvedID, err := resolveShortBeadsID(identifier)
		if err != nil {
			return fmt.Errorf("failed to resolve beads ID: %w", err)
		}
		beadsID = resolvedID

		// Find workspace by beads ID
		// For cross-project agents, look in the detected project directory
		searchDir := currentDir
		if crossProjectDir != "" {
			searchDir = crossProjectDir
		}
		workspacePath, agentName = findWorkspaceByBeadsID(searchDir, beadsID)
	}

	// Determine beads project directory using the shared helper
	projectResult, err := resolveProjectDir(workdir, workspacePath, currentDir)
	if err != nil {
		return err
	}
	beadsProjectDir := projectResult.ProjectDir

	// Log resolution source for transparency
	switch projectResult.Source {
	case "workdir":
		fmt.Printf("Using explicit workdir: %s\n", beadsProjectDir)
	case "workspace":
		fmt.Printf("Auto-detected cross-project: %s\n", filepath.Base(beadsProjectDir))
	}

	// Set beads.DefaultDir for cross-project operations BEFORE any beads operations
	projectResult.SetBeadsDefaultDir()

	// Check if this is an untracked agent (no beads issue exists)
	// Orchestrator sessions are implicitly untracked (they skip beads entirely)
	isUntracked := isOrchestratorSession || (beadsID != "" && isUntrackedBeadsID(beadsID)) || beadsID == ""

	// For tracked agents, verify the beads issue exists
	var issue *verify.Issue
	var isClosed bool
	if !isUntracked {
		var err error
		issue, err = verify.GetIssue(beadsID)
		if err != nil {
			// Provide helpful error message for cross-project issues
			projectName := filepath.Base(beadsProjectDir)
			issuePrefix := strings.Split(beadsID, "-")[0]
			if len(strings.Split(beadsID, "-")) > 1 {
				issuePrefix = strings.Join(strings.Split(beadsID, "-")[:len(strings.Split(beadsID, "-"))-1], "-")
			}
			if issuePrefix != projectName {
				return fmt.Errorf("failed to get beads issue %s: %w\n\nHint: The issue ID suggests it belongs to project '%s', but you're in '%s'.\nTry: orch complete %s --workdir ~/path/to/%s", beadsID, err, issuePrefix, projectName, beadsID, issuePrefix)
			}
			return fmt.Errorf("failed to get beads issue: %w", err)
		}

		// Check if already closed
		isClosed = issue.Status == "closed"
		if isClosed {
			fmt.Printf("Issue %s is already closed in beads\n", beadsID)
		}
	} else if isOrchestratorSession {
		fmt.Printf("Note: %s is an orchestrator session (no beads tracking)\n", agentName)
		// Orchestrator sessions are treated as not closed (we'll clean them up)
		isClosed = false
	} else {
		fmt.Printf("Note: %s is an untracked agent (no beads issue)\n", identifier)
		// Untracked agents are treated as not closed (we'll clean them up)
		isClosed = false
	}

	// If --approve flag is set, add approval comment BEFORE verification
	// This ensures the visual verification gate sees the approval
	// Skip for untracked agents (no beads issue to comment on)
	if completeApprove && !isUntracked {
		approvalComment := "✅ APPROVED - Visual changes reviewed and approved by orchestrator"
		if err := addApprovalComment(beadsID, approvalComment); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to add approval comment: %v\n", err)
			// Continue anyway - the approval might already exist or we can fallback
		} else {
			fmt.Printf("Added approval: %s\n", approvalComment)
		}
	}

	// Track verification state for event emission
	var verificationPassed bool = true
	var gatesFailed []string
	var skillName string

	// Auto-rebuild Go binaries BEFORE verification
	// This ensures verification runs against fresh binaries when agents modify Go code
	// Handles both same-project and cross-project scenarios
	rebuildGoProjectsIfNeeded(beadsProjectDir, workspacePath)

	// Check if this is a question entity (strategic node, not agent work)
	// Questions don't have agents, so they skip Phase: Complete requirement
	isQuestion := issue != nil && issue.IssueType == "question"

	// Verify completion status
	// - For orchestrator sessions: check SYNTHESIS.md exists AND has content
	// - For question entities: skip Phase: Complete (strategic nodes, not agent work)
	// - For regular agents: check Phase: Complete via beads comments
	// - Skip flags allow targeted bypass of specific gates
	if !completeForce {
		if isQuestion {
			// Question entities are strategic nodes - they're answered through
			// investigations, discussions, etc., not by agents reporting Phase: Complete.
			// Just close them without verification.
			fmt.Printf("Question entity: %s (skipping Phase: Complete - strategic node)\n", beadsID)
		} else if isOrchestratorSession {
			// Orchestrator sessions use SYNTHESIS.md as completion signal
			// Use full verification which includes content validation
			if workspacePath != "" {
				fmt.Printf("Workspace: %s\n", agentName)
			}

			result := verify.VerifyOrchestratorCompletion(workspacePath)
			skillName = result.Skill

			// Apply skip config to filter out bypassed gates
			if skipConfig.hasAnySkip() && !result.Passed {
				var filteredErrors []string
				var filteredGates []string
				var skippedGatesFound []string

				for _, gate := range result.GatesFailed {
					if skipConfig.shouldSkipGate(gate) {
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
						// Match error messages to gates
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
					logSkipEvents(skipConfig, "", agentName, skillName)
				}

				// Persist build skip memory when build gate is bypassed (orchestrator path)
				for _, gate := range skippedGatesFound {
					if gate == verify.GateBuild {
						if err := verify.WriteBuildSkipMemory(beadsProjectDir, skipConfig.Reason, agentName); err != nil {
							fmt.Fprintf(os.Stderr, "Warning: failed to persist build skip memory: %v\n", err)
						} else {
							fmt.Printf("Build skip memory saved (auto-skips build for subsequent completions, expires in %v)\n", verify.BuildSkipDuration)
						}
						break
					}
				}

				// Update result with filtered data
				result.GatesFailed = filteredGates
				result.Errors = filteredErrors
				result.Passed = len(filteredGates) == 0
			}

			if !result.Passed {
				verificationPassed = false
				gatesFailed = result.GatesFailed

				// Emit verification.failed event
				logger := events.NewLogger(events.DefaultLogPath())
				if err := logger.LogVerificationFailed(events.VerificationFailedData{
					Workspace:   agentName,
					GatesFailed: gatesFailed,
					Errors:      result.Errors,
					Skill:       skillName,
				}); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log verification failure event: %v\n", err)
				}

				fmt.Fprintf(os.Stderr, "Cannot complete orchestrator session - verification failed:\n\n")
				printGateResults(result.GateResults, result.GatesFailed)
				return fmt.Errorf("verification failed")
			}
			fmt.Println("Completion signal: SYNTHESIS.md verified (content validated)")
		} else if !isUntracked {
			// Regular agents use beads phase verification
			// Workspace already found at top of function
			if workspacePath != "" {
				fmt.Printf("Workspace: %s\n", agentName)
			}

			// Use beadsProjectDir for verification (where the beads issue lives)
			result, err := verify.VerifyCompletionFull(beadsID, workspacePath, beadsProjectDir, "", serverURL)
			if err != nil {
				return fmt.Errorf("verification failed: %w", err)
			}

			// Track skill name for event
			skillName = result.Skill

			// If skip flags are set, filter out the skipped gates from failures
			if skipConfig.hasAnySkip() && !result.Passed {
				var filteredErrors []string
				var filteredGates []string
				var skippedGatesFound []string

				for _, gate := range result.GatesFailed {
					if skipConfig.shouldSkipGate(gate) {
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
						// Match error messages to gates (crude but effective)
						if strings.Contains(strings.ToLower(e), strings.ReplaceAll(gate, "_", " ")) ||
							strings.Contains(strings.ToLower(e), strings.ReplaceAll(gate, "_", "-")) {
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

				// Persist build skip memory when build gate is bypassed
				// This auto-skips build for subsequent completions without --skip-build
				for _, gate := range skippedGatesFound {
					if gate == verify.GateBuild {
						skippedByID := beadsID
						if skippedByID == "" {
							skippedByID = agentName
						}
						if err := verify.WriteBuildSkipMemory(beadsProjectDir, skipConfig.Reason, skippedByID); err != nil {
							fmt.Fprintf(os.Stderr, "Warning: failed to persist build skip memory: %v\n", err)
						} else {
							fmt.Printf("Build skip memory saved (auto-skips build for subsequent completions, expires in %v)\n", verify.BuildSkipDuration)
						}
						break
					}
				}

				// Update result with filtered data
				result.GatesFailed = filteredGates
				result.Errors = filteredErrors
				result.Passed = len(filteredGates) == 0
			}

			if !result.Passed {
				verificationPassed = false
				gatesFailed = result.GatesFailed

				// Emit verification.failed event
				logger := events.NewLogger(events.DefaultLogPath())
				if err := logger.LogVerificationFailed(events.VerificationFailedData{
					BeadsID:     beadsID,
					Workspace:   agentName,
					GatesFailed: gatesFailed,
					Errors:      result.Errors,
					Skill:       skillName,
				}); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log verification failure event: %v\n", err)
				}

				fmt.Fprintf(os.Stderr, "Cannot complete agent - verification failed:\n\n")
				printGateResults(result.GateResults, result.GatesFailed)
				return fmt.Errorf("verification failed")
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

			// Behavioral validation checkpoint - structured output, not blocking
			// This helps orchestrators identify when behavioral verification is warranted
			if beadsID != "" && beadsProjectDir != "" {
				comments, _ := verify.GetComments(beadsID)
				behavioralResult := verify.CheckBehavioralValidationForCompletion(beadsID, workspacePath, beadsProjectDir, comments)
				if behavioralResult != nil && behavioralResult.BehavioralValidationSuggested {
					printBehavioralValidationInfo(behavioralResult)
				}
			}
		} else {
			fmt.Println("Skipping phase verification (untracked agent)")
		}
	} else {
		// --force was used, run verification anyway to capture which gates would have failed
		if !isOrchestratorSession && !isUntracked {
			result, err := verify.VerifyCompletionFull(beadsID, workspacePath, beadsProjectDir, "", serverURL)
			if err == nil {
				skillName = result.Skill
				if !result.Passed {
					verificationPassed = false
					gatesFailed = result.GatesFailed
				}
			}
		} else if isOrchestratorSession {
			// Run verification to capture what would have failed
			result := verify.VerifyOrchestratorCompletion(workspacePath)
			skillName = result.Skill
			if !result.Passed {
				verificationPassed = false
				gatesFailed = result.GatesFailed
			}
		}
		fmt.Println("Skipping all verification (--force) - DEPRECATED: use targeted --skip-* flags")
	}

	// Check liveness before closing - warn if agent appears still running
	// BUT: Skip this check if Phase: Complete was reported - agent said it's done,
	// so whether its session is still open is irrelevant.
	// This prevents false positives from OpenCode sessions that persist to disk.
	// Also skip for untracked/orchestrator agents
	if !completeForce && !isUntracked {
		// Check if Phase: Complete was reported (only for regular agents with beads)
		phaseComplete := false
		if !isOrchestratorSession && beadsID != "" {
			phaseComplete, _ = verify.IsPhaseComplete(beadsID)
		}

		// Only check liveness if agent hasn't reported completion
		if !phaseComplete {
			liveness := state.GetLiveness(beadsID, serverURL, beadsProjectDir)
			if liveness.IsAlive() {
				// Build warning message with details about what's still running
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

				// Check if stdin is a terminal for interactive prompting
				if !term.IsTerminal(int(os.Stdin.Fd())) {
					return fmt.Errorf("agent still running and stdin is not a terminal; use --force to complete anyway")
				}

				// Prompt user for confirmation
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
			}
		}
	}

	// DISABLED: Reproduction verification gate (Jan 4, 2026)
	// This was added to ensure bugs are actually fixed before closing, but it created
	// too much friction - agents couldn't complete without manual intervention.
	// Keeping the code commented for potential future re-enablement with better UX.
	// See: .kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md
	/*
		if !completeSkipReproCheck {
			reproResult, err := verify.GetReproForCompletion(beadsID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to check reproduction: %v\n", err)
			} else if reproResult != nil && reproResult.IsBug {
				// ... gate logic disabled ...
			}
		}
	*/
	_ = completeSkipReproCheck  // silence unused variable warning
	_ = completeSkipReproReason // silence unused variable warning

	// Gate completion on discovered work disposition (workspace already found at top)
	// This ensures recommendations from agents don't get silently dropped
	if workspacePath != "" && !completeForce {
		synthesis, err := verify.ParseSynthesis(workspacePath)
		if err == nil && synthesis != nil {
			// Collect discovered work items
			items := verify.CollectDiscoveredWork(synthesis)

			if len(items) > 0 {
				fmt.Println("\n--- Discovered Work Gate ---")

				if synthesis.Recommendation != "" && synthesis.Recommendation != "close" {
					fmt.Printf("Recommendation: %s\n", synthesis.Recommendation)
				}

				fmt.Printf("%d discovered work item(s) require disposition:\n", len(items))

				// Check if stdin is a terminal for interactive prompting
				if !term.IsTerminal(int(os.Stdin.Fd())) {
					fmt.Println("(Skipping interactive prompts - stdin is not a terminal)")
					fmt.Println("Use --force to complete without disposition, or run interactively")
				} else {
					// Prompt for disposition of each item
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
						// Clean up the item description for issue title
						title := strings.TrimPrefix(item.Description, "- ")
						title = strings.TrimPrefix(title, "* ")
						// Remove numbered prefixes like "1. "
						if len(title) > 3 && title[0] >= '0' && title[0] <= '9' && (title[1] == '.' || (title[1] >= '0' && title[1] <= '9' && title[2] == '.')) {
							if idx := strings.Index(title, ". "); idx != -1 && idx < 4 {
								title = title[idx+2:]
							}
						}

						// Build labels with triage:review and suggested area label
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

					// Log skip-all reason if used
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

	// Detect knowledge gaps - cross-check agent questions against existing kb
	// This helps identify when agents surface questions that kb already answers,
	// revealing gaps in knowledge surfacing mechanisms.
	if workspacePath != "" && !completeForce {
		gapResult, err := verify.DetectKnowledgeGaps(workspacePath, beadsID, skillName, beadsProjectDir)
		if err != nil {
			// Non-critical - warn but don't fail completion
			fmt.Fprintf(os.Stderr, "Warning: failed to detect knowledge gaps: %v\n", err)
		} else if gapResult != nil && gapResult.GapsDetected > 0 {
			// Log gaps to ~/.orch/knowledge-gaps.jsonl
			if err := verify.LogKnowledgeGaps(gapResult.Gaps); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to log knowledge gaps: %v\n", err)
			} else {
				// Inform user about detected gaps (informational, not blocking)
				fmt.Printf("\nℹ️  Knowledge Gap Detection: %d gap(s) detected and logged\n", gapResult.GapsDetected)
				fmt.Printf("   Agent surfaced questions that kb already answers.\n")
				fmt.Printf("   Review: cat ~/.orch/knowledge-gaps.jsonl | jq 'select(.workspace==\"%s\")'\n", agentName)
			}
		}
	}

	// TODO: Update synthesis with spawn completion info (Capture at Context principle)
	// This is only for worker agents, not orchestrator sessions (which manage their own handoffs)
	// UpdateHandoffAfterComplete was planned but never implemented - see:
	// .kb/investigations/2026-01-14-inv-orch-complete-triggers-handoff-updates.md
	// if !isOrchestratorSession && agentName != "" && beadsID != "" {
	// 	if err := UpdateHandoffAfterComplete(beadsProjectDir, agentName, beadsID, skillName); err != nil {
	// 		// Non-critical - warn but don't fail completion
	// 		fmt.Fprintf(os.Stderr, "Warning: failed to update synthesis: %v\n", err)
	// 	}
	// }

	// Determine close reason
	reason := completeReason
	if reason == "" {
		// For tracked agents, try to get summary from phase status
		if !isUntracked && beadsID != "" {
			status, _ := verify.GetPhaseStatus(beadsID)
			if status.Summary != "" {
				reason = status.Summary
			}
		}
		if reason == "" {
			if isOrchestratorSession {
				reason = "Orchestrator session completed"
			} else {
				reason = "Completed via orch complete"
			}
		}
	}

	// Close the beads issue if not already closed
	// Skip for untracked agents and orchestrator sessions (they have no beads issue to close)
	if !isClosed && !isUntracked && beadsID != "" {
		// Epic protection: check for open children before closing
		if issue != nil && issue.IssueType == "epic" && !completeForceCloseEpic {
			openChildren, err := verify.GetOpenEpicChildren(beadsID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to check epic children: %v\n", err)
				// Continue - don't block on failure to check children
			} else if len(openChildren) > 0 {
				fmt.Fprintf(os.Stderr, "Cannot complete epic %s - has %d open children:\n", beadsID, len(openChildren))
				// Show up to 5 children
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
		}

		// Epic orphan logging: emit attention signal when force-closing epic with open children
		if issue != nil && issue.IssueType == "epic" && completeForceCloseEpic {
			openChildren, err := verify.GetOpenEpicChildren(beadsID)
			if err == nil && len(openChildren) > 0 {
				// Log orphaned children to events for attention system
				orphanIDs := make([]string, len(openChildren))
				for i, child := range openChildren {
					orphanIDs[i] = child.ID
				}
				logger := events.NewLogger(events.DefaultLogPath())
				if err := logger.LogEpicOrphaned(events.EpicOrphanedData{
					EpicID:           beadsID,
					EpicTitle:        issue.Title,
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
		}

		// Pass force flag when --skip-phase-complete is set
		// This bypasses bd close's independent Phase: Complete gate
		if err := verify.CloseIssueForce(beadsID, reason, skipConfig.PhaseComplete); err != nil {
			return fmt.Errorf("failed to close issue: %w", err)
		}
		fmt.Printf("Closed beads issue: %s\n", beadsID)

		// Epic auto-close: Check if this was the last open child of a parent epic
		if !isOrchestratorSession {
			parentInfo, err := verify.GetParentEpicInfo(beadsID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to check parent epic: %v\n", err)
			} else if parentInfo != nil && parentInfo.Status != "closed" && parentInfo.OpenChildrenLeft == 0 {
				// All siblings are complete - prompt to close parent epic
				if completeAutoCloseParent {
					// Auto-close mode: close without prompting
					if err := verify.CloseIssue(parentInfo.ID, "All children completed"); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to auto-close parent epic %s: %v\n", parentInfo.ID, err)
					} else {
						fmt.Printf("Auto-closed parent epic: %s (%s)\n", parentInfo.ID, parentInfo.Title)
					}
				} else {
					// Interactive mode: prompt user
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
						fmt.Printf("Parent epic left open. Use \047orch complete %s\047 to close later.\n", parentInfo.ID)
					}
				}
			}
		}

		// Remove triage:ready label on successful completion
		// This ensures failed/abandoned agents leave issues in ready queue for daemon retry
		if err := verify.RemoveTriageReadyLabel(beadsID); err != nil {
			// Non-critical - the issue may not have had this label
			// or it was already removed
		}
	} else if isOrchestratorSession {
		fmt.Printf("Completed orchestrator session: %s\n", agentName)
		// Update orchestrator session status to "completed" in the registry
		// We update rather than unregister to preserve session history for tracking
		registry := session.NewRegistry("")
		if err := registry.Update(agentName, func(s *session.OrchestratorSession) {
			s.Status = "completed"
		}); err != nil {
			if err == session.ErrSessionNotFound {
				// Session wasn't in registry - likely a legacy workspace or spawned before registry existed
				fmt.Printf("Note: Session %s was not in registry (legacy workspace)\n", agentName)
			} else {
				fmt.Fprintf(os.Stderr, "Warning: failed to update session status in registry: %v\n", err)
			}
		} else {
			fmt.Printf("Updated session registry: status → completed\n")
		}
	} else if isUntracked {
		fmt.Printf("Cleaned up untracked agent: %s\n", identifier)
	}
	fmt.Printf("Reason: %s\n", reason)

	// Export activity to ACTIVITY.json for archival (Tier 2 persistence)
	// This is done BEFORE deleting the session (needs API access) and BEFORE archiving.
	// Only for non-orchestrator sessions - orchestrators export transcript separately.
	if workspacePath != "" && !isOrchestratorSession {
		sessionFile := filepath.Join(workspacePath, ".session_id")
		if data, err := os.ReadFile(sessionFile); err == nil {
			sessionID := strings.TrimSpace(string(data))
			if sessionID != "" {
				if activityPath, err := activity.ExportToWorkspace(sessionID, workspacePath, serverURL); err != nil {
					// Non-fatal - activity export is for archival only
					fmt.Fprintf(os.Stderr, "Warning: failed to export activity: %v\n", err)
				} else if activityPath != "" {
					fmt.Printf("Exported activity: %s\n", filepath.Base(activityPath))
				}
			}
		}
	}

	// Collect telemetry (duration and tokens) for model performance tracking
	// IMPORTANT: This MUST happen BEFORE DeleteSession() because GetSessionTokens()
	// requires the session to still exist in OpenCode.
	var durationSecs, tokensIn, tokensOut int
	var outcome string
	if workspacePath != "" {
		durationSecs, tokensIn, tokensOut, outcome = collectCompletionTelemetry(workspacePath, completeForce, verificationPassed)
	}

	// Delete OpenCode session to prevent ghost agents in orch status
	// This is done after collecting telemetry (which needs session data) but before
	// cleanup. If deletion fails, the issue is still properly closed.
	if workspacePath != "" {
		// Try to get session ID from workspace .session_id file
		sessionFile := filepath.Join(workspacePath, ".session_id")
		if data, err := os.ReadFile(sessionFile); err == nil {
			sessionID := strings.TrimSpace(string(data))
			if sessionID != "" {
				client := opencode.NewClient(serverURL)
				if err := client.DeleteSession(sessionID); err != nil {
					// Non-fatal - session might already be deleted or not exist
					fmt.Fprintf(os.Stderr, "Warning: failed to delete OpenCode session %s: %v\n", sessionID[:12], err)
				} else {
					fmt.Printf("Deleted OpenCode session: %s\n", sessionID[:12])
				}
			}
		}

		// Terminate the OpenCode process if it's still running
		// This prevents orphaned processes when agents crash or are killed
		// Read process ID from workspace .process_id file
		pid := spawn.ReadProcessID(workspacePath)
		if pid > 0 {
			if process.Terminate(pid, "opencode") {
				// Process was terminated (logged by process.Terminate)
			}
		}
	}

	// For orchestrator sessions, export transcript before cleanup
	if workspacePath != "" && isOrchestratorSession {
		// Use agentName (workspace name) as identifier for orchestrator transcript export
		if err := exportOrchestratorTranscript(workspacePath, beadsProjectDir, agentName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to export orchestrator transcript: %v\n", err)
		}
	}

	// Archive workspace after successful completion (unless --no-archive is set)
	// This happens after all workspace reads (session deletion, transcript export)
	// but before tmux cleanup. The workspace is moved to .orch/workspace/archived/
	var archivedPath string
	if workspacePath != "" && !completeNoArchive {
		var err error
		archivedPath, err = archiveWorkspace(workspacePath, beadsProjectDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to archive workspace: %v\n", err)
		} else {
			fmt.Printf("Archived workspace: %s\n", filepath.Base(archivedPath))

			// Update session registry with archived path (orchestrator sessions only)
			if isOrchestratorSession && archivedPath != "" {
				registry := session.NewRegistry("")
				if err := registry.Update(agentName, func(s *session.OrchestratorSession) {
					s.ArchivedPath = archivedPath
				}); err != nil {
					// Non-critical - session may not be in registry
					if err != session.ErrSessionNotFound {
						fmt.Fprintf(os.Stderr, "Warning: failed to update archived path in registry: %v\n", err)
					}
				}
			}
		}
	} else if completeNoArchive && workspacePath != "" {
		fmt.Println("Skipped workspace archival (--no-archive)")
	}

	// Clean up Docker container if this was a docker-backend spawn
	// This must happen before tmux cleanup since killing tmux might leave container orphaned
	if workspacePath != "" {
		containerName := spawn.ReadContainerID(workspacePath)
		if containerName != "" {
			if err := spawn.CleanupDockerContainer(containerName); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to clean up Docker container %s: %v\n", containerName, err)
			} else {
				fmt.Printf("Cleaned up Docker container: %s\n", containerName)
			}
		}
	}

	// Clean up tmux window if it exists (prevents phantom accumulation)
	// For orchestrators, search by workspace name; for regular agents, search by beads ID
	var window *tmux.WindowInfo
	var tmuxSessionName string
	var findErr error

	if isOrchestratorSession {
		// Orchestrator windows only contain workspace names, not beads IDs
		window, tmuxSessionName, findErr = tmux.FindWindowByWorkspaceNameAllSessions(agentName)
	} else {
		// Worker windows contain beads IDs in format [beadsID]
		var windowSearchID string
		if beadsID != "" {
			windowSearchID = beadsID
		} else {
			windowSearchID = identifier
		}
		window, tmuxSessionName, findErr = tmux.FindWindowByBeadsIDAllSessions(windowSearchID)
	}

	if findErr == nil && window != nil {
		if err := tmux.KillWindow(window.Target); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close tmux window %s: %v\n", window.Target, err)
		} else {
			fmt.Printf("Closed tmux window: %s:%s\n", tmuxSessionName, window.Name)
		}
	}

	// Check for new CLI commands that may need skill documentation
	// Note: Auto-rebuild is done BEFORE verification via rebuildGoProjectsIfNeeded()
	if hasGoChangesInRecentCommits(beadsProjectDir) {
		newCommands := detectNewCLICommands(beadsProjectDir)
		if len(newCommands) > 0 {
			// Track new commands in doc debt registry
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

	// Check for notable changelog entries (BREAKING/behavioral changes, especially skill changes)
	if !completeNoChangelogCheck {
		// Extract agent's skill from workspace if available
		var agentSkill string
		if workspacePath != "" {
			agentSkill, _ = verify.ExtractSkillNameFromSpawnContext(workspacePath)
		}

		notableEntries := detectNotableChangelogEntries(beadsProjectDir, agentSkill)
		if len(notableEntries) > 0 {
			fmt.Println()
			fmt.Println("┌─────────────────────────────────────────────────────────────┐")
			fmt.Println("│  ⚠️  NOTABLE ECOSYSTEM CHANGES DETECTED                      │")
			fmt.Println("├─────────────────────────────────────────────────────────────┤")
			for _, entry := range notableEntries {
				// Wrap long entries
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

	// Log the completion with verification metadata
	// Note: Telemetry (durationSecs, tokensIn, tokensOut, outcome) was collected earlier,
	// before session deletion, to ensure token data is available.
	logger := events.NewLogger(events.DefaultLogPath())
	completedData := events.AgentCompletedData{
		Reason:             reason,
		Forced:             completeForce,
		Untracked:          isUntracked,
		Orchestrator:       isOrchestratorSession,
		VerificationPassed: verificationPassed,
		Skill:              skillName,
		DurationSeconds:    durationSecs,
		TokensInput:        tokensIn,
		TokensOutput:       tokensOut,
		Outcome:            outcome,
	}
	if beadsID != "" {
		completedData.BeadsID = beadsID
	}
	if agentName != "" {
		completedData.Workspace = agentName
	}
	// If completion was forced, record which gates were bypassed
	if completeForce && len(gatesFailed) > 0 {
		completedData.GatesBypassed = gatesFailed
	}
	if err := logger.LogAgentCompleted(completedData); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Invalidate orch serve cache to ensure dashboard shows updated status immediately.
	// Without this, the TTL cache holds stale "active" status after completion.
	invalidateServeCache()

	return nil
}
