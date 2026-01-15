// Package main provides the complete command for completing agents and closing beads issues.
// Extracted from main.go as part of the main.go refactoring (Phase 3).
package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
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

	// Targeted skip flags (replace blanket --force)
	// Each requires completeSkipReason to be set (min 10 chars)
	completeSkipTestEvidence      bool
	completeSkipVisual            bool
	completeSkipGitDiff           bool
	completeSkipSynthesis         bool
	completeSkipBuild             bool
	completeSkipConstraint        bool
	completeSkipPhaseGate         bool
	completeSkipSkillOutput       bool
	completeSkipDecisionPatch     bool
	completeSkipPhaseComplete     bool
	completeSkipReason            string // Required for all --skip-* flags (min 10 chars)
)

var completeCmd = &cobra.Command{
	Use:   "complete [beads-id-or-workspace]",
	Short: "Complete an agent and close the beads issue",
	Long: `Complete an agent's work by verifying Phase: Complete and closing the beads issue.

Checks that the agent has reported "Phase: Complete" via beads comments before
closing the issue.

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

TARGETED SKIP FLAGS:
Use --skip-{gate} with --skip-reason to bypass specific gates:
  --skip-test-evidence    Skip test evidence gate
  --skip-visual           Skip visual verification gate
  --skip-git-diff         Skip git diff verification gate
  --skip-synthesis        Skip SYNTHESIS.md gate
  --skip-build            Skip build verification gate
  --skip-constraint       Skip constraint verification gate
  --skip-phase-gate       Skip phase gate verification
  --skip-skill-output     Skip skill output verification gate
  --skip-decision-patch   Skip decision patch count gate
  --skip-phase-complete   Skip Phase: Complete gate

Each --skip-* flag requires --skip-reason with a minimum of 10 characters
explaining why the gate is being bypassed. Bypasses are logged for audit.

DEPRECATION: --force is deprecated. Use targeted --skip-* flags instead.
Using --force will show a deprecation warning.

For orchestrator sessions (spawned with orchestrator or meta-orchestrator skill),
the argument is the workspace name instead of beads ID. Orchestrators use
SESSION_HANDOFF.md as completion signal instead of Phase: Complete.

For agents that modified web/ files (UI tasks), --approve is required to explicitly
confirm human review of the visual changes. This prevents agents from self-certifying
UI correctness.

For cross-project completion (agents spawned with --workdir in another project),
the command auto-detects the project from the workspace's SPAWN_CONTEXT.md.
Use --workdir as explicit override when auto-detection fails.

Examples:
  orch-go complete proj-123
  orch-go complete proj-123 --reason "All tests passing"
  orch-go complete proj-123 --approve       # Approve UI changes after visual review
  orch-go complete proj-123 --skip-test-evidence --skip-reason "Tests run in CI"
  orch-go complete proj-123 --skip-git-diff --skip-synthesis --skip-reason "Docs-only change"
  orch-go complete kb-cli-123 --workdir ~/projects/kb-cli  # Cross-project completion

  # Orchestrator session completion (by workspace name)
  orch-go complete og-orch-goal-04jan       # Complete orchestrator session

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
	completeCmd.Flags().BoolVar(&completeNoChangelogCheck, "no-changelog-check", false, "Skip changelog detection for notable changes")
	completeCmd.Flags().BoolVar(&completeSkipReproCheck, "skip-repro-check", false, "Skip reproduction verification for bug issues (requires --reason)")
	completeCmd.Flags().StringVar(&completeSkipReproReason, "skip-repro-reason", "", "Reason for skipping reproduction verification")

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
	completeCmd.Flags().StringVar(&completeSkipReason, "skip-reason", "", "Reason for skip (required for all --skip-* flags, min 10 chars)")
}

// SkipConfig holds the configuration for which verification gates to skip.
type SkipConfig struct {
	TestEvidence      bool
	Visual            bool
	GitDiff           bool
	Synthesis         bool
	Build             bool
	Constraint        bool
	PhaseGate         bool
	SkillOutput       bool
	DecisionPatch     bool
	PhaseComplete     bool
	Reason            string // Required reason for skips
}

// hasAnySkip returns true if any skip flag is set.
func (c SkipConfig) hasAnySkip() bool {
	return c.TestEvidence || c.Visual || c.GitDiff || c.Synthesis ||
		c.Build || c.Constraint || c.PhaseGate || c.SkillOutput ||
		c.DecisionPatch || c.PhaseComplete
}

// skippedGates returns a list of gate names that are being skipped.
func (c SkipConfig) skippedGates() []string {
	var gates []string
	if c.TestEvidence {
		gates = append(gates, verify.GateTestEvidence)
	}
	if c.Visual {
		gates = append(gates, verify.GateVisualVerify)
	}
	if c.GitDiff {
		gates = append(gates, verify.GateGitDiff)
	}
	if c.Synthesis {
		gates = append(gates, verify.GateSynthesis)
	}
	if c.Build {
		gates = append(gates, verify.GateBuild)
	}
	if c.Constraint {
		gates = append(gates, verify.GateConstraint)
	}
	if c.PhaseGate {
		gates = append(gates, verify.GatePhaseGate)
	}
	if c.SkillOutput {
		gates = append(gates, verify.GateSkillOutput)
	}
	if c.DecisionPatch {
		gates = append(gates, verify.GateDecisionPatchLimit)
	}
	if c.PhaseComplete {
		gates = append(gates, verify.GatePhaseComplete)
	}
	return gates
}

// shouldSkipGate returns true if the given gate should be skipped.
func (c SkipConfig) shouldSkipGate(gate string) bool {
	switch gate {
	case verify.GateTestEvidence:
		return c.TestEvidence
	case verify.GateVisualVerify:
		return c.Visual
	case verify.GateGitDiff:
		return c.GitDiff
	case verify.GateSynthesis:
		return c.Synthesis
	case verify.GateBuild:
		return c.Build
	case verify.GateConstraint:
		return c.Constraint
	case verify.GatePhaseGate:
		return c.PhaseGate
	case verify.GateSkillOutput:
		return c.SkillOutput
	case verify.GateDecisionPatchLimit:
		return c.DecisionPatch
	case verify.GatePhaseComplete:
		return c.PhaseComplete
	default:
		return false
	}
}

// getSkipConfig builds the skip configuration from command-line flags.
func getSkipConfig() SkipConfig {
	return SkipConfig{
		TestEvidence:  completeSkipTestEvidence,
		Visual:        completeSkipVisual,
		GitDiff:       completeSkipGitDiff,
		Synthesis:     completeSkipSynthesis,
		Build:         completeSkipBuild,
		Constraint:    completeSkipConstraint,
		PhaseGate:     completeSkipPhaseGate,
		SkillOutput:   completeSkipSkillOutput,
		DecisionPatch: completeSkipDecisionPatch,
		PhaseComplete: completeSkipPhaseComplete,
		Reason:        completeSkipReason,
	}
}

// validateSkipFlags validates that --skip-reason is provided when --skip-* flags are used.
func validateSkipFlags(skipConfig SkipConfig) error {
	if !skipConfig.hasAnySkip() {
		return nil
	}

	if skipConfig.Reason == "" {
		return fmt.Errorf("--skip-reason is required when using --skip-* flags")
	}

	if len(skipConfig.Reason) < 10 {
		return fmt.Errorf("--skip-reason must be at least 10 characters (got %d)", len(skipConfig.Reason))
	}

	return nil
}

// logSkipEvents logs verification.bypassed events for all skipped gates.
func logSkipEvents(skipConfig SkipConfig, beadsID, workspace, skill string) {
	logger := events.NewLogger(events.DefaultLogPath())
	for _, gate := range skipConfig.skippedGates() {
		if err := logger.LogVerificationBypassed(events.VerificationBypassedData{
			BeadsID:   beadsID,
			Workspace: workspace,
			Gate:      gate,
			Reason:    skipConfig.Reason,
			Skill:     skill,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log bypass event for %s: %v\n", gate, err)
		}
	}
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
		// Resolve short beads ID to full ID (e.g., "qdaa" -> "orch-go-qdaa")
		resolvedID, err := resolveShortBeadsID(identifier)
		if err != nil {
			return fmt.Errorf("failed to resolve beads ID: %w", err)
		}
		beadsID = resolvedID
		// Find workspace by beads ID
		workspacePath, agentName = findWorkspaceByBeadsID(currentDir, beadsID)
	}

	// Determine beads project directory:
	// 1. If --workdir provided, use that
	// 2. Otherwise, try to auto-detect from workspace SPAWN_CONTEXT.md
	// 3. Fall back to current directory
	var beadsProjectDir string

	if workdir != "" {
		// Explicit --workdir flag provided
		beadsProjectDir, err = filepath.Abs(workdir)
		if err != nil {
			return fmt.Errorf("failed to resolve workdir path: %w", err)
		}
		// Verify directory exists
		if stat, err := os.Stat(beadsProjectDir); err != nil {
			return fmt.Errorf("workdir does not exist: %s", beadsProjectDir)
		} else if !stat.IsDir() {
			return fmt.Errorf("workdir is not a directory: %s", beadsProjectDir)
		}
		fmt.Printf("Using explicit workdir: %s\n", beadsProjectDir)
	} else if workspacePath != "" {
		// Try to extract PROJECT_DIR from workspace SPAWN_CONTEXT.md
		projectDirFromWorkspace := extractProjectDirFromWorkspace(workspacePath)
		if projectDirFromWorkspace != "" && projectDirFromWorkspace != currentDir {
			// Cross-project agent detected
			beadsProjectDir = projectDirFromWorkspace
			fmt.Printf("Auto-detected cross-project: %s\n", filepath.Base(beadsProjectDir))
		} else {
			beadsProjectDir = currentDir
		}
	} else {
		beadsProjectDir = currentDir
	}

	// Set beads.DefaultDir for cross-project operations BEFORE any beads operations
	if beadsProjectDir != currentDir {
		beads.DefaultDir = beadsProjectDir
	}

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

	// Verify completion status
	// - For orchestrator sessions: check SESSION_HANDOFF.md exists
	// - For regular agents: check Phase: Complete via beads comments
	// - Skip flags allow targeted bypass of specific gates
	if !completeForce {
		if isOrchestratorSession {
			// Orchestrator sessions use SESSION_HANDOFF.md as completion signal
			if workspacePath != "" {
				fmt.Printf("Workspace: %s\n", agentName)
			}

			if !hasSessionHandoff(workspacePath) {
				verificationPassed = false
				gatesFailed = append(gatesFailed, verify.GateSessionHandoff)

				// Emit verification.failed event
				logger := events.NewLogger(events.DefaultLogPath())
				if err := logger.LogVerificationFailed(events.VerificationFailedData{
					Workspace:   agentName,
					GatesFailed: gatesFailed,
					Errors:      []string{"SESSION_HANDOFF.md not found"},
					Skill:       "orchestrator",
				}); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log verification failure event: %v\n", err)
				}

				fmt.Fprintf(os.Stderr, "Cannot complete orchestrator session - SESSION_HANDOFF.md not found\n")
				fmt.Fprintf(os.Stderr, "\nOrchestrator must run: orch session end\n")
				fmt.Fprintf(os.Stderr, "Or use --force to skip verification\n")
				return fmt.Errorf("verification failed: SESSION_HANDOFF.md not found")
			}
			fmt.Println("Completion signal: SESSION_HANDOFF.md found")
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

				fmt.Fprintf(os.Stderr, "Cannot complete agent - verification failed:\n")
				for _, e := range result.Errors {
					fmt.Fprintf(os.Stderr, "  - %s\n", e)
				}
				fmt.Fprintf(os.Stderr, "\nAgent must run: bd comment %s \"Phase: Complete - <summary>\"\n", beadsID)
				fmt.Fprintf(os.Stderr, "Or use --skip-<gate> --skip-reason to bypass specific gates\n")
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
		} else if isOrchestratorSession && !hasSessionHandoff(workspacePath) {
			verificationPassed = false
			gatesFailed = append(gatesFailed, verify.GateSessionHandoff)
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

	// Update session handoff with spawn completion info (Capture at Context principle)
	// This is only for worker agents, not orchestrator sessions (which manage their own handoffs)
	if !isOrchestratorSession && agentName != "" && beadsID != "" {
		if err := UpdateHandoffAfterComplete(beadsProjectDir, agentName, beadsID, skillName); err != nil {
			// Non-critical - warn but don't fail completion
			fmt.Fprintf(os.Stderr, "Warning: failed to update session handoff: %v\n", err)
		}
	}

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
		if err := verify.CloseIssue(beadsID, reason); err != nil {
			return fmt.Errorf("failed to close issue: %w", err)
		}
		fmt.Printf("Closed beads issue: %s\n", beadsID)

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

	// Delete OpenCode session to prevent ghost agents in orch status
	// This is done after closing the beads issue but before cleanup, so if
	// deletion fails, the issue is still properly closed.
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
	}

	// For orchestrator sessions, export transcript before cleanup
	if workspacePath != "" && isOrchestratorSession {
		// Use agentName (workspace name) as identifier for orchestrator transcript export
		if err := exportOrchestratorTranscript(workspacePath, beadsProjectDir, agentName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to export orchestrator transcript: %v\n", err)
		}
	}

	// Clean up tmux window if it exists (prevents phantom accumulation)
	// For orchestrators, search by workspace name; for regular agents, search by beads ID
	var windowSearchID string
	if isOrchestratorSession {
		windowSearchID = agentName
	} else if beadsID != "" {
		windowSearchID = beadsID
	} else {
		windowSearchID = identifier
	}
	if window, sessionName, err := tmux.FindWindowByBeadsIDAllSessions(windowSearchID); err == nil && window != nil {
		if err := tmux.KillWindow(window.Target); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close tmux window %s: %v\n", window.Target, err)
		} else {
			fmt.Printf("Closed tmux window: %s:%s\n", sessionName, window.Name)
		}
	}

	// Auto-rebuild if agent committed Go changes (in the beads project)
	if hasGoChangesInRecentCommits(beadsProjectDir) {
		fmt.Println("Detected Go file changes in recent commits")
		if err := runAutoRebuild(beadsProjectDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: auto-rebuild failed: %v\n", err)
		} else {
			fmt.Println("Auto-rebuild completed: make install")
			// Restart orch serve if running
			if restarted, err := restartOrchServe(beadsProjectDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to restart orch serve: %v\n", err)
			} else if restarted {
				fmt.Println("Restarted orch serve")
			}
		}

		// Check for new CLI commands that may need skill documentation
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
	logger := events.NewLogger(events.DefaultLogPath())
	completedData := events.AgentCompletedData{
		Reason:             reason,
		Forced:             completeForce,
		Untracked:          isUntracked,
		Orchestrator:       isOrchestratorSession,
		VerificationPassed: verificationPassed,
		Skill:              skillName,
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

// invalidateServeCache sends a request to orch serve to invalidate its caches.
// This ensures the dashboard shows updated agent status immediately after completion.
// Silently fails if orch serve is not running (cache will refresh via TTL).
func invalidateServeCache() {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Post(
		fmt.Sprintf("http://localhost:%d/api/cache/invalidate", DefaultServePort),
		"application/json",
		nil,
	)
	if err != nil {
		// Silent failure - orch serve might not be running
		return
	}
	defer resp.Body.Close()
	// We don't care about the response - if it worked, great; if not, TTL will eventually refresh
}

// addApprovalComment adds an approval comment to a beads issue.
// This is used by --approve flag to mark visual changes as human-reviewed.
func addApprovalComment(beadsID, comment string) error {
	// Try RPC client first with auto-reconnect
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		// Use "orchestrator" as the author for approval comments
		err := client.AddComment(beadsID, "orchestrator", comment)
		if err == nil {
			return nil
		}
		// Fall through to CLI fallback on RPC error
	}

	// Fallback to CLI
	return beads.FallbackAddComment(beadsID, comment)
}

// hasGoChangesInRecentCommits checks if any of the last 5 commits contain changes
// to cmd/orch/*.go or pkg/*.go files.
func hasGoChangesInRecentCommits(projectDir string) bool {
	// Get changed files from last 5 commits
	cmd := exec.Command("git", "diff", "--name-only", "HEAD~5..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// If git command fails (e.g., not enough commits), try last 1 commit
		cmd = exec.Command("git", "diff", "--name-only", "HEAD~1..HEAD")
		cmd.Dir = projectDir
		output, err = cmd.Output()
		if err != nil {
			return false
		}
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Check if file matches cmd/orch/*.go or pkg/*.go or pkg/**/*.go
		if strings.HasPrefix(line, "cmd/orch/") && strings.HasSuffix(line, ".go") {
			return true
		}
		if strings.HasPrefix(line, "pkg/") && strings.HasSuffix(line, ".go") {
			return true
		}
	}
	return false
}

// detectNewCLICommands checks if any of the last 5 commits added new CLI command files
// to cmd/orch/. A file is considered a new command if:
// 1. It's in cmd/orch/*.go (not a test file)
// 2. It was added (not modified) in recent commits
// 3. It contains cobra.Command definitions
// Returns the list of new command file names (without path prefix).
func detectNewCLICommands(projectDir string) []string {
	var newCommands []string

	// Get files added (not modified) in last 5 commits
	// The 'A' status means added
	cmd := exec.Command("git", "diff", "--name-status", "HEAD~5..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// If git command fails (e.g., not enough commits), try last 1 commit
		cmd = exec.Command("git", "diff", "--name-status", "HEAD~1..HEAD")
		cmd.Dir = projectDir
		output, err = cmd.Output()
		if err != nil {
			return nil
		}
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Parse status line: "A\tcmd/orch/newcmd.go" or "M\tcmd/orch/main.go"
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		status := parts[0]
		filePath := parts[1]

		// Only care about added files (not modified)
		if status != "A" {
			continue
		}

		// Only check cmd/orch/*.go files (not test files)
		if !strings.HasPrefix(filePath, "cmd/orch/") || !strings.HasSuffix(filePath, ".go") {
			continue
		}
		if strings.HasSuffix(filePath, "_test.go") {
			continue
		}

		// Read the file to check if it contains cobra command definitions
		fullPath := filepath.Join(projectDir, filePath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		// Look for cobra command pattern: "var xxxCmd = &cobra.Command{"
		if strings.Contains(string(content), "cobra.Command{") &&
			strings.Contains(string(content), "rootCmd.AddCommand(") {
			// Extract just the filename
			fileName := filepath.Base(filePath)
			newCommands = append(newCommands, fileName)
		}
	}

	return newCommands
}

// trackDocDebt adds new commands to the doc debt tracker.
// Returns the number of newly tracked commands.
func trackDocDebt(commands []string) int {
	debt, err := userconfig.LoadDocDebt()
	if err != nil {
		// Silent failure - don't break completion for doc tracking issues
		return 0
	}

	newlyTracked := 0
	for _, cmd := range commands {
		if debt.AddCommand(cmd) {
			newlyTracked++
		}
	}

	if newlyTracked > 0 {
		if err := userconfig.SaveDocDebt(debt); err != nil {
			// Silent failure
			return 0
		}
	}

	return newlyTracked
}

// NotableChangelogEntry represents a notable change from the changelog.
type NotableChangelogEntry struct {
	Commit CommitInfo
	Reason string // Why this is notable (e.g., "BREAKING", "skill-relevant", "behavioral")
}

// detectNotableChangelogEntries checks recent commits across ecosystem repos for
// notable changes that the orchestrator should be aware of:
// - BREAKING changes
// - Behavioral changes (feat/fix commits)
// - Skill changes relevant to the agent's skill
// Returns formatted strings for display.
func detectNotableChangelogEntries(projectDir string, agentSkill string) []string {
	var entries []string

	// Get changelog data for last 3 days (recent enough to be relevant)
	result, err := GetChangelog(3, "all")
	if err != nil {
		return nil
	}

	// Iterate through commits looking for notable entries
	for _, dateCommits := range result.CommitsByDate {
		for _, commit := range dateCommits {
			var reasons []string

			// Check for BREAKING changes
			if commit.SemanticInfo.IsBreaking {
				reasons = append(reasons, "BREAKING")
			}

			// Check for behavioral changes (feat/fix)
			if commit.SemanticInfo.ChangeType == ChangeTypeBehavioral {
				// Only surface if it's in a category that could affect agents
				if commit.Category == "skills" || commit.Category == "skill-behavioral" ||
					commit.Category == "cmd" || commit.Category == "pkg" {
					reasons = append(reasons, "behavioral")
				}
			}

			// Check for skill-relevant changes
			if agentSkill != "" && isSkillRelevantChange(commit, agentSkill) {
				reasons = append(reasons, fmt.Sprintf("relevant to %s", agentSkill))
			}

			// If we have reasons, add to the list
			if len(reasons) > 0 {
				icon := "📌"
				if commit.SemanticInfo.IsBreaking {
					icon = "🚨"
				} else if strings.Contains(strings.Join(reasons, ","), "relevant to") {
					icon = "🎯"
				}

				entry := fmt.Sprintf("%s [%s] %s (%s)",
					icon,
					commit.Repo,
					truncateString(commit.Subject, 40),
					strings.Join(reasons, ", "))
				entries = append(entries, entry)
			}
		}
	}

	// Limit to top 5 most notable entries to avoid noise
	if len(entries) > 5 {
		entries = entries[:5]
	}

	return entries
}

// isSkillRelevantChange checks if a commit affects files related to a specific skill.
func isSkillRelevantChange(commit CommitInfo, skillName string) bool {
	for _, file := range commit.Files {
		// Check for skill-specific paths (handles both "skills/" prefix and "/skills/")
		if strings.Contains(file, "skills/") {
			// Check if this skill is mentioned in the path
			if strings.Contains(file, "/"+skillName+"/") ||
				strings.Contains(file, "/"+skillName+".") ||
				strings.HasPrefix(file, "skills/"+skillName+"/") ||
				strings.Contains(file, "/skills/"+skillName+"/") {
				return true
			}
		}

		// Check for SPAWN_CONTEXT or spawn package changes (affects all skills)
		if strings.Contains(file, "SPAWN_CONTEXT") ||
			strings.Contains(file, "pkg/spawn/") {
			return true
		}

		// Check for skill verification changes
		if strings.Contains(file, "pkg/verify/skill") {
			return true
		}
	}
	return false
}

// runAutoRebuild runs make install in the project directory.
func runAutoRebuild(projectDir string) error {
	cmd := exec.Command("make", "install")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// restartOrchServe checks if orch serve is running and restarts it.
// Returns true if it was restarted, false if it wasn't running.
func restartOrchServe(projectDir string) (bool, error) {
	// Find the orch serve process
	// We look for processes matching "orch serve" or "orch-go serve"
	cmd := exec.Command("pgrep", "-f", "orch.*serve")
	output, err := cmd.Output()
	if err != nil {
		// No process found - that's fine, just means serve isn't running
		return false, nil
	}

	pids := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(pids) == 0 || pids[0] == "" {
		return false, nil
	}

	// Get the current PID to avoid killing ourselves
	currentPID := os.Getpid()

	// Kill the serve process(es)
	var killedAny bool
	for _, pidStr := range pids {
		pid, err := strconv.Atoi(strings.TrimSpace(pidStr))
		if err != nil {
			continue
		}
		// Don't kill ourselves
		if pid == currentPID {
			continue
		}
		// Send SIGTERM for graceful shutdown
		killCmd := exec.Command("kill", "-TERM", pidStr)
		if err := killCmd.Run(); err == nil {
			killedAny = true
		}
	}

	if !killedAny {
		return false, nil
	}

	// Wait a moment for the process to stop
	time.Sleep(500 * time.Millisecond)

	// Start orch serve in the background
	// We use nohup to ensure it survives after we exit
	serveCmd := exec.Command("nohup", "orch", "serve")
	serveCmd.Dir = projectDir
	// Redirect output to files to avoid blocking
	devNull, _ := os.OpenFile("/dev/null", os.O_WRONLY, 0)
	serveCmd.Stdout = devNull
	serveCmd.Stderr = devNull
	if err := serveCmd.Start(); err != nil {
		return true, fmt.Errorf("killed old serve but failed to start new: %w", err)
	}

	return true, nil
}

// exportOrchestratorTranscript exports the session transcript for orchestrator sessions.
// It checks for .orchestrator marker, sends /export to the tmux window, waits for the
// export file, and moves it to the workspace as TRANSCRIPT.md.
func exportOrchestratorTranscript(workspacePath, projectDir, beadsID string) error {
	// Check if this is an orchestrator session (has .orchestrator or .meta-orchestrator marker)
	orchestratorMarker := filepath.Join(workspacePath, ".orchestrator")
	metaOrchestratorMarker := filepath.Join(workspacePath, ".meta-orchestrator")

	isOrchestrator := false
	if _, err := os.Stat(orchestratorMarker); err == nil {
		isOrchestrator = true
	} else if _, err := os.Stat(metaOrchestratorMarker); err == nil {
		isOrchestrator = true
	}

	if !isOrchestrator {
		return nil // Not an orchestrator, nothing to do
	}

	// Find the tmux window for this agent
	window, _, err := tmux.FindWindowByBeadsIDAllSessions(beadsID)
	if err != nil || window == nil {
		return fmt.Errorf("could not find tmux window for orchestrator")
	}

	// Record existing session export files before sending /export
	existingExports := make(map[string]bool)
	pattern := filepath.Join(projectDir, "session-ses_*.md")
	matches, _ := filepath.Glob(pattern)
	for _, m := range matches {
		existingExports[m] = true
	}

	// Send /export command to the tmux window
	if err := tmux.SendKeys(window.Target, "/export"); err != nil {
		return fmt.Errorf("failed to send /export: %w", err)
	}
	if err := tmux.SendEnter(window.Target); err != nil {
		return fmt.Errorf("failed to send enter: %w", err)
	}

	fmt.Println("Exporting orchestrator transcript...")

	// Wait for new export file to appear (poll for up to 10 seconds)
	var newExportPath string
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			if !existingExports[m] {
				newExportPath = m
				break
			}
		}
		if newExportPath != "" {
			break
		}
	}

	if newExportPath == "" {
		return fmt.Errorf("timeout waiting for export file")
	}

	// Move export to workspace as TRANSCRIPT.md
	destPath := filepath.Join(workspacePath, "TRANSCRIPT.md")
	if err := os.Rename(newExportPath, destPath); err != nil {
		// If rename fails (cross-device), try copy+delete
		input, err := os.ReadFile(newExportPath)
		if err != nil {
			return fmt.Errorf("failed to read export: %w", err)
		}
		if err := os.WriteFile(destPath, input, 0644); err != nil {
			return fmt.Errorf("failed to write transcript: %w", err)
		}
		os.Remove(newExportPath)
	}

	fmt.Printf("Saved transcript: %s\n", destPath)
	return nil
}
