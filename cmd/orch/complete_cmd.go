// Package main provides the complete command for completing agents and closing beads issues.
// Extracted from main.go as part of the main.go refactoring (Phase 3).
//
// Pipeline phases are in complete_pipeline.go:
//   resolveCompletionTarget → executeVerificationGates → runCompletionAdvisories → executeLifecycleTransition
// Helper functions are in complete_actions.go.
// Post-lifecycle helpers (cache, rebuild, telemetry, accretion) are in complete_postlifecycle.go.
package main

import (
	"fmt"
	"os"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
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

	// Targeted skip flags (replace blanket --force)
	// Each requires completeSkipReason to be set (min 10 chars)
	completeSkipTestEvidence   bool
	completeSkipVisual         bool
	completeSkipGitDiff        bool
	completeSkipSynthesis      bool
	completeSkipBuild          bool
	completeSkipConstraint     bool
	completeSkipPhaseGate      bool
	completeSkipSkillOutput    bool
	completeSkipDecisionPatch  bool
	completeSkipPhaseComplete  bool
	completeSkipHandoffContent bool
	completeSkipExplainBack    bool
	completeSkipAccretion           bool
	completeSkipArchitecturalChoices bool
	completeSkipProbeModelMerge     bool
	completeSkipArchitectHandoff    bool
	completeSkipReason              string // Required for all --skip-* flags (min 10 chars)

	// Explain-back flag: orchestrator provides explanation text
	completeExplain string

	// Behavioral verification flag: orchestrator confirms agent behavior verified
	completeVerified bool

	// Review tier override flag
	completeReviewTier string
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
  - architect_handoff:    Architect SYNTHESIS.md has explicit **Recommendation:** field
  - handoff_content:      SESSION_HANDOFF.md has actual content (orchestrator only)
  - explain_back:         Orchestrator provides --explain text (gate1: comprehension)
  - verified:             Orchestrator confirms behavioral verification (gate2)

TIER-AWARE VERIFICATION:
Checkpoint requirements are inferred from beads issue type:
  Tier 1 (feature/bug/decision): Both gate1 (--explain) and gate2 (--verified) required
  Tier 2 (investigation/probe):  Gate1 (--explain) only
  Tier 3 (task/question/other):  No checkpoint required

EXPLAIN-BACK GATE (gate1 - comprehension):
The explain-back gate requires the orchestrator to provide an explanation of what
was built via --explain. The conversational quality check stays with the AI
orchestrator - the CLI only gates on non-empty text.

  orch complete proj-123 --explain 'Built X because Y, verified by Z'

BEHAVIORAL VERIFICATION (gate2 - verified):
For Tier 1 work, the --verified flag confirms the orchestrator has verified
the agent's actual behavior, not just comprehended what was built.

  orch complete proj-123 --explain '...' --verified

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
  --skip-handoff-content  Skip handoff content validation (orchestrator only)
  --skip-explain-back     Skip explain-back verification gate
  --skip-accretion        Skip accretion (file size growth) gate

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

For cross-project completion, the beads project (where the issue lives) is derived
from the beads ID prefix, and the work project (where the agent worked) is derived
from the workspace manifest. These are resolved independently, so completion works
even when the issue and workspace live in different repos.
Use --workdir to explicitly override the work project directory.

Examples:
  orch-go complete proj-123 --explain 'Reworked auth to use JWT tokens' --verified
  orch-go complete proj-123 --explain 'Fixed login bug' --verified --reason "All tests passing"
  orch-go complete proj-123 --approve --explain 'Added dark mode toggle' --verified
  orch-go complete proj-123 --skip-test-evidence --skip-reason "Tests run in CI" --explain 'Refactored config'
  orch-go complete proj-123 --skip-explain-back --skip-reason "Automated completion, no human review"
  orch-go complete kb-cli-123 --workdir ~/projects/kb-cli --explain 'Cross-project fix' --verified
  orch-go complete proj-123 --verified  # Add gate2 when gate1 already recorded

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
	completeCmd.Flags().BoolVar(&completeNoArchive, "no-archive", false, "Skip automatic workspace archival after completion")

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
	completeCmd.Flags().BoolVar(&completeSkipExplainBack, "skip-explain-back", false, "Skip explain-back verification gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipAccretion, "skip-accretion", false, "Skip accretion (file size growth) verification gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipArchitecturalChoices, "skip-architectural-choices", false, "Skip architectural choices verification gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipProbeModelMerge, "skip-probe-model-merge", false, "Skip probe-to-model merge gate (requires --skip-reason)")
	completeCmd.Flags().BoolVar(&completeSkipArchitectHandoff, "skip-architect-handoff", false, "Skip architect handoff (recommendation) gate (requires --skip-reason)")
	completeCmd.Flags().StringVar(&completeSkipReason, "skip-reason", "", "Reason for skip (required for all --skip-* flags, min 10 chars)")

	// Explain-back flag
	completeCmd.Flags().StringVar(&completeExplain, "explain", "", "Explanation of what was built and why (required by explain-back gate)")

	// Behavioral verification flag (gate2)
	completeCmd.Flags().BoolVar(&completeVerified, "verified", false, "Record behavioral verification (gate2) - confirms orchestrator verified agent behavior (required for Tier 1 work)")

	// Review tier override
	completeCmd.Flags().StringVar(&completeReviewTier, "review-tier", "", "Override review tier (auto/scan/review/deep) — overrides manifest value")
}

// getSkipConfig builds the skip configuration from command-line flags.
func getSkipConfig() verify.SkipConfig {
	return verify.SkipConfig{
		TestEvidence:   completeSkipTestEvidence,
		Visual:         completeSkipVisual,
		GitDiff:        completeSkipGitDiff,
		Synthesis:      completeSkipSynthesis,
		Build:          completeSkipBuild,
		Constraint:     completeSkipConstraint,
		PhaseGate:      completeSkipPhaseGate,
		SkillOutput:    completeSkipSkillOutput,
		DecisionPatch:  completeSkipDecisionPatch,
		PhaseComplete:  completeSkipPhaseComplete,
		HandoffContent: completeSkipHandoffContent,
		ExplainBack:          completeSkipExplainBack,
		Accretion:            completeSkipAccretion,
		ArchitecturalChoices: completeSkipArchitecturalChoices,
		ProbeModelMerge:      completeSkipProbeModelMerge,
		ArchitectHandoff:     completeSkipArchitectHandoff,
		Reason:               completeSkipReason,
	}
}

// logSkipEvents logs verification.bypassed events for all skipped gates.
func logSkipEvents(skipConfig verify.SkipConfig, beadsID, workspace, skill string) {
	logger := events.NewLogger(events.DefaultLogPath())
	for _, gate := range skipConfig.SkippedGates() {
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

// runComplete orchestrates the completion pipeline:
// 1. resolveCompletionTarget — find workspace, beads ID, project directory
// 2. executeVerificationGates — checkpoint, verification, liveness checks
// 3. runCompletionAdvisories — discovered work, probes, explain-back, checklist
// 4. executeLifecycleTransition — close issue, archive, post-lifecycle operations
func runComplete(identifier, workdir string) error {
	// Validate skip flags before doing anything else
	skipConfig := getSkipConfig()
	if err := verify.ValidateSkipFlags(skipConfig); err != nil {
		return err
	}

	// Show deprecation warning for --force and validate --reason requirement
	if completeForce {
		if completeReason == "" {
			return fmt.Errorf("--reason is required when using --force (min 10 chars)")
		}
		if len(completeReason) < 10 {
			return fmt.Errorf("--reason must be at least 10 characters (got %d)", len(completeReason))
		}
		fmt.Fprintln(os.Stderr, "⚠️  DEPRECATED: --force is deprecated. Use targeted --skip-* flags instead.")
		fmt.Fprintln(os.Stderr, "   Example: --skip-test-evidence --skip-reason \"Tests run in CI\"")
		fmt.Fprintln(os.Stderr, "   This flag will be removed in a future version.")
		fmt.Fprintln(os.Stderr)
	}

	// Phase 1: Resolve completion target
	target, err := resolveCompletionTarget(identifier, workdir)
	if err != nil {
		return err
	}

	// Resolve effective review tier: flag override > workspace manifest > default
	if completeReviewTier != "" {
		if !spawn.IsValidReviewTier(completeReviewTier) {
			return fmt.Errorf("invalid review tier %q: must be auto, scan, review, or deep", completeReviewTier)
		}
		target.ReviewTier = completeReviewTier
		fmt.Printf("Review tier: %s (override)\n", target.ReviewTier)
	} else if target.WorkspacePath != "" && !target.IsOrchestratorSession {
		target.ReviewTier = verify.ReadReviewTierFromWorkspace(target.WorkspacePath)
		fmt.Printf("Review tier: %s\n", target.ReviewTier)
	} else {
		target.ReviewTier = spawn.ReviewReview // Conservative default
		fmt.Printf("Review tier: %s (default)\n", target.ReviewTier)
	}

	// Review tier escalation: check if completion signals warrant a higher review tier.
	// Only run for non-orchestrator sessions with a workspace and no explicit --review-tier override.
	if completeReviewTier == "" && target.WorkspacePath != "" && !target.IsOrchestratorSession {
		signals := verify.BuildEscalationSignals(target.WorkspacePath, target.WorkProjectDir)
		// Add hotspot match count (hotspot analysis lives in cmd/orch, not pkg/verify)
		if target.WorkProjectDir != "" {
			signals.HotspotMatchCount = countHotspotAdvisoryMatches(target.WorkProjectDir, target.WorkspacePath)
		}
		escalation := verify.CheckReviewTierEscalation(signals, target.ReviewTier)
		if escalation.Escalated {
			fmt.Printf("Review tier escalated: %s → %s\n", escalation.OriginalTier, escalation.EscalatedTier)
			for _, reason := range escalation.Reasons {
				fmt.Printf("  - %s\n", reason)
			}
			target.ReviewTier = escalation.EscalatedTier

			// Log escalation event
			logger := events.NewLogger(events.DefaultLogPath())
			if err := logger.LogReviewTierEscalated(events.ReviewTierEscalatedData{
				BeadsID:      target.BeadsID,
				Workspace:    target.AgentName,
				OriginalTier: escalation.OriginalTier,
				EscalatedTo:  escalation.EscalatedTier,
				Reasons:      escalation.Reasons,
			}); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to log tier escalation event: %v\n", err)
			}
		}
	}

	// Phase 2: Execute verification gates
	outcome, err := executeVerificationGates(target, skipConfig)
	if err != nil {
		// Don't kill tmux window on gate failure — preserve evidence for recovery.
		// The window stays alive so the orchestrator can inspect, re-run gates,
		// or send follow-up messages without re-spawning.
		return err
	}

	// Phase 3: Run completion advisories
	advisories, err := runCompletionAdvisories(target, outcome, skipConfig)
	if err != nil {
		return err
	}

	// Phase 4: Execute lifecycle transition
	lifecycleCleanedUp, err := executeLifecycleTransition(target, outcome, advisories)
	if err != nil {
		return err
	}

	// Clean up tmux window only after all gates pass and lifecycle transition succeeds.
	// Skip if LifecycleManager already handled cleanup (it kills the window as part of Complete()).
	if !lifecycleCleanedUp {
		cleanupTmuxWindow(target.IsOrchestratorSession, target.AgentName, target.BeadsID, identifier)
	}

	return nil
}
