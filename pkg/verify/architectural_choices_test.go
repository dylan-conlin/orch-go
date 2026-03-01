package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyArchitecturalChoices(t *testing.T) {
	tests := []struct {
		name       string
		skill      string
		content    string
		wantPassed bool
		wantError  bool // expect error in Errors list, not Go error
	}{
		{
			name:  "feature-impl with architectural choices present",
			skill: "feature-impl",
			content: `# Session Synthesis

## TLDR

Built the thing.

---

## Architectural Choices

### Chose X over Y
- **What I chose:** X
- **What I rejected:** Y
- **Why:** Simpler
- **Risk accepted:** Slightly slower

---

## Knowledge (What Was Learned)

Done.
`,
			wantPassed: true,
		},
		{
			name:  "feature-impl with no-choices declaration",
			skill: "feature-impl",
			content: `# Session Synthesis

## TLDR

Simple change.

---

## Architectural Choices

No architectural choices — task was within existing patterns.

---

## Knowledge (What Was Learned)

Done.
`,
			wantPassed: true,
		},
		{
			name:  "feature-impl missing architectural choices section",
			skill: "feature-impl",
			content: `# Session Synthesis

## TLDR

Built something.

---

## Knowledge (What Was Learned)

Done.
`,
			wantPassed: false,
			wantError:  true,
		},
		{
			name:  "architect with architectural choices present",
			skill: "architect",
			content: `# Session Synthesis

## TLDR

Designed the thing.

---

## Architectural Choices

### Chose approach A
- **What I chose:** A
- **What I rejected:** B
- **Why:** Better fit
- **Risk accepted:** More complexity

---
`,
			wantPassed: true,
		},
		{
			name:  "systematic-debugging with missing choices",
			skill: "systematic-debugging",
			content: `# Session Synthesis

## TLDR

Debugged something.

---

## Knowledge (What Was Learned)

Found root cause.
`,
			wantPassed: false,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")
			if err := os.WriteFile(synthesisPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write SYNTHESIS.md: %v", err)
			}

			result := VerifyArchitecturalChoices(tmpDir, tt.skill)

			if result.Passed != tt.wantPassed {
				t.Errorf("Passed = %v, want %v (errors: %v)", result.Passed, tt.wantPassed, result.Errors)
			}

			if tt.wantError && len(result.Errors) == 0 {
				t.Error("expected errors in result, got none")
			}

			if !tt.wantError && len(result.Errors) > 0 {
				t.Errorf("expected no errors, got: %v", result.Errors)
			}
		})
	}
}

func TestVerifyArchitecturalChoices_NoSynthesis(t *testing.T) {
	// When SYNTHESIS.md doesn't exist, the gate passes —
	// the synthesis gate handles the missing file separately.
	tmpDir := t.TempDir()
	result := VerifyArchitecturalChoices(tmpDir, "feature-impl")
	if !result.Passed {
		t.Errorf("Expected pass when SYNTHESIS.md missing, got errors: %v", result.Errors)
	}
}

func TestExtractArchitecturalChoicesContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string // non-empty means has content
	}{
		{
			name: "has structured choices",
			content: `## Architectural Choices

### Chose direct query
- **What I chose:** Direct API query
- **What I rejected:** Local cache
- **Why:** Drift prevention
- **Risk accepted:** Higher latency`,
			want: "Chose direct query",
		},
		{
			name:    "no choices section",
			content: "## Knowledge\n\nSomething.",
			want:    "",
		},
		{
			name: "no-choices declaration",
			content: `## Architectural Choices

No architectural choices — task was within existing patterns.`,
			want: "No architectural choices",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractArchitecturalChoicesContent(tt.content)
			if tt.want == "" {
				if got != "" {
					t.Errorf("expected empty, got %q", got)
				}
			} else {
				if got == "" {
					t.Errorf("expected content containing %q, got empty", tt.want)
				}
			}
		})
	}
}
