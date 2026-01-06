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

// TestClient_Show_ArrayFormat tests the RPC client's Show method with array response.
// Some beads daemon versions return arrays from bd show --json.
// The client should handle both array and single object formats.
func TestClient_Show_ArrayFormat(t *testing.T) {
	childIssue := Issue{
		ID:        "test-array-123",
		Title:     "Array Format Issue",
		Status:    "open",
		Priority:  1,
		IssueType: "task",
	}

	socketPath, cleanup := mockDaemon(t, func(conn net.Conn) {
		defer conn.Close()

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
				// Return array format (like bd show --json CLI does)
				issues := []Issue{childIssue}
				issueData, _ := json.Marshal(issues)
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

	issue, err := c.Show("test-array-123")
	if err != nil {
		t.Fatalf("Show failed: %v", err)
	}

	if issue.ID != "test-array-123" {
		t.Errorf("Issue.ID = %q, want %q", issue.ID, "test-array-123")
	}
	if issue.Title != "Array Format Issue" {
		t.Errorf("Issue.Title = %q, want %q", issue.Title, "Array Format Issue")
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

// TestWithAutoReconnect tests the WithAutoReconnect option.
func TestWithAutoReconnect(t *testing.T) {
	c := NewClient("/path/to/bd.sock", WithAutoReconnect(3))
	if !c.autoReconnect {
		t.Error("autoReconnect should be true")
	}
	if c.maxRetries != 3 {
		t.Errorf("maxRetries = %d, want %d", c.maxRetries, 3)
	}
}

// TestIsConnectionError tests the isConnectionError function.
func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{nil, false},
		{fmt.Errorf("some random error"), false},
		{fmt.Errorf("connection reset by peer"), true},
		{fmt.Errorf("broken pipe"), true},
		{fmt.Errorf("connection refused"), true},
		{fmt.Errorf("failed to read response: EOF"), true},
		{fmt.Errorf("failed to write request: broken pipe"), true},
		{fmt.Errorf("i/o timeout"), true},
	}

	for _, tt := range tests {
		result := isConnectionError(tt.err)
		if result != tt.expected {
			t.Errorf("isConnectionError(%v) = %v, want %v", tt.err, result, tt.expected)
		}
	}
}

// TestClient_AutoReconnect tests that autoReconnect enables lazy connection.
func TestClient_AutoReconnect(t *testing.T) {
	// Create a mock daemon
	socketPath, cleanup := mockDaemon(t, func(conn net.Conn) {
		defer conn.Close()

		// Handle multiple requests
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
			case OpPing:
				resp = Response{
					Success: true,
					Data:    json.RawMessage(`{"message":"pong"}`),
				}
			default:
				resp = Response{
					Success: false,
					Error:   "unknown operation",
				}
			}

			respJSON, _ := json.Marshal(resp)
			conn.Write(append(respJSON, '\n'))
		}
	})
	defer cleanup()

	// Create client with autoReconnect but don't explicitly connect
	c := NewClient(socketPath, WithAutoReconnect(3))

	// Client should not be connected yet
	if c.IsConnected() {
		t.Error("client should not be connected before first operation")
	}

	// Ping should auto-connect
	if err := c.Ping(); err != nil {
		t.Errorf("Ping failed: %v", err)
	}

	// Client should now be connected
	if !c.IsConnected() {
		t.Error("client should be connected after successful operation")
	}

	c.Close()
}

// TestUpdateArgs_JSON tests JSON marshaling of UpdateArgs.
func TestUpdateArgs_JSON(t *testing.T) {
	title := "New Title"
	priority := 2
	args := UpdateArgs{
		ID:        "issue-123",
		Title:     &title,
		Priority:  &priority,
		AddLabels: []string{"urgent"},
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("failed to marshal UpdateArgs: %v", err)
	}

	var decoded UpdateArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal UpdateArgs: %v", err)
	}

	if decoded.ID != "issue-123" {
		t.Errorf("ID = %q, want %q", decoded.ID, "issue-123")
	}
	if decoded.Title == nil || *decoded.Title != "New Title" {
		t.Errorf("Title = %v, want %q", decoded.Title, "New Title")
	}
	if decoded.Priority == nil || *decoded.Priority != 2 {
		t.Errorf("Priority = %v, want %d", decoded.Priority, 2)
	}
	if len(decoded.AddLabels) != 1 || decoded.AddLabels[0] != "urgent" {
		t.Errorf("AddLabels = %v, want %v", decoded.AddLabels, []string{"urgent"})
	}
}

// TestDeleteArgs_JSON tests JSON marshaling of DeleteArgs.
func TestDeleteArgs_JSON(t *testing.T) {
	args := DeleteArgs{
		IDs:    []string{"issue-1", "issue-2"},
		Force:  true,
		Reason: "cleanup",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("failed to marshal DeleteArgs: %v", err)
	}

	var decoded DeleteArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal DeleteArgs: %v", err)
	}

	if len(decoded.IDs) != 2 {
		t.Errorf("IDs count = %d, want %d", len(decoded.IDs), 2)
	}
	if !decoded.Force {
		t.Error("Force should be true")
	}
	if decoded.Reason != "cleanup" {
		t.Errorf("Reason = %q, want %q", decoded.Reason, "cleanup")
	}
}

// TestStaleArgs_JSON tests JSON marshaling of StaleArgs.
func TestStaleArgs_JSON(t *testing.T) {
	args := StaleArgs{
		Days:   30,
		Status: "open",
		Limit:  10,
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("failed to marshal StaleArgs: %v", err)
	}

	var decoded StaleArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal StaleArgs: %v", err)
	}

	if decoded.Days != 30 {
		t.Errorf("Days = %d, want %d", decoded.Days, 30)
	}
	if decoded.Status != "open" {
		t.Errorf("Status = %q, want %q", decoded.Status, "open")
	}
}

// TestLabelArgs_JSON tests JSON marshaling of label operation args.
func TestLabelArgs_JSON(t *testing.T) {
	addArgs := LabelAddArgs{ID: "issue-1", Label: "urgent"}
	removeArgs := LabelRemoveArgs{ID: "issue-2", Label: "low-priority"}

	data, err := json.Marshal(addArgs)
	if err != nil {
		t.Fatalf("failed to marshal LabelAddArgs: %v", err)
	}
	var decodedAdd LabelAddArgs
	if err := json.Unmarshal(data, &decodedAdd); err != nil {
		t.Fatalf("failed to unmarshal LabelAddArgs: %v", err)
	}
	if decodedAdd.ID != "issue-1" || decodedAdd.Label != "urgent" {
		t.Errorf("LabelAddArgs = %+v, want ID=issue-1 Label=urgent", decodedAdd)
	}

	data, err = json.Marshal(removeArgs)
	if err != nil {
		t.Fatalf("failed to marshal LabelRemoveArgs: %v", err)
	}
	var decodedRemove LabelRemoveArgs
	if err := json.Unmarshal(data, &decodedRemove); err != nil {
		t.Fatalf("failed to unmarshal LabelRemoveArgs: %v", err)
	}
	if decodedRemove.ID != "issue-2" || decodedRemove.Label != "low-priority" {
		t.Errorf("LabelRemoveArgs = %+v, want ID=issue-2 Label=low-priority", decodedRemove)
	}
}

// TestDepArgs_JSON tests JSON marshaling of dependency operation args.
func TestDepArgs_JSON(t *testing.T) {
	addArgs := DepAddArgs{FromID: "child", ToID: "parent", DepType: "blocks"}
	removeArgs := DepRemoveArgs{FromID: "child", ToID: "parent"}

	data, err := json.Marshal(addArgs)
	if err != nil {
		t.Fatalf("failed to marshal DepAddArgs: %v", err)
	}
	var decodedAdd DepAddArgs
	if err := json.Unmarshal(data, &decodedAdd); err != nil {
		t.Fatalf("failed to unmarshal DepAddArgs: %v", err)
	}
	if decodedAdd.FromID != "child" || decodedAdd.ToID != "parent" || decodedAdd.DepType != "blocks" {
		t.Errorf("DepAddArgs = %+v, unexpected values", decodedAdd)
	}

	data, err = json.Marshal(removeArgs)
	if err != nil {
		t.Fatalf("failed to marshal DepRemoveArgs: %v", err)
	}
	var decodedRemove DepRemoveArgs
	if err := json.Unmarshal(data, &decodedRemove); err != nil {
		t.Fatalf("failed to unmarshal DepRemoveArgs: %v", err)
	}
	if decodedRemove.FromID != "child" || decodedRemove.ToID != "parent" {
		t.Errorf("DepRemoveArgs = %+v, unexpected values", decodedRemove)
	}
}

// TestCountArgs_JSON tests JSON marshaling of CountArgs.
func TestCountArgs_JSON(t *testing.T) {
	args := CountArgs{
		Status:  "open",
		GroupBy: "priority",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("failed to marshal CountArgs: %v", err)
	}

	var decoded CountArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal CountArgs: %v", err)
	}

	if decoded.Status != "open" {
		t.Errorf("Status = %q, want %q", decoded.Status, "open")
	}
	if decoded.GroupBy != "priority" {
		t.Errorf("GroupBy = %q, want %q", decoded.GroupBy, "priority")
	}
}

// TestStatusResponse_JSON tests JSON marshaling of StatusResponse.
func TestStatusResponse_JSON(t *testing.T) {
	status := StatusResponse{
		Version:       "1.0.0",
		WorkspacePath: "/path/to/workspace",
		DatabasePath:  "/path/to/db",
		SocketPath:    "/path/to/sock",
		PID:           12345,
		UptimeSeconds: 3600.5,
		AutoCommit:    true,
		LocalMode:     false,
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("failed to marshal StatusResponse: %v", err)
	}

	var decoded StatusResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal StatusResponse: %v", err)
	}

	if decoded.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", decoded.Version, "1.0.0")
	}
	if decoded.PID != 12345 {
		t.Errorf("PID = %d, want %d", decoded.PID, 12345)
	}
	if !decoded.AutoCommit {
		t.Error("AutoCommit should be true")
	}
}

// TestParseDependencies tests parsing dependencies from raw JSON.
func TestParseDependencies(t *testing.T) {
	tests := []struct {
		name        string
		deps        json.RawMessage
		wantCount   int
		wantBlocker string
	}{
		{
			name:      "nil dependencies",
			deps:      nil,
			wantCount: 0,
		},
		{
			name:      "empty array",
			deps:      json.RawMessage(`[]`),
			wantCount: 0,
		},
		{
			name:        "single blocking dependency",
			deps:        json.RawMessage(`[{"id":"dep-1","title":"Blocker","status":"open","dependency_type":"blocks"}]`),
			wantCount:   1,
			wantBlocker: "dep-1",
		},
		{
			name:      "closed dependency (not blocking)",
			deps:      json.RawMessage(`[{"id":"dep-2","title":"Done","status":"closed","dependency_type":"blocks"}]`),
			wantCount: 1,
		},
		{
			name:      "multiple dependencies",
			deps:      json.RawMessage(`[{"id":"dep-1","title":"Blocker","status":"open","dependency_type":"blocks"},{"id":"dep-2","title":"Done","status":"closed","dependency_type":"blocks"}]`),
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := Issue{
				ID:           "test-issue",
				Dependencies: tt.deps,
			}

			deps := issue.ParseDependencies()
			if len(deps) != tt.wantCount {
				t.Errorf("ParseDependencies() returned %d deps, want %d", len(deps), tt.wantCount)
			}
			if tt.wantBlocker != "" && len(deps) > 0 && deps[0].ID != tt.wantBlocker {
				t.Errorf("first dep ID = %q, want %q", deps[0].ID, tt.wantBlocker)
			}
		})
	}
}

// TestGetBlockingDependencies tests filtering for open/in_progress dependencies.
func TestGetBlockingDependencies(t *testing.T) {
	tests := []struct {
		name          string
		deps          json.RawMessage
		wantCount     int
		wantBlockerID string
	}{
		{
			name:      "nil dependencies",
			deps:      nil,
			wantCount: 0,
		},
		{
			name:      "all closed (no blockers)",
			deps:      json.RawMessage(`[{"id":"dep-1","title":"Done","status":"closed","dependency_type":"blocks"},{"id":"dep-2","title":"Also Done","status":"closed","dependency_type":"blocks"}]`),
			wantCount: 0,
		},
		{
			name:          "one open (blocking)",
			deps:          json.RawMessage(`[{"id":"dep-1","title":"Blocker","status":"open","dependency_type":"blocks"}]`),
			wantCount:     1,
			wantBlockerID: "dep-1",
		},
		{
			name:          "one in_progress (blocking)",
			deps:          json.RawMessage(`[{"id":"dep-1","title":"In Progress","status":"in_progress","dependency_type":"blocks"}]`),
			wantCount:     1,
			wantBlockerID: "dep-1",
		},
		{
			name:          "mixed - one open, one closed",
			deps:          json.RawMessage(`[{"id":"dep-1","title":"Open","status":"open","dependency_type":"blocks"},{"id":"dep-2","title":"Closed","status":"closed","dependency_type":"blocks"}]`),
			wantCount:     1,
			wantBlockerID: "dep-1",
		},
		{
			name:      "all statuses - two blocking",
			deps:      json.RawMessage(`[{"id":"dep-1","title":"Open","status":"open","dependency_type":"blocks"},{"id":"dep-2","title":"In Progress","status":"in_progress","dependency_type":"blocks"},{"id":"dep-3","title":"Closed","status":"closed","dependency_type":"blocks"}]`),
			wantCount: 2,
		},
		// Parent-child dependency tests
		{
			name:          "parent-child: open parent blocks child",
			deps:          json.RawMessage(`[{"id":"epic-1","title":"Parent Epic","status":"open","dependency_type":"parent-child"}]`),
			wantCount:     1,
			wantBlockerID: "epic-1",
		},
		{
			name:      "parent-child: in_progress parent does NOT block child",
			deps:      json.RawMessage(`[{"id":"epic-1","title":"Parent Epic","status":"in_progress","dependency_type":"parent-child"}]`),
			wantCount: 0,
		},
		{
			name:      "parent-child: closed parent does NOT block child",
			deps:      json.RawMessage(`[{"id":"epic-1","title":"Parent Epic","status":"closed","dependency_type":"parent-child"}]`),
			wantCount: 0,
		},
		{
			name:          "mixed: blocks in_progress + parent-child in_progress",
			deps:          json.RawMessage(`[{"id":"dep-1","title":"Blocks Dep","status":"in_progress","dependency_type":"blocks"},{"id":"epic-1","title":"Parent Epic","status":"in_progress","dependency_type":"parent-child"}]`),
			wantCount:     1, // Only the "blocks" dep blocks, not parent-child
			wantBlockerID: "dep-1",
		},
		{
			name:          "mixed: blocks closed + parent-child open",
			deps:          json.RawMessage(`[{"id":"dep-1","title":"Blocks Dep","status":"closed","dependency_type":"blocks"},{"id":"epic-1","title":"Parent Epic","status":"open","dependency_type":"parent-child"}]`),
			wantCount:     1, // Only parent-child blocks (because open)
			wantBlockerID: "epic-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := Issue{
				ID:           "test-issue",
				Dependencies: tt.deps,
			}

			blockers := issue.GetBlockingDependencies()
			if len(blockers) != tt.wantCount {
				t.Errorf("GetBlockingDependencies() returned %d blockers, want %d", len(blockers), tt.wantCount)
			}
			if tt.wantBlockerID != "" && len(blockers) > 0 && blockers[0].ID != tt.wantBlockerID {
				t.Errorf("first blocker ID = %q, want %q", blockers[0].ID, tt.wantBlockerID)
			}
		})
	}
}

// TestBlockingDependencyError tests the error message formatting.
func TestBlockingDependencyError(t *testing.T) {
	err := &BlockingDependencyError{
		IssueID: "proj-xyz",
		Blockers: []BlockingDependency{
			{ID: "proj-abc", Title: "First blocker", Status: "open"},
			{ID: "proj-def", Title: "Second blocker", Status: "in_progress"},
		},
		ForceMessage: "Use --force to override",
	}

	errStr := err.Error()

	// Check that the error message contains expected parts
	if !containsString(errStr, "proj-xyz") {
		t.Errorf("error message should contain issue ID: %s", errStr)
	}
	if !containsString(errStr, "proj-abc (open)") {
		t.Errorf("error message should contain first blocker: %s", errStr)
	}
	if !containsString(errStr, "proj-def (in_progress)") {
		t.Errorf("error message should contain second blocker: %s", errStr)
	}
	if !containsString(errStr, "Use --force to override") {
		t.Errorf("error message should contain force message: %s", errStr)
	}
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
