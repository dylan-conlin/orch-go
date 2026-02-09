package verify

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestIsSkillRequiringModelConnection(t *testing.T) {
	tests := []struct {
		name  string
		skill string
		want  bool
	}{
		{name: "investigation", skill: "investigation", want: true},
		{name: "research", skill: "research", want: true},
		{name: "architect", skill: "architect", want: true},
		{name: "feature impl", skill: "feature-impl", want: false},
		{name: "case insensitive", skill: "InVeStIgAtIoN", want: true},
		{name: "empty", skill: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSkillRequiringModelConnection(tt.skill)
			if got != tt.want {
				t.Errorf("IsSkillRequiringModelConnection(%q) = %v, want %v", tt.skill, got, tt.want)
			}
		})
	}
}

func TestVerifyModelConnectionForCompletionSkipsNonKnowledgeSkill(t *testing.T) {
	result := VerifyModelConnectionForCompletion("feature-impl", t.TempDir(), t.TempDir())
	if result != nil {
		t.Fatalf("expected nil for non-knowledge skill, got %+v", result)
	}
}

func TestVerifyModelConnectionPassesWithModelCandidate(t *testing.T) {
	workspace := t.TempDir()
	project := t.TempDir()

	err := os.WriteFile(filepath.Join(workspace, "SYNTHESIS.md"), []byte("**Model candidate:** completion-verification"), 0644)
	if err != nil {
		t.Fatalf("failed to write SYNTHESIS.md: %v", err)
	}

	result := VerifyModelConnectionForCompletion("investigation", workspace, project)
	if result == nil {
		t.Fatal("expected result for investigation skill")
	}
	if !result.Passed {
		t.Fatalf("expected pass, got errors: %v", result.Errors)
	}
	if !result.HasModelCandidate {
		t.Fatal("expected model candidate to be detected")
	}
}

func TestVerifyModelConnectionPassesWithProbeFile(t *testing.T) {
	project := t.TempDir()
	runGitModelConnection(t, project, "init")

	err := os.WriteFile(filepath.Join(project, "README.md"), []byte("init\n"), 0644)
	if err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}
	runGitModelConnection(t, project, "add", "README.md")
	runGitModelConnection(t, project, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "init")
	baseline := runGitModelConnection(t, project, "rev-parse", "HEAD")

	probePath := filepath.Join(project, ".kb", "models", "completion-verification", "probes", "2026-02-09-test.md")
	err = os.MkdirAll(filepath.Dir(probePath), 0755)
	if err != nil {
		t.Fatalf("failed to create probe dir: %v", err)
	}
	err = os.WriteFile(probePath, []byte("# Probe\n"), 0644)
	if err != nil {
		t.Fatalf("failed to write probe file: %v", err)
	}

	workspace := t.TempDir()
	err = spawn.WriteAgentManifest(workspace, spawn.AgentManifest{
		WorkspaceName: "og-inv-test-09feb",
		Skill:         "investigation",
		ProjectDir:    project,
		GitBaseline:   strings.TrimSpace(baseline),
		SpawnTime:     time.Now().Format(time.RFC3339),
		Tier:          "light",
	})
	if err != nil {
		t.Fatalf("failed to write AGENT_MANIFEST.json: %v", err)
	}

	result := VerifyModelConnectionForCompletion("investigation", workspace, project)
	if result == nil {
		t.Fatal("expected result for investigation skill")
	}
	if !result.Passed {
		t.Fatalf("expected pass, got errors: %v", result.Errors)
	}
	if !result.HasProbeConnection {
		t.Fatal("expected probe connection to be detected")
	}
}

func TestVerifyModelConnectionFailsWithoutProbeOrCandidate(t *testing.T) {
	workspace := t.TempDir()
	project := t.TempDir()

	result := VerifyModelConnectionForCompletion("research", workspace, project)
	if result == nil {
		t.Fatal("expected result for research skill")
	}
	if result.Passed {
		t.Fatal("expected failure when model connection evidence is missing")
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected blocking errors")
	}
}

func runGitModelConnection(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
	return strings.TrimSpace(string(out))
}
