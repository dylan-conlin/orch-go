// Package daemon provides autonomous overnight processing capabilities.
// This file contains pattern detectors for the trigger scan system:
//
// Phase 1:
//   - recurring_bugs: detects issues that have been reworked 2+ times
//   - investigation_orphans: detects active investigations with no matching open issue
//   - thread_staleness: detects open threads older than 7 days without updates
//
// Phase 2:
//   - model_contradictions: detects unresolved probe contradictions in kb models
//   - hotspot_acceleration: detects files growing rapidly (>200 lines/30d)
//   - knowledge_decay: detects models with no recent probes (30d+)
//   - skill_performance_drift: detects skills whose success rate dropped significantly
package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/thread"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// DefaultTriggerDetectors returns the registered pattern detectors with default sources.
func DefaultTriggerDetectors() []PatternDetector {
	return []PatternDetector{
		&RecurringBugsDetector{Source: &defaultRecurringBugsSource{}},
		&InvestigationOrphansDetector{Source: &defaultInvestigationOrphansSource{}},
		&ThreadStalenessDetector{Source: &defaultThreadStalenessSource{}},
		&ModelContradictionsDetector{Source: &defaultModelContradictionsSource{}},
		// HotspotAccelerationDetector disabled — detector removed from rotation
		&KnowledgeDecayDetector{Source: &defaultKnowledgeDecaySource{}},
		&SkillPerformanceDriftDetector{Source: &defaultSkillPerformanceDriftSource{}},
	}
}

// --- Recurring Bugs Detector ---

// RecurringBugsSource provides I/O for the recurring bugs detector.
type RecurringBugsSource interface {
	// ListClosedIssuesWithRework returns issues that have been closed with rework labels
	// or have multiple completion events. Returns (issueID, reworkCount) pairs.
	ListClosedIssuesWithRework(minReworks int) ([]ReworkedIssue, error)
	// HasOpenIssue checks if there's already an open issue for this recurring bug.
	HasOpenIssue(issueID string) (bool, error)
}

// ReworkedIssue is an issue that has been reworked multiple times.
type ReworkedIssue struct {
	ID          string
	Title       string
	ReworkCount int
}

// RecurringBugsDetector detects issues that have been reworked 2+ times.
type RecurringBugsDetector struct {
	Source     RecurringBugsSource
	MinReworks int // Minimum reworks to trigger (default 2)
}

func (d *RecurringBugsDetector) Name() string { return "recurring_bugs" }

func (d *RecurringBugsDetector) Detect() ([]TriggerSuggestion, error) {
	if d.Source == nil {
		return nil, fmt.Errorf("recurring bugs source not configured")
	}
	minReworks := d.MinReworks
	if minReworks <= 0 {
		minReworks = 2
	}

	reworked, err := d.Source.ListClosedIssuesWithRework(minReworks)
	if err != nil {
		return nil, err
	}

	var suggestions []TriggerSuggestion
	for _, issue := range reworked {
		hasOpen, err := d.Source.HasOpenIssue(issue.ID)
		if err != nil || hasOpen {
			continue
		}

		suggestions = append(suggestions, TriggerSuggestion{
			Detector:    "recurring_bugs",
			Key:         issue.ID,
			Title:       fmt.Sprintf("Recurring bug: %s (%dx rework)", issue.Title, issue.ReworkCount),
			Description: fmt.Sprintf("Issue %s has been reworked %d times. This suggests a root cause that hasn't been addressed. Consider an investigation to find the underlying issue.", issue.ID, issue.ReworkCount),
			IssueType:   "bug",
			Priority:    2,
		})
	}
	return suggestions, nil
}

// --- Investigation Orphans Detector ---

// InvestigationOrphansSource provides I/O for the investigation orphans detector.
type InvestigationOrphansSource interface {
	// ListActiveInvestigations returns investigation files with Status: Active.
	ListActiveInvestigations() ([]OrphanedInvestigation, error)
	// HasOpenIssueForInvestigation checks if a beads issue references this investigation.
	HasOpenIssueForInvestigation(slug string) (bool, error)
}

// OrphanedInvestigation is an active investigation with no tracking issue.
type OrphanedInvestigation struct {
	Path string // File path
	Slug string // Investigation slug (from filename)
	Age  time.Duration
}

// InvestigationOrphansDetector detects active investigations with no matching beads issue.
type InvestigationOrphansDetector struct {
	Source InvestigationOrphansSource
	MaxAge time.Duration // Only report investigations older than this (default 3 days)
}

func (d *InvestigationOrphansDetector) Name() string { return "investigation_orphans" }

func (d *InvestigationOrphansDetector) Detect() ([]TriggerSuggestion, error) {
	if d.Source == nil {
		return nil, fmt.Errorf("investigation orphans source not configured")
	}

	maxAge := d.MaxAge
	if maxAge <= 0 {
		maxAge = 3 * 24 * time.Hour // 3 days
	}

	investigations, err := d.Source.ListActiveInvestigations()
	if err != nil {
		return nil, err
	}

	var suggestions []TriggerSuggestion
	for _, inv := range investigations {
		if inv.Age < maxAge {
			continue
		}

		hasOpen, err := d.Source.HasOpenIssueForInvestigation(inv.Slug)
		if err != nil || hasOpen {
			continue
		}

		suggestions = append(suggestions, TriggerSuggestion{
			Detector:    "investigation_orphans",
			Key:         inv.Slug,
			Title:       fmt.Sprintf("Orphaned investigation: %s (active %dd)", inv.Slug, int(inv.Age.Hours()/24)),
			Description: fmt.Sprintf("Investigation %s has Status: Active but no matching beads issue. Either complete/close the investigation or create an issue to track remaining work.", inv.Path),
			IssueType:   "task",
			Priority:    3,
		})
	}
	return suggestions, nil
}

// --- Thread Staleness Detector ---

// ThreadStalenessSource provides I/O for the thread staleness detector.
type ThreadStalenessSource interface {
	// ListOpenThreads returns open threads with their last update date.
	ListOpenThreads() ([]StaleThread, error)
}

// StaleThread is a thread that hasn't been updated recently.
type StaleThread struct {
	Slug    string
	Title   string
	Updated time.Time
	Age     time.Duration // Time since last update
}

// ThreadStalenessDetector detects open threads that haven't been updated in 7+ days.
type ThreadStalenessDetector struct {
	Source   ThreadStalenessSource
	MaxStale time.Duration // Staleness threshold (default 7 days)
}

func (d *ThreadStalenessDetector) Name() string { return "thread_staleness" }

func (d *ThreadStalenessDetector) Detect() ([]TriggerSuggestion, error) {
	if d.Source == nil {
		return nil, fmt.Errorf("thread staleness source not configured")
	}

	maxStale := d.MaxStale
	if maxStale <= 0 {
		maxStale = 7 * 24 * time.Hour // 7 days
	}

	threads, err := d.Source.ListOpenThreads()
	if err != nil {
		return nil, err
	}

	var suggestions []TriggerSuggestion
	for _, t := range threads {
		if t.Age < maxStale {
			continue
		}

		suggestions = append(suggestions, TriggerSuggestion{
			Detector:    "thread_staleness",
			Key:         t.Slug,
			Title:       fmt.Sprintf("Stale thread: %s (%dd since update)", t.Title, int(t.Age.Hours()/24)),
			Description: fmt.Sprintf("Thread %q has been open for %d days without updates. Consider resolving it or adding new findings.", t.Title, int(t.Age.Hours()/24)),
			IssueType:   "task",
			Priority:    4,
		})
	}
	return suggestions, nil
}

// --- Default Source Implementations ---

// defaultThreadStalenessSource reads from .kb/threads/.
type defaultThreadStalenessSource struct{}

func (s *defaultThreadStalenessSource) ListOpenThreads() ([]StaleThread, error) {
	projectDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	threadsDir := filepath.Join(projectDir, ".kb", "threads")
	summaries, err := thread.List(threadsDir)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var result []StaleThread
	for _, ts := range summaries {
		if ts.Status != "open" {
			continue
		}

		updated, err := time.Parse("2006-01-02", ts.Updated)
		if err != nil {
			continue
		}

		age := now.Sub(updated)
		result = append(result, StaleThread{
			Slug:    ts.Name,
			Title:   ts.Title,
			Updated: updated,
			Age:     age,
		})
	}
	return result, nil
}

// defaultInvestigationOrphansSource scans .kb/investigations/ for active investigations.
type defaultInvestigationOrphansSource struct{}

func (s *defaultInvestigationOrphansSource) ListActiveInvestigations() ([]OrphanedInvestigation, error) {
	projectDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	invDir := filepath.Join(projectDir, ".kb", "investigations")
	entries, err := os.ReadDir(invDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	now := time.Now()
	var result []OrphanedInvestigation
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(invDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "Status: Active") &&
			!strings.Contains(contentStr, "status: Active") &&
			!strings.Contains(contentStr, "status: active") {
			continue
		}

		slug := strings.TrimSuffix(entry.Name(), ".md")
		if len(slug) > 11 && slug[4] == '-' && slug[7] == '-' && slug[10] == '-' {
			slug = slug[11:]
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}
		age := now.Sub(info.ModTime())

		result = append(result, OrphanedInvestigation{
			Path: path,
			Slug: slug,
			Age:  age,
		})
	}
	return result, nil
}

func (s *defaultInvestigationOrphansSource) HasOpenIssueForInvestigation(slug string) (bool, error) {
	issues, err := ListIssuesWithLabel("triage:ready")
	if err != nil {
		return false, err
	}
	for _, issue := range issues {
		if strings.Contains(strings.ToLower(issue.Title), strings.ToLower(slug)) {
			return true, nil
		}
	}
	issues, err = ListIssuesWithLabel("")
	if err != nil {
		return false, err
	}
	for _, issue := range issues {
		if issue.Status == "in_progress" && strings.Contains(strings.ToLower(issue.Title), strings.ToLower(slug)) {
			return true, nil
		}
	}
	return false, nil
}

// defaultRecurringBugsSource uses events.jsonl retry patterns to find reworked issues.
type defaultRecurringBugsSource struct{}

func (s *defaultRecurringBugsSource) ListClosedIssuesWithRework(minReworks int) ([]ReworkedIssue, error) {
	patterns, err := verify.GetAllRetryPatterns()
	if err != nil {
		return nil, err
	}

	var result []ReworkedIssue
	for _, stats := range patterns {
		// Count reworks as spawn attempts beyond the first
		reworkCount := stats.SpawnCount - 1
		if reworkCount < minReworks {
			continue
		}
		result = append(result, ReworkedIssue{
			ID:          stats.BeadsID,
			Title:       stats.BeadsID, // Use beads ID as title fallback
			ReworkCount: reworkCount,
		})
	}
	return result, nil
}

func (s *defaultRecurringBugsSource) HasOpenIssue(issueID string) (bool, error) {
	// Check if there's already an open trigger issue for this recurring bug
	issues, err := ListIssuesWithLabel(TriggerLabel)
	if err != nil {
		return false, err
	}
	for _, issue := range issues {
		if strings.Contains(issue.Title, issueID) {
			return true, nil
		}
	}
	return false, nil
}
