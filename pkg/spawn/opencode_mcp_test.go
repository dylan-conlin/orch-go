package spawn

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCodeMCPPresets(t *testing.T) {
	t.Run("playwright preset has correct format", func(t *testing.T) {
		preset, ok := opencodeMCPPresets["playwright"]
		if !ok {
			t.Fatal("playwright preset not found in opencodeMCPPresets")
		}
		if preset.Type != "local" {
			t.Errorf("type = %q, want %q", preset.Type, "local")
		}
		if !preset.Enabled {
			t.Error("enabled = false, want true")
		}
		if len(preset.Command) < 3 {
			t.Fatalf("command has %d elements, want at least 3", len(preset.Command))
		}
		if preset.Command[0] != "npx" {
			t.Errorf("command[0] = %q, want %q", preset.Command[0], "npx")
		}
	})

	t.Run("unknown preset returns false", func(t *testing.T) {
		_, ok := opencodeMCPPresets["nonexistent"]
		if ok {
			t.Error("nonexistent preset found, want not found")
		}
	})
}

func TestEnsureOpenCodeMCP(t *testing.T) {
	t.Run("creates opencode.json when missing", func(t *testing.T) {
		dir := t.TempDir()
		if err := EnsureOpenCodeMCP(dir, "playwright"); err != nil {
			t.Fatalf("EnsureOpenCodeMCP() error = %v", err)
		}

		// Read and verify the created file
		data, err := os.ReadFile(filepath.Join(dir, "opencode.json"))
		if err != nil {
			t.Fatalf("failed to read opencode.json: %v", err)
		}

		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err != nil {
			t.Fatalf("invalid JSON: %v\nContent: %s", err, data)
		}

		// Verify mcp key exists
		mcp, ok := config["mcp"].(map[string]interface{})
		if !ok {
			t.Fatalf("missing or invalid 'mcp' key in: %s", data)
		}

		// Verify playwright entry
		pw, ok := mcp["playwright"].(map[string]interface{})
		if !ok {
			t.Fatalf("missing or invalid 'playwright' in mcp: %s", data)
		}
		if pw["type"] != "local" {
			t.Errorf("playwright type = %v, want 'local'", pw["type"])
		}
		if pw["enabled"] != true {
			t.Errorf("playwright enabled = %v, want true", pw["enabled"])
		}
	})

	t.Run("merges into existing opencode.json preserving other keys", func(t *testing.T) {
		dir := t.TempDir()

		// Write an existing opencode.json with other config
		existing := `{
  "$schema": "https://opencode.ai/config.json",
  "instructions": ["CLAUDE.md"]
}`
		if err := os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(existing), 0644); err != nil {
			t.Fatal(err)
		}

		if err := EnsureOpenCodeMCP(dir, "playwright"); err != nil {
			t.Fatalf("EnsureOpenCodeMCP() error = %v", err)
		}

		data, err := os.ReadFile(filepath.Join(dir, "opencode.json"))
		if err != nil {
			t.Fatal(err)
		}

		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		// Verify existing keys preserved
		if config["$schema"] != "https://opencode.ai/config.json" {
			t.Errorf("$schema lost, got %v", config["$schema"])
		}
		instructions, ok := config["instructions"].([]interface{})
		if !ok || len(instructions) != 1 || instructions[0] != "CLAUDE.md" {
			t.Errorf("instructions lost, got %v", config["instructions"])
		}

		// Verify mcp added
		mcp, ok := config["mcp"].(map[string]interface{})
		if !ok {
			t.Fatalf("missing mcp key")
		}
		if _, ok := mcp["playwright"]; !ok {
			t.Error("playwright not added to mcp")
		}
	})

	t.Run("merges into existing mcp preserving other servers", func(t *testing.T) {
		dir := t.TempDir()

		// Write an existing opencode.json with another MCP server
		existing := `{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "some-other-server": {
      "type": "local",
      "command": ["some-cmd"],
      "enabled": true
    }
  }
}`
		if err := os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(existing), 0644); err != nil {
			t.Fatal(err)
		}

		if err := EnsureOpenCodeMCP(dir, "playwright"); err != nil {
			t.Fatalf("EnsureOpenCodeMCP() error = %v", err)
		}

		data, err := os.ReadFile(filepath.Join(dir, "opencode.json"))
		if err != nil {
			t.Fatal(err)
		}

		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		mcp := config["mcp"].(map[string]interface{})

		// Verify existing server preserved
		if _, ok := mcp["some-other-server"]; !ok {
			t.Error("existing MCP server 'some-other-server' was lost")
		}

		// Verify new server added
		if _, ok := mcp["playwright"]; !ok {
			t.Error("playwright not added")
		}
	})

	t.Run("no-op when preset already exists", func(t *testing.T) {
		dir := t.TempDir()

		// Write opencode.json that already has playwright
		existing := `{
  "mcp": {
    "playwright": {
      "type": "local",
      "command": ["npx", "-y", "@playwright/mcp@latest"],
      "enabled": true
    }
  }
}`
		if err := os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(existing), 0644); err != nil {
			t.Fatal(err)
		}

		if err := EnsureOpenCodeMCP(dir, "playwright"); err != nil {
			t.Fatalf("EnsureOpenCodeMCP() error = %v", err)
		}

		// Verify file is still valid
		data, err := os.ReadFile(filepath.Join(dir, "opencode.json"))
		if err != nil {
			t.Fatal(err)
		}

		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err != nil {
			t.Fatalf("invalid JSON after no-op: %v", err)
		}
	})

	t.Run("unknown preset returns error", func(t *testing.T) {
		dir := t.TempDir()
		err := EnsureOpenCodeMCP(dir, "nonexistent")
		if err == nil {
			t.Error("expected error for unknown preset, got nil")
		}
	})
}
