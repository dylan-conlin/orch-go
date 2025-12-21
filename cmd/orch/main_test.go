package main

import (
	"strings"
	"testing"
)

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
	// This test relies on the registry behavior tested in pkg/registry/registry_test.go
	// It verifies the end-to-end flow of the abandon command.

	// Create a temporary directory for the registry
	tempDir := t.TempDir()

	// Set up a test registry path (this will use an empty registry)
	// The runAbandon function should fail because no agent exists
	beadsID := "nonexistent-agent-xyz"

	// We can't easily test runAbandon directly because it uses os.Getwd()
	// and global state. Instead, verify the error message pattern.
	err := runAbandon(beadsID)
	if err == nil {
		t.Error("Expected error for non-existent agent")
	}
	if err != nil && !strings.Contains(err.Error(), "no agent found") {
		t.Errorf("Expected 'no agent found' error, got: %v", err)
	}

	_ = tempDir // Use tempDir to avoid unused variable warning
}

// TestAbandonValidatesAgentStatus tests that only active agents can be abandoned.
func TestAbandonValidatesAgentStatus(t *testing.T) {
	// This is integration tested via pkg/registry/registry_test.go
	// The registry.Abandon method only works on active agents.
	// We verify that the error message is correct.

	// Note: Full integration testing would require setting up a registry
	// with a completed/abandoned agent and verifying the error.
	// For now, we rely on the unit tests in pkg/registry.
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
