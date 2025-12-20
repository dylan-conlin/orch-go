// Package account provides multi-account management for Claude Max usage tracking.
//
// Configuration file: ~/.orch/accounts.yaml
//
// Example config:
//
//	accounts:
//	  personal:
//	    email: user@example.com
//	    refresh_token: sk-ant-ort01-...
//	    source: saved
//	  work:
//	    email: user@company.com
//	    refresh_token: sk-ant-ort01-...
//	    source: saved
//	default: personal
package account

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// OAuth constants for Anthropic API
const (
	TokenEndpoint = "https://console.anthropic.com/v1/oauth/token"
	OAuthClientID = "9d1c250a-e61b-44d9-88ed-5944d1962f5e" // OpenCode's public client ID
)

// TokenInfo holds OAuth token information from refresh exchange.
type TokenInfo struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64 // Unix timestamp in milliseconds
}

// TokenRefreshError is returned when token refresh fails.
type TokenRefreshError struct {
	Message string
}

func (e *TokenRefreshError) Error() string {
	return e.Message
}

// Account represents a saved account configuration.
type Account struct {
	Email        string `yaml:"email"`
	RefreshToken string `yaml:"refresh_token"`
	Source       string `yaml:"source"` // "saved", "opencode", "keychain", "docker"
}

// Config holds the accounts configuration.
type Config struct {
	Accounts map[string]Account `yaml:"accounts"`
	Default  string             `yaml:"default"`
}

// ErrNotFound is returned when an account is not found.
var ErrNotFound = fmt.Errorf("account not found")

// ErrNoAccounts is returned when there are no configured accounts.
var ErrNoAccounts = fmt.Errorf("no accounts configured")

// ConfigPath returns the path to accounts.yaml.
func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orch", "accounts.yaml")
}

// OpenCodeAuthPath returns the path to OpenCode's auth.json.
func OpenCodeAuthPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "opencode", "auth.json")
}

// LoadConfig loads accounts from ~/.orch/accounts.yaml.
func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Accounts: make(map[string]Account)}, nil
		}
		return nil, fmt.Errorf("failed to read accounts.yaml: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse accounts.yaml: %w", err)
	}

	if cfg.Accounts == nil {
		cfg.Accounts = make(map[string]Account)
	}

	return &cfg, nil
}

// SaveConfig saves accounts to ~/.orch/accounts.yaml.
func SaveConfig(cfg *Config) error {
	// Ensure directory exists
	dir := filepath.Dir(ConfigPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal accounts.yaml: %w", err)
	}

	// Write with restrictive permissions (contains tokens)
	if err := os.WriteFile(ConfigPath(), data, 0600); err != nil {
		return fmt.Errorf("failed to write accounts.yaml: %w", err)
	}

	return nil
}

// Get retrieves an account by name.
func (c *Config) Get(name string) (*Account, error) {
	if acc, ok := c.Accounts[name]; ok {
		return &acc, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
}

// List returns all account names.
func (c *Config) List() []string {
	names := make([]string, 0, len(c.Accounts))
	for name := range c.Accounts {
		names = append(names, name)
	}
	return names
}

// Remove deletes an account by name.
func (c *Config) Remove(name string) error {
	if _, ok := c.Accounts[name]; !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, name)
	}

	delete(c.Accounts, name)

	// Clear default if it was this account
	if c.Default == name {
		c.Default = ""
		// Set a new default if there are other saved accounts
		for n, acc := range c.Accounts {
			if acc.Source == "saved" {
				c.Default = n
				break
			}
		}
	}

	return nil
}

// Save adds or updates an account.
func (c *Config) Save(name string, acc Account, setDefault bool) {
	c.Accounts[name] = acc

	if setDefault || c.Default == "" {
		c.Default = name
	}
}

// OpenCodeAuth represents the OpenCode auth.json structure.
type OpenCodeAuth struct {
	Anthropic struct {
		Type    string `json:"type"`
		Refresh string `json:"refresh"`
		Access  string `json:"access"`
		Expires int64  `json:"expires"`
	} `json:"anthropic"`
}

// LoadOpenCodeAuth loads auth from OpenCode's auth.json.
func LoadOpenCodeAuth() (*OpenCodeAuth, error) {
	data, err := os.ReadFile(OpenCodeAuthPath())
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenCode auth file: %w", err)
	}

	var auth OpenCodeAuth
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, fmt.Errorf("failed to parse OpenCode auth file: %w", err)
	}

	return &auth, nil
}

// SaveOpenCodeAuth saves auth to OpenCode's auth.json.
func SaveOpenCodeAuth(auth *OpenCodeAuth) error {
	// Ensure directory exists
	dir := filepath.Dir(OpenCodeAuthPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create auth directory: %w", err)
	}

	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth.json: %w", err)
	}

	// Write with restrictive permissions
	if err := os.WriteFile(OpenCodeAuthPath(), data, 0600); err != nil {
		return fmt.Errorf("failed to write auth.json: %w", err)
	}

	return nil
}

// AccountInfo represents info about a saved account for display.
type AccountInfo struct {
	Name      string
	Email     string
	IsDefault bool
	IsActive  bool
}

// ListAccountInfo returns account info for all saved accounts.
func ListAccountInfo() ([]AccountInfo, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	// Try to get current active account email
	currentEmail := ""
	if auth, err := LoadOpenCodeAuth(); err == nil && auth.Anthropic.Access != "" {
		// We have an active token - could fetch profile email here
		// For now, just note that we have one
		_ = auth
	}

	var result []AccountInfo
	for name, acc := range cfg.Accounts {
		if acc.Source != "saved" {
			continue
		}

		result = append(result, AccountInfo{
			Name:      name,
			Email:     acc.Email,
			IsDefault: cfg.Default == name,
			IsActive:  currentEmail != "" && acc.Email == currentEmail,
		})
	}

	return result, nil
}

// tokenRequest represents the OAuth token refresh request body.
type tokenRequest struct {
	GrantType    string `json:"grant_type"`
	RefreshToken string `json:"refresh_token"`
	ClientID     string `json:"client_id"`
}

// tokenResponse represents the OAuth token refresh response.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// RefreshOAuthToken exchanges a refresh token for new access and refresh tokens.
func RefreshOAuthToken(refreshToken string) (*TokenInfo, error) {
	reqBody := tokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		ClientID:     OAuthClientID,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, &TokenRefreshError{Message: fmt.Sprintf("failed to marshal request: %v", err)}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("POST", TokenEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, &TokenRefreshError{Message: fmt.Sprintf("failed to create request: %v", err)}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, &TokenRefreshError{Message: fmt.Sprintf("token refresh request failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var respBody bytes.Buffer
		respBody.ReadFrom(resp.Body)
		errMsg := respBody.String()
		if len(errMsg) > 200 {
			errMsg = errMsg[:200]
		}
		return nil, &TokenRefreshError{
			Message: fmt.Sprintf("token refresh failed with status %d: %s", resp.StatusCode, errMsg),
		}
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, &TokenRefreshError{Message: fmt.Sprintf("failed to decode response: %v", err)}
	}

	if tokenResp.AccessToken == "" || tokenResp.RefreshToken == "" {
		return nil, &TokenRefreshError{Message: "invalid token response: missing tokens"}
	}

	// expires_in is in seconds, convert to milliseconds timestamp
	expiresIn := tokenResp.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 28800 // Default 8 hours
	}
	expiresAt := (time.Now().Unix() + int64(expiresIn)) * 1000

	return &TokenInfo{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// SwitchAccount switches to a saved account by refreshing its OAuth tokens.
// It updates the OpenCode auth file and saves the new refresh token back to accounts.yaml.
func SwitchAccount(name string) (email string, err error) {
	cfg, err := LoadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load accounts: %w", err)
	}

	acc, ok := cfg.Accounts[name]
	if !ok {
		available := cfg.List()
		if len(available) == 0 {
			return "", fmt.Errorf("account '%s' not found, no accounts configured", name)
		}
		return "", fmt.Errorf("account '%s' not found, available: %v", name, available)
	}

	if acc.Source != "saved" {
		return "", fmt.Errorf("account '%s' is not a saved account (source: %s), only saved accounts can be switched to", name, acc.Source)
	}

	if acc.RefreshToken == "" {
		return "", fmt.Errorf("account '%s' has no refresh token", name)
	}

	// Exchange refresh token for new tokens
	tokenInfo, err := RefreshOAuthToken(acc.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("failed to refresh token for '%s': %w", name, err)
	}

	// Update the saved account with new refresh token
	acc.RefreshToken = tokenInfo.RefreshToken
	cfg.Accounts[name] = acc
	if err := SaveConfig(cfg); err != nil {
		return "", fmt.Errorf("failed to save updated config: %w", err)
	}

	// Write to OpenCode auth file
	auth := &OpenCodeAuth{}
	auth.Anthropic.Type = "oauth"
	auth.Anthropic.Refresh = tokenInfo.RefreshToken
	auth.Anthropic.Access = tokenInfo.AccessToken
	auth.Anthropic.Expires = tokenInfo.ExpiresAt

	if err := SaveOpenCodeAuth(auth); err != nil {
		return "", fmt.Errorf("failed to update OpenCode auth: %w", err)
	}

	if acc.Email != "" {
		return acc.Email, nil
	}
	return name, nil
}
