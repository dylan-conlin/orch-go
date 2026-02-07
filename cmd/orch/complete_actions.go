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
