package main

import (
	"testing"
)

// TestDecisionGateBlocksTestFeature verifies that the decision gate
// correctly detects and blocks spawns with "test feature" in the task.
func TestDecisionGateBlocksTestFeature(t *testing.T) {
	task := "implement test feature"
	projectDir := "/Users/dylanconlin/Documents/personal/orch-go"
	acknowledgedDecision := "" // No acknowledgment

	result, err := checkDecisionConflicts(task, projectDir, acknowledgedDecision)

	// The gate should block this spawn
	if err == nil {
		t.Errorf("Expected error blocking spawn, but got nil")
	}

	if result == nil {
		t.Fatalf("Expected result, but got nil")
	}

	if !result.ConflictFound {
		t.Errorf("Expected ConflictFound to be true, but got false")
	}

	if result.Acknowledged {
		t.Errorf("Expected Acknowledged to be false (no acknowledgment provided), but got true")
	}

	t.Logf("Decision gate correctly blocked spawn: %v", err)
	t.Logf("Conflict details - ID: %s, Matched on: %s", result.DecisionID, result.MatchedOn)
}

// TestDecisionGateAllowsWithAcknowledgment verifies that the decision gate
// allows spawns when the decision is acknowledged.
func TestDecisionGateAllowsWithAcknowledgment(t *testing.T) {
	task := "implement test feature"
	projectDir := "/Users/dylanconlin/Documents/personal/orch-go"
	acknowledgedDecision := "2026-01-28-test-decision-gate" // Acknowledge the decision

	result, err := checkDecisionConflicts(task, projectDir, acknowledgedDecision)

	// The gate should allow the spawn with acknowledgment
	if err != nil {
		t.Errorf("Expected no error with acknowledgment, but got: %v", err)
	}

	if result == nil {
		t.Fatalf("Expected result, but got nil")
	}

	if !result.ConflictFound {
		t.Errorf("Expected ConflictFound to be true (conflict exists), but got false")
	}

	if !result.Acknowledged {
		t.Errorf("Expected Acknowledged to be true (decision was acknowledged), but got false")
	}

	t.Logf("Decision gate correctly allowed spawn with acknowledgment")
	t.Logf("Conflict details - ID: %s, Matched on: %s", result.DecisionID, result.MatchedOn)
}

// TestDecisionGateAllowsNonConflictingTask verifies that the decision gate
// allows spawns that don't match any blocking decisions.
func TestDecisionGateAllowsNonConflictingTask(t *testing.T) {
	task := "implement user authentication"
	projectDir := "/Users/dylanconlin/Documents/personal/orch-go"
	acknowledgedDecision := ""

	result, err := checkDecisionConflicts(task, projectDir, acknowledgedDecision)

	// The gate should allow this spawn (no conflict)
	if err != nil {
		t.Errorf("Expected no error for non-conflicting task, but got: %v", err)
	}

	if result == nil {
		t.Fatalf("Expected result, but got nil")
	}

	if result.ConflictFound {
		t.Errorf("Expected ConflictFound to be false (no conflict), but got true")
	}

	t.Logf("Decision gate correctly allowed non-conflicting spawn")
}
