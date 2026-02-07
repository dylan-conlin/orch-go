// Package main provides post-completion actions for the complete command.
// Includes beads issue closing, archival, transcript export, and cache invalidation.
// Extracted from complete_cmd.go for maintainability.
package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// invalidateServeCache sends a request to orch serve to invalidate its caches.
// This ensures the dashboard shows updated agent status immediately after completion.
// Silently fails if orch serve is not running (cache will refresh via TTL).
func invalidateServeCache() {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Post(
		fmt.Sprintf("http://localhost:%d/api/cache/invalidate", DefaultServePort),
		"application/json",
		nil,
	)
	if err != nil {
		// Silent failure - orch serve might not be running
		return
	}
	defer resp.Body.Close()
	// We don't care about the response - if it worked, great; if not, TTL will eventually refresh
}

// addApprovalComment adds an approval comment to a beads issue.
// This is used by --approve flag to mark visual changes as human-reviewed.
func addApprovalComment(beadsID, comment string) error {
	err := beads.Do("", func(client *beads.Client) error {
		// Use "orchestrator" as the author for approval comments
		return client.AddComment(beadsID, "orchestrator", comment)
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return nil
	}

	// Fallback to CLI
	return beads.FallbackAddComment(beadsID, comment)
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

// collectCompletionTelemetry collects duration and token usage for telemetry.
// Returns (durationSeconds, tokensInput, tokensOutput, outcome).
// Returns zeros if telemetry collection fails (non-blocking).
func collectCompletionTelemetry(workspacePath string, forced bool, verificationPassed bool) (int, int, int, string) {
	return collectCompletionTelemetryWithClient(opencode.NewClient("http://127.0.0.1:4096"), workspacePath, forced, verificationPassed)
}

func collectCompletionTelemetryWithClient(client opencode.ClientInterface, workspacePath string, forced bool, verificationPassed bool) (int, int, int, string) {
	var durationSeconds int
	var tokensInput int
	var tokensOutput int
	var outcome string

	// Determine outcome
	if forced {
		outcome = "forced"
	} else if verificationPassed {
		outcome = "success"
	} else {
		outcome = "failed"
	}

	// Read spawn time from workspace
	spawnTimeFile := filepath.Join(workspacePath, ".spawn_time")
	if spawnTimeBytes, err := os.ReadFile(spawnTimeFile); err == nil {
		spawnTimeStr := strings.TrimSpace(string(spawnTimeBytes))
		if spawnTime, err := time.Parse(time.RFC3339, spawnTimeStr); err == nil {
			durationSeconds = int(time.Since(spawnTime).Seconds())
		}
	}

	// Read session ID from workspace
	sessionIDFile := filepath.Join(workspacePath, ".session_id")
	if sessionIDBytes, err := os.ReadFile(sessionIDFile); err == nil {
		sessionID := strings.TrimSpace(string(sessionIDBytes))
		if sessionID != "" {
			// Get token usage from OpenCode API
			if tokenStats, err := client.GetSessionTokens(sessionID); err == nil && tokenStats != nil {
				tokensInput = tokenStats.InputTokens
				tokensOutput = tokenStats.OutputTokens
			}
		}
	}

	return durationSeconds, tokensInput, tokensOutput, outcome
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
