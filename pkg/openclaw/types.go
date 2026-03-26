package openclaw

import "encoding/json"

// --- Protocol Framing ---

// RequestFrame is a client-to-server RPC request.
type RequestFrame struct {
	Type   string          `json:"type"`   // always "req"
	ID     string          `json:"id"`     // unique request ID
	Method string          `json:"method"` // RPC method name
	Params json.RawMessage `json:"params,omitempty"`
}

// ResponseFrame is a server-to-client RPC response.
type ResponseFrame struct {
	Type    string          `json:"type"`             // always "res"
	ID      string          `json:"id"`               // matches request ID
	OK      bool            `json:"ok"`               // true if successful
	Payload json.RawMessage `json:"payload,omitempty"` // response data
	Error   *ErrorShape     `json:"error,omitempty"`   // error details if !ok
}

// EventFrame is a server-to-client push event.
type EventFrame struct {
	Type    string          `json:"type"`             // always "event"
	Event   string          `json:"event"`            // event name
	Payload json.RawMessage `json:"payload,omitempty"` // event data
	Seq     *int            `json:"seq,omitempty"`     // sequence number
}

// ErrorShape is the standard gateway error format.
type ErrorShape struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RequestError is returned when the gateway responds with ok=false.
type RequestError struct {
	Code string
	Msg  string
}

func (e *RequestError) Error() string {
	return "openclaw: " + e.Code + ": " + e.Msg
}

// --- Connect Handshake ---

// ConnectParams is sent as the "connect" request after receiving the challenge nonce.
type ConnectParams struct {
	MinProtocol int            `json:"minProtocol"`
	MaxProtocol int            `json:"maxProtocol"`
	Client      ConnectClient  `json:"client"`
	Auth        *ConnectAuth   `json:"auth,omitempty"`
	Role        string         `json:"role,omitempty"`
	Scopes      []string       `json:"scopes,omitempty"`
	Caps        []string       `json:"caps,omitempty"`
}

// ConnectClient identifies the connecting client.
type ConnectClient struct {
	ID       string `json:"id"`
	Version  string `json:"version"`
	Platform string `json:"platform"`
	Mode     string `json:"mode"`
}

// ConnectAuth carries authentication credentials.
type ConnectAuth struct {
	Token    string `json:"token,omitempty"`
	Password string `json:"password,omitempty"`
}

// HelloOk is the server response to a successful connect.
type HelloOk struct {
	Type     string        `json:"type"`     // "hello-ok"
	Protocol int           `json:"protocol"`
	Server   HelloServer   `json:"server"`
	Features HelloFeatures `json:"features"`
	Policy   HelloPolicy   `json:"policy"`
}

// HelloServer contains gateway server info.
type HelloServer struct {
	Version string `json:"version"`
	ConnID  string `json:"connId"`
}

// HelloFeatures lists supported methods and events.
type HelloFeatures struct {
	Methods []string `json:"methods"`
	Events  []string `json:"events"`
}

// HelloPolicy contains connection policy limits.
type HelloPolicy struct {
	MaxPayload      int `json:"maxPayload"`
	MaxBufferedBytes int `json:"maxBufferedBytes"`
	TickIntervalMs  int `json:"tickIntervalMs"`
}

// --- Agent Methods ---

// AgentParams is the request for the "agent" method.
type AgentParams struct {
	Message           string `json:"message"`
	AgentID           string `json:"agentId,omitempty"`
	Provider          string `json:"provider,omitempty"`
	Model             string `json:"model,omitempty"`
	SessionKey        string `json:"sessionKey,omitempty"`
	ExtraSystemPrompt string `json:"extraSystemPrompt,omitempty"`
	IdempotencyKey    string `json:"idempotencyKey"`
	Lane              string `json:"lane,omitempty"`
	Timeout           int    `json:"timeout,omitempty"`
	Label             string `json:"label,omitempty"`
}

// AgentResult is the response from the "agent" method.
type AgentResult struct {
	RunID      string `json:"runId"`
	SessionKey string `json:"sessionKey,omitempty"`
	Status     string `json:"status,omitempty"`
}

// AgentWaitParams is the request for the "agent.wait" method.
type AgentWaitParams struct {
	RunID     string `json:"runId"`
	TimeoutMs int64  `json:"timeoutMs,omitempty"` // default 30000 (30s)
}

// AgentWaitResult is the response from "agent.wait".
type AgentWaitResult struct {
	RunID     string `json:"runId"`
	Status    string `json:"status"` // "ok", "error", "timeout"
	StartedAt int64  `json:"startedAt,omitempty"`
	EndedAt   int64  `json:"endedAt,omitempty"`
	Error     string `json:"error,omitempty"`
}

// --- Session Methods ---

// SessionsListParams is the request for "sessions.list".
type SessionsListParams struct {
	Limit         int    `json:"limit,omitempty"`
	ActiveMinutes int    `json:"activeMinutes,omitempty"`
	AgentID       string `json:"agentId,omitempty"`
	Label         string `json:"label,omitempty"`
	SpawnedBy     string `json:"spawnedBy,omitempty"`
}

// SessionsListResult is the response from "sessions.list".
type SessionsListResult struct {
	Sessions []SessionEntry `json:"sessions"`
}

// SessionEntry represents a session in the list response.
type SessionEntry struct {
	Key       string `json:"key"`
	AgentID   string `json:"agentId,omitempty"`
	Title     string `json:"title,omitempty"`
	Label     string `json:"label,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	SpawnedBy string `json:"spawnedBy,omitempty"`
}

// SessionsDeleteParams is the request for "sessions.delete".
type SessionsDeleteParams struct {
	Key             string `json:"key"`
	DeleteTranscript *bool  `json:"deleteTranscript,omitempty"`
}

// SessionsAbortParams is the request for "sessions.abort".
type SessionsAbortParams struct {
	Key   string `json:"key"`
	RunID string `json:"runId,omitempty"`
}
