// Package verify provides verification helpers for agent completion.
package verify

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// Pre-compiled regex patterns for skill name extraction.
var (
	regexSkillGuidanceHeader = regexp.MustCompile(`(?i)##\s*SKILL\s+GUIDANCE\s*\(([a-z0-9-]+)\)`)
	regexSkillNameField      = regexp.MustCompile(`(?i)(?:\*\*Skill:\*\*|^name:)\s*([a-z0-9-]+)`)
)

// ExtractSkillNameFromSpawnContext extracts the skill name from SPAWN_CONTEXT.md.
// Looks for patterns like "## SKILL GUIDANCE (feature-impl)" or "**Skill:** feature-impl".
func ExtractSkillNameFromSpawnContext(workspacePath string) (string, error) {
	spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	file, err := os.Open(spawnContextPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No SPAWN_CONTEXT.md
		}
		return "", fmt.Errorf("failed to open SPAWN_CONTEXT.md: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if matches := regexSkillGuidanceHeader.FindStringSubmatch(line); len(matches) >= 2 {
			return matches[1], nil
		}
		if matches := regexSkillNameField.FindStringSubmatch(line); len(matches) >= 2 {
			return matches[1], nil
		}
	}
	return "", scanner.Err()
}
