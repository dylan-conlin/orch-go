package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

// TestProcessedCache_SurvivesDaemonRestart verifies that the cache persists
// across daemon restarts - the key success criterion.
func TestProcessedCache_SurvivesDaemonRestart(t *testing.T) {
	// Create temp cache file
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "processed-issues.jsonl")

	// Mock functions to avoid external dependencies
	mockSessionChecker := func(beadsID string) bool { return false }
	mockPhaseChecker := func(beadsID string) (bool, error) { return false, nil }

	// Create first cache instance and mark an issue
	cache1, err := NewProcessedIssueCache(cachePath, DefaultProcessedIssueCacheMaxEntries, DefaultProcessedIssueCacheTTL)
	if err != nil {
		t.Fatalf("Failed to create first cache: %v", err)
	}
	cache1.sessionChecker = mockSessionChecker
	cache1.phaseCompleteChecker = mockPhaseChecker

	// Mark issue as processed
	if err := cache1.MarkProcessed("test-issue-1"); err != nil {
		t.Fatalf("Failed to mark issue: %v", err)
	}

	// Verify it's blocked
	if cache1.ShouldProcess("test-issue-1") {
		t.Error("Expected issue to be blocked after marking")
	}

	// Simulate daemon restart by creating a new cache instance
	cache2, err := NewProcessedIssueCache(cachePath, DefaultProcessedIssueCacheMaxEntries, DefaultProcessedIssueCacheTTL)
	if err != nil {
		t.Fatalf("Failed to create second cache: %v", err)
	}
	cache2.sessionChecker = mockSessionChecker
	cache2.phaseCompleteChecker = mockPhaseChecker

	// Verify issue is still blocked (cache survived restart)
	if cache2.ShouldProcess("test-issue-1") {
		t.Error("Expected issue to remain blocked after daemon restart")
	}
}

// TestProcessedCache_IntegrationWithDaemon verifies the daemon uses the cache correctly.
func TestProcessedCache_IntegrationWithDaemon(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "processed-issues.jsonl")

	// Create daemon with cache
	config := DefaultConfig()
	config.Verbose = true  // Enable verbose to see filtering reasons
	config.Label = ""      // Disable label filtering for this test
	config.GracePeriod = 0 // Disable grace period for deterministic test behavior
	d := NewWithConfig(config)

	// Replace cache with test cache
	cache, err := NewProcessedIssueCache(cachePath, DefaultProcessedIssueCacheMaxEntries, DefaultProcessedIssueCacheTTL)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	cache.sessionChecker = func(beadsID string) bool { return false }
	cache.phaseCompleteChecker = func(beadsID string) (bool, error) { return false, nil }
	d.ProcessedCache = cache

	// Disable legacy SpawnedIssues tracker to test ProcessedCache only
	d.SpawnedIssues = nil

	// Mock functions
	d.listIssuesFunc = func() ([]Issue, error) {
		return []Issue{
			{ID: "test-issue-1", Status: "open", Priority: 0, Title: "Test Issue", IssueType: "task"},
		}, nil
	}
	d.spawnFunc = func(beadsID string) error {
		return nil
	}

	// First spawn should succeed
	result1, err := d.Once()
	if err != nil {
		t.Fatalf("First Once() failed: %v", err)
	}
	if !result1.Processed {
		t.Errorf("Expected first spawn to succeed, got: %v", result1.Message)
	}

	// Verify issue was marked in cache
	if d.ProcessedCache.ShouldProcess("test-issue-1") {
		t.Error("Expected issue to be marked in cache after spawn")
	}

	// Second spawn should be blocked by cache
	result2, err := d.Once()
	if err != nil {
		t.Fatalf("Second Once() failed: %v", err)
	}
	if result2.Processed {
		t.Errorf("Expected second spawn to be blocked by cache, got: %v", result2.Message)
	}
}

// TestProcessedCache_CacheFileLocation verifies the cache file is created in the right place.
func TestProcessedCache_CacheFileLocation(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "processed-issues.jsonl")

	cache, err := NewProcessedIssueCache(cachePath, DefaultProcessedIssueCacheMaxEntries, DefaultProcessedIssueCacheTTL)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Mark an issue to trigger file creation
	if err := cache.MarkProcessed("test-issue-1"); err != nil {
		t.Fatalf("Failed to mark issue: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("Expected cache file to exist")
	}

	// Verify directory was created
	dir := filepath.Dir(cachePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("Expected cache directory to exist")
	}
}
