package main

import "testing"

func TestFilterIssuesToParentAndDescendants(t *testing.T) {
	issues := []beadsIssue{
		{ID: "orch-go-100"},
		{ID: "orch-go-100.1"},
		{ID: "orch-go-100.1.1"},
		{ID: "orch-go-1000"},
		{ID: "orch-go-200"},
	}

	filtered := filterIssuesToParentAndDescendants(issues, "orch-go-100")
	if len(filtered) != 3 {
		t.Fatalf("expected 3 filtered issues, got %d", len(filtered))
	}

	want := map[string]bool{
		"orch-go-100":     true,
		"orch-go-100.1":   true,
		"orch-go-100.1.1": true,
	}
	for _, issue := range filtered {
		if !want[issue.ID] {
			t.Fatalf("unexpected filtered issue: %s", issue.ID)
		}
	}
}

func TestFilterToParentAndDescendantsEdges(t *testing.T) {
	nodes := []GraphNode{
		{ID: "orch-go-100"},
		{ID: "orch-go-100.1"},
		{ID: "orch-go-100.2"},
		{ID: "orch-go-200"},
	}
	edges := []GraphEdge{
		{From: "orch-go-100.1", To: "orch-go-100", Type: "parent-child"},
		{From: "orch-go-100.2", To: "orch-go-100", Type: "parent-child"},
		{From: "orch-go-100.1", To: "orch-go-200", Type: "blocks"},
	}

	filteredNodes, filteredEdges := filterToParentAndDescendants(nodes, edges, "orch-go-100")
	if len(filteredNodes) != 3 {
		t.Fatalf("expected 3 filtered nodes, got %d", len(filteredNodes))
	}
	if len(filteredEdges) != 2 {
		t.Fatalf("expected 2 filtered edges, got %d", len(filteredEdges))
	}
}
