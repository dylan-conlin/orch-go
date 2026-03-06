package debrief

import (
	"strings"
	"testing"
)

func TestCollectFrictionSummary(t *testing.T) {
	inputs := []FrictionSummaryInput{
		{BeadsID: "orch-go-abc1", Category: "bug", Description: "beads dir resolution fails"},
		{BeadsID: "orch-go-abc1", Category: "tooling", Description: "bd sync error noise"},
		{BeadsID: "orch-go-def2", Category: "bug", Description: "git hook false positive"},
		{BeadsID: "orch-go-ghi3", Category: "ceremony", Description: "12 line fix took 30min"},
	}

	categories := CollectFrictionSummary(inputs)
	if len(categories) != 3 {
		t.Fatalf("expected 3 categories, got %d", len(categories))
	}

	// Should be sorted by count descending — bug has 2
	if categories[0].Category != "bug" {
		t.Errorf("expected first category 'bug', got %q", categories[0].Category)
	}
	if categories[0].Count != 2 {
		t.Errorf("expected bug count 2, got %d", categories[0].Count)
	}
	if len(categories[0].Descriptions) != 2 {
		t.Errorf("expected 2 bug descriptions, got %d", len(categories[0].Descriptions))
	}
}

func TestCollectFrictionSummaryEmpty(t *testing.T) {
	categories := CollectFrictionSummary(nil)
	if categories != nil {
		t.Errorf("expected nil for empty inputs, got %d", len(categories))
	}
}

func TestCollectFrictionSummaryDeduplicatesDescriptions(t *testing.T) {
	inputs := []FrictionSummaryInput{
		{BeadsID: "a", Category: "bug", Description: "same issue"},
		{BeadsID: "b", Category: "bug", Description: "same issue"},
	}

	categories := CollectFrictionSummary(inputs)
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}
	if categories[0].Count != 2 {
		t.Errorf("expected count 2, got %d", categories[0].Count)
	}
	if len(categories[0].Descriptions) != 1 {
		t.Errorf("expected 1 unique description, got %d", len(categories[0].Descriptions))
	}
}

func TestFormatFrictionSummary(t *testing.T) {
	categories := []FrictionCategory{
		{Category: "bug", Count: 2, Descriptions: []string{"beads resolution", "hook false positive"}},
		{Category: "tooling", Count: 1, Descriptions: []string{"bd sync noise"}},
	}

	lines := FormatFrictionSummary(categories)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	// Multi-description uses count format
	if !strings.Contains(lines[0], "bug") {
		t.Errorf("expected 'bug' in line, got: %s", lines[0])
	}
	if !strings.Contains(lines[0], "2") {
		t.Errorf("expected count in line, got: %s", lines[0])
	}

	// Single description uses inline format
	if !strings.Contains(lines[1], "bd sync noise") {
		t.Errorf("expected description in line, got: %s", lines[1])
	}
}

func TestFormatFrictionSummaryEmpty(t *testing.T) {
	lines := FormatFrictionSummary(nil)
	if lines != nil {
		t.Errorf("expected nil for empty categories, got %d lines", len(lines))
	}
}

func TestFormatFrictionSummaryNoDescription(t *testing.T) {
	categories := []FrictionCategory{
		{Category: "tooling", Count: 3, Descriptions: nil},
	}

	lines := FormatFrictionSummary(categories)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "3 report(s)") {
		t.Errorf("expected count-only format, got: %s", lines[0])
	}
}
