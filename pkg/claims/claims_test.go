package claims

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseAndMarshal(t *testing.T) {
	input := `model: test-model
version: 1
last_audit: "2026-03-19"
claims:
  - id: TM-01
    text: "Test claim one"
    type: mechanism
    scope: local
    confidence: confirmed
    priority: core
    evidence:
      - source: "Test source"
        date: "2026-03-17"
        verdict: confirms
    last_validated: "2026-03-17"
    domain_tags: ["testing", "claims"]
    falsifies_if: "Testing never fails"
    tensions:
      - claim: MH-05
        model: measurement-honesty
        type: extends
        note: "Related tension"
    model_md_ref: "## Summary"
  - id: TM-02
    text: "Test claim two"
    type: observation
    scope: bounded
    confidence: unconfirmed
    priority: supporting
    domain_tags: ["testing"]
    falsifies_if: "Observations are wrong"
`

	f, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if f.Model != "test-model" {
		t.Errorf("Model = %q, want %q", f.Model, "test-model")
	}
	if f.Version != 1 {
		t.Errorf("Version = %d, want 1", f.Version)
	}
	if len(f.Claims) != 2 {
		t.Fatalf("Claims count = %d, want 2", len(f.Claims))
	}

	c1 := f.Claims[0]
	if c1.ID != "TM-01" {
		t.Errorf("Claim 0 ID = %q, want %q", c1.ID, "TM-01")
	}
	if c1.Type != TypeMechanism {
		t.Errorf("Claim 0 Type = %q, want %q", c1.Type, TypeMechanism)
	}
	if c1.Confidence != Confirmed {
		t.Errorf("Claim 0 Confidence = %q, want %q", c1.Confidence, Confirmed)
	}
	if len(c1.Evidence) != 1 {
		t.Fatalf("Claim 0 Evidence count = %d, want 1", len(c1.Evidence))
	}
	if c1.Evidence[0].Verdict != "confirms" {
		t.Errorf("Claim 0 Evidence verdict = %q, want %q", c1.Evidence[0].Verdict, "confirms")
	}
	if len(c1.Tensions) != 1 {
		t.Fatalf("Claim 0 Tensions count = %d, want 1", len(c1.Tensions))
	}
	if c1.Tensions[0].Claim != "MH-05" {
		t.Errorf("Tension claim = %q, want %q", c1.Tensions[0].Claim, "MH-05")
	}

	// Round-trip marshal
	data, err := Marshal(f)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	f2, err := Parse(data)
	if err != nil {
		t.Fatalf("Re-parse failed: %v", err)
	}
	if len(f2.Claims) != 2 {
		t.Errorf("Re-parsed claims count = %d, want 2", len(f2.Claims))
	}
	if f2.Claims[0].ID != "TM-01" {
		t.Errorf("Re-parsed claim 0 ID = %q, want %q", f2.Claims[0].ID, "TM-01")
	}
}

func TestClaimIsStale(t *testing.T) {
	now := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		lastValidated string
		wantStale     bool
	}{
		{"empty is stale", "", true},
		{"recent is not stale", "2026-03-10", false},
		{"old is stale", "2026-02-01", true},
		{"exactly 30 days is not stale", "2026-02-17", false},
		{"31 days is stale", "2026-02-16", true},
		{"invalid date is stale", "not-a-date", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Claim{LastValidated: tt.lastValidated}
			got := c.IsStale(now)
			if got != tt.wantStale {
				t.Errorf("IsStale() = %v, want %v", got, tt.wantStale)
			}
		})
	}
}

func TestClaimIsProbeEligible(t *testing.T) {
	now := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		claim      Claim
		wantResult bool
	}{
		{
			"unconfirmed core is eligible",
			Claim{Confidence: Unconfirmed, Priority: PriorityCore},
			true,
		},
		{
			"unconfirmed supporting is eligible",
			Claim{Confidence: Unconfirmed, Priority: PrioritySupporting},
			true,
		},
		{
			"unconfirmed peripheral is not eligible",
			Claim{Confidence: Unconfirmed, Priority: PriorityPeripheral},
			false,
		},
		{
			"confirmed recent core is not eligible",
			Claim{Confidence: Confirmed, Priority: PriorityCore, LastValidated: "2026-03-10"},
			false,
		},
		{
			"confirmed stale core is eligible",
			Claim{Confidence: Confirmed, Priority: PriorityCore, LastValidated: "2026-01-01"},
			true,
		},
		{
			"stale confidence is eligible",
			Claim{Confidence: Stale, Priority: PrioritySupporting},
			true,
		},
		{
			"contested is not eligible (needs orchestrator review)",
			Claim{Confidence: Contested, Priority: PriorityCore},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.claim.IsProbeEligible(now)
			if got != tt.wantResult {
				t.Errorf("IsProbeEligible() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}

func TestHasDomainOverlap(t *testing.T) {
	c := &Claim{DomainTags: []string{"gates", "enforcement", "accretion"}}

	if !c.HasDomainOverlap([]string{"gates", "spawning"}) {
		t.Error("expected overlap with 'gates'")
	}
	if c.HasDomainOverlap([]string{"testing", "spawning"}) {
		t.Error("expected no overlap with 'testing'/'spawning'")
	}
	if c.HasDomainOverlap(nil) {
		t.Error("expected no overlap with nil keywords")
	}
}

func TestCollectEdges(t *testing.T) {
	now := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)

	files := map[string]*File{
		"model-a": {
			Model: "model-a",
			Claims: []Claim{
				{
					ID:         "A-01",
					Text:       "Claim with tension",
					Confidence: Confirmed,
					Priority:   PriorityCore,
					Tensions: []Tension{
						{Claim: "B-01", Model: "model-b", Type: "contradicts", Note: "They disagree"},
					},
				},
				{
					ID:            "A-02",
					Text:          "Stale claim in active area",
					Confidence:    Confirmed,
					Priority:      PriorityCore,
					LastValidated: "2026-01-01",
					DomainTags:    []string{"gates"},
				},
			},
		},
		"model-b": {
			Model: "model-b",
			Claims: []Claim{
				{
					ID:         "B-01",
					Text:       "Unconfirmed core claim",
					Confidence: Unconfirmed,
					Priority:   PriorityCore,
				},
			},
		},
	}

	edges := CollectEdges(files, now, []string{"gates", "spawn"}, 5)

	if len(edges) != 3 {
		t.Fatalf("got %d edges, want 3", len(edges))
	}

	// Should be ordered: tensions, stale_active, unconfirmed_core
	if edges[0].Type != "tension" {
		t.Errorf("edge 0 type = %q, want %q", edges[0].Type, "tension")
	}
	if edges[1].Type != "stale_active" {
		t.Errorf("edge 1 type = %q, want %q", edges[1].Type, "stale_active")
	}
	if edges[2].Type != "unconfirmed_core" {
		t.Errorf("edge 2 type = %q, want %q", edges[2].Type, "unconfirmed_core")
	}
}

func TestCollectEdges_MaxLimit(t *testing.T) {
	now := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)

	files := map[string]*File{
		"model-a": {
			Model: "model-a",
			Claims: []Claim{
				{
					ID: "A-01", Text: "t1", Confidence: Confirmed, Priority: PriorityCore,
					Tensions: []Tension{
						{Claim: "X-01", Model: "x", Type: "contradicts", Note: "n1"},
						{Claim: "X-02", Model: "x", Type: "contradicts", Note: "n2"},
						{Claim: "X-03", Model: "x", Type: "contradicts", Note: "n3"},
					},
				},
			},
		},
	}

	edges := CollectEdges(files, now, nil, 2)
	if len(edges) != 2 {
		t.Errorf("got %d edges with max=2, want 2", len(edges))
	}
}

func TestFormatEdges(t *testing.T) {
	edges := []Edge{
		{Type: "tension", ClaimID: "A-01", Detail: "A-01 vs B-01: they disagree"},
		{Type: "stale_active", ClaimID: "A-02", Detail: "A-02 (stale claim): last validated 2026-01-01"},
	}

	output := FormatEdges(edges)

	if !strings.Contains(output, "Knowledge Edges:") {
		t.Error("missing 'Knowledge Edges:' header")
	}
	if !strings.Contains(output, "Tensions:") {
		t.Error("missing 'Tensions:' subheader")
	}
	if !strings.Contains(output, "Stale in active area:") {
		t.Error("missing 'Stale in active area:' subheader")
	}
	if !strings.Contains(output, "A-01 vs B-01") {
		t.Error("missing tension detail")
	}
}

func TestFormatEdges_Empty(t *testing.T) {
	output := FormatEdges(nil)
	if output != "" {
		t.Errorf("FormatEdges(nil) = %q, want empty", output)
	}
}

func TestScanAll(t *testing.T) {
	dir := t.TempDir()

	// Create model with claims.yaml
	modelDir := filepath.Join(dir, "test-model")
	os.MkdirAll(modelDir, 0755)
	claimsData := `model: test-model
version: 1
claims:
  - id: T-01
    text: "Test"
    type: observation
    scope: local
    confidence: confirmed
    priority: core
`
	os.WriteFile(filepath.Join(modelDir, "claims.yaml"), []byte(claimsData), 0644)

	// Create model without claims.yaml
	os.MkdirAll(filepath.Join(dir, "no-claims"), 0755)

	result, err := ScanAll(dir)
	if err != nil {
		t.Fatalf("ScanAll failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("got %d models, want 1", len(result))
	}
	if _, ok := result["test-model"]; !ok {
		t.Error("missing 'test-model' in results")
	}
	if result["test-model"].Claims[0].ID != "T-01" {
		t.Errorf("claim ID = %q, want %q", result["test-model"].Claims[0].ID, "T-01")
	}
}

func TestScanAll_NoDir(t *testing.T) {
	result, err := ScanAll("/nonexistent/path")
	if err != nil {
		t.Fatalf("ScanAll on nonexistent should not error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for nonexistent dir, got %v", result)
	}
}

func TestSaveFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "claims.yaml")

	f := &File{
		Model:   "test",
		Version: 1,
		Claims: []Claim{
			{ID: "T-01", Text: "Test", Type: TypeObservation, Confidence: Confirmed, Priority: PriorityCore},
		},
	}

	if err := SaveFile(path, f); err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}
	if loaded.Model != "test" {
		t.Errorf("Model = %q, want %q", loaded.Model, "test")
	}
	if len(loaded.Claims) != 1 {
		t.Errorf("Claims count = %d, want 1", len(loaded.Claims))
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("truncate short = %q", got)
	}
	if got := truncate("this is a longer string", 10); got != "this is..." {
		t.Errorf("truncate long = %q", got)
	}
}
