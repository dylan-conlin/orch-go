// Package stability provides automatic stability measurement for Phase 3 reliability tracking.
// Records health snapshots and manual recovery interventions to ~/.orch/stability.jsonl.
// Computes clean-session streaks: time since last manual intervention.
package stability

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"
)

// Entry types in stability.jsonl.
const (
	TypeSnapshot     = "snapshot"
	TypeIntervention = "intervention"
)

// Intervention sources — what triggered the streak-breaking event.
const (
	SourceManualRecovery = "manual_recovery" // Service recovered without daemon action
	SourceAgentAbandoned = "agent_abandoned" // Agent abandoned via orch abandon
	SourceDoctorFix      = "doctor_fix"      // Manual orch doctor --fix invocation
)

// Entry is a single line in stability.jsonl.
type Entry struct {
	Type     string          `json:"type"`
	Ts       int64           `json:"ts"`
	Healthy  *bool           `json:"healthy,omitempty"`  // snapshot only
	Services map[string]bool `json:"services,omitempty"` // snapshot: service_name -> running
	Source   string          `json:"source,omitempty"`   // intervention only
	Detail   string          `json:"detail,omitempty"`   // intervention only
	Affected []string        `json:"affected,omitempty"` // intervention: affected services
	BeadsID  string          `json:"beads_id,omitempty"` // intervention: beads ID if applicable
}

// Report is the computed stability report.
type Report struct {
	CurrentStreak    time.Duration `json:"current_streak_seconds"`
	TargetDuration   time.Duration `json:"target_duration_seconds"`
	ProgressPercent  float64       `json:"progress_percent"`
	Interventions    []Entry       `json:"interventions"`
	SnapshotsTotal   int           `json:"snapshots_total"`
	SnapshotsHealthy int           `json:"snapshots_healthy"`
	HealthPercent    float64       `json:"health_percent"`
	FirstSnapshot    time.Time     `json:"first_snapshot"`
	LastIntervention *time.Time    `json:"last_intervention,omitempty"`
	HasData          bool          `json:"has_data"`
}

// DefaultPath returns the default path to stability.jsonl.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orch/stability.jsonl"
	}
	return filepath.Join(home, ".orch", "stability.jsonl")
}

// Recorder writes stability entries to a JSONL file.
type Recorder struct {
	path string
}

// NewRecorder creates a new stability recorder.
func NewRecorder(path string) *Recorder {
	return &Recorder{path: path}
}

// RecordSnapshot appends a health snapshot entry.
func (r *Recorder) RecordSnapshot(healthy bool, services map[string]bool) error {
	return r.write(Entry{
		Type:     TypeSnapshot,
		Ts:       time.Now().Unix(),
		Healthy:  &healthy,
		Services: services,
	})
}

// RecordIntervention appends a manual recovery intervention entry.
func (r *Recorder) RecordIntervention(source, detail string, affected []string, beadsID string) error {
	return r.write(Entry{
		Type:     TypeIntervention,
		Ts:       time.Now().Unix(),
		Source:   source,
		Detail:   detail,
		Affected: affected,
		BeadsID:  beadsID,
	})
}

func (r *Recorder) write(entry Entry) error {
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create stability directory: %w", err)
	}

	f, err := os.OpenFile(r.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open stability file: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal stability entry: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write stability entry: %w", err)
	}
	return nil
}

// TargetDuration is the Phase 3 success target: 1 week without manual intervention.
const TargetDuration = 7 * 24 * time.Hour

// isInfrastructureIntervention returns true if the intervention source represents
// an infrastructure failure (vs agent-level hygiene operation).
//
// Infrastructure interventions reset the crash-free streak because they indicate
// system stability problems. Agent interventions (like abandoning stuck agents)
// are routine hygiene and don't reset the streak.
func isInfrastructureIntervention(source string) bool {
	switch source {
	case SourceManualRecovery, SourceDoctorFix:
		return true
	case SourceAgentAbandoned:
		return false
	default:
		// Unknown sources default to infrastructure (fail-safe)
		return true
	}
}

// ComputeReport reads stability.jsonl and computes the stability report.
func ComputeReport(path string, days int) (*Report, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Report{
				TargetDuration: TargetDuration,
				HasData:        false,
			}, nil
		}
		return nil, fmt.Errorf("failed to open stability file: %w", err)
	}
	defer f.Close()

	now := time.Now()
	cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)

	report := &Report{
		TargetDuration: TargetDuration,
		HasData:        false,
	}

	var firstSnapshotTs int64
	var lastInfrastructureInterventionTs int64

	scanner := bufio.NewScanner(f)
	// Increase buffer size for safety
	scanner.Buffer(make([]byte, 0, 64*1024), 64*1024)

	for scanner.Scan() {
		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // skip malformed lines
		}

		entryTime := time.Unix(entry.Ts, 0)

		// Track first snapshot regardless of cutoff (for streak calculation)
		if entry.Type == TypeSnapshot && (firstSnapshotTs == 0 || entry.Ts < firstSnapshotTs) {
			firstSnapshotTs = entry.Ts
		}

		// Track last INFRASTRUCTURE intervention regardless of cutoff (for streak calculation)
		// Agent interventions (like agent_abandoned) don't reset the streak
		if entry.Type == TypeIntervention && isInfrastructureIntervention(entry.Source) {
			if entry.Ts > lastInfrastructureInterventionTs {
				lastInfrastructureInterventionTs = entry.Ts
			}
		}

		// Only include in summary stats if within the reporting window
		if entryTime.Before(cutoff) {
			continue
		}

		report.HasData = true

		switch entry.Type {
		case TypeSnapshot:
			report.SnapshotsTotal++
			if entry.Healthy != nil && *entry.Healthy {
				report.SnapshotsHealthy++
			}
		case TypeIntervention:
			report.Interventions = append(report.Interventions, entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read stability file: %w", err)
	}

	// Also count data from before cutoff
	if firstSnapshotTs > 0 {
		report.HasData = true
		report.FirstSnapshot = time.Unix(firstSnapshotTs, 0)
	}

	// Compute streak: time since last INFRASTRUCTURE intervention
	if lastInfrastructureInterventionTs > 0 {
		t := time.Unix(lastInfrastructureInterventionTs, 0)
		report.LastIntervention = &t
		report.CurrentStreak = now.Sub(t)
	} else if firstSnapshotTs > 0 {
		// No infrastructure interventions ever — streak is since first snapshot
		report.CurrentStreak = now.Sub(time.Unix(firstSnapshotTs, 0))
	}

	// Compute health percentage
	if report.SnapshotsTotal > 0 {
		report.HealthPercent = float64(report.SnapshotsHealthy) / float64(report.SnapshotsTotal) * 100
	}

	// Compute progress percentage (capped at 100%)
	if report.CurrentStreak > 0 {
		report.ProgressPercent = math.Min(float64(report.CurrentStreak)/float64(TargetDuration)*100, 100)
	}

	return report, nil
}

// FormatDuration formats a duration as "Xd Yh Zm".
func FormatDuration(d time.Duration) string {
	if d <= 0 {
		return "0m"
	}

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

// ProgressBar returns a text progress bar string.
func ProgressBar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}
	bar := make([]byte, width)
	for i := range bar {
		if i < filled {
			bar[i] = '#'
		} else {
			bar[i] = '-'
		}
	}
	return string(bar)
}
