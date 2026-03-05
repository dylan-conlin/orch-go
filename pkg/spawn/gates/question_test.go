package gates

import (
	"fmt"
	"testing"
)

func TestCheckOpenQuestions_NilChecker(t *testing.T) {
	result, err := CheckOpenQuestions("issue-1", false, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}
}

func TestCheckOpenQuestions_NoIssueID(t *testing.T) {
	checker := func(issueID string) (*OpenQuestionResult, error) {
		t.Fatal("checker should not be called with empty issueID")
		return nil, nil
	}
	result, err := CheckOpenQuestions("", false, checker)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}
}

func TestCheckOpenQuestions_NoOpenQuestions(t *testing.T) {
	checker := func(issueID string) (*OpenQuestionResult, error) {
		return &OpenQuestionResult{}, nil
	}
	result, err := CheckOpenQuestions("issue-1", false, checker)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil || result.HasOpenQuestions() {
		t.Fatal("expected result with no open questions")
	}
}

func TestCheckOpenQuestions_DirectQuestion(t *testing.T) {
	checker := func(issueID string) (*OpenQuestionResult, error) {
		return &OpenQuestionResult{
			Questions: []OpenQuestion{
				{
					IssueID: "q-1",
					Title:   "Should we use JWT or sessions?",
					Status:  "open",
					Path:    []string{"issue-1", "q-1"},
				},
			},
		}, nil
	}
	result, err := CheckOpenQuestions("issue-1", false, checker)
	if err != nil {
		t.Fatalf("expected no error (warning only), got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.HasOpenQuestions() {
		t.Fatal("expected open questions")
	}
	if len(result.Questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(result.Questions))
	}
	if result.Questions[0].IssueID != "q-1" {
		t.Fatalf("expected question q-1, got %s", result.Questions[0].IssueID)
	}
}

func TestCheckOpenQuestions_TransitiveQuestion(t *testing.T) {
	checker := func(issueID string) (*OpenQuestionResult, error) {
		return &OpenQuestionResult{
			Questions: []OpenQuestion{
				{
					IssueID: "q-deep",
					Title:   "What auth backend?",
					Status:  "open",
					Path:    []string{"issue-1", "blocker-1", "q-deep"},
				},
			},
		}, nil
	}
	result, err := CheckOpenQuestions("issue-1", false, checker)
	if err != nil {
		t.Fatalf("expected no error (warning only), got %v", err)
	}
	if result == nil || !result.HasOpenQuestions() {
		t.Fatal("expected open questions")
	}
	if len(result.Questions[0].Path) != 3 {
		t.Fatalf("expected path length 3, got %d", len(result.Questions[0].Path))
	}
}

func TestCheckOpenQuestions_DaemonSilent(t *testing.T) {
	checker := func(issueID string) (*OpenQuestionResult, error) {
		return &OpenQuestionResult{
			Questions: []OpenQuestion{
				{IssueID: "q-1", Title: "Question?", Status: "open", Path: []string{"issue-1", "q-1"}},
			},
		}, nil
	}
	// Daemon-driven should still return result but not print warnings
	result, err := CheckOpenQuestions("issue-1", true, checker)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil || !result.HasOpenQuestions() {
		t.Fatal("expected result with questions even for daemon")
	}
}

func TestCheckOpenQuestions_CheckerError(t *testing.T) {
	checker := func(issueID string) (*OpenQuestionResult, error) {
		return nil, fmt.Errorf("beads connection failed")
	}
	// Infrastructure errors should not block spawn
	result, err := CheckOpenQuestions("issue-1", false, checker)
	if err != nil {
		t.Fatalf("expected no error on checker failure, got %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result on checker failure, got %+v", result)
	}
}

func TestOpenQuestionResult_HasOpenQuestions(t *testing.T) {
	empty := &OpenQuestionResult{}
	if empty.HasOpenQuestions() {
		t.Fatal("empty result should not have open questions")
	}

	withQuestions := &OpenQuestionResult{
		Questions: []OpenQuestion{{IssueID: "q-1"}},
	}
	if !withQuestions.HasOpenQuestions() {
		t.Fatal("result with questions should report HasOpenQuestions")
	}
}

func TestBuildOpenQuestionChecker_WalksTransitiveDeps(t *testing.T) {
	// Simulate: issue-1 -> dep-1 (task, open) -> q-1 (question, open)
	issues := map[string]*mockIssue{
		"issue-1": {
			id: "issue-1", issueType: "task", status: "open",
			deps: []mockDep{{id: "dep-1", depType: "blocks", status: "open"}},
		},
		"dep-1": {
			id: "dep-1", issueType: "task", status: "open",
			deps: []mockDep{{id: "q-1", depType: "blocks", status: "open"}},
		},
		"q-1": {
			id: "q-1", issueType: "question", status: "open", title: "Which approach?",
			deps: nil,
		},
	}

	checker := buildTestChecker(issues)
	result, err := checker("issue-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || !result.HasOpenQuestions() {
		t.Fatal("expected to find transitive open question")
	}
	if len(result.Questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(result.Questions))
	}
	q := result.Questions[0]
	if q.IssueID != "q-1" {
		t.Fatalf("expected q-1, got %s", q.IssueID)
	}
	// Path should be issue-1 -> dep-1 -> q-1
	if len(q.Path) != 3 {
		t.Fatalf("expected path length 3, got %d: %v", len(q.Path), q.Path)
	}
}

func TestBuildOpenQuestionChecker_ClosedQuestionIgnored(t *testing.T) {
	issues := map[string]*mockIssue{
		"issue-1": {
			id: "issue-1", issueType: "task", status: "open",
			deps: []mockDep{{id: "q-1", depType: "blocks", status: "closed"}},
		},
		"q-1": {
			id: "q-1", issueType: "question", status: "closed", title: "Answered",
			deps: nil,
		},
	}

	checker := buildTestChecker(issues)
	result, err := checker("issue-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil && result.HasOpenQuestions() {
		t.Fatal("closed questions should not be reported")
	}
}

func TestBuildOpenQuestionChecker_AnsweredQuestionIgnored(t *testing.T) {
	issues := map[string]*mockIssue{
		"issue-1": {
			id: "issue-1", issueType: "task", status: "open",
			deps: []mockDep{{id: "q-1", depType: "blocks", status: "answered"}},
		},
		"q-1": {
			id: "q-1", issueType: "question", status: "answered", title: "Resolved",
			deps: nil,
		},
	}

	checker := buildTestChecker(issues)
	result, err := checker("issue-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil && result.HasOpenQuestions() {
		t.Fatal("answered questions should not be reported")
	}
}

func TestBuildOpenQuestionChecker_ParentChildIgnored(t *testing.T) {
	issues := map[string]*mockIssue{
		"issue-1": {
			id: "issue-1", issueType: "task", status: "open",
			deps: []mockDep{{id: "q-1", depType: "parent-child", status: "open"}},
		},
		"q-1": {
			id: "q-1", issueType: "question", status: "open", title: "Not blocking",
			deps: nil,
		},
	}

	checker := buildTestChecker(issues)
	result, err := checker("issue-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil && result.HasOpenQuestions() {
		t.Fatal("parent-child deps should not be walked for blocking questions")
	}
}

func TestBuildOpenQuestionChecker_CycleProtection(t *testing.T) {
	issues := map[string]*mockIssue{
		"issue-1": {
			id: "issue-1", issueType: "task", status: "open",
			deps: []mockDep{{id: "issue-2", depType: "blocks", status: "open"}},
		},
		"issue-2": {
			id: "issue-2", issueType: "task", status: "open",
			deps: []mockDep{{id: "issue-1", depType: "blocks", status: "open"}},
		},
	}

	checker := buildTestChecker(issues)
	result, err := checker("issue-1")
	if err != nil {
		t.Fatalf("expected no error on cycle, got %v", err)
	}
	// Should complete without infinite loop
	if result != nil && result.HasOpenQuestions() {
		t.Fatal("cycle with no questions should not report questions")
	}
}

func TestBuildOpenQuestionChecker_DepthLimit(t *testing.T) {
	// Build a chain of 15 issues deep with a question at the end
	issues := make(map[string]*mockIssue)
	for i := 0; i < 15; i++ {
		id := fmt.Sprintf("issue-%d", i)
		nextID := fmt.Sprintf("issue-%d", i+1)
		issues[id] = &mockIssue{
			id: id, issueType: "task", status: "open",
			deps: []mockDep{{id: nextID, depType: "blocks", status: "open"}},
		}
	}
	issues["issue-15"] = &mockIssue{
		id: "issue-15", issueType: "question", status: "open", title: "Deep question",
		deps: nil,
	}

	checker := buildTestChecker(issues)
	result, err := checker("issue-0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Depth limit is 10, so question at depth 15 should not be found
	if result != nil && result.HasOpenQuestions() {
		t.Fatal("questions beyond depth limit should not be found")
	}
}

// --- Test helpers ---

type mockDep struct {
	id      string
	depType string
	status  string
}

type mockIssue struct {
	id        string
	issueType string
	status    string
	title     string
	deps      []mockDep
}

// buildTestChecker creates an OpenQuestionChecker that uses mock issue data
// and the real WalkTransitiveDeps logic.
func buildTestChecker(issues map[string]*mockIssue) OpenQuestionChecker {
	fetcher := func(issueID string) (*IssueSummary, error) {
		mi, ok := issues[issueID]
		if !ok {
			return nil, fmt.Errorf("issue not found: %s", issueID)
		}
		var deps []DepSummary
		for _, d := range mi.deps {
			deps = append(deps, DepSummary{
				ID:     d.id,
				Type:   d.depType,
				Status: d.status,
			})
		}
		return &IssueSummary{
			ID:        mi.id,
			IssueType: mi.issueType,
			Status:    mi.status,
			Title:     mi.title,
			Deps:      deps,
		}, nil
	}
	return BuildOpenQuestionChecker(fetcher)
}
