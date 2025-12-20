package account

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_NotExist(t *testing.T) {
	// Save original path and restore after test
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig() returned nil config")
	}

	if len(cfg.Accounts) != 0 {
		t.Errorf("LoadConfig() accounts = %d, want 0", len(cfg.Accounts))
	}
}

func TestConfigOperations(t *testing.T) {
	cfg := &Config{
		Accounts: make(map[string]Account),
	}

	// Test Save
	acc := Account{
		Email:        "test@example.com",
		RefreshToken: "test-token",
		Source:       "saved",
	}
	cfg.Save("test", acc, true)

	if cfg.Default != "test" {
		t.Errorf("After Save with setDefault=true, Default = %q, want %q", cfg.Default, "test")
	}

	// Test Get
	got, err := cfg.Get("test")
	if err != nil {
		t.Errorf("Get(test) error = %v", err)
	}
	if got.Email != "test@example.com" {
		t.Errorf("Get(test).Email = %q, want %q", got.Email, "test@example.com")
	}

	// Test Get not found
	_, err = cfg.Get("notexist")
	if err == nil {
		t.Error("Get(notexist) should return error")
	}

	// Test List
	names := cfg.List()
	if len(names) != 1 || names[0] != "test" {
		t.Errorf("List() = %v, want [test]", names)
	}

	// Test Remove
	err = cfg.Remove("test")
	if err != nil {
		t.Errorf("Remove(test) error = %v", err)
	}

	if len(cfg.Accounts) != 0 {
		t.Errorf("After Remove, accounts = %d, want 0", len(cfg.Accounts))
	}

	// Default should be cleared
	if cfg.Default != "" {
		t.Errorf("After Remove, Default = %q, want empty", cfg.Default)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Use temp directory
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create .orch directory
	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	cfg := &Config{
		Accounts: map[string]Account{
			"personal": {
				Email:        "user@example.com",
				RefreshToken: "token123",
				Source:       "saved",
			},
		},
		Default: "personal",
	}

	// Save
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Load
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loaded.Default != "personal" {
		t.Errorf("Loaded Default = %q, want %q", loaded.Default, "personal")
	}

	acc, ok := loaded.Accounts["personal"]
	if !ok {
		t.Fatal("Loaded config missing 'personal' account")
	}

	if acc.Email != "user@example.com" {
		t.Errorf("Loaded Email = %q, want %q", acc.Email, "user@example.com")
	}
}
