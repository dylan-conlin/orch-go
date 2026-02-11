// Package main provides the complete command for completing agents and closing beads issues.
//
// Architecture: runComplete() is a thin pipeline orchestrator that delegates to phase
// functions in complete_pipeline.go. Each phase has typed I/O for isolated testability.
//
// Pipeline phases (see complete_pipeline.go):
//  1. resolveTarget:      identifier → CompletionTarget
//  2. verifyCompletion:   target + skipConfig → VerificationOutcome
//  3. checkLiveness:      target → (prompt or continue)
//  4. processGates:       target → (discovered work, knowledge gaps)
//  5. integrateAgentBranch: target → (cherry-pick onto base branch)
//  6. closeIssue:         target + skipConfig → reason string
//  7. runCleanup:         target → CleanupOutcome
//  8. postComplete:       target + outcomes + telemetry → (events, cache)
//
// Related files:
//   - complete_pipeline.go:  Pipeline types and phase implementations
//   - complete_pipeline_test.go: Unit tests for each phase
//   - complete_verify.go:    SkipConfig type and verification skip logic
//   - complete_actions.go:   Post-completion actions (archival, transcript, telemetry, cache)
//   - complete_helpers.go:   Changelog detection, auto-rebuild, CLI command detection, display
package main

import (
	"fmt"
	"os"
	"strings"

	statedb "github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
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
	completeBatch            bool // Batch mode: run only Tier 1 (core) gates

	// Targeted skip flags (replace blanket --force)
	// Each requires completeSkipReason to be set (min 10 chars)
	completeSkipTestEvidence     bool
	completeSkipModelConnection  bool
	completeSkipVisual           bool
	completeSkipGitDiff          bool
	completeSkipSynthesis        bool
	completeSkipBuild            bool
	completeSkipConstraint       bool
	completeSkipPhaseGate        bool
	completeSkipSkillOutput      bool
	completeSkipDecisionPatch    bool
	completeSkipPhaseComplete    bool
	completeSkipAgentRunning     bool
	completeSkipHandoffContent   bool
	completeSkipDashboardHealth  bool
	completeSkipVerificationSpec bool
	completeSkipCommitEvidence   bool
	completeSkipReason           string // Required for all --skip-* flags (min 10 chars)
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
  - test_evidence:        Test execution evidence (feature-impl/systematic-debugging/reliability-testing)
  - model_connection:     Probe file or Model candidate evidence (investigation/research/architect)
  - visual_verification:  Visual verification for web/ changes
  - git_diff:             Git changes match SYNTHESIS.md claims
  - build:                Project builds successfully
  - constraint:           Skill constraints satisfied
  - phase_gate:           Required skill phases completed
  - skill_output:         Required skill outputs exist
  - decision_patch_limit: Decision patch count not exceeded
  - handoff_content:      SYNTHESIS.md has actual content (orchestrator only)
  - dashboard_health:     Dashboard API endpoints healthy for web/ or serve_*.go changes
  - verification_spec:    VERIFICATION_SPEC executable checks pass for workspace tier
  - commit_evidence:      Agent branch has at least one commit (prevents ghost completions)
  - branch_integration:   Agent branch rebased + merged fast-forward into base branch

TARGETED SKIP FLAGS:
Use --skip-{gate} with --skip-reason to bypass specific gates:
  --skip-test-evidence     Skip test evidence gate
  --skip-model-connection  Skip model connection gate
  --skip-visual            Skip visual verification gate
  --skip-git-diff          Skip git diff verification gate
  --skip-synthesis         Skip SYNTHESIS.md gate
  --skip-build             Skip build verification gate
  --skip-constraint        Skip constraint verification gate
  --skip-phase-gate        Skip phase gate verification
  --skip-skill-output      Skip skill output verification gate
  --skip-decision-patch    Skip decision patch count gate
  --skip-phase-complete    Skip Phase: Complete gate
  --skip-agent-running     Skip liveness gate when agent appears active
  --skip-handoff-content   Skip handoff content validation (orchestrator only)
  --skip-dashboard-health  Skip dashboard health check for web/ or serve_*.go changes
  --skip-verification-spec Skip VERIFICATION_SPEC executable checks gate
  --skip-commit-evidence   Skip commit evidence gate (allow zero-commit completion)

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
	// Wire up state.db fallback for Phase: Complete checks.
	// This breaks the circular dependency between verify and state packages.
	verify.StateDBPhaseChecker = func(beadsID string) (string, string, bool, error) {
		db, err := statedb.OpenDefault()
		if err != nil || db == nil {
			return "", "", false, err
		}
		defer db.Close()
		agent, err := db.GetAgentByBeadsID(beadsID)
		if err != nil {
			return "", "", false, err
		}
		if agent == nil || strings.TrimSpace(agent.Phase) == "" {
			return "", "", false, nil
		}
		return agent.Phase, agent.PhaseSummary, true, nil
	}

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
	completeCmd.Flags().BoolVar(&completeBatch, "batch", false, "Batch mode: run only Tier 1 (core) gates, skip Tier 2 (quality) gates")

	// Targeted skip flags - each bypasses a specific verification gate
	completeCmd.Flags().BoolVar(&completeSkipTestEvidence, "skip-test-evidence", false, "Skip test execution evidence gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipModelConnection, "skip-model-connection", false, "Skip model connection gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipVisual, "skip-visual", false, "Skip visual verification gate for web/ changes (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipGitDiff, "skip-git-diff", false, "Skip git diff verification gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipSynthesis, "skip-synthesis", false, "Skip SYNTHESIS.md verification gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipBuild, "skip-build", false, "Skip project build verification gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipConstraint, "skip-constraint", false, "Skip constraint verification gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipPhaseGate, "skip-phase-gate", false, "Skip phase gate verification (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipSkillOutput, "skip-skill-output", false, "Skip skill output verification gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipDecisionPatch, "skip-decision-patch", false, "Skip decision patch count verification gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipPhaseComplete, "skip-phase-complete", false, "Skip Phase: Complete verification gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipAgentRunning, "skip-agent-running", false, "Skip liveness gate when agent appears active (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipHandoffContent, "skip-handoff-content", false, "Skip handoff content validation gate for orchestrators (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipDashboardHealth, "skip-dashboard-health", false, "Skip dashboard health check gate for web/ or serve_*.go changes (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipVerificationSpec, "skip-verification-spec", false, "Skip VERIFICATION_SPEC executable checks gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipCommitEvidence, "skip-commit-evidence", false, "Skip commit evidence gate - allows completion with zero commits (requires --skip-reason)")
	completeCmd.Flags().StringVar(&completeSkipReason, "skip-reason", "", "Reason for skip (required for all --skip-* flags, min 10 chars)")
}

// runComplete is the pipeline orchestrator for agent completion.
// It delegates to phase functions in complete_pipeline.go, each with typed I/O.
func runComplete(identifier, workdir string) error {
	// Validate skip flags before doing anything else
	skipConfig := getSkipConfig()
	if err := validateSkipFlags(skipConfig); err != nil {
		return err
	}

	// Pre-pipeline notices
	if skipConfig.BatchMode {
		fmt.Println("Batch mode: running Tier 1 (core) gates only, skipping Tier 2 (quality) gates")
	}
	if completeForce {
		fmt.Fprintln(os.Stderr, "DEPRECATED: --force is deprecated. Use targeted --skip-* flags instead.")
		fmt.Fprintln(os.Stderr, "   Example: --skip-test-evidence --skip-reason \"Tests run in CI\"")
		fmt.Fprintln(os.Stderr, "   This flag will be removed in a future version.")
		fmt.Fprintln(os.Stderr)
	}

	// Phase 1: Resolve target
	target, err := resolveTarget(identifier, workdir)
	if err != nil {
		return err
	}

	// Pre-verification: approval comment + auto-rebuild
	if completeApprove && !target.IsUntracked {
		approvalComment := "APPROVED - Visual changes reviewed and approved by orchestrator"
		if err := addApprovalComment(target.BeadsID, approvalComment); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to add approval comment: %v\n", err)
		} else {
			fmt.Printf("Added approval: %s\n", approvalComment)
		}
	}
	rebuildGoProjectsIfNeeded(target.gitDir(), target.WorkspacePath)

	// Phase 2: Verify completion
	vOutcome, err := verifyCompletion(target, skipConfig)
	if err != nil {
		return err
	}

	// Phase 3: Check liveness
	if err := checkLiveness(target, skipConfig); err != nil {
		return err
	}

	// Phase 4: Process gates (discovered work, knowledge gaps)
	if err := processGates(target, vOutcome.SkillName); err != nil {
		return err
	}

	// Phase 5: Integrate branch (cherry-pick onto base branch)
	if err := integrateAgentBranch(target); err != nil {
		return err
	}

	// Phase 6: Close issue
	reason, err := closeIssue(target, skipConfig)
	if err != nil {
		return err
	}

	// Collect telemetry BEFORE cleanup (needs live session + workspace)
	var telemetry CompletionTelemetry
	if target.WorkspacePath != "" {
		telemetry.DurationSecs, telemetry.TokensIn, telemetry.TokensOut, telemetry.Outcome =
			collectCompletionTelemetry(target.WorkspacePath, completeForce, vOutcome.Passed)
	}

	// Phase 7: Cleanup (session deletion, archival, docker, tmux)
	_ = runCleanup(target)

	// Phase 8: Post-complete (CLI commands, changelog, events, cache)
	postComplete(target, vOutcome, reason, telemetry)

	// Silence unused variable warnings for disabled reproduction gate
	_ = completeSkipReproCheck
	_ = completeSkipReproReason

	return nil
}
