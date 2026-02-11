// Package main provides the phase command for direct agent phase reporting via SQLite.
//
// Agents call `orch phase <beads_id> <phase> [summary]` which:
//  1. Writes directly to ~/.orch/state.db (~1ms) for runtime visibility
//  2. Fires a background bd comment for permanent audit trail in beads
//
// This means agents only need ONE command (`orch phase`) to get both
// fast runtime state AND searchable audit history. The bd comment is
// non-blocking — if it fails, the phase is still recorded in SQLite.
//
// See: .kb/investigations/2026-02-06-design-single-source-agent-state.md (Fork 1)
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/episodic"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/spf13/cobra"
)

var phaseSkipComment bool

var phaseCmd = &cobra.Command{
	Use:   "phase <beads_id> <phase> [summary...]",
	Short: "Report agent phase transition (SQLite + bd comment)",
	Long: `Report an agent's phase transition to the state database and beads.

Writes phase directly to SQLite (~1ms) for immediate visibility in
orch status and the dashboard, then fires a background bd comment
for permanent searchable audit trail in beads.

Agents only need this ONE command for both runtime speed and audit.

Use --no-comment to skip the bd comment (e.g., in tests or when
beads is unavailable).

Examples:
  orch phase orch-go-12345 Planning "Analyzing codebase structure"
  orch phase orch-go-12345 Implementing "Building SQLite schema"
  orch phase orch-go-12345 Complete "All tests passing, ready for review"
  orch phase orch-go-12345 BLOCKED "Need clarification on API contract"`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		phase := args[1]
		summary := ""
		if len(args) > 2 {
			summary = strings.Join(args[2:], " ")
		}
		return runPhase(beadsID, phase, summary)
	},
}

func init() {
	phaseCmd.Flags().BoolVar(&phaseSkipComment, "no-comment", false, "Skip bd comment (SQLite only)")
	rootCmd.AddCommand(phaseCmd)
}

func runPhase(beadsID, phase, summary string) error {
	return runPhaseWithDB(beadsID, phase, summary, "", !phaseSkipComment)
}

// checkWorktreeCommits checks if a worktree has any commits ahead of its merge-base.
// Returns the commit count and any error. Returns (0, nil) if no commits found.
// This is used for ghost completion early detection when agents report Phase: Complete.
func checkWorktreeCommits(worktreePath string) (int, error) {
	// Get the current branch name
	cmd := exec.Command("git", "-C", worktreePath, "symbolic-ref", "--short", "HEAD")
	branchOut, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get branch name: %w", err)
	}
	branch := strings.TrimSpace(string(branchOut))

	// Determine base branch (typically main or master)
	baseBranch := "main"
	cmd = exec.Command("git", "-C", worktreePath, "rev-parse", "--verify", "main")
	if err := cmd.Run(); err != nil {
		// Try master if main doesn't exist
		baseBranch = "master"
	}

	// Find merge-base between base branch and current branch
	cmd = exec.Command("git", "-C", worktreePath, "merge-base", baseBranch, branch)
	mergeBaseOut, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to find merge-base: %w", err)
	}
	mergeBase := strings.TrimSpace(string(mergeBaseOut))

	// Count commits on current branch beyond merge-base
	cmd = exec.Command("git", "-C", worktreePath, "rev-list", "--count", mergeBase+".."+branch)
	countOut, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to count commits: %w", err)
	}

	count := strings.TrimSpace(string(countOut))
	var commitCount int
	if _, err := fmt.Sscanf(count, "%d", &commitCount); err != nil {
		return 0, fmt.Errorf("failed to parse commit count: %w", err)
	}

	return commitCount, nil
}

// formatPhaseComment formats a phase transition as a bd comment string.
// Follows the established "Phase: X - summary" convention that orch complete parses.
func formatPhaseComment(phase, summary string) string {
	switch phase {
	case "BLOCKED":
		if summary != "" {
			return fmt.Sprintf("BLOCKED: %s", summary)
		}
		return "BLOCKED"
	case "QUESTION":
		if summary != "" {
			return fmt.Sprintf("QUESTION: %s", summary)
		}
		return "QUESTION"
	default:
		if summary != "" {
			return fmt.Sprintf("Phase: %s - %s", phase, summary)
		}
		return fmt.Sprintf("Phase: %s", phase)
	}
}

// runPhaseWithDB writes a phase update to the state database and optionally
// fires a bd comment for audit trail.
// If dbPath is empty, uses the default path (~/.orch/state.db).
// If writeComment is true, also writes a bd comment (non-blocking on failure).
func runPhaseWithDB(beadsID, phase, summary, dbPath string, writeComment bool) error {
	// Open the state database
	var db *state.DB
	var err error
	if dbPath != "" {
		db, err = state.Open(dbPath)
	} else {
		db, err = state.OpenDefault()
	}
	if err != nil {
		return fmt.Errorf("failed to open state database: %w", err)
	}
	if db == nil {
		return fmt.Errorf("could not determine state database path")
	}
	defer db.Close()

	// Write phase directly to SQLite (~1ms)
	if err := db.UpdatePhaseByBeadsID(beadsID, phase, summary); err != nil {
		return fmt.Errorf("failed to update phase: %w", err)
	}

	// Invalidate serve cache to ensure dashboard shows updated phase immediately.
	// This is non-critical so we just call it and ignore errors.
	invalidateServeCache()

	// Look up agent to get project directory for beads operations.
	// This must happen BEFORE FallbackAddComment so we can set beads.DefaultDir.
	workspace := beadsID
	sessionID := ""
	project := projectFromCWD()
	agent, err := db.GetAgentByBeadsID(beadsID)
	if err == nil && agent != nil {
		if strings.TrimSpace(agent.WorkspaceName) != "" {
			workspace = agent.WorkspaceName
		}
		if strings.TrimSpace(agent.SessionID) != "" {
			sessionID = agent.SessionID
		}
		if strings.TrimSpace(agent.ProjectName) != "" {
			project = agent.ProjectName
		} else if strings.TrimSpace(agent.ProjectDir) != "" {
			project = filepath.Base(agent.ProjectDir)
		}

		// Ghost completion early detection: warn when Phase: Complete with 0 commits
		if phase == "Complete" && agent.ProjectDir != "" {
			// Construct worktree path from project dir and workspace name
			worktreePath := filepath.Join(agent.ProjectDir, ".orch", "worktrees", agent.WorkspaceName)
			if count, err := checkWorktreeCommits(worktreePath); err == nil {
				if count == 0 {
					fmt.Fprintf(os.Stderr, "⚠️  WARNING: Phase: Complete with 0 commits detected\n")
					fmt.Fprintf(os.Stderr, "   This may indicate ghost completion (work reported but not committed)\n")
					fmt.Fprintf(os.Stderr, "   Worktree: %s\n", worktreePath)
					fmt.Fprintf(os.Stderr, "   COMMIT_EVIDENCE gate will block during 'orch complete'\n")
				}
			}
			// Silently ignore errors (e.g., not a git repo, worktree doesn't exist)
			// The COMMIT_EVIDENCE gate at orch complete is the final safety net
		}
	}

	// Fire bd comment for backward compatibility and audit trail.
	// This is best-effort — if beads is unavailable, the phase is still
	// recorded in SQLite. We run it synchronously but don't fail the command
	// if the comment fails. The comment follows the established format that
	// orch complete parses for phase detection.
	//
	// IMPORTANT: Set beads.DefaultDir to the agent's project directory before
	// calling FallbackAddComment. This ensures the bd command runs from the
	// correct directory, which is critical when agents run in worktrees.
	// Without this, bd --sandbox writes to the worktree's local .beads/issues.jsonl
	// instead of the main project's beads database.
	if writeComment {
		// Set beads.DefaultDir to ensure cross-project operations work correctly.
		// This is essential for worktree contexts where cwd != project root.
		if agent != nil && strings.TrimSpace(agent.ProjectDir) != "" {
			beads.DefaultDir = agent.ProjectDir
		}
		comment := formatPhaseComment(phase, summary)
		if err := beads.FallbackAddComment(beadsID, comment); err != nil {
			// Non-fatal: phase was already written to SQLite.
			// Print warning so agents can see if comment failed.
			fmt.Fprintf(os.Stderr, "warning: bd comment failed (phase still recorded in SQLite): %v\n", err)
		}
	}

	phaseEvent := events.Event{
		Type:      "session.phase",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"phase":     phase,
			"summary":   summary,
			"beads_id":  beadsID,
			"workspace": workspace,
		},
	}
	recordEpisodicEvent(phaseEvent, episodic.Context{
		Boundary:  episodic.BoundaryCommand,
		Project:   project,
		Workspace: workspace,
		SessionID: sessionID,
		BeadsID:   beadsID,
	})

	// Confirm to caller
	if summary != "" {
		fmt.Printf("Phase: %s - %s (beads_id: %s)\n", phase, summary, beadsID)
	} else {
		fmt.Printf("Phase: %s (beads_id: %s)\n", phase, beadsID)
	}

	return nil
}
