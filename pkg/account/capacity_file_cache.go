package account

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CachedAccountCapacity holds account capacity data with a fetch timestamp.
type CachedAccountCapacity struct {
	Name      string        `json:"name"`
	Email     string        `json:"email,omitempty"`
	IsDefault bool          `json:"is_default,omitempty"`
	Capacity  *CapacityInfo `json:"capacity,omitempty"`
}

// CapacityFileCache holds the full cache file contents.
type CapacityFileCache struct {
	FetchedAt time.Time                `json:"fetched_at"`
	Accounts  []CachedAccountCapacity  `json:"accounts"`
}

// DefaultCapacityFileCachePath returns ~/.orch/capacity-cache.json.
func DefaultCapacityFileCachePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".orch/capacity-cache.json"
	}
	return filepath.Join(homeDir, ".orch", "capacity-cache.json")
}

// WriteCapacityFileCache writes account capacity data to a JSON file atomically.
func WriteCapacityFileCache(path string, accounts []AccountWithCapacity) error {
	cache := CapacityFileCache{
		FetchedAt: time.Now(),
	}
	for _, awc := range accounts {
		cache.Accounts = append(cache.Accounts, CachedAccountCapacity{
			Name:      awc.Name,
			Email:     awc.Email,
			IsDefault: awc.IsDefault,
			Capacity:  awc.Capacity,
		})
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal capacity cache: %w", err)
	}

	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp cache file: %w", err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename cache file: %w", err)
	}

	return nil
}

// ReadCapacityFileCache reads cached capacity data from a JSON file.
// Returns the cache contents and an error if the file is missing or unreadable.
func ReadCapacityFileCache(path string) (*CapacityFileCache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read capacity cache: %w", err)
	}

	var cache CapacityFileCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse capacity cache: %w", err)
	}

	return &cache, nil
}
