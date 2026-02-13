package backends

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// HeadlessBackend spawns the agent using CLI subprocess without a TUI.
// This is useful for automation and daemon-driven spawns.
// Uses opencode CLI with --format json to properly support model selection
// (the HTTP API ignores the model parameter).
// Includes retry logic for transient network failures.
type HeadlessBackend struct{}

// Name returns the backend name.
func (b *HeadlessBackend) Name() string {
	return "headless"
}

// headlessSpawnResult contains the result of starting a headless session.
type headlessSpawnResult struct {
	SessionID string
	cmd       *exec.Cmd
	stdout    io.ReadCloser
}

// StartBackgroundCleanup starts a goroutine to drain stdout and wait for the process.
func (r *headlessSpawnResult) StartBackgroundCleanup() {
	if r.stdout == nil || r.cmd == nil {
		return
	}
	go func() {
		// Drain remaining stdout
		io.Copy(io.Discard, r.stdout)
		// Wait for process to complete (cleanup)
		r.cmd.Wait()
	}()
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

	// Start background cleanup goroutine
	result.StartBackgroundCleanup()

	// Register orchestrator session in registry (workers use beads instead)
	RegisterOrchestratorSession(req.Config, sessionID, req.Task)

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

// startHeadlessSession starts an opencode session and extracts the session ID.
// Returns the result with session ID and resources for cleanup.
// Note: Uses CLI mode instead of HTTP API because OpenCode's HTTP API ignores the model parameter.
// CLI mode correctly honors the --model flag.
// See: .kb/investigations/2025-12-23-inv-model-selection-issue-architect-agent.md
func startHeadlessSession(client *opencode.Client, serverURL, sessionTitle, minimalPrompt string, cfg *spawn.Config) (*headlessSpawnResult, error) {
	cmd := client.BuildSpawnCommand(minimalPrompt, sessionTitle, cfg.Model)
	cmd.Dir = cfg.ProjectDir
	// Set ORCH_WORKER=1 so agents know they are orch-managed workers
	cmd.Env = append(os.Environ(), "ORCH_WORKER=1")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		spawnErr := spawn.WrapSpawnError(err, "Failed to get stdout pipe")
		return nil, spawnErr
	}

	// Discard stderr in headless mode (no TUI to display it)
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		spawnErr := spawn.WrapSpawnError(err, "Failed to start opencode process")
		return nil, spawnErr
	}

	// Process stdout to extract session ID, then let the process run in background
	// We need to read at least until we get the session ID
	sessionID, err := opencode.ExtractSessionIDFromReader(stdout)
	if err != nil {
		// Try to kill the process if we couldn't get session ID
		cmd.Process.Kill()
		spawnErr := spawn.WrapSpawnError(err, "Failed to extract session ID")
		return nil, spawnErr
	}

	return &headlessSpawnResult{
		SessionID: sessionID,
		cmd:       cmd,
		stdout:    stdout,
	}, nil
}
