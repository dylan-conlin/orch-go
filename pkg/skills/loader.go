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

// SectionFilter configures which sections to keep when filtering skill content.
// Used for progressive skill disclosure: only include the phases/modes relevant
// to a specific spawn, reducing prompt token count.
type SectionFilter struct {
	Phases    []string // Include only these phases (empty = all)
	Mode      string   // Include only this mode (empty = all)
	SpawnMode string   // "interactive" or "autonomous" (empty = all)
}

// isEmpty returns true if the filter has no active constraints.
func (f *SectionFilter) IsEmpty() bool {
	return len(f.Phases) == 0 && f.Mode == "" && f.SpawnMode == ""
}

// FilterSkillSections removes @section-annotated sections that don't match the filter.
// Sections without annotations are always preserved.
// If filter is nil, returns content unchanged (backward compatible).
//
// Marker format: <!-- @section: key=value, key=value -->
// Close marker:  <!-- @/section -->
//
// Supported keys:
//   - phase: matches against filter.Phases (e.g., phase=investigation)
//   - mode: matches against filter.Mode (e.g., mode=tdd)
//   - spawn-mode: matches against filter.SpawnMode (e.g., spawn-mode=autonomous)
func FilterSkillSections(content string, filter *SectionFilter) string {
	if filter == nil || filter.IsEmpty() {
		return content
	}

	lines := strings.Split(content, "\n")
	var result []string
	skipping := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if skipping {
			if trimmed == "<!-- @/section -->" {
				skipping = false
			}
			continue
		}

		// Check for section open marker
		if strings.HasPrefix(trimmed, "<!-- @section:") && strings.HasSuffix(trimmed, "-->") {
			attrs := parseSectionAttrs(trimmed)
			if !sectionMatches(attrs, filter) {
				skipping = true
				continue
			}
			// Matching section — skip the marker line but include content
			continue
		}

		// Close marker for an included section — skip the marker itself
		if trimmed == "<!-- @/section -->" {
			continue
		}

		result = append(result, line)
	}

	// Collapse runs of 3+ consecutive blank lines to 2
	output := strings.Join(result, "\n")
	for strings.Contains(output, "\n\n\n\n") {
		output = strings.ReplaceAll(output, "\n\n\n\n", "\n\n\n")
	}

	return output
}

// parseSectionAttrs extracts key=value pairs from a section marker line.
// Example: "<!-- @section: phase=investigation, mode=tdd -->" returns
// map[string]string{"phase": "investigation", "mode": "tdd"}.
func parseSectionAttrs(marker string) map[string]string {
	attrs := make(map[string]string)

	start := strings.Index(marker, "<!-- @section:")
	if start == -1 {
		return attrs
	}
	end := strings.Index(marker, "-->")
	if end == -1 || end <= start {
		return attrs
	}

	inner := marker[start+len("<!-- @section:") : end]
	inner = strings.TrimSpace(inner)

	for _, part := range strings.Split(inner, ",") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			attrs[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	return attrs
}

// sectionMatches returns true if a section's attributes match the filter.
// A section matches when every attribute present in the section is accepted by
// the corresponding filter field. Attributes not mentioned in the section are
// not checked (i.e., a section with only phase=X is included regardless of Mode
// if the phase matches).
func sectionMatches(attrs map[string]string, filter *SectionFilter) bool {
	if phase, ok := attrs["phase"]; ok && len(filter.Phases) > 0 {
		found := false
		for _, p := range filter.Phases {
			if p == phase {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if mode, ok := attrs["mode"]; ok && filter.Mode != "" {
		if mode != filter.Mode {
			return false
		}
	}

	if spawnMode, ok := attrs["spawn-mode"]; ok && filter.SpawnMode != "" {
		if spawnMode != filter.SpawnMode {
			return false
		}
	}

	return true
}

// LoadSkillFiltered loads a skill with dependencies and applies section filtering.
func (l *Loader) LoadSkillFiltered(skillName string, filter *SectionFilter) (string, error) {
	content, err := l.LoadSkillWithDependencies(skillName)
	if err != nil {
		return "", err
	}
	return FilterSkillSections(content, filter), nil
}
