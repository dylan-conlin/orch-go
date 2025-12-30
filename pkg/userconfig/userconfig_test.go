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

func intPtr(i int) *int {
	return &i
}

func TestDaemonMaxAgents_Default(t *testing.T) {
	cfg := &Config{} // No explicit daemon setting

	got := cfg.DaemonMaxAgents()
	if got != 3 {
		t.Errorf("DaemonMaxAgents() = %d, want 3 (default)", got)
	}
}

func TestDaemonMaxAgents_Explicit(t *testing.T) {
	cfg := &Config{
		Daemon: DaemonConfig{
			MaxAgents: intPtr(5),
		},
	}

	got := cfg.DaemonMaxAgents()
	if got != 5 {
		t.Errorf("DaemonMaxAgents() = %d, want 5", got)
	}
}

func TestDaemonMaxSpawnsPerHour_Default(t *testing.T) {
	cfg := &Config{} // No explicit daemon setting

	got := cfg.DaemonMaxSpawnsPerHour()
	if got != 20 {
		t.Errorf("DaemonMaxSpawnsPerHour() = %d, want 20 (default)", got)
	}
}

func TestDaemonMaxSpawnsPerHour_Explicit(t *testing.T) {
	cfg := &Config{
		Daemon: DaemonConfig{
			MaxSpawnsPerHour: intPtr(10),
		},
	}

	got := cfg.DaemonMaxSpawnsPerHour()
	if got != 10 {
		t.Errorf("DaemonMaxSpawnsPerHour() = %d, want 10", got)
	}
}

func TestDaemonConfig_ZeroValues(t *testing.T) {
	// Test that zero values are handled correctly (means no limit)
	cfg := &Config{
		Daemon: DaemonConfig{
			MaxAgents:        intPtr(0),
			MaxSpawnsPerHour: intPtr(0),
		},
	}

	// When explicitly set to 0, should return 0 (no limit)
	if cfg.DaemonMaxAgents() != 0 {
		t.Errorf("DaemonMaxAgents() = %d, want 0 when explicitly set", cfg.DaemonMaxAgents())
	}
	if cfg.DaemonMaxSpawnsPerHour() != 0 {
		t.Errorf("DaemonMaxSpawnsPerHour() = %d, want 0 when explicitly set", cfg.DaemonMaxSpawnsPerHour())
	}
}

func TestLoadDaemonConfig(t *testing.T) {
	// Save original home and restore after test
	originalHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create config with daemon section
	configDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configContent := `backend: opencode
daemon:
  max_agents: 5
  max_spawns_per_hour: 15
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.DaemonMaxAgents() != 5 {
		t.Errorf("Load() DaemonMaxAgents() = %d, want 5", cfg.DaemonMaxAgents())
	}

	if cfg.DaemonMaxSpawnsPerHour() != 15 {
		t.Errorf("Load() DaemonMaxSpawnsPerHour() = %d, want 15", cfg.DaemonMaxSpawnsPerHour())
	}
}

func TestLoadMissingDaemonSection(t *testing.T) {
	// Save original home and restore after test
	originalHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create config without daemon section
	configDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configContent := `backend: opencode
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	// Should use defaults
	if cfg.DaemonMaxAgents() != 3 {
		t.Errorf("Load() without daemon section should default MaxAgents to 3, got %d", cfg.DaemonMaxAgents())
	}

	if cfg.DaemonMaxSpawnsPerHour() != 20 {
		t.Errorf("Load() without daemon section should default MaxSpawnsPerHour to 20, got %d", cfg.DaemonMaxSpawnsPerHour())
	}
}
