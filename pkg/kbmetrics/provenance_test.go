package kbmetrics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAuditProvenance_FullCoverage(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	modelDir := filepath.Join(kbDir, "models", "test-model")
	os.MkdirAll(filepath.Join(modelDir, "probes"), 0755)

	// Model with all claims annotated
	modelContent := `# Model: Test

## Core Claim

Knowledge accretion is real.

**Evidence quality:** Replicated (3 independent sources).

## Critical Invariants

1. **Every convention without a gate will eventually be violated.** The knowledge system proves this.

**Evidence quality:** Multi-source analytical (2 investigations).

2. **Models are the fundamental unit.** Without models, knowledge is homeless.

**Evidence quality:** Single-source measured.
`
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(modelContent), 0644)

	reports, err := AuditProvenance(kbDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}

	r := reports[0]
	if r.Name != "test-model" {
		t.Errorf("Name = %q, want %q", r.Name, "test-model")
	}
	if r.TotalClaims != 3 {
		t.Errorf("TotalClaims = %d, want 3", r.TotalClaims)
	}
	if r.AnnotatedClaims != 3 {
		t.Errorf("AnnotatedClaims = %d, want 3", r.AnnotatedClaims)
	}
	if r.CoveragePercent != 100.0 {
		t.Errorf("CoveragePercent = %.1f, want 100.0", r.CoveragePercent)
	}
	if len(r.UnannotatedClaims) != 0 {
		t.Errorf("UnannotatedClaims = %d, want 0", len(r.UnannotatedClaims))
	}
}

func TestAuditProvenance_Unannotated(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	modelDir := filepath.Join(kbDir, "models", "test-model")
	os.MkdirAll(filepath.Join(modelDir, "probes"), 0755)

	// Model with some claims missing annotations
	modelContent := `# Model: Test

## Core Claim

Knowledge accretion is real.

## Critical Invariants

1. **First invariant.** Details here.

**Evidence quality:** Replicated.

2. **Second invariant.** More details.
`
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(modelContent), 0644)

	reports, err := AuditProvenance(kbDir)
	if err != nil {
		t.Fatal(err)
	}

	r := reports[0]
	if r.TotalClaims != 3 {
		t.Errorf("TotalClaims = %d, want 3", r.TotalClaims)
	}
	if r.AnnotatedClaims != 1 {
		t.Errorf("AnnotatedClaims = %d, want 1", r.AnnotatedClaims)
	}
	if len(r.UnannotatedClaims) != 2 {
		t.Errorf("UnannotatedClaims = %d, want 2", len(r.UnannotatedClaims))
	}
}

func TestAuditProvenance_OrphanContradictions(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	modelDir := filepath.Join(kbDir, "models", "test-model")
	probeDir := filepath.Join(modelDir, "probes")
	os.MkdirAll(probeDir, 0755)

	// Model last updated 2026-02-01
	modelContent := `# Model: Test

**Last Updated:** 2026-02-01

## Core Claim

Something here.
`
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(modelContent), 0644)

	// Probe from 2026-02-15 with contradiction (after model update)
	probeContent := `# Probe: Contradiction Test

## Model Impact

- [x] **Contradicts** invariant: First invariant is wrong.
- [x] **Extends** model with: New findings.
`
	os.WriteFile(filepath.Join(probeDir, "2026-02-15-probe-test.md"), []byte(probeContent), 0644)

	reports, err := AuditProvenance(kbDir)
	if err != nil {
		t.Fatal(err)
	}

	r := reports[0]
	if len(r.OrphanContradictions) != 1 {
		t.Fatalf("OrphanContradictions = %d, want 1", len(r.OrphanContradictions))
	}
	oc := r.OrphanContradictions[0]
	if oc.ProbeDate != "2026-02-15" {
		t.Errorf("ProbeDate = %q, want %q", oc.ProbeDate, "2026-02-15")
	}
}

func TestAuditProvenance_NoOrphanWhenModelUpdatedAfterProbe(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	modelDir := filepath.Join(kbDir, "models", "test-model")
	probeDir := filepath.Join(modelDir, "probes")
	os.MkdirAll(probeDir, 0755)

	// Model updated AFTER the probe
	modelContent := `# Model: Test

**Last Updated:** 2026-03-01

## Core Claim

Something here.
`
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(modelContent), 0644)

	// Probe from before model update with contradiction
	probeContent := `# Probe: Old Contradiction

## Model Impact

- [x] **Contradicts** invariant: Something was wrong.
`
	os.WriteFile(filepath.Join(probeDir, "2026-02-15-probe-old.md"), []byte(probeContent), 0644)

	reports, err := AuditProvenance(kbDir)
	if err != nil {
		t.Fatal(err)
	}

	r := reports[0]
	if len(r.OrphanContradictions) != 0 {
		t.Errorf("OrphanContradictions = %d, want 0 (model was updated after probe)", len(r.OrphanContradictions))
	}
}

func TestAuditProvenance_LowConfidenceClaims(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	modelDir := filepath.Join(kbDir, "models", "test-model")
	os.MkdirAll(filepath.Join(modelDir, "probes"), 0755)

	modelContent := `# Model: Test

## Core Claim

Dilution budget is ~4 constraints.

**Evidence quality:** Single-source measured (N=3, replication failed).

## Critical Invariants

1. **Knowledge transfers reliably.** Confirmed.

**Evidence quality:** Replicated (4 independent sources).

2. **Emphasis helps at scale.** Partial.

**Evidence quality:** Assumed (no direct measurement).
`
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(modelContent), 0644)

	reports, err := AuditProvenance(kbDir)
	if err != nil {
		t.Fatal(err)
	}

	r := reports[0]
	if len(r.LowConfidenceClaims) != 2 {
		t.Errorf("LowConfidenceClaims = %d, want 2", len(r.LowConfidenceClaims))
	}
}

func TestAuditProvenance_EmptyModelsDir(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	os.MkdirAll(filepath.Join(kbDir, "models"), 0755)

	reports, err := AuditProvenance(kbDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 0 {
		t.Errorf("expected 0 reports, got %d", len(reports))
	}
}

func TestAuditProvenance_MultipleModels(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")

	// Model A: well-annotated
	modelADir := filepath.Join(kbDir, "models", "model-a")
	os.MkdirAll(filepath.Join(modelADir, "probes"), 0755)
	os.WriteFile(filepath.Join(modelADir, "model.md"), []byte(`# Model: A

## Core Claim

Claim A.

**Evidence quality:** Replicated.
`), 0644)

	// Model B: no annotations
	modelBDir := filepath.Join(kbDir, "models", "model-b")
	os.MkdirAll(filepath.Join(modelBDir, "probes"), 0755)
	os.WriteFile(filepath.Join(modelBDir, "model.md"), []byte(`# Model: B

## Core Claim

Claim B.

## Critical Invariants

1. **Invariant one.** Details.
2. **Invariant two.** Details.
`), 0644)

	reports, err := AuditProvenance(kbDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 2 {
		t.Fatalf("expected 2 reports, got %d", len(reports))
	}

	// Reports should be sorted: worst coverage first
	if reports[0].CoveragePercent >= reports[1].CoveragePercent {
		t.Errorf("expected worse coverage first: %.1f >= %.1f",
			reports[0].CoveragePercent, reports[1].CoveragePercent)
	}
}

func TestAuditProvenance_ClaimHeadings(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	modelDir := filepath.Join(kbDir, "models", "test-model")
	os.MkdirAll(filepath.Join(modelDir, "probes"), 0755)

	// Model using ### Claim N format (like orchestrator-skill)
	modelContent := `# Model: Test

## Core Claims

### Claim 1: Skills are probability-shaping documents

Paragraph of text explaining the claim.

**Evidence quality:** Multi-source analytical (2 investigations).

### Claim 2: Knowledge transfers reliably

Another paragraph.

**Evidence quality:** Single-source measured.

### Claim 3: No annotation here

Paragraph without evidence quality.
`
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(modelContent), 0644)

	reports, err := AuditProvenance(kbDir)
	if err != nil {
		t.Fatal(err)
	}

	r := reports[0]
	if r.TotalClaims != 3 {
		t.Errorf("TotalClaims = %d, want 3", r.TotalClaims)
	}
	if r.AnnotatedClaims != 2 {
		t.Errorf("AnnotatedClaims = %d, want 2", r.AnnotatedClaims)
	}
	if len(r.UnannotatedClaims) != 1 {
		t.Errorf("UnannotatedClaims = %d, want 1", len(r.UnannotatedClaims))
	}
	if len(r.LowConfidenceClaims) != 1 {
		t.Errorf("LowConfidenceClaims = %d, want 1 (single-source)", len(r.LowConfidenceClaims))
	}
}

func TestAuditProvenance_DriftDetection(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	modelDir := filepath.Join(kbDir, "models", "test-model")
	os.MkdirAll(filepath.Join(modelDir, "probes"), 0755)

	// Claim marked "observed" but uses overclaim language
	modelContent := `# Model: Test

## Core Claims

### Claim 1: Communication fundamentally cannot produce coordination

Messaging-based frameworks are fundamentally flawed in all cases.

**Evidence quality:** Observed (single experiment, N=10).

### Claim 2: Placement works in tested scenarios

In the tested same-file scenarios, structural placement prevented conflicts.

**Evidence quality:** Observed (single experiment, N=10).
`
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(modelContent), 0644)

	reports, err := AuditProvenance(kbDir)
	if err != nil {
		t.Fatal(err)
	}

	r := reports[0]
	// Claim 1 should trigger drift (fundamentally + all at observed tier)
	// Claim 2 should NOT trigger drift (scoped language)
	if len(r.DriftFlags) == 0 {
		t.Fatal("expected at least one drift flag for overclaimed language at observed tier")
	}
	if len(r.DriftFlags) > 1 {
		t.Errorf("expected 1 drift flag (only claim 1), got %d", len(r.DriftFlags))
	}
}

func TestFormatProvenanceText_Output(t *testing.T) {
	reports := []ProvenanceReport{
		{
			Name:             "test-model",
			TotalClaims:      6,
			AnnotatedClaims:  4,
			CoveragePercent:  66.7,
			UnannotatedClaims: []UnannotatedClaim{
				{Line: 10, Text: "Unannotated claim 1"},
				{Line: 25, Text: "Unannotated claim 2"},
			},
			LowConfidenceClaims: []LowConfidenceClaim{
				{Line: 15, Text: "Caveated claim", Level: "single-source"},
			},
			OrphanContradictions: []OrphanContradiction{
				{ProbePath: "probes/2026-02-15-test.md", ProbeDate: "2026-02-15", ContradictionText: "invariant was wrong"},
			},
		},
	}

	output := FormatProvenanceText(reports)
	if output == "" {
		t.Error("FormatProvenanceText returned empty string")
	}

	// Check key elements are present
	for _, want := range []string{
		"test-model",
		"66.7%",
		"4/6",
		"single-source",
		"2026-02-15",
	} {
		if !containsStr(output, want) {
			t.Errorf("output missing %q:\n%s", want, output)
		}
	}
}
