package hook

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Decision represents the interpreted decision from a hook's output.
type Decision string

const (
	DecisionAllow Decision = "ALLOW"
	DecisionDeny  Decision = "DENY"
	DecisionBlock Decision = "BLOCK"
	DecisionAsk   Decision = "ASK"
	DecisionNone  Decision = "NONE"
)

// ValidationResult contains the parsed and validated output from a hook execution.
type ValidationResult struct {
	Decision Decision
	Reason   string
	Context  string // additionalContext if present
	Warnings []string
	Valid    bool
	Raw      map[string]interface{} // Raw parsed JSON
}

// ValidateOutput parses hook output and validates it against the expected schema
// for the given event type. Returns validation result with any format warnings.
func ValidateOutput(event string, stdout []byte, exitCode int) *ValidationResult {
	result := &ValidationResult{
		Valid: true,
	}

	// Empty output with exit 0 = allow (implicit)
	trimmed := strings.TrimSpace(string(stdout))
	if trimmed == "" {
		if exitCode == 0 {
			result.Decision = DecisionAllow
			result.Reason = "no output"
			return result
		}
		// Non-zero exit with no output = error
		result.Decision = DecisionBlock
		result.Reason = fmt.Sprintf("exit code %d with no output", exitCode)
		result.Warnings = append(result.Warnings, "Non-zero exit code without JSON output — Claude Code will block the tool use")
		return result
	}

	// Try to parse as JSON
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		// Non-JSON output
		result.Raw = nil
		if exitCode == 0 {
			result.Decision = DecisionAllow
			result.Reason = "non-JSON output (treated as stdout text)"
			result.Context = trimmed
		} else {
			result.Decision = DecisionBlock
			result.Reason = fmt.Sprintf("exit code %d with non-JSON output", exitCode)
		}
		result.Warnings = append(result.Warnings, "Output is not valid JSON — Claude Code may not parse hook-specific fields")
		return result
	}
	result.Raw = raw

	// Route to event-specific validation
	switch event {
	case "PreToolUse":
		validatePreToolUseOutput(result, raw)
	case "PostToolUse":
		validatePostToolUseOutput(result, raw)
	case "UserPromptSubmit":
		validateUserPromptSubmitOutput(result, raw)
	case "Stop":
		validateStopOutput(result, raw)
	default:
		// For other events (SessionStart, SessionEnd, PreCompact),
		// there's no specific output schema — just check for basic fields
		validateGenericOutput(result, raw)
	}

	return result
}

// validatePreToolUseOutput validates output for PreToolUse hooks.
// Expected format: {"hookSpecificOutput": {"hookEventName": "PreToolUse", "permissionDecision": "allow|deny", ...}}
func validatePreToolUseOutput(result *ValidationResult, raw map[string]interface{}) {
	// Check for hookSpecificOutput
	hso, hasHSO := raw["hookSpecificOutput"]
	if !hasHSO {
		// Check for common mistakes: fields at root level instead of nested
		if _, hasDecision := raw["permissionDecision"]; hasDecision {
			result.Warnings = append(result.Warnings,
				"'permissionDecision' found at root level — for PreToolUse, this must be inside 'hookSpecificOutput'. Claude Code will IGNORE this field.")
			result.Valid = false
		}
		if _, hasReason := raw["permissionDecisionReason"]; hasReason {
			result.Warnings = append(result.Warnings,
				"'permissionDecisionReason' found at root level — for PreToolUse, this must be inside 'hookSpecificOutput'. Claude Code will IGNORE this field.")
			result.Valid = false
		}

		// Check for top-level decision (PostToolUse pattern used for PreToolUse)
		if _, hasTopDecision := raw["decision"]; hasTopDecision {
			result.Warnings = append(result.Warnings,
				"'decision' field found at root level — this is the PostToolUse format. PreToolUse uses 'hookSpecificOutput.permissionDecision' instead.")
			result.Valid = false
		}

		if !result.Valid {
			result.Decision = DecisionNone
			result.Reason = "output format mismatch (see warnings)"
			return
		}

		// No hookSpecificOutput and no root-level fields = allow
		result.Decision = DecisionAllow
		result.Reason = "no hookSpecificOutput"
		return
	}

	hsoMap, ok := hso.(map[string]interface{})
	if !ok {
		result.Warnings = append(result.Warnings, "'hookSpecificOutput' is not an object")
		result.Valid = false
		result.Decision = DecisionNone
		return
	}

	// Check hookEventName
	if eventName, has := hsoMap["hookEventName"]; has {
		if eventStr, ok := eventName.(string); ok && eventStr != "PreToolUse" {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("hookEventName is '%s' but expected 'PreToolUse'", eventStr))
		}
	}

	// Extract permissionDecision
	if pd, has := hsoMap["permissionDecision"]; has {
		pdStr, _ := pd.(string)
		switch strings.ToLower(pdStr) {
		case "allow":
			result.Decision = DecisionAllow
		case "deny":
			result.Decision = DecisionDeny
		case "ask":
			result.Decision = DecisionAsk
		default:
			result.Decision = DecisionNone
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("unrecognized permissionDecision: '%s' (expected: allow, deny, ask)", pdStr))
		}
	} else {
		result.Decision = DecisionAllow
		result.Reason = "no permissionDecision in hookSpecificOutput"
	}

	// Extract reason
	if reason, has := hsoMap["permissionDecisionReason"]; has {
		result.Reason, _ = reason.(string)
	}

	// Extract additionalContext
	if ctx, has := hsoMap["additionalContext"]; has {
		result.Context, _ = ctx.(string)
	}
}

// validatePostToolUseOutput validates output for PostToolUse hooks.
// Expected format: {"decision": "block"} or exit code based.
func validatePostToolUseOutput(result *ValidationResult, raw map[string]interface{}) {
	// Check for top-level decision
	if decision, has := raw["decision"]; has {
		dStr, _ := decision.(string)
		switch strings.ToLower(dStr) {
		case "block":
			result.Decision = DecisionBlock
		default:
			result.Decision = DecisionAllow
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("unrecognized decision value: '%s' (PostToolUse only supports 'block')", dStr))
		}
	} else {
		result.Decision = DecisionAllow
	}

	// Check for PreToolUse-style output used in PostToolUse (common mistake)
	if _, hasHSO := raw["hookSpecificOutput"]; hasHSO {
		result.Warnings = append(result.Warnings,
			"'hookSpecificOutput' found in PostToolUse output — PostToolUse uses top-level 'decision' field, not hookSpecificOutput")
	}

	// Extract reason
	if reason, has := raw["reason"]; has {
		result.Reason, _ = reason.(string)
	}
}

// validateUserPromptSubmitOutput validates output for UserPromptSubmit hooks.
// Expected format: {"additionalContext": "..."} at root level.
func validateUserPromptSubmitOutput(result *ValidationResult, raw map[string]interface{}) {
	if ctx, has := raw["additionalContext"]; has {
		result.Context, _ = ctx.(string)
		result.Decision = DecisionAllow
	}

	// Check for decision
	if decision, has := raw["decision"]; has {
		dStr, _ := decision.(string)
		if strings.ToLower(dStr) == "block" {
			result.Decision = DecisionBlock
		}
	}

	if result.Decision == "" {
		result.Decision = DecisionAllow
	}
}

// validateStopOutput validates output for Stop hooks.
// Expected format: {"decision": "block", "reason": "..."} or exit 0 with no output to allow.
// Claude Code only recognizes "block" for Stop hooks. To allow stopping, exit 0 with no output.
func validateStopOutput(result *ValidationResult, raw map[string]interface{}) {
	// Check for top-level decision
	if decision, has := raw["decision"]; has {
		dStr, _ := decision.(string)
		switch strings.ToLower(dStr) {
		case "block":
			result.Decision = DecisionBlock
		default:
			result.Decision = DecisionAllow
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("unrecognized decision value: '%s' (Stop only supports 'block'; to allow stopping, exit 0 with no output)", dStr))
		}
	} else {
		result.Decision = DecisionAllow
	}

	// Check for PreToolUse-style output used in Stop (common mistake)
	if _, hasHSO := raw["hookSpecificOutput"]; hasHSO {
		result.Warnings = append(result.Warnings,
			"'hookSpecificOutput' found in Stop output — Stop uses top-level 'decision' field, not hookSpecificOutput")
	}

	// Extract reason
	if reason, has := raw["reason"]; has {
		result.Reason, _ = reason.(string)
	}
}

// validateGenericOutput validates output for events without specific schemas.
func validateGenericOutput(result *ValidationResult, raw map[string]interface{}) {
	// Check for common output patterns
	if _, has := raw["hookSpecificOutput"]; has {
		// Has hookSpecificOutput
		hso, ok := raw["hookSpecificOutput"].(map[string]interface{})
		if ok {
			if pd, has := hso["permissionDecision"]; has {
				pdStr, _ := pd.(string)
				switch strings.ToLower(pdStr) {
				case "allow":
					result.Decision = DecisionAllow
				case "deny":
					result.Decision = DecisionDeny
				}
			}
		}
	}

	if result.Decision == "" {
		result.Decision = DecisionAllow
	}
}

// FormatExpectedSchema returns a human-readable description of the expected
// output format for a given event type.
func FormatExpectedSchema(event string) string {
	switch event {
	case "PreToolUse":
		return `Expected PreToolUse output format:
  {"hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow" | "deny" | "ask",
    "permissionDecisionReason": "...",
    "additionalContext": "..."
  }}`
	case "PostToolUse":
		return `Expected PostToolUse output format:
  {"decision": "block"}
  or exit code 2 to block`
	case "UserPromptSubmit":
		return `Expected UserPromptSubmit output format:
  {"additionalContext": "..."}
  or {"decision": "block"}`
	case "Stop":
		return `Expected Stop output format:
  {"decision": "block", "reason": "..."}
  or exit 0 with no output to allow stopping`
	default:
		return fmt.Sprintf("No specific output schema for %s — text output is passed as context", event)
	}
}
