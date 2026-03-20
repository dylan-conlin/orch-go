// Package verify provides verification helpers for agent completion.
// This file implements the Consequence Sensor verification gate for architect skills.
// Ensures that every gate/hook recommended by an architect declares how its effect
// will be observed (consequence sensor). If no sensor exists, it must explicitly
// state "none — open loop" to make the gap visible at design time.
package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GateConsequenceSensor is the gate name for consequence sensor verification.
const GateConsequenceSensor = "consequence_sensor"

// ConsequenceSensorResult holds the result of consequence sensor verification.
type ConsequenceSensorResult struct {
	Passed      bool
	Errors      []string
	Warnings    []string
	GatesFailed []string
	OpenLoops   []string // Mechanisms declared as "none — open loop"
}

// gateHookKeywords are terms that indicate a gate or hook recommendation in prose.
var gateHookKeywords = regexp.MustCompile(`(?i)\b(pre-commit hook|spawn gate|completion gate|verification gate|hook|gate)\b`)

// CheckConsequenceSensors verifies that architect investigations declaring gates/hooks
// include an Enforcement Mechanisms table with a Consequence Sensor column.
// Open loops (sensor = "none — open loop") are surfaced as warnings, not errors.
func CheckConsequenceSensors(workspacePath, skill string) *ConsequenceSensorResult {
	result := &ConsequenceSensorResult{Passed: true}

	if skill != "architect" {
		return result
	}

	// Find investigation files in the workspace
	investigations := findInvestigations(workspacePath)
	if len(investigations) == 0 {
		return result
	}

	for _, path := range investigations {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(data)

		// Check if this investigation recommends gates/hooks
		recSection := extractRecommendationsSection(content)
		if recSection == "" {
			continue
		}

		hasGateHookMention := gateHookKeywords.MatchString(recSection)
		if !hasGateHookMention {
			continue
		}

		// Gate/hook mentioned — check for Enforcement Mechanisms table
		tableStart := strings.Index(recSection, "### Enforcement Mechanisms")
		if tableStart == -1 {
			result.Passed = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Investigation %s recommends gates/hooks but has no '### Enforcement Mechanisms' table. "+
					"Add a table with columns: Mechanism | Type | Consequence Sensor",
					filepath.Base(path)))
			result.GatesFailed = append(result.GatesFailed, GateConsequenceSensor)
			continue
		}

		// Parse the table
		tableContent := recSection[tableStart:]
		tableResult := parseEnforcementTable(tableContent)

		if !tableResult.found {
			// Table exists but missing Consequence Sensor column
			result.Passed = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Investigation %s has Enforcement Mechanisms table but missing 'Consequence Sensor' column",
					filepath.Base(path)))
			result.GatesFailed = append(result.GatesFailed, GateConsequenceSensor)
			continue
		}

		// Collect open loops as warnings
		for _, loop := range tableResult.openLoops {
			result.OpenLoops = append(result.OpenLoops, loop)
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Open loop: %s — consider adding a measurement surface", loop))
		}
	}

	return result
}

// findInvestigations returns paths to investigation files in the workspace.
// Checks .kb/investigations/ for design-*.md or *design*.md files.
func findInvestigations(workspacePath string) []string {
	var files []string

	kbDir := filepath.Join(workspacePath, ".kb", "investigations")
	entries, err := os.ReadDir(kbDir)
	if err != nil {
		return files
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".md") && strings.Contains(name, "design") {
			files = append(files, filepath.Join(kbDir, name))
		}
	}

	return files
}

// extractRecommendationsSection extracts the ## Recommendations section from content.
func extractRecommendationsSection(content string) string {
	pattern := regexp.MustCompile(`(?s)## Recommendations\s*\n(.*?)(?:\n## |\z)`)
	matches := pattern.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// enforcementTableResult holds parsed enforcement table data.
type enforcementTableResult struct {
	found     bool     // true if table with Consequence Sensor column exists
	openLoops []string // mechanisms declared as "none — open loop"
}

// parseEnforcementTable parses the Enforcement Mechanisms table.
// Returns found=false if the Consequence Sensor column is missing.
func parseEnforcementTable(tableSection string) enforcementTableResult {
	lines := strings.Split(tableSection, "\n")

	// Find the header row
	headerIdx := -1
	sensorCol := -1
	mechCol := -1

	for i, line := range lines {
		if strings.Contains(line, "|") && strings.Contains(strings.ToLower(line), "mechanism") {
			headerIdx = i
			cols := splitTableRow(line)
			for j, col := range cols {
				lower := strings.ToLower(strings.TrimSpace(col))
				if strings.Contains(lower, "consequence sensor") {
					sensorCol = j
				}
				if strings.Contains(lower, "mechanism") {
					mechCol = j
				}
			}
			break
		}
	}

	if headerIdx == -1 || sensorCol == -1 {
		return enforcementTableResult{found: false}
	}

	result := enforcementTableResult{found: true}

	// Parse data rows (skip header and separator)
	for i := headerIdx + 2; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || !strings.Contains(line, "|") {
			break
		}

		cols := splitTableRow(line)
		if sensorCol >= len(cols) {
			continue
		}

		sensor := strings.TrimSpace(cols[sensorCol])
		mechanism := ""
		if mechCol >= 0 && mechCol < len(cols) {
			mechanism = strings.TrimSpace(cols[mechCol])
		}

		if isOpenLoop(sensor) {
			result.openLoops = append(result.openLoops, mechanism)
		}
	}

	return result
}

// splitTableRow splits a markdown table row by | and trims empty edge cells.
func splitTableRow(line string) []string {
	parts := strings.Split(line, "|")
	// Trim leading/trailing empty parts from | at start and end of line
	if len(parts) > 0 && strings.TrimSpace(parts[0]) == "" {
		parts = parts[1:]
	}
	if len(parts) > 0 && strings.TrimSpace(parts[len(parts)-1]) == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

// isOpenLoop returns true if the sensor field indicates no observation mechanism.
func isOpenLoop(sensor string) bool {
	lower := strings.ToLower(sensor)
	return strings.Contains(lower, "none") && strings.Contains(lower, "open loop")
}
