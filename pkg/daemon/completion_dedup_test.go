package daemon

import (
	"testing"
	"time"
)

func TestCompletionDedupTracker_Basic(t *testing.T) {
	tracker := NewCompletionDedupTracker()

	// Initially not completed
	if tracker.IsCompleted("proj-1", "summary A") {
		t.Error("expected false for untracked issue")
	}

	// Mark completed
	tracker.MarkCompleted("proj-1", "summary A")

	// Same issue + same summary = completed
	if !tracker.IsCompleted("proj-1", "summary A") {
		t.Error("expected true for same issue and summary")
	}

	// Same issue + different summary = not completed (reused for new task)
	if tracker.IsCompleted("proj-1", "summary B") {
		t.Error("expected false for different summary (issue reused)")
	}

	// Different issue = not completed
	if tracker.IsCompleted("proj-2", "summary A") {
		t.Error("expected false for different issue")
	}
}

func TestCompletionDedupTracker_Clear(t *testing.T) {
	tracker := NewCompletionDedupTracker()

	tracker.MarkCompleted("proj-1", "summary A")
	if !tracker.IsCompleted("proj-1", "summary A") {
		t.Fatal("expected true after marking")
	}

	tracker.Clear("proj-1")
	if tracker.IsCompleted("proj-1", "summary A") {
		t.Error("expected false after clearing")
	}
}

func TestCompletionDedupTracker_TTLExpiry(t *testing.T) {
	tracker := &CompletionDedupTracker{
		entries: make(map[string]completionEntry),
		ttl:     10 * time.Millisecond,
	}

	tracker.MarkCompleted("proj-1", "summary A")
	if !tracker.IsCompleted("proj-1", "summary A") {
		t.Fatal("expected true immediately after marking")
	}

	time.Sleep(15 * time.Millisecond)

	if tracker.IsCompleted("proj-1", "summary A") {
		t.Error("expected false after TTL expiry")
	}
}
