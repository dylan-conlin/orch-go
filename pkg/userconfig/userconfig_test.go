package userconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Backend != "opencode" {
		t.Errorf("DefaultConfig().Backend = %q, want %q", cfg.Backend, "opencode")
	}

	if !cfg.NotificationsEnabled() {
		t.Error("DefaultConfig().NotificationsEnabled() = false, want true")
	}
}

func TestNotificationsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  *bool
		expected bool
	}{
		{
			name:     "nil defaults to true",
			enabled:  nil,
			expected: true,
		},
		{
			name:     "explicit true",
			enabled:  boolPtr(true),
			expected: true,
		},
		{
			name:     "explicit false",
			enabled:  boolPtr(false),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Notifications: NotificationConfig{
					Enabled: tt.enabled,
				},
			}
			if got := cfg.NotificationsEnabled(); got != tt.expected {
				t.Errorf("NotificationsEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLoadNonExistent(t *testing.T) {
	// Save original home and restore after test
	originalHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg == nil {
		t.Fatal("Load() returned nil config for non-existent file")
	}

	// Should return default config
	if !cfg.NotificationsEnabled() {
		t.Error("Load() for non-existent file should default notifications to enabled")
	}
}

func TestLoadExistingConfig(t *testing.T) {
	// Save original home and restore after test
	originalHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create config directory and file
	configDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configContent := `backend: opencode
auto_export_transcript: true
notifications:
  enabled: false
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.Backend != "opencode" {
		t.Errorf("Load() Backend = %q, want %q", cfg.Backend, "opencode")
	}

	if !cfg.AutoExportTranscript {
		t.Error("Load() AutoExportTranscript = false, want true")
	}

	if cfg.NotificationsEnabled() {
		t.Error("Load() NotificationsEnabled() = true, want false")
	}
}

func TestSave(t *testing.T) {
	// Save original home and restore after test
	originalHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	enabled := false
	cfg := &Config{
		Backend:              "opencode",
		AutoExportTranscript: true,
		Notifications: NotificationConfig{
			Enabled: &enabled,
		},
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load it back
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() after Save() error = %v", err)
	}

	if loaded.Backend != cfg.Backend {
		t.Errorf("Loaded Backend = %q, want %q", loaded.Backend, cfg.Backend)
	}

	if loaded.AutoExportTranscript != cfg.AutoExportTranscript {
		t.Errorf("Loaded AutoExportTranscript = %v, want %v", loaded.AutoExportTranscript, cfg.AutoExportTranscript)
	}

	if loaded.NotificationsEnabled() != cfg.NotificationsEnabled() {
		t.Errorf("Loaded NotificationsEnabled() = %v, want %v", loaded.NotificationsEnabled(), cfg.NotificationsEnabled())
	}
}

func TestLoadMissingNotificationsSection(t *testing.T) {
	// Save original home and restore after test
	originalHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create config without notifications section (like existing configs)
	configDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configContent := `backend: opencode
auto_export_transcript: true
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	// Should default to enabled when notifications section is missing
	if !cfg.NotificationsEnabled() {
		t.Error("Load() without notifications section should default to enabled")
	}
}

func boolPtr(b bool) *bool {
	return &b
}
