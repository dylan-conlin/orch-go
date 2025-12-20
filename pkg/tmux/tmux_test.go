package tmux

import (
	"strings"
	"testing"
	"time"
)

func TestSessionExists(t *testing.T) {
	// This test requires tmux to be installed
	// Skip if tmux is not available
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// Test with a session name that almost certainly doesn't exist
	exists := SessionExists("orch-go-test-nonexistent-session-12345")
	if exists {
		t.Error("Expected session to not exist")
	}
}

func TestGetWorkersSessionName(t *testing.T) {
	tests := []struct {
		project  string
		expected string
	}{
		{"orch-go", "workers-orch-go"},
		{"beads", "workers-beads"},
		{"price-watch", "workers-price-watch"},
	}

	for _, tt := range tests {
		t.Run(tt.project, func(t *testing.T) {
			result := GetWorkersSessionName(tt.project)
			if result != tt.expected {
				t.Errorf("GetWorkersSessionName(%q) = %q, want %q", tt.project, result, tt.expected)
			}
		})
	}
}

func TestBuildWindowName(t *testing.T) {
	tests := []struct {
		workspace  string
		skillName  string
		beadsID    string
		wantPrefix string // We check prefix since emoji handling can vary
	}{
		{"og-inv-test-19dec", "investigation", "", "🔬"},
		{"og-feat-add-feature-19dec", "feature-impl", "", "🏗️"},
		{"og-debug-fix-bug-19dec", "systematic-debugging", "", "🐛"},
		{"og-arch-design-19dec", "architect", "", "📐"},
		{"og-work-task-19dec", "", "", "⚙️"}, // Default emoji
	}

	for _, tt := range tests {
		t.Run(tt.workspace, func(t *testing.T) {
			result := BuildWindowName(tt.workspace, tt.skillName, tt.beadsID)
			if !strings.HasPrefix(result, tt.wantPrefix) {
				t.Errorf("BuildWindowName(%q, %q, %q) = %q, want prefix %q",
					tt.workspace, tt.skillName, tt.beadsID, result, tt.wantPrefix)
			}
		})
	}
}

func TestBuildWindowNameWithBeadsID(t *testing.T) {
	result := BuildWindowName("og-inv-test-19dec", "investigation", "proj-123")
	// Should include beads ID
	if !strings.Contains(result, "proj-123") {
		t.Errorf("Expected window name to contain beads ID, got %q", result)
	}
}

func TestBuildSpawnCommand(t *testing.T) {
	cfg := &SpawnConfig{
		ServerURL:     "http://127.0.0.1:4096",
		Prompt:        "test prompt",
		Title:         "test-title",
		ProjectDir:    "/test/project",
		WorkspaceName: "og-inv-test-19dec",
	}

	cmd := BuildSpawnCommand(cfg)

	// Verify command structure
	if cmd.Path == "" {
		t.Error("Expected command path to be set")
	}

	// Check args include required flags
	args := strings.Join(cmd.Args, " ")
	if !strings.Contains(args, "--attach") {
		t.Error("Expected --attach flag")
	}
	// Note: --format json should NOT be included for tmux spawn
	// (tmux spawn should show TUI, not JSON output)
	if strings.Contains(args, "--format json") {
		t.Error("--format json should NOT be included for tmux spawn (TUI needed)")
	}
	if !strings.Contains(args, "--title") {
		t.Error("Expected --title flag")
	}
}

func TestSpawnResult(t *testing.T) {
	// Test SpawnResult structure
	result := SpawnResult{
		SessionID:     "ses_abc123",
		Window:        "workers-orch-go:5",
		WindowID:      "@1234",
		WindowName:    "🔬 og-inv-test-19dec",
		WorkspaceName: "og-inv-test-19dec",
	}

	if result.SessionID == "" {
		t.Error("Expected SessionID to be set")
	}
	if result.Window == "" {
		t.Error("Expected Window to be set")
	}
}

func TestBuildStandaloneCommand(t *testing.T) {
	cfg := &StandaloneConfig{
		ProjectDir: "/test/project",
		Model:      "anthropic/claude-sonnet-4-20250514",
	}

	cmd := BuildStandaloneCommand(cfg)

	// Verify command structure
	if cmd.Path == "" {
		t.Error("Expected command path to be set")
	}

	// Check args - should be: opencode {dir} --model {model}
	args := strings.Join(cmd.Args, " ")

	// Should include project dir
	if !strings.Contains(args, "/test/project") {
		t.Errorf("Expected project dir in args, got: %s", args)
	}

	// Should include --model flag
	if !strings.Contains(args, "--model") {
		t.Errorf("Expected --model flag, got: %s", args)
	}

	// Should include model value
	if !strings.Contains(args, "anthropic/claude-sonnet-4-20250514") {
		t.Errorf("Expected model value, got: %s", args)
	}

	// Should NOT include --attach (that's attach mode)
	if strings.Contains(args, "--attach") {
		t.Error("--attach should NOT be in standalone mode")
	}

	// Should NOT include run subcommand
	if strings.Contains(args, " run ") {
		t.Error("'run' subcommand should NOT be in standalone mode")
	}
}

func TestStandaloneConfigWithEnvVars(t *testing.T) {
	cfg := &StandaloneConfig{
		ProjectDir: "/test/project",
		Model:      "anthropic/claude-sonnet-4-20250514",
		EnvVars: map[string]string{
			"ORCH_WORKER": "true",
		},
	}

	// EnvVars should be set on the config
	if cfg.EnvVars["ORCH_WORKER"] != "true" {
		t.Error("Expected ORCH_WORKER env var to be set")
	}
}

func TestIsOpenCodeReady(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "empty content",
			content:  "",
			expected: false,
		},
		{
			name:     "prompt box only",
			content:  "┃ Some text",
			expected: false,
		},
		{
			name:     "prompt box with build text",
			content:  "┃ Some text\nBuild\nalt+x",
			expected: true,
		},
		{
			name:     "prompt box with agent text",
			content:  "┃ Type your message\nagent: claude",
			expected: true,
		},
		{
			name:     "prompt box with commands hint",
			content:  "┃ Input\ncommands\nalt+x",
			expected: true,
		},
		{
			name:     "no TUI indicators",
			content:  "Loading opencode...",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsOpenCodeReady(tt.content)
			if result != tt.expected {
				t.Errorf("IsOpenCodeReady(%q) = %v, want %v", tt.content, result, tt.expected)
			}
		})
	}
}

func TestWaitForOpenCodeReadyConfig(t *testing.T) {
	// Test that WaitConfig has correct default values
	cfg := DefaultWaitConfig()

	if cfg.Timeout <= 0 {
		t.Error("Expected positive timeout")
	}

	if cfg.PollInterval <= 0 {
		t.Error("Expected positive poll interval")
	}

	// Timeout should be reasonable (15s default from Python)
	if cfg.Timeout < 10*time.Second || cfg.Timeout > 30*time.Second {
		t.Errorf("Unexpected timeout: %v", cfg.Timeout)
	}

	// Poll interval should be short (200ms from Python)
	if cfg.PollInterval < 100*time.Millisecond || cfg.PollInterval > 500*time.Millisecond {
		t.Errorf("Unexpected poll interval: %v", cfg.PollInterval)
	}
}

func TestSendPromptAfterReadyConfig(t *testing.T) {
	// Test that SendPromptConfig has correct default values
	cfg := DefaultSendPromptConfig()

	// Post-ready delay should be 1 second (from Python)
	if cfg.PostReadyDelay < 500*time.Millisecond || cfg.PostReadyDelay > 2*time.Second {
		t.Errorf("Unexpected post-ready delay: %v", cfg.PostReadyDelay)
	}
}

// Integration test for WaitForOpenCodeReady - only run if tmux is available
func TestWaitForOpenCodeReady_Integration(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// Create a test session and window
	project := "orch-go-test-wait"
	projectDir := "/tmp/orch-go-test-wait"

	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	windowTarget, _, err := CreateWindow(sessionName, "test-wait", projectDir)
	if err != nil {
		t.Fatalf("Could not create window: %v", err)
	}

	// Window starts empty, so WaitForOpenCodeReady should timeout quickly with short timeout
	cfg := WaitConfig{
		Timeout:      100 * time.Millisecond,
		PollInterval: 50 * time.Millisecond,
	}

	err = WaitForOpenCodeReady(windowTarget, cfg)
	if err == nil {
		t.Error("Expected timeout error when TUI is not present")
	}

	// Error should indicate timeout
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// Integration test - only runs if tmux is available
func TestEnsureWorkersSession(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// Use a test session name to avoid interfering with real sessions
	project := "orch-go-test"
	projectDir := "/tmp/orch-go-test"

	// This should create or find the workers session
	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session (may need tmux server): %v", err)
	}

	expected := "workers-orch-go-test"
	if sessionName != expected {
		t.Errorf("EnsureWorkersSession returned %q, want %q", sessionName, expected)
	}

	// Clean up test session if we created it
	_ = KillSession(sessionName)
}
