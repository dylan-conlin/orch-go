// Package orch provides orchestration-level utilities for agent spawn management.
// This includes spawn pipeline functions extracted from cmd/orch/spawn_cmd.go.
package orch

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	OrientationFrame   string // Separate context from task title (e.g., issue description)
	SkillName          string
	ProjectDir         string
	ProjectName        string
	WorkspaceName      string
	SkillContent       string
	BeadsID            string
	IsOrchestrator     bool
	IsMetaOrchestrator bool
	ResolvedModel      model.ModelSpec
	ResolvedSettings   spawn.ResolvedSpawnSettings
	KBContext          string
	GapAnalysis        *spawn.GapAnalysis
	HasInjectedModels  bool
	PrimaryModelPath   string
	CrossRepoModelDir  string
	IsBug              bool
	ReproSteps         string
	ReworkFeedback     string
	ReworkNumber       int
	PriorSynthesis     string
	PriorWorkspace     string
	UsageInfo          *spawn.UsageInfo
	Account            string
	AccountConfigDir   string
	SpawnBackend       string
	Tier               string
	VerifyLevel        string // Explicit verification level override (V0-V3)
	Scope              string
	HotspotArea        bool
	HotspotFiles       []string
	DesignMockupPath   string
	DesignPromptPath   string
	DesignNotes        string
	// BeadsDir is the absolute path to the .beads/ directory for cross-repo spawns.
	// When the beads issue is in a different project than the agent's working directory,
	// this is set so BEADS_DIR env var can be injected into the Claude CLI launch command.
	BeadsDir string
}

// ResolvedSpawnResult holds resolved spawn settings and the parsed model spec.
type ResolvedSpawnResult struct {
	Settings spawn.ResolvedSpawnSettings
	Model    model.ModelSpec
}

// GapCheckResult contains the results of a pre-spawn gap check.
type GapCheckResult struct {
	Context      string                       // Formatted context to include in SPAWN_CONTEXT.md
	GapAnalysis  *spawn.GapAnalysis           // Gap analysis results for further processing
	Blocked      bool                         // True if spawn should be blocked due to gaps
	BlockReason  string                       // Reason for blocking (if Blocked is true)
	FormatResult *spawn.KBContextFormatResult // Full format result including HasInjectedModels
}

// sessionScopeRegex moved to pkg/spawn.regexSessionScope (canonical location)

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

// DetermineSpawnTier determines the spawn tier based on flags, config, task scope signals,
// and skill defaults.
// Priority: --light flag > --full flag > userconfig.default_tier > task scope signals > skill default
func DetermineSpawnTier(skillName, task string, lightFlag, fullFlag bool) string {
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
	// Use task scope signals to upgrade to full tier when needed
	if inferredTier := inferTierFromTask(task); inferredTier != "" {
		return inferredTier
	}
	// Fall back to skill default
	return spawn.DefaultTierForSkill(skillName)
}

func inferTierFromTask(task string) string {
	if task == "" {
		return ""
	}

	if scope := spawn.ParseScopeFromTask(task); scope != "" {
		switch scope {
		case "medium", "large", "full", "4-6h", "4-6h+", "2-4h":
			return spawn.TierFull
		}
	}

	lower := strings.ToLower(task)

	score := 0
	if containsAny(lower, []string{
		"create package",
		"new package",
		"create module",
		"new module",
		"new pkg/",
		"create pkg/",
		"new package/",
		"create package/",
	}) {
		score += 2
	}
	if containsAny(lower, []string{
		"comprehensive tests",
		"test suite",
		"integration tests",
		"unit tests",
		"tests for",
		"add tests",
	}) {
		score++
	}

	if score >= 2 {
		return spawn.TierFull
	}

	return ""
}

// parseSessionScope is now delegated to spawn.ParseScopeFromTask
// to avoid duplicating regex and parsing logic across packages.

func containsAny(text string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
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
// Returns usage check result for telemetry, hotspot result for context injection,
// agreements result for telemetry, or error if any check fails.
// hotspotCheckFunc and agreementsCheckFunc are passed from cmd/orch to avoid circular dependencies.
func RunPreFlightChecks(input *SpawnInput, preCheckDir string, bypassTriage, bypassVerification, forceHotspot bool, architectRef, bypassReason string, maxAgents int, extractBeadsIDFunc func(string) string, hotspotCheckFunc func(string, string) (*gates.HotspotResult, error), agreementsCheckFunc func(string) (*gates.AgreementsResult, error)) (*gates.UsageCheckResult, *gates.HotspotResult, *gates.AgreementsResult, error) {
	// Check for --bypass-triage flag (required for manual spawns)
	// Daemon-driven spawns skip this check (issue already triaged)
	if err := gates.CheckTriageBypass(input.DaemonDriven, bypassTriage, input.SkillName, input.Task); err != nil {
		return nil, nil, nil, err
	}

	// Log the triage bypass for Phase 2 review (only for manual bypasses, not daemon-driven)
	if !input.DaemonDriven && bypassTriage {
		gates.LogTriageBypass(input.SkillName, input.Task)
	}

	// Check verification gate (Phase 3: Session Continuity Gate)
	// Block spawn if unverified Tier 1 work exists (prevents cascade pattern)
	// Independent parallel work can use --bypass-verification to override
	if err := gates.CheckVerificationGate(bypassVerification, bypassReason); err != nil {
		return nil, nil, nil, err
	}

	// Check concurrency limit before spawning
	if err := gates.CheckConcurrency(input.ServerURL, maxAgents, extractBeadsIDFunc); err != nil {
		return nil, nil, nil, err
	}

	// Proactive rate limit monitoring: warn at 80%, block at 95%
	usageCheckResult, usageErr := gates.CheckRateLimit()
	if usageErr != nil {
		// usageErr contains formatted blocking message
		return nil, nil, nil, usageErr
	}

	// STRATEGIC-FIRST ORCHESTRATION: Check for hotspots in task target area
	// Blocks implementation skills (feature-impl, systematic-debugging) on CRITICAL files (>1500 lines).
	// Exempt: architect, investigation, capture-knowledge, codebase-audit (read-only/strategic skills).
	// Override: --force-hotspot --architect-ref <issue-id> bypasses the block with proof of architect review.
	// Auto-detection: searches for closed architect issues covering the critical files.
	var hotspotResult *gates.HotspotResult
	if hotspotCheckFunc != nil {
		// Always build verifier — needed for both explicit --architect-ref and auto-detection
		architectVerifier := buildArchitectVerifier()
		// Build finder for auto-detection of prior architect reviews
		architectFinder := buildArchitectFinder()
		var err error
		hotspotResult, err = gates.CheckHotspot(preCheckDir, input.Task, input.SkillName, input.DaemonDriven, forceHotspot, architectRef, hotspotCheckFunc, architectVerifier, architectFinder)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	// KB AGREEMENTS CHECK: Warning-only gate (Phase 3).
	// Displays agreement failures as warnings but never blocks spawn.
	// Daemon-driven spawns suppress output but still return result for telemetry.
	var agreementsResult *gates.AgreementsResult
	if agreementsCheckFunc != nil {
		agreementsResult, _ = gates.CheckAgreements(preCheckDir, input.DaemonDriven, agreementsCheckFunc)
	}

	return usageCheckResult, hotspotResult, agreementsResult, nil
}

// buildArchitectVerifier creates a function that validates an architect issue reference.
// Checks: issue exists, was spawned with architect skill, and is closed.
func buildArchitectVerifier() gates.ArchitectVerifier {
	return func(issueID string) error {
		issue, err := verify.GetIssue(issueID)
		if err != nil {
			return fmt.Errorf("--architect-ref %s: issue not found", issueID)
		}

		// Check if issue was spawned with architect skill
		if !isArchitectIssue(issue) {
			return fmt.Errorf("--architect-ref %s: not an architect issue (type=%s)", issueID, issue.IssueType)
		}

		// Check if architect review is complete
		if issue.Status != "closed" {
			return fmt.Errorf("--architect-ref %s: architect review not complete (status=%s)", issueID, issue.Status)
		}

		return nil
	}
}

// buildArchitectFinder creates a function that searches for closed architect issues
// covering the given critical files. Used for automatic hotspot gate bypass when
// an architect has already reviewed the area.
func buildArchitectFinder() gates.ArchitectFinder {
	return func(criticalFiles []string) (string, error) {
		return FindPriorArchitectReview(criticalFiles)
	}
}

// FindPriorArchitectReview searches for a closed architect issue that reviewed
// any of the given critical files. Returns the issue ID if found, empty string if none.
//
// Matching strategy: extract basenames and path components from critical files,
// then check closed architect issue titles for mentions of these terms.
func FindPriorArchitectReview(criticalFiles []string) (string, error) {
	if len(criticalFiles) == 0 {
		return "", nil
	}

	// Build search terms from critical file paths
	searchTerms := extractSearchTerms(criticalFiles)
	if len(searchTerms) == 0 {
		return "", nil
	}

	// Query beads for closed architect issues
	socketPath, err := beads.FindSocketPath("")
	if err != nil {
		return "", nil // Graceful degradation
	}
	client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
	defer client.Close()

	issues, err := client.List(&beads.ListArgs{
		Status: "closed",
		Labels: []string{"skill:architect"},
	})
	if err != nil {
		return "", nil // Graceful degradation
	}

	// Also search by title pattern for architect issues without the label
	titleIssues, err := client.List(&beads.ListArgs{
		Status: "closed",
		Title:  "architect:",
	})
	if err == nil {
		// Merge, dedup by ID
		seen := make(map[string]bool)
		for _, i := range issues {
			seen[i.ID] = true
		}
		for _, i := range titleIssues {
			if !seen[i.ID] {
				issues = append(issues, i)
			}
		}
	}

	// Find the first issue whose title matches any search term
	for _, issue := range issues {
		titleLower := strings.ToLower(issue.Title)
		for _, term := range searchTerms {
			if strings.Contains(titleLower, term) {
				return issue.ID, nil
			}
		}
	}

	return "", nil
}

// extractSearchTerms builds a list of search terms from critical file paths.
// For "cmd/orch/main.go" → ["main.go", "cmd/orch/main.go"]
// For "pkg/daemon/daemon.go" → ["daemon.go", "pkg/daemon/daemon.go"]
// For "pkg/orch/extraction.go" → ["extraction.go", "pkg/orch/extraction.go"]
func extractSearchTerms(criticalFiles []string) []string {
	seen := make(map[string]bool)
	var terms []string

	for _, file := range criticalFiles {
		normalized := strings.ToLower(strings.TrimSpace(file))
		if normalized == "" {
			continue
		}

		// Add the full path
		if !seen[normalized] {
			terms = append(terms, normalized)
			seen[normalized] = true
		}

		// Add the basename (e.g., "daemon.go")
		parts := strings.Split(normalized, "/")
		basename := parts[len(parts)-1]
		// Strip .go extension for broader matching (e.g., "extraction" matches "extraction.go structure analysis")
		nameOnly := strings.TrimSuffix(basename, ".go")
		if nameOnly != "" && !seen[nameOnly] {
			terms = append(terms, nameOnly)
			seen[nameOnly] = true
		}
	}

	return terms
}

// isArchitectIssue returns true if the issue was spawned with the architect skill.
// Checks for skill:architect label or "architect:" in the title (from CreateBeadsIssue pattern).
func isArchitectIssue(issue *verify.Issue) bool {
	for _, label := range issue.Labels {
		if label == "skill:architect" {
			return true
		}
	}
	// Check title pattern: "[project] architect: task"
	if strings.Contains(strings.ToLower(issue.Title), "architect:") {
		return true
	}
	return false
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
// workspaceName is used to set the assignee on the beads issue when status is set to in_progress.
// Returns final beads ID (empty if untracked), or error if setup fails.
func SetupBeadsTracking(skillName, task, projectName, beadsIssueFlag string, isOrchestrator, isMetaOrchestrator bool, serverURL string, noTrack bool, workspaceName string, createBeadsFn func(string, string, string) (string, error)) (string, error) {
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
						fmt.Fprintf(os.Stderr, "Note: found stale session %s for issue %s (no activity in 30m)\n", shortID(s.ID), beadsID)
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

	// Update beads issue status to in_progress (for any tracked issue, including auto-created ones)
	// Skip for orchestrators since they don't use beads tracking
	// Uses beadsID (not beadsIssueFlag) to also cover auto-created issues from determineBeadsID
	if !noTrack && !skipBeadsForOrchestrator && beadsID != "" {
		if err := verify.UpdateIssueStatus(beadsID, "in_progress"); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update beads issue status: %v\n", err)
			// Continue anyway
		}
		// Set assignee to workspace name so dashboard shows which agent is working on this issue
		if workspaceName != "" {
			if err := verify.UpdateIssueAssignee(beadsID, workspaceName); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to set assignee on beads issue: %v\n", err)
				// Continue anyway - assignee is supplementary metadata
			}
		}
	}

	return beadsID, nil
}

// ResolveAndValidateModel resolves model aliases and validates model choice.
// Returns error if flash model is requested (unsupported).
func ResolveAndValidateModel(modelFlag string) (model.ModelSpec, error) {
	// Load user config for custom model aliases
	cfg, _ := userconfig.Load()
	var configModels map[string]string
	if cfg != nil {
		configModels = cfg.Models
	}

	// If no model flag specified, check config default_model before hardcoded default
	effectiveSpec := modelFlag
	if effectiveSpec == "" && cfg != nil && cfg.DefaultModel != "" {
		effectiveSpec = cfg.DefaultModel
	}

	// Resolve model - convert aliases to full format
	// Config aliases take precedence over built-in aliases
	resolvedModel := model.ResolveWithConfig(effectiveSpec, configModels)

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

// ResolveSpawnSettings resolves spawn settings using the centralized resolver and
// emits any warnings or infrastructure escape hatch messages.
func ResolveSpawnSettings(input spawn.ResolveInput) (ResolvedSpawnResult, error) {
	settings, err := spawn.Resolve(input)
	if err != nil {
		return ResolvedSpawnResult{}, err
	}

	for _, warning := range settings.Warnings {
		fmt.Fprintf(os.Stderr, "⚠️  %s\n", warning)
		if strings.Contains(warning, "infrastructure work detected") {
			fmt.Fprintf(os.Stderr, "   Recommendation: Use --backend claude for infrastructure work to survive server restarts.\n")
		}
	}

	if input.InfrastructureDetected && settings.Backend.Source == spawn.SourceHeuristic && settings.Backend.Detail == "infra-escape-hatch" {
		fmt.Println("🔧 Infrastructure work detected - auto-applying escape hatch (--backend claude --tmux)")
		fmt.Println("   This ensures the agent survives OpenCode server restarts.")

		logger := events.NewLogger(events.DefaultLogPath())
		event := events.Event{
			Type:      "spawn.infrastructure_detected",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"task":     input.Task,
				"beads_id": input.BeadsID,
				"skill":    input.SkillName,
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log infrastructure detection: %v\n", err)
		}
	}

	resolvedModel := model.ResolveWithConfig(settings.Model.Value, nil)
	return ResolvedSpawnResult{Settings: settings, Model: resolvedModel}, nil
}

// GatherSpawnContext gathers KB context and performs gap analysis.
// Returns context string, gap analysis, model injection info, or error.
func GatherSpawnContext(skillContent, task, beadsID, projectDir, workspaceName, skillName string, skipArtifactCheck, gateOnGap, skipGapGate bool, gapThreshold int) (
	kbContext string,
	gapAnalysis *spawn.GapAnalysis,
	hasInjectedModels bool,
	primaryModelPath string,
	crossRepoModelDir string,
	err error) {
	stalenessMeta := &spawn.StalenessEventMeta{
		SpawnID:    workspaceName,
		AgentSkill: skillName,
	}

	if skipArtifactCheck {
		fmt.Println("Skipping context check (--skip-artifact-check)")
		return "", nil, false, "", "", nil
	}

	// Parse skill requirements to determine what context to gather
	requires := spawn.ParseSkillRequires(skillContent)

	if requires != nil && requires.HasRequirements() {
		// Skill-driven context gathering
		fmt.Printf("Gathering context (skill requires: %s)\n", requires.String())
		kbContext = spawn.GatherRequiredContext(requires, task, beadsID, projectDir, stalenessMeta)
		// For skill-driven context, create a basic gap analysis from the results
		// This is a placeholder - skills may provide their own gap info
		gapAnalysis = spawn.AnalyzeGaps(nil, task, projectDir)
	} else {
		// Fall back to default kb context check with full gap analysis
		gapResult := runPreSpawnKBCheckFull(task, projectDir, stalenessMeta)
		kbContext = gapResult.Context
		gapAnalysis = gapResult.GapAnalysis

		// Extract model injection info for probe vs investigation routing
		if gapResult.FormatResult != nil {
			hasInjectedModels = gapResult.FormatResult.HasInjectedModels
			if hasInjectedModels {
				// Extract primary model path from KB context result
				primaryModelPath = extractPrimaryModelPath(gapResult.FormatResult)
			}
			crossRepoModelDir = gapResult.FormatResult.CrossRepoModelDir
		}
	}

	// Check gap gating - may block spawn if context quality is too low
	if err := checkGapGating(gapAnalysis, gateOnGap, skipGapGate, gapThreshold); err != nil {
		return "", nil, false, "", "", err
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

	return kbContext, gapAnalysis, hasInjectedModels, primaryModelPath, crossRepoModelDir, nil
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
// Priority: explicit --backend flag > explicit model (CLI or user default_model) > project config > user config > infrastructure detection (advisory) > default.
// When --backend, explicit model, or config is explicit, infrastructure detection becomes advisory (warning only).
// This prevents the escape hatch from silently overriding user intent.
func DetermineSpawnBackend(resolvedModel model.ModelSpec, task, beadsID, projectDir, backendFlag, spawnModel string) (string, error) {
	// Load project config (used for backend default)
	projCfg, projMeta, _ := config.LoadWithMeta(projectDir)
	projectSpawnModeExplicit := projMeta != nil && projMeta.Explicit["spawn_mode"]

	// Load user config (~/.orch/config.yaml) for backend fallback
	userCfg, userMeta, _ := userconfig.LoadWithMeta()
	userCfgExplicit := userMeta != nil && userMeta.Explicit["backend"] && userCfg != nil && userCfg.Backend != ""
	userDefaultModelExplicit := userMeta != nil && userMeta.Explicit["default_model"] && userCfg != nil && userCfg.DefaultModel != ""

	// Default to opencode (primary spawn path)
	backend := "opencode"

	// Track whether flags were explicitly set by user
	// Explicit flags ALWAYS win over auto-detection (including infrastructure escape hatch)
	explicitBackend := backendFlag != ""
	explicitModel := spawnModel != "" || userDefaultModelExplicit

	if explicitBackend {
		// Explicit --backend flag: highest priority, always wins
		backend = backendFlag
		// Validate backend value
		if backend != "claude" && backend != "opencode" {
			return "", fmt.Errorf("invalid --backend value: %s (must be 'claude' or 'opencode')", backend)
		}

		// Advisory: warn if infrastructure work detected but user chose different backend
		if isInfrastructureWork(task, beadsID) && backend != "claude" {
			fmt.Fprintf(os.Stderr, "⚠️  Infrastructure work detected but respecting explicit --backend %s\n", backend)
			fmt.Fprintf(os.Stderr, "   Recommendation: Use --backend claude for infrastructure work to survive server restarts.\n")
		}
	} else if explicitModel {
		modelName := spawnModel
		if modelName == "" && userDefaultModelExplicit {
			modelName = userCfg.DefaultModel
		}
		if modelName == "" {
			modelName = resolvedModel.Format()
		}
		// Explicit --model flag: model choice implies backend requirements
		// Don't let infrastructure detection override — the user chose a specific model
		// that may require a specific backend (e.g., codex requires opencode)
		// Resolution: project config > user config > hardcoded default
		if projCfg != nil && projectSpawnModeExplicit && projCfg.SpawnMode != "" {
			backend = projCfg.SpawnMode
		} else if userCfgExplicit {
			backend = userCfg.Backend
		}
		// Advisory: warn if infrastructure work detected
		if isInfrastructureWork(task, beadsID) && backend != "claude" {
			fmt.Fprintf(os.Stderr, "⚠️  Infrastructure work detected but respecting explicit model %s (backend: %s)\n", modelName, backend)
			fmt.Fprintf(os.Stderr, "   Recommendation: Use --backend claude for infrastructure work to survive server restarts.\n")
		}
	} else if projCfg != nil && projectSpawnModeExplicit && projCfg.SpawnMode != "" {
		// Config default: respect project spawn_mode setting
		backend = projCfg.SpawnMode
		if isInfrastructureWork(task, beadsID) && backend != "claude" {
			fmt.Fprintf(os.Stderr, "⚠️  Infrastructure work detected but respecting project spawn_mode %s\n", backend)
			fmt.Fprintf(os.Stderr, "   Recommendation: Use --backend claude for infrastructure work to survive server restarts.\n")
		}
	} else if userCfgExplicit {
		// User config default: respect user-level backend setting (~/.orch/config.yaml)
		backend = userCfg.Backend
		if isInfrastructureWork(task, beadsID) && backend != "claude" {
			fmt.Fprintf(os.Stderr, "⚠️  Infrastructure work detected but respecting user config backend %s\n", backend)
			fmt.Fprintf(os.Stderr, "   Recommendation: Use --backend claude for infrastructure work to survive server restarts.\n")
		}
	} else if isInfrastructureWork(task, beadsID) {
		// No explicit flags or config: auto-apply escape hatch
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
	// Infer verify level if not explicitly set
	verifyLevel := ctx.VerifyLevel
	if verifyLevel == "" {
		issueType := ""
		if ctx.IsBug {
			issueType = "bug"
		}
		verifyLevel = spawn.DefaultVerifyLevel(ctx.SkillName, issueType)
	}

	return &spawn.Config{
		Task:               ctx.Task,
		OrientationFrame:   ctx.OrientationFrame,
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
		ResolvedSettings:   ctx.ResolvedSettings,
		MCP:                mcp,
		Tier:               ctx.Tier,
		VerifyLevel:        verifyLevel,
		Scope:              ctx.Scope,
		NoTrack:            noTrack || ctx.IsOrchestrator || ctx.IsMetaOrchestrator,
		SkipArtifactCheck:  skipArtifactCheck,
		KBContext:          ctx.KBContext,
		HasInjectedModels:  ctx.HasInjectedModels,
		PrimaryModelPath:   ctx.PrimaryModelPath,
		CrossRepoModelDir:  ctx.CrossRepoModelDir,
		IncludeServers:     spawn.DefaultIncludeServersForSkill(ctx.SkillName),
		GapAnalysis:        ctx.GapAnalysis,
		IsBug:              ctx.IsBug,
		ReproSteps:         ctx.ReproSteps,
		ReworkFeedback:     ctx.ReworkFeedback,
		ReworkNumber:       ctx.ReworkNumber,
		PriorSynthesis:     ctx.PriorSynthesis,
		PriorWorkspace:     ctx.PriorWorkspace,
		IsOrchestrator:     ctx.IsOrchestrator,
		IsMetaOrchestrator: ctx.IsMetaOrchestrator,
		UsageInfo:          ctx.UsageInfo,
		Account:            ctx.Account,
		AccountConfigDir:   ctx.AccountConfigDir,
		SpawnMode:          ctx.SpawnBackend,
		HotspotArea:        ctx.HotspotArea,
		HotspotFiles:       ctx.HotspotFiles,
		DesignWorkspace:    "", // Will be set by caller if needed
		DesignMockupPath:   ctx.DesignMockupPath,
		DesignPromptPath:   ctx.DesignPromptPath,
		DesignNotes:        ctx.DesignNotes,
		BeadsDir:           ctx.BeadsDir,
	}
}

// ValidateAndWriteContext validates context size, writes workspace via atomic spawn Phase 1, and generates prompt.
// Returns minimal prompt, rollback function (for undoing Phase 1 on spawn failure), or error if validation fails.
// The rollback function should be called if session creation fails to undo beads tagging and workspace writes.
func ValidateAndWriteContext(cfg *spawn.Config, force bool) (minimalPrompt string, rollback func(), err error) {
	// Pre-spawn token estimation and validation
	if err := spawn.ValidateContextSize(cfg); err != nil {
		return "", nil, fmt.Errorf("pre-spawn validation failed: %w", err)
	}

	// Warn about large contexts (but don't block)
	if shouldWarn, warning := spawn.ShouldWarnAboutSize(cfg); shouldWarn {
		fmt.Fprintf(os.Stderr, "%s", warning)
	}

	// Warn if task text references a different beads ID than the tracking issue
	if warning := spawn.ValidateBeadsIDConsistency(cfg.Task, cfg.BeadsID); warning != "" {
		fmt.Fprintf(os.Stderr, "%s\n", warning)
	}

	// Check for existing workspace before writing context
	// This prevents accidentally overwriting SESSION_HANDOFF.md from completed sessions
	// Note: With unique suffixes in workspace names (since Jan 2026), collisions are rare
	// but this provides an extra safety net and meaningful error messages
	if err := checkWorkspaceExists(cfg.WorkspacePath(), force); err != nil {
		return "", nil, err
	}

	// Atomic spawn Phase 1: tag beads with orch:agent + write workspace (SPAWN_CONTEXT.md, manifest, dotfiles)
	// Returns rollback function that undoes all Phase 1 writes on spawn failure.
	atomicOpts := &spawn.AtomicSpawnOpts{
		Config:  cfg,
		BeadsID: cfg.BeadsID,
		NoTrack: cfg.NoTrack,
	}
	rollback, atomicErr := spawn.AtomicSpawnPhase1(atomicOpts)
	if atomicErr != nil {
		return "", nil, fmt.Errorf("failed to write spawn context: %w", atomicErr)
	}

	// Record orientation frame in beads comments at spawn time.
	// Use OrientationFrame (issue description) if available for richer context;
	// fall back to Task (issue title) for manual spawns without separate framing.
	// Skip writing if a FRAME comment already exists (e.g., added by orchestrator before spawn).
	if !cfg.NoTrack && !cfg.IsOrchestrator && !cfg.IsMetaOrchestrator && cfg.BeadsID != "" {
		existingFrame := spawn.ExtractFrameFromBeadsComments(cfg.BeadsID)
		if existingFrame == "" {
			// No existing FRAME — write one from OrientationFrame or Task
			frame := strings.TrimSpace(cfg.OrientationFrame)
			if frame == "" {
				frame = strings.TrimSpace(cfg.Task)
			}
			if frame != "" {
				if err := addBeadsComment(cfg.BeadsID, fmt.Sprintf("FRAME: %s", frame)); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to add frame comment: %v\n", err)
				}
			}
		}
	}

	// Record spawn in session (if session is active)
	if sessionStore, err := session.New(""); err == nil {
		if err := sessionStore.RecordSpawn(cfg.BeadsID, cfg.SkillName, cfg.Task, cfg.ProjectDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to record spawn in session: %v\n", err)
		}
	}

	// Generate minimal prompt
	minimalPrompt = spawn.MinimalPrompt(cfg)
	return minimalPrompt, rollback, nil
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
func runPreSpawnKBCheck(task, projectDir string, stalenessMeta *spawn.StalenessEventMeta) string {
	result := runPreSpawnKBCheckFull(task, projectDir, stalenessMeta)
	return result.Context
}

// runPreSpawnKBCheckFull runs kb context check with full gap analysis results.
// This allows callers to access gap analysis for gating decisions.
func runPreSpawnKBCheckFull(task, projectDir string, stalenessMeta *spawn.StalenessEventMeta) *GapCheckResult {
	gcr := &GapCheckResult{}

	// Extract keywords from task description
	// Try with 3 keywords first (more specific), fall back to 1 keyword (more broad)
	keywords := spawn.ExtractKeywords(task, 3)
	if keywords == "" {
		// Perform gap analysis even when no keywords extracted
		gcr.GapAnalysis = spawn.AnalyzeGaps(nil, task, projectDir)
		if gcr.GapAnalysis.ShouldWarnAboutGaps() {
			// Use prominent warning format for better visibility
			fmt.Fprintf(os.Stderr, "%s", gcr.GapAnalysis.FormatProminentWarning())
		}
		return gcr
	}

	fmt.Printf("Checking kb context for: %q\n", keywords)

	// Run kb context check (use projectDir for cross-project group resolution)
	result, err := spawn.RunKBContextCheckForDir(keywords, projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: kb context check failed: %v\n", err)
		return gcr
	}

	// If no matches with multiple keywords, try with just the first keyword
	if result == nil || !result.HasMatches {
		firstKeyword := spawn.ExtractKeywords(task, 1)
		if firstKeyword != "" && firstKeyword != keywords {
			fmt.Printf("Trying broader search for: %q\n", firstKeyword)
			result, err = spawn.RunKBContextCheckForDir(firstKeyword, projectDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: kb context check failed: %v\n", err)
				return gcr
			}
		}
	}

	// Perform gap analysis to detect context gaps
	gcr.GapAnalysis = spawn.AnalyzeGaps(result, keywords, projectDir)
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

	// Scope-appropriate filtering: when task targets specific files, strip heavyweight
	// categories (models, guides, investigations, open questions) and use reduced budget.
	// Saves 1,000-4,000 tokens per prompt for scoped tasks.
	maxChars := spawn.MaxKBContextChars
	if spawn.TaskIsScoped(task) {
		originalCount := len(result.Matches)
		result.Matches = spawn.FilterForScopedTask(result.Matches)
		result.HasMatches = len(result.Matches) > 0
		maxChars = spawn.ScopedMaxKBContextChars
		fmt.Printf("Scoped task detected: filtered %d → %d matches (budget: %dk chars)\n",
			originalCount, len(result.Matches), maxChars/1000)
		if !result.HasMatches {
			fmt.Println("No relevant context after scoped filtering.")
			return gcr
		}
	}

	// Format context with limit and capture full result (includes HasInjectedModels)
	formatResult := spawn.FormatContextForSpawnWithLimitAndMeta(result, maxChars, projectDir, stalenessMeta)
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

// IsInfrastructureWork exposes infrastructure work detection for callers.
func IsInfrastructureWork(task string, beadsID string) bool {
	return isInfrastructureWork(task, beadsID)
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
