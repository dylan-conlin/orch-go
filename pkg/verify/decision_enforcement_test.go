package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyDecisionEnforcement_NonArchitect(t *testing.T) {
	result := VerifyDecisionEnforcement("", "feature-impl", "")
	if result != nil {
		t.Errorf("expected nil for non-architect skill, got %+v", result)
	}
}

func TestVerifyDecisionEnforcement_NoSynthesis(t *testing.T) {
	dir := t.TempDir()
	result := VerifyDecisionEnforcement(dir, "architect", "")
	if result != nil {
		t.Errorf("expected nil when no SYNTHESIS.md, got %+v", result)
	}
}

func TestVerifyDecisionEnforcement_NoDecisionRefs(t *testing.T) {
	dir := t.TempDir()
	writeSynthesis(t, dir, `# SYNTHESIS

## TLDR
Designed a new cache layer.

## Next
**Recommendation:** implement
`)
	result := VerifyDecisionEnforcement(dir, "architect", "")
	if result != nil {
		t.Errorf("expected nil when no decision references, got %+v", result)
	}
}

func TestVerifyDecisionEnforcement_DecisionWithEnforcement(t *testing.T) {
	dir := t.TempDir()
	projectDir := t.TempDir()

	// Create decision file with enforcement field
	decDir := filepath.Join(projectDir, ".kb", "decisions")
	os.MkdirAll(decDir, 0o755)
	decPath := filepath.Join(decDir, "2026-03-20-test-decision.md")
	os.WriteFile(decPath, []byte(`# Decision: Test Decision

**Date:** 2026-03-20
**Status:** Accepted
**Enforcement:** gate

## Context
Test context.
`), 0o644)

	writeSynthesis(t, dir, `# SYNTHESIS

## TLDR
Designed something.

## Knowledge
Created decision: .kb/decisions/2026-03-20-test-decision.md

## Next
**Recommendation:** implement
`)

	result := VerifyDecisionEnforcement(dir, "architect", projectDir)
	if result != nil && !result.Passed {
		t.Errorf("expected passing result when enforcement declared, got errors: %v", result.Errors)
	}
}

func TestVerifyDecisionEnforcement_DecisionMissingEnforcement(t *testing.T) {
	dir := t.TempDir()
	projectDir := t.TempDir()

	// Create decision file WITHOUT enforcement field
	decDir := filepath.Join(projectDir, ".kb", "decisions")
	os.MkdirAll(decDir, 0o755)
	decPath := filepath.Join(decDir, "2026-03-20-test-decision.md")
	os.WriteFile(decPath, []byte(`# Decision: Test Decision

**Date:** 2026-03-20
**Status:** Accepted

## Context
Test context.
`), 0o644)

	writeSynthesis(t, dir, `# SYNTHESIS

## TLDR
Designed something.

## Knowledge
Created decision: .kb/decisions/2026-03-20-test-decision.md

## Next
**Recommendation:** implement
`)

	result := VerifyDecisionEnforcement(dir, "architect", projectDir)
	if result == nil {
		t.Fatal("expected non-nil result when enforcement missing")
	}
	if result.Passed {
		t.Error("expected failure when enforcement missing")
	}
	if len(result.GatesFailed) == 0 || result.GatesFailed[0] != GateDecisionEnforcement {
		t.Errorf("expected gate %s in failures, got %v", GateDecisionEnforcement, result.GatesFailed)
	}
}

func TestVerifyDecisionEnforcement_InvalidEnforcementType(t *testing.T) {
	dir := t.TempDir()
	projectDir := t.TempDir()

	decDir := filepath.Join(projectDir, ".kb", "decisions")
	os.MkdirAll(decDir, 0o755)
	decPath := filepath.Join(decDir, "2026-03-20-test-decision.md")
	os.WriteFile(decPath, []byte(`# Decision: Test Decision

**Date:** 2026-03-20
**Status:** Accepted
**Enforcement:** maybe

## Context
Test context.
`), 0o644)

	writeSynthesis(t, dir, `# SYNTHESIS

## TLDR
Designed something.

## Knowledge
Created decision: .kb/decisions/2026-03-20-test-decision.md

## Next
**Recommendation:** implement
`)

	result := VerifyDecisionEnforcement(dir, "architect", projectDir)
	if result == nil {
		t.Fatal("expected non-nil result for invalid enforcement type")
	}
	if result.Passed {
		t.Error("expected failure for invalid enforcement type")
	}
}

func TestValidEnforcementTypes(t *testing.T) {
	validTypes := []string{"gate", "hook", "convention", "context-only"}
	for _, et := range validTypes {
		if !IsValidEnforcementType(et) {
			t.Errorf("expected %q to be valid enforcement type", et)
		}
	}

	invalidTypes := []string{"maybe", "none", ""}
	for _, et := range invalidTypes {
		if IsValidEnforcementType(et) {
			t.Errorf("expected %q to be invalid enforcement type", et)
		}
	}
}

func writeSynthesis(t *testing.T, dir, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, "SYNTHESIS.md"), []byte(content), 0o644)
	if err != nil {
		t.Fatal(err)
	}
}
