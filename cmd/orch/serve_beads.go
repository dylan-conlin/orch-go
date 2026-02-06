package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/kb"
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

// hasLabel checks if a label slice contains a specific label (case-insensitive).
func hasLabel(labels []string, label string) bool {
	for _, l := range labels {
		if strings.EqualFold(l, label) {
			return true
		}
	}
	return false
}

// filterTriageReadyIssues returns only issues that have the triage:ready label.
// This matches the daemon behavior which only spawns issues with this label.
func filterTriageReadyIssues(issues []ReadyIssueResponse) []ReadyIssueResponse {
	result := make([]ReadyIssueResponse, 0, len(issues))
	for _, issue := range issues {
		if hasLabel(issue.Labels, "triage:ready") {
			result = append(result, issue)
		}
	}
	return result
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

	// Filter to only include issues with triage:ready label.
	// The daemon only spawns issues with this label, so showing all ready issues
	// in the "queued" section is misleading - it makes issues appear "stuck"
	// when they were never going to be spawned.
	readyIssues := make([]ReadyIssueResponse, 0, len(issues))
	for _, issue := range issues {
		if !hasLabel(issue.Labels, "triage:ready") {
			continue
		}
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

// GraphNode represents a node in the decidability graph.
// Can be a beads issue or a kb artifact (investigation/decision).
type GraphNode struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Type        string `json:"type"`                  // beads: task, bug, feature, epic, question; kb: investigation, decision
	Status      string `json:"status"`                // open, in_progress, closed, blocked, Complete, Accepted, etc.
	Priority    int    `json:"priority"`              // 0-4 for beads, 0 for kb artifacts
	Source      string `json:"source"`                // "beads" or "kb"
	Date        string `json:"date,omitempty"`        // for kb artifacts
	CreatedAt   string `json:"created_at,omitempty"`  // creation timestamp
	Description string `json:"description,omitempty"` // issue description
	Layer       int    `json:"layer"`                 // execution layer from topological sort (0 = no blocking deps)
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

// computeLayers assigns execution layers to nodes using topological sort.
// Layer 0 contains nodes with no blocking dependencies.
// Layer N contains nodes whose blockers are all in layers 0..N-1.
// Only "blocks" type edges affect layers (not parent-child or references).
// Cycles are assigned to layer 0 (matching CLI behavior).
func computeLayers(nodes []GraphNode, edges []GraphEdge) []GraphNode {
	if len(nodes) == 0 {
		return nodes
	}

	// Build lookup map: id -> index in nodes slice
	nodeIndex := make(map[string]int)
	for i, node := range nodes {
		nodeIndex[node.ID] = i
		nodes[i].Layer = -1 // Mark as unassigned
	}

	// Build dependency map (only "blocks" dependencies)
	// dependsOn[id] = list of IDs that this node depends on (is blocked by)
	dependsOn := make(map[string][]string)
	for _, edge := range edges {
		if edge.Type == "blocks" {
			// edge.From is blocked by edge.To
			// So edge.From depends on edge.To completing first
			dependsOn[edge.From] = append(dependsOn[edge.From], edge.To)
		}
	}

	// Assign layers using longest path from sources
	// Layer 0 = nodes with no dependencies
	changed := true
	for changed {
		changed = false
		for id, idx := range nodeIndex {
			if nodes[idx].Layer >= 0 {
				continue // Already assigned
			}

			deps := dependsOn[id]
			if len(deps) == 0 {
				// No dependencies - layer 0
				nodes[idx].Layer = 0
				changed = true
			} else {
				// Check if all dependencies have layers assigned
				maxDepLayer := -1
				allAssigned := true
				for _, depID := range deps {
					depIdx, exists := nodeIndex[depID]
					if !exists || nodes[depIdx].Layer < 0 {
						allAssigned = false
						break
					}
					if nodes[depIdx].Layer > maxDepLayer {
						maxDepLayer = nodes[depIdx].Layer
					}
				}
				if allAssigned {
					nodes[idx].Layer = maxDepLayer + 1
					changed = true
				}
			}
		}
	}

	// Handle any unassigned nodes (cycles or dependencies not in graph)
	for i := range nodes {
		if nodes[i].Layer < 0 {
			nodes[i].Layer = 0
		}
	}

	return nodes
}

// beadsIssue is the parsed structure from bd list --json
type beadsIssue struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Status          string `json:"status"`
	Priority        int    `json:"priority"`
	IssueType       string `json:"issue_type"`
	Description     string `json:"description,omitempty"`
	CreatedAt       string `json:"created_at,omitempty"`
	DependencyCount int    `json:"dependency_count"`
	DependentCount  int    `json:"dependent_count"`
	Parent          string `json:"parent,omitempty"`
}

// beadsShowIssue is the parsed structure from bd show --json
type beadsShowIssue struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description,omitempty"`
	Status       string `json:"status"`
	Priority     int    `json:"priority"`
	IssueType    string `json:"issue_type"`
	CreatedAt    string `json:"created_at,omitempty"`
	Dependencies []struct {
		ID             string `json:"id"`
		DependencyType string `json:"dependency_type"`
	} `json:"dependencies"`
	Dependents []struct {
		ID             string `json:"id"`
		DependencyType string `json:"dependency_type"`
	} `json:"dependents"`
}

// handleBeadsGraph returns the dependency graph for visualization.
// Query params:
//   - project_dir: Optional project directory to query. If not provided, uses default.
//   - scope: "focus" (default) shows active working set, "open" shows all open issues
//
// Focus scope includes:
//   - All in_progress issues (the active work)
//   - Their blockers (what's preventing completion)
//   - Their immediate dependents (what's waiting)
//   - Any P0/P1 issues (urgent items regardless of status)
func handleBeadsGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query params
	projectDir := r.URL.Query().Get("project_dir")
	scope := r.URL.Query().Get("scope")
	parentID := r.URL.Query().Get("parent")
	if scope == "" {
		scope = "focus" // Default to focus view - the useful working set
	}

	// Determine the directory to use
	workDir := projectDir
	if workDir == "" {
		workDir = beads.DefaultDir
	}

	var nodes []GraphNode
	var edges []GraphEdge
	var err error

	if scope == "focus" {
		nodes, edges, err = buildFocusGraph(workDir)
	} else {
		// scope=open or scope=all
		includeAll := scope == "all"
		nodes, edges, err = buildFullGraph(workDir, includeAll)
	}

	if err != nil {
		resp := BeadsGraphAPIResponse{
			Nodes:      []GraphNode{},
			Edges:      []GraphEdge{},
			ProjectDir: projectDir,
			Error:      err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Filter to parent and descendants if parent ID specified
	if parentID != "" {
		nodes, edges = filterToParentAndDescendants(nodes, edges, parentID)
	}

	// Compute execution layers for all nodes
	nodes = computeLayers(nodes, edges)
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

// filterToParentAndDescendants filters nodes and edges to only include
// the specified parent issue and all its descendants (children, grandchildren, etc.)
// Uses the beads ID hierarchy pattern: orch-go-123 is parent of orch-go-123.1, etc.
func filterToParentAndDescendants(nodes []GraphNode, edges []GraphEdge, parentID string) ([]GraphNode, []GraphEdge) {
	// Build a set of IDs that match: the parent itself and any ID that starts with parent + "."
	isDescendant := func(id string) bool {
		if id == parentID {
			return true
		}
		// Check if id is a child (starts with parentID + ".")
		return strings.HasPrefix(id, parentID+".")
	}

	// Filter nodes
	filteredNodes := make([]GraphNode, 0)
	nodeIDs := make(map[string]bool)
	for _, node := range nodes {
		if isDescendant(node.ID) {
			filteredNodes = append(filteredNodes, node)
			nodeIDs[node.ID] = true
		}
	}

	// Filter edges to only include those where both endpoints are in the filtered set
	filteredEdges := make([]GraphEdge, 0)
	for _, edge := range edges {
		if nodeIDs[edge.From] && nodeIDs[edge.To] {
			filteredEdges = append(filteredEdges, edge)
		}
	}

	return filteredNodes, filteredEdges
}

// buildFocusGraph builds a focused graph showing the active working set:
// - in_progress issues
// - their blockers (recursive)
// - their immediate dependents
// - P0/P1 issues
func buildFocusGraph(workDir string) ([]GraphNode, []GraphEdge, error) {
	// First get all open issues to have the full picture
	allIssues, err := listBeadsIssues(workDir, false)
	if err != nil {
		return nil, nil, err
	}

	// Build lookup maps
	issueByID := make(map[string]beadsIssue)
	for _, issue := range allIssues {
		issueByID[issue.ID] = issue
	}

	// Collect the focus set IDs
	focusSet := make(map[string]bool)

	// 1. Add all in_progress issues
	for _, issue := range allIssues {
		if issue.Status == "in_progress" {
			focusSet[issue.ID] = true
		}
	}

	// 2. Add P0/P1 issues (urgent regardless of status)
	for _, issue := range allIssues {
		if issue.Priority <= 1 {
			focusSet[issue.ID] = true
		}
	}

	// 3. For each focus issue, get blockers and dependents
	// We need to call bd show for dependency details
	edges := make([]GraphEdge, 0)
	processedForDeps := make(map[string]bool)

	// Process in_progress issues to get their relationships
	for id := range focusSet {
		if processedForDeps[id] {
			continue
		}
		processedForDeps[id] = true

		showIssue, err := showBeadsIssue(workDir, id)
		if err != nil {
			continue
		}

		// Add blockers (dependencies) to focus set
		for _, dep := range showIssue.Dependencies {
			focusSet[dep.ID] = true
		}

		// Add immediate dependents (things waiting on this)
		for _, dep := range showIssue.Dependents {
			focusSet[dep.ID] = true
		}

		// Build edges with proper types from bd dep list
		deps, depErr := listIssueDependencies(workDir, id)
		if depErr == nil {
			for _, dep := range deps {
				edges = append(edges, GraphEdge{
					From: id,
					To:   dep.ID,
					Type: dep.DependencyType,
				})
			}
		}
		dependents, depErr := listIssueDependents(workDir, id)
		if depErr == nil {
			for _, dep := range dependents {
				edges = append(edges, GraphEdge{
					From: dep.ID,
					To:   id,
					Type: dep.DependencyType,
				})
			}
		}
	}

	// Build nodes for everything in focus set
	nodes := make([]GraphNode, 0, len(focusSet))
	for id := range focusSet {
		if issue, ok := issueByID[id]; ok {
			nodes = append(nodes, GraphNode{
				ID:          issue.ID,
				Title:       issue.Title,
				Type:        issue.IssueType,
				Status:      issue.Status,
				Priority:    issue.Priority,
				Source:      "beads",
				Description: issue.Description,
				CreatedAt:   issue.CreatedAt,
			})
		} else {
			// Issue might be closed but still a blocker - fetch it
			showIssue, err := showBeadsIssue(workDir, id)
			if err == nil {
				nodes = append(nodes, GraphNode{
					ID:          showIssue.ID,
					Title:       showIssue.Title,
					Type:        showIssue.IssueType,
					Status:      showIssue.Status,
					Priority:    showIssue.Priority,
					Source:      "beads",
					Description: showIssue.Description,
					CreatedAt:   showIssue.CreatedAt,
				})
			}
		}
	}

	// 4. Add kb artifacts that reference focus set issues (last 14 days)
	kbDir := filepath.Join(workDir, ".kb")
	kbArtifacts, err := kb.ListRecentArtifacts(kbDir, 14)
	if err == nil {
		for _, artifact := range kbArtifacts {
			// Check if this artifact references any focus set issue
			hasRelevantRef := false
			for _, ref := range artifact.References {
				if focusSet[ref] {
					hasRelevantRef = true
					// Add edge from artifact to referenced issue
					edges = append(edges, GraphEdge{
						From: artifact.ID,
						To:   ref,
						Type: "references",
					})
				}
			}

			// Only include artifact if it references something in focus set
			if hasRelevantRef {
				nodes = append(nodes, GraphNode{
					ID:     artifact.ID,
					Title:  artifact.Title,
					Type:   string(artifact.Type),
					Status: artifact.Status,
					Source: "kb",
					Date:   artifact.Date,
				})
			}
		}
	}

	return nodes, edges, nil
}

// buildFullGraph builds the full graph with optional status filtering
func buildFullGraph(workDir string, includeAll bool) ([]GraphNode, []GraphEdge, error) {
	issues, err := listBeadsIssues(workDir, includeAll)
	if err != nil {
		return nil, nil, err
	}

	// Build nodes
	nodes := make([]GraphNode, 0, len(issues))
	for _, issue := range issues {
		nodes = append(nodes, GraphNode{
			ID:          issue.ID,
			Title:       issue.Title,
			Type:        issue.IssueType,
			Status:      issue.Status,
			Priority:    issue.Priority,
			Source:      "beads",
			Description: issue.Description,
			CreatedAt:   issue.CreatedAt,
		})
	}

	// Collect IDs of issues that have dependencies
	idsWithDeps := make([]string, 0)
	for _, issue := range issues {
		if issue.DependencyCount > 0 {
			idsWithDeps = append(idsWithDeps, issue.ID)
		}
	}

	// Fetch dependencies with proper types
	edges := make([]GraphEdge, 0)
	for _, id := range idsWithDeps {
		deps, err := listIssueDependencies(workDir, id)
		if err != nil {
			continue
		}
		for _, dep := range deps {
			edges = append(edges, GraphEdge{
				From: id,
				To:   dep.ID,
				Type: dep.DependencyType,
			})
		}
	}

	return nodes, edges, nil
}

// listBeadsIssues calls bd list and returns parsed issues
func listBeadsIssues(workDir string, includeAll bool) ([]beadsIssue, error) {
	args := []string{"list", "--json", "--limit", "0"}
	if includeAll {
		args = append(args, "--all")
	} else {
		// Include both open and in_progress issues (active work)
		// This ensures in_progress issues appear in the work graph
		args = append(args, "--status", "open", "--status", "in_progress")
	}

	cmd := exec.Command(getBdPath(), args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Env = append(os.Environ(), "BEADS_NO_DAEMON=1")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd list failed: %w", err)
	}

	var issues []beadsIssue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("parse issues: %w", err)
	}

	return issues, nil
}

// showBeadsIssue calls bd show and returns the parsed issue with dependencies
func showBeadsIssue(workDir, id string) (*beadsShowIssue, error) {
	cmd := exec.Command(getBdPath(), "show", id, "--json")
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Env = append(os.Environ(), "BEADS_NO_DAEMON=1")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd show %s failed: %w", id, err)
	}

	// bd show returns an array
	var issues []beadsShowIssue
	if err := json.Unmarshal(output, &issues); err != nil || len(issues) == 0 {
		return nil, fmt.Errorf("parse show output: %w", err)
	}

	return &issues[0], nil
}

// getBdPath returns the resolved bd path or falls back to "bd".
func getBdPath() string {
	if beads.BdPath != "" {
		return beads.BdPath
	}
	return "bd"
}

// listIssueDependencies returns dependencies for an issue with proper types using bd dep list.
// This is more reliable than extracting from bd show which may not populate dependency_type.
func listIssueDependencies(workDir, id string) ([]struct {
	ID             string `json:"id"`
	DependencyType string `json:"dependency_type"`
}, error) {
	cmd := exec.Command(getBdPath(), "dep", "list", id, "--json")
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Env = append(os.Environ(), "BEADS_NO_DAEMON=1")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd dep list %s failed: %w", id, err)
	}

	var deps []struct {
		ID             string `json:"id"`
		DependencyType string `json:"dependency_type"`
	}
	if err := json.Unmarshal(output, &deps); err != nil {
		return nil, fmt.Errorf("parse dep list output: %w", err)
	}

	return deps, nil
}

// listIssueDependents returns dependents for an issue with proper types using bd dep list --direction up.
func listIssueDependents(workDir, id string) ([]struct {
	ID             string `json:"id"`
	DependencyType string `json:"dependency_type"`
}, error) {
	cmd := exec.Command(getBdPath(), "dep", "list", id, "--direction", "up", "--json")
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Env = append(os.Environ(), "BEADS_NO_DAEMON=1")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd dep list --direction up %s failed: %w", id, err)
	}

	var deps []struct {
		ID             string `json:"id"`
		DependencyType string `json:"dependency_type"`
	}
	if err := json.Unmarshal(output, &deps); err != nil {
		return nil, fmt.Errorf("parse dep list dependents output: %w", err)
	}

	return deps, nil
}

// AttemptHistoryEntry represents a single attempt for an issue.
type AttemptHistoryEntry struct {
	AttemptNumber int      `json:"attempt_number"`
	Timestamp     string   `json:"timestamp"`      // ISO 8601 timestamp
	Outcome       string   `json:"outcome"`        // success, failed, died, closed→reopened, in_progress
	Phase         string   `json:"phase"`          // last reported phase (e.g., Complete, Implementing, Planning)
	Artifacts     []string `json:"artifacts"`      // list of artifact paths/names
	WorkspaceName string   `json:"workspace_name"` // workspace directory name for reference
}

// AttemptHistoryAPIResponse is the JSON structure returned by /api/beads/{id}/attempts.
type AttemptHistoryAPIResponse struct {
	BeadsID  string                `json:"beads_id"`
	Attempts []AttemptHistoryEntry `json:"attempts"`
	Count    int                   `json:"count"`
	Error    string                `json:"error,omitempty"`
}

// handleBeadsAttempts returns attempt history for a specific beads issue.
// URL format: /api/beads/{id}/attempts
// Scans all workspaces (including archived) for the given beads ID and collects:
// - Attempt number (chronological order based on spawn time)
// - Timestamp (spawn time)
// - Outcome (derived from workspace state and beads comments)
// - Phase (last reported phase from beads comments)
// - Artifacts (files produced: SYNTHESIS.md, investigations, etc.)
func handleBeadsAttempts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract beads ID from URL path: /api/beads/{id}/attempts
	pathPrefix := "/api/beads/"
	pathSuffix := "/attempts"
	if !strings.HasPrefix(r.URL.Path, pathPrefix) || !strings.HasSuffix(r.URL.Path, pathSuffix) {
		resp := AttemptHistoryAPIResponse{Error: "Invalid URL format"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	beadsID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, pathPrefix), pathSuffix)
	if beadsID == "" {
		resp := AttemptHistoryAPIResponse{Error: "Missing beads ID"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Scan workspaces for this beads ID
	attempts, err := collectAttemptHistory(beadsID)
	if err != nil {
		resp := AttemptHistoryAPIResponse{
			BeadsID: beadsID,
			Error:   fmt.Sprintf("Failed to collect attempt history: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := AttemptHistoryAPIResponse{
		BeadsID:  beadsID,
		Attempts: attempts,
		Count:    len(attempts),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode attempt history: %v", err), http.StatusInternalServerError)
		return
	}
}

// collectAttemptHistory scans all workspaces (including archived) for a given beads ID
// and builds the attempt history.
func collectAttemptHistory(beadsID string) ([]AttemptHistoryEntry, error) {
	workspaceDir := filepath.Join(sourceDir, ".orch", "workspace")
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		return []AttemptHistoryEntry{}, nil // No workspaces exist yet
	}

	// Collect all workspace paths for this beads ID (including archived)
	type workspaceInfo struct {
		path      string
		spawnTime time.Time
	}
	var workspaces []workspaceInfo

	// Scan active workspaces
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip archived directory itself
		if entry.Name() == "archived" {
			continue
		}

		dirPath := filepath.Join(workspaceDir, entry.Name())

		// Check if this workspace belongs to our beads ID
		beadsIDFile := filepath.Join(dirPath, ".beads_id")
		beadsIDData, err := os.ReadFile(beadsIDFile)
		if err != nil {
			continue // No beads ID file
		}

		wsBeadsID := strings.TrimSpace(string(beadsIDData))
		if wsBeadsID != beadsID {
			continue // Different beads ID
		}

		// Get spawn time
		spawnTimeFile := filepath.Join(dirPath, ".spawn_time")
		spawnTimeData, err := os.ReadFile(spawnTimeFile)
		if err != nil {
			continue // No spawn time
		}

		var spawnTimeNs int64
		if _, err := fmt.Sscanf(string(spawnTimeData), "%d", &spawnTimeNs); err != nil {
			continue // Invalid spawn time
		}
		spawnTime := time.Unix(0, spawnTimeNs)

		workspaces = append(workspaces, workspaceInfo{
			path:      dirPath,
			spawnTime: spawnTime,
		})
	}

	// Scan archived workspaces
	archivedDir := filepath.Join(workspaceDir, "archived")
	if archivedEntries, err := os.ReadDir(archivedDir); err == nil {
		for _, entry := range archivedEntries {
			if !entry.IsDir() {
				continue
			}

			dirPath := filepath.Join(archivedDir, entry.Name())

			// Check if this workspace belongs to our beads ID
			beadsIDFile := filepath.Join(dirPath, ".beads_id")
			beadsIDData, err := os.ReadFile(beadsIDFile)
			if err != nil {
				continue
			}

			wsBeadsID := strings.TrimSpace(string(beadsIDData))
			if wsBeadsID != beadsID {
				continue
			}

			// Get spawn time
			spawnTimeFile := filepath.Join(dirPath, ".spawn_time")
			spawnTimeData, err := os.ReadFile(spawnTimeFile)
			if err != nil {
				continue
			}

			var spawnTimeNs int64
			if _, err := fmt.Sscanf(string(spawnTimeData), "%d", &spawnTimeNs); err != nil {
				continue
			}
			spawnTime := time.Unix(0, spawnTimeNs)

			workspaces = append(workspaces, workspaceInfo{
				path:      dirPath,
				spawnTime: spawnTime,
			})
		}
	}

	// Sort workspaces by spawn time (oldest first) to assign attempt numbers
	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].spawnTime.Before(workspaces[j].spawnTime)
	})

	// Build attempt history entries
	attempts := make([]AttemptHistoryEntry, 0, len(workspaces))
	for attemptNum, ws := range workspaces {
		entry := AttemptHistoryEntry{
			AttemptNumber: attemptNum + 1, // 1-indexed
			Timestamp:     ws.spawnTime.Format(time.RFC3339),
			WorkspaceName: filepath.Base(ws.path),
		}

		// Determine outcome from workspace state
		entry.Outcome = determineOutcome(ws.path)

		// Find artifacts
		entry.Artifacts = findArtifacts(ws.path)

		attempts = append(attempts, entry)
	}

	// Get phase information from beads comments (async to avoid blocking)
	// We'll do this synchronously for now since we're already in a handler goroutine
	if len(attempts) > 0 {
		// Get comments for this issue
		var comments []beads.Comment
		socketPath, err := beads.FindSocketPath(sourceDir)
		if err == nil {
			client := beads.NewClient(socketPath,
				beads.WithAutoReconnect(3),
				beads.WithTimeout(5*time.Second),
			)
			comments, err = client.Comments(beadsID)
			if err != nil {
				// Fallback to CLI
				cmd := exec.Command(getBdPath(), "comments", beadsID, "--json")
				cmd.Dir = sourceDir
				cmd.Env = append(os.Environ(), "BEADS_NO_DAEMON=1")
				if output, cmdErr := cmd.Output(); cmdErr == nil {
					json.Unmarshal(output, &comments)
				}
			}
		} else {
			// Use CLI directly
			cmd := exec.Command(getBdPath(), "comments", beadsID, "--json")
			cmd.Dir = sourceDir
			cmd.Env = append(os.Environ(), "BEADS_NO_DAEMON=1")
			if output, cmdErr := cmd.Output(); cmdErr == nil {
				json.Unmarshal(output, &comments)
			}
		}

		// Match phase comments to attempts based on timestamp proximity
		// Phase comments should occur shortly after spawn time
		for i := range attempts {
			phase := findPhaseForAttempt(attempts[i].Timestamp, comments)
			if phase != "" {
				attempts[i].Phase = phase

				// Refine outcome based on phase
				if strings.EqualFold(phase, "Complete") {
					// If agent reported Phase: Complete, mark as success
					attempts[i].Outcome = "success"
				}
			}
		}
	}

	return attempts, nil
}

// determineOutcome infers the outcome from workspace state.
// Returns: success, failed, died, closed→reopened, in_progress
func determineOutcome(workspacePath string) string {
	// Check for SYNTHESIS.md (successful completion for full tier)
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	hasSynthesis := false
	if info, err := os.Stat(synthesisPath); err == nil && info.Size() > 0 {
		hasSynthesis = true
	}

	// Check tier
	tierFile := filepath.Join(workspacePath, ".tier")
	tier := "full" // default
	if tierData, err := os.ReadFile(tierFile); err == nil {
		tier = strings.TrimSpace(string(tierData))
	}

	// For full tier with SYNTHESIS.md, consider it success
	if tier == "full" && hasSynthesis {
		return "success"
	}

	// For light tier without SYNTHESIS.md requirement, we need to check other signals
	// Light tier completion is indicated by Phase: Complete comment in beads
	// (This will be filled in by the phase matching logic)

	// Check if workspace is archived (suggests completion or abandonment)
	isArchived := strings.Contains(workspacePath, "/archived/")
	if isArchived && !hasSynthesis && tier == "full" {
		// Archived full-tier workspace without SYNTHESIS.md suggests failure or death
		return "died"
	}

	// Default to in_progress - will be refined by phase information
	return "in_progress"
}

// findArtifacts scans the workspace for produced artifacts.
func findArtifacts(workspacePath string) []string {
	artifacts := []string{}

	// Check for SYNTHESIS.md
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	if _, err := os.Stat(synthesisPath); err == nil {
		artifacts = append(artifacts, "SYNTHESIS.md")
	}

	// Check for investigation files in .kb/investigations/
	kbDir := filepath.Join(sourceDir, ".kb", "investigations")
	if entries, err := os.ReadDir(kbDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			// Match investigation files by checking if they reference this workspace
			filePath := filepath.Join(kbDir, entry.Name())
			if content, err := os.ReadFile(filePath); err == nil {
				workspaceName := filepath.Base(workspacePath)
				if strings.Contains(string(content), workspaceName) {
					artifacts = append(artifacts, filepath.Join(".kb/investigations", entry.Name()))
				}
			}
		}
	}

	return artifacts
}

// findPhaseForAttempt finds the phase reported closest to the attempt timestamp.
// Phase comments are typically posted shortly after spawn (within first few minutes).
func findPhaseForAttempt(attemptTimestamp string, comments []beads.Comment) string {
	attemptTime, err := time.Parse(time.RFC3339, attemptTimestamp)
	if err != nil {
		return ""
	}

	// Look for Phase: comments within 2 hours after spawn time
	// The latest phase within this window is the final phase for this attempt
	var latestPhase string
	var latestPhaseTime time.Time

	phaseRegex := regexp.MustCompile(`(?i)Phase:\s*(\w+)`)

	for _, comment := range comments {
		matches := phaseRegex.FindStringSubmatch(comment.Text)
		if len(matches) < 2 {
			continue
		}

		// Parse comment timestamp
		commentTime, err := time.Parse(time.RFC3339, comment.CreatedAt)
		if err != nil {
			continue
		}

		// Check if comment is within window after attempt spawn
		if commentTime.After(attemptTime) && commentTime.Before(attemptTime.Add(2*time.Hour)) {
			if latestPhase == "" || commentTime.After(latestPhaseTime) {
				latestPhase = matches[1]
				latestPhaseTime = commentTime
			}
		}
	}

	return latestPhase
}

// CloseIssueRequest is the JSON request body for POST /api/beads/close.
type CloseIssueRequest struct {
	ID         string `json:"id"`
	Reason     string `json:"reason,omitempty"`
	ProjectDir string `json:"project_dir,omitempty"`
}

// CloseIssueResponse is the JSON response for POST /api/beads/close.
type CloseIssueResponse struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// handleBeadsClose handles POST /api/beads/close - closes a beads issue.
// This is used by the work graph to close issues via keyboard shortcut.
func handleBeadsClose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req CloseIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp := CloseIssueResponse{Success: false, Error: fmt.Sprintf("Invalid request body: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Validate ID
	if req.ID == "" {
		resp := CloseIssueResponse{Success: false, Error: "Issue ID is required"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Determine the directory to use
	workDir := req.ProjectDir
	if workDir == "" {
		workDir = beads.DefaultDir
	}

	// Use CLI client to close the issue
	// Set BEADS_NO_DAEMON=1 to use direct storage mode, matching the read operations
	// (listBeadsIssues, showBeadsIssue). Without this, close goes through daemon while
	// reads bypass it, causing sync issues where closed items reappear after refresh.
	cliClient := beads.NewCLIClient(
		beads.WithWorkDir(workDir),
		beads.WithEnv(append(os.Environ(), "BEADS_NO_DAEMON=1")),
	)
	if err := cliClient.CloseIssue(req.ID, req.Reason); err != nil {
		resp := CloseIssueResponse{
			ID:      req.ID,
			Success: false,
			Error:   fmt.Sprintf("Failed to close issue: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Invalidate cache to reflect the change
	if globalBeadsStatsCache != nil {
		globalBeadsStatsCache.invalidate(req.ProjectDir)
	}

	resp := CloseIssueResponse{
		ID:      req.ID,
		Success: true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
