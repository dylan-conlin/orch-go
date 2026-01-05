package session

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestRegistryRegister(t *testing.T) {
	tmpDir := t.TempDir()
	regPath := filepath.Join(tmpDir, "sessions.json")
	reg := NewRegistry(regPath)

	session := OrchestratorSession{
		WorkspaceName: "og-orch-ship-feature-05jan",
		SessionID:     "ses_abc123",
		ProjectDir:    "/Users/dylan/projects/orch-go",
		SpawnTime:     time.Now(),
		Goal:          "Ship the feature",
		Status:        "active",
	}

	if err := reg.Register(session); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(regPath); os.IsNotExist(err) {
		t.Error("Registry file was not created")
	}

	// Verify session can be retrieved
	got, err := reg.Get(session.WorkspaceName)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.WorkspaceName != session.WorkspaceName {
		t.Errorf("Get().WorkspaceName = %q, want %q", got.WorkspaceName, session.WorkspaceName)
	}
	if got.SessionID != session.SessionID {
		t.Errorf("Get().SessionID = %q, want %q", got.SessionID, session.SessionID)
	}
	if got.Goal != session.Goal {
		t.Errorf("Get().Goal = %q, want %q", got.Goal, session.Goal)
	}
}

func TestRegistryRegisterUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(filepath.Join(tmpDir, "sessions.json"))

	session := OrchestratorSession{
		WorkspaceName: "test-session",
		SessionID:     "ses_old",
		ProjectDir:    "/tmp",
		SpawnTime:     time.Now(),
		Goal:          "Old goal",
		Status:        "active",
	}

	if err := reg.Register(session); err != nil {
		t.Fatalf("Register() first error = %v", err)
	}

	// Register same workspace with different data (should update)
	session.SessionID = "ses_new"
	session.Goal = "New goal"

	if err := reg.Register(session); err != nil {
		t.Fatalf("Register() second error = %v", err)
	}

	// Verify update happened
	sessions, err := reg.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len(sessions) = %d, want 1 (should update, not add)", len(sessions))
	}
	if sessions[0].SessionID != "ses_new" {
		t.Errorf("SessionID = %q, want %q", sessions[0].SessionID, "ses_new")
	}
	if sessions[0].Goal != "New goal" {
		t.Errorf("Goal = %q, want %q", sessions[0].Goal, "New goal")
	}
}

func TestRegistryUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(filepath.Join(tmpDir, "sessions.json"))

	session := OrchestratorSession{
		WorkspaceName: "test-session",
		SessionID:     "ses_123",
		ProjectDir:    "/tmp",
		SpawnTime:     time.Now(),
		Goal:          "Original goal",
		Status:        "active",
	}

	if err := reg.Register(session); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Update the session
	err := reg.Update("test-session", func(s *OrchestratorSession) {
		s.Status = "completed"
		s.Goal = "Updated goal"
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	got, err := reg.Get("test-session")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Status != "completed" {
		t.Errorf("Status = %q, want %q", got.Status, "completed")
	}
	if got.Goal != "Updated goal" {
		t.Errorf("Goal = %q, want %q", got.Goal, "Updated goal")
	}
}

func TestRegistryUpdateNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(filepath.Join(tmpDir, "sessions.json"))

	err := reg.Update("nonexistent", func(s *OrchestratorSession) {
		s.Status = "completed"
	})
	if err != ErrSessionNotFound {
		t.Errorf("Update() error = %v, want ErrSessionNotFound", err)
	}
}

func TestRegistryUnregister(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(filepath.Join(tmpDir, "sessions.json"))

	// Register two sessions
	session1 := OrchestratorSession{
		WorkspaceName: "session-1",
		SessionID:     "ses_1",
		ProjectDir:    "/tmp",
		SpawnTime:     time.Now(),
		Goal:          "Goal 1",
		Status:        "active",
	}
	session2 := OrchestratorSession{
		WorkspaceName: "session-2",
		SessionID:     "ses_2",
		ProjectDir:    "/tmp",
		SpawnTime:     time.Now(),
		Goal:          "Goal 2",
		Status:        "active",
	}

	if err := reg.Register(session1); err != nil {
		t.Fatalf("Register() session1 error = %v", err)
	}
	if err := reg.Register(session2); err != nil {
		t.Fatalf("Register() session2 error = %v", err)
	}

	// Unregister first session
	if err := reg.Unregister("session-1"); err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}

	// Verify only second session remains
	sessions, err := reg.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len(sessions) = %d, want 1", len(sessions))
	}
	if sessions[0].WorkspaceName != "session-2" {
		t.Errorf("WorkspaceName = %q, want %q", sessions[0].WorkspaceName, "session-2")
	}
}

func TestRegistryUnregisterNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(filepath.Join(tmpDir, "sessions.json"))

	err := reg.Unregister("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("Unregister() error = %v, want ErrSessionNotFound", err)
	}
}

func TestRegistryList(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(filepath.Join(tmpDir, "sessions.json"))

	// Empty list initially
	sessions, err := reg.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("len(sessions) = %d, want 0", len(sessions))
	}

	// Add sessions
	for i := 0; i < 3; i++ {
		session := OrchestratorSession{
			WorkspaceName: "session-" + string(rune('a'+i)),
			SessionID:     "ses_" + string(rune('a'+i)),
			ProjectDir:    "/tmp",
			SpawnTime:     time.Now(),
			Goal:          "Goal",
			Status:        "active",
		}
		if err := reg.Register(session); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	sessions, err = reg.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(sessions) != 3 {
		t.Errorf("len(sessions) = %d, want 3", len(sessions))
	}
}

func TestRegistryGet(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(filepath.Join(tmpDir, "sessions.json"))

	session := OrchestratorSession{
		WorkspaceName: "test-session",
		SessionID:     "ses_xyz",
		ProjectDir:    "/home/user/project",
		SpawnTime:     time.Now(),
		Goal:          "Test goal",
		Status:        "active",
	}

	if err := reg.Register(session); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	got, err := reg.Get("test-session")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.WorkspaceName != session.WorkspaceName {
		t.Errorf("WorkspaceName = %q, want %q", got.WorkspaceName, session.WorkspaceName)
	}
	if got.SessionID != session.SessionID {
		t.Errorf("SessionID = %q, want %q", got.SessionID, session.SessionID)
	}
	if got.ProjectDir != session.ProjectDir {
		t.Errorf("ProjectDir = %q, want %q", got.ProjectDir, session.ProjectDir)
	}
}

func TestRegistryGetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(filepath.Join(tmpDir, "sessions.json"))

	_, err := reg.Get("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("Get() error = %v, want ErrSessionNotFound", err)
	}
}

func TestRegistryListActive(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(filepath.Join(tmpDir, "sessions.json"))

	// Register mix of active and completed sessions
	sessions := []OrchestratorSession{
		{WorkspaceName: "active-1", Status: "active", SessionID: "s1", ProjectDir: "/tmp", SpawnTime: time.Now()},
		{WorkspaceName: "completed-1", Status: "completed", SessionID: "s2", ProjectDir: "/tmp", SpawnTime: time.Now()},
		{WorkspaceName: "active-2", Status: "active", SessionID: "s3", ProjectDir: "/tmp", SpawnTime: time.Now()},
		{WorkspaceName: "abandoned-1", Status: "abandoned", SessionID: "s4", ProjectDir: "/tmp", SpawnTime: time.Now()},
	}

	for _, s := range sessions {
		if err := reg.Register(s); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	active, err := reg.ListActive()
	if err != nil {
		t.Fatalf("ListActive() error = %v", err)
	}
	if len(active) != 2 {
		t.Fatalf("len(active) = %d, want 2", len(active))
	}

	// Verify both are active
	for _, s := range active {
		if s.Status != "active" {
			t.Errorf("ListActive returned session with status %q", s.Status)
		}
	}
}

func TestRegistryListByProject(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(filepath.Join(tmpDir, "sessions.json"))

	sessions := []OrchestratorSession{
		{WorkspaceName: "proj1-a", ProjectDir: "/project1", SessionID: "s1", Status: "active", SpawnTime: time.Now()},
		{WorkspaceName: "proj2-a", ProjectDir: "/project2", SessionID: "s2", Status: "active", SpawnTime: time.Now()},
		{WorkspaceName: "proj1-b", ProjectDir: "/project1", SessionID: "s3", Status: "completed", SpawnTime: time.Now()},
	}

	for _, s := range sessions {
		if err := reg.Register(s); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	// Get project1 sessions
	proj1Sessions, err := reg.ListByProject("/project1")
	if err != nil {
		t.Fatalf("ListByProject() error = %v", err)
	}
	if len(proj1Sessions) != 2 {
		t.Fatalf("len(proj1Sessions) = %d, want 2", len(proj1Sessions))
	}

	// Get project2 sessions
	proj2Sessions, err := reg.ListByProject("/project2")
	if err != nil {
		t.Fatalf("ListByProject() error = %v", err)
	}
	if len(proj2Sessions) != 1 {
		t.Fatalf("len(proj2Sessions) = %d, want 1", len(proj2Sessions))
	}
}

func TestRegistryPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	regPath := filepath.Join(tmpDir, "sessions.json")

	// Create registry and add session
	reg1 := NewRegistry(regPath)
	session := OrchestratorSession{
		WorkspaceName: "persistent-session",
		SessionID:     "ses_persist",
		ProjectDir:    "/persistent/path",
		SpawnTime:     time.Now(),
		Goal:          "Persistent goal",
		Status:        "active",
	}

	if err := reg1.Register(session); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Create new registry from same file (simulating process restart)
	reg2 := NewRegistry(regPath)

	got, err := reg2.Get("persistent-session")
	if err != nil {
		t.Fatalf("Get() after reload error = %v", err)
	}
	if got.WorkspaceName != session.WorkspaceName {
		t.Errorf("WorkspaceName = %q after reload, want %q", got.WorkspaceName, session.WorkspaceName)
	}
	if got.Goal != session.Goal {
		t.Errorf("Goal = %q after reload, want %q", got.Goal, session.Goal)
	}
}

func TestRegistryConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewRegistry(filepath.Join(tmpDir, "sessions.json"))

	var wg sync.WaitGroup
	numGoroutines := 10

	// Concurrent registrations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			session := OrchestratorSession{
				WorkspaceName: "concurrent-" + string(rune('a'+i)),
				SessionID:     "ses_" + string(rune('a'+i)),
				ProjectDir:    "/tmp",
				SpawnTime:     time.Now(),
				Goal:          "Concurrent goal",
				Status:        "active",
			}
			if err := reg.Register(session); err != nil {
				t.Errorf("Concurrent Register() error = %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all sessions were registered
	sessions, err := reg.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(sessions) != numGoroutines {
		t.Errorf("len(sessions) = %d, want %d", len(sessions), numGoroutines)
	}
}

func TestRegistryDefaultPath(t *testing.T) {
	// Test that NewRegistry with empty path uses default
	reg := NewRegistry("")
	if reg.path != RegistryPath() {
		t.Errorf("path = %q, want %q", reg.path, RegistryPath())
	}
}

func TestRegistryEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	regPath := filepath.Join(tmpDir, "sessions.json")

	// Create empty file
	if err := os.WriteFile(regPath, []byte{}, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	reg := NewRegistry(regPath)

	// Should handle empty file gracefully
	sessions, err := reg.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("len(sessions) = %d for empty file, want 0", len(sessions))
	}
}

func TestRegistryStaleLockCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	regPath := filepath.Join(tmpDir, "sessions.json")
	lockPath := regPath + ".lock"
	reg := NewRegistry(regPath)

	// Create a stale lock file (older than 60 seconds)
	if err := os.WriteFile(lockPath, []byte("stale"), 0644); err != nil {
		t.Fatalf("WriteFile() lock error = %v", err)
	}
	// Backdate the lock file
	oldTime := time.Now().Add(-2 * time.Minute)
	if err := os.Chtimes(lockPath, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}

	// Should be able to acquire lock despite stale lock file
	session := OrchestratorSession{
		WorkspaceName: "test-after-stale-lock",
		SessionID:     "ses_123",
		ProjectDir:    "/tmp",
		SpawnTime:     time.Now(),
		Goal:          "Test",
		Status:        "active",
	}

	if err := reg.Register(session); err != nil {
		t.Fatalf("Register() with stale lock error = %v", err)
	}

	// Verify registration succeeded
	got, err := reg.Get("test-after-stale-lock")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got == nil {
		t.Fatal("Get() returned nil")
	}
}
