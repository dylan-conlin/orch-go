package opencode

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestListSessions tests the ListSessions API call.
func TestListSessions(t *testing.T) {
	// Create mock server with session list response
	mockSessions := `[
		{"id":"ses_abc123","title":"Test Session 1","directory":"/home/user/project1","time":{"created":1766200000000,"updated":1766200010000},"summary":{"additions":10,"deletions":5,"files":2}},
		{"id":"ses_xyz789","title":"Test Session 2","directory":"/home/user/project2","time":{"created":1766199000000,"updated":1766199010000},"summary":{"additions":20,"deletions":10,"files":4}}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/session" {
			t.Errorf("Expected path /session, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockSessions))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	sessions, err := client.ListSessions("")
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("Expected 2 sessions, got %d", len(sessions))
	}

	// Verify first session
	if sessions[0].ID != "ses_abc123" {
		t.Errorf("sessions[0].ID = %s, want ses_abc123", sessions[0].ID)
	}
	if sessions[0].Title != "Test Session 1" {
		t.Errorf("sessions[0].Title = %s, want Test Session 1", sessions[0].Title)
	}
	if sessions[0].Directory != "/home/user/project1" {
		t.Errorf("sessions[0].Directory = %s, want /home/user/project1", sessions[0].Directory)
	}

	// Verify second session
	if sessions[1].ID != "ses_xyz789" {
		t.Errorf("sessions[1].ID = %s, want ses_xyz789", sessions[1].ID)
	}
}

// TestFindRecentSession tests the FindRecentSession method.
// It matches by directory and creation time only (within 30 seconds).
func TestFindRecentSession(t *testing.T) {
	projectDir := "/home/user/project1"
	nowMs := time.Now().UnixMilli()
	// Old session (more than 30 seconds ago)
	oldMs := nowMs - 60*1000
	// New session (just created)
	newMs := nowMs - 1000 // 1 second ago
	// Other project session (even newer, but different directory)
	otherMs := nowMs - 500

	mockSessions := fmt.Sprintf(`[
		{"id":"ses_old","directory":"/home/user/project1","time":{"created":%d}},
		{"id":"ses_new","directory":"/home/user/project1","time":{"created":%d}},
		{"id":"ses_other","directory":"/home/user/other","time":{"created":%d}}
	]`, oldMs, newMs, otherMs)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/session" {
			t.Errorf("Expected path /session, got %s", r.URL.Path)
		}
		if r.Header.Get("x-opencode-directory") != projectDir {
			t.Errorf("Expected header x-opencode-directory: %s, got %s", projectDir, r.Header.Get("x-opencode-directory"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockSessions))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	sessionID, err := client.FindRecentSession(projectDir)
	if err != nil {
		t.Fatalf("FindRecentSession() error = %v", err)
	}

	if sessionID != "ses_new" {
		t.Errorf("sessionID = %s, want ses_new", sessionID)
	}
}

// TestFindRecentSessionWithRetry tests the retry logic for session discovery.
func TestFindRecentSessionWithRetry(t *testing.T) {
	projectDir := "/home/user/project1"

	t.Run("succeeds on first attempt", func(t *testing.T) {
		callCount := 0
		nowMs := time.Now().UnixMilli()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.Header().Set("Content-Type", "application/json")
			// Return a session created "now" so it's within the 30s window
			w.Write([]byte(fmt.Sprintf(`[{"id":"ses_found","directory":"%s","time":{"created":%d}}]`, projectDir, nowMs)))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		sessionID, err := client.FindRecentSessionWithRetry(projectDir, 3, 10*time.Millisecond)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if sessionID != "ses_found" {
			t.Errorf("sessionID = %s, want ses_found", sessionID)
		}
		if callCount != 1 {
			t.Errorf("callCount = %d, want 1", callCount)
		}
	})

	t.Run("succeeds on second attempt", func(t *testing.T) {
		callCount := 0
		nowMs := time.Now().UnixMilli()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.Header().Set("Content-Type", "application/json")
			if callCount == 1 {
				// First call returns empty
				w.Write([]byte(`[]`))
			} else {
				// Second call returns the session
				w.Write([]byte(fmt.Sprintf(`[{"id":"ses_found","directory":"%s","time":{"created":%d}}]`, projectDir, nowMs)))
			}
		}))
		defer server.Close()

		client := NewClient(server.URL)
		sessionID, err := client.FindRecentSessionWithRetry(projectDir, 3, 10*time.Millisecond)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if sessionID != "ses_found" {
			t.Errorf("sessionID = %s, want ses_found", sessionID)
		}
		if callCount != 2 {
			t.Errorf("callCount = %d, want 2", callCount)
		}
	})

	t.Run("returns error after max attempts", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.Header().Set("Content-Type", "application/json")
			// Always return empty
			w.Write([]byte(`[]`))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		sessionID, err := client.FindRecentSessionWithRetry(projectDir, 3, 10*time.Millisecond)
		if err == nil {
			t.Error("Expected error after max attempts")
		}
		if sessionID != "" {
			t.Errorf("sessionID = %s, want empty string", sessionID)
		}
		if callCount != 3 {
			t.Errorf("callCount = %d, want 3", callCount)
		}
	})
}

// TestListSessionsEmpty tests ListSessions with empty response.
func TestListSessionsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	sessions, err := client.ListSessions("")
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(sessions))
	}
}

// TestListSessionsError tests ListSessions with server error.
func TestListSessionsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.ListSessions("")
	if err == nil {
		t.Error("Expected error for server error response")
	}
}

// TestListSessionsConnectionError tests ListSessions with connection error.
func TestListSessionsConnectionError(t *testing.T) {
	client := NewClient("http://127.0.0.1:99999") // Invalid port
	_, err := client.ListSessions("")
	if err == nil {
		t.Error("Expected error for connection failure")
	}
}

// TestListDiskSessions tests the ListDiskSessions API call.
func TestListDiskSessions(t *testing.T) {
	projectDir := "/Users/dylan/project"
	mockSessions := `[
		{"id":"ses_abc123","title":"Session 1","directory":"/Users/dylan/project","time":{"created":1766200000000,"updated":1766200010000}},
		{"id":"ses_xyz789","title":"Session 2","directory":"/Users/dylan/project","time":{"created":1766199000000,"updated":1766199010000}}
	]`

	var receivedHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/session" {
			t.Errorf("Expected path /session, got %s", r.URL.Path)
		}
		receivedHeader = r.Header.Get("x-opencode-directory")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockSessions))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	sessions, err := client.ListDiskSessions(projectDir)
	if err != nil {
		t.Fatalf("ListDiskSessions() error = %v", err)
	}

	// Verify header was sent
	if receivedHeader != projectDir {
		t.Errorf("Expected x-opencode-directory header %q, got %q", projectDir, receivedHeader)
	}

	// Verify sessions returned
	if len(sessions) != 2 {
		t.Fatalf("Expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].ID != "ses_abc123" {
		t.Errorf("sessions[0].ID = %s, want ses_abc123", sessions[0].ID)
	}
}

// TestListDiskSessionsRequiresDirectory tests that ListDiskSessions fails without directory.
func TestListDiskSessionsRequiresDirectory(t *testing.T) {
	client := NewClient("http://localhost:4096")
	_, err := client.ListDiskSessions("")
	if err == nil {
		t.Error("Expected error when directory is empty")
	}
	if !strings.Contains(err.Error(), "directory is required") {
		t.Errorf("Expected 'directory is required' error, got: %v", err)
	}
}

// TestListDiskSessionsEmpty tests ListDiskSessions with no sessions.
func TestListDiskSessionsEmpty(t *testing.T) {
	projectDir := "/Users/dylan/empty-project"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	sessions, err := client.ListDiskSessions(projectDir)
	if err != nil {
		t.Fatalf("ListDiskSessions() error = %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(sessions))
	}
}

// TestListDiskSessionsServerError tests ListDiskSessions with server error.
func TestListDiskSessionsServerError(t *testing.T) {
	projectDir := "/Users/dylan/project"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.ListDiskSessions(projectDir)
	if err == nil {
		t.Error("Expected error for server error response")
	}
}

// TestDeleteSession tests the DeleteSession API call.
func TestDeleteSession(t *testing.T) {
	sessionID := "ses_to_delete"
	deleted := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/session/"+sessionID {
			t.Errorf("Expected path /session/%s, got %s", sessionID, r.URL.Path)
		}
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.DeleteSession(sessionID)
	if err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}

	if !deleted {
		t.Error("Expected DELETE request to be made")
	}
}

// TestDeleteSessionOK tests DeleteSession with 200 OK response.
func TestDeleteSessionOK(t *testing.T) {
	sessionID := "ses_ok"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.DeleteSession(sessionID)
	if err != nil {
		t.Fatalf("DeleteSession() should accept 200 OK, got error: %v", err)
	}
}

// TestDeleteSessionNotFound tests DeleteSession when session doesn't exist.
func TestDeleteSessionNotFound(t *testing.T) {
	sessionID := "ses_notfound"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"session not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.DeleteSession(sessionID)
	if err == nil {
		t.Error("Expected error for 404 response")
	}
}

// TestDeleteSessionServerError tests DeleteSession with server error.
func TestDeleteSessionServerError(t *testing.T) {
	sessionID := "ses_error"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.DeleteSession(sessionID)
	if err == nil {
		t.Error("Expected error for server error response")
	}
}

// TestCreateSession tests the CreateSession API call.
func TestCreateSession(t *testing.T) {
	title := "test-session"
	directory := "/Users/dylan/project"
	model := "anthropic/claude-opus-4"

	var receivedRequest CreateSessionRequest
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/session" {
			t.Errorf("Expected path /session, got %s", r.URL.Path)
		}

		// Capture headers
		receivedHeaders = r.Header.Clone()

		// Decode request body
		if err := json.NewDecoder(r.Body).Decode(&receivedRequest); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Return success
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := CreateSessionResponse{
			ID:        "ses_test123",
			Title:     title,
			Directory: directory,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	variant := "high"
	resp, err := client.CreateSession(title, directory, model, variant, true)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Verify response
	if resp.ID != "ses_test123" {
		t.Errorf("resp.ID = %s, want ses_test123", resp.ID)
	}
	if resp.Title != title {
		t.Errorf("resp.Title = %s, want %s", resp.Title, title)
	}

	// Verify request included model parameter
	if receivedRequest.Title != title {
		t.Errorf("receivedRequest.Title = %s, want %s", receivedRequest.Title, title)
	}
	if receivedRequest.Directory != directory {
		t.Errorf("receivedRequest.Directory = %s, want %s", receivedRequest.Directory, directory)
	}
	if receivedRequest.Model != model {
		t.Errorf("receivedRequest.Model = %s, want %s", receivedRequest.Model, model)
	}

	// Verify ORCH_WORKER header is set for worker sessions
	if orchWorker := receivedHeaders.Get("x-opencode-env-ORCH_WORKER"); orchWorker != "1" {
		t.Errorf("x-opencode-env-ORCH_WORKER header = %q, want \"1\"", orchWorker)
	}
}

// TestCreateSessionWithoutModel tests CreateSession without model parameter.
func TestCreateSessionWithoutModel(t *testing.T) {
	title := "test-session"
	directory := "/Users/dylan/project"
	model := "" // Empty model

	var receivedRequest CreateSessionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedRequest); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := CreateSessionResponse{
			ID:        "ses_test456",
			Title:     title,
			Directory: directory,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	variant := ""
	_, err := client.CreateSession(title, directory, model, variant, false)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Verify empty model was sent (omitempty should exclude it from JSON)
	if receivedRequest.Model != "" {
		t.Errorf("receivedRequest.Model = %s, want empty string", receivedRequest.Model)
	}
}

// TestIsSessionProcessing tests the IsSessionProcessing method.
func TestIsSessionProcessing(t *testing.T) {
	sessionID := "ses_test123"
	nowMs := time.Now().UnixMilli()

	tests := []struct {
		name           string
		messages       string
		wantProcessing bool
	}{
		{
			name: "processing - assistant message with null finish",
			messages: fmt.Sprintf(`[
				{"info":{"id":"msg_1","sessionID":"%s","role":"user","time":{"created":%d}},"parts":[]},
				{"info":{"id":"msg_2","sessionID":"%s","role":"assistant","time":{"created":%d,"completed":0},"finish":""},"parts":[]}
			]`, sessionID, nowMs-1000, sessionID, nowMs),
			wantProcessing: true,
		},
		{
			name: "idle - assistant message with finish stop",
			messages: fmt.Sprintf(`[
				{"info":{"id":"msg_1","sessionID":"%s","role":"user","time":{"created":%d}},"parts":[]},
				{"info":{"id":"msg_2","sessionID":"%s","role":"assistant","time":{"created":%d,"completed":%d},"finish":"stop"},"parts":[]}
			]`, sessionID, nowMs-2000, sessionID, nowMs-1000, nowMs-500),
			wantProcessing: false,
		},
		{
			name: "idle - assistant message with finish tool-calls",
			messages: fmt.Sprintf(`[
				{"info":{"id":"msg_1","sessionID":"%s","role":"user","time":{"created":%d}},"parts":[]},
				{"info":{"id":"msg_2","sessionID":"%s","role":"assistant","time":{"created":%d,"completed":%d},"finish":"tool-calls"},"parts":[]}
			]`, sessionID, nowMs-2000, sessionID, nowMs-1000, nowMs-500),
			wantProcessing: false,
		},
		{
			name: "processing - user message just sent (within 30s)",
			messages: fmt.Sprintf(`[
				{"info":{"id":"msg_1","sessionID":"%s","role":"assistant","time":{"created":%d,"completed":%d},"finish":"stop"},"parts":[]},
				{"info":{"id":"msg_2","sessionID":"%s","role":"user","time":{"created":%d}},"parts":[]}
			]`, sessionID, nowMs-2000, nowMs-1500, sessionID, nowMs-5000), // 5 seconds ago
			wantProcessing: true,
		},
		{
			name: "idle - user message old (more than 30s ago)",
			messages: fmt.Sprintf(`[
				{"info":{"id":"msg_1","sessionID":"%s","role":"assistant","time":{"created":%d,"completed":%d},"finish":"stop"},"parts":[]},
				{"info":{"id":"msg_2","sessionID":"%s","role":"user","time":{"created":%d}},"parts":[]}
			]`, sessionID, nowMs-60000, nowMs-59000, sessionID, nowMs-35000), // 35 seconds ago
			wantProcessing: false,
		},
		{
			name:           "no messages",
			messages:       "[]",
			wantProcessing: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.messages))
			}))
			defer server.Close()

			client := NewClient(server.URL)
			isProcessing := client.IsSessionProcessing(sessionID)
			if isProcessing != tt.wantProcessing {
				t.Errorf("IsSessionProcessing() = %v, want %v", isProcessing, tt.wantProcessing)
			}
		})
	}
}

// TestIsSessionProcessingServerError tests IsSessionProcessing with server error.
func TestIsSessionProcessingServerError(t *testing.T) {
	sessionID := "ses_error"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	isProcessing := client.IsSessionProcessing(sessionID)
	if isProcessing {
		t.Error("Expected false when server returns error")
	}
}

// TestIsSessionActive tests the IsSessionActive method.
func TestIsSessionActive(t *testing.T) {
	sessionID := "ses_test123"
	nowMs := time.Now().UnixMilli()

	tests := []struct {
		name        string
		sessionJSON string
		maxIdleTime time.Duration
		wantActive  bool
	}{
		{
			name: "active - updated recently",
			sessionJSON: fmt.Sprintf(`{
				"id": "%s",
				"title": "Test Session",
				"directory": "/tmp/project",
				"time": {"created": %d, "updated": %d}
			}`, sessionID, nowMs-60000, nowMs-5000), // updated 5 seconds ago
			maxIdleTime: 30 * time.Minute,
			wantActive:  true,
		},
		{
			name: "stale - updated more than maxIdleTime ago",
			sessionJSON: fmt.Sprintf(`{
				"id": "%s",
				"title": "Test Session",
				"directory": "/tmp/project",
				"time": {"created": %d, "updated": %d}
			}`, sessionID, nowMs-3600000, nowMs-3600000), // updated 1 hour ago
			maxIdleTime: 30 * time.Minute,
			wantActive:  false,
		},
		{
			name: "active - just under maxIdleTime",
			sessionJSON: fmt.Sprintf(`{
				"id": "%s",
				"title": "Test Session",
				"directory": "/tmp/project",
				"time": {"created": %d, "updated": %d}
			}`, sessionID, nowMs-1790000, nowMs-1790000), // updated ~29.8 minutes ago
			maxIdleTime: 30 * time.Minute,
			wantActive:  true, // < maxIdleTime so still active
		},
		{
			name: "stale - just over maxIdleTime",
			sessionJSON: fmt.Sprintf(`{
				"id": "%s",
				"title": "Test Session",
				"directory": "/tmp/project",
				"time": {"created": %d, "updated": %d}
			}`, sessionID, nowMs-1860000, nowMs-1860000), // updated 31 minutes ago
			maxIdleTime: 30 * time.Minute,
			wantActive:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/session/"+sessionID {
					t.Errorf("Expected path /session/%s, got %s", sessionID, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.sessionJSON))
			}))
			defer server.Close()

			client := NewClient(server.URL)
			isActive := client.IsSessionActive(sessionID, tt.maxIdleTime)
			if isActive != tt.wantActive {
				t.Errorf("IsSessionActive() = %v, want %v", isActive, tt.wantActive)
			}
		})
	}
}

// TestIsSessionActiveServerError tests IsSessionActive with server error.
func TestIsSessionActiveServerError(t *testing.T) {
	sessionID := "ses_error"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	isActive := client.IsSessionActive(sessionID, 30*time.Minute)
	if isActive {
		t.Error("Expected false when server returns error")
	}
}

// TestIsSessionActiveNotFound tests IsSessionActive when session doesn't exist.
func TestIsSessionActiveNotFound(t *testing.T) {
	sessionID := "ses_notfound"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	isActive := client.IsSessionActive(sessionID, 30*time.Minute)
	if isActive {
		t.Error("Expected false when session not found")
	}
}
