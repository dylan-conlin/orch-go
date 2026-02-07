// Package main provides the abandon command for abandoning stuck agents.
// Extracted from main.go as part of the main.go refactoring.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/process"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	statedb "github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	// Abandon command flags
	abandonReason  string
	abandonWorkdir string
)

var abandonCmd = &cobra.Command{
	Use:   "abandon [beads-id]",
	Short: "Abandon a stuck or frozen agent",
	Long: `Abandon an agent and kill its tmux window.

Use this command for stuck or frozen agents that are not responding.
The agent's beads issue is NOT closed - you can restart work with 'orch work'.

The session transcript is automatically exported to SESSION_LOG.md in the agent's
workspace before deletion. This preserves conversation history for post-mortem analysis
to help debug why agents get stuck.

When --reason is provided, a FAILURE_REPORT.md is also generated in the workspace
documenting what went wrong and recommendations for retry.

For cross-project abandonment, use --workdir to specify the target project directory
where the beads issue lives.

Examples:
  orch-go abandon proj-123                                      # Abandon agent in current project
  orch-go abandon proj-123 --reason "Out of context"            # Abandon with failure report
  orch-go abandon proj-123 --reason "Stuck in loop"             # Document the failure
  orch-go abandon kb-cli-123 --workdir ~/projects/kb-cli        # Abandon agent in another project`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runAbandon(beadsID, abandonReason, abandonWorkdir)
	},
}

func init() {
	abandonCmd.Flags().StringVar(&abandonReason, "reason", "", "Reason for abandonment (generates FAILURE_REPORT.md)")
	abandonCmd.Flags().StringVar(&abandonWorkdir, "workdir", "", "Target project directory (for cross-project abandonment)")
	abandonCmd.Flags().StringVar(&abandonWorkdir, "project", "", "Alias for --workdir")
	abandonCmd.Flags().MarkHidden("project")
}

// abandonContext holds all resolved state needed during the abandon workflow.
// Populated incrementally by each phase of runAbandon.
type abandonContext struct {
	BeadsID     string
	Reason      string
	ProjectDir  string
	IsUntracked bool

	// Resolved from beads — nil for untracked agents.
	Issue *verify.Issue

	// Agent identity — resolved from state DB, then discovery fallbacks.
	AgentName     string
	SessionID     string
	WorkspacePath string
	WindowInfo    *tmux.WindowInfo

	// State DB handle — opened once, reused across phases.
	DB *statedb.DB
}

// runAbandon orchestrates the abandon workflow in discrete phases:
//  1. Resolve project directory
//  2. Verify the beads target (tracked vs untracked)
//  3. Resolve agent identity (state DB + fallback discovery)
//  4. Clean up agent resources (docker, tmux, session, process)
//  5. Generate failure report (if reason provided)
//  6. Log abandonment event + telemetry
//  7. Update tracking state (registry, beads, state DB)
//  8. Print summary
func runAbandon(beadsID, reason, workdir string) error {
	ctx := &abandonContext{
		BeadsID: beadsID,
		Reason:  reason,
	}

	// Phase 1: Resolve project directory
	projectDir, err := resolveAbandonProjectDir(workdir)
	if err != nil {
		return err
	}
	ctx.ProjectDir = projectDir

	// Phase 2: Verify the beads target
	if err := verifyAbandonTarget(ctx); err != nil {
		return err
	}

	// Phase 3: Resolve agent identity (state DB + discovery fallbacks)
	resolveAgentIdentity(ctx)

	// Phase 4: Clean up agent resources
	cleanupAgentResources(ctx)

	// Phase 5: Generate failure report (if reason provided)
	generateFailureReport(ctx)

	// Phase 6: Log abandonment event + telemetry
	logAbandonmentEvent(ctx)

	// Phase 7: Update tracking state (registry, beads, state DB)
	updateAbandonTrackingState(ctx)

	// Phase 8: Print summary
	printAbandonSummary(ctx)

	return nil
}

// resolveAbandonProjectDir resolves the project directory for abandon operations.
// If workdir is provided, validates it and configures beads.DefaultDir for cross-project access.
// Otherwise, uses the current working directory.
func resolveAbandonProjectDir(workdir string) (string, error) {
	if workdir != "" {
		projectDir, err := filepath.Abs(workdir)
		if err != nil {
			return "", fmt.Errorf("failed to resolve workdir path: %w", err)
		}
		if stat, err := os.Stat(projectDir); err != nil {
			return "", fmt.Errorf("workdir does not exist: %s", projectDir)
		} else if !stat.IsDir() {
			return "", fmt.Errorf("workdir is not a directory: %s", projectDir)
		}
		// Set DefaultDir for beads client to find the correct socket
		beads.DefaultDir = projectDir
		return projectDir, nil
	}

	projectDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	return projectDir, nil
}

// verifyAbandonTarget checks whether the beads ID is tracked or untracked,
// and for tracked agents, verifies the issue exists and isn't already closed.
func verifyAbandonTarget(ctx *abandonContext) error {
	ctx.IsUntracked = isUntrackedBeadsID(ctx.BeadsID)

	if ctx.IsUntracked {
		fmt.Printf("Note: %s is an untracked agent (no beads issue)\n", ctx.BeadsID)
		return nil
	}

	issue, err := verify.GetIssue(ctx.BeadsID)
	if err != nil {
		return crossProjectIssueError(ctx.BeadsID, ctx.ProjectDir, err)
	}

	if issue.Status == "closed" {
		return fmt.Errorf("issue %s is already closed - nothing to abandon", ctx.BeadsID)
	}

	ctx.Issue = issue
	return nil
}

// crossProjectIssueError provides a helpful error message when a beads issue
// lookup fails, hinting at cross-project usage if the ID prefix doesn't match.
func crossProjectIssueError(beadsID, projectDir string, err error) error {
	projectName := filepath.Base(projectDir)
	parts := strings.Split(beadsID, "-")
	issuePrefix := parts[0]
	if len(parts) > 1 {
		issuePrefix = strings.Join(parts[:len(parts)-1], "-")
	}
	if issuePrefix != projectName {
		return fmt.Errorf("failed to get beads issue %s: %w\n\nHint: The issue ID suggests it belongs to project '%s', but you're in '%s'.\nTry: orch abandon %s --workdir ~/path/to/%s",
			beadsID, err, issuePrefix, projectName, beadsID, issuePrefix)
	}
	return fmt.Errorf("failed to get beads issue: %w", err)
}

// resolveAgentIdentity populates the abandonContext with agent identity info
// (workspace path, agent name, session ID, tmux window) using a layered
// resolution strategy: state DB first, then filesystem/tmux/API discovery fallbacks.
func resolveAgentIdentity(ctx *abandonContext) {
	// Layer 1: State DB lookup
	resolveFromStateDB(ctx)

	// Layer 2: Workspace discovery fallback
	if ctx.WorkspacePath == "" || ctx.AgentName == "" {
		wPath, aName := findWorkspaceByBeadsID(ctx.ProjectDir, ctx.BeadsID)
		if ctx.WorkspacePath == "" {
			ctx.WorkspacePath = wPath
		}
		if ctx.AgentName == "" {
			ctx.AgentName = aName
		}
	}

	// Layer 3: Tmux window discovery fallback
	if ctx.WindowInfo == nil {
		ctx.WindowInfo = discoverTmuxWindow(ctx.BeadsID, ctx.WorkspacePath, ctx.AgentName)
	}

	// Layer 4: OpenCode session discovery fallback
	if ctx.SessionID == "" {
		ctx.SessionID = discoverOpenCodeSession(ctx.BeadsID, ctx.ProjectDir)
	}

	// Coherence validation: cross-check workspace .session_id against discovered session
	validateSessionCoherence(ctx)

	if ctx.AgentName == "" {
		ctx.AgentName = ctx.BeadsID // Use beads ID as fallback
	}

	// Report what we found
	if ctx.WindowInfo != nil {
		fmt.Printf("Found tmux window: %s\n", ctx.WindowInfo.Target)
	}
	if ctx.SessionID != "" {
		fmt.Printf("Found OpenCode session: %s\n", ctx.SessionID[:12])
	}
	if ctx.WindowInfo == nil && ctx.SessionID == "" {
		fmt.Printf("Note: No active tmux window or OpenCode session found for %s\n", ctx.BeadsID)
		fmt.Printf("The agent may have already exited.\n")
	}
}

// resolveFromStateDB attempts to populate agent identity from the state database.
// Rejects stale rows (abandoned/completed) in favor of live discovery.
func resolveFromStateDB(ctx *abandonContext) {
	db, dbErr := statedb.OpenDefault()
	if dbErr != nil || db == nil {
		return
	}
	ctx.DB = db

	dbAgent, err := db.GetAgentByBeadsID(ctx.BeadsID)
	if err != nil || dbAgent == nil {
		return
	}

	// Coherence check: reject stale state DB rows
	if dbAgent.IsAbandoned || dbAgent.IsCompleted {
		fmt.Printf("State DB has stale row for %s (abandoned=%v, completed=%v) — using live discovery\n",
			ctx.BeadsID, dbAgent.IsAbandoned, dbAgent.IsCompleted)
		return
	}

	fmt.Printf("Found agent in state DB: %s (mode: %s)\n", dbAgent.WorkspaceName, dbAgent.Mode)
	ctx.AgentName = dbAgent.WorkspaceName
	ctx.SessionID = dbAgent.SessionID

	if (dbAgent.Mode == "claude" || dbAgent.Mode == "docker") && dbAgent.TmuxWindow != "" {
		ctx.WindowInfo = &tmux.WindowInfo{
			Target: dbAgent.TmuxWindow,
			Name:   dbAgent.TmuxWindow,
		}
	}

	if dbAgent.ProjectDir != "" {
		ctx.WorkspacePath = filepath.Join(dbAgent.ProjectDir, ".orch", "workspace", dbAgent.WorkspaceName)
	}
}

// discoverTmuxWindow searches tmux sessions for a window matching the beads ID.
// For orchestrator workspaces, also searches by workspace name since orchestrator
// windows don't contain beads IDs.
func discoverTmuxWindow(beadsID, workspacePath, agentName string) *tmux.WindowInfo {
	// Try searching by beads ID first (for worker sessions)
	sessions, _ := tmux.ListWorkersSessions()
	for _, sess := range sessions {
		window, err := tmux.FindWindowByBeadsID(sess, beadsID)
		if err == nil && window != nil {
			return window
		}
	}

	// Orchestrator windows only contain workspace names, not beads IDs
	if workspacePath != "" && isOrchestratorWorkspace(workspacePath) {
		window, _, err := tmux.FindWindowByWorkspaceNameAllSessions(agentName)
		if err == nil && window != nil {
			return window
		}
	}

	return nil
}

// discoverOpenCodeSession searches OpenCode API for a session matching the beads ID.
func discoverOpenCodeSession(beadsID, projectDir string) string {
	return discoverOpenCodeSessionWithClient(opencode.NewClient(serverURL), beadsID, projectDir)
}

func discoverOpenCodeSessionWithClient(client opencode.ClientInterface, beadsID, projectDir string) string {
	allSessions, _ := client.ListSessions(projectDir)
	for _, s := range allSessions {
		if strings.Contains(s.Title, beadsID) || extractBeadsIDFromTitle(s.Title) == beadsID {
			return s.ID
		}
	}
	return ""
}

// validateSessionCoherence cross-checks the discovered session ID against the
// workspace's .session_id file. The workspace file is considered more authoritative.
func validateSessionCoherence(ctx *abandonContext) {
	if ctx.SessionID == "" || ctx.WorkspacePath == "" {
		return
	}

	wsSessionFile := filepath.Join(ctx.WorkspacePath, ".session_id")
	wsSessionBytes, err := os.ReadFile(wsSessionFile)
	if err != nil {
		return
	}

	wsSessionID := strings.TrimSpace(string(wsSessionBytes))
	if wsSessionID == "" || wsSessionID == ctx.SessionID {
		return
	}

	fmt.Fprintf(os.Stderr, "WARNING: Coherence mismatch detected!\n")
	fmt.Fprintf(os.Stderr, "  Workspace %s has session_id=%s\n", filepath.Base(ctx.WorkspacePath), wsSessionID[:min(12, len(wsSessionID))])
	fmt.Fprintf(os.Stderr, "  But discovered session %s from OpenCode\n", ctx.SessionID[:min(12, len(ctx.SessionID))])
	fmt.Fprintf(os.Stderr, "  Using workspace's session_id (more authoritative for this workspace)\n")
	ctx.SessionID = wsSessionID
}

// cleanupAgentResources kills the agent's runtime resources in the correct order:
// docker container → tmux window → export transcript → delete session → terminate process.
func cleanupAgentResources(ctx *abandonContext) {
	cleanupAgentResourcesWithClient(opencode.NewClient(serverURL), ctx)
}

func cleanupAgentResourcesWithClient(client opencode.ClientInterface, ctx *abandonContext) {

	// Docker container must be cleaned up before tmux (tmux kill might orphan it)
	cleanupDockerContainer(ctx.WorkspacePath)

	// Kill tmux window
	if ctx.WindowInfo != nil {
		fmt.Printf("Killing tmux window: %s\n", ctx.WindowInfo.Target)
		if err := tmux.KillWindow(ctx.WindowInfo.Target); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to kill tmux window: %v\n", err)
		}
	}

	// Export session transcript before deleting the session
	exportSessionTranscript(client, ctx.SessionID, ctx.WorkspacePath)

	// Delete OpenCode session to remove from `orch status`
	deleteOpenCodeSession(client, ctx.SessionID)

	// Terminate orphaned OpenCode process
	terminateAgentProcess(ctx.WorkspacePath)
}

// cleanupDockerContainer removes the Docker container associated with a workspace.
func cleanupDockerContainer(workspacePath string) {
	if workspacePath == "" {
		return
	}
	containerName := spawn.ReadContainerID(workspacePath)
	if containerName == "" {
		return
	}
	if err := spawn.CleanupDockerContainer(containerName); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to clean up Docker container %s: %v\n", containerName, err)
	} else {
		fmt.Printf("Cleaned up Docker container: %s\n", containerName)
	}
}

// exportSessionTranscript saves the session transcript to SESSION_LOG.md for post-mortem analysis.
func exportSessionTranscript(client opencode.ClientInterface, sessionID, workspacePath string) {
	if sessionID == "" || workspacePath == "" {
		return
	}
	transcript, err := client.ExportSessionTranscript(sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to export session transcript: %v\n", err)
		return
	}
	if transcript == "" {
		return
	}
	transcriptPath := filepath.Join(workspacePath, "SESSION_LOG.md")
	if err := os.WriteFile(transcriptPath, []byte(transcript), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write session transcript: %v\n", err)
	} else {
		fmt.Printf("Exported session transcript: %s\n", transcriptPath)
	}
}

// deleteOpenCodeSession removes the session from OpenCode so it no longer appears in `orch status`.
func deleteOpenCodeSession(client opencode.ClientInterface, sessionID string) {
	if sessionID == "" {
		return
	}
	fmt.Printf("Deleting OpenCode session: %s\n", sessionID[:12])
	if err := client.DeleteSession(sessionID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to delete OpenCode session: %v\n", err)
	} else {
		fmt.Printf("Deleted OpenCode session\n")
	}
}

// terminateAgentProcess kills the OpenCode process to prevent orphaned processes.
func terminateAgentProcess(workspacePath string) {
	if workspacePath == "" {
		return
	}
	pid := spawn.ReadProcessID(workspacePath)
	if pid > 0 {
		process.Terminate(pid, "opencode")
	}
}

// generateFailureReport writes FAILURE_REPORT.md to the workspace when a reason is provided.
func generateFailureReport(ctx *abandonContext) {
	if ctx.Reason == "" || ctx.WorkspacePath == "" {
		return
	}

	if err := spawn.EnsureFailureReportTemplate(ctx.ProjectDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to ensure failure report template: %v\n", err)
	}

	// For untracked agents, issue is nil so use empty title
	issueTitle := ""
	if ctx.Issue != nil {
		issueTitle = ctx.Issue.Title
	}

	reportPath, err := spawn.WriteFailureReport(ctx.WorkspacePath, ctx.AgentName, ctx.BeadsID, ctx.Reason, issueTitle)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write failure report: %v\n", err)
	} else {
		fmt.Printf("Generated failure report: %s\n", reportPath)
	}
}

// logAbandonmentEvent logs both the structured event and telemetry data for the abandonment.
func logAbandonmentEvent(ctx *abandonContext) {
	logger := events.NewLogger(events.DefaultLogPath())

	// Log structured event
	logStructuredEvent(logger, ctx)

	// Log telemetry for model performance tracking
	logAbandonTelemetry(logger, ctx)
}

// logStructuredEvent writes the agent.abandoned event with all available context.
func logStructuredEvent(logger *events.Logger, ctx *abandonContext) {
	eventData := map[string]interface{}{
		"beads_id":  ctx.BeadsID,
		"agent_id":  ctx.AgentName,
		"untracked": ctx.IsUntracked,
	}
	if ctx.WindowInfo != nil {
		eventData["window_id"] = ctx.WindowInfo.ID
		eventData["window_target"] = ctx.WindowInfo.Target
	}
	if ctx.SessionID != "" {
		eventData["session_id"] = ctx.SessionID
	}
	if ctx.WorkspacePath != "" {
		eventData["workspace_path"] = ctx.WorkspacePath
	}
	if ctx.Reason != "" {
		eventData["reason"] = ctx.Reason
	}
	event := events.Event{
		Type:      "agent.abandoned",
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}
}

// logAbandonTelemetry collects duration, token usage, and skill info from the workspace,
// then logs the telemetry event for model performance tracking.
func logAbandonTelemetry(logger *events.Logger, ctx *abandonContext) {
	abandonedData := events.AgentAbandonedData{
		BeadsID:   ctx.BeadsID,
		Workspace: ctx.AgentName,
		Reason:    ctx.Reason,
		Outcome:   "abandoned",
	}

	if ctx.WorkspacePath != "" {
		collectTelemetryFromWorkspace(ctx.WorkspacePath, &abandonedData)
	}

	if err := logger.LogAgentAbandoned(abandonedData); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log abandonment event: %v\n", err)
	}
}

// collectTelemetryFromWorkspace reads duration, token usage, and skill from workspace files.
func collectTelemetryFromWorkspace(workspacePath string, data *events.AgentAbandonedData) {
	collectTelemetryFromWorkspaceWithClient(opencode.NewClient("http://127.0.0.1:4096"), workspacePath, data)
}

func collectTelemetryFromWorkspaceWithClient(client opencode.ClientInterface, workspacePath string, data *events.AgentAbandonedData) {
	// Read spawn time for duration calculation
	spawnTimeFile := filepath.Join(workspacePath, ".spawn_time")
	if spawnTimeBytes, err := os.ReadFile(spawnTimeFile); err == nil {
		spawnTimeStr := strings.TrimSpace(string(spawnTimeBytes))
		if spawnTime, err := time.Parse(time.RFC3339, spawnTimeStr); err == nil {
			data.DurationSeconds = int(time.Since(spawnTime).Seconds())
		}
	}

	// Read session ID and get token usage
	sessionIDFile := filepath.Join(workspacePath, ".session_id")
	if sessionIDBytes, err := os.ReadFile(sessionIDFile); err == nil {
		sessionIDStr := strings.TrimSpace(string(sessionIDBytes))
		if sessionIDStr != "" {
			if tokenStats, err := client.GetSessionTokens(sessionIDStr); err == nil && tokenStats != nil {
				data.TokensInput = tokenStats.InputTokens
				data.TokensOutput = tokenStats.OutputTokens
			}
		}
	}

	// Extract skill from SPAWN_CONTEXT.md
	data.Skill = extractSkillFromSpawnContext(workspacePath)
}

// extractSkillFromSpawnContext parses the skill name from the SKILL GUIDANCE header.
func extractSkillFromSpawnContext(workspacePath string) string {
	spawnContextFile := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	contextBytes, err := os.ReadFile(spawnContextFile)
	if err != nil {
		return ""
	}

	contextStr := string(contextBytes)
	if !strings.Contains(contextStr, "## SKILL GUIDANCE") {
		return ""
	}

	for _, line := range strings.Split(contextStr, "\n") {
		if !strings.Contains(line, "## SKILL GUIDANCE") {
			continue
		}
		// Format: "## SKILL GUIDANCE (feature-impl)"
		start := strings.Index(line, "(")
		if start == -1 {
			break
		}
		end := strings.Index(line[start:], ")")
		if end == -1 {
			break
		}
		return line[start+1 : start+end]
	}
	return ""
}

// updateAbandonTrackingState updates all tracking systems to reflect the abandonment:
// orchestrator session registry, beads issue status, triage labels, and state DB.
func updateAbandonTrackingState(ctx *abandonContext) {
	// Update orchestrator session registry
	updateOrchestratorRegistry(ctx.WorkspacePath, ctx.AgentName)

	// Reset beads status and remove triage label (tracked agents only)
	if !ctx.IsUntracked {
		resetBeadsStatus(ctx.BeadsID)
		removeTriageLabel(ctx.BeadsID)
	}

	// Record abandonment in state database
	if err := statedb.RecordAbandon(ctx.AgentName, ctx.BeadsID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to record abandonment in state db: %v\n", err)
	}

	// Close state DB if we opened it
	if ctx.DB != nil {
		ctx.DB.Close()
	}
}

// updateOrchestratorRegistry marks the session as abandoned in the orchestrator registry.
func updateOrchestratorRegistry(workspacePath, agentName string) {
	if workspacePath == "" || !isOrchestratorWorkspace(workspacePath) {
		return
	}

	registry := session.NewRegistry("")
	if err := registry.Update(agentName, func(s *session.OrchestratorSession) {
		s.Status = "abandoned"
	}); err != nil {
		if err == session.ErrSessionNotFound {
			fmt.Printf("Note: Session %s was not in registry (legacy workspace)\n", agentName)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: failed to update session status in registry: %v\n", err)
		}
	} else {
		fmt.Printf("Updated session registry: status → abandoned\n")
	}
}

// resetBeadsStatus changes the beads issue from in_progress back to open so respawn works.
func resetBeadsStatus(beadsID string) {
	if err := verify.UpdateIssueStatus(beadsID, "open"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to reset beads status: %v\n", err)
	} else {
		fmt.Printf("Reset beads status: in_progress → open\n")
	}
}

// removeTriageLabel removes the triage:ready label to prevent daemon from
// immediately respawning abandoned work.
func removeTriageLabel(beadsID string) {
	if err := verify.RemoveTriageReadyLabel(beadsID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to remove triage:ready label: %v\n", err)
	} else {
		fmt.Printf("Removed triage:ready label (use 'bd label %s triage:ready' to re-queue)\n", beadsID)
	}
}

// printAbandonSummary outputs the final status of the abandon operation.
func printAbandonSummary(ctx *abandonContext) {
	fmt.Printf("Abandoned agent: %s\n", ctx.AgentName)
	fmt.Printf("  Beads ID: %s\n", ctx.BeadsID)
	if ctx.Reason != "" {
		fmt.Printf("  Reason: %s\n", ctx.Reason)
	}
	if ctx.IsUntracked {
		fmt.Println("  (Untracked agent - no beads issue to respawn)")
	} else {
		fmt.Printf("  Use 'orch work %s' to restart work on this issue\n", ctx.BeadsID)
	}
}
