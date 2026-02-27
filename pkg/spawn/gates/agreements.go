package gates

import (
	"fmt"
	"os"
)

// AgreementFailure represents a single failing kb agreement check.
type AgreementFailure struct {
	AgreementID string `json:"agreement_id"`
	Title       string `json:"title"`
	Severity    string `json:"severity"` // "error" or "warning"
	Message     string `json:"message"`
}

// AgreementsResult contains the result of running kb agreements check.
type AgreementsResult struct {
	Total    int                `json:"total"`
	Passed   int                `json:"passed"`
	Failed   int                `json:"failed"`
	Failures []AgreementFailure `json:"failures,omitempty"`
}

// HasFailures returns true if any agreement checks failed.
func (r *AgreementsResult) HasFailures() bool {
	return len(r.Failures) > 0
}

// HasErrors returns true if any error-severity agreement checks failed.
func (r *AgreementsResult) HasErrors() bool {
	for _, f := range r.Failures {
		if f.Severity == "error" {
			return true
		}
	}
	return false
}

// WarningCount returns the number of warning-severity failures.
func (r *AgreementsResult) WarningCount() int {
	count := 0
	for _, f := range r.Failures {
		if f.Severity == "warning" {
			count++
		}
	}
	return count
}

// ErrorCount returns the number of error-severity failures.
func (r *AgreementsResult) ErrorCount() int {
	count := 0
	for _, f := range r.Failures {
		if f.Severity == "error" {
			count++
		}
	}
	return count
}

// AgreementsChecker runs kb agreements check for a given project directory.
// Returns nil result if no agreements are defined or check is unavailable.
type AgreementsChecker func(projectDir string) (*AgreementsResult, error)

// CheckAgreements runs kb agreements check and displays warnings if any agreements are failing.
// This is a WARNING-ONLY gate (Phase 3) — it never blocks spawn.
// daemonDriven spawns suppress output (triage already happened).
// Returns the result for telemetry/context injection, never returns an error that blocks spawn.
func CheckAgreements(projectDir string, daemonDriven bool, checker AgreementsChecker) (*AgreementsResult, error) {
	if checker == nil || projectDir == "" {
		return nil, nil
	}

	result, err := checker(projectDir)
	if err != nil {
		// Log warning but don't block spawn on infrastructure error
		if !daemonDriven {
			fmt.Fprintf(os.Stderr, "Warning: kb agreements check failed: %v\n", err)
		}
		return nil, nil
	}

	if result == nil {
		return nil, nil
	}

	// Daemon-driven spawns stay silent but still return result for telemetry
	if daemonDriven {
		return result, nil
	}

	// Display warnings for failing agreements
	if result.HasFailures() {
		showAgreementsWarning(result)
	}

	return result, nil
}

// showAgreementsWarning displays agreement failures as a non-blocking warning.
func showAgreementsWarning(result *AgreementsResult) {
	fmt.Fprintf(os.Stderr, "\n⚠️  KB Agreements: %d/%d checks failing\n", result.Failed, result.Total)
	for _, f := range result.Failures {
		icon := "⚠"
		if f.Severity == "error" {
			icon = "✗"
		}
		fmt.Fprintf(os.Stderr, "   %s %s: %s\n", icon, f.AgreementID, f.Title)
		if f.Message != "" {
			fmt.Fprintf(os.Stderr, "     %s\n", f.Message)
		}
	}
	fmt.Fprintf(os.Stderr, "   Run 'kb agreements check' for details\n\n")
}
