// Package main provides workspace cleanup functions for the clean command.
// Extracted from clean_cmd.go for cohesion (workspace archival, investigation cleanup, expired archives).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

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
		openIssues, err := verify.ListOpenIssues("")
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

// cleanExpiredArchives deletes archived workspaces older than ttlDays.
// Uses .spawn_time (nanosecond epoch) or AGENT_MANIFEST.json SpawnTime,
// falling back to directory modification time when neither exists.
// Returns the number of workspaces deleted (or would-be-deleted in dry-run mode).
func cleanExpiredArchives(projectDir string, ttlDays int, dryRun bool) (int, error) {
	archivedDir := filepath.Join(projectDir, ".orch", "workspace", "archived")

	if _, err := os.Stat(archivedDir); os.IsNotExist(err) {
		return 0, nil
	}

	fmt.Printf("\nScanning for expired archived workspaces (older than %d days)...\n", ttlDays)

	cutoff := time.Now().AddDate(0, 0, -ttlDays)

	entries, err := os.ReadDir(archivedDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read archived directory: %w", err)
	}

	deleted := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirPath := filepath.Join(archivedDir, entry.Name())

		// Determine workspace age: manifest → .spawn_time → fallback to modtime
		var wsTime time.Time

		manifest := spawn.ReadAgentManifestWithFallback(dirPath)
		wsTime = manifest.ParseSpawnTime()

		if wsTime.IsZero() {
			wsTime, _ = fallbackWorkspaceSpawnTime(dirPath)
		}

		// Last resort: directory modtime
		if wsTime.IsZero() {
			if info, err := os.Stat(dirPath); err == nil {
				wsTime = info.ModTime()
			}
		}

		if wsTime.IsZero() {
			continue // Cannot determine age, skip
		}

		if wsTime.After(cutoff) {
			continue // Not expired yet
		}

		age := time.Since(wsTime).Hours() / 24

		if dryRun {
			fmt.Printf("    [DRY-RUN] Would delete: %s (%.0f days old)\n", entry.Name(), age)
			deleted++
			continue
		}

		if err := os.RemoveAll(dirPath); err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: failed to delete %s: %v\n", entry.Name(), err)
			continue
		}

		fmt.Printf("    Deleted: %s (%.0f days old)\n", entry.Name(), age)
		deleted++
	}

	if deleted == 0 {
		fmt.Println("  No expired archived workspaces found")
	}

	return deleted, nil
}
