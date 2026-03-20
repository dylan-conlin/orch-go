package backends

import (
	"context"
	"fmt"
	"os"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// HeadlessBackend spawns the agent via the OpenCode HTTP API without a TUI.
// This is useful for automation and daemon-driven spawns.
// Uses HTTP API with x-opencode-directory header to properly set the session's
// working directory for cross-project spawns.
// Includes retry logic for transient network failures.
type HeadlessBackend struct{}

// Name returns the backend name.
func (b *HeadlessBackend) Name() string {
	return "headless"
}

// headlessSpawnResult contains the result of starting a headless session.
type headlessSpawnResult struct {
	SessionID string
}

// Spawn creates a new agent session in headless mode.
func (b *HeadlessBackend) Spawn(ctx context.Context, req *SpawnRequest) (*Result, error) {
	client := opencode.NewClient(req.ServerURL)

	// Format title with beads ID so orch status can match sessions
	sessionTitle := FormatSessionTitle(req.Config.WorkspaceName, req.BeadsID)

	// Use retry logic for transient failures (network issues, server temporarily unavailable)
	retryCfg := spawn.DefaultRetryConfig()
	result, retryResult := spawn.Retry(retryCfg, func() (*headlessSpawnResult, error) {
		return startHeadlessSession(client, req.ServerURL, sessionTitle, req.MinimalPrompt, req.Config)
	})

	if retryResult.LastErr != nil {
		// Wrap the error with user-friendly message and recovery guidance
		spawnErr := spawn.WrapSpawnError(retryResult.LastErr, "Headless spawn failed")
		if retryResult.Attempts > 1 {
			fmt.Fprintf(os.Stderr, "Spawn failed after %d attempts\n", retryResult.Attempts)
		}
		// Print formatted error with recovery guidance
		fmt.Fprintf(os.Stderr, "\n%s\n", spawn.FormatSpawnError(spawnErr))
		return nil, spawnErr
	}

	if retryResult.Attempts > 1 {
		fmt.Printf("Spawn succeeded after %d attempts\n", retryResult.Attempts)
	}

	sessionID := result.SessionID

	// Write session ID to workspace file for later lookups
	if err := spawn.WriteSessionID(req.Config.WorkspacePath(), sessionID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write session ID: %v\n", err)
	}

	// Orchestrator sessions use workspace artifacts for tracking
	// Build extra event data for retries
	var extraData map[string]interface{}
	if retryResult.Attempts > 1 {
		extraData = map[string]interface{}{
			"retry_attempts": retryResult.Attempts,
		}
	}

	// Log the session creation
	if err := LogSpawnEvent(sessionID, req, "headless", extraData); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	return &Result{
		SessionID:     sessionID,
		SpawnMode:     "headless",
		RetryAttempts: retryResult.Attempts,
	}, nil
}

// startHeadlessSession creates an OpenCode session via HTTP API and sends the initial prompt.
// Uses HTTP API instead of CLI subprocess to properly set the session's working directory
// via x-opencode-directory header. This fixes cross-project spawns where --workdir differs
// from the orchestrator's CWD.
// Model selection is handled per-message by SendMessageInDirectory (providerID/modelID format).
func startHeadlessSession(client *opencode.Client, serverURL, sessionTitle, minimalPrompt string, cfg *spawn.Config) (*headlessSpawnResult, error) {
	// Step 1: Create session via HTTP API with correct directory
	// CreateSession passes x-opencode-directory header so the server uses the target project dir
	metadata := map[string]string{
		"beads_id":       cfg.BeadsID,
		"workspace_path": cfg.WorkspacePath(),
		"tier":           cfg.Tier,
		"spawn_mode":     "headless",
	}

	// Calculate TTL based on session type
	// Worker sessions: 4 hours (14400 seconds)
	// Orchestrator sessions: 0 (no expiration)
	// Explore orchestrators use worker TTL (they run in worker lifecycle)
	timeTTL := 4 * 60 * 60 // 4 hours in seconds
	if cfg.IsOrchestrator && !cfg.Explore {
		timeTTL = 0 // Orchestrator sessions persist until manually cleaned
	}

	session, err := client.CreateSession(sessionTitle, cfg.ProjectDir, cfg.Model, metadata, timeTTL)
	if err != nil {
		return nil, spawn.WrapSpawnError(err, "Failed to create session via API")
	}

	// Step 2: Send the initial prompt with model selection and directory context
	// The directory header ensures the server resolves the correct project context
	if err := client.SendMessageInDirectory(session.ID, minimalPrompt, cfg.Model, cfg.ProjectDir); err != nil {
		return nil, spawn.WrapSpawnError(err, "Failed to send prompt to session")
	}

	return &headlessSpawnResult{
		SessionID: session.ID,
	}, nil
}
