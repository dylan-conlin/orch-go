package daemon

import (
	"testing"
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
