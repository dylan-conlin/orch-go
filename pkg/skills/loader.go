// Package skills provides skill discovery and loading from ~/.claude/skills/.
package skills

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	// ErrSkillNotFound is returned when a skill cannot be found.
	ErrSkillNotFound = errors.New("skill not found")
	// ErrNoFrontmatter is returned when skill content has no YAML frontmatter.
	ErrNoFrontmatter = errors.New("no YAML frontmatter found")
)

// SkillMetadata represents parsed skill YAML frontmatter.
type SkillMetadata struct {
	Name         string   `yaml:"name"`
	SkillType    string   `yaml:"skill-type"`
	Audience     string   `yaml:"audience"`
	Spawnable    bool     `yaml:"spawnable"`
	Composable   bool     `yaml:"composable"`
	Category     string   `yaml:"category"`
	Description  string   `yaml:"description"`
	Dependencies []string `yaml:"dependencies"`
}

// Loader discovers and loads skills from a skills directory.
type Loader struct {
	skillsDir string
}

// NewLoader creates a new skill loader for the given skills directory.
func NewLoader(skillsDir string) *Loader {
	return &Loader{skillsDir: skillsDir}
}

// DefaultLoader creates a loader for the default ~/.claude/skills/ directory.
func DefaultLoader() *Loader {
	home, err := os.UserHomeDir()
	if err != nil {
		return &Loader{skillsDir: ""}
	}
	return &Loader{skillsDir: filepath.Join(home, ".claude", "skills")}
}

// FindSkillPath finds the path to a skill's SKILL.md file.
// It searches:
// 1. Direct symlinks: skillsDir/skillName -> .../SKILL.md
// 2. Subdirectories: skillsDir/*/skillName/SKILL.md
func (l *Loader) FindSkillPath(skillName string) (string, error) {
	if l.skillsDir == "" {
		return "", ErrSkillNotFound
	}

	// Check direct symlink first: skillsDir/skillName/SKILL.md
	directPath := filepath.Join(l.skillsDir, skillName, "SKILL.md")
	if _, err := os.Stat(directPath); err == nil {
		return directPath, nil
	}

	// Search subdirectories: skillsDir/*/skillName/SKILL.md
	entries, err := os.ReadDir(l.skillsDir)
	if err != nil {
		return "", ErrSkillNotFound
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Skip hidden directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		potentialPath := filepath.Join(l.skillsDir, entry.Name(), skillName, "SKILL.md")
		if _, err := os.Stat(potentialPath); err == nil {
			return potentialPath, nil
		}
	}

	return "", ErrSkillNotFound
}

// LoadSkillContent loads the full content of a skill's SKILL.md file.
func (l *Loader) LoadSkillContent(skillName string) (string, error) {
	path, err := l.FindSkillPath(skillName)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// LoadSkillWithDependencies loads a skill's content with all dependencies prepended.
// Dependencies are loaded in order and their content is prepended before the main skill.
// This provides composable skill inheritance where worker-base patterns are available
// to all skills that depend on it.
func (l *Loader) LoadSkillWithDependencies(skillName string) (string, error) {
	// Load main skill content
	mainContent, err := l.LoadSkillContent(skillName)
	if err != nil {
		return "", err
	}

	// Parse metadata to check for dependencies
	metadata, err := ParseSkillMetadata(mainContent)
	if err != nil {
		// If we can't parse metadata, just return the main content
		return mainContent, nil
	}

	// If no dependencies, return as-is
	if len(metadata.Dependencies) == 0 {
		return mainContent, nil
	}

	// Load each dependency and build combined content
	var combined strings.Builder

	for _, dep := range metadata.Dependencies {
		depContent, err := l.LoadSkillContent(dep)
		if err != nil {
			// Dependency not found - could log a warning here, but continue
			continue
		}

		// Strip the frontmatter from dependency content since the main skill
		// already has its own frontmatter
		depBody := stripFrontmatter(depContent)
		combined.WriteString(depBody)
		combined.WriteString("\n\n")
	}

	// Append main skill content (keeping its frontmatter)
	combined.WriteString(mainContent)

	return combined.String(), nil
}

// stripFrontmatter removes YAML frontmatter from skill content, returning just the body.
func stripFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}

	// Find the closing ---
	endIndex := strings.Index(content[3:], "---")
	if endIndex == -1 {
		return content
	}

	// Return content after the closing ---, trimming leading newlines
	afterFrontmatter := content[3+endIndex+3:]
	return strings.TrimLeft(afterFrontmatter, "\n")
}

// ParseSkillMetadata extracts YAML frontmatter from skill content.
func ParseSkillMetadata(content string) (*SkillMetadata, error) {
	// YAML frontmatter is delimited by ---
	if !strings.HasPrefix(content, "---") {
		return nil, ErrNoFrontmatter
	}

	// Find the closing ---
	endIndex := strings.Index(content[3:], "---")
	if endIndex == -1 {
		return nil, ErrNoFrontmatter
	}

	// Extract frontmatter YAML (between the --- delimiters)
	frontmatter := content[3 : 3+endIndex]

	var metadata SkillMetadata
	if err := yaml.Unmarshal([]byte(frontmatter), &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// LoadSkillFiltered loads a skill with dependencies and applies section filtering.
func (l *Loader) LoadSkillFiltered(skillName string, filter *SectionFilter) (string, error) {
	content, err := l.LoadSkillWithDependencies(skillName)
	if err != nil {
		return "", err
	}
	return FilterSkillSections(content, filter), nil
}
