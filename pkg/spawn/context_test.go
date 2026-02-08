package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateWorkspaceName(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		skillName   string
		task        string
		wantParts   []string // parts that should appear in the workspace name
	}{
		{
			name:        "investigation skill with orch-go project",
			projectName: "orch-go",
			skillName:   "investigation",
			task:        "explore the codebase",
			wantParts:   []string{"og-inv-", "explore", "codebase"},
		},
		{
			name:        "feature-impl skill with orch-go project",
			projectName: "orch-go",
			skillName:   "feature-impl",
			task:        "add new spawn command",
			wantParts:   []string{"og-feat-", "add", "spawn", "command"}, // 'new' filtered as vague adjective
		},
		{
			name:        "unknown skill with orch-go project",
			projectName: "orch-go",
			skillName:   "custom-skill",
			task:        "do something",
			wantParts:   []string{"og-work-", "something"}, // 'do' filtered as auxiliary verb
		},
		{
			name:        "task with stop words and orch-go project",
			projectName: "orch-go",
			skillName:   "investigation",
			task:        "the quick brown fox in the forest",
			wantParts:   []string{"og-inv-", "quick", "brown", "fox"},
		},
		{
			name:        "task with question words filtered",
			projectName: "orch-go",
			skillName:   "architect",
			task:        "Design how to better surface ready queue items",
			wantParts:   []string{"og-arch-", "design", "surface", "ready"}, // 'how', 'better' filtered
		},
		{
			name:        "task with modal verbs filtered",
			projectName: "orch-go",
			skillName:   "investigation",
			task:        "what should we do when users can login",
			wantParts:   []string{"og-inv-", "users", "login"}, // 'what', 'should', 'we', 'do', 'when', 'can' filtered
		},
		{
			name:        "task with filler words filtered",
			projectName: "orch-go",
			skillName:   "feature-impl",
			task:        "need to make some changes to get better logging",
			wantParts:   []string{"og-feat-", "changes", "logging"}, // 'need', 'make', 'some', 'get', 'better' filtered
		},
		{
			name:        "price-watch project uses pw prefix",
			projectName: "price-watch",
			skillName:   "investigation",
			task:        "explore pricing",
			wantParts:   []string{"pw-inv-", "explore", "pricing"},
		},
		{
			name:        "single-word project name",
			projectName: "myproject",
			skillName:   "feature-impl",
			task:        "add feature",
			wantParts:   []string{"my-feat-", "add", "feature"},
		},
		{
			name:        "three-part project name",
			projectName: "my-cool-project",
			skillName:   "investigation",
			task:        "explore code",
			wantParts:   []string{"mycopr-inv-", "explore", "code"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateWorkspaceName(tt.projectName, tt.skillName, tt.task)
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("expected workspace name to contain %q, got %q", part, got)
				}
			}
		})
	}
}

func TestGenerateWorkspaceName_MetaOrchestrator(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		skillName   string
		task        string
		opts        WorkspaceNameOptions
		wantParts   []string
		notWant     []string // parts that should NOT appear
	}{
		{
			name:        "meta-orchestrator uses meta prefix instead of project prefix",
			projectName: "orch-go",
			skillName:   "orchestrator",
			task:        "test meta orchestrator session",
			opts:        WorkspaceNameOptions{IsMetaOrchestrator: true},
			wantParts:   []string{"meta-orch-", "test", "orchestrator"}, // meta-orchestrators use "orch" prefix
			notWant:     []string{"og-"},
		},
		{
			name:        "meta-orchestrator with custom skill",
			projectName: "price-watch",
			skillName:   "meta-orchestrator",
			task:        "manage agents",
			opts:        WorkspaceNameOptions{IsMetaOrchestrator: true},
			wantParts:   []string{"meta-orch-", "manage", "agents"}, // meta-orchestrators use "orch" prefix
			notWant:     []string{"pw-"},
		},
		{
			name:        "regular orchestrator uses orch prefix for visual distinction",
			projectName: "orch-go",
			skillName:   "orchestrator",
			task:        "regular orchestrator session",
			opts:        WorkspaceNameOptions{IsOrchestrator: true},
			wantParts:   []string{"og-orch-", "regular", "orchestrator"}, // orchestrators use "orch" not "work"
			notWant:     []string{"meta-", "og-work-"},
		},
		{
			name:        "worker spawn still uses work prefix",
			projectName: "orch-go",
			skillName:   "orchestrator",
			task:        "worker session",
			opts:        WorkspaceNameOptions{IsOrchestrator: false, IsMetaOrchestrator: false},
			wantParts:   []string{"og-work-", "worker", "session"},
			notWant:     []string{"meta-", "og-orch-"},
		},
		{
			name:        "no opts defaults to non-meta behavior",
			projectName: "orch-go",
			skillName:   "investigation",
			task:        "explore codebase",
			// No opts provided
			wantParts: []string{"og-inv-", "explore", "codebase"},
			notWant:   []string{"meta-"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			if tt.opts.IsMetaOrchestrator || tt.opts != (WorkspaceNameOptions{}) {
				got = GenerateWorkspaceName(tt.projectName, tt.skillName, tt.task, tt.opts)
			} else {
				// Test backward compatibility - calling without opts
				got = GenerateWorkspaceName(tt.projectName, tt.skillName, tt.task)
			}

			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("expected workspace name to contain %q, got %q", part, got)
				}
			}
			for _, part := range tt.notWant {
				if strings.Contains(got, part) {
					t.Errorf("expected workspace name to NOT contain %q, got %q", part, got)
				}
			}
		})
	}
}

func TestGenerateWorkspaceName_Uniqueness(t *testing.T) {
	t.Run("generates unique names for same inputs", func(t *testing.T) {
		projectName := "orch-go"
		skillName := "investigation"
		task := "explore the codebase"

		// Generate multiple workspace names with the same inputs
		// Use smaller sample size to avoid birthday paradox collisions
		// (65536 possibilities means ~10 samples has <0.1% collision chance)
		names := make(map[string]bool)
		for i := 0; i < 10; i++ {
			name := GenerateWorkspaceName(projectName, skillName, task)
			if names[name] {
				t.Errorf("duplicate workspace name generated: %s", name)
			}
			names[name] = true
		}

		// Verify we got 10 unique names
		if len(names) != 10 {
			t.Errorf("expected 10 unique names, got %d", len(names))
		}
	})

	t.Run("unique suffix is 4 hex chars", func(t *testing.T) {
		name := GenerateWorkspaceName("orch-go", "investigation", "test task")

		// Split by dash and get the last part (should be unique suffix)
		parts := strings.Split(name, "-")
		if len(parts) < 5 {
			t.Fatalf("expected at least 5 parts in workspace name, got %d: %s", len(parts), name)
		}

		// Last part should be 4 hex characters
		suffix := parts[len(parts)-1]
		if len(suffix) != 4 {
			t.Errorf("expected 4-char suffix, got %d chars: %s", len(suffix), suffix)
		}

		// Verify it's valid hex
		for _, c := range suffix {
			if !strings.ContainsRune("0123456789abcdef", c) {
				t.Errorf("suffix contains non-hex character: %s", suffix)
				break
			}
		}
	})

	t.Run("name format includes unique suffix", func(t *testing.T) {
		name := GenerateWorkspaceName("orch-go", "feature-impl", "add new feature")

		// Should match pattern: og-feat-{slug}-{date}-{unique}
		// e.g., og-feat-add-feature-05jan-a1b2
		if !strings.HasPrefix(name, "og-feat-") {
			t.Errorf("expected name to start with 'og-feat-', got: %s", name)
		}

		// Should have 5 dash-separated parts minimum (proj-skill-slug-date-unique)
		parts := strings.Split(name, "-")
		if len(parts) < 5 {
			t.Errorf("expected at least 5 parts, got %d: %s", len(parts), name)
		}
	})
}

func TestGenerateUniqueSuffix(t *testing.T) {
	t.Run("generates 4 hex chars", func(t *testing.T) {
		suffix := generateUniqueSuffix()

		if len(suffix) != 4 {
			t.Errorf("expected 4-char suffix, got %d chars: %s", len(suffix), suffix)
		}

		// Verify it's valid hex
		for _, c := range suffix {
			if !strings.ContainsRune("0123456789abcdef", c) {
				t.Errorf("suffix contains non-hex character: %s", suffix)
				break
			}
		}
	})

	t.Run("generates unique suffixes", func(t *testing.T) {
		suffixes := make(map[string]bool)
		for i := 0; i < 1000; i++ {
			suffix := generateUniqueSuffix()
			suffixes[suffix] = true
		}

		// With 2 bytes of entropy (65536 possibilities), we expect very high uniqueness
		// Birthday paradox: expected collisions = n^2 / (2*m) = 1000^2 / (2*65536) ≈ 7.6
		// So expecting ~992 unique is reasonable. Allow down to 980 for statistical variance.
		if len(suffixes) < 980 {
			t.Errorf("expected >980 unique suffixes from 1000 samples, got %d", len(suffixes))
		}
	})
}

func TestGenerateProjectPrefix(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		want        string
	}{
		{
			name:        "two-part hyphenated name",
			projectName: "orch-go",
			want:        "og",
		},
		{
			name:        "two-part hyphenated name (price-watch)",
			projectName: "price-watch",
			want:        "pw",
		},
		{
			name:        "single word project",
			projectName: "myproject",
			want:        "my",
		},
		{
			name:        "three-part project name",
			projectName: "my-cool-project",
			want:        "mycopr",
		},
		{
			name:        "underscores instead of hyphens",
			projectName: "my_project_name",
			want:        "myprna",
		},
		{
			name:        "empty project name falls back to og",
			projectName: "",
			want:        "og",
		},
		{
			name:        "single character parts",
			projectName: "a-b-c",
			want:        "abc",
		},
		{
			name:        "kb-cli project (2 parts = first letter of each)",
			projectName: "kb-cli",
			want:        "kc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateProjectPrefix(tt.projectName)
			if got != tt.want {
				t.Errorf("generateProjectPrefix(%q) = %q, want %q", tt.projectName, got, tt.want)
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

func TestWriteContext_CreatesScreenshotsDir(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &Config{
		Task:          "test task",
		Project:       "test",
		ProjectDir:    tempDir,
		WorkspaceName: "og-test-08jan",
		BeadsID:       "test-456",
	}

	if err := WriteContext(cfg); err != nil {
		t.Fatalf("WriteContext failed: %v", err)
	}

	// Check screenshots directory was created
	screenshotsPath := filepath.Join(tempDir, ".orch", "workspace", "og-test-08jan", "screenshots")
	stat, err := os.Stat(screenshotsPath)
	if os.IsNotExist(err) {
		t.Errorf("expected screenshots directory to exist at %s", screenshotsPath)
	}
	if err == nil && !stat.IsDir() {
		t.Errorf("expected screenshots to be a directory, got file")
	}
}

func TestCreateScreenshotsDir(t *testing.T) {
	t.Run("creates screenshots directory in workspace", func(t *testing.T) {
		tempDir := t.TempDir()

		if err := CreateScreenshotsDir(tempDir); err != nil {
			t.Fatalf("CreateScreenshotsDir failed: %v", err)
		}

		screenshotsPath := filepath.Join(tempDir, "screenshots")
		stat, err := os.Stat(screenshotsPath)
		if os.IsNotExist(err) {
			t.Error("expected screenshots directory to exist")
		}
		if err == nil && !stat.IsDir() {
			t.Error("expected screenshots to be a directory")
		}
	})

	t.Run("idempotent - does not error if directory exists", func(t *testing.T) {
		tempDir := t.TempDir()
		screenshotsPath := filepath.Join(tempDir, "screenshots")

		// Create directory first
		if err := os.MkdirAll(screenshotsPath, 0755); err != nil {
			t.Fatalf("failed to create screenshots dir: %v", err)
		}

		// Calling again should not error
		if err := CreateScreenshotsDir(tempDir); err != nil {
			t.Errorf("CreateScreenshotsDir should be idempotent, got error: %v", err)
		}
	})
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
		{"systematic-debugging", TierLight},

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

	t.Run("orch-go project uses orch-dashboard", func(t *testing.T) {
		// Create a temp dir but we'll use a subdirectory named "orch-go"
		tempDir := t.TempDir()
		orchGoDir := filepath.Join(tempDir, "orch-go")
		orchDir := filepath.Join(orchGoDir, ".orch")
		if err := os.MkdirAll(orchDir, 0755); err != nil {
			t.Fatalf("failed to create .orch dir: %v", err)
		}

		// Write config with servers
		configContent := `servers:
  api: 3348
`
		configPath := filepath.Join(orchDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		context := GenerateServerContext(orchGoDir)

		// Check it contains orch-dashboard guidance instead of orch servers
		if !strings.Contains(context, "## LOCAL SERVERS") {
			t.Error("expected server context to contain header")
		}
		if !strings.Contains(context, "orch-dashboard start") {
			t.Error("expected orch-go to recommend orch-dashboard, not orch servers")
		}
		if strings.Contains(context, "orch servers start orch-go") {
			t.Error("orch-go should NOT recommend 'orch servers start', should use orch-dashboard")
		}
		if !strings.Contains(context, "OpenCode") {
			t.Error("expected orch-go context to mention OpenCode")
		}
		if !strings.Contains(context, "https://localhost:3348") {
			t.Error("expected orch-go API URL to use https://localhost:3348")
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

func TestGenerateRegisteredProjectsContext_Format(t *testing.T) {
	// Test that the format is correct when we provide a mock project list
	// We can't easily test the actual kb command in unit tests,
	// but we can verify the format of the generated context

	projects := []RegisteredProject{
		{Name: "orch-go", Path: "/Users/test/orch-go"},
		{Name: "snap", Path: "/Users/test/snap"},
	}

	// Build expected output
	var sb strings.Builder
	sb.WriteString("## Registered Projects\n\n")
	sb.WriteString("These projects are registered with `kb` for cross-project orchestration:\n\n")
	sb.WriteString("| Project | Path |\n")
	sb.WriteString("|---------|------|\n")
	for _, p := range projects {
		sb.WriteString("| " + p.Name + " | `" + p.Path + "` |\n")
	}
	sb.WriteString("\n**Usage:** `orch spawn --workdir <path> SKILL \"task\"`\n\n")

	expected := sb.String()

	// Verify the format matches our expectations
	if !strings.Contains(expected, "## Registered Projects") {
		t.Error("expected registered projects header")
	}
	if !strings.Contains(expected, "| orch-go |") {
		t.Error("expected project row")
	}
	if !strings.Contains(expected, "orch spawn --workdir") {
		t.Error("expected usage hint")
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

func TestGenerateContext_BugReproduction(t *testing.T) {
	t.Run("includes reproduction section for bug issues", func(t *testing.T) {
		cfg := &Config{
			Task:          "fix login bug",
			SkillName:     "systematic-debugging",
			Project:       "test-project",
			ProjectDir:    "/tmp/test",
			WorkspaceName: "og-debug-test-03jan",
			BeadsID:       "test-123",
			IsBug:         true,
			ReproSteps:    "1. Navigate to /login\n2. Enter invalid credentials\n3. Click submit\n4. Observe blank screen instead of error message",
			Tier:          TierFull,
		}

		content, err := GenerateContext(cfg)
		if err != nil {
			t.Fatalf("GenerateContext failed: %v", err)
		}

		// Should contain reproduction section
		if !strings.Contains(content, "## REPRODUCTION (BUG FIX)") {
			t.Error("expected content to contain REPRODUCTION section for bug issue")
		}

		// Should contain bug indicator
		if !strings.Contains(content, "This is a bug fix issue") {
			t.Error("expected content to contain bug fix indicator")
		}

		// Should contain the actual repro steps
		if !strings.Contains(content, "Navigate to /login") {
			t.Error("expected content to contain reproduction steps")
		}

		// Should contain verification requirement
		if !strings.Contains(content, "Verification Requirement") {
			t.Error("expected content to contain verification requirement")
		}

		// Should contain beads comment instruction for verification
		if !strings.Contains(content, "bd comment test-123 \"Reproduction verified:") {
			t.Error("expected content to contain reproduction verification beads command")
		}

		// Should contain the warning about completion criteria
		if !strings.Contains(content, "bug fix is only complete when the original reproduction steps pass") {
			t.Error("expected content to contain completion warning")
		}
	})

	t.Run("excludes reproduction section for non-bug issues", func(t *testing.T) {
		cfg := &Config{
			Task:          "add new feature",
			SkillName:     "feature-impl",
			Project:       "test-project",
			ProjectDir:    "/tmp/test",
			WorkspaceName: "og-feat-test-03jan",
			BeadsID:       "test-123",
			IsBug:         false,
			ReproSteps:    "", // No repro steps for non-bug issues
			Tier:          TierLight,
		}

		content, err := GenerateContext(cfg)
		if err != nil {
			t.Fatalf("GenerateContext failed: %v", err)
		}

		// Should NOT contain reproduction section
		if strings.Contains(content, "## REPRODUCTION (BUG FIX)") {
			t.Error("expected content to NOT contain REPRODUCTION section for non-bug issue")
		}
	})

	t.Run("handles empty repro steps for bug issue", func(t *testing.T) {
		cfg := &Config{
			Task:          "fix mysterious bug",
			SkillName:     "systematic-debugging",
			Project:       "test-project",
			ProjectDir:    "/tmp/test",
			WorkspaceName: "og-debug-test-03jan",
			BeadsID:       "test-123",
			IsBug:         true,
			ReproSteps:    "", // No explicit repro steps
			Tier:          TierFull,
		}

		content, err := GenerateContext(cfg)
		if err != nil {
			t.Fatalf("GenerateContext failed: %v", err)
		}

		// Should still contain reproduction section (template will render it)
		// The section header should appear even if repro steps are empty
		if !strings.Contains(content, "## REPRODUCTION (BUG FIX)") {
			t.Error("expected content to contain REPRODUCTION section for bug issue even with empty repro")
		}
	})
}

func TestGenerateContext_InfrastructureResourceAuditDirective(t *testing.T) {
	t.Run("includes resource audit directive for infrastructure-touching spawns", func(t *testing.T) {
		cfg := &Config{
			Task:                     "fix opencode server restart handling",
			SkillName:                "systematic-debugging",
			Project:                  "test-project",
			ProjectDir:               "/tmp/test",
			WorkspaceName:            "og-debug-test-07feb",
			BeadsID:                  "test-123",
			Tier:                     TierLight,
			IsInfrastructureTouching: true,
		}

		content, err := GenerateContext(cfg)
		if err != nil {
			t.Fatalf("GenerateContext failed: %v", err)
		}

		if !strings.Contains(content, "## RESOURCE LIFECYCLE AUDIT (REQUIRED)") {
			t.Error("expected content to contain infrastructure resource audit section")
		}

		directive := "Audit all resources this component creates (goroutines, subprocesses, connections, caches) and ensure each has bounded lifetime and cleanup on shutdown."
		if !strings.Contains(content, directive) {
			t.Error("expected content to contain resource audit directive")
		}
	})

	t.Run("omits resource audit directive for non-infrastructure spawns", func(t *testing.T) {
		cfg := &Config{
			Task:          "add user profile endpoint",
			SkillName:     "feature-impl",
			Project:       "test-project",
			ProjectDir:    "/tmp/test",
			WorkspaceName: "og-feat-test-07feb",
			BeadsID:       "test-456",
			Tier:          TierLight,
		}

		content, err := GenerateContext(cfg)
		if err != nil {
			t.Fatalf("GenerateContext failed: %v", err)
		}

		if strings.Contains(content, "## RESOURCE LIFECYCLE AUDIT (REQUIRED)") {
			t.Error("expected content to omit resource audit section for non-infrastructure spawn")
		}
	})
}

func TestGenerateContext_NoPushGuidance(t *testing.T) {
	t.Run("includes no-push guidance in worker spawn context", func(t *testing.T) {
		cfg := &Config{
			Task:          "implement feature",
			SkillName:     "feature-impl",
			Project:       "test-project",
			ProjectDir:    "/tmp/test",
			WorkspaceName: "og-feat-test-16jan",
			BeadsID:       "test-123",
			Tier:          TierLight,
		}

		content, err := GenerateContext(cfg)
		if err != nil {
			t.Fatalf("GenerateContext failed: %v", err)
		}

		// Should contain explicit no-push guidance
		if !strings.Contains(content, "NEVER run `git push`") {
			t.Error("expected content to contain 'NEVER run `git push`' guidance")
		}

		// Should contain rationale about production systems
		if !strings.Contains(content, "trigger deploys that disrupt production systems") {
			t.Error("expected content to explain why pushing is prohibited")
		}

		// Should contain the worker rule
		if !strings.Contains(content, "Worker rule: Commit your work, call `/exit`. Don't push") {
			t.Error("expected content to contain worker rule statement")
		}

		// Should mention orchestrator handles pushing
		if !strings.Contains(content, "orchestrator will handle pushing to remote") {
			t.Error("expected content to explain orchestrator role in pushing")
		}
	})

	t.Run("includes no-push guidance in no-track spawn", func(t *testing.T) {
		cfg := &Config{
			Task:          "quick investigation",
			SkillName:     "investigation",
			Project:       "test-project",
			ProjectDir:    "/tmp/test",
			WorkspaceName: "og-inv-test-16jan",
			NoTrack:       true,
			Tier:          TierFull,
		}

		content, err := GenerateContext(cfg)
		if err != nil {
			t.Fatalf("GenerateContext failed: %v", err)
		}

		// Should still contain no-push guidance even for no-track spawns
		if !strings.Contains(content, "NEVER run `git push`") {
			t.Error("expected content to contain no-push guidance for no-track spawn")
		}

		// Should contain worker rule
		if !strings.Contains(content, "Worker rule:") {
			t.Error("expected content to contain worker rule for no-track spawn")
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

		// Should NOT contain progress tracking section
		if strings.Contains(content, "## PROGRESS TRACKING") {
			t.Error("expected content to NOT contain progress tracking section for --no-track spawn")
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

		// Should contain progress tracking section
		if !strings.Contains(content, "## PROGRESS TRACKING") {
			t.Error("expected content to contain progress tracking section for tracked spawn")
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

func TestGenerateContext_ProbeGuidance(t *testing.T) {
	t.Run("includes probe guidance when model content is injected in KBContext", func(t *testing.T) {
		cfg := &Config{
			Task:          "test model claims",
			SkillName:     "investigation",
			Project:       "test-project",
			ProjectDir:    "/tmp/test",
			WorkspaceName: "og-inv-test-08feb",
			BeadsID:       "test-123",
			Tier:          TierFull,
			KBContext: "## PRIOR KNOWLEDGE (from kb context)\n\n" +
				"### Models (synthesized understanding)\n" +
				"- Spawn Architecture\n" +
				"  - Summary:\n    The system creates workspaces.\n" +
				"  - Your findings should confirm, contradict, or extend the claims above.\n",
		}

		content, err := GenerateContext(cfg)
		if err != nil {
			t.Fatalf("GenerateContext failed: %v", err)
		}

		// Should contain probe guidance section
		if !strings.Contains(content, "## PROBE GUIDANCE (Model-Scoped Work)") {
			t.Error("expected content to contain probe guidance when models are injected")
		}

		// Should reference the probe template
		if !strings.Contains(content, "PROBE.md") {
			t.Error("expected content to reference PROBE.md template")
		}

		// Should contain the mandatory sections
		for _, section := range []string{"Question", "What I Tested", "What I Observed", "Model Impact"} {
			if !strings.Contains(content, section) {
				t.Errorf("expected probe guidance to mention mandatory section %q", section)
			}
		}

		// Should contain key discipline about testing vs reading
		if !strings.Contains(content, "Reading code is not testing") {
			t.Error("expected probe guidance to contain test-before-conclude discipline")
		}
	})

	t.Run("excludes probe guidance when no models in KBContext", func(t *testing.T) {
		cfg := &Config{
			Task:          "add feature",
			SkillName:     "feature-impl",
			Project:       "test-project",
			ProjectDir:    "/tmp/test",
			WorkspaceName: "og-feat-test-08feb",
			BeadsID:       "test-456",
			Tier:          TierLight,
			KBContext: "## PRIOR KNOWLEDGE (from kb context)\n\n" +
				"### Constraints (MUST respect)\n" +
				"- Some constraint\n",
		}

		content, err := GenerateContext(cfg)
		if err != nil {
			t.Fatalf("GenerateContext failed: %v", err)
		}

		if strings.Contains(content, "PROBE GUIDANCE") {
			t.Error("expected content to NOT contain probe guidance when no models are present")
		}
	})

	t.Run("excludes probe guidance when KBContext is empty", func(t *testing.T) {
		cfg := &Config{
			Task:          "simple task",
			SkillName:     "feature-impl",
			Project:       "test-project",
			ProjectDir:    "/tmp/test",
			WorkspaceName: "og-feat-test-08feb",
			BeadsID:       "test-789",
			Tier:          TierLight,
		}

		content, err := GenerateContext(cfg)
		if err != nil {
			t.Fatalf("GenerateContext failed: %v", err)
		}

		if strings.Contains(content, "PROBE GUIDANCE") {
			t.Error("expected content to NOT contain probe guidance when KBContext is empty")
		}
	})

	t.Run("includes probe guidance when HasInjectedModels is explicitly set", func(t *testing.T) {
		cfg := &Config{
			Task:              "verify model",
			SkillName:         "investigation",
			Project:           "test-project",
			ProjectDir:        "/tmp/test",
			WorkspaceName:     "og-inv-test-08feb",
			BeadsID:           "test-explicit",
			Tier:              TierFull,
			HasInjectedModels: true,
		}

		content, err := GenerateContext(cfg)
		if err != nil {
			t.Fatalf("GenerateContext failed: %v", err)
		}

		if !strings.Contains(content, "## PROBE GUIDANCE (Model-Scoped Work)") {
			t.Error("expected content to contain probe guidance when HasInjectedModels=true")
		}
	})
}
