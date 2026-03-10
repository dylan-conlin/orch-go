package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/spf13/cobra"
)

// ============================================================================
// Session Migrate Command - Migrate legacy handoffs to window-scoped structure
// ============================================================================

var sessionMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate legacy session handoffs to window-scoped structure",
	Long: `Migrate legacy session handoffs to window-scoped structure.

Before window-scoping was added, session handoffs were stored in:
  .orch/session/{timestamp}/SESSION_HANDOFF.md

After window-scoping, they're stored in:
  .orch/session/{window-name}/{timestamp}/SESSION_HANDOFF.md

This command migrates old handoffs to the new structure.

Examples:
  orch session migrate              # Migrate to current window
  orch session migrate --all        # Show migration status for all windows`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionMigrate()
	},
}

func runSessionMigrate() error {
	// Get current directory to find .orch/session
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find project root by walking up to .orch directory
	projectDir := currentDir
	for {
		sessionDir := filepath.Join(projectDir, ".orch", "session")
		if _, err := os.Stat(sessionDir); err == nil {
			break
		}
		parent := filepath.Dir(projectDir)
		if parent == projectDir {
			return fmt.Errorf("no .orch/session directory found (not in an orch-managed project)")
		}
		projectDir = parent
	}

	sessionBaseDir := filepath.Join(projectDir, ".orch", "session")

	// Get current window name for migration target
	windowName, err := tmux.GetCurrentWindowName()
	if err != nil {
		return fmt.Errorf("failed to get window name: %w", err)
	}

	// Check for legacy handoffs (non-window-scoped directories)
	entries, err := os.ReadDir(sessionBaseDir)
	if err != nil {
		return fmt.Errorf("failed to read session directory: %w", err)
	}

	var legacyDirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Legacy directories are timestamp format: YYYY-MM-DD-HHMM
		// Window-scoped directories are names (e.g., "default", "pw", "og-feat-...")
		name := entry.Name()
		// Check if it looks like a timestamp (starts with digit)
		if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
			legacyDirs = append(legacyDirs, name)
		}
	}

	if len(legacyDirs) == 0 {
		fmt.Println("✅ No legacy handoffs found - already using window-scoped structure")
		return nil
	}

	// Show what will be migrated
	fmt.Printf("Found %d legacy handoff(s) to migrate:\n\n", len(legacyDirs))
	for _, dir := range legacyDirs {
		handoffPath := filepath.Join(sessionBaseDir, dir, "SESSION_HANDOFF.md")
		if _, err := os.Stat(handoffPath); err == nil {
			fmt.Printf("  • %s → .orch/session/%s/%s\n", dir, windowName, dir)
		}
	}

	fmt.Printf("\nMigrate to window-scoped structure for window %q? (y/N): ", windowName)
	var response string
	fmt.Scanln(&response)

	if response != "y" && response != "Y" {
		fmt.Println("Migration cancelled")
		return nil
	}

	// Perform migration
	windowScopedDir := filepath.Join(sessionBaseDir, windowName)
	if err := os.MkdirAll(windowScopedDir, 0755); err != nil {
		return fmt.Errorf("failed to create window-scoped directory: %w", err)
	}

	migratedCount := 0
	for _, dir := range legacyDirs {
		sourcePath := filepath.Join(sessionBaseDir, dir)
		destPath := filepath.Join(windowScopedDir, dir)

		// Check if handoff exists
		handoffPath := filepath.Join(sourcePath, "SESSION_HANDOFF.md")
		if _, err := os.Stat(handoffPath); err != nil {
			// Skip directories without handoffs
			continue
		}

		// Move the directory
		if err := os.Rename(sourcePath, destPath); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to migrate %s: %v\n", dir, err)
			continue
		}
		migratedCount++
	}

	// Update latest symlink to point to most recent migrated handoff
	if migratedCount > 0 {
		// Find most recent timestamp directory
		var latestTimestamp string
		entries, _ := os.ReadDir(windowScopedDir)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if name > latestTimestamp && len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
				latestTimestamp = name
			}
		}

		if latestTimestamp != "" {
			latestSymlink := filepath.Join(windowScopedDir, "latest")
			_ = os.Remove(latestSymlink) // Remove old symlink if exists
			if err := os.Symlink(latestTimestamp, latestSymlink); err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Failed to update latest symlink: %v\n", err)
			}
		}
	}

	// Remove legacy latest symlink at root level
	legacyLatest := filepath.Join(sessionBaseDir, "latest")
	if _, err := os.Lstat(legacyLatest); err == nil {
		if err := os.Remove(legacyLatest); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to remove legacy latest symlink: %v\n", err)
		}
	}

	fmt.Printf("\n✅ Successfully migrated %d handoff(s) to window-scoped structure\n", migratedCount)
	fmt.Printf("   Window: %s\n", windowName)
	fmt.Printf("   Location: .orch/session/%s/\n", windowName)

	return nil
}
