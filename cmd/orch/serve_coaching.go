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

// CoachingResponse is the API response format.
type CoachingResponse struct {
	Session struct {
		SessionID       string `json:"session_id"`
		Started         string `json:"started"`
		DurationMinutes int    `json:"duration_minutes"`
	} `json:"session"`
	Metrics map[string]struct {
		Value  float64 `json:"value"`
		Label  string  `json:"label"`
		Status string  `json:"status"` // good/warning/poor
	} `json:"metrics"`
	Coaching []string `json:"coaching"`
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

// aggregateMetrics aggregates metrics by session and calculates latest values.
func aggregateMetrics(metrics []CoachingMetric) CoachingResponse {
	resp := CoachingResponse{
		Metrics: make(map[string]struct {
			Value  float64 `json:"value"`
			Label  string  `json:"label"`
			Status string  `json:"status"`
		}),
		Coaching: []string{},
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
	}

	// Context ratio
	if val, ok := metricValues["context_ratio"]; ok {
		status := "poor"
		if val >= 0.7 {
			status = "good"
		} else if val >= 0.4 {
			status = "warning"
		}

		resp.Metrics["context_ratio"] = struct {
			Value  float64 `json:"value"`
			Label  string  `json:"label"`
			Status string  `json:"status"`
		}{
			Value:  val,
			Label:  "Context checks per spawn",
			Status: status,
		}

		// Generate coaching
		if status == "good" {
			resp.Coaching = append(resp.Coaching, fmt.Sprintf("✅ Good context-gathering ratio (%.2f)", val))
		} else if status == "warning" {
			resp.Coaching = append(resp.Coaching, fmt.Sprintf("⚠️  Consider more kb context checks before spawning (%.2f)", val))
		} else {
			resp.Coaching = append(resp.Coaching, fmt.Sprintf("❌ Low context-gathering ratio (%.2f) - use 'kb context' before spawns", val))
		}
	}

	// Action ratio
	if val, ok := metricValues["action_ratio"]; ok {
		status := "poor"
		if val >= 0.5 {
			status = "good"
		} else if val >= 0.3 {
			status = "warning"
		}

		resp.Metrics["action_ratio"] = struct {
			Value  float64 `json:"value"`
			Label  string  `json:"label"`
			Status string  `json:"status"`
		}{
			Value:  val,
			Label:  "Actions per reads",
			Status: status,
		}

		// Generate coaching
		if status == "good" {
			resp.Coaching = append(resp.Coaching, fmt.Sprintf("✅ Good action-to-read balance (%.2f)", val))
		} else if status == "warning" {
			resp.Coaching = append(resp.Coaching, fmt.Sprintf("⚠️  Low action ratio (%.2f) - consider more decisive action", val))
		} else {
			resp.Coaching = append(resp.Coaching, fmt.Sprintf("❌ Too many reads without actions (%.2f) - analysis paralysis?", val))
		}
	}

	// Analysis paralysis
	if val, ok := metricValues["analysis_paralysis"]; ok {
		status := "good"
		if val >= 3 {
			status = "poor"
		} else if val >= 1 {
			status = "warning"
		}

		resp.Metrics["analysis_paralysis"] = struct {
			Value  float64 `json:"value"`
			Label  string  `json:"label"`
			Status string  `json:"status"`
		}{
			Value:  val,
			Label:  "Tool repetition sequences",
			Status: status,
		}

		// Generate coaching
		if status == "good" {
			resp.Coaching = append(resp.Coaching, "✅ No analysis paralysis detected")
		} else if status == "warning" {
			resp.Coaching = append(resp.Coaching, fmt.Sprintf("⚠️  %d tool repetition sequence(s) detected", int(val)))
		} else {
			resp.Coaching = append(resp.Coaching, fmt.Sprintf("❌ %d tool repetition sequences - consider more decisive action", int(val)))
		}
	}

	// If no metrics yet, add placeholder
	if len(resp.Coaching) == 0 {
		resp.Coaching = append(resp.Coaching, "No behavioral metrics yet - continue working to generate coaching insights")
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
