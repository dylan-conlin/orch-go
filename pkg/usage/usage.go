// Package usage provides Claude Max subscription usage tracking.
//
// Provides programmatic access to Claude Max weekly usage limits using
// the undocumented oauth/usage endpoint.
//
// NOTE: This uses an undocumented API endpoint that could change without notice.
// See: https://codelynx.dev/posts/claude-code-usage-limits-statusline
package usage

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/anthropic"
	"github.com/dylan-conlin/orch-go/pkg/cache"
)

// API configuration - uses shared constants from pkg/anthropic.
// Local aliases for backward compatibility with any external consumers.
const (
	UsageEndpoint   = anthropic.UsageEndpoint
	ProfileEndpoint = anthropic.ProfileEndpoint
	UserAgent       = anthropic.UserAgent
)

// usageCacheEntry represents a cached usage info with timestamp.
type usageCacheEntry struct {
	data      *UsageInfo
	timestamp time.Time
}

// usageCache provides thread-safe caching of usage API responses.
type usageCache struct {
	mu         sync.Mutex
	entries    map[string]*usageCacheEntry
	maxEntries int
	ttl        time.Duration
}

// newUsageCache creates a new usage cache with the specified TTL.
func newUsageCache(maxSize int, ttl time.Duration) *usageCache {
	bounds := cache.NewNamedCache("usage cache", maxSize, ttl)

	return &usageCache{
		entries:    make(map[string]*usageCacheEntry),
		maxEntries: bounds.MaxSize(),
		ttl:        bounds.TTL(),
	}
}

// get retrieves cached usage info if available and not expired.
// Returns (data, true) on cache hit, (nil, false) on cache miss.
func (c *usageCache) get(token string) (*UsageInfo, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[token]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Since(entry.timestamp) > c.ttl {
		delete(c.entries, token)
		return nil, false
	}

	return entry.data, true
}

// set stores usage info in the cache for the given token.
func (c *usageCache) set(token string, data *UsageInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.deleteExpiredLocked()

	if _, exists := c.entries[token]; !exists && len(c.entries) >= c.maxEntries {
		c.evictOldestLocked()
	}

	c.entries[token] = &usageCacheEntry{
		data:      data,
		timestamp: time.Now(),
	}
}

// invalidate clears all cached entries.
func (c *usageCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*usageCacheEntry, c.maxEntries)
}

func (c *usageCache) deleteExpiredLocked() {
	now := time.Now()
	for token, entry := range c.entries {
		if now.Sub(entry.timestamp) > c.ttl {
			delete(c.entries, token)
		}
	}
}

func (c *usageCache) evictOldestLocked() {
	var oldestToken string
	var oldestTime time.Time

	for token, entry := range c.entries {
		if oldestToken == "" || entry.timestamp.Before(oldestTime) {
			oldestToken = token
			oldestTime = entry.timestamp
		}
	}

	if oldestToken != "" {
		delete(c.entries, oldestToken)
	}
}

// globalUsageCache is the package-level cache instance with 60s TTL.
const defaultUsageCacheMaxEntries = 8

var globalUsageCache = newUsageCache(defaultUsageCacheMaxEntries, 60*time.Second)

// UsageLimit represents a single usage limit (5-hour or 7-day).
type UsageLimit struct {
	Utilization float64    // 0-100 percentage
	ResetsAt    *time.Time // When the limit resets
}

// Remaining returns the remaining usage (100 - utilization).
func (u *UsageLimit) Remaining() float64 {
	return 100.0 - u.Utilization
}

// TimeUntilReset returns a human-readable time until reset.
func (u *UsageLimit) TimeUntilReset() string {
	if u.ResetsAt == nil {
		return ""
	}

	now := time.Now().UTC()
	if u.ResetsAt.Before(now) || u.ResetsAt.Equal(now) {
		return "now"
	}

	delta := u.ResetsAt.Sub(now)
	days := int(delta.Hours() / 24)
	hours := int(delta.Hours()) % 24
	minutes := int(delta.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// UsageInfo contains complete usage information from the API.
type UsageInfo struct {
	FiveHour          *UsageLimit
	SevenDay          *UsageLimit
	SevenDayOpus      *UsageLimit
	SevenDayOAuthApps *UsageLimit
	Email             string
	Error             string
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

// OAuthTokenError is returned when OAuth token cannot be retrieved.
type OAuthTokenError struct {
	Message string
}

func (e *OAuthTokenError) Error() string {
	return e.Message
}

// UsageAPIError is returned when the usage API call fails.
type UsageAPIError struct {
	Message string
}

func (e *UsageAPIError) Error() string {
	return e.Message
}

// openCodeAuthPath returns the path to OpenCode's auth.json.
func openCodeAuthPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "opencode", "auth.json")
}

// GetOAuthToken retrieves the OAuth access token from OpenCode auth.json.
func GetOAuthToken() (string, error) {
	authPath := openCodeAuthPath()

	data, err := os.ReadFile(authPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", &OAuthTokenError{
				Message: fmt.Sprintf("OpenCode auth file not found: %s\nTry running OpenCode and authenticating first.", authPath),
			}
		}
		return "", &OAuthTokenError{
			Message: fmt.Sprintf("Failed to read OpenCode auth file: %v", err),
		}
	}

	var auth OpenCodeAuth
	if err := json.Unmarshal(data, &auth); err != nil {
		return "", &OAuthTokenError{
			Message: fmt.Sprintf("Failed to parse OpenCode auth file: %v", err),
		}
	}

	if auth.Anthropic.Access == "" {
		return "", &OAuthTokenError{
			Message: "No access token found in OpenCode auth file. Make sure you're logged in with your Claude Max account.",
		}
	}

	// Check if token is expired
	if auth.Anthropic.Expires > 0 {
		expiresAt := time.Unix(auth.Anthropic.Expires, 0)
		if time.Now().After(expiresAt) {
			return "", &OAuthTokenError{
				Message: "OpenCode OAuth token has expired. Restart OpenCode to refresh credentials.",
			}
		}
	}

	return auth.Anthropic.Access, nil
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

// parseLimit converts a limitResponse to UsageLimit.
func parseLimit(resp *limitResponse) *UsageLimit {
	if resp == nil {
		return nil
	}

	limit := &UsageLimit{
		Utilization: resp.Utilization,
	}

	if resp.ResetsAt != "" {
		// Parse ISO 8601 timestamp
		t, err := time.Parse(time.RFC3339, resp.ResetsAt)
		if err == nil {
			limit.ResetsAt = &t
		}
	}

	return limit
}

// fetchProfileEmail fetches the account email from the profile API.
func fetchProfileEmail(token string, client *http.Client) string {
	req, err := anthropic.NewAPIRequest("GET", ProfileEndpoint, token)
	if err != nil {
		return ""
	}

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

// FetchUsage fetches Claude Max usage information from the API with caching.
// Results are cached for 60 seconds to reduce API overhead during high-frequency operations.
func FetchUsage() *UsageInfo {
	token, err := GetOAuthToken()
	if err != nil {
		return &UsageInfo{Error: err.Error()}
	}

	// Check cache first
	if cached, ok := globalUsageCache.get(token); ok {
		return cached
	}

	// Cache miss - fetch from API
	info := fetchUsageFromAPI(token)

	// Only cache successful responses (not errors)
	if info.Error == "" {
		globalUsageCache.set(token, info)
	}

	return info
}

// fetchUsageFromAPI performs the actual API call without caching.
func fetchUsageFromAPI(token string) *UsageInfo {
	client := &http.Client{Timeout: 30 * time.Second}

	// Fetch email from profile (optional, non-blocking failure)
	email := fetchProfileEmail(token, client)

	// Fetch usage data
	req, err := anthropic.NewAPIRequest("GET", UsageEndpoint, token)
	if err != nil {
		return &UsageInfo{Error: fmt.Sprintf("Failed to create request: %v", err)}
	}

	resp, err := client.Do(req)
	if err != nil {
		return &UsageInfo{Error: fmt.Sprintf("Request failed: %v", err)}
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return &UsageInfo{
			Error: "Authentication failed (401). OAuth token may be expired. Try restarting OpenCode to refresh credentials.",
		}
	case http.StatusForbidden:
		return &UsageInfo{
			Error: "Access forbidden (403). This feature may require a Max subscription.",
		}
	}

	if resp.StatusCode != http.StatusOK {
		return &UsageInfo{
			Error: fmt.Sprintf("API returned status %d", resp.StatusCode),
		}
	}

	var apiResp usageAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return &UsageInfo{Error: fmt.Sprintf("Failed to parse response: %v", err)}
	}

	return &UsageInfo{
		FiveHour:          parseLimit(apiResp.FiveHour),
		SevenDay:          parseLimit(apiResp.SevenDay),
		SevenDayOpus:      parseLimit(apiResp.SevenDayOpus),
		SevenDayOAuthApps: parseLimit(apiResp.SevenDayOAuthApps),
		Email:             email,
	}
}

// InvalidateUsageCache clears the usage cache.
// This should be called after account switching to ensure fresh data.
func InvalidateUsageCache() {
	globalUsageCache.invalidate()
}

// FormatDisplay formats usage info for terminal display.
func FormatDisplay(info *UsageInfo) string {
	if info.Error != "" {
		return fmt.Sprintf("\u274C Error: %s", info.Error)
	}

	var lines []string
	lines = append(lines, "\U0001F4CA Claude Max Usage")
	if info.Email != "" {
		lines = append(lines, fmt.Sprintf("   Account: %s", info.Email))
	}
	lines = append(lines, "")

	formatLimit := func(name string, limit *UsageLimit, warningThreshold float64) []string {
		if limit == nil {
			return []string{fmt.Sprintf("   %s: N/A", name)}
		}

		var emoji string
		if limit.Utilization >= 95 {
			emoji = "\U0001F534" // red circle
		} else if limit.Utilization >= warningThreshold {
			emoji = "\U0001F7E1" // yellow circle
		} else {
			emoji = "\U0001F7E2" // green circle
		}

		result := []string{
			fmt.Sprintf("   %s %s: %.1f%% used (%.1f%% remaining)", emoji, name, limit.Utilization, limit.Remaining()),
		}

		if limit.ResetsAt != nil {
			resetStr := limit.TimeUntilReset()
			result = append(result, fmt.Sprintf("      Resets in: %s", resetStr))
		}

		return result
	}

	lines = append(lines, formatLimit("5-Hour Session", info.FiveHour, 80.0)...)
	lines = append(lines, "")
	lines = append(lines, formatLimit("Weekly Limit", info.SevenDay, 80.0)...)

	if info.SevenDayOpus != nil && info.SevenDayOpus.Utilization > 0 {
		lines = append(lines, "")
		lines = append(lines, formatLimit("Weekly Opus", info.SevenDayOpus, 80.0)...)
	}

	return strings.Join(lines, "\n")
}

// GetUsageSummary returns a brief one-line usage summary.
// Returns (summary_string, is_warning).
func GetUsageSummary() (string, bool) {
	info := FetchUsage()

	if info.Error != "" {
		short := info.Error
		if len(short) > 50 {
			short = short[:50] + "..."
		}
		return fmt.Sprintf("\u274C Usage check failed: %s", short), true
	}

	if info.SevenDay == nil {
		return "\U0001F4CA Usage: N/A", false
	}

	usage := info.SevenDay.Utilization
	remaining := info.SevenDay.Remaining()

	var emoji string
	var isWarning bool
	if usage >= 95 {
		emoji = "\U0001F534" // red circle
		isWarning = true
	} else if usage >= 80 {
		emoji = "\U0001F7E1" // yellow circle
		isWarning = true
	} else {
		emoji = "\U0001F7E2" // green circle
		isWarning = false
	}

	resetStr := ""
	if info.SevenDay.ResetsAt != nil {
		resetStr = fmt.Sprintf(" (resets in %s)", info.SevenDay.TimeUntilReset())
	}

	return fmt.Sprintf("%s Weekly usage: %.0f%% used, %.0f%% remaining%s", emoji, usage, remaining, resetStr), isWarning
}
