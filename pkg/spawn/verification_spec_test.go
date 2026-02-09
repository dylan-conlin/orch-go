package spawn

import (
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
