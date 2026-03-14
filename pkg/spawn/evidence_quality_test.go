package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEvidenceQuality_SurfacedInModelInjection verifies that when a model's
// Critical Invariants section contains **Evidence quality:** annotations,
// they are preserved in the spawn context injection.
//
// This is Layer 1 of confidence propagation: evidence quality visible to agents
// via kb context → spawn context, without requiring any behavioral compliance.
func TestEvidenceQuality_SurfacedInModelInjection(t *testing.T) {
	dir := t.TempDir()
	modelDir := filepath.Join(dir, ".kb", "models", "test-model")
	os.MkdirAll(filepath.Join(modelDir, "probes"), 0755)

	modelContent := `# Model: Test Model

## Summary

This model tracks behavioral constraints in the orchestrator skill system.

## Critical Invariants

1. **Every convention without a gate will eventually be violated.** The knowledge system proves this repeatedly — behavioral constraints alone have 0% enforcement guarantee.

**Evidence quality:** Replicated (accretion gate, probe merge gate, hotspot gate — 3 independent confirmations).

2. **Dilution budget is approximately 4 behavioral constraints.** Adding more than 4 causes overall compliance to drop below 50%.

**Evidence quality:** Single-source measured (N=3, replication failed Mar 4). HYPOTHESIZED — do not cite as established.

3. **Knowledge content transfers reliably.** Factual content in skill context is absorbed at high rates.

**Evidence quality:** Replicated (4 independent sources across 67 measured claims).

## Why This Fails

- Agents treat all claims equally regardless of evidence quality
- Behavioral constraints without gates degrade over time
`
	modelPath := filepath.Join(modelDir, "model.md")
	os.WriteFile(modelPath, []byte(modelContent), 0644)

	// Extract sections as the spawn context pipeline does
	sections, err := extractModelSectionsForSpawn(modelPath)
	if err != nil {
		t.Fatal(err)
	}

	// Critical Invariants section should contain evidence quality annotations
	if sections.criticalInvariants == "" {
		t.Fatal("expected non-empty Critical Invariants section")
	}

	// Verify evidence quality annotations are preserved
	checks := []struct {
		label string
		want  string
	}{
		{"replicated annotation", "Replicated"},
		{"single-source annotation", "Single-source measured"},
		{"replication failure caveat", "replication failed"},
		{"hypothesized warning", "HYPOTHESIZED"},
		{"do not cite warning", "do not cite as established"},
	}

	for _, c := range checks {
		if !strings.Contains(sections.criticalInvariants, c.want) {
			t.Errorf("Critical Invariants section missing %s (%q):\n%s", c.label, c.want, sections.criticalInvariants)
		}
	}

	// Also verify Summary section exists
	if sections.summary == "" {
		t.Error("expected non-empty Summary section")
	}

	// Verify Why This Fails section exists
	if sections.whyThisFails == "" {
		t.Error("expected non-empty Why This Fails section")
	}
}

// TestEvidenceQuality_FormattedInSpawnMatch verifies that the full model
// match formatting (as it appears in SPAWN_CONTEXT.md) preserves evidence
// quality annotations through the formatting pipeline.
func TestEvidenceQuality_FormattedInSpawnMatch(t *testing.T) {
	dir := t.TempDir()
	modelDir := filepath.Join(dir, ".kb", "models", "test-model")
	os.MkdirAll(filepath.Join(modelDir, "probes"), 0755)

	modelContent := `# Model: Test Model

## Critical Invariants

1. **Gate-enforced constraints have near-100% compliance.**

**Evidence quality:** Replicated (3 independent gates measured).

2. **Behavioral dilution budget is ~4 constraints.**

**Evidence quality:** Single-source measured (N=3, replication failed). HYPOTHESIZED.
`
	modelPath := filepath.Join(modelDir, "model.md")
	os.WriteFile(modelPath, []byte(modelContent), 0644)

	match := KBContextMatch{
		Type:  "model",
		Title: "Test Model",
		Path:  modelPath,
	}

	output, _ := formatModelMatchForSpawn(match, dir, nil)

	// Evidence quality annotations should survive the formatting pipeline
	if !strings.Contains(output, "Replicated") {
		t.Errorf("formatted output missing 'Replicated' annotation:\n%s", output)
	}
	if !strings.Contains(output, "Single-source measured") {
		t.Errorf("formatted output missing 'Single-source measured' annotation:\n%s", output)
	}
	if !strings.Contains(output, "HYPOTHESIZED") {
		t.Errorf("formatted output missing 'HYPOTHESIZED' caveat:\n%s", output)
	}
	if !strings.Contains(output, "replication failed") {
		t.Errorf("formatted output missing 'replication failed' caveat:\n%s", output)
	}
}

// TestEvidenceQuality_FullKBContextFormat verifies the end-to-end formatting
// from KBContextResult through FormatContextForSpawn preserves evidence quality.
func TestEvidenceQuality_FullKBContextFormat(t *testing.T) {
	dir := t.TempDir()
	modelDir := filepath.Join(dir, ".kb", "models", "confidence-test")
	os.MkdirAll(filepath.Join(modelDir, "probes"), 0755)

	modelContent := `# Model: Confidence Test

## Summary

Tests that evidence quality propagates through kb context.

## Critical Invariants

1. **Claim with high confidence.**

**Evidence quality:** Replicated (multiple sources).

2. **Claim with low confidence.**

**Evidence quality:** Assumed (no direct measurement).
`
	modelPath := filepath.Join(modelDir, "model.md")
	os.WriteFile(modelPath, []byte(modelContent), 0644)

	result := &KBContextResult{
		Query:      "confidence propagation",
		HasMatches: true,
		Matches: []KBContextMatch{
			{
				Type:  "model",
				Title: "Confidence Test",
				Path:  modelPath,
			},
		},
	}

	formatResult := FormatContextForSpawnWithLimitAndMeta(result, MaxKBContextChars, dir, nil)
	content := formatResult.Content

	if content == "" {
		t.Fatal("expected non-empty formatted context")
	}

	// Evidence quality annotations should appear in the final output
	if !strings.Contains(content, "Replicated") {
		t.Errorf("formatted context missing 'Replicated':\n%s", content)
	}
	if !strings.Contains(content, "Assumed") {
		t.Errorf("formatted context missing 'Assumed':\n%s", content)
	}
}
