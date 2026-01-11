package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// CoachingMetric represents a single behavioral metric entry.
type CoachingMetric struct {
	Timestamp string                 `json:"timestamp"`
	SessionID string                 `json:"session_id,omitempty"`
	Type      string                 `json:"metric_type"`
	Value     float64                `json:"value"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// CoachingResponse is the API response format (simplified for Frame 2).
type CoachingResponse struct {
	OverallStatus    string `json:"overall_status"` // good/warning/poor
	StatusMessage    string `json:"status_message"` // e.g., "Orchestrator delegating well"
	LastCoachingTime string `json:"last_coaching_time,omitempty"`
	Session          struct {
		SessionID       string `json:"session_id"`
		Started         string `json:"started"`
		DurationMinutes int    `json:"duration_minutes"`
	} `json:"session"`
}

// readCoachingMetrics reads the last N lines from coaching-metrics.jsonl.
func readCoachingMetrics(limit int) ([]CoachingMetric, error) {
	path := filepath.Join(os.Getenv("HOME"), ".orch", "coaching-metrics.jsonl")

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []CoachingMetric{}, nil // No metrics yet
		}
		return nil, fmt.Errorf("failed to open metrics file: %w", err)
	}
	defer file.Close()

	var metrics []CoachingMetric
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var metric CoachingMetric
		if err := json.Unmarshal([]byte(line), &metric); err != nil {
			// Skip malformed lines
			continue
		}

		metrics = append(metrics, metric)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading metrics: %w", err)
	}

	// Return last N lines
	if len(metrics) > limit {
		metrics = metrics[len(metrics)-limit:]
	}

	return metrics, nil
}

// aggregateMetrics aggregates metrics by session and calculates overall health status (Frame 2).
func aggregateMetrics(metrics []CoachingMetric) CoachingResponse {
	resp := CoachingResponse{
		OverallStatus: "good",
		StatusMessage: "No metrics yet",
	}

	if len(metrics) == 0 {
		return resp
	}

	// Sort by timestamp to get latest session
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].Timestamp < metrics[j].Timestamp
	})

	// Get latest session ID
	latestSessionID := metrics[len(metrics)-1].SessionID
	resp.Session.SessionID = latestSessionID

	// Filter metrics for latest session
	var sessionMetrics []CoachingMetric
	for _, m := range metrics {
		if m.SessionID == latestSessionID {
			sessionMetrics = append(sessionMetrics, m)
		}
	}

	if len(sessionMetrics) == 0 {
		return resp
	}

	// Calculate session duration
	firstTimestamp, _ := time.Parse(time.RFC3339, sessionMetrics[0].Timestamp)
	lastTimestamp, _ := time.Parse(time.RFC3339, sessionMetrics[len(sessionMetrics)-1].Timestamp)
	resp.Session.Started = sessionMetrics[0].Timestamp
	resp.Session.DurationMinutes = int(lastTimestamp.Sub(firstTimestamp).Minutes())

	// Aggregate by metric type (use latest value)
	metricValues := make(map[string]float64)
	for _, m := range sessionMetrics {
		metricValues[m.Type] = m.Value
		// Track last coaching time (when metric was written)
		resp.LastCoachingTime = m.Timestamp
	}

	// Calculate overall health status based on aggregated metrics
	// Thresholds: good = all metrics good, warning = any warning, poor = any poor
	goodCount := 0
	warningCount := 0
	poorCount := 0

	// Action ratio
	if val, ok := metricValues["action_ratio"]; ok {
		if val >= 0.5 {
			goodCount++
		} else if val >= 0.3 {
			warningCount++
		} else {
			poorCount++
		}
	}

	// Analysis paralysis
	if val, ok := metricValues["analysis_paralysis"]; ok {
		if val < 1 {
			goodCount++
		} else if val < 3 {
			warningCount++
		} else {
			poorCount++
		}
	}

	// Determine overall status
	if poorCount > 0 {
		resp.OverallStatus = "poor"
		resp.StatusMessage = "Orchestrator doing worker work"
	} else if warningCount > 0 {
		resp.OverallStatus = "warning"
		resp.StatusMessage = "Orchestrator may be stuck - check in"
	} else if goodCount > 0 {
		resp.OverallStatus = "good"
		resp.StatusMessage = "Orchestrator delegating well"
	} else {
		resp.OverallStatus = "good"
		resp.StatusMessage = "No behavioral patterns detected yet"
	}

	return resp
}

// handleCoaching serves the /api/coaching endpoint.
func handleCoaching(w http.ResponseWriter, r *http.Request) {
	// Read last 100 metrics
	metrics, err := readCoachingMetrics(100)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read metrics: %v", err), http.StatusInternalServerError)
		return
	}

	// Aggregate and respond
	resp := aggregateMetrics(metrics)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}
