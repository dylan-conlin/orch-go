package skills

import (
	"strings"
	"testing"
)

func TestFilterSkillSections_NilFilter(t *testing.T) {
	content := "# Heading\n\nSome content here.\n"
	got := FilterSkillSections(content, nil)
	if got != content {
		t.Errorf("nil filter should return content unchanged\ngot:  %q\nwant: %q", got, content)
	}
}

func TestFilterSkillSections_EmptyFilter(t *testing.T) {
	content := "# Heading\n\nSome content here.\n"
	got := FilterSkillSections(content, &SectionFilter{})
	if got != content {
		t.Errorf("empty filter should return content unchanged\ngot:  %q\nwant: %q", got, content)
	}
}

func TestFilterSkillSections_PhaseFiltering(t *testing.T) {
	content := `# Skill

## Always included

<!-- @section: phase=investigation -->
### Investigation Phase

Investigation content here.
<!-- @/section -->

<!-- @section: phase=implementation -->
### Implementation Phase

Implementation content here.
<!-- @/section -->

<!-- @section: phase=validation -->
### Validation Phase

Validation content here.
<!-- @/section -->

## Also always included
`

	filter := &SectionFilter{Phases: []string{"implementation", "validation"}}
	got := FilterSkillSections(content, filter)

	if strings.Contains(got, "Investigation Phase") {
		t.Error("expected investigation phase to be filtered out")
	}
	if !strings.Contains(got, "Implementation Phase") {
		t.Error("expected implementation phase to be included")
	}
	if !strings.Contains(got, "Validation Phase") {
		t.Error("expected validation phase to be included")
	}
	if !strings.Contains(got, "Always included") {
		t.Error("expected unmarked content to be preserved")
	}
	if !strings.Contains(got, "Also always included") {
		t.Error("expected trailing unmarked content to be preserved")
	}
	// Markers should be stripped
	if strings.Contains(got, "@section") {
		t.Error("expected section markers to be stripped from output")
	}
}

func TestFilterSkillSections_ModeFiltering(t *testing.T) {
	content := `# Implementation

<!-- @section: phase=implementation, mode=tdd -->
### TDD Mode

Write tests first.
<!-- @/section -->

<!-- @section: phase=implementation, mode=direct -->
### Direct Mode

Implement directly.
<!-- @/section -->

<!-- @section: phase=implementation, mode=verification-first -->
### Verification-First Mode

Spec first.
<!-- @/section -->
`

	filter := &SectionFilter{
		Phases: []string{"implementation"},
		Mode:   "tdd",
	}
	got := FilterSkillSections(content, filter)

	if !strings.Contains(got, "TDD Mode") {
		t.Error("expected TDD mode to be included")
	}
	if strings.Contains(got, "Direct Mode") {
		t.Error("expected direct mode to be filtered out")
	}
	if strings.Contains(got, "Verification-First Mode") {
		t.Error("expected verification-first mode to be filtered out")
	}
}

func TestFilterSkillSections_SpawnModeFiltering(t *testing.T) {
	content := `# Architect

## Always included

<!-- @section: spawn-mode=autonomous -->
### Autonomous Mode

Run without interaction.
<!-- @/section -->

<!-- @section: spawn-mode=interactive -->
### Interactive Mode

Discuss with user.
<!-- @/section -->

## Footer
`

	filter := &SectionFilter{SpawnMode: "autonomous"}
	got := FilterSkillSections(content, filter)

	if !strings.Contains(got, "Autonomous Mode") {
		t.Error("expected autonomous mode to be included")
	}
	if strings.Contains(got, "Interactive Mode") {
		t.Error("expected interactive mode to be filtered out")
	}
	if !strings.Contains(got, "Always included") {
		t.Error("expected unmarked content to be preserved")
	}
	if !strings.Contains(got, "Footer") {
		t.Error("expected footer to be preserved")
	}
}

func TestFilterSkillSections_MalformedMarkers(t *testing.T) {
	content := `# Heading

<!-- @section: this has no closing angle bracket
Content that follows malformed marker.

<!-- @section: -->
Content after marker with no key-value.
<!-- @/section -->

Some normal content.
`

	// Malformed open marker (no -->) should be passed through as regular content
	filter := &SectionFilter{Phases: []string{"implementation"}}
	got := FilterSkillSections(content, filter)

	if !strings.Contains(got, "Content that follows malformed marker") {
		t.Error("expected content after malformed marker to be preserved")
	}
	if !strings.Contains(got, "Some normal content") {
		t.Error("expected normal content to be preserved")
	}
}

func TestFilterSkillSections_PhaseOnlySectionNoModeFilter(t *testing.T) {
	// A section with only phase=implementation should be included when
	// filter has Mode set but the section doesn't specify mode
	content := `# Skill

<!-- @section: phase=implementation -->
### Harm Assessment

Pre-implementation checkpoint.
<!-- @/section -->

<!-- @section: phase=implementation, mode=tdd -->
### TDD Mode

TDD content.
<!-- @/section -->
`

	filter := &SectionFilter{
		Phases: []string{"implementation"},
		Mode:   "tdd",
	}
	got := FilterSkillSections(content, filter)

	if !strings.Contains(got, "Harm Assessment") {
		t.Error("expected phase-only section (no mode attr) to be included")
	}
	if !strings.Contains(got, "TDD Mode") {
		t.Error("expected matching phase+mode section to be included")
	}
}

func TestFilterSkillSections_NoMarkersUnchanged(t *testing.T) {
	content := `# Simple Skill

This skill has no section markers.

## Section One

Content one.

## Section Two

Content two.
`

	filter := &SectionFilter{Phases: []string{"implementation"}, Mode: "tdd"}
	got := FilterSkillSections(content, filter)

	if got != content {
		t.Errorf("content without markers should be unchanged\ngot:  %q\nwant: %q", got, content)
	}
}

func TestFilterSkillSections_CollapsesBlankLines(t *testing.T) {
	// When multiple adjacent sections are removed, blank lines collapse
	content := "# Header\n\n<!-- @section: phase=a -->\nA content\n<!-- @/section -->\n\n<!-- @section: phase=b -->\nB content\n<!-- @/section -->\n\n<!-- @section: phase=c -->\nC content\n<!-- @/section -->\n\n# Footer\n"

	filter := &SectionFilter{Phases: []string{"c"}}
	got := FilterSkillSections(content, filter)

	if !strings.Contains(got, "C content") {
		t.Error("expected phase c to be included")
	}
	if strings.Contains(got, "A content") || strings.Contains(got, "B content") {
		t.Error("expected phases a and b to be filtered out")
	}
	// Should not have more than 3 consecutive newlines
	if strings.Contains(got, "\n\n\n\n") {
		t.Error("expected blank lines to be collapsed (max 3 consecutive newlines)")
	}
}

func TestParseSectionAttrs(t *testing.T) {
	tests := []struct {
		name   string
		marker string
		want   map[string]string
	}{
		{
			name:   "single key-value",
			marker: "<!-- @section: phase=investigation -->",
			want:   map[string]string{"phase": "investigation"},
		},
		{
			name:   "multiple key-values",
			marker: "<!-- @section: phase=implementation, mode=tdd -->",
			want:   map[string]string{"phase": "implementation", "mode": "tdd"},
		},
		{
			name:   "spawn-mode",
			marker: "<!-- @section: spawn-mode=autonomous -->",
			want:   map[string]string{"spawn-mode": "autonomous"},
		},
		{
			name:   "extra whitespace",
			marker: "<!-- @section:  phase = investigation , mode = tdd  -->",
			want:   map[string]string{"phase": "investigation", "mode": "tdd"},
		},
		{
			name:   "empty marker",
			marker: "<!-- @section: -->",
			want:   map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSectionAttrs(tt.marker)
			if len(got) != len(tt.want) {
				t.Errorf("parseSectionAttrs() returned %d attrs, want %d: %v", len(got), len(tt.want), got)
				return
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("parseSectionAttrs()[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestSectionFilter_IsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		filter SectionFilter
		want   bool
	}{
		{"zero value", SectionFilter{}, true},
		{"phases only", SectionFilter{Phases: []string{"a"}}, false},
		{"mode only", SectionFilter{Mode: "tdd"}, false},
		{"spawn-mode only", SectionFilter{SpawnMode: "autonomous"}, false},
		{"empty phases slice", SectionFilter{Phases: []string{}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}
