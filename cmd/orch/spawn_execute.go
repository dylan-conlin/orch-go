package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/episodic"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/process"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	statedb "github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

const sessionIDExtractionRetryDelay = 300 * time.Millisecond

var sessionIDExtractionRetrySleep = time.Sleep

func noTrackWaitHints(beadsID string, noTrack bool) (string, string, bool) {
	if !noTrack || beadsID == "" {
		return "", "", false
	}

	display := formatBeadsIDForDisplay(beadsID)
	handle := beadsID
	if display != beadsID {
		handle = fmt.Sprintf("%s (status: %s)", beadsID, display)
	}

	return handle, fmt.Sprintf("orch wait %s", beadsID), true
}

func printNoTrackWaitHandle(beadsID string, noTrack bool) {
	handle, waitCmd, ok := noTrackWaitHints(beadsID, noTrack)
	if !ok {
		return
	}

	fmt.Printf("  Handle:     %s\n", handle)
	fmt.Printf("  Wait:       %s\n", waitCmd)
}

// ensureSessionTitle enforces the expected session title after creation.
// This prevents OpenCode from falling back to auto-generated titles (e.g. skill names)
// when sessions are created through attach/tmux flows.
func ensureSessionTitle(client opencode.ClientInterface, sessionID, title string) {
	if sessionID == "" || title == "" {
		return
	}
	if err := client.UpdateSessionTitle(sessionID, title); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to enforce session title for %s: %v\n", sessionID, err)
	}
}

func addAttemptIDToEventData(eventData map[string]interface{}, cfg *spawn.Config) {
	if cfg == nil {
		return
	}

	attemptID := strings.TrimSpace(cfg.AttemptID)
	if attemptID == "" {
		attemptID = spawn.ReadAttemptID(cfg.WorkspacePath())
	}
	if attemptID != "" {
		eventData["attempt_id"] = attemptID
	}
}

// runSpawnInline spawns the agent inline (blocking) - original behavior.
func runSpawnInline(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {
	return runSpawnInlineWithClient(opencode.NewClient(serverURL), serverURL, cfg, minimalPrompt, beadsID, skillName, task)
}

func runSpawnInlineWithClient(client opencode.ClientInterface, serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {
	// Spawn opencode session
	// Format title with beads ID so orch status can match sessions
	sessionTitle := formatSessionTitle(cfg.WorkspaceName, beadsID)
	cmd := client.BuildSpawnCommand(minimalPrompt, sessionTitle, cfg.Model, cfg.Variant)
	cmd.Stderr = os.Stderr
	cmd.Dir = cfg.RuntimeDir()
	// Set ORCH_WORKER=1 so agents know they are orch-managed workers
	cmd.Env = append(os.Environ(), "ORCH_WORKER=1")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	result, err := opencode.ProcessOutput(stdout)
	if err != nil {
		return fmt.Errorf("failed to process output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("opencode exited with error: %w", err)
	}

	if result.SessionID != "" {
		if err := spawn.WriteSessionID(cfg.WorkspacePath(), result.SessionID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write session ID: %v\n", err)
		}
		if err := statedb.RecordSessionID(cfg.WorkspaceName, result.SessionID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to record session ID in state db: %v\n", err)
		}
		ensureSessionTitle(client, result.SessionID, sessionTitle)
	}

	registerOrchestratorSession(cfg, result.SessionID, task)

	inlineLogger := events.NewLogger(events.DefaultLogPath())
	inlineEventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"spawn_mode":          "inline",
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	if cfg.MCP != "" {
		inlineEventData["mcp"] = cfg.MCP
	}
	addGapAnalysisToEventData(inlineEventData, cfg.GapAnalysis)
	addUsageInfoToEventData(inlineEventData, cfg.UsageInfo)
	addAttemptIDToEventData(inlineEventData, cfg)
	inlineEvent := events.Event{
		Type:      "session.spawned",
		SessionID: result.SessionID,
		Timestamp: time.Now().Unix(),
		Data:      inlineEventData,
	}
	if err := inlineLogger.Log(inlineEvent); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}
	recordEpisodicEvent(inlineEvent, episodic.Context{
		Boundary:  episodic.BoundarySpawn,
		Project:   projectFromDir(cfg.ProjectDir),
		Workspace: cfg.WorkspaceName,
		SessionID: result.SessionID,
		BeadsID:   beadsID,
	})

	printSpawnSummaryWithGapWarning(cfg.GapAnalysis)

	fmt.Printf("Spawned agent:\n")
	fmt.Printf("  Session ID: %s\n", result.SessionID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	printNoTrackWaitHandle(beadsID, cfg.NoTrack)
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))

	return nil
}

// runSpawnHeadless spawns the agent using CLI subprocess without a TUI.
// This is useful for automation and daemon-driven spawns.
// Uses opencode CLI with --format json to properly support model selection
// (the HTTP API ignores the model parameter).
// Includes retry logic for transient network failures.
func runSpawnHeadless(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {
	return runSpawnHeadlessWithClient(opencode.NewClient(serverURL), serverURL, cfg, minimalPrompt, beadsID, skillName, task)
}

func runSpawnHeadlessWithClient(client opencode.ClientInterface, serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {
	sessionTitle := formatSessionTitle(cfg.WorkspaceName, beadsID)

	retryCfg := spawn.DefaultRetryConfig()
	result, retryResult := spawn.Retry(retryCfg, func() (*headlessSpawnResult, error) {
		return startHeadlessSession(client, serverURL, sessionTitle, minimalPrompt, cfg)
	})

	result, didSessionIDRetry := retryHeadlessSpawnAfterSessionIDExtractionFailure(result, retryResult, sessionIDExtractionRetryDelay, func() (*headlessSpawnResult, error) {
		return startHeadlessSession(client, serverURL, sessionTitle, minimalPrompt, cfg)
	})
	if didSessionIDRetry {
		fmt.Fprintf(os.Stderr, "Session ID extraction failed on initial attempt; retried once after %s\n", sessionIDExtractionRetryDelay)
	}

	if retryResult.LastErr != nil {
		spawnErr := spawn.WrapSpawnError(retryResult.LastErr, "Headless spawn failed")
		if retryResult.Attempts > 1 {
			fmt.Fprintf(os.Stderr, "Spawn failed after %d attempts\n", retryResult.Attempts)
		}
		fmt.Fprintf(os.Stderr, "\n%s\n", spawn.FormatSpawnError(spawnErr))
		return spawnErr
	}

	if retryResult.Attempts > 1 {
		fmt.Printf("Spawn succeeded after %d attempts\n", retryResult.Attempts)
	}

	sessionID := result.SessionID
	ensureSessionTitle(client, sessionID, sessionTitle)

	if err := spawn.WriteSessionID(cfg.WorkspacePath(), sessionID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write session ID: %v\n", err)
	}
	if err := statedb.RecordSessionID(cfg.WorkspaceName, sessionID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to record session ID in state db: %v\n", err)
	}

	if result.cmd != nil && result.cmd.Process != nil {
		childPID := result.cmd.Process.Pid
		if err := spawn.WriteProcessID(cfg.WorkspacePath(), childPID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write process ID: %v\n", err)
		}

		ledger := process.NewDefaultLedger()
		pgid, _ := syscall.Getpgid(childPID)
		ledgerEntry := process.LedgerEntry{
			Workspace: cfg.WorkspaceName,
			BeadsID:   beadsID,
			SessionID: sessionID,
			SpawnPID:  os.Getpid(),
			ChildPID:  childPID,
			PGID:      pgid,
			StartedAt: time.Now(),
			LastSeen:  time.Now(),
		}
		if err := ledger.Record(ledgerEntry); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to record process in ledger: %v\n", err)
		}
	}

	result.StartBackgroundCleanup()

	registerOrchestratorSession(cfg, sessionID, task)

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
	addAttemptIDToEventData(eventData, cfg)
	event := events.Event{
		Type:      "session.spawned",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}
	recordEpisodicEvent(event, episodic.Context{
		Boundary:  episodic.BoundarySpawn,
		Project:   projectFromDir(cfg.ProjectDir),
		Workspace: cfg.WorkspaceName,
		SessionID: sessionID,
		BeadsID:   beadsID,
	})

	printSpawnSummaryWithGapWarning(cfg.GapAnalysis)

	fmt.Printf("Spawned agent (headless):\n")
	fmt.Printf("  Session ID: %s\n", sessionID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	printNoTrackWaitHandle(beadsID, cfg.NoTrack)
	fmt.Printf("  Model:      %s\n", cfg.Model)
	if cfg.MCP != "" {
		fmt.Printf("  MCP:        %s\n", cfg.MCP)
	}
	if cfg.NoTrack {
		fmt.Printf("  Tracking:   disabled (--no-track)\n")
	}
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))

	return nil
}

func shouldRetrySessionIDExtraction(retryResult *spawn.RetryResult) bool {
	if retryResult == nil || retryResult.LastErr == nil {
		return false
	}

	var spawnErr *spawn.SpawnError
	if errors.As(retryResult.LastErr, &spawnErr) {
		return spawnErr.Kind == spawn.ErrKindSession && strings.HasPrefix(spawnErr.Message, "Failed to extract session ID")
	}

	return strings.Contains(retryResult.LastErr.Error(), "Failed to extract session ID")
}

func retryHeadlessSpawnAfterSessionIDExtractionFailure(initialResult *headlessSpawnResult, retryResult *spawn.RetryResult, retryDelay time.Duration, retryStart func() (*headlessSpawnResult, error)) (*headlessSpawnResult, bool) {
	if !shouldRetrySessionIDExtraction(retryResult) {
		return initialResult, false
	}

	if retryDelay > 0 {
		sessionIDExtractionRetrySleep(retryDelay)
	}

	retryResult.Attempts++
	recoveredResult, retryErr := retryStart()
	if retryErr != nil {
		retryResult.LastErr = retryErr
		return initialResult, true
	}

	retryResult.LastErr = nil
	return recoveredResult, true
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
		io.Copy(io.Discard, r.stdout)
		r.cmd.Wait()
	}()
}

// ansiRegex matches ANSI escape sequences (colors, formatting, etc.)
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes ANSI escape codes from a string for cleaner error messages.
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// startHeadlessSession starts an opencode session and extracts the session ID.
// Returns the result with session ID and resources for cleanup.
// Note: Uses CLI mode instead of HTTP API because OpenCode's HTTP API ignores the model parameter.
// CLI mode correctly honors the --model flag.
// See: .kb/investigations/2025-12-23-inv-model-selection-issue-architect-agent.md
func startHeadlessSession(client opencode.ClientInterface, serverURL, sessionTitle, minimalPrompt string, cfg *spawn.Config) (*headlessSpawnResult, error) {
	cmd := client.BuildSpawnCommand(minimalPrompt, sessionTitle, cfg.Model, cfg.Variant)
	cmd.Dir = cfg.RuntimeDir()
	cmd.Env = append(os.Environ(), "ORCH_WORKER=1")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		spawnErr := spawn.WrapSpawnError(err, "Failed to get stdout pipe")
		return nil, spawnErr
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		spawnErr := spawn.WrapSpawnError(err, "Failed to start opencode process")
		return nil, spawnErr
	}

	sessionID, err := opencode.ExtractSessionIDFromReader(stdout)
	if err != nil {
		cmd.Process.Kill()
		stderrContent := strings.TrimSpace(stderrBuf.String())
		stderrContent = stripANSI(stderrContent)
		errMsg := "Failed to extract session ID"
		if stderrContent != "" {
			errMsg = fmt.Sprintf("Failed to extract session ID: %s", stderrContent)
		}
		spawnErr := spawn.WrapSpawnError(err, errMsg)
		return nil, spawnErr
	}

	return &headlessSpawnResult{
		SessionID: sessionID,
		cmd:       cmd,
		stdout:    stdout,
	}, nil
}

// runSpawnTmux spawns the agent in a tmux window (interactive, returns immediately).
// Creates a tmux window in workers-{project} session (or orchestrator session for orchestrator skills).
func runSpawnTmux(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string, attach bool) error {
	return runSpawnTmuxWithClient(opencode.NewClient(serverURL), serverURL, cfg, minimalPrompt, beadsID, skillName, task, attach)
}

func runSpawnTmuxWithClient(client opencode.ClientInterface, serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string, attach bool) error {
	var sessionName string
	var err error

	if cfg.IsMetaOrchestrator || cfg.IsOrchestrator {
		sessionName, err = tmux.EnsureOrchestratorSession()
	} else {
		sessionName, err = tmux.EnsureWorkersSession(cfg.Project, cfg.ProjectDir)
	}
	if err != nil {
		return fmt.Errorf("failed to ensure tmux session: %w", err)
	}

	windowName := tmux.BuildWindowName(cfg.WorkspaceName, cfg.SkillName, beadsID)

	windowTarget, windowID, err := tmux.CreateWindow(sessionName, windowName, cfg.RuntimeDir())
	if err != nil {
		return fmt.Errorf("failed to create tmux window: %w", err)
	}

	sessionTitle := formatSessionTitle(cfg.WorkspaceName, beadsID)

	var preCreatedSessionID string
	resp, createErr := client.CreateSession(sessionTitle, cfg.RuntimeDir(), cfg.Model, cfg.Variant, !cfg.IsOrchestrator && !cfg.IsMetaOrchestrator)
	if createErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to pre-create session with title %q: %v (falling back to attach without pre-created session)\n", sessionTitle, createErr)
	} else {
		preCreatedSessionID = resp.ID
	}

	attachCfg := &tmux.OpencodeAttachConfig{
		ServerURL:  serverURL,
		ProjectDir: cfg.RuntimeDir(),
		SessionID:  preCreatedSessionID,
	}
	opencodeCmd := tmux.BuildOpencodeAttachCommand(attachCfg)

	if err := tmux.SendKeys(windowTarget, opencodeCmd); err != nil {
		return fmt.Errorf("failed to send opencode command: %w", err)
	}
	if err := tmux.SendEnter(windowTarget); err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	waitCfg := tmux.DefaultWaitConfig()
	if err := tmux.WaitForOpenCodeReady(windowTarget, waitCfg); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	sessionID := preCreatedSessionID
	if sessionID == "" {
		sessionID, _ = client.FindRecentSessionWithRetry(cfg.RuntimeDir(), 3, 500*time.Millisecond)
	}
	ensureSessionTitle(client, sessionID, sessionTitle)

	sendCfg := tmux.DefaultSendPromptConfig()
	time.Sleep(sendCfg.PostReadyDelay)
	if err := tmux.SendKeysLiteral(windowTarget, minimalPrompt); err != nil {
		return fmt.Errorf("failed to send prompt: %w", err)
	}
	if err := tmux.SendEnter(windowTarget); err != nil {
		return fmt.Errorf("failed to send enter: %w", err)
	}

	if sessionID != "" {
		if err := spawn.WriteSessionID(cfg.WorkspacePath(), sessionID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write session ID: %v\n", err)
		}
		if err := statedb.RecordSessionID(cfg.WorkspaceName, sessionID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to record session ID in state db: %v\n", err)
		}
	}

	if err := statedb.RecordTmuxWindow(cfg.WorkspaceName, windowTarget); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to record tmux window in state db: %v\n", err)
	}

	registerOrchestratorSession(cfg, sessionID, task)

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
	addAttemptIDToEventData(eventData, cfg)
	event := events.Event{
		Type:      "session.spawned",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}
	recordEpisodicEvent(event, episodic.Context{
		Boundary:  episodic.BoundarySpawn,
		Project:   filepath.Base(cfg.ProjectDir),
		Workspace: cfg.WorkspaceName,
		SessionID: sessionID,
		BeadsID:   beadsID,
	})

	if !cfg.DaemonDriven {
		selectCmd := exec.Command("tmux", "select-window", "-t", windowTarget)
		if err := selectCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to focus window: %v\n", err)
		}
	}

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
	printNoTrackWaitHandle(beadsID, cfg.NoTrack)
	fmt.Printf("  Model:      %s\n", cfg.Model)
	if cfg.MCP != "" {
		fmt.Printf("  MCP:        %s\n", cfg.MCP)
	}
	if cfg.NoTrack {
		fmt.Printf("  Tracking:   disabled (--no-track)\n")
	}
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))

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

	if err := statedb.RecordTmuxWindow(cfg.WorkspaceName, result.Window); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to record tmux window in state db: %v\n", err)
	}

	registerOrchestratorSession(cfg, "", task)

	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"window":              result.Window,
		"window_id":           result.WindowID,
		"spawn_mode":          "claude",
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	addGapAnalysisToEventData(eventData, cfg.GapAnalysis)
	addUsageInfoToEventData(eventData, cfg.UsageInfo)
	addAttemptIDToEventData(eventData, cfg)
	event := events.Event{
		Type:      "session.spawned",
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}
	recordEpisodicEvent(event, episodic.Context{
		Boundary:  episodic.BoundarySpawn,
		Project:   projectFromDir(cfg.ProjectDir),
		Workspace: cfg.WorkspaceName,
		BeadsID:   beadsID,
	})

	if !cfg.DaemonDriven {
		selectCmd := exec.Command("tmux", "select-window", "-t", result.Window)
		if err := selectCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to focus window: %v\n", err)
		}
	}

	printSpawnSummaryWithGapWarning(cfg.GapAnalysis)

	fmt.Printf("Spawned agent in Claude mode (tmux):\n")
	fmt.Printf("  Window:     %s\n", result.Window)
	fmt.Printf("  Window ID:  %s\n", result.WindowID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	printNoTrackWaitHandle(beadsID, cfg.NoTrack)
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))

	if attach {
		if err := tmux.Attach(result.Window); err != nil {
			return fmt.Errorf("failed to attach to tmux: %w", err)
		}
	}

	return nil
}

// runSpawnClaudeInline spawns the agent using Claude Code CLI inline (blocking).
// This runs claude directly in the current terminal without tmux, for interactive sessions.
func runSpawnClaudeInline(serverURL string, cfg *spawn.Config, beadsID, skillName, task string) error {
	registerOrchestratorSession(cfg, "", task)

	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"spawn_mode":          "claude-inline",
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	addGapAnalysisToEventData(eventData, cfg.GapAnalysis)
	addUsageInfoToEventData(eventData, cfg.UsageInfo)
	addAttemptIDToEventData(eventData, cfg)
	event := events.Event{
		Type:      "session.spawned",
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}
	recordEpisodicEvent(event, episodic.Context{
		Boundary:  episodic.BoundarySpawn,
		Project:   projectFromDir(cfg.ProjectDir),
		Workspace: cfg.WorkspaceName,
		BeadsID:   beadsID,
	})

	printSpawnSummaryWithGapWarning(cfg.GapAnalysis)

	fmt.Printf("Spawning agent in Claude mode (inline):\n")
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	printNoTrackWaitHandle(beadsID, cfg.NoTrack)
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))
	fmt.Println()

	if err := spawn.SpawnClaudeInline(cfg); err != nil {
		return err
	}

	return nil
}

// runSpawnDocker spawns the agent using Docker for Statsig fingerprint isolation.
// This is an escape hatch for rate limit scenarios - provides fresh fingerprint per spawn.
func runSpawnDocker(serverURL string, cfg *spawn.Config, beadsID, skillName, task string, attach bool) error {
	result, err := spawn.SpawnDocker(cfg)
	if err != nil {
		return err
	}

	if err := statedb.RecordTmuxWindow(cfg.WorkspaceName, result.Window); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to record tmux window in state db: %v\n", err)
	}

	registerOrchestratorSession(cfg, "", task)

	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"window":              result.Window,
		"window_id":           result.WindowID,
		"spawn_mode":          "docker",
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	addGapAnalysisToEventData(eventData, cfg.GapAnalysis)
	addUsageInfoToEventData(eventData, cfg.UsageInfo)
	addAttemptIDToEventData(eventData, cfg)
	event := events.Event{
		Type:      "session.spawned",
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}
	recordEpisodicEvent(event, episodic.Context{
		Boundary:  episodic.BoundarySpawn,
		Project:   projectFromDir(cfg.ProjectDir),
		Workspace: cfg.WorkspaceName,
		BeadsID:   beadsID,
	})

	if !cfg.DaemonDriven {
		selectCmd := exec.Command("tmux", "select-window", "-t", result.Window)
		if err := selectCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to focus window: %v\n", err)
		}
	}

	printSpawnSummaryWithGapWarning(cfg.GapAnalysis)

	fmt.Printf("Spawned agent in Docker mode (rate limit escape hatch):\n")
	fmt.Printf("  Window:     %s\n", result.Window)
	fmt.Printf("  Window ID:  %s\n", result.WindowID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	printNoTrackWaitHandle(beadsID, cfg.NoTrack)
	fmt.Printf("  Container:  %s\n", spawn.DockerImageName)
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))

	if attach {
		if err := tmux.Attach(result.Window); err != nil {
			return fmt.Errorf("failed to attach to tmux: %w", err)
		}
	}

	return nil
}
