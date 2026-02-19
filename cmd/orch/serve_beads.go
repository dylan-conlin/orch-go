package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/graph"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// projectCacheEntry holds cached data for a single project.
type projectCacheEntry struct {
	stats          *beads.Stats
	statsFetchedAt time.Time

	readyIssues    []beads.Issue
	readyFetchedAt time.Time

	reviewQueueIssues []beads.Issue
	reviewFetchedAt   time.Time

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
		workDir = beads.DefaultDir
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
	if projectDir != "" && projectDir != beads.DefaultDir {
		cliClient := beads.NewCLIClient(beads.WithWorkDir(projectDir))
		stats, err = cliClient.Stats()
	} else if currentClient != nil && socketExists {
		stats, err = currentClient.Stats()
		if err != nil {
			// Fallback to CLI on RPC error
			stats, err = beads.FallbackStats()
		}
	} else {
		stats, err = beads.FallbackStats()
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
		workDir = beads.DefaultDir
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
	if projectDir != "" && projectDir != beads.DefaultDir {
		cliClient := beads.NewCLIClient(beads.WithWorkDir(projectDir))
		issues, err = cliClient.Ready(nil)
	} else if currentClient != nil && socketExists {
		issues, err = currentClient.Ready(nil)
		if err != nil {
			// Fallback to CLI on RPC error
			issues, err = beads.FallbackReady()
		}
	} else {
		issues, err = beads.FallbackReady()
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

// getReviewQueueIssues returns cached review queue issues or fetches fresh if stale.
// Review queue issues are in_progress issues labeled daemon:ready-review.
func (c *beadsStatsCache) getReviewQueueIssues(projectDir string) ([]beads.Issue, error) {
	entry := c.getOrCreateEntry(projectDir)

	c.mu.RLock()
	if entry.reviewQueueIssues != nil && time.Since(entry.reviewFetchedAt) < c.reviewTTL {
		result := entry.reviewQueueIssues
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	// Determine the directory to use
	workDir := projectDir
	if workDir == "" {
		workDir = beads.DefaultDir
	}

	// Fetch fresh review queue issues
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
	}

	// Capture client reference under lock for use after unlock
	currentClient := beadsClient
	beadsClientMu.Unlock()

	listArgs := &beads.ListArgs{
		Status: "in_progress",
		Labels: []string{"daemon:ready-review"},
	}

	// For non-default projects, always use CLI client with project dir
	if projectDir != "" && projectDir != beads.DefaultDir {
		cliClient := beads.NewCLIClient(beads.WithWorkDir(projectDir))
		issues, err = cliClient.List(listArgs)
	} else if currentClient != nil && socketExists {
		issues, err = currentClient.List(listArgs)
		if err != nil {
			// Fallback to CLI on RPC error
			cliClient := beads.NewCLIClient(beads.WithWorkDir(workDir))
			issues, err = cliClient.List(listArgs)
		}
	} else {
		cliClient := beads.NewCLIClient(beads.WithWorkDir(workDir))
		issues, err = cliClient.List(listArgs)
	}

	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	entry.reviewQueueIssues = issues
	entry.reviewFetchedAt = time.Now()
	c.mu.Unlock()

	return issues, nil
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
		workDir = beads.DefaultDir
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
	if projectDir != "" && projectDir != beads.DefaultDir {
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
			openIssues, err = beads.FallbackList("open")
			if err != nil {
				return nil, err
			}
		}
		inProgressIssues, err := currentClient.List(&beads.ListArgs{Status: "in_progress"})
		if err != nil {
			// Fallback to CLI on RPC error
			inProgressIssues, err = beads.FallbackList("in_progress")
			if err != nil {
				return nil, err
			}
		}
		issues = append(openIssues, inProgressIssues...)
	} else {
		// CLI fallback
		openIssues, err := beads.FallbackList("open")
		if err != nil {
			return nil, err
		}
		inProgressIssues, err := beads.FallbackList("in_progress")
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

// BeadsAPIResponse is the JSON structure returned by /api/beads.
type BeadsAPIResponse struct {
	TotalIssues    int     `json:"total_issues"`
	OpenIssues     int     `json:"open_issues"`
	InProgress     int     `json:"in_progress_issues"`
	BlockedIssues  int     `json:"blocked_issues"`
	ReadyIssues    int     `json:"ready_issues"`
	ClosedIssues   int     `json:"closed_issues"`
	AvgLeadTimeHrs float64 `json:"avg_lead_time_hours,omitempty"`
	ProjectDir     string  `json:"project_dir,omitempty"`
	Error          string  `json:"error,omitempty"`
}

// handleBeads returns beads stats using cached data when available.
// The cache has a 30s TTL to balance freshness with performance.
// Without caching, each request spawns a bd process (~1.5s overhead).
// Query params:
//   - project_dir: Optional project directory to query. If not provided, uses default.
func handleBeads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get project_dir from query params (for following orchestrator context)
	projectDir := r.URL.Query().Get("project_dir")

	// Use cached stats when available
	stats, err := globalBeadsStatsCache.getStats(projectDir)
	if err != nil {
		resp := BeadsAPIResponse{
			Error:      fmt.Sprintf("Failed to get bd stats: %v", err),
			ProjectDir: projectDir,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := BeadsAPIResponse{
		TotalIssues:    stats.Summary.TotalIssues,
		OpenIssues:     stats.Summary.OpenIssues,
		InProgress:     stats.Summary.InProgressIssues,
		BlockedIssues:  stats.Summary.BlockedIssues,
		ReadyIssues:    stats.Summary.ReadyIssues,
		ClosedIssues:   stats.Summary.ClosedIssues,
		AvgLeadTimeHrs: stats.Summary.AvgLeadTimeHours,
		ProjectDir:     projectDir,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode beads: %v", err), http.StatusInternalServerError)
		return
	}
}

// ReadyIssueResponse represents a ready issue for the dashboard queue.
type ReadyIssueResponse struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Priority  int      `json:"priority"`
	IssueType string   `json:"issue_type"`
	Labels    []string `json:"labels,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
}

// BeadsReadyAPIResponse is the JSON structure returned by /api/beads/ready.
type BeadsReadyAPIResponse struct {
	Issues     []ReadyIssueResponse `json:"issues"`
	Count      int                  `json:"count"`
	ProjectDir string               `json:"project_dir,omitempty"`
	Error      string               `json:"error,omitempty"`
}

// ReviewQueueIssueResponse represents an issue ready for review.
type ReviewQueueIssueResponse struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Priority  int      `json:"priority"`
	IssueType string   `json:"issue_type"`
	Status    string   `json:"status"`
	Labels    []string `json:"labels,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
	UpdatedAt string   `json:"updated_at,omitempty"`
}

// BeadsReviewQueueResponse is the JSON structure returned by /api/beads/review-queue.
type BeadsReviewQueueResponse struct {
	Issues     []ReviewQueueIssueResponse `json:"issues"`
	Count      int                        `json:"count"`
	ProjectDir string                     `json:"project_dir,omitempty"`
	Error      string                     `json:"error,omitempty"`
}

// handleBeadsReady returns list of ready issues for dashboard queue visibility.
// The cache has a 15s TTL to balance freshness with performance.
// Without caching, each request spawns a bd process (~80ms overhead).
// Query params:
//   - project_dir: Optional project directory to query. If not provided, uses default.
func handleBeadsReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get project_dir from query params (for following orchestrator context)
	projectDir := r.URL.Query().Get("project_dir")

	// Use cached ready issues when available
	issues, err := globalBeadsStatsCache.getReadyIssues(projectDir)
	if err != nil {
		resp := BeadsReadyAPIResponse{
			Issues:     []ReadyIssueResponse{},
			Count:      0,
			ProjectDir: projectDir,
			Error:      fmt.Sprintf("Failed to get ready issues: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Convert beads.Issue to ReadyIssueResponse
	readyIssues := make([]ReadyIssueResponse, 0, len(issues))
	for _, issue := range issues {
		readyIssues = append(readyIssues, ReadyIssueResponse{
			ID:        issue.ID,
			Title:     issue.Title,
			Priority:  issue.Priority,
			IssueType: issue.IssueType,
			Labels:    issue.Labels,
			CreatedAt: issue.CreatedAt,
		})
	}

	resp := BeadsReadyAPIResponse{
		Issues:     readyIssues,
		Count:      len(readyIssues),
		ProjectDir: projectDir,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode beads ready: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleBeadsReviewQueue returns list of issues awaiting review (daemon:ready-review label).
// The cache has a 15s TTL to balance freshness with performance.
// Query params:
//   - project_dir: Optional project directory to query. If not provided, uses default.
func handleBeadsReviewQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectDir := r.URL.Query().Get("project_dir")

	issues, err := globalBeadsStatsCache.getReviewQueueIssues(projectDir)
	if err != nil {
		resp := BeadsReviewQueueResponse{
			Issues:     []ReviewQueueIssueResponse{},
			Count:      0,
			ProjectDir: projectDir,
			Error:      fmt.Sprintf("Failed to get review queue: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	reviewIssues := make([]ReviewQueueIssueResponse, 0, len(issues))
	for _, issue := range issues {
		reviewIssues = append(reviewIssues, ReviewQueueIssueResponse{
			ID:        issue.ID,
			Title:     issue.Title,
			Priority:  issue.Priority,
			IssueType: issue.IssueType,
			Status:    issue.Status,
			Labels:    issue.Labels,
			CreatedAt: issue.CreatedAt,
			UpdatedAt: issue.UpdatedAt,
		})
	}

	resp := BeadsReviewQueueResponse{
		Issues:     reviewIssues,
		Count:      len(reviewIssues),
		ProjectDir: projectDir,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode review queue: %v", err), http.StatusInternalServerError)
		return
	}
}

// CreateIssueRequest is the JSON request body for POST /api/issues.
type CreateIssueRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	IssueType   string   `json:"issue_type,omitempty"` // task, bug, etc.
	Priority    int      `json:"priority,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	ParentID    string   `json:"parent_id,omitempty"` // Optional parent issue for follow-ups
}

// CreateIssueResponse is the JSON response for POST /api/issues.
type CreateIssueResponse struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// handleIssues handles POST /api/issues - creates a new beads issue.
// This is used by the dashboard to create follow-up issues from synthesis recommendations.
func handleIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req CreateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp := CreateIssueResponse{Success: false, Error: fmt.Sprintf("Invalid request body: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Validate title
	if req.Title == "" {
		resp := CreateIssueResponse{Success: false, Error: "Title is required"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Use persistent RPC client (with auto-reconnect), fallback to CLI if unavailable
	var issue *beads.Issue
	var err error

	// Thread-safe access to beadsClient
	beadsClientMu.RLock()
	currentClient := beadsClient
	beadsClientMu.RUnlock()

	if currentClient != nil {
		issue, err = currentClient.Create(&beads.CreateArgs{
			Title:       req.Title,
			Description: req.Description,
			IssueType:   req.IssueType,
			Priority:    req.Priority,
			Labels:      req.Labels,
		})
		if err != nil {
			// Fall through to CLI fallback on RPC error
			issue, err = beads.FallbackCreate(req.Title, req.Description, req.IssueType, req.Priority, req.Labels)
		}
	} else {
		issue, err = beads.FallbackCreate(req.Title, req.Description, req.IssueType, req.Priority, req.Labels)
	}

	if err != nil {
		resp := CreateIssueResponse{Success: false, Error: fmt.Sprintf("Failed to create issue: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := CreateIssueResponse{
		ID:      issue.ID,
		Title:   issue.Title,
		Success: true,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// QuestionResponse represents a question for the dashboard.
type QuestionResponse struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	Priority    int      `json:"priority"`
	Labels      []string `json:"labels,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty"`
	ClosedAt    string   `json:"closed_at,omitempty"`
	CloseReason string   `json:"close_reason,omitempty"`
	Blocking    []string `json:"blocking,omitempty"` // IDs of issues this question blocks
}

// QuestionsAPIResponse is the JSON structure returned by /api/questions.
type QuestionsAPIResponse struct {
	Open          []QuestionResponse `json:"open"`
	Investigating []QuestionResponse `json:"investigating"`
	Answered      []QuestionResponse `json:"answered"`
	TotalCount    int                `json:"total_count"`
	Error         string             `json:"error,omitempty"`
}

// ActiveAgentInfo represents active agent data for graph node enrichment.
// Matches frontend active_agent interface in web/src/lib/stores/work-graph.ts
type ActiveAgentInfo struct {
	Phase   string `json:"phase,omitempty"`
	Runtime string `json:"runtime,omitempty"`
	Model   string `json:"model,omitempty"`
}

// GraphNode represents a node in the work graph (from /api/beads/graph).
// Matches frontend GraphNode interface in web/src/lib/stores/work-graph.ts
type GraphNode struct {
	ID                string           `json:"id"`
	Title             string           `json:"title"`
	Type              string           `json:"type"`     // task, bug, feature, epic, question
	Status            string           `json:"status"`   // open, in_progress, closed, blocked
	Priority          int              `json:"priority"` // 0-4 for beads
	EffectivePriority string           `json:"effective_priority"`
	Source            string           `json:"source"` // "beads"
	CreatedAt         string           `json:"created_at,omitempty"`
	Description       string           `json:"description,omitempty"`
	Labels            []string         `json:"labels,omitempty"`
	Layer             int              `json:"layer"`
	ActiveAgent       *ActiveAgentInfo `json:"active_agent,omitempty"`
}

// GraphEdge represents a dependency edge in the work graph.
// Matches frontend GraphEdge interface in web/src/lib/stores/work-graph.ts
type GraphEdge struct {
	From string `json:"from"` // ID of the issue that has the dependency
	To   string `json:"to"`   // ID of the issue being depended on
	Type string `json:"type"` // dependency_type: blocks, parent-child, relates_to
}

// WorkGraphResponse is the JSON structure returned by /api/beads/graph.
// Matches frontend WorkGraphResponse interface in web/src/lib/stores/work-graph.ts
type WorkGraphResponse struct {
	Nodes      []GraphNode `json:"nodes"`
	Edges      []GraphEdge `json:"edges"`
	NodeCount  int         `json:"node_count"`
	EdgeCount  int         `json:"edge_count"`
	ProjectDir string      `json:"project_dir,omitempty"`
	Error      string      `json:"error,omitempty"`
}

// handleQuestions returns questions grouped by status for the dashboard.
// Questions are issues with type=question.
// Groups:
//   - open: Questions needing answers (status=open)
//   - investigating: Questions with active investigation (status=in_progress)
//   - answered: Recently closed questions (status=closed, last 7 days)
func handleQuestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Fetch questions using CLI client (type=question)
	cliClient := beads.NewCLIClient()

	// Get all questions including closed (recent)
	allQuestions, err := cliClient.List(&beads.ListArgs{
		IssueType: "question",
		Limit:     100, // Reasonable limit for dashboard
	})
	if err != nil {
		resp := QuestionsAPIResponse{
			Open:          []QuestionResponse{},
			Investigating: []QuestionResponse{},
			Answered:      []QuestionResponse{},
			Error:         fmt.Sprintf("Failed to list questions: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Group questions by status
	var open, investigating, answered []QuestionResponse

	// For calculating "recent" answered (last 7 days)
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)

	for _, q := range allQuestions {
		qr := QuestionResponse{
			ID:          q.ID,
			Title:       q.Title,
			Status:      q.Status,
			Priority:    q.Priority,
			Labels:      q.Labels,
			CreatedAt:   q.CreatedAt,
			ClosedAt:    q.ClosedAt,
			CloseReason: q.CloseReason,
		}

		// Get blocking info (what issues this question blocks)
		// Use bd show to get dependents
		if fullIssue, err := cliClient.Show(q.ID); err == nil {
			// Parse dependents from the raw dependencies field
			// bd show returns dependents in the response
			var dependents []struct {
				ID string `json:"id"`
			}
			if fullIssue.Dependencies != nil {
				// Note: bd show puts dependents in the dependencies field for questions
				json.Unmarshal(fullIssue.Dependencies, &dependents)
				for _, dep := range dependents {
					qr.Blocking = append(qr.Blocking, dep.ID)
				}
			}
		}

		switch q.Status {
		case "open":
			open = append(open, qr)
		case "in_progress", "investigating":
			investigating = append(investigating, qr)
		case "closed", "answered":
			// Only include if closed within last 7 days
			if q.ClosedAt != "" {
				closedTime, err := time.Parse(time.RFC3339, q.ClosedAt)
				if err == nil && closedTime.After(sevenDaysAgo) {
					answered = append(answered, qr)
				}
			} else {
				// No closed_at but status is closed - include anyway
				answered = append(answered, qr)
			}
		}
	}

	// Return empty slices instead of nil for cleaner JSON
	if open == nil {
		open = []QuestionResponse{}
	}
	if investigating == nil {
		investigating = []QuestionResponse{}
	}
	if answered == nil {
		answered = []QuestionResponse{}
	}

	resp := QuestionsAPIResponse{
		Open:          open,
		Investigating: investigating,
		Answered:      answered,
		TotalCount:    len(open) + len(investigating) + len(answered),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode questions: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleBeadsGraph returns work graph nodes and edges for dependency visualization.
// The cache has a 15s TTL to balance freshness with performance.
// Without caching, each request spawns multiple bd processes (list + show for dependencies).
// Query params:
//   - project_dir: Optional project directory to query. If not provided, uses default.
//   - scope: Optional scope filter (currently not used, reserved for future filtering)
//   - parent: Optional parent issue ID filter (reserved for future epic filtering)
func handleBeadsGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query params
	projectDir := r.URL.Query().Get("project_dir")
	// scope := r.URL.Query().Get("scope")     // Reserved for future use
	// parent := r.URL.Query().Get("parent")   // Reserved for future use

	// Use cached graph issues when available
	issues, err := globalBeadsStatsCache.getGraphIssues(projectDir)
	if err != nil {
		resp := WorkGraphResponse{
			Nodes:      []GraphNode{},
			Edges:      []GraphEdge{},
			NodeCount:  0,
			EdgeCount:  0,
			ProjectDir: projectDir,
			Error:      fmt.Sprintf("Failed to get graph issues: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Build nodes from issues
	nodes := make([]GraphNode, 0, len(issues))
	for _, issue := range issues {
		nodes = append(nodes, GraphNode{
			ID:          issue.ID,
			Title:       issue.Title,
			Type:        issue.IssueType,
			Status:      issue.Status,
			Priority:    issue.Priority,
			Source:      "beads",
			CreatedAt:   issue.CreatedAt,
			Description: issue.Description,
			Labels:      issue.Labels,
		})
	}

	// Build edges from dependencies
	edges := make([]GraphEdge, 0)
	for _, issue := range issues {
		// Parse dependencies from the raw JSON
		deps := issue.ParseDependencies()
		if deps == nil {
			continue
		}

		// Create edges for each dependency
		for _, dep := range deps {
			depID := dep.EffectiveID()
			if depID == "" {
				continue // Skip edges with no target
			}
			edge := GraphEdge{
				From: issue.ID,
				To:   depID,
				Type: dep.EffectiveType(),
			}
			// Default to "blocks" if no type specified
			if edge.Type == "" {
				edge.Type = "blocks"
			}
			edges = append(edges, edge)
		}
	}

	// Compute effective priority and topological layers
	nodeInputs := make([]graph.Node, 0, len(nodes))
	for _, node := range nodes {
		nodeInputs = append(nodeInputs, graph.Node{ID: node.ID, Priority: node.Priority})
	}
	edgeInputs := make([]graph.Edge, 0, len(edges))
	for _, edge := range edges {
		edgeInputs = append(edgeInputs, graph.Edge{From: edge.From, To: edge.To, Type: edge.Type})
	}
	priorityByID := graph.ComputeEffectivePriority(nodeInputs, edgeInputs)
	layersByID := graph.ComputeLayers(nodeInputs, edgeInputs)
	for i := range nodes {
		if eff, ok := priorityByID[nodes[i].ID]; ok {
			nodes[i].EffectivePriority = strconv.Itoa(eff)
		} else {
			nodes[i].EffectivePriority = strconv.Itoa(nodes[i].Priority)
		}
		if layer, ok := layersByID[nodes[i].ID]; ok {
			nodes[i].Layer = layer
		}
	}

	// Enrich nodes with active agent data for in-progress issues.
	// This enables the frontend to show which agent is working on each issue
	// via the active_agent field (phase, runtime, model).
	activeAgentMap := buildActiveAgentMap()
	for i := range nodes {
		if info, ok := activeAgentMap[nodes[i].ID]; ok {
			nodes[i].ActiveAgent = info
		}
	}

	resp := WorkGraphResponse{
		Nodes:      nodes,
		Edges:      edges,
		NodeCount:  len(nodes),
		EdgeCount:  len(edges),
		ProjectDir: projectDir,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode graph: %v", err), http.StatusInternalServerError)
		return
	}
}

// buildActiveAgentMap returns a map of beads_id -> active agent info.
// Queries OpenCode sessions and tmux windows for running agents,
// then enriches with phase from beads comments.
func buildActiveAgentMap() map[string]*ActiveAgentInfo {
	result := make(map[string]*ActiveAgentInfo)
	now := time.Now()

	// 1. OpenCode sessions (primary agent source)
	client := opencode.NewClient(serverURL)
	sessions, err := client.ListSessions("")
	if err == nil {
		for _, s := range sessions {
			beadsID := extractBeadsIDFromTitle(s.Title)
			if beadsID == "" {
				continue
			}
			createdAt := time.Unix(s.Time.Created/1000, 0)
			result[beadsID] = &ActiveAgentInfo{
				Runtime: formatDuration(now.Sub(createdAt)),
			}
		}
	}

	// 2. Tmux windows (catches Claude CLI agents not in OpenCode)
	workersSessions, _ := tmux.ListWorkersSessions()
	for _, sessionName := range workersSessions {
		windows, _ := tmux.ListWindows(sessionName)
		for _, win := range windows {
			if win.Name == "servers" || win.Name == "zsh" {
				continue
			}
			beadsID := extractBeadsIDFromWindowName(win.Name)
			if beadsID != "" && result[beadsID] == nil {
				result[beadsID] = &ActiveAgentInfo{}
			}
		}
	}

	// 3. Enrich with phase from beads comments
	beadsIDs := make([]string, 0, len(result))
	for id := range result {
		beadsIDs = append(beadsIDs, id)
	}
	if len(beadsIDs) > 0 {
		commentsMap := globalBeadsCache.getComments(beadsIDs, nil)
		for id, comments := range commentsMap {
			if info, ok := result[id]; ok {
				phaseStatus := verify.ParsePhaseFromComments(comments)
				if phaseStatus.Found {
					info.Phase = phaseStatus.Phase
				}
			}
		}
	}

	return result
}
