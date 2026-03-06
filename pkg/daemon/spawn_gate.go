// Package daemon provides autonomous overnight processing capabilities.
//
// spawn_gate.go defines the SpawnGate pipeline for deduplication checking.
// This extracts the 6-layer dedup gauntlet from spawnIssue() into composable,
// testable checks with clear fail modes and a single pipeline runner.
//
// Architecture:
//   - SpawnGate: blocking check that can reject a spawn (fail-fast or fail-open)
//   - AdvisoryCheck: non-blocking check that produces warnings (never blocks spawn)
//   - SpawnPipeline: runs gates in order, short-circuits on rejection, collects advisories
//
// See: .kb/investigations/2026-03-01-inv-structural-review-daemon-dedup-after.md
package daemon

import (
	"fmt"
)

// GateVerdict represents the outcome of a SpawnGate check.
type GateVerdict int

const (
	// GateAllow means the check passed — spawn may proceed.
	GateAllow GateVerdict = iota
	// GateReject means the check found a reason to block the spawn.
	GateReject
	// GateError means the check encountered an infrastructure error.
	// The pipeline uses the gate's FailMode to decide whether to allow or reject.
	GateError
)

// FailMode determines how infrastructure errors are handled.
type FailMode int

const (
	// FailOpen allows spawn when the check encounters an error.
	// Use for heuristic/advisory checks where blocking work is worse than a potential duplicate.
	FailOpen FailMode = iota
	// FailFast rejects spawn when the check encounters an error.
	// Use for structural checks where proceeding without the check risks duplicates.
	FailFast
)

// GateResult is the outcome of running a single gate.
type GateResult struct {
	// Gate is the name of the gate that produced this result.
	Gate string
	// Verdict is the gate's decision (allow, reject, or error).
	Verdict GateVerdict
	// Message describes why the gate allowed/rejected/errored.
	Message string
	// Err is set when Verdict is GateError.
	Err error
}

// SpawnGate is a blocking dedup check that can prevent a spawn.
type SpawnGate interface {
	// Name returns a human-readable identifier for this gate (for logging/debugging).
	Name() string
	// Check evaluates whether the issue should be spawned.
	// Returns GateAllow to proceed, GateReject to block, GateError on infrastructure failure.
	Check(issue *Issue) GateResult
	// FailMode returns how infrastructure errors should be handled.
	FailMode() FailMode
}

// Advisory is a non-blocking check that produces warnings but never blocks spawn.
type Advisory struct {
	// Name is a human-readable identifier.
	Name string
	// Warning is set if the check found something notable.
	Warning string
}

// AdvisoryCheck is a non-blocking check that produces informational warnings.
type AdvisoryCheck interface {
	// Name returns a human-readable identifier.
	Name() string
	// Check evaluates the issue and returns a warning message, or "" if nothing notable.
	Check(issue *Issue) string
}

// PipelineResult is the outcome of running the full spawn gate pipeline.
type PipelineResult struct {
	// Allowed is true if all gates passed (spawn should proceed).
	Allowed bool
	// RejectedBy is the name of the gate that rejected the spawn (empty if allowed).
	RejectedBy string
	// RejectionMessage describes why the spawn was rejected.
	RejectionMessage string
	// Advisories are warnings from non-blocking checks (populated regardless of outcome).
	Advisories []Advisory
	// GateResults contains the result from each gate that was evaluated.
	GateResults []GateResult
}

// SpawnPipeline runs a sequence of gates and advisory checks.
type SpawnPipeline struct {
	// Gates are evaluated in order. First rejection stops the pipeline.
	Gates []SpawnGate
	// AdvisoryChecks are evaluated after all gates pass.
	AdvisoryChecks []AdvisoryCheck
	// Verbose enables debug logging.
	Verbose bool
}

// Run evaluates gates in order, short-circuiting on the first rejection.
// If all gates pass, advisory checks are evaluated for informational warnings.
func (p *SpawnPipeline) Run(issue *Issue) PipelineResult {
	result := PipelineResult{Allowed: true}

	for _, gate := range p.Gates {
		gr := gate.Check(issue)
		result.GateResults = append(result.GateResults, gr)

		switch gr.Verdict {
		case GateReject:
			result.Allowed = false
			result.RejectedBy = gate.Name()
			result.RejectionMessage = gr.Message
			return result

		case GateError:
			if gate.FailMode() == FailFast {
				result.Allowed = false
				result.RejectedBy = gate.Name()
				result.RejectionMessage = fmt.Sprintf("%s (error: %v)", gr.Message, gr.Err)
				return result
			}
			// FailOpen: continue to next gate

		case GateAllow:
			// Continue to next gate
		}
	}

	// All gates passed — run advisory checks
	for _, check := range p.AdvisoryChecks {
		warning := check.Check(issue)
		if warning != "" {
			result.Advisories = append(result.Advisories, Advisory{
				Name:    check.Name(),
				Warning: warning,
			})
		}
	}

	return result
}

// --- Concrete Gate Implementations ---

// SpawnTrackerGate checks the in-memory spawn tracker (L1).
// Rejects if the issue was recently spawned (within TTL).
type SpawnTrackerGate struct {
	Tracker *SpawnedIssueTracker
}

func (g *SpawnTrackerGate) Name() string     { return "spawn-tracker" }
func (g *SpawnTrackerGate) FailMode() FailMode { return FailOpen }
func (g *SpawnTrackerGate) Check(issue *Issue) GateResult {
	if g.Tracker == nil {
		return GateResult{Gate: g.Name(), Verdict: GateAllow}
	}
	if g.Tracker.IsSpawned(issue.ID) {
		return GateResult{
			Gate:    g.Name(),
			Verdict: GateReject,
			Message: fmt.Sprintf("issue %s recently spawned (in spawn cache)", issue.ID),
		}
	}
	return GateResult{Gate: g.Name(), Verdict: GateAllow}
}

// SessionDedupGate checks for existing OpenCode sessions or tmux windows (L2).
// Rejects if an active session/window exists for this issue.
type SessionDedupGate struct {
	// CheckFunc allows injection for testing. When nil, uses HasExistingSessionForBeadsID.
	CheckFunc func(beadsID string) bool
}

func (g *SessionDedupGate) Name() string     { return "session-dedup" }
func (g *SessionDedupGate) FailMode() FailMode { return FailOpen }
func (g *SessionDedupGate) Check(issue *Issue) GateResult {
	checkFn := g.CheckFunc
	if checkFn == nil {
		checkFn = HasExistingSessionForBeadsID
	}
	if checkFn(issue.ID) {
		return GateResult{
			Gate:    g.Name(),
			Verdict: GateReject,
			Message: fmt.Sprintf("existing session/window found for %s", issue.ID),
		}
	}
	return GateResult{Gate: g.Name(), Verdict: GateAllow}
}

// TitleDedupMemoryGate checks in-memory title dedup (L3).
// Rejects if another issue with the same title was recently spawned.
type TitleDedupMemoryGate struct {
	Tracker *SpawnedIssueTracker
}

func (g *TitleDedupMemoryGate) Name() string     { return "title-dedup-memory" }
func (g *TitleDedupMemoryGate) FailMode() FailMode { return FailOpen }
func (g *TitleDedupMemoryGate) Check(issue *Issue) GateResult {
	if g.Tracker == nil {
		return GateResult{Gate: g.Name(), Verdict: GateAllow}
	}
	if spawned, dupID := g.Tracker.IsTitleSpawned(issue.Title); spawned && dupID != issue.ID {
		return GateResult{
			Gate:    g.Name(),
			Verdict: GateReject,
			Message: fmt.Sprintf("title matches recently spawned %s", dupID),
		}
	}
	return GateResult{Gate: g.Name(), Verdict: GateAllow}
}

// TitleDedupBeadsGate checks beads DB for in_progress issues with same title (L4).
// Rejects if a content duplicate is already in_progress.
type TitleDedupBeadsGate struct {
	// FindFunc allows injection for testing. When nil, uses FindInProgressByTitle.
	FindFunc func(title string) *Issue
}

func (g *TitleDedupBeadsGate) Name() string     { return "title-dedup-beads" }
func (g *TitleDedupBeadsGate) FailMode() FailMode { return FailOpen }
func (g *TitleDedupBeadsGate) Check(issue *Issue) GateResult {
	findFn := g.FindFunc
	if findFn == nil {
		findFn = FindInProgressByTitle
	}
	dup := findFn(issue.Title)
	if dup != nil && dup.ID != issue.ID {
		return GateResult{
			Gate:    g.Name(),
			Verdict: GateReject,
			Message: fmt.Sprintf("duplicate of in_progress issue %s with same title", dup.ID),
		}
	}
	return GateResult{Gate: g.Name(), Verdict: GateAllow}
}

// FreshStatusGate re-fetches issue status to catch TOCTOU races (L5).
// Rejects if the issue is no longer "open".
type FreshStatusGate struct {
	// GetStatusFunc fetches current status. When nil, gate is skipped (allow).
	GetStatusFunc func(beadsID string) (string, error)
	// GetStatusForProjectFunc fetches status for cross-project issues.
	GetStatusForProjectFunc func(beadsID, projectDir string) (string, error)
}

func (g *FreshStatusGate) Name() string     { return "fresh-status" }
func (g *FreshStatusGate) FailMode() FailMode { return FailOpen }
func (g *FreshStatusGate) Check(issue *Issue) GateResult {
	var currentStatus string
	var err error

	if issue.ProjectDir != "" && g.GetStatusForProjectFunc != nil {
		currentStatus, err = g.GetStatusForProjectFunc(issue.ID, issue.ProjectDir)
	} else if g.GetStatusFunc != nil {
		currentStatus, err = g.GetStatusFunc(issue.ID)
	} else {
		// No status check configured — allow
		return GateResult{Gate: g.Name(), Verdict: GateAllow}
	}

	if err != nil {
		return GateResult{
			Gate:    g.Name(),
			Verdict: GateError,
			Message: "failed to fetch fresh status",
			Err:     err,
		}
	}

	if currentStatus != "open" {
		return GateResult{
			Gate:    g.Name(),
			Verdict: GateReject,
			Message: fmt.Sprintf("issue %s is already %s", issue.ID, currentStatus),
		}
	}

	return GateResult{Gate: g.Name(), Verdict: GateAllow}
}

// --- Advisory Check Implementations ---

// SpawnCountAdvisory warns when an issue has been spawned multiple times (thrashing).
type SpawnCountAdvisory struct {
	Tracker   *SpawnedIssueTracker
	Threshold int // Warn when spawn count >= this (default: 3)
}

func (a *SpawnCountAdvisory) Name() string { return "spawn-count" }
func (a *SpawnCountAdvisory) Check(issue *Issue) string {
	if a.Tracker == nil {
		return ""
	}
	threshold := a.Threshold
	if threshold == 0 {
		threshold = 3
	}
	count := a.Tracker.SpawnCount(issue.ID)
	if count >= threshold {
		return fmt.Sprintf("issue %s spawned %d times (possible thrashing)", issue.ID, count)
	}
	return ""
}
