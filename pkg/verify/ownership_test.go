package verify

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestClassifyArtifact(t *testing.T) {
	tests := []struct {
		path     string
		wantName string
		wantOwn  bool
	}{
		// Source files require ownership
		{"pkg/daemon/ooda.go", "source", true},
		{"cmd/orch/main.go", "source", true},
		{"main.go", "source", true},

		// Docs require ownership
		{"docs/README.md", "docs", true},
		{"CLAUDE.md", "docs", true},

		// Knowledge backlog is allowed residue
		{".kb/guides/foo.md", "knowledge-backlog", false},
		{".kb/models/test/model.md", "knowledge-backlog", false},
		{".kb/quick/entries.jsonl", "knowledge-backlog", false},

		// Local state never requires ownership
		{".beads/issues.jsonl", "local-state", false},

		// Generated workspace never requires ownership
		{".orch/workspace/foo/bar.md", "generated-workspace", false},

		// Experiment results never requires ownership
		{"experiments/coordination-demo/redesign/results/foo.csv", "experiment-results", false},

		// Skill stats are allowed residue
		{"skills/src/worker/feature-impl/stats.json", "skill-stats", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			class := ClassifyArtifact(tt.path)
			if class.Name != tt.wantName {
				t.Errorf("ClassifyArtifact(%q).Name = %q, want %q", tt.path, class.Name, tt.wantName)
			}
			if class.RequiresOwnership != tt.wantOwn {
				t.Errorf("ClassifyArtifact(%q).RequiresOwnership = %v, want %v", tt.path, class.RequiresOwnership, tt.wantOwn)
			}
		})
	}
}

func TestVerifyOwnershipReconciliation_NoBaseline(t *testing.T) {
	// Without a baseline, the gate should return nil (not applicable)
	result := VerifyOwnershipReconciliation("", "/tmp/fakedir")
	if result != nil {
		t.Errorf("expected nil result when workspace is empty, got %+v", result)
	}
}

func TestVerifyOwnershipReconciliation_CleanWorktree(t *testing.T) {
	// Set up a temp git repo with no dirty files
	dir := t.TempDir()
	ownershipSetupGitRepo(t, dir)

	// Create a workspace with a manifest
	workspace := filepath.Join(dir, ".orch", "workspace", "test")
	os.MkdirAll(workspace, 0755)

	// Get current HEAD for baseline
	baseline := ownershipGetHead(t, dir)

	ownershipWriteManifest(t, workspace, baseline)

	result := VerifyOwnershipReconciliation(workspace, dir)
	// Clean worktree with no dirty files = nil (not applicable) or pass
	if result != nil && !result.Passed {
		t.Errorf("expected pass on clean worktree, got errors: %v", result.Errors)
	}
}

func TestVerifyOwnershipReconciliation_DirtySourceFile(t *testing.T) {
	dir := t.TempDir()
	ownershipSetupGitRepo(t, dir)

	// Commit a source file
	ownershipWriteFile(t, filepath.Join(dir, "main.go"), "package main\n")
	ownershipRun(t, dir, "git", "add", "main.go")
	ownershipRun(t, dir, "git", "commit", "-m", "add main.go")

	baseline := ownershipGetHead(t, dir)

	// Agent makes a commit after baseline (partial work)
	ownershipWriteFile(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n")
	ownershipRun(t, dir, "git", "add", "main.go")
	ownershipRun(t, dir, "git", "commit", "-m", "partial implementation")

	// Then dirties the file further without committing
	ownershipWriteFile(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() { println(\"hello\") }\n")

	workspace := filepath.Join(dir, ".orch", "workspace", "test")
	os.MkdirAll(workspace, 0755)
	ownershipWriteManifest(t, workspace, baseline)

	result := VerifyOwnershipReconciliation(workspace, dir)
	if result == nil {
		t.Fatal("expected non-nil result for dirty source file")
	}
	if result.Passed {
		t.Error("expected gate to fail for uncommitted source file")
	}
	if len(result.UnownedFiles) == 0 {
		t.Error("expected UnownedFiles to contain main.go")
	}
}

func TestVerifyOwnershipReconciliation_AllowedResidue(t *testing.T) {
	dir := t.TempDir()
	ownershipSetupGitRepo(t, dir)

	// Commit a .kb file then dirty it
	kbDir := filepath.Join(dir, ".kb", "guides")
	os.MkdirAll(kbDir, 0755)
	ownershipWriteFile(t, filepath.Join(kbDir, "test.md"), "# Test\n")
	ownershipRun(t, dir, "git", "add", ".kb/guides/test.md")
	ownershipRun(t, dir, "git", "commit", "-m", "add kb file")

	baseline := ownershipGetHead(t, dir)

	// Dirty it
	ownershipWriteFile(t, filepath.Join(kbDir, "test.md"), "# Test\nUpdated\n")

	workspace := filepath.Join(dir, ".orch", "workspace", "test")
	os.MkdirAll(workspace, 0755)
	ownershipWriteManifest(t, workspace, baseline)

	result := VerifyOwnershipReconciliation(workspace, dir)
	// .kb/ files are knowledge-backlog (allowed residue) — should pass
	if result != nil && !result.Passed {
		t.Errorf("expected pass for allowed-residue .kb file, got errors: %v", result.Errors)
	}
}

func TestVerifyOwnershipReconciliation_PreBaselineDirt(t *testing.T) {
	dir := t.TempDir()
	ownershipSetupGitRepo(t, dir)

	// Commit a source file
	ownershipWriteFile(t, filepath.Join(dir, "old.go"), "package main\n")
	ownershipRun(t, dir, "git", "add", "old.go")
	ownershipRun(t, dir, "git", "commit", "-m", "add old.go")

	// Dirty it BEFORE baseline
	ownershipWriteFile(t, filepath.Join(dir, "old.go"), "package main\n// dirty before baseline\n")

	baseline := ownershipGetHead(t, dir)

	workspace := filepath.Join(dir, ".orch", "workspace", "test")
	os.MkdirAll(workspace, 0755)
	ownershipWriteManifest(t, workspace, baseline)

	result := VerifyOwnershipReconciliation(workspace, dir)
	// Pre-baseline dirt is NOT this agent's responsibility
	if result != nil && !result.Passed {
		t.Errorf("expected pass for pre-baseline dirty file, got errors: %v", result.Errors)
	}
}

func TestVerifyOwnershipReconciliation_NewUncommittedFile(t *testing.T) {
	dir := t.TempDir()
	ownershipSetupGitRepo(t, dir)

	baseline := ownershipGetHead(t, dir)

	// Create and track a new source file after baseline but don't commit it
	ownershipWriteFile(t, filepath.Join(dir, "new.go"), "package main\n")
	ownershipRun(t, dir, "git", "add", "new.go")
	// Staged but not committed — this is a dirty tracked file

	workspace := filepath.Join(dir, ".orch", "workspace", "test")
	os.MkdirAll(workspace, 0755)
	ownershipWriteManifest(t, workspace, baseline)

	result := VerifyOwnershipReconciliation(workspace, dir)
	if result == nil {
		t.Fatal("expected non-nil result for staged but uncommitted source file")
	}
	if result.Passed {
		t.Error("expected gate to fail for staged but uncommitted source file")
	}
}

// --- Test helpers ---

func ownershipSetupGitRepo(t *testing.T, dir string) {
	t.Helper()
	ownershipRun(t, dir, "git", "init")
	ownershipRun(t, dir, "git", "config", "user.email", "test@test.com")
	ownershipRun(t, dir, "git", "config", "user.name", "Test")
	// Initial commit so HEAD exists
	ownershipWriteFile(t, filepath.Join(dir, ".gitkeep"), "")
	ownershipRun(t, dir, "git", "add", ".gitkeep")
	ownershipRun(t, dir, "git", "commit", "-m", "initial commit")
}

func ownershipGetHead(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	return string(out[:len(out)-1]) // trim newline
}

func ownershipWriteFile(t *testing.T, path, content string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func ownershipRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %v\n%s", args, err, out)
	}
}

func ownershipWriteManifest(t *testing.T, workspace, baseline string) {
	t.Helper()
	content := `{
  "workspace_name": "test",
  "beads_id": "test-123",
  "git_baseline": "` + baseline + `",
  "skill": "feature-impl"
}`
	ownershipWriteFile(t, filepath.Join(workspace, "AGENT_MANIFEST.json"), content)
}
