package daemon

import (
	"fmt"
	"testing"
)

// =============================================================================
// Tests for Fresh Status Check (TOCTOU race prevention)
// =============================================================================

func TestDaemon_Once_FreshStatusCheck_SkipsInProgressIssue(t *testing.T) {
	spawnCalled := false
	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
				}, nil
			},
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				return "in_progress", nil
			},
		},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
			spawnCalled = true
			return nil
		}},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() unexpected error: %v", err)
	}
	if result.Processed {
		t.Error("Once() should not process an in_progress issue")
	}
	if spawnCalled {
		t.Error("spawnFunc should not be called when fresh status check shows in_progress")
	}
	if result.Issue == nil || result.Issue.ID != "proj-1" {
		t.Error("result.Issue should still reference the skipped issue")
	}
}

func TestDaemon_Once_FreshStatusCheck_AllowsOpenIssue(t *testing.T) {
	spawnCalled := false
	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
				}, nil
			},
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				return "open", nil
			},
		},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
			spawnCalled = true
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() unexpected error: %v", err)
	}
	if !result.Processed {
		t.Error("Once() should process an open issue")
	}
	if !spawnCalled {
		t.Error("spawnFunc should be called when fresh status check confirms open")
	}
}

func TestDaemon_Once_FreshStatusCheck_FailOpenOnError(t *testing.T) {
	spawnCalled := false
	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
				}, nil
			},
			GetIssueStatusFunc: func(beadsID string) (string, error) {
				return "", fmt.Errorf("beads daemon unavailable")
			},
		},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
			spawnCalled = true
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() unexpected error: %v", err)
	}
	if !result.Processed {
		t.Error("Once() should still process when fresh status check fails (fail-open)")
	}
	if !spawnCalled {
		t.Error("spawnFunc should be called when fresh status check errors (fail-open)")
	}
}

func TestDaemon_Once_FreshStatusCheck_NilFunc(t *testing.T) {
	spawnCalled := false
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		}},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
			spawnCalled = true
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	result, err := d.Once()
	if err != nil {
		t.Fatalf("Once() unexpected error: %v", err)
	}
	if !result.Processed {
		t.Error("Once() should process when Issues has no GetIssueStatusFunc")
	}
	if !spawnCalled {
		t.Error("spawnFunc should be called when Issues has no GetIssueStatusFunc")
	}
}

// =============================================================================
// Tests for Concurrent Daemon Dedup
// =============================================================================

func TestDaemon_ConcurrentDaemonDedup(t *testing.T) {
	issueStatus := "open"
	spawnCount := 0

	makeDaemon := func() *Daemon {
		return &Daemon{
			Issues: &mockIssueQuerier{
				ListReadyIssuesFunc: func() ([]Issue, error) {
					return []Issue{
						{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
					}, nil
				},
				GetIssueStatusFunc: func(beadsID string) (string, error) {
					return issueStatus, nil
				},
			},
			Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir, account string) error {
				spawnCount++
				return nil
			}},
			StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
				issueStatus = status
				return nil
			}},
		}
	}

	d1 := makeDaemon()
	result1, err := d1.Once()
	if err != nil {
		t.Fatalf("Daemon 1 Once() unexpected error: %v", err)
	}
	if !result1.Processed {
		t.Error("Daemon 1 should have processed the issue")
	}

	issueStatus = "in_progress"

	d2 := makeDaemon()
	result2, err := d2.Once()
	if err != nil {
		t.Fatalf("Daemon 2 Once() unexpected error: %v", err)
	}
	if result2.Processed {
		t.Error("Daemon 2 should NOT have processed the issue (fresh status check should catch in_progress)")
	}

	if spawnCount != 1 {
		t.Errorf("Expected exactly 1 spawn, got %d", spawnCount)
	}
}
