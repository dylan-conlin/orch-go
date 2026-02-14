// Package main provides the CLI entry point for orch-go.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// ============================================================================
// Session Handoff Creation, Validation, and Interactive Completion
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

// UserResponse holds a user's response to a section prompt.
type UserResponse struct {
	Section  HandoffSection
	Response string
}

// SpawnCompletionInfo holds information about a completed spawn for handoff updates.
type SpawnCompletionInfo struct {
	WorkspaceName string // e.g., "og-feat-auth-middleware-14jan-a1b2"
	BeadsID       string // e.g., "orch-go-abc1"
	Skill         string // e.g., "feature-impl"
	Outcome       string // success | partial | failed
	KeyFinding    string // 1-line insight from the completion
}

// createActiveSessionHandoff creates SESSION_HANDOFF.md in {project}/.orch/session/{sessionName}/active/.
// Returns the full path to the created handoff file.
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

		// Check for required empty fields
		if section.Required && response == "" {
			return nil, fmt.Errorf("%s is required and cannot be empty", section.Name)
		}

		// Validate options if provided
		if len(section.Options) > 0 && response != "" {
			valid := false
			for _, opt := range section.Options {
				if response == opt {
					valid = true
					break
				}
			}
			if !valid {
				return nil, fmt.Errorf("invalid option: %q (must be one of: %s)", response, strings.Join(section.Options, ", "))
			}
		}

		responses = append(responses, UserResponse{
			Section:  section,
			Response: response,
		})
		fmt.Println()
	}

	return responses, nil
}

// promptForStartSections prompts for session start sections (TLDR, Where We Started).
// Part of Progressive Session Capture - capturing context when it's freshest.
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
