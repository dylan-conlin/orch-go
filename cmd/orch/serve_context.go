package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/cache"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// ContextAPIResponse is the JSON structure returned by /api/context.
// This provides the current orchestrator context for "follow the orchestrator" filtering.
type ContextAPIResponse struct {
	// Cwd is the current working directory of the orchestrator tmux pane
	Cwd string `json:"cwd,omitempty"`
	// ProjectDir is the resolved project directory (contains .beads/ or .orch/)
	ProjectDir string `json:"project_dir,omitempty"`
	// Project is the project name (last segment of project_dir)
	Project string `json:"project,omitempty"`
	// IncludedProjects lists all projects that should be shown (for multi-project configs)
	IncludedProjects []string `json:"included_projects,omitempty"`
	// Error is set if there was a problem getting the context
	Error string `json:"error,omitempty"`
}

// contextCache caches the tmux context polling result to reduce tmux command overhead.
// The cache has a short TTL since the context can change frequently.
type contextCache struct {
	mu         sync.RWMutex
	maxEntries int
	cwd        string
	fetchedAt  time.Time
	ttl        time.Duration
}

const (
	defaultContextCacheTTL        = 1 * time.Second
	defaultContextCacheMaxEntries = 1
)

var globalContextCache = newContextCache(defaultContextCacheMaxEntries, defaultContextCacheTTL)

func newContextCache(maxSize int, ttl time.Duration) *contextCache {
	bounds := cache.NewNamedCache("context cache", maxSize, ttl)

	return &contextCache{
		maxEntries: bounds.MaxSize(),
		ttl:        bounds.TTL(),
	}
}

// getCachedCwd returns cached cwd or fetches fresh.
func (c *contextCache) getCachedCwd() (string, error) {
	c.mu.RLock()
	if c.cwd != "" && time.Since(c.fetchedAt) < c.ttl {
		cwd := c.cwd
		c.mu.RUnlock()
		return cwd, nil
	}
	c.mu.RUnlock()

	// Fetch fresh
	cwd, err := tmux.GetTmuxCwd(tmux.OrchestratorSessionName)
	if err != nil {
		return "", err
	}

	c.mu.Lock()
	c.cwd = cwd
	c.fetchedAt = time.Now()
	c.mu.Unlock()

	return cwd, nil
}

// handleContext returns the current orchestrator context for dashboard filtering.
// This enables "follow the orchestrator" mode where the dashboard auto-filters
// to show only agents from the project the orchestrator is currently working in.
func (s *Server) handleContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := ContextAPIResponse{}

	// Get tmux cwd (cached to reduce command overhead)
	cwd, err := globalContextCache.getCachedCwd()
	if err != nil {
		// Tmux not available or orchestrator session not found
		// Return empty response rather than error - dashboard can handle this gracefully
		resp.Error = err.Error()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp.Cwd = cwd

	// Find project directory (contains .beads/ or .orch/)
	projectDir := findProjectDir(cwd)
	if projectDir != "" {
		resp.ProjectDir = projectDir
		resp.Project = filepath.Base(projectDir)

		// Get included projects (handles multi-project config like orch-go)
		configs := tmux.DefaultMultiProjectConfigs()
		resp.IncludedProjects = tmux.GetIncludedProjects(resp.Project, configs)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode context", http.StatusInternalServerError)
		return
	}
}

// findProjectDir walks up from cwd to find a directory containing .beads/ or .orch/.
// This is a local copy to avoid import cycle with pkg/tmux.
func findProjectDir(cwd string) string {
	// Delegate to tmux package's implementation
	opts := tmux.DefaultFollowerOptions()
	follower := tmux.NewFollower(opts)
	_ = follower // We just need the package to be imported for findProjectDir

	// For now, use a simple inline implementation to avoid export issues
	// This duplicates the logic but avoids circular dependencies
	return findProjectDirInline(cwd)
}

// findProjectDirInline is a local implementation of project directory finding.
func findProjectDirInline(cwd string) string {
	if cwd == "" {
		return ""
	}

	current := cwd
	for {
		// Check for .beads/ directory (beads-managed project)
		beadsPath := filepath.Join(current, ".beads")
		if isDir(beadsPath) {
			return current
		}

		// Check for .orch/ directory (orchestrator workspace)
		orchPath := filepath.Join(current, ".orch")
		if isDir(orchPath) {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached root without finding project
			return ""
		}
		current = parent
	}
}

// isDir checks if a path exists and is a directory.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
