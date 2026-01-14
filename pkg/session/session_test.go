package session

import (
	"os"
	"path/filepath"
	"strings"
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

func TestGetCheckpointStatus(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.json")

	store, err := New(sessionPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Test with no active session
	status := store.GetCheckpointStatus()
	if status != nil {
		t.Error("GetCheckpointStatus() should return nil when no session is active")
	}

	// Start session
	if err := store.Start("Test checkpoint"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Test immediate checkpoint status (should be "ok")
	status = store.GetCheckpointStatus()
	if status == nil {
		t.Fatal("GetCheckpointStatus() returned nil for active session")
	}
	if status.Level != "ok" {
		t.Errorf("status.Level = %q, want 'ok' for new session", status.Level)
	}
	if status.Duration < 0 {
		t.Errorf("status.Duration = %v, should be >= 0", status.Duration)
	}
	if status.NextThreshold <= 0 {
		t.Errorf("status.NextThreshold = %v, should be > 0 for new session", status.NextThreshold)
	}
}

func TestCheckpointStatusLevels(t *testing.T) {
	// Test the threshold logic directly by examining the constants
	tests := []struct {
		duration  time.Duration
		wantLevel string
	}{
		{30 * time.Minute, "ok"},
		{90 * time.Minute, "ok"},
		{2*time.Hour + 1*time.Minute, "warning"},
		{2*time.Hour + 30*time.Minute, "warning"},
		{3*time.Hour + 1*time.Minute, "strong"},
		{3*time.Hour + 30*time.Minute, "strong"},
		{4*time.Hour + 1*time.Minute, "exceeded"},
		{5 * time.Hour, "exceeded"},
	}

	for _, tt := range tests {
		t.Run(tt.duration.String(), func(t *testing.T) {
			var level string
			switch {
			case tt.duration >= CheckpointMaxDuration:
				level = "exceeded"
			case tt.duration >= CheckpointStrongDuration:
				level = "strong"
			case tt.duration >= CheckpointWarningDuration:
				level = "warning"
			default:
				level = "ok"
			}

			if level != tt.wantLevel {
				t.Errorf("level for %v = %q, want %q", tt.duration, level, tt.wantLevel)
			}
		})
	}
}

func TestCheckpointConstants(t *testing.T) {
	// Verify checkpoint constants are in expected order
	if CheckpointWarningDuration >= CheckpointStrongDuration {
		t.Errorf("CheckpointWarningDuration (%v) should be < CheckpointStrongDuration (%v)",
			CheckpointWarningDuration, CheckpointStrongDuration)
	}
	if CheckpointStrongDuration >= CheckpointMaxDuration {
		t.Errorf("CheckpointStrongDuration (%v) should be < CheckpointMaxDuration (%v)",
			CheckpointStrongDuration, CheckpointMaxDuration)
	}

	// Verify expected values
	if CheckpointWarningDuration != 2*time.Hour {
		t.Errorf("CheckpointWarningDuration = %v, want 2h", CheckpointWarningDuration)
	}
	if CheckpointStrongDuration != 3*time.Hour {
		t.Errorf("CheckpointStrongDuration = %v, want 3h", CheckpointStrongDuration)
	}
	if CheckpointMaxDuration != 4*time.Hour {
		t.Errorf("CheckpointMaxDuration = %v, want 4h", CheckpointMaxDuration)
	}
}

func TestDefaultThresholds(t *testing.T) {
	// Verify default agent thresholds
	agentThresholds := DefaultAgentThresholds()
	if agentThresholds.Warning != 2*time.Hour {
		t.Errorf("DefaultAgentThresholds().Warning = %v, want 2h", agentThresholds.Warning)
	}
	if agentThresholds.Strong != 3*time.Hour {
		t.Errorf("DefaultAgentThresholds().Strong = %v, want 3h", agentThresholds.Strong)
	}
	if agentThresholds.Max != 4*time.Hour {
		t.Errorf("DefaultAgentThresholds().Max = %v, want 4h", agentThresholds.Max)
	}

	// Verify default orchestrator thresholds (longer than agent)
	orchThresholds := DefaultOrchestratorThresholds()
	if orchThresholds.Warning != 4*time.Hour {
		t.Errorf("DefaultOrchestratorThresholds().Warning = %v, want 4h", orchThresholds.Warning)
	}
	if orchThresholds.Strong != 6*time.Hour {
		t.Errorf("DefaultOrchestratorThresholds().Strong = %v, want 6h", orchThresholds.Strong)
	}
	if orchThresholds.Max != 8*time.Hour {
		t.Errorf("DefaultOrchestratorThresholds().Max = %v, want 8h", orchThresholds.Max)
	}

	// Verify orchestrator thresholds are longer than agent thresholds
	if orchThresholds.Warning <= agentThresholds.Warning {
		t.Errorf("Orchestrator warning (%v) should be > agent warning (%v)",
			orchThresholds.Warning, agentThresholds.Warning)
	}
	if orchThresholds.Strong <= agentThresholds.Strong {
		t.Errorf("Orchestrator strong (%v) should be > agent strong (%v)",
			orchThresholds.Strong, agentThresholds.Strong)
	}
	if orchThresholds.Max <= agentThresholds.Max {
		t.Errorf("Orchestrator max (%v) should be > agent max (%v)",
			orchThresholds.Max, agentThresholds.Max)
	}
}

func TestGetCheckpointStatusWithType(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.json")

	store, err := New(sessionPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Start session
	if err := store.Start("Test type-aware checkpoints"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Test SessionTypeAgent returns agent thresholds
	agentStatus := store.GetCheckpointStatusWithType(SessionTypeAgent)
	if agentStatus == nil {
		t.Fatal("GetCheckpointStatusWithType(agent) returned nil")
	}
	if agentStatus.Level != "ok" {
		t.Errorf("Agent status level = %q, want 'ok' for new session", agentStatus.Level)
	}

	// Test SessionTypeOrchestrator returns orchestrator thresholds
	orchStatus := store.GetCheckpointStatusWithType(SessionTypeOrchestrator)
	if orchStatus == nil {
		t.Fatal("GetCheckpointStatusWithType(orchestrator) returned nil")
	}
	if orchStatus.Level != "ok" {
		t.Errorf("Orchestrator status level = %q, want 'ok' for new session", orchStatus.Level)
	}
}

func TestOrchestratorThresholdsAreLonger(t *testing.T) {
	// Test that at 3h, agent would be at "strong" but orchestrator is still "ok"
	agentThresholds := DefaultAgentThresholds()
	orchThresholds := DefaultOrchestratorThresholds()

	testDuration := 3 * time.Hour

	// At 3h, agent should hit strong threshold
	var agentLevel string
	switch {
	case testDuration >= agentThresholds.Max:
		agentLevel = "exceeded"
	case testDuration >= agentThresholds.Strong:
		agentLevel = "strong"
	case testDuration >= agentThresholds.Warning:
		agentLevel = "warning"
	default:
		agentLevel = "ok"
	}
	if agentLevel != "strong" {
		t.Errorf("Agent at 3h = %q, want 'strong'", agentLevel)
	}

	// At 3h, orchestrator should still be "ok"
	var orchLevel string
	switch {
	case testDuration >= orchThresholds.Max:
		orchLevel = "exceeded"
	case testDuration >= orchThresholds.Strong:
		orchLevel = "strong"
	case testDuration >= orchThresholds.Warning:
		orchLevel = "warning"
	default:
		orchLevel = "ok"
	}
	if orchLevel != "ok" {
		t.Errorf("Orchestrator at 3h = %q, want 'ok'", orchLevel)
	}
}

func TestGetCheckpointStatusWithThresholds(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "session.json")

	store, err := New(sessionPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Start session
	if err := store.Start("Test custom thresholds"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Test with custom thresholds
	customThresholds := CheckpointThresholds{
		Warning: 1 * time.Minute,
		Strong:  2 * time.Minute,
		Max:     3 * time.Minute,
	}

	status := store.GetCheckpointStatusWithThresholds(customThresholds)
	if status == nil {
		t.Fatal("GetCheckpointStatusWithThresholds() returned nil")
	}
	// New session should be "ok" even with short thresholds
	if status.Level != "ok" {
		t.Errorf("Status level = %q, want 'ok' for new session", status.Level)
	}
}

func TestGenerateSessionName(t *testing.T) {
	// Create temp project directory
	tmpDir := t.TempDir()
	
	// Create a subdirectory to simulate a real project name
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	tests := []struct {
		name           string
		existingSessions []string
		wantName       string
	}{
		{
			name:           "no existing sessions",
			existingSessions: nil,
			wantName:       "test-project-1",
		},
		{
			name:           "one existing session",
			existingSessions: []string{"test-project-1"},
			wantName:       "test-project-2",
		},
		{
			name:           "multiple existing sessions",
			existingSessions: []string{"test-project-1", "test-project-2", "test-project-3"},
			wantName:       "test-project-4",
		},
		{
			name:           "non-sequential numbers",
			existingSessions: []string{"test-project-1", "test-project-5", "test-project-3"},
			wantName:       "test-project-6",
		},
		{
			name:           "mixed with other projects",
			existingSessions: []string{"test-project-1", "other-project-1", "test-project-2"},
			wantName:       "test-project-3",
		},
		{
			name:           "non-matching directories ignored",
			existingSessions: []string{"test-project-1", "random-name", "2026-01-13-1000"},
			wantName:       "test-project-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up session directory
			sessionDir := filepath.Join(projectDir, ".orch", "session")
			os.RemoveAll(sessionDir)

			// Create existing session directories
			for _, name := range tt.existingSessions {
				dir := filepath.Join(sessionDir, name)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("Failed to create session dir %s: %v", name, err)
				}
			}

			// Generate session name
			got, err := GenerateSessionName(projectDir)
			if err != nil {
				t.Fatalf("GenerateSessionName() error = %v", err)
			}

			if got != tt.wantName {
				t.Errorf("GenerateSessionName() = %q, want %q", got, tt.wantName)
			}
		})
	}
}

func TestCountProjectSessions(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, ".orch", "session")

	tests := []struct {
		name     string
		existing []string
		project  string
		want     int
	}{
		{
			name:     "empty directory",
			existing: nil,
			project:  "test",
			want:     0,
		},
		{
			name:     "single session",
			existing: []string{"test-1"},
			project:  "test",
			want:     1,
		},
		{
			name:     "multiple sessions",
			existing: []string{"test-1", "test-2", "test-3"},
			project:  "test",
			want:     3,
		},
		{
			name:     "highest number returned",
			existing: []string{"test-5", "test-2", "test-10"},
			project:  "test",
			want:     10,
		},
		{
			name:     "filter by project name",
			existing: []string{"test-1", "other-5", "test-2"},
			project:  "test",
			want:     2,
		},
		{
			name:     "ignore non-matching",
			existing: []string{"test-1", "random", "2026-01-13"},
			project:  "test",
			want:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			os.RemoveAll(sessionDir)

			// Create directories
			if tt.existing != nil {
				for _, name := range tt.existing {
					dir := filepath.Join(sessionDir, name)
					if err := os.MkdirAll(dir, 0755); err != nil {
						t.Fatalf("Failed to create dir %s: %v", name, err)
					}
				}
			}

			got, err := countProjectSessions(sessionDir, tt.project)
			if err != nil {
				t.Fatalf("countProjectSessions() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("countProjectSessions() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGenerateSessionNameProjectNameExtraction(t *testing.T) {
	tests := []struct {
		name        string
		projectPath string
		wantPrefix  string
	}{
		{
			name:        "simple project name",
			projectPath: "/path/to/my-project",
			wantPrefix:  "my-project-",
		},
		{
			name:        "project with underscores",
			projectPath: "/path/to/my_cool_project",
			wantPrefix:  "my_cool_project-",
		},
		{
			name:        "nested project",
			projectPath: "/very/deep/path/to/project",
			wantPrefix:  "project-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory with the project name
			tmpDir := t.TempDir()
			projectDir := filepath.Join(tmpDir, filepath.Base(tt.projectPath))
			if err := os.MkdirAll(projectDir, 0755); err != nil {
				t.Fatalf("Failed to create project dir: %v", err)
			}

			got, err := GenerateSessionName(projectDir)
			if err != nil {
				t.Fatalf("GenerateSessionName() error = %v", err)
			}

			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("GenerateSessionName() = %q, want prefix %q", got, tt.wantPrefix)
			}

			// Should end with -1 for first session
			if !strings.HasSuffix(got, "-1") {
				t.Errorf("GenerateSessionName() = %q, want suffix '-1'", got)
			}
		})
	}
}
