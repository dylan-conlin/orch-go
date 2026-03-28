package research

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadModelStatus_ClaimsYAML(t *testing.T) {
	dir := t.TempDir()

	// Write claims.yaml
	claimsYAML := `model: test-model
version: 1
claims:
  - id: TM-01
    text: "First claim"
    confidence: confirmed
    priority: core
  - id: TM-02
    text: "Second claim"
    confidence: unconfirmed
    priority: supporting
`
	if err := os.WriteFile(filepath.Join(dir, "claims.yaml"), []byte(claimsYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Write a probe that references TM-01
	probesDir := filepath.Join(dir, "probes")
	if err := os.MkdirAll(probesDir, 0755); err != nil {
		t.Fatal(err)
	}

	probe := `# Probe: Test TM-01

**Model:** test-model
**Date:** 2026-03-28
**Status:** Complete
**claim:** TM-01
**verdict:** confirms
`
	if err := os.WriteFile(filepath.Join(probesDir, "2026-03-28-probe-test.md"), []byte(probe), 0644); err != nil {
		t.Fatal(err)
	}

	ms, err := LoadModelStatus(dir)
	if err != nil {
		t.Fatalf("LoadModelStatus error: %v", err)
	}
	if ms == nil {
		t.Fatal("expected non-nil ModelStatus")
	}

	if ms.TotalClaims != 2 {
		t.Errorf("TotalClaims = %d, want 2", ms.TotalClaims)
	}
	// TM-01 has a probe -> confirmed; TM-02 has no probe + unconfirmed -> untested
	if ms.TestedClaims != 1 {
		t.Errorf("TestedClaims = %d, want 1", ms.TestedClaims)
	}

	// Check TM-01 status
	tm01 := FindClaim(ms, "TM-01")
	if tm01 == nil {
		t.Fatal("TM-01 not found")
	}
	if tm01.TestStatus != StatusConfirmed {
		t.Errorf("TM-01 TestStatus = %s, want confirmed", tm01.TestStatus)
	}
	if len(tm01.Probes) != 1 {
		t.Errorf("TM-01 has %d probes, want 1", len(tm01.Probes))
	}

	// Check TM-02 status
	tm02 := FindClaim(ms, "TM-02")
	if tm02 == nil {
		t.Fatal("TM-02 not found")
	}
	if tm02.TestStatus != StatusUntested {
		t.Errorf("TM-02 TestStatus = %s, want untested", tm02.TestStatus)
	}
}

func TestLoadModelStatus_MarkdownOnly(t *testing.T) {
	dir := t.TempDir()

	modelMd := `# Model: Test

## Claims (Testable)

| ID | Claim | How to Verify |
|----|-------|---------------|
| X-01 | First claim | Test it |
| X-02 | Second claim | Test it too |

## References
`
	if err := os.WriteFile(filepath.Join(dir, "model.md"), []byte(modelMd), 0644); err != nil {
		t.Fatal(err)
	}

	ms, err := LoadModelStatus(dir)
	if err != nil {
		t.Fatalf("LoadModelStatus error: %v", err)
	}
	if ms == nil {
		t.Fatal("expected non-nil ModelStatus")
	}

	if ms.Source != "model.md" {
		t.Errorf("Source = %q, want model.md", ms.Source)
	}
	if ms.TotalClaims != 2 {
		t.Errorf("TotalClaims = %d, want 2", ms.TotalClaims)
	}
}

func TestLoadModelStatus_NoClaims(t *testing.T) {
	dir := t.TempDir()
	// Empty dir — no claims.yaml, no model.md
	ms, err := LoadModelStatus(dir)
	if err != nil {
		t.Fatalf("LoadModelStatus error: %v", err)
	}
	if ms != nil {
		t.Errorf("expected nil for dir with no claims, got %+v", ms)
	}
}

func TestLoadAllModels(t *testing.T) {
	kbDir := t.TempDir()
	modelsDir := filepath.Join(kbDir, "models")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Model with claims.yaml
	m1Dir := filepath.Join(modelsDir, "model-one")
	if err := os.MkdirAll(m1Dir, 0755); err != nil {
		t.Fatal(err)
	}
	claimsYAML := `model: model-one
version: 1
claims:
  - id: M1-01
    text: "Claim one"
    confidence: confirmed
    priority: core
`
	if err := os.WriteFile(filepath.Join(m1Dir, "claims.yaml"), []byte(claimsYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Model without claims
	m2Dir := filepath.Join(modelsDir, "model-two")
	if err := os.MkdirAll(m2Dir, 0755); err != nil {
		t.Fatal(err)
	}

	results, err := LoadAllModels(kbDir)
	if err != nil {
		t.Fatalf("LoadAllModels error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 model with claims, got %d", len(results))
	}
	if len(results) > 0 && results[0].ModelName != "model-one" {
		t.Errorf("model name = %q, want model-one", results[0].ModelName)
	}
}

func TestFindModel(t *testing.T) {
	kbDir := t.TempDir()
	modelsDir := filepath.Join(kbDir, "models")
	for _, name := range []string{"named-incompleteness", "knowledge-accretion", "named-other"} {
		if err := os.MkdirAll(filepath.Join(modelsDir, name), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Exact match
	path, err := FindModel(kbDir, "knowledge-accretion")
	if err != nil {
		t.Fatalf("exact match error: %v", err)
	}
	if filepath.Base(path) != "knowledge-accretion" {
		t.Errorf("exact match = %q, want knowledge-accretion", filepath.Base(path))
	}

	// Prefix match
	path, err = FindModel(kbDir, "knowledge")
	if err != nil {
		t.Fatalf("prefix match error: %v", err)
	}
	if filepath.Base(path) != "knowledge-accretion" {
		t.Errorf("prefix match = %q, want knowledge-accretion", filepath.Base(path))
	}

	// Ambiguous prefix
	_, err = FindModel(kbDir, "named")
	if err == nil {
		t.Error("expected error for ambiguous prefix")
	}

	// Not found
	_, err = FindModel(kbDir, "nonexistent")
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestDeriveTestStatus(t *testing.T) {
	tests := []struct {
		name       string
		confidence string
		probes     []ProbeResult
		want       TestStatus
	}{
		{
			name:       "no probes, unconfirmed",
			confidence: "",
			probes:     nil,
			want:       StatusUntested,
		},
		{
			name:       "no probes, confirmed in yaml",
			confidence: "confirmed",
			probes:     nil,
			want:       StatusConfirmed,
		},
		{
			name: "single confirms probe",
			probes: []ProbeResult{
				{Verdict: "confirms"},
			},
			want: StatusConfirmed,
		},
		{
			name: "single contradicts probe",
			probes: []ProbeResult{
				{Verdict: "contradicts"},
			},
			want: StatusContradicted,
		},
		{
			name: "single extends probe",
			probes: []ProbeResult{
				{Verdict: "extends"},
			},
			want: StatusExtended,
		},
		{
			name: "mixed verdicts",
			probes: []ProbeResult{
				{Verdict: "confirms"},
				{Verdict: "disconfirms"},
			},
			want: StatusMixed,
		},
		{
			name: "probes with no verdict",
			probes: []ProbeResult{
				{Verdict: ""},
			},
			want: StatusUntested,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := ClaimStatus{
				Confidence: tt.confidence,
				Probes:     tt.probes,
			}
			got := deriveTestStatus(cs)
			if got != tt.want {
				t.Errorf("deriveTestStatus() = %s, want %s", got, tt.want)
			}
		})
	}
}
