package verify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

func TestCalculateOverrideTrend_Empty(t *testing.T) {
	// Point at a directory with no event files
	dir := t.TempDir()
	t.Setenv("ORCH_EVENTS_PATH", filepath.Join(dir, "events.jsonl"))

	trend, err := CalculateOverrideTrend(7)
	if err != nil {
		t.Fatal(err)
	}
	if trend.CurrentCount != 0 || trend.PreviousCount != 0 {
		t.Errorf("expected zero counts, got current=%d previous=%d", trend.CurrentCount, trend.PreviousCount)
	}
	if trend.Direction != "flat" {
		t.Errorf("expected direction=flat, got %s", trend.Direction)
	}
}

func TestCalculateOverrideTrend_CountsCorrectly(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "events.jsonl")
	t.Setenv("ORCH_EVENTS_PATH", eventsPath)

	now := time.Now()

	// Write events to the current month's rotated file (where ScanEventsFromPath will find them)
	rotated := filepath.Join(dir, now.Format("events-2006-01")+".jsonl")

	writeTestEvent(t, rotated, events.Event{
		Type:      events.EventTypeVerificationBypassed,
		Timestamp: now.Add(-2 * 24 * time.Hour).Unix(), // 2 days ago → current window
	})
	writeTestEvent(t, rotated, events.Event{
		Type:      events.EventTypeVerificationBypassed,
		Timestamp: now.Add(-3 * 24 * time.Hour).Unix(), // 3 days ago → current window
	})
	writeTestEvent(t, rotated, events.Event{
		Type:      events.EventTypeVerificationBypassed,
		Timestamp: now.Add(-10 * 24 * time.Hour).Unix(), // 10 days ago → previous window
	})
	// Unrelated event — should be ignored
	writeTestEvent(t, rotated, events.Event{
		Type:      events.EventTypeSessionSpawned,
		Timestamp: now.Add(-1 * 24 * time.Hour).Unix(),
	})

	trend, err := CalculateOverrideTrend(7)
	if err != nil {
		t.Fatal(err)
	}
	if trend.CurrentCount != 2 {
		t.Errorf("CurrentCount = %d, want 2", trend.CurrentCount)
	}
	if trend.PreviousCount != 1 {
		t.Errorf("PreviousCount = %d, want 1", trend.PreviousCount)
	}
	if trend.Direction != "up" {
		t.Errorf("Direction = %q, want \"up\"", trend.Direction)
	}
	if trend.Delta != 1 {
		t.Errorf("Delta = %d, want 1", trend.Delta)
	}
}

func TestCalculateOverrideTrend_SkipsLegacyWhenStale(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "events.jsonl")
	t.Setenv("ORCH_EVENTS_PATH", eventsPath)

	now := time.Now()

	// Write an old bypass event to the legacy file and set its mtime to the past
	writeTestEvent(t, eventsPath, events.Event{
		Type:      events.EventTypeVerificationBypassed,
		Timestamp: now.Add(-2 * 24 * time.Hour).Unix(),
	})
	oldTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	os.Chtimes(eventsPath, oldTime, oldTime)

	// With a 7-day window, previousStart is 14 days ago. The legacy file's mtime
	// (Jan 2025) is before that, so EventFiles should skip it entirely.
	trend, err := CalculateOverrideTrend(7)
	if err != nil {
		t.Fatal(err)
	}
	// The event was in the legacy file which should be skipped due to stale mtime
	if trend.CurrentCount != 0 {
		t.Errorf("CurrentCount = %d, want 0 (legacy file should be skipped)", trend.CurrentCount)
	}
}

func writeTestEvent(t *testing.T, path string, event events.Event) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	data, _ := json.Marshal(event)
	f.Write(append(data, '\n'))
}
