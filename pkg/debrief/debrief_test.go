package debrief

import (
	"strings"
	"testing"
	"time"
)

func TestCollectWhatHappened(t *testing.T) {
	events := []SessionEvent{
		{Type: "agent.completed", Timestamp: time.Now().Unix(), Data: map[string]interface{}{
			"beads_id": "orch-go-abc1",
			"skill":    "feature-impl",
			"reason":   "Added JWT auth middleware with refresh tokens",
		}},
		{Type: "session.spawned", Timestamp: time.Now().Unix(), Data: map[string]interface{}{
			"beads_id": "orch-go-xyz9",
			"skill":    "investigation",
			"task":     "investigate auth patterns",
		}},
		{Type: "agent.abandoned", Timestamp: time.Now().Unix(), Data: map[string]interface{}{
			"beads_id": "orch-go-def2",
			"reason":   "stuck in loop",
		}},
	}

	lines := CollectWhatHappened(events)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (one per event), got %d: %v", len(lines), lines)
	}

	// Completions show skill + beads_id + reason
	if !strings.Contains(lines[0], "feature-impl") {
		t.Errorf("expected skill in completion line, got: %s", lines[0])
	}
	if !strings.Contains(lines[0], "orch-go-abc1") {
		t.Errorf("expected beads_id in completion line, got: %s", lines[0])
	}
	if !strings.Contains(lines[0], "Added JWT auth") {
		t.Errorf("expected reason in completion line, got: %s", lines[0])
	}

	// Spawns show skill + task
	if !strings.Contains(lines[1], "investigation") {
		t.Errorf("expected skill in spawn line, got: %s", lines[1])
	}
	if !strings.Contains(lines[1], "investigate auth patterns") {
		t.Errorf("expected task in spawn line, got: %s", lines[1])
	}

	// Abandonments show beads_id + reason
	if !strings.Contains(lines[2], "orch-go-def2") {
		t.Errorf("expected beads_id in abandon line, got: %s", lines[2])
	}
	if !strings.Contains(lines[2], "stuck in loop") {
		t.Errorf("expected reason in abandon line, got: %s", lines[2])
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
		WhatWeLearned: []string{
			"Added JWT auth middleware with refresh tokens",
		},
		WhatHappened: []string{
			"Completed: `feature-impl` (orch-go-abc1) — Added JWT auth",
			"Spawned: `investigation` — investigate auth patterns",
		},
		InFlight: []string{
			"orch-go-def2: fix bug Y (in_progress)",
		},
		WhatsNext: []string{
			"Integrate debrief into orch orient",
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

	// Check sections present in comprehension order
	if !strings.Contains(output, "## What We Learned") {
		t.Error("expected What We Learned section")
	}

	if !strings.Contains(output, "## What's In Flight") {
		t.Error("expected In Flight section")
	}

	if !strings.Contains(output, "## What's Next") {
		t.Error("expected What's Next section")
	}

	if !strings.Contains(output, "## What Happened") {
		t.Error("expected What Happened section")
	}
	if !strings.Contains(output, "feature-impl") {
		t.Error("expected skill in output")
	}

	// Verify section order: Learned before Happened (comprehension order)
	learnedIdx := strings.Index(output, "## What We Learned")
	happenedIdx := strings.Index(output, "## What Happened")
	if learnedIdx >= happenedIdx {
		t.Error("What We Learned should appear before What Happened")
	}

	// What's Next should use bullet list, not numbered
	nextIdx := strings.Index(output, "## What's Next")
	nextSection := output[nextIdx:]
	if strings.Contains(nextSection, "1. ") {
		t.Error("What's Next should use bullet list, not numbered list")
	}

	// Health section should NOT be present
	if strings.Contains(output, "Session Health") {
		t.Error("Health section should be removed")
	}
}

func TestRenderDebriefWithDriftAndFriction(t *testing.T) {
	data := &DebriefData{
		Date:  "2026-03-05",
		Focus: "testing drift/friction",
		WhatWeLearned: []string{
			"Template structure matters because agents follow structural cues",
		},
		DriftSummary: []string{
			"**agent-lifecycle:** 3 stale spawn(s), 2 changed",
		},
		FrictionSummary: []string{
			"**bug:** beads dir resolution fails",
			"**tooling:** bd sync noise",
		},
		WhatHappened: []string{
			"Spawned: `feature-impl` — test",
		},
	}

	output := RenderDebrief(data)

	// Check new sections present
	if !strings.Contains(output, "## Drift Summary") {
		t.Error("expected Drift Summary section")
	}
	if !strings.Contains(output, "## Friction Summary") {
		t.Error("expected Friction Summary section")
	}

	// Check section order: Learned < Drift < Friction < In Flight
	learnedIdx := strings.Index(output, "## What We Learned")
	driftIdx := strings.Index(output, "## Drift Summary")
	frictionIdx := strings.Index(output, "## Friction Summary")
	inFlightIdx := strings.Index(output, "## What's In Flight")

	if learnedIdx >= driftIdx {
		t.Error("Drift Summary should appear after What We Learned")
	}
	if driftIdx >= frictionIdx {
		t.Error("Friction Summary should appear after Drift Summary")
	}
	if frictionIdx >= inFlightIdx {
		t.Error("What's In Flight should appear after Friction Summary")
	}

	// Verify content rendered
	if !strings.Contains(output, "agent-lifecycle") {
		t.Error("expected drift domain in output")
	}
	if !strings.Contains(output, "beads dir resolution") {
		t.Error("expected friction description in output")
	}
}

func TestRenderDebriefOmitsEmptyDriftFriction(t *testing.T) {
	data := &DebriefData{
		Date:  "2026-03-05",
		Focus: "no drift or friction",
	}

	output := RenderDebrief(data)

	// Empty drift/friction should not appear
	if strings.Contains(output, "Drift Summary") {
		t.Error("Drift Summary should not appear when empty")
	}
	if strings.Contains(output, "Friction Summary") {
		t.Error("Friction Summary should not appear when empty")
	}
}

func TestRenderDebriefEmptySections(t *testing.T) {
	data := &DebriefData{
		Date:  "2026-02-28",
		Focus: "testing",
	}

	output := RenderDebrief(data)

	// Empty sections should have placeholder
	if !strings.Contains(output, "- (none)") {
		t.Error("expected placeholder for empty sections")
	}
}

func TestCollectWhatWeLearnedFromEvents(t *testing.T) {
	events := []SessionEvent{
		{Type: "agent.completed", Timestamp: time.Now().Unix(), Data: map[string]interface{}{
			"beads_id": "orch-go-abc1",
			"skill":    "feature-impl",
			"reason":   "Added JWT auth middleware with refresh tokens",
		}},
		{Type: "agent.completed", Timestamp: time.Now().Unix(), Data: map[string]interface{}{
			"beads_id": "orch-go-def2",
			"skill":    "investigation",
			"reason":   "Investigated auth patterns and documented findings",
		}},
		{Type: "session.spawned", Timestamp: time.Now().Unix(), Data: map[string]interface{}{
			"task": "some spawn",
		}},
	}

	lines := CollectWhatWeLearned(events)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (one per completion), got %d: %v", len(lines), lines)
	}
	if !strings.Contains(lines[0], "Added JWT auth") {
		t.Errorf("expected reason text, got: %s", lines[0])
	}
	if !strings.Contains(lines[1], "Investigated auth") {
		t.Errorf("expected reason text, got: %s", lines[1])
	}
}

func TestCollectWhatWeLearnedEmpty(t *testing.T) {
	lines := CollectWhatWeLearned(nil)
	if len(lines) != 0 {
		t.Errorf("expected empty lines, got %d", len(lines))
	}
}

func TestCollectWhatWeLearnedSkipsDuplicateBeadsID(t *testing.T) {
	events := []SessionEvent{
		{Type: "agent.completed", Timestamp: time.Now().Unix(), Data: map[string]interface{}{
			"beads_id": "orch-go-abc1",
			"reason":   "First completion",
		}},
		{Type: "agent.completed", Timestamp: time.Now().Unix(), Data: map[string]interface{}{
			"beads_id": "orch-go-abc1",
			"reason":   "Duplicate completion",
		}},
	}

	lines := CollectWhatWeLearned(events)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line (deduped), got %d: %v", len(lines), lines)
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

func TestFormatThreadEntries(t *testing.T) {
	entries := []ThreadEntryItem{
		{ThreadTitle: "When does detection become prevention?", Text: "Deploy or Delete has a named principle and detection mechanism. The real question: when does detection become prevention?"},
		{ThreadTitle: "How enforcement relates to comprehension", Text: "Enforcement without comprehension is just compliance theater."},
	}

	lines := FormatThreadEntries(entries)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	// Should include thread title as context
	if !strings.Contains(lines[0], "detection become prevention") {
		t.Errorf("expected thread title context, got: %s", lines[0])
	}
	if !strings.Contains(lines[0], "Deploy or Delete") {
		t.Errorf("expected entry text, got: %s", lines[0])
	}

	// Second entry
	if !strings.Contains(lines[1], "enforcement") {
		t.Errorf("expected thread title context in second line, got: %s", lines[1])
	}
}

func TestFormatThreadEntriesEmpty(t *testing.T) {
	lines := FormatThreadEntries(nil)
	if len(lines) != 0 {
		t.Errorf("expected empty lines for nil entries, got %d", len(lines))
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero returns empty", 0, ""},
		{"30 seconds returns empty", 30 * time.Second, ""},
		{"5 minutes", 5 * time.Minute, "~5m"},
		{"90 minutes", 90 * time.Minute, "~1h"},
		{"3 hours", 3 * time.Hour, "~3h"},
		{"23 hours", 23 * time.Hour, "~23h"},
		{"exactly 24 hours is stale", 24 * time.Hour, ""},
		{"25 hours is stale", 25 * time.Hour, ""},
		{"954 hours is stale", 954 * time.Hour, ""},
		{"negative returns empty", -1 * time.Hour, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatDuration(%v): expected %q, got %q", tt.duration, tt.expected, result)
			}
		})
	}
}
