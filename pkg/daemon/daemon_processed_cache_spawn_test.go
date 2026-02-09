package daemon

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func newProcessedCacheForSpawnTests(t *testing.T) *ProcessedIssueCache {
	t.Helper()

	cache, err := NewProcessedIssueCache(
		filepath.Join(t.TempDir(), "processed-issues.jsonl"),
		DefaultProcessedIssueCacheMaxEntries,
		DefaultProcessedIssueCacheTTL,
	)
	if err != nil {
		t.Fatalf("NewProcessedIssueCache() failed: %v", err)
	}

	cache.sessionChecker = func(string) bool { return false }
	cache.phaseCompleteChecker = func(string) (bool, error) { return false, nil }

	return cache
}

func TestDaemon_OnceExcluding_ProcessedCacheMarkedAfterSuccessfulSpawn(t *testing.T) {
	cache := newProcessedCacheForSpawnTests(t)

	cacheBlockedDuringSpawn := false
	d := &Daemon{
		Config:         Config{Label: "triage:ready"},
		ProcessedCache: cache,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{{
				ID:        "issue-1",
				Title:     "Test",
				Priority:  0,
				IssueType: "bug",
				Status:    "open",
				Labels:    []string{"triage:ready"},
			}}, nil
		},
		spawnFunc: func(beadsID string) error {
			cacheBlockedDuringSpawn = !cache.ShouldProcess(beadsID)
			return nil
		},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() error: %v", err)
	}
	if !result.Processed {
		t.Fatalf("expected Processed=true, got message: %s", result.Message)
	}
	if cacheBlockedDuringSpawn {
		t.Fatal("processed cache should not block issue during spawn execution")
	}
	if cache.ShouldProcess("issue-1") {
		t.Fatal("issue should be marked in processed cache after successful spawn")
	}
}

func TestDaemon_OnceExcluding_ProcessedCacheNotMarkedOnSpawnFailure(t *testing.T) {
	cache := newProcessedCacheForSpawnTests(t)

	cacheBlockedDuringSpawn := false
	d := &Daemon{
		Config:         Config{Label: "triage:ready"},
		ProcessedCache: cache,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{{
				ID:        "issue-1",
				Title:     "Test",
				Priority:  0,
				IssueType: "bug",
				Status:    "open",
				Labels:    []string{"triage:ready"},
			}}, nil
		},
		spawnFunc: func(beadsID string) error {
			cacheBlockedDuringSpawn = !cache.ShouldProcess(beadsID)
			return fmt.Errorf("spawn failed")
		},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() error: %v", err)
	}
	if result.Processed {
		t.Fatalf("expected Processed=false on spawn failure")
	}
	if cacheBlockedDuringSpawn {
		t.Fatal("processed cache should not block issue during failed spawn")
	}
	if !cache.ShouldProcess("issue-1") {
		t.Fatal("issue should remain unmarked in processed cache after spawn failure")
	}
}

func TestDaemon_CrossProjectOnceExcluding_ProcessedCacheMarkedAfterSuccessfulSpawn(t *testing.T) {
	cache := newProcessedCacheForSpawnTests(t)

	cacheBlockedDuringSpawn := false
	d := &Daemon{
		Config:         Config{Label: "triage:ready", CrossProject: true},
		ProcessedCache: cache,
		listProjectsFunc: func() ([]Project, error) {
			return []Project{{Name: "proj", Path: "/tmp/proj"}}, nil
		},
		listIssuesForProjectFunc: func(string) ([]Issue, error) {
			return []Issue{{
				ID:        "issue-1",
				Title:     "Test",
				Priority:  0,
				IssueType: "bug",
				Status:    "open",
				Labels:    []string{"triage:ready"},
			}}, nil
		},
		spawnForProjectFunc: func(beadsID, projectPath string) error {
			cacheBlockedDuringSpawn = !cache.ShouldProcess(beadsID)
			return nil
		},
	}

	result, err := d.CrossProjectOnceExcluding(nil)
	if err != nil {
		t.Fatalf("CrossProjectOnceExcluding() error: %v", err)
	}
	if !result.Processed {
		t.Fatalf("expected Processed=true, got message: %s", result.Message)
	}
	if cacheBlockedDuringSpawn {
		t.Fatal("processed cache should not block issue during cross-project spawn execution")
	}
	if cache.ShouldProcess("issue-1") {
		t.Fatal("issue should be marked in processed cache after successful cross-project spawn")
	}
}

func TestDaemon_CrossProjectOnceExcluding_ProcessedCacheNotMarkedOnSpawnFailure(t *testing.T) {
	cache := newProcessedCacheForSpawnTests(t)

	cacheBlockedDuringSpawn := false
	d := &Daemon{
		Config:         Config{Label: "triage:ready", CrossProject: true},
		ProcessedCache: cache,
		listProjectsFunc: func() ([]Project, error) {
			return []Project{{Name: "proj", Path: "/tmp/proj"}}, nil
		},
		listIssuesForProjectFunc: func(string) ([]Issue, error) {
			return []Issue{{
				ID:        "issue-1",
				Title:     "Test",
				Priority:  0,
				IssueType: "bug",
				Status:    "open",
				Labels:    []string{"triage:ready"},
			}}, nil
		},
		spawnForProjectFunc: func(beadsID, projectPath string) error {
			cacheBlockedDuringSpawn = !cache.ShouldProcess(beadsID)
			return fmt.Errorf("spawn failed")
		},
	}

	result, err := d.CrossProjectOnceExcluding(nil)
	if err != nil {
		t.Fatalf("CrossProjectOnceExcluding() error: %v", err)
	}
	if result.Processed {
		t.Fatalf("expected Processed=false on spawn failure")
	}
	if cacheBlockedDuringSpawn {
		t.Fatal("processed cache should not block issue during failed cross-project spawn")
	}
	if !cache.ShouldProcess("issue-1") {
		t.Fatal("issue should remain unmarked in processed cache after failed cross-project spawn")
	}
}

func TestDaemon_OnceExcluding_RejectedIssueNotCached(t *testing.T) {
	cache := newProcessedCacheForSpawnTests(t)

	labels := []string{}
	spawnCalls := 0
	d := &Daemon{
		Config:         Config{Label: "triage:ready"},
		ProcessedCache: cache,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{{
				ID:        "issue-1",
				Title:     "Test",
				Priority:  0,
				IssueType: "bug",
				Status:    "open",
				Labels:    labels,
			}}, nil
		},
		spawnFunc: func(string) error {
			spawnCalls++
			return nil
		},
	}

	result1, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("first OnceExcluding() error: %v", err)
	}
	if result1.Processed {
		t.Fatal("expected first call to reject issue without required label")
	}
	if spawnCalls != 0 {
		t.Fatalf("spawn should not be called for rejected issue, got %d calls", spawnCalls)
	}
	if !cache.ShouldProcess("issue-1") {
		t.Fatal("rejected issue should not be marked in processed cache")
	}

	labels = []string{"triage:ready"}

	result2, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("second OnceExcluding() error: %v", err)
	}
	if !result2.Processed {
		t.Fatalf("expected issue to be spawnable after adding required label, got: %s", result2.Message)
	}
	if spawnCalls != 1 {
		t.Fatalf("expected one spawn call after label added, got %d", spawnCalls)
	}
}

func TestDaemon_OnceExcluding_ReopenedIssueRespawnsEvenWhenCached(t *testing.T) {
	cache := newProcessedCacheForSpawnTests(t)

	processedAt := time.Now().Add(-1 * time.Hour).UTC().Truncate(time.Second)
	cache.entries["issue-1"] = processedAt

	spawnCalls := 0
	d := &Daemon{
		Config:         Config{Label: "triage:ready"},
		ProcessedCache: cache,
		listIssuesFunc: func() ([]Issue, error) {
			return []Issue{{
				ID:        "issue-1",
				Title:     "Reopened issue",
				Priority:  0,
				IssueType: "bug",
				Status:    "open",
				Labels:    []string{"triage:ready"},
				UpdatedAt: processedAt.Add(5 * time.Minute).Format(time.RFC3339Nano),
			}}, nil
		},
		spawnFunc: func(string) error {
			spawnCalls++
			return nil
		},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() error: %v", err)
	}
	if !result.Processed {
		t.Fatalf("expected reopened issue to respawn, got: %s", result.Message)
	}
	if spawnCalls != 1 {
		t.Fatalf("expected one spawn call for reopened issue, got %d", spawnCalls)
	}
}
