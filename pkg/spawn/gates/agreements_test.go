package gates

import (
	"fmt"
	"testing"
)

func TestCheckAgreements_NilChecker(t *testing.T) {
	result, err := CheckAgreements("", false, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result with nil checker, got: %v", result)
	}
}

func TestCheckAgreements_EmptyDir(t *testing.T) {
	checkerCalled := false
	checker := func(dir string) (*AgreementsResult, error) {
		checkerCalled = true
		return nil, nil
	}
	result, err := CheckAgreements("", false, checker)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result with empty dir, got: %v", result)
	}
	if checkerCalled {
		t.Error("checker should not be called with empty dir")
	}
}

func TestCheckAgreements_AllPass(t *testing.T) {
	checker := func(dir string) (*AgreementsResult, error) {
		return &AgreementsResult{
			Total:    3,
			Passed:   3,
			Failed:   0,
			Failures: nil,
		}, nil
	}
	result, err := CheckAgreements("/some/dir", false, checker)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result when all pass")
	}
	if result.HasFailures() {
		t.Error("expected no failures when all pass")
	}
}

func TestCheckAgreements_WithWarningFailures(t *testing.T) {
	checker := func(dir string) (*AgreementsResult, error) {
		return &AgreementsResult{
			Total:  5,
			Passed: 4,
			Failed: 1,
			Failures: []AgreementFailure{
				{
					AgreementID: "kb-agr-002",
					Title:       "Investigations have status",
					Severity:    "warning",
					Message:     "15/110 files missing pattern",
				},
			},
		}, nil
	}
	result, err := CheckAgreements("/some/dir", false, checker)
	if err != nil {
		t.Fatalf("agreements warnings should not block spawn, got error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.HasFailures() {
		t.Error("expected HasFailures to be true")
	}
	if result.HasErrors() {
		t.Error("warning-only failures should not count as errors")
	}
}

func TestCheckAgreements_WithErrorFailures_StillWarning(t *testing.T) {
	// Phase 3 is warning-only — even error-severity failures don't block spawn
	checker := func(dir string) (*AgreementsResult, error) {
		return &AgreementsResult{
			Total:  5,
			Passed: 4,
			Failed: 1,
			Failures: []AgreementFailure{
				{
					AgreementID: "kb-agr-001",
					Title:       "KB directory initialized",
					Severity:    "error",
					Message:     "missing .kb/investigations",
				},
			},
		}, nil
	}
	result, err := CheckAgreements("/some/dir", false, checker)
	if err != nil {
		t.Fatalf("Phase 3: even error-severity agreements should not block spawn, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.HasErrors() {
		t.Error("expected HasErrors to be true for error-severity failures")
	}
}

func TestCheckAgreements_DaemonDrivenSilent(t *testing.T) {
	checker := func(dir string) (*AgreementsResult, error) {
		return &AgreementsResult{
			Total:  5,
			Passed: 4,
			Failed: 1,
			Failures: []AgreementFailure{
				{
					AgreementID: "kb-agr-002",
					Title:       "Investigations have status",
					Severity:    "warning",
					Message:     "15/110 files missing",
				},
			},
		}, nil
	}
	result, err := CheckAgreements("/some/dir", true, checker)
	if err != nil {
		t.Fatalf("daemon-driven should not error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result even for daemon-driven")
	}
	// Result should be returned even for daemon-driven (for telemetry)
	if !result.HasFailures() {
		t.Error("expected HasFailures for daemon-driven with failures")
	}
}

func TestCheckAgreements_CheckerReturnsError(t *testing.T) {
	// When checker fails (kb not installed, timeout), gracefully return nil
	checker := func(dir string) (*AgreementsResult, error) {
		return nil, fmt.Errorf("kb: command not found")
	}
	result, err := CheckAgreements("/some/dir", false, checker)
	if err != nil {
		t.Fatalf("checker failure should not block spawn, got: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result on checker failure, got: %v", result)
	}
}

func TestCheckAgreements_CheckerReturnsNil(t *testing.T) {
	checker := func(dir string) (*AgreementsResult, error) {
		return nil, nil
	}
	result, err := CheckAgreements("/some/dir", false, checker)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result when checker returns nil, got: %v", result)
	}
}

func TestCheckAgreements_MultipleFailures(t *testing.T) {
	checker := func(dir string) (*AgreementsResult, error) {
		return &AgreementsResult{
			Total:  5,
			Passed: 3,
			Failed: 2,
			Failures: []AgreementFailure{
				{
					AgreementID: "kb-agr-001",
					Title:       "KB directory initialized",
					Severity:    "error",
					Message:     "missing .kb/investigations",
				},
				{
					AgreementID: "kb-agr-002",
					Title:       "Investigations have status",
					Severity:    "warning",
					Message:     "5/20 files missing pattern",
				},
			},
		}, nil
	}
	result, err := CheckAgreements("/some/dir", false, checker)
	if err != nil {
		t.Fatalf("agreements should not block spawn, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Failures) != 2 {
		t.Errorf("expected 2 failures, got %d", len(result.Failures))
	}
	if !result.HasErrors() {
		t.Error("expected HasErrors when error-severity failure exists")
	}
	if !result.HasFailures() {
		t.Error("expected HasFailures")
	}
}

// --- AgreementsResult method tests ---

func TestAgreementsResult_HasFailures(t *testing.T) {
	tests := []struct {
		name     string
		result   AgreementsResult
		expected bool
	}{
		{"no failures", AgreementsResult{Total: 3, Passed: 3, Failed: 0}, false},
		{"with failures", AgreementsResult{Total: 3, Passed: 2, Failed: 1, Failures: []AgreementFailure{{Severity: "warning"}}}, true},
		{"empty failures slice", AgreementsResult{Total: 3, Passed: 3, Failed: 0, Failures: []AgreementFailure{}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.HasFailures(); got != tt.expected {
				t.Errorf("HasFailures() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAgreementsResult_HasErrors(t *testing.T) {
	tests := []struct {
		name     string
		result   AgreementsResult
		expected bool
	}{
		{"no failures", AgreementsResult{}, false},
		{"warning only", AgreementsResult{Failures: []AgreementFailure{{Severity: "warning"}}}, false},
		{"error present", AgreementsResult{Failures: []AgreementFailure{{Severity: "error"}}}, true},
		{"mixed", AgreementsResult{Failures: []AgreementFailure{{Severity: "warning"}, {Severity: "error"}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.HasErrors(); got != tt.expected {
				t.Errorf("HasErrors() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAgreementsResult_WarningCount(t *testing.T) {
	result := AgreementsResult{
		Failures: []AgreementFailure{
			{Severity: "warning"},
			{Severity: "error"},
			{Severity: "warning"},
		},
	}
	if got := result.WarningCount(); got != 2 {
		t.Errorf("WarningCount() = %d, want 2", got)
	}
}

func TestAgreementsResult_ErrorCount(t *testing.T) {
	result := AgreementsResult{
		Failures: []AgreementFailure{
			{Severity: "warning"},
			{Severity: "error"},
			{Severity: "error"},
		},
	}
	if got := result.ErrorCount(); got != 2 {
		t.Errorf("ErrorCount() = %d, want 2", got)
	}
}
