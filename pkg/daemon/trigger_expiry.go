// Package daemon provides autonomous overnight processing capabilities.
// Trigger expiry auto-closes daemon:trigger issues not acted on within a configurable
// max age (default 14 days). This addresses creation/removal asymmetry — the system
// can create trigger issues faster than agents close them, so stale triggers are
// retired automatically.
package daemon

import (
	"fmt"
	"strings"
	"time"
)

// ExpiredTriggerIssue represents a daemon:trigger issue that has exceeded its max age.
type ExpiredTriggerIssue struct {
	ID     string
	Title  string
	Age    time.Duration
	Labels []string
}

// DetectorName extracts the detector name from the issue's labels.
// Looks for labels matching "daemon:trigger:{name}" and returns the name portion.
// Returns "unknown" if no detector label is found.
func (e *ExpiredTriggerIssue) DetectorName() string {
	for _, label := range e.Labels {
		if strings.HasPrefix(label, "daemon:trigger:") {
			name := strings.TrimPrefix(label, "daemon:trigger:")
			if name != "" {
				return name
			}
		}
	}
	return "unknown"
}

// TriggerExpiryService provides I/O for the trigger expiry mechanism.
type TriggerExpiryService interface {
	// ListExpiredTriggerIssues returns open daemon:trigger issues older than maxAge.
	ListExpiredTriggerIssues(maxAge time.Duration) ([]ExpiredTriggerIssue, error)
	// ExpireTriggerIssue closes the issue and adds the daemon:expired label.
	ExpireTriggerIssue(id, reason string) error
}

// TriggerExpiryResult contains the result of running the trigger expiry task.
type TriggerExpiryResult struct {
	// Expired is the number of issues successfully expired.
	Expired int
	// Errors is the number of issues that failed to expire.
	Errors int
	// ExpiredIssues contains the IDs of expired issues.
	ExpiredIssues []string
	// DetectorOutcomes maps detector name to count of false positives (expired issues).
	// This provides per-detector quality tracking — detectors with high false positive
	// counts relative to their creation counts need tuning or removal.
	DetectorOutcomes map[string]int
	// Message is a human-readable summary.
	Message string
	// Error is set if a fatal error occurred (e.g., listing failed).
	Error error
}

// RunPeriodicTriggerExpiry closes stale daemon:trigger issues that haven't been
// acted on within TriggerExpiryMaxAge. Returns nil if the task is not due.
func (d *Daemon) RunPeriodicTriggerExpiry() *TriggerExpiryResult {
	if !d.Scheduler.IsDue(TaskTriggerExpiry) {
		return nil
	}

	svc := d.TriggerExpiry
	if svc == nil {
		return &TriggerExpiryResult{
			Error:   fmt.Errorf("trigger expiry service not configured"),
			Message: "Trigger expiry: service not configured",
		}
	}

	maxAge := d.Config.TriggerExpiryMaxAge
	if maxAge <= 0 {
		maxAge = 14 * 24 * time.Hour // default 14 days
	}

	expired, err := svc.ListExpiredTriggerIssues(maxAge)
	if err != nil {
		d.Scheduler.MarkRun(TaskTriggerExpiry)
		return &TriggerExpiryResult{
			Error:   fmt.Errorf("failed to list expired trigger issues: %w", err),
			Message: fmt.Sprintf("Trigger expiry: failed to list expired issues: %v", err),
		}
	}

	result := &TriggerExpiryResult{
		DetectorOutcomes: make(map[string]int),
	}

	if len(expired) == 0 {
		result.Message = "Trigger expiry: no expired issues found"
		d.Scheduler.MarkRun(TaskTriggerExpiry)
		return result
	}

	reason := fmt.Sprintf("Auto-expired: pattern trigger not acted upon within %d days", int(maxAge.Hours()/24))

	for _, issue := range expired {
		if err := svc.ExpireTriggerIssue(issue.ID, reason); err != nil {
			result.Errors++
			continue
		}
		result.Expired++
		result.ExpiredIssues = append(result.ExpiredIssues, issue.ID)

		// Track per-detector false positive
		detector := issue.DetectorName()
		result.DetectorOutcomes[detector]++
	}

	// Build summary
	if result.Expired > 0 {
		result.Message = fmt.Sprintf("Trigger expiry: expired %d issue(s)", result.Expired)
		if result.Errors > 0 {
			result.Message += fmt.Sprintf(", %d error(s)", result.Errors)
		}
		// Append per-detector breakdown
		if len(result.DetectorOutcomes) > 0 {
			var parts []string
			for detector, count := range result.DetectorOutcomes {
				parts = append(parts, fmt.Sprintf("%s=%d", detector, count))
			}
			result.Message += fmt.Sprintf(" [detectors: %s]", strings.Join(parts, ", "))
		}
	} else {
		result.Message = fmt.Sprintf("Trigger expiry: 0 expired, %d error(s)", result.Errors)
	}

	d.Scheduler.MarkRun(TaskTriggerExpiry)
	return result
}
