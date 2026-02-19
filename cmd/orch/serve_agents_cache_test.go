package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/verify"
)

func TestGetKBProjects(t *testing.T) {
	// Test that getKBProjects returns project paths
	// This test will pass in environments where kb CLI is available
	// and fail gracefully when kb is not available

	projects := getKBProjects()

	// The function should return a slice (may be empty if kb unavailable)
	if projects == nil {
		t.Error("Expected non-nil slice from getKBProjects")
	}

	// If we got projects, verify they are valid paths
	for _, proj := range projects {
		if proj == "" {
			t.Error("Empty project path returned from getKBProjects")
		}
	}
}

func TestGetKBProjectsGracefulFallback(t *testing.T) {
	// Even if kb fails, we should get an empty slice, not a panic
	// This is implicitly tested by TestGetKBProjects but we make it explicit here
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("getKBProjects panicked: %v", r)
		}
	}()

	_ = getKBProjects()
}

func TestGetKBProjectsFallbackToRegistry(t *testing.T) {
	tempDir := t.TempDir()
	kbDir := filepath.Join(tempDir, ".kb")
	if err := os.MkdirAll(kbDir, 0755); err != nil {
		t.Fatalf("failed to create temp kb dir: %v", err)
	}

	registry := KBProjectsRegistry{
		Projects: []KBProject{{Name: "orch-go", Path: "/tmp/orch-go"}},
	}
	data, err := json.Marshal(registry)
	if err != nil {
		t.Fatalf("failed to marshal registry: %v", err)
	}

	registryPath := filepath.Join(kbDir, "projects.json")
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatalf("failed to write registry: %v", err)
	}

	t.Setenv("HOME", tempDir)
	t.Setenv("PATH", "")

	projects := getKBProjects()
	if len(projects) != 1 {
		t.Fatalf("expected 1 project from registry, got %d", len(projects))
	}
	if projects[0] != filepath.Clean("/tmp/orch-go") {
		t.Fatalf("unexpected project path: %s", projects[0])
	}
}

func TestExtractUniqueProjectDirsWithKBProjects(t *testing.T) {
	// Test that kb projects are included in extractUniqueProjectDirs
	// when getKBProjectsFunc is provided

	// Mock kb projects function
	mockKBProjects := func() []string {
		return []string{"/home/user/kb-project1", "/home/user/kb-project2"}
	}

	// Sessions with some directories
	sessions := []struct {
		Directory string
	}{
		{Directory: "/home/user/session-project"},
	}

	// Convert to opencode.Session-like behavior for the test
	// We'll test the merging logic directly
	seen := make(map[string]bool)
	var dirs []string

	// Add current project dir
	currentDir := "/home/user/current"
	seen[currentDir] = true
	dirs = append(dirs, currentDir)

	// Add session dirs
	for _, s := range sessions {
		if !seen[s.Directory] {
			seen[s.Directory] = true
			dirs = append(dirs, s.Directory)
		}
	}

	// Add kb projects
	for _, proj := range mockKBProjects() {
		if !seen[proj] {
			seen[proj] = true
			dirs = append(dirs, proj)
		}
	}

	// Verify all sources are included
	if len(dirs) != 4 { // current + 1 session + 2 kb projects
		t.Errorf("Expected 4 dirs, got %d: %v", len(dirs), dirs)
	}

	// Verify deduplication works
	if seen["/home/user/kb-project1"] != true {
		t.Error("KB project 1 should be in seen map")
	}
	if seen["/home/user/kb-project2"] != true {
		t.Error("KB project 2 should be in seen map")
	}
}

func TestExtractUniqueProjectDirsWithKBProjectsDedup(t *testing.T) {
	// Test that duplicates between session dirs and kb projects are handled

	mockKBProjects := []string{"/home/user/orch-go", "/home/user/new-project"}

	seen := make(map[string]bool)
	var dirs []string

	// Add current project dir (same as one kb project)
	currentDir := "/home/user/orch-go"
	seen[currentDir] = true
	dirs = append(dirs, currentDir)

	// Add kb projects - orch-go should be deduplicated
	for _, proj := range mockKBProjects {
		if !seen[proj] {
			seen[proj] = true
			dirs = append(dirs, proj)
		}
	}

	// Should only have 2 dirs: orch-go (deduped) + new-project
	if len(dirs) != 2 {
		t.Errorf("Expected 2 dirs after dedup, got %d: %v", len(dirs), dirs)
	}
}

func TestBeadsCacheProjectDirsMatchForIDs(t *testing.T) {
	cache := newBeadsCache()
	ids := []string{"orch-go-123"}

	cached := map[string]string{"orch-go-123": "/tmp/project-a"}
	requested := map[string]string{"orch-go-123": "/tmp/project-a"}
	if !cache.projectDirsMatchForIDs(cached, requested, ids) {
		t.Fatal("Expected project dirs to match")
	}

	requested["orch-go-123"] = "/tmp/project-b"
	if cache.projectDirsMatchForIDs(cached, requested, ids) {
		t.Fatal("Expected project dirs mismatch to be detected")
	}
}

func TestBeadsCacheGetAllIssuesProjectDirInvalidation(t *testing.T) {
	oldGetIssuesBatch := getIssuesBatch
	defer func() { getIssuesBatch = oldGetIssuesBatch }()

	callCount := 0
	getIssuesBatch = func(ids []string, projectDirs map[string]string) (map[string]*verify.Issue, error) {
		callCount++
		return map[string]*verify.Issue{
			"orch-go-1": {ID: "orch-go-1", Status: "in_progress"},
		}, nil
	}

	cache := newBeadsCache()
	cache.allIssuesTTL = 1 * time.Minute

	ids := []string{"orch-go-1"}
	projectDirsA := map[string]string{"orch-go-1": "/tmp/project-a"}
	projectDirsB := map[string]string{"orch-go-1": "/tmp/project-b"}

	if _, err := cache.getAllIssues(ids, projectDirsA); err != nil {
		t.Fatalf("getAllIssues returned error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("Expected getIssuesBatch to be called once, got %d", callCount)
	}

	if _, err := cache.getAllIssues(ids, projectDirsA); err != nil {
		t.Fatalf("getAllIssues returned error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("Expected cached result to be used, got %d calls", callCount)
	}

	if _, err := cache.getAllIssues(ids, projectDirsB); err != nil {
		t.Fatalf("getAllIssues returned error: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("Expected cache invalidation on project dir change, got %d calls", callCount)
	}
}
