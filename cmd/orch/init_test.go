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
	t.Run("creates all directories", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test1")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		result, err := initProject(testDir, false, true, "") // skip beads
		if err != nil {
			t.Fatalf("initProject failed: %v", err)
		}

		// Check that directories were created
		expectedDirs := []string{
			".orch/workspace",
			".orch/templates",
			".kb/investigations",
			".kb/decisions",
		}

		for _, dir := range expectedDirs {
			fullPath := filepath.Join(testDir, dir)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				t.Errorf("expected directory %s to exist", dir)
			}
		}

		// Check that all directories were marked as created
		if len(result.DirsCreated) != 4 {
			t.Errorf("expected 4 directories created, got %d", len(result.DirsCreated))
		}
	})

	t.Run("idempotent - second run reports existing", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test2")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		// First init
		_, err := initProject(testDir, false, true, "")
		if err != nil {
			t.Fatalf("first initProject failed: %v", err)
		}

		// Second init
		result, err := initProject(testDir, false, true, "")
		if err != nil {
			t.Fatalf("second initProject failed: %v", err)
		}

		// All directories should exist now
		if len(result.DirsCreated) != 0 {
			t.Errorf("expected 0 directories created on second run, got %d", len(result.DirsCreated))
		}
		if len(result.DirsExisted) != 4 {
			t.Errorf("expected 4 directories already existed, got %d", len(result.DirsExisted))
		}
	})

	t.Run("force recreates directories", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test3")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		// First init
		_, err := initProject(testDir, false, true, "")
		if err != nil {
			t.Fatalf("first initProject failed: %v", err)
		}

		// Second init with force
		result, err := initProject(testDir, true, true, "")
		if err != nil {
			t.Fatalf("force initProject failed: %v", err)
		}

		// With force, all directories should be marked as created
		if len(result.DirsCreated) != 4 {
			t.Errorf("expected 4 directories created with force, got %d", len(result.DirsCreated))
		}
	})

	t.Run("skip beads sets flag", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test4")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		result, err := initProject(testDir, false, true, "")
		if err != nil {
			t.Fatalf("initProject failed: %v", err)
		}

		if !result.BeadsSkipped {
			t.Error("expected BeadsSkipped to be true")
		}
	})

	t.Run("synthesis template is written", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "test5")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		_, err := initProject(testDir, false, true, "")
		if err != nil {
			t.Fatalf("initProject failed: %v", err)
		}

		// Check that SYNTHESIS.md template exists
		synthPath := filepath.Join(testDir, ".orch", "templates", "SYNTHESIS.md")
		if _, err := os.Stat(synthPath); os.IsNotExist(err) {
			t.Error("expected SYNTHESIS.md template to exist")
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
