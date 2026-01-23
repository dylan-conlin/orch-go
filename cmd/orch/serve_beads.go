package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// projectCacheEntry holds cached data for a single project.
type projectCacheEntry struct {
	stats          *beads.Stats
	statsFetchedAt time.Time

	readyIssues    []beads.Issue
	readyFetchedAt time.Time
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

	// TTL for stats and ready issues
	statsTTL time.Duration
	readyTTL time.Duration
}

// Global beads stats cache, initialized in runServe
var globalBeadsStatsCache *beadsStatsCache

func newBeadsStatsCache() *beadsStatsCache {
	return &beadsStatsCache{
		projects: make(map[string]*projectCacheEntry),
		statsTTL: 30 * time.Second, // Stats change infrequently
		readyTTL: 15 * time.Second, // Ready queue changes more often
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

// GraphNode represents a node (issue) in the dependency graph.
type GraphNode struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Type     string `json:"type"`     // issue_type: task, bug, feature, epic, question
	Status   string `json:"status"`   // open, in_progress, closed, blocked
	Priority int    `json:"priority"` // 0-4
}

// GraphEdge represents an edge (dependency) in the graph.
type GraphEdge struct {
	From string `json:"from"` // ID of the issue that has the dependency
	To   string `json:"to"`   // ID of the issue being depended on
	Type string `json:"type"` // dependency_type: blocks, parent-child, relates_to
}

// BeadsGraphAPIResponse is the JSON structure returned by /api/beads/graph.
type BeadsGraphAPIResponse struct {
	Nodes      []GraphNode `json:"nodes"`
	Edges      []GraphEdge `json:"edges"`
	NodeCount  int         `json:"node_count"`
	EdgeCount  int         `json:"edge_count"`
	ProjectDir string      `json:"project_dir,omitempty"`
	Error      string      `json:"error,omitempty"`
}

// handleBeadsGraph returns the full dependency graph for visualization.
// Returns all issues as nodes and all dependencies as edges.
// Query params:
//   - project_dir: Optional project directory to query. If not provided, uses default.
//   - status: Optional status filter (open, in_progress, closed, all). Default is "all".
func handleBeadsGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query params
	projectDir := r.URL.Query().Get("project_dir")
	statusFilter := r.URL.Query().Get("status")
	if statusFilter == "" {
		statusFilter = "all"
	}

	// Determine the directory to use
	workDir := projectDir
	if workDir == "" {
		workDir = beads.DefaultDir
	}

	// Build command args for bd list
	// Use --all to include closed issues, --limit 0 to get all
	args := []string{"list", "--json", "--limit", "0"}
	if statusFilter != "all" {
		args = append(args, "--status", statusFilter)
	} else {
		args = append(args, "--all") // Include closed issues
	}

	// Get all issues
	cmd := exec.Command(getBdPath(), args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Env = append(os.Environ(), "BEADS_NO_DAEMON=1")

	output, err := cmd.Output()
	if err != nil {
		resp := BeadsGraphAPIResponse{
			Nodes:      []GraphNode{},
			Edges:      []GraphEdge{},
			ProjectDir: projectDir,
			Error:      fmt.Sprintf("Failed to list issues: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Parse the list output
	var listIssues []struct {
		ID              string `json:"id"`
		Title           string `json:"title"`
		Status          string `json:"status"`
		Priority        int    `json:"priority"`
		IssueType       string `json:"issue_type"`
		DependencyCount int    `json:"dependency_count"`
		DependentCount  int    `json:"dependent_count"`
		Parent          string `json:"parent,omitempty"`
	}
	if err := json.Unmarshal(output, &listIssues); err != nil {
		resp := BeadsGraphAPIResponse{
			Nodes:      []GraphNode{},
			Edges:      []GraphEdge{},
			ProjectDir: projectDir,
			Error:      fmt.Sprintf("Failed to parse issues: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Build nodes
	nodes := make([]GraphNode, 0, len(listIssues))
	for _, issue := range listIssues {
		nodes = append(nodes, GraphNode{
			ID:       issue.ID,
			Title:    issue.Title,
			Type:     issue.IssueType,
			Status:   issue.Status,
			Priority: issue.Priority,
		})
	}

	// Collect IDs of issues that have dependencies
	// (we need to call bd show to get the actual dependency details)
	idsWithDeps := make([]string, 0)
	for _, issue := range listIssues {
		if issue.DependencyCount > 0 {
			idsWithDeps = append(idsWithDeps, issue.ID)
		}
	}

	// Fetch dependencies for issues that have them
	edges := make([]GraphEdge, 0)
	for _, id := range idsWithDeps {
		showCmd := exec.Command(getBdPath(), "show", id, "--json")
		if workDir != "" {
			showCmd.Dir = workDir
		}
		showCmd.Env = append(os.Environ(), "BEADS_NO_DAEMON=1")

		showOutput, err := showCmd.Output()
		if err != nil {
			// Log but continue - one failed show shouldn't break the whole graph
			continue
		}

		// bd show returns an array
		var showIssues []struct {
			ID           string `json:"id"`
			Dependencies []struct {
				ID             string `json:"id"`
				DependencyType string `json:"dependency_type"`
			} `json:"dependencies"`
		}
		if err := json.Unmarshal(showOutput, &showIssues); err != nil || len(showIssues) == 0 {
			continue
		}

		// Add edges for each dependency
		for _, dep := range showIssues[0].Dependencies {
			edges = append(edges, GraphEdge{
				From: id,       // This issue depends on...
				To:   dep.ID,   // ...this issue
				Type: dep.DependencyType,
			})
		}
	}

	resp := BeadsGraphAPIResponse{
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

// getBdPath returns the resolved bd path or falls back to "bd".
func getBdPath() string {
	if beads.BdPath != "" {
		return beads.BdPath
	}
	return "bd"
}
