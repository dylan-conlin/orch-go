package spawn

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModelNameFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "model file in models dir",
			path:     "/path/.kb/models/spawn-architecture.md",
			expected: "spawn-architecture",
		},
		{
			name:     "model file in models dir without leading path",
			path:     ".kb/models/completion-verification.md",
			expected: "completion-verification",
		},
		{
			name:     "probe file inside model subdirectory",
			path:     "/path/.kb/models/spawn-architecture/probes/2026-02-08-test.md",
			expected: "spawn-architecture",
		},
		{
			name:     "model name with date suffix",
			path:     "/path/.kb/models/system-reliability-feb2026.md",
			expected: "system-reliability-feb2026",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ModelNameFromPath(tt.path)
			if got != tt.expected {
				t.Errorf("ModelNameFromPath(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestProbesDirForModel(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "model file in models dir",
			path:     "/path/.kb/models/spawn-architecture.md",
			expected: "/path/.kb/models/spawn-architecture/probes",
		},
		{
			name:     "another model file",
			path:     "/path/.kb/models/completion-verification.md",
			expected: "/path/.kb/models/completion-verification/probes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProbesDirForModel(tt.path)
			if got != tt.expected {
				t.Errorf("ProbesDirForModel(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestProbeFilePath(t *testing.T) {
	path := ProbeFilePath("/path/.kb/models/spawn-architecture.md", "check-workspace-cleanup")
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	expectedDir := "/path/.kb/models/spawn-architecture/probes"
	if dir != expectedDir {
		t.Errorf("ProbeFilePath dir = %q, want %q", dir, expectedDir)
	}

	// Check format: YYYY-MM-DD-slug.md
	if len(base) < 15 { // minimum "2026-02-08-x.md"
		t.Errorf("ProbeFilePath base too short: %q", base)
	}
	if filepath.Ext(base) != ".md" {
		t.Errorf("ProbeFilePath should have .md extension, got %q", base)
	}
	if base[4] != '-' || base[7] != '-' {
		t.Errorf("ProbeFilePath should start with YYYY-MM-DD, got %q", base)
	}
	expectedSuffix := "-check-workspace-cleanup.md"
	if base[10:] != expectedSuffix {
		t.Errorf("ProbeFilePath slug portion = %q, want %q", base[10:], expectedSuffix)
	}
}

func TestListRecentProbes_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	modelsDir := filepath.Join(tmpDir, ".kb", "models")
	probesDir := filepath.Join(modelsDir, "test-model", "probes")
	if err := os.MkdirAll(probesDir, 0755); err != nil {
		t.Fatal(err)
	}

	modelPath := filepath.Join(modelsDir, "test-model.md")
	if err := os.WriteFile(modelPath, []byte("# Test Model"), 0644); err != nil {
		t.Fatal(err)
	}

	probes := ListRecentProbes(modelPath, 5)
	if len(probes) != 0 {
		t.Errorf("ListRecentProbes on empty dir should return 0 probes, got %d", len(probes))
	}
}

func TestListRecentProbes_WithProbes(t *testing.T) {
	tmpDir := t.TempDir()
	modelsDir := filepath.Join(tmpDir, ".kb", "models")
	probesDir := filepath.Join(modelsDir, "test-model", "probes")
	if err := os.MkdirAll(probesDir, 0755); err != nil {
		t.Fatal(err)
	}

	modelPath := filepath.Join(modelsDir, "test-model.md")
	if err := os.WriteFile(modelPath, []byte("# Test Model"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create some probe files
	probeFiles := []string{
		"2026-02-05-check-old-claim.md",
		"2026-02-07-verify-session-handling.md",
		"2026-02-08-test-workspace-cleanup.md",
	}
	for _, name := range probeFiles {
		path := filepath.Join(probesDir, name)
		if err := os.WriteFile(path, []byte("# Probe"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Also add a .gitkeep file (should be ignored)
	if err := os.WriteFile(filepath.Join(probesDir, ".gitkeep"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	probes := ListRecentProbes(modelPath, 5)
	if len(probes) != 3 {
		t.Errorf("ListRecentProbes should return 3 probes, got %d", len(probes))
	}
}

func TestListRecentProbes_MaxLimit(t *testing.T) {
	tmpDir := t.TempDir()
	modelsDir := filepath.Join(tmpDir, ".kb", "models")
	probesDir := filepath.Join(modelsDir, "test-model", "probes")
	if err := os.MkdirAll(probesDir, 0755); err != nil {
		t.Fatal(err)
	}

	modelPath := filepath.Join(modelsDir, "test-model.md")
	if err := os.WriteFile(modelPath, []byte("# Test Model"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create 10 probe files with distinct names
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("2026-02-%02d-probe-%d.md", i+1, i)
		path := filepath.Join(probesDir, name)
		if err := os.WriteFile(path, []byte("# Probe"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	probes := ListRecentProbes(modelPath, 3)
	if len(probes) != 3 {
		t.Errorf("ListRecentProbes with max=3 should return 3 probes, got %d", len(probes))
	}
}

func TestListRecentProbes_NoDir(t *testing.T) {
	probes := ListRecentProbes("/nonexistent/.kb/models/fake-model.md", 5)
	if probes != nil {
		t.Errorf("ListRecentProbes on nonexistent dir should return nil, got %v", probes)
	}
}

func TestFormatProbesForSpawn_Empty(t *testing.T) {
	result := FormatProbesForSpawn(nil)
	if result != "" {
		t.Errorf("FormatProbesForSpawn(nil) should return empty string, got %q", result)
	}

	result = FormatProbesForSpawn([]probeEntry{})
	if result != "" {
		t.Errorf("FormatProbesForSpawn([]) should return empty string, got %q", result)
	}
}

func TestFormatProbesForSpawn_WithProbes(t *testing.T) {
	probes := []probeEntry{
		{
			Path:     "/path/.kb/models/test-model/probes/2026-02-08-check-session.md",
			Name:     "2026-02-08-check-session",
			ModelDir: "test-model",
		},
		{
			Path:     "/path/.kb/models/test-model/probes/2026-02-07-verify-cleanup.md",
			Name:     "2026-02-07-verify-cleanup",
			ModelDir: "test-model",
		},
	}

	result := FormatProbesForSpawn(probes)
	if result == "" {
		t.Error("FormatProbesForSpawn should return non-empty string for probes")
	}

	// Check expected content
	expectedParts := []string{
		"Recent Probes:",
		"2026-02-08-check-session",
		"2026-02-07-verify-cleanup",
		"See:",
	}
	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("FormatProbesForSpawn result should contain %q, got:\n%s", part, result)
		}
	}
}

func TestEnsureProbesDir(t *testing.T) {
	tmpDir := t.TempDir()
	modelsDir := filepath.Join(tmpDir, ".kb", "models")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		t.Fatal(err)
	}

	modelPath := filepath.Join(modelsDir, "test-model.md")
	if err := os.WriteFile(modelPath, []byte("# Test Model"), 0644); err != nil {
		t.Fatal(err)
	}

	probesDir, err := EnsureProbesDir(modelPath)
	if err != nil {
		t.Fatalf("EnsureProbesDir failed: %v", err)
	}

	expectedDir := filepath.Join(modelsDir, "test-model", "probes")
	if probesDir != expectedDir {
		t.Errorf("EnsureProbesDir returned %q, want %q", probesDir, expectedDir)
	}

	// Verify directory was created
	info, err := os.Stat(probesDir)
	if err != nil {
		t.Fatalf("Probes dir should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("Probes dir should be a directory")
	}
}
