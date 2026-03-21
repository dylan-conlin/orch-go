// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"

	"github.com/dylan-conlin/orch-go/pkg/artifactsync"
)

// ArtifactSyncResult contains the result of running artifact sync analysis.
type ArtifactSyncResult struct {
	DriftDetected bool
	EntriesCount  int
	EventsCount   int
	Report        *artifactsync.DriftReport
	IssueID       string
	Deduped       bool
	AgentSpawned  bool
	OverBudget    bool
	CLAUDEMDLines int
	Message       string
	Error         error
}

// ArtifactSyncService provides artifact drift analysis and issue creation.
type ArtifactSyncService interface {
	// Analyze loads the manifest and drift events, returns drift analysis.
	Analyze(projectDir string) (*ArtifactSyncResult, error)
	// HasOpenIssue checks if an artifact-sync issue already exists (dedup).
	HasOpenIssue() (bool, error)
	// CreateIssue creates a beads issue for drifted artifacts.
	CreateIssue(report *artifactsync.DriftReport) (string, error)
	// SpawnSyncAgent spawns an artifact-sync agent to fix drift.
	SpawnSyncAgent(report *artifactsync.DriftReport) error
	// SpawnBudgetAwareSyncAgent spawns a sync agent with instructions to trim
	// lowest-relevance content before adding new content (used when CLAUDE.md
	// exceeds its line budget).
	SpawnBudgetAwareSyncAgent(report *artifactsync.DriftReport, currentLines, budget int) error
	// CLAUDEMDLineCount returns the current line count of CLAUDE.md in the project.
	CLAUDEMDLineCount(projectDir string) (int, error)
}

// ShouldRunArtifactSync returns true if periodic artifact sync should run.
func (d *Daemon) ShouldRunArtifactSync() bool {
	return d.Scheduler.IsDue(TaskArtifactSync)
}

// RunPeriodicArtifactSync runs artifact drift analysis if due.
// Creates beads issues for drifted artifacts (with dedup) and optionally
// auto-spawns a sync agent when drift exceeds the configured threshold.
// When CLAUDE.md exceeds the configured line budget, the sync agent is
// instructed to remove lowest-relevance content before adding new content.
// Returns the result if analysis was run, or nil if it wasn't due.
func (d *Daemon) RunPeriodicArtifactSync() *ArtifactSyncResult {
	if !d.ShouldRunArtifactSync() {
		return nil
	}

	svc := d.ArtifactSync
	if svc == nil {
		return &ArtifactSyncResult{
			Error:   fmt.Errorf("artifact sync service not configured"),
			Message: "Artifact sync: service not configured",
		}
	}

	// Analyze drift
	projectDir := d.Config.ArtifactSyncProjectDir
	result, err := svc.Analyze(projectDir)
	if err != nil {
		return &ArtifactSyncResult{
			Error:   err,
			Message: fmt.Sprintf("Artifact sync failed: %v", err),
		}
	}

	// No drift detected — mark run and return
	if !result.DriftDetected {
		d.Scheduler.MarkRun(TaskArtifactSync)
		return result
	}

	// Check for dedup — skip if open issue exists
	hasOpen, err := svc.HasOpenIssue()
	if err != nil {
		return &ArtifactSyncResult{
			Error:   err,
			Message: fmt.Sprintf("Artifact sync dedup check failed: %v", err),
		}
	}
	if hasOpen {
		result.Deduped = true
		result.Message = fmt.Sprintf("Artifact sync: drift detected (%d entries) but open issue exists, skipping", result.EntriesCount)
		d.Scheduler.MarkRun(TaskArtifactSync)
		return result
	}

	// Create beads issue for the drift
	issueID, err := svc.CreateIssue(result.Report)
	if err != nil {
		return &ArtifactSyncResult{
			DriftDetected: result.DriftDetected,
			EntriesCount:  result.EntriesCount,
			EventsCount:   result.EventsCount,
			Error:         err,
			Message:       fmt.Sprintf("Artifact sync issue creation failed: %v", err),
		}
	}
	result.IssueID = issueID

	// Check CLAUDE.md line budget before spawning
	budget := d.Config.ArtifactSyncCLAUDEMDLineBudget
	if budget <= 0 {
		budget = 300 // fallback default
	}
	if lineCount, err := svc.CLAUDEMDLineCount(projectDir); err == nil {
		result.CLAUDEMDLines = lineCount
		result.OverBudget = lineCount > budget
	}

	// Auto-spawn sync agent if enabled and entries exceed threshold
	if d.Config.ArtifactSyncAutoSpawn && result.EntriesCount >= d.Config.ArtifactSyncAutoSpawnThreshold {
		var spawnErr error
		if result.OverBudget {
			spawnErr = svc.SpawnBudgetAwareSyncAgent(result.Report, result.CLAUDEMDLines, budget)
		} else {
			spawnErr = svc.SpawnSyncAgent(result.Report)
		}
		if spawnErr != nil {
			result.Message = fmt.Sprintf("Artifact sync: created issue %s, but spawn failed: %v", issueID, spawnErr)
			// Don't return error — issue was created successfully
		} else {
			result.AgentSpawned = true
			if result.OverBudget {
				result.Message = fmt.Sprintf("Artifact sync: created issue %s, spawned budget-aware sync agent (%d entries, CLAUDE.md %d/%d lines)", issueID, result.EntriesCount, result.CLAUDEMDLines, budget)
			} else {
				result.Message = fmt.Sprintf("Artifact sync: created issue %s, spawned sync agent (%d entries)", issueID, result.EntriesCount)
			}
		}
	} else {
		result.Message = fmt.Sprintf("Artifact sync: created issue %s (%d entries, %d events)", issueID, result.EntriesCount, result.EventsCount)
	}

	d.Scheduler.MarkRun(TaskArtifactSync)
	return result
}
