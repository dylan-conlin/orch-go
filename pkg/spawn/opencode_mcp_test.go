package spawn

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCodeMCPPresets(t *testing.T) {
	t.Run("playwright is NOT an MCP preset", func(t *testing.T) {
		// playwright-cli is a standalone CLI tool, not an MCP server.
		// It's handled via context injection, not opencode.json MCP config.
		_, ok := opencodeMCPPresets["playwright"]
		if ok {
			t.Error("playwright found in opencodeMCPPresets, want not found (playwright-cli is not MCP)")
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
	t.Run("playwright returns error (not an MCP preset)", func(t *testing.T) {
		dir := t.TempDir()
		err := EnsureOpenCodeMCP(dir, "playwright")
		if err == nil {
			t.Error("expected error for playwright (not MCP), got nil")
		}
	})

	t.Run("unknown preset returns error", func(t *testing.T) {
		dir := t.TempDir()
		err := EnsureOpenCodeMCP(dir, "nonexistent")
		if err == nil {
			t.Error("expected error for unknown preset, got nil")
		}
	})

	t.Run("merges into existing opencode.json preserving other keys", func(t *testing.T) {
		dir := t.TempDir()

		// Add a test preset temporarily
		opencodeMCPPresets["test-server"] = OpenCodeMCPServerConfig{
			Type:    "local",
			Command: []string{"test-cmd"},
			Enabled: true,
		}
		defer delete(opencodeMCPPresets, "test-server")

		// Write an existing opencode.json with other config
		existing := `{
  "$schema": "https://opencode.ai/config.json",
  "instructions": ["CLAUDE.md"]
}`
		if err := os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(existing), 0644); err != nil {
			t.Fatal(err)
		}

		if err := EnsureOpenCodeMCP(dir, "test-server"); err != nil {
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
		if _, ok := mcp["test-server"]; !ok {
			t.Error("test-server not added to mcp")
		}
	})
}
