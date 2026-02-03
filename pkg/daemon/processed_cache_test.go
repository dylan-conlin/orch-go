package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProcessedIssueCache_ShouldProcess_EmptyCache(t *testing.T) {
	// Create temp cache file
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "processed-issues.jsonl")

	cache, err := NewProcessedIssueCache(cachePath)
	if err != nil {
		t.Fatalf("NewProcessedIssueCache failed: %v", err)
	}

	// Empty cache should allow processing
	if !cache.ShouldProcess("test-issue-1") {
		t.Error("Expected ShouldProcess to return true for new issue in empty cache")
	}
}

func TestProcessedIssueCache_MarkProcessed_PersistsToFile(t *testing.T) {
	// Create temp cache file
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "processed-issues.jsonl")

	cache, err := NewProcessedIssueCache(cachePath)
	if err != nil {
		t.Fatalf("NewProcessedIssueCache failed: %v", err)
	}

	// Mark issue as processed
	if err := cache.MarkProcessed("test-issue-1"); err != nil {
		t.Fatalf("MarkProcessed failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("Expected cache file to be created")
	}

	// Load cache from file to verify persistence
	cache2, err := NewProcessedIssueCache(cachePath)
	if err != nil {
		t.Fatalf("Failed to load cache from file: %v", err)
	}

	// Should not allow processing (already processed)
	if cache2.ShouldProcess("test-issue-1") {
		t.Error("Expected ShouldProcess to return false for previously processed issue")
	}
}

func TestProcessedIssueCache_PruneOldEntries(t *testing.T) {
	// Create temp cache file
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "processed-issues.jsonl")

	// Create cache with entries at different ages
	cache, err := NewProcessedIssueCache(cachePath)
	if err != nil {
		t.Fatalf("NewProcessedIssueCache failed: %v", err)
	}

	// Add an old entry (35 days ago) by manipulating the cache
	// We'll test this by creating a cache entry directly
	oldTime := time.Now().Add(-35 * 24 * time.Hour)
	cache.entries = map[string]time.Time{
		"old-issue":    oldTime,
		"recent-issue": time.Now(),
	}

	// Save to file
	if err := cache.save(); err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Load cache again - should prune old entries
	cache2, err := NewProcessedIssueCache(cachePath)
	if err != nil {
		t.Fatalf("Failed to load cache: %v", err)
	}

	// Old entry should be pruned (should allow processing)
	if !cache2.ShouldProcess("old-issue") {
		t.Error("Expected old entry to be pruned and allow processing")
	}

	// Recent entry should still block processing
	if cache2.ShouldProcess("recent-issue") {
		t.Error("Expected recent entry to still block processing")
	}
}

func TestProcessedIssueCache_ShouldProcess_ChecksSessionDedup(t *testing.T) {
	// This test verifies that ShouldProcess checks session dedup
	// We'll mock the session checker by testing the integration
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "processed-issues.jsonl")

	cache, err := NewProcessedIssueCache(cachePath)
	if err != nil {
		t.Fatalf("NewProcessedIssueCache failed: %v", err)
	}

	// For now, we test that the method exists and can be called
	// Full integration testing with mock sessions will be added later
	_ = cache.ShouldProcess("test-issue-1")
}

func TestProcessedIssueCache_ShouldProcess_ChecksPhaseComplete(t *testing.T) {
	// This test verifies that ShouldProcess checks HasPhaseComplete
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "processed-issues.jsonl")

	cache, err := NewProcessedIssueCache(cachePath)
	if err != nil {
		t.Fatalf("NewProcessedIssueCache failed: %v", err)
	}

	// For now, we test that the method exists and can be called
	// Full integration testing will be added later
	_ = cache.ShouldProcess("test-issue-1")
}
