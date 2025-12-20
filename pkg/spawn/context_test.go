package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateWorkspaceName(t *testing.T) {
	tests := []struct {
		name      string
		skillName string
		task      string
		wantParts []string // parts that should appear in the workspace name
	}{
		{
			name:      "investigation skill",
			skillName: "investigation",
			task:      "explore the codebase",
			wantParts: []string{"og-inv-", "explore", "codebase"},
		},
		{
			name:      "feature-impl skill",
			skillName: "feature-impl",
			task:      "add new spawn command",
			wantParts: []string{"og-feat-", "add", "new", "spawn"}, // only 3 words in slug
		},
		{
			name:      "unknown skill",
			skillName: "custom-skill",
			task:      "do something",
			wantParts: []string{"og-work-", "do", "something"},
		},
		{
			name:      "task with stop words",
			skillName: "investigation",
			task:      "the quick brown fox in the forest",
			wantParts: []string{"og-inv-", "quick", "brown", "fox"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateWorkspaceName(tt.skillName, tt.task)
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("expected workspace name to contain %q, got %q", part, got)
				}
			}
		})
	}
}

func TestConfigPaths(t *testing.T) {
	cfg := &Config{
		ProjectDir:    "/home/user/project",
		WorkspaceName: "og-inv-test-19dec",
	}

	wantWorkspace := "/home/user/project/.orch/workspace/og-inv-test-19dec"
	if got := cfg.WorkspacePath(); got != wantWorkspace {
		t.Errorf("WorkspacePath() = %q, want %q", got, wantWorkspace)
	}

	wantContext := wantWorkspace + "/SPAWN_CONTEXT.md"
	if got := cfg.ContextFilePath(); got != wantContext {
		t.Errorf("ContextFilePath() = %q, want %q", got, wantContext)
	}
}

func TestGenerateContext(t *testing.T) {
	cfg := &Config{
		Task:       "implement spawn command",
		SkillName:  "feature-impl",
		Project:    "orch-go",
		ProjectDir: "/Users/test/orch-go",
		BeadsID:    "orch-go-123",
		Phases:     "implementation,validation",
		Mode:       "tdd",
		Validation: "tests",
		SkillContent: `---
name: feature-impl
---

# Feature Implementation

Test skill content.
`,
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	// Check key sections are present
	checks := []string{
		"TASK: implement spawn command",
		"PROJECT_DIR: /Users/test/orch-go",
		"bd comment orch-go-123",
		"SKILL GUIDANCE (feature-impl)",
		"Phases: implementation,validation",
		"Mode: tdd",
		"Validation: tests",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("expected content to contain %q", check)
		}
	}
}

func TestGenerateContext_NoSkill(t *testing.T) {
	cfg := &Config{
		Task:       "simple task",
		Project:    "test-project",
		ProjectDir: "/tmp/test",
		BeadsID:    "test-123",
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	// Should NOT contain skill section when no skill content
	if strings.Contains(content, "SKILL GUIDANCE") {
		t.Error("expected no SKILL GUIDANCE section when no skill content")
	}
}

func TestWriteContext(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &Config{
		Task:          "test task",
		Project:       "test",
		ProjectDir:    tempDir,
		WorkspaceName: "og-test-19dec",
		BeadsID:       "test-123",
	}

	if err := WriteContext(cfg); err != nil {
		t.Fatalf("WriteContext failed: %v", err)
	}

	// Check file was created
	contextPath := filepath.Join(tempDir, ".orch", "workspace", "og-test-19dec", "SPAWN_CONTEXT.md")
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		t.Errorf("expected context file to exist at %s", contextPath)
	}

	// Check content
	content, err := os.ReadFile(contextPath)
	if err != nil {
		t.Fatalf("failed to read context file: %v", err)
	}

	if !strings.Contains(string(content), "TASK: test task") {
		t.Error("context file should contain task")
	}
}

func TestMinimalPrompt(t *testing.T) {
	cfg := &Config{
		ProjectDir:    "/Users/test/orch-go",
		WorkspaceName: "og-feat-spawn-19dec",
	}

	prompt := MinimalPrompt(cfg)

	if !strings.Contains(prompt, "/Users/test/orch-go/.orch/workspace/og-feat-spawn-19dec/SPAWN_CONTEXT.md") {
		t.Errorf("minimal prompt should contain full path to context file, got: %s", prompt)
	}
}
