package main

import (
	"os"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/cache"
)

// projectCacheEntry holds cached data for a single project.
type projectCacheEntry struct {
	lastAccessedAt time.Time

	stats          *beads.Stats
	statsFetchedAt time.Time

	readyIssues    []beads.Issue
	readyFetchedAt time.Time

	// Graph cache keyed by "scope:parent" (e.g., "focus:", "open:", "focus:orch-go-123")
	graphCache map[string]*graphCacheEntry

	// Dependency graph cache keyed by scope (e.g., "open", "all")
	dependencyGraphCache map[string]*dependencyGraphCacheEntry
}

// graphCacheEntry holds a cached graph response.
type graphCacheEntry struct {
	response  *BeadsGraphAPIResponse
	fetchedAt time.Time
}

// dependencyGraphCacheEntry holds cached dependency edges for a scope.
type dependencyGraphCacheEntry struct {
	edges     []GraphEdge
	fetchedAt time.Time
}

// beadsStatsCache provides TTL-based caching for /api/beads, /api/beads/ready, and /api/beads/graph.
// Without caching, each request spawns a bd process which takes ~1.5s for stats.
// With 5s TTL, most dashboard polls hit cache (instant) while data stays fresh.
// Cache is project-aware: each project_dir has its own cache entry.
type beadsStatsCache struct {
	mu sync.RWMutex

	maxEntries int

	// Per-project cache entries (keyed by project directory)
	// Empty string key is used for default project (sourceDir)
	projects map[string]*projectCacheEntry

	// TTL for stats, ready issues, and graph
	statsTTL time.Duration
	readyTTL time.Duration
	graphTTL time.Duration
}

const (
	defaultBeadsStatsCacheTTL        = 5 * time.Second
	defaultBeadsStatsCacheMaxEntries = 256
)

func newBeadsStatsCache(maxSize int, ttl time.Duration) *beadsStatsCache {
	bounds := cache.NewNamedCache("beads stats cache", maxSize, ttl)

	return &beadsStatsCache{
		maxEntries: bounds.MaxSize(),
		projects:   make(map[string]*projectCacheEntry),
		statsTTL:   bounds.TTL(),
		readyTTL:   bounds.TTL(),
		graphTTL:   bounds.TTL(),
	}
}

// getOrCreateEntry returns the cache entry for a project, creating one if needed.
func (c *beadsStatsCache) getOrCreateEntry(projectDir string) *projectCacheEntry {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.projects == nil {
		c.projects = make(map[string]*projectCacheEntry)
	}

	now := time.Now()
	entry, ok := c.projects[projectDir]
	if ok {
		entry.lastAccessedAt = now
		return entry
	}

	if len(c.projects) >= c.maxEntries {
		c.evictOldestProjectLocked()
	}

	entry = &projectCacheEntry{
		lastAccessedAt: now,
	}
	c.projects[projectDir] = entry

	return entry
}

func (c *beadsStatsCache) evictOldestProjectLocked() {
	var oldestProject string
	var oldestTime time.Time

	for projectDir, entry := range c.projects {
		entryTime := entry.lastAccessedAt
		if entryTime.IsZero() {
			entryTime = entry.statsFetchedAt
		}
		if entryTime.IsZero() {
			entryTime = entry.readyFetchedAt
		}
		if entryTime.IsZero() {
			entryTime = time.Unix(0, 0)
		}

		if oldestProject == "" || entryTime.Before(oldestTime) {
			oldestProject = projectDir
			oldestTime = entryTime
		}
	}

	if oldestProject != "" {
		delete(c.projects, oldestProject)
	}
}

// getStats returns cached stats or fetches fresh if stale.
func (c *beadsStatsCache) getStats(srv *Server, projectDir string) (*beads.Stats, error) {
	entry := c.getOrCreateEntry(projectDir)

	c.mu.RLock()
	if entry.stats != nil && time.Since(entry.statsFetchedAt) < c.statsTTL {
		result := entry.stats
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	workDir := projectDir
	if workDir == "" {
		workDir = beads.DefaultDir
	}

	result, err, _ := srv.bdLimitedStats(workDir, func() (interface{}, error) {
		socketPath, findErr := beads.FindSocketPath(workDir)
		socketExists := findErr == nil && socketPath != ""
		if socketExists {
			if _, statErr := os.Stat(socketPath); statErr != nil {
				socketExists = false
			}
		}

		srv.BeadsClientMu.Lock()
		if !socketExists && srv.BeadsClient != nil {
			srv.BeadsClient.Close()
			srv.BeadsClient = nil
		}
		if socketExists && srv.BeadsClient == nil && socketPath != "" {
			srv.BeadsClient = beads.NewClient(socketPath,
				beads.WithAutoReconnect(3),
				beads.WithTimeout(5*time.Second),
			)
		}
		currentClient := srv.BeadsClient
		srv.BeadsClientMu.Unlock()

		var stats *beads.Stats
		var fetchErr error

		if projectDir != "" && projectDir != beads.DefaultDir {
			cliClient := beads.NewCLIClient(beads.WithWorkDir(projectDir))
			stats, fetchErr = cliClient.Stats()
		} else if currentClient != nil && socketExists {
			stats, fetchErr = currentClient.Stats()
			if fetchErr != nil {
				stats, fetchErr = beads.FallbackStats()
			}
		} else {
			stats, fetchErr = beads.FallbackStats()
		}
		return stats, fetchErr
	})

	if err != nil {
		return nil, err
	}

	stats := result.(*beads.Stats)
	c.mu.Lock()
	entry.stats = stats
	entry.statsFetchedAt = time.Now()
	c.mu.Unlock()

	return stats, nil
}

// getReadyIssues returns cached ready issues or fetches fresh if stale.
func (c *beadsStatsCache) getReadyIssues(srv *Server, projectDir string) ([]beads.Issue, error) {
	entry := c.getOrCreateEntry(projectDir)

	c.mu.RLock()
	if entry.readyIssues != nil && time.Since(entry.readyFetchedAt) < c.readyTTL {
		result := entry.readyIssues
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	workDir := projectDir
	if workDir == "" {
		workDir = beads.DefaultDir
	}

	result, err, _ := srv.bdLimitedReady(workDir, func() (interface{}, error) {
		socketPath, findErr := beads.FindSocketPath(workDir)
		socketExists := findErr == nil && socketPath != ""
		if socketExists {
			if _, statErr := os.Stat(socketPath); statErr != nil {
				socketExists = false
			}
		}

		srv.BeadsClientMu.Lock()
		if !socketExists && srv.BeadsClient != nil {
			srv.BeadsClient.Close()
			srv.BeadsClient = nil
		}
		if socketExists && srv.BeadsClient == nil && socketPath != "" {
			srv.BeadsClient = beads.NewClient(socketPath,
				beads.WithAutoReconnect(3),
				beads.WithTimeout(5*time.Second),
			)
		}
		currentClient := srv.BeadsClient
		srv.BeadsClientMu.Unlock()

		var issues []beads.Issue
		var fetchErr error

		if projectDir != "" && projectDir != beads.DefaultDir {
			cliClient := beads.NewCLIClient(beads.WithWorkDir(projectDir))
			issues, fetchErr = cliClient.Ready(nil)
		} else if currentClient != nil && socketExists {
			issues, fetchErr = currentClient.Ready(nil)
			if fetchErr != nil {
				issues, fetchErr = beads.FallbackReady()
			}
		} else {
			issues, fetchErr = beads.FallbackReady()
		}
		return issues, fetchErr
	})

	if err != nil {
		return nil, err
	}

	issues := result.([]beads.Issue)
	c.mu.Lock()
	entry.readyIssues = issues
	entry.readyFetchedAt = time.Now()
	c.mu.Unlock()

	return issues, nil
}

// getGraph returns a cached graph response or builds fresh if stale.
func (c *beadsStatsCache) getGraph(projectDir, cacheKey string, buildFn func() (*BeadsGraphAPIResponse, error)) (*BeadsGraphAPIResponse, error) {
	entry := c.getOrCreateEntry(projectDir)

	c.mu.RLock()
	if entry.graphCache != nil {
		if ge, ok := entry.graphCache[cacheKey]; ok && time.Since(ge.fetchedAt) < c.graphTTL {
			result := ge.response
			c.mu.RUnlock()
			return result, nil
		}
	}
	c.mu.RUnlock()

	resp, err := buildFn()
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	if entry.graphCache == nil {
		entry.graphCache = make(map[string]*graphCacheEntry)
	}
	if _, exists := entry.graphCache[cacheKey]; !exists && len(entry.graphCache) >= c.maxEntries {
		evictOldestGraphEntry(entry.graphCache)
	}
	entry.graphCache[cacheKey] = &graphCacheEntry{
		response:  resp,
		fetchedAt: time.Now(),
	}
	c.mu.Unlock()

	return resp, nil
}

// getDependencyGraph returns cached dependency edges or fetches fresh if stale.
func (c *beadsStatsCache) getDependencyGraph(projectDir, cacheKey string, buildFn func() ([]GraphEdge, error)) ([]GraphEdge, error) {
	entry := c.getOrCreateEntry(projectDir)

	c.mu.RLock()
	if entry.dependencyGraphCache != nil {
		if ge, ok := entry.dependencyGraphCache[cacheKey]; ok && time.Since(ge.fetchedAt) < c.graphTTL {
			result := ge.edges
			c.mu.RUnlock()
			return result, nil
		}
	}
	c.mu.RUnlock()

	edges, err := buildFn()
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	if entry.dependencyGraphCache == nil {
		entry.dependencyGraphCache = make(map[string]*dependencyGraphCacheEntry)
	}
	if _, exists := entry.dependencyGraphCache[cacheKey]; !exists && len(entry.dependencyGraphCache) >= c.maxEntries {
		evictOldestDependencyGraphEntry(entry.dependencyGraphCache)
	}
	entry.dependencyGraphCache[cacheKey] = &dependencyGraphCacheEntry{
		edges:     edges,
		fetchedAt: time.Now(),
	}
	c.mu.Unlock()

	return edges, nil
}

// invalidate clears cached data, forcing fresh fetches on next request.
func (c *beadsStatsCache) invalidate(projectDir string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if projectDir == "" {
		c.projects = make(map[string]*projectCacheEntry, c.maxEntries)
	} else {
		delete(c.projects, projectDir)
	}
}

func evictOldestGraphEntry(cache map[string]*graphCacheEntry) {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range cache {
		entryTime := entry.fetchedAt
		if oldestKey == "" || entryTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = entryTime
		}
	}

	if oldestKey != "" {
		delete(cache, oldestKey)
	}
}

func evictOldestDependencyGraphEntry(cache map[string]*dependencyGraphCacheEntry) {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range cache {
		entryTime := entry.fetchedAt
		if oldestKey == "" || entryTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = entryTime
		}
	}

	if oldestKey != "" {
		delete(cache, oldestKey)
	}
}
