// Package account provides multi-account management for Claude Max usage tracking.
// This file contains OAuth authorization code flow implementation with PKCE.

package account

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// OAuth authorization constants
const (
	// AuthorizationEndpoint is the Claude Max OAuth endpoint.
	// IMPORTANT: For Claude Max/Pro subscription tokens with inference scope,
	// we must use claude.ai - not console.anthropic.com (which is for API keys).
	// See: opencode-anthropic-auth plugin for reference implementation.
	AuthorizationEndpoint = "https://claude.ai/oauth/authorize"
	// AnthropicCallbackURL is Anthropic's official OAuth callback URL.
	// Anthropic only allows their own callback URL - local servers are not permitted.
	AnthropicCallbackURL = "https://console.anthropic.com/oauth/code/callback"
)

// OAuthConfig holds configuration for the OAuth authorization flow.
type OAuthConfig struct {
	// Timeout is the maximum time to wait for the user to paste the code
	Timeout time.Duration
}

// DefaultOAuthConfig returns the default OAuth configuration.
func DefaultOAuthConfig() *OAuthConfig {
	return &OAuthConfig{
		Timeout: 10 * time.Minute,
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
// Following opencode-anthropic-auth plugin pattern, we use the code verifier as state.
func buildAuthorizationURL(codeChallenge, codeVerifier string) string {
	params := url.Values{
		"code":                  {"true"}, // Signal that we want a code displayed, not redirect
		"client_id":             {OAuthClientID},
		"redirect_uri":          {AnthropicCallbackURL},
		"response_type":         {"code"},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
		"state":                 {codeVerifier}, // Use verifier as state (matches opencode pattern)
		"scope":                 {"org:create_api_key user:profile user:inference"},
	}
	return AuthorizationEndpoint + "?" + params.Encode()
}

// exchangeCodeForTokens exchanges an authorization code for tokens.
// The code may contain state appended after '#' (e.g., "code#state").
func exchangeCodeForTokens(rawCode, codeVerifier string) (*AuthorizationResult, error) {
	// Split code and state if combined (Anthropic returns "code#state" format)
	code := rawCode
	state := ""
	if idx := strings.Index(rawCode, "#"); idx != -1 {
		code = rawCode[:idx]
		state = rawCode[idx+1:]
	}

	// Build request body as JSON (matching opencode-anthropic-auth pattern)
	reqBody := map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  AnthropicCallbackURL,
		"client_id":     OAuthClientID,
		"code_verifier": codeVerifier,
	}
	// Include state if present
	if state != "" {
		reqBody["state"] = state
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, &OAuthError{Message: "failed to marshal token request", Cause: err}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("POST", TokenEndpoint, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, &OAuthError{Message: "failed to create token request", Cause: err}
	}
	req.Header.Set("Content-Type", "application/json")

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

// StartOAuthFlow initiates the OAuth authorization code flow with PKCE.
// It opens the browser for user authentication and prompts the user to paste
// the authorization code displayed by Anthropic's callback page.
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

	// Build authorization URL (using code verifier as state, matching opencode pattern)
	authURL := buildAuthorizationURL(codeChallenge, codeVerifier)

	fmt.Println("Opening browser for authentication...")
	fmt.Printf("\nIf browser doesn't open, visit:\n%s\n\n", authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Warning: could not open browser: %v\n", err)
		fmt.Println("Please open the URL above manually.")
	}

	// Prompt user to paste the authorization code
	fmt.Println("After authorizing, Anthropic will display an authorization code.")
	fmt.Print("Paste the authorization code here: ")

	reader := bufio.NewReader(os.Stdin)
	rawCode, err := reader.ReadString('\n')
	if err != nil {
		return nil, &OAuthError{Message: "failed to read authorization code", Cause: err}
	}

	rawCode = strings.TrimSpace(rawCode)
	if rawCode == "" {
		return nil, &OAuthError{Message: "no authorization code provided"}
	}

	fmt.Println("\nExchanging code for tokens...")

	// Exchange code for tokens
	tokens, err := exchangeCodeForTokens(rawCode, codeVerifier)
	if err != nil {
		return nil, err
	}

	// Fetch email from profile
	tokens.Email = fetchProfileEmailFromToken(tokens.AccessToken)

	return tokens, nil
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
