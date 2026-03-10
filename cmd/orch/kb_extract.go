package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// KBProjectsRegistry represents the ~/.kb/projects.json structure.
type KBProjectsRegistry struct {
	Projects []KBProject `json:"projects"`
}

// KBProject represents a single project entry.
type KBProject struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// runKBExtract extracts an artifact to another project with lineage tracking.
func runKBExtract(artifactPath, targetProject string, updateSource bool) error {
	// Resolve artifact path to absolute
	absArtifactPath, err := resolveArtifactPath(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to resolve artifact path: %w", err)
	}

	// Verify artifact exists
	if _, err := os.Stat(absArtifactPath); os.IsNotExist(err) {
		return fmt.Errorf("artifact not found: %s", absArtifactPath)
	}

	// Find target project path from registry
	targetPath, err := findProjectPath(targetProject)
	if err != nil {
		return err
	}

	// Determine artifact type and target directory
	targetDir, err := determineTargetDir(absArtifactPath, targetPath)
	if err != nil {
		return err
	}

	// Read original artifact
	content, err := os.ReadFile(absArtifactPath)
	if err != nil {
		return fmt.Errorf("failed to read artifact: %w", err)
	}

	// Get source project name for lineage
	sourceProject := getProjectName(absArtifactPath)

	// Add lineage header
	newContent := addLineageHeader(string(content), absArtifactPath, sourceProject)

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Write to target
	targetFile := filepath.Join(targetDir, filepath.Base(absArtifactPath))
	if err := os.WriteFile(targetFile, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write artifact: %w", err)
	}

	fmt.Printf("Extracted: %s\n", absArtifactPath)
	fmt.Printf("       To: %s\n", targetFile)

	// Optionally update source with extracted-to reference
	if updateSource {
		if err := addExtractedToReference(absArtifactPath, targetFile, targetProject); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update source: %v\n", err)
		} else {
			fmt.Printf("   Updated source with extracted-to reference\n")
		}
	}

	return nil
}

// resolveArtifactPath converts a path to absolute, handling relative paths.
func resolveArtifactPath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return filepath.Join(cwd, path), nil
}

// findProjectPath looks up a project in ~/.kb/projects.json and returns its path.
func findProjectPath(projectName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	registryPath := filepath.Join(homeDir, ".kb", "projects.json")
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return "", fmt.Errorf("failed to read projects registry: %w", err)
	}

	var registry KBProjectsRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return "", fmt.Errorf("failed to parse projects registry: %w", err)
	}

	for _, project := range registry.Projects {
		if project.Name == projectName {
			return project.Path, nil
		}
	}

	return "", fmt.Errorf("project not found in registry: %s (use 'kb projects list' to see available projects)", projectName)
}

// determineTargetDir determines the appropriate .kb/ subdirectory for an artifact.
func determineTargetDir(artifactPath, targetProjectPath string) (string, error) {
	// Extract the .kb/ relative path from the artifact
	// e.g., /path/project/.kb/investigations/foo.md -> investigations
	// e.g., /path/project/.kb/decisions/bar.md -> decisions

	artifactPath = filepath.Clean(artifactPath)

	// Find .kb/ in the path
	kbIndex := strings.Index(artifactPath, "/.kb/")
	if kbIndex == -1 {
		kbIndex = strings.Index(artifactPath, "\\.kb\\") // Windows compatibility
	}

	if kbIndex == -1 {
		// Not in a .kb directory - put in .kb/extracted/
		return filepath.Join(targetProjectPath, ".kb", "extracted"), nil
	}

	// Get the relative path after .kb/
	relativePath := artifactPath[kbIndex+5:] // len("/.kb/") = 5
	relativeDir := filepath.Dir(relativePath)

	return filepath.Join(targetProjectPath, ".kb", relativeDir), nil
}

// getProjectName extracts project name from a path.
func getProjectName(path string) string {
	path = filepath.Clean(path)

	// Find .kb/ in path and get the directory before it
	kbIndex := strings.Index(path, "/.kb/")
	if kbIndex == -1 {
		kbIndex = strings.Index(path, "\\.kb\\")
	}

	if kbIndex == -1 {
		// Fallback: use directory name
		return filepath.Base(filepath.Dir(path))
	}

	projectDir := path[:kbIndex]
	return filepath.Base(projectDir)
}

// addLineageHeader adds extracted-from metadata to artifact content.
func addLineageHeader(content, originalPath, sourceProject string) string {
	timestamp := time.Now().Format("2006-01-02")

	lineageComment := fmt.Sprintf(`<!-- Lineage metadata (added by kb extract) -->
<!-- extracted-from: %s -->
<!-- source-project: %s -->
<!-- extraction-date: %s -->

`, originalPath, sourceProject, timestamp)

	// Check if content starts with YAML frontmatter (---)
	if strings.HasPrefix(strings.TrimSpace(content), "---") {
		// Find end of frontmatter
		lines := strings.SplitN(content, "\n", -1)
		frontmatterEnd := -1
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				frontmatterEnd = i
				break
			}
		}

		if frontmatterEnd > 0 {
			// Insert after frontmatter
			before := strings.Join(lines[:frontmatterEnd+1], "\n")
			after := strings.Join(lines[frontmatterEnd+1:], "\n")
			return before + "\n\n" + lineageComment + after
		}
	}

	// No frontmatter - prepend lineage comment
	return lineageComment + content
}

// addExtractedToReference adds a reference to the source file indicating where it was extracted to.
func addExtractedToReference(sourcePath, targetPath, targetProject string) error {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	timestamp := time.Now().Format("2006-01-02")
	extractedToComment := fmt.Sprintf("\n<!-- extracted-to: %s (project: %s, date: %s) -->\n", targetPath, targetProject, timestamp)

	// Append to end of file
	newContent := string(content) + extractedToComment

	return os.WriteFile(sourcePath, []byte(newContent), 0644)
}
