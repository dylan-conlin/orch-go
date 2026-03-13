package daemon

import (
	"testing"
)

func TestRouteCompletion_EffortSmall(t *testing.T) {
	agent := CompletedAgent{
		BeadsID: "proj-1",
		Labels:  []string{"effort:small"},
	}
	route := RouteCompletion(agent)
	if route.Action != "auto-complete-light" {
		t.Errorf("RouteCompletion() Action = %q, want %q", route.Action, "auto-complete-light")
	}
}

func TestRouteCompletion_AutoTier(t *testing.T) {
	// Without workspace (reviewTier empty), should fall through to label
	agent := CompletedAgent{
		BeadsID: "proj-1",
		Labels:  []string{"effort:medium"},
	}
	route := RouteCompletion(agent)
	if route.Action != "label-ready-review" {
		t.Errorf("RouteCompletion() Action = %q, want %q", route.Action, "label-ready-review")
	}
}

func TestRouteCompletion_DefaultLabelReview(t *testing.T) {
	agent := CompletedAgent{
		BeadsID: "proj-1",
		Labels:  []string{"effort:large"},
	}
	route := RouteCompletion(agent)
	if route.Action != "label-ready-review" {
		t.Errorf("RouteCompletion() Action = %q, want %q", route.Action, "label-ready-review")
	}
}

func TestPrioritizeIssues_SortsByPriority(t *testing.T) {
	d := &Daemon{}
	issues := []Issue{
		{ID: "proj-3", Priority: 2, IssueType: "feature", Status: "open"},
		{ID: "proj-1", Priority: 0, IssueType: "feature", Status: "open"},
		{ID: "proj-2", Priority: 1, IssueType: "feature", Status: "open"},
	}

	sorted, _, err := d.PrioritizeIssues(issues)
	if err != nil {
		t.Fatalf("PrioritizeIssues() error: %v", err)
	}
	if len(sorted) != 3 {
		t.Fatalf("PrioritizeIssues() returned %d issues, want 3", len(sorted))
	}
	if sorted[0].ID != "proj-1" {
		t.Errorf("PrioritizeIssues() first issue = %s, want proj-1", sorted[0].ID)
	}
	if sorted[1].ID != "proj-2" {
		t.Errorf("PrioritizeIssues() second issue = %s, want proj-2", sorted[1].ID)
	}
	if sorted[2].ID != "proj-3" {
		t.Errorf("PrioritizeIssues() third issue = %s, want proj-3", sorted[2].ID)
	}
}

func TestRouteIssueForSpawn_NoHotspotChecker(t *testing.T) {
	d := &Daemon{}
	issue := &Issue{ID: "proj-1", Title: "Test", IssueType: "feature"}
	route, err := d.RouteIssueForSpawn(issue, "feature-impl", "opus")
	if err != nil {
		t.Fatalf("RouteIssueForSpawn() error: %v", err)
	}
	if route.Skill != "feature-impl" {
		t.Errorf("RouteIssueForSpawn() Skill = %q, want %q", route.Skill, "feature-impl")
	}
	if route.Model != "opus" {
		t.Errorf("RouteIssueForSpawn() Model = %q, want %q", route.Model, "opus")
	}
	if route.ExtractionSpawned {
		t.Error("RouteIssueForSpawn() should not spawn extraction without hotspot checker")
	}
	if route.ArchitectEscalated {
		t.Error("RouteIssueForSpawn() should not escalate without hotspot checker")
	}
}

func TestSkillRoute_PassthroughWhenNoHotspot(t *testing.T) {
	// When HotspotChecker returns no hotspots, the route should pass through unchanged
	d := &Daemon{
		HotspotChecker: &mockHotspotChecker{hotspots: nil},
	}
	issue := &Issue{ID: "proj-1", Title: "Test", IssueType: "feature"}
	route, err := d.RouteIssueForSpawn(issue, "feature-impl", "opus")
	if err != nil {
		t.Fatalf("RouteIssueForSpawn() error: %v", err)
	}
	if route.Skill != "feature-impl" {
		t.Errorf("RouteIssueForSpawn() Skill = %q, want %q", route.Skill, "feature-impl")
	}
}
