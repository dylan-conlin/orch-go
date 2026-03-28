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
		{
			"unconfirmed with recent evidence is not eligible (already probed)",
			Claim{
				Confidence: Unconfirmed, Priority: PriorityCore,
				Evidence: []Evidence{{Source: "probe result", Date: "2026-03-10", Verdict: "extends"}},
			},
			false,
		},
		{
			"unconfirmed with old evidence is eligible",
			Claim{
				Confidence: Unconfirmed, Priority: PriorityCore,
				Evidence: []Evidence{{Source: "old probe", Date: "2026-01-01", Verdict: "extends"}},
			},
			true,
		},
		{
			"stale with recent evidence is not eligible",
			Claim{
				Confidence: Stale, Priority: PriorityCore,
				Evidence: []Evidence{{Source: "recent probe", Date: "2026-03-15", Verdict: "confirms"}},
			},
			false,
		},
		{
			"confirmed stale with recent evidence is not eligible",
			Claim{
				Confidence: Confirmed, Priority: PriorityCore, LastValidated: "2026-01-01",
				Evidence: []Evidence{{Source: "recent re-check", Date: "2026-03-18", Verdict: "confirms"}},
			},
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

func TestClaimHasRecentEvidence(t *testing.T) {
	now := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		evidence []Evidence
		want     bool
	}{
		{"no evidence", nil, false},
		{"empty date", []Evidence{{Date: ""}}, false},
		{"invalid date", []Evidence{{Date: "bad"}}, false},
		{"recent evidence", []Evidence{{Date: "2026-03-10"}}, true},
		{"old evidence", []Evidence{{Date: "2026-01-01"}}, false},
		{"mixed old and recent", []Evidence{{Date: "2026-01-01"}, {Date: "2026-03-15"}}, true},
		{"exactly at threshold", []Evidence{{Date: "2026-02-17"}}, true},
		{"one day past threshold", []Evidence{{Date: "2026-02-16"}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Claim{Evidence: tt.evidence}
			got := c.HasRecentEvidence(now)
			if got != tt.want {
				t.Errorf("HasRecentEvidence() = %v, want %v", got, tt.want)
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

func TestCollectClaimStatus(t *testing.T) {
	now := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)

	files := map[string]*File{
		"model-a": {
			Model: "model-a",
			Claims: []Claim{
				{ID: "A-01", Confidence: Confirmed, Priority: PriorityCore, LastValidated: "2026-03-20"},
				{ID: "A-02", Confidence: Unconfirmed, Priority: PriorityCore},
				{ID: "A-03", Confidence: Confirmed, Priority: PrioritySupporting, LastValidated: "2026-03-15"},
			},
		},
		"model-b": {
			Model: "model-b",
			Claims: []Claim{
				{ID: "B-01", Confidence: Confirmed, Priority: PriorityCore, LastValidated: "2026-03-20"},
				{ID: "B-02", Confidence: Confirmed, Priority: PrioritySupporting, LastValidated: "2026-03-20"},
			},
		},
		"model-c": {
			Model: "model-c",
			Claims: []Claim{
				{ID: "C-01", Confidence: Unconfirmed, Priority: PriorityCore},
				{ID: "C-02", Confidence: Unconfirmed, Priority: PriorityCore},
				{ID: "C-03", Confidence: Contested, Priority: PrioritySupporting},
			},
		},
	}

	statuses := CollectClaimStatus(files, now)

	// model-b should be excluded (all confirmed, not stale)
	if len(statuses) != 2 {
		t.Fatalf("got %d statuses, want 2 (model-b excluded)", len(statuses))
	}

	// model-c should be first (2 core untested > model-a's 1)
	if statuses[0].ModelName != "model-c" {
		t.Errorf("first status model = %q, want %q", statuses[0].ModelName, "model-c")
	}
	if statuses[0].CoreUntested != 2 {
		t.Errorf("model-c CoreUntested = %d, want 2", statuses[0].CoreUntested)
	}
	if statuses[0].Contested != 1 {
		t.Errorf("model-c Contested = %d, want 1", statuses[0].Contested)
	}

	// model-a should be second
	if statuses[1].ModelName != "model-a" {
		t.Errorf("second status model = %q, want %q", statuses[1].ModelName, "model-a")
	}
	if statuses[1].CoreUntested != 1 {
		t.Errorf("model-a CoreUntested = %d, want 1", statuses[1].CoreUntested)
	}
	if statuses[1].Confirmed != 2 {
		t.Errorf("model-a Confirmed = %d, want 2", statuses[1].Confirmed)
	}
}

func TestCollectClaimStatus_StaleConfirmed(t *testing.T) {
	now := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)

	files := map[string]*File{
		"model-a": {
			Model: "model-a",
			Claims: []Claim{
				{ID: "A-01", Confidence: Confirmed, Priority: PriorityCore, LastValidated: "2026-01-01"},
			},
		},
	}

	statuses := CollectClaimStatus(files, now)

	if len(statuses) != 1 {
		t.Fatalf("got %d statuses, want 1", len(statuses))
	}
	if statuses[0].Stale != 1 {
		t.Errorf("Stale = %d, want 1", statuses[0].Stale)
	}
	if statuses[0].CoreUntested != 1 {
		t.Errorf("CoreUntested = %d, want 1", statuses[0].CoreUntested)
	}
}

func TestCollectClaimStatus_Empty(t *testing.T) {
	now := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)
	statuses := CollectClaimStatus(nil, now)
	if len(statuses) != 0 {
		t.Errorf("got %d statuses for nil input, want 0", len(statuses))
	}
}

func TestCollectRecentDisconfirmations(t *testing.T) {
	now := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)

	files := map[string]*File{
		"model-a": {
			Model: "model-a",
			Claims: []Claim{
				{
					ID: "A-01", Text: "Claim that was contradicted",
					Evidence: []Evidence{
						{Source: "Old confirmation", Date: "2026-01-15", Verdict: "confirms"},
						{Source: "Recent contradiction", Date: "2026-03-25", Verdict: "contradicts"},
					},
				},
				{
					ID: "A-02", Text: "Claim with old contradiction",
					Evidence: []Evidence{
						{Source: "Old contradiction", Date: "2026-02-01", Verdict: "contradicts"},
					},
				},
			},
		},
		"model-b": {
			Model: "model-b",
			Claims: []Claim{
				{
					ID: "B-01", Text: "Another recent contradiction",
					Evidence: []Evidence{
						{Source: "Very recent finding", Date: "2026-03-27", Verdict: "contradicts"},
					},
				},
				{
					ID: "B-02", Text: "Confirmed claim",
					Evidence: []Evidence{
						{Source: "Confirmation", Date: "2026-03-26", Verdict: "confirms"},
					},
				},
			},
		},
	}

	result := CollectRecentDisconfirmations(files, now, 7)

	if len(result) != 2 {
		t.Fatalf("got %d disconfirmations, want 2", len(result))
	}

	// Check that we got A-01 and B-01 (not A-02's old contradiction or B-02's confirmation)
	ids := map[string]bool{}
	for _, d := range result {
		ids[d.ClaimID] = true
	}
	if !ids["A-01"] {
		t.Error("missing A-01 (recent contradiction)")
	}
	if !ids["B-01"] {
		t.Error("missing B-01 (recent contradiction)")
	}
	if ids["A-02"] {
		t.Error("should not include A-02 (old contradiction)")
	}
}

func TestCollectRecentDisconfirmations_Empty(t *testing.T) {
	now := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)
	result := CollectRecentDisconfirmations(nil, now, 7)
	if len(result) != 0 {
		t.Errorf("got %d for nil input, want 0", len(result))
	}
}

func TestFormatClaimSurface(t *testing.T) {
	statuses := []ModelClaimStatus{
		{ModelName: "model-a", Total: 6, Confirmed: 4, Unconfirmed: 1, CoreUntested: 1, Contested: 1},
	}
	disconfirmations := []RecentDisconfirmation{
		{ModelName: "model-b", ClaimID: "B-01", ClaimText: "Test claim", Source: "Recent finding", Date: "2026-03-25"},
	}
	edges := []Edge{
		{Type: "tension", ClaimID: "A-01", Detail: "A-01 vs B-01: they disagree"},
	}

	output := FormatClaimSurface(statuses, disconfirmations, edges)

	if !strings.Contains(output, "Knowledge Edges:") {
		t.Error("missing 'Knowledge Edges:' header")
	}
	if !strings.Contains(output, "Untested claims:") {
		t.Error("missing 'Untested claims:' section")
	}
	if !strings.Contains(output, "model-a: 4/6 confirmed, 1 untested core, 1 contested") {
		t.Errorf("missing or wrong model-a summary in output:\n%s", output)
	}
	if !strings.Contains(output, "Recently disconfirmed:") {
		t.Error("missing 'Recently disconfirmed:' section")
	}
	if !strings.Contains(output, "B-01 (model-b)") {
		t.Error("missing disconfirmation detail")
	}
	if !strings.Contains(output, "Tensions:") {
		t.Error("missing 'Tensions:' section")
	}
}

func TestFormatClaimSurface_Empty(t *testing.T) {
	output := FormatClaimSurface(nil, nil, nil)
	if output != "" {
		t.Errorf("expected empty output, got %q", output)
	}
}

func TestFormatClaimSurface_OnlyEdges(t *testing.T) {
	edges := []Edge{
		{Type: "unconfirmed_core", ClaimID: "A-01", Detail: "A-01: untested core claim"},
	}
	output := FormatClaimSurface(nil, nil, edges)
	if !strings.Contains(output, "Knowledge Edges:") {
		t.Error("missing header")
	}
	if !strings.Contains(output, "Unconfirmed core:") {
		t.Error("missing unconfirmed core section")
	}
	if strings.Contains(output, "Untested claims:") {
		t.Error("should not have untested claims section when no statuses")
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
