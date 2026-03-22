// Package main provides the lifecycle transition phase of the completion pipeline.
// Extracted from complete_pipeline.go for cohesion: close reason, pre-lifecycle exports,
// LifecycleManager execution, and post-lifecycle operations.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/activity"
	"github.com/dylan-conlin/orch-go/pkg/agent"
	"github.com/dylan-conlin/orch-go/pkg/artifactsync"
	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// executeLifecycleTransition handles close reason determination, pre-lifecycle exports,
// LifecycleManager.Complete execution, and all post-lifecycle operations (triage label,
// daemon signal, auto-rebuild, changelog, telemetry, event logging, cache invalidation).
// Returns true if LifecycleManager handled tmux cleanup (so deferred cleanup can be skipped).
func executeLifecycleTransition(target CompletionTarget, outcome VerificationOutcome, advisoryResults AdvisoryResults) (lifecycleCleanedUp bool, err error) {
	// Determine close reason
	reason := completeReason
	if reason == "" {
		if target.BeadsID != "" {
			status, _ := verify.GetPhaseStatus(target.BeadsID, target.BeadsProjectDir)
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

	// NOTE: Auto-create of implementation issues for architect completions has
	// been moved to executeVerificationGates (complete_verification.go) so the
	// architect_handoff gate can verify the issue exists before completion proceeds.

	// --- Pre-lifecycle operations (need session/workspace alive) ---

	// Fallback: if verification didn't extract skill (e.g., missing SKILL GUIDANCE
	// in SPAWN_CONTEXT.md), try the agent manifest (written at spawn time).
	// Must happen before lifecycle transition which archives the workspace.
	if outcome.SkillName == "" && target.WorkspacePath != "" {
		manifest := spawn.ReadAgentManifestWithFallback(target.WorkspacePath)
		if manifest.Skill != "" {
			outcome.SkillName = manifest.Skill
		}
	}

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
	if target.WorkspacePath != "" && target.WorkProjectDir != "" {
		accretionData = collectAccretionDelta(target.WorkProjectDir, target.WorkspacePath)
	}

	// Capture change-scope classification before archival (needs workspace for git baseline)
	if target.WorkspacePath != "" && target.WorkProjectDir != "" && !target.IsOrchestratorSession {
		manifest := spawn.ReadAgentManifestWithFallback(target.WorkspacePath)
		if manifest.GitBaseline != "" {
			scopes, changedFiles := artifactsync.CaptureChangeScopes(target.WorkProjectDir, manifest.GitBaseline)
			if len(scopes) > 0 {
				driftEvent := artifactsync.DriftEvent{
					BeadsID:      target.BeadsID,
					Skill:        outcome.SkillName,
					ChangeScopes: scopes,
					FilesChanged: changedFiles,
					CommitRange:  manifest.GitBaseline + "..HEAD",
				}
				if err := artifactsync.LogDriftEvent(artifactsync.DefaultDriftLogPath(), driftEvent); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log artifact drift event: %v\n", err)
				} else {
					fmt.Printf("Artifact drift: %s\n", strings.Join(scopes, ", "))
				}
			}
		}
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
		if err := exportOrchestratorTranscript(target.WorkspacePath, target.WorkProjectDir, target.AgentName); err != nil {
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

	if !target.IsClosed || target.IsOrchestratorSession {
		if target.IsOrchestratorSession {
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

	// Belt-and-suspenders: always ensure orch:agent label is removed on completion.
	// Catches two failure modes:
	// 1. IsClosed=true: lifecycle manager was skipped entirely (issue closed by
	//    another path like daemon verification, and on_close hook's label removal
	//    failed silently due to JSONL lock contention or similar)
	// 2. IsClosed=false but lifecycle manager's non-critical remove_label effect
	//    failed silently
	// Removing a non-existent label is a no-op in beads.
	if target.BeadsID != "" && !target.IsOrchestratorSession {
		if err := verify.RemoveOrchAgentLabel(target.BeadsID, target.BeadsProjectDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove orch:agent label: %v\n", err)
		}
	}

	if target.IsOrchestratorSession {
		fmt.Printf("Completed orchestrator session: %s\n", target.AgentName)
	}
	fmt.Printf("Reason: %s\n", reason)

	// Post-lifecycle operations

	// Remove triage:ready label on successful completion
	if !target.IsClosed && target.BeadsID != "" {
		if err := verify.RemoveTriageReadyLabel(target.BeadsID, target.BeadsProjectDir); err != nil {
			// Non-critical
		}

		// Remove comprehension:pending — this completion is now comprehended
		if err := daemon.RemoveComprehensionPendingInDir(target.BeadsID, target.BeadsProjectDir); err != nil {
			// Non-critical: label may not exist (e.g., manual completion, not daemon-queued)
		}

		// Signal human verification to daemon
		if err := daemon.WriteVerificationSignal(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to signal human verification to daemon: %v\n", err)
		}
	}

	if completeNoArchive && target.WorkspacePath != "" {
		fmt.Println("Skipped workspace archival (--no-archive)")
	}

	// Auto-rebuild if agent committed Go changes (scoped to agent baseline)
	rebuildStep := events.PipelineStepTiming{Name: "auto_rebuild"}
	if hasAgentGoChanges(target.WorkspacePath, target.WorkProjectDir) {
		fmt.Println("Detected Go file changes in recent commits")
		rebuildStart := time.Now()
		if err := runAutoRebuild(target.WorkProjectDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: auto-rebuild failed: %v\n", err)
		} else {
			fmt.Println("Auto-rebuild completed: make install")
			if restarted, err := restartOrchServe(target.WorkProjectDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to restart orch serve: %v\n", err)
			} else if restarted {
				fmt.Println("Restarted orch serve")
			}
		}
		rebuildStep.DurationMs = int(time.Since(rebuildStart).Milliseconds())

		// Check for new CLI commands that may need skill documentation
		newCommands := detectNewCLICommands(target.WorkProjectDir)
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
	} else {
		rebuildStep.Skipped = true
		rebuildStep.SkipReason = "no_go_changes"
	}
	advisoryResults.PipelineTiming = append(advisoryResults.PipelineTiming, rebuildStep)

	// Check for notable changelog entries
	if !completeNoChangelogCheck {
		var agentSkill string
		if target.WorkspacePath != "" {
			agentSkill, _ = verify.ExtractSkillNameFromSpawnContext(target.WorkspacePath)
		}

		notableEntries := detectNotableChangelogEntries(target.WorkProjectDir, agentSkill)
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
		Untracked:          false,
		Orchestrator:       target.IsOrchestratorSession,
		VerificationPassed: outcome.Passed,
		Skill:              outcome.SkillName,
		VerificationLevel:  outcome.Result.VerifyLevel,
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
	if len(advisoryResults.PipelineTiming) > 0 {
		completedData.PipelineTiming = advisoryResults.PipelineTiming
		completedData.PipelineTotalMs = advisoryResults.PipelineTotalMs
	}
	if err := logger.LogAgentCompleted(completedData); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Log accretion delta metrics (collected pre-lifecycle above)
	if accretionData != nil {
		accretionData.BeadsID = target.BeadsID
		accretionData.Workspace = target.AgentName
		accretionData.Skill = outcome.SkillName
		// Populate model from agent manifest for model-comparative analysis (HE-08)
		if target.WorkspacePath != "" {
			m := spawn.ReadAgentManifestWithFallback(target.WorkspacePath)
			accretionData.Model = m.Model
		}

		if err := logger.LogAccretionDelta(*accretionData); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log accretion delta: %v\n", err)
		}
	}

	// Invalidate orch serve cache
	invalidateServeCache()

	return lifecycleCleanedUp, nil
}
