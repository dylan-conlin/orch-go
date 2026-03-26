// Package main provides the clean command for cleaning up completed agent resources.
// Extracted from main.go as part of the main.go refactoring (Phase 4).
//
// Sub-files:
//   - clean_workspaces.go: Workspace archival, investigation cleanup, expired archives
//   - clean_sessions.go: Stale tmux window detection and cleanup
//   - clean_orphans.go: Orphan GC and ghost agent label cleanup
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/execution"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	// Clean command flags
	cleanDryRun               bool
	cleanWorkspaces           bool
	cleanSessions             bool
	cleanOrphans              bool
	cleanGhosts               bool
	cleanWorkspaceDays        int
	cleanSessionDays          int
	cleanPreserveOrchestrator bool
	cleanAll                  bool
	cleanArchivedTTL          int
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up stale workspaces and tmux windows",
	Long: `Clean up stale agent resources (workspaces, tmux windows, orphaned agents).

NOTE: OpenCode session cleanup is now handled automatically via TTL (opencode-fork commit f3c3865).
The --sessions flag now only cleans stale tmux windows. OpenCode sessions are managed by the server.

By default, this command only REPORTS what could be cleaned - it does not delete
anything. Workspace directories are archived (moved to archived/), never deleted.

Cleanup actions:
  --workspaces        Archive old completed workspaces and empty investigation files
  --sessions          Clean stale tmux windows (OpenCode sessions are auto-cleaned by server)
  --orphans           Detect and GC orphaned agents via LifecycleManager (ForceComplete/ForceAbandon)
  --ghosts            Remove stale orch:agent labels from cross-project issues with no active agent
  --all               Enable all cleanup actions (workspaces + sessions + orphans + ghosts)

Age thresholds:
  --workspace-days N  Set age threshold for --workspaces (default: 7)
  --archived-ttl N    TTL in days for archived workspace expiry (default: 30)

Protection options:
  --preserve-orchestrator  Skip orchestrator/meta-orchestrator workspaces and sessions

Examples:
  orch clean                    # List completed agents and orphans (no changes)
  orch clean --dry-run          # Preview mode (same as default)
  orch clean --all              # Comprehensive cleanup
  orch clean --all --dry-run    # Preview comprehensive cleanup
  orch clean --all --preserve-orchestrator  # Clean everything except orchestrator sessions
  orch clean --sessions         # Clean stale tmux windows
  orch clean --orphans          # GC orphaned agents via lifecycle transitions
  orch clean --workspaces       # Archive old workspaces and empty investigations
  orch clean --workspaces --workspace-days 14  # Archive workspaces older than 14 days
  orch clean --ghosts            # Remove stale orch:agent labels from cross-project dead agents`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If --all is specified, enable all cleanup flags
		if cleanAll {
			cleanWorkspaces = true
			cleanSessions = true
			cleanOrphans = true
			cleanGhosts = true
		}
		return runClean(cleanDryRun, cleanWorkspaces, cleanSessions, cleanOrphans, cleanGhosts, cleanWorkspaceDays, cleanSessionDays, cleanPreserveOrchestrator, cleanArchivedTTL)
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "Show what would be cleaned without making changes")
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Enable all cleanup actions (workspaces + sessions)")
	cleanCmd.Flags().BoolVar(&cleanWorkspaces, "workspaces", false, "Archive old completed workspaces and empty investigation files")
	cleanCmd.Flags().BoolVar(&cleanSessions, "sessions", false, "Clean stale tmux windows (OpenCode sessions are auto-cleaned by server)")
	cleanCmd.Flags().BoolVar(&cleanOrphans, "orphans", false, "Detect and GC orphaned agents via LifecycleManager (ForceComplete/ForceAbandon)")
	cleanCmd.Flags().BoolVar(&cleanGhosts, "ghosts", false, "Remove stale orch:agent labels from cross-project issues with no active agent")
	cleanCmd.Flags().IntVar(&cleanWorkspaceDays, "workspace-days", 7, "Age threshold in days for --workspaces (default: 7)")
	cleanCmd.Flags().IntVar(&cleanSessionDays, "session-days", 7, "Age threshold in days for --sessions (default: 7)")
	cleanCmd.Flags().IntVar(&cleanArchivedTTL, "archived-ttl", 30, "TTL in days for archived workspace expiry (default: 30)")
	cleanCmd.Flags().BoolVar(&cleanPreserveOrchestrator, "preserve-orchestrator", false, "Skip orchestrator/meta-orchestrator workspaces and sessions")
}

// DefaultLivenessChecker checks if tmux windows and OpenCode sessions exist.
type DefaultLivenessChecker struct {
	client execution.SessionClient
}

// NewDefaultLivenessChecker creates a new liveness checker.
func NewDefaultLivenessChecker(serverURL string) *DefaultLivenessChecker {
	return &DefaultLivenessChecker{
		client: execution.NewOpenCodeAdapter(serverURL),
	}
}

// WindowExists checks if a tmux window ID exists.
func (c *DefaultLivenessChecker) WindowExists(windowID string) bool {
	return tmux.WindowExistsByID(windowID)
}

// SessionExists checks if an OpenCode session ID exists.
func (c *DefaultLivenessChecker) SessionExists(sessionID string) bool {
	return c.client.SessionExists(context.Background(), execution.SessionHandle(sessionID))
}

// DefaultBeadsStatusChecker checks beads issue status using the verify package.
type DefaultBeadsStatusChecker struct{}

// NewDefaultBeadsStatusChecker creates a new beads status checker.
func NewDefaultBeadsStatusChecker() *DefaultBeadsStatusChecker {
	return &DefaultBeadsStatusChecker{}
}

// IsIssueClosed checks if a beads issue is closed.
func (c *DefaultBeadsStatusChecker) IsIssueClosed(beadsID string) bool {
	issue, err := verify.GetIssue(beadsID, "")
	if err != nil {
		// If we can't get the issue, assume it's not closed
		// (could be network error, issue not found, etc.)
		return false
	}
	return issue.Status == "closed"
}

// DefaultCompletionIndicatorChecker checks for completion indicators (SYNTHESIS.md, Phase: Complete).
// This is used to determine if an agent completed its work.
type DefaultCompletionIndicatorChecker struct{}

// NewDefaultCompletionIndicatorChecker creates a new completion indicator checker.
func NewDefaultCompletionIndicatorChecker() *DefaultCompletionIndicatorChecker {
	return &DefaultCompletionIndicatorChecker{}
}

// SynthesisExists checks if SYNTHESIS.md exists in the agent's workspace.
func (c *DefaultCompletionIndicatorChecker) SynthesisExists(workspacePath string) bool {
	exists, err := verify.VerifySynthesis(workspacePath)
	if err != nil {
		// If we can't check (e.g., directory doesn't exist), assume no synthesis
		return false
	}
	return exists
}

// IsPhaseComplete checks if beads shows Phase: Complete for the agent.
func (c *DefaultCompletionIndicatorChecker) IsPhaseComplete(beadsID string) bool {
	complete, err := verify.IsPhaseComplete(beadsID, "")
	if err != nil {
		// If we can't check (e.g., beads error), assume not complete
		return false
	}
	return complete
}

// cleanUntrackedDiskSessions has been removed - OpenCode now handles session cleanup via TTL
// (see opencode-fork commit f3c3865)

func runClean(dryRun bool, doWorkspaces bool, doSessions bool, doOrphans bool, doGhosts bool, workspaceDays int, sessionDays int, preserveOrchestrator bool, archivedTTL int) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Default mode (no flags): scan and report completed workspaces + orphaned agents
	if !doWorkspaces && !doSessions && !doOrphans && !doGhosts {
		fmt.Println("Scanning workspaces for completed agents...")
		beadsChecker := NewDefaultBeadsStatusChecker()
		cleanableWorkspaces := findCleanableWorkspaces(projectDir, beadsChecker)

		fmt.Printf("\nFound %d completed workspaces\n", len(cleanableWorkspaces))

		if len(cleanableWorkspaces) > 0 {
			fmt.Printf("\nCompleted workspaces:\n")
			for _, ws := range cleanableWorkspaces {
				fmt.Printf("  %s (%s)\n", ws.Name, ws.Reason)
			}
		}

		// Also report orphaned agents (detection only, no GC)
		orphanReport, err := detectOrphansReport(projectDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: orphan detection failed: %v\n", err)
		} else if len(orphanReport) > 0 {
			fmt.Printf("\nOrphaned agents (%d):\n", len(orphanReport))
			for _, line := range orphanReport {
				fmt.Printf("  %s\n", line)
			}
		}

		if len(cleanableWorkspaces) == 0 && len(orphanReport) == 0 {
			fmt.Println("No completed or orphaned agents found")
			return nil
		}

		fmt.Printf("\nNote: Use --workspaces, --sessions, --orphans, or --all to clean up resources.\n")
		return nil
	}

	// Track cleanup stats
	var workspacesArchived, investigationsArchived, archivesExpired int
	var windowsClosed int
	var orphansForceCompleted, orphansForceAbandoned int
	var ghostsCleaned int

	// --workspaces: Archive old workspaces + empty investigations + expire old archives
	if doWorkspaces {
		workspacesArchived, err = archiveStaleWorkspaces(projectDir, workspaceDays, dryRun, preserveOrchestrator)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to archive stale workspaces: %v\n", err)
		}

		archivesExpired, err = cleanExpiredArchives(projectDir, archivedTTL, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean expired archives: %v\n", err)
		}

		investigationsArchived, err = archiveEmptyInvestigations(projectDir, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to archive empty investigations: %v\n", err)
		}
	}

	// --sessions: Clean all stale session infrastructure
	if doSessions {
		// NOTE: OpenCode now handles session cleanup automatically via TTL (see opencode-fork commit f3c3865)
		// Session cleanup logic has been removed from orch-go. Only tmux window cleanup remains.

		// 1. Close stale tmux windows (no active session behind them)
		windowsClosed, err = cleanStaleTmuxWindows(serverURL, projectDir, dryRun, preserveOrchestrator)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean stale tmux windows: %v\n", err)
		}

		// Session cleanup (steps 2-3) removed - OpenCode handles this via TTL
	}

	// --orphans: Detect and GC orphaned agents via LifecycleManager
	if doOrphans {
		orphansForceCompleted, orphansForceAbandoned, err = runOrphanGC(projectDir, dryRun, preserveOrchestrator)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: orphan GC failed: %v\n", err)
		}
	}

	// --ghosts: Remove stale orch:agent labels from cross-project dead agents
	if doGhosts {
		ghostsCleaned, err = cleanGhostAgents(projectDir, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: ghost cleanup failed: %v\n", err)
		}
	}

	// Dry-run summary
	if dryRun {
		fmt.Printf("\nDry run complete.")
		if doWorkspaces {
			if workspacesArchived > 0 {
				fmt.Printf(" Would archive %d stale workspaces.", workspacesArchived)
			}
			if archivesExpired > 0 {
				fmt.Printf(" Would delete %d expired archived workspaces.", archivesExpired)
			}
			if investigationsArchived > 0 {
				fmt.Printf(" Would archive %d empty investigations.", investigationsArchived)
			}
		}
		if doSessions {
			if windowsClosed > 0 {
				fmt.Printf(" Would close %d stale tmux windows.", windowsClosed)
			}
		}
		if doOrphans {
			if orphansForceCompleted > 0 {
				fmt.Printf(" Would force-complete %d orphaned agents.", orphansForceCompleted)
			}
			if orphansForceAbandoned > 0 {
				fmt.Printf(" Would force-abandon %d orphaned agents.", orphansForceAbandoned)
			}
		}
		if doGhosts {
			if ghostsCleaned > 0 {
				fmt.Printf(" Would remove orch:agent label from %d cross-project ghost agents.", ghostsCleaned)
			}
		}
		fmt.Println()
		return nil
	}

	// Log event if any cleanup actions were taken
	totalCleaned := workspacesArchived + archivesExpired + investigationsArchived + windowsClosed + orphansForceCompleted + orphansForceAbandoned + ghostsCleaned
	if totalCleaned > 0 {
		projectName := filepath.Base(projectDir)
		logger := events.NewLogger(events.DefaultLogPath())
		event := events.Event{
			Type:      "agents.cleaned",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"workspaces_archived":     workspacesArchived,
				"archives_expired":        archivesExpired,
				"investigations_archived": investigationsArchived,
				"windows_closed":          windowsClosed,
				"orphans_force_completed": orphansForceCompleted,
				"orphans_force_abandoned": orphansForceAbandoned,
				"project":                 projectName,
				"clean_workspaces":        doWorkspaces,
				"clean_sessions":          doSessions,
				"clean_orphans":           doOrphans,
				"clean_ghosts":            doGhosts,
				"ghosts_cleaned":          ghostsCleaned,
				"workspace_days":          workspaceDays,
				"archived_ttl_days":       archivedTTL,
				"session_days":            sessionDays,
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
		}
	}

	// Print summary
	if totalCleaned > 0 {
		fmt.Println()
		if workspacesArchived > 0 {
			fmt.Printf("Archived %d stale workspaces\n", workspacesArchived)
		}
		if archivesExpired > 0 {
			fmt.Printf("Deleted %d expired archived workspaces\n", archivesExpired)
		}
		if investigationsArchived > 0 {
			fmt.Printf("Archived %d empty investigation files\n", investigationsArchived)
		}
		if windowsClosed > 0 {
			fmt.Printf("Closed %d stale tmux windows\n", windowsClosed)
		}
		if orphansForceCompleted > 0 {
			fmt.Printf("Force-completed %d orphaned agents\n", orphansForceCompleted)
		}
		if orphansForceAbandoned > 0 {
			fmt.Printf("Force-abandoned %d orphaned agents (will retry via respawn)\n", orphansForceAbandoned)
		}
		if ghostsCleaned > 0 {
			fmt.Printf("Removed orch:agent label from %d cross-project ghost agents\n", ghostsCleaned)
		}
	}

	return nil
}
