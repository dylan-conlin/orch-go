package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
)

func TestSynthesisWarningThreshold(t *testing.T) {
	// Verify the threshold constant matches kb-cli's SynthesisIssueThreshold
	if SynthesisWarningThreshold != 10 {
		t.Errorf("SynthesisWarningThreshold = %d, want 10 (to match kb-cli)", SynthesisWarningThreshold)
	}
}

func TestSuggestionFreshnessHours(t *testing.T) {
	// Verify the freshness check is 24 hours
	if SuggestionFreshnessHours != 24 {
		t.Errorf("SuggestionFreshnessHours = %d, want 24", SuggestionFreshnessHours)
	}
}

func TestFilterHighCountSynthesis(t *testing.T) {
	tests := []struct {
		name      string
		synthesis []daemon.SynthesisSuggestion
		wantCount int
	}{
		{
			name: "filters below threshold",
			synthesis: []daemon.SynthesisSuggestion{
				{Topic: "low", Count: 3},
				{Topic: "medium", Count: 9},
				{Topic: "high", Count: 10},
				{Topic: "veryhigh", Count: 50},
			},
			wantCount: 2, // high and veryhigh
		},
		{
			name:      "empty list",
			synthesis: []daemon.SynthesisSuggestion{},
			wantCount: 0,
		},
		{
			name: "all below threshold",
			synthesis: []daemon.SynthesisSuggestion{
				{Topic: "a", Count: 3},
				{Topic: "b", Count: 5},
				{Topic: "c", Count: 9},
			},
			wantCount: 0,
		},
		{
			name: "all above threshold",
			synthesis: []daemon.SynthesisSuggestion{
				{Topic: "a", Count: 10},
				{Topic: "b", Count: 20},
				{Topic: "c", Count: 30},
			},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var highCount []daemon.SynthesisSuggestion
			for _, s := range tt.synthesis {
				if s.Count >= SynthesisWarningThreshold {
					highCount = append(highCount, s)
				}
			}
			if len(highCount) != tt.wantCount {
				t.Errorf("filtered count = %d, want %d", len(highCount), tt.wantCount)
			}
		})
	}
}

func TestSuggestionFreshnessCheck(t *testing.T) {
	tests := []struct {
		name      string
		timestamp time.Time
		wantFresh bool
	}{
		{
			name:      "fresh - 1 hour ago",
			timestamp: time.Now().Add(-1 * time.Hour),
			wantFresh: true,
		},
		{
			name:      "fresh - 23 hours ago",
			timestamp: time.Now().Add(-23 * time.Hour),
			wantFresh: true,
		},
		{
			name:      "stale - 25 hours ago",
			timestamp: time.Now().Add(-25 * time.Hour),
			wantFresh: false,
		},
		{
			name:      "stale - 48 hours ago",
			timestamp: time.Now().Add(-48 * time.Hour),
			wantFresh: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isFresh := time.Since(tt.timestamp).Hours() <= SuggestionFreshnessHours
			if isFresh != tt.wantFresh {
				t.Errorf("freshness = %v, want %v", isFresh, tt.wantFresh)
			}
		})
	}
}

func TestUpdateHandoffTemplate(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a test handoff file with placeholders
	testContent := `# Session Handoff

**Orchestrator:** test-session
**Focus:** Test focus
**Duration:** 2026-01-14 15:00 → {end-time}
**Outcome:** {success | partial | blocked | failed}

## TLDR

[Fill within first 5 tool calls: What is this session trying to accomplish?]

## More sections...
`
	handoffPath := filepath.Join(tmpDir, "SESSION_HANDOFF.md")
	if err := os.WriteFile(handoffPath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test handoff: %v", err)
	}

	// Test updating the template
	summary := &sessionSummary{
		Outcome: "success",
		Summary: "Fixed the bug and added tests",
	}
	endTime := "2026-01-14 16:00"

	if err := updateHandoffTemplate(tmpDir, summary, endTime); err != nil {
		t.Fatalf("updateHandoffTemplate failed: %v", err)
	}

	// Read updated content
	updatedContent, err := os.ReadFile(handoffPath)
	if err != nil {
		t.Fatalf("Failed to read updated handoff: %v", err)
	}

	updated := string(updatedContent)

	// Verify {end-time} was replaced
	if strings.Contains(updated, "{end-time}") {
		t.Error("Expected {end-time} to be replaced, but it's still present")
	}
	if !strings.Contains(updated, endTime) {
		t.Errorf("Expected updated content to contain %q, but it doesn't", endTime)
	}

	// Verify outcome placeholder was replaced
	if strings.Contains(updated, "{success | partial | blocked | failed}") {
		t.Error("Expected outcome placeholder to be replaced, but it's still present")
	}
	if !strings.Contains(updated, "success") {
		t.Error("Expected updated content to contain 'success', but it doesn't")
	}

	// Verify TLDR placeholder was replaced with summary
	if strings.Contains(updated, "[Fill within first 5 tool calls: What is this session trying to accomplish?]") {
		t.Error("Expected TLDR placeholder to be replaced, but it's still present")
	}
	if !strings.Contains(updated, summary.Summary) {
		t.Errorf("Expected updated content to contain summary %q, but it doesn't", summary.Summary)
	}
}

func TestUpdateHandoffTemplateNoSummary(t *testing.T) {
	// Test that when no summary is provided, TLDR placeholder remains
	tmpDir := t.TempDir()

	testContent := `# Session Handoff

**Duration:** 2026-01-14 15:00 → {end-time}
**Outcome:** {success | partial | blocked | failed}

## TLDR

[Fill within first 5 tool calls: What is this session trying to accomplish?]
`
	handoffPath := filepath.Join(tmpDir, "SESSION_HANDOFF.md")
	if err := os.WriteFile(handoffPath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test handoff: %v", err)
	}

	summary := &sessionSummary{
		Outcome: "partial",
		Summary: "", // Empty summary
	}
	endTime := "2026-01-14 16:00"

	if err := updateHandoffTemplate(tmpDir, summary, endTime); err != nil {
		t.Fatalf("updateHandoffTemplate failed: %v", err)
	}

	updatedContent, err := os.ReadFile(handoffPath)
	if err != nil {
		t.Fatalf("Failed to read updated handoff: %v", err)
	}

	updated := string(updatedContent)

	// End time should still be replaced
	if strings.Contains(updated, "{end-time}") {
		t.Error("Expected {end-time} to be replaced")
	}

	// Outcome should be replaced
	if !strings.Contains(updated, "partial") {
		t.Error("Expected outcome to be 'partial'")
	}

	// TLDR placeholder should remain when no summary provided
	if !strings.Contains(updated, "[Fill within first 5 tool calls: What is this session trying to accomplish?]") {
		t.Error("Expected TLDR placeholder to remain when no summary provided")
	}
}
