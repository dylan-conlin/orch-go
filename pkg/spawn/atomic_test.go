package spawn

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAtomicSpawnPhase1_WritesWorkspace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "atomic-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .orch/workspace dir structure
	orchDir := filepath.Join(tmpDir, ".orch", "workspace")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("failed to create orch dir: %v", err)
	}

	cfg := &Config{
		Task:          "test task",
		SkillName:     "feature-impl",
		Project:       "test-proj",
		ProjectDir:    tmpDir,
		WorkspaceName: "test-workspace-abc1",
		BeadsID:       "test-proj-xyz1",
		Tier:          TierLight,
		SpawnMode:     "opencode",
		Model:         "anthropic/claude-sonnet-4-20250514",
	}

	opts := &AtomicSpawnOpts{
		Config:  cfg,
		BeadsID: "test-proj-xyz1",
		NoTrack: true, // Skip beads tagging in test (no beads daemon)
	}

	rollback, err := AtomicSpawnPhase1(opts)
	if err != nil {
		t.Fatalf("AtomicSpawnPhase1 failed: %v", err)
	}

	// Verify workspace was created
	workspacePath := cfg.WorkspacePath()
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		t.Fatal("workspace directory was not created")
	}

	// Verify SPAWN_CONTEXT.md exists
	contextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		t.Fatal("SPAWN_CONTEXT.md was not created")
	}

	// Verify AGENT_MANIFEST.json exists
	manifestPath := filepath.Join(workspacePath, AgentManifestFilename)
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Fatal("AGENT_MANIFEST.json was not created")
	}

	// Read manifest and verify no SessionID (Phase 1 doesn't set it)
	manifest, err := ReadAgentManifest(workspacePath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}
	if manifest.SessionID != "" {
		t.Errorf("SessionID should be empty after Phase 1, got %q", manifest.SessionID)
	}

	// Verify rollback works
	rollback()
	if _, err := os.Stat(workspacePath); !os.IsNotExist(err) {
		t.Error("workspace should be removed after rollback")
	}
}

func TestAtomicSpawnPhase2_UpdatesManifest(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "atomic-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create workspace with a manifest (simulating Phase 1 output)
	workspacePath := filepath.Join(tmpDir, ".orch", "workspace", "test-workspace-abc1")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	manifest := AgentManifest{
		WorkspaceName: "test-workspace-abc1",
		Skill:         "feature-impl",
		BeadsID:       "test-proj-xyz1",
		ProjectDir:    tmpDir,
		SpawnTime:     "2026-02-19T10:00:00Z",
		Tier:          TierLight,
		SpawnMode:     "opencode",
		Model:         "anthropic/claude-sonnet-4-20250514",
	}
	if err := WriteAgentManifest(workspacePath, manifest); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	cfg := &Config{
		ProjectDir:    tmpDir,
		WorkspaceName: "test-workspace-abc1",
	}
	opts := &AtomicSpawnOpts{
		Config:  cfg,
		BeadsID: "test-proj-xyz1",
	}

	sessionID := "session-12345"
	if err := AtomicSpawnPhase2(opts, sessionID); err != nil {
		t.Fatalf("AtomicSpawnPhase2 failed: %v", err)
	}

	// Verify session ID dotfile was written
	readID := ReadSessionID(workspacePath)
	if readID != sessionID {
		t.Errorf("session ID dotfile: got %q, want %q", readID, sessionID)
	}

	// Verify manifest was updated with SessionID
	updatedManifest, err := ReadAgentManifest(workspacePath)
	if err != nil {
		t.Fatalf("failed to read updated manifest: %v", err)
	}
	if updatedManifest.SessionID != sessionID {
		t.Errorf("manifest SessionID: got %q, want %q", updatedManifest.SessionID, sessionID)
	}

	// Verify other fields were preserved
	if updatedManifest.BeadsID != "test-proj-xyz1" {
		t.Errorf("manifest BeadsID: got %q, want %q", updatedManifest.BeadsID, "test-proj-xyz1")
	}
	if updatedManifest.Skill != "feature-impl" {
		t.Errorf("manifest Skill: got %q, want %q", updatedManifest.Skill, "feature-impl")
	}
}

func TestAtomicSpawnPhase2_EmptySessionID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "atomic-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create workspace with manifest
	workspacePath := filepath.Join(tmpDir, ".orch", "workspace", "test-workspace-abc1")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	manifest := AgentManifest{
		WorkspaceName: "test-workspace-abc1",
		Skill:         "feature-impl",
		BeadsID:       "test-proj-xyz1",
		ProjectDir:    tmpDir,
		SpawnTime:     "2026-02-19T10:00:00Z",
		Tier:          TierLight,
	}
	if err := WriteAgentManifest(workspacePath, manifest); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	cfg := &Config{
		ProjectDir:    tmpDir,
		WorkspaceName: "test-workspace-abc1",
	}
	opts := &AtomicSpawnOpts{
		Config:  cfg,
		BeadsID: "test-proj-xyz1",
	}

	// Phase 2 with empty session ID (claude backend)
	if err := AtomicSpawnPhase2(opts, ""); err != nil {
		t.Fatalf("AtomicSpawnPhase2 with empty session ID failed: %v", err)
	}

	// Session ID dotfile should not exist
	sessionFile := filepath.Join(workspacePath, SessionIDFilename)
	if _, err := os.Stat(sessionFile); !os.IsNotExist(err) {
		t.Error("session ID file should not exist for empty session ID")
	}

	// Manifest should still not have SessionID
	updatedManifest, err := ReadAgentManifest(workspacePath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}
	if updatedManifest.SessionID != "" {
		t.Errorf("manifest SessionID should be empty, got %q", updatedManifest.SessionID)
	}
}

func TestAgentManifest_SessionID_Serialization(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "manifest-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manifest := AgentManifest{
		WorkspaceName: "test-workspace",
		Skill:         "feature-impl",
		BeadsID:       "proj-123",
		ProjectDir:    "/test/dir",
		SpawnTime:     "2026-02-19T10:00:00Z",
		Tier:          TierLight,
		SessionID:     "session-abc123",
	}

	if err := WriteAgentManifest(tmpDir, manifest); err != nil {
		t.Fatalf("WriteAgentManifest failed: %v", err)
	}

	// Read back and verify
	readManifest, err := ReadAgentManifest(tmpDir)
	if err != nil {
		t.Fatalf("ReadAgentManifest failed: %v", err)
	}
	if readManifest.SessionID != "session-abc123" {
		t.Errorf("SessionID: got %q, want %q", readManifest.SessionID, "session-abc123")
	}

	// Verify JSON has session_id field
	data, _ := os.ReadFile(filepath.Join(tmpDir, AgentManifestFilename))
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)
	if _, ok := raw["session_id"]; !ok {
		t.Error("JSON should contain session_id field")
	}
}

func TestAgentManifest_SessionID_OmitEmpty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "manifest-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manifest := AgentManifest{
		WorkspaceName: "test-workspace",
		Skill:         "feature-impl",
		ProjectDir:    "/test/dir",
		SpawnTime:     "2026-02-19T10:00:00Z",
		Tier:          TierLight,
		SessionID:     "", // Empty - should be omitted
	}

	if err := WriteAgentManifest(tmpDir, manifest); err != nil {
		t.Fatalf("WriteAgentManifest failed: %v", err)
	}

	// Verify JSON does NOT have session_id field (omitempty)
	data, _ := os.ReadFile(filepath.Join(tmpDir, AgentManifestFilename))
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)
	if _, ok := raw["session_id"]; ok {
		t.Error("JSON should NOT contain session_id field when empty (omitempty)")
	}
}

func TestLookupManifestsByBeadsIDs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lookup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create workspace directory structure
	workspaceRoot := filepath.Join(tmpDir, ".orch", "workspace")

	// Workspace 1: matches
	ws1 := filepath.Join(workspaceRoot, "og-feat-task1-19feb-a1b2")
	os.MkdirAll(ws1, 0755)
	WriteAgentManifest(ws1, AgentManifest{
		WorkspaceName: "og-feat-task1-19feb-a1b2",
		BeadsID:       "proj-111",
		SessionID:     "session-aaa",
		Skill:         "feature-impl",
		ProjectDir:    tmpDir,
		SpawnTime:     "2026-02-19T10:00:00Z",
		Tier:          TierLight,
	})

	// Workspace 2: matches
	ws2 := filepath.Join(workspaceRoot, "og-feat-task2-19feb-c3d4")
	os.MkdirAll(ws2, 0755)
	WriteAgentManifest(ws2, AgentManifest{
		WorkspaceName: "og-feat-task2-19feb-c3d4",
		BeadsID:       "proj-222",
		SessionID:     "session-bbb",
		Skill:         "investigation",
		ProjectDir:    tmpDir,
		SpawnTime:     "2026-02-19T11:00:00Z",
		Tier:          TierFull,
	})

	// Workspace 3: does NOT match (different beads ID)
	ws3 := filepath.Join(workspaceRoot, "og-feat-task3-19feb-e5f6")
	os.MkdirAll(ws3, 0755)
	WriteAgentManifest(ws3, AgentManifest{
		WorkspaceName: "og-feat-task3-19feb-e5f6",
		BeadsID:       "proj-999",
		Skill:         "feature-impl",
		ProjectDir:    tmpDir,
		SpawnTime:     "2026-02-19T12:00:00Z",
		Tier:          TierLight,
	})

	// Lookup only proj-111 and proj-222
	result, err := LookupManifestsByBeadsIDs(tmpDir, []string{"proj-111", "proj-222"})
	if err != nil {
		t.Fatalf("LookupManifestsByBeadsIDs failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(result))
	}

	// Verify proj-111
	m1, ok := result["proj-111"]
	if !ok {
		t.Fatal("proj-111 not found in results")
	}
	if m1.SessionID != "session-aaa" {
		t.Errorf("proj-111 SessionID: got %q, want %q", m1.SessionID, "session-aaa")
	}

	// Verify proj-222
	m2, ok := result["proj-222"]
	if !ok {
		t.Fatal("proj-222 not found in results")
	}
	if m2.SessionID != "session-bbb" {
		t.Errorf("proj-222 SessionID: got %q, want %q", m2.SessionID, "session-bbb")
	}

	// proj-999 should NOT be in results
	if _, ok := result["proj-999"]; ok {
		t.Error("proj-999 should not be in results")
	}
}

func TestLookupManifestsByBeadsIDs_EmptyInput(t *testing.T) {
	result, err := LookupManifestsByBeadsIDs("/tmp", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}
}

func TestLookupManifestsByBeadsIDs_NoWorkspaceDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lookup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// No .orch/workspace directory
	result, err := LookupManifestsByBeadsIDs(tmpDir, []string{"proj-111"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}
}

func TestAtomicSpawnPhase1_RollbackOnWorkspaceWriteFail(t *testing.T) {
	// Create a read-only directory to cause workspace write to fail
	tmpDir, err := os.MkdirTemp("", "atomic-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .orch/workspace as a regular file to make MkdirAll fail
	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("failed to create orch dir: %v", err)
	}
	// Create workspace as a FILE (not directory) to make WriteContext fail
	workspaceFile := filepath.Join(orchDir, "workspace")
	if err := os.WriteFile(workspaceFile, []byte("blocker"), 0644); err != nil {
		t.Fatalf("failed to create blocker file: %v", err)
	}

	cfg := &Config{
		Task:          "test task",
		SkillName:     "feature-impl",
		Project:       "test-proj",
		ProjectDir:    tmpDir,
		WorkspaceName: "test-workspace-abc1",
		BeadsID:       "test-proj-xyz1",
		Tier:          TierLight,
	}

	opts := &AtomicSpawnOpts{
		Config:  cfg,
		BeadsID: "test-proj-xyz1",
		NoTrack: true, // Skip beads in test
	}

	_, err = AtomicSpawnPhase1(opts)
	if err == nil {
		t.Fatal("AtomicSpawnPhase1 should fail when workspace write fails")
	}

	// Verify the error mentions workspace write
	if !strings.Contains(err.Error(), "workspace write failed") {
		t.Errorf("error should mention workspace write failure, got: %v", err)
	}
}
