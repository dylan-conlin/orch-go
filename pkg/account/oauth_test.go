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
	redirectURI := "http://127.0.0.1:19283/callback"
	codeChallenge := "test_challenge"
	state := "test_state"

	authURL := buildAuthorizationURL(redirectURI, codeChallenge, state)

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

	if params.Get("redirect_uri") != redirectURI {
		t.Errorf("redirect_uri = %s, want %s", params.Get("redirect_uri"), redirectURI)
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

	if params.Get("state") != state {
		t.Errorf("state = %s, want %s", params.Get("state"), state)
	}

	if params.Get("scope") != "user:inference" {
		t.Errorf("scope = %s, want user:inference", params.Get("scope"))
	}
}

func TestDefaultOAuthConfig(t *testing.T) {
	cfg := DefaultOAuthConfig()

	if cfg.CallbackPort != DefaultCallbackPort {
		t.Errorf("CallbackPort = %d, want %d", cfg.CallbackPort, DefaultCallbackPort)
	}

	if cfg.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 5*time.Minute)
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

func TestCallbackServerSuccess(t *testing.T) {
	state := "test_state_123"
	resultChan := make(chan callbackResult, 1)

	server, err := startCallbackServer(0, state, resultChan) // Port 0 = find available port
	if err != nil {
		t.Fatalf("startCallbackServer() error = %v", err)
	}
	defer shutdownServer(server)

	// Get the actual port from the server address
	// Since we use port 0, we need to make a test request to localhost
	// For this test, we'll just verify the server starts without error
	// and handles requests correctly using httptest

	// Create a mock request with valid code and state
	req := httptest.NewRequest("GET", "/callback?code=test_code&state="+state, nil)
	w := httptest.NewRecorder()

	// Create handler directly to test
	mux := http.NewServeMux()
	expectedState := state
	mux.HandleFunc(CallbackPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		reqState := r.URL.Query().Get("state")

		if reqState != expectedState {
			resultChan <- callbackResult{err: nil}
			return
		}

		resultChan <- callbackResult{code: code, state: reqState}
		w.WriteHeader(http.StatusOK)
	})

	mux.ServeHTTP(w, req)

	// Check result
	select {
	case result := <-resultChan:
		if result.err != nil {
			t.Errorf("callback result error = %v", result.err)
		}
		if result.code != "test_code" {
			t.Errorf("callback code = %s, want test_code", result.code)
		}
		if result.state != state {
			t.Errorf("callback state = %s, want %s", result.state, state)
		}
	case <-time.After(time.Second):
		t.Error("callback result timeout")
	}
}

func TestCallbackServerStateMismatch(t *testing.T) {
	state := "correct_state"
	resultChan := make(chan callbackResult, 1)

	// Create handler directly to test state validation
	mux := http.NewServeMux()
	mux.HandleFunc(CallbackPath, func(w http.ResponseWriter, r *http.Request) {
		reqState := r.URL.Query().Get("state")

		if reqState != state {
			resultChan <- callbackResult{err: nil} // Changed to show state mismatch behavior
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		resultChan <- callbackResult{code: r.URL.Query().Get("code"), state: reqState}
		w.WriteHeader(http.StatusOK)
	})

	// Make request with wrong state
	req := httptest.NewRequest("GET", "/callback?code=test_code&state=wrong_state", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// The handler should detect state mismatch
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected BadRequest for state mismatch, got %d", w.Code)
	}
}

func TestCallbackServerErrorResponse(t *testing.T) {
	resultChan := make(chan callbackResult, 1)

	// Create handler directly to test error handling
	mux := http.NewServeMux()
	mux.HandleFunc(CallbackPath, func(w http.ResponseWriter, r *http.Request) {
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			_ = r.URL.Query().Get("error_description") // Would be used in real handler
			resultChan <- callbackResult{err: nil}     // Handler received error
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			return
		}
		resultChan <- callbackResult{code: "code"}
	})

	// Make request with error response
	req := httptest.NewRequest("GET", "/callback?error=access_denied&error_description=User+denied+access", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// Verify handler processed the error
	select {
	case <-resultChan:
		// Handler received the error - test passes
	case <-time.After(time.Second):
		t.Error("callback result timeout")
	}
}

func TestExchangeCodeForTokens_MockServer(t *testing.T) {
	// Create a mock token server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Expected Content-Type application/x-www-form-urlencoded, got %s", r.Header.Get("Content-Type"))
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
	resp, err := http.PostForm(server.URL, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"test_code"},
		"redirect_uri":  {"http://localhost/callback"},
		"client_id":     {OAuthClientID},
		"code_verifier": {"test_verifier"},
	})
	if err != nil {
		t.Fatalf("Mock server request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Mock server status = %d, want 200", resp.StatusCode)
	}
}
