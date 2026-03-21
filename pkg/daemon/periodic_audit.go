package daemon

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
)

// AuditSelectResult holds the outcome of a periodic audit selection.
type AuditSelectResult struct {
	Error    error
	Message  string
	Selected []beads.Issue
}

// AuditSelectService abstracts issue listing and labeling for testability.
type AuditSelectService interface {
	// RecentClosedIssues returns issues closed within the given window,
	// excluding those already labeled for audit.
	RecentClosedIssues(window time.Duration) ([]beads.Issue, error)
	// ForcedBeadsIDs returns the set of beads IDs that were auto-completed
	// (forced) based on event data.
	ForcedBeadsIDs() (map[string]bool, error)
	// LabelForAudit adds the audit:deep-review label to the given issue.
	LabelForAudit(issueID string) error
}

// RunPeriodicAuditSelect randomly selects completed issues for quality audit.
// Selection is weighted toward auto-completed work per AuditAutoCompleteWeight.
// Returns nil if not due.
func (d *Daemon) RunPeriodicAuditSelect() *AuditSelectResult {
	if !d.Scheduler.IsDue(TaskAuditSelect) {
		return nil
	}

	svc := d.AuditSelect
	if svc == nil {
		svc = &defaultAuditSelectService{}
	}

	result := runAuditSelection(svc, d.Config.AuditSelectCount, d.Config.AuditAutoCompleteWeight, d.Config.AuditSelectInterval)
	d.Scheduler.MarkRun(TaskAuditSelect)
	return result
}

func runAuditSelection(svc AuditSelectService, selectCount int, autoCompleteWeight float64, window time.Duration) *AuditSelectResult {
	recent, err := svc.RecentClosedIssues(window)
	if err != nil {
		return &AuditSelectResult{
			Error:   err,
			Message: fmt.Sprintf("failed to list closed issues: %v", err),
		}
	}

	if len(recent) == 0 {
		return &AuditSelectResult{
			Message: "no eligible issues for audit selection",
		}
	}

	// Split into auto-completed and other pools
	forcedIDs, err := svc.ForcedBeadsIDs()
	if err != nil {
		// Non-fatal: fall back to uniform selection
		forcedIDs = nil
	}

	var autoPool, otherPool []beads.Issue
	for _, issue := range recent {
		if forcedIDs[issue.ID] {
			autoPool = append(autoPool, issue)
		} else {
			otherPool = append(otherPool, issue)
		}
	}

	selected := weightedSelection(autoPool, otherPool, selectCount, autoCompleteWeight)

	// Label selected issues
	var labeled []beads.Issue
	for _, issue := range selected {
		if err := svc.LabelForAudit(issue.ID); err != nil {
			continue // skip issues we can't label
		}
		labeled = append(labeled, issue)
	}

	if len(labeled) == 0 {
		return &AuditSelectResult{
			Message: "audit selection found candidates but failed to label any",
		}
	}

	return &AuditSelectResult{
		Selected: labeled,
		Message:  fmt.Sprintf("selected %d issue(s) for audit deep review", len(labeled)),
	}
}

// weightedSelection picks selectCount items using weighted sampling.
// autoWeight fraction comes from autoPool, the rest from otherPool.
// Falls back gracefully when pools are too small.
func weightedSelection(autoPool, otherPool []beads.Issue, selectCount int, autoWeight float64) []beads.Issue {
	if autoWeight < 0 {
		autoWeight = 0
	}
	if autoWeight > 1 {
		autoWeight = 1
	}

	// Calculate target counts from each pool
	autoTarget := int(float64(selectCount) * autoWeight)
	otherTarget := selectCount - autoTarget

	// If auto pool is empty, take all from other pool (and vice versa)
	if len(autoPool) == 0 {
		autoTarget = 0
		otherTarget = selectCount
	}
	if len(otherPool) == 0 {
		otherTarget = 0
		autoTarget = selectCount
	}

	// Cap at pool size
	if autoTarget > len(autoPool) {
		overflow := autoTarget - len(autoPool)
		autoTarget = len(autoPool)
		otherTarget += overflow
	}
	if otherTarget > len(otherPool) {
		overflow := otherTarget - len(otherPool)
		otherTarget = len(otherPool)
		autoTarget += overflow
		if autoTarget > len(autoPool) {
			autoTarget = len(autoPool)
		}
	}

	var result []beads.Issue
	if autoTarget > 0 {
		picked, _ := cryptoRandPick(autoPool, autoTarget)
		result = append(result, picked...)
	}
	if otherTarget > 0 {
		picked, _ := cryptoRandPick(otherPool, otherTarget)
		result = append(result, picked...)
	}
	return result
}

// cryptoRandPick selects n items from a slice using crypto/rand Fisher-Yates shuffle.
func cryptoRandPick(issues []beads.Issue, n int) ([]beads.Issue, error) {
	if len(issues) <= n {
		return issues, nil
	}

	pool := make([]beads.Issue, len(issues))
	copy(pool, issues)

	for i := len(pool) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return nil, fmt.Errorf("crypto rand: %w", err)
		}
		pool[i], pool[j.Int64()] = pool[j.Int64()], pool[i]
	}
	return pool[:n], nil
}

// defaultAuditSelectService is the production implementation using beads CLI and events.
type defaultAuditSelectService struct{}

func (s *defaultAuditSelectService) RecentClosedIssues(window time.Duration) ([]beads.Issue, error) {
	issues, err := beads.FallbackList("closed", "")
	if err != nil {
		return nil, fmt.Errorf("listing closed issues: %w", err)
	}

	cutoff := time.Now().Add(-window)
	var recent []beads.Issue
	for _, issue := range issues {
		if issue.ClosedAt == "" {
			continue
		}
		closedAt, parseErr := time.Parse(time.RFC3339Nano, issue.ClosedAt)
		if parseErr != nil {
			closedAt, parseErr = time.Parse(time.RFC3339, issue.ClosedAt)
			if parseErr != nil {
				continue
			}
		}
		if closedAt.After(cutoff) {
			if hasAuditLabel(issue.Labels) {
				continue
			}
			recent = append(recent, issue)
		}
	}
	return recent, nil
}

func hasAuditLabel(labels []string) bool {
	for _, l := range labels {
		if l == "audit:deep-review" {
			return true
		}
	}
	return false
}

func (s *defaultAuditSelectService) ForcedBeadsIDs() (map[string]bool, error) {
	return scanForcedCompletions(events.DefaultLogPath())
}

func (s *defaultAuditSelectService) LabelForAudit(issueID string) error {
	return beads.FallbackAddLabel(issueID, "audit:deep-review", "")
}

// scanForcedCompletions reads event files and returns the set of beads IDs
// that were completed with forced=true (auto-completed by daemon).
func scanForcedCompletions(eventsPath string) (map[string]bool, error) {
	forced := make(map[string]bool)
	err := events.ScanEventsFromPath(eventsPath, time.Time{}, time.Time{}, func(event events.Event) {
		if event.Type != events.EventTypeAgentCompleted {
			return
		}
		isForced, _ := event.Data["forced"].(bool)
		beadsID, _ := event.Data["beads_id"].(string)
		if isForced && beadsID != "" {
			forced[beadsID] = true
		}
	})
	return forced, err
}
