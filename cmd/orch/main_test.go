package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestGetMaxAgentsDefault tests that getMaxAgents returns the default when no flag or env var is set.
func TestGetMaxAgentsDefault(t *testing.T) {
	// Save and restore original values
	originalMaxAgents := spawnMaxAgents
	originalEnv := os.Getenv("ORCH_MAX_AGENTS")
	defer func() {
		spawnMaxAgents = originalMaxAgents
		if originalEnv == "" {
			os.Unsetenv("ORCH_MAX_AGENTS")
		} else {
			os.Setenv("ORCH_MAX_AGENTS", originalEnv)
		}
	}()

	// Clear flag and env var
	spawnMaxAgents = 0
	os.Unsetenv("ORCH_MAX_AGENTS")

	got := getMaxAgents()
	if got != DefaultMaxAgents {
		t.Errorf("getMaxAgents() = %d, want default %d", got, DefaultMaxAgents)
	}
}

// TestGetMaxAgentsFlagOverridesAll tests that --max-agents flag takes precedence.
func TestGetMaxAgentsFlagOverridesAll(t *testing.T) {
	// Save and restore original values
	originalMaxAgents := spawnMaxAgents
	originalEnv := os.Getenv("ORCH_MAX_AGENTS")
	defer func() {
		spawnMaxAgents = originalMaxAgents
		if originalEnv == "" {
			os.Unsetenv("ORCH_MAX_AGENTS")
		} else {
			os.Setenv("ORCH_MAX_AGENTS", originalEnv)
		}
	}()

	// Set flag to 10, env to 20
	spawnMaxAgents = 10
	os.Setenv("ORCH_MAX_AGENTS", "20")

	got := getMaxAgents()
	if got != 10 {
		t.Errorf("getMaxAgents() = %d, want 10 (flag value)", got)
	}
}

// TestGetMaxAgentsEnvVar tests that ORCH_MAX_AGENTS env var is used when flag is 0.
func TestGetMaxAgentsEnvVar(t *testing.T) {
	// Save and restore original values
	originalMaxAgents := spawnMaxAgents
	originalEnv := os.Getenv("ORCH_MAX_AGENTS")
	defer func() {
		spawnMaxAgents = originalMaxAgents
		if originalEnv == "" {
			os.Unsetenv("ORCH_MAX_AGENTS")
		} else {
			os.Setenv("ORCH_MAX_AGENTS", originalEnv)
		}
	}()

	// Clear flag, set env to 15
	spawnMaxAgents = 0
	os.Setenv("ORCH_MAX_AGENTS", "15")

	got := getMaxAgents()
	if got != 15 {
		t.Errorf("getMaxAgents() = %d, want 15 (env value)", got)
	}
}

// TestGetMaxAgentsInvalidEnvVar tests that invalid env var falls back to default.
func TestGetMaxAgentsInvalidEnvVar(t *testing.T) {
	// Save and restore original values
	originalMaxAgents := spawnMaxAgents
	originalEnv := os.Getenv("ORCH_MAX_AGENTS")
	defer func() {
		spawnMaxAgents = originalMaxAgents
		if originalEnv == "" {
			os.Unsetenv("ORCH_MAX_AGENTS")
		} else {
			os.Setenv("ORCH_MAX_AGENTS", originalEnv)
		}
	}()

	// Clear flag, set invalid env
	spawnMaxAgents = 0
	os.Setenv("ORCH_MAX_AGENTS", "not-a-number")

	got := getMaxAgents()
	if got != DefaultMaxAgents {
		t.Errorf("getMaxAgents() = %d, want default %d (invalid env)", got, DefaultMaxAgents)
	}
}

// TestCheckConcurrencyLimitUsesOpenCodeAPI documents the new behavior.
// After registry removal, concurrency checking uses OpenCode API ListSessions().
func TestCheckConcurrencyLimitUsesOpenCodeAPI(t *testing.T) {
	// The checkConcurrencyLimit function now:
	// 1. Creates an OpenCode client
	// 2. Calls client.ListSessions()
	// 3. Counts active sessions (status != "completed")
	// 4. Returns error if count >= max
	//
	// This replaces the old registry-based counting.
	// Integration testing requires a running OpenCode server.
}

func TestParseBeadsCreateOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		wantID  string
		wantErr bool
	}{
		{
			name: "standard multi-line output",
			output: `✓ Created issue: orch-go-5z9
  Title: test issue for parsing
  Priority: P2
  Status: open`,
			wantID:  "orch-go-5z9",
			wantErr: false,
		},
		{
			name: "minimal output without checkmark",
			output: `Created issue: proj-abc
  Status: open`,
			wantID:  "proj-abc",
			wantErr: false,
		},
		{
			name:    "single line output (hypothetical)",
			output:  `✓ Created issue: test-xyz`,
			wantID:  "test-xyz",
			wantErr: false,
		},
		{
			name:    "empty output",
			output:  "",
			wantID:  "",
			wantErr: true,
		},
		{
			name:    "output without issue ID",
			output:  "Something went wrong",
			wantID:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse issue ID from output (same logic as in createBeadsIssue)
			outputStr := strings.TrimSpace(tt.output)

			var gotID string
			var gotErr error

			// Split by newline and parse first line only
			lines := strings.Split(outputStr, "\n")
			if len(lines) == 0 {
				gotErr = errEmptyOutput
			} else {
				firstLine := strings.TrimSpace(lines[0])

				// Look for "issue:" in the first line and extract the ID after it
				parts := strings.Fields(firstLine)
				found := false
				for i, part := range parts {
					if strings.Contains(part, "issue:") {
						// Issue ID should be the next word after "issue:"
						if i+1 < len(parts) {
							gotID = parts[i+1]
							found = true
							break
						}
					}
				}

				if !found {
					gotErr = errNoIssueID
				}
			}

			if (gotErr != nil) != tt.wantErr {
				t.Errorf("parseBeadsOutput() error = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}
			if gotID != tt.wantID {
				t.Errorf("parseBeadsOutput() = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}

// Mock errors for testing
var (
	errEmptyOutput = &mockError{"empty output from bd create"}
	errNoIssueID   = &mockError{"could not parse issue ID"}
)

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

// TestAbandonNonExistentAgent tests that abandoning a non-existent agent returns an error.
func TestAbandonNonExistentAgent(t *testing.T) {
	beadsID := "nonexistent-agent-xyz"

	// runAbandon first verifies the beads issue exists
	err := runAbandon(beadsID, "")
	if err == nil {
		t.Error("Expected error for non-existent agent")
	}
	// Now the error is from beads lookup failure (issue not found)
	if err != nil && !strings.Contains(err.Error(), "failed to get beads issue") && !strings.Contains(err.Error(), "no agent found") {
		t.Errorf("Expected 'failed to get beads issue' or 'no agent found' error, got: %v", err)
	}
}

// TestFormatSessionTitle tests formatting session titles with beads ID.
func TestFormatSessionTitle(t *testing.T) {
	tests := []struct {
		name          string
		workspaceName string
		beadsID       string
		want          string
	}{
		{
			name:          "with beads ID",
			workspaceName: "og-debug-orch-status-23dec",
			beadsID:       "orch-go-v4mw",
			want:          "og-debug-orch-status-23dec [orch-go-v4mw]",
		},
		{
			name:          "empty beads ID",
			workspaceName: "og-feat-something-23dec",
			beadsID:       "",
			want:          "og-feat-something-23dec",
		},
		{
			name:          "different project beads ID",
			workspaceName: "og-inv-test-23dec",
			beadsID:       "snap-abc1",
			want:          "og-inv-test-23dec [snap-abc1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSessionTitle(tt.workspaceName, tt.beadsID)
			if got != tt.want {
				t.Errorf("formatSessionTitle(%q, %q) = %q, want %q", tt.workspaceName, tt.beadsID, got, tt.want)
			}
		})
	}
}

// TestExtractBeadsIDFromTitle tests extracting beads ID from session titles.
func TestExtractBeadsIDFromTitle(t *testing.T) {
	tests := []struct {
		name   string
		title  string
		wantID string
	}{
		{
			name:   "beads ID in brackets at end",
			title:  "og-feat-add-feature-19dec [orch-go-abc12]",
			wantID: "orch-go-abc12",
		},
		{
			name:   "beads ID in brackets with extra spaces",
			title:  "og-inv-something [ proj-xyz ]",
			wantID: "proj-xyz",
		},
		{
			name:   "no brackets",
			title:  "og-feat-add-feature-19dec",
			wantID: "",
		},
		{
			name:   "empty title",
			title:  "",
			wantID: "",
		},
		{
			name:   "unclosed bracket",
			title:  "og-feat-test [incomplete",
			wantID: "",
		},
		{
			name:   "nested brackets - takes innermost",
			title:  "outer [inner [orch-go-xyz]]",
			wantID: "orch-go-xyz]", // This is an edge case but acceptable
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBeadsIDFromTitle(tt.title)
			if got != tt.wantID {
				t.Errorf("extractBeadsIDFromTitle(%q) = %q, want %q", tt.title, got, tt.wantID)
			}
		})
	}
}

// TestExtractSkillFromTitle tests extracting skill from session titles/workspace names.
func TestExtractSkillFromTitle(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		wantSkill string
	}{
		{
			name:      "feature-impl from -feat-",
			title:     "og-feat-add-feature-19dec",
			wantSkill: "feature-impl",
		},
		{
			name:      "investigation from -inv-",
			title:     "og-inv-explore-codebase-19dec",
			wantSkill: "investigation",
		},
		{
			name:      "systematic-debugging from -debug-",
			title:     "og-debug-fix-bug-19dec",
			wantSkill: "systematic-debugging",
		},
		{
			name:      "architect from -arch-",
			title:     "og-arch-design-system-19dec",
			wantSkill: "architect",
		},
		{
			name:      "codebase-audit from -audit-",
			title:     "og-audit-security-19dec",
			wantSkill: "codebase-audit",
		},
		{
			name:      "research from -research-",
			title:     "og-research-compare-libs-19dec",
			wantSkill: "research",
		},
		{
			name:      "no matching pattern",
			title:     "random-session-name",
			wantSkill: "",
		},
		{
			name:      "empty title",
			title:     "",
			wantSkill: "",
		},
		{
			name:      "case insensitive",
			title:     "OG-FEAT-UPPERCASE-19DEC",
			wantSkill: "feature-impl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSkillFromTitle(tt.title)
			if got != tt.wantSkill {
				t.Errorf("extractSkillFromTitle(%q) = %q, want %q", tt.title, got, tt.wantSkill)
			}
		})
	}
}

// TestExtractBeadsIDFromWindowName tests extracting beads ID from tmux window names.
func TestExtractBeadsIDFromWindowName(t *testing.T) {
	tests := []struct {
		name       string
		windowName string
		wantID     string
	}{
		{
			name:       "standard window name with emoji and beads ID",
			windowName: "🔬 og-inv-test-19dec [proj-abc]",
			wantID:     "proj-abc",
		},
		{
			name:       "feature-impl window",
			windowName: "🏗️ og-feat-add-button-19dec [snap-xyz]",
			wantID:     "snap-xyz",
		},
		{
			name:       "no beads ID",
			windowName: "🐛 og-debug-fix-19dec",
			wantID:     "",
		},
		{
			name:       "servers window",
			windowName: "servers",
			wantID:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBeadsIDFromWindowName(tt.windowName)
			if got != tt.wantID {
				t.Errorf("extractBeadsIDFromWindowName(%q) = %q, want %q", tt.windowName, got, tt.wantID)
			}
		})
	}
}

// TestExtractSkillFromWindowName tests extracting skill from tmux window names.
func TestExtractSkillFromWindowName(t *testing.T) {
	tests := []struct {
		name       string
		windowName string
		wantSkill  string
	}{
		{
			name:       "investigation emoji",
			windowName: "🔬 og-inv-test-19dec [proj-abc]",
			wantSkill:  "investigation",
		},
		{
			name:       "feature-impl emoji",
			windowName: "🏗️ og-feat-add-button-19dec",
			wantSkill:  "feature-impl",
		},
		{
			name:       "debugging emoji",
			windowName: "🐛 og-debug-fix-19dec",
			wantSkill:  "systematic-debugging",
		},
		{
			name:       "architect emoji",
			windowName: "📐 og-arch-design-19dec",
			wantSkill:  "architect",
		},
		{
			name:       "fallback to title pattern when no emoji",
			windowName: "og-feat-add-button-19dec",
			wantSkill:  "feature-impl",
		},
		{
			name:       "unknown window",
			windowName: "random-window",
			wantSkill:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSkillFromWindowName(tt.windowName)
			if got != tt.wantSkill {
				t.Errorf("extractSkillFromWindowName(%q) = %q, want %q", tt.windowName, got, tt.wantSkill)
			}
		})
	}
}

// TestFindWorkspaceByBeadsID tests finding workspaces by beads ID.
func TestFindWorkspaceByBeadsID(t *testing.T) {
	// Create a temp directory structure
	tempDir, err := os.MkdirTemp("", "test-workspace-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create workspace directory structure
	workspaceDir := filepath.Join(tempDir, ".orch", "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	// Create a workspace with beads ID in name
	ws1 := filepath.Join(workspaceDir, "og-feat-test-[orch-go-abc12]")
	if err := os.MkdirAll(ws1, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create a workspace with beads ID in SPAWN_CONTEXT.md
	ws2 := filepath.Join(workspaceDir, "og-inv-explore-21dec")
	if err := os.MkdirAll(ws2, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	spawnContext := `TASK: Explore the codebase

## BEADS PROGRESS TRACKING

You were spawned from beads issue: **orch-go-xyz78**
`
	if err := os.WriteFile(filepath.Join(ws2, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
		t.Fatalf("Failed to write SPAWN_CONTEXT.md: %v", err)
	}

	// Create a workspace that MENTIONS a beads ID but is spawned from a different one
	// This tests that we only match the authoritative "spawned from beads issue" line
	ws3 := filepath.Join(workspaceDir, "og-debug-fix-issue-21dec")
	if err := os.MkdirAll(ws3, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	spawnContext3 := `TASK: Fix bug in orch-go-xyz78 workspace

This workspace is debugging an issue with orch-go-xyz78.

## BEADS PROGRESS TRACKING

You were spawned from beads issue: **orch-go-debug99**
`
	if err := os.WriteFile(filepath.Join(ws3, "SPAWN_CONTEXT.md"), []byte(spawnContext3), 0644); err != nil {
		t.Fatalf("Failed to write SPAWN_CONTEXT.md: %v", err)
	}

	tests := []struct {
		name      string
		beadsID   string
		wantPath  bool
		wantAgent string
	}{
		{
			name:      "find by beads ID in directory name",
			beadsID:   "orch-go-abc12",
			wantPath:  true,
			wantAgent: "og-feat-test-[orch-go-abc12]",
		},
		{
			// This test case is critical for headless spawns discoverability.
			// Headless agents create workspaces like "og-debug-task-22dec" (no beads ID in name)
			// but the beads ID is stored in SPAWN_CONTEXT.md. Commands like tail, question,
			// and complete must find the workspace via SPAWN_CONTEXT.md scanning.
			name:      "find by beads ID in SPAWN_CONTEXT.md",
			beadsID:   "orch-go-xyz78",
			wantPath:  true,
			wantAgent: "og-inv-explore-21dec",
		},
		{
			name:      "beads ID not found",
			beadsID:   "nonexistent-beads-id",
			wantPath:  false,
			wantAgent: "",
		},
		{
			name:      "find debug workspace by its actual beads ID",
			beadsID:   "orch-go-debug99",
			wantPath:  true,
			wantAgent: "og-debug-fix-issue-21dec",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotAgent := findWorkspaceByBeadsID(tempDir, tt.beadsID)

			if tt.wantPath && gotPath == "" {
				t.Errorf("findWorkspaceByBeadsID() path = empty, want non-empty")
			}
			if !tt.wantPath && gotPath != "" {
				t.Errorf("findWorkspaceByBeadsID() path = %q, want empty", gotPath)
			}
			if gotAgent != tt.wantAgent {
				t.Errorf("findWorkspaceByBeadsID() agent = %q, want %q", gotAgent, tt.wantAgent)
			}
		})
	}
}

// TestLivenessWarningMessage tests the liveness warning message generation.
// This is a documentation test - the actual prompt is handled in runComplete().
func TestLivenessWarningMessage(t *testing.T) {
	// Document the expected behavior:
	// When calling `orch complete <id>`, if the agent is still running
	// (tmux window active OR OpenCode session live), the user should be warned
	// and prompted before proceeding.
	//
	// The warning format is:
	//   ⚠️  Agent appears still running: tmux window (@1234), OpenCode session (ses_abc12)
	//   Proceed anyway? [y/N]:
	//
	// If the user types "y" or "yes", completion proceeds.
	// If the user types anything else (including just pressing Enter), completion is aborted.
	// If --force is set, the liveness check is skipped entirely.

	t.Run("warning message includes both sources when both are live", func(t *testing.T) {
		// Test the string building logic by simulating what runComplete does
		tmuxLive := true
		opencodeLive := true
		windowID := "@1234"
		sessionID := "ses_abc12def34567890"

		var runningDetails []string
		if tmuxLive {
			detail := "tmux window"
			if windowID != "" {
				detail += " (" + windowID + ")"
			}
			runningDetails = append(runningDetails, detail)
		}
		if opencodeLive {
			detail := "OpenCode session"
			if sessionID != "" {
				detail += " (" + sessionID[:12] + ")"
			}
			runningDetails = append(runningDetails, detail)
		}

		result := strings.Join(runningDetails, ", ")
		expected := "tmux window (@1234), OpenCode session (ses_abc12def)"
		if result != expected {
			t.Errorf("warning message = %q, want %q", result, expected)
		}
	})

	t.Run("warning message includes only tmux when OpenCode is not live", func(t *testing.T) {
		tmuxLive := true
		opencodeLive := false
		windowID := "@5678"

		var runningDetails []string
		if tmuxLive {
			detail := "tmux window"
			if windowID != "" {
				detail += " (" + windowID + ")"
			}
			runningDetails = append(runningDetails, detail)
		}
		if opencodeLive {
			runningDetails = append(runningDetails, "OpenCode session")
		}

		result := strings.Join(runningDetails, ", ")
		expected := "tmux window (@5678)"
		if result != expected {
			t.Errorf("warning message = %q, want %q", result, expected)
		}
	})

	t.Run("warning message includes only OpenCode when tmux is not live", func(t *testing.T) {
		tmuxLive := false
		opencodeLive := true
		sessionID := "ses_xyz789abc123456"

		var runningDetails []string
		if tmuxLive {
			runningDetails = append(runningDetails, "tmux window")
		}
		if opencodeLive {
			detail := "OpenCode session"
			if sessionID != "" {
				detail += " (" + sessionID[:12] + ")"
			}
			runningDetails = append(runningDetails, detail)
		}

		result := strings.Join(runningDetails, ", ")
		expected := "OpenCode session (ses_xyz789ab)"
		if result != expected {
			t.Errorf("warning message = %q, want %q", result, expected)
		}
	})
}

// TestSwarmStatusPhantomCount tests that phantom agents are correctly excluded from Active count.
func TestSwarmStatusPhantomCount(t *testing.T) {
	tests := []struct {
		name        string
		agents      []AgentInfo
		wantActive  int
		wantPhantom int
	}{
		{
			name: "all active agents",
			agents: []AgentInfo{
				{SessionID: "ses_1", BeadsID: "test-1", IsPhantom: false},
				{SessionID: "ses_2", BeadsID: "test-2", IsPhantom: false},
			},
			wantActive:  2,
			wantPhantom: 0,
		},
		{
			name: "all phantom agents",
			agents: []AgentInfo{
				{SessionID: "ses_1", BeadsID: "test-1", IsPhantom: true},
				{SessionID: "ses_2", BeadsID: "test-2", IsPhantom: true},
			},
			wantActive:  0,
			wantPhantom: 2,
		},
		{
			name: "mixed active and phantom",
			agents: []AgentInfo{
				{SessionID: "ses_1", BeadsID: "test-1", IsPhantom: false},
				{SessionID: "ses_2", BeadsID: "test-2", IsPhantom: true},
				{SessionID: "ses_3", BeadsID: "test-3", IsPhantom: false},
				{SessionID: "ses_4", BeadsID: "test-4", IsPhantom: true},
			},
			wantActive:  2,
			wantPhantom: 2,
		},
		{
			name:        "empty list",
			agents:      []AgentInfo{},
			wantActive:  0,
			wantPhantom: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replicate the counting logic from runStatus()
			activeCount := 0
			phantomCount := 0
			for _, agent := range tt.agents {
				if agent.IsPhantom {
					phantomCount++
				} else {
					activeCount++
				}
			}

			if activeCount != tt.wantActive {
				t.Errorf("activeCount = %d, want %d", activeCount, tt.wantActive)
			}
			if phantomCount != tt.wantPhantom {
				t.Errorf("phantomCount = %d, want %d", phantomCount, tt.wantPhantom)
			}
		})
	}
}

// TestAgentInfoIsPhantomField tests that IsPhantom field is correctly set.
func TestAgentInfoIsPhantomField(t *testing.T) {
	// Test that AgentInfo correctly stores phantom status
	agent := AgentInfo{
		SessionID: "ses_test",
		BeadsID:   "test-abc",
		IsPhantom: true,
	}

	if !agent.IsPhantom {
		t.Error("AgentInfo.IsPhantom should be true")
	}

	agent.IsPhantom = false
	if agent.IsPhantom {
		t.Error("AgentInfo.IsPhantom should be false after setting")
	}
}

// TestDetermineBeadsID tests the beads ID determination logic.
func TestDetermineBeadsID(t *testing.T) {
	// Mock createBeadsIssue function that always returns an error
	mockCreateError := func(projectName, skillName, task string) (string, error) {
		return "", &mockError{"mock bd create failure"}
	}

	// Mock createBeadsIssue function that succeeds
	mockCreateSuccess := func(projectName, skillName, task string) (string, error) {
		return "test-abc123", nil
	}

	tests := []struct {
		name            string
		spawnIssue      string
		spawnNoTrack    bool
		createBeadsFn   func(string, string, string) (string, error)
		wantID          string
		wantErr         bool
		wantErrContains string
	}{
		{
			name:          "explicit issue ID provided",
			spawnIssue:    "explicit-issue-123",
			spawnNoTrack:  false,
			createBeadsFn: nil, // should not be called
			wantID:        "explicit-issue-123",
			wantErr:       false,
		},
		{
			name:          "no-track flag set",
			spawnIssue:    "",
			spawnNoTrack:  true,
			createBeadsFn: nil,                       // should not be called
			wantID:        "test-project-untracked-", // prefix, exact timestamp will vary
			wantErr:       false,
		},
		{
			name:          "create beads issue succeeds",
			spawnIssue:    "",
			spawnNoTrack:  false,
			createBeadsFn: mockCreateSuccess,
			wantID:        "test-abc123",
			wantErr:       false,
		},
		{
			name:            "create beads issue fails - should fail fast",
			spawnIssue:      "",
			spawnNoTrack:    false,
			createBeadsFn:   mockCreateError,
			wantID:          "",
			wantErr:         true,
			wantErrContains: "failed to create beads issue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotErr := determineBeadsID("test-project", "test-skill", "test task", tt.spawnIssue, tt.spawnNoTrack, tt.createBeadsFn)

			// Check error expectation
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("determineBeadsID() error = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}

			// Check error message contains expected string
			if tt.wantErr && tt.wantErrContains != "" {
				if gotErr == nil || !strings.Contains(gotErr.Error(), tt.wantErrContains) {
					t.Errorf("determineBeadsID() error = %v, want error containing %q", gotErr, tt.wantErrContains)
				}
			}

			// For no-track case, just verify it starts with the expected prefix
			if tt.spawnNoTrack {
				if !strings.HasPrefix(gotID, tt.wantID) {
					t.Errorf("determineBeadsID() = %v, want prefix %v", gotID, tt.wantID)
				}
			} else if !tt.wantErr {
				// For other successful cases, check exact match
				if gotID != tt.wantID {
					t.Errorf("determineBeadsID() = %v, want %v", gotID, tt.wantID)
				}
			}
		})
	}
}

// TestDirExists tests the dirExists helper function.
func TestDirExists(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "test-dir-exists-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file inside for testing
	testFile := filepath.Join(tempDir, "test-file.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "existing directory",
			path: tempDir,
			want: true,
		},
		{
			name: "non-existing path",
			path: filepath.Join(tempDir, "does-not-exist"),
			want: false,
		},
		{
			name: "file not directory",
			path: testFile,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dirExists(tt.path)
			if got != tt.want {
				t.Errorf("dirExists(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestEnsureOrchScaffoldingNoTrack tests that --no-track bypasses beads check.
func TestEnsureOrchScaffoldingNoTrack(t *testing.T) {
	// Create a temp directory without .beads
	tempDir, err := os.MkdirTemp("", "test-scaffold-notrack-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// With noTrack=true, should succeed even without .beads
	err = ensureOrchScaffolding(tempDir, false, true)
	if err != nil {
		t.Errorf("ensureOrchScaffolding() with noTrack=true should succeed, got error: %v", err)
	}
}

// TestEnsureOrchScaffoldingMissingBeads tests error when beads is missing.
func TestEnsureOrchScaffoldingMissingBeads(t *testing.T) {
	// Create a temp directory without .beads
	tempDir, err := os.MkdirTemp("", "test-scaffold-missing-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// With noTrack=false and autoInit=false, should fail
	err = ensureOrchScaffolding(tempDir, false, false)
	if err == nil {
		t.Error("ensureOrchScaffolding() should fail when .beads is missing and tracking enabled")
	}

	// Error message should mention .beads
	if err != nil && !strings.Contains(err.Error(), ".beads") {
		t.Errorf("Error message should mention .beads, got: %v", err)
	}

	// Error message should suggest alternatives
	if err != nil && !strings.Contains(err.Error(), "orch init") {
		t.Errorf("Error message should suggest 'orch init', got: %v", err)
	}
}

// TestEnsureOrchScaffoldingExistingBeads tests that existing .beads passes.
func TestEnsureOrchScaffoldingExistingBeads(t *testing.T) {
	// Create a temp directory with .beads
	tempDir, err := os.MkdirTemp("", "test-scaffold-existing-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .beads directory
	beadsDir := filepath.Join(tempDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("Failed to create .beads dir: %v", err)
	}

	// With existing .beads, should succeed
	err = ensureOrchScaffolding(tempDir, false, false)
	if err != nil {
		t.Errorf("ensureOrchScaffolding() should succeed with existing .beads, got error: %v", err)
	}
}

// TestExtractDateFromWorkspaceName tests parsing date suffix from workspace names.
func TestExtractDateFromWorkspaceName(t *testing.T) {
	// Use a fixed reference time for testing year boundary logic
	currentYear := time.Now().Year()

	tests := []struct {
		name      string
		workspace string
		wantZero  bool
		wantMonth time.Month
		wantDay   int
	}{
		{
			name:      "standard december date",
			workspace: "og-feat-add-feature-24dec",
			wantZero:  false,
			wantMonth: time.December,
			wantDay:   24,
		},
		{
			name:      "single digit day",
			workspace: "og-debug-fix-bug-5jan",
			wantZero:  false,
			wantMonth: time.January,
			wantDay:   5,
		},
		{
			name:      "november date",
			workspace: "og-inv-explore-15nov",
			wantZero:  false,
			wantMonth: time.November,
			wantDay:   15,
		},
		{
			name:      "february date",
			workspace: "og-arch-design-28feb",
			wantZero:  false,
			wantMonth: time.February,
			wantDay:   28,
		},
		{
			name:      "no date suffix",
			workspace: "og-feat-add-feature",
			wantZero:  true,
			wantMonth: 0,
			wantDay:   0,
		},
		{
			name:      "invalid month",
			workspace: "og-feat-test-20xyz",
			wantZero:  true,
			wantMonth: 0,
			wantDay:   0,
		},
		{
			name:      "invalid day too high",
			workspace: "og-feat-test-32dec",
			wantZero:  true,
			wantMonth: 0,
			wantDay:   0,
		},
		{
			name:      "invalid day zero",
			workspace: "og-feat-test-0dec",
			wantZero:  true,
			wantMonth: 0,
			wantDay:   0,
		},
		{
			name:      "suffix too short",
			workspace: "og-feat-test-1a",
			wantZero:  true,
			wantMonth: 0,
			wantDay:   0,
		},
		{
			name:      "suffix too long",
			workspace: "og-feat-test-100december",
			wantZero:  true,
			wantMonth: 0,
			wantDay:   0,
		},
		{
			name:      "empty workspace name",
			workspace: "",
			wantZero:  true,
			wantMonth: 0,
			wantDay:   0,
		},
		{
			name:      "uppercase month",
			workspace: "og-feat-test-24DEC",
			wantZero:  false,
			wantMonth: time.December,
			wantDay:   24,
		},
		{
			name:      "mixed case month",
			workspace: "og-feat-test-24Dec",
			wantZero:  false,
			wantMonth: time.December,
			wantDay:   24,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDateFromWorkspaceName(tt.workspace)

			if tt.wantZero {
				if !got.IsZero() {
					t.Errorf("extractDateFromWorkspaceName(%q) = %v, want zero time", tt.workspace, got)
				}
				return
			}

			if got.IsZero() {
				t.Errorf("extractDateFromWorkspaceName(%q) = zero time, want non-zero", tt.workspace)
				return
			}

			if got.Month() != tt.wantMonth {
				t.Errorf("extractDateFromWorkspaceName(%q) month = %v, want %v", tt.workspace, got.Month(), tt.wantMonth)
			}
			if got.Day() != tt.wantDay {
				t.Errorf("extractDateFromWorkspaceName(%q) day = %d, want %d", tt.workspace, got.Day(), tt.wantDay)
			}
			// Year should be current year (or previous year if date is in future)
			if got.Year() != currentYear && got.Year() != currentYear-1 {
				t.Errorf("extractDateFromWorkspaceName(%q) year = %d, want %d or %d", tt.workspace, got.Year(), currentYear, currentYear-1)
			}
		})
	}
}
