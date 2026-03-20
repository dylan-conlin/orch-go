package backends

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// TmuxBackend spawns the agent in a tmux window (interactive, returns immediately).
// Creates a tmux window in workers-{project} session (or orchestrator session for orchestrator skills).
type TmuxBackend struct{}

// Name returns the backend name.
func (b *TmuxBackend) Name() string {
	return "tmux"
}

// Spawn creates a new agent session in tmux.
func (b *TmuxBackend) Spawn(ctx context.Context, req *SpawnRequest) (*Result, error) {
	var sessionName string
	var err error

	// Exploration orchestrators go into 'workers-{project}' (worker lifecycle)
	// Meta-orchestrators go into 'meta-orchestrator' tmux session
	// Orchestrator skills go into the 'orchestrator' tmux session
	// Workers go into 'workers-{project}' session
	if req.Config.Explore {
		sessionName, err = tmux.EnsureWorkersSession(req.Config.Project, req.Config.ProjectDir)
	} else if req.Config.IsMetaOrchestrator {
		sessionName, err = tmux.EnsureMetaOrchestratorSession()
	} else if req.Config.IsOrchestrator {
		sessionName, err = tmux.EnsureOrchestratorSession()
	} else {
		sessionName, err = tmux.EnsureWorkersSession(req.Config.Project, req.Config.ProjectDir)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to ensure tmux session: %w", err)
	}

	// Build window name with emoji and beads ID
	windowName := tmux.BuildWindowName(req.Config.WorkspaceName, req.Config.SkillName, req.BeadsID)

	// Create new tmux window
	windowTarget, windowID, err := tmux.CreateWindow(sessionName, windowName, req.Config.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create tmux window: %w", err)
	}

	// Pre-create session via HTTP API when model is specified.
	// opencode attach doesn't support --model, so we create the session first
	// (HTTP API accepts model) and then attach to it by session ID.
	var preCreatedSessionID string
	client := opencode.NewClient(req.ServerURL)
	if req.Config.Model != "" {
		sessionTitle := req.Config.WorkspaceName
		resp, err := client.CreateSession(sessionTitle, req.Config.ProjectDir, req.Config.Model, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to pre-create session with model %s: %w", req.Config.Model, err)
		}
		preCreatedSessionID = resp.ID
	}

	// Build opencode attach command (no --model; session ID used for pre-created sessions)
	opencodeCmd := tmux.BuildOpencodeAttachCommand(&tmux.OpencodeAttachConfig{
		ServerURL:     req.ServerURL,
		ProjectDir:    req.Config.ProjectDir,
		SessionID:     preCreatedSessionID,
		ClaudeContext: req.Config.ClaudeContext(),
	})

	// Send command and execute
	if err := tmux.SendKeys(windowTarget, opencodeCmd); err != nil {
		return nil, fmt.Errorf("failed to send opencode command: %w", err)
	}
	if err := tmux.SendEnter(windowTarget); err != nil {
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}

	// Wait for OpenCode TUI to be ready
	waitCfg := tmux.DefaultWaitConfig()
	if err := tmux.WaitForOpenCodeReady(windowTarget, waitCfg); err != nil {
		return nil, fmt.Errorf("failed to start opencode: %w", err)
	}

	// Determine session ID: use pre-created ID if available, otherwise discover from API
	sessionID := preCreatedSessionID
	if sessionID == "" {
		// Capture session ID from API with retry (OpenCode may not have registered yet)
		// Uses 3 attempts with 500ms initial delay, doubling each time (500ms, 1s, 2s)
		// Matches by directory + creation time (within 30s), not by title
		sessionID, _ = client.FindRecentSessionWithRetry(req.Config.ProjectDir, 3, 500*time.Millisecond)
		// Note: We silently ignore errors here since window_id is sufficient for tmux monitoring
	}

	// Send prompt
	sendCfg := tmux.DefaultSendPromptConfig()
	time.Sleep(sendCfg.PostReadyDelay)
	if err := tmux.SendKeysLiteral(windowTarget, req.MinimalPrompt); err != nil {
		return nil, fmt.Errorf("failed to send prompt: %w", err)
	}
	if err := tmux.SendEnter(windowTarget); err != nil {
		return nil, fmt.Errorf("failed to send enter: %w", err)
	}

	// Write session ID to workspace file for later lookups
	if sessionID != "" {
		if err := spawn.WriteSessionID(req.Config.WorkspacePath(), sessionID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write session ID: %v\n", err)
		}
	}

	// Orchestrator sessions use workspace artifacts for tracking
	// Build extra event data for tmux
	extraData := map[string]interface{}{
		"window":       windowTarget,
		"window_id":    windowID,
		"session_name": sessionName,
	}

	// Log the session creation
	if err := LogSpawnEvent(sessionID, req, "tmux", extraData); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Focus the newly created window
	if err := tmux.SelectWindow(windowTarget); err != nil {
		// Non-fatal - window was created successfully
		fmt.Fprintf(os.Stderr, "Warning: failed to focus window: %v\n", err)
	}

	result := &Result{
		SessionID: sessionID,
		SpawnMode: "tmux",
		TmuxInfo: &TmuxInfo{
			SessionName:  sessionName,
			WindowTarget: windowTarget,
			WindowID:     windowID,
		},
	}

	// Attach if requested
	if req.Attach {
		if err := tmux.Attach(windowTarget); err != nil {
			return result, fmt.Errorf("failed to attach to tmux: %w", err)
		}
	}

	return result, nil
}
