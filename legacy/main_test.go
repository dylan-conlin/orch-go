package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Test structures for parsing opencode output
func TestParseOpenCodeEvent(t *testing.T) {
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
			event, err := ParseOpenCodeEvent(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseOpenCodeEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && event.Type != tt.wantType {
				t.Errorf("ParseOpenCodeEvent() type = %v, want %v", event.Type, tt.wantType)
			}
		})
	}
}

// Test SSE event parsing
func TestParseSSEEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantEvent string
		wantData  string
	}{
		{
			name:      "session.status event",
			input:     "event: session.status\ndata: {\"status\":\"idle\"}\n\n",
			wantEvent: "session.status",
			wantData:  `{"status":"idle"}`,
		},
		{
			name:      "session.created event",
			input:     "event: session.created\ndata: {\"id\":\"ses_123\"}\n\n",
			wantEvent: "session.created",
			wantData:  `{"id":"ses_123"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, data := ParseSSEEvent(tt.input)
			if event != tt.wantEvent {
				t.Errorf("ParseSSEEvent() event = %v, want %v", event, tt.wantEvent)
			}
			if data != tt.wantData {
				t.Errorf("ParseSSEEvent() data = %v, want %v", data, tt.wantData)
			}
		})
	}
}

// Test session status detection
func TestSessionStatusFromSSE(t *testing.T) {
	data := `{"status":"idle","session_id":"ses_abc123"}`
	status, sessionID := ParseSessionStatus(data)
	if status != "idle" {
		t.Errorf("ParseSessionStatus() status = %v, want idle", status)
	}
	if sessionID != "ses_abc123" {
		t.Errorf("ParseSessionStatus() sessionID = %v, want ses_abc123", sessionID)
	}
}

// Test event logging
func TestLogEvent(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")

	event := Event{
		Type:      "session.created",
		SessionID: "ses_123",
		Timestamp: time.Now().Unix(),
		Data:      map[string]interface{}{"title": "test"},
	}

	err := LogEvent(logPath, event)
	if err != nil {
		t.Fatalf("LogEvent() error = %v", err)
	}

	// Read the file and verify
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(data), "session.created") {
		t.Errorf("Log file doesn't contain expected event type")
	}
	if !strings.Contains(string(data), "ses_123") {
		t.Errorf("Log file doesn't contain expected session ID")
	}
}

// Test extract session ID from opencode output
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

// Test build opencode command
func TestBuildSpawnCommand(t *testing.T) {
	cmd := BuildSpawnCommand("http://localhost:4096", "say hello", "test-title")

	expectedArgs := []string{
		"run",
		"--attach", "http://localhost:4096",
		"--format", "json",
		"--title", "test-title",
		"say hello",
	}

	if len(cmd.Args) < len(expectedArgs)+1 { // +1 for command name
		t.Errorf("BuildSpawnCommand() args length = %v, want at least %v", len(cmd.Args), len(expectedArgs)+1)
	}
}

// Test build ask command
func TestBuildAskCommand(t *testing.T) {
	cmd := BuildAskCommand("http://localhost:4096", "ses_123", "what did you do?")

	expectedArgs := []string{
		"run",
		"--attach", "http://localhost:4096",
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

// Test SSE client with mock server
func TestSSEClient(t *testing.T) {
	// Create a mock SSE server
	events := []string{
		"event: server.connected\ndata: {}\n\n",
		"event: session.status\ndata: {\"status\":\"busy\",\"session_id\":\"ses_test\"}\n\n",
		"event: session.status\ndata: {\"status\":\"idle\",\"session_id\":\"ses_test\"}\n\n",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE not supported", http.StatusInternalServerError)
			return
		}

		for _, event := range events {
			w.Write([]byte(event))
			flusher.Flush()
		}
	}))
	defer server.Close()

	// Test SSE client
	client := NewSSEClient(server.URL)

	eventChan := make(chan SSEEvent, 10)
	errChan := make(chan error, 1)

	go func() {
		err := client.Connect(eventChan)
		if err != nil && err != io.EOF {
			errChan <- err
		}
		close(eventChan)
	}()

	// Collect events with timeout
	var receivedEvents []SSEEvent
	timeout := time.After(2 * time.Second)

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				goto done
			}
			receivedEvents = append(receivedEvents, event)
		case err := <-errChan:
			t.Fatalf("SSE client error: %v", err)
		case <-timeout:
			t.Fatal("Timeout waiting for SSE events")
		}
	}

done:
	if len(receivedEvents) < 3 {
		t.Errorf("Expected at least 3 events, got %d", len(receivedEvents))
	}
}

// Test completion detection logic
func TestDetectCompletion(t *testing.T) {
	events := []SSEEvent{
		{Event: "session.status", Data: `{"status":"busy","session_id":"ses_123"}`},
		{Event: "message.updated", Data: `{"content":"working..."}`},
		{Event: "session.status", Data: `{"status":"idle","session_id":"ses_123"}`},
	}

	sessionID, completed := DetectCompletion(events)
	if !completed {
		t.Error("DetectCompletion() should detect completion")
	}
	if sessionID != "ses_123" {
		t.Errorf("DetectCompletion() sessionID = %v, want ses_123", sessionID)
	}
}

// Test CLI argument parsing
func TestParseArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCmd    string
		wantPrompt string
		wantSessID string
	}{
		{
			name:       "spawn command",
			args:       []string{"orch-go", "spawn", "say hello"},
			wantCmd:    "spawn",
			wantPrompt: "say hello",
		},
		{
			name:       "ask command",
			args:       []string{"orch-go", "ask", "ses_123", "follow-up question"},
			wantCmd:    "ask",
			wantSessID: "ses_123",
			wantPrompt: "follow-up question",
		},
		{
			name:    "monitor command",
			args:    []string{"orch-go", "monitor"},
			wantCmd: "monitor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseArgs(tt.args)
			if err != nil {
				t.Fatalf("ParseArgs() error = %v", err)
			}
			if cfg.Command != tt.wantCmd {
				t.Errorf("ParseArgs() command = %v, want %v", cfg.Command, tt.wantCmd)
			}
			if tt.wantPrompt != "" && cfg.Prompt != tt.wantPrompt {
				t.Errorf("ParseArgs() prompt = %v, want %v", cfg.Prompt, tt.wantPrompt)
			}
			if tt.wantSessID != "" && cfg.SessionID != tt.wantSessID {
				t.Errorf("ParseArgs() sessionID = %v, want %v", cfg.SessionID, tt.wantSessID)
			}
		})
	}
}

// Test notification message formatting
func TestFormatNotification(t *testing.T) {
	title, body := FormatNotification("ses_abc123", "completed")
	if title == "" {
		t.Error("FormatNotification() title should not be empty")
	}
	if body == "" {
		t.Error("FormatNotification() body should not be empty")
	}
	if !strings.Contains(body, "ses_abc123") {
		t.Error("FormatNotification() body should contain session ID")
	}
}

// Test event JSON serialization
func TestEventSerialization(t *testing.T) {
	event := Event{
		Type:      "session.status",
		SessionID: "ses_test",
		Timestamp: 1703001600,
		Data:      map[string]interface{}{"status": "idle"},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var parsed Event
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if parsed.Type != event.Type {
		t.Errorf("Type mismatch: got %v, want %v", parsed.Type, event.Type)
	}
	if parsed.SessionID != event.SessionID {
		t.Errorf("SessionID mismatch: got %v, want %v", parsed.SessionID, event.SessionID)
	}
}

// Integration test helpers
type mockOpenCodeOutput struct {
	events []string
}

func (m *mockOpenCodeOutput) Read(p []byte) (n int, err error) {
	if len(m.events) == 0 {
		return 0, io.EOF
	}
	event := m.events[0]
	m.events = m.events[1:]
	copy(p, event+"\n")
	return len(event) + 1, nil
}

func TestProcessOpenCodeOutput(t *testing.T) {
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

	result, err := ProcessOpenCodeOutput(&output)
	if err != nil {
		t.Fatalf("ProcessOpenCodeOutput() error = %v", err)
	}

	if result.SessionID != "ses_xyz" {
		t.Errorf("SessionID = %v, want ses_xyz", result.SessionID)
	}
	if len(result.Events) != 3 {
		t.Errorf("Events count = %d, want 3", len(result.Events))
	}
}
