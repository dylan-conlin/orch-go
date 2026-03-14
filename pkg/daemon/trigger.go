// Package daemon provides autonomous overnight processing capabilities.
// Trigger scan runs pattern detectors that create beads issues for detected patterns.
// This implements the autonomous trigger layer — detectors surface patterns,
// the orchestrator creates issues, and issues flow through the normal OODA pipeline.
//
// Critical constraint: a global trigger budget prevents creation/removal asymmetry
// from bloating the issue queue.
package daemon

import "fmt"

const (
	// TriggerLabel is the beads label for all daemon-triggered issues (for budget counting).
	TriggerLabel = "daemon:trigger"
)

// TriggerSuggestion is a single finding from a pattern detector.
type TriggerSuggestion struct {
	// Detector is the name of the detector that produced this suggestion.
	Detector string
	// Key is a dedup key unique to this detector. Used to prevent duplicate issues.
	Key string
	// Title is the beads issue title.
	Title string
	// Description is the beads issue description.
	Description string
	// IssueType is the beads issue type (bug, task, feature).
	IssueType string
	// Priority is the beads issue priority (0-4).
	Priority int
	// Labels are additional beads labels (TriggerLabel is always added).
	Labels []string
}

// PatternDetector detects a specific pattern and returns trigger suggestions.
// Each detector is independent and can fail without affecting others.
type PatternDetector interface {
	// Name returns the detector's unique name (used in labels and logging).
	Name() string
	// Detect runs the detection logic and returns any findings.
	// Errors are non-fatal — the orchestrator skips broken detectors.
	Detect() ([]TriggerSuggestion, error)
}

// TriggerBudget enforces the global autonomous budget.
// Prevents creation/removal asymmetry from bloating the issue queue.
type TriggerBudget struct {
	// MaxOpen is the maximum number of open daemon:trigger issues allowed.
	MaxOpen int
}

// CanCreate returns true if the budget allows creating another trigger issue.
func (b TriggerBudget) CanCreate(currentOpen int) bool {
	if b.MaxOpen <= 0 {
		return false
	}
	return currentOpen < b.MaxOpen
}

// TriggerScanService provides I/O operations for the trigger scan orchestrator.
type TriggerScanService interface {
	// CountOpenTriggerIssues counts all open issues with the daemon:trigger label.
	CountOpenTriggerIssues() (int, error)
	// HasOpenTriggerIssue checks if an open issue exists for this detector+key combo.
	HasOpenTriggerIssue(detectorName, key string) (bool, error)
	// CreateTriggerIssue creates a beads issue from a trigger suggestion.
	// Returns the created issue ID.
	CreateTriggerIssue(s TriggerSuggestion) (string, error)
}

// TriggerScanResult contains the result of running all pattern detectors.
type TriggerScanResult struct {
	// Detected is the total number of patterns found across all detectors.
	Detected int
	// Created is the number of beads issues created.
	Created int
	// Skipped is the total number of suggestions skipped (budget + dedup).
	Skipped int
	// SkippedBudget is how many were skipped due to budget exhaustion.
	SkippedBudget int
	// SkippedDedup is how many were skipped due to existing open issues.
	SkippedDedup int
	// DetectorErrors is the number of detectors that returned errors.
	DetectorErrors int
	// CreatedIssues contains the beads IDs of created issues.
	CreatedIssues []string
	// Message is a human-readable summary.
	Message string
	// Error is set if a fatal error occurred (e.g., budget count failed).
	Error error
}

// RunPeriodicTriggerScan runs all pattern detectors and creates issues for findings.
// Enforces the global trigger budget to prevent issue queue bloat.
// Returns nil if the task is not due.
func (d *Daemon) RunPeriodicTriggerScan(detectors []PatternDetector) *TriggerScanResult {
	if !d.Scheduler.IsDue(TaskTriggerScan) {
		return nil
	}

	svc := d.TriggerScan
	if svc == nil {
		return &TriggerScanResult{
			Error:   fmt.Errorf("trigger scan service not configured"),
			Message: "Trigger scan: service not configured",
		}
	}

	budget := TriggerBudget{MaxOpen: d.Config.TriggerBudgetMax}

	// Get current open count for budget enforcement
	currentOpen, err := svc.CountOpenTriggerIssues()
	if err != nil {
		d.Scheduler.MarkRun(TaskTriggerScan)
		return &TriggerScanResult{
			Error:   fmt.Errorf("failed to count open trigger issues: %w", err),
			Message: fmt.Sprintf("Trigger scan: failed to count open issues: %v", err),
		}
	}

	result := &TriggerScanResult{}

	// Run each detector and collect suggestions
	var allSuggestions []TriggerSuggestion
	for _, detector := range detectors {
		suggestions, err := detector.Detect()
		if err != nil {
			result.DetectorErrors++
			continue
		}
		allSuggestions = append(allSuggestions, suggestions...)
	}

	result.Detected = len(allSuggestions)

	// Process each suggestion through budget + dedup gates
	for _, s := range allSuggestions {
		// Gate 1: Budget check
		if !budget.CanCreate(currentOpen) {
			result.Skipped++
			result.SkippedBudget++
			continue
		}

		// Gate 2: Dedup check
		hasOpen, err := svc.HasOpenTriggerIssue(s.Detector, s.Key)
		if err != nil {
			// Fail-safe: skip on error
			result.Skipped++
			continue
		}
		if hasOpen {
			result.Skipped++
			result.SkippedDedup++
			continue
		}

		// Create the issue
		issueID, err := svc.CreateTriggerIssue(s)
		if err != nil {
			result.Error = err
			continue
		}

		result.Created++
		result.CreatedIssues = append(result.CreatedIssues, issueID)
		currentOpen++ // Track budget consumption within this scan
	}

	// Build summary
	if result.Created > 0 {
		result.Message = fmt.Sprintf("Trigger scan: created %d issue(s) from %d pattern(s) detected", result.Created, result.Detected)
		if result.Skipped > 0 {
			result.Message += fmt.Sprintf(", skipped %d (budget: %d, dedup: %d)",
				result.Skipped, result.SkippedBudget, result.SkippedDedup)
		}
	} else if result.Detected > 0 {
		result.Message = fmt.Sprintf("Trigger scan: %d pattern(s) detected, all skipped (budget: %d, dedup: %d)",
			result.Detected, result.SkippedBudget, result.SkippedDedup)
	} else {
		result.Message = "Trigger scan: no patterns detected"
	}
	if result.DetectorErrors > 0 {
		result.Message += fmt.Sprintf(", %d detector error(s)", result.DetectorErrors)
	}

	d.Scheduler.MarkRun(TaskTriggerScan)
	return result
}
