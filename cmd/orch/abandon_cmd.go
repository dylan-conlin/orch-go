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
	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/identity"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

// activeAgentThreshold is the duration within which a phase comment is considered
// "recent" — indicating the agent may still be actively running. Agents that reported
// a non-Complete phase within this window trigger a warning before abandon.
const activeAgentThreshold = 30 * time.Minute

var (
	// Abandon command flags
	abandonReason  string
	abandonWorkdir string
	abandonForce   bool
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

If the agent has recent phase activity (within 30 minutes), a warning is shown
and --force is required to proceed. This prevents accidentally killing cross-project
agents that appear as 'phantom' locally but are actively running elsewhere.

Examples:
  orch-go abandon proj-123                                      # Abandon agent in current project
  orch-go abandon proj-123 --reason "Out of context"            # Abandon with failure report
  orch-go abandon proj-123 --reason "Stuck in loop"             # Document the failure
  orch-go abandon kb-cli-123 --workdir ~/projects/kb-cli        # Abandon agent in another project
  orch-go abandon proj-123 --force                              # Force abandon despite recent activity`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		return runAbandon(beadsID, abandonReason, abandonWorkdir, abandonForce)
	},
}

func init() {
	abandonCmd.Flags().StringVar(&abandonReason, "reason", "", "Reason for abandonment (generates FAILURE_REPORT.md)")
	abandonCmd.Flags().StringVar(&abandonWorkdir, "workdir", "", "Target project directory (for cross-project abandonment)")
	abandonCmd.Flags().BoolVar(&abandonForce, "force", false, "Force abandon even if agent has recent activity")
}

func runAbandon(beadsID, reason, workdir string, force bool) error {
	// --- Phase 1: Resolve project directory ---

	projectDir, err := identity.ResolveProject(beadsID, workdir)
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	// --- Phase 2: Validate beads issue ---

	issue, err := verify.GetIssue(beadsID, projectDir)
	if err != nil {
		return fmt.Errorf("failed to get beads issue %s in %s: %w", beadsID, filepath.Base(projectDir), err)
	}

	if issue.Status == "closed" {
		return fmt.Errorf("issue %s is already closed - nothing to abandon", beadsID)
	}

	// --- Phase 2b: Activity check (protect recently-active agents) ---
	// Cross-project agents may appear as "phantom" locally but be actively running
	// in another tmux session. Check for recent phase comments before killing.

	if !force {
		if err := checkRecentActivity(beadsID, projectDir); err != nil {
			return err
		}
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

	// Use LifecycleManager for correct state transition.
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

	// --- Phase 5b: Clear daemon spawn tracker ---
	// The daemon's disk-persisted spawn cache (~/.orch/spawn_cache.json) tracks
	// recently-spawned issues to prevent duplicate spawns. Without clearing this,
	// abandoned issues stay blocked for up to 6 hours (the cache TTL).
	if cachePath := daemon.DefaultSpawnCachePath(); cachePath != "" {
		tracker := daemon.NewSpawnedIssueTrackerWithFile(cachePath)
		tracker.Unmark(beadsID)
		fmt.Printf("Cleared daemon spawn tracker\n")
	}

	// --- Phase 6: Telemetry (model performance tracking) ---

	logAbandonmentTelemetry(beadsID, agentName, reason, workspacePath)

	// --- Phase 7: Summary ---

	fmt.Printf("Abandoned agent: %s\n", agentName)
	fmt.Printf("  Beads ID: %s\n", beadsID)
	if reason != "" {
		fmt.Printf("  Reason: %s\n", reason)
	}
	fmt.Printf("  Use 'orch work %s' to restart work on this issue\n", beadsID)

	return nil
}

// checkRecentActivity checks if the agent appears alive using phase-based liveness.
// Returns an error (blocking abandon) if the agent is actively running.
// Uses VerifyLiveness for grace period + phase checks, then falls back to
// checkPhaseRecency for the 30-minute recency window specific to abandon.
// Comment fetch failures are non-blocking (best effort).
func checkRecentActivity(beadsID, projectDir string) error {
	comments, err := verify.GetComments(beadsID, projectDir)
	if err != nil {
		// Can't fetch comments — don't block the abandon.
		// This is best-effort protection, not a hard gate.
		return nil
	}

	// Phase-based liveness catches recently-spawned agents (grace period)
	liveness := verify.VerifyLiveness(verify.LivenessInput{
		Comments: comments,
		Now:      time.Now(),
	})
	if liveness.IsAlive() && liveness.Reason == verify.ReasonRecentlySpawned {
		return fmt.Errorf(
			"agent %s was recently spawned and may not have reported its first phase yet\n\n"+
				"Use --force to abandon anyway: orch abandon %s --force",
			beadsID, beadsID,
		)
	}

	// Fall back to recency check for the 30-minute activity window
	phase := verify.ParsePhaseFromComments(comments)
	return checkPhaseRecency(beadsID, phase, time.Now())
}

// checkPhaseRecency determines if a parsed phase status indicates recent activity.
// Returns an error if the phase is non-Complete and was reported within activeAgentThreshold.
// Pure function for testability — no I/O.
func checkPhaseRecency(beadsID string, phase verify.PhaseStatus, now time.Time) error {
	if !phase.Found {
		return nil
	}

	// Phase: Complete means the agent finished — safe to abandon (cleanup)
	if strings.EqualFold(phase.Phase, "Complete") {
		return nil
	}

	// No timestamp means we can't determine recency — allow abandon
	if phase.PhaseReportedAt == nil {
		return nil
	}

	elapsed := now.Sub(*phase.PhaseReportedAt)
	if elapsed < activeAgentThreshold {
		return fmt.Errorf(
			"agent %s appears to be actively running\n"+
				"  Last phase: %s (reported %s ago)\n"+
				"  Summary: %s\n\n"+
				"This agent may be running in another tmux session (cross-project phantom).\n"+
				"Use --force to abandon anyway: orch abandon %s --force",
			beadsID,
			phase.Phase,
			formatDuration(elapsed),
			phase.Summary,
			beadsID,
		)
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
