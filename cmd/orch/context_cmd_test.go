package main

import (
	"testing"
)

func TestExtractAdditionalContext(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
		want   string
	}{
		{
			name:   "empty output",
			stdout: "",
			want:   "",
		},
		{
			name:   "hookSpecificOutput format",
			stdout: `{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"# Context\nHello world"}}`,
			want:   "# Context\nHello world",
		},
		{
			name:   "root-level additionalContext",
			stdout: `{"additionalContext":"Some context here"}`,
			want:   "Some context here",
		},
		{
			name:   "plain text output",
			stdout: "Success",
			want:   "Success",
		},
		{
			name:   "JSON without additionalContext",
			stdout: `{"hookSpecificOutput":{"hookEventName":"SessionStart"}}`,
			want:   "",
		},
		{
			name:   "whitespace-only output",
			stdout: "   \n  ",
			want:   "",
		},
		{
			name:   "empty additionalContext",
			stdout: `{"hookSpecificOutput":{"additionalContext":""}}`,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAdditionalContext(tt.stdout)
			if got != tt.want {
				t.Errorf("extractAdditionalContext() = %q, want %q", got, tt.want)
			}
		})
	}
}
