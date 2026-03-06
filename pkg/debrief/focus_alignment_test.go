package debrief

import (
	"strings"
	"testing"
)

func TestCollectFocusAlignment_OnTrack(t *testing.T) {
	data := CollectFocusAlignment(
		"on-track",
		"Ship snap MVP",
		"orch-go-abc1",
		"Focused issue is among active work",
		[]string{"orch-go-abc1: Ship snap MVP"},
	)

	if data == nil {
		t.Fatal("expected non-nil data for on-track")
	}
	if data.Verdict != "on-track" {
		t.Errorf("expected verdict 'on-track', got %q", data.Verdict)
	}
	if data.Goal != "Ship snap MVP" {
		t.Errorf("expected goal 'Ship snap MVP', got %q", data.Goal)
	}
	if data.FocusedIssue != "orch-go-abc1" {
		t.Errorf("expected focused issue 'orch-go-abc1', got %q", data.FocusedIssue)
	}
}

func TestCollectFocusAlignment_Drifting(t *testing.T) {
	data := CollectFocusAlignment(
		"drifting",
		"Ship snap MVP",
		"orch-go-abc1",
		"Active work does not include focused issue",
		[]string{"orch-go-def2: Fix bug Y", "orch-go-ghi3: Refactor"},
	)

	if data == nil {
		t.Fatal("expected non-nil data for drifting")
	}
	if data.Verdict != "drifting" {
		t.Errorf("expected verdict 'drifting', got %q", data.Verdict)
	}
	if len(data.ActiveWork) != 2 {
		t.Errorf("expected 2 active work items, got %d", len(data.ActiveWork))
	}
}

func TestCollectFocusAlignment_NoFocusReturnsNil(t *testing.T) {
	data := CollectFocusAlignment("no-focus", "", "", "No focus set", nil)
	if data != nil {
		t.Error("expected nil for no-focus verdict")
	}
}

func TestCollectFocusAlignment_EmptyVerdictReturnsNil(t *testing.T) {
	data := CollectFocusAlignment("", "", "", "", nil)
	if data != nil {
		t.Error("expected nil for empty verdict")
	}
}

func TestCollectFocusAlignment_Unverified(t *testing.T) {
	data := CollectFocusAlignment(
		"unverified",
		"Ship snap MVP",
		"",
		"Focus has no specific issue — review active work against goal",
		[]string{"orch-go-def2: Fix bug Y"},
	)

	if data == nil {
		t.Fatal("expected non-nil data for unverified")
	}
	if data.Verdict != "unverified" {
		t.Errorf("expected verdict 'unverified', got %q", data.Verdict)
	}
	if data.FocusedIssue != "" {
		t.Errorf("expected empty focused issue, got %q", data.FocusedIssue)
	}
}

func TestFormatFocusAlignment_OnTrack(t *testing.T) {
	data := &FocusAlignmentData{
		Verdict:      "on-track",
		Goal:         "Ship snap MVP",
		FocusedIssue: "orch-go-abc1",
		Reason:       "Focused issue is among active work",
		ActiveWork:   []string{"orch-go-abc1: Ship snap MVP"},
	}

	lines := FormatFocusAlignment(data)
	if len(lines) == 0 {
		t.Fatal("expected non-empty lines")
	}

	// Verdict line
	if !strings.Contains(lines[0], "ON TRACK") {
		t.Errorf("expected ON TRACK indicator, got: %s", lines[0])
	}
	if !strings.Contains(lines[0], "Focused issue is among active work") {
		t.Errorf("expected reason in line, got: %s", lines[0])
	}

	// Should include focused issue
	found := false
	for _, line := range lines {
		if strings.Contains(line, "Focused issue: orch-go-abc1") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected focused issue line, got: %v", lines)
	}
}

func TestFormatFocusAlignment_Drifting(t *testing.T) {
	data := &FocusAlignmentData{
		Verdict:      "drifting",
		Goal:         "Ship snap MVP",
		FocusedIssue: "orch-go-abc1",
		Reason:       "Active work does not include focused issue",
		ActiveWork:   []string{"orch-go-def2: Fix bug Y"},
	}

	lines := FormatFocusAlignment(data)
	if len(lines) == 0 {
		t.Fatal("expected non-empty lines")
	}

	if !strings.Contains(lines[0], "DRIFTING") {
		t.Errorf("expected DRIFTING indicator, got: %s", lines[0])
	}

	// Should include active work
	found := false
	for _, line := range lines {
		if strings.Contains(line, "Active: orch-go-def2") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected active work line, got: %v", lines)
	}
}

func TestFormatFocusAlignment_Nil(t *testing.T) {
	lines := FormatFocusAlignment(nil)
	if lines != nil {
		t.Errorf("expected nil for nil data, got %d lines", len(lines))
	}
}

func TestFormatFocusAlignment_NoFocusedIssue(t *testing.T) {
	data := &FocusAlignmentData{
		Verdict: "unverified",
		Goal:    "Ship snap MVP",
		Reason:  "Focus has no specific issue",
	}

	lines := FormatFocusAlignment(data)
	for _, line := range lines {
		if strings.Contains(line, "Focused issue:") {
			t.Errorf("should not include focused issue line when empty, got: %s", line)
		}
	}
}
