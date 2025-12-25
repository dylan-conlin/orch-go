package beads

import (
	"encoding/json"
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
