package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateOrchestratorContext(t *testing.T) {
	cfg := &Config{
		Task:           "Ship the metrics feature",
		SessionGoal:    "Ship the metrics feature end-to-end",
		SkillName:      "orchestrator",
		Project:        "orch-go",
		ProjectDir:     "/Users/test/orch-go",
		WorkspaceName:  "og-orch-metrics-04jan",
		IsOrchestrator: true,
		SkillContent: `---
name: orchestrator
skill-type: policy
---

# Orchestrator Skill

Test skill content.
`,
	}

	content, err := GenerateOrchestratorContext(cfg)
	if err != nil {
		t.Fatalf("GenerateOrchestratorContext failed: %v", err)
	}

	// Check key sections are present
	checks := []string{
		"# Orchestrator Session Context",
		"**Session Goal:** Ship the metrics feature end-to-end",
		"**Skill:** orchestrator",
		"**Project:** /Users/test/orch-go",
		"spawned orchestrator",
		"orch session end",
		"SESSION_HANDOFF.md",
		"## Skill Guidance",
		"# Orchestrator Skill",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("expected content to contain %q", check)
		}
	}

	// Should NOT contain worker-specific instructions as requirements
	// (Note: /exit may appear in the "Do NOT use /exit" warning, which is expected)
	workerPatterns := []string{
		"Phase: Complete",
		"bd comment",
		"SYNTHESIS.md",
	}

	for _, pattern := range workerPatterns {
		if strings.Contains(content, pattern) {
			t.Errorf("orchestrator context should NOT contain worker pattern %q", pattern)
		}
	}

	// /exit should only appear in warnings (explaining what NOT to use), not as a completion instruction
	// The template has two mentions: one in session type explanation, one in completion protocol
	// Both are explaining that /exit is for workers, not orchestrators
	if strings.Contains(content, "Run: `/exit`") {
		t.Error("orchestrator context should NOT tell users to run /exit as a completion step")
	}
}

func TestGenerateOrchestratorContext_UsesTaskAsSessionGoal(t *testing.T) {
	cfg := &Config{
		Task:           "Ship the metrics feature",
		SessionGoal:    "", // Empty - should use Task
		SkillName:      "orchestrator",
		ProjectDir:     "/tmp/test",
		WorkspaceName:  "og-orch-test-04jan",
		IsOrchestrator: true,
	}

	content, err := GenerateOrchestratorContext(cfg)
	if err != nil {
		t.Fatalf("GenerateOrchestratorContext failed: %v", err)
	}

	// Should use Task as session goal when SessionGoal is empty
	if !strings.Contains(content, "**Session Goal:** Ship the metrics feature") {
		t.Error("expected content to use Task as SessionGoal when SessionGoal is empty")
	}
}

func TestWriteOrchestratorContext(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &Config{
		Task:           "orchestrate work",
		SessionGoal:    "Complete the feature epic",
		SkillName:      "orchestrator",
		ProjectDir:     tempDir,
		WorkspaceName:  "og-orch-test-04jan",
		IsOrchestrator: true,
	}

	if err := WriteOrchestratorContext(cfg); err != nil {
		t.Fatalf("WriteOrchestratorContext failed: %v", err)
	}

	workspacePath := filepath.Join(tempDir, ".orch", "workspace", "og-orch-test-04jan")

	// Check ORCHESTRATOR_CONTEXT.md was created
	contextPath := filepath.Join(workspacePath, "ORCHESTRATOR_CONTEXT.md")
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		t.Errorf("expected orchestrator context file to exist at %s", contextPath)
	}

	// Check content
	content, err := os.ReadFile(contextPath)
	if err != nil {
		t.Fatalf("failed to read context file: %v", err)
	}

	if !strings.Contains(string(content), "# Orchestrator Session Context") {
		t.Error("context file should contain orchestrator header")
	}

	// Check .orchestrator marker was created
	markerPath := filepath.Join(workspacePath, ".orchestrator")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Error("expected .orchestrator marker file to exist")
	}

	// Check .spawn_time was created
	spawnTimePath := filepath.Join(workspacePath, ".spawn_time")
	if _, err := os.Stat(spawnTimePath); os.IsNotExist(err) {
		t.Error("expected .spawn_time file to exist")
	}

	// Check screenshots directory was created
	screenshotsPath := filepath.Join(workspacePath, "screenshots")
	stat, err := os.Stat(screenshotsPath)
	if os.IsNotExist(err) {
		t.Error("expected screenshots directory to exist")
	}
	if err == nil && !stat.IsDir() {
		t.Error("expected screenshots to be a directory")
	}
}

func TestWriteContext_RoutesToOrchestrator(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &Config{
		Task:           "orchestrate work",
		SessionGoal:    "Complete the feature epic",
		SkillName:      "orchestrator",
		ProjectDir:     tempDir,
		WorkspaceName:  "og-orch-test-04jan",
		IsOrchestrator: true, // This should route to orchestrator template
	}

	// Use the main WriteContext function - it should route to orchestrator
	if err := WriteContext(cfg); err != nil {
		t.Fatalf("WriteContext failed: %v", err)
	}

	workspacePath := filepath.Join(tempDir, ".orch", "workspace", "og-orch-test-04jan")

	// Check ORCHESTRATOR_CONTEXT.md was created (not SPAWN_CONTEXT.md)
	orchestratorPath := filepath.Join(workspacePath, "ORCHESTRATOR_CONTEXT.md")
	if _, err := os.Stat(orchestratorPath); os.IsNotExist(err) {
		t.Error("expected ORCHESTRATOR_CONTEXT.md to exist when IsOrchestrator=true")
	}

	// Check SPAWN_CONTEXT.md was NOT created
	spawnPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	if _, err := os.Stat(spawnPath); !os.IsNotExist(err) {
		t.Error("expected SPAWN_CONTEXT.md to NOT exist when IsOrchestrator=true")
	}

	// Check .orchestrator marker exists
	markerPath := filepath.Join(workspacePath, ".orchestrator")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Error("expected .orchestrator marker to exist")
	}
}

func TestWriteContext_WorkerDoesNotRouteToOrchestrator(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &Config{
		Task:           "implement feature",
		SkillName:      "feature-impl",
		ProjectDir:     tempDir,
		WorkspaceName:  "og-feat-test-04jan",
		BeadsID:        "test-123",
		IsOrchestrator: false, // Worker spawn
		Tier:           TierLight,
	}

	if err := WriteContext(cfg); err != nil {
		t.Fatalf("WriteContext failed: %v", err)
	}

	workspacePath := filepath.Join(tempDir, ".orch", "workspace", "og-feat-test-04jan")

	// Check SPAWN_CONTEXT.md was created (not ORCHESTRATOR_CONTEXT.md)
	spawnPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	if _, err := os.Stat(spawnPath); os.IsNotExist(err) {
		t.Error("expected SPAWN_CONTEXT.md to exist when IsOrchestrator=false")
	}

	// Check ORCHESTRATOR_CONTEXT.md was NOT created
	orchestratorPath := filepath.Join(workspacePath, "ORCHESTRATOR_CONTEXT.md")
	if _, err := os.Stat(orchestratorPath); !os.IsNotExist(err) {
		t.Error("expected ORCHESTRATOR_CONTEXT.md to NOT exist when IsOrchestrator=false")
	}

	// Check .orchestrator marker does NOT exist
	markerPath := filepath.Join(workspacePath, ".orchestrator")
	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Error("expected .orchestrator marker to NOT exist for worker spawn")
	}
}

func TestMinimalPrompt_RoutesToOrchestrator(t *testing.T) {
	t.Run("orchestrator spawn points to ORCHESTRATOR_CONTEXT.md", func(t *testing.T) {
		cfg := &Config{
			ProjectDir:     "/Users/test/orch-go",
			WorkspaceName:  "og-orch-metrics-04jan",
			IsOrchestrator: true,
		}

		prompt := MinimalPrompt(cfg)

		if !strings.Contains(prompt, "ORCHESTRATOR_CONTEXT.md") {
			t.Errorf("orchestrator minimal prompt should contain ORCHESTRATOR_CONTEXT.md, got: %s", prompt)
		}
		if strings.Contains(prompt, "SPAWN_CONTEXT.md") {
			t.Errorf("orchestrator minimal prompt should NOT contain SPAWN_CONTEXT.md, got: %s", prompt)
		}
		if !strings.Contains(prompt, "session goal") {
			t.Errorf("orchestrator minimal prompt should mention session goal, got: %s", prompt)
		}
	})

	t.Run("worker spawn points to SPAWN_CONTEXT.md", func(t *testing.T) {
		cfg := &Config{
			ProjectDir:     "/Users/test/orch-go",
			WorkspaceName:  "og-feat-spawn-04jan",
			IsOrchestrator: false,
		}

		prompt := MinimalPrompt(cfg)

		if !strings.Contains(prompt, "SPAWN_CONTEXT.md") {
			t.Errorf("worker minimal prompt should contain SPAWN_CONTEXT.md, got: %s", prompt)
		}
		if strings.Contains(prompt, "ORCHESTRATOR_CONTEXT.md") {
			t.Errorf("worker minimal prompt should NOT contain ORCHESTRATOR_CONTEXT.md, got: %s", prompt)
		}
	})
}

func TestMinimalOrchestratorPrompt(t *testing.T) {
	cfg := &Config{
		ProjectDir:    "/Users/test/orch-go",
		WorkspaceName: "og-orch-metrics-04jan",
	}

	prompt := MinimalOrchestratorPrompt(cfg)

	if !strings.Contains(prompt, "/Users/test/orch-go/.orch/workspace/og-orch-metrics-04jan/ORCHESTRATOR_CONTEXT.md") {
		t.Errorf("orchestrator prompt should contain full path to ORCHESTRATOR_CONTEXT.md, got: %s", prompt)
	}
	if !strings.Contains(prompt, "session goal") {
		t.Errorf("orchestrator prompt should mention session goal, got: %s", prompt)
	}
}

func TestEnsureSessionHandoffTemplate(t *testing.T) {
	t.Run("creates template when missing", func(t *testing.T) {
		tempDir := t.TempDir()

		// Ensure template doesn't exist initially
		templatePath := filepath.Join(tempDir, ".orch", "templates", "SESSION_HANDOFF.md")
		if _, err := os.Stat(templatePath); !os.IsNotExist(err) {
			t.Fatal("template should not exist initially")
		}

		// Call EnsureSessionHandoffTemplate
		if err := EnsureSessionHandoffTemplate(tempDir); err != nil {
			t.Fatalf("EnsureSessionHandoffTemplate failed: %v", err)
		}

		// Check template was created
		if _, err := os.Stat(templatePath); os.IsNotExist(err) {
			t.Error("template should exist after EnsureSessionHandoffTemplate")
		}

		// Check content
		content, err := os.ReadFile(templatePath)
		if err != nil {
			t.Fatalf("failed to read template: %v", err)
		}

		if !strings.Contains(string(content), "# Session Handoff") {
			t.Error("template should contain session handoff header")
		}
		if !strings.Contains(string(content), "## Summary") {
			t.Error("template should contain Summary section")
		}
		if !strings.Contains(string(content), "## Work Completed") {
			t.Error("template should contain Work Completed section")
		}
		if !strings.Contains(string(content), "## Recommendations for Next Session") {
			t.Error("template should contain Recommendations section")
		}
	})

	t.Run("does not overwrite existing template", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create templates directory and custom template
		templatesDir := filepath.Join(tempDir, ".orch", "templates")
		if err := os.MkdirAll(templatesDir, 0755); err != nil {
			t.Fatalf("failed to create templates dir: %v", err)
		}

		customContent := "# Custom Session Handoff Template\n\nThis is a custom template."
		templatePath := filepath.Join(templatesDir, "SESSION_HANDOFF.md")
		if err := os.WriteFile(templatePath, []byte(customContent), 0644); err != nil {
			t.Fatalf("failed to write custom template: %v", err)
		}

		// Call EnsureSessionHandoffTemplate
		if err := EnsureSessionHandoffTemplate(tempDir); err != nil {
			t.Fatalf("EnsureSessionHandoffTemplate failed: %v", err)
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

func TestGenerateOrchestratorContext_WithKBContext(t *testing.T) {
	cfg := &Config{
		Task:           "Ship feature",
		SessionGoal:    "Ship feature end-to-end",
		SkillName:      "orchestrator",
		ProjectDir:     "/tmp/test",
		WorkspaceName:  "og-orch-test-04jan",
		IsOrchestrator: true,
		KBContext: `## PRIOR KNOWLEDGE

**Query:** "feature"

### Constraints (MUST respect)
- All tests must pass before shipping

### Prior Decisions
- Use the new API design
`,
	}

	content, err := GenerateOrchestratorContext(cfg)
	if err != nil {
		t.Fatalf("GenerateOrchestratorContext failed: %v", err)
	}

	// Should contain KB context
	if !strings.Contains(content, "## PRIOR KNOWLEDGE") {
		t.Error("expected content to contain KB context")
	}
	if !strings.Contains(content, "All tests must pass before shipping") {
		t.Error("expected content to contain constraint from KB context")
	}
}

func TestGenerateOrchestratorContext_WithServerContext(t *testing.T) {
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
		Task:           "Ship feature",
		SessionGoal:    "Ship feature end-to-end",
		SkillName:      "orchestrator",
		ProjectDir:     tempDir,
		WorkspaceName:  "og-orch-test-04jan",
		IsOrchestrator: true,
		IncludeServers: true,
	}

	content, err := GenerateOrchestratorContext(cfg)
	if err != nil {
		t.Fatalf("GenerateOrchestratorContext failed: %v", err)
	}

	// Should contain server context
	if !strings.Contains(content, "## LOCAL SERVERS") {
		t.Error("expected content to contain server context")
	}
	if !strings.Contains(content, "http://localhost:5173") {
		t.Error("expected content to contain web port")
	}
}

func TestGenerateOrchestratorContext_WithRegisteredProjects(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &Config{
		Task:           "Ship cross-project feature",
		SessionGoal:    "Coordinate work across projects",
		SkillName:      "orchestrator",
		ProjectDir:     tempDir,
		WorkspaceName:  "og-orch-test-06jan",
		IsOrchestrator: true,
		RegisteredProjects: `## Registered Projects

These projects are registered with ` + "`kb`" + ` for cross-project orchestration:

| Project | Path |
|---------|------|
| orch-go | ` + "`/Users/test/orch-go`" + ` |
| snap | ` + "`/Users/test/snap`" + ` |

**Usage:** ` + "`orch spawn --workdir <path> SKILL \"task\"`" + `

`,
	}

	content, err := GenerateOrchestratorContext(cfg)
	if err != nil {
		t.Fatalf("GenerateOrchestratorContext failed: %v", err)
	}

	// Should contain registered projects context
	if !strings.Contains(content, "## Registered Projects") {
		t.Error("expected content to contain registered projects section")
	}
	if !strings.Contains(content, "orch-go") {
		t.Error("expected content to contain project name")
	}
	if !strings.Contains(content, "orch spawn --workdir") {
		t.Error("expected content to contain usage hint")
	}
}

func TestWriteOrchestratorContext_CopiesSessionHandoffTemplate(t *testing.T) {
	t.Run("copies template when it exists", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create the template in .orch/templates/
		templatesDir := filepath.Join(tempDir, ".orch", "templates")
		if err := os.MkdirAll(templatesDir, 0755); err != nil {
			t.Fatalf("failed to create templates dir: %v", err)
		}

		customTemplate := `# Custom Session Handoff

**Focus:** {session-goal}
**Outcome:** {success | partial | blocked}

## TLDR
[Summary here]

## Custom Section
This is a project-specific template.
`
		templatePath := filepath.Join(templatesDir, "SESSION_HANDOFF.md")
		if err := os.WriteFile(templatePath, []byte(customTemplate), 0644); err != nil {
			t.Fatalf("failed to write template: %v", err)
		}

		cfg := &Config{
			Task:           "orchestrate work",
			SessionGoal:    "Complete the feature epic",
			SkillName:      "orchestrator",
			ProjectDir:     tempDir,
			WorkspaceName:  "og-orch-test-04jan",
			IsOrchestrator: true,
		}

		if err := WriteOrchestratorContext(cfg); err != nil {
			t.Fatalf("WriteOrchestratorContext failed: %v", err)
		}

		workspacePath := filepath.Join(tempDir, ".orch", "workspace", "og-orch-test-04jan")

		// Check SESSION_HANDOFF.template.md was copied to workspace
		copiedTemplatePath := filepath.Join(workspacePath, "SESSION_HANDOFF.template.md")
		if _, err := os.Stat(copiedTemplatePath); os.IsNotExist(err) {
			t.Errorf("expected SESSION_HANDOFF.template.md to exist in workspace at %s", copiedTemplatePath)
		}

		// Check content matches the source template
		copiedContent, err := os.ReadFile(copiedTemplatePath)
		if err != nil {
			t.Fatalf("failed to read copied template: %v", err)
		}

		if string(copiedContent) != customTemplate {
			t.Error("copied template content does not match source template")
		}
	})

	t.Run("spawn works when template does not exist", func(t *testing.T) {
		tempDir := t.TempDir()

		// No template file exists
		cfg := &Config{
			Task:           "orchestrate work",
			SessionGoal:    "Complete the feature epic",
			SkillName:      "orchestrator",
			ProjectDir:     tempDir,
			WorkspaceName:  "og-orch-notemplate-04jan",
			IsOrchestrator: true,
		}

		// Should NOT fail when template doesn't exist
		if err := WriteOrchestratorContext(cfg); err != nil {
			t.Fatalf("WriteOrchestratorContext should not fail when template is missing: %v", err)
		}

		workspacePath := filepath.Join(tempDir, ".orch", "workspace", "og-orch-notemplate-04jan")

		// Check ORCHESTRATOR_CONTEXT.md was still created
		contextPath := filepath.Join(workspacePath, "ORCHESTRATOR_CONTEXT.md")
		if _, err := os.Stat(contextPath); os.IsNotExist(err) {
			t.Error("expected ORCHESTRATOR_CONTEXT.md to exist even without template")
		}

		// SESSION_HANDOFF.template.md should NOT exist
		copiedTemplatePath := filepath.Join(workspacePath, "SESSION_HANDOFF.template.md")
		if _, err := os.Stat(copiedTemplatePath); !os.IsNotExist(err) {
			t.Error("SESSION_HANDOFF.template.md should NOT exist when source template is missing")
		}
	})
}

func TestGenerateOrchestratorContext_MentionsTemplateWhenCopied(t *testing.T) {
	t.Run("context mentions template when it was copied", func(t *testing.T) {
		cfg := &Config{
			Task:                      "orchestrate work",
			SessionGoal:               "Complete the feature epic",
			SkillName:                 "orchestrator",
			ProjectDir:                "/tmp/test",
			WorkspaceName:             "og-orch-test-04jan",
			IsOrchestrator:            true,
			HasSessionHandoffTemplate: true, // This flag should be set by WriteOrchestratorContext
		}

		content, err := GenerateOrchestratorContext(cfg)
		if err != nil {
			t.Fatalf("GenerateOrchestratorContext failed: %v", err)
		}

		// Should mention the template file
		if !strings.Contains(content, "SESSION_HANDOFF.template.md") {
			t.Error("expected context to mention SESSION_HANDOFF.template.md when HasSessionHandoffTemplate is true")
		}
	})

	t.Run("context does not mention template when not copied", func(t *testing.T) {
		cfg := &Config{
			Task:                      "orchestrate work",
			SessionGoal:               "Complete the feature epic",
			SkillName:                 "orchestrator",
			ProjectDir:                "/tmp/test",
			WorkspaceName:             "og-orch-test-04jan",
			IsOrchestrator:            true,
			HasSessionHandoffTemplate: false,
		}

		content, err := GenerateOrchestratorContext(cfg)
		if err != nil {
			t.Fatalf("GenerateOrchestratorContext failed: %v", err)
		}

		// Should NOT mention template.md file (the .template.md version)
		if strings.Contains(content, "SESSION_HANDOFF.template.md") {
			t.Error("context should NOT mention SESSION_HANDOFF.template.md when HasSessionHandoffTemplate is false")
		}
	})
}

func TestEnsureSessionHandoffTemplate_PrefersProjectTemplate(t *testing.T) {
	t.Run("uses content from project template when it exists", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create project template with custom content
		templatesDir := filepath.Join(tempDir, ".orch", "templates")
		if err := os.MkdirAll(templatesDir, 0755); err != nil {
			t.Fatalf("failed to create templates dir: %v", err)
		}

		customContent := `# Custom Project Handoff Template

This is a project-specific template with custom sections.

## Special Section
Only this project has this section.
`
		templatePath := filepath.Join(templatesDir, "SESSION_HANDOFF.md")
		if err := os.WriteFile(templatePath, []byte(customContent), 0644); err != nil {
			t.Fatalf("failed to write template: %v", err)
		}

		// EnsureSessionHandoffTemplate should NOT overwrite
		if err := EnsureSessionHandoffTemplate(tempDir); err != nil {
			t.Fatalf("EnsureSessionHandoffTemplate failed: %v", err)
		}

		// Verify custom content is preserved
		content, err := os.ReadFile(templatePath)
		if err != nil {
			t.Fatalf("failed to read template: %v", err)
		}

		if string(content) != customContent {
			t.Error("EnsureSessionHandoffTemplate should preserve existing project template")
		}
	})
}

func TestWriteOrchestratorContext_PreCreatesSessionHandoff(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &Config{
		Task:           "orchestrate work",
		SessionGoal:    "Complete the feature epic",
		SkillName:      "orchestrator",
		ProjectDir:     tempDir,
		WorkspaceName:  "og-orch-test-05jan",
		IsOrchestrator: true,
	}

	if err := WriteOrchestratorContext(cfg); err != nil {
		t.Fatalf("WriteOrchestratorContext failed: %v", err)
	}

	workspacePath := filepath.Join(tempDir, ".orch", "workspace", "og-orch-test-05jan")

	// Check SESSION_HANDOFF.md was pre-created
	handoffPath := filepath.Join(workspacePath, "SESSION_HANDOFF.md")
	if _, err := os.Stat(handoffPath); os.IsNotExist(err) {
		t.Error("expected SESSION_HANDOFF.md to be pre-created")
	}

	// Check content has metadata filled in
	content, err := os.ReadFile(handoffPath)
	if err != nil {
		t.Fatalf("failed to read session handoff: %v", err)
	}

	contentStr := string(content)

	// Check workspace name is filled in
	if !strings.Contains(contentStr, "og-orch-test-05jan") {
		t.Error("SESSION_HANDOFF.md should contain workspace name")
	}

	// Check session goal is filled in
	if !strings.Contains(contentStr, "Complete the feature epic") {
		t.Error("SESSION_HANDOFF.md should contain session goal")
	}

	// Check it has the progressive documentation comment
	if !strings.Contains(contentStr, "Progressive Documentation") {
		t.Error("SESSION_HANDOFF.md should contain progressive documentation guidance")
	}

	// Check it prompts for TLDR to be filled
	if !strings.Contains(contentStr, "Fill within first 5 tool calls") {
		t.Error("SESSION_HANDOFF.md should prompt for early section fills")
	}
}

func TestGeneratePreFilledSessionHandoff(t *testing.T) {
	content, err := GeneratePreFilledSessionHandoff("og-test-workspace", "Test session goal", "2026-01-05 15:00")
	if err != nil {
		t.Fatalf("GeneratePreFilledSessionHandoff failed: %v", err)
	}

	// Check all metadata is present
	if !strings.Contains(content, "og-test-workspace") {
		t.Error("content should contain workspace name")
	}
	if !strings.Contains(content, "Test session goal") {
		t.Error("content should contain session goal")
	}
	if !strings.Contains(content, "2026-01-05 15:00") {
		t.Error("content should contain start time")
	}

	// Check key sections exist
	sections := []string{
		"## TLDR",
		"## Spawns (Agents Managed)",
		"## Evidence (What Was Observed)",
		"## Knowledge (What Was Learned)",
		"## Friction (What Was Harder Than It Should Be)",
		"## Focus Progress",
		"### Where We Started",
		"## Next (What Should Happen)",
		"## Session Metadata",
	}

	for _, section := range sections {
		if !strings.Contains(content, section) {
			t.Errorf("content should contain section: %s", section)
		}
	}
}

func TestOrchestratorContext_HasProgressiveHandoffInstruction(t *testing.T) {
	cfg := &Config{
		Task:           "orchestrate work",
		SessionGoal:    "Complete the feature epic",
		SkillName:      "orchestrator",
		ProjectDir:     "/tmp/test",
		WorkspaceName:  "og-orch-test-05jan",
		IsOrchestrator: true,
	}

	content, err := GenerateOrchestratorContext(cfg)
	if err != nil {
		t.Fatalf("GenerateOrchestratorContext failed: %v", err)
	}

	// Check the instruction mentions filling TLDR and Where We Started
	if !strings.Contains(content, "Fill SESSION_HANDOFF.md sections") {
		t.Error("context should mention filling SESSION_HANDOFF.md sections")
	}

	if !strings.Contains(content, "TLDR") {
		t.Error("context should mention filling TLDR section")
	}

	if !strings.Contains(content, "Where We Started") {
		t.Error("context should mention filling Where We Started section")
	}

	// Check it mentions progressive handoff
	if !strings.Contains(content, "Progressive Handoff") {
		t.Error("context should mention progressive handoff pattern")
	}
}
