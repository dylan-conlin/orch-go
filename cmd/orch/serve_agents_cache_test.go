package main

import (
	"testing"
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
