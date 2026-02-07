// Package main provides the clean command for cleaning up completed agent resources.
// Extracted from main.go as part of the main.go refactoring (Phase 4).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/cleanup"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/process"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
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
	cleanUntracked            bool
	cleanUntrackedDays        int
	cleanSessions             bool
	cleanSessionsDays         int
	cleanPreserveOrchestrator bool
	cleanAll                  bool
	cleanProcesses            bool
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
  --all                  Enable all cleanup actions (windows, phantoms, verify-opencode, investigations, stale, untracked, sessions, processes)

Optional cleanup actions:
  --windows              Close tmux windows for completed agents
  --phantoms             Close phantom tmux windows (beads ID but no active session)
  --verify-opencode      Delete orphaned OpenCode disk sessions (not tracked in workspaces)
  --investigations       Archive empty investigation files (agents died before filling template)
  --stale                Archive old completed workspaces (default: 7 days)
  --stale-days N         Set age threshold for --stale (default: 7)
  --untracked            Archive old untracked workspaces (default: 7 days)
  --untracked-days N     Set age threshold for --untracked (default: 7)
  --sessions             Delete stale OpenCode sessions (default: older than 7 days)
  --sessions-days N      Set age threshold for --sessions (default: 7)
  --processes            Kill orphaned bun processes (agent processes without active sessions)

Process cleanup:
  --processes uses OS-level process discovery (ps) to find bun agent processes
  that have no matching active OpenCode session. This catches orphans that survived
  session deletion, workspace archival, or bd close workarounds.
  Recommended: use --all or --processes periodically to prevent memory accumulation.

Note: This command never deletes workspace directories - they are kept for 
investigation reference. Use 'rm -rf .orch/workspace/<name>' to manually delete.

Examples:
  orch-go clean                    # List completed agents (no changes)
  orch-go clean --dry-run          # Preview mode (same as default)
  orch-go clean --all              # Comprehensive cleanup of all agent status sources
  orch-go clean --all --dry-run    # Preview comprehensive cleanup
  orch-go clean --all --preserve-orchestrator  # Clean everything except orchestrator sessions
  orch-go clean --windows          # Close tmux windows for completed agents
  orch-go clean --phantoms         # Close phantom tmux windows
  orch-go clean --verify-opencode  # Delete orphaned OpenCode disk sessions
  orch-go clean --investigations   # Archive empty investigation templates
  orch-go clean --stale            # Archive completed workspaces older than 7 days
  orch-go clean --stale --stale-days 14  # Archive completed workspaces older than 14 days
  orch-go clean --untracked        # Archive untracked workspaces older than 7 days
  orch-go clean --untracked --untracked-days 14  # Archive untracked workspaces older than 14 days
  orch-go clean --sessions         # Delete OpenCode sessions older than 7 days
  orch-go clean --sessions --sessions-days 14  # Delete sessions older than 14 days
  orch-go clean --sessions --preserve-orchestrator  # Clean sessions but protect orchestrators
  orch-go clean --processes                         # Kill orphaned bun processes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If --all is specified, enable all cleanup flags
		if cleanAll {
			cleanWindows = true
			cleanPhantoms = true
			cleanVerifyOpenCode = true
			cleanInvestigations = true
			cleanStale = true
			cleanUntracked = true
			cleanSessions = true
			cleanProcesses = true
		}
		return runClean(cleanDryRun, cleanVerifyOpenCode, cleanWindows, cleanPhantoms, cleanInvestigations, cleanStale, cleanStaleDays, cleanUntracked, cleanUntrackedDays, cleanSessions, cleanSessionsDays, cleanPreserveOrchestrator, cleanProcesses)
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "Show what would be cleaned without making changes")
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Enable all cleanup actions (windows, phantoms, verify-opencode, investigations, stale, untracked, sessions, processes)")
	cleanCmd.Flags().BoolVar(&cleanVerifyOpenCode, "verify-opencode", false, "Also verify OpenCode disk sessions (slower)")
	cleanCmd.Flags().BoolVar(&cleanWindows, "windows", false, "Close tmux windows for completed agents")
	cleanCmd.Flags().BoolVar(&cleanPhantoms, "phantoms", false, "Close all phantom tmux windows (stale agent windows)")
	cleanCmd.Flags().BoolVar(&cleanInvestigations, "investigations", false, "Archive empty investigation files to .kb/investigations/archived/")
	cleanCmd.Flags().BoolVar(&cleanStale, "stale", false, "Archive completed workspaces older than N days (default: 7)")
	cleanCmd.Flags().IntVar(&cleanStaleDays, "stale-days", 7, "Age threshold in days for --stale (default: 7)")
	cleanCmd.Flags().BoolVar(&cleanUntracked, "untracked", false, "Archive untracked workspaces older than N days (default: 7)")
	cleanCmd.Flags().IntVar(&cleanUntrackedDays, "untracked-days", 7, "Age threshold in days for --untracked (default: 7)")
	cleanCmd.Flags().BoolVar(&cleanSessions, "sessions", false, "Delete stale OpenCode sessions older than N days (default: 7)")
	cleanCmd.Flags().IntVar(&cleanSessionsDays, "sessions-days", 7, "Age threshold in days for --sessions (default: 7)")
	cleanCmd.Flags().BoolVar(&cleanPreserveOrchestrator, "preserve-orchestrator", false, "Skip orchestrator/meta-orchestrator workspaces and sessions")
	cleanCmd.Flags().BoolVar(&cleanProcesses, "processes", false, "Kill orphaned bun processes (agent processes without active sessions)")
}

// DefaultLivenessChecker checks if tmux windows and OpenCode sessions exist.
type DefaultLivenessChecker struct {
	client opencode.ClientInterface
}

// NewDefaultLivenessChecker creates a new liveness checker.
func NewDefaultLivenessChecker(serverURL string) *DefaultLivenessChecker {
	return &DefaultLivenessChecker{
		client: opencode.NewClient(serverURL),
	}
}

// WindowExists checks if a tmux window ID exists.
func (c *DefaultLivenessChecker) WindowExists(windowID string) bool {
	return tmux.WindowExistsByID(windowID)
}

// SessionExists checks if an OpenCode session ID exists.
func (c *DefaultLivenessChecker) SessionExists(sessionID string) bool {
	return c.client.SessionExists(sessionID)
}

// DefaultBeadsStatusChecker checks beads issue status using the verify package.
type DefaultBeadsStatusChecker struct{}

// NewDefaultBeadsStatusChecker creates a new beads status checker.
func NewDefaultBeadsStatusChecker() *DefaultBeadsStatusChecker {
	return &DefaultBeadsStatusChecker{}
}

// IsIssueClosed checks if a beads issue is closed.
func (c *DefaultBeadsStatusChecker) IsIssueClosed(beadsID string) bool {
	issue, err := verify.GetIssue(beadsID)
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
	complete, err := verify.IsPhaseComplete(beadsID)
	if err != nil {
		// If we can't check (e.g., beads error), assume not complete
		return false
	}
	return complete
}

// CleanableWorkspace represents a workspace that can be cleaned.
type CleanableWorkspace struct {
	Name       string // Workspace directory name
	Path       string // Full path to workspace
	BeadsID    string // Beads issue ID (extracted from SPAWN_CONTEXT.md)
	IsComplete bool   // Has SYNTHESIS.md
	Reason     string // Why it's cleanable
}

// findCleanableWorkspaces scans .orch/workspace/ for completed/abandoned workspaces.
// Returns workspaces that have SYNTHESIS.md OR whose beads issue is closed.
// Uses batch beads lookup for performance (~16s -> ~1s with 400+ workspaces).
func findCleanableWorkspaces(projectDir string, beadsChecker *DefaultBeadsStatusChecker) []CleanableWorkspace {
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return nil
	}

	var cleanable []CleanableWorkspace
	var needsBeadsCheck []CleanableWorkspace

	// First pass: Check file-based completion (fast)
	// Collect workspaces that need beads status check
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip the archived directory
		if entry.Name() == "archived" {
			continue
		}

		dirName := entry.Name()
		dirPath := filepath.Join(workspaceDir, dirName)

		// Extract beads ID from SPAWN_CONTEXT.md
		beadsID := ""
		spawnContextPath := filepath.Join(dirPath, "SPAWN_CONTEXT.md")
		if content, err := os.ReadFile(spawnContextPath); err == nil {
			// Look for "beads issue: **xxx**" pattern
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.Contains(line, "beads issue:") || strings.Contains(line, "BEADS ISSUE:") {
					// Extract beads ID from the line
					parts := strings.Fields(line)
					for _, part := range parts {
						// Look for pattern like "orch-go-xxxx" or similar
						if strings.Contains(part, "-") && !strings.HasPrefix(part, "beads") && !strings.HasPrefix(part, "BEADS") {
							// Clean up markdown formatting
							beadsID = strings.Trim(part, "*`[]")
							break
						}
					}
				}
			}
		}

		workspace := CleanableWorkspace{
			Name:    dirName,
			Path:    dirPath,
			BeadsID: beadsID,
		}

		// Check for SYNTHESIS.md (completion indicator) - fast file check
		synthesisPath := filepath.Join(dirPath, "SYNTHESIS.md")
		if info, err := os.Stat(synthesisPath); err == nil && info.Size() > 0 {
			workspace.IsComplete = true
			workspace.Reason = "SYNTHESIS.md exists"
			cleanable = append(cleanable, workspace)
			continue
		}

		// Queue for beads status check if we have a beads ID
		if beadsID != "" {
			needsBeadsCheck = append(needsBeadsCheck, workspace)
		}
	}

	// Second pass: Batch beads status check (optimized)
	// Use ListOpenIssues to get all open issues in a single API call
	// If a beads ID is NOT in the open issues map, it's closed
	if len(needsBeadsCheck) > 0 {
		openIssues, err := verify.ListOpenIssues()
		if err != nil {
			// Fallback to sequential check if batch fails
			for _, ws := range needsBeadsCheck {
				if beadsChecker.IsIssueClosed(ws.BeadsID) {
					ws.IsComplete = true
					ws.Reason = "beads issue closed"
					cleanable = append(cleanable, ws)
				}
			}
		} else {
			// Check if each beads ID is NOT in open issues (= closed)
			for _, ws := range needsBeadsCheck {
				if _, isOpen := openIssues[ws.BeadsID]; !isOpen {
					ws.IsComplete = true
					ws.Reason = "beads issue closed"
					cleanable = append(cleanable, ws)
				}
			}
		}
	}

	return cleanable
}

func runClean(dryRun bool, verifyOpenCode bool, closeWindows bool, cleanPhantoms bool, cleanInvestigations bool, archiveStale bool, staleDays int, archiveUntracked bool, untrackedDays int, cleanSessions bool, sessionsDays int, preserveOrchestrator bool, killProcesses bool) error {
	projectDir, err := currentProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Skip slow beads status check if only doing stale/untracked archival or investigations
	// These operations do their own completion checks
	needsCompletedWorkspaces := closeWindows || (!archiveStale && !archiveUntracked && !cleanPhantoms && !verifyOpenCode && !cleanInvestigations)

	// Track cleanup stats
	windowsClosed := 0
	var cleanableWorkspaces []CleanableWorkspace

	if needsCompletedWorkspaces {
		// Find completed workspaces using derived lookups (slow due to beads API calls)
		fmt.Println("Scanning workspaces for completed agents...")
		beadsChecker := NewDefaultBeadsStatusChecker()
		cleanableWorkspaces = findCleanableWorkspaces(projectDir, beadsChecker)

		fmt.Printf("\nFound %d completed workspaces\n", len(cleanableWorkspaces))

		if len(cleanableWorkspaces) == 0 && !verifyOpenCode && !cleanPhantoms && !cleanInvestigations && !archiveStale && !archiveUntracked {
			fmt.Println("No completed agents found")
			return nil
		}

		// List completed workspaces
		if len(cleanableWorkspaces) > 0 {
			fmt.Printf("\nCompleted workspaces:\n")
			for _, ws := range cleanableWorkspaces {
				fmt.Printf("  %s (%s)\n", ws.Name, ws.Reason)

				// Close tmux window if --windows flag is set
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

	// Verify and clean OpenCode disk sessions (optional)
	var diskSessionsDeleted int
	if verifyOpenCode {
		diskSessionsDeleted, err = cleanOrphanedDiskSessions(serverURL, dryRun, preserveOrchestrator)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean disk sessions: %v\n", err)
		}
	}

	// Clean phantom tmux windows (optional)
	var phantomsClosed int
	if cleanPhantoms {
		phantomsClosed, err = cleanPhantomWindows(serverURL, dryRun, preserveOrchestrator)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean phantom windows: %v\n", err)
		}
	}

	// Clean empty investigation files (optional)
	var investigationsArchived int
	if cleanInvestigations {
		investigationsArchived, err = archiveEmptyInvestigations(projectDir, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to archive empty investigations: %v\n", err)
		}
	}

	// Archive stale workspaces (optional)
	var workspacesArchived int
	if archiveStale {
		workspacesArchived, err = archiveStaleWorkspaces(projectDir, staleDays, dryRun, preserveOrchestrator)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to archive stale workspaces: %v\n", err)
		}
	}

	// Archive untracked workspaces (optional)
	var untrackedArchived int
	if archiveUntracked {
		untrackedArchived, err = archiveUntrackedWorkspaces(projectDir, untrackedDays, dryRun, preserveOrchestrator)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to archive untracked workspaces: %v\n", err)
		}
	}

	// Clean stale OpenCode sessions (optional)
	var staleSessionsDeleted int
	if cleanSessions {
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

	// Kill orphan bun processes (optional)
	var processesKilled int
	if killProcesses {
		processesKilled, err = cleanOrphanProcesses(serverURL, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean orphan processes: %v\n", err)
		}
	}

	// Check if any cleanup actions were taken or would be taken
	hasCleanupActions := closeWindows || cleanPhantoms || verifyOpenCode || cleanInvestigations || archiveStale || archiveUntracked || cleanSessions || killProcesses

	if dryRun {
		if hasCleanupActions {
			fmt.Printf("\nDry run complete.")
			if closeWindows {
				// Count potential windows to close
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

	// Log if any cleanup actions were taken
	if windowsClosed > 0 || phantomsClosed > 0 || diskSessionsDeleted > 0 || investigationsArchived > 0 || workspacesArchived > 0 || untrackedArchived > 0 || staleSessionsDeleted > 0 || processesKilled > 0 {
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
				"untracked_archived":      untrackedArchived,
				"project":                 projectName,
				"verify_opencode":         verifyOpenCode,
				"close_windows":           closeWindows,
				"clean_phantoms":          cleanPhantoms,
				"clean_investigations":    cleanInvestigations,
				"archive_stale":           archiveStale,
				"stale_days":              staleDays,
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

	// Print summary of actions taken (not misleading "cleaned X workspaces")
	if windowsClosed > 0 || phantomsClosed > 0 || diskSessionsDeleted > 0 || investigationsArchived > 0 || workspacesArchived > 0 || untrackedArchived > 0 || staleSessionsDeleted > 0 || processesKilled > 0 {
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
		// Default: just listing completed workspaces
		fmt.Printf("\nNote: Workspace directories are preserved. Use --windows, --phantoms, --verify-opencode, --investigations, --stale, or --untracked to clean up resources.\n")
	}

	return nil
}

// cleanOrphanedDiskSessions finds and deletes OpenCode disk sessions that aren't tracked via workspace files.
// If preserveOrchestrator is true, sessions associated with orchestrator workspaces are skipped.
// Returns the number of sessions deleted and any error encountered.
func cleanOrphanedDiskSessions(serverURL string, dryRun bool, preserveOrchestrator bool) (int, error) {
	return cleanOrphanedDiskSessionsWithClient(opencode.NewClient(serverURL), dryRun, preserveOrchestrator)
}

func cleanOrphanedDiskSessionsWithClient(client opencode.ClientInterface, dryRun bool, preserveOrchestrator bool) (int, error) {
	// Get current project directory
	projectDir, err := currentProjectDir()
	if err != nil {
		return 0, fmt.Errorf("failed to get current directory: %w", err)
	}

	fmt.Printf("\nVerifying OpenCode disk sessions for %s...\n", projectDir)

	// Fetch all disk sessions for this directory
	diskSessions, err := client.ListDiskSessions(projectDir)
	if err != nil {
		return 0, fmt.Errorf("failed to list disk sessions: %w", err)
	}

	fmt.Printf("  Found %d disk sessions\n", len(diskSessions))

	// Build a set of session IDs that are tracked via workspace files
	// Also track which ones are orchestrator sessions (for --preserve-orchestrator)
	trackedSessionIDs := make(map[string]bool)
	orchestratorSessionIDs := make(map[string]bool)
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	if entries, err := os.ReadDir(workspaceDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				wsPath := filepath.Join(workspaceDir, entry.Name())
				sessionID := spawn.ReadSessionID(wsPath)
				if sessionID != "" {
					trackedSessionIDs[sessionID] = true
					// Check if this is an orchestrator workspace
					if isOrchestratorWorkspace(wsPath) {
						orchestratorSessionIDs[sessionID] = true
					}
				}
			}
		}
	}

	fmt.Printf("  Workspaces track %d session IDs\n", len(trackedSessionIDs))
	if preserveOrchestrator && len(orchestratorSessionIDs) > 0 {
		fmt.Printf("  Found %d orchestrator session IDs to preserve\n", len(orchestratorSessionIDs))
	}

	// Find orphaned sessions (disk sessions not tracked in workspaces)
	// IMPORTANT: Exclude sessions that are actively processing (e.g., the current orchestrator session)
	// The orchestrator/interactive sessions don't have workspace .session_id files, but they're
	// still valid sessions that should not be deleted.
	//
	// We use two heuristics to detect active sessions (no extra API calls needed):
	// 1. Recently updated sessions (within last 5 minutes) - likely in use
	// 2. Sessions that are currently processing (expensive check, only if recently updated)
	var orphanedSessions []opencode.Session
	var skippedActive int
	now := time.Now()
	const recentActivityThreshold = 5 * time.Minute

	for _, session := range diskSessions {
		if !trackedSessionIDs[session.ID] {
			// First, quick check: was this session recently active? (using data we already have)
			updatedAt := time.Unix(session.Time.Updated/1000, 0)
			isRecentlyActive := now.Sub(updatedAt) <= recentActivityThreshold

			if isRecentlyActive {
				// Session is recently active - check if it's actually processing
				// This is the expensive check, but we only do it for recently active sessions
				if client.IsSessionProcessing(session.ID) {
					skippedActive++
					continue
				}
			}
			orphanedSessions = append(orphanedSessions, session)
		}
	}

	if skippedActive > 0 {
		fmt.Printf("  Skipped %d active sessions (currently processing)\n", skippedActive)
	}

	if len(orphanedSessions) == 0 {
		fmt.Println("  No orphaned disk sessions found")
		return 0, nil
	}

	fmt.Printf("  Found %d orphaned disk sessions:\n", len(orphanedSessions))

	// Delete orphaned sessions
	deleted := 0
	skippedOrch := 0
	for _, session := range orphanedSessions {
		title := session.Title
		if title == "" {
			title = "(untitled)"
		}

		// Check if this session should be preserved (orchestrator session)
		if preserveOrchestrator && orchestratorSessionIDs[session.ID] {
			skippedOrch++
			continue
		}

		// Also check title for orchestrator indicators (sessions without workspace files)
		if preserveOrchestrator && cleanup.IsOrchestratorSessionTitle(title) {
			skippedOrch++
			continue
		}

		if dryRun {
			fmt.Printf("    [DRY-RUN] Would delete: %s (%s)\n", session.ID[:12], title)
			deleted++
			continue
		}

		if err := client.DeleteSession(session.ID); err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: failed to delete %s: %v\n", session.ID[:12], err)
			continue
		}

		fmt.Printf("    Deleted: %s (%s)\n", session.ID[:12], title)
		deleted++
	}

	if skippedOrch > 0 {
		fmt.Printf("  Skipped %d orchestrator sessions (--preserve-orchestrator)\n", skippedOrch)
	}

	return deleted, nil
}

// cleanPhantomWindows finds and closes tmux windows that are phantoms
// (have a beads ID in the window name but no active OpenCode session).
// If preserveOrchestrator is true, windows in orchestrator/meta-orchestrator sessions are skipped.
// Returns the number of windows closed and any error encountered.
func cleanPhantomWindows(serverURL string, dryRun bool, preserveOrchestrator bool) (int, error) {
	return cleanPhantomWindowsWithClient(opencode.NewClient(serverURL), dryRun, preserveOrchestrator)
}

func cleanPhantomWindowsWithClient(client opencode.ClientInterface, dryRun bool, preserveOrchestrator bool) (int, error) {
	now := time.Now()
	const maxIdleTime = 30 * time.Minute

	fmt.Println("\nScanning for phantom tmux windows...")

	// Get all OpenCode sessions and build a map of recently active beads IDs
	sessions, err := client.ListSessions("")
	if err != nil {
		return 0, fmt.Errorf("failed to list sessions: %w", err)
	}

	activeBeadsIDs := make(map[string]bool)
	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if now.Sub(updatedAt) <= maxIdleTime {
			beadsID := extractBeadsIDFromTitle(s.Title)
			if beadsID != "" {
				activeBeadsIDs[beadsID] = true
			}
		}
	}

	fmt.Printf("  Found %d active OpenCode sessions\n", len(activeBeadsIDs))

	// Scan all workers sessions for phantom windows
	var phantomWindows []struct {
		window      *tmux.WindowInfo
		sessionName string
		beadsID     string
	}

	skippedOrch := 0
	workersSessions, _ := tmux.ListWorkersSessions()
	for _, sessionName := range workersSessions {
		// Skip orchestrator and meta-orchestrator sessions entirely
		if preserveOrchestrator && (sessionName == tmux.OrchestratorSessionName || sessionName == tmux.MetaOrchestratorSessionName) {
			skippedOrch++
			continue
		}

		windows, err := tmux.ListWindows(sessionName)
		if err != nil {
			continue
		}

		for _, w := range windows {
			// Skip known non-agent windows
			if w.Name == "servers" || w.Name == "zsh" {
				continue
			}

			beadsID := extractBeadsIDFromWindowName(w.Name)
			if beadsID == "" {
				continue
			}

			// If beads ID is not in active sessions, it's a phantom
			if !activeBeadsIDs[beadsID] {
				windowCopy := w
				phantomWindows = append(phantomWindows, struct {
					window      *tmux.WindowInfo
					sessionName string
					beadsID     string
				}{&windowCopy, sessionName, beadsID})
			}
		}
	}

	if skippedOrch > 0 {
		fmt.Printf("  Skipped %d orchestrator sessions (--preserve-orchestrator)\n", skippedOrch)
	}

	if len(phantomWindows) == 0 {
		fmt.Println("  No phantom windows found")
		return 0, nil
	}

	fmt.Printf("  Found %d phantom windows:\n", len(phantomWindows))

	// Close phantom windows
	closed := 0
	for _, pw := range phantomWindows {
		if dryRun {
			fmt.Printf("    [DRY-RUN] Would close: %s:%s\n", pw.sessionName, pw.window.Name)
			closed++
			continue
		}

		if err := tmux.KillWindow(pw.window.Target); err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: failed to close %s: %v\n", pw.window.Name, err)
			continue
		}

		fmt.Printf("    Closed: %s:%s\n", pw.sessionName, pw.window.Name)
		closed++
	}

	return closed, nil
}

// emptyInvestigationPlaceholders are patterns that indicate an investigation file was never filled in.
// These are template placeholders from kb create investigation that agents should replace.
var emptyInvestigationPlaceholders = []string{
	"[Brief, descriptive title]",
	"[Clear, specific question",
	"[Concrete observations, data, examples]",
	"[File paths with line numbers",
	"[Explanation of the insight",
}

// isEmptyInvestigation checks if an investigation file still has template placeholders.
// Returns true if the file contains multiple placeholder patterns, indicating it was never filled in.
func isEmptyInvestigation(path string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	contentStr := string(content)
	placeholderCount := 0
	for _, placeholder := range emptyInvestigationPlaceholders {
		if strings.Contains(contentStr, placeholder) {
			placeholderCount++
		}
	}

	// Require at least 2 placeholder patterns to be considered empty
	// (to avoid false positives from files that just mention placeholders in documentation)
	return placeholderCount >= 2
}

// archiveEmptyInvestigations moves empty investigation files to .kb/investigations/archived/.
// Returns the number of files archived and any error encountered.
func archiveEmptyInvestigations(projectDir string, dryRun bool) (int, error) {
	investigationsDir := filepath.Join(projectDir, ".kb", "investigations")
	archivedDir := filepath.Join(investigationsDir, "archived")

	// Check if investigations directory exists
	if _, err := os.Stat(investigationsDir); os.IsNotExist(err) {
		fmt.Println("\nNo .kb/investigations directory found")
		return 0, nil
	}

	fmt.Println("\nScanning for empty investigation files...")

	// Find all empty investigation files
	var emptyFiles []string
	err := filepath.Walk(investigationsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories and non-markdown files
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		// Skip files already in archived folder
		if strings.Contains(path, "/archived/") {
			return nil
		}

		if isEmptyInvestigation(path) {
			emptyFiles = append(emptyFiles, path)
		}

		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to scan investigations: %w", err)
	}

	if len(emptyFiles) == 0 {
		fmt.Println("  No empty investigation files found")
		return 0, nil
	}

	fmt.Printf("  Found %d empty investigation files:\n", len(emptyFiles))

	// Create archived directory if needed
	if !dryRun {
		if err := os.MkdirAll(archivedDir, 0755); err != nil {
			return 0, fmt.Errorf("failed to create archived directory: %w", err)
		}
	}

	// Archive empty files
	archived := 0
	for _, path := range emptyFiles {
		filename := filepath.Base(path)

		// Preserve subdirectory structure (e.g., simple/)
		relPath, _ := filepath.Rel(investigationsDir, path)
		destDir := filepath.Join(archivedDir, filepath.Dir(relPath))
		destPath := filepath.Join(destDir, filename)

		if dryRun {
			fmt.Printf("    [DRY-RUN] Would archive: %s\n", relPath)
			archived++
			continue
		}

		// Create destination subdirectory if needed
		if err := os.MkdirAll(destDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: failed to create directory %s: %v\n", destDir, err)
			continue
		}

		// Check if destination already exists
		finalDestPath := destPath
		if _, err := os.Stat(destPath); err == nil {
			// Destination exists - add timestamp suffix to make it unique
			suffix := time.Now().Format("150405") // HHMMSS format
			// Insert suffix before .md extension
			baseName := strings.TrimSuffix(filename, ".md")
			finalDestPath = filepath.Join(destDir, baseName+"-"+suffix+".md")
			fmt.Printf("    Note: Archive destination exists, using: %s-%s.md\n", baseName, suffix)
		}

		// Move file to archived
		if err := os.Rename(path, finalDestPath); err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: failed to archive %s: %v\n", relPath, err)
			continue
		}

		fmt.Printf("    Archived: %s\n", relPath)
		archived++
	}

	return archived, nil
}

// archiveStaleWorkspaces moves old completed workspaces to .orch/workspace/archived/.
// A workspace is considered "stale" if:
// 1. It has a .spawn_time older than staleDays
// 2. It is completed (SYNTHESIS.md exists OR beads issue is closed)
// If preserveOrchestrator is true, orchestrator/meta-orchestrator workspaces are skipped.
// Returns the number of workspaces archived and any error encountered.
func archiveStaleWorkspaces(projectDir string, staleDays int, dryRun bool, preserveOrchestrator bool) (int, error) {
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	archivedDir := filepath.Join(workspaceDir, "archived")

	// Check if workspace directory exists
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		fmt.Println("\nNo .orch/workspace directory found")
		return 0, nil
	}

	fmt.Printf("\nScanning for stale workspaces (older than %d days)...\n", staleDays)

	// Calculate the cutoff time
	cutoff := time.Now().AddDate(0, 0, -staleDays)

	// Find stale workspaces
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read workspace directory: %w", err)
	}

	// NOTE: We use file-based indicators only (no beads API calls) for performance.
	// For stale workspaces (7+ days old), we accept:
	// 1. SYNTHESIS.md exists → completed full-tier spawn
	// 2. Light tier (.tier = "light") → no SYNTHESIS.md required by design
	// 3. Has .beads_id file → tracked spawn (was a real agent, not a test)
	// This avoids slow beads API calls while still being conservative.
	var staleWorkspaces []struct {
		name      string
		path      string
		spawnTime time.Time
		reason    string
	}

	skippedOrch := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip the archived directory itself
		if entry.Name() == "archived" {
			continue
		}

		dirPath := filepath.Join(workspaceDir, entry.Name())

		// Skip orchestrator workspaces if --preserve-orchestrator is set
		if preserveOrchestrator && isOrchestratorWorkspace(dirPath) {
			skippedOrch++
			continue
		}

		// Read spawn time
		spawnTimeFile := filepath.Join(dirPath, ".spawn_time")
		spawnTimeData, err := os.ReadFile(spawnTimeFile)
		if err != nil {
			continue // Skip workspaces without spawn time
		}

		// Parse spawn time (nanoseconds)
		var spawnTimeNs int64
		if _, err := fmt.Sscanf(string(spawnTimeData), "%d", &spawnTimeNs); err != nil {
			continue
		}
		spawnTime := time.Unix(0, spawnTimeNs)

		// Check if workspace is old enough
		if spawnTime.After(cutoff) {
			continue // Not stale yet
		}

		// Check if workspace is completed (using file-based indicators only for speed)
		reason := ""

		// Check for SYNTHESIS.md (full-tier completion)
		synthesisPath := filepath.Join(dirPath, "SYNTHESIS.md")
		if info, err := os.Stat(synthesisPath); err == nil && info.Size() > 0 {
			reason = "SYNTHESIS.md exists"
		}

		// Check for light tier (light tier doesn't require SYNTHESIS.md by design)
		if reason == "" {
			tierFile := filepath.Join(dirPath, ".tier")
			if tierData, err := os.ReadFile(tierFile); err == nil {
				tier := strings.TrimSpace(string(tierData))
				if tier == "light" {
					reason = "light tier (no SYNTHESIS.md required)"
				}
			}
		}

		// Check for .beads_id file (indicates tracked spawn)
		if reason == "" {
			beadsIDFile := filepath.Join(dirPath, ".beads_id")
			if _, err := os.Stat(beadsIDFile); err == nil {
				reason = "tracked spawn (has .beads_id)"
			}
		}

		if reason == "" {
			continue // Not completed, don't archive
		}

		staleWorkspaces = append(staleWorkspaces, struct {
			name      string
			path      string
			spawnTime time.Time
			reason    string
		}{
			name:      entry.Name(),
			path:      dirPath,
			spawnTime: spawnTime,
			reason:    reason,
		})
	}

	if skippedOrch > 0 {
		fmt.Printf("  Skipped %d orchestrator workspaces (--preserve-orchestrator)\n", skippedOrch)
	}

	if len(staleWorkspaces) == 0 {
		fmt.Println("  No stale completed workspaces found")
		return 0, nil
	}

	fmt.Printf("  Found %d stale workspaces:\n", len(staleWorkspaces))

	// Create archived directory if needed
	if !dryRun {
		if err := os.MkdirAll(archivedDir, 0755); err != nil {
			return 0, fmt.Errorf("failed to create archived directory: %w", err)
		}
	}

	// Archive stale workspaces
	archived := 0
	for _, ws := range staleWorkspaces {
		destPath := filepath.Join(archivedDir, ws.name)
		age := time.Since(ws.spawnTime).Hours() / 24

		if dryRun {
			fmt.Printf("    [DRY-RUN] Would archive: %s (%.0f days old, %s)\n", ws.name, age, ws.reason)
			archived++
			continue
		}

		// Check if destination already exists
		finalDestPath := destPath
		if _, err := os.Stat(destPath); err == nil {
			// Destination exists - add timestamp suffix to make it unique
			suffix := time.Now().Format("150405") // HHMMSS format
			finalDestPath = destPath + "-" + suffix
			fmt.Printf("    Note: Archive destination exists, using: %s-%s\n", ws.name, suffix)
		}

		// Move workspace to archived
		if err := os.Rename(ws.path, finalDestPath); err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: failed to archive %s: %v\n", ws.name, err)
			continue
		}

		fmt.Printf("    Archived: %s (%.0f days old, %s)\n", ws.name, age, ws.reason)
		archived++
	}

	return archived, nil
}

// archiveUntrackedWorkspaces moves old untracked workspaces to .orch/workspace/archived/.
// A workspace is considered "untracked" if:
// 1. It has no beads ID in SPAWN_CONTEXT.md
// 2. OR it has a beads ID containing "-untracked-" (spawned with --no-track)
// If preserveOrchestrator is true, orchestrator/meta-orchestrator workspaces are skipped.
// Returns the number of workspaces archived and any error encountered.
func archiveUntrackedWorkspaces(projectDir string, untrackedDays int, dryRun bool, preserveOrchestrator bool) (int, error) {
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	archivedDir := filepath.Join(workspaceDir, "archived")

	// Check if workspace directory exists
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		fmt.Println("\nNo .orch/workspace directory found")
		return 0, nil
	}

	fmt.Printf("\nScanning for untracked workspaces (older than %d days)...\n", untrackedDays)

	// Calculate the cutoff time
	cutoff := time.Now().AddDate(0, 0, -untrackedDays)

	// Find untracked workspaces
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read workspace directory: %w", err)
	}

	var untrackedWorkspaces []struct {
		name      string
		path      string
		spawnTime time.Time
		beadsID   string
	}

	skippedOrch := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip the archived directory itself
		if entry.Name() == "archived" {
			continue
		}

		dirPath := filepath.Join(workspaceDir, entry.Name())

		// Skip orchestrator workspaces if --preserve-orchestrator is set
		if preserveOrchestrator && isOrchestratorWorkspace(dirPath) {
			skippedOrch++
			continue
		}

		// Extract beads ID from SPAWN_CONTEXT.md
		beadsID := extractBeadsIDFromWorkspace(dirPath)

		// Check if workspace is untracked
		// Untracked means: no beads ID OR beads ID contains "-untracked-"
		isUntracked := beadsID == "" || isUntrackedBeadsID(beadsID)
		if !isUntracked {
			continue // Skip tracked workspaces
		}

		// Read spawn time
		spawnTimeFile := filepath.Join(dirPath, ".spawn_time")
		spawnTimeData, err := os.ReadFile(spawnTimeFile)
		if err != nil {
			continue // Skip workspaces without spawn time
		}

		// Parse spawn time (nanoseconds)
		var spawnTimeNs int64
		if _, err := fmt.Sscanf(string(spawnTimeData), "%d", &spawnTimeNs); err != nil {
			continue
		}
		spawnTime := time.Unix(0, spawnTimeNs)

		// Check if workspace is old enough
		if spawnTime.After(cutoff) {
			continue // Not old enough yet
		}

		untrackedWorkspaces = append(untrackedWorkspaces, struct {
			name      string
			path      string
			spawnTime time.Time
			beadsID   string
		}{
			name:      entry.Name(),
			path:      dirPath,
			spawnTime: spawnTime,
			beadsID:   beadsID,
		})
	}

	if skippedOrch > 0 {
		fmt.Printf("  Skipped %d orchestrator workspaces (--preserve-orchestrator)\n", skippedOrch)
	}

	if len(untrackedWorkspaces) == 0 {
		fmt.Println("  No untracked workspaces found")
		return 0, nil
	}

	fmt.Printf("  Found %d untracked workspaces:\n", len(untrackedWorkspaces))

	// Create archived directory if needed
	if !dryRun {
		if err := os.MkdirAll(archivedDir, 0755); err != nil {
			return 0, fmt.Errorf("failed to create archived directory: %w", err)
		}
	}

	// Archive untracked workspaces
	archived := 0
	for _, ws := range untrackedWorkspaces {
		destPath := filepath.Join(archivedDir, ws.name)
		age := time.Since(ws.spawnTime).Hours() / 24

		beadsDisplay := "no beads ID"
		if ws.beadsID != "" {
			beadsDisplay = formatBeadsIDForDisplay(ws.beadsID)
		}

		if dryRun {
			fmt.Printf("    [DRY-RUN] Would archive: %s (%.0f days old, %s)\n", ws.name, age, beadsDisplay)
			archived++
			continue
		}

		// Check if destination already exists
		finalDestPath := destPath
		if _, err := os.Stat(destPath); err == nil {
			// Destination exists - add timestamp suffix to make it unique
			suffix := time.Now().Format("150405") // HHMMSS format
			finalDestPath = destPath + "-" + suffix
			fmt.Printf("    Note: Archive destination exists, using: %s-%s\n", ws.name, suffix)
		}

		// Move workspace to archived
		if err := os.Rename(ws.path, finalDestPath); err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: failed to archive %s: %v\n", ws.name, err)
			continue
		}

		fmt.Printf("    Archived: %s (%.0f days old, %s)\n", ws.name, age, beadsDisplay)
		archived++
	}

	return archived, nil
}

// cleanOrphanProcesses finds and kills bun agent processes that are not associated
// with any active OpenCode session. Returns the number of processes killed.
func cleanOrphanProcesses(serverURL string, dryRun bool) (int, error) {
	return cleanOrphanProcessesWithClient(opencode.NewClient(serverURL), dryRun)
}

func cleanOrphanProcessesWithClient(client opencode.ClientInterface, dryRun bool) (int, error) {

	fmt.Println("\nScanning for orphan bun processes...")

	// Get all active OpenCode sessions to build a set of active titles
	sessions, err := client.ListSessions("")
	if err != nil {
		return 0, fmt.Errorf("failed to list sessions: %w", err)
	}

	// Build set of active session titles (workspace names and beads IDs)
	activeTitles := make(map[string]bool)
	for _, s := range sessions {
		title := s.Title
		if title == "" {
			continue
		}
		activeTitles[title] = true
		// Also extract workspace name from title (format: "workspace-name [beads-id]")
		if idx := strings.Index(title, " ["); idx != -1 {
			activeTitles[strings.TrimSpace(title[:idx])] = true
		}
	}

	fmt.Printf("  Found %d active OpenCode sessions\n", len(sessions))

	// Find orphan processes
	orphans, err := process.FindOrphanProcesses(activeTitles)
	if err != nil {
		return 0, fmt.Errorf("failed to find orphan processes: %w", err)
	}

	if len(orphans) == 0 {
		fmt.Println("  No orphan bun processes found")
		return 0, nil
	}

	fmt.Printf("  Found %d orphan bun processes:\n", len(orphans))

	killed := 0
	for _, orphan := range orphans {
		name := orphan.WorkspaceName
		if name == "" {
			name = "(unknown)"
		}
		beadsInfo := ""
		if orphan.BeadsID != "" {
			beadsInfo = fmt.Sprintf(" [%s]", orphan.BeadsID)
		}

		if dryRun {
			fmt.Printf("    [DRY-RUN] Would kill: PID %d (%s%s)\n", orphan.PID, name, beadsInfo)
			killed++
			continue
		}

		if process.Terminate(orphan.PID, "bun (orphan)") {
			fmt.Printf("    Killed: PID %d (%s%s)\n", orphan.PID, name, beadsInfo)
			killed++
		} else {
			fmt.Printf("    Already dead: PID %d (%s%s)\n", orphan.PID, name, beadsInfo)
		}
	}

	return killed, nil
}

// NOTE: extractBeadsIDFromWorkspace is defined in review.go
