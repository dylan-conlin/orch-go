package main

import (
	"os"
	"testing"
)

func TestCurrentProjectDirPrefersWorkingDirectory(t *testing.T) {
	oldSourceDir := sourceDir
	sourceDir = t.TempDir()
	t.Cleanup(func() {
		sourceDir = oldSourceDir
	})

	cwd := t.TempDir()
	t.Chdir(cwd)

	got, err := currentProjectDir()
	if err != nil {
		t.Fatalf("currentProjectDir() error = %v", err)
	}

	if got != cwd {
		t.Fatalf("currentProjectDir() = %q, want %q", got, cwd)
	}
}

func TestFormatBeadsIDForDisplay(t *testing.T) {
	// Note: These tests use specific timestamps and expect local timezone conversion
	// Timestamp 1768090360 = Sat Jan 10 16:12:40 PST 2026
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "regular beads ID unchanged",
			input:    "orch-go-abc123",
			expected: "orch-go-abc123",
		},
		{
			name:     "untracked ID with valid timestamp",
			input:    "orch-go-untracked-1768090360",
			expected: "untracked-Jan10-1612", // Jan 10, 2026 16:12 PST
		},
		{
			name:     "untracked ID with different project",
			input:    "my-project-untracked-1768090360",
			expected: "untracked-Jan10-1612",
		},
		{
			name:     "malformed untracked ID (too few parts)",
			input:    "untracked-123",
			expected: "untracked-123", // Should pass through unchanged
		},
		{
			name:     "untracked ID with non-numeric timestamp",
			input:    "orch-go-untracked-notanumber",
			expected: "orch-go-untracked-notanumber", // Should pass through unchanged
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "untracked with Unix epoch (timestamp 0)",
			input:    "test-untracked-0",
			expected: "untracked-Dec31-1600", // Dec 31, 1969 16:00 PST (epoch in PST)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBeadsIDForDisplay(tt.input)
			if got != tt.expected {
				t.Errorf("formatBeadsIDForDisplay(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsUntrackedBeadsID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid untracked ID",
			input:    "orch-go-untracked-1768090360",
			expected: true,
		},
		{
			name:     "regular beads ID",
			input:    "orch-go-abc123",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "contains word untracked in task name",
			input:    "orch-go-fix-untracked-bug-abc123",
			expected: true, // This is a limitation - it matches any ID containing "-untracked-"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUntrackedBeadsID(tt.input)
			if got != tt.expected {
				t.Errorf("isUntrackedBeadsID(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestResolveProjectDir(t *testing.T) {
	// Create a temp directory for testing
	tempDir := t.TempDir()
	currentDir := tempDir + "/current"
	workdir := tempDir + "/workdir"
	workspacePath := tempDir + "/workspace"

	// Create directories
	if err := os.MkdirAll(currentDir, 0755); err != nil {
		t.Fatalf("failed to create current dir: %v", err)
	}
	if err := os.MkdirAll(workdir, 0755); err != nil {
		t.Fatalf("failed to create workdir: %v", err)
	}
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Create SPAWN_CONTEXT.md with PROJECT_DIR
	spawnContextPath := workspacePath + "/SPAWN_CONTEXT.md"
	projectDirFromContext := tempDir + "/context-project"
	if err := os.MkdirAll(projectDirFromContext, 0755); err != nil {
		t.Fatalf("failed to create context project dir: %v", err)
	}
	spawnContext := "Some content\nPROJECT_DIR: " + projectDirFromContext + "\nMore content"
	if err := os.WriteFile(spawnContextPath, []byte(spawnContext), 0644); err != nil {
		t.Fatalf("failed to write SPAWN_CONTEXT.md: %v", err)
	}

	tests := []struct {
		name           string
		workdir        string
		workspacePath  string
		currentDir     string
		wantProjectDir string
		wantSource     string
		wantCross      bool
		wantErr        bool
	}{
		{
			name:           "explicit workdir takes precedence",
			workdir:        workdir,
			workspacePath:  workspacePath,
			currentDir:     currentDir,
			wantProjectDir: workdir,
			wantSource:     "workdir",
			wantCross:      true,
			wantErr:        false,
		},
		{
			name:           "workspace auto-detect when no workdir",
			workdir:        "",
			workspacePath:  workspacePath,
			currentDir:     currentDir,
			wantProjectDir: projectDirFromContext,
			wantSource:     "workspace",
			wantCross:      true,
			wantErr:        false,
		},
		{
			name:           "falls back to current dir",
			workdir:        "",
			workspacePath:  "",
			currentDir:     currentDir,
			wantProjectDir: currentDir,
			wantSource:     "current",
			wantCross:      false,
			wantErr:        false,
		},
		{
			name:          "workdir not a directory",
			workdir:       spawnContextPath, // file, not directory
			workspacePath: "",
			currentDir:    currentDir,
			wantErr:       true,
		},
		{
			name:          "workdir does not exist",
			workdir:       tempDir + "/nonexistent",
			workspacePath: "",
			currentDir:    currentDir,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolveProjectDir(tt.workdir, tt.workspacePath, tt.currentDir)
			if tt.wantErr {
				if err == nil {
					t.Errorf("resolveProjectDir() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("resolveProjectDir() unexpected error: %v", err)
				return
			}
			if result.ProjectDir != tt.wantProjectDir {
				t.Errorf("ProjectDir = %q, want %q", result.ProjectDir, tt.wantProjectDir)
			}
			if result.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", result.Source, tt.wantSource)
			}
			if result.IsCrossProject != tt.wantCross {
				t.Errorf("IsCrossProject = %v, want %v", result.IsCrossProject, tt.wantCross)
			}
		})
	}
}

func TestExtractProjectDirFromWorkspace(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name          string
		content       string
		expected      string
		createContext bool
	}{
		{
			name:          "extracts PROJECT_DIR from valid context",
			content:       "Some header\nPROJECT_DIR: /path/to/project\nMore content",
			expected:      "/path/to/project",
			createContext: true,
		},
		{
			name:          "extracts PROJECT_DIR with extra whitespace",
			content:       "  PROJECT_DIR:   /path/with/spaces  \n",
			expected:      "/path/with/spaces",
			createContext: true,
		},
		{
			name:          "returns empty for missing PROJECT_DIR",
			content:       "No project dir here",
			expected:      "",
			createContext: true,
		},
		{
			name:          "returns empty for missing SPAWN_CONTEXT.md",
			content:       "",
			expected:      "",
			createContext: false,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspacePath := tempDir + "/workspace" + string(rune('0'+i))
			if err := os.MkdirAll(workspacePath, 0755); err != nil {
				t.Fatalf("failed to create workspace: %v", err)
			}

			if tt.createContext {
				spawnContextPath := workspacePath + "/SPAWN_CONTEXT.md"
				if err := os.WriteFile(spawnContextPath, []byte(tt.content), 0644); err != nil {
					t.Fatalf("failed to write SPAWN_CONTEXT.md: %v", err)
				}
			}

			got := extractProjectDirFromWorkspace(workspacePath)
			if got != tt.expected {
				t.Errorf("extractProjectDirFromWorkspace() = %q, want %q", got, tt.expected)
			}
		})
	}
}
