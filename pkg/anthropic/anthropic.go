// Package anthropic provides shared HTTP client utilities for Anthropic API calls.
//
// This package centralizes the common HTTP header setup required by the
// Anthropic OAuth API (usage, profile endpoints). All packages that make
// requests to the Anthropic API should use NewAPIRequest to ensure consistent
// headers.
package anthropic

import (
	"fmt"
	"net/http"
	"strings"
)

// API endpoints for Anthropic OAuth.
const (
	UsageEndpoint   = "https://api.anthropic.com/api/oauth/usage"
	ProfileEndpoint = "https://api.anthropic.com/api/oauth/profile"
	UserAgent       = "claude-code/2.0.32"
)

// BetaHeaders are required for OAuth tokens to work with Claude Code credentials.
var BetaHeaders = strings.Join([]string{
	"oauth-2025-04-20",
	"claude-code-20250219",
	"interleaved-thinking-2025-05-14",
	"fine-grained-tool-streaming-2025-05-14",
}, ",")

// NewAPIRequest creates an http.Request with all standard Anthropic API headers set.
// This is the single place where Authorization, anthropic-beta, Content-Type, Accept,
// and User-Agent headers are configured.
func NewAPIRequest(method, url, token string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", BetaHeaders)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	return req, nil
}
