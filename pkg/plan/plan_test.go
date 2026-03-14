package plan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseContent(t *testing.T) {
	content := `## Summary (D.E.K.N.)

Some summary text.

---

# Plan: Gate Signal vs Noise

**Date:** 2026-03-11
**Status:** Active
**Owner:** Dylan + orchestrator

## Phases

### Phase 1: Gate census and classification

**Goal:** Enumerate every gate, classify as signal/noise/unknown based on existing data.
**Beads:**
**Depends on:** Nothing

### Phase 2: Fix noise gates

**Goal:** For each noise gate, either fix or remove.
**Beads:** orch-go-a1b2
**Depends on:** Phase 1

### Phase 3: Retrospective accuracy audit

**Goal:** Determine false positive rate from historical data.
**Beads:** orch-go-c3d4, orch-go-e5f6
**Depends on:** Phase 1

## Success Criteria

Some criteria text.
`

	plan := ParseContent(content, "2026-03-11-gate-signal-vs-noise.md")

	if plan.Title != "Gate Signal vs Noise" {
		t.Errorf("expected title 'Gate Signal vs Noise', got %q", plan.Title)
	}
	if plan.Status != "active" {
		t.Errorf("expected status 'active', got %q", plan.Status)
	}
	if plan.Date != "2026-03-11" {
		t.Errorf("expected date '2026-03-11', got %q", plan.Date)
	}
	if len(plan.Phases) != 3 {
		t.Fatalf("expected 3 phases, got %d", len(plan.Phases))
	}

	// Phase 1: no beads
	if len(plan.Phases[0].BeadsIDs) != 0 {
		t.Errorf("phase 1 should have no beads IDs, got %v", plan.Phases[0].BeadsIDs)
	}

	// Phase 2: one bead
	if len(plan.Phases[1].BeadsIDs) != 1 || plan.Phases[1].BeadsIDs[0] != "orch-go-a1b2" {
		t.Errorf("phase 2 beads wrong: %v", plan.Phases[1].BeadsIDs)
	}

	// Phase 3: two beads
	if len(plan.Phases[2].BeadsIDs) != 2 {
		t.Errorf("phase 3 should have 2 beads IDs, got %v", plan.Phases[2].BeadsIDs)
	}
}

func TestIsHydrated(t *testing.T) {
	unhydrated := &File{
		Phases: []Phase{
			{Name: "Phase 1", BeadsIDs: nil},
			{Name: "Phase 2", BeadsIDs: nil},
		},
	}
	if unhydrated.IsHydrated() {
		t.Error("expected unhydrated plan to return false")
	}

	hydrated := &File{
		Phases: []Phase{
			{Name: "Phase 1", BeadsIDs: []string{"orch-go-abc"}},
			{Name: "Phase 2", BeadsIDs: nil},
		},
	}
	if !hydrated.IsHydrated() {
		t.Error("expected hydrated plan to return true")
	}
}

func TestExtractSlugFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"2026-03-11-gate-signal-vs-noise.md", "gate-signal-vs-noise"},
		{"2026-03-05-comprehension-measurement-program.md", "comprehension-measurement-program"},
		{"no-date-prefix.md", "no-date-prefix"},
	}

	for _, tt := range tests {
		got := ExtractSlugFromFilename(tt.filename)
		if got != tt.want {
			t.Errorf("ExtractSlugFromFilename(%q) = %q, want %q", tt.filename, got, tt.want)
		}
	}
}

func TestFilterByStatus(t *testing.T) {
	plans := []File{
		{Title: "Plan 1", Status: "active"},
		{Title: "Plan 2", Status: "completed"},
		{Title: "Plan 3", Status: "active"},
	}

	active := FilterByStatus(plans, "active")
	if len(active) != 2 {
		t.Errorf("expected 2 active plans, got %d", len(active))
	}
}

func TestScanDir(t *testing.T) {
	dir := t.TempDir()

	// Write a plan file
	content := `# Plan: Test Plan

**Date:** 2026-03-11
**Status:** active

## Phases

### Phase 1: First

**Goal:** Do first thing
**Beads:**
**Depends on:** Nothing
`
	if err := os.WriteFile(filepath.Join(dir, "2026-03-11-test-plan.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a non-plan file (should be ignored)
	if err := os.WriteFile(filepath.Join(dir, "README.txt"), []byte("not a plan"), 0o644); err != nil {
		t.Fatal(err)
	}

	plans, err := ScanDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if plans[0].Title != "Test Plan" {
		t.Errorf("expected title 'Test Plan', got %q", plans[0].Title)
	}
}

func TestScanDir_NotExists(t *testing.T) {
	plans, err := ScanDir("/nonexistent/path")
	if err != nil {
		t.Fatalf("expected nil error for nonexistent dir, got %v", err)
	}
	if plans != nil {
		t.Errorf("expected nil plans for nonexistent dir, got %v", plans)
	}
}

func TestParseDependsOn(t *testing.T) {
	tests := []struct {
		input string
		want  []int
	}{
		{"Nothing", nil},
		{"none", nil},
		{"", nil},
		{"Phase 1", []int{0}},
		{"Phase 2", []int{1}},
		{"Phases 1-3", []int{0, 1, 2}},
		{"Phase 1, Phase 3", []int{0, 2}},
		{"Phase 1 (some text)", []int{0}},
	}

	for _, tt := range tests {
		got := ParseDependsOn(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("ParseDependsOn(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("ParseDependsOn(%q)[%d] = %d, want %d", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}
