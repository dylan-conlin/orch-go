package main

import (
	"bufio"
	"encoding/json"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

// gapAnalysisCache caches getGapAnalysisFromEvents results to avoid re-reading
// the entire events.jsonl file (62MB) on every /api/agents request.
// Gap data only changes when agents are spawned, so a 60s TTL is appropriate.
type gapAnalysisCache struct {
	mu        sync.RWMutex
	data      map[string]*GapAPIResponse
	fetchedAt time.Time
	ttl       time.Duration
	key       string // sorted beads IDs as cache key
}

var globalGapAnalysisCache = &gapAnalysisCache{
	ttl: 60 * time.Second,
}

func (c *gapAnalysisCache) get(beadsIDs []string) map[string]*GapAPIResponse {
	sorted := make([]string, len(beadsIDs))
	copy(sorted, beadsIDs)
	sort.Strings(sorted)
	key := strings.Join(sorted, ",")

	c.mu.RLock()
	if c.data != nil && c.key == key && time.Since(c.fetchedAt) < c.ttl {
		result := c.data
		c.mu.RUnlock()
		return result
	}
	c.mu.RUnlock()

	data := getGapAnalysisFromEventsUncached(beadsIDs)

	c.mu.Lock()
	c.data = data
	c.key = key
	c.fetchedAt = time.Now()
	c.mu.Unlock()

	return data
}

func (c *gapAnalysisCache) invalidate() {
	c.mu.Lock()
	c.data = nil
	c.fetchedAt = time.Time{}
	c.mu.Unlock()
}

// getGapAnalysisFromEvents returns cached gap analysis data for the given beads IDs.
func getGapAnalysisFromEvents(beadsIDs []string) map[string]*GapAPIResponse {
	return globalGapAnalysisCache.get(beadsIDs)
}

// getGapAnalysisFromEventsUncached reads spawn events and extracts gap analysis data for given beads IDs.
// Returns a map of beads ID -> GapAPIResponse.
func getGapAnalysisFromEventsUncached(beadsIDs []string) map[string]*GapAPIResponse {
	result := make(map[string]*GapAPIResponse)
	if len(beadsIDs) == 0 {
		return result
	}

	// Build a set of beads IDs for fast lookup
	beadsIDSet := make(map[string]bool)
	for _, id := range beadsIDs {
		beadsIDSet[id] = true
	}

	// Read events file
	logPath := events.DefaultLogPath()
	file, err := os.Open(logPath)
	if err != nil {
		return result
	}
	defer file.Close()

	// Scan events for spawn events matching our beads IDs
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event events.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		// Only process spawn events
		if event.Type != "session.spawned" {
			continue
		}

		// Check if this event is for one of our beads IDs
		beadsID, ok := event.Data["beads_id"].(string)
		if !ok || !beadsIDSet[beadsID] {
			continue
		}

		// Already have gap analysis for this beads ID? Skip (we want the most recent)
		// Since we read chronologically, later entries overwrite earlier ones
		if _, exists := result[beadsID]; exists {
			// We could skip, but let's allow overwrites for resumptions
		}

		// Extract gap analysis data from event
		gapData := extractGapAnalysisFromEvent(event.Data)
		if gapData != nil {
			result[beadsID] = gapData
		}
	}

	return result
}

// extractGapAnalysisFromEvent extracts gap analysis data from a spawn event's data map.
func extractGapAnalysisFromEvent(data map[string]interface{}) *GapAPIResponse {
	// Check if gap data exists
	hasGaps, ok := data["gap_has_gaps"].(bool)
	if !ok {
		return nil
	}

	contextQuality := 0
	if cq, ok := data["gap_context_quality"].(float64); ok {
		contextQuality = int(cq)
	}

	shouldWarn := false
	if sw, ok := data["gap_should_warn"].(bool); ok {
		shouldWarn = sw
	}

	matchCount := 0
	if mc, ok := data["gap_match_total"].(float64); ok {
		matchCount = int(mc)
	}

	constraints := 0
	if c, ok := data["gap_match_constraints"].(float64); ok {
		constraints = int(c)
	}

	decisions := 0
	if d, ok := data["gap_match_decisions"].(float64); ok {
		decisions = int(d)
	}

	investigations := 0
	if i, ok := data["gap_match_investigations"].(float64); ok {
		investigations = int(i)
	}

	return &GapAPIResponse{
		HasGaps:        hasGaps,
		ContextQuality: contextQuality,
		ShouldWarn:     shouldWarn,
		MatchCount:     matchCount,
		Constraints:    constraints,
		Decisions:      decisions,
		Investigations: investigations,
	}
}
