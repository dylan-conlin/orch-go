package daemon

import (
	"testing"
)

func TestTitleSimilarity(t *testing.T) {
	tests := []struct {
		a, b       string
		wantAbove  float64
		wantBelow  float64
	}{
		// Identical titles
		{"fix auth flow", "fix auth flow", 1.0, 1.01},
		// Near-duplicates
		{"fix: OAuth token refresh fails", "fix: OAuth token refresh failure", 0.6, 1.01},
		// Different
		{"add dashboard widget", "fix auth flow", 0.0, 0.3},
		// Empty
		{"", "", 0.0, 0.01},
		{"something", "", 0.0, 0.01},
	}
	for _, tt := range tests {
		sim := TitleSimilarity(tt.a, tt.b)
		if sim < tt.wantAbove || sim > tt.wantBelow {
			t.Errorf("TitleSimilarity(%q, %q) = %f, want in [%f, %f]",
				tt.a, tt.b, sim, tt.wantAbove, tt.wantBelow)
		}
	}
}

func TestComputeWorkGraph_TitleSimilarity(t *testing.T) {
	issues := []Issue{
		{ID: "proj-1", Title: "fix OAuth token refresh fails on retry", IssueType: "bug", Status: "open"},
		{ID: "proj-2", Title: "fix OAuth token refresh failure on retry", IssueType: "bug", Status: "open"},
		{ID: "proj-3", Title: "add dashboard widget for metrics", IssueType: "feature", Status: "open"},
	}

	graph := ComputeWorkGraph(issues, nil)

	// Should detect near-duplicate between proj-1 and proj-2
	found := false
	for _, dup := range graph.TitleDuplicates {
		if (dup.IssueA == "proj-1" && dup.IssueB == "proj-2") ||
			(dup.IssueA == "proj-2" && dup.IssueB == "proj-1") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected title duplicate between proj-1 and proj-2, got duplicates: %+v", graph.TitleDuplicates)
	}

	// proj-3 should not be a duplicate of either
	for _, dup := range graph.TitleDuplicates {
		if dup.IssueA == "proj-3" || dup.IssueB == "proj-3" {
			t.Errorf("unexpected duplicate involving proj-3: %+v", dup)
		}
	}
}

func TestComputeWorkGraph_InvestigationChain(t *testing.T) {
	issues := []Issue{
		{ID: "proj-1", Title: "investigate auth token handling", IssueType: "task",
			Description: "follow-up from proj-99 investigation findings"},
		{ID: "proj-2", Title: "add metrics dashboard", IssueType: "feature",
			Description: "new feature for monitoring"},
		{ID: "proj-3", Title: "debug OAuth flow", IssueType: "bug",
			Description: "see investigation proj-1 and prior work in proj-99"},
	}

	graph := ComputeWorkGraph(issues, nil)

	// proj-1 references proj-99
	found99 := false
	// proj-3 references proj-1 and proj-99
	foundChain := false
	for _, chain := range graph.InvestigationChains {
		if chain.IssueID == "proj-1" {
			for _, ref := range chain.ReferencedIssues {
				if ref == "proj-99" {
					found99 = true
				}
			}
		}
		if chain.IssueID == "proj-3" {
			foundChain = true
			if len(chain.ReferencedIssues) < 2 {
				t.Errorf("expected proj-3 to reference at least 2 issues, got %d", len(chain.ReferencedIssues))
			}
		}
	}
	if !found99 {
		t.Errorf("expected proj-1 to reference proj-99, got chains: %+v", graph.InvestigationChains)
	}
	if !foundChain {
		t.Errorf("expected proj-3 investigation chain, got chains: %+v", graph.InvestigationChains)
	}
}

func TestComputeWorkGraph_FileOverlap(t *testing.T) {
	issues := []Issue{
		{ID: "proj-1", Title: "fix bug in pkg/daemon/daemon.go", IssueType: "bug", Status: "open",
			Description: "The bug is in pkg/daemon/daemon.go and also affects pkg/daemon/coordination.go"},
		{ID: "proj-2", Title: "refactor daemon coordination", IssueType: "task", Status: "open",
			Description: "Refactor pkg/daemon/coordination.go to separate concerns"},
		{ID: "proj-3", Title: "add new API endpoint", IssueType: "feature", Status: "open",
			Description: "Add endpoint in cmd/orch/serve.go"},
	}

	graph := ComputeWorkGraph(issues, nil)

	// proj-1 and proj-2 share coordination.go
	found := false
	for _, overlap := range graph.FileOverlaps {
		if (overlap.IssueA == "proj-1" && overlap.IssueB == "proj-2") ||
			(overlap.IssueA == "proj-2" && overlap.IssueB == "proj-1") {
			found = true
			if len(overlap.SharedFiles) == 0 {
				t.Error("expected shared files between proj-1 and proj-2")
			}
			break
		}
	}
	if !found {
		t.Errorf("expected file overlap between proj-1 and proj-2, got overlaps: %+v", graph.FileOverlaps)
	}
}

func TestComputeWorkGraph_Empty(t *testing.T) {
	graph := ComputeWorkGraph(nil, nil)
	if len(graph.TitleDuplicates) != 0 {
		t.Errorf("expected 0 title duplicates, got %d", len(graph.TitleDuplicates))
	}
	if len(graph.FileOverlaps) != 0 {
		t.Errorf("expected 0 file overlaps, got %d", len(graph.FileOverlaps))
	}
	if len(graph.InvestigationChains) != 0 {
		t.Errorf("expected 0 investigation chains, got %d", len(graph.InvestigationChains))
	}
}

func TestComputeWorkGraph_RecentCompletions(t *testing.T) {
	issues := []Issue{
		{ID: "proj-5", Title: "fix OAuth token refresh", IssueType: "bug", Status: "open"},
	}
	// A recently completed issue had a very similar title
	recentCompletions := []Issue{
		{ID: "proj-2", Title: "fix OAuth token refresh bug", IssueType: "bug", Status: "closed"},
	}

	graph := ComputeWorkGraph(issues, recentCompletions)

	// Should detect similarity with the recently completed issue
	found := false
	for _, dup := range graph.TitleDuplicates {
		if (dup.IssueA == "proj-5" && dup.IssueB == "proj-2") ||
			(dup.IssueA == "proj-2" && dup.IssueB == "proj-5") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected title duplicate between open proj-5 and completed proj-2, got: %+v", graph.TitleDuplicates)
	}
}

func TestRemovalCandidates(t *testing.T) {
	graph := WorkGraph{
		TitleDuplicates: []TitleDuplicate{
			{IssueA: "proj-1", IssueB: "proj-2", Similarity: 0.85},
		},
		FileOverlaps: []FileOverlap{
			{IssueA: "proj-1", IssueB: "proj-3", SharedFiles: []string{"pkg/daemon/daemon.go"}},
		},
	}

	candidates := graph.RemovalCandidates()
	if len(candidates) < 1 {
		t.Errorf("expected at least 1 removal candidate, got %d", len(candidates))
	}

	// Title duplicates should always be candidates
	found := false
	for _, c := range candidates {
		if c.Reason == "title_duplicate" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a title_duplicate removal candidate")
	}
}
