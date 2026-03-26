package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRotatedLogPath(t *testing.T) {
	base := "/home/user/.orch/events.jsonl"
	got := RotatedLogPath(base)
	now := time.Now()
	want := filepath.Join("/home/user/.orch", now.Format("events-2006-01")+".jsonl")
	if got != want {
		t.Errorf("RotatedLogPath() = %q, want %q", got, want)
	}
}

func TestEventFiles_LegacyOnly(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "events.jsonl")
	os.WriteFile(legacy, []byte(`{"type":"test","timestamp":1}`+"\n"), 0644)

	files, err := EventFiles(dir, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != legacy {
		t.Errorf("EventFiles() = %v, want [%s]", files, legacy)
	}
}

func TestEventFiles_RotatedAndLegacy(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "events.jsonl")
	rot1 := filepath.Join(dir, "events-2026-01.jsonl")
	rot2 := filepath.Join(dir, "events-2026-02.jsonl")
	rot3 := filepath.Join(dir, "events-2026-03.jsonl")

	for _, f := range []string{legacy, rot1, rot2, rot3} {
		os.WriteFile(f, []byte(""), 0644)
	}

	// No time bounds: all files returned
	files, err := EventFiles(dir, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 4 {
		t.Errorf("expected 4 files, got %d: %v", len(files), files)
	}
}

func TestEventFiles_TimeFiltering(t *testing.T) {
	dir := t.TempDir()
	rot1 := filepath.Join(dir, "events-2026-01.jsonl")
	rot2 := filepath.Join(dir, "events-2026-02.jsonl")
	rot3 := filepath.Join(dir, "events-2026-03.jsonl")

	for _, f := range []string{rot1, rot2, rot3} {
		os.WriteFile(f, []byte(""), 0644)
	}

	// Only February events
	after := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	files, err := EventFiles(dir, after, before)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d: %v", len(files), files)
	}
	if len(files) > 0 && files[0] != rot2 {
		t.Errorf("expected %s, got %s", rot2, files[0])
	}
}

func TestEventFiles_AfterOnly(t *testing.T) {
	dir := t.TempDir()
	rot1 := filepath.Join(dir, "events-2026-01.jsonl")
	rot2 := filepath.Join(dir, "events-2026-02.jsonl")
	rot3 := filepath.Join(dir, "events-2026-03.jsonl")

	for _, f := range []string{rot1, rot2, rot3} {
		os.WriteFile(f, []byte(""), 0644)
	}

	// Events from Feb 15 onwards
	after := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)

	files, err := EventFiles(dir, after, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	// Feb file covers [Feb 1, Mar 1) which includes Feb 15, plus Mar
	if len(files) != 2 {
		t.Errorf("expected 2 files (Feb+Mar), got %d: %v", len(files), files)
	}
}

func TestScanEvents_MultiFile(t *testing.T) {
	dir := t.TempDir()

	// Write events to legacy file (old events)
	legacy := filepath.Join(dir, "events.jsonl")
	writeEvent(t, legacy, Event{
		Type:      EventTypeSessionSpawned,
		Timestamp: time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC).Unix(),
		Data:      map[string]interface{}{"skill": "old-skill"},
	})

	// Write events to rotated file
	rot := filepath.Join(dir, "events-2026-03.jsonl")
	writeEvent(t, rot, Event{
		Type:      EventTypeSessionSpawned,
		Timestamp: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC).Unix(),
		Data:      map[string]interface{}{"skill": "new-skill"},
	})

	// Scan all events
	var count int
	var skills []string
	err := ScanEvents(dir, time.Time{}, time.Time{}, func(e Event) {
		count++
		if s, ok := e.Data["skill"].(string); ok {
			skills = append(skills, s)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("expected 2 events, got %d", count)
	}
}

func TestScanEvents_TimeFiltered(t *testing.T) {
	dir := t.TempDir()

	legacy := filepath.Join(dir, "events.jsonl")
	writeEvent(t, legacy, Event{
		Type:      EventTypeSessionSpawned,
		Timestamp: time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC).Unix(),
		Data:      map[string]interface{}{"skill": "old"},
	})

	rot := filepath.Join(dir, "events-2026-03.jsonl")
	writeEvent(t, rot, Event{
		Type:      EventTypeSessionSpawned,
		Timestamp: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC).Unix(),
		Data:      map[string]interface{}{"skill": "new"},
	})

	// Only scan March 2026 events
	after := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	var count int
	err := ScanEvents(dir, after, time.Time{}, func(e Event) {
		count++
	})
	if err != nil {
		t.Fatal(err)
	}
	// Legacy file is included (mtime is "now" which is after the query bound),
	// but event-level filter skips the old event.
	if count != 1 {
		t.Errorf("expected 1 event in March, got %d", count)
	}
}

func TestScanEventsFromPath_BackwardCompat(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, "events.jsonl")
	writeEvent(t, legacyPath, Event{
		Type:      EventTypeSessionSpawned,
		Timestamp: time.Now().Unix(),
		Data:      map[string]interface{}{"skill": "test"},
	})

	var count int
	err := ScanEventsFromPath(legacyPath, time.Time{}, time.Time{}, func(e Event) {
		count++
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected 1 event, got %d", count)
	}
}

func TestLoggerWritesToRotatedFile(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, "events.jsonl")
	logger := NewLogger(legacyPath)

	logger.Log(Event{
		Type:      EventTypeSessionSpawned,
		Timestamp: time.Now().Unix(),
		Data:      map[string]interface{}{"skill": "test"},
	})

	// Legacy file should NOT exist (new events go to rotated file)
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Error("Logger should write to rotated file, not legacy path")
	}

	// Rotated file should exist
	now := time.Now()
	rotated := filepath.Join(dir, now.Format("events-2006-01")+".jsonl")
	if _, err := os.Stat(rotated); os.IsNotExist(err) {
		t.Errorf("Expected rotated file at %s", rotated)
	}
}

func TestComputeLearning_WithRotatedFiles(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, "events.jsonl")

	// Write events via logger (goes to rotated file)
	logger := NewLogger(legacyPath)
	logger.Log(Event{
		Type:      EventTypeSessionSpawned,
		Timestamp: time.Now().Unix(),
		Data:      map[string]interface{}{"skill": "feature-impl"},
	})
	logger.Log(Event{
		Type:      EventTypeAgentCompleted,
		Timestamp: time.Now().Unix(),
		Data:      map[string]interface{}{"skill": "feature-impl", "outcome": "success"},
	})

	// ComputeLearning should find events in rotated file
	store, err := ComputeLearning(legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	sl, ok := store.Skills["feature-impl"]
	if !ok {
		t.Fatal("skill 'feature-impl' not found")
	}
	if sl.SpawnCount != 1 {
		t.Errorf("SpawnCount = %d, want 1", sl.SpawnCount)
	}
	if sl.SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1", sl.SuccessCount)
	}
}

func TestEventFiles_LegacySkippedByMtime(t *testing.T) {
	dir := t.TempDir()

	// Create a legacy file and set its mtime to January 2025
	legacy := filepath.Join(dir, "events.jsonl")
	os.WriteFile(legacy, []byte(`{"type":"test","timestamp":1}`+"\n"), 0644)
	oldTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	os.Chtimes(legacy, oldTime, oldTime)

	// Create a rotated file for March 2026
	rot := filepath.Join(dir, "events-2026-03.jsonl")
	os.WriteFile(rot, []byte(""), 0644)

	// Query with after=March 1 2026 — legacy file (mtime Jan 2025) should be skipped
	after := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	files, err := EventFiles(dir, after, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file (only rotated), got %d: %v", len(files), files)
	}
	if len(files) > 0 && files[0] != rot {
		t.Errorf("expected %s, got %s", rot, files[0])
	}
}

func TestEventFiles_LegacyIncludedWhenRecent(t *testing.T) {
	dir := t.TempDir()

	// Create a legacy file with recent mtime (default — "now")
	legacy := filepath.Join(dir, "events.jsonl")
	os.WriteFile(legacy, []byte(`{"type":"test","timestamp":1}`+"\n"), 0644)

	// Query with after=7 days ago — legacy file mtime is "now", should be included
	after := time.Now().AddDate(0, 0, -7)
	files, err := EventFiles(dir, after, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range files {
		if f == legacy {
			found = true
		}
	}
	if !found {
		t.Errorf("expected legacy file to be included when mtime is recent, got: %v", files)
	}
}

func TestScanEvents_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	var count int
	err := ScanEvents(dir, time.Time{}, time.Time{}, func(e Event) {
		count++
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("expected 0 events from empty dir, got %d", count)
	}
}

func writeEvent(t *testing.T, path string, event Event) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	data, _ := json.Marshal(event)
	f.Write(append(data, '\n'))
}
