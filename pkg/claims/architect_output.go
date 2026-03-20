package claims

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ArchitectOutput represents the parsed ARCHITECT_OUTPUT.yaml produced by an
// architect session resolving a tension cluster.
type ArchitectOutput struct {
	ClusterID      string                `yaml:"cluster_id"`
	ResolutionType string                `yaml:"resolution_type"` // restructure, strengthen, accept, defer
	Summary        string                `yaml:"summary"`
	Issues         []ArchitectOutputItem `yaml:"issues"`
}

// ArchitectOutputItem is a single implementation issue from the architect output.
type ArchitectOutputItem struct {
	Title           string   `yaml:"title"`
	Skill           string   `yaml:"skill"`
	Priority        int      `yaml:"priority"`
	ClaimProvenance []string `yaml:"claim_provenance"`
	DependsOn       []int    `yaml:"depends_on"`
	Description     string   `yaml:"description"`
}

// validResolutionTypes lists the allowed resolution_type values.
var validResolutionTypes = map[string]bool{
	"restructure": true,
	"strengthen":  true,
	"accept":      true,
	"defer":       true,
}

// LoadArchitectOutput reads and parses an ARCHITECT_OUTPUT.yaml file.
func LoadArchitectOutput(path string) (*ArchitectOutput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read architect output: %w", err)
	}
	return ParseArchitectOutput(data)
}

// ParseArchitectOutput parses YAML bytes into an ArchitectOutput, validating
// required fields, resolution type, claim provenance, and dependency DAG.
func ParseArchitectOutput(data []byte) (*ArchitectOutput, error) {
	var out ArchitectOutput
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse architect output yaml: %w", err)
	}

	if out.ClusterID == "" {
		return nil, fmt.Errorf("architect output: cluster_id is required")
	}
	if !validResolutionTypes[out.ResolutionType] {
		return nil, fmt.Errorf("architect output: invalid resolution_type %q (must be restructure, strengthen, accept, or defer)", out.ResolutionType)
	}
	if len(out.Issues) == 0 {
		return nil, fmt.Errorf("architect output: at least one issue is required")
	}

	for i, issue := range out.Issues {
		if issue.Title == "" {
			return nil, fmt.Errorf("architect output: issue[%d] missing title", i)
		}
		if issue.Skill == "" {
			return nil, fmt.Errorf("architect output: issue[%d] missing skill", i)
		}
		if len(issue.ClaimProvenance) == 0 {
			return nil, fmt.Errorf("architect output: issue[%d] missing claim_provenance", i)
		}
		for _, dep := range issue.DependsOn {
			if dep < 0 || dep >= len(out.Issues) {
				return nil, fmt.Errorf("architect output: issue[%d] depends_on index %d out of range [0, %d)", i, dep, len(out.Issues))
			}
			if dep == i {
				return nil, fmt.Errorf("architect output: issue[%d] has self-dependency", i)
			}
		}
	}

	if err := validateDAG(out.Issues); err != nil {
		return nil, fmt.Errorf("architect output: %w", err)
	}

	return &out, nil
}

// validateDAG checks that depends_on indices form a DAG (no cycles).
func validateDAG(issues []ArchitectOutputItem) error {
	n := len(issues)
	// 0 = unvisited, 1 = in-stack, 2 = done
	state := make([]int, n)

	var visit func(i int) error
	visit = func(i int) error {
		if state[i] == 1 {
			return fmt.Errorf("dependency cycle detected involving issue[%d] %q", i, issues[i].Title)
		}
		if state[i] == 2 {
			return nil
		}
		state[i] = 1
		for _, dep := range issues[i].DependsOn {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[i] = 2
		return nil
	}

	for i := range issues {
		if state[i] == 0 {
			if err := visit(i); err != nil {
				return err
			}
		}
	}
	return nil
}
