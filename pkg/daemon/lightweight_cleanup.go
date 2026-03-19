// Package daemon provides autonomous overnight processing capabilities.
// This file contains lightweight issue cleanup: auto-closes stale tier:lightweight
// issues that are stuck in_progress. These are created by --no-track spawns
// (including exploration worker children) and are ephemeral by design.
// Without this cleanup, exploration child issues sit in_progress indefinitely
// after the parent exploration completes.
package daemon

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// LabelLightweight is the label applied to --no-track spawns.
const LabelLightweight = "tier:lightweight"

// LightweightCleanupResult contains the result of a lightweight cleanup scan.
type LightweightCleanupResult struct {
	// ClosedCount is the number of lightweight issues auto-closed.
	ClosedCount int

	// ScannedCount is the total number of lightweight in_progress issues checked.
	ScannedCount int

	// SkippedCount is the number skipped (too new).
	SkippedCount int

	// Closed lists the beads IDs that were auto-closed.
	Closed []ClosedLightweightIssue

	// Error is set if the scan failed.
	Error error

	// Message is a human-readable summary.
	Message string
}

// ClosedLightweightIssue represents a lightweight issue that was auto-closed.
type ClosedLightweightIssue struct {
	BeadsID string
	Title   string
	Age     time.Duration
}

// LightweightCleanupSnapshot is a point-in-time snapshot for the daemon status file.
type LightweightCleanupSnapshot struct {
	ClosedCount  int       `json:"closed_count"`
	ScannedCount int       `json:"scanned_count"`
	LastCheck    time.Time `json:"last_check"`
}

// Snapshot converts a LightweightCleanupResult to a dashboard-ready snapshot.
func (r *LightweightCleanupResult) Snapshot() LightweightCleanupSnapshot {
	return LightweightCleanupSnapshot{
		ClosedCount:  r.ClosedCount,
		ScannedCount: r.ScannedCount,
		LastCheck:    time.Now(),
	}
}

// RunPeriodicLightweightCleanup scans for stale tier:lightweight issues and
// auto-closes them. These issues are created by --no-track spawns (exploration
// workers, judges) and should be cleaned up when they've been idle too long.
//
// Returns nil if the scan wasn't due.
func (d *Daemon) RunPeriodicLightweightCleanup() *LightweightCleanupResult {
	if !d.Scheduler.IsDue(TaskLightweightCleanup) {
		return nil
	}

	issues, err := listLightweightInProgressIssues()
	if err != nil {
		return &LightweightCleanupResult{
			Error:   err,
			Message: fmt.Sprintf("Lightweight cleanup scan failed: %v", err),
		}
	}

	now := time.Now()
	timeout := d.Config.LightweightCleanupTimeout
	closed := 0
	skipped := 0
	var closedIssues []ClosedLightweightIssue

	for _, issue := range issues {
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

		// Auto-close the stale lightweight issue
		reason := fmt.Sprintf("Auto-closed: lightweight issue idle >%s (cascade cleanup)", timeout.Round(time.Minute))
		if err := verify.ForceCloseIssue(issue.ID, reason, ""); err != nil {
			if d.Config.Verbose {
				fmt.Printf("  Failed to close lightweight issue %s: %v\n", issue.ID, err)
			}
			skipped++
			continue
		}

		closedIssues = append(closedIssues, ClosedLightweightIssue{
			BeadsID: issue.ID,
			Title:   issue.Title,
			Age:     age,
		})
		closed++
	}

	d.Scheduler.MarkRun(TaskLightweightCleanup)

	msg := fmt.Sprintf("Lightweight cleanup: %d closed, %d scanned, %d skipped",
		closed, len(issues), skipped)

	return &LightweightCleanupResult{
		ClosedCount:  closed,
		ScannedCount: len(issues),
		SkippedCount: skipped,
		Closed:       closedIssues,
		Message:      msg,
	}
}

// listLightweightInProgressIssues queries beads for in_progress issues
// with the tier:lightweight label.
func listLightweightInProgressIssues() ([]Issue, error) {
	// Try RPC first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()

			result, err := client.List(&beads.ListArgs{
				Status:    "in_progress",
				LabelsAny: []string{LabelLightweight},
				Limit:     beads.IntPtr(0),
			})
			if err == nil {
				return convertBeadsIssues(result), nil
			}
		}
	}

	// Fallback to CLI
	return listLightweightInProgressIssuesCLI()
}

// listLightweightInProgressIssuesCLI is the CLI fallback.
func listLightweightInProgressIssuesCLI() ([]Issue, error) {
	output, err := runBdCommand("list", "--json", "--limit", "0", "--status", "in_progress", "--label", LabelLightweight)
	if err != nil {
		return nil, fmt.Errorf("failed to list lightweight issues: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}
	return issues, nil
}
