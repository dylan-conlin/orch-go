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
		"No SYNTHESIS.md required",
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
		"produce a SYNTHESIS.md",
		"/exit",           // Meta-orchestrators don't exit
		"Phase: Complete", // Worker-level tracking
		"bd comment",      // Worker-level tracking
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
		IsOrchestrator:     true, // Both are true
		IsMetaOrchestrator: true, // Meta should take priority
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
		"No SYNTHESIS.md required",
	}

	for _, marker := range interactiveMarkers {
		if !strings.Contains(metaContent, marker) {
			t.Errorf("meta-orchestrator should have interactive marker %q", marker)
		}
	}

	// Orchestrator should have task-completion framing
	orchTaskMarkers := []string{
		"Session Goal:",
		
	}

	for _, marker := range orchTaskMarkers {
		if !strings.Contains(orchContent, marker) {
			t.Errorf("orchestrator should have task marker %q", marker)
		}
	}

	// Orchestrator requires SYNTHESIS.md as a deliverable
	if !strings.Contains(orchContent, "Create SYNTHESIS.md") {
		t.Error("orchestrator should require creating SYNTHESIS.md")
	}

	// Meta-orchestrator should NOT have task-completion framing
	// Note: Meta-orchestrator mentions "No SYNTHESIS.md required" which is fine
	// The key is it shouldn't have "Session Goal:" or 
	metaForbiddenMarkers := []string{
		"Session Goal:",
		
		"Create SYNTHESIS.md", // This is the task-completion requirement
	}

	for _, marker := range metaForbiddenMarkers {
		if strings.Contains(metaContent, marker) {
			t.Errorf("meta-orchestrator should NOT have task marker %q", marker)
		}
	}
}

func TestFindPriorMetaOrchestratorHandoff(t *testing.T) {
	// Create a temp project directory with some meta-orchestrator workspaces
	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, ".orch", "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace dir: %v", err)
	}

	// Create a meta-orchestrator workspace WITHOUT SYNTHESIS.md (incomplete)
	incompleteWs := filepath.Join(workspaceDir, "meta-orch-incomplete-05jan")
	if err := os.MkdirAll(incompleteWs, 0755); err != nil {
		t.Fatalf("failed to create incomplete workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(incompleteWs, ".meta-orchestrator"), []byte("meta"), 0644); err != nil {
		t.Fatalf("failed to write meta marker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(incompleteWs, ".spawn_time"), []byte("1704470400000000000"), 0644); err != nil { // Earlier timestamp
		t.Fatalf("failed to write spawn time: %v", err)
	}

	// Create a meta-orchestrator workspace WITH SYNTHESIS.md (complete)
	completeWs := filepath.Join(workspaceDir, "meta-orch-complete-05jan")
	if err := os.MkdirAll(completeWs, 0755); err != nil {
		t.Fatalf("failed to create complete workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(completeWs, ".meta-orchestrator"), []byte("meta"), 0644); err != nil {
		t.Fatalf("failed to write meta marker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(completeWs, ".spawn_time"), []byte("1704556800000000000"), 0644); err != nil { // Later timestamp
		t.Fatalf("failed to write spawn time: %v", err)
	}
	synthesisContent := "# Session Handoff\n\nTest handoff content."
	if err := os.WriteFile(filepath.Join(completeWs, "SYNTHESIS.md"), []byte(synthesisContent), 0644); err != nil {
		t.Fatalf("failed to write handoff: %v", err)
	}

	// Create a regular orchestrator workspace with handoff (should be ignored)
	regularOrchWs := filepath.Join(workspaceDir, "og-orch-regular-05jan")
	if err := os.MkdirAll(regularOrchWs, 0755); err != nil {
		t.Fatalf("failed to create regular orch workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(regularOrchWs, ".orchestrator"), []byte("orch"), 0644); err != nil {
		t.Fatalf("failed to write orch marker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(regularOrchWs, "SYNTHESIS.md"), []byte("# Regular orch handoff"), 0644); err != nil {
		t.Fatalf("failed to write orch handoff: %v", err)
	}

	// Test: Find should return the complete meta-orchestrator workspace
	result := FindPriorMetaOrchestratorHandoff(tempDir)
	expected := filepath.Join(completeWs, "SYNTHESIS.md")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestFindPriorMetaOrchestratorHandoff_Empty(t *testing.T) {
	// Create a temp project directory with NO meta-orchestrator workspaces
	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, ".orch", "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace dir: %v", err)
	}

	// Test: Find should return empty string when no meta-orchestrator handoffs exist
	result := FindPriorMetaOrchestratorHandoff(tempDir)
	if result != "" {
		t.Errorf("expected empty string, got %s", result)
	}
}

func TestFindPriorMetaOrchestratorHandoff_ExcludesCurrent(t *testing.T) {
	// Test that the current workspace is excluded from search
	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, ".orch", "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace dir: %v", err)
	}

	// Create a single meta-orchestrator workspace with handoff
	currentWs := filepath.Join(workspaceDir, "meta-orch-current-06jan")
	if err := os.MkdirAll(currentWs, 0755); err != nil {
		t.Fatalf("failed to create current workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(currentWs, ".meta-orchestrator"), []byte("meta"), 0644); err != nil {
		t.Fatalf("failed to write meta marker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(currentWs, "SYNTHESIS.md"), []byte("# Current synthesis"), 0644); err != nil {
		t.Fatalf("failed to write handoff: %v", err)
	}

	// Test: When excluding the current workspace, should find nothing
	result := findPriorMetaOrchestratorHandoffExcluding(tempDir, "meta-orch-current-06jan")
	if result != "" {
		t.Errorf("expected empty string when excluding current workspace, got %s", result)
	}

	// Test: Without exclusion, should find it
	resultWithoutExclusion := findPriorMetaOrchestratorHandoffExcluding(tempDir, "")
	expected := filepath.Join(currentWs, "SYNTHESIS.md")
	if resultWithoutExclusion != expected {
		t.Errorf("expected %s, got %s", expected, resultWithoutExclusion)
	}
}

func TestFindPriorMetaOrchestratorHandoff_SearchesArchive(t *testing.T) {
	// Test that archived workspaces are also searched
	tempDir := t.TempDir()

	// Create workspace-archive with a meta-orchestrator handoff
	archiveDir := filepath.Join(tempDir, ".orch", "workspace-archive")
	archivedWs := filepath.Join(archiveDir, "meta-orch-archived-04jan")
	if err := os.MkdirAll(archivedWs, 0755); err != nil {
		t.Fatalf("failed to create archived workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(archivedWs, ".meta-orchestrator"), []byte("meta"), 0644); err != nil {
		t.Fatalf("failed to write meta marker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(archivedWs, ".spawn_time"), []byte("1704384000000000000"), 0644); err != nil {
		t.Fatalf("failed to write spawn time: %v", err)
	}
	if err := os.WriteFile(filepath.Join(archivedWs, "SYNTHESIS.md"), []byte("# Archived handoff"), 0644); err != nil {
		t.Fatalf("failed to write handoff: %v", err)
	}

	// Test: Should find the archived handoff
	result := FindPriorMetaOrchestratorHandoff(tempDir)
	expected := filepath.Join(archivedWs, "SYNTHESIS.md")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestFindPriorMetaOrchestratorHandoff_MostRecent(t *testing.T) {
	// Test that the most recent handoff is returned when multiple exist
	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, ".orch", "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace dir: %v", err)
	}

	// Create older workspace
	olderWs := filepath.Join(workspaceDir, "meta-orch-older-03jan")
	if err := os.MkdirAll(olderWs, 0755); err != nil {
		t.Fatalf("failed to create older workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(olderWs, ".meta-orchestrator"), []byte("meta"), 0644); err != nil {
		t.Fatalf("failed to write meta marker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(olderWs, ".spawn_time"), []byte("1704297600000000000"), 0644); err != nil { // 2024-01-03
		t.Fatalf("failed to write spawn time: %v", err)
	}
	if err := os.WriteFile(filepath.Join(olderWs, "SYNTHESIS.md"), []byte("# Older handoff"), 0644); err != nil {
		t.Fatalf("failed to write handoff: %v", err)
	}

	// Create newer workspace
	newerWs := filepath.Join(workspaceDir, "meta-orch-newer-05jan")
	if err := os.MkdirAll(newerWs, 0755); err != nil {
		t.Fatalf("failed to create newer workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(newerWs, ".meta-orchestrator"), []byte("meta"), 0644); err != nil {
		t.Fatalf("failed to write meta marker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(newerWs, ".spawn_time"), []byte("1704470400000000000"), 0644); err != nil { // 2024-01-05
		t.Fatalf("failed to write spawn time: %v", err)
	}
	if err := os.WriteFile(filepath.Join(newerWs, "SYNTHESIS.md"), []byte("# Newer handoff"), 0644); err != nil {
		t.Fatalf("failed to write handoff: %v", err)
	}

	// Test: Should return the newer workspace
	result := FindPriorMetaOrchestratorHandoff(tempDir)
	expected := filepath.Join(newerWs, "SYNTHESIS.md")
	if result != expected {
		t.Errorf("expected %s (newer), got %s", expected, result)
	}
}

func TestGenerateMetaOrchestratorContext_WithPriorSynthesis(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &Config{
		SkillName:          "meta-orchestrator",
		ProjectDir:         tempDir,
		WorkspaceName:      "meta-orch-new-06jan",
		IsOrchestrator:     true,
		IsMetaOrchestrator: true,
		PriorSynthesisPath:   "/path/to/prior/SYNTHESIS.md",
	}

	content, err := GenerateMetaOrchestratorContext(cfg)
	if err != nil {
		t.Fatalf("GenerateMetaOrchestratorContext failed: %v", err)
	}

	// Should contain prior handoff section
	if !strings.Contains(content, "## Prior Session Context") {
		t.Error("expected content to contain prior session context section")
	}
	if !strings.Contains(content, "/path/to/prior/SYNTHESIS.md") {
		t.Error("expected content to contain prior handoff path")
	}
	if !strings.Contains(content, "Read the prior SYNTHESIS.md") {
		t.Error("expected content to contain instruction to read prior handoff")
	}
}

func TestGenerateMetaOrchestratorContext_NoPriorSynthesis(t *testing.T) {
	cfg := &Config{
		SkillName:          "meta-orchestrator",
		ProjectDir:         "/tmp/test",
		WorkspaceName:      "meta-orch-new-06jan",
		IsOrchestrator:     true,
		IsMetaOrchestrator: true,
		// No PriorSynthesisPath set
	}

	content, err := GenerateMetaOrchestratorContext(cfg)
	if err != nil {
		t.Fatalf("GenerateMetaOrchestratorContext failed: %v", err)
	}

	// Should NOT contain prior handoff section
	if strings.Contains(content, "## Prior Session Context") {
		t.Error("expected content to NOT contain prior session context section when no prior handoff")
	}
}

func TestGenerateMetaOrchestratorContext_AutoFindsPriorSynthesis(t *testing.T) {
	// Test that GenerateMetaOrchestratorContext automatically finds prior handoff
	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, ".orch", "workspace")

	// Create a prior meta-orchestrator workspace with handoff
	priorWs := filepath.Join(workspaceDir, "meta-orch-prior-05jan")
	if err := os.MkdirAll(priorWs, 0755); err != nil {
		t.Fatalf("failed to create prior workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(priorWs, ".meta-orchestrator"), []byte("meta"), 0644); err != nil {
		t.Fatalf("failed to write meta marker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(priorWs, ".spawn_time"), []byte("1704470400000000000"), 0644); err != nil {
		t.Fatalf("failed to write spawn time: %v", err)
	}
	synthesisContent := "# Prior Session Handoff\n\nContext from prior session."
	if err := os.WriteFile(filepath.Join(priorWs, "SYNTHESIS.md"), []byte(synthesisContent), 0644); err != nil {
		t.Fatalf("failed to write handoff: %v", err)
	}

	cfg := &Config{
		SkillName:          "meta-orchestrator",
		ProjectDir:         tempDir,
		WorkspaceName:      "meta-orch-new-06jan", // Different from prior
		IsOrchestrator:     true,
		IsMetaOrchestrator: true,
		// PriorSynthesisPath is NOT set - should be auto-discovered
	}

	content, err := GenerateMetaOrchestratorContext(cfg)
	if err != nil {
		t.Fatalf("GenerateMetaOrchestratorContext failed: %v", err)
	}

	// Should contain prior handoff section with auto-discovered path
	if !strings.Contains(content, "## Prior Session Context") {
		t.Error("expected content to contain prior session context section (auto-discovered)")
	}
	expectedPath := filepath.Join(priorWs, "SYNTHESIS.md")
	if !strings.Contains(content, expectedPath) {
		t.Errorf("expected content to contain auto-discovered prior handoff path %s", expectedPath)
	}
}
