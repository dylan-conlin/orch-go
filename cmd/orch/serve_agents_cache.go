package main

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
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
	fetchedAt   time.Time
	ttl         time.Duration
	projectDirs []string // Track which project dirs the cache was built with
}

// Global workspace cache
var globalWorkspaceCacheInstance = &globalWorkspaceCacheType{
	ttl: 30 * time.Second, // Workspace metadata changes infrequently
}

// getCachedWorkspace returns cached workspace data or builds fresh if stale.
// Rebuilds cache if:
// 1. Cache is nil (never built or invalidated)
// 2. Cache TTL expired
// 3. Project directories have changed (new projects registered)
func (c *globalWorkspaceCacheType) getCachedWorkspace(projectDirs []string) *workspaceCache {
	c.mu.RLock()
	cacheValid := c.cache != nil && time.Since(c.fetchedAt) < c.ttl
	dirsMatch := projectDirsMatch(c.projectDirs, projectDirs)
	if cacheValid && dirsMatch {
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
	c.projectDirs = projectDirs // Store the project dirs this cache was built with
	c.mu.Unlock()

	return wsCache
}

// projectDirsMatch checks if two slices of project directories contain the same entries.
// Order doesn't matter, but all entries must match.
func projectDirsMatch(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	// Create a set from a
	aSet := make(map[string]bool, len(a))
	for _, dir := range a {
		aSet[dir] = true
	}
	// Check all entries in b are in a
	for _, dir := range b {
		if !aSet[dir] {
			return false
		}
	}
	return true
}

// Default TTLs for cached data
// These TTLs balance freshness with performance. With singleflight dedup protecting
// cache misses, TTLs can be shorter without causing subprocess stampede.
// Use /api/cache/invalidate to force refresh when needed (e.g., after orch complete).
const (
	defaultOpenIssuesTTL = 10 * time.Second // Open issues — singleflight prevents stampede on cache miss
	defaultAllIssuesTTL  = 30 * time.Second // Closed issues change even less
	defaultCommentsTTL   = 10 * time.Second // Comments change more often (phase updates)
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
	c.projectDirs = nil
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
	issues, err := verify.GetIssuesBatch(beadsIDs, nil)
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

// workspaceCache stores pre-computed workspace metadata to avoid repeated directory scans.
// Built once per request and used for all lookups within that request.
type workspaceCache struct {
	// beadsToWorkspace maps beadsID -> workspace path (absolute)
	beadsToWorkspace map[string]string
	// beadsToProjectDir maps beadsID -> PROJECT_DIR from SPAWN_CONTEXT.md
	beadsToProjectDir map[string]string
	// workspaceEntries stores directory entries for reuse
	workspaceEntries []os.DirEntry
	// workspaceDir is the base workspace directory path
	workspaceDir string
	// workspaceEntryToPath maps directory entry name -> absolute workspace path
	// This is needed for multi-project scenarios where entries come from different projects
	workspaceEntryToPath map[string]string
}

// kbProject represents a project entry from kb projects list --json
type kbProject struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// getKBProjects fetches registered project directories from kb CLI.
// Returns empty slice if kb is unavailable or fails (graceful degradation).
// This enables cross-project workspace scanning by providing project paths
// independent of OpenCode session state.
func getKBProjects() []string {
	cmd := exec.Command("kb", "projects", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		// Log warning but don't fail - graceful degradation
		log.Printf("Warning: kb projects list failed: %v (cross-project visibility may be limited)", err)
		return []string{}
	}

	var projects []kbProject
	if err := json.Unmarshal(output, &projects); err != nil {
		log.Printf("Warning: failed to parse kb projects output: %v", err)
		return []string{}
	}

	paths := make([]string, 0, len(projects))
	for _, p := range projects {
		if p.Path != "" {
			// Normalize path
			paths = append(paths, filepath.Clean(p.Path))
		}
	}

	return paths
}

// extractUniqueProjectDirs collects unique project directories from OpenCode sessions
// and registered kb projects.
// Returns a deduplicated slice of directory paths that have active agents or are registered projects.
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

	// Add registered kb projects for cross-project visibility
	// This solves the problem where OpenCode --attach uses server cwd,
	// causing cross-project workspaces to never be scanned
	for _, proj := range getKBProjects() {
		if !seen[proj] {
			seen[proj] = true
			dirs = append(dirs, proj)
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

			// Extract beads ID from "spawned from beads issue: **xxx**" (authoritative source)
			// This line is the authoritative declaration of the agent's beads ID.
			// IMPORTANT: We must stop parsing beads ID after finding this line to prevent
			// template examples like "bd comment <beads-id>" from overwriting the correct value.
			// See .kb/investigations/2026-01-29-inv-dashboard-follow-mode-doesn-show.md
			if beadsID == "" && strings.Contains(strings.ToLower(line), "spawned from beads issue:") {
				// Pattern: "spawned from beads issue: **orch-go-xxxx**"
				// Extract the beads ID between ** markers or after the colon
				if idx := strings.Index(line, "**"); idx != -1 {
					rest := line[idx+2:]
					if endIdx := strings.Index(rest, "**"); endIdx != -1 {
						beadsID = rest[:endIdx]
					}
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
