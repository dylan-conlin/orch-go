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
	Name        string `yaml:"name"`
	SkillType   string `yaml:"skill-type"`
	Audience    string `yaml:"audience"`
	Spawnable   bool   `yaml:"spawnable"`
	Category    string `yaml:"category"`
	Description string `yaml:"description"`
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
