package spawn

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/action"
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
			wantParts: []string{"og-feat-", "add", "spawn", "command"}, // 'new' filtered as vague adjective
		},
		{
			name:      "unknown skill",
			skillName: "custom-skill",
			task:      "do something",
			wantParts: []string{"og-work-", "something"}, // 'do' filtered as auxiliary verb
		},
		{
			name:      "task with stop words",
			skillName: "investigation",
			task:      "the quick brown fox in the forest",
			wantParts: []string{"og-inv-", "quick", "brown", "fox"},
		},
		{
			name:      "task with question words filtered",
			skillName: "architect",
			task:      "Design how to better surface ready queue items",
			wantParts: []string{"og-arch-", "design", "surface", "ready"}, // 'how', 'better' filtered
		},
		{
			name:      "task with modal verbs filtered",
			skillName: "investigation",
			task:      "what should we do when users can login",
			wantParts: []string{"og-inv-", "users", "login"}, // 'what', 'should', 'we', 'do', 'when', 'can' filtered
		},
		{
			name:      "task with filler words filtered",
			skillName: "feature-impl",
			task:      "need to make some changes to get better logging",
			wantParts: []string{"og-feat-", "changes", "logging"}, // 'need', 'make', 'some', 'get', 'better' filtered
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
		"CREATE SYNTHESIS.md",
		"SYNTHESIS.md is created",
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

func TestEnsureSynthesisTemplate(t *testing.T) {
	t.Run("creates template when missing", func(t *testing.T) {
		tempDir := t.TempDir()

		// Ensure template doesn't exist initially
		templatePath := filepath.Join(tempDir, ".orch", "templates", "SYNTHESIS.md")
		if _, err := os.Stat(templatePath); !os.IsNotExist(err) {
			t.Fatal("template should not exist initially")
		}

		// Call EnsureSynthesisTemplate
		if err := EnsureSynthesisTemplate(tempDir); err != nil {
			t.Fatalf("EnsureSynthesisTemplate failed: %v", err)
		}

		// Check template was created
		if _, err := os.Stat(templatePath); os.IsNotExist(err) {
			t.Error("template should exist after EnsureSynthesisTemplate")
		}

		// Check content
		content, err := os.ReadFile(templatePath)
		if err != nil {
			t.Fatalf("failed to read template: %v", err)
		}

		if !strings.Contains(string(content), "# Session Synthesis") {
			t.Error("template should contain synthesis header")
		}
		if !strings.Contains(string(content), "## TLDR") {
			t.Error("template should contain TLDR section")
		}
		if !strings.Contains(string(content), "## Delta") {
			t.Error("template should contain Delta section")
		}
	})

	t.Run("does not overwrite existing template", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create templates directory and custom template
		templatesDir := filepath.Join(tempDir, ".orch", "templates")
		if err := os.MkdirAll(templatesDir, 0755); err != nil {
			t.Fatalf("failed to create templates dir: %v", err)
		}

		customContent := "# Custom Synthesis Template\n\nThis is a custom template."
		templatePath := filepath.Join(templatesDir, "SYNTHESIS.md")
		if err := os.WriteFile(templatePath, []byte(customContent), 0644); err != nil {
			t.Fatalf("failed to write custom template: %v", err)
		}

		// Call EnsureSynthesisTemplate
		if err := EnsureSynthesisTemplate(tempDir); err != nil {
			t.Fatalf("EnsureSynthesisTemplate failed: %v", err)
		}

		// Check content was NOT overwritten
		content, err := os.ReadFile(templatePath)
		if err != nil {
			t.Fatalf("failed to read template: %v", err)
		}

		if string(content) != customContent {
			t.Error("existing template should not be overwritten")
		}
	})
}

func TestWriteContext_CreatesSynthesisTemplate(t *testing.T) {
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

	// Check SYNTHESIS.md template was created
	templatePath := filepath.Join(tempDir, ".orch", "templates", "SYNTHESIS.md")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		t.Error("WriteContext should create SYNTHESIS.md template")
	}
}

func TestEnsureFailureReportTemplate(t *testing.T) {
	t.Run("creates template when missing", func(t *testing.T) {
		tempDir := t.TempDir()

		// Ensure template doesn't exist initially
		templatePath := filepath.Join(tempDir, ".orch", "templates", "FAILURE_REPORT.md")
		if _, err := os.Stat(templatePath); !os.IsNotExist(err) {
			t.Fatal("template should not exist initially")
		}

		// Call EnsureFailureReportTemplate
		if err := EnsureFailureReportTemplate(tempDir); err != nil {
			t.Fatalf("EnsureFailureReportTemplate failed: %v", err)
		}

		// Check template was created
		if _, err := os.Stat(templatePath); os.IsNotExist(err) {
			t.Error("template should exist after EnsureFailureReportTemplate")
		}

		// Check content
		content, err := os.ReadFile(templatePath)
		if err != nil {
			t.Fatalf("failed to read template: %v", err)
		}

		if !strings.Contains(string(content), "# Failure Report") {
			t.Error("template should contain failure report header")
		}
		if !strings.Contains(string(content), "## Failure Summary") {
			t.Error("template should contain Failure Summary section")
		}
		if !strings.Contains(string(content), "## Recovery Recommendations") {
			t.Error("template should contain Recovery Recommendations section")
		}
	})

	t.Run("does not overwrite existing template", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create templates directory and custom template
		templatesDir := filepath.Join(tempDir, ".orch", "templates")
		if err := os.MkdirAll(templatesDir, 0755); err != nil {
			t.Fatalf("failed to create templates dir: %v", err)
		}

		customContent := "# Custom Failure Report Template\n\nThis is a custom template."
		templatePath := filepath.Join(templatesDir, "FAILURE_REPORT.md")
		if err := os.WriteFile(templatePath, []byte(customContent), 0644); err != nil {
			t.Fatalf("failed to write custom template: %v", err)
		}

		// Call EnsureFailureReportTemplate
		if err := EnsureFailureReportTemplate(tempDir); err != nil {
			t.Fatalf("EnsureFailureReportTemplate failed: %v", err)
		}

		// Check content was NOT overwritten
		content, err := os.ReadFile(templatePath)
		if err != nil {
			t.Fatalf("failed to read template: %v", err)
		}

		if string(content) != customContent {
			t.Error("existing template should not be overwritten")
		}
	})
}

func TestWriteFailureReport(t *testing.T) {
	tempDir := t.TempDir()
	workspacePath := filepath.Join(tempDir, "og-test-workspace")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	reportPath, err := WriteFailureReport(
		workspacePath,
		"og-test-workspace",
		"test-123",
		"Out of context",
		"implement test feature",
	)
	if err != nil {
		t.Fatalf("WriteFailureReport failed: %v", err)
	}

	// Check file was created
	expectedPath := filepath.Join(workspacePath, "FAILURE_REPORT.md")
	if reportPath != expectedPath {
		t.Errorf("expected report path %q, got %q", expectedPath, reportPath)
	}

	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Error("failure report should exist after WriteFailureReport")
	}

	// Check content
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("failed to read failure report: %v", err)
	}

	checks := []string{
		"# Failure Report",
		"**Agent:** og-test-workspace",
		"**Issue:** test-123",
		"**Reason:** Out of context",
		"**Task:** implement test feature",
		"## Failure Summary",
		"## Recovery Recommendations",
		"orch spawn {skill}",
		"--issue test-123",
	}

	for _, check := range checks {
		if !strings.Contains(string(content), check) {
			t.Errorf("failure report should contain %q", check)
		}
	}
}

func TestGenerateFailureReport(t *testing.T) {
	report := generateFailureReport(
		"og-debug-test-21dec",
		"orch-go-abc",
		"Stuck in loop",
		"debug the authentication issue",
	)

	checks := []string{
		"**Agent:** og-debug-test-21dec",
		"**Issue:** orch-go-abc",
		"**Reason:** Stuck in loop",
		"**Task:** debug the authentication issue",
		"**Primary Cause:** Stuck in loop",
		"--issue orch-go-abc",
		"bd show orch-go-abc",
	}

	for _, check := range checks {
		if !strings.Contains(report, check) {
			t.Errorf("generated report should contain %q", check)
		}
	}
}

func TestDefaultTierForSkill(t *testing.T) {
	tests := []struct {
		skill string
		want  string
	}{
		// Full tier skills (produce knowledge artifacts)
		{"investigation", TierFull},
		{"architect", TierFull},
		{"research", TierFull},
		{"codebase-audit", TierFull},
		{"design-session", TierFull},
		{"systematic-debugging", TierFull},

		// Light tier skills (implementation-focused)
		{"feature-impl", TierLight},
		{"reliability-testing", TierLight},
		{"issue-creation", TierLight},

		// Unknown skill defaults to full (conservative)
		{"unknown-skill", TierFull},
		{"", TierFull},
	}

	for _, tt := range tests {
		t.Run(tt.skill, func(t *testing.T) {
			got := DefaultTierForSkill(tt.skill)
			if got != tt.want {
				t.Errorf("DefaultTierForSkill(%q) = %q, want %q", tt.skill, got, tt.want)
			}
		})
	}
}

func TestGenerateContext_LightTier(t *testing.T) {
	cfg := &Config{
		Task:       "implement quick fix",
		SkillName:  "feature-impl",
		Project:    "test-project",
		ProjectDir: "/tmp/test",
		BeadsID:    "test-123",
		Tier:       TierLight,
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	// Should contain light tier indicator
	if !strings.Contains(content, "SPAWN TIER: light") {
		t.Error("expected content to contain 'SPAWN TIER: light'")
	}
	if !strings.Contains(content, "LIGHT TIER") {
		t.Error("expected content to contain light tier message")
	}
	if !strings.Contains(content, "SYNTHESIS.md is NOT required") {
		t.Error("expected content to indicate SYNTHESIS.md is not required")
	}

	// Should NOT contain full tier messaging
	if strings.Contains(content, "FULL TIER") {
		t.Error("light tier context should not contain FULL TIER messaging")
	}
}

func TestGenerateContext_FullTier(t *testing.T) {
	cfg := &Config{
		Task:       "deep investigation",
		SkillName:  "investigation",
		Project:    "test-project",
		ProjectDir: "/tmp/test",
		BeadsID:    "test-123",
		Tier:       TierFull,
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	// Should contain full tier indicator
	if !strings.Contains(content, "SPAWN TIER: full") {
		t.Error("expected content to contain 'SPAWN TIER: full'")
	}
	if !strings.Contains(content, "FULL TIER") {
		t.Error("expected content to contain full tier message")
	}
	if !strings.Contains(content, "requires SYNTHESIS.md") {
		t.Error("expected content to indicate SYNTHESIS.md is required")
	}
}

func TestWriteContext_LightTierSkipsSynthesisTemplate(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &Config{
		Task:          "quick fix",
		Project:       "test",
		ProjectDir:    tempDir,
		WorkspaceName: "og-feat-19dec",
		BeadsID:       "test-123",
		Tier:          TierLight,
	}

	if err := WriteContext(cfg); err != nil {
		t.Fatalf("WriteContext failed: %v", err)
	}

	// Check SYNTHESIS.md template was NOT created (light tier)
	templatePath := filepath.Join(tempDir, ".orch", "templates", "SYNTHESIS.md")
	if _, err := os.Stat(templatePath); !os.IsNotExist(err) {
		t.Error("light tier WriteContext should NOT create SYNTHESIS.md template")
	}

	// Check tier file was created
	tierPath := filepath.Join(tempDir, ".orch", "workspace", "og-feat-19dec", ".tier")
	content, err := os.ReadFile(tierPath)
	if err != nil {
		t.Fatalf("failed to read tier file: %v", err)
	}
	if strings.TrimSpace(string(content)) != TierLight {
		t.Errorf("tier file should contain 'light', got %q", string(content))
	}
}

func TestWriteContext_FullTierCreatesSynthesisTemplate(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &Config{
		Task:          "investigation",
		Project:       "test",
		ProjectDir:    tempDir,
		WorkspaceName: "og-inv-19dec",
		BeadsID:       "test-123",
		Tier:          TierFull,
	}

	if err := WriteContext(cfg); err != nil {
		t.Fatalf("WriteContext failed: %v", err)
	}

	// Check SYNTHESIS.md template WAS created (full tier)
	templatePath := filepath.Join(tempDir, ".orch", "templates", "SYNTHESIS.md")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		t.Error("full tier WriteContext should create SYNTHESIS.md template")
	}

	// Check tier file was created
	tierPath := filepath.Join(tempDir, ".orch", "workspace", "og-inv-19dec", ".tier")
	content, err := os.ReadFile(tierPath)
	if err != nil {
		t.Fatalf("failed to read tier file: %v", err)
	}
	if strings.TrimSpace(string(content)) != TierFull {
		t.Errorf("tier file should contain 'full', got %q", string(content))
	}
}

func TestDefaultIncludeServersForSkill(t *testing.T) {
	tests := []struct {
		skill string
		want  bool
	}{
		// UI-focused skills include servers
		{"feature-impl", true},
		{"systematic-debugging", true},
		{"reliability-testing", true},

		// Investigation-type skills don't include servers by default
		{"investigation", false},
		{"architect", false},
		{"research", false},
		{"codebase-audit", false},

		// Unknown skill defaults to false
		{"unknown-skill", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.skill, func(t *testing.T) {
			got := DefaultIncludeServersForSkill(tt.skill)
			if got != tt.want {
				t.Errorf("DefaultIncludeServersForSkill(%q) = %v, want %v", tt.skill, got, tt.want)
			}
		})
	}
}

func TestGenerateServerContext(t *testing.T) {
	t.Run("with servers configured", func(t *testing.T) {
		tempDir := t.TempDir()
		orchDir := filepath.Join(tempDir, ".orch")
		if err := os.MkdirAll(orchDir, 0755); err != nil {
			t.Fatalf("failed to create .orch dir: %v", err)
		}

		// Write config with servers
		configContent := `servers:
  web: 5173
  api: 3000
`
		configPath := filepath.Join(orchDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		context := GenerateServerContext(tempDir)

		// Check it contains expected content
		if !strings.Contains(context, "## LOCAL SERVERS") {
			t.Error("expected server context to contain header")
		}
		if !strings.Contains(context, "http://localhost:5173") {
			t.Error("expected server context to contain web port")
		}
		if !strings.Contains(context, "http://localhost:3000") {
			t.Error("expected server context to contain api port")
		}
		if !strings.Contains(context, "orch servers start") {
			t.Error("expected server context to contain quick commands")
		}
	})

	t.Run("without config file", func(t *testing.T) {
		tempDir := t.TempDir()

		context := GenerateServerContext(tempDir)

		// Should return empty string when no config
		if context != "" {
			t.Errorf("expected empty string when no config, got: %s", context)
		}
	})

	t.Run("with empty servers", func(t *testing.T) {
		tempDir := t.TempDir()
		orchDir := filepath.Join(tempDir, ".orch")
		if err := os.MkdirAll(orchDir, 0755); err != nil {
			t.Fatalf("failed to create .orch dir: %v", err)
		}

		// Write config with empty servers
		configContent := `servers: {}`
		configPath := filepath.Join(orchDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		context := GenerateServerContext(tempDir)

		// Should return empty string when no servers
		if context != "" {
			t.Errorf("expected empty string when no servers, got: %s", context)
		}
	})
}

func TestGenerateContext_WithServerContext(t *testing.T) {
	tempDir := t.TempDir()
	orchDir := filepath.Join(tempDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("failed to create .orch dir: %v", err)
	}

	// Write config with servers
	configContent := `servers:
  web: 5173
  api: 3000
`
	configPath := filepath.Join(orchDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg := &Config{
		Task:           "implement feature",
		SkillName:      "feature-impl",
		Project:        "test-project",
		ProjectDir:     tempDir,
		BeadsID:        "test-123",
		IncludeServers: true,
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	// Should contain server context
	if !strings.Contains(content, "## LOCAL SERVERS") {
		t.Error("expected content to contain server context")
	}
	if !strings.Contains(content, "http://localhost:5173") {
		t.Error("expected content to contain web port")
	}
}

func TestGenerateContext_WithoutServerContext(t *testing.T) {
	tempDir := t.TempDir()
	orchDir := filepath.Join(tempDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("failed to create .orch dir: %v", err)
	}

	// Write config with servers (but IncludeServers is false)
	configContent := `servers:
  web: 5173
`
	configPath := filepath.Join(orchDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg := &Config{
		Task:           "investigate something",
		SkillName:      "investigation",
		Project:        "test-project",
		ProjectDir:     tempDir,
		BeadsID:        "test-123",
		IncludeServers: false, // Explicitly disabled
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	// Should NOT contain server context
	if strings.Contains(content, "## LOCAL SERVERS") {
		t.Error("expected content to NOT contain server context when disabled")
	}
}

func TestStripBeadsInstructions(t *testing.T) {
	t.Run("removes beads code blocks", func(t *testing.T) {
		content := `## Some Section

Some text here.

### Report via Beads

` + "```" + `bash
bd comment <beads-id> "Phase: Planning - starting work"
bd comment <beads-id> "Phase: Complete - done"
` + "```" + `

### Next Section

More text here.
`
		result := StripBeadsInstructions(content)

		// Should remove the beads section
		if strings.Contains(result, "### Report via Beads") {
			t.Error("should remove Report via Beads section")
		}
		if strings.Contains(result, "bd comment <beads-id>") {
			t.Error("should remove bd comment commands")
		}

		// Should keep other sections
		if !strings.Contains(result, "## Some Section") {
			t.Error("should keep other sections")
		}
		if !strings.Contains(result, "### Next Section") {
			t.Error("should keep Next Section")
		}
	})

	t.Run("removes completion criteria with beads reporting", func(t *testing.T) {
		content := `## Completion Criteria

Before marking complete, verify ALL:

- [ ] **Root cause identified**
- [ ] **Fix implemented**
- [ ] **Tests passing**
- [ ] **Reported** - ` + "`bd comment <beads-id> \"Phase: Complete - [summary]\"`" + `

**If ANY unchecked, work is NOT complete.**
`
		result := StripBeadsInstructions(content)

		// Should remove the beads-specific criteria line
		if strings.Contains(result, "**Reported**") && strings.Contains(result, "bd comment") {
			t.Error("should remove Reported beads criteria line")
		}

		// Should keep other criteria
		if !strings.Contains(result, "**Root cause identified**") {
			t.Error("should keep other criteria")
		}
		if !strings.Contains(result, "**Tests passing**") {
			t.Error("should keep other criteria")
		}
	})

	t.Run("removes After All Criteria Met code blocks", func(t *testing.T) {
		content := `### After All Criteria Met

` + "```" + `bash
bd comment <beads-id> "Phase: Complete - Root cause: [X], Fix: [Y], Tests passing"
bd close <beads-id> --reason "Root cause: [X], Fix: [Y]"
` + "```" + `

Then call ` + "`/exit`" + ` to close agent session.
`
		result := StripBeadsInstructions(content)

		// Should remove the beads command block
		if strings.Contains(result, "bd close <beads-id>") {
			t.Error("should remove bd close command")
		}

		// Should keep the /exit instruction
		if !strings.Contains(result, "/exit") {
			t.Error("should keep /exit instruction")
		}
	})

	t.Run("handles empty content", func(t *testing.T) {
		result := StripBeadsInstructions("")
		if result != "" {
			t.Error("should return empty string for empty input")
		}
	})

	t.Run("handles content with no beads instructions", func(t *testing.T) {
		content := `## Investigation Phase

Analyze the code.

### Steps

1. Read the file
2. Test the hypothesis
3. Document findings
`
		result := StripBeadsInstructions(content)

		// Should be mostly unchanged (some whitespace normalization may occur)
		if !strings.Contains(result, "## Investigation Phase") {
			t.Error("should keep content unchanged when no beads instructions")
		}
		if !strings.Contains(result, "### Steps") {
			t.Error("should keep all sections")
		}
	})

	t.Run("removes Beads Progress Tracking section", func(t *testing.T) {
		content := `## Self-Review

Check your work.

### Beads Progress Tracking

Use beads to track:
- Phase transitions
- Completions

### Next Steps

Continue work.
`
		result := StripBeadsInstructions(content)

		// Should remove the beads tracking section
		if strings.Contains(result, "### Beads Progress Tracking") {
			t.Error("should remove Beads Progress Tracking section")
		}

		// Should keep other sections
		if !strings.Contains(result, "## Self-Review") {
			t.Error("should keep Self-Review section")
		}
		if !strings.Contains(result, "### Next Steps") {
			t.Error("should keep Next Steps section")
		}
	})
}

func TestGenerateContext_NoTrackStripsSkillBeadsInstructions(t *testing.T) {
	// Skill content with beads instructions (simulating real skill content)
	skillContent := `---
name: systematic-debugging
---

# Systematic Debugging

## Self-Review

Check your work.

### Report via Beads

` + "```" + `bash
# If issues found and fixed:
bd comment <beads-id> "Self-review: Fixed [issue summary]"

# If passed:
bd comment <beads-id> "Self-review passed - ready for completion"
` + "```" + `

## Completion Criteria

- [ ] **Root cause identified**
- [ ] **Fix implemented**
- [ ] **Reported** - ` + "`bd comment <beads-id> \"Phase: Complete - [summary]\"`" + `

### After All Criteria Met

` + "```" + `bash
bd comment <beads-id> "Phase: Complete - Root cause: [X], Fix: [Y], Tests passing"
bd close <beads-id> --reason "Root cause: [X], Fix: [Y]"
` + "```" + `

Then call ` + "`/exit`" + ` to close agent session.
`

	cfg := &Config{
		Task:          "debug an issue",
		SkillName:     "systematic-debugging",
		Project:       "test-project",
		ProjectDir:    "/tmp/test",
		WorkspaceName: "og-debug-test-26dec",
		SkillContent:  skillContent,
		NoTrack:       true,
		Tier:          TierFull,
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	// Should contain skill guidance section
	if !strings.Contains(content, "## SKILL GUIDANCE (systematic-debugging)") {
		t.Error("expected content to contain skill guidance section")
	}

	// Should NOT contain beads commands from skill content
	if strings.Contains(content, "bd comment <beads-id>") {
		t.Error("expected skill content to have beads commands stripped for --no-track spawn")
	}
	if strings.Contains(content, "bd close <beads-id>") {
		t.Error("expected skill content to have bd close commands stripped for --no-track spawn")
	}

	// Should still contain the main skill guidance (non-beads parts)
	if !strings.Contains(content, "# Systematic Debugging") {
		t.Error("expected content to still contain skill title")
	}
	if !strings.Contains(content, "## Self-Review") {
		t.Error("expected content to still contain Self-Review section")
	}
	if !strings.Contains(content, "## Completion Criteria") {
		t.Error("expected content to still contain Completion Criteria section")
	}

	// Should keep /exit instruction
	if !strings.Contains(content, "/exit") {
		t.Error("expected content to keep /exit instruction")
	}

	// Should keep non-beads completion criteria
	if !strings.Contains(content, "**Root cause identified**") {
		t.Error("expected content to keep non-beads completion criteria")
	}
}

func TestGenerateContext_TrackedKeepsSkillBeadsInstructions(t *testing.T) {
	// Same skill content as above
	skillContent := `---
name: systematic-debugging
---

# Systematic Debugging

### Report via Beads

` + "```" + `bash
bd comment <beads-id> "Phase: Complete"
` + "```" + `
`

	cfg := &Config{
		Task:          "debug an issue",
		SkillName:     "systematic-debugging",
		Project:       "test-project",
		ProjectDir:    "/tmp/test",
		WorkspaceName: "og-debug-test-26dec",
		SkillContent:  skillContent,
		BeadsID:       "test-123",
		NoTrack:       false, // Tracked spawn
		Tier:          TierFull,
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	// Should contain beads commands from skill content (not stripped)
	if !strings.Contains(content, "bd comment <beads-id>") {
		t.Error("expected skill content to keep beads commands for tracked spawn")
	}
}

func TestGenerateContext_SurfaceBeforeCircumvent(t *testing.T) {
	t.Run("includes Surface Before Circumvent section with beads tracking", func(t *testing.T) {
		cfg := &Config{
			Task:          "implement feature",
			SkillName:     "feature-impl",
			Project:       "test-project",
			ProjectDir:    "/tmp/test",
			WorkspaceName: "og-feat-test-26dec",
			BeadsID:       "test-123",
			NoTrack:       false,
			Tier:          TierLight,
		}

		content, err := GenerateContext(cfg)
		if err != nil {
			t.Fatalf("GenerateContext failed: %v", err)
		}

		// Should contain Surface Before Circumvent section
		if !strings.Contains(content, "**Surface Before Circumvent:**") {
			t.Error("expected content to contain Surface Before Circumvent section")
		}

		// Should contain beads comment instruction for surfacing constraints
		if !strings.Contains(content, "bd comment test-123 \"CONSTRAINT:") {
			t.Error("expected content to contain bd comment CONSTRAINT instruction with beads ID")
		}

		// Should contain the "wait for orchestrator" instruction
		if !strings.Contains(content, "Wait for orchestrator acknowledgment") {
			t.Error("expected content to contain wait for orchestrator instruction")
		}

		// Should contain examples of what constraints apply to
		if !strings.Contains(content, "System constraints discovered during work") {
			t.Error("expected content to contain examples of constraint types")
		}

		// Should contain the "why" explanation
		if !strings.Contains(content, "Prevents the system from learning") {
			t.Error("expected content to explain why surfacing matters")
		}
	})

	t.Run("uses investigation file fallback for no-track spawn", func(t *testing.T) {
		cfg := &Config{
			Task:          "quick investigation",
			SkillName:     "investigation",
			Project:       "test-project",
			ProjectDir:    "/tmp/test",
			WorkspaceName: "og-inv-test-26dec",
			NoTrack:       true,
			Tier:          TierFull,
		}

		content, err := GenerateContext(cfg)
		if err != nil {
			t.Fatalf("GenerateContext failed: %v", err)
		}

		// Should contain Surface Before Circumvent section
		if !strings.Contains(content, "**Surface Before Circumvent:**") {
			t.Error("expected content to contain Surface Before Circumvent section")
		}

		// Should NOT contain bd comment CONSTRAINT instruction (no-track)
		// The pattern we're looking for is the actual beads command: `bd comment ... "CONSTRAINT:`
		// (not the informational message "Progress tracking via bd comment is NOT available")
		if strings.Contains(content, `bd comment og-inv-test-26dec "CONSTRAINT:`) {
			t.Error("expected content to NOT contain bd comment CONSTRAINT instruction for --no-track spawn")
		}

		// Should contain investigation file fallback
		if !strings.Contains(content, "Document it in your investigation file") {
			t.Error("expected content to contain investigation file fallback for --no-track spawn")
		}

		// Should mention SYNTHESIS.md for documentation
		if !strings.Contains(content, "Include the constraint and your reasoning in SYNTHESIS.md") {
			t.Error("expected content to mention SYNTHESIS.md for constraint documentation")
		}
	})
}

func TestGenerateContext_NoTrack(t *testing.T) {
	t.Run("excludes beads instructions when NoTrack is true", func(t *testing.T) {
		cfg := &Config{
			Task:          "quick ad-hoc task",
			SkillName:     "investigation",
			Project:       "test-project",
			ProjectDir:    "/tmp/test",
			WorkspaceName: "og-inv-test-26dec",
			NoTrack:       true,
			Tier:          TierFull,
		}

		content, err := GenerateContext(cfg)
		if err != nil {
			t.Fatalf("GenerateContext failed: %v", err)
		}

		// Should contain no-track indicator
		if !strings.Contains(content, "AD-HOC SPAWN (--no-track)") {
			t.Error("expected content to contain ad-hoc spawn indicator")
		}

		// Should NOT contain beads tracking section
		if strings.Contains(content, "## BEADS PROGRESS TRACKING") {
			t.Error("expected content to NOT contain beads tracking section for --no-track spawn")
		}

		// Should NOT contain bd comment instructions (backtick-quoted command examples)
		// Note: The informational message "Progress tracking via bd comment is NOT available" is OK
		if strings.Contains(content, "`bd comment") {
			t.Error("expected content to NOT contain bd comment command instructions for --no-track spawn")
		}

		// Should NOT contain bd close warning
		if strings.Contains(content, "bd close") {
			t.Error("expected content to NOT contain bd close warning for --no-track spawn")
		}

		// Should still contain /exit instruction
		if !strings.Contains(content, "/exit") {
			t.Error("expected content to contain /exit instruction")
		}

		// Should still contain SYNTHESIS.md requirement for full tier
		if !strings.Contains(content, "SYNTHESIS.md") {
			t.Error("expected content to contain SYNTHESIS.md requirement for full tier")
		}

		// Should contain workspace phase file instructions for untracked agents
		if !strings.Contains(content, "PHASE REPORTING (WORKSPACE FILE)") {
			t.Error("expected content to contain phase reporting instructions for --no-track spawn")
		}

		// Should contain echo command to write phase to .phase file
		if !strings.Contains(content, ".phase") {
			t.Error("expected content to contain .phase file reference for --no-track spawn")
		}

		// Should contain full workspace path in phase instructions
		expectedWorkspacePath := "/tmp/test/.orch/workspace/og-inv-test-26dec"
		if !strings.Contains(content, expectedWorkspacePath) {
			t.Errorf("expected content to contain workspace path %q", expectedWorkspacePath)
		}
	})

	t.Run("includes beads instructions when NoTrack is false", func(t *testing.T) {
		cfg := &Config{
			Task:          "tracked task",
			SkillName:     "investigation",
			Project:       "test-project",
			ProjectDir:    "/tmp/test",
			WorkspaceName: "og-inv-test-26dec",
			BeadsID:       "test-123",
			NoTrack:       false,
			Tier:          TierFull,
		}

		content, err := GenerateContext(cfg)
		if err != nil {
			t.Fatalf("GenerateContext failed: %v", err)
		}

		// Should NOT contain no-track indicator
		if strings.Contains(content, "AD-HOC SPAWN (--no-track)") {
			t.Error("expected content to NOT contain ad-hoc spawn indicator for tracked spawn")
		}

		// Should contain beads tracking section
		if !strings.Contains(content, "## BEADS PROGRESS TRACKING") {
			t.Error("expected content to contain beads tracking section for tracked spawn")
		}

		// Should contain bd comment instructions with correct beads ID
		if !strings.Contains(content, "bd comment test-123") {
			t.Error("expected content to contain bd comment instructions with beads ID")
		}

		// Should contain bd close warning
		if !strings.Contains(content, "NEVER run `bd close`") {
			t.Error("expected content to contain bd close warning")
		}
	})

	t.Run("light tier no-track omits SYNTHESIS.md requirement", func(t *testing.T) {
		cfg := &Config{
			Task:          "quick fix",
			SkillName:     "feature-impl",
			Project:       "test-project",
			ProjectDir:    "/tmp/test",
			WorkspaceName: "og-feat-test-26dec",
			NoTrack:       true,
			Tier:          TierLight,
		}

		content, err := GenerateContext(cfg)
		if err != nil {
			t.Fatalf("GenerateContext failed: %v", err)
		}

		// Should indicate SYNTHESIS.md is not required
		if !strings.Contains(content, "SYNTHESIS.md is NOT required") {
			t.Error("expected content to indicate SYNTHESIS.md is not required for light tier")
		}

		// Final protocol should only have /exit
		// Check that the final protocol section doesn't mention bd comment
		finalIdx := strings.LastIndex(content, "FINAL STEP - SESSION COMPLETE PROTOCOL")
		if finalIdx == -1 {
			t.Fatal("expected content to contain final step protocol")
		}
		finalSection := content[finalIdx:]
		if strings.Contains(finalSection, "bd comment") {
			t.Error("final protocol should not contain bd comment for --no-track spawn")
		}
	})
}

// NOTE: Ecosystem context tests have been moved to ecosystem_test.go

func TestGenerateContext_WithEcosystemContext(t *testing.T) {
	// This test verifies ecosystem context is included in spawn context
	// when the project is part of the ecosystem AND ~/.orch/ECOSYSTEM.md exists

	cfg := &Config{
		Task:       "test ecosystem integration",
		SkillName:  "investigation",
		Project:    "orch-go", // Must be an ecosystem repo
		ProjectDir: "/tmp/test",
		BeadsID:    "test-123",
		Tier:       TierFull,
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	// If ecosystem file exists, content should include ecosystem section
	ecosystemContext := GenerateEcosystemContext()
	if ecosystemContext != "" {
		if !strings.Contains(content, "LOCAL PROJECT ECOSYSTEM") {
			t.Error("expected content to contain LOCAL PROJECT ECOSYSTEM section when file exists")
		}
		// Should include key repos from ecosystem
		if !strings.Contains(content, "orch-go") {
			t.Error("expected ecosystem context to mention orch-go")
		}
	}
}

func TestGenerateContext_WithoutEcosystemContext_NonEcosystemRepo(t *testing.T) {
	// This test verifies ecosystem context is NOT included when
	// the project is NOT part of the orchestration ecosystem

	cfg := &Config{
		Task:       "test non-ecosystem project",
		SkillName:  "investigation",
		Project:    "random-external-project", // Not an ecosystem repo
		ProjectDir: "/tmp/test",
		BeadsID:    "test-123",
		Tier:       TierFull,
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	// Should NOT contain ecosystem context for non-ecosystem projects
	if strings.Contains(content, "LOCAL PROJECT ECOSYSTEM") {
		t.Error("expected non-ecosystem project to NOT contain LOCAL PROJECT ECOSYSTEM section")
	}
}

// NOTE: TestGenerateEcosystemContext_Integration has been moved to ecosystem_test.go

func TestGenerateBehavioralPatternsContext(t *testing.T) {
	t.Run("returns empty when no action log exists", func(t *testing.T) {
		// Use a custom path that doesn't exist
		tempDir := t.TempDir()
		nonExistentPath := filepath.Join(tempDir, "action-log.jsonl")

		// Override the logger path for this test
		originalPathFunc := action.GetLoggerPathFunc()
		action.SetLoggerPathFunc(func() string { return nonExistentPath })
		defer action.SetLoggerPathFunc(originalPathFunc)

		result := GenerateBehavioralPatternsContext("test-workspace")
		if result != "" {
			t.Error("expected empty string when no action log exists")
		}
	})

	t.Run("returns empty when no patterns detected", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "action-log.jsonl")

		// Create an action log with only success events (no patterns)
		logger := action.NewLogger(logPath)
		if err := logger.LogSuccess("Read", "/some/file.go"); err != nil {
			t.Fatalf("failed to log: %v", err)
		}
		if err := logger.LogSuccess("Bash", "go test"); err != nil {
			t.Fatalf("failed to log: %v", err)
		}

		// Override the logger path
		originalPathFunc := action.GetLoggerPathFunc()
		action.SetLoggerPathFunc(func() string { return logPath })
		defer action.SetLoggerPathFunc(originalPathFunc)

		result := GenerateBehavioralPatternsContext("test-workspace")
		if result != "" {
			t.Error("expected empty string when no patterns (only successes)")
		}
	})

	t.Run("returns patterns when futile actions detected", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "action-log.jsonl")

		// Create an action log with repeated empty results (futile action)
		logger := action.NewLogger(logPath)
		for i := 0; i < 5; i++ {
			if err := logger.LogEmpty("Read", "SYNTHESIS.md"); err != nil {
				t.Fatalf("failed to log: %v", err)
			}
		}

		// Override the logger path
		originalPathFunc := action.GetLoggerPathFunc()
		action.SetLoggerPathFunc(func() string { return logPath })
		defer action.SetLoggerPathFunc(originalPathFunc)

		result := GenerateBehavioralPatternsContext("test-workspace")
		if result == "" {
			t.Error("expected patterns when futile actions detected")
		}
		if !strings.Contains(result, "Read") {
			t.Error("expected pattern to contain tool name 'Read'")
		}
		// Target is kept as-is when not a full path (no slashes)
		if !strings.Contains(result, "SYNTHESIS.md") {
			t.Errorf("expected pattern to contain target (got: %s)", result)
		}
		if !strings.Contains(result, "returns empty") {
			t.Error("expected pattern to describe outcome")
		}
	})

	t.Run("limits patterns to avoid context bloat", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "action-log.jsonl")

		// Create many different patterns
		logger := action.NewLogger(logPath)
		for i := 0; i < 10; i++ {
			target := fmt.Sprintf("file%d.txt", i)
			for j := 0; j < 5; j++ {
				if err := logger.LogEmpty("Read", target); err != nil {
					t.Fatalf("failed to log: %v", err)
				}
			}
		}

		// Override the logger path
		originalPathFunc := action.GetLoggerPathFunc()
		action.SetLoggerPathFunc(func() string { return logPath })
		defer action.SetLoggerPathFunc(originalPathFunc)

		result := GenerateBehavioralPatternsContext("test-workspace")
		if result == "" {
			t.Error("expected patterns")
		}
		// Should contain message about more patterns
		if !strings.Contains(result, "more patterns") {
			t.Error("expected 'more patterns' message when >5 patterns exist")
		}
	})
}

func TestGenerateContext_WithBehavioralPatterns(t *testing.T) {
	t.Run("includes behavioral patterns when detected", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "action-log.jsonl")

		// Create an action log with futile actions
		logger := action.NewLogger(logPath)
		for i := 0; i < 5; i++ {
			if err := logger.LogEmpty("Read", "MISSING.md"); err != nil {
				t.Fatalf("failed to log: %v", err)
			}
		}

		// Override the logger path
		originalPathFunc := action.GetLoggerPathFunc()
		action.SetLoggerPathFunc(func() string { return logPath })
		defer action.SetLoggerPathFunc(originalPathFunc)

		cfg := &Config{
			Task:          "test task",
			SkillName:     "feature-impl",
			Project:       "test-project",
			ProjectDir:    tempDir,
			WorkspaceName: "og-test-workspace",
			BeadsID:       "test-123",
			Tier:          TierLight,
		}

		content, err := GenerateContext(cfg)
		if err != nil {
			t.Fatalf("GenerateContext failed: %v", err)
		}

		// Should contain behavioral patterns section
		if !strings.Contains(content, "BEHAVIORAL PATTERNS WARNING") {
			t.Error("expected content to contain BEHAVIORAL PATTERNS WARNING section")
		}
		if !strings.Contains(content, "Why this matters") {
			t.Error("expected content to contain explanation")
		}
	})

	t.Run("excludes behavioral patterns when none detected", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "action-log.jsonl")

		// Create empty action log (or just success events)
		logger := action.NewLogger(logPath)
		if err := logger.LogSuccess("Read", "exists.go"); err != nil {
			t.Fatalf("failed to log: %v", err)
		}

		// Override the logger path
		originalPathFunc := action.GetLoggerPathFunc()
		action.SetLoggerPathFunc(func() string { return logPath })
		defer action.SetLoggerPathFunc(originalPathFunc)

		cfg := &Config{
			Task:          "test task",
			SkillName:     "feature-impl",
			Project:       "test-project",
			ProjectDir:    tempDir,
			WorkspaceName: "og-test-workspace",
			BeadsID:       "test-123",
			Tier:          TierLight,
		}

		content, err := GenerateContext(cfg)
		if err != nil {
			t.Fatalf("GenerateContext failed: %v", err)
		}

		// Should NOT contain behavioral patterns section
		if strings.Contains(content, "BEHAVIORAL PATTERNS WARNING") {
			t.Error("expected content to NOT contain BEHAVIORAL PATTERNS WARNING when no patterns")
		}
	})

	t.Run("uses provided behavioral patterns over auto-generated", func(t *testing.T) {
		cfg := &Config{
			Task:               "test task",
			SkillName:          "feature-impl",
			Project:            "test-project",
			ProjectDir:         "/tmp/test",
			WorkspaceName:      "og-test-workspace",
			BeadsID:            "test-123",
			Tier:               TierLight,
			BehavioralPatterns: "Custom pattern warning here",
		}

		content, err := GenerateContext(cfg)
		if err != nil {
			t.Fatalf("GenerateContext failed: %v", err)
		}

		// Should contain the custom patterns
		if !strings.Contains(content, "Custom pattern warning here") {
			t.Error("expected content to contain custom behavioral patterns")
		}
	})
}

func TestCheckFailureReport(t *testing.T) {
	t.Run("returns empty status when no workspace exists", func(t *testing.T) {
		tempDir := t.TempDir()
		status := CheckFailureReport(tempDir, "test-123")

		if status.Exists {
			t.Error("expected Exists to be false when no workspace exists")
		}
		if status.IsFilled {
			t.Error("expected IsFilled to be false when no report exists")
		}
	})

	t.Run("returns empty status when workspace exists but no FAILURE_REPORT.md", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create workspace directory with SPAWN_CONTEXT.md
		workspacePath := filepath.Join(tempDir, ".orch", "workspace", "og-test-workspace")
		if err := os.MkdirAll(workspacePath, 0755); err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		// Create SPAWN_CONTEXT.md with beads ID reference
		spawnContext := "You were spawned from beads issue: **test-123**\n\nSome other content."
		if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
			t.Fatalf("failed to write SPAWN_CONTEXT.md: %v", err)
		}

		status := CheckFailureReport(tempDir, "test-123")

		if status.Exists {
			t.Error("expected Exists to be false when no FAILURE_REPORT.md exists")
		}
	})

	t.Run("detects unfilled failure report", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create workspace directory with SPAWN_CONTEXT.md
		workspacePath := filepath.Join(tempDir, ".orch", "workspace", "og-test-workspace")
		if err := os.MkdirAll(workspacePath, 0755); err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		// Create SPAWN_CONTEXT.md with beads ID reference
		spawnContext := "You were spawned from beads issue: **test-123**\n\nSome other content."
		if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
			t.Fatalf("failed to write SPAWN_CONTEXT.md: %v", err)
		}

		// Create unfilled FAILURE_REPORT.md (with placeholders)
		failureReport := `# Failure Report

**Agent:** og-test-workspace
**Issue:** test-123
**Abandoned:** 2025-12-29 10:00:00
**Reason:** Out of context

---

## Context

**Task:** implement feature

**What was attempted:**
[Brief description of what the agent was trying to do]

---

## Failure Summary

**Primary Cause:** Out of context

**Details:**
[Describe what went wrong - symptoms observed, errors encountered, or why the agent was stuck]

---

## Learnings

**Root cause analysis:**
- [If known, why did this fail? External dependency? Tool issue? Scope creep? Context exhaustion?]

---

## Recovery Recommendations

**If yes, what should be different:**
- [Suggestion 1 - different approach]
`
		if err := os.WriteFile(filepath.Join(workspacePath, "FAILURE_REPORT.md"), []byte(failureReport), 0644); err != nil {
			t.Fatalf("failed to write FAILURE_REPORT.md: %v", err)
		}

		status := CheckFailureReport(tempDir, "test-123")

		if !status.Exists {
			t.Error("expected Exists to be true")
		}
		if status.IsFilled {
			t.Error("expected IsFilled to be false for unfilled report")
		}
		if status.WhatWasAttempted {
			t.Error("expected WhatWasAttempted to be false")
		}
		if status.Details {
			t.Error("expected Details to be false")
		}
		if status.RootCauseAnalysis {
			t.Error("expected RootCauseAnalysis to be false")
		}
		if status.WhatShouldDifferent {
			t.Error("expected WhatShouldDifferent to be false")
		}
		if len(status.UnfilledSections) != 4 {
			t.Errorf("expected 4 unfilled sections, got %d: %v", len(status.UnfilledSections), status.UnfilledSections)
		}
	})

	t.Run("detects filled failure report", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create workspace directory with SPAWN_CONTEXT.md
		workspacePath := filepath.Join(tempDir, ".orch", "workspace", "og-test-workspace")
		if err := os.MkdirAll(workspacePath, 0755); err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		// Create SPAWN_CONTEXT.md with beads ID reference
		spawnContext := "You were spawned from beads issue: **test-123**\n\nSome other content."
		if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
			t.Fatalf("failed to write SPAWN_CONTEXT.md: %v", err)
		}

		// Create filled FAILURE_REPORT.md (no placeholders)
		failureReport := `# Failure Report

**Agent:** og-test-workspace
**Issue:** test-123
**Abandoned:** 2025-12-29 10:00:00
**Reason:** Out of context

---

## Context

**Task:** implement feature

**What was attempted:**
Started implementing the authentication middleware but ran out of context.

---

## Failure Summary

**Primary Cause:** Out of context

**Details:**
The agent consumed too much context reading test files before starting implementation.

---

## Learnings

**Root cause analysis:**
- Context exhaustion from reading too many files upfront
- Should have focused on smaller scope first

---

## Recovery Recommendations

**If yes, what should be different:**
- Start with a smaller scope (auth middleware only, no tests)
- Use targeted file reads instead of exploring broadly
`
		if err := os.WriteFile(filepath.Join(workspacePath, "FAILURE_REPORT.md"), []byte(failureReport), 0644); err != nil {
			t.Fatalf("failed to write FAILURE_REPORT.md: %v", err)
		}

		status := CheckFailureReport(tempDir, "test-123")

		if !status.Exists {
			t.Error("expected Exists to be true")
		}
		if !status.IsFilled {
			t.Error("expected IsFilled to be true for filled report")
		}
		if !status.WhatWasAttempted {
			t.Error("expected WhatWasAttempted to be true")
		}
		if !status.Details {
			t.Error("expected Details to be true")
		}
		if !status.RootCauseAnalysis {
			t.Error("expected RootCauseAnalysis to be true")
		}
		if !status.WhatShouldDifferent {
			t.Error("expected WhatShouldDifferent to be true")
		}
		if len(status.UnfilledSections) != 0 {
			t.Errorf("expected 0 unfilled sections, got %d: %v", len(status.UnfilledSections), status.UnfilledSections)
		}
	})

	t.Run("ignores workspaces for different beads ID", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create workspace directory with SPAWN_CONTEXT.md for different beads ID
		workspacePath := filepath.Join(tempDir, ".orch", "workspace", "og-other-workspace")
		if err := os.MkdirAll(workspacePath, 0755); err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		// Create SPAWN_CONTEXT.md with DIFFERENT beads ID
		spawnContext := "You were spawned from beads issue: **other-456**\n\nSome other content."
		if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
			t.Fatalf("failed to write SPAWN_CONTEXT.md: %v", err)
		}

		// Create FAILURE_REPORT.md in this workspace
		failureReport := `# Failure Report
**Issue:** other-456
**What was attempted:**
[Brief description of what the agent was trying to do]
`
		if err := os.WriteFile(filepath.Join(workspacePath, "FAILURE_REPORT.md"), []byte(failureReport), 0644); err != nil {
			t.Fatalf("failed to write FAILURE_REPORT.md: %v", err)
		}

		// Check for a different beads ID - should NOT find the failure report
		status := CheckFailureReport(tempDir, "test-123")

		if status.Exists {
			t.Error("expected Exists to be false when no workspace matches the beads ID")
		}
	})

	t.Run("detects partially filled failure report", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create workspace directory with SPAWN_CONTEXT.md
		workspacePath := filepath.Join(tempDir, ".orch", "workspace", "og-test-workspace")
		if err := os.MkdirAll(workspacePath, 0755); err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		// Create SPAWN_CONTEXT.md with beads ID reference
		spawnContext := "You were spawned from beads issue: **test-123**\n\nSome other content."
		if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
			t.Fatalf("failed to write SPAWN_CONTEXT.md: %v", err)
		}

		// Create partially filled FAILURE_REPORT.md (some placeholders remain)
		failureReport := `# Failure Report

**What was attempted:**
I was trying to implement the auth middleware.

**Details:**
[Describe what went wrong - symptoms observed, errors encountered, or why the agent was stuck]

**Root cause analysis:**
- Context exhaustion, ran out of context after 2 hours

**If yes, what should be different:**
[Suggestion 1 - different approach]
`
		if err := os.WriteFile(filepath.Join(workspacePath, "FAILURE_REPORT.md"), []byte(failureReport), 0644); err != nil {
			t.Fatalf("failed to write FAILURE_REPORT.md: %v", err)
		}

		status := CheckFailureReport(tempDir, "test-123")

		if !status.Exists {
			t.Error("expected Exists to be true")
		}
		if status.IsFilled {
			t.Error("expected IsFilled to be false for partially filled report")
		}
		if !status.WhatWasAttempted {
			t.Error("expected WhatWasAttempted to be true (was filled)")
		}
		if status.Details {
			t.Error("expected Details to be false (still has placeholder)")
		}
		if !status.RootCauseAnalysis {
			t.Error("expected RootCauseAnalysis to be true (was filled)")
		}
		if status.WhatShouldDifferent {
			t.Error("expected WhatShouldDifferent to be false (still has placeholder)")
		}
		if len(status.UnfilledSections) != 2 {
			t.Errorf("expected 2 unfilled sections, got %d: %v", len(status.UnfilledSections), status.UnfilledSections)
		}
	})
}

func TestFormatFailureReportGateError(t *testing.T) {
	status := &FailureReportStatus{
		Exists:           true,
		FilePath:         "/tmp/test/.orch/workspace/og-test/FAILURE_REPORT.md",
		WorkspaceName:    "og-test",
		IsFilled:         false,
		UnfilledSections: []string{"What was attempted", "Root cause analysis"},
	}

	errMsg := FormatFailureReportGateError(status, "test-123")

	// Check error message contains key information
	if !strings.Contains(errMsg, "FAILURE_REPORT.md has unfilled sections") {
		t.Error("expected error to mention unfilled sections")
	}
	if !strings.Contains(errMsg, status.FilePath) {
		t.Error("expected error to contain file path")
	}
	if !strings.Contains(errMsg, "What was attempted") {
		t.Error("expected error to list unfilled section")
	}
	if !strings.Contains(errMsg, "Root cause analysis") {
		t.Error("expected error to list unfilled section")
	}
	if !strings.Contains(errMsg, "--skip-failure-review") {
		t.Error("expected error to mention bypass flag")
	}
	if !strings.Contains(errMsg, "test-123") {
		t.Error("expected error to include beads ID")
	}
}
