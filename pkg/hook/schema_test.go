package hook

import (
	"testing"
)

func TestValidateOutput_EmptyAllow(t *testing.T) {
	result := ValidateOutput("PreToolUse", []byte(""), 0)
	if result.Decision != DecisionAllow {
		t.Errorf("expected ALLOW, got %s", result.Decision)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", result.Warnings)
	}
}

func TestValidateOutput_EmptyNonZeroExit(t *testing.T) {
	result := ValidateOutput("PreToolUse", []byte(""), 1)
	if result.Decision != DecisionBlock {
		t.Errorf("expected BLOCK, got %s", result.Decision)
	}
	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(result.Warnings))
	}
}

func TestValidateOutput_PreToolUse_ValidDeny(t *testing.T) {
	output := `{"hookSpecificOutput": {"hookEventName": "PreToolUse", "permissionDecision": "deny", "permissionDecisionReason": "blocked"}}`
	result := ValidateOutput("PreToolUse", []byte(output), 0)
	if result.Decision != DecisionDeny {
		t.Errorf("expected DENY, got %s", result.Decision)
	}
	if result.Reason != "blocked" {
		t.Errorf("expected reason 'blocked', got '%s'", result.Reason)
	}
	if !result.Valid {
		t.Error("expected valid output")
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", result.Warnings)
	}
}

func TestValidateOutput_PreToolUse_ValidAllow(t *testing.T) {
	output := `{"hookSpecificOutput": {"hookEventName": "PreToolUse", "permissionDecision": "allow", "additionalContext": "coaching nudge"}}`
	result := ValidateOutput("PreToolUse", []byte(output), 0)
	if result.Decision != DecisionAllow {
		t.Errorf("expected ALLOW, got %s", result.Decision)
	}
	if result.Context != "coaching nudge" {
		t.Errorf("expected context 'coaching nudge', got '%s'", result.Context)
	}
}

func TestValidateOutput_PreToolUse_RootLevelPermission(t *testing.T) {
	// This is the bug the architect investigation describes — permissionDecision at root level
	output := `{"permissionDecision": "deny", "permissionDecisionReason": "should be nested"}`
	result := ValidateOutput("PreToolUse", []byte(output), 0)
	if result.Valid {
		t.Error("expected invalid — permissionDecision at root level is wrong for PreToolUse")
	}
	if len(result.Warnings) < 1 {
		t.Error("expected warnings about root-level fields")
	}
	// Should mention hookSpecificOutput
	found := false
	for _, w := range result.Warnings {
		if containsStr(w, "hookSpecificOutput") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning mentioning hookSpecificOutput, got %v", result.Warnings)
	}
}

func TestValidateOutput_PreToolUse_PostToolUseFormat(t *testing.T) {
	// Common mistake: using PostToolUse format in PreToolUse
	output := `{"decision": "block"}`
	result := ValidateOutput("PreToolUse", []byte(output), 0)
	if result.Valid {
		t.Error("expected invalid — PostToolUse format used in PreToolUse")
	}
	if len(result.Warnings) < 1 {
		t.Error("expected warnings")
	}
}

func TestValidateOutput_PostToolUse_ValidBlock(t *testing.T) {
	output := `{"decision": "block"}`
	result := ValidateOutput("PostToolUse", []byte(output), 0)
	if result.Decision != DecisionBlock {
		t.Errorf("expected BLOCK, got %s", result.Decision)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", result.Warnings)
	}
}

func TestValidateOutput_PostToolUse_WithHSO(t *testing.T) {
	// Common mistake: PreToolUse format in PostToolUse
	output := `{"hookSpecificOutput": {"permissionDecision": "deny"}, "decision": "block"}`
	result := ValidateOutput("PostToolUse", []byte(output), 0)
	if result.Decision != DecisionBlock {
		t.Errorf("expected BLOCK, got %s", result.Decision)
	}
	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning about hookSpecificOutput in PostToolUse, got %d", len(result.Warnings))
	}
}

func TestValidateOutput_UserPromptSubmit(t *testing.T) {
	output := `{"additionalContext": "extra info for the prompt"}`
	result := ValidateOutput("UserPromptSubmit", []byte(output), 0)
	if result.Decision != DecisionAllow {
		t.Errorf("expected ALLOW, got %s", result.Decision)
	}
	if result.Context != "extra info for the prompt" {
		t.Errorf("expected context, got '%s'", result.Context)
	}
}

func TestValidateOutput_SessionStart(t *testing.T) {
	output := `{"hookSpecificOutput": {"hookEventName": "SessionStart"}}`
	result := ValidateOutput("SessionStart", []byte(output), 0)
	if result.Decision != DecisionAllow {
		t.Errorf("expected ALLOW for SessionStart, got %s", result.Decision)
	}
}

func TestValidateOutput_NonJSON(t *testing.T) {
	output := `just some text output`
	result := ValidateOutput("SessionStart", []byte(output), 0)
	if result.Decision != DecisionAllow {
		t.Errorf("expected ALLOW for non-JSON with exit 0, got %s", result.Decision)
	}
	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning about non-JSON, got %d", len(result.Warnings))
	}
}

func TestFormatExpectedSchema(t *testing.T) {
	schema := FormatExpectedSchema("PreToolUse")
	if !containsStr(schema, "hookSpecificOutput") {
		t.Error("PreToolUse schema should mention hookSpecificOutput")
	}
	if !containsStr(schema, "permissionDecision") {
		t.Error("PreToolUse schema should mention permissionDecision")
	}

	schema = FormatExpectedSchema("PostToolUse")
	if !containsStr(schema, "decision") {
		t.Error("PostToolUse schema should mention decision")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && searchStr(s, substr))
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
