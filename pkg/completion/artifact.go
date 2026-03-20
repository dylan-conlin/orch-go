// Package completion provides completion artifact validation for agent work.
// It parses COMPLETION.yaml files, enforces required fields per work type,
// and pre-populates from SYNTHESIS.md when available.
package completion

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/verify"
	"gopkg.in/yaml.v3"
)

// Artifact represents the fields in a COMPLETION.yaml file.
type Artifact struct {
	Verification string `yaml:"verification"`
	Finding      string `yaml:"finding"`
	KBAtom       string `yaml:"kb_atom"`
	FollowUp     string `yaml:"follow_up"`
	Placement    string `yaml:"placement"`
}

// ArtifactResult is the outcome of artifact validation.
type ArtifactResult struct {
	Passed bool
	Errors []string
}

// requiredFields maps issue types to the fields that must be non-empty.
// Unknown types fall back to "task" requirements.
var requiredFields = map[string][]string{
	"feature":       {"verification", "finding", "kb_atom", "follow_up", "placement"},
	"bug":           {"verification", "finding", "follow_up"},
	"task":          {"verification", "finding"},
	"investigation": {"finding", "kb_atom"},
	"question":      {"finding"},
}

// placeholders are values that count as empty.
var placeholders = []string{"todo", "tbd", "n/a", ""}

// ParseArtifact reads and parses COMPLETION.yaml from the given workspace directory.
func ParseArtifact(workspacePath string) (*Artifact, error) {
	path := filepath.Join(workspacePath, "COMPLETION.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read COMPLETION.yaml: %w", err)
	}

	var art Artifact
	if err := yaml.Unmarshal(data, &art); err != nil {
		return nil, fmt.Errorf("parse COMPLETION.yaml: %w", err)
	}
	return &art, nil
}

// ValidateArtifact checks that all required fields for the given issue type
// are present and not placeholders. Returns a list of validation errors.
func ValidateArtifact(art *Artifact, issueType string) []string {
	required, ok := requiredFields[issueType]
	if !ok {
		required = requiredFields["task"] // default fallback
	}

	var errs []string
	for _, field := range required {
		val := fieldValue(art, field)
		if isPlaceholder(val) {
			errs = append(errs, fmt.Sprintf("COMPLETION.yaml: %q is required for %s work but is empty/placeholder", field, issueType))
		}
	}
	return errs
}

// PrePopulateFromSynthesis creates an Artifact pre-populated from SYNTHESIS.md.
// Returns an empty Artifact (not error) if SYNTHESIS.md doesn't exist.
func PrePopulateFromSynthesis(workspacePath string) (*Artifact, error) {
	art := &Artifact{}

	syn, err := verify.ParseSynthesis(workspacePath)
	if err != nil {
		if os.IsNotExist(err) {
			return art, nil
		}
		return art, nil // non-fatal: return empty artifact
	}

	// verification ← Evidence section
	if syn.Evidence != "" {
		art.Verification = syn.Evidence
	}

	// finding ← TLDR, falling back to Delta
	if syn.TLDR != "" {
		art.Finding = syn.TLDR
	} else if syn.Delta != "" {
		art.Finding = syn.Delta
	}

	// kb_atom ← Knowledge section (extract first .kb/ path if present)
	if syn.Knowledge != "" {
		art.KBAtom = extractKBPath(syn.Knowledge)
	}

	// follow_up ← Next section
	if syn.Next != "" {
		art.FollowUp = syn.Next
	} else if len(syn.NextActions) > 0 {
		art.FollowUp = strings.Join(syn.NextActions, "\n")
	}

	return art, nil
}

// CheckArtifact is the top-level validation entry point.
// It parses COMPLETION.yaml, validates required fields, and returns the result.
// When COMPLETION.yaml is missing, it falls back to deriving fields from
// SYNTHESIS.md — agents are instructed to create SYNTHESIS.md (not COMPLETION.yaml),
// so the gate should accept synthesis-derived data rather than always failing.
func CheckArtifact(workspacePath, issueType string) ArtifactResult {
	art, err := ParseArtifact(workspacePath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			// Parse error (malformed YAML) — report as failure
			return ArtifactResult{
				Passed: false,
				Errors: []string{fmt.Sprintf("COMPLETION.yaml: %v", err)},
			}
		}

		// COMPLETION.yaml doesn't exist — fall back to SYNTHESIS.md
		art, err = PrePopulateFromSynthesis(workspacePath)
		if err != nil {
			return ArtifactResult{
				Passed: false,
				Errors: []string{fmt.Sprintf("COMPLETION.yaml missing and SYNTHESIS.md unreadable: %v", err)},
			}
		}

		// Check if synthesis provided any useful data
		if art.Finding == "" && art.Verification == "" {
			return ArtifactResult{
				Passed: false,
				Errors: []string{"COMPLETION.yaml missing and SYNTHESIS.md has no extractable completion data (needs TLDR/Delta and Evidence sections)"},
			}
		}
	}

	errs := ValidateArtifact(art, issueType)
	return ArtifactResult{
		Passed: len(errs) == 0,
		Errors: errs,
	}
}

// fieldValue returns the value of a named field on the Artifact.
func fieldValue(art *Artifact, field string) string {
	switch field {
	case "verification":
		return art.Verification
	case "finding":
		return art.Finding
	case "kb_atom":
		return art.KBAtom
	case "follow_up":
		return art.FollowUp
	case "placement":
		return art.Placement
	default:
		return ""
	}
}

// isPlaceholder returns true if val is empty or a known placeholder.
func isPlaceholder(val string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(val))
	for _, p := range placeholders {
		if trimmed == p {
			return true
		}
	}
	return false
}

// extractKBPath finds the first .kb/ path in text.
func extractKBPath(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		// Strip markdown list markers
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")
		idx := strings.Index(line, ".kb/")
		if idx >= 0 {
			// Extract the path (up to next space or end of line)
			path := line[idx:]
			if spaceIdx := strings.IndexAny(path, " \t)"); spaceIdx > 0 {
				path = path[:spaceIdx]
			}
			return path
		}
	}
	return ""
}
