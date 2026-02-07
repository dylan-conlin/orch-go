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

// TestExportSessionTranscript tests the ExportSessionTranscript method.
func TestExportSessionTranscript(t *testing.T) {
	sessionID := "ses_transcript_test"
	nowMs := time.Now().UnixMilli()

	mockSession := fmt.Sprintf(`{
		"id": "%s",
		"title": "Test Transcript Session",
		"directory": "/Users/test/project",
		"time": {"created": %d, "updated": %d},
		"summary": {"additions": 10, "deletions": 5, "files": 3}
	}`, sessionID, nowMs-60000, nowMs-1000)

	mockMessages := fmt.Sprintf(`[
		{
			"info": {"id": "msg_1", "sessionID": "%s", "role": "user", "time": {"created": %d}},
			"parts": [{"id": "prt_1", "sessionID": "%s", "messageID": "msg_1", "type": "text", "text": "Hello, can you help me?"}]
		},
		{
			"info": {"id": "msg_2", "sessionID": "%s", "role": "assistant", "time": {"created": %d, "completed": %d}, "finish": "stop", "tokens": {"input": 100, "output": 50}, "cost": 0.0015},
			"parts": [
				{"id": "prt_2a", "sessionID": "%s", "messageID": "msg_2", "type": "text", "text": "Of course! I'd be happy to help you."},
				{"id": "prt_2b", "sessionID": "%s", "messageID": "msg_2", "type": "tool", "text": ""}
			]
		}
	]`, sessionID, nowMs-30000, sessionID, sessionID, nowMs-25000, nowMs-20000, sessionID, sessionID)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/session/" + sessionID:
			w.Write([]byte(mockSession))
		case "/session/" + sessionID + "/message":
			w.Write([]byte(mockMessages))
		default:
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	transcript, err := client.ExportSessionTranscript(sessionID)
	if err != nil {
		t.Fatalf("ExportSessionTranscript() error = %v", err)
	}

	// Verify transcript contains expected content
	if !strings.Contains(transcript, "# Session Transcript") {
		t.Error("Transcript missing header")
	}
	if !strings.Contains(transcript, "**Title:** Test Transcript Session") {
		t.Error("Transcript missing title")
	}
	if !strings.Contains(transcript, fmt.Sprintf("**Session ID:** `%s`", sessionID)) {
		t.Error("Transcript missing session ID")
	}
	if !strings.Contains(transcript, "**Directory:** `/Users/test/project`") {
		t.Error("Transcript missing directory")
	}
	if !strings.Contains(transcript, "**Changes:** +10/-5 in 3 files") {
		t.Error("Transcript missing changes summary")
	}
	if !strings.Contains(transcript, "## User") {
		t.Error("Transcript missing user message")
	}
	if !strings.Contains(transcript, "Hello, can you help me?") {
		t.Error("Transcript missing user message content")
	}
	if !strings.Contains(transcript, "## Assistant") {
		t.Error("Transcript missing assistant message")
	}
	if !strings.Contains(transcript, "Of course! I'd be happy to help you.") {
		t.Error("Transcript missing assistant message content")
	}
	if !strings.Contains(transcript, "*Tokens:") {
		t.Error("Transcript missing token info")
	}
}

// TestExportSessionTranscriptEmpty tests ExportSessionTranscript with no messages.
func TestExportSessionTranscriptEmpty(t *testing.T) {
	sessionID := "ses_empty_transcript"
	nowMs := time.Now().UnixMilli()

	mockSession := fmt.Sprintf(`{
		"id": "%s",
		"title": "Empty Session",
		"directory": "/Users/test/project",
		"time": {"created": %d, "updated": %d}
	}`, sessionID, nowMs-60000, nowMs-1000)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/session/" + sessionID:
			w.Write([]byte(mockSession))
		case "/session/" + sessionID + "/message":
			w.Write([]byte("[]")) // Empty messages
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	transcript, err := client.ExportSessionTranscript(sessionID)
	if err != nil {
		t.Fatalf("ExportSessionTranscript() error = %v", err)
	}

	// Empty messages should return empty string
	if transcript != "" {
		t.Errorf("Expected empty transcript for session with no messages, got: %s", transcript)
	}
}

// TestExportSessionTranscriptSessionError tests ExportSessionTranscript with session fetch error.
func TestExportSessionTranscriptSessionError(t *testing.T) {
	sessionID := "ses_error"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.ExportSessionTranscript(sessionID)
	if err == nil {
		t.Error("Expected error when session fetch fails")
	}
	if !strings.Contains(err.Error(), "failed to get session") {
		t.Errorf("Expected 'failed to get session' error, got: %v", err)
	}
}

// TestExportSessionTranscriptMessagesError tests ExportSessionTranscript with messages fetch error.
func TestExportSessionTranscriptMessagesError(t *testing.T) {
	sessionID := "ses_messages_error"
	nowMs := time.Now().UnixMilli()

	mockSession := fmt.Sprintf(`{
		"id": "%s",
		"title": "Test Session",
		"directory": "/Users/test/project",
		"time": {"created": %d, "updated": %d}
	}`, sessionID, nowMs-60000, nowMs-1000)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/session/" + sessionID:
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(mockSession))
		case "/session/" + sessionID + "/message":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.ExportSessionTranscript(sessionID)
	if err == nil {
		t.Error("Expected error when messages fetch fails")
	}
	if !strings.Contains(err.Error(), "failed to get messages") {
		t.Errorf("Expected 'failed to get messages' error, got: %v", err)
	}
}

// TestFormatMessagesAsTranscript tests the FormatMessagesAsTranscript function.
func TestFormatMessagesAsTranscript(t *testing.T) {
	nowMs := time.Now().UnixMilli()
	session := &Session{
		ID:        "ses_format_test",
		Title:     "Format Test Session",
		Directory: "/test/dir",
		Time: SessionTime{
			Created: nowMs - 60000,
			Updated: nowMs - 1000,
		},
		Summary: SessionSummary{
			Additions: 5,
			Deletions: 2,
			Files:     1,
		},
	}

	messages := []Message{
		{
			Info: MessageInfo{
				ID:        "msg_1",
				SessionID: "ses_format_test",
				Role:      "user",
				Time:      MessageTime{Created: nowMs - 30000},
			},
			Parts: []MessagePart{
				{Type: "text", Text: "Test user message"},
			},
		},
		{
			Info: MessageInfo{
				ID:        "msg_2",
				SessionID: "ses_format_test",
				Role:      "assistant",
				Time:      MessageTime{Created: nowMs - 25000, Completed: nowMs - 20000},
				Tokens:    &MessageToken{Input: 50, Output: 30},
				Cost:      0.001,
			},
			Parts: []MessagePart{
				{Type: "text", Text: "Test assistant response"},
				{Type: "tool-invocation", Text: ""},
			},
		},
	}

	transcript := FormatMessagesAsTranscript(session, messages)

	// Verify header
	if !strings.Contains(transcript, "# Session Transcript") {
		t.Error("Missing header")
	}

	// Verify session metadata
	if !strings.Contains(transcript, "**Title:** Format Test Session") {
		t.Error("Missing title")
	}
	if !strings.Contains(transcript, "**Session ID:** `ses_format_test`") {
		t.Error("Missing session ID")
	}
	if !strings.Contains(transcript, "**Directory:** `/test/dir`") {
		t.Error("Missing directory")
	}
	if !strings.Contains(transcript, "**Changes:** +5/-2 in 1 files") {
		t.Error("Missing changes summary")
	}

	// Verify messages
	if !strings.Contains(transcript, "## User") {
		t.Error("Missing user header")
	}
	if !strings.Contains(transcript, "Test user message") {
		t.Error("Missing user message content")
	}
	if !strings.Contains(transcript, "## Assistant") {
		t.Error("Missing assistant header")
	}
	if !strings.Contains(transcript, "Test assistant response") {
		t.Error("Missing assistant message content")
	}
	if !strings.Contains(transcript, "*Tokens:") {
		t.Error("Missing token info")
	}
	if !strings.Contains(transcript, "**Tools:**") {
		t.Error("Missing tools section")
	}
}
