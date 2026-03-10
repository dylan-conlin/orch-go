package kbgate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanNoveltyLanguage_FindsHits(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	pubDir := filepath.Join(kbDir, "publications")
	modDir := filepath.Join(kbDir, "models", "test-model")
	os.MkdirAll(pubDir, 0755)
	os.MkdirAll(modDir, 0755)

	// Publication with novelty language
	os.WriteFile(filepath.Join(pubDir, "draft.md"), []byte(`# My Draft

This is a novel framework for understanding coordination.
We discovered this pattern through empirical observation.
The dynamics are substrate-independent and follow a physics of knowledge.
This is absent from published literature.
This is the first systematic treatment of the problem.
We propose a new discipline for studying these effects.
`), 0644)

	// Model with novelty language
	os.WriteFile(filepath.Join(modDir, "model.md"), []byte(`# Test Model

## Summary
A new framework for agent coordination discovered through observation.
`), 0644)

	hits := ScanNoveltyLanguage(kbDir)

	// Should find hits in both files
	if len(hits) == 0 {
		t.Fatal("expected novelty language hits, got none")
	}

	// Check specific terms are caught
	terms := map[string]bool{
		"novel":                false,
		"discovered":           false,
		"substrate-independent": false,
		"physics":              false,
		"absent from":          false,
		"first":                false,
		"new discipline":       false,
		"new framework":        false,
	}
	for _, h := range hits {
		for term := range terms {
			if strings.Contains(strings.ToLower(h.Match), term) {
				terms[term] = true
			}
		}
	}
	for term, found := range terms {
		if !found {
			t.Errorf("expected to find term %q in hits", term)
		}
	}
}

func TestScanNoveltyLanguage_NoFalsePositives(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	pubDir := filepath.Join(kbDir, "publications")
	os.MkdirAll(pubDir, 0755)

	// Clean content — no novelty claims
	os.WriteFile(filepath.Join(pubDir, "clean.md"), []byte(`# Working Model

This is a working model describing coordination patterns we observed.
The patterns appear in our system under specific conditions.
`), 0644)

	hits := ScanNoveltyLanguage(kbDir)
	if len(hits) > 0 {
		t.Errorf("expected no hits for clean content, got %d: %v", len(hits), hits)
	}
}

func TestScanNoveltyLanguage_PhysicsInFilename(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	modDir := filepath.Join(kbDir, "models", "knowledge-physics")
	os.MkdirAll(modDir, 0755)

	// "physics" appears in filename path but not as a claim in body
	os.WriteFile(filepath.Join(modDir, "model.md"), []byte(`# Knowledge Observations

## Summary
We observed file growth patterns across the system.
`), 0644)

	hits := ScanNoveltyLanguage(kbDir)
	if len(hits) > 0 {
		t.Errorf("expected no hits when physics only in filename, got %d", len(hits))
	}
}

func TestScanProbeConclusions_FlagsSelfValidation(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	probeDir := filepath.Join(kbDir, "models", "test-model", "probes")
	os.MkdirAll(probeDir, 0755)

	os.WriteFile(filepath.Join(probeDir, "probe.md"), []byte(`# Probe: Test

## Findings
Some findings here.

## Model Impact

- **Confirms** invariant: workspace name is kebab-case
- **Extends** the model with new failure mode: config drift
- **Confirms** the state vs infrastructure distinction
`), 0644)

	hits := ScanProbeConclusions(kbDir)

	if len(hits) == 0 {
		t.Fatal("expected probe conclusion hits, got none")
	}

	// All three should be flagged (no external citations)
	if len(hits) != 3 {
		t.Errorf("expected 3 hits, got %d", len(hits))
	}

	for _, h := range hits {
		if h.Code != "SELF_VALIDATING_PROBE" {
			t.Errorf("expected code SELF_VALIDATING_PROBE, got %s", h.Code)
		}
	}
}

func TestScanProbeConclusions_ExternalCitationOK(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	probeDir := filepath.Join(kbDir, "models", "test-model", "probes")
	os.MkdirAll(probeDir, 0755)

	os.WriteFile(filepath.Join(probeDir, "probe.md"), []byte(`# Probe: Test

## Model Impact

- **Confirms** the pain-as-signal principle. Bainbridge (1983) found that automation hiding problems creates worse outcomes. See https://example.com/bainbridge1983
`), 0644)

	hits := ScanProbeConclusions(kbDir)

	if len(hits) != 0 {
		t.Errorf("expected 0 hits when external citation present, got %d: %v", len(hits), hits)
	}
}

func TestScanProbeConclusions_OnlyInModelImpact(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	probeDir := filepath.Join(kbDir, "models", "test-model", "probes")
	os.MkdirAll(probeDir, 0755)

	// "confirms" outside Model Impact section should not be flagged
	os.WriteFile(filepath.Join(probeDir, "probe.md"), []byte(`# Probe: Test

## Background
The user confirms the task is ready.

## Model Impact
No model changes needed.
`), 0644)

	hits := ScanProbeConclusions(kbDir)

	if len(hits) != 0 {
		t.Errorf("expected 0 hits for confirms outside Model Impact, got %d", len(hits))
	}
}

func TestScanCausalLanguage_FindsHits(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	modDir := filepath.Join(kbDir, "models", "test-model")
	os.MkdirAll(modDir, 0755)

	os.WriteFile(filepath.Join(modDir, "model.md"), []byte(`# Test Model

## Summary
Agent throughput causes convention decay. This will always produce entropy.
Knowledge contributions determine system health. The model guarantees that
coordination costs never decrease under amnesiac agents. File growth ensures
degradation. We can predict the onset of entropy spirals.

## Mechanism
Implementation details here.
`), 0644)

	hits := ScanCausalLanguage(kbDir)

	if len(hits) == 0 {
		t.Fatal("expected causal language hits, got none")
	}

	// Should find: causes, always, determine, guarantees, never, ensures, predict
	terms := map[string]bool{
		"cause":     false,
		"always":    false,
		"determine": false,
		"guarantee": false,
		"never":     false,
		"ensure":    false,
		"predict":   false,
	}
	for _, h := range hits {
		for term := range terms {
			if strings.Contains(strings.ToLower(h.Match), term) {
				terms[term] = true
			}
		}
	}
	for term, found := range terms {
		if !found {
			t.Errorf("expected to find causal term %q", term)
		}
	}
}

func TestScanCausalLanguage_OnlySummarySection(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	modDir := filepath.Join(kbDir, "models", "test-model")
	os.MkdirAll(modDir, 0755)

	os.WriteFile(filepath.Join(modDir, "model.md"), []byte(`# Test Model

## Summary
This is a clean summary with no causal claims.

## Mechanism
This function always ensures the value is never null. It determines the output
and guarantees correctness. It can predict the next state and cause side effects.
`), 0644)

	hits := ScanCausalLanguage(kbDir)

	if len(hits) != 0 {
		t.Errorf("expected 0 hits for causal language outside Summary, got %d", len(hits))
	}
}

func TestScanCausalLanguage_NoFalsePositives(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	modDir := filepath.Join(kbDir, "models", "test-model")
	os.MkdirAll(modDir, 0755)

	os.WriteFile(filepath.Join(modDir, "model.md"), []byte(`# Test Model

## Summary
We observed coordination patterns that correlate with file growth.
The patterns suggest a relationship between throughput and degradation.
`), 0644)

	hits := ScanCausalLanguage(kbDir)

	if len(hits) != 0 {
		t.Errorf("expected no hits for hedged language, got %d", len(hits))
	}
}

func TestClaimHit_Format(t *testing.T) {
	h := ClaimHit{
		File:  "test.md",
		Line:  42,
		Match: "novel framework",
		Code:  "NOVELTY_LANGUAGE",
	}
	s := h.String()
	if !strings.Contains(s, "test.md:42") {
		t.Errorf("expected file:line format, got %s", s)
	}
	if !strings.Contains(s, "novel framework") {
		t.Errorf("expected match text, got %s", s)
	}
}

func TestScanAllClaims_AggregatesResults(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	pubDir := filepath.Join(kbDir, "publications")
	modDir := filepath.Join(kbDir, "models", "test-model")
	probeDir := filepath.Join(modDir, "probes")
	os.MkdirAll(pubDir, 0755)
	os.MkdirAll(probeDir, 0755)

	os.WriteFile(filepath.Join(pubDir, "draft.md"), []byte(`# Draft
This is a novel finding.
`), 0644)

	os.WriteFile(filepath.Join(modDir, "model.md"), []byte(`# Model
## Summary
This always causes degradation.
`), 0644)

	os.WriteFile(filepath.Join(probeDir, "probe.md"), []byte(`# Probe
## Model Impact
- **Confirms** the model claim.
`), 0644)

	result := ScanAllClaims(kbDir)

	if len(result.Novelty) == 0 {
		t.Error("expected novelty hits")
	}
	if len(result.ProbeConclusions) == 0 {
		t.Error("expected probe conclusion hits")
	}
	if len(result.CausalLanguage) == 0 {
		t.Error("expected causal language hits")
	}
	if result.Total() == 0 {
		t.Error("expected non-zero total")
	}
}

func TestScanAgainstCorpus(t *testing.T) {
	// Test against actual .kb/ if it exists
	kbDir := filepath.Join("..", "..", ".kb")
	if _, err := os.Stat(kbDir); os.IsNotExist(err) {
		t.Skip("no .kb/ directory found — skipping corpus test")
	}

	result := ScanAllClaims(kbDir)

	// Based on prior scan: 12+ novelty hits in publications, massive confirms/extends in probes
	if result.Total() < 10 {
		t.Errorf("expected at least 10 total hits against real corpus, got %d", result.Total())
	}

	// Publications should have novelty hits
	if len(result.Novelty) < 5 {
		t.Errorf("expected at least 5 novelty hits, got %d", len(result.Novelty))
	}

	// Probes should have self-validating patterns
	if len(result.ProbeConclusions) < 5 {
		t.Errorf("expected at least 5 probe conclusion hits, got %d", len(result.ProbeConclusions))
	}
}
