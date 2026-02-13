package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateVerificationSpecSkeleton_UsesSyntaxValidFailingPlaceholders(t *testing.T) {
	cfg := &Config{
		SkillName:     "feature-impl",
		Tier:          TierLight,
		WorkspaceName: "og-feat-proof-09feb",
		BeadsID:       "orch-go-proof1",
	}

	content, err := GenerateVerificationSpecSkeleton(cfg)
	if err != nil {
		t.Fatalf("GenerateVerificationSpecSkeleton failed: %v", err)
	}

	checks := []string{
		"TODO: replace build command in VERIFICATION_SPEC.yaml",
		"TODO: replace test command in VERIFICATION_SPEC.yaml",
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Fatalf("expected skeleton to contain %q", check)
		}
	}

	if strings.Contains(content, "<fill") {
		t.Fatalf("expected skeleton to avoid syntax-invalid <fill ...> placeholders: %s", content)
	}
}

func TestPlaceholderCommand_HasValidBashSyntax(t *testing.T) {
	if err := validateBashSyntax(placeholderCommand("build")); err != nil {
		t.Fatalf("expected placeholder command to be syntax-valid, got: %v", err)
	}
}

func TestValidateVerificationEntryCommands_RejectsMalformedSyntax(t *testing.T) {
	entries := []verificationSpecSkeletonRow{
		{
			ID:      "bad",
			Method:  "cli_smoke",
			Command: "echo hi )",
		},
	}

	err := validateVerificationEntryCommands(entries)
	if err == nil {
		t.Fatal("expected malformed syntax error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid bash syntax") {
		t.Fatalf("expected invalid bash syntax error, got: %v", err)
	}
}

func TestGenerateVerificationSpecSkeleton_DetectsGoBuildAndTestCommands(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	cfg := &Config{
		SkillName:     "feature-impl",
		Tier:          TierLight,
		WorkspaceName: "og-feat-proof-09feb",
		BeadsID:       "orch-go-proof1",
		ProjectDir:    projectDir,
	}

	content, err := GenerateVerificationSpecSkeleton(cfg)
	if err != nil {
		t.Fatalf("GenerateVerificationSpecSkeleton failed: %v", err)
	}

	if !strings.Contains(content, "command: go build ./...") {
		t.Fatalf("expected go build command in skeleton, got: %s", content)
	}
	if !strings.Contains(content, "command: go test ./...") {
		t.Fatalf("expected go test command in skeleton, got: %s", content)
	}
	if !strings.Contains(content, "cwd: "+verificationSpecRuntimeCWDToken) {
		t.Fatalf("expected runtime cwd token in skeleton, got: %s", content)
	}
	if strings.Contains(content, "TODO: replace build command") || strings.Contains(content, "TODO: replace test command") {
		t.Fatalf("expected no TODO placeholders for go project, got: %s", content)
	}
}

// TestWriteVerificationSpecSkeleton_AlwaysWritesFresh verifies that
// WriteVerificationSpecSkeleton always writes a fresh skeleton even when
// a previous VERIFICATION_SPEC.yaml exists. This prevents stale specs
// from being inherited when workspaces are recycled (reused worktrees).
//
// Regression test for: orch-go-1ugvu (stale VERIFICATION_SPEC inheritance)
func TestWriteVerificationSpecSkeleton_AlwaysWritesFresh(t *testing.T) {
	projectDir := t.TempDir()

	// Create workspace directory
	workspaceName := "og-feat-test-recycled"
	workspacePath := filepath.Join(projectDir, ".orch", "workspace", workspaceName)
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Simulate a "stale" VERIFICATION_SPEC.yaml from a prior task
	staleSpecPath := filepath.Join(workspacePath, verificationSpecFileName)
	staleContent := `# Stale spec from prior task
version: 1
scope:
  beads_id: old-task-123
  workspace: og-old-task
  skill: old-skill
verification:
  - id: verify-stale
    method: cli_smoke
    command: make release-stable
`
	if err := os.WriteFile(staleSpecPath, []byte(staleContent), 0644); err != nil {
		t.Fatalf("failed to write stale spec: %v", err)
	}

	// Now write a fresh spec for a NEW task
	cfg := &Config{
		SkillName:     "feature-impl",
		Tier:          TierLight,
		WorkspaceName: workspaceName,
		BeadsID:       "new-task-456",
		ProjectDir:    projectDir,
	}

	if err := WriteVerificationSpecSkeleton(cfg); err != nil {
		t.Fatalf("WriteVerificationSpecSkeleton failed: %v", err)
	}

	// Read the written spec
	content, err := os.ReadFile(staleSpecPath)
	if err != nil {
		t.Fatalf("failed to read verification spec: %v", err)
	}

	// Verify it contains the NEW task's info, not the stale info
	if strings.Contains(string(content), "old-task-123") {
		t.Fatal("expected fresh spec, but found stale beads_id 'old-task-123'")
	}
	if strings.Contains(string(content), "og-old-task") {
		t.Fatal("expected fresh spec, but found stale workspace 'og-old-task'")
	}
	if strings.Contains(string(content), "make release-stable") {
		t.Fatal("expected fresh spec, but found stale command 'make release-stable'")
	}

	// Verify it contains the NEW task's info
	if !strings.Contains(string(content), "new-task-456") {
		t.Fatalf("expected spec to contain new beads_id 'new-task-456', got: %s", content)
	}
	if !strings.Contains(string(content), workspaceName) {
		t.Fatalf("expected spec to contain workspace name '%s', got: %s", workspaceName, content)
	}
}
