package main

import (
	"testing"
)

func TestCheckBeadsIntegrity(t *testing.T) {
	status := checkBeadsIntegrity()

	// The check should always return a properly structured status
	if status.Name != "Beads DB Integrity" {
		t.Errorf("Expected name 'Beads DB Integrity', got %s", status.Name)
	}
	if status.CanFix {
		t.Error("Expected CanFix to be false (DB recovery is manual)")
	}
	if status.FixAction == "" {
		t.Error("Expected FixAction to be set")
	}

	// Running should be true (either DB is OK, or no DB exists, or can't get pwd)
	// Running = false only if actual corruption is detected
	// In test environment, we expect either "No beads database" or "Database integrity verified"
	if status.Details == "" {
		t.Error("Expected Details to be set")
	}
}

func TestCheckDockerBackend(t *testing.T) {
	status := checkDockerBackend()

	// The check should always return a properly structured status
	if status.Name != "Docker Backend" {
		t.Errorf("Expected name 'Docker Backend', got %s", status.Name)
	}
	if status.CanFix {
		t.Error("Expected CanFix to be false (Docker must be started manually)")
	}
	if status.FixAction == "" {
		t.Error("Expected FixAction to be set")
	}
	if status.Details == "" {
		t.Error("Expected Details to be set")
	}

	// If Docker is not installed, status.Running should be true (optional)
	// If Docker is installed, status.Running depends on whether Docker daemon is running
	// We don't assert Running value as it depends on environment
}

func TestCorrectnessChecksReturnValidStatus(t *testing.T) {
	// All correctness checks should return valid ServiceStatus
	// with at least Name and Details set

	checks := []func() ServiceStatus{
		checkBeadsIntegrity,
		checkDockerBackend,
	}

	checkNames := []string{
		"checkBeadsIntegrity",
		"checkDockerBackend",
	}

	for i, check := range checks {
		t.Run(checkNames[i], func(t *testing.T) {
			status := check()
			if status.Name == "" {
				t.Error("Expected Name to be set")
			}
			if status.Details == "" {
				t.Error("Expected Details to be set")
			}
			// Running can be true or false depending on environment
			// CanFix should always be false for correctness checks
			if status.CanFix {
				t.Error("Expected CanFix to be false for correctness checks")
			}
		})
	}
}
