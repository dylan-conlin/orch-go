// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Tier constants define the spawn tiers.
const (
	TierLight = "light" // Lightweight spawn - skips SYNTHESIS.md requirement
	TierFull  = "full"  // Full spawn - requires SYNTHESIS.md for knowledge externalization
)

// SkillTierDefaults maps skills to their default tier.
// Skills that produce knowledge artifacts default to "full".
// Skills that primarily produce code changes default to "light".
var SkillTierDefaults = map[string]string{
	// Full tier: Investigation-type skills that produce knowledge artifacts
	"investigation":        TierFull,
	"architect":            TierFull,
	"research":             TierFull,
	"codebase-audit":       TierFull,
	"design-session":       TierFull,
	"systematic-debugging": TierFull, // Produces investigation file with findings

	// Light tier: Implementation-focused skills
	"feature-impl":        TierLight,
	"reliability-testing": TierLight,
	"issue-creation":      TierLight, // Creates beads issue, doesn't need synthesis
}

// DefaultTierForSkill returns the default tier for a given skill.
// Returns TierFull for unknown skills (conservative default).
func DefaultTierForSkill(skillName string) string {
	if tier, ok := SkillTierDefaults[skillName]; ok {
		return tier
	}
	return TierFull // Conservative default for unknown skills
}

// SkillIncludesServers maps skills to whether they should include server context.
// UI-focused skills get server info by default to save discovery time.
var SkillIncludesServers = map[string]bool{
	"feature-impl":         true, // Often involves web UI work
	"systematic-debugging": true, // May need to access running servers
	"reliability-testing":  true, // Needs to test live servers
}

// DefaultIncludeServersForSkill returns whether a skill should include server context by default.
// Returns false for unknown skills (conservative default).
func DefaultIncludeServersForSkill(skillName string) bool {
	if include, ok := SkillIncludesServers[skillName]; ok {
		return include
	}
	return false // Don't include for investigation-type skills by default
}

// Config holds configuration for spawning an agent.
type Config struct {
	// Task description
	Task string
	// SkillName is the name of the skill to use (e.g., "investigation", "feature-impl")
	SkillName string
	// Project name (e.g., "orch-go")
	Project string
	// ProjectDir is the absolute path to the project directory
	ProjectDir string
	// WorkspaceName is the generated workspace directory name
	WorkspaceName string

	// Skill content (full SKILL.md content)
	SkillContent string

	// BeadsID is the beads issue ID for lifecycle tracking
	BeadsID string

	// Feature-impl configuration
	Phases     string // Comma-separated phases (e.g., "implementation,validation")
	Mode       string // Implementation mode: "tdd" or "direct"
	Validation string // Validation level: "none", "tests", "smoke-test", "multi-phase"

	// Investigation configuration
	InvestigationType string // Investigation type: "simple", "systems", "feasibility", etc.

	// Model to use for standalone spawns
	Model string

	// MCP server configuration (e.g., "playwright" for browser automation)
	MCP string

	// Tier specifies the spawn tier: "light" or "full"
	// Light tier skips SYNTHESIS.md requirement on completion
	// Full tier requires SYNTHESIS.md for knowledge externalization
	Tier string

	// NoTrack opts out of beads issue tracking (ad-hoc work)
	NoTrack bool

	// SkipArtifactCheck bypasses pre-spawn kb context check
	SkipArtifactCheck bool

	// KBContext is the formatted kb context to include in SPAWN_CONTEXT.md
	KBContext string

	// IncludeServers controls whether server context is included in SPAWN_CONTEXT.md
	// Default is based on skill type (true for UI-focused skills)
	IncludeServers bool

	// ServerContext is the formatted server info to include in SPAWN_CONTEXT.md
	// Populated by GenerateServerContext() if IncludeServers is true
	ServerContext string

	// GapAnalysis contains the results of pre-spawn context gap analysis.
	// Used for surfacing gaps in dashboard and tracking patterns.
	GapAnalysis *GapAnalysis
}

// GenerateWorkspaceName creates a workspace name from skill and task.
// Format: og-{skill-prefix}-{task-slug}-{date}
func GenerateWorkspaceName(skillName, task string) string {
	// Skill prefix mapping (similar to Python's SKILL_PREFIXES)
	prefixes := map[string]string{
		"investigation":        "inv",
		"feature-impl":         "feat",
		"systematic-debugging": "debug",
		"architect":            "arch",
		"codebase-audit":       "audit",
		"research":             "research",
	}

	prefix := "work"
	if p, ok := prefixes[skillName]; ok {
		prefix = p
	}

	// Generate date suffix
	date := time.Now().Format("02Jan")
	date = strings.ToLower(date)

	// Generate task slug from first few meaningful words
	slug := generateSlug(task, 3)

	return fmt.Sprintf("og-%s-%s-%s", prefix, slug, date)
}

// generateSlug extracts meaningful words from text and creates a slug.
func generateSlug(text string, maxWords int) string {
	// Stop words to exclude
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"for": true, "to": true, "in": true, "on": true, "at": true,
		"is": true, "are": true, "was": true, "were": true, "be": true,
		"this": true, "that": true, "with": true, "from": true, "of": true,
	}

	// Extract words (lowercase, alphanumeric only)
	re := regexp.MustCompile(`[a-zA-Z0-9]+`)
	matches := re.FindAllString(strings.ToLower(text), -1)

	var words []string
	for _, word := range matches {
		if !stopWords[word] && len(word) > 1 {
			words = append(words, word)
			if len(words) >= maxWords {
				break
			}
		}
	}

	if len(words) == 0 {
		return "task"
	}

	return strings.Join(words, "-")
}

// WorkspacePath returns the full path to the workspace directory.
func (c *Config) WorkspacePath() string {
	return filepath.Join(c.ProjectDir, ".orch", "workspace", c.WorkspaceName)
}

// ContextFilePath returns the path to SPAWN_CONTEXT.md.
func (c *Config) ContextFilePath() string {
	return filepath.Join(c.WorkspacePath(), "SPAWN_CONTEXT.md")
}
