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
		"CREATE SYNTHESIS.md",
		"SYNTHESIS.md is created and committed",
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
