package tmux

import (
	"os"
	"path/filepath"
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

func TestSessionNameConstants(t *testing.T) {
	// Verify session name constants are correct
	if OrchestratorSessionName != "orchestrator" {
		t.Errorf("OrchestratorSessionName = %q, want %q", OrchestratorSessionName, "orchestrator")
	}
	// Note: MetaOrchestratorSessionName exists for backwards compatibility but
	// both meta-orchestrators and orchestrators now spawn into OrchestratorSessionName
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
		ServerURL:     "http://localhost:4096",
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
}

func TestBuildOpencodeAttachCommand(t *testing.T) {
	cfg := &OpencodeAttachConfig{
		ServerURL:     "http://localhost:4096",
		ProjectDir:    "/home/user/project",
		SessionID:     "ses_123",
		ClaudeContext: "worker",
	}

	got := BuildOpencodeAttachCommand(cfg)
	wantParts := []string{
		"attach",
		"http://localhost:4096",
		"--dir",
		"/home/user/project",
		"--session",
		"ses_123",
		"CLAUDE_CONTEXT=worker",
	}

	for _, part := range wantParts {
		if !strings.Contains(got, part) {
			t.Errorf("BuildOpencodeAttachCommand() = %q, want to contain %q", got, part)
		}
	}

	// Verify --model is NOT included (opencode attach doesn't support it)
	if strings.Contains(got, "--model") {
		t.Errorf("BuildOpencodeAttachCommand() = %q, should NOT contain --model (unsupported by opencode attach)", got)
	}
}

func TestBuildOpencodeAttachCommandNoSession(t *testing.T) {
	cfg := &OpencodeAttachConfig{
		ServerURL:  "http://localhost:4096",
		ProjectDir: "/home/user/project",
	}

	got := BuildOpencodeAttachCommand(cfg)

	// Should not include --session when no session ID
	if strings.Contains(got, "--session") {
		t.Errorf("BuildOpencodeAttachCommand() = %q, should NOT contain --session when empty", got)
	}
}

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

func TestSelectWindow(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// Create a test session and window
	project := "orch-go-test-select"
	projectDir := "/tmp/orch-go-test-select"

	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	windowTarget, _, err := CreateWindow(sessionName, "test-select", projectDir)
	if err != nil {
		t.Fatalf("Could not create window: %v", err)
	}

	// Select window
	err = SelectWindow(windowTarget)
	if err != nil {
		t.Errorf("SelectWindow failed: %v", err)
	}
}

func TestKillSession(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// Create a test session
	project := "orch-go-test-kill-session"
	projectDir := "/tmp/orch-go-test-kill-session"

	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}

	// Kill session
	err = KillSession(sessionName)
	if err != nil {
		t.Errorf("KillSession failed: %v", err)
	}

	// Verify session no longer exists
	if SessionExists(sessionName) {
		t.Error("Expected session to not exist after kill")
	}
}

func TestListWorkersSessions(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// Create a test session
	project := "orch-go-test-list-sessions"
	projectDir := "/tmp/orch-go-test-list-sessions"

	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	// List workers sessions
	sessions, err := ListWorkersSessions()
	if err != nil {
		t.Fatalf("ListWorkersSessions failed: %v", err)
	}

	// Should contain our test session
	found := false
	for _, s := range sessions {
		if s == sessionName {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected to find session %s in %v", sessionName, sessions)
	}
}

func TestBuildAttachCommand(t *testing.T) {
	tests := []struct {
		name         string
		windowTarget string
		insideTmux   bool
		wantArgs     []string // args to check (excluding tmux binary path)
	}{
		{
			name:         "inside tmux",
			windowTarget: "session:1",
			insideTmux:   true,
			wantArgs:     []string{"switch-client", "-t", "session:1"},
		},
		{
			name:         "outside tmux",
			windowTarget: "session:1",
			insideTmux:   false,
			wantArgs:     []string{"attach-session", "-t", "session:1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := BuildAttachCommand(tt.windowTarget, tt.insideTmux)
			if err != nil {
				t.Fatalf("BuildAttachCommand() error = %v", err)
			}
			if cmd.Path == "" {
				t.Error("Expected command path to be set")
			}
			// Check that expected args are present (socket args may be prepended)
			args := cmd.Args
			for _, wantArg := range tt.wantArgs {
				found := false
				for _, gotArg := range args {
					if gotArg == wantArg {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected arg %q in %v", wantArg, args)
				}
			}
		})
	}
}

func TestAttach(t *testing.T) {
	// This test is mostly to ensure it doesn't crash and handles the TMUX env var
	// We can't easily test the actual attachment in a unit test
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// We'll just test that it doesn't error out when given a nonexistent target
	// (it will error because tmux will fail, but we want to see it try)
	err := Attach("nonexistent-session:0")
	if err == nil {
		t.Error("Expected error when attaching to nonexistent session")
	}
}

func TestWindowExistsByID(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// Test with a window ID that almost certainly doesn't exist
	exists := WindowExistsByID("@99999999")
	if exists {
		t.Error("Expected window @99999999 to not exist")
	}

	// Create a test session and window, then verify it exists
	project := "orch-go-test-window-exists"
	projectDir := "/tmp/orch-go-test-window-exists"

	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	_, windowID, err := CreateWindow(sessionName, "test-exists", projectDir)
	if err != nil {
		t.Fatalf("Could not create window: %v", err)
	}

	// Window should exist
	if !WindowExistsByID(windowID) {
		t.Errorf("Expected window %s to exist", windowID)
	}

	// Kill the window
	err = KillWindowByID(windowID)
	if err != nil {
		t.Fatalf("Could not kill window: %v", err)
	}

	// Window should no longer exist
	if WindowExistsByID(windowID) {
		t.Errorf("Expected window %s to not exist after kill", windowID)
	}
}

func TestFindWindowByWorkspaceName(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// Create a test session
	project := "orch-go-test-find-ws"
	projectDir := "/tmp/orch-go-test-find-ws"

	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	// Create a window with a workspace name pattern
	workspaceName := "og-feat-test-find-22dec"
	windowName := BuildWindowName(workspaceName, "feature-impl", "test-123")
	_, windowID, err := CreateWindow(sessionName, windowName, projectDir)
	if err != nil {
		t.Fatalf("Could not create window: %v", err)
	}

	// Find the window by workspace name
	found, err := FindWindowByWorkspaceName(sessionName, workspaceName)
	if err != nil {
		t.Fatalf("FindWindowByWorkspaceName failed: %v", err)
	}
	if found == nil {
		t.Errorf("Expected to find window with workspace name %q", workspaceName)
	}
	if found != nil && found.ID != windowID {
		t.Errorf("Found window ID %s doesn't match expected %s", found.ID, windowID)
	}

	// Try to find a non-existent workspace
	notFound, err := FindWindowByWorkspaceName(sessionName, "nonexistent-workspace")
	if err != nil {
		t.Fatalf("FindWindowByWorkspaceName failed: %v", err)
	}
	if notFound != nil {
		t.Errorf("Expected not to find nonexistent workspace, but got %+v", notFound)
	}
}

func TestFindWindowByWorkspaceNameAllSessions(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// Create a test session
	project := "orch-go-test-find-all"
	projectDir := "/tmp/orch-go-test-find-all"

	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	// Create a window with a workspace name pattern
	workspaceName := "og-inv-test-all-22dec"
	windowName := BuildWindowName(workspaceName, "investigation", "")
	_, windowID, err := CreateWindow(sessionName, windowName, projectDir)
	if err != nil {
		t.Fatalf("Could not create window: %v", err)
	}

	// Find the window across all sessions
	found, foundSession, err := FindWindowByWorkspaceNameAllSessions(workspaceName)
	if err != nil {
		t.Fatalf("FindWindowByWorkspaceNameAllSessions failed: %v", err)
	}
	if found == nil {
		t.Errorf("Expected to find window with workspace name %q across all sessions", workspaceName)
	}
	if found != nil && found.ID != windowID {
		t.Errorf("Found window ID %s doesn't match expected %s", found.ID, windowID)
	}
	if foundSession != sessionName {
		t.Errorf("Found session %s doesn't match expected %s", foundSession, sessionName)
	}

	// Try to find a non-existent workspace
	notFound, _, err := FindWindowByWorkspaceNameAllSessions("nonexistent-workspace-all")
	if err != nil {
		t.Fatalf("FindWindowByWorkspaceNameAllSessions failed: %v", err)
	}
	if notFound != nil {
		t.Errorf("Expected not to find nonexistent workspace, but got %+v", notFound)
	}
}

// TestBuildSpawnCommandEnv verifies ORCH_WORKER=1 is set in the command environment.
func TestBuildSpawnCommandEnv(t *testing.T) {
	cfg := &SpawnConfig{
		ServerURL:     "http://localhost:4096",
		Prompt:        "test prompt",
		Title:         "test-title",
		ProjectDir:    "/test/project",
		WorkspaceName: "og-inv-test-23dec",
	}

	cmd := BuildSpawnCommand(cfg)

	// Check that ORCH_WORKER=1 is in the environment
	hasOrchWorker := false
	for _, env := range cmd.Env {
		if env == "ORCH_WORKER=1" {
			hasOrchWorker = true
			break
		}
	}

	if !hasOrchWorker {
		t.Errorf("BuildSpawnCommand() should set ORCH_WORKER=1 in environment, got env: %v", cmd.Env)
	}
}

// TestBuildOpencodeAttachCommandEnv verifies ORCH_WORKER=1 and CLAUDE_CONTEXT are prefixed in the command string.
func TestBuildOpencodeAttachCommandEnv(t *testing.T) {
	cfg := &OpencodeAttachConfig{
		ServerURL:  "http://localhost:4096",
		ProjectDir: "/home/user/project",
	}

	cmd := BuildOpencodeAttachCommand(cfg)

	// Check that ORCH_WORKER=1 is set
	if !strings.Contains(cmd, "ORCH_WORKER=1") {
		t.Errorf("BuildOpencodeAttachCommand() should contain 'ORCH_WORKER=1', got: %q", cmd)
	}

	// Check that CLAUDE_CONTEXT defaults to worker when not set
	if !strings.Contains(cmd, "CLAUDE_CONTEXT=worker") {
		t.Errorf("BuildOpencodeAttachCommand() should default to 'CLAUDE_CONTEXT=worker', got: %q", cmd)
	}
}

// TestBuildOpencodeAttachCommandClaudeContext verifies CLAUDE_CONTEXT is set correctly for different roles.
func TestBuildOpencodeAttachCommandClaudeContext(t *testing.T) {
	tests := []struct {
		name          string
		claudeContext string
		want          string
	}{
		{"worker", "worker", "CLAUDE_CONTEXT=worker"},
		{"orchestrator", "orchestrator", "CLAUDE_CONTEXT=orchestrator"},
		{"meta-orchestrator", "meta-orchestrator", "CLAUDE_CONTEXT=meta-orchestrator"},
		{"empty defaults to worker", "", "CLAUDE_CONTEXT=worker"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &OpencodeAttachConfig{
				ServerURL:     "http://localhost:4096",
				ProjectDir:    "/home/user/project",
				ClaudeContext: tt.claudeContext,
			}
			cmd := BuildOpencodeAttachCommand(cfg)
			if !strings.Contains(cmd, tt.want) {
				t.Errorf("BuildOpencodeAttachCommand() = %q, want to contain %q", cmd, tt.want)
			}
		})
	}
}

// TestGetTmuxCwd tests the GetTmuxCwd function returns the active window's cwd.
func TestGetTmuxCwd(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// Create a test session with multiple windows in different directories
	project := "orch-go-test-cwd"
	projectDir1 := "/tmp/orch-go-test-cwd-1"
	projectDir2 := "/tmp/orch-go-test-cwd-2"

	// Create test directories
	_ = os.MkdirAll(projectDir1, 0755)
	_ = os.MkdirAll(projectDir2, 0755)
	defer os.RemoveAll(projectDir1)
	defer os.RemoveAll(projectDir2)

	sessionName, err := EnsureWorkersSession(project, projectDir1)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	// The first window is created by EnsureWorkersSession, it will be in projectDir1
	// Create a second window in a different directory
	window2Target, _, err := CreateWindow(sessionName, "test-cwd-2", projectDir2)
	if err != nil {
		t.Fatalf("Could not create window 2: %v", err)
	}

	// Select window 2 (make it active)
	err = SelectWindow(window2Target)
	if err != nil {
		t.Fatalf("Could not select window 2: %v", err)
	}

	// GetTmuxCwd should now return projectDir2 (the active window's cwd)
	cwd, err := GetTmuxCwd(sessionName)
	if err != nil {
		t.Fatalf("GetTmuxCwd failed: %v", err)
	}

	// Resolve symlinks for comparison (macOS /tmp -> /private/tmp)
	expectedDir2, _ := evalSymlinks(projectDir2)

	// The cwd should be projectDir2 since that window is active
	if cwd != expectedDir2 && cwd != projectDir2 {
		t.Errorf("GetTmuxCwd() = %q, want %q (should return active window's cwd, not first window)", cwd, expectedDir2)
	}
}

// evalSymlinks resolves symlinks in a path, returning the original if resolution fails.
func evalSymlinks(path string) (string, error) {
	resolved, err := os.Readlink(path)
	if err != nil {
		// Not a symlink or can't resolve - try filepath.EvalSymlinks
		return evalSymlinksRecursive(path)
	}
	return resolved, nil
}

// evalSymlinksRecursive resolves all symlinks in a path.
func evalSymlinksRecursive(path string) (string, error) {
	// Use filepath.EvalSymlinks for full resolution
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path, err
	}
	return resolved, nil
}

// TestGetTmuxCwdNonExistentSession tests GetTmuxCwd handles non-existent session gracefully.
func TestGetTmuxCwdNonExistentSession(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	_, err := GetTmuxCwd("nonexistent-session-12345")
	if err == nil {
		t.Error("Expected error when getting cwd for non-existent session")
	}
}

// TestSendTextAndSubmit verifies that text is typed and Enter is submitted
// with a delay between them.
func TestSendTextAndSubmit(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	// Create a test session and window
	project := "orch-go-test-send-submit"
	projectDir := "/tmp/orch-go-test-send-submit"
	_ = os.MkdirAll(projectDir, 0755)
	defer os.RemoveAll(projectDir)

	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	windowTarget, _, err := CreateWindow(sessionName, "test-send-submit", projectDir)
	if err != nil {
		t.Fatalf("Could not create window: %v", err)
	}

	// Wait for shell to be ready
	time.Sleep(500 * time.Millisecond)

	// Send an echo command via SendTextAndSubmit
	testMsg := "echo SEND_SUBMIT_TEST_OK"
	err = SendTextAndSubmit(windowTarget, testMsg, DefaultSendDelay)
	if err != nil {
		t.Fatalf("SendTextAndSubmit failed: %v", err)
	}

	// Wait for command to execute
	time.Sleep(1 * time.Second)

	// Capture pane content and verify the command was submitted (output should contain the echo result)
	content, err := GetPaneContent(windowTarget)
	if err != nil {
		t.Fatalf("GetPaneContent failed: %v", err)
	}

	if !strings.Contains(content, "SEND_SUBMIT_TEST_OK") {
		t.Errorf("Expected pane to contain 'SEND_SUBMIT_TEST_OK' after SendTextAndSubmit, got:\n%s", content)
	}
}

// TestDefaultSendDelay verifies the constant is a reasonable value.
func TestDefaultSendDelay(t *testing.T) {
	if DefaultSendDelay < 100*time.Millisecond {
		t.Errorf("DefaultSendDelay = %v, too short (should be >= 100ms)", DefaultSendDelay)
	}
	if DefaultSendDelay > 2*time.Second {
		t.Errorf("DefaultSendDelay = %v, too long (should be <= 2s)", DefaultSendDelay)
	}
}

// TestDetectMainSocket tests socket detection for overmind environments.
func TestDetectMainSocket(t *testing.T) {
	tests := []struct {
		name     string
		tmuxEnv  string
		wantEmpty bool
	}{
		{
			name:      "not in tmux",
			tmuxEnv:   "",
			wantEmpty: true,
		},
		{
			name:      "in regular tmux",
			tmuxEnv:   "/tmp/tmux-501/default,12345,0",
			wantEmpty: true,
		},
		{
			name:      "in overmind tmux",
			tmuxEnv:   "/private/tmp/tmux-501/overmind-orch-go-abc123,55715,0",
			wantEmpty: false, // should detect main socket (if it exists on disk)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore TMUX env
			origTmux := os.Getenv("TMUX")
			defer func() {
				if origTmux != "" {
					os.Setenv("TMUX", origTmux)
				} else {
					os.Unsetenv("TMUX")
				}
			}()

			if tt.tmuxEnv == "" {
				os.Unsetenv("TMUX")
			} else {
				os.Setenv("TMUX", tt.tmuxEnv)
			}

			result := detectMainSocket()

			if tt.wantEmpty && result != "" {
				t.Errorf("detectMainSocket() = %q, want empty", result)
			}
			// For overmind case, result depends on whether /tmp/tmux-501/default exists
			// on this machine. We just verify the logic doesn't crash.
			if !tt.wantEmpty && tt.name == "in overmind tmux" {
				// The result will be empty if the main socket doesn't exist on disk,
				// which is fine - the important thing is it tried the right path
				t.Logf("detectMainSocket() = %q (empty is OK if socket doesn't exist on disk)", result)
			}
		})
	}
}

// TestGetCurrentWindowName tests getting the current tmux window name.
func TestGetCurrentWindowName(t *testing.T) {
	tests := []struct {
		name        string
		inTmux      bool
		expectedErr bool
		wantDefault bool
	}{
		{
			name:        "not in tmux",
			inTmux:      false,
			expectedErr: false,
			wantDefault: true,
		},
		{
			name:        "in tmux",
			inTmux:      true,
			expectedErr: false,
			wantDefault: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.inTmux && !IsAvailable() {
				t.Skip("tmux not available")
			}

			// Save original TMUX env
			originalTmux := os.Getenv("TMUX")
			defer func() {
				if originalTmux != "" {
					os.Setenv("TMUX", originalTmux)
				} else {
					os.Unsetenv("TMUX")
				}
			}()

			if !tt.inTmux {
				// Simulate not being in tmux
				os.Unsetenv("TMUX")
			} else {
				// For in-tmux test, we need an actual tmux session
				// Create a test session
				sessionName := "test-window-name-session"
				windowName := "test-window"

				cmd, err := tmuxCommand("new-session", "-d", "-s", sessionName, "-n", windowName)
				if err != nil {
					t.Fatalf("Failed to create tmux command: %v", err)
				}
				if err := cmd.Run(); err != nil {
					t.Skipf("Could not create test tmux session: %v", err)
				}
				defer func() {
					killCmd, _ := tmuxCommand("kill-session", "-t", sessionName)
					_ = killCmd.Run()
				}()

				// Note: We can't actually test GetCurrentWindowName from inside the test
				// because the test isn't running in that tmux session.
				// We'll just test the "not in tmux" case properly.
				t.Skip("Cannot test in-tmux case without running test inside tmux")
			}

			result, err := GetCurrentWindowName()

			if (err != nil) != tt.expectedErr {
				t.Errorf("GetCurrentWindowName() error = %v, wantErr %v", err, tt.expectedErr)
				return
			}

			if tt.wantDefault && result != "default" {
				t.Errorf("GetCurrentWindowName() = %q, want %q (not in tmux)", result, "default")
			}

			if !tt.wantDefault && result == "default" {
				t.Errorf("GetCurrentWindowName() = %q, should not be default when in tmux", result)
			}
		})
	}
}

func TestIdleShellCommands(t *testing.T) {
	// Verify all common shell names are in the idle list
	expectedIdle := []string{"zsh", "bash", "sh", "fish", "-zsh", "-bash", "-sh", "login"}
	for _, shell := range expectedIdle {
		if !idleShellCommands[shell] {
			t.Errorf("idleShellCommands missing %q", shell)
		}
	}

	// Verify agent processes are NOT idle
	expectedActive := []string{"claude", "opencode", "node", "python", "go"}
	for _, proc := range expectedActive {
		if idleShellCommands[proc] {
			t.Errorf("idleShellCommands should not contain %q", proc)
		}
	}
}

func TestHasChildProcesses(t *testing.T) {
	// PID 1 (launchd/init) always has children
	if !hasChildProcesses("1") {
		t.Error("PID 1 should have child processes")
	}

	// A very large PID should not exist
	if hasChildProcesses("999999999") {
		t.Error("PID 999999999 should not have child processes")
	}
}

// TestFindAllWindowsByBeadsID verifies that FindAllWindowsByBeadsID returns
// ALL matching windows, not just the first one.
func TestFindAllWindowsByBeadsID(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	project := "orch-go-test-find-all-beads"
	projectDir := "/tmp/orch-go-test-find-all-beads"

	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	beadsID := "test-beads-dup-123"

	// Create TWO windows with the same beads ID (simulating respawn/duplicate)
	windowName1 := BuildWindowName("og-feat-first-05mar", "feature-impl", beadsID)
	_, windowID1, err := CreateWindow(sessionName, windowName1, projectDir)
	if err != nil {
		t.Fatalf("Could not create window 1: %v", err)
	}

	windowName2 := BuildWindowName("og-feat-second-05mar", "feature-impl", beadsID)
	_, windowID2, err := CreateWindow(sessionName, windowName2, projectDir)
	if err != nil {
		t.Fatalf("Could not create window 2: %v", err)
	}

	// FindAllWindowsByBeadsID should return BOTH windows
	found, err := FindAllWindowsByBeadsID(sessionName, beadsID)
	if err != nil {
		t.Fatalf("FindAllWindowsByBeadsID failed: %v", err)
	}

	if len(found) != 2 {
		t.Errorf("FindAllWindowsByBeadsID returned %d windows, want 2", len(found))
	}

	// Verify both window IDs are present
	foundIDs := map[string]bool{}
	for _, w := range found {
		foundIDs[w.ID] = true
	}
	if !foundIDs[windowID1] {
		t.Errorf("FindAllWindowsByBeadsID missing window ID %s", windowID1)
	}
	if !foundIDs[windowID2] {
		t.Errorf("FindAllWindowsByBeadsID missing window ID %s", windowID2)
	}
}

// TestFindAllWindowsByBeadsIDAllSessions verifies that FindAllWindowsByBeadsIDAllSessions
// returns matching windows across multiple tmux sessions.
func TestFindAllWindowsByBeadsIDAllSessions(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	projectDir := "/tmp/orch-go-test-find-all-beads-sessions"

	session1, err := EnsureWorkersSession("orch-go-test-fab-1", projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session 1: %v", err)
	}
	defer func() { _ = KillSession(session1) }()

	session2, err := EnsureWorkersSession("orch-go-test-fab-2", projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session 2: %v", err)
	}
	defer func() { _ = KillSession(session2) }()

	beadsID := "test-beads-cross-sess-456"

	// Create one window in each session with the same beads ID
	windowName1 := BuildWindowName("og-feat-sess1-05mar", "feature-impl", beadsID)
	_, windowID1, err := CreateWindow(session1, windowName1, projectDir)
	if err != nil {
		t.Fatalf("Could not create window in session 1: %v", err)
	}

	windowName2 := BuildWindowName("og-feat-sess2-05mar", "feature-impl", beadsID)
	_, windowID2, err := CreateWindow(session2, windowName2, projectDir)
	if err != nil {
		t.Fatalf("Could not create window in session 2: %v", err)
	}

	// FindAllWindowsByBeadsIDAllSessions should return BOTH
	matches, err := FindAllWindowsByBeadsIDAllSessions(beadsID)
	if err != nil {
		t.Fatalf("FindAllWindowsByBeadsIDAllSessions failed: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("FindAllWindowsByBeadsIDAllSessions returned %d matches, want 2", len(matches))
	}

	// Verify both window IDs are present
	foundIDs := map[string]bool{}
	for _, m := range matches {
		foundIDs[m.Window.ID] = true
	}
	if !foundIDs[windowID1] {
		t.Errorf("FindAllWindowsByBeadsIDAllSessions missing window ID %s from session 1", windowID1)
	}
	if !foundIDs[windowID2] {
		t.Errorf("FindAllWindowsByBeadsIDAllSessions missing window ID %s from session 2", windowID2)
	}
}

// TestFindAllWindowsByWorkspaceName verifies that FindAllWindowsByWorkspaceName
// returns ALL matching windows in a session.
func TestFindAllWindowsByWorkspaceName(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	project := "orch-go-test-find-all-ws"
	projectDir := "/tmp/orch-go-test-find-all-ws"

	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	workspaceName := "og-feat-dupws-05mar"

	// Create two windows with the same workspace name (different beads IDs)
	windowName1 := BuildWindowName(workspaceName, "feature-impl", "beads-a")
	_, windowID1, err := CreateWindow(sessionName, windowName1, projectDir)
	if err != nil {
		t.Fatalf("Could not create window 1: %v", err)
	}

	windowName2 := BuildWindowName(workspaceName, "feature-impl", "beads-b")
	_, windowID2, err := CreateWindow(sessionName, windowName2, projectDir)
	if err != nil {
		t.Fatalf("Could not create window 2: %v", err)
	}

	found, err := FindAllWindowsByWorkspaceName(sessionName, workspaceName)
	if err != nil {
		t.Fatalf("FindAllWindowsByWorkspaceName failed: %v", err)
	}

	if len(found) != 2 {
		t.Errorf("FindAllWindowsByWorkspaceName returned %d windows, want 2", len(found))
	}

	foundIDs := map[string]bool{}
	for _, w := range found {
		foundIDs[w.ID] = true
	}
	if !foundIDs[windowID1] {
		t.Errorf("FindAllWindowsByWorkspaceName missing window ID %s", windowID1)
	}
	if !foundIDs[windowID2] {
		t.Errorf("FindAllWindowsByWorkspaceName missing window ID %s", windowID2)
	}
}

// TestFindAllWindowsByBeadsIDNoMatch verifies FindAllWindowsByBeadsID returns
// empty slice (not nil) when no windows match.
func TestFindAllWindowsByBeadsIDNoMatch(t *testing.T) {
	if !IsAvailable() {
		t.Skip("tmux not available")
	}

	project := "orch-go-test-find-all-none"
	projectDir := "/tmp/orch-go-test-find-all-none"

	sessionName, err := EnsureWorkersSession(project, projectDir)
	if err != nil {
		t.Skipf("Could not ensure workers session: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	found, err := FindAllWindowsByBeadsID(sessionName, "nonexistent-beads-id")
	if err != nil {
		t.Fatalf("FindAllWindowsByBeadsID failed: %v", err)
	}

	if len(found) != 0 {
		t.Errorf("FindAllWindowsByBeadsID returned %d windows for nonexistent ID, want 0", len(found))
	}
}

func TestIsPaneActive_ConservativeOnError(t *testing.T) {
	// IsPaneActive should return true (conservative) for invalid window IDs
	// to avoid false negatives that would miscount active agents.
	result := IsPaneActive("@999999")
	// On most systems, this window doesn't exist, so GetPaneCurrentCommand fails.
	// Conservative behavior: treat as active.
	if !result {
		t.Error("IsPaneActive should return true (conservative) for invalid window ID")
	}
}
