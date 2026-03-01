// Package daemon provides autonomous overnight processing capabilities.
// This file contains periodic agreement checking: runs kb agreements check
// and auto-creates beads issues for failing error-severity agreements.
// Uses label-based dedup (agreement:<id>) to prevent duplicate issues.
package daemon

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// AgreementCheckResult contains the result of a periodic agreement check.
type AgreementCheckResult struct {
	// Total is the total number of agreements checked.
	Total int

	// Passed is the number of agreements passing.
	Passed int

	// Failed is the number of agreements failing.
	Failed int

	// IssuesCreated is the number of new issues created this cycle.
	IssuesCreated int

	// Skipped is the number of failures skipped (open issue exists or severity too low).
	Skipped int

	// Failures holds details for each failing agreement.
	Failures []AgreementFailureDetail

	// Error is set if the check failed.
	Error error

	// Message is a human-readable summary.
	Message string
}

// AgreementFailureDetail holds details about a single failing agreement.
type AgreementFailureDetail struct {
	AgreementID  string
	Title        string
	Severity     string
	Message      string // Check output (failure details)
	AutoFix      *bool  // nil = use severity default
	IssueCreated bool   // Whether issue was created this cycle
	SkipReason   string // Why issue wasn't created (dedup, severity, etc.)
}

// AgreementCheckSnapshot is a point-in-time snapshot for the daemon status file.
type AgreementCheckSnapshot struct {
	Total         int       `json:"total"`
	Passed        int       `json:"passed"`
	Failed        int       `json:"failed"`
	IssuesCreated int       `json:"issues_created"`
	LastCheck     time.Time `json:"last_check"`
}

// Snapshot converts an AgreementCheckResult to a dashboard-ready snapshot.
func (r *AgreementCheckResult) Snapshot() AgreementCheckSnapshot {
	return AgreementCheckSnapshot{
		Total:         r.Total,
		Passed:        r.Passed,
		Failed:        r.Failed,
		IssuesCreated: r.IssuesCreated,
		LastCheck:     time.Now(),
	}
}

// AgreementCheckService provides agreement checking operations.
type AgreementCheckService interface {
	Check() (*AgreementCheckResult, error)
	CreateIssue(failure AgreementFailureDetail) error
	HasOpenIssue(agreementID string) (bool, error)
}

// agreementCheckJSON matches the kb agreements check --json output format.
type agreementCheckJSON struct {
	AgreementID string `json:"agreement_id"`
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	Pass        bool   `json:"pass"`
	Message     string `json:"message"`
	AutoFix     string `json:"auto_fix,omitempty"`
}

// DefaultAgreementCheck shells out to kb agreements check --json and parses results.
func DefaultAgreementCheck() (*AgreementCheckResult, error) {
	cmd := exec.Command("kb", "agreements", "check", "--json")
	output, err := cmd.Output()
	if err != nil {
		// kb agreements check exits non-zero on failures, but still outputs JSON.
		if exitErr, ok := err.(*exec.ExitError); ok {
			if len(output) == 0 {
				return nil, fmt.Errorf("kb agreements check failed: %s", string(exitErr.Stderr))
			}
			// Fall through to parse JSON output
		} else {
			return nil, fmt.Errorf("kb agreements check: %w", err)
		}
	}

	var checks []agreementCheckJSON
	if err := json.Unmarshal(output, &checks); err != nil {
		return nil, fmt.Errorf("failed to parse kb agreements check output: %w", err)
	}

	result := &AgreementCheckResult{
		Total: len(checks),
	}

	for _, check := range checks {
		if check.Pass {
			result.Passed++
		} else {
			result.Failed++
			detail := AgreementFailureDetail{
				AgreementID: check.AgreementID,
				Title:       check.Title,
				Severity:    check.Severity,
				Message:     check.Message,
			}
			// Parse auto_fix if present in JSON output
			if check.AutoFix == "true" {
				t := true
				detail.AutoFix = &t
			} else if check.AutoFix == "false" {
				f := false
				detail.AutoFix = &f
			}
			result.Failures = append(result.Failures, detail)
		}
	}

	return result, nil
}

// shouldAutoCreate determines if a failing agreement should auto-create an issue.
// Default: error severity creates issues, warning/info does not.
// The AutoFix field overrides the severity default when present.
func shouldAutoCreate(failure AgreementFailureDetail) bool {
	if failure.AutoFix != nil {
		return *failure.AutoFix
	}
	return failure.Severity == "error"
}

// DefaultHasOpenAgreementIssue checks if an open issue already exists for the given agreement.
func DefaultHasOpenAgreementIssue(agreementID string) (bool, error) {
	label := fmt.Sprintf("agreement:%s", agreementID)
	cmd := exec.Command("bd", "list", "--status=open", "-l", label)
	output, err := cmd.Output()
	if err != nil {
		// If bd list fails, return false (fail-open: allow creation)
		return false, nil
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			return true, nil
		}
	}
	return false, nil
}

// DefaultCreateAgreementIssue creates a triage:ready issue for a failing agreement.
func DefaultCreateAgreementIssue(failure AgreementFailureDetail) error {
	priority := "2"
	if failure.Severity == "warning" {
		priority = "3"
	}

	title := fmt.Sprintf("Agreement violation: %s (%s)", failure.Title, failure.AgreementID)
	agreementLabel := fmt.Sprintf("agreement:%s", failure.AgreementID)

	description := fmt.Sprintf("## Agreement Violation: %s\n\n**Severity:** %s\n**Agreement ID:** %s\n\n### Check Output\n%s\n\n### Fix Guidance\nFix the code or documentation to satisfy the agreement contract.\nAfter fixing, verify with: kb agreements check",
		failure.Title, failure.Severity, failure.AgreementID, failure.Message)

	cmd := exec.Command("bd", "create",
		"--title", title,
		"--type", "task",
		"--priority", priority,
		"--description", description,
		"-l", "triage:ready",
		"-l", agreementLabel,
		"-l", "area:agreements",
	)
	if _, err := cmd.Output(); err != nil {
		return fmt.Errorf("failed to create agreement issue: %w", err)
	}
	return nil
}

// ShouldRunAgreementCheck returns true if periodic agreement check should run.
func (d *Daemon) ShouldRunAgreementCheck() bool {
	if !d.Config.AgreementCheckEnabled || d.Config.AgreementCheckInterval <= 0 {
		return false
	}
	if d.lastAgreementCheck.IsZero() {
		return true
	}
	return time.Since(d.lastAgreementCheck) >= d.Config.AgreementCheckInterval
}

// RunPeriodicAgreementCheck runs agreement checks if due.
// For each failing error-severity agreement, checks for existing open issue (dedup)
// and creates a new triage:ready issue if none exists.
// Returns the result if the check was run, or nil if it wasn't due.
func (d *Daemon) RunPeriodicAgreementCheck() *AgreementCheckResult {
	if !d.ShouldRunAgreementCheck() {
		return nil
	}

	ac := d.AgreementCheck
	if ac == nil {
		ac = &defaultAgreementCheckService{}
	}

	result, err := ac.Check()
	if err != nil {
		return &AgreementCheckResult{
			Error:   err,
			Message: fmt.Sprintf("Agreement check failed: %v", err),
		}
	}

	// Process failures: dedup and auto-create issues
	for i := range result.Failures {
		failure := &result.Failures[i]

		if !shouldAutoCreate(*failure) {
			failure.SkipReason = fmt.Sprintf("severity %s does not auto-create", failure.Severity)
			result.Skipped++
			continue
		}

		// Dedup: check for existing open issue with this agreement's label
		hasOpen, err := ac.HasOpenIssue(failure.AgreementID)
		if err != nil {
			failure.SkipReason = fmt.Sprintf("dedup check failed: %v", err)
			result.Skipped++
			continue
		}
		if hasOpen {
			failure.SkipReason = "open issue already exists"
			result.Skipped++
			continue
		}

		// Create issue
		if err := ac.CreateIssue(*failure); err != nil {
			failure.SkipReason = fmt.Sprintf("issue creation failed: %v", err)
			result.Skipped++
			continue
		}

		failure.IssueCreated = true
		result.IssuesCreated++
	}

	// Build summary message
	if result.Failed == 0 {
		result.Message = fmt.Sprintf("Agreement check: %d/%d passed", result.Passed, result.Total)
	} else {
		result.Message = fmt.Sprintf("Agreement check: %d/%d passed, %d failed (%d issues created, %d skipped)",
			result.Passed, result.Total, result.Failed, result.IssuesCreated, result.Skipped)
	}

	d.lastAgreementCheck = time.Now()

	return result
}

// LastAgreementCheckTime returns when agreement check was last run.
func (d *Daemon) LastAgreementCheckTime() time.Time {
	return d.lastAgreementCheck
}
