// Package main provides helper functions used by the completion pipeline phases.
// These are extracted from complete_cmd.go to keep the command file focused on
// CLI definition and the thin pipeline orchestrator.
//
// Checklist and changelog UI rendering are in complete_checklist.go.
// Post-lifecycle helpers (cache invalidation, auto-rebuild, telemetry, accretion)
// are in complete_postlifecycle.go.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// archiveWorkspace moves a completed workspace to the archived directory.
// Returns the new archived path on success, or an error if archival fails.
// The function handles name collisions by adding a timestamp suffix.
//
// Note: This function is superseded by LifecycleManager.Archive() but retained
// for backward compatibility with tests.
func archiveWorkspace(workspacePath, projectDir string) (string, error) {
	if workspacePath == "" {
		return "", fmt.Errorf("workspace path is empty")
	}

	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		return "", fmt.Errorf("workspace does not exist: %s", workspacePath)
	}

	workspaceName := filepath.Base(workspacePath)
	archivedDir := filepath.Join(projectDir, ".orch", "workspace", "archived")

	if err := os.MkdirAll(archivedDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create archived directory: %w", err)
	}

	destPath := filepath.Join(archivedDir, workspaceName)

	if _, err := os.Stat(destPath); err == nil {
		suffix := time.Now().Format("150405")
		destPath = filepath.Join(archivedDir, workspaceName+"-"+suffix)
		fmt.Printf("Note: Archive destination exists, using: %s-%s\n", workspaceName, suffix)
	}

	if err := os.Rename(workspacePath, destPath); err != nil {
		return "", fmt.Errorf("failed to archive workspace: %w", err)
	}

	return destPath, nil
}
