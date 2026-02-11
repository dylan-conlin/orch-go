package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// ──────────────────────────────────────────────────────────────
// Phase 1: resolveTarget tests
// ──────────────────────────────────────────────────────────────

// TestResolveTargetOrchestratorWorkspace tests that resolveTarget correctly
// identifies orchestrator workspaces by name.
func TestResolveTargetOrchestratorWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")

	// Create an orchestrator workspace
	wsOrch := filepath.Join(workspaceDir, "og-orch-test-06feb")
	if err := os.MkdirAll(wsOrch, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsOrch, ".orchestrator"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create .orchestrator marker: %v", err)
	}

	// Change to tmpDir so findWorkspaceByName works
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	target, err := resolveTarget("og-orch-test-06feb", "")
	if err != nil {
		t.Fatalf("resolveTarget failed: %v", err)
	}

	if !target.IsOrchestratorSession {
		t.Error("Expected IsOrchestratorSession to be true")
	}
	if target.AgentName != "og-orch-test-06feb" {
		t.Errorf("Expected AgentName 'og-orch-test-06feb', got '%s'", target.AgentName)
	}
	// macOS /var is symlinked to /private/var, so compare base names
	if filepath.Base(target.WorkspacePath) != filepath.Base(wsOrch) {
		t.Errorf("Expected WorkspacePath base '%s', got '%s'", filepath.Base(wsOrch), filepath.Base(target.WorkspacePath))
	}
	if !target.IsUntracked {
		t.Error("Orchestrator sessions should be untracked")
	}
}

// TestResolveTargetWorkerWorkspaceWithUntracked tests that resolveTarget correctly
// identifies untracked worker workspaces (no beads issue to fetch).
func TestResolveTargetWorkerWorkspaceWithUntracked(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")

	// Create a worker workspace with an untracked beads ID
	wsWorker := filepath.Join(workspaceDir, "og-feat-my-feature-06feb")
	if err := os.MkdirAll(wsWorker, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	untrackedID := "orch-go-untracked-1234567890"
	if err := os.WriteFile(filepath.Join(wsWorker, ".beads_id"), []byte(untrackedID), 0644); err != nil {
		t.Fatalf("Failed to create .beads_id: %v", err)
	}

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	target, err := resolveTarget("og-feat-my-feature-06feb", "")
	if err != nil {
		t.Fatalf("resolveTarget failed: %v", err)
	}

	if target.IsOrchestratorSession {
		t.Error("Expected IsOrchestratorSession to be false")
	}
	if target.AgentName != "og-feat-my-feature-06feb" {
		t.Errorf("Expected AgentName 'og-feat-my-feature-06feb', got '%s'", target.AgentName)
	}
	if target.BeadsID != untrackedID {
		t.Errorf("Expected BeadsID '%s', got '%s'", untrackedID, target.BeadsID)
	}
	if !target.IsUntracked {
		t.Error("Expected IsUntracked to be true for untracked beads ID")
	}
}

// ──────────────────────────────────────────────────────────────
// Phase 2: applySkipFiltering tests
// ──────────────────────────────────────────────────────────────

// TestApplySkipFilteringRemovesSkippedGates verifies that skip filtering
// correctly removes matching quality gates from the failure list.
// Core gates are never removed by skip filtering.
func TestApplySkipFilteringRemovesSkippedGates(t *testing.T) {
	gatesFailed := []string{
		verify.GateBuild,
		verify.GateModelConnection,
		verify.GateConstraint,
	}
	errors := []string{
		"build: go build failed",
		"model connection: no probe found",
		"constraint: scope exceeded",
	}

	skipConfig := SkipConfig{
		Build:           true,
		ModelConnection: true,
		Reason:          "docs-only change, no build needed",
	}

	target := &CompletionTarget{
		BeadsID:         "test-id",
		AgentName:       "test-agent",
		BeadsProjectDir: t.TempDir(),
	}

	applySkipFiltering(&gatesFailed, &errors, skipConfig, target)

	// Only constraint should remain (build and model_connection were skipped)
	if len(gatesFailed) != 1 {
		t.Errorf("Expected 1 remaining gate, got %d: %v", len(gatesFailed), gatesFailed)
	}
	if len(gatesFailed) > 0 && gatesFailed[0] != verify.GateConstraint {
		t.Errorf("Expected remaining gate to be constraint, got %s", gatesFailed[0])
	}
}

// TestApplySkipFilteringCoreGatesNeverSkipped verifies that core gates
// cannot be removed by skip filtering, even if skip flags are set.
func TestApplySkipFilteringCoreGatesNeverSkipped(t *testing.T) {
	gatesFailed := []string{
		verify.GatePhaseComplete,
		verify.GateCommitEvidence,
		verify.GateBuild,
	}
	errors := []string{
		"phase complete: not reported",
		"commit evidence: no commits",
		"build: failed",
	}

	// Even with core gate skip flags set, core gates should NOT be filtered
	skipConfig := SkipConfig{
		PhaseComplete:  true,
		CommitEvidence: true,
		Build:          true,
		Reason:         "attempting to skip everything",
	}

	target := &CompletionTarget{
		BeadsID:         "test-id",
		AgentName:       "test-agent",
		BeadsProjectDir: t.TempDir(),
	}

	applySkipFiltering(&gatesFailed, &errors, skipConfig, target)

	// Core gates (phase_complete, commit_evidence) should remain; only build should be skipped
	if len(gatesFailed) != 2 {
		t.Errorf("Expected 2 remaining gates (core gates unskippable), got %d: %v", len(gatesFailed), gatesFailed)
	}
}

// TestApplySkipFilteringNoMatchingGates verifies that skip filtering
// doesn't remove anything when no gates match the skip config.
func TestApplySkipFilteringNoMatchingGates(t *testing.T) {
	gatesFailed := []string{
		verify.GateTestEvidence,
		verify.GateGitDiff,
	}
	errors := []string{
		"test evidence: no test output",
		"git diff: mismatch",
	}

	skipConfig := SkipConfig{
		Build:  true,
		Reason: "build not needed",
	}

	target := &CompletionTarget{
		BeadsID:         "test-id",
		AgentName:       "test-agent",
		BeadsProjectDir: t.TempDir(),
	}

	applySkipFiltering(&gatesFailed, &errors, skipConfig, target)

	if len(gatesFailed) != 2 {
		t.Errorf("Expected 2 remaining gates, got %d", len(gatesFailed))
	}
}

// ──────────────────────────────────────────────────────────────
// Phase 5: resolveCloseReason tests
// ──────────────────────────────────────────────────────────────

// TestResolveCloseReasonDefault tests default close reason for regular agents.
func TestResolveCloseReasonDefault(t *testing.T) {
	target := &CompletionTarget{
		IsUntracked:           true,
		IsOrchestratorSession: false,
	}

	// Reset the global flag
	origReason := completeReason
	completeReason = ""
	defer func() { completeReason = origReason }()

	reason := resolveCloseReason(target)
	if reason != "Completed via orch complete" {
		t.Errorf("Expected 'Completed via orch complete', got '%s'", reason)
	}
}

// TestResolveCloseReasonOrchestrator tests close reason for orchestrator sessions.
func TestResolveCloseReasonOrchestrator(t *testing.T) {
	target := &CompletionTarget{
		IsUntracked:           true,
		IsOrchestratorSession: true,
	}

	origReason := completeReason
	completeReason = ""
	defer func() { completeReason = origReason }()

	reason := resolveCloseReason(target)
	if reason != "Orchestrator session completed" {
		t.Errorf("Expected 'Orchestrator session completed', got '%s'", reason)
	}
}

// TestResolveCloseReasonExplicit tests that explicit reason takes precedence.
func TestResolveCloseReasonExplicit(t *testing.T) {
	target := &CompletionTarget{
		IsUntracked: false,
		BeadsID:     "test-id",
	}

	origReason := completeReason
	completeReason = "My custom reason"
	defer func() { completeReason = origReason }()

	reason := resolveCloseReason(target)
	if reason != "My custom reason" {
		t.Errorf("Expected 'My custom reason', got '%s'", reason)
	}
}

// ──────────────────────────────────────────────────────────────
// Phase 6: cleanup helper tests
// ──────────────────────────────────────────────────────────────

// TestCleanupNoWorkspace tests that cleanup handles empty workspace path gracefully.
func TestCleanupNoWorkspace(t *testing.T) {
	target := &CompletionTarget{
		WorkspacePath: "",
	}

	outcome := runCleanup(target)
	if outcome == nil {
		t.Fatal("Expected non-nil outcome")
	}
	if outcome.ArchivedPath != "" {
		t.Error("Expected empty archived path for no workspace")
	}
}

// ──────────────────────────────────────────────────────────────
// CompletionTarget type tests
// ──────────────────────────────────────────────────────────────

// TestCompletionTargetFields verifies the CompletionTarget struct fields.
func TestCompletionTargetFields(t *testing.T) {
	target := CompletionTarget{
		Identifier:            "orch-go-abc1",
		BeadsID:               "orch-go-abc1",
		WorkspacePath:         "/tmp/workspace",
		AgentName:             "og-feat-test",
		BeadsProjectDir:       "/tmp/project",
		SourceProjectDir:      "/tmp/project",
		GitWorktreeDir:        "/tmp/project/.orch/worktrees/og-feat-test",
		GitBranch:             "agent/orch-go-abc1",
		IsOrchestratorSession: false,
		IsUntracked:           false,
		IsQuestion:            false,
		IsClosed:              false,
	}

	if target.Identifier != "orch-go-abc1" {
		t.Error("Identifier field mismatch")
	}
	if target.IsOrchestratorSession {
		t.Error("Expected IsOrchestratorSession false")
	}
}

// TestVerificationOutcomeDefaults verifies the VerificationOutcome struct defaults.
func TestVerificationOutcomeDefaults(t *testing.T) {
	outcome := VerificationOutcome{Passed: true}
	if !outcome.Passed {
		t.Error("Expected Passed true")
	}
	if outcome.SkillName != "" {
		t.Error("Expected empty SkillName")
	}
	if len(outcome.GatesFailed) != 0 {
		t.Error("Expected empty GatesFailed")
	}
}

func TestEnrichGitTargetFromManifest(t *testing.T) {
	workspace := t.TempDir()
	manifest := map[string]string{
		"workspace_name":     "og-feat-test",
		"skill":              "feature-impl",
		"source_project_dir": "/tmp/source",
		"project_dir":        "/tmp/source",
		"git_worktree_dir":   "/tmp/worktree",
		"git_branch":         "agent/test",
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, spawn.AgentManifestFilename), data, 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	target := &CompletionTarget{
		WorkspacePath:   workspace,
		BeadsProjectDir: "/tmp/fallback",
	}
	enrichGitTarget(target)

	if target.SourceProjectDir != "/tmp/source" {
		t.Fatalf("SourceProjectDir = %q, want %q", target.SourceProjectDir, "/tmp/source")
	}
	if target.GitWorktreeDir != "/tmp/worktree" {
		t.Fatalf("GitWorktreeDir = %q, want %q", target.GitWorktreeDir, "/tmp/worktree")
	}
	if target.GitBranch != "agent/test" {
		t.Fatalf("GitBranch = %q, want %q", target.GitBranch, "agent/test")
	}
}

func TestCompletionTargetGitDirFallback(t *testing.T) {
	target := &CompletionTarget{BeadsProjectDir: "/tmp/project"}
	if got := target.gitDir(); got != "/tmp/project" {
		t.Fatalf("gitDir() = %q, want %q", got, "/tmp/project")
	}
}
