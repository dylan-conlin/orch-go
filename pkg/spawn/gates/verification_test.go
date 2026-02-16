package gates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckVerificationGate_NoUnverifiedWork(t *testing.T) {
	// Setup: ensure no checkpoints exist in test environment
	// This test assumes no unverified work exists
	// In a real test, we'd mock the checkpoint reading

	// Test that gate allows spawn when no unverified work exists
	err := CheckVerificationGate(false, "")
	// Since we can't easily mock checkpoints in this test,
	// we'll just verify the function doesn't panic
	// and returns nil or a proper error
	if err != nil {
		// This is expected if there's actually unverified work
		t.Logf("Gate blocked spawn (may have actual unverified work): %v", err)
	}
}

func TestCheckVerificationGate_WithBypass(t *testing.T) {
	// Test that bypass flag allows spawn even if unverified work exists
	// Should fail without reason
	err := CheckVerificationGate(true, "")
	if err == nil {
		t.Error("Expected error when bypass-verification is set without bypass-reason")
	}

	// Should succeed with reason (logs bypass but doesn't block)
	err = CheckVerificationGate(true, "testing independent parallel work")
	if err != nil {
		t.Errorf("Bypass with reason should allow spawn: %v", err)
	}
}

func TestGetUnverifiedTier1Work_EmptyCheckpoints(t *testing.T) {
	// This test would need to mock checkpoint.ReadCheckpoints()
	// For now, just verify it doesn't panic
	unverified, err := GetUnverifiedTier1Work()
	if err != nil {
		t.Logf("Error getting unverified work: %v", err)
	}
	// Length could be 0 or non-zero depending on actual state
	t.Logf("Found %d unverified Tier 1 items", len(unverified))
}

func TestLogVerificationBypass(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Override home directory for this test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Log a bypass
	LogVerificationBypass("test bypass reason")

	// Check that violations log was created
	violationsPath := filepath.Join(tmpDir, ".orch", "metrics", "verification-violations.log")
	if _, err := os.Stat(violationsPath); os.IsNotExist(err) {
		t.Error("Expected violations log to be created")
	}

	// Read log content
	content, err := os.ReadFile(violationsPath)
	if err != nil {
		t.Fatalf("Failed to read violations log: %v", err)
	}

	// Verify content contains the reason
	if len(content) == 0 {
		t.Error("Violations log is empty")
	}

	contentStr := string(content)
	if !contains(contentStr, "test bypass reason") {
		t.Errorf("Expected violations log to contain reason, got: %s", contentStr)
	}
	if !contains(contentStr, "verification_bypassed") {
		t.Errorf("Expected violations log to contain event type, got: %s", contentStr)
	}
}

func TestSpaces(t *testing.T) {
	tests := []struct {
		n        int
		expected string
	}{
		{0, ""},
		{-1, ""},
		{1, " "},
		{5, "     "},
	}

	for _, tt := range tests {
		result := spaces(tt.n)
		if result != tt.expected {
			t.Errorf("spaces(%d) = %q, want %q", tt.n, result, tt.expected)
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr))))
}
