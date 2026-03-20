// Package main provides the CLI entry point for orch-go.
// This file contains the periodic task scheduler extracted from runDaemonLoop.
// It runs all periodic maintenance tasks (reflection, cleanup, recovery, etc.)
// in a single function and handles logging/event emission.
package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/notify"
)

// planStalenessNotified tracks the last notification time per plan slug.
// Plans are only re-notified after planNotifyCooldown has elapsed.
var (
	planStalenessNotified   = make(map[string]time.Time)
	planStalenessNotifiedMu sync.Mutex
	planNotifyCooldown      = 24 * time.Hour
)

// periodicTasksResult holds outputs from periodic tasks needed downstream.
type periodicTasksResult struct {
	KnowledgeHealthSnapshot      *daemon.KnowledgeHealthSnapshot
	PhaseTimeoutSnapshot         *daemon.PhaseTimeoutSnapshot
	QuestionDetectionSnapshot    *daemon.QuestionDetectionSnapshot
	AgreementCheckSnapshot       *daemon.AgreementCheckSnapshot
	BeadsHealthSnapshot          *daemon.BeadsHealthSnapshot
	FrictionAccumulationSnapshot *daemon.FrictionAccumulationSnapshot
	PlanStalenessSnapshot        *daemon.PlanStalenessSnapshot
	TriggerSnapshot              *daemon.TriggerSnapshot
	InvestigationOrphanSnapshot  *daemon.InvestigationOrphanSnapshot
	TensionClusterSnapshot       *daemon.TensionClusterSnapshot
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
		if r.Error == nil && len(r.Created) > 0 {
			for _, issueID := range r.Created {
				logModelDriftDecision(logger, d.Config.Compliance.Default, issueID)
			}
		}
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
		if r.Error == nil && r.Created > 0 {
			for _, issueID := range r.CreatedIssues {
				logSynthesisIssueDecision(logger, d.Config.Compliance.Default, issueID)
			}
		}
	}

	// Learning refresh + compliance auto-adjust
	if r := d.RunPeriodicLearningRefresh(); r != nil {
		handleLearningRefreshResult(r, timestamp, verbose, logger)
		if r.Error == nil && r.DowngradesApplied > 0 {
			for _, s := range r.Suggestions {
				logComplianceDowngradeDecision(logger, d.Config.Compliance.Default, s.Skill, s.CurrentLevel, s.SuggestedLevel, s.SuccessRate, s.SampleSize)
			}
		}
	}

	// Plan staleness detection
	if r := d.RunPeriodicPlanStaleness(); r != nil {
		handlePlanStalenessResult(r, timestamp, verbose, logger)
		if r.Error == nil {
			snapshot := r.Snapshot()
			result.PlanStalenessSnapshot = &snapshot
		}
	}

	// Proactive extraction (1200-line threshold architect issue creation)
	if r := d.RunPeriodicProactiveExtraction(); r != nil {
		handleProactiveExtractionResult(r, timestamp, verbose, logger)
	}

	// Trigger scan (pattern detectors create issues for recurring problems)
	if r := d.RunPeriodicTriggerScan(d.TriggerDetectors); r != nil {
		handleTriggerScanResult(r, timestamp, verbose, logger)
		if r.Error == nil {
			snapshot := r.Snapshot()
			result.TriggerSnapshot = &snapshot
		}
	}

	// Trigger expiry (auto-close stale daemon:trigger issues)
	if r := d.RunPeriodicTriggerExpiry(); r != nil {
		handleTriggerExpiryResult(r, timestamp, verbose, logger)
	}

	// Digest producer (scans .kb/ artifacts, creates thinking products)
	if r := d.RunPeriodicDigest(); r != nil {
		handleDigestResult(r, timestamp, verbose, logger)
	}

	// Investigation orphan surfacing (investigations in_progress >48h)
	if r := d.RunPeriodicInvestigationOrphan(); r != nil {
		handleInvestigationOrphanResult(r, timestamp, verbose, logger)
		if r.Error == nil {
			snapshot := r.Snapshot()
			result.InvestigationOrphanSnapshot = &snapshot
		}
	}

	// Verification-failed escalation (promote stuck verification-failed to triage:review)
	if r := d.RunPeriodicVerificationFailedEscalation(); r != nil {
		handleVerificationFailedEscalationResult(r, timestamp, verbose, logger)
	}

	// Lightweight cleanup (close stale --no-track / exploration child issues)
	if r := d.RunPeriodicLightweightCleanup(); r != nil {
		handleLightweightCleanupResult(r, timestamp, verbose, logger)
	}

	// Claim probe generation (create investigation issues for stale/unconfirmed claims)
	if r := d.RunPeriodicClaimProbeGeneration(); r != nil {
		handleClaimProbeResult(r, timestamp, verbose)
	}

	// Tension cluster scan (create architect issues for cross-model tension clusters)
	if r := d.RunPeriodicTensionClusterScan(); r != nil {
		handleTensionClusterResult(r, timestamp, verbose, logger)
		if r.Error == nil {
			snapshot := r.Snapshot()
			result.TensionClusterSnapshot = &snapshot
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
			"created":        r.Created,
			"skipped":        r.Skipped,
			"evaluated":      r.Evaluated,
			"created_issues": r.CreatedIssues,
			"message":        r.Message,
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

func handlePlanStalenessResult(r *daemon.PlanStalenessResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Plan staleness error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.plan_staleness", map[string]interface{}{
			"stale":   0,
			"error":   r.Error.Error(),
			"message": r.Message,
		})
	} else if len(r.StalePlans) > 0 {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
		for _, sp := range r.StalePlans {
			fmt.Printf("[%s]   %s: %s (%s)\n", timestamp, sp.Slug, sp.Reason, sp.StalenessType)
		}

		// Send batched desktop notification for stale plans (deduped with 24h cooldown)
		newStaleSlugs := filterNewStalePlans(r.StalePlans)
		if len(newStaleSlugs) > 0 {
			notifier := notify.Default()
			title := fmt.Sprintf("%d stale plans need attention", len(newStaleSlugs))
			message := strings.Join(newStaleSlugs, ", ")
			if err := notifier.Send(title, message); err != nil {
				fmt.Fprintf(os.Stderr, "[%s] Failed to send plan staleness notification: %v\n", timestamp, err)
			}
			markPlansNotified(newStaleSlugs)
		}

		slugs := make([]string, 0, len(r.StalePlans))
		for _, sp := range r.StalePlans {
			slugs = append(slugs, sp.Slug)
		}
		logDaemonEvent(logger, "daemon.plan_staleness", map[string]interface{}{
			"stale":   len(r.StalePlans),
			"scanned": r.ScannedCount,
			"slugs":   slugs,
			"message": r.Message,
		})
	} else if verbose {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
	}
}

func handleProactiveExtractionResult(r *daemon.ProactiveExtractionResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Proactive extraction error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.proactive_extraction", map[string]interface{}{
			"created": 0,
			"error":   r.Error.Error(),
			"message": r.Message,
		})
	} else if r.Created > 0 {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
		logDaemonEvent(logger, "daemon.proactive_extraction", map[string]interface{}{
			"created":          r.Created,
			"skipped":          r.Skipped,
			"skipped_critical": r.SkippedCritical,
			"scanned":          r.Scanned,
			"created_issues":   r.CreatedIssues,
			"message":          r.Message,
		})
	} else if verbose {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
	}
}

func handleTriggerScanResult(r *daemon.TriggerScanResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Trigger scan error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.trigger_scan", map[string]interface{}{
			"created": 0,
			"error":   r.Error.Error(),
			"message": r.Message,
		})
	} else if r.Created > 0 {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
		logDaemonEvent(logger, "daemon.trigger_scan", map[string]interface{}{
			"created":        r.Created,
			"detected":       r.Detected,
			"skipped":        r.Skipped,
			"skipped_budget": r.SkippedBudget,
			"skipped_dedup":  r.SkippedDedup,
			"created_issues": r.CreatedIssues,
			"message":        r.Message,
		})
	} else if verbose {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
	}
}

func handleTriggerExpiryResult(r *daemon.TriggerExpiryResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Trigger expiry error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.trigger_expiry", map[string]interface{}{
			"expired": 0,
			"error":   r.Error.Error(),
			"message": r.Message,
		})
	} else if r.Expired > 0 {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
		logDaemonEvent(logger, "daemon.trigger_expiry", map[string]interface{}{
			"expired":          r.Expired,
			"errors":           r.Errors,
			"expired_issues":   r.ExpiredIssues,
			"detector_outcomes": r.DetectorOutcomes,
			"message":          r.Message,
		})
		// Emit per-detector false positive events for outcome tracking
		for detector, count := range r.DetectorOutcomes {
			logDaemonEvent(logger, events.EventTypeTriggerOutcome, map[string]interface{}{
				"detector": detector,
				"outcome":  "false_positive",
				"count":    count,
			})
		}
	} else if verbose {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
	}
}

func handleDigestResult(r *daemon.DigestResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Digest error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.digest", map[string]interface{}{
			"produced": 0,
			"error":    r.Error.Error(),
			"message":  r.Message,
		})
	} else if r.Produced > 0 {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
		logDaemonEvent(logger, "daemon.digest", map[string]interface{}{
			"produced": r.Produced,
			"skipped":  r.Skipped,
			"scanned":  r.Scanned,
			"message":  r.Message,
		})
	} else if verbose {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
	}
}

func handleInvestigationOrphanResult(r *daemon.InvestigationOrphanResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Investigation orphan error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.investigation_orphan", map[string]interface{}{
			"orphans": 0,
			"error":   r.Error.Error(),
			"message": r.Message,
		})
	} else if r.OrphanCount > 0 {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
		for _, o := range r.Orphans {
			fmt.Printf("[%s]   %s: %s (age %s)\n", timestamp, o.BeadsID, o.Title, o.Age.Round(time.Hour))
		}

		// Send desktop notification for orphaned investigations
		notifier := notify.Default()
		if err := notifier.Send(
			fmt.Sprintf("%d orphaned investigations", r.OrphanCount),
			fmt.Sprintf("Investigations in_progress >%s without completion", r.Orphans[0].Age.Round(time.Hour)),
		); err != nil {
			fmt.Fprintf(os.Stderr, "[%s] Failed to send investigation orphan notification: %v\n", timestamp, err)
		}

		orphanIDs := make([]string, 0, len(r.Orphans))
		for _, o := range r.Orphans {
			orphanIDs = append(orphanIDs, o.BeadsID)
		}
		logDaemonEvent(logger, "daemon.investigation_orphan", map[string]interface{}{
			"orphans":    r.OrphanCount,
			"scanned":    r.ScannedCount,
			"orphan_ids": orphanIDs,
			"message":    r.Message,
		})
	} else if verbose {
		fmt.Printf("[%s] %s\n", timestamp, r.Message)
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
			"escalated":    r.EscalatedCount,
			"scanned":      r.ScannedCount,
			"escalated_ids": escalatedIDs,
			"message":      r.Message,
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

func handleClaimProbeResult(r *daemon.ClaimProbeResult, timestamp string, verbose bool) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Claim probe error: %v\n", timestamp, r.Error)
	} else if r.ProbeCount > 0 {
		fmt.Printf("[%s] Claim probe: %s\n", timestamp, r.Message)
	} else if verbose {
		fmt.Printf("[%s] Claim probe: %s\n", timestamp, r.Message)
	}
}

func handleTensionClusterResult(r *daemon.TensionClusterResult, timestamp string, verbose bool, logger *events.Logger) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] Tension cluster scan error: %v\n", timestamp, r.Error)
		logDaemonEvent(logger, "daemon.tension_cluster", map[string]interface{}{
			"clusters": 0,
			"error":    r.Error.Error(),
			"message":  r.Message,
		})
	} else if r.IssueCreated != "" {
		fmt.Printf("[%s] Tension cluster: %s\n", timestamp, r.Message)
		logDaemonEvent(logger, "daemon.tension_cluster", map[string]interface{}{
			"clusters":      r.ClusterCount,
			"issue_created": r.IssueCreated,
			"message":       r.Message,
		})
	} else if verbose {
		fmt.Printf("[%s] Tension cluster: %s\n", timestamp, r.Message)
	}
}

// filterNewStalePlans returns slugs of stale plans that haven't been notified within the cooldown period.
func filterNewStalePlans(stalePlans []daemon.StalePlan) []string {
	planStalenessNotifiedMu.Lock()
	defer planStalenessNotifiedMu.Unlock()

	now := time.Now()
	var newSlugs []string
	seen := make(map[string]bool) // dedup within a single result (plan can appear multiple times for different staleness types)
	for _, sp := range stalePlans {
		if seen[sp.Slug] {
			continue
		}
		seen[sp.Slug] = true
		if lastNotified, ok := planStalenessNotified[sp.Slug]; ok {
			if now.Sub(lastNotified) < planNotifyCooldown {
				continue
			}
		}
		newSlugs = append(newSlugs, sp.Slug)
	}
	return newSlugs
}

// markPlansNotified records the current time as the last notification time for the given plan slugs.
func markPlansNotified(slugs []string) {
	planStalenessNotifiedMu.Lock()
	defer planStalenessNotifiedMu.Unlock()

	now := time.Now()
	for _, slug := range slugs {
		planStalenessNotified[slug] = now
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
