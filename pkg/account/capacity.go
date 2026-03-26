package account

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

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

	// SevenDayOpusUsed is the Opus-specific weekly utilization (0-100).
	// This tracks Opus usage separately from generic Claude capacity.
	// When SevenDayOpusRemaining is low but SevenDayRemaining is healthy,
	// Sonnet is still viable on the same account.
	SevenDayOpusUsed float64
	// SevenDayOpusRemaining is the remaining Opus-specific weekly capacity (0-100).
	SevenDayOpusRemaining float64
	// SevenDayOpusResets is when the Opus-specific weekly limit resets.
	SevenDayOpusResets *time.Time

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

// IsOpusHealthy returns true if the Opus-specific weekly capacity is above 10%.
// When Opus capacity data is not available (SevenDayOpusUsed == 0 and SevenDayOpusRemaining == 0),
// falls back to the generic IsHealthy check (assumes Opus is available if Claude is healthy).
func (c *CapacityInfo) IsOpusHealthy() bool {
	if c.Error != "" {
		return false
	}
	// If the API didn't return Opus-specific data, fall back to generic health
	if c.SevenDayOpusUsed == 0 && c.SevenDayOpusRemaining == 0 {
		return c.IsHealthy()
	}
	return c.SevenDayOpusRemaining > 10
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

	// Parse 7-day Opus-specific limit
	if apiResp.SevenDayOpus != nil {
		capacity.SevenDayOpusUsed = apiResp.SevenDayOpus.Utilization
		capacity.SevenDayOpusRemaining = 100.0 - apiResp.SevenDayOpus.Utilization
		if apiResp.SevenDayOpus.ResetsAt != "" {
			if t, err := time.Parse(time.RFC3339, apiResp.SevenDayOpus.ResetsAt); err == nil {
				capacity.SevenDayOpusResets = &t
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
// Uses tier-weighted 5-hour headroom as the primary metric.
// Returns empty string if no accounts have healthy capacity.
func FindBestAccount() (string, *CapacityInfo, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return "", nil, err
	}

	var bestName string
	var bestCapacity *CapacityInfo
	var bestAbsHeadroom float64 = -1

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

		// Tier-weighted 5-hour headroom as primary metric
		tier := ParseTierMultiplier(acc.Tier)
		absHeadroom := capacity.FiveHourRemaining * tier
		if absHeadroom > bestAbsHeadroom {
			bestName = name
			bestCapacity = capacity
			bestAbsHeadroom = absHeadroom
		}
	}

	if bestName == "" {
		return "", nil, &CapacityError{Message: "no healthy accounts found"}
	}

	return bestName, bestCapacity, nil
}

// RecommendAccount returns the name of the recommended account for spawning.
// Uses the same tier-weighted 5h headroom algorithm as resolveAccount:
//  1. Collect capacity for all accounts (regardless of role)
//  2. Compute absolute 5h headroom = FiveHourRemaining% * tier_multiplier
//  3. Pick the account with highest absolute 5h headroom
//  4. Tie-break: tier-weighted weekly headroom, then alphabetical name
//  5. Without capacity data, recommend first primary account
//
// Returns empty string if no accounts are configured.
func RecommendAccount(accounts []AccountInfo, capacityFetcher func(string) *CapacityInfo) string {
	if len(accounts) == 0 {
		return ""
	}

	// Build tier lookup from accounts
	tierByName := make(map[string]string)
	for _, acc := range accounts {
		tierByName[acc.Name] = acc.Tier
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

	// Tier-weighted 5h headroom: pick account with most absolute 5h capacity
	type candidate struct {
		name        string
		capacity    *CapacityInfo
		fiveHourAbs float64
		weeklyAbs   float64
	}
	var candidates []candidate
	for _, name := range allNames {
		cap := capacityFetcher(name)
		if cap != nil {
			tier := ParseTierMultiplier(tierByName[name])
			candidates = append(candidates, candidate{
				name:        name,
				capacity:    cap,
				fiveHourAbs: cap.FiveHourRemaining * tier,
				weeklyAbs:   cap.SevenDayRemaining * tier,
			})
		}
	}

	if len(candidates) == 0 {
		return allNames[0]
	}

	// Sort: highest absolute 5h headroom first, then absolute weekly, then name
	sort.Slice(candidates, func(i, j int) bool {
		ci, cj := candidates[i], candidates[j]
		if ci.fiveHourAbs != cj.fiveHourAbs {
			return ci.fiveHourAbs > cj.fiveHourAbs
		}
		if ci.weeklyAbs != cj.weeklyAbs {
			return ci.weeklyAbs > cj.weeklyAbs
		}
		return ci.name < cj.name
	})

	return candidates[0].name
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

// ParseTierMultiplier parses a tier string like "5x" or "20x" into a float64 multiplier.
// Returns 1.0 if the tier string is empty or unparseable (safe default).
func ParseTierMultiplier(tier string) float64 {
	tier = strings.TrimSpace(strings.ToLower(tier))
	if tier == "" {
		return 1.0
	}
	tier = strings.TrimSuffix(tier, "x")
	val, err := strconv.ParseFloat(tier, 64)
	if err != nil || val <= 0 {
		return 1.0
	}
	return val
}
