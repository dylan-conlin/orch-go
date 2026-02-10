// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/attention"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

const (
	completionSourcePhaseComplete = "phase_complete"
	completionSourceIdleTracked   = "idle_tracked"

	defaultIdleCompletionThreshold = 15 * time.Minute
)

// CompletionConfig holds configuration for the completion processing loop.
type CompletionConfig struct {
	// PollInterval is the time between polling cycles.
	PollInterval time.Duration

	// DryRun shows what would be processed without actually closing issues.
	DryRun bool

	// Verbose enables detailed output.
	Verbose bool

	// WorkspaceDir is the base directory for agent workspaces.
	// Defaults to .orch/workspace/ relative to project root.
	WorkspaceDir string

	// ProjectDir is the project root directory.
	// Used to locate workspaces and verify constraints.
	ProjectDir string

	// ServerURL is the OpenCode server URL.
	// Used for mode-aware backend verification.
	ServerURL string

	// IdleCompletionThreshold is the minimum session idle duration required
	// before tracked-session auto-detection can mark an agent complete.
	IdleCompletionThreshold time.Duration
}

// DefaultCompletionConfig returns sensible defaults for completion configuration.
func DefaultCompletionConfig() CompletionConfig {
	return CompletionConfig{
		PollInterval:            60 * time.Second,
		DryRun:                  false,
		Verbose:                 false,
		IdleCompletionThreshold: defaultIdleCompletionThreshold,
	}
}

// CompletedAgent represents an agent that can be auto-completed by the daemon.
type CompletedAgent struct {
	BeadsID       string
	Title         string
	Status        string // open or in_progress
	PhaseSummary  string // Summary from "Phase: Complete - <summary>"
	WorkspacePath string // Path to agent workspace (if found)
	Source        string // completionSourcePhaseComplete | completionSourceIdleTracked
	SessionID     string // OpenCode session ID for post-completion cleanup
	IdleDuration  time.Duration
}

// CompletionResult contains the result of processing a completion.
type CompletionResult struct {
	BeadsID      string
	Processed    bool
	CloseReason  string
	Error        error
	Verification verify.VerificationResult
	Escalation   verify.EscalationLevel // Escalation level for this completion
}

// CompletionLoopResult contains the results of a completion loop iteration.
type CompletionLoopResult struct {
	Processed []CompletionResult
	Errors    []error
}

// ListCompletedAgents finds all agents that have reported Phase: Complete
// but whose beads issues are still open or in_progress.
func (d *Daemon) ListCompletedAgents(config CompletionConfig) ([]CompletedAgent, error) {
	if d.listCompletedAgentsFunc != nil {
		return d.listCompletedAgentsFunc(config)
	}
	return ListCompletedAgentsDefault(config)
}

// ListCompletedAgentsDefault is the default implementation that queries beads.
func ListCompletedAgentsDefault(config CompletionConfig) ([]CompletedAgent, error) {
	// Get all open/in_progress issues
	openIssues, err := verify.ListOpenIssues()
	if err != nil {
		return nil, fmt.Errorf("failed to list open issues: %w", err)
	}

	if len(openIssues) == 0 {
		return nil, nil
	}

	// Collect beads IDs for batch comment fetch
	var beadsIDs []string
	for id := range openIssues {
		beadsIDs = append(beadsIDs, id)
	}

	// Fetch comments for all issues in batch
	commentMap := verify.GetCommentsBatch(beadsIDs)

	var completed []CompletedAgent
	detector := newIdleCompletionDetector(config.ServerURL, config.IdleCompletionThreshold)

	for id, issue := range openIssues {
		comments, ok := commentMap[id]
		if !ok {
			continue
		}

		workspacePath := findWorkspaceForIssue(id, config.WorkspaceDir, config.ProjectDir)
		sessionID := ""
		if detector != nil {
			sessionID = detector.SessionIDForBeads(id, workspacePath)
		} else if workspacePath != "" {
			sessionID = spawn.ReadSessionID(workspacePath)
		}

		// Parse phase from comments
		phaseStatus := verify.ParsePhaseFromComments(comments)

		if phaseStatus.Found && strings.EqualFold(phaseStatus.Phase, "Complete") {
			completed = append(completed, CompletedAgent{
				BeadsID:       id,
				Title:         issue.Title,
				Status:        issue.Status,
				PhaseSummary:  phaseStatus.Summary,
				WorkspacePath: workspacePath,
				Source:        completionSourcePhaseComplete,
				SessionID:     sessionID,
			})
			continue
		}

		// Aggressive auto-detection path: tracked in-progress session + idle window.
		if !strings.EqualFold(issue.Status, "in_progress") || detector == nil {
			continue
		}

		signal, detected := detector.Detect(id, workspacePath)
		if !detected {
			continue
		}

		completed = append(completed, CompletedAgent{
			BeadsID:       id,
			Title:         issue.Title,
			Status:        issue.Status,
			PhaseSummary:  signal.Summary,
			WorkspacePath: workspacePath,
			Source:        completionSourceIdleTracked,
			SessionID:     signal.SessionID,
			IdleDuration:  signal.IdleDuration,
		})
	}

	return completed, nil
}

type idleCompletionSignal struct {
	SessionID    string
	IdleDuration time.Duration
	Summary      string
}

type idleCompletionDetector struct {
	client       opencode.ClientInterface
	now          time.Time
	idleWindow   time.Duration
	sessionsByID map[string]opencode.Session
	sessionByID  map[string]string
}

func newIdleCompletionDetector(serverURL string, idleWindow time.Duration) *idleCompletionDetector {
	idleWindow = normalizeIdleCompletionThreshold(idleWindow)
	if serverURL == "" {
		serverURL = opencode.DefaultServerURL
	}

	client := opencode.NewClient(serverURL)
	sessions, err := client.ListSessions("")
	if err != nil || len(sessions) == 0 {
		return nil
	}

	d := &idleCompletionDetector{
		client:       client,
		now:          time.Now(),
		idleWindow:   idleWindow,
		sessionsByID: make(map[string]opencode.Session, len(sessions)),
		sessionByID:  make(map[string]string, len(sessions)),
	}

	for _, s := range sessions {
		d.sessionsByID[s.ID] = s
		if beadsID := extractBeadsIDFromSessionTitle(s.Title); beadsID != "" && !isUntrackedBeadsID(beadsID) {
			d.sessionByID[beadsID] = s.ID
		}
	}

	return d
}

func (d *idleCompletionDetector) Detect(beadsID, workspacePath string) (idleCompletionSignal, bool) {
	if beadsID == "" {
		return idleCompletionSignal{}, false
	}

	sessionID := d.SessionIDForBeads(beadsID, workspacePath)
	if sessionID == "" {
		return idleCompletionSignal{}, false
	}

	session, ok := d.sessionsByID[sessionID]
	if !ok {
		return idleCompletionSignal{}, false
	}

	updatedAt := time.Unix(session.Time.Updated/1000, 0)
	idleDuration := d.now.Sub(updatedAt)
	if idleDuration < d.idleWindow {
		return idleCompletionSignal{}, false
	}

	if d.client != nil && d.client.IsSessionProcessing(sessionID) {
		return idleCompletionSignal{}, false
	}

	return idleCompletionSignal{
		SessionID:    sessionID,
		IdleDuration: idleDuration,
		Summary: fmt.Sprintf(
			"Auto-detected by daemon: tracked session %s idle for %s",
			shortSessionID(sessionID),
			idleDuration.Round(time.Minute),
		),
	}, true
}

func (d *idleCompletionDetector) SessionIDForBeads(beadsID, workspacePath string) string {
	if workspacePath != "" {
		if sessionID := spawn.ReadSessionID(workspacePath); sessionID != "" {
			return sessionID
		}
	}

	return d.sessionByID[beadsID]
}

func normalizeIdleCompletionThreshold(idleWindow time.Duration) time.Duration {
	if idleWindow <= 0 {
		return defaultIdleCompletionThreshold
	}
	return idleWindow
}

func shortSessionID(sessionID string) string {
	if len(sessionID) <= 12 {
		return sessionID
	}
	return sessionID[:12]
}

func sessionHasSuccessfulGitCommit(messages []opencode.Message) bool {
	callCommands := make(map[string]string)

	for _, message := range messages {
		for _, part := range message.Parts {
			command := ""
			if part.State != nil {
				if raw, ok := part.State.Input["command"]; ok {
					if cmd, ok := raw.(string); ok {
						command = cmd
					}
				}
			}

			if command != "" && part.CallID != "" {
				callCommands[part.CallID] = command
			}
			if command == "" && part.CallID != "" {
				command = callCommands[part.CallID]
			}

			if !containsGitCommitCommand(command) {
				continue
			}

			if part.State == nil {
				continue
			}

			status := strings.ToLower(strings.TrimSpace(part.State.Status))
			if status != "" && status != "completed" {
				continue
			}

			if exitCode, ok := parseExitCode(part.State.Metadata); ok && exitCode != 0 {
				continue
			}

			outputLower := strings.ToLower(part.State.Output)
			if strings.Contains(outputLower, "nothing to commit") || strings.Contains(outputLower, "no changes added to commit") {
				continue
			}

			return true
		}
	}

	return false
}

func containsGitCommitCommand(command string) bool {
	if strings.TrimSpace(command) == "" {
		return false
	}
	return strings.Contains(strings.ToLower(command), "git commit")
}

func parseExitCode(metadata map[string]interface{}) (int, bool) {
	if len(metadata) == 0 {
		return 0, false
	}

	raw, ok := metadata["exit_code"]
	if !ok {
		raw, ok = metadata["exitCode"]
	}
	if !ok {
		return 0, false
	}

	switch value := raw.(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	case float64:
		return int(value), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

// findWorkspaceForIssue tries to find the workspace directory for a beads issue.
// It scans .orch/workspace/ for directories that might match the issue.
func findWorkspaceForIssue(beadsID, workspaceDir, projectDir string) string {
	if workspaceDir == "" && projectDir != "" {
		workspaceDir = filepath.Join(projectDir, ".orch", "workspace")
	}
	if workspaceDir == "" {
		// Try current directory
		cwd, _ := os.Getwd()
		workspaceDir = filepath.Join(cwd, ".orch", "workspace")
	}

	// Check if workspace directory exists
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		return ""
	}

	// Scan workspace directories for SPAWN_CONTEXT.md that references this beads ID
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		wsPath := filepath.Join(workspaceDir, entry.Name())
		spawnContext := filepath.Join(wsPath, "SPAWN_CONTEXT.md")

		// Check if SPAWN_CONTEXT.md exists and references this beads ID
		data, err := os.ReadFile(spawnContext)
		if err != nil {
			continue
		}

		// Look for beads ID in spawn context (e.g., "bd comment <id>" or "--issue <id>")
		if strings.Contains(string(data), beadsID) {
			return wsPath
		}
	}

	return ""
}

// ProcessCompletion verifies and closes a single completed agent.
// It runs the same verification as `orch complete` and closes the beads issue.
// Uses the escalation model to determine whether to auto-complete:
//   - EscalationNone/Info/Review: Auto-complete (issue closed)
//   - EscalationBlock/Failed: Do not auto-complete (issue remains open)
func (d *Daemon) ProcessCompletion(agent CompletedAgent, config CompletionConfig) CompletionResult {
	result := CompletionResult{
		BeadsID: agent.BeadsID,
	}

	if agent.Source == completionSourceIdleTracked {
		if err := ensureAutoPhaseComplete(agent, config.ProjectDir); err != nil {
			result.Error = fmt.Errorf("failed to backfill Phase: Complete comment: %w", err)
			result.Escalation = verify.EscalationFailed
			return result
		}
	}

	// Determine tier from workspace if available
	tier := ""
	if agent.WorkspacePath != "" {
		tier = verify.ReadTierFromWorkspace(agent.WorkspacePath)
	}

	// Run full verification
	verificationResult, err := verify.VerifyCompletionFull(
		agent.BeadsID,
		agent.WorkspacePath,
		config.ProjectDir,
		tier,
		config.ServerURL,
	)
	if err != nil {
		result.Error = fmt.Errorf("verification failed: %w", err)
		result.Verification = verificationResult
		result.Escalation = verify.EscalationFailed
		return result
	}

	result.Verification = verificationResult

	// Try to parse synthesis for escalation signals
	var synthesis *verify.Synthesis
	if agent.WorkspacePath != "" {
		synthesis, _ = verify.ParseSynthesis(agent.WorkspacePath)
	}

	// Determine escalation level
	escalation := verify.DetermineEscalationFromCompletion(
		verificationResult,
		synthesis,
		agent.BeadsID,
		agent.WorkspacePath,
		config.ProjectDir,
	)
	result.Escalation = escalation

	// Check if verification passed
	if !verificationResult.Passed {
		result.Error = fmt.Errorf("verification failed: %s", strings.Join(verificationResult.Errors, "; "))
		// Emit verify_failed attention signal for visibility
		emitVerifyFailedSignal(agent, verificationResult.GatesFailed, verificationResult.Errors)

		return result
	}

	// Check if escalation allows auto-completion
	if !escalation.ShouldAutoComplete() {
		reason := verify.ExplainEscalation(verify.EscalationInput{
			VerificationPassed:  verificationResult.Passed,
			VerificationErrors:  verificationResult.Errors,
			NeedsVisualApproval: escalation == verify.EscalationBlock,
		})
		result.Error = fmt.Errorf("requires human review: %s", reason.Reason)
		// Emit verify_failed attention signal (escalation blocked auto-completion)
		emitVerifyFailedSignal(agent, []string{"escalation_blocked"}, []string{result.Error.Error()})

		return result
	}

	// Build close reason from phase summary
	closeReason := "Phase: Complete"
	if agent.PhaseSummary != "" {
		closeReason = fmt.Sprintf("Phase: Complete - %s", agent.PhaseSummary)
	}

	// Close the issue (unless dry run), using force to bypass bd's redundant
	// Phase: Complete gate since we already verified it via ListCompletedAgents
	if !config.DryRun {
		if err := deleteCompletedAgentSession(agent, config.ServerURL); err != nil {
			result.Error = fmt.Errorf("failed to delete session: %w", err)
			return result
		}

		if err := verify.CloseIssueForce(agent.BeadsID, closeReason, true); err != nil {
			result.Error = fmt.Errorf("failed to close issue: %w", err)
			return result
		}
	}

	result.Processed = true
	result.CloseReason = closeReason
	return result
}

// CompletionOnce runs a single iteration of the completion loop.
// It finds all Phase: Complete agents and processes their completions.
func (d *Daemon) CompletionOnce(config CompletionConfig) (*CompletionLoopResult, error) {
	result := &CompletionLoopResult{}

	// Find completed agents (explicit Phase: Complete or idle tracked-session detection)
	completed, err := d.ListCompletedAgents(config)
	if err != nil {
		return nil, fmt.Errorf("failed to list completed agents: %w", err)
	}

	if len(completed) == 0 {
		return result, nil
	}

	// Process each completed agent
	logger := events.NewDefaultLogger()

	for _, agent := range completed {
		if config.Verbose {
			fmt.Printf("  Processing completion for %s: %s\n", agent.BeadsID, agent.Title)
		}

		compResult := d.ProcessCompletion(agent, config)
		result.Processed = append(result.Processed, compResult)

		if compResult.Error != nil {
			result.Errors = append(result.Errors, compResult.Error)
			if config.Verbose {
				fmt.Printf("    Error: %v (escalation=%s)\n", compResult.Error, compResult.Escalation)
			}
		} else if compResult.Processed {
			// Log successful auto-completion with escalation level
			if err := logger.LogAutoCompletedWithEscalation(agent.BeadsID, compResult.CloseReason, compResult.Escalation.String()); err != nil && config.Verbose {
				fmt.Printf("    Warning: failed to log completion event: %v\n", err)
			}
			if config.Verbose {
				fmt.Printf("    Closed: %s (escalation=%s)\n", compResult.CloseReason, compResult.Escalation)
			}
		}
	}

	return result, nil
}

// CompletionLoop runs the completion processing loop continuously.
// It polls for Phase: Complete agents and closes their issues.
// The loop continues until the context is cancelled.
func (d *Daemon) CompletionLoop(ctx context.Context, config CompletionConfig) error {
	ticker := time.NewTicker(config.PollInterval)
	defer ticker.Stop()

	// Run immediately on first call
	if _, err := d.CompletionOnce(config); err != nil && config.Verbose {
		fmt.Printf("Completion loop error: %v\n", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := d.CompletionOnce(config); err != nil && config.Verbose {
				fmt.Printf("Completion loop error: %v\n", err)
			}
		}
	}
}

// PreviewCompletions shows what agents would be completed without actually closing them.
func (d *Daemon) PreviewCompletions(config CompletionConfig) ([]CompletedAgent, error) {
	return d.ListCompletedAgents(config)
}

// emitVerifyFailedSignal stores a verification failure signal for attention system visibility.
// This enables the Work Graph to show issues stuck in "verification purgatory".
func emitVerifyFailedSignal(agent CompletedAgent, failedGates, errors []string) {
	entry := attention.VerifyFailedEntry{
		BeadsID:      agent.BeadsID,
		Title:        agent.Title,
		FailedGates:  failedGates,
		Errors:       errors,
		PhaseSummary: agent.PhaseSummary,
	}

	// Store the failure - errors are logged but don't block completion processing
	if err := attention.StoreVerifyFailed(entry); err != nil {
		// Log but don't fail - this is observability, not critical path
		fmt.Printf("Warning: failed to store verify_failed signal for %s: %v\n", agent.BeadsID, err)
	}
}

func ensureAutoPhaseComplete(agent CompletedAgent, projectDir string) error {
	comments, err := verify.GetCommentsWithDir(agent.BeadsID, projectDir)
	if err != nil {
		return fmt.Errorf("failed to fetch comments: %w", err)
	}

	phase := verify.ParsePhaseFromComments(comments)
	if phase.Found && strings.EqualFold(phase.Phase, "Complete") {
		return nil
	}

	summary := strings.TrimSpace(agent.PhaseSummary)
	if summary == "" {
		summary = "Auto-detected by daemon: tracked session idle"
	}

	comment := fmt.Sprintf("Phase: Complete - %s", summary)

	err = beads.Do(projectDir, func(client *beads.Client) error {
		return client.AddComment(agent.BeadsID, "", comment)
	}, beads.WithAutoReconnect(3))
	if err == nil {
		return nil
	}

	cli := beads.NewCLIClient(
		beads.WithWorkDir(projectDir),
		beads.WithEnv(append(os.Environ(), "BEADS_NO_DAEMON=1")),
	)
	if cliErr := cli.AddComment(agent.BeadsID, "", comment); cliErr != nil {
		return fmt.Errorf("rpc=%v; cli=%w", err, cliErr)
	}

	return nil
}

func deleteCompletedAgentSession(agent CompletedAgent, serverURL string) error {
	sessionID := strings.TrimSpace(agent.SessionID)
	if sessionID == "" && agent.WorkspacePath != "" {
		sessionID = spawn.ReadSessionID(agent.WorkspacePath)
	}
	if sessionID == "" {
		return nil
	}

	if strings.TrimSpace(serverURL) == "" {
		serverURL = opencode.DefaultServerURL
	}

	client := opencode.NewClient(serverURL)
	if err := client.DeleteSession(sessionID); err != nil {
		// Another cleanup path may have already deleted this session.
		if strings.Contains(err.Error(), "status 404") {
			return nil
		}
		return fmt.Errorf("session %s: %w", shortSessionID(sessionID), err)
	}

	return nil
}
