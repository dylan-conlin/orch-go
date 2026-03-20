package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/claims"
)

func TestExtractClaimIDFromLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels []string
		want   string
	}{
		{"no labels", nil, ""},
		{"no claim label", []string{"triage:ready", "daemon:claim-probe"}, ""},
		{"claim label present", []string{"daemon:claim-probe", "claim:ATE-01", "triage:ready"}, "ATE-01"},
		{"multiple claim labels returns first", []string{"claim:ATE-01", "claim:ATE-02"}, "ATE-01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractClaimIDFromLabels(tt.labels)
			if got != tt.want {
				t.Errorf("ExtractClaimIDFromLabels() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLookupClaimContext(t *testing.T) {
	// Create temp directory structure with claims.yaml
	tmpDir := t.TempDir()
	modelsDir := filepath.Join(tmpDir, ".kb", "models", "test-model")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		t.Fatal(err)
	}

	claimsYAML := `model: test-model
version: 1
claims:
  - id: TM-01
    text: "Convention-layer constraints are insufficient"
    type: mechanism
    scope: bounded
    confidence: unconfirmed
    priority: core
    evidence:
      - source: "Prior probe found bypass via curl"
        date: "2026-03-06"
        verdict: confirms
    falsifies_if: "A convention that reliably prevents bypass"
  - id: TM-02
    text: "OS-level enforcement cannot be bypassed"
    type: mechanism
    scope: bounded
    confidence: confirmed
    priority: core
    falsifies_if: "Agent bypasses sandbox without root"
`
	if err := os.WriteFile(filepath.Join(modelsDir, "claims.yaml"), []byte(claimsYAML), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("finds claim by ID", func(t *testing.T) {
		cc := LookupClaimContext("TM-01", tmpDir)
		if cc == nil {
			t.Fatal("expected non-nil ClaimContext")
		}
		if cc.ClaimID != "TM-01" {
			t.Errorf("ClaimID = %q, want %q", cc.ClaimID, "TM-01")
		}
		if cc.ModelName != "test-model" {
			t.Errorf("ModelName = %q, want %q", cc.ModelName, "test-model")
		}
		if cc.FalsifiesIf != "A convention that reliably prevents bypass" {
			t.Errorf("FalsifiesIf = %q", cc.FalsifiesIf)
		}
		if len(cc.Evidence) != 1 {
			t.Errorf("Evidence count = %d, want 1", len(cc.Evidence))
		}
	})

	t.Run("returns nil for unknown claim", func(t *testing.T) {
		cc := LookupClaimContext("NOPE-99", tmpDir)
		if cc != nil {
			t.Errorf("expected nil, got %+v", cc)
		}
	})

	t.Run("returns nil for empty inputs", func(t *testing.T) {
		if cc := LookupClaimContext("", tmpDir); cc != nil {
			t.Error("expected nil for empty claimID")
		}
		if cc := LookupClaimContext("TM-01", ""); cc != nil {
			t.Error("expected nil for empty projectDir")
		}
	})
}

func TestFormatClaimContext(t *testing.T) {
	t.Run("nil returns empty", func(t *testing.T) {
		if got := FormatClaimContext(nil); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("formats all fields", func(t *testing.T) {
		cc := &ClaimContext{
			ClaimID:     "ATE-01",
			ModelName:   "agent-trust-enforcement",
			ClaimText:   "Convention-layer constraints are insufficient",
			FalsifiesIf: "A convention that reliably prevents bypass",
			Evidence: []claims.Evidence{
				{Source: "Probe found bypass via curl", Date: "2026-03-06", Verdict: "confirms"},
			},
			ClaimsFile: ".kb/models/agent-trust-enforcement/claims.yaml",
		}
		got := FormatClaimContext(cc)

		checks := []string{
			"## CLAIM PROBE CONTEXT",
			"**Claim ID:** ATE-01",
			"**Model:** agent-trust-enforcement",
			"**Claim:** Convention-layer constraints are insufficient",
			"**Falsifies if:** A convention that reliably prevents bypass",
			"Existing evidence",
			"[confirms] Probe found bypass via curl (2026-03-06)",
			"Evidence independence constraint",
		}
		for _, check := range checks {
			if !strings.Contains(got, check) {
				t.Errorf("output missing %q", check)
			}
		}
	})

	t.Run("no evidence section when empty", func(t *testing.T) {
		cc := &ClaimContext{
			ClaimID:     "ATE-01",
			ModelName:   "test",
			ClaimText:   "text",
			FalsifiesIf: "falsifies",
			ClaimsFile:  "path",
		}
		got := FormatClaimContext(cc)
		if strings.Contains(got, "Existing evidence") {
			t.Error("should not contain evidence section when no evidence")
		}
	})
}

func TestClaimContextIntegrationInTemplate(t *testing.T) {
	// Verify that claim context flows through GenerateContext when set
	cc := &ClaimContext{
		ClaimID:     "ATE-01",
		ModelName:   "agent-trust-enforcement",
		ClaimText:   "Convention-layer constraints are insufficient",
		FalsifiesIf: "A convention that reliably prevents bypass",
		Evidence: []claims.Evidence{
			{Source: "Probe found bypass", Date: "2026-03-06", Verdict: "confirms"},
		},
		ClaimsFile: ".kb/models/agent-trust-enforcement/claims.yaml",
	}

	cfg := &Config{
		Task:         "Probe claim ATE-01",
		SkillName:    "investigation",
		ProjectDir:   "/tmp/test",
		BeadsID:      "orch-go-test1",
		ClaimContext: FormatClaimContext(cc),
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(content, "## CLAIM PROBE CONTEXT") {
		t.Error("generated context missing CLAIM PROBE CONTEXT section")
	}
	if !strings.Contains(content, "ATE-01") {
		t.Error("generated context missing claim ID")
	}
	if !strings.Contains(content, "Evidence independence constraint") {
		t.Error("generated context missing evidence independence constraint")
	}
}
