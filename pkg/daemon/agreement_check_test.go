package daemon

import (
	"fmt"
	"testing"
	"time"
)

func TestDaemon_ShouldRunAgreementCheck_Disabled(t *testing.T) {
	d := &Daemon{
		Config: Config{
			AgreementCheckEnabled:  false,
			AgreementCheckInterval: 30 * time.Minute,
		},
	}

	if d.ShouldRunAgreementCheck() {
		t.Error("ShouldRunAgreementCheck() should return false when disabled")
	}
}

func TestDaemon_ShouldRunAgreementCheck_ZeroInterval(t *testing.T) {
	d := &Daemon{
		Config: Config{
			AgreementCheckEnabled:  true,
			AgreementCheckInterval: 0,
		},
	}

	if d.ShouldRunAgreementCheck() {
		t.Error("ShouldRunAgreementCheck() should return false when interval is 0")
	}
}

func TestDaemon_ShouldRunAgreementCheck_NeverRun(t *testing.T) {
	d := &Daemon{
		Config: Config{
			AgreementCheckEnabled:  true,
			AgreementCheckInterval: 30 * time.Minute,
		},
	}

	if !d.ShouldRunAgreementCheck() {
		t.Error("ShouldRunAgreementCheck() should return true when never run before")
	}
}

func TestDaemon_ShouldRunAgreementCheck_IntervalElapsed(t *testing.T) {
	d := &Daemon{
		Config: Config{
			AgreementCheckEnabled:  true,
			AgreementCheckInterval: 30 * time.Minute,
		},
		lastAgreementCheck: time.Now().Add(-1 * time.Hour),
	}

	if !d.ShouldRunAgreementCheck() {
		t.Error("ShouldRunAgreementCheck() should return true when interval has elapsed")
	}
}

func TestDaemon_ShouldRunAgreementCheck_IntervalNotElapsed(t *testing.T) {
	d := &Daemon{
		Config: Config{
			AgreementCheckEnabled:  true,
			AgreementCheckInterval: 30 * time.Minute,
		},
		lastAgreementCheck: time.Now().Add(-10 * time.Minute),
	}

	if d.ShouldRunAgreementCheck() {
		t.Error("ShouldRunAgreementCheck() should return false when interval has not elapsed")
	}
}

func TestDaemon_RunPeriodicAgreementCheck_NotDue(t *testing.T) {
	called := false
	d := &Daemon{
		Config: Config{
			AgreementCheckEnabled:  true,
			AgreementCheckInterval: 30 * time.Minute,
		},
		lastAgreementCheck: time.Now(),
		AgreementCheck: &mockAgreementCheckService{CheckFunc: func() (*AgreementCheckResult, error) {
			called = true
			return &AgreementCheckResult{}, nil
		}},
	}

	result := d.RunPeriodicAgreementCheck()
	if result != nil {
		t.Error("RunPeriodicAgreementCheck() should return nil when not due")
	}
	if called {
		t.Error("AgreementCheck.Check should not be called when not due")
	}
}

func TestDaemon_RunPeriodicAgreementCheck_AllPassing(t *testing.T) {
	d := &Daemon{
		Config: Config{
			AgreementCheckEnabled:  true,
			AgreementCheckInterval: 30 * time.Minute,
		},
		AgreementCheck: &mockAgreementCheckService{CheckFunc: func() (*AgreementCheckResult, error) {
			return &AgreementCheckResult{
				Total:  3,
				Passed: 3,
				Failed: 0,
			}, nil
		}},
	}

	result := d.RunPeriodicAgreementCheck()
	if result == nil {
		t.Fatal("RunPeriodicAgreementCheck() should return result when due")
	}
	if result.Total != 3 {
		t.Errorf("Total = %d, want 3", result.Total)
	}
	if result.Passed != 3 {
		t.Errorf("Passed = %d, want 3", result.Passed)
	}
	if result.IssuesCreated != 0 {
		t.Errorf("IssuesCreated = %d, want 0", result.IssuesCreated)
	}
	if d.lastAgreementCheck.IsZero() {
		t.Error("lastAgreementCheck should be updated after running")
	}
}

func TestDaemon_RunPeriodicAgreementCheck_Error(t *testing.T) {
	d := &Daemon{
		Config: Config{
			AgreementCheckEnabled:  true,
			AgreementCheckInterval: 30 * time.Minute,
		},
		AgreementCheck: &mockAgreementCheckService{CheckFunc: func() (*AgreementCheckResult, error) {
			return nil, fmt.Errorf("kb agreements check failed")
		}},
	}

	result := d.RunPeriodicAgreementCheck()
	if result == nil {
		t.Fatal("RunPeriodicAgreementCheck() should return result on error")
	}
	if result.Error == nil {
		t.Error("Result should have error")
	}
}

func TestDaemon_RunPeriodicAgreementCheck_ErrorSeverityCreatesIssue(t *testing.T) {
	issueCalled := false
	d := &Daemon{
		Config: Config{
			AgreementCheckEnabled:  true,
			AgreementCheckInterval: 30 * time.Minute,
		},
		AgreementCheck: &mockAgreementCheckService{
			CheckFunc: func() (*AgreementCheckResult, error) {
				return &AgreementCheckResult{
					Total:  2,
					Passed: 1,
					Failed: 1,
					Failures: []AgreementFailureDetail{
						{
							AgreementID: "test-agreement",
							Title:       "Test Agreement",
							Severity:    "error",
							Message:     "Contract violated",
						},
					},
				}, nil
			},
			CreateIssueFunc: func(failure AgreementFailureDetail) error {
				issueCalled = true
				if failure.AgreementID != "test-agreement" {
					t.Errorf("CreateIssue called with wrong agreement ID: %s", failure.AgreementID)
				}
				return nil
			},
			HasOpenIssueFunc: func(agreementID string) (bool, error) {
				return false, nil
			},
		},
	}

	result := d.RunPeriodicAgreementCheck()
	if result == nil {
		t.Fatal("RunPeriodicAgreementCheck() should return result")
	}
	if !issueCalled {
		t.Error("CreateIssue should be called for error-severity failure")
	}
	if result.IssuesCreated != 1 {
		t.Errorf("IssuesCreated = %d, want 1", result.IssuesCreated)
	}
}

func TestDaemon_RunPeriodicAgreementCheck_WarningSeveritySkipped(t *testing.T) {
	issueCalled := false
	d := &Daemon{
		Config: Config{
			AgreementCheckEnabled:  true,
			AgreementCheckInterval: 30 * time.Minute,
		},
		AgreementCheck: &mockAgreementCheckService{
			CheckFunc: func() (*AgreementCheckResult, error) {
				return &AgreementCheckResult{
					Total:  2,
					Passed: 1,
					Failed: 1,
					Failures: []AgreementFailureDetail{
						{
							AgreementID: "warn-agreement",
							Title:       "Warning Agreement",
							Severity:    "warning",
							Message:     "Something is off",
						},
					},
				}, nil
			},
			CreateIssueFunc: func(failure AgreementFailureDetail) error {
				issueCalled = true
				return nil
			},
		},
	}

	result := d.RunPeriodicAgreementCheck()
	if result == nil {
		t.Fatal("RunPeriodicAgreementCheck() should return result")
	}
	if issueCalled {
		t.Error("CreateIssue should NOT be called for warning-severity failure")
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Skipped)
	}
	if result.IssuesCreated != 0 {
		t.Errorf("IssuesCreated = %d, want 0", result.IssuesCreated)
	}
}

func TestDaemon_RunPeriodicAgreementCheck_DedupSkips(t *testing.T) {
	issueCalled := false
	d := &Daemon{
		Config: Config{
			AgreementCheckEnabled:  true,
			AgreementCheckInterval: 30 * time.Minute,
		},
		AgreementCheck: &mockAgreementCheckService{
			CheckFunc: func() (*AgreementCheckResult, error) {
				return &AgreementCheckResult{
					Total:  2,
					Passed: 1,
					Failed: 1,
					Failures: []AgreementFailureDetail{
						{
							AgreementID: "existing-agreement",
							Title:       "Existing Agreement",
							Severity:    "error",
							Message:     "Still failing",
						},
					},
				}, nil
			},
			HasOpenIssueFunc: func(agreementID string) (bool, error) {
				return true, nil // Already has open issue
			},
			CreateIssueFunc: func(failure AgreementFailureDetail) error {
				issueCalled = true
				return nil
			},
		},
	}

	result := d.RunPeriodicAgreementCheck()
	if result == nil {
		t.Fatal("RunPeriodicAgreementCheck() should return result")
	}
	if issueCalled {
		t.Error("CreateIssue should NOT be called when open issue already exists")
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Skipped)
	}
}

func TestDaemon_RunPeriodicAgreementCheck_AutoFixOverride(t *testing.T) {
	issueCalled := false
	autoFixTrue := true
	d := &Daemon{
		Config: Config{
			AgreementCheckEnabled:  true,
			AgreementCheckInterval: 30 * time.Minute,
		},
		AgreementCheck: &mockAgreementCheckService{
			CheckFunc: func() (*AgreementCheckResult, error) {
				return &AgreementCheckResult{
					Total:  1,
					Passed: 0,
					Failed: 1,
					Failures: []AgreementFailureDetail{
						{
							AgreementID: "warn-with-autofix",
							Title:       "Warning With AutoFix",
							Severity:    "warning",
							AutoFix:     &autoFixTrue,
							Message:     "Can be auto-fixed",
						},
					},
				}, nil
			},
			HasOpenIssueFunc: func(agreementID string) (bool, error) {
				return false, nil
			},
			CreateIssueFunc: func(failure AgreementFailureDetail) error {
				issueCalled = true
				return nil
			},
		},
	}

	result := d.RunPeriodicAgreementCheck()
	if result == nil {
		t.Fatal("RunPeriodicAgreementCheck() should return result")
	}
	if !issueCalled {
		t.Error("CreateIssue should be called for warning with auto_fix=true")
	}
	if result.IssuesCreated != 1 {
		t.Errorf("IssuesCreated = %d, want 1", result.IssuesCreated)
	}
}

func TestDaemon_RunPeriodicAgreementCheck_AutoFixFalseOverridesError(t *testing.T) {
	issueCalled := false
	autoFixFalse := false
	d := &Daemon{
		Config: Config{
			AgreementCheckEnabled:  true,
			AgreementCheckInterval: 30 * time.Minute,
		},
		AgreementCheck: &mockAgreementCheckService{
			CheckFunc: func() (*AgreementCheckResult, error) {
				return &AgreementCheckResult{
					Total:  1,
					Passed: 0,
					Failed: 1,
					Failures: []AgreementFailureDetail{
						{
							AgreementID: "error-no-autofix",
							Title:       "Error No AutoFix",
							Severity:    "error",
							AutoFix:     &autoFixFalse,
							Message:     "Needs manual fix",
						},
					},
				}, nil
			},
			CreateIssueFunc: func(failure AgreementFailureDetail) error {
				issueCalled = true
				return nil
			},
		},
	}

	result := d.RunPeriodicAgreementCheck()
	if result == nil {
		t.Fatal("RunPeriodicAgreementCheck() should return result")
	}
	if issueCalled {
		t.Error("CreateIssue should NOT be called for error with auto_fix=false")
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Skipped)
	}
}

func TestShouldAutoCreate_ErrorSeverity(t *testing.T) {
	if !shouldAutoCreate(AgreementFailureDetail{Severity: "error"}) {
		t.Error("error severity should auto-create by default")
	}
}

func TestShouldAutoCreate_WarningSeverity(t *testing.T) {
	if shouldAutoCreate(AgreementFailureDetail{Severity: "warning"}) {
		t.Error("warning severity should not auto-create by default")
	}
}

func TestShouldAutoCreate_InfoSeverity(t *testing.T) {
	if shouldAutoCreate(AgreementFailureDetail{Severity: "info"}) {
		t.Error("info severity should not auto-create by default")
	}
}

func TestShouldAutoCreate_AutoFixTrueOverridesWarning(t *testing.T) {
	autoFix := true
	if !shouldAutoCreate(AgreementFailureDetail{Severity: "warning", AutoFix: &autoFix}) {
		t.Error("auto_fix=true should override warning severity to auto-create")
	}
}

func TestShouldAutoCreate_AutoFixFalseOverridesError(t *testing.T) {
	autoFix := false
	if shouldAutoCreate(AgreementFailureDetail{Severity: "error", AutoFix: &autoFix}) {
		t.Error("auto_fix=false should override error severity to not auto-create")
	}
}

func TestAgreementCheckResult_Snapshot(t *testing.T) {
	result := &AgreementCheckResult{
		Total:         5,
		Passed:        3,
		Failed:        2,
		IssuesCreated: 1,
	}

	snapshot := result.Snapshot()
	if snapshot.Total != 5 {
		t.Errorf("Snapshot.Total = %d, want 5", snapshot.Total)
	}
	if snapshot.Passed != 3 {
		t.Errorf("Snapshot.Passed = %d, want 3", snapshot.Passed)
	}
	if snapshot.Failed != 2 {
		t.Errorf("Snapshot.Failed = %d, want 2", snapshot.Failed)
	}
	if snapshot.IssuesCreated != 1 {
		t.Errorf("Snapshot.IssuesCreated = %d, want 1", snapshot.IssuesCreated)
	}
	if snapshot.LastCheck.IsZero() {
		t.Error("Snapshot.LastCheck should not be zero")
	}
}

func TestDefaultConfig_IncludesAgreementCheck(t *testing.T) {
	config := DefaultConfig()

	if !config.AgreementCheckEnabled {
		t.Error("DefaultConfig().AgreementCheckEnabled should be true")
	}
	if config.AgreementCheckInterval != 30*time.Minute {
		t.Errorf("DefaultConfig().AgreementCheckInterval = %v, want 30m", config.AgreementCheckInterval)
	}
}
