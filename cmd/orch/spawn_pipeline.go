// Package main provides pipeline phases for the spawn command.
// runSpawnWithSkillInternal is decomposed into sequential phases for readability and testability.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/skills"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	statedb "github.com/dylan-conlin/orch-go/pkg/state"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// spawnPipeline holds the accumulated state across pipeline phases.
// Each phase reads and writes to this struct, replacing the long chain
// of local variables in the original monolithic function.
type spawnPipeline struct {
	// Dependencies
	client opencode.ClientInterface

	// Inputs (from caller)
	serverURL    string
	skillName    string
	task         string
	inline       bool
	headless     bool
	tmux         bool
	attach       bool
	daemonDriven bool

	// Phase 1: Pre-flight validation
	resolvedModel    model.ModelSpec
	usageCheckResult *UsageCheckResult

	// Phase 2: Project resolution
	projectDir  string
	projectName string
	projCfg     *config.Config
	globalCfg   *userconfig.Config

	// Phase 3: Skill loading
	isOrchestrator     bool
	isMetaOrchestrator bool
	workspaceName      string
	skillContent       string

	// Phase 4: Issue tracking
	beadsID                  string
	skipBeadsForOrchestrator bool

	// Phase 5: Context gathering
	kbContext   string
	gapAnalysis *spawn.GapAnalysis

	// Phase 6: Config building
	tier                     string
	isBug                    bool
	reproSteps               string
	usageInfo                *spawn.UsageInfo
	spawnBackend             string
	variant                  string
	designMockupPath         string
	designPromptPath         string
	designNotes              string
	issueTitle               string
	issueType                string
	issuePriority            int
	isInfrastructureTouching bool
	cfg                      *spawn.Config
}

// newSpawnPipeline creates a pipeline with inputs from the caller.
func newSpawnPipeline(serverURL, skillName, task string, inline, headless, tmux, attach, daemonDriven bool) *spawnPipeline {
	return &spawnPipeline{
		client:       opencode.NewClient(serverURL),
		serverURL:    serverURL,
		skillName:    skillName,
		task:         task,
		inline:       inline,
		headless:     headless,
		tmux:         tmux,
		attach:       attach,
		daemonDriven: daemonDriven,
	}
}

// runPreFlightValidation checks triage bypass, concurrency limits, rate limits,
// and hotspot analysis before proceeding with the spawn.
func (p *spawnPipeline) runPreFlightValidation() error {
	// Check for --bypass-triage flag (required for manual spawns)
	// Daemon-driven spawns skip this check (issue already triaged)
	if !p.daemonDriven && !spawnBypassTriage {
		return showTriageBypassRequired(p.skillName, p.task)
	}

	// Log the triage bypass for Phase 2 review (only for manual bypasses, not daemon-driven)
	if !p.daemonDriven && spawnBypassTriage {
		logTriageBypass(p.skillName, p.task)
	}

	// Check concurrency limit before spawning
	if err := checkConcurrencyLimit(); err != nil {
		return err
	}

	// Resolve model early to check if Anthropic rate limit applies
	p.resolvedModel = model.Resolve(spawnModel)

	// Proactive rate limit monitoring: warn at 80%, block at 95%
	// Only applies to Anthropic models — non-Anthropic providers have their own limits
	if p.resolvedModel.IsAnthropic() {
		var usageErr error
		p.usageCheckResult, usageErr = checkUsageBeforeSpawn()
		if usageErr != nil {
			return usageErr
		}
	} else {
		p.usageCheckResult = &UsageCheckResult{}
		fmt.Fprintf(os.Stderr, "ℹ️  Non-Anthropic model (%s) — skipping Anthropic rate limit check\n", p.resolvedModel.Format())
	}

	// Get project directory early for hotspot check
	var preCheckDir string
	if spawnWorkdir != "" {
		if absPath, err := filepath.Abs(spawnWorkdir); err == nil {
			preCheckDir = absPath
		}
	} else {
		preCheckDir, _ = currentProjectDir()
	}

	// STRATEGIC-FIRST ORCHESTRATION: Check for hotspots in task target area
	if preCheckDir != "" {
		if hotspotResult, err := RunHotspotCheckForSpawn(preCheckDir, p.task); err == nil && hotspotResult != nil {
			isStrategicSkill := p.skillName == "architect"

			if !p.daemonDriven && !spawnForce && !isStrategicSkill {
				fmt.Fprint(os.Stderr, hotspotResult.Warning)
				fmt.Fprintln(os.Stderr, "💡 Consider: spawn architect first for strategic approach in hotspot area")
				fmt.Fprintln(os.Stderr, "")
			} else if p.daemonDriven {
				// Daemon-driven: triage already happened, silent bypass
			} else if spawnForce {
				fmt.Fprint(os.Stderr, hotspotResult.Warning)
				fmt.Fprintln(os.Stderr, "⚠️  --force used: bypassing strategic-first gate")
				fmt.Fprintln(os.Stderr, "")
			} else if isStrategicSkill {
				fmt.Fprint(os.Stderr, hotspotResult.Warning)
				fmt.Fprintln(os.Stderr, "✓ Strategic approach: architect skill in hotspot area")
				fmt.Fprintln(os.Stderr, "")
			}
		}
	}

	return nil
}

// resolveProject resolves the project directory, name, and ensures scaffolding exists.
func (p *spawnPipeline) resolveProject() error {
	var err error
	if spawnWorkdir != "" {
		p.projectDir, err = filepath.Abs(spawnWorkdir)
		if err != nil {
			return fmt.Errorf("failed to resolve workdir path: %w", err)
		}
		if stat, err := os.Stat(p.projectDir); err != nil {
			return fmt.Errorf("workdir does not exist: %s", p.projectDir)
		} else if !stat.IsDir() {
			return fmt.Errorf("workdir is not a directory: %s", p.projectDir)
		}
	} else {
		p.projectDir, err = currentProjectDir()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	p.projectName = filepath.Base(p.projectDir)

	// Check and optionally auto-initialize scaffolding
	if err := ensureOrchScaffolding(p.projectDir, spawnAutoInit, spawnNoTrack); err != nil {
		return err
	}

	// Load project config once so earlier phases (context gating) can use it.
	p.projCfg, _ = config.Load(p.projectDir)

	return nil
}

// loadSkill loads skill content and detects skill type (orchestrator, meta-orchestrator).
// Also generates the workspace name.
func (p *spawnPipeline) loadSkill() error {
	loader := skills.DefaultLoader()

	// First load raw skill content (without dependencies) to detect skill type
	rawSkillContent, err := loader.LoadSkillContent(p.skillName)
	if err == nil {
		if metadata, err := skills.ParseSkillMetadata(rawSkillContent); err == nil {
			p.isOrchestrator = metadata.SkillType == "policy" || metadata.SkillType == "orchestrator"
		}
	}

	if p.skillName == "meta-orchestrator" {
		p.isMetaOrchestrator = true
	}

	// Generate workspace name
	p.workspaceName = spawn.GenerateWorkspaceName(p.projectName, p.skillName, p.task, spawn.WorkspaceNameOptions{
		IsMetaOrchestrator: p.isMetaOrchestrator,
		IsOrchestrator:     p.isOrchestrator,
	})

	// Load full skill content with dependencies
	p.skillContent, err = loader.LoadSkillWithDependencies(p.skillName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load skill '%s': %v\n", p.skillName, err)
		p.skillContent = ""
	}

	return nil
}

// setupIssueTracking determines beads ID, checks for duplicates, retry patterns,
// existing agents, parent epic status, and updates issue status.
func (p *spawnPipeline) setupIssueTracking() error {
	// Orchestrators skip beads tracking entirely
	p.skipBeadsForOrchestrator = p.isOrchestrator || p.isMetaOrchestrator

	var err error
	p.beadsID, err = determineBeadsID(p.projectName, p.skillName, p.task, spawnIssue, spawnWorkdir, spawnNoTrack || p.skipBeadsForOrchestrator, createBeadsIssue)
	if err != nil {
		return fmt.Errorf("failed to determine beads ID: %w", err)
	}
	if p.skipBeadsForOrchestrator {
		fmt.Println("Skipping beads tracking (orchestrator session)")
	} else if spawnNoTrack {
		fmt.Println("Skipping beads tracking (--no-track)")
	}

	// DUPLICATE AGENT CHECK
	if !spawnNoTrack && !p.skipBeadsForOrchestrator && p.beadsID != "" && !spawnForce {
		if activeAgent, err := checkActiveAgentForBeadsID(p.beadsID); err == nil && activeAgent != nil {
			return formatActiveAgentError(p.beadsID, activeAgent)
		}
	}

	// Check for retry patterns on existing issues
	if !spawnNoTrack && !p.skipBeadsForOrchestrator && spawnIssue != "" {
		if stats, err := verify.GetFixAttemptStats(p.beadsID); err == nil && stats.IsRetryPattern() {
			warning := verify.FormatRetryWarning(stats)
			if warning != "" {
				fmt.Fprintf(os.Stderr, "\n%s\n", warning)
			}
		}
	}

	// DISABLED: Dependency check gate (Jan 4, 2026)
	_ = spawnForce // silence unused variable warning

	// Check if issue is already being worked on
	if !spawnNoTrack && !p.skipBeadsForOrchestrator && spawnIssue != "" {
		if issue, err := verify.GetIssue(p.beadsID); err == nil {
			if issue.Status == "closed" {
				return fmt.Errorf("issue %s is already closed", p.beadsID)
			}
			if complete, err := verify.IsPhaseComplete(p.beadsID); err == nil && complete {
				return fmt.Errorf("issue %s has Phase: Complete but is not closed. Run 'orch complete %s' first", p.beadsID, p.beadsID)
			}
			if issue.Status == "in_progress" {
				sessions, _ := p.client.ListSessions("")
				for _, s := range sessions {
					if strings.Contains(s.Title, p.beadsID) {
						if p.client.IsSessionActive(s.ID, 30*time.Minute) {
							return fmt.Errorf("issue %s is already in_progress with active agent (session %s). Use 'orch send %s' to interact or 'orch abandon %s' to restart", p.beadsID, s.ID, s.ID, p.beadsID)
						}
						fmt.Fprintf(os.Stderr, "Note: found stale session %s for issue %s (no activity in 30m)\n", s.ID[:12], p.beadsID)
					}
				}
				fmt.Fprintf(os.Stderr, "Warning: issue %s is in_progress but no active agent found. Respawning.\n", p.beadsID)
			}
		}
	}

	// Area label check
	if !spawnNoTrack && !p.skipBeadsForOrchestrator && spawnIssue != "" {
		if issue, err := verify.GetIssue(p.beadsID); err == nil {
			if !beads.HasAreaLabel(issue.Labels) {
				suggested := beads.SuggestAreaLabel(issue.Title, issue.Description)
				warning := beads.FormatAreaLabelWarning(issue.Labels, suggested)
				if warning != "" {
					fmt.Fprint(os.Stderr, warning)
				}
			}
		}
	}

	// Pre-flight check: warn if spawning under a closed parent epic
	if !spawnNoTrack && !p.skipBeadsForOrchestrator && spawnIssue != "" {
		parentID := verify.ExtractParentID(p.beadsID)
		if parentID != "" {
			closed, err := verify.IsEpicClosed(parentID)
			if err == nil && closed {
				fmt.Fprintf(os.Stderr, "\033[1;33mWarning: Parent epic %s is already closed.\033[0m\n", parentID)
				fmt.Fprintf(os.Stderr, "This child issue may be orphaned. Continue? [y/N]: ")
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
					return fmt.Errorf("spawn cancelled: parent epic is closed")
				}
				fmt.Fprintf(os.Stderr, "Proceeding with spawn under closed epic...\n")
			}
		}
	}

	// Update beads issue status to in_progress
	if !spawnNoTrack && !p.skipBeadsForOrchestrator && spawnIssue != "" {
		if err := verify.UpdateIssueStatus(p.beadsID, "in_progress"); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update beads issue status: %v\n", err)
		}
	}

	return nil
}

// gatherContext gathers KB context and performs gap analysis.
func (p *spawnPipeline) gatherContext() error {
	// Parse skill requirements to determine what context to gather
	requires := spawn.ParseSkillRequires(p.skillContent)

	if !spawnSkipArtifactCheck {
		// Load config early to check for domain override
		earlyProjCfg, _ := config.Load(p.projectDir)
		var domainOverride string
		if earlyProjCfg != nil && earlyProjCfg.Domain != "" {
			domainOverride = earlyProjCfg.Domain
		}

		if requires != nil && requires.HasRequirements() {
			fmt.Printf("Gathering context (skill requires: %s)\n", requires.String())
			p.kbContext = spawn.GatherRequiredContext(requires, p.task, p.beadsID, p.projectDir)
			p.gapAnalysis = spawn.AnalyzeGaps(nil, p.task)
		} else {
			gapResult := runPreSpawnKBCheckFull(p.task, p.projectDir, domainOverride)
			p.kbContext = gapResult.Context
			p.gapAnalysis = gapResult.GapAnalysis
		}

		threshold := spawnGapThreshold
		if threshold <= 0 && p.projCfg != nil {
			threshold = p.projCfg.SpawnContextQualityThreshold()
		}

		// Check gap gating
		if err := checkGapGating(p.gapAnalysis, spawnGateOnGap, spawnSkipGapGate, threshold); err != nil {
			logGapGateBlock(p, threshold)
			return err
		}

		// Record gap for learning loop
		if p.gapAnalysis != nil && p.gapAnalysis.HasGaps {
			recordGapForLearning(p.gapAnalysis, p.skillName, p.task)
		}

		// Log if skip-gap-gate was used
		if spawnSkipGapGate && p.gapAnalysis != nil && p.gapAnalysis.ShouldBlockSpawn(threshold) {
			fmt.Fprintf(os.Stderr, "⚠️  Bypassing gap gate (--skip-gap-gate): context quality %d\n", p.gapAnalysis.ContextQuality)
			logGapGateBypass(p)
		}
	} else {
		fmt.Println("Skipping context check (--skip-artifact-check)")
	}

	return nil
}

// logGapGateBlock logs a blocked spawn due to gap gating.
func logGapGateBlock(p *spawnPipeline, threshold int) {
	logger := events.NewLogger(events.DefaultLogPath())

	criticalGaps := []string{}
	if p.gapAnalysis != nil {
		for _, gap := range p.gapAnalysis.Gaps {
			if gap.Severity == spawn.GapSeverityCritical {
				criticalGaps = append(criticalGaps, gap.Description)
			}
		}
	}

	event := events.Event{
		Type:      "spawn.blocked.gap_gate",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"task":            p.task,
			"context_quality": p.gapAnalysis.ContextQuality,
			"threshold":       threshold,
			"beads_id":        p.beadsID,
			"skill":           p.skillName,
			"critical_gaps":   criticalGaps,
		},
	}
	if logErr := logger.Log(event); logErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log gap gate block: %v\n", logErr)
	}
}

// logGapGateBypass logs a gap gate bypass event.
func logGapGateBypass(p *spawnPipeline) {
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "spawn.gap.gate.bypassed",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"task":            p.task,
			"context_quality": p.gapAnalysis.ContextQuality,
			"beads_id":        p.beadsID,
			"skill":           p.skillName,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log gap bypass: %v\n", err)
	}
}

// buildSpawnConfig builds the spawn.Config from all accumulated pipeline state.
func (p *spawnPipeline) buildSpawnConfig() error {
	// Determine spawn tier
	p.tier = determineSpawnTier(p.skillName, spawnLight, spawnFull)

	// Extract reproduction info for bug issues
	if !spawnNoTrack && p.beadsID != "" {
		if reproResult, err := verify.GetReproForCompletion(p.beadsID); err == nil && reproResult != nil {
			p.isBug = reproResult.IsBug
			p.reproSteps = reproResult.Repro
			if p.isBug && p.reproSteps != "" {
				fmt.Printf("🐛 Bug issue detected - reproduction steps included in context\n")
			}
		}
	}

	// Build usage info from check result
	if p.usageCheckResult != nil && p.usageCheckResult.CapacityInfo != nil {
		p.usageInfo = &spawn.UsageInfo{
			FiveHourUsed: p.usageCheckResult.CapacityInfo.FiveHourUsed,
			SevenDayUsed: p.usageCheckResult.CapacityInfo.SevenDayUsed,
			AccountEmail: p.usageCheckResult.CapacityInfo.Email,
			AutoSwitched: p.usageCheckResult.Switched,
			SwitchReason: p.usageCheckResult.SwitchReason,
		}
	}

	// Load project config if not already loaded during resolveProject.
	if p.projCfg == nil {
		p.projCfg, _ = config.Load(p.projectDir)
	}

	// Determine spawn backend
	p.globalCfg, _ = userconfig.Load()
	resolution := resolveBackend(
		spawnBackendFlag,
		spawnOpus,
		spawnInfra,
		spawnModel,
		p.projCfg,
		p.globalCfg,
		p.task,
		p.beadsID,
	)

	if resolution.Error != nil {
		return fmt.Errorf("backend resolution failed: %w", resolution.Error)
	}
	for _, warning := range resolution.Warnings {
		fmt.Println(warning)
	}
	if os.Getenv("ORCH_DEBUG") != "" {
		fmt.Printf("Backend: %s (%s)\n", resolution.Backend, resolution.Reason)
	}
	p.spawnBackend = resolution.Backend

	// Validate model+backend compatibility
	if warning := validateBackendModelCompatibility(p.spawnBackend, spawnModel); warning != "" {
		fmt.Println(warning)
	}

	// Resolve model with config support
	p.resolvedModel = resolveModelWithConfig(spawnModel, p.spawnBackend, p.skillName, p.projCfg, p.globalCfg)

	// Validate flash model
	if p.resolvedModel.Provider == "google" && strings.Contains(strings.ToLower(p.resolvedModel.ModelID), "flash") {
		return fmt.Errorf(`
┌─────────────────────────────────────────────────────────────────────────────┐
│  🚫 Flash model not supported                                                │
├─────────────────────────────────────────────────────────────────────────────┤
│  Gemini Flash has TPM (tokens per minute) rate limits that make it           │
│  unsuitable for agent work. Use opus (default) or sonnet instead.            │
│                                                                             │
│  Available options:                                                         │
│    • --model opus    (default, Max subscription via claude CLI)             │
│    • --model sonnet  (pay-per-token, requires --backend opencode)           │
└─────────────────────────────────────────────────────────────────────────────┘
`)
	}

	// Validate mode+model combination
	if err := validateModeModelCombo(p.spawnBackend, p.resolvedModel); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  %v\n", err)
	}

	// Read design artifacts if --design-workspace is provided
	if spawnDesignWorkspace != "" {
		p.designMockupPath, p.designPromptPath, p.designNotes = readDesignArtifacts(p.projectDir, spawnDesignWorkspace)
		if p.designMockupPath != "" {
			fmt.Printf("📐 Design handoff from workspace: %s\n", spawnDesignWorkspace)
			fmt.Printf("   Mockup: %s\n", p.designMockupPath)
			if p.designPromptPath != "" {
				fmt.Printf("   Prompt: %s\n", p.designPromptPath)
			}
		}
	}

	// Determine extended thinking variant
	p.variant = spawnVariant
	if p.variant == "" {
		p.variant = spawn.DefaultVariantForRole(p.isOrchestrator, p.isMetaOrchestrator, p.skillName)
	}
	if p.variant == "none" {
		p.variant = ""
	}

	// Fetch issue metadata for state DB recording
	if !spawnNoTrack && !p.skipBeadsForOrchestrator && p.beadsID != "" {
		if issue, err := verify.GetIssue(p.beadsID); err == nil && issue != nil {
			p.issueTitle = issue.Title
			p.issueType = issue.IssueType
		}
		if p.issuePriority == 0 {
			p.issuePriority = 2 // Default P2
		}
	}

	// Include resource lifecycle audit guidance for infrastructure-touching spawns.
	// Explicit --infra always qualifies. Otherwise, use broader infrastructure
	// detection focused on resource-lifecycle risk areas.
	p.isInfrastructureTouching = spawnInfra || requiresResourceLifecycleAudit(p.task, p.beadsID)

	// Behavioral acceptance criteria require integration validation for feature work.
	validationLevel, escalated, matchedCriteria := determineValidationLevel(p.skillName, spawnValidation, p.task)
	if escalated {
		fmt.Fprintf(os.Stderr, "⚠️  Behavioral acceptance detected - escalating validation to integration\n")
		for _, c := range matchedCriteria {
			fmt.Fprintf(os.Stderr, "   - %s\n", c)
		}
	}

	// Build spawn config
	p.cfg = &spawn.Config{
		Task:                     p.task,
		SkillName:                p.skillName,
		Project:                  p.projectName,
		ProjectDir:               p.projectDir,
		WorkspaceName:            p.workspaceName,
		SkillContent:             p.skillContent,
		BeadsID:                  p.beadsID,
		Phases:                   spawnPhases,
		Mode:                     spawnMode,
		Validation:               validationLevel,
		Model:                    p.resolvedModel.Format(),
		Variant:                  p.variant,
		MCP:                      spawnMCP,
		Tier:                     p.tier,
		NoTrack:                  spawnNoTrack || p.skipBeadsForOrchestrator,
		SkipArtifactCheck:        spawnSkipArtifactCheck,
		KBContext:                p.kbContext,
		IncludeServers:           spawn.DefaultIncludeServersForSkill(p.skillName),
		GapAnalysis:              p.gapAnalysis,
		IsBug:                    p.isBug,
		ReproSteps:               p.reproSteps,
		IsInfrastructureTouching: p.isInfrastructureTouching,
		IsOrchestrator:           p.isOrchestrator,
		IsMetaOrchestrator:       p.isMetaOrchestrator,
		UsageInfo:                p.usageInfo,
		SpawnMode:                p.spawnBackend,
		DesignWorkspace:          spawnDesignWorkspace,
		DesignMockupPath:         p.designMockupPath,
		DesignPromptPath:         p.designPromptPath,
		DesignNotes:              p.designNotes,
		DaemonDriven:             p.daemonDriven,
		IssueComments:            fetchIssueCommentsForSpawn(p.beadsID),
		FailureContext:           fetchFailureContextForSpawn(p.beadsID),
		IssueTitle:               p.issueTitle,
		IssueType:                p.issueType,
		IssuePriority:            p.issuePriority,
	}

	return nil
}

// determineValidationLevel normalizes validation level and auto-escalates for behavioral criteria.
func determineValidationLevel(skillName, requestedValidation, task string) (string, bool, []string) {
	validation := strings.ToLower(strings.TrimSpace(requestedValidation))
	if validation == "" {
		validation = "tests"
	}

	if strings.ToLower(strings.TrimSpace(skillName)) != "feature-impl" {
		return validation, false, nil
	}

	behavioral, criteria := verify.DetectBehavioralAcceptanceCriteria(task)
	if !behavioral {
		return validation, false, nil
	}

	if validation == "none" || validation == "tests" {
		return "integration", true, criteria
	}

	return validation, false, criteria
}

// executeSpawn validates the config, writes context, records state, and dispatches
// to the appropriate spawn backend (headless, tmux, claude, docker, inline).
func (p *spawnPipeline) executeSpawn() error {
	// Pre-spawn token estimation and validation
	if err := spawn.ValidateContextSize(p.cfg); err != nil {
		return fmt.Errorf("pre-spawn validation failed: %w", err)
	}

	// Warn about large contexts (but don't block)
	if shouldWarn, warning := spawn.ShouldWarnAboutSize(p.cfg); shouldWarn {
		fmt.Fprintf(os.Stderr, "%s", warning)
	}

	// Check for existing workspace before writing context
	if err := checkWorkspaceExists(p.cfg.WorkspacePath(), spawnForce); err != nil {
		return err
	}

	// Write SPAWN_CONTEXT.md
	if err := spawn.WriteContext(p.cfg); err != nil {
		return fmt.Errorf("failed to write spawn context: %w", err)
	}

	// Record agent in state database (non-fatal)
	if err := statedb.RecordSpawn(p.cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to record agent in state db: %v\n", err)
	}

	// Generate minimal prompt
	minimalPrompt := spawn.MinimalPrompt(p.cfg)

	// Connect MCP servers if --mcp is specified (opencode backend only)
	if p.cfg.MCP != "" && p.cfg.SpawnMode != "claude" && p.cfg.SpawnMode != "docker" {
		for _, name := range strings.Split(p.cfg.MCP, ",") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			fmt.Printf("Connecting MCP server: %s\n", name)
			if err := p.client.MCPConnect(name); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to connect MCP server %s: %v\n", name, err)
			}
		}
	}

	// Dispatch to backend-specific spawn function
	return p.dispatchSpawn(minimalPrompt)
}

// dispatchSpawn routes to the appropriate backend spawn function.
func (p *spawnPipeline) dispatchSpawn(minimalPrompt string) error {
	// Explicit backend config takes priority
	if p.cfg.SpawnMode == "claude" {
		if p.inline {
			return runSpawnClaudeInline(p.serverURL, p.cfg, p.beadsID, p.skillName, p.task)
		}
		return runSpawnClaude(p.serverURL, p.cfg, p.beadsID, p.skillName, p.task, p.attach)
	}

	if p.cfg.SpawnMode == "docker" {
		return runSpawnDocker(p.serverURL, p.cfg, p.beadsID, p.skillName, p.task, p.attach)
	}

	// Inline mode (blocking) for opencode backend
	if p.inline {
		return runSpawnInline(p.serverURL, p.cfg, minimalPrompt, p.beadsID, p.skillName, p.task)
	}

	// Headless flag only applies when no explicit backend is configured
	if p.headless {
		return runSpawnHeadless(p.serverURL, p.cfg, minimalPrompt, p.beadsID, p.skillName, p.task)
	}

	// Orchestrator-type skills default to tmux mode
	useTmux := p.tmux || p.attach || p.cfg.IsOrchestrator || spawnInfra
	if useTmux {
		return runSpawnTmux(p.serverURL, p.cfg, minimalPrompt, p.beadsID, p.skillName, p.task, p.attach)
	}

	// Default for workers: Headless mode
	return runSpawnHeadless(p.serverURL, p.cfg, minimalPrompt, p.beadsID, p.skillName, p.task)
}
