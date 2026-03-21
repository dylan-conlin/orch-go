package daemon

import (
	"fmt"
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

func TestDaemon_RunPeriodicArtifactSync_OverBudget_SpawnsBudgetAwareAgent(t *testing.T) {
	budgetSpawnCalled := false
	normalSpawnCalled := false
	cfg := Config{
		ArtifactSyncEnabled:            true,
		ArtifactSyncInterval:           24 * time.Hour,
		ArtifactSyncAutoSpawn:          true,
		ArtifactSyncAutoSpawnThreshold: 1,
		ArtifactSyncCLAUDEMDLineBudget: 300,
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
							{ArtifactPath: "CLAUDE.md", SectionName: "Events", Triggers: []string{"new-event"}},
						},
					},
				}, nil
			},
			HasOpenIssueFunc: func() (bool, error) {
				return false, nil
			},
			CreateIssueFunc: func(report *artifactsync.DriftReport) (string, error) {
				return "orch-go-budget1", nil
			},
			CLAUDEMDLineCountFunc: func(projectDir string) (int, error) {
				return 450, nil // Over the 300-line budget
			},
			SpawnSyncAgentFunc: func(report *artifactsync.DriftReport) error {
				normalSpawnCalled = true
				return nil
			},
			SpawnBudgetAwareSyncAgentFunc: func(report *artifactsync.DriftReport, currentLines, budget int) error {
				budgetSpawnCalled = true
				if currentLines != 450 {
					t.Errorf("expected currentLines=450, got %d", currentLines)
				}
				if budget != 300 {
					t.Errorf("expected budget=300, got %d", budget)
				}
				return nil
			},
		},
	}
	d.Scheduler.SetLastRun(TaskArtifactSync, time.Now().Add(-48*time.Hour))

	result := d.RunPeriodicArtifactSync()
	if result == nil {
		t.Fatal("expected result")
	}
	if !budgetSpawnCalled {
		t.Error("expected budget-aware sync agent to be spawned when over budget")
	}
	if normalSpawnCalled {
		t.Error("should not spawn normal sync agent when over budget")
	}
	if !result.OverBudget {
		t.Error("expected OverBudget to be true")
	}
	if result.CLAUDEMDLines != 450 {
		t.Errorf("expected CLAUDEMDLines=450, got %d", result.CLAUDEMDLines)
	}
	if !result.AgentSpawned {
		t.Error("expected AgentSpawned to be true")
	}
}

func TestDaemon_RunPeriodicArtifactSync_UnderBudget_SpawnsNormalAgent(t *testing.T) {
	budgetSpawnCalled := false
	normalSpawnCalled := false
	cfg := Config{
		ArtifactSyncEnabled:            true,
		ArtifactSyncInterval:           24 * time.Hour,
		ArtifactSyncAutoSpawn:          true,
		ArtifactSyncAutoSpawnThreshold: 1,
		ArtifactSyncCLAUDEMDLineBudget: 300,
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
							{ArtifactPath: "CLAUDE.md", SectionName: "Events", Triggers: []string{"new-event"}},
						},
					},
				}, nil
			},
			HasOpenIssueFunc: func() (bool, error) {
				return false, nil
			},
			CreateIssueFunc: func(report *artifactsync.DriftReport) (string, error) {
				return "orch-go-budget2", nil
			},
			CLAUDEMDLineCountFunc: func(projectDir string) (int, error) {
				return 200, nil // Under the 300-line budget
			},
			SpawnSyncAgentFunc: func(report *artifactsync.DriftReport) error {
				normalSpawnCalled = true
				return nil
			},
			SpawnBudgetAwareSyncAgentFunc: func(report *artifactsync.DriftReport, currentLines, budget int) error {
				budgetSpawnCalled = true
				return nil
			},
		},
	}
	d.Scheduler.SetLastRun(TaskArtifactSync, time.Now().Add(-48*time.Hour))

	result := d.RunPeriodicArtifactSync()
	if result == nil {
		t.Fatal("expected result")
	}
	if budgetSpawnCalled {
		t.Error("should not spawn budget-aware agent when under budget")
	}
	if !normalSpawnCalled {
		t.Error("expected normal sync agent to be spawned when under budget")
	}
	if result.OverBudget {
		t.Error("expected OverBudget to be false")
	}
	if result.CLAUDEMDLines != 200 {
		t.Errorf("expected CLAUDEMDLines=200, got %d", result.CLAUDEMDLines)
	}
}

func TestBuildBudgetAwareSyncTask_ContainsBudgetInstructions(t *testing.T) {
	report := &artifactsync.DriftReport{
		Entries: []artifactsync.DriftReportEntry{
			{ArtifactPath: "CLAUDE.md", SectionName: "Commands", Triggers: []string{"new-command"}},
		},
	}
	task := buildBudgetAwareSyncTask(report, 450, 300)

	if !contains(task, "LINE BUDGET EXCEEDED") {
		t.Error("expected task to contain budget exceeded warning")
	}
	if !contains(task, "450") {
		t.Error("expected task to contain current line count")
	}
	if !contains(task, "300") {
		t.Error("expected task to contain budget")
	}
	if !contains(task, "remove lowest-relevance content") {
		t.Error("expected task to contain trim instructions")
	}
	if !contains(task, "CLAUDE.md:Commands") {
		t.Error("expected task to contain drift entries")
	}
}

func TestDaemon_RunPeriodicArtifactSync_LineCountError_FallsBackToNormalSpawn(t *testing.T) {
	normalSpawnCalled := false
	cfg := Config{
		ArtifactSyncEnabled:            true,
		ArtifactSyncInterval:           24 * time.Hour,
		ArtifactSyncAutoSpawn:          true,
		ArtifactSyncAutoSpawnThreshold: 1,
		ArtifactSyncCLAUDEMDLineBudget: 300,
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
				}, nil
			},
			HasOpenIssueFunc: func() (bool, error) {
				return false, nil
			},
			CreateIssueFunc: func(report *artifactsync.DriftReport) (string, error) {
				return "orch-go-budget3", nil
			},
			CLAUDEMDLineCountFunc: func(projectDir string) (int, error) {
				return 0, fmt.Errorf("CLAUDE.md not found")
			},
			SpawnSyncAgentFunc: func(report *artifactsync.DriftReport) error {
				normalSpawnCalled = true
				return nil
			},
		},
	}
	d.Scheduler.SetLastRun(TaskArtifactSync, time.Now().Add(-48*time.Hour))

	result := d.RunPeriodicArtifactSync()
	if result == nil {
		t.Fatal("expected result")
	}
	if !normalSpawnCalled {
		t.Error("expected normal spawn when line count fails")
	}
	if result.OverBudget {
		t.Error("expected OverBudget=false when line count fails")
	}
}
