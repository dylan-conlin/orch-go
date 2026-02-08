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

// SkillTierDefaults maps skills to their default tier.
// Skills that produce knowledge artifacts default to "full".
// Skills that primarily produce code changes default to "light".
var SkillTierDefaults = map[string]string{
	// Full tier: Investigation-type skills that produce knowledge artifacts
	"investigation":  TierFull,
	"architect":      TierFull,
	"research":       TierFull,
	"codebase-audit": TierFull,
	"design-session": TierFull,

	// Light tier: Implementation-focused skills (code changes auto-complete)
	"feature-impl":         TierLight,
	"systematic-debugging": TierLight, // Code-focused debugging, auto-completes
	"reliability-testing":  TierLight,
	"issue-creation":       TierLight, // Creates beads issue, doesn't need synthesis
}

// DefaultTierForSkill returns the default tier for a given skill.
// Returns TierFull for unknown skills (conservative default).
func DefaultTierForSkill(skillName string) string {
	if tier, ok := SkillTierDefaults[skillName]; ok {
		return tier
	}
	return TierFull // Conservative default for unknown skills
}

// SkillVariantDefaults maps skills to their default extended thinking variant.
// Extended thinking enables reasoning tokens for complex tasks.
// Variants: "high" (16k tokens), "max" (32k tokens), "" (disabled).
var SkillVariantDefaults = map[string]string{
	// High variant (16k tokens): Complex reasoning tasks
	"investigation":        "high",
	"feature-impl":         "high",
	"systematic-debugging": "high",
	"reliability-testing":  "high",
	"research":             "high",

	// Max variant (32k tokens): Deep strategic reasoning
	"architect":      "max",
	"design-session": "max",

	// No extended thinking: Simple tasks
	// (unlisted skills default to empty string)
}

// DefaultVariantForSkill returns the default extended thinking variant for a skill.
// Returns empty string for unknown skills (no extended thinking).
func DefaultVariantForSkill(skillName string) string {
	if variant, ok := SkillVariantDefaults[skillName]; ok {
		return variant
	}
	return "" // No extended thinking for unknown skills
}

// DefaultVariantForRole returns the default extended thinking variant based on role flags.
// Orchestrators and meta-orchestrators use "high" for strategic decisions.
// Workers default to skill-based variant.
func DefaultVariantForRole(isOrchestrator, isMetaOrchestrator bool, skillName string) string {
	if isMetaOrchestrator || isOrchestrator {
		return "high" // Orchestrators need extended thinking for strategic decisions
	}
	return DefaultVariantForSkill(skillName)
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

// SkillRequiresInvestigation maps skills to whether they mandate investigation files.
// Investigation-type skills produce knowledge artifacts and require investigation files.
// Implementation-focused skills produce code changes and do NOT require investigation files.
var SkillRequiresInvestigation = map[string]bool{
	// Skills that require investigation files (knowledge artifact producers)
	"investigation":  true,
	"architect":      true,
	"research":       true,
	"codebase-audit": true,

	// Skills that do NOT require investigation files (implementation-focused)
	"feature-impl":         false,
	"systematic-debugging": false,
	"reliability-testing":  false,
	"issue-creation":       false,
	"design-session":       false, // Produces design artifacts, not investigation files
}

// IsInvestigationSkill returns whether a skill requires investigation files.
// Returns true for investigation-type skills (investigation, architect, research).
// Returns false for implementation-focused skills (feature-impl, systematic-debugging).
// Unknown skills default to false (conservative - don't mandate investigation files).
func IsInvestigationSkill(skillName string) bool {
	if requires, ok := SkillRequiresInvestigation[skillName]; ok {
		return requires
	}
	return false // Conservative default: don't mandate investigation files for unknown skills
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
	Validation string // Validation level: "none", "tests", "integration", "smoke-test", "multi-phase"

	// Investigation configuration
	InvestigationType string // Investigation type: "simple", "systems", "feasibility", etc.

	// Model to use for standalone spawns
	Model string

	// Variant specifies extended thinking mode: "high" (16k tokens), "max" (32k tokens), or "" (disabled)
	Variant string

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

	// HasInjectedModels indicates whether model content (summary, invariants, failures)
	// was injected into the KB context. When true, agents see probe guidance directing
	// them to produce lightweight probes (~30-50 lines) instead of full investigations.
	HasInjectedModels bool

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

	// IsInfrastructureTouching indicates whether this spawn touches infrastructure code.
	// When true, SPAWN_CONTEXT.md includes a required resource lifecycle audit directive.
	IsInfrastructureTouching bool

	// IsOrchestrator indicates whether the skill is an orchestrator-type skill.
	// Orchestrator skills (skill-type: policy/orchestrator) have different defaults:
	// - Default to tmux mode (visible interaction instead of headless)
	// - Use ORCHESTRATOR_CONTEXT.md template instead of SPAWN_CONTEXT.md
	// - Completion verification via SYNTHESIS.md (like workers, for consistency)
	// - No beads tracking (orchestrators manage sessions, not issues)
	IsOrchestrator bool

	// IsMetaOrchestrator indicates whether the skill is the meta-orchestrator skill.
	// Meta-orchestrators (skill-name: meta-orchestrator) have different framing:
	// - Use META_ORCHESTRATOR_CONTEXT.md template instead of ORCHESTRATOR_CONTEXT.md
	// - Interactive session framing ("managing orchestrator sessions" not "work toward goal")
	// - No SYNTHESIS.md requirement (stay interactive and available)
	// - First action: check orch status for sessions to complete/review
	IsMetaOrchestrator bool

	// SessionGoal is the high-level goal for orchestrator sessions.
	// Used in ORCHESTRATOR_CONTEXT.md to set session focus.
	// Only applicable when IsOrchestrator is true.
	SessionGoal string

	// HasSynthesisTemplate indicates whether a SYNTHESIS.template.md
	// was copied to the workspace. When true, the ORCHESTRATOR_CONTEXT.md will
	// instruct the agent to use it as the structure for their SYNTHESIS.md.
	// Set by WriteOrchestratorContext based on template availability.
	HasSynthesisTemplate bool

	// RegisteredProjects is the formatted list of registered projects to include
	// in orchestrator context templates. Populated by GenerateRegisteredProjectsContext()
	// for orchestrator and meta-orchestrator spawns to enable cross-project work.
	RegisteredProjects string

	// PriorSynthesisPath is the path to the most recent prior orchestrator's
	// SYNTHESIS.md. When set, the new meta-orchestrator session context will
	// include a reference to this file so the agent can pick up context from the
	// previous session. Only used for meta-orchestrator spawns.
	PriorSynthesisPath string

	// UsageInfo contains the current account usage at spawn time.
	// Used for telemetry and monitoring. May be nil if usage check failed.
	UsageInfo *UsageInfo

	// SpawnMode specifies the spawn backend: "opencode" or "claude"
	SpawnMode string

	// Design handoff fields (for ui-design-session → feature-impl handoff)
	// DesignWorkspace is the workspace name from a prior ui-design-session spawn
	DesignWorkspace string
	// DesignMockupPath is the path to the approved mockup screenshot
	DesignMockupPath string
	// DesignPromptPath is the path to the prompt that generated the mockup
	DesignPromptPath string
	// DesignNotes are notes from the design session SYNTHESIS.md
	DesignNotes string

	// IssueTitle is the beads issue title (from existing issue or newly created).
	// Populated during spawn for state DB recording.
	IssueTitle string

	// IssueType is the beads issue type (e.g., "task", "bug", "feature").
	// Populated during spawn for state DB recording.
	IssueType string

	// IssuePriority is the beads issue priority (e.g., 1, 2, 3).
	// Populated during spawn for state DB recording.
	IssuePriority int

	// IssueComments contains comments from the beads issue (if --issue was provided).
	// These are orchestrator notes added after issue creation that provide additional
	// context, clarifications, or guidance for the spawned agent.
	IssueComments []IssueComment

	// DaemonDriven indicates whether this spawn was initiated by the daemon.
	// When true, skip focus-stealing behaviors like tmux select-window
	// to avoid interrupting the orchestrator's workflow.
	DaemonDriven bool

	// FailureContext contains post-completion failure information when this is a rework spawn.
	// Populated from POST-COMPLETION-FAILURE comments on the beads issue.
	// When IsRework is true, the failure context should be prominently displayed.
	FailureContext *FailureContext
}

// IssueComment represents a comment on a beads issue.
// Used to pass orchestrator notes to spawned agents via SPAWN_CONTEXT.md.
type IssueComment struct {
	// Author is the comment author (e.g., "orchestrator", "dylan")
	Author string
	// Text is the comment content
	Text string
	// CreatedAt is when the comment was created (ISO 8601 format)
	CreatedAt string
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

// FailureType constants define the types of post-completion failures.
const (
	// FailureTypeVerification indicates agent claimed success but didnt properly verify.
	FailureTypeVerification = "verification"
	// FailureTypeImplementation indicates the code has a bug.
	FailureTypeImplementation = "implementation"
	// FailureTypeSpec indicates the spec was wrong or incomplete.
	FailureTypeSpec = "spec"
	// FailureTypeIntegration indicates the feature works in isolation but fails in context.
	FailureTypeIntegration = "integration"
)

// FailureContext contains information about a post-completion failure.
// This is extracted from POST-COMPLETION-FAILURE comments on reopened issues.
// When present, it indicates this is a rework attempt and provides context
// about what went wrong in the previous attempt.
type FailureContext struct {
	// FailureType categorizes the failure (verification, implementation, spec, integration).
	FailureType string
	// Description is the human-readable description of what failed.
	Description string
	// PriorAttemptContext is any additional context from the prior attempt.
	PriorAttemptContext string
	// SuggestedSkill is the recommended skill based on failure type.
	SuggestedSkill string
	// IsRework indicates this is a rework spawn (has POST-COMPLETION-FAILURE comment).
	IsRework bool
}

// SuggestSkillForFailure returns the recommended skill based on failure type.
// This helps route rework to the appropriate skill based on what went wrong.
func SuggestSkillForFailure(failureType string) string {
	switch failureType {
	case FailureTypeVerification:
		// Verification failure = agent didnt properly verify
		// Use reliability-testing to enforce proper verification
		return "reliability-testing"
	case FailureTypeImplementation:
		// Implementation bug = code doesnt work
		// Use systematic-debugging to find and fix the bug
		return "systematic-debugging"
	case FailureTypeSpec:
		// Spec was wrong = need investigation first
		// Use investigation to refine the spec
		return "investigation"
	case FailureTypeIntegration:
		// Integration failure = works in isolation
		// Use reliability-testing to test in full context
		return "reliability-testing"
	default:
		// Unknown failure type = systematic debugging as safe default
		return "systematic-debugging"
	}
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
