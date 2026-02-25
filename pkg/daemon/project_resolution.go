// Package daemon provides autonomous overnight processing capabilities.
package daemon

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
// This enables the daemon to resolve which project directory an issue
// belongs to, supporting cross-project spawning with --workdir.
type ProjectRegistry struct {
	prefixToDir map[string]string
	currentDir  string
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
		// Fall back to directory basename
		return filepath.Base(projectPath)
	}

	var cfg beadsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil || cfg.IssuePrefix == "" {
		return filepath.Base(projectPath)
	}

	return cfg.IssuePrefix
}

// ProjectEntry represents a registered project with its prefix and directory.
type ProjectEntry struct {
	Prefix string
	Dir    string
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
