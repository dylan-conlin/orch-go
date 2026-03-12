package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFetchArchitectDesign_Success(t *testing.T) {
	tempDir := t.TempDir()
	archivedDir := filepath.Join(tempDir, ".orch", "workspace", "archived")

	// Create an archived architect workspace with SYNTHESIS.md
	wsDir := filepath.Join(archivedDir, "og-arch-design-hotspot-11mar-4b1c")
	os.MkdirAll(wsDir, 0755)

	manifest := `{"workspace_name":"og-arch-design-hotspot-11mar-4b1c","skill":"architect","beads_id":"orch-go-abc1","project_dir":"` + tempDir + `","spawn_time":"2026-03-11T10:00:00Z"}`
	os.WriteFile(filepath.Join(wsDir, "AGENT_MANIFEST.json"), []byte(manifest), 0644)

	synthesisContent := `# SYNTHESIS

## TLDR

Designed extraction plan for spawn_cmd.go: split into spawn_cmd.go (flags+cobra), spawn_pipeline.go (orchestration), and spawn_helpers.go (utilities).

## Delta

- spawn_cmd.go reduced from 1800 to ~600 lines
- Three clear files with single responsibilities
- No behavioral changes, pure extraction

## Key Decisions

1. Keep cobra command definition inline (not worth extracting)
2. Pipeline functions move first (highest coupling)
3. Helper functions move last (most independent)
`
	os.WriteFile(filepath.Join(wsDir, "SYNTHESIS.md"), []byte(synthesisContent), 0644)

	result := FetchArchitectDesign("orch-go-abc1", tempDir)

	if result == "" {
		t.Fatal("expected non-empty architect design content")
	}
	if !strings.Contains(result, "Designed extraction plan") {
		t.Error("expected SYNTHESIS.md content in result")
	}
	if !strings.Contains(result, "Key Decisions") {
		t.Error("expected Key Decisions section in result")
	}
}

func TestFetchArchitectDesign_NoArchivedWorkspace(t *testing.T) {
	tempDir := t.TempDir()
	// No archived workspace directory

	result := FetchArchitectDesign("orch-go-nonexist", tempDir)

	if result != "" {
		t.Errorf("expected empty result when no archived workspace exists, got: %s", result)
	}
}

func TestFetchArchitectDesign_NoSynthesis(t *testing.T) {
	tempDir := t.TempDir()
	archivedDir := filepath.Join(tempDir, ".orch", "workspace", "archived")

	// Create workspace without SYNTHESIS.md
	wsDir := filepath.Join(archivedDir, "og-arch-design-something-11mar-5b6f")
	os.MkdirAll(wsDir, 0755)

	manifest := `{"workspace_name":"og-arch-design-something-11mar-5b6f","skill":"architect","beads_id":"orch-go-xyz2","project_dir":"` + tempDir + `","spawn_time":"2026-03-11T10:00:00Z"}`
	os.WriteFile(filepath.Join(wsDir, "AGENT_MANIFEST.json"), []byte(manifest), 0644)

	result := FetchArchitectDesign("orch-go-xyz2", tempDir)

	if result != "" {
		t.Errorf("expected empty result when no SYNTHESIS.md exists, got: %s", result)
	}
}

func TestFetchArchitectDesign_EmptyBeadsID(t *testing.T) {
	result := FetchArchitectDesign("", "/some/path")

	if result != "" {
		t.Errorf("expected empty result for empty beads ID, got: %s", result)
	}
}

func TestFetchArchitectDesign_ActiveWorkspace(t *testing.T) {
	tempDir := t.TempDir()

	// Create an ACTIVE workspace (not archived) — should also be found
	wsDir := filepath.Join(tempDir, ".orch", "workspace", "og-arch-design-active-11mar-9a9b")
	os.MkdirAll(wsDir, 0755)

	manifest := `{"workspace_name":"og-arch-design-active-11mar-9a9b","skill":"architect","beads_id":"orch-go-act1","project_dir":"` + tempDir + `","spawn_time":"2026-03-11T12:00:00Z"}`
	os.WriteFile(filepath.Join(wsDir, "AGENT_MANIFEST.json"), []byte(manifest), 0644)

	synthesisContent := `# SYNTHESIS

## TLDR

Active workspace design.

## Approach

Use the active workspace approach.
`
	os.WriteFile(filepath.Join(wsDir, "SYNTHESIS.md"), []byte(synthesisContent), 0644)

	result := FetchArchitectDesign("orch-go-act1", tempDir)

	if result == "" {
		t.Fatal("expected non-empty result from active workspace")
	}
	if !strings.Contains(result, "Active workspace design") {
		t.Error("expected content from active workspace SYNTHESIS.md")
	}
}

func TestArchitectDesignInSpawnContext(t *testing.T) {
	cfg := &Config{
		Task:          "implement feature X",
		BeadsID:       "test-123",
		ProjectDir:    "/tmp/test-project",
		WorkspaceName: "og-feat-test-12mar-ab12",
		SkillName:     "feature-impl",
		Tier:          "light",
		ArchitectDesign: "## Design from architect\n\nExtract foo.go into bar.go and baz.go.",
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	if !strings.Contains(content, "## Architect Design") {
		t.Error("expected '## Architect Design' section in spawn context")
	}
	if !strings.Contains(content, "Extract foo.go into bar.go and baz.go") {
		t.Error("expected architect design content in spawn context")
	}
}

func TestArchitectDesignNotInSpawnContext_WhenEmpty(t *testing.T) {
	cfg := &Config{
		Task:          "implement feature X",
		BeadsID:       "test-123",
		ProjectDir:    "/tmp/test-project",
		WorkspaceName: "og-feat-test-12mar-ab12",
		SkillName:     "feature-impl",
		Tier:          "light",
		// ArchitectDesign is empty
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	if strings.Contains(content, "## Architect Design") {
		t.Error("expected no '## Architect Design' section when field is empty")
	}
}
