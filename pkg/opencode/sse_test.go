package opencode

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

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
		{
			name:      "message.updated event",
			input:     "event: message.updated\ndata: {\"content\":\"working...\"}\n\n",
			wantEvent: "message.updated",
			wantData:  `{"content":"working..."}`,
		},
		{
			name:      "step_finish event",
			input:     "event: step_finish\ndata: {\"step_id\":\"step_1\",\"tokens\":100}\n\n",
			wantEvent: "step_finish",
			wantData:  `{"step_id":"step_1","tokens":100}`,
		},
		{
			name:      "server.connected event",
			input:     "event: server.connected\ndata: {}\n\n",
			wantEvent: "server.connected",
			wantData:  `{}`,
		},
		{
			name:      "empty input",
			input:     "",
			wantEvent: "",
			wantData:  "",
		},
		{
			name:      "only event line",
			input:     "event: session.status\n",
			wantEvent: "session.status",
			wantData:  "",
		},
		{
			name:      "only data line",
			input:     "data: {\"test\":true}\n",
			wantEvent: "",
			wantData:  `{"test":true}`,
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

// Test session status parsing from SSE data
func TestParseSessionStatus(t *testing.T) {
	tests := []struct {
		name          string
		data          string
		wantStatus    string
		wantSessionID string
	}{
		{
			name:          "idle status with session ID",
			data:          `{"status":"idle","session_id":"ses_abc123"}`,
			wantStatus:    "idle",
			wantSessionID: "ses_abc123",
		},
		{
			name:          "busy status with session ID",
			data:          `{"status":"busy","session_id":"ses_xyz789"}`,
			wantStatus:    "busy",
			wantSessionID: "ses_xyz789",
		},
		{
			name:          "status only",
			data:          `{"status":"idle"}`,
			wantStatus:    "idle",
			wantSessionID: "",
		},
		{
			name:          "invalid json",
			data:          `not json`,
			wantStatus:    "",
			wantSessionID: "",
		},
		{
			name:          "empty json",
			data:          `{}`,
			wantStatus:    "",
			wantSessionID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, sessionID := ParseSessionStatus(tt.data)
			if status != tt.wantStatus {
				t.Errorf("ParseSessionStatus() status = %v, want %v", status, tt.wantStatus)
			}
			if sessionID != tt.wantSessionID {
				t.Errorf("ParseSessionStatus() sessionID = %v, want %v", sessionID, tt.wantSessionID)
			}
		})
	}
}

// Test completion detection logic
func TestDetectCompletion(t *testing.T) {
	tests := []struct {
		name          string
		events        []SSEEvent
		wantSessionID string
		wantCompleted bool
	}{
		{
			name: "busy to idle transition",
			events: []SSEEvent{
				{Event: "session.status", Data: `{"status":"busy","session_id":"ses_123"}`},
				{Event: "message.updated", Data: `{"content":"working..."}`},
				{Event: "session.status", Data: `{"status":"idle","session_id":"ses_123"}`},
			},
			wantSessionID: "ses_123",
			wantCompleted: true,
		},
		{
			name: "still busy",
			events: []SSEEvent{
				{Event: "session.status", Data: `{"status":"busy","session_id":"ses_456"}`},
				{Event: "message.updated", Data: `{"content":"working..."}`},
			},
			wantSessionID: "ses_456",
			wantCompleted: false,
		},
		{
			name: "no session status events",
			events: []SSEEvent{
				{Event: "message.updated", Data: `{"content":"hello"}`},
			},
			wantSessionID: "",
			wantCompleted: false,
		},
		{
			name: "idle without session ID",
			events: []SSEEvent{
				{Event: "session.status", Data: `{"status":"idle"}`},
			},
			wantSessionID: "",
			wantCompleted: false,
		},
		{
			name:          "empty events",
			events:        []SSEEvent{},
			wantSessionID: "",
			wantCompleted: false,
		},
		{
			name: "multiple sessions - last wins",
			events: []SSEEvent{
				{Event: "session.status", Data: `{"status":"idle","session_id":"ses_first"}`},
				{Event: "session.status", Data: `{"status":"busy","session_id":"ses_second"}`},
				{Event: "session.status", Data: `{"status":"idle","session_id":"ses_second"}`},
			},
			wantSessionID: "ses_second",
			wantCompleted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionID, completed := DetectCompletion(tt.events)
			if sessionID != tt.wantSessionID {
				t.Errorf("DetectCompletion() sessionID = %v, want %v", sessionID, tt.wantSessionID)
			}
			if completed != tt.wantCompleted {
				t.Errorf("DetectCompletion() completed = %v, want %v", completed, tt.wantCompleted)
			}
		})
	}
}

// Test SSE stream reading
func TestReadSSEStream(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantEvents []SSEEvent
	}{
		{
			name:  "single event",
			input: "event: session.status\ndata: {\"status\":\"idle\"}\n\n",
			wantEvents: []SSEEvent{
				{Event: "session.status", Data: `{"status":"idle"}`},
			},
		},
		{
			name:  "multiple events",
			input: "event: server.connected\ndata: {}\n\nevent: session.status\ndata: {\"status\":\"busy\"}\n\nevent: session.status\ndata: {\"status\":\"idle\"}\n\n",
			wantEvents: []SSEEvent{
				{Event: "server.connected", Data: `{}`},
				{Event: "session.status", Data: `{"status":"busy"}`},
				{Event: "session.status", Data: `{"status":"idle"}`},
			},
		},
		{
			name:       "empty stream",
			input:      "",
			wantEvents: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := make(chan SSEEvent, 10)
			reader := strings.NewReader(tt.input)

			err := ReadSSEStream(reader, events)
			close(events)

			if err != nil {
				t.Fatalf("ReadSSEStream() error = %v", err)
			}

			var gotEvents []SSEEvent
			for event := range events {
				gotEvents = append(gotEvents, event)
			}

			if len(gotEvents) != len(tt.wantEvents) {
				t.Errorf("ReadSSEStream() got %d events, want %d", len(gotEvents), len(tt.wantEvents))
				return
			}

			for i, got := range gotEvents {
				want := tt.wantEvents[i]
				if got.Event != want.Event || got.Data != want.Data {
					t.Errorf("ReadSSEStream() event[%d] = %+v, want %+v", i, got, want)
				}
			}
		})
	}
}

// Test SSE client with mock HTTP server
func TestSSEClient(t *testing.T) {
	// Create a mock SSE server
	events := []string{
		"event: server.connected\ndata: {}\n\n",
		"event: session.status\ndata: {\"status\":\"busy\",\"session_id\":\"ses_test\"}\n\n",
		"event: message.updated\ndata: {\"content\":\"working\"}\n\n",
		"event: step_finish\ndata: {\"tokens\":150}\n\n",
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
	// Verify we received the expected events
	if len(receivedEvents) != 5 {
		t.Errorf("Expected 5 events, got %d", len(receivedEvents))
	}

	// Verify we can detect completion
	sessionID, completed := DetectCompletion(receivedEvents)
	if !completed {
		t.Error("DetectCompletion() should detect completion")
	}
	if sessionID != "ses_test" {
		t.Errorf("DetectCompletion() sessionID = %v, want ses_test", sessionID)
	}
}

// Test SSEClient constructor
func TestNewSSEClient(t *testing.T) {
	url := "http://127.0.0.1:4096/event"
	client := NewSSEClient(url)

	if client == nil {
		t.Fatal("NewSSEClient() returned nil")
	}
	if client.URL != url {
		t.Errorf("NewSSEClient() URL = %v, want %v", client.URL, url)
	}
}

// Test connection error handling
func TestSSEClientConnectionError(t *testing.T) {
	// Use an invalid URL that will fail to connect
	client := NewSSEClient("http://127.0.0.1:99999/event")

	events := make(chan SSEEvent, 10)
	err := client.Connect(events)

	if err == nil {
		t.Error("Connect() should return error for invalid URL")
	}
}
