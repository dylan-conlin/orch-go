package main

import (
	"testing"
)

func TestPrintStep(t *testing.T) {
	tests := []struct {
		name   string
		step   DeployStep
		wantOK bool
	}{
		{
			name: "pending step",
			step: DeployStep{
				Name:   "Building binary",
				Status: "pending",
			},
			wantOK: true,
		},
		{
			name: "running step",
			step: DeployStep{
				Name:   "Building binary",
				Status: "running",
			},
			wantOK: true,
		},
		{
			name: "success step",
			step: DeployStep{
				Name:    "Building binary",
				Status:  "success",
				Message: "Built successfully",
			},
			wantOK: true,
		},
		{
			name: "failed step",
			step: DeployStep{
				Name:    "Building binary",
				Status:  "failed",
				Message: "make build failed",
			},
			wantOK: true,
		},
		{
			name: "skipped step",
			step: DeployStep{
				Name:    "Building binary",
				Status:  "skipped",
				Message: "Skipped (--skip-build)",
			},
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This just tests that printStep doesn't panic
			printStep(tt.step)
		})
	}
}

func TestIsPortResponding(t *testing.T) {
	// Test with a port that's definitely not listening
	if isPortResponding(59999) {
		t.Error("expected port 59999 to not be responding")
	}

	// Note: We can't easily test a responding port in unit tests
	// since we'd need to start a server. This is tested via integration testing.
}

func TestFindOrchProjectDir(t *testing.T) {
	// Test that the function returns something (may be empty in test environment)
	result := findOrchProjectDir()
	// In test environment, it should at least try to find the Procfile
	// The function should not panic
	t.Logf("findOrchProjectDir returned: %q", result)
}
