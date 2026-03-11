package orient

import (
	"os"
	"path/filepath"
	"testing"
)

const activePlanContent = `# Coordination Plan: Ship Dashboard V2

**Created:** 2026-03-01
**Status:** active
**Projects:** orch-go, price-watch

## TLDR
Multi-project dashboard rewrite with new agent cards and plan visibility.

## Phases
### Phase 1: Foundation
**Status:** complete
**Beads:** orch-go-abc1, orch-go-def2

### Phase 2: Agent Cards
**Status:** in-progress
**Beads:** orch-go-ghi3

### Phase 3: Plan Panel
**Status:** ready
**Beads:** orch-go-jkl4

## Blocking Logic
Phase 2 depends on Phase 1 foundation. Phase 3 blocked until agent cards stable.
`

const completedPlanContent = `# Coordination Plan: Migrate to RPC

**Created:** 2026-02-15
**Status:** completed
**Projects:** orch-go

## TLDR
Replace CLI subprocess calls with RPC client.

## Phases
### Phase 1: RPC Client
**Status:** complete
`

const supersededPlanContent = `# Coordination Plan: Old Approach

**Created:** 2026-01-10
**Status:** superseded
**Projects:** orch-go

## TLDR
This approach was replaced.
`

func TestScanActivePlans(t *testing.T) {
	dir := t.TempDir()

	// Write plan files
	os.WriteFile(filepath.Join(dir, "2026-03-01-plan-dashboard-v2.md"), []byte(activePlanContent), 0644)
	os.WriteFile(filepath.Join(dir, "2026-02-15-plan-rpc-migration.md"), []byte(completedPlanContent), 0644)
	os.WriteFile(filepath.Join(dir, "2026-01-10-plan-old-approach.md"), []byte(supersededPlanContent), 0644)
	os.WriteFile(filepath.Join(dir, ".gitkeep"), []byte(""), 0644) // non-md file

	plans, err := ScanActivePlans(dir)
	if err != nil {
		t.Fatalf("ScanActivePlans failed: %v", err)
	}

	if len(plans) != 1 {
		t.Fatalf("expected 1 active plan, got %d", len(plans))
	}

	plan := plans[0]
	if plan.Title != "Ship Dashboard V2" {
		t.Errorf("expected title 'Ship Dashboard V2', got %q", plan.Title)
	}
	if plan.Status != "active" {
		t.Errorf("expected status 'active', got %q", plan.Status)
	}
	if plan.TLDR != "Multi-project dashboard rewrite with new agent cards and plan visibility." {
		t.Errorf("unexpected TLDR: %q", plan.TLDR)
	}
	if len(plan.Projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(plan.Projects))
	}
	if plan.Projects[0] != "orch-go" || plan.Projects[1] != "price-watch" {
		t.Errorf("unexpected projects: %v", plan.Projects)
	}
	if len(plan.Phases) != 3 {
		t.Fatalf("expected 3 phases, got %d", len(plan.Phases))
	}
	if plan.Phases[0].Status != "complete" {
		t.Errorf("phase 1 status: expected 'complete', got %q", plan.Phases[0].Status)
	}
	if plan.Phases[1].Status != "in-progress" {
		t.Errorf("phase 2 status: expected 'in-progress', got %q", plan.Phases[1].Status)
	}
	if plan.Phases[2].Status != "ready" {
		t.Errorf("phase 3 status: expected 'ready', got %q", plan.Phases[2].Status)
	}
}

func TestScanActivePlans_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	plans, err := ScanActivePlans(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plans) != 0 {
		t.Errorf("expected 0 plans, got %d", len(plans))
	}
}

func TestScanActivePlans_NonexistentDir(t *testing.T) {
	_, err := ScanActivePlans("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestParsePlanFile_Name(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "2026-03-01-plan-dashboard-v2.md"), []byte(activePlanContent), 0644)

	plans, _ := ScanActivePlans(dir)
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if plans[0].Name != "2026-03-01-plan-dashboard-v2" {
		t.Errorf("expected name '2026-03-01-plan-dashboard-v2', got %q", plans[0].Name)
	}
}

func TestParsePlanFile_BeadsIDs(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "2026-03-01-plan-dashboard-v2.md"), []byte(activePlanContent), 0644)

	plans, err := ScanActivePlans(dir)
	if err != nil {
		t.Fatalf("ScanActivePlans failed: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}

	plan := plans[0]

	// Phase 1 has two beads IDs
	if len(plan.Phases[0].BeadsIDs) != 2 {
		t.Fatalf("phase 1: expected 2 beads IDs, got %d", len(plan.Phases[0].BeadsIDs))
	}
	if plan.Phases[0].BeadsIDs[0] != "orch-go-abc1" {
		t.Errorf("phase 1 beads[0]: expected 'orch-go-abc1', got %q", plan.Phases[0].BeadsIDs[0])
	}
	if plan.Phases[0].BeadsIDs[1] != "orch-go-def2" {
		t.Errorf("phase 1 beads[1]: expected 'orch-go-def2', got %q", plan.Phases[0].BeadsIDs[1])
	}

	// Phase 2 has one beads ID
	if len(plan.Phases[1].BeadsIDs) != 1 {
		t.Fatalf("phase 2: expected 1 beads ID, got %d", len(plan.Phases[1].BeadsIDs))
	}
	if plan.Phases[1].BeadsIDs[0] != "orch-go-ghi3" {
		t.Errorf("phase 2 beads[0]: expected 'orch-go-ghi3', got %q", plan.Phases[1].BeadsIDs[0])
	}

	// Phase 3 has one beads ID
	if len(plan.Phases[2].BeadsIDs) != 1 {
		t.Fatalf("phase 3: expected 1 beads ID, got %d", len(plan.Phases[2].BeadsIDs))
	}
}

func TestParsePlanFile_NoBeadsIDs(t *testing.T) {
	content := `# Coordination Plan: No Beads Plan

**Created:** 2026-03-01
**Status:** active

## Phases
### Phase 1: Setup
**Status:** ready
`
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "no-beads.md"), []byte(content), 0644)

	plans, _ := ScanActivePlans(dir)
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if len(plans[0].Phases[0].BeadsIDs) != 0 {
		t.Errorf("expected no beads IDs, got %v", plans[0].Phases[0].BeadsIDs)
	}
}

func TestApplyBeadsProgress(t *testing.T) {
	plans := []PlanSummary{
		{
			Title: "Test Plan",
			Phases: []PlanPhase{
				{Name: "Phase 1: Setup", BeadsIDs: []string{"orch-go-abc1", "orch-go-def2"}},
				{Name: "Phase 2: Build", BeadsIDs: []string{"orch-go-ghi3"}},
				{Name: "Phase 3: Ship", BeadsIDs: []string{"orch-go-jkl4"}},
				{Name: "Phase 4: No Issues"},
			},
		},
	}

	statusMap := map[string]string{
		"orch-go-abc1": "closed",
		"orch-go-def2": "closed",
		"orch-go-ghi3": "in_progress",
		"orch-go-jkl4": "open",
	}

	ApplyBeadsProgress(plans, statusMap)

	// Phase 1: all closed → complete
	if plans[0].Phases[0].Status != "complete" {
		t.Errorf("phase 1: expected 'complete', got %q", plans[0].Phases[0].Status)
	}
	// Phase 2: in_progress
	if plans[0].Phases[1].Status != "in-progress" {
		t.Errorf("phase 2: expected 'in-progress', got %q", plans[0].Phases[1].Status)
	}
	// Phase 3: open → ready
	if plans[0].Phases[2].Status != "ready" {
		t.Errorf("phase 3: expected 'ready', got %q", plans[0].Phases[2].Status)
	}
	// Phase 4: no issues — status unchanged
	if plans[0].Phases[3].Status != "" {
		t.Errorf("phase 4: expected empty status (unchanged), got %q", plans[0].Phases[3].Status)
	}

	// Check progress summary
	if plans[0].Progress == "" {
		t.Error("expected non-empty progress summary")
	}
	if plans[0].Progress != "1/3 complete" {
		t.Errorf("expected '1/3 complete', got %q", plans[0].Progress)
	}
}

func TestCollectPlanBeadsIDs(t *testing.T) {
	plans := []PlanSummary{
		{
			Phases: []PlanPhase{
				{BeadsIDs: []string{"id1", "id2"}},
				{BeadsIDs: []string{"id3"}},
				{}, // no beads
			},
		},
		{
			Phases: []PlanPhase{
				{BeadsIDs: []string{"id4"}},
			},
		},
	}

	ids := CollectPlanBeadsIDs(plans)
	if len(ids) != 4 {
		t.Fatalf("expected 4 IDs, got %d", len(ids))
	}
}
