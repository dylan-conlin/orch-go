// Package main provides the lifecycle transition phase of the completion pipeline.
// Extracted from complete_pipeline.go for cohesion: close reason, pre-lifecycle exports,
// LifecycleManager execution, and post-lifecycle operations.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylan-conlin/orch-go/pkg/activity"
	"github.com/dylan-conlin/orch-go/pkg/agent"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// executeLifecycleTransition handles close reason determination, pre-lifecycle exports,
// LifecycleManager.Complete execution, and all post-lifecycle operations (triage label,
// daemon signal, auto-rebuild, changelog, telemetry, event logging, cache invalidation).
// Returns true if LifecycleManager handled tmux cleanup (so deferred cleanup can be skipped).
func executeLifecycleTransition(target CompletionTarget, outcome VerificationOutcome, _ AdvisoryResults) (lifecycleCleanedUp bool, err error) {
	// Determine close reason
	reason := completeReason
	if reason == "" {
		if !target.IsUntracked && target.BeadsID != "" {
			status, _ := verify.GetPhaseStatus(target.BeadsID)
			if status.Summary != "" {
				reason = status.Summary
			}
		}
		if reason == "" {
			if target.IsOrchestratorSession {
				reason = "Orchestrator session completed"
			} else {
				reason = "Completed via orch complete"
			}
		}
	}

	// Auto-create implementation issue for architect completions
	if !target.IsUntracked && !target.IsOrchestratorSession && target.WorkspacePath != "" {
		maybeAutoCreateImplementationIssue(outcome.SkillName, target.BeadsID, target.WorkspacePath)
	}

	// --- Pre-lifecycle operations (need session/workspace alive) ---

	// Collect telemetry BEFORE lifecycle transition, because lm.Complete()
	// archives the workspace (moves it to archived/), making the manifest
	// unreadable at the original path.
	var durationSecs, tokensIn, tokensOut int
	var telemetryOutcome string
	if target.WorkspacePath != "" {
		durationSecs, tokensIn, tokensOut, telemetryOutcome = collectCompletionTelemetry(target.WorkspacePath, completeForce, outcome.Passed)
	}

	// Collect accretion delta before archival (same reason as telemetry above)
	var accretionData *events.AccretionDeltaData
	if target.WorkspacePath != "" && target.BeadsProjectDir != "" {
		accretionData = collectAccretionDelta(target.BeadsProjectDir, target.WorkspacePath)
	}

	// Export activity to ACTIVITY.json for archival
	if target.WorkspacePath != "" && !target.IsOrchestratorSession {
		sid := spawn.ReadSessionID(target.WorkspacePath)
		if sid != "" {
			if activityPath, err := activity.ExportToWorkspace(sid, target.WorkspacePath, serverURL); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to export activity: %v\n", err)
			} else if activityPath != "" {
				fmt.Printf("Exported activity: %s\n", filepath.Base(activityPath))
			}
		}
	}

	// For orchestrator sessions, export transcript before lifecycle transition
	if target.WorkspacePath != "" && target.IsOrchestratorSession {
		if err := exportOrchestratorTranscript(target.WorkspacePath, target.BeadsProjectDir, target.AgentName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to export orchestrator transcript: %v\n", err)
		}
	}

	// --- Execute lifecycle transition via LifecycleManager ---
	sessionID := spawn.ReadSessionID(target.WorkspacePath)

	lifecycleWorkspacePath := target.WorkspacePath
	if completeNoArchive {
		lifecycleWorkspacePath = ""
	}

	agentRef := agent.AgentRef{
		BeadsID:       target.BeadsID,
		WorkspaceName: target.AgentName,
		WorkspacePath: lifecycleWorkspacePath,
		SessionID:     sessionID,
		ProjectDir:    target.BeadsProjectDir,
	}

	if !target.IsClosed || target.IsOrchestratorSession || target.IsUntracked {
		if target.IsOrchestratorSession || target.IsUntracked {
			agentRef.BeadsID = ""
		}

		lm := buildLifecycleManager(target.BeadsProjectDir, serverURL, target.AgentName, target.BeadsID)
		event, err := lm.Complete(agentRef, reason)
		if err != nil {
			return false, fmt.Errorf("complete transition failed: %w", err)
		}

		// Report lifecycle effects
		for _, e := range event.Effects {
			if e.Success {
				switch e.Operation {
				case "close_issue":
					fmt.Printf("Closed beads issue: %s\n", target.BeadsID)
				case "remove_label":
					fmt.Printf("Removed orch:agent label\n")
				case "kill_window":
					fmt.Printf("Killed tmux window\n")
				case "delete_session":
					fmt.Printf("Deleted OpenCode session: %s\n", shortID(sessionID))
				case "archive":
					fmt.Printf("Archived workspace: %s\n", target.AgentName)
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
					return false, fmt.Errorf("failed to close issue: %v", e.Error)
				}
			}
		}

		lifecycleCleanedUp = true
	}

	if target.IsOrchestratorSession {
		fmt.Printf("Completed orchestrator session: %s\n", target.AgentName)
	} else if target.IsUntracked {
		fmt.Printf("Cleaned up untracked agent: %s\n", target.Identifier)
	}
	fmt.Printf("Reason: %s\n", reason)

	// Post-lifecycle operations

	// Remove triage:ready label on successful completion
	if !target.IsClosed && !target.IsUntracked && target.BeadsID != "" {
		if err := verify.RemoveTriageReadyLabel(target.BeadsID); err != nil {
			// Non-critical
		}
	}

	if completeNoArchive && target.WorkspacePath != "" {
		fmt.Println("Skipped workspace archival (--no-archive)")
	}

	// Auto-rebuild if agent committed Go changes
	if hasGoChangesInRecentCommits(target.BeadsProjectDir) {
		fmt.Println("Detected Go file changes in recent commits")
		if err := runAutoRebuild(target.BeadsProjectDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: auto-rebuild failed: %v\n", err)
		} else {
			fmt.Println("Auto-rebuild completed: make install")
			if restarted, err := restartOrchServe(target.BeadsProjectDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to restart orch serve: %v\n", err)
			} else if restarted {
				fmt.Println("Restarted orch serve")
			}
		}

		// Check for new CLI commands that may need skill documentation
		newCommands := detectNewCLICommands(target.BeadsProjectDir)
		if len(newCommands) > 0 {
			newlyTracked := trackDocDebt(newCommands)

			fmt.Println()
			fmt.Println("┌─────────────────────────────────────────────────────────────┐")
			fmt.Println("│  📚 NEW CLI COMMANDS DETECTED                               │")
			fmt.Println("├─────────────────────────────────────────────────────────────┤")
			for _, cmd := range newCommands {
				fmt.Printf("│  • %s\n", cmd)
			}
			fmt.Println("├─────────────────────────────────────────────────────────────┤")
			fmt.Println("│  Consider updating skill documentation:                     │")
			fmt.Println("│  - ~/.claude/skills/meta/orchestrator/SKILL.md              │")
			fmt.Println("│  - docs/orch-commands-reference.md                          │")
			fmt.Println("├─────────────────────────────────────────────────────────────┤")
			if newlyTracked > 0 {
				fmt.Printf("│  📝 Added %d command(s) to doc debt tracker                  │\n", newlyTracked)
			}
			fmt.Println("│  Run 'orch doctor --docs' to see all undocumented commands  │")
			fmt.Println("└─────────────────────────────────────────────────────────────┘")
		}
	}

	// Check for notable changelog entries
	if !completeNoChangelogCheck {
		var agentSkill string
		if target.WorkspacePath != "" {
			agentSkill, _ = verify.ExtractSkillNameFromSpawnContext(target.WorkspacePath)
		}

		notableEntries := detectNotableChangelogEntries(target.BeadsProjectDir, agentSkill)
		if len(notableEntries) > 0 {
			fmt.Println()
			fmt.Println("┌─────────────────────────────────────────────────────────────┐")
			fmt.Println("│  ⚠️  NOTABLE ECOSYSTEM CHANGES DETECTED                      │")
			fmt.Println("├─────────────────────────────────────────────────────────────┤")
			for _, entry := range notableEntries {
				if len(entry) > 55 {
					fmt.Printf("│  %s\n", entry[:55])
					fmt.Printf("│    %s\n", entry[55:])
				} else {
					fmt.Printf("│  %s\n", entry)
				}
			}
			fmt.Println("├─────────────────────────────────────────────────────────────┤")
			fmt.Println("│  Review recent changes that may affect agent behavior       │")
			fmt.Println("│  Run: orch changelog --days 3                               │")
			fmt.Println("└─────────────────────────────────────────────────────────────┘")
		}
	}

	// Log the completion with verification metadata (telemetry collected pre-lifecycle above)
	logger := events.NewLogger(events.DefaultLogPath())
	completedData := events.AgentCompletedData{
		Reason:             reason,
		Forced:             completeForce,
		Untracked:          target.IsUntracked,
		Orchestrator:       target.IsOrchestratorSession,
		VerificationPassed: outcome.Passed,
		Skill:              outcome.SkillName,
		DurationSeconds:    durationSecs,
		TokensInput:        tokensIn,
		TokensOutput:       tokensOut,
		Outcome:            telemetryOutcome,
	}
	if target.BeadsID != "" {
		completedData.BeadsID = target.BeadsID
	}
	if target.AgentName != "" {
		completedData.Workspace = target.AgentName
	}
	if completeForce && len(outcome.GatesFailed) > 0 {
		completedData.GatesBypassed = outcome.GatesFailed
	}
	if completeForce && completeReason != "" {
		completedData.ForceReason = completeReason
	}
	if err := logger.LogAgentCompleted(completedData); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Log accretion delta metrics (collected pre-lifecycle above)
	if accretionData != nil {
		accretionData.BeadsID = target.BeadsID
		accretionData.Workspace = target.AgentName
		accretionData.Skill = outcome.SkillName

		if err := logger.LogAccretionDelta(*accretionData); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log accretion delta: %v\n", err)
		}
	}

	// Invalidate orch serve cache
	invalidateServeCache()

	return lifecycleCleanedUp, nil
}
