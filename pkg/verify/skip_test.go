package verify

import (
	"testing"
)

// TestSkipConfigHasAnySkip tests the HasAnySkip method.
func TestSkipConfigHasAnySkip(t *testing.T) {
	tests := []struct {
		name   string
		config SkipConfig
		want   bool
	}{
		{
			name:   "empty config",
			config: SkipConfig{},
			want:   false,
		},
		{
			name:   "only reason set",
			config: SkipConfig{Reason: "some reason"},
			want:   false,
		},
		{
			name:   "test evidence skip",
			config: SkipConfig{TestEvidence: true, Reason: "test reason"},
			want:   true,
		},
		{
			name:   "visual skip",
			config: SkipConfig{Visual: true, Reason: "test reason"},
			want:   true,
		},
		{
			name:   "git diff skip",
			config: SkipConfig{GitDiff: true, Reason: "test reason"},
			want:   true,
		},
		{
			name:   "synthesis skip",
			config: SkipConfig{Synthesis: true, Reason: "test reason"},
			want:   true,
		},
		{
			name:   "build skip",
			config: SkipConfig{Build: true, Reason: "test reason"},
			want:   true,
		},
		{
			name:   "multiple skips",
			config: SkipConfig{TestEvidence: true, GitDiff: true, Reason: "test"},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.HasAnySkip()
			if got != tt.want {
				t.Errorf("HasAnySkip() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSkipConfigSkippedGates tests the SkippedGates method.
func TestSkipConfigSkippedGates(t *testing.T) {
	tests := []struct {
		name   string
		config SkipConfig
		want   []string
	}{
		{
			name:   "empty config",
			config: SkipConfig{},
			want:   nil,
		},
		{
			name:   "single skip - test evidence",
			config: SkipConfig{TestEvidence: true},
			want:   []string{"test_evidence"},
		},
		{
			name:   "single skip - visual",
			config: SkipConfig{Visual: true},
			want:   []string{"visual_verification"},
		},
		{
			name:   "multiple skips",
			config: SkipConfig{TestEvidence: true, GitDiff: true, Synthesis: true},
			want:   []string{"test_evidence", "git_diff", "synthesis"},
		},
		{
			name: "all skips",
			config: SkipConfig{
				TestEvidence:         true,
				Visual:               true,
				GitDiff:              true,
				Synthesis:            true,
				Build:                true,
				Constraint:           true,
				PhaseGate:            true,
				SkillOutput:          true,
				DecisionPatch:        true,
				PhaseComplete:        true,
				HandoffContent:       true,
				ExplainBack:          true,
				Accretion:            true,
				ArchitecturalChoices: true,
			},
			want: []string{
				"test_evidence",
				"visual_verification",
				"git_diff",
				"synthesis",
				"build",
				"constraint",
				"phase_gate",
				"skill_output",
				"decision_patch_limit",
				"phase_complete",
				"handoff_content",
				"explain_back",
				"accretion",
				"architectural_choices",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.SkippedGates()
			if len(got) != len(tt.want) {
				t.Errorf("SkippedGates() = %v, want %v", got, tt.want)
				return
			}
			for i, g := range got {
				if g != tt.want[i] {
					t.Errorf("SkippedGates()[%d] = %s, want %s", i, g, tt.want[i])
				}
			}
		})
	}
}

// TestSkipConfigShouldSkipGate tests the ShouldSkipGate method.
func TestSkipConfigShouldSkipGate(t *testing.T) {
	config := SkipConfig{
		TestEvidence: true,
		GitDiff:      true,
		Synthesis:    false,
		Build:        true,
	}

	tests := []struct {
		gate string
		want bool
	}{
		{"test_evidence", true},
		{"git_diff", true},
		{"build", true},
		{"synthesis", false},
		{"visual_verification", false},
		{"constraint", false},
		{"unknown_gate", false},
	}

	for _, tt := range tests {
		t.Run(tt.gate, func(t *testing.T) {
			got := config.ShouldSkipGate(tt.gate)
			if got != tt.want {
				t.Errorf("ShouldSkipGate(%s) = %v, want %v", tt.gate, got, tt.want)
			}
		})
	}
}

// TestValidateSkipFlags tests the skip flag validation logic.
func TestValidateSkipFlags(t *testing.T) {
	tests := []struct {
		name    string
		config  SkipConfig
		wantErr string
	}{
		{
			name:    "no skips - no error",
			config:  SkipConfig{},
			wantErr: "",
		},
		{
			name:    "no skips with reason - no error",
			config:  SkipConfig{Reason: "some reason"},
			wantErr: "",
		},
		{
			name:    "skip without reason - error",
			config:  SkipConfig{TestEvidence: true},
			wantErr: "--skip-reason is required when using --skip-* flags",
		},
		{
			name:    "skip with short reason - error",
			config:  SkipConfig{TestEvidence: true, Reason: "short"},
			wantErr: "--skip-reason must be at least 10 characters (got 5)",
		},
		{
			name:    "skip with 9 char reason - error",
			config:  SkipConfig{TestEvidence: true, Reason: "123456789"},
			wantErr: "--skip-reason must be at least 10 characters (got 9)",
		},
		{
			name:    "skip with 10 char reason - ok",
			config:  SkipConfig{TestEvidence: true, Reason: "1234567890"},
			wantErr: "",
		},
		{
			name:    "skip with long reason - ok",
			config:  SkipConfig{TestEvidence: true, Reason: "This is a valid reason for skipping the test evidence gate"},
			wantErr: "",
		},
		{
			name:    "multiple skips with valid reason - ok",
			config:  SkipConfig{TestEvidence: true, GitDiff: true, Reason: "Docs-only change"},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSkipFlags(tt.config)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("ValidateSkipFlags() unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateSkipFlags() expected error containing %q, got nil", tt.wantErr)
				} else if err.Error() != tt.wantErr {
					t.Errorf("ValidateSkipFlags() error = %q, want %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}
