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

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

func TestHandleAgents(t *testing.T) {
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

func TestHandleUsageMethodNotAllowed(t *testing.T) {
	// Test POST method is not allowed
	req := httptest.NewRequest(http.MethodPost, "/api/usage", nil)
	w := httptest.NewRecorder()

	handleUsage(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleUsageJSONResponse(t *testing.T) {
	// Test that usage endpoint returns valid JSON
	req := httptest.NewRequest(http.MethodGet, "/api/usage", nil)
	w := httptest.NewRecorder()

	handleUsage(w, req)

	resp := w.Result()
	// Should be 200 even if auth fails (returns error in JSON)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Verify it returns valid JSON
	var usageResp UsageAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&usageResp); err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}

	// Response should either have data or an error
	// If no auth is configured, we expect an error message
	if usageResp.Error == "" && usageResp.Account == "" && usageResp.FiveHour == 0 && usageResp.Weekly == 0 {
		t.Log("Usage response has no data and no error - auth may be working")
	}
}

func TestUsageAPIResponseJSONFormat(t *testing.T) {
	// Test that UsageAPIResponse serializes correctly to JSON
	usage := &UsageAPIResponse{
		Account:    "test@example.com",
		FiveHour:   45.5,
		Weekly:     72.3,
		WeeklyOpus: 15.0,
	}

	data, err := json.Marshal(usage)
	if err != nil {
		t.Fatalf("Failed to marshal UsageAPIResponse: %v", err)
	}

	// Verify the JSON contains expected fields
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if result["account"] != "test@example.com" {
		t.Errorf("Expected account 'test@example.com', got %v", result["account"])
	}
	if result["five_hour_percent"] != 45.5 {
		t.Errorf("Expected five_hour_percent 45.5, got %v", result["five_hour_percent"])
	}
	if result["weekly_percent"] != 72.3 {
		t.Errorf("Expected weekly_percent 72.3, got %v", result["weekly_percent"])
	}
	if result["weekly_opus_percent"] != 15.0 {
		t.Errorf("Expected weekly_opus_percent 15.0, got %v", result["weekly_opus_percent"])
	}
}

func TestServeStatusWithMockServer(t *testing.T) {
	// Create a mock server that responds to /health
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Parse the port from the test server URL
	// The URL is in format http://127.0.0.1:PORT (httptest.Server always uses 127.0.0.1)
	var testPort int
	_, err := fmt.Sscanf(server.URL, "http://127.0.0.1:%d", &testPort)
	if err != nil {
		t.Fatalf("Failed to parse test server port: %v", err)
	}

	// Call runServeStatus with the test port
	// This should succeed without error
	err = runServeStatus(testPort)
	if err != nil {
		t.Errorf("Expected no error from runServeStatus, got: %v", err)
	}
}

func TestServeStatusWithNoServer(t *testing.T) {
	// Use a port that is unlikely to be in use
	unusedPort := 59999

	// Call runServeStatus with the unused port
	// This should NOT return an error (it prints status and returns nil)
	err := runServeStatus(unusedPort)
	if err != nil {
		t.Errorf("Expected no error from runServeStatus (should print 'not running'), got: %v", err)
	}
}

func TestDefaultServePort(t *testing.T) {
	// Verify the default port constant
	if DefaultServePort != 3348 {
		t.Errorf("Expected DefaultServePort to be 3348, got %d", DefaultServePort)
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
	// Import the opencode package types inline
	type testSession struct {
		Directory string
	}

	tests := []struct {
		name              string
		currentProjectDir string
		sessionDirs       []string
		expectedCount     int
	}{
		{
			name:              "empty sessions with current dir",
			currentProjectDir: "/home/user/project1",
			sessionDirs:       []string{},
			expectedCount:     1, // just current dir
		},
		{
			name:              "single session same as current",
			currentProjectDir: "/home/user/project1",
			sessionDirs:       []string{"/home/user/project1"},
			expectedCount:     1, // deduplicated
		},
		{
			name:              "multiple sessions different dirs",
			currentProjectDir: "/home/user/project1",
			sessionDirs:       []string{"/home/user/project2", "/home/user/project3"},
			expectedCount:     3, // current + 2 others
		},
		{
			name:              "duplicate session directories",
			currentProjectDir: "/home/user/project1",
			sessionDirs:       []string{"/home/user/project2", "/home/user/project2", "/home/user/project3"},
			expectedCount:     3, // deduped
		},
		{
			name:              "empty current dir",
			currentProjectDir: "",
			sessionDirs:       []string{"/home/user/project2", "/home/user/project3"},
			expectedCount:     2,
		},
		{
			name:              "empty session dir skipped",
			currentProjectDir: "/home/user/project1",
			sessionDirs:       []string{"", "/home/user/project2"},
			expectedCount:     2, // empty is skipped
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
			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d project dirs, got %d: %v", tt.expectedCount, len(result), result)
			}

			// Verify current project dir is always first if provided
			if tt.currentProjectDir != "" && len(result) > 0 {
				if result[0] != filepath.Clean(tt.currentProjectDir) {
					t.Errorf("Expected current project dir %q to be first, got %q", tt.currentProjectDir, result[0])
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

func TestHandleErrorsMethodNotAllowed(t *testing.T) {
	// Test POST method is not allowed
	req := httptest.NewRequest(http.MethodPost, "/api/errors", nil)
	w := httptest.NewRecorder()

	handleErrors(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleErrorsJSONResponse(t *testing.T) {
	// Test that errors endpoint returns valid JSON
	req := httptest.NewRequest(http.MethodGet, "/api/errors", nil)
	w := httptest.NewRecorder()

	handleErrors(w, req)

	resp := w.Result()
	// Should be 200 even if events file doesn't exist
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Verify it returns valid JSON
	var errorsResp ErrorsAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&errorsResp); err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}

	// Verify ByType map is initialized (not nil)
	if errorsResp.ByType == nil {
		t.Error("Expected ByType to be initialized map, got nil")
	}
}

func TestErrorsAPIResponseJSONFormat(t *testing.T) {
	// Test that ErrorsAPIResponse serializes correctly to JSON
	errors := &ErrorsAPIResponse{
		TotalErrors:    5,
		ErrorsLast24h:  2,
		ErrorsLast7d:   4,
		AbandonedCount: 3,
		SessionErrors:  2,
		RecentErrors: []ErrorEvent{
			{
				Type:      "agent.abandoned",
				BeadsID:   "test-abc123",
				Timestamp: "2025-12-26T12:00:00Z",
				Message:   "Stalled during execution",
				Skill:     "feature-impl",
			},
		},
		Patterns: []ErrorPattern{
			{
				Pattern:    "Stalled during",
				Count:      3,
				LastSeen:   "2025-12-26T12:00:00Z",
				BeadsIDs:   []string{"test-abc1", "test-abc2", "test-abc3"},
				Suggestion: "Check agent for long-running operations or API timeouts",
			},
		},
		ByType: map[string]int{
			"agent.abandoned": 3,
			"session.error":   2,
		},
	}

	data, err := json.Marshal(errors)
	if err != nil {
		t.Fatalf("Failed to marshal ErrorsAPIResponse: %v", err)
	}

	// Verify the JSON contains expected fields
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if result["total_errors"] != float64(5) {
		t.Errorf("Expected total_errors 5, got %v", result["total_errors"])
	}
	if result["errors_last_24h"] != float64(2) {
		t.Errorf("Expected errors_last_24h 2, got %v", result["errors_last_24h"])
	}
	if result["abandoned_count"] != float64(3) {
		t.Errorf("Expected abandoned_count 3, got %v", result["abandoned_count"])
	}

	// Check patterns array exists
	if _, ok := result["patterns"]; !ok {
		t.Error("Expected patterns field in JSON")
	}

	// Check by_type map
	byType, ok := result["by_type"].(map[string]interface{})
	if !ok {
		t.Error("Expected by_type map in JSON")
	} else {
		if byType["agent.abandoned"] != float64(3) {
			t.Errorf("Expected by_type[agent.abandoned] 3, got %v", byType["agent.abandoned"])
		}
	}
}

func TestExtractSkillFromAgentID(t *testing.T) {
	tests := []struct {
		agentID  string
		expected string
	}{
		{"og-feat-test-26dec", "feature-impl"},
		{"og-debug-something-26dec", "systematic-debugging"},
		{"og-inv-investigation-26dec", "investigation"},
		{"og-arch-design-26dec", "architect"},
		{"og-work-session-26dec", "design-session"},
		{"og-unknown-test", "unknown"},
		{"invalid", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.agentID, func(t *testing.T) {
			result := extractSkillFromAgentID(tt.agentID)
			if result != tt.expected {
				t.Errorf("extractSkillFromAgentID(%q) = %q, want %q", tt.agentID, result, tt.expected)
			}
		})
	}
}

func TestNormalizeErrorMessage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple error", "Simple error"},
		{"  Trimmed  ", "Trimmed"},
		{"", ""},
		{
			// Long message should be truncated to 100 chars
			"This is a very long error message that exceeds the 100 character limit and should be truncated for pattern matching purposes",
			"This is a very long error message that exceeds the 100 character limit and should be truncated for p",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeErrorMessage(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeErrorMessage(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		slice    []string
		s        string
		expected bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
		{nil, "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			result := containsString(tt.slice, tt.s)
			if result != tt.expected {
				t.Errorf("containsString(%v, %q) = %v, want %v", tt.slice, tt.s, result, tt.expected)
			}
		})
	}
}

func TestSuggestRemediation(t *testing.T) {
	tests := []struct {
		pattern  string
		expected string
	}{
		{"Agent stalled during execution", "Check agent for long-running operations or API timeouts"},
		{"Connection timeout occurred", "Review API response times or increase timeout limits"},
		{"At capacity limit", "Increase daemon capacity or check for stuck agents"},
		{"Daemon not responding", "Check daemon logs at ~/.orch/daemon.log"},
		{"Missing context information", "Review spawn context for missing or incorrect information"},
		{"Connection refused", "Check network connectivity or API endpoint availability"},
		{"Unknown error", "Review agent workspace for more details"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			result := suggestRemediation(tt.pattern)
			if result != tt.expected {
				t.Errorf("suggestRemediation(%q) = %q, want %q", tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestHandleErrorsWithTestData(t *testing.T) {
	// Create a temporary events file with test data
	tmpDir := t.TempDir()
	tmpLogPath := filepath.Join(tmpDir, "events.jsonl")

	now := time.Now()
	testEvents := []events.Event{
		{
			Type:      "session.error",
			SessionID: "sess_123",
			Timestamp: now.Add(-1 * time.Hour).Unix(),
			Data:      map[string]interface{}{"error": "Connection timeout"},
		},
		{
			Type:      "agent.abandoned",
			SessionID: "",
			Timestamp: now.Add(-2 * time.Hour).Unix(),
			Data: map[string]interface{}{
				"beads_id":       "test-abc123",
				"reason":         "Stalled during lunch",
				"agent_id":       "og-feat-test-26dec",
				"workspace_path": "/path/to/workspace/og-feat-test-26dec",
			},
		},
		{
			Type:      "session.spawned", // Non-error event, should be skipped
			SessionID: "sess_456",
			Timestamp: now.Unix(),
			Data:      map[string]interface{}{"title": "test spawn"},
		},
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

	// Note: handleErrors uses events.DefaultLogPath() which we can't easily override
	// This test verifies the parsing logic through helper functions instead
	t.Run("verify error event types are recognized", func(t *testing.T) {
		if events.EventTypeSessionError != "session.error" {
			t.Errorf("Expected session.error, got %s", events.EventTypeSessionError)
		}
	})

	t.Run("verify skill extraction from agent abandoned events", func(t *testing.T) {
		skill := extractSkillFromAgentID("og-feat-test-26dec")
		if skill != "feature-impl" {
			t.Errorf("Expected feature-impl, got %s", skill)
		}
	})
}

// TestIsUntrackedBeadsIDServe tests the untracked beads ID detection function.
func TestIsUntrackedBeadsIDServe(t *testing.T) {
	tests := []struct {
		name    string
		beadsID string
		want    bool
	}{
		{"standard tracked ID", "orch-go-abc123", false},
		{"tracked ID with suffix", "orch-go-feat-add-feature-19dec", false},
		{"untracked ID", "orch-go-untracked-1766695797", true},
		{"untracked ID other project", "kb-cli-untracked-1766695797", true},
		{"empty ID", "", false},
		{"project-untracked-pattern", "snap-untracked-1234567890", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUntrackedBeadsIDServe(tt.beadsID)
			if got != tt.want {
				t.Errorf("isUntrackedBeadsIDServe(%q) = %v, want %v", tt.beadsID, got, tt.want)
			}
		})
	}
}

// TestReadWorkspacePhase tests the workspace phase file reading function.
func TestReadWorkspacePhase(t *testing.T) {
	t.Run("returns empty for non-existent path", func(t *testing.T) {
		phase := readWorkspacePhase("/nonexistent/path")
		if phase != "" {
			t.Errorf("Expected empty string for non-existent path, got %q", phase)
		}
	})

	t.Run("returns empty for empty path", func(t *testing.T) {
		phase := readWorkspacePhase("")
		if phase != "" {
			t.Errorf("Expected empty string for empty path, got %q", phase)
		}
	})

	t.Run("reads phase from .phase file", func(t *testing.T) {
		// Create temp workspace
		tmpDir := t.TempDir()
		phasePath := filepath.Join(tmpDir, ".phase")

		// Write phase file
		if err := os.WriteFile(phasePath, []byte("Implementing\n"), 0644); err != nil {
			t.Fatalf("Failed to write phase file: %v", err)
		}

		phase := readWorkspacePhase(tmpDir)
		if phase != "Implementing" {
			t.Errorf("Expected 'Implementing', got %q", phase)
		}
	})

	t.Run("trims whitespace from phase", func(t *testing.T) {
		tmpDir := t.TempDir()
		phasePath := filepath.Join(tmpDir, ".phase")

		if err := os.WriteFile(phasePath, []byte("  Planning  \n\n"), 0644); err != nil {
			t.Fatalf("Failed to write phase file: %v", err)
		}

		phase := readWorkspacePhase(tmpDir)
		if phase != "Planning" {
			t.Errorf("Expected 'Planning', got %q", phase)
		}
	})
}

// TestCheckWorkspaceSynthesis tests the synthesis file detection function.
func TestCheckWorkspaceSynthesisFunction(t *testing.T) {
	t.Run("returns false for non-existent path", func(t *testing.T) {
		result := checkWorkspaceSynthesis("/nonexistent/path")
		if result {
			t.Error("Expected false for non-existent path")
		}
	})

	t.Run("returns false for empty path", func(t *testing.T) {
		result := checkWorkspaceSynthesis("")
		if result {
			t.Error("Expected false for empty path")
		}
	})

	t.Run("returns true when SYNTHESIS.md exists and is non-empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")

		if err := os.WriteFile(synthesisPath, []byte("# Synthesis\n\nContent here"), 0644); err != nil {
			t.Fatalf("Failed to write synthesis file: %v", err)
		}

		result := checkWorkspaceSynthesis(tmpDir)
		if !result {
			t.Error("Expected true when SYNTHESIS.md exists and is non-empty")
		}
	})

	t.Run("returns false when SYNTHESIS.md is empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		synthesisPath := filepath.Join(tmpDir, "SYNTHESIS.md")

		if err := os.WriteFile(synthesisPath, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to write empty synthesis file: %v", err)
		}

		result := checkWorkspaceSynthesis(tmpDir)
		if result {
			t.Error("Expected false when SYNTHESIS.md is empty")
		}
	})
}
