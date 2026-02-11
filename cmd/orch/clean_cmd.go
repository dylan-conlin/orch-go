// Package main provides the clean command for cleaning up completed agent resources.
// Extracted from main.go as part of the main.go refactoring (Phase 4).
//
// Per-concern files:
//   - clean_workspaces.go: workspace/investigation archival, cleanable workspace detection, checker types
//   - clean_windows.go: phantom tmux window cleanup
//   - clean_sessions.go: orphaned OpenCode disk session cleanup
//   - clean_processes.go: orphan bun process cleanup
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/cleanup"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/spf13/cobra"
)

var (
	// Clean command flags
	cleanDryRun               bool
	cleanVerifyOpenCode       bool
	cleanWindows              bool
	cleanPhantoms             bool
	cleanInvestigations       bool
	cleanStale                bool
	cleanStaleDays            int
	cleanWorktrees            bool
	cleanWorktreeDays         int
	cleanUntracked            bool
	cleanUntrackedDays        int
	cleanSessions             bool
	cleanSessionsDays         int
	cleanPreserveOrchestrator bool
	cleanAll                  bool
	cleanProcesses            bool
	cleanWorkdir              string
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "List completed agents and optionally close their resources",
	Long: `List completed agents and optionally clean up their resources.

By default, this command only REPORTS what could be cleaned - it does not delete
anything. Workspace directories are always preserved for investigation reference.

What counts as "completed":
- Workspaces with SYNTHESIS.md file
- Workspaces whose beads issue is closed

Protection options:
  --preserve-orchestrator  Skip orchestrator/meta-orchestrator workspaces and sessions

Comprehensive cleanup:
  --all                  Enable all cleanup actions (windows, phantoms, verify-opencode, investigations, stale, worktrees, untracked, sessions, processes)

Optional cleanup actions:
  --windows              Close tmux windows for completed agents
  --phantoms             Close phantom tmux windows (beads ID but no active session)
  --verify-opencode      Delete orphaned OpenCode disk sessions (not tracked in workspaces)
  --investigations       Archive empty investigation files (agents died before filling template)
  --stale                Reconcile stale state rows then archive old completed workspaces (default: 7 days)
  --stale-days N         Set age threshold for --stale (default: 7)
  --worktrees            Prune orphaned git worktrees under .orch/worktrees (active agents in state.db are preserved)
  --worktree-days N      Optional age threshold for --worktrees (default: 0, disable age filtering)
  --untracked            Archive old untracked workspaces (default: 7 days)
  --untracked-days N     Set age threshold for --untracked (default: 7)
  --sessions             Delete stale OpenCode sessions (default: older than 7 days)
  --sessions-days N      Set age threshold for --sessions (default: 7, WARNING: 0 deletes ALL sessions)
  --preserve-orchestrator  Protect orchestrator/meta-orch sessions from deletion (default: true)
  --processes            Kill orphaned bun processes (agent processes and untracked dashboard web bun)

Process cleanup:
  --processes uses OS-level process discovery (ps/lsof) to find bun processes that
  have no active owner (agent sessions or tracked dashboard web PID). This catches
  orphans that survived session deletion, workspace archival, or dashboard restarts.
  Recommended: use --all or --processes periodically to prevent memory accumulation.

Note: This command never deletes workspace directories - they are kept for 
investigation reference. Use 'rm -rf .orch/workspace/<name>' to manually delete.

Examples:
  orch-go clean                    # List completed agents (no changes)
  orch-go clean --dry-run          # Preview mode (same as default)
  orch-go clean --all              # Comprehensive cleanup of all agent status sources
  orch-go clean --all --dry-run    # Preview comprehensive cleanup
  orch-go clean --all                          # Clean everything (orchestrator sessions protected by default)
  orch-go clean --windows          # Close tmux windows for completed agents
  orch-go clean --phantoms         # Close phantom tmux windows
  orch-go clean --verify-opencode  # Delete orphaned OpenCode disk sessions
  orch-go clean --investigations   # Archive empty investigation templates
  orch-go clean --stale            # Reconcile state cache + archive completed workspaces older than 7 days
  orch-go clean --stale --stale-days 14  # Archive completed workspaces older than 14 days
  orch-go clean --worktrees        # Prune orphaned git worktrees
  orch-go clean --worktrees --worktree-days 14  # Prune orphaned git worktrees older than 14 days
  orch-go clean --untracked        # Archive untracked workspaces older than 7 days
  orch-go clean --untracked --untracked-days 14  # Archive untracked workspaces older than 14 days
  orch-go clean --sessions         # Delete OpenCode sessions older than 7 days
  orch-go clean --sessions --sessions-days 14  # Delete sessions older than 14 days
  orch-go clean --sessions --preserve-orchestrator=false  # Clean sessions INCLUDING orchestrators
  orch-go clean --processes                         # Kill orphaned bun processes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cleanAll {
			cleanWindows = true
			cleanPhantoms = true
			cleanVerifyOpenCode = true
			cleanInvestigations = true
			cleanStale = true
			cleanWorktrees = true
			cleanUntracked = true
			cleanSessions = true
			cleanProcesses = true
		}
		return runClean(cleanDryRun, cleanVerifyOpenCode, cleanWindows, cleanPhantoms, cleanInvestigations, cleanStale, cleanStaleDays, cleanWorktrees, cleanWorktreeDays, cleanUntracked, cleanUntrackedDays, cleanSessions, cleanSessionsDays, cleanPreserveOrchestrator, cleanProcesses, cleanWorkdir)
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "Show what would be cleaned without making changes")
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Enable all cleanup actions (windows, phantoms, verify-opencode, investigations, stale, worktrees, untracked, sessions, processes)")
	cleanCmd.Flags().BoolVar(&cleanVerifyOpenCode, "verify-opencode", false, "Also verify OpenCode disk sessions (slower)")
	cleanCmd.Flags().BoolVar(&cleanWindows, "windows", false, "Close tmux windows for completed agents")
	cleanCmd.Flags().BoolVar(&cleanPhantoms, "phantoms", false, "Close all phantom tmux windows (stale agent windows)")
	cleanCmd.Flags().BoolVar(&cleanInvestigations, "investigations", false, "Archive empty investigation files to .kb/investigations/archived/")
	cleanCmd.Flags().BoolVar(&cleanStale, "stale", false, "Reconcile stale state rows and archive completed workspaces older than N days (default: 7)")
	cleanCmd.Flags().IntVar(&cleanStaleDays, "stale-days", 7, "Age threshold in days for --stale (default: 7)")
	cleanCmd.Flags().BoolVar(&cleanWorktrees, "worktrees", false, "Prune orphaned git worktrees under .orch/worktrees (active agents in state.db are preserved)")
	cleanCmd.Flags().IntVar(&cleanWorktreeDays, "worktree-days", 0, "Optional age threshold in days for --worktrees (default: 0)")
	cleanCmd.Flags().BoolVar(&cleanUntracked, "untracked", false, "Archive untracked workspaces older than N days (default: 7)")
	cleanCmd.Flags().IntVar(&cleanUntrackedDays, "untracked-days", 7, "Age threshold in days for --untracked (default: 7)")
	cleanCmd.Flags().BoolVar(&cleanSessions, "sessions", false, "Delete stale OpenCode sessions older than N days (default: 7)")
	cleanCmd.Flags().IntVar(&cleanSessionsDays, "sessions-days", 7, "Age threshold in days for --sessions (default: 7)")
	cleanCmd.Flags().BoolVar(&cleanPreserveOrchestrator, "preserve-orchestrator", true, "Skip orchestrator/meta-orchestrator workspaces and sessions (default: true, use --preserve-orchestrator=false to include them)")
	cleanCmd.Flags().BoolVar(&cleanProcesses, "processes", false, "Kill orphaned bun processes (agent processes and untracked dashboard web bun)")
	cleanCmd.Flags().StringVar(&cleanWorkdir, "workdir", "", "Target project directory (for cross-project cleanup)")
	cleanCmd.Flags().StringVar(&cleanWorkdir, "project", "", "Alias for --workdir")
	cleanCmd.Flags().MarkHidden("project")
}

// runClean orchestrates all cleanup subcommands based on the provided flags.
func runClean(dryRun bool, verifyOpenCode bool, closeWindows bool, cleanPhantoms bool, cleanInvestigations bool, archiveStale bool, staleDays int, cleanWorktrees bool, worktreeDays int, archiveUntracked bool, untrackedDays int, cleanSessions bool, sessionsDays int, preserveOrchestrator bool, killProcesses bool, workdir string) error {
	currentDir, err := currentProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	projectResult, err := resolveProjectDir(workdir, "", currentDir)
	if err != nil {
		return err
	}
	projectResult.SetBeadsDefaultDir()
	projectDir := projectResult.ProjectDir

	needsCompletedWorkspaces := closeWindows || (!archiveStale && !archiveUntracked && !cleanPhantoms && !verifyOpenCode && !cleanInvestigations)

	windowsClosed := 0
	var cleanableWorkspaces []CleanableWorkspace

	if needsCompletedWorkspaces {
		fmt.Println("Scanning workspaces for completed agents...")
		beadsChecker := NewDefaultBeadsStatusChecker()
		cleanableWorkspaces = findCleanableWorkspaces(projectDir, beadsChecker)

		fmt.Printf("\nFound %d completed workspaces\n", len(cleanableWorkspaces))

		if len(cleanableWorkspaces) == 0 && !verifyOpenCode && !cleanPhantoms && !cleanInvestigations && !archiveStale && !archiveUntracked {
			fmt.Println("No completed agents found")
			return nil
		}

		if len(cleanableWorkspaces) > 0 {
			fmt.Printf("\nCompleted workspaces:\n")
			for _, ws := range cleanableWorkspaces {
				fmt.Printf("  %s (%s)\n", ws.Name, ws.Reason)
				if closeWindows && !dryRun {
					if window, sessionName, _ := tmux.FindWindowByWorkspaceNameAllSessions(ws.Name); window != nil {
						if err := tmux.KillWindow(window.Target); err != nil {
							fmt.Fprintf(os.Stderr, "    Warning: failed to close window %s: %v\n", window.Name, err)
						} else {
							fmt.Printf("    Closed window: %s in session %s\n", window.Name, sessionName)
							windowsClosed++
						}
					}
				} else if closeWindows && dryRun {
					if window, sessionName, _ := tmux.FindWindowByWorkspaceNameAllSessions(ws.Name); window != nil {
						fmt.Printf("    [DRY-RUN] Would close window: %s in session %s\n", window.Name, sessionName)
					}
				}
			}
		}
	}

	var diskSessionsDeleted int
	if verifyOpenCode {
		diskSessionsDeleted, err = cleanOrphanedDiskSessions(serverURL, projectDir, dryRun, preserveOrchestrator)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean disk sessions: %v\n", err)
		}
	}

	var phantomsClosed int
	if cleanPhantoms {
		phantomsClosed, err = cleanPhantomWindows(serverURL, dryRun, preserveOrchestrator)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean phantom windows: %v\n", err)
		}
	}

	var investigationsArchived int
	if cleanInvestigations {
		investigationsArchived, err = archiveEmptyInvestigations(projectDir, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to archive empty investigations: %v\n", err)
		}
	}

	var stateRowsCompleted int
	var stateRowsAbandoned int
	var stateOpenMinusBefore int
	var stateOpenMinusAfter int
	var registryRowsUpdated int
	var workspacesArchived int
	if archiveStale {
		reconcileResult, reconcileErr := cleanup.ReconcileState(cleanup.ReconcileStateOptions{
			ServerURL:         serverURL,
			DryRun:            dryRun,
			Quiet:             false,
			ReconcileRegistry: true,
		})
		if reconcileErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to reconcile state cache: %v\n", reconcileErr)
		} else if reconcileResult != nil {
			stateRowsCompleted = reconcileResult.CompletedRows
			stateRowsAbandoned = reconcileResult.AbandonedRows
			stateOpenMinusBefore = reconcileResult.OpenMinusLiveBefore
			stateOpenMinusAfter = reconcileResult.OpenMinusLiveAfter
			registryRowsUpdated = reconcileResult.RegistryUpdated
		}

		workspacesArchived, err = archiveStaleWorkspaces(projectDir, staleDays, dryRun, preserveOrchestrator)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to archive stale workspaces: %v\n", err)
		}
	}

	var untrackedArchived int
	if archiveUntracked {
		untrackedArchived, err = archiveUntrackedWorkspaces(projectDir, untrackedDays, dryRun, preserveOrchestrator)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to archive untracked workspaces: %v\n", err)
		}
	}

	worktreePruned := 0
	worktreeBranchesDeleted := 0
	if cleanWorktrees {
		janitorResult, janitorErr := cleanupStaleManagedWorktrees(projectDir, worktreeDays, dryRun)
		if janitorErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to prune orphaned worktrees: %v\n", janitorErr)
		} else {
			worktreePruned = janitorResult.WorktreesPruned
			worktreeBranchesDeleted = janitorResult.BranchesDeleted
		}
	}

	var staleSessionsDeleted int
	if cleanSessions {
		if sessionsDays == 0 && !preserveOrchestrator {
			fmt.Fprintf(os.Stderr, "Error: --sessions-days 0 without --preserve-orchestrator=true would delete ALL sessions including active orchestrator sessions. Refusing.\n")
			fmt.Fprintf(os.Stderr, "Use --preserve-orchestrator=false explicitly to override.\n")
			return fmt.Errorf("dangerous session cleanup refused")
		}
		if sessionsDays == 0 {
			fmt.Fprintf(os.Stderr, "Warning: --sessions-days 0 will delete ALL non-orchestrator sessions regardless of age\n")
		}
		staleSessionsDeleted, err = cleanup.CleanStaleSessions(cleanup.CleanStaleSessionsOptions{
			ServerURL:            serverURL,
			StaleDays:            sessionsDays,
			DryRun:               dryRun,
			PreserveOrchestrator: preserveOrchestrator,
			Quiet:                false,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean stale sessions: %v\n", err)
		}
	}

	var processesKilled int
	if killProcesses {
		processesKilled, err = cleanOrphanProcesses(serverURL, projectDir, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean orphan processes: %v\n", err)
		}
	}

	hasCleanupActions := closeWindows || cleanPhantoms || verifyOpenCode || cleanInvestigations || archiveStale || cleanWorktrees || archiveUntracked || cleanSessions || killProcesses

	if dryRun {
		if hasCleanupActions {
			fmt.Printf("\nDry run complete.")
			if closeWindows {
				windowCount := 0
				for _, ws := range cleanableWorkspaces {
					if window, _, _ := tmux.FindWindowByWorkspaceNameAllSessions(ws.Name); window != nil {
						windowCount++
					}
				}
				if windowCount > 0 {
					fmt.Printf(" Would close %d tmux windows.", windowCount)
				}
			}
			if cleanPhantoms && phantomsClosed > 0 {
				fmt.Printf(" Would close %d phantom windows.", phantomsClosed)
			}
			if verifyOpenCode && diskSessionsDeleted > 0 {
				fmt.Printf(" Would delete %d orphaned disk sessions.", diskSessionsDeleted)
			}
			if cleanInvestigations && investigationsArchived > 0 {
				fmt.Printf(" Would archive %d empty investigations.", investigationsArchived)
			}
			if archiveStale && workspacesArchived > 0 {
				fmt.Printf(" Would archive %d stale workspaces.", workspacesArchived)
			}
			if archiveStale && (stateRowsCompleted > 0 || stateRowsAbandoned > 0) {
				fmt.Printf(" Would reconcile %d stale state rows.", stateRowsCompleted+stateRowsAbandoned)
			}
			if archiveStale && registryRowsUpdated > 0 {
				fmt.Printf(" Would reconcile %d sessions.json entries.", registryRowsUpdated)
			}
			if cleanWorktrees && worktreePruned > 0 {
				fmt.Printf(" Would prune %d orphaned git worktrees.", worktreePruned)
				if worktreeBranchesDeleted > 0 {
					fmt.Printf(" Would delete %d agent branches.", worktreeBranchesDeleted)
				}
			}
			if archiveUntracked && untrackedArchived > 0 {
				fmt.Printf(" Would archive %d untracked workspaces.", untrackedArchived)
			}
			if cleanSessions && staleSessionsDeleted > 0 {
				fmt.Printf(" Would delete %d stale OpenCode sessions.", staleSessionsDeleted)
			}
			if killProcesses && processesKilled > 0 {
				fmt.Printf(" Would kill %d orphan processes.", processesKilled)
			}
			fmt.Println()
		}
		return nil
	}

	if windowsClosed > 0 || phantomsClosed > 0 || diskSessionsDeleted > 0 || investigationsArchived > 0 || workspacesArchived > 0 || worktreePruned > 0 || untrackedArchived > 0 || staleSessionsDeleted > 0 || processesKilled > 0 || stateRowsCompleted > 0 || stateRowsAbandoned > 0 || registryRowsUpdated > 0 {
		projectName := filepath.Base(projectDir)
		logger := events.NewLogger(events.DefaultLogPath())
		event := events.Event{
			Type:      "agents.cleaned",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"completed_workspaces":    len(cleanableWorkspaces),
				"windows_closed":          windowsClosed,
				"phantoms_closed":         phantomsClosed,
				"disk_sessions_deleted":   diskSessionsDeleted,
				"investigations_archived": investigationsArchived,
				"workspaces_archived":     workspacesArchived,
				"state_rows_completed":    stateRowsCompleted,
				"state_rows_abandoned":    stateRowsAbandoned,
				"state_open_minus_before": stateOpenMinusBefore,
				"state_open_minus_after":  stateOpenMinusAfter,
				"registry_rows_updated":   registryRowsUpdated,
				"worktrees_pruned":        worktreePruned,
				"worktree_branches":       worktreeBranchesDeleted,
				"untracked_archived":      untrackedArchived,
				"project":                 projectName,
				"verify_opencode":         verifyOpenCode,
				"close_windows":           closeWindows,
				"clean_phantoms":          cleanPhantoms,
				"clean_investigations":    cleanInvestigations,
				"archive_stale":           archiveStale,
				"stale_days":              staleDays,
				"clean_worktrees":         cleanWorktrees,
				"worktree_days":           worktreeDays,
				"archive_untracked":       archiveUntracked,
				"untracked_days":          untrackedDays,
				"clean_sessions":          cleanSessions,
				"sessions_days":           sessionsDays,
				"stale_sessions_deleted":  staleSessionsDeleted,
				"processes_killed":        processesKilled,
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
		}
	}

	if windowsClosed > 0 || phantomsClosed > 0 || diskSessionsDeleted > 0 || investigationsArchived > 0 || workspacesArchived > 0 || worktreePruned > 0 || untrackedArchived > 0 || staleSessionsDeleted > 0 || processesKilled > 0 || stateRowsCompleted > 0 || stateRowsAbandoned > 0 || registryRowsUpdated > 0 {
		fmt.Println()
		if windowsClosed > 0 {
			fmt.Printf("Closed %d tmux windows\n", windowsClosed)
		}
		if phantomsClosed > 0 {
			fmt.Printf("Closed %d phantom windows\n", phantomsClosed)
		}
		if diskSessionsDeleted > 0 {
			fmt.Printf("Deleted %d orphaned disk sessions\n", diskSessionsDeleted)
		}
		if investigationsArchived > 0 {
			fmt.Printf("Archived %d empty investigation files\n", investigationsArchived)
		}
		if workspacesArchived > 0 {
			fmt.Printf("Archived %d stale workspaces\n", workspacesArchived)
		}
		if stateRowsCompleted > 0 || stateRowsAbandoned > 0 {
			fmt.Printf("Reconciled %d stale state rows (completed=%d, abandoned=%d, open_minus_live %d->%d)\n", stateRowsCompleted+stateRowsAbandoned, stateRowsCompleted, stateRowsAbandoned, stateOpenMinusBefore, stateOpenMinusAfter)
		}
		if registryRowsUpdated > 0 {
			fmt.Printf("Reconciled %d stale sessions.json entries\n", registryRowsUpdated)
		}
		if worktreePruned > 0 {
			fmt.Printf("Pruned %d orphaned git worktrees\n", worktreePruned)
			if worktreeBranchesDeleted > 0 {
				fmt.Printf("Deleted %d orphaned agent branches\n", worktreeBranchesDeleted)
			}
		}
		if untrackedArchived > 0 {
			fmt.Printf("Archived %d untracked workspaces\n", untrackedArchived)
		}
		if staleSessionsDeleted > 0 {
			fmt.Printf("Deleted %d stale OpenCode sessions\n", staleSessionsDeleted)
		}
		if processesKilled > 0 {
			fmt.Printf("Killed %d orphan bun processes\n", processesKilled)
		}
	} else if !hasCleanupActions {
		fmt.Printf("\nNote: Workspace directories are preserved. Use --windows, --phantoms, --verify-opencode, --investigations, --stale, or --untracked to clean up resources.\n")
	}

	return nil
}

// NOTE: extractBeadsIDFromWorkspace is defined in review.go
