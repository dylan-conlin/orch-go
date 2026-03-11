package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/spf13/cobra"
)

var harnessSnapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Capture directory-level line count snapshot for accretion velocity tracking",
	Long: `Emit an accretion.snapshot event with per-directory line counts.

Scans code directories (cmd/, pkg/, web/src/) and records:
  - total_lines: sum of all code file lines
  - file_count: number of code files
  - files_over_800: files exceeding 800 lines (accretion risk)
  - files_over_1500: files exceeding 1500 lines (critical hotspot)
  - largest_file: biggest file in the directory

Snapshots enable velocity computation: (snapshot[n].lines - snapshot[n-1].lines) / days_between

Examples:
  orch harness snapshot              # Emit weekly snapshot
  orch harness snapshot --baseline   # Emit baseline snapshot (pre-gate freeze)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHarnessSnapshot()
	},
}

var snapshotBaseline bool

func init() {
	harnessSnapshotCmd.Flags().BoolVar(&snapshotBaseline, "baseline", false, "Mark as baseline snapshot")
	harnessCmd.AddCommand(harnessSnapshotCmd)
}

func runHarnessSnapshot() error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	snapshots := collectAllSnapshots(projectDir)
	if len(snapshots) == 0 {
		fmt.Fprintln(os.Stderr, "No code directories found.")
		return nil
	}

	snapshotType := "weekly"
	if snapshotBaseline {
		snapshotType = "baseline"
	}

	logger := events.NewLogger(events.DefaultLogPath())
	data := events.AccretionSnapshotData{
		Directories:  snapshots,
		SnapshotType: snapshotType,
	}

	if err := logger.LogAccretionSnapshot(data); err != nil {
		return fmt.Errorf("emitting snapshot event: %w", err)
	}

	// Print summary
	totalLines := 0
	totalFiles := 0
	totalOver800 := 0
	totalOver1500 := 0
	for _, s := range snapshots {
		totalLines += s.TotalLines
		totalFiles += s.FileCount
		totalOver800 += s.FilesOver800
		totalOver1500 += s.FilesOver1500
	}

	fmt.Fprintf(os.Stderr, "Snapshot emitted (%s):\n", snapshotType)
	fmt.Fprintf(os.Stderr, "  Directories:   %d\n", len(snapshots))
	fmt.Fprintf(os.Stderr, "  Total files:   %d\n", totalFiles)
	fmt.Fprintf(os.Stderr, "  Total lines:   %d\n", totalLines)
	fmt.Fprintf(os.Stderr, "  Files >800:    %d\n", totalOver800)
	fmt.Fprintf(os.Stderr, "  Files >1500:   %d\n", totalOver1500)
	for _, s := range snapshots {
		fmt.Fprintf(os.Stderr, "  %-20s %5d lines, %d files", s.Directory, s.TotalLines, s.FileCount)
		if s.LargestFile != "" {
			fmt.Fprintf(os.Stderr, " (largest: %s @ %d)", s.LargestFile, s.LargestLines)
		}
		fmt.Fprintln(os.Stderr)
	}

	return nil
}

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

// parseEventsForSnapshot reads events.jsonl and returns all events.
// This is a lightweight parser that only extracts type and timestamp.
func parseEventsForSnapshot(path string) []events.Event {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var result []events.Event
	scanner := bufio.NewScanner(f)
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
