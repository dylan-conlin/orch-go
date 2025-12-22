// Package account provides multi-account management for Claude Max usage tracking.
// This file contains OAuth authorization code flow implementation with PKCE.

package account

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// OAuth authorization constants
const (
	AuthorizationEndpoint = "https://console.anthropic.com/oauth/authorize"
	// DefaultCallbackPort is the default port for the local OAuth callback server
	DefaultCallbackPort = 19283
	// CallbackPath is the path for the OAuth callback
	CallbackPath = "/callback"
)

// OAuthConfig holds configuration for the OAuth authorization flow.
type OAuthConfig struct {
	// CallbackPort is the port for the local callback server (default: 19283)
	CallbackPort int
	// Timeout is the maximum time to wait for the authorization callback
	Timeout time.Duration
}

// DefaultOAuthConfig returns the default OAuth configuration.
func DefaultOAuthConfig() *OAuthConfig {
	return &OAuthConfig{
		CallbackPort: DefaultCallbackPort,
		Timeout:      5 * time.Minute,
	}
}

// AuthorizationResult holds the result of a successful OAuth authorization.
type AuthorizationResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64  // Unix timestamp in milliseconds
	Email        string // Email from profile (if fetched)
}

// OAuthError is returned when OAuth authorization fails.
type OAuthError struct {
	Message string
	Cause   error
}

func (e *OAuthError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *OAuthError) Unwrap() error {
	return e.Cause
}

// generateCodeVerifier generates a random code verifier for PKCE.
// Returns a 43-128 character URL-safe string.
func generateCodeVerifier() (string, error) {
	// Generate 32 random bytes (will result in 43 base64url characters)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// generateCodeChallenge generates the code challenge from a code verifier.
// Uses S256 method (SHA-256 hash, base64url encoded).
func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// buildAuthorizationURL builds the OAuth authorization URL.
func buildAuthorizationURL(redirectURI, codeChallenge, state string) string {
	params := url.Values{
		"client_id":             {OAuthClientID},
		"redirect_uri":          {redirectURI},
		"response_type":         {"code"},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
		"scope":                 {"user:inference"},
	}
	return AuthorizationEndpoint + "?" + params.Encode()
}

// exchangeCodeForTokens exchanges an authorization code for tokens.
func exchangeCodeForTokens(code, codeVerifier, redirectURI string) (*AuthorizationResult, error) {
	reqBody := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {OAuthClientID},
		"code_verifier": {codeVerifier},
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("POST", TokenEndpoint, strings.NewReader(reqBody.Encode()))
	if err != nil {
		return nil, &OAuthError{Message: "failed to create token request", Cause: err}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil, &OAuthError{Message: "token request failed", Cause: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return nil, &OAuthError{
				Message: fmt.Sprintf("token exchange failed: %s - %s", errResp.Error, errResp.ErrorDescription),
			}
		}
		return nil, &OAuthError{
			Message: fmt.Sprintf("token exchange failed with status %d", resp.StatusCode),
		}
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, &OAuthError{Message: "failed to decode token response", Cause: err}
	}

	if tokenResp.AccessToken == "" || tokenResp.RefreshToken == "" {
		return nil, &OAuthError{Message: "invalid token response: missing tokens"}
	}

	// Calculate expiration timestamp (default 8 hours if not provided)
	expiresIn := tokenResp.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 28800 // 8 hours
	}
	expiresAt := (time.Now().Unix() + int64(expiresIn)) * 1000

	return &AuthorizationResult{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// openBrowser opens the default browser with the given URL.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}

// callbackResult holds the result from the OAuth callback.
type callbackResult struct {
	code  string
	state string
	err   error
}

// StartOAuthFlow initiates the OAuth authorization code flow with PKCE.
// It opens the browser for user authentication and waits for the callback.
// Returns an AuthorizationResult with tokens on success.
func StartOAuthFlow(cfg *OAuthConfig) (*AuthorizationResult, error) {
	if cfg == nil {
		cfg = DefaultOAuthConfig()
	}

	// Generate PKCE parameters
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return nil, &OAuthError{Message: "failed to generate code verifier", Cause: err}
	}
	codeChallenge := generateCodeChallenge(codeVerifier)

	// Generate state for CSRF protection
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, &OAuthError{Message: "failed to generate state", Cause: err}
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	// Build redirect URI
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d%s", cfg.CallbackPort, CallbackPath)

	// Start local callback server
	resultChan := make(chan callbackResult, 1)
	server, err := startCallbackServer(cfg.CallbackPort, state, resultChan)
	if err != nil {
		return nil, &OAuthError{Message: "failed to start callback server", Cause: err}
	}

	// Build and open authorization URL
	authURL := buildAuthorizationURL(redirectURI, codeChallenge, state)

	fmt.Println("Opening browser for authentication...")
	fmt.Printf("If browser doesn't open, visit:\n%s\n\n", authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Warning: could not open browser: %v\n", err)
		fmt.Println("Please open the URL above manually.")
	}

	fmt.Println("Waiting for authorization...")

	// Wait for callback with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	var result callbackResult
	select {
	case result = <-resultChan:
	case <-ctx.Done():
		shutdownServer(server)
		return nil, &OAuthError{Message: "authorization timed out"}
	}

	// Shutdown the server
	shutdownServer(server)

	if result.err != nil {
		return nil, &OAuthError{Message: "authorization failed", Cause: result.err}
	}

	// Exchange code for tokens
	tokens, err := exchangeCodeForTokens(result.code, codeVerifier, redirectURI)
	if err != nil {
		return nil, err
	}

	// Fetch email from profile
	tokens.Email = fetchProfileEmailFromToken(tokens.AccessToken)

	return tokens, nil
}

// startCallbackServer starts a local HTTP server to receive the OAuth callback.
func startCallbackServer(port int, expectedState string, resultChan chan<- callbackResult) (*http.Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	mux := http.NewServeMux()
	server := &http.Server{Handler: mux}

	mux.HandleFunc(CallbackPath, func(w http.ResponseWriter, r *http.Request) {
		// Check for error response
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			errDesc := r.URL.Query().Get("error_description")
			resultChan <- callbackResult{
				err: fmt.Errorf("%s: %s", errParam, errDesc),
			}
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<html><body><h1>Authorization Failed</h1><p>%s: %s</p><p>You can close this window.</p></body></html>`, errParam, errDesc)
			return
		}

		// Get code and state
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		// Validate state
		if state != expectedState {
			resultChan <- callbackResult{
				err: fmt.Errorf("state mismatch: possible CSRF attack"),
			}
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><body><h1>Authorization Failed</h1><p>State validation failed.</p><p>You can close this window.</p></body></html>`)
			return
		}

		if code == "" {
			resultChan <- callbackResult{
				err: fmt.Errorf("no authorization code received"),
			}
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><body><h1>Authorization Failed</h1><p>No authorization code received.</p><p>You can close this window.</p></body></html>`)
			return
		}

		// Success
		resultChan <- callbackResult{
			code:  code,
			state: state,
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h1>Authorization Successful!</h1><p>You can close this window and return to the terminal.</p></body></html>`)
	})

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			resultChan <- callbackResult{err: fmt.Errorf("server error: %w", err)}
		}
	}()

	return server, nil
}

// shutdownServer gracefully shuts down the callback server.
func shutdownServer(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

// fetchProfileEmailFromToken fetches the user's email from the profile API.
func fetchProfileEmailFromToken(accessToken string) string {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", ProfileEndpoint, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("anthropic-beta", AnthropicBetaHeaders)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var profile profileAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return ""
	}

	return profile.Account.Email
}

// AddAccount performs the OAuth flow and saves the account to accounts.yaml.
// Returns the email of the added account on success.
func AddAccount(name string, setDefault bool, cfg *OAuthConfig) (string, error) {
	// Perform OAuth flow
	result, err := StartOAuthFlow(cfg)
	if err != nil {
		return "", err
	}

	// Load existing config
	config, err := LoadConfig()
	if err != nil {
		return "", &OAuthError{Message: "failed to load accounts config", Cause: err}
	}

	// Create account entry
	acc := Account{
		Email:        result.Email,
		RefreshToken: result.RefreshToken,
		Source:       "saved",
	}

	// Save account
	config.Save(name, acc, setDefault)

	// Write config
	if err := SaveConfig(config); err != nil {
		return "", &OAuthError{Message: "failed to save accounts config", Cause: err}
	}

	// Also update OpenCode auth file so the new account is immediately active
	auth := &OpenCodeAuth{}
	auth.Anthropic.Type = "oauth"
	auth.Anthropic.Refresh = result.RefreshToken
	auth.Anthropic.Access = result.AccessToken
	auth.Anthropic.Expires = result.ExpiresAt

	if err := SaveOpenCodeAuth(auth); err != nil {
		// Non-fatal: account is saved, but OpenCode won't be updated
		fmt.Printf("Warning: failed to update OpenCode auth: %v\n", err)
	}

	email := result.Email
	if email == "" {
		email = name
	}

	return email, nil
}
