package debrief

import (
	"testing"
)

func TestCheckQuality_PassesGoodContent(t *testing.T) {
	data := &DebriefData{
		Date:  "2026-03-05",
		Focus: "Ship synthesis comprehension",
		WhatWeLearned: []string{
			"Template structure matters more than skill text because agents follow structural cues over instructions",
			"The debrief was producing event logs instead of insights, which means orient has no comprehension to surface",
		},
		WhatHappened: []string{
			"Completed: `feature-impl` (orch-go-abc1) — restructured debrief template",
		},
		WhatsNext: []string{
			"Integrate debrief insights into orient so comprehension threads persist across sessions",
		},
	}

	result := CheckQuality(data)
	if !result.Pass {
		t.Errorf("expected pass for good content, got fail with warnings: %v", result.Warnings)
	}
}

func TestCheckQuality_FlagsEmptySections(t *testing.T) {
	data := &DebriefData{
		Date:  "2026-03-05",
		Focus: "testing",
		// WhatWeLearned is empty
		WhatHappened: []string{
			"Completed: `feature-impl` (orch-go-abc1)",
		},
	}

	result := CheckQuality(data)
	if result.Pass {
		t.Error("expected fail when WhatWeLearned is empty")
	}

	found := false
	for _, w := range result.Warnings {
		if w.Pattern == "empty_learned" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected empty_learned warning, got: %v", result.Warnings)
	}
}

func TestCheckQuality_FlagsActionVerbOnly(t *testing.T) {
	data := &DebriefData{
		Date:  "2026-03-05",
		Focus: "testing",
		WhatWeLearned: []string{
			"Added JWT auth middleware",
			"Fixed the login bug",
			"Implemented refresh tokens",
		},
		WhatHappened: []string{
			"Completed: `feature-impl` (orch-go-abc1)",
		},
	}

	result := CheckQuality(data)
	if result.Pass {
		t.Error("expected fail when WhatWeLearned is all action-verb summaries")
	}

	found := false
	for _, w := range result.Warnings {
		if w.Pattern == "action_verb_only" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected action_verb_only warning, got: %v", result.Warnings)
	}
}

func TestCheckQuality_FlagsMissingConnectives(t *testing.T) {
	data := &DebriefData{
		Date:  "2026-03-05",
		Focus: "testing",
		WhatWeLearned: []string{
			"JWT auth middleware works",
			"Login bug is in the token flow",
		},
		WhatHappened: []string{
			"Completed: `feature-impl` (orch-go-abc1)",
		},
	}

	result := CheckQuality(data)
	if result.Pass {
		t.Error("expected fail when learned items lack connective language")
	}

	found := false
	for _, w := range result.Warnings {
		if w.Pattern == "missing_connectives" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_connectives warning, got: %v", result.Warnings)
	}
}

func TestCheckQuality_AcceptsConnectiveLanguage(t *testing.T) {
	// Items with connectives should pass
	data := &DebriefData{
		Date:  "2026-03-05",
		Focus: "testing",
		WhatWeLearned: []string{
			"Template structure matters more than skill text because agents follow structural cues",
			"The debrief was producing event logs which means orient has nothing to surface",
		},
		WhatHappened: []string{
			"Completed: something",
		},
	}

	result := CheckQuality(data)
	// Should not have missing_connectives
	for _, w := range result.Warnings {
		if w.Pattern == "missing_connectives" {
			t.Error("should not flag missing_connectives when connectives are present")
		}
	}
}

func TestCheckQuality_MixedContent(t *testing.T) {
	// One good item + one action-verb item should still warn
	data := &DebriefData{
		Date:  "2026-03-05",
		Focus: "testing",
		WhatWeLearned: []string{
			"Added JWT auth",
			"Template structure matters because agents follow cues over instructions",
		},
		WhatHappened: []string{
			"Completed: something",
		},
	}

	result := CheckQuality(data)
	// Should pass because at least one item has comprehension
	if !result.Pass {
		t.Errorf("mixed content with at least one good item should pass, got warnings: %v", result.Warnings)
	}
}

func TestIsActionVerbSentence(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"Added JWT auth middleware", true},
		{"Fixed the login bug", true},
		{"Implemented refresh tokens", true},
		{"Updated the config file", true},
		{"Refactored the auth module", true},
		{"Template structure matters because agents follow cues", false},
		{"The debrief was producing event logs", false},
		{"We discovered that X means Y", false},
	}

	for _, tt := range tests {
		result := IsActionVerbSentence(tt.input)
		if result != tt.expected {
			t.Errorf("IsActionVerbSentence(%q): expected %v, got %v", tt.input, tt.expected, result)
		}
	}
}

func TestHasConnectiveLanguage(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"X matters because Y", true},
		{"X which means Y", true},
		{"doing X therefore Y", true},
		{"this implies that Y", true},
		{"Added JWT auth", false},
		{"Fixed the bug", false},
		{"Login token flow works", false},
	}

	for _, tt := range tests {
		result := HasConnectiveLanguage(tt.input)
		if result != tt.expected {
			t.Errorf("HasConnectiveLanguage(%q): expected %v, got %v", tt.input, tt.expected, result)
		}
	}
}

func TestComprehensionPrompt(t *testing.T) {
	prompt := ComprehensionPrompt()
	if prompt == "" {
		t.Error("comprehension prompt should not be empty")
	}
	// Should mention Thread, Insight, Position
	for _, keyword := range []string{"Thread", "Insight", "Position"} {
		if !containsCI(prompt, keyword) {
			t.Errorf("comprehension prompt should mention %q", keyword)
		}
	}
}

// containsCI is case-insensitive contains for test use.
func containsCI(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		contains(lower(s), lower(substr))
}

func lower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
