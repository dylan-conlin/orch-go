package main

import (
	"os"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// projectCacheEntry holds cached data for a single project.
type projectCacheEntry struct {
	stats          *beads.Stats
	statsFetchedAt time.Time

	readyIssues    []beads.Issue
	readyFetchedAt time.Time

	reviewQueueItems []verify.UnverifiedItem
	reviewFetchedAt  time.Time

	graphIssues    []beads.Issue
	graphFetchedAt time.Time
}

// beadsStatsCache provides TTL-based caching for /api/beads and /api/beads/ready.
// Without caching, each request spawns a bd process which takes ~1.5s for stats.
// With 30s TTL, most dashboard polls hit cache (instant) while data stays fresh.
// Cache is project-aware: each project_dir has its own cache entry.
type beadsStatsCache struct {
	mu sync.RWMutex

	// Per-project cache entries (keyed by project directory)
	// Empty string key is used for default project (sourceDir)
	projects map[string]*projectCacheEntry

	// TTL for stats, ready issues, and graph data
	statsTTL  time.Duration
	readyTTL  time.Duration
	reviewTTL time.Duration
	graphTTL  time.Duration
}

// Global beads stats cache, initialized in runServe
var globalBeadsStatsCache *beadsStatsCache

func newBeadsStatsCache() *beadsStatsCache {
	return &beadsStatsCache{
		projects:  make(map[string]*projectCacheEntry),
		statsTTL:  30 * time.Second, // Stats change infrequently
		readyTTL:  15 * time.Second, // Ready queue changes more often
		reviewTTL: 15 * time.Second, // Review queue changes with completions
		graphTTL:  15 * time.Second, // Graph data changes with ready queue
	}
}

// getOrCreateEntry returns the cache entry for a project, creating one if needed.
func (c *beadsStatsCache) getOrCreateEntry(projectDir string) *projectCacheEntry {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.projects == nil {
		c.projects = make(map[string]*projectCacheEntry)
	}

	entry, ok := c.projects[projectDir]
	if !ok {
		entry = &projectCacheEntry{}
		c.projects[projectDir] = entry
	}
	return entry
}

// getStats returns cached stats or fetches fresh if stale.
// projectDir specifies which project's beads to query. Empty string uses default.
func (c *beadsStatsCache) getStats(projectDir string) (*beads.Stats, error) {
	entry := c.getOrCreateEntry(projectDir)

	c.mu.RLock()
	if entry.stats != nil && time.Since(entry.statsFetchedAt) < c.statsTTL {
		result := entry.stats
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	// Determine the directory to use
	workDir := projectDir
	if workDir == "" {
		workDir = sourceDir
	}

	// Fetch fresh stats
	var stats *beads.Stats
	var err error

	// Check if socket exists before attempting RPC to avoid slow timeout on dead daemon.
	// This happens when daemon crashes but server keeps stale connection reference.
	socketPath, findErr := beads.FindSocketPath(workDir)
	socketExists := findErr == nil && socketPath != ""
	if socketExists {
		if _, statErr := os.Stat(socketPath); statErr != nil {
			socketExists = false
		}
	}

	// Thread-safe cleanup of stale beadsClient when socket disappears.
	// This prevents holding broken connection state when daemon restarts.
	beadsClientMu.Lock()
	if !socketExists && beadsClient != nil {
		beadsClient.Close()
		beadsClient = nil
	}

	// Reinitialize beadsClient if socket reappears and client is nil.
	// This handles daemon restarts gracefully without server restart.
	if socketExists && beadsClient == nil && socketPath != "" {
		beadsClient = beads.NewClient(socketPath,
			beads.WithAutoReconnect(3),
			beads.WithTimeout(5*time.Second),
		)
		// Don't block on connection - let execute() handle reconnect
	}

	// Capture client reference under lock for use after unlock
	currentClient := beadsClient
	beadsClientMu.Unlock()

	// For non-default projects, always use CLI client with project dir
	if projectDir != "" && projectDir != sourceDir {
		cliClient := beads.NewCLIClient(beads.WithWorkDir(projectDir))
		stats, err = cliClient.Stats()
	} else if currentClient != nil && socketExists {
		stats, err = currentClient.Stats()
		if err != nil {
			// Fallback to CLI on RPC error
			stats, err = beads.FallbackStats(workDir)
		}
	} else {
		stats, err = beads.FallbackStats(workDir)
	}

	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	entry.stats = stats
	entry.statsFetchedAt = time.Now()
	c.mu.Unlock()

	return stats, nil
}

// getReadyIssues returns cached ready issues or fetches fresh if stale.
// projectDir specifies which project's beads to query. Empty string uses default.
func (c *beadsStatsCache) getReadyIssues(projectDir string) ([]beads.Issue, error) {
	entry := c.getOrCreateEntry(projectDir)

	c.mu.RLock()
	if entry.readyIssues != nil && time.Since(entry.readyFetchedAt) < c.readyTTL {
		result := entry.readyIssues
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	// Determine the directory to use
	workDir := projectDir
	if workDir == "" {
		workDir = sourceDir
	}

	// Fetch fresh ready issues
	var issues []beads.Issue
	var err error

	// Check if socket exists before attempting RPC to avoid slow timeout on dead daemon.
	// This happens when daemon crashes but server keeps stale connection reference.
	socketPath, findErr := beads.FindSocketPath(workDir)
	socketExists := findErr == nil && socketPath != ""
	if socketExists {
		if _, statErr := os.Stat(socketPath); statErr != nil {
			socketExists = false
		}
	}

	// Thread-safe cleanup of stale beadsClient when socket disappears.
	// This prevents holding broken connection state when daemon restarts.
	beadsClientMu.Lock()
	if !socketExists && beadsClient != nil {
		beadsClient.Close()
		beadsClient = nil
	}

	// Reinitialize beadsClient if socket reappears and client is nil.
	// This handles daemon restarts gracefully without server restart.
	if socketExists && beadsClient == nil && socketPath != "" {
		beadsClient = beads.NewClient(socketPath,
			beads.WithAutoReconnect(3),
			beads.WithTimeout(5*time.Second),
		)
		// Don't block on connection - let execute() handle reconnect
	}

	// Capture client reference under lock for use after unlock
	currentClient := beadsClient
	beadsClientMu.Unlock()

	// For non-default projects, always use CLI client with project dir
	if projectDir != "" && projectDir != sourceDir {
		cliClient := beads.NewCLIClient(beads.WithWorkDir(projectDir))
		issues, err = cliClient.Ready(nil)
	} else if currentClient != nil && socketExists {
		issues, err = currentClient.Ready(nil)
		if err != nil {
			// Fallback to CLI on RPC error
			issues, err = beads.FallbackReady(workDir)
		}
	} else {
		issues, err = beads.FallbackReady(workDir)
	}

	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	entry.readyIssues = issues
	entry.readyFetchedAt = time.Now()
	c.mu.Unlock()

	return issues, nil
}

// getReviewQueueItems returns cached unverified work items or fetches fresh if stale.
// Uses verify.ListUnverifiedWorkWithDir() as the canonical source of truth —
// the same source the daemon uses to seed its verification counter.
// This ensures the review queue count matches the header's "to review" count.
func (c *beadsStatsCache) getReviewQueueItems(projectDir string) ([]verify.UnverifiedItem, error) {
	entry := c.getOrCreateEntry(projectDir)

	c.mu.RLock()
	if entry.reviewQueueItems != nil && time.Since(entry.reviewFetchedAt) < c.reviewTTL {
		result := entry.reviewQueueItems
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	// Fetch fresh unverified work items from the canonical source
	items, err := verify.ListUnverifiedWorkWithDir(projectDir)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []verify.UnverifiedItem{}
	}

	c.mu.Lock()
	entry.reviewQueueItems = items
	entry.reviewFetchedAt = time.Now()
	c.mu.Unlock()

	return items, nil
}

// getGraphIssues returns cached graph issues or fetches fresh if stale.
// projectDir specifies which project's beads to query. Empty string uses default.
func (c *beadsStatsCache) getGraphIssues(projectDir string) ([]beads.Issue, error) {
	entry := c.getOrCreateEntry(projectDir)

	c.mu.RLock()
	if entry.graphIssues != nil && time.Since(entry.graphFetchedAt) < c.graphTTL {
		result := entry.graphIssues
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	// Determine the directory to use
	workDir := projectDir
	if workDir == "" {
		workDir = sourceDir
	}

	// Fetch fresh graph issues (open + in_progress)
	var issues []beads.Issue
	var err error

	// Check if socket exists before attempting RPC to avoid slow timeout on dead daemon.
	socketPath, findErr := beads.FindSocketPath(workDir)
	socketExists := findErr == nil && socketPath != ""
	if socketExists {
		if _, statErr := os.Stat(socketPath); statErr != nil {
			socketExists = false
		}
	}

	// Thread-safe cleanup of stale beadsClient when socket disappears.
	beadsClientMu.Lock()
	if !socketExists && beadsClient != nil {
		beadsClient.Close()
		beadsClient = nil
	}

	// Reinitialize beadsClient if socket reappears and client is nil.
	if socketExists && beadsClient == nil && socketPath != "" {
		beadsClient = beads.NewClient(socketPath,
			beads.WithAutoReconnect(3),
			beads.WithTimeout(5*time.Second),
		)
		// Don't block on connection - let execute() handle reconnect
	}

	// Capture client reference under lock for use after unlock
	currentClient := beadsClient
	beadsClientMu.Unlock()

	// For non-default projects, always use CLI client with project dir
	if projectDir != "" && projectDir != sourceDir {
		cliClient := beads.NewCLIClient(beads.WithWorkDir(projectDir))
		// List open and in_progress issues
		var openIssues, inProgressIssues []beads.Issue
		openIssues, err = cliClient.List(&beads.ListArgs{Status: "open"})
		if err != nil {
			return nil, err
		}
		inProgressIssues, err = cliClient.List(&beads.ListArgs{Status: "in_progress"})
		if err != nil {
			return nil, err
		}
		issues = append(openIssues, inProgressIssues...)
	} else if currentClient != nil && socketExists {
		// List open and in_progress issues via RPC
		openIssues, err := currentClient.List(&beads.ListArgs{Status: "open"})
		if err != nil {
			// Fallback to CLI on RPC error
			openIssues, err = beads.FallbackList("open", workDir)
			if err != nil {
				return nil, err
			}
		}
		inProgressIssues, err := currentClient.List(&beads.ListArgs{Status: "in_progress"})
		if err != nil {
			// Fallback to CLI on RPC error
			inProgressIssues, err = beads.FallbackList("in_progress", workDir)
			if err != nil {
				return nil, err
			}
		}
		issues = append(openIssues, inProgressIssues...)
	} else {
		// CLI fallback
		openIssues, err := beads.FallbackList("open", workDir)
		if err != nil {
			return nil, err
		}
		inProgressIssues, err := beads.FallbackList("in_progress", workDir)
		if err != nil {
			return nil, err
		}
		issues = append(openIssues, inProgressIssues...)
	}

	c.mu.Lock()
	entry.graphIssues = issues
	entry.graphFetchedAt = time.Now()
	c.mu.Unlock()

	return issues, nil
}

// invalidate clears cached data, forcing fresh fetches on next request.
// If projectDir is empty, clears all projects.
func (c *beadsStatsCache) invalidate(projectDir string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if projectDir == "" {
		// Clear all
		c.projects = make(map[string]*projectCacheEntry)
	} else {
		delete(c.projects, projectDir)
	}
}
