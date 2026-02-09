// Package dialogue provides direct Anthropic Messages API access for
// lightweight dialogue turns used by orch dialogue.
package dialogue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/model"
)

const (
	// DefaultMessagesEndpoint is Anthropic's Messages API endpoint.
	DefaultMessagesEndpoint = "https://api.anthropic.com/v1/messages"
	// AnthropicVersion is the API version header required for Messages requests.
	AnthropicVersion = "2023-06-01"
	// DefaultMaxTokens is used when a request does not specify max tokens.
	DefaultMaxTokens = 1024

	defaultTimeout = 30 * time.Second

	oauthUserAgent      = "claude-cli/2.1.15 (external, cli)"
	oauthBetaHeader     = "claude-code-20250219,oauth-2025-04-20"
	oauthIdentityPrompt = "You are Claude Code, Anthropic's official CLI for Claude."
)

// AuthMode describes how the client authenticates with Anthropic.
type AuthMode string

const (
	// AuthModeAPIKey uses an Anthropic API key via x-api-key.
	AuthModeAPIKey AuthMode = "api_key"
	// AuthModeOAuth uses OpenCode/Claude Max OAuth credentials.
	AuthModeOAuth AuthMode = "oauth"
)

// Config configures a dialogue Messages API client.
//
// Authentication resolution order:
//  1. APIKey field
//  2. OAuthToken field
//  3. ANTHROPIC_API_KEY env var
//  4. OpenCode auth.json OAuth access token
type Config struct {
	Endpoint   string
	Model      string
	MaxTokens  int
	APIKey     string
	OAuthToken string
	HTTPClient *http.Client
}

// Message is a single dialogue turn passed to Anthropic Messages API.
type Message struct {
	Role    string
	Content string
}

// CompletionRequest is a single Messages API call.
type CompletionRequest struct {
	Model        string
	MaxTokens    int
	SystemPrompt string
	Messages     []Message
	Temperature  *float64
}

// Usage captures token usage returned by Anthropic.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// CompletionResponse is the parsed result from Anthropic Messages API.
type CompletionResponse struct {
	ID         string
	Model      string
	StopReason string
	Text       string
	Usage      Usage
}

// APIError wraps non-2xx Anthropic responses with parsed details.
type APIError struct {
	StatusCode int
	Type       string
	Message    string
	RequestID  string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Type != "" {
		return fmt.Sprintf("anthropic API error (%d, %s): %s", e.StatusCode, e.Type, e.Message)
	}
	return fmt.Sprintf("anthropic API error (%d): %s", e.StatusCode, e.Message)
}

// Client sends direct requests to Anthropic Messages API.
type Client struct {
	endpoint   string
	model      string
	maxTokens  int
	httpClient *http.Client
	authMode   AuthMode
	secret     string
}

// NewClient creates a new Messages API client.
func NewClient(cfg Config) (*Client, error) {
	creds, err := resolveCredentials(cfg)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = DefaultMessagesEndpoint
	}

	modelID := strings.TrimSpace(cfg.Model)
	if modelID == "" {
		modelID = model.Resolve("sonnet").ModelID
	}

	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = DefaultMaxTokens
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}

	return &Client{
		endpoint:   endpoint,
		model:      modelID,
		maxTokens:  maxTokens,
		httpClient: httpClient,
		authMode:   creds.mode,
		secret:     creds.secret,
	}, nil
}

// Complete performs one Anthropic Messages API call and returns assistant text.
func (c *Client) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages are required")
	}

	modelID := strings.TrimSpace(req.Model)
	if modelID == "" {
		modelID = c.model
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = c.maxTokens
	}

	apiMsgs := make([]apiMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		role := strings.TrimSpace(msg.Role)
		if role != "user" && role != "assistant" {
			return nil, fmt.Errorf("unsupported message role: %q", msg.Role)
		}
		apiMsgs = append(apiMsgs, apiMessage{Role: role, Content: msg.Content})
	}

	payload := messagesRequest{
		Model:       modelID,
		MaxTokens:   maxTokens,
		Messages:    apiMsgs,
		Temperature: req.Temperature,
	}

	if strings.TrimSpace(req.SystemPrompt) != "" {
		payload.System = append(payload.System, systemBlock{Type: "text", Text: req.SystemPrompt})
	}

	if c.authMode == AuthModeOAuth {
		payload.System = append([]systemBlock{{
			Type: "text",
			Text: oauthIdentityPrompt,
			CacheControl: &cacheControl{
				Type: "ephemeral",
			},
		}}, payload.System...)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, parseAPIError(resp.StatusCode, respBody)
	}

	var parsed messagesResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &CompletionResponse{
		ID:         parsed.ID,
		Model:      parsed.Model,
		StopReason: parsed.StopReason,
		Text:       extractText(parsed.Content),
		Usage:      parsed.Usage,
	}, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("anthropic-version", AnthropicVersion)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.authMode == AuthModeAPIKey {
		req.Header.Set("x-api-key", c.secret)
		req.Header.Set("User-Agent", "orch-go-dialogue/1.0")
		return
	}

	req.Header.Set("Authorization", "Bearer "+c.secret)
	req.Header.Set("anthropic-beta", oauthBetaHeader)
	req.Header.Set("anthropic-dangerous-direct-browser-access", "true")
	req.Header.Set("x-app", "cli")
	req.Header.Set("User-Agent", oauthUserAgent)
}

type credentials struct {
	mode   AuthMode
	secret string
}

func resolveCredentials(cfg Config) (credentials, error) {
	if key := strings.TrimSpace(cfg.APIKey); key != "" {
		return credentials{mode: AuthModeAPIKey, secret: key}, nil
	}

	if token := strings.TrimSpace(cfg.OAuthToken); token != "" {
		return credentials{mode: AuthModeOAuth, secret: token}, nil
	}

	if key := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")); key != "" {
		return credentials{mode: AuthModeAPIKey, secret: key}, nil
	}

	auth, err := account.LoadOpenCodeAuth()
	if err != nil {
		return credentials{}, fmt.Errorf("no dialogue credentials found: set ANTHROPIC_API_KEY or authenticate OpenCode (%w)", err)
	}

	token := strings.TrimSpace(auth.Anthropic.Access)
	if token == "" {
		return credentials{}, fmt.Errorf("no dialogue credentials found: OpenCode auth has empty anthropic access token")
	}

	if oauthTokenExpired(auth.Anthropic.Expires) {
		return credentials{}, fmt.Errorf("OpenCode OAuth token appears expired; restart OpenCode to refresh credentials")
	}

	return credentials{mode: AuthModeOAuth, secret: token}, nil
}

func oauthTokenExpired(expires int64) bool {
	if expires <= 0 {
		return false
	}
	// OpenCode auth stores milliseconds. Some legacy paths may use seconds.
	if expires > 1_000_000_000_000 {
		expires /= 1000
	}
	return time.Now().Unix() >= expires
}

func parseAPIError(statusCode int, body []byte) error {
	parsed := struct {
		Type      string `json:"type"`
		RequestID string `json:"request_id"`
		Error     struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}{}

	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Error.Message != "" {
		return &APIError{
			StatusCode: statusCode,
			Type:       parsed.Error.Type,
			Message:    parsed.Error.Message,
			RequestID:  parsed.RequestID,
		}
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(statusCode)
	}

	return &APIError{StatusCode: statusCode, Message: message}
}

func extractText(blocks []responseContentBlock) string {
	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		if block.Type == "text" && block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
}

type messagesRequest struct {
	Model       string        `json:"model"`
	MaxTokens   int           `json:"max_tokens"`
	System      []systemBlock `json:"system,omitempty"`
	Messages    []apiMessage  `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
}

type systemBlock struct {
	Type         string        `json:"type"`
	Text         string        `json:"text"`
	CacheControl *cacheControl `json:"cache_control,omitempty"`
}

type cacheControl struct {
	Type string `json:"type"`
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesResponse struct {
	ID         string                 `json:"id"`
	Model      string                 `json:"model"`
	StopReason string                 `json:"stop_reason"`
	Content    []responseContentBlock `json:"content"`
	Usage      Usage                  `json:"usage"`
}

type responseContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
