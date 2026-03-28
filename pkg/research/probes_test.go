package research

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseClaimRefs(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"NI-01", []string{"NI-01"}},
		{"NI-01, NI-03", []string{"NI-01", "NI-03"}},
		{"CA-01, CA-02, CA-03, CA-04", []string{"CA-01", "CA-02", "CA-03", "CA-04"}},
		{"n/a", nil},
		{"extends (no prior claim — new capability area)", nil},
		{"n/a (system-level probe, no single claim ID)", nil},
		{"CI-03 (Open Question: scoped vs global)", []string{"CI-03"}},
		{"COORD-04", []string{"COORD-04"}},
		{"DAO-15", []string{"DAO-15"}},
		{"(implicit — \"Align is the meta-primitive\")", nil},
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseClaimRefs(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseClaimRefs(%q) = %v, want %v", tt.input, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseClaimRefs(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIsClaimID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"NI-01", true},
		{"CA-06", true},
		{"DAO-15", true},
		{"COORD-04", true},
		{"KA-01", true},
		{"SG-01", true},
		{"n/a", false},
		{"", false},
		{"extends", false},
		{"01", false},
		{"-01", false},
		{"NI-", false},
		{"NI01", false},
		{"123-ABC", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isClaimID(tt.input)
			if got != tt.want {
				t.Errorf("isClaimID(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeVerdict(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"confirms", "confirms"},
		{"contradicts", "contradicts"},
		{"extends", "extends"},
		{"confirms (with extensions)", "confirms"},
		{"disconfirms (with extension)", "disconfirms"},
		{"scopes", "scopes"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeVerdict(tt.input)
			if got != tt.want {
				t.Errorf("normalizeVerdict(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestScanProbes(t *testing.T) {
	// Create temp dir with probe files
	dir := t.TempDir()

	probe1 := `# Probe: Test Clustering in Corpus

**Model:** named-incompleteness
**Date:** 2026-03-28
**Status:** Complete
**claim:** NI-01
**verdict:** confirms

---

## Question
Does NI-01 hold?
`
	probe2 := `# Probe: Multi-Claim Probe

**Model:** named-incompleteness
**Date:** 2026-03-27
**Status:** Complete
**claim:** NI-01, NI-03
**verdict:** extends

---

## Question
Testing multiple claims.
`
	probe3 := `# Probe: No Claim Reference

**Model:** some-model
**Date:** 2026-03-26
**Status:** Active
**claim:** n/a
**verdict:** extends

---
`

	writeFile(t, dir, "2026-03-28-probe-test-clustering.md", probe1)
	writeFile(t, dir, "2026-03-27-probe-multi-claim.md", probe2)
	writeFile(t, dir, "2026-03-26-probe-no-claim.md", probe3)

	results, err := ScanProbes(dir)
	if err != nil {
		t.Fatalf("ScanProbes error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 probes, got %d", len(results))
	}

	// Find probe1 by date
	var p1, p2, p3 *ProbeResult
	for i := range results {
		switch results[i].Date {
		case "2026-03-28":
			p1 = &results[i]
		case "2026-03-27":
			p2 = &results[i]
		case "2026-03-26":
			p3 = &results[i]
		}
	}

	if p1 == nil || p2 == nil || p3 == nil {
		t.Fatal("could not find all probes by date")
	}

	// Probe 1: single claim
	if len(p1.Claims) != 1 || p1.Claims[0] != "NI-01" {
		t.Errorf("probe1 claims = %v, want [NI-01]", p1.Claims)
	}
	if p1.Verdict != "confirms" {
		t.Errorf("probe1 verdict = %q, want confirms", p1.Verdict)
	}
	if p1.Title != "Test Clustering in Corpus" {
		t.Errorf("probe1 title = %q, want 'Test Clustering in Corpus'", p1.Title)
	}

	// Probe 2: multiple claims
	if len(p2.Claims) != 2 {
		t.Errorf("probe2 claims = %v, want [NI-01 NI-03]", p2.Claims)
	}

	// Probe 3: no claims
	if len(p3.Claims) != 0 {
		t.Errorf("probe3 claims = %v, want empty", p3.Claims)
	}
}

func TestScanProbes_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	results, err := ScanProbes(dir)
	if err != nil {
		t.Fatalf("ScanProbes error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 probes, got %d", len(results))
	}
}

func TestScanProbes_NonexistentDir(t *testing.T) {
	results, err := ScanProbes("/nonexistent/path")
	if err != nil {
		t.Fatalf("ScanProbes should return nil error for nonexistent dir, got: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results, got %v", results)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	if err != nil {
		t.Fatalf("writeFile %s: %v", name, err)
	}
}
