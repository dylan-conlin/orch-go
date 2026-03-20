package completion

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseArtifact_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	yaml := `verification: "go test ./pkg/completion/ — 5 passed"
finding: "Added COMPLETION.yaml validator with per-type field enforcement"
kb_atom: ".kb/decisions/2026-03-15-completion-artifact-validator.md"
follow_up: "Wire into daemon completion loop"
placement: "pkg/completion/ — new package in verification pipeline"
`
	writeFile(t, dir, "COMPLETION.yaml", yaml)

	art, err := ParseArtifact(dir)
	if err != nil {
		t.Fatalf("ParseArtifact: %v", err)
	}
	if art.Verification != "go test ./pkg/completion/ — 5 passed" {
		t.Errorf("Verification = %q", art.Verification)
	}
	if art.Finding != "Added COMPLETION.yaml validator with per-type field enforcement" {
		t.Errorf("Finding = %q", art.Finding)
	}
	if art.KBAtom != ".kb/decisions/2026-03-15-completion-artifact-validator.md" {
		t.Errorf("KBAtom = %q", art.KBAtom)
	}
	if art.FollowUp != "Wire into daemon completion loop" {
		t.Errorf("FollowUp = %q", art.FollowUp)
	}
	if art.Placement != "pkg/completion/ — new package in verification pipeline" {
		t.Errorf("Placement = %q", art.Placement)
	}
}

func TestParseArtifact_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := ParseArtifact(dir)
	if err == nil {
		t.Fatal("expected error for missing COMPLETION.yaml")
	}
}

func TestParseArtifact_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "COMPLETION.yaml", "verification: [unterminated")
	_, err := ParseArtifact(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestValidateArtifact_Feature_AllRequired(t *testing.T) {
	art := &Artifact{
		Verification: "tests pass",
		Finding:      "built feature",
		KBAtom:       ".kb/decisions/foo.md",
		FollowUp:     "none",
		Placement:    "pkg/foo/",
	}
	errs := ValidateArtifact(art, "feature")
	if len(errs) != 0 {
		t.Errorf("expected no errors for complete feature artifact, got: %v", errs)
	}
}

func TestValidateArtifact_Feature_MissingFields(t *testing.T) {
	art := &Artifact{
		Verification: "tests pass",
		// Missing: Finding, KBAtom, FollowUp, Placement
	}
	errs := ValidateArtifact(art, "feature")
	if len(errs) != 4 {
		t.Errorf("expected 4 errors for feature with 4 missing fields, got %d: %v", len(errs), errs)
	}
}

func TestValidateArtifact_Bug_RequiredSubset(t *testing.T) {
	// Bug requires: verification, finding, follow_up
	art := &Artifact{
		Verification: "repro confirmed fixed",
		Finding:      "null pointer in handler",
		FollowUp:     "none",
	}
	errs := ValidateArtifact(art, "bug")
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid bug artifact, got: %v", errs)
	}
}

func TestValidateArtifact_Task_RequiredSubset(t *testing.T) {
	// Task requires: verification, finding
	art := &Artifact{
		Verification: "make build passes",
		Finding:      "refactored extraction",
	}
	errs := ValidateArtifact(art, "task")
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid task artifact, got: %v", errs)
	}
}

func TestValidateArtifact_Investigation_RequiredSubset(t *testing.T) {
	// Investigation requires: finding, kb_atom
	art := &Artifact{
		Finding: "discovered race condition",
		KBAtom:  ".kb/investigations/foo.md",
	}
	errs := ValidateArtifact(art, "investigation")
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid investigation artifact, got: %v", errs)
	}
}

func TestValidateArtifact_Question_RequiredSubset(t *testing.T) {
	// Question requires: finding
	art := &Artifact{
		Finding: "answered: use approach B",
	}
	errs := ValidateArtifact(art, "question")
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid question artifact, got: %v", errs)
	}
}

func TestValidateArtifact_UnknownType_DefaultsToTask(t *testing.T) {
	// Unknown type falls back to task requirements
	art := &Artifact{
		Verification: "ok",
		Finding:      "ok",
	}
	errs := ValidateArtifact(art, "epic")
	if len(errs) != 0 {
		t.Errorf("expected no errors for unknown type with task-level fields, got: %v", errs)
	}
}

func TestValidateArtifact_PlaceholderDetection(t *testing.T) {
	art := &Artifact{
		Verification: "TODO",
		Finding:      "",
		KBAtom:       "TBD",
		FollowUp:     "todo",
		Placement:    "tbd",
	}
	errs := ValidateArtifact(art, "feature")
	// All 5 should fail — placeholders count as empty
	if len(errs) != 5 {
		t.Errorf("expected 5 errors for all-placeholder feature, got %d: %v", len(errs), errs)
	}
}

func TestPrePopulateFromSynthesis(t *testing.T) {
	dir := t.TempDir()
	synthesis := `# SYNTHESIS

**Agent:** test-agent
**Issue:** orch-go-test
**Outcome:** success

## TLDR

Built the completion artifact validator.

## Delta (What Changed)

- Added pkg/completion/artifact.go
- Added pkg/completion/artifact_test.go

## Evidence (What Was Observed)

go test ./pkg/completion/ — 8 passed, 0 failed

## Knowledge (What Was Learned)

Created .kb/decisions/2026-03-15-artifact-validator.md

## Next (What Should Happen)

**Recommendation:** close

### Follow-up Work
- Wire artifact gate into daemon completion loop
`
	writeFile(t, dir, "SYNTHESIS.md", synthesis)

	art, err := PrePopulateFromSynthesis(dir)
	if err != nil {
		t.Fatalf("PrePopulateFromSynthesis: %v", err)
	}

	if art.Verification == "" {
		t.Error("expected verification to be pre-populated from Evidence")
	}
	if art.Finding == "" {
		t.Error("expected finding to be pre-populated from TLDR/Delta")
	}
	if art.FollowUp == "" {
		t.Error("expected follow_up to be pre-populated from Next")
	}
}

func TestPrePopulateFromSynthesis_NoSynthesis(t *testing.T) {
	dir := t.TempDir()
	art, err := PrePopulateFromSynthesis(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return empty artifact, not error
	if art.Finding != "" || art.Verification != "" {
		t.Error("expected empty artifact when no SYNTHESIS.md")
	}
}

func TestCheckArtifact_Integration(t *testing.T) {
	dir := t.TempDir()
	yaml := `verification: "tests pass"
finding: "built feature"
kb_atom: ".kb/foo.md"
follow_up: "none"
placement: "pkg/completion/"
`
	writeFile(t, dir, "COMPLETION.yaml", yaml)

	result := CheckArtifact(dir, "feature")
	if !result.Passed {
		t.Errorf("expected pass, got errors: %v", result.Errors)
	}
}

func TestCheckArtifact_MissingYAML_NoSynthesis(t *testing.T) {
	dir := t.TempDir()
	result := CheckArtifact(dir, "feature")
	if result.Passed {
		t.Error("expected failure when both COMPLETION.yaml and SYNTHESIS.md are missing")
	}
	if len(result.Errors) == 0 {
		t.Error("expected at least one error")
	}
}

func TestCheckArtifact_MissingYAML_FallbackToSynthesis(t *testing.T) {
	dir := t.TempDir()
	// Create SYNTHESIS.md with enough content to derive artifact fields
	synthesis := `# SYNTHESIS

**Agent:** test-agent
**Issue:** orch-go-test
**Outcome:** success

## TLDR

Fixed the authentication bug in session handler.

## Delta (What Changed)

- Fixed null pointer in auth middleware

## Evidence (What Was Observed)

go test ./pkg/auth/ — 12 passed, 0 failed

## Knowledge (What Was Learned)

Created .kb/decisions/2026-03-20-auth-fix.md

## Next (What Should Happen)

**Recommendation:** close

### Follow-up Work
- Monitor for regressions
`
	writeFile(t, dir, "SYNTHESIS.md", synthesis)

	// Bug type requires: verification, finding, follow_up — all derivable from synthesis
	result := CheckArtifact(dir, "bug")
	if !result.Passed {
		t.Errorf("expected pass when SYNTHESIS.md provides required fields, got errors: %v", result.Errors)
	}
}

func TestCheckArtifact_MissingYAML_SynthesisMissingRequiredFields(t *testing.T) {
	dir := t.TempDir()
	// SYNTHESIS.md with only TLDR (no Evidence section)
	synthesis := `# SYNTHESIS

## TLDR

Did some work.
`
	writeFile(t, dir, "SYNTHESIS.md", synthesis)

	// Feature type requires verification (Evidence), kb_atom, follow_up, placement
	// Only finding (from TLDR) will be populated
	result := CheckArtifact(dir, "feature")
	if result.Passed {
		t.Error("expected failure when SYNTHESIS.md doesn't provide all required feature fields")
	}
}

func TestCheckArtifact_MissingYAML_EmptySynthesis(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "SYNTHESIS.md", "")

	result := CheckArtifact(dir, "task")
	if result.Passed {
		t.Error("expected failure for empty SYNTHESIS.md")
	}
}

func TestCheckArtifact_ValidationFailure(t *testing.T) {
	dir := t.TempDir()
	yaml := `verification: "tests pass"
finding: ""
`
	writeFile(t, dir, "COMPLETION.yaml", yaml)

	result := CheckArtifact(dir, "feature")
	if result.Passed {
		t.Error("expected failure for incomplete feature artifact")
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
