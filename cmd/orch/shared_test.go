package main

import (
	"testing"
)

func TestFormatBeadsIDForDisplay(t *testing.T) {
	// Note: These tests use specific timestamps and expect local timezone conversion
	// Timestamp 1768090360 = Sat Jan 10 16:12:40 PST 2026
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "regular beads ID unchanged",
			input:    "orch-go-abc123",
			expected: "orch-go-abc123",
		},
		{
			name:     "untracked ID with valid timestamp",
			input:    "orch-go-untracked-1768090360",
			expected: "untracked-Jan10-1612", // Jan 10, 2026 16:12 PST
		},
		{
			name:     "untracked ID with different project",
			input:    "my-project-untracked-1768090360",
			expected: "untracked-Jan10-1612",
		},
		{
			name:     "malformed untracked ID (too few parts)",
			input:    "untracked-123",
			expected: "untracked-123", // Should pass through unchanged
		},
		{
			name:     "untracked ID with non-numeric timestamp",
			input:    "orch-go-untracked-notanumber",
			expected: "orch-go-untracked-notanumber", // Should pass through unchanged
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "untracked with Unix epoch (timestamp 0)",
			input:    "test-untracked-0",
			expected: "untracked-Dec31-1600", // Dec 31, 1969 16:00 PST (epoch in PST)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBeadsIDForDisplay(tt.input)
			if got != tt.expected {
				t.Errorf("formatBeadsIDForDisplay(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsUntrackedBeadsID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid untracked ID",
			input:    "orch-go-untracked-1768090360",
			expected: true,
		},
		{
			name:     "regular beads ID",
			input:    "orch-go-abc123",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "contains word untracked in task name",
			input:    "orch-go-fix-untracked-bug-abc123",
			expected: true, // This is a limitation - it matches any ID containing "-untracked-"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUntrackedBeadsID(tt.input)
			if got != tt.expected {
				t.Errorf("isUntrackedBeadsID(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
