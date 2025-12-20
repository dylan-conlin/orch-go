package skills

import (
	"os"
	"path/filepath"
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
