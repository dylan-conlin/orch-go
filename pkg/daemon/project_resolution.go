// Package daemon provides autonomous overnight processing capabilities.
// This file re-exports project identity types from pkg/identity for
// backward compatibility within the daemon package.
package daemon

import "github.com/dylan-conlin/orch-go/pkg/identity"

// ProjectRegistry is an alias for identity.ProjectRegistry.
type ProjectRegistry = identity.ProjectRegistry

// ProjectEntry is an alias for identity.ProjectEntry.
type ProjectEntry = identity.ProjectEntry

// NewProjectRegistry delegates to identity.NewProjectRegistry.
func NewProjectRegistry() (*ProjectRegistry, error) {
	return identity.NewProjectRegistry()
}

// NewProjectRegistryFromMap delegates to identity.NewProjectRegistryFromMap.
func NewProjectRegistryFromMap(prefixToDir map[string]string, currentDir string) *ProjectRegistry {
	return identity.NewProjectRegistryFromMap(prefixToDir, currentDir)
}

// BuildProjectDirNames delegates to identity.BuildProjectDirNames.
func BuildProjectDirNames(registry *ProjectRegistry) map[string]string {
	return identity.BuildProjectDirNames(registry)
}

// ListReadyIssuesMultiProject is defined in issue_adapter.go and uses *ProjectRegistry.
// It stays in daemon because it depends on beads CLI integration.
