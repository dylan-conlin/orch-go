package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/session"
)

func TestDiscoverSessionHandoff(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		setup       func(string) error
		expectError bool
		expectPath  string // relative to tmpDir
	}{
		{
			name: "finds handoff via symlink",
			setup: func(root string) error {
				// Create session directory structure
				sessionDir := filepath.Join(root, ".orch", "session", "2026-01-13-0830")
				if err := os.MkdirAll(sessionDir, 0755); err != nil {
					return err
				}

				// Create SESSION_HANDOFF.md
				handoffPath := filepath.Join(sessionDir, "SESSION_HANDOFF.md")
				if err := os.WriteFile(handoffPath, []byte("test content"), 0644); err != nil {
					return err
				}

				// Create latest symlink
				latestSymlink := filepath.Join(root, ".orch", "session", "latest")
				return os.Symlink("2026-01-13-0830", latestSymlink)
			},
			expectError: false,
			expectPath:  ".orch/session/2026-01-13-0830/SESSION_HANDOFF.md",
		},
		{
			name: "finds handoff via directory (no symlink)",
			setup: func(root string) error {
				// Create session directory structure
				sessionDir := filepath.Join(root, ".orch", "session", "latest")
				if err := os.MkdirAll(sessionDir, 0755); err != nil {
					return err
				}

				// Create SESSION_HANDOFF.md
				handoffPath := filepath.Join(sessionDir, "SESSION_HANDOFF.md")
				return os.WriteFile(handoffPath, []byte("test content"), 0644)
			},
			expectError: false,
			expectPath:  ".orch/session/latest/SESSION_HANDOFF.md",
		},
		{
			name: "walks up directory tree",
			setup: func(root string) error {
				// Create nested directory
				nestedDir := filepath.Join(root, "sub1", "sub2", "sub3")
				if err := os.MkdirAll(nestedDir, 0755); err != nil {
					return err
				}

				// Create session directory at root level
				sessionDir := filepath.Join(root, ".orch", "session", "2026-01-13-0900")
				if err := os.MkdirAll(sessionDir, 0755); err != nil {
					return err
				}

				// Create SESSION_HANDOFF.md
				handoffPath := filepath.Join(sessionDir, "SESSION_HANDOFF.md")
				if err := os.WriteFile(handoffPath, []byte("test content"), 0644); err != nil {
					return err
				}

				// Create latest symlink
				latestSymlink := filepath.Join(root, ".orch", "session", "latest")
				if err := os.Symlink("2026-01-13-0900", latestSymlink); err != nil {
					return err
				}

				// Change to nested directory for test
				return os.Chdir(nestedDir)
			},
			expectError: false,
			expectPath:  ".orch/session/2026-01-13-0900/SESSION_HANDOFF.md",
		},
		{
			name: "returns error when no handoff found",
			setup: func(root string) error {
				// Create directory but no session structure
				return nil
			},
			expectError: true,
		},
		{
			name: "returns error when symlink broken",
			setup: func(root string) error {
				// Create latest symlink pointing to non-existent directory
				sessionBase := filepath.Join(root, ".orch", "session")
				if err := os.MkdirAll(sessionBase, 0755); err != nil {
					return err
				}
				latestSymlink := filepath.Join(sessionBase, "latest")
				return os.Symlink("nonexistent", latestSymlink)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original working directory
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(originalDir)

			// Create test-specific subdirectory
			testDir := filepath.Join(tmpDir, tt.name)
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Change to test directory
			if err := os.Chdir(testDir); err != nil {
				t.Fatal(err)
			}

			// Run setup
			if err := tt.setup(testDir); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			// Test discovery
			got, err := discoverSessionHandoff()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify path (resolve symlinks for comparison to handle /var vs /private/var on macOS)
			expectedPath := filepath.Join(testDir, tt.expectPath)
			gotResolved, _ := filepath.EvalSymlinks(got)
			expectedResolved, _ := filepath.EvalSymlinks(expectedPath)
			if gotResolved != expectedResolved {
				t.Errorf("got path %q, want %q", gotResolved, expectedResolved)
			}

			// Verify file exists and is readable
			content, err := os.ReadFile(got)
			if err != nil {
				t.Errorf("failed to read discovered handoff: %v", err)
			}
			if len(content) == 0 {
				t.Error("handoff file is empty")
			}
		})
	}
}

func TestCreateSessionHandoffDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock session
	sess := &session.Session{
		Goal:      "Test goal",
		StartedAt: time.Now(),
	}

	// Test creating session handoff directory
	err := createSessionHandoffDirectory(tmpDir, sess)
	if err != nil {
		t.Fatalf("createSessionHandoffDirectory() failed: %v", err)
	}

	// Verify latest symlink exists
	latestSymlink := filepath.Join(tmpDir, ".orch", "session", "latest")
	stat, err := os.Lstat(latestSymlink)
	if err != nil {
		t.Fatalf("latest symlink not created: %v", err)
	}
	if stat.Mode()&os.ModeSymlink == 0 {
		t.Error("latest is not a symlink")
	}

	// Verify symlink target exists and has SESSION_HANDOFF.md
	target, err := os.Readlink(latestSymlink)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}

	handoffPath := filepath.Join(tmpDir, ".orch", "session", target, "SESSION_HANDOFF.md")
	content, err := os.ReadFile(handoffPath)
	if err != nil {
		t.Fatalf("SESSION_HANDOFF.md not created: %v", err)
	}

	// Verify content contains expected fields
	contentStr := string(content)
	if !contains(contentStr, "Test goal") {
		t.Error("handoff missing session goal")
	}
	if !contains(contentStr, "Session Handoff") {
		t.Error("handoff missing title")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && stringContains(s, substr)
}
