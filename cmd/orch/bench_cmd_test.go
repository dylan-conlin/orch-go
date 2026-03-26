package main

import (
	"testing"
)

func TestExtractBeadsID(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "typical spawn output",
			output: "Spawning feature-impl...\nCreated: orch-go-ab12m\nWorkspace: og-feat-something",
			want:   "orch-go-ab12m",
		},
		{
			name:   "beads ID on last line",
			output: "some output\norch-go-xyz99",
			want:   "orch-go-xyz99",
		},
		{
			name:   "no beads ID",
			output: "error: something failed",
			want:   "",
		},
		{
			name:   "beads ID with project prefix",
			output: "Created price-watch-abc12 in ~/price-watch",
			want:   "price-watch-abc12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBeadsID(tt.output)
			if got != tt.want {
				t.Errorf("extractBeadsID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsBeadsID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"orch-go-ab12m", true},
		{"price-watch-xyz99", true},
		{"orch-go-abc", false},        // too short
		{"orch-go-abcdefg", false},    // too long
		{"orch-go-ABC12", false},      // uppercase
		{"ab12m", false},              // no prefix
		{"orch-go", false},            // no suffix
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isBeadsID(tt.input)
			if got != tt.want {
				t.Errorf("isBeadsID(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
