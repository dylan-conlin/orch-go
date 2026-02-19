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

	"github.com/dylan-conlin/orch-go/pkg/activity"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/checkpoint"
	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/orch"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
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
	completeSkipReason         string // Required for all --skip-* flags (min 10 chars)

	// Explain-back flag: orchestrator provides explanation text
	completeExplain string

	// Behavioral verification flag: orchestrator confirms agent behavior verified
	completeVerified bool
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
	completeCmd.Flags().StringVar(&completeSkipReason, "skip-reason", "", "Reason for skip (required for all --skip-* flags, min 10 chars)")

	// Explain-back flag
	completeCmd.Flags().StringVar(&completeExplain, "explain", "", "Explanation of what was built and why (required by explain-back gate)")

	// Behavioral verification flag (gate2)
	completeCmd.Flags().BoolVar(&completeVerified, "verified", false, "Record behavioral verification (gate2) - confirms orchestrator verified agent behavior (required for Tier 1 work)")
}

// SkipConfig holds the configuration for which verification gates to skip.
type SkipConfig struct {
	TestEvidence   bool
	Visual         bool
	GitDiff        bool
	Synthesis      bool
	Build          bool
	Constraint     bool
	PhaseGate      bool
	SkillOutput    bool
	DecisionPatch  bool
	PhaseComplete  bool
	HandoffContent bool
	ExplainBack    bool
	Reason         string // Required reason for skips
}

// hasAnySkip returns true if any skip flag is set.
func (c SkipConfig) hasAnySkip() bool {
	return c.TestEvidence || c.Visual || c.GitDiff || c.Synthesis ||
		c.Build || c.Constraint || c.PhaseGate || c.SkillOutput ||
		c.DecisionPatch || c.PhaseComplete || c.HandoffContent || c.ExplainBack
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
	if c.HandoffContent {
		gates = append(gates, verify.GateHandoffContent)
	}
	if c.ExplainBack {
		gates = append(gates, verify.GateExplainBack)
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
	case verify.GateHandoffContent:
		return c.HandoffContent
	case verify.GateExplainBack:
		return c.ExplainBack
	default:
		return false
	}
}

// getSkipConfig builds the skip configuration from command-line flags.
func getSkipConfig() SkipConfig {
	return SkipConfig{
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
		ExplainBack:    completeSkipExplainBack,
		Reason:         completeSkipReason,
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
	// 1. Try to find workspace by name in current directory
	// 2. If it looks like a workspace name, search known projects
	// 3. Fall back to beads ID lookup for worker sessions
	var workspacePath, agentName string
	var beadsID string
	var isOrchestratorSession bool

	// Step 1: Try direct workspace name lookup in current directory
	if workspacePath == "" {
		directWorkspacePath := findWorkspaceByName(currentDir, identifier)
		if directWorkspacePath != "" {
			workspacePath = directWorkspacePath
			agentName = identifier
			// Check if this is an orchestrator workspace (no beads tracking)
			if isOrchestratorWorkspace(workspacePath) {
				isOrchestratorSession = true
				fmt.Printf("Orchestrator session: %s\n", agentName)
			} else {
				// Non-orchestrator workspace found by name - read beads ID from manifest
				manifest := spawn.ReadAgentManifestWithFallback(workspacePath)
				beadsID = manifest.BeadsID
			}
		}
	}

	// Step 2: Search for workspace across known projects if identifier looks like a workspace name
	if workspacePath == "" && looksLikeWorkspaceName(identifier) {
		if foundPath := findWorkspaceByNameAcrossProjects(identifier); foundPath != "" {
			workspacePath = foundPath
			agentName = identifier
			if isOrchestratorWorkspace(workspacePath) {
				isOrchestratorSession = true
				fmt.Printf("Orchestrator session (cross-project): %s\n", agentName)
			} else {
				manifest := spawn.ReadAgentManifestWithFallback(workspacePath)
				beadsID = manifest.BeadsID
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

	// Checkpoint verification gate (Verifiability-first enforcement)
	// Tier-aware: Tier 1 requires both gates, Tier 2 requires gate1 only, Tier 3 no checkpoint.
	if !isUntracked && !completeForce && issue != nil {
		tier := checkpoint.TierForIssueType(issue.IssueType)

		if checkpoint.RequiresCheckpoint(issue.IssueType) {
			// Gate 1 (comprehension) check
			hasGate1, err := checkpoint.HasGate1Checkpoint(beadsID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to check verification checkpoint: %v\n", err)
				// Continue with completion - checkpoint check is advisory for now
			} else if !hasGate1 && !skipConfig.ExplainBack && completeExplain == "" {
				// No checkpoint exists, explain-back not being skipped, and no --explain text provided
				fmt.Fprintf(os.Stderr, "❌ Comprehension gate (gate1) missing for Tier %d work (%s)\n", tier, issue.IssueType)
				fmt.Fprintf(os.Stderr, "\nTier %d work requires comprehension verification:\n", tier)
				fmt.Fprintf(os.Stderr, "  orch complete %s --explain 'Built X because Y, verified by Z'\n", beadsID)
				fmt.Fprintf(os.Stderr, "\nOr bypass with:\n")
				fmt.Fprintf(os.Stderr, "  --skip-explain-back --skip-reason \"...\"\n")
				return fmt.Errorf("verification checkpoint required for Tier %d work", tier)
			} else if hasGate1 {
				fmt.Println("✓ Comprehension gate (gate1) passed")
			}
		}

		// Gate 2 (behavioral) check - Tier 1 only
		if checkpoint.RequiresGate2(issue.IssueType) {
			hasGate2, err := checkpoint.HasGate2Checkpoint(beadsID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to check gate2 checkpoint: %v\n", err)
			} else if !hasGate2 && !completeVerified {
				fmt.Fprintf(os.Stderr, "❌ Behavioral verification (gate2) missing for Tier 1 work (%s)\n", issue.IssueType)
				fmt.Fprintf(os.Stderr, "\nTier 1 work (features/bugs/decisions) requires behavioral verification:\n")
				fmt.Fprintf(os.Stderr, "  orch complete %s --verified --explain '...'\n", beadsID)
				fmt.Fprintf(os.Stderr, "\nThe --verified flag confirms the orchestrator has verified the agent's behavior,\n")
				fmt.Fprintf(os.Stderr, "not just comprehended what was built.\n")
				return fmt.Errorf("behavioral verification (gate2) required for Tier 1 work")
			} else if hasGate2 {
				fmt.Println("✓ Behavioral verification (gate2) passed")
			}
		}
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
	var completionResult verify.VerificationResult
	var completionResultSet bool

	// Verify completion status
	// - For orchestrator sessions: check SESSION_HANDOFF.md exists AND has content
	// - For regular agents: check Phase: Complete via beads comments
	// - Skip flags allow targeted bypass of specific gates
	if !completeForce {
		if isOrchestratorSession {
			// Orchestrator sessions use SESSION_HANDOFF.md as completion signal
			// Use full verification which includes content validation
			if workspacePath != "" {
				fmt.Printf("Workspace: %s\n", agentName)
			}

			result := verify.VerifyOrchestratorCompletion(workspacePath)
			skillName = result.Skill
			completionResult = result
			completionResultSet = true

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

				fmt.Fprintf(os.Stderr, "Cannot complete orchestrator session - verification failed:\n")
				for _, e := range result.Errors {
					fmt.Fprintf(os.Stderr, "  - %s\n", e)
				}
				fmt.Fprintf(os.Stderr, "\nOrchestrator must fill SESSION_HANDOFF.md with:\n")
				fmt.Fprintf(os.Stderr, "  - TLDR section (actual content, not placeholder)\n")
				fmt.Fprintf(os.Stderr, "  - Outcome field (success, partial, blocked, or failed)\n")
				fmt.Fprintf(os.Stderr, "Or use --skip-handoff-content --skip-reason \"...\" to bypass\n")
				return fmt.Errorf("verification failed")
			}
			fmt.Println("Completion signal: SESSION_HANDOFF.md verified (content validated)")
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
			completionResult = result
			completionResultSet = true

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

			// Surface model references for modified files (informational only)
			if workspacePath != "" && beadsProjectDir != "" {
				matches, err := verify.FindModelReferencesForModifiedFiles(workspacePath, beadsProjectDir)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to check model code references: %v\n", err)
				} else if note := verify.FormatModelReferenceNote(matches); note != "" {
					fmt.Println(note)
				}
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
				completionResult = result
				completionResultSet = true
				if !result.Passed {
					verificationPassed = false
					gatesFailed = result.GatesFailed
				}
			}
		} else if isOrchestratorSession {
			// Run verification to capture what would have failed
			result := verify.VerifyOrchestratorCompletion(workspacePath)
			skillName = result.Skill
			completionResult = result
			completionResultSet = true
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

	// Surface probe verdicts for orchestrator review
	// Probes in .kb/models/*/probes/ produced during this agent's session
	// contain Model Impact verdicts that need to be surfaced for model merging.
	if workspacePath != "" {
		probeVerdicts := verify.FindProbesForWorkspace(workspacePath, beadsProjectDir)
		if len(probeVerdicts) > 0 {
			fmt.Print(verify.FormatProbeVerdicts(probeVerdicts))
		}
	}

	// Explain-back verification gate
	// After all verification passes, require human to explain what was built and why.
	// This creates an unfakeable verification gate - can't rubber-stamp a conversational explanation.
	// Extracted to pkg/orch/completion.go for reusability.
	// The gate prompts for explanation AND stores it as a beads comment internally.
	//
	// Skip if gate1 already exists and no --explain provided (gate2-only update path).
	// In that case, the user already explained in a previous run.
	priorGate1, _ := checkpoint.HasGate1Checkpoint(beadsID)
	if completeExplain != "" || !priorGate1 {
		if err := orch.RunExplainBackGate(
			beadsID,
			completeForce,
			skipConfig.ExplainBack,
			skipConfig.Reason,
			isOrchestratorSession,
			isUntracked,
			completeExplain,
			completeVerified,
			os.Stdout,
		); err != nil {
			return err
		}

	}

	// Record gate2 checkpoint if --verified flag is set and explain-back gate didn't run
	// (gate1 already existed from a previous completion attempt)
	if completeVerified && !isUntracked && !isOrchestratorSession && beadsID != "" {
		hasGate2, _ := checkpoint.HasGate2Checkpoint(beadsID)
		if !hasGate2 && priorGate1 && completeExplain == "" {
			// Gate1 exists, gate2 doesn't, explain-back didn't run → record gate2 standalone
			if err := orch.RecordGate2Checkpoint(beadsID, os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to record gate2 checkpoint: %v\n", err)
			}
		}
	}

	// Surface verification checklist before closing
	if completionResultSet && !isUntracked {
		gate1Complete := false
		gate2Complete := false
		if beadsID != "" && !isOrchestratorSession {
			gate1Complete, _ = checkpoint.HasGate1Checkpoint(beadsID)
			gate2Complete, _ = checkpoint.HasGate2Checkpoint(beadsID)
		}
		tier := ""
		if workspacePath != "" && !isOrchestratorSession {
			tier = verify.ReadTierFromWorkspace(workspacePath)
		}
		issueType := ""
		if issue != nil {
			issueType = issue.IssueType
		}
		checklist := buildVerificationChecklist(completionResult, issueType, tier, isOrchestratorSession, skipConfig, gate1Complete, gate2Complete)
		printVerificationChecklist(checklist)
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

		// Signal human verification to daemon.
		// This resets the completion counter and unpauses the daemon if it was paused.
		// We use a file-based signal so orch complete doesn't need direct access to the daemon instance.
		if err := daemon.WriteVerificationSignal(); err != nil {
			// Log warning but don't fail completion - the issue is already closed
			fmt.Fprintf(os.Stderr, "Warning: failed to signal human verification to daemon: %v\n", err)
		}
	} else if isOrchestratorSession {
		fmt.Printf("Completed orchestrator session: %s\n", agentName)
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

		}
	} else if completeNoArchive && workspacePath != "" {
		fmt.Println("Skipped workspace archival (--no-archive)")
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

	// Collect telemetry (duration and tokens) for model performance tracking
	var durationSecs, tokensIn, tokensOut int
	var outcome string
	if workspacePath != "" {
		durationSecs, tokensIn, tokensOut, outcome = collectCompletionTelemetry(workspacePath, completeForce, verificationPassed)
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

	// Collect and log accretion delta metrics (file growth/shrinkage tracking)
	// This helps monitor whether accretion gravity is being inverted over time.
	// Non-blocking - failures are silently ignored.
	if workspacePath != "" && beadsProjectDir != "" {
		if accretionData := collectAccretionDelta(beadsProjectDir, workspacePath); accretionData != nil {
			// Populate metadata
			accretionData.BeadsID = beadsID
			accretionData.Workspace = agentName
			accretionData.Skill = skillName

			// Log the accretion delta event
			if err := logger.LogAccretionDelta(*accretionData); err != nil {
				// Silent failure - accretion tracking is nice-to-have, not critical
				fmt.Fprintf(os.Stderr, "Warning: failed to log accretion delta: %v\n", err)
			}
		}
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
//
// When running under Overmind (dashboard), uses "overmind restart api" to avoid
// tearing down the whole process group (web, opencode). Falls back to kill+nohup
// for standalone orch serve.
func restartOrchServe(projectDir string) (bool, error) {
	// Check if Overmind is managing orch serve by looking for its socket.
	// When orch-dashboard starts services, Overmind creates .overmind.sock in the project dir.
	// Killing the api process directly causes Overmind to tear down all services (web, opencode).
	overmindSock := filepath.Join(projectDir, ".overmind.sock")
	if _, err := os.Stat(overmindSock); err == nil {
		cmd := exec.Command("overmind", "restart", "api")
		cmd.Dir = projectDir
		if err := cmd.Run(); err != nil {
			return false, fmt.Errorf("overmind restart api failed: %w", err)
		}
		return true, nil
	}

	// Fallback: not under Overmind, use pgrep + kill + nohup

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

func looksLikeWorkspaceName(identifier string) bool {
	return strings.HasPrefix(identifier, "og-") ||
		strings.HasPrefix(identifier, "meta-") ||
		strings.HasPrefix(identifier, "orch-")
}

func findWorkspaceByNameAcrossProjects(workspaceName string) string {
	for _, project := range getKBProjectsWithNames() {
		if wsPath := findWorkspaceByName(project.Path, workspaceName); wsPath != "" {
			return wsPath
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	rootCandidates := []string{
		filepath.Join(homeDir, "Documents", "personal"),
		filepath.Join(homeDir, "projects"),
		filepath.Join(homeDir, "src"),
	}

	for _, root := range rootCandidates {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			projectDir := filepath.Join(root, entry.Name())
			if wsPath := findWorkspaceByName(projectDir, workspaceName); wsPath != "" {
				return wsPath
			}
		}
	}

	return ""
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

// archiveWorkspace moves a completed workspace to the archived directory.
// Returns the new archived path on success, or an error if archival fails.
// The function handles name collisions by adding a timestamp suffix.
func archiveWorkspace(workspacePath, projectDir string) (string, error) {
	if workspacePath == "" {
		return "", fmt.Errorf("workspace path is empty")
	}

	// Verify workspace exists
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		return "", fmt.Errorf("workspace does not exist: %s", workspacePath)
	}

	// Determine workspace name and archived directory
	workspaceName := filepath.Base(workspacePath)
	archivedDir := filepath.Join(projectDir, ".orch", "workspace", "archived")

	// Create archived directory if needed
	if err := os.MkdirAll(archivedDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create archived directory: %w", err)
	}

	// Determine destination path
	destPath := filepath.Join(archivedDir, workspaceName)

	// Handle name collision (if archive already exists, add timestamp suffix)
	if _, err := os.Stat(destPath); err == nil {
		suffix := time.Now().Format("150405") // HHMMSS format
		destPath = filepath.Join(archivedDir, workspaceName+"-"+suffix)
		fmt.Printf("Note: Archive destination exists, using: %s-%s\n", workspaceName, suffix)
	}

	// Move workspace to archived
	if err := os.Rename(workspacePath, destPath); err != nil {
		return "", fmt.Errorf("failed to archive workspace: %w", err)
	}

	return destPath, nil
}

// collectCompletionTelemetry collects duration and token usage for telemetry.
// Returns (durationSeconds, tokensInput, tokensOutput, outcome).
// Returns zeros if telemetry collection fails (non-blocking).
func collectCompletionTelemetry(workspacePath string, forced bool, verificationPassed bool) (int, int, int, string) {
	var durationSeconds int
	var tokensInput int
	var tokensOutput int
	var outcome string

	// Determine outcome
	if forced {
		outcome = "forced"
	} else if verificationPassed {
		outcome = "success"
	} else {
		outcome = "failed"
	}

	// Read spawn time from manifest (falls back to dotfiles)
	manifest := spawn.ReadAgentManifestWithFallback(workspacePath)
	if spawnTime := manifest.ParseSpawnTime(); !spawnTime.IsZero() {
		durationSeconds = int(time.Since(spawnTime).Seconds())
	}

	// Read session ID from workspace (.session_id stays separate - infrastructure handle)
	sessionID := spawn.ReadSessionID(workspacePath)
	if sessionID != "" {
		// Get token usage from OpenCode API
		client := opencode.NewClient("http://127.0.0.1:4096")
		if tokenStats, err := client.GetSessionTokens(sessionID); err == nil && tokenStats != nil {
			tokensInput = tokenStats.InputTokens
			tokensOutput = tokenStats.OutputTokens
		}
	}

	return durationSeconds, tokensInput, tokensOutput, outcome
}

type verificationChecklistItem struct {
	Label  string
	Status string // passed, pending, skipped
}

func buildVerificationChecklist(
	result verify.VerificationResult,
	issueType string,
	tier string,
	isOrchestrator bool,
	skipConfig SkipConfig,
	gate1Complete bool,
	gate2Complete bool,
) []verificationChecklistItem {
	items := []verificationChecklistItem{}
	appendItem := func(label, status string) {
		if status == "n/a" {
			return
		}
		items = append(items, verificationChecklistItem{Label: label, Status: status})
	}

	gateStatus := func(gate string) string {
		if skipConfig.shouldSkipGate(gate) {
			return "skipped"
		}
		for _, failed := range result.GatesFailed {
			if failed == gate {
				return "pending"
			}
		}
		return "passed"
	}

	if isOrchestrator {
		appendItem("session handoff", gateStatus(verify.GateSessionHandoff))
		appendItem("handoff content", gateStatus(verify.GateHandoffContent))
		return items
	}

	if issueType != "" && checkpoint.RequiresCheckpoint(issueType) {
		explainStatus := "pending"
		if skipConfig.ExplainBack {
			explainStatus = "skipped"
		} else if gate1Complete {
			explainStatus = "passed"
		}
		appendItem("explain-back (gate1)", explainStatus)
	} else {
		appendItem("explain-back (gate1)", "n/a")
	}

	if issueType != "" && checkpoint.RequiresGate2(issueType) {
		behaviorStatus := "pending"
		if gate2Complete {
			behaviorStatus = "passed"
		}
		appendItem("behavioral verification (gate2)", behaviorStatus)
	} else {
		appendItem("behavioral verification (gate2)", "n/a")
	}

	appendItem("phase complete", gateStatus(verify.GatePhaseComplete))

	if tier == "light" || verify.IsKnowledgeProducingSkill(result.Skill) {
		appendItem("synthesis", "n/a")
	} else {
		appendItem("synthesis", gateStatus(verify.GateSynthesis))
	}

	appendItem("test evidence", gateStatus(verify.GateTestEvidence))
	appendItem("visual verification", gateStatus(verify.GateVisualVerify))
	appendItem("git diff", gateStatus(verify.GateGitDiff))
	appendItem("build", gateStatus(verify.GateBuild))
	appendItem("constraint", gateStatus(verify.GateConstraint))
	appendItem("phase gate", gateStatus(verify.GatePhaseGate))
	appendItem("skill output", gateStatus(verify.GateSkillOutput))
	appendItem("decision patch limit", gateStatus(verify.GateDecisionPatchLimit))
	appendItem("accretion", gateStatus(verify.GateAccretion))

	return items
}

func printVerificationChecklist(items []verificationChecklistItem) {
	if len(items) == 0 {
		return
	}

	fmt.Println("\n--- Verification Checklist ---")
	for _, item := range items {
		fmt.Printf("  [%s] %s\n", formatChecklistStatus(item.Status), item.Label)
	}
	fmt.Println("--------------------------------")
}

func formatChecklistStatus(status string) string {
	switch status {
	case "passed":
		return "PASS"
	case "pending":
		return "PEND"
	case "skipped":
		return "SKIP"
	default:
		return "N/A"
	}
}

// collectAccretionDelta collects file growth/shrinkage metrics from git diff.
// This tracks which files grew vs shrank during the agent's session, helping
// monitor whether accretion gravity is being inverted (files getting smaller
// rather than larger over time).
//
// Returns nil if collection fails (non-blocking).
func collectAccretionDelta(projectDir, workspacePath string) *events.AccretionDeltaData {
	if workspacePath == "" {
		return nil
	}

	// Read spawn time from manifest (falls back to dotfiles)
	manifest := spawn.ReadAgentManifestWithFallback(workspacePath)
	spawnTime := manifest.ParseSpawnTime()
	if spawnTime.IsZero() {
		return nil
	}

	// Get commits since spawn time that touch the workspace
	sinceStr := spawnTime.UTC().Format("2006-01-02T15:04:05Z")

	// Convert workspace path to relative path from project dir for git matching
	relWorkspace := workspacePath
	if filepath.IsAbs(workspacePath) && filepath.IsAbs(projectDir) {
		rel, err := filepath.Rel(projectDir, workspacePath)
		if err == nil {
			relWorkspace = rel
		}
	}

	// Get commit hashes since spawn time that touch the workspace
	cmd := exec.Command("git", "log", "--since="+sinceStr, "--format=%H", "--", relWorkspace)
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil || len(strings.TrimSpace(string(output))) == 0 {
		// No commits touching workspace
		return nil
	}

	commitHashes := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(commitHashes) == 0 {
		return nil
	}

	// Collect numstat for all commits touching workspace
	fileDeltas := make(map[string]*events.FileDelta)

	for _, hash := range commitHashes {
		if hash == "" {
			continue
		}

		// Get numstat for this commit
		cmd := exec.Command("git", "show", "--numstat", "--format=", hash)
		cmd.Dir = projectDir
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		// Parse numstat output: added<TAB>removed<TAB>filepath
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			parts := strings.Split(line, "\t")
			if len(parts) < 3 {
				continue
			}

			filePath := parts[2]

			// Parse line counts (may be "-" for binary files)
			added := 0
			removed := 0
			if parts[0] != "-" {
				var n int
				_, err := fmt.Sscanf(parts[0], "%d", &n)
				if err == nil {
					added = n
				}
			}
			if parts[1] != "-" {
				var n int
				_, err := fmt.Sscanf(parts[1], "%d", &n)
				if err == nil {
					removed = n
				}
			}

			// Aggregate changes per file across all commits
			if existing, ok := fileDeltas[filePath]; ok {
				existing.LinesAdded += added
				existing.LinesRemoved += removed
				existing.NetDelta = existing.LinesAdded - existing.LinesRemoved
			} else {
				fileDeltas[filePath] = &events.FileDelta{
					Path:         filePath,
					LinesAdded:   added,
					LinesRemoved: removed,
					NetDelta:     added - removed,
				}
			}
		}
	}

	// Count current lines in each file and check for accretion risk
	var totalAdded, totalRemoved, riskFiles int
	var deltas []events.FileDelta

	for _, delta := range fileDeltas {
		// Count current lines in the file
		fullPath := filepath.Join(projectDir, delta.Path)
		if lineCount, err := countFileLines(fullPath); err == nil {
			delta.TotalLines = lineCount
			delta.IsAccretionRisk = lineCount > 800

			// Track files >800 lines that grew
			if delta.IsAccretionRisk && delta.NetDelta > 0 {
				riskFiles++
			}
		}

		totalAdded += delta.LinesAdded
		totalRemoved += delta.LinesRemoved
		deltas = append(deltas, *delta)
	}

	if len(deltas) == 0 {
		return nil
	}

	return &events.AccretionDeltaData{
		FileDeltas:   deltas,
		TotalFiles:   len(deltas),
		TotalAdded:   totalAdded,
		TotalRemoved: totalRemoved,
		NetDelta:     totalAdded - totalRemoved,
		RiskFiles:    riskFiles,
	}
}

// countFileLines counts the number of lines in a file.
// Returns 0 if file doesn't exist or can't be read.
func countFileLines(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return lineCount, nil
}
