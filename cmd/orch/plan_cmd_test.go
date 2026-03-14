package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/plan"
)

func TestParsePlanFile(t *testing.T) {
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

	p := plan.ParseContent(content, "2026-03-05-plan-toolshed-pw.md")

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

func TestParsePlanFile_MinimalContent(t *testing.T) {
	content := `# Plan: Simple Task

**Date:** 2026-03-01
**Status:** completed
`

	p := plan.ParseContent(content, "2026-03-01-plan-simple.md")

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

func TestParsePlanFile_SupersededStatus(t *testing.T) {
	content := `# Plan: Old Plan

**Date:** 2026-02-01
**Status:** superseded
**Superseded-By:** .kb/plans/2026-03-01-plan-new.md
`

	p := plan.ParseContent(content, "2026-02-01-plan-old.md")

	if p.Status != "superseded" {
		t.Errorf("status = %q, want %q", p.Status, "superseded")
	}
	if p.SupersededBy != ".kb/plans/2026-03-01-plan-new.md" {
		t.Errorf("superseded_by = %q", p.SupersededBy)
	}
}

func TestScanPlansDir(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, ".kb", "plans")
	os.MkdirAll(plansDir, 0o755)

	// Create two plan files
	plan1 := `# Plan: Active Plan

**Date:** 2026-03-05
**Status:** active
`
	plan2 := `# Plan: Completed Plan

**Date:** 2026-02-01
**Status:** completed
`
	plan3 := `# Plan: Superseded Plan

**Date:** 2026-01-15
**Status:** superseded
`

	os.WriteFile(filepath.Join(plansDir, "2026-03-05-plan-active.md"), []byte(plan1), 0o644)
	os.WriteFile(filepath.Join(plansDir, "2026-02-01-plan-completed.md"), []byte(plan2), 0o644)
	os.WriteFile(filepath.Join(plansDir, "2026-01-15-plan-superseded.md"), []byte(plan3), 0o644)
	os.WriteFile(filepath.Join(plansDir, ".gitkeep"), []byte(""), 0o644) // Should be ignored

	plans, err := plan.ScanDir(plansDir)
	if err != nil {
		t.Fatalf("ScanDir error: %v", err)
	}

	if len(plans) != 3 {
		t.Fatalf("plans len = %d, want 3", len(plans))
	}
}

func TestScanPlansDir_FilterActive(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, ".kb", "plans")
	os.MkdirAll(plansDir, 0o755)

	plan1 := `# Plan: Active Plan

**Date:** 2026-03-05
**Status:** active
`
	plan2 := `# Plan: Completed Plan

**Date:** 2026-02-01
**Status:** completed
`

	os.WriteFile(filepath.Join(plansDir, "2026-03-05-plan-active.md"), []byte(plan1), 0o644)
	os.WriteFile(filepath.Join(plansDir, "2026-02-01-plan-completed.md"), []byte(plan2), 0o644)

	plans, err := plan.ScanDir(plansDir)
	if err != nil {
		t.Fatalf("ScanDir error: %v", err)
	}

	active := plan.FilterByStatus(plans, "active")
	if len(active) != 1 {
		t.Fatalf("active plans len = %d, want 1", len(active))
	}
	if active[0].Title != "Active Plan" {
		t.Errorf("active plan title = %q", active[0].Title)
	}
}

func TestCollectAllBeadsIDs(t *testing.T) {
	p := &PlanFile{
		Phases: []PlanPhase{
			{BeadsIDs: []string{"abc", "def"}},
			{BeadsIDs: []string{"ghi"}},
			{BeadsIDs: []string{}},
		},
	}

	ids := plan.CollectAllBeadsIDs(p)
	if len(ids) != 3 {
		t.Fatalf("ids len = %d, want 3", len(ids))
	}
}

func TestFormatPlanShow(t *testing.T) {
	p := &PlanFile{
		Title:    "Test Plan",
		Status:   "active",
		Date:     "2026-03-05",
		Owner:    "dylan",
		Filename: "2026-03-05-plan-test.md",
		Projects: []string{"orch-go", "price-watch"},
		Phases: []PlanPhase{
			{
				Name:      "Setup",
				DependsOn: "none",
				BeadsIDs:  []string{"orch-go-abc"},
			},
			{
				Name:      "Implementation",
				DependsOn: "Phase 1",
				BeadsIDs:  []string{"orch-go-def", "orch-go-ghi"},
			},
		},
	}

	output := formatPlanShow(p, nil)

	// Should contain key elements
	if !planTestContains(output, "Test Plan") {
		t.Error("missing title")
	}
	if !planTestContains(output, "active") {
		t.Error("missing status")
	}
	if !planTestContains(output, "Setup") {
		t.Error("missing phase 1")
	}
	if !planTestContains(output, "Implementation") {
		t.Error("missing phase 2")
	}
}

func TestFormatPlanStatus(t *testing.T) {
	plans := []PlanFile{
		{
			Title:    "Plan A",
			Status:   "active",
			Date:     "2026-03-05",
			Filename: "2026-03-05-plan-a.md",
			Phases:   []PlanPhase{{Name: "Phase 1"}},
		},
		{
			Title:    "Plan B",
			Status:   "completed",
			Date:     "2026-02-01",
			Filename: "2026-02-01-plan-b.md",
			Phases:   []PlanPhase{{Name: "Phase 1"}, {Name: "Phase 2"}},
		},
	}

	output := formatPlanStatus(plans)

	if !planTestContains(output, "Plan A") {
		t.Error("missing Plan A")
	}
	if !planTestContains(output, "Plan B") {
		t.Error("missing Plan B")
	}
	if !planTestContains(output, "active") {
		t.Error("missing active status")
	}
}

func planTestContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestParsePlanFile_CoordinationPlanTitle(t *testing.T) {
	content := `# Coordination Plan: My Coordination Plan

**Date:** 2026-03-05
**Status:** active
`

	p := plan.ParseContent(content, "2026-03-05-plan-coord.md")

	if p.Title != "My Coordination Plan" {
		t.Errorf("title = %q, want %q", p.Title, "My Coordination Plan")
	}
	if p.Status != "active" {
		t.Errorf("status = %q, want %q", p.Status, "active")
	}
}

func TestParsePlanFile_StatusNotOverriddenByPhaseStatus(t *testing.T) {
	content := `# Plan: Test

**Status:** active

## Phases
### Phase 1: Do stuff
**Status:** in-progress
`

	p := plan.ParseContent(content, "test.md")

	if p.Status != "active" {
		t.Errorf("status = %q, want %q (phase status should not override plan status)", p.Status, "active")
	}
}

func TestParsePlanPhaseBeads(t *testing.T) {
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
		got := plan.ParseBeadsLine(tt.line)
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
