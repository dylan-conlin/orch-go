// Package main provides the CLI entry point for orch-go.
// This file contains the periodic task scheduler extracted from runDaemonLoop.
// It runs all periodic maintenance tasks (reflection, cleanup, recovery, etc.)
// in a single function and handles logging/event emission.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/notify"
)

// periodicTasksResult holds outputs from periodic tasks needed downstream.
type periodicTasksResult struct {
	PhaseTimeoutSnapshot      *daemon.PhaseTimeoutSnapshot
	QuestionDetectionSnapshot *daemon.QuestionDetectionSnapshot
	AgreementCheckSnapshot    *daemon.AgreementCheckSnapshot
	BeadsHealthSnapshot       *daemon.BeadsHealthSnapshot
}

// runPeriodicTasks runs all periodic maintenance tasks and handles their output.
// Returns any snapshots needed by the caller for status file writing.
func runPeriodicTasks(d *daemon.Daemon, timestamp string, verbose bool, logger *events.Logger) periodicTasksResult {
	// Cache agent discovery for this cycle: recovery, orphan detection,
	// phase timeout, and question detection all call GetActiveAgents().
	// BeginCycle wraps d.Agents so all four share a single beads query.
	d.BeginCycle()
	defer d.EndCycle()

	var result periodicTasksResult

	// --- Core daemon operations ---

	// Session cleanup
	if r := d.RunPeriodicCleanup(); r != nil {
		handleCleanupResult(r, timestamp, verbose, logger)
	}

	// Stuck agent recovery
	if r := d.RunPeriodicRecovery(); r != nil {
		handleRecoveryResult(r, timestamp, verbose, logger)
		if r.Error == nil && r.ResumedCount > 0 {
			logResumeStuckDecision(logger, d.Config.Compliance.Default, r.ResumedCount)
		}
	}

	// Orphan detection
	if r := d.RunPeriodicOrphanDetection(); r != nil {
		handleOrphanDetectionResult(r, timestamp, verbose, logger)
		if r.Error == nil && r.ResetCount > 0 {
			for _, orphan := range r.Orphans {
				logResetOrphanDecision(logger, d.Config.Compliance.Default, orphan.BeadsID)
			}
		}
	}

	// Project registry refresh (picks up new projects without daemon restart)
	if r := d.RunPeriodicRegistryRefresh(); r != nil {
		handleRegistryRefreshResult(r, timestamp, verbose, logger)
	}

	// --- Agent lifecycle ---

	// Phase timeout detection
	if r := d.RunPeriodicPhaseTimeout(); r != nil {
		handlePhaseTimeoutResult(r, timestamp, verbose, logger)
		if r.Error == nil && r.UnresponsiveCount > 0 {
			for _, a := range r.Agents {
				logPhaseTimeoutDecision(logger, d.Config.Compliance.Default, a.BeadsID)
			}
		}
		if r.Error == nil {
			snapshot := r.Snapshot()
			result.PhaseTimeoutSnapshot = &snapshot
		}
	}

	// Frustration boundary handling for headless workers
	if r := d.RunPeriodicFrustrationBoundary(); r != nil {
		handleFrustrationBoundaryResult(r, timestamp, verbose, logger)
	}

	// QUESTION phase detection and notification
	if r := d.RunPeriodicQuestionDetection(); r != nil {
		handleQuestionDetectionResult(r, timestamp, verbose, logger)
		if r.Error == nil {
			snapshot := r.Snapshot()
			result.QuestionDetectionSnapshot = &snapshot
		}
	}

	// Agreement check
	if r := d.RunPeriodicAgreementCheck(); r != nil {
		handleAgreementCheckResult(r, timestamp, verbose, logger)
		if r.Error == nil && r.IssuesCreated > 0 {
			logAgreementIssueDecision(logger, d.Config.Compliance.Default, r.IssuesCreated)
		}
		if r.Error == nil {
			snapshot := r.Snapshot()
			result.AgreementCheckSnapshot = &snapshot
		}
	}

	// Verification-failed escalation (promote stuck verification-failed to triage:review)
	if r := d.RunPeriodicVerificationFailedEscalation(); r != nil {
		handleVerificationFailedEscalationResult(r, timestamp, verbose, logger)
	}

	// --- Hygiene ---

	// Beads health snapshot collection
	if r := d.RunPeriodicBeadsHealth(); r != nil {
		handleBeadsHealthResult(r, timestamp, verbose, logger)
		if r.Error == nil {
			snapshot := r.Snapshot()
			result.BeadsHealthSnapshot = &snapshot
		}
	}

	// Artifact sync
	if r := d.RunPeriodicArtifactSync(); r != nil {
		handleArtifactSyncResult(r, timestamp, verbose, logger)
	}

	// Lightweight cleanup (close stale --no-track / exploration child issues)
	if r := d.RunPeriodicLightweightCleanup(); r != nil {
		handleLightweightCleanupResult(r, timestamp, verbose, logger)
	}

	// Account capacity poll (write cache for orch status)
	if r := d.RunPeriodicCapacityPoll(); r != nil {
		handleCapacityPollResult(r, timestamp, verbose, logger)
	}

	// Random quality audit selection (weighted toward auto-completed work)
	if r := d.RunPeriodicAuditSelect(); r != nil {
		handleAuditSelectResult(r, timestamp, verbose, logger)
	}

	return result
}

func handleCleanupResult(r *daemon.CleanupResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Cleanup error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.cleanup", map[string]interface{}{
			"deleted": 0,
			"error":   r.Error.Error(),
			"message": r.Message,
		})
	} else if r.Deleted > 0 {
		fmt.Printf("[%s] Cleanup: %s\n", timestamp, r.Message)
		logDaemonEvent(logger, "daemon.cleanup", map[string]interface{}{
			"deleted": r.Deleted,
			"message": r.Message,
		})
	} else if verbose {
		fmt.Printf("[%s] Cleanup: no stale sessions found\n", timestamp)
	}
}

func handleRecoveryResult(r *daemon.RecoveryResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Recovery error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.recovery", map[string]interface{}{
			"resumed": 0,
			"skipped": r.SkippedCount,
			"error":   r.Error.Error(),
			"message": r.Message,
		})
	} else if r.ResumedCount > 0 {
		fmt.Printf("[%s] Recovery: %s\n", timestamp, r.Message)
		logDaemonEvent(logger, "daemon.recovery", map[string]interface{}{
			"resumed": r.ResumedCount,
			"skipped": r.SkippedCount,
			"message": r.Message,
		})
	} else if verbose {
		fmt.Printf("[%s] Recovery: no stuck agents found\n", timestamp)
	}
}

func handleOrphanDetectionResult(r *daemon.OrphanDetectionResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Orphan detection error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.orphan_detection", map[string]interface{}{
			"reset":   0,
			"skipped": r.SkippedCount,
			"error":   r.Error.Error(),
			"message": r.Message,
		})
	} else if r.ResetCount > 0 {
		fmt.Printf("[%s] Orphan detection: %s\n", timestamp, r.Message)
		logDaemonEvent(logger, "daemon.orphan_detection", map[string]interface{}{
			"reset":   r.ResetCount,
			"skipped": r.SkippedCount,
			"message": r.Message,
		})
	} else if verbose {
		fmt.Printf("[%s] Orphan detection: no orphans found\n", timestamp)
	}
}

func handlePhaseTimeoutResult(r *daemon.PhaseTimeoutResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Phase timeout error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.phase_timeout", map[string]interface{}{
			"unresponsive": 0,
			"error":        r.Error.Error(),
			"message":      r.Message,
		})
	} else if r.UnresponsiveCount > 0 {
		fmt.Printf("[%s] \u26a0\ufe0f  %s\n", timestamp, r.Message)
		agentIDs := make([]string, 0, len(r.Agents))
		for _, a := range r.Agents {
			agentIDs = append(agentIDs, a.BeadsID)
		}
		logDaemonEvent(logger, "daemon.phase_timeout", map[string]interface{}{
			"unresponsive": r.UnresponsiveCount,
			"agents":       agentIDs,
			"message":      r.Message,
		})
		// Desktop notification for unresponsive agents
		notifier := notify.Default()
		for _, a := range r.Agents {
			if err := notifier.AgentUnresponsive(a.BeadsID, a.IdleDuration); err != nil {
				fmt.Fprintf(os.Stderr, "[%s] Failed to send unresponsive notification: %v\n", timestamp, err)
			}
		}
	} else if verbose {
		fmt.Printf("[%s] Phase timeout: all agents responsive\n", timestamp)
	}
}

func handleQuestionDetectionResult(r *daemon.QuestionDetectionResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Question detection error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.question_detection", map[string]interface{}{
			"new":     0,
			"total":   0,
			"error":   r.Error.Error(),
			"message": r.Message,
		})
		return
	}

	if len(r.NewQuestions) > 0 {
		// Send desktop notification for each new question
		notifier := notify.Default()
		for _, q := range r.NewQuestions {
			questionText := q.Question
			if questionText == "" {
				questionText = q.Phase
			}
			fmt.Printf("[%s] Agent QUESTION: %s - %s\n", timestamp, q.BeadsID, questionText)
			if err := notifier.QuestionPending(q.BeadsID, questionText); err != nil {
				fmt.Fprintf(os.Stderr, "[%s] Failed to send question notification: %v\n", timestamp, err)
			}
		}

		agentIDs := make([]string, 0, len(r.NewQuestions))
		for _, q := range r.NewQuestions {
			agentIDs = append(agentIDs, q.BeadsID)
		}
		logDaemonEvent(logger, "daemon.question_detected", map[string]interface{}{
			"new":     len(r.NewQuestions),
			"total":   r.TotalQuestions,
			"agents":  agentIDs,
			"message": r.Message,
		})
	} else if verbose {
		fmt.Printf("[%s] Question detection: %s\n", timestamp, r.Message)
	}
}

func handleFrustrationBoundaryResult(r *daemon.FrustrationBoundaryResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Frustration boundary error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.frustration_boundary", map[string]interface{}{
			"handled": 0,
			"error":   r.Error.Error(),
			"message": r.Message,
		})
		return
	}

	if r.HandledCount > 0 {
		agentIDs := make([]string, 0, len(r.Agents))
		for _, agent := range r.Agents {
			agentIDs = append(agentIDs, agent.BeadsID)
		}
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
		logDaemonEvent(logger, "daemon.frustration_boundary", map[string]interface{}{
			"handled": r.HandledCount,
			"agents":  agentIDs,
			"message": r.Message,
		})
	} else if verbose {
		fmt.Printf("[%s] Frustration boundary: no boundary agents found\n", timestamp)
	}
}

func handleAgreementCheckResult(r *daemon.AgreementCheckResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Agreement check error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.agreement_check", map[string]interface{}{
			"total":          0,
			"passed":         0,
			"failed":         0,
			"issues_created": 0,
			"error":          r.Error.Error(),
			"message":        r.Message,
		})
	} else if r.IssuesCreated > 0 || r.Failed > 0 {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
		logDaemonEvent(logger, "daemon.agreement_check", map[string]interface{}{
			"total":          r.Total,
			"passed":         r.Passed,
			"failed":         r.Failed,
			"issues_created": r.IssuesCreated,
			"skipped":        r.Skipped,
			"message":        r.Message,
		})
	} else if verbose {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
	}
}

func handleBeadsHealthResult(r *daemon.BeadsHealthResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Beads health error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.beads_health", map[string]interface{}{
			"error":   r.Error.Error(),
			"message": r.Message,
		})
	} else {
		if verbose {
			fmt.Printf("[%s] %s\n", timestamp, r.Message)
		}
		logDaemonEvent(logger, "daemon.beads_health", map[string]interface{}{
			"open_issues":    r.OpenIssues,
			"blocked_issues": r.BlockedIssues,
			"stale_issues":   r.StaleIssues,
			"bloated_files":  r.BloatedFiles,
			"fix_feat_ratio": r.FixFeatRatio,
			"message":        r.Message,
		})
	}
}

func handleArtifactSyncResult(r *daemon.ArtifactSyncResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Artifact sync error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.artifact_sync", map[string]interface{}{
			"error":   r.Error.Error(),
			"message": r.Message,
		})
	} else if r.DriftDetected {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
		logDaemonEvent(logger, "daemon.artifact_sync", map[string]interface{}{
			"drift_detected":  true,
			"entries":         r.EntriesCount,
			"events":          r.EventsCount,
			"issue_id":        r.IssueID,
			"deduped":         r.Deduped,
			"agent_spawned":   r.AgentSpawned,
			"over_budget":     r.OverBudget,
			"claude_md_lines": r.CLAUDEMDLines,
			"message":         r.Message,
		})
	} else if verbose {
		fmt.Printf("[%s] Artifact sync: %s\n", timestamp, r.Message)
	}
}

func handleRegistryRefreshResult(r *daemon.RegistryRefreshResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Registry refresh error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.registry_refresh", map[string]interface{}{
			"error":   r.Error.Error(),
			"message": r.Message,
		})
	} else if r.Changed {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
		if len(r.Added) > 0 {
			fmt.Printf("[%s]   Added: %s\n", timestamp, strings.Join(r.Added, ", "))
		}
		if len(r.Removed) > 0 {
			fmt.Printf("[%s]   Removed: %s\n", timestamp, strings.Join(r.Removed, ", "))
		}
		logDaemonEvent(logger, "daemon.registry_refresh", map[string]interface{}{
			"changed": true,
			"added":   r.Added,
			"removed": r.Removed,
			"message": r.Message,
		})
	} else if verbose {
		fmt.Printf("[%s] Registry refresh: unchanged\n", timestamp)
	}
}

func handleVerificationFailedEscalationResult(r *daemon.VerificationFailedEscalationResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Verification-failed escalation error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.verification_failed_escalation", map[string]interface{}{
			"escalated": 0,
			"error":     r.Error.Error(),
			"message":   r.Message,
		})
	} else if r.EscalatedCount > 0 {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
		escalatedIDs := make([]string, 0, len(r.Escalated))
		for _, e := range r.Escalated {
			escalatedIDs = append(escalatedIDs, e.BeadsID)
		}
		logDaemonEvent(logger, "daemon.verification_failed_escalation", map[string]interface{}{
			"escalated":     r.EscalatedCount,
			"scanned":       r.ScannedCount,
			"escalated_ids": escalatedIDs,
			"message":       r.Message,
		})
	} else if verbose {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
	}
}

func handleLightweightCleanupResult(r *daemon.LightweightCleanupResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Lightweight cleanup error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.lightweight_cleanup", map[string]interface{}{
			"closed":  0,
			"error":   r.Error.Error(),
			"message": r.Message,
		})
	} else if r.ClosedCount > 0 {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
		closedIDs := make([]string, 0, len(r.Closed))
		for _, c := range r.Closed {
			closedIDs = append(closedIDs, c.BeadsID)
		}
		logDaemonEvent(logger, "daemon.lightweight_cleanup", map[string]interface{}{
			"closed":     r.ClosedCount,
			"scanned":    r.ScannedCount,
			"closed_ids": closedIDs,
			"message":    r.Message,
		})
	} else if verbose {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
	}
}

func handleCapacityPollResult(r *daemon.CapacityPollResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Capacity poll error: %v\n", timestamp, r.Error)
	} else if verbose {
		fmt.Printf("[%s] Capacity poll: %s (%d accounts)\n", timestamp, r.Message, r.AccountCount)
	}
}

func handleAuditSelectResult(r *daemon.AuditSelectResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Audit select error: %v\n", timestamp, r.Error)
	} else if len(r.Selected) > 0 {
		ids := make([]string, len(r.Selected))
		for i, s := range r.Selected {
			ids[i] = s.ID
		}
		fmt.Printf("[%s] Audit select: %s (%s)\n", timestamp, r.Message, strings.Join(ids, ", "))
		if logger != nil {
			logDaemonEvent(logger, "daemon.audit_select", map[string]interface{}{
				"selected_count": len(r.Selected),
				"selected_ids":   ids,
			})
		}
	} else if verbose {
		fmt.Printf("[%s] Audit select: %s\n", timestamp, r.Message)
	}
}

// logDaemonEvent logs a daemon event, suppressing errors to stderr.
func logDaemonEvent(logger *events.Logger, eventType string, data map[string]interface{}) {
	event := events.Event{
		Type:      eventType,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log %s event: %v\n", eventType, err)
	}
}
