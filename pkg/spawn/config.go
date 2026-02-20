// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Pre-compiled regex patterns for config.go
var regexAlphanumeric = regexp.MustCompile(`[a-zA-Z0-9]+`)

// Tier constants define the spawn tiers.
const (
	TierLight = "light" // Lightweight spawn - skips SYNTHESIS.md requirement
	TierFull  = "full"  // Full spawn - requires SYNTHESIS.md for knowledge externalization
)

// Scope constants define session scope levels.
const (
	ScopeSmall  = "small"
	ScopeMedium = "medium"
	ScopeLarge  = "large"
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

// SkillProducesInvestigation maps skills to whether they produce investigation artifacts
// as their primary deliverable. Only these skills get investigation file setup in SPAWN_CONTEXT.md.
var SkillProducesInvestigation = map[string]bool{
	"investigation":       true,
	"architect":           true,
	"research":            true,
	"codebase-audit":      true,
	"reliability-testing": true,
}

// DefaultProducesInvestigationForSkill returns whether a skill should produce investigation artifacts.
// For feature-impl, this depends on whether the phases include "investigation".
// Returns false for unknown skills (most skills don't produce investigations).
func DefaultProducesInvestigationForSkill(skillName string, phases string) bool {
	if produces, ok := SkillProducesInvestigation[skillName]; ok {
		return produces
	}
	// feature-impl produces investigation only when investigation phase is configured
	if skillName == "feature-impl" && phases != "" {
		for _, phase := range strings.Split(phases, ",") {
			if strings.TrimSpace(phase) == "investigation" {
				return true
			}
		}
	}
	return false
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

	// ResolvedSettings captures spawn settings with provenance for SPAWN_CONTEXT.md
	ResolvedSettings ResolvedSpawnSettings

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

	// HasInjectedModels indicates whether model content was injected into KB context.
	// When true, the spawn will use probe template instead of investigation template.
	// This implements the Feb 8 decision: model exists → probe, no model → investigation.
	HasInjectedModels bool

	// PrimaryModelPath is the file path to the primary model when HasInjectedModels is true.
	// Used to determine probe file location (.kb/models/{model-name}/probes/).
	PrimaryModelPath string

	// IncludeServers controls whether server context is included in SPAWN_CONTEXT.md
	// Default is based on skill type (true for UI-focused skills)
	IncludeServers bool

	// ServerContext is the formatted server info to include in SPAWN_CONTEXT.md
	// Populated by GenerateServerContext() if IncludeServers is true
	ServerContext string

	// GapAnalysis contains the results of pre-spawn context gap analysis.
	// Used for surfacing gaps in dashboard and tracking patterns.
	GapAnalysis *GapAnalysis

	// IsBug indicates whether the associated beads issue is a bug type.
	// When true, ReproSteps will be included in SPAWN_CONTEXT.md.
	IsBug bool

	// ReproSteps contains reproduction steps extracted from a bug issue.
	// Included in SPAWN_CONTEXT.md to help agents understand how to verify the fix.
	ReproSteps string

	// Rework fields provide context for rework spawns.
	ReworkFeedback string // Orchestrator's rework instructions
	ReworkNumber   int    // Rework attempt number
	PriorSynthesis string // TLDR + Delta from prior SYNTHESIS.md
	PriorWorkspace string // Path to archived prior workspace

	// IsOrchestrator indicates whether the skill is an orchestrator-type skill.
	// Orchestrator skills (skill-type: policy/orchestrator) have different defaults:
	// - Default to tmux mode (visible interaction instead of headless)
	// - Use ORCHESTRATOR_CONTEXT.md template instead of SPAWN_CONTEXT.md
	// - Different completion verification (SESSION_HANDOFF.md instead of SYNTHESIS.md)
	// - No beads tracking (orchestrators manage sessions, not issues)
	IsOrchestrator bool

	// IsMetaOrchestrator indicates whether the skill is the meta-orchestrator skill.
	// Meta-orchestrators (skill-name: meta-orchestrator) have different framing:
	// - Use META_ORCHESTRATOR_CONTEXT.md template instead of ORCHESTRATOR_CONTEXT.md
	// - Interactive session framing ("managing orchestrator sessions" not "work toward goal")
	// - No SESSION_HANDOFF.md requirement (stay interactive and available)
	// - First action: check orch status for sessions to complete/review
	IsMetaOrchestrator bool

	// SessionGoal is the high-level goal for orchestrator sessions.
	// Used in ORCHESTRATOR_CONTEXT.md to set session focus.
	// Only applicable when IsOrchestrator is true.
	SessionGoal string

	// HasSessionHandoffTemplate indicates whether a SESSION_HANDOFF.template.md
	// was copied to the workspace. When true, the ORCHESTRATOR_CONTEXT.md will
	// instruct the agent to use it as the structure for their SESSION_HANDOFF.md.
	// Set by WriteOrchestratorContext based on template availability.
	HasSessionHandoffTemplate bool

	// RegisteredProjects is the formatted list of registered projects to include
	// in orchestrator context templates. Populated by GenerateRegisteredProjectsContext()
	// for orchestrator and meta-orchestrator spawns to enable cross-project work.
	RegisteredProjects string

	// PriorHandoffPath is the path to the most recent prior meta-orchestrator's
	// SESSION_HANDOFF.md. When set, the new meta-orchestrator session context will
	// include a reference to this file so the agent can pick up context from the
	// previous session. Only used for meta-orchestrator spawns.
	PriorHandoffPath string

	// UsageInfo contains the current account usage at spawn time.
	// Used for telemetry and monitoring. May be nil if usage check failed.
	UsageInfo *UsageInfo

	// SpawnMode specifies the spawn backend: "opencode" or "claude"
	SpawnMode string

	// Scope specifies the session scope: "small", "medium", or "large"
	// Parsed from task description or set via --scope flag
	// Affects checkpoint recommendations in SPAWN_CONTEXT.md
	Scope string

	// Design handoff fields (for ui-design-session → feature-impl handoff)
	// DesignWorkspace is the workspace name from a prior ui-design-session spawn
	DesignWorkspace string
	// DesignMockupPath is the path to the approved mockup screenshot
	DesignMockupPath string
	// DesignPromptPath is the path to the prompt that generated the mockup
	DesignPromptPath string
	// DesignNotes are notes from the design session SYNTHESIS.md
	DesignNotes string
}

// UsageInfo contains account usage data at spawn time.
// This is a simplified copy of account.CapacityInfo for spawn context.
type UsageInfo struct {
	// FiveHourUsed is the 5-hour session utilization (0-100).
	FiveHourUsed float64
	// SevenDayUsed is the weekly utilization (0-100).
	SevenDayUsed float64
	// AccountEmail is the account email (if available).
	AccountEmail string
	// AutoSwitched indicates if account was auto-switched before spawn.
	AutoSwitched bool
	// SwitchReason explains why account was switched.
	SwitchReason string
}

// WorkspaceNameOptions provides optional configuration for workspace name generation.
type WorkspaceNameOptions struct {
	// IsMetaOrchestrator indicates this is a meta-orchestrator spawn.
	// When true, the workspace name will use "meta-" prefix instead of project prefix.
	IsMetaOrchestrator bool

	// IsOrchestrator indicates this is an orchestrator-type skill spawn.
	// When true, the workspace name will use "orch" as the skill prefix
	// instead of "work" for visual distinction from worker workspaces.
	IsOrchestrator bool
}

// GenerateWorkspaceName creates a workspace name from project, skill, and task.
// Format: {project-prefix}-{skill-prefix}-{task-slug}-{date}-{unique}
// The project prefix is derived from the project name (first 2 chars of each word,
// or first 2 chars if single word). Examples: "orch-go" -> "og", "price-watch" -> "pw"
// For meta-orchestrator spawns (via opts), the prefix is "meta-" instead.
// For orchestrator spawns (via opts), the skill prefix is "orch" for visual distinction.
// The unique suffix is a 4-character hex string to prevent collisions between sessions
// spawned on the same day with similar tasks.
func GenerateWorkspaceName(projectName, skillName, task string, opts ...WorkspaceNameOptions) string {
	// Check for options
	var isMetaOrchestrator, isOrchestrator bool
	if len(opts) > 0 {
		isMetaOrchestrator = opts[0].IsMetaOrchestrator
		isOrchestrator = opts[0].IsOrchestrator
	}

	// Generate project prefix
	var projectPrefix string
	if isMetaOrchestrator {
		projectPrefix = "meta"
	} else {
		projectPrefix = generateProjectPrefix(projectName)
	}
	// Skill prefix mapping (similar to Python's SKILL_PREFIXES)
	prefixes := map[string]string{
		"investigation":        "inv",
		"feature-impl":         "feat",
		"systematic-debugging": "debug",
		"architect":            "arch",
		"codebase-audit":       "audit",
		"research":             "research",
	}

	// Default prefix depends on whether this is an orchestrator spawn
	prefix := "work"
	if isOrchestrator || isMetaOrchestrator {
		prefix = "orch" // Use "orch" for orchestrator-type skills for visual distinction
	}
	if p, ok := prefixes[skillName]; ok {
		prefix = p
	}

	// Generate date suffix
	date := time.Now().Format("02Jan")
	date = strings.ToLower(date)

	// Generate task slug from first few meaningful words
	slug := generateSlug(task, 3)

	// Generate unique suffix to prevent workspace name collisions
	// Uses 2 random bytes (4 hex chars) for sufficient uniqueness within a day
	unique := generateUniqueSuffix()

	return fmt.Sprintf("%s-%s-%s-%s-%s", projectPrefix, prefix, slug, date, unique)
}

// generateUniqueSuffix creates a 4-character hex string for workspace name uniqueness.
// This prevents collisions when spawning multiple sessions on the same day with similar tasks.
func generateUniqueSuffix() string {
	b := make([]byte, 2)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based suffix if crypto/rand fails
		// This shouldn't happen in practice but ensures we never return empty
		return fmt.Sprintf("%04x", time.Now().UnixNano()&0xFFFF)
	}
	return hex.EncodeToString(b)
}

// generateSlug extracts meaningful words from text and creates a slug.
func generateSlug(text string, maxWords int) string {
	// Stop words to exclude - including question words, auxiliary verbs, and vague adjectives
	// These rarely add meaning to workspace names
	stopWords := map[string]bool{
		// Articles and conjunctions
		"the": true, "a": true, "an": true, "and": true, "or": true,
		// Prepositions
		"for": true, "to": true, "in": true, "on": true, "at": true,
		"with": true, "from": true, "of": true, "by": true, "as": true,
		// Be verbs
		"is": true, "are": true, "was": true, "were": true, "be": true,
		"been": true, "being": true,
		// Demonstratives
		"this": true, "that": true, "these": true, "those": true,
		// Question words (rarely meaningful in workspace names)
		"how": true, "what": true, "when": true, "where": true, "why": true, "which": true,
		// Auxiliary/modal verbs
		"should": true, "could": true, "would": true, "can": true, "will": true,
		"may": true, "might": true, "must": true, "shall": true,
		"do": true, "does": true, "did": true, "done": true,
		"have": true, "has": true, "had": true,
		// Vague adjectives/adverbs that don't add specificity
		"better": true, "best": true, "good": true, "bad": true, "new": true,
		"more": true, "less": true, "very": true, "really": true, "just": true,
		// Common task description filler words
		"need": true, "needs": true, "want": true, "wants": true,
		"make": true, "makes": true, "get": true, "gets": true,
		"use": true, "uses": true, "using": true,
		"some": true, "any": true, "all": true, "each": true, "every": true,
		// Pronouns
		"it": true, "its": true, "we": true, "our": true, "they": true, "their": true,
	}

	// Extract words (lowercase, alphanumeric only)
	matches := regexAlphanumeric.FindAllString(strings.ToLower(text), -1)

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

// generateProjectPrefix creates a short prefix from a project name.
// For two-part names like "orch-go" or "price-watch", takes first letter of each part.
// For single-word names like "myproject", takes first two characters.
// For three or more parts, takes first two characters of each part.
// Examples:
//   - "orch-go" -> "og" (2 parts: first letter of each)
//   - "price-watch" -> "pw" (2 parts: first letter of each)
//   - "myproject" -> "my" (1 part: first two chars)
//   - "my-cool-project" -> "mycopr" (3 parts: first two chars of each)
func generateProjectPrefix(projectName string) string {
	if projectName == "" {
		return "og" // Fallback for empty project name
	}

	// Split on hyphens and underscores
	parts := strings.FieldsFunc(projectName, func(r rune) bool {
		return r == '-' || r == '_'
	})

	if len(parts) == 0 {
		return "og" // Fallback
	}

	var prefix strings.Builder

	// For exactly 2 parts, take first letter of each (creates 2-char prefix like "og", "pw")
	// This is the common case for hyphenated project names
	if len(parts) == 2 {
		for _, part := range parts {
			if len(part) >= 1 {
				prefix.WriteString(strings.ToLower(string(part[0])))
			}
		}
	} else {
		// For 1 part or 3+ parts, take first 2 characters of each part
		for _, part := range parts {
			if len(part) >= 2 {
				prefix.WriteString(strings.ToLower(part[:2]))
			} else if len(part) == 1 {
				prefix.WriteString(strings.ToLower(part))
			}
		}
	}

	result := prefix.String()
	if result == "" {
		return "og" // Fallback
	}
	return result
}

// WorkspacePath returns the full path to the workspace directory.
func (c *Config) WorkspacePath() string {
	return filepath.Join(c.ProjectDir, ".orch", "workspace", c.WorkspaceName)
}

// ContextFilePath returns the path to the context file.
// For meta-orchestrator skills, it points to META_ORCHESTRATOR_CONTEXT.md.
// For orchestrator skills, it points to ORCHESTRATOR_CONTEXT.md.
// For all other skills, it points to SPAWN_CONTEXT.md.
func (c *Config) ContextFilePath() string {
	var filename string
	switch {
	case c.IsMetaOrchestrator:
		filename = "META_ORCHESTRATOR_CONTEXT.md"
	case c.IsOrchestrator:
		filename = "ORCHESTRATOR_CONTEXT.md"
	default:
		filename = "SPAWN_CONTEXT.md"
	}
	return filepath.Join(c.WorkspacePath(), filename)
}
