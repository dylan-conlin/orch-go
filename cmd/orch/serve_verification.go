package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
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

// overrideTrendCache caches CalculateOverrideTrend results to avoid re-reading
// the entire events.jsonl file (62MB, 181K lines, ~750ms) on every request.
// Override trends change very slowly (counts events over 7+ days), so a 60s TTL
// eliminates >98% of file reads with negligible staleness.
type overrideTrendCache struct {
	mu        sync.RWMutex
	trend     *verify.OverrideTrend
	fetchedAt time.Time
	ttl       time.Duration
	days      int // track which window was cached
}

var globalOverrideTrendCache = &overrideTrendCache{
	ttl: 60 * time.Second,
}

func (c *overrideTrendCache) get(days int) (*verify.OverrideTrend, error) {
	c.mu.RLock()
	if c.trend != nil && c.days == days && time.Since(c.fetchedAt) < c.ttl {
		result := c.trend
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	trend, err := verify.CalculateOverrideTrend(days)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.trend = trend
	c.days = days
	c.fetchedAt = time.Now()
	c.mu.Unlock()

	return trend, nil
}

func (c *overrideTrendCache) invalidate() {
	c.mu.Lock()
	c.trend = nil
	c.fetchedAt = time.Time{}
	c.mu.Unlock()
}

// unverifiedCountCache caches CountUnverifiedWorkWithDir results to avoid
// shelling out to beads CLI on every /api/verification request.
type unverifiedCountCache struct {
	mu        sync.RWMutex
	count     int
	err       error
	fetchedAt time.Time
	ttl       time.Duration
	dir       string
}

var globalUnverifiedCountCache = &unverifiedCountCache{
	ttl: 30 * time.Second,
}

func (c *unverifiedCountCache) get(projectDir string) (int, error) {
	c.mu.RLock()
	if c.dir == projectDir && time.Since(c.fetchedAt) < c.ttl && c.err == nil {
		count := c.count
		c.mu.RUnlock()
		return count, nil
	}
	c.mu.RUnlock()

	count, err := verify.CountUnverifiedWorkWithDir(projectDir)

	c.mu.Lock()
	c.count = count
	c.err = err
	c.dir = projectDir
	c.fetchedAt = time.Now()
	c.mu.Unlock()

	return count, err
}

func (c *unverifiedCountCache) invalidate() {
	c.mu.Lock()
	c.count = 0
	c.err = nil
	c.fetchedAt = time.Time{}
	c.mu.Unlock()
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

	// Unverified count (scoped by project when provided) — cached to avoid beads CLI spawning
	count, err := globalUnverifiedCountCache.get(projectDir)
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

	// Override trend (verification bypasses) — cached to avoid re-reading 62MB events.jsonl
	trend, err := globalOverrideTrendCache.get(trendDays)
	if err == nil && trend != nil {
		resp.OverrideTrend = trend
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode verification status: %v", err), http.StatusInternalServerError)
		return
	}
}
