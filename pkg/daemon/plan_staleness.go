// Package daemon provides autonomous overnight processing capabilities.
// This file contains plan staleness detection: scans active plans in .kb/plans/,
// checks their phase progress against beads status, and flags stale plans for
// advancement pressure. Three staleness conditions are detected:
//
//  1. Unhydrated plans — active plans with no beads issues created
//  2. Phase advancement stalls — completed phases whose successors haven't started
//  3. No-progress plans — hydrated plans with zero in_progress or closed phases
package daemon

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/plan"
)

// PlanStalenessResult contains the result of a plan staleness detection operation.
type PlanStalenessResult struct {
	// StalePlans lists plans detected as stale.
	StalePlans []StalePlan

	// ScannedCount is the total number of active plans scanned.
	ScannedCount int

	// Error is set if the detection failed.
	Error error

	// Message is a human-readable summary.
	Message string
}

// StalePlan represents a plan that needs advancement attention.
type StalePlan struct {
	// Slug is the plan slug (derived from filename).
	Slug string

	// Title is the plan title.
	Title string

	// Reason describes why the plan is stale.
	Reason string

	// StalenessType categorizes the staleness condition.
	StalenessType PlanStalenessType
}

// PlanStalenessType categorizes the type of plan staleness.
type PlanStalenessType string

const (
	// StalenessUnhydrated means the plan is active but has no beads issues.
	StalenessUnhydrated PlanStalenessType = "unhydrated"

	// StalenessAdvancementStall means a phase is complete but the next phase hasn't started.
	StalenessAdvancementStall PlanStalenessType = "advancement_stall"

	// StalenessNoProgress means the plan is hydrated but no phases are in progress or complete.
	StalenessNoProgress PlanStalenessType = "no_progress"
)

// PlanStalenessSnapshot is a point-in-time snapshot for the daemon status file.
type PlanStalenessSnapshot struct {
	StaleCount int       `json:"stale_count"`
	LastCheck  time.Time `json:"last_check"`
}

// Snapshot converts a PlanStalenessResult to a dashboard-ready snapshot.
func (r *PlanStalenessResult) Snapshot() PlanStalenessSnapshot {
	return PlanStalenessSnapshot{
		StaleCount: len(r.StalePlans),
		LastCheck:  time.Now(),
	}
}

// PlanStatusQuerier queries beads for issue statuses used by plan staleness detection.
// This interface allows testing without shelling out to bd.
type PlanStatusQuerier interface {
	// QueryIssueStatus returns the beads status for an issue ID.
	// Returns "unknown" on error.
	QueryIssueStatus(id string) string
}

// ShouldRunPlanStaleness returns true if periodic plan staleness detection should run.
func (d *Daemon) ShouldRunPlanStaleness() bool {
	return d.Scheduler.IsDue(TaskPlanStaleness)
}

// RunPeriodicPlanStaleness runs plan staleness detection if due.
// Returns the result if detection was run, or nil if it wasn't due.
//
// A stale plan is one that:
// 1. Is active but has no beads issues (unhydrated > threshold)
// 2. Has completed phases whose successor phases haven't started (advancement stall)
// 3. Is hydrated but has zero in-progress or closed issues (no progress)
func (d *Daemon) RunPeriodicPlanStaleness() *PlanStalenessResult {
	if !d.ShouldRunPlanStaleness() {
		return nil
	}

	// Use the first project dir for plan scanning, or fall back to cwd
	projectDir := d.resolvePlanProjectDir()
	if projectDir == "" {
		d.Scheduler.MarkRun(TaskPlanStaleness)
		return &PlanStalenessResult{
			Message: "Plan staleness: no project directory found",
		}
	}

	plansDir := filepath.Join(projectDir, ".kb", "plans")
	plans, err := plan.ScanDir(plansDir)
	if err != nil {
		return &PlanStalenessResult{
			Error:   err,
			Message: fmt.Sprintf("Plan staleness detection failed: %v", err),
		}
	}

	activePlans := plan.FilterByStatus(plans, "active")
	if len(activePlans) == 0 {
		d.Scheduler.MarkRun(TaskPlanStaleness)
		return &PlanStalenessResult{
			ScannedCount: 0,
			Message:      "Plan staleness: no active plans",
		}
	}

	querier := d.PlanStatusQuerier
	if querier == nil {
		querier = &defaultPlanStatusQuerier{}
	}

	var stalePlans []StalePlan

	for i := range activePlans {
		p := &activePlans[i]
		slug := plan.ExtractSlugFromFilename(p.Filename)

		stale := detectPlanStaleness(p, slug, querier)
		stalePlans = append(stalePlans, stale...)
	}

	d.Scheduler.MarkRun(TaskPlanStaleness)

	msg := fmt.Sprintf("Plan staleness: %d stale of %d active plans", len(stalePlans), len(activePlans))

	return &PlanStalenessResult{
		StalePlans:   stalePlans,
		ScannedCount: len(activePlans),
		Message:      msg,
	}
}

// detectPlanStaleness checks a single plan for staleness conditions.
func detectPlanStaleness(p *plan.File, slug string, querier PlanStatusQuerier) []StalePlan {
	var results []StalePlan

	// Condition 1: Unhydrated active plan
	if !p.IsHydrated() {
		results = append(results, StalePlan{
			Slug:          slug,
			Title:         p.Title,
			Reason:        fmt.Sprintf("Active plan has no beads issues. Run: orch plan hydrate %s", slug),
			StalenessType: StalenessUnhydrated,
		})
		return results // Can't check other conditions without beads
	}

	// Collect beads statuses for all phases
	phaseStatuses := make([]string, len(p.Phases))
	for i, phase := range p.Phases {
		phaseStatuses[i] = computePhaseStatusFromQuerier(phase.BeadsIDs, querier)
	}

	// Condition 2: Phase advancement stall
	// A phase is complete but its dependent successor hasn't started
	for i, status := range phaseStatuses {
		if status != "complete" {
			continue
		}
		// Check if any later phase depends on this one and is still "ready" (not started)
		for j := i + 1; j < len(p.Phases); j++ {
			depIndices := plan.ParseDependsOn(p.Phases[j].DependsOn)
			dependsOnThis := false
			for _, idx := range depIndices {
				if idx == i {
					dependsOnThis = true
					break
				}
			}
			if !dependsOnThis {
				continue
			}
			if phaseStatuses[j] == "ready" || phaseStatuses[j] == "no-issues" {
				results = append(results, StalePlan{
					Slug:  slug,
					Title: p.Title,
					Reason: fmt.Sprintf("Phase %d (%s) complete but Phase %d (%s) not started",
						i+1, p.Phases[i].Name, j+1, p.Phases[j].Name),
					StalenessType: StalenessAdvancementStall,
				})
			}
		}
	}

	// Condition 3: No-progress plan — hydrated but no phases in_progress or complete
	hasProgress := false
	for _, status := range phaseStatuses {
		if status == "in-progress" || status == "complete" {
			hasProgress = true
			break
		}
	}
	if !hasProgress {
		results = append(results, StalePlan{
			Slug:          slug,
			Title:         p.Title,
			Reason:        "Hydrated plan has no phases in progress or complete",
			StalenessType: StalenessNoProgress,
		})
	}

	return results
}

// computePhaseStatusFromQuerier determines the overall phase status from its beads issues.
func computePhaseStatusFromQuerier(beadsIDs []string, querier PlanStatusQuerier) string {
	if len(beadsIDs) == 0 {
		return "no-issues"
	}

	allClosed := true
	anyInProgress := false
	anyOpen := false

	for _, id := range beadsIDs {
		status := querier.QueryIssueStatus(id)
		switch status {
		case "closed":
			// ok
		case "in_progress":
			allClosed = false
			anyInProgress = true
		default:
			allClosed = false
			anyOpen = true
		}
	}

	if allClosed {
		return "complete"
	}
	if anyInProgress {
		return "in-progress"
	}
	if anyOpen {
		return "ready"
	}
	return "unknown"
}

// defaultPlanStatusQuerier shells out to bd to query issue status.
type defaultPlanStatusQuerier struct{}

func (q *defaultPlanStatusQuerier) QueryIssueStatus(id string) string {
	// Import beads package for real queries
	// For now, return "unknown" — the real implementation wires through beads.CLIClient
	return "unknown"
}

// resolvePlanProjectDir returns the first project directory from the registry,
// or empty string if none available.
func (d *Daemon) resolvePlanProjectDir() string {
	if d.ProjectRegistry != nil {
		for _, entry := range d.ProjectRegistry.Projects() {
			return entry.Dir
		}
	}
	return ""
}

// LastPlanStalenessTime returns when plan staleness detection was last run.
func (d *Daemon) LastPlanStalenessTime() time.Time {
	return d.Scheduler.LastRunTime(TaskPlanStaleness)
}

// NextPlanStalenessTime returns when the next plan staleness detection is scheduled.
func (d *Daemon) NextPlanStalenessTime() time.Time {
	return d.Scheduler.NextRunTime(TaskPlanStaleness)
}
