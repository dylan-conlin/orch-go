package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// GapsAPIResponse is the JSON structure returned by /api/gaps.
type GapsAPIResponse struct {
	TotalGaps         int                    `json:"total_gaps"`
	RecurringPatterns int                    `json:"recurring_patterns"`
	BySkill           map[string]int         `json:"by_skill"`
	RecentGaps        int                    `json:"recent_gaps,omitempty"`       // Gaps in last 7 days
	Suggestions       []GapSuggestionSummary `json:"suggestions,omitempty"`       // Top recurring gap suggestions
	Error             string                 `json:"error,omitempty"`
}

// ReflectAPIResponse is the JSON structure returned by /api/reflect.
// It exposes the reflect-suggestions.json data with synthesis/promote/stale info.
type ReflectAPIResponse struct {
	Timestamp string                   `json:"timestamp"`
	Synthesis []ReflectSynthesisSummary `json:"synthesis"`
	Refine    []ReflectRefineSummary   `json:"refine,omitempty"`
	Error     string                   `json:"error,omitempty"`
}

// ReflectRefineSummary represents a kn entry that refines an existing principle.
type ReflectRefineSummary struct {
	ID         string   `json:"id"`
	Content    string   `json:"content"`
	Principle  string   `json:"principle"`
	MatchTerms []string `json:"match_terms"`
	Suggestion string   `json:"suggestion"`
}

// ReflectSynthesisSummary represents a topic with accumulated investigations.
type ReflectSynthesisSummary struct {
	Topic          string   `json:"topic"`
	Count          int      `json:"count"`
	Investigations []string `json:"investigations"`
	Suggestion     string   `json:"suggestion"`
}

// GapSuggestionSummary is a condensed version of LearningSuggestion for the API.
type GapSuggestionSummary struct {
	Query      string `json:"query"`
	Count      int    `json:"count"`
	Priority   string `json:"priority"`
	Suggestion string `json:"suggestion"`
}

// handleGaps returns gap tracker statistics from ~/.orch/gap-tracker.json.
func handleGaps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tracker, err := spawn.LoadTracker()
	if err != nil {
		resp := GapsAPIResponse{Error: fmt.Sprintf("Failed to load gap tracker: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Calculate by_skill breakdown
	bySkill := tracker.GetSkillGapRates()

	// Find recurring patterns (gaps that occurred 3+ times)
	suggestions := tracker.FindRecurringGaps()

	// Count recent gaps (last 7 days)
	recentGaps := 0
	weekAgo := time.Now().Add(-7 * 24 * time.Hour)
	for _, event := range tracker.Events {
		if event.Timestamp.After(weekAgo) {
			recentGaps++
		}
	}

	// Convert suggestions to API format (top 5)
	var apiSuggestions []GapSuggestionSummary
	for i, s := range suggestions {
		if i >= 5 {
			break
		}
		apiSuggestions = append(apiSuggestions, GapSuggestionSummary{
			Query:      s.Query,
			Count:      s.Count,
			Priority:   s.Priority,
			Suggestion: s.Suggestion,
		})
	}

	resp := GapsAPIResponse{
		TotalGaps:         len(tracker.Events),
		RecurringPatterns: len(suggestions),
		BySkill:           bySkill,
		RecentGaps:        recentGaps,
		Suggestions:       apiSuggestions,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode gaps: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleReflect returns reflect suggestions from ~/.orch/reflect-suggestions.json.
// This exposes synthesis/promote/stale data for kb reflect UI integration.
func handleReflect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read reflect-suggestions.json from ~/.orch/
	homeDir, err := os.UserHomeDir()
	if err != nil {
		resp := ReflectAPIResponse{Error: fmt.Sprintf("Failed to get home directory: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	reflectPath := filepath.Join(homeDir, ".orch", "reflect-suggestions.json")
	data, err := os.ReadFile(reflectPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty response if file doesn't exist yet
			resp := ReflectAPIResponse{
				Timestamp: "",
				Synthesis: []ReflectSynthesisSummary{},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		resp := ReflectAPIResponse{Error: fmt.Sprintf("Failed to read reflect-suggestions.json: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Parse the raw JSON structure
	var rawReflect struct {
		Timestamp string `json:"timestamp"`
		Synthesis []struct {
			Topic          string   `json:"topic"`
			Count          int      `json:"count"`
			Investigations []string `json:"investigations"`
			Suggestion     string   `json:"suggestion"`
		} `json:"synthesis"`
		Refine []struct {
			ID         string   `json:"id"`
			Content    string   `json:"content"`
			Principle  string   `json:"principle"`
			MatchTerms []string `json:"match_terms"`
			Suggestion string   `json:"suggestion"`
		} `json:"refine"`
	}

	if err := json.Unmarshal(data, &rawReflect); err != nil {
		resp := ReflectAPIResponse{Error: fmt.Sprintf("Failed to parse reflect-suggestions.json: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Convert to API response format
	var synthesis []ReflectSynthesisSummary
	for _, s := range rawReflect.Synthesis {
		synthesis = append(synthesis, ReflectSynthesisSummary{
			Topic:          s.Topic,
			Count:          s.Count,
			Investigations: s.Investigations,
			Suggestion:     s.Suggestion,
		})
	}

	// Convert refine data
	var refine []ReflectRefineSummary
	for _, r := range rawReflect.Refine {
		refine = append(refine, ReflectRefineSummary{
			ID:         r.ID,
			Content:    r.Content,
			Principle:  r.Principle,
			MatchTerms: r.MatchTerms,
			Suggestion: r.Suggestion,
		})
	}

	resp := ReflectAPIResponse{
		Timestamp: rawReflect.Timestamp,
		Synthesis: synthesis,
		Refine:    refine,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode reflect: %v", err), http.StatusInternalServerError)
		return
	}
}
