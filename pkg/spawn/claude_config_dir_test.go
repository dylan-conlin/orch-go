package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpawnClaudeIncludesClaudeConfigDirInLaunchCommand(t *testing.T) {
	restore := stubTmuxLifecycle(t)
	defer restore()

	var launchCmd string
	sendTmuxKeys = func(windowTarget, keys string) error {
		launchCmd = keys
		return nil
	}

	cfg := &Config{
		Project:         "orch-go",
		ProjectDir:      "/tmp/orch-go",
		WorkspaceName:   "ws-test",
		SkillName:       "feature-impl",
		BeadsID:         "orch-go-123",
		ClaudeConfigDir: "/tmp/.claude-work",
	}

	if _, err := SpawnClaude(cfg); err != nil {
		t.Fatalf("SpawnClaude() error = %v", err)
	}

	if !strings.Contains(launchCmd, "CLAUDE_CONFIG_DIR=") {
		t.Fatalf("launch command missing CLAUDE_CONFIG_DIR: %q", launchCmd)
	}
	if !strings.Contains(launchCmd, `"/tmp/.claude-work"`) {
		t.Fatalf("launch command missing quoted CLAUDE_CONFIG_DIR value: %q", launchCmd)
	}
}

func TestSpawnClaudeOmitsClaudeConfigDirWhenUnset(t *testing.T) {
	restore := stubTmuxLifecycle(t)
	defer restore()

	var launchCmd string
	sendTmuxKeys = func(windowTarget, keys string) error {
		launchCmd = keys
		return nil
	}

	cfg := &Config{
		Project:       "orch-go",
		ProjectDir:    "/tmp/orch-go",
		WorkspaceName: "ws-test",
		SkillName:     "feature-impl",
		BeadsID:       "orch-go-123",
	}

	if _, err := SpawnClaude(cfg); err != nil {
		t.Fatalf("SpawnClaude() error = %v", err)
	}

	if strings.Contains(launchCmd, "CLAUDE_CONFIG_DIR=") {
		t.Fatalf("launch command should not include CLAUDE_CONFIG_DIR when unset: %q", launchCmd)
	}
}

func TestSpawnClaudeInlineSetsClaudeConfigDirEnv(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	envLog := filepath.Join(tmpDir, "claude-config-dir.log")
	claudeScript := filepath.Join(binDir, "claude")
	if err := os.WriteFile(claudeScript, []byte("#!/bin/sh\necho \"$CLAUDE_CONFIG_DIR\" > \"$ORCH_TEST_CLAUDE_CONFIG_LOG\"\ncat >/dev/null\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to write fake claude script: %v", err)
	}

	t.Setenv("ORCH_TEST_CLAUDE_CONFIG_LOG", envLog)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	projectDir := filepath.Join(tmpDir, "project")
	cfg := &Config{
		ProjectDir:      projectDir,
		WorkspaceName:   "ws-inline",
		ClaudeConfigDir: "/tmp/.claude-work",
	}

	if err := os.MkdirAll(cfg.WorkspacePath(), 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	if err := os.WriteFile(cfg.ContextFilePath(), []byte("context\n"), 0644); err != nil {
		t.Fatalf("failed to write context file: %v", err)
	}

	if err := SpawnClaudeInline(cfg); err != nil {
		t.Fatalf("SpawnClaudeInline() error = %v", err)
	}

	data, err := os.ReadFile(envLog)
	if err != nil {
		t.Fatalf("failed to read env log: %v", err)
	}
	if got := strings.TrimSpace(string(data)); got != "/tmp/.claude-work" {
		t.Fatalf("CLAUDE_CONFIG_DIR = %q, want %q", got, "/tmp/.claude-work")
	}
}
