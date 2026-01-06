package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateMetaOrchestratorContext(t *testing.T) {
	cfg := &Config{
		SkillName:          "meta-orchestrator",
		Project:            "orch-go",
		ProjectDir:         "/Users/test/orch-go",
		WorkspaceName:      "og-work-meta-orch-04jan",
		IsOrchestrator:     true,
		IsMetaOrchestrator: true,
		SkillContent: `---
name: meta-orchestrator
skill-type: policy
---

# Meta-Orchestrator Skill

Test skill content for meta-orchestrator.
`,
	}

	content, err := GenerateMetaOrchestratorContext(cfg)
	if err != nil {
		t.Fatalf("GenerateMetaOrchestratorContext failed: %v", err)
	}

	// Check key sections are present - these distinguish meta-orchestrator from orchestrator
	checks := []string{
		"# Meta-Orchestrator Session Context",
		"**Role:** You are managing orchestrator sessions",
		"**Skill:** meta-orchestrator",
		"**Project:** /Users/test/orch-go",
		"meta-orchestrator",
		"managing orchestrator sessions",
		"orch status",
		"orch review",
		"## Skill Guidance",
		"# Meta-Orchestrator Skill",
		// Interactive framing - should NOT have task-completion language
		"Stay available",
		"No SESSION_HANDOFF.md required",
		// Level distinction
		"one level above orchestrators",
		"spawn orchestrators",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("expected content to contain %q", check)
		}
	}

	// Should NOT contain orchestrator task-completion framing
	forbiddenPatterns := []string{
		"Session Goal:", // Meta-orchestrators don't have a goal to accomplish
		"work toward",   // Task-completion language
		"When you've accomplished",
		"produce a SESSION_HANDOFF.md",
		"/exit",        // Meta-orchestrators don't exit
		"Phase: Complete", // Worker-level tracking
		"bd comment",   // Worker-level tracking
	}

	for _, pattern := range forbiddenPatterns {
		if strings.Contains(content, pattern) {
			t.Errorf("meta-orchestrator context should NOT contain %q", pattern)
		}
	}
}

func TestWriteMetaOrchestratorContext(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &Config{
		SkillName:          "meta-orchestrator",
		ProjectDir:         tempDir,
		WorkspaceName:      "og-work-meta-orch-04jan",
		IsOrchestrator:     true,
		IsMetaOrchestrator: true,
	}

	if err := WriteMetaOrchestratorContext(cfg); err != nil {
		t.Fatalf("WriteMetaOrchestratorContext failed: %v", err)
	}

	workspacePath := filepath.Join(tempDir, ".orch", "workspace", "og-work-meta-orch-04jan")

	// Check META_ORCHESTRATOR_CONTEXT.md was created
	contextPath := filepath.Join(workspacePath, "META_ORCHESTRATOR_CONTEXT.md")
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		t.Errorf("expected meta-orchestrator context file to exist at %s", contextPath)
	}

	// Check content
	content, err := os.ReadFile(contextPath)
	if err != nil {
		t.Fatalf("failed to read context file: %v", err)
	}

	if !strings.Contains(string(content), "# Meta-Orchestrator Session Context") {
		t.Error("context file should contain meta-orchestrator header")
	}

	// Check .meta-orchestrator marker was created
	markerPath := filepath.Join(workspacePath, ".meta-orchestrator")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Error("expected .meta-orchestrator marker file to exist")
	}

	// Check .spawn_time was created
	spawnTimePath := filepath.Join(workspacePath, ".spawn_time")
	if _, err := os.Stat(spawnTimePath); os.IsNotExist(err) {
		t.Error("expected .spawn_time file to exist")
	}
}

func TestWriteContext_RoutesToMetaOrchestrator(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &Config{
		SkillName:          "meta-orchestrator",
		ProjectDir:         tempDir,
		WorkspaceName:      "og-work-meta-orch-04jan",
		IsOrchestrator:     true,
		IsMetaOrchestrator: true, // This should route to meta-orchestrator template
	}

	// Use the main WriteContext function - it should route to meta-orchestrator
	if err := WriteContext(cfg); err != nil {
		t.Fatalf("WriteContext failed: %v", err)
	}

	workspacePath := filepath.Join(tempDir, ".orch", "workspace", "og-work-meta-orch-04jan")

	// Check META_ORCHESTRATOR_CONTEXT.md was created (not ORCHESTRATOR_CONTEXT.md or SPAWN_CONTEXT.md)
	metaOrchestratorPath := filepath.Join(workspacePath, "META_ORCHESTRATOR_CONTEXT.md")
	if _, err := os.Stat(metaOrchestratorPath); os.IsNotExist(err) {
		t.Error("expected META_ORCHESTRATOR_CONTEXT.md to exist when IsMetaOrchestrator=true")
	}

	// Check ORCHESTRATOR_CONTEXT.md was NOT created
	orchestratorPath := filepath.Join(workspacePath, "ORCHESTRATOR_CONTEXT.md")
	if _, err := os.Stat(orchestratorPath); !os.IsNotExist(err) {
		t.Error("expected ORCHESTRATOR_CONTEXT.md to NOT exist when IsMetaOrchestrator=true")
	}

	// Check SPAWN_CONTEXT.md was NOT created
	spawnPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	if _, err := os.Stat(spawnPath); !os.IsNotExist(err) {
		t.Error("expected SPAWN_CONTEXT.md to NOT exist when IsMetaOrchestrator=true")
	}

	// Check .meta-orchestrator marker exists
	markerPath := filepath.Join(workspacePath, ".meta-orchestrator")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Error("expected .meta-orchestrator marker to exist")
	}
}

func TestMinimalPrompt_RoutesToMetaOrchestrator(t *testing.T) {
	t.Run("meta-orchestrator spawn points to META_ORCHESTRATOR_CONTEXT.md", func(t *testing.T) {
		cfg := &Config{
			ProjectDir:         "/Users/test/orch-go",
			WorkspaceName:      "og-work-meta-orch-04jan",
			IsOrchestrator:     true,
			IsMetaOrchestrator: true,
		}

		prompt := MinimalPrompt(cfg)

		if !strings.Contains(prompt, "META_ORCHESTRATOR_CONTEXT.md") {
			t.Errorf("meta-orchestrator minimal prompt should contain META_ORCHESTRATOR_CONTEXT.md, got: %s", prompt)
		}
		// Check it's not pointing to the non-meta ORCHESTRATOR_CONTEXT.md
		// Note: "ORCHESTRATOR_CONTEXT.md" is a substring of "META_ORCHESTRATOR_CONTEXT.md"
		// so we check it's specifically pointing to META_
		if strings.Contains(prompt, "ORCHESTRATOR_CONTEXT.md") && !strings.Contains(prompt, "META_ORCHESTRATOR_CONTEXT.md") {
			t.Errorf("meta-orchestrator minimal prompt should NOT point to ORCHESTRATOR_CONTEXT.md (should point to META_), got: %s", prompt)
		}
		if strings.Contains(prompt, "SPAWN_CONTEXT.md") {
			t.Errorf("meta-orchestrator minimal prompt should NOT contain SPAWN_CONTEXT.md, got: %s", prompt)
		}
		if !strings.Contains(prompt, "managing orchestrator sessions") {
			t.Errorf("meta-orchestrator minimal prompt should mention managing orchestrator sessions, got: %s", prompt)
		}
	})

	t.Run("orchestrator spawn points to ORCHESTRATOR_CONTEXT.md (not meta)", func(t *testing.T) {
		cfg := &Config{
			ProjectDir:         "/Users/test/orch-go",
			WorkspaceName:      "og-orch-metrics-04jan",
			IsOrchestrator:     true,
			IsMetaOrchestrator: false, // Regular orchestrator, not meta
		}

		prompt := MinimalPrompt(cfg)

		if !strings.Contains(prompt, "ORCHESTRATOR_CONTEXT.md") {
			t.Errorf("orchestrator minimal prompt should contain ORCHESTRATOR_CONTEXT.md, got: %s", prompt)
		}
		if strings.Contains(prompt, "META_ORCHESTRATOR_CONTEXT.md") {
			t.Errorf("orchestrator minimal prompt should NOT contain META_ORCHESTRATOR_CONTEXT.md, got: %s", prompt)
		}
	})
}

func TestMinimalMetaOrchestratorPrompt(t *testing.T) {
	cfg := &Config{
		ProjectDir:    "/Users/test/orch-go",
		WorkspaceName: "og-work-meta-orch-04jan",
	}

	prompt := MinimalMetaOrchestratorPrompt(cfg)

	if !strings.Contains(prompt, "/Users/test/orch-go/.orch/workspace/og-work-meta-orch-04jan/META_ORCHESTRATOR_CONTEXT.md") {
		t.Errorf("meta-orchestrator prompt should contain full path to META_ORCHESTRATOR_CONTEXT.md, got: %s", prompt)
	}
	if !strings.Contains(prompt, "managing orchestrator sessions") {
		t.Errorf("meta-orchestrator prompt should mention managing orchestrator sessions, got: %s", prompt)
	}
}

func TestGenerateMetaOrchestratorContext_WithKBContext(t *testing.T) {
	cfg := &Config{
		SkillName:          "meta-orchestrator",
		ProjectDir:         "/tmp/test",
		WorkspaceName:      "og-work-meta-orch-04jan",
		IsOrchestrator:     true,
		IsMetaOrchestrator: true,
		KBContext: `## PRIOR KNOWLEDGE

**Query:** "orchestration"

### Constraints (MUST respect)
- Max 3 orchestrator sessions at once

### Prior Decisions
- Use focus-based session model
`,
	}

	content, err := GenerateMetaOrchestratorContext(cfg)
	if err != nil {
		t.Fatalf("GenerateMetaOrchestratorContext failed: %v", err)
	}

	// Should contain KB context
	if !strings.Contains(content, "## PRIOR KNOWLEDGE") {
		t.Error("expected content to contain KB context")
	}
	if !strings.Contains(content, "Max 3 orchestrator sessions at once") {
		t.Error("expected content to contain constraint from KB context")
	}
}

func TestGenerateMetaOrchestratorContext_WithServerContext(t *testing.T) {
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
		SkillName:          "meta-orchestrator",
		ProjectDir:         tempDir,
		WorkspaceName:      "og-work-meta-orch-04jan",
		IsOrchestrator:     true,
		IsMetaOrchestrator: true,
		IncludeServers:     true,
	}

	content, err := GenerateMetaOrchestratorContext(cfg)
	if err != nil {
		t.Fatalf("GenerateMetaOrchestratorContext failed: %v", err)
	}

	// Should contain server context
	if !strings.Contains(content, "## LOCAL SERVERS") {
		t.Error("expected content to contain server context")
	}
	if !strings.Contains(content, "http://localhost:5173") {
		t.Error("expected content to contain web port")
	}
}

func TestGenerateMetaOrchestratorContext_WithRegisteredProjects(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &Config{
		SkillName:          "meta-orchestrator",
		ProjectDir:         tempDir,
		WorkspaceName:      "meta-orch-test-06jan",
		IsOrchestrator:     true,
		IsMetaOrchestrator: true,
		RegisteredProjects: `## Registered Projects

These projects are registered with ` + "`kb`" + ` for cross-project orchestration:

| Project | Path |
|---------|------|
| orch-go | ` + "`/Users/test/orch-go`" + ` |
| snap | ` + "`/Users/test/snap`" + ` |
| price-watch | ` + "`/Users/test/price-watch`" + ` |

**Usage:** ` + "`orch spawn --workdir <path> SKILL \"task\"`" + `

`,
	}

	content, err := GenerateMetaOrchestratorContext(cfg)
	if err != nil {
		t.Fatalf("GenerateMetaOrchestratorContext failed: %v", err)
	}

	// Should contain registered projects context
	if !strings.Contains(content, "## Registered Projects") {
		t.Error("expected content to contain registered projects section")
	}
	if !strings.Contains(content, "price-watch") {
		t.Error("expected content to contain project name")
	}
	if !strings.Contains(content, "orch spawn --workdir") {
		t.Error("expected content to contain usage hint")
	}
}

func TestWriteContext_PriorityOrder(t *testing.T) {
	// Test that IsMetaOrchestrator takes priority over IsOrchestrator
	// Both can be true at the same time, and meta-orchestrator should win
	tempDir := t.TempDir()

	cfg := &Config{
		SkillName:          "meta-orchestrator",
		ProjectDir:         tempDir,
		WorkspaceName:      "og-work-priority-test-04jan",
		IsOrchestrator:     true,  // Both are true
		IsMetaOrchestrator: true,  // Meta should take priority
	}

	if err := WriteContext(cfg); err != nil {
		t.Fatalf("WriteContext failed: %v", err)
	}

	workspacePath := filepath.Join(tempDir, ".orch", "workspace", "og-work-priority-test-04jan")

	// Only META_ORCHESTRATOR_CONTEXT.md should exist
	metaPath := filepath.Join(workspacePath, "META_ORCHESTRATOR_CONTEXT.md")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Error("expected META_ORCHESTRATOR_CONTEXT.md to exist (meta takes priority)")
	}

	orchPath := filepath.Join(workspacePath, "ORCHESTRATOR_CONTEXT.md")
	if _, err := os.Stat(orchPath); !os.IsNotExist(err) {
		t.Error("expected ORCHESTRATOR_CONTEXT.md to NOT exist (meta takes priority)")
	}

	spawnPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	if _, err := os.Stat(spawnPath); !os.IsNotExist(err) {
		t.Error("expected SPAWN_CONTEXT.md to NOT exist (meta takes priority)")
	}
}

func TestGenerateMetaOrchestratorContext_InteractiveFraming(t *testing.T) {
	// Verify the key framing differences between meta-orchestrator and orchestrator
	metaCfg := &Config{
		SkillName:          "meta-orchestrator",
		ProjectDir:         "/tmp/test",
		WorkspaceName:      "og-work-meta-04jan",
		IsOrchestrator:     true,
		IsMetaOrchestrator: true,
	}

	metaContent, err := GenerateMetaOrchestratorContext(metaCfg)
	if err != nil {
		t.Fatalf("GenerateMetaOrchestratorContext failed: %v", err)
	}

	orchCfg := &Config{
		SessionGoal:    "Ship the feature",
		SkillName:      "orchestrator",
		ProjectDir:     "/tmp/test",
		WorkspaceName:  "og-orch-test-04jan",
		IsOrchestrator: true,
	}

	orchContent, err := GenerateOrchestratorContext(orchCfg)
	if err != nil {
		t.Fatalf("GenerateOrchestratorContext failed: %v", err)
	}

	// Meta-orchestrator should have interactive framing
	interactiveMarkers := []string{
		"Stay available",
		"stay interactive",
		"Ask Dylan",
		"No SESSION_HANDOFF.md required",
	}

	for _, marker := range interactiveMarkers {
		if !strings.Contains(metaContent, marker) {
			t.Errorf("meta-orchestrator should have interactive marker %q", marker)
		}
	}

	// Orchestrator should have task-completion framing
	orchTaskMarkers := []string{
		"Session Goal:",
		"orch session end",
	}

	for _, marker := range orchTaskMarkers {
		if !strings.Contains(orchContent, marker) {
			t.Errorf("orchestrator should have task marker %q", marker)
		}
	}

	// Orchestrator requires SESSION_HANDOFF.md as a deliverable
	if !strings.Contains(orchContent, "Create SESSION_HANDOFF.md") {
		t.Error("orchestrator should require creating SESSION_HANDOFF.md")
	}

	// Meta-orchestrator should NOT have task-completion framing
	// Note: Meta-orchestrator mentions "No SESSION_HANDOFF.md required" which is fine
	// The key is it shouldn't have "Session Goal:" or "orch session end"
	metaForbiddenMarkers := []string{
		"Session Goal:",
		"orch session end",
		"Create SESSION_HANDOFF.md", // This is the task-completion requirement
	}

	for _, marker := range metaForbiddenMarkers {
		if strings.Contains(metaContent, marker) {
			t.Errorf("meta-orchestrator should NOT have task marker %q", marker)
		}
	}
}
