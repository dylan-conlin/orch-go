// Package group provides project group resolution for orch-go.
// Groups define collections of related projects that share kb context scope,
// daemon polling scope, and account routing.
package group

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Group represents a named collection of related projects.
type Group struct {
	Name     string   `yaml:"-"`                     // Set from map key during load
	Account  string   `yaml:"account,omitempty"`     // Account to use for this group
	Parent   string   `yaml:"parent,omitempty"`      // Parent project name (children auto-discovered from paths)
	Projects []string `yaml:"projects,omitempty"`    // Explicit member project names
}

// Config holds the groups configuration from groups.yaml.
type Config struct {
	Groups map[string]Group `yaml:"groups"`
}

// DefaultConfigPath returns the default path to groups.yaml.
// Prefers ~/.kb/groups.yaml; falls back to ~/.orch/groups.yaml for backward compat.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	primary := filepath.Join(home, ".kb", "groups.yaml")
	if _, err := os.Stat(primary); err == nil {
		return primary
	}
	fallback := filepath.Join(home, ".orch", "groups.yaml")
	if _, err := os.Stat(fallback); err == nil {
		return fallback
	}
	return primary // default to primary even if neither exists
}

// Load reads groups.yaml and returns the config.
// Returns an error if the file doesn't exist or can't be parsed.
func Load() (*Config, error) {
	return LoadFromFile(DefaultConfigPath())
}

// LoadFromFile reads a groups.yaml from the specified path.
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read groups config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse groups config: %w", err)
	}

	// Set Name from map key
	for name, g := range cfg.Groups {
		g.Name = name
		cfg.Groups[name] = g
	}

	return &cfg, nil
}

// GroupsForProject returns all groups a project belongs to.
// Resolution:
// 1. Check explicit groups: is projectName in any group's projects list?
// 2. Check parent groups: is projectName's path a subdirectory of any group's parent project path?
// 3. Parent project itself is always a member of its own group.
// kbProjects maps project name -> absolute path (from kb projects list).
func (c *Config) GroupsForProject(projectName string, kbProjects map[string]string) []Group {
	if c == nil {
		return nil
	}

	var matched []Group
	for groupName, g := range c.Groups {
		// Check explicit membership
		for _, p := range g.Projects {
			if p == projectName {
				result := g
				result.Name = groupName
				matched = append(matched, result)
				goto nextGroup
			}
		}

		// Check parent-child inference
		if g.Parent != "" {
			parentPath, parentExists := kbProjects[g.Parent]
			if parentExists {
				// Parent project itself is a member
				if projectName == g.Parent {
					result := g
					result.Name = groupName
					matched = append(matched, result)
					goto nextGroup
				}

				// Check if project path is under parent path
				projectPath, projectExists := kbProjects[projectName]
				if projectExists && isSubdirectory(projectPath, parentPath) {
					result := g
					result.Name = groupName
					matched = append(matched, result)
				}
			}
		}

	nextGroup:
	}

	return matched
}

// SiblingsOf returns all projects in the same group(s) as the given project,
// excluding the project itself. kbProjects maps project name -> absolute path.
func (c *Config) SiblingsOf(projectName string, kbProjects map[string]string) []string {
	groups := c.GroupsForProject(projectName, kbProjects)
	if len(groups) == 0 {
		return nil
	}

	seen := map[string]bool{projectName: true} // exclude self
	var siblings []string

	for _, g := range groups {
		members := c.ResolveGroupMembers(g.Name, kbProjects)
		for _, m := range members {
			if !seen[m] {
				seen[m] = true
				siblings = append(siblings, m)
			}
		}
	}

	return siblings
}

// ResolveGroupMembers returns all project names belonging to a group,
// including both explicit members and parent-inferred children.
func (c *Config) ResolveGroupMembers(groupName string, kbProjects map[string]string) []string {
	g, ok := c.Groups[groupName]
	if !ok {
		return nil
	}

	seen := map[string]bool{}
	var members []string

	// Add explicit members
	for _, p := range g.Projects {
		if !seen[p] {
			seen[p] = true
			members = append(members, p)
		}
	}

	// Add parent-inferred members
	if g.Parent != "" {
		parentPath, parentExists := kbProjects[g.Parent]
		if parentExists {
			// Add parent itself
			if !seen[g.Parent] {
				seen[g.Parent] = true
				members = append(members, g.Parent)
			}

			// Add children (projects whose path is under parent path)
			for name, path := range kbProjects {
				if !seen[name] && isSubdirectory(path, parentPath) {
					seen[name] = true
					members = append(members, name)
				}
			}
		}
	}

	return members
}

// AllProjectsInGroups returns all unique project names across the given groups.
func AllProjectsInGroups(groups []Group) []string {
	seen := map[string]bool{}
	var projects []string

	for _, g := range groups {
		for _, p := range g.Projects {
			if !seen[p] {
				seen[p] = true
				projects = append(projects, p)
			}
		}
	}

	return projects
}

// isSubdirectory returns true if child is a subdirectory of parent.
// Both paths should be absolute.
func isSubdirectory(child, parent string) bool {
	// Ensure parent ends with separator for proper prefix matching
	parentPrefix := strings.TrimSuffix(parent, string(filepath.Separator)) + string(filepath.Separator)
	return strings.HasPrefix(child, parentPrefix)
}
