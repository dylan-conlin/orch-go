// Package daemon provides autonomous overnight processing capabilities.
// This file contains verification-failed escalation: promotes daemon:verification-failed
// issues to triage:review after a timeout so humans can address them.
// Without this, verification-failed agents sit in_progress indefinitely
// because the completion scanner filters them out after retry budget exhaustion.
package daemon

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// VerificationFailedEscalationResult contains the result of a verification-failed escalation scan.
type VerificationFailedEscalationResult struct {
	// EscalatedCount is the number of issues escalated to triage:review.
	EscalatedCount int

	// ScannedCount is the total number of verification-failed issues checked.
	ScannedCount int

	// SkippedCount is the number skipped (too new, already has triage:review).
	SkippedCount int

	// Escalated lists the beads IDs that were escalated.
	Escalated []EscalatedIssue

	// Error is set if the scan failed.
	Error error

	// Message is a human-readable summary.
	Message string
}

// EscalatedIssue represents a verification-failed issue that was escalated.
type EscalatedIssue struct {
	BeadsID string
	Title   string
	Age     time.Duration
}

// VerificationFailedEscalationSnapshot is a point-in-time snapshot for the daemon status file.
type VerificationFailedEscalationSnapshot struct {
	EscalatedCount int       `json:"escalated_count"`
	ScannedCount   int       `json:"scanned_count"`
	LastCheck      time.Time `json:"last_check"`
}

// Snapshot converts a VerificationFailedEscalationResult to a dashboard-ready snapshot.
func (r *VerificationFailedEscalationResult) Snapshot() VerificationFailedEscalationSnapshot {
	return VerificationFailedEscalationSnapshot{
		EscalatedCount: r.EscalatedCount,
		ScannedCount:   r.ScannedCount,
		LastCheck:      time.Now(),
	}
}

// LabelTriageReview is the label used to escalate issues for human review.
const LabelTriageReview = "triage:review"

// RunPeriodicVerificationFailedEscalation scans for daemon:verification-failed issues
// that have been stuck for longer than the configured timeout and escalates them
// to triage:review for human attention.
//
// Returns nil if the scan wasn't due.
func (d *Daemon) RunPeriodicVerificationFailedEscalation() *VerificationFailedEscalationResult {
	if !d.Scheduler.IsDue(TaskVerificationFailedEscalation) {
		return nil
	}

	issues, err := listVerificationFailedIssues()
	if err != nil {
		return &VerificationFailedEscalationResult{
			Error:   err,
			Message: fmt.Sprintf("Verification-failed escalation scan failed: %v", err),
		}
	}

	now := time.Now()
	timeout := d.Config.VerificationFailedEscalationTimeout
	escalated := 0
	skipped := 0
	var escalatedIssues []EscalatedIssue

	for _, issue := range issues {
		// Skip issues that already have triage:review (already escalated)
		if issue.HasLabel(LabelTriageReview) {
			skipped++
			continue
		}

		// Check age threshold
		createdAt, err := time.Parse(time.RFC3339, issue.CreatedAt)
		if err != nil {
			skipped++
			continue
		}

		age := now.Sub(createdAt)
		if age < timeout {
			skipped++
			continue
		}

		// Escalate: add triage:review label
		if err := verify.AddLabel(issue.ID, LabelTriageReview, ""); err != nil {
			if d.Config.Verbose {
				fmt.Printf("  Failed to escalate %s: %v\n", issue.ID, err)
			}
			skipped++
			continue
		}

		escalatedIssues = append(escalatedIssues, EscalatedIssue{
			BeadsID: issue.ID,
			Title:   issue.Title,
			Age:     age,
		})
		escalated++
	}

	d.Scheduler.MarkRun(TaskVerificationFailedEscalation)

	msg := fmt.Sprintf("Verification-failed escalation: %d escalated to triage:review, %d scanned, %d skipped",
		escalated, len(issues), skipped)

	return &VerificationFailedEscalationResult{
		EscalatedCount: escalated,
		ScannedCount:   len(issues),
		SkippedCount:   skipped,
		Escalated:      escalatedIssues,
		Message:        msg,
	}
}

// listVerificationFailedIssues queries beads for in_progress issues
// with the daemon:verification-failed label.
func listVerificationFailedIssues() ([]Issue, error) {
	// Try RPC first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()

			result, err := client.List(&beads.ListArgs{
				Status:    "in_progress",
				LabelsAny: []string{LabelVerificationFailed},
				Limit:     beads.IntPtr(0),
			})
			if err == nil {
				return convertBeadsIssues(result), nil
			}
		}
	}

	// Fallback to CLI
	return listVerificationFailedIssuesCLI()
}

// listVerificationFailedIssuesCLI is the CLI fallback.
func listVerificationFailedIssuesCLI() ([]Issue, error) {
	output, err := runBdCommand("list", "--json", "--limit", "0", "--status", "in_progress", "--label", LabelVerificationFailed)
	if err != nil {
		return nil, fmt.Errorf("failed to list verification-failed issues: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}
	return issues, nil
}
