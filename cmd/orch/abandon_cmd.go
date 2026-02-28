// Package main provides the abandon command for abandoning stuck agents.
// Extracted from main.go as part of the main.go refactoring.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/agent"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
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
	// --- Phase 1: Resolve project directory ---

	var projectDir string
	var err error
	if workdir != "" {
		projectDir, err = filepath.Abs(workdir)
		if err != nil {
			return fmt.Errorf("failed to resolve workdir path: %w", err)
		}
		if stat, err := os.Stat(projectDir); err != nil {
			return fmt.Errorf("workdir does not exist: %s", projectDir)
		} else if !stat.IsDir() {
			return fmt.Errorf("workdir is not a directory: %s", projectDir)
		}
		beads.DefaultDir = projectDir
	} else {
		projectDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// --- Phase 2: Validate beads issue (tracked agents only) ---

	isUntracked := isUntrackedBeadsID(beadsID)

	var issue *verify.Issue
	if !isUntracked {
		issue, err = verify.GetIssue(beadsID)
		if err != nil && workdir == "" {
			// Local lookup failed and no --workdir specified.
			// Auto-resolve by searching registered kb projects.
			resolvedDir, beadsIssue := resolveProjectDirForBeadsID(beadsID)
			if resolvedDir != "" && beadsIssue != nil {
				fmt.Printf("Auto-resolved cross-project issue: %s in %s\n", beadsID, resolvedDir)
				projectDir = resolvedDir
				beads.DefaultDir = resolvedDir
				issue = &verify.Issue{
					ID:        beadsIssue.ID,
					Title:     beadsIssue.Title,
					Status:    beadsIssue.Status,
					IssueType: beadsIssue.IssueType,
					Labels:    beadsIssue.Labels,
				}
				err = nil
			}
		}
		if err != nil {
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

	// --- Phase 3: Discover agent resources ---

	client := opencode.NewClient(serverURL)

	var sessionID string
	var workspacePath, agentName string

	// Look up workspace by beads ID
	wPath, aName := findWorkspaceByBeadsID(projectDir, beadsID)
	if wPath != "" {
		workspacePath = wPath
		agentName = aName
		sessionID = spawn.ReadSessionID(wPath)
		if sessionID != "" {
			fmt.Printf("Found agent workspace: %s (session: %s)\n", agentName, shortID(sessionID))
		} else {
			fmt.Printf("Found agent workspace: %s\n", agentName)
		}
	}

	if sessionID == "" {
		// Fall back to OpenCode session search
		allSessions, _ := client.ListSessions(projectDir)
		for _, s := range allSessions {
			if strings.Contains(s.Title, beadsID) || extractBeadsIDFromTitle(s.Title) == beadsID {
				sessionID = s.ID
				break
			}
		}
	}

	if workspacePath == "" || agentName == "" {
		wPath, aName := findWorkspaceByBeadsID(projectDir, beadsID)
		if workspacePath == "" {
			workspacePath = wPath
		}
		if agentName == "" {
			agentName = aName
		}
	}

	if agentName == "" {
		agentName = beadsID
	}

	if sessionID != "" {
		fmt.Printf("Found OpenCode session: %s\n", shortID(sessionID))
	}

	if sessionID == "" {
		fmt.Printf("Note: No active OpenCode session found for %s\n", beadsID)
		fmt.Printf("The agent may have already exited.\n")
	}

	// --- Phase 4: Export session transcript BEFORE lifecycle transition ---
	// Transcript export must happen before the session is deleted.

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

	// --- Phase 5: Execute lifecycle transition ---

	if !isUntracked {
		// Tracked agents: use LifecycleManager for correct state transition.
		// This handles the critical orch:agent label removal (ghost agent fix),
		// assignee clearing, status reset, and all cleanup effects.
		agentRef := agent.AgentRef{
			BeadsID:       beadsID,
			WorkspaceName: agentName,
			WorkspacePath: workspacePath,
			SessionID:     sessionID,
			ProjectDir:    projectDir,
		}

		lm := buildLifecycleManager(projectDir, serverURL, agentName, beadsID)
		event, err := lm.Abandon(agentRef, reason)
		if err != nil {
			return fmt.Errorf("abandon transition failed: %w", err)
		}

		// Report lifecycle effects
		for _, e := range event.Effects {
			if e.Success {
				switch e.Operation {
				case "remove_label":
					fmt.Printf("Removed orch:agent label\n")
				case "clear_assignee":
					fmt.Printf("Cleared assignee\n")
				case "update_status":
					fmt.Printf("Reset beads status: in_progress → open\n")
				case "kill_window":
					fmt.Printf("Killed tmux window\n")
				case "delete_session":
					fmt.Printf("Deleted OpenCode session\n")
				case "write_failure_report":
					fmt.Printf("Generated failure report\n")
				}
			}
		}

		// Report warnings (non-critical failures)
		for _, w := range event.Warnings {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
		}

		// Report critical failures
		if !event.Success {
			for _, e := range event.Effects {
				if e.Critical && !e.Success {
					fmt.Fprintf(os.Stderr, "Error: %s/%s failed: %v\n", e.Subsystem, e.Operation, e.Error)
				}
			}
		}
	} else {
		// Untracked agents: no beads operations needed, just cleanup.
		// Kill tmux window if found
		window, _, _ := tmux.FindWindowByWorkspaceNameAllSessions(agentName)
		if window != nil {
			fmt.Printf("Killing tmux window: %s (%s)\n", window.Name, window.ID)
			if err := tmux.KillWindowByID(window.ID); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to kill tmux window: %v\n", err)
			}
		}

		// Delete OpenCode session
		if sessionID != "" {
			fmt.Printf("Deleting OpenCode session: %s\n", shortID(sessionID))
			if err := client.DeleteSession(sessionID); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to delete OpenCode session: %v\n", err)
			} else {
				fmt.Printf("Deleted OpenCode session\n")
			}
		}

		// Generate failure report
		if reason != "" && workspacePath != "" {
			_ = spawn.EnsureFailureReportTemplate(projectDir)
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

		// Log abandonment event
		logger := events.NewLogger(events.DefaultLogPath())
		if err := logger.Log(events.Event{
			Type:      "agent.abandoned",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"beads_id":  beadsID,
				"workspace": agentName,
				"reason":    reason,
				"untracked": true,
			},
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
		}
	}

	// --- Phase 6: Telemetry (model performance tracking) ---

	logAbandonmentTelemetry(beadsID, agentName, reason, workspacePath)

	// --- Phase 7: Summary ---

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

// logAbandonmentTelemetry collects duration, tokens, and skill info, then logs
// an agent.abandoned telemetry event for model performance tracking.
func logAbandonmentTelemetry(beadsID, agentName, reason, workspacePath string) {
	logger := events.NewLogger(events.DefaultLogPath())
	abandonedData := events.AgentAbandonedData{
		BeadsID:   beadsID,
		Workspace: agentName,
		Reason:    reason,
		Outcome:   "abandoned",
	}

	if workspacePath != "" {
		manifest := spawn.ReadAgentManifestWithFallback(workspacePath)
		if spawnTime := manifest.ParseSpawnTime(); !spawnTime.IsZero() {
			abandonedData.DurationSeconds = int(time.Since(spawnTime).Seconds())
		}

		sessionIDStr := spawn.ReadSessionID(workspacePath)
		if sessionIDStr != "" {
			oc := opencode.NewClient("http://127.0.0.1:4096")
			tokenStats, tokErr := oc.GetSessionTokens(sessionIDStr)
			if tokErr == nil && tokenStats != nil {
				abandonedData.TokensInput = tokenStats.InputTokens
				abandonedData.TokensOutput = tokenStats.OutputTokens
			}
		}

		spawnContextFile := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
		contextBytes, readErr := os.ReadFile(spawnContextFile)
		if readErr == nil {
			contextStr := string(contextBytes)
			if strings.Contains(contextStr, "## SKILL GUIDANCE") {
				lines := strings.Split(contextStr, "\n")
				for _, line := range lines {
					if strings.Contains(line, "## SKILL GUIDANCE") {
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

	if err := logger.LogAgentAbandoned(abandonedData); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log abandonment telemetry: %v\n", err)
	}
}
