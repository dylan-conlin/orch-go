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

	"github.com/dylan-conlin/orch-go/pkg/timeline"
)

// timelineCache provides TTL-based caching for /api/timeline.
type timelineCache struct {
	mu sync.RWMutex

	// Cached timeline data
	data         *timeline.Timeline
	lastModified time.Time

	// TTL for timeline cache
	ttl time.Duration
}

var globalTimelineCache *timelineCache

func newTimelineCache() *timelineCache {
	return &timelineCache{
		ttl: 5 * time.Second, // Timeline changes when events are logged
	}
}

// get returns cached timeline or builds fresh if stale.
func (c *timelineCache) get(projectDir string, opts timeline.ExtractOptions) (*timeline.Timeline, error) {
	c.mu.RLock()
	if c.data != nil && time.Since(c.lastModified) < c.ttl {
		result := c.data
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	// Build fresh timeline
	data, err := timeline.Extract(opts)
	if err != nil {
		return nil, err
	}

	// Update cache
	c.mu.Lock()
	c.data = data
	c.lastModified = time.Now()
	c.mu.Unlock()

	return data, nil
}

// invalidate clears the timeline cache to force refresh on next request.
func (c *timelineCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = nil
}

// handleTimeline handles GET /api/timeline requests.
// Query parameters:
//   - session: filter to specific session ID (optional)
//   - limit: maximum number of sessions to return (default: 10)
//
// Returns timeline grouped by session as JSON.
func handleTimeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	sessionID := r.URL.Query().Get("session")

	limitStr := r.URL.Query().Get("limit")
	limit := 10 // Default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Build extraction options
	opts := timeline.ExtractOptions{
		ProjectDir: sourceDir,
		SessionID:  sessionID,
		Limit:      limit,
	}

	// Get timeline (from cache or fresh)
	data, err := globalTimelineCache.get(sourceDir, opts)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to extract timeline: %v", err), http.StatusInternalServerError)
		return
	}

	// Return JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode timeline: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleTimelineEvents handles GET /api/events/timeline for SSE timeline updates.
// Watches .orch/events.jsonl and other data sources for changes and pushes timeline updates.
func handleTimelineEvents(w http.ResponseWriter, r *http.Request) {
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

	// Parse query parameters (same as handleTimeline)
	sessionID := r.URL.Query().Get("session")

	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Build extraction options
	opts := timeline.ExtractOptions{
		ProjectDir: sourceDir,
		SessionID:  sessionID,
		Limit:      limit,
	}

	// Send connected event
	fmt.Fprintf(w, "event: connected\ndata: {\"status\": \"connected\"}\n\n")
	flusher.Flush()

	ctx := r.Context()

	// Track last modification times to detect changes
	orchDir := filepath.Join(sourceDir, ".orch")
	eventsFile := filepath.Join(orchDir, "events.jsonl")
	labelsFile := filepath.Join(orchDir, "session_labels.json")
	kbDir := filepath.Join(sourceDir, ".kb")

	lastEventsModTime := getFileModTime(eventsFile)
	lastLabelsModTime := getFileModTime(labelsFile)
	lastKbModTime := getLastModTime(kbDir)

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
			eventsModTime := getFileModTime(eventsFile)
			labelsModTime := getFileModTime(labelsFile)
			kbModTime := getLastModTime(kbDir)

			changed := false
			if !eventsModTime.Equal(lastEventsModTime) {
				lastEventsModTime = eventsModTime
				changed = true
			}
			if !labelsModTime.Equal(lastLabelsModTime) {
				lastLabelsModTime = labelsModTime
				changed = true
			}
			if !kbModTime.Equal(lastKbModTime) {
				lastKbModTime = kbModTime
				changed = true
			}

			if !changed {
				continue
			}

			// Invalidate cache to force fresh timeline extraction
			globalTimelineCache.invalidate()

			// Build fresh timeline
			data, err := globalTimelineCache.get(sourceDir, opts)
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: {\"error\": \"Failed to extract timeline: %s\"}\n\n", err.Error())
				flusher.Flush()
				continue
			}

			// Encode as JSON
			jsonData, err := json.Marshal(data)
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: {\"error\": \"Failed to encode timeline: %s\"}\n\n", err.Error())
				flusher.Flush()
				continue
			}

			// Send timeline-update event
			fmt.Fprintf(w, "event: timeline-update\ndata: %s\n\n", string(jsonData))
			flusher.Flush()
		}
	}
}

// getFileModTime returns the last modification time of a single file.
func getFileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
