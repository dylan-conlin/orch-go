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

func TestExtractPhases(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantNil    bool
		wantCount  int
		wantPhases []PhaseInfo // spot-check specific phases
	}{
		{
			name:    "no phases returns nil",
			content: "Simple design with no phases.",
			wantNil: true,
		},
		{
			name:    "single phase returns nil (need 2+)",
			content: "### Phase 1: Implement the handler\nDo the thing.",
			wantNil: true,
		},
		{
			name: "three phases with headings",
			content: `### Phase 1: Add parser
Parse incoming data.

### Phase 2: Add validator
Validate data against schema.

### Phase 3: Wire tests
End-to-end verification.`,
			wantCount: 3,
			wantPhases: []PhaseInfo{
				{Number: 1, Title: "Add parser"},
				{Number: 2, Title: "Add validator"},
				{Number: 3, Title: "Wire tests"},
			},
		},
		{
			name: "bold format phases",
			content: `**Phase 1:** Data access layer
Build the DAL.

**Phase 2:** Business logic
Implement core logic.

**Phase 3:** API surface
Wire the endpoints.`,
			wantCount: 3,
			wantPhases: []PhaseInfo{
				{Number: 1, Title: "Data access layer"},
				{Number: 2, Title: "Business logic"},
				{Number: 3, Title: "API surface"},
			},
		},
		{
			name: "layers keyword",
			content: `### Layer 1: Persistence
Database operations.

### Layer 2: Domain
Business rules.`,
			wantCount: 2,
			wantPhases: []PhaseInfo{
				{Number: 1, Title: "Persistence"},
				{Number: 2, Title: "Domain"},
			},
		},
		{
			name: "steps keyword",
			content: `## Step 1: Extract
Pull data out.

## Step 2: Transform
Process data.

## Step 3: Load
Push data in.`,
			wantCount: 3,
		},
		{
			name: "duplicate phase numbers keep first",
			content: `### Phase 1: Start here
First attempt.

### Phase 1: Also start here
Duplicate.

### Phase 2: Continue
Next step.`,
			wantCount: 2,
			wantPhases: []PhaseInfo{
				{Number: 1, Title: "Start here"},
				{Number: 2, Title: "Continue"},
			},
		},
		{
			name: "descriptions captured between phases",
			content: `### Phase 1: Add parser
Parse incoming data.
Handle edge cases.

### Phase 2: Add validator
Validate against schema.`,
			wantCount: 2,
		},
		{
			name: "phases within Next section context",
			content: `## TLDR
Designed a pipeline.

## Next
**Recommendation:** implement

### Phase 1: Add data parser
Extract and normalize incoming data.

### Phase 2: Add validation layer
Validate normalized data against schema.

### Phase 3: Wire integration tests
End-to-end verification of the pipeline.`,
			wantCount: 3,
			wantPhases: []PhaseInfo{
				{Number: 1, Title: "Add data parser"},
				{Number: 2, Title: "Add validation layer"},
				{Number: 3, Title: "Wire integration tests"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPhases(tt.content)
			if tt.wantNil {
				if got != nil {
					t.Errorf("ExtractPhases() = %v, want nil", got)
				}
				return
			}
			if len(got) != tt.wantCount {
				t.Errorf("ExtractPhases() returned %d phases, want %d", len(got), tt.wantCount)
				return
			}
			for i, want := range tt.wantPhases {
				if i >= len(got) {
					break
				}
				if got[i].Number != want.Number {
					t.Errorf("phase[%d].Number = %d, want %d", i, got[i].Number, want.Number)
				}
				if got[i].Title != want.Title {
					t.Errorf("phase[%d].Title = %q, want %q", i, got[i].Title, want.Title)
				}
			}
		})
	}
}

func TestExtractPhases_DescriptionContent(t *testing.T) {
	content := `### Phase 1: Add parser
Parse incoming data.
Handle edge cases.

### Phase 2: Add validator
Validate against schema.
Return errors.`

	phases := ExtractPhases(content)
	if len(phases) != 2 {
		t.Fatalf("expected 2 phases, got %d", len(phases))
	}

	// Phase 1 description should include its content but not phase 2's
	if !strings.Contains(phases[0].Description, "Parse incoming data") {
		t.Errorf("phase 1 description should contain 'Parse incoming data', got %q", phases[0].Description)
	}
	if strings.Contains(phases[0].Description, "Validate against schema") {
		t.Errorf("phase 1 description should NOT contain phase 2 content, got %q", phases[0].Description)
	}

	// Phase 2 description should include its content
	if !strings.Contains(phases[1].Description, "Validate against schema") {
		t.Errorf("phase 2 description should contain 'Validate against schema', got %q", phases[1].Description)
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
