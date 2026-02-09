package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
)

func setDaemonCacheClearAll(t *testing.T, value bool) {
	t.Helper()
	prev := daemonCacheClearAll
	daemonCacheClearAll = value
	t.Cleanup(func() {
		daemonCacheClearAll = prev
	})
}

func seedProcessedCache(t *testing.T, ids ...string) {
	t.Helper()

	cache, err := daemon.NewProcessedIssueCache(
		daemon.DefaultProcessedIssueCachePath(),
		daemon.DefaultProcessedIssueCacheMaxEntries,
		daemon.DefaultProcessedIssueCacheTTL,
	)
	if err != nil {
		t.Fatalf("NewProcessedIssueCache() failed: %v", err)
	}
	for _, id := range ids {
		if err := cache.MarkProcessed(id); err != nil {
			t.Fatalf("MarkProcessed(%s) failed: %v", id, err)
		}
	}
}

func readProcessedCacheCount(t *testing.T) int {
	t.Helper()

	cache, err := daemon.NewProcessedIssueCache(
		daemon.DefaultProcessedIssueCachePath(),
		daemon.DefaultProcessedIssueCacheMaxEntries,
		daemon.DefaultProcessedIssueCacheTTL,
	)
	if err != nil {
		t.Fatalf("NewProcessedIssueCache() failed: %v", err)
	}
	return cache.Count()
}

func TestRunDaemonCacheClear_All(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	seedProcessedCache(t, "orch-go-1", "orch-go-2")

	setDaemonCacheClearAll(t, true)
	if err := runDaemonCacheClear(nil); err != nil {
		t.Fatalf("runDaemonCacheClear() error: %v", err)
	}

	if got := readProcessedCacheCount(t); got != 0 {
		t.Fatalf("cache count = %d, want 0 after --all clear", got)
	}
}

func TestRunDaemonCacheClear_SelectedIDs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	seedProcessedCache(t, "orch-go-1", "orch-go-2", "orch-go-3")

	setDaemonCacheClearAll(t, false)
	if err := runDaemonCacheClear([]string{"orch-go-1", "orch-go-3"}); err != nil {
		t.Fatalf("runDaemonCacheClear() error: %v", err)
	}

	if got := readProcessedCacheCount(t); got != 1 {
		t.Fatalf("cache count = %d, want 1 after clearing two of three IDs", got)
	}
}

func TestRunDaemonCacheClear_RequiresArgsOrAll(t *testing.T) {
	setDaemonCacheClearAll(t, false)
	if err := runDaemonCacheClear(nil); err == nil {
		t.Fatal("expected error when no IDs provided and --all is false")
	}
}
