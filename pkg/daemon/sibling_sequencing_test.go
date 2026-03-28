package daemon

import (
	"fmt"
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

// Helper to build epicChildIDs from a list of issue IDs.
func epicChildren(ids ...string) map[string]bool {
	m := make(map[string]bool)
	for _, id := range ids {
		m[id] = true
	}
	return m
}

func TestShouldDeferTestIssue_DefersWhenImplSiblingsReady(t *testing.T) {
	testIssue := Issue{ID: "scrape-9w3", Title: "Table-driven strategy selection tests", Status: "open"}
	allIssues := []Issue{
		testIssue,
		{ID: "scrape-52p", Title: "Implement GitHub API extraction via gh CLI", Status: "open"},
		{ID: "scrape-gdh", Title: "Integrate vision model for screenshot analysis", Status: "open"},
	}
	// All are epic children — deferral applies
	epic := epicChildren("scrape-9w3", "scrape-52p", "scrape-gdh")

	shouldDefer, reason := ShouldDeferTestIssue(testIssue, allIssues, nil, epic)
	if !shouldDefer {
		t.Error("ShouldDeferTestIssue() should defer when implementation siblings are open epic children")
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
	epic := epicChildren("scrape-9w3", "scrape-52p")

	shouldDefer, _ := ShouldDeferTestIssue(testIssue, allIssues, nil, epic)
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
	epic := epicChildren("scrape-9w3", "scrape-52p")

	shouldDefer, _ := ShouldDeferTestIssue(testIssue, allIssues, nil, epic)
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
	epic := epicChildren("scrape-52p", "scrape-9w3")

	shouldDefer, _ := ShouldDeferTestIssue(implIssue, allIssues, nil, epic)
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
	epic := epicChildren("proj-1", "proj-2")

	shouldDefer, _ := ShouldDeferTestIssue(testIssue, allIssues, nil, epic)
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
	epic := epicChildren("scrape-9w3", "orch-go-abc")

	shouldDefer, _ := ShouldDeferTestIssue(testIssue, allIssues, nil, epic)
	if shouldDefer {
		t.Error("ShouldDeferTestIssue() should NOT defer based on siblings from different projects")
	}
}

func TestShouldDeferTestIssue_NoSiblings(t *testing.T) {
	testIssue := Issue{ID: "proj-1", Title: "Write tests for module A", Status: "open"}
	allIssues := []Issue{testIssue}
	epic := epicChildren("proj-1")

	shouldDefer, _ := ShouldDeferTestIssue(testIssue, allIssues, nil, epic)
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
	epic := epicChildren("proj-1", "proj-2", "proj-3")

	shouldDefer, _ := ShouldDeferTestIssue(testIssue, allIssues, nil, epic)
	if !shouldDefer {
		t.Error("ShouldDeferTestIssue() should defer when ANY implementation sibling is still active")
	}
}

// Integration test: verifies that Decide() skips test issues and selects
// implementation issues first when both are in the ready queue as epic children.
func TestDecide_DefersTestIssueSelectsImpl(t *testing.T) {
	d := &Daemon{
		Issues: &mockIssueQuerier{},
	}
	orient := OrientResult{
		Sense: SenseResult{GateSignal: SpawnGateSignal{Allowed: true}},
		PrioritizedIssues: []Issue{
			// Test issue appears first in priority order
			{ID: "scrape-9w3", Title: "Table-driven strategy selection tests", IssueType: "task", Status: "open"},
			{ID: "scrape-52p", Title: "Implement GitHub API extraction via gh CLI", IssueType: "feature", Status: "open"},
			{ID: "scrape-gdh", Title: "Integrate vision model for screenshot analysis", IssueType: "feature", Status: "open"},
		},
		// All are epic children — sibling sequencing applies
		EpicChildIDs: epicChildren("scrape-9w3", "scrape-52p", "scrape-gdh"),
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
	d := &Daemon{
		Issues: &mockIssueQuerier{},
	}
	orient := OrientResult{
		Sense: SenseResult{GateSignal: SpawnGateSignal{Allowed: true}},
		PrioritizedIssues: []Issue{
			// Only test issues — no implementation siblings
			{ID: "scrape-9w3", Title: "Table-driven strategy selection tests", IssueType: "task", Status: "open"},
			{ID: "scrape-abc", Title: "Add tests for utility module", IssueType: "task", Status: "open"},
		},
		EpicChildIDs: epicChildren("scrape-9w3", "scrape-abc"),
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

// Integration test: verifies that Decide() does not defer test issues when
// the blocking sibling is a ghost issue (exists in ready list but not in beads).
func TestDecide_IgnoresGhostSiblingForDeferral(t *testing.T) {
	d := &Daemon{
		Issues: &mockIssueQuerier{
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				if beadsID == "orch-go-ehz" {
					return "", fmt.Errorf("issue not found: orch-go-ehz")
				}
				return "open", nil
			},
		},
	}
	orient := OrientResult{
		Sense: SenseResult{GateSignal: SpawnGateSignal{Allowed: true}},
		PrioritizedIssues: []Issue{
			// Test issues that should NOT be deferred because the sibling is a ghost
			{ID: "orch-go-kxtrd", Title: "Write tests for extraction pipeline", IssueType: "task", Status: "open"},
			// Ghost sibling — returned by ListReadyIssues but bd show fails
			{ID: "orch-go-ehz", Title: "Implement feature X", IssueType: "feature", Status: "open"},
		},
		EpicChildIDs: epicChildren("orch-go-kxtrd", "orch-go-ehz"),
	}

	decision := d.Decide(orient, nil)
	if !decision.ShouldSpawn {
		t.Fatalf("Decide() ShouldSpawn = false, want true; BlockReason: %s", decision.BlockReason)
	}
	if decision.Issue == nil {
		t.Fatal("Decide() Issue is nil")
	}
	// The test issue should NOT be deferred — the ghost sibling should be ignored
	if decision.Issue.ID != "orch-go-kxtrd" {
		t.Errorf("Decide() selected %s, want orch-go-kxtrd (ghost sibling should not block)", decision.Issue.ID)
	}
}

func TestShouldDeferTestIssue_GhostSiblingIgnored(t *testing.T) {
	// When a sibling exists in the ready list but not in beads (ghost issue),
	// the validator returns false and the test issue should NOT be deferred.
	testIssue := Issue{ID: "orch-go-kxtrd", Title: "Write tests for extraction", Status: "open"}
	allIssues := []Issue{
		testIssue,
		{ID: "orch-go-ehz", Title: "Implement feature X", Status: "open"}, // ghost
	}
	epic := epicChildren("orch-go-kxtrd", "orch-go-ehz")

	// Validator says orch-go-ehz does not exist
	ghostValidator := func(id string) bool {
		return id != "orch-go-ehz"
	}

	shouldDefer, _ := ShouldDeferTestIssue(testIssue, allIssues, ghostValidator, epic)
	if shouldDefer {
		t.Error("ShouldDeferTestIssue() should NOT defer when blocking sibling is a ghost issue")
	}
}

func TestShouldDeferTestIssue_ValidSiblingStillDefers(t *testing.T) {
	// When the validator confirms the sibling exists, deferral should still happen.
	testIssue := Issue{ID: "orch-go-abc", Title: "Write tests for module A", Status: "open"}
	allIssues := []Issue{
		testIssue,
		{ID: "orch-go-def", Title: "Implement module A", Status: "open"},
	}
	epic := epicChildren("orch-go-abc", "orch-go-def")

	// Validator confirms all siblings exist
	allExist := func(id string) bool { return true }

	shouldDefer, _ := ShouldDeferTestIssue(testIssue, allIssues, allExist, epic)
	if !shouldDefer {
		t.Error("ShouldDeferTestIssue() should defer when validator confirms sibling exists")
	}
}

func TestShouldDeferTestIssue_NilValidatorTrustsAllSiblings(t *testing.T) {
	// When validator is nil, trust all siblings (backwards compatible).
	testIssue := Issue{ID: "orch-go-abc", Title: "Write tests for module A", Status: "open"}
	allIssues := []Issue{
		testIssue,
		{ID: "orch-go-def", Title: "Implement module A", Status: "open"},
	}
	epic := epicChildren("orch-go-abc", "orch-go-def")

	shouldDefer, _ := ShouldDeferTestIssue(testIssue, allIssues, nil, epic)
	if !shouldDefer {
		t.Error("ShouldDeferTestIssue() with nil validator should defer (trust all siblings)")
	}
}

func TestShouldDeferTestIssue_GhostSiblingSkippedRealSiblingDefers(t *testing.T) {
	// Mixed: one ghost sibling and one real sibling. Should still defer because
	// the real sibling exists.
	testIssue := Issue{ID: "orch-go-abc", Title: "Write tests for module A", Status: "open"}
	allIssues := []Issue{
		testIssue,
		{ID: "orch-go-ghost", Title: "Implement feature ghost", Status: "open"},
		{ID: "orch-go-real", Title: "Implement feature real", Status: "open"},
	}
	epic := epicChildren("orch-go-abc", "orch-go-ghost", "orch-go-real")

	validator := func(id string) bool { return id != "orch-go-ghost" }

	shouldDefer, reason := ShouldDeferTestIssue(testIssue, allIssues, validator, epic)
	if !shouldDefer {
		t.Error("ShouldDeferTestIssue() should defer when at least one real sibling exists")
	}
	if !strings.Contains(reason, "orch-go-real") {
		t.Errorf("reason should reference real sibling, got: %s", reason)
	}
}

func TestShouldDeferTestIssue_ReasonIncludesSiblingID(t *testing.T) {
	testIssue := Issue{ID: "scrape-9w3", Title: "Table-driven tests", Status: "open"}
	allIssues := []Issue{
		testIssue,
		{ID: "scrape-52p", Title: "Implement GitHub API", Status: "open"},
	}
	epic := epicChildren("scrape-9w3", "scrape-52p")

	_, reason := ShouldDeferTestIssue(testIssue, allIssues, nil, epic)
	if reason == "" {
		t.Fatal("expected non-empty reason")
	}
	if !strings.Contains(reason, "scrape-52p") {
		t.Errorf("reason should include sibling ID, got: %s", reason)
	}
}

// Regression test for orch-go-e4uiq: investigation issues that mention testing
// concepts (e.g., "property-based testing as verification layer") must NOT be
// classified as test-like issues. Investigations produce knowledge, not code —
// deferring them behind implementation siblings is meaningless.
func TestIsTestLikeIssue_InvestigationExempt(t *testing.T) {
	issue := Issue{
		ID:          "orch-go-kxtrd",
		Title:       "Investigate Antithesis Hegel — property-based testing as agent verification layer",
		Description: "Hegel testing libraries wrap Hypothesis. testing as the verification layer for AI agents.",
		IssueType:   "investigation",
		Status:      "open",
	}
	if isTestLikeIssue(issue) {
		t.Error("investigation issue mentioning testing should NOT be classified as test-like")
	}
}

func TestIsTestLikeIssue_QuestionExempt(t *testing.T) {
	issue := Issue{
		ID:          "proj-q1",
		Title:       "Should we add integration tests for the auth module?",
		Description: "Testing strategy question.",
		IssueType:   "question",
		Status:      "open",
	}
	if isTestLikeIssue(issue) {
		t.Error("question issue mentioning testing should NOT be classified as test-like")
	}
}

func TestIsTestLikeIssue_FeatureStillMatches(t *testing.T) {
	issue := Issue{
		ID:        "proj-1",
		Title:     "Add tests for auth module",
		IssueType: "feature",
		Status:    "open",
	}
	if !isTestLikeIssue(issue) {
		t.Error("feature issue about writing tests should still be classified as test-like")
	}
}

func TestShouldDeferTestIssue_InvestigationNotDeferred(t *testing.T) {
	investigation := Issue{
		ID:          "orch-go-kxtrd",
		Title:       "Investigate property-based testing frameworks",
		Description: "Research testing approaches.",
		IssueType:   "investigation",
		Status:      "open",
	}
	allIssues := []Issue{
		investigation,
		{ID: "orch-go-impl", Title: "Implement feature X", IssueType: "feature", Status: "open"},
	}
	epic := epicChildren("orch-go-kxtrd", "orch-go-impl")

	shouldDefer, _ := ShouldDeferTestIssue(investigation, allIssues, nil, epic)
	if shouldDefer {
		t.Error("investigation issue should never be deferred as test-like, even with open impl siblings")
	}
}

// Regression test for orch-go-cn3j0: standalone issues in the same project must
// NOT be treated as siblings. Only epic children should trigger deferral.
func TestShouldDeferTestIssue_StandaloneIssuesNotSiblings(t *testing.T) {
	// Simulates the bug: orch-go-47ppm (test-like) deferred behind orch-go-3zxfz
	// (unrelated bug) just because they share the orch-go prefix.
	testIssue := Issue{ID: "orch-go-47ppm", Title: "Write tests for daemon triage", Status: "open"}
	allIssues := []Issue{
		testIssue,
		{ID: "orch-go-3zxfz", Title: "Re-enable hotspot checker", Status: "open"},
	}
	// Neither is an epic child — they're standalone issues
	noEpic := map[string]bool{}

	shouldDefer, _ := ShouldDeferTestIssue(testIssue, allIssues, nil, noEpic)
	if shouldDefer {
		t.Error("ShouldDeferTestIssue() must NOT defer standalone issues — only epic children are siblings")
	}
}

// Verify that nil epicChildIDs disables deferral entirely.
func TestShouldDeferTestIssue_NilEpicChildIDsNoDeferral(t *testing.T) {
	testIssue := Issue{ID: "proj-1", Title: "Write tests for module A", Status: "open"}
	allIssues := []Issue{
		testIssue,
		{ID: "proj-2", Title: "Implement module A", Status: "open"},
	}

	shouldDefer, _ := ShouldDeferTestIssue(testIssue, allIssues, nil, nil)
	if shouldDefer {
		t.Error("ShouldDeferTestIssue() with nil epicChildIDs should never defer")
	}
}

// Test that a test issue which IS an epic child is not deferred when the impl
// sibling is NOT an epic child (different origin).
func TestShouldDeferTestIssue_OnlyTestIsEpicChild(t *testing.T) {
	testIssue := Issue{ID: "proj-1", Title: "Write tests for module A", Status: "open"}
	allIssues := []Issue{
		testIssue,
		{ID: "proj-2", Title: "Implement module A", Status: "open"},
	}
	// Only the test is an epic child, not the impl sibling
	epic := epicChildren("proj-1")

	shouldDefer, _ := ShouldDeferTestIssue(testIssue, allIssues, nil, epic)
	if shouldDefer {
		t.Error("ShouldDeferTestIssue() should NOT defer when impl sibling is not an epic child")
	}
}

// Decide integration: standalone issues in the same project must not trigger deferral.
func TestDecide_StandaloneIssuesNotDeferred(t *testing.T) {
	d := &Daemon{
		Issues: &mockIssueQuerier{},
	}
	orient := OrientResult{
		Sense: SenseResult{GateSignal: SpawnGateSignal{Allowed: true}},
		PrioritizedIssues: []Issue{
			// Test-like issue first in priority
			{ID: "orch-go-47ppm", Title: "Write tests for daemon triage", IssueType: "task", Status: "open"},
			// Unrelated bug in same project
			{ID: "orch-go-3zxfz", Title: "Re-enable hotspot checker", IssueType: "bug", Status: "open"},
		},
		// Neither is an epic child — standalone issues
		EpicChildIDs: map[string]bool{},
	}

	decision := d.Decide(orient, nil)
	if !decision.ShouldSpawn {
		t.Fatalf("Decide() ShouldSpawn = false, want true; BlockReason: %s", decision.BlockReason)
	}
	// Test issue should NOT be deferred — it's a standalone issue, not an epic sibling
	if decision.Issue.ID != "orch-go-47ppm" {
		t.Errorf("Decide() selected %s, want orch-go-47ppm (standalone issues should not be deferred)", decision.Issue.ID)
	}
}
