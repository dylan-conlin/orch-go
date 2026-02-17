package orch

import (
	"strings"
	"testing"
)

func TestValidateMode(t *testing.T) {
	tests := []struct {
		name      string
		mode      string
		wantErr   bool
		errSubstr string // expected substring in error message
	}{
		{
			name:    "valid: tdd",
			mode:    "tdd",
			wantErr: false,
		},
		{
			name:    "valid: direct",
			mode:    "direct",
			wantErr: false,
		},
		{
			name:    "valid: verification-first",
			mode:    "verification-first",
			wantErr: false,
		},
		{
			name:      "invalid: claude (backend name)",
			mode:      "claude",
			wantErr:   true,
			errSubstr: "--backend claude",
		},
		{
			name:      "invalid: opencode (backend name)",
			mode:      "opencode",
			wantErr:   true,
			errSubstr: "--backend opencode",
		},
		{
			name:      "invalid: arbitrary string",
			mode:      "foobar",
			wantErr:   true,
			errSubstr: "invalid --mode value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMode(tt.mode)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateMode(%q) = nil, want error", tt.mode)
				} else if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("ValidateMode(%q) error = %q, want substring %q", tt.mode, err.Error(), tt.errSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateMode(%q) = %v, want nil", tt.mode, err)
				}
			}
		})
	}
}

func TestValidateMode_BackendHint(t *testing.T) {
	// Verify the error message for backend names includes helpful guidance
	err := ValidateMode("claude")
	if err == nil {
		t.Fatal("expected error for --mode claude")
	}
	msg := err.Error()
	if !strings.Contains(msg, "is a backend") {
		t.Errorf("error should explain 'claude' is a backend, got: %s", msg)
	}
	if !strings.Contains(msg, "Use --backend claude") {
		t.Errorf("error should suggest --backend claude, got: %s", msg)
	}
	if !strings.Contains(msg, "tdd, direct, verification-first") {
		t.Errorf("error should list valid modes, got: %s", msg)
	}
}
