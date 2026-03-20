package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

// codeFileExtensions defines which file extensions count as code files.
var codeFileExtensions = map[string]bool{
	".go":     true,
	".ts":     true,
	".tsx":    true,
	".js":     true,
	".jsx":    true,
	".svelte": true,
	".css":    true,
	".py":     true,
	".sh":     true,
	".sql":    true,
}

// collectAllSnapshots scans the project for code directories and collects snapshots.
func collectAllSnapshots(projectDir string) []events.DirectorySnapshot {
	var snapshots []events.DirectorySnapshot

	// Scan cmd/ subdirectories
	cmdDir := filepath.Join(projectDir, "cmd")
	if entries, err := os.ReadDir(cmdDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				dir := filepath.Join(cmdDir, e.Name())
				label := "cmd/" + e.Name() + "/"
				snap := collectDirectorySnapshot(dir, label)
				if snap.FileCount > 0 {
					snapshots = append(snapshots, snap)
				}
			}
		}
	}

	// Scan pkg/ subdirectories
	pkgDir := filepath.Join(projectDir, "pkg")
	if entries, err := os.ReadDir(pkgDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				dir := filepath.Join(pkgDir, e.Name())
				label := "pkg/" + e.Name() + "/"
				snap := collectDirectorySnapshot(dir, label)
				if snap.FileCount > 0 {
					snapshots = append(snapshots, snap)
				}
			}
		}
	}

	// Scan web/src/ as a single directory
	webSrcDir := filepath.Join(projectDir, "web", "src")
	if info, err := os.Stat(webSrcDir); err == nil && info.IsDir() {
		snap := collectDirectorySnapshot(webSrcDir, "web/src/")
		if snap.FileCount > 0 {
			snapshots = append(snapshots, snap)
		}
	}

	return snapshots
}

// collectDirectorySnapshot walks a directory tree and counts code file lines.
func collectDirectorySnapshot(dir string, label string) events.DirectorySnapshot {
	snap := events.DirectorySnapshot{
		Directory: label,
	}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if !codeFileExtensions[ext] {
			return nil
		}

		lines, err := countFileLines(path)
		if err != nil {
			return nil
		}
		snap.TotalLines += lines
		snap.FileCount++

		if lines > 800 {
			snap.FilesOver800++
		}
		if lines > 1500 {
			snap.FilesOver1500++
		}
		if lines > snap.LargestLines {
			snap.LargestLines = lines
			// Store relative name within the directory
			rel, err := filepath.Rel(dir, path)
			if err == nil {
				snap.LargestFile = rel
			} else {
				snap.LargestFile = filepath.Base(path)
			}
		}

		return nil
	})

	return snap
}

// shouldEmitSnapshot checks if a new snapshot should be emitted based on event history.
// Returns true if no prior snapshot exists or the last one is >6 days old.
func shouldEmitSnapshot(allEvents []events.Event) bool {
	var lastSnapshotTime int64
	for _, e := range allEvents {
		if e.Type == events.EventTypeAccretionSnapshot && e.Timestamp > lastSnapshotTime {
			lastSnapshotTime = e.Timestamp
		}
	}

	if lastSnapshotTime == 0 {
		return true // No prior snapshot
	}

	sixDaysAgo := time.Now().Add(-6 * 24 * time.Hour).Unix()
	return lastSnapshotTime < sixDaysAgo
}

// emitDaemonSnapshot is called from daemon periodic tasks to emit a snapshot
// if the last one is >6 days old. Returns true if a snapshot was emitted.
func emitDaemonSnapshot(logger *events.Logger, projectDir string) bool {
	// Read existing events to check last snapshot time
	eventsPath := events.DefaultLogPath()
	allEvents := parseEventsForSnapshot(eventsPath)

	if !shouldEmitSnapshot(allEvents) {
		return false
	}

	snapshots := collectAllSnapshots(projectDir)
	if len(snapshots) == 0 {
		return false
	}

	data := events.AccretionSnapshotData{
		Directories:  snapshots,
		SnapshotType: "weekly",
	}

	if err := logger.LogAccretionSnapshot(data); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to emit accretion snapshot: %v\n", err)
		return false
	}

	return true
}

// parseEventsForSnapshot reads recent events.jsonl to find accretion snapshots.
// Only reads the last 30 days since snapshots are emitted weekly.
func parseEventsForSnapshot(path string) []events.Event {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var reader io.Reader = f
	since := time.Now().Unix() - 30*86400
	if sr, ok := seekToTimestamp(f, since); ok {
		reader = sr
	}

	var result []events.Event
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e events.Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		// Only keep snapshot events to save memory
		if e.Type == events.EventTypeAccretionSnapshot {
			result = append(result, e)
		}
	}

	return result
}
