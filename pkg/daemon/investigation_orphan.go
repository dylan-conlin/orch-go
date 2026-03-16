// Package daemon provides autonomous overnight processing capabilities.
// This file contains investigation orphan detection: surfaces investigations
// that have been in_progress for longer than the configured threshold (default 48h)
// without completion. Creates closure pressure via notifications and logging
// without taking corrective action (advisory-only).
package daemon

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// InvestigationOrphanResult contains the result of an investigation orphan scan.
type InvestigationOrphanResult struct {
	// OrphanCount is the number of investigations flagged as orphaned.
	OrphanCount int

	// ScannedCount is the total number of in-progress investigations checked.
	ScannedCount int

	// Orphans lists the investigations detected as orphaned.
	Orphans []StaleInvestigation

	// Error is set if the scan failed.
	Error error

	// Message is a human-readable summary.
	Message string
}

// StaleInvestigation represents an investigation that has been in_progress
// for longer than the configured threshold without completion.
type StaleInvestigation struct {
	BeadsID string
	Title   string
	Age     time.Duration
	Labels  []string
}

// InvestigationOrphanSnapshot is a point-in-time snapshot for the daemon status file.
type InvestigationOrphanSnapshot struct {
	OrphanCount  int       `json:"orphan_count"`
	ScannedCount int       `json:"scanned_count"`
	LastCheck    time.Time `json:"last_check"`
}

// Snapshot converts an InvestigationOrphanResult to a dashboard-ready snapshot.
func (r *InvestigationOrphanResult) Snapshot() InvestigationOrphanSnapshot {
	return InvestigationOrphanSnapshot{
		OrphanCount:  r.OrphanCount,
		ScannedCount: r.ScannedCount,
		LastCheck:    time.Now(),
	}
}

// ShouldRunInvestigationOrphan returns true if the periodic scan should run.
func (d *Daemon) ShouldRunInvestigationOrphan() bool {
	return d.Scheduler.IsDue(TaskInvestigationOrphan)
}

// RunPeriodicInvestigationOrphan scans for orphaned investigations if due.
// Returns the result if the scan was run, or nil if it wasn't due.
//
// An orphaned investigation is one that:
// 1. Has status "in_progress"
// 2. Has issue_type "investigation" OR label "skill:investigation"
// 3. Was created more than InvestigationOrphanThreshold ago (default 48h)
//
// This is advisory-only: it surfaces orphans via notifications and logging
// without taking corrective action. The orchestrator decides whether to
// close, rework, or extend orphaned investigations.
func (d *Daemon) RunPeriodicInvestigationOrphan() *InvestigationOrphanResult {
	if !d.ShouldRunInvestigationOrphan() {
		return nil
	}

	issues, err := listInProgressInvestigations()
	if err != nil {
		return &InvestigationOrphanResult{
			Error:   err,
			Message: fmt.Sprintf("Investigation orphan scan failed: %v", err),
		}
	}

	now := time.Now()
	threshold := d.Config.InvestigationOrphanThreshold
	var orphans []StaleInvestigation
	scanned := 0

	for _, issue := range issues {
		scanned++

		createdAt, err := time.Parse(time.RFC3339, issue.CreatedAt)
		if err != nil {
			// Skip issues with unparseable timestamps
			continue
		}

		age := now.Sub(createdAt)
		if age < threshold {
			continue
		}

		orphans = append(orphans, StaleInvestigation{
			BeadsID: issue.ID,
			Title:   issue.Title,
			Age:     age,
			Labels:  issue.Labels,
		})
	}

	d.Scheduler.MarkRun(TaskInvestigationOrphan)

	msg := fmt.Sprintf("Investigation orphans: %d orphaned (>%s), %d scanned",
		len(orphans), threshold.Round(time.Hour), scanned)

	return &InvestigationOrphanResult{
		OrphanCount:  len(orphans),
		ScannedCount: scanned,
		Orphans:      orphans,
		Message:      msg,
	}
}

// listInProgressInvestigations queries beads for in-progress investigation issues.
// Returns issues matching either issue_type="investigation" or label "skill:investigation".
func listInProgressInvestigations() ([]Issue, error) {
	// Try RPC first — query by issue_type
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()

			// Query by issue_type=investigation + status=in_progress
			byType, err := client.List(&beads.ListArgs{
				Status:    "in_progress",
				IssueType: "investigation",
				Limit:     beads.IntPtr(0),
			})
			if err == nil {
				issues := convertBeadsIssues(byType)

				// Also query by skill:investigation label to catch mistyped issues
				byLabel, err := client.List(&beads.ListArgs{
					Status:    "in_progress",
					LabelsAny: []string{"skill:investigation"},
					Limit:     beads.IntPtr(0),
				})
				if err == nil {
					labelIssues := convertBeadsIssues(byLabel)
					issues = deduplicateIssues(issues, labelIssues)
				}

				return issues, nil
			}
		}
	}

	// Fallback to CLI: list in_progress and filter client-side
	return listInProgressInvestigationsCLI()
}

// listInProgressInvestigationsCLI is the CLI fallback for listing in-progress investigations.
func listInProgressInvestigationsCLI() ([]Issue, error) {
	output, err := runBdCommand("list", "--json", "--limit", "0", "--status", "in_progress")
	if err != nil {
		return nil, fmt.Errorf("failed to list in-progress issues: %w", err)
	}

	var allIssues []Issue
	if err := json.Unmarshal(output, &allIssues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	// Filter to investigation type or skill label
	var filtered []Issue
	for _, issue := range allIssues {
		if isInvestigation(issue) {
			filtered = append(filtered, issue)
		}
	}
	return filtered, nil
}

// isInvestigation returns true if the issue is an investigation (by type or label).
func isInvestigation(issue Issue) bool {
	if issue.IssueType == "investigation" {
		return true
	}
	return issue.HasLabel("skill:investigation")
}

// deduplicateIssues merges two issue slices, removing duplicates by ID.
func deduplicateIssues(a, b []Issue) []Issue {
	seen := make(map[string]bool, len(a))
	for _, issue := range a {
		seen[issue.ID] = true
	}
	result := make([]Issue, len(a))
	copy(result, a)
	for _, issue := range b {
		if !seen[issue.ID] {
			result = append(result, issue)
			seen[issue.ID] = true
		}
	}
	return result
}
