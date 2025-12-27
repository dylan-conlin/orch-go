package opencode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseEvent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
		wantErr  bool
	}{
		{
			name:     "step_start event",
			input:    `{"type":"step_start","step":{"id":"step_123"}}`,
			wantType: "step_start",
			wantErr:  false,
		},
		{
			name:     "text event",
			input:    `{"type":"text","content":"hello"}`,
			wantType: "text",
			wantErr:  false,
		},
		{
			name:     "step_finish event",
			input:    `{"type":"step_finish","step":{"id":"step_123"}}`,
			wantType: "step_finish",
			wantErr:  false,
		},
		{
			name:     "invalid json",
			input:    `not json`,
			wantType: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParseEvent(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && event.Type != tt.wantType {
				t.Errorf("ParseEvent() type = %v, want %v", event.Type, tt.wantType)
			}
		})
	}
}

func TestExtractSessionID(t *testing.T) {
	tests := []struct {
		name    string
		events  []string
		wantID  string
		wantErr bool
	}{
		{
			name: "sessionID at top level (actual opencode format)",
			events: []string{
				`{"type":"step_start","timestamp":1766199826875,"sessionID":"ses_abc123"}`,
			},
			wantID:  "ses_abc123",
			wantErr: false,
		},
		{
			name: "no sessionID in output",
			events: []string{
				`{"type":"text","content":"hello"}`,
			},
			wantID:  "",
			wantErr: true,
		},
		{
			name: "sessionID in second event",
			events: []string{
				`{"type":"init"}`,
				`{"type":"step_start","sessionID":"ses_xyz789"}`,
			},
			wantID:  "ses_xyz789",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ExtractSessionID(tt.events)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractSessionID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if id != tt.wantID {
				t.Errorf("ExtractSessionID() = %v, want %v", id, tt.wantID)
			}
		})
	}
}

func TestExtractSessionIDFromReader(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantID  string
		wantErr bool
	}{
		{
			name:    "sessionID in first event",
			input:   `{"type":"step_start","sessionID":"ses_abc123"}` + "\n" + `{"type":"text","content":"hello"}` + "\n",
			wantID:  "ses_abc123",
			wantErr: false,
		},
		{
			name:    "sessionID in second event",
			input:   `{"type":"init"}` + "\n" + `{"type":"step_start","sessionID":"ses_xyz789"}` + "\n",
			wantID:  "ses_xyz789",
			wantErr: false,
		},
		{
			name:    "no sessionID in output",
			input:   `{"type":"text","content":"hello"}` + "\n",
			wantID:  "",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantID:  "",
			wantErr: true,
		},
		{
			name:    "mixed valid and invalid lines",
			input:   "invalid line\n" + `{"type":"init"}` + "\n" + `{"type":"step_start","sessionID":"ses_mixed"}` + "\n",
			wantID:  "ses_mixed",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewBufferString(tt.input)
			id, err := ExtractSessionIDFromReader(reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractSessionIDFromReader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if id != tt.wantID {
				t.Errorf("ExtractSessionIDFromReader() = %v, want %v", id, tt.wantID)
			}
		})
	}
}

func TestProcessOutput(t *testing.T) {
	// Use actual opencode format with sessionID at top level
	events := []string{
		`{"type":"step_start","timestamp":1766199826875,"sessionID":"ses_xyz","step":{"id":"step_1"}}`,
		`{"type":"text","sessionID":"ses_xyz","content":"Hello!"}`,
		`{"type":"step_finish","sessionID":"ses_xyz","step":{"id":"step_1"}}`,
	}

	var output bytes.Buffer
	for _, e := range events {
		output.WriteString(e + "\n")
	}

	result, err := ProcessOutput(&output)
	if err != nil {
		t.Fatalf("ProcessOutput() error = %v", err)
	}

	if result.SessionID != "ses_xyz" {
		t.Errorf("SessionID = %v, want ses_xyz", result.SessionID)
	}
	if len(result.Events) != 3 {
		t.Errorf("Events count = %d, want 3", len(result.Events))
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient("http://127.0.0.1:4096")
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client.ServerURL != "http://127.0.0.1:4096" {
		t.Errorf("ServerURL = %v, want http://127.0.0.1:4096", client.ServerURL)
	}
}

func TestBuildSpawnCommand(t *testing.T) {
	client := NewClient("http://127.0.0.1:4096")
	cmd := client.BuildSpawnCommand("say hello", "test-title", "")

	expectedArgs := []string{
		"run",
		"--attach", "http://127.0.0.1:4096",
		"--format", "json",
		"--title", "test-title",
		"say hello",
	}

	if len(cmd.Args) < len(expectedArgs)+1 { // +1 for command name
		t.Errorf("BuildSpawnCommand() args length = %v, want at least %v", len(cmd.Args), len(expectedArgs)+1)
	}
}

func TestBuildSpawnCommandWithModel(t *testing.T) {
	client := NewClient("http://127.0.0.1:4096")
	cmd := client.BuildSpawnCommand("say hello", "test-title", "anthropic/claude-opus-4")

	expectedArgs := []string{
		"run",
		"--attach", "http://127.0.0.1:4096",
		"--format", "json",
		"--model", "anthropic/claude-opus-4",
		"--title", "test-title",
		"say hello",
	}

	// Check that all expected args are present
	found := 0
	for _, expected := range expectedArgs {
		for _, arg := range cmd.Args {
			if arg == expected {
				found++
				break
			}
		}
	}

	if found < len(expectedArgs) {
		t.Errorf("BuildSpawnCommand() missing expected args, found %v of %v. Args: %v", found, len(expectedArgs), cmd.Args)
	}

	// Verify --model flag is included
	hasModel := false
	for i, arg := range cmd.Args {
		if arg == "--model" && i+1 < len(cmd.Args) && cmd.Args[i+1] == "anthropic/claude-opus-4" {
			hasModel = true
			break
		}
	}
	if !hasModel {
		t.Errorf("BuildSpawnCommand() should include --model flag when model is provided. Args: %v", cmd.Args)
	}
}

func TestBuildSpawnCommandWithoutModel(t *testing.T) {
	client := NewClient("http://127.0.0.1:4096")
	cmd := client.BuildSpawnCommand("say hello", "test-title", "")

	// Verify --model flag is NOT included when model is empty
	for i, arg := range cmd.Args {
		if arg == "--model" {
			t.Errorf("BuildSpawnCommand() should not include --model flag when model is empty. Found at index %d. Args: %v", i, cmd.Args)
		}
	}
}

func TestBuildAskCommand(t *testing.T) {
	client := NewClient("http://127.0.0.1:4096")
	cmd := client.BuildAskCommand("ses_123", "what did you do?")

	expectedArgs := []string{
		"run",
		"--attach", "http://127.0.0.1:4096",
		"--session", "ses_123",
		"--format", "json",
		"what did you do?",
	}

	found := 0
	for _, expected := range expectedArgs {
		for _, arg := range cmd.Args {
			if arg == expected {
				found++
				break
			}
		}
	}

	if found < len(expectedArgs) {
		t.Errorf("BuildAskCommand() missing expected args, found %v of %v", found, len(expectedArgs))
	}
}

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
	sessionID, err := client.FindRecentSession(projectDir, "")
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
		sessionID, err := client.FindRecentSessionWithRetry(projectDir, "", 3, 10*time.Millisecond)
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
		sessionID, err := client.FindRecentSessionWithRetry(projectDir, "", 3, 10*time.Millisecond)
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
		sessionID, err := client.FindRecentSessionWithRetry(projectDir, "", 3, 10*time.Millisecond)
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

// TestProcessOutputWithStreaming tests ProcessOutputWithStreaming extracts text content.
func TestProcessOutputWithStreaming(t *testing.T) {
	events := []string{
		`{"type":"step_start","timestamp":1766199826875,"sessionID":"ses_xyz","step":{"id":"step_1"}}`,
		`{"type":"text","sessionID":"ses_xyz","content":"Hello, "}`,
		`{"type":"text","sessionID":"ses_xyz","content":"world!"}`,
		`{"type":"step_finish","sessionID":"ses_xyz","step":{"id":"step_1"}}`,
	}

	var output bytes.Buffer
	for _, e := range events {
		output.WriteString(e + "\n")
	}

	var streamedContent bytes.Buffer
	result, err := ProcessOutputWithStreaming(&output, &streamedContent)
	if err != nil {
		t.Fatalf("ProcessOutputWithStreaming() error = %v", err)
	}

	if result.SessionID != "ses_xyz" {
		t.Errorf("SessionID = %v, want ses_xyz", result.SessionID)
	}
	if len(result.Events) != 4 {
		t.Errorf("Events count = %d, want 4", len(result.Events))
	}

	// Verify streamed content contains the text
	streamed := streamedContent.String()
	if !strings.Contains(streamed, "Hello, ") {
		t.Errorf("Streamed content missing 'Hello, ', got: %s", streamed)
	}
	if !strings.Contains(streamed, "world!") {
		t.Errorf("Streamed content missing 'world!', got: %s", streamed)
	}
}

// TestProcessOutputWithStreamingEmpty tests streaming with no text events.
func TestProcessOutputWithStreamingEmpty(t *testing.T) {
	events := []string{
		`{"type":"step_start","sessionID":"ses_xyz","step":{"id":"step_1"}}`,
		`{"type":"step_finish","sessionID":"ses_xyz","step":{"id":"step_1"}}`,
	}

	var output bytes.Buffer
	for _, e := range events {
		output.WriteString(e + "\n")
	}

	var streamedContent bytes.Buffer
	result, err := ProcessOutputWithStreaming(&output, &streamedContent)
	if err != nil {
		t.Fatalf("ProcessOutputWithStreaming() error = %v", err)
	}

	if result.SessionID != "ses_xyz" {
		t.Errorf("SessionID = %v, want ses_xyz", result.SessionID)
	}
	if len(result.Events) != 2 {
		t.Errorf("Events count = %d, want 2", len(result.Events))
	}

	// Streamed content should be empty (no text events)
	if streamedContent.String() != "" {
		t.Errorf("Expected empty streamed content, got: %s", streamedContent.String())
	}
}

// TestSendMessageWithStreaming tests the SendMessageWithStreaming method.
func TestSendMessageWithStreaming(t *testing.T) {
	sessionID := "ses_test123"
	messageReceived := false
	sseRequestReceived := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/session/" + sessionID + "/prompt_async":
			// Verify the message was received correctly
			messageReceived = true
			w.WriteHeader(http.StatusOK)

		case "/event":
			// SSE endpoint - send a series of events
			sseRequestReceived = true
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("Expected http.Flusher")
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.WriteHeader(http.StatusOK)

			// Send session busy event
			busyEvent := `event: session.status
data: {"type":"session.status","properties":{"sessionID":"` + sessionID + `","status":{"type":"busy"}}}

`
			w.Write([]byte(busyEvent))
			flusher.Flush()

			// Send text streaming events
			textEvent1 := `event: message.part
data: {"type":"message.part","properties":{"sessionID":"` + sessionID + `","messageID":"msg_1","part":{"type":"text","text":"Hello, "}}}

`
			w.Write([]byte(textEvent1))
			flusher.Flush()

			textEvent2 := `event: message.part
data: {"type":"message.part","properties":{"sessionID":"` + sessionID + `","messageID":"msg_1","part":{"type":"text","text":"world!"}}}

`
			w.Write([]byte(textEvent2))
			flusher.Flush()

			// Send session idle event (completion)
			idleEvent := `event: session.status
data: {"type":"session.status","properties":{"sessionID":"` + sessionID + `","status":{"type":"idle"}}}

`
			w.Write([]byte(idleEvent))
			flusher.Flush()

		default:
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	var streamedContent bytes.Buffer
	err := client.SendMessageWithStreaming(sessionID, "test message", &streamedContent)
	if err != nil {
		t.Fatalf("SendMessageWithStreaming() error = %v", err)
	}

	if !messageReceived {
		t.Error("Message was not sent to async endpoint")
	}
	if !sseRequestReceived {
		t.Error("SSE request was not made")
	}

	// Check streamed content
	content := streamedContent.String()
	if !strings.Contains(content, "Hello, ") {
		t.Errorf("Streamed content missing 'Hello, ', got: %s", content)
	}
	if !strings.Contains(content, "world!") {
		t.Errorf("Streamed content missing 'world!', got: %s", content)
	}
}

// TestSendMessageWithStreamingSessionError tests that session errors are properly detected.
func TestSendMessageWithStreamingSessionError(t *testing.T) {
	sessionID := "ses_error_test"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/session/" + sessionID + "/prompt_async":
			w.WriteHeader(http.StatusOK)

		case "/event":
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("Expected http.Flusher")
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.WriteHeader(http.StatusOK)

			// Send session busy event
			busyEvent := `event: session.status
data: {"type":"session.status","properties":{"sessionID":"` + sessionID + `","status":{"type":"busy"}}}

`
			w.Write([]byte(busyEvent))
			flusher.Flush()

			// Send session error event
			errorEvent := `event: session.error
data: {"type":"session.error","properties":{"sessionID":"` + sessionID + `","error":{"message":"No user message found in stream. This should never happen."}}}

`
			w.Write([]byte(errorEvent))
			flusher.Flush()

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	var streamedContent bytes.Buffer
	err := client.SendMessageWithStreaming(sessionID, "test message", &streamedContent)

	// Should return an error
	if err == nil {
		t.Fatal("SendMessageWithStreaming() expected error for session.error event")
	}

	// Error should contain the session error message
	if !strings.Contains(err.Error(), "No user message found") {
		t.Errorf("Error should contain 'No user message found', got: %v", err)
	}
}

// TestGetMessages tests the GetMessages API call.
func TestGetMessages(t *testing.T) {
	sessionID := "ses_test123"
	mockMessages := `[
		{
			"info": {
				"id": "msg_1",
				"sessionID": "ses_test123",
				"role": "user",
				"time": {"created": 1766282439689}
			},
			"parts": [
				{
					"id": "prt_1",
					"sessionID": "ses_test123",
					"messageID": "msg_1",
					"type": "text",
					"text": "Hello, world!"
				}
			]
		},
		{
			"info": {
				"id": "msg_2",
				"sessionID": "ses_test123",
				"role": "assistant",
				"time": {"created": 1766282440000, "completed": 1766282441000}
			},
			"parts": [
				{
					"id": "prt_2a",
					"sessionID": "ses_test123",
					"messageID": "msg_2",
					"type": "reasoning",
					"text": "I will respond to the greeting."
				},
				{
					"id": "prt_2b",
					"sessionID": "ses_test123",
					"messageID": "msg_2",
					"type": "text",
					"text": "Hello! How can I help you today?"
				}
			]
		}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/session/"+sessionID+"/message" {
			t.Errorf("Expected path /session/%s/message, got %s", sessionID, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockMessages))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	messages, err := client.GetMessages(sessionID)
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(messages))
	}

	// Verify first message
	if messages[0].Info.ID != "msg_1" {
		t.Errorf("messages[0].Info.ID = %s, want msg_1", messages[0].Info.ID)
	}
	if messages[0].Info.Role != "user" {
		t.Errorf("messages[0].Info.Role = %s, want user", messages[0].Info.Role)
	}
	if len(messages[0].Parts) != 1 {
		t.Fatalf("Expected 1 part in message 0, got %d", len(messages[0].Parts))
	}
	if messages[0].Parts[0].Text != "Hello, world!" {
		t.Errorf("messages[0].Parts[0].Text = %s, want 'Hello, world!'", messages[0].Parts[0].Text)
	}

	// Verify second message has two parts
	if len(messages[1].Parts) != 2 {
		t.Fatalf("Expected 2 parts in message 1, got %d", len(messages[1].Parts))
	}
}

// TestGetMessagesEmpty tests GetMessages with empty response.
func TestGetMessagesEmpty(t *testing.T) {
	sessionID := "ses_empty"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	messages, err := client.GetMessages(sessionID)
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(messages))
	}
}

// TestGetMessagesError tests GetMessages with server error.
func TestGetMessagesError(t *testing.T) {
	sessionID := "ses_error"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.GetMessages(sessionID)
	if err == nil {
		t.Error("Expected error for server error response")
	}
}

// TestExtractRecentText tests the ExtractRecentText function.
func TestExtractRecentText(t *testing.T) {
	messages := []Message{
		{
			Info: MessageInfo{ID: "msg_1", Role: "user"},
			Parts: []MessagePart{
				{Type: "text", Text: "Hello"},
			},
		},
		{
			Info: MessageInfo{ID: "msg_2", Role: "assistant"},
			Parts: []MessagePart{
				{Type: "reasoning", Text: "Thinking..."},
				{Type: "text", Text: "Line 1\nLine 2\nLine 3"},
			},
		},
		{
			Info: MessageInfo{ID: "msg_3", Role: "user"},
			Parts: []MessagePart{
				{Type: "text", Text: "Follow up"},
			},
		},
		{
			Info: MessageInfo{ID: "msg_4", Role: "assistant"},
			Parts: []MessagePart{
				{Type: "text", Text: "Response\nWith multiple\nLines"},
			},
		},
	}

	// Request 5 lines
	result := ExtractRecentText(messages, 5)
	if len(result) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(result))
	}

	// Check that we got the most recent lines
	expected := []string{"Follow up", "Response", "With multiple", "Lines"}
	// The last 4 lines should be from the last two messages
	foundLast := false
	for _, line := range result {
		if line == "Lines" {
			foundLast = true
		}
	}
	if !foundLast {
		t.Errorf("Expected to find 'Lines' in result, got: %v", result)
	}
	_ = expected // silence unused warning
}

// TestExtractRecentTextSkipsNonText tests that non-text parts are skipped.
func TestExtractRecentTextSkipsNonText(t *testing.T) {
	messages := []Message{
		{
			Info: MessageInfo{ID: "msg_1", Role: "assistant"},
			Parts: []MessagePart{
				{Type: "step-start", Text: ""},
				{Type: "reasoning", Text: "I am thinking..."},
				{Type: "text", Text: "Only this should appear"},
				{Type: "step-finish", Text: ""},
			},
		},
	}

	result := ExtractRecentText(messages, 10)
	if len(result) != 1 {
		t.Errorf("Expected 1 line, got %d: %v", len(result), result)
	}
	if len(result) > 0 && result[0] != "Only this should appear" {
		t.Errorf("Expected 'Only this should appear', got: %s", result[0])
	}
}

// TestExtractRecentTextEmpty tests ExtractRecentText with no messages.
func TestExtractRecentTextEmpty(t *testing.T) {
	var messages []Message
	result := ExtractRecentText(messages, 10)
	if len(result) != 0 {
		t.Errorf("Expected 0 lines, got %d", len(result))
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
	client := NewClient("http://127.0.0.1:4096")
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

// TestSendMessageWithStreamingIgnoresOtherSessions tests that we only stream from target session.
func TestSendMessageWithStreamingIgnoresOtherSessions(t *testing.T) {
	sessionID := "ses_target"
	otherSessionID := "ses_other"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/session/" + sessionID + "/prompt_async":
			w.WriteHeader(http.StatusOK)

		case "/event":
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("Expected http.Flusher")
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			// Send busy event for target session
			busyEvent := `event: session.status
data: {"type":"session.status","properties":{"sessionID":"` + sessionID + `","status":{"type":"busy"}}}

`
			w.Write([]byte(busyEvent))
			flusher.Flush()

			// Send text from OTHER session - should be ignored
			otherEvent := `event: message.part
data: {"type":"message.part","properties":{"sessionID":"` + otherSessionID + `","messageID":"msg_other","part":{"type":"text","text":"WRONG SESSION"}}}

`
			w.Write([]byte(otherEvent))
			flusher.Flush()

			// Send text from target session
			targetEvent := `event: message.part
data: {"type":"message.part","properties":{"sessionID":"` + sessionID + `","messageID":"msg_1","part":{"type":"text","text":"correct"}}}

`
			w.Write([]byte(targetEvent))
			flusher.Flush()

			// Send completion for target session
			idleEvent := `event: session.status
data: {"type":"session.status","properties":{"sessionID":"` + sessionID + `","status":{"type":"idle"}}}

`
			w.Write([]byte(idleEvent))
			flusher.Flush()

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	var streamedContent bytes.Buffer
	err := client.SendMessageWithStreaming(sessionID, "test", &streamedContent)
	if err != nil {
		t.Fatalf("SendMessageWithStreaming() error = %v", err)
	}

	content := streamedContent.String()
	if strings.Contains(content, "WRONG SESSION") {
		t.Errorf("Streamed content should not include events from other sessions, got: %s", content)
	}
	if !strings.Contains(content, "correct") {
		t.Errorf("Streamed content missing 'correct', got: %s", content)
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
	resp, err := client.CreateSession(title, directory, model)
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

	// Verify ORCH_WORKER header is set
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
	_, err := client.CreateSession(title, directory, model)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Verify empty model was sent (omitempty should exclude it from JSON)
	if receivedRequest.Model != "" {
		t.Errorf("receivedRequest.Model = %s, want empty string", receivedRequest.Model)
	}
}

// TestSendMessageAsyncWithModel tests that SendMessageAsync includes model parameter as an object in payload.
func TestSendMessageAsyncWithModel(t *testing.T) {
	sessionID := "ses_test123"
	content := "test message"
	model := "anthropic/claude-opus-4"

	var receivedPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		expectedPath := "/session/" + sessionID + "/prompt_async"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Decode request body
		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.SendMessageAsync(sessionID, content, model)
	if err != nil {
		t.Fatalf("SendMessageAsync() error = %v", err)
	}

	// Verify model is included in payload as an object with providerID and modelID
	modelObj, ok := receivedPayload["model"].(map[string]interface{})
	if !ok {
		t.Fatalf("receivedPayload[\"model\"] is not an object, got %T: %v", receivedPayload["model"], receivedPayload["model"])
	}
	if modelObj["providerID"] != "anthropic" {
		t.Errorf("model.providerID = %v, want anthropic", modelObj["providerID"])
	}
	if modelObj["modelID"] != "claude-opus-4" {
		t.Errorf("model.modelID = %v, want claude-opus-4", modelObj["modelID"])
	}

	// Verify parts are included
	parts, ok := receivedPayload["parts"].([]interface{})
	if !ok {
		t.Fatalf("receivedPayload[\"parts\"] is not an array")
	}
	if len(parts) != 1 {
		t.Fatalf("Expected 1 part, got %d", len(parts))
	}

	// Verify agent is included
	if receivedPayload["agent"] != "build" {
		t.Errorf("receivedPayload[\"agent\"] = %v, want build", receivedPayload["agent"])
	}
}

// TestParseModelSpec tests the parseModelSpec helper function.
func TestParseModelSpec(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantNil  bool
		provider string
		modelID  string
	}{
		{
			name:     "valid provider/modelID format",
			input:    "google/gemini-2.5-flash",
			wantNil:  false,
			provider: "google",
			modelID:  "gemini-2.5-flash",
		},
		{
			name:     "valid anthropic model",
			input:    "anthropic/claude-opus-4-5-20251101",
			wantNil:  false,
			provider: "anthropic",
			modelID:  "claude-opus-4-5-20251101",
		},
		{
			name:    "empty string",
			input:   "",
			wantNil: true,
		},
		{
			name:    "no slash",
			input:   "claude-opus-4",
			wantNil: true,
		},
		{
			name:    "empty provider",
			input:   "/modelID",
			wantNil: true,
		},
		{
			name:    "empty modelID",
			input:   "provider/",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseModelSpec(tt.input)
			if tt.wantNil {
				if result != nil {
					t.Errorf("parseModelSpec(%q) = %v, want nil", tt.input, result)
				}
				return
			}
			if result == nil {
				t.Fatalf("parseModelSpec(%q) = nil, want non-nil", tt.input)
			}
			if result["providerID"] != tt.provider {
				t.Errorf("providerID = %v, want %v", result["providerID"], tt.provider)
			}
			if result["modelID"] != tt.modelID {
				t.Errorf("modelID = %v, want %v", result["modelID"], tt.modelID)
			}
		})
	}
}

// TestSendMessageAsyncWithoutModel tests that SendMessageAsync excludes model when empty.
func TestSendMessageAsyncWithoutModel(t *testing.T) {
	sessionID := "ses_test123"
	content := "test message"
	model := "" // Empty model

	var receivedPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.SendMessageAsync(sessionID, content, model)
	if err != nil {
		t.Fatalf("SendMessageAsync() error = %v", err)
	}

	// Verify model is NOT included in payload when empty
	if _, hasModel := receivedPayload["model"]; hasModel {
		t.Errorf("receivedPayload should not include 'model' field when empty, but got: %v", receivedPayload["model"])
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

// TestGetLastMessage tests the GetLastMessage method.
func TestGetLastMessage(t *testing.T) {
	sessionID := "ses_test123"
	mockMessages := `[
		{"info":{"id":"msg_1","sessionID":"ses_test123","role":"user","time":{"created":1766282439689}},"parts":[]},
		{"info":{"id":"msg_2","sessionID":"ses_test123","role":"assistant","time":{"created":1766282440000,"completed":1766282441000},"finish":"stop"},"parts":[]}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockMessages))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	msg, err := client.GetLastMessage(sessionID)
	if err != nil {
		t.Fatalf("GetLastMessage() error = %v", err)
	}
	if msg == nil {
		t.Fatal("GetLastMessage() returned nil")
	}
	if msg.Info.ID != "msg_2" {
		t.Errorf("GetLastMessage().Info.ID = %s, want msg_2", msg.Info.ID)
	}
	if msg.Info.Finish != "stop" {
		t.Errorf("GetLastMessage().Info.Finish = %s, want stop", msg.Info.Finish)
	}
}

// TestGetLastMessageEmpty tests GetLastMessage with no messages.
func TestGetLastMessageEmpty(t *testing.T) {
	sessionID := "ses_empty"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	msg, err := client.GetLastMessage(sessionID)
	if err != nil {
		t.Fatalf("GetLastMessage() error = %v", err)
	}
	if msg != nil {
		t.Errorf("GetLastMessage() = %v, want nil for empty session", msg)
	}
}

// TestGetLastMessageError tests GetLastMessage with server error.
func TestGetLastMessageError(t *testing.T) {
	sessionID := "ses_error"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.GetLastMessage(sessionID)
	if err == nil {
		t.Error("Expected error for server error response")
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
