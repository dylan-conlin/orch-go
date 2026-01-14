package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/tmux"
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

func TestArchiveActiveSessionHandoff(t *testing.T) {
	tmpDir := t.TempDir()

	// Get current window name to construct the window-scoped path
	windowName, err := tmux.GetCurrentWindowName()
	if err != nil {
		t.Fatalf("failed to get window name: %v", err)
	}

	// Create active directory with SESSION_HANDOFF.md
	activeDir := filepath.Join(tmpDir, ".orch", "session", windowName, "active")
	if err := os.MkdirAll(activeDir, 0755); err != nil {
		t.Fatalf("failed to create active directory: %v", err)
	}

	// Write a test handoff file
	handoffPath := filepath.Join(activeDir, "SESSION_HANDOFF.md")
	testContent := "Test session handoff content"
	if err := os.WriteFile(handoffPath, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write SESSION_HANDOFF.md: %v", err)
	}

	// Test archiving active directory
	err = archiveActiveSessionHandoff(tmpDir)
	if err != nil {
		t.Fatalf("archiveActiveSessionHandoff() failed: %v", err)
	}

	// Verify active directory was removed
	if _, err := os.Stat(activeDir); !os.IsNotExist(err) {
		t.Error("active directory still exists after archiving")
	}

	// Verify latest symlink exists
	latestSymlink := filepath.Join(tmpDir, ".orch", "session", windowName, "latest")
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

	archivedHandoffPath := filepath.Join(tmpDir, ".orch", "session", windowName, target, "SESSION_HANDOFF.md")
	content, err := os.ReadFile(archivedHandoffPath)
	if err != nil {
		t.Fatalf("SESSION_HANDOFF.md not found in archived directory: %v", err)
	}

	// Verify content matches original
	if string(content) != testContent {
		t.Errorf("archived content = %q, want %q", string(content), testContent)
	}
}

func TestArchiveActiveSessionHandoff_NoActiveDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Test archiving when no active directory exists (should not error)
	err := archiveActiveSessionHandoff(tmpDir)
	if err != nil {
		t.Errorf("archiveActiveSessionHandoff() should not error when active directory doesn't exist, got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && stringContains(s, substr)
}

// TestDiscoverSessionHandoff_WindowScoped tests the new window-scoped discovery
func TestDiscoverSessionHandoff_WindowScoped(t *testing.T) {
	tmpDir := t.TempDir()

	// Save and restore working directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDir)

	// Change to test directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Get the actual window name that will be used in discovery
	windowName, err := tmux.GetCurrentWindowName()
	if err != nil {
		t.Fatalf("failed to get window name: %v", err)
	}

	// Create window-scoped session directory structure
	sessionDir := filepath.Join(tmpDir, ".orch", "session", windowName, "2026-01-13-1400")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create SESSION_HANDOFF.md
	handoffPath := filepath.Join(sessionDir, "SESSION_HANDOFF.md")
	if err := os.WriteFile(handoffPath, []byte("window-scoped content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create latest symlink in window-scoped directory
	latestSymlink := filepath.Join(tmpDir, ".orch", "session", windowName, "latest")
	if err := os.Symlink("2026-01-13-1400", latestSymlink); err != nil {
		t.Fatal(err)
	}

	// Test discovery
	got, err := discoverSessionHandoff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it found the window-scoped handoff
	content, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("failed to read discovered handoff: %v", err)
	}
	if string(content) != "window-scoped content" {
		t.Errorf("got content %q, want %q", string(content), "window-scoped content")
	}
}

// TestDiscoverSessionHandoff_BackwardCompatibility tests fallback to legacy structure
func TestDiscoverSessionHandoff_BackwardCompatibility(t *testing.T) {
	tmpDir := t.TempDir()

	// Save and restore working directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDir)

	// Change to test directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create ONLY legacy (non-window-scoped) structure
	sessionDir := filepath.Join(tmpDir, ".orch", "session", "2026-01-13-1500")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create SESSION_HANDOFF.md in legacy location
	handoffPath := filepath.Join(sessionDir, "SESSION_HANDOFF.md")
	if err := os.WriteFile(handoffPath, []byte("legacy content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create legacy latest symlink (at root session level, not window-scoped)
	latestSymlink := filepath.Join(tmpDir, ".orch", "session", "latest")
	if err := os.Symlink("2026-01-13-1500", latestSymlink); err != nil {
		t.Fatal(err)
	}

	// Test discovery - should fall back to legacy structure
	got, err := discoverSessionHandoff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it found the legacy handoff
	content, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("failed to read discovered handoff: %v", err)
	}
	if string(content) != "legacy content" {
		t.Errorf("got content %q, want %q", string(content), "legacy content")
	}
}

// TestDiscoverSessionHandoff_PreferWindowScoped tests that window-scoped is preferred over legacy
func TestDiscoverSessionHandoff_PreferWindowScoped(t *testing.T) {
	tmpDir := t.TempDir()

	// Save and restore working directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDir)

	// Change to test directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Get the actual window name that will be used in discovery
	windowName, err := tmux.GetCurrentWindowName()
	if err != nil {
		t.Fatalf("failed to get window name: %v", err)
	}

	// Window-scoped structure
	windowScopedDir := filepath.Join(tmpDir, ".orch", "session", windowName, "2026-01-13-1600")
	if err := os.MkdirAll(windowScopedDir, 0755); err != nil {
		t.Fatal(err)
	}
	windowScopedHandoff := filepath.Join(windowScopedDir, "SESSION_HANDOFF.md")
	if err := os.WriteFile(windowScopedHandoff, []byte("window-scoped content"), 0644); err != nil {
		t.Fatal(err)
	}
	windowScopedLatest := filepath.Join(tmpDir, ".orch", "session", windowName, "latest")
	if err := os.Symlink("2026-01-13-1600", windowScopedLatest); err != nil {
		t.Fatal(err)
	}

	// Legacy structure
	legacyDir := filepath.Join(tmpDir, ".orch", "session", "2026-01-13-1500")
	if err := os.MkdirAll(legacyDir, 0755); err != nil {
		t.Fatal(err)
	}
	legacyHandoff := filepath.Join(legacyDir, "SESSION_HANDOFF.md")
	if err := os.WriteFile(legacyHandoff, []byte("legacy content"), 0644); err != nil {
		t.Fatal(err)
	}
	legacyLatest := filepath.Join(tmpDir, ".orch", "session", "latest")
	if err := os.Symlink("2026-01-13-1500", legacyLatest); err != nil {
		t.Fatal(err)
	}

	// Test discovery - should prefer window-scoped over legacy
	got, err := discoverSessionHandoff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it found the window-scoped handoff (not legacy)
	content, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("failed to read discovered handoff: %v", err)
	}
	if string(content) != "window-scoped content" {
		t.Errorf("got content %q, want %q (should prefer window-scoped)", string(content), "window-scoped content")
	}
}
