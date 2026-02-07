package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

// KBHealthResponse is the JSON structure returned by /api/kb-health.
// It provides knowledge hygiene signals for the Strategic Center dashboard.
type KBHealthResponse struct {
	Synthesis              KBHealthCategory `json:"synthesis"`
	Promote                KBHealthCategory `json:"promote"`
	Stale                  KBHealthCategory `json:"stale"`
	InvestigationPromotion KBHealthCategory `json:"investigation_promotion"`
	InvestigationAuthority KBHealthCategory `json:"investigation_authority"`
	Total                  int              `json:"total"`
	LastUpdated            string           `json:"last_updated"`
	Error                  string           `json:"error,omitempty"`
}

// KBHealthCategory represents a single knowledge hygiene category.
type KBHealthCategory struct {
	Count int                      `json:"count"`
	Items []map[string]interface{} `json:"items"`
}

// kbHealthCache provides TTL-based caching for /api/kb-health.
// kb reflect can be slow with many artifacts, so we cache with 5-minute TTL.
type kbHealthCache struct {
	mu sync.RWMutex

	data      *KBHealthResponse
	fetchedAt time.Time
	ttl       time.Duration
}

// Global kb health cache, initialized in runServe
var globalKBHealthCache *kbHealthCache

func newKBHealthCache() *kbHealthCache {
	return &kbHealthCache{
		ttl: 5 * time.Minute, // Knowledge changes slowly
	}
}

// get returns cached data or fetches fresh if stale.
func (c *kbHealthCache) get(projectDir string) (*KBHealthResponse, error) {
	c.mu.RLock()
	if c.data != nil && time.Since(c.fetchedAt) < c.ttl {
		result := c.data
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	// Fetch fresh data
	data, err := fetchKBHealth(projectDir)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.data = data
	c.fetchedAt = time.Now()
	c.mu.Unlock()

	return data, nil
}

// invalidate clears the cache, forcing fresh fetch on next request.
func (c *kbHealthCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = nil
}

// fetchKBHealth calls kb reflect for each type and aggregates results.
func fetchKBHealth(projectDir string) (*KBHealthResponse, error) {
	response := &KBHealthResponse{
		Synthesis: KBHealthCategory{
			Count: 0,
			Items: []map[string]interface{}{},
		},
		Promote: KBHealthCategory{
			Count: 0,
			Items: []map[string]interface{}{},
		},
		Stale: KBHealthCategory{
			Count: 0,
			Items: []map[string]interface{}{},
		},
		InvestigationPromotion: KBHealthCategory{
			Count: 0,
			Items: []map[string]interface{}{},
		},
		InvestigationAuthority: KBHealthCategory{
			Count: 0,
			Items: []map[string]interface{}{},
		},
		Total:       0,
		LastUpdated: time.Now().Format(time.RFC3339),
	}

	// Fetch each reflection type in sequence
	// Note: Could parallelize with goroutines + WaitGroup if performance matters
	types := []struct {
		name   string
		target *KBHealthCategory
	}{
		{"synthesis", &response.Synthesis},
		{"promote", &response.Promote},
		{"stale", &response.Stale},
		{"investigation-promotion", &response.InvestigationPromotion},
		{"investigation-authority", &response.InvestigationAuthority},
	}

	for _, t := range types {
		items, err := fetchKBReflect(projectDir, t.name)
		if err != nil {
			// Graceful degradation: continue with other types on error
			continue
		}
		t.target.Items = items
		t.target.Count = len(items)
		response.Total += len(items)
	}

	return response, nil
}

// fetchKBReflect calls kb reflect for a specific type and returns parsed items.
// Returns empty slice on error for graceful degradation.
func fetchKBReflect(projectDir, reflectType string) ([]map[string]interface{}, error) {
	// Run kb reflect --type TYPE --format json --limit 5
	cmd := exec.Command("kb", "reflect", "--type", reflectType, "--format", "json", "--limit", "5")
	if projectDir != "" {
		cmd.Dir = projectDir
	}

	output, err := cmd.Output()
	if err != nil {
		// kb CLI not available or command failed - graceful degradation
		return []map[string]interface{}{}, nil
	}

	// kb reflect returns nested JSON: {"synthesis": [...], "promote": [...], etc}
	// We need to extract the array for the specific type
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		// Parse error - return empty for graceful degradation
		return []map[string]interface{}{}, nil
	}

	// Extract the array for this reflection type
	// The key matches the reflectType (synthesis, promote, stale, investigation-promotion)
	// But investigation-promotion and investigation-authority use underscore in JSON
	jsonKey := reflectType
	if reflectType == "investigation-promotion" {
		jsonKey = "investigation_promotion"
	}
	if reflectType == "investigation-authority" {
		jsonKey = "investigation_authority"
	}

	itemsRaw, ok := result[jsonKey]
	if !ok {
		// Key not found - return empty for graceful degradation
		return []map[string]interface{}{}, nil
	}

	// Convert to []map[string]interface{}
	itemsArray, ok := itemsRaw.([]interface{})
	if !ok {
		// Not an array - return empty for graceful degradation
		return []map[string]interface{}{}, nil
	}

	items := make([]map[string]interface{}, 0, len(itemsArray))
	for _, item := range itemsArray {
		if itemMap, ok := item.(map[string]interface{}); ok {
			items = append(items, itemMap)
		}
	}

	return items, nil
}

// handleKBHealth returns knowledge hygiene signals from kb reflect.
// Used by Strategic Center dashboard to surface synthesis opportunities,
// pending promotions, stale decisions, and investigation promotions.
func (s *Server) handleKBHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get project directory from query parameter (default to sourceDir)
	projectDir := r.URL.Query().Get("project")
	if projectDir == "" {
		projectDir, _ = currentProjectDir()
	}

	// Check if cache is initialized
	if s.KBHealthCache == nil {
		// Fallback: fetch without cache if not initialized
		data, err := fetchKBHealth(projectDir)
		if err != nil {
			resp := &KBHealthResponse{
				Error: fmt.Sprintf("Failed to fetch kb health: %v", err),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
		return
	}

	// Get cached or fresh data
	data, err := s.KBHealthCache.get(projectDir)
	if err != nil {
		resp := &KBHealthResponse{
			Error: fmt.Sprintf("Failed to fetch kb health: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode kb health: %v", err), http.StatusInternalServerError)
		return
	}
}
