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
	"gopkg.in/yaml.v3"
)

// Pre-compiled regex patterns for skill_outputs.go
var (
	regexSkillGuidanceHeader = regexp.MustCompile(`(?i)##\s*SKILL\s+GUIDANCE\s*\(([a-z0-9-]+)\)`)
	regexSkillNameField      = regexp.MustCompile(`(?i)(?:\*\*Skill:\*\*|^name:)\s*([a-z0-9-]+)`)
)

// SkillOutput represents a required output from a skill's outputs.required section.
type SkillOutput struct {
	Pattern     string `yaml:"pattern"`
	Description string `yaml:"description"`
}

// SkillOutputs represents the outputs section of a skill.yaml file.
type SkillOutputs struct {
	Required []SkillOutput `yaml:"required"`
}

// SkillManifest represents relevant parts of a skill.yaml file for verification.
type SkillManifest struct {
	Name    string       `yaml:"name"`
	Outputs SkillOutputs `yaml:"outputs"`
}

// SkillOutputResult represents the result of verifying a single skill output.
type SkillOutputResult struct {
	Output       SkillOutput
	Matched      bool     // Whether at least one file matched
	MatchedFiles []string // Files that matched the pattern
	Error        error    // Any error during verification
}

// SkillOutputVerificationResult represents the result of verifying all skill outputs.
type SkillOutputVerificationResult struct {
	SkillName string              // Name of the skill
	Passed    bool                // All required outputs have at least one match
	Results   []SkillOutputResult // Individual results for each output
	Errors    []string            // Error messages for failed outputs
	Warnings  []string            // Informational warnings
}

// ExtractSkillNameFromSpawnContext extracts the skill name from SPAWN_CONTEXT.md.
// Looks for patterns like "## SKILL GUIDANCE (feature-impl)" or "**Skill:** feature-impl".
// If SPAWN_CONTEXT.md is missing or doesn't include a skill marker, it falls back to
// AGENT_MANIFEST.json (canonical spawn metadata).
func ExtractSkillNameFromSpawnContext(workspacePath string) (string, error) {
	spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	file, err := os.Open(spawnContextPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to open SPAWN_CONTEXT.md: %w", err)
		}
	} else {
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()

			// Check for SKILL GUIDANCE pattern first (more reliable)
			if matches := regexSkillGuidanceHeader.FindStringSubmatch(line); len(matches) >= 2 {
				return strings.ToLower(strings.TrimSpace(matches[1])), nil
			}

			// Check for skill name in YAML-like format
			if matches := regexSkillNameField.FindStringSubmatch(line); len(matches) >= 2 {
				return strings.ToLower(strings.TrimSpace(matches[1])), nil
			}
		}

		if scanErr := scanner.Err(); scanErr != nil {
			return "", scanErr
		}
	}

	manifest, manifestErr := spawn.ReadAgentManifest(workspacePath)
	if manifestErr == nil {
		skill := strings.ToLower(strings.TrimSpace(manifest.Skill))
		if skill != "" {
			return skill, nil
		}
	}

	return "", nil
}

// ResolveSkillName extracts the skill name using multiple strategies.
//
// Strategy 1: Direct extraction from workspacePath (SPAWN_CONTEXT.md or AGENT_MANIFEST.json)
// Strategy 2: If workspacePath is a project root (not a workspace dir), search
// .orch/workspace/*/ for a workspace matching beadsID and extract skill from there.
// Strategy 3: Same search using projectDir if different from workspacePath.
//
// This fixes the 87% skill=unknown bug: for headless (non-worktree) spawns,
// orch complete passes the project root as workspacePath (via artifactsDir()),
// but spawn artifacts live in .orch/workspace/{name}/.
func ResolveSkillName(workspacePath, projectDir, beadsID string) string {
	// Strategy 1: direct extraction
	skill, _ := ExtractSkillNameFromSpawnContext(workspacePath)
	if skill != "" {
		return skill
	}

	// Strategy 2+3: find workspace by beadsID
	if beadsID == "" {
		return ""
	}

	for _, searchDir := range []string{workspacePath, projectDir} {
		if searchDir == "" {
			continue
		}
		wsPath := findSpawnWorkspaceByBeadsID(searchDir, beadsID)
		if wsPath != "" {
			skill, _ = ExtractSkillNameFromSpawnContext(wsPath)
			if skill != "" {
				return skill
			}
		}
	}

	return ""
}

// findSpawnWorkspaceByBeadsID searches .orch/workspace/*/.beads_id files to find
// the canonical workspace directory for a given beads ID.
// Returns the workspace path if found, empty string otherwise.
func findSpawnWorkspaceByBeadsID(projectDir, beadsID string) string {
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		wsPath := filepath.Join(workspaceDir, entry.Name())
		idFile := filepath.Join(wsPath, ".beads_id")
		data, err := os.ReadFile(idFile)
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(data)) == beadsID {
			return wsPath
		}
	}

	return ""
}

// FindSkillManifest locates and parses the skill.yaml file for a given skill.
// Searches in standard locations:
// 1. ~/.claude/skills/worker/{skill}/.skillc/skill.yaml
// 2. ~/orch-knowledge/skills/src/worker/{skill}/.skillc/skill.yaml
func FindSkillManifest(skillName string) (*SkillManifest, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Search paths in order of preference
	searchPaths := []string{
		filepath.Join(homeDir, "orch-knowledge", "skills", "src", "worker", skillName, ".skillc", "skill.yaml"),
		filepath.Join(homeDir, ".claude", "skills", "worker", skillName, ".skillc", "skill.yaml"),
		filepath.Join(homeDir, ".claude", "skills", "worker", skillName, "skill.yaml"),
	}

	for _, path := range searchPaths {
		manifest, err := ParseSkillManifest(path)
		if err == nil {
			return manifest, nil
		}
		// Continue searching on error (file not found, parse error, etc.)
	}

	return nil, fmt.Errorf("skill manifest not found for '%s' (searched in %d locations)", skillName, len(searchPaths))
}

// ParseSkillManifest parses a skill.yaml file.
func ParseSkillManifest(path string) (*SkillManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill manifest: %w", err)
	}

	var manifest SkillManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse skill manifest: %w", err)
	}

	return &manifest, nil
}

// VerifySkillOutputs checks if all required skill outputs exist in the project directory.
// Returns a result with Passed=true if all required outputs have at least one matching file.
func VerifySkillOutputs(manifest *SkillManifest, projectDir string, spawnTime time.Time) SkillOutputVerificationResult {
	result := SkillOutputVerificationResult{
		SkillName: manifest.Name,
		Passed:    true,
	}

	// No required outputs means verification passes
	if len(manifest.Outputs.Required) == 0 {
		return result
	}

	for _, output := range manifest.Outputs.Required {
		or := verifySkillOutput(output, projectDir, spawnTime)
		result.Results = append(result.Results, or)

		if or.Error != nil {
			result.Passed = false
			result.Errors = append(result.Errors, fmt.Sprintf("output verification error: %s - %v", output.Pattern, or.Error))
		} else if !or.Matched {
			result.Passed = false
			result.Errors = append(result.Errors, fmt.Sprintf("required output not found: %s (%s)", output.Pattern, output.Description))
		}
	}

	return result
}

// verifySkillOutput checks if a single skill output exists.
func verifySkillOutput(output SkillOutput, projectDir string, spawnTime time.Time) SkillOutputResult {
	result := SkillOutputResult{
		Output: output,
	}

	// Convert pattern to glob
	globPattern := PatternToGlob(output.Pattern)

	// Combine with project directory
	fullPattern := filepath.Join(projectDir, globPattern)

	// Find matching files
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		result.Error = fmt.Errorf("invalid glob pattern: %w", err)
		return result
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

	result.MatchedFiles = matches
	result.Matched = len(matches) > 0

	return result
}

// VerifySkillOutputsForCompletion is a convenience function that extracts the skill name
// from a workspace's SPAWN_CONTEXT.md, finds the skill manifest, and verifies outputs.
// Returns nil result if skill has no outputs.required defined (graceful skip).
func VerifySkillOutputsForCompletion(workspacePath, projectDir string) (*SkillOutputVerificationResult, error) {
	// Extract skill name from SPAWN_CONTEXT.md
	skillName, err := ExtractSkillNameFromSpawnContext(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract skill name: %w", err)
	}
	if skillName == "" {
		// No skill found in spawn context - skip verification
		return nil, nil
	}

	// Find and parse skill manifest
	manifest, err := FindSkillManifest(skillName)
	if err != nil {
		// Skill manifest not found - this is not an error, just skip verification
		// Many skills don't have outputs.required defined
		return nil, nil
	}

	// No required outputs defined - skip verification
	if len(manifest.Outputs.Required) == 0 {
		return nil, nil
	}

	// Read spawn time from workspace using the spawn package
	spawnTime := spawn.ReadSpawnTime(workspacePath)

	// Verify outputs
	result := VerifySkillOutputs(manifest, projectDir, spawnTime)
	return &result, nil
}
