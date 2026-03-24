package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/discovery"
	"github.com/dylan-conlin/orch-go/pkg/graph"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

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

// ReviewQueueIssueResponse represents a completion awaiting verification review.
type ReviewQueueIssueResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	IssueType string `json:"issue_type"`
	Tier      int    `json:"tier"`  // 1=feature/bug, 2=investigation, 3=task
	Gate1     bool   `json:"gate1"` // Comprehension gate passed
	Gate2     bool   `json:"gate2"` // Behavioral gate passed
	HasBrief  bool   `json:"has_brief"` // True if .kb/briefs/{id}.md exists
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

// handleBeadsReviewQueue returns completions awaiting verification review.
// Uses verify.ListUnverifiedWork() — the same canonical source the daemon uses
// to seed its verification counter. This ensures the review queue count matches
// the header's "to review" count.
// The cache has a 15s TTL to balance freshness with performance.
// Query params:
//   - project_dir: Optional project directory to query. If not provided, uses default.
func handleBeadsReviewQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectDir := r.URL.Query().Get("project_dir")

	items, err := globalBeadsStatsCache.getReviewQueueItems(projectDir)
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

	reviewIssues := make([]ReviewQueueIssueResponse, 0, len(items))
	for _, item := range items {
		reviewIssues = append(reviewIssues, ReviewQueueIssueResponse{
			ID:        item.BeadsID,
			Title:     item.Title,
			IssueType: item.IssueType,
			Tier:      item.Tier,
			Gate1:     item.Gate1,
			Gate2:     item.Gate2,
			HasBrief:  hasBriefFile(item.BeadsID),
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
			issue, err = beads.FallbackCreate(req.Title, req.Description, req.IssueType, req.Priority, req.Labels, sourceDir)
		}
	} else {
		issue, err = beads.FallbackCreate(req.Title, req.Description, req.IssueType, req.Priority, req.Labels, sourceDir)
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
	EffectivePriority int              `json:"effective_priority"`
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
		Limit:     beads.IntPtr(100), // Reasonable limit for dashboard
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
			nodes[i].EffectivePriority = eff
		} else {
			nodes[i].EffectivePriority = nodes[i].Priority
		}
		if layer, ok := layersByID[nodes[i].ID]; ok {
			nodes[i].Layer = layer
		}
	}

	// Enrich nodes with active agent data for in-progress issues.
	// This enables the frontend to show which agent is working on each issue
	// via the active_agent field (phase, runtime, model).
	activeAgentMap := getCachedActiveAgentMap()
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

// activeAgentMapCache caches the result of buildActiveAgentMap with a short TTL.
// This avoids redundant OpenCode HTTP calls and beads comment fetches on every
// /api/beads/graph request.
var activeAgentMapCache struct {
	mu        sync.RWMutex
	data      map[string]*ActiveAgentInfo
	fetchedAt time.Time
	ttl       time.Duration
}

func init() {
	activeAgentMapCache.ttl = 15 * time.Second
}

// getCachedActiveAgentMap returns cached active agent data or rebuilds if stale.
func getCachedActiveAgentMap() map[string]*ActiveAgentInfo {
	activeAgentMapCache.mu.RLock()
	if activeAgentMapCache.data != nil && time.Since(activeAgentMapCache.fetchedAt) < activeAgentMapCache.ttl {
		result := activeAgentMapCache.data
		activeAgentMapCache.mu.RUnlock()
		return result
	}
	activeAgentMapCache.mu.RUnlock()

	data := buildActiveAgentMap()

	activeAgentMapCache.mu.Lock()
	activeAgentMapCache.data = data
	activeAgentMapCache.fetchedAt = time.Now()
	activeAgentMapCache.mu.Unlock()

	return data
}

// buildActiveAgentMap returns a map of beads_id -> active agent info.
// Uses the trackedAgentsCache as the primary source for agent data (phase, model),
// which ensures consistency with /api/agents. Falls back to direct OpenCode/tmux
// queries for agents not in the cache (e.g., untracked sessions).
//
// Before orch-go-1183: this function independently queried tmux via ListWorkersSessions()
// and ListWindows(), which could return different results from the tmux liveness check
// in queryTrackedAgents, causing the dashboard to oscillate between correct status
// and 'unassigned' on every poll cycle.
func buildActiveAgentMap() map[string]*ActiveAgentInfo {
	now := time.Now()

	// Run steps 1 (tracked agents) and 2 (OpenCode sessions) in parallel
	// since they are independent data sources.
	type trackedResult struct {
		agents []discovery.AgentStatus
		err    error
	}
	type sessionsResult struct {
		sessions []opencode.Session
		err      error
	}

	trackedCh := make(chan trackedResult, 1)
	sessionsCh := make(chan sessionsResult, 1)

	// Step 1: Tracked agents from cache (primary source, consistent with /api/agents)
	go func() {
		projectDirs := uniqueProjectDirs(append([]string{sourceDir}, getKBProjectsFn()...))
		tracked, err := globalTrackedAgentsCache.get(projectDirs)
		trackedCh <- trackedResult{agents: tracked, err: err}
	}()

	// Step 2: OpenCode sessions across all projects
	go func() {
		client := opencode.NewClient(serverURL)
		sessions, err := listSessionsAcrossProjects(client, sourceDir)
		sessionsCh <- sessionsResult{sessions: sessions, err: err}
	}()

	// Collect results
	trackedRes := <-trackedCh
	sessionsRes := <-sessionsCh

	// Merge tracked agents into result map
	result := make(map[string]*ActiveAgentInfo)
	if trackedRes.err == nil {
		for _, agent := range trackedRes.agents {
			if agent.BeadsID == "" {
				continue
			}
			result[agent.BeadsID] = &ActiveAgentInfo{
				Phase: agent.Phase,
				Model: agent.Model,
			}
		}
	}

	// Merge OpenCode sessions (for agents not tracked in beads)
	if sessionsRes.err == nil {
		for _, s := range sessionsRes.sessions {
			beadsID := extractBeadsIDFromTitle(s.Title)
			if beadsID == "" || result[beadsID] != nil {
				continue
			}
			createdAt := time.Unix(s.Time.Created/1000, 0)
			result[beadsID] = &ActiveAgentInfo{
				Runtime: formatDuration(now.Sub(createdAt)),
			}
		}

		// Step 3: Enrich tracked agents with runtime from OpenCode sessions
		for _, s := range sessionsRes.sessions {
			beadsID := extractBeadsIDFromTitle(s.Title)
			if beadsID == "" {
				continue
			}
			if info, ok := result[beadsID]; ok && info.Runtime == "" {
				createdAt := time.Unix(s.Time.Created/1000, 0)
				info.Runtime = formatDuration(now.Sub(createdAt))
			}
		}
	}

	// Step 4: Enrich with phase from beads comments for any agents missing phase
	beadsIDsNeedingPhase := make([]string, 0)
	for id, info := range result {
		if info.Phase == "" {
			beadsIDsNeedingPhase = append(beadsIDsNeedingPhase, id)
		}
	}
	if len(beadsIDsNeedingPhase) > 0 {
		commentsMap := globalBeadsCache.getComments(beadsIDsNeedingPhase, nil)
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
