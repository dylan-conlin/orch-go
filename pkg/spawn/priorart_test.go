package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

func TestGatherPriorArt_NoPriorWork(t *testing.T) {
	// When no beads ID is provided, should return empty
	result := GatherPriorArt("", "/tmp/fake", nil)
	if result != "" {
		t.Errorf("expected empty result for empty beads ID, got: %s", result)
	}
}

func TestGatherPriorArt_NoArchivedWorkspaces(t *testing.T) {
	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".orch", "workspace", "archived"), 0755)

	result := GatherPriorArt("test-123", tempDir, nil)
	if result != "" {
		t.Errorf("expected empty result when no archived workspaces exist, got: %s", result)
	}
}

func TestGatherPriorArt_FindsByBeadsID(t *testing.T) {
	tempDir := t.TempDir()
	archivedDir := filepath.Join(tempDir, ".orch", "workspace", "archived")

	// Create an archived workspace with matching beads ID and SYNTHESIS.md
	wsDir := filepath.Join(archivedDir, "og-feat-prior-work-28feb-a1b2")
	os.MkdirAll(wsDir, 0755)

	manifest := `{"workspace_name":"og-feat-prior-work-28feb-a1b2","skill":"feature-impl","beads_id":"test-123","project_dir":"` + tempDir + `","spawn_time":"2026-02-28T10:00:00Z"}`
	os.WriteFile(filepath.Join(wsDir, "AGENT_MANIFEST.json"), []byte(manifest), 0644)

	synthesis := `# SYNTHESIS

## TLDR

Implemented the widget factory pattern for reusable component creation.

## Delta

- Added pkg/widget/factory.go
- Added pkg/widget/factory_test.go
`
	os.WriteFile(filepath.Join(wsDir, "SYNTHESIS.md"), []byte(synthesis), 0644)

	result := GatherPriorArt("test-123", tempDir, nil)

	if !strings.Contains(result, "PRIOR COMPLETIONS") {
		t.Error("expected result to contain PRIOR COMPLETIONS header")
	}
	if !strings.Contains(result, "test-123") {
		t.Error("expected result to contain beads ID")
	}
	if !strings.Contains(result, "widget factory pattern") {
		t.Error("expected result to contain TLDR content from SYNTHESIS.md")
	}
}

func TestGatherPriorArt_FindsByBeadsID_UsesCloseReason(t *testing.T) {
	tempDir := t.TempDir()
	archivedDir := filepath.Join(tempDir, ".orch", "workspace", "archived")

	// Create archived workspace without SYNTHESIS.md (light tier)
	wsDir := filepath.Join(archivedDir, "og-feat-light-task-28feb-c3d4")
	os.MkdirAll(wsDir, 0755)

	manifest := `{"workspace_name":"og-feat-light-task-28feb-c3d4","skill":"feature-impl","beads_id":"test-456","project_dir":"` + tempDir + `","spawn_time":"2026-02-28T10:00:00Z"}`
	os.WriteFile(filepath.Join(wsDir, "AGENT_MANIFEST.json"), []byte(manifest), 0644)

	// Use beads client to provide close_reason as fallback
	mockClient := beads.NewMockClient()
	mockClient.Issues["test-456"] = &beads.Issue{
		ID:          "test-456",
		Title:       "Implement light task",
		Status:      "closed",
		CloseReason: "Added error handling for edge cases",
	}

	result := GatherPriorArt("test-456", tempDir, mockClient)

	if !strings.Contains(result, "PRIOR COMPLETIONS") {
		t.Error("expected result to contain PRIOR COMPLETIONS header")
	}
	if !strings.Contains(result, "error handling") {
		t.Error("expected result to contain close_reason fallback content")
	}
}

func TestGatherPriorArt_MultiplePriorAttempts(t *testing.T) {
	tempDir := t.TempDir()
	archivedDir := filepath.Join(tempDir, ".orch", "workspace", "archived")

	// Create two archived workspaces for the same beads ID
	ws1 := filepath.Join(archivedDir, "og-feat-first-try-27feb-a1b2")
	os.MkdirAll(ws1, 0755)
	manifest1 := `{"workspace_name":"og-feat-first-try-27feb-a1b2","skill":"feature-impl","beads_id":"test-789","project_dir":"` + tempDir + `","spawn_time":"2026-02-27T10:00:00Z"}`
	os.WriteFile(filepath.Join(ws1, "AGENT_MANIFEST.json"), []byte(manifest1), 0644)
	os.WriteFile(filepath.Join(ws1, "SYNTHESIS.md"), []byte("# SYNTHESIS\n\n## TLDR\n\nFirst attempt at auth middleware.\n"), 0644)

	ws2 := filepath.Join(archivedDir, "og-feat-second-try-28feb-c3d4")
	os.MkdirAll(ws2, 0755)
	manifest2 := `{"workspace_name":"og-feat-second-try-28feb-c3d4","skill":"feature-impl","beads_id":"test-789","project_dir":"` + tempDir + `","spawn_time":"2026-02-28T10:00:00Z"}`
	os.WriteFile(filepath.Join(ws2, "AGENT_MANIFEST.json"), []byte(manifest2), 0644)
	os.WriteFile(filepath.Join(ws2, "SYNTHESIS.md"), []byte("# SYNTHESIS\n\n## TLDR\n\nSecond attempt fixed token refresh race.\n"), 0644)

	result := GatherPriorArt("test-789", tempDir, nil)

	if !strings.Contains(result, "First attempt") {
		t.Error("expected result to contain first attempt summary")
	}
	if !strings.Contains(result, "Second attempt") {
		t.Error("expected result to contain second attempt summary")
	}
}

func TestGatherPriorArt_ExcludesActiveWorkspaces(t *testing.T) {
	tempDir := t.TempDir()

	// Create active workspace (not archived) with same beads ID - should be excluded
	activeDir := filepath.Join(tempDir, ".orch", "workspace", "og-feat-active-28feb-e5f6")
	os.MkdirAll(activeDir, 0755)
	manifest := `{"workspace_name":"og-feat-active-28feb-e5f6","skill":"feature-impl","beads_id":"test-active","project_dir":"` + tempDir + `","spawn_time":"2026-02-28T10:00:00Z"}`
	os.WriteFile(filepath.Join(activeDir, "AGENT_MANIFEST.json"), []byte(manifest), 0644)
	os.WriteFile(filepath.Join(activeDir, "SYNTHESIS.md"), []byte("# SYNTHESIS\n\n## TLDR\n\nActive work.\n"), 0644)

	// Also create archived dir (empty)
	os.MkdirAll(filepath.Join(tempDir, ".orch", "workspace", "archived"), 0755)

	result := GatherPriorArt("test-active", tempDir, nil)
	if result != "" {
		t.Errorf("expected empty result when only active workspace exists, got: %s", result)
	}
}

func TestFormatPriorCompletions(t *testing.T) {
	completions := []PriorCompletion{
		{
			BeadsID:   "test-123",
			Skill:     "feature-impl",
			Summary:   "Implemented the widget factory pattern for reusable component creation.",
			Workspace: "og-feat-widget-28feb-a1b2",
		},
		{
			BeadsID:   "test-456",
			Skill:     "investigation",
			Summary:   "Found that factory pattern works best for this use case.",
			Workspace: "og-inv-factory-27feb-c3d4",
		},
	}

	result := FormatPriorCompletions(completions)

	if !strings.Contains(result, "## PRIOR COMPLETIONS") {
		t.Error("expected PRIOR COMPLETIONS header")
	}
	if !strings.Contains(result, "test-123") {
		t.Error("expected first completion beads ID")
	}
	if !strings.Contains(result, "test-456") {
		t.Error("expected second completion beads ID")
	}
	if !strings.Contains(result, "widget factory") {
		t.Error("expected first completion summary")
	}
	if !strings.Contains(result, "factory pattern works best") {
		t.Error("expected second completion summary")
	}
	// Should include instruction to avoid re-doing completed work
	if !strings.Contains(result, "Do NOT re-do") {
		t.Error("expected instruction to avoid re-doing work")
	}
}

func TestFormatPriorCompletions_Empty(t *testing.T) {
	result := FormatPriorCompletions(nil)
	if result != "" {
		t.Errorf("expected empty result for nil completions, got: %s", result)
	}
}

func TestExtractTLDRFromSynthesis(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "standard TLDR section",
			content: `# SYNTHESIS

## TLDR

Implemented auth middleware with JWT validation.

## Delta

- Added middleware
`,
			want: "Implemented auth middleware with JWT validation.",
		},
		{
			name: "no TLDR section",
			content: `# SYNTHESIS

## Delta

- Added middleware
`,
			want: "",
		},
		{
			name: "multiline TLDR",
			content: `# SYNTHESIS

## TLDR

First line of summary.
Second line of summary.

## Delta
`,
			want: "First line of summary.\nSecond line of summary.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTLDRFromSynthesis(tt.content)
			if got != tt.want {
				t.Errorf("ExtractTLDRFromSynthesis() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestContextTemplate_WithPriorCompletions(t *testing.T) {
	cfg := &Config{
		Task:          "Implement feature X",
		BeadsID:       "test-001",
		ProjectDir:    "/tmp/test",
		WorkspaceName: "test-ws",
		SkillName:     "feature-impl",
		Tier:          TierLight,
		PriorCompletions: `## PRIOR COMPLETIONS

Prior agents completed related work on this issue. Review before starting:

- **test-prior** (feature-impl): Implemented initial version of feature X.
`,
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	if !strings.Contains(content, "PRIOR COMPLETIONS") {
		t.Error("expected generated context to contain PRIOR COMPLETIONS section")
	}
	if !strings.Contains(content, "test-prior") {
		t.Error("expected generated context to contain prior completion beads ID")
	}
}

func TestContextTemplate_WithoutPriorCompletions(t *testing.T) {
	cfg := &Config{
		Task:          "Implement feature Y",
		BeadsID:       "test-002",
		ProjectDir:    "/tmp/test",
		WorkspaceName: "test-ws",
		SkillName:     "feature-impl",
		Tier:          TierLight,
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	if strings.Contains(content, "PRIOR COMPLETIONS") {
		t.Error("expected generated context to NOT contain PRIOR COMPLETIONS when field is empty")
	}
}

func TestFindArchivedWorkspacesByBeadsID(t *testing.T) {
	tempDir := t.TempDir()
	archivedDir := filepath.Join(tempDir, ".orch", "workspace", "archived")

	// Create 3 archived workspaces, 2 matching the beads ID
	ws1 := filepath.Join(archivedDir, "og-feat-match1-27feb-a1b2")
	os.MkdirAll(ws1, 0755)
	os.WriteFile(filepath.Join(ws1, "AGENT_MANIFEST.json"), []byte(`{"beads_id":"target-id","skill":"feature-impl","spawn_time":"2026-02-27T10:00:00Z"}`), 0644)

	ws2 := filepath.Join(archivedDir, "og-feat-match2-28feb-c3d4")
	os.MkdirAll(ws2, 0755)
	os.WriteFile(filepath.Join(ws2, "AGENT_MANIFEST.json"), []byte(`{"beads_id":"target-id","skill":"investigation","spawn_time":"2026-02-28T10:00:00Z"}`), 0644)

	ws3 := filepath.Join(archivedDir, "og-feat-other-28feb-e5f6")
	os.MkdirAll(ws3, 0755)
	os.WriteFile(filepath.Join(ws3, "AGENT_MANIFEST.json"), []byte(`{"beads_id":"other-id","skill":"feature-impl","spawn_time":"2026-02-28T12:00:00Z"}`), 0644)

	matches := FindArchivedWorkspacesByBeadsID(tempDir, "target-id")

	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}
}

func TestFindArchivedWorkspacesByBeadsID_NoArchiveDir(t *testing.T) {
	tempDir := t.TempDir()
	// No .orch/workspace/archived directory

	matches := FindArchivedWorkspacesByBeadsID(tempDir, "test-id")
	if len(matches) != 0 {
		t.Errorf("expected 0 matches when archive dir doesn't exist, got %d", len(matches))
	}
}
