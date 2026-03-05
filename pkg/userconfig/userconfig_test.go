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

func TestLoadWithMetaExistingConfig(t *testing.T) {
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
allow_anthropic_opencode: true
default_model: gpt4o
default_tier: full
notifications:
  enabled: false
reflect:
  enabled: false
daemon:
  label: custom:ready
session:
  orchestrator_checkpoints:
    warning_minutes: 300
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	_, meta, err := LoadWithMeta()
	if err != nil {
		t.Fatalf("LoadWithMeta() error = %v, want nil", err)
	}

	if meta == nil {
		t.Fatal("LoadWithMeta() returned nil meta")
	}

	if !meta.Explicit["backend"] {
		t.Error("Expected backend to be explicit")
	}
	if !meta.Explicit["default_model"] {
		t.Error("Expected default_model to be explicit")
	}
	if !meta.Explicit["allow_anthropic_opencode"] {
		t.Error("Expected allow_anthropic_opencode to be explicit")
	}
	if !meta.Explicit["default_tier"] {
		t.Error("Expected default_tier to be explicit")
	}
	if !meta.Explicit["notifications"] {
		t.Error("Expected notifications to be explicit")
	}
	if !meta.Explicit["reflect"] {
		t.Error("Expected reflect to be explicit")
	}
	if !meta.Explicit["daemon"] {
		t.Error("Expected daemon to be explicit")
	}
	if !meta.Explicit["session"] {
		t.Error("Expected session to be explicit")
	}
	if !meta.ExplicitNotifications["enabled"] {
		t.Error("Expected notifications.enabled to be explicit")
	}
	if !meta.ExplicitReflect["enabled"] {
		t.Error("Expected reflect.enabled to be explicit")
	}
	if !meta.ExplicitDaemon["label"] {
		t.Error("Expected daemon.label to be explicit")
	}
	if !meta.ExplicitSessionOrchestratorCheckpts["warning_minutes"] {
		t.Error("Expected session.orchestrator_checkpoints.warning_minutes to be explicit")
	}
}

func TestLoadWithMetaMissingConfig(t *testing.T) {
	// Save original home and restore after test
	originalHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	_, meta, err := LoadWithMeta()
	if err != nil {
		t.Fatalf("LoadWithMeta() error = %v, want nil", err)
	}

	if meta == nil {
		t.Fatal("LoadWithMeta() returned nil meta")
	}

	if len(meta.Explicit) != 0 {
		t.Errorf("Expected no explicit keys, got %d", len(meta.Explicit))
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

// =============================================================================
// Tests for ReflectConfig
// =============================================================================

func TestReflectEnabled(t *testing.T) {
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
				Reflect: ReflectConfig{
					Enabled: tt.enabled,
				},
			}
			if got := cfg.ReflectEnabled(); got != tt.expected {
				t.Errorf("ReflectEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestReflectIntervalMinutes(t *testing.T) {
	tests := []struct {
		name     string
		interval *int
		expected int
	}{
		{
			name:     "nil defaults to 60",
			interval: nil,
			expected: 60,
		},
		{
			name:     "explicit value",
			interval: intPtr(30),
			expected: 30,
		},
		{
			name:     "explicit zero",
			interval: intPtr(0),
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Reflect: ReflectConfig{
					IntervalMinutes: tt.interval,
				},
			}
			if got := cfg.ReflectIntervalMinutes(); got != tt.expected {
				t.Errorf("ReflectIntervalMinutes() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestReflectCreateIssues(t *testing.T) {
	tests := []struct {
		name         string
		createIssues *bool
		expected     bool
	}{
		{
			name:         "nil defaults to true",
			createIssues: nil,
			expected:     true,
		},
		{
			name:         "explicit true",
			createIssues: boolPtr(true),
			expected:     true,
		},
		{
			name:         "explicit false",
			createIssues: boolPtr(false),
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Reflect: ReflectConfig{
					CreateIssues: tt.createIssues,
				},
			}
			if got := cfg.ReflectCreateIssues(); got != tt.expected {
				t.Errorf("ReflectCreateIssues() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLoadReflectConfig(t *testing.T) {
	// Save original home and restore after test
	originalHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create config directory and file with reflect settings
	configDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configContent := `backend: opencode
reflect:
  enabled: false
  interval_minutes: 30
  create_issues: false
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.ReflectEnabled() {
		t.Error("Load() ReflectEnabled() = true, want false")
	}

	if cfg.ReflectIntervalMinutes() != 30 {
		t.Errorf("Load() ReflectIntervalMinutes() = %d, want 30", cfg.ReflectIntervalMinutes())
	}

	if cfg.ReflectCreateIssues() {
		t.Error("Load() ReflectCreateIssues() = true, want false")
	}
}

func TestLoadMissingReflectSection(t *testing.T) {
	// Save original home and restore after test
	originalHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create config without reflect section
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

	// Should default to enabled
	if !cfg.ReflectEnabled() {
		t.Error("Load() without reflect section should default to enabled")
	}

	// Should default to 60 minutes
	if cfg.ReflectIntervalMinutes() != 60 {
		t.Errorf("Load() without reflect section should default to 60 minutes, got %d", cfg.ReflectIntervalMinutes())
	}

	// Should default to creating issues
	if !cfg.ReflectCreateIssues() {
		t.Error("Load() without reflect section should default to creating issues")
	}
}

// =============================================================================
// Tests for DefaultTier
// =============================================================================

func TestGetDefaultTier(t *testing.T) {
	tests := []struct {
		name        string
		defaultTier string
		expected    string
	}{
		{
			name:        "empty string returns empty (use skill defaults)",
			defaultTier: "",
			expected:    "",
		},
		{
			name:        "light returns empty (use skill defaults)",
			defaultTier: "light",
			expected:    "",
		},
		{
			name:        "full returns full",
			defaultTier: "full",
			expected:    "full",
		},
		{
			name:        "invalid value returns empty (use skill defaults)",
			defaultTier: "invalid",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				DefaultTier: tt.defaultTier,
			}
			if got := cfg.GetDefaultTier(); got != tt.expected {
				t.Errorf("GetDefaultTier() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLoadDefaultTierConfig(t *testing.T) {
	// Save original home and restore after test
	originalHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create config directory and file with default_tier setting
	configDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configContent := `backend: opencode
default_tier: full
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.DefaultTier != "full" {
		t.Errorf("Load() DefaultTier = %q, want %q", cfg.DefaultTier, "full")
	}

	if cfg.GetDefaultTier() != "full" {
		t.Errorf("Load() GetDefaultTier() = %q, want %q", cfg.GetDefaultTier(), "full")
	}
}

func TestLoadMissingDefaultTier(t *testing.T) {
	// Save original home and restore after test
	originalHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create config without default_tier
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

	// Should default to empty (use skill defaults)
	if cfg.GetDefaultTier() != "" {
		t.Errorf("Load() without default_tier should return empty string, got %q", cfg.GetDefaultTier())
	}
}

// =============================================================================
// Tests for DefaultModel
// =============================================================================

func TestDefaultModelConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Default config should have empty DefaultModel
	if cfg.DefaultModel != "" {
		t.Errorf("DefaultConfig().DefaultModel = %q, want empty", cfg.DefaultModel)
	}
}

func TestLoadDefaultModelConfig(t *testing.T) {
	// Save original home and restore after test
	originalHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create config directory and file with default_model
	configDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configContent := `backend: opencode
default_model: gpt4o
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.DefaultModel != "gpt4o" {
		t.Errorf("Load() DefaultModel = %q, want %q", cfg.DefaultModel, "gpt4o")
	}
}

func TestSaveDefaultModelConfig(t *testing.T) {
	// Save original home and restore after test
	originalHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	cfg := &Config{
		Backend:      "opencode",
		DefaultModel: "gpt4o",
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load it back
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() after Save() error = %v", err)
	}

	if loaded.DefaultModel != "gpt4o" {
		t.Errorf("Loaded DefaultModel = %q, want %q", loaded.DefaultModel, "gpt4o")
	}
}

func TestLoadMissingDefaultModel(t *testing.T) {
	// Save original home and restore after test
	originalHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create config without default_model
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

	// Should default to empty (use hardcoded default)
	if cfg.DefaultModel != "" {
		t.Errorf("Load() without default_model should return empty string, got %q", cfg.DefaultModel)
	}
}

// =============================================================================
// Tests for DaemonConfig
// =============================================================================

func TestDaemonPollInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval *int
		expected int
	}{
		{
			name:     "nil defaults to 60",
			interval: nil,
			expected: 60,
		},
		{
			name:     "explicit value",
			interval: intPtr(30),
			expected: 30,
		},
		{
			name:     "explicit zero",
			interval: intPtr(0),
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Daemon: DaemonConfig{
					PollInterval: tt.interval,
				},
			}
			if got := cfg.DaemonPollInterval(); got != tt.expected {
				t.Errorf("DaemonPollInterval() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDaemonMaxAgents(t *testing.T) {
	tests := []struct {
		name      string
		maxAgents *int
		expected  int
	}{
		{
			name:      "nil defaults to 5",
			maxAgents: nil,
			expected:  5,
		},
		{
			name:      "explicit value",
			maxAgents: intPtr(5),
			expected:  5,
		},
		{
			name:      "explicit zero",
			maxAgents: intPtr(0),
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Daemon: DaemonConfig{
					MaxAgents: tt.maxAgents,
				},
			}
			if got := cfg.DaemonMaxAgents(); got != tt.expected {
				t.Errorf("DaemonMaxAgents() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDaemonLabel(t *testing.T) {
	tests := []struct {
		name     string
		label    string
		expected string
	}{
		{
			name:     "empty defaults to triage:ready",
			label:    "",
			expected: "triage:ready",
		},
		{
			name:     "explicit value",
			label:    "custom:label",
			expected: "custom:label",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Daemon: DaemonConfig{
					Label: tt.label,
				},
			}
			if got := cfg.DaemonLabel(); got != tt.expected {
				t.Errorf("DaemonLabel() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDaemonVerbose(t *testing.T) {
	tests := []struct {
		name     string
		verbose  *bool
		expected bool
	}{
		{
			name:     "nil defaults to true",
			verbose:  nil,
			expected: true,
		},
		{
			name:     "explicit true",
			verbose:  boolPtr(true),
			expected: true,
		},
		{
			name:     "explicit false",
			verbose:  boolPtr(false),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Daemon: DaemonConfig{
					Verbose: tt.verbose,
				},
			}
			if got := cfg.DaemonVerbose(); got != tt.expected {
				t.Errorf("DaemonVerbose() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDaemonReflectIssues(t *testing.T) {
	tests := []struct {
		name          string
		reflectIssues *bool
		expected      bool
	}{
		{
			name:          "nil defaults to false",
			reflectIssues: nil,
			expected:      false, // THIS IS THE BUG-CAUSING FLAG - default should be false!
		},
		{
			name:          "explicit true",
			reflectIssues: boolPtr(true),
			expected:      true,
		},
		{
			name:          "explicit false",
			reflectIssues: boolPtr(false),
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Daemon: DaemonConfig{
					ReflectIssues: tt.reflectIssues,
				},
			}
			if got := cfg.DaemonReflectIssues(); got != tt.expected {
				t.Errorf("DaemonReflectIssues() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDaemonReflectOpen(t *testing.T) {
	tests := []struct {
		name        string
		reflectOpen *bool
		expected    bool
	}{
		{
			name:        "nil defaults to false",
			reflectOpen: nil,
			expected:    false,
		},
		{
			name:        "explicit true",
			reflectOpen: boolPtr(true),
			expected:    true,
		},
		{
			name:        "explicit false",
			reflectOpen: boolPtr(false),
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Daemon: DaemonConfig{
					ReflectOpen: tt.reflectOpen,
				},
			}
			if got := cfg.DaemonReflectOpen(); got != tt.expected {
				t.Errorf("DaemonReflectOpen() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDaemonWorkingDirectory(t *testing.T) {
	// Get actual home dir for testing
	home, _ := os.UserHomeDir()

	tests := []struct {
		name     string
		workDir  string
		expected string
	}{
		{
			name:     "empty defaults to ~/Documents/personal/orch-go",
			workDir:  "",
			expected: filepath.Join(home, "Documents", "personal", "orch-go"),
		},
		{
			name:     "tilde expansion",
			workDir:  "~/custom/path",
			expected: filepath.Join(home, "custom/path"),
		},
		{
			name:     "absolute path unchanged",
			workDir:  "/absolute/path",
			expected: "/absolute/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Daemon: DaemonConfig{
					WorkingDirectory: tt.workDir,
				},
			}
			if got := cfg.DaemonWorkingDirectory(); got != tt.expected {
				t.Errorf("DaemonWorkingDirectory() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDaemonPath(t *testing.T) {
	// Get actual home dir for testing
	home, _ := os.UserHomeDir()

	tests := []struct {
		name     string
		path     []string
		expected []string
	}{
		{
			name: "empty defaults to common paths",
			path: nil,
			expected: []string{
				filepath.Join(home, ".bun", "bin"),
				filepath.Join(home, "bin"),
				filepath.Join(home, "go", "bin"),
				"/opt/homebrew/bin",
				filepath.Join(home, ".local", "bin"),
			},
		},
		{
			name: "explicit paths with tilde expansion",
			path: []string{"~/.custom/bin", "/opt/bin"},
			expected: []string{
				filepath.Join(home, ".custom/bin"),
				"/opt/bin",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Daemon: DaemonConfig{
					Path: tt.path,
				},
			}
			got := cfg.DaemonPath()
			if len(got) != len(tt.expected) {
				t.Errorf("DaemonPath() returned %d paths, want %d", len(got), len(tt.expected))
				return
			}
			for i, path := range got {
				if path != tt.expected[i] {
					t.Errorf("DaemonPath()[%d] = %q, want %q", i, path, tt.expected[i])
				}
			}
		})
	}
}

func TestLoadDaemonConfig(t *testing.T) {
	// Save original home and restore after test
	originalHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create config directory and file with daemon settings
	configDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configContent := `backend: opencode
daemon:
  poll_interval: 30
  max_agents: 5
  label: "custom:ready"
  verbose: false
  reflect_issues: true
  reflect_open: true
  working_directory: ~/custom/dir
  path:
    - ~/.custom/bin
    - /opt/custom/bin
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.DaemonPollInterval() != 30 {
		t.Errorf("Load() DaemonPollInterval() = %d, want 30", cfg.DaemonPollInterval())
	}

	if cfg.DaemonMaxAgents() != 5 {
		t.Errorf("Load() DaemonMaxAgents() = %d, want 5", cfg.DaemonMaxAgents())
	}

	if cfg.DaemonLabel() != "custom:ready" {
		t.Errorf("Load() DaemonLabel() = %q, want %q", cfg.DaemonLabel(), "custom:ready")
	}

	if cfg.DaemonVerbose() {
		t.Error("Load() DaemonVerbose() = true, want false")
	}

	if !cfg.DaemonReflectIssues() {
		t.Error("Load() DaemonReflectIssues() = false, want true")
	}

	if !cfg.DaemonReflectOpen() {
		t.Error("Load() DaemonReflectOpen() = false, want true")
	}

	expectedWorkDir := filepath.Join(tmpDir, "custom/dir")
	if cfg.DaemonWorkingDirectory() != expectedWorkDir {
		t.Errorf("Load() DaemonWorkingDirectory() = %q, want %q", cfg.DaemonWorkingDirectory(), expectedWorkDir)
	}

	paths := cfg.DaemonPath()
	if len(paths) != 2 {
		t.Errorf("Load() DaemonPath() returned %d paths, want 2", len(paths))
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
	if cfg.DaemonPollInterval() != 60 {
		t.Errorf("Load() without daemon section should default poll_interval to 60, got %d", cfg.DaemonPollInterval())
	}

	if cfg.DaemonMaxAgents() != 5 {
		t.Errorf("Load() without daemon section should default max_agents to 5, got %d", cfg.DaemonMaxAgents())
	}

	if cfg.DaemonLabel() != "triage:ready" {
		t.Errorf("Load() without daemon section should default label to triage:ready, got %q", cfg.DaemonLabel())
	}

	if !cfg.DaemonVerbose() {
		t.Error("Load() without daemon section should default verbose to true")
	}

	if cfg.DaemonReflectIssues() {
		t.Error("Load() without daemon section should default reflect_issues to false")
	}

	if cfg.DaemonReflectOpen() {
		t.Error("Load() without daemon section should default reflect_open to false")
	}

	// Path should have defaults
	paths := cfg.DaemonPath()
	if len(paths) < 3 {
		t.Errorf("Load() without daemon section should have default paths, got %d", len(paths))
	}
}

// =============================================================================
// Tests for SessionConfig - Checkpoint Thresholds
// =============================================================================

func TestSessionCheckpointDefaults(t *testing.T) {
	cfg := &Config{}

	// Test orchestrator defaults
	if cfg.OrchestratorCheckpointWarning() != DefaultOrchestratorWarningMinutes {
		t.Errorf("OrchestratorCheckpointWarning() = %d, want %d", cfg.OrchestratorCheckpointWarning(), DefaultOrchestratorWarningMinutes)
	}
	if cfg.OrchestratorCheckpointStrong() != DefaultOrchestratorStrongMinutes {
		t.Errorf("OrchestratorCheckpointStrong() = %d, want %d", cfg.OrchestratorCheckpointStrong(), DefaultOrchestratorStrongMinutes)
	}
	if cfg.OrchestratorCheckpointMax() != DefaultOrchestratorMaxMinutes {
		t.Errorf("OrchestratorCheckpointMax() = %d, want %d", cfg.OrchestratorCheckpointMax(), DefaultOrchestratorMaxMinutes)
	}

	// Test agent defaults
	if cfg.AgentCheckpointWarning() != DefaultAgentWarningMinutes {
		t.Errorf("AgentCheckpointWarning() = %d, want %d", cfg.AgentCheckpointWarning(), DefaultAgentWarningMinutes)
	}
	if cfg.AgentCheckpointStrong() != DefaultAgentStrongMinutes {
		t.Errorf("AgentCheckpointStrong() = %d, want %d", cfg.AgentCheckpointStrong(), DefaultAgentStrongMinutes)
	}
	if cfg.AgentCheckpointMax() != DefaultAgentMaxMinutes {
		t.Errorf("AgentCheckpointMax() = %d, want %d", cfg.AgentCheckpointMax(), DefaultAgentMaxMinutes)
	}

	// Verify orchestrator thresholds are longer than agent thresholds
	if cfg.OrchestratorCheckpointWarning() <= cfg.AgentCheckpointWarning() {
		t.Errorf("Orchestrator warning (%d) should be > agent warning (%d)",
			cfg.OrchestratorCheckpointWarning(), cfg.AgentCheckpointWarning())
	}
	if cfg.OrchestratorCheckpointStrong() <= cfg.AgentCheckpointStrong() {
		t.Errorf("Orchestrator strong (%d) should be > agent strong (%d)",
			cfg.OrchestratorCheckpointStrong(), cfg.AgentCheckpointStrong())
	}
	if cfg.OrchestratorCheckpointMax() <= cfg.AgentCheckpointMax() {
		t.Errorf("Orchestrator max (%d) should be > agent max (%d)",
			cfg.OrchestratorCheckpointMax(), cfg.AgentCheckpointMax())
	}
}

func TestSessionCheckpointCustomValues(t *testing.T) {
	cfg := &Config{
		Session: SessionConfig{
			OrchestratorCheckpoints: &CheckpointThresholds{
				WarningMinutes: intPtr(300), // 5h
				StrongMinutes:  intPtr(420), // 7h
				MaxMinutes:     intPtr(540), // 9h
			},
			AgentCheckpoints: &CheckpointThresholds{
				WarningMinutes: intPtr(90),  // 1.5h
				StrongMinutes:  intPtr(150), // 2.5h
				MaxMinutes:     intPtr(210), // 3.5h
			},
		},
	}

	// Test custom orchestrator values
	if cfg.OrchestratorCheckpointWarning() != 300 {
		t.Errorf("OrchestratorCheckpointWarning() = %d, want 300", cfg.OrchestratorCheckpointWarning())
	}
	if cfg.OrchestratorCheckpointStrong() != 420 {
		t.Errorf("OrchestratorCheckpointStrong() = %d, want 420", cfg.OrchestratorCheckpointStrong())
	}
	if cfg.OrchestratorCheckpointMax() != 540 {
		t.Errorf("OrchestratorCheckpointMax() = %d, want 540", cfg.OrchestratorCheckpointMax())
	}

	// Test custom agent values
	if cfg.AgentCheckpointWarning() != 90 {
		t.Errorf("AgentCheckpointWarning() = %d, want 90", cfg.AgentCheckpointWarning())
	}
	if cfg.AgentCheckpointStrong() != 150 {
		t.Errorf("AgentCheckpointStrong() = %d, want 150", cfg.AgentCheckpointStrong())
	}
	if cfg.AgentCheckpointMax() != 210 {
		t.Errorf("AgentCheckpointMax() = %d, want 210", cfg.AgentCheckpointMax())
	}
}

func TestLoadSessionConfig(t *testing.T) {
	// Save original home and restore after test
	originalHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create config directory and file with session settings
	configDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configContent := `backend: opencode
session:
  orchestrator_checkpoints:
    warning_minutes: 300
    strong_minutes: 420
    max_minutes: 540
  agent_checkpoints:
    warning_minutes: 90
    strong_minutes: 150
    max_minutes: 210
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	// Test loaded orchestrator values
	if cfg.OrchestratorCheckpointWarning() != 300 {
		t.Errorf("Load() OrchestratorCheckpointWarning() = %d, want 300", cfg.OrchestratorCheckpointWarning())
	}
	if cfg.OrchestratorCheckpointStrong() != 420 {
		t.Errorf("Load() OrchestratorCheckpointStrong() = %d, want 420", cfg.OrchestratorCheckpointStrong())
	}
	if cfg.OrchestratorCheckpointMax() != 540 {
		t.Errorf("Load() OrchestratorCheckpointMax() = %d, want 540", cfg.OrchestratorCheckpointMax())
	}

	// Test loaded agent values
	if cfg.AgentCheckpointWarning() != 90 {
		t.Errorf("Load() AgentCheckpointWarning() = %d, want 90", cfg.AgentCheckpointWarning())
	}
	if cfg.AgentCheckpointStrong() != 150 {
		t.Errorf("Load() AgentCheckpointStrong() = %d, want 150", cfg.AgentCheckpointStrong())
	}
	if cfg.AgentCheckpointMax() != 210 {
		t.Errorf("Load() AgentCheckpointMax() = %d, want 210", cfg.AgentCheckpointMax())
	}
}

func TestLoadMissingSessionSection(t *testing.T) {
	// Save original home and restore after test
	originalHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create config without session section
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
	if cfg.OrchestratorCheckpointWarning() != DefaultOrchestratorWarningMinutes {
		t.Errorf("Load() without session section should default orchestrator warning to %d, got %d",
			DefaultOrchestratorWarningMinutes, cfg.OrchestratorCheckpointWarning())
	}
	if cfg.AgentCheckpointWarning() != DefaultAgentWarningMinutes {
		t.Errorf("Load() without session section should default agent warning to %d, got %d",
			DefaultAgentWarningMinutes, cfg.AgentCheckpointWarning())
	}
}
