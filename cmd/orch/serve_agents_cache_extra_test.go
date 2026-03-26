package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/execution"
)

func TestExtractUniqueProjectDirs(t *testing.T) {
	kbProjectCount := len(getKBProjects())

	tests := []struct {
		name              string
		currentProjectDir string
		sessionDirs       []string
		minExpectedCount  int
	}{
		{
			name:              "empty sessions with current dir",
			currentProjectDir: "/home/user/project1",
			sessionDirs:       []string{},
			minExpectedCount:  1,
		},
		{
			name:              "single session same as current",
			currentProjectDir: "/home/user/project1",
			sessionDirs:       []string{"/home/user/project1"},
			minExpectedCount:  1,
		},
		{
			name:              "multiple sessions different dirs",
			currentProjectDir: "/home/user/project1",
			sessionDirs:       []string{"/home/user/project2", "/home/user/project3"},
			minExpectedCount:  3,
		},
		{
			name:              "duplicate session directories",
			currentProjectDir: "/home/user/project1",
			sessionDirs:       []string{"/home/user/project2", "/home/user/project2", "/home/user/project3"},
			minExpectedCount:  3,
		},
		{
			name:              "empty current dir",
			currentProjectDir: "",
			sessionDirs:       []string{"/home/user/project2", "/home/user/project3"},
			minExpectedCount:  2,
		},
		{
			name:              "empty session dir skipped",
			currentProjectDir: "/home/user/project1",
			sessionDirs:       []string{"", "/home/user/project2"},
			minExpectedCount:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessions := make([]execution.SessionInfo, len(tt.sessionDirs))
			for i, dir := range tt.sessionDirs {
				sessions[i] = execution.SessionInfo{Directory: dir}
			}

			result := extractUniqueProjectDirs(sessions, tt.currentProjectDir)

			if len(result) < tt.minExpectedCount {
				t.Errorf("Expected at least %d project dirs, got %d: %v", tt.minExpectedCount, len(result), result)
			}

			if tt.currentProjectDir != "" && len(result) > 0 {
				if result[0] != filepath.Clean(tt.currentProjectDir) {
					t.Errorf("Expected current project dir %q to be first, got %q", tt.currentProjectDir, result[0])
				}
			}

			if kbProjectCount > 0 {
				foundKBProject := false
				kbProjects := getKBProjects()
				for _, proj := range kbProjects {
					for _, dir := range result {
						if dir == proj {
							foundKBProject = true
							break
						}
					}
					if foundKBProject {
						break
					}
				}
				if !foundKBProject && len(kbProjects) > 0 {
					t.Error("Expected at least one kb project to be included in result")
				}
			}
		})
	}
}

func TestBuildWorkspaceCache(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	testCases := []struct {
		workspaceName string
		beadsID       string
		projectDir    string
	}{
		{
			workspaceName: "og-feat-test1-26dec",
			beadsID:       "orch-go-abc1",
			projectDir:    "/home/user/orch-go",
		},
		{
			workspaceName: "og-feat-test2-26dec",
			beadsID:       "kb-cli-def2",
			projectDir:    "/home/user/kb-cli",
		},
	}

	for _, tc := range testCases {
		wsPath := filepath.Join(workspaceDir, tc.workspaceName)
		if err := os.MkdirAll(wsPath, 0755); err != nil {
			t.Fatalf("Failed to create workspace %s: %v", tc.workspaceName, err)
		}

		spawnContext := fmt.Sprintf(`TASK: Test task

You were spawned from beads issue: **%s**

PROJECT_DIR: %s

AUTHORITY: Standard
`, tc.beadsID, tc.projectDir)
		if err := os.WriteFile(filepath.Join(wsPath, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
			t.Fatalf("Failed to create SPAWN_CONTEXT.md for %s: %v", tc.workspaceName, err)
		}
	}

	cache := buildWorkspaceCache(tmpDir)

	if len(cache.beadsToWorkspace) != 2 {
		t.Errorf("Expected 2 entries in beadsToWorkspace, got %d", len(cache.beadsToWorkspace))
	}

	if wsPath, ok := cache.beadsToWorkspace["orch-go-abc1"]; !ok {
		t.Error("Expected orch-go-abc1 in beadsToWorkspace")
	} else if !filepath.IsAbs(wsPath) {
		t.Errorf("Expected absolute path for workspace, got %s", wsPath)
	}

	if projDir := cache.beadsToProjectDir["orch-go-abc1"]; projDir != "/home/user/orch-go" {
		t.Errorf("Expected projectDir /home/user/orch-go, got %s", projDir)
	}

	if projDir := cache.beadsToProjectDir["kb-cli-def2"]; projDir != "/home/user/kb-cli" {
		t.Errorf("Expected projectDir /home/user/kb-cli, got %s", projDir)
	}

	if len(cache.workspaceEntryToPath) != 2 {
		t.Errorf("Expected 2 entries in workspaceEntryToPath, got %d", len(cache.workspaceEntryToPath))
	}
}

func TestBuildMultiProjectWorkspaceCache(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	wsDir1 := filepath.Join(tmpDir1, ".orch", "workspace")
	if err := os.MkdirAll(wsDir1, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir 1: %v", err)
	}

	ws1Path := filepath.Join(wsDir1, "og-feat-test1-26dec")
	if err := os.MkdirAll(ws1Path, 0755); err != nil {
		t.Fatalf("Failed to create workspace 1: %v", err)
	}
	spawnContext1 := `TASK: Test task 1

You were spawned from beads issue: **proj1-abc1**

PROJECT_DIR: /home/user/project1
`
	if err := os.WriteFile(filepath.Join(ws1Path, "SPAWN_CONTEXT.md"), []byte(spawnContext1), 0644); err != nil {
		t.Fatalf("Failed to create SPAWN_CONTEXT.md for ws1: %v", err)
	}

	wsDir2 := filepath.Join(tmpDir2, ".orch", "workspace")
	if err := os.MkdirAll(wsDir2, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir 2: %v", err)
	}

	ws2Path := filepath.Join(wsDir2, "og-feat-test2-26dec")
	if err := os.MkdirAll(ws2Path, 0755); err != nil {
		t.Fatalf("Failed to create workspace 2: %v", err)
	}
	spawnContext2 := `TASK: Test task 2

You were spawned from beads issue: **proj2-def2**

PROJECT_DIR: /home/user/project2
`
	if err := os.WriteFile(filepath.Join(ws2Path, "SPAWN_CONTEXT.md"), []byte(spawnContext2), 0644); err != nil {
		t.Fatalf("Failed to create SPAWN_CONTEXT.md for ws2: %v", err)
	}

	projectDirs := []string{tmpDir1, tmpDir2}
	cache := buildMultiProjectWorkspaceCache(projectDirs)

	if len(cache.beadsToWorkspace) != 2 {
		t.Errorf("Expected 2 entries in merged beadsToWorkspace, got %d", len(cache.beadsToWorkspace))
	}

	if _, ok := cache.beadsToWorkspace["proj1-abc1"]; !ok {
		t.Error("Expected proj1-abc1 in merged beadsToWorkspace")
	}

	if _, ok := cache.beadsToWorkspace["proj2-def2"]; !ok {
		t.Error("Expected proj2-def2 in merged beadsToWorkspace")
	}

	if cache.beadsToProjectDir["proj1-abc1"] != "/home/user/project1" {
		t.Errorf("Expected projectDir /home/user/project1, got %s", cache.beadsToProjectDir["proj1-abc1"])
	}
	if cache.beadsToProjectDir["proj2-def2"] != "/home/user/project2" {
		t.Errorf("Expected projectDir /home/user/project2, got %s", cache.beadsToProjectDir["proj2-def2"])
	}

	if len(cache.workspaceEntries) != 2 {
		t.Errorf("Expected 2 workspace entries, got %d", len(cache.workspaceEntries))
	}

	if len(cache.workspaceEntryToPath) != 2 {
		t.Errorf("Expected 2 entries in workspaceEntryToPath, got %d", len(cache.workspaceEntryToPath))
	}
}

func TestBuildMultiProjectWorkspaceCacheSingleProject(t *testing.T) {
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, ".orch", "workspace")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	wsPath := filepath.Join(wsDir, "og-feat-test-26dec")
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	spawnContext := `TASK: Test task

You were spawned from beads issue: **proj-abc1**

PROJECT_DIR: /home/user/project
`
	if err := os.WriteFile(filepath.Join(wsPath, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatalf("Failed to create SPAWN_CONTEXT.md: %v", err)
	}

	projectDirs := []string{tmpDir}
	cache := buildMultiProjectWorkspaceCache(projectDirs)

	if len(cache.beadsToWorkspace) != 1 {
		t.Errorf("Expected 1 entry in beadsToWorkspace, got %d", len(cache.beadsToWorkspace))
	}
	if _, ok := cache.beadsToWorkspace["proj-abc1"]; !ok {
		t.Error("Expected proj-abc1 in beadsToWorkspace")
	}
}

func TestBuildMultiProjectWorkspaceCacheEmpty(t *testing.T) {
	cache := buildMultiProjectWorkspaceCache([]string{})

	if cache == nil {
		t.Fatal("Expected non-nil cache")
	}
	if cache.beadsToWorkspace == nil {
		t.Error("Expected initialized beadsToWorkspace map")
	}
	if cache.beadsToProjectDir == nil {
		t.Error("Expected initialized beadsToProjectDir map")
	}
}

func TestWorkspaceCacheLookupMethods(t *testing.T) {
	cache := &workspaceCache{
		beadsToWorkspace:     map[string]string{"test-id": "/path/to/workspace"},
		beadsToProjectDir:    map[string]string{"test-id": "/path/to/project"},
		workspaceDir:         "/default/workspace/dir",
		workspaceEntryToPath: map[string]string{"og-feat-test": "/specific/path/og-feat-test"},
	}

	if ws := cache.lookupWorkspace("test-id"); ws != "/path/to/workspace" {
		t.Errorf("Expected /path/to/workspace, got %s", ws)
	}
	if ws := cache.lookupWorkspace("nonexistent"); ws != "" {
		t.Errorf("Expected empty string for nonexistent beads ID, got %s", ws)
	}

	if pd := cache.lookupProjectDir("test-id"); pd != "/path/to/project" {
		t.Errorf("Expected /path/to/project, got %s", pd)
	}

	if path := cache.lookupWorkspacePathByEntry("og-feat-test"); path != "/specific/path/og-feat-test" {
		t.Errorf("Expected /specific/path/og-feat-test, got %s", path)
	}

	if path := cache.lookupWorkspacePathByEntry("unknown-entry"); path != "/default/workspace/dir/unknown-entry" {
		t.Errorf("Expected /default/workspace/dir/unknown-entry, got %s", path)
	}
}

func TestBeadsCacheInvalidate(t *testing.T) {
	cache := newBeadsCache()

	cache.mu.Lock()
	cache.openIssues["test-id"] = nil
	cache.allIssues["test-id"] = nil
	cache.comments["test-id"] = nil
	cache.openIssuesFetchedAt = time.Now()
	cache.allIssuesFetchedAt = time.Now()
	cache.commentsFetchedAt = time.Now()
	cache.allIssuesFetchedFor = []string{"test-id"}
	cache.commentsFetchedFor = []string{"test-id"}
	cache.mu.Unlock()

	cache.mu.RLock()
	if len(cache.openIssues) != 1 {
		t.Errorf("Expected 1 open issue before invalidate, got %d", len(cache.openIssues))
	}
	cache.mu.RUnlock()

	cache.invalidate()

	cache.mu.RLock()
	defer cache.mu.RUnlock()

	if len(cache.openIssues) != 0 {
		t.Errorf("Expected 0 open issues after invalidate, got %d", len(cache.openIssues))
	}
	if len(cache.allIssues) != 0 {
		t.Errorf("Expected 0 all issues after invalidate, got %d", len(cache.allIssues))
	}
	if len(cache.comments) != 0 {
		t.Errorf("Expected 0 comments after invalidate, got %d", len(cache.comments))
	}
	if !cache.openIssuesFetchedAt.IsZero() {
		t.Errorf("Expected zero openIssuesFetchedAt after invalidate")
	}
	if !cache.allIssuesFetchedAt.IsZero() {
		t.Errorf("Expected zero allIssuesFetchedAt after invalidate")
	}
	if !cache.commentsFetchedAt.IsZero() {
		t.Errorf("Expected zero commentsFetchedAt after invalidate")
	}
	if cache.allIssuesFetchedFor != nil {
		t.Errorf("Expected nil allIssuesFetchedFor after invalidate")
	}
	if cache.commentsFetchedFor != nil {
		t.Errorf("Expected nil commentsFetchedFor after invalidate")
	}
}

func TestGlobalWorkspaceCacheInvalidate(t *testing.T) {
	globalWorkspaceCacheInstance.mu.Lock()
	globalWorkspaceCacheInstance.cache = &workspaceCache{
		beadsToWorkspace: map[string]string{"test-id": "/path/to/workspace"},
	}
	globalWorkspaceCacheInstance.fetchedAt = time.Now()
	globalWorkspaceCacheInstance.mu.Unlock()

	globalWorkspaceCacheInstance.mu.RLock()
	if globalWorkspaceCacheInstance.cache == nil {
		t.Errorf("Expected cache to be set before invalidate")
	}
	globalWorkspaceCacheInstance.mu.RUnlock()

	globalWorkspaceCacheInstance.invalidate()

	globalWorkspaceCacheInstance.mu.RLock()
	defer globalWorkspaceCacheInstance.mu.RUnlock()

	if globalWorkspaceCacheInstance.cache != nil {
		t.Errorf("Expected cache to be nil after invalidate")
	}
	if !globalWorkspaceCacheInstance.fetchedAt.IsZero() {
		t.Errorf("Expected zero fetchedAt after invalidate")
	}
}

func TestHandleCacheInvalidate(t *testing.T) {
	if globalBeadsCache == nil {
		globalBeadsCache = newBeadsCache()
	}

	globalBeadsCache.mu.Lock()
	globalBeadsCache.openIssues["test"] = nil
	globalBeadsCache.openIssuesFetchedAt = time.Now()
	globalBeadsCache.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/api/cache/invalidate", nil)
	w := httptest.NewRecorder()

	handleCacheInvalidate(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", result["status"])
	}

	globalBeadsCache.mu.RLock()
	if len(globalBeadsCache.openIssues) != 0 {
		t.Errorf("Expected cache to be empty after invalidate")
	}
	globalBeadsCache.mu.RUnlock()
}

func TestBuildWorkspaceCacheSkipsArchived(t *testing.T) {
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, ".orch", "workspace")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	// Create an active workspace
	activeWs := filepath.Join(wsDir, "og-feat-active-19feb")
	if err := os.MkdirAll(activeWs, 0755); err != nil {
		t.Fatalf("Failed to create active workspace: %v", err)
	}
	spawnContext := `You were spawned from beads issue: **proj-abc1**
PROJECT_DIR: ` + tmpDir
	if err := os.WriteFile(filepath.Join(activeWs, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatalf("Failed to create SPAWN_CONTEXT.md: %v", err)
	}

	// Create archived directory with a workspace inside
	archivedDir := filepath.Join(wsDir, "archived")
	archivedWs := filepath.Join(archivedDir, "og-feat-old-18feb")
	if err := os.MkdirAll(archivedWs, 0755); err != nil {
		t.Fatalf("Failed to create archived workspace: %v", err)
	}
	archivedContext := `You were spawned from beads issue: **proj-def2**
PROJECT_DIR: ` + tmpDir
	if err := os.WriteFile(filepath.Join(archivedWs, "SPAWN_CONTEXT.md"), []byte(archivedContext), 0644); err != nil {
		t.Fatalf("Failed to create archived SPAWN_CONTEXT.md: %v", err)
	}

	cache := buildWorkspaceCache(tmpDir)

	// Active workspace should be found
	if _, ok := cache.beadsToWorkspace["proj-abc1"]; !ok {
		t.Error("Expected active workspace proj-abc1 in cache")
	}

	// Archived workspace should NOT be found
	if _, ok := cache.beadsToWorkspace["proj-def2"]; ok {
		t.Error("Archived workspace proj-def2 should not be in cache")
	}

	// workspaceEntries should not include archived directory
	for _, entry := range cache.workspaceEntries {
		if entry.Name() == "archived" {
			t.Error("workspaceEntries should not contain 'archived' directory")
		}
	}

	if len(cache.workspaceEntries) != 1 {
		t.Errorf("Expected 1 workspace entry, got %d", len(cache.workspaceEntries))
	}
}

func TestHandleCacheInvalidateMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/cache/invalidate", nil)
	w := httptest.NewRecorder()

	handleCacheInvalidate(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}
