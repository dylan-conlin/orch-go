package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/kb"
)

// GraphNode represents a node in the decidability graph.
// Can be a beads issue or a kb artifact (investigation/decision).
type GraphNode struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Type        string   `json:"type"`                  // beads: task, bug, feature, epic, question; kb: investigation, decision
	Status      string   `json:"status"`                // open, in_progress, closed, blocked, Complete, Accepted, etc.
	Priority    int      `json:"priority"`              // 0-4 for beads, 0 for kb artifacts
	Source      string   `json:"source"`                // "beads" or "kb"
	Date        string   `json:"date,omitempty"`        // for kb artifacts
	CreatedAt   string   `json:"created_at,omitempty"`  // creation timestamp
	Description string   `json:"description,omitempty"` // issue description
	Labels      []string `json:"labels,omitempty"`      // issue labels (area:*, effort:*, triage:*, etc.)
	Layer       int      `json:"layer"`                 // execution layer from topological sort (0 = no blocking deps)
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

type graphBuildTimings struct {
	List         time.Duration
	Dependencies time.Duration
	ParentFilter time.Duration
	TreeBuild    time.Duration
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

	nodeIndex := make(map[string]int)
	for i, node := range nodes {
		nodeIndex[node.ID] = i
		nodes[i].Layer = -1
	}

	dependsOn := make(map[string][]string)
	for _, edge := range edges {
		if edge.Type == "blocks" {
			dependsOn[edge.From] = append(dependsOn[edge.From], edge.To)
		}
	}

	changed := true
	for changed {
		changed = false
		for id, idx := range nodeIndex {
			if nodes[idx].Layer >= 0 {
				continue
			}
			deps := dependsOn[id]
			if len(deps) == 0 {
				nodes[idx].Layer = 0
				changed = true
			} else {
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

	for i := range nodes {
		if nodes[i].Layer < 0 {
			nodes[i].Layer = 0
		}
	}

	return nodes
}

// beadsIssue is the parsed structure from bd list --json
type beadsIssue struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Status          string   `json:"status"`
	Priority        int      `json:"priority"`
	IssueType       string   `json:"issue_type"`
	Description     string   `json:"description,omitempty"`
	CreatedAt       string   `json:"created_at,omitempty"`
	Labels          []string `json:"labels,omitempty"`
	DependencyCount int      `json:"dependency_count"`
	DependentCount  int      `json:"dependent_count"`
	Parent          string   `json:"parent,omitempty"`
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
func (s *Server) handleBeadsGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectDir := r.URL.Query().Get("project_dir")
	scope := r.URL.Query().Get("scope")
	parentID := r.URL.Query().Get("parent")
	if scope == "" {
		scope = "focus"
	}

	workDir := projectDir
	if workDir == "" {
		workDir = beads.DefaultDir
	}

	cacheKey := scope + ":" + parentID

	resp, err := s.BeadsStatsCache.getGraph(projectDir, cacheKey, func() (*BeadsGraphAPIResponse, error) {
		buildStart := time.Now()
		var nodes []GraphNode
		var edges []GraphEdge
		var buildErr error
		var stageTimings graphBuildTimings

		if scope == "focus" {
			nodes, edges, buildErr = s.buildFocusGraph(workDir)
		} else {
			includeAll := scope == "all"
			nodes, edges, stageTimings, buildErr = s.buildFullGraph(workDir, includeAll, parentID, projectDir)
		}

		if buildErr != nil {
			return &BeadsGraphAPIResponse{
				Nodes:      []GraphNode{},
				Edges:      []GraphEdge{},
				ProjectDir: projectDir,
				Error:      buildErr.Error(),
			}, nil
		}

		focusParentFilterStart := time.Now()
		if parentID != "" && scope == "focus" {
			nodes, edges = filterToParentAndDescendants(nodes, edges, parentID)
			stageTimings.ParentFilter = time.Since(focusParentFilterStart)
		}

		treeBuildStart := time.Now()
		nodes = computeLayers(nodes, edges)
		stageTimings.TreeBuild = time.Since(treeBuildStart)
		totalDuration := time.Since(buildStart)

		if scope != "focus" {
			log.Printf("[beads-graph] stage=list scope=%s parent=%q duration=%s", scope, parentID, stageTimings.List)
			log.Printf("[beads-graph] stage=dep-lookups scope=%s parent=%q duration=%s", scope, parentID, stageTimings.Dependencies)
		}
		log.Printf("[beads-graph] stage=tree-building scope=%s parent=%q duration=%s", scope, parentID, stageTimings.TreeBuild)

		log.Printf("[beads-graph] scope=%s parent=%q nodes=%d edges=%d total=%s list=%s deps=%s parent_filter=%s tree_build=%s",
			scope,
			parentID,
			len(nodes),
			len(edges),
			totalDuration,
			stageTimings.List,
			stageTimings.Dependencies,
			stageTimings.ParentFilter,
			stageTimings.TreeBuild,
		)

		return &BeadsGraphAPIResponse{
			Nodes:      nodes,
			Edges:      edges,
			NodeCount:  len(nodes),
			EdgeCount:  len(edges),
			ProjectDir: projectDir,
		}, nil
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to build graph: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode graph: %v", err), http.StatusInternalServerError)
		return
	}
}

// filterToParentAndDescendants filters nodes and edges to only include
// the specified parent issue and all its descendants.
func filterToParentAndDescendants(nodes []GraphNode, edges []GraphEdge, parentID string) ([]GraphNode, []GraphEdge) {
	isDescendant := func(id string) bool {
		return id == parentID || strings.HasPrefix(id, parentID+".")
	}

	filteredNodes := make([]GraphNode, 0)
	nodeIDs := make(map[string]bool)
	for _, node := range nodes {
		if isDescendant(node.ID) {
			filteredNodes = append(filteredNodes, node)
			nodeIDs[node.ID] = true
		}
	}

	filteredEdges := make([]GraphEdge, 0)
	for _, edge := range edges {
		if nodeIDs[edge.From] && nodeIDs[edge.To] {
			filteredEdges = append(filteredEdges, edge)
		}
	}

	return filteredNodes, filteredEdges
}

func filterIssuesToParentAndDescendants(issues []beadsIssue, parentID string) []beadsIssue {
	if parentID == "" {
		return issues
	}

	isDescendant := func(id string) bool {
		return id == parentID || strings.HasPrefix(id, parentID+".")
	}

	filtered := make([]beadsIssue, 0)
	for _, issue := range issues {
		if isDescendant(issue.ID) {
			filtered = append(filtered, issue)
		}
	}

	return filtered
}

// buildFocusGraph builds a focused graph showing the active working set.
func (s *Server) buildFocusGraph(workDir string) ([]GraphNode, []GraphEdge, error) {
	allIssues, err := s.listBeadsIssues(workDir, false)
	if err != nil {
		return nil, nil, err
	}

	issueByID := make(map[string]beadsIssue)
	for _, issue := range allIssues {
		issueByID[issue.ID] = issue
	}

	focusSet := make(map[string]bool)
	for _, issue := range allIssues {
		if issue.Status == "in_progress" {
			focusSet[issue.ID] = true
		}
	}
	for _, issue := range allIssues {
		if issue.Priority <= 1 {
			focusSet[issue.ID] = true
		}
	}

	edges := make([]GraphEdge, 0)
	processedForDeps := make(map[string]bool)

	for id := range focusSet {
		if processedForDeps[id] {
			continue
		}
		processedForDeps[id] = true

		showIssue, err := s.showBeadsIssue(workDir, id)
		if err != nil {
			continue
		}

		for _, dep := range showIssue.Dependencies {
			focusSet[dep.ID] = true
		}
		for _, dep := range showIssue.Dependents {
			focusSet[dep.ID] = true
		}

		deps, depErr := s.listIssueDependencies(workDir, id)
		if depErr == nil {
			for _, dep := range deps {
				edges = append(edges, GraphEdge{From: id, To: dep.ID, Type: dep.DependencyType})
			}
		}
		dependents, depErr := s.listIssueDependents(workDir, id)
		if depErr == nil {
			for _, dep := range dependents {
				edges = append(edges, GraphEdge{From: dep.ID, To: id, Type: dep.DependencyType})
			}
		}
	}

	nodes := make([]GraphNode, 0, len(focusSet))
	for id := range focusSet {
		if issue, ok := issueByID[id]; ok {
			nodes = append(nodes, GraphNode{
				ID: issue.ID, Title: issue.Title, Type: issue.IssueType,
				Status: issue.Status, Priority: issue.Priority, Source: "beads",
				Description: issue.Description, CreatedAt: issue.CreatedAt, Labels: issue.Labels,
			})
		} else {
			showIssue, err := s.showBeadsIssue(workDir, id)
			if err == nil {
				nodes = append(nodes, GraphNode{
					ID: showIssue.ID, Title: showIssue.Title, Type: showIssue.IssueType,
					Status: showIssue.Status, Priority: showIssue.Priority, Source: "beads",
					Description: showIssue.Description, CreatedAt: showIssue.CreatedAt,
				})
			}
		}
	}

	kbDir := filepath.Join(workDir, ".kb")
	kbArtifacts, err := kb.ListRecentArtifacts(kbDir, 14)
	if err == nil {
		for _, artifact := range kbArtifacts {
			hasRelevantRef := false
			for _, ref := range artifact.References {
				if focusSet[ref] {
					hasRelevantRef = true
					edges = append(edges, GraphEdge{From: artifact.ID, To: ref, Type: "references"})
				}
			}
			if hasRelevantRef {
				nodes = append(nodes, GraphNode{
					ID: artifact.ID, Title: artifact.Title, Type: string(artifact.Type),
					Status: artifact.Status, Source: "kb", Date: artifact.Date,
				})
			}
		}
	}

	return nodes, edges, nil
}

// buildFullGraph builds the full graph with optional status/parent filtering.
func (s *Server) buildFullGraph(workDir string, includeAll bool, parentID, cacheProjectDir string) ([]GraphNode, []GraphEdge, graphBuildTimings, error) {
	var timings graphBuildTimings

	listStart := time.Now()
	issues, err := s.listBeadsIssues(workDir, includeAll)
	timings.List = time.Since(listStart)
	if err != nil {
		return nil, nil, timings, err
	}

	if parentID != "" {
		parentFilterStart := time.Now()
		issues = filterIssuesToParentAndDescendants(issues, parentID)
		timings.ParentFilter = time.Since(parentFilterStart)
	}

	nodes := make([]GraphNode, 0, len(issues))
	nodeIDs := make(map[string]bool, len(issues))
	for _, issue := range issues {
		nodeIDs[issue.ID] = true
		nodes = append(nodes, GraphNode{
			ID: issue.ID, Title: issue.Title, Type: issue.IssueType,
			Status: issue.Status, Priority: issue.Priority, Source: "beads",
			Description: issue.Description, CreatedAt: issue.CreatedAt, Labels: issue.Labels,
		})
	}

	if len(issues) == 0 {
		return nodes, []GraphEdge{}, timings, nil
	}

	depsStart := time.Now()
	edges := make([]GraphEdge, 0)

	dependencyCacheKey := "open"
	if includeAll {
		dependencyCacheKey = "all"
	}

	fetchDependencyGraph := func() ([]GraphEdge, error) {
		return s.listGraphEdges(workDir)
	}

	var allEdges []GraphEdge
	var graphErr error
	if s.BeadsStatsCache != nil {
		allEdges, graphErr = s.BeadsStatsCache.getDependencyGraph(cacheProjectDir, dependencyCacheKey, fetchDependencyGraph)
	} else {
		allEdges, graphErr = fetchDependencyGraph()
	}

	if graphErr == nil {
		for _, edge := range allEdges {
			if parentID == "" {
				if nodeIDs[edge.From] {
					edges = append(edges, edge)
				}
				continue
			}
			if nodeIDs[edge.From] && nodeIDs[edge.To] {
				edges = append(edges, edge)
			}
		}
		timings.Dependencies = time.Since(depsStart)
		return nodes, edges, timings, nil
	}

	idsWithDeps := make([]string, 0)
	for _, issue := range issues {
		if issue.DependencyCount > 0 {
			idsWithDeps = append(idsWithDeps, issue.ID)
		}
	}

	dependencyMap := s.fetchIssueDependencyMap(workDir, idsWithDeps)
	for _, id := range idsWithDeps {
		for _, dep := range dependencyMap[id] {
			edges = append(edges, GraphEdge{From: id, To: dep.ID, Type: dep.DependencyType})
		}
	}

	if parentID != "" {
		filteredEdges := make([]GraphEdge, 0, len(edges))
		for _, edge := range edges {
			if nodeIDs[edge.From] && nodeIDs[edge.To] {
				filteredEdges = append(filteredEdges, edge)
			}
		}
		edges = filteredEdges
	}

	timings.Dependencies = time.Since(depsStart)
	return nodes, edges, timings, nil
}

func (s *Server) fetchIssueDependencyMap(workDir string, ids []string) map[string][]depEntry {
	if len(ids) == 0 {
		return map[string][]depEntry{}
	}

	results := make(map[string][]depEntry, len(ids))
	var resultsMu sync.Mutex

	workerLimit := maxBdConcurrent
	if workerLimit < 1 {
		workerLimit = 1
	}
	semaphore := make(chan struct{}, workerLimit)

	var wg sync.WaitGroup
	for _, id := range ids {
		id := id
		wg.Add(1)
		go func() {
			defer wg.Done()

			semaphore <- struct{}{}
			deps, depErr := s.listIssueDependencies(workDir, id)
			<-semaphore
			if depErr != nil {
				return
			}

			resultsMu.Lock()
			results[id] = deps
			resultsMu.Unlock()
		}()
	}

	wg.Wait()
	return results
}

// listGraphEdges returns full dependency edges from a single bd graph call.
func (s *Server) listGraphEdges(workDir string) ([]GraphEdge, error) {
	key := workDir + ":graph:all"

	result, err, _ := s.bdLimitedList(key, func() (interface{}, error) {
		output, cmdErr := runBdCommandOutput(workDir, "graph", "--all", "--json")
		if cmdErr != nil {
			return nil, fmt.Errorf("bd graph --all failed: %w", cmdErr)
		}

		var graph struct {
			Edges []GraphEdge `json:"edges"`
		}
		if parseErr := json.Unmarshal(output, &graph); parseErr != nil {
			return nil, fmt.Errorf("parse graph output: %w", parseErr)
		}

		return graph.Edges, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]GraphEdge), nil
}

// listBeadsIssues calls bd list and returns parsed issues.
func (s *Server) listBeadsIssues(workDir string, includeAll bool) ([]beadsIssue, error) {
	scope := "open"
	if includeAll {
		scope = "all"
	}
	key := workDir + ":" + scope

	result, err, _ := s.bdLimitedList(key, func() (interface{}, error) {
		if includeAll {
			args := []string{"list", "--json", "--limit", "0", "--all"}
			output, cmdErr := runBdCommandOutput(workDir, args...)
			if cmdErr != nil {
				return nil, fmt.Errorf("bd list failed: %w", cmdErr)
			}
			var issues []beadsIssue
			if parseErr := json.Unmarshal(output, &issues); parseErr != nil {
				return nil, fmt.Errorf("parse issues: %w", parseErr)
			}
			return issues, nil
		}

		var allIssues []beadsIssue
		for _, status := range []string{"open", "in_progress"} {
			args := []string{"list", "--json", "--limit", "0", "--status", status}
			output, cmdErr := runBdCommandOutput(workDir, args...)
			if cmdErr != nil {
				continue
			}
			var issues []beadsIssue
			if parseErr := json.Unmarshal(output, &issues); parseErr != nil {
				continue
			}
			allIssues = append(allIssues, issues...)
		}

		seen := make(map[string]bool)
		unique := make([]beadsIssue, 0, len(allIssues))
		for _, issue := range allIssues {
			if !seen[issue.ID] {
				seen[issue.ID] = true
				unique = append(unique, issue)
			}
		}

		return unique, nil
	})

	if err != nil {
		return nil, err
	}
	return result.([]beadsIssue), nil
}

// showBeadsIssue calls bd show and returns the parsed issue with dependencies.
func (s *Server) showBeadsIssue(workDir, id string) (*beadsShowIssue, error) {
	key := workDir + ":" + id

	result, err, _ := s.bdLimitedShow(key, func() (interface{}, error) {
		output, cmdErr := runBdCommandOutput(workDir, "show", id, "--json")
		if cmdErr != nil {
			return nil, fmt.Errorf("bd show %s failed: %w", id, cmdErr)
		}

		var issues []beadsShowIssue
		if parseErr := json.Unmarshal(output, &issues); parseErr != nil || len(issues) == 0 {
			return nil, fmt.Errorf("parse show output: %w", parseErr)
		}

		return &issues[0], nil
	})

	if err != nil {
		return nil, err
	}
	return result.(*beadsShowIssue), nil
}

// getBdPath returns the resolved bd path or falls back to "bd".
func getBdPath() string {
	if beads.BdPath != "" {
		return beads.BdPath
	}
	return "bd"
}

func runBdCommandOutput(workDir string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmdArgs := args
	if len(cmdArgs) == 0 || cmdArgs[0] != "--sandbox" {
		cmdArgs = append([]string{"--sandbox"}, cmdArgs...)
	}

	cmd := exec.CommandContext(ctx, getBdPath(), cmdArgs...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Env = append(os.Environ(), "BEADS_NO_DAEMON=1")

	output, err := cmd.Output()
	if err != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, fmt.Errorf("bd command timed out after 10s")
	}

	return output, err
}

// depEntry represents a single dependency/dependent relationship.
type depEntry struct {
	ID             string `json:"id"`
	DependencyType string `json:"dependency_type"`
}

// listIssueDependencies returns dependencies for an issue with proper types.
func (s *Server) listIssueDependencies(workDir, id string) ([]depEntry, error) {
	key := workDir + ":deps:" + id

	result, err, _ := s.bdLimitedDep(key, func() (interface{}, error) {
		output, cmdErr := runBdCommandOutput(workDir, "dep", "list", id, "--json")
		if cmdErr != nil {
			return nil, fmt.Errorf("bd dep list %s failed: %w", id, cmdErr)
		}
		var deps []depEntry
		if parseErr := json.Unmarshal(output, &deps); parseErr != nil {
			return nil, fmt.Errorf("parse dep list output: %w", parseErr)
		}
		return deps, nil
	})

	if err != nil {
		return nil, err
	}
	return result.([]depEntry), nil
}

// listIssueDependents returns dependents for an issue with proper types.
func (s *Server) listIssueDependents(workDir, id string) ([]depEntry, error) {
	key := workDir + ":dependents:" + id

	result, err, _ := s.bdLimitedDep(key, func() (interface{}, error) {
		output, cmdErr := runBdCommandOutput(workDir, "dep", "list", id, "--direction", "up", "--json")
		if cmdErr != nil {
			return nil, fmt.Errorf("bd dep list --direction up %s failed: %w", id, cmdErr)
		}
		var deps []depEntry
		if parseErr := json.Unmarshal(output, &deps); parseErr != nil {
			return nil, fmt.Errorf("parse dep list dependents output: %w", parseErr)
		}
		return deps, nil
	})

	if err != nil {
		return nil, err
	}
	return result.([]depEntry), nil
}
