package events

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RotatedLogPath returns the path for the current month's rotated event file.
// Format: events-YYYY-MM.jsonl in the same directory as the base path.
func RotatedLogPath(basePath string) string {
	dir := filepath.Dir(basePath)
	now := time.Now()
	return filepath.Join(dir, now.Format("events-2006-01")+".jsonl")
}

// EventsDir returns the directory containing event files, derived from the base path.
func EventsDir(basePath string) string {
	return filepath.Dir(basePath)
}

// EventFiles returns sorted event file paths that may contain events in the given
// time window [after, before). Includes the legacy events.jsonl if it exists.
// Pass zero time for either bound to leave it open-ended.
func EventFiles(dir string, after, before time.Time) ([]string, error) {
	// Collect candidate files
	var files []string

	// Legacy events.jsonl (contains events from before rotation started).
	// Skip it when its mtime is before the query's "after" bound — if no new
	// events have been written to the legacy file since before the window,
	// it cannot contain any relevant events.
	legacy := filepath.Join(dir, "events.jsonl")
	if info, err := os.Stat(legacy); err == nil {
		if after.IsZero() || !info.ModTime().Before(after) {
			files = append(files, legacy)
		}
	}

	// Rotated files: events-YYYY-MM.jsonl
	matches, err := filepath.Glob(filepath.Join(dir, "events-????-??.jsonl"))
	if err != nil {
		return nil, err
	}

	// Filter rotated files by time window
	for _, path := range matches {
		base := filepath.Base(path)
		// Parse month from filename: events-2026-03.jsonl
		monthStr := strings.TrimPrefix(base, "events-")
		monthStr = strings.TrimSuffix(monthStr, ".jsonl")
		fileMonth, err := time.Parse("2006-01", monthStr)
		if err != nil {
			continue // skip files that don't match format
		}

		// File covers [fileMonth, fileMonth+1month)
		fileEnd := fileMonth.AddDate(0, 1, 0)

		// Skip if file's month ends at or before the 'after' bound
		if !after.IsZero() && !fileEnd.After(after) {
			continue
		}
		// Skip if file's month starts at or after the 'before' bound
		if !before.IsZero() && !fileMonth.Before(before) {
			continue
		}

		files = append(files, path)
	}

	sort.Strings(files)
	return files, nil
}

// multiFileReader reads from multiple files sequentially.
type multiFileReader struct {
	files   []string
	idx     int
	current *os.File
}

// OpenEventFiles returns an io.ReadCloser that reads from all relevant event files
// for the given time window. The legacy events.jsonl is always included (it has no
// month boundary). Pass zero times to read all files.
func OpenEventFiles(dir string, after, before time.Time) (io.ReadCloser, error) {
	files, err := EventFiles(dir, after, before)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return io.NopCloser(strings.NewReader("")), nil
	}
	return &multiFileReader{files: files}, nil
}

func (r *multiFileReader) Read(p []byte) (int, error) {
	for {
		if r.current == nil {
			if r.idx >= len(r.files) {
				return 0, io.EOF
			}
			f, err := os.Open(r.files[r.idx])
			if err != nil {
				// Skip files that can't be opened
				r.idx++
				continue
			}
			r.current = f
		}

		n, err := r.current.Read(p)
		if err == io.EOF {
			r.current.Close()
			r.current = nil
			r.idx++
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}

func (r *multiFileReader) Close() error {
	if r.current != nil {
		return r.current.Close()
	}
	return nil
}

// ScanEvents reads events from all relevant files in the directory for the given
// time window and calls fn for each event. This is the primary way to read events
// from rotated files. Pass zero times to read all events.
func ScanEvents(dir string, after, before time.Time, fn func(Event)) error {
	reader, err := OpenEventFiles(dir, after, before)
	if err != nil {
		return err
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 256*1024), 256*1024)
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}

		// Apply time filter at event level (file-level filter is coarse)
		ts := time.Unix(event.Timestamp, 0)
		if !after.IsZero() && ts.Before(after) {
			continue
		}
		if !before.IsZero() && !ts.Before(before) {
			continue
		}

		fn(event)
	}
	return scanner.Err()
}

// ScanEventsFromPath is a convenience wrapper that derives the events directory
// from a legacy events.jsonl path. This makes it easy to migrate callers that
// currently pass DefaultLogPath().
func ScanEventsFromPath(eventsPath string, after, before time.Time, fn func(Event)) error {
	dir := filepath.Dir(eventsPath)
	return ScanEvents(dir, after, before, fn)
}
