package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

// stubOpenCodeClient implements opencode.ClientInterface with configurable token responses.
type stubOpenCodeClient struct {
	opencode.ClientInterface
	tokenResponses []*opencode.TokenStats
	tokenIndex     int
	tokenErr       error
	createdSession *opencode.CreateSessionResponse
	createErr      error
}

func (s *stubOpenCodeClient) GetSessionTokens(sessionID string) (*opencode.TokenStats, error) {
	if s.tokenErr != nil {
		return nil, s.tokenErr
	}
	if s.tokenIndex >= len(s.tokenResponses) {
		return nil, nil
	}
	resp := s.tokenResponses[s.tokenIndex]
	s.tokenIndex++
	return resp, nil
}

func (s *stubOpenCodeClient) CreateSession(title, directory, model, variant string, isWorker bool) (*opencode.CreateSessionResponse, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	if s.createdSession != nil {
		return s.createdSession, nil
	}
	return &opencode.CreateSessionResponse{ID: "ses_test_123"}, nil
}

// fakeTime provides controllable time for tests.
type fakeTime struct {
	current time.Time
}

func (ft *fakeTime) Now() time.Time {
	return ft.current
}

func (ft *fakeTime) Advance(d time.Duration) {
	ft.current = ft.current.Add(d)
}

// TestHealthcheckPassWhenModelResponds verifies the happy path:
// agent spawns successfully, model responds with tokens.
func TestHealthcheckPassWhenModelResponds(t *testing.T) {
	// Create a real temporary git repo for worktree operations
	projectDir := setupTestGitRepo(t)

	ft := &fakeTime{current: time.Now()}
	sleepCount := 0

	client := &stubOpenCodeClient{
		tokenResponses: []*opencode.TokenStats{
			nil, // First poll: no tokens yet
			{TotalTokens: 150, InputTokens: 100, OutputTokens: 50}, // Second poll: tokens!
		},
	}

	deps := &HealthcheckDeps{
		Client:     client,
		ProjectDir: projectDir,
		ServerURL:  "http://127.0.0.1:4096",
		Now:        ft.Now,
		Sleep: func(d time.Duration) {
			sleepCount++
			ft.Advance(d)
		},
		SpawnHeadless: func(c opencode.ClientInterface, serverURL, title, prompt, model, variant, runtimeDir string) (string, error) {
			return "ses_healthcheck_test", nil
		},
		Verbose: false,
	}

	result := executeHealthcheck(context.Background(), deps, 90*time.Second, "sonnet")

	if !result.Pass {
		t.Fatalf("expected healthcheck to pass, got: %s (diagnostic: %s)", result.Message, result.Diagnostic)
	}
	if result.TokensObserved != 150 {
		t.Fatalf("expected 150 tokens observed, got %d", result.TokensObserved)
	}
	if result.SessionID != "ses_healthcheck_test" {
		t.Fatalf("expected session ID ses_healthcheck_test, got %s", result.SessionID)
	}
	if sleepCount != 1 {
		t.Fatalf("expected 1 sleep (first poll nil, second poll success), got %d", sleepCount)
	}
}

// TestHealthcheckFailWhenModelNotResponding verifies that 0 tokens after model timeout
// produces the correct failure message.
func TestHealthcheckFailWhenModelNotResponding(t *testing.T) {
	projectDir := setupTestGitRepo(t)

	ft := &fakeTime{current: time.Now()}

	client := &stubOpenCodeClient{
		// Always return nil tokens
		tokenResponses: make([]*opencode.TokenStats, 20),
	}

	deps := &HealthcheckDeps{
		Client:     client,
		ProjectDir: projectDir,
		ServerURL:  "http://127.0.0.1:4096",
		Now:        ft.Now,
		Sleep: func(d time.Duration) {
			ft.Advance(d)
		},
		SpawnHeadless: func(c opencode.ClientInterface, serverURL, title, prompt, model, variant, runtimeDir string) (string, error) {
			return "ses_timeout_test", nil
		},
		Verbose: false,
	}

	result := executeHealthcheck(context.Background(), deps, 90*time.Second, "sonnet")

	if result.Pass {
		t.Fatalf("expected healthcheck to fail when model not responding")
	}
	if result.Message != "FAIL: model not responding (check model config)" {
		t.Fatalf("unexpected failure message: %s", result.Message)
	}
}

// TestHealthcheckFailWhenSpawnFails verifies that spawn failures are reported correctly.
func TestHealthcheckFailWhenSpawnFails(t *testing.T) {
	projectDir := setupTestGitRepo(t)

	ft := &fakeTime{current: time.Now()}

	deps := &HealthcheckDeps{
		Client:     &stubOpenCodeClient{},
		ProjectDir: projectDir,
		ServerURL:  "http://127.0.0.1:4096",
		Now:        ft.Now,
		Sleep:      func(d time.Duration) { ft.Advance(d) },
		SpawnHeadless: func(c opencode.ClientInterface, serverURL, title, prompt, model, variant, runtimeDir string) (string, error) {
			return "", fmt.Errorf("connection refused")
		},
		Verbose: false,
	}

	result := executeHealthcheck(context.Background(), deps, 90*time.Second, "sonnet")

	if result.Pass {
		t.Fatalf("expected healthcheck to fail when spawn fails")
	}
	if result.Message != "Failed to spawn agent" {
		t.Fatalf("unexpected failure message: %s", result.Message)
	}
	if result.Diagnostic != "connection refused" {
		t.Fatalf("unexpected diagnostic: %s", result.Diagnostic)
	}
}

// TestHealthcheckCancellation verifies that context cancellation is respected.
func TestHealthcheckCancellation(t *testing.T) {
	projectDir := setupTestGitRepo(t)

	ft := &fakeTime{current: time.Now()}

	ctx, cancel := context.WithCancel(context.Background())

	deps := &HealthcheckDeps{
		Client:     &stubOpenCodeClient{},
		ProjectDir: projectDir,
		ServerURL:  "http://127.0.0.1:4096",
		Now:        ft.Now,
		Sleep: func(d time.Duration) {
			ft.Advance(d)
			cancel() // Cancel after first sleep
		},
		SpawnHeadless: func(c opencode.ClientInterface, serverURL, title, prompt, model, variant, runtimeDir string) (string, error) {
			return "ses_cancel_test", nil
		},
		Verbose: false,
	}

	result := executeHealthcheck(ctx, deps, 90*time.Second, "sonnet")

	if result.Pass {
		t.Fatalf("expected healthcheck to fail on cancellation")
	}
	if result.Message != "Healthcheck cancelled" {
		t.Fatalf("unexpected message: %s", result.Message)
	}
}

// TestHealthcheckCleanupOnSuccess verifies that the worktree is cleaned up on success.
func TestHealthcheckCleanupOnSuccess(t *testing.T) {
	projectDir := setupTestGitRepo(t)

	ft := &fakeTime{current: time.Now()}

	client := &stubOpenCodeClient{
		tokenResponses: []*opencode.TokenStats{
			{TotalTokens: 100, InputTokens: 80, OutputTokens: 20},
		},
	}

	deps := &HealthcheckDeps{
		Client:     client,
		ProjectDir: projectDir,
		ServerURL:  "http://127.0.0.1:4096",
		Now:        ft.Now,
		Sleep:      func(d time.Duration) { ft.Advance(d) },
		SpawnHeadless: func(c opencode.ClientInterface, serverURL, title, prompt, model, variant, runtimeDir string) (string, error) {
			return "ses_cleanup_test", nil
		},
		Verbose: false,
	}

	result := executeHealthcheck(context.Background(), deps, 90*time.Second, "sonnet")

	if !result.Pass {
		t.Fatalf("expected healthcheck to pass, got: %s", result.Message)
	}

	// Verify worktree was cleaned up
	worktreesDir := filepath.Join(projectDir, ".orch", "worktrees")
	entries, err := os.ReadDir(worktreesDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to read worktrees dir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != ".gitkeep" {
			t.Fatalf("expected worktree to be cleaned up, but found: %s", entry.Name())
		}
	}
}

// TestHealthcheckLeaveWorkspaceOnFailure verifies that failed healthchecks leave the workspace for debugging.
func TestHealthcheckLeaveWorkspaceOnFailure(t *testing.T) {
	projectDir := setupTestGitRepo(t)

	ft := &fakeTime{current: time.Now()}

	deps := &HealthcheckDeps{
		Client:     &stubOpenCodeClient{},
		ProjectDir: projectDir,
		ServerURL:  "http://127.0.0.1:4096",
		Now:        ft.Now,
		Sleep:      func(d time.Duration) { ft.Advance(d) },
		SpawnHeadless: func(c opencode.ClientInterface, serverURL, title, prompt, model, variant, runtimeDir string) (string, error) {
			return "", fmt.Errorf("simulated failure")
		},
		Verbose: false,
	}

	result := executeHealthcheck(context.Background(), deps, 90*time.Second, "sonnet")

	if result.Pass {
		t.Fatalf("expected healthcheck to fail")
	}

	// Verify worktree still exists (left for debugging)
	worktreesDir := filepath.Join(projectDir, ".orch", "worktrees")
	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		t.Fatalf("failed to read worktrees dir: %v", err)
	}

	found := false
	for _, entry := range entries {
		if entry.IsDir() {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected worktree to be preserved on failure for debugging")
	}
}

// TestResolveModelForHealthcheck verifies model alias resolution.
func TestResolveModelForHealthcheck(t *testing.T) {
	tests := []struct {
		alias    string
		contains string
	}{
		{"sonnet", "anthropic/"},
		{"opus", "anthropic/"},
		{"haiku", "anthropic/"},
		{"flash", "google/"},
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			result := resolveModelForHealthcheck(tt.alias)
			if result == "" {
				t.Fatalf("expected non-empty model spec for alias %s", tt.alias)
			}
			if len(tt.contains) > 0 && !strings.Contains(result, tt.contains) {
				t.Fatalf("expected model spec %q to contain %q", result, tt.contains)
			}
		})
	}
}

// setupTestGitRepo creates a temporary git repository for test worktree operations.
func setupTestGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init", dir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v: %s", err, output)
	}

	// Configure git user (required for commits)
	cmd = exec.Command("git", "-C", dir, "config", "user.email", "test@test.com")
	cmd.Run()
	cmd = exec.Command("git", "-C", dir, "config", "user.name", "Test")
	cmd.Run()

	// Create initial commit (required for worktrees)
	readmePath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test repo\n"), 0644); err != nil {
		t.Fatalf("failed to create README: %v", err)
	}
	cmd = exec.Command("git", "-C", dir, "add", ".")
	cmd.Run()
	cmd = exec.Command("git", "-C", dir, "commit", "-m", "initial commit")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v: %s", err, output)
	}

	// Create .orch/worktrees directory
	worktreesDir := filepath.Join(dir, ".orch", "worktrees")
	if err := os.MkdirAll(worktreesDir, 0755); err != nil {
		t.Fatalf("failed to create worktrees dir: %v", err)
	}

	return dir
}
