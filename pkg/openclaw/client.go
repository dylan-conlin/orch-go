// Package openclaw provides a WebSocket JSON-RPC client for the OpenClaw gateway.
//
// OpenClaw is a multi-model AI agent platform with a WebSocket-based gateway API.
// This client implements the subset of methods needed by orch-go for agent execution:
// agent dispatch, completion polling, session management, and health checks.
//
// Protocol: The gateway uses a custom framing protocol over WebSocket:
//   - Client connects, receives connect.challenge event with nonce
//   - Client sends "connect" request with auth + nonce
//   - Server responds with hello-ok containing capabilities
//   - Subsequent requests use {type:"req", id, method, params} / {type:"res", id, ok, payload, error}
package openclaw

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// DefaultURL is the default OpenClaw gateway WebSocket URL.
const DefaultURL = "ws://127.0.0.1:18789"

// DefaultRequestTimeout is the default timeout for individual RPC requests.
const DefaultRequestTimeout = 30 * time.Second

// DefaultPollInterval is the default interval between agent.wait polls
// when using WaitForCompletion.
const DefaultPollInterval = 5 * time.Second

// Options configures an OpenClaw client.
type Options struct {
	URL            string        // WebSocket URL (default: ws://127.0.0.1:18789)
	Token          string        // Auth token for gateway
	RequestTimeout time.Duration // Per-request timeout (default: 30s)
}

// Client is a WebSocket JSON-RPC client for the OpenClaw gateway.
type Client struct {
	opts    Options
	conn    *websocket.Conn
	mu      sync.Mutex
	pending map[string]chan rawResponse
	done    chan struct{}
}

type rawResponse struct {
	OK      bool            `json:"ok"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   *ErrorShape     `json:"error,omitempty"`
}

// NewClient creates a new OpenClaw client with the given options.
func NewClient(opts Options) *Client {
	if opts.URL == "" {
		opts.URL = DefaultURL
	}
	if opts.RequestTimeout == 0 {
		opts.RequestTimeout = DefaultRequestTimeout
	}
	return &Client{
		opts:    opts,
		pending: make(map[string]chan rawResponse),
		done:    make(chan struct{}),
	}
}

// Connect establishes a WebSocket connection to the gateway and completes
// the challenge/response handshake. The context controls the connection timeout.
func (c *Client) Connect(ctx context.Context) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, c.opts.URL, nil)
	if err != nil {
		return fmt.Errorf("openclaw: dial failed: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.done = make(chan struct{})
	c.pending = make(map[string]chan rawResponse)
	c.mu.Unlock()

	// Start message reader in background
	go c.readLoop()

	// Wait for connect.challenge and complete handshake
	if err := c.handshake(ctx); err != nil {
		conn.Close()
		return err
	}

	return nil
}

// handshake waits for the connect.challenge event, then sends the connect
// request and waits for the hello-ok response.
func (c *Client) handshake(ctx context.Context) error {
	// The readLoop is already running and will dispatch responses.
	// We need to wait for the nonce to arrive via an event, then send connect.
	// Since events are handled in readLoop, we use a channel.

	nonceCh := make(chan string, 1)

	c.mu.Lock()
	c.pending["__nonce__"] = make(chan rawResponse, 1)
	c.mu.Unlock()

	// Set up a goroutine that waits for the nonce event
	go func() {
		c.mu.Lock()
		ch := c.pending["__nonce__"]
		c.mu.Unlock()
		select {
		case resp := <-ch:
			var payload struct {
				Nonce string `json:"nonce"`
			}
			json.Unmarshal(resp.Payload, &payload)
			nonceCh <- payload.Nonce
		case <-ctx.Done():
		}
	}()

	// Wait for nonce
	var nonce string
	select {
	case nonce = <-nonceCh:
		if nonce == "" {
			return fmt.Errorf("openclaw: empty nonce in connect challenge")
		}
	case <-ctx.Done():
		return fmt.Errorf("openclaw: connect timeout waiting for challenge: %w", ctx.Err())
	}

	// Clean up nonce pending entry
	c.mu.Lock()
	delete(c.pending, "__nonce__")
	c.mu.Unlock()

	// Send connect request
	connectParams := ConnectParams{
		MinProtocol: 1,
		MaxProtocol: 1,
		Client: ConnectClient{
			ID:       "orch-go",
			Version:  "1.0.0",
			Platform: "darwin",
			Mode:     "backend",
		},
		Auth: &ConnectAuth{
			Token: c.opts.Token,
		},
		Role:   "operator",
		Scopes: []string{"operator.admin"},
	}

	_, err := c.request(ctx, "connect", connectParams)
	if err != nil {
		return fmt.Errorf("openclaw: connect handshake failed: %w", err)
	}

	return nil
}

// readLoop reads WebSocket messages and dispatches them to pending requests.
func (c *Client) readLoop() {
	defer func() {
		c.mu.Lock()
		close(c.done)
		// Fail all pending requests
		for id, ch := range c.pending {
			select {
			case ch <- rawResponse{OK: false, Error: &ErrorShape{Code: "DISCONNECTED", Message: "connection closed"}}:
			default:
			}
			delete(c.pending, id)
		}
		c.mu.Unlock()
	}()

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		// Parse the frame type
		var frame struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &frame); err != nil {
			continue
		}

		switch frame.Type {
		case "event":
			var evt EventFrame
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Event == "connect.challenge" {
				// Route to handshake via special nonce channel
				c.mu.Lock()
				ch, ok := c.pending["__nonce__"]
				c.mu.Unlock()
				if ok {
					select {
					case ch <- rawResponse{OK: true, Payload: evt.Payload}:
					default:
					}
				}
			}
			// Other events (tick, etc.) are ignored for now

		case "res":
			var resp ResponseFrame
			if err := json.Unmarshal(raw, &resp); err != nil {
				continue
			}
			c.mu.Lock()
			ch, ok := c.pending[resp.ID]
			c.mu.Unlock()
			if ok {
				select {
				case ch <- rawResponse{OK: resp.OK, Payload: resp.Payload, Error: resp.Error}:
				default:
				}
				c.mu.Lock()
				delete(c.pending, resp.ID)
				c.mu.Unlock()
			}
		}
	}
}

// request sends a JSON-RPC request and waits for the response.
func (c *Client) request(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return nil, fmt.Errorf("openclaw: not connected")
	}

	id := generateID()
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("openclaw: marshal params: %w", err)
	}

	frame := RequestFrame{
		Type:   "req",
		ID:     id,
		Method: method,
		Params: json.RawMessage(paramsJSON),
	}

	ch := make(chan rawResponse, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	if err := conn.WriteJSON(frame); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("openclaw: send %s: %w", method, err)
	}

	select {
	case resp := <-ch:
		if !resp.OK {
			errMsg := "unknown error"
			errCode := "UNKNOWN"
			if resp.Error != nil {
				errMsg = resp.Error.Message
				errCode = resp.Error.Code
			}
			return nil, &RequestError{Code: errCode, Msg: errMsg}
		}
		return resp.Payload, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("openclaw: %s request timeout: %w", method, ctx.Err())
	case <-c.done:
		return nil, fmt.Errorf("openclaw: connection closed during %s request", method)
	}
}

// Agent sends a message to the OpenClaw agent and returns the run result.
// This is the primary method for spawning agent work.
func (c *Client) Agent(ctx context.Context, params AgentParams) (*AgentResult, error) {
	payload, err := c.request(ctx, "agent", params)
	if err != nil {
		return nil, err
	}
	var result AgentResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("openclaw: unmarshal agent result: %w", err)
	}
	return &result, nil
}

// AgentWait polls for the completion of a specific agent run.
// Returns when the run reaches a terminal state or the timeout expires.
// The gateway default timeout is 30s; for long-running agents, use WaitForCompletion instead.
func (c *Client) AgentWait(ctx context.Context, params AgentWaitParams) (*AgentWaitResult, error) {
	payload, err := c.request(ctx, "agent.wait", params)
	if err != nil {
		return nil, err
	}
	var result AgentWaitResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("openclaw: unmarshal agent.wait result: %w", err)
	}
	return &result, nil
}

// WaitForCompletion polls agent.wait repeatedly until the agent reaches a terminal
// state (ok, error) or the overall timeout expires. This handles the fact that
// agent.wait has a 30s default server-side timeout — for agents that run 30-60min,
// we need a polling loop. The pollInterval controls the delay between polls;
// pass 0 to use DefaultPollInterval.
func (c *Client) WaitForCompletion(ctx context.Context, runID string, timeout time.Duration, pollInterval ...time.Duration) (*AgentWaitResult, error) {
	deadline := time.Now().Add(timeout)
	pollTimeout := int64(60000) // 60s per poll, server returns "timeout" if not done

	interval := DefaultPollInterval
	if len(pollInterval) > 0 && pollInterval[0] > 0 {
		interval = pollInterval[0]
	}

	for {
		if time.Now().After(deadline) {
			return &AgentWaitResult{RunID: runID, Status: "timeout"}, nil
		}

		result, err := c.AgentWait(ctx, AgentWaitParams{
			RunID:     runID,
			TimeoutMs: pollTimeout,
		})
		if err != nil {
			return nil, err
		}

		// Terminal states: "ok" or "error"
		if result.Status == "ok" || result.Status == "error" {
			return result, nil
		}

		// "timeout" means the server-side wait expired, keep polling
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
			// continue polling
		}
	}
}

// SessionsList returns a list of sessions from the gateway.
func (c *Client) SessionsList(ctx context.Context, params SessionsListParams) (*SessionsListResult, error) {
	payload, err := c.request(ctx, "sessions.list", params)
	if err != nil {
		return nil, err
	}
	var result SessionsListResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("openclaw: unmarshal sessions.list result: %w", err)
	}
	return &result, nil
}

// SessionsDelete deletes a session by key.
func (c *Client) SessionsDelete(ctx context.Context, params SessionsDeleteParams) error {
	_, err := c.request(ctx, "sessions.delete", params)
	return err
}

// SessionsAbort aborts a running session.
func (c *Client) SessionsAbort(ctx context.Context, params SessionsAbortParams) error {
	_, err := c.request(ctx, "sessions.abort", params)
	return err
}

// Health checks gateway health. Returns true if the gateway is healthy.
func (c *Client) Health(ctx context.Context) (bool, error) {
	_, err := c.request(ctx, "health", nil)
	if err != nil {
		return false, err
	}
	return true, nil
}

// IsReachable performs a fast TCP probe to check if the gateway is listening.
func (c *Client) IsReachable() bool {
	u, err := url.Parse(c.opts.URL)
	if err != nil {
		return false
	}
	host := u.Host
	if host == "" {
		return false
	}
	if !strings.Contains(host, ":") {
		host += ":18789"
	}
	conn, err := net.DialTimeout("tcp", host, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// IsConnected returns true if the client has an active WebSocket connection.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	select {
	case <-c.done:
		return false
	default:
		return c.conn != nil
	}
}

// Close closes the WebSocket connection.
func (c *Client) Close() error {
	c.mu.Lock()
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()

	if conn != nil {
		return conn.Close()
	}
	return nil
}

// generateID returns a unique request ID.
var idCounter uint64
var idMu sync.Mutex

func generateID() string {
	idMu.Lock()
	idCounter++
	id := idCounter
	idMu.Unlock()
	return fmt.Sprintf("orch-%d-%d", time.Now().UnixNano(), id)
}
