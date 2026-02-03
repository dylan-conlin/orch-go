package main

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{
			name:     "days suffix",
			input:    "7d",
			expected: 7 * 24 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "single day",
			input:    "1d",
			expected: 24 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "30 days",
			input:    "30d",
			expected: 30 * 24 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "hours suffix",
			input:    "168h",
			expected: 168 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "minutes suffix",
			input:    "60m",
			expected: 60 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "seconds suffix",
			input:    "3600s",
			expected: 3600 * time.Second,
			wantErr:  false,
		},
		{
			name:     "invalid format",
			input:    "abc",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "invalid days format",
			input:    "xd",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
