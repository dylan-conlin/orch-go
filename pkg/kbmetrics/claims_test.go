package kbmetrics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractClaims_NumberedItems(t *testing.T) {
	content := `# Model: Test

## Critical Invariants

1. **Every convention without a gate will eventually be violated.** The knowledge system proves this.
2. **Models are the fundamental unit.** Without models, knowledge is homeless.
3. **Attention-primed attractors may be weaker.** Code attractors work through compilation.
`
	claims := ExtractClaims(content)
	if len(claims) != 3 {
		t.Errorf("expected 3 numbered claims, got %d: %v", len(claims), claims)
	}
	for _, c := range claims {
		if c.Type != ClaimTypeInvariant {
			t.Errorf("expected invariant type, got %s", c.Type)
		}
	}
}

func TestExtractClaims_BoldBullets(t *testing.T) {
	content := `## Core Mechanism

- **Orphan investigations** — work product with no structural connection.
- **Quick entry duplication** — confirmed duplicate pair.
- Regular bullet without bold prefix should not be a claim.
- **Synthesis backlog** — 4 clusters totaling 17 investigations.
`
	claims := ExtractClaims(content)
	if len(claims) != 3 {
		t.Errorf("expected 3 bold-bullet claims, got %d: %v", len(claims), claims)
	}
	for _, c := range claims {
		if c.Type != ClaimTypeAssertion {
			t.Errorf("expected assertion type, got %s", c.Type)
		}
	}
}

func TestExtractClaims_CoreClaim(t *testing.T) {
	content := `## Core Claim

Knowledge exhibits the same physics as code when multiple amnesiac agents contribute.

## Other Section
`
	claims := ExtractClaims(content)
	found := false
	for _, c := range claims {
		if c.Type == ClaimTypeCore {
			found = true
		}
	}
	if !found {
		t.Error("expected to find a core claim")
	}
}

func TestExtractClaims_TableRows(t *testing.T) {
	content := `## Metrics

| Code Metric | Knowledge Equivalent | Status |
|-------------|---------------------|--------|
| Lines of code per file | Claims per model | **Not tracked** |
| File bloat (>1,500 lines) | Model bloat (>N claims) | **Not tracked** |
| Fix:feat ratio | Contradiction:extension ratio | **Not tracked** |
`
	claims := ExtractClaims(content)
	if len(claims) != 3 {
		t.Errorf("expected 3 table claims, got %d: %v", len(claims), claims)
	}
	for _, c := range claims {
		if c.Type != ClaimTypeData {
			t.Errorf("expected data type, got %s", c.Type)
		}
	}
}

func TestExtractClaims_IgnoresNonClaims(t *testing.T) {
	content := `# Model: Test

Just regular text here, not making claims.

## References

- some-file.md
- another-file.md

## Evolution

**2026-01-10:** Something happened.
**2026-02-14:** Another thing.
`
	claims := ExtractClaims(content)
	if len(claims) != 0 {
		t.Errorf("expected 0 claims from non-claim content, got %d: %v", len(claims), claims)
	}
}

func TestExtractClaims_ConstraintSection(t *testing.T) {
	content := `## Constraints

### Why Can't Plugins Analyze LLM Response Text?

**Constraint:** OpenCode plugins only see tool calls, not free-text responses.

**Implication:** All detection must use tool usage as behavioral proxies.

### Why Behavioral Proxies?

**Constraint:** Since plugins can't see LLM text, detection requires inferring.

**Implication:** Metrics are proxies, not direct measurements.
`
	claims := ExtractClaims(content)
	// Constraints and implications are claims
	if len(claims) < 2 {
		t.Errorf("expected at least 2 constraint claims, got %d: %v", len(claims), claims)
	}
}

func TestAnalyzeModels(t *testing.T) {
	dir := t.TempDir()
	modelsDir := filepath.Join(dir, ".kb", "models")

	// Create two models
	model1Dir := filepath.Join(modelsDir, "small-model")
	model2Dir := filepath.Join(modelsDir, "large-model")
	os.MkdirAll(model1Dir, 0755)
	os.MkdirAll(model2Dir, 0755)

	small := `# Model: Small

## Core Claim

One simple claim.

## Critical Invariants

1. **First invariant.** Details.
`
	large := `# Model: Large

## Core Claim

Knowledge accretion claim.

## Critical Invariants

1. **First.** Details.
2. **Second.** Details.
3. **Third.** Details.

## Core Mechanism

- **Pattern A** — description.
- **Pattern B** — description.
- **Pattern C** — description.
- **Pattern D** — description.
- **Pattern E** — description.

## Constraints

### Why X?

**Constraint:** Because of Y.

**Implication:** Must do Z.
`
	os.WriteFile(filepath.Join(model1Dir, "model.md"), []byte(small), 0644)
	os.WriteFile(filepath.Join(model2Dir, "model.md"), []byte(large), 0644)

	results, err := AnalyzeModels(modelsDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Results should be sorted by claim count descending
	if results[0].ClaimCount < results[1].ClaimCount {
		t.Errorf("expected results sorted by claim count desc, got %d < %d",
			results[0].ClaimCount, results[1].ClaimCount)
	}
}

func TestBloatLevel(t *testing.T) {
	tests := []struct {
		count int
		want  string
	}{
		{10, "healthy"},
		{29, "healthy"},
		{30, "warning"},
		{49, "warning"},
		{50, "critical"},
		{100, "critical"},
	}

	for _, tt := range tests {
		got := BloatLevel(tt.count)
		if got != tt.want {
			t.Errorf("BloatLevel(%d) = %s, want %s", tt.count, got, tt.want)
		}
	}
}

func TestExtractClaims_FailureModeSection(t *testing.T) {
	content := `## Why This Fails

### Failure Mode 1: Knowledge Accretion Outpaces Synthesis

**Symptom:** Agent spawned receives wrong context.

**Root cause:** Query runs from wrong CWD.

### Failure Mode 2: Advisory Gates Are Non-Gates

**Symptom:** Conventions degrade under pressure.

**Root cause:** No enforcement mechanism.
`
	claims := ExtractClaims(content)
	// Failure modes with symptom/root cause are claims
	if len(claims) < 2 {
		t.Errorf("expected at least 2 failure mode claims, got %d: %v", len(claims), claims)
	}
}
