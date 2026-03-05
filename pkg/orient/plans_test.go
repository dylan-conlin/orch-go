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
