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

func TestParseContent_FullMetadata(t *testing.T) {
	content := `## Summary (D.E.K.N.)

**Delta:** Integrate toolshed with priceworks
**Evidence:** Investigation findings
**Knowledge:** Key insight
**Next:** Wire PriceCurvePanel

---

# Plan: Toolshed PriceWorks Integration

**Date:** 2026-03-05
**Status:** active
**Owner:** dylan
**Projects:** toolshed, price-watch

**Extracted-From:** .kb/investigations/2026-03-05-inv-design-foo.md
**Supersedes:**
**Superseded-By:**

---

## Objective
Integrate toolshed pricing panel with priceworks data.

---

## Phases
### Phase 1: Wire PriceCurvePanel
**Goal:** Connect panel to data source
**Deliverables:** Working panel component
**Exit criteria:** Panel renders with real data
**Depends on:** none
**Beads:** orch-go-abc12, orch-go-def34

### Phase 2: Forward Simulation
**Goal:** Build simulation engine
**Deliverables:** Simulation API endpoint
**Exit criteria:** API returns valid results
**Depends on:** Phase 1
**Beads:** orch-go-ghi56

### Phase 3: Strategic Landscape
**Goal:** Build dashboard view
**Deliverables:** Dashboard component
**Exit criteria:** Dashboard displays landscape
**Depends on:** Phase 2
**Beads:** orch-go-jkl78, orch-go-mno90

---

## Success Criteria
- [ ] Panel renders with real data
- [x] API contract defined
`

	p := ParseContent(content, "2026-03-05-plan-toolshed-pw.md")

	if p.Title != "Toolshed PriceWorks Integration" {
		t.Errorf("title = %q, want %q", p.Title, "Toolshed PriceWorks Integration")
	}
	if p.Status != "active" {
		t.Errorf("status = %q, want %q", p.Status, "active")
	}
	if p.Date != "2026-03-05" {
		t.Errorf("date = %q, want %q", p.Date, "2026-03-05")
	}
	if p.Owner != "dylan" {
		t.Errorf("owner = %q, want %q", p.Owner, "dylan")
	}
	if p.Filename != "2026-03-05-plan-toolshed-pw.md" {
		t.Errorf("filename = %q, want %q", p.Filename, "2026-03-05-plan-toolshed-pw.md")
	}
	if len(p.Projects) != 2 {
		t.Fatalf("projects len = %d, want 2", len(p.Projects))
	}
	if p.Projects[0] != "toolshed" || p.Projects[1] != "price-watch" {
		t.Errorf("projects = %v, want [toolshed, price-watch]", p.Projects)
	}
	if len(p.Phases) != 3 {
		t.Fatalf("phases len = %d, want 3", len(p.Phases))
	}

	// Phase 1
	p1 := p.Phases[0]
	if p1.Name != "Wire PriceCurvePanel" {
		t.Errorf("phase1 name = %q", p1.Name)
	}
	if len(p1.BeadsIDs) != 2 {
		t.Fatalf("phase1 beads len = %d, want 2", len(p1.BeadsIDs))
	}
	if p1.BeadsIDs[0] != "orch-go-abc12" || p1.BeadsIDs[1] != "orch-go-def34" {
		t.Errorf("phase1 beads = %v", p1.BeadsIDs)
	}
	if p1.DependsOn != "none" {
		t.Errorf("phase1 depends = %q, want %q", p1.DependsOn, "none")
	}

	// Phase 2
	p2 := p.Phases[1]
	if p2.Name != "Forward Simulation" {
		t.Errorf("phase2 name = %q", p2.Name)
	}
	if len(p2.BeadsIDs) != 1 {
		t.Fatalf("phase2 beads len = %d, want 1", len(p2.BeadsIDs))
	}
	if p2.DependsOn != "Phase 1" {
		t.Errorf("phase2 depends = %q", p2.DependsOn)
	}

	// Phase 3
	p3 := p.Phases[2]
	if p3.Name != "Strategic Landscape" {
		t.Errorf("phase3 name = %q", p3.Name)
	}
	if len(p3.BeadsIDs) != 2 {
		t.Fatalf("phase3 beads len = %d, want 2", len(p3.BeadsIDs))
	}
}

func TestParseContent_MinimalContent(t *testing.T) {
	content := `# Plan: Simple Task

**Date:** 2026-03-01
**Status:** completed
`

	p := ParseContent(content, "2026-03-01-plan-simple.md")

	if p.Title != "Simple Task" {
		t.Errorf("title = %q, want %q", p.Title, "Simple Task")
	}
	if p.Status != "completed" {
		t.Errorf("status = %q, want %q", p.Status, "completed")
	}
	if len(p.Phases) != 0 {
		t.Errorf("phases len = %d, want 0", len(p.Phases))
	}
}

func TestParseContent_SupersededStatus(t *testing.T) {
	content := `# Plan: Old Plan

**Date:** 2026-02-01
**Status:** superseded
**Superseded-By:** .kb/plans/2026-03-01-plan-new.md
`

	p := ParseContent(content, "2026-02-01-plan-old.md")

	if p.Status != "superseded" {
		t.Errorf("status = %q, want %q", p.Status, "superseded")
	}
	if p.SupersededBy != ".kb/plans/2026-03-01-plan-new.md" {
		t.Errorf("superseded_by = %q", p.SupersededBy)
	}
}

func TestParseContent_CoordinationPlanTitle(t *testing.T) {
	content := `# Coordination Plan: My Coordination Plan

**Date:** 2026-03-05
**Status:** active
`

	p := ParseContent(content, "2026-03-05-plan-coord.md")

	if p.Title != "My Coordination Plan" {
		t.Errorf("title = %q, want %q", p.Title, "My Coordination Plan")
	}
	if p.Status != "active" {
		t.Errorf("status = %q, want %q", p.Status, "active")
	}
}

func TestParseContent_StatusNotOverriddenByPhaseStatus(t *testing.T) {
	content := `# Plan: Test

**Status:** active

## Phases
### Phase 1: Do stuff
**Status:** in-progress
`

	p := ParseContent(content, "test.md")

	if p.Status != "active" {
		t.Errorf("status = %q, want %q (phase status should not override plan status)", p.Status, "active")
	}
}

func TestCollectAllBeadsIDs(t *testing.T) {
	p := &File{
		Phases: []Phase{
			{BeadsIDs: []string{"abc", "def"}},
			{BeadsIDs: []string{"ghi"}},
			{BeadsIDs: []string{}},
		},
	}

	ids := CollectAllBeadsIDs(p)
	if len(ids) != 3 {
		t.Fatalf("ids len = %d, want 3", len(ids))
	}
}

func TestParseBeadsLine(t *testing.T) {
	tests := []struct {
		line string
		want []string
	}{
		{"**Beads:** orch-go-abc12, orch-go-def34", []string{"orch-go-abc12", "orch-go-def34"}},
		{"**Beads:** orch-go-abc12", []string{"orch-go-abc12"}},
		{"**Beads:** ", nil},
		{"**Beads:** none", nil},
	}

	for _, tt := range tests {
		got := ParseBeadsLine(tt.line)
		if len(got) != len(tt.want) {
			t.Errorf("ParseBeadsLine(%q) len = %d, want %d", tt.line, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("ParseBeadsLine(%q)[%d] = %q, want %q", tt.line, i, got[i], tt.want[i])
			}
		}
	}
}
