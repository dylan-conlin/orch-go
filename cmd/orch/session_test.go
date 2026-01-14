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

func TestParseDurationFromHandoff(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name: "full timestamp format - 8 hours",
			content: `# Session Handoff

**Orchestrator:** test-session
**Focus:** Test focus
**Duration:** 2026-01-14 12:54 → 2026-01-14 20:54
**Outcome:** success
`,
			expected: 480, // 8 hours = 480 minutes
		},
		{
			name: "full timestamp format - 38 minutes",
			content: `# Session Handoff

**Duration:** 2026-01-14 11:52 → 2026-01-14 12:30
**Outcome:** success
`,
			expected: 38,
		},
		{
			name: "same-day format with time only on end",
			content: `# Session Handoff

**Duration:** 2026-01-14 11:52 → 12:30 (38m)
**Outcome:** success
`,
			expected: 38,
		},
		{
			name: "incomplete session with placeholder",
			content: `# Session Handoff

**Duration:** 2026-01-14 07:29 → {end-time}
**Outcome:** {success | partial | blocked | failed}
`,
			expected: -1, // Can't parse placeholder
		},
		{
			name: "legacy format with seconds",
			content: `# Session Handoff

**Duration:** 3.296167s
**Outcome:** success
`,
			expected: -1, // Legacy format not supported
		},
		{
			name: "no duration line",
			content: `# Session Handoff

**Orchestrator:** test-session
**Focus:** Test focus
**Outcome:** success
`,
			expected: -1,
		},
		{
			name: "short session - 2 minutes",
			content: `# Session Handoff

**Duration:** 2026-01-14 15:00 → 2026-01-14 15:02
**Outcome:** success
`,
			expected: 2,
		},
		{
			name: "exactly 5 minutes",
			content: `# Session Handoff

**Duration:** 2026-01-14 15:00 → 2026-01-14 15:05
**Outcome:** success
`,
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file with content
			tmpDir := t.TempDir()
			handoffPath := filepath.Join(tmpDir, "SESSION_HANDOFF.md")
			if err := os.WriteFile(handoffPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test handoff: %v", err)
			}

			result := parseDurationFromHandoff(handoffPath)
			if result != tt.expected {
				t.Errorf("parseDurationFromHandoff() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestScanAllWindowsForMostRecent_DurationAware(t *testing.T) {
	// Create test session directory structure
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, ".orch", "session")

	// Create window directories with sessions
	// Window 1: Substantive session (30 minutes) - older timestamp
	window1Dir := filepath.Join(sessionDir, "window1", "2026-01-14-1000")
	if err := os.MkdirAll(window1Dir, 0755); err != nil {
		t.Fatalf("Failed to create window1 dir: %v", err)
	}
	// Create latest symlink
	if err := os.Symlink("2026-01-14-1000", filepath.Join(sessionDir, "window1", "latest")); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}
	substantiveContent := `# Session Handoff

**Duration:** 2026-01-14 10:00 → 2026-01-14 10:30
**Outcome:** success

## TLDR
Real work session with substantive content.
`
	if err := os.WriteFile(filepath.Join(window1Dir, "SESSION_HANDOFF.md"), []byte(substantiveContent), 0644); err != nil {
		t.Fatalf("Failed to write substantive handoff: %v", err)
	}

	// Window 2: Brief test session (2 minutes) - newer timestamp
	window2Dir := filepath.Join(sessionDir, "window2", "2026-01-14-1100")
	if err := os.MkdirAll(window2Dir, 0755); err != nil {
		t.Fatalf("Failed to create window2 dir: %v", err)
	}
	if err := os.Symlink("2026-01-14-1100", filepath.Join(sessionDir, "window2", "latest")); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}
	briefContent := `# Session Handoff

**Duration:** 2026-01-14 11:00 → 2026-01-14 11:02
**Outcome:** success

## TLDR
Quick test session.
`
	if err := os.WriteFile(filepath.Join(window2Dir, "SESSION_HANDOFF.md"), []byte(briefContent), 0644); err != nil {
		t.Fatalf("Failed to write brief handoff: %v", err)
	}

	// Run the scan
	result, err := scanAllWindowsForMostRecent(sessionDir)
	if err != nil {
		t.Fatalf("scanAllWindowsForMostRecent failed: %v", err)
	}

	// Should prefer substantive session over brief test session despite older timestamp
	expectedPath := filepath.Join(window1Dir, "SESSION_HANDOFF.md")
	if result != expectedPath {
		t.Errorf("Expected substantive session %q, got %q", expectedPath, result)
	}
}

func TestScanAllWindowsForMostRecent_FallbackToAny(t *testing.T) {
	// Create test session directory with only brief sessions
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, ".orch", "session")

	// Window 1: Brief session (2 minutes)
	window1Dir := filepath.Join(sessionDir, "test1", "2026-01-14-1000")
	if err := os.MkdirAll(window1Dir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	if err := os.Symlink("2026-01-14-1000", filepath.Join(sessionDir, "test1", "latest")); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}
	briefContent1 := `# Session Handoff

**Duration:** 2026-01-14 10:00 → 2026-01-14 10:02
**Outcome:** success
`
	if err := os.WriteFile(filepath.Join(window1Dir, "SESSION_HANDOFF.md"), []byte(briefContent1), 0644); err != nil {
		t.Fatalf("Failed to write handoff: %v", err)
	}

	// Window 2: Brief session (3 minutes) - newer timestamp
	window2Dir := filepath.Join(sessionDir, "test2", "2026-01-14-1100")
	if err := os.MkdirAll(window2Dir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	if err := os.Symlink("2026-01-14-1100", filepath.Join(sessionDir, "test2", "latest")); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}
	briefContent2 := `# Session Handoff

**Duration:** 2026-01-14 11:00 → 2026-01-14 11:03
**Outcome:** success
`
	if err := os.WriteFile(filepath.Join(window2Dir, "SESSION_HANDOFF.md"), []byte(briefContent2), 0644); err != nil {
		t.Fatalf("Failed to write handoff: %v", err)
	}

	// Run the scan
	result, err := scanAllWindowsForMostRecent(sessionDir)
	if err != nil {
		t.Fatalf("scanAllWindowsForMostRecent failed: %v", err)
	}

	// When all sessions are brief, should fall back to most recent any
	expectedPath := filepath.Join(window2Dir, "SESSION_HANDOFF.md")
	if result != expectedPath {
		t.Errorf("Expected most recent brief session %q, got %q", expectedPath, result)
	}
}
