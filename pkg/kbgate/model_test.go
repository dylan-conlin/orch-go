package kbgate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckModel_MissingFile(t *testing.T) {
	result := CheckModel("/nonexistent/model.md")
	if result.Pass {
		t.Error("expected failure for missing file")
	}
	assertHasVerdict(t, result, "FILE_NOT_FOUND")
}

func TestCheckModel_MissingClaimTable(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.md")
	content := `# Model: Test

## Summary
Some summary.

## Core Mechanism
Some mechanism.
`
	os.WriteFile(modelPath, []byte(content), 0644)

	result := CheckModel(modelPath)
	if result.Pass {
		t.Error("expected failure for missing claim table")
	}
	assertHasVerdict(t, result, "MISSING_CLAIM_TABLE")
}

func TestCheckModel_MissingCanonTable(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.md")
	content := `# Model: Test

## Summary
Some summary.

## Claim Ledger

| claim_id | claim_text | claim_type | scope | novelty_level | evidence_refs |
|----------|------------|------------|-------|---------------|---------------|
| C1 | Conventions decay under throughput | observation | code workflows | restatement | inv:2026-01-01 |
`
	os.WriteFile(modelPath, []byte(content), 0644)

	result := CheckModel(modelPath)
	if result.Pass {
		t.Error("expected failure for missing canonicalization table")
	}
	assertHasVerdict(t, result, "MISSING_CANON_TABLE")
}

func TestCheckModel_ValidClaimTable(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.md")
	content := `# Model: Test

## Summary
Some summary.

## Claim Ledger

| claim_id | claim_text | claim_type | scope | novelty_level | evidence_refs |
|----------|------------|------------|-------|---------------|---------------|
| C1 | Conventions decay under throughput | observation | code workflows | restatement | inv:2026-01-01 |

## Vocabulary Canonicalization

| term | plain_language | nearest_existing_concepts | claimed_delta | verdict |
|------|----------------|---------------------------|---------------|---------|
| accretion | things grow over time | tech debt, entropy | none | restatement |
`
	os.WriteFile(modelPath, []byte(content), 0644)

	result := CheckModel(modelPath)
	if hasVerdict(result, "MISSING_CLAIM_TABLE") {
		t.Error("should not report missing claim table when present")
	}
	if hasVerdict(result, "MISSING_CANON_TABLE") {
		t.Error("should not report missing canon table when present")
	}
}

func TestCheckModel_InvalidClaimType(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.md")
	content := `# Model: Test

## Claim Ledger

| claim_id | claim_text | claim_type | scope | novelty_level | evidence_refs |
|----------|------------|------------|-------|---------------|---------------|
| C1 | Some claim | invalid_type | code workflows | restatement | inv:2026-01-01 |

## Vocabulary Canonicalization

| term | plain_language | nearest_existing_concepts | claimed_delta | verdict |
|------|----------------|---------------------------|---------------|---------|
| accretion | things grow | tech debt | none | restatement |
`
	os.WriteFile(modelPath, []byte(content), 0644)

	result := CheckModel(modelPath)
	if result.Pass {
		t.Error("expected failure for invalid claim type")
	}
	assertHasVerdict(t, result, "INVALID_CLAIM_ENTRY")
}

func TestCheckModel_InvalidNoveltyLevel(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.md")
	content := `# Model: Test

## Claim Ledger

| claim_id | claim_text | claim_type | scope | novelty_level | evidence_refs |
|----------|------------|------------|-------|---------------|---------------|
| C1 | Some claim | observation | code workflows | groundbreaking | inv:2026-01-01 |

## Vocabulary Canonicalization

| term | plain_language | nearest_existing_concepts | claimed_delta | verdict |
|------|----------------|---------------------------|---------------|---------|
| x | y | z | none | restatement |
`
	os.WriteFile(modelPath, []byte(content), 0644)

	result := CheckModel(modelPath)
	if result.Pass {
		t.Error("expected failure for invalid novelty level")
	}
	assertHasVerdict(t, result, "INVALID_CLAIM_ENTRY")
}

func TestCheckModel_EmptyEvidenceRefs(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.md")
	content := `# Model: Test

## Claim Ledger

| claim_id | claim_text | claim_type | scope | novelty_level | evidence_refs |
|----------|------------|------------|-------|---------------|---------------|
| C1 | Some claim | mechanism | code workflows | synthesis |  |

## Vocabulary Canonicalization

| term | plain_language | nearest_existing_concepts | claimed_delta | verdict |
|------|----------------|---------------------------|---------------|---------|
| x | y | z | none | restatement |
`
	os.WriteFile(modelPath, []byte(content), 0644)

	result := CheckModel(modelPath)
	if result.Pass {
		t.Error("expected failure for empty evidence refs on non-observation claim")
	}
	assertHasVerdict(t, result, "MISSING_EVIDENCE")
}

func TestCheckModel_ObservationAllowsEmptyEvidence(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.md")
	content := `# Model: Test

## Claim Ledger

| claim_id | claim_text | claim_type | scope | novelty_level | evidence_refs |
|----------|------------|------------|-------|---------------|---------------|
| C1 | We saw files growing | observation | code workflows | restatement |  |

## Vocabulary Canonicalization

| term | plain_language | nearest_existing_concepts | claimed_delta | verdict |
|------|----------------|---------------------------|---------------|---------|
| x | y | z | none | restatement |
`
	os.WriteFile(modelPath, []byte(content), 0644)

	result := CheckModel(modelPath)
	if hasVerdict(result, "MISSING_EVIDENCE") {
		t.Error("observations should not require evidence_refs")
	}
}

func TestCheckModel_VocabInflation(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.md")
	content := `# Model: Test

## Claim Ledger

| claim_id | claim_text | claim_type | scope | novelty_level | evidence_refs |
|----------|------------|------------|-------|---------------|---------------|
| C1 | Things grow | observation | local | restatement | inv:2026-01-01 |

## Vocabulary Canonicalization

| term | plain_language | nearest_existing_concepts | claimed_delta | verdict |
|------|----------------|---------------------------|---------------|---------|
| accretion dynamics | things grow because addition is cheaper | tech debt, institutional drift |  | review |
`
	os.WriteFile(modelPath, []byte(content), 0644)

	result := CheckModel(modelPath)
	assertHasVerdict(t, result, "VOCABULARY_INFLATION")
}

func TestCheckModel_VocabMissingPriorArt(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.md")
	content := `# Model: Test

## Claim Ledger

| claim_id | claim_text | claim_type | scope | novelty_level | evidence_refs |
|----------|------------|------------|-------|---------------|---------------|
| C1 | Things grow | observation | local | restatement | inv:2026-01-01 |

## Vocabulary Canonicalization

| term | plain_language | nearest_existing_concepts | claimed_delta | verdict |
|------|----------------|---------------------------|---------------|---------|
| accretion dynamics | things grow | | predicts where decay appears | review |
`
	os.WriteFile(modelPath, []byte(content), 0644)

	result := CheckModel(modelPath)
	assertHasVerdict(t, result, "MISSING_PRIOR_ART")
}

func TestCheckModel_FullPass(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.md")
	content := `# Model: Test

## Summary
A working model for how conventions decay.

## Claim Ledger

| claim_id | claim_text | claim_type | scope | novelty_level | evidence_refs |
|----------|------------|------------|-------|---------------|---------------|
| C1 | Conventions decay under agent throughput | observation | code workflows | restatement | inv:2026-01-01 |
| C2 | Gate enforcement reduces decay rate | mechanism | orch-go | synthesis | inv:2026-01-15, inv:2026-02-01 |

## Vocabulary Canonicalization

| term | plain_language | nearest_existing_concepts | claimed_delta | verdict |
|------|----------------|---------------------------|---------------|---------|
| accretion | things accumulate over time | tech debt, entropy | predicts where decay concentrates | review |
| gate enforcement | automated blocking checks | CI checks, linting | applied to knowledge artifacts not just code | review |

## Core Mechanism
Details here.
`
	os.WriteFile(modelPath, []byte(content), 0644)

	result := CheckModel(modelPath)
	if !result.Pass {
		t.Errorf("expected full pass, got: %v", verdictCodes(result))
	}
}

func TestCheckModel_EndogenousEvidenceWarning(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.md")
	content := `# Model: Test

## Claim Ledger

| claim_id | claim_text | claim_type | scope | novelty_level | evidence_refs |
|----------|------------|------------|-------|---------------|---------------|
| C1 | Knowledge has structure | generalization | cross-domain | novel | model:test, probe:2026-01-01 |

## Vocabulary Canonicalization

| term | plain_language | nearest_existing_concepts | claimed_delta | verdict |
|------|----------------|---------------------------|---------------|---------|
| x | y | z | w | review |
`
	os.WriteFile(modelPath, []byte(content), 0644)

	result := CheckModel(modelPath)
	// Model gate warns (not blocks) on endogenous evidence
	assertHasVerdict(t, result, "ENDOGENOUS_EVIDENCE_WARNING")
}

func TestCheckModel_ClaimTableMissingColumns(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.md")
	content := `# Model: Test

## Claim Ledger

| claim_id | claim_text |
|----------|------------|
| C1 | Some claim |

## Vocabulary Canonicalization

| term | plain_language | nearest_existing_concepts | claimed_delta | verdict |
|------|----------------|---------------------------|---------------|---------|
| x | y | z | w | review |
`
	os.WriteFile(modelPath, []byte(content), 0644)

	result := CheckModel(modelPath)
	if result.Pass {
		t.Error("expected failure for missing columns in claim table")
	}
	assertHasVerdict(t, result, "INVALID_CLAIM_TABLE")
}

func TestCheckModel_CanonTableMissingColumns(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.md")
	content := `# Model: Test

## Claim Ledger

| claim_id | claim_text | claim_type | scope | novelty_level | evidence_refs |
|----------|------------|------------|-------|---------------|---------------|
| C1 | Claim | observation | local | restatement | inv:x |

## Vocabulary Canonicalization

| term | plain_language |
|------|----------------|
| x | y |
`
	os.WriteFile(modelPath, []byte(content), 0644)

	result := CheckModel(modelPath)
	if result.Pass {
		t.Error("expected failure for missing columns in canon table")
	}
	assertHasVerdict(t, result, "INVALID_CANON_TABLE")
}

func TestCheckModel_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.md")
	content := `# Model: Test

## Claim Ledger

| claim_id | claim_text | claim_type | scope | novelty_level | evidence_refs |
|----------|------------|------------|-------|---------------|---------------|
| C1 | Things | observation | local | restatement | inv:x |

## Vocabulary Canonicalization

| term | plain_language | nearest_existing_concepts | claimed_delta | verdict |
|------|----------------|---------------------------|---------------|---------|
| x | y | z | w | review |
`
	os.WriteFile(modelPath, []byte(content), 0644)

	result := CheckModel(modelPath)
	// Just verify it returns a valid GateResult
	if len(result.Verdicts) > 0 && result.Verdicts[0].Code == "" {
		t.Error("verdicts should have codes")
	}
}

func TestFormatModelResult(t *testing.T) {
	result := GateResult{
		Pass: true,
		Verdicts: []Verdict{
			{Code: "CLAIM_TABLE", Status: "pass", Note: "found 2 claims"},
		},
	}
	output := FormatModelResult(result)
	if output == "" {
		t.Error("expected non-empty output")
	}
}
