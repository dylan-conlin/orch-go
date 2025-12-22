// Package claudemd provides CLAUDE.md template generation for different project types.
package claudemd

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates/*.md
var embeddedTemplates embed.FS

// ProjectType represents a known project type.
type ProjectType string

const (
	ProjectTypeGoCLI     ProjectType = "go-cli"
	ProjectTypeSvelteApp ProjectType = "svelte-app"
	ProjectTypePythonCLI ProjectType = "python-cli"
	ProjectTypeMinimal   ProjectType = "minimal"
)

// TemplateData holds the variables for CLAUDE.md template rendering.
type TemplateData struct {
	ProjectName string
	ProjectType ProjectType
	PortWeb     int
	PortAPI     int
}

// UserTemplateDir returns the path to user-customizable templates.
func UserTemplateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".orch", "templates", "claude")
}

// templateFileName returns the filename for a project type.
func templateFileName(projectType ProjectType) string {
	return string(projectType) + ".md"
}

// LoadTemplate loads a template for the given project type.
// It first checks user-customizable path (~/.orch/templates/claude/),
// then falls back to embedded templates.
func LoadTemplate(projectType ProjectType) (string, error) {
	filename := templateFileName(projectType)

	// Check user-customizable path first
	userPath := filepath.Join(UserTemplateDir(), filename)
	if content, err := os.ReadFile(userPath); err == nil {
		return string(content), nil
	}

	// Fall back to embedded templates
	content, err := embeddedTemplates.ReadFile("templates/" + filename)
	if err != nil {
		return "", fmt.Errorf("template not found for project type %q: %w", projectType, err)
	}

	return string(content), nil
}

// ListAvailableTypes returns all available project types.
func ListAvailableTypes() []ProjectType {
	return []ProjectType{
		ProjectTypeGoCLI,
		ProjectTypeSvelteApp,
		ProjectTypePythonCLI,
		ProjectTypeMinimal,
	}
}

// Render loads and renders a template with the given data.
func Render(data TemplateData) (string, error) {
	if data.ProjectType == "" {
		data.ProjectType = ProjectTypeMinimal
	}

	templateContent, err := LoadTemplate(data.ProjectType)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("claudemd").Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// WriteToProject writes a rendered CLAUDE.md to the project directory.
// Returns the path to the created file.
func WriteToProject(projectDir string, data TemplateData) (string, error) {
	content, err := Render(data)
	if err != nil {
		return "", err
	}

	claudePath := filepath.Join(projectDir, "CLAUDE.md")

	// Check if file already exists
	if _, err := os.Stat(claudePath); err == nil {
		return "", fmt.Errorf("CLAUDE.md already exists at %s", claudePath)
	}

	if err := os.WriteFile(claudePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write CLAUDE.md: %w", err)
	}

	return claudePath, nil
}

// DetectProjectType attempts to detect the project type from the directory contents.
func DetectProjectType(projectDir string) ProjectType {
	// Check for Go CLI indicators
	if fileExists(filepath.Join(projectDir, "go.mod")) {
		if fileExists(filepath.Join(projectDir, "cmd")) || fileExists(filepath.Join(projectDir, "main.go")) {
			return ProjectTypeGoCLI
		}
	}

	// Check for SvelteKit indicators
	if fileExists(filepath.Join(projectDir, "svelte.config.js")) ||
		fileExists(filepath.Join(projectDir, "svelte.config.ts")) {
		return ProjectTypeSvelteApp
	}

	// Check for Python CLI indicators
	if fileExists(filepath.Join(projectDir, "pyproject.toml")) {
		return ProjectTypePythonCLI
	}

	// Default to minimal
	return ProjectTypeMinimal
}

// fileExists checks if a file or directory exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
