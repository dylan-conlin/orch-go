package main

import (
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/verify"
)

func TestCheckPhaseRecency_RecentPlanningBlocks(t *testing.T) {
	reportedAt := time.Now().Add(-5 * time.Minute)
	phase := verify.PhaseStatus{
		Phase:           "Planning",
		Summary:         "Analyzing codebase structure",
		Found:           true,
		PhaseReportedAt: &reportedAt,
	}

	err := checkPhaseRecency("test-abc", phase, time.Now())
	if err == nil {
		t.Fatal("Expected error for recent Planning phase, got nil")
	}

	if !strings.Contains(err.Error(), "appears to be actively running") {
		t.Errorf("Expected 'actively running' message, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "Planning") {
		t.Errorf("Expected phase name in error, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("Expected --force hint in error, got: %s", err.Error())
	}
}

func TestCheckPhaseRecency_RecentImplementingBlocks(t *testing.T) {
	reportedAt := time.Now().Add(-10 * time.Minute)
	phase := verify.PhaseStatus{
		Phase:           "Implementing",
		Summary:         "Adding auth middleware",
		Found:           true,
		PhaseReportedAt: &reportedAt,
	}

	err := checkPhaseRecency("test-xyz", phase, time.Now())
	if err == nil {
		t.Fatal("Expected error for recent Implementing phase, got nil")
	}
	if !strings.Contains(err.Error(), "Implementing") {
		t.Errorf("Expected 'Implementing' in error, got: %s", err.Error())
	}
}

func TestCheckPhaseRecency_OldPhaseAllows(t *testing.T) {
	reportedAt := time.Now().Add(-2 * time.Hour)
	phase := verify.PhaseStatus{
		Phase:           "Planning",
		Summary:         "Old activity",
		Found:           true,
		PhaseReportedAt: &reportedAt,
	}

	err := checkPhaseRecency("test-abc", phase, time.Now())
	if err != nil {
		t.Fatalf("Expected nil for old phase comment, got: %v", err)
	}
}

func TestCheckPhaseRecency_CompletePhaseAllows(t *testing.T) {
	reportedAt := time.Now().Add(-5 * time.Minute)
	phase := verify.PhaseStatus{
		Phase:           "Complete",
		Summary:         "All tests passing",
		Found:           true,
		PhaseReportedAt: &reportedAt,
	}

	err := checkPhaseRecency("test-abc", phase, time.Now())
	if err != nil {
		t.Fatalf("Expected nil for Phase: Complete, got: %v", err)
	}
}

func TestCheckPhaseRecency_CompleteCaseInsensitive(t *testing.T) {
	reportedAt := time.Now().Add(-2 * time.Minute)
	phase := verify.PhaseStatus{
		Phase:           "complete",
		Summary:         "Done",
		Found:           true,
		PhaseReportedAt: &reportedAt,
	}

	err := checkPhaseRecency("test-abc", phase, time.Now())
	if err != nil {
		t.Fatalf("Expected nil for 'complete' (lowercase), got: %v", err)
	}
}

func TestCheckPhaseRecency_NoPhaseFound(t *testing.T) {
	phase := verify.PhaseStatus{
		Found: false,
	}

	err := checkPhaseRecency("test-abc", phase, time.Now())
	if err != nil {
		t.Fatalf("Expected nil when no phase found, got: %v", err)
	}
}

func TestCheckPhaseRecency_NoTimestamp(t *testing.T) {
	phase := verify.PhaseStatus{
		Phase:           "Implementing",
		Summary:         "Working on it",
		Found:           true,
		PhaseReportedAt: nil,
	}

	err := checkPhaseRecency("test-abc", phase, time.Now())
	if err != nil {
		t.Fatalf("Expected nil when no timestamp available, got: %v", err)
	}
}

func TestCheckPhaseRecency_ExactThresholdBoundary(t *testing.T) {
	// At exactly 30 minutes, the elapsed time equals the threshold.
	// elapsed < threshold is false, so abandon should be allowed.
	reportedAt := time.Now().Add(-activeAgentThreshold)
	phase := verify.PhaseStatus{
		Phase:           "Implementing",
		Summary:         "Working",
		Found:           true,
		PhaseReportedAt: &reportedAt,
	}

	err := checkPhaseRecency("test-abc", phase, time.Now())
	if err != nil {
		t.Fatalf("Expected nil at exact threshold boundary, got: %v", err)
	}
}

func TestCheckPhaseRecency_JustUnderThreshold(t *testing.T) {
	// One second under threshold — should block
	reportedAt := time.Now().Add(-activeAgentThreshold + time.Second)
	phase := verify.PhaseStatus{
		Phase:           "Implementing",
		Summary:         "Working",
		Found:           true,
		PhaseReportedAt: &reportedAt,
	}

	err := checkPhaseRecency("test-abc", phase, time.Now())
	if err == nil {
		t.Fatal("Expected error just under threshold, got nil")
	}
}

func TestCheckPhaseRecency_ErrorContainsBeadsID(t *testing.T) {
	reportedAt := time.Now().Add(-5 * time.Minute)
	phase := verify.PhaseStatus{
		Phase:           "Planning",
		Summary:         "Starting work",
		Found:           true,
		PhaseReportedAt: &reportedAt,
	}

	err := checkPhaseRecency("scs-sp-f9u", phase, time.Now())
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "scs-sp-f9u") {
		t.Errorf("Expected beads ID in error, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "orch abandon scs-sp-f9u --force") {
		t.Errorf("Expected force command hint, got: %s", err.Error())
	}
}

func TestCheckPhaseRecency_BlockedPhaseBlocks(t *testing.T) {
	reportedAt := time.Now().Add(-3 * time.Minute)
	phase := verify.PhaseStatus{
		Phase:           "BLOCKED",
		Summary:         "Waiting on dependency",
		Found:           true,
		PhaseReportedAt: &reportedAt,
	}

	err := checkPhaseRecency("test-abc", phase, time.Now())
	if err == nil {
		t.Fatal("Expected error for recent BLOCKED phase, got nil")
	}
}
