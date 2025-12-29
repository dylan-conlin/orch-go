package sessions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultOrchestratorPath(t *testing.T) {
	path := DefaultOrchestratorPath()
	if path == "" {
		t.Error("DefaultOrchestratorPath returned empty string")
	}

	// Should end with .orch/session.json
	if !hasSuffix(path, ".orch/session.json") {
		t.Errorf("unexpected path: %s", path)
	}
}

func TestGenerateSessionID(t *testing.T) {
	id := GenerateSessionID()

	// Should start with sess_
	if !hasPrefix(id, "sess_") {
		t.Errorf("session ID should start with sess_, got: %s", id)
	}

	// Should be 20 characters: sess_ + YYYYMMDD + _ + HHMMSS
	if len(id) != 20 {
		t.Errorf("session ID should be 20 chars, got %d: %s", len(id), id)
	}
}

func TestNewOrchestratorStore(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "session.json")

	store, err := NewOrchestratorStore(path)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Should have no active session initially
	if store.Get() != nil {
		t.Error("expected no active session initially")
	}

	if store.IsActive() {
		t.Error("expected IsActive to be false initially")
	}
}

func TestOrchestratorStore_Start(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "session.json")

	store, err := NewOrchestratorStore(path)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Start a session
	session, err := store.Start("Ship snap MVP")
	if err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	// Verify session properties
	if session.ID == "" {
		t.Error("session ID should not be empty")
	}
	if session.Goal != "Ship snap MVP" {
		t.Errorf("unexpected goal: %s", session.Goal)
	}
	if session.Started.IsZero() {
		t.Error("started time should not be zero")
	}
	if len(session.Spawns) != 0 {
		t.Errorf("spawns should be empty, got %d", len(session.Spawns))
	}

	// Verify store state
	if !store.IsActive() {
		t.Error("expected IsActive to be true after start")
	}
	if store.Get().ID != session.ID {
		t.Error("Get() should return the started session")
	}

	// Verify persistence
	store2, err := NewOrchestratorStore(path)
	if err != nil {
		t.Fatalf("failed to reload store: %v", err)
	}
	if store2.Get() == nil {
		t.Error("session should persist to disk")
	}
	if store2.Get().ID != session.ID {
		t.Error("persisted session ID should match")
	}
}

func TestOrchestratorStore_End(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "session.json")

	store, err := NewOrchestratorStore(path)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Start and end a session
	session, _ := store.Start("Test goal")
	ended, err := store.End()
	if err != nil {
		t.Fatalf("failed to end session: %v", err)
	}

	// Verify returned session
	if ended.ID != session.ID {
		t.Error("ended session ID should match started session")
	}

	// Verify store state
	if store.IsActive() {
		t.Error("expected IsActive to be false after end")
	}
	if store.Get() != nil {
		t.Error("Get() should return nil after end")
	}

	// Verify ending non-existent session
	_, err = store.End()
	if err == nil {
		t.Error("ending non-existent session should fail")
	}
}

func TestOrchestratorStore_RecordSpawn(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "session.json")

	store, err := NewOrchestratorStore(path)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Record spawn without active session (should be no-op)
	err = store.RecordSpawn("beads-123", "investigation", "ses_abc")
	if err != nil {
		t.Errorf("RecordSpawn without session should not error: %v", err)
	}

	// Start session and record spawns
	store.Start("Test goal")

	err = store.RecordSpawn("beads-123", "investigation", "ses_abc")
	if err != nil {
		t.Fatalf("failed to record spawn: %v", err)
	}

	err = store.RecordSpawn("beads-456", "feature-impl", "ses_def")
	if err != nil {
		t.Fatalf("failed to record spawn: %v", err)
	}

	// Verify spawns recorded
	session := store.Get()
	if len(session.Spawns) != 2 {
		t.Errorf("expected 2 spawns, got %d", len(session.Spawns))
	}
	if session.Spawns[0].BeadsID != "beads-123" {
		t.Errorf("unexpected beads ID: %s", session.Spawns[0].BeadsID)
	}
	if session.Spawns[0].Skill != "investigation" {
		t.Errorf("unexpected skill: %s", session.Spawns[0].Skill)
	}
	if session.Spawns[1].BeadsID != "beads-456" {
		t.Errorf("unexpected beads ID: %s", session.Spawns[1].BeadsID)
	}
}

func TestOrchestratorStore_SetFocusID(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "session.json")

	store, err := NewOrchestratorStore(path)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// SetFocusID without session should fail
	err = store.SetFocusID("focus-123")
	if err == nil {
		t.Error("SetFocusID without session should fail")
	}

	// Start session and set focus ID
	store.Start("Test goal")
	err = store.SetFocusID("focus-123")
	if err != nil {
		t.Fatalf("failed to set focus ID: %v", err)
	}

	// Verify
	if store.Get().FocusID != "focus-123" {
		t.Errorf("unexpected focus ID: %s", store.Get().FocusID)
	}
}

func TestOrchestratorStore_Duration(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "session.json")

	store, err := NewOrchestratorStore(path)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Duration without session should be 0
	if store.Duration() != 0 {
		t.Error("Duration without session should be 0")
	}

	// Start session and check duration
	store.Start("Test")
	time.Sleep(10 * time.Millisecond)

	duration := store.Duration()
	if duration < 10*time.Millisecond {
		t.Errorf("Duration should be >= 10ms, got %v", duration)
	}
}

func TestOrchestratorStore_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "session.json")

	store, err := NewOrchestratorStore(path)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Start and clear session
	store.Start("Test")
	err = store.Clear()
	if err != nil {
		t.Fatalf("failed to clear: %v", err)
	}

	// Verify cleared
	if store.IsActive() {
		t.Error("expected IsActive to be false after clear")
	}
}

func TestOrchestratorStore_StartOverwritesPrevious(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "session.json")

	store, err := NewOrchestratorStore(path)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Start first session
	_, err = store.Start("Goal 1")
	if err != nil {
		t.Fatalf("failed to start first session: %v", err)
	}
	store.RecordSpawn("beads-1", "investigation", "ses_1")

	// Verify first session has spawn
	if len(store.Get().Spawns) != 1 {
		t.Errorf("first session should have 1 spawn, got %d", len(store.Get().Spawns))
	}

	// Start second session (should overwrite)
	session2, err := store.Start("Goal 2")
	if err != nil {
		t.Fatalf("failed to start second session: %v", err)
	}

	// Verify second session is active (checking goal is the key assertion)
	if store.Get().Goal != "Goal 2" {
		t.Errorf("unexpected goal: %s", store.Get().Goal)
	}
	// Session 2 should have the new goal
	if session2.Goal != "Goal 2" {
		t.Errorf("returned session should have Goal 2, got: %s", session2.Goal)
	}
	// Spawns should be empty (new session overwrites previous)
	if len(store.Get().Spawns) != 0 {
		t.Errorf("new session should have no spawns, got %d", len(store.Get().Spawns))
	}
}

func TestOrchestratorStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "session.json")

	// Create and populate store
	store, _ := NewOrchestratorStore(path)
	store.Start("Persist test")
	store.RecordSpawn("beads-persist", "investigation", "ses_persist")
	store.SetFocusID("focus-persist")

	// Read file directly to verify JSON format
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read session file: %v", err)
	}

	var storeData orchestratorStoreData
	if err := json.Unmarshal(data, &storeData); err != nil {
		t.Fatalf("failed to unmarshal session data: %v", err)
	}

	if storeData.Session == nil {
		t.Fatal("session should be present in file")
	}
	if storeData.Session.Goal != "Persist test" {
		t.Errorf("unexpected goal in file: %s", storeData.Session.Goal)
	}
	if len(storeData.Session.Spawns) != 1 {
		t.Errorf("expected 1 spawn in file, got %d", len(storeData.Session.Spawns))
	}
	if storeData.Session.FocusID != "focus-persist" {
		t.Errorf("unexpected focus ID in file: %s", storeData.Session.FocusID)
	}
}
