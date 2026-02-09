package spawn

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func TestWriteReadSessionID(t *testing.T) {
	// Create temp directory as workspace
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sessionID := "session-abc123"

	// Write session ID
	if err := WriteSessionID(tmpDir, sessionID); err != nil {
		t.Fatalf("WriteSessionID failed: %v", err)
	}

	// Verify file exists
	sessionFile := filepath.Join(tmpDir, SessionIDFilename)
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		t.Fatalf("session ID file not created")
	}

	// Read session ID
	readID := ReadSessionID(tmpDir)
	if readID != sessionID {
		t.Errorf("ReadSessionID returned %q, want %q", readID, sessionID)
	}
}

func TestWriteSessionID_EmptyID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Writing empty ID should succeed but not create file
	if err := WriteSessionID(tmpDir, ""); err != nil {
		t.Fatalf("WriteSessionID with empty ID failed: %v", err)
	}

	// File should not exist
	sessionFile := filepath.Join(tmpDir, SessionIDFilename)
	if _, err := os.Stat(sessionFile); !os.IsNotExist(err) {
		t.Errorf("session ID file should not exist for empty ID")
	}
}

func TestReadSessionID_NoFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Reading from non-existent file should return empty string
	readID := ReadSessionID(tmpDir)
	if readID != "" {
		t.Errorf("ReadSessionID returned %q for non-existent file, want empty string", readID)
	}
}

func TestSessionIDPath(t *testing.T) {
	workspace := "/some/workspace/path"
	expected := filepath.Join(workspace, SessionIDFilename)
	got := SessionIDPath(workspace)
	if got != expected {
		t.Errorf("SessionIDPath returned %q, want %q", got, expected)
	}
}

func TestReadBeadsID(t *testing.T) {
	// Create temp directory as workspace
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	beadsID := "orch-go-20985"

	// Write beads ID file
	beadsFile := filepath.Join(tmpDir, BeadsIDFilename)
	if err := os.WriteFile(beadsFile, []byte(beadsID+"\n"), 0644); err != nil {
		t.Fatalf("failed to write beads ID file: %v", err)
	}

	// Read beads ID
	readID := ReadBeadsID(tmpDir)
	if readID != beadsID {
		t.Errorf("ReadBeadsID returned %q, want %q", readID, beadsID)
	}
}

func TestReadBeadsID_NoFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Reading from non-existent file should return empty string
	readID := ReadBeadsID(tmpDir)
	if readID != "" {
		t.Errorf("ReadBeadsID returned %q for non-existent file, want empty string", readID)
	}
}

func TestBeadsIDPath(t *testing.T) {
	workspace := "/some/workspace/path"
	expected := filepath.Join(workspace, BeadsIDFilename)
	got := BeadsIDPath(workspace)
	if got != expected {
		t.Errorf("BeadsIDPath returned %q, want %q", got, expected)
	}
}

func TestWriteReadTier(t *testing.T) {
	// Create temp directory as workspace
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tier := TierLight

	// Write tier
	if err := WriteTier(tmpDir, tier); err != nil {
		t.Fatalf("WriteTier failed: %v", err)
	}

	// Verify file exists
	tierFile := filepath.Join(tmpDir, TierFilename)
	if _, err := os.Stat(tierFile); os.IsNotExist(err) {
		t.Fatalf("tier file not created")
	}

	// Read tier
	readTier := ReadTier(tmpDir)
	if readTier != tier {
		t.Errorf("ReadTier returned %q, want %q", readTier, tier)
	}
}

func TestWriteTier_EmptyTier(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Writing empty tier should succeed but not create file
	if err := WriteTier(tmpDir, ""); err != nil {
		t.Fatalf("WriteTier with empty tier failed: %v", err)
	}

	// File should not exist
	tierFile := filepath.Join(tmpDir, TierFilename)
	if _, err := os.Stat(tierFile); !os.IsNotExist(err) {
		t.Errorf("tier file should not exist for empty tier")
	}
}

func TestReadTier_NoFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Reading from non-existent file should return TierFull (conservative default)
	readTier := ReadTier(tmpDir)
	if readTier != TierFull {
		t.Errorf("ReadTier returned %q for non-existent file, want %q (conservative default)", readTier, TierFull)
	}
}

func TestTierPath(t *testing.T) {
	workspace := "/some/workspace/path"
	expected := filepath.Join(workspace, TierFilename)
	got := TierPath(workspace)
	if got != expected {
		t.Errorf("TierPath returned %q, want %q", got, expected)
	}
}

func TestWriteReadSpawnTime(t *testing.T) {
	// Create temp directory as workspace
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use a specific time to test precision
	spawnTime := time.Date(2025, 12, 23, 14, 30, 45, 123456789, time.UTC)

	// Write spawn time
	if err := WriteSpawnTime(tmpDir, spawnTime); err != nil {
		t.Fatalf("WriteSpawnTime failed: %v", err)
	}

	// Verify file exists
	spawnTimeFile := filepath.Join(tmpDir, SpawnTimeFilename)
	if _, err := os.Stat(spawnTimeFile); os.IsNotExist(err) {
		t.Fatalf("spawn time file not created")
	}

	// Read spawn time
	readTime := ReadSpawnTime(tmpDir)
	if !readTime.Equal(spawnTime) {
		t.Errorf("ReadSpawnTime returned %v, want %v", readTime, spawnTime)
	}
}

func TestReadSpawnTime_NoFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Reading from non-existent file should return zero time
	readTime := ReadSpawnTime(tmpDir)
	if !readTime.IsZero() {
		t.Errorf("ReadSpawnTime returned %v for non-existent file, want zero time", readTime)
	}
}

func TestReadSpawnTime_InvalidContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write invalid content
	spawnTimeFile := filepath.Join(tmpDir, SpawnTimeFilename)
	if err := os.WriteFile(spawnTimeFile, []byte("not-a-number\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Reading invalid content should return zero time
	readTime := ReadSpawnTime(tmpDir)
	if !readTime.IsZero() {
		t.Errorf("ReadSpawnTime returned %v for invalid content, want zero time", readTime)
	}
}

func TestSpawnTimePath(t *testing.T) {
	workspace := "/some/workspace/path"
	expected := filepath.Join(workspace, SpawnTimeFilename)
	got := SpawnTimePath(workspace)
	if got != expected {
		t.Errorf("SpawnTimePath returned %q, want %q", got, expected)
	}
}

func TestGenerateAttemptID(t *testing.T) {
	attemptID, err := GenerateAttemptID()
	if err != nil {
		t.Fatalf("GenerateAttemptID failed: %v", err)
	}

	pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !pattern.MatchString(attemptID) {
		t.Fatalf("GenerateAttemptID returned %q, expected UUIDv4 format", attemptID)
	}
}

func TestWriteReadAttemptID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	attemptID := "a1b2c3d4-1111-4aaa-8bbb-1234567890ab"
	if err := WriteAttemptID(tmpDir, attemptID); err != nil {
		t.Fatalf("WriteAttemptID failed: %v", err)
	}

	readID := ReadAttemptID(tmpDir)
	if readID != attemptID {
		t.Errorf("ReadAttemptID returned %q, want %q", readID, attemptID)
	}
}

func TestReadAttemptID_NoFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	readID := ReadAttemptID(tmpDir)
	if readID != "" {
		t.Errorf("ReadAttemptID returned %q for non-existent file, want empty string", readID)
	}
}

func TestAttemptIDPath(t *testing.T) {
	workspace := "/some/workspace/path"
	expected := filepath.Join(workspace, AttemptIDFilename)
	got := AttemptIDPath(workspace)
	if got != expected {
		t.Errorf("AttemptIDPath returned %q, want %q", got, expected)
	}
}

func TestWriteReadAgentManifest(t *testing.T) {
	// Create temp directory as workspace
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manifest := AgentManifest{
		WorkspaceName: "og-feat-test-17jan-abc1",
		Skill:         "feature-impl",
		BeadsID:       "orch-go-xyz1",
		ProjectDir:    "/Users/test/orch-go",
		GitBaseline:   "abc123def456",
		SpawnTime:     "2026-01-17T10:30:00Z",
		Tier:          TierFull,
		SpawnMode:     "opencode",
	}

	// Write manifest
	if err := WriteAgentManifest(tmpDir, manifest); err != nil {
		t.Fatalf("WriteAgentManifest failed: %v", err)
	}

	// Verify file exists
	manifestFile := filepath.Join(tmpDir, AgentManifestFilename)
	if _, err := os.Stat(manifestFile); os.IsNotExist(err) {
		t.Fatalf("manifest file not created")
	}

	// Read manifest
	readManifest, err := ReadAgentManifest(tmpDir)
	if err != nil {
		t.Fatalf("ReadAgentManifest failed: %v", err)
	}

	// Verify all fields
	if readManifest.WorkspaceName != manifest.WorkspaceName {
		t.Errorf("WorkspaceName: got %q, want %q", readManifest.WorkspaceName, manifest.WorkspaceName)
	}
	if readManifest.Skill != manifest.Skill {
		t.Errorf("Skill: got %q, want %q", readManifest.Skill, manifest.Skill)
	}
	if readManifest.BeadsID != manifest.BeadsID {
		t.Errorf("BeadsID: got %q, want %q", readManifest.BeadsID, manifest.BeadsID)
	}
	if readManifest.ProjectDir != manifest.ProjectDir {
		t.Errorf("ProjectDir: got %q, want %q", readManifest.ProjectDir, manifest.ProjectDir)
	}
	if readManifest.GitBaseline != manifest.GitBaseline {
		t.Errorf("GitBaseline: got %q, want %q", readManifest.GitBaseline, manifest.GitBaseline)
	}
	if readManifest.SpawnTime != manifest.SpawnTime {
		t.Errorf("SpawnTime: got %q, want %q", readManifest.SpawnTime, manifest.SpawnTime)
	}
	if readManifest.Tier != manifest.Tier {
		t.Errorf("Tier: got %q, want %q", readManifest.Tier, manifest.Tier)
	}
	if readManifest.SpawnMode != manifest.SpawnMode {
		t.Errorf("SpawnMode: got %q, want %q", readManifest.SpawnMode, manifest.SpawnMode)
	}
}

func TestWriteAgentManifest_NoBeadsID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Manifest without BeadsID (--no-track spawn)
	manifest := AgentManifest{
		WorkspaceName: "og-feat-test-17jan-abc1",
		Skill:         "investigation",
		BeadsID:       "", // Empty for --no-track
		ProjectDir:    "/Users/test/orch-go",
		GitBaseline:   "abc123",
		SpawnTime:     "2026-01-17T10:30:00Z",
		Tier:          TierLight,
	}

	if err := WriteAgentManifest(tmpDir, manifest); err != nil {
		t.Fatalf("WriteAgentManifest failed: %v", err)
	}

	readManifest, err := ReadAgentManifest(tmpDir)
	if err != nil {
		t.Fatalf("ReadAgentManifest failed: %v", err)
	}

	// BeadsID should be empty
	if readManifest.BeadsID != "" {
		t.Errorf("BeadsID: got %q, want empty string", readManifest.BeadsID)
	}
}

func TestWriteAgentManifest_NoGitBaseline(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Manifest without git baseline (not in git repo)
	manifest := AgentManifest{
		WorkspaceName: "og-feat-test-17jan-abc1",
		Skill:         "feature-impl",
		BeadsID:       "orch-go-xyz1",
		ProjectDir:    "/Users/test/orch-go",
		GitBaseline:   "", // Empty when not in git repo
		SpawnTime:     "2026-01-17T10:30:00Z",
		Tier:          TierFull,
	}

	if err := WriteAgentManifest(tmpDir, manifest); err != nil {
		t.Fatalf("WriteAgentManifest failed: %v", err)
	}

	readManifest, err := ReadAgentManifest(tmpDir)
	if err != nil {
		t.Fatalf("ReadAgentManifest failed: %v", err)
	}

	// GitBaseline should be empty
	if readManifest.GitBaseline != "" {
		t.Errorf("GitBaseline: got %q, want empty string", readManifest.GitBaseline)
	}
}

func TestReadAgentManifest_NoFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Reading from non-existent file should return error
	_, err = ReadAgentManifest(tmpDir)
	if err == nil {
		t.Error("ReadAgentManifest should return error for non-existent file")
	}
}

func TestReadAgentManifest_InvalidJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spawn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write invalid JSON
	manifestFile := filepath.Join(tmpDir, AgentManifestFilename)
	if err := os.WriteFile(manifestFile, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Reading invalid JSON should return error
	_, err = ReadAgentManifest(tmpDir)
	if err == nil {
		t.Error("ReadAgentManifest should return error for invalid JSON")
	}
}

func TestAgentManifestPath(t *testing.T) {
	workspace := "/some/workspace/path"
	expected := filepath.Join(workspace, AgentManifestFilename)
	got := AgentManifestPath(workspace)
	if got != expected {
		t.Errorf("AgentManifestPath returned %q, want %q", got, expected)
	}
}
