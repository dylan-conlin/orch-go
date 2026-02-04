package spawn

import (
	"testing"
)

func TestSuggestSkillForFailure(t *testing.T) {
	tests := []struct {
		name         string
		failureType  string
		expectedSkill string
	}{
		{
			name:          "verification failure",
			failureType:   FailureTypeVerification,
			expectedSkill: "reliability-testing",
		},
		{
			name:          "implementation failure",
			failureType:   FailureTypeImplementation,
			expectedSkill: "systematic-debugging",
		},
		{
			name:          "spec failure",
			failureType:   FailureTypeSpec,
			expectedSkill: "investigation",
		},
		{
			name:          "integration failure",
			failureType:   FailureTypeIntegration,
			expectedSkill: "reliability-testing",
		},
		{
			name:          "unknown failure type",
			failureType:   "unknown",
			expectedSkill: "systematic-debugging",
		},
		{
			name:          "empty failure type",
			failureType:   "",
			expectedSkill: "systematic-debugging",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SuggestSkillForFailure(tt.failureType)
			if got != tt.expectedSkill {
				t.Errorf("SuggestSkillForFailure(%q) = %q, want %q", tt.failureType, got, tt.expectedSkill)
			}
		})
	}
}

func TestFailureContextConstants(t *testing.T) {
	// Verify constants have expected values
	if FailureTypeVerification != "verification" {
		t.Errorf("FailureTypeVerification = %q, want %q", FailureTypeVerification, "verification")
	}
	if FailureTypeImplementation != "implementation" {
		t.Errorf("FailureTypeImplementation = %q, want %q", FailureTypeImplementation, "implementation")
	}
	if FailureTypeSpec != "spec" {
		t.Errorf("FailureTypeSpec = %q, want %q", FailureTypeSpec, "spec")
	}
	if FailureTypeIntegration != "integration" {
		t.Errorf("FailureTypeIntegration = %q, want %q", FailureTypeIntegration, "integration")
	}
}
