package execution

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

// newTestServer creates a test HTTP server and returns a SessionClient wrapping it.
func newTestServer(t *testing.T, handler http.Handler) (SessionClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := NewOpenCodeAdapter(srv.URL)
	return client, srv
}

func TestOpenCodeAdapter_GetSession(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/session/sess-123", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(opencode.Session{
			ID:        "sess-123",
			Directory: "/tmp/test",
			Title:     "test session",
			Time:      opencode.SessionTime{Created: 1700000000000, Updated: 1700000060000},
			Summary:   opencode.SessionSummary{Additions: 10, Deletions: 2, Files: 3},
			Metadata:  map[string]string{"skill": "feature-impl"},
		})
	})

	client, _ := newTestServer(t, mux)
	ctx := context.Background()

	info, err := client.GetSession(ctx, SessionHandle("sess-123"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ID != "sess-123" {
		t.Errorf("got ID %q, want %q", info.ID, "sess-123")
	}
	if info.Directory != "/tmp/test" {
		t.Errorf("got Directory %q, want %q", info.Directory, "/tmp/test")
	}
	if info.Title != "test session" {
		t.Errorf("got Title %q, want %q", info.Title, "test session")
	}
	if info.Summary.Additions != 10 {
		t.Errorf("got Additions %d, want %d", info.Summary.Additions, 10)
	}
	if info.Metadata["skill"] != "feature-impl" {
		t.Errorf("got Metadata[skill] %q, want %q", info.Metadata["skill"], "feature-impl")
	}
}

func TestOpenCodeAdapter_ListSessions(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
		sessions := []opencode.Session{
			{ID: "s1", Title: "first", Time: opencode.SessionTime{Created: 1700000000000}},
			{ID: "s2", Title: "second", Time: opencode.SessionTime{Created: 1700000010000}},
		}
		json.NewEncoder(w).Encode(sessions)
	})

	client, _ := newTestServer(t, mux)
	ctx := context.Background()

	sessions, err := client.ListSessions(ctx, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(sessions))
	}
	if sessions[0].ID != "s1" {
		t.Errorf("got ID %q, want %q", sessions[0].ID, "s1")
	}
}

func TestOpenCodeAdapter_GetMessages(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/session/sess-123/message", func(w http.ResponseWriter, r *http.Request) {
		messages := []opencode.Message{
			{
				Info: opencode.MessageInfo{
					ID:        "msg-1",
					SessionID: "sess-123",
					Role:      "user",
					Time:      opencode.MessageTime{Created: 1700000000000},
				},
				Parts: []opencode.MessagePart{
					{Type: "text", Text: "hello"},
				},
			},
			{
				Info: opencode.MessageInfo{
					ID:        "msg-2",
					SessionID: "sess-123",
					Role:      "assistant",
					Time:      opencode.MessageTime{Created: 1700000001000, Completed: 1700000005000},
					Finish:    "stop",
					Cost:      0.01,
					Tokens:    &opencode.MessageToken{Input: 100, Output: 200, Reasoning: 50, Cache: &opencode.TokenCache{Read: 30}},
				},
				Parts: []opencode.MessagePart{
					{Type: "text", Text: "hi there"},
				},
			},
		}
		json.NewEncoder(w).Encode(messages)
	})

	client, _ := newTestServer(t, mux)
	ctx := context.Background()

	messages, err := client.GetMessages(ctx, SessionHandle("sess-123"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("got %d messages, want 2", len(messages))
	}

	// Check user message
	if messages[0].Role != "user" {
		t.Errorf("got Role %q, want %q", messages[0].Role, "user")
	}
	if messages[0].Parts[0].Text != "hello" {
		t.Errorf("got Text %q, want %q", messages[0].Parts[0].Text, "hello")
	}

	// Check assistant message with tokens
	assistant := messages[1]
	if assistant.Tokens == nil {
		t.Fatal("expected non-nil Tokens")
	}
	if assistant.Tokens.Input != 100 {
		t.Errorf("got Input %d, want %d", assistant.Tokens.Input, 100)
	}
	if assistant.Tokens.CacheRead != 30 {
		t.Errorf("got CacheRead %d, want %d", assistant.Tokens.CacheRead, 30)
	}
	if assistant.Cost != 0.01 {
		t.Errorf("got Cost %f, want %f", assistant.Cost, 0.01)
	}
	if assistant.Finish != "stop" {
		t.Errorf("got Finish %q, want %q", assistant.Finish, "stop")
	}
}

func TestOpenCodeAdapter_GetSessionTokens(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/session/sess-123/message", func(w http.ResponseWriter, r *http.Request) {
		messages := []opencode.Message{
			{
				Info: opencode.MessageInfo{
					ID:     "msg-1",
					Role:   "assistant",
					Tokens: &opencode.MessageToken{Input: 100, Output: 50, Reasoning: 25},
				},
			},
			{
				Info: opencode.MessageInfo{
					ID:     "msg-2",
					Role:   "assistant",
					Tokens: &opencode.MessageToken{Input: 200, Output: 100, Cache: &opencode.TokenCache{Read: 50}},
				},
			},
		}
		json.NewEncoder(w).Encode(messages)
	})

	client, _ := newTestServer(t, mux)
	ctx := context.Background()

	stats, err := client.GetSessionTokens(ctx, SessionHandle("sess-123"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if stats.InputTokens != 300 {
		t.Errorf("got InputTokens %d, want 300", stats.InputTokens)
	}
	if stats.OutputTokens != 150 {
		t.Errorf("got OutputTokens %d, want 150", stats.OutputTokens)
	}
	if stats.ReasoningTokens != 25 {
		t.Errorf("got ReasoningTokens %d, want 25", stats.ReasoningTokens)
	}
	if stats.CacheReadTokens != 50 {
		t.Errorf("got CacheReadTokens %d, want 50", stats.CacheReadTokens)
	}
	if stats.TotalTokens != 475 {
		t.Errorf("got TotalTokens %d, want 475", stats.TotalTokens)
	}
}

func TestOpenCodeAdapter_GetSessionStatus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/session/status", func(w http.ResponseWriter, r *http.Request) {
		result := map[string]opencode.SessionStatusInfo{
			"sess-123": {Type: "busy", Message: "processing"},
		}
		json.NewEncoder(w).Encode(result)
	})

	client, _ := newTestServer(t, mux)
	ctx := context.Background()

	status, err := client.GetSessionStatus(ctx, SessionHandle("sess-123"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Type != "busy" {
		t.Errorf("got Type %q, want %q", status.Type, "busy")
	}
	if !status.IsBusy() {
		t.Error("expected IsBusy() to be true")
	}
}

func TestOpenCodeAdapter_IsReachable(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	client, _ := newTestServer(t, mux)
	ctx := context.Background()

	if !client.IsReachable(ctx) {
		t.Error("expected IsReachable to return true for running server")
	}
}

func TestOpenCodeAdapter_DeleteSession(t *testing.T) {
	deleted := false
	mux := http.NewServeMux()
	mux.HandleFunc("/session/sess-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		}
	})

	client, _ := newTestServer(t, mux)
	ctx := context.Background()

	err := client.DeleteSession(ctx, SessionHandle("sess-123"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected DELETE to be called")
	}
}

func TestOpenCodeAdapter_CreateSession(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			json.NewEncoder(w).Encode(opencode.CreateSessionResponse{
				ID:    "new-sess-456",
				Title: "test",
			})
		}
	})

	client, _ := newTestServer(t, mux)
	ctx := context.Background()

	handle, err := client.CreateSession(ctx, SessionRequest{
		Title:     "test",
		Directory: "/tmp/project",
		Model:     "anthropic/claude-opus-4-5-20251101",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handle != SessionHandle("new-sess-456") {
		t.Errorf("got handle %q, want %q", handle, "new-sess-456")
	}
}

func TestAggregateTokens(t *testing.T) {
	messages := []Message{
		{Tokens: &TokenCount{Input: 100, Output: 50, Reasoning: 10, CacheRead: 20}},
		{Tokens: nil}, // should be skipped
		{Tokens: &TokenCount{Input: 200, Output: 100, Reasoning: 0, CacheRead: 30}},
	}

	stats := AggregateTokens(messages)
	if stats.InputTokens != 300 {
		t.Errorf("got InputTokens %d, want 300", stats.InputTokens)
	}
	if stats.OutputTokens != 150 {
		t.Errorf("got OutputTokens %d, want 150", stats.OutputTokens)
	}
	if stats.ReasoningTokens != 10 {
		t.Errorf("got ReasoningTokens %d, want 10", stats.ReasoningTokens)
	}
	if stats.CacheReadTokens != 50 {
		t.Errorf("got CacheReadTokens %d, want 50", stats.CacheReadTokens)
	}
	if stats.TotalTokens != 460 {
		t.Errorf("got TotalTokens %d, want 460", stats.TotalTokens)
	}
}

func TestSessionHandle_String(t *testing.T) {
	h := SessionHandle("test-id")
	if h.String() != "test-id" {
		t.Errorf("got %q, want %q", h.String(), "test-id")
	}
}

func TestCompletionStatus(t *testing.T) {
	complete := CompletionStatus{Status: "completed", Duration: 5 * time.Minute}
	if !complete.IsComplete() {
		t.Error("expected IsComplete() to be true")
	}
	if complete.IsError() {
		t.Error("expected IsError() to be false")
	}

	errStatus := CompletionStatus{Status: "error", Error: "something broke"}
	if errStatus.IsComplete() {
		t.Error("expected IsComplete() to be false")
	}
	if !errStatus.IsError() {
		t.Error("expected IsError() to be true")
	}
}

func TestConvertSession_TimestampConversion(t *testing.T) {
	s := &opencode.Session{
		ID:   "test",
		Time: opencode.SessionTime{Created: 1700000000000, Updated: 1700000060000},
	}
	info := convertSession(s)

	// 1700000000000ms = 1700000000s = 2023-11-14T22:13:20Z
	expectedCreated := time.Unix(1700000000, 0)
	if !info.Created.Equal(expectedCreated) {
		t.Errorf("got Created %v, want %v", info.Created, expectedCreated)
	}

	expectedUpdated := time.Unix(1700000060, 0)
	if !info.Updated.Equal(expectedUpdated) {
		t.Errorf("got Updated %v, want %v", info.Updated, expectedUpdated)
	}
}
