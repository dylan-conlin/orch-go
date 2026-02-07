// Package advisor provides model recommendation functionality using live API data.
package advisor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// OpenRouterModel represents a model from the OpenRouter API.
type OpenRouterModel struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	ContextLength int          `json:"context_length"`
	Pricing       Pricing      `json:"pricing"`
	Architecture  Architecture `json:"architecture"`
	TopProvider   TopProvider  `json:"top_provider"`
}

// Pricing represents the cost structure for a model.
type Pricing struct {
	Prompt     string `json:"prompt"`               // Cost per input token (as string, e.g., "0.00000175")
	Completion string `json:"completion"`           // Cost per output token
	Request    string `json:"request,omitempty"`    // Per-request cost
	Image      string `json:"image,omitempty"`      // Cost per image
	WebSearch  string `json:"web_search,omitempty"` // Cost per web search
}

// Architecture represents the model's technical details.
type Architecture struct {
	Modality         string   `json:"modality"`
	InputModalities  []string `json:"input_modalities"`
	OutputModalities []string `json:"output_modalities"`
	Tokenizer        string   `json:"tokenizer"`
}

// TopProvider represents the best provider for this model.
type TopProvider struct {
	ContextLength       int  `json:"context_length"`
	MaxCompletionTokens int  `json:"max_completion_tokens"`
	IsModerated         bool `json:"is_moderated"`
}

// OpenRouterResponse is the full API response structure.
type OpenRouterResponse struct {
	Data []OpenRouterModel `json:"data"`
}

// Client handles fetching model data from OpenRouter API.
type Client struct {
	apiURL     string
	cachePath  string
	cacheTTL   time.Duration
	httpClient *http.Client
}

// NewClient creates a new OpenRouter API client with default settings.
func NewClient() *Client {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	cachePath := filepath.Join(home, ".orch", "model-cache.json")

	return &Client{
		apiURL:    "https://openrouter.ai/api/v1/models",
		cachePath: cachePath,
		cacheTTL:  24 * time.Hour, // Refresh daily
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchModels retrieves model data from OpenRouter API or cache.
// Returns cached data if fresh (< 24 hours old), otherwise fetches from API.
func (c *Client) FetchModels() ([]OpenRouterModel, error) {
	// Try cache first
	if models, err := c.loadCache(); err == nil {
		return models, nil
	}

	// Cache miss or stale - fetch from API
	return c.fetchFromAPI()
}

// FetchModelsForce bypasses cache and fetches fresh data from API.
func (c *Client) FetchModelsForce() ([]OpenRouterModel, error) {
	return c.fetchFromAPI()
}

// fetchFromAPI retrieves model data from OpenRouter API and updates cache.
func (c *Client) fetchFromAPI() ([]OpenRouterModel, error) {
	resp, err := c.httpClient.Get(c.apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from OpenRouter API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenRouter API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp OpenRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}

	// Save to cache
	if err := c.saveCache(apiResp.Data); err != nil {
		// Log but don't fail - we have the data
		fmt.Fprintf(os.Stderr, "Warning: failed to save cache: %v\n", err)
	}

	return apiResp.Data, nil
}

// loadCache loads models from cache if fresh.
func (c *Client) loadCache() ([]OpenRouterModel, error) {
	// Check if cache exists and is fresh
	info, err := os.Stat(c.cachePath)
	if err != nil {
		return nil, fmt.Errorf("cache not found: %w", err)
	}

	// Check if cache is stale
	age := time.Since(info.ModTime())
	if age > c.cacheTTL {
		return nil, fmt.Errorf("cache is stale (age: %v, TTL: %v)", age, c.cacheTTL)
	}

	// Read cache file
	data, err := os.ReadFile(c.cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache: %w", err)
	}

	var models []OpenRouterModel
	if err := json.Unmarshal(data, &models); err != nil {
		return nil, fmt.Errorf("failed to parse cache: %w", err)
	}

	return models, nil
}

// saveCache saves models to cache file.
func (c *Client) saveCache(models []OpenRouterModel) error {
	// Ensure cache directory exists
	dir := filepath.Dir(c.cachePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(models, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal models: %w", err)
	}

	// Write to file
	if err := os.WriteFile(c.cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache: %w", err)
	}

	return nil
}

// CacheAge returns the age of the cache file, or -1 if not found.
func (c *Client) CacheAge() time.Duration {
	info, err := os.Stat(c.cachePath)
	if err != nil {
		return -1
	}
	return time.Since(info.ModTime())
}
