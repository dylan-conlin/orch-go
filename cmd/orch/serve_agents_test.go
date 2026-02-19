package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

func findAgentByBeadsID(agents []AgentAPIResponse, beadsID string) (AgentAPIResponse, bool) {
	for _, agent := range agents {
		if agent.BeadsID == beadsID {
			return agent, true
		}
	}
	return AgentAPIResponse{}, false
}

func newTestOpenCodeServer(t *testing.T, sessions map[string]opencode.Session, messages map[string][]opencode.Message) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/session/") {
			http.NotFound(w, r)
			return
		}

		relative := strings.TrimPrefix(r.URL.Path, "/session/")
		parts := strings.Split(strings.Trim(relative, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			http.NotFound(w, r)
			return
		}

		sessionID := parts[0]
		if len(parts) == 1 {
			session, ok := sessions[sessionID]
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(session)
			return
		}

		if len(parts) == 2 && parts[1] == "message" {
			payload, ok := messages[sessionID]
			if !ok {
				payload = []opencode.Message{}
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(payload)
			return
		}

		http.NotFound(w, r)
	}))
}

func TestHandleAgents(t *testing.T) {
	// Initialize the global caches that handleAgents depends on
	if globalBeadsCache == nil {
		globalBeadsCache = newBeadsCache()
	}

	// Create a test request
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	w := httptest.NewRecorder()

	// Call the handler
	handleAgents(w, req)

	// Check the response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Verify it returns valid JSON (even if empty array)
	var agents []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}
}

func TestHandleAgentsMethodNotAllowed(t *testing.T) {
	// Test POST method is not allowed
	req := httptest.NewRequest(http.MethodPost, "/api/agents", nil)
	w := httptest.NewRecorder()

	handleAgents(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleEventsMethodNotAllowed(t *testing.T) {
	// Test POST method is not allowed
	req := httptest.NewRequest(http.MethodPost, "/api/events", nil)
	w := httptest.NewRecorder()

	handleEvents(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestAgentAPIResponseJSONFormat(t *testing.T) {
	// Test that AgentAPIResponse serializes correctly to JSON
	synthesis := &SynthesisResponse{
		TLDR:           "Test synthesis summary",
		Outcome:        "success",
		Recommendation: "close",
		DeltaSummary:   "2 files created, 1 modified",
		NextActions:    []string{"- Review changes", "- Update docs"},
	}

	agent := &AgentAPIResponse{
		Synthesis: synthesis,
	}

	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("Failed to marshal AgentAPIResponse: %v", err)
	}

	// Verify the JSON contains expected fields
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	synthData, ok := result["synthesis"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected synthesis field in JSON")
	}

	if synthData["tldr"] != "Test synthesis summary" {
		t.Errorf("Expected tldr 'Test synthesis summary', got %v", synthData["tldr"])
	}
	if synthData["outcome"] != "success" {
		t.Errorf("Expected outcome 'success', got %v", synthData["outcome"])
	}
	if synthData["recommendation"] != "close" {
		t.Errorf("Expected recommendation 'close', got %v", synthData["recommendation"])
	}
}

func TestHandleAgentlogMethodNotAllowed(t *testing.T) {
	// Test POST method is not allowed
	req := httptest.NewRequest(http.MethodPost, "/api/agentlog", nil)
	w := httptest.NewRecorder()

	handleAgentlog(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleAgentlogEmptyFile(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()
	tmpLogPath := filepath.Join(tmpDir, "events.jsonl")

	// Test with non-existent file - should return empty array
	eventList, err := readLastNEvents(tmpLogPath, 100)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Create empty file
	if err := os.WriteFile(tmpLogPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	eventList, err = readLastNEvents(tmpLogPath, 100)
	if err != nil {
		t.Errorf("Expected no error for empty file, got: %v", err)
	}
	if len(eventList) != 0 {
		t.Errorf("Expected empty event list, got %d events", len(eventList))
	}
}

func TestReadLastNEvents(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()
	tmpLogPath := filepath.Join(tmpDir, "events.jsonl")

	// Create test events
	testEvents := []events.Event{
		{Type: "session.spawned", SessionID: "sess1", Timestamp: time.Now().Unix()},
		{Type: "session.status", SessionID: "sess1", Timestamp: time.Now().Unix()},
		{Type: "session.completed", SessionID: "sess1", Timestamp: time.Now().Unix()},
	}

	// Write events to file
	file, err := os.Create(tmpLogPath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	for _, event := range testEvents {
		data, _ := json.Marshal(event)
		file.Write(append(data, '\n'))
	}
	file.Close()

	// Test reading all events
	eventList, err := readLastNEvents(tmpLogPath, 100)
	if err != nil {
		t.Fatalf("Failed to read events: %v", err)
	}
	if len(eventList) != 3 {
		t.Errorf("Expected 3 events, got %d", len(eventList))
	}

	// Test reading last 2 events
	eventList, err = readLastNEvents(tmpLogPath, 2)
	if err != nil {
		t.Fatalf("Failed to read events: %v", err)
	}
	if len(eventList) != 2 {
		t.Errorf("Expected 2 events, got %d", len(eventList))
	}
	if eventList[0].Type != "session.status" {
		t.Errorf("Expected first event to be session.status, got %s", eventList[0].Type)
	}
	if eventList[1].Type != "session.completed" {
		t.Errorf("Expected second event to be session.completed, got %s", eventList[1].Type)
	}
}

func TestHandleAgentlogJSONResponse(t *testing.T) {
	// Note: This test uses the default log path which may or may not exist
	// In production, we'd want to inject the path, but for now we just verify
	// the endpoint returns valid JSON
	req := httptest.NewRequest(http.MethodGet, "/api/agentlog", nil)
	w := httptest.NewRecorder()

	handleAgentlog(w, req)

	resp := w.Result()
	// Should be 200 even if file doesn't exist (returns empty array)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Verify it returns valid JSON array
	var eventList []events.Event
	if err := json.NewDecoder(resp.Body).Decode(&eventList); err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}
}

func TestCheckWorkspaceSynthesisForCompletion(t *testing.T) {
	// Create a temporary project directory with workspace
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")

	// Test 1: Workspace with SYNTHESIS.md should indicate completion
	t.Run("workspace with SYNTHESIS.md", func(t *testing.T) {
		workspaceName := "og-feat-test-25dec"
		workspacePath := filepath.Join(workspaceDir, workspaceName)
		if err := os.MkdirAll(workspacePath, 0755); err != nil {
			t.Fatalf("Failed to create workspace dir: %v", err)
		}

		// Create SYNTHESIS.md
		synthesisContent := `# Session Synthesis
TLDR: Test completed successfully
`
		if err := os.WriteFile(filepath.Join(workspacePath, "SYNTHESIS.md"), []byte(synthesisContent), 0644); err != nil {
			t.Fatalf("Failed to create SYNTHESIS.md: %v", err)
		}

		// Check if synthesis exists
		synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
		if _, err := os.Stat(synthesisPath); err != nil {
			t.Errorf("Expected SYNTHESIS.md to exist, got error: %v", err)
		}
	})

	// Test 2: Workspace without SYNTHESIS.md should not indicate completion
	t.Run("workspace without SYNTHESIS.md", func(t *testing.T) {
		workspaceName := "og-feat-no-synthesis-25dec"
		workspacePath := filepath.Join(workspaceDir, workspaceName)
		if err := os.MkdirAll(workspacePath, 0755); err != nil {
			t.Fatalf("Failed to create workspace dir: %v", err)
		}

		// Create only SPAWN_CONTEXT.md (no SYNTHESIS.md)
		spawnContextContent := `TASK: Test task
`
		if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContextContent), 0644); err != nil {
			t.Fatalf("Failed to create SPAWN_CONTEXT.md: %v", err)
		}

		// Check that synthesis does NOT exist
		synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
		if _, err := os.Stat(synthesisPath); err == nil {
			t.Errorf("Expected SYNTHESIS.md to NOT exist")
		}
	})
}

func TestCheckWorkspaceSynthesis(t *testing.T) {
	// Create a temporary workspace
	tmpDir := t.TempDir()

	// Test case 1: No SYNTHESIS.md
	exists := checkWorkspaceSynthesis(tmpDir)
	if exists {
		t.Error("Expected checkWorkspaceSynthesis to return false for empty workspace")
	}

	// Test case 2: With SYNTHESIS.md
	synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")
	if err := os.WriteFile(synthesisPath, []byte("# Synthesis\nTLDR: Test\n"), 0644); err != nil {
		t.Fatalf("Failed to write SYNTHESIS.md: %v", err)
	}

	exists = checkWorkspaceSynthesis(tmpDir)
	if !exists {
		t.Error("Expected checkWorkspaceSynthesis to return true when SYNTHESIS.md exists")
	}

	// Test case 3: With empty SYNTHESIS.md
	if err := os.WriteFile(synthesisPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write empty SYNTHESIS.md: %v", err)
	}

	exists = checkWorkspaceSynthesis(tmpDir)
	if exists {
		t.Error("Expected checkWorkspaceSynthesis to return false for empty SYNTHESIS.md")
	}
}

func TestExtractUniqueProjectDirs(t *testing.T) {
	// Note: extractUniqueProjectDirs now includes kb projects, so tests verify
	// relative behavior rather than exact counts (kb project count varies by env)

	// Get kb project count to adjust expectations
	kbProjectCount := len(getKBProjects())

	tests := []struct {
		name              string
		currentProjectDir string
		sessionDirs       []string
		minExpectedCount  int // minimum expected from non-kb sources
	}{
		{
			name:              "empty sessions with current dir",
			currentProjectDir: "/home/user/project1",
			sessionDirs:       []string{},
			minExpectedCount:  1, // just current dir
		},
		{
			name:              "single session same as current",
			currentProjectDir: "/home/user/project1",
			sessionDirs:       []string{"/home/user/project1"},
			minExpectedCount:  1, // deduplicated
		},
		{
			name:              "multiple sessions different dirs",
			currentProjectDir: "/home/user/project1",
			sessionDirs:       []string{"/home/user/project2", "/home/user/project3"},
			minExpectedCount:  3, // current + 2 others
		},
		{
			name:              "duplicate session directories",
			currentProjectDir: "/home/user/project1",
			sessionDirs:       []string{"/home/user/project2", "/home/user/project2", "/home/user/project3"},
			minExpectedCount:  3, // deduped
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
			minExpectedCount:  2, // empty is skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock sessions - we need to use the actual type
			sessions := make([]opencode.Session, len(tt.sessionDirs))
			for i, dir := range tt.sessionDirs {
				sessions[i] = opencode.Session{Directory: dir}
			}

			result := extractUniqueProjectDirs(sessions, tt.currentProjectDir)

			// Verify at least the expected non-kb sources are present
			// (kb projects are deduplicated so count may be less than min + kbProjectCount)
			if len(result) < tt.minExpectedCount {
				t.Errorf("Expected at least %d project dirs, got %d: %v", tt.minExpectedCount, len(result), result)
			}

			// Verify current project dir is always first if provided
			if tt.currentProjectDir != "" && len(result) > 0 {
				if result[0] != filepath.Clean(tt.currentProjectDir) {
					t.Errorf("Expected current project dir %q to be first, got %q", tt.currentProjectDir, result[0])
				}
			}

			// Verify kb projects are included (if available)
			if kbProjectCount > 0 {
				// At least some kb projects should be in the result
				// (exact count depends on deduplication with session dirs)
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
	// Create a temporary project directory structure
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, ".orch", "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	// Create test workspaces with SPAWN_CONTEXT.md
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

	// Build the cache
	cache := buildWorkspaceCache(tmpDir)

	// Verify cache contents
	if len(cache.beadsToWorkspace) != 2 {
		t.Errorf("Expected 2 entries in beadsToWorkspace, got %d", len(cache.beadsToWorkspace))
	}

	// Check first workspace
	if wsPath, ok := cache.beadsToWorkspace["orch-go-abc1"]; !ok {
		t.Error("Expected orch-go-abc1 in beadsToWorkspace")
	} else if !filepath.IsAbs(wsPath) {
		t.Errorf("Expected absolute path for workspace, got %s", wsPath)
	}

	if projDir := cache.beadsToProjectDir["orch-go-abc1"]; projDir != "/home/user/orch-go" {
		t.Errorf("Expected projectDir /home/user/orch-go, got %s", projDir)
	}

	// Check second workspace
	if projDir := cache.beadsToProjectDir["kb-cli-def2"]; projDir != "/home/user/kb-cli" {
		t.Errorf("Expected projectDir /home/user/kb-cli, got %s", projDir)
	}

	// Verify workspaceEntryToPath
	if len(cache.workspaceEntryToPath) != 2 {
		t.Errorf("Expected 2 entries in workspaceEntryToPath, got %d", len(cache.workspaceEntryToPath))
	}
}

func TestBuildMultiProjectWorkspaceCache(t *testing.T) {
	// Create two temporary project directories
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Create workspace structure for project 1
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

	// Create workspace structure for project 2
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

	// Build multi-project cache
	projectDirs := []string{tmpDir1, tmpDir2}
	cache := buildMultiProjectWorkspaceCache(projectDirs)

	// Verify merged cache contents
	if len(cache.beadsToWorkspace) != 2 {
		t.Errorf("Expected 2 entries in merged beadsToWorkspace, got %d", len(cache.beadsToWorkspace))
	}

	// Check workspace from project 1
	if _, ok := cache.beadsToWorkspace["proj1-abc1"]; !ok {
		t.Error("Expected proj1-abc1 in merged beadsToWorkspace")
	}

	// Check workspace from project 2
	if _, ok := cache.beadsToWorkspace["proj2-def2"]; !ok {
		t.Error("Expected proj2-def2 in merged beadsToWorkspace")
	}

	// Verify both project dirs are in the cache
	if cache.beadsToProjectDir["proj1-abc1"] != "/home/user/project1" {
		t.Errorf("Expected projectDir /home/user/project1, got %s", cache.beadsToProjectDir["proj1-abc1"])
	}
	if cache.beadsToProjectDir["proj2-def2"] != "/home/user/project2" {
		t.Errorf("Expected projectDir /home/user/project2, got %s", cache.beadsToProjectDir["proj2-def2"])
	}

	// Verify workspace entries are merged
	if len(cache.workspaceEntries) != 2 {
		t.Errorf("Expected 2 workspace entries, got %d", len(cache.workspaceEntries))
	}

	// Verify workspaceEntryToPath is merged
	if len(cache.workspaceEntryToPath) != 2 {
		t.Errorf("Expected 2 entries in workspaceEntryToPath, got %d", len(cache.workspaceEntryToPath))
	}
}

func TestBuildMultiProjectWorkspaceCacheSingleProject(t *testing.T) {
	// Create a temporary project directory
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

	// Build multi-project cache with single project (should use optimized path)
	projectDirs := []string{tmpDir}
	cache := buildMultiProjectWorkspaceCache(projectDirs)

	// Should still work correctly
	if len(cache.beadsToWorkspace) != 1 {
		t.Errorf("Expected 1 entry in beadsToWorkspace, got %d", len(cache.beadsToWorkspace))
	}
	if _, ok := cache.beadsToWorkspace["proj-abc1"]; !ok {
		t.Error("Expected proj-abc1 in beadsToWorkspace")
	}
}

func TestBuildMultiProjectWorkspaceCacheEmpty(t *testing.T) {
	// Build cache with empty project dirs
	cache := buildMultiProjectWorkspaceCache([]string{})

	// Should return empty cache, not nil
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

	// Test lookupWorkspace
	if ws := cache.lookupWorkspace("test-id"); ws != "/path/to/workspace" {
		t.Errorf("Expected /path/to/workspace, got %s", ws)
	}
	if ws := cache.lookupWorkspace("nonexistent"); ws != "" {
		t.Errorf("Expected empty string for nonexistent beads ID, got %s", ws)
	}

	// Test lookupProjectDir
	if pd := cache.lookupProjectDir("test-id"); pd != "/path/to/project" {
		t.Errorf("Expected /path/to/project, got %s", pd)
	}

	// Test lookupWorkspacePathByEntry - should use map first
	if path := cache.lookupWorkspacePathByEntry("og-feat-test"); path != "/specific/path/og-feat-test" {
		t.Errorf("Expected /specific/path/og-feat-test, got %s", path)
	}

	// Test lookupWorkspacePathByEntry - should fallback to workspaceDir
	if path := cache.lookupWorkspacePathByEntry("unknown-entry"); path != "/default/workspace/dir/unknown-entry" {
		t.Errorf("Expected /default/workspace/dir/unknown-entry, got %s", path)
	}
}

func TestBeadsCacheInvalidate(t *testing.T) {
	// Create a cache with some data
	cache := newBeadsCache()

	// Populate the cache with test data
	cache.mu.Lock()
	cache.openIssues["test-id"] = nil // Just need a key to verify invalidation
	cache.allIssues["test-id"] = nil
	cache.comments["test-id"] = nil
	cache.openIssuesFetchedAt = time.Now()
	cache.allIssuesFetchedAt = time.Now()
	cache.commentsFetchedAt = time.Now()
	cache.allIssuesFetchedFor = []string{"test-id"}
	cache.commentsFetchedFor = []string{"test-id"}
	cache.mu.Unlock()

	// Verify cache has data
	cache.mu.RLock()
	if len(cache.openIssues) != 1 {
		t.Errorf("Expected 1 open issue before invalidate, got %d", len(cache.openIssues))
	}
	cache.mu.RUnlock()

	// Invalidate the cache
	cache.invalidate()

	// Verify cache is cleared
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
	// Setup the global cache with some data
	globalWorkspaceCacheInstance.mu.Lock()
	globalWorkspaceCacheInstance.cache = &workspaceCache{
		beadsToWorkspace: map[string]string{"test-id": "/path/to/workspace"},
	}
	globalWorkspaceCacheInstance.fetchedAt = time.Now()
	globalWorkspaceCacheInstance.mu.Unlock()

	// Verify cache has data
	globalWorkspaceCacheInstance.mu.RLock()
	if globalWorkspaceCacheInstance.cache == nil {
		t.Errorf("Expected cache to be set before invalidate")
	}
	globalWorkspaceCacheInstance.mu.RUnlock()

	// Invalidate the cache
	globalWorkspaceCacheInstance.invalidate()

	// Verify cache is cleared
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
	// Initialize the global cache
	if globalBeadsCache == nil {
		globalBeadsCache = newBeadsCache()
	}

	// Populate with some test data
	globalBeadsCache.mu.Lock()
	globalBeadsCache.openIssues["test"] = nil
	globalBeadsCache.openIssuesFetchedAt = time.Now()
	globalBeadsCache.mu.Unlock()

	// Test that POST works
	req := httptest.NewRequest(http.MethodPost, "/api/cache/invalidate", nil)
	w := httptest.NewRecorder()

	handleCacheInvalidate(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify JSON response
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", result["status"])
	}

	// Verify cache was invalidated
	globalBeadsCache.mu.RLock()
	if len(globalBeadsCache.openIssues) != 0 {
		t.Errorf("Expected cache to be empty after invalidate")
	}
	globalBeadsCache.mu.RUnlock()
}

func TestHandleCacheInvalidateMethodNotAllowed(t *testing.T) {
	// Test GET method is not allowed
	req := httptest.NewRequest(http.MethodGet, "/api/cache/invalidate", nil)
	w := httptest.NewRecorder()

	handleCacheInvalidate(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

// TestDetermineAgentStatus tests the Priority Cascade model for agent status determination.
// Priority order:
//  1. Beads issue closed → "completed"
//  2. Phase: Complete reported AND dead → "awaiting-cleanup"
//  3. Phase: Complete reported → "completed"
//  4. SYNTHESIS.md exists AND dead → "awaiting-cleanup"
//  5. SYNTHESIS.md exists → "completed"
//  6. Session activity → "active", "idle", or "dead"
func TestDetermineAgentStatus(t *testing.T) {
	// Create a temporary workspace with SYNTHESIS.md for testing
	tmpDir := t.TempDir()
	synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")

	tests := []struct {
		name           string
		issueClosed    bool
		phaseComplete  bool
		hasSynthesis   bool
		sessionStatus  string // "active", "idle", or "dead" based on activity
		expectedStatus string
	}{
		// Priority 1: Beads closed overrides everything
		{
			name:           "beads_closed_overrides_all",
			issueClosed:    true,
			phaseComplete:  false,
			hasSynthesis:   false,
			sessionStatus:  "active",
			expectedStatus: "completed",
		},
		{
			name:           "beads_closed_even_if_idle",
			issueClosed:    true,
			phaseComplete:  false,
			hasSynthesis:   false,
			sessionStatus:  "idle",
			expectedStatus: "completed",
		},
		{
			name:           "beads_closed_even_if_dead",
			issueClosed:    true,
			phaseComplete:  true,
			hasSynthesis:   true,
			sessionStatus:  "dead",
			expectedStatus: "completed",
		},
		// Priority 2: Phase: Complete + dead → awaiting-cleanup
		{
			name:           "phase_complete_dead_awaiting_cleanup",
			issueClosed:    false,
			phaseComplete:  true,
			hasSynthesis:   false,
			sessionStatus:  "dead",
			expectedStatus: "awaiting-cleanup",
		},
		// Priority 3: Phase: Complete + active/idle → completed
		{
			name:           "phase_complete_overrides_session",
			issueClosed:    false,
			phaseComplete:  true,
			hasSynthesis:   false,
			sessionStatus:  "active",
			expectedStatus: "completed",
		},
		{
			name:           "phase_complete_overrides_idle",
			issueClosed:    false,
			phaseComplete:  true,
			hasSynthesis:   false,
			sessionStatus:  "idle",
			expectedStatus: "completed",
		},
		// Priority 4: SYNTHESIS.md + dead → awaiting-cleanup
		{
			name:           "synthesis_dead_awaiting_cleanup",
			issueClosed:    false,
			phaseComplete:  false,
			hasSynthesis:   true,
			sessionStatus:  "dead",
			expectedStatus: "awaiting-cleanup",
		},
		// Priority 5: SYNTHESIS.md + active/idle → completed
		{
			name:           "synthesis_overrides_session",
			issueClosed:    false,
			phaseComplete:  false,
			hasSynthesis:   true,
			sessionStatus:  "active",
			expectedStatus: "completed",
		},
		{
			name:           "synthesis_overrides_idle",
			issueClosed:    false,
			phaseComplete:  false,
			hasSynthesis:   true,
			sessionStatus:  "idle",
			expectedStatus: "completed",
		},
		// Priority 6: Session activity is the fallback
		{
			name:           "active_session",
			issueClosed:    false,
			phaseComplete:  false,
			hasSynthesis:   false,
			sessionStatus:  "active",
			expectedStatus: "active",
		},
		{
			name:           "idle_session",
			issueClosed:    false,
			phaseComplete:  false,
			hasSynthesis:   false,
			sessionStatus:  "idle",
			expectedStatus: "idle",
		},
		{
			name:           "dead_session_no_completion",
			issueClosed:    false,
			phaseComplete:  false,
			hasSynthesis:   false,
			sessionStatus:  "dead",
			expectedStatus: "dead",
		},
		// Combined scenarios - higher priority wins
		{
			name:           "beads_closed_with_phase_complete",
			issueClosed:    true,
			phaseComplete:  true,
			hasSynthesis:   true,
			sessionStatus:  "idle",
			expectedStatus: "completed",
		},
		{
			name:           "phase_complete_with_synthesis",
			issueClosed:    false,
			phaseComplete:  true,
			hasSynthesis:   true,
			sessionStatus:  "active",
			expectedStatus: "completed",
		},
		{
			name:           "phase_complete_with_synthesis_dead",
			issueClosed:    false,
			phaseComplete:  true,
			hasSynthesis:   true,
			sessionStatus:  "dead",
			expectedStatus: "awaiting-cleanup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up or remove SYNTHESIS.md based on test case
			if tt.hasSynthesis {
				if err := os.WriteFile(synthesisPath, []byte("# Synthesis\nTLDR: Test"), 0644); err != nil {
					t.Fatalf("Failed to write SYNTHESIS.md: %v", err)
				}
			} else {
				os.Remove(synthesisPath)
			}

			result := determineAgentStatus(tt.issueClosed, tt.phaseComplete, tmpDir, tt.sessionStatus)

			if result != tt.expectedStatus {
				t.Errorf("determineAgentStatus() = %q, want %q", result, tt.expectedStatus)
			}
		})
	}
}

// TestDetermineAgentStatusEmptyWorkspace tests that empty workspace path is handled correctly.
func TestDetermineAgentStatusEmptyWorkspace(t *testing.T) {
	// With empty workspace, SYNTHESIS.md check should be skipped
	result := determineAgentStatus(false, false, "", "idle")
	if result != "idle" {
		t.Errorf("Expected 'idle' for empty workspace, got %q", result)
	}
}

// TestDetermineAgentStatusNonExistentWorkspace tests non-existent workspace path.
func TestDetermineAgentStatusNonExistentWorkspace(t *testing.T) {
	// With non-existent workspace, SYNTHESIS.md check should return false
	result := determineAgentStatus(false, false, "/nonexistent/path/workspace", "active")
	if result != "active" {
		t.Errorf("Expected 'active' for non-existent workspace, got %q", result)
	}
}

// TestExtractWorkspaceKeywords tests keyword extraction from workspace names.
func TestExtractWorkspaceKeywords(t *testing.T) {
	tests := []struct {
		name          string
		workspaceName string
		wantKeywords  []string
	}{
		{
			name:          "standard_investigation_workspace",
			workspaceName: "og-inv-skillc-deploy-06jan-ed96",
			wantKeywords:  []string{"skillc", "deploy"},
		},
		{
			name:          "feature_workspace",
			workspaceName: "og-feat-dashboard-auto-discover-06jan-dfc6",
			wantKeywords:  []string{"dashboard", "auto", "discover"},
		},
		{
			name:          "debug_workspace",
			workspaceName: "og-debug-status-polling-05dec-ab12",
			wantKeywords:  []string{"status", "polling"},
		},
		{
			name:          "short_workspace_name",
			workspaceName: "og-inv",
			wantKeywords:  nil,
		},
		{
			name:          "empty_workspace_name",
			workspaceName: "",
			wantKeywords:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractWorkspaceKeywords(tt.workspaceName)
			if len(got) != len(tt.wantKeywords) {
				t.Errorf("extractWorkspaceKeywords(%q) = %v, want %v", tt.workspaceName, got, tt.wantKeywords)
				return
			}
			for i := range got {
				if got[i] != tt.wantKeywords[i] {
					t.Errorf("extractWorkspaceKeywords(%q)[%d] = %q, want %q", tt.workspaceName, i, got[i], tt.wantKeywords[i])
				}
			}
		})
	}
}

// TestIsHexLike tests the hex-like string detection.
func TestIsHexLike(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"abcd", true},
		{"1234", true},
		{"a1b2", true},
		{"ed96", true},
		{"dfc6", true},
		{"ABCD", false}, // uppercase not allowed
		{"ghij", false}, // g-z not allowed
		{"test", false}, // contains non-hex chars
		{"", true},      // empty is trivially hex-like
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isHexLike(tt.input); got != tt.expected {
				t.Errorf("isHexLike(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

// TestDiscoverInvestigationPath tests auto-discovery of investigation files.
func TestDiscoverInvestigationPath(t *testing.T) {
	// Create a temporary project directory structure
	tmpDir := t.TempDir()

	// Create .kb/investigations/ directory with some investigation files
	invDir := filepath.Join(tmpDir, ".kb", "investigations")
	if err := os.MkdirAll(invDir, 0755); err != nil {
		t.Fatalf("Failed to create investigations dir: %v", err)
	}

	// Create some investigation files
	invFiles := []string{
		"2026-01-06-inv-dashboard-auto-discover.md",
		"2026-01-05-inv-status-polling.md",
		"2026-01-04-inv-skillc-deploy-structure.md",
	}
	for _, name := range invFiles {
		if err := os.WriteFile(filepath.Join(invDir, name), []byte("# Investigation"), 0644); err != nil {
			t.Fatalf("Failed to create investigation file: %v", err)
		}
	}

	// Create .kb/investigations/simple/ directory
	simpleDir := filepath.Join(invDir, "simple")
	if err := os.MkdirAll(simpleDir, 0755); err != nil {
		t.Fatalf("Failed to create simple dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(simpleDir, "2026-01-06-simple-test.md"), []byte("# Simple"), 0644); err != nil {
		t.Fatalf("Failed to create simple investigation: %v", err)
	}

	// Create workspace directory with .md files
	wsDir := filepath.Join(tmpDir, ".orch", "workspace", "og-inv-my-workspace-06jan-1234")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}
	// Create standard workspace files that should be skipped
	if err := os.WriteFile(filepath.Join(wsDir, "SPAWN_CONTEXT.md"), []byte("# Context"), 0644); err != nil {
		t.Fatalf("Failed to create SPAWN_CONTEXT.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsDir, "SYNTHESIS.md"), []byte("# Synthesis"), 0644); err != nil {
		t.Fatalf("Failed to create SYNTHESIS.md: %v", err)
	}
	// Create an investigation file in workspace
	if err := os.WriteFile(filepath.Join(wsDir, "inv-local-findings.md"), []byte("# Findings"), 0644); err != nil {
		t.Fatalf("Failed to create local investigation: %v", err)
	}

	tests := []struct {
		name          string
		workspaceName string
		beadsID       string
		projectDir    string
		wantFound     bool
		wantContains  string // substring that should be in the result
	}{
		{
			name:          "match_by_workspace_keywords",
			workspaceName: "og-feat-dashboard-auto-discover-06jan-dfc6",
			beadsID:       "orch-go-wrrks",
			projectDir:    tmpDir,
			wantFound:     true,
			wantContains:  "dashboard-auto-discover",
		},
		{
			name:          "match_by_workspace_keywords_skillc",
			workspaceName: "og-inv-skillc-deploy-structure-06jan-ed96",
			beadsID:       "orch-go-xyz",
			projectDir:    tmpDir,
			wantFound:     true,
			wantContains:  "skillc-deploy-structure",
		},
		{
			name:          "no_project_dir",
			workspaceName: "og-inv-test",
			beadsID:       "test-123",
			projectDir:    "",
			wantFound:     false,
			wantContains:  "",
		},
		{
			name:          "no_matching_investigation",
			workspaceName: "og-inv-nonexistent-topic-06jan-1234",
			beadsID:       "orch-go-nomatch",
			projectDir:    tmpDir,
			wantFound:     false,
			wantContains:  "",
		},
		{
			name:          "workspace_with_local_inv_file",
			workspaceName: "og-inv-my-workspace-06jan-1234",
			beadsID:       "orch-go-local",
			projectDir:    tmpDir,
			wantFound:     true,
			wantContains:  "inv-local-findings.md",
		},
	}

	// Build cache for the test project directory
	cache := buildInvestigationDirCache([]string{tmpDir})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := discoverInvestigationPath(tt.workspaceName, tt.beadsID, tt.projectDir, cache)
			if tt.wantFound && got == "" {
				t.Errorf("discoverInvestigationPath() = empty, want path containing %q", tt.wantContains)
			}
			if !tt.wantFound && got != "" {
				t.Errorf("discoverInvestigationPath() = %q, want empty", got)
			}
			if tt.wantFound && tt.wantContains != "" && !filepath.IsAbs(got) {
				t.Errorf("discoverInvestigationPath() = %q, want absolute path", got)
			}
		})
	}
}

func TestListInProgressIssuesSingleProject(t *testing.T) {
	oldListOpenIssues := listOpenIssues
	oldListOpenIssuesWithDir := listOpenIssuesWithDir
	defer func() {
		listOpenIssues = oldListOpenIssues
		listOpenIssuesWithDir = oldListOpenIssuesWithDir
	}()

	listOpenIssues = func() (map[string]*verify.Issue, error) {
		return nil, fmt.Errorf("unexpected call")
	}
	listOpenIssuesWithDir = func(projectDir string) (map[string]*verify.Issue, error) {
		return map[string]*verify.Issue{
			"orch-go-open":   {ID: "orch-go-open", Status: "open"},
			"orch-go-active": {ID: "orch-go-active", Status: "in_progress"},
			"orch-go-closed": {ID: "orch-go-closed", Status: "closed"},
		}, nil
	}

	issues, projectDirs := listInProgressIssues([]string{"/tmp/project"})
	if len(projectDirs) != 1 {
		t.Fatalf("Expected 1 project dir, got %d", len(projectDirs))
	}
	if _, ok := issues["orch-go-active"]; !ok {
		t.Fatal("Expected in_progress issue to be included")
	}
	if _, ok := issues["orch-go-open"]; ok {
		t.Fatal("Expected open issue to be excluded")
	}
	if _, ok := issues["orch-go-closed"]; ok {
		t.Fatal("Expected closed issue to be excluded")
	}
}

func TestListInProgressIssuesCrossProjectDedup(t *testing.T) {
	oldListOpenIssues := listOpenIssues
	oldListOpenIssuesWithDir := listOpenIssuesWithDir
	defer func() {
		listOpenIssues = oldListOpenIssues
		listOpenIssuesWithDir = oldListOpenIssuesWithDir
	}()

	listOpenIssues = func() (map[string]*verify.Issue, error) {
		return nil, fmt.Errorf("unexpected call")
	}
	listOpenIssuesWithDir = func(projectDir string) (map[string]*verify.Issue, error) {
		switch projectDir {
		case "/tmp/project-a":
			return map[string]*verify.Issue{
				"orch-go-a1":     {ID: "orch-go-a1", Status: "in_progress"},
				"orch-go-shared": {ID: "orch-go-shared", Status: "in_progress"},
			}, nil
		case "/tmp/project-b":
			return map[string]*verify.Issue{
				"orch-go-shared": {ID: "orch-go-shared", Status: "in_progress"},
				"orch-go-b1":     {ID: "orch-go-b1", Status: "in_progress"},
			}, nil
		default:
			return nil, nil
		}
	}

	issues, projectDirs := listInProgressIssues([]string{"/tmp/project-a", "/tmp/project-b"})
	if len(issues) != 3 {
		t.Fatalf("Expected 3 deduplicated issues, got %d", len(issues))
	}
	if projectDirs["orch-go-shared"] != "/tmp/project-a" {
		t.Fatalf("Expected shared issue to keep first project dir, got %s", projectDirs["orch-go-shared"])
	}
}

func TestListInProgressIssuesEmptyProjectDirs(t *testing.T) {
	oldListOpenIssues := listOpenIssues
	oldListOpenIssuesWithDir := listOpenIssuesWithDir
	defer func() {
		listOpenIssues = oldListOpenIssues
		listOpenIssuesWithDir = oldListOpenIssuesWithDir
	}()

	listOpenIssues = func() (map[string]*verify.Issue, error) {
		return map[string]*verify.Issue{
			"orch-go-active": {ID: "orch-go-active", Status: "in_progress"},
		}, nil
	}
	listOpenIssuesWithDir = func(projectDir string) (map[string]*verify.Issue, error) {
		return nil, fmt.Errorf("unexpected call")
	}

	issues, projectDirs := listInProgressIssues(nil)
	if len(projectDirs) != 0 {
		t.Fatalf("Expected no project dir mappings, got %d", len(projectDirs))
	}
	if _, ok := issues["orch-go-active"]; !ok {
		t.Fatal("Expected in_progress issue to be included")
	}
}

func TestListInProgressIssuesErrorHandling(t *testing.T) {
	oldListOpenIssues := listOpenIssues
	oldListOpenIssuesWithDir := listOpenIssuesWithDir
	defer func() {
		listOpenIssues = oldListOpenIssues
		listOpenIssuesWithDir = oldListOpenIssuesWithDir
	}()

	listOpenIssues = func() (map[string]*verify.Issue, error) {
		return nil, fmt.Errorf("unexpected call")
	}
	listOpenIssuesWithDir = func(projectDir string) (map[string]*verify.Issue, error) {
		if projectDir == "/tmp/project-a" {
			return nil, fmt.Errorf("boom")
		}
		return map[string]*verify.Issue{
			"orch-go-ok": {ID: "orch-go-ok", Status: "in_progress"},
		}, nil
	}

	issues, _ := listInProgressIssues([]string{"/tmp/project-a", "/tmp/project-b"})
	if _, ok := issues["orch-go-ok"]; !ok {
		t.Fatal("Expected in_progress issue from healthy project")
	}
}

func TestHandleAgentsBeadsFirstWorkspaceEnrichment(t *testing.T) {
	oldSourceDir := sourceDir
	oldServerURL := serverURL
	oldListOpenIssues := listOpenIssues
	oldListOpenIssuesWithDir := listOpenIssuesWithDir
	oldGetIssuesBatch := getIssuesBatch
	oldGetCommentsBatch := getCommentsBatchWithProjectDirs
	oldGetKBProjectsFn := getKBProjectsFn
	oldBeadsCache := globalBeadsCache

	defer func() {
		sourceDir = oldSourceDir
		serverURL = oldServerURL
		listOpenIssues = oldListOpenIssues
		listOpenIssuesWithDir = oldListOpenIssuesWithDir
		getIssuesBatch = oldGetIssuesBatch
		getCommentsBatchWithProjectDirs = oldGetCommentsBatch
		getKBProjectsFn = oldGetKBProjectsFn
		globalBeadsCache = oldBeadsCache
		globalWorkspaceCacheInstance.invalidate()
	}()

	projectDir := t.TempDir()
	sourceDir = projectDir
	getKBProjectsFn = func() []string { return nil }
	globalWorkspaceCacheInstance.invalidate()
	globalBeadsCache = newBeadsCache()

	workspaceName := "og-feat-test-20feb-acde"
	workspacePath := filepath.Join(projectDir, ".orch", "workspace", workspaceName)
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	spawnContext := fmt.Sprintf(`TASK: Test task

You were spawned from beads issue: **orch-go-aaa1**

PROJECT_DIR: %s
`, filepath.Join(projectDir, "target-project"))
	if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatalf("Failed to write SPAWN_CONTEXT.md: %v", err)
	}
	if err := spawn.WriteSessionID(workspacePath, "sess-123"); err != nil {
		t.Fatalf("Failed to write session ID: %v", err)
	}
	spawnTime := time.Now().Add(-2 * time.Minute)
	if err := spawn.WriteSpawnTime(workspacePath, spawnTime); err != nil {
		t.Fatalf("Failed to write spawn time: %v", err)
	}

	sessions := map[string]opencode.Session{
		"sess-123": {
			ID:        "sess-123",
			Directory: "/wrong/project",
			Title:     "workspace title",
			Time: opencode.SessionTime{
				Created: time.Now().Add(-5 * time.Minute).UnixMilli(),
				Updated: time.Now().Add(-2 * time.Minute).UnixMilli(),
			},
		},
	}
	server := newTestOpenCodeServer(t, sessions, map[string][]opencode.Message{})
	serverURL = server.URL
	defer server.Close()

	listOpenIssues = func() (map[string]*verify.Issue, error) {
		return nil, fmt.Errorf("unexpected call")
	}
	listOpenIssuesWithDir = func(dir string) (map[string]*verify.Issue, error) {
		return map[string]*verify.Issue{
			"orch-go-aaa1": {ID: "orch-go-aaa1", Title: "Primary", Status: "in_progress"},
			"orch-go-bbb2": {ID: "orch-go-bbb2", Title: "No workspace", Status: "in_progress"},
		}, nil
	}
	getIssuesBatch = func(ids []string, projectDirs map[string]string) (map[string]*verify.Issue, error) {
		return map[string]*verify.Issue{
			"orch-go-aaa1": {ID: "orch-go-aaa1", Title: "Primary", Status: "in_progress"},
			"orch-go-bbb2": {ID: "orch-go-bbb2", Title: "No workspace", Status: "in_progress"},
		}, nil
	}
	getCommentsBatchWithProjectDirs = func(ids []string, projectDirs map[string]string) map[string][]verify.Comment {
		return map[string][]verify.Comment{
			"orch-go-aaa1": {
				{Text: "Phase: Complete - done", CreatedAt: time.Now().Format(time.RFC3339)},
			},
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	w := httptest.NewRecorder()
	handleAgents(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var agents []AgentAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	primary, ok := findAgentByBeadsID(agents, "orch-go-aaa1")
	if !ok {
		t.Fatal("Expected primary agent to be present")
	}
	if primary.SessionID != "sess-123" {
		t.Fatalf("Expected SessionID from workspace, got %s", primary.SessionID)
	}
	if primary.SpawnedAt == "" {
		t.Fatalf("Expected SpawnedAt from workspace")
	}
	if primary.ProjectDir != filepath.Join(projectDir, "target-project") {
		t.Fatalf("Expected ProjectDir from workspace, got %s", primary.ProjectDir)
	}
	if primary.Status != "completed" {
		t.Fatalf("Expected completed status, got %s", primary.Status)
	}

	secondary, ok := findAgentByBeadsID(agents, "orch-go-bbb2")
	if !ok {
		t.Fatal("Expected secondary agent to be present")
	}
	if secondary.ID != "" {
		t.Fatalf("Expected no workspace ID for secondary agent, got %s", secondary.ID)
	}
	if secondary.Status != "dead" {
		t.Fatalf("Expected dead status for secondary agent, got %s", secondary.Status)
	}
}

func TestHandleAgentsSessionStatusDerivedFromActivity(t *testing.T) {
	oldSourceDir := sourceDir
	oldServerURL := serverURL
	oldListOpenIssues := listOpenIssues
	oldListOpenIssuesWithDir := listOpenIssuesWithDir
	oldGetIssuesBatch := getIssuesBatch
	oldGetCommentsBatch := getCommentsBatchWithProjectDirs
	oldGetKBProjectsFn := getKBProjectsFn
	oldBeadsCache := globalBeadsCache

	defer func() {
		sourceDir = oldSourceDir
		serverURL = oldServerURL
		listOpenIssues = oldListOpenIssues
		listOpenIssuesWithDir = oldListOpenIssuesWithDir
		getIssuesBatch = oldGetIssuesBatch
		getCommentsBatchWithProjectDirs = oldGetCommentsBatch
		getKBProjectsFn = oldGetKBProjectsFn
		globalBeadsCache = oldBeadsCache
		globalWorkspaceCacheInstance.invalidate()
	}()

	projectDir := t.TempDir()
	sourceDir = projectDir
	getKBProjectsFn = func() []string { return nil }
	globalWorkspaceCacheInstance.invalidate()
	globalBeadsCache = newBeadsCache()

	workspaceName := "og-feat-test-active-20feb-acde"
	workspacePath := filepath.Join(projectDir, ".orch", "workspace", workspaceName)
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	spawnContext := `TASK: Test task

You were spawned from beads issue: **orch-go-active**

PROJECT_DIR: /tmp/active
`
	if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatalf("Failed to write SPAWN_CONTEXT.md: %v", err)
	}
	if err := spawn.WriteSessionID(workspacePath, "sess-active"); err != nil {
		t.Fatalf("Failed to write session ID: %v", err)
	}

	now := time.Now()
	sessions := map[string]opencode.Session{
		"sess-active": {
			ID:        "sess-active",
			Directory: projectDir,
			Time: opencode.SessionTime{
				Created: now.Add(-4 * time.Minute).UnixMilli(),
				Updated: now.Add(-2 * time.Minute).UnixMilli(),
			},
		},
	}
	server := newTestOpenCodeServer(t, sessions, map[string][]opencode.Message{})
	serverURL = server.URL
	defer server.Close()

	listOpenIssues = func() (map[string]*verify.Issue, error) {
		return nil, fmt.Errorf("unexpected call")
	}
	listOpenIssuesWithDir = func(dir string) (map[string]*verify.Issue, error) {
		return map[string]*verify.Issue{
			"orch-go-active": {ID: "orch-go-active", Title: "Active", Status: "in_progress"},
		}, nil
	}
	getIssuesBatch = func(ids []string, projectDirs map[string]string) (map[string]*verify.Issue, error) {
		return map[string]*verify.Issue{
			"orch-go-active": {ID: "orch-go-active", Title: "Active", Status: "in_progress"},
		}, nil
	}
	getCommentsBatchWithProjectDirs = func(ids []string, projectDirs map[string]string) map[string][]verify.Comment {
		return map[string][]verify.Comment{}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	w := httptest.NewRecorder()
	handleAgents(w, req)

	var agents []AgentAPIResponse
	if err := json.NewDecoder(w.Result().Body).Decode(&agents); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	agent, ok := findAgentByBeadsID(agents, "orch-go-active")
	if !ok {
		t.Fatal("Expected active agent to be present")
	}
	if agent.Status != "active" {
		t.Fatalf("Expected active status from session activity, got %s", agent.Status)
	}
}

func TestHandleAgentsSessionNotFoundDefaultsToDead(t *testing.T) {
	oldSourceDir := sourceDir
	oldServerURL := serverURL
	oldListOpenIssues := listOpenIssues
	oldListOpenIssuesWithDir := listOpenIssuesWithDir
	oldGetIssuesBatch := getIssuesBatch
	oldGetCommentsBatch := getCommentsBatchWithProjectDirs
	oldGetKBProjectsFn := getKBProjectsFn
	oldBeadsCache := globalBeadsCache

	defer func() {
		sourceDir = oldSourceDir
		serverURL = oldServerURL
		listOpenIssues = oldListOpenIssues
		listOpenIssuesWithDir = oldListOpenIssuesWithDir
		getIssuesBatch = oldGetIssuesBatch
		getCommentsBatchWithProjectDirs = oldGetCommentsBatch
		getKBProjectsFn = oldGetKBProjectsFn
		globalBeadsCache = oldBeadsCache
		globalWorkspaceCacheInstance.invalidate()
	}()

	projectDir := t.TempDir()
	sourceDir = projectDir
	getKBProjectsFn = func() []string { return nil }
	globalWorkspaceCacheInstance.invalidate()
	globalBeadsCache = newBeadsCache()

	workspaceName := "og-feat-test-dead-20feb-acde"
	workspacePath := filepath.Join(projectDir, ".orch", "workspace", workspaceName)
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	spawnContext := `TASK: Test task

You were spawned from beads issue: **orch-go-dead**

PROJECT_DIR: /tmp/dead
`
	if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatalf("Failed to write SPAWN_CONTEXT.md: %v", err)
	}
	if err := spawn.WriteSessionID(workspacePath, "sess-missing"); err != nil {
		t.Fatalf("Failed to write session ID: %v", err)
	}

	server := newTestOpenCodeServer(t, map[string]opencode.Session{}, map[string][]opencode.Message{})
	serverURL = server.URL
	defer server.Close()

	listOpenIssues = func() (map[string]*verify.Issue, error) {
		return nil, fmt.Errorf("unexpected call")
	}
	listOpenIssuesWithDir = func(dir string) (map[string]*verify.Issue, error) {
		return map[string]*verify.Issue{
			"orch-go-dead": {ID: "orch-go-dead", Title: "Dead", Status: "in_progress"},
		}, nil
	}
	getIssuesBatch = func(ids []string, projectDirs map[string]string) (map[string]*verify.Issue, error) {
		return map[string]*verify.Issue{
			"orch-go-dead": {ID: "orch-go-dead", Title: "Dead", Status: "in_progress"},
		}, nil
	}
	getCommentsBatchWithProjectDirs = func(ids []string, projectDirs map[string]string) map[string][]verify.Comment {
		return map[string][]verify.Comment{}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	w := httptest.NewRecorder()
	handleAgents(w, req)

	var agents []AgentAPIResponse
	if err := json.NewDecoder(w.Result().Body).Decode(&agents); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	agent, ok := findAgentByBeadsID(agents, "orch-go-dead")
	if !ok {
		t.Fatal("Expected dead agent to be present")
	}
	if agent.Status != "dead" {
		t.Fatalf("Expected dead status when session fetch fails, got %s", agent.Status)
	}
}
