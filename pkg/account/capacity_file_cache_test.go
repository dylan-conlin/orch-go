package account

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteAndReadCapacityFileCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "capacity-cache.json")

	fiveMin := time.Now().Add(5 * time.Hour)
	sevenDay := time.Now().Add(7 * 24 * time.Hour)

	accounts := []AccountWithCapacity{
		{
			Name:      "work",
			Email:     "work@example.com",
			IsDefault: true,
			Capacity: &CapacityInfo{
				FiveHourUsed:      45.0,
				FiveHourRemaining: 55.0,
				FiveHourResets:    &fiveMin,
				SevenDayUsed:      30.0,
				SevenDayRemaining: 70.0,
				SevenDayResets:    &sevenDay,
				Email:             "work@example.com",
			},
		},
		{
			Name:  "personal",
			Email: "personal@example.com",
			Capacity: &CapacityInfo{
				FiveHourUsed:      10.0,
				FiveHourRemaining: 90.0,
				SevenDayUsed:      5.0,
				SevenDayRemaining: 95.0,
				Email:             "personal@example.com",
			},
		},
	}

	// Write
	if err := WriteCapacityFileCache(path, accounts); err != nil {
		t.Fatalf("WriteCapacityFileCache() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("cache file was not created")
	}

	// Read back
	cache, err := ReadCapacityFileCache(path)
	if err != nil {
		t.Fatalf("ReadCapacityFileCache() error = %v", err)
	}

	if len(cache.Accounts) != 2 {
		t.Fatalf("got %d accounts, want 2", len(cache.Accounts))
	}

	if cache.FetchedAt.IsZero() {
		t.Error("FetchedAt should not be zero")
	}

	// Verify first account
	if cache.Accounts[0].Name != "work" {
		t.Errorf("first account name = %q, want %q", cache.Accounts[0].Name, "work")
	}
	if !cache.Accounts[0].IsDefault {
		t.Error("first account should be default")
	}
	if cache.Accounts[0].Capacity == nil {
		t.Fatal("first account capacity should not be nil")
	}
	if cache.Accounts[0].Capacity.FiveHourUsed != 45.0 {
		t.Errorf("FiveHourUsed = %f, want 45.0", cache.Accounts[0].Capacity.FiveHourUsed)
	}
	if cache.Accounts[0].Capacity.SevenDayUsed != 30.0 {
		t.Errorf("SevenDayUsed = %f, want 30.0", cache.Accounts[0].Capacity.SevenDayUsed)
	}
}

func TestReadCapacityFileCache_MissingFile(t *testing.T) {
	_, err := ReadCapacityFileCache("/nonexistent/path/capacity-cache.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestWriteCapacityFileCache_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "capacity-cache.json")

	err := WriteCapacityFileCache(path, nil)
	if err != nil {
		t.Fatalf("WriteCapacityFileCache() error = %v", err)
	}

	cache, err := ReadCapacityFileCache(path)
	if err != nil {
		t.Fatalf("ReadCapacityFileCache() error = %v", err)
	}
	if len(cache.Accounts) != 0 {
		t.Errorf("expected 0 accounts, got %d", len(cache.Accounts))
	}
}

func TestWriteCapacityFileCache_NilCapacity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "capacity-cache.json")

	accounts := []AccountWithCapacity{
		{
			Name:  "broken",
			Email: "broken@example.com",
			// Capacity is nil (API call failed)
		},
	}

	if err := WriteCapacityFileCache(path, accounts); err != nil {
		t.Fatalf("WriteCapacityFileCache() error = %v", err)
	}

	cache, err := ReadCapacityFileCache(path)
	if err != nil {
		t.Fatalf("ReadCapacityFileCache() error = %v", err)
	}
	if cache.Accounts[0].Capacity != nil {
		t.Error("expected nil capacity for broken account")
	}
}
