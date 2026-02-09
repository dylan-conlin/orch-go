package events

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMaybeRotateLogWhenExceedingLimit(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "events.jsonl")

	if err := os.WriteFile(logPath, []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write seed log: %v", err)
	}
	if err := os.Truncate(logPath, MaxEventsLogSizeBytes); err != nil {
		t.Fatalf("truncate log: %v", err)
	}

	if err := maybeRotateLog(logPath, 1); err != nil {
		t.Fatalf("maybeRotateLog: %v", err)
	}

	if _, err := os.Stat(logPath + ".1"); err != nil {
		t.Fatalf("expected rotated file .1 to exist: %v", err)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("expected current log to be moved during rotation")
	}
}

func TestRotateLogFilesKeepsThreeArchives(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "events.jsonl")

	writeFile := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile(logPath, "current\n")
	writeFile(logPath+".1", "arch1\n")
	writeFile(logPath+".2", "arch2\n")
	writeFile(logPath+".3", "arch3\n")

	if err := rotateLogFiles(logPath, MaxRotatedEventLogs); err != nil {
		t.Fatalf("rotateLogFiles: %v", err)
	}

	read := func(path string) string {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		return string(data)
	}

	if got := read(logPath + ".1"); got != "current\n" {
		t.Fatalf(".1 content = %q, want current", got)
	}
	if got := read(logPath + ".2"); got != "arch1\n" {
		t.Fatalf(".2 content = %q, want arch1", got)
	}
	if got := read(logPath + ".3"); got != "arch2\n" {
		t.Fatalf(".3 content = %q, want arch2", got)
	}
}

func TestReadCompactedJSONLReadsChronologicalOrder(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "events.jsonl")

	if err := os.WriteFile(logPath+".2", []byte("older-1\nolder-2\n"), 0o644); err != nil {
		t.Fatalf("write .2: %v", err)
	}
	if err := os.WriteFile(logPath+".1", []byte("old-1\n"), 0o644); err != nil {
		t.Fatalf("write .1: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("new-1\n"), 0o644); err != nil {
		t.Fatalf("write current: %v", err)
	}

	var got []string
	err := ReadCompactedJSONL(logPath, func(line string) error {
		got = append(got, line)
		return nil
	})
	if err != nil {
		t.Fatalf("ReadCompactedJSONL: %v", err)
	}

	want := []string{"older-1", "older-2", "old-1", "new-1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("lines = %#v, want %#v", got, want)
	}
}

func TestLoggerLogRotatesBeforeAppend(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "events.jsonl")
	if err := os.WriteFile(logPath, []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write seed log: %v", err)
	}
	if err := os.Truncate(logPath, MaxEventsLogSizeBytes); err != nil {
		t.Fatalf("truncate log: %v", err)
	}

	logger := NewLogger(logPath)
	if err := logger.Log(Event{Type: "session.spawned", SessionID: "s1", Timestamp: 1}); err != nil {
		t.Fatalf("Log: %v", err)
	}

	if _, err := os.Stat(logPath + ".1"); err != nil {
		t.Fatalf("expected rotated archive to exist: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read current log: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("expected current log to contain new event")
	}
}
