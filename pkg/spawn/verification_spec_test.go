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
