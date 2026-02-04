package spawn

import (
	"os"
	"os/exec"
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

		"SYNTHESIS.md",
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
		templatePath := filepath.Join(templatesDir, "SYNTHESIS.md")
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

		// Check SYNTHESIS.template.md was copied to workspace
		copiedTemplatePath := filepath.Join(workspacePath, "SYNTHESIS.template.md")
		if _, err := os.Stat(copiedTemplatePath); os.IsNotExist(err) {
			t.Errorf("expected SYNTHESIS.template.md to exist in workspace at %s", copiedTemplatePath)
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

		// SYNTHESIS.template.md should NOT exist
		copiedTemplatePath := filepath.Join(workspacePath, "SYNTHESIS.template.md")
		if _, err := os.Stat(copiedTemplatePath); !os.IsNotExist(err) {
			t.Error("SYNTHESIS.template.md should NOT exist when source template is missing")
		}
	})
}

func TestGenerateOrchestratorContext_MentionsTemplateWhenCopied(t *testing.T) {
	t.Run("context mentions template when it was copied", func(t *testing.T) {
		cfg := &Config{
			Task:                 "orchestrate work",
			SessionGoal:          "Complete the feature epic",
			SkillName:            "orchestrator",
			ProjectDir:           "/tmp/test",
			WorkspaceName:        "og-orch-test-04jan",
			IsOrchestrator:       true,
			HasSynthesisTemplate: true, // This flag should be set by WriteOrchestratorContext
		}

		content, err := GenerateOrchestratorContext(cfg)
		if err != nil {
			t.Fatalf("GenerateOrchestratorContext failed: %v", err)
		}

		// Should mention the template file
		if !strings.Contains(content, "SYNTHESIS.template.md") {
			t.Error("expected context to mention SYNTHESIS.template.md when HasSynthesisTemplate is true")
		}
	})

	t.Run("context does not mention template when not copied", func(t *testing.T) {
		cfg := &Config{
			Task:                 "orchestrate work",
			SessionGoal:          "Complete the feature epic",
			SkillName:            "orchestrator",
			ProjectDir:           "/tmp/test",
			WorkspaceName:        "og-orch-test-04jan",
			IsOrchestrator:       true,
			HasSynthesisTemplate: false,
		}

		content, err := GenerateOrchestratorContext(cfg)
		if err != nil {
			t.Fatalf("GenerateOrchestratorContext failed: %v", err)
		}

		// Should NOT mention template.md file (the .template.md version)
		if strings.Contains(content, "SYNTHESIS.template.md") {
			t.Error("context should NOT mention SYNTHESIS.template.md when HasSynthesisTemplate is false")
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

	// Check SYNTHESIS.md was pre-created
	handoffPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	if _, err := os.Stat(handoffPath); os.IsNotExist(err) {
		t.Error("expected SYNTHESIS.md to be pre-created")
	}

	// Check content has metadata filled in
	content, err := os.ReadFile(handoffPath)
	if err != nil {
		t.Fatalf("failed to read synthesis: %v", err)
	}

	contentStr := string(content)

	// Check workspace name is filled in
	if !strings.Contains(contentStr, "og-orch-test-05jan") {
		t.Error("SYNTHESIS.md should contain workspace name")
	}

	// Check session goal is filled in
	if !strings.Contains(contentStr, "Complete the feature epic") {
		t.Error("SYNTHESIS.md should contain session goal")
	}

	// Check it has the progressive documentation comment
	if !strings.Contains(contentStr, "Progressive Documentation") {
		t.Error("SYNTHESIS.md should contain progressive documentation guidance")
	}

	// Check it prompts for TLDR to be filled
	if !strings.Contains(contentStr, "Fill within first 5 tool calls") {
		t.Error("SYNTHESIS.md should prompt for early section fills")
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
	if !strings.Contains(content, "Fill SYNTHESIS.md sections") {
		t.Error("context should mention filling SYNTHESIS.md sections")
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

func TestGenerateGitLogContext(t *testing.T) {
	t.Run("generates git log context in git repository", func(t *testing.T) {
		// Use the current project directory (orch-go) which is a git repo
		wd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}

		// Navigate up to project root if we're in pkg/spawn
		projectDir := wd
		if strings.HasSuffix(wd, "pkg/spawn") {
			projectDir = filepath.Join(wd, "../..")
		}

		gitLogContext := GenerateGitLogContext(projectDir)

		// Should return non-empty string for a git repository
		if gitLogContext == "" {
			t.Skip("skipping test - not in a git repository or no commits in last 7 days")
		}

		// Should contain git log output (hash + message + time)
		// Example: "7b7b91c investigation: SPAWN_CONTEXT generation issues (2 hours ago)"
		if !strings.Contains(gitLogContext, "(") || !strings.Contains(gitLogContext, "ago)") {
			t.Errorf("git log context should contain relative time, got: %s", gitLogContext)
		}
	})

	t.Run("highlights beads IDs when present", func(t *testing.T) {
		// Use the current project directory
		wd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}

		projectDir := wd
		if strings.HasSuffix(wd, "pkg/spawn") {
			projectDir = filepath.Join(wd, "../..")
		}

		gitLogContext := GenerateGitLogContext(projectDir)

		if gitLogContext == "" {
			t.Skip("skipping test - not in a git repository or no commits in last 7 days")
		}

		// If any commits contain beads IDs (like orch-go-21074), they should be bolded
		// We can't guarantee they exist, so we'll just check the format is reasonable
		lines := strings.Split(gitLogContext, "\n")
		if len(lines) == 0 {
			t.Error("git log context should contain at least one line")
		}

		// Each line should match the expected format: hash + message + (time ago)
		for _, line := range lines {
			if line == "" {
				continue
			}
			// Lines should contain relative time format
			if !strings.Contains(line, "ago)") {
				t.Errorf("each line should contain relative time, got: %s", line)
			}
		}
	})

	t.Run("returns empty string for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		gitLogContext := GenerateGitLogContext(tempDir)

		// Should return empty string for non-git directory
		if gitLogContext != "" {
			t.Errorf("expected empty string for non-git directory, got: %s", gitLogContext)
		}
	})

	t.Run("returns message when no recent commits", func(t *testing.T) {
		// Create a temporary git repo with no recent commits
		tempDir := t.TempDir()

		// Initialize git repo
		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Skip("git not available for testing")
		}

		// Don't create any commits - git log should return empty
		gitLogContext := GenerateGitLogContext(tempDir)

		// Should return a message about no recent commits
		if gitLogContext != "No recent commits in the last 7 days." {
			t.Errorf("expected 'No recent commits' message, got: %s", gitLogContext)
		}
	})
}

func TestGenerateOrchestratorContext_WithGitLogContext(t *testing.T) {
	// Use a temporary directory that's a git repo
	tempDir := t.TempDir()

	// Initialize git repo and create a commit
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Skip("git not available for testing")
	}

	// Configure git user for the test
	exec.Command("git", "config", "user.email", "test@example.com").Dir = tempDir
	exec.Command("git", "config", "user.name", "Test User").Dir = tempDir

	// Create a commit with a beads ID
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	exec.Command("git", "add", "test.txt").Dir = tempDir
	commitCmd := exec.Command("git", "commit", "-m", "fix: test commit for orch-go-12345")
	commitCmd.Dir = tempDir
	if err := commitCmd.Run(); err != nil {
		t.Skip("failed to create test commit")
	}

	cfg := &Config{
		Task:           "Ship feature",
		SessionGoal:    "Ship feature end-to-end",
		SkillName:      "orchestrator",
		ProjectDir:     tempDir,
		WorkspaceName:  "og-orch-test-30jan",
		IsOrchestrator: true,
	}

	content, err := GenerateOrchestratorContext(cfg)
	if err != nil {
		t.Fatalf("GenerateOrchestratorContext failed: %v", err)
	}

	// Should contain git log context section
	if !strings.Contains(content, "## Recent Activity") {
		t.Error("expected content to contain Recent Activity section")
	}
	if !strings.Contains(content, "Recent commits in this project") {
		t.Error("expected content to contain git log context header")
	}

	// Should contain the test commit message
	if !strings.Contains(content, "test commit") {
		t.Error("expected content to contain commit message")
	}

	// Should highlight the beads ID
	if !strings.Contains(content, "**orch-go-12345**") {
		t.Error("expected content to highlight beads ID in bold")
	}
}
