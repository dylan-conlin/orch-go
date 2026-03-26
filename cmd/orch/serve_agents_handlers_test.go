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

	"github.com/dylan-conlin/orch-go/pkg/discovery"
	"github.com/dylan-conlin/orch-go/pkg/execution"
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
		// Handle /session/status endpoint
		if r.URL.Path == "/session/status" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]execution.SessionStatusInfo{})
			return
		}

		if r.URL.Path == "/session" || r.URL.Path == "/session/" {
			// List sessions endpoint
			result := make([]opencode.Session, 0, len(sessions))
			for _, s := range sessions {
				result = append(result, s)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(result)
			return
		}

		if r.URL.Path == "/session" && r.Method == http.MethodPut {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(opencode.Session{})
			return
		}

		parts := splitSessionPath(r.URL.Path)
		if parts == nil {
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

// splitSessionPath extracts parts after /session/ prefix.
func splitSessionPath(path string) []string {
	const prefix = "/session/"
	if len(path) <= len(prefix) || path[:len(prefix)] != prefix {
		return nil
	}
	relative := path[len(prefix):]
	parts := make([]string, 0, 2)
	for _, p := range splitTrim(relative, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return nil
	}
	return parts
}

func splitTrim(s, sep string) []string {
	var result []string
	for s != "" {
		idx := indexOf(s, sep)
		if idx < 0 {
			result = append(result, s)
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	return result
}

func indexOf(s, substr string) int {
	for i := range s {
		if i+len(substr) <= len(s) && s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// testHandlerSetup saves and restores global state for handler tests.
type testHandlerSetup struct {
	oldSourceDir                    string
	oldServerURL                    string
	oldGetKBProjectsFn              func() []string
	oldBeadsCache                   *beadsCache
	oldQueryTrackedAgentsFn         func([]string) ([]discovery.AgentStatus, error)
	oldGetIssuesBatch               func([]string, map[string]string) (map[string]*verify.Issue, error)
	oldGetCommentsBatchWithProjDirs func([]string, map[string]string) map[string][]verify.Comment
}

func setupHandlerTest(t *testing.T, projectDir string) *testHandlerSetup {
	t.Helper()
	s := &testHandlerSetup{
		oldSourceDir:                    sourceDir,
		oldServerURL:                    serverURL,
		oldGetKBProjectsFn:              getKBProjectsFn,
		oldBeadsCache:                   globalBeadsCache,
		oldQueryTrackedAgentsFn:         queryTrackedAgentsFn,
		oldGetIssuesBatch:               getIssuesBatch,
		oldGetCommentsBatchWithProjDirs: getCommentsBatchWithProjectDirs,
	}

	sourceDir = projectDir
	getKBProjectsFn = func() []string { return nil }
	globalWorkspaceCacheInstance.invalidate()
	globalTrackedAgentsCache.invalidate()
	globalBeadsCache = newBeadsCache()

	return s
}

func (s *testHandlerSetup) restore() {
	sourceDir = s.oldSourceDir
	serverURL = s.oldServerURL
	getKBProjectsFn = s.oldGetKBProjectsFn
	globalBeadsCache = s.oldBeadsCache
	queryTrackedAgentsFn = s.oldQueryTrackedAgentsFn
	getIssuesBatch = s.oldGetIssuesBatch
	getCommentsBatchWithProjectDirs = s.oldGetCommentsBatchWithProjDirs
	globalWorkspaceCacheInstance.invalidate()
	globalTrackedAgentsCache.invalidate()
}

func TestHandleAgents(t *testing.T) {
	if globalBeadsCache == nil {
		globalBeadsCache = newBeadsCache()
	}

	// Mock queryTrackedAgentsFn to return empty (no beads/workspace in test env)
	oldFn := queryTrackedAgentsFn
	defer func() { queryTrackedAgentsFn = oldFn }()
	queryTrackedAgentsFn = func(dirs []string) ([]discovery.AgentStatus, error) {
		return nil, nil
	}
	globalTrackedAgentsCache.invalidate()

	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	w := httptest.NewRecorder()

	handleAgents(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var agents []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}
}

func TestHandleAgentsMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/agents", nil)
	w := httptest.NewRecorder()

	handleAgents(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleAgentsBeadsFirstWorkspaceEnrichment(t *testing.T) {
	projectDir := t.TempDir()
	setup := setupHandlerTest(t, projectDir)
	defer setup.restore()

	// Create workspace directory with SPAWN_CONTEXT.md for enrichment path
	workspaceName := "og-feat-test-20feb-acde"
	workspacePath := filepath.Join(projectDir, ".orch", "workspace", workspaceName)
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	targetProjectDir := filepath.Join(projectDir, "target-project")
	spawnContext := fmt.Sprintf(`TASK: Test task

You were spawned from beads issue: **orch-go-aaa1**

PROJECT_DIR: %s
`, targetProjectDir)
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

	// Set up OpenCode test server with session data
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

	// Mock queryTrackedAgentsFn - the core discovery path
	// Status "active" so determineAgentStatus sees Phase Complete + active session → "completed"
	queryTrackedAgentsFn = func(dirs []string) ([]discovery.AgentStatus, error) {
		return []discovery.AgentStatus{
			{
				BeadsID:       "orch-go-aaa1",
				Title:         "Primary",
				SessionID:     "sess-123",
				ProjectDir:    targetProjectDir,
				WorkspaceName: workspaceName,
				Phase:         "Complete - done",
				Status:        "active",
			},
		}, nil
	}

	// Mock enrichment data sources (still used by dashboard enrichment)
	getIssuesBatch = func(ids []string, projectDirs map[string]string) (map[string]*verify.Issue, error) {
		return map[string]*verify.Issue{
			"orch-go-aaa1": {ID: "orch-go-aaa1", Title: "Primary", Status: "in_progress"},
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
	if primary.ProjectDir != targetProjectDir {
		t.Fatalf("Expected ProjectDir from workspace, got %s", primary.ProjectDir)
	}
	if primary.Status != "completed" {
		t.Fatalf("Expected completed status (Phase Complete + workspace), got %s", primary.Status)
	}
}

func TestHandleAgentsSessionStatusDerivedFromActivity(t *testing.T) {
	projectDir := t.TempDir()
	setup := setupHandlerTest(t, projectDir)
	defer setup.restore()

	// Create workspace for enrichment
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

	// Mock queryTrackedAgentsFn - agent is active
	queryTrackedAgentsFn = func(dirs []string) ([]discovery.AgentStatus, error) {
		return []discovery.AgentStatus{
			{
				BeadsID:       "orch-go-active",
				Title:         "Active",
				SessionID:     "sess-active",
				ProjectDir:    "/tmp/active",
				WorkspaceName: workspaceName,
				Status:        "active",
			},
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

func TestHandleAgentsOpenStatusIssueVisible(t *testing.T) {
	projectDir := t.TempDir()
	setup := setupHandlerTest(t, projectDir)
	defer setup.restore()

	// Create workspace for enrichment
	workspaceName := "og-feat-test-open-20feb-acde"
	workspacePath := filepath.Join(projectDir, ".orch", "workspace", workspaceName)
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	spawnContext := fmt.Sprintf(`TASK: Auto-created task

You were spawned from beads issue: **orch-go-open1**

PROJECT_DIR: %s
`, projectDir)
	if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatalf("Failed to write SPAWN_CONTEXT.md: %v", err)
	}
	if err := spawn.WriteSessionID(workspacePath, "sess-open1"); err != nil {
		t.Fatalf("Failed to write session ID: %v", err)
	}

	now := time.Now()
	sessions := map[string]opencode.Session{
		"sess-open1": {
			ID:        "sess-open1",
			Directory: projectDir,
			Time: opencode.SessionTime{
				Created: now.Add(-3 * time.Minute).UnixMilli(),
				Updated: now.Add(-1 * time.Minute).UnixMilli(),
			},
		},
	}
	server := newTestOpenCodeServer(t, sessions, map[string][]opencode.Message{})
	serverURL = server.URL
	defer server.Close()

	// Mock queryTrackedAgentsFn - agent with "open" status issue
	queryTrackedAgentsFn = func(dirs []string) ([]discovery.AgentStatus, error) {
		return []discovery.AgentStatus{
			{
				BeadsID:       "orch-go-open1",
				Title:         "Auto-created task",
				SessionID:     "sess-open1",
				ProjectDir:    projectDir,
				WorkspaceName: workspaceName,
				Status:        "active",
			},
		}, nil
	}

	getIssuesBatch = func(ids []string, projectDirs map[string]string) (map[string]*verify.Issue, error) {
		return map[string]*verify.Issue{
			"orch-go-open1": {ID: "orch-go-open1", Title: "Auto-created task", Status: "open"},
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

	agent, ok := findAgentByBeadsID(agents, "orch-go-open1")
	if !ok {
		t.Fatal("Expected agent with 'open' status beads issue to be visible in dashboard (regression: orch-go-1066)")
	}
	if agent.Status == "" {
		t.Fatal("Expected agent to have a status")
	}
	if agent.SessionID != "sess-open1" {
		t.Fatalf("Expected session ID from workspace, got %s", agent.SessionID)
	}
}

func TestHandleAgentsSessionNotFoundDefaultsToDead(t *testing.T) {
	projectDir := t.TempDir()
	setup := setupHandlerTest(t, projectDir)
	defer setup.restore()

	// Create workspace for enrichment
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

	// OpenCode server has NO sessions - simulates missing session
	server := newTestOpenCodeServer(t, map[string]opencode.Session{}, map[string][]opencode.Message{})
	serverURL = server.URL
	defer server.Close()

	// Mock queryTrackedAgentsFn - agent has session ID but session is idle/dead
	queryTrackedAgentsFn = func(dirs []string) ([]discovery.AgentStatus, error) {
		return []discovery.AgentStatus{
			{
				BeadsID:       "orch-go-dead",
				Title:         "Dead",
				SessionID:     "sess-missing",
				ProjectDir:    "/tmp/dead",
				WorkspaceName: workspaceName,
				Status:        "idle",
				SessionDead:   true,
				Reason:        "session_idle",
			},
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
		t.Fatalf("Expected dead status when session is idle, got %s", agent.Status)
	}
}

func TestHandleAgentsReviewLabelWithoutWorkspace(t *testing.T) {
	projectDir := t.TempDir()
	setup := setupHandlerTest(t, projectDir)
	defer setup.restore()

	server := newTestOpenCodeServer(t, map[string]opencode.Session{}, map[string][]opencode.Message{})
	serverURL = server.URL
	defer server.Close()

	// Mock queryTrackedAgentsFn - issue with no workspace binding (missing_binding)
	queryTrackedAgentsFn = func(dirs []string) ([]discovery.AgentStatus, error) {
		return []discovery.AgentStatus{
			{
				BeadsID:        "orch-go-review",
				Title:          "Needs review",
				Status:         "unknown",
				MissingBinding: true,
				Reason:         "missing_binding",
			},
		}, nil
	}

	getIssuesBatch = func(ids []string, projectDirs map[string]string) (map[string]*verify.Issue, error) {
		return map[string]*verify.Issue{
			"orch-go-review": {ID: "orch-go-review", Title: "Needs review", Status: "in_progress", Labels: []string{"daemon:ready-review"}},
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

	if _, ok := findAgentByBeadsID(agents, "orch-go-review"); !ok {
		t.Fatal("Expected review-queue issue to be included without workspace/session")
	}
}

func TestHandleAgents_ClaudeActiveIsProcessing(t *testing.T) {
	// Integration: active Claude agent with IsProcessing from discovery → IsProcessing=true
	projectDir := t.TempDir()
	setup := setupHandlerTest(t, projectDir)
	defer setup.restore()

	// No OpenCode server needed — Claude agents don't have sessions
	serverURL = ""

	// Discovery now provides IsProcessing directly from IsPaneActive.
	// No consumer-side tmux override needed.
	queryTrackedAgentsFn = func(dirs []string) ([]discovery.AgentStatus, error) {
		return []discovery.AgentStatus{
			{
				BeadsID:       "orch-go-clive",
				Title:         "Claude agent actively running",
				ProjectDir:    projectDir,
				SpawnMode:     "claude",
				WorkspaceName: "og-feat-claude-live-25mar-abcd",
				Phase:         "Implementing - Writing tests",
				Status:        "active",
				Reason:        "tmux_pane_active",
				IsProcessing:  true,
				TmuxWindowID:  "@42",
			},
		}, nil
	}
	getIssuesBatch = func(ids []string, projectDirs map[string]string) (map[string]*verify.Issue, error) {
		return map[string]*verify.Issue{
			"orch-go-clive": {ID: "orch-go-clive", Title: "Claude agent actively running", Status: "in_progress"},
		}, nil
	}
	getCommentsBatchWithProjectDirs = func(ids []string, projectDirs map[string]string) map[string][]verify.Comment {
		return map[string][]verify.Comment{
			"orch-go-clive": {
				{Text: "Phase: Implementing - Writing tests", CreatedAt: time.Now().Add(-2 * time.Minute).Format(time.RFC3339)},
			},
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	w := httptest.NewRecorder()
	handleAgents(w, req)

	var agents []AgentAPIResponse
	if err := json.NewDecoder(w.Result().Body).Decode(&agents); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	agent, ok := findAgentByBeadsID(agents, "orch-go-clive")
	if !ok {
		t.Fatal("Expected Claude agent to be present")
	}
	if !agent.IsProcessing {
		t.Error("Expected IsProcessing=true for Claude agent with active tmux pane")
	}
	if agent.Status != "active" {
		t.Errorf("Expected Status active, got %s", agent.Status)
	}
}

func TestHandleAgents_ClaudeDeadNotProcessing(t *testing.T) {
	// Integration: dead Claude agent (discovery reports dead) → IsProcessing=false
	projectDir := t.TempDir()
	setup := setupHandlerTest(t, projectDir)
	defer setup.restore()

	serverURL = ""

	// Discovery now reports dead status and IsProcessing=false for agents
	// whose tmux pane is gone. No consumer-side tmux check needed.
	queryTrackedAgentsFn = func(dirs []string) ([]discovery.AgentStatus, error) {
		return []discovery.AgentStatus{
			{
				BeadsID:       "orch-go-cdead",
				Title:         "Claude agent that died",
				ProjectDir:    projectDir,
				SpawnMode:     "claude",
				WorkspaceName: "og-feat-claude-dead-25mar-efgh",
				Phase:         "Implementing - Was working",
				Status:        "dead",
				Reason:        "no_tmux_window",
				IsProcessing:  false,
			},
		}, nil
	}
	getIssuesBatch = func(ids []string, projectDirs map[string]string) (map[string]*verify.Issue, error) {
		return map[string]*verify.Issue{
			"orch-go-cdead": {ID: "orch-go-cdead", Title: "Claude agent that died", Status: "in_progress"},
		}, nil
	}
	getCommentsBatchWithProjectDirs = func(ids []string, projectDirs map[string]string) map[string][]verify.Comment {
		return map[string][]verify.Comment{
			"orch-go-cdead": {
				{Text: "Phase: Implementing - Was working", CreatedAt: time.Now().Add(-45 * time.Minute).Format(time.RFC3339)},
			},
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	w := httptest.NewRecorder()
	handleAgents(w, req)

	var agents []AgentAPIResponse
	if err := json.NewDecoder(w.Result().Body).Decode(&agents); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	agent, ok := findAgentByBeadsID(agents, "orch-go-cdead")
	if !ok {
		t.Fatal("Expected dead Claude agent to be present")
	}
	if agent.IsProcessing {
		t.Error("Expected IsProcessing=false for Claude agent with no tmux window")
	}
	if agent.Status != "dead" {
		t.Errorf("Expected Status dead, got %s", agent.Status)
	}
}

func TestHandleAgents_ClaudeIdlePaneNotProcessing(t *testing.T) {
	// Integration: Claude agent with idle pane (discovery reports dead) → IsProcessing=false
	projectDir := t.TempDir()
	setup := setupHandlerTest(t, projectDir)
	defer setup.restore()

	serverURL = ""

	// Discovery now detects idle pane and sets Status=dead, IsProcessing=false.
	// No consumer-side tmux check needed.
	queryTrackedAgentsFn = func(dirs []string) ([]discovery.AgentStatus, error) {
		return []discovery.AgentStatus{
			{
				BeadsID:       "orch-go-cidle",
				Title:         "Claude agent idle pane",
				ProjectDir:    projectDir,
				SpawnMode:     "claude",
				WorkspaceName: "og-feat-claude-idle-25mar-ijkl",
				Phase:         "Implementing - Was working",
				Status:        "dead",
				Reason:        "tmux_pane_idle",
				TmuxWindowID:  "@99",
				IsProcessing:  false,
			},
		}, nil
	}
	getIssuesBatch = func(ids []string, projectDirs map[string]string) (map[string]*verify.Issue, error) {
		return map[string]*verify.Issue{
			"orch-go-cidle": {ID: "orch-go-cidle", Title: "Claude agent idle pane", Status: "in_progress"},
		}, nil
	}
	getCommentsBatchWithProjectDirs = func(ids []string, projectDirs map[string]string) map[string][]verify.Comment {
		return map[string][]verify.Comment{
			"orch-go-cidle": {
				{Text: "Phase: Implementing - Was working", CreatedAt: time.Now().Add(-10 * time.Minute).Format(time.RFC3339)},
			},
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	w := httptest.NewRecorder()
	handleAgents(w, req)

	var agents []AgentAPIResponse
	if err := json.NewDecoder(w.Result().Body).Decode(&agents); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	agent, ok := findAgentByBeadsID(agents, "orch-go-cidle")
	if !ok {
		t.Fatal("Expected idle-pane Claude agent to be present")
	}
	if agent.IsProcessing {
		t.Error("Expected IsProcessing=false for Claude agent with idle tmux pane")
	}
	if agent.Status != "dead" {
		t.Errorf("Expected Status dead, got %s", agent.Status)
	}
}
