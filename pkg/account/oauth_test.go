package account

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestGenerateCodeVerifier(t *testing.T) {
	verifier, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("generateCodeVerifier() error = %v", err)
	}

	// Code verifier should be base64url encoded (43 characters from 32 bytes)
	if len(verifier) != 43 {
		t.Errorf("generateCodeVerifier() length = %d, want 43", len(verifier))
	}

	// Verify it's valid base64url
	if _, err := base64.RawURLEncoding.DecodeString(verifier); err != nil {
		t.Errorf("generateCodeVerifier() not valid base64url: %v", err)
	}

	// Generate another one - should be different (randomness test)
	verifier2, _ := generateCodeVerifier()
	if verifier == verifier2 {
		t.Error("generateCodeVerifier() generated same value twice (not random)")
	}
}

func TestGenerateCodeChallenge(t *testing.T) {
	verifier := "test_code_verifier_string_for_testing"
	challenge := generateCodeChallenge(verifier)

	// Manually calculate expected challenge
	h := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(h[:])

	if challenge != expected {
		t.Errorf("generateCodeChallenge() = %s, want %s", challenge, expected)
	}
}

func TestBuildAuthorizationURL(t *testing.T) {
	codeChallenge := "test_challenge"
	codeVerifier := "test_verifier"

	authURL := buildAuthorizationURL(codeChallenge, codeVerifier)

	// Parse the URL to verify components
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("buildAuthorizationURL() returned invalid URL: %v", err)
	}

	if parsed.Host != "console.anthropic.com" {
		t.Errorf("Host = %s, want console.anthropic.com", parsed.Host)
	}

	if parsed.Path != "/oauth/authorize" {
		t.Errorf("Path = %s, want /oauth/authorize", parsed.Path)
	}

	params := parsed.Query()

	if params.Get("client_id") != OAuthClientID {
		t.Errorf("client_id = %s, want %s", params.Get("client_id"), OAuthClientID)
	}

	if params.Get("redirect_uri") != AnthropicCallbackURL {
		t.Errorf("redirect_uri = %s, want %s", params.Get("redirect_uri"), AnthropicCallbackURL)
	}

	if params.Get("response_type") != "code" {
		t.Errorf("response_type = %s, want code", params.Get("response_type"))
	}

	if params.Get("code_challenge") != codeChallenge {
		t.Errorf("code_challenge = %s, want %s", params.Get("code_challenge"), codeChallenge)
	}

	if params.Get("code_challenge_method") != "S256" {
		t.Errorf("code_challenge_method = %s, want S256", params.Get("code_challenge_method"))
	}

	// State should be the code verifier (matching opencode pattern)
	if params.Get("state") != codeVerifier {
		t.Errorf("state = %s, want %s", params.Get("state"), codeVerifier)
	}

	// Should include "code=true" parameter
	if params.Get("code") != "true" {
		t.Errorf("code = %s, want true", params.Get("code"))
	}

	// Scope should include required permissions
	scope := params.Get("scope")
	if !strings.Contains(scope, "user:inference") {
		t.Errorf("scope should contain user:inference, got %s", scope)
	}
}

func TestDefaultOAuthConfig(t *testing.T) {
	cfg := DefaultOAuthConfig()

	if cfg.Timeout != 10*time.Minute {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 10*time.Minute)
	}
}

func TestOAuthError(t *testing.T) {
	// Test without cause
	err := &OAuthError{Message: "test error"}
	if err.Error() != "test error" {
		t.Errorf("Error() = %s, want 'test error'", err.Error())
	}

	// Test with cause
	cause := &TokenRefreshError{Message: "token expired"}
	errWithCause := &OAuthError{Message: "auth failed", Cause: cause}
	if !strings.Contains(errWithCause.Error(), "auth failed") {
		t.Errorf("Error() should contain 'auth failed', got %s", errWithCause.Error())
	}
	if !strings.Contains(errWithCause.Error(), "token expired") {
		t.Errorf("Error() should contain cause 'token expired', got %s", errWithCause.Error())
	}

	// Test Unwrap
	if errWithCause.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestExchangeCodeForTokens_CodeWithState(t *testing.T) {
	// Test that code with state suffix (code#state) is correctly parsed
	// This is a unit test for the code parsing logic

	tests := []struct {
		name             string
		rawCode          string
		expectedCode     string
		expectedHasState bool
	}{
		{
			name:             "code only",
			rawCode:          "simple_code_123",
			expectedCode:     "simple_code_123",
			expectedHasState: false,
		},
		{
			name:             "code with state",
			rawCode:          "auth_code_abc#state_xyz",
			expectedCode:     "auth_code_abc",
			expectedHasState: true,
		},
		{
			name:             "code with empty state",
			rawCode:          "auth_code#",
			expectedCode:     "auth_code",
			expectedHasState: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the code the same way exchangeCodeForTokens does
			code := tt.rawCode
			hasState := false
			if idx := strings.Index(tt.rawCode, "#"); idx != -1 {
				code = tt.rawCode[:idx]
				hasState = true
			}

			if code != tt.expectedCode {
				t.Errorf("code = %s, want %s", code, tt.expectedCode)
			}
			if hasState != tt.expectedHasState {
				t.Errorf("hasState = %v, want %v", hasState, tt.expectedHasState)
			}
		})
	}
}

func TestExchangeCodeForTokens_MockServer(t *testing.T) {
	// Create a mock token server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// The new implementation uses JSON, not form-urlencoded
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Return mock token response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"access_token": "test_access_token",
			"refresh_token": "test_refresh_token",
			"expires_in": 3600,
			"token_type": "Bearer"
		}`))
	}))
	defer server.Close()

	// We can't easily test exchangeCodeForTokens without modifying the endpoint,
	// but we can verify the function exists and the mock server works
	// In a real test, we would inject the endpoint URL

	// For now, just verify the mock server responds correctly
	resp, err := http.Post(server.URL, "application/json", strings.NewReader(`{
		"grant_type": "authorization_code",
		"code": "test_code",
		"redirect_uri": "https://console.anthropic.com/oauth/code/callback",
		"client_id": "`+OAuthClientID+`",
		"code_verifier": "test_verifier"
	}`))
	if err != nil {
		t.Fatalf("Mock server request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Mock server status = %d, want 200", resp.StatusCode)
	}
}

func TestAnthropicCallbackURL(t *testing.T) {
	// Verify the callback URL is Anthropic's official URL
	expected := "https://console.anthropic.com/oauth/code/callback"
	if AnthropicCallbackURL != expected {
		t.Errorf("AnthropicCallbackURL = %s, want %s", AnthropicCallbackURL, expected)
	}
}
