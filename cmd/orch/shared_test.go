package main

import (
	"testing"
)

func TestShortID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"long ID truncated to 12", "ses_abc12def3456789", "ses_abc12def"},
		{"exactly 12 chars", "abcdefghijkl", "abcdefghijkl"},
		{"short ID returned as-is", "abcd", "abcd"},
		{"empty string", "", ""},
		{"1 char", "x", "x"},
		{"13 chars truncated", "1234567890123", "123456789012"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shortID(tt.input)
			if got != tt.expected {
				t.Errorf("shortID(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

