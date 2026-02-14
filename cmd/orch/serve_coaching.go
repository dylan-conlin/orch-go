package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/coaching"
)

// WorkerHealthMetrics represents health signals for a worker session.
// These are different from orchestrator metrics (action_ratio, analysis_paralysis).
type WorkerHealthMetrics struct {
	SessionID string `json:"session_id"`
	// tool_failure_rate: consecutive tool failures (>=3 is warning)
	ToolFailureRate int `json:"tool_failure_rate"`
	// context_usage: estimated token usage percentage (>=80 is warning)
	ContextUsage int `json:"context_usage"`
	// time_in_phase: minutes since last phase change (>=15 is warning)
	TimeInPhase int `json:"time_in_phase"`
	// commit_gap: minutes since last commit (>=30 is warning)
	CommitGap int `json:"commit_gap"`
	// Derived health status: good/warning/critical
	HealthStatus string `json:"health_status"`
	// Last update timestamp
	LastUpdated string `json:"last_updated"`
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
	// Worker health metrics keyed by session ID
	WorkerHealth map[string]WorkerHealthMetrics `json:"worker_health,omitempty"`
}

// getCoachingMetricsPath returns the path to the coaching metrics file.
func getCoachingMetricsPath() string {
	return filepath.Join(os.Getenv("HOME"), ".orch", "coaching-metrics.jsonl")
}

// Worker health metric types
var workerHealthMetricTypes = map[string]bool{
	"tool_failure_rate": true,
	"context_usage":     true,
	"time_in_phase":     true,
	"commit_gap":        true,
}

// isWorkerHealthMetric checks if a metric type is a worker health metric
func isWorkerHealthMetric(metricType string) bool {
	return workerHealthMetricTypes[metricType]
}

// calculateWorkerHealthStatus derives overall health status from metrics
func calculateWorkerHealthStatus(health WorkerHealthMetrics) string {
	criticalCount := 0
	warningCount := 0

	// tool_failure_rate: >=5 is critical, >=3 is warning
	if health.ToolFailureRate >= 5 {
		criticalCount++
	} else if health.ToolFailureRate >= 3 {
		warningCount++
	}

	// context_usage: >=90 is critical, >=80 is warning
	if health.ContextUsage >= 90 {
		criticalCount++
	} else if health.ContextUsage >= 80 {
		warningCount++
	}

	// time_in_phase: >=30 is critical, >=15 is warning
	if health.TimeInPhase >= 30 {
		criticalCount++
	} else if health.TimeInPhase >= 15 {
		warningCount++
	}

	// commit_gap: >=60 is critical, >=30 is warning
	if health.CommitGap >= 60 {
		criticalCount++
	} else if health.CommitGap >= 30 {
		warningCount++
	}

	if criticalCount > 0 {
		return "critical"
	} else if warningCount > 0 {
		return "warning"
	}
	return "good"
}

// aggregateWorkerHealthMetrics aggregates worker health metrics by session
func aggregateWorkerHealthMetrics(metrics []coaching.Metric) map[string]WorkerHealthMetrics {
	result := make(map[string]WorkerHealthMetrics)

	for _, m := range metrics {
		if !isWorkerHealthMetric(m.Type) {
			continue
		}

		sessionID := m.SessionID
		if sessionID == "" {
			continue
		}

		// Get or create worker health entry
		health, exists := result[sessionID]
		if !exists {
			health = WorkerHealthMetrics{
				SessionID: sessionID,
			}
		}

		// Update the appropriate metric (use latest value)
		switch m.Type {
		case "tool_failure_rate":
			health.ToolFailureRate = int(m.Value)
		case "context_usage":
			health.ContextUsage = int(m.Value)
		case "time_in_phase":
			health.TimeInPhase = int(m.Value)
		case "commit_gap":
			health.CommitGap = int(m.Value)
		}

		// Track last update timestamp
		if health.LastUpdated == "" || m.Timestamp > health.LastUpdated {
			health.LastUpdated = m.Timestamp
		}

		result[sessionID] = health
	}

	// Calculate derived health status for each session
	for sessionID, health := range result {
		health.HealthStatus = calculateWorkerHealthStatus(health)
		result[sessionID] = health
	}

	return result
}

// aggregateMetrics aggregates metrics by session and calculates overall health status (Frame 2).
func aggregateMetrics(metrics []coaching.Metric) CoachingResponse {
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

	// Aggregate worker health metrics (for all sessions)
	resp.WorkerHealth = aggregateWorkerHealthMetrics(metrics)

	// Get latest session ID (for orchestrator metrics)
	latestSessionID := metrics[len(metrics)-1].SessionID
	resp.Session.SessionID = latestSessionID

	// Filter metrics for latest session (orchestrator metrics only)
	var sessionMetrics []coaching.Metric
	for _, m := range metrics {
		if m.SessionID == latestSessionID && !isWorkerHealthMetric(m.Type) {
			sessionMetrics = append(sessionMetrics, m)
		}
	}

	if len(sessionMetrics) == 0 {
		// Check if there are worker health metrics even without orchestrator metrics
		if len(resp.WorkerHealth) > 0 {
			resp.StatusMessage = "Worker health metrics active"
		}
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
	metrics, err := coaching.ReadMetrics(getCoachingMetricsPath(), 100)
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
