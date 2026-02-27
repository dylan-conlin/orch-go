package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

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
	mu        sync.RWMutex
	cwd       string
	fetchedAt time.Time
	ttl       time.Duration
}

var globalContextCache = &contextCache{
	ttl: 1 * time.Second, // Short TTL - context can change frequently
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

// invalidate forces the next getCachedCwd to fetch fresh data.
func (c *contextCache) invalidate() {
	c.mu.Lock()
	c.fetchedAt = time.Time{} // Zero time = always stale
	c.mu.Unlock()
}

// --- Context SSE Broadcaster ---
// Pushes context changes to connected dashboard clients in real-time,
// eliminating the need for frequent polling.

// contextBroadcaster manages SSE clients subscribed to context changes.
type contextBroadcaster struct {
	mu      sync.RWMutex
	clients map[chan ContextAPIResponse]struct{}
}

var globalContextBroadcaster = &contextBroadcaster{
	clients: make(map[chan ContextAPIResponse]struct{}),
}

// subscribe registers a client channel for context change events.
func (b *contextBroadcaster) subscribe() chan ContextAPIResponse {
	ch := make(chan ContextAPIResponse, 1) // Buffered to prevent blocking broadcaster
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// unsubscribe removes a client channel.
func (b *contextBroadcaster) unsubscribe(ch chan ContextAPIResponse) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
	close(ch)
}

// broadcast sends a context change to all connected clients.
func (b *contextBroadcaster) broadcast(ctx ContextAPIResponse) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		// Non-blocking send: drop if client is behind
		select {
		case ch <- ctx:
		default:
		}
	}
}

// --- Tmux Follower Integration ---
// The follower polls tmux at 500ms and pushes context changes via SSE.

var globalContextFollower *tmux.FollowerState

// startContextFollower starts the tmux follower that detects project changes
// and pushes them to connected SSE clients.
func startContextFollower() {
	opts := tmux.DefaultFollowerOptions()
	// Reduce stability threshold to 1 for faster detection.
	// The old threshold of 2 added 500ms latency to prevent "flicker",
	// but with SSE push the frontend handles rapid changes gracefully.
	opts.StabilityThreshold = 1

	globalContextFollower = tmux.NewFollower(opts)

	globalContextFollower.SetOnChange(func(event tmux.ProjectChangeEvent) {
		// Invalidate the cache so GET /api/context returns fresh data immediately
		globalContextCache.invalidate()

		// Build the context response
		resp := buildContextResponse(event.Cwd, event.ProjectDir)

		// Push to all connected SSE clients
		globalContextBroadcaster.broadcast(resp)

		fmt.Printf("[context-follower] Project changed: %s → %s\n",
			filepath.Base(event.PrevDir), filepath.Base(event.ProjectDir))
	})

	globalContextFollower.SetOnError(func(err error) {
		// Silently ignore errors - tmux may not be available
	})

	globalContextFollower.Start()
	fmt.Println("Started context follower (polling tmux every 500ms, SSE push enabled)")
}

// buildContextResponse creates a ContextAPIResponse from cwd and projectDir.
func buildContextResponse(cwd, projectDir string) ContextAPIResponse {
	resp := ContextAPIResponse{
		Cwd:        cwd,
		ProjectDir: projectDir,
	}
	if projectDir != "" {
		resp.Project = filepath.Base(projectDir)
		configs := tmux.DefaultMultiProjectConfigs()
		resp.IncludedProjects = tmux.GetIncludedProjects(resp.Project, configs)
	}
	return resp
}

// --- HTTP Handlers ---

// handleContext returns the current orchestrator context for dashboard filtering.
// This enables "follow the orchestrator" mode where the dashboard auto-filters
// to show only agents from the project the orchestrator is currently working in.
func handleContext(w http.ResponseWriter, r *http.Request) {
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

// handleContextNotify accepts a POST to force an immediate context refresh.
// Called by the tmux after-select-window hook for instant notification.
// This bypasses the 500ms tmux polling interval.
func handleContextNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Invalidate cache to force fresh tmux query
	globalContextCache.invalidate()

	// Fetch fresh context
	cwd, err := tmux.GetTmuxCwd(tmux.OrchestratorSessionName)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
		return
	}

	// Update cache with fresh data
	globalContextCache.mu.Lock()
	globalContextCache.cwd = cwd
	globalContextCache.fetchedAt = time.Now()
	globalContextCache.mu.Unlock()

	// Build and broadcast the context
	projectDir := findProjectDirInline(cwd)
	resp := buildContextResponse(cwd, projectDir)
	globalContextBroadcaster.broadcast(resp)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"project": resp.Project,
	})
}

// handleContextEvents streams context changes via SSE.
// Clients subscribe and receive real-time notifications when the orchestrator
// switches tmux windows (and thus projects).
func handleContextEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Subscribe to context changes
	ch := globalContextBroadcaster.subscribe()
	defer globalContextBroadcaster.unsubscribe(ch)

	// Send the current context immediately so the client has initial state
	cwd, err := globalContextCache.getCachedCwd()
	if err == nil {
		projectDir := findProjectDirInline(cwd)
		initialCtx := buildContextResponse(cwd, projectDir)
		data, _ := json.Marshal(initialCtx)
		fmt.Fprintf(w, "event: context.changed\ndata: %s\n\n", data)
		flusher.Flush()
	}

	ctx := r.Context()

	// Stream context changes to client
	for {
		select {
		case <-ctx.Done():
			return
		case ctxResp, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(ctxResp)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: context.changed\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// findProjectDir walks up from cwd to find a directory containing .beads/ or .orch/.
// This is a local copy to avoid import cycle with pkg/tmux.
func findProjectDir(cwd string) string {
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
