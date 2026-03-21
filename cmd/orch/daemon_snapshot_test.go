package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

func TestCollectDirectorySnapshot(t *testing.T) {
	// Create a temp directory with known file structure
	dir := t.TempDir()

	// Create files with known line counts
	writeFileWithLines(t, filepath.Join(dir, "small.go"), 100)
	writeFileWithLines(t, filepath.Join(dir, "medium.go"), 900)
	writeFileWithLines(t, filepath.Join(dir, "large.go"), 1600)

	snap := collectDirectorySnapshot(dir, "test/")
	if snap.Directory != "test/" {
		t.Errorf("expected directory 'test/', got %q", snap.Directory)
	}
	if snap.FileCount != 3 {
		t.Errorf("expected 3 files, got %d", snap.FileCount)
	}
	// Total lines should be approximately 100+900+1600 = 2600
	if snap.TotalLines < 2500 || snap.TotalLines > 2700 {
		t.Errorf("expected ~2600 total lines, got %d", snap.TotalLines)
	}
	if snap.FilesOver800 != 2 { // medium.go and large.go
		t.Errorf("expected 2 files over 800, got %d", snap.FilesOver800)
	}
	if snap.FilesOver1500 != 1 { // large.go only
		t.Errorf("expected 1 file over 1500, got %d", snap.FilesOver1500)
	}
	if snap.LargestFile != "large.go" {
		t.Errorf("expected largest file 'large.go', got %q", snap.LargestFile)
	}
	if snap.LargestLines < 1590 || snap.LargestLines > 1610 {
		t.Errorf("expected ~1600 largest lines, got %d", snap.LargestLines)
	}
}

func TestCollectDirectorySnapshotSkipsNonCode(t *testing.T) {
	dir := t.TempDir()

	writeFileWithLines(t, filepath.Join(dir, "code.go"), 100)
	writeFileWithLines(t, filepath.Join(dir, "data.json"), 500)
	writeFileWithLines(t, filepath.Join(dir, "readme.md"), 200)

	snap := collectDirectorySnapshot(dir, "test/")
	// Should only count .go files (and other code extensions), not .json or .md
	if snap.FileCount != 1 {
		t.Errorf("expected 1 code file, got %d (should skip .json and .md)", snap.FileCount)
	}
}

func TestCollectDirectorySnapshotSubdirs(t *testing.T) {
	dir := t.TempDir()

	sub := filepath.Join(dir, "sub")
	os.MkdirAll(sub, 0755)
	writeFileWithLines(t, filepath.Join(dir, "root.go"), 100)
	writeFileWithLines(t, filepath.Join(sub, "child.go"), 200)

	snap := collectDirectorySnapshot(dir, "test/")
	if snap.FileCount != 2 {
		t.Errorf("expected 2 files (including subdirs), got %d", snap.FileCount)
	}
	if snap.TotalLines < 290 || snap.TotalLines > 310 {
		t.Errorf("expected ~300 total lines, got %d", snap.TotalLines)
	}
}

func TestCollectAllSnapshots(t *testing.T) {
	// Create a project dir with cmd/orch and pkg/events subdirs
	dir := t.TempDir()

	cmdDir := filepath.Join(dir, "cmd", "orch")
	pkgDir := filepath.Join(dir, "pkg", "events")
	os.MkdirAll(cmdDir, 0755)
	os.MkdirAll(pkgDir, 0755)

	writeFileWithLines(t, filepath.Join(cmdDir, "main.go"), 100)
	writeFileWithLines(t, filepath.Join(pkgDir, "logger.go"), 200)

	snaps := collectAllSnapshots(dir)
	if len(snaps) < 2 {
		t.Errorf("expected at least 2 directory snapshots, got %d", len(snaps))
	}

	// Check that cmd/orch/ and pkg/events/ are present
	dirs := make(map[string]bool)
	for _, s := range snaps {
		dirs[s.Directory] = true
	}
	if !dirs["cmd/orch/"] {
		t.Error("expected cmd/orch/ in snapshots")
	}
	if !dirs["pkg/events/"] {
		t.Error("expected pkg/events/ in snapshots")
	}
}

func TestShouldEmitSnapshot(t *testing.T) {
	// No prior events — should emit
	if !shouldEmitSnapshot(nil) {
		t.Error("expected shouldEmitSnapshot=true with no events")
	}

	// Recent snapshot (1 day ago) — should NOT emit
	recentEvent := events.Event{
		Type:      events.EventTypeAccretionSnapshot,
		Timestamp: time.Now().Add(-24 * time.Hour).Unix(),
	}
	if shouldEmitSnapshot([]events.Event{recentEvent}) {
		t.Error("expected shouldEmitSnapshot=false with recent snapshot")
	}

	// Old snapshot (7 days ago) — should emit
	oldEvent := events.Event{
		Type:      events.EventTypeAccretionSnapshot,
		Timestamp: time.Now().Add(-7 * 24 * time.Hour).Unix(),
	}
	if !shouldEmitSnapshot([]events.Event{oldEvent}) {
		t.Error("expected shouldEmitSnapshot=true with 7-day-old snapshot")
	}
}

func TestLogAccretionSnapshot(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "events.jsonl")
	logger := events.NewLogger(logPath)

	data := events.AccretionSnapshotData{
		Directories: []events.DirectorySnapshot{
			{
				Directory:     "cmd/orch/",
				TotalLines:    14523,
				FileCount:     24,
				FilesOver800:  12,
				FilesOver1500: 2,
				LargestFile:   "stats_cmd.go",
				LargestLines:  1125,
			},
		},
		SnapshotType: "baseline",
	}

	if err := logger.LogAccretionSnapshot(data); err != nil {
		t.Fatalf("LogAccretionSnapshot failed: %v", err)
	}

	// Read back and verify
	content, err := os.ReadFile(logger.CurrentPath())
	if err != nil {
		t.Fatalf("reading events file: %v", err)
	}

	var event events.Event
	if err := json.Unmarshal(content, &event); err != nil {
		t.Fatalf("unmarshaling event: %v", err)
	}

	if event.Type != "accretion.snapshot" {
		t.Errorf("expected type accretion.snapshot, got %q", event.Type)
	}
	if event.Data["snapshot_type"] != "baseline" {
		t.Errorf("expected snapshot_type baseline, got %v", event.Data["snapshot_type"])
	}
}

func writeFileWithLines(t *testing.T, path string, lines int) {
	t.Helper()
	var b strings.Builder
	for i := 0; i < lines; i++ {
		b.WriteString("// line\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		t.Fatalf("writing test file %s: %v", path, err)
	}
}
