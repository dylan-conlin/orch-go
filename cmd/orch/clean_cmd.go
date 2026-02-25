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
	cleanDryRun               bool
	cleanWorkspaces           bool
	cleanSessions             bool
	cleanWorkspaceDays        int
	cleanSessionDays          int
	cleanPreserveOrchestrator bool
	cleanAll                  bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up stale workspaces and tmux windows",
	Long: `Clean up stale agent resources (workspaces, tmux windows).

NOTE: OpenCode session cleanup is now handled automatically via TTL (opencode-fork commit f3c3865).
The --sessions flag now only cleans stale tmux windows. OpenCode sessions are managed by the server.

By default, this command only REPORTS what could be cleaned - it does not delete
anything. Workspace directories are archived (moved to archived/), never deleted.

Cleanup actions:
  --workspaces        Archive old completed workspaces and empty investigation files
  --sessions          Clean stale tmux windows (OpenCode sessions are auto-cleaned by server)
  --all               Enable all cleanup actions (workspaces + sessions)

Age thresholds:
  --workspace-days N  Set age threshold for --workspaces (default: 7)

Protection options:
  --preserve-orchestrator  Skip orchestrator/meta-orchestrator workspaces and sessions

Examples:
  orch clean                    # List completed agents (no changes)
  orch clean --dry-run          # Preview mode (same as default)
  orch clean --all              # Comprehensive cleanup
  orch clean --all --dry-run    # Preview comprehensive cleanup
  orch clean --all --preserve-orchestrator  # Clean everything except orchestrator sessions
  orch clean --sessions         # Clean stale tmux windows
  orch clean --workspaces       # Archive old workspaces and empty investigations
  orch clean --workspaces --workspace-days 14  # Archive workspaces older than 14 days`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If --all is specified, enable all cleanup flags
		if cleanAll {
			cleanWorkspaces = true
			cleanSessions = true
		}
		return runClean(cleanDryRun, cleanWorkspaces, cleanSessions, cleanWorkspaceDays, cleanSessionDays, cleanPreserveOrchestrator)
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "Show what would be cleaned without making changes")
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Enable all cleanup actions (workspaces + sessions)")
	cleanCmd.Flags().BoolVar(&cleanWorkspaces, "workspaces", false, "Archive old completed workspaces and empty investigation files")
	cleanCmd.Flags().BoolVar(&cleanSessions, "sessions", false, "Clean stale tmux windows (OpenCode sessions are auto-cleaned by server)")
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

func runClean(dryRun bool, doWorkspaces bool, doSessions bool, workspaceDays int, sessionDays int, preserveOrchestrator bool) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Default mode (no flags): scan and report completed workspaces
	if !doWorkspaces && !doSessions {
		fmt.Println("Scanning workspaces for completed agents...")
		beadsChecker := NewDefaultBeadsStatusChecker()
		cleanableWorkspaces := findCleanableWorkspaces(projectDir, beadsChecker)

		fmt.Printf("\nFound %d completed workspaces\n", len(cleanableWorkspaces))

		if len(cleanableWorkspaces) == 0 {
			fmt.Println("No completed agents found")
			return nil
		}

		fmt.Printf("\nCompleted workspaces:\n")
		for _, ws := range cleanableWorkspaces {
			fmt.Printf("  %s (%s)\n", ws.Name, ws.Reason)
		}

		fmt.Printf("\nNote: Use --workspaces, --sessions, or --all to clean up resources.\n")
		return nil
	}

	// Track cleanup stats
	var workspacesArchived, investigationsArchived int
	var windowsClosed, untrackedSessionsDeleted, staleSessionsDeleted int

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
		windowsClosed, err = cleanStaleTmuxWindows(serverURL, dryRun, preserveOrchestrator)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean stale tmux windows: %v\n", err)
		}

		// Session cleanup (steps 2-3) removed - OpenCode handles this via TTL
		untrackedSessionsDeleted = 0
		staleSessionsDeleted = 0
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
			if untrackedSessionsDeleted > 0 {
				fmt.Printf(" Would delete %d untracked sessions.", untrackedSessionsDeleted)
			}
			if staleSessionsDeleted > 0 {
				fmt.Printf(" Would delete %d stale sessions.", staleSessionsDeleted)
			}
		}
		fmt.Println()
		return nil
	}

	// Log event if any cleanup actions were taken
	totalCleaned := workspacesArchived + investigationsArchived + windowsClosed + untrackedSessionsDeleted + staleSessionsDeleted
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
				"untracked_sessions_deleted": untrackedSessionsDeleted,
				"stale_sessions_deleted":     staleSessionsDeleted,
				"project":                    projectName,
				"clean_workspaces":           doWorkspaces,
				"clean_sessions":             doSessions,
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
		if untrackedSessionsDeleted > 0 {
			fmt.Printf("Deleted %d untracked sessions\n", untrackedSessionsDeleted)
		}
		if staleSessionsDeleted > 0 {
			fmt.Printf("Deleted %d stale sessions\n", staleSessionsDeleted)
		}
	}

	return nil
}

// cleanUntrackedDiskSessions finds and deletes OpenCode disk sessions that aren't tracked via workspace files.
// If preserveOrchestrator is true, sessions associated with orchestrator workspaces are skipped.
// Returns the number of sessions deleted and any error encountered.
// cleanUntrackedDiskSessions has been removed - OpenCode now handles session cleanup via TTL
// (see opencode-fork commit f3c3865)

// cleanStaleTmuxWindows finds and closes tmux windows with no active OpenCode session backing them.
// If preserveOrchestrator is true, windows in orchestrator/meta-orchestrator sessions are skipped.
// Returns the number of windows closed and any error encountered.
func cleanStaleTmuxWindows(serverURL string, dryRun bool, preserveOrchestrator bool) (int, error) {
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

	// Scan all workers sessions for stale windows
	var staleWindows []struct {
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

			// If beads ID is not in active sessions, the window is stale
			if !activeBeadsIDs[beadsID] {
				windowCopy := w
				staleWindows = append(staleWindows, struct {
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

	if len(staleWindows) == 0 {
		fmt.Println("  No stale tmux windows found")
		return 0, nil
	}

	fmt.Printf("  Found %d stale tmux windows:\n", len(staleWindows))

	// Close stale windows
	closed := 0
	for _, pw := range staleWindows {
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

// NOTE: extractBeadsIDFromWorkspace is defined in review.go
