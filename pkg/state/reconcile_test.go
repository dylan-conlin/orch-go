package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsLive(t *testing.T) {
	// This tests the basic structure of the IsLive function.
	// Integration tests would require mocking tmux and OpenCode.

	tests := []struct {
		name       string
		beadsID    string
		serverURL  string
		projectDir string
		// We can't easily test tmux/OpenCode without mocks,
		// so we test the function returns expected defaults for invalid inputs
	}{
		{
			name:       "empty beads ID returns false for both",
			beadsID:    "",
			serverURL:  "http://localhost:4096",
			projectDir: "/tmp/nonexistent",
		},
		{
			name:       "nonexistent beads ID returns false for both",
			beadsID:    "nonexistent-abc123",
			serverURL:  "http://localhost:4096",
			projectDir: "/tmp/nonexistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmuxLive, opencodeLive := IsLive(tt.beadsID, tt.serverURL, tt.projectDir)
			// For invalid/nonexistent inputs, both should be false
			if tmuxLive || opencodeLive {
				t.Errorf("IsLive(%q) = (%v, %v), want (false, false) for invalid input",
					tt.beadsID, tmuxLive, opencodeLive)
			}
		})
	}
}

func TestFindWorkspaceByBeadsID(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	workspaceBase := filepath.Join(tempDir, ".orch", "workspace")

	// Create workspace directories
	err := os.MkdirAll(filepath.Join(workspaceBase, "og-feat-test-abc12-22dec"), 0755)
	if err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	// Create a SPAWN_CONTEXT.md with beads ID
	spawnContext := `TASK: Test task
You were spawned from beads issue: **proj-xyz78**
`
	err = os.WriteFile(
		filepath.Join(workspaceBase, "og-feat-test-abc12-22dec", "SPAWN_CONTEXT.md"),
		[]byte(spawnContext),
		0644,
	)
	if err != nil {
		t.Fatalf("Failed to write SPAWN_CONTEXT.md: %v", err)
	}

	tests := []struct {
		name          string
		projectDir    string
		beadsID       string
		wantPath      bool // whether we expect a non-empty path
		wantAgentName string
	}{
		{
			name:          "find by beads ID in directory name",
			projectDir:    tempDir,
			beadsID:       "abc12",
			wantPath:      true,
			wantAgentName: "og-feat-test-abc12-22dec",
		},
		{
			name:          "find by beads ID in SPAWN_CONTEXT.md",
			projectDir:    tempDir,
			beadsID:       "proj-xyz78",
			wantPath:      true,
			wantAgentName: "og-feat-test-abc12-22dec",
		},
		{
			name:          "nonexistent beads ID",
			projectDir:    tempDir,
			beadsID:       "nonexistent",
			wantPath:      false,
			wantAgentName: "",
		},
		{
			name:          "empty project dir",
			projectDir:    "",
			beadsID:       "abc12",
			wantPath:      false,
			wantAgentName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, agentName := FindWorkspaceByBeadsID(tt.projectDir, tt.beadsID)

			if tt.wantPath && path == "" {
				t.Errorf("FindWorkspaceByBeadsID(%q, %q) returned empty path, want non-empty",
					tt.projectDir, tt.beadsID)
			}
			if !tt.wantPath && path != "" {
				t.Errorf("FindWorkspaceByBeadsID(%q, %q) = %q, want empty path",
					tt.projectDir, tt.beadsID, path)
			}
			if agentName != tt.wantAgentName {
				t.Errorf("FindWorkspaceByBeadsID(%q, %q) agentName = %q, want %q",
					tt.projectDir, tt.beadsID, agentName, tt.wantAgentName)
			}
		})
	}
}

func TestLivenessResult(t *testing.T) {
	// Test the LivenessResult helper methods
	t.Run("IsAlive returns true if any source is live", func(t *testing.T) {
		tests := []struct {
			name   string
			result LivenessResult
			want   bool
		}{
			{
				name:   "both dead",
				result: LivenessResult{TmuxLive: false, OpencodeLive: false},
				want:   false,
			},
			{
				name:   "tmux alive only",
				result: LivenessResult{TmuxLive: true, OpencodeLive: false},
				want:   true,
			},
			{
				name:   "opencode alive only",
				result: LivenessResult{TmuxLive: false, OpencodeLive: true},
				want:   true,
			},
			{
				name:   "both alive",
				result: LivenessResult{TmuxLive: true, OpencodeLive: true},
				want:   true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.result.IsAlive(); got != tt.want {
					t.Errorf("LivenessResult.IsAlive() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("IsPhantom returns true when beads open but no live sources", func(t *testing.T) {
		tests := []struct {
			name   string
			result LivenessResult
			want   bool
		}{
			{
				name:   "phantom - beads open but nothing running",
				result: LivenessResult{BeadsOpen: true, TmuxLive: false, OpencodeLive: false},
				want:   true,
			},
			{
				name:   "not phantom - beads open and tmux running",
				result: LivenessResult{BeadsOpen: true, TmuxLive: true, OpencodeLive: false},
				want:   false,
			},
			{
				name:   "not phantom - beads closed",
				result: LivenessResult{BeadsOpen: false, TmuxLive: false, OpencodeLive: false},
				want:   false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.result.IsPhantom(); got != tt.want {
					t.Errorf("LivenessResult.IsPhantom() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}

func TestAgentStatusResult(t *testing.T) {
	// Test AgentStatusResult status values
	t.Run("Status constants have expected values", func(t *testing.T) {
		if StatusRunning != "running" {
			t.Errorf("StatusRunning = %q, want %q", StatusRunning, "running")
		}
		if StatusIdle != "idle" {
			t.Errorf("StatusIdle = %q, want %q", StatusIdle, "idle")
		}
		if StatusCompleted != "completed" {
			t.Errorf("StatusCompleted = %q, want %q", StatusCompleted, "completed")
		}
		if StatusStale != "stale" {
			t.Errorf("StatusStale = %q, want %q", StatusStale, "stale")
		}
	})

	t.Run("Status string can be used in comparisons", func(t *testing.T) {
		result := AgentStatusResult{Status: StatusRunning}
		if string(result.Status) != "running" {
			t.Errorf("string(Status) = %q, want %q", string(result.Status), "running")
		}
	})
}

func TestDetermineAgentStatusWithoutClient(t *testing.T) {
	// Test DetermineAgentStatus with nil client (basic status checks)
	// Full integration tests would require mocking OpenCode and beads

	t.Run("Empty beads ID returns stale status", func(t *testing.T) {
		result := DetermineAgentStatus(nil, nil, "", "/tmp", 30*time.Minute)
		if result.Status != StatusStale {
			t.Errorf("Status = %q, want %q for empty beads ID", result.Status, StatusStale)
		}
	})

	t.Run("Result includes beads ID", func(t *testing.T) {
		result := DetermineAgentStatus(nil, nil, "test-123", "/tmp", 30*time.Minute)
		if result.BeadsID != "test-123" {
			t.Errorf("BeadsID = %q, want %q", result.BeadsID, "test-123")
		}
	})
}

func TestExtractProjectDirFromSpawnContext(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "valid PROJECT_DIR",
			content: "TASK: Test\n\nPROJECT_DIR: /Users/test/project\n\nSome other stuff",
			want:    "/Users/test/project",
		},
		{
			name:    "PROJECT_DIR with spaces after colon",
			content: "PROJECT_DIR:   /Users/test/project  \n",
			want:    "/Users/test/project",
		},
		{
			name:    "no PROJECT_DIR",
			content: "TASK: Test\nSome content without project dir",
			want:    "",
		},
		{
			name:    "empty content",
			content: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractProjectDirFromSpawnContext(tt.content)
			if got != tt.want {
				t.Errorf("extractProjectDirFromSpawnContext() = %q, want %q", got, tt.want)
			}
		})
	}
}
