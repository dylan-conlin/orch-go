package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

// handleHotspot returns hotspot analysis data for the dashboard.
// This reuses the same logic as the orch hotspot CLI command.
func (s *Server) handleHotspot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Use sourceDir since serve may run from any working directory
	projectDir, _ := s.currentProjectDir()

	// Use default thresholds from hotspot.go
	daysBack := 28
	fixThreshold := 5
	invThreshold := 3

	report := HotspotReport{
		GeneratedAt:    time.Now().Format(time.RFC3339),
		AnalysisPeriod: fmt.Sprintf("Last %d days", daysBack),
		FixThreshold:   fixThreshold,
		InvThreshold:   invThreshold,
		Hotspots:       []Hotspot{},
	}

	// Analyze git log for fix commit density
	fixHotspots, totalFixes, err := analyzeFixCommits(projectDir, daysBack, fixThreshold)
	if err == nil {
		report.TotalFixCommits = totalFixes
		report.Hotspots = append(report.Hotspots, fixHotspots...)
	}

	// Query kb reflect for investigation clustering
	invHotspots, totalInv, err := analyzeInvestigationClusters(projectDir, invThreshold)
	if err == nil {
		report.TotalInvestigations = totalInv
		report.Hotspots = append(report.Hotspots, invHotspots...)
	}

	// Sort hotspots by score (descending)
	sort.Slice(report.Hotspots, func(i, j int) bool {
		return report.Hotspots[i].Score > report.Hotspots[j].Score
	})

	report.HasArchitectWork = len(report.Hotspots) > 0

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(report); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode hotspot report: %v", err), http.StatusInternalServerError)
		return
	}
}
