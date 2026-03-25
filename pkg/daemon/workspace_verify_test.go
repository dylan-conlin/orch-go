package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceExistsForIssue(t *testing.T) {
	t.Run("finds workspace by SPAWN_CONTEXT.md content", func(t *testing.T) {
		tmpDir := t.TempDir()
		wsDir := filepath.Join(tmpDir, ".orch", "workspace", "og-feat-test-25mar-a1b2")
		os.MkdirAll(wsDir, 0755)
		os.WriteFile(filepath.Join(wsDir, "SPAWN_CONTEXT.md"), []byte("You were spawned from beads issue: **proj-abc1**\n"), 0644)

		if !workspaceExistsForIssue("proj-abc1", tmpDir) {
			t.Error("expected workspace to be found by SPAWN_CONTEXT.md content")
		}
	})

	t.Run("returns false when no workspace matches", func(t *testing.T) {
		tmpDir := t.TempDir()
		wsDir := filepath.Join(tmpDir, ".orch", "workspace", "og-feat-other-25mar-c3d4")
		os.MkdirAll(wsDir, 0755)
		os.WriteFile(filepath.Join(wsDir, "SPAWN_CONTEXT.md"), []byte("You were spawned from beads issue: **proj-other**\n"), 0644)

		if workspaceExistsForIssue("proj-abc1", tmpDir) {
			t.Error("expected workspace NOT to be found for different beads ID")
		}
	})

	t.Run("returns false when workspace dir is empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.MkdirAll(filepath.Join(tmpDir, ".orch", "workspace"), 0755)

		if workspaceExistsForIssue("proj-abc1", tmpDir) {
			t.Error("expected false when workspace directory is empty")
		}
	})

	t.Run("returns false for empty inputs", func(t *testing.T) {
		if workspaceExistsForIssue("", "/tmp/proj") {
			t.Error("expected false for empty beads ID")
		}
		if workspaceExistsForIssue("proj-1", "") {
			t.Error("expected false for empty project dir")
		}
	})

	t.Run("skips archived directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivedDir := filepath.Join(tmpDir, ".orch", "workspace", "archived", "og-feat-old")
		os.MkdirAll(archivedDir, 0755)
		os.WriteFile(filepath.Join(archivedDir, "SPAWN_CONTEXT.md"), []byte("beads issue: **proj-old**\n"), 0644)

		if workspaceExistsForIssue("proj-old", tmpDir) {
			t.Error("expected false when workspace is in archived directory")
		}
	})

	t.Run("returns false when no SPAWN_CONTEXT.md exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		wsDir := filepath.Join(tmpDir, ".orch", "workspace", "og-feat-test-25mar-e5f6")
		os.MkdirAll(wsDir, 0755)
		// No SPAWN_CONTEXT.md

		if workspaceExistsForIssue("proj-abc1", tmpDir) {
			t.Error("expected false when workspace has no SPAWN_CONTEXT.md")
		}
	})
}
