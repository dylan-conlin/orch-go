// Package cleanup provides utilities for cleaning up workspace directories.
package cleanup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ArchiveStaleWorkspacesOptions configures the workspace archival behavior.
type ArchiveStaleWorkspacesOptions struct {
	// ProjectDir is the root directory of the project (contains .orch/workspace/)
	ProjectDir string
	// StaleDays is the number of days after which a workspace is considered stale
	StaleDays int
	// DryRun if true, only reports what would be archived without actually archiving
	DryRun bool
	// PreserveOrchestrator if true, skips orchestrator workspaces
	PreserveOrchestrator bool
	// Quiet if true, suppresses progress output (for daemon use)
	Quiet bool
}

// ArchiveStaleWorkspaces moves old completed workspaces to .orch/workspace/archived/.
// A workspace is considered "stale" if:
// 1. It has a .spawn_time older than staleDays
// 2. It is completed (SYNTHESIS.md exists OR light tier OR has .beads_id)
// If preserveOrchestrator is true, orchestrator/meta-orchestrator workspaces are skipped.
// Returns the number of workspaces archived and any error encountered.
func ArchiveStaleWorkspaces(opts ArchiveStaleWorkspacesOptions) (int, error) {
	workspaceDir := filepath.Join(opts.ProjectDir, ".orch", "workspace")
	archivedDir := filepath.Join(workspaceDir, "archived")

	// Check if workspace directory exists
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		if !opts.Quiet {
			fmt.Println("\nNo .orch/workspace directory found")
		}
		return 0, nil
	}

	if !opts.Quiet {
		fmt.Printf("\nScanning for stale workspaces (older than %d days)...\n", opts.StaleDays)
	}

	// Calculate the cutoff time
	cutoff := time.Now().AddDate(0, 0, -opts.StaleDays)

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
		if opts.PreserveOrchestrator && isOrchestratorWorkspace(dirPath) {
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

	if !opts.Quiet && skippedOrch > 0 {
		fmt.Printf("  Skipped %d orchestrator workspaces (--preserve-orchestrator)\n", skippedOrch)
	}

	if len(staleWorkspaces) == 0 {
		if !opts.Quiet {
			fmt.Println("  No stale completed workspaces found")
		}
		return 0, nil
	}

	if !opts.Quiet {
		fmt.Printf("  Found %d stale workspaces:\n", len(staleWorkspaces))
	}

	// Create archived directory if needed
	if !opts.DryRun {
		if err := os.MkdirAll(archivedDir, 0755); err != nil {
			return 0, fmt.Errorf("failed to create archived directory: %w", err)
		}
	}

	// Archive stale workspaces
	archived := 0
	for _, ws := range staleWorkspaces {
		destPath := filepath.Join(archivedDir, ws.name)
		age := time.Since(ws.spawnTime).Hours() / 24

		if opts.DryRun {
			if !opts.Quiet {
				fmt.Printf("    [DRY-RUN] Would archive: %s (%.0f days old, %s)\n", ws.name, age, ws.reason)
			}
			archived++
			continue
		}

		// Check if destination already exists
		finalDestPath := destPath
		if _, err := os.Stat(destPath); err == nil {
			// Destination exists - add timestamp suffix to make it unique
			suffix := time.Now().Format("150405") // HHMMSS format
			finalDestPath = destPath + "-" + suffix
			if !opts.Quiet {
				fmt.Printf("    Note: Archive destination exists, using: %s-%s\n", ws.name, suffix)
			}
		}

		// Move workspace to archived
		if err := os.Rename(ws.path, finalDestPath); err != nil {
			if !opts.Quiet {
				fmt.Fprintf(os.Stderr, "    Warning: failed to archive %s: %v\n", ws.name, err)
			}
			continue
		}

		if !opts.Quiet {
			fmt.Printf("    Archived: %s (%.0f days old, %s)\n", ws.name, age, ws.reason)
		}
		archived++
	}

	return archived, nil
}

// isOrchestratorWorkspace checks if a workspace is an orchestrator workspace.
// Orchestrator workspaces are identified by:
// - Having .tier file with content "orchestrator"
// - OR having workspace name matching orchestrator patterns
func isOrchestratorWorkspace(workspacePath string) bool {
	// Check .tier file
	tierFile := filepath.Join(workspacePath, ".tier")
	if tierData, err := os.ReadFile(tierFile); err == nil {
		tier := strings.TrimSpace(string(tierData))
		if tier == "orchestrator" {
			return true
		}
	}

	// Check workspace name patterns
	workspaceName := filepath.Base(workspacePath)
	nameLower := strings.ToLower(workspaceName)
	if strings.Contains(nameLower, "orchestrator") ||
		strings.Contains(nameLower, "meta-orch") ||
		strings.HasPrefix(nameLower, "meta-") {
		return true
	}

	return false
}
