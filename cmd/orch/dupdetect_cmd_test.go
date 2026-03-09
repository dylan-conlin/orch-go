package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/dupdetect"
)

func TestDupdetectReportToBeads_Integration(t *testing.T) {
	// Verify the full pipeline: detect -> report -> issue creation
	mock := beads.NewMockClient()

	pairs := []dupdetect.DupPair{
		{
			FuncA: dupdetect.FuncInfo{
				Name: "handleAgentlog", File: "serve_agents_events.go",
				StartLine: 10, Lines: 14,
			},
			FuncB: dupdetect.FuncInfo{
				Name: "handleServiceEvents", File: "serve_services_events.go",
				StartLine: 5, Lines: 14,
			},
			Similarity: 1.0,
		},
	}

	result, err := dupdetect.ReportToBeads(mock, pairs, dupdetect.ReportConfig{
		Threshold: 0.80,
	})
	if err != nil {
		t.Fatalf("ReportToBeads failed: %v", err)
	}

	if result.Created != 1 {
		t.Fatalf("expected 1 created, got %d", result.Created)
	}

	// Verify issue content
	issue, err := mock.Show(result.IssueIDs[0])
	if err != nil {
		t.Fatalf("Show failed: %v", err)
	}

	expectedTitle := "Extract shared logic: handleAgentlog / handleServiceEvents (100% similar)"
	if issue.Title != expectedTitle {
		t.Errorf("title = %q, want %q", issue.Title, expectedTitle)
	}

	// Verify labels
	labelSet := make(map[string]bool)
	for _, l := range issue.Labels {
		labelSet[l] = true
	}
	if !labelSet["dupdetect"] {
		t.Error("missing dupdetect label")
	}
	if !labelSet["triage:review"] {
		t.Error("missing triage:review label")
	}
}

func TestDupdetectTitleDeterminism(t *testing.T) {
	// Same pair in different order should produce same title
	pair := dupdetect.DupPair{
		FuncA:      dupdetect.FuncInfo{Name: "zebra"},
		FuncB:      dupdetect.FuncInfo{Name: "alpha"},
		Similarity: 0.95,
	}
	title := dupdetect.DupPairTitle(pair)

	reversed := dupdetect.DupPair{
		FuncA:      dupdetect.FuncInfo{Name: "alpha"},
		FuncB:      dupdetect.FuncInfo{Name: "zebra"},
		Similarity: 0.95,
	}
	reversedTitle := dupdetect.DupPairTitle(reversed)

	if title != reversedTitle {
		t.Errorf("titles differ:\n  %q\n  %q", title, reversedTitle)
	}
}
