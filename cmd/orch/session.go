// Package main provides the CLI entry point for orch-go.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/focus"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/spf13/cobra"
)

// ============================================================================
// Session Command - Manage orchestrator work sessions
// ============================================================================

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage orchestrator work sessions",
	Long: `Manage orchestrator work sessions.

A session represents a focused work period with:
- A goal (north star priority)
- Start time
- Tracked spawns during the session

Session status derives agent state at query time via actual liveness checks,
not stored state. This prevents stale tracking.

Examples:
  orch session start "Ship snap MVP"    # Start a new session
  orch session status                   # Show current session status
  orch session end                      # End the current session`,
}

var (
	sessionJSON        bool
	resumeForInjection bool
	resumeCheck        bool
	validateJSON       bool
)

func init() {
	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionStatusCmd)
	sessionCmd.AddCommand(sessionEndCmd)
	sessionCmd.AddCommand(sessionResumeCmd)
	sessionCmd.AddCommand(sessionMigrateCmd)
	sessionCmd.AddCommand(sessionValidateCmd)

	// Add --json flag to status command
	sessionStatusCmd.Flags().BoolVar(&sessionJSON, "json", false, "Output as JSON")

	// Add flags for resume command
	sessionResumeCmd.Flags().BoolVar(&resumeForInjection, "for-injection", false, "Output condensed format for hook injection")
	sessionResumeCmd.Flags().BoolVar(&resumeCheck, "check", false, "Check if handoff exists (exit code only)")

	// Add --json flag for validate command
	sessionValidateCmd.Flags().BoolVar(&validateJSON, "json", false, "Output as JSON")

	rootCmd.AddCommand(sessionCmd)
}

// ============================================================================
// Session Start Command
// ============================================================================

var sessionStartCmd = &cobra.Command{
	Use:   "start [goal]",
	Short: "Start a new orchestrator work session",
	Long: `Start a new orchestrator work session with a focus goal.

The session tracks:
- Your focus goal
- When the session started
- Agents spawned during the session

If a session is already active, it will be replaced.

Examples:
  orch session start "Ship snap MVP"
  orch session start "Fix auth bugs"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		goal := strings.Join(args, " ")
		return runSessionStart(goal)
	},
}

func runSessionStart(goal string) error {
	store, err := session.New("")
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	// Check if session was already active
	wasActive := store.IsActive()

	// Get current working directory (project directory)
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get project directory: %w", err)
	}

	// Generate session name in format {project}-{count}
	sessionName, err := session.GenerateSessionName(projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to generate session name: %v\n", err)
		// Fall back to timestamp-based name
		sessionName = fmt.Sprintf("session-%s", time.Now().Format("20060102-1504"))
	}

	// Rename tmux window to match session name (auto-naming pattern)
	if err := tmux.RenameCurrentWindow(sessionName); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to rename tmux window: %v\n", err)
		// Continue anyway - window renaming is nice-to-have
	}

	// Capture the window name AFTER renaming - this is the name used for session directories
	// We store this in the session so that session end can archive to the correct directory
	// even if called from a different tmux window
	windowName, err := tmux.GetCurrentWindowName()
	if err != nil {
		// Fall back to session name if we can't get window name
		windowName = sessionName
	}

	// Start the session with the captured window name
	if err := store.Start(goal, windowName); err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}

	// Create active session handoff in project-specific location
	// This replaces the global ~/.orch workspace with project/.orch/session/{sessionName}/active/
	handoffPath, err := createActiveSessionHandoff(goal, sessionName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create active session handoff: %v\n", err)
		// Continue anyway - handoff is nice-to-have for interactive sessions
	}

	// Progressive Session Capture: Prompt for TLDR and Where We Started
	// Part of decision 2026-01-14 - capture context when it's freshest
	if handoffPath != "" {
		responses, err := promptForStartSections()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to collect start section responses: %v\n", err)
			// Continue anyway - we can still end the session and prompt for these later
		} else if len(responses) > 0 {
			if err := updateHandoffWithStartResponses(handoffPath, responses); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to update handoff with responses: %v\n", err)
			} else {
				fmt.Println("   ✅ Initial context captured in SESSION_HANDOFF.md")
			}
		}
	}

	// Log the session start
	logger := events.NewLogger(events.DefaultLogPath())
	eventData := map[string]interface{}{
		"goal":         goal,
		"was_active":   wasActive,
		"started_at":   time.Now().Format(session.TimeFormat),
		"session_name": sessionName,
	}
	if handoffPath != "" {
		eventData["handoff_path"] = handoffPath
	}
	event := events.Event{
		Type:      "session.started",
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	if wasActive {
		fmt.Println("Previous session ended.")
	}
	fmt.Printf("Session started: %s\n", goal)
	fmt.Printf("  Name:       %s\n", sessionName)
	fmt.Printf("  Start time: %s\n", time.Now().Format("15:04"))
	if handoffPath != "" {
		fmt.Printf("  Handoff:    %s\n", handoffPath)
	}

	// Surface reflection suggestions for high-count synthesis opportunities
	// This proactively surfaces consolidation needs that accumulated since last reflection
	surfaceReflectSuggestions()

	// Surface focus guidance - group ready issues into thematic threads
	// Part of Capture at Context principle - surface context when it matters
	surfaceFocusGuidance()

	return nil
}

// createActiveSessionHandoff creates SESSION_HANDOFF.md in {project}/.orch/session/{sessionName}/active/
// for progressive documentation during the session. This is the Active Directory Pattern:
// - Session start creates active/ with PreFilledSessionHandoffTemplate
// - Orchestrators fill sections as they work
// - Session end archives active/ to timestamped directory
func createActiveSessionHandoff(goal, sessionName string) (string, error) {
	// Get current working directory (project directory)
	projectDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get project directory: %w", err)
	}

	// Create active directory: .orch/session/{sessionName}/active/
	activeDir := filepath.Join(projectDir, ".orch", "session", sessionName, "active")
	if err := os.MkdirAll(activeDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create active directory: %w", err)
	}

	// Generate workspace name for SESSION_HANDOFF.md
	// Interactive sessions use "interactive-" prefix + date + time suffix
	dateStr := time.Now().Format("2006-01-02")
	timeStr := time.Now().Format("150405")
	workspaceName := fmt.Sprintf("interactive-%s-%s", dateStr, timeStr)
	startTime := time.Now().Format("2006-01-02 15:04")

	// Use "Interactive session" as default goal if empty
	sessionGoal := goal
	if sessionGoal == "" {
		sessionGoal = "Interactive session"
	}

	// Generate pre-filled SESSION_HANDOFF.md content using comprehensive template
	content, err := spawn.GeneratePreFilledSessionHandoff(workspaceName, sessionGoal, startTime)
	if err != nil {
		return "", fmt.Errorf("failed to generate session handoff: %w", err)
	}

	// Write SESSION_HANDOFF.md to active/
	handoffPath := filepath.Join(activeDir, "SESSION_HANDOFF.md")
	if err := os.WriteFile(handoffPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write session handoff: %w", err)
	}

	return handoffPath, nil
}

// ============================================================================
// Session Handoff Validation and Interactive Completion
// Implements "Capture at Context" principle: validate all sections, prompt for unfilled ones
// ============================================================================

// HandoffSection describes a section in the SESSION_HANDOFF.md that needs validation.
type HandoffSection struct {
	Name        string   // Display name for the section
	Placeholder string   // Pattern that indicates the section is unfilled
	Required    bool     // Whether the section must be filled
	SkipValue   string   // For optional sections, the value that means "skip"
	Prompt      string   // Question to ask user if unfilled
	Options     []string // Valid options (for choice-based sections)
}

// handoffSections defines all sections that need validation at session end.
// Order matters - sections are validated and prompted in this order.
var handoffSections = []HandoffSection{
	{
		Name:        "Outcome",
		Placeholder: "{success | partial | blocked | failed}",
		Required:    true,
		Prompt:      "Session outcome [success/partial/blocked/failed]",
		Options:     []string{"success", "partial", "blocked", "failed"},
	},
	{
		Name:        "TLDR",
		Placeholder: "[Fill within first 5 tool calls: What is this session trying to accomplish?]",
		Required:    true,
		Prompt:      "Brief summary of what this session accomplished",
	},
	{
		Name:        "Where We Ended",
		Placeholder: "{state of focus goal now}",
		Required:    true,
		Prompt:      "Current state of the focus goal (what's the situation now?)",
	},
	{
		Name:        "Next Recommendation",
		Placeholder: "{continue-focus | shift-focus | escalate | pause}",
		Required:    true,
		Prompt:      "Recommendation for next session [continue-focus/shift-focus/escalate/pause]",
		Options:     []string{"continue-focus", "shift-focus", "escalate", "pause"},
	},
	{
		Name:        "Evidence",
		Placeholder: "[Pattern 1:",
		Required:    false,
		SkipValue:   "nothing notable",
		Prompt:      "Notable patterns observed (or 'nothing notable' to skip)",
	},
	{
		Name:        "Knowledge",
		Placeholder: "{topic}:",
		Required:    false,
		SkipValue:   "none",
		Prompt:      "Key decisions/constraints discovered (or 'none' to skip)",
	},
	{
		Name:        "Friction",
		Placeholder: "[Tool gap or UX issue]",
		Required:    false,
		SkipValue:   "smooth",
		Prompt:      "Tooling or context friction encountered (or 'smooth' to skip)",
	},
}

// startSections defines sections to prompt for at session start.
// Part of Progressive Session Capture (decision 2026-01-14).
// These capture context when it's freshest: at session beginning.
var startSections = []HandoffSection{
	{
		Name:        "TLDR",
		Placeholder: "[Fill within first 5 tool calls: What is this session trying to accomplish?]",
		Required:    true,
		Prompt:      "What is this session trying to accomplish? (1-2 sentences)",
	},
	{
		Name:        "Where We Started",
		Placeholder: "[Fill within first 5 tool calls: What is the current state before you begin working?]",
		Required:    true,
		Prompt:      "What is the current state before you begin? (context for next session)",
	},
}

// ValidationResult holds the result of validating a session handoff.
type ValidationResult struct {
	Unfilled []HandoffSection // Sections that still have placeholders
	Content  string           // Current handoff content
}

// validateHandoff reads SESSION_HANDOFF.md and checks for unfilled sections.
// Returns the list of sections that still contain placeholder patterns.
func validateHandoff(activeDir string) (*ValidationResult, error) {
	handoffPath := filepath.Join(activeDir, "SESSION_HANDOFF.md")

	content, err := os.ReadFile(handoffPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read handoff: %w", err)
	}

	contentStr := string(content)
	var unfilled []HandoffSection

	for _, section := range handoffSections {
		if strings.Contains(contentStr, section.Placeholder) {
			unfilled = append(unfilled, section)
		}
	}

	return &ValidationResult{
		Unfilled: unfilled,
		Content:  contentStr,
	}, nil
}

// UserResponse holds a user's response to a section prompt.
type UserResponse struct {
	Section  HandoffSection
	Response string
}

// promptForUnfilledSections prompts the user for each unfilled section.
// Returns the collected responses.
func promptForUnfilledSections(unfilled []HandoffSection) ([]UserResponse, error) {
	if len(unfilled) == 0 {
		return nil, nil
	}

	reader := bufio.NewReader(os.Stdin)
	var responses []UserResponse

	fmt.Println("\n📋 Session Handoff Completion")
	fmt.Printf("   %d section(s) need to be filled before archiving:\n", len(unfilled))
	fmt.Println()

	for _, section := range unfilled {
		// Show section name and prompt
		if section.Required {
			fmt.Printf("   [REQUIRED] %s\n", section.Name)
		} else {
			fmt.Printf("   [Optional] %s\n", section.Name)
		}

		// Show options if available
		if len(section.Options) > 0 {
			fmt.Printf("   Options: %s\n", strings.Join(section.Options, ", "))
		}
		if !section.Required && section.SkipValue != "" {
			fmt.Printf("   (Enter '%s' to skip)\n", section.SkipValue)
		}

		fmt.Printf("   %s: ", section.Prompt)

		// Read response
		response, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read input: %w", err)
		}
		response = strings.TrimSpace(response)

		// Validate choice-based sections
		if len(section.Options) > 0 && response != "" {
			valid := false
			for _, opt := range section.Options {
				if response == opt {
					valid = true
					break
				}
			}
			if !valid {
				return nil, fmt.Errorf("invalid value for %s: %q (must be one of: %s)",
					section.Name, response, strings.Join(section.Options, ", "))
			}
		}

		// Check for empty required fields
		if section.Required && response == "" {
			return nil, fmt.Errorf("%s is required and cannot be empty", section.Name)
		}

		responses = append(responses, UserResponse{
			Section:  section,
			Response: response,
		})
		fmt.Println()
	}

	return responses, nil
}

// promptForStartSections prompts the user for session start sections (TLDR, Where We Started).
// Part of Progressive Session Capture - capture context when it's freshest.
// Returns the collected responses.
func promptForStartSections() ([]UserResponse, error) {
	reader := bufio.NewReader(os.Stdin)
	var responses []UserResponse

	fmt.Println("\n📋 Session Start - Capture Initial Context")
	fmt.Println("   (Part of Progressive Session Capture - these help with session handoffs)")
	fmt.Println()

	for _, section := range startSections {
		fmt.Printf("   %s:\n", section.Name)
		fmt.Printf("   %s\n", section.Prompt)
		fmt.Printf("   > ")

		// Read response
		response, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read input: %w", err)
		}
		response = strings.TrimSpace(response)

		// Check for empty required fields
		if section.Required && response == "" {
			return nil, fmt.Errorf("%s is required and cannot be empty", section.Name)
		}

		responses = append(responses, UserResponse{
			Section:  section,
			Response: response,
		})
		fmt.Println()
	}

	return responses, nil
}

// updateHandoffWithStartResponses updates the handoff file with session start responses.
// This replaces the placeholder text with actual user input.
func updateHandoffWithStartResponses(handoffPath string, responses []UserResponse) error {
	content, err := os.ReadFile(handoffPath)
	if err != nil {
		return fmt.Errorf("failed to read handoff: %w", err)
	}

	result := string(content)

	// Apply each response by replacing the placeholder
	for _, r := range responses {
		result = strings.ReplaceAll(result, r.Section.Placeholder, r.Response)
	}

	// Write updated content
	if err := os.WriteFile(handoffPath, []byte(result), 0644); err != nil {
		return fmt.Errorf("failed to write updated handoff: %w", err)
	}

	return nil
}

// updateHandoffWithResponses updates the handoff content with user responses.
// Also updates {end-time} placeholder.
func updateHandoffWithResponses(content string, responses []UserResponse, endTime string) string {
	result := content

	// Always update end-time
	result = strings.ReplaceAll(result, "{end-time}", endTime)

	// Apply each response
	for _, r := range responses {
		result = strings.ReplaceAll(result, r.Section.Placeholder, r.Response)
	}

	return result
}

// completeAndArchiveHandoff validates the handoff, prompts for unfilled sections,
// updates the file, and archives it. This is the main entry point for session end.
// Returns nil if no active handoff exists (legacy sessions).
func completeAndArchiveHandoff(projectDir, windowName string) error {
	activeDir := filepath.Join(projectDir, ".orch", "session", windowName, "active")

	// Check if active handoff exists
	if _, err := os.Stat(activeDir); os.IsNotExist(err) {
		// No active handoff - legacy session, nothing to complete
		return nil
	}

	// Validate handoff
	validation, err := validateHandoff(activeDir)
	if err != nil {
		return fmt.Errorf("failed to validate handoff: %w", err)
	}

	// Show current state
	if len(validation.Unfilled) == 0 {
		fmt.Println("\n✅ All handoff sections already filled")
	} else {
		// Prompt for unfilled sections
		responses, err := promptForUnfilledSections(validation.Unfilled)
		if err != nil {
			return fmt.Errorf("failed to collect responses: %w", err)
		}

		// Update handoff with responses
		endTime := time.Now().Format("2006-01-02 15:04")
		updatedContent := updateHandoffWithResponses(validation.Content, responses, endTime)

		// Write updated content
		handoffPath := filepath.Join(activeDir, "SESSION_HANDOFF.md")
		if err := os.WriteFile(handoffPath, []byte(updatedContent), 0644); err != nil {
			return fmt.Errorf("failed to write updated handoff: %w", err)
		}
	}

	// Archive the handoff
	if err := archiveActiveSessionHandoff(projectDir, windowName); err != nil {
		return fmt.Errorf("failed to archive handoff: %w", err)
	}

	return nil
}

// archiveActiveSessionHandoff moves {project}/.orch/session/{window}/active/ to a timestamped directory
// and updates the latest symlink. This completes the Active Directory Pattern lifecycle.
// The windowName parameter should be the window name stored in the session at start time,
// NOT the current window name (which may be different if session end is called from another window).
// Returns nil if active/ doesn't exist (not an error - session may predate active pattern).
func archiveActiveSessionHandoff(projectDir, windowName string) error {
	// Check if active/ directory exists
	activeDir := filepath.Join(projectDir, ".orch", "session", windowName, "active")
	if _, err := os.Stat(activeDir); os.IsNotExist(err) {
		// Active directory doesn't exist - this is OK (session may predate active pattern)
		return nil
	}

	// Create timestamped directory name (YYYY-MM-DD-HHMM format)
	timestamp := time.Now().Format("2006-01-02-1504")
	timestampedDir := filepath.Join(projectDir, ".orch", "session", windowName, timestamp)

	// Rename active/ to timestamped directory (atomic move)
	if err := os.Rename(activeDir, timestampedDir); err != nil {
		return fmt.Errorf("failed to rename active to timestamped directory: %w", err)
	}

	// Update latest symlink to point to new timestamped directory
	latestSymlink := filepath.Join(projectDir, ".orch", "session", windowName, "latest")

	// Remove existing symlink if present
	_ = os.Remove(latestSymlink)

	// Create new symlink (relative path to avoid absolute path issues)
	if err := os.Symlink(timestamp, latestSymlink); err != nil {
		return fmt.Errorf("failed to create latest symlink: %w", err)
	}

	fmt.Printf("\n📋 Session handoff archived: %s\n", timestampedDir)
	fmt.Printf("   Latest symlink updated: .orch/session/%s/latest -> %s\n", windowName, timestamp)

	return nil
}

// SynthesisWarningThreshold is the minimum count of investigations to show a warning.
// Matches SynthesisIssueThreshold in kb-cli to maintain consistency.
const SynthesisWarningThreshold = 10

// SuggestionFreshnessHours is the maximum age of suggestions to consider fresh.
// Suggestions older than this are considered stale and won't be shown.
const SuggestionFreshnessHours = 24

// InvestigationPromotionThreshold is the count above which session end will warn.
// Gates accumulation of promotion candidates that need triage.
const InvestigationPromotionThreshold = 5

// surfaceFocusGuidance loads ready issues and displays them grouped into thematic threads.
// This helps orchestrators orient at session start: "Here are your active threads. What's nagging you?"
// Part of Capture at Context principle.
func surfaceFocusGuidance() {
	guidance, err := focus.GenerateFocusGuidance()
	if err != nil {
		// Failed to load issues - silently skip (not critical)
		return
	}

	if guidance.TotalIssues == 0 {
		// No ready issues - brief message only
		fmt.Println("\n📋 No ready issues found")
		return
	}

	// Display formatted guidance
	fmt.Print(focus.FormatFocusGuidance(guidance))
}

// surfaceReflectSuggestions loads and displays synthesis warnings from reflect-suggestions.json.
// This proactively surfaces consolidation needs at session start so orchestrators are aware
// of accumulated investigation clusters that need synthesis into guides.
func surfaceReflectSuggestions() {
	suggestions, err := daemon.LoadSuggestions()
	if err != nil || suggestions == nil {
		// No suggestions file or failed to load - silently skip
		return
	}

	// Check freshness - skip stale suggestions
	if time.Since(suggestions.Timestamp).Hours() > SuggestionFreshnessHours {
		return
	}

	// Filter to high-count synthesis opportunities
	var highCount []daemon.SynthesisSuggestion
	for _, s := range suggestions.Synthesis {
		if s.Count >= SynthesisWarningThreshold {
			highCount = append(highCount, s)
		}
	}

	if len(highCount) == 0 {
		return
	}

	// Display synthesis warnings
	fmt.Println()
	fmt.Println("📚 SYNTHESIS OPPORTUNITIES")
	fmt.Printf("   %d topics have accumulated %d+ investigations:\n", len(highCount), SynthesisWarningThreshold)

	// Show top 5 topics
	maxShow := 5
	if len(highCount) < maxShow {
		maxShow = len(highCount)
	}
	for i := 0; i < maxShow; i++ {
		s := highCount[i]
		fmt.Printf("   • %s: %d investigations → kb create guide \"%s\"\n", s.Topic, s.Count, s.Topic)
	}
	if len(highCount) > maxShow {
		fmt.Printf("   ... and %d more topics\n", len(highCount)-maxShow)
	}
	fmt.Printf("   Run 'kb reflect --type synthesis' for details.\n")
}

// InvestigationPromotionItem represents a single investigation promotion candidate.
type InvestigationPromotionItem struct {
	File       string `json:"file"`
	Title      string `json:"title"`
	AgeDays    int    `json:"age_days"`
	Suggestion string `json:"suggestion"`
}

// InvestigationPromotionResult holds the JSON output from kb reflect --type investigation-promotion.
type InvestigationPromotionResult struct {
	InvestigationPromotion []InvestigationPromotionItem `json:"investigation_promotion"`
}

// checkInvestigationPromotions runs kb reflect --type investigation-promotion --format json
// and returns the count of promotion candidates. Returns 0 and logs warning on error.
func checkInvestigationPromotions() int {
	cmd := exec.Command("kb", "reflect", "--type", "investigation-promotion", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		// kb reflect may not be available or may fail - not critical, just skip
		return 0
	}

	var result InvestigationPromotionResult
	if err := json.Unmarshal(output, &result); err != nil {
		// Parse error - skip silently
		return 0
	}

	return len(result.InvestigationPromotion)
}

// gateInvestigationPromotions checks for accumulated investigation promotion candidates
// and prompts user to triage if above threshold. Returns error if user aborts.
// This is a gate at session end to prevent accumulation of promotion candidates.
func gateInvestigationPromotions() error {
	count := checkInvestigationPromotions()
	if count <= InvestigationPromotionThreshold {
		return nil // Below threshold, proceed
	}

	fmt.Println()
	fmt.Printf("⚠️  INVESTIGATION PROMOTION BACKLOG\n")
	fmt.Printf("   %d investigations need promotion review (threshold: %d)\n", count, InvestigationPromotionThreshold)
	fmt.Printf("   Run 'kb reflect --type investigation-promotion' to triage.\n")
	fmt.Println()

	// Prompt user to confirm proceeding
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("   Continue ending session anyway? (y/N): ")
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		fmt.Println("   Session end aborted. Please triage investigation promotions first.")
		return fmt.Errorf("session end aborted: investigation promotion backlog needs triage")
	}

	return nil
}

// ============================================================================
// Session Status Command
// ============================================================================

var sessionStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current session status with spawn reconciliation",
	Long: `Show current session status including spawns with reconciled states.

Spawn states are derived at query time by checking:
- OpenCode session liveness
- tmux window existence
- Beads issue status

This ensures accurate state rather than trusting potentially stale stored data.

States:
  active    - Agent is running (has live OpenCode session or tmux window)
  completed - Agent finished (beads issue closed, no live session)
  phantom   - Agent lost (beads issue open, but no live session)

Examples:
  orch session status         # Show status
  orch session status --json  # Output as JSON for scripting`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionStatus()
	},
}

// SessionStatusOutput is the JSON output format for session status.
type SessionStatusOutput struct {
	Active     bool                      `json:"active"`
	Goal       string                    `json:"goal,omitempty"`
	StartedAt  string                    `json:"started_at,omitempty"`
	Duration   string                    `json:"duration,omitempty"`
	Spawns     []session.SpawnStatus     `json:"spawns,omitempty"`
	Counts     *SpawnCounts              `json:"counts,omitempty"`
	Checkpoint *session.CheckpointStatus `json:"checkpoint,omitempty"`
}

// SpawnCounts summarizes spawn states.
type SpawnCounts struct {
	Total     int `json:"total"`
	Active    int `json:"active"`
	Completed int `json:"completed"`
	Phantom   int `json:"phantom"`
}

func runSessionStatus() error {
	store, err := session.New("")
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	sess := store.Get()

	// Build output
	output := SessionStatusOutput{
		Active: sess != nil,
	}

	if sess != nil {
		output.Goal = sess.Goal
		output.StartedAt = sess.StartedAt.Format(session.TimeFormat)
		output.Duration = formatSessionDuration(store.Duration())

		// Get spawn statuses with reconciliation
		statuses := store.GetSpawnStatuses(serverURL)
		output.Spawns = statuses

		// Count by state
		counts := &SpawnCounts{Total: len(statuses)}
		for _, s := range statuses {
			switch s.State {
			case "active":
				counts.Active++
			case "completed":
				counts.Completed++
			case "phantom":
				counts.Phantom++
			}
		}
		output.Counts = counts

		// Get checkpoint status using orchestrator thresholds
		// (orch session is for orchestrator sessions, not agent sessions)
		output.Checkpoint = store.GetCheckpointStatusWithType(session.SessionTypeOrchestrator)
	}

	// JSON output
	if sessionJSON {
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Human-readable output
	if !output.Active {
		fmt.Println("No active session")
		fmt.Println("\nStart a session with: orch session start \"your goal\"")
		return nil
	}

	fmt.Printf("Session active:\n")
	fmt.Printf("  Goal:     %s\n", output.Goal)
	fmt.Printf("  Duration: %s", output.Duration)

	// Show checkpoint status inline with duration
	if output.Checkpoint != nil {
		switch output.Checkpoint.Level {
		case "exceeded":
			fmt.Printf(" ⛔")
		case "strong":
			fmt.Printf(" 🔴")
		case "warning":
			fmt.Printf(" 🟡")
		}
	}
	fmt.Println()

	fmt.Printf("  Spawns:   %d total", output.Counts.Total)
	if output.Counts.Active > 0 {
		fmt.Printf(" (%d active", output.Counts.Active)
		if output.Counts.Completed > 0 {
			fmt.Printf(", %d completed", output.Counts.Completed)
		}
		if output.Counts.Phantom > 0 {
			fmt.Printf(", %d phantom", output.Counts.Phantom)
		}
		fmt.Printf(")")
	}
	fmt.Println()

	// Show spawn details if any
	if len(output.Spawns) > 0 {
		fmt.Println("\nSpawns:")
		for _, spawn := range output.Spawns {
			stateIcon := stateToIcon(spawn.State)
			age := formatSessionDuration(time.Since(spawn.SpawnedAt))
			fmt.Printf("  %s %s (%s) - %s ago\n", stateIcon, spawn.BeadsID, spawn.Skill, age)
		}
	}

	// Show checkpoint warning if applicable
	if output.Checkpoint != nil && output.Checkpoint.Level != "ok" {
		fmt.Println()
		switch output.Checkpoint.Level {
		case "exceeded":
			fmt.Printf("⛔ CHECKPOINT EXCEEDED: %s\n", output.Checkpoint.Message)
			fmt.Println("   Session has run too long. Quality may be degraded.")
			fmt.Println("   Action: Run 'orch session end' and start fresh.")
		case "strong":
			fmt.Printf("🔴 CHECKPOINT STRONGLY RECOMMENDED: %s\n", output.Checkpoint.Message)
			fmt.Printf("   Time until max: %s\n", formatSessionDuration(output.Checkpoint.NextThreshold))
			fmt.Println("   Action: Write SESSION_HANDOFF.md, consider ending session.")
		case "warning":
			fmt.Printf("🟡 CHECKPOINT SUGGESTED: %s\n", output.Checkpoint.Message)
			fmt.Printf("   Time until strong warning: %s\n", formatSessionDuration(output.Checkpoint.NextThreshold))
			fmt.Println("   Action: Assess progress, write interim handoff if needed.")
		}
	}

	return nil
}

// stateToIcon converts state to a visual indicator.
func stateToIcon(state string) string {
	switch state {
	case "active":
		return "🟢"
	case "completed":
		return "✅"
	case "phantom":
		return "👻"
	default:
		return "❓"
	}
}

// formatSessionDuration formats a duration in a human-readable way for session display.
func formatSessionDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, mins)
}

// ============================================================================
// Session End Command
// ============================================================================

var sessionEndCmd = &cobra.Command{
	Use:   "end",
	Short: "End the current session",
	Long: `End the current orchestrator work session.

This clears the session state. Use before:
- Taking a break
- Handing off to another orchestrator
- Changing focus to a different goal

The session summary is logged for posterity.

Examples:
  orch session end`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionEnd()
	},
}

func runSessionEnd() error {
	store, err := session.New("")
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	if !store.IsActive() {
		fmt.Println("No active session to end")
		return nil
	}

	// Gate: Check for accumulated investigation promotion candidates
	// This prevents backlog accumulation by prompting triage before session end
	if err := gateInvestigationPromotions(); err != nil {
		return err
	}

	// Get session info before ending - IMPORTANT: Get the session object to access WindowName
	// which was captured at session start. This is used for archiving, NOT GetCurrentWindowName().
	sess := store.Get()
	duration := store.Duration()
	spawnCount := store.SpawnCount()

	// Get spawn statuses for final summary
	statuses := store.GetSpawnStatuses(serverURL)
	activeCount := 0
	for _, s := range statuses {
		if s.State == "active" {
			activeCount++
		}
	}

	// Get project directory
	projectDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to get project directory: %v\n", err)
	} else {
		// Use the stored window name from session start, NOT GetCurrentWindowName()
		// This ensures we archive to the correct directory even if called from a different window
		windowName := sess.WindowName
		if windowName == "" {
			// Fallback for sessions created before WindowName was added
			windowName, err = tmux.GetCurrentWindowName()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to get window name: %v\n", err)
				windowName = "default"
			}
		}

		// Complete and archive the session handoff
		// This validates unfilled sections, prompts for completion, then archives
		if err := completeAndArchiveHandoff(projectDir, windowName); err != nil {
			// Only warn - not all sessions will have active handoffs (pre-active-pattern sessions)
			fmt.Fprintf(os.Stderr, "Warning: failed to complete/archive session handoff: %v\n", err)
		}
	}

	// End the session
	ended, err := store.End()
	if err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	// Log the session end
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "session.ended",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"goal":          ended.Goal,
			"started_at":    ended.StartedAt.Format(session.TimeFormat),
			"duration":      duration.String(),
			"spawn_count":   spawnCount,
			"active_at_end": activeCount,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	fmt.Printf("Session ended: %s\n", ended.Goal)
	fmt.Printf("  Duration:  %s\n", formatSessionDuration(duration))
	fmt.Printf("  Spawns:    %d total\n", spawnCount)

	if activeCount > 0 {
		fmt.Printf("\n⚠️  %d agent(s) still active. Use 'orch status' to monitor.\n", activeCount)
	}

	// Show checkpoint advice based on session duration using orchestrator thresholds
	orchThresholds := session.DefaultOrchestratorThresholds()
	if duration >= orchThresholds.Max {
		fmt.Printf("\n⛔ Session exceeded %s checkpoint max.\n", formatSessionDuration(orchThresholds.Max))
		fmt.Println("   Consider shorter sessions to maintain quality.")
	} else if duration >= orchThresholds.Strong {
		fmt.Printf("\n🟡 Session was %s+. Good to hand off, but review quality of late work.\n", formatSessionDuration(orchThresholds.Strong))
	}

	return nil
}

// ============================================================================
// Session Resume Command
// ============================================================================

var sessionResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume orchestrator session by injecting prior handoff",
	Long: `Resume an orchestrator session by discovering and displaying the most recent SESSION_HANDOFF.md.

This command walks up the directory tree to find .orch/session/latest/SESSION_HANDOFF.md
and displays it in the format appropriate for the use case.

Modes:
  Default (interactive):  Display formatted handoff for manual review
  --for-injection:        Output condensed format for hook injection (no decorations)
  --check:                Just check if handoff exists (exit code 0 if yes, 1 if no)

Discovery:
  1. Starts from current directory
  2. Walks up directory tree looking for .orch/session/latest symlink
  3. Reads SESSION_HANDOFF.md from the symlink target
  4. Fails gracefully if no handoff found (valid for fresh sessions)

Examples:
  orch session resume                  # Interactive display
  orch session resume --for-injection  # For hooks (condensed format)
  orch session resume --check          # Check existence only`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionResume()
	},
}

func runSessionResume() error {
	// Discover handoff by walking up directory tree
	handoffPath, err := discoverSessionHandoff()
	if err != nil {
		if resumeCheck {
			// Exit code 1 for --check mode when handoff not found
			os.Exit(1)
		}
		return err
	}

	if resumeCheck {
		// Exit code 0 for --check mode when handoff exists
		os.Exit(0)
	}

	// Read the handoff content
	content, err := os.ReadFile(handoffPath)
	if err != nil {
		return fmt.Errorf("failed to read handoff: %w", err)
	}

	// Output based on mode
	if resumeForInjection {
		// Condensed format for hooks (just the content, no decorations)
		fmt.Print(string(content))
	} else {
		// Interactive format with metadata
		fmt.Printf("📋 Session Handoff\n")
		fmt.Printf("   Source: %s\n", handoffPath)
		fmt.Println()
		fmt.Print(string(content))
	}

	return nil
}

// parseDurationFromHandoff reads a SESSION_HANDOFF.md file and extracts the session duration.
// Parses the Duration line in format: "**Duration:** YYYY-MM-DD HH:MM → YYYY-MM-DD HH:MM"
// Returns duration in minutes, or -1 if duration cannot be parsed (unparseable format or incomplete session).
func parseDurationFromHandoff(handoffPath string) int {
	file, err := os.Open(handoffPath)
	if err != nil {
		return -1
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	maxLines := 20 // Duration line is always in the header

	for scanner.Scan() && lineCount < maxLines {
		line := scanner.Text()
		lineCount++

		// Look for Duration line: **Duration:** YYYY-MM-DD HH:MM → YYYY-MM-DD HH:MM
		if strings.HasPrefix(line, "**Duration:**") {
			// Extract the content after "**Duration:** "
			content := strings.TrimPrefix(line, "**Duration:**")
			content = strings.TrimSpace(content)

			// Split by arrow (→) to get start and end timestamps
			parts := strings.Split(content, "→")
			if len(parts) != 2 {
				return -1 // Not in expected format
			}

			startStr := strings.TrimSpace(parts[0])
			endStr := strings.TrimSpace(parts[1])

			// Handle same-day format where end is just HH:MM
			// e.g., "2026-01-14 11:52 → 12:30 (38m)"
			// Strip optional duration suffix like "(38m)"
			if idx := strings.Index(endStr, "("); idx != -1 {
				endStr = strings.TrimSpace(endStr[:idx])
			}

			// Parse start timestamp (always has date)
			startTime, err := time.Parse("2006-01-02 15:04", startStr)
			if err != nil {
				return -1
			}

			// Try parsing end as full timestamp first
			endTime, err := time.Parse("2006-01-02 15:04", endStr)
			if err != nil {
				// Try parsing as time only (same day)
				endTime, err = time.Parse("15:04", endStr)
				if err != nil {
					return -1
				}
				// Use start date with end time
				endTime = time.Date(
					startTime.Year(), startTime.Month(), startTime.Day(),
					endTime.Hour(), endTime.Minute(), 0, 0, time.UTC,
				)
			}

			// Check for incomplete sessions (end time is a placeholder)
			if endTime.Before(startTime) || endTime.Equal(startTime) {
				return -1
			}

			duration := endTime.Sub(startTime)
			return int(duration.Minutes())
		}
	}

	return -1 // Duration line not found
}

// scanAllWindowsForMostRecent scans all window-scoped session directories in .orch/session/
// and returns the most recent SESSION_HANDOFF.md by comparing timestamps.
// Prefers substantive sessions (≥5 minutes) over brief test sessions.
// Returns empty string if no handoffs found across all windows.
func scanAllWindowsForMostRecent(sessionBaseDir string) (string, error) {
	// Read all entries in .orch/session/
	entries, err := os.ReadDir(sessionBaseDir)
	if err != nil {
		return "", err
	}

	// Track two candidates:
	// - mostRecentSubstantive: sessions ≥5 minutes (real work sessions)
	// - mostRecentAny: all sessions regardless of duration (fallback)
	const minSubstantiveMinutes = 5

	var mostRecentSubstantivePath string
	var mostRecentSubstantiveTimestamp string
	var mostRecentAnyPath string
	var mostRecentAnyTimestamp string

	// Scan each window directory
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		windowName := entry.Name()

		// Skip legacy timestamp directories (start with digit) and special directories
		if len(windowName) > 0 && windowName[0] >= '0' && windowName[0] <= '9' {
			continue
		}
		if windowName == "latest" || windowName == "active" {
			continue
		}

		// Check for latest symlink in this window's directory
		latestPath := filepath.Join(sessionBaseDir, windowName, "latest")
		stat, err := os.Lstat(latestPath)
		if err != nil {
			continue // No latest symlink for this window
		}

		// Resolve the symlink to get the timestamp directory
		var sessionDir string
		if stat.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(latestPath)
			if err != nil {
				continue
			}
			if !filepath.IsAbs(target) {
				sessionDir = filepath.Join(sessionBaseDir, windowName, target)
			} else {
				sessionDir = target
			}
		} else {
			sessionDir = latestPath
		}

		// Check if SESSION_HANDOFF.md exists
		handoffPath := filepath.Join(sessionDir, "SESSION_HANDOFF.md")
		if _, err := os.Stat(handoffPath); err != nil {
			continue // No handoff in this session directory
		}

		// Extract timestamp from directory name for comparison
		// Format: YYYY-MM-DD-HHMM (e.g., "2026-01-13-0830")
		timestamp := filepath.Base(sessionDir)

		// Always track mostRecentAny (fallback candidate)
		if timestamp > mostRecentAnyTimestamp {
			mostRecentAnyTimestamp = timestamp
			mostRecentAnyPath = handoffPath
		}

		// Parse duration to determine if this is a substantive session
		durationMinutes := parseDurationFromHandoff(handoffPath)
		if durationMinutes >= minSubstantiveMinutes {
			// This is a substantive session (≥5 minutes)
			if timestamp > mostRecentSubstantiveTimestamp {
				mostRecentSubstantiveTimestamp = timestamp
				mostRecentSubstantivePath = handoffPath
			}
		}
	}

	// Prefer substantive sessions over brief test sessions
	if mostRecentSubstantivePath != "" {
		return mostRecentSubstantivePath, nil
	}
	return mostRecentAnyPath, nil
}

// discoverSessionHandoff walks up the directory tree to find the most relevant SESSION_HANDOFF.md.
// Returns the full path to the handoff file, or an error if not found.
// Discovery order prioritizes recency while respecting active sessions:
// 1. Current window's active directory (mid-session resume - your in-progress work)
// 2. Cross-window scan for most recent (includes all windows - user wants latest context)
// 3. Legacy non-window-scoped structure (backward compatibility)
//
// Note: Window-scoping is for WRITING (prevent clobbering), not reading.
// For resume, users want the most recent context regardless of window name.
func discoverSessionHandoff() (string, error) {
	// Get current tmux window name (or "default" if not in tmux)
	windowName, err := tmux.GetCurrentWindowName()
	if err != nil {
		return "", fmt.Errorf("failed to get tmux window name: %w", err)
	}

	// Start from current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Walk up the directory tree
	dir := currentDir
	for {
		// PRIORITY 1: Current window's active/ (mid-session resume)
		// If you're mid-session in this window, that's your in-progress work
		activePath := filepath.Join(dir, ".orch", "session", windowName, "active")
		if _, err := os.Stat(activePath); err == nil {
			handoffPath := filepath.Join(activePath, "SESSION_HANDOFF.md")
			if _, err := os.Stat(handoffPath); err == nil {
				return handoffPath, nil
			}
		}

		// PRIORITY 2: Cross-window scan for most recent handoff
		// This finds the most recent archived session across ALL windows (including current)
		// User starting new session wants latest context, not stale window-matched handoff
		sessionBaseDir := filepath.Join(dir, ".orch", "session")
		if _, err := os.Stat(sessionBaseDir); err == nil {
			mostRecentPath, err := scanAllWindowsForMostRecent(sessionBaseDir)
			if err == nil && mostRecentPath != "" {
				return mostRecentPath, nil
			}
		}

		// BACKWARD COMPATIBILITY: Check for old non-window-scoped structure
		// This fallback enables session resume for handoffs created before window-scoping was added
		legacyLatestPath := filepath.Join(dir, ".orch", "session", "latest")
		if stat, err := os.Lstat(legacyLatestPath); err == nil {
			var sessionDir string
			if stat.Mode()&os.ModeSymlink != 0 {
				// It's a symlink - resolve it
				target, err := os.Readlink(legacyLatestPath)
				if err != nil {
					return "", fmt.Errorf("failed to read legacy latest symlink: %w", err)
				}
				// If target is relative, resolve it relative to .orch/session/
				if !filepath.IsAbs(target) {
					sessionDir = filepath.Join(dir, ".orch", "session", target)
				} else {
					sessionDir = target
				}
			} else {
				// It's a directory (not a symlink)
				sessionDir = legacyLatestPath
			}

			// Check for SESSION_HANDOFF.md in the legacy session directory
			handoffPath := filepath.Join(sessionDir, "SESSION_HANDOFF.md")
			if _, err := os.Stat(handoffPath); err == nil {
				// Found legacy handoff - emit warning about migration
				fmt.Fprintf(os.Stderr, "⚠️  Using legacy session handoff structure.\n")
				fmt.Fprintf(os.Stderr, "   Run 'orch session migrate' to update to window-scoped structure.\n")
				fmt.Fprintf(os.Stderr, "   (This prevents concurrent orchestrators from clobbering each other's context)\n\n")
				return handoffPath, nil
			}
		}

		// PROJECT BOUNDARY CHECK: If .orch/ exists in this directory, this is the project root.
		// Don't continue walking up - session handoffs are project-specific and should not
		// leak across project boundaries. This prevents finding parent project or global handoffs.
		orchDir := filepath.Join(dir, ".orch")
		if _, err := os.Stat(orchDir); err == nil {
			// This is the project root - we've checked all possible session handoff locations
			// within this project (window-scoped, cross-window, and legacy). Stop the walk.
			break
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	// Enhanced error message showing both paths checked
	windowScopedPath := fmt.Sprintf(".orch/session/%s/latest/SESSION_HANDOFF.md", windowName)
	legacyPath := ".orch/session/latest/SESSION_HANDOFF.md"
	return "", fmt.Errorf("no session handoff found for window %q\nChecked:\n  - Window-scoped: %s\n  - Legacy: %s", windowName, windowScopedPath, legacyPath)
}

// ============================================================================
// Session Validate Command - Check handoff quality without ending session
// ============================================================================

var sessionValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Show unfilled handoff sections without ending session",
	Long: `Validate SESSION_HANDOFF.md quality by showing unfilled sections.

This command checks the active session handoff for placeholder patterns
and displays which sections still need to be filled. Unlike 'session end',
it does NOT prompt for input or archive the handoff.

Use cases:
- Check handoff quality mid-session
- Debug validation logic
- Verify handoff is ready before ending session

The command looks for the active session handoff in:
  .orch/session/{window-name}/active/SESSION_HANDOFF.md

If no active handoff exists, it reports that state.

Examples:
  orch session validate          # Human-readable output
  orch session validate --json   # Machine-readable output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionValidate()
	},
}

// ValidationOutput is the JSON output format for session validate.
type ValidationOutput struct {
	Found           bool                    `json:"found"`
	HandoffPath     string                  `json:"handoff_path,omitempty"`
	WindowName      string                  `json:"window_name"`
	TotalSections   int                     `json:"total_sections"`
	UnfilledCount   int                     `json:"unfilled_count"`
	RequiredFilled  int                     `json:"required_filled"`
	RequiredTotal   int                     `json:"required_total"`
	OptionalFilled  int                     `json:"optional_filled"`
	OptionalTotal   int                     `json:"optional_total"`
	UnfilledDetails []ValidationSectionInfo `json:"unfilled_details,omitempty"`
}

// ValidationSectionInfo describes an unfilled section for JSON output.
type ValidationSectionInfo struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Placeholder string `json:"placeholder"`
	Prompt      string `json:"prompt,omitempty"`
}

func runSessionValidate() error {
	// Get window name from active session or current tmux window
	windowName, err := getWindowNameForValidation()
	if err != nil {
		if validateJSON {
			return outputValidationJSON(&ValidationOutput{
				Found:      false,
				WindowName: "",
			})
		}
		return err
	}

	// Get project directory
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get project directory: %w", err)
	}

	// Look for active handoff
	activeDir := filepath.Join(projectDir, ".orch", "session", windowName, "active")
	handoffPath := filepath.Join(activeDir, "SESSION_HANDOFF.md")

	// Check if active handoff exists
	if _, err := os.Stat(handoffPath); os.IsNotExist(err) {
		if validateJSON {
			return outputValidationJSON(&ValidationOutput{
				Found:      false,
				WindowName: windowName,
			})
		}
		fmt.Printf("No active handoff found for window %q\n", windowName)
		fmt.Printf("  Expected path: %s\n", handoffPath)
		fmt.Println("\nStart a session with: orch session start \"your goal\"")
		return nil
	}

	// Validate the handoff
	validation, err := validateHandoff(activeDir)
	if err != nil {
		return fmt.Errorf("failed to validate handoff: %w", err)
	}

	// Count required vs optional sections
	requiredTotal := 0
	optionalTotal := 0
	for _, section := range handoffSections {
		if section.Required {
			requiredTotal++
		} else {
			optionalTotal++
		}
	}

	// Count unfilled by type
	unfilledRequired := 0
	unfilledOptional := 0
	for _, section := range validation.Unfilled {
		if section.Required {
			unfilledRequired++
		} else {
			unfilledOptional++
		}
	}

	// Build output
	output := &ValidationOutput{
		Found:          true,
		HandoffPath:    handoffPath,
		WindowName:     windowName,
		TotalSections:  len(handoffSections),
		UnfilledCount:  len(validation.Unfilled),
		RequiredFilled: requiredTotal - unfilledRequired,
		RequiredTotal:  requiredTotal,
		OptionalFilled: optionalTotal - unfilledOptional,
		OptionalTotal:  optionalTotal,
	}

	// Add unfilled details
	for _, section := range validation.Unfilled {
		output.UnfilledDetails = append(output.UnfilledDetails, ValidationSectionInfo{
			Name:        section.Name,
			Required:    section.Required,
			Placeholder: section.Placeholder,
			Prompt:      section.Prompt,
		})
	}

	if validateJSON {
		return outputValidationJSON(output)
	}

	// Human-readable output
	fmt.Printf("📋 Session Handoff Validation\n")
	fmt.Printf("   Window: %s\n", windowName)
	fmt.Printf("   Path:   %s\n\n", handoffPath)

	if len(validation.Unfilled) == 0 {
		fmt.Println("✅ All sections filled!")
		fmt.Printf("   Required: %d/%d filled\n", requiredTotal, requiredTotal)
		fmt.Printf("   Optional: %d/%d filled\n", optionalTotal, optionalTotal)
		fmt.Println("\n   Ready for: orch session end")
		return nil
	}

	// Show summary
	fmt.Printf("Status: %d/%d sections need attention\n\n", len(validation.Unfilled), len(handoffSections))

	// Group by required vs optional
	var requiredUnfilled, optionalUnfilled []HandoffSection
	for _, section := range validation.Unfilled {
		if section.Required {
			requiredUnfilled = append(requiredUnfilled, section)
		} else {
			optionalUnfilled = append(optionalUnfilled, section)
		}
	}

	// Show required unfilled
	if len(requiredUnfilled) > 0 {
		fmt.Printf("🔴 Required (%d unfilled):\n", len(requiredUnfilled))
		for _, section := range requiredUnfilled {
			fmt.Printf("   • %s\n", section.Name)
			if len(section.Options) > 0 {
				fmt.Printf("     Options: %s\n", strings.Join(section.Options, ", "))
			}
		}
		fmt.Println()
	} else {
		fmt.Printf("✅ Required: %d/%d filled\n\n", requiredTotal, requiredTotal)
	}

	// Show optional unfilled
	if len(optionalUnfilled) > 0 {
		fmt.Printf("🟡 Optional (%d unfilled):\n", len(optionalUnfilled))
		for _, section := range optionalUnfilled {
			skipInfo := ""
			if section.SkipValue != "" {
				skipInfo = fmt.Sprintf(" (skip with: %q)", section.SkipValue)
			}
			fmt.Printf("   • %s%s\n", section.Name, skipInfo)
		}
		fmt.Println()
	} else {
		fmt.Printf("✅ Optional: %d/%d filled\n\n", optionalTotal, optionalTotal)
	}

	// Show next action
	if len(requiredUnfilled) > 0 {
		fmt.Println("Next: Fill required sections in SESSION_HANDOFF.md, or run 'orch session end' to fill interactively")
	} else {
		fmt.Println("Next: Ready for 'orch session end' (optional sections will be prompted)")
	}

	return nil
}

// getWindowNameForValidation returns the window name to use for validation.
// It first checks for an active session (which stores the window name from session start),
// then falls back to the current tmux window name.
func getWindowNameForValidation() (string, error) {
	// Try to get window name from active session first
	store, err := session.New("")
	if err == nil && store.IsActive() {
		sess := store.Get()
		if sess != nil && sess.WindowName != "" {
			return sess.WindowName, nil
		}
	}

	// Fall back to current tmux window
	return tmux.GetCurrentWindowName()
}

// outputValidationJSON marshals and prints validation output as JSON.
func outputValidationJSON(output *ValidationOutput) error {
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// ============================================================================
// Session Migrate Command - Migrate legacy handoffs to window-scoped structure
// ============================================================================

var sessionMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate legacy session handoffs to window-scoped structure",
	Long: `Migrate legacy session handoffs to window-scoped structure.

Before window-scoping was added, session handoffs were stored in:
  .orch/session/{timestamp}/SESSION_HANDOFF.md

After window-scoping, they're stored in:
  .orch/session/{window-name}/{timestamp}/SESSION_HANDOFF.md

This command migrates old handoffs to the new structure.

Examples:
  orch session migrate              # Migrate to current window
  orch session migrate --all        # Show migration status for all windows`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessionMigrate()
	},
}

func runSessionMigrate() error {
	// Get current directory to find .orch/session
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find project root by walking up to .orch directory
	projectDir := currentDir
	for {
		sessionDir := filepath.Join(projectDir, ".orch", "session")
		if _, err := os.Stat(sessionDir); err == nil {
			break
		}
		parent := filepath.Dir(projectDir)
		if parent == projectDir {
			return fmt.Errorf("no .orch/session directory found (not in an orch-managed project)")
		}
		projectDir = parent
	}

	sessionBaseDir := filepath.Join(projectDir, ".orch", "session")

	// Get current window name for migration target
	windowName, err := tmux.GetCurrentWindowName()
	if err != nil {
		return fmt.Errorf("failed to get window name: %w", err)
	}

	// Check for legacy handoffs (non-window-scoped directories)
	entries, err := os.ReadDir(sessionBaseDir)
	if err != nil {
		return fmt.Errorf("failed to read session directory: %w", err)
	}

	var legacyDirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Legacy directories are timestamp format: YYYY-MM-DD-HHMM
		// Window-scoped directories are names (e.g., "default", "pw", "og-feat-...")
		name := entry.Name()
		// Check if it looks like a timestamp (starts with digit)
		if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
			legacyDirs = append(legacyDirs, name)
		}
	}

	if len(legacyDirs) == 0 {
		fmt.Println("✅ No legacy handoffs found - already using window-scoped structure")
		return nil
	}

	// Show what will be migrated
	fmt.Printf("Found %d legacy handoff(s) to migrate:\n\n", len(legacyDirs))
	for _, dir := range legacyDirs {
		handoffPath := filepath.Join(sessionBaseDir, dir, "SESSION_HANDOFF.md")
		if _, err := os.Stat(handoffPath); err == nil {
			fmt.Printf("  • %s → .orch/session/%s/%s\n", dir, windowName, dir)
		}
	}

	fmt.Printf("\nMigrate to window-scoped structure for window %q? (y/N): ", windowName)
	var response string
	fmt.Scanln(&response)

	if response != "y" && response != "Y" {
		fmt.Println("Migration cancelled")
		return nil
	}

	// Perform migration
	windowScopedDir := filepath.Join(sessionBaseDir, windowName)
	if err := os.MkdirAll(windowScopedDir, 0755); err != nil {
		return fmt.Errorf("failed to create window-scoped directory: %w", err)
	}

	migratedCount := 0
	for _, dir := range legacyDirs {
		sourcePath := filepath.Join(sessionBaseDir, dir)
		destPath := filepath.Join(windowScopedDir, dir)

		// Check if handoff exists
		handoffPath := filepath.Join(sourcePath, "SESSION_HANDOFF.md")
		if _, err := os.Stat(handoffPath); err != nil {
			// Skip directories without handoffs
			continue
		}

		// Move the directory
		if err := os.Rename(sourcePath, destPath); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to migrate %s: %v\n", dir, err)
			continue
		}
		migratedCount++
	}

	// Update latest symlink to point to most recent migrated handoff
	if migratedCount > 0 {
		// Find most recent timestamp directory
		var latestTimestamp string
		entries, _ := os.ReadDir(windowScopedDir)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if name > latestTimestamp && len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
				latestTimestamp = name
			}
		}

		if latestTimestamp != "" {
			latestSymlink := filepath.Join(windowScopedDir, "latest")
			_ = os.Remove(latestSymlink) // Remove old symlink if exists
			if err := os.Symlink(latestTimestamp, latestSymlink); err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Failed to update latest symlink: %v\n", err)
			}
		}
	}

	// Remove legacy latest symlink at root level
	legacyLatest := filepath.Join(sessionBaseDir, "latest")
	if _, err := os.Lstat(legacyLatest); err == nil {
		if err := os.Remove(legacyLatest); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to remove legacy latest symlink: %v\n", err)
		}
	}

	fmt.Printf("\n✅ Successfully migrated %d handoff(s) to window-scoped structure\n", migratedCount)
	fmt.Printf("   Window: %s\n", windowName)
	fmt.Printf("   Location: .orch/session/%s/\n", windowName)

	return nil
}

// ============================================================================
// Handoff Update After Complete - Progressive Capture
// Part of "Capture at Context" principle: update handoff when agent completes
// ============================================================================

// SpawnCompletionInfo holds information about a completed spawn for handoff update.
type SpawnCompletionInfo struct {
	WorkspaceName string // e.g., "og-feat-auth-middleware-14jan-a1b2"
	BeadsID       string // e.g., "orch-go-abc1"
	Skill         string // e.g., "feature-impl"
	Outcome       string // success | partial | failed
	KeyFinding    string // 1-line insight from the completion
}

// findActiveSessionHandoff finds the active SESSION_HANDOFF.md for the current session.
// Walks up from projectDir looking for .orch/session/{windowName}/active/SESSION_HANDOFF.md
// Returns empty string if no active handoff exists.
func findActiveSessionHandoff(projectDir string) string {
	// Get current tmux window name (session identifier)
	windowName, err := tmux.GetCurrentWindowName()
	if err != nil {
		return ""
	}

	// Look for active handoff in project's session directory
	activePath := filepath.Join(projectDir, ".orch", "session", windowName, "active", "SESSION_HANDOFF.md")
	if _, err := os.Stat(activePath); err == nil {
		return activePath
	}

	return ""
}

// promptSpawnCompletion prompts for spawn outcome and key finding after an agent completes.
// Returns nil values if user skips the prompt.
func promptSpawnCompletion(workspaceName, beadsID, skill string) (*SpawnCompletionInfo, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n📋 Session Handoff - Spawn Completion")
	fmt.Printf("   Agent: %s (%s)\n", workspaceName, beadsID)
	fmt.Println()

	// Prompt for outcome
	fmt.Print("   Outcome [success/partial/failed] (Enter to skip): ")
	outcomeStr, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read outcome: %w", err)
	}
	outcomeStr = strings.TrimSpace(outcomeStr)

	// Allow skipping
	if outcomeStr == "" {
		fmt.Println("   Skipped spawn update")
		return nil, nil
	}

	// Validate outcome
	validOutcomes := []string{"success", "partial", "failed"}
	valid := false
	for _, v := range validOutcomes {
		if outcomeStr == v {
			valid = true
			break
		}
	}
	if !valid {
		return nil, fmt.Errorf("invalid outcome: %q (must be one of: success, partial, failed)", outcomeStr)
	}

	// Prompt for key finding
	fmt.Print("   Key finding (1-line insight, Enter to skip): ")
	keyFinding, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read key finding: %w", err)
	}
	keyFinding = strings.TrimSpace(keyFinding)

	if keyFinding == "" {
		keyFinding = "Completed"
	}

	return &SpawnCompletionInfo{
		WorkspaceName: workspaceName,
		BeadsID:       beadsID,
		Skill:         skill,
		Outcome:       outcomeStr,
		KeyFinding:    keyFinding,
	}, nil
}

// promptEvidenceAndKnowledge prompts for optional evidence and knowledge additions.
// Returns empty strings if user skips.
func promptEvidenceAndKnowledge() (evidence, knowledge string, err error) {
	reader := bufio.NewReader(os.Stdin)

	// Prompt for evidence (pattern observation)
	fmt.Println()
	fmt.Print("   Evidence pattern (optional, Enter to skip): ")
	evidence, err = reader.ReadString('\n')
	if err != nil {
		return "", "", fmt.Errorf("failed to read evidence: %w", err)
	}
	evidence = strings.TrimSpace(evidence)

	// Prompt for knowledge (decision/constraint)
	fmt.Print("   Knowledge learned (decision/constraint, Enter to skip): ")
	knowledge, err = reader.ReadString('\n')
	if err != nil {
		return "", "", fmt.Errorf("failed to read knowledge: %w", err)
	}
	knowledge = strings.TrimSpace(knowledge)

	return evidence, knowledge, nil
}

// updateSpawnsTable adds a completed spawn row to the handoff's Spawns table.
// Inserts under "### Completed" section.
func updateSpawnsTable(content string, info *SpawnCompletionInfo) string {
	// Find the "### Completed" table and add a row
	completedMarker := "### Completed"
	idx := strings.Index(content, completedMarker)
	if idx == -1 {
		return content // No Completed section found
	}

	// Find the table header row (starts with |)
	tableStart := strings.Index(content[idx:], "| Agent |")
	if tableStart == -1 {
		return content
	}
	tableStart += idx

	// Find the separator row (|-----)
	separatorIdx := strings.Index(content[tableStart:], "|----")
	if separatorIdx == -1 {
		return content
	}
	separatorEnd := tableStart + separatorIdx
	// Find end of separator line
	newlineIdx := strings.Index(content[separatorEnd:], "\n")
	if newlineIdx == -1 {
		return content
	}
	insertPoint := separatorEnd + newlineIdx + 1

	// Check if there's a placeholder row (contains {workspace})
	// Look at the next line after separator
	restOfContent := content[insertPoint:]
	nextNewline := strings.Index(restOfContent, "\n")
	var nextLine string
	if nextNewline != -1 {
		nextLine = restOfContent[:nextNewline]
	} else {
		nextLine = restOfContent
	}

	// Build new row
	newRow := fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
		info.WorkspaceName, info.BeadsID, info.Skill, info.Outcome, info.KeyFinding)

	// If next line is a placeholder, replace it; otherwise insert before it
	if strings.Contains(nextLine, "{workspace}") || strings.Contains(nextLine, "{beads-id}") {
		// Replace placeholder row
		if nextNewline != -1 {
			return content[:insertPoint] + newRow + content[insertPoint+nextNewline+1:]
		}
		return content[:insertPoint] + newRow
	}

	// Insert new row
	return content[:insertPoint] + newRow + content[insertPoint:]
}

// updateEvidenceSection adds a pattern observation to the Evidence section.
func updateEvidenceSection(content, evidence string) string {
	if evidence == "" {
		return content
	}

	// Find "## Evidence" section
	evidenceMarker := "## Evidence"
	idx := strings.Index(content, evidenceMarker)
	if idx == -1 {
		return content
	}

	// Find "### Patterns Across Agents" subsection
	patternMarker := "### Patterns Across Agents"
	patternIdx := strings.Index(content[idx:], patternMarker)
	if patternIdx == -1 {
		return content
	}
	patternIdx += idx

	// Find end of subsection header line
	newlineIdx := strings.Index(content[patternIdx:], "\n")
	if newlineIdx == -1 {
		return content
	}
	insertPoint := patternIdx + newlineIdx + 1

	// Look at the next line to check for placeholder
	restOfContent := content[insertPoint:]
	nextNewline := strings.Index(restOfContent, "\n")
	var nextLine string
	if nextNewline != -1 {
		nextLine = restOfContent[:nextNewline]
	} else {
		nextLine = restOfContent
	}

	// Build new line
	newLine := fmt.Sprintf("- %s\n", evidence)

	// If next line is a placeholder, replace it; otherwise insert
	if strings.Contains(nextLine, "[Pattern 1:") || strings.Contains(nextLine, "[Pattern 2:") {
		if nextNewline != -1 {
			return content[:insertPoint] + newLine + content[insertPoint+nextNewline+1:]
		}
		return content[:insertPoint] + newLine
	}

	// Insert after header
	return content[:insertPoint] + newLine + content[insertPoint:]
}

// updateKnowledgeSection adds a decision/constraint to the Knowledge section.
func updateKnowledgeSection(content, knowledge string) string {
	if knowledge == "" {
		return content
	}

	// Find "## Knowledge" section
	knowledgeMarker := "## Knowledge"
	idx := strings.Index(content, knowledgeMarker)
	if idx == -1 {
		return content
	}

	// Find "### Decisions Made" subsection
	decisionMarker := "### Decisions Made"
	decisionIdx := strings.Index(content[idx:], decisionMarker)
	if decisionIdx == -1 {
		return content
	}
	decisionIdx += idx

	// Find end of subsection header line
	newlineIdx := strings.Index(content[decisionIdx:], "\n")
	if newlineIdx == -1 {
		return content
	}
	insertPoint := decisionIdx + newlineIdx + 1

	// Look at the next line to check for placeholder
	restOfContent := content[insertPoint:]
	nextNewline := strings.Index(restOfContent, "\n")
	var nextLine string
	if nextNewline != -1 {
		nextLine = restOfContent[:nextNewline]
	} else {
		nextLine = restOfContent
	}

	// Build new line
	newLine := fmt.Sprintf("- %s\n", knowledge)

	// If next line is a placeholder, replace it; otherwise insert
	if strings.Contains(nextLine, "{topic}:") || strings.Contains(nextLine, "{decision}") {
		if nextNewline != -1 {
			return content[:insertPoint] + newLine + content[insertPoint+nextNewline+1:]
		}
		return content[:insertPoint] + newLine
	}

	// Insert after header
	return content[:insertPoint] + newLine + content[insertPoint:]
}

// UpdateHandoffAfterComplete prompts for handoff updates after an agent completes.
// This is called from orch complete to implement the "Capture at Context" principle.
// The handoff is updated with:
// - Spawns table row (outcome, key finding)
// - Evidence section (optional pattern observation)
// - Knowledge section (optional decision/constraint)
//
// Parameters:
//   - projectDir: The project directory where .orch/session/ lives
//   - workspaceName: The agent workspace name
//   - beadsID: The beads issue ID
//   - skill: The skill used for the agent
//
// Returns nil if no active session handoff exists (not an error - session may not be active).
func UpdateHandoffAfterComplete(projectDir, workspaceName, beadsID, skill string) error {
	// Find active session handoff
	handoffPath := findActiveSessionHandoff(projectDir)
	if handoffPath == "" {
		// No active session - nothing to update
		return nil
	}

	// Read current handoff content
	content, err := os.ReadFile(handoffPath)
	if err != nil {
		return fmt.Errorf("failed to read handoff: %w", err)
	}
	contentStr := string(content)

	// Prompt for spawn completion info
	info, err := promptSpawnCompletion(workspaceName, beadsID, skill)
	if err != nil {
		return err
	}
	if info == nil {
		// User skipped
		return nil
	}

	// Update spawns table
	contentStr = updateSpawnsTable(contentStr, info)

	// Prompt for evidence and knowledge
	evidence, knowledge, err := promptEvidenceAndKnowledge()
	if err != nil {
		return err
	}

	// Update evidence section
	contentStr = updateEvidenceSection(contentStr, evidence)

	// Update knowledge section
	contentStr = updateKnowledgeSection(contentStr, knowledge)

	// Write updated content
	if err := os.WriteFile(handoffPath, []byte(contentStr), 0644); err != nil {
		return fmt.Errorf("failed to write updated handoff: %w", err)
	}

	fmt.Printf("\n✅ Updated session handoff: %s\n", filepath.Base(filepath.Dir(filepath.Dir(handoffPath))))

	return nil
}
