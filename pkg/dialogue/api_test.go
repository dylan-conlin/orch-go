package dialogue

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompleteAPIKeyAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != "test-api-key" {
			t.Fatalf("x-api-key = %q, want %q", got, "test-api-key")
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if got := body["model"]; got != "claude-sonnet-4-5-20250929" {
			t.Fatalf("model = %v, want claude-sonnet-4-5-20250929", got)
		}
		if got := int(body["max_tokens"].(float64)); got != DefaultMaxTokens {
			t.Fatalf("max_tokens = %d, want %d", got, DefaultMaxTokens)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_1","model":"claude-sonnet-4-5-20250929","stop_reason":"end_turn","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":12,"output_tokens":3}}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Endpoint:   server.URL,
		APIKey:     "test-api-key",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	resp, err := client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "Reply with ok."}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if resp.Text != "ok" {
		t.Fatalf("resp.Text = %q, want %q", resp.Text, "ok")
	}
	if resp.Usage.InputTokens != 12 || resp.Usage.OutputTokens != 3 {
		t.Fatalf("usage = %+v, want input=12 output=3", resp.Usage)
	}
}

func TestCompleteOAuthAuthAddsStealthHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer oauth-token" {
			t.Fatalf("Authorization = %q, want %q", got, "Bearer oauth-token")
		}
		if got := r.Header.Get("anthropic-beta"); got != oauthBetaHeader {
			t.Fatalf("anthropic-beta = %q, want %q", got, oauthBetaHeader)
		}
		if got := r.Header.Get("x-app"); got != "cli" {
			t.Fatalf("x-app = %q, want %q", got, "cli")
		}
		if got := r.Header.Get("anthropic-dangerous-direct-browser-access"); got != "true" {
			t.Fatalf("anthropic-dangerous-direct-browser-access = %q, want %q", got, "true")
		}
		if got := r.Header.Get("User-Agent"); got != oauthUserAgent {
			t.Fatalf("User-Agent = %q, want %q", got, oauthUserAgent)
		}

		var body struct {
			System []struct {
				Type         string `json:"type"`
				Text         string `json:"text"`
				CacheControl struct {
					Type string `json:"type"`
				} `json:"cache_control"`
			} `json:"system"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if len(body.System) != 2 {
			t.Fatalf("len(system) = %d, want 2", len(body.System))
		}
		if body.System[0].Text != oauthIdentityPrompt {
			t.Fatalf("system[0].text = %q, want %q", body.System[0].Text, oauthIdentityPrompt)
		}
		if body.System[0].CacheControl.Type != "ephemeral" {
			t.Fatalf("system[0].cache_control.type = %q, want %q", body.System[0].CacheControl.Type, "ephemeral")
		}
		if body.System[1].Text != "Ask one question." {
			t.Fatalf("system[1].text = %q, want %q", body.System[1].Text, "Ask one question.")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_2","model":"claude-sonnet-4-5-20250929","stop_reason":"end_turn","content":[{"type":"text","text":"what is the data shape?"}],"usage":{"input_tokens":20,"output_tokens":8}}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Endpoint:   server.URL,
		OAuthToken: "oauth-token",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	resp, err := client.Complete(context.Background(), CompletionRequest{
		SystemPrompt: "Ask one question.",
		Messages:     []Message{{Role: "user", Content: "Start"}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if resp.Text == "" {
		t.Fatal("resp.Text should not be empty")
	}
}

func TestCompleteReturnsParsedAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"credential rejected"},"request_id":"req_123"}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Endpoint:   server.URL,
		APIKey:     "test-api-key",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("Complete() error = nil, want APIError")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusBadRequest)
	}
	if apiErr.Type != "invalid_request_error" {
		t.Fatalf("Type = %q, want %q", apiErr.Type, "invalid_request_error")
	}
	if apiErr.RequestID != "req_123" {
		t.Fatalf("RequestID = %q, want %q", apiErr.RequestID, "req_123")
	}
}

func TestNewClientUsesEnvAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "env-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != "env-key" {
			t.Fatalf("x-api-key = %q, want %q", got, "env-key")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_3","model":"claude-sonnet-4-5-20250929","stop_reason":"end_turn","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{Endpoint: server.URL, HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
}

func TestCompleteRejectsUnsupportedRole(t *testing.T) {
	client, err := NewClient(Config{APIKey: "test-key"})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "system", Content: "not allowed"}},
	})
	if err == nil {
		t.Fatal("Complete() error = nil, want validation error")
	}
}
