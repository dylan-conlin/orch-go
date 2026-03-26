package openclaw

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// mockGateway simulates an OpenClaw gateway for testing.
type mockGateway struct {
	t          *testing.T
	upgrader   websocket.Upgrader
	server     *httptest.Server
	mu         sync.Mutex
	handleFunc func(conn *websocket.Conn)
}

func newMockGateway(t *testing.T, handler func(conn *websocket.Conn)) *mockGateway {
	t.Helper()
	mg := &mockGateway{
		t: t,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		handleFunc: handler,
	}
	mg.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := mg.upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("mock gateway upgrade failed: %v", err)
			return
		}
		defer conn.Close()
		mg.handleFunc(conn)
	}))
	return mg
}

func (mg *mockGateway) wsURL() string {
	return "ws" + strings.TrimPrefix(mg.server.URL, "http")
}

func (mg *mockGateway) close() {
	mg.server.Close()
}

// defaultGatewayHandler implements the connect challenge + hello-ok + request/response flow.
func defaultGatewayHandler(t *testing.T, methodHandlers map[string]func(id string, params json.RawMessage) (bool, interface{}, *ErrorShape)) func(conn *websocket.Conn) {
	t.Helper()
	return func(conn *websocket.Conn) {
		// Send connect.challenge event
		challenge := EventFrame{
			Type:    "event",
			Event:   "connect.challenge",
			Payload: json.RawMessage(`{"nonce":"test-nonce-123"}`),
		}
		if err := conn.WriteJSON(challenge); err != nil {
			t.Errorf("failed to send challenge: %v", err)
			return
		}

		// Read connect request
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("failed to read connect: %v", err)
			return
		}
		var req RequestFrame
		if err := json.Unmarshal(raw, &req); err != nil {
			t.Errorf("failed to parse connect request: %v", err)
			return
		}
		if req.Method != "connect" {
			t.Errorf("expected connect method, got %s", req.Method)
			return
		}

		// Send hello-ok response
		helloPayload, _ := json.Marshal(HelloOk{
			Type:     "hello-ok",
			Protocol: 1,
			Server: HelloServer{
				Version: "test-1.0.0",
				ConnID:  "test-conn-id",
			},
			Features: HelloFeatures{
				Methods: []string{"agent", "agent.wait", "sessions.list", "sessions.delete", "sessions.abort", "health"},
				Events:  []string{"tick"},
			},
			Policy: HelloPolicy{
				MaxPayload:      25 * 1024 * 1024,
				MaxBufferedBytes: 50 * 1024 * 1024,
				TickIntervalMs:  30000,
			},
		})
		resp := ResponseFrame{
			Type:    "res",
			ID:      req.ID,
			OK:      true,
			Payload: json.RawMessage(helloPayload),
		}
		if err := conn.WriteJSON(resp); err != nil {
			t.Errorf("failed to send hello-ok: %v", err)
			return
		}

		// Handle subsequent requests
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return // connection closed
			}
			var reqFrame RequestFrame
			if err := json.Unmarshal(raw, &reqFrame); err != nil {
				continue
			}

			handler, ok := methodHandlers[reqFrame.Method]
			if !ok {
				errResp := ResponseFrame{
					Type: "res",
					ID:   reqFrame.ID,
					OK:   false,
					Error: &ErrorShape{
						Code:    "METHOD_NOT_FOUND",
						Message: "unknown method: " + reqFrame.Method,
					},
				}
				conn.WriteJSON(errResp)
				continue
			}

			ok2, payload, errShape := handler(reqFrame.ID, reqFrame.Params)
			var payloadJSON json.RawMessage
			if payload != nil {
				payloadJSON, _ = json.Marshal(payload)
			}
			respFrame := ResponseFrame{
				Type:    "res",
				ID:      reqFrame.ID,
				OK:      ok2,
				Payload: payloadJSON,
				Error:   errShape,
			}
			conn.WriteJSON(respFrame)
		}
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient(Options{
		URL:   "ws://127.0.0.1:18789",
		Token: "test-token",
	})
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.opts.URL != "ws://127.0.0.1:18789" {
		t.Errorf("expected URL ws://127.0.0.1:18789, got %s", c.opts.URL)
	}
}

func TestConnect(t *testing.T) {
	gw := newMockGateway(t, defaultGatewayHandler(t, nil))
	defer gw.close()

	c := NewClient(Options{
		URL:   gw.wsURL(),
		Token: "test-token",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer c.Close()

	if !c.IsConnected() {
		t.Error("expected client to be connected")
	}
}

func TestAgent(t *testing.T) {
	handlers := map[string]func(string, json.RawMessage) (bool, interface{}, *ErrorShape){
		"agent": func(id string, params json.RawMessage) (bool, interface{}, *ErrorShape) {
			var p AgentParams
			json.Unmarshal(params, &p)
			if p.Message == "" {
				return false, nil, &ErrorShape{Code: "INVALID_REQUEST", Message: "message required"}
			}
			return true, AgentResult{
				RunID:      "run-abc-123",
				SessionKey: "session-key-xyz",
			}, nil
		},
	}

	gw := newMockGateway(t, defaultGatewayHandler(t, handlers))
	defer gw.close()

	c := NewClient(Options{URL: gw.wsURL(), Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer c.Close()

	result, err := c.Agent(ctx, AgentParams{
		Message:        "Implement the feature",
		IdempotencyKey: "idem-001",
	})
	if err != nil {
		t.Fatalf("Agent failed: %v", err)
	}
	if result.RunID != "run-abc-123" {
		t.Errorf("expected RunID run-abc-123, got %s", result.RunID)
	}
	if result.SessionKey != "session-key-xyz" {
		t.Errorf("expected SessionKey session-key-xyz, got %s", result.SessionKey)
	}
}

func TestAgentWithExtraSystemPrompt(t *testing.T) {
	var receivedPrompt string
	handlers := map[string]func(string, json.RawMessage) (bool, interface{}, *ErrorShape){
		"agent": func(id string, params json.RawMessage) (bool, interface{}, *ErrorShape) {
			var p AgentParams
			json.Unmarshal(params, &p)
			receivedPrompt = p.ExtraSystemPrompt
			return true, AgentResult{RunID: "run-1", SessionKey: "sk-1"}, nil
		},
	}

	gw := newMockGateway(t, defaultGatewayHandler(t, handlers))
	defer gw.close()

	c := NewClient(Options{URL: gw.wsURL(), Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Connect(ctx)
	defer c.Close()

	c.Agent(ctx, AgentParams{
		Message:           "do something",
		IdempotencyKey:    "idem-002",
		ExtraSystemPrompt: "You are a coding assistant.",
	})

	if receivedPrompt != "You are a coding assistant." {
		t.Errorf("expected extraSystemPrompt to be passed through, got %q", receivedPrompt)
	}
}

func TestAgentWait(t *testing.T) {
	handlers := map[string]func(string, json.RawMessage) (bool, interface{}, *ErrorShape){
		"agent.wait": func(id string, params json.RawMessage) (bool, interface{}, *ErrorShape) {
			var p AgentWaitParams
			json.Unmarshal(params, &p)
			return true, AgentWaitResult{
				RunID:     p.RunID,
				Status:    "ok",
				StartedAt: time.Now().Add(-5 * time.Minute).UnixMilli(),
				EndedAt:   time.Now().UnixMilli(),
			}, nil
		},
	}

	gw := newMockGateway(t, defaultGatewayHandler(t, handlers))
	defer gw.close()

	c := NewClient(Options{URL: gw.wsURL(), Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Connect(ctx)
	defer c.Close()

	result, err := c.AgentWait(ctx, AgentWaitParams{
		RunID:     "run-abc-123",
		TimeoutMs: 60000,
	})
	if err != nil {
		t.Fatalf("AgentWait failed: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("expected status ok, got %s", result.Status)
	}
	if result.RunID != "run-abc-123" {
		t.Errorf("expected RunID run-abc-123, got %s", result.RunID)
	}
}

func TestAgentWaitTimeout(t *testing.T) {
	handlers := map[string]func(string, json.RawMessage) (bool, interface{}, *ErrorShape){
		"agent.wait": func(id string, params json.RawMessage) (bool, interface{}, *ErrorShape) {
			return true, AgentWaitResult{
				RunID:  "run-1",
				Status: "timeout",
			}, nil
		},
	}

	gw := newMockGateway(t, defaultGatewayHandler(t, handlers))
	defer gw.close()

	c := NewClient(Options{URL: gw.wsURL(), Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Connect(ctx)
	defer c.Close()

	result, err := c.AgentWait(ctx, AgentWaitParams{RunID: "run-1", TimeoutMs: 1000})
	if err != nil {
		t.Fatalf("AgentWait failed: %v", err)
	}
	if result.Status != "timeout" {
		t.Errorf("expected timeout status, got %s", result.Status)
	}
}

func TestWaitForCompletion(t *testing.T) {
	callCount := 0
	handlers := map[string]func(string, json.RawMessage) (bool, interface{}, *ErrorShape){
		"agent.wait": func(id string, params json.RawMessage) (bool, interface{}, *ErrorShape) {
			callCount++
			if callCount < 3 {
				return true, AgentWaitResult{
					RunID:  "run-1",
					Status: "timeout",
				}, nil
			}
			return true, AgentWaitResult{
				RunID:     "run-1",
				Status:    "ok",
				StartedAt: time.Now().Add(-10 * time.Minute).UnixMilli(),
				EndedAt:   time.Now().UnixMilli(),
			}, nil
		},
	}

	gw := newMockGateway(t, defaultGatewayHandler(t, handlers))
	defer gw.close()

	c := NewClient(Options{URL: gw.wsURL(), Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c.Connect(ctx)
	defer c.Close()

	result, err := c.WaitForCompletion(ctx, "run-1", 30*time.Minute, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForCompletion failed: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("expected ok status, got %s", result.Status)
	}
	if callCount < 3 {
		t.Errorf("expected at least 3 poll calls, got %d", callCount)
	}
}

func TestSessionsList(t *testing.T) {
	handlers := map[string]func(string, json.RawMessage) (bool, interface{}, *ErrorShape){
		"sessions.list": func(id string, params json.RawMessage) (bool, interface{}, *ErrorShape) {
			return true, SessionsListResult{
				Sessions: []SessionEntry{
					{Key: "session-1", AgentID: "agent-a", Title: "Test session"},
					{Key: "session-2", AgentID: "agent-b", Title: "Another session"},
				},
			}, nil
		},
	}

	gw := newMockGateway(t, defaultGatewayHandler(t, handlers))
	defer gw.close()

	c := NewClient(Options{URL: gw.wsURL(), Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Connect(ctx)
	defer c.Close()

	result, err := c.SessionsList(ctx, SessionsListParams{})
	if err != nil {
		t.Fatalf("SessionsList failed: %v", err)
	}
	if len(result.Sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(result.Sessions))
	}
	if result.Sessions[0].Key != "session-1" {
		t.Errorf("expected session-1, got %s", result.Sessions[0].Key)
	}
}

func TestSessionsDelete(t *testing.T) {
	var deletedKey string
	handlers := map[string]func(string, json.RawMessage) (bool, interface{}, *ErrorShape){
		"sessions.delete": func(id string, params json.RawMessage) (bool, interface{}, *ErrorShape) {
			var p SessionsDeleteParams
			json.Unmarshal(params, &p)
			deletedKey = p.Key
			return true, map[string]interface{}{"ok": true}, nil
		},
	}

	gw := newMockGateway(t, defaultGatewayHandler(t, handlers))
	defer gw.close()

	c := NewClient(Options{URL: gw.wsURL(), Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Connect(ctx)
	defer c.Close()

	err := c.SessionsDelete(ctx, SessionsDeleteParams{Key: "session-to-delete"})
	if err != nil {
		t.Fatalf("SessionsDelete failed: %v", err)
	}
	if deletedKey != "session-to-delete" {
		t.Errorf("expected session-to-delete, got %s", deletedKey)
	}
}

func TestSessionsAbort(t *testing.T) {
	var abortedKey string
	handlers := map[string]func(string, json.RawMessage) (bool, interface{}, *ErrorShape){
		"sessions.abort": func(id string, params json.RawMessage) (bool, interface{}, *ErrorShape) {
			var p SessionsAbortParams
			json.Unmarshal(params, &p)
			abortedKey = p.Key
			return true, map[string]interface{}{"ok": true}, nil
		},
	}

	gw := newMockGateway(t, defaultGatewayHandler(t, handlers))
	defer gw.close()

	c := NewClient(Options{URL: gw.wsURL(), Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Connect(ctx)
	defer c.Close()

	err := c.SessionsAbort(ctx, SessionsAbortParams{Key: "session-to-abort"})
	if err != nil {
		t.Fatalf("SessionsAbort failed: %v", err)
	}
	if abortedKey != "session-to-abort" {
		t.Errorf("expected session-to-abort, got %s", abortedKey)
	}
}

func TestHealth(t *testing.T) {
	handlers := map[string]func(string, json.RawMessage) (bool, interface{}, *ErrorShape){
		"health": func(id string, params json.RawMessage) (bool, interface{}, *ErrorShape) {
			return true, map[string]interface{}{
				"status": "ok",
				"uptime": 3600,
			}, nil
		},
	}

	gw := newMockGateway(t, defaultGatewayHandler(t, handlers))
	defer gw.close()

	c := NewClient(Options{URL: gw.wsURL(), Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Connect(ctx)
	defer c.Close()

	healthy, err := c.Health(ctx)
	if err != nil {
		t.Fatalf("Health failed: %v", err)
	}
	if !healthy {
		t.Error("expected healthy")
	}
}

func TestIsReachable(t *testing.T) {
	gw := newMockGateway(t, defaultGatewayHandler(t, nil))
	defer gw.close()

	c := NewClient(Options{URL: gw.wsURL(), Token: "test-token"})
	if !c.IsReachable() {
		t.Error("expected reachable when server is running")
	}

	gw.close()
	// After close, should not be reachable
	if c.IsReachable() {
		t.Error("expected not reachable after server close")
	}
}

func TestConnectContextCanceled(t *testing.T) {
	// Server that never responds with challenge
	gw := newMockGateway(t, func(conn *websocket.Conn) {
		time.Sleep(10 * time.Second)
	})
	defer gw.close()

	c := NewClient(Options{URL: gw.wsURL(), Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := c.Connect(ctx)
	if err == nil {
		t.Fatal("expected error from cancelled context")
		c.Close()
	}
}

func TestRequestError(t *testing.T) {
	handlers := map[string]func(string, json.RawMessage) (bool, interface{}, *ErrorShape){
		"agent": func(id string, params json.RawMessage) (bool, interface{}, *ErrorShape) {
			return false, nil, &ErrorShape{
				Code:    "INVALID_REQUEST",
				Message: "provider/model overrides are not authorized",
			}
		},
	}

	gw := newMockGateway(t, defaultGatewayHandler(t, handlers))
	defer gw.close()

	c := NewClient(Options{URL: gw.wsURL(), Token: "test-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Connect(ctx)
	defer c.Close()

	_, err := c.Agent(ctx, AgentParams{
		Message:        "test",
		IdempotencyKey: "idem-err",
		Provider:       "anthropic",
	})
	if err == nil {
		t.Fatal("expected error from gateway")
	}
	gwErr, ok := err.(*RequestError)
	if !ok {
		t.Fatalf("expected *RequestError, got %T", err)
	}
	if gwErr.Code != "INVALID_REQUEST" {
		t.Errorf("expected INVALID_REQUEST code, got %s", gwErr.Code)
	}
}
