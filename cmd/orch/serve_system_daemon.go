// serve_system_daemon.go provides the HTTP handler for daemon status reporting.
// Extracted from serve_system.go — handles /api/daemon endpoint.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
)

// DaemonAPIResponse is the JSON structure returned by /api/daemon.
type DaemonAPIResponse struct {
	Running       bool   `json:"running"`                   // Whether the daemon is currently running
	Status        string `json:"status,omitempty"`          // "running", "stalled", or empty if not running
	LastPoll      string `json:"last_poll,omitempty"`       // ISO 8601 timestamp of last poll
	LastPollAgo   string `json:"last_poll_ago,omitempty"`   // Human-readable time since last poll
	LastSpawn     string `json:"last_spawn,omitempty"`      // ISO 8601 timestamp of last spawn
	LastSpawnAgo  string `json:"last_spawn_ago,omitempty"`  // Human-readable time since last spawn
	ReadyCount    int    `json:"ready_count"`               // Number of issues ready to process
	CapacityMax   int    `json:"capacity_max"`              // Maximum concurrent agents
	CapacityUsed  int    `json:"capacity_used"`             // Currently active agents
	CapacityFree  int    `json:"capacity_free"`             // Available slots for spawning
	IssuesPerHour int    `json:"issues_per_hour,omitempty"` // Approximate processing rate (future)

	// Utilization metrics - tracks daemon vs manual spawn ratio to surface triage discipline
	Utilization *DaemonUtilizationMetrics `json:"utilization,omitempty"`

	// Verification holds verification tracking information for gate visibility
	Verification *DaemonVerificationStatus `json:"verification,omitempty"`
}

// DaemonUtilizationMetrics tracks the ratio of daemon-spawned vs manual-spawned agents.
// This surfaces where triage discipline is slipping (manual spawns bypassing daemon workflow).
type DaemonUtilizationMetrics struct {
	TotalSpawns     int     `json:"total_spawns"`      // Total spawns in analysis window
	DaemonSpawns    int     `json:"daemon_spawns"`     // Spawns triggered by daemon
	ManualSpawns    int     `json:"manual_spawns"`     // Spawns triggered manually (not by daemon)
	DaemonSpawnRate float64 `json:"daemon_spawn_rate"` // % of spawns from daemon (higher is better)
	TriageBypassed  int     `json:"triage_bypassed"`   // Count of spawns that bypassed triage
	TriageSlipRate  float64 `json:"triage_slip_rate"`  // % of spawns that bypassed triage (lower is better)
	AutoCompletions int     `json:"auto_completions"`  // Count of daemon auto-completions
	AnalysisPeriod  string  `json:"analysis_period"`   // Time window description (e.g., "Last 7 days")
	DaysAnalyzed    int     `json:"days_analyzed"`     // Number of days in analysis window
}

// DaemonVerificationStatus tracks verification gate state for dashboard visibility.
type DaemonVerificationStatus struct {
	IsPaused                     bool   `json:"is_paused"`                       // Whether daemon is paused due to verification threshold
	CompletionsSinceVerification int    `json:"completions_since_verification"`  // Count of auto-completions since last verification
	Threshold                    int    `json:"threshold"`                       // Maximum auto-completions before pausing
	RemainingBeforePause         int    `json:"remaining_before_pause"`          // Completions allowed before pause
	LastVerification             string `json:"last_verification,omitempty"`     // ISO 8601 timestamp of last verification
	LastVerificationAgo          string `json:"last_verification_ago,omitempty"` // Human-readable time since last verification
}

// handleDaemon returns the daemon status from ~/.orch/daemon-status.json.
// If the daemon is not running (file doesn't exist), returns running: false.
// Also includes utilization metrics computed from events.jsonl.
func handleDaemon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse optional days parameter (default: 7)
	days := 7
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 90 {
			days = d
		}
	}

	resp := DaemonAPIResponse{
		Running: false,
	}

	// Try to read daemon status file (validates PID liveness to detect stale files)
	status, err := daemon.ReadValidatedStatusFile()
	if err == nil && status != nil {
		resp.Running = true
		resp.Status = status.Status
		resp.ReadyCount = status.ReadyCount
		resp.CapacityMax = status.Capacity.Max
		resp.CapacityUsed = status.Capacity.Active
		resp.CapacityFree = status.Capacity.Available

		// Format timestamps
		if !status.LastPoll.IsZero() {
			resp.LastPoll = status.LastPoll.Format(time.RFC3339)
			resp.LastPollAgo = formatDurationAgo(time.Since(status.LastPoll))
		}
		if !status.LastSpawn.IsZero() {
			resp.LastSpawn = status.LastSpawn.Format(time.RFC3339)
			resp.LastSpawnAgo = formatDurationAgo(time.Since(status.LastSpawn))
		}

		// Populate verification status for gate visibility
		if status.Verification != nil {
			resp.Verification = &DaemonVerificationStatus{
				IsPaused:                     status.Verification.IsPaused,
				CompletionsSinceVerification: status.Verification.CompletionsSinceVerification,
				Threshold:                    status.Verification.Threshold,
				RemainingBeforePause:         status.Verification.RemainingBeforePause,
			}
			if !status.Verification.LastVerification.IsZero() {
				resp.Verification.LastVerification = status.Verification.LastVerification.Format(time.RFC3339)
				resp.Verification.LastVerificationAgo = formatDurationAgo(time.Since(status.Verification.LastVerification))
			}
		}
	}

	// Get utilization metrics from events.jsonl
	utilization, err := daemon.GetUtilizationMetrics(days)
	if err == nil && utilization != nil {
		resp.Utilization = &DaemonUtilizationMetrics{
			TotalSpawns:     utilization.TotalSpawns,
			DaemonSpawns:    utilization.DaemonSpawns,
			ManualSpawns:    utilization.ManualSpawns,
			DaemonSpawnRate: utilization.DaemonSpawnRate,
			TriageBypassed:  utilization.TriageBypassed,
			TriageSlipRate:  utilization.TriageSlipRate,
			AutoCompletions: utilization.AutoCompletions,
			AnalysisPeriod:  utilization.AnalysisPeriod,
			DaysAnalyzed:    utilization.DaysAnalyzed,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode daemon status: %v", err), http.StatusInternalServerError)
		return
	}
}

// formatDurationAgo formats a duration into a human-readable "X ago" string.
func formatDurationAgo(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}
