// Package identity provides project identification and resolution.
// It maps issue ID prefixes to project directories and resolves
// project directories from working directory or explicit paths.
package identity

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProjectRegistry maps issue ID prefixes to project directories.
// This enables cross-project operations by resolving which project
// directory an issue belongs to.
type ProjectRegistry struct {
	prefixToDir map[string]string
	currentDir  string
}

// ProjectEntry represents a registered project with its prefix and directory.
type ProjectEntry struct {
	Prefix string
	Dir    string
}

// beadsConfig represents the minimal structure of .beads/config.yaml.
type beadsConfig struct {
	IssuePrefix string `yaml:"issue-prefix"`
}

// kbProject represents a project entry from `kb projects list --json`.
type kbProject struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// NewProjectRegistryFromMap creates a ProjectRegistry from an explicit prefix-to-directory
// mapping. Useful for testing and for callers that already have project data.
func NewProjectRegistryFromMap(prefixToDir map[string]string, currentDir string) *ProjectRegistry {
	m := make(map[string]string, len(prefixToDir))
	for k, v := range prefixToDir {
		m[k] = v
	}
	return &ProjectRegistry{
		prefixToDir: m,
		currentDir:  currentDir,
	}
}

// NewProjectRegistry builds a registry by querying `kb projects list --json`
// and reading each project's .beads/config.yaml for the issue prefix.
// Falls back to using the directory basename as the prefix.
func NewProjectRegistry() (*ProjectRegistry, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	cmd := exec.Command("kb", "projects", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run kb projects list: %w", err)
	}

	var projects []kbProject
	if err := json.Unmarshal(output, &projects); err != nil {
		return nil, fmt.Errorf("failed to parse kb projects list output: %w", err)
	}

	registry := &ProjectRegistry{
		prefixToDir: make(map[string]string),
		currentDir:  currentDir,
	}

	for _, proj := range projects {
		prefix := resolvePrefix(proj.Path)
		if prefix != "" {
			registry.prefixToDir[prefix] = proj.Path
		}
	}

	return registry, nil
}

// resolvePrefix reads .beads/config.yaml for the issue-prefix field.
// Falls back to the directory basename if the config is missing or unreadable.
func resolvePrefix(projectPath string) string {
	configPath := filepath.Join(projectPath, ".beads", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return filepath.Base(projectPath)
	}

	var cfg beadsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil || cfg.IssuePrefix == "" {
		return filepath.Base(projectPath)
	}

	return cfg.IssuePrefix
}

// Projects returns all registered projects as prefix-directory pairs.
func (r *ProjectRegistry) Projects() []ProjectEntry {
	if r == nil {
		return nil
	}
	entries := make([]ProjectEntry, 0, len(r.prefixToDir))
	for prefix, dir := range r.prefixToDir {
		entries = append(entries, ProjectEntry{Prefix: prefix, Dir: dir})
	}
	return entries
}

// CurrentDir returns the registry's current working directory.
func (r *ProjectRegistry) CurrentDir() string {
	if r == nil {
		return ""
	}
	return r.currentDir
}

// ExtractPrefix returns the prefix portion of an issue ID.
// Issue IDs follow the format "prefix-hash" (e.g., "orch-go-1169", "bd-85487068").
// Returns the longest matching registered prefix, or the text before the last hyphen.
func (r *ProjectRegistry) ExtractPrefix(issueID string) string {
	if r == nil || issueID == "" {
		return ""
	}

	// Try longest-match against registered prefixes.
	// This handles multi-segment prefixes like "orch-go" correctly.
	bestMatch := ""
	for prefix := range r.prefixToDir {
		if strings.HasPrefix(issueID, prefix+"-") && len(prefix) > len(bestMatch) {
			bestMatch = prefix
		}
	}
	if bestMatch != "" {
		return bestMatch
	}

	// Fallback: text before the last hyphen
	lastDash := strings.LastIndex(issueID, "-")
	if lastDash <= 0 {
		return ""
	}
	return issueID[:lastDash]
}

// Resolve returns the project directory for an issue ID.
// Returns empty string if the issue belongs to the current project
// (no --workdir needed) or if the prefix is not found in the registry.
func (r *ProjectRegistry) Resolve(issueID string) string {
	if r == nil {
		return ""
	}

	prefix := r.ExtractPrefix(issueID)
	if prefix == "" {
		return ""
	}

	dir, ok := r.prefixToDir[prefix]
	if !ok {
		return ""
	}

	// If it resolves to the current directory, no workdir needed
	if dir == r.currentDir {
		return ""
	}

	return dir
}

// ResolveProjectDirectory determines the project directory and name.
// Uses workdir if provided, otherwise current working directory.
func ResolveProjectDirectory(workdir string) (projectDir, projectName string, err error) {
	if workdir != "" {
		projectDir, err = filepath.Abs(workdir)
		if err != nil {
			return "", "", fmt.Errorf("failed to resolve workdir path: %w", err)
		}
		if stat, err := os.Stat(projectDir); err != nil {
			return "", "", fmt.Errorf("workdir does not exist: %s", projectDir)
		} else if !stat.IsDir() {
			return "", "", fmt.Errorf("workdir is not a directory: %s", projectDir)
		}
	} else {
		projectDir, err = os.Getwd()
		if err != nil {
			return "", "", fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	projectName = filepath.Base(projectDir)
	return projectDir, projectName, nil
}

// ResolveProject determines the project directory for a beads ID.
// Three-layer fallback:
//  1. Explicit workdirOverride (highest priority)
//  2. ProjectRegistry prefix → directory mapping (O(1), fast)
//  3. Fall back to current working directory
//
// Returns the absolute project directory path. If the beads ID resolves
// to the current directory (same project), returns the current directory.
func ResolveProject(beadsID, workdirOverride string) (string, error) {
	// Layer 1: Explicit workdir override
	if workdirOverride != "" {
		dir, _, err := ResolveProjectDirectory(workdirOverride)
		return dir, err
	}

	// Layer 2: ProjectRegistry prefix → directory mapping
	if beadsID != "" {
		registry, err := NewProjectRegistry()
		if err == nil {
			dir := registry.Resolve(beadsID)
			if dir != "" {
				return dir, nil
			}
		}
		// Registry construction or lookup failed — fall through to CWD
	}

	// Layer 3: Current working directory
	dir, _, err := ResolveProjectDirectory("")
	return dir, err
}

// ResolveProjectFrom is like ResolveProject but accepts a pre-built registry
// to avoid repeated kb projects list calls in commands that resolve multiple IDs.
func ResolveProjectFrom(registry *ProjectRegistry, beadsID, workdirOverride string) (string, error) {
	// Layer 1: Explicit workdir override
	if workdirOverride != "" {
		dir, _, err := ResolveProjectDirectory(workdirOverride)
		return dir, err
	}

	// Layer 2: Registry lookup
	if beadsID != "" && registry != nil {
		dir := registry.Resolve(beadsID)
		if dir != "" {
			return dir, nil
		}
	}

	// Layer 3: Current working directory
	dir, _, err := ResolveProjectDirectory("")
	return dir, err
}

// BuildProjectDirNames builds a map from project prefix to directory basename
// using the ProjectRegistry. Returns empty map if registry is nil.
func BuildProjectDirNames(registry *ProjectRegistry) map[string]string {
	if registry == nil {
		return map[string]string{}
	}

	names := make(map[string]string)
	for _, proj := range registry.Projects() {
		basename := filepath.Base(proj.Dir)
		names[proj.Prefix] = basename
	}
	return names
}
