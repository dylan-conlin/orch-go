package tmux

import (
	"strings"
	"testing"
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
