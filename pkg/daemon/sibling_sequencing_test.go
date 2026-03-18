package daemon

import (
	"strings"
	"testing"
)

func TestIsTestLikeIssue_TitlePatterns(t *testing.T) {
	tests := []struct {
		title       string
		description string
		want        bool
	}{
		{"Table-driven strategy selection tests", "", true},
		{"Write tests for GitHub extraction", "", true},
		{"Add tests for vision pipeline", "", true},
		{"Test coverage for API module", "", true},
		{"Unit test the parser", "", true},
		{"Integration test for auth flow", "", true},
		{"Implement GitHub API extraction via gh CLI", "", false},
		{"Integrate vision model for screenshot analysis", "", false},
		{"Fix null pointer in daemon loop", "", false},
		{"Refactor spawn pipeline", "", false},
		// Description-based detection
		{"Improve API module", "add tests for the new endpoints", true},
		{"Improve API module", "refactor the handler code", false},
		// Test-driven in description
		{"Build feature X", "use test-driven development approach", true},
	}

	for _, tt := range tests {
		issue := Issue{Title: tt.title, Description: tt.description}
		got := isTestLikeIssue(issue)
		if got != tt.want {
			t.Errorf("isTestLikeIssue(%q, %q) = %v, want %v", tt.title, tt.description, got, tt.want)
		}
	}
}

func TestShouldDeferTestIssue_DefersWhenImplSiblingsReady(t *testing.T) {
	testIssue := Issue{ID: "scrape-9w3", Title: "Table-driven strategy selection tests", Status: "open"}
	allIssues := []Issue{
		testIssue,
		{ID: "scrape-52p", Title: "Implement GitHub API extraction via gh CLI", Status: "open"},
		{ID: "scrape-gdh", Title: "Integrate vision model for screenshot analysis", Status: "open"},
	}

	shouldDefer, reason := ShouldDeferTestIssue(testIssue, allIssues)
	if !shouldDefer {
		t.Error("ShouldDeferTestIssue() should defer when implementation siblings are open")
	}
	if reason == "" {
		t.Error("ShouldDeferTestIssue() should provide a reason when deferring")
	}
}

func TestShouldDeferTestIssue_DefersWhenImplSiblingsInProgress(t *testing.T) {
	testIssue := Issue{ID: "scrape-9w3", Title: "Write tests for extraction pipeline", Status: "open"}
	allIssues := []Issue{
		testIssue,
		{ID: "scrape-52p", Title: "Implement GitHub API extraction", Status: "in_progress"},
	}

	shouldDefer, _ := ShouldDeferTestIssue(testIssue, allIssues)
	if !shouldDefer {
		t.Error("ShouldDeferTestIssue() should defer when implementation siblings are in_progress")
	}
}

func TestShouldDeferTestIssue_NoDefferWhenImplSiblingsClosed(t *testing.T) {
	testIssue := Issue{ID: "scrape-9w3", Title: "Write tests for extraction pipeline", Status: "open"}
	allIssues := []Issue{
		testIssue,
		{ID: "scrape-52p", Title: "Implement GitHub API extraction", Status: "closed"},
	}

	shouldDefer, _ := ShouldDeferTestIssue(testIssue, allIssues)
	if shouldDefer {
		t.Error("ShouldDeferTestIssue() should NOT defer when all implementation siblings are closed")
	}
}

func TestShouldDeferTestIssue_NoDefferForNonTestIssue(t *testing.T) {
	implIssue := Issue{ID: "scrape-52p", Title: "Implement GitHub API extraction", Status: "open"}
	allIssues := []Issue{
		implIssue,
		{ID: "scrape-9w3", Title: "Write tests for extraction pipeline", Status: "open"},
	}

	shouldDefer, _ := ShouldDeferTestIssue(implIssue, allIssues)
	if shouldDefer {
		t.Error("ShouldDeferTestIssue() should NOT defer non-test issues")
	}
}

func TestShouldDeferTestIssue_NoDefferWhenAllSiblingsAreTests(t *testing.T) {
	testIssue := Issue{ID: "proj-1", Title: "Write tests for module A", Status: "open"}
	allIssues := []Issue{
		testIssue,
		{ID: "proj-2", Title: "Add tests for module B", Status: "open"},
	}

	shouldDefer, _ := ShouldDeferTestIssue(testIssue, allIssues)
	if shouldDefer {
		t.Error("ShouldDeferTestIssue() should NOT defer when all siblings are also test-like")
	}
}

func TestShouldDeferTestIssue_DifferentProjectSiblingsIgnored(t *testing.T) {
	testIssue := Issue{ID: "scrape-9w3", Title: "Write tests for extraction", Status: "open"}
	allIssues := []Issue{
		testIssue,
		// Different project — should not cause deferral
		{ID: "orch-go-abc", Title: "Implement new feature", Status: "open"},
	}

	shouldDefer, _ := ShouldDeferTestIssue(testIssue, allIssues)
	if shouldDefer {
		t.Error("ShouldDeferTestIssue() should NOT defer based on siblings from different projects")
	}
}

func TestShouldDeferTestIssue_NoSiblings(t *testing.T) {
	testIssue := Issue{ID: "proj-1", Title: "Write tests for module A", Status: "open"}
	allIssues := []Issue{testIssue}

	shouldDefer, _ := ShouldDeferTestIssue(testIssue, allIssues)
	if shouldDefer {
		t.Error("ShouldDeferTestIssue() should NOT defer when no siblings exist")
	}
}

func TestShouldDeferTestIssue_MixedStatusSiblings(t *testing.T) {
	// One impl sibling closed, one still open — should still defer
	testIssue := Issue{ID: "proj-1", Title: "Write tests for extraction", Status: "open"}
	allIssues := []Issue{
		testIssue,
		{ID: "proj-2", Title: "Implement module A", Status: "closed"},
		{ID: "proj-3", Title: "Implement module B", Status: "in_progress"},
	}

	shouldDefer, _ := ShouldDeferTestIssue(testIssue, allIssues)
	if !shouldDefer {
		t.Error("ShouldDeferTestIssue() should defer when ANY implementation sibling is still active")
	}
}

// Integration test: verifies that Decide() skips test issues and selects
// implementation issues first when both are in the ready queue.
func TestDecide_DefersTestIssueSelectsImpl(t *testing.T) {
	d := &Daemon{}
	orient := OrientResult{
		Sense: SenseResult{GateSignal: SpawnGateSignal{Allowed: true}},
		PrioritizedIssues: []Issue{
			// Test issue appears first in priority order
			{ID: "scrape-9w3", Title: "Table-driven strategy selection tests", IssueType: "task", Status: "open"},
			{ID: "scrape-52p", Title: "Implement GitHub API extraction via gh CLI", IssueType: "feature", Status: "open"},
			{ID: "scrape-gdh", Title: "Integrate vision model for screenshot analysis", IssueType: "feature", Status: "open"},
		},
		EpicChildIDs: make(map[string]bool),
	}

	decision := d.Decide(orient, nil)
	if !decision.ShouldSpawn {
		t.Fatalf("Decide() ShouldSpawn = false, want true; BlockReason: %s", decision.BlockReason)
	}
	if decision.Issue == nil {
		t.Fatal("Decide() Issue is nil")
	}
	// Should select an implementation issue, not the test issue
	if decision.Issue.ID == "scrape-9w3" {
		t.Error("Decide() selected test issue scrape-9w3 instead of deferring it — should select implementation sibling first")
	}
}

// Integration test: verifies test issues are spawned when no impl siblings remain.
func TestDecide_SpawnsTestWhenNoImplSiblings(t *testing.T) {
	d := &Daemon{}
	orient := OrientResult{
		Sense: SenseResult{GateSignal: SpawnGateSignal{Allowed: true}},
		PrioritizedIssues: []Issue{
			// Only test issues — no implementation siblings
			{ID: "scrape-9w3", Title: "Table-driven strategy selection tests", IssueType: "task", Status: "open"},
			{ID: "scrape-abc", Title: "Add tests for utility module", IssueType: "task", Status: "open"},
		},
		EpicChildIDs: make(map[string]bool),
	}

	decision := d.Decide(orient, nil)
	if !decision.ShouldSpawn {
		t.Fatalf("Decide() ShouldSpawn = false, want true; BlockReason: %s", decision.BlockReason)
	}
	// Should spawn the test issue since no impl siblings exist
	if decision.Issue.ID != "scrape-9w3" {
		t.Errorf("Decide() selected %s, want scrape-9w3 (first test issue when no impl siblings)", decision.Issue.ID)
	}
}

func TestShouldDeferTestIssue_ReasonIncludesSiblingID(t *testing.T) {
	testIssue := Issue{ID: "scrape-9w3", Title: "Table-driven tests", Status: "open"}
	allIssues := []Issue{
		testIssue,
		{ID: "scrape-52p", Title: "Implement GitHub API", Status: "open"},
	}

	_, reason := ShouldDeferTestIssue(testIssue, allIssues)
	if reason == "" {
		t.Fatal("expected non-empty reason")
	}
	if !strings.Contains(reason, "scrape-52p") {
		t.Errorf("reason should include sibling ID, got: %s", reason)
	}
}
