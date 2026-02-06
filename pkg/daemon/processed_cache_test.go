package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProcessedIssueCache_ShouldProcess_EmptyCache(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "processed-issues.jsonl")

	cache, err := NewProcessedIssueCache(cachePath)
	if err != nil {
		t.Fatalf("NewProcessedIssueCache failed: %v", err)
	}

	// Inject no-op checkers to avoid real network/CLI calls
	cache.sessionChecker = func(beadsID string) bool { return false }
	cache.phaseCompleteChecker = func(beadsID string) (bool, error) { return false, nil }

	// Empty cache with no session and no phase complete should allow processing
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

	// Inject no-op checkers to avoid real network/CLI calls
	cache2.sessionChecker = func(beadsID string) bool { return false }
	cache2.phaseCompleteChecker = func(beadsID string) (bool, error) { return false, nil }

	// Should not allow processing (already in cache from previous session)
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

	// Inject no-op checkers to avoid real network/CLI calls
	cache2.sessionChecker = func(beadsID string) bool { return false }
	cache2.phaseCompleteChecker = func(beadsID string) (bool, error) { return false, nil }

	// Old entry should be pruned (should allow processing)
	if !cache2.ShouldProcess("old-issue") {
		t.Error("Expected old entry to be pruned and allow processing")
	}

	// Recent entry should still block processing
	if cache2.ShouldProcess("recent-issue") {
		t.Error("Expected recent entry to still block processing")
	}
}

func TestProcessedIssueCache_ShouldProcess_SessionDedupBlocks(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "processed-issues.jsonl")

	cache, err := NewProcessedIssueCache(cachePath)
	if err != nil {
		t.Fatalf("NewProcessedIssueCache failed: %v", err)
	}

	// Inject session checker that reports session exists
	cache.sessionChecker = func(beadsID string) bool { return true }
	cache.phaseCompleteChecker = func(beadsID string) (bool, error) { return false, nil }

	if cache.ShouldProcess("test-issue-1") {
		t.Error("Expected ShouldProcess to return false when session exists")
	}
}

func TestProcessedIssueCache_ShouldProcess_PhaseCompleteBlocks(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "processed-issues.jsonl")

	cache, err := NewProcessedIssueCache(cachePath)
	if err != nil {
		t.Fatalf("NewProcessedIssueCache failed: %v", err)
	}

	// Inject checkers: no session, but phase complete
	cache.sessionChecker = func(beadsID string) bool { return false }
	cache.phaseCompleteChecker = func(beadsID string) (bool, error) { return true, nil }

	if cache.ShouldProcess("test-issue-1") {
		t.Error("Expected ShouldProcess to return false when Phase: Complete found")
	}
}

func TestProcessedIssueCache_ShouldProcess_PhaseCompleteErrorFailsSafe(t *testing.T) {
	// When phaseCompleteChecker returns an error, ShouldProcess should
	// return false (don't spawn) to prevent duplicate agents.
	// This is the fail-safe pattern: assume exists on error.
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "processed-issues.jsonl")

	cache, err := NewProcessedIssueCache(cachePath)
	if err != nil {
		t.Fatalf("NewProcessedIssueCache failed: %v", err)
	}

	// Inject checkers: no session, phase complete check errors
	cache.sessionChecker = func(beadsID string) bool { return false }
	cache.phaseCompleteChecker = func(beadsID string) (bool, error) {
		return false, fmt.Errorf("bd command failed")
	}

	if cache.ShouldProcess("test-issue-1") {
		t.Error("Expected ShouldProcess to return false (fail-safe) when phase complete check errors")
	}
}

func TestProcessedIssueCache_ShouldProcess_AllChecksPass(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "processed-issues.jsonl")

	cache, err := NewProcessedIssueCache(cachePath)
	if err != nil {
		t.Fatalf("NewProcessedIssueCache failed: %v", err)
	}

	// Inject checkers: no session, no phase complete, no errors
	cache.sessionChecker = func(beadsID string) bool { return false }
	cache.phaseCompleteChecker = func(beadsID string) (bool, error) { return false, nil }

	if !cache.ShouldProcess("test-issue-1") {
		t.Error("Expected ShouldProcess to return true when all checks pass")
	}
}
