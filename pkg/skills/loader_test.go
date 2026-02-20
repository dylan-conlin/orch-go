package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindSkillPath(t *testing.T) {
	// Create temp skills directory for testing
	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, ".claude", "skills")

	// Create test skill structure: worker/investigation/SKILL.md
	investigationDir := filepath.Join(skillsDir, "worker", "investigation")
	if err := os.MkdirAll(investigationDir, 0755); err != nil {
		t.Fatalf("failed to create investigation dir: %v", err)
	}

	skillContent := `---
name: investigation
skill-type: procedure
audience: worker
---

# Investigation Skill

This is a test skill.
`
	skillPath := filepath.Join(investigationDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0644); err != nil {
		t.Fatalf("failed to write skill file: %v", err)
	}

	// Create another skill for testing: worker/feature-impl/SKILL.md
	featureImplDir := filepath.Join(skillsDir, "worker", "feature-impl")
	if err := os.MkdirAll(featureImplDir, 0755); err != nil {
		t.Fatalf("failed to create feature-impl dir: %v", err)
	}

	featureImplContent := `---
name: feature-impl
skill-type: procedure
---

# Feature Implementation
`
	featureImplPath := filepath.Join(featureImplDir, "SKILL.md")
	if err := os.WriteFile(featureImplPath, []byte(featureImplContent), 0644); err != nil {
		t.Fatalf("failed to write feature-impl skill: %v", err)
	}

	// Create a symlink skill (like real skills directory)
	symlinkPath := filepath.Join(skillsDir, "investigation")
	if err := os.Symlink(filepath.Join("worker", "investigation"), symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	tests := []struct {
		name      string
		skillName string
		wantFound bool
	}{
		{
			name:      "find skill via symlink",
			skillName: "investigation",
			wantFound: true,
		},
		{
			name:      "find skill in subdirectory",
			skillName: "feature-impl",
			wantFound: true,
		},
		{
			name:      "skill not found",
			skillName: "nonexistent",
			wantFound: false,
		},
	}

	loader := NewLoader(skillsDir)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := loader.FindSkillPath(tt.skillName)
			if tt.wantFound {
				if err != nil {
					t.Errorf("expected to find skill, got error: %v", err)
				}
				if path == "" {
					t.Error("expected non-empty path")
				}
			} else {
				if err == nil {
					t.Error("expected error for nonexistent skill")
				}
			}
		})
	}
}

func TestLoadSkillContent(t *testing.T) {
	// Create temp skills directory
	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, ".claude", "skills")
	investigationDir := filepath.Join(skillsDir, "worker", "investigation")
	if err := os.MkdirAll(investigationDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	expectedContent := `---
name: investigation
---

# Investigation Skill

Test content here.
`
	skillPath := filepath.Join(investigationDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(expectedContent), 0644); err != nil {
		t.Fatalf("failed to write skill: %v", err)
	}

	// Create symlink
	symlinkPath := filepath.Join(skillsDir, "investigation")
	if err := os.Symlink(filepath.Join("worker", "investigation"), symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	loader := NewLoader(skillsDir)

	content, err := loader.LoadSkillContent("investigation")
	if err != nil {
		t.Fatalf("failed to load skill content: %v", err)
	}

	if content != expectedContent {
		t.Errorf("content mismatch:\ngot: %q\nwant: %q", content, expectedContent)
	}
}

func TestParseSkillMetadata(t *testing.T) {
	content := `---
name: feature-impl
skill-type: procedure
audience: worker
spawnable: true
category: implementation
description: Unified feature implementation with configurable phases.

deliverables:
  investigation:
    required: false
    description: "Investigation file"

verification:
  requirements:
    - "All tests pass"
    - "Implementation complete"
---

# Feature Implementation

Content here.
`

	metadata, err := ParseSkillMetadata(content)
	if err != nil {
		t.Fatalf("failed to parse metadata: %v", err)
	}

	if metadata.Name != "feature-impl" {
		t.Errorf("name: got %q, want %q", metadata.Name, "feature-impl")
	}
	if metadata.SkillType != "procedure" {
		t.Errorf("skill-type: got %q, want %q", metadata.SkillType, "procedure")
	}
	if metadata.Audience != "worker" {
		t.Errorf("audience: got %q, want %q", metadata.Audience, "worker")
	}
	if !metadata.Spawnable {
		t.Error("expected spawnable to be true")
	}
	if metadata.Description == "" {
		t.Error("expected description to be set")
	}
}

func TestParseSkillMetadata_InvalidYAML(t *testing.T) {
	content := `No frontmatter here, just plain text`

	_, err := ParseSkillMetadata(content)
	if err == nil {
		t.Error("expected error for content without frontmatter")
	}
}

func TestParseSkillMetadata_WithDependencies(t *testing.T) {
	content := `---
name: investigation
skill-type: procedure
audience: worker
spawnable: true
dependencies:
  - worker-base
---

# Investigation Skill

Content here.
`

	metadata, err := ParseSkillMetadata(content)
	if err != nil {
		t.Fatalf("failed to parse metadata: %v", err)
	}

	if len(metadata.Dependencies) != 1 {
		t.Errorf("dependencies: got %d, want 1", len(metadata.Dependencies))
	}
	if metadata.Dependencies[0] != "worker-base" {
		t.Errorf("dependency: got %q, want %q", metadata.Dependencies[0], "worker-base")
	}
}

func TestLoadSkillWithDependencies(t *testing.T) {
	// Create temp skills directory
	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, ".claude", "skills")

	// Create worker-base skill (the dependency)
	workerBaseDir := filepath.Join(skillsDir, "shared", "worker-base")
	if err := os.MkdirAll(workerBaseDir, 0755); err != nil {
		t.Fatalf("failed to create worker-base dir: %v", err)
	}

	workerBaseContent := `---
name: worker-base
skill-type: foundation
composable: true
---

## Authority Delegation

You have authority to decide implementation details.

## Beads Progress Tracking

Use bd comment for progress updates.
`
	if err := os.WriteFile(filepath.Join(workerBaseDir, "SKILL.md"), []byte(workerBaseContent), 0644); err != nil {
		t.Fatalf("failed to write worker-base skill: %v", err)
	}

	// Create investigation skill that depends on worker-base
	investigationDir := filepath.Join(skillsDir, "worker", "investigation")
	if err := os.MkdirAll(investigationDir, 0755); err != nil {
		t.Fatalf("failed to create investigation dir: %v", err)
	}

	investigationContent := `---
name: investigation
skill-type: procedure
audience: worker
spawnable: true
dependencies:
  - worker-base
---

# Investigation Skill

This is the investigation skill content.
`
	if err := os.WriteFile(filepath.Join(investigationDir, "SKILL.md"), []byte(investigationContent), 0644); err != nil {
		t.Fatalf("failed to write investigation skill: %v", err)
	}

	// Create symlinks for discovery
	if err := os.Symlink(filepath.Join("shared", "worker-base"), filepath.Join(skillsDir, "worker-base")); err != nil {
		t.Fatalf("failed to create worker-base symlink: %v", err)
	}
	if err := os.Symlink(filepath.Join("worker", "investigation"), filepath.Join(skillsDir, "investigation")); err != nil {
		t.Fatalf("failed to create investigation symlink: %v", err)
	}

	loader := NewLoader(skillsDir)

	// Load investigation with dependencies
	content, err := loader.LoadSkillWithDependencies("investigation")
	if err != nil {
		t.Fatalf("failed to load skill with dependencies: %v", err)
	}

	// Verify worker-base content is included
	if !strings.Contains(content, "Authority Delegation") {
		t.Error("expected worker-base content (Authority Delegation) to be included")
	}
	if !strings.Contains(content, "Beads Progress Tracking") {
		t.Error("expected worker-base content (Beads Progress Tracking) to be included")
	}

	// Verify investigation content is included
	if !strings.Contains(content, "Investigation Skill") {
		t.Error("expected investigation skill content to be included")
	}

	// Verify worker-base content comes before investigation content
	authorityIdx := strings.Index(content, "Authority Delegation")
	investigationIdx := strings.Index(content, "Investigation Skill")
	if authorityIdx > investigationIdx {
		t.Error("expected worker-base content to come before investigation content")
	}
}

func TestLoadSkillWithDependencies_NoDependencies(t *testing.T) {
	// Create temp skills directory
	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, ".claude", "skills")

	// Create a skill without dependencies
	featureImplDir := filepath.Join(skillsDir, "worker", "feature-impl")
	if err := os.MkdirAll(featureImplDir, 0755); err != nil {
		t.Fatalf("failed to create feature-impl dir: %v", err)
	}

	featureImplContent := `---
name: feature-impl
skill-type: procedure
---

# Feature Implementation

Content here.
`
	if err := os.WriteFile(filepath.Join(featureImplDir, "SKILL.md"), []byte(featureImplContent), 0644); err != nil {
		t.Fatalf("failed to write feature-impl skill: %v", err)
	}

	// Create symlink
	if err := os.Symlink(filepath.Join("worker", "feature-impl"), filepath.Join(skillsDir, "feature-impl")); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	loader := NewLoader(skillsDir)

	content, err := loader.LoadSkillWithDependencies("feature-impl")
	if err != nil {
		t.Fatalf("failed to load skill: %v", err)
	}

	// Should just be the skill content itself
	if content != featureImplContent {
		t.Errorf("content mismatch for skill without dependencies")
	}
}

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

func TestStripFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "with frontmatter",
			content: `---
name: test
---

# Heading

Content here.
`,
			want: `# Heading

Content here.
`,
		},
		{
			name:    "without frontmatter",
			content: "# Just Content\n\nNo frontmatter.",
			want:    "# Just Content\n\nNo frontmatter.",
		},
		{
			name:    "incomplete frontmatter",
			content: "---\nname: test\nNo closing delimiter",
			want:    "---\nname: test\nNo closing delimiter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripFrontmatter(tt.content)
			if got != tt.want {
				t.Errorf("stripFrontmatter() = %q, want %q", got, tt.want)
			}
		})
	}
}
