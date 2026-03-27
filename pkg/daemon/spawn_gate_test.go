package daemon

import (
	"fmt"
	"testing"
	"time"
)

// =============================================================================
// SpawnPipeline Tests
// =============================================================================

func TestSpawnPipeline_AllGatesAllow(t *testing.T) {
	pipeline := &SpawnPipeline{
		Gates: []SpawnGate{
			&alwaysAllowGate{name: "gate-1"},
			&alwaysAllowGate{name: "gate-2"},
		},
	}
	issue := &Issue{ID: "test-1", Title: "Test issue"}
	result := pipeline.Run(issue)

	if !result.Allowed {
		t.Errorf("pipeline.Run() Allowed = false, want true")
	}
	if result.RejectedBy != "" {
		t.Errorf("pipeline.Run() RejectedBy = %q, want empty", result.RejectedBy)
	}
	if len(result.GateResults) != 2 {
		t.Errorf("pipeline.Run() GateResults length = %d, want 2", len(result.GateResults))
	}
}

func TestSpawnPipeline_FirstRejectionShortCircuits(t *testing.T) {
	gate3Called := false
	pipeline := &SpawnPipeline{
		Gates: []SpawnGate{
			&alwaysAllowGate{name: "gate-1"},
			&alwaysRejectGate{name: "gate-2", message: "duplicate found"},
			&callTrackingGate{name: "gate-3", called: &gate3Called},
		},
	}
	issue := &Issue{ID: "test-1", Title: "Test issue"}
	result := pipeline.Run(issue)

	if result.Allowed {
		t.Error("pipeline.Run() Allowed = true, want false")
	}
	if result.RejectedBy != "gate-2" {
		t.Errorf("pipeline.Run() RejectedBy = %q, want 'gate-2'", result.RejectedBy)
	}
	if result.RejectionMessage != "duplicate found" {
		t.Errorf("pipeline.Run() RejectionMessage = %q, want 'duplicate found'", result.RejectionMessage)
	}
	if gate3Called {
		t.Error("gate-3 should not be called after gate-2 rejects")
	}
	// Only 2 gate results (gate-1 allow + gate-2 reject), gate-3 never ran
	if len(result.GateResults) != 2 {
		t.Errorf("pipeline.Run() GateResults length = %d, want 2", len(result.GateResults))
	}
}

func TestSpawnPipeline_FailOpenContinues(t *testing.T) {
	pipeline := &SpawnPipeline{
		Gates: []SpawnGate{
			&errorGate{name: "flaky-gate", failMode: FailOpen, err: fmt.Errorf("network timeout")},
			&alwaysAllowGate{name: "gate-2"},
		},
	}
	issue := &Issue{ID: "test-1", Title: "Test issue"}
	result := pipeline.Run(issue)

	if !result.Allowed {
		t.Error("pipeline.Run() Allowed = false, want true (fail-open gate should not block)")
	}
	if len(result.GateResults) != 2 {
		t.Errorf("pipeline.Run() GateResults length = %d, want 2", len(result.GateResults))
	}
}

func TestSpawnPipeline_FailFastRejects(t *testing.T) {
	pipeline := &SpawnPipeline{
		Gates: []SpawnGate{
			&errorGate{name: "critical-gate", failMode: FailFast, err: fmt.Errorf("beads unavailable")},
			&alwaysAllowGate{name: "gate-2"},
		},
	}
	issue := &Issue{ID: "test-1", Title: "Test issue"}
	result := pipeline.Run(issue)

	if result.Allowed {
		t.Error("pipeline.Run() Allowed = true, want false (fail-fast gate should block on error)")
	}
	if result.RejectedBy != "critical-gate" {
		t.Errorf("pipeline.Run() RejectedBy = %q, want 'critical-gate'", result.RejectedBy)
	}
}

func TestSpawnPipeline_AdvisoryChecksRunAfterGatesPass(t *testing.T) {
	pipeline := &SpawnPipeline{
		Gates: []SpawnGate{
			&alwaysAllowGate{name: "gate-1"},
		},
		AdvisoryChecks: []AdvisoryCheck{
			&staticAdvisory{name: "thrash-check", warning: "spawned 5 times"},
			&staticAdvisory{name: "clean-check", warning: ""},
		},
	}
	issue := &Issue{ID: "test-1", Title: "Test issue"}
	result := pipeline.Run(issue)

	if !result.Allowed {
		t.Error("pipeline.Run() Allowed = false, want true")
	}
	if len(result.Advisories) != 1 {
		t.Fatalf("pipeline.Run() Advisories length = %d, want 1 (only non-empty)", len(result.Advisories))
	}
	if result.Advisories[0].Name != "thrash-check" {
		t.Errorf("Advisories[0].Name = %q, want 'thrash-check'", result.Advisories[0].Name)
	}
	if result.Advisories[0].Warning != "spawned 5 times" {
		t.Errorf("Advisories[0].Warning = %q, want 'spawned 5 times'", result.Advisories[0].Warning)
	}
}

func TestSpawnPipeline_AdvisoryChecksSkippedOnRejection(t *testing.T) {
	advisoryCalled := false
	pipeline := &SpawnPipeline{
		Gates: []SpawnGate{
			&alwaysRejectGate{name: "blocker", message: "blocked"},
		},
		AdvisoryChecks: []AdvisoryCheck{
			&callTrackingAdvisory{name: "advisory", called: &advisoryCalled},
		},
	}
	issue := &Issue{ID: "test-1", Title: "Test issue"}
	result := pipeline.Run(issue)

	if result.Allowed {
		t.Error("pipeline.Run() should be rejected")
	}
	if advisoryCalled {
		t.Error("advisory checks should not run when pipeline is rejected")
	}
}

func TestSpawnPipeline_EmptyPipelineAllows(t *testing.T) {
	pipeline := &SpawnPipeline{}
	issue := &Issue{ID: "test-1", Title: "Test issue"}
	result := pipeline.Run(issue)

	if !result.Allowed {
		t.Error("empty pipeline should allow spawn")
	}
}

// =============================================================================
// SpawnTrackerGate Tests
// =============================================================================

func TestSpawnTrackerGate_AllowsUnknownIssue(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	gate := &SpawnTrackerGate{Tracker: tracker}
	issue := &Issue{ID: "test-1", Title: "Test"}

	result := gate.Check(issue)
	if result.Verdict != GateAllow {
		t.Errorf("SpawnTrackerGate.Check() = %v, want GateAllow for unknown issue", result.Verdict)
	}
}

func TestSpawnTrackerGate_RejectsRecentlySpawned(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	tracker.MarkSpawned("test-1")
	gate := &SpawnTrackerGate{Tracker: tracker}
	issue := &Issue{ID: "test-1", Title: "Test"}

	result := gate.Check(issue)
	if result.Verdict != GateReject {
		t.Errorf("SpawnTrackerGate.Check() = %v, want GateReject for recently spawned", result.Verdict)
	}
}

func TestSpawnTrackerGate_NilTracker(t *testing.T) {
	gate := &SpawnTrackerGate{Tracker: nil}
	issue := &Issue{ID: "test-1", Title: "Test"}

	result := gate.Check(issue)
	if result.Verdict != GateAllow {
		t.Errorf("SpawnTrackerGate.Check() with nil tracker = %v, want GateAllow", result.Verdict)
	}
}

// =============================================================================
// SessionDedupGate Tests
// =============================================================================

func TestSessionDedupGate_AllowsWhenNoSession(t *testing.T) {
	gate := &SessionDedupGate{CheckFunc: func(beadsID string) bool { return false }}
	issue := &Issue{ID: "test-1", Title: "Test"}

	result := gate.Check(issue)
	if result.Verdict != GateAllow {
		t.Errorf("SessionDedupGate.Check() = %v, want GateAllow when no session", result.Verdict)
	}
}

func TestSessionDedupGate_RejectsExistingSession(t *testing.T) {
	gate := &SessionDedupGate{CheckFunc: func(beadsID string) bool { return true }}
	issue := &Issue{ID: "test-1", Title: "Test"}

	result := gate.Check(issue)
	if result.Verdict != GateReject {
		t.Errorf("SessionDedupGate.Check() = %v, want GateReject when session exists", result.Verdict)
	}
}

// =============================================================================
// TitleDedupMemoryGate Tests
// =============================================================================

func TestTitleDedupMemoryGate_AllowsUniqueTitle(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	tracker.MarkSpawnedWithTitle("other-1", "Different title")
	gate := &TitleDedupMemoryGate{Tracker: tracker}
	issue := &Issue{ID: "test-1", Title: "Unique title"}

	result := gate.Check(issue)
	if result.Verdict != GateAllow {
		t.Errorf("TitleDedupMemoryGate.Check() = %v, want GateAllow for unique title", result.Verdict)
	}
}

func TestTitleDedupMemoryGate_RejectsDuplicateTitle(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	tracker.MarkSpawnedWithTitle("other-1", "Same title")
	gate := &TitleDedupMemoryGate{Tracker: tracker}
	issue := &Issue{ID: "test-2", Title: "Same title"}

	result := gate.Check(issue)
	if result.Verdict != GateReject {
		t.Errorf("TitleDedupMemoryGate.Check() = %v, want GateReject for duplicate title", result.Verdict)
	}
}

func TestTitleDedupMemoryGate_AllowsSameIssueID(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	tracker.MarkSpawnedWithTitle("test-1", "Same title")
	gate := &TitleDedupMemoryGate{Tracker: tracker}
	// Same issue ID, same title — this is NOT a duplicate
	issue := &Issue{ID: "test-1", Title: "Same title"}

	result := gate.Check(issue)
	if result.Verdict != GateAllow {
		t.Errorf("TitleDedupMemoryGate.Check() = %v, want GateAllow for same issue ID", result.Verdict)
	}
}

// =============================================================================
// TitleDedupBeadsGate Tests
// =============================================================================

func TestTitleDedupBeadsGate_AllowsWhenNoDuplicate(t *testing.T) {
	gate := &TitleDedupBeadsGate{FindFunc: func(title string) *Issue { return nil }}
	issue := &Issue{ID: "test-1", Title: "Unique"}

	result := gate.Check(issue)
	if result.Verdict != GateAllow {
		t.Errorf("TitleDedupBeadsGate.Check() = %v, want GateAllow", result.Verdict)
	}
}

func TestTitleDedupBeadsGate_RejectsDuplicate(t *testing.T) {
	gate := &TitleDedupBeadsGate{FindFunc: func(title string) *Issue {
		return &Issue{ID: "other-1", Title: title}
	}}
	issue := &Issue{ID: "test-1", Title: "Duplicate title"}

	result := gate.Check(issue)
	if result.Verdict != GateReject {
		t.Errorf("TitleDedupBeadsGate.Check() = %v, want GateReject", result.Verdict)
	}
}

func TestTitleDedupBeadsGate_AllowsSameIssueID(t *testing.T) {
	gate := &TitleDedupBeadsGate{FindFunc: func(title string) *Issue {
		return &Issue{ID: "test-1", Title: title}
	}}
	issue := &Issue{ID: "test-1", Title: "My title"}

	result := gate.Check(issue)
	if result.Verdict != GateAllow {
		t.Errorf("TitleDedupBeadsGate.Check() = %v, want GateAllow for same ID", result.Verdict)
	}
}

// =============================================================================
// FreshStatusGate Tests
// =============================================================================

func TestFreshStatusGate_AllowsOpenIssue(t *testing.T) {
	gate := &FreshStatusGate{
		GetStatusFunc: func(beadsID string) (string, error) { return "open", nil },
	}
	issue := &Issue{ID: "test-1", Title: "Test"}

	result := gate.Check(issue)
	if result.Verdict != GateAllow {
		t.Errorf("FreshStatusGate.Check() = %v, want GateAllow for open issue", result.Verdict)
	}
}

func TestFreshStatusGate_RejectsInProgressIssue(t *testing.T) {
	gate := &FreshStatusGate{
		GetStatusFunc: func(beadsID string) (string, error) { return "in_progress", nil },
	}
	issue := &Issue{ID: "test-1", Title: "Test"}

	result := gate.Check(issue)
	if result.Verdict != GateReject {
		t.Errorf("FreshStatusGate.Check() = %v, want GateReject for in_progress", result.Verdict)
	}
}

func TestFreshStatusGate_ErrorReturnsGateError(t *testing.T) {
	gate := &FreshStatusGate{
		GetStatusFunc: func(beadsID string) (string, error) {
			return "", fmt.Errorf("beads unavailable")
		},
	}
	issue := &Issue{ID: "test-1", Title: "Test"}

	result := gate.Check(issue)
	if result.Verdict != GateError {
		t.Errorf("FreshStatusGate.Check() = %v, want GateError", result.Verdict)
	}
	if result.Err == nil {
		t.Error("FreshStatusGate.Check() Err should be set on error")
	}
}

func TestFreshStatusGate_NilFuncAllows(t *testing.T) {
	gate := &FreshStatusGate{} // No func configured
	issue := &Issue{ID: "test-1", Title: "Test"}

	result := gate.Check(issue)
	if result.Verdict != GateAllow {
		t.Errorf("FreshStatusGate.Check() = %v, want GateAllow when no func configured", result.Verdict)
	}
}

func TestFreshStatusGate_CrossProject(t *testing.T) {
	gate := &FreshStatusGate{
		GetStatusFunc: func(beadsID string) (string, error) {
			t.Error("GetStatusFunc should not be called for cross-project issues")
			return "open", nil
		},
		GetStatusForProjectFunc: func(beadsID, projectDir string) (string, error) {
			if projectDir != "/tmp/other" {
				t.Errorf("GetStatusForProjectFunc projectDir = %q, want '/tmp/other'", projectDir)
			}
			return "in_progress", nil
		},
	}
	issue := &Issue{ID: "test-1", Title: "Test", ProjectDir: "/tmp/other"}

	result := gate.Check(issue)
	if result.Verdict != GateReject {
		t.Errorf("FreshStatusGate.Check() = %v, want GateReject for cross-project in_progress", result.Verdict)
	}
}

// =============================================================================
// SpawnCountAdvisory Tests
// =============================================================================

func TestSpawnCountAdvisory_NoWarningBelowThreshold(t *testing.T) {
	tracker := NewSpawnedIssueTrackerWithTTL(1 * time.Hour)
	tracker.MarkSpawned("test-1")
	tracker.MarkSpawned("test-1")
	advisory := &SpawnCountAdvisory{Tracker: tracker, Threshold: 3}

	warning := advisory.Check(&Issue{ID: "test-1"})
	if warning != "" {
		t.Errorf("SpawnCountAdvisory.Check() = %q, want empty (below threshold)", warning)
	}
}

func TestSpawnCountAdvisory_WarnsAtThreshold(t *testing.T) {
	tracker := NewSpawnedIssueTrackerWithTTL(1 * time.Hour)
	tracker.MarkSpawned("test-1")
	tracker.MarkSpawned("test-1")
	tracker.MarkSpawned("test-1")
	advisory := &SpawnCountAdvisory{Tracker: tracker, Threshold: 3}

	warning := advisory.Check(&Issue{ID: "test-1"})
	if warning == "" {
		t.Error("SpawnCountAdvisory.Check() should warn at threshold")
	}
}

func TestSpawnCountAdvisory_NilTracker(t *testing.T) {
	advisory := &SpawnCountAdvisory{Tracker: nil}
	warning := advisory.Check(&Issue{ID: "test-1"})
	if warning != "" {
		t.Errorf("SpawnCountAdvisory.Check() with nil tracker = %q, want empty", warning)
	}
}

// =============================================================================
// Integration: Pipeline with Real Gates
// =============================================================================

func TestSpawnPipeline_Integration_AllLayersPass(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	pipeline := &SpawnPipeline{
		Gates: []SpawnGate{
			&SpawnTrackerGate{Tracker: tracker},
			&SessionDedupGate{CheckFunc: func(beadsID string) bool { return false }},
			&TitleDedupMemoryGate{Tracker: tracker},
			&TitleDedupBeadsGate{FindFunc: func(title string) *Issue { return nil }},
			&FreshStatusGate{GetStatusFunc: func(beadsID string) (string, error) { return "open", nil }},
			&CommitDedupGate{HasCommitsFunc: func(beadsID string) bool { return false }},
			&KeywordDedupGate{FindOverlapFunc: func(title, selfID string) (bool, string) { return false, "" }},
		},
		AdvisoryChecks: []AdvisoryCheck{
			&SpawnCountAdvisory{Tracker: tracker, Threshold: 3},
		},
	}

	issue := &Issue{ID: "test-1", Title: "New feature"}
	result := pipeline.Run(issue)

	if !result.Allowed {
		t.Errorf("Pipeline rejected unexpectedly: %s by %s", result.RejectionMessage, result.RejectedBy)
	}
	if len(result.GateResults) != 7 {
		t.Errorf("GateResults length = %d, want 7", len(result.GateResults))
	}
}

func TestSpawnPipeline_Integration_SpawnTrackerRejectsEarly(t *testing.T) {
	tracker := NewSpawnedIssueTracker()
	tracker.MarkSpawned("test-1")

	sessionChecked := false
	pipeline := &SpawnPipeline{
		Gates: []SpawnGate{
			&SpawnTrackerGate{Tracker: tracker},
			&SessionDedupGate{CheckFunc: func(beadsID string) bool {
				sessionChecked = true
				return false
			}},
		},
	}

	issue := &Issue{ID: "test-1", Title: "Already spawned"}
	result := pipeline.Run(issue)

	if result.Allowed {
		t.Error("Pipeline should reject recently spawned issue")
	}
	if result.RejectedBy != "spawn-tracker" {
		t.Errorf("RejectedBy = %q, want 'spawn-tracker'", result.RejectedBy)
	}
	if sessionChecked {
		t.Error("Session dedup should not run after spawn tracker rejects")
	}
}

// =============================================================================
// Test Helpers (gate/advisory implementations for testing)
// =============================================================================

type alwaysAllowGate struct{ name string }

func (g *alwaysAllowGate) Name() string       { return g.name }
func (g *alwaysAllowGate) FailMode() FailMode  { return FailOpen }
func (g *alwaysAllowGate) Check(*Issue) GateResult {
	return GateResult{Gate: g.name, Verdict: GateAllow}
}

type alwaysRejectGate struct {
	name    string
	message string
}

func (g *alwaysRejectGate) Name() string       { return g.name }
func (g *alwaysRejectGate) FailMode() FailMode  { return FailOpen }
func (g *alwaysRejectGate) Check(*Issue) GateResult {
	return GateResult{Gate: g.name, Verdict: GateReject, Message: g.message}
}

type errorGate struct {
	name     string
	failMode FailMode
	err      error
}

func (g *errorGate) Name() string       { return g.name }
func (g *errorGate) FailMode() FailMode  { return g.failMode }
func (g *errorGate) Check(*Issue) GateResult {
	return GateResult{Gate: g.name, Verdict: GateError, Message: "check failed", Err: g.err}
}

type callTrackingGate struct {
	name   string
	called *bool
}

func (g *callTrackingGate) Name() string       { return g.name }
func (g *callTrackingGate) FailMode() FailMode  { return FailOpen }
func (g *callTrackingGate) Check(*Issue) GateResult {
	*g.called = true
	return GateResult{Gate: g.name, Verdict: GateAllow}
}

type staticAdvisory struct {
	name    string
	warning string
}

func (a *staticAdvisory) Name() string         { return a.name }
func (a *staticAdvisory) Check(*Issue) string { return a.warning }

type callTrackingAdvisory struct {
	name   string
	called *bool
}

func (a *callTrackingAdvisory) Name() string { return a.name }
func (a *callTrackingAdvisory) Check(*Issue) string {
	*a.called = true
	return "warning"
}
