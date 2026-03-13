package daemon

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/daemonconfig"
)

func TestCheckPreSpawnGates_AllPass(t *testing.T) {
	d := &Daemon{}
	signal := d.CheckPreSpawnGates()
	if !signal.Allowed {
		t.Errorf("CheckPreSpawnGates() Allowed = false, want true; Reason: %s", signal.Reason)
	}
}

func TestCheckPreSpawnGates_VerificationPaused(t *testing.T) {
	tracker := NewVerificationTracker(1) // threshold = 1
	tracker.RecordCompletion("some-agent")

	d := &Daemon{VerificationTracker: tracker}
	signal := d.CheckPreSpawnGates()
	if signal.Allowed {
		t.Error("CheckPreSpawnGates() should block when verification is paused")
	}
	if signal.Reason == "" {
		t.Error("CheckPreSpawnGates() should provide a reason when blocked")
	}
}

func TestCheckPreSpawnGates_CompletionHealthFailed(t *testing.T) {
	tracker := NewCompletionFailureTracker()
	tracker.RecordFailure("error 1")
	tracker.RecordFailure("error 2")
	tracker.RecordFailure("error 3")

	d := &Daemon{CompletionFailureTracker: tracker}
	signal := d.CheckPreSpawnGates()
	if signal.Allowed {
		t.Error("CheckPreSpawnGates() should block when completion health is bad")
	}
}

func TestCheckPreSpawnGates_RateLimited(t *testing.T) {
	limiter := NewRateLimiter(1)
	limiter.RecordSpawn()

	d := &Daemon{RateLimiter: limiter}
	signal := d.CheckPreSpawnGates()
	if signal.Allowed {
		t.Error("CheckPreSpawnGates() should block when rate limited")
	}
}

func TestCheckPreSpawnGates_ShortCircuits(t *testing.T) {
	// Verification pause should be checked first, even if rate limiter allows
	tracker := NewVerificationTracker(1)
	tracker.RecordCompletion("some-agent")

	d := &Daemon{
		VerificationTracker: tracker,
		RateLimiter:         NewRateLimiter(100), // plenty of capacity
	}
	signal := d.CheckPreSpawnGates()
	if signal.Allowed {
		t.Error("CheckPreSpawnGates() should short-circuit on verification pause")
	}
}

func TestCheckIssueCompliance_PassesCleanIssue(t *testing.T) {
	d := &Daemon{}
	issue := Issue{ID: "proj-1", IssueType: "feature", Status: "open"}
	result := d.CheckIssueCompliance(issue, nil, nil)
	if !result.Passed {
		t.Errorf("CheckIssueCompliance() Passed = false for clean issue; Reason: %s", result.Reason)
	}
}

func TestCheckIssueCompliance_SkipSet(t *testing.T) {
	d := &Daemon{}
	issue := Issue{ID: "proj-1", IssueType: "feature", Status: "open"}
	skip := map[string]bool{"proj-1": true}
	result := d.CheckIssueCompliance(issue, skip, nil)
	if result.Passed {
		t.Error("CheckIssueCompliance() should filter issue in skip set")
	}
}

func TestCheckIssueCompliance_RecentlySpawned(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	tracker.MarkSpawned("proj-1")

	d := &Daemon{SpawnedIssues: tracker}
	issue := Issue{ID: "proj-1", IssueType: "feature", Status: "open"}
	result := d.CheckIssueCompliance(issue, nil, nil)
	if result.Passed {
		t.Error("CheckIssueCompliance() should filter recently spawned issue")
	}
}

func TestCheckIssueCompliance_NonSpawnableType(t *testing.T) {
	d := &Daemon{}
	issue := Issue{ID: "proj-1", IssueType: "epic", Status: "open"}
	result := d.CheckIssueCompliance(issue, nil, nil)
	if result.Passed {
		t.Error("CheckIssueCompliance() should filter non-spawnable type")
	}
}

func TestCheckIssueCompliance_BlockedStatus(t *testing.T) {
	d := &Daemon{}
	issue := Issue{ID: "proj-1", IssueType: "feature", Status: "blocked"}
	result := d.CheckIssueCompliance(issue, nil, nil)
	if result.Passed {
		t.Error("CheckIssueCompliance() should filter blocked issues")
	}
}

func TestCheckIssueCompliance_InProgressStatus(t *testing.T) {
	d := &Daemon{}
	issue := Issue{ID: "proj-1", IssueType: "feature", Status: "in_progress"}
	result := d.CheckIssueCompliance(issue, nil, nil)
	if result.Passed {
		t.Error("CheckIssueCompliance() should filter in_progress issues")
	}
}

func TestCheckIssueCompliance_CompletionLabels(t *testing.T) {
	d := &Daemon{}

	for _, label := range []string{LabelReadyReview, LabelVerificationFailed} {
		issue := Issue{ID: "proj-1", IssueType: "feature", Status: "open", Labels: []string{label}}
		result := d.CheckIssueCompliance(issue, nil, nil)
		if result.Passed {
			t.Errorf("CheckIssueCompliance() should filter issue with label %s", label)
		}
	}
}

func TestCheckIssueCompliance_LabelMismatch(t *testing.T) {
	d := &Daemon{Config: Config{Label: "triage:ready"}}
	issue := Issue{ID: "proj-1", IssueType: "feature", Status: "open", Labels: []string{"other-label"}}
	result := d.CheckIssueCompliance(issue, nil, nil)
	if result.Passed {
		t.Error("CheckIssueCompliance() should filter issue missing required label")
	}
}

func TestCheckIssueCompliance_EpicChildExemptFromLabel(t *testing.T) {
	d := &Daemon{Config: Config{Label: "triage:ready"}}
	issue := Issue{ID: "proj-1", IssueType: "feature", Status: "open"}
	epicChildIDs := map[string]bool{"proj-1": true}
	result := d.CheckIssueCompliance(issue, nil, epicChildIDs)
	if !result.Passed {
		t.Errorf("CheckIssueCompliance() should allow epic child without label; Reason: %s", result.Reason)
	}
}

// --- Compliance-level-aware gate tests ---

func TestNewWithConfig_ComplianceDerivedVerificationThreshold(t *testing.T) {
	// When compliance default is "relaxed", verification threshold should be derived (20)
	cfg := DefaultConfig()
	cfg.Compliance = daemonconfig.ComplianceConfig{Default: daemonconfig.ComplianceRelaxed}
	d := NewWithConfig(cfg)

	status := d.VerificationTracker.Status()
	want := daemonconfig.DeriveVerificationThreshold(daemonconfig.ComplianceRelaxed)
	if status.Threshold != want {
		t.Errorf("VerificationTracker threshold = %d, want %d (derived from relaxed)", status.Threshold, want)
	}
}

func TestNewWithConfig_ComplianceDerivedInvariantThreshold(t *testing.T) {
	// When compliance default is "standard", invariant threshold should be derived (5)
	cfg := DefaultConfig()
	cfg.Compliance = daemonconfig.ComplianceConfig{Default: daemonconfig.ComplianceStandard}
	d := NewWithConfig(cfg)

	if d.InvariantChecker == nil {
		t.Fatal("InvariantChecker should be initialized")
	}
	// The invariant checker's threshold is internal; test via behavior.
	// With standard level, threshold is 5. Record 4 violation cycles — should not pause.
	for i := 0; i < 4; i++ {
		d.InvariantChecker.Check(&InvariantInput{
			ActiveCount: -1, // trigger a violation
			MaxAgents:   5,
		})
	}
	if d.InvariantChecker.IsPaused() {
		t.Error("InvariantChecker should NOT be paused after 4 violations with standard threshold (5)")
	}
	// 5th violation should trigger pause
	d.InvariantChecker.Check(&InvariantInput{
		ActiveCount: -1,
		MaxAgents:   5,
	})
	if !d.InvariantChecker.IsPaused() {
		t.Error("InvariantChecker should be paused after 5 violations with standard threshold (5)")
	}
}

func TestNewWithConfig_ComplianceStrictPreservesDefaults(t *testing.T) {
	// With strict compliance (default), thresholds should match the derive functions
	cfg := DefaultConfig()
	// No compliance override — defaults to strict
	d := NewWithConfig(cfg)

	status := d.VerificationTracker.Status()
	want := daemonconfig.DeriveVerificationThreshold(daemonconfig.ComplianceStrict)
	if status.Threshold != want {
		t.Errorf("VerificationTracker threshold = %d, want %d (derived from strict)", status.Threshold, want)
	}
}

func TestNewWithConfig_ComplianceAutonomousDisablesVerification(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Compliance = daemonconfig.ComplianceConfig{Default: daemonconfig.ComplianceAutonomous}
	d := NewWithConfig(cfg)

	status := d.VerificationTracker.Status()
	if status.Threshold != 0 {
		t.Errorf("VerificationTracker threshold = %d, want 0 (autonomous disables)", status.Threshold)
	}
}
