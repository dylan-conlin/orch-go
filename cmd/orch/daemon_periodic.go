// Package main provides the CLI entry point for orch-go.
// This file contains the periodic task scheduler extracted from runDaemonLoop.
// It runs all periodic maintenance tasks (reflection, cleanup, recovery, etc.)
// in a single function and handles logging/event emission.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/notify"
)

// periodicTasksResult holds outputs from periodic tasks needed downstream.
type periodicTasksResult struct {
	KnowledgeHealthSnapshot      *daemon.KnowledgeHealthSnapshot
	PhaseTimeoutSnapshot         *daemon.PhaseTimeoutSnapshot
	QuestionDetectionSnapshot    *daemon.QuestionDetectionSnapshot
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
