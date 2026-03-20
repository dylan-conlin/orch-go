package orch

import (
	"fmt"
	"os"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// headlessSpawnResult contains the result of starting a headless session.
type headlessSpawnResult struct {
	SessionID string
}

// DispatchSpawn routes to the appropriate spawn mode function.
// Handles inline, headless, claude, and tmux modes.
func DispatchSpawn(input *SpawnInput, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task, serverURL string) error {
	// Wire MCP config into opencode.json for OpenCode backend spawns.
	// Claude backend handles MCP via --mcp-config CLI flag (see BuildClaudeLaunchCommand).
	// OpenCode reads MCP config from opencode.json in the project directory.
	// Browser automation via playwright-cli is handled separately via BrowserTool field
	// and context injection, not through MCP config.
	if cfg.MCP != "" && cfg.SpawnMode != "claude" {
		if err := spawn.EnsureOpenCodeMCP(cfg.ProjectDir, cfg.MCP); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to inject MCP config into opencode.json: %v\n", err)
		}
	}

	// Spawn mode: inline (blocking TUI), tmux (opt-in for workers, default for orchestrators), claude (tmux), or headless (default for workers)
	if input.Inline {
		// Inline mode (blocking) - run in current terminal with TUI
		return runSpawnInline(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
	}

	// Explicit --headless flag overrides all other mode decisions
	if input.Headless {
		return runSpawnHeadless(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
	}

	// Claude mode: Use tmux + claude CLI
	if cfg.SpawnMode == "claude" {
		return runSpawnClaude(serverURL, cfg, beadsID, skillName, task, input.Attach)
	}

	// Orchestrator-type skills default to tmux mode (visible interaction)
	// Workers default to headless mode (automation-friendly)
	useTmux := input.Tmux || input.Attach || cfg.IsOrchestrator
	if useTmux {
		// Tmux mode - visible, interruptible
		// Default for orchestrator skills, opt-in for workers
		return runSpawnTmux(serverURL, cfg, minimalPrompt, beadsID, skillName, task, input.Attach)
	}

	// Default for workers: Headless mode - spawn via HTTP API (automation-friendly, no TUI overhead)
	return runSpawnHeadless(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
}

// runSpawnInline spawns the agent inline (blocking) using the HTTP API.
// Uses CreateSession + SendMessageInDirectory to properly pass x-opencode-directory
// header, ensuring the session is created in the correct project directory.
// Blocks until the session completes by watching SSE events.
func runSpawnInline(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {
	client := opencode.NewClient(serverURL)
	sessionTitle := formatSessionTitle(cfg.WorkspaceName, beadsID)

	// Step 1: Create session via HTTP API with correct directory
	// CreateSession passes x-opencode-directory header so the server uses the target project dir
	metadata := map[string]string{
		"beads_id":       cfg.BeadsID,
		"workspace_path": cfg.WorkspacePath(),
		"tier":           cfg.Tier,
		"spawn_mode":     "inline",
		"skill":          cfg.SkillName,
		"model":          cfg.Model,
	}

	// Calculate TTL based on session type
	// Worker sessions: 4 hours (14400 seconds)
	// Orchestrator sessions: 0 (no expiration)
	timeTTL := 4 * 60 * 60 // 4 hours in seconds
	if cfg.IsOrchestrator {
		timeTTL = 0 // Orchestrator sessions persist until manually cleaned
	}

	session, err := client.CreateSession(sessionTitle, cfg.ProjectDir, cfg.Model, metadata, timeTTL)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	sessionID := session.ID

	// Step 2: Send the initial prompt with model selection and directory context
	// The directory header ensures the server resolves the correct project context
	if err := client.SendMessageInDirectory(sessionID, minimalPrompt, cfg.Model, cfg.ProjectDir); err != nil {
		return fmt.Errorf("failed to send prompt: %w", err)
	}

	fmt.Printf("Inline agent spawned (session: %s), waiting for completion...\n", sessionID)

	// Step 3: Wait for session to complete (blocking)
	// Watches SSE events for busy→idle transition to maintain inline mode's blocking behavior
	if err := client.WaitForSessionIdle(sessionID); err != nil {
		return fmt.Errorf("error waiting for session: %w", err)
	}

	// Atomic spawn Phase 2: update manifest with session ID
	inlineAtomicOpts := &spawn.AtomicSpawnOpts{Config: cfg, BeadsID: beadsID}
	if err := spawn.AtomicSpawnPhase2(inlineAtomicOpts, sessionID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update manifest with session ID: %v\n", err)
	}

	// Log the session creation
	inlineLogger := events.NewLogger(events.DefaultLogPath())
	inlineEventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"spawn_mode":          "inline",
		"model":               cfg.Model,
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	if cfg.MCP != "" {
		inlineEventData["mcp"] = cfg.MCP
	}
	addGapAnalysisToEventData(inlineEventData, cfg.GapAnalysis)
	addUsageInfoToEventData(inlineEventData, cfg.UsageInfo)
	inlineEvent := events.Event{
		Type:      "session.spawned",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data:      inlineEventData,
	}
	if err := inlineLogger.Log(inlineEvent); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print spawn summary with prominent gap warning if needed
	printSpawnSummaryWithGapWarning(cfg.GapAnalysis)

	fmt.Printf("Agent completed:\n")
	fmt.Printf("  Session ID: %s\n", sessionID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	// Print context quality with visual indicators
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))

	return nil
}

// runSpawnHeadless spawns the agent using CLI subprocess without a TUI.
// This is useful for automation and daemon-driven spawns.
// Uses opencode CLI with --format json to properly support model selection
// (the HTTP API ignores the model parameter).
// Includes retry logic for transient network failures.
func runSpawnHeadless(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {
	client := opencode.NewClient(serverURL)

	// Build opencode command using CLI (like inline mode) to support model selection
	// The HTTP API ignores model parameter - only CLI mode honors --model flag
	// Format title with beads ID so orch status can match sessions
	sessionTitle := formatSessionTitle(cfg.WorkspaceName, beadsID)

	// Use retry logic for transient failures (network issues, server temporarily unavailable)
	retryCfg := spawn.DefaultRetryConfig()
	result, retryResult := spawn.Retry(retryCfg, func() (*headlessSpawnResult, error) {
		return startHeadlessSession(client, serverURL, sessionTitle, minimalPrompt, cfg)
	})

	if retryResult.LastErr != nil {
		// Wrap the error with user-friendly message and recovery guidance
		spawnErr := spawn.WrapSpawnError(retryResult.LastErr, "Headless spawn failed")
		if retryResult.Attempts > 1 {
			fmt.Fprintf(os.Stderr, "Spawn failed after %d attempts\n", retryResult.Attempts)
		}
		// Print formatted error with recovery guidance
		fmt.Fprintf(os.Stderr, "\n%s\n", spawn.FormatSpawnError(spawnErr))
		return spawnErr
	}

	if retryResult.Attempts > 1 {
		fmt.Printf("Spawn succeeded after %d attempts\n", retryResult.Attempts)
	}

	sessionID := result.SessionID

	// Atomic spawn Phase 2: update manifest with session ID
	headlessAtomicOpts := &spawn.AtomicSpawnOpts{Config: cfg, BeadsID: beadsID}
	if err := spawn.AtomicSpawnPhase2(headlessAtomicOpts, sessionID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update manifest with session ID: %v\n", err)
	}

	// Log the session creation
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"session_id":          sessionID,
		"spawn_mode":          "headless",
		"model":               cfg.Model,
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	if retryResult.Attempts > 1 {
		eventData["retry_attempts"] = retryResult.Attempts
	}
	if cfg.MCP != "" {
		eventData["mcp"] = cfg.MCP
	}
	addGapAnalysisToEventData(eventData, cfg.GapAnalysis)
	addUsageInfoToEventData(eventData, cfg.UsageInfo)
	event := events.Event{
		Type:      "session.spawned",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print spawn summary with prominent gap warning if needed
	printSpawnSummaryWithGapWarning(cfg.GapAnalysis)

	fmt.Printf("Spawned agent (headless):\n")
	fmt.Printf("  Session ID: %s\n", sessionID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	fmt.Printf("  Model:      %s\n", cfg.Model)
	if cfg.MCP != "" {
		fmt.Printf("  MCP:        %s\n", cfg.MCP)
	}
	if cfg.NoTrack {
		fmt.Printf("  Tracking:   disabled (--no-track)\n")
	}
	// Print context quality with visual indicators
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))

	return nil
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
		"skill":          cfg.SkillName,
		"model":          cfg.Model,
	}

	// Calculate TTL based on session type
	// Worker sessions: 4 hours (14400 seconds)
	// Orchestrator sessions: 0 (no expiration)
	timeTTL := 4 * 60 * 60 // 4 hours in seconds
	if cfg.IsOrchestrator {
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
	if err := client.VerifySessionAfterPrompt(session.ID, cfg.ProjectDir, 3*time.Second); err != nil {
		return nil, spawn.WrapSpawnError(err, "Session failed verification after prompt")
	}

	return &headlessSpawnResult{
		SessionID: session.ID,
	}, nil
}

// runSpawnTmux spawns the agent in a tmux window (interactive, returns immediately).
// Creates a tmux window in workers-{project} session (or orchestrator session for orchestrator skills).
func runSpawnTmux(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string, attach bool) error {
	var sessionName string
	var err error

	// Meta-orchestrators and orchestrators go into 'orchestrator' tmux session
	// Workers go into 'workers-{project}' session
	if cfg.IsMetaOrchestrator || cfg.IsOrchestrator {
		sessionName, err = tmux.EnsureOrchestratorSession()
	} else {
		sessionName, err = tmux.EnsureWorkersSession(cfg.Project, cfg.ProjectDir)
	}
	if err != nil {
		return fmt.Errorf("failed to ensure tmux session: %w", err)
	}

	// Build window name with emoji and beads ID
	windowName := tmux.BuildWindowName(cfg.WorkspaceName, cfg.SkillName, beadsID)

	// Create new tmux window
	windowTarget, windowID, err := tmux.CreateWindow(sessionName, windowName, cfg.ProjectDir)
	if err != nil {
		return fmt.Errorf("failed to create tmux window: %w", err)
	}

	// Pre-create session via HTTP API when model is specified.
	// opencode attach doesn't support --model, so we create the session first
	// (HTTP API accepts model) and then attach to it by session ID.
	var preCreatedSessionID string
	client := opencode.NewClient(serverURL)
	if cfg.Model != "" {
		sessionTitle := cfg.WorkspaceName
		resp, err := client.CreateSession(sessionTitle, cfg.ProjectDir, cfg.Model, nil, 0)
		if err != nil {
			return fmt.Errorf("failed to pre-create session with model %s: %w", cfg.Model, err)
		}
		preCreatedSessionID = resp.ID
	}

	// Build opencode attach command (no --model; session ID used for pre-created sessions)
	opencodeCmd := tmux.BuildOpencodeAttachCommand(&tmux.OpencodeAttachConfig{
		ServerURL:     serverURL,
		ProjectDir:    cfg.ProjectDir,
		SessionID:     preCreatedSessionID,
		ClaudeContext: cfg.ClaudeContext(),
	})

	// Send command and execute
	if err := tmux.SendKeys(windowTarget, opencodeCmd); err != nil {
		return fmt.Errorf("failed to send opencode command: %w", err)
	}
	if err := tmux.SendEnter(windowTarget); err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	// Wait for OpenCode TUI to be ready
	waitCfg := tmux.DefaultWaitConfig()
	if err := tmux.WaitForOpenCodeReady(windowTarget, waitCfg); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	// Determine session ID: use pre-created ID if available, otherwise discover from API
	sessionID := preCreatedSessionID
	if sessionID == "" {
		// Capture session ID from API with retry (OpenCode may not have registered yet)
		// Uses 3 attempts with 500ms initial delay, doubling each time (500ms, 1s, 2s)
		// Matches by directory + creation time (within 30s), not by title
		sessionID, _ = client.FindRecentSessionWithRetry(cfg.ProjectDir, 3, 500*time.Millisecond)
		// Note: We silently ignore errors here since window_id is sufficient for tmux monitoring
	}

	// Send prompt
	sendCfg := tmux.DefaultSendPromptConfig()
	time.Sleep(sendCfg.PostReadyDelay)
	if err := tmux.SendKeysLiteral(windowTarget, minimalPrompt); err != nil {
		return fmt.Errorf("failed to send prompt: %w", err)
	}
	if err := tmux.SendEnter(windowTarget); err != nil {
		return fmt.Errorf("failed to send enter: %w", err)
	}

	// Atomic spawn Phase 2: update manifest with session ID + set session metadata
	if sessionID != "" {
		metadata := map[string]string{
			"beads_id":       cfg.BeadsID,
			"workspace_path": cfg.WorkspacePath(),
			"tier":           cfg.Tier,
			"spawn_mode":     "tmux",
			"skill":          cfg.SkillName,
			"model":          cfg.Model,
		}
		if err := client.SetSessionMetadata(sessionID, metadata); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to set session metadata: %v\n", err)
		}
		tmuxAtomicOpts := &spawn.AtomicSpawnOpts{Config: cfg, BeadsID: beadsID}
		if err := spawn.AtomicSpawnPhase2(tmuxAtomicOpts, sessionID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update manifest with session ID: %v\n", err)
		}
	}

	// Log the session creation
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"session_id":          sessionID,
		"window":              windowTarget,
		"window_id":           windowID,
		"session_name":        sessionName,
		"spawn_mode":          "tmux",
		"model":               cfg.Model,
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	if cfg.MCP != "" {
		eventData["mcp"] = cfg.MCP
	}
	addGapAnalysisToEventData(eventData, cfg.GapAnalysis)
	addUsageInfoToEventData(eventData, cfg.UsageInfo)
	event := events.Event{
		Type:      "session.spawned",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Focus the newly created window
	if err := tmux.SelectWindow(windowTarget); err != nil {
		// Non-fatal - window was created successfully
		fmt.Fprintf(os.Stderr, "Warning: failed to focus window: %v\n", err)
	}

	// Print spawn summary with prominent gap warning if needed
	printSpawnSummaryWithGapWarning(cfg.GapAnalysis)

	fmt.Printf("Spawned agent in tmux:\n")
	fmt.Printf("  Session:    %s\n", sessionName)
	if sessionID != "" {
		fmt.Printf("  Session ID: %s\n", sessionID)
	}
	fmt.Printf("  Window:     %s\n", windowTarget)
	fmt.Printf("  Window ID:  %s\n", windowID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	fmt.Printf("  Model:      %s\n", cfg.Model)
	if cfg.MCP != "" {
		fmt.Printf("  MCP:        %s\n", cfg.MCP)
	}
	if cfg.NoTrack {
		fmt.Printf("  Tracking:   disabled (--no-track)\n")
	}
	// Print context quality with visual indicators
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))

	// Attach if requested
	if attach {
		if err := tmux.Attach(windowTarget); err != nil {
			return fmt.Errorf("failed to attach to tmux: %w", err)
		}
	}

	return nil
}

// runSpawnClaude spawns the agent using Claude Code CLI in a tmux window.
func runSpawnClaude(serverURL string, cfg *spawn.Config, beadsID, skillName, task string, attach bool) error {
	result, err := spawn.SpawnClaude(cfg)
	if err != nil {
		return err
	}

	// Atomic spawn Phase 2: write window ID as session tracking breadcrumb.
	// Claude backend doesn't have an OpenCode session ID, but the tmux window ID
	// provides lifecycle visibility (detect if process died vs still running).
	claudeAtomicOpts := &spawn.AtomicSpawnOpts{Config: cfg, BeadsID: beadsID}
	if err := spawn.AtomicSpawnPhase2(claudeAtomicOpts, result.WindowID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update manifest with window ID: %v\n", err)
	}

	// Log the session creation
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"window":              result.Window,
		"window_id":           result.WindowID,
		"spawn_mode":          "claude",
		"model":               cfg.Model,
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	if cfg.MCP != "" {
		eventData["mcp"] = cfg.MCP
	}
	addGapAnalysisToEventData(eventData, cfg.GapAnalysis)
	addUsageInfoToEventData(eventData, cfg.UsageInfo)
	event := events.Event{
		Type:      "session.spawned",
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Focus the newly created window
	if err := tmux.SelectWindow(result.Window); err != nil {
		// Non-fatal
		fmt.Fprintf(os.Stderr, "Warning: failed to focus window: %v\n", err)
	}

	// Print spawn summary with prominent gap warning if needed
	printSpawnSummaryWithGapWarning(cfg.GapAnalysis)

	fmt.Printf("Spawned agent in Claude mode (tmux):\n")
	fmt.Printf("  Window:     %s\n", result.Window)
	fmt.Printf("  Window ID:  %s\n", result.WindowID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	if cfg.MCP != "" {
		fmt.Printf("  MCP:        %s\n", cfg.MCP)
	}
	// Print context quality with visual indicators
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))

	// Attach if requested
	if attach {
		if err := tmux.Attach(result.Window); err != nil {
			return fmt.Errorf("failed to attach to tmux: %w", err)
		}
	}

	return nil
}
