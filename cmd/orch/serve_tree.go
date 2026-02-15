package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/tree"
)

// treeCache provides TTL-based caching for /api/tree.
// Tree extraction can be expensive, so we cache the result.
type treeCache struct {
	mu sync.RWMutex

	// Cached tree data (separate for knowledge and work views)
	knowledgeTree *tree.KnowledgeNode
	knowledgeTime time.Time

	workTree     []*tree.KnowledgeNode
	workTreeTime time.Time

	// TTL for tree cache
	ttl time.Duration

	// Clusters (shared across views)
	clusters     []*tree.Cluster
	clustersTime time.Time
}

var globalTreeCache *treeCache

func newTreeCache() *treeCache {
	return &treeCache{
		ttl: 10 * time.Second, // Tree changes when bd create/kb commands run
	}
}

// getKnowledgeTree returns cached knowledge tree or builds fresh if stale.
func (c *treeCache) getKnowledgeTree(kbDir string, opts tree.TreeOptions) (*tree.KnowledgeNode, []*tree.Cluster, error) {
	c.mu.RLock()
	if c.knowledgeTree != nil && time.Since(c.knowledgeTime) < c.ttl {
		result := c.knowledgeTree
		clusters := c.clusters
		c.mu.RUnlock()
		return result, clusters, nil
	}
	c.mu.RUnlock()

	// Build fresh tree
	root, clusters, err := tree.BuildKnowledgeTree(kbDir, opts)
	if err != nil {
		return nil, nil, err
	}

	// Update cache
	c.mu.Lock()
	c.knowledgeTree = root
	c.knowledgeTime = time.Now()
	c.clusters = clusters
	c.clustersTime = time.Now()
	c.mu.Unlock()

	return root, clusters, nil
}

// getWorkTree returns cached work tree or builds fresh if stale.
func (c *treeCache) getWorkTree(kbDir string, projectDir string, opts tree.TreeOptions) ([]*tree.KnowledgeNode, error) {
	c.mu.RLock()
	if c.workTree != nil && time.Since(c.workTreeTime) < c.ttl {
		result := c.workTree
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	// Build fresh tree
	workTree, err := tree.BuildWorkTree(kbDir, projectDir, opts)
	if err != nil {
		return nil, err
	}

	// Update cache
	c.mu.Lock()
	c.workTree = workTree
	c.workTreeTime = time.Now()
	c.mu.Unlock()

	return workTree, nil
}

// invalidate clears the tree cache to force refresh on next request.
func (c *treeCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.knowledgeTree = nil
	c.workTree = nil
	c.clusters = nil
}

// handleTree handles GET /api/tree requests.
// Query parameters:
//   - view: "knowledge" (default) or "work"
//   - cluster: filter to specific cluster (optional)
//   - depth: maximum depth to render (default: 0 = unlimited)
//
// Returns full tree as JSON.
func handleTree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	view := r.URL.Query().Get("view")
	if view == "" {
		view = "knowledge"
	}

	cluster := r.URL.Query().Get("cluster")

	depthStr := r.URL.Query().Get("depth")
	depth := 0
	if depthStr != "" {
		if d, err := strconv.Atoi(depthStr); err == nil && d > 0 {
			depth = d
		}
	}

	// Build tree options
	opts := tree.TreeOptions{
		ClusterFilter: cluster,
		Depth:         depth,
		Format:        "json", // Always JSON for API
		WorkView:      view == "work",
	}

	// Determine paths
	kbDir := filepath.Join(sourceDir, ".kb")
	projectDir := sourceDir

	var result interface{}
	var err error

	if view == "work" {
		// Build work tree (issues as primary nodes)
		var workTree []*tree.KnowledgeNode
		workTree, err = globalTreeCache.getWorkTree(kbDir, projectDir, opts)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to build work tree: %v", err), http.StatusInternalServerError)
			return
		}

		// Wrap in root node for consistent structure
		result = &tree.KnowledgeNode{
			Type:     tree.NodeTypeCluster,
			Title:    "work-view",
			Children: workTree,
		}
	} else {
		// Build knowledge tree
		var root *tree.KnowledgeNode
		root, _, err = globalTreeCache.getKnowledgeTree(kbDir, opts)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to build knowledge tree: %v", err), http.StatusInternalServerError)
			return
		}
		result = root
	}

	// Return JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode tree: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleTreeEvents handles GET /api/events/tree for SSE tree updates.
// Watches .beads/ and .kb/ directories for changes and pushes tree updates.
func handleTreeEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Get flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Parse query parameters (same as handleTree)
	view := r.URL.Query().Get("view")
	if view == "" {
		view = "knowledge"
	}

	cluster := r.URL.Query().Get("cluster")

	depthStr := r.URL.Query().Get("depth")
	depth := 0
	if depthStr != "" {
		if d, err := strconv.Atoi(depthStr); err == nil && d > 0 {
			depth = d
		}
	}

	// Build tree options
	opts := tree.TreeOptions{
		ClusterFilter: cluster,
		Depth:         depth,
		Format:        "json",
		WorkView:      view == "work",
	}

	// Determine paths
	kbDir := filepath.Join(sourceDir, ".kb")
	beadsDir := filepath.Join(sourceDir, ".beads")
	projectDir := sourceDir

	// Send connected event
	fmt.Fprintf(w, "event: connected\ndata: {\"view\": \"%s\"}\n\n", view)
	flusher.Flush()

	ctx := r.Context()

	// Track last modification times to detect changes
	lastKbModTime := getLastModTime(kbDir)
	lastBeadsModTime := getLastModTime(beadsDir)

	// Poll for changes every 2 seconds
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return
		case <-ticker.C:
			// Check for changes
			kbModTime := getLastModTime(kbDir)
			beadsModTime := getLastModTime(beadsDir)

			changed := false
			if !kbModTime.Equal(lastKbModTime) {
				lastKbModTime = kbModTime
				changed = true
			}
			if !beadsModTime.Equal(lastBeadsModTime) {
				lastBeadsModTime = beadsModTime
				changed = true
			}

			if !changed {
				continue
			}

			// Invalidate cache to force fresh tree extraction
			globalTreeCache.invalidate()

			// Build fresh tree
			var result interface{}
			var err error

			if view == "work" {
				var workTree []*tree.KnowledgeNode
				workTree, err = globalTreeCache.getWorkTree(kbDir, projectDir, opts)
				if err != nil {
					fmt.Fprintf(w, "event: error\ndata: {\"error\": \"Failed to build work tree: %s\"}\n\n", err.Error())
					flusher.Flush()
					continue
				}

				result = &tree.KnowledgeNode{
					Type:     tree.NodeTypeCluster,
					Title:    "work-view",
					Children: workTree,
				}
			} else {
				var root *tree.KnowledgeNode
				root, _, err = globalTreeCache.getKnowledgeTree(kbDir, opts)
				if err != nil {
					fmt.Fprintf(w, "event: error\ndata: {\"error\": \"Failed to build knowledge tree: %s\"}\n\n", err.Error())
					flusher.Flush()
					continue
				}
				result = root
			}

			// Encode as JSON
			jsonData, err := json.Marshal(result)
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: {\"error\": \"Failed to encode tree: %s\"}\n\n", err.Error())
				flusher.Flush()
				continue
			}

			// Send tree-update event
			fmt.Fprintf(w, "event: tree-update\ndata: %s\n\n", string(jsonData))
			flusher.Flush()
		}
	}
}

// getLastModTime returns the last modification time of the most recently modified file in a directory tree.
// This is used to detect when files change in .kb/ or .beads/ directories.
func getLastModTime(dir string) time.Time {
	var lastMod time.Time

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() && info.ModTime().After(lastMod) {
			lastMod = info.ModTime()
		}
		return nil
	})

	return lastMod
}
