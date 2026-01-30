// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// DefaultActiveCount returns the number of active agents by querying OpenCode API.
// Counts only recently-active sessions (updated within the last 30 minutes) to avoid
// counting stale sessions that persist indefinitely in OpenCode.
// Excludes untracked agents (spawned with --no-track) which have "-untracked-" in their beads ID.
// Excludes sessions whose beads issues are already closed (completed agents).
func DefaultActiveCount() int {
	// Use OpenCode API to count active sessions
	// The default server URL is used; this works because the daemon runs
	// on the same machine as OpenCode server.
	serverURL := os.Getenv("OPENCODE_URL")
	if serverURL == "" {
		serverURL = "http://127.0.0.1:4096"
	}

	// Make HTTP request to list sessions
	resp, err := http.Get(serverURL + "/session")
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	var sessions []struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		Directory string `json:"directory"` // Session's working directory (for cross-project resolution)
		Time      struct {
			Updated int64 `json:"updated"` // Unix timestamp in milliseconds
		} `json:"time"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return 0
	}

	// Only count sessions that have been active recently.
	// OpenCode sessions persist indefinitely (including old test sessions),
	// so we filter to sessions updated within the last 30 minutes.
	// This matches the same threshold used in orch status for agent matching.
	const maxIdleTime = 30 * time.Minute
	now := time.Now()

	// Collect beads IDs and project directories for batch lookup.
	// Using session.Directory is more reliable than kb projects list because:
	// 1. It directly identifies where the agent is running
	// 2. It doesn't require the project to be registered in kb
	var recentBeadsIDs []string
	beadsIDToSession := make(map[string]bool)
	beadsIDToProjectDir := make(map[string]string)
	for _, s := range sessions {
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		if now.Sub(updatedAt) > maxIdleTime {
			continue
		}

		// Extract beads ID from title (format: "workspace-name [beads-id]")
		// Skip untracked agents which have "-untracked-" in their beads ID.
		// These are ad-hoc spawns that shouldn't count against daemon capacity.
		beadsID := extractBeadsIDFromSessionTitle(s.Title)
		if beadsID == "" || isUntrackedBeadsID(beadsID) {
			continue
		}

		recentBeadsIDs = append(recentBeadsIDs, beadsID)
		beadsIDToSession[beadsID] = true

		// Use session directory for cross-project resolution.
		// Skip "/" as it's not a valid project directory.
		if s.Directory != "" && s.Directory != "/" {
			beadsIDToProjectDir[beadsID] = s.Directory
		}
	}

	// If no recent sessions, return early
	if len(recentBeadsIDs) == 0 {
		return 0
	}

	// Batch fetch issue status to check if closed.
	// Use project dirs from sessions (more reliable for cross-project scenarios).
	closedIssues := GetClosedIssuesBatchWithProjectDirs(recentBeadsIDs, beadsIDToProjectDir)

	// Count sessions with open issues only
	activeCount := 0
	for beadsID := range beadsIDToSession {
		if closedIssues[beadsID] {
			// Issue is closed, don't count this session
			continue
		}
		activeCount++
	}

	return activeCount
}

// GetClosedIssuesBatch checks which beads IDs have closed issues.
// Returns a map of beadsID -> true for closed issues.
// Supports cross-project lookups by parsing project prefix from beads IDs
// and querying the correct beads database for each project.
// Exported for use by checkConcurrencyLimit in spawn_cmd.go.
func GetClosedIssuesBatch(beadsIDs []string) map[string]bool {
	// Delegate to the version with empty project dirs.
	// This will fall back to kb projects list for project resolution.
	return GetClosedIssuesBatchWithProjectDirs(beadsIDs, nil)
}

// GetClosedIssuesBatchWithProjectDirs checks which beads IDs have closed issues.
// Returns a map of beadsID -> true for closed issues.
// The projectDirs map provides explicit beadsID -> projectDir mappings (typically from session.Directory).
// For beads IDs not in projectDirs, falls back to kb projects list resolution.
// This is the preferred method when session directories are available, as it's more reliable
// than deriving project paths from beads ID prefixes.
func GetClosedIssuesBatchWithProjectDirs(beadsIDs []string, projectDirs map[string]string) map[string]bool {
	closed := make(map[string]bool)
	if len(beadsIDs) == 0 {
		return closed
	}

	// Build project name -> path map for cross-project resolution (used as fallback)
	projectNamePaths := buildProjectPathMap()

	// Group beads IDs by project directory.
	// Priority: 1) explicit projectDirs, 2) kb projects list, 3) current directory
	idsByProjectDir := make(map[string][]string)
	for _, id := range beadsIDs {
		var projectDir string

		// First, check if we have an explicit project dir from the caller
		if projectDirs != nil {
			if dir, ok := projectDirs[id]; ok && dir != "" {
				projectDir = dir
			}
		}

		// Fall back to kb projects list resolution by beads ID prefix
		if projectDir == "" {
			projectName := extractProjectFromBeadsID(id)
			if path, ok := projectNamePaths[projectName]; ok {
				projectDir = path
			}
		}

		// Group by project dir (empty string = current directory)
		idsByProjectDir[projectDir] = append(idsByProjectDir[projectDir], id)
	}

	// Check each project's issues
	for projectDir, ids := range idsByProjectDir {
		closedInProject := getClosedIssuesForProject(projectDir, ids)
		for id := range closedInProject {
			closed[id] = true
		}
	}

	return closed
}

// buildProjectPathMap returns a map of project name -> project path
// by querying kb projects list.
func buildProjectPathMap() map[string]string {
	projects, _ := ListProjects()
	pathMap := make(map[string]string, len(projects))
	for _, p := range projects {
		pathMap[p.Name] = p.Path
	}
	return pathMap
}

// extractProjectFromBeadsID extracts the project name from a beads ID.
// Beads IDs have the format "{project-name}-{hash}" like "orch-go-abc1".
// Returns empty string if format is not recognized.
func extractProjectFromBeadsID(beadsID string) string {
	// Find the last dash followed by alphanumeric hash (usually 4-5 chars)
	// The project name is everything before that
	lastDash := strings.LastIndex(beadsID, "-")
	if lastDash <= 0 {
		return ""
	}
	return beadsID[:lastDash]
}

// groupBeadsIDsByProject groups beads IDs by their project path.
// IDs without a known project path are grouped under "" (empty string)
// which means they'll use the current directory.
func groupBeadsIDsByProject(beadsIDs []string, projectPaths map[string]string) map[string][]string {
	grouped := make(map[string][]string)
	for _, id := range beadsIDs {
		projectName := extractProjectFromBeadsID(id)
		projectPath := projectPaths[projectName]
		// If not found in projects, projectPath will be "" which uses current dir
		grouped[projectPath] = append(grouped[projectPath], id)
	}
	return grouped
}

// getClosedIssuesForProject checks which beads IDs are closed for a specific project.
// If projectPath is empty, uses the current directory.
//
// IMPORTANT: Lookup failures are treated as "closed" to prevent capacity leaks.
// If we can't confirm a session is active, we don't count it toward capacity.
// This is conservative: better to potentially allow extra spawns than to get
// permanently stuck at capacity (which requires manual daemon restart).
//
// Note: "Issue not found" errors are expected when cross-project sessions exist
// (e.g., specs-platform-36 when running from orch-go). These are silently treated
// as closed without logging warnings, since they're expected behavior not errors.
func getClosedIssuesForProject(projectPath string, beadsIDs []string) map[string]bool {
	closed := make(map[string]bool)
	if len(beadsIDs) == 0 {
		return closed
	}

	// Try beads RPC client first
	socketPath, err := beads.FindSocketPath(projectPath)
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(2))
		if err := client.Connect(); err == nil {
			defer client.Close()
			// Check each issue status
			for _, id := range beadsIDs {
				issue, err := client.Show(id)
				if err != nil {
					// Treat lookup failures as "closed" to prevent capacity leaks.
					// If we can't confirm a session is active (open beads issue),
					// we shouldn't count it toward capacity. This is conservative:
					// better to potentially spawn extra agents than get permanently
					// stuck at capacity. Common failure cases:
					// - Issue was deleted
					// - RPC connection timeout
					// - Wrong project directory
					// - Beads daemon not running
					//
					// Only log warning for unexpected errors, not "issue not found"
					// which is expected for cross-project sessions.
					if !errors.Is(err, beads.ErrIssueNotFound) {
						log.Printf("Warning: beads lookup failed for %s (via RPC): %v - treating as closed", id, err)
					}
					closed[id] = true
					continue
				}
				if strings.EqualFold(issue.Status, "closed") {
					closed[id] = true
				}
			}
			return closed
		}
	}

	// Fallback to CLI for each issue, setting the working directory
	for _, id := range beadsIDs {
		issue, err := beads.FallbackShowWithDir(id, projectPath)
		if err != nil {
			// Treat lookup failures as "closed" to prevent capacity leaks.
			// Same rationale as RPC path above.
			//
			// Only log warning for unexpected errors, not "issue not found"
			// which is expected for cross-project sessions.
			if !errors.Is(err, beads.ErrIssueNotFound) {
				log.Printf("Warning: beads lookup failed for %s (via CLI): %v - treating as closed", id, err)
			}
			closed[id] = true
			continue
		}
		if strings.EqualFold(issue.Status, "closed") {
			closed[id] = true
		}
	}

	return closed
}

// extractBeadsIDFromSessionTitle extracts beads ID from an OpenCode session title.
// Session titles follow format: "workspace-name [beads-id]" (e.g., "og-feat-add-feature-24dec [orch-go-3anf]")
func extractBeadsIDFromSessionTitle(title string) string {
	// Look for "[beads-id]" pattern at the end
	if start := strings.LastIndex(title, "["); start != -1 {
		if end := strings.LastIndex(title, "]"); end != -1 && end > start {
			return strings.TrimSpace(title[start+1 : end])
		}
	}
	return ""
}

// isUntrackedBeadsID returns true if the beads ID indicates an untracked agent.
// Untracked agents are spawned with --no-track and have IDs like "project-untracked-1766695797".
func isUntrackedBeadsID(beadsID string) bool {
	return strings.Contains(beadsID, "-untracked-")
}

// DockerActiveCount returns the number of active agents by counting Docker containers.
// Counts running containers using the claude-code-mcp image.
// This is used when backend is "docker" since Docker spawns don't register with OpenCode.
func DockerActiveCount() int {
	// Run docker ps to count running claude-code-mcp containers
	cmd := exec.Command("docker", "ps", "--filter", "ancestor=claude-code-mcp", "--format", "{{.ID}}")
	output, err := cmd.Output()
	if err != nil {
		// Docker not available or error - return 0
		return 0
	}

	// Count non-empty lines
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0
	}
	return len(lines)
}
