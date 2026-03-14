package kbmetrics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDilutionCurveCase_ProvenanceDetectsOrphanAndLowConfidence verifies that
// `kb audit provenance` detects the dilution curve scenario: a model with a
// claim cited as established, a probe that contradicts it (replication failed),
// and the model not being updated to reflect the caveat.
//
// This is the motivating case for confidence propagation (DC-5):
// - Probe on Mar 4 noted replication failure
// - Model last updated Feb 20 (before probe)
// - 4 downstream artifacts treated thresholds as established
// The audit should detect: (1) orphan contradiction, (2) low-confidence claim
func TestDilutionCurveCase_ProvenanceDetectsOrphanAndLowConfidence(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	modelDir := filepath.Join(kbDir, "models", "orchestrator-skill")
	probeDir := filepath.Join(modelDir, "probes")
	os.MkdirAll(probeDir, 0755)

	// Model last updated before the contradiction probe
	modelContent := `# Model: Orchestrator Skill

**Last Updated:** 2026-02-20

## Summary

The orchestrator skill shapes agent behavior through structured context.

## Core Claims

### Claim 1: Skills are probability-shaping documents

Skills shift behavioral distributions but cannot guarantee compliance.

**Evidence quality:** Multi-source analytical (2 investigations).

### Claim 2: Knowledge content transfers reliably

Agents absorb factual content from skill context at high rates.

**Evidence quality:** Replicated (4 independent sources).

### Claim 3: Emphasis language has no measurable effect

Adding emphasis (CRITICAL, MUST, bold) does not improve compliance.

**Evidence quality:** Multi-source analytical (2 investigations).

### Claim 4: Behavioral dilution budget is ~4 constraints

Adding more than 4 behavioral constraints causes overall compliance to drop.

**Evidence quality:** Single-source measured (N=3, replication failed Mar 4).

### Claim 5: Gate-enforced constraints have near-100% compliance

Infrastructure enforcement (gates, hooks) provides reliable constraint compliance.

**Evidence quality:** Replicated (accretion gate, probe merge gate, hotspot gate).

### Claim 6: Emphasis language helps at scale

At 8+ constraints, emphasis provides minor lift.

**Evidence quality:** Assumed (extrapolated from Claim 4 data, no direct measurement).
`
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(modelContent), 0644)

	// Probe from Mar 4 contradicting the dilution curve thresholds
	probeContent := `# Probe: Dilution Curve Replication Attempt

**Model:** orchestrator-skill

## Question

Can the dilution budget threshold (≤4 behavioral constraints) be replicated?

## Findings

Replication attempt failed. The N=3 sample size was too small and the
measurement methodology was not reproducible. Thresholds should be treated
as unreplicated hypotheses.

## Model Impact

- [x] **Contradicts** invariant: Claim 4 threshold values are unreplicated — treating as established is unsupported.
- [x] **Extends** model with: Evidence quality for Claim 4 must be downgraded to "single-source, replication failed."
`
	os.WriteFile(filepath.Join(probeDir, "2026-03-04-probe-dilution-replication.md"), []byte(probeContent), 0644)

	// Run the provenance audit
	reports, err := AuditProvenance(kbDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}

	r := reports[0]
	if r.Name != "orchestrator-skill" {
		t.Errorf("Name = %q, want %q", r.Name, "orchestrator-skill")
	}

	// Should detect 6 claims total
	if r.TotalClaims != 6 {
		t.Errorf("TotalClaims = %d, want 6", r.TotalClaims)
	}

	// All claims are annotated (100% coverage)
	if r.AnnotatedClaims != 6 {
		t.Errorf("AnnotatedClaims = %d, want 6", r.AnnotatedClaims)
	}

	// Should detect orphan contradiction (probe from 2026-03-04 > model updated 2026-02-20)
	if len(r.OrphanContradictions) == 0 {
		t.Fatal("expected orphan contradictions for dilution curve probe, got 0")
	}

	foundDilutionOrphan := false
	for _, oc := range r.OrphanContradictions {
		if oc.ProbeDate == "2026-03-04" {
			foundDilutionOrphan = true
			if !strings.Contains(oc.ContradictionText, "unreplicated") {
				t.Errorf("orphan contradiction text should mention 'unreplicated', got: %s", oc.ContradictionText)
			}
		}
	}
	if !foundDilutionOrphan {
		t.Error("did not find orphan contradiction from 2026-03-04 dilution replication probe")
	}

	// Should detect 2 low-confidence claims: Claim 4 (single-source) and Claim 6 (assumed)
	if len(r.LowConfidenceClaims) != 2 {
		t.Errorf("LowConfidenceClaims = %d, want 2 (Claim 4 single-source, Claim 6 assumed)", len(r.LowConfidenceClaims))
	}

	foundSingleSource := false
	foundAssumed := false
	for _, lc := range r.LowConfidenceClaims {
		if lc.Level == "single-source" {
			foundSingleSource = true
		}
		if lc.Level == "assumed" {
			foundAssumed = true
		}
	}
	if !foundSingleSource {
		t.Error("expected a single-source low-confidence claim (Claim 4)")
	}
	if !foundAssumed {
		t.Error("expected an assumed low-confidence claim (Claim 6)")
	}
}

// TestDilutionCurveCase_NoOrphanAfterModelUpdate verifies that updating
// the model after the contradiction probe resolves the orphan detection.
// This confirms the "fix path": merge probe findings into model → audit clears.
func TestDilutionCurveCase_NoOrphanAfterModelUpdate(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	modelDir := filepath.Join(kbDir, "models", "orchestrator-skill")
	probeDir := filepath.Join(modelDir, "probes")
	os.MkdirAll(probeDir, 0755)

	// Model updated AFTER the probe (Mar 12 > Mar 4)
	modelContent := `# Model: Orchestrator Skill

**Last Updated:** 2026-03-12

## Core Claims

### Claim 4: Behavioral dilution budget is ~4 constraints

Adding more than 4 behavioral constraints causes overall compliance to drop.

**Evidence quality:** Single-source measured (N=3, replication failed Mar 4). HYPOTHESIZED — do not cite as established.
`
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(modelContent), 0644)

	// Same probe from Mar 4
	probeContent := `# Probe: Dilution Curve Replication Attempt

## Model Impact

- [x] **Contradicts** invariant: Claim 4 threshold values are unreplicated.
`
	os.WriteFile(filepath.Join(probeDir, "2026-03-04-probe-dilution-replication.md"), []byte(probeContent), 0644)

	reports, err := AuditProvenance(kbDir)
	if err != nil {
		t.Fatal(err)
	}

	r := reports[0]
	if len(r.OrphanContradictions) != 0 {
		t.Errorf("expected 0 orphan contradictions after model update, got %d", len(r.OrphanContradictions))
	}
}

// TestProvenanceFormat_DilutionCurveOutput verifies that the text output
// for the dilution curve case includes all critical information an operator
// needs to assess the confidence gap.
func TestProvenanceFormat_DilutionCurveOutput(t *testing.T) {
	reports := []ProvenanceReport{
		{
			Name:            "orchestrator-skill",
			TotalClaims:     6,
			AnnotatedClaims: 6,
			CoveragePercent: 100.0,
			LowConfidenceClaims: []LowConfidenceClaim{
				{Line: 30, Text: "### Claim 4: Behavioral dilution budget is ~4 constraints", Level: "single-source"},
				{Line: 45, Text: "### Claim 6: Emphasis language helps at scale", Level: "assumed"},
			},
			OrphanContradictions: []OrphanContradiction{
				{
					ProbePath:         "probes/2026-03-04-probe-dilution-replication.md",
					ProbeDate:         "2026-03-04",
					ContradictionText: "Claim 4 threshold values are unreplicated",
				},
			},
		},
	}

	output := FormatProvenanceText(reports)

	// Verify all critical elements are surfaced
	checks := map[string]string{
		"model name":          "orchestrator-skill",
		"coverage percent":    "100.0%",
		"single-source level": "single-source",
		"assumed level":       "assumed",
		"orphan probe date":   "2026-03-04",
		"orphan contradiction": "unreplicated",
	}

	for label, want := range checks {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %s (%q):\n%s", label, want, output)
		}
	}
}
