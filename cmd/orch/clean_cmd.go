// Package main provides the clean command for cleaning up completed agent resources.
// Extracted from main.go as part of the main.go refactoring (Phase 4).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	// Clean command flags
	cleanDryRun         bool
	cleanVerifyOpenCode bool
	cleanWindows        bool
	cleanPhantoms       bool
	cleanInvestigations bool
	cleanStale          bool
	cleanStaleDays      int
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

Optional cleanup actions:
  --windows         Close tmux windows for completed agents
  --phantoms        Close phantom tmux windows (beads ID but no active session)
  --verify-opencode Delete orphaned OpenCode disk sessions (not tracked in workspaces)
  --investigations  Archive empty investigation files (agents died before filling template)
  --stale           Archive old completed workspaces (default: 7 days)
  --stale-days N    Set age threshold for --stale (default: 7)

Note: This command never deletes workspace directories - they are kept for 
investigation reference. Use 'rm -rf .orch/workspace/<name>' to manually delete.

Examples:
  orch-go clean                    # List completed agents (no changes)
  orch-go clean --dry-run          # Preview mode (same as default)
  orch-go clean --windows          # Close tmux windows for completed agents
  orch-go clean --phantoms         # Close phantom tmux windows
  orch-go clean --verify-opencode  # Delete orphaned OpenCode disk sessions
  orch-go clean --investigations   # Archive empty investigation templates
  orch-go clean --stale            # Archive workspaces older than 7 days
  orch-go clean --stale --stale-days 14  # Archive workspaces older than 14 days`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClean(cleanDryRun, cleanVerifyOpenCode, cleanWindows, cleanPhantoms, cleanInvestigations, cleanStale, cleanStaleDays)
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "Show what would be cleaned without making changes")
	cleanCmd.Flags().BoolVar(&cleanVerifyOpenCode, "verify-opencode", false, "Also verify OpenCode disk sessions (slower)")
	cleanCmd.Flags().BoolVar(&cleanWindows, "windows", false, "Close tmux windows for completed agents")
	cleanCmd.Flags().BoolVar(&cleanPhantoms, "phantoms", false, "Close all phantom tmux windows (stale agent windows)")
	cleanCmd.Flags().BoolVar(&cleanInvestigations, "investigations", false, "Archive empty investigation files to .kb/investigations/archived/")
	cleanCmd.Flags().BoolVar(&cleanStale, "stale", false, "Archive completed workspaces older than N days (default: 7)")
	cleanCmd.Flags().IntVar(&cleanStaleDays, "stale-days", 7, "Age threshold in days for --stale (default: 7)")
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
func findCleanableWorkspaces(projectDir string, beadsChecker *DefaultBeadsStatusChecker) []CleanableWorkspace {
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return nil
	}

	var cleanable []CleanableWorkspace

	for _, entry := range entries {
		if !entry.IsDir() {
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

		// Check for SYNTHESIS.md (completion indicator)
		synthesisPath := filepath.Join(dirPath, "SYNTHESIS.md")
		if info, err := os.Stat(synthesisPath); err == nil && info.Size() > 0 {
			workspace.IsComplete = true
			workspace.Reason = "SYNTHESIS.md exists"
			cleanable = append(cleanable, workspace)
			continue
		}

		// Check beads issue status if we have a beads ID
		if beadsID != "" && beadsChecker.IsIssueClosed(beadsID) {
			workspace.IsComplete = true
			workspace.Reason = "beads issue closed"
			cleanable = append(cleanable, workspace)
			continue
		}

		// Check if workspace is orphaned (no tmux window, no OpenCode session, no active beads issue)
		// This would be a workspace from a crashed or abandoned agent
		// For now, we only clean explicitly completed workspaces
	}

	return cleanable
}

func runClean(dryRun bool, verifyOpenCode bool, closeWindows bool, cleanPhantoms bool, cleanInvestigations bool, archiveStale bool, staleDays int) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Skip slow beads status check if only doing stale archival or investigations
	// These operations do their own completion checks
	needsCompletedWorkspaces := closeWindows || (!archiveStale && !cleanPhantoms && !verifyOpenCode && !cleanInvestigations)

	// Track cleanup stats
	windowsClosed := 0
	var cleanableWorkspaces []CleanableWorkspace

	if needsCompletedWorkspaces {
		// Find completed workspaces using derived lookups (slow due to beads API calls)
		fmt.Println("Scanning workspaces for completed agents...")
		beadsChecker := NewDefaultBeadsStatusChecker()
		cleanableWorkspaces = findCleanableWorkspaces(projectDir, beadsChecker)

		fmt.Printf("\nFound %d completed workspaces\n", len(cleanableWorkspaces))

		if len(cleanableWorkspaces) == 0 && !verifyOpenCode && !cleanPhantoms && !cleanInvestigations && !archiveStale {
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
		diskSessionsDeleted, err = cleanOrphanedDiskSessions(serverURL, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean disk sessions: %v\n", err)
		}
	}

	// Clean phantom tmux windows (optional)
	var phantomsClosed int
	if cleanPhantoms {
		phantomsClosed, err = cleanPhantomWindows(serverURL, dryRun)
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
		workspacesArchived, err = archiveStaleWorkspaces(projectDir, staleDays, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to archive stale workspaces: %v\n", err)
		}
	}

	// Check if any cleanup actions were taken or would be taken
	hasCleanupActions := closeWindows || cleanPhantoms || verifyOpenCode || cleanInvestigations || archiveStale

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
			fmt.Println()
		}
		return nil
	}

	// Log if any cleanup actions were taken
	if windowsClosed > 0 || phantomsClosed > 0 || diskSessionsDeleted > 0 || investigationsArchived > 0 || workspacesArchived > 0 {
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
				"project":                 projectName,
				"verify_opencode":         verifyOpenCode,
				"close_windows":           closeWindows,
				"clean_phantoms":          cleanPhantoms,
				"clean_investigations":    cleanInvestigations,
				"archive_stale":           archiveStale,
				"stale_days":              staleDays,
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
		}
	}

	// Print summary of actions taken (not misleading "cleaned X workspaces")
	if windowsClosed > 0 || phantomsClosed > 0 || diskSessionsDeleted > 0 || investigationsArchived > 0 || workspacesArchived > 0 {
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
	} else if !hasCleanupActions {
		// Default: just listing completed workspaces
		fmt.Printf("\nNote: Workspace directories are preserved. Use --windows, --phantoms, --verify-opencode, --investigations, or --stale to clean up resources.\n")
	}

	return nil
}

// cleanOrphanedDiskSessions finds and deletes OpenCode disk sessions that aren't tracked via workspace files.
// Returns the number of sessions deleted and any error encountered.
func cleanOrphanedDiskSessions(serverURL string, dryRun bool) (int, error) {
	// Get current project directory
	projectDir, err := os.Getwd()
	if err != nil {
		return 0, fmt.Errorf("failed to get current directory: %w", err)
	}

	fmt.Printf("\nVerifying OpenCode disk sessions for %s...\n", projectDir)

	client := opencode.NewClient(serverURL)

	// Fetch all disk sessions for this directory
	diskSessions, err := client.ListDiskSessions(projectDir)
	if err != nil {
		return 0, fmt.Errorf("failed to list disk sessions: %w", err)
	}

	fmt.Printf("  Found %d disk sessions\n", len(diskSessions))

	// Build a set of session IDs that are tracked via workspace files
	trackedSessionIDs := make(map[string]bool)
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	if entries, err := os.ReadDir(workspaceDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				sessionID := spawn.ReadSessionID(filepath.Join(workspaceDir, entry.Name()))
				if sessionID != "" {
					trackedSessionIDs[sessionID] = true
				}
			}
		}
	}

	fmt.Printf("  Workspaces track %d session IDs\n", len(trackedSessionIDs))

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
	for _, session := range orphanedSessions {
		title := session.Title
		if title == "" {
			title = "(untitled)"
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

	return deleted, nil
}

// cleanPhantomWindows finds and closes tmux windows that are phantoms
// (have a beads ID in the window name but no active OpenCode session).
// Returns the number of windows closed and any error encountered.
func cleanPhantomWindows(serverURL string, dryRun bool) (int, error) {
	client := opencode.NewClient(serverURL)
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

	workersSessions, _ := tmux.ListWorkersSessions()
	for _, sessionName := range workersSessions {
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

		// Move file to archived
		if err := os.Rename(path, destPath); err != nil {
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
// Returns the number of workspaces archived and any error encountered.
func archiveStaleWorkspaces(projectDir string, staleDays int, dryRun bool) (int, error) {
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

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip the archived directory itself
		if entry.Name() == "archived" {
			continue
		}

		dirPath := filepath.Join(workspaceDir, entry.Name())

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

		// Move workspace to archived
		if err := os.Rename(ws.path, destPath); err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: failed to archive %s: %v\n", ws.name, err)
			continue
		}

		fmt.Printf("    Archived: %s (%.0f days old, %s)\n", ws.name, age, ws.reason)
		archived++
	}

	return archived, nil
}

// NOTE: extractBeadsIDFromWorkspace is defined in review.go
