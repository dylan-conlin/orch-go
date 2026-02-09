// Package main provides validation and gap checking for spawn commands.
package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"gopkg.in/yaml.v3"
)

const triageBypassEnvVar = "ORCH_BYPASS_TRIAGE"
const hotspotSuppressEnvVar = "ORCH_SUPPRESS_HOTSPOT"

// GapCheckResult contains the results of a pre-spawn gap check.
type GapCheckResult struct {
	Context     string             // Formatted context to include in SPAWN_CONTEXT.md
	GapAnalysis *spawn.GapAnalysis // Gap analysis results for further processing
	Blocked     bool               // True if spawn should be blocked due to gaps
	BlockReason string             // Reason for blocking (if Blocked is true)
}

// runPreSpawnKBCheckFull runs kb context check with full gap analysis results.
// This allows callers to access gap analysis for gating decisions.
// If projectDir is provided, domain is auto-detected for ecosystem filtering.
// Domain can be explicitly overridden via domainOverride parameter.
func runPreSpawnKBCheckFull(task string, projectDir string, domainOverride ...string) *GapCheckResult {
	gcr := &GapCheckResult{}

	// Determine domain: explicit override > config file > auto-detection
	var domain string
	if len(domainOverride) > 0 && domainOverride[0] != "" {
		domain = domainOverride[0]
		fmt.Printf("Using domain override: %s\n", domain)
	} else if projectDir != "" {
		domain = spawn.DetectDomain(projectDir)
		fmt.Printf("Detected domain: %s (from %s)\n", domain, projectDir)
	} else {
		domain = spawn.DomainPersonal
	}

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

	// Run kb context check with domain-aware filtering
	// Pass projectDir to ensure kb searches the target project's .kb directory
	result, err := spawn.RunKBContextCheckWithDomain(keywords, domain, projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: kb context check failed: %v\n", err)
		return gcr
	}

	// If no matches with multiple keywords, try with just the first keyword
	if result == nil || !result.HasMatches {
		firstKeyword := spawn.ExtractKeywords(task, 1)
		if firstKeyword != "" && firstKeyword != keywords {
			fmt.Printf("Trying broader search for: %q\n", firstKeyword)
			result, err = spawn.RunKBContextCheckWithDomain(firstKeyword, domain, projectDir)
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

	// Include gap summary in spawn context if there are significant gaps
	contextContent := spawn.FormatContextForSpawn(result)
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
		// Display loud visual warning before the detailed message
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "🚨🚨🚨 SPAWN BLOCKED BY GAP GATE 🚨🚨🚨\n")
		fmt.Fprintf(os.Stderr, "\n")

		// Display the block message
		fmt.Fprintf(os.Stderr, "%s", gapAnalysis.FormatGateBlockMessage())

		// Add visual separator after the message for prominence
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "⚠️  This spawn has been BLOCKED. The orchestrator should add context or use --skip-gap-gate.\n")
		fmt.Fprintf(os.Stderr, "\n")

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

// shouldAutoBypassTriage determines if a skill should automatically bypass triage ceremony.
// Orchestrator and meta-orchestrator skills inherently perform triage as part of their
// coordination work, so requiring --bypass-triage is redundant ceremony.
// Returns (bypass, reason) where reason identifies the auto-bypass source.
func shouldAutoBypassTriage(skillName string) (bool, string) {
	switch skillName {
	case "orchestrator":
		return true, "orchestrator"
	case "meta-orchestrator":
		return true, "meta-orchestrator"
	default:
		return false, ""
	}
}

// resolveTriageBypass returns whether triage bypass is enabled and how it was enabled.
// Sources: explicit --bypass-triage flag, session-level ORCH_BYPASS_TRIAGE env var,
// or auto-bypass for orchestrator skills.
func resolveTriageBypass() (bool, string) {
	if spawnBypassTriage {
		return true, "flag"
	}

	v := strings.TrimSpace(strings.ToLower(os.Getenv(triageBypassEnvVar)))
	if v == "1" || v == "true" || v == "yes" || v == "on" {
		return true, "env"
	}

	return false, ""
}

// resolveHotspotSuppression returns whether hotspot warning output should be suppressed.
// Sources: explicit --acknowledge-hotspot flag or session-level ORCH_SUPPRESS_HOTSPOT env var.
func resolveHotspotSuppression() (bool, string) {
	if spawnAcknowledgeHotspot {
		return true, "flag"
	}

	v := strings.TrimSpace(strings.ToLower(os.Getenv(hotspotSuppressEnvVar)))
	if v == "1" || v == "true" || v == "yes" || v == "on" {
		return true, "env"
	}

	return false, ""
}

// showTriageBypassRequired displays a warning and returns an error when triage bypass is not provided.
// This creates friction to encourage the daemon-driven workflow over manual spawning.
func showTriageBypassRequired(skillName, task string) error {
	fmt.Fprintf(os.Stderr, `
┌─────────────────────────────────────────────────────────────────────────────┐
│  ⚠️  TRIAGE BYPASS REQUIRED                                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│  Tracked manual spawn requires --bypass-triage flag.                        │
│                                                                             │
│  The preferred workflow is daemon-driven triage:                            │
│    1. Create issue: bd create "task" --type task -l triage:ready            │
│    2. Daemon auto-spawns: orch daemon run                                   │
│                                                                             │
│  Manual spawn is for exceptions only:                                       │
│    - Single urgent item requiring immediate attention                       │
│    - Complex/ambiguous task needing custom context                          │
│    - Skill selection requires orchestrator judgment                         │
│                                                                             │
│  For ad-hoc work without issue tracking, use --no-track.                    │
│                                                                             │
│  To proceed with manual spawn, use one of:                                  │
│    1. One-off: orch spawn --bypass-triage ...                               │
│    2. Session: export ORCH_BYPASS_TRIAGE=1                                  │
│                                                                             │
│  Example one-off:                                                           │
│    orch spawn --bypass-triage %s "%s"                          │
└─────────────────────────────────────────────────────────────────────────────┘

`, skillName, truncate(task, 30))
	return fmt.Errorf("spawn blocked: triage bypass required for manual spawns (--bypass-triage or %s=1)", triageBypassEnvVar)
}

// logTriageBypass logs a triage bypass event to events.jsonl for Phase 2 review.
// This tracks how often manual spawns occur vs daemon-driven spawns.
func logTriageBypass(skillName, task, source string) {
	if source == "" {
		source = "unknown"
	}

	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "spawn.triage_bypassed",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"skill":  skillName,
			"task":   task,
			"source": source,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log triage bypass: %v\n", err)
	}
}

// isCriticalInfrastructureWork detects if a task involves CRITICAL infrastructure
// work that could restart the OpenCode server and kill connected agents.
//
// This is intentionally NARROW - only files that directly affect server lifecycle:
// - serve.go (OpenCode server startup/shutdown)
// - pkg/opencode/* (OpenCode client that connects to server)
// - spawn_cmd.go (spawn logic that uses OpenCode API)
//
// Explicitly EXCLUDED (non-critical):
// - Dashboard UI, agent cards, frontend components
// - Skill system, skillc compiler
// - General orchestration work
// - Status commands, monitoring
//
// Returns true if CRITICAL infrastructure work is detected, false otherwise.
func isCriticalInfrastructureWork(task string, beadsID string) bool {
	// CRITICAL keywords - only files that could restart the OpenCode server
	// These are patterns that indicate work on the server lifecycle itself
	criticalKeywords := []string{
		"serve.go",         // OpenCode server startup
		"pkg/opencode",     // OpenCode client code
		"opencode server",  // Explicit server work
		"opencode api",     // API client that connects to server
		"restart opencode", // Explicit restart
		"server restart",   // Explicit restart
		"server startup",   // Startup changes
		"server shutdown",  // Shutdown changes
	}

	// Check task description (case-insensitive)
	taskLower := strings.ToLower(task)
	for _, keyword := range criticalKeywords {
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
			for _, keyword := range criticalKeywords {
				if strings.Contains(titleLower, keyword) {
					return true
				}
			}
			// Check description
			descLower := strings.ToLower(issue.Description)
			for _, keyword := range criticalKeywords {
				if strings.Contains(descLower, keyword) {
					return true
				}
			}
		}
	}

	return false
}

// requiresResourceLifecycleAudit detects infrastructure-touching work that should
// include the resource lifecycle audit directive in SPAWN_CONTEXT.
//
// This is intentionally BROADER than isCriticalInfrastructureWork:
// - It includes infrastructure packages that commonly create long-lived resources.
// - It includes explicit resource-creation signals (exec.Command, goroutines).
// - It is prompt guidance only; it does not drive backend selection.
func requiresResourceLifecycleAudit(task string, beadsID string) bool {
	auditTriggers := []string{
		"pkg/daemon",
		"pkg/spawn",
		"cmd/orch/serve",
		"exec.command",
		"exec command",
		"goroutine",
		"go func(",
	}

	containsAuditTrigger := func(text string) bool {
		textLower := strings.ToLower(text)
		for _, trigger := range auditTriggers {
			if strings.Contains(textLower, trigger) {
				return true
			}
		}
		return false
	}

	if containsAuditTrigger(task) {
		return true
	}

	if beadsID == "" {
		return false
	}

	issue, err := verify.GetIssue(beadsID)
	if err != nil {
		return false
	}

	return containsAuditTrigger(issue.Title) || containsAuditTrigger(issue.Description)
}

// checkWorkspaceExists verifies if a workspace already exists and has content.
// Returns an error if the workspace contains SPAWN_CONTEXT.md or SYNTHESIS.md
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
		"SYNTHESIS.md",
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

// fetchIssueCommentsForSpawn retrieves comments from a beads issue to include in spawn context.
// Returns orchestrator notes that were added after issue creation.
// Filters out Phase: comments (progress tracking) to only include substantive guidance.
func fetchIssueCommentsForSpawn(beadsID string) []spawn.IssueComment {
	// Use beads CLIClient to get comments
	client := beads.NewCLIClient()
	beadsComments, err := client.Comments(beadsID)
	if err != nil {
		// Silently fail - comments are supplementary context
		return nil
	}

	// Filter and convert comments
	var comments []spawn.IssueComment
	for _, c := range beadsComments {
		// Skip Phase: comments (progress tracking, not guidance)
		if strings.HasPrefix(c.Text, "Phase:") {
			continue
		}
		// Skip empty comments
		if strings.TrimSpace(c.Text) == "" {
			continue
		}
		comments = append(comments, spawn.IssueComment{
			Author:    c.Author,
			Text:      c.Text,
			CreatedAt: c.CreatedAt,
		})
	}

	return comments
}

// DecisionBlock represents a block rule in decision frontmatter.
type DecisionBlock struct {
	Keywords []string `yaml:"keywords"`
	Patterns []string `yaml:"patterns"`
}

// DecisionFrontmatter represents the YAML frontmatter in a decision file.
type DecisionFrontmatter struct {
	Blocks []DecisionBlock `yaml:"blocks"`
}

// DecisionConflict represents a decision that blocks a spawn.
type DecisionConflict struct {
	DecisionID   string   // Decision filename without extension
	DecisionPath string   // Full path to decision file
	Title        string   // Decision title
	Summary      string   // First paragraph of decision
	MatchedOn    []string // Keywords or patterns that matched
}

// DecisionCheckResult contains the result of a decision conflict check.
type DecisionCheckResult struct {
	ConflictFound bool
	Acknowledged  bool
	DecisionID    string
	MatchedOn     string
}

// checkDecisionConflicts checks if any decisions block this spawn.
// Returns an error if a decision conflict is found and not acknowledged.
// Also returns metadata about the check for logging purposes.
func checkDecisionConflicts(task, projectDir, acknowledgedDecision string) (*DecisionCheckResult, error) {
	result := &DecisionCheckResult{}

	conflicts, err := findBlockingDecisions(task, projectDir)
	if err != nil {
		// FAIL-CLOSED: Block spawn when decision checking fails.
		// Security/safety-critical gates should fail closed, not open.
		// If we can't verify no conflicts exist, we must assume they might.
		fmt.Fprintf(os.Stderr, "Error: decision check failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "Spawn blocked due to decision check failure (fail-closed).\n")
		return result, fmt.Errorf("spawn blocked: decision check failed: %w", err)
	}

	if len(conflicts) == 0 {
		return result, nil
	}

	result.ConflictFound = true

	// Check if conflict was acknowledged
	for _, conflict := range conflicts {
		if conflict.DecisionID == acknowledgedDecision {
			// Conflict acknowledged, allow spawn but log it
			result.Acknowledged = true
			result.DecisionID = conflict.DecisionID
			result.MatchedOn = strings.Join(conflict.MatchedOn, ", ")
			fmt.Fprintf(os.Stderr, "⚠️  Decision conflict acknowledged: %s\n", conflict.DecisionID)
			return result, nil
		}
	}

	// Display conflict warning
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "⚠️ ⚠️ ⚠️  DECISION CONFLICT  ⚠️ ⚠️ ⚠️\n")
	fmt.Fprintf(os.Stderr, "\n")

	for _, conflict := range conflicts {
		fmt.Fprintf(os.Stderr, "Decision: %s\n", conflict.Title)
		fmt.Fprintf(os.Stderr, "File: %s\n", conflict.DecisionID)
		fmt.Fprintf(os.Stderr, "\n")
		if conflict.Summary != "" {
			fmt.Fprintf(os.Stderr, "%s\n", conflict.Summary)
			fmt.Fprintf(os.Stderr, "\n")
		}
		if len(conflict.MatchedOn) > 0 {
			fmt.Fprintf(os.Stderr, "Matched on: %s\n", strings.Join(conflict.MatchedOn, ", "))
			fmt.Fprintf(os.Stderr, "\n")
		}
	}

	fmt.Fprintf(os.Stderr, "To proceed, acknowledge this decision:\n")
	fmt.Fprintf(os.Stderr, "  orch spawn --acknowledge-decision %s [other flags] <skill> \"<task>\"\n", conflicts[0].DecisionID)
	fmt.Fprintf(os.Stderr, "\n")

	result.DecisionID = conflicts[0].DecisionID
	result.MatchedOn = strings.Join(conflicts[0].MatchedOn, ", ")
	return result, fmt.Errorf("spawn blocked: decision conflict (use --acknowledge-decision to override)")
}

// findBlockingDecisions finds decisions that block the given task.
func findBlockingDecisions(task, projectDir string) ([]DecisionConflict, error) {
	// Find .kb directory
	kbDir := filepath.Join(projectDir, ".kb", "decisions")
	if _, err := os.Stat(kbDir); os.IsNotExist(err) {
		return nil, nil // No .kb/decisions directory, no conflicts
	}

	// Extract keywords from task
	taskKeywords := spawn.ExtractKeywords(task, 10)
	taskKeywordList := strings.Fields(strings.ToLower(taskKeywords))
	taskLower := strings.ToLower(task)

	// Read all decision files
	files, err := ioutil.ReadDir(kbDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read decisions directory: %w", err)
	}

	var conflicts []DecisionConflict

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".md") {
			continue
		}

		decisionPath := filepath.Join(kbDir, file.Name())
		content, err := ioutil.ReadFile(decisionPath)
		if err != nil {
			continue // Skip files we can't read
		}

		// Parse frontmatter
		frontmatter, err := parseDecisionFrontmatter(string(content))
		if err != nil || frontmatter == nil || len(frontmatter.Blocks) == 0 {
			continue // No blocks defined, skip
		}

		// Check if any blocks match
		var matchedOn []string
		for _, block := range frontmatter.Blocks {
			// Check keywords
			for _, keyword := range block.Keywords {
				keywordLower := strings.ToLower(keyword)
				// Check if keyword appears in task
				if strings.Contains(taskLower, keywordLower) {
					matchedOn = append(matchedOn, keyword)
				}
				// Check if keyword matches any extracted task keywords
				for _, taskKw := range taskKeywordList {
					if strings.Contains(taskKw, keywordLower) || strings.Contains(keywordLower, taskKw) {
						matchedOn = append(matchedOn, keyword)
					}
				}
			}

			// Check patterns (file/path patterns like "**/api/**")
			// Extract meaningful path segments from glob patterns and check
			// if any appear in the task description.
			for _, pattern := range block.Patterns {
				// Extract non-wildcard path segments from the pattern
				segments := strings.Split(pattern, "/")
				for _, seg := range segments {
					seg = strings.TrimSpace(seg)
					if seg == "" || seg == "*" || seg == "**" || strings.Contains(seg, "*") {
						continue
					}
					if strings.Contains(taskLower, strings.ToLower(seg)) {
						matchedOn = append(matchedOn, "pattern: "+pattern)
						break
					}
				}
			}
		}

		if len(matchedOn) > 0 {
			// Extract decision title and summary
			title, summary := extractDecisionInfo(string(content))
			decisionID := strings.TrimSuffix(file.Name(), ".md")

			conflicts = append(conflicts, DecisionConflict{
				DecisionID:   decisionID,
				DecisionPath: decisionPath,
				Title:        title,
				Summary:      summary,
				MatchedOn:    matchedOn,
			})
		}
	}

	return conflicts, nil
}

// parseDecisionFrontmatter parses YAML frontmatter from a decision file.
// Returns nil if no frontmatter found or parsing fails.
func parseDecisionFrontmatter(content string) (*DecisionFrontmatter, error) {
	// Check if content starts with YAML frontmatter (---)
	if !strings.HasPrefix(content, "---\n") {
		return nil, nil
	}

	// Find the closing ---
	// Handle both "---\n<yaml>\n---\n" and empty frontmatter "---\n---\n"
	rest := content[4:]
	endIdx := strings.Index(rest, "\n---\n")
	if endIdx == -1 {
		// Check for empty frontmatter: "---\n---\n..."
		if strings.HasPrefix(rest, "---\n") || rest == "---" {
			return &DecisionFrontmatter{}, nil
		}
		return nil, nil
	}

	// Extract YAML content
	yamlContent := rest[:endIdx]

	// Parse YAML
	var frontmatter DecisionFrontmatter
	if err := yaml.Unmarshal([]byte(yamlContent), &frontmatter); err != nil {
		return nil, err
	}

	return &frontmatter, nil
}

// extractDecisionInfo extracts the title and first paragraph from a decision file.
func extractDecisionInfo(content string) (title, summary string) {
	lines := strings.Split(content, "\n")

	// Skip frontmatter if present
	startIdx := 0
	if strings.HasPrefix(content, "---\n") {
		endIdx := strings.Index(content[4:], "\n---\n")
		if endIdx != -1 {
			startIdx = len(strings.Split(content[:4+endIdx+5], "\n"))
		}
	}

	// Find title (first # heading)
	titleRe := regexp.MustCompile(`^#\s+(.+)$`)
	for i := startIdx; i < len(lines); i++ {
		if match := titleRe.FindStringSubmatch(lines[i]); match != nil {
			title = match[1]
			break
		}
	}

	// Extract first paragraph after title (non-empty, non-heading lines)
	var summaryLines []string
	inSummary := false
	for i := startIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Skip until we find the title
		if !inSummary && strings.HasPrefix(line, "# ") {
			inSummary = true
			continue
		}

		if !inSummary {
			continue
		}

		// Stop at next heading or end of first paragraph
		if strings.HasPrefix(line, "#") {
			break
		}

		// Skip empty lines before we have content
		if line == "" && len(summaryLines) == 0 {
			continue
		}

		// Stop at first empty line after we have content (end of paragraph)
		if line == "" && len(summaryLines) > 0 {
			break
		}

		summaryLines = append(summaryLines, line)

		// Limit to ~3 lines for summary
		if len(summaryLines) >= 3 {
			break
		}
	}

	summary = strings.Join(summaryLines, " ")
	return title, summary
}

// ActiveAgentInfo contains information about an active agent for a beads issue.
type ActiveAgentInfo struct {
	ID        string // Agent/workspace name
	SessionID string // OpenCode session ID if available
	Status    string // Agent status (active, idle, dead, etc.)
	Phase     string // Current phase (Planning, Implementing, etc.)
	SpawnedAt string // ISO 8601 timestamp of when agent was spawned
}

// checkActiveAgentForBeadsID checks if there's already an active agent for the given beads ID.
// This prevents duplicate spawns when manual spawn and daemon spawn race.
//
// Returns an ActiveAgentInfo if an active agent exists, nil otherwise.
// An error is returned if the check itself fails (e.g., server not reachable).
func checkActiveAgentForBeadsID(beadsID string) (*ActiveAgentInfo, error) {
	if beadsID == "" {
		return nil, nil
	}

	// Query the orch serve /api/agents endpoint
	// Use the default orch serve port (3348)
	orchServeURL := fmt.Sprintf("https://127.0.0.1:%d/api/agents?since=24h", DefaultServePort)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(orchServeURL)
	if err != nil {
		// Server not running - this is OK, just means no agents to check
		// Return nil to allow spawn to proceed
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Server returned an error - this is unusual but not blocking
		return nil, nil
	}

	// Parse the response
	var agents []struct {
		ID        string `json:"id"`
		SessionID string `json:"session_id"`
		BeadsID   string `json:"beads_id"`
		Status    string `json:"status"`
		Phase     string `json:"phase"`
		SpawnedAt string `json:"spawned_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
		// Failed to parse - not blocking
		return nil, nil
	}

	// Look for an active agent with this beads ID
	// "Active" means status is one of: active, idle, dead (needs attention)
	// "Completed" or "awaiting-cleanup" means the agent is done
	for _, agent := range agents {
		if agent.BeadsID == beadsID {
			// Skip completed agents - they're done and don't block new spawns
			if agent.Status == "completed" || agent.Status == "awaiting-cleanup" {
				continue
			}

			// Found an active/idle/dead agent - this blocks spawn
			return &ActiveAgentInfo{
				ID:        agent.ID,
				SessionID: agent.SessionID,
				Status:    agent.Status,
				Phase:     agent.Phase,
				SpawnedAt: agent.SpawnedAt,
			}, nil
		}
	}

	return nil, nil
}

// formatActiveAgentError formats an error message when an active agent already exists.
func formatActiveAgentError(beadsID string, agent *ActiveAgentInfo) error {
	var statusMsg string
	switch agent.Status {
	case "active":
		statusMsg = "actively running"
	case "idle":
		statusMsg = "idle (may still be processing)"
	case "dead":
		statusMsg = "dead (needs attention - may have crashed)"
	default:
		statusMsg = agent.Status
	}

	phaseInfo := ""
	if agent.Phase != "" {
		phaseInfo = fmt.Sprintf(" (Phase: %s)", agent.Phase)
	}

	return fmt.Errorf("agent already exists for issue %s\n\n  Agent:   %s\n  Status:  %s%s\n  Session: %s\n\nTo force spawn anyway (not recommended - may cause duplicate work):\n  orch spawn --force [other flags] <skill> <task>\n\nTo interact with the existing agent:\n  orch send %s \"your message\"\n\nTo abandon the existing agent and restart:\n  orch abandon %s\n", beadsID, agent.ID, statusMsg, phaseInfo, agent.SessionID, agent.SessionID, beadsID)
}

// PostCompletionFailurePrefix is the marker for post-completion failure comments.
// Format: POST-COMPLETION-FAILURE: [type] - [description]
// Where type is one of: verification, implementation, spec, integration
const PostCompletionFailurePrefix = "POST-COMPLETION-FAILURE:"

// extractPostCompletionFailure looks for POST-COMPLETION-FAILURE comments in the issue comments
// and extracts the failure context for rework spawns.
// Returns nil if no failure comment is found.
func extractPostCompletionFailure(beadsID string) *spawn.FailureContext {
	client := beads.NewCLIClient()
	comments, err := client.Comments(beadsID)
	if err != nil {
		return nil
	}

	// Look for POST-COMPLETION-FAILURE comments (most recent first)
	// Multiple failures might exist if issue was reopened multiple times
	for i := len(comments) - 1; i >= 0; i-- {
		c := comments[i]
		if strings.HasPrefix(c.Text, PostCompletionFailurePrefix) {
			return parseFailureComment(c.Text)
		}
	}

	return nil
}

// parseFailureComment parses a POST-COMPLETION-FAILURE comment into FailureContext.
// Expected format: POST-COMPLETION-FAILURE: [type] - [description]
// If no type is specified, defaults to "implementation" and uses the full text as description.
func parseFailureComment(text string) *spawn.FailureContext {
	// Remove prefix
	content := strings.TrimPrefix(text, PostCompletionFailurePrefix)
	content = strings.TrimSpace(content)

	if content == "" {
		return &spawn.FailureContext{
			IsRework:       true,
			FailureType:    spawn.FailureTypeImplementation,
			Description:    "No failure details provided",
			SuggestedSkill: spawn.SuggestSkillForFailure(spawn.FailureTypeImplementation),
		}
	}

	// Try to parse "type - description" format
	// Types: verification, implementation, spec, integration
	var failureType, description string

	// Check for explicit type prefix
	lowerContent := strings.ToLower(content)
	for _, ft := range []string{spawn.FailureTypeVerification, spawn.FailureTypeImplementation, spawn.FailureTypeSpec, spawn.FailureTypeIntegration} {
		if strings.HasPrefix(lowerContent, ft) {
			failureType = ft
			// Remove type and optional separator
			remainder := content[len(ft):]
			remainder = strings.TrimPrefix(remainder, " - ")
			remainder = strings.TrimPrefix(remainder, ": ")
			remainder = strings.TrimPrefix(remainder, " ")
			description = strings.TrimSpace(remainder)
			break
		}
	}

	// If no explicit type found, default to implementation and use full content as description
	if failureType == "" {
		failureType = spawn.FailureTypeImplementation
		description = content
	}

	return &spawn.FailureContext{
		IsRework:       true,
		FailureType:    failureType,
		Description:    description,
		SuggestedSkill: spawn.SuggestSkillForFailure(failureType),
	}
}

// fetchFailureContextForSpawn retrieves POST-COMPLETION-FAILURE context for a beads issue.
// This is called during spawn to detect rework scenarios and inject failure context.
// Returns nil if this is not a rework spawn (no POST-COMPLETION-FAILURE comment).
func fetchFailureContextForSpawn(beadsID string) *spawn.FailureContext {
	if beadsID == "" {
		return nil
	}
	return extractPostCompletionFailure(beadsID)
}
