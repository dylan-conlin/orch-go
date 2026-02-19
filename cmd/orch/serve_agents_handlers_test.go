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
	if globalBeadsCache == nil {
		globalBeadsCache = newBeadsCache()
	}

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

	if _, ok := findAgentByBeadsID(agents, "orch-go-bbb2"); ok {
		t.Fatal("Expected secondary agent without workspace/session/label to be filtered out")
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

func TestHandleAgentsOpenStatusIssueVisible(t *testing.T) {
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

	listOpenIssues = func() (map[string]*verify.Issue, error) {
		return nil, fmt.Errorf("unexpected call")
	}
	listOpenIssuesWithDir = func(dir string) (map[string]*verify.Issue, error) {
		return map[string]*verify.Issue{
			"orch-go-open1": {ID: "orch-go-open1", Title: "Auto-created task", Status: "open"},
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

func TestHandleAgentsReviewLabelWithoutWorkspace(t *testing.T) {
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

	server := newTestOpenCodeServer(t, map[string]opencode.Session{}, map[string][]opencode.Message{})
	serverURL = server.URL
	defer server.Close()

	listOpenIssues = func() (map[string]*verify.Issue, error) {
		return nil, fmt.Errorf("unexpected call")
	}
	listOpenIssuesWithDir = func(dir string) (map[string]*verify.Issue, error) {
		return map[string]*verify.Issue{
			"orch-go-review": {ID: "orch-go-review", Title: "Needs review", Status: "in_progress", Labels: []string{"daemon:ready-review"}},
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
