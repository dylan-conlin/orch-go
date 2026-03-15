// Package main provides the CLI entry point for orch-go.
// This file provides decision protocol logging helpers for the daemon loop.
// Wraps daemon.LogDecision with decision-class-specific functions to keep
// decision class constants out of daemon_loop.go and daemon_periodic.go.
package main

import (
	"fmt"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/daemonconfig"
	"github.com/dylan-conlin/orch-go/pkg/events"
)

// logSpawnDecision logs a DecisionSelectIssue event after a successful spawn.
func logSpawnDecision(logger *events.Logger, compliance daemonconfig.ComplianceLevel, issueID, skill, model string) {
	daemon.LogDecision(logger, daemon.DecisionLogEntry{
		Class:      daemonconfig.DecisionSelectIssue,
		Compliance: compliance,
		Target:     issueID,
		Reason:     fmt.Sprintf("skill=%s model=%s", skill, model),
	})
}

// logExtractionDecision logs a DecisionRouteExtraction event.
func logExtractionDecision(logger *events.Logger, compliance daemonconfig.ComplianceLevel, originalIssueID, extractionIssueID string) {
	daemon.LogDecision(logger, daemon.DecisionLogEntry{
		Class:      daemonconfig.DecisionRouteExtraction,
		Compliance: compliance,
		Target:     originalIssueID,
		Reason:     fmt.Sprintf("extraction=%s", extractionIssueID),
	})
}

// logArchitectEscalateDecision logs a DecisionArchitectEscalate event.
func logArchitectEscalateDecision(logger *events.Logger, compliance daemonconfig.ComplianceLevel, issueID string) {
	daemon.LogDecision(logger, daemon.DecisionLogEntry{
		Class:      daemonconfig.DecisionArchitectEscalate,
		Compliance: compliance,
		Target:     issueID,
		Reason:     "hotspot area detected",
	})
}

// logCompletionDecision logs a completion routing decision event.
func logCompletionDecision(logger *events.Logger, compliance daemonconfig.ComplianceLevel, beadsID string, autoCompleted bool, closeReason string) {
	class := daemonconfig.DecisionLabelForReview
	if autoCompleted {
		class = daemonconfig.DecisionAutoCompleteFull
	}
	daemon.LogDecision(logger, daemon.DecisionLogEntry{
		Class:      class,
		Compliance: compliance,
		Target:     beadsID,
		Reason:     closeReason,
	})
}

// logModelDriftDecision logs a DecisionCreateModelDriftIssue event.
func logModelDriftDecision(logger *events.Logger, compliance daemonconfig.ComplianceLevel, issueID string) {
	daemon.LogDecision(logger, daemon.DecisionLogEntry{
		Class:      daemonconfig.DecisionCreateModelDriftIssue,
		Compliance: compliance,
		Target:     issueID,
		Reason:     "model drift detected",
	})
}

// logResumeStuckDecision logs a DecisionResumeStuck event.
func logResumeStuckDecision(logger *events.Logger, compliance daemonconfig.ComplianceLevel, count int) {
	daemon.LogDecision(logger, daemon.DecisionLogEntry{
		Class:      daemonconfig.DecisionResumeStuck,
		Compliance: compliance,
		Reason:     fmt.Sprintf("resumed %d idle agents", count),
	})
}

// logResetOrphanDecision logs a DecisionResetOrphan event.
func logResetOrphanDecision(logger *events.Logger, compliance daemonconfig.ComplianceLevel, beadsID string) {
	daemon.LogDecision(logger, daemon.DecisionLogEntry{
		Class:      daemonconfig.DecisionResetOrphan,
		Compliance: compliance,
		Target:     beadsID,
		Reason:     "orphan detected",
	})
}

// logPhaseTimeoutDecision logs a DecisionFlagPhaseTimeout event.
func logPhaseTimeoutDecision(logger *events.Logger, compliance daemonconfig.ComplianceLevel, beadsID string) {
	daemon.LogDecision(logger, daemon.DecisionLogEntry{
		Class:      daemonconfig.DecisionFlagPhaseTimeout,
		Compliance: compliance,
		Target:     beadsID,
		Reason:     "unresponsive",
	})
}

// logAgreementIssueDecision logs a DecisionCreateAgreementIssue event.
func logAgreementIssueDecision(logger *events.Logger, compliance daemonconfig.ComplianceLevel, count int) {
	daemon.LogDecision(logger, daemon.DecisionLogEntry{
		Class:      daemonconfig.DecisionCreateAgreementIssue,
		Compliance: compliance,
		Reason:     fmt.Sprintf("created %d issues for failing agreements", count),
	})
}

// logSynthesisIssueDecision logs a DecisionCreateSynthesisIssue event.
func logSynthesisIssueDecision(logger *events.Logger, compliance daemonconfig.ComplianceLevel, issueID string) {
	daemon.LogDecision(logger, daemon.DecisionLogEntry{
		Class:      daemonconfig.DecisionCreateSynthesisIssue,
		Compliance: compliance,
		Target:     issueID,
		Reason:     "investigation cluster without model",
	})
}

// logComplianceDowngradeDecision logs a DecisionDowngradeCompliance event.
func logComplianceDowngradeDecision(logger *events.Logger, compliance daemonconfig.ComplianceLevel, skill string, from, to daemonconfig.ComplianceLevel, successRate float64, sampleSize int) {
	daemon.LogDecision(logger, daemon.DecisionLogEntry{
		Class:      daemonconfig.DecisionDowngradeCompliance,
		Compliance: compliance,
		Target:     skill,
		Reason:     fmt.Sprintf("%s -> %s (success=%.0f%%, n=%d)", from, to, successRate*100, sampleSize),
	})
}

// logSurfaceRemovalDecision logs a DecisionSurfaceRemoval event.
func logSurfaceRemovalDecision(logger *events.Logger, compliance daemonconfig.ComplianceLevel, issueID, detail string) {
	daemon.LogDecision(logger, daemon.DecisionLogEntry{
		Class:      daemonconfig.DecisionSurfaceRemoval,
		Compliance: compliance,
		Target:     issueID,
		Reason:     detail,
	})
}

// logDetectDuplicateDecision logs a DecisionDetectDuplicate event.
func logDetectDuplicateDecision(logger *events.Logger, compliance daemonconfig.ComplianceLevel, issueA, issueB string, similarity float64) {
	daemon.LogDecision(logger, daemon.DecisionLogEntry{
		Class:      daemonconfig.DecisionDetectDuplicate,
		Compliance: compliance,
		Target:     issueA,
		Reason:     fmt.Sprintf("duplicate of %s (%.0f%%)", issueB, similarity*100),
	})
}
