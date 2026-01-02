package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionLifecycle(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.json")

	store, err := New(sessionPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Test initial state (no session)
	if store.IsActive() {
		t.Error("IsActive() = true, want false for new store")
	}
	if store.Duration() != 0 {
		t.Errorf("Duration() = %v, want 0", store.Duration())
	}

	// Test session start
	goal := "Ship feature X"
	if err := store.Start(goal); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !store.IsActive() {
		t.Error("IsActive() = false after Start, want true")
	}

	session := store.Get()
	if session == nil {
		t.Fatal("Get() returned nil after Start")
	}
	if session.Goal != goal {
		t.Errorf("session.Goal = %q, want %q", session.Goal, goal)
	}
	if len(session.Spawns) != 0 {
		t.Errorf("len(session.Spawns) = %d, want 0", len(session.Spawns))
	}

	// Test duration
	time.Sleep(10 * time.Millisecond)
	if store.Duration() < 10*time.Millisecond {
		t.Errorf("Duration() = %v, want >= 10ms", store.Duration())
	}

	// Test session end
	ended, err := store.End()
	if err != nil {
		t.Fatalf("End() error = %v", err)
	}
	if ended == nil {
		t.Fatal("End() returned nil session")
	}
	if ended.Goal != goal {
		t.Errorf("ended.Goal = %q, want %q", ended.Goal, goal)
	}

	if store.IsActive() {
		t.Error("IsActive() = true after End, want false")
	}
}

func TestRecordSpawn(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.json")

	store, err := New(sessionPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Test recording spawn with no active session (should be no-op)
	if err := store.RecordSpawn("test-123", "investigation", "task", "/tmp"); err != nil {
		t.Errorf("RecordSpawn() with no session error = %v", err)
	}
	if store.SpawnCount() != 0 {
		t.Errorf("SpawnCount() = %d after recording with no session, want 0", store.SpawnCount())
	}

	// Start session
	if err := store.Start("Test session"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Record spawns
	if err := store.RecordSpawn("test-123", "investigation", "investigate X", "/project1"); err != nil {
		t.Fatalf("RecordSpawn() error = %v", err)
	}
	if store.SpawnCount() != 1 {
		t.Errorf("SpawnCount() = %d, want 1", store.SpawnCount())
	}

	if err := store.RecordSpawn("test-456", "feature-impl", "implement Y", "/project2"); err != nil {
		t.Fatalf("RecordSpawn() error = %v", err)
	}
	if store.SpawnCount() != 2 {
		t.Errorf("SpawnCount() = %d, want 2", store.SpawnCount())
	}

	// Verify spawn records
	session := store.Get()
	if session == nil {
		t.Fatal("Get() returned nil")
	}
	if len(session.Spawns) != 2 {
		t.Fatalf("len(session.Spawns) = %d, want 2", len(session.Spawns))
	}

	spawn1 := session.Spawns[0]
	if spawn1.BeadsID != "test-123" {
		t.Errorf("spawn1.BeadsID = %q, want %q", spawn1.BeadsID, "test-123")
	}
	if spawn1.Skill != "investigation" {
		t.Errorf("spawn1.Skill = %q, want %q", spawn1.Skill, "investigation")
	}

	spawn2 := session.Spawns[1]
	if spawn2.BeadsID != "test-456" {
		t.Errorf("spawn2.BeadsID = %q, want %q", spawn2.BeadsID, "test-456")
	}
}

func TestPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.json")

	// Create and start session
	store1, err := New(sessionPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	goal := "Persistent goal"
	if err := store1.Start(goal); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := store1.RecordSpawn("persist-123", "feature-impl", "test", "/tmp"); err != nil {
		t.Fatalf("RecordSpawn() error = %v", err)
	}

	// Create new store from same file (simulating restart)
	store2, err := New(sessionPath)
	if err != nil {
		t.Fatalf("New() for reload error = %v", err)
	}

	if !store2.IsActive() {
		t.Error("IsActive() = false after reload, want true")
	}

	session := store2.Get()
	if session == nil {
		t.Fatal("Get() returned nil after reload")
	}
	if session.Goal != goal {
		t.Errorf("session.Goal = %q after reload, want %q", session.Goal, goal)
	}
	if len(session.Spawns) != 1 {
		t.Fatalf("len(session.Spawns) = %d after reload, want 1", len(session.Spawns))
	}
	if session.Spawns[0].BeadsID != "persist-123" {
		t.Errorf("spawn.BeadsID = %q after reload, want %q", session.Spawns[0].BeadsID, "persist-123")
	}
}

func TestSessionReplace(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.json")

	store, err := New(sessionPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Start first session
	if err := store.Start("First goal"); err != nil {
		t.Fatalf("Start() first error = %v", err)
	}
	if err := store.RecordSpawn("first-123", "inv", "first task", "/tmp"); err != nil {
		t.Fatalf("RecordSpawn() first error = %v", err)
	}

	// Start second session (should replace)
	if err := store.Start("Second goal"); err != nil {
		t.Fatalf("Start() second error = %v", err)
	}

	session := store.Get()
	if session == nil {
		t.Fatal("Get() returned nil")
	}
	if session.Goal != "Second goal" {
		t.Errorf("session.Goal = %q, want %q", session.Goal, "Second goal")
	}
	if len(session.Spawns) != 0 {
		t.Errorf("len(session.Spawns) = %d, want 0 (new session)", len(session.Spawns))
	}
}

func TestEndNoSession(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.json")

	store, err := New(sessionPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// End with no active session (should return nil, no error)
	ended, err := store.End()
	if err != nil {
		t.Errorf("End() with no session error = %v", err)
	}
	if ended != nil {
		t.Errorf("End() with no session returned %v, want nil", ended)
	}
}

func TestGetReturnsCopy(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.json")

	store, err := New(sessionPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := store.Start("Test goal"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := store.RecordSpawn("test-123", "inv", "task", "/tmp"); err != nil {
		t.Fatalf("RecordSpawn() error = %v", err)
	}

	// Get session
	session := store.Get()

	// Modify the returned session (should not affect stored session)
	session.Goal = "Modified"
	session.Spawns = append(session.Spawns, SpawnRecord{BeadsID: "fake"})

	// Get again
	session2 := store.Get()
	if session2.Goal != "Test goal" {
		t.Errorf("Modifying returned session affected store: Goal = %q", session2.Goal)
	}
	if len(session2.Spawns) != 1 {
		t.Errorf("Modifying returned session affected store: len(Spawns) = %d", len(session2.Spawns))
	}
}

func TestMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "nonexistent", "session.json")

	// New should create parent dirs on first save
	store, err := New(sessionPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if store.IsActive() {
		t.Error("IsActive() = true for nonexistent file, want false")
	}

	// Start session should create the file and parent directories
	if err := store.Start("Test"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		t.Error("Session file was not created after Start()")
	}
}
