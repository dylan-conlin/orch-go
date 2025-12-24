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
}

func TestBuildOpencodeAttachCommand(t *testing.T) {
	cfg := &OpencodeAttachConfig{
		ServerURL:  "http://127.0.0.1:4096",
		ProjectDir: "/home/user/project",
		Model:      "anthropic/claude-opus",
		SessionID:  "ses_123",
	}

	got := BuildOpencodeAttachCommand(cfg)
	wantParts := []string{
		"attach",
		"http://127.0.0.1:4096",
		"--dir",
		"/home/user/project",
		"--model",
		"anthropic/claude-opus",
		"--session",
		"ses_123",
	}

	for _, part := range wantParts {
		if !strings.Contains(got, part) {
			t.Errorf("BuildOpencodeAttachCommand() = %q, want to contain %q", got, part)
		}
	}
}

func TestBuildStandaloneCommand(t *testing.T) {

	tests := []struct {
		name      string
		cfg       *StandaloneConfig
		wantParts []string
		dontWant  []string
	}{
		{
			name: "basic standalone command",
			cfg: &StandaloneConfig{
				ProjectDir: "/test/project",
				Model:      "anthropic/claude-sonnet-4-20250514",
			},
			wantParts: []string{"opencode", "/test/project", "--model", "anthropic/claude-sonnet-4-20250514"},
			dontWant:  []string{"run", "--prompt", "--title", "--attach"},
		},
		{
			name: "project dir with spaces",
			cfg: &StandaloneConfig{
				ProjectDir: "/my project/with spaces",
				Model:      "anthropic/claude-opus-4-5-20251101",
			},
			wantParts: []string{"opencode", "my project", "--model"},
			dontWant:  []string{"run"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildStandaloneCommand(tt.cfg)

			// Check expected parts are present
			for _, part := range tt.wantParts {
				if !strings.Contains(result, part) {
					t.Errorf("BuildStandaloneCommand() = %q, want to contain %q", result, part)
				}
			}

			// Check unwanted parts are absent
			for _, part := range tt.dontWant {
				if strings.Contains(result, part) {
					t.Errorf("BuildStandaloneCommand() = %q, should NOT contain %q", result, part)
				}
			}
		})
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
		wantArgs     []string
	}{
		{
			name:         "inside tmux",
			windowTarget: "session:1",
			insideTmux:   true,
			wantArgs:     []string{"tmux", "switch-client", "-t", "session:1"},
		},
		{
			name:         "outside tmux",
			windowTarget: "session:1",
			insideTmux:   false,
			wantArgs:     []string{"tmux", "attach-session", "-t", "session:1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := BuildAttachCommand(tt.windowTarget, tt.insideTmux)
			if cmd.Path == "" {
				t.Error("Expected command path to be set")
			}
			// Check args
			for i, arg := range tt.wantArgs {
				if i >= len(cmd.Args) {
					t.Errorf("Missing arg at index %d: want %s", i, arg)
					continue
				}
				if cmd.Args[i] != arg {
					t.Errorf("Arg at index %d = %q, want %q", i, cmd.Args[i], arg)
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

// TestBuildRunCommandEnv verifies ORCH_WORKER=1 is set in the command environment.
func TestBuildRunCommandEnv(t *testing.T) {
	cfg := &RunConfig{
		ProjectDir: "/test/project",
		Model:      "anthropic/claude-opus",
		Title:      "test-title",
		Prompt:     "test prompt",
	}

	cmd := BuildRunCommand(cfg)

	// Check that ORCH_WORKER=1 is in the environment
	hasOrchWorker := false
	for _, env := range cmd.Env {
		if env == "ORCH_WORKER=1" {
			hasOrchWorker = true
			break
		}
	}

	if !hasOrchWorker {
		t.Errorf("BuildRunCommand() should set ORCH_WORKER=1 in environment, got env: %v", cmd.Env)
	}
}

// TestBuildSpawnCommandEnv verifies ORCH_WORKER=1 is set in the command environment.
func TestBuildSpawnCommandEnv(t *testing.T) {
	cfg := &SpawnConfig{
		ServerURL:     "http://127.0.0.1:4096",
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

// TestBuildOpencodeAttachCommandEnv verifies ORCH_WORKER=1 is prefixed in the command string.
func TestBuildOpencodeAttachCommandEnv(t *testing.T) {
	cfg := &OpencodeAttachConfig{
		ServerURL:  "http://127.0.0.1:4096",
		ProjectDir: "/home/user/project",
		Model:      "anthropic/claude-opus",
	}

	cmd := BuildOpencodeAttachCommand(cfg)

	// Check that the command starts with ORCH_WORKER=1
	if !strings.HasPrefix(cmd, "ORCH_WORKER=1 ") {
		t.Errorf("BuildOpencodeAttachCommand() should start with 'ORCH_WORKER=1 ', got: %q", cmd)
	}
}

// TestBuildStandaloneCommandEnv verifies ORCH_WORKER=1 is prefixed in the command string.
func TestBuildStandaloneCommandEnv(t *testing.T) {
	cfg := &StandaloneConfig{
		ProjectDir: "/test/project",
		Model:      "anthropic/claude-opus",
	}

	cmd := BuildStandaloneCommand(cfg)

	// Check that the command starts with ORCH_WORKER=1
	if !strings.HasPrefix(cmd, "ORCH_WORKER=1 ") {
		t.Errorf("BuildStandaloneCommand() should start with 'ORCH_WORKER=1 ', got: %q", cmd)
	}
}
