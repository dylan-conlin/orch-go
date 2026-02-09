package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestSendHeadlessPromptIncludesDirectoryAndModel(t *testing.T) {
	const (
		sessionID = "ses_prompt"
		directory = "/tmp/worktree"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/session/"+sessionID+"/prompt_async" {
			t.Fatalf("path = %s, want /session/%s/prompt_async", r.URL.Path, sessionID)
		}
		if got := r.Header.Get("x-opencode-directory"); got != directory {
			t.Fatalf("x-opencode-directory = %q, want %q", got, directory)
		}
		if got := r.Header.Get("x-opencode-env-ORCH_WORKER"); got != "1" {
			t.Fatalf("x-opencode-env-ORCH_WORKER = %q, want %q", got, "1")
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to decode JSON payload: %v", err)
		}

		if payload["agent"] != "build" {
			t.Fatalf("agent = %#v, want %q", payload["agent"], "build")
		}
		if payload["variant"] != "high" {
			t.Fatalf("variant = %#v, want %q", payload["variant"], "high")
		}

		model, ok := payload["model"].(map[string]any)
		if !ok {
			t.Fatalf("model payload = %#v, want object", payload["model"])
		}
		if model["providerID"] != "openai" {
			t.Fatalf("providerID = %#v, want %q", model["providerID"], "openai")
		}
		if model["modelID"] != "gpt-5" {
			t.Fatalf("modelID = %#v, want %q", model["modelID"], "gpt-5")
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := sendHeadlessPrompt(server.URL, sessionID, "hello", "openai/gpt-5", "high", directory)
	if err != nil {
		t.Fatalf("sendHeadlessPrompt() error = %v", err)
	}
}

func TestStartHeadlessSessionUsesRuntimeDirForPrompt(t *testing.T) {
	const (
		runtimeDir = "/tmp/worktree"
		title      = "spawn title"
		prompt     = "read spawn context"
	)

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/session":
			var reqBody map[string]any
			if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
				t.Fatalf("failed to decode create-session payload: %v", err)
			}
			if reqBody["title"] != title {
				t.Fatalf("title = %#v, want %q", reqBody["title"], title)
			}
			if reqBody["directory"] != runtimeDir {
				t.Fatalf("directory = %#v, want %q", reqBody["directory"], runtimeDir)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"ses_created"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/session/ses_created/prompt_async":
			if got := r.Header.Get("x-opencode-directory"); got != runtimeDir {
				t.Fatalf("x-opencode-directory = %q, want %q", got, runtimeDir)
			}
			var reqBody map[string]any
			if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
				t.Fatalf("failed to decode prompt payload: %v", err)
			}
			if reqBody["variant"] != "max" {
				t.Fatalf("variant = %#v, want %q", reqBody["variant"], "max")
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := opencode.NewClient(server.URL)
	cfg := &spawn.Config{
		ProjectDir: "/tmp/project",
		CWD:        runtimeDir,
		Model:      "anthropic/claude-opus-4",
		Variant:    "max",
	}

	result, err := startHeadlessSession(client, server.URL, title, prompt, cfg)
	if err != nil {
		t.Fatalf("startHeadlessSession() error = %v", err)
	}
	if result.SessionID != "ses_created" {
		t.Fatalf("session ID = %q, want %q", result.SessionID, "ses_created")
	}
	if result.cmd != nil || result.stdout != nil {
		t.Fatalf("headless result should not keep subprocess resources: %#v", result)
	}
	if requestCount != 2 {
		t.Fatalf("request count = %d, want 2", requestCount)
	}
}
