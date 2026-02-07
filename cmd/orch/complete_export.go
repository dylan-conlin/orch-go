// Package main provides completion export and archival helpers.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/activity"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// exportActivity exports agent activity to the workspace.
func exportActivity(target *CompletionTarget) {
	sessionFile := filepath.Join(target.WorkspacePath, ".session_id")
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return
	}
	sessionID := strings.TrimSpace(string(data))
	if sessionID == "" {
		return
	}

	activityPath, err := activity.ExportToWorkspace(sessionID, target.WorkspacePath, serverURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to export activity: %v\n", err)
	} else if activityPath != "" {
		fmt.Printf("Exported activity: %s\n", filepath.Base(activityPath))
	}
}

// archiveWorkspace moves a completed workspace to the archived directory.
// Returns the new archived path on success, or an error if archival fails.
// The function handles name collisions by adding a timestamp suffix.
func archiveWorkspace(workspacePath, projectDir string) (string, error) {
	if workspacePath == "" {
		return "", fmt.Errorf("workspace path is empty")
	}

	// Verify workspace exists
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		return "", fmt.Errorf("workspace does not exist: %s", workspacePath)
	}

	// Determine workspace name and archived directory
	workspaceName := filepath.Base(workspacePath)
	archivedDir := filepath.Join(projectDir, ".orch", "workspace", "archived")

	// Create archived directory if needed
	if err := os.MkdirAll(archivedDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create archived directory: %w", err)
	}

	// Determine destination path
	destPath := filepath.Join(archivedDir, workspaceName)

	// Handle name collision (if archive already exists, add timestamp suffix)
	if _, err := os.Stat(destPath); err == nil {
		suffix := time.Now().Format("150405") // HHMMSS format
		destPath = filepath.Join(archivedDir, workspaceName+"-"+suffix)
		fmt.Printf("Note: Archive destination exists, using: %s-%s\n", workspaceName, suffix)
	}

	// Move workspace to archived
	if err := os.Rename(workspacePath, destPath); err != nil {
		return "", fmt.Errorf("failed to archive workspace: %w", err)
	}

	return destPath, nil
}

// exportOrchestratorTranscript exports the session transcript for orchestrator sessions.
// It checks for .orchestrator marker, sends /export to the tmux window, waits for the
// export file, and moves it to the workspace as TRANSCRIPT.md.
func exportOrchestratorTranscript(workspacePath, projectDir, beadsID string) error {
	// Check if this is an orchestrator session (has .orchestrator or .meta-orchestrator marker)
	orchestratorMarker := filepath.Join(workspacePath, ".orchestrator")
	metaOrchestratorMarker := filepath.Join(workspacePath, ".meta-orchestrator")

	isOrchestrator := false
	if _, err := os.Stat(orchestratorMarker); err == nil {
		isOrchestrator = true
	} else if _, err := os.Stat(metaOrchestratorMarker); err == nil {
		isOrchestrator = true
	}

	if !isOrchestrator {
		return nil // Not an orchestrator, nothing to do
	}

	// Find the tmux window for this agent
	window, _, err := tmux.FindWindowByBeadsIDAllSessions(beadsID)
	if err != nil || window == nil {
		return fmt.Errorf("could not find tmux window for orchestrator")
	}

	// Record existing session export files before sending /export
	existingExports := make(map[string]bool)
	pattern := filepath.Join(projectDir, "session-ses_*.md")
	matches, _ := filepath.Glob(pattern)
	for _, m := range matches {
		existingExports[m] = true
	}

	// Send /export command to the tmux window
	if err := tmux.SendKeys(window.Target, "/export"); err != nil {
		return fmt.Errorf("failed to send /export: %w", err)
	}
	if err := tmux.SendEnter(window.Target); err != nil {
		return fmt.Errorf("failed to send enter: %w", err)
	}

	fmt.Println("Exporting orchestrator transcript...")

	// Wait for new export file to appear (poll for up to 10 seconds)
	var newExportPath string
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			if !existingExports[m] {
				newExportPath = m
				break
			}
		}
		if newExportPath != "" {
			break
		}
	}

	if newExportPath == "" {
		return fmt.Errorf("timeout waiting for export file")
	}

	// Move export to workspace as TRANSCRIPT.md
	destPath := filepath.Join(workspacePath, "TRANSCRIPT.md")
	if err := os.Rename(newExportPath, destPath); err != nil {
		// If rename fails (cross-device), try copy+delete
		input, err := os.ReadFile(newExportPath)
		if err != nil {
			return fmt.Errorf("failed to read export: %w", err)
		}
		if err := os.WriteFile(destPath, input, 0644); err != nil {
			return fmt.Errorf("failed to write transcript: %w", err)
		}
		os.Remove(newExportPath)
	}

	fmt.Printf("Saved transcript: %s\n", destPath)
	return nil
}
