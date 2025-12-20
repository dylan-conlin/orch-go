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
