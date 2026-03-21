package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSeekToTimestamp_SmallFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "small.jsonl")
	os.WriteFile(path, []byte(`{"type":"test","timestamp":100}`+"\n"), 0644)

	f, _ := os.Open(path)
	defer f.Close()

	_, ok := SeekToTimestamp(f, 500)
	if ok {
		t.Error("expected SeekToTimestamp to skip for small files")
	}
}

func TestSeekToTimestamp_SeeksCorrectly(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "events.jsonl")

	// Write enough data to exceed 4KB threshold
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	baseTS := int64(1000000)
	enc := json.NewEncoder(f)
	for i := 0; i < 200; i++ {
		enc.Encode(Event{
			Type:      "test.event",
			Timestamp: baseTS + int64(i*100),
			Data:      map[string]interface{}{"padding": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"},
		})
	}
	f.Close()

	// Verify file is large enough
	stat, _ := os.Stat(path)
	if stat.Size() < 4096 {
		t.Fatalf("test file too small: %d bytes", stat.Size())
	}

	// Seek to midpoint timestamp
	midTS := baseTS + 10000 // halfway through
	f2, _ := os.Open(path)
	defer f2.Close()

	reader, ok := SeekToTimestamp(f2, midTS)
	if !ok {
		t.Fatal("expected SeekToTimestamp to succeed")
	}
	if reader == nil {
		t.Fatal("expected non-nil reader")
	}
}

func TestSeekToTimestamp_SinceBeforeFirstEvent(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "events.jsonl")

	f, _ := os.Create(path)
	enc := json.NewEncoder(f)
	for i := 0; i < 200; i++ {
		enc.Encode(Event{
			Type:      "test.event",
			Timestamp: int64(1000000 + i*100),
			Data:      map[string]interface{}{"padding": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"},
		})
	}
	f.Close()

	f2, _ := os.Open(path)
	defer f2.Close()

	// Since before all events — should return false (read from start)
	_, ok := SeekToTimestamp(f2, 500)
	if ok {
		t.Error("expected false when since is before first event")
	}
}
