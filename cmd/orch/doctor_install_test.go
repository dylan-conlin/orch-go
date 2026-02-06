package main

import (
	"testing"
)

func TestGetDoctorPlistPath(t *testing.T) {
	path := getDoctorPlistPath()
	if path == "" {
		t.Error("Expected non-empty plist path")
	}
	// Check that path contains expected filename
	expected := "com.orch.doctor.plist"
	found := false
	for i := 0; i <= len(path)-len(expected); i++ {
		if path[i:i+len(expected)] == expected {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected path to contain '%s', got %s", expected, path)
	}
}
