package orient

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDebriefSummary_BasicDebrief(t *testing.T) {
	content := `# Session Debrief: 2026-02-28

**Date:** 2026-02-28
**Duration:** ~3h
**Focus:** Ship snap MVP

---

## What We Learned

- decided to use JWT for auth
- new constraint on debriefs

## Open Questions

- should we split auth refresh from the middleware path?

## What's In Flight

- orch-go-abc1: Fix spawn crash on empty skill (in_progress)
- orch-go-ghi3: Add model drift detection (in_progress)

## What's Next

- fix auth
- ship snap MVP
- review hotspot results

## What Happened

- Completed 2: orch-go-abc1 (feature-impl), orch-go-def2 (investigation)
- Spawned 5 agent(s)
`

	summary := ParseDebriefSummary(content)
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}

	if summary.Date != "2026-02-28" {
		t.Errorf("expected date '2026-02-28', got %q", summary.Date)
	}

	if len(summary.WhatWeLearned) != 2 {
		t.Errorf("expected 2 WhatWeLearned items, got %d", len(summary.WhatWeLearned))
	}

	if len(summary.OpenQuestions) != 1 {
		t.Errorf("expected 1 OpenQuestions item, got %d", len(summary.OpenQuestions))
	}

	if len(summary.InFlight) != 2 {
		t.Errorf("expected 2 InFlight items, got %d", len(summary.InFlight))
	}

	if len(summary.WhatsNext) != 3 {
		t.Errorf("expected 3 WhatsNext items, got %d", len(summary.WhatsNext))
	}

	if len(summary.WhatHappened) != 2 {
		t.Errorf("expected 2 WhatHappened items, got %d", len(summary.WhatHappened))
	}

	// Verify content parsing strips list markers
	if summary.WhatWeLearned[0] != "decided to use JWT for auth" {
		t.Errorf("expected 'decided to use JWT for auth', got %q", summary.WhatWeLearned[0])
	}
	if summary.OpenQuestions[0] != "should we split auth refresh from the middleware path?" {
		t.Errorf("unexpected open question: %q", summary.OpenQuestions[0])
	}

	// Verify bullet list items are parsed correctly
	if summary.WhatsNext[0] != "fix auth" {
		t.Errorf("expected 'fix auth', got %q", summary.WhatsNext[0])
	}
}

func TestParseDebriefSummary_EmptyContent(t *testing.T) {
	summary := ParseDebriefSummary("")
	if summary != nil {
		t.Error("expected nil summary for empty content")
	}
}

func TestParseDebriefSummary_NoSections(t *testing.T) {
	content := `# Session Debrief: 2026-02-28

**Date:** 2026-02-28
**Focus:** Something
`
	summary := ParseDebriefSummary(content)
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if summary.Date != "2026-02-28" {
		t.Errorf("expected date '2026-02-28', got %q", summary.Date)
	}
	if len(summary.WhatHappened) != 0 {
		t.Errorf("expected 0 WhatHappened items, got %d", len(summary.WhatHappened))
	}
}

func TestParseDebriefSummary_TruncatesLongLists(t *testing.T) {
	content := `# Session Debrief: 2026-02-28

**Date:** 2026-02-28

## What We Learned

- change 1
- change 2
- change 3
- change 4
- change 5
`
	summary := ParseDebriefSummary(content)
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}

	if len(summary.WhatWeLearned) != maxDebriefItems {
		t.Errorf("expected %d WhatWeLearned items (max), got %d", maxDebriefItems, len(summary.WhatWeLearned))
	}
}

func TestParseDebriefSummary_NoneItems(t *testing.T) {
	content := `# Session Debrief: 2026-02-28

**Date:** 2026-02-28

## What Happened

- (none)

## What We Learned

- (none)
`
	summary := ParseDebriefSummary(content)
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}

	if len(summary.WhatHappened) != 0 {
		t.Errorf("expected 0 WhatHappened (none items filtered), got %d", len(summary.WhatHappened))
	}
	if len(summary.WhatWeLearned) != 0 {
		t.Errorf("expected 0 WhatWeLearned (none items filtered), got %d", len(summary.WhatWeLearned))
	}
}

func TestFindLatestDebrief(t *testing.T) {
	dir := t.TempDir()

	// Create some debrief files
	os.WriteFile(filepath.Join(dir, "2026-02-26-debrief.md"), []byte("old"), 0644)
	os.WriteFile(filepath.Join(dir, "2026-02-27-debrief.md"), []byte("middle"), 0644)
	os.WriteFile(filepath.Join(dir, "2026-02-28-debrief.md"), []byte("latest"), 0644)
	// Non-debrief file should be ignored
	os.WriteFile(filepath.Join(dir, "notes.md"), []byte("notes"), 0644)

	path, err := FindLatestDebrief(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(dir, "2026-02-28-debrief.md")
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestFindLatestDebrief_NoFiles(t *testing.T) {
	dir := t.TempDir()

	_, err := FindLatestDebrief(dir)
	if err == nil {
		t.Error("expected error for empty directory")
	}
}

func TestFindLatestDebrief_MissingDir(t *testing.T) {
	_, err := FindLatestDebrief("/nonexistent/path")
	if err == nil {
		t.Error("expected error for missing directory")
	}
}

func TestFormatPreviousSession(t *testing.T) {
	summary := &DebriefSummary{
		Date: "2026-02-28",
		WhatHappened: []string{
			"Completed 2: orch-go-abc1 (feature-impl), orch-go-def2 (investigation)",
			"Spawned 5 agent(s)",
		},
		WhatWeLearned: []string{
			"decided to use JWT for auth",
		},
		OpenQuestions: []string{
			"should we split auth refresh from the middleware path?",
		},
		InFlight: []string{
			"orch-go-abc1: Fix spawn crash on empty skill (in_progress)",
		},
		WhatsNext: []string{
			"fix auth",
			"ship snap MVP",
		},
	}

	result := FormatPreviousSession(summary)

	// Check section header
	if !contains(result, "Previous session (2026-02-28):") {
		t.Errorf("expected 'Previous session (2026-02-28):' header, got:\n%s", result)
	}

	// Check all sections present
	if !contains(result, "Happened:") {
		t.Errorf("expected 'Happened:' section, got:\n%s", result)
	}
	if !contains(result, "Learned:") {
		t.Errorf("expected 'Changed:' section, got:\n%s", result)
	}
	if !contains(result, "Open questions:") {
		t.Errorf("expected 'Open questions:' section, got:\n%s", result)
	}
	if !contains(result, "In flight:") {
		t.Errorf("expected 'In flight:' section, got:\n%s", result)
	}
	if !contains(result, "Next:") {
		t.Errorf("expected 'Next:' section, got:\n%s", result)
	}
}

func TestFormatPreviousSession_Nil(t *testing.T) {
	result := FormatPreviousSession(nil)
	if result != "" {
		t.Errorf("expected empty string for nil summary, got %q", result)
	}
}

func TestFormatLastSessionInsight(t *testing.T) {
	summary := &DebriefSummary{
		Date: "2026-03-04",
		WhatWeLearned: []string{
			"Discovered that injection level matters less than density for skill effectiveness",
			"System injection 71% vs append 58% vs user 56%",
		},
	}

	result := FormatLastSessionInsight(summary)

	if !strings.Contains(result, "Last session insight (2026-03-04):") {
		t.Errorf("expected 'Last session insight (2026-03-04):' header, got:\n%s", result)
	}
	if !strings.Contains(result, "injection level matters less") {
		t.Errorf("expected insight content, got:\n%s", result)
	}
}

func TestFormatLastSessionInsight_Nil(t *testing.T) {
	result := FormatLastSessionInsight(nil)
	if result != "" {
		t.Errorf("expected empty string for nil summary, got %q", result)
	}
}

func TestFormatLastSessionInsight_NoLearnings(t *testing.T) {
	summary := &DebriefSummary{
		Date:         "2026-03-04",
		WhatHappened: []string{"Spawned 5 agents"},
	}

	result := FormatLastSessionInsight(summary)
	if result != "" {
		t.Errorf("expected empty string when no learnings, got %q", result)
	}
}

func TestFormatPreviousSession_AllEmpty(t *testing.T) {
	summary := &DebriefSummary{
		Date: "2026-02-28",
	}

	result := FormatPreviousSession(summary)
	if result != "" {
		t.Errorf("expected empty string for summary with no content, got %q", result)
	}
}

func TestFormatHealth_WithDebrief(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1, Completions: 2},
		PreviousSession: &DebriefSummary{
			Date:          "2026-02-27",
			WhatHappened:  []string{"Completed 2: orch-go-abc1, orch-go-def2"},
			WhatWeLearned: []string{"decided to use JWT for auth"},
			WhatsNext:     []string{"fix auth", "ship snap"},
		},
	}

	output := FormatHealth(data)

	if !strings.Contains(output, "Previous session (2026-02-27):") {
		t.Errorf("expected 'Previous session' section, got:\n%s", output)
	}
	if !strings.Contains(output, "Happened:") {
		t.Errorf("expected 'Happened:' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Learned:") {
		t.Errorf("expected 'Changed:' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Next:") {
		t.Errorf("expected 'Next:' in output, got:\n%s", output)
	}
}

func TestFormatOrientation_WithLastSessionInsight(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1, Completions: 2},
		PreviousSession: &DebriefSummary{
			Date:          "2026-03-04",
			WhatHappened:  []string{"Spawned 5 agents"},
			WhatWeLearned: []string{"Density matters 2x more than injection level"},
		},
	}

	output := FormatOrientation(data)

	if strings.Contains(output, "Last session insight") {
		t.Errorf("last session insight should not appear in thinking surface, got:\n%s", output)
	}
	if strings.Contains(output, "Density matters 2x more") {
		t.Errorf("last session insight content should not appear in thinking surface, got:\n%s", output)
	}
}

func TestFormatOrientation_NoInsightWhenNoLearnings(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1, Completions: 2},
		PreviousSession: &DebriefSummary{
			Date:         "2026-03-04",
			WhatHappened: []string{"Spawned 5 agents"},
		},
	}

	output := FormatOrientation(data)

	if strings.Contains(output, "Last session insight") {
		t.Errorf("should not show 'Last session insight' when no learnings, got:\n%s", output)
	}
}

func TestFormatOrientation_WithoutDebrief(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1, Completions: 2},
	}

	output := FormatOrientation(data)

	if strings.Contains(output, "Previous session") {
		t.Errorf("should not contain 'Previous session' when no debrief, got:\n%s", output)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
