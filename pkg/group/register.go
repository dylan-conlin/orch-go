package group

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// RegisterProject adds a project to the specified group in groups.yaml.
// If groups.yaml doesn't exist, it creates one with the group.
// If the project is already in the group, this is a no-op (idempotent).
// Returns true if the project was added (false if already present).
func RegisterProject(configPath, projectName, groupName string) (bool, error) {
	cfg, err := LoadFromFile(configPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		// Create new groups.yaml with this project
		cfg = &Config{
			Groups: map[string]Group{
				groupName: {Projects: []string{projectName}},
			},
		}
		return true, writeConfig(configPath, cfg)
	}

	g, ok := cfg.Groups[groupName]
	if !ok {
		// Group doesn't exist, create it
		if cfg.Groups == nil {
			cfg.Groups = make(map[string]Group)
		}
		cfg.Groups[groupName] = Group{Projects: []string{projectName}}
		return true, writeConfig(configPath, cfg)
	}

	// Check if already registered (idempotent)
	for _, p := range g.Projects {
		if p == projectName {
			return false, nil
		}
	}

	// Add to group
	g.Projects = append(g.Projects, projectName)
	cfg.Groups[groupName] = g
	return true, writeConfig(configPath, cfg)
}

// AutoDetectGroup determines which group a project likely belongs to
// based on directory proximity to existing group members.
// memberPaths maps project name -> absolute path (e.g. from kb projects list).
// Returns empty string if no match found.
func AutoDetectGroup(projectDir string, memberPaths map[string]string) string {
	cfg, err := Load()
	if err != nil {
		return ""
	}
	return AutoDetectGroupFromConfig(cfg, projectDir, memberPaths)
}

// AutoDetectGroupFromConfig determines group membership using a pre-loaded config.
// Checks two heuristics:
//  1. Explicit members: does any explicit member share the same parent directory?
//  2. Parent-inferred: is the project under a group's parent project path?
func AutoDetectGroupFromConfig(cfg *Config, projectDir string, memberPaths map[string]string) string {
	if cfg == nil {
		return ""
	}

	parentDir := filepath.Dir(projectDir)

	for groupName, g := range cfg.Groups {
		// Check explicit members: same parent directory?
		for _, memberName := range g.Projects {
			if memberPath, ok := memberPaths[memberName]; ok {
				if filepath.Dir(memberPath) == parentDir {
					return groupName
				}
			}
		}

		// Check parent-inferred groups: under the parent project?
		if g.Parent != "" {
			if parentPath, ok := memberPaths[g.Parent]; ok {
				if isSubdirectory(projectDir, parentPath) {
					return groupName
				}
				// Also match if sibling of the parent project
				if filepath.Dir(parentPath) == parentDir {
					return groupName
				}
			}
		}
	}

	return ""
}

// writeConfig marshals and writes a Config to the given path.
func writeConfig(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create groups config: %w", err)
	}
	defer f.Close()

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("failed to write groups config: %w", err)
	}
	return enc.Close()
}
