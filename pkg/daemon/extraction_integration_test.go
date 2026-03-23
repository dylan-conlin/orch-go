package daemon

import (
	"fmt"
	"strings"
	"testing"
)

func TestOnceExcluding_AutoExtraction_SpawnsExtractionWhenCriticalHotspot(t *testing.T) {
	// When a triage:ready issue targets a CRITICAL hotspot file (>1500 lines),
	// the daemon should create an extraction issue and spawn it instead.
	spawnedID := ""
	d := &Daemon{
		Config: Config{Verbose: true},
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{
						ID:        "proj-1",
						Title:     "Add feature to cmd/orch/spawn_cmd.go",
						Priority:  2,
						IssueType: "feature",
						Status:    "open",
					},
				}, nil
			},
			CreateExtractionIssueFunc: func(task, parentID string) (string, error) {
				// Verify the extraction task was generated correctly
				if parentID != "proj-1" {
					t.Errorf("CreateExtractionIssue parentID = %q, want 'proj-1'", parentID)
				}
				if !strings.Contains(task, "Extract") {
					t.Errorf("CreateExtractionIssue task should contain 'Extract', got: %s", task)
				}
				return "proj-ext1", nil
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, skill, model, workdir, account string) error {
				spawnedID = beadsID
				return nil
			},
		},
		HotspotChecker: &mockHotspotChecker{
			hotspots: []HotspotWarning{
				{Path: "cmd/orch/spawn_cmd.go", Type: "bloat-size", Score: 2000},
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID string, status string) error {
				return nil // Mock: always succeed
			},
		},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if result == nil || !result.Processed {
		t.Fatal("OnceExcluding() expected processed result")
	}

	// Should have spawned the extraction issue, not the original
	if spawnedID != "proj-ext1" {
		t.Errorf("Spawner called with %q, want 'proj-ext1' (extraction issue)", spawnedID)
	}
	if !result.ExtractionSpawned {
		t.Error("OnceResult.ExtractionSpawned should be true")
	}
	if result.OriginalIssueID != "proj-1" {
		t.Errorf("OnceResult.OriginalIssueID = %q, want 'proj-1'", result.OriginalIssueID)
	}
	if result.Issue.ID != "proj-ext1" {
		t.Errorf("OnceResult.Issue.ID = %q, want 'proj-ext1'", result.Issue.ID)
	}
}

func TestOnceExcluding_AutoExtraction_SkipsWhenNoCriticalHotspot(t *testing.T) {
	// When hotspot check finds no CRITICAL files, spawn normally.
	spawnedID := ""
	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{
						ID:        "proj-1",
						Title:     "Add feature to pkg/daemon/daemon.go",
						Priority:  2,
						IssueType: "feature",
						Status:    "open",
					},
				}, nil
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, skill, model, workdir, account string) error {
				spawnedID = beadsID
				return nil
			},
		},
		HotspotChecker: &mockHotspotChecker{
			hotspots: []HotspotWarning{
				// Below critical threshold
				{Path: "pkg/daemon/daemon.go", Type: "bloat-size", Score: 1200},
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID string, status string) error {
				return nil // Mock: always succeed
			},
		},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if result == nil || !result.Processed {
		t.Fatal("OnceExcluding() expected processed result")
	}

	// Should have spawned the original issue normally
	if spawnedID != "proj-1" {
		t.Errorf("Spawner called with %q, want 'proj-1' (original issue)", spawnedID)
	}
	if result.ExtractionSpawned {
		t.Error("OnceResult.ExtractionSpawned should be false")
	}
}

func TestOnceExcluding_AutoExtraction_FailsFastOnExtractionFailure(t *testing.T) {
	// When extraction issue creation fails, skip the issue (fail-fast).
	// Extraction gate is non-negotiable - do not proceed with normal spawn.
	spawnedID := ""
	d := &Daemon{
		Config: Config{Verbose: true},
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{
						ID:        "proj-1",
						Title:     "Add feature to cmd/orch/spawn_cmd.go",
						Priority:  2,
						IssueType: "feature",
						Status:    "open",
					},
				}, nil
			},
			CreateExtractionIssueFunc: func(task, parentID string) (string, error) {
				return "", fmt.Errorf("bd create failed: command not found")
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, skill, model, workdir, account string) error {
				spawnedID = beadsID
				return nil
			},
		},
		HotspotChecker: &mockHotspotChecker{
			hotspots: []HotspotWarning{
				{Path: "cmd/orch/spawn_cmd.go", Type: "bloat-size", Score: 2000},
			},
		},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("OnceExcluding() expected non-nil result")
	}

	// Should NOT have processed/spawned - extraction gate is non-negotiable
	if result.Processed {
		t.Error("OnceExcluding() should NOT process when extraction setup fails (fail-fast)")
	}

	// Should not have spawned the original issue
	if spawnedID != "" {
		t.Errorf("Spawner should not be called when extraction fails, but was called with %q", spawnedID)
	}

	// Should have a message explaining the skip
	if result.Message == "" {
		t.Error("OnceResult.Message should explain why issue was skipped")
	}
}

func TestOnceExcluding_AutoExtraction_SkipsWhenNoHotspotChecker(t *testing.T) {
	// When HotspotChecker is nil, no extraction check happens.
	spawnedID := ""
	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{
						ID:        "proj-1",
						Title:     "Add feature to cmd/orch/spawn_cmd.go",
						Priority:  2,
						IssueType: "feature",
						Status:    "open",
					},
				}, nil
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, skill, model, workdir, account string) error {
				spawnedID = beadsID
				return nil
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID string, status string) error {
				return nil // Mock: always succeed
			},
		},
		// HotspotChecker is nil
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if result == nil || !result.Processed {
		t.Fatal("OnceExcluding() expected processed result")
	}

	// Should have spawned normally without extraction check
	if spawnedID != "proj-1" {
		t.Errorf("Spawner called with %q, want 'proj-1'", spawnedID)
	}
	if result.ExtractionSpawned {
		t.Error("OnceResult.ExtractionSpawned should be false when no checker")
	}
}
