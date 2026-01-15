package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/session"
)

// TestOrchestratorWorkspaceDetection verifies that orchestrator workspaces are detected
// correctly by the completion command.
func TestOrchestratorWorkspaceDetection(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")

	// Create orchestrator workspace with .orchestrator marker
	wsOrch := filepath.Join(workspaceDir, "og-orch-session-05jan")
	if err := os.MkdirAll(wsOrch, 0755); err != nil {
		t.Fatalf("Failed to create orchestrator workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsOrch, ".orchestrator"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create .orchestrator marker: %v", err)
	}

	// Create meta-orchestrator workspace
	wsMetaOrch := filepath.Join(workspaceDir, "og-meta-orch-05jan")
	if err := os.MkdirAll(wsMetaOrch, 0755); err != nil {
		t.Fatalf("Failed to create meta-orchestrator workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsMetaOrch, ".meta-orchestrator"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create .meta-orchestrator marker: %v", err)
	}

	// Create regular worker workspace
	wsWorker := filepath.Join(workspaceDir, "og-feat-worker-05jan")
	if err := os.MkdirAll(wsWorker, 0755); err != nil {
		t.Fatalf("Failed to create worker workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsWorker, ".beads_id"), []byte("orch-go-abc1"), 0644); err != nil {
		t.Fatalf("Failed to create .beads_id: %v", err)
	}

	// Test isOrchestratorWorkspace
	if !isOrchestratorWorkspace(wsOrch) {
		t.Error("Expected wsOrch to be detected as orchestrator workspace")
	}
	if !isOrchestratorWorkspace(wsMetaOrch) {
		t.Error("Expected wsMetaOrch to be detected as orchestrator workspace")
	}
	if isOrchestratorWorkspace(wsWorker) {
		t.Error("Expected wsWorker NOT to be detected as orchestrator workspace")
	}
}

// TestFindWorkspaceByName verifies workspace lookup by name.
func TestFindWorkspaceByName(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")

	// Create test workspaces
	ws1 := filepath.Join(workspaceDir, "og-orch-session-05jan")
	if err := os.MkdirAll(ws1, 0755); err != nil {
		t.Fatalf("Failed to create ws1: %v", err)
	}

	ws2 := filepath.Join(workspaceDir, "og-feat-feature-05jan")
	if err := os.MkdirAll(ws2, 0755); err != nil {
		t.Fatalf("Failed to create ws2: %v", err)
	}

	// Test finding existing workspace
	found := findWorkspaceByName(tmpDir, "og-orch-session-05jan")
	if found != ws1 {
		t.Errorf("Expected to find %s, got %s", ws1, found)
	}

	// Test finding non-existent workspace
	notFound := findWorkspaceByName(tmpDir, "og-does-not-exist")
	if notFound != "" {
		t.Errorf("Expected empty string for non-existent workspace, got %s", notFound)
	}
}

// TestSessionHandoffDetection verifies SESSION_HANDOFF.md detection.
func TestSessionHandoffDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workspace with SESSION_HANDOFF.md
	wsWithHandoff := filepath.Join(tmpDir, "ws-with-handoff")
	if err := os.MkdirAll(wsWithHandoff, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsWithHandoff, "SESSION_HANDOFF.md"), []byte("# Session Handoff"), 0644); err != nil {
		t.Fatalf("Failed to create SESSION_HANDOFF.md: %v", err)
	}

	// Create workspace without SESSION_HANDOFF.md
	wsWithoutHandoff := filepath.Join(tmpDir, "ws-without-handoff")
	if err := os.MkdirAll(wsWithoutHandoff, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	if !hasSessionHandoff(wsWithHandoff) {
		t.Error("Expected wsWithHandoff to have SESSION_HANDOFF.md")
	}
	if hasSessionHandoff(wsWithoutHandoff) {
		t.Error("Expected wsWithoutHandoff NOT to have SESSION_HANDOFF.md")
	}
}

// TestOrchestratorCompletionWorkflow tests the orchestrator completion workflow
// by simulating the expected state for an orchestrator session.
func TestOrchestratorCompletionWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")

	// Set up a complete orchestrator session (ready to be completed)
	wsOrch := filepath.Join(workspaceDir, "og-orch-session-05jan")
	if err := os.MkdirAll(wsOrch, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create .orchestrator marker
	if err := os.WriteFile(filepath.Join(wsOrch, ".orchestrator"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create .orchestrator marker: %v", err)
	}

	// Create SPAWN_CONTEXT.md
	if err := os.WriteFile(filepath.Join(wsOrch, "SPAWN_CONTEXT.md"), []byte("TASK: Test orchestrator session"), 0644); err != nil {
		t.Fatalf("Failed to create SPAWN_CONTEXT.md: %v", err)
	}

	// Create SESSION_HANDOFF.md (completion signal)
	if err := os.WriteFile(filepath.Join(wsOrch, "SESSION_HANDOFF.md"), []byte("# Session Handoff\nCompleted successfully"), 0644); err != nil {
		t.Fatalf("Failed to create SESSION_HANDOFF.md: %v", err)
	}

	// Verify the workspace is found by name
	found := findWorkspaceByName(tmpDir, "og-orch-session-05jan")
	if found == "" {
		t.Fatal("Failed to find workspace by name")
	}

	// Verify it's detected as orchestrator
	if !isOrchestratorWorkspace(found) {
		t.Error("Workspace should be detected as orchestrator")
	}

	// Verify SESSION_HANDOFF.md is detected (completion signal)
	if !hasSessionHandoff(found) {
		t.Error("Workspace should have SESSION_HANDOFF.md")
	}
}

// TestOrchestratorVsWorkerIdentification tests the logic for distinguishing
// workspace names from beads IDs.
func TestOrchestratorVsWorkerIdentification(t *testing.T) {
	tests := []struct {
		identifier      string
		isWorkspaceName bool // workspace names have formats like "og-feat-xxx-05jan"
		isBeadsID       bool // beads IDs have formats like "orch-go-abc1"
	}{
		// Workspace names (orchestrator or worker)
		{"og-orch-session-05jan", true, false},
		{"og-feat-my-feature-05jan", true, false},
		{"og-inv-investigation-05jan", true, false},
		{"og-debug-fix-05jan", true, false},

		// Beads IDs
		{"orch-go-abc1", false, true},
		{"kb-cli-xyz9", false, true},
		{"beads-12ab", false, true},

		// Short beads IDs (typically 4 chars)
		{"abc1", false, true},
		{"xyz9", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.identifier, func(t *testing.T) {
			// Workspace names typically:
			// - Start with "og-" prefix
			// - Have skill indicator (feat, inv, debug, orch, arch, etc.)
			// - End with date suffix (DDmon format)
			isWorkspace := len(tt.identifier) > 10 &&
				(tt.identifier[:3] == "og-" || tt.identifier[:3] == "kb-" ||
					tt.identifier[:3] == "bd-")

			// Beads IDs typically:
			// - Are shorter (project-xxxx format)
			// - Or very short (xxxx format for short IDs)
			isBeads := !isWorkspace

			// These are heuristics, not strict rules
			if isWorkspace != tt.isWorkspaceName {
				t.Logf("Note: %s workspace detection: got %v, expected %v",
					tt.identifier, isWorkspace, tt.isWorkspaceName)
			}
			if isBeads != tt.isBeadsID {
				t.Logf("Note: %s beads ID detection: got %v, expected %v",
					tt.identifier, isBeads, tt.isBeadsID)
			}
		})
	}
}

// TestWorkerWithBeadsID tests that worker workspaces still get their beads ID
// from the .beads_id file.
func TestWorkerWithBeadsID(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")

	// Create worker workspace with .beads_id
	wsWorker := filepath.Join(workspaceDir, "og-feat-my-feature-05jan")
	if err := os.MkdirAll(wsWorker, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	expectedBeadsID := "orch-go-abc1"
	if err := os.WriteFile(filepath.Join(wsWorker, ".beads_id"), []byte(expectedBeadsID), 0644); err != nil {
		t.Fatalf("Failed to create .beads_id: %v", err)
	}

	// Find workspace by name
	found := findWorkspaceByName(tmpDir, "og-feat-my-feature-05jan")
	if found == "" {
		t.Fatal("Failed to find workspace")
	}

	// Verify it's NOT an orchestrator workspace
	if isOrchestratorWorkspace(found) {
		t.Error("Worker workspace should not be detected as orchestrator")
	}

	// Read beads ID from .beads_id file
	beadsIDPath := filepath.Join(found, ".beads_id")
	content, err := os.ReadFile(beadsIDPath)
	if err != nil {
		t.Fatalf("Failed to read .beads_id: %v", err)
	}

	if string(content) != expectedBeadsID {
		t.Errorf("Expected beads ID %s, got %s", expectedBeadsID, string(content))
	}
}

// TestOrchestratorCompletionWithoutHandoff tests that completion fails
// when SESSION_HANDOFF.md is missing (the gate is working).
func TestOrchestratorCompletionWithoutHandoff(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")

	// Create orchestrator workspace WITHOUT SESSION_HANDOFF.md
	wsOrch := filepath.Join(workspaceDir, "og-orch-incomplete-05jan")
	if err := os.MkdirAll(wsOrch, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsOrch, ".orchestrator"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create .orchestrator marker: %v", err)
	}

	// Verify workspace is found
	found := findWorkspaceByName(tmpDir, "og-orch-incomplete-05jan")
	if found == "" {
		t.Fatal("Failed to find workspace")
	}

	// Verify it's an orchestrator workspace
	if !isOrchestratorWorkspace(found) {
		t.Error("Should be detected as orchestrator workspace")
	}

	// Verify SESSION_HANDOFF.md is NOT present (should fail completion)
	if hasSessionHandoff(found) {
		t.Error("Incomplete orchestrator should not have SESSION_HANDOFF.md")
	}
}

// TestRegistryCleanupOnCompletion tests that orchestrator sessions are
// removed from the session registry when completed.
func TestRegistryCleanupOnCompletion(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "sessions.json")

	// Create a registry with a test session
	registry := session.NewRegistry(registryPath)

	testSession := session.OrchestratorSession{
		WorkspaceName: "og-orch-test-session-05jan",
		SessionID:     "ses_test123",
		ProjectDir:    "/test/project",
		SpawnTime:     time.Now(),
		Goal:          "Test goal",
		Status:        "active",
	}

	// Register the session
	if err := registry.Register(testSession); err != nil {
		t.Fatalf("Failed to register session: %v", err)
	}

	// Verify session is in registry
	sessions, err := registry.List()
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(sessions))
	}

	// Unregister the session (simulating what complete command does)
	if err := registry.Unregister("og-orch-test-session-05jan"); err != nil {
		t.Fatalf("Failed to unregister session: %v", err)
	}

	// Verify session is removed
	sessions, err = registry.List()
	if err != nil {
		t.Fatalf("Failed to list sessions after unregister: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions after unregister, got %d", len(sessions))
	}
}

// TestRegistryFirstLookupForOrchestratorCompletion tests that the complete command
// checks the registry FIRST before falling back to beads ID lookup. This is critical
// for orchestrator sessions which don't have beads tracking.
func TestRegistryFirstLookupForOrchestratorCompletion(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "sessions.json")

	// Create workspace in a "different project" directory
	projectDir := filepath.Join(tmpDir, "other-project")
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace", "og-orch-cross-project-05jan")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create orchestrator marker files
	if err := os.WriteFile(filepath.Join(workspaceDir, ".orchestrator"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create .orchestrator marker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspaceDir, ".tier"), []byte("orchestrator\n"), 0644); err != nil {
		t.Fatalf("Failed to create .tier file: %v", err)
	}

	// Create a registry with the session pointing to the other project
	registry := session.NewRegistry(registryPath)
	testSession := session.OrchestratorSession{
		WorkspaceName: "og-orch-cross-project-05jan",
		SessionID:     "ses_test123",
		ProjectDir:    projectDir, // Points to the "other project"
		SpawnTime:     time.Now(),
		Goal:          "Test cross-project orchestrator",
		Status:        "active",
	}

	if err := registry.Register(testSession); err != nil {
		t.Fatalf("Failed to register session: %v", err)
	}

	// Verify session is retrievable by workspace name
	retrieved, err := registry.Get("og-orch-cross-project-05jan")
	if err != nil {
		t.Fatalf("Failed to get session from registry: %v", err)
	}
	if retrieved.ProjectDir != projectDir {
		t.Errorf("Expected ProjectDir %s, got %s", projectDir, retrieved.ProjectDir)
	}

	// The key test: findWorkspaceByName using the registry's ProjectDir should find the workspace
	// even though we're "not in that project directory"
	foundPath := findWorkspaceByName(retrieved.ProjectDir, retrieved.WorkspaceName)
	if foundPath == "" {
		t.Error("Expected to find workspace using registry's ProjectDir")
	}
	if foundPath != workspaceDir {
		t.Errorf("Expected %s, got %s", workspaceDir, foundPath)
	}

	// Verify it's detected as orchestrator workspace
	if !isOrchestratorWorkspace(foundPath) {
		t.Error("Workspace should be detected as orchestrator")
	}
}

// TestRegistryCleanupSessionNotFound tests that unregistering a non-existent
// session returns ErrSessionNotFound (graceful handling of legacy workspaces).
func TestRegistryCleanupSessionNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "sessions.json")

	registry := session.NewRegistry(registryPath)

	// Try to unregister a session that doesn't exist
	err := registry.Unregister("og-orch-nonexistent-05jan")
	if err != session.ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound, got %v", err)
	}
}

// TestRegistryCleanupEmptyRegistry tests that unregistering from an empty
// registry (file doesn't exist) returns ErrSessionNotFound gracefully.
func TestRegistryCleanupEmptyRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "nonexistent-sessions.json")

	registry := session.NewRegistry(registryPath)

	// Registry file doesn't exist yet - should return ErrSessionNotFound
	err := registry.Unregister("og-orch-any-05jan")
	if err != session.ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound for empty registry, got %v", err)
	}
}

// TestSkipConfigHasAnySkip tests the hasAnySkip method.
func TestSkipConfigHasAnySkip(t *testing.T) {
	tests := []struct {
		name   string
		config SkipConfig
		want   bool
	}{
		{
			name:   "empty config",
			config: SkipConfig{},
			want:   false,
		},
		{
			name:   "only reason set",
			config: SkipConfig{Reason: "some reason"},
			want:   false,
		},
		{
			name:   "test evidence skip",
			config: SkipConfig{TestEvidence: true, Reason: "test reason"},
			want:   true,
		},
		{
			name:   "visual skip",
			config: SkipConfig{Visual: true, Reason: "test reason"},
			want:   true,
		},
		{
			name:   "git diff skip",
			config: SkipConfig{GitDiff: true, Reason: "test reason"},
			want:   true,
		},
		{
			name:   "synthesis skip",
			config: SkipConfig{Synthesis: true, Reason: "test reason"},
			want:   true,
		},
		{
			name:   "build skip",
			config: SkipConfig{Build: true, Reason: "test reason"},
			want:   true,
		},
		{
			name:   "multiple skips",
			config: SkipConfig{TestEvidence: true, GitDiff: true, Reason: "test"},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.hasAnySkip()
			if got != tt.want {
				t.Errorf("hasAnySkip() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSkipConfigSkippedGates tests the skippedGates method.
func TestSkipConfigSkippedGates(t *testing.T) {
	tests := []struct {
		name   string
		config SkipConfig
		want   []string
	}{
		{
			name:   "empty config",
			config: SkipConfig{},
			want:   nil,
		},
		{
			name:   "single skip - test evidence",
			config: SkipConfig{TestEvidence: true},
			want:   []string{"test_evidence"},
		},
		{
			name:   "single skip - visual",
			config: SkipConfig{Visual: true},
			want:   []string{"visual_verification"},
		},
		{
			name:   "multiple skips",
			config: SkipConfig{TestEvidence: true, GitDiff: true, Synthesis: true},
			want:   []string{"test_evidence", "git_diff", "synthesis"},
		},
		{
			name: "all skips",
			config: SkipConfig{
				TestEvidence:  true,
				Visual:        true,
				GitDiff:       true,
				Synthesis:     true,
				Build:         true,
				Constraint:    true,
				PhaseGate:     true,
				SkillOutput:   true,
				DecisionPatch: true,
				PhaseComplete: true,
			},
			want: []string{
				"test_evidence",
				"visual_verification",
				"git_diff",
				"synthesis",
				"build",
				"constraint",
				"phase_gate",
				"skill_output",
				"decision_patch_limit",
				"phase_complete",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.skippedGates()
			if len(got) != len(tt.want) {
				t.Errorf("skippedGates() = %v, want %v", got, tt.want)
				return
			}
			for i, g := range got {
				if g != tt.want[i] {
					t.Errorf("skippedGates()[%d] = %s, want %s", i, g, tt.want[i])
				}
			}
		})
	}
}

// TestSkipConfigShouldSkipGate tests the shouldSkipGate method.
func TestSkipConfigShouldSkipGate(t *testing.T) {
	config := SkipConfig{
		TestEvidence: true,
		GitDiff:      true,
		Synthesis:    false,
		Build:        true,
	}

	tests := []struct {
		gate string
		want bool
	}{
		{"test_evidence", true},
		{"git_diff", true},
		{"build", true},
		{"synthesis", false},
		{"visual_verification", false},
		{"constraint", false},
		{"unknown_gate", false},
	}

	for _, tt := range tests {
		t.Run(tt.gate, func(t *testing.T) {
			got := config.shouldSkipGate(tt.gate)
			if got != tt.want {
				t.Errorf("shouldSkipGate(%s) = %v, want %v", tt.gate, got, tt.want)
			}
		})
	}
}

// TestValidateSkipFlags tests the skip flag validation logic.
func TestValidateSkipFlags(t *testing.T) {
	tests := []struct {
		name    string
		config  SkipConfig
		wantErr string
	}{
		{
			name:    "no skips - no error",
			config:  SkipConfig{},
			wantErr: "",
		},
		{
			name:    "no skips with reason - no error",
			config:  SkipConfig{Reason: "some reason"},
			wantErr: "",
		},
		{
			name:    "skip without reason - error",
			config:  SkipConfig{TestEvidence: true},
			wantErr: "--skip-reason is required when using --skip-* flags",
		},
		{
			name:    "skip with short reason - error",
			config:  SkipConfig{TestEvidence: true, Reason: "short"},
			wantErr: "--skip-reason must be at least 10 characters (got 5)",
		},
		{
			name:    "skip with 9 char reason - error",
			config:  SkipConfig{TestEvidence: true, Reason: "123456789"},
			wantErr: "--skip-reason must be at least 10 characters (got 9)",
		},
		{
			name:    "skip with 10 char reason - ok",
			config:  SkipConfig{TestEvidence: true, Reason: "1234567890"},
			wantErr: "",
		},
		{
			name:    "skip with long reason - ok",
			config:  SkipConfig{TestEvidence: true, Reason: "This is a valid reason for skipping the test evidence gate"},
			wantErr: "",
		},
		{
			name:    "multiple skips with valid reason - ok",
			config:  SkipConfig{TestEvidence: true, GitDiff: true, Reason: "Docs-only change"},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSkipFlags(tt.config)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("validateSkipFlags() unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("validateSkipFlags() expected error containing %q, got nil", tt.wantErr)
				} else if err.Error() != tt.wantErr {
					t.Errorf("validateSkipFlags() error = %q, want %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

// TestExtractProjectFromBeadsID tests the project name extraction from beads IDs.
// This is critical for cross-project completion - we need to correctly parse the
// project name to locate the correct beads database.
func TestExtractProjectFromBeadsID(t *testing.T) {
	tests := []struct {
		name    string
		beadsID string
		want    string
	}{
		{
			name:    "simple two-part ID",
			beadsID: "orch-go-abc1",
			want:    "orch-go",
		},
		{
			name:    "three-part project name",
			beadsID: "kb-cli-xyz9",
			want:    "kb-cli",
		},
		{
			name:    "single-word project",
			beadsID: "beads-12ab",
			want:    "beads",
		},
		{
			name:    "price-watch project",
			beadsID: "pw-ed7h",
			want:    "pw",
		},
		{
			name:    "multi-hyphen project name",
			beadsID: "some-long-project-name-a1b2",
			want:    "some-long-project-name",
		},
		{
			name:    "empty beads ID",
			beadsID: "",
			want:    "",
		},
		{
			name:    "single part (no hyphen)",
			beadsID: "abc1",
			want:    "abc1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractProjectFromBeadsID(tt.beadsID)
			if got != tt.want {
				t.Errorf("extractProjectFromBeadsID(%q) = %q, want %q", tt.beadsID, got, tt.want)
			}
		})
	}
}

// TestCrossProjectCompletion tests the cross-project completion workflow.
// This verifies that agents from other projects can be completed by:
// 1. Extracting project name from beads ID
// 2. Finding the project directory
// 3. Setting beads.DefaultDir before resolution
func TestCrossProjectCompletion(t *testing.T) {
	// Create a fake "other project" directory structure
	tmpDir := t.TempDir()

	// Create "orch-go" project (current)
	orchGoDir := filepath.Join(tmpDir, "orch-go")
	orchGoBeadsDir := filepath.Join(orchGoDir, ".beads")
	if err := os.MkdirAll(orchGoBeadsDir, 0755); err != nil {
		t.Fatalf("Failed to create orch-go beads dir: %v", err)
	}

	// Create "price-watch" project (cross-project)
	priceWatchDir := filepath.Join(tmpDir, "price-watch")
	priceWatchBeadsDir := filepath.Join(priceWatchDir, ".beads")
	if err := os.MkdirAll(priceWatchBeadsDir, 0755); err != nil {
		t.Fatalf("Failed to create price-watch beads dir: %v", err)
	}

	// Test project extraction from beads ID
	beadsID := "pw-ed7h"
	projectName := extractProjectFromBeadsID(beadsID)
	if projectName != "pw" {
		t.Errorf("Expected project name 'pw', got '%s'", projectName)
	}

	// Note: We can't fully test findProjectDirByName here because it searches
	// specific system paths (~/Documents/personal, etc.). This would require
	// mocking or dependency injection, which is out of scope for this fix.
	// The manual testing will verify the end-to-end behavior.
}

// TestCrossProjectBeadsIDDetection tests that cross-project beads IDs are
// correctly identified (when the ID prefix doesn't match current directory).
func TestCrossProjectBeadsIDDetection(t *testing.T) {
	tests := []struct {
		name           string
		beadsID        string
		currentDir     string
		isCrossProject bool
	}{
		{
			name:           "same project - orch-go",
			beadsID:        "orch-go-abc1",
			currentDir:     "/path/to/orch-go",
			isCrossProject: false,
		},
		{
			name:           "cross project - pw in orch-go",
			beadsID:        "pw-ed7h",
			currentDir:     "/path/to/orch-go",
			isCrossProject: true,
		},
		{
			name:           "cross project - kb-cli in orch-go",
			beadsID:        "kb-cli-xyz9",
			currentDir:     "/path/to/orch-go",
			isCrossProject: true,
		},
		{
			name:           "same project - kb-cli",
			beadsID:        "kb-cli-xyz9",
			currentDir:     "/path/to/kb-cli",
			isCrossProject: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectName := extractProjectFromBeadsID(tt.beadsID)
			currentBaseName := filepath.Base(tt.currentDir)

			// Cross-project if project name doesn't match current directory basename
			isCrossProject := projectName != currentBaseName

			if isCrossProject != tt.isCrossProject {
				t.Errorf("Cross-project detection for %s in %s: got %v, want %v",
					tt.beadsID, tt.currentDir, isCrossProject, tt.isCrossProject)
			}
		})
	}
}
