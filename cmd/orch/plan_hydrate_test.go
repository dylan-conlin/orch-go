package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDependsOn(t *testing.T) {
	tests := []struct {
		input string
		want  []int // 0-indexed phase numbers
	}{
		{"Nothing", nil},
		{"none", nil},
		{"Nothing — uses existing data.", nil},
		{"Phase 1", []int{0}},
		{"Phase 1 (need classification before acting)", []int{0}},
		{"Phase 2", []int{1}},
		{"Phases 1-3", []int{0, 1, 2}},
		{"Phases 1-3 + 2-4 weeks of data accumulation.", []int{0, 1, 2}},
		{"Phase 1, Phase 3", []int{0, 2}},
	}

	for _, tt := range tests {
		got := parseDependsOn(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("parseDependsOn(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parseDependsOn(%q)[%d] = %d, want %d", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestExtractSlugFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"2026-03-11-gate-signal-vs-noise.md", "gate-signal-vs-noise"},
		{"2026-03-05-plan-toolshed-pw.md", "plan-toolshed-pw"},
		{"2026-03-08-harness-engineering-implementation.md", "harness-engineering-implementation"},
		{"no-date-prefix.md", "no-date-prefix"},
	}

	for _, tt := range tests {
		got := extractSlugFromFilename(tt.filename)
		if got != tt.want {
			t.Errorf("extractSlugFromFilename(%q) = %q, want %q", tt.filename, got, tt.want)
		}
	}
}

func TestUpdatePlanWithBeadsIDs(t *testing.T) {
	content := `# Plan: Test Plan

**Date:** 2026-03-11
**Status:** active

## Phases

### Phase 1: Setup
**Goal:** Set things up
**Depends on:** Nothing
**Beads:**

### Phase 2: Build
**Goal:** Build the thing
**Depends on:** Phase 1
**Beads:**

### Phase 3: Verify
**Goal:** Verify it works
**Depends on:** Phase 2
**Beads:** orch-go-existing

## Success Criteria
- [ ] Done
`

	dir := t.TempDir()
	planPath := filepath.Join(dir, "test-plan.md")
	os.WriteFile(planPath, []byte(content), 0o644)

	// Phase 0 and 1 get IDs, phase 2 already has one (should not be touched)
	phaseIDs := map[int]string{
		0: "orch-go-aaa",
		1: "orch-go-bbb",
	}

	err := updatePlanWithBeadsIDs(planPath, phaseIDs)
	if err != nil {
		t.Fatalf("updatePlanWithBeadsIDs: %v", err)
	}

	updated, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("read updated plan: %v", err)
	}

	result := string(updated)

	// Phase 1 should have the new ID
	if !strings.Contains(result, "**Beads:** orch-go-aaa") {
		t.Error("Phase 1 should have beads ID orch-go-aaa")
	}

	// Phase 2 should have the new ID
	if !strings.Contains(result, "**Beads:** orch-go-bbb") {
		t.Error("Phase 2 should have beads ID orch-go-bbb")
	}

	// Phase 3 should keep existing ID
	if !strings.Contains(result, "**Beads:** orch-go-existing") {
		t.Error("Phase 3 should still have existing beads ID")
	}

	// Rest of content preserved
	if !strings.Contains(result, "## Success Criteria") {
		t.Error("Success Criteria section should be preserved")
	}
}

func TestBuildPhaseTitle(t *testing.T) {
	got := buildPhaseTitle("Gate Signal vs Noise", 1, "Gate census and classification")
	want := "Plan: Gate Signal vs Noise — Phase 1: Gate census and classification"
	if got != want {
		t.Errorf("buildPhaseTitle = %q, want %q", got, want)
	}
}

func TestBuildPhaseDescription(t *testing.T) {
	phase := PlanPhase{
		Name:      "Gate census",
		Goal:      "Enumerate every gate",
		DependsOn: "Nothing",
	}
	desc := buildPhaseDescription(phase, "Gate Signal vs Noise", "orch-go-parent")
	if !strings.Contains(desc, "Enumerate every gate") {
		t.Error("description should contain goal")
	}
	if !strings.Contains(desc, "orch-go-parent") {
		t.Error("description should reference parent plan issue")
	}
}

func TestHydratePlan_SkipsAlreadyHydrated(t *testing.T) {
	// Plan where all phases already have beads IDs
	plan := PlanFile{
		Title:    "Already Hydrated",
		Filename: "2026-03-11-already-hydrated.md",
		Status:   "active",
		Phases: []PlanPhase{
			{Name: "Phase 1", BeadsIDs: []string{"orch-go-abc"}},
			{Name: "Phase 2", BeadsIDs: []string{"orch-go-def"}},
		},
	}

	toHydrate := phasesNeedingHydration(plan)
	if len(toHydrate) != 0 {
		t.Errorf("expected 0 phases needing hydration, got %d", len(toHydrate))
	}
}

func TestHydratePlan_FindsUnhydrated(t *testing.T) {
	plan := PlanFile{
		Title:    "Partial",
		Filename: "2026-03-11-partial.md",
		Status:   "active",
		Phases: []PlanPhase{
			{Name: "Phase 1", BeadsIDs: []string{"orch-go-abc"}},
			{Name: "Phase 2", BeadsIDs: nil},
			{Name: "Phase 3", BeadsIDs: []string{}},
		},
	}

	toHydrate := phasesNeedingHydration(plan)
	if len(toHydrate) != 2 {
		t.Errorf("expected 2 phases needing hydration, got %d: %v", len(toHydrate), toHydrate)
	}
	if toHydrate[0] != 1 || toHydrate[1] != 2 {
		t.Errorf("expected indices [1, 2], got %v", toHydrate)
	}
}
