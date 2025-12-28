package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitProject(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "orch-init-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test basic initialization
	// Args: projectDir, force, skipBeads, skipKB, skipKN, skipClaudeMD, skipTmuxinator, beadsPrefix, projectType
	t.Run("creates all directories", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test1")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		result, err := initProject(testDir, false, true, true, true, true, true, "", "") // skip beads, kb, kn, claudemd, tmuxinator
		if err != nil {
			t.Fatalf("initProject failed: %v", err)
		}

		// Check that directories were created (.orch only, .kb handled by kb init)
		expectedDirs := []string{
			".orch/workspace",
			".orch/templates",
		}

		for _, dir := range expectedDirs {
			fullPath := filepath.Join(testDir, dir)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				t.Errorf("expected directory %s to exist", dir)
			}
		}

		// Check that directories were marked as created (2 .orch dirs)
		if len(result.DirsCreated) != 2 {
			t.Errorf("expected 2 directories created, got %d", len(result.DirsCreated))
		}
	})

	t.Run("idempotent - second run reports existing", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test2")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		// First init
		_, err := initProject(testDir, false, true, true, true, true, true, "", "")
		if err != nil {
			t.Fatalf("first initProject failed: %v", err)
		}

		// Second init
		result, err := initProject(testDir, false, true, true, true, true, true, "", "")
		if err != nil {
			t.Fatalf("second initProject failed: %v", err)
		}

		// All directories should exist now
		if len(result.DirsCreated) != 0 {
			t.Errorf("expected 0 directories created on second run, got %d", len(result.DirsCreated))
		}
		if len(result.DirsExisted) != 2 {
			t.Errorf("expected 2 directories already existed, got %d", len(result.DirsExisted))
		}
	})

	t.Run("force recreates directories", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test3")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		// First init
		_, err := initProject(testDir, false, true, true, true, true, true, "", "")
		if err != nil {
			t.Fatalf("first initProject failed: %v", err)
		}

		// Second init with force
		result, err := initProject(testDir, true, true, true, true, true, true, "", "")
		if err != nil {
			t.Fatalf("force initProject failed: %v", err)
		}

		// With force, all directories should be marked as created
		if len(result.DirsCreated) != 2 {
			t.Errorf("expected 2 directories created with force, got %d", len(result.DirsCreated))
		}
	})

	t.Run("skip beads sets flag", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test4")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		result, err := initProject(testDir, false, true, true, true, true, true, "", "")
		if err != nil {
			t.Fatalf("initProject failed: %v", err)
		}

		if !result.BeadsSkipped {
			t.Error("expected BeadsSkipped to be true")
		}
	})

	t.Run("skip kb sets flag", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test4b")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		result, err := initProject(testDir, false, true, true, true, true, true, "", "")
		if err != nil {
			t.Fatalf("initProject failed: %v", err)
		}

		if !result.KBSkipped {
			t.Error("expected KBSkipped to be true")
		}
	})

	t.Run("skip kn sets flag", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test4c")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		result, err := initProject(testDir, false, true, true, true, true, true, "", "")
		if err != nil {
			t.Fatalf("initProject failed: %v", err)
		}

		if !result.KNSkipped {
			t.Error("expected KNSkipped to be true")
		}
	})

	t.Run("synthesis template is written", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test5")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		_, err := initProject(testDir, false, true, true, true, true, true, "", "")
		if err != nil {
			t.Fatalf("initProject failed: %v", err)
		}

		// Check that SYNTHESIS.md template exists
		synthPath := filepath.Join(testDir, ".orch", "templates", "SYNTHESIS.md")
		if _, err := os.Stat(synthPath); os.IsNotExist(err) {
			t.Error("expected SYNTHESIS.md template to exist")
		}
	})

	t.Run("CLAUDE.md is generated with auto-detection", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test6")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		// Create go.mod and cmd/ to trigger go-cli detection
		if err := os.WriteFile(filepath.Join(testDir, "go.mod"), []byte("module test"), 0644); err != nil {
			t.Fatalf("failed to create go.mod: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(testDir, "cmd"), 0755); err != nil {
			t.Fatalf("failed to create cmd dir: %v", err)
		}

		result, err := initProject(testDir, false, true, true, true, false, true, "", "")
		if err != nil {
			t.Fatalf("initProject failed: %v", err)
		}

		if !result.ClaudeMDCreated {
			t.Error("expected ClaudeMDCreated to be true")
		}

		if result.ProjectType != "go-cli" {
			t.Errorf("expected ProjectType go-cli, got %s", result.ProjectType)
		}

		// Check that CLAUDE.md exists
		claudePath := filepath.Join(testDir, "CLAUDE.md")
		if _, err := os.Stat(claudePath); os.IsNotExist(err) {
			t.Error("expected CLAUDE.md to exist")
		}

		// Check content contains project name
		content, _ := os.ReadFile(claudePath)
		if !containsSubstring(string(content), "test6") {
			t.Error("expected project name in CLAUDE.md")
		}
	})

	t.Run("skip CLAUDE.md sets flag", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test7")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		result, err := initProject(testDir, false, true, true, true, true, true, "", "")
		if err != nil {
			t.Fatalf("initProject failed: %v", err)
		}

		if !result.ClaudeMDSkipped {
			t.Error("expected ClaudeMDSkipped to be true")
		}

		// Check that CLAUDE.md does NOT exist
		claudePath := filepath.Join(testDir, "CLAUDE.md")
		if _, err := os.Stat(claudePath); err == nil {
			t.Error("expected CLAUDE.md to NOT exist when skipped")
		}
	})

	t.Run("CLAUDE.md with explicit type", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test8")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		result, err := initProject(testDir, false, true, true, true, false, true, "", "svelte-app")
		if err != nil {
			t.Fatalf("initProject failed: %v", err)
		}

		if result.ProjectType != "svelte-app" {
			t.Errorf("expected ProjectType svelte-app, got %s", result.ProjectType)
		}

		// Check content contains svelte-specific content
		claudePath := filepath.Join(testDir, "CLAUDE.md")
		content, _ := os.ReadFile(claudePath)
		if !containsSubstring(string(content), "bun") {
			t.Error("expected svelte-app template content in CLAUDE.md")
		}
	})

	t.Run("tmuxinator config is generated", func(t *testing.T) {
		// Use a unique name based on timestamp to avoid conflicts with previous test runs
		testDir := filepath.Join(tmpDir, "tmux-test")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		result, err := initProject(testDir, false, true, true, true, true, false, "", "")
		if err != nil {
			t.Fatalf("initProject failed: %v", err)
		}

		// Accept either created or updated - both are valid outcomes
		if !result.TmuxinatorCreated && !result.TmuxinatorUpdated {
			t.Error("expected TmuxinatorCreated or TmuxinatorUpdated to be true")
		}

		if result.TmuxinatorPath == "" {
			t.Error("expected TmuxinatorPath to be set")
		}

		// Check that tmuxinator config file exists
		if _, err := os.Stat(result.TmuxinatorPath); os.IsNotExist(err) {
			t.Errorf("expected tmuxinator config at %s to exist", result.TmuxinatorPath)
		}
	})

	t.Run("skip tmuxinator sets flag", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test10")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		result, err := initProject(testDir, false, true, true, true, true, true, "", "")
		if err != nil {
			t.Fatalf("initProject failed: %v", err)
		}

		if !result.TmuxinatorSkipped {
			t.Error("expected TmuxinatorSkipped to be true")
		}
	})
}

func TestEnsureDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orch-ensuredir-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("creates non-existent directory", func(t *testing.T) {
		path := filepath.Join(tmpDir, "new-dir")
		created, err := ensureDir(path, false)
		if err != nil {
			t.Fatalf("ensureDir failed: %v", err)
		}
		if !created {
			t.Error("expected created to be true")
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Error("directory should exist")
		}
	})

	t.Run("returns false for existing directory", func(t *testing.T) {
		path := filepath.Join(tmpDir, "existing-dir")
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		created, err := ensureDir(path, false)
		if err != nil {
			t.Fatalf("ensureDir failed: %v", err)
		}
		if created {
			t.Error("expected created to be false for existing directory")
		}
	})

	t.Run("force returns true for existing directory", func(t *testing.T) {
		path := filepath.Join(tmpDir, "force-dir")
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		created, err := ensureDir(path, true)
		if err != nil {
			t.Fatalf("ensureDir failed: %v", err)
		}
		if !created {
			t.Error("expected created to be true with force flag")
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		path := filepath.Join(tmpDir, "a", "b", "c", "d")
		created, err := ensureDir(path, false)
		if err != nil {
			t.Fatalf("ensureDir failed: %v", err)
		}
		if !created {
			t.Error("expected created to be true")
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Error("nested directory should exist")
		}
	})
}

func TestWriteSynthesisTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orch-synth-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	path := filepath.Join(tmpDir, "SYNTHESIS.md")
	if err := writeSynthesisTemplate(path); err != nil {
		t.Fatalf("writeSynthesisTemplate failed: %v", err)
	}

	// Check file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("SYNTHESIS.md should exist")
	}

	// Check content has expected sections
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	expectedSections := []string{
		"# Synthesis",
		"## Summary",
		"## Key Deliverables",
		"## Changes Made",
		"## Discoveries",
		"## Status",
	}

	for _, section := range expectedSections {
		if !containsSubstring(string(content), section) {
			t.Errorf("expected %q in template content", section)
		}
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestInitCreatesProjectConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Run init (skip beads, kb, kn, claude, tmuxinator to focus on config)
	result, err := initProject(tmpDir, false, true, true, true, true, true, "", "")
	if err != nil {
		t.Fatalf("initProject failed: %v", err)
	}

	// Verify .orch/config.yaml was created
	configPath := filepath.Join(tmpDir, ".orch", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal(".orch/config.yaml should be created by init")
	}

	// Verify config contains servers section
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	content := string(data)
	if !containsSubstring(content, "servers:") {
		t.Error("config should contain 'servers:' section")
	}

	// Verify web and api ports are declared
	if result.PortWeb > 0 {
		if !containsSubstring(content, "web:") {
			t.Error("config should declare web port")
		}
	}
	if result.PortAPI > 0 {
		if !containsSubstring(content, "api:") {
			t.Error("config should declare api port")
		}
	}
}

func TestInitProjectConfigWithAllocatedPorts(t *testing.T) {
	tmpDir := t.TempDir()

	// Run init
	result, err := initProject(tmpDir, false, true, true, true, true, true, "", "")
	if err != nil {
		t.Fatalf("initProject failed: %v", err)
	}

	// Ports should be allocated
	if result.PortWeb == 0 {
		t.Error("PortWeb should be allocated")
	}
	if result.PortAPI == 0 {
		t.Error("PortAPI should be allocated")
	}

	// Config should reflect these ports
	configPath := filepath.Join(tmpDir, ".orch", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	content := string(data)
	// Should contain the allocated ports
	if !containsSubstring(content, "web:") {
		t.Error("config should contain web port declaration")
	}
	if !containsSubstring(content, "api:") {
		t.Error("config should contain api port declaration")
	}
}
