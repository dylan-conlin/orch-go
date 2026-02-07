package binutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveBinary_EnvVarOverride(t *testing.T) {
	// Create a temporary binary
	tmpDir := t.TempDir()
	tmpBin := filepath.Join(tmpDir, "testbin")
	if err := os.WriteFile(tmpBin, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	// Set environment variable
	envVar := "TEST_BIN_PATH"
	t.Setenv(envVar, tmpBin)

	// Resolve should use env var first
	path, err := ResolveBinary("nonexistent", envVar, []string{})
	if err != nil {
		t.Fatalf("expected to find binary via env var, got error: %v", err)
	}

	if path != tmpBin {
		t.Errorf("expected path %s, got %s", tmpBin, path)
	}
}

func TestResolveBinary_EnvVarWithHome(t *testing.T) {
	// Create a temporary binary
	tmpDir := t.TempDir()
	tmpBin := filepath.Join(tmpDir, "testbin")
	if err := os.WriteFile(tmpBin, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	// Set HOME to temp directory
	home := os.Getenv("HOME")
	relPath := strings.Replace(tmpBin, home, "$HOME", 1)

	// Set environment variable with $HOME
	envVar := "TEST_BIN_PATH"
	t.Setenv(envVar, relPath)

	// Resolve should expand $HOME
	path, err := ResolveBinary("testbin", envVar, []string{})
	if err != nil {
		t.Fatalf("expected to find binary via env var with $HOME, got error: %v", err)
	}

	if path != tmpBin {
		t.Errorf("expected path %s, got %s", tmpBin, path)
	}
}

func TestResolveBinary_PATH(t *testing.T) {
	// Create a temporary directory and add to PATH
	tmpDir := t.TempDir()
	tmpBin := filepath.Join(tmpDir, "testbin")
	if err := os.WriteFile(tmpBin, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	// Add tmpDir to PATH
	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+":"+originalPath)

	// Resolve should find in PATH
	path, err := ResolveBinary("testbin", "", []string{})
	if err != nil {
		t.Fatalf("expected to find binary in PATH, got error: %v", err)
	}

	if !strings.Contains(path, "testbin") {
		t.Errorf("expected path to contain 'testbin', got %s", path)
	}
}

func TestResolveBinary_SearchPaths(t *testing.T) {
	// Create a temporary binary in a known location
	tmpDir := t.TempDir()
	tmpBin := filepath.Join(tmpDir, "testbin")
	if err := os.WriteFile(tmpBin, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	// Use search paths
	searchPaths := []string{tmpBin}

	// Resolve should find in search paths
	path, err := ResolveBinary("testbin", "", searchPaths)
	if err != nil {
		t.Fatalf("expected to find binary in search paths, got error: %v", err)
	}

	if path != tmpBin {
		t.Errorf("expected path %s, got %s", tmpBin, path)
	}
}

func TestResolveBinary_NotFound(t *testing.T) {
	// Try to resolve a binary that doesn't exist
	_, err := ResolveBinary("nonexistent_binary_12345", "NONEXISTENT_VAR", []string{"/fake/path/binary"})
	if err == nil {
		t.Fatal("expected error when binary not found")
	}

	// Error should list searched locations
	errMsg := err.Error()
	if !strings.Contains(errMsg, "not found") {
		t.Errorf("error should mention 'not found', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "Searched:") {
		t.Errorf("error should list searched locations, got: %s", errMsg)
	}
}

func TestResolveBinary_PrecedenceOrder(t *testing.T) {
	// Create binaries in different locations
	tmpDir1 := t.TempDir()
	tmpBin1 := filepath.Join(tmpDir1, "testbin")
	if err := os.WriteFile(tmpBin1, []byte("#!/bin/sh\necho env"), 0755); err != nil {
		t.Fatal(err)
	}

	tmpDir2 := t.TempDir()
	tmpBin2 := filepath.Join(tmpDir2, "testbin")
	if err := os.WriteFile(tmpBin2, []byte("#!/bin/sh\necho path"), 0755); err != nil {
		t.Fatal(err)
	}

	tmpDir3 := t.TempDir()
	tmpBin3 := filepath.Join(tmpDir3, "testbin")
	if err := os.WriteFile(tmpBin3, []byte("#!/bin/sh\necho search"), 0755); err != nil {
		t.Fatal(err)
	}

	// Set env var to tmpBin1
	envVar := "TEST_BIN"
	t.Setenv(envVar, tmpBin1)

	// Add tmpBin2 to PATH
	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir2+":"+originalPath)

	// Use tmpBin3 in search paths
	searchPaths := []string{tmpBin3}

	// Should find env var first (highest precedence)
	path, err := ResolveBinary("testbin", envVar, searchPaths)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path != tmpBin1 {
		t.Errorf("expected env var path %s, got %s", tmpBin1, path)
	}

	// Now unset env var - should find PATH next
	os.Unsetenv(envVar)
	path, err = ResolveBinary("testbin", envVar, searchPaths)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(path, tmpDir2) {
		t.Errorf("expected PATH binary from %s, got %s", tmpDir2, path)
	}
}

func TestCommonSearchPaths(t *testing.T) {
	paths := CommonSearchPaths("testbin")

	// Should include common locations
	expectedPaths := []string{
		"$HOME/bin/testbin",
		"$HOME/go/bin/testbin",
		"$HOME/.bun/bin/testbin",
		"$HOME/.local/bin/testbin",
		"/usr/local/bin/testbin",
		"/opt/homebrew/bin/testbin",
	}

	if len(paths) != len(expectedPaths) {
		t.Errorf("expected %d paths, got %d", len(expectedPaths), len(paths))
	}

	for i, expected := range expectedPaths {
		if i >= len(paths) {
			break
		}
		if paths[i] != expected {
			t.Errorf("path[%d]: expected %s, got %s", i, expected, paths[i])
		}
	}
}

func TestExpandHome(t *testing.T) {
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME not set")
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"$HOME/bin/test", home + "/bin/test"},
		{"/usr/local/bin/test", "/usr/local/bin/test"},
		{"$HOME", home},
	}

	for _, tt := range tests {
		result := expandHome(tt.input)
		if result != tt.expected {
			t.Errorf("expandHome(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}
