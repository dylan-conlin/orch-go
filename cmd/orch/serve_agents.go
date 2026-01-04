package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/port"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// beadsCache provides TTL-based caching for beads data to prevent excessive
// bd process spawning when the dashboard polls /api/agents frequently.
// Without caching, each request can spawn 20+ concurrent bd processes for 600+ workspaces.
type beadsCache struct {
	mu sync.RWMutex

	// Cached data
	openIssues map[string]*verify.Issue
	allIssues  map[string]*verify.Issue
	comments   map[string][]beads.Comment

	// Cache metadata
	openIssuesFetchedAt time.Time
	allIssuesFetchedAt  time.Time
	allIssuesFetchedFor []string // Track which beads IDs were fetched
	commentsFetchedAt   time.Time
	commentsFetchedFor  []string // Track which beads IDs were fetched

	// TTL configuration
	openIssuesTTL time.Duration
	allIssuesTTL  time.Duration
	commentsTTL   time.Duration
}

// globalWorkspaceCache provides TTL-based caching for workspace metadata.
// Without caching, each /api/agents request scans 600+ SPAWN_CONTEXT.md files.
type globalWorkspaceCacheType struct {
	mu sync.RWMutex

	// Cached data
	cache *workspaceCache

	// Cache metadata
	fetchedAt time.Time
	ttl       time.Duration
}

// Global workspace cache
var globalWorkspaceCacheInstance = &globalWorkspaceCacheType{
	ttl: 30 * time.Second, // Workspace metadata changes infrequently
}

// getCachedWorkspace returns cached workspace data or builds fresh if stale.
func (c *globalWorkspaceCacheType) getCachedWorkspace(projectDirs []string) *workspaceCache {
	c.mu.RLock()
	if c.cache != nil && time.Since(c.fetchedAt) < c.ttl {
		result := c.cache
		c.mu.RUnlock()
		return result
	}
	c.mu.RUnlock()

	// Build fresh workspace cache
	wsCache := buildMultiProjectWorkspaceCache(projectDirs)

	c.mu.Lock()
	c.cache = wsCache
	c.fetchedAt = time.Now()
	c.mu.Unlock()

	return wsCache
}

// Default TTLs for cached data
const (
	defaultOpenIssuesTTL = 10 * time.Second // Open issues change infrequently
	defaultAllIssuesTTL  = 30 * time.Second // Closed issues change even less
	defaultCommentsTTL   = 5 * time.Second  // Comments change more often (phase updates)
)

// invalidate clears all cached data, forcing fresh fetches on next request.
// This is called when agents complete to ensure the dashboard shows current status.
func (c *beadsCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Reset all cached data
	c.openIssues = make(map[string]*verify.Issue)
	c.allIssues = make(map[string]*verify.Issue)
	c.comments = make(map[string][]beads.Comment)

	// Reset timestamps to force refetch
	c.openIssuesFetchedAt = time.Time{}
	c.allIssuesFetchedAt = time.Time{}
	c.commentsFetchedAt = time.Time{}
	c.allIssuesFetchedFor = nil
	c.commentsFetchedFor = nil
}

// invalidate clears the cached workspace data, forcing a fresh scan on next request.
// This is called when agents complete to ensure the dashboard shows current status.
func (c *globalWorkspaceCacheType) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = nil
	c.fetchedAt = time.Time{}
}

// Global beads cache instance, initialized in runServe
var globalBeadsCache *beadsCache

// newBeadsCache creates a new beads cache with default TTLs.
func newBeadsCache() *beadsCache {
	return &beadsCache{
		openIssues:    make(map[string]*verify.Issue),
		allIssues:     make(map[string]*verify.Issue),
		comments:      make(map[string][]beads.Comment),
		openIssuesTTL: defaultOpenIssuesTTL,
		allIssuesTTL:  defaultAllIssuesTTL,
		commentsTTL:   defaultCommentsTTL,
	}
}

// getOpenIssues returns cached open issues or fetches fresh data if cache is stale.
func (c *beadsCache) getOpenIssues() (map[string]*verify.Issue, error) {
	c.mu.RLock()
	if time.Since(c.openIssuesFetchedAt) < c.openIssuesTTL && len(c.openIssues) > 0 {
		result := c.openIssues
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	// Fetch fresh data
	issues, err := verify.ListOpenIssues()
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.openIssues = issues
	c.openIssuesFetchedAt = time.Now()
	c.mu.Unlock()

	return issues, nil
}

// getAllIssues returns cached issues or fetches fresh data if cache is stale.
// The beadsIDs parameter specifies which issues to fetch. If the cached set
// matches the requested set and is not expired, returns cached data.
func (c *beadsCache) getAllIssues(beadsIDs []string) (map[string]*verify.Issue, error) {
	c.mu.RLock()
	if time.Since(c.allIssuesFetchedAt) < c.allIssuesTTL && c.containsAllIDs(c.allIssuesFetchedFor, beadsIDs) {
		result := c.allIssues
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	// Fetch fresh data
	issues, err := verify.GetIssuesBatch(beadsIDs)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.allIssues = issues
	c.allIssuesFetchedAt = time.Now()
	c.allIssuesFetchedFor = beadsIDs
	c.mu.Unlock()

	return issues, nil
}

// getComments returns cached comments or fetches fresh data if cache is stale.
// The beadsIDs and projectDirs parameters specify which comments to fetch.
func (c *beadsCache) getComments(beadsIDs []string, projectDirs map[string]string) map[string][]beads.Comment {
	c.mu.RLock()
	if time.Since(c.commentsFetchedAt) < c.commentsTTL && c.containsAllIDs(c.commentsFetchedFor, beadsIDs) {
		result := c.comments
		c.mu.RUnlock()
		return result
	}
	c.mu.RUnlock()

	// Fetch fresh data
	comments := verify.GetCommentsBatchWithProjectDirs(beadsIDs, projectDirs)

	c.mu.Lock()
	c.comments = comments
	c.commentsFetchedAt = time.Now()
	c.commentsFetchedFor = beadsIDs
	c.mu.Unlock()

	return comments
}

// containsAllIDs checks if cachedIDs contains all requestedIDs.
func (c *beadsCache) containsAllIDs(cachedIDs, requestedIDs []string) bool {
	if len(cachedIDs) == 0 {
		return false
	}
	cachedSet := make(map[string]bool, len(cachedIDs))
	for _, id := range cachedIDs {
		cachedSet[id] = true
	}
	for _, id := range requestedIDs {
		if !cachedSet[id] {
			return false
		}
	}
	return true
}

// AgentAPIResponse is the JSON structure returned by /api/agents.
type AgentAPIResponse struct {
	ID           string               `json:"id"`
	SessionID    string               `json:"session_id,omitempty"`
	BeadsID      string               `json:"beads_id,omitempty"`
	BeadsTitle   string               `json:"beads_title,omitempty"`
	Skill        string               `json:"skill,omitempty"`
	Status       string               `json:"status"`            // "active", "idle", "completed", etc.
	Phase        string               `json:"phase,omitempty"`   // "Planning", "Implementing", "Complete", etc.
	Task         string               `json:"task,omitempty"`    // Task description from beads issue
	Project      string               `json:"project,omitempty"` // Project name (orch-go, skillc, etc.)
	Runtime      string               `json:"runtime,omitempty"`
	Window       string               `json:"window,omitempty"`
	IsProcessing bool                 `json:"is_processing,omitempty"` // True if actively generating response
	SpawnedAt    string               `json:"spawned_at,omitempty"`    // ISO 8601 timestamp
	UpdatedAt    string               `json:"updated_at,omitempty"`    // ISO 8601 timestamp
	Synthesis    *SynthesisResponse   `json:"synthesis,omitempty"`
	CloseReason  string               `json:"close_reason,omitempty"` // Beads close reason, fallback when synthesis is null
	GapAnalysis  *GapAPIResponse      `json:"gap_analysis,omitempty"` // Context gap analysis from spawn time
	Tokens       *opencode.TokenStats `json:"tokens,omitempty"`       // Token usage for the session
}

// GapAPIResponse represents gap analysis data for the API.
type GapAPIResponse struct {
	HasGaps        bool `json:"has_gaps"`
	ContextQuality int  `json:"context_quality"`
	ShouldWarn     bool `json:"should_warn"`
	MatchCount     int  `json:"match_count,omitempty"`
	Constraints    int  `json:"constraints,omitempty"`
	Decisions      int  `json:"decisions,omitempty"`
	Investigations int  `json:"investigations,omitempty"`
}

// SynthesisResponse is a condensed version of verify.Synthesis for the API.
// Uses the D.E.K.N. structure: Delta, Evidence, Knowledge, Next.
type SynthesisResponse struct {
	// Header fields
	TLDR           string `json:"tldr,omitempty"`
	Outcome        string `json:"outcome,omitempty"`        // success, partial, blocked, failed
	Recommendation string `json:"recommendation,omitempty"` // close, continue, escalate

	// Condensed sections
	DeltaSummary string   `json:"delta_summary,omitempty"` // e.g., "3 files created, 2 modified, 5 commits"
	NextActions  []string `json:"next_actions,omitempty"`  // Follow-up items
}

// workspaceCache stores pre-computed workspace metadata to avoid repeated directory scans.
// Built once per request and used for all lookups within that request.
type workspaceCache struct {
	// beadsToWorkspace maps beadsID → workspace path (absolute)
	beadsToWorkspace map[string]string
	// beadsToProjectDir maps beadsID → PROJECT_DIR from SPAWN_CONTEXT.md
	beadsToProjectDir map[string]string
	// workspaceEntries stores directory entries for reuse
	workspaceEntries []os.DirEntry
	// workspaceDir is the base workspace directory path
	workspaceDir string
	// workspaceEntryToPath maps directory entry name → absolute workspace path
	// This is needed for multi-project scenarios where entries come from different projects
	workspaceEntryToPath map[string]string
}

// extractUniqueProjectDirs collects unique project directories from OpenCode sessions.
// Returns a deduplicated slice of directory paths that have active agents.
// This enables multi-project workspace aggregation for cross-project agent visibility.
func extractUniqueProjectDirs(sessions []opencode.Session, currentProjectDir string) []string {
	seen := make(map[string]bool)
	var dirs []string

	// Always include current project directory
	if currentProjectDir != "" {
		seen[currentProjectDir] = true
		dirs = append(dirs, currentProjectDir)
	}

	// Add unique directories from sessions
	for _, s := range sessions {
		dir := s.Directory
		if dir == "" {
			continue
		}

		// Normalize path (resolve any symlinks, clean path)
		dir = filepath.Clean(dir)

		if !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}

	return dirs
}

// buildMultiProjectWorkspaceCache builds workspace caches for multiple project directories
// and merges them into a unified cache. Scans in parallel for performance.
// This enables cross-project agent visibility by aggregating workspace metadata
// from all projects with active OpenCode sessions.
func buildMultiProjectWorkspaceCache(projectDirs []string) *workspaceCache {
	if len(projectDirs) == 0 {
		return &workspaceCache{
			beadsToWorkspace:  make(map[string]string),
			beadsToProjectDir: make(map[string]string),
		}
	}

	// If only one project directory, use the simpler single-project scan
	if len(projectDirs) == 1 {
		return buildWorkspaceCache(projectDirs[0])
	}

	// Build caches in parallel using goroutines
	type cacheResult struct {
		cache *workspaceCache
	}
	results := make(chan cacheResult, len(projectDirs))

	for _, dir := range projectDirs {
		go func(projectDir string) {
			cache := buildWorkspaceCache(projectDir)
			results <- cacheResult{cache: cache}
		}(dir)
	}

	// Merge all caches into a unified cache
	merged := &workspaceCache{
		beadsToWorkspace:     make(map[string]string),
		beadsToProjectDir:    make(map[string]string),
		workspaceEntryToPath: make(map[string]string),
	}

	for i := 0; i < len(projectDirs); i++ {
		result := <-results

		// Merge beadsToWorkspace map (later entries don't overwrite earlier ones)
		for beadsID, wsPath := range result.cache.beadsToWorkspace {
			if _, exists := merged.beadsToWorkspace[beadsID]; !exists {
				merged.beadsToWorkspace[beadsID] = wsPath
			}
		}

		// Merge beadsToProjectDir map
		for beadsID, projDir := range result.cache.beadsToProjectDir {
			if _, exists := merged.beadsToProjectDir[beadsID]; !exists {
				merged.beadsToProjectDir[beadsID] = projDir
			}
		}

		// Merge workspaceEntryToPath map (for multi-project workspace path resolution)
		for entryName, wsPath := range result.cache.workspaceEntryToPath {
			if _, exists := merged.workspaceEntryToPath[entryName]; !exists {
				merged.workspaceEntryToPath[entryName] = wsPath
			}
		}

		// Merge workspace entries (for completed workspace scanning)
		merged.workspaceEntries = append(merged.workspaceEntries, result.cache.workspaceEntries...)

		// Keep track of workspace dir for backward compatibility
		// (use first non-empty workspace dir)
		if merged.workspaceDir == "" && result.cache.workspaceDir != "" {
			merged.workspaceDir = result.cache.workspaceDir
		}
	}

	return merged
}

// buildWorkspaceCache scans the workspace directory once and builds lookup maps.
// This replaces multiple calls to findWorkspaceByBeadsID which each scanned all 400+ directories.
func buildWorkspaceCache(projectDir string) *workspaceCache {
	cache := &workspaceCache{
		beadsToWorkspace:     make(map[string]string),
		beadsToProjectDir:    make(map[string]string),
		workspaceDir:         filepath.Join(projectDir, ".orch", "workspace"),
		workspaceEntryToPath: make(map[string]string),
	}

	entries, err := os.ReadDir(cache.workspaceDir)
	if err != nil {
		return cache // Empty cache if directory doesn't exist
	}
	cache.workspaceEntries = entries

	// Single scan: extract beads ID and project dir from each workspace
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		dirPath := filepath.Join(cache.workspaceDir, dirName)
		spawnContextPath := filepath.Join(dirPath, "SPAWN_CONTEXT.md")

		// Store entry name to absolute path mapping for multi-project support
		cache.workspaceEntryToPath[dirName] = dirPath

		// Read SPAWN_CONTEXT.md once to extract both beads ID and PROJECT_DIR
		content, err := os.ReadFile(spawnContextPath)
		if err != nil {
			continue // Skip workspaces without SPAWN_CONTEXT.md
		}
		contentStr := string(content)

		var beadsID, agentProjectDir string

		// Parse once, extracting both pieces of info
		for _, line := range strings.Split(contentStr, "\n") {
			lineTrimmed := strings.TrimSpace(line)

			// Extract beads ID from "spawned from beads issue: **xxx**" or "bd comment xxx"
			if strings.Contains(strings.ToLower(line), "spawned from beads issue:") {
				// Pattern: "spawned from beads issue: **orch-go-xxxx**"
				// Extract the beads ID between ** markers or after the colon
				if idx := strings.Index(line, "**"); idx != -1 {
					rest := line[idx+2:]
					if endIdx := strings.Index(rest, "**"); endIdx != -1 {
						beadsID = rest[:endIdx]
					}
				}
			} else if strings.HasPrefix(lineTrimmed, "bd comment ") {
				// Pattern: "bd comment orch-go-xxxx ..."
				parts := strings.Fields(lineTrimmed)
				if len(parts) >= 3 {
					beadsID = parts[2]
				}
			}

			// Extract PROJECT_DIR
			if strings.HasPrefix(lineTrimmed, "PROJECT_DIR:") {
				agentProjectDir = strings.TrimSpace(strings.TrimPrefix(lineTrimmed, "PROJECT_DIR:"))
			}
		}

		// Store in cache if beads ID found
		if beadsID != "" {
			cache.beadsToWorkspace[beadsID] = dirPath
			if agentProjectDir != "" {
				cache.beadsToProjectDir[beadsID] = agentProjectDir
			}
		}
	}

	return cache
}

// lookupWorkspace returns the workspace path for a beads ID (O(1) lookup).
func (c *workspaceCache) lookupWorkspace(beadsID string) string {
	return c.beadsToWorkspace[beadsID]
}

// lookupProjectDir returns the PROJECT_DIR for a beads ID (O(1) lookup).
func (c *workspaceCache) lookupProjectDir(beadsID string) string {
	return c.beadsToProjectDir[beadsID]
}

// lookupWorkspacePathByEntry returns the absolute workspace path for a directory entry name.
// This is used in multi-project scenarios where workspace entries come from different projects.
func (c *workspaceCache) lookupWorkspacePathByEntry(entryName string) string {
	if path, ok := c.workspaceEntryToPath[entryName]; ok {
		return path
	}
	// Fallback to single-project path construction
	if c.workspaceDir != "" {
		return filepath.Join(c.workspaceDir, entryName)
	}
	return ""
}

// handleAgents returns JSON list of active agents from OpenCode/tmux and completed workspaces.
func handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Use sourceDir (set at build time) since serve may run from any working directory
	projectDir := sourceDir
	if projectDir == "" || projectDir == "unknown" {
		projectDir, _ = os.Getwd()
	}

	client := opencode.NewClient(serverURL)

	// Get active sessions from OpenCode
	// Don't filter by directory - show all sessions across all projects
	// (serve process CWD may not match project directory)
	sessions, err := client.ListSessions("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list sessions: %v", err), http.StatusInternalServerError)
		return
	}

	// Build multi-project workspace cache for cross-project agent visibility.
	// This aggregates workspace metadata from all projects with active sessions,
	// enabling the dashboard to show correct status for agents spawned with --workdir.
	// Previously: Only scanned current project's .orch/workspace/
	// Now: Scans all unique project directories found in OpenCode sessions
	// CACHED: Workspace scanning is slow (600+ files), so cache with 30s TTL.
	projectDirs := extractUniqueProjectDirs(sessions, projectDir)
	wsCache := globalWorkspaceCacheInstance.getCachedWorkspace(projectDirs)

	now := time.Now()
	agents := []AgentAPIResponse{} // Initialize as empty slice, not nil, to return [] instead of null

	// Collect beads IDs for batch fetching
	var beadsIDsToFetch []string
	seenBeadsIDs := make(map[string]bool)

	// Track project directories for cross-project agents
	// Key: beadsID, Value: projectDir from workspace SPAWN_CONTEXT.md
	beadsProjectDirs := make(map[string]string)

	// Add active sessions from OpenCode
	// Filter: only show sessions updated in the last 10 minutes as "active"
	// Sessions idle > 30 min are filtered out AFTER checking beads Phase status
	// (completed agents should still be shown regardless of activity time)
	activeThreshold := 10 * time.Minute
	displayThreshold := 30 * time.Minute

	// Track which agents need post-filtering by beads ID (idle > displayThreshold)
	// These will be filtered out after Phase check unless Phase: Complete
	pendingFilterByBeadsID := make(map[string]bool)

	// Track agents by title to deduplicate (OpenCode can have multiple sessions with same title)
	// Keep the most recently updated session for each title
	seenTitles := make(map[string]int) // title -> index in agents slice

	for _, s := range sessions {
		createdAt := time.Unix(s.Time.Created/1000, 0)
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		runtime := now.Sub(createdAt)
		timeSinceUpdate := now.Sub(updatedAt)

		// Determine status based on recent activity
		status := "active"
		if timeSinceUpdate > activeThreshold {
			status = "idle" // Session exists but hasn't had recent activity
		}

		// NOTE: IsProcessing is now populated client-side via SSE session.status events.
		// Previously we called client.IsSessionProcessing(s.ID) here, but that makes
		// an HTTP call per session which caused 125% CPU when dashboard polled frequently.
		// The frontend already receives busy/idle state from OpenCode SSE and updates
		// is_processing in real-time, so we don't need to fetch it here.

		agent := AgentAPIResponse{
			ID:           s.Title,
			SessionID:    s.ID,
			Status:       status,
			Runtime:      formatDuration(runtime),
			SpawnedAt:    createdAt.Format(time.RFC3339),
			UpdatedAt:    updatedAt.Format(time.RFC3339),
			IsProcessing: false, // Populated client-side via SSE
		}

		// Derive beadsID and skill from session title
		if s.Title != "" {
			agent.BeadsID = extractBeadsIDFromTitle(s.Title)
			agent.Skill = extractSkillFromTitle(s.Title)
			agent.Project = extractProjectFromBeadsID(agent.BeadsID)

		}

		// Only include sessions that were spawned via orch spawn (have beads ID)
		// This filters out interactive/ad-hoc OpenCode sessions
		if agent.BeadsID == "" {
			continue
		}

		// Track if this agent should be filtered after Phase check
		// Don't filter yet - we need to check beads Phase: Complete first
		if status == "idle" && timeSinceUpdate > displayThreshold {
			pendingFilterByBeadsID[agent.BeadsID] = true
		}

		// Collect beads ID for batch fetch - include ALL agents with beads ID.
		// Previously this had a `status == "active"` optimization that skipped idle agents,
		// but this caused incorrect status for idle agents with Phase: Complete.
		// The TTL cache prevents CPU spikes, so we can safely fetch for all agents.
		// See .kb/investigations/2026-01-04-design-dashboard-agent-status-model.md
		if agent.BeadsID != "" && !seenBeadsIDs[agent.BeadsID] {
			beadsIDsToFetch = append(beadsIDsToFetch, agent.BeadsID)
			seenBeadsIDs[agent.BeadsID] = true

			// For cross-project agent visibility: use cached PROJECT_DIR
			// This replaces expensive directory scanning with O(1) lookup
			if agentProjectDir := wsCache.lookupProjectDir(agent.BeadsID); agentProjectDir != "" {
				beadsProjectDirs[agent.BeadsID] = agentProjectDir
			}
		}

		// Deduplicate by title - keep the most recently updated session
		// OpenCode can have multiple sessions with the same title (e.g., resumed agents)
		if existingIdx, exists := seenTitles[s.Title]; exists {
			// Compare updated_at to keep the more recent session
			existingUpdatedAt, _ := time.Parse(time.RFC3339, agents[existingIdx].UpdatedAt)
			if updatedAt.After(existingUpdatedAt) {
				// Replace the existing agent with this newer one
				agents[existingIdx] = agent
			}
			// Skip appending since we either replaced or kept the existing one
			continue
		}

		seenTitles[s.Title] = len(agents)
		agents = append(agents, agent)
	}

	// Add tmux-only agents
	workersSessions, _ := tmux.ListWorkersSessions()
	for _, sessionName := range workersSessions {
		windows, _ := tmux.ListWindows(sessionName)
		for _, win := range windows {
			if win.Name == "servers" || win.Name == "zsh" {
				continue
			}

			beadsID := extractBeadsIDFromWindowName(win.Name)
			skill := extractSkillFromWindowName(win.Name)
			project := extractProjectFromBeadsID(beadsID)

			// Check if already in agents list
			alreadyIn := false
			for _, a := range agents {
				if (beadsID != "" && a.BeadsID == beadsID) || (a.ID != "" && strings.Contains(win.Name, a.ID)) {
					alreadyIn = true
					break
				}
			}

			if !alreadyIn {
				agents = append(agents, AgentAPIResponse{
					ID:      win.Name,
					BeadsID: beadsID,
					Skill:   skill,
					Project: project,
					Status:  "active",
					Window:  win.Target,
				})

				// Collect beads ID for batch fetch
				if beadsID != "" && !seenBeadsIDs[beadsID] {
					beadsIDsToFetch = append(beadsIDsToFetch, beadsID)
					seenBeadsIDs[beadsID] = true

					// For cross-project agent visibility: use cached PROJECT_DIR
					// This replaces expensive directory scanning with O(1) lookup
					if agentProjectDir := wsCache.lookupProjectDir(beadsID); agentProjectDir != "" {
						beadsProjectDirs[beadsID] = agentProjectDir
					}
				}
			}
		}
	}

	// Add completed workspaces (those with SYNTHESIS.md or light-tier completions)
	// Reuse cached workspace entries to avoid redundant directory reads
	// Multi-project support: entries may come from different project workspace directories
	if len(wsCache.workspaceEntries) > 0 {
		for _, entry := range wsCache.workspaceEntries {
			if !entry.IsDir() {
				continue
			}

			// Check if already in active list
			// Active session IDs have format "workspace [beads-id]", workspace names don't
			alreadyIn := false
			workspaceName := entry.Name()
			for _, a := range agents {
				if a.ID == workspaceName || strings.HasPrefix(a.ID, workspaceName+" ") {
					alreadyIn = true
					break
				}
			}

			if alreadyIn {
				continue
			}

			// Use the lookup method for multi-project support
			workspacePath := wsCache.lookupWorkspacePathByEntry(entry.Name())
			synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
			hasSynthesis := false

			// Check if SYNTHESIS.md exists (indicates full-tier completion)
			if _, err := os.Stat(synthesisPath); err == nil {
				hasSynthesis = true
			}

			// Only add workspaces that have SYNTHESIS.md for now
			// Light-tier completions will be detected via Phase: Complete in beads comments
			if !hasSynthesis {
				// For light-tier, check if there's a SPAWN_CONTEXT.md (indicates it's a valid spawn)
				spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
				if _, err := os.Stat(spawnContextPath); err != nil {
					continue // Not a valid spawn workspace
				}
			}

			agent := AgentAPIResponse{
				ID:     entry.Name(),
				Status: "completed",
			}

			// Set updated_at from workspace name date suffix or file modification time
			// This ensures proper sorting in archive section
			if parsedDate := extractDateFromWorkspaceName(entry.Name()); !parsedDate.IsZero() {
				agent.UpdatedAt = parsedDate.Format(time.RFC3339)
			} else if hasSynthesis {
				// Fallback to file modification time of SYNTHESIS.md
				if info, err := os.Stat(synthesisPath); err == nil {
					agent.UpdatedAt = info.ModTime().Format(time.RFC3339)
				}
			} else {
				// For light-tier, use SPAWN_CONTEXT.md modification time
				spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
				if info, err := os.Stat(spawnContextPath); err == nil {
					agent.UpdatedAt = info.ModTime().Format(time.RFC3339)
				}
			}

			// Read session ID from workspace
			if sessionID := spawn.ReadSessionID(workspacePath); sessionID != "" {
				agent.SessionID = sessionID
			}

			// Parse synthesis (only for full-tier)
			if hasSynthesis {
				if synthesis, err := verify.ParseSynthesis(workspacePath); err == nil {
					agent.Synthesis = &SynthesisResponse{
						TLDR:           synthesis.TLDR,
						Outcome:        synthesis.Outcome,
						Recommendation: synthesis.Recommendation,
						DeltaSummary:   summarizeDelta(synthesis.Delta),
						NextActions:    synthesis.NextActions,
					}
				}
			}

			// Extract beadsID from workspace SPAWN_CONTEXT.md (more reliable than parsing name)
			agent.BeadsID = extractBeadsIDFromWorkspace(workspacePath)
			// Fallback to extracting from workspace name if SPAWN_CONTEXT.md doesn't have it
			if agent.BeadsID == "" {
				agent.BeadsID = extractBeadsIDFromTitle(entry.Name())
			}
			agent.Skill = extractSkillFromTitle(entry.Name())

			// NOTE: We intentionally DON'T extract PROJECT_DIR or fetch beads data for completed workspaces.
			// Completed workspaces have already done their work - they don't need phase updates.
			// The close_reason is nice-to-have but not worth spawning bd processes.
			// This optimization prevents CPU spikes from fetching beads data for 600+ historical workspaces.

			agents = append(agents, agent)
		}
	}

	// Batch fetch beads data (phase from comments, task from issues, close_reason for completed)
	// This is the same pattern used by orch status for efficiency.
	// Uses TTL cache to prevent CPU spikes from spawning 20+ bd processes per request.
	if len(beadsIDsToFetch) > 0 {
		// Fetch all open issues in one call (cached with TTL)
		openIssues, _ := globalBeadsCache.getOpenIssues()

		// Batch fetch all issues (including closed) for close_reason (cached with TTL)
		// Uses bd show which works for any issue status
		allIssues, _ := globalBeadsCache.getAllIssues(beadsIDsToFetch)

		// Batch fetch comments for all beads IDs (cached with TTL)
		// Use project-aware batch fetch for cross-project agent visibility
		commentsMap := globalBeadsCache.getComments(beadsIDsToFetch, beadsProjectDirs)

		// Populate phase, task, close_reason, and status for each agent using Priority Cascade model.
		// See .kb/investigations/2026-01-04-design-dashboard-agent-status-model.md for design.
		for i := range agents {
			if agents[i].BeadsID == "" {
				continue
			}

			// Get task from open issue title first
			if issue, ok := openIssues[agents[i].BeadsID]; ok {
				agents[i].Task = truncate(issue.Title, 60)
			}

			// If not in open issues, try all issues (for closed ones)
			if agents[i].Task == "" {
				if issue, ok := allIssues[agents[i].BeadsID]; ok {
					agents[i].Task = truncate(issue.Title, 60)
					// For completed agents without synthesis, use close_reason as fallback
					if agents[i].Synthesis == nil && issue.CloseReason != "" {
						agents[i].CloseReason = issue.CloseReason
					}
				}
			}

			// Gather completion signals for Priority Cascade model
			issueClosed := false
			phaseComplete := false

			// Check if beads issue is closed (Priority 1)
			if issue, ok := allIssues[agents[i].BeadsID]; ok {
				issueClosed = strings.EqualFold(issue.Status, "closed")
				// Capture close_reason if available
				if issueClosed && issue.CloseReason != "" && agents[i].CloseReason == "" {
					agents[i].CloseReason = issue.CloseReason
				}
			}

			// Get phase from comments (Priority 2)
			if comments, ok := commentsMap[agents[i].BeadsID]; ok {
				phaseStatus := verify.ParsePhaseFromComments(comments)
				if phaseStatus.Found {
					agents[i].Phase = phaseStatus.Phase
					phaseComplete = strings.EqualFold(phaseStatus.Phase, "Complete")
				}
			}

			// Get workspace path for SYNTHESIS.md check (Priority 3)
			workspacePath := wsCache.lookupWorkspace(agents[i].BeadsID)
			// Fallback: For untracked agents, try looking up by workspace name from session title
			if workspacePath == "" && agents[i].ID != "" {
				workspaceName := agents[i].ID
				if idx := strings.Index(workspaceName, " ["); idx != -1 {
					workspaceName = workspaceName[:idx]
				}
				workspacePath = wsCache.lookupWorkspacePathByEntry(workspaceName)
			}

			// Use Priority Cascade to determine final status
			// Priority order: issueClosed > phaseComplete > SYNTHESIS.md > sessionStatus
			agents[i].Status = determineAgentStatus(issueClosed, phaseComplete, workspacePath, agents[i].Status)

			// For completed agents, also check close_reason if synthesis is null
			if agents[i].Status == "completed" && agents[i].Synthesis == nil && agents[i].CloseReason == "" {
				if issue, ok := allIssues[agents[i].BeadsID]; ok && issue.CloseReason != "" {
					agents[i].CloseReason = issue.CloseReason
				}
			}
		}

		// Fetch gap analysis from spawn events for each agent
		gapAnalysisMap := getGapAnalysisFromEvents(beadsIDsToFetch)
		for i := range agents {
			if agents[i].BeadsID == "" {
				continue
			}
			if gapData, ok := gapAnalysisMap[agents[i].BeadsID]; ok {
				agents[i].GapAnalysis = gapData
			}
		}

		// Post-Phase filtering: remove agents that were idle > displayThreshold
		// and are NOT Phase: Complete. This deferred filtering ensures completed
		// agents are shown regardless of activity time.
		filtered := make([]AgentAPIResponse, 0, len(agents))
		for _, agent := range agents {
			if pendingFilterByBeadsID[agent.BeadsID] && agent.Status != "completed" {
				// Skip idle agents that are not completed
				continue
			}
			filtered = append(filtered, agent)
		}
		agents = filtered
	}

	// NOTE: The duplicate SYNTHESIS.md check that was here has been removed.
	// All agents are now included in beadsIDsToFetch, so the Priority Cascade
	// in determineAgentStatus() handles all status determination in one place.
	// See .kb/investigations/2026-01-04-design-dashboard-agent-status-model.md

	// Fetch token usage for agents with valid session IDs
	// Parallelized to avoid sequential HTTP calls causing ~20s delays with 200+ agents.
	// Uses goroutines with semaphore to limit concurrent requests.
	type tokenResult struct {
		index  int
		tokens *opencode.TokenStats
	}
	tokenChan := make(chan tokenResult, len(agents))

	// Limit concurrent HTTP requests to avoid overwhelming the OpenCode server
	const maxConcurrent = 20
	sem := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup
	for i := range agents {
		// Skip agents without session ID, completed agents, or idle agents.
		// Token data is static for completed agents, and idle agents are unlikely to have changed.
		// This prevents CPU spikes from making HTTP calls to OpenCode for 300+ inactive sessions.
		if agents[i].SessionID == "" || agents[i].Status == "completed" || agents[i].Status == "idle" {
			continue
		}

		wg.Add(1)
		go func(idx int, sessionID string) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			tokens, err := client.GetSessionTokens(sessionID)
			if err == nil && tokens != nil {
				tokenChan <- tokenResult{index: idx, tokens: tokens}
			}
		}(i, agents[i].SessionID)
	}

	// Wait for all goroutines to complete, then close channel
	go func() {
		wg.Wait()
		close(tokenChan)
	}()

	// Collect results
	for result := range tokenChan {
		agents[result.index].Tokens = result.tokens
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(agents); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode agents: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleEvents proxies the OpenCode SSE stream to the client.
// It connects to http://localhost:4096/event and forwards events.
func handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Get flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Connect to OpenCode SSE stream
	opencodeURL := serverURL + "/event"
	resp, err := http.Get(opencodeURL)
	if err != nil {
		// Send error as SSE event
		fmt.Fprintf(w, "event: error\ndata: {\"error\": \"Failed to connect to OpenCode: %s\"}\n\n", err.Error())
		flusher.Flush()
		return
	}
	defer resp.Body.Close()

	// Check if OpenCode returned an error
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(w, "event: error\ndata: {\"error\": \"OpenCode returned status %d\"}\n\n", resp.StatusCode)
		flusher.Flush()
		return
	}

	// Send connected event
	fmt.Fprintf(w, "event: connected\ndata: {\"source\": \"%s\"}\n\n", opencodeURL)
	flusher.Flush()

	// Create a done channel to handle client disconnect
	ctx := r.Context()

	// Read and forward SSE events
	reader := bufio.NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					// Connection closed by OpenCode
					fmt.Fprintf(w, "event: disconnected\ndata: {\"reason\": \"upstream closed\"}\n\n")
					flusher.Flush()
					return
				}
				// Read error
				fmt.Fprintf(w, "event: error\ndata: {\"error\": \"Read error: %s\"}\n\n", err.Error())
				flusher.Flush()
				return
			}

			// Forward the line as-is (preserves SSE format)
			if strings.HasPrefix(line, "data:") {
				fmt.Printf("Forwarding SSE event: %s", line)
			}
			fmt.Fprint(w, line)
			flusher.Flush()
		}
	}
}

// handleAgentlog returns agent lifecycle events from ~/.orch/events.jsonl.
// Without query params: returns last 100 events as JSON array.
// With ?follow=true: streams new events via SSE.
func handleAgentlog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	follow := r.URL.Query().Get("follow") == "true"

	if follow {
		handleAgentlogSSE(w, r)
	} else {
		handleAgentlogJSON(w, r)
	}
}

// handleAgentlogJSON returns the last 100 events as JSON array.
func handleAgentlogJSON(w http.ResponseWriter, r *http.Request) {
	logPath := events.DefaultLogPath()

	eventList, err := readLastNEvents(logPath, 100)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty array if file doesn't exist yet
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
			return
		}
		http.Error(w, fmt.Sprintf("Failed to read events: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(eventList); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode events: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleAgentlogSSE streams new events via SSE as they are appended to events.jsonl.
func handleAgentlogSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Get flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	logPath := events.DefaultLogPath()
	ctx := r.Context()

	// Open file for reading
	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Send connected event, file doesn't exist yet
			fmt.Fprintf(w, "event: connected\ndata: {\"source\": \"%s\", \"status\": \"waiting\"}\n\n", logPath)
			flusher.Flush()
		} else {
			fmt.Fprintf(w, "event: error\ndata: {\"error\": \"Failed to open log file: %s\"}\n\n", err.Error())
			flusher.Flush()
			return
		}
	} else {
		defer file.Close()
	}

	// Send connected event
	fmt.Fprintf(w, "event: connected\ndata: {\"source\": \"%s\"}\n\n", logPath)
	flusher.Flush()

	// Seek to end of file to only stream new events
	if file != nil {
		file.Seek(0, io.SeekEnd)
	}

	// Poll for new events
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	reader := bufio.NewReader(file)
	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return
		case <-ticker.C:
			// Try to read new lines
			if file == nil {
				// Try to open file if it didn't exist before
				file, err = os.Open(logPath)
				if err != nil {
					continue // File still doesn't exist
				}
				file.Seek(0, io.SeekEnd)
				reader = bufio.NewReader(file)
			}

			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						break // No more data, wait for next poll
					}
					fmt.Fprintf(w, "event: error\ndata: {\"error\": \"Read error: %s\"}\n\n", err.Error())
					flusher.Flush()
					return
				}

				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				// Validate it's valid JSON and forward as SSE event
				var event events.Event
				if err := json.Unmarshal([]byte(line), &event); err != nil {
					continue // Skip invalid lines
				}

				fmt.Fprintf(w, "event: agentlog\ndata: %s\n\n", line)
				flusher.Flush()
			}
		}
	}
}

// readLastNEvents reads the last n events from a JSONL file.
func readLastNEvents(path string, n int) ([]events.Event, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var allEvents []events.Event
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event events.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue // Skip invalid lines
		}
		allEvents = append(allEvents, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Return last n events
	if len(allEvents) > n {
		return allEvents[len(allEvents)-n:], nil
	}
	return allEvents, nil
}

// getProjectAPIPort returns the allocated API port for the current project.
// Returns 0 if no allocation exists or on error.
func getProjectAPIPort() int {
	projectDir, err := os.Getwd()
	if err != nil {
		return 0
	}
	projectName := filepath.Base(projectDir)

	registry, err := port.New("")
	if err != nil {
		return 0
	}

	alloc := registry.Find(projectName, "api")
	if alloc == nil {
		return 0
	}

	return alloc.Port
}

// checkWorkspaceSynthesis checks if a workspace has a non-empty SYNTHESIS.md file.
// This is used to detect completion for untracked agents (--no-track) where
// there's no beads issue to check Phase: Complete.
func checkWorkspaceSynthesis(workspacePath string) bool {
	if workspacePath == "" {
		return false
	}
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	info, err := os.Stat(synthesisPath)
	if err != nil {
		return false
	}
	// SYNTHESIS.md must exist and be non-empty
	return info.Size() > 0
}

// determineAgentStatus implements the Priority Cascade model for agent status.
// This is the single source of truth for determining agent status.
//
// Priority order (highest to lowest):
//  1. Beads issue closed → "completed" (orchestrator verified completion)
//  2. Phase: Complete reported → "completed" (agent declared done)
//  3. SYNTHESIS.md exists → "completed" (artifact proves completion)
//  4. Session activity → sessionStatus ("active" or "idle")
//
// This replaces the scattered status determination logic with a clear, deterministic model.
// See .kb/investigations/2026-01-04-design-dashboard-agent-status-model.md for design rationale.
func determineAgentStatus(issueClosed bool, phaseComplete bool, workspacePath string, sessionStatus string) string {
	// Priority 1: Beads issue closed → completed (orchestrator verified completion)
	if issueClosed {
		return "completed"
	}

	// Priority 2: Phase: Complete reported → completed (agent declared done)
	if phaseComplete {
		return "completed"
	}

	// Priority 3: SYNTHESIS.md exists → completed (artifact proves completion)
	if checkWorkspaceSynthesis(workspacePath) {
		return "completed"
	}

	// Priority 4: Session activity (fallback)
	return sessionStatus
}

// getGapAnalysisFromEvents reads spawn events and extracts gap analysis data for given beads IDs.
// Returns a map of beads ID -> GapAPIResponse.
func getGapAnalysisFromEvents(beadsIDs []string) map[string]*GapAPIResponse {
	result := make(map[string]*GapAPIResponse)
	if len(beadsIDs) == 0 {
		return result
	}

	// Build a set of beads IDs for fast lookup
	beadsIDSet := make(map[string]bool)
	for _, id := range beadsIDs {
		beadsIDSet[id] = true
	}

	// Read events file
	logPath := events.DefaultLogPath()
	file, err := os.Open(logPath)
	if err != nil {
		return result
	}
	defer file.Close()

	// Scan events for spawn events matching our beads IDs
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event events.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		// Only process spawn events
		if event.Type != "session.spawned" {
			continue
		}

		// Check if this event is for one of our beads IDs
		beadsID, ok := event.Data["beads_id"].(string)
		if !ok || !beadsIDSet[beadsID] {
			continue
		}

		// Already have gap analysis for this beads ID? Skip (we want the most recent)
		// Since we read chronologically, later entries overwrite earlier ones
		if _, exists := result[beadsID]; exists {
			// We could skip, but let's allow overwrites for resumptions
		}

		// Extract gap analysis data from event
		gapData := extractGapAnalysisFromEvent(event.Data)
		if gapData != nil {
			result[beadsID] = gapData
		}
	}

	return result
}

// extractGapAnalysisFromEvent extracts gap analysis data from a spawn event's data map.
func extractGapAnalysisFromEvent(data map[string]interface{}) *GapAPIResponse {
	// Check if gap data exists
	hasGaps, ok := data["gap_has_gaps"].(bool)
	if !ok {
		return nil
	}

	contextQuality := 0
	if cq, ok := data["gap_context_quality"].(float64); ok {
		contextQuality = int(cq)
	}

	shouldWarn := false
	if sw, ok := data["gap_should_warn"].(bool); ok {
		shouldWarn = sw
	}

	matchCount := 0
	if mc, ok := data["gap_match_total"].(float64); ok {
		matchCount = int(mc)
	}

	constraints := 0
	if c, ok := data["gap_match_constraints"].(float64); ok {
		constraints = int(c)
	}

	decisions := 0
	if d, ok := data["gap_match_decisions"].(float64); ok {
		decisions = int(d)
	}

	investigations := 0
	if i, ok := data["gap_match_investigations"].(float64); ok {
		investigations = int(i)
	}

	return &GapAPIResponse{
		HasGaps:        hasGaps,
		ContextQuality: contextQuality,
		ShouldWarn:     shouldWarn,
		MatchCount:     matchCount,
		Constraints:    constraints,
		Decisions:      decisions,
		Investigations: investigations,
	}
}

// handleCacheInvalidate clears all dashboard caches to force fresh data on next request.
// This is called by orch complete to ensure the dashboard shows updated agent status.
// Without this, the TTL cache holds stale "active" status after agents complete.
func handleCacheInvalidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Invalidate beads cache (open issues, all issues, comments)
	if globalBeadsCache != nil {
		globalBeadsCache.invalidate()
	}

	// Invalidate workspace cache (workspace metadata)
	globalWorkspaceCacheInstance.invalidate()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Cache invalidated",
	})
}
