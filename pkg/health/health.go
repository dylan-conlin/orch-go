// Package health provides time-series health monitoring for orch doctor.
// It tracks system invariants over time and alerts when trends cross thresholds.
package health

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"
)

// Snapshot represents a single point-in-time health measurement.
type Snapshot struct {
	Timestamp      time.Time `json:"timestamp"`
	OpenIssues     int       `json:"open_issues"`
	BlockedIssues  int       `json:"blocked_issues"`
	StaleIssues    int       `json:"stale_issues"`    // Open issues with no activity in 14+ days
	OrphanedIssues int       `json:"orphaned_issues"` // Open issues with no comments in 7+ days
	BloatedFiles   int       `json:"bloated_files"`   // Files over accretion threshold (800 lines)
	FixFeatRatio   float64   `json:"fix_feat_ratio"`  // Ratio of fix: to feat: commits (28-day window)
	FixCommits     int       `json:"fix_commits"`
	FeatCommits    int       `json:"feat_commits"`
}

// Trend represents the direction of a metric over time.
type Trend int

const (
	TrendStable Trend = iota
	TrendUp
	TrendDown
)

// String returns a display symbol for the trend direction.
func (t Trend) String() string {
	switch t {
	case TrendUp:
		return "↑"
	case TrendDown:
		return "↓"
	default:
		return "→"
	}
}

// TrendSet holds trend directions for all tracked metrics.
type TrendSet struct {
	OpenIssues     Trend
	BlockedIssues  Trend
	StaleIssues    Trend
	OrphanedIssues Trend
	BloatedFiles   Trend
	FixFeatRatio   Trend
}

// Alert represents a threshold crossing that needs attention.
type Alert struct {
	Metric  string `json:"metric"`
	Message string `json:"message"`
	Level   string `json:"level"` // "warn" or "critical"
}

// Report is the complete health analysis combining current state, trends, and alerts.
type Report struct {
	Current       Snapshot  `json:"current"`
	SnapshotCount int       `json:"snapshot_count"`
	Trends        TrendSet  `json:"trends"`
	Alerts        []Alert   `json:"alerts"`
	GeneratedAt   time.Time `json:"generated_at"`
}

// Store handles persistence of health snapshots as JSONL.
type Store struct {
	path string
}

// NewStore creates a new store at the given file path.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// Append writes a snapshot to the JSONL file.
func (s *Store) Append(snap Snapshot) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write snapshot: %w", err)
	}

	return nil
}

// ReadAll reads all snapshots from the JSONL file.
func (s *Store) ReadAll() ([]Snapshot, error) {
	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	var snapshots []Snapshot
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var snap Snapshot
		if err := json.Unmarshal(line, &snap); err != nil {
			continue // skip malformed lines
		}
		snapshots = append(snapshots, snap)
	}

	return snapshots, scanner.Err()
}

// ReadRecent reads the last n snapshots from the file.
func (s *Store) ReadRecent(n int) ([]Snapshot, error) {
	all, err := s.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(all) <= n {
		return all, nil
	}
	return all[len(all)-n:], nil
}

// ComputeTrend determines the direction of a series of values.
// Uses simple linear regression slope to determine direction.
func ComputeTrend(values []float64) Trend {
	n := len(values)
	if n < 2 {
		return TrendStable
	}

	// Simple linear regression: compute slope
	var sumX, sumY, sumXY, sumX2 float64
	for i, v := range values {
		x := float64(i)
		sumX += x
		sumY += v
		sumXY += x * v
		sumX2 += x * x
	}

	nf := float64(n)
	denominator := nf*sumX2 - sumX*sumX
	if denominator == 0 {
		return TrendStable
	}

	slope := (nf*sumXY - sumX*sumY) / denominator

	// Normalize slope by the mean to get relative change
	mean := sumY / nf
	if mean == 0 {
		if slope > 0.1 {
			return TrendUp
		}
		if slope < -0.1 {
			return TrendDown
		}
		return TrendStable
	}

	relativeSlope := slope / math.Abs(mean)

	// Threshold: 5% relative change per step is considered a trend
	if relativeSlope > 0.05 {
		return TrendUp
	}
	if relativeSlope < -0.05 {
		return TrendDown
	}
	return TrendStable
}

// GenerateReport creates a health report from a series of snapshots.
func GenerateReport(snapshots []Snapshot) Report {
	report := Report{
		SnapshotCount: len(snapshots),
		GeneratedAt:   time.Now(),
		Alerts:        []Alert{},
	}

	if len(snapshots) == 0 {
		return report
	}

	// Current = most recent snapshot
	report.Current = snapshots[len(snapshots)-1]

	// Extract value series for trend analysis
	openIssues := make([]float64, len(snapshots))
	blockedIssues := make([]float64, len(snapshots))
	staleIssues := make([]float64, len(snapshots))
	orphanedIssues := make([]float64, len(snapshots))
	bloatedFiles := make([]float64, len(snapshots))
	fixFeatRatio := make([]float64, len(snapshots))

	for i, s := range snapshots {
		openIssues[i] = float64(s.OpenIssues)
		blockedIssues[i] = float64(s.BlockedIssues)
		staleIssues[i] = float64(s.StaleIssues)
		orphanedIssues[i] = float64(s.OrphanedIssues)
		bloatedFiles[i] = float64(s.BloatedFiles)
		fixFeatRatio[i] = s.FixFeatRatio
	}

	report.Trends = TrendSet{
		OpenIssues:     ComputeTrend(openIssues),
		BlockedIssues:  ComputeTrend(blockedIssues),
		StaleIssues:    ComputeTrend(staleIssues),
		OrphanedIssues: ComputeTrend(orphanedIssues),
		BloatedFiles:   ComputeTrend(bloatedFiles),
		FixFeatRatio:   ComputeTrend(fixFeatRatio),
	}

	// Generate alerts based on current values and trends
	current := report.Current

	// Fix:feat ratio > 2.0 means maintenance mode
	if current.FixFeatRatio > 2.0 {
		report.Alerts = append(report.Alerts, Alert{
			Metric:  "fix_feat_ratio",
			Message: fmt.Sprintf("Fix:feat ratio is %.1f — system in maintenance mode (more fixes than features)", current.FixFeatRatio),
			Level:   "critical",
		})
	} else if current.FixFeatRatio > 1.5 {
		report.Alerts = append(report.Alerts, Alert{
			Metric:  "fix_feat_ratio",
			Message: fmt.Sprintf("Fix:feat ratio is %.1f — approaching maintenance mode", current.FixFeatRatio),
			Level:   "warn",
		})
	}

	// Stale issues trending up
	if report.Trends.StaleIssues == TrendUp && current.StaleIssues > 5 {
		report.Alerts = append(report.Alerts, Alert{
			Metric:  "stale_issues",
			Message: fmt.Sprintf("%d stale issues and trending up — backlog growing", current.StaleIssues),
			Level:   "warn",
		})
	}

	// Bloated files increasing
	if report.Trends.BloatedFiles == TrendUp {
		report.Alerts = append(report.Alerts, Alert{
			Metric:  "bloated_files",
			Message: fmt.Sprintf("%d bloated files and increasing — accretion boundaries being violated", current.BloatedFiles),
			Level:   "warn",
		})
	}

	// Blocked issues > 30% of open issues
	if current.OpenIssues > 0 {
		blockedPct := float64(current.BlockedIssues) / float64(current.OpenIssues) * 100
		if blockedPct > 30 {
			report.Alerts = append(report.Alerts, Alert{
				Metric:  "blocked_issues",
				Message: fmt.Sprintf("%.0f%% of open issues are blocked (%d/%d) — dependency bottleneck", blockedPct, current.BlockedIssues, current.OpenIssues),
				Level:   "warn",
			})
		}
	}

	// Orphaned issues > 10
	if current.OrphanedIssues > 10 {
		report.Alerts = append(report.Alerts, Alert{
			Metric:  "orphaned_issues",
			Message: fmt.Sprintf("%d orphaned issues (no activity in 7+ days) — need triage", current.OrphanedIssues),
			Level:   "warn",
		})
	}

	return report
}
