// Package main provides agent discovery, enrichment, and filtering for the status command.
// Extracted from status_cmd.go as part of the status_cmd.go refactoring.
package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/session"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// isSessionLikelyProcessing checks if a session might be processing based on its last update time.
// Only makes the expensive IsSessionProcessing HTTP call for recently updated sessions.
// For sessions not updated recently, assumes they are idle (saves ~100ms per call).
func isSessionLikelyProcessing(client *opencode.Client, sessionID string, lastUpdated time.Time, now time.Time) bool {
	// If the session hasn't been updated in the last 5 minutes, it's definitely not processing
	if now.Sub(lastUpdated) > processingCheckMaxAge {
		return false
	}
	// For recently active sessions, make the HTTP call to check processing status
	return client.IsSessionProcessing(sessionID)
}

// getPhaseAndTask retrieves the current phase and task description from beads.
func getPhaseAndTask(beadsID string) (phase, task string) {
	// Get issue for task description
	issue, err := verify.GetIssue(beadsID)
	if err == nil {
		task = truncate(issue.Title, 40)
	}

	// Get phase from comments
	status, err := verify.GetPhaseStatus(beadsID)
	if err == nil && status.Found {
		phase = status.Phase
	}

	return phase, task
}

// computeIsPhantom determines whether an agent should be classified as phantom.
// Phantom means: there is a real, non-closed beads issue, but there is no active runtime
// signal (no OpenCode session and no tmux window).
//
// IMPORTANT: --no-track spawns (project-untracked-*) intentionally have no beads issue,
// so they are never phantom.
func computeIsPhantom(agent AgentInfo, issue *verify.Issue, issueExists bool) bool {
	// Untracked sessions (no beads tracking) are never phantom.
	if agent.IsUntracked || (agent.BeadsID != "" && isUntrackedBeadsID(agent.BeadsID)) {
		return false
	}

	// Any runtime signal means the agent isn't phantom.
	if agent.SessionID != "" && agent.SessionID != "tmux-stalled" {
		return false
	}
	if agent.Window != "" {
		return false
	}

	// Must correspond to a real, open beads issue.
	if !issueExists || issue == nil {
		return false
	}
	if strings.EqualFold(issue.Status, "closed") {
		return false
	}

	return true
}

// computeSwarmStatus builds swarm aggregate counts.
// Counts are computed on the full discovered agent list (before display filtering).
func computeSwarmStatus(agents []AgentInfo) SwarmStatus {
	activeCount := 0
	processingCount := 0
	idleCount := 0
	phantomCount := 0
	completedCount := 0
	untrackedCount := 0

	for _, agent := range agents {
		if agent.IsUntracked {
			untrackedCount++
			if agent.IsProcessing {
				processingCount++
			}
			continue
		}

		// Closed beads issues are completed, even if other fields are inconsistent.
		if agent.IsCompleted {
			completedCount++
			continue
		}

		if agent.IsPhantom {
			phantomCount++
			continue
		}

		activeCount++
		if agent.IsProcessing {
			processingCount++
		} else {
			idleCount++
		}
	}

	return SwarmStatus{
		Active:     activeCount,
		Processing: processingCount,
		Idle:       idleCount,
		Phantom:    phantomCount,
		Untracked:  untrackedCount,
		Queued:     0,              // TODO: implement queuing system
		Completed:  completedCount, // Agents with closed beads issues
	}
}

// determineAgentSource returns the primary source indicator for an agent.
// Priority: T (tmux) > O (OpenCode) > B (beads phantom) > W (workspace).
// Returns: T=tmux, O=opencode, B=beads phantom, W=workspace, or empty string if unknown.
func determineAgentSource(agent AgentInfo, projectDir string) string {
	// Tmux has highest priority (visible TUI)
	if agent.Window != "" {
		return "T"
	}

	// OpenCode session (headless or API mode)
	if agent.SessionID != "" && agent.SessionID != "tmux-stalled" && agent.SessionID != "api-stalled" {
		return "O"
	}

	// Beads phantom (issue exists but no active runtime)
	if agent.BeadsID != "" && agent.IsPhantom {
		return "B"
	}

	// Workspace (has workspace directory)
	if agent.BeadsID != "" && projectDir != "" {
		workspacePath, _ := findWorkspaceByBeadsID(projectDir, agent.BeadsID)
		if workspacePath != "" {
			return "W"
		}
	}

	return ""
}

// getOrchestratorSessions fetches active orchestrator sessions from the registry.
// If project is non-empty, filters to only sessions in that project.
func getOrchestratorSessions(project string) []OrchestratorSessionInfo {
	registry := session.NewRegistry("")
	sessions, err := registry.ListActive()
	if err != nil {
		return nil // Silently fail - registry may not exist yet
	}

	now := time.Now()
	var result []OrchestratorSessionInfo

	for _, s := range sessions {
		// Extract project name from directory path
		projectName := filepath.Base(s.ProjectDir)

		// Filter by project if specified
		if project != "" && projectName != project {
			continue
		}

		duration := formatDuration(now.Sub(s.SpawnTime))

		result = append(result, OrchestratorSessionInfo{
			WorkspaceName: s.WorkspaceName,
			Goal:          s.Goal,
			Duration:      duration,
			Project:       projectName,
			Status:        s.Status,
		})
	}

	return result
}

// AgentManifest represents metadata from AGENT_MANIFEST.json.
type AgentManifest struct {
	WorkspaceName string `json:"workspace_name"`
	Skill         string `json:"skill"`
	BeadsID       string `json:"beads_id"`
	ProjectDir    string `json:"project_dir"`
	SpawnMode     string `json:"spawn_mode"`
	Tier          string `json:"tier"`
	Model         string `json:"model,omitempty"`
}

// readAgentManifest reads AGENT_MANIFEST.json from a workspace directory.
// Returns nil if the file doesn't exist or can't be parsed.
func readAgentManifest(workspacePath string) *AgentManifest {
	manifestPath := filepath.Join(workspacePath, "AGENT_MANIFEST.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil
	}

	var manifest AgentManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil
	}

	return &manifest
}

// getBeadsIssuePrefix reads the issue_prefix for a project using bd CLI.
// Returns empty string if the command fails or project doesn't have beads.
func getBeadsIssuePrefix(projectPath string) string {
	cmd := exec.Command("bd", "config", "get", "issue_prefix")
	cmd.Dir = projectPath
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Output is just the value (e.g., "pw\n")
	return strings.TrimSpace(string(output))
}

// getKBProjectsWithNames fetches registered projects from kb with name and path.
// Returns empty slice if kb is unavailable or fails (graceful degradation).
func getKBProjectsWithNames() []kbProject {
	cmd := exec.Command("kb", "projects", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return []kbProject{}
	}

	var projects []kbProject
	if err := json.Unmarshal(output, &projects); err != nil {
		return []kbProject{}
	}

	return projects
}

// findProjectByBeadsPrefix searches for a project with the given beads issue prefix.
// First checks kb's project registry, then falls back to standard locations.
// Returns the project directory path, or empty string if not found.
func findProjectByBeadsPrefix(prefix string) string {
	// Try kb's project registry first
	for _, project := range getKBProjectsWithNames() {
		if projectPrefix := getBeadsIssuePrefix(project.Path); projectPrefix == prefix {
			return project.Path
		}
	}

	// Fall back to checking standard locations
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	candidatePaths := []string{
		filepath.Join(homeDir, "Documents", "personal", prefix),
		filepath.Join(homeDir, prefix),
		filepath.Join(homeDir, "projects", prefix),
		filepath.Join(homeDir, "src", prefix),
	}

	for _, path := range candidatePaths {
		if projectPrefix := getBeadsIssuePrefix(path); projectPrefix == prefix {
			return path
		}
	}

	return ""
}

// findProjectDirByName looks up a project directory by its name or beads prefix.
// First checks kb's project registry, then searches common project locations.
// Verifies the project has a .beads/ directory.
// Returns empty string if not found.
func findProjectDirByName(projectName string) string {
	// Try kb's project registry first (handles non-standard locations)
	for _, project := range getKBProjectsWithNames() {
		if project.Name == projectName {
			// Verify it has a .beads directory
			beadsPath := filepath.Join(project.Path, ".beads")
			if info, err := os.Stat(beadsPath); err == nil && info.IsDir() {
				return project.Path
			}
		}
	}

	// If projectName looks like a beads prefix (short, no hyphens except separators),
	// try finding by prefix instead
	if len(projectName) <= 10 && !strings.Contains(projectName, "/") {
		if path := findProjectByBeadsPrefix(projectName); path != "" {
			return path
		}
	}

	// Fall back to checking standard locations
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Common project locations in order of priority
	candidatePaths := []string{
		filepath.Join(homeDir, "Documents", "personal", projectName),
		filepath.Join(homeDir, projectName),
		filepath.Join(homeDir, "projects", projectName),
		filepath.Join(homeDir, "src", projectName),
	}

	for _, path := range candidatePaths {
		// Check if directory exists and has .beads/ (confirms it's a beads-tracked project)
		beadsPath := filepath.Join(path, ".beads")
		if info, err := os.Stat(beadsPath); err == nil && info.IsDir() {
			return path
		}
	}

	return ""
}

// extractDateFromWorkspaceName parses the date suffix from a workspace name.
// Workspace names follow format: prefix-description-DDmon (e.g., og-feat-add-feature-24dec)
// Returns zero time if no valid date found.
func extractDateFromWorkspaceName(name string) time.Time {
	// Month abbreviations (lowercase)
	months := map[string]time.Month{
		"jan": time.January,
		"feb": time.February,
		"mar": time.March,
		"apr": time.April,
		"may": time.May,
		"jun": time.June,
		"jul": time.July,
		"aug": time.August,
		"sep": time.September,
		"oct": time.October,
		"nov": time.November,
		"dec": time.December,
	}

	// Get the last segment after the final hyphen
	parts := strings.Split(name, "-")
	if len(parts) == 0 {
		return time.Time{}
	}
	lastPart := strings.ToLower(parts[len(parts)-1])

	// Pattern: 1-2 digits followed by 3-letter month abbreviation (e.g., "24dec", "5jan")
	if len(lastPart) < 4 || len(lastPart) > 5 {
		return time.Time{}
	}

	// Extract the month abbreviation (last 3 chars)
	monthStr := lastPart[len(lastPart)-3:]
	month, ok := months[monthStr]
	if !ok {
		return time.Time{}
	}

	// Extract the day (remaining digits)
	dayStr := lastPart[:len(lastPart)-3]
	day, err := strconv.Atoi(dayStr)
	if err != nil || day < 1 || day > 31 {
		return time.Time{}
	}

	// Use current year, adjusting for year boundary
	// (if the date is in the future within this calendar, it's probably from last year)
	now := time.Now()
	year := now.Year()
	parsedDate := time.Date(year, month, day, 12, 0, 0, 0, time.Local)

	// If the parsed date is more than a week in the future, assume it's from last year
	if parsedDate.After(now.AddDate(0, 0, 7)) {
		parsedDate = time.Date(year-1, month, day, 12, 0, 0, 0, time.Local)
	}

	return parsedDate
}
