package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// VerificationAPIResponse is the JSON structure returned by /api/verification.
type VerificationAPIResponse struct {
	UnverifiedCount int                   `json:"unverified_count"`
	HeartbeatAt     string                `json:"heartbeat_at,omitempty"`
	HeartbeatAgo    string                `json:"heartbeat_ago,omitempty"`
	DaemonPaused    bool                  `json:"daemon_paused"`
	DaemonRunning   bool                  `json:"daemon_running"`
	DaemonStatus    string                `json:"daemon_status,omitempty"`
	OverrideTrend   *verify.OverrideTrend `json:"override_trend,omitempty"`
	ProjectDir      string                `json:"project_dir,omitempty"`
	Error           string                `json:"error,omitempty"`
}

// handleVerification returns verification status for the dashboard.
// Query params:
//   - project_dir: Optional project directory to query for unverified count
//   - days: Optional days window for override trend (default: 7)
func handleVerification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectDir := r.URL.Query().Get("project_dir")

	// Parse optional days parameter for override trend
	trendDays := 7
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 90 {
			trendDays = d
		}
	}

	resp := VerificationAPIResponse{
		ProjectDir: projectDir,
	}

	// Unverified count (scoped by project when provided)
	count, err := verify.CountUnverifiedWorkWithDir(projectDir)
	if err != nil {
		resp.Error = fmt.Sprintf("Failed to count unverified work: %v", err)
	} else {
		resp.UnverifiedCount = count
	}

	// Heartbeat age (last human verification signal)
	heartbeatAt, err := daemon.ReadVerificationSignal()
	if err == nil && !heartbeatAt.IsZero() {
		resp.HeartbeatAt = heartbeatAt.Format(time.RFC3339)
		resp.HeartbeatAgo = formatDurationAgo(time.Since(heartbeatAt))
	}

	// Daemon pause state (validates PID liveness to detect stale files)
	status, err := daemon.ReadValidatedStatusFile()
	if err == nil && status != nil {
		resp.DaemonRunning = true
		resp.DaemonStatus = status.Status
		if status.Verification != nil {
			resp.DaemonPaused = status.Verification.IsPaused
		}
	}

	// Override trend (verification bypasses)
	trend, err := verify.CalculateOverrideTrend(trendDays)
	if err == nil && trend != nil {
		resp.OverrideTrend = trend
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode verification status: %v", err), http.StatusInternalServerError)
		return
	}
}
