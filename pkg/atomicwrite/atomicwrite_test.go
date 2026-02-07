package atomicwrite

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteFile_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if err := WriteFile(path, []byte("hello\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Verify content
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(data) != "hello\n" {
		t.Errorf("content = %q, want %q", string(data), "hello\n")
	}

	// Verify no temp file left behind
	tmpPath := path + TempSuffix
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("temp file %s should not exist after successful write", tmpPath)
	}
}

func TestWriteFile_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if err := WriteFile(path, []byte("first"), 0644); err != nil {
		t.Fatalf("first WriteFile failed: %v", err)
	}

	if err := WriteFile(path, []byte("second"), 0644); err != nil {
		t.Fatalf("second WriteFile failed: %v", err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "second" {
		t.Errorf("content = %q, want %q", string(data), "second")
	}
}

func TestWriteFile_ErrorOnBadDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent", "subdir", "test.txt")

	err := WriteFile(path, []byte("hello"), 0644)
	if err == nil {
		t.Fatal("expected error writing to non-existent directory")
	}
}

func TestWriteFileWithDir_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "deep", "test.txt")

	if err := WriteFileWithDir(path, []byte("hello"), 0644); err != nil {
		t.Fatalf("WriteFileWithDir failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("content = %q, want %q", string(data), "hello")
	}
}

func TestWriteFile_CleansTempOnRenameError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target")
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	err := WriteFile(path, []byte("hello"), 0644)
	if err == nil {
		t.Fatal("expected error renaming over directory")
	}

	tmpPath := path + TempSuffix
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("temp file %s should have been cleaned up after rename error", tmpPath)
	}
}

func TestCleanupStaleTempFiles_RemovesStaleFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a stale .tmp file with old mtime
	staleTmp := filepath.Join(dir, "data.json.tmp")
	if err := os.WriteFile(staleTmp, []byte("stale"), 0644); err != nil {
		t.Fatalf("write stale temp: %v", err)
	}
	// Set mtime to 1 minute ago (well past StaleThreshold)
	oldTime := time.Now().Add(-1 * time.Minute)
	os.Chtimes(staleTmp, oldTime, oldTime)

	// Create a non-tmp file (should not be removed)
	normalFile := filepath.Join(dir, "data.json")
	if err := os.WriteFile(normalFile, []byte("keep"), 0644); err != nil {
		t.Fatalf("write normal file: %v", err)
	}

	cleaned, errs := CleanupStaleTempFiles(dir)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if cleaned != 1 {
		t.Errorf("cleaned = %d, want 1", cleaned)
	}

	// Stale tmp should be gone
	if _, err := os.Stat(staleTmp); !os.IsNotExist(err) {
		t.Error("stale tmp file should have been removed")
	}

	// Normal file should still exist
	if _, err := os.Stat(normalFile); os.IsNotExist(err) {
		t.Error("normal file should still exist")
	}
}

func TestCleanupStaleTempFiles_SkipsFreshFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a fresh .tmp file (just written)
	freshTmp := filepath.Join(dir, "fresh.tmp")
	if err := os.WriteFile(freshTmp, []byte("fresh"), 0644); err != nil {
		t.Fatalf("write fresh temp: %v", err)
	}

	cleaned, errs := CleanupStaleTempFiles(dir)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if cleaned != 0 {
		t.Errorf("cleaned = %d, want 0 (fresh file should not be removed)", cleaned)
	}

	// Fresh tmp should still exist
	if _, err := os.Stat(freshTmp); os.IsNotExist(err) {
		t.Error("fresh tmp file should still exist")
	}
}

func TestCleanupStaleTempFiles_NonexistentDir(t *testing.T) {
	cleaned, errs := CleanupStaleTempFiles("/nonexistent/path")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors for nonexistent dir: %v", errs)
	}
	if cleaned != 0 {
		t.Errorf("cleaned = %d, want 0", cleaned)
	}
}

func TestCleanupStaleTempFilesInWorkspaces(t *testing.T) {
	root := t.TempDir()

	// Create workspace subdirectories
	ws1 := filepath.Join(root, "workspace-1")
	ws2 := filepath.Join(root, "workspace-2")
	os.Mkdir(ws1, 0755)
	os.Mkdir(ws2, 0755)

	// Create stale tmp files in each workspace
	stale1 := filepath.Join(ws1, ".session_id.tmp")
	stale2 := filepath.Join(ws2, ".tier.tmp")
	os.WriteFile(stale1, []byte("old"), 0644)
	os.WriteFile(stale2, []byte("old"), 0644)

	// Set old mtimes
	oldTime := time.Now().Add(-1 * time.Minute)
	os.Chtimes(stale1, oldTime, oldTime)
	os.Chtimes(stale2, oldTime, oldTime)

	cleaned, errs := CleanupStaleTempFilesInWorkspaces(root)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if cleaned != 2 {
		t.Errorf("cleaned = %d, want 2", cleaned)
	}
}

func TestCleanupStaleTempFilesInWorkspaces_NonexistentRoot(t *testing.T) {
	cleaned, errs := CleanupStaleTempFilesInWorkspaces("/nonexistent/root")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if cleaned != 0 {
		t.Errorf("cleaned = %d, want 0", cleaned)
	}
}
