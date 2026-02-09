// Package config provides project configuration management for orch-go.
package config

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create temp directory with config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".orch", "config.yaml")

	// Ensure .orch directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	// Write sample config
	content := `servers:
  web: 5173
  api: 3000
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Load config
	var (
		cfg *Config
		err error
	)
	_ = captureStderr(t, func() {
		cfg, err = Load(tmpDir)
	})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	// Verify servers
	if len(cfg.Servers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(cfg.Servers))
	}

	if cfg.Servers["web"] != 5173 {
		t.Errorf("Expected web port 5173, got %d", cfg.Servers["web"])
	}

	if cfg.Servers["api"] != 3000 {
		t.Errorf("Expected api port 3000, got %d", cfg.Servers["api"])
	}
}

func TestLoadConfigMissing(t *testing.T) {
	tmpDir := t.TempDir()

	// Loading missing config should return error
	_, err := Load(tmpDir)
	if err == nil {
		t.Error("Load() should return error for missing config")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".orch", "config.yaml")

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	// Write invalid YAML
	content := `servers:
  web: 5173
    invalid indentation
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	_, err := Load(tmpDir)
	if err == nil {
		t.Error("Load() should return error for invalid YAML")
	}
}

func TestGetServerPort(t *testing.T) {
	cfg := &Config{
		Servers: map[string]int{
			"web": 5173,
			"api": 3000,
		},
	}

	// Existing service
	port, ok := cfg.GetServerPort("web")
	if !ok {
		t.Error("GetServerPort('web') should return true")
	}
	if port != 5173 {
		t.Errorf("GetServerPort('web') = %d, want 5173", port)
	}

	// Non-existent service
	port, ok = cfg.GetServerPort("nonexistent")
	if ok {
		t.Error("GetServerPort('nonexistent') should return false")
	}
	if port != 0 {
		t.Errorf("GetServerPort('nonexistent') should return 0, got %d", port)
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		Servers: map[string]int{
			"web": 5173,
			"api": 3000,
		},
	}

	// Save config
	if err := Save(tmpDir, cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, ".orch", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Save() did not create config file")
	}

	// Load it back
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() after Save() failed: %v", err)
	}

	if len(loaded.Servers) != 2 {
		t.Errorf("Loaded config has %d servers, want 2", len(loaded.Servers))
	}

	if loaded.Servers["web"] != 5173 {
		t.Errorf("Loaded web port = %d, want 5173", loaded.Servers["web"])
	}
}

func TestDefaultPath(t *testing.T) {
	projectDir := "/tmp/myproject"
	expected := "/tmp/myproject/.orch/config.yaml"

	path := DefaultPath(projectDir)
	if path != expected {
		t.Errorf("DefaultPath(%s) = %s, want %s", projectDir, path, expected)
	}
}

func TestPolicyGetterDefaults(t *testing.T) {
	cfg := &Config{}

	if got := cfg.DaemonCleanupIntervalMinutes(); got != DefaultDaemonCleanupIntervalMinutes {
		t.Errorf("DaemonCleanupIntervalMinutes() = %d, want %d", got, DefaultDaemonCleanupIntervalMinutes)
	}
	if got := cfg.DaemonDeadSessionIntervalMinutes(); got != DefaultDaemonDeadSessionIntervalMinutes {
		t.Errorf("DaemonDeadSessionIntervalMinutes() = %d, want %d", got, DefaultDaemonDeadSessionIntervalMinutes)
	}
	if got := cfg.DashboardAgentsDeadMinutes(); got != DefaultDashboardAgentsDeadMinutes {
		t.Errorf("DashboardAgentsDeadMinutes() = %d, want %d", got, DefaultDashboardAgentsDeadMinutes)
	}
	if got := cfg.SpawnContextQualityThreshold(); got != DefaultSpawnContextQualityThreshold {
		t.Errorf("SpawnContextQualityThreshold() = %d, want %d", got, DefaultSpawnContextQualityThreshold)
	}
	if got := cfg.CompletionAutoRebuildTimeoutSeconds(); got != DefaultCompletionAutoRebuildTimeoutSeconds {
		t.Errorf("CompletionAutoRebuildTimeoutSeconds() = %d, want %d", got, DefaultCompletionAutoRebuildTimeoutSeconds)
	}
	if got := cfg.CompletionTranscriptExportTimeoutSeconds(); got != DefaultCompletionTranscriptExportTimeoutSeconds {
		t.Errorf("CompletionTranscriptExportTimeoutSeconds() = %d, want %d", got, DefaultCompletionTranscriptExportTimeoutSeconds)
	}
	if got := cfg.CompletionCacheInvalidateTimeoutSeconds(); got != DefaultCompletionCacheInvalidateTimeoutSeconds {
		t.Errorf("CompletionCacheInvalidateTimeoutSeconds() = %d, want %d", got, DefaultCompletionCacheInvalidateTimeoutSeconds)
	}
}

func TestPolicyGettersFromConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".orch", "config.yaml")

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	content := `daemon:
  cleanup:
    interval_minutes: 240
    sessions_age_days: 10
    workspaces_age_days: 12
  dead_session:
    interval_minutes: 7
    max_retries: 4
  orphan_reap:
    interval_minutes: 3
  dashboard_watchdog:
    interval_seconds: 45
    failures_before_restart: 3
    restart_cooldown_minutes: 8
dashboard:
  agents:
    active_minutes: 8
    ghost_display_hours: 6
    dead_minutes: 2
    stalled_minutes: 20
    beads_fetch_hours: 5
spawn:
  context_quality:
    threshold: 35
completion:
  auto_rebuild:
    timeout_seconds: 300
  transcript_export:
    timeout_seconds: 25
  cache_invalidate:
    timeout_seconds: 9
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	var (
		cfg *Config
		err error
	)
	_ = captureStderr(t, func() {
		cfg, err = Load(tmpDir)
	})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if got := cfg.DaemonCleanupIntervalMinutes(); got != 240 {
		t.Errorf("DaemonCleanupIntervalMinutes() = %d, want 240", got)
	}
	if got := cfg.DaemonCleanupSessionsAgeDays(); got != 10 {
		t.Errorf("DaemonCleanupSessionsAgeDays() = %d, want 10", got)
	}
	if got := cfg.DaemonCleanupWorkspacesAgeDays(); got != 12 {
		t.Errorf("DaemonCleanupWorkspacesAgeDays() = %d, want 12", got)
	}
	if got := cfg.DaemonDeadSessionIntervalMinutes(); got != 7 {
		t.Errorf("DaemonDeadSessionIntervalMinutes() = %d, want 7", got)
	}
	if got := cfg.DaemonMaxDeadSessionRetries(); got != 4 {
		t.Errorf("DaemonMaxDeadSessionRetries() = %d, want 4", got)
	}
	if got := cfg.DaemonOrphanReapIntervalMinutes(); got != 3 {
		t.Errorf("DaemonOrphanReapIntervalMinutes() = %d, want 3", got)
	}
	if got := cfg.DaemonDashboardWatchdogIntervalSeconds(); got != 45 {
		t.Errorf("DaemonDashboardWatchdogIntervalSeconds() = %d, want 45", got)
	}
	if got := cfg.DaemonDashboardWatchdogFailuresBeforeRestart(); got != 3 {
		t.Errorf("DaemonDashboardWatchdogFailuresBeforeRestart() = %d, want 3", got)
	}
	if got := cfg.DaemonDashboardWatchdogRestartCooldownMinutes(); got != 8 {
		t.Errorf("DaemonDashboardWatchdogRestartCooldownMinutes() = %d, want 8", got)
	}
	if got := cfg.DashboardAgentsActiveMinutes(); got != 8 {
		t.Errorf("DashboardAgentsActiveMinutes() = %d, want 8", got)
	}
	if got := cfg.DashboardAgentsGhostDisplayHours(); got != 6 {
		t.Errorf("DashboardAgentsGhostDisplayHours() = %d, want 6", got)
	}
	if got := cfg.DashboardAgentsDeadMinutes(); got != 2 {
		t.Errorf("DashboardAgentsDeadMinutes() = %d, want 2", got)
	}
	if got := cfg.DashboardAgentsStalledMinutes(); got != 20 {
		t.Errorf("DashboardAgentsStalledMinutes() = %d, want 20", got)
	}
	if got := cfg.DashboardAgentsBeadsFetchHours(); got != 5 {
		t.Errorf("DashboardAgentsBeadsFetchHours() = %d, want 5", got)
	}
	if got := cfg.SpawnContextQualityThreshold(); got != 35 {
		t.Errorf("SpawnContextQualityThreshold() = %d, want 35", got)
	}
	if got := cfg.CompletionAutoRebuildTimeoutSeconds(); got != 300 {
		t.Errorf("CompletionAutoRebuildTimeoutSeconds() = %d, want 300", got)
	}
	if got := cfg.CompletionTranscriptExportTimeoutSeconds(); got != 25 {
		t.Errorf("CompletionTranscriptExportTimeoutSeconds() = %d, want 25", got)
	}
	if got := cfg.CompletionCacheInvalidateTimeoutSeconds(); got != 9 {
		t.Errorf("CompletionCacheInvalidateTimeoutSeconds() = %d, want 9", got)
	}
}

func TestLoadLegacyFlatKeysAutoMigratesAndWarns(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".orch", "config.yaml")

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	content := `spawn_mode: opencode
daemon_cleanup_interval_minutes: 240
dashboard_agents_dead_minutes: 2
spawn_context_quality_threshold: 33
completion_auto_rebuild_timeout_seconds: 300
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	var (
		cfg     *Config
		loadErr error
	)
	stderr := captureStderr(t, func() {
		cfg, loadErr = Load(tmpDir)
	})
	if loadErr != nil {
		t.Fatalf("Load() failed: %v", loadErr)
	}

	if got := cfg.DaemonCleanupIntervalMinutes(); got != 240 {
		t.Errorf("DaemonCleanupIntervalMinutes() = %d, want 240", got)
	}
	if got := cfg.DashboardAgentsDeadMinutes(); got != 2 {
		t.Errorf("DashboardAgentsDeadMinutes() = %d, want 2", got)
	}
	if got := cfg.SpawnContextQualityThreshold(); got != 33 {
		t.Errorf("SpawnContextQualityThreshold() = %d, want 33", got)
	}
	if got := cfg.CompletionAutoRebuildTimeoutSeconds(); got != 300 {
		t.Errorf("CompletionAutoRebuildTimeoutSeconds() = %d, want 300", got)
	}

	if !strings.Contains(stderr, "DEPRECATED: legacy flat config keys") {
		t.Errorf("expected deprecation warning, got stderr: %q", stderr)
	}
	if !strings.Contains(stderr, legacyFlatConfigMigrationGuidePath) {
		t.Errorf("expected migration guide path in warning, got stderr: %q", stderr)
	}

	updated, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read migrated config: %v", err)
	}
	updatedStr := string(updated)
	if strings.Contains(updatedStr, "daemon_cleanup_interval_minutes") {
		t.Errorf("legacy key should be removed after auto-migration: %s", updatedStr)
	}
	if !strings.Contains(updatedStr, "daemon:") || !strings.Contains(updatedStr, "cleanup:") {
		t.Errorf("expected nested daemon cleanup keys after migration: %s", updatedStr)
	}
}

func TestLoadLegacyFlatKeyUsesTypedValueWhenBothSet(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".orch", "config.yaml")

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	content := `daemon:
  cleanup:
    interval_minutes: 111
daemon_cleanup_interval_minutes: 222
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if got := cfg.DaemonCleanupIntervalMinutes(); got != 111 {
		t.Errorf("DaemonCleanupIntervalMinutes() = %d, want 111", got)
	}

	updated, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read migrated config: %v", err)
	}
	if strings.Contains(string(updated), "daemon_cleanup_interval_minutes") {
		t.Errorf("legacy key should be removed when typed key already exists: %s", string(updated))
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}

	os.Stderr = w
	defer func() {
		os.Stderr = oldStderr
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close stderr writer: %v", err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read stderr: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("failed to close stderr reader: %v", err)
	}

	return string(out)
}
