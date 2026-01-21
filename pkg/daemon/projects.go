// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
)

// Project represents a kb-registered project.
type Project struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// BuildListProjectsCommand creates the kb projects list command.
// Separated from execution to enable unit testing of command construction.
func BuildListProjectsCommand() *exec.Cmd {
	return exec.Command("kb", "projects", "list", "--json")
}

// ListProjects returns all kb-registered projects by parsing `kb projects list --json`.
// Projects are sorted alphabetically by name for deterministic ordering.
// Returns an empty slice (not error) if kb is unavailable or returns no projects.
func ListProjects() ([]Project, error) {
	return listProjectsWithCommand(BuildListProjectsCommand)
}

// listProjectsWithCommand is the internal implementation that accepts a command builder
// for testing. This allows unit tests to mock the kb command output.
func listProjectsWithCommand(buildCmd func() *exec.Cmd) ([]Project, error) {
	cmd := buildCmd()
	output, err := cmd.Output()
	if err != nil {
		// kb not available or command failed - return empty list gracefully
		// This is not an error condition for the daemon (kb may not be installed)
		return []Project{}, nil
	}

	// Handle empty output
	if len(output) == 0 {
		return []Project{}, nil
	}

	var projects []Project
	if err := json.Unmarshal(output, &projects); err != nil {
		return nil, fmt.Errorf("failed to parse kb projects list output: %w", err)
	}

	// Sort by name for deterministic ordering
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return projects, nil
}
