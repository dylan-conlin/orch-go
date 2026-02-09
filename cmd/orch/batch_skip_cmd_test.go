package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadBatchSkipSetMissingFile(t *testing.T) {
	dir := t.TempDir()

	skipSet, err := readBatchSkipSet(dir)
	if err != nil {
		t.Fatalf("readBatchSkipSet returned error: %v", err)
	}
	if len(skipSet) != 0 {
		t.Fatalf("expected empty skip set, got %d entries", len(skipSet))
	}
}

func TestWriteAndReadBatchSkipSet(t *testing.T) {
	dir := t.TempDir()

	original := map[string]struct{}{
		"orch-go-abc1": {},
		"orch-go-def2": {},
	}

	if err := writeBatchSkipSet(dir, original); err != nil {
		t.Fatalf("writeBatchSkipSet returned error: %v", err)
	}

	loaded, err := readBatchSkipSet(dir)
	if err != nil {
		t.Fatalf("readBatchSkipSet returned error: %v", err)
	}

	if len(loaded) != len(original) {
		t.Fatalf("expected %d entries, got %d", len(original), len(loaded))
	}
	for id := range original {
		if _, ok := loaded[id]; !ok {
			t.Fatalf("missing expected ID %q", id)
		}
	}
}

func TestWriteBatchSkipSetRemovesFileWhenEmpty(t *testing.T) {
	dir := t.TempDir()

	if err := writeBatchSkipSet(dir, map[string]struct{}{"orch-go-abc1": {}}); err != nil {
		t.Fatalf("seed write failed: %v", err)
	}

	if err := writeBatchSkipSet(dir, map[string]struct{}{}); err != nil {
		t.Fatalf("clear write failed: %v", err)
	}

	path := batchSkipPath(dir)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed", filepath.Base(path))
	}
}

func TestBatchAndSkipCommandsRegistered(t *testing.T) {
	commandNames := []string{"batch-complete", "skip-set", "skip-list", "skip-clear"}

	for _, name := range commandNames {
		cmd, _, err := rootCmd.Find([]string{name})
		if err != nil {
			t.Fatalf("expected command %q to be registered: %v", name, err)
		}
		if cmd == nil || cmd.Name() != name {
			t.Fatalf("expected command %q, got %#v", name, cmd)
		}
	}
}
