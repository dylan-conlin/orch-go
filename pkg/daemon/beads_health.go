package daemon

import (
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/health"
)

// BeadsHealthResult contains the result of a beads health snapshot collection.
type BeadsHealthResult struct {
	OpenIssues    int
	BlockedIssues int
	StaleIssues   int
	BloatedFiles  int
	FixFeatRatio  float64

	// Error is set if collection failed.
	Error error

	// Message is a human-readable summary.
	Message string

	// RawSnapshot preserves the full health snapshot from collection.
	// When set, Store() uses this instead of reconstructing a partial snapshot.
	// This prevents field loss (HotspotCount, GateCoverage, TotalSourceFiles, etc.)
	// that caused 78% of stored snapshots to be incomplete.
	RawSnapshot *health.Snapshot
}

// BeadsHealthSnapshot is a point-in-time snapshot for the daemon status file.
type BeadsHealthSnapshot struct {
	OpenIssues    int       `json:"open_issues"`
	BlockedIssues int       `json:"blocked_issues"`
	StaleIssues   int       `json:"stale_issues"`
	BloatedFiles  int       `json:"bloated_files"`
	FixFeatRatio  float64   `json:"fix_feat_ratio"`
	LastCheck     time.Time `json:"last_check"`
}

// Snapshot converts a BeadsHealthResult to a dashboard-ready snapshot.
func (r *BeadsHealthResult) Snapshot() BeadsHealthSnapshot {
	return BeadsHealthSnapshot{
		OpenIssues:    r.OpenIssues,
		BlockedIssues: r.BlockedIssues,
		StaleIssues:   r.StaleIssues,
		BloatedFiles:  r.BloatedFiles,
		FixFeatRatio:  r.FixFeatRatio,
		LastCheck:     time.Now(),
	}
}

// BeadsHealthService provides beads health collection and storage.
type BeadsHealthService interface {
	Collect() (*BeadsHealthResult, error)
	Store(result *BeadsHealthResult) error
}

// NewDefaultBeadsHealthService creates a production BeadsHealthService.
// The collector function should return a health.Snapshot (typically collectHealthSnapshot from doctor_health.go).
func NewDefaultBeadsHealthService(collector func() health.Snapshot, store *health.Store) BeadsHealthService {
	return &defaultBeadsHealthService{collector: collector, store: store}
}

// defaultBeadsHealthService is the production implementation.
// It delegates to orch doctor --health snapshot collection and the health.Store.
type defaultBeadsHealthService struct {
	// collector is a function that collects health metrics.
	// In production, this calls collectHealthSnapshot() from doctor_health.go.
	collector func() health.Snapshot
	store     *health.Store
}

func (s *defaultBeadsHealthService) Collect() (*BeadsHealthResult, error) {
	snap := s.collector()
	return &BeadsHealthResult{
		OpenIssues:    snap.OpenIssues,
		BlockedIssues: snap.BlockedIssues,
		StaleIssues:   snap.StaleIssues,
		BloatedFiles:  snap.BloatedFiles,
		FixFeatRatio:  snap.FixFeatRatio,
		RawSnapshot:   &snap,
	}, nil
}

func (s *defaultBeadsHealthService) Store(result *BeadsHealthResult) error {
	// Use the full snapshot from collection when available.
	// This preserves all fields (HotspotCount, GateCoverage, TotalSourceFiles, etc.)
	// that were previously lost when reconstructing from BeadsHealthResult's 5 fields.
	if result.RawSnapshot != nil {
		snap := *result.RawSnapshot
		if snap.Timestamp.IsZero() {
			snap.Timestamp = time.Now()
		}
		return s.store.Append(snap)
	}
	// Fallback for mock/test scenarios without RawSnapshot.
	snap := health.Snapshot{
		Timestamp:     time.Now(),
		OpenIssues:    result.OpenIssues,
		BlockedIssues: result.BlockedIssues,
		StaleIssues:   result.StaleIssues,
		BloatedFiles:  result.BloatedFiles,
		FixFeatRatio:  result.FixFeatRatio,
	}
	return s.store.Append(snap)
}

// ShouldRunBeadsHealth returns true if periodic beads health collection should run.
func (d *Daemon) ShouldRunBeadsHealth() bool {
	return d.Scheduler.IsDue(TaskBeadsHealth)
}

// RunPeriodicBeadsHealth collects beads health metrics and stores a snapshot.
// Returns the result if the check was run, or nil if it wasn't due.
func (d *Daemon) RunPeriodicBeadsHealth() *BeadsHealthResult {
	if !d.ShouldRunBeadsHealth() {
		return nil
	}

	svc := d.BeadsHealth
	if svc == nil {
		return &BeadsHealthResult{
			Error:   fmt.Errorf("no BeadsHealth service configured"),
			Message: "Beads health: no service configured",
		}
	}

	result, err := svc.Collect()
	if err != nil {
		return &BeadsHealthResult{
			Error:   err,
			Message: fmt.Sprintf("Beads health collection failed: %v", err),
		}
	}

	// Store the snapshot
	if err := svc.Store(result); err != nil {
		result.Error = err
		result.Message = fmt.Sprintf("Beads health: collected but store failed: %v", err)
	} else {
		result.Message = fmt.Sprintf("Beads health: open=%d blocked=%d stale=%d bloated=%d fix:feat=%.1f",
			result.OpenIssues, result.BlockedIssues, result.StaleIssues, result.BloatedFiles, result.FixFeatRatio)
	}

	d.Scheduler.MarkRun(TaskBeadsHealth)

	return result
}

// LastBeadsHealthTime returns when beads health was last checked.
func (d *Daemon) LastBeadsHealthTime() time.Time {
	return d.Scheduler.LastRunTime(TaskBeadsHealth)
}
