package backends

import (
	"context"
	"fmt"
	"os"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// InlineBackend spawns the agent inline (blocking) - original behavior.
// This runs the opencode TUI in the current terminal and blocks until completion.
type InlineBackend struct{}

// Name returns the backend name.
func (b *InlineBackend) Name() string {
	return "inline"
}

// Spawn creates a new agent session inline (blocking with TUI).
func (b *InlineBackend) Spawn(ctx context.Context, req *SpawnRequest) (*Result, error) {
	client := opencode.NewClient(req.ServerURL)

	// Format title with beads ID so orch status can match sessions
	sessionTitle := FormatSessionTitle(req.Config.WorkspaceName, req.BeadsID)

	cmd := client.BuildSpawnCommand(req.MinimalPrompt, sessionTitle, req.Config.Model)
	cmd.Stderr = os.Stderr
	cmd.Dir = req.Config.ProjectDir
	// Set ORCH_WORKER=1 so agents know they are orch-managed workers.
	// Set CLAUDE_CONTEXT explicitly to prevent inheriting orchestrator context from parent.
	env := []string{"ORCH_WORKER=1", "CLAUDE_CONTEXT=" + req.Config.ClaudeContext()}
	if req.BeadsID != "" {
		env = append(env, "ORCH_BEADS_ID="+req.BeadsID)
	}
	cmd.Env = append(os.Environ(), env...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start opencode: %w", err)
	}

	processResult, err := opencode.ProcessOutput(stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to process output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("opencode exited with error: %w", err)
	}

	// Write session ID to workspace file for later lookups
	if processResult.SessionID != "" {
		if err := spawn.WriteSessionID(req.Config.WorkspacePath(), processResult.SessionID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write session ID: %v\n", err)
		}
	}

	// Orchestrator sessions use workspace artifacts for tracking
	// Log the session creation
	if err := LogSpawnEvent(processResult.SessionID, req, "inline", nil); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	return &Result{
		SessionID: processResult.SessionID,
		SpawnMode: "inline",
	}, nil
}
