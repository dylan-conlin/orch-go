// Package main provides the CLI entry point for orch-go.
package main

import (
	"time"

	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	// Review command flags
	reviewProject     string
	reviewNeedsReview bool
	reviewDoneYes     bool
	reviewStale       bool
	reviewAll         bool
	reviewNoPrompt    bool
	reviewLimit       int
	reviewWorkdir     string
)

// StaleThreshold defines how long an agent must be in a non-Complete phase to be considered stale.
const StaleThreshold = 24 * time.Hour

var reviewCmd = &cobra.Command{
	Use:   "review [beads-id]",
	Short: "Review agent work before completing",
	Long: `Review agent work before completing.

Without arguments: Shows actionable pending completions grouped by project.
With beads-id: Shows detailed review for a single agent.

By default, stale agents (in non-Complete phase for >24h) and untracked agents
(spawned with --no-track) are excluded from the output. Use --stale to see them,
or --all to see everything.

Single-agent review shows:
  - SYNTHESIS.md summary (TLDR, outcome, recommendation)
  - Recent commits with stats
  - Beads comments history
  - Artifacts produced (investigations, design docs)

For cross-project review (agents spawned with --workdir in another project),
use --workdir to specify the target project directory.

Examples:
  orch review                       # Actionable completions only (excludes stale/untracked)
  orch review --limit 5             # Show at most 5 completions
  orch review --all                 # Show everything including stale/untracked
  orch review --stale               # Show only stale/untracked agents
  orch review orch-go-3anf          # Single agent: detailed review
  orch review -p orch-cli           # Filter by project
  orch review --needs               # Show failures only (shorthand for --needs-review)
  orch review --workdir ~/projects/kb-cli     # Cross-project review`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Single-agent mode if beads ID provided
		if len(args) > 0 {
			return runReviewSingle(args[0], reviewWorkdir)
		}
		// Batch mode
		return runReview(reviewProject, reviewNeedsReview, reviewStale, reviewAll, reviewLimit, reviewWorkdir)
	},
}

var reviewDoneCmd = &cobra.Command{
	Use:   "done [project]",
	Short: "Complete all agents for a project",
	Long: `Complete all agents for a project by closing their beads issues.

This runs the completion workflow for each agent with Phase: Complete status,
closing the beads issue and cleaning up resources.

For each agent with synthesis recommendations (NextActions in SYNTHESIS.md),
you'll be prompted to create follow-up issues:
  - y: Create beads issues for all recommendations
  - n: Skip this agent's recommendations
  - skip-all: Skip prompts for all remaining agents

Use --no-prompt to skip all recommendation prompts (for automation/scripting).

Agents that fail verification (no Phase: Complete) will be skipped.

For cross-project completion (agents spawned with --workdir in another project),
use --workdir to specify the target project directory.

Examples:
  orch-go review done orch-cli           # Complete with recommendation prompts
  orch-go review done orch-cli -y        # Skip initial confirmation
  orch-go review done orch-cli --no-prompt  # Skip recommendation prompts
  orch-go review done kb-cli --workdir ~/projects/kb-cli  # Cross-project completion`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReviewDone(args[0], reviewWorkdir)
	},
}

func init() {
	reviewCmd.Flags().StringVarP(&reviewProject, "project", "p", "", "Filter by project")
	reviewCmd.Flags().BoolVar(&reviewNeedsReview, "needs-review", false, "Show failures only")
	reviewCmd.Flags().BoolVar(&reviewNeedsReview, "needs", false, "Show failures only (shorthand for --needs-review)")
	reviewCmd.Flags().BoolVar(&reviewStale, "stale", false, "Show stale/untracked agents only")
	reviewCmd.Flags().BoolVar(&reviewAll, "all", false, "Show all agents including stale/untracked")
	reviewCmd.Flags().IntVarP(&reviewLimit, "limit", "l", 0, "Maximum number of completions to show (0 = no limit)")
	reviewCmd.Flags().StringVar(&reviewWorkdir, "workdir", "", "Target project directory (for cross-project review)")
	reviewDoneCmd.Flags().BoolVarP(&reviewDoneYes, "yes", "y", false, "Skip confirmation prompt")
	reviewDoneCmd.Flags().BoolVar(&reviewNoPrompt, "no-prompt", false, "Skip recommendation prompts (auto-close without reviewing synthesis)")
	reviewDoneCmd.Flags().StringVar(&reviewWorkdir, "workdir", "", "Target project directory (for cross-project review)")
	reviewCmd.AddCommand(reviewDoneCmd)
}

// CompletionInfo holds information about a completed agent for review.
type CompletionInfo struct {
	WorkspaceID   string // Workspace directory name
	WorkspacePath string // Full path to workspace directory
	BeadsID       string // Beads issue ID
	Project       string
	VerifyOK      bool
	VerifyError   string
	Phase         string
	Summary       string
	Skill         string
	Synthesis     *verify.Synthesis
	ModTime       time.Time // Workspace modification time
	IsUntracked   bool      // True if agent was spawned with --no-track
	IsStale       bool      // True if agent is in non-Complete phase for >24h
	IsLightTier   bool      // True if agent was spawned as light tier (no SYNTHESIS.md by design)
}
