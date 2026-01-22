// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
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
		ID    string `json:"id"`
		Title string `json:"title"`
		Time  struct {
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

	// Collect beads IDs for batch lookup
	var recentBeadsIDs []string
	beadsIDToSession := make(map[string]bool)
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
	}

	// If no recent sessions, return early
	if len(recentBeadsIDs) == 0 {
		return 0
	}

	// Batch fetch issue status to check if closed
	// This prevents counting completed agents (beads issue closed but session still exists)
	closedIssues := GetClosedIssuesBatch(recentBeadsIDs)

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
	closed := make(map[string]bool)
	if len(beadsIDs) == 0 {
		return closed
	}

	// Build project name -> path map for cross-project resolution
	projectPaths := buildProjectPathMap()

	// Group beads IDs by project
	idsByProject := groupBeadsIDsByProject(beadsIDs, projectPaths)

	// Check each project's issues
	for projectPath, ids := range idsByProject {
		closedInProject := getClosedIssuesForProject(projectPath, ids)
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
					// If we can't find the issue, assume it's not running
					// (might have been deleted or never existed)
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
