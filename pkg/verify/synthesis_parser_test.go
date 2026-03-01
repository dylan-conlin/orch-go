package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractRecommendation(t *testing.T) {
	tests := []struct {
		name        string
		nextSection string
		want        string
	}{
		{
			name:        "simple word recommendation",
			nextSection: "**Recommendation:** close",
			want:        "close",
		},
		{
			name:        "hyphenated recommendation",
			nextSection: "**Recommendation:** spawn-follow-up",
			want:        "spawn-follow-up",
		},
		{
			name:        "escalate recommendation",
			nextSection: "**Recommendation:** escalate",
			want:        "escalate",
		},
		{
			name:        "resume recommendation",
			nextSection: "**Recommendation:** resume",
			want:        "resume",
		},
		{
			name:        "no recommendation",
			nextSection: "Nothing here",
			want:        "",
		},
		{
			name:        "recommendation in multiline context",
			nextSection: "Some intro text.\n\n**Recommendation:** spawn-follow-up\n\nMore details here.",
			want:        "spawn-follow-up",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRecommendation(tt.nextSection)
			if got != tt.want {
				t.Errorf("extractRecommendation() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseSynthesis_HyphenatedRecommendation(t *testing.T) {
	content := `# Session Synthesis

**Agent:** test-agent
**Issue:** orch-go-1234
**Outcome:** success

## TLDR

Fixed the widget.

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

- Create follow-up issue for remaining work
`
	tmpDir := t.TempDir()
	synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")
	if err := os.WriteFile(synthesisPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write SYNTHESIS.md: %v", err)
	}

	s, err := ParseSynthesis(tmpDir)
	if err != nil {
		t.Fatalf("ParseSynthesis failed: %v", err)
	}

	if s.Recommendation != "spawn-follow-up" {
		t.Errorf("Recommendation = %q, want %q", s.Recommendation, "spawn-follow-up")
	}
}

func TestParseSynthesis_ArchitecturalChoices(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string // expected ArchitecturalChoices content (non-empty)
	}{
		{
			name: "architectural choices section present",
			content: `# Session Synthesis

**Agent:** test-agent
**Issue:** orch-go-1234
**Duration:** 10:00 → 11:00
**Outcome:** success

## TLDR

Implemented caching layer for agent status.

---

## Evidence (What Was Observed)

- Found status queries taking 2s each

---

## Architectural Choices

### Chose direct query over local cache
- **What I chose:** Query OpenCode API directly for each status request
- **What I rejected:** Local in-memory cache with TTL
- **Why:** Caching creates drift risk (see 6-week registry cycle)
- **Risk accepted:** Slower status queries (~200ms vs ~5ms cached)

---

## Knowledge (What Was Learned)

### Decisions Made
- Decision 1: Direct query approach
`,
			want: "Chose direct query over local cache",
		},
		{
			name: "no architectural choices section",
			content: `# Session Synthesis

**Agent:** test-agent
**Outcome:** success

## TLDR

Simple bug fix.

---

## Evidence (What Was Observed)

- Found the bug

---

## Knowledge (What Was Learned)

Nothing special.
`,
			want: "",
		},
		{
			name: "architectural choices with no-choices declaration",
			content: `# Session Synthesis

**Agent:** test-agent
**Outcome:** success

## TLDR

Config change.

---

## Architectural Choices

No architectural choices — task was within existing patterns.

---

## Knowledge (What Was Learned)

Nothing.
`,
			want: "No architectural choices",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")
			if err := os.WriteFile(synthesisPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write SYNTHESIS.md: %v", err)
			}

			s, err := ParseSynthesis(tmpDir)
			if err != nil {
				t.Fatalf("ParseSynthesis failed: %v", err)
			}

			if tt.want == "" {
				if s.ArchitecturalChoices != "" {
					t.Errorf("expected empty ArchitecturalChoices, got %q", s.ArchitecturalChoices)
				}
			} else {
				if s.ArchitecturalChoices == "" {
					t.Error("expected non-empty ArchitecturalChoices, got empty")
				}
				if !strings.Contains(s.ArchitecturalChoices, tt.want) {
					t.Errorf("ArchitecturalChoices should contain %q, got %q", tt.want, s.ArchitecturalChoices)
				}
			}
		})
	}
}
