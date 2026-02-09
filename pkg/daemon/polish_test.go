package daemon

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestDaemon_ShouldRunPolish(t *testing.T) {
	d := &Daemon{
		Config: Config{
			PolishEnabled:  true,
			PolishInterval: time.Hour,
		},
	}

	if !d.ShouldRunPolish() {
		t.Fatal("ShouldRunPolish() should return true when never run")
	}

	d.lastPolish = time.Now().Add(-30 * time.Minute)
	if d.ShouldRunPolish() {
		t.Fatal("ShouldRunPolish() should return false before interval elapses")
	}

	d.lastPolish = time.Now().Add(-2 * time.Hour)
	if !d.ShouldRunPolish() {
		t.Fatal("ShouldRunPolish() should return true after interval elapses")
	}

	d.Config.PolishEnabled = false
	if d.ShouldRunPolish() {
		t.Fatal("ShouldRunPolish() should return false when disabled")
	}
}

func TestDaemon_RunPolish_RespectsCycleAndDailyCaps(t *testing.T) {
	created := 0

	d := &Daemon{
		Config: Config{
			PolishEnabled:           true,
			PolishInterval:          time.Minute,
			PolishMaxIssuesPerCycle: 2,
			PolishMaxIssuesPerDay:   3,
		},
		collectPolishCandidatesFunc: func(projectDir string) ([]PolishIssueSpec, error) {
			return []PolishIssueSpec{
				{Audit: "metadata", DedupLabel: "polish:metadata", Title: "metadata"},
				{Audit: "knowledge", DedupLabel: "polish:knowledge-synthesis", Title: "knowledge"},
				{Audit: "stale", DedupLabel: "polish:knowledge-stale", Title: "stale"},
				{Audit: "quality", DedupLabel: "polish:quality", Title: "quality"},
			}, nil
		},
		listAllIssuesFunc: func() ([]Issue, error) {
			return []Issue{}, nil
		},
		createPolishIssueFunc: func(spec PolishIssueSpec) (string, error) {
			created++
			return fmt.Sprintf("orch-go-polish-%d", created), nil
		},
	}

	result1 := d.RunPolish("")
	if result1 == nil {
		t.Fatal("RunPolish() should run on first invocation")
	}
	if result1.Created != 2 {
		t.Fatalf("first run created %d issues, want 2", result1.Created)
	}

	d.lastPolish = time.Now().Add(-2 * time.Minute)
	result2 := d.RunPolish("")
	if result2 == nil {
		t.Fatal("RunPolish() second run should execute")
	}
	if result2.Created != 1 {
		t.Fatalf("second run created %d issues, want 1 (daily remaining)", result2.Created)
	}

	d.lastPolish = time.Now().Add(-2 * time.Minute)
	result3 := d.RunPolish("")
	if result3 == nil {
		t.Fatal("RunPolish() third run should execute")
	}
	if result3.Created != 0 {
		t.Fatalf("third run created %d issues, want 0 (daily cap reached)", result3.Created)
	}
	if !strings.Contains(result3.Message, "daily cap reached") {
		t.Fatalf("third run message %q should mention daily cap", result3.Message)
	}
}

func TestDaemon_RunPolish_DedupsByOpenPolishLabel(t *testing.T) {
	createdLabels := make([]string, 0)

	d := &Daemon{
		Config: Config{
			PolishEnabled:           true,
			PolishInterval:          time.Minute,
			PolishMaxIssuesPerCycle: 3,
			PolishMaxIssuesPerDay:   10,
		},
		collectPolishCandidatesFunc: func(projectDir string) ([]PolishIssueSpec, error) {
			return []PolishIssueSpec{
				{Audit: "metadata", DedupLabel: "polish:metadata", Title: "metadata"},
				{Audit: "quality", DedupLabel: "polish:quality", Title: "quality"},
			}, nil
		},
		listAllIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "orch-go-existing", Status: "open", Labels: []string{"polish:metadata"}},
			}, nil
		},
		createPolishIssueFunc: func(spec PolishIssueSpec) (string, error) {
			createdLabels = append(createdLabels, spec.DedupLabel)
			return "orch-go-created", nil
		},
	}

	result := d.RunPolish("")
	if result == nil {
		t.Fatal("RunPolish() should run")
	}
	if result.Created != 1 {
		t.Fatalf("created %d issues, want 1", result.Created)
	}
	if result.Skipped != 1 {
		t.Fatalf("skipped %d issues, want 1", result.Skipped)
	}
	if len(createdLabels) != 1 || createdLabels[0] != "polish:quality" {
		t.Fatalf("created labels = %v, want [polish:quality]", createdLabels)
	}
}

func TestDaemon_BuildMetadataPolishIssue(t *testing.T) {
	d := &Daemon{
		listAllIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "orch-go-a", Status: "open", Labels: []string{"triage:review"}},
				{ID: "orch-go-b", Status: "in_progress", Labels: []string{"area:daemon"}},
			}, nil
		},
	}

	spec, err := d.buildMetadataPolishIssue()
	if err != nil {
		t.Fatalf("buildMetadataPolishIssue() unexpected error: %v", err)
	}
	if spec == nil {
		t.Fatal("buildMetadataPolishIssue() returned nil, want issue spec")
	}
	if spec.DedupLabel != "polish:metadata" {
		t.Fatalf("DedupLabel = %q, want polish:metadata", spec.DedupLabel)
	}
	if !strings.Contains(spec.Description, "orch-go-a") {
		t.Fatalf("Description should include missing issue ID, got %q", spec.Description)
	}
}
