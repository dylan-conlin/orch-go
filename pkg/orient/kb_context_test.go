package orient

import (
	"strings"
	"testing"
)

func TestParseKBContext_FullJSON(t *testing.T) {
	jsonData := []byte(`{
		"constraints": [
			{"type": "constraint", "content": "kb context command hangs on some queries", "reason": "Blocks orch spawn from returning", "result": "", "tags": "", "scope": "", "status": "active"},
			{"type": "constraint", "content": "Worker spawns must set ORCH_WORKER=1", "reason": "Orchestrator skill wastes context budget", "result": "", "tags": "", "scope": "", "status": "active"}
		],
		"decisions": [
			{"type": "decision", "content": "orch spawn context delivery is reliable", "reason": "Verified that SPAWN_CONTEXT.md is correctly populated", "result": "", "tags": "", "scope": "", "status": "active"}
		],
		"attempts": [
			{"type": "attempt", "content": "BEADS_NO_DAEMON=1 in .zshrc only", "reason": "", "result": "Doesn't propagate to launchd agents", "tags": "", "scope": "", "status": "active"}
		],
		"questions": null,
		"investigations": []
	}`)

	entries := ParseKBContext(jsonData, 2)

	if len(entries) == 0 {
		t.Fatal("expected entries, got none")
	}

	// Should have 2 constraints + 1 decision + 1 attempt = 4 (maxPerType=2 limits constraints)
	constraintCount := 0
	decisionCount := 0
	attemptCount := 0
	for _, e := range entries {
		switch e.Type {
		case "constraint":
			constraintCount++
		case "decision":
			decisionCount++
		case "attempt":
			attemptCount++
		}
	}

	if constraintCount != 2 {
		t.Errorf("expected 2 constraints, got %d", constraintCount)
	}
	if decisionCount != 1 {
		t.Errorf("expected 1 decision, got %d", decisionCount)
	}
	if attemptCount != 1 {
		t.Errorf("expected 1 attempt, got %d", attemptCount)
	}
}

func TestParseKBContext_MaxPerType(t *testing.T) {
	jsonData := []byte(`{
		"constraints": [
			{"type": "constraint", "content": "c1", "reason": "r1"},
			{"type": "constraint", "content": "c2", "reason": "r2"},
			{"type": "constraint", "content": "c3", "reason": "r3"}
		],
		"decisions": [
			{"type": "decision", "content": "d1", "reason": "r1"},
			{"type": "decision", "content": "d2", "reason": "r2"}
		],
		"attempts": null
	}`)

	entries := ParseKBContext(jsonData, 1)

	// maxPerType=1: 1 constraint + 1 decision = 2
	if len(entries) != 2 {
		t.Errorf("expected 2 entries with maxPerType=1, got %d", len(entries))
	}
}

func TestParseKBContext_InvalidJSON(t *testing.T) {
	entries := ParseKBContext([]byte("not json"), 2)
	if entries != nil {
		t.Errorf("expected nil for invalid JSON, got %v", entries)
	}
}

func TestParseKBContext_EmptyFields(t *testing.T) {
	jsonData := []byte(`{"constraints": null, "decisions": null, "attempts": null}`)
	entries := ParseKBContext(jsonData, 2)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for null fields, got %d", len(entries))
	}
}

func TestParseKBContext_AttemptUsesResult(t *testing.T) {
	jsonData := []byte(`{
		"constraints": null,
		"decisions": null,
		"attempts": [
			{"type": "attempt", "content": "tried X", "reason": "", "result": "Failed because Y"}
		]
	}`)

	entries := ParseKBContext(jsonData, 2)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Reason != "Failed because Y" {
		t.Errorf("expected attempt reason to use result field, got %q", entries[0].Reason)
	}
}

func TestSelectTopEntries(t *testing.T) {
	entries := []KBEntry{
		{Type: "constraint", Content: "c1"},
		{Type: "decision", Content: "d1"},
		{Type: "attempt", Content: "a1"},
		{Type: "decision", Content: "d2"},
		{Type: "constraint", Content: "c2"},
	}

	top := SelectTopEntries(entries, 2)
	if len(top) != 2 {
		t.Fatalf("expected 2 top entries, got %d", len(top))
	}

	// Constraints should come first (highest priority)
	if top[0].Type != "constraint" {
		t.Errorf("expected first entry to be constraint, got %s", top[0].Type)
	}
	if top[1].Type != "constraint" {
		t.Errorf("expected second entry to be constraint, got %s", top[1].Type)
	}
}

func TestSelectTopEntries_MixedPriority(t *testing.T) {
	entries := []KBEntry{
		{Type: "constraint", Content: "c1"},
		{Type: "attempt", Content: "a1"},
		{Type: "decision", Content: "d1"},
	}

	// maxTotal=2 should get: constraint, then attempt (higher priority than decision)
	top := SelectTopEntries(entries, 2)
	if len(top) != 2 {
		t.Fatalf("expected 2, got %d", len(top))
	}
	if top[0].Type != "constraint" {
		t.Errorf("expected constraint first, got %s", top[0].Type)
	}
	if top[1].Type != "attempt" {
		t.Errorf("expected attempt second, got %s", top[1].Type)
	}
}

func TestSelectTopEntries_UnderLimit(t *testing.T) {
	entries := []KBEntry{
		{Type: "decision", Content: "d1"},
	}

	top := SelectTopEntries(entries, 5)
	if len(top) != 1 {
		t.Errorf("expected 1 entry when under limit, got %d", len(top))
	}
}

func TestFormatReadyIssuesWithContext(t *testing.T) {
	issues := []ReadyIssue{
		{
			ID:       "orch-go-abc1",
			Title:    "Fix spawn bug",
			Priority: "P1",
			KBContext: []KBEntry{
				{Type: "constraint", Content: "kb context command hangs on some queries"},
				{Type: "decision", Content: "orch spawn context delivery is reliable"},
			},
		},
		{
			ID:       "orch-go-def2",
			Title:    "Add model drift",
			Priority: "P2",
		},
	}

	var b strings.Builder
	formatReadyIssues(&b, issues)
	output := b.String()

	// Issue lines should still be present
	if !strings.Contains(output, "orch-go-abc1") {
		t.Error("missing issue ID")
	}
	if !strings.Contains(output, "Fix spawn bug") {
		t.Error("missing issue title")
	}

	// Decision context should appear under the issue
	if !strings.Contains(output, "constraint") {
		t.Error("missing constraint label")
	}
	if !strings.Contains(output, "kb context command hangs") {
		t.Error("missing constraint content")
	}
	if !strings.Contains(output, "decision") {
		t.Error("missing decision label")
	}

	// Second issue without context should still render
	if !strings.Contains(output, "orch-go-def2") {
		t.Error("missing second issue")
	}
}

func TestFormatReadyIssuesWithContext_NoContext(t *testing.T) {
	issues := []ReadyIssue{
		{ID: "orch-go-abc1", Title: "Fix spawn bug", Priority: "P1"},
	}

	var b strings.Builder
	formatReadyIssues(&b, issues)
	output := b.String()

	// Should not have any context lines, just the issue line
	lines := strings.Split(strings.TrimSpace(output), "\n")
	contextLines := 0
	for _, line := range lines {
		if strings.Contains(line, "constraint") || strings.Contains(line, "decision") || strings.Contains(line, "attempt") {
			contextLines++
		}
	}
	if contextLines != 0 {
		t.Errorf("expected no context lines for issues without kb context, got %d", contextLines)
	}
}
