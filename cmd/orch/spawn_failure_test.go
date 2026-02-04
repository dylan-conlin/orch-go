package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestParseFailureComment(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantIsRework    bool
		wantType        string
		wantDescription string
	}{
		{
			name:            "empty content after prefix",
			input:           "POST-COMPLETION-FAILURE:",
			wantIsRework:    true,
			wantType:        spawn.FailureTypeImplementation,
			wantDescription: "No failure details provided",
		},
		{
			name:            "verification failure with description",
			input:           "POST-COMPLETION-FAILURE: verification - Agent claimed tests pass but didn't run them",
			wantIsRework:    true,
			wantType:        spawn.FailureTypeVerification,
			wantDescription: "Agent claimed tests pass but didn't run them",
		},
		{
			name:            "implementation failure with description",
			input:           "POST-COMPLETION-FAILURE: implementation - Code doesn't work as expected",
			wantIsRework:    true,
			wantType:        spawn.FailureTypeImplementation,
			wantDescription: "Code doesn't work as expected",
		},
		{
			name:            "spec failure with description",
			input:           "POST-COMPLETION-FAILURE: spec - The spec was wrong",
			wantIsRework:    true,
			wantType:        spawn.FailureTypeSpec,
			wantDescription: "The spec was wrong",
		},
		{
			name:            "integration failure with description",
			input:           "POST-COMPLETION-FAILURE: integration - Works in isolation but fails in context",
			wantIsRework:    true,
			wantType:        spawn.FailureTypeIntegration,
			wantDescription: "Works in isolation but fails in context",
		},
		{
			name:            "no type specified defaults to implementation",
			input:           "POST-COMPLETION-FAILURE: Feature doesn't work properly",
			wantIsRework:    true,
			wantType:        spawn.FailureTypeImplementation,
			wantDescription: "Feature doesn't work properly",
		},
		{
			name:            "type with colon separator",
			input:           "POST-COMPLETION-FAILURE: verification: Tests not actually run",
			wantIsRework:    true,
			wantType:        spawn.FailureTypeVerification,
			wantDescription: "Tests not actually run",
		},
		{
			name:            "uppercase type still matches",
			input:           "POST-COMPLETION-FAILURE: VERIFICATION - Tests failed",
			wantIsRework:    true,
			wantType:        spawn.FailureTypeVerification,
			wantDescription: "Tests failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFailureComment(tt.input)

			if got == nil {
				t.Fatal("parseFailureComment returned nil, expected non-nil")
			}

			if got.IsRework != tt.wantIsRework {
				t.Errorf("IsRework = %v, want %v", got.IsRework, tt.wantIsRework)
			}

			if got.FailureType != tt.wantType {
				t.Errorf("FailureType = %q, want %q", got.FailureType, tt.wantType)
			}

			if got.Description != tt.wantDescription {
				t.Errorf("Description = %q, want %q", got.Description, tt.wantDescription)
			}

			// Verify SuggestedSkill is populated based on type
			expectedSkill := spawn.SuggestSkillForFailure(tt.wantType)
			if got.SuggestedSkill != expectedSkill {
				t.Errorf("SuggestedSkill = %q, want %q", got.SuggestedSkill, expectedSkill)
			}
		})
	}
}

func TestPostCompletionFailurePrefix(t *testing.T) {
	// Verify the prefix constant
	if PostCompletionFailurePrefix != "POST-COMPLETION-FAILURE:" {
		t.Errorf("PostCompletionFailurePrefix = %q, want %q", PostCompletionFailurePrefix, "POST-COMPLETION-FAILURE:")
	}
}
