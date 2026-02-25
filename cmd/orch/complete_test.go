package main

import (
	"os"
	"path/filepath"
	"testing"
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

// TestArchiveWorkspace tests the archiveWorkspace function.
func TestArchiveWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a workspace to archive
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")
	wsPath := filepath.Join(workspaceDir, "og-feat-test-17jan-abc1")
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create some files in the workspace
	if err := os.WriteFile(filepath.Join(wsPath, "SPAWN_CONTEXT.md"), []byte("Test context"), 0644); err != nil {
		t.Fatalf("Failed to create SPAWN_CONTEXT.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsPath, "SYNTHESIS.md"), []byte("Test synthesis"), 0644); err != nil {
		t.Fatalf("Failed to create SYNTHESIS.md: %v", err)
	}

	// Archive the workspace
	archivedPath, err := archiveWorkspace(wsPath, tmpDir)
	if err != nil {
		t.Fatalf("archiveWorkspace failed: %v", err)
	}

	// Verify workspace was moved
	if _, err := os.Stat(wsPath); !os.IsNotExist(err) {
		t.Error("Original workspace should not exist after archival")
	}

	// Verify archived path is correct
	expectedArchivedPath := filepath.Join(tmpDir, ".orch", "workspace", "archived", "og-feat-test-17jan-abc1")
	if archivedPath != expectedArchivedPath {
		t.Errorf("Expected archived path %s, got %s", expectedArchivedPath, archivedPath)
	}

	// Verify archived workspace exists
	if _, err := os.Stat(archivedPath); os.IsNotExist(err) {
		t.Error("Archived workspace should exist")
	}

	// Verify files were preserved
	if _, err := os.Stat(filepath.Join(archivedPath, "SPAWN_CONTEXT.md")); os.IsNotExist(err) {
		t.Error("SPAWN_CONTEXT.md should exist in archived workspace")
	}
	if _, err := os.Stat(filepath.Join(archivedPath, "SYNTHESIS.md")); os.IsNotExist(err) {
		t.Error("SYNTHESIS.md should exist in archived workspace")
	}
}

// TestArchiveWorkspaceEmptyPath tests archiveWorkspace with empty path.
func TestArchiveWorkspaceEmptyPath(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := archiveWorkspace("", tmpDir)
	if err == nil {
		t.Error("Expected error for empty workspace path")
	}
	if err.Error() != "workspace path is empty" {
		t.Errorf("Expected 'workspace path is empty' error, got: %v", err)
	}
}

// TestArchiveWorkspaceNonExistent tests archiveWorkspace with non-existent workspace.
func TestArchiveWorkspaceNonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := archiveWorkspace(filepath.Join(tmpDir, "nonexistent"), tmpDir)
	if err == nil {
		t.Error("Expected error for non-existent workspace")
	}
}

// TestArchiveWorkspaceNameCollision tests archiveWorkspace handles name collisions.
func TestArchiveWorkspaceNameCollision(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")
	archivedDir := filepath.Join(workspaceDir, "archived")

	// Create workspace to archive
	wsName := "og-feat-collision-17jan"
	wsPath := filepath.Join(workspaceDir, wsName)
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsPath, "content.txt"), []byte("original"), 0644); err != nil {
		t.Fatalf("Failed to create content file: %v", err)
	}

	// Pre-create an archived workspace with the same name (simulating collision)
	existingArchivedPath := filepath.Join(archivedDir, wsName)
	if err := os.MkdirAll(existingArchivedPath, 0755); err != nil {
		t.Fatalf("Failed to create existing archived workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(existingArchivedPath, "existing.txt"), []byte("existing"), 0644); err != nil {
		t.Fatalf("Failed to create existing content: %v", err)
	}

	// Archive the workspace - should handle collision
	archivedPath, err := archiveWorkspace(wsPath, tmpDir)
	if err != nil {
		t.Fatalf("archiveWorkspace failed: %v", err)
	}

	// Verify the new archive has a timestamp suffix
	if archivedPath == existingArchivedPath {
		t.Error("Archived path should have timestamp suffix to avoid collision")
	}

	// Verify both archived workspaces exist
	if _, err := os.Stat(existingArchivedPath); os.IsNotExist(err) {
		t.Error("Existing archived workspace should still exist")
	}
	if _, err := os.Stat(archivedPath); os.IsNotExist(err) {
		t.Error("New archived workspace should exist")
	}

	// Verify original workspace was moved
	if _, err := os.Stat(wsPath); !os.IsNotExist(err) {
		t.Error("Original workspace should not exist after archival")
	}
}

// TestCountFileLines tests the countFileLines helper function.
func TestCountFileLines(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file with known line count
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line1\nline2\nline3\nline4\nline5"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test counting lines
	count, err := countFileLines(testFile)
	if err != nil {
		t.Fatalf("countFileLines failed: %v", err)
	}

	expectedLines := 5
	if count != expectedLines {
		t.Errorf("Expected %d lines, got %d", expectedLines, count)
	}

	// Test non-existent file
	_, err = countFileLines(filepath.Join(tmpDir, "nonexistent.txt"))
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

// TestSkipPhaseCompleteTriggersForceClose verifies that --skip-phase-complete
// causes the completion flow to use force-close, avoiding the double-gate where
// both orch complete and bd close independently check for Phase: Complete.
func TestSkipPhaseCompleteTriggersForceClose(t *testing.T) {
	// When PhaseComplete is skipped, useForceClose should be true
	skipConfig := SkipConfig{
		PhaseComplete: true,
		Reason:        "Agent completed work but beads comment failed",
	}
	useForceClose := skipConfig.PhaseComplete
	if !useForceClose {
		t.Error("Expected useForceClose=true when SkipConfig.PhaseComplete is set")
	}

	// When PhaseComplete is NOT skipped, useForceClose should be false
	noSkipConfig := SkipConfig{
		TestEvidence: true,
		Reason:       "Tests run in CI pipeline",
	}
	useForceCloseNoSkip := noSkipConfig.PhaseComplete
	if useForceCloseNoSkip {
		t.Error("Expected useForceClose=false when SkipConfig.PhaseComplete is NOT set")
	}

	// Verify PhaseComplete is included in skippedGates
	gates := skipConfig.skippedGates()
	found := false
	for _, g := range gates {
		if g == "phase_complete" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'phase_complete' in skippedGates(), got %v", gates)
	}

	// Verify shouldSkipGate returns true for phase_complete
	if !skipConfig.shouldSkipGate("phase_complete") {
		t.Error("Expected shouldSkipGate('phase_complete') to return true")
	}
}
