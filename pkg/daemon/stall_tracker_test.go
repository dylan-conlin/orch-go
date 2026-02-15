package daemon

import (
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

func TestStallTracker_NoProgressDetection(t *testing.T) {
	tracker := NewStallTracker(1 * time.Second) // Use short threshold for testing

	sessionID := "test-session-1"
	tokens := &opencode.TokenStats{
		InputTokens:  1000,
		OutputTokens: 500,
	}

	// First update - should not be stalled (no previous snapshot)
	isStalled := tracker.Update(sessionID, tokens)
	if isStalled {
		t.Error("First update should not be stalled (no baseline)")
	}

	// Wait for threshold to pass
	time.Sleep(1100 * time.Millisecond)

	// Second update with SAME tokens - should be stalled
	isStalled = tracker.Update(sessionID, tokens)
	if !isStalled {
		t.Error("Agent with no token progress should be stalled after threshold")
	}
}

func TestStallTracker_ProgressDetection(t *testing.T) {
	tracker := NewStallTracker(1 * time.Second)

	sessionID := "test-session-2"
	tokens1 := &opencode.TokenStats{
		InputTokens:  1000,
		OutputTokens: 500,
	}

	// First update
	tracker.Update(sessionID, tokens1)

	// Wait for threshold
	time.Sleep(1100 * time.Millisecond)

	// Second update with INCREASED tokens - should NOT be stalled
	tokens2 := &opencode.TokenStats{
		InputTokens:  1000,
		OutputTokens: 600, // Increased by 100
	}
	isStalled := tracker.Update(sessionID, tokens2)
	if isStalled {
		t.Error("Agent making token progress should not be stalled")
	}
}

func TestStallTracker_GetStallDuration(t *testing.T) {
	tracker := NewStallTracker(500 * time.Millisecond)

	sessionID := "test-session-3"
	tokens := &opencode.TokenStats{
		InputTokens:  1000,
		OutputTokens: 500,
	}

	// First update
	tracker.Update(sessionID, tokens)

	// Wait a bit
	time.Sleep(600 * time.Millisecond)

	// Check stall duration
	duration := tracker.GetStallDuration(sessionID, tokens)
	if duration < 500*time.Millisecond {
		t.Errorf("Expected stall duration >= 500ms, got %v", duration)
	}
	if duration > 700*time.Millisecond {
		t.Errorf("Expected stall duration <= 700ms, got %v", duration)
	}
}

func TestStallTracker_CleanStale(t *testing.T) {
	tracker := NewStallTracker(1 * time.Second)

	sessionID := "test-session-4"
	tokens := &opencode.TokenStats{
		InputTokens:  1000,
		OutputTokens: 500,
	}

	// Add a snapshot
	tracker.Update(sessionID, tokens)

	// Verify snapshot exists
	if len(tracker.snapshots) != 1 {
		t.Errorf("Expected 1 snapshot, got %d", len(tracker.snapshots))
	}

	// Clean with maxAge=1ms should remove snapshots older than 1ms
	time.Sleep(2 * time.Millisecond)
	tracker.CleanStale(1 * time.Millisecond)

	// Verify snapshot was removed
	if len(tracker.snapshots) != 0 {
		t.Errorf("Expected 0 snapshots after cleanup, got %d", len(tracker.snapshots))
	}
}

func TestStallTracker_MultipleAgents(t *testing.T) {
	tracker := NewStallTracker(1 * time.Second)

	session1 := "test-session-5"
	session2 := "test-session-6"
	tokens := &opencode.TokenStats{
		InputTokens:  1000,
		OutputTokens: 500,
	}

	// Update both sessions
	tracker.Update(session1, tokens)
	tracker.Update(session2, tokens)

	// Verify both snapshots exist
	if len(tracker.snapshots) != 2 {
		t.Errorf("Expected 2 snapshots, got %d", len(tracker.snapshots))
	}

	// Wait for threshold
	time.Sleep(1100 * time.Millisecond)

	// Only session1 makes progress
	tokensProgress := &opencode.TokenStats{
		InputTokens:  1000,
		OutputTokens: 600,
	}
	stalled1 := tracker.Update(session1, tokensProgress)
	stalled2 := tracker.Update(session2, tokens)

	if stalled1 {
		t.Error("Session with progress should not be stalled")
	}
	if !stalled2 {
		t.Error("Session without progress should be stalled")
	}
}
