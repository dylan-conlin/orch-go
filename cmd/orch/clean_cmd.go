// Package main provides the clean command for cleaning up completed agent resources.
// Extracted from main.go as part of the main.go refactoring (Phase 4).
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/agent"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
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
		return runClean(cleanDryRun, cleanWorkspaces, cleanSessions, cleanOrphans, cleanGhosts, cleanWorkspaceDays, cleanSessionDays, cleanPreserveOrchestrator)
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
	cleanCmd.Flags().BoolVar(&cleanPreserveOrchestrator, "preserve-orchestrator", false, "Skip orchestrator/meta-orchestrator workspaces and sessions")
}

// DefaultLivenessChecker checks if tmux windows and OpenCode sessions exist.
type DefaultLivenessChecker struct {
	client *opencode.Client
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

func runClean(dryRun bool, doWorkspaces bool, doSessions bool, doOrphans bool, doGhosts bool, workspaceDays int, sessionDays int, preserveOrchestrator bool) error {
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
	var workspacesArchived, investigationsArchived int
	var windowsClosed int
	var orphansForceCompleted, orphansForceAbandoned int
	var ghostsCleaned int

	// --workspaces: Archive old workspaces + empty investigations
	if doWorkspaces {
		workspacesArchived, err = archiveStaleWorkspaces(projectDir, workspaceDays, dryRun, preserveOrchestrator)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to archive stale workspaces: %v\n", err)
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
	totalCleaned := workspacesArchived + investigationsArchived + windowsClosed + orphansForceCompleted + orphansForceAbandoned + ghostsCleaned
	if totalCleaned > 0 {
		projectName := filepath.Base(projectDir)
		logger := events.NewLogger(events.DefaultLogPath())
		event := events.Event{
			Type:      "agents.cleaned",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"workspaces_archived":        workspacesArchived,
				"investigations_archived":    investigationsArchived,
				"windows_closed":             windowsClosed,
				"orphans_force_completed":    orphansForceCompleted,
				"orphans_force_abandoned":    orphansForceAbandoned,
				"project":                    projectName,
				"clean_workspaces":           doWorkspaces,
				"clean_sessions":             doSessions,
				"clean_orphans":              doOrphans,
				"clean_ghosts":              doGhosts,
				"ghosts_cleaned":            ghostsCleaned,
				"workspace_days":             workspaceDays,
				"session_days":               sessionDays,
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

// cleanUntrackedDiskSessions finds and deletes OpenCode disk sessions that aren't tracked via workspace files.
// If preserveOrchestrator is true, sessions associated with orchestrator workspaces are skipped.
// Returns the number of sessions deleted and any error encountered.
// cleanUntrackedDiskSessions has been removed - OpenCode now handles session cleanup via TTL
// (see opencode-fork commit f3c3865)

// PaneProcessChecker determines if a tmux window's pane has an active (non-shell) process.
// This is used as a last-resort safety check before classifying a window as stale.
type PaneProcessChecker interface {
	HasActiveProcess(windowID string) bool
}

// DefaultPaneProcessChecker checks process liveness via two signals:
// 1. tmux pane_current_command — if a non-shell process is in the foreground, the pane is active
// 2. child process detection — if the pane's shell has child processes, an agent is running
//
// The dual approach is needed because macOS tmux may report "zsh" for pane_current_command
// even when a child process (like claude) is actively running in the foreground.
type DefaultPaneProcessChecker struct{}

// idleShellCommands are process names that indicate a pane is idle (no agent running).
// These are shells that remain after an agent process exits.
var idleShellCommands = map[string]bool{
	"zsh": true, "bash": true, "sh": true, "fish": true,
	"-zsh": true, "-bash": true, "-sh": true, "login": true,
}

// HasActiveProcess checks if a tmux window has an active agent process.
// Uses two signals: pane_current_command and child process detection.
// Returns true (conservative) if the check fails, to avoid killing active agents.
func (c *DefaultPaneProcessChecker) HasActiveProcess(windowID string) bool {
	// Signal 1: Check pane_current_command — if it's not a shell, something is running.
	cmd, err := tmux.GetPaneCurrentCommand(windowID)
	if err != nil {
		// Can't determine — be conservative, assume alive
		return true
	}
	if !idleShellCommands[cmd] {
		return true
	}

	// Signal 2: pane_current_command shows a shell, but on macOS this can be
	// unreliable when child processes (claude, opencode) are running.
	// Check if the pane's shell PID has any child processes.
	pid, err := tmux.GetPanePID(windowID)
	if err != nil {
		// Can't determine — be conservative, assume alive
		return true
	}
	return hasChildProcesses(pid)
}

// hasChildProcesses checks if a process has any child processes.
// Uses pgrep -P which returns exit 0 if children found, 1 if not.
func hasChildProcesses(pid string) bool {
	cmd := exec.Command("pgrep", "-P", pid)
	err := cmd.Run()
	return err == nil // exit 0 = children found
}

// staleTmuxWindow represents a tmux window identified as stale.
type staleTmuxWindow struct {
	window      *tmux.WindowInfo
	sessionName string
	beadsID     string
}

// classifyTmuxWindows identifies tmux windows that are stale (no active OpenCode session,
// no open beads issue, and no running process). This protects daemon-spawned Claude CLI
// agents that have tmux windows but no corresponding OpenCode sessions.
//
// The process checker is the last-resort safety net: even if OpenCode has zero sessions
// (e.g., server restarted) and the beads issue is closed, a window with a running agent
// process (claude, opencode, etc.) will be protected from cleanup.
func classifyTmuxWindows(
	windows []tmux.WindowInfo,
	sessionName string,
	activeBeadsIDs map[string]bool,
	openIssues map[string]*verify.Issue,
	processChecker PaneProcessChecker,
) (stale []staleTmuxWindow, protected int) {
	for _, w := range windows {
		// Skip known non-agent windows
		if w.Name == "servers" || w.Name == "zsh" {
			continue
		}

		beadsID := extractBeadsIDFromWindowName(w.Name)
		if beadsID == "" {
			continue
		}

		// Has an active OpenCode session? Not stale.
		if activeBeadsIDs[beadsID] {
			continue
		}

		// Has an open beads issue? Protected (likely Claude CLI daemon spawn).
		if _, isOpen := openIssues[beadsID]; isOpen {
			protected++
			continue
		}

		// Has an active process running in the pane? Protected.
		// This catches Claude CLI agents that have no OpenCode session and whose
		// beads issue may be closed, but are still actively running.
		if processChecker != nil && processChecker.HasActiveProcess(w.ID) {
			protected++
			continue
		}

		// Window is truly stale (no OpenCode session AND beads issue not open AND no active process)
		windowCopy := w
		stale = append(stale, staleTmuxWindow{&windowCopy, sessionName, beadsID})
	}
	return stale, protected
}

// cleanStaleTmuxWindows finds and closes tmux windows with no active OpenCode session backing them.
// Windows with open beads issues are protected (handles Claude CLI daemon spawns).
// If preserveOrchestrator is true, windows in orchestrator/meta-orchestrator sessions are skipped.
// Returns the number of windows closed and any error encountered.
func cleanStaleTmuxWindows(serverURL string, projectDir string, dryRun bool, preserveOrchestrator bool) (int, error) {
	client := opencode.NewClient(serverURL)
	now := time.Now()
	const maxIdleTime = 30 * time.Minute

	fmt.Println("\nScanning for stale tmux windows...")

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

	// Get open beads issues to protect active agents without OpenCode sessions
	// (e.g., Claude CLI daemon spawns)
	openIssues, err := verify.ListOpenIssuesWithDir(projectDir)
	if err != nil {
		return 0, fmt.Errorf("failed to check beads issues (needed to protect active agents): %w", err)
	}

	// Scan all workers sessions and classify windows
	var allStale []staleTmuxWindow
	totalProtected := 0

	processChecker := &DefaultPaneProcessChecker{}

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

		stale, protected := classifyTmuxWindows(windows, sessionName, activeBeadsIDs, openIssues, processChecker)
		allStale = append(allStale, stale...)
		totalProtected += protected
	}

	if skippedOrch > 0 {
		fmt.Printf("  Skipped %d orchestrator sessions (--preserve-orchestrator)\n", skippedOrch)
	}

	if totalProtected > 0 {
		fmt.Printf("  Protected %d windows with open beads issues (no OpenCode session)\n", totalProtected)
	}

	if len(allStale) == 0 {
		fmt.Println("  No stale tmux windows found")
		return 0, nil
	}

	fmt.Printf("  Found %d stale tmux windows:\n", len(allStale))

	// Close stale windows
	closed := 0
	for _, pw := range allStale {
		if dryRun {
			fmt.Printf("    [DRY-RUN] Would close: %s:%s\n", pw.sessionName, pw.window.Name)
			closed++
			continue
		}

		if err := tmux.KillWindowByID(pw.window.ID); err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: failed to close %s (%s): %v\n", pw.window.Name, pw.window.ID, err)
			continue
		}

		fmt.Printf("    Closed: %s:%s (%s)\n", pw.sessionName, pw.window.Name, pw.window.ID)
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

		// Read agent state from manifest (falls back to dotfiles)
		manifest := spawn.ReadAgentManifestWithFallback(dirPath)
		spawnTime := manifest.ParseSpawnTime()
		if spawnTime.IsZero() {
			spawnTime, _ = fallbackWorkspaceSpawnTime(dirPath)
			if spawnTime.IsZero() {
				continue // Skip workspaces without usable spawn time
			}
		}

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
		if reason == "" && manifest.Tier == "light" {
			reason = "light tier (no SYNTHESIS.md required)"
		}

		// Check for beads_id (indicates tracked spawn)
		if reason == "" && manifest.BeadsID != "" {
			reason = "tracked spawn (has beads_id)"
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

func fallbackWorkspaceSpawnTime(workspacePath string) (time.Time, string) {
	candidates := []struct {
		name  string
		label string
	}{
		{"SPAWN_CONTEXT.md", "SPAWN_CONTEXT.md mtime"},
		{spawn.AgentManifestFilename, "AGENT_MANIFEST.json mtime"},
		{spawn.SpawnTimeFilename, ".spawn_time mtime"},
	}

	for _, candidate := range candidates {
		info, err := os.Stat(filepath.Join(workspacePath, candidate.name))
		if err == nil {
			return info.ModTime(), candidate.label
		}
	}

	info, err := os.Stat(workspacePath)
	if err == nil {
		return info.ModTime(), "workspace mtime"
	}

	return time.Time{}, ""
}

// detectOrphansReport runs orphan detection and returns formatted report lines.
// Used in default mode (no flags) for reporting only — no GC actions taken.
func detectOrphansReport(projectDir string) ([]string, error) {
	lm := buildLifecycleManager(projectDir, serverURL, "", "")
	result, err := lm.DetectOrphans([]string{projectDir}, 30*time.Minute)
	if err != nil {
		return nil, err
	}
	if len(result.Orphans) == 0 {
		return nil, nil
	}

	var lines []string
	for _, orphan := range result.Orphans {
		action := "force-complete"
		if orphan.ShouldRetry {
			action = "force-abandon"
		}
		detail := orphan.Reason
		if orphan.LastPhase != "" {
			detail += fmt.Sprintf(", phase: %s", orphan.LastPhase)
		}
		if orphan.StaleFor > 0 {
			detail += fmt.Sprintf(", stale %v", orphan.StaleFor.Round(time.Minute))
		}
		lines = append(lines, fmt.Sprintf("%s → %s (%s)", orphan.Agent.BeadsID, action, detail))
	}
	return lines, nil
}

// runOrphanGC detects orphaned agents and performs lifecycle GC transitions.
// Uses LifecycleManager.DetectOrphans to find agents tagged orch:agent with no live
// execution, then applies ForceComplete (for completed orphans) or ForceAbandon
// (for retryable orphans) to clean up state consistently.
func runOrphanGC(projectDir string, dryRun bool, preserveOrchestrator bool) (forceCompleted int, forceAbandoned int, err error) {
	fmt.Println("\nScanning for orphaned agents...")

	// Build lifecycle manager for detection (agentName/beadsID not needed for DetectOrphans)
	lm := buildLifecycleManager(projectDir, serverURL, "", "")

	result, err := lm.DetectOrphans([]string{projectDir}, 30*time.Minute)
	if err != nil {
		return 0, 0, fmt.Errorf("orphan detection failed: %w", err)
	}

	fmt.Printf("  Scanned %d tracked agents in %v\n", result.Scanned, result.Elapsed.Round(time.Millisecond))

	if len(result.Orphans) == 0 {
		fmt.Println("  No orphaned agents found")
		return 0, 0, nil
	}

	fmt.Printf("  Found %d orphaned agents:\n", len(result.Orphans))

	for _, orphan := range result.Orphans {
		// Skip orchestrator workspaces if requested
		if preserveOrchestrator && orphan.Agent.WorkspacePath != "" && isOrchestratorWorkspace(orphan.Agent.WorkspacePath) {
			fmt.Printf("    Skipped (orchestrator): %s\n", orphan.Agent.BeadsID)
			continue
		}

		action := "force-complete"
		if orphan.ShouldRetry {
			action = "force-abandon"
		}

		// Format details for output
		detail := orphan.Reason
		if orphan.LastPhase != "" {
			detail += fmt.Sprintf(", phase: %s", orphan.LastPhase)
		}
		if orphan.StaleFor > 0 {
			detail += fmt.Sprintf(", stale %v", orphan.StaleFor.Round(time.Minute))
		}

		if dryRun {
			fmt.Printf("    [DRY-RUN] Would %s: %s (%s)\n", action, orphan.Agent.BeadsID, detail)
			if orphan.ShouldRetry {
				forceAbandoned++
			} else {
				forceCompleted++
			}
			continue
		}

		// Build per-agent lifecycle manager (workspace adapter needs agent-specific params)
		agentLM := buildLifecycleManager(projectDir, serverURL, orphan.Agent.WorkspaceName, orphan.Agent.BeadsID)

		var event *agent.TransitionEvent
		if orphan.ShouldRetry {
			event, err = agentLM.ForceAbandon(orphan.Agent)
			if err != nil {
				fmt.Fprintf(os.Stderr, "    Warning: force-abandon failed for %s: %v\n", orphan.Agent.BeadsID, err)
				continue
			}
			if event.Success {
				fmt.Printf("    Force-abandoned: %s (will retry via respawn)\n", orphan.Agent.BeadsID)
				forceAbandoned++
			}
		} else {
			reason := fmt.Sprintf("GC: orphaned agent (%s)", detail)
			event, err = agentLM.ForceComplete(orphan.Agent, reason)
			if err != nil {
				fmt.Fprintf(os.Stderr, "    Warning: force-complete failed for %s: %v\n", orphan.Agent.BeadsID, err)
				continue
			}
			if event.Success {
				fmt.Printf("    Force-completed: %s (%s)\n", orphan.Agent.BeadsID, detail)
				forceCompleted++
			}
		}

		// Report effect details
		for _, e := range event.Effects {
			if e.Critical && !e.Success {
				fmt.Fprintf(os.Stderr, "    Warning: %s/%s failed for %s: %v\n", e.Subsystem, e.Operation, orphan.Agent.BeadsID, e.Error)
			}
		}
		for _, w := range event.Warnings {
			fmt.Fprintf(os.Stderr, "    Warning: %s\n", w)
		}
	}

	return forceCompleted, forceAbandoned, nil
}

// NOTE: extractBeadsIDFromWorkspace is defined in review.go

// cleanGhostAgents finds cross-project beads issues with stale orch:agent labels
// and removes the label. A "ghost" is an issue that appears in orch status via
// cross-project beads query (orch:agent label + in_progress) but has no active
// agent working on it (no workspace, no session).
//
// Ghost agents are caused by agents that died without proper cleanup — the
// orch:agent label was never removed. This makes them permanently visible in
// orch status with no way to dismiss them.
func cleanGhostAgents(currentProjectDir string, dryRun bool) (int, error) {
	projectDirs := getKBProjectsFn()
	if len(projectDirs) == 0 {
		return 0, nil
	}

	client := opencode.NewClient(opencode.DefaultServerURL)
	cleaned := 0

	for _, dir := range projectDirs {
		// Skip current project — local orphans are handled by --orphans
		if filepath.Clean(dir) == filepath.Clean(currentProjectDir) {
			continue
		}

		// Find orch:agent labeled issues in this project
		issues, err := beads.FallbackListWithLabelInDir("orch:agent", dir)
		if err != nil {
			continue
		}

		for _, issue := range issues {
			if issue.Status != "open" && issue.Status != "in_progress" {
				continue
			}

			// Check if there's an active workspace for this issue
			wPath, _ := findWorkspaceByBeadsID(dir, issue.ID)

			// Check if there's an active OpenCode session
			hasSession := false
			if wPath != "" {
				sessionID := spawn.ReadSessionID(wPath)
				if sessionID != "" {
					hasSession = client.SessionExists(sessionID)
				}
			}

			// If no workspace and no session, this is a ghost
			if wPath == "" || !hasSession {
				if dryRun {
					fmt.Printf("  Ghost: %s in %s (%s)\n", issue.ID, filepath.Base(dir), issue.Title)
					cleaned++
					continue
				}

				// Remove orch:agent label via bd CLI in target directory
				removeLabelErr := beads.FallbackRemoveLabelInDir(issue.ID, "orch:agent", dir)
				if removeLabelErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to remove orch:agent from %s: %v\n", issue.ID, removeLabelErr)
					continue
				}
				fmt.Printf("  Cleaned ghost: %s in %s\n", issue.ID, filepath.Base(dir))
				cleaned++
			}
		}
	}

	return cleaned, nil
}
