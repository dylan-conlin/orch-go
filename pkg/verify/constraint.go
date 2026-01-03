// Package verify provides verification helpers for agent completion.
package verify

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// Pre-compiled regex pattern for constraint.go
var regexConstraintPattern = regexp.MustCompile(`<!--\s*(required|optional):\s*(.+?)\s*\|\s*(.+?)\s*-->`)

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

		// Extract constraint (pattern: <!-- required: pattern | description --> or <!-- optional: pattern | description -->)
		matches := regexConstraintPattern.FindStringSubmatch(line)
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
// This function does NOT filter by spawn time - use VerifyConstraintsWithSpawnTime for that.
func VerifyConstraints(constraints []Constraint, projectDir string) ConstraintVerificationResult {
	return VerifyConstraintsWithSpawnTime(constraints, projectDir, time.Time{})
}

// VerifyConstraintsWithSpawnTime checks if all constraints are satisfied in the project directory.
// If spawnTime is non-zero, only files with mtime >= spawnTime are considered matches.
// This prevents constraints from matching files created by previous spawns.
// Required constraints must have at least one matching file.
// Optional constraints are informational only.
func VerifyConstraintsWithSpawnTime(constraints []Constraint, projectDir string, spawnTime time.Time) ConstraintVerificationResult {
	result := ConstraintVerificationResult{
		Passed: true,
	}

	for _, c := range constraints {
		cr := verifyConstraintWithSpawnTime(c, projectDir, spawnTime)
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
// Does not filter by spawn time - for backward compatibility.
func verifyConstraint(c Constraint, projectDir string) ConstraintResult {
	return verifyConstraintWithSpawnTime(c, projectDir, time.Time{})
}

// verifyConstraintWithSpawnTime checks if a single constraint is satisfied.
// If spawnTime is non-zero, only files with mtime >= spawnTime are considered matches.
func verifyConstraintWithSpawnTime(c Constraint, projectDir string, spawnTime time.Time) ConstraintResult {
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

	// Filter by spawn time if provided
	if !spawnTime.IsZero() {
		var filteredMatches []string
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				continue // Skip files we can't stat
			}
			// Only include files modified at or after spawn time
			if !info.ModTime().Before(spawnTime) {
				filteredMatches = append(filteredMatches, match)
			}
		}
		matches = filteredMatches
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
// It reads the spawn time from the workspace and only matches files created during this spawn.
func VerifyConstraintsForCompletion(workspacePath, projectDir string) (ConstraintVerificationResult, error) {
	constraints, err := ExtractConstraints(workspacePath)
	if err != nil {
		return ConstraintVerificationResult{}, fmt.Errorf("failed to extract constraints: %w", err)
	}

	// No constraints means verification passes
	if len(constraints) == 0 {
		return ConstraintVerificationResult{Passed: true}, nil
	}

	// Read spawn time from workspace to scope constraint matching
	// If no spawn time file exists (legacy workspace), all files will match (zero time = no filtering)
	spawnTime := spawn.ReadSpawnTime(workspacePath)

	return VerifyConstraintsWithSpawnTime(constraints, projectDir, spawnTime), nil
}
