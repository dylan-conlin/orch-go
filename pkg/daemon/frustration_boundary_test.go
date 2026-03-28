package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunPeriodicFrustrationBoundary_TransitionsBoundaryAgent(t *testing.T) {
	projectDir := t.TempDir()
	workspacePath := filepath.Join(projectDir, ".orch", "workspace", "og-feat-boundary-test")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	spawnContext := "TASK: Implement frustration boundary headless track\n\nbd comments add orch-go-test\n"
	if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatalf("failed to write spawn context: %v", err)
	}

	d := NewWithConfig(Config{
		PhaseTimeoutEnabled:  true,
		PhaseTimeoutInterval: time.Minute,
	})
	d.ProjectRegistry = NewProjectRegistryFromMap(map[string]string{"orch-go": projectDir}, projectDir)
	d.Agents = &mockAgentDiscoverer{
		GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
			return []ActiveAgent{{BeadsID: "orch-go-test", Title: "Test worker", Phase: "Boundary - thrashing and stuck"}}, nil
		},
	}

	var transitioned struct {
		beadsID  string
		feedback string
		workdir  string
	}
	d.BoundaryTransitioner = &mockBoundaryTransitioner{
		TransitionFunc: func(beadsID, feedback, workdir string) error {
			transitioned.beadsID = beadsID
			transitioned.feedback = feedback
			transitioned.workdir = workdir
			return nil
		},
	}

	result := d.RunPeriodicFrustrationBoundary()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.HandledCount != 1 {
		t.Fatalf("HandledCount = %d, want 1", result.HandledCount)
	}
	if transitioned.beadsID != "orch-go-test" {
		t.Fatalf("transitioned beadsID = %q, want orch-go-test", transitioned.beadsID)
	}
	if transitioned.workdir != projectDir {
		t.Fatalf("transition workdir = %q, want %q", transitioned.workdir, projectDir)
	}
	if !strings.Contains(transitioned.feedback, "Original question: Implement frustration boundary headless track") {
		t.Fatalf("feedback missing original task: %q", transitioned.feedback)
	}
	artifactPath := filepath.Join(workspacePath, "FRUSTRATION_BOUNDARY.md")
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("expected artifact to exist: %v", err)
	}
	if !strings.Contains(string(data), "Implement frustration boundary headless track") {
		t.Fatalf("artifact missing original task: %s", string(data))
	}
	if len(result.Agents) != 1 || result.Agents[0].ArtifactPath != artifactPath {
		t.Fatalf("result artifact path = %v, want %q", result.Agents, artifactPath)
	}
}

func TestRunPeriodicFrustrationBoundary_DedupsHandledBoundary(t *testing.T) {
	d := NewWithConfig(Config{
		PhaseTimeoutEnabled:  true,
		PhaseTimeoutInterval: time.Minute,
	})
	d.boundaryHandled["orch-go-test"] = time.Now()
	d.Agents = &mockAgentDiscoverer{
		GetActiveAgentsFunc: func() ([]ActiveAgent, error) {
			return []ActiveAgent{{BeadsID: "orch-go-test", Phase: "Boundary - duplicate"}}, nil
		},
	}
	d.BoundaryTransitioner = &mockBoundaryTransitioner{
		TransitionFunc: func(beadsID, feedback, workdir string) error {
			t.Fatalf("Transition should not be called for already-handled boundary")
			return nil
		},
	}

	result := d.RunPeriodicFrustrationBoundary()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.HandledCount != 0 {
		t.Fatalf("HandledCount = %d, want 0", result.HandledCount)
	}
}

func TestBuildBoundaryFeedback(t *testing.T) {
	feedback := buildBoundaryFeedback(
		"Original question",
		"thrashing and stuck",
		[]string{"Tried path A", "Tried path B"},
		"/tmp/FRUSTRATION_BOUNDARY.md",
	)
	checks := []string{
		"Original question: Original question",
		"What was tried:",
		"- Tried path A",
		"What did not work: thrashing and stuck",
		"/tmp/FRUSTRATION_BOUNDARY.md",
	}
	for _, check := range checks {
		if !strings.Contains(feedback, check) {
			t.Fatalf("feedback missing %q: %s", check, feedback)
		}
	}
}
