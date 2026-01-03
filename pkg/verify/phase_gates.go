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

// Pre-compiled regex patterns for phase_gates.go
var (
	regexPhaseDeclaration = regexp.MustCompile(`<!--\s*phase:\s*(\w+)\s*\|\s*required:\s*(true|false)\s*-->`)
	regexPhaseReport      = regexp.MustCompile(`(?i)Phase:\s*(\w+)(?:\s*[-–—]\s*(.*))?`)
)

// Phase represents a phase declaration extracted from SPAWN_CONTEXT.md.
type Phase struct {
	Name     string // Phase name (e.g., "investigation", "design", "implementation")
	Required bool   // Whether this phase is required for completion
}

// PhaseGateResult represents the result of verifying phase gates.
type PhaseGateResult struct {
	Passed         bool     // All required phases were reported in order
	RequiredPhases []Phase  // Phases extracted from SKILL-PHASES block
	ReportedPhases []string // Phases reported via beads comments (in order)
	MissingPhases  []string // Required phases that were not reported
	Errors         []string // Error messages for failed phase gate checks
}

// ExtractPhases parses SPAWN_CONTEXT.md and extracts phase declarations.
// Looks for a <!-- SKILL-PHASES --> block containing phase definitions.
// Format: <!-- phase: name | required: true/false -->
func ExtractPhases(workspacePath string) ([]Phase, error) {
	spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	return ExtractPhasesFromFile(spawnContextPath)
}

// ExtractPhasesFromFile parses phases from a file (for testing).
func ExtractPhasesFromFile(filePath string) ([]Phase, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No SPAWN_CONTEXT.md means no phases
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return ExtractPhasesFromReader(file)
}

// ExtractPhasesFromReader parses phases from any *os.File (for testing).
func ExtractPhasesFromReader(file *os.File) ([]Phase, error) {
	var phases []Phase
	inPhaseBlock := false

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Check for block markers
		if strings.Contains(line, "<!-- SKILL-PHASES -->") {
			inPhaseBlock = true
			continue
		}
		if strings.Contains(line, "<!-- /SKILL-PHASES -->") {
			inPhaseBlock = false
			continue
		}

		// Only parse phases within the block
		if !inPhaseBlock {
			continue
		}

		// Extract phase (pattern: <!-- phase: name | required: true/false -->)
		matches := regexPhaseDeclaration.FindStringSubmatch(line)
		if len(matches) == 3 {
			required := strings.ToLower(matches[2]) == "true"
			phases = append(phases, Phase{
				Name:     strings.ToLower(strings.TrimSpace(matches[1])),
				Required: required,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return phases, nil
}

// ExtractReportedPhases parses beads comments and extracts phase reports.
// Returns phases in the order they were reported.
// Format: "Phase: <name> - <summary>" or "Phase: <name>"
func ExtractReportedPhases(comments []Comment) []string {
	var phases []string
	seen := make(map[string]bool)

	for _, comment := range comments {
		matches := regexPhaseReport.FindStringSubmatch(comment.Text)
		if len(matches) >= 2 {
			phase := strings.ToLower(matches[1])
			// Only add each phase once (first occurrence)
			if !seen[phase] {
				phases = append(phases, phase)
				seen[phase] = true
			}
		}
	}

	return phases
}

// VerifyPhaseGates checks if all required phases were reported in beads comments.
// Returns a PhaseGateResult with details about missing phases.
func VerifyPhaseGates(requiredPhases []Phase, comments []Comment) PhaseGateResult {
	result := PhaseGateResult{
		Passed:         true,
		RequiredPhases: requiredPhases,
	}

	// No phases defined = verification passes
	if len(requiredPhases) == 0 {
		return result
	}

	// Extract reported phases from beads comments
	result.ReportedPhases = ExtractReportedPhases(comments)

	// Build a set of reported phases for quick lookup
	reportedSet := make(map[string]bool)
	for _, phase := range result.ReportedPhases {
		reportedSet[phase] = true
	}

	// Check each required phase
	for _, phase := range requiredPhases {
		if !phase.Required {
			continue
		}

		phaseName := strings.ToLower(phase.Name)
		if !reportedSet[phaseName] {
			result.MissingPhases = append(result.MissingPhases, phase.Name)
		}
	}

	// If any required phases are missing, verification fails
	if len(result.MissingPhases) > 0 {
		result.Passed = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("required phases not reported: %s", strings.Join(result.MissingPhases, ", ")))
	}

	return result
}

// VerifyPhaseGatesForCompletion is a convenience function that extracts phases
// from a workspace's SPAWN_CONTEXT.md and verifies them against beads comments.
func VerifyPhaseGatesForCompletion(workspacePath, beadsID string) (PhaseGateResult, error) {
	// Extract phases from SPAWN_CONTEXT.md
	phases, err := ExtractPhases(workspacePath)
	if err != nil {
		return PhaseGateResult{}, fmt.Errorf("failed to extract phases: %w", err)
	}

	// No phases defined means verification passes
	if len(phases) == 0 {
		return PhaseGateResult{Passed: true}, nil
	}

	// Get beads comments
	comments, err := GetComments(beadsID)
	if err != nil {
		return PhaseGateResult{}, fmt.Errorf("failed to get comments: %w", err)
	}

	return VerifyPhaseGates(phases, comments), nil
}
