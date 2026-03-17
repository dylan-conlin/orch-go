package account

import (
	"encoding/json"
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

// ============================================================================
// Account Schema Tests (Tier, Role, ConfigDir fields)
// ============================================================================

func TestSaveAndLoadConfig_WithTierRoleConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	cfg := &Config{
		Accounts: map[string]Account{
			"work": {
				Email:        "work@example.com",
				RefreshToken: "work-token",
				Source:       "saved",
				Tier:         "20x",
				Role:         "primary",
				ConfigDir:    "~/.claude",
			},
			"personal": {
				Email:        "personal@example.com",
				RefreshToken: "personal-token",
				Source:       "saved",
				Tier:         "5x",
				Role:         "spillover",
				ConfigDir:    "~/.claude-personal",
			},
		},
		Default: "personal",
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	work := loaded.Accounts["work"]
	if work.Tier != "20x" {
		t.Errorf("work.Tier = %q, want %q", work.Tier, "20x")
	}
	if work.Role != "primary" {
		t.Errorf("work.Role = %q, want %q", work.Role, "primary")
	}
	if work.ConfigDir != "~/.claude" {
		t.Errorf("work.ConfigDir = %q, want %q", work.ConfigDir, "~/.claude")
	}

	personal := loaded.Accounts["personal"]
	if personal.Tier != "5x" {
		t.Errorf("personal.Tier = %q, want %q", personal.Tier, "5x")
	}
	if personal.Role != "spillover" {
		t.Errorf("personal.Role = %q, want %q", personal.Role, "spillover")
	}
	if personal.ConfigDir != "~/.claude-personal" {
		t.Errorf("personal.ConfigDir = %q, want %q", personal.ConfigDir, "~/.claude-personal")
	}
}

func TestSaveAndLoadConfig_BackwardCompatible(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	// Save config WITHOUT new fields (backward compat)
	cfg := &Config{
		Accounts: map[string]Account{
			"old": {
				Email:        "old@example.com",
				RefreshToken: "old-token",
				Source:       "saved",
			},
		},
		Default: "old",
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	old := loaded.Accounts["old"]
	if old.Tier != "" {
		t.Errorf("old.Tier = %q, want empty (backward compat)", old.Tier)
	}
	if old.Role != "" {
		t.Errorf("old.Role = %q, want empty (backward compat)", old.Role)
	}
	if old.ConfigDir != "" {
		t.Errorf("old.ConfigDir = %q, want empty (backward compat)", old.ConfigDir)
	}
}

func TestGetConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	cfg := &Config{
		Accounts: map[string]Account{
			"work": {
				Email:     "work@example.com",
				Source:    "saved",
				ConfigDir: "~/.claude",
			},
			"personal": {
				Email:     "personal@example.com",
				Source:    "saved",
				ConfigDir: "~/.claude-personal",
			},
		},
		Default: "work",
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if got := GetConfigDir("work"); got != "~/.claude" {
		t.Errorf("GetConfigDir(work) = %q, want %q", got, "~/.claude")
	}
	if got := GetConfigDir("personal"); got != "~/.claude-personal" {
		t.Errorf("GetConfigDir(personal) = %q, want %q", got, "~/.claude-personal")
	}
	if got := GetConfigDir("nonexistent"); got != "" {
		t.Errorf("GetConfigDir(nonexistent) = %q, want empty", got)
	}
	if got := GetConfigDir(""); got != "" {
		t.Errorf("GetConfigDir('') = %q, want empty", got)
	}
}

// ============================================================================
// TokenRefreshError Tests
// ============================================================================

func TestTokenRefreshError(t *testing.T) {
	// Test basic error
	err := &TokenRefreshError{Message: "token expired"}
	if got := err.Error(); got != "token expired" {
		t.Errorf("TokenRefreshError.Error() = %q, want %q", got, "token expired")
	}

	// Test WithAccount
	errWithAccount := err.WithAccount("personal")
	if errWithAccount.AccountName != "personal" {
		t.Errorf("WithAccount().AccountName = %q, want %q", errWithAccount.AccountName, "personal")
	}
	if errWithAccount.Message != "token expired" {
		t.Errorf("WithAccount().Message = %q, want %q", errWithAccount.Message, "token expired")
	}

	// Test ActionableGuidance with account name
	guidance := errWithAccount.ActionableGuidance()
	expected := "To re-authorize: orch account add personal"
	if guidance != expected {
		t.Errorf("ActionableGuidance() = %q, want %q", guidance, expected)
	}

	// Test ActionableGuidance without account name
	guidanceNoAccount := err.ActionableGuidance()
	expectedNoAccount := "To re-authorize: orch account add <name>"
	if guidanceNoAccount != expectedNoAccount {
		t.Errorf("ActionableGuidance() without account = %q, want %q", guidanceNoAccount, expectedNoAccount)
	}
}

// ============================================================================
// SaveOpenCodeAuth Tests
// ============================================================================

// TestSaveOpenCodeAuth_PreservesOtherProviders tests that saving anthropic auth
// doesn't nuke other provider credentials (e.g., openai OAuth).
func TestSaveOpenCodeAuth_PreservesOtherProviders(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create the auth directory
	authDir := filepath.Join(tmpDir, ".local", "share", "opencode")
	if err := os.MkdirAll(authDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write an existing auth.json with both anthropic AND openai sections
	existingAuth := map[string]interface{}{
		"anthropic": map[string]interface{}{
			"type":    "oauth",
			"refresh": "old-anthropic-refresh",
			"access":  "old-anthropic-access",
			"expires": 1000000,
		},
		"openai": map[string]interface{}{
			"type":    "oauth",
			"refresh": "openai-refresh-token",
			"access":  "openai-access-token",
			"expires": 2000000,
		},
	}
	existingData, _ := json.MarshalIndent(existingAuth, "", "  ")
	if err := os.WriteFile(filepath.Join(authDir, "auth.json"), existingData, 0600); err != nil {
		t.Fatal(err)
	}

	// Now save new anthropic credentials (simulating account switch)
	auth := &OpenCodeAuth{}
	auth.Anthropic.Type = "oauth"
	auth.Anthropic.Refresh = "new-anthropic-refresh"
	auth.Anthropic.Access = "new-anthropic-access"
	auth.Anthropic.Expires = 9999999

	if err := SaveOpenCodeAuth(auth); err != nil {
		t.Fatalf("SaveOpenCodeAuth() error = %v", err)
	}

	// Read back the file and verify openai section is preserved
	savedData, err := os.ReadFile(filepath.Join(authDir, "auth.json"))
	if err != nil {
		t.Fatal(err)
	}

	var saved map[string]interface{}
	if err := json.Unmarshal(savedData, &saved); err != nil {
		t.Fatal(err)
	}

	// Check anthropic was updated
	anthropic, ok := saved["anthropic"].(map[string]interface{})
	if !ok {
		t.Fatal("anthropic section missing from saved auth.json")
	}
	if anthropic["refresh"] != "new-anthropic-refresh" {
		t.Errorf("anthropic.refresh = %v, want %v", anthropic["refresh"], "new-anthropic-refresh")
	}

	// Check openai was preserved
	openai, ok := saved["openai"].(map[string]interface{})
	if !ok {
		t.Fatal("openai section was nuked from auth.json — this is the bug")
	}
	if openai["refresh"] != "openai-refresh-token" {
		t.Errorf("openai.refresh = %v, want %v", openai["refresh"], "openai-refresh-token")
	}
	if openai["access"] != "openai-access-token" {
		t.Errorf("openai.access = %v, want %v", openai["access"], "openai-access-token")
	}
}

// TestSaveOpenCodeAuth_NoExistingFile tests that saving works when no auth.json exists yet.
func TestSaveOpenCodeAuth_NoExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	auth := &OpenCodeAuth{}
	auth.Anthropic.Type = "oauth"
	auth.Anthropic.Refresh = "fresh-refresh"
	auth.Anthropic.Access = "fresh-access"
	auth.Anthropic.Expires = 5555555

	if err := SaveOpenCodeAuth(auth); err != nil {
		t.Fatalf("SaveOpenCodeAuth() error = %v", err)
	}

	// Verify file was created with anthropic section
	savedData, err := os.ReadFile(OpenCodeAuthPath())
	if err != nil {
		t.Fatal(err)
	}

	var saved map[string]interface{}
	if err := json.Unmarshal(savedData, &saved); err != nil {
		t.Fatal(err)
	}

	anthropic, ok := saved["anthropic"].(map[string]interface{})
	if !ok {
		t.Fatal("anthropic section missing")
	}
	if anthropic["refresh"] != "fresh-refresh" {
		t.Errorf("anthropic.refresh = %v, want %v", anthropic["refresh"], "fresh-refresh")
	}
}

// ============================================================================
// AddAccount Metadata Preservation Tests
// ============================================================================

func TestAddAccountPreservesMetadata_ExistingAccount(t *testing.T) {
	// Simulates the re-auth flow: account exists with metadata, re-add preserves it
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	// Pre-populate config with metadata
	cfg := &Config{
		Accounts: map[string]Account{
			"work": {
				Email:        "old@example.com",
				RefreshToken: "old-token",
				Source:       "saved",
				Tier:         "20x",
				Role:         "primary",
				ConfigDir:    "~/.claude",
			},
		},
		Default: "work",
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Simulate what AddAccount does internally (without the actual OAuth flow):
	// Load config, create new account, merge existing metadata, save
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	acc := Account{
		Email:        "new@example.com",
		RefreshToken: "new-token",
		Source:       "saved",
	}
	// This is the fix: merge existing metadata
	if existing, ok := config.Accounts["work"]; ok {
		acc.Tier = existing.Tier
		acc.Role = existing.Role
		acc.ConfigDir = existing.ConfigDir
	}
	config.Save("work", acc, false)

	if err := SaveConfig(config); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Verify metadata survived
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	work := loaded.Accounts["work"]
	if work.Email != "new@example.com" {
		t.Errorf("work.Email = %q, want %q (should be updated)", work.Email, "new@example.com")
	}
	if work.RefreshToken != "new-token" {
		t.Errorf("work.RefreshToken = %q, want %q (should be updated)", work.RefreshToken, "new-token")
	}
	if work.Tier != "20x" {
		t.Errorf("work.Tier = %q, want %q (should be preserved)", work.Tier, "20x")
	}
	if work.Role != "primary" {
		t.Errorf("work.Role = %q, want %q (should be preserved)", work.Role, "primary")
	}
	if work.ConfigDir != "~/.claude" {
		t.Errorf("work.ConfigDir = %q, want %q (should be preserved)", work.ConfigDir, "~/.claude")
	}
}

func TestAddAccountNoMetadata_NewAccount(t *testing.T) {
	// When account doesn't exist yet, no metadata to merge
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	acc := Account{
		Email:        "new@example.com",
		RefreshToken: "new-token",
		Source:       "saved",
	}
	// No existing account — merge is a no-op
	if existing, ok := config.Accounts["work"]; ok {
		acc.Tier = existing.Tier
		acc.Role = existing.Role
		acc.ConfigDir = existing.ConfigDir
	}
	config.Save("work", acc, true)

	if err := SaveConfig(config); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	work := loaded.Accounts["work"]
	if work.Tier != "" {
		t.Errorf("work.Tier = %q, want empty (new account)", work.Tier)
	}
	if work.Role != "" {
		t.Errorf("work.Role = %q, want empty (new account)", work.Role)
	}
	if work.ConfigDir != "" {
		t.Errorf("work.ConfigDir = %q, want empty (new account)", work.ConfigDir)
	}
}

// ============================================================================
// LoadAndSaveConfig Tests
// ============================================================================

func TestLoadAndSaveConfig_AtomicModify(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	// Pre-populate
	cfg := &Config{
		Accounts: map[string]Account{
			"work": {
				Email:        "work@example.com",
				RefreshToken: "old-token",
				Source:       "saved",
				Tier:         "20x",
			},
		},
		Default: "work",
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Use LoadAndSaveConfig to atomically update token
	err := LoadAndSaveConfig(func(cfg *Config) error {
		acc := cfg.Accounts["work"]
		acc.RefreshToken = "new-token"
		cfg.Accounts["work"] = acc
		return nil
	})
	if err != nil {
		t.Fatalf("LoadAndSaveConfig() error = %v", err)
	}

	// Verify update
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	work := loaded.Accounts["work"]
	if work.RefreshToken != "new-token" {
		t.Errorf("work.RefreshToken = %q, want %q", work.RefreshToken, "new-token")
	}
	if work.Tier != "20x" {
		t.Errorf("work.Tier = %q, want %q (should be preserved)", work.Tier, "20x")
	}
}
