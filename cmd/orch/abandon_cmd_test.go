package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
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

// TestReadSpawnTimeFromWorkspace_RecentSpawnTriggersGracePeriod verifies that
// readSpawnTimeFromWorkspace correctly reads spawn time from the agent manifest,
// and that VerifyLiveness uses it to classify a recently-spawned agent with no
// phase comments as active (not dead). This is the integration point fixed by
// plumbing SpawnTime into checkRecentActivity.
func TestReadSpawnTimeFromWorkspace_RecentSpawnTriggersGracePeriod(t *testing.T) {
	// Create a temp workspace with a manifest containing a recent spawn time
	tmpDir := t.TempDir()
	recentSpawnTime := time.Now().Add(-2 * time.Minute) // 2 minutes ago

	manifest := spawn.AgentManifest{
		WorkspaceName: "test-agent",
		BeadsID:       "test-abc",
		SpawnTime:     recentSpawnTime.Format(time.RFC3339),
	}
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, spawn.AgentManifestFilename), manifestData, 0644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	// Read spawn time via the same path checkRecentActivity now uses
	spawnTime := readSpawnTimeFromWorkspace(tmpDir)
	if spawnTime.IsZero() {
		t.Fatal("Expected non-zero spawn time from workspace manifest")
	}

	// Verify that VerifyLiveness with this spawn time and no comments
	// classifies the agent as alive (recently spawned)
	liveness := verify.VerifyLiveness(verify.LivenessInput{
		Comments:  nil, // no phase comments yet
		SpawnTime: spawnTime,
		Now:       time.Now(),
	})

	if !liveness.IsAlive() {
		t.Fatalf("Expected recently-spawned agent to be classified as alive, got status=%s reason=%s",
			liveness.Status, liveness.Reason)
	}
	if liveness.Reason != verify.ReasonRecentlySpawned {
		t.Fatalf("Expected reason ReasonRecentlySpawned, got %s", liveness.Reason)
	}
}

// TestReadSpawnTimeFromWorkspace_EmptyPathReturnsZero verifies that
// readSpawnTimeFromWorkspace returns zero time when workspace path is empty,
// matching the fallback behavior (grace period doesn't fire).
func TestReadSpawnTimeFromWorkspace_EmptyPathReturnsZero(t *testing.T) {
	spawnTime := readSpawnTimeFromWorkspace("")
	if !spawnTime.IsZero() {
		t.Fatalf("Expected zero time for empty workspace path, got %v", spawnTime)
	}
}

// TestReadSpawnTimeFromWorkspace_OldSpawnNoGracePeriod verifies that an agent
// spawned long ago (> grace period) with no phase comments is classified as dead.
func TestReadSpawnTimeFromWorkspace_OldSpawnNoGracePeriod(t *testing.T) {
	tmpDir := t.TempDir()
	oldSpawnTime := time.Now().Add(-1 * time.Hour) // 1 hour ago

	manifest := spawn.AgentManifest{
		WorkspaceName: "test-agent",
		BeadsID:       "test-abc",
		SpawnTime:     oldSpawnTime.Format(time.RFC3339),
	}
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, spawn.AgentManifestFilename), manifestData, 0644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	spawnTime := readSpawnTimeFromWorkspace(tmpDir)

	liveness := verify.VerifyLiveness(verify.LivenessInput{
		Comments:  nil,
		SpawnTime: spawnTime,
		Now:       time.Now(),
	})

	if liveness.IsAlive() {
		t.Fatalf("Expected old agent with no phase comments to be classified as dead, got status=%s reason=%s",
			liveness.Status, liveness.Reason)
	}
}
