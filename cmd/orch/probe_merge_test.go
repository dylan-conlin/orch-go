package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestResolveProbeMergeRootPrefersWorkspace(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, ".kb", "models"), 0755); err != nil {
		t.Fatal(err)
	}

	root := resolveProbeMergeRoot(&CompletionTarget{
		WorkspacePath:   workspace,
		BeadsProjectDir: "/tmp/project",
	})
	if root != workspace {
		t.Fatalf("root = %q, want %q", root, workspace)
	}
}

func TestResolveProbeMergeRootFallsBackToProject(t *testing.T) {
	root := resolveProbeMergeRoot(&CompletionTarget{
		WorkspacePath:   "",
		BeadsProjectDir: "/tmp/project",
	})
	if root != "/tmp/project" {
		t.Fatalf("root = %q, want %q", root, "/tmp/project")
	}
}

func TestProbeVerdict(t *testing.T) {
	tests := []struct {
		name   string
		impact string
		want   string
	}{
		{name: "confirms", impact: "**Verdict:** confirms — matches model", want: "confirms"},
		{name: "extends", impact: "**Verdict:** extends — new behavior", want: "extends"},
		{name: "contradicts", impact: "**Verdict:** contradicts — model wrong", want: "contradicts"},
		{name: "missing", impact: "No verdict line", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := probeVerdict(tt.impact)
			if got != tt.want {
				t.Fatalf("probeVerdict() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMergeProbesNonInteractiveByVerdict(t *testing.T) {
	root := t.TempDir()
	models := filepath.Join(root, ".kb", "models")
	if err := os.MkdirAll(models, 0755); err != nil {
		t.Fatal(err)
	}

	modelPath := filepath.Join(models, "test-model.md")
	model := "# Model: test\n\n**Last Updated:** 2026-01-01\n\n## Summary\nTest\n"
	if err := os.WriteFile(modelPath, []byte(model), 0644); err != nil {
		t.Fatal(err)
	}

	probesDir := filepath.Join(models, "test-model", "probes")
	if err := os.MkdirAll(probesDir, 0755); err != nil {
		t.Fatal(err)
	}

	writeProbe := func(name, verdict string) {
		content := "# Probe\n\n## Model Impact\n\n**Verdict:** " + verdict + " — test\n\n**Details:**\nDetails\n"
		if err := os.WriteFile(filepath.Join(probesDir, name+".md"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	writeProbe("2026-02-08-confirms", "confirms")
	writeProbe("2026-02-08-extends", "extends")
	writeProbe("2026-02-08-contradicts", "contradicts")

	probes := spawn.FindProjectProbes(root)
	if len(probes) != 3 {
		t.Fatalf("expected 3 probes, got %d", len(probes))
	}

	result := mergeProbesNonInteractive(probes)
	if result.merged != 2 {
		t.Fatalf("merged = %d, want 2", result.merged)
	}
	if result.confirms != 1 {
		t.Fatalf("confirms = %d, want 1", result.confirms)
	}
	if result.extends != 1 {
		t.Fatalf("extends = %d, want 1", result.extends)
	}
	if len(result.changelog) != 1 {
		t.Fatalf("changelog notes = %d, want 1", len(result.changelog))
	}
	if len(result.review) != 1 {
		t.Fatalf("review items = %d, want 1", len(result.review))
	}
	if !strings.Contains(result.review[0], "contradicts") {
		t.Fatalf("review entry = %q, expected contradicting probe", result.review[0])
	}

	data, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "2026-02-08-confirms") {
		t.Fatal("expected confirms probe to be merged")
	}
	if !strings.Contains(content, "2026-02-08-extends") {
		t.Fatal("expected extends probe to be merged")
	}
	if strings.Contains(content, "2026-02-08-contradicts") {
		t.Fatal("did not expect contradicting probe to be merged")
	}
}

func TestCommitProbeMergeArtifactsCommitsProbeAndModelFiles(t *testing.T) {
	root := t.TempDir()
	initGitRepo(t, root)

	modelPath := filepath.Join(root, ".kb", "models", "test-model.md")
	probesDir := filepath.Join(root, ".kb", "models", "test-model", "probes")
	if err := os.MkdirAll(probesDir, 0755); err != nil {
		t.Fatal(err)
	}

	model := "# Model: test\n\n**Last Updated:** 2026-01-01\n\n## Summary\nTest\n"
	if err := os.WriteFile(modelPath, []byte(model), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, root, "add", ".")
	runGitCmd(t, root, "commit", "-m", "initial")

	probePath := filepath.Join(probesDir, "2026-02-08-confirms.md")
	probe := "# Probe\n\n## Model Impact\n\n**Verdict:** confirms\n"
	if err := os.WriteFile(probePath, []byte(probe), 0644); err != nil {
		t.Fatal(err)
	}

	probes := spawn.FindProjectProbes(root)
	if len(probes) != 1 {
		t.Fatalf("expected 1 probe, got %d", len(probes))
	}
	if err := mergeProbeIntoModel(probes[0]); err != nil {
		t.Fatal(err)
	}

	committed, err := commitProbeMergeArtifacts(&CompletionTarget{
		AgentName:       "og-probe-merge",
		WorkspacePath:   "",
		BeadsProjectDir: root,
	}, probes)
	if err != nil {
		t.Fatalf("commitProbeMergeArtifacts() unexpected error: %v", err)
	}
	if !committed {
		t.Fatal("expected probe/model changes to be committed")
	}

	subject := strings.TrimSpace(runGitCmd(t, root, "log", "-1", "--pretty=%s"))
	if subject != "chore: merge probes from og-probe-merge" {
		t.Fatalf("commit subject = %q", subject)
	}
}

func TestCommitProbeMergeArtifactsNoChangesNoCommit(t *testing.T) {
	root := t.TempDir()
	initGitRepo(t, root)

	modelPath := filepath.Join(root, ".kb", "models", "test-model.md")
	probesDir := filepath.Join(root, ".kb", "models", "test-model", "probes")
	if err := os.MkdirAll(probesDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(modelPath, []byte("# Model\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(probesDir, "2026-02-08-probe.md"), []byte("# Probe\n\n## Model Impact\n\n**Verdict:** confirms\n"), 0644); err != nil {
		t.Fatal(err)
	}

	runGitCmd(t, root, "add", ".")
	runGitCmd(t, root, "commit", "-m", "initial")

	before := strings.TrimSpace(runGitCmd(t, root, "rev-parse", "HEAD"))
	probes := spawn.FindProjectProbes(root)
	committed, err := commitProbeMergeArtifacts(&CompletionTarget{AgentName: "og-probe-merge", BeadsProjectDir: root}, probes)
	if err != nil {
		t.Fatalf("commitProbeMergeArtifacts() unexpected error: %v", err)
	}
	if committed {
		t.Fatal("expected no commit when probe/model files are unchanged")
	}
	after := strings.TrimSpace(runGitCmd(t, root, "rev-parse", "HEAD"))
	if before != after {
		t.Fatalf("expected HEAD to remain unchanged, before=%s after=%s", before, after)
	}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGitCmd(t, dir, "init")
	runGitCmd(t, dir, "config", "user.email", "test@test.com")
	runGitCmd(t, dir, "config", "user.name", "Test User")
}

func runGitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
	return string(out)
}
