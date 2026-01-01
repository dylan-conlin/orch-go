package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestSessionEndFlags(t *testing.T) {
	t.Run("--skip-reflection without --reason returns error", func(t *testing.T) {
		// Reset global flags for test isolation
		sessionEndSkipReflection = true
		sessionEndSkipReason = ""
		sessionEndNoHandoff = false
		
		// Create a command and execute the RunE function directly
		cmd := sessionEndCmd
		err := cmd.RunE(cmd, []string{})
		
		if err == nil {
			t.Error("Expected error when --skip-reflection used without --reason")
		}
		if err != nil && !strings.Contains(err.Error(), "--skip-reflection requires --reason") {
			t.Errorf("Expected error about --reason being required, got: %v", err)
		}
		
		// Clean up
		sessionEndSkipReflection = false
	})
	
	t.Run("--skip-reflection with --reason succeeds", func(t *testing.T) {
		// Reset global flags for test isolation
		sessionEndSkipReflection = true
		sessionEndSkipReason = "quick context switch"
		sessionEndNoHandoff = false
		
		// Create a command and execute the RunE function directly
		cmd := sessionEndCmd
		err := cmd.RunE(cmd, []string{})
		
		// This will fail with "No active session to end" which is expected behavior
		// The validation passed if we got past the flag check
		if err != nil && strings.Contains(err.Error(), "--skip-reflection requires --reason") {
			t.Error("Validation should pass when --reason is provided")
		}
		
		// Clean up
		sessionEndSkipReflection = false
		sessionEndSkipReason = ""
	})
	
	t.Run("deprecated --no-handoff sets skip-reflection", func(t *testing.T) {
		// Reset global flags for test isolation
		sessionEndNoHandoff = true
		sessionEndSkipReflection = false
		sessionEndSkipReason = ""
		
		// Create a command and execute the RunE function directly
		cmd := sessionEndCmd
		_ = cmd.RunE(cmd, []string{})
		
		// After RunE, sessionEndSkipReflection should be set to true
		if !sessionEndSkipReflection {
			t.Error("--no-handoff should set sessionEndSkipReflection to true")
		}
		
		// And reason should be set to deprecated message
		if sessionEndSkipReason == "" {
			t.Error("--no-handoff should set a default skip reason")
		}
		
		// Clean up
		sessionEndNoHandoff = false
		sessionEndSkipReflection = false
		sessionEndSkipReason = ""
	})
}

func TestSessionEndCmdHelp(t *testing.T) {
	// Verify the help text contains the new flag info
	var buf bytes.Buffer
	sessionEndCmd.SetOut(&buf)
	sessionEndCmd.Help()
	
	helpText := buf.String()
	
	if !strings.Contains(helpText, "--skip-reflection") {
		t.Error("Help text should mention --skip-reflection flag")
	}
	
	if !strings.Contains(helpText, "--reason") {
		t.Error("Help text should mention --reason flag")
	}
	
	if !strings.Contains(helpText, "Gate Over Remind") {
		t.Error("Help text should reference Gate Over Remind principle")
	}
}
