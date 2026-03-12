package daemon

import (
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/artifactsync"
)

func TestDaemon_ShouldRunArtifactSync_Disabled(t *testing.T) {
	cfg := Config{
		ArtifactSyncEnabled:  false,
		ArtifactSyncInterval: 24 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
	}

	if d.ShouldRunArtifactSync() {
		t.Error("ShouldRunArtifactSync() should return false when disabled")
	}
}

func TestDaemon_RunPeriodicArtifactSync_NotDue(t *testing.T) {
	cfg := Config{
		ArtifactSyncEnabled:  true,
		ArtifactSyncInterval: 24 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		ArtifactSync: &mockArtifactSyncService{},
	}
	d.Scheduler.SetLastRun(TaskArtifactSync, time.Now())

	result := d.RunPeriodicArtifactSync()
	if result != nil {
		t.Error("RunPeriodicArtifactSync() should return nil when not due")
	}
}

func TestDaemon_RunPeriodicArtifactSync_NoDriftEvents(t *testing.T) {
	cfg := Config{
		ArtifactSyncEnabled:  true,
		ArtifactSyncInterval: 24 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		ArtifactSync: &mockArtifactSyncService{
			AnalyzeFunc: func(projectDir string) (*ArtifactSyncResult, error) {
				return &ArtifactSyncResult{
					Message: "No drift events found",
				}, nil
			},
		},
	}
	// Make it due by setting last run in the past
	d.Scheduler.SetLastRun(TaskArtifactSync, time.Now().Add(-48*time.Hour))

	result := d.RunPeriodicArtifactSync()
	if result == nil {
		t.Fatal("RunPeriodicArtifactSync() should return result when due")
	}
	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}
}

func TestDaemon_RunPeriodicArtifactSync_DriftDetected_IssueCreated(t *testing.T) {
	issueCreated := false
	cfg := Config{
		ArtifactSyncEnabled:  true,
		ArtifactSyncInterval: 24 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		ArtifactSync: &mockArtifactSyncService{
			AnalyzeFunc: func(projectDir string) (*ArtifactSyncResult, error) {
				return &ArtifactSyncResult{
					DriftDetected: true,
					EntriesCount:  2,
					EventsCount:   5,
					Report: &artifactsync.DriftReport{
						Entries: []artifactsync.DriftReportEntry{
							{ArtifactPath: "CLAUDE.md", SectionName: "Commands", Triggers: []string{"new-command"}},
							{ArtifactPath: "CLAUDE.md", SectionName: "Key Packages", Triggers: []string{"new-package"}},
						},
					},
					Message: "Artifact drift: 2 entries affected by 5 events",
				}, nil
			},
			CreateIssueFunc: func(report *artifactsync.DriftReport) (string, error) {
				issueCreated = true
				return "orch-go-sync1", nil
			},
			HasOpenIssueFunc: func() (bool, error) {
				return false, nil
			},
		},
	}
	d.Scheduler.SetLastRun(TaskArtifactSync, time.Now().Add(-48*time.Hour))

	result := d.RunPeriodicArtifactSync()
	if result == nil {
		t.Fatal("expected result")
	}
	if !issueCreated {
		t.Error("expected issue to be created for drifted artifacts")
	}
	if result.IssueID != "orch-go-sync1" {
		t.Errorf("expected issue ID orch-go-sync1, got %s", result.IssueID)
	}
}

func TestDaemon_RunPeriodicArtifactSync_DedupSkipsWhenOpenIssue(t *testing.T) {
	issueCreated := false
	cfg := Config{
		ArtifactSyncEnabled:  true,
		ArtifactSyncInterval: 24 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		ArtifactSync: &mockArtifactSyncService{
			AnalyzeFunc: func(projectDir string) (*ArtifactSyncResult, error) {
				return &ArtifactSyncResult{
					DriftDetected: true,
					EntriesCount:  2,
					EventsCount:   3,
					Report: &artifactsync.DriftReport{
						Entries: []artifactsync.DriftReportEntry{
							{ArtifactPath: "CLAUDE.md", SectionName: "Commands", Triggers: []string{"new-command"}},
						},
					},
					Message: "Artifact drift: 2 entries",
				}, nil
			},
			CreateIssueFunc: func(report *artifactsync.DriftReport) (string, error) {
				issueCreated = true
				return "orch-go-sync2", nil
			},
			HasOpenIssueFunc: func() (bool, error) {
				return true, nil // Already has open issue
			},
		},
	}
	d.Scheduler.SetLastRun(TaskArtifactSync, time.Now().Add(-48*time.Hour))

	result := d.RunPeriodicArtifactSync()
	if result == nil {
		t.Fatal("expected result")
	}
	if issueCreated {
		t.Error("should not create issue when one is already open (dedup)")
	}
	if result.Deduped != true {
		t.Error("expected Deduped to be true")
	}
}

func TestDaemon_RunPeriodicArtifactSync_AutoSpawn(t *testing.T) {
	spawnCalled := false
	cfg := Config{
		ArtifactSyncEnabled:            true,
		ArtifactSyncInterval:           24 * time.Hour,
		ArtifactSyncAutoSpawn:          true,
		ArtifactSyncAutoSpawnThreshold: 3,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		ArtifactSync: &mockArtifactSyncService{
			AnalyzeFunc: func(projectDir string) (*ArtifactSyncResult, error) {
				return &ArtifactSyncResult{
					DriftDetected: true,
					EntriesCount:  4, // Exceeds threshold of 3
					EventsCount:   8,
					Report: &artifactsync.DriftReport{
						Entries: []artifactsync.DriftReportEntry{
							{ArtifactPath: "CLAUDE.md", SectionName: "Commands", Triggers: []string{"new-command"}},
							{ArtifactPath: "CLAUDE.md", SectionName: "Flags", Triggers: []string{"new-flag"}},
							{ArtifactPath: ".kb/guides/spawn.md", Triggers: []string{"config-change"}},
							{ArtifactPath: ".kb/guides/daemon.md", Triggers: []string{"new-event"}},
						},
					},
					Message: "Artifact drift: 4 entries",
				}, nil
			},
			HasOpenIssueFunc: func() (bool, error) {
				return false, nil
			},
			CreateIssueFunc: func(report *artifactsync.DriftReport) (string, error) {
				return "orch-go-sync3", nil
			},
			SpawnSyncAgentFunc: func(report *artifactsync.DriftReport) error {
				spawnCalled = true
				return nil
			},
		},
	}
	d.Scheduler.SetLastRun(TaskArtifactSync, time.Now().Add(-48*time.Hour))

	result := d.RunPeriodicArtifactSync()
	if result == nil {
		t.Fatal("expected result")
	}
	if !spawnCalled {
		t.Error("expected sync agent to be auto-spawned when entries exceed threshold")
	}
	if !result.AgentSpawned {
		t.Error("expected AgentSpawned to be true")
	}
}

func TestDaemon_RunPeriodicArtifactSync_AutoSpawn_BelowThreshold(t *testing.T) {
	spawnCalled := false
	cfg := Config{
		ArtifactSyncEnabled:            true,
		ArtifactSyncInterval:           24 * time.Hour,
		ArtifactSyncAutoSpawn:          true,
		ArtifactSyncAutoSpawnThreshold: 3,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		ArtifactSync: &mockArtifactSyncService{
			AnalyzeFunc: func(projectDir string) (*ArtifactSyncResult, error) {
				return &ArtifactSyncResult{
					DriftDetected: true,
					EntriesCount:  2, // Below threshold of 3
					EventsCount:   3,
					Report: &artifactsync.DriftReport{
						Entries: []artifactsync.DriftReportEntry{
							{ArtifactPath: "CLAUDE.md", SectionName: "Commands", Triggers: []string{"new-command"}},
							{ArtifactPath: "CLAUDE.md", SectionName: "Flags", Triggers: []string{"new-flag"}},
						},
					},
					Message: "Artifact drift: 2 entries",
				}, nil
			},
			HasOpenIssueFunc: func() (bool, error) {
				return false, nil
			},
			CreateIssueFunc: func(report *artifactsync.DriftReport) (string, error) {
				return "orch-go-sync4", nil
			},
			SpawnSyncAgentFunc: func(report *artifactsync.DriftReport) error {
				spawnCalled = true
				return nil
			},
		},
	}
	d.Scheduler.SetLastRun(TaskArtifactSync, time.Now().Add(-48*time.Hour))

	result := d.RunPeriodicArtifactSync()
	if result == nil {
		t.Fatal("expected result")
	}
	if spawnCalled {
		t.Error("should not auto-spawn when entries below threshold")
	}
	if result.AgentSpawned {
		t.Error("expected AgentSpawned to be false")
	}
}

func TestDaemon_RunPeriodicArtifactSync_MarkRunOnSuccess(t *testing.T) {
	cfg := Config{
		ArtifactSyncEnabled:  true,
		ArtifactSyncInterval: 24 * time.Hour,
	}
	d := &Daemon{
		Config:    cfg,
		Scheduler: NewSchedulerFromConfig(cfg),
		ArtifactSync: &mockArtifactSyncService{
			AnalyzeFunc: func(projectDir string) (*ArtifactSyncResult, error) {
				return &ArtifactSyncResult{Message: "No drift events"}, nil
			},
		},
	}
	d.Scheduler.SetLastRun(TaskArtifactSync, time.Now().Add(-48*time.Hour))

	before := d.Scheduler.LastRunTime(TaskArtifactSync)
	d.RunPeriodicArtifactSync()
	after := d.Scheduler.LastRunTime(TaskArtifactSync)

	if !after.After(before) {
		t.Error("expected scheduler to mark run time after successful execution")
	}
}
