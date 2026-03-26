package spawn

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// getGitBaseline returns the current git commit SHA for the project directory.
// Returns empty string if not in a git repository or if git command fails.
// This is used as the baseline for git-based change detection during verification.
func getGitBaseline(projectDir string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// Not in a git repo or git command failed - return empty
		return ""
	}
	return strings.TrimSpace(string(output))
}

// Pre-compiled regex patterns for context.go
var (
	regexBeadsSectionHeader    = regexp.MustCompile(`(?i)^#+\s*(report\s+(via|to)\s+beads|beads\s+(progress\s+)?tracking)`)
	regexNextSectionHeader     = regexp.MustCompile(`^#{1,6}\s+[A-Z]`)
	regexBeadsReportedCriteria = regexp.MustCompile(`(?i)\*\*Reported\*\*.*bd\s+comment`)
	regexBeadsIDPlaceholder    = regexp.MustCompile(`bd\s+(comment|close|show)\s+<beads-id>`)
	regexMultiNewline          = regexp.MustCompile(`\n{3,}`)
	regexSessionScope          = regexp.MustCompile(`(?mi)^\s*session\s+scope:\s*([^\r\n]+)`)
	// regexBeadsIDInText matches beads-like IDs in text: project-prefix followed by hyphen and digits.
	// Examples: pw-8972, orch-go-1141, pw-123
	regexBeadsIDInText = regexp.MustCompile(`\b([a-z][\w-]*-\d+)\b`)
)

// ParseScopeFromTask extracts a session scope value from a task description.
// Looks for patterns like "SESSION SCOPE: Small" or "Session scope: Large".
// Returns the lowercase first word of the scope value (e.g., "small", "medium", "large"),
// or empty string if no scope is found.
func ParseScopeFromTask(task string) string {
	matches := regexSessionScope.FindStringSubmatch(task)
	if len(matches) < 2 {
		return ""
	}
	scope := strings.TrimSpace(strings.ToLower(matches[1]))
	if scope == "" {
		return ""
	}
	fields := strings.Fields(scope)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// ResolveScope determines the session scope for a spawn.
// Priority: explicit scope parameter > parsed from task > default "medium".
func ResolveScope(explicitScope, task string) string {
	if explicitScope != "" {
		return strings.ToLower(explicitScope)
	}
	if parsed := ParseScopeFromTask(task); parsed != "" {
		return parsed
	}
	return ScopeMedium
}

// CreateScreenshotsDir creates the screenshots/ subdirectory in a workspace.
// This directory is for agent-produced visual artifacts (e.g., UI screenshots for verification).
func CreateScreenshotsDir(workspacePath string) error {
	screenshotsPath := filepath.Join(workspacePath, "screenshots")
	if err := os.MkdirAll(screenshotsPath, 0755); err != nil {
		return fmt.Errorf("failed to create screenshots directory: %w", err)
	}
	return nil
}

// GenerateContext generates the SPAWN_CONTEXT.md content.
func GenerateContext(cfg *Config) (string, error) {
	funcMap := template.FuncMap{
		"subtract": func(a, b int) int { return a - b },
	}
	tmpl, err := template.New("spawn_context").Funcs(funcMap).Parse(SpawnContextTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Generate investigation slug from task
	slug := generateSlug(cfg.Task, 5)

	// Generate server context if enabled
	serverContext := cfg.ServerContext
	if cfg.IncludeServers && serverContext == "" {
		serverContext = GenerateServerContext(cfg.ProjectDir)
	}

	// When SystemPromptFile is set, skill content is injected at system prompt level
	// via --append-system-prompt. Omit it from SPAWN_CONTEXT.md to prevent double-loading.
	var skillContent string
	if cfg.SystemPromptFile == "" {
		// User-level injection: embed skill content in SPAWN_CONTEXT.md (current behavior)
		skillContent = cfg.SkillContent
		if cfg.NoTrack && skillContent != "" {
			skillContent = StripBeadsInstructions(skillContent)
		}
		if skillContent != "" {
			skillContent = ProcessSkillContentTemplate(skillContent, cfg.BeadsID, cfg.Tier)
		}
	}
	// When SystemPromptFile is set, skillContent remains empty — skill content
	// is already written to SKILL_PROMPT.md and will be injected via CLI flag

	// Generate cluster summary for area awareness
	// Detect area from task description or beads issue labels
	clusterSummary := ""
	if detectedArea := DetectAreaFromTask(cfg.Task, cfg.BeadsID, cfg.ProjectDir); detectedArea != "" {
		if summary := GetClusterSummary(detectedArea, cfg.ProjectDir); summary != "" {
			clusterSummary = fmt.Sprintf("\n## AREA CONTEXT: %s\n\n%s\n", detectedArea, summary)
		}
	}

	// Generate governance context for worker agents (not orchestrators)
	governanceContext := GenerateGovernanceContext(cfg.NoTrack)

	data := contextData{
		Task:                  cfg.Task,
		BeadsID:               cfg.BeadsID,
		ProjectDir:            cfg.ProjectDir,
		WorkspaceName:         cfg.WorkspaceName,
		SkillName:             cfg.SkillName,
		SkillContent:          skillContent,
		InvestigationSlug:     slug,
		ProducesInvestigation: DefaultProducesInvestigationForSkill(cfg.SkillName, cfg.Phases),
		HasInjectedModels:     cfg.HasInjectedModels,
		CrossRepoModelDir:     cfg.CrossRepoModelDir,
		Phases:                cfg.Phases,
		Mode:                  cfg.Mode,
		Validation:            cfg.Validation,
		InvestigationType:     cfg.InvestigationType,
		KBContext:             cfg.KBContext,
		ClusterSummary:        clusterSummary,
		ConfigResolution:      FormatResolvedSpawnSettings(cfg.ResolvedSettings),
		Tier:                  cfg.Tier,
		Scope:                 ResolveScope(cfg.Scope, cfg.Task),
		ServerContext:         serverContext,
		NoTrack:               cfg.NoTrack,
		IsBug:                 cfg.IsBug,
		ReproSteps:            cfg.ReproSteps,
		ReworkFeedback:        cfg.ReworkFeedback,
		ReworkNumber:          cfg.ReworkNumber,
		PriorSynthesis:        cfg.PriorSynthesis,
		PriorWorkspace:        cfg.PriorWorkspace,
		HotspotArea:           cfg.HotspotArea,
		HotspotFiles:          cfg.HotspotFiles,
		HotspotDefectClasses:  cfg.HotspotDefectClasses,
		ArchitectDesign:       cfg.ArchitectDesign,
		DesignWorkspace:       cfg.DesignWorkspace,
		DesignMockupPath:      cfg.DesignMockupPath,
		DesignPromptPath:      cfg.DesignPromptPath,
		DesignNotes:           cfg.DesignNotes,
		OrientationFrame:      cfg.OrientationFrame,
		IntentType:            cfg.IntentType,
		PriorCompletions:      cfg.PriorCompletions,
		BrowserAutomation:     cfg.BrowserTool == "playwright-cli",
		Explore:               cfg.Explore,
		ExploreBreadth:        cfg.ExploreBreadth,
		ExploreDepth:          cfg.ExploreDepth,
		ExploreParentSkill:    cfg.ExploreParentSkill,
		ExploreJudgeModel:     cfg.ExploreJudgeModel,
		GovernanceContext:     governanceContext,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// WriteContext writes the SPAWN_CONTEXT.md file to the workspace.
// For orchestrator-type skills (IsOrchestrator=true), it delegates to
// WriteOrchestratorContext which generates ORCHESTRATOR_CONTEXT.md instead.
// For meta-orchestrator skills (IsMetaOrchestrator=true), it delegates to
// WriteMetaOrchestratorContext which generates META_ORCHESTRATOR_CONTEXT.md.
func WriteContext(cfg *Config) error {
	// Route meta-orchestrator spawns to dedicated template (check first, more specific)
	if cfg.IsMetaOrchestrator {
		return WriteMetaOrchestratorContext(cfg)
	}

	// Route orchestrator spawns to dedicated template
	if cfg.IsOrchestrator {
		return WriteOrchestratorContext(cfg)
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		return err
	}

	// Ensure SYNTHESIS.md template exists in the project (only for full tier)
	if cfg.Tier != TierLight {
		if err := EnsureSynthesisTemplate(cfg.ProjectDir); err != nil {
			return fmt.Errorf("failed to ensure synthesis template: %w", err)
		}
	}

	// Ensure PROBE.md template exists in the project (for probe-type spawns)
	if cfg.HasInjectedModels {
		if err := EnsureProbeTemplate(cfg.ProjectDir); err != nil {
			return fmt.Errorf("failed to ensure probe template: %w", err)
		}
	}

	// Create workspace directory
	workspacePath := cfg.WorkspacePath()
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Create screenshots subdirectory for agent-produced visual artifacts
	if err := CreateScreenshotsDir(workspacePath); err != nil {
		return err
	}

	// Write context file
	contextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	if err := os.WriteFile(contextPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write context file: %w", err)
	}

	// Write tier metadata file for orch complete to read
	if err := WriteTier(workspacePath, cfg.Tier); err != nil {
		return fmt.Errorf("failed to write tier file: %w", err)
	}

	// Write spawn time for constraint verification scoping
	// Constraints should only match files created after this spawn, not pre-existing files
	if err := WriteSpawnTime(workspacePath, time.Now()); err != nil {
		return fmt.Errorf("failed to write spawn time file: %w", err)
	}

	// Write beads ID file for workspace lookup during orch complete
	if cfg.BeadsID != "" {
		beadsIDPath := filepath.Join(workspacePath, ".beads_id")
		if err := os.WriteFile(beadsIDPath, []byte(cfg.BeadsID), 0644); err != nil {
			return fmt.Errorf("failed to write beads ID file: %w", err)
		}
	}

	// Write spawn mode file for orch complete to be mode-aware
	if cfg.SpawnMode != "" {
		spawnModePath := filepath.Join(workspacePath, ".spawn_mode")
		if err := os.WriteFile(spawnModePath, []byte(cfg.SpawnMode), 0644); err != nil {
			return fmt.Errorf("failed to write spawn mode file: %w", err)
		}
	}

	// Write agent manifest JSON for canonical agent identity and spawn-time metadata
	// This provides a single source of truth for git-based scoping and verification gates
	spawnTime := time.Now()
	// Build routing impact report from resolved settings
	var routingImpact *RoutingImpact
	if cfg.ResolvedSettings.Backend.Value != "" {
		ri := BuildRoutingImpact(cfg.ResolvedSettings)
		if ri.Triggered {
			routingImpact = &ri
		}
	}

	manifest := AgentManifest{
		WorkspaceName: cfg.WorkspaceName,
		Skill:         cfg.SkillName,
		BeadsID:       cfg.BeadsID,
		ProjectDir:    cfg.ProjectDir,
		GitBaseline:   getGitBaseline(cfg.ProjectDir),
		SpawnTime:     spawnTime.Format(time.RFC3339),
		Tier:          cfg.Tier,
		SpawnMode:     cfg.SpawnMode,
		Model:         cfg.Model,
		VerifyLevel:   cfg.VerifyLevel,
		ReviewTier:    cfg.ReviewTier,
		RoutingImpact: routingImpact,
	}
	if err := WriteAgentManifest(workspacePath, manifest); err != nil {
		return fmt.Errorf("failed to write agent manifest: %w", err)
	}

	// Write prior workspace reference for rework spawns (if provided)
	if cfg.PriorWorkspace != "" {
		priorWorkspacePath := filepath.Join(workspacePath, ".prior_workspace")
		content := cfg.PriorWorkspace
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if err := os.WriteFile(priorWorkspacePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write prior workspace file: %w", err)
		}
	}

	return nil
}

// EnsureProbeTemplate ensures the PROBE.md template exists in the project.
// If the project doesn't have .orch/templates/PROBE.md, it creates one from
// the DefaultProbeTemplate in probes.go.
func EnsureProbeTemplate(projectDir string) error {
	templatesDir := filepath.Join(projectDir, ".orch", "templates")
	templatePath := filepath.Join(templatesDir, "PROBE.md")

	// Check if template already exists
	if _, err := os.Stat(templatePath); err == nil {
		return nil // Template exists, nothing to do
	}

	// Create templates directory if needed
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	// Write the default template
	if err := os.WriteFile(templatePath, []byte(DefaultProbeTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write probe template: %w", err)
	}

	return nil
}

// extractProjectPrefix extracts the project prefix from a beads ID.
// Given "pw-8972", returns "pw". Given "orch-go-1141", returns "orch-go".
// The prefix is everything before the final hyphen-number sequence.
// MinimalPrompt generates the minimal prompt for opencode run.
// For meta-orchestrator skills, it points to META_ORCHESTRATOR_CONTEXT.md.
// For orchestrator-type skills, it points to ORCHESTRATOR_CONTEXT.md instead.
func MinimalPrompt(cfg *Config) string {
	if cfg.IsMetaOrchestrator {
		return MinimalMetaOrchestratorPrompt(cfg)
	}
	if cfg.IsOrchestrator {
		return MinimalOrchestratorPrompt(cfg)
	}
	return fmt.Sprintf(
		"Read your spawn context from %s/.orch/workspace/%s/SPAWN_CONTEXT.md. The instructions in SPAWN_CONTEXT.md are mandatory protocol. Your first tool call may read SPAWN_CONTEXT.md; immediately after reading, report Phase: Planning via the bd comment command specified there. Do not end a turn with narrative unless you are BLOCKED, have a QUESTION, or are COMPLETE. Continue making tool calls until all required deliverables (including Phase: Complete reporting and any required files) are done. Begin the task.",
		cfg.ProjectDir,
		cfg.WorkspaceName,
	)
}

// GenerateInvestigationSlug creates a slug for the investigation file name.
func GenerateInvestigationSlug(task string) string {
	slug := generateSlug(task, 5)
	date := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s-inv-%s", date, slug)
}

