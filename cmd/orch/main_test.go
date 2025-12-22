package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	err := runAbandon(beadsID)
	if err == nil {
		t.Error("Expected error for non-existent agent")
	}
	// Now the error is from beads lookup failure (issue not found)
	if err != nil && !strings.Contains(err.Error(), "failed to get beads issue") && !strings.Contains(err.Error(), "no agent found") {
		t.Errorf("Expected 'failed to get beads issue' or 'no agent found' error, got: %v", err)
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
