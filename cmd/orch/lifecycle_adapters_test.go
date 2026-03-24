package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/agent"
)

// TestAdaptersImplementInterfaces verifies that all adapters satisfy
// the pkg/agent lifecycle interfaces at compile time. The compile-time
// checks in lifecycle_adapters.go (var _ agent.X = (*adapter)(nil))
// already enforce this, but this test documents the expectation.
func TestAdaptersImplementInterfaces(t *testing.T) {
	// These type assertions verify interface compliance.
	// They would fail at compile time if any method is missing.
	var _ agent.BeadsClient = (*beadsAdapter)(nil)
	var _ agent.OpenCodeClient = (*openCodeAdapter)(nil)
	var _ agent.TmuxClient = (*tmuxAdapter)(nil)
	var _ agent.EventLogger = (*eventLoggerAdapter)(nil)
	var _ agent.WorkspaceManager = (*workspaceAdapter)(nil)
}

// TestBuildLifecycleManager verifies the factory constructs a valid manager.
func TestBuildLifecycleManager(t *testing.T) {
	lm := buildLifecycleManager("/tmp/project", "http://localhost:4096", "test-agent", "proj-123")
	if lm == nil {
		t.Fatal("buildLifecycleManager returned nil")
	}
}

func TestCopyBrief_CopiesFile(t *testing.T) {
	tmpDir := t.TempDir()
	wsPath := filepath.Join(tmpDir, "workspace")
	os.MkdirAll(wsPath, 0755)
	os.WriteFile(filepath.Join(wsPath, "BRIEF.md"), []byte("# Brief\nContent here"), 0644)

	adapter := &workspaceAdapter{}
	err := adapter.CopyBrief(wsPath, "orch-go-abc12", tmpDir)
	if err != nil {
		t.Fatalf("CopyBrief() error: %v", err)
	}

	dest := filepath.Join(tmpDir, ".kb", "briefs", "orch-go-abc12.md")
	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("brief not found at %s: %v", dest, err)
	}
	if string(content) != "# Brief\nContent here" {
		t.Errorf("brief content mismatch: got %q", string(content))
	}
}

func TestCopyBrief_NoBriefFile_ReturnsNil(t *testing.T) {
	tmpDir := t.TempDir()
	wsPath := filepath.Join(tmpDir, "workspace")
	os.MkdirAll(wsPath, 0755)

	adapter := &workspaceAdapter{}
	err := adapter.CopyBrief(wsPath, "orch-go-abc12", tmpDir)
	if err != nil {
		t.Fatalf("CopyBrief() should return nil when no BRIEF.md, got: %v", err)
	}
}

func TestCopyBrief_CreatesBriefsDir(t *testing.T) {
	tmpDir := t.TempDir()
	wsPath := filepath.Join(tmpDir, "workspace")
	os.MkdirAll(wsPath, 0755)
	os.WriteFile(filepath.Join(wsPath, "BRIEF.md"), []byte("content"), 0644)

	adapter := &workspaceAdapter{}
	err := adapter.CopyBrief(wsPath, "orch-go-xyz99", tmpDir)
	if err != nil {
		t.Fatalf("CopyBrief() error: %v", err)
	}

	briefsDir := filepath.Join(tmpDir, ".kb", "briefs")
	info, err := os.Stat(briefsDir)
	if err != nil {
		t.Fatalf(".kb/briefs/ not created: %v", err)
	}
	if !info.IsDir() {
		t.Error(".kb/briefs/ is not a directory")
	}
}

func TestCleanStaleBriefs_RemovesOldFiles(t *testing.T) {
	tmpDir := t.TempDir()
	briefsDir := filepath.Join(tmpDir, ".kb", "briefs")
	os.MkdirAll(briefsDir, 0755)

	// Create a "stale" brief (backdate mod time)
	stalePath := filepath.Join(briefsDir, "orch-go-old1.md")
	os.WriteFile(stalePath, []byte("old"), 0644)
	staleTime := time.Now().Add(-31 * 24 * time.Hour)
	os.Chtimes(stalePath, staleTime, staleTime)

	// Create a "fresh" brief
	freshPath := filepath.Join(briefsDir, "orch-go-new1.md")
	os.WriteFile(freshPath, []byte("new"), 0644)

	adapter := &workspaceAdapter{}
	err := adapter.CleanStaleBriefs(tmpDir, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("CleanStaleBriefs() error: %v", err)
	}

	// Stale brief should be removed
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Error("stale brief was NOT removed")
	}

	// Fresh brief should remain
	if _, err := os.Stat(freshPath); err != nil {
		t.Error("fresh brief was incorrectly removed")
	}
}

func TestCleanStaleBriefs_NoBriefsDir_ReturnsNil(t *testing.T) {
	tmpDir := t.TempDir()

	adapter := &workspaceAdapter{}
	err := adapter.CleanStaleBriefs(tmpDir, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("CleanStaleBriefs() should return nil when no briefs dir, got: %v", err)
	}
}
