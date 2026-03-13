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
	KnowledgeHealthSnapshot        *daemon.KnowledgeHealthSnapshot
	PhaseTimeoutSnapshot           *daemon.PhaseTimeoutSnapshot
	QuestionDetectionSnapshot      *daemon.QuestionDetectionSnapshot
	AgreementCheckSnapshot         *daemon.AgreementCheckSnapshot
	BeadsHealthSnapshot            *daemon.BeadsHealthSnapshot
	FrictionAccumulationSnapshot   *daemon.FrictionAccumulationSnapshot
}

// runPeriodicTasks runs all periodic maintenance tasks and handles their output.
// Returns any snapshots needed by the caller for status file writing.
func runPeriodicTasks(d *daemon.Daemon, timestamp string, verbose bool, logger *events.Logger) periodicTasksResult {
	var result periodicTasksResult

	// Reflection
	if r := d.RunPeriodicReflection(); r != nil {
		handleReflectionResult(r, timestamp, verbose)
	}

	// Model drift reflection
	if r := d.RunPeriodicModelDriftReflection(); r != nil {
		handleModelDriftResult(r, timestamp, verbose)
	}

	// Knowledge health
	if r := d.RunPeriodicKnowledgeHealth(); r != nil {
		handleKnowledgeHealthResult(r, timestamp, verbose)
		if r.Error == nil {
			snapshot := r.Snapshot()
			result.KnowledgeHealthSnapshot = &snapshot
		}
	}

	// Session cleanup
	if r := d.RunPeriodicCleanup(); r != nil {
		handleCleanupResult(r, timestamp, verbose, logger)
	}

	// Stuck agent recovery
	if r := d.RunPeriodicRecovery(); r != nil {
		handleRecoveryResult(r, timestamp, verbose, logger)
	}

	// Orphan detection
	if r := d.RunPeriodicOrphanDetection(); r != nil {
		handleOrphanDetectionResult(r, timestamp, verbose, logger)
	}

	// Phase timeout detection
	if r := d.RunPeriodicPhaseTimeout(); r != nil {
		handlePhaseTimeoutResult(r, timestamp, verbose, logger)
		if r.Error == nil {
			snapshot := r.Snapshot()
			result.PhaseTimeoutSnapshot = &snapshot
		}
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
		if r.Error == nil {
			snapshot := r.Snapshot()
			result.AgreementCheckSnapshot = &snapshot
		}
	}

	// Beads health snapshot collection
	if r := d.RunPeriodicBeadsHealth(); r != nil {
		handleBeadsHealthResult(r, timestamp, verbose, logger)
		if r.Error == nil {
			snapshot := r.Snapshot()
			result.BeadsHealthSnapshot = &snapshot
		}
	}

	// Friction accumulation
	if r := d.RunPeriodicFrictionAccumulation(); r != nil {
		handleFrictionAccumulationResult(r, timestamp, verbose, logger)
		if r.Error == nil {
			snapshot := r.Snapshot()
			result.FrictionAccumulationSnapshot = &snapshot
		}
	}

	// Artifact sync
	if r := d.RunPeriodicArtifactSync(); r != nil {
		handleArtifactSyncResult(r, timestamp, verbose, logger)
	}

	// Project registry refresh (picks up new projects without daemon restart)
	if r := d.RunPeriodicRegistryRefresh(); r != nil {
		handleRegistryRefreshResult(r, timestamp, verbose, logger)
	}

	// Synthesis auto-create (creates issues for investigation clusters without models)
	if r := d.RunPeriodicSynthesisAutoCreate(); r != nil {
		handleSynthesisAutoCreateResult(r, timestamp, verbose, logger)
	}

	// Learning refresh + compliance auto-adjust
	if r := d.RunPeriodicLearningRefresh(); r != nil {
		handleLearningRefreshResult(r, timestamp, verbose, logger)
	}

	return result
}

func handleReflectionResult(r *daemon.ReflectResult, timestamp string, verbose bool) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Reflection error: %v\n", timestamp, r.Error)
	} else if r.Suggestions != nil && r.Suggestions.HasSuggestions() {
		fmt.Printf("[%s] Reflection: %s\n", timestamp, r.Suggestions.Summary())
	} else if verbose {
		fmt.Printf("[%s] Reflection: no suggestions found\n", timestamp)
	}
}

func handleModelDriftResult(r *daemon.ModelDriftResult, timestamp string, verbose bool) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Model drift error: %v\n", timestamp, r.Error)
	} else if r.Message != "" {
		fmt.Printf("[%s] Model drift: %s\n", timestamp, r.Message)
	} else if verbose {
		fmt.Printf("[%s] Model drift: no updates\n", timestamp)
	}
}

func handleKnowledgeHealthResult(r *daemon.KnowledgeHealthResult, timestamp string, verbose bool) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Knowledge health error: %v\n", timestamp, r.Error)
	} else if r.ThresholdExceeded {
		fmt.Printf("[%s] \u26a0\ufe0f  %s\n", timestamp, r.Message)
	} else if verbose {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
	}
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

func handleFrictionAccumulationResult(r *daemon.FrictionAccumulationResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Friction accumulation error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.friction_accumulation", map[string]interface{}{
			"new_items": 0,
			"error":     r.Error.Error(),
			"message":   r.Message,
		})
	} else if r.NewItems > 0 {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
		logDaemonEvent(logger, "daemon.friction_accumulation", map[string]interface{}{
			"new_items":         r.NewItems,
			"by_category_count": r.ByCategoryCount,
			"message":           r.Message,
		})
	} else if verbose {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
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
			"drift_detected": true,
			"entries":        r.EntriesCount,
			"events":         r.EventsCount,
			"issue_id":       r.IssueID,
			"deduped":        r.Deduped,
			"agent_spawned":  r.AgentSpawned,
			"message":        r.Message,
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

func handleSynthesisAutoCreateResult(r *daemon.SynthesisAutoCreateResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Synthesis auto-create error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.synthesis_auto_create", map[string]interface{}{
			"created": 0,
			"error":   r.Error.Error(),
			"message": r.Message,
		})
	} else if r.Created > 0 {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
		logDaemonEvent(logger, "daemon.synthesis_auto_create", map[string]interface{}{
			"created":       r.Created,
			"skipped":       r.Skipped,
			"evaluated":     r.Evaluated,
			"created_issues": r.CreatedIssues,
			"message":       r.Message,
		})
	} else if verbose {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
	}
}

func handleLearningRefreshResult(r *daemon.LearningRefreshResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Learning refresh error: %v\n", timestamp, r.Error)
	} else if r.DowngradesApplied > 0 {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
		for _, s := range r.Suggestions {
			fmt.Printf("[%s]   %s: %s -> %s (success rate=%.0f%%, samples=%d)\n",
				timestamp, s.Skill, s.CurrentLevel, s.SuggestedLevel, s.SuccessRate*100, s.SampleSize)
		}
		logDaemonEvent(logger, "daemon.learning_refresh", map[string]interface{}{
			"downgrades_applied": r.DowngradesApplied,
			"message":            r.Message,
		})
	} else if verbose {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
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
