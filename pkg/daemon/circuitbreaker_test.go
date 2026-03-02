package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/control"
)

func TestOnceExcluding_HaltFileBlocksSpawn(t *testing.T) {
	// Create a temporary halt file
	tmpDir := t.TempDir()
	haltPath := filepath.Join(tmpDir, "halt")
	haltContent := `reason: Rolling 3-day average exceeded (75 commits/day, threshold 70)
triggered_by: rolling_avg
triggered_at: 2026-02-14T12:00:00Z`
	if err := os.WriteFile(haltPath, []byte(haltContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Override the default halt path for this test
	origHaltPath := control.DefaultHaltPath
	control.DefaultHaltPath = func() string { return haltPath }
	defer func() { control.DefaultHaltPath = origHaltPath }()

	spawnCalled := false
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		}},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
			spawnCalled = true
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}

	if result.Processed {
		t.Error("should not process when halt file exists")
	}
	if spawnCalled {
		t.Error("spawn should not be called when halted")
	}
	if !strings.Contains(result.Message, "Circuit breaker HALTED") {
		t.Errorf("message should mention circuit breaker halt, got: %s", result.Message)
	}
	if !strings.Contains(result.Message, "rolling_avg") {
		t.Errorf("message should include trigger, got: %s", result.Message)
	}
}

func TestOnceExcluding_NoHaltFileAllowsSpawn(t *testing.T) {
	// Point to a non-existent halt file
	tmpDir := t.TempDir()
	haltPath := filepath.Join(tmpDir, "halt")

	origHaltPath := control.DefaultHaltPath
	control.DefaultHaltPath = func() string { return haltPath }
	defer func() { control.DefaultHaltPath = origHaltPath }()

	spawnCalled := false
	d := &Daemon{
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "Test", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		}},
		Spawner: &mockSpawner{SpawnWorkFunc: func(beadsID, model, workdir string) error {
			spawnCalled = true
			return nil
		}},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil
		}},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}

	if !result.Processed {
		t.Errorf("should process when no halt file, got message: %s", result.Message)
	}
	if !spawnCalled {
		t.Error("spawn should be called when not halted")
	}
}
