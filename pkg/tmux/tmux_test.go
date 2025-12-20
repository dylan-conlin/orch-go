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

func TestCaptureLines(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// Create a test session and window
	project := "orch-go-test-capture"
	projectDir := "/tmp/orch-go-test-capture"

	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	windowTarget, _, err := CreateWindow(sessionName, "test-capture", projectDir)
	if err != nil {
		t.Fatalf("Could not create window: %v", err)
	}

	// Send some content to the window
	_ = SendKeysLiteral(windowTarget, "echo 'line1'")
	_ = SendEnter(windowTarget)

	// Give tmux time to process
	time.Sleep(100 * time.Millisecond)

	// Capture with different line counts
	lines, err := CaptureLines(windowTarget, 10)
	if err != nil {
		t.Fatalf("CaptureLines failed: %v", err)
	}

	// Should have some content
	if len(lines) == 0 {
		t.Error("Expected some captured lines")
	}
}

func TestCaptureLinesDefault(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// Create a test session and window
	project := "orch-go-test-capture-default"
	projectDir := "/tmp/orch-go-test-capture-default"

	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	windowTarget, _, err := CreateWindow(sessionName, "test-capture-default", projectDir)
	if err != nil {
		t.Fatalf("Could not create window: %v", err)
	}

	// Capture with default (0 means all visible)
	lines, err := CaptureLines(windowTarget, 0)
	if err != nil {
		t.Fatalf("CaptureLines failed: %v", err)
	}

	// Should return some lines (pane is visible)
	// Even empty pane has some lines
	if lines == nil {
		t.Error("Expected non-nil result")
	}
}

func TestListWindows(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// Create a test session
	project := "orch-go-test-list-win"
	projectDir := "/tmp/orch-go-test-list-win"

	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	// Create a window with known name
	windowName := "🔬 og-test [test-abc123]"
	_, _, err = CreateWindow(sessionName, windowName, projectDir)
	if err != nil {
		t.Fatalf("Could not create window: %v", err)
	}

	// List windows
	windows, err := ListWindows(sessionName)
	if err != nil {
		t.Fatalf("ListWindows failed: %v", err)
	}

	// Should have at least 2 windows (servers + test window)
	if len(windows) < 2 {
		t.Errorf("Expected at least 2 windows, got %d", len(windows))
	}

	// Find our test window
	found := false
	for _, w := range windows {
		if strings.Contains(w.Name, "og-test") {
			found = true
			if w.Index == "" {
				t.Error("Window index should not be empty")
			}
			if w.Target == "" {
				t.Error("Window target should not be empty")
			}
		}
	}
	if !found {
		t.Error("Expected to find test window")
	}
}

func TestFindWindowByBeadsID(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// Create a test session
	project := "orch-go-test-find"
	projectDir := "/tmp/orch-go-test-find"

	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	// Create a window with beads ID in name
	beadsID := "find-xyz789"
	windowName := BuildWindowName("og-test-find", "investigation", beadsID)
	_, _, err = CreateWindow(sessionName, windowName, projectDir)
	if err != nil {
		t.Fatalf("Could not create window: %v", err)
	}

	// Find window by beads ID
	window, err := FindWindowByBeadsID(sessionName, beadsID)
	if err != nil {
		t.Fatalf("FindWindowByBeadsID failed: %v", err)
	}

	if window == nil {
		t.Fatal("Expected to find window")
	}

	if !strings.Contains(window.Name, beadsID) {
		t.Errorf("Window name %q should contain beads ID %q", window.Name, beadsID)
	}
}

func TestFindWindowByBeadsIDNotFound(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// Create a test session
	project := "orch-go-test-find-notfound"
	projectDir := "/tmp/orch-go-test-find-notfound"

	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	// Try to find window that doesn't exist
	window, err := FindWindowByBeadsID(sessionName, "nonexistent-abc123")
	if err != nil {
		t.Fatalf("FindWindowByBeadsID should not error: %v", err)
	}

	if window != nil {
		t.Error("Expected nil when window not found")
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

// TestKillWindow verifies KillWindow closes a tmux window.
func TestKillWindow(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// Create a test session and window
	project := "orch-go-test-kill"
	projectDir := "/tmp/orch-go-test-kill"

	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	// Create a window
	windowTarget, _, err := CreateWindow(sessionName, "test-kill-window", projectDir)
	if err != nil {
		t.Fatalf("Could not create window: %v", err)
	}

	// Verify window exists
	if !WindowExists(windowTarget) {
		t.Fatal("Expected window to exist after creation")
	}

	// Kill the window
	err = KillWindow(windowTarget)
	if err != nil {
		t.Fatalf("KillWindow failed: %v", err)
	}

	// Verify window no longer exists
	if WindowExists(windowTarget) {
		t.Error("Expected window to not exist after kill")
	}
}

// TestKillWindowByID verifies KillWindowByID closes a tmux window by ID.
func TestKillWindowByID(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// Create a test session and window
	project := "orch-go-test-kill-id"
	projectDir := "/tmp/orch-go-test-kill-id"

	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	// Create a window and capture its ID
	windowTarget, windowID, err := CreateWindow(sessionName, "test-kill-by-id", projectDir)
	if err != nil {
		t.Fatalf("Could not create window: %v", err)
	}

	// Verify window exists
	if !WindowExists(windowTarget) {
		t.Fatal("Expected window to exist after creation")
	}

	// Kill the window by ID
	err = KillWindowByID(windowID)
	if err != nil {
		t.Fatalf("KillWindowByID failed: %v", err)
	}

	// Verify window no longer exists
	if WindowExists(windowTarget) {
		t.Error("Expected window to not exist after kill by ID")
	}
}

// TestListWindowIDs verifies ListWindowIDs returns active window IDs.
func TestListWindowIDs(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// Create a test session
	project := "orch-go-test-list"
	projectDir := "/tmp/orch-go-test-list"

	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	// Create a few windows
	_, windowID1, err := CreateWindow(sessionName, "test-list-1", projectDir)
	if err != nil {
		t.Fatalf("Could not create window 1: %v", err)
	}

	_, windowID2, err := CreateWindow(sessionName, "test-list-2", projectDir)
	if err != nil {
		t.Fatalf("Could not create window 2: %v", err)
	}

	// List windows for the session
	ids, err := ListWindowIDs(sessionName)
	if err != nil {
		t.Fatalf("ListWindowIDs failed: %v", err)
	}

	// Should contain both window IDs
	hasID1 := false
	hasID2 := false
	for _, id := range ids {
		if id == windowID1 {
			hasID1 = true
		}
		if id == windowID2 {
			hasID2 = true
		}
	}

	if !hasID1 {
		t.Errorf("Expected window ID %s to be in list %v", windowID1, ids)
	}
	if !hasID2 {
		t.Errorf("Expected window ID %s to be in list %v", windowID2, ids)
	}
}
