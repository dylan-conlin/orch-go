package focus

import (
	"path/filepath"
	"testing"
	"time"
)

// TestSetAndGetFocus verifies that setting and getting focus works.
func TestSetAndGetFocus(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "focus.json")

	store, err := New(path)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Initially no focus set
	f := store.Get()
	if f != nil {
		t.Errorf("expected nil focus initially, got %+v", f)
	}

	// Set focus
	focus := &Focus{
		Goal:    "Complete orch-go MVP",
		BeadsID: "orch-go-123",
	}
	if err := store.Set(focus); err != nil {
		t.Fatalf("failed to set focus: %v", err)
	}

	// Get focus
	got := store.Get()
	if got == nil {
		t.Fatal("expected focus to be set")
	}
	if got.Goal != "Complete orch-go MVP" {
		t.Errorf("expected goal 'Complete orch-go MVP', got %q", got.Goal)
	}
	if got.BeadsID != "orch-go-123" {
		t.Errorf("expected beads_id 'orch-go-123', got %q", got.BeadsID)
	}
	if got.SetAt == "" {
		t.Error("expected set_at to be populated")
	}
}

// TestFocusWithoutBeadsID verifies that focus can be set without a beads ID.
func TestFocusWithoutBeadsID(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "focus.json")

	store, err := New(path)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	focus := &Focus{
		Goal: "Learn Go testing patterns",
	}
	if err := store.Set(focus); err != nil {
		t.Fatalf("failed to set focus: %v", err)
	}

	got := store.Get()
	if got.Goal != "Learn Go testing patterns" {
		t.Errorf("expected goal 'Learn Go testing patterns', got %q", got.Goal)
	}
	if got.BeadsID != "" {
		t.Errorf("expected empty beads_id, got %q", got.BeadsID)
	}
}

// TestClearFocus verifies that focus can be cleared.
func TestClearFocus(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "focus.json")

	store, err := New(path)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Set focus
	focus := &Focus{Goal: "Some goal"}
	if err := store.Set(focus); err != nil {
		t.Fatalf("failed to set focus: %v", err)
	}

	// Clear focus
	if err := store.Clear(); err != nil {
		t.Fatalf("failed to clear focus: %v", err)
	}

	// Should be nil
	got := store.Get()
	if got != nil {
		t.Errorf("expected nil focus after clear, got %+v", got)
	}
}

// TestFocusPersistence verifies that focus persists across instances.
func TestFocusPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "focus.json")

	// Create and set focus
	store1, err := New(path)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	focus := &Focus{
		Goal:    "Persistent goal",
		BeadsID: "proj-456",
	}
	if err := store1.Set(focus); err != nil {
		t.Fatalf("failed to set focus: %v", err)
	}

	// Create new instance and verify persistence
	store2, err := New(path)
	if err != nil {
		t.Fatalf("failed to create second store: %v", err)
	}

	got := store2.Get()
	if got == nil {
		t.Fatal("expected focus to persist")
	}
	if got.Goal != "Persistent goal" {
		t.Errorf("expected goal 'Persistent goal', got %q", got.Goal)
	}
	if got.BeadsID != "proj-456" {
		t.Errorf("expected beads_id 'proj-456', got %q", got.BeadsID)
	}
}

// TestDefaultPath verifies that DefaultPath returns a sensible path.
func TestDefaultPath(t *testing.T) {
	path := DefaultPath()
	if path == "" {
		t.Error("DefaultPath should not be empty")
	}
}

// TestNewWithEmptyPath verifies that New with empty path uses default.
func TestNewWithEmptyPath(t *testing.T) {
	// Just verify it doesn't panic
	_, err := New("")
	if err != nil {
		t.Errorf("New with empty path should not error: %v", err)
	}
}

// TestFocusSetAtIsUpdated verifies that SetAt is updated when focus changes.
func TestFocusSetAtIsUpdated(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "focus.json")

	store, err := New(path)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Set initial focus
	focus1 := &Focus{Goal: "First goal"}
	if err := store.Set(focus1); err != nil {
		t.Fatalf("failed to set focus: %v", err)
	}
	firstSetAt := store.Get().SetAt

	// Small delay
	time.Sleep(10 * time.Millisecond)

	// Set new focus
	focus2 := &Focus{Goal: "Second goal"}
	if err := store.Set(focus2); err != nil {
		t.Fatalf("failed to set focus: %v", err)
	}
	secondSetAt := store.Get().SetAt

	if secondSetAt <= firstSetAt {
		t.Errorf("expected secondSetAt > firstSetAt, got %s <= %s", secondSetAt, firstSetAt)
	}
}

// TestCheckDrift verifies drift detection against active agents.
func TestCheckDrift(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "focus.json")

	store, err := New(path)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Set focus on a specific beads issue
	focus := &Focus{
		Goal:    "Complete feature X",
		BeadsID: "proj-123",
	}
	if err := store.Set(focus); err != nil {
		t.Fatalf("failed to set focus: %v", err)
	}

	// Case 1: Working on the focused issue - no drift
	activeIssues := []string{"proj-123"}
	drift := store.CheckDrift(activeIssues)
	if drift.IsDrifting {
		t.Error("expected no drift when working on focused issue")
	}
	if drift.FocusedIssue != "proj-123" {
		t.Errorf("expected focused issue 'proj-123', got %q", drift.FocusedIssue)
	}

	// Case 2: Working on different issue - drift detected
	activeIssues = []string{"proj-456"}
	drift = store.CheckDrift(activeIssues)
	if !drift.IsDrifting {
		t.Error("expected drift when working on different issue")
	}
	if len(drift.ActiveIssues) != 1 || drift.ActiveIssues[0] != "proj-456" {
		t.Errorf("expected active issues [proj-456], got %v", drift.ActiveIssues)
	}

	// Case 3: No active work - also considered drift
	activeIssues = []string{}
	drift = store.CheckDrift(activeIssues)
	if !drift.IsDrifting {
		t.Error("expected drift when no active work")
	}

	// Case 4: Focus includes active issue among others - no drift
	activeIssues = []string{"proj-456", "proj-123", "proj-789"}
	drift = store.CheckDrift(activeIssues)
	if drift.IsDrifting {
		t.Error("expected no drift when focused issue is among active issues")
	}
}

// TestCheckDriftNoFocus verifies drift check when no focus is set.
func TestCheckDriftNoFocus(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "focus.json")

	store, err := New(path)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// No focus set
	drift := store.CheckDrift([]string{"proj-123"})
	if drift.IsDrifting {
		t.Error("expected no drift when no focus is set")
	}
	if drift.FocusedIssue != "" {
		t.Errorf("expected empty focused issue, got %q", drift.FocusedIssue)
	}
}

// TestCheckDriftFocusWithoutBeadsID verifies drift when focus has no beads ID.
func TestCheckDriftFocusWithoutBeadsID(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "focus.json")

	store, err := New(path)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Set focus without beads ID
	focus := &Focus{
		Goal: "Learn something",
	}
	if err := store.Set(focus); err != nil {
		t.Fatalf("failed to set focus: %v", err)
	}

	// Any active work is fine when focus has no specific issue
	drift := store.CheckDrift([]string{"proj-123", "proj-456"})
	if drift.IsDrifting {
		t.Error("expected no drift when focus has no specific issue")
	}
}

// TestSuggestNext verifies the next action suggestion logic.
func TestSuggestNext(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "focus.json")

	store, err := New(path)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Case 1: No focus set
	suggestion := store.SuggestNext([]string{})
	if suggestion.Action != "set-focus" {
		t.Errorf("expected action 'set-focus', got %q", suggestion.Action)
	}

	// Set focus
	focus := &Focus{
		Goal:    "Complete proj-123",
		BeadsID: "proj-123",
	}
	if err := store.Set(focus); err != nil {
		t.Fatalf("failed to set focus: %v", err)
	}

	// Case 2: Focus set, no active work
	suggestion = store.SuggestNext([]string{})
	if suggestion.Action != "start-work" {
		t.Errorf("expected action 'start-work', got %q", suggestion.Action)
	}
	if suggestion.BeadsID != "proj-123" {
		t.Errorf("expected beads_id 'proj-123', got %q", suggestion.BeadsID)
	}

	// Case 3: Already working on focused issue
	suggestion = store.SuggestNext([]string{"proj-123"})
	if suggestion.Action != "continue" {
		t.Errorf("expected action 'continue', got %q", suggestion.Action)
	}

	// Case 4: Working on wrong issue (drift)
	suggestion = store.SuggestNext([]string{"proj-456"})
	if suggestion.Action != "refocus" {
		t.Errorf("expected action 'refocus', got %q", suggestion.Action)
	}
}
