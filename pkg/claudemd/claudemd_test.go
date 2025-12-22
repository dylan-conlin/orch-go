package claudemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListAvailableTypes(t *testing.T) {
	types := ListAvailableTypes()

	expected := []ProjectType{
		ProjectTypeGoCLI,
		ProjectTypeSvelteApp,
		ProjectTypePythonCLI,
		ProjectTypeMinimal,
	}

	if len(types) != len(expected) {
		t.Errorf("Expected %d types, got %d", len(expected), len(types))
	}

	for _, e := range expected {
		found := false
		for _, typ := range types {
			if typ == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected type %q not found", e)
		}
	}
}

func TestLoadTemplate_Embedded(t *testing.T) {
	tests := []struct {
		projectType ProjectType
		contains    string
	}{
		{ProjectTypeGoCLI, "make build"},
		{ProjectTypeSvelteApp, "bun install"},
		{ProjectTypePythonCLI, "uv sync"},
		{ProjectTypeMinimal, "Brief project description"},
	}

	for _, tt := range tests {
		t.Run(string(tt.projectType), func(t *testing.T) {
			content, err := LoadTemplate(tt.projectType)
			if err != nil {
				t.Fatalf("LoadTemplate failed: %v", err)
			}

			if !strings.Contains(content, tt.contains) {
				t.Errorf("Template for %s should contain %q", tt.projectType, tt.contains)
			}
		})
	}
}

func TestLoadTemplate_UserOverride(t *testing.T) {
	// Create a temporary user template directory
	tmpDir := t.TempDir()
	userDir := filepath.Join(tmpDir, ".orch", "templates", "claude")
	if err := os.MkdirAll(userDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Write a custom template
	customContent := "# Custom {{.ProjectName}} Template\nThis is a user override."
	if err := os.WriteFile(filepath.Join(userDir, "minimal.md"), []byte(customContent), 0644); err != nil {
		t.Fatalf("Failed to write custom template: %v", err)
	}

	// Save and restore HOME
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Load should use the user template
	content, err := LoadTemplate(ProjectTypeMinimal)
	if err != nil {
		t.Fatalf("LoadTemplate failed: %v", err)
	}

	if !strings.Contains(content, "Custom") {
		t.Errorf("Expected user template override, got embedded template")
	}
}

func TestLoadTemplate_InvalidType(t *testing.T) {
	_, err := LoadTemplate("invalid-type")
	if err == nil {
		t.Error("Expected error for invalid project type")
	}
}

func TestRender(t *testing.T) {
	data := TemplateData{
		ProjectName: "my-project",
		ProjectType: ProjectTypeGoCLI,
		PortWeb:     5173,
		PortAPI:     3333,
	}

	content, err := Render(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Check that template variables were substituted
	if !strings.Contains(content, "my-project") {
		t.Error("Expected project name to be substituted")
	}

	if !strings.Contains(content, "make build") {
		t.Error("Expected go-cli template content")
	}
}

func TestRender_DefaultsToMinimal(t *testing.T) {
	data := TemplateData{
		ProjectName: "test-project",
		// ProjectType intentionally left empty
	}

	content, err := Render(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Should use minimal template
	if !strings.Contains(content, "test-project") {
		t.Error("Expected project name to be substituted")
	}
}

func TestDetectProjectType(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(dir string) error
		expected ProjectType
	}{
		{
			name: "go-cli",
			setup: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644); err != nil {
					return err
				}
				return os.Mkdir(filepath.Join(dir, "cmd"), 0755)
			},
			expected: ProjectTypeGoCLI,
		},
		{
			name: "svelte-app",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "svelte.config.js"), []byte("export default {}"), 0644)
			},
			expected: ProjectTypeSvelteApp,
		},
		{
			name: "python-cli",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]"), 0644)
			},
			expected: ProjectTypePythonCLI,
		},
		{
			name: "minimal-empty",
			setup: func(dir string) error {
				return nil // Empty directory
			},
			expected: ProjectTypeMinimal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			if err := tt.setup(tmpDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			result := DetectProjectType(tmpDir)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestWriteToProject(t *testing.T) {
	tmpDir := t.TempDir()

	data := TemplateData{
		ProjectName: "test-project",
		ProjectType: ProjectTypeMinimal,
	}

	path, err := WriteToProject(tmpDir, data)
	if err != nil {
		t.Fatalf("WriteToProject failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "CLAUDE.md")
	if path != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, path)
	}

	// Verify file exists
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if !strings.Contains(string(content), "test-project") {
		t.Error("Expected project name in file content")
	}
}

func TestWriteToProject_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create existing CLAUDE.md
	existingPath := filepath.Join(tmpDir, "CLAUDE.md")
	if err := os.WriteFile(existingPath, []byte("existing content"), 0644); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	data := TemplateData{
		ProjectName: "test-project",
		ProjectType: ProjectTypeMinimal,
	}

	_, err := WriteToProject(tmpDir, data)
	if err == nil {
		t.Error("Expected error when CLAUDE.md already exists")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected 'already exists' error, got: %v", err)
	}
}

func TestUserTemplateDir(t *testing.T) {
	dir := UserTemplateDir()

	// Should contain .orch/templates/claude
	if !strings.Contains(dir, ".orch") || !strings.Contains(dir, "claude") {
		t.Errorf("Unexpected user template dir: %s", dir)
	}
}
