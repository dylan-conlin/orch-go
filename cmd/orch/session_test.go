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

func TestValidateHandoff(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a test handoff file with all placeholders unfilled
	testContent := `# Session Handoff

**Orchestrator:** test-session
**Focus:** Test focus
**Duration:** 2026-01-14 15:00 → {end-time}
**Outcome:** {success | partial | blocked | failed}

## TLDR

[Fill within first 5 tool calls: What is this session trying to accomplish?]

## Focus Progress

### Where We Ended
- {state of focus goal now}

## Next

**Recommendation:** {continue-focus | shift-focus | escalate | pause}

## Evidence

### Patterns Across Agents
- [Pattern 1: test pattern]

## Knowledge

### Decisions Made
- **{topic}:** decision

## Friction

### Tooling Friction
- [Tool gap or UX issue]
`
	handoffPath := filepath.Join(tmpDir, "SESSION_HANDOFF.md")
	if err := os.WriteFile(handoffPath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test handoff: %v", err)
	}

	// Validate the handoff
	result, err := validateHandoff(tmpDir)
	if err != nil {
		t.Fatalf("validateHandoff failed: %v", err)
	}

	// Should detect all 7 unfilled sections
	if len(result.Unfilled) != 7 {
		t.Errorf("Expected 7 unfilled sections, got %d", len(result.Unfilled))
	}

	// Verify specific sections are detected
	foundSections := make(map[string]bool)
	for _, section := range result.Unfilled {
		foundSections[section.Name] = true
	}

	expectedSections := []string{"Outcome", "TLDR", "Where We Ended", "Next Recommendation", "Evidence", "Knowledge", "Friction"}
	for _, expected := range expectedSections {
		if !foundSections[expected] {
			t.Errorf("Expected section %q to be detected as unfilled", expected)
		}
	}
}

func TestValidateHandoffPartiallyFilled(t *testing.T) {
	// Test a handoff with some sections already filled
	tmpDir := t.TempDir()

	testContent := `# Session Handoff

**Outcome:** success

## TLDR

Fixed the authentication bug and added integration tests.

## Focus Progress

### Where We Ended
- {state of focus goal now}

## Next

**Recommendation:** continue-focus
`
	handoffPath := filepath.Join(tmpDir, "SESSION_HANDOFF.md")
	if err := os.WriteFile(handoffPath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test handoff: %v", err)
	}

	result, err := validateHandoff(tmpDir)
	if err != nil {
		t.Fatalf("validateHandoff failed: %v", err)
	}

	// Should only detect 1 unfilled section (Where We Ended)
	if len(result.Unfilled) != 1 {
		t.Errorf("Expected 1 unfilled section, got %d", len(result.Unfilled))
	}

	if len(result.Unfilled) > 0 && result.Unfilled[0].Name != "Where We Ended" {
		t.Errorf("Expected 'Where We Ended' to be unfilled, got %q", result.Unfilled[0].Name)
	}
}

func TestUpdateHandoffWithResponses(t *testing.T) {
	content := `# Session Handoff

**Duration:** 2026-01-14 15:00 → {end-time}
**Outcome:** {success | partial | blocked | failed}

## TLDR

[Fill within first 5 tool calls: What is this session trying to accomplish?]
`

	// Create responses
	responses := []UserResponse{
		{
			Section:  handoffSections[0], // Outcome
			Response: "success",
		},
		{
			Section:  handoffSections[1], // TLDR
			Response: "Fixed the bug and added tests",
		},
	}

	endTime := "2026-01-14 16:00"
	updated := updateHandoffWithResponses(content, responses, endTime)

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

	// Verify TLDR placeholder was replaced
	if strings.Contains(updated, "[Fill within first 5 tool calls: What is this session trying to accomplish?]") {
		t.Error("Expected TLDR placeholder to be replaced, but it's still present")
	}
	if !strings.Contains(updated, "Fixed the bug and added tests") {
		t.Error("Expected updated content to contain the TLDR response")
	}
}

func TestHandoffSectionsDefinition(t *testing.T) {
	// Verify the handoff sections are properly defined
	if len(handoffSections) != 7 {
		t.Errorf("Expected 7 handoff sections, got %d", len(handoffSections))
	}

	// Verify required sections
	requiredSections := map[string]bool{
		"Outcome":             true,
		"TLDR":                true,
		"Where We Ended":      true,
		"Next Recommendation": true,
	}

	for _, section := range handoffSections {
		if required, exists := requiredSections[section.Name]; exists {
			if !section.Required {
				t.Errorf("Section %q should be required, but it's not", section.Name)
			}
			if !required {
				t.Errorf("Section %q marked as not required in test data", section.Name)
			}
		} else {
			// Optional section
			if section.Required {
				t.Errorf("Section %q should be optional, but it's marked as required", section.Name)
			}
			if section.SkipValue == "" {
				t.Errorf("Optional section %q should have a skip value", section.Name)
			}
		}
	}
}

func TestHandoffValidationWithOptions(t *testing.T) {
	// Verify that choice-based sections have options defined
	for _, section := range handoffSections {
		if section.Name == "Outcome" && len(section.Options) != 4 {
			t.Errorf("Outcome section should have 4 options, got %d", len(section.Options))
		}
		if section.Name == "Next Recommendation" && len(section.Options) != 4 {
			t.Errorf("Next Recommendation section should have 4 options, got %d", len(section.Options))
		}
	}
}
