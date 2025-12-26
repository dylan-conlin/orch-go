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
