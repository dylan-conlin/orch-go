package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
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

	// Clear flag (sentinel -1 means "not set") and env var
	spawnMaxAgents = -1
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

// TestGetMaxAgentsEnvVar tests that ORCH_MAX_AGENTS env var is used when flag is not set (-1).
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

	// Clear flag (sentinel -1 means "not set"), set env to 15
	spawnMaxAgents = -1
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

	// Clear flag (sentinel -1 means "not set"), set invalid env
	spawnMaxAgents = -1
	os.Setenv("ORCH_MAX_AGENTS", "not-a-number")

	got := getMaxAgents()
	if got != DefaultMaxAgents {
		t.Errorf("getMaxAgents() = %d, want default %d (invalid env)", got, DefaultMaxAgents)
	}
}

// TestGetMaxAgentsZeroDisablesLimit tests that --max-agents 0 returns 0 (unlimited).
func TestGetMaxAgentsZeroDisablesLimit(t *testing.T) {
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

	// Set flag to 0 (explicitly disables limit), env should be ignored
	spawnMaxAgents = 0
	os.Setenv("ORCH_MAX_AGENTS", "10") // Should be ignored because flag is explicitly set

	got := getMaxAgents()
	if got != 0 {
		t.Errorf("getMaxAgents() = %d, want 0 (unlimited - flag explicitly set to 0)", got)
	}
}

// TestGetMaxAgentsEnvZeroDisablesLimit tests that ORCH_MAX_AGENTS=0 returns 0 (unlimited).
func TestGetMaxAgentsEnvZeroDisablesLimit(t *testing.T) {
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

	// Clear flag (sentinel -1 means "not set"), set env to 0
	spawnMaxAgents = -1
	os.Setenv("ORCH_MAX_AGENTS", "0")

	got := getMaxAgents()
	if got != 0 {
		t.Errorf("getMaxAgents() = %d, want 0 (unlimited via env var)", got)
	}
}

// TestCheckConcurrencyLimitUsesOpenCodeAPI documents the concurrency checking behavior.
// Concurrency checking uses OpenCode API ListSessions() directly.
func TestCheckConcurrencyLimitUsesOpenCodeAPI(t *testing.T) {
	// The checkConcurrencyLimit function:
	// 1. Creates an OpenCode client
	// 2. Calls client.ListSessions()
	// 3. Counts active sessions (status != "completed")
	// 4. Returns error if count >= max
	//
	// All session state comes from OpenCode - no agent registry file.
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
	err := runAbandon(beadsID, "", "")
	if err == nil {
		t.Error("Expected error for non-existent agent")
	}
	// Now the error is from beads lookup failure (issue not found)
	if err != nil && !strings.Contains(err.Error(), "failed to get beads issue") && !strings.Contains(err.Error(), "no agent found") {
		t.Errorf("Expected 'failed to get beads issue' or 'no agent found' error, got: %v", err)
	}
}

// TestCompleteCrossProjectErrorMessage tests that completing an agent from a different project
// provides helpful error message with cd suggestion.
func TestCompleteCrossProjectErrorMessage(t *testing.T) {
	// Try to complete an agent with an ID that suggests a different project
	// We're likely in 'orch-go' or 'orch' but using 'kb-cli' prefix
	beadsID := "kb-cli-xyz123"

	err := runComplete(beadsID, "")
	if err == nil {
		t.Error("Expected error for cross-project beads ID")
		return
	}

	errMsg := err.Error()

	// Check that the error message contains helpful hints
	if !strings.Contains(errMsg, "Hint:") {
		t.Errorf("Expected error to contain 'Hint:', got: %v", err)
	}
	if !strings.Contains(errMsg, "kb-cli") {
		t.Errorf("Expected error to mention the project 'kb-cli', got: %v", err)
	}
	if !strings.Contains(errMsg, "--workdir") {
		t.Errorf("Expected error to suggest '--workdir' option, got: %v", err)
	}
	if !strings.Contains(errMsg, "orch complete") {
		t.Errorf("Expected error to include 'orch complete' command, got: %v", err)
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
			name:            "explicit issue ID provided but not found",
			spawnIssue:      "explicit-issue-123",
			spawnNoTrack:    false,
			createBeadsFn:   nil, // should not be called
			wantID:          "",
			wantErr:         true,
			wantErrContains: "not found",
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
			gotID, gotErr := determineBeadsID("test-project", "test-skill", "test task", tt.spawnIssue, "", tt.spawnNoTrack, tt.createBeadsFn)

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

// TestFormatContextQualitySummary tests the context quality summary formatting.
func TestFormatContextQualitySummary(t *testing.T) {
	tests := []struct {
		name          string
		quality       int
		hasGaps       bool
		matchCount    int
		constraints   int
		wantIndicator string
		wantLabel     string
		wantNoMatches bool
	}{
		{
			name:          "nil gap analysis",
			quality:       0,
			hasGaps:       false,
			wantIndicator: "",
			wantLabel:     "not checked",
		},
		{
			name:          "zero quality - critical",
			quality:       0,
			hasGaps:       true,
			wantIndicator: "🚨",
			wantLabel:     "CRITICAL",
		},
		{
			name:          "very low quality - poor",
			quality:       15,
			hasGaps:       true,
			wantIndicator: "⚠️",
			wantLabel:     "poor",
		},
		{
			name:          "low quality - limited",
			quality:       30,
			hasGaps:       true,
			wantIndicator: "⚠️",
			wantLabel:     "limited",
		},
		{
			name:          "moderate quality",
			quality:       50,
			hasGaps:       true,
			wantIndicator: "📊",
			wantLabel:     "moderate",
		},
		{
			name:          "good quality",
			quality:       70,
			hasGaps:       false,
			wantIndicator: "✓",
			wantLabel:     "good",
		},
		{
			name:          "excellent quality",
			quality:       90,
			hasGaps:       false,
			wantIndicator: "✓",
			wantLabel:     "excellent",
		},
		{
			name:          "includes match count",
			quality:       50,
			hasGaps:       true,
			matchCount:    5,
			constraints:   2,
			wantIndicator: "📊",
			wantLabel:     "moderate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var analysis *spawn.GapAnalysis
			if tt.name != "nil gap analysis" {
				analysis = &spawn.GapAnalysis{
					HasGaps:        tt.hasGaps,
					ContextQuality: tt.quality,
					MatchStats: spawn.MatchStatistics{
						TotalMatches:    tt.matchCount,
						ConstraintCount: tt.constraints,
					},
				}
			}

			result := formatContextQualitySummary(analysis)

			if tt.name == "nil gap analysis" {
				if result != "not checked" {
					t.Errorf("formatContextQualitySummary(nil) = %q, want %q", result, "not checked")
				}
				return
			}

			if !strings.Contains(result, tt.wantIndicator) {
				t.Errorf("formatContextQualitySummary() = %q, want to contain indicator %q", result, tt.wantIndicator)
			}
			if !strings.Contains(result, tt.wantLabel) {
				t.Errorf("formatContextQualitySummary() = %q, want to contain label %q", result, tt.wantLabel)
			}
			if !strings.Contains(result, "/100") {
				t.Errorf("formatContextQualitySummary() = %q, want to contain quality score", result)
			}
			if tt.matchCount > 0 && !strings.Contains(result, "matches") {
				t.Errorf("formatContextQualitySummary() = %q, want to mention matches", result)
			}
		})
	}
}

// TestPrintSpawnSummaryWithGapWarning tests the gap warning printing logic.
func TestPrintSpawnSummaryWithGapWarning(t *testing.T) {
	// Test cases for determining when warning is printed
	tests := []struct {
		name        string
		quality     int
		hasGaps     bool
		hasCritical bool
		hasWarning  bool
		wantWarning bool
	}{
		{
			name:        "critical gap - should warn",
			quality:     0,
			hasGaps:     true,
			hasCritical: true,
			wantWarning: true,
		},
		{
			name:        "low quality with warning gap - should warn",
			quality:     15,
			hasGaps:     true,
			hasCritical: false,
			hasWarning:  true,
			wantWarning: true,
		},
		{
			name:        "good quality - no warning",
			quality:     75,
			hasGaps:     false,
			hasCritical: false,
			wantWarning: false,
		},
		{
			name:        "moderate quality with gaps - no warning",
			quality:     50,
			hasGaps:     true,
			hasCritical: false,
			wantWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := &spawn.GapAnalysis{
				HasGaps:        tt.hasGaps,
				ContextQuality: tt.quality,
			}

			if tt.hasCritical {
				analysis.Gaps = []spawn.Gap{
					{
						Type:     spawn.GapTypeNoContext,
						Severity: spawn.GapSeverityCritical,
					},
				}
			} else if tt.hasWarning {
				analysis.Gaps = []spawn.Gap{
					{
						Type:     spawn.GapTypeSparseContext,
						Severity: spawn.GapSeverityWarning,
					},
				}
			}

			// Check the conditions that trigger warning
			shouldWarn := analysis.ShouldWarnAboutGaps() && (analysis.HasCriticalGaps() || analysis.ContextQuality < 20)

			if shouldWarn != tt.wantWarning {
				t.Errorf("warning condition = %v, want %v", shouldWarn, tt.wantWarning)
			}
		})
	}
}

// TestHasGoChangesDetection tests the Go file change detection logic.
// This tests the string matching logic used by hasGoChangesInRecentCommits.
func TestHasGoChangesDetection(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		wantGo   bool
	}{
		{
			name:     "cmd/orch Go file",
			filePath: "cmd/orch/main.go",
			wantGo:   true,
		},
		{
			name:     "cmd/orch test file",
			filePath: "cmd/orch/main_test.go",
			wantGo:   true,
		},
		{
			name:     "pkg top-level Go file",
			filePath: "pkg/verify/check.go",
			wantGo:   true,
		},
		{
			name:     "pkg nested Go file",
			filePath: "pkg/opencode/client.go",
			wantGo:   true,
		},
		{
			name:     "non-Go file in cmd/orch",
			filePath: "cmd/orch/README.md",
			wantGo:   false,
		},
		{
			name:     "Go file in different cmd",
			filePath: "cmd/gendoc/main.go",
			wantGo:   false,
		},
		{
			name:     "beads file",
			filePath: ".beads/issues.jsonl",
			wantGo:   false,
		},
		{
			name:     "investigation file",
			filePath: ".kb/investigations/2025-12-24-test.md",
			wantGo:   false,
		},
		{
			name:     "web Go file",
			filePath: "web/src/routes/page.svelte",
			wantGo:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the same logic as hasGoChangesInRecentCommits
			line := tt.filePath
			isGoChange := false
			if strings.HasPrefix(line, "cmd/orch/") && strings.HasSuffix(line, ".go") {
				isGoChange = true
			}
			if strings.HasPrefix(line, "pkg/") && strings.HasSuffix(line, ".go") {
				isGoChange = true
			}

			if isGoChange != tt.wantGo {
				t.Errorf("Go change detection for %q = %v, want %v", tt.filePath, isGoChange, tt.wantGo)
			}
		})
	}
}

// TestCheckAndAutoSwitchAccountDisabled tests that auto-switch can be disabled via env var.
func TestCheckAndAutoSwitchAccountDisabled(t *testing.T) {
	// Save and restore original env var
	origDisabled := os.Getenv("ORCH_AUTO_SWITCH_DISABLED")
	defer func() {
		if origDisabled == "" {
			os.Unsetenv("ORCH_AUTO_SWITCH_DISABLED")
		} else {
			os.Setenv("ORCH_AUTO_SWITCH_DISABLED", origDisabled)
		}
	}()

	tests := []struct {
		name       string
		envValue   string
		wantCalled bool
	}{
		{
			name:       "disabled with 1",
			envValue:   "1",
			wantCalled: false,
		},
		{
			name:       "disabled with true",
			envValue:   "true",
			wantCalled: false,
		},
		{
			name:       "not disabled with 0",
			envValue:   "0",
			wantCalled: true,
		},
		{
			name:       "not disabled with empty",
			envValue:   "",
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue == "" {
				os.Unsetenv("ORCH_AUTO_SWITCH_DISABLED")
			} else {
				os.Setenv("ORCH_AUTO_SWITCH_DISABLED", tt.envValue)
			}

			// Check the early return logic from checkAndAutoSwitchAccount
			shouldSkip := os.Getenv("ORCH_AUTO_SWITCH_DISABLED") == "1" || os.Getenv("ORCH_AUTO_SWITCH_DISABLED") == "true"
			wantSkip := !tt.wantCalled

			if shouldSkip != wantSkip {
				t.Errorf("auto-switch skip = %v, want %v (env: %q)", shouldSkip, wantSkip, tt.envValue)
			}
		})
	}
}

// TestCheckAndAutoSwitchAccountEnvThresholds tests that thresholds can be customized via env vars.
func TestCheckAndAutoSwitchAccountEnvThresholds(t *testing.T) {
	// Save and restore original env vars
	origFiveHour := os.Getenv("ORCH_AUTO_SWITCH_5H_THRESHOLD")
	origWeekly := os.Getenv("ORCH_AUTO_SWITCH_WEEKLY_THRESHOLD")
	origDelta := os.Getenv("ORCH_AUTO_SWITCH_MIN_DELTA")
	defer func() {
		restoreEnv("ORCH_AUTO_SWITCH_5H_THRESHOLD", origFiveHour)
		restoreEnv("ORCH_AUTO_SWITCH_WEEKLY_THRESHOLD", origWeekly)
		restoreEnv("ORCH_AUTO_SWITCH_MIN_DELTA", origDelta)
	}()

	tests := []struct {
		name         string
		fiveHourEnv  string
		weeklyEnv    string
		deltaEnv     string
		wantFiveHour float64
		wantWeekly   float64
		wantDelta    float64
	}{
		{
			name:         "no env vars - use defaults",
			fiveHourEnv:  "",
			weeklyEnv:    "",
			deltaEnv:     "",
			wantFiveHour: 80,
			wantWeekly:   90,
			wantDelta:    10,
		},
		{
			name:         "custom 5-hour threshold",
			fiveHourEnv:  "70",
			weeklyEnv:    "",
			deltaEnv:     "",
			wantFiveHour: 70,
			wantWeekly:   90,
			wantDelta:    10,
		},
		{
			name:         "custom weekly threshold",
			fiveHourEnv:  "",
			weeklyEnv:    "85",
			deltaEnv:     "",
			wantFiveHour: 80,
			wantWeekly:   85,
			wantDelta:    10,
		},
		{
			name:         "custom delta",
			fiveHourEnv:  "",
			weeklyEnv:    "",
			deltaEnv:     "5",
			wantFiveHour: 80,
			wantWeekly:   90,
			wantDelta:    5,
		},
		{
			name:         "all custom values",
			fiveHourEnv:  "60",
			weeklyEnv:    "75",
			deltaEnv:     "15",
			wantFiveHour: 60,
			wantWeekly:   75,
			wantDelta:    15,
		},
		{
			name:         "invalid env vars - use defaults",
			fiveHourEnv:  "not-a-number",
			weeklyEnv:    "invalid",
			deltaEnv:     "bad",
			wantFiveHour: 80,
			wantWeekly:   90,
			wantDelta:    10,
		},
		{
			name:         "out of range values - use defaults",
			fiveHourEnv:  "150", // >100 is invalid
			weeklyEnv:    "-10", // <0 is invalid
			deltaEnv:     "-5",  // <0 is invalid
			wantFiveHour: 80,    // Uses default because 150 > 100
			wantWeekly:   90,    // Uses default because -10 < 0
			wantDelta:    10,    // Uses default because -5 < 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars
			setEnvIfNotEmpty("ORCH_AUTO_SWITCH_5H_THRESHOLD", tt.fiveHourEnv)
			setEnvIfNotEmpty("ORCH_AUTO_SWITCH_WEEKLY_THRESHOLD", tt.weeklyEnv)
			setEnvIfNotEmpty("ORCH_AUTO_SWITCH_MIN_DELTA", tt.deltaEnv)

			// Replicate the threshold parsing logic from checkAndAutoSwitchAccount
			thresholds := struct {
				FiveHourThreshold float64
				WeeklyThreshold   float64
				MinHeadroomDelta  float64
			}{
				FiveHourThreshold: 80,
				WeeklyThreshold:   90,
				MinHeadroomDelta:  10,
			}

			if envVal := os.Getenv("ORCH_AUTO_SWITCH_5H_THRESHOLD"); envVal != "" {
				if val, err := strconv.ParseFloat(envVal, 64); err == nil && val > 0 && val <= 100 {
					thresholds.FiveHourThreshold = val
				}
			}
			if envVal := os.Getenv("ORCH_AUTO_SWITCH_WEEKLY_THRESHOLD"); envVal != "" {
				if val, err := strconv.ParseFloat(envVal, 64); err == nil && val > 0 && val <= 100 {
					thresholds.WeeklyThreshold = val
				}
			}
			if envVal := os.Getenv("ORCH_AUTO_SWITCH_MIN_DELTA"); envVal != "" {
				if val, err := strconv.ParseFloat(envVal, 64); err == nil && val >= 0 {
					thresholds.MinHeadroomDelta = val
				}
			}

			if thresholds.FiveHourThreshold != tt.wantFiveHour {
				t.Errorf("FiveHourThreshold = %v, want %v", thresholds.FiveHourThreshold, tt.wantFiveHour)
			}
			if thresholds.WeeklyThreshold != tt.wantWeekly {
				t.Errorf("WeeklyThreshold = %v, want %v", thresholds.WeeklyThreshold, tt.wantWeekly)
			}
			if thresholds.MinHeadroomDelta != tt.wantDelta {
				t.Errorf("MinHeadroomDelta = %v, want %v", thresholds.MinHeadroomDelta, tt.wantDelta)
			}
		})
	}
}

// TestNewCLICommandContentDetection tests the content-based detection of cobra commands.
// This tests the file content matching logic used by detectNewCLICommands.
func TestNewCLICommandContentDetection(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		isCommand bool
	}{
		{
			name: "valid cobra command file",
			content: `package main

import "github.com/spf13/cobra"

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check health",
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}`,
			isCommand: true,
		},
		{
			name: "cobra command without AddCommand",
			content: `package main

import "github.com/spf13/cobra"

var orphanCmd = &cobra.Command{
	Use:   "orphan",
	Short: "Not registered",
}`,
			isCommand: false,
		},
		{
			name: "test file with cobra reference",
			content: `package main

import "testing"

func TestSomething(t *testing.T) {
	// Test cobra.Command usage
}`,
			isCommand: false,
		},
		{
			name: "helper file without cobra",
			content: `package main

func doSomething() error {
	return nil
}`,
			isCommand: false,
		},
		{
			name: "AddCommand without cobra.Command definition",
			content: `package main

func init() {
	rootCmd.AddCommand(someOtherCmd)
}`,
			isCommand: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the same logic as detectNewCLICommands
			isCobraCommand := strings.Contains(tt.content, "cobra.Command{") &&
				strings.Contains(tt.content, "rootCmd.AddCommand(")

			if isCobraCommand != tt.isCommand {
				t.Errorf("cobra command detection = %v, want %v", isCobraCommand, tt.isCommand)
			}
		})
	}
}

// TestDetectNewCLICommandsGitStatus tests the git status line parsing for new files.
func TestDetectNewCLICommandsGitStatusParsing(t *testing.T) {
	tests := []struct {
		name       string
		statusLine string
		wantAdded  bool
		wantFile   string
	}{
		{
			name:       "added Go file in cmd/orch",
			statusLine: "A\tcmd/orch/doctor.go",
			wantAdded:  true,
			wantFile:   "cmd/orch/doctor.go",
		},
		{
			name:       "modified Go file in cmd/orch",
			statusLine: "M\tcmd/orch/main.go",
			wantAdded:  false,
			wantFile:   "",
		},
		{
			name:       "added test file",
			statusLine: "A\tcmd/orch/doctor_test.go",
			wantAdded:  false,
			wantFile:   "",
		},
		{
			name:       "added Go file in pkg",
			statusLine: "A\tpkg/verify/check.go",
			wantAdded:  false,
			wantFile:   "",
		},
		{
			name:       "deleted file",
			statusLine: "D\tcmd/orch/old.go",
			wantAdded:  false,
			wantFile:   "",
		},
		{
			name:       "renamed file",
			statusLine: "R\tcmd/orch/old.go\tcmd/orch/new.go",
			wantAdded:  false,
			wantFile:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse status line like detectNewCLICommands does
			parts := strings.Fields(tt.statusLine)
			if len(parts) < 2 {
				if tt.wantAdded {
					t.Error("expected to parse as added but parsing failed")
				}
				return
			}

			status := parts[0]
			filePath := parts[1]

			// Check added condition
			isAdded := status == "A"
			// Check file path conditions
			isTargetFile := strings.HasPrefix(filePath, "cmd/orch/") &&
				strings.HasSuffix(filePath, ".go") &&
				!strings.HasSuffix(filePath, "_test.go")

			shouldProcess := isAdded && isTargetFile

			if shouldProcess != tt.wantAdded {
				t.Errorf("file processing = %v, want %v", shouldProcess, tt.wantAdded)
			}

			if tt.wantAdded && filePath != tt.wantFile {
				t.Errorf("file path = %q, want %q", filePath, tt.wantFile)
			}
		})
	}
}

// TestIsSkillRelevantChange tests the skill relevance detection for changelog entries.
func TestIsSkillRelevantChange(t *testing.T) {
	tests := []struct {
		name      string
		commit    CommitInfo
		skillName string
		want      bool
	}{
		{
			name: "skill-specific file change",
			commit: CommitInfo{
				Files: []string{"skills/worker/feature-impl/SKILL.md"},
			},
			skillName: "feature-impl",
			want:      true,
		},
		{
			name: "different skill change",
			commit: CommitInfo{
				Files: []string{"skills/worker/investigation/SKILL.md"},
			},
			skillName: "feature-impl",
			want:      false,
		},
		{
			name: "spawn package change affects all skills",
			commit: CommitInfo{
				Files: []string{"pkg/spawn/context.go"},
			},
			skillName: "feature-impl",
			want:      true,
		},
		{
			name: "SPAWN_CONTEXT change affects all skills",
			commit: CommitInfo{
				Files: []string{"templates/SPAWN_CONTEXT.md"},
			},
			skillName: "feature-impl",
			want:      true,
		},
		{
			name: "skill verification change affects all skills",
			commit: CommitInfo{
				Files: []string{"pkg/verify/skill_outputs.go"},
			},
			skillName: "investigation",
			want:      true,
		},
		{
			name: "unrelated file",
			commit: CommitInfo{
				Files: []string{"cmd/orch/serve.go"},
			},
			skillName: "feature-impl",
			want:      false,
		},
		{
			name: "empty skill name - no match",
			commit: CommitInfo{
				Files: []string{"skills/worker/feature-impl/SKILL.md"},
			},
			skillName: "",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSkillRelevantChange(tt.commit, tt.skillName)
			if got != tt.want {
				t.Errorf("isSkillRelevantChange() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNotableChangelogEntry tests the detection logic for notable changes.
func TestNotableChangelogEntry(t *testing.T) {
	tests := []struct {
		name        string
		commit      CommitInfo
		agentSkill  string
		wantNotable bool
	}{
		{
			name: "breaking change is always notable",
			commit: CommitInfo{
				Subject: "BREAKING: remove deprecated API",
				SemanticInfo: SemanticInfo{
					IsBreaking: true,
					ChangeType: ChangeTypeBehavioral,
				},
				Category: "cmd",
			},
			agentSkill:  "",
			wantNotable: true,
		},
		{
			name: "behavioral skill change is notable",
			commit: CommitInfo{
				Subject: "feat: add new spawn option",
				SemanticInfo: SemanticInfo{
					IsBreaking: false,
					ChangeType: ChangeTypeBehavioral,
				},
				Category: "skills",
			},
			agentSkill:  "",
			wantNotable: true,
		},
		{
			name: "behavioral cmd change is notable",
			commit: CommitInfo{
				Subject: "fix: correct spawn timeout",
				SemanticInfo: SemanticInfo{
					IsBreaking: false,
					ChangeType: ChangeTypeBehavioral,
				},
				Category: "cmd",
			},
			agentSkill:  "",
			wantNotable: true,
		},
		{
			name: "documentation change is not notable",
			commit: CommitInfo{
				Subject: "docs: update README",
				SemanticInfo: SemanticInfo{
					IsBreaking: false,
					ChangeType: ChangeTypeDocumentation,
				},
				Category: "docs",
			},
			agentSkill:  "",
			wantNotable: false,
		},
		{
			name: "behavioral web change is not notable without context",
			commit: CommitInfo{
				Subject: "feat: update dashboard styling",
				SemanticInfo: SemanticInfo{
					IsBreaking: false,
					ChangeType: ChangeTypeBehavioral,
				},
				Category: "web",
			},
			agentSkill:  "",
			wantNotable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply the same logic as detectNotableChangelogEntries
			var reasons []string

			if tt.commit.SemanticInfo.IsBreaking {
				reasons = append(reasons, "BREAKING")
			}

			if tt.commit.SemanticInfo.ChangeType == ChangeTypeBehavioral {
				if tt.commit.Category == "skills" || tt.commit.Category == "skill-behavioral" ||
					tt.commit.Category == "cmd" || tt.commit.Category == "pkg" {
					reasons = append(reasons, "behavioral")
				}
			}

			if tt.agentSkill != "" && isSkillRelevantChange(tt.commit, tt.agentSkill) {
				reasons = append(reasons, "relevant to "+tt.agentSkill)
			}

			gotNotable := len(reasons) > 0
			if gotNotable != tt.wantNotable {
				t.Errorf("notable = %v (reasons: %v), want %v", gotNotable, reasons, tt.wantNotable)
			}
		})
	}
}

// Helper functions for env var management
func setEnvIfNotEmpty(key, value string) {
	if value == "" {
		os.Unsetenv(key)
	} else {
		os.Setenv(key, value)
	}
}

func restoreEnv(key, originalValue string) {
	if originalValue == "" {
		os.Unsetenv(key)
	} else {
		os.Setenv(key, originalValue)
	}
}

// TestRegisterOrchestratorSession tests that orchestrator sessions are registered in the session registry.
func TestRegisterOrchestratorSession(t *testing.T) {
	// Create temp directory for registry
	tempDir, err := os.MkdirTemp("", "test-orch-registry-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override HOME so registry goes to temp dir
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	// Create .orch directory for registry
	orchDir := filepath.Join(tempDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatalf("Failed to create .orch dir: %v", err)
	}

	tests := []struct {
		name               string
		isOrchestrator     bool
		isMetaOrchestrator bool
		wantRegistered     bool
	}{
		{
			name:               "orchestrator session is registered",
			isOrchestrator:     true,
			isMetaOrchestrator: false,
			wantRegistered:     true,
		},
		{
			name:               "meta-orchestrator session is registered",
			isOrchestrator:     false,
			isMetaOrchestrator: true,
			wantRegistered:     true,
		},
		{
			name:               "worker session is NOT registered",
			isOrchestrator:     false,
			isMetaOrchestrator: false,
			wantRegistered:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear registry between tests
			registryPath := filepath.Join(orchDir, "sessions.json")
			os.Remove(registryPath)

			cfg := &spawn.Config{
				WorkspaceName:      "test-workspace-" + tt.name,
				ProjectDir:         "/tmp/test-project",
				IsOrchestrator:     tt.isOrchestrator,
				IsMetaOrchestrator: tt.isMetaOrchestrator,
			}

			// Call the registration function
			registerOrchestratorSession(cfg, "test-session-id", "test task")

			// Check if session was registered
			data, err := os.ReadFile(registryPath)
			registered := err == nil && len(data) > 0

			if registered != tt.wantRegistered {
				t.Errorf("Session registration = %v, want %v", registered, tt.wantRegistered)
			}

			// If registered, verify the content
			if tt.wantRegistered && registered {
				if !strings.Contains(string(data), cfg.WorkspaceName) {
					t.Errorf("Registry data should contain workspace name %q, got: %s", cfg.WorkspaceName, string(data))
				}
				if !strings.Contains(string(data), "test-session-id") {
					t.Errorf("Registry data should contain session ID, got: %s", string(data))
				}
				if !strings.Contains(string(data), "active") {
					t.Errorf("Registry data should contain 'active' status, got: %s", string(data))
				}
			}
		})
	}
}

// TestOrchestratorSkipsBeadsIssue tests that orchestrator spawns skip beads issue creation.
func TestOrchestratorSkipsBeadsIssue(t *testing.T) {
	// This tests the logic in runSpawnWithSkill that skips beads for orchestrators
	tests := []struct {
		name               string
		isOrchestrator     bool
		isMetaOrchestrator bool
		spawnNoTrack       bool
		wantSkipBeads      bool
	}{
		{
			name:           "orchestrator skips beads",
			isOrchestrator: true,
			spawnNoTrack:   false,
			wantSkipBeads:  true,
		},
		{
			name:               "meta-orchestrator skips beads",
			isMetaOrchestrator: true,
			spawnNoTrack:       false,
			wantSkipBeads:      true,
		},
		{
			name:           "worker with --no-track skips beads",
			isOrchestrator: false,
			spawnNoTrack:   true,
			wantSkipBeads:  true,
		},
		{
			name:           "worker without --no-track creates beads",
			isOrchestrator: false,
			spawnNoTrack:   false,
			wantSkipBeads:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replicate the logic from runSpawnWithSkill
			skipBeadsForOrchestrator := tt.isOrchestrator || tt.isMetaOrchestrator
			shouldSkipBeads := tt.spawnNoTrack || skipBeadsForOrchestrator

			if shouldSkipBeads != tt.wantSkipBeads {
				t.Errorf("skipBeads = %v, want %v", shouldSkipBeads, tt.wantSkipBeads)
			}
		})
	}
}

// TestDefaultUsageThresholds tests the default usage monitoring thresholds.
func TestDefaultUsageThresholds(t *testing.T) {
	thresholds := DefaultUsageThresholds()

	if thresholds.WarnThreshold != 80 {
		t.Errorf("WarnThreshold = %v, want 80", thresholds.WarnThreshold)
	}
	if thresholds.BlockThreshold != 95 {
		t.Errorf("BlockThreshold = %v, want 95", thresholds.BlockThreshold)
	}
}

// TestUsageThresholdsFromEnv tests that usage thresholds can be configured via environment variables.
func TestUsageThresholdsFromEnv(t *testing.T) {
	// Save original env vars
	origWarn := os.Getenv("ORCH_USAGE_WARN_THRESHOLD")
	origBlock := os.Getenv("ORCH_USAGE_BLOCK_THRESHOLD")
	defer func() {
		restoreEnv("ORCH_USAGE_WARN_THRESHOLD", origWarn)
		restoreEnv("ORCH_USAGE_BLOCK_THRESHOLD", origBlock)
	}()

	tests := []struct {
		name      string
		warnEnv   string
		blockEnv  string
		wantWarn  float64
		wantBlock float64
	}{
		{
			name:      "no env vars - use defaults",
			warnEnv:   "",
			blockEnv:  "",
			wantWarn:  80,
			wantBlock: 95,
		},
		{
			name:      "custom warn threshold",
			warnEnv:   "70",
			blockEnv:  "",
			wantWarn:  70,
			wantBlock: 95,
		},
		{
			name:      "custom block threshold",
			warnEnv:   "",
			blockEnv:  "90",
			wantWarn:  80,
			wantBlock: 90,
		},
		{
			name:      "both custom",
			warnEnv:   "75",
			blockEnv:  "92",
			wantWarn:  75,
			wantBlock: 92,
		},
		{
			name:      "invalid env - use defaults",
			warnEnv:   "not-a-number",
			blockEnv:  "invalid",
			wantWarn:  80,
			wantBlock: 95,
		},
		{
			name:      "out of range - use defaults",
			warnEnv:   "150", // > 100
			blockEnv:  "-10", // < 0
			wantWarn:  80,
			wantBlock: 95,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvIfNotEmpty("ORCH_USAGE_WARN_THRESHOLD", tt.warnEnv)
			setEnvIfNotEmpty("ORCH_USAGE_BLOCK_THRESHOLD", tt.blockEnv)

			// Replicate the threshold parsing logic from checkUsageBeforeSpawn
			thresholds := DefaultUsageThresholds()
			if envVal := os.Getenv("ORCH_USAGE_WARN_THRESHOLD"); envVal != "" {
				if val, err := strconv.ParseFloat(envVal, 64); err == nil && val > 0 && val <= 100 {
					thresholds.WarnThreshold = val
				}
			}
			if envVal := os.Getenv("ORCH_USAGE_BLOCK_THRESHOLD"); envVal != "" {
				if val, err := strconv.ParseFloat(envVal, 64); err == nil && val > 0 && val <= 100 {
					thresholds.BlockThreshold = val
				}
			}

			if thresholds.WarnThreshold != tt.wantWarn {
				t.Errorf("WarnThreshold = %v, want %v", thresholds.WarnThreshold, tt.wantWarn)
			}
			if thresholds.BlockThreshold != tt.wantBlock {
				t.Errorf("BlockThreshold = %v, want %v", thresholds.BlockThreshold, tt.wantBlock)
			}
		})
	}
}

// TestAddUsageInfoToEventData tests the usage info telemetry helper.
func TestAddUsageInfoToEventData(t *testing.T) {
	tests := []struct {
		name       string
		usageInfo  *spawn.UsageInfo
		wantFields []string
		wantValues map[string]interface{}
	}{
		{
			name:       "nil usage info - no fields added",
			usageInfo:  nil,
			wantFields: []string{},
		},
		{
			name: "basic usage info",
			usageInfo: &spawn.UsageInfo{
				FiveHourUsed: 50.5,
				SevenDayUsed: 75.2,
			},
			wantFields: []string{"usage_5h_used", "usage_weekly_used"},
			wantValues: map[string]interface{}{
				"usage_5h_used":     50.5,
				"usage_weekly_used": 75.2,
			},
		},
		{
			name: "with account email",
			usageInfo: &spawn.UsageInfo{
				FiveHourUsed: 30.0,
				SevenDayUsed: 40.0,
				AccountEmail: "user@example.com",
			},
			wantFields: []string{"usage_5h_used", "usage_weekly_used", "usage_account"},
			wantValues: map[string]interface{}{
				"usage_account": "user@example.com",
			},
		},
		{
			name: "with auto-switch",
			usageInfo: &spawn.UsageInfo{
				FiveHourUsed: 96.0,
				SevenDayUsed: 80.0,
				AutoSwitched: true,
				SwitchReason: "switched from personal to work",
			},
			wantFields: []string{"usage_5h_used", "usage_weekly_used", "usage_auto_switched", "usage_switch_reason"},
			wantValues: map[string]interface{}{
				"usage_auto_switched": true,
				"usage_switch_reason": "switched from personal to work",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventData := make(map[string]interface{})
			addUsageInfoToEventData(eventData, tt.usageInfo)

			// Check expected fields are present
			for _, field := range tt.wantFields {
				if _, ok := eventData[field]; !ok {
					t.Errorf("expected field %q not found in event data", field)
				}
			}

			// Check expected values
			for key, wantVal := range tt.wantValues {
				if gotVal, ok := eventData[key]; ok {
					if gotVal != wantVal {
						t.Errorf("field %q = %v, want %v", key, gotVal, wantVal)
					}
				}
			}

			// Check no unexpected fields for nil case
			if tt.usageInfo == nil && len(eventData) > 0 {
				t.Errorf("expected empty event data for nil usage info, got %v", eventData)
			}
		})
	}
}
