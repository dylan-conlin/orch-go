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
	"github.com/dylan-conlin/orch-go/pkg/registry"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
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
}

func runAbandon(beadsID, reason, workdir string) error {
	// Strategy: Use registry to determine mode and route abandonment
	// Fall back to direct discovery if not in registry

	// Determine project directory - use --workdir if provided, otherwise current directory
	var projectDir string
	var err error
	if workdir != "" {
		projectDir, err = filepath.Abs(workdir)
		if err != nil {
			return fmt.Errorf("failed to resolve workdir path: %w", err)
		}
		// Verify directory exists
		if stat, err := os.Stat(projectDir); err != nil {
			return fmt.Errorf("workdir does not exist: %s", projectDir)
		} else if !stat.IsDir() {
			return fmt.Errorf("workdir is not a directory: %s", projectDir)
		}
		// Set DefaultDir for beads client to find the correct socket
		beads.DefaultDir = projectDir
	} else {
		projectDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Check if this is an untracked agent (no beads issue exists)
	isUntracked := isUntrackedBeadsID(beadsID)

	// For tracked agents, verify the beads issue exists
	var issue *verify.Issue
	if !isUntracked {
		var err error
		issue, err = verify.GetIssue(beadsID)
		if err != nil {
			// Provide helpful error message for cross-project issues
			projectName := filepath.Base(projectDir)
			issuePrefix := strings.Split(beadsID, "-")[0]
			if len(strings.Split(beadsID, "-")) > 1 {
				issuePrefix = strings.Join(strings.Split(beadsID, "-")[:len(strings.Split(beadsID, "-"))-1], "-")
			}
			if issuePrefix != projectName {
				return fmt.Errorf("failed to get beads issue %s: %w\n\nHint: The issue ID suggests it belongs to project '%s', but you're in '%s'.\nTry: orch abandon %s --workdir ~/path/to/%s", beadsID, err, issuePrefix, projectName, beadsID, issuePrefix)
			}
			return fmt.Errorf("failed to get beads issue: %w", err)
		}

		if issue.Status == "closed" {
			return fmt.Errorf("issue %s is already closed - nothing to abandon", beadsID)
		}
	} else {
		fmt.Printf("Note: %s is an untracked agent (no beads issue)\n", beadsID)
	}

	client := opencode.NewClient(serverURL)

	// Try registry first (primary source of truth)
	agentReg, _ := registry.New("")
	var agent *registry.Agent
	if agentReg != nil {
		agent = agentReg.Find(beadsID)
	}

	var windowInfo *tmux.WindowInfo
	var sessionID string
	var workspacePath, agentName string

	if agent != nil {
		fmt.Printf("Found agent in registry: %s (mode: %s)\n", agent.ID, agent.Mode)
		agentName = agent.ID
		sessionID = agent.SessionID
		if agent.Mode == registry.ModeTmux && agent.TmuxWindow != "" {
			windowInfo = &tmux.WindowInfo{
				Target: agent.TmuxWindow,
				Name:   agent.TmuxWindow,
			}
		}
		// Resolve workspace path from project dir and agent ID
		if agent.ProjectDir != "" {
			workspacePath = filepath.Join(agent.ProjectDir, ".orch", "workspace", agent.ID)
		}
	}

	// Discovery fallback if registry didn't give us everything
	if windowInfo == nil {
		// Try searching by beads ID first (for worker sessions)
		sessions, _ := tmux.ListWorkersSessions()
		for _, session := range sessions {
			window, err := tmux.FindWindowByBeadsID(session, beadsID)
			if err == nil && window != nil {
				windowInfo = window
				break
			}
		}

		// If beads ID search failed and we have a workspace, check if it's an orchestrator
		// Orchestrator windows only contain workspace names, not beads IDs
		if windowInfo == nil && workspacePath != "" && isOrchestratorWorkspace(workspacePath) {
			// Search by workspace name for orchestrator sessions
			window, _, err := tmux.FindWindowByWorkspaceNameAllSessions(agentName)
			if err == nil && window != nil {
				windowInfo = window
			}
		}
	}

	if sessionID == "" {
		// Check for OpenCode session
		allSessions, _ := client.ListSessions(projectDir)
		for _, s := range allSessions {
			if strings.Contains(s.Title, beadsID) || extractBeadsIDFromTitle(s.Title) == beadsID {
				sessionID = s.ID
				break
			}
		}
	}

	if workspacePath == "" || agentName == "" {
		// Find workspace for logging
		wPath, aName := findWorkspaceByBeadsID(projectDir, beadsID)
		if workspacePath == "" {
			workspacePath = wPath
		}
		if agentName == "" {
			agentName = aName
		}
	}

	if agentName == "" {
		agentName = beadsID // Use beads ID as fallback
	}

	// Report what we found
	if windowInfo != nil {
		fmt.Printf("Found tmux window: %s\n", windowInfo.Target)
	}
	if sessionID != "" {
		fmt.Printf("Found OpenCode session: %s\n", sessionID[:12])
	}

	// If neither found, warn but still allow abandonment
	if windowInfo == nil && sessionID == "" {
		fmt.Printf("Note: No active tmux window or OpenCode session found for %s\n", beadsID)
		fmt.Printf("The agent may have already exited.\n")
	}

	// Optionally kill the tmux window if it exists
	if windowInfo != nil {
		fmt.Printf("Killing tmux window: %s\n", windowInfo.Target)
		if err := tmux.KillWindow(windowInfo.Target); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to kill tmux window: %v\n", err)
		}
	}

	// Export session transcript before deletion (for post-mortem analysis)
	if sessionID != "" && workspacePath != "" {
		transcript, err := client.ExportSessionTranscript(sessionID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to export session transcript: %v\n", err)
		} else if transcript != "" {
			transcriptPath := filepath.Join(workspacePath, "SESSION_LOG.md")
			if err := os.WriteFile(transcriptPath, []byte(transcript), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to write session transcript: %v\n", err)
			} else {
				fmt.Printf("Exported session transcript: %s\n", transcriptPath)
			}
		}
	}

	// Delete the OpenCode session if it exists
	// This prevents abandoned agents from appearing in `orch status`
	if sessionID != "" {
		fmt.Printf("Deleting OpenCode session: %s\n", sessionID[:12])
		if err := client.DeleteSession(sessionID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to delete OpenCode session: %v\n", err)
		} else {
			fmt.Printf("Deleted OpenCode session\n")
		}
	}

	// Generate FAILURE_REPORT.md if reason is provided
	if reason != "" && workspacePath != "" {
		// Ensure the failure report template exists in the project
		if err := spawn.EnsureFailureReportTemplate(projectDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to ensure failure report template: %v\n", err)
		}

		// Generate and write the failure report
		// For untracked agents, issue is nil so use empty title
		issueTitle := ""
		if issue != nil {
			issueTitle = issue.Title
		}
		reportPath, err := spawn.WriteFailureReport(workspacePath, agentName, beadsID, reason, issueTitle)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write failure report: %v\n", err)
		} else {
			fmt.Printf("Generated failure report: %s\n", reportPath)
		}
	}

	// Log the abandonment
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"beads_id":  beadsID,
		"agent_id":  agentName,
		"untracked": isUntracked,
	}
	if windowInfo != nil {
		eventData["window_id"] = windowInfo.ID
		eventData["window_target"] = windowInfo.Target
	}
	if sessionID != "" {
		eventData["session_id"] = sessionID
	}
	if workspacePath != "" {
		eventData["workspace_path"] = workspacePath
	}
	if reason != "" {
		eventData["reason"] = reason
	}
	event := events.Event{
		Type:      "agent.abandoned",
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Update orchestrator session registry if this is an orchestrator workspace
	// This ensures `orch status` shows correct session status
	if workspacePath != "" && isOrchestratorWorkspace(workspacePath) {
		registry := session.NewRegistry("")
		if err := registry.Update(agentName, func(s *session.OrchestratorSession) {
			s.Status = "abandoned"
		}); err != nil {
			if err == session.ErrSessionNotFound {
				// Session wasn't in registry - likely a legacy workspace
				fmt.Printf("Note: Session %s was not in registry (legacy workspace)\n", agentName)
			} else {
				fmt.Fprintf(os.Stderr, "Warning: failed to update session status in registry: %v\n", err)
			}
		} else {
			fmt.Printf("Updated session registry: status → abandoned\n")
		}
	}

	// Reset beads status to open so respawn works without manual bd update
	// Skip for untracked agents (no beads issue to update)
	if !isUntracked {
		if err := verify.UpdateIssueStatus(beadsID, "open"); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to reset beads status: %v\n", err)
		} else {
			fmt.Printf("Reset beads status: in_progress → open\n")
		}
	}

	// Log abandonment with telemetry for model performance tracking
	logger = events.NewLogger(events.DefaultLogPath())
	abandonedData := events.AgentAbandonedData{
		BeadsID:   beadsID,
		Workspace: agentName,
		Reason:    reason,
		Outcome:   "abandoned",
	}

	// Collect telemetry (duration and tokens) if workspace is available
	if workspacePath != "" {
		// Read spawn time
		spawnTimeFile := filepath.Join(workspacePath, ".spawn_time")
		spawnTimeBytes, readErr := os.ReadFile(spawnTimeFile)
		if readErr == nil {
			spawnTimeStr := strings.TrimSpace(string(spawnTimeBytes))
			spawnTime, parseErr := time.Parse(time.RFC3339, spawnTimeStr)
			if parseErr == nil {
				abandonedData.DurationSeconds = int(time.Since(spawnTime).Seconds())
			}
		}

		// Read session ID and get token usage
		sessionIDFile := filepath.Join(workspacePath, ".session_id")
		sessionIDBytes, readErr := os.ReadFile(sessionIDFile)
		if readErr == nil {
			sessionIDStr := strings.TrimSpace(string(sessionIDBytes))
			if sessionIDStr != "" {
				client := opencode.NewClient("http://127.0.0.1:4096")
				tokenStats, tokErr := client.GetSessionTokens(sessionIDStr)
				if tokErr == nil && tokenStats != nil {
					abandonedData.TokensInput = tokenStats.InputTokens
					abandonedData.TokensOutput = tokenStats.OutputTokens
				}
			}
		}

		// Try to read skill from SPAWN_CONTEXT.md
		spawnContextFile := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
		contextBytes, readErr := os.ReadFile(spawnContextFile)
		if readErr == nil {
			// Simple extraction of skill from "SKILL GUIDANCE (skill-name)"
			contextStr := string(contextBytes)
			if strings.Contains(contextStr, "## SKILL GUIDANCE") {
				// Try to extract skill name from the section header
				lines := strings.Split(contextStr, "\n")
				for _, line := range lines {
					if strings.Contains(line, "## SKILL GUIDANCE") {
						// Format: "## SKILL GUIDANCE (feature-impl)"
						if start := strings.Index(line, "("); start != -1 {
							if end := strings.Index(line[start:], ")"); end != -1 {
								abandonedData.Skill = line[start+1 : start+end]
							}
						}
						break
					}
				}
			}
		}
	}

	logErr := logger.LogAgentAbandoned(abandonedData)
	if logErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log abandonment event: %v\n", logErr)
	}

	fmt.Printf("Abandoned agent: %s\n", agentName)
	fmt.Printf("  Beads ID: %s\n", beadsID)
	if reason != "" {
		fmt.Printf("  Reason: %s\n", reason)
	}
	if isUntracked {
		fmt.Println("  (Untracked agent - no beads issue to respawn)")
	} else {
		fmt.Printf("  Use 'orch work %s' to restart work on this issue\n", beadsID)
	}

	return nil
}
