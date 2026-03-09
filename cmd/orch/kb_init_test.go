package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKBInit(t *testing.T) {
	t.Run("creates all expected directories", func(t *testing.T) {
		tmpDir := t.TempDir()

		result, err := kbInitProject(tmpDir)
		if err != nil {
			t.Fatalf("kbInitProject failed: %v", err)
		}

		expectedDirs := []string{
			"models",
			"investigations",
			"decisions",
			"quick",
		}

		for _, dir := range expectedDirs {
			fullPath := filepath.Join(tmpDir, ".kb", dir)
			info, err := os.Stat(fullPath)
			if os.IsNotExist(err) {
				t.Errorf("expected directory .kb/%s to exist", dir)
				continue
			}
			if !info.IsDir() {
				t.Errorf("expected .kb/%s to be a directory", dir)
			}
		}

		if len(result.DirsCreated) != len(expectedDirs) {
			t.Errorf("expected %d directories created, got %d: %v", len(expectedDirs), len(result.DirsCreated), result.DirsCreated)
		}
	})

	t.Run("creates README", func(t *testing.T) {
		tmpDir := t.TempDir()

		result, err := kbInitProject(tmpDir)
		if err != nil {
			t.Fatalf("kbInitProject failed: %v", err)
		}

		readmePath := filepath.Join(tmpDir, ".kb", "README.md")
		content, err := os.ReadFile(readmePath)
		if err != nil {
			t.Fatalf("expected README.md to exist: %v", err)
		}

		if !result.ReadmeCreated {
			t.Error("expected ReadmeCreated to be true")
		}

		// Check that README contains directory descriptions
		s := string(content)
		for _, section := range []string{"models/", "investigations/", "decisions/", "quick/"} {
			if !containsSubstring(s, section) {
				t.Errorf("expected README to mention %s", section)
			}
		}
	})

	t.Run("idempotent - safe to run twice", func(t *testing.T) {
		tmpDir := t.TempDir()

		// First run
		_, err := kbInitProject(tmpDir)
		if err != nil {
			t.Fatalf("first kbInitProject failed: %v", err)
		}

		// Write a file into models/ to verify it's preserved
		markerPath := filepath.Join(tmpDir, ".kb", "models", "test-model.md")
		if err := os.WriteFile(markerPath, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to write marker: %v", err)
		}

		// Second run
		result, err := kbInitProject(tmpDir)
		if err != nil {
			t.Fatalf("second kbInitProject failed: %v", err)
		}

		// Marker file should still exist
		if _, err := os.Stat(markerPath); os.IsNotExist(err) {
			t.Error("marker file should be preserved on second run")
		}

		// All dirs should report as existed
		if len(result.DirsCreated) != 0 {
			t.Errorf("expected 0 dirs created on second run, got %d", len(result.DirsCreated))
		}
		if len(result.DirsExisted) != 4 {
			t.Errorf("expected 4 dirs existed on second run, got %d", len(result.DirsExisted))
		}

		// README should not be overwritten
		if result.ReadmeCreated {
			t.Error("expected ReadmeCreated to be false on second run")
		}
		if !result.ReadmeExisted {
			t.Error("expected ReadmeExisted to be true on second run")
		}
	})

	t.Run("creates .kb root directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, err := kbInitProject(tmpDir)
		if err != nil {
			t.Fatalf("kbInitProject failed: %v", err)
		}

		kbDir := filepath.Join(tmpDir, ".kb")
		info, err := os.Stat(kbDir)
		if os.IsNotExist(err) {
			t.Fatal("expected .kb/ to exist")
		}
		if !info.IsDir() {
			t.Fatal("expected .kb/ to be a directory")
		}
	})
}
