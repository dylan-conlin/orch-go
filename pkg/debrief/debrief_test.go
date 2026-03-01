package debrief

import (
	"strings"
	"testing"
	"time"
)

func TestCollectWhatHappened(t *testing.T) {
	events := []SessionEvent{
		{Type: "agent.completed", Timestamp: time.Now().Unix(), Data: map[string]interface{}{"beads_id": "orch-go-abc1", "skill": "feature-impl"}},
		{Type: "session.spawned", Timestamp: time.Now().Unix(), Data: map[string]interface{}{"title": "investigate X"}},
		{Type: "agent.abandoned", Timestamp: time.Now().Unix(), Data: map[string]interface{}{"beads_id": "orch-go-def2", "reason": "stuck in loop"}},
	}

	lines := CollectWhatHappened(events)
	if len(lines) == 0 {
		t.Fatal("expected non-empty What Happened lines")
	}

	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "Completed") {
		t.Errorf("expected completion line, got: %s", joined)
	}
	if !strings.Contains(joined, "Spawned") {
		t.Errorf("expected spawn line, got: %s", joined)
	}
	if !strings.Contains(joined, "Abandoned") {
		t.Errorf("expected abandon line, got: %s", joined)
	}
}

func TestCollectWhatHappenedEmpty(t *testing.T) {
	lines := CollectWhatHappened(nil)
	if len(lines) != 0 {
		t.Errorf("expected empty lines for nil events, got %d", len(lines))
	}
}

func TestCollectInFlight(t *testing.T) {
	issues := []InFlightIssue{
		{ID: "orch-go-abc1", Title: "implement feature X", Status: "in_progress"},
		{ID: "orch-go-def2", Title: "fix bug Y", Status: "in_progress"},
	}

	lines := CollectInFlight(issues)
	if len(lines) != 2 {
		t.Fatalf("expected 2 in-flight lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "orch-go-abc1") {
		t.Errorf("expected beads ID in line, got: %s", lines[0])
	}
}

func TestCollectInFlightEmpty(t *testing.T) {
	lines := CollectInFlight(nil)
	if len(lines) != 0 {
		t.Errorf("expected empty lines for nil issues, got %d", len(lines))
	}
}

func TestRenderDebrief(t *testing.T) {
	data := &DebriefData{
		Date:     "2026-02-28",
		Duration: "~3h",
		Focus:    "Ship debrief command",
		WhatHappened: []string{
			"Completed orch-go-abc1 (feature-impl)",
			"Spawned 2 agents",
		},
		WhatChanged: []string{
			"Decided to use .kb/sessions/ for debrief artifacts",
		},
		InFlight: []string{
			"orch-go-def2: fix bug Y (in_progress)",
		},
		WhatsNext: []string{
			"Integrate debrief into orch orient",
		},
		Health: HealthData{
			Checkpoint:    "ok",
			FrameCollapse: "none",
			DiscoveredWork: "none",
		},
	}

	output := RenderDebrief(data)

	// Check header
	if !strings.Contains(output, "# Session Debrief: 2026-02-28") {
		t.Error("expected header with date")
	}
	if !strings.Contains(output, "**Focus:** Ship debrief command") {
		t.Error("expected focus line")
	}

	// Check sections
	if !strings.Contains(output, "## What Happened") {
		t.Error("expected What Happened section")
	}
	if !strings.Contains(output, "Completed orch-go-abc1") {
		t.Error("expected completion line in output")
	}

	if !strings.Contains(output, "## What Changed") {
		t.Error("expected What Changed section")
	}

	if !strings.Contains(output, "## What's In Flight") {
		t.Error("expected In Flight section")
	}

	if !strings.Contains(output, "## What's Next") {
		t.Error("expected What's Next section")
	}

	if !strings.Contains(output, "## Session Health") {
		t.Error("expected Session Health section")
	}
}

func TestRenderDebriefEmptySections(t *testing.T) {
	data := &DebriefData{
		Date:  "2026-02-28",
		Focus: "testing",
		Health: HealthData{
			Checkpoint:    "ok",
			FrameCollapse: "none",
			DiscoveredWork: "none",
		},
	}

	output := RenderDebrief(data)

	// Empty sections should have placeholder
	if !strings.Contains(output, "- (none)") {
		t.Error("expected placeholder for empty sections")
	}
}

func TestFilterEventsToday(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	events := []SessionEvent{
		{Type: "agent.completed", Timestamp: today.Add(2 * time.Hour).Unix()},
		{Type: "session.spawned", Timestamp: today.Add(-25 * time.Hour).Unix()}, // yesterday
		{Type: "agent.abandoned", Timestamp: today.Add(5 * time.Hour).Unix()},
	}

	filtered := FilterEventsToday(events, now)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 today events, got %d", len(filtered))
	}
}

func TestParseChangedFlag(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"decided X because Y", []string{"decided X because Y"}},
		{"one;two;three", []string{"one", "two", "three"}},
		{"", nil},
	}

	for _, tt := range tests {
		result := ParseMultiValue(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("ParseMultiValue(%q): expected %d items, got %d", tt.input, len(tt.expected), len(result))
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("ParseMultiValue(%q)[%d]: expected %q, got %q", tt.input, i, tt.expected[i], v)
			}
		}
	}
}

func TestDebriefFilePath(t *testing.T) {
	path := DebriefFilePath("/project", "2026-02-28")
	expected := "/project/.kb/sessions/2026-02-28-debrief.md"
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}
