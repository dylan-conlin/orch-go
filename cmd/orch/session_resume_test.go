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
	err = archiveActiveSessionHandoff(tmpDir, windowName)
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
	// Use any window name since there's no active directory anyway
	err := archiveActiveSessionHandoff(tmpDir, "test-window")
	if err != nil {
		t.Errorf("archiveActiveSessionHandoff() should not error when active directory doesn't exist, got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && stringContains(s, substr)
}

// TestDiscoverSessionHandoff_WindowScoped tests the new window-scoped discovery
// Note: Cross-window scan requires an active/ directory somewhere to work,
// so this test creates an active directory to simulate mid-session crash recovery.
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

	// Create window-scoped session directory structure with ARCHIVED session
	sessionDir := filepath.Join(tmpDir, ".orch", "session", windowName, "2026-01-13-1400")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create SESSION_HANDOFF.md in archived session
	handoffPath := filepath.Join(sessionDir, "SESSION_HANDOFF.md")
	if err := os.WriteFile(handoffPath, []byte("window-scoped content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create latest symlink in window-scoped directory
	latestSymlink := filepath.Join(tmpDir, ".orch", "session", windowName, "latest")
	if err := os.Symlink("2026-01-13-1400", latestSymlink); err != nil {
		t.Fatal(err)
	}

	// IMPORTANT: Create an active/ directory to simulate crash recovery scenario.
	// Without this, cross-window scan is skipped (explicit session end behavior).
	activeDir := filepath.Join(tmpDir, ".orch", "session", windowName, "active")
	if err := os.MkdirAll(activeDir, 0755); err != nil {
		t.Fatal(err)
	}
	activeHandoff := filepath.Join(activeDir, "SESSION_HANDOFF.md")
	if err := os.WriteFile(activeHandoff, []byte("active session content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test discovery - should find active/ first (Priority 1)
	got, err := discoverSessionHandoff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it found the active handoff (not the archived one)
	content, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("failed to read discovered handoff: %v", err)
	}
	if string(content) != "active session content" {
		t.Errorf("got content %q, want %q", string(content), "active session content")
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

// TestDiscoverSessionHandoff_PreferActiveOverArchived tests that active/ is preferred over archived
func TestDiscoverSessionHandoff_PreferActiveOverArchived(t *testing.T) {
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

	// Archived window-scoped structure
	archivedDir := filepath.Join(tmpDir, ".orch", "session", windowName, "2026-01-13-1600")
	if err := os.MkdirAll(archivedDir, 0755); err != nil {
		t.Fatal(err)
	}
	archivedHandoff := filepath.Join(archivedDir, "SESSION_HANDOFF.md")
	if err := os.WriteFile(archivedHandoff, []byte("archived content"), 0644); err != nil {
		t.Fatal(err)
	}
	windowScopedLatest := filepath.Join(tmpDir, ".orch", "session", windowName, "latest")
	if err := os.Symlink("2026-01-13-1600", windowScopedLatest); err != nil {
		t.Fatal(err)
	}

	// Active directory (should be found first - Priority 1)
	activeDir := filepath.Join(tmpDir, ".orch", "session", windowName, "active")
	if err := os.MkdirAll(activeDir, 0755); err != nil {
		t.Fatal(err)
	}
	activeHandoff := filepath.Join(activeDir, "SESSION_HANDOFF.md")
	if err := os.WriteFile(activeHandoff, []byte("active content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Legacy structure (should be ignored when active exists)
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

	// Test discovery - should prefer active/ over archived
	got, err := discoverSessionHandoff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it found the active handoff (not archived or legacy)
	content, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("failed to read discovered handoff: %v", err)
	}
	if string(content) != "active content" {
		t.Errorf("got content %q, want %q (should prefer active/)", string(content), "active content")
	}
}

// TestDiscoverSessionHandoff_CrossWindowScan tests cross-window scan when current window has no history
// Note: Cross-window scan only works when there's an active/ directory somewhere,
// indicating a session is in progress (crash recovery scenario).
func TestDiscoverSessionHandoff_CrossWindowScan(t *testing.T) {
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

	// Create multiple window directories with different timestamps
	// Window 1: older session (2026-01-13-0800) with active/ to enable cross-window scan
	window1Dir := filepath.Join(tmpDir, ".orch", "session", "window1", "2026-01-13-0800")
	if err := os.MkdirAll(window1Dir, 0755); err != nil {
		t.Fatal(err)
	}
	window1Handoff := filepath.Join(window1Dir, "SESSION_HANDOFF.md")
	if err := os.WriteFile(window1Handoff, []byte("window1 content"), 0644); err != nil {
		t.Fatal(err)
	}
	window1Latest := filepath.Join(tmpDir, ".orch", "session", "window1", "latest")
	if err := os.Symlink("2026-01-13-0800", window1Latest); err != nil {
		t.Fatal(err)
	}
	// IMPORTANT: Create active/ in window1 to simulate crash recovery (enables cross-window scan)
	window1Active := filepath.Join(tmpDir, ".orch", "session", "window1", "active")
	if err := os.MkdirAll(window1Active, 0755); err != nil {
		t.Fatal(err)
	}
	window1ActiveHandoff := filepath.Join(window1Active, "SESSION_HANDOFF.md")
	if err := os.WriteFile(window1ActiveHandoff, []byte("window1 active"), 0644); err != nil {
		t.Fatal(err)
	}

	// Window 2: most recent ARCHIVED session (2026-01-13-1430) - this should be found
	window2Dir := filepath.Join(tmpDir, ".orch", "session", "window2", "2026-01-13-1430")
	if err := os.MkdirAll(window2Dir, 0755); err != nil {
		t.Fatal(err)
	}
	window2Handoff := filepath.Join(window2Dir, "SESSION_HANDOFF.md")
	if err := os.WriteFile(window2Handoff, []byte("window2 most recent"), 0644); err != nil {
		t.Fatal(err)
	}
	window2Latest := filepath.Join(tmpDir, ".orch", "session", "window2", "latest")
	if err := os.Symlink("2026-01-13-1430", window2Latest); err != nil {
		t.Fatal(err)
	}

	// Window 3: middle timestamp (2026-01-13-1200)
	window3Dir := filepath.Join(tmpDir, ".orch", "session", "window3", "2026-01-13-1200")
	if err := os.MkdirAll(window3Dir, 0755); err != nil {
		t.Fatal(err)
	}
	window3Handoff := filepath.Join(window3Dir, "SESSION_HANDOFF.md")
	if err := os.WriteFile(window3Handoff, []byte("window3 content"), 0644); err != nil {
		t.Fatal(err)
	}
	window3Latest := filepath.Join(tmpDir, ".orch", "session", "window3", "latest")
	if err := os.Symlink("2026-01-13-1200", window3Latest); err != nil {
		t.Fatal(err)
	}

	// Current window has NO history (will trigger cross-window scan)
	// discoverSessionHandoff will get the current window name, find no history,
	// then scan all windows and return the most recent archived (window2)
	// Note: window1's active/ enables the cross-window scan, but the scan returns
	// the most recent ARCHIVED session which is window2.

	// Test discovery - should find window2's handoff (most recent archived across all windows)
	got, err := discoverSessionHandoff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it found the most recent archived handoff across all windows
	content, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("failed to read discovered handoff: %v", err)
	}
	if string(content) != "window2 most recent" {
		t.Errorf("got content %q, want %q (should find most recent archived across all windows)", string(content), "window2 most recent")
	}
}

// TestDiscoverSessionHandoff_NoHandoffAfterExplicitEnd tests that no handoff is returned
// after explicit session end (no active/ directory exists anywhere).
// This is the key fix for the "stale handoff injection" bug.
func TestDiscoverSessionHandoff_NoHandoffAfterExplicitEnd(t *testing.T) {
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

	// Create ONLY archived sessions (simulating state after "orch session end")
	// NO active/ directory anywhere
	sessionDir := filepath.Join(tmpDir, ".orch", "session", windowName, "2026-01-13-1600")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(sessionDir, "SESSION_HANDOFF.md")
	if err := os.WriteFile(handoffPath, []byte("archived session content"), 0644); err != nil {
		t.Fatal(err)
	}
	latestSymlink := filepath.Join(tmpDir, ".orch", "session", windowName, "latest")
	if err := os.Symlink("2026-01-13-1600", latestSymlink); err != nil {
		t.Fatal(err)
	}

	// Also create another window's archived session (to test cross-window scan is skipped)
	otherWindowDir := filepath.Join(tmpDir, ".orch", "session", "other-window", "2026-01-13-1700")
	if err := os.MkdirAll(otherWindowDir, 0755); err != nil {
		t.Fatal(err)
	}
	otherHandoff := filepath.Join(otherWindowDir, "SESSION_HANDOFF.md")
	if err := os.WriteFile(otherHandoff, []byte("other window content"), 0644); err != nil {
		t.Fatal(err)
	}
	otherLatest := filepath.Join(tmpDir, ".orch", "session", "other-window", "latest")
	if err := os.Symlink("2026-01-13-1700", otherLatest); err != nil {
		t.Fatal(err)
	}

	// Test discovery - should return error (no handoff) because no active/ exists
	// This is the expected behavior after explicit "orch session end"
	_, err = discoverSessionHandoff()
	if err == nil {
		t.Error("expected error (no handoff) after explicit session end, but got nil")
	}
	// Verify the error mentions "no session handoff found"
	if err != nil && !stringContains(err.Error(), "no session handoff found") {
		t.Errorf("expected error about 'no session handoff found', got: %v", err)
	}
}
