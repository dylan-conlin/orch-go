// Package verify provides verification helpers for agent completion.
package verify

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ConstraintType indicates whether a constraint is required or optional.
type ConstraintType string

const (
	ConstraintRequired ConstraintType = "required"
	ConstraintOptional ConstraintType = "optional"
)

// Constraint represents a skill output constraint extracted from SPAWN_CONTEXT.md.
type Constraint struct {
	Type        ConstraintType
	Pattern     string // Raw pattern from skill (e.g., ".kb/investigations/{date}-inv-*.md")
	Description string // Human-readable description
}

// ConstraintResult represents the result of verifying a single constraint.
type ConstraintResult struct {
	Constraint   Constraint
	Matched      bool     // Whether at least one file matched
	MatchedFiles []string // Files that matched the pattern
	Error        error    // Any error during verification
}

// ConstraintVerificationResult represents the result of verifying all constraints.
type ConstraintVerificationResult struct {
	Passed   bool               // All required constraints have at least one match
	Results  []ConstraintResult // Individual results for each constraint
	Errors   []string           // Error messages for failed required constraints
	Warnings []string           // Info about optional constraints
}

// ExtractConstraints parses SPAWN_CONTEXT.md and extracts skill constraints.
// Looks for a <!-- SKILL-CONSTRAINTS --> block containing constraint definitions.
func ExtractConstraints(workspacePath string) ([]Constraint, error) {
	spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	return ExtractConstraintsFromFile(spawnContextPath)
}

// ExtractConstraintsFromFile parses constraints from a file (for testing).
func ExtractConstraintsFromFile(filePath string) ([]Constraint, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No SPAWN_CONTEXT.md means no constraints
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return ExtractConstraintsFromReader(file)
}

// ExtractConstraintsFromReader parses constraints from any io.Reader (for testing).
func ExtractConstraintsFromReader(file *os.File) ([]Constraint, error) {
	var constraints []Constraint
	inConstraintBlock := false

	// Pattern: <!-- required: pattern | description --> or <!-- optional: pattern | description -->
	constraintPattern := regexp.MustCompile(`<!--\s*(required|optional):\s*(.+?)\s*\|\s*(.+?)\s*-->`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Check for block markers
		if strings.Contains(line, "<!-- SKILL-CONSTRAINTS -->") {
			inConstraintBlock = true
			continue
		}
		if strings.Contains(line, "<!-- /SKILL-CONSTRAINTS -->") {
			inConstraintBlock = false
			continue
		}

		// Only parse constraints within the block
		if !inConstraintBlock {
			continue
		}

		// Extract constraint
		matches := constraintPattern.FindStringSubmatch(line)
		if len(matches) == 4 {
			constraintType := ConstraintRequired
			if matches[1] == "optional" {
				constraintType = ConstraintOptional
			}

			constraints = append(constraints, Constraint{
				Type:        constraintType,
				Pattern:     strings.TrimSpace(matches[2]),
				Description: strings.TrimSpace(matches[3]),
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return constraints, nil
}

// VerifyConstraints checks if all constraints are satisfied in the project directory.
// Required constraints must have at least one matching file.
// Optional constraints are informational only.
func VerifyConstraints(constraints []Constraint, projectDir string) ConstraintVerificationResult {
	result := ConstraintVerificationResult{
		Passed: true,
	}

	for _, c := range constraints {
		cr := verifyConstraint(c, projectDir)
		result.Results = append(result.Results, cr)

		if c.Type == ConstraintRequired {
			if cr.Error != nil {
				result.Passed = false
				result.Errors = append(result.Errors, fmt.Sprintf("constraint error: %s - %v", c.Pattern, cr.Error))
			} else if !cr.Matched {
				result.Passed = false
				result.Errors = append(result.Errors, fmt.Sprintf("required constraint not satisfied: %s (%s)", c.Pattern, c.Description))
			}
		} else {
			// Optional constraints
			if !cr.Matched && cr.Error == nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("optional constraint not matched: %s (%s)", c.Pattern, c.Description))
			}
		}
	}

	return result
}

// verifyConstraint checks if a single constraint is satisfied.
func verifyConstraint(c Constraint, projectDir string) ConstraintResult {
	cr := ConstraintResult{
		Constraint: c,
	}

	// Convert pattern to glob
	globPattern := PatternToGlob(c.Pattern)

	// Combine with project directory
	fullPattern := filepath.Join(projectDir, globPattern)

	// Find matching files
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		cr.Error = fmt.Errorf("invalid glob pattern: %w", err)
		return cr
	}

	cr.MatchedFiles = matches
	cr.Matched = len(matches) > 0

	return cr
}

// PatternToGlob converts a skill constraint pattern to a glob pattern.
// Replaces variables like {date} with appropriate wildcards.
//
// Variable substitutions:
//   - {date} -> *  (matches any date like 2025-12-23)
//   - {workspace} -> *  (matches any workspace name)
//   - {beads} -> *  (matches any beads issue ID)
//
// The pattern already uses * for wildcard matching, which is preserved.
func PatternToGlob(pattern string) string {
	// Replace known variables with wildcards
	result := pattern

	// {date} matches YYYY-MM-DD format, but for globbing we just use *
	result = strings.ReplaceAll(result, "{date}", "*")

	// {workspace} matches the workspace name
	result = strings.ReplaceAll(result, "{workspace}", "*")

	// {beads} matches a beads issue ID
	result = strings.ReplaceAll(result, "{beads}", "*")

	return result
}

// VerifyConstraintsForCompletion is a convenience function that extracts and verifies
// constraints from a workspace's SPAWN_CONTEXT.md against the project directory.
func VerifyConstraintsForCompletion(workspacePath, projectDir string) (ConstraintVerificationResult, error) {
	constraints, err := ExtractConstraints(workspacePath)
	if err != nil {
		return ConstraintVerificationResult{}, fmt.Errorf("failed to extract constraints: %w", err)
	}

	// No constraints means verification passes
	if len(constraints) == 0 {
		return ConstraintVerificationResult{Passed: true}, nil
	}

	return VerifyConstraints(constraints, projectDir), nil
}
