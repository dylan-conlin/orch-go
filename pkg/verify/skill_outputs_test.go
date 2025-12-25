package verify

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExtractSkillNameFromSpawnContext(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "SKILL GUIDANCE pattern",
			content: `TASK: Do something

## SKILL GUIDANCE (feature-impl)

**IMPORTANT:** You have been spawned...`,
			expected: "feature-impl",
		},
		{
			name: "SKILL GUIDANCE with investigation",
			content: `## SKILL GUIDANCE (investigation)

Follow the investigation skill.`,
			expected: "investigation",
		},
		{
			name: "name in YAML front matter",
			content: `---
name: systematic-debugging
skill-type: procedure
---`,
			expected: "systematic-debugging",
		},
		{
			name: "no skill found",
			content: `TASK: Do something

Some random content.`,
			expected: "",
		},
		{
			name:     "skill guidance case insensitive",
			content:  `## skill guidance (architect)`,
			expected: "architect",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory with SPAWN_CONTEXT.md
			tmpDir := t.TempDir()
			spawnContextPath := filepath.Join(tmpDir, "SPAWN_CONTEXT.md")
			if err := os.WriteFile(spawnContextPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write SPAWN_CONTEXT.md: %v", err)
			}

			got, err := ExtractSkillNameFromSpawnContext(tmpDir)
			if err != nil {
				t.Fatalf("ExtractSkillNameFromSpawnContext() error = %v", err)
			}
			if got != tt.expected {
				t.Errorf("ExtractSkillNameFromSpawnContext() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExtractSkillNameFromSpawnContext_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	got, err := ExtractSkillNameFromSpawnContext(tmpDir)
	if err != nil {
		t.Fatalf("ExtractSkillNameFromSpawnContext() error = %v", err)
	}
	if got != "" {
		t.Errorf("ExtractSkillNameFromSpawnContext() = %q, want empty string for missing file", got)
	}
}

func TestParseSkillManifest(t *testing.T) {
	content := `name: investigation
skill-type: procedure
outputs:
  required:
    - pattern: ".kb/investigations/{date}-inv-*.md"
      description: "Investigation file with findings"
`

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "skill.yaml")
	if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write skill.yaml: %v", err)
	}

	manifest, err := ParseSkillManifest(manifestPath)
	if err != nil {
		t.Fatalf("ParseSkillManifest() error = %v", err)
	}

	if manifest.Name != "investigation" {
		t.Errorf("manifest.Name = %q, want %q", manifest.Name, "investigation")
	}

	if len(manifest.Outputs.Required) != 1 {
		t.Fatalf("len(manifest.Outputs.Required) = %d, want 1", len(manifest.Outputs.Required))
	}

	output := manifest.Outputs.Required[0]
	if output.Pattern != ".kb/investigations/{date}-inv-*.md" {
		t.Errorf("output.Pattern = %q, want %q", output.Pattern, ".kb/investigations/{date}-inv-*.md")
	}
	if output.Description != "Investigation file with findings" {
		t.Errorf("output.Description = %q, want %q", output.Description, "Investigation file with findings")
	}
}

func TestParseSkillManifest_NoOutputs(t *testing.T) {
	content := `name: feature-impl
skill-type: procedure
deliverables:
  tests:
    required: false
`

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "skill.yaml")
	if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write skill.yaml: %v", err)
	}

	manifest, err := ParseSkillManifest(manifestPath)
	if err != nil {
		t.Fatalf("ParseSkillManifest() error = %v", err)
	}

	if manifest.Name != "feature-impl" {
		t.Errorf("manifest.Name = %q, want %q", manifest.Name, "feature-impl")
	}

	if len(manifest.Outputs.Required) != 0 {
		t.Errorf("len(manifest.Outputs.Required) = %d, want 0", len(manifest.Outputs.Required))
	}
}

func TestVerifySkillOutputs_MatchFound(t *testing.T) {
	// Create temp project directory
	projectDir := t.TempDir()

	// Create the required output file
	kbDir := filepath.Join(projectDir, ".kb", "investigations")
	if err := os.MkdirAll(kbDir, 0755); err != nil {
		t.Fatalf("failed to create .kb/investigations: %v", err)
	}
	invFile := filepath.Join(kbDir, "2025-12-25-inv-test-investigation.md")
	if err := os.WriteFile(invFile, []byte("# Investigation"), 0644); err != nil {
		t.Fatalf("failed to write investigation file: %v", err)
	}

	manifest := &SkillManifest{
		Name: "investigation",
		Outputs: SkillOutputs{
			Required: []SkillOutput{
				{
					Pattern:     ".kb/investigations/{date}-inv-*.md",
					Description: "Investigation file with findings",
				},
			},
		},
	}

	result := VerifySkillOutputs(manifest, projectDir, time.Time{})

	if !result.Passed {
		t.Errorf("VerifySkillOutputs() Passed = false, want true")
		for _, e := range result.Errors {
			t.Logf("Error: %s", e)
		}
	}

	if len(result.Results) != 1 {
		t.Fatalf("len(result.Results) = %d, want 1", len(result.Results))
	}

	if !result.Results[0].Matched {
		t.Errorf("result.Results[0].Matched = false, want true")
	}

	if len(result.Results[0].MatchedFiles) != 1 {
		t.Errorf("len(result.Results[0].MatchedFiles) = %d, want 1", len(result.Results[0].MatchedFiles))
	}
}

func TestVerifySkillOutputs_NoMatch(t *testing.T) {
	// Create temp project directory with no investigation files
	projectDir := t.TempDir()

	manifest := &SkillManifest{
		Name: "investigation",
		Outputs: SkillOutputs{
			Required: []SkillOutput{
				{
					Pattern:     ".kb/investigations/{date}-inv-*.md",
					Description: "Investigation file with findings",
				},
			},
		},
	}

	result := VerifySkillOutputs(manifest, projectDir, time.Time{})

	if result.Passed {
		t.Errorf("VerifySkillOutputs() Passed = true, want false")
	}

	if len(result.Errors) != 1 {
		t.Fatalf("len(result.Errors) = %d, want 1", len(result.Errors))
	}

	if len(result.Results) != 1 {
		t.Fatalf("len(result.Results) = %d, want 1", len(result.Results))
	}

	if result.Results[0].Matched {
		t.Errorf("result.Results[0].Matched = true, want false")
	}
}

func TestVerifySkillOutputs_NoRequiredOutputs(t *testing.T) {
	projectDir := t.TempDir()

	manifest := &SkillManifest{
		Name: "feature-impl",
		Outputs: SkillOutputs{
			Required: []SkillOutput{}, // Empty - no required outputs
		},
	}

	result := VerifySkillOutputs(manifest, projectDir, time.Time{})

	if !result.Passed {
		t.Errorf("VerifySkillOutputs() Passed = false, want true for no required outputs")
	}

	if len(result.Errors) != 0 {
		t.Errorf("len(result.Errors) = %d, want 0", len(result.Errors))
	}
}

func TestVerifySkillOutputs_SpawnTimeFiltering(t *testing.T) {
	// Create temp project directory
	projectDir := t.TempDir()

	// Create the investigation file
	kbDir := filepath.Join(projectDir, ".kb", "investigations")
	if err := os.MkdirAll(kbDir, 0755); err != nil {
		t.Fatalf("failed to create .kb/investigations: %v", err)
	}
	invFile := filepath.Join(kbDir, "2025-12-25-inv-old-file.md")
	if err := os.WriteFile(invFile, []byte("# Old Investigation"), 0644); err != nil {
		t.Fatalf("failed to write investigation file: %v", err)
	}

	manifest := &SkillManifest{
		Name: "investigation",
		Outputs: SkillOutputs{
			Required: []SkillOutput{
				{
					Pattern:     ".kb/investigations/{date}-inv-*.md",
					Description: "Investigation file with findings",
				},
			},
		},
	}

	// Set spawn time to the future - the file was created before spawn
	futureTime := time.Now().Add(1 * time.Hour)

	result := VerifySkillOutputs(manifest, projectDir, futureTime)

	if result.Passed {
		t.Errorf("VerifySkillOutputs() Passed = true, want false (file too old)")
	}

	// Now set spawn time to the past - the file was created after spawn
	pastTime := time.Now().Add(-1 * time.Hour)

	result = VerifySkillOutputs(manifest, projectDir, pastTime)

	if !result.Passed {
		t.Errorf("VerifySkillOutputs() Passed = false, want true (file after spawn time)")
	}
}

func TestVerifySkillOutputsForCompletion_NoSpawnContext(t *testing.T) {
	workspacePath := t.TempDir()
	projectDir := t.TempDir()

	result, err := VerifySkillOutputsForCompletion(workspacePath, projectDir)
	if err != nil {
		t.Fatalf("VerifySkillOutputsForCompletion() error = %v", err)
	}

	// Should return nil (no spawn context, no verification)
	if result != nil {
		t.Errorf("VerifySkillOutputsForCompletion() = %v, want nil", result)
	}
}

func TestVerifySkillOutputsForCompletion_SkillNotFound(t *testing.T) {
	// Create workspace with spawn context referencing a skill that doesn't exist
	workspacePath := t.TempDir()
	projectDir := t.TempDir()

	content := `## SKILL GUIDANCE (nonexistent-skill)

Follow the guidance.`
	if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write SPAWN_CONTEXT.md: %v", err)
	}

	result, err := VerifySkillOutputsForCompletion(workspacePath, projectDir)
	if err != nil {
		t.Fatalf("VerifySkillOutputsForCompletion() error = %v", err)
	}

	// Should return nil (skill manifest not found, graceful skip)
	if result != nil {
		t.Errorf("VerifySkillOutputsForCompletion() = %v, want nil (skill not found)", result)
	}
}
