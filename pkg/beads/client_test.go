package beads

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindSocketPath(t *testing.T) {
	// Create a temp directory structure
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("failed to create beads dir: %v", err)
	}

	// Create a fake socket file
	socketPath := filepath.Join(beadsDir, "bd.sock")
	if err := os.WriteFile(socketPath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create socket file: %v", err)
	}

	// Test finding socket from the root directory
	found, err := FindSocketPath(tmpDir)
	if err != nil {
		t.Errorf("FindSocketPath failed: %v", err)
	}
	if found != socketPath {
		t.Errorf("FindSocketPath = %q, want %q", found, socketPath)
	}

	// Test finding socket from a subdirectory
	subDir := filepath.Join(tmpDir, "subdir", "deep")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	found, err = FindSocketPath(subDir)
	if err != nil {
		t.Errorf("FindSocketPath from subdir failed: %v", err)
	}
	if found != socketPath {
		t.Errorf("FindSocketPath from subdir = %q, want %q", found, socketPath)
	}
}

func TestFindSocketPath_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := FindSocketPath(tmpDir)
	if err == nil {
		t.Error("FindSocketPath should fail when no socket exists")
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient("/path/to/bd.sock")
	if c.socketPath != "/path/to/bd.sock" {
		t.Errorf("socketPath = %q, want %q", c.socketPath, "/path/to/bd.sock")
	}
	if c.timeout != 30*time.Second {
		t.Errorf("timeout = %v, want %v", c.timeout, 30*time.Second)
	}
}

func TestNewClient_WithOptions(t *testing.T) {
	c := NewClient("/path/to/bd.sock",
		WithTimeout(10*time.Second),
		WithCwd("/some/dir"))

	if c.timeout != 10*time.Second {
		t.Errorf("timeout = %v, want %v", c.timeout, 10*time.Second)
	}
	if c.cwd != "/some/dir" {
		t.Errorf("cwd = %q, want %q", c.cwd, "/some/dir")
	}
}

func TestConnect_SocketNotFound(t *testing.T) {
	c := NewClient("/nonexistent/path/bd.sock")
	err := c.Connect()
	if err == nil {
		t.Error("Connect should fail when socket doesn't exist")
	}
}

func TestIsConnected(t *testing.T) {
	c := NewClient("/path/to/bd.sock")
	if c.IsConnected() {
		t.Error("IsConnected should be false before Connect")
	}
}

// mockDaemon creates a mock Unix socket daemon for testing.
// Returns the socket path and a cleanup function.
func mockDaemon(t *testing.T, handler func(conn net.Conn)) (string, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "bd.sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}

	done := make(chan struct{})
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					continue
				}
			}
			go handler(conn)
		}
	}()

	return socketPath, func() {
		close(done)
		listener.Close()
	}
}

func TestClient_Health(t *testing.T) {
	socketPath, cleanup := mockDaemon(t, func(conn net.Conn) {
		defer conn.Close()

		// Read the request
		buf := make([]byte, 4096)
		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		var req Request
		if err := json.Unmarshal(buf[:n], &req); err != nil {
			return
		}

		if req.Operation != OpHealth {
			return
		}

		// Send health response
		healthData, _ := json.Marshal(HealthResponse{
			Status:  "healthy",
			Version: "1.0.0",
			Uptime:  123.45,
		})
		resp := Response{
			Success: true,
			Data:    healthData,
		}
		respJSON, _ := json.Marshal(resp)
		conn.Write(append(respJSON, '\n'))
	})
	defer cleanup()

	c := NewClient(socketPath)
	if err := c.Connect(); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer c.Close()

	if !c.IsConnected() {
		t.Error("IsConnected should be true after successful Connect")
	}
}

func TestClient_Close(t *testing.T) {
	socketPath, cleanup := mockDaemon(t, func(conn net.Conn) {
		defer conn.Close()

		// Read the health check request
		buf := make([]byte, 4096)
		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		var req Request
		if err := json.Unmarshal(buf[:n], &req); err != nil {
			return
		}

		// Always respond with healthy
		healthData, _ := json.Marshal(HealthResponse{
			Status:  "healthy",
			Version: "1.0.0",
			Uptime:  1.0,
		})
		resp := Response{
			Success: true,
			Data:    healthData,
		}
		respJSON, _ := json.Marshal(resp)
		conn.Write(append(respJSON, '\n'))
	})
	defer cleanup()

	c := NewClient(socketPath)
	if err := c.Connect(); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if !c.IsConnected() {
		t.Error("IsConnected should be true after Connect")
	}

	if err := c.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	if c.IsConnected() {
		t.Error("IsConnected should be false after Close")
	}

	// Close again should not error
	if err := c.Close(); err != nil {
		t.Errorf("second Close should not error: %v", err)
	}
}

func TestRequestResponse_JSON(t *testing.T) {
	req := Request{
		Operation:     OpReady,
		Args:          json.RawMessage(`{"limit":10}`),
		ClientVersion: "0.1.0",
		Cwd:           "/some/path",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	var decoded Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal request: %v", err)
	}

	if decoded.Operation != OpReady {
		t.Errorf("Operation = %q, want %q", decoded.Operation, OpReady)
	}
	if decoded.ClientVersion != "0.1.0" {
		t.Errorf("ClientVersion = %q, want %q", decoded.ClientVersion, "0.1.0")
	}
}

func TestIssue_JSON(t *testing.T) {
	issue := Issue{
		ID:        "test-123",
		Title:     "Test Issue",
		Status:    "open",
		Priority:  1,
		IssueType: "task",
		Labels:    []string{"bug", "urgent"},
	}

	data, err := json.Marshal(issue)
	if err != nil {
		t.Fatalf("failed to marshal issue: %v", err)
	}

	var decoded Issue
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal issue: %v", err)
	}

	if decoded.ID != "test-123" {
		t.Errorf("ID = %q, want %q", decoded.ID, "test-123")
	}
	if len(decoded.Labels) != 2 {
		t.Errorf("Labels count = %d, want %d", len(decoded.Labels), 2)
	}
}

func TestComment_JSON(t *testing.T) {
	comment := Comment{
		ID:        1,
		IssueID:   "issue-1",
		Author:    "agent",
		Text:      "Phase: Complete",
		CreatedAt: "2025-01-01T00:00:00Z",
	}

	data, err := json.Marshal(comment)
	if err != nil {
		t.Fatalf("failed to marshal comment: %v", err)
	}

	var decoded Comment
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal comment: %v", err)
	}

	if decoded.Author != "agent" {
		t.Errorf("Author = %q, want %q", decoded.Author, "agent")
	}
	if decoded.Text != "Phase: Complete" {
		t.Errorf("Text = %q, want %q", decoded.Text, "Phase: Complete")
	}
}

func TestStats_JSON(t *testing.T) {
	stats := Stats{
		Summary: StatsSummary{
			TotalIssues:      100,
			OpenIssues:       80,
			ClosedIssues:     20,
			BlockedIssues:    5,
			ReadyIssues:      10,
			InProgressIssues: 5,
		},
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("failed to marshal stats: %v", err)
	}

	var decoded Stats
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal stats: %v", err)
	}

	if decoded.Summary.TotalIssues != 100 {
		t.Errorf("TotalIssues = %d, want %d", decoded.Summary.TotalIssues, 100)
	}
	if decoded.Summary.ReadyIssues != 10 {
		t.Errorf("ReadyIssues = %d, want %d", decoded.Summary.ReadyIssues, 10)
	}
}

func TestCreateArgs_JSON(t *testing.T) {
	args := CreateArgs{
		Title:       "New Issue",
		Description: "Description here",
		IssueType:   "feature",
		Priority:    2,
		Labels:      []string{"enhancement"},
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("failed to marshal CreateArgs: %v", err)
	}

	var decoded CreateArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal CreateArgs: %v", err)
	}

	if decoded.Title != "New Issue" {
		t.Errorf("Title = %q, want %q", decoded.Title, "New Issue")
	}
	if decoded.Priority != 2 {
		t.Errorf("Priority = %d, want %d", decoded.Priority, 2)
	}
}

func TestCloseArgs_JSON(t *testing.T) {
	args := CloseArgs{
		ID:     "issue-123",
		Reason: "Fixed in commit abc123",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("failed to marshal CloseArgs: %v", err)
	}

	var decoded CloseArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal CloseArgs: %v", err)
	}

	if decoded.ID != "issue-123" {
		t.Errorf("ID = %q, want %q", decoded.ID, "issue-123")
	}
	if decoded.Reason != "Fixed in commit abc123" {
		t.Errorf("Reason = %q, want %q", decoded.Reason, "Fixed in commit abc123")
	}
}

func TestBdShowArrayFormat(t *testing.T) {
	// bd show --json always returns an array, even for a single issue
	// This test verifies we can correctly parse the array format

	// Test case 1: Single issue (most common case)
	singleIssueJSON := `[
  {
    "id": "test-abc",
    "title": "Test Issue",
    "status": "open",
    "priority": 0,
    "issue_type": "task"
  }
]`
	var issues []Issue
	if err := json.Unmarshal([]byte(singleIssueJSON), &issues); err != nil {
		t.Fatalf("failed to unmarshal single issue array: %v", err)
	}
	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].ID != "test-abc" {
		t.Errorf("ID = %q, want %q", issues[0].ID, "test-abc")
	}

	// Test case 2: Epic child with dependencies (parent) - includes extra fields
	epicChildJSON := `[
  {
    "id": "proj-ph1.9",
    "title": "Go CLI: Project scaffolding",
    "status": "closed",
    "priority": 1,
    "issue_type": "task",
    "dependencies": [
      {
        "id": "proj-ph1",
        "title": "Epic: Parent Epic",
        "status": "closed",
        "priority": 1,
        "issue_type": "epic",
        "dependency_type": "parent-child"
      }
    ]
  }
]`
	var childIssues []Issue
	if err := json.Unmarshal([]byte(epicChildJSON), &childIssues); err != nil {
		t.Fatalf("failed to unmarshal epic child array: %v", err)
	}
	if len(childIssues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(childIssues))
	}
	if childIssues[0].ID != "proj-ph1.9" {
		t.Errorf("ID = %q, want %q", childIssues[0].ID, "proj-ph1.9")
	}
	// Note: The dependencies field is parsed as json.RawMessage
	// This allows accepting both string arrays and nested Issue objects
}

// TestChildIDPatterns tests parsing of child ID patterns (dot notation)
// such as "proj-ph1.9" which are epic children. These IDs contain dots
// which distinguishes them from regular IDs.
func TestChildIDPatterns(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		wantID   string
		hasDepth int // expected depth based on dot count
	}{
		{
			name:     "simple ID",
			id:       "proj-abc",
			wantID:   "proj-abc",
			hasDepth: 0,
		},
		{
			name:     "child ID level 1",
			id:       "proj-ph1.1",
			wantID:   "proj-ph1.1",
			hasDepth: 1,
		},
		{
			name:     "child ID level 1 double digit",
			id:       "proj-ph1.12",
			wantID:   "proj-ph1.12",
			hasDepth: 1,
		},
		{
			name:     "grandchild ID level 2",
			id:       "proj-ph1.1.1",
			wantID:   "proj-ph1.1.1",
			hasDepth: 2,
		},
		{
			name:     "complex prefix with child",
			id:       "orch-go-re8n.3",
			wantID:   "orch-go-re8n.3",
			hasDepth: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that Issue struct correctly parses various ID patterns
			issueJSON := fmt.Sprintf(`{
				"id": %q,
				"title": "Test",
				"status": "open",
				"priority": 0,
				"issue_type": "task"
			}`, tt.id)

			var issue Issue
			if err := json.Unmarshal([]byte(issueJSON), &issue); err != nil {
				t.Fatalf("failed to unmarshal issue with id %q: %v", tt.id, err)
			}

			if issue.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", issue.ID, tt.wantID)
			}
		})
	}
}

// TestDependenciesParsingFormats tests that the Dependencies field
// correctly handles different JSON formats returned by bd CLI.
func TestDependenciesParsingFormats(t *testing.T) {
	tests := []struct {
		name         string
		json         string
		wantID       string
		wantParseErr bool
	}{
		{
			name: "no dependencies",
			json: `{
				"id": "test-1",
				"title": "Test",
				"status": "open",
				"priority": 0,
				"issue_type": "task"
			}`,
			wantID: "test-1",
		},
		{
			name: "empty dependencies array",
			json: `{
				"id": "test-2",
				"title": "Test",
				"status": "open",
				"priority": 0,
				"issue_type": "task",
				"dependencies": []
			}`,
			wantID: "test-2",
		},
		{
			name: "string ID dependencies (legacy format)",
			json: `{
				"id": "test-3",
				"title": "Test",
				"status": "open",
				"priority": 0,
				"issue_type": "task",
				"dependencies": ["dep-1", "dep-2"]
			}`,
			wantID: "test-3",
		},
		{
			name: "nested Issue object dependencies (bd show format)",
			json: `{
				"id": "proj-ph1.9",
				"title": "Child Issue",
				"status": "open",
				"priority": 1,
				"issue_type": "task",
				"dependencies": [
					{
						"id": "proj-ph1",
						"title": "Parent Epic",
						"status": "closed",
						"priority": 1,
						"issue_type": "epic",
						"dependency_type": "parent-child"
					}
				]
			}`,
			wantID: "proj-ph1.9",
		},
		{
			name: "mixed dependencies format",
			json: `{
				"id": "proj-ph1.5",
				"title": "Mixed Deps",
				"status": "open",
				"priority": 2,
				"issue_type": "task",
				"dependencies": [
					{
						"id": "proj-ph1",
						"title": "Parent",
						"status": "open",
						"priority": 1,
						"issue_type": "epic",
						"dependency_type": "parent-child"
					},
					{
						"id": "proj-other",
						"title": "Blocker",
						"status": "closed",
						"priority": 1,
						"issue_type": "task",
						"dependency_type": "blocks"
					}
				]
			}`,
			wantID: "proj-ph1.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var issue Issue
			err := json.Unmarshal([]byte(tt.json), &issue)

			if tt.wantParseErr {
				if err == nil {
					t.Errorf("expected parse error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			if issue.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", issue.ID, tt.wantID)
			}
		})
	}
}

// TestClient_Show_ChildID tests the RPC client's Show method with child IDs.
// This uses a mock daemon to verify the client correctly handles child ID responses.
func TestClient_Show_ChildID(t *testing.T) {
	childIssue := Issue{
		ID:        "proj-epic.3",
		Title:     "Epic Child Task",
		Status:    "open",
		Priority:  1,
		IssueType: "task",
		// Dependencies field would be set by bd show but we test without it
	}

	socketPath, cleanup := mockDaemon(t, func(conn net.Conn) {
		defer conn.Close()

		// This handler runs in a loop to handle multiple requests on the same connection
		// The client uses a single connection for health check + subsequent operations
		for {
			buf := make([]byte, 4096)
			n, err := conn.Read(buf)
			if err != nil {
				return
			}

			var req Request
			if err := json.Unmarshal(buf[:n], &req); err != nil {
				return
			}

			var resp Response

			switch req.Operation {
			case OpHealth:
				healthData, _ := json.Marshal(HealthResponse{
					Status:  "healthy",
					Version: "1.0.0",
					Uptime:  1.0,
				})
				resp = Response{
					Success: true,
					Data:    healthData,
				}

			case OpShow:
				var args ShowArgs
				if err := json.Unmarshal(req.Args, &args); err != nil {
					return
				}

				// Verify we received the child ID correctly
				if args.ID != "proj-epic.3" {
					t.Errorf("Show received ID = %q, want %q", args.ID, "proj-epic.3")
				}

				issueData, _ := json.Marshal(childIssue)
				resp = Response{
					Success: true,
					Data:    issueData,
				}

			default:
				resp = Response{
					Success: false,
					Error:   fmt.Sprintf("unknown operation: %s", req.Operation),
				}
			}

			respJSON, _ := json.Marshal(resp)
			conn.Write(append(respJSON, '\n'))
		}
	})
	defer cleanup()

	c := NewClient(socketPath)
	if err := c.Connect(); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer c.Close()

	issue, err := c.Show("proj-epic.3")
	if err != nil {
		t.Fatalf("Show failed: %v", err)
	}

	if issue.ID != "proj-epic.3" {
		t.Errorf("Issue.ID = %q, want %q", issue.ID, "proj-epic.3")
	}
	if issue.Title != "Epic Child Task" {
		t.Errorf("Issue.Title = %q, want %q", issue.Title, "Epic Child Task")
	}
}

// TestEpicChildWithParentDependency tests parsing a complete epic child
// response with parent dependency as returned by bd show --json.
func TestEpicChildWithParentDependency(t *testing.T) {
	// This is the actual format returned by `bd show child-id --json`
	jsonResponse := `[
		{
			"id": "orch-go-re8n.1",
			"title": "Implement feature X",
			"description": "First child of epic",
			"status": "in_progress",
			"priority": 1,
			"issue_type": "task",
			"labels": ["skill:feature-impl", "triage:ready"],
			"dependencies": [
				{
					"id": "orch-go-re8n",
					"title": "Epic: Major Feature",
					"status": "open",
					"priority": 1,
					"issue_type": "epic",
					"dependency_type": "parent-child"
				}
			],
			"created_at": "2025-12-25T10:00:00Z",
			"updated_at": "2025-12-25T12:00:00Z"
		}
	]`

	var issues []Issue
	if err := json.Unmarshal([]byte(jsonResponse), &issues); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]

	// Verify all fields are correctly parsed
	if issue.ID != "orch-go-re8n.1" {
		t.Errorf("ID = %q, want %q", issue.ID, "orch-go-re8n.1")
	}
	if issue.Title != "Implement feature X" {
		t.Errorf("Title = %q, want %q", issue.Title, "Implement feature X")
	}
	if issue.Status != "in_progress" {
		t.Errorf("Status = %q, want %q", issue.Status, "in_progress")
	}
	if issue.Priority != 1 {
		t.Errorf("Priority = %d, want %d", issue.Priority, 1)
	}
	if issue.IssueType != "task" {
		t.Errorf("IssueType = %q, want %q", issue.IssueType, "task")
	}
	if len(issue.Labels) != 2 {
		t.Errorf("Labels count = %d, want 2", len(issue.Labels))
	}
	if issue.CreatedAt != "2025-12-25T10:00:00Z" {
		t.Errorf("CreatedAt = %q, want %q", issue.CreatedAt, "2025-12-25T10:00:00Z")
	}

	// Dependencies is json.RawMessage - verify it's not nil
	if issue.Dependencies == nil {
		t.Error("Dependencies should not be nil")
	}

	// Verify the raw dependencies can be parsed if needed
	var deps []map[string]interface{}
	if err := json.Unmarshal(issue.Dependencies, &deps); err != nil {
		t.Fatalf("failed to parse dependencies: %v", err)
	}
	if len(deps) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(deps))
	}
	if deps[0]["id"] != "orch-go-re8n" {
		t.Errorf("dependency id = %v, want %q", deps[0]["id"], "orch-go-re8n")
	}
	if deps[0]["dependency_type"] != "parent-child" {
		t.Errorf("dependency_type = %v, want %q", deps[0]["dependency_type"], "parent-child")
	}
}

// TestMultiLevelChildID tests deeply nested child IDs (grandchildren).
func TestMultiLevelChildID(t *testing.T) {
	jsonResponse := `[
		{
			"id": "proj-epic.1.2",
			"title": "Grandchild Task",
			"status": "open",
			"priority": 2,
			"issue_type": "task",
			"dependencies": [
				{
					"id": "proj-epic.1",
					"title": "Child Epic",
					"status": "open",
					"priority": 1,
					"issue_type": "epic",
					"dependency_type": "parent-child"
				}
			]
		}
	]`

	var issues []Issue
	if err := json.Unmarshal([]byte(jsonResponse), &issues); err != nil {
		t.Fatalf("failed to unmarshal grandchild: %v", err)
	}

	if issues[0].ID != "proj-epic.1.2" {
		t.Errorf("ID = %q, want %q", issues[0].ID, "proj-epic.1.2")
	}
}
