// Package main provides workspace and investigation cleanup for the clean command.
// Extracted from clean_cmd.go for per-concern file organization.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

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
		return false
	}
	return issue.Status == "closed"
}

// DefaultCompletionIndicatorChecker checks for completion indicators (SYNTHESIS.md, Phase: Complete).
type DefaultCompletionIndicatorChecker struct{}

// NewDefaultCompletionIndicatorChecker creates a new completion indicator checker.
func NewDefaultCompletionIndicatorChecker() *DefaultCompletionIndicatorChecker {
	return &DefaultCompletionIndicatorChecker{}
}

// SynthesisExists checks if SYNTHESIS.md exists in the agent's workspace.
func (c *DefaultCompletionIndicatorChecker) SynthesisExists(workspacePath string) bool {
	exists, err := verify.VerifySynthesis(workspacePath)
	if err != nil {
		return false
	}
	return exists
}

// IsPhaseComplete checks if beads shows Phase: Complete for the agent.
func (c *DefaultCompletionIndicatorChecker) IsPhaseComplete(beadsID string) bool {
	complete, err := verify.IsPhaseComplete(beadsID)
	if err != nil {
		return false
	}
	return complete
}

// CleanableWorkspace represents a workspace that can be cleaned.
type CleanableWorkspace struct {
	Name       string
	Path       string
	BeadsID    string
	IsComplete bool
	Reason     string
}

// findCleanableWorkspaces scans .orch/workspace/ for completed/abandoned workspaces.
func findCleanableWorkspaces(projectDir string, beadsChecker *DefaultBeadsStatusChecker) []CleanableWorkspace {
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return nil
	}

	var cleanable []CleanableWorkspace
	var needsBeadsCheck []CleanableWorkspace

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "archived" {
			continue
		}

		dirName := entry.Name()
		dirPath := filepath.Join(workspaceDir, dirName)

		beadsID := ""
		spawnContextPath := filepath.Join(dirPath, "SPAWN_CONTEXT.md")
		if content, err := os.ReadFile(spawnContextPath); err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.Contains(line, "beads issue:") || strings.Contains(line, "BEADS ISSUE:") {
					parts := strings.Fields(line)
					for _, part := range parts {
						if strings.Contains(part, "-") && !strings.HasPrefix(part, "beads") && !strings.HasPrefix(part, "BEADS") {
							beadsID = strings.Trim(part, "*`[]")
							break
						}
					}
				}
			}
		}

		workspace := CleanableWorkspace{Name: dirName, Path: dirPath, BeadsID: beadsID}

		synthesisPath := filepath.Join(dirPath, "SYNTHESIS.md")
		if info, err := os.Stat(synthesisPath); err == nil && info.Size() > 0 {
			workspace.IsComplete = true
			workspace.Reason = "SYNTHESIS.md exists"
			cleanable = append(cleanable, workspace)
			continue
		}

		if beadsID != "" {
			needsBeadsCheck = append(needsBeadsCheck, workspace)
		}
	}

	if len(needsBeadsCheck) > 0 {
		openIssues, err := verify.ListOpenIssues()
		if err != nil {
			for _, ws := range needsBeadsCheck {
				if beadsChecker.IsIssueClosed(ws.BeadsID) {
					ws.IsComplete = true
					ws.Reason = "beads issue closed"
					cleanable = append(cleanable, ws)
				}
			}
		} else {
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
var emptyInvestigationPlaceholders = []string{
	"[Brief, descriptive title]",
	"[Clear, specific question",
	"[Concrete observations, data, examples]",
	"[File paths with line numbers",
	"[Explanation of the insight",
}

// isEmptyInvestigation checks if an investigation file still has template placeholders.
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
	return placeholderCount >= 2
}

// archiveEmptyInvestigations moves empty investigation files to .kb/investigations/archived/.
func archiveEmptyInvestigations(projectDir string, dryRun bool) (int, error) {
	investigationsDir := filepath.Join(projectDir, ".kb", "investigations")
	archivedDir := filepath.Join(investigationsDir, "archived")

	if _, err := os.Stat(investigationsDir); os.IsNotExist(err) {
		fmt.Println("\nNo .kb/investigations directory found")
		return 0, nil
	}

	fmt.Println("\nScanning for empty investigation files...")

	var emptyFiles []string
	err := filepath.Walk(investigationsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") || strings.Contains(path, "/archived/") {
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

	if !dryRun {
		if err := os.MkdirAll(archivedDir, 0755); err != nil {
			return 0, fmt.Errorf("failed to create archived directory: %w", err)
		}
	}

	archived := 0
	for _, path := range emptyFiles {
		filename := filepath.Base(path)
		relPath, _ := filepath.Rel(investigationsDir, path)
		destDir := filepath.Join(archivedDir, filepath.Dir(relPath))
		destPath := filepath.Join(destDir, filename)

		if dryRun {
			fmt.Printf("    [DRY-RUN] Would archive: %s\n", relPath)
			archived++
			continue
		}

		if err := os.MkdirAll(destDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: failed to create directory %s: %v\n", destDir, err)
			continue
		}

		finalDestPath := destPath
		if _, err := os.Stat(destPath); err == nil {
			suffix := time.Now().Format("150405")
			baseName := strings.TrimSuffix(filename, ".md")
			finalDestPath = filepath.Join(destDir, baseName+"-"+suffix+".md")
			fmt.Printf("    Note: Archive destination exists, using: %s-%s.md\n", baseName, suffix)
		}

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
func archiveStaleWorkspaces(projectDir string, staleDays int, dryRun bool, preserveOrchestrator bool) (int, error) {
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	archivedDir := filepath.Join(workspaceDir, "archived")

	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		fmt.Println("\nNo .orch/workspace directory found")
		return 0, nil
	}

	fmt.Printf("\nScanning for stale workspaces (older than %d days)...\n", staleDays)
	cutoff := time.Now().AddDate(0, 0, -staleDays)

	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read workspace directory: %w", err)
	}

	var staleWorkspaces []struct {
		name      string
		path      string
		spawnTime time.Time
		reason    string
	}

	skippedOrch := 0
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "archived" {
			continue
		}

		dirPath := filepath.Join(workspaceDir, entry.Name())

		if preserveOrchestrator && isOrchestratorWorkspace(dirPath) {
			skippedOrch++
			continue
		}

		spawnTimeData, err := os.ReadFile(filepath.Join(dirPath, ".spawn_time"))
		if err != nil {
			continue
		}

		var spawnTimeNs int64
		if _, err := fmt.Sscanf(string(spawnTimeData), "%d", &spawnTimeNs); err != nil {
			continue
		}
		spawnTime := time.Unix(0, spawnTimeNs)

		if spawnTime.After(cutoff) {
			continue
		}

		reason := ""
		synthesisPath := filepath.Join(dirPath, "SYNTHESIS.md")
		if info, err := os.Stat(synthesisPath); err == nil && info.Size() > 0 {
			reason = "SYNTHESIS.md exists"
		}
		if reason == "" {
			if tierData, err := os.ReadFile(filepath.Join(dirPath, ".tier")); err == nil {
				if strings.TrimSpace(string(tierData)) == "light" {
					reason = "light tier (no SYNTHESIS.md required)"
				}
			}
		}
		if reason == "" {
			if _, err := os.Stat(filepath.Join(dirPath, ".beads_id")); err == nil {
				reason = "tracked spawn (has .beads_id)"
			}
		}
		if reason == "" {
			continue
		}

		staleWorkspaces = append(staleWorkspaces, struct {
			name      string
			path      string
			spawnTime time.Time
			reason    string
		}{name: entry.Name(), path: dirPath, spawnTime: spawnTime, reason: reason})
	}

	if skippedOrch > 0 {
		fmt.Printf("  Skipped %d orchestrator workspaces (--preserve-orchestrator)\n", skippedOrch)
	}
	if len(staleWorkspaces) == 0 {
		fmt.Println("  No stale completed workspaces found")
		return 0, nil
	}

	fmt.Printf("  Found %d stale workspaces:\n", len(staleWorkspaces))

	if !dryRun {
		if err := os.MkdirAll(archivedDir, 0755); err != nil {
			return 0, fmt.Errorf("failed to create archived directory: %w", err)
		}
	}

	archived := 0
	for _, ws := range staleWorkspaces {
		destPath := filepath.Join(archivedDir, ws.name)
		age := time.Since(ws.spawnTime).Hours() / 24

		if dryRun {
			fmt.Printf("    [DRY-RUN] Would archive: %s (%.0f days old, %s)\n", ws.name, age, ws.reason)
			archived++
			continue
		}

		finalDestPath := destPath
		if _, err := os.Stat(destPath); err == nil {
			suffix := time.Now().Format("150405")
			finalDestPath = destPath + "-" + suffix
			fmt.Printf("    Note: Archive destination exists, using: %s-%s\n", ws.name, suffix)
		}

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
func archiveUntrackedWorkspaces(projectDir string, untrackedDays int, dryRun bool, preserveOrchestrator bool) (int, error) {
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	archivedDir := filepath.Join(workspaceDir, "archived")

	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		fmt.Println("\nNo .orch/workspace directory found")
		return 0, nil
	}

	fmt.Printf("\nScanning for untracked workspaces (older than %d days)...\n", untrackedDays)
	cutoff := time.Now().AddDate(0, 0, -untrackedDays)

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
		if !entry.IsDir() || entry.Name() == "archived" {
			continue
		}

		dirPath := filepath.Join(workspaceDir, entry.Name())

		if preserveOrchestrator && isOrchestratorWorkspace(dirPath) {
			skippedOrch++
			continue
		}

		beadsID := extractBeadsIDFromWorkspace(dirPath)
		if beadsID != "" && !isUntrackedBeadsID(beadsID) {
			continue
		}

		spawnTimeData, err := os.ReadFile(filepath.Join(dirPath, ".spawn_time"))
		if err != nil {
			continue
		}

		var spawnTimeNs int64
		if _, err := fmt.Sscanf(string(spawnTimeData), "%d", &spawnTimeNs); err != nil {
			continue
		}
		spawnTime := time.Unix(0, spawnTimeNs)

		if spawnTime.After(cutoff) {
			continue
		}

		untrackedWorkspaces = append(untrackedWorkspaces, struct {
			name      string
			path      string
			spawnTime time.Time
			beadsID   string
		}{name: entry.Name(), path: dirPath, spawnTime: spawnTime, beadsID: beadsID})
	}

	if skippedOrch > 0 {
		fmt.Printf("  Skipped %d orchestrator workspaces (--preserve-orchestrator)\n", skippedOrch)
	}
	if len(untrackedWorkspaces) == 0 {
		fmt.Println("  No untracked workspaces found")
		return 0, nil
	}

	fmt.Printf("  Found %d untracked workspaces:\n", len(untrackedWorkspaces))

	if !dryRun {
		if err := os.MkdirAll(archivedDir, 0755); err != nil {
			return 0, fmt.Errorf("failed to create archived directory: %w", err)
		}
	}

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

		finalDestPath := destPath
		if _, err := os.Stat(destPath); err == nil {
			suffix := time.Now().Format("150405")
			finalDestPath = destPath + "-" + suffix
			fmt.Printf("    Note: Archive destination exists, using: %s-%s\n", ws.name, suffix)
		}

		if err := os.Rename(ws.path, finalDestPath); err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: failed to archive %s: %v\n", ws.name, err)
			continue
		}

		fmt.Printf("    Archived: %s (%.0f days old, %s)\n", ws.name, age, beadsDisplay)
		archived++
	}

	return archived, nil
}
