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
	"sort"
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
	Message     string
	AccountName string // The account name, if known (for actionable guidance)
}

func (e *TokenRefreshError) Error() string {
	return e.Message
}

// WithAccount returns a new TokenRefreshError with the account name set.
func (e *TokenRefreshError) WithAccount(name string) *TokenRefreshError {
	return &TokenRefreshError{
		Message:     e.Message,
		AccountName: name,
	}
}

// ActionableGuidance returns a string with actionable guidance for resolving the error.
func (e *TokenRefreshError) ActionableGuidance() string {
	if e.AccountName != "" {
		return fmt.Sprintf("To re-authorize: orch account remove %s && orch account add %s", e.AccountName, e.AccountName)
	}
	return "To re-authorize: orch account remove <name> && orch account add <name>"
}

// Account represents a saved account configuration.
type Account struct {
	Email        string `yaml:"email"`
	RefreshToken string `yaml:"refresh_token"`
	Source       string `yaml:"source"`     // "saved", "opencode", "keychain", "docker"
	Tier         string `yaml:"tier,omitempty"`       // Subscription tier: "5x", "20x"
	Role         string `yaml:"role,omitempty"`       // Routing role: "primary", "spillover"
	ConfigDir    string `yaml:"config_dir,omitempty"` // Claude CLI config directory (e.g., "~/.claude-personal")
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

// GetConfigDir returns the config_dir for a named account.
// Returns empty string if account not found or has no config_dir set.
func GetConfigDir(name string) string {
	cfg, err := LoadConfig()
	if err != nil || name == "" {
		return ""
	}
	acc, ok := cfg.Accounts[name]
	if !ok {
		return ""
	}
	return acc.ConfigDir
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
// It merges the anthropic section into the existing file, preserving any
// other provider credentials (e.g., openai OAuth) that may be present.
func SaveOpenCodeAuth(auth *OpenCodeAuth) error {
	// Ensure directory exists
	dir := filepath.Dir(OpenCodeAuthPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create auth directory: %w", err)
	}

	// Read existing file to preserve non-anthropic fields (e.g., openai OAuth)
	existing := make(map[string]interface{})
	if data, err := os.ReadFile(OpenCodeAuthPath()); err == nil {
		// Best-effort parse — if it fails, we start fresh
		_ = json.Unmarshal(data, &existing)
	}

	// Update only the anthropic section
	existing["anthropic"] = auth.Anthropic

	data, err := json.MarshalIndent(existing, "", "  ")
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
	Tier      string // Subscription tier: "5x", "20x"
	Role      string // Routing role: "primary", "spillover"
	ConfigDir string // Claude CLI config directory
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
			Tier:      acc.Tier,
			Role:      acc.Role,
			ConfigDir: acc.ConfigDir,
		})
	}

	return result, nil
}

// RecommendAccount returns the name of the recommended account for spawning.
// Uses the same lowest-weekly-usage algorithm as resolveAccount:
//  1. Collect capacity for all accounts (regardless of role)
//  2. Pick the account with highest SevenDayRemaining (most weekly headroom)
//  3. Tie-break: FiveHourRemaining, then alphabetical name
//  4. Without capacity data, recommend first primary account
//
// Returns empty string if no accounts are configured.
func RecommendAccount(accounts []AccountInfo, capacityFetcher func(string) *CapacityInfo) string {
	if len(accounts) == 0 {
		return ""
	}

	// Collect all account names sorted for deterministic behavior
	var allNames []string
	for _, acc := range accounts {
		allNames = append(allNames, acc.Name)
	}
	sort.Strings(allNames)

	if capacityFetcher == nil {
		// Without capacity data, recommend first primary account (sorted for determinism)
		var primaries []string
		for _, acc := range accounts {
			if acc.Role == "primary" || acc.Role == "" {
				primaries = append(primaries, acc.Name)
			}
		}
		sort.Strings(primaries)
		if len(primaries) > 0 {
			return primaries[0]
		}
		return allNames[0]
	}

	// Lowest-weekly-usage: pick account with most remaining weekly capacity
	type candidate struct {
		name     string
		capacity *CapacityInfo
	}
	var candidates []candidate
	for _, name := range allNames {
		cap := capacityFetcher(name)
		if cap != nil {
			candidates = append(candidates, candidate{name: name, capacity: cap})
		}
	}

	if len(candidates) == 0 {
		return allNames[0]
	}

	// Sort: highest SevenDayRemaining first, then FiveHourRemaining, then name
	sort.Slice(candidates, func(i, j int) bool {
		ci, cj := candidates[i], candidates[j]
		if ci.capacity.SevenDayRemaining != cj.capacity.SevenDayRemaining {
			return ci.capacity.SevenDayRemaining > cj.capacity.SevenDayRemaining
		}
		if ci.capacity.FiveHourRemaining != cj.capacity.FiveHourRemaining {
			return ci.capacity.FiveHourRemaining > cj.capacity.FiveHourRemaining
		}
		return ci.name < cj.name
	})

	return candidates[0].name
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
		// If it's a TokenRefreshError, attach the account name for actionable guidance
		if tokenErr, ok := err.(*TokenRefreshError); ok {
			return "", tokenErr.WithAccount(name)
		}
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

// ============================================================================
// Capacity Tracking
// ============================================================================

// API configuration for capacity tracking
const (
	UsageEndpoint   = "https://api.anthropic.com/api/oauth/usage"
	ProfileEndpoint = "https://api.anthropic.com/api/oauth/profile"
	UserAgent       = "claude-code/2.0.32"
)

// AnthropicBetaHeaders are required for OAuth tokens to work with Claude Code credentials.
var AnthropicBetaHeaders = "oauth-2025-04-20,claude-code-20250219,interleaved-thinking-2025-05-14,fine-grained-tool-streaming-2025-05-14"

// CapacityInfo represents usage capacity for an account.
type CapacityInfo struct {
	// FiveHourUsed is the 5-hour session utilization (0-100).
	FiveHourUsed float64
	// FiveHourRemaining is the remaining 5-hour capacity (0-100).
	FiveHourRemaining float64
	// FiveHourResets is when the 5-hour limit resets.
	FiveHourResets *time.Time

	// SevenDayUsed is the weekly utilization (0-100).
	SevenDayUsed float64
	// SevenDayRemaining is the remaining weekly capacity (0-100).
	SevenDayRemaining float64
	// SevenDayResets is when the weekly limit resets.
	SevenDayResets *time.Time

	// Email is the account email (if available).
	Email string
	// Error is set if capacity fetch failed.
	Error string
}

// IsHealthy returns true if the account has >20% remaining capacity on both limits.
func (c *CapacityInfo) IsHealthy() bool {
	if c.Error != "" {
		return false
	}
	return c.FiveHourRemaining > 20 && c.SevenDayRemaining > 20
}

// IsLow returns true if either limit is below 20% remaining.
func (c *CapacityInfo) IsLow() bool {
	if c.Error != "" {
		return true
	}
	return c.FiveHourRemaining < 20 || c.SevenDayRemaining < 20
}

// IsCritical returns true if either limit is below 5% remaining.
func (c *CapacityInfo) IsCritical() bool {
	if c.Error != "" {
		return true
	}
	return c.FiveHourRemaining < 5 || c.SevenDayRemaining < 5
}

// usageAPIResponse represents the raw API response structure.
type usageAPIResponse struct {
	FiveHour          *limitResponse `json:"five_hour"`
	SevenDay          *limitResponse `json:"seven_day"`
	SevenDayOpus      *limitResponse `json:"seven_day_opus"`
	SevenDayOAuthApps *limitResponse `json:"seven_day_oauth_apps"`
}

type limitResponse struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    string  `json:"resets_at"`
}

// profileAPIResponse represents the profile API response structure.
type profileAPIResponse struct {
	Account struct {
		Email string `json:"email"`
	} `json:"account"`
}

// CapacityError is returned when capacity fetch fails.
type CapacityError struct {
	Message string
}

func (e *CapacityError) Error() string {
	return e.Message
}

// GetCurrentCapacity fetches capacity info for the currently active account.
// It reads the OAuth token from OpenCode's auth.json and queries the Anthropic API.
func GetCurrentCapacity() (*CapacityInfo, error) {
	// Load auth to get access token
	auth, err := LoadOpenCodeAuth()
	if err != nil {
		return &CapacityInfo{Error: fmt.Sprintf("failed to load auth: %v", err)}, err
	}

	if auth.Anthropic.Access == "" {
		return &CapacityInfo{Error: "no access token found"}, &CapacityError{Message: "no access token found in OpenCode auth file"}
	}

	// Check if token is expired
	if auth.Anthropic.Expires > 0 {
		expiresAt := time.Unix(auth.Anthropic.Expires/1000, 0)
		if time.Now().After(expiresAt) {
			return &CapacityInfo{Error: "access token expired"}, &CapacityError{Message: "OAuth token has expired - restart OpenCode to refresh"}
		}
	}

	return fetchCapacityWithToken(auth.Anthropic.Access)
}

// GetAccountCapacity fetches capacity info for a specific saved account.
// This temporarily refreshes the account's token to check capacity without switching.
// Note: This does NOT switch the active account - it only peeks at capacity.
func GetAccountCapacity(name string) (*CapacityInfo, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return &CapacityInfo{Error: fmt.Sprintf("failed to load config: %v", err)}, err
	}

	acc, ok := cfg.Accounts[name]
	if !ok {
		return &CapacityInfo{Error: fmt.Sprintf("account not found: %s", name)}, fmt.Errorf("%w: %s", ErrNotFound, name)
	}

	if acc.RefreshToken == "" {
		return &CapacityInfo{Error: fmt.Sprintf("account has no refresh token: %s", name)}, &CapacityError{Message: fmt.Sprintf("account '%s' has no refresh token", name)}
	}

	// Check if this is the currently active account in OpenCode
	// We need to update OpenCode auth.json if this account is active, otherwise
	// the token rotation will invalidate active agent sessions
	currentAuth, authErr := LoadOpenCodeAuth()
	isActiveAccount := authErr == nil && currentAuth.Anthropic.Refresh == acc.RefreshToken

	// Refresh the token to get a temporary access token
	tokenInfo, err := RefreshOAuthToken(acc.RefreshToken)
	if err != nil {
		// If it's a TokenRefreshError, attach the account name for actionable guidance
		if tokenErr, ok := err.(*TokenRefreshError); ok {
			return &CapacityInfo{Error: fmt.Sprintf("token refresh failed: %v", err)}, tokenErr.WithAccount(name)
		}
		return &CapacityInfo{Error: fmt.Sprintf("token refresh failed: %v", err)}, err
	}

	// Save the updated refresh token back to config
	acc.RefreshToken = tokenInfo.RefreshToken
	cfg.Accounts[name] = acc
	if err := SaveConfig(cfg); err != nil {
		// Log warning but don't fail - we still have the access token
		fmt.Fprintf(os.Stderr, "Warning: failed to save updated refresh token: %v\n", err)
	}

	// If this is the active account, also update OpenCode auth.json
	// This prevents active agents from losing their sessions due to token rotation
	if isActiveAccount {
		currentAuth.Anthropic.Refresh = tokenInfo.RefreshToken
		currentAuth.Anthropic.Access = tokenInfo.AccessToken
		currentAuth.Anthropic.Expires = tokenInfo.ExpiresAt
		if err := SaveOpenCodeAuth(currentAuth); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update OpenCode auth: %v\n", err)
		}
	}

	// Fetch capacity with the temporary access token
	capacity, err := fetchCapacityWithToken(tokenInfo.AccessToken)
	if capacity != nil && acc.Email != "" {
		capacity.Email = acc.Email
	}
	return capacity, err
}

// fetchCapacityWithToken fetches capacity info using a specific access token.
func fetchCapacityWithToken(accessToken string) (*CapacityInfo, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	// Fetch email from profile (optional, non-blocking failure)
	email := fetchProfileEmail(accessToken, client)

	// Fetch usage data
	req, err := http.NewRequest("GET", UsageEndpoint, nil)
	if err != nil {
		return &CapacityInfo{Error: fmt.Sprintf("request creation failed: %v", err)}, &CapacityError{Message: fmt.Sprintf("failed to create request: %v", err)}
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("anthropic-beta", AnthropicBetaHeaders)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return &CapacityInfo{Error: fmt.Sprintf("request failed: %v", err)}, &CapacityError{Message: fmt.Sprintf("request failed: %v", err)}
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return &CapacityInfo{Error: "authentication failed (401)"}, &CapacityError{Message: "authentication failed - token may be expired"}
	case http.StatusForbidden:
		return &CapacityInfo{Error: "access forbidden (403)"}, &CapacityError{Message: "access forbidden - may require Max subscription"}
	}

	if resp.StatusCode != http.StatusOK {
		return &CapacityInfo{Error: fmt.Sprintf("API status %d", resp.StatusCode)}, &CapacityError{Message: fmt.Sprintf("API returned status %d", resp.StatusCode)}
	}

	var apiResp usageAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return &CapacityInfo{Error: fmt.Sprintf("parse failed: %v", err)}, &CapacityError{Message: fmt.Sprintf("failed to parse response: %v", err)}
	}

	capacity := &CapacityInfo{Email: email}

	// Parse 5-hour limit
	if apiResp.FiveHour != nil {
		capacity.FiveHourUsed = apiResp.FiveHour.Utilization
		capacity.FiveHourRemaining = 100.0 - apiResp.FiveHour.Utilization
		if apiResp.FiveHour.ResetsAt != "" {
			if t, err := time.Parse(time.RFC3339, apiResp.FiveHour.ResetsAt); err == nil {
				capacity.FiveHourResets = &t
			}
		}
	}

	// Parse 7-day limit
	if apiResp.SevenDay != nil {
		capacity.SevenDayUsed = apiResp.SevenDay.Utilization
		capacity.SevenDayRemaining = 100.0 - apiResp.SevenDay.Utilization
		if apiResp.SevenDay.ResetsAt != "" {
			if t, err := time.Parse(time.RFC3339, apiResp.SevenDay.ResetsAt); err == nil {
				capacity.SevenDayResets = &t
			}
		}
	}

	return capacity, nil
}

// fetchProfileEmail fetches the account email from the profile API.
func fetchProfileEmail(token string, client *http.Client) string {
	req, err := http.NewRequest("GET", ProfileEndpoint, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("Authorization", "Bearer "+token)
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

// ListAccountsWithCapacity returns all saved accounts with their current capacity.
// This makes API calls to check capacity for each account.
func ListAccountsWithCapacity() ([]AccountWithCapacity, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	var result []AccountWithCapacity
	for name, acc := range cfg.Accounts {
		if acc.Source != "saved" {
			continue
		}

		awc := AccountWithCapacity{
			Name:      name,
			Email:     acc.Email,
			IsDefault: cfg.Default == name,
		}

		// Fetch capacity for this account
		capacity, _ := GetAccountCapacity(name)
		if capacity != nil {
			awc.Capacity = capacity
		}

		result = append(result, awc)
	}

	return result, nil
}

// AccountWithCapacity combines account info with capacity data.
type AccountWithCapacity struct {
	Name      string
	Email     string
	IsDefault bool
	Capacity  *CapacityInfo
}

// FindBestAccount returns the saved account with the most remaining capacity.
// Returns empty string if no accounts have healthy capacity.
func FindBestAccount() (string, *CapacityInfo, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return "", nil, err
	}

	var bestName string
	var bestCapacity *CapacityInfo

	for name, acc := range cfg.Accounts {
		if acc.Source != "saved" {
			continue
		}

		capacity, err := GetAccountCapacity(name)
		if err != nil {
			continue
		}

		if capacity.Error != "" {
			continue
		}

		// Use 7-day remaining as primary metric
		if bestCapacity == nil || capacity.SevenDayRemaining > bestCapacity.SevenDayRemaining {
			bestName = name
			bestCapacity = capacity
		}
	}

	if bestName == "" {
		return "", nil, &CapacityError{Message: "no healthy accounts found"}
	}

	return bestName, bestCapacity, nil
}

// ============================================================================
// Auto Account Switching
// ============================================================================

// AutoSwitchThresholds configures when to auto-switch accounts.
type AutoSwitchThresholds struct {
	// FiveHourThreshold is the 5-hour usage % above which to consider switching (default 80).
	FiveHourThreshold float64
	// WeeklyThreshold is the weekly usage % above which to consider switching (default 90).
	WeeklyThreshold float64
	// MinHeadroomDelta is the minimum additional headroom an alternate account must have
	// over the current account to justify switching (default 10%).
	MinHeadroomDelta float64
}

// DefaultAutoSwitchThresholds returns sensible defaults.
func DefaultAutoSwitchThresholds() AutoSwitchThresholds {
	return AutoSwitchThresholds{
		FiveHourThreshold: 80,
		WeeklyThreshold:   90,
		MinHeadroomDelta:  10,
	}
}

// AutoSwitchResult describes the outcome of an auto-switch check.
type AutoSwitchResult struct {
	// Switched is true if an account switch occurred.
	Switched bool
	// FromAccount is the previous account (if switched).
	FromAccount string
	// ToAccount is the new account (if switched).
	ToAccount string
	// FromCapacity is the capacity of the previous account.
	FromCapacity *CapacityInfo
	// ToCapacity is the capacity of the new account.
	ToCapacity *CapacityInfo
	// Reason explains why a switch did or didn't happen.
	Reason string
}

// ShouldAutoSwitch checks if the current account usage exceeds thresholds
// and if an alternate account has more headroom. Does NOT perform the switch.
func ShouldAutoSwitch(thresholds AutoSwitchThresholds) (*AutoSwitchResult, error) {
	result := &AutoSwitchResult{}

	// Get current account capacity
	currentCapacity, err := GetCurrentCapacity()
	if err != nil {
		return nil, fmt.Errorf("failed to get current capacity: %w", err)
	}

	if currentCapacity.Error != "" {
		return nil, &CapacityError{Message: currentCapacity.Error}
	}

	// Check if current account is over thresholds
	fiveHourUsed := currentCapacity.FiveHourUsed
	weeklyUsed := currentCapacity.SevenDayUsed

	result.FromCapacity = currentCapacity

	overFiveHour := fiveHourUsed > thresholds.FiveHourThreshold
	overWeekly := weeklyUsed > thresholds.WeeklyThreshold

	if !overFiveHour && !overWeekly {
		result.Switched = false
		result.Reason = fmt.Sprintf("current account healthy (5h: %.1f%%, weekly: %.1f%%)", fiveHourUsed, weeklyUsed)
		return result, nil
	}

	// Current account is over threshold - check alternates
	cfg, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load accounts: %w", err)
	}

	// Find the current account name by matching email
	var currentName string
	for name, acc := range cfg.Accounts {
		if acc.Email == currentCapacity.Email && acc.Source == "saved" {
			currentName = name
			result.FromAccount = name
			break
		}
	}

	// If we couldn't identify the current account, log and continue
	if currentName == "" {
		// Try to identify by refresh token from OpenCode auth
		auth, authErr := LoadOpenCodeAuth()
		if authErr == nil {
			for name, acc := range cfg.Accounts {
				if acc.RefreshToken == auth.Anthropic.Refresh && acc.Source == "saved" {
					currentName = name
					result.FromAccount = name
					break
				}
			}
		}
	}

	// Find best alternate account
	var bestName string
	var bestCapacity *CapacityInfo
	var bestHeadroom float64 = -1

	// Calculate current headroom (use the tighter constraint)
	currentFiveHourHeadroom := 100.0 - currentCapacity.FiveHourUsed
	currentWeeklyHeadroom := 100.0 - currentCapacity.SevenDayUsed
	currentHeadroom := min(currentFiveHourHeadroom, currentWeeklyHeadroom)

	for name, acc := range cfg.Accounts {
		if acc.Source != "saved" {
			continue
		}

		// Skip the current account
		if name == currentName {
			continue
		}

		capacity, err := GetAccountCapacity(name)
		if err != nil {
			continue
		}

		if capacity.Error != "" {
			continue
		}

		// Calculate headroom for this account
		fiveHourHeadroom := 100.0 - capacity.FiveHourUsed
		weeklyHeadroom := 100.0 - capacity.SevenDayUsed
		headroom := min(fiveHourHeadroom, weeklyHeadroom)

		// Must have more headroom than current + delta
		if headroom > bestHeadroom && headroom > currentHeadroom+thresholds.MinHeadroomDelta {
			bestName = name
			bestCapacity = capacity
			bestHeadroom = headroom
		}
	}

	if bestName == "" {
		result.Switched = false
		result.Reason = fmt.Sprintf("no alternate account has enough headroom (current: %.1f%%, need: %.1f%% more)",
			currentHeadroom, thresholds.MinHeadroomDelta)
		return result, nil
	}

	result.Switched = true
	result.ToAccount = bestName
	result.ToCapacity = bestCapacity
	result.Reason = fmt.Sprintf("switching from %s (%.1f%% headroom) to %s (%.1f%% headroom)",
		currentName, currentHeadroom, bestName, bestHeadroom)

	return result, nil
}

// AutoSwitchIfNeeded checks usage and switches to a better account if needed.
// Returns the result of the check/switch operation.
func AutoSwitchIfNeeded(thresholds AutoSwitchThresholds) (*AutoSwitchResult, error) {
	result, err := ShouldAutoSwitch(thresholds)
	if err != nil {
		return nil, err
	}

	if !result.Switched {
		return result, nil
	}

	// Perform the actual switch
	email, err := SwitchAccount(result.ToAccount)
	if err != nil {
		result.Switched = false
		result.Reason = fmt.Sprintf("switch failed: %v", err)
		return result, fmt.Errorf("auto-switch to %s failed: %w", result.ToAccount, err)
	}

	// Update reason with successful switch info
	result.Reason = fmt.Sprintf("switched to %s (%s)", result.ToAccount, email)

	return result, nil
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
