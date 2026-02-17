// Package orch provides orchestration-level utilities for agent spawn management.
// This includes spawn pipeline functions extracted from cmd/orch/spawn_cmd.go.
package orch

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/skills"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/spawn/gates"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// SpawnInput holds all input parameters for spawn operation.
// This follows the pattern from complete_cmd.go for organizing function parameters.
type SpawnInput struct {
	ServerURL    string
	SkillName    string
	Task         string
	Inline       bool
	Headless     bool
	Tmux         bool
	Attach       bool
	DaemonDriven bool
}

// SpawnContext holds all computed context for spawn operation.
// This accumulates values as we progress through the spawn pipeline.
type SpawnContext struct {
	Task               string
	SkillName          string
	ProjectDir         string
	ProjectName        string
	WorkspaceName      string
	SkillContent       string
	BeadsID            string
	IsOrchestrator     bool
	IsMetaOrchestrator bool
	ResolvedModel      model.ModelSpec
	KBContext          string
	GapAnalysis        *spawn.GapAnalysis
	HasInjectedModels  bool
	PrimaryModelPath   string
	IsBug              bool
	ReproSteps         string
	UsageInfo          *spawn.UsageInfo
	SpawnBackend       string
	Tier               string
	DesignMockupPath   string
	DesignPromptPath   string
	DesignNotes        string
}

// headlessSpawnResult contains the result of starting a headless session.
type headlessSpawnResult struct {
	SessionID string
}

// GapCheckResult contains the results of a pre-spawn gap check.
type GapCheckResult struct {
	Context      string                       // Formatted context to include in SPAWN_CONTEXT.md
	GapAnalysis  *spawn.GapAnalysis           // Gap analysis results for further processing
	Blocked      bool                         // True if spawn should be blocked due to gaps
	BlockReason  string                       // Reason for blocking (if Blocked is true)
	FormatResult *spawn.KBContextFormatResult // Full format result including HasInjectedModels
}

// ansiRegex matches ANSI escape sequences (colors, formatting, etc.)
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// InferSkillFromIssueType maps issue types to appropriate skills.
// Returns an error for types that cannot be spawned (e.g., epic) or unknown types.
//
// Bug handling: Defaults to "architect" (understand before fixing) rather than
// "systematic-debugging". This implements the "Premise Before Solution" principle -
// most bugs reported as vague symptoms need understanding before patching.
// Use explicit skill:systematic-debugging label for isolated bugs with clear cause.
func InferSkillFromIssueType(issueType string) (string, error) {
	switch issueType {
	case "bug":
		// Default to architect: understand before fixing
		// Use skill:systematic-debugging label for clear, isolated bugs
		return "architect", nil
	case "feature":
		return "feature-impl", nil
	case "task":
		return "feature-impl", nil
	case "investigation":
		return "investigation", nil
	case "epic":
		return "", fmt.Errorf("cannot spawn work on epic issues - epics are decomposed into sub-issues")
	case "":
		return "", fmt.Errorf("issue type is empty")
	default:
		return "", fmt.Errorf("unknown issue type: %s", issueType)
	}
}

// DetermineSpawnTier determines the spawn tier based on flags, config, and skill defaults.
// Priority: --light flag > --full flag > userconfig.default_tier > skill default > TierFull (conservative)
func DetermineSpawnTier(skillName string, lightFlag, fullFlag bool) string {
	// Explicit flags take precedence
	if lightFlag {
		return spawn.TierLight
	}
	if fullFlag {
		return spawn.TierFull
	}
	// Check userconfig for default tier override
	cfg, err := userconfig.Load()
	if err == nil && cfg.GetDefaultTier() != "" {
		return cfg.GetDefaultTier()
	}
	// Fall back to skill default
	return spawn.DefaultTierForSkill(skillName)
}

// CheckAndAutoSwitchAccount checks if the current account is over usage thresholds
// and automatically switches to a better account if available.
// Returns nil if no switch was needed or switch succeeded.
// Logs the switch action if one occurs.
func CheckAndAutoSwitchAccount() error {
	// Get thresholds from environment or use defaults
	thresholds := account.DefaultAutoSwitchThresholds()

	// Allow override via environment variables
	if envVal := os.Getenv("ORCH_AUTO_SWITCH_5H_THRESHOLD"); envVal != "" {
		if val, err := strconv.ParseFloat(envVal, 64); err == nil && val > 0 && val <= 100 {
			thresholds.FiveHourThreshold = val
		}
	}
	if envVal := os.Getenv("ORCH_AUTO_SWITCH_WEEKLY_THRESHOLD"); envVal != "" {
		if val, err := strconv.ParseFloat(envVal, 64); err == nil && val > 0 && val <= 100 {
			thresholds.WeeklyThreshold = val
		}
	}
	if envVal := os.Getenv("ORCH_AUTO_SWITCH_MIN_DELTA"); envVal != "" {
		if val, err := strconv.ParseFloat(envVal, 64); err == nil && val >= 0 {
			thresholds.MinHeadroomDelta = val
		}
	}

	// Check if auto-switch is explicitly disabled
	if os.Getenv("ORCH_AUTO_SWITCH_DISABLED") == "1" || os.Getenv("ORCH_AUTO_SWITCH_DISABLED") == "true" {
		return nil
	}

	result, err := account.AutoSwitchIfNeeded(thresholds)
	if err != nil {
		// Log warning but don't block spawn - continue with current account
		fmt.Fprintf(os.Stderr, "Warning: auto-switch check failed: %v\n", err)

		// Check if the underlying error is a TokenRefreshError and provide guidance
		var tokenErr *account.TokenRefreshError
		if errors.As(err, &tokenErr) {
			fmt.Fprintf(os.Stderr, "  → %s\n", tokenErr.ActionableGuidance())
		}
		return nil
	}

	if result.Switched {
		// Log the switch
		logger := events.NewLogger(events.DefaultLogPath())
		event := events.Event{
			Type:      "account.auto_switched",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"from_account":     result.FromAccount,
				"to_account":       result.ToAccount,
				"reason":           result.Reason,
				"from_5h_used":     result.FromCapacity.FiveHourUsed,
				"from_weekly_used": result.FromCapacity.SevenDayUsed,
				"to_5h_used":       result.ToCapacity.FiveHourUsed,
				"to_weekly_used":   result.ToCapacity.SevenDayUsed,
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log account switch: %v\n", err)
		}

		fmt.Printf("🔄 Auto-switched account: %s\n", result.Reason)
	}

	return nil
}

// validateModeModelCombo checks for known invalid mode+model combinations.
// Returns a warning error (non-blocking) if an invalid combination is detected.
func validateModeModelCombo(backend string, resolvedModel model.ModelSpec) error {
	// Invalid combination: opencode + opus
	// Opus requires Claude Code CLI auth, opencode backend creates zombie agents
	if backend == "opencode" && strings.Contains(strings.ToLower(resolvedModel.ModelID), "opus") {
		return fmt.Errorf(`Warning: opencode backend with opus model may fail (auth blocked).
  Recommendation: Use --model sonnet (default) or let auto-selection use claude backend`)
	}

	// Note: Flash model is blocked earlier in the flow (hard error, not warning)
	// Claude backend + non-opus models work but are non-optimal (using Max sub for cheap models)

	return nil
}

// inferSkillFromBeadsIssue infers skill from a beads issue using labels, title, then type.
func inferSkillFromBeadsIssue(issue *beads.Issue) string {
	// Check for skill:* labels first
	for _, label := range issue.Labels {
		if strings.HasPrefix(label, "skill:") {
			return strings.TrimPrefix(label, "skill:")
		}
	}

	// Check for title patterns (e.g., synthesis issues)
	if strings.HasPrefix(issue.Title, "Synthesize ") && strings.Contains(issue.Title, " investigations") {
		return "kb-reflect"
	}

	// Fall back to type-based inference
	skill, err := InferSkillFromIssueType(issue.IssueType)
	if err != nil {
		return "feature-impl" // Default fallback
	}
	return skill
}

// inferMCPFromBeadsIssue extracts MCP server requirements from issue labels.
// Returns the MCP server name if found (e.g., "playwright" from "needs:playwright"),
// or empty string if no MCP-related label is present.
//
// This allows daemon-spawned agents to automatically get browser access when
// working on UI/CSS fixes that require visual verification.
func inferMCPFromBeadsIssue(issue *beads.Issue) string {
	for _, label := range issue.Labels {
		if strings.HasPrefix(label, "needs:") {
			need := strings.TrimPrefix(label, "needs:")
			// Map needs labels to MCP servers
			switch need {
			case "playwright":
				return "playwright"
				// Future: add more mappings as needed
			}
		}
	}
	return ""
}

// RunPreFlightChecks performs all pre-spawn validation checks.
// Returns usage check result for telemetry, or error if any check fails.
// hotspotCheckFunc is passed from cmd/orch to avoid circular dependencies.
func RunPreFlightChecks(input *SpawnInput, preCheckDir string, bypassTriage, bypassVerification bool, bypassReason string, maxAgents int, extractBeadsIDFunc func(string) string, hotspotCheckFunc func(string, string) (*gates.HotspotResult, error)) (*gates.UsageCheckResult, error) {
	// Check for --bypass-triage flag (required for manual spawns)
	// Daemon-driven spawns skip this check (issue already triaged)
	if err := gates.CheckTriageBypass(input.DaemonDriven, bypassTriage, input.SkillName, input.Task); err != nil {
		return nil, err
	}

	// Log the triage bypass for Phase 2 review (only for manual bypasses, not daemon-driven)
	if !input.DaemonDriven && bypassTriage {
		gates.LogTriageBypass(input.SkillName, input.Task)
	}

	// Check verification gate (Phase 3: Session Continuity Gate)
	// Block spawn if unverified Tier 1 work exists (prevents cascade pattern)
	// Independent parallel work can use --bypass-verification to override
	if err := gates.CheckVerificationGate(bypassVerification, bypassReason); err != nil {
		return nil, err
	}

	// Check concurrency limit before spawning
	if err := gates.CheckConcurrency(input.ServerURL, maxAgents, extractBeadsIDFunc); err != nil {
		return nil, err
	}

	// Proactive rate limit monitoring: warn at 80%, block at 95%
	usageCheckResult, usageErr := gates.CheckRateLimit()
	if usageErr != nil {
		// usageErr contains formatted blocking message
		return nil, usageErr
	}

	// STRATEGIC-FIRST ORCHESTRATION: Check for hotspots in task target area
	// In hotspot areas (5+ bugs, persistent failures), strategic approach is recommended
	// Warning shown but spawn proceeds (non-blocking)
	if hotspotCheckFunc != nil {
		gates.CheckHotspot(preCheckDir, input.Task, input.SkillName, input.DaemonDriven, hotspotCheckFunc)
	}

	return usageCheckResult, nil
}

// ResolveProjectDirectory determines the project directory and name.
// Uses workdir if provided, otherwise current working directory.
func ResolveProjectDirectory(workdir string) (projectDir, projectName string, err error) {
	if workdir != "" {
		// User specified target directory via --workdir
		projectDir, err = filepath.Abs(workdir)
		if err != nil {
			return "", "", fmt.Errorf("failed to resolve workdir path: %w", err)
		}
		// Verify directory exists
		if stat, err := os.Stat(projectDir); err != nil {
			return "", "", fmt.Errorf("workdir does not exist: %s", projectDir)
		} else if !stat.IsDir() {
			return "", "", fmt.Errorf("workdir is not a directory: %s", projectDir)
		}
	} else {
		// Default: use current working directory
		projectDir, err = os.Getwd()
		if err != nil {
			return "", "", fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Get project name from directory
	projectName = filepath.Base(projectDir)
	return projectDir, projectName, nil
}

// LoadSkillAndGenerateWorkspace loads skill content and generates workspace name.
// The ensureScaffoldingFunc is called to check/initialize scaffolding (passed from cmd/orch).
func LoadSkillAndGenerateWorkspace(skillName, projectName, task, projectDir string, autoInit, noTrack bool, ensureScaffoldingFunc func(string, bool, bool) error) (
	skillContent, workspaceName string,
	isOrchestrator, isMetaOrchestrator bool,
	err error) {

	// Check and optionally auto-initialize scaffolding
	if ensureScaffoldingFunc != nil {
		if err := ensureScaffoldingFunc(projectDir, autoInit, noTrack); err != nil {
			return "", "", false, false, err
		}
	}

	// Load skill content with dependencies (e.g., worker-base patterns)
	loader := skills.DefaultLoader()

	// First load raw skill content (without dependencies) to detect skill type
	// This is needed because LoadSkillWithDependencies puts dependency content first,
	// which means the main skill's frontmatter isn't at the start of the combined content
	rawSkillContent, err := loader.LoadSkillContent(skillName)
	if err == nil {
		if metadata, err := skills.ParseSkillMetadata(rawSkillContent); err == nil {
			isOrchestrator = metadata.SkillType == "policy" || metadata.SkillType == "orchestrator"
		}
	}
	// Detect meta-orchestrator by skill name (not skill-type)
	// This enables tiered context templates for different orchestrator levels
	if skillName == "meta-orchestrator" {
		isMetaOrchestrator = true
	}

	// Generate workspace name
	// Meta-orchestrators use "meta-" prefix instead of project prefix for visual distinction
	// Orchestrators use "orch" skill prefix instead of "work" for visual distinction from workers
	workspaceName = spawn.GenerateWorkspaceName(projectName, skillName, task, spawn.WorkspaceNameOptions{
		IsMetaOrchestrator: isMetaOrchestrator,
		IsOrchestrator:     isOrchestrator,
	})

	// Now load full skill content with dependencies for the actual spawn
	skillContent, err = loader.LoadSkillWithDependencies(skillName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load skill '%s': %v\n", skillName, err)
		skillContent = "" // Continue without skill content
	}

	return skillContent, workspaceName, isOrchestrator, isMetaOrchestrator, nil
}

// SetupBeadsTracking determines beads ID and manages issue lifecycle.
// Returns final beads ID (empty if untracked), or error if setup fails.
func SetupBeadsTracking(skillName, task, projectName, beadsIssueFlag string, isOrchestrator, isMetaOrchestrator bool, serverURL string, noTrack bool, createBeadsFn func(string, string, string) (string, error)) (string, error) {
	// Determine beads ID - either from flag, create new issue, or skip if --no-track
	// Orchestrators skip beads tracking entirely - they're interactive sessions with Dylan,
	// not autonomous tasks. SESSION_HANDOFF.md is richer than beads comments.
	skipBeadsForOrchestrator := isOrchestrator || isMetaOrchestrator
	beadsID, err := determineBeadsID(projectName, skillName, task, beadsIssueFlag, noTrack || skipBeadsForOrchestrator, createBeadsFn)
	if err != nil {
		return "", fmt.Errorf("failed to determine beads ID: %w", err)
	}
	if skipBeadsForOrchestrator {
		fmt.Println("Skipping beads tracking (orchestrator session)")
	} else if noTrack {
		fmt.Println("Skipping beads tracking (--no-track)")
	}

	// Check for retry patterns on existing issues - surface to prevent blind respawning
	// Skip for orchestrators since they don't use beads tracking
	if !noTrack && !skipBeadsForOrchestrator && beadsIssueFlag != "" {
		if stats, err := verify.GetFixAttemptStats(beadsID); err == nil && stats.IsRetryPattern() {
			warning := verify.FormatRetryWarning(stats)
			if warning != "" {
				fmt.Fprintf(os.Stderr, "\n%s\n", warning)
			}
		}
	}

	// Check if issue is already being worked on (prevent duplicate spawns)
	// Skip for orchestrators since they don't use beads tracking
	if !noTrack && !skipBeadsForOrchestrator && beadsIssueFlag != "" {
		if issue, err := verify.GetIssue(beadsID); err == nil {
			if issue.Status == "closed" {
				return "", fmt.Errorf("issue %s is already closed", beadsID)
			}
			if issue.Status == "in_progress" {
				// Check if there's a truly active agent for this issue
				// OpenCode persists sessions to disk, so we must verify liveness not just existence
				client := opencode.NewClient(serverURL)
				sessions, _ := client.ListSessions("")
				for _, s := range sessions {
					if strings.Contains(s.Title, beadsID) {
						// Session exists - but is it actually active (recently updated)?
						// Use 30 minute threshold - if no activity, session is stale
						if client.IsSessionActive(s.ID, 30*time.Minute) {
							return "", fmt.Errorf("issue %s is already in_progress with active agent (session %s). Use 'orch send %s' to interact or 'orch abandon %s' to restart", beadsID, s.ID, s.ID, beadsID)
						}
						// Session exists but is stale - log and continue (allow respawn)
						fmt.Fprintf(os.Stderr, "Note: found stale session %s for issue %s (no activity in 30m)\n", s.ID[:12], beadsID)
					}
				}
				// No active session - check if Phase: Complete was reported
				// If so, orchestrator needs to run 'orch complete' before respawning
				if complete, err := verify.IsPhaseComplete(beadsID); err == nil && complete {
					return "", fmt.Errorf("issue %s has Phase: Complete but is not closed. Run 'orch complete %s' first", beadsID, beadsID)
				}
				// In progress but no active agent and not Phase: Complete - warn but allow respawn
				fmt.Fprintf(os.Stderr, "Warning: issue %s is in_progress but no active agent found. Respawning.\n", beadsID)
			}
		}
	}

	// Update beads issue status to in_progress (only if tracking a real issue)
	// Skip for orchestrators since they don't use beads tracking
	if !noTrack && !skipBeadsForOrchestrator && beadsIssueFlag != "" {
		if err := verify.UpdateIssueStatus(beadsID, "in_progress"); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update beads issue status: %v\n", err)
			// Continue anyway
		}
	}

	return beadsID, nil
}

// ResolveAndValidateModel resolves model aliases and validates model choice.
// Returns error if flash model is requested (unsupported).
func ResolveAndValidateModel(modelFlag string) (model.ModelSpec, error) {
	// Resolve model - convert aliases to full format
	resolvedModel := model.Resolve(modelFlag)

	// Validate flash model - TPM rate limits make it unusable
	if resolvedModel.Provider == "google" && strings.Contains(strings.ToLower(resolvedModel.ModelID), "flash") {
		return resolvedModel, fmt.Errorf(`
┌─────────────────────────────────────────────────────────────────────────────┐
│  🚫 Flash model not supported                                                │
├─────────────────────────────────────────────────────────────────────────────┤
│  Gemini Flash has TPM (tokens per minute) rate limits that make it           │
│  unsuitable for agent work. Use sonnet (default) or opus instead.            │
│                                                                             │
│  Available options:                                                         │
│    • --model sonnet  (default, pay-per-token via OpenCode)                  │
│    • --model opus    (Max subscription via claude CLI)                      │
└─────────────────────────────────────────────────────────────────────────────┘
`)
	}

	return resolvedModel, nil
}

// GatherSpawnContext gathers KB context and performs gap analysis.
// Returns context string, gap analysis, model injection info, or error.
func GatherSpawnContext(skillContent, task, beadsID, projectDir string, skipArtifactCheck, gateOnGap, skipGapGate bool, gapThreshold int) (
	kbContext string,
	gapAnalysis *spawn.GapAnalysis,
	hasInjectedModels bool,
	primaryModelPath string,
	err error) {

	if skipArtifactCheck {
		fmt.Println("Skipping context check (--skip-artifact-check)")
		return "", nil, false, "", nil
	}

	// Parse skill requirements to determine what context to gather
	requires := spawn.ParseSkillRequires(skillContent)

	if requires != nil && requires.HasRequirements() {
		// Skill-driven context gathering
		fmt.Printf("Gathering context (skill requires: %s)\n", requires.String())
		kbContext = spawn.GatherRequiredContext(requires, task, beadsID, projectDir)
		// For skill-driven context, create a basic gap analysis from the results
		// This is a placeholder - skills may provide their own gap info
		gapAnalysis = spawn.AnalyzeGaps(nil, task)
	} else {
		// Fall back to default kb context check with full gap analysis
		gapResult := runPreSpawnKBCheckFull(task)
		kbContext = gapResult.Context
		gapAnalysis = gapResult.GapAnalysis

		// Extract model injection info for probe vs investigation routing
		if gapResult.FormatResult != nil {
			hasInjectedModels = gapResult.FormatResult.HasInjectedModels
			if hasInjectedModels {
				// Extract primary model path from KB context result
				primaryModelPath = extractPrimaryModelPath(gapResult.FormatResult)
			}
		}
	}

	// Check gap gating - may block spawn if context quality is too low
	if err := checkGapGating(gapAnalysis, gateOnGap, skipGapGate, gapThreshold); err != nil {
		return "", nil, false, "", err
	}

	// Record gap for learning loop (if gaps detected)
	if gapAnalysis != nil && gapAnalysis.HasGaps {
		recordGapForLearning(gapAnalysis, skillContent, task)
	}

	// Log if skip-gap-gate was used (documents conscious bypass)
	if skipGapGate && gapAnalysis != nil && gapAnalysis.ShouldBlockSpawn(gapThreshold) {
		fmt.Fprintf(os.Stderr, "⚠️  Bypassing gap gate (--skip-gap-gate): context quality %d\n", gapAnalysis.ContextQuality)
		// Log the bypass for pattern detection
		logger := events.NewLogger(events.DefaultLogPath())
		event := events.Event{
			Type:      "gap.gate.bypassed",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"task":            task,
				"context_quality": gapAnalysis.ContextQuality,
				"beads_id":        beadsID,
				"skill":           skillContent,
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log gap bypass: %v\n", err)
		}
	}

	return kbContext, gapAnalysis, hasInjectedModels, primaryModelPath, nil
}

// ExtractBugReproInfo extracts reproduction steps if the issue is a bug.
// Returns isBug flag and reproduction steps.
func ExtractBugReproInfo(beadsID string, noTrack bool) (isBug bool, reproSteps string) {
	if noTrack || beadsID == "" {
		return false, ""
	}

	if reproResult, err := verify.GetReproForCompletion(beadsID); err == nil && reproResult != nil {
		isBug = reproResult.IsBug
		reproSteps = reproResult.Repro
		if isBug && reproSteps != "" {
			fmt.Printf("🐛 Bug issue detected - reproduction steps included in context\n")
		}
	}
	return isBug, reproSteps
}

// BuildUsageInfo converts rate limit check result to UsageInfo struct.
// Returns nil if no usage check result available.
func BuildUsageInfo(usageCheckResult *gates.UsageCheckResult) *spawn.UsageInfo {
	if usageCheckResult == nil || usageCheckResult.CapacityInfo == nil {
		return nil
	}

	return &spawn.UsageInfo{
		FiveHourUsed: usageCheckResult.CapacityInfo.FiveHourUsed,
		SevenDayUsed: usageCheckResult.CapacityInfo.SevenDayUsed,
		AccountEmail: usageCheckResult.CapacityInfo.Email,
		AutoSwitched: usageCheckResult.Switched,
		SwitchReason: usageCheckResult.SwitchReason,
	}
}

// DetermineSpawnBackend determines spawn backend with auto-selection.
// Priority: --backend flag > --opus flag > infrastructure detection > model-based > config > default.
// When --backend is explicit, it always wins - infrastructure detection becomes advisory (warning only).
func DetermineSpawnBackend(resolvedModel model.ModelSpec, task, beadsID, projectDir, backendFlag, spawnModel string, opusFlag bool) (string, error) {
	// Load project config (used for backend default)
	projCfg, _ := config.Load(projectDir)

	// Default to claude (Max subscription covers Claude CLI usage)
	backend := "claude"

	// Track whether --backend was explicitly set by user
	explicitBackend := backendFlag != ""

	if explicitBackend {
		// Explicit --backend flag: highest priority, always wins
		backend = backendFlag
		// Validate backend value
		if backend != "claude" && backend != "opencode" {
			return "", fmt.Errorf("invalid --backend value: %s (must be 'claude' or 'opencode')", backend)
		}

		// Advisory: warn if --opus conflicts with explicit --backend
		if opusFlag && backend != "claude" {
			fmt.Fprintf(os.Stderr, "⚠️  --opus flag ignored: explicit --backend %s takes priority\n", backend)
		}

		// Advisory: warn if infrastructure work detected but user chose different backend
		if isInfrastructureWork(task, beadsID) && backend != "claude" {
			fmt.Fprintf(os.Stderr, "⚠️  Infrastructure work detected but respecting explicit --backend %s\n", backend)
			fmt.Fprintf(os.Stderr, "   Recommendation: Use --backend claude for infrastructure work to survive server restarts.\n")
		}
	} else if opusFlag {
		// Explicit --opus flag: use claude CLI
		backend = "claude"
	} else if isInfrastructureWork(task, beadsID) {
		// Infrastructure work detection: auto-apply escape hatch
		// Agents working on OpenCode/orch infrastructure need claude backend + tmux
		// to survive server restarts (prevent agents from killing themselves)
		backend = "claude"
		fmt.Println("🔧 Infrastructure work detected - auto-applying escape hatch (--backend claude --tmux)")
		fmt.Println("   This ensures the agent survives OpenCode server restarts.")

		// Log the infrastructure work detection for pattern analysis
		logger := events.NewLogger(events.DefaultLogPath())
		event := events.Event{
			Type:      "spawn.infrastructure_detected",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"task":     task,
				"beads_id": beadsID,
				"skill":    "", // Will be filled by caller
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log infrastructure detection: %v\n", err)
		}
	} else if spawnModel != "" {
		// Auto-select backend based on model
		modelLower := strings.ToLower(spawnModel)
		if modelLower == "opus" || strings.Contains(modelLower, "opus") {
			// Opus model: use claude CLI (Max subscription)
			backend = "claude"
			fmt.Println("Auto-selected claude backend for opus model")
		} else if modelLower == "sonnet" || strings.Contains(modelLower, "sonnet") {
			// Sonnet model: use opencode (pay-per-token API)
			backend = "opencode"
		}
		// Other models keep the default backend (claude)
	} else if projCfg != nil && projCfg.SpawnMode == "claude" {
		// Config default: respect project spawn_mode setting
		backend = "claude"
	}

	// Validate mode+model combination
	if err := validateModeModelCombo(backend, resolvedModel); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  %v\n", err)
	}

	return backend, nil
}

// LoadDesignArtifacts reads design artifacts if --design-workspace is provided.
// Returns mockup path, prompt path, and design notes.
func LoadDesignArtifacts(designWorkspace, projectDir string) (mockupPath, promptPath, notes string) {
	if designWorkspace == "" {
		return "", "", ""
	}

	mockupPath, promptPath, notes = readDesignArtifacts(projectDir, designWorkspace)
	if mockupPath != "" {
		fmt.Printf("📐 Design handoff from workspace: %s\n", designWorkspace)
		fmt.Printf("   Mockup: %s\n", mockupPath)
		if promptPath != "" {
			fmt.Printf("   Prompt: %s\n", promptPath)
		}
	}
	return mockupPath, promptPath, notes
}

// BuildSpawnConfig constructs the spawn.Config from SpawnContext.
func BuildSpawnConfig(ctx *SpawnContext, phases, mode, validation, mcp string, noTrack, skipArtifactCheck bool) *spawn.Config {
	return &spawn.Config{
		Task:               ctx.Task,
		SkillName:          ctx.SkillName,
		Project:            ctx.ProjectName,
		ProjectDir:         ctx.ProjectDir,
		WorkspaceName:      ctx.WorkspaceName,
		SkillContent:       ctx.SkillContent,
		BeadsID:            ctx.BeadsID,
		Phases:             phases,
		Mode:               mode,
		Validation:         validation,
		Model:              ctx.ResolvedModel.Format(),
		MCP:                mcp,
		Tier:               ctx.Tier,
		NoTrack:            noTrack || ctx.IsOrchestrator || ctx.IsMetaOrchestrator,
		SkipArtifactCheck:  skipArtifactCheck,
		KBContext:          ctx.KBContext,
		HasInjectedModels:  ctx.HasInjectedModels,
		PrimaryModelPath:   ctx.PrimaryModelPath,
		IncludeServers:     spawn.DefaultIncludeServersForSkill(ctx.SkillName),
		GapAnalysis:        ctx.GapAnalysis,
		IsBug:              ctx.IsBug,
		ReproSteps:         ctx.ReproSteps,
		IsOrchestrator:     ctx.IsOrchestrator,
		IsMetaOrchestrator: ctx.IsMetaOrchestrator,
		UsageInfo:          ctx.UsageInfo,
		SpawnMode:          ctx.SpawnBackend,
		DesignWorkspace:    "", // Will be set by caller if needed
		DesignMockupPath:   ctx.DesignMockupPath,
		DesignPromptPath:   ctx.DesignPromptPath,
		DesignNotes:        ctx.DesignNotes,
	}
}

// ValidateAndWriteContext validates context size, writes SPAWN_CONTEXT.md, and generates prompt.
// Returns minimal prompt, or error if validation fails.
func ValidateAndWriteContext(cfg *spawn.Config, force bool) (minimalPrompt string, err error) {
	// Pre-spawn token estimation and validation
	if err := spawn.ValidateContextSize(cfg); err != nil {
		return "", fmt.Errorf("pre-spawn validation failed: %w", err)
	}

	// Warn about large contexts (but don't block)
	if shouldWarn, warning := spawn.ShouldWarnAboutSize(cfg); shouldWarn {
		fmt.Fprintf(os.Stderr, "%s", warning)
	}

	// Check for existing workspace before writing context
	// This prevents accidentally overwriting SESSION_HANDOFF.md from completed sessions
	// Note: With unique suffixes in workspace names (since Jan 2026), collisions are rare
	// but this provides an extra safety net and meaningful error messages
	if err := checkWorkspaceExists(cfg.WorkspacePath(), force); err != nil {
		return "", err
	}

	// Write SPAWN_CONTEXT.md
	if err := spawn.WriteContext(cfg); err != nil {
		return "", fmt.Errorf("failed to write spawn context: %w", err)
	}

	// Record spawn in session (if session is active)
	if sessionStore, err := session.New(""); err == nil {
		if err := sessionStore.RecordSpawn(cfg.BeadsID, cfg.SkillName, cfg.Task, cfg.ProjectDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to record spawn in session: %v\n", err)
		}
	}

	// Generate minimal prompt
	minimalPrompt = spawn.MinimalPrompt(cfg)
	return minimalPrompt, nil
}

// DispatchSpawn routes to the appropriate spawn mode function.
// Handles inline, headless, claude, and tmux modes.
func DispatchSpawn(input *SpawnInput, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task, serverURL string) error {
	// Spawn mode: inline (blocking TUI), tmux (opt-in for workers, default for orchestrators), claude (tmux), or headless (default for workers)
	if input.Inline {
		// Inline mode (blocking) - run in current terminal with TUI
		return runSpawnInline(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
	}

	// Explicit --headless flag overrides all other mode decisions
	if input.Headless {
		return runSpawnHeadless(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
	}

	// Claude mode: Use tmux + claude CLI
	if cfg.SpawnMode == "claude" {
		return runSpawnClaude(serverURL, cfg, beadsID, skillName, task, input.Attach)
	}

	// Orchestrator-type skills default to tmux mode (visible interaction)
	// Workers default to headless mode (automation-friendly)
	useTmux := input.Tmux || input.Attach || cfg.IsOrchestrator
	if useTmux {
		// Tmux mode - visible, interruptible
		// Default for orchestrator skills, opt-in for workers
		return runSpawnTmux(serverURL, cfg, minimalPrompt, beadsID, skillName, task, input.Attach)
	}

	// Default for workers: Headless mode - spawn via HTTP API (automation-friendly, no TUI overhead)
	return runSpawnHeadless(serverURL, cfg, minimalPrompt, beadsID, skillName, task)
}

// ===== Helper functions (unexported) =====

// stripANSI removes ANSI escape codes from a string for cleaner error messages.
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// formatSessionTitle formats the session title to include beads ID for matching.
// Format: "workspace-name [beads-id]" (e.g., "og-debug-orch-status-23dec [orch-go-v4mw]")
// This allows extractBeadsIDFromTitle to find agents in orch status.
func formatSessionTitle(workspaceName, beadsID string) string {
	if beadsID == "" {
		return workspaceName
	}
	return fmt.Sprintf("%s [%s]", workspaceName, beadsID)
}

// registerOrchestratorSession registers an orchestrator session in the session registry.
// This is called after successful spawn for orchestrator-type skills.
// Workers do not use the session registry - they use beads for lifecycle tracking.
func registerOrchestratorSession(cfg *spawn.Config, sessionID, task string) {
	if !cfg.IsOrchestrator && !cfg.IsMetaOrchestrator {
		return // Only register orchestrator sessions
	}

	registry := session.NewRegistry("")
	orchSession := session.OrchestratorSession{
		WorkspaceName: cfg.WorkspaceName,
		SessionID:     sessionID,
		ProjectDir:    cfg.ProjectDir,
		SpawnTime:     time.Now(),
		Goal:          task,
		Status:        "active",
	}
	if err := registry.Register(orchSession); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to register orchestrator session: %v\n", err)
	}
}

// addGapAnalysisToEventData adds gap analysis information to an event data map.
// This enables tracking of context gaps for pattern analysis and dashboard surfacing.
func addGapAnalysisToEventData(eventData map[string]interface{}, gapAnalysis *spawn.GapAnalysis) {
	if gapAnalysis == nil {
		return
	}

	eventData["gap_has_gaps"] = gapAnalysis.HasGaps
	eventData["gap_context_quality"] = gapAnalysis.ContextQuality

	if gapAnalysis.HasGaps {
		eventData["gap_should_warn"] = gapAnalysis.ShouldWarnAboutGaps()
		eventData["gap_match_total"] = gapAnalysis.MatchStats.TotalMatches
		eventData["gap_match_constraints"] = gapAnalysis.MatchStats.ConstraintCount
		eventData["gap_match_decisions"] = gapAnalysis.MatchStats.DecisionCount
		eventData["gap_match_investigations"] = gapAnalysis.MatchStats.InvestigationCount

		// Capture gap types for pattern analysis
		var gapTypes []string
		for _, gap := range gapAnalysis.Gaps {
			gapTypes = append(gapTypes, string(gap.Type))
		}
		if len(gapTypes) > 0 {
			eventData["gap_types"] = gapTypes
		}
	}
}

// addUsageInfoToEventData adds usage information to an event data map.
// This enables tracking of rate limit patterns and account utilization at spawn time.
func addUsageInfoToEventData(eventData map[string]interface{}, usageInfo *spawn.UsageInfo) {
	if usageInfo == nil {
		return
	}

	eventData["usage_5h_used"] = usageInfo.FiveHourUsed
	eventData["usage_weekly_used"] = usageInfo.SevenDayUsed
	if usageInfo.AccountEmail != "" {
		eventData["usage_account"] = usageInfo.AccountEmail
	}
	if usageInfo.AutoSwitched {
		eventData["usage_auto_switched"] = true
		eventData["usage_switch_reason"] = usageInfo.SwitchReason
	}
}

// formatContextQualitySummary formats context quality for spawn summary output.
// Returns a formatted string with visual indicators for gap severity.
// This is the "prominent" surfacing that makes gaps hard to ignore.
func formatContextQualitySummary(gapAnalysis *spawn.GapAnalysis) string {
	if gapAnalysis == nil {
		return "not checked"
	}

	quality := gapAnalysis.ContextQuality

	// Determine visual indicator and label based on quality level
	var indicator, label string
	switch {
	case quality == 0:
		indicator = "🚨"
		label = "CRITICAL - No context"
	case quality < 20:
		indicator = "⚠️"
		label = "poor"
	case quality < 40:
		indicator = "⚠️"
		label = "limited"
	case quality < 60:
		indicator = "📊"
		label = "moderate"
	case quality < 80:
		indicator = "✓"
		label = "good"
	default:
		indicator = "✓"
		label = "excellent"
	}

	// Format the summary line
	summary := fmt.Sprintf("%s %d/100 (%s)", indicator, quality, label)

	// Add match breakdown for transparency
	if gapAnalysis.MatchStats.TotalMatches > 0 {
		summary += fmt.Sprintf(" - %d matches", gapAnalysis.MatchStats.TotalMatches)
		if gapAnalysis.MatchStats.ConstraintCount > 0 {
			summary += fmt.Sprintf(" (%d constraints)", gapAnalysis.MatchStats.ConstraintCount)
		}
	}

	return summary
}

// printSpawnSummaryWithGapWarning prints the spawn summary with prominent gap warnings.
// This ensures gaps are visible in the final output, not just during context gathering.
func printSpawnSummaryWithGapWarning(gapAnalysis *spawn.GapAnalysis) {
	if gapAnalysis == nil || !gapAnalysis.ShouldWarnAboutGaps() {
		return
	}

	// Print a prominent warning box for critical gaps
	if gapAnalysis.HasCriticalGaps() || gapAnalysis.ContextQuality < 20 {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "┌─────────────────────────────────────────────────────────────┐\n")
		fmt.Fprintf(os.Stderr, "│  ⚠️  GAP WARNING: Agent spawned with limited context         │\n")
		fmt.Fprintf(os.Stderr, "├─────────────────────────────────────────────────────────────┤\n")
		fmt.Fprintf(os.Stderr, "│  Agent may compensate by guessing patterns/conventions.    │\n")
		fmt.Fprintf(os.Stderr, "│  Consider: kn decide / kn constrain / kb create            │\n")
		fmt.Fprintf(os.Stderr, "└─────────────────────────────────────────────────────────────┘\n")
	}
}

// runSpawnInline spawns the agent inline (blocking) using the HTTP API.
// Uses CreateSession + SendMessageInDirectory to properly pass x-opencode-directory
// header, ensuring the session is created in the correct project directory.
// Blocks until the session completes by watching SSE events.
func runSpawnInline(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {
	client := opencode.NewClient(serverURL)
	sessionTitle := formatSessionTitle(cfg.WorkspaceName, beadsID)

	// Step 1: Create session via HTTP API with correct directory
	// CreateSession passes x-opencode-directory header so the server uses the target project dir
	metadata := map[string]string{
		"beads_id":       cfg.BeadsID,
		"workspace_path": cfg.WorkspacePath(),
		"tier":           cfg.Tier,
		"spawn_mode":     "inline",
	}

	// Calculate TTL based on session type
	// Worker sessions: 4 hours (14400 seconds)
	// Orchestrator sessions: 0 (no expiration)
	timeTTL := 4 * 60 * 60 // 4 hours in seconds
	if cfg.IsOrchestrator {
		timeTTL = 0 // Orchestrator sessions persist until manually cleaned
	}

	session, err := client.CreateSession(sessionTitle, cfg.ProjectDir, cfg.Model, metadata, timeTTL)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	sessionID := session.ID

	// Step 2: Send the initial prompt with model selection and directory context
	// The directory header ensures the server resolves the correct project context
	if err := client.SendMessageInDirectory(sessionID, minimalPrompt, cfg.Model, cfg.ProjectDir); err != nil {
		return fmt.Errorf("failed to send prompt: %w", err)
	}

	fmt.Printf("Inline agent spawned (session: %s), waiting for completion...\n", sessionID)

	// Step 3: Wait for session to complete (blocking)
	// Watches SSE events for busy→idle transition to maintain inline mode's blocking behavior
	if err := client.WaitForSessionIdle(sessionID); err != nil {
		return fmt.Errorf("error waiting for session: %w", err)
	}

	// Write session ID to workspace file for later lookups
	if err := spawn.WriteSessionID(cfg.WorkspacePath(), sessionID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write session ID: %v\n", err)
	}

	// Register orchestrator session in registry (workers use beads instead)
	registerOrchestratorSession(cfg, sessionID, task)

	// Log the session creation
	inlineLogger := events.NewLogger(events.DefaultLogPath())
	inlineEventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"spawn_mode":          "inline",
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	if cfg.MCP != "" {
		inlineEventData["mcp"] = cfg.MCP
	}
	addGapAnalysisToEventData(inlineEventData, cfg.GapAnalysis)
	addUsageInfoToEventData(inlineEventData, cfg.UsageInfo)
	inlineEvent := events.Event{
		Type:      "session.spawned",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data:      inlineEventData,
	}
	if err := inlineLogger.Log(inlineEvent); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print spawn summary with prominent gap warning if needed
	printSpawnSummaryWithGapWarning(cfg.GapAnalysis)

	fmt.Printf("Agent completed:\n")
	fmt.Printf("  Session ID: %s\n", sessionID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	// Print context quality with visual indicators
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))

	return nil
}

// runSpawnHeadless spawns the agent using CLI subprocess without a TUI.
// This is useful for automation and daemon-driven spawns.
// Uses opencode CLI with --format json to properly support model selection
// (the HTTP API ignores the model parameter).
// Includes retry logic for transient network failures.
func runSpawnHeadless(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string) error {
	client := opencode.NewClient(serverURL)

	// Build opencode command using CLI (like inline mode) to support model selection
	// The HTTP API ignores model parameter - only CLI mode honors --model flag
	// Format title with beads ID so orch status can match sessions
	sessionTitle := formatSessionTitle(cfg.WorkspaceName, beadsID)

	// Use retry logic for transient failures (network issues, server temporarily unavailable)
	retryCfg := spawn.DefaultRetryConfig()
	result, retryResult := spawn.Retry(retryCfg, func() (*headlessSpawnResult, error) {
		return startHeadlessSession(client, serverURL, sessionTitle, minimalPrompt, cfg)
	})

	if retryResult.LastErr != nil {
		// Wrap the error with user-friendly message and recovery guidance
		spawnErr := spawn.WrapSpawnError(retryResult.LastErr, "Headless spawn failed")
		if retryResult.Attempts > 1 {
			fmt.Fprintf(os.Stderr, "Spawn failed after %d attempts\n", retryResult.Attempts)
		}
		// Print formatted error with recovery guidance
		fmt.Fprintf(os.Stderr, "\n%s\n", spawn.FormatSpawnError(spawnErr))
		return spawnErr
	}

	if retryResult.Attempts > 1 {
		fmt.Printf("Spawn succeeded after %d attempts\n", retryResult.Attempts)
	}

	sessionID := result.SessionID

	// Write session ID to workspace file for later lookups
	if err := spawn.WriteSessionID(cfg.WorkspacePath(), sessionID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write session ID: %v\n", err)
	}

	// Register orchestrator session in registry (workers use beads instead)
	registerOrchestratorSession(cfg, sessionID, task)

	// Log the session creation
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"session_id":          sessionID,
		"spawn_mode":          "headless",
		"model":               cfg.Model,
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	if retryResult.Attempts > 1 {
		eventData["retry_attempts"] = retryResult.Attempts
	}
	if cfg.MCP != "" {
		eventData["mcp"] = cfg.MCP
	}
	addGapAnalysisToEventData(eventData, cfg.GapAnalysis)
	addUsageInfoToEventData(eventData, cfg.UsageInfo)
	event := events.Event{
		Type:      "session.spawned",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Print spawn summary with prominent gap warning if needed
	printSpawnSummaryWithGapWarning(cfg.GapAnalysis)

	fmt.Printf("Spawned agent (headless):\n")
	fmt.Printf("  Session ID: %s\n", sessionID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	fmt.Printf("  Model:      %s\n", cfg.Model)
	if cfg.MCP != "" {
		fmt.Printf("  MCP:        %s\n", cfg.MCP)
	}
	if cfg.NoTrack {
		fmt.Printf("  Tracking:   disabled (--no-track)\n")
	}
	// Print context quality with visual indicators
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))

	return nil
}

// startHeadlessSession creates an OpenCode session via HTTP API and sends the initial prompt.
// Uses HTTP API instead of CLI subprocess to properly set the session's working directory
// via x-opencode-directory header. This fixes cross-project spawns where --workdir differs
// from the orchestrator's CWD.
// Model selection is handled per-message by SendMessageInDirectory (providerID/modelID format).
func startHeadlessSession(client *opencode.Client, serverURL, sessionTitle, minimalPrompt string, cfg *spawn.Config) (*headlessSpawnResult, error) {
	// Step 1: Create session via HTTP API with correct directory
	// CreateSession passes x-opencode-directory header so the server uses the target project dir
	metadata := map[string]string{
		"beads_id":       cfg.BeadsID,
		"workspace_path": cfg.WorkspacePath(),
		"tier":           cfg.Tier,
		"spawn_mode":     "headless",
	}

	// Calculate TTL based on session type
	// Worker sessions: 4 hours (14400 seconds)
	// Orchestrator sessions: 0 (no expiration)
	timeTTL := 4 * 60 * 60 // 4 hours in seconds
	if cfg.IsOrchestrator {
		timeTTL = 0 // Orchestrator sessions persist until manually cleaned
	}

	session, err := client.CreateSession(sessionTitle, cfg.ProjectDir, cfg.Model, metadata, timeTTL)
	if err != nil {
		return nil, spawn.WrapSpawnError(err, "Failed to create session via API")
	}

	// Step 2: Send the initial prompt with model selection and directory context
	// The directory header ensures the server resolves the correct project context
	if err := client.SendMessageInDirectory(session.ID, minimalPrompt, cfg.Model, cfg.ProjectDir); err != nil {
		return nil, spawn.WrapSpawnError(err, "Failed to send prompt to session")
	}

	return &headlessSpawnResult{
		SessionID: session.ID,
	}, nil
}

// runSpawnTmux spawns the agent in a tmux window (interactive, returns immediately).
// Creates a tmux window in workers-{project} session (or orchestrator session for orchestrator skills).
func runSpawnTmux(serverURL string, cfg *spawn.Config, minimalPrompt, beadsID, skillName, task string, attach bool) error {
	var sessionName string
	var err error

	// Meta-orchestrators and orchestrators go into 'orchestrator' tmux session
	// Workers go into 'workers-{project}' session
	if cfg.IsMetaOrchestrator || cfg.IsOrchestrator {
		sessionName, err = tmux.EnsureOrchestratorSession()
	} else {
		sessionName, err = tmux.EnsureWorkersSession(cfg.Project, cfg.ProjectDir)
	}
	if err != nil {
		return fmt.Errorf("failed to ensure tmux session: %w", err)
	}

	// Build window name with emoji and beads ID
	windowName := tmux.BuildWindowName(cfg.WorkspaceName, cfg.SkillName, beadsID)

	// Create new tmux window
	windowTarget, windowID, err := tmux.CreateWindow(sessionName, windowName, cfg.ProjectDir)
	if err != nil {
		return fmt.Errorf("failed to create tmux window: %w", err)
	}

	// Build opencode command using tmux package
	opencodeCmd := tmux.BuildOpencodeAttachCommand(&tmux.OpencodeAttachConfig{
		ServerURL:  serverURL,
		ProjectDir: cfg.ProjectDir,
		Model:      cfg.Model,
	})

	// Send command and execute
	if err := tmux.SendKeys(windowTarget, opencodeCmd); err != nil {
		return fmt.Errorf("failed to send opencode command: %w", err)
	}
	if err := tmux.SendEnter(windowTarget); err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	// Wait for OpenCode TUI to be ready
	waitCfg := tmux.DefaultWaitConfig()
	if err := tmux.WaitForOpenCodeReady(windowTarget, waitCfg); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	// Capture session ID from API with retry (OpenCode may not have registered yet)
	// Uses 3 attempts with 500ms initial delay, doubling each time (500ms, 1s, 2s)
	// Matches by directory + creation time (within 30s), not by title
	client := opencode.NewClient(serverURL)
	sessionID, _ := client.FindRecentSessionWithRetry(cfg.ProjectDir, 3, 500*time.Millisecond)
	// Note: We silently ignore errors here since window_id is sufficient for tmux monitoring

	// Send prompt
	sendCfg := tmux.DefaultSendPromptConfig()
	time.Sleep(sendCfg.PostReadyDelay)
	if err := tmux.SendKeysLiteral(windowTarget, minimalPrompt); err != nil {
		return fmt.Errorf("failed to send prompt: %w", err)
	}
	if err := tmux.SendEnter(windowTarget); err != nil {
		return fmt.Errorf("failed to send enter: %w", err)
	}

	// Write session ID to workspace file for later lookups
	if sessionID != "" {
		if err := spawn.WriteSessionID(cfg.WorkspacePath(), sessionID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write session ID: %v\n", err)
		}
	}

	// Register orchestrator session in registry (workers use beads instead)
	registerOrchestratorSession(cfg, sessionID, task)

	// Log the session creation
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"session_id":          sessionID,
		"window":              windowTarget,
		"window_id":           windowID,
		"session_name":        sessionName,
		"spawn_mode":          "tmux",
		"model":               cfg.Model,
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	if cfg.MCP != "" {
		eventData["mcp"] = cfg.MCP
	}
	addGapAnalysisToEventData(eventData, cfg.GapAnalysis)
	addUsageInfoToEventData(eventData, cfg.UsageInfo)
	event := events.Event{
		Type:      "session.spawned",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Focus the newly created window
	selectCmd := exec.Command("tmux", "select-window", "-t", windowTarget)
	if err := selectCmd.Run(); err != nil {
		// Non-fatal - window was created successfully
		fmt.Fprintf(os.Stderr, "Warning: failed to focus window: %v\n", err)
	}

	// Print spawn summary with prominent gap warning if needed
	printSpawnSummaryWithGapWarning(cfg.GapAnalysis)

	fmt.Printf("Spawned agent in tmux:\n")
	fmt.Printf("  Session:    %s\n", sessionName)
	if sessionID != "" {
		fmt.Printf("  Session ID: %s\n", sessionID)
	}
	fmt.Printf("  Window:     %s\n", windowTarget)
	fmt.Printf("  Window ID:  %s\n", windowID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	fmt.Printf("  Model:      %s\n", cfg.Model)
	if cfg.MCP != "" {
		fmt.Printf("  MCP:        %s\n", cfg.MCP)
	}
	if cfg.NoTrack {
		fmt.Printf("  Tracking:   disabled (--no-track)\n")
	}
	// Print context quality with visual indicators
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))

	// Attach if requested
	if attach {
		if err := tmux.Attach(windowTarget); err != nil {
			return fmt.Errorf("failed to attach to tmux: %w", err)
		}
	}

	return nil
}

// runSpawnClaude spawns the agent using Claude Code CLI in a tmux window.
func runSpawnClaude(serverURL string, cfg *spawn.Config, beadsID, skillName, task string, attach bool) error {
	result, err := spawn.SpawnClaude(cfg)
	if err != nil {
		return err
	}

	// Register orchestrator session in registry if needed
	registerOrchestratorSession(cfg, "", task)

	// Log the session creation
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"skill":               skillName,
		"task":                task,
		"workspace":           cfg.WorkspaceName,
		"beads_id":            beadsID,
		"window":              result.Window,
		"window_id":           result.WindowID,
		"spawn_mode":          "claude",
		"no_track":            cfg.NoTrack,
		"skip_artifact_check": cfg.SkipArtifactCheck,
	}
	addGapAnalysisToEventData(eventData, cfg.GapAnalysis)
	addUsageInfoToEventData(eventData, cfg.UsageInfo)
	event := events.Event{
		Type:      "session.spawned",
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	// Focus the newly created window
	selectCmd := exec.Command("tmux", "select-window", "-t", result.Window)
	if err := selectCmd.Run(); err != nil {
		// Non-fatal
		fmt.Fprintf(os.Stderr, "Warning: failed to focus window: %v\n", err)
	}

	// Print spawn summary with prominent gap warning if needed
	printSpawnSummaryWithGapWarning(cfg.GapAnalysis)

	fmt.Printf("Spawned agent in Claude mode (tmux):\n")
	fmt.Printf("  Window:     %s\n", result.Window)
	fmt.Printf("  Window ID:  %s\n", result.WindowID)
	fmt.Printf("  Workspace:  %s\n", cfg.WorkspaceName)
	fmt.Printf("  Beads ID:   %s\n", beadsID)
	// Print context quality with visual indicators
	fmt.Printf("  Context:    %s\n", formatContextQualitySummary(cfg.GapAnalysis))

	// Attach if requested
	if attach {
		if err := tmux.Attach(result.Window); err != nil {
			return fmt.Errorf("failed to attach to tmux: %w", err)
		}
	}

	return nil
}

// determineBeadsID determines the beads ID to use for an agent.
// It returns an error if beads issue creation fails and --no-track is not set.
// The createBeadsFn parameter allows for dependency injection in tests.
func determineBeadsID(projectName, skillName, task, spawnIssue string, spawnNoTrack bool, createBeadsFn func(string, string, string) (string, error)) (string, error) {
	// If explicit issue ID provided via --issue flag, resolve it to full ID
	if spawnIssue != "" {
		return resolveShortBeadsID(spawnIssue)
	}

	// If --no-track flag is set, generate a local-only ID
	if spawnNoTrack {
		return fmt.Sprintf("%s-untracked-%d", projectName, time.Now().Unix()), nil
	}

	// Create a new beads issue (default behavior)
	beadsID, err := createBeadsFn(projectName, skillName, task)
	if err != nil {
		return "", fmt.Errorf("failed to create beads issue: %w", err)
	}

	return beadsID, nil
}

// CreateBeadsIssue creates a new beads issue for tracking the agent.
// It uses the beads RPC client when available, falling back to the bd CLI.
func CreateBeadsIssue(projectName, skillName, task string) (string, error) {
	// Build issue title
	title := fmt.Sprintf("[%s] %s: %s", projectName, skillName, truncate(task, 50))

	// Try RPC client first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath)
		if err := client.Connect(); err == nil {
			defer client.Close()

			issue, err := client.Create(&beads.CreateArgs{
				Title:     title,
				IssueType: "task",
				Priority:  2, // Default P2
			})
			if err == nil {
				return issue.ID, nil
			}
			// Fall through to CLI fallback on RPC error
		}
	}

	// Fallback to CLI
	issue, err := beads.FallbackCreate(title, "", "task", 2, nil)
	if err != nil {
		return "", err
	}

	return issue.ID, nil
}

// (ensureOrchScaffolding moved back to cmd/orch/spawn_cmd.go to avoid circular dependencies)

// dirExists returns true if the path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// checkWorkspaceExists verifies if a workspace already exists and has content.
// Returns an error if the workspace contains SPAWN_CONTEXT.md or SESSION_HANDOFF.md
// (indicating an active or completed session), unless force is true.
// This prevents accidental data loss from overwriting existing session artifacts.
func checkWorkspaceExists(workspacePath string, force bool) error {
	// Check if workspace directory exists
	if !dirExists(workspacePath) {
		return nil // Workspace doesn't exist, safe to create
	}

	// Check for critical files that indicate an active or completed session
	criticalFiles := []string{
		"SPAWN_CONTEXT.md",
		"SESSION_HANDOFF.md",
		"ORCHESTRATOR_CONTEXT.md",
	}

	for _, file := range criticalFiles {
		filePath := filepath.Join(workspacePath, file)
		if _, err := os.Stat(filePath); err == nil {
			if force {
				fmt.Fprintf(os.Stderr, "Warning: Overwriting existing workspace at %s (--force)\n", workspacePath)
				return nil
			}
			return fmt.Errorf("workspace already exists with %s at %s\n\nThis indicates an existing session. Use --force to overwrite or spawn with a different task", file, workspacePath)
		}
	}

	return nil // Directory exists but has no critical files, safe to reuse
}

// extractPrimaryModelPath extracts the file path of the first model from KB context result.
// Returns empty string if no model paths found.
func extractPrimaryModelPath(formatResult *spawn.KBContextFormatResult) string {
	if formatResult == nil {
		return ""
	}
	return formatResult.PrimaryModelPath
}

// runPreSpawnKBCheck runs kb context check before spawning an agent.
// Returns formatted context string to include in SPAWN_CONTEXT.md, or empty string if no matches.
// Also performs gap analysis and displays warnings for sparse or missing context.
func runPreSpawnKBCheck(task string) string {
	result := runPreSpawnKBCheckFull(task)
	return result.Context
}

// runPreSpawnKBCheckFull runs kb context check with full gap analysis results.
// This allows callers to access gap analysis for gating decisions.
func runPreSpawnKBCheckFull(task string) *GapCheckResult {
	gcr := &GapCheckResult{}

	// Extract keywords from task description
	// Try with 3 keywords first (more specific), fall back to 1 keyword (more broad)
	keywords := spawn.ExtractKeywords(task, 3)
	if keywords == "" {
		// Perform gap analysis even when no keywords extracted
		gcr.GapAnalysis = spawn.AnalyzeGaps(nil, task)
		if gcr.GapAnalysis.ShouldWarnAboutGaps() {
			// Use prominent warning format for better visibility
			fmt.Fprintf(os.Stderr, "%s", gcr.GapAnalysis.FormatProminentWarning())
		}
		return gcr
	}

	fmt.Printf("Checking kb context for: %q\n", keywords)

	// Run kb context check
	result, err := spawn.RunKBContextCheck(keywords)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: kb context check failed: %v\n", err)
		return gcr
	}

	// If no matches with multiple keywords, try with just the first keyword
	if result == nil || !result.HasMatches {
		firstKeyword := spawn.ExtractKeywords(task, 1)
		if firstKeyword != "" && firstKeyword != keywords {
			fmt.Printf("Trying broader search for: %q\n", firstKeyword)
			result, err = spawn.RunKBContextCheck(firstKeyword)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: kb context check failed: %v\n", err)
				return gcr
			}
		}
	}

	// Perform gap analysis to detect context gaps
	gcr.GapAnalysis = spawn.AnalyzeGaps(result, keywords)
	if gcr.GapAnalysis.ShouldWarnAboutGaps() {
		// Use prominent warning format for better visibility
		fmt.Fprintf(os.Stderr, "%s", gcr.GapAnalysis.FormatProminentWarning())
	}

	if result == nil || !result.HasMatches {
		fmt.Println("No prior knowledge found.")
		return gcr
	}

	// Always include kb context in spawn - the orchestrator has already decided to spawn
	// No interactive prompt needed; context is automatically included
	fmt.Printf("Found %d relevant context entries - including in spawn context.\n", len(result.Matches))

	// Format context with limit and capture full result (includes HasInjectedModels)
	formatResult := spawn.FormatContextForSpawnWithLimit(result, spawn.MaxKBContextChars)
	gcr.FormatResult = formatResult

	// Include gap summary in spawn context if there are significant gaps
	contextContent := formatResult.Content
	if gapSummary := gcr.GapAnalysis.FormatGapSummary(); gapSummary != "" {
		contextContent = gapSummary + "\n\n" + contextContent
	}

	gcr.Context = contextContent
	return gcr
}

// checkGapGating checks if spawn should be blocked due to context gaps.
// Returns an error if spawn should be blocked, nil otherwise.
func checkGapGating(gapAnalysis *spawn.GapAnalysis, gateEnabled, skipGate bool, threshold int) error {
	// Skip gating if not enabled or explicitly bypassed
	if !gateEnabled || skipGate {
		return nil
	}

	// No gap analysis means no gating
	if gapAnalysis == nil {
		return nil
	}

	// Check if quality is below threshold
	if threshold <= 0 {
		threshold = spawn.DefaultGateThreshold
	}

	if gapAnalysis.ShouldBlockSpawn(threshold) {
		// Display the block message
		fmt.Fprintf(os.Stderr, "%s", gapAnalysis.FormatGateBlockMessage())
		return fmt.Errorf("spawn blocked: context quality %d is below threshold %d", gapAnalysis.ContextQuality, threshold)
	}

	return nil
}

// recordGapForLearning records a gap event for the learning loop.
// This builds up a history of gaps that can be used to suggest improvements.
func recordGapForLearning(gapAnalysis *spawn.GapAnalysis, skill, task string) {
	// Load existing tracker
	tracker, err := spawn.LoadTracker()
	if err != nil {
		// Don't fail spawn for learning loop errors
		fmt.Fprintf(os.Stderr, "Warning: failed to load gap tracker: %v\n", err)
		return
	}

	// Record the gap
	tracker.RecordGap(gapAnalysis, skill, task)

	// Check for recurring patterns and display suggestions
	suggestions := tracker.FindRecurringGaps()
	if len(suggestions) > 0 {
		// Only show suggestions if there are high-priority ones
		hasHighPriority := false
		for _, s := range suggestions {
			if s.Priority == "high" && s.Count >= spawn.RecurrenceThreshold {
				hasHighPriority = true
				break
			}
		}
		if hasHighPriority {
			fmt.Fprintf(os.Stderr, "%s", spawn.FormatSuggestions(suggestions))
		}
	}

	// Save tracker
	if err := tracker.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save gap tracker: %v\n", err)
	}
}

// isInfrastructureWork detects if a task involves infrastructure work that requires
// the escape hatch (--backend claude --tmux) to prevent agents from killing themselves
// when they restart the OpenCode server.
//
// Detection strategy:
// - Check task description for infrastructure keywords
// - Check beads issue title/description if spawning from issue
// - Check for file paths that match infrastructure patterns
//
// Returns true if infrastructure work is detected, false otherwise.
func isInfrastructureWork(task string, beadsID string) bool {
	// Infrastructure keywords to check for
	infrastructureKeywords := []string{
		"opencode",
		"orch-go",
		"pkg/spawn",
		"pkg/opencode",
		"pkg/verify",
		"pkg/state",
		"cmd/orch",
		"spawn_cmd.go",
		"serve.go",
		"status.go",
		"main.go",
		"dashboard",
		"agent-card",
		"agents.ts",
		"daemon.ts",
		"skillc",
		"skill.yaml",
		"SPAWN_CONTEXT",
		"spawn system",
		"spawn logic",
		"spawn template",
		"orchestration infrastructure",
		"orchestration system",
	}

	// Check task description (case-insensitive)
	taskLower := strings.ToLower(task)
	for _, keyword := range infrastructureKeywords {
		if strings.Contains(taskLower, keyword) {
			return true
		}
	}

	// Check beads issue if available
	if beadsID != "" {
		issue, err := verify.GetIssue(beadsID)
		if err == nil {
			// Check title
			titleLower := strings.ToLower(issue.Title)
			for _, keyword := range infrastructureKeywords {
				if strings.Contains(titleLower, keyword) {
					return true
				}
			}
			// Check description
			descLower := strings.ToLower(issue.Description)
			for _, keyword := range infrastructureKeywords {
				if strings.Contains(descLower, keyword) {
					return true
				}
			}
		}
	}

	return false
}

// readDesignArtifacts reads design artifacts from a ui-design-session workspace.
// Returns mockup path, prompt path, and design notes from SYNTHESIS.md.
// If the workspace doesn't exist or artifacts are missing, returns empty strings.
func readDesignArtifacts(projectDir, designWorkspace string) (mockupPath, promptPath, designNotes string) {
	workspacePath := filepath.Join(projectDir, ".orch", "workspace", designWorkspace)

	// Check if workspace exists
	if _, err := os.Stat(workspacePath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: design workspace not found: %s\n", workspacePath)
		return "", "", ""
	}

	// Look for mockup in screenshots/ directory
	// Convention: approved.png or any .png file
	screenshotsPath := filepath.Join(workspacePath, "screenshots")
	if entries, err := os.ReadDir(screenshotsPath); err == nil {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".png") {
				mockupPath = filepath.Join(screenshotsPath, entry.Name())
				// Check for corresponding .prompt.md file
				promptName := strings.TrimSuffix(entry.Name(), ".png") + ".prompt.md"
				promptPath = filepath.Join(screenshotsPath, promptName)
				if _, err := os.Stat(promptPath); err != nil {
					promptPath = "" // Prompt file doesn't exist
				}
				break // Use first .png found
			}
		}
	}

	// Read design notes from SYNTHESIS.md
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	if content, err := os.ReadFile(synthesisPath); err == nil {
		// Extract relevant sections from SYNTHESIS.md
		// For now, just include the TLDR and Knowledge sections
		designNotes = extractDesignNotes(string(content))
	}

	return mockupPath, promptPath, designNotes
}

// extractDesignNotes extracts relevant sections from SYNTHESIS.md for design handoff.
// Returns TLDR and Knowledge sections which contain key design insights.
func extractDesignNotes(content string) string {
	var notes strings.Builder

	// Extract TLDR section
	if tldr := extractSection(content, "## TLDR"); tldr != "" {
		notes.WriteString("**Design TLDR:**\n")
		notes.WriteString(tldr)
		notes.WriteString("\n\n")
	}

	// Extract Knowledge section
	if knowledge := extractSection(content, "## Knowledge"); knowledge != "" {
		notes.WriteString("**Design Knowledge:**\n")
		notes.WriteString(knowledge)
	}

	return notes.String()
}

// extractSection extracts content between a section header and the next ## header.
// Returns empty string if section not found.
func extractSection(content, sectionHeader string) string {
	lines := strings.Split(content, "\n")
	var sectionLines []string
	inSection := false

	for _, line := range lines {
		if strings.HasPrefix(line, sectionHeader) {
			inSection = true
			continue
		}
		if inSection && strings.HasPrefix(line, "##") {
			break // Reached next section
		}
		if inSection {
			sectionLines = append(sectionLines, line)
		}
	}

	if len(sectionLines) == 0 {
		return ""
	}

	return strings.TrimSpace(strings.Join(sectionLines, "\n"))
}

// truncate truncates a string to a maximum length, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// resolveShortBeadsID resolves a short beads ID (e.g., "123") to full ID (e.g., "orch-go-123").
// If already in full format, returns as-is.
func resolveShortBeadsID(id string) (string, error) {
	// If already in full format, return as-is
	if strings.Contains(id, "-") {
		return id, nil
	}

	// Otherwise, we need to get the project name from current directory
	projectDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	projectName := filepath.Base(projectDir)

	return fmt.Sprintf("%s-%s", projectName, id), nil
}
