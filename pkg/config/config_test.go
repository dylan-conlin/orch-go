// Package config provides project configuration management for orch-go.
package config

import (
	"os"
	"path/filepath"
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
	cfg, err := Load(tmpDir)
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

func TestLoadWithMeta(t *testing.T) {
	// Create temp directory with config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".orch", "config.yaml")

	// Ensure .orch directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	// Write sample config
	content := `spawn_mode: claude
claude:
  model: opus
opencode:
  server: http://127.0.0.1:4096
servers:
  web: 5173
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Load config with metadata
	_, meta, err := LoadWithMeta(tmpDir)
	if err != nil {
		t.Fatalf("LoadWithMeta() failed: %v", err)
	}

	if meta == nil {
		t.Fatal("LoadWithMeta() returned nil meta")
	}

	if !meta.Explicit["spawn_mode"] {
		t.Error("Expected spawn_mode to be explicit")
	}
	if !meta.Explicit["claude"] {
		t.Error("Expected claude to be explicit")
	}
	if !meta.Explicit["opencode"] {
		t.Error("Expected opencode to be explicit")
	}
	if !meta.Explicit["servers"] {
		t.Error("Expected servers to be explicit")
	}
	if !meta.ExplicitClaude["model"] {
		t.Error("Expected claude.model to be explicit")
	}
	if meta.ExplicitClaude["tmux_session"] {
		t.Error("Did not expect claude.tmux_session to be explicit")
	}
	if !meta.ExplicitOpenCode["server"] {
		t.Error("Expected opencode.server to be explicit")
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
