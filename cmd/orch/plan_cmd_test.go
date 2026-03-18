package main

import (
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/plan"
)

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

	if !strings.Contains(output, "Test Plan") {
		t.Error("missing title")
	}
	if !strings.Contains(output, "active") {
		t.Error("missing status")
	}
	if !strings.Contains(output, "Setup") {
		t.Error("missing phase 1")
	}
	if !strings.Contains(output, "Implementation") {
		t.Error("missing phase 2")
	}
}

func TestFormatPlanStatus(t *testing.T) {
	plans := []plan.File{
		{
			Title:    "Plan A",
			Status:   "active",
			Date:     "2026-03-05",
			Filename: "2026-03-05-plan-a.md",
			Phases:   []plan.Phase{{Name: "Phase 1"}},
		},
		{
			Title:    "Plan B",
			Status:   "completed",
			Date:     "2026-02-01",
			Filename: "2026-02-01-plan-b.md",
			Phases:   []plan.Phase{{Name: "Phase 1"}, {Name: "Phase 2"}},
		},
	}

	output := formatPlanStatus(plans)

	if !strings.Contains(output, "Plan A") {
		t.Error("missing Plan A")
	}
	if !strings.Contains(output, "Plan B") {
		t.Error("missing Plan B")
	}
	if !strings.Contains(output, "active") {
		t.Error("missing active status")
	}
}
