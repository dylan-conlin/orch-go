package hook

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildInput_PreToolUse(t *testing.T) {
	input := BuildInput("PreToolUse", "Bash", nil)

	if input["hook_event_name"] != "PreToolUse" {
		t.Errorf("expected hook_event_name 'PreToolUse', got '%v'", input["hook_event_name"])
	}
	if input["tool_name"] != "Bash" {
		t.Errorf("expected tool_name 'Bash', got '%v'", input["tool_name"])
	}
	if _, ok := input["session_id"]; !ok {
		t.Error("expected session_id field")
	}
	if _, ok := input["permission_mode"]; !ok {
		t.Error("expected permission_mode field")
	}
}

func TestBuildInput_WithOverrides(t *testing.T) {
	userInput := map[string]interface{}{
		"command": "bd close orch-go-1234",
	}
	input := BuildInput("PreToolUse", "Bash", userInput)

	if input["command"] != "bd close orch-go-1234" {
		t.Errorf("expected user input to be merged, got '%v'", input["command"])
	}
}

func TestBuildInput_PostToolUse(t *testing.T) {
	input := BuildInput("PostToolUse", "Bash", nil)
	if _, ok := input["tool_response"]; !ok {
		t.Error("expected tool_response field for PostToolUse")
	}
}

func TestBuildInput_SessionStart(t *testing.T) {
	input := BuildInput("SessionStart", "", nil)
	if input["hook_event_name"] != "SessionStart" {
		t.Errorf("expected SessionStart, got %v", input["hook_event_name"])
	}
}

func TestRunHook_DryRun(t *testing.T) {
	hook := ResolvedHook{
		Event:       "PreToolUse",
		Command:     "echo test",
		ExpandedCmd: "echo test",
	}

	result := RunHook(hook, RunOptions{DryRun: true})
	if result.Error != nil {
		t.Errorf("dry run should not error: %v", result.Error)
	}
	if result.Stdout != "" {
		t.Errorf("dry run should not produce output, got '%s'", result.Stdout)
	}
}

func TestRunHook_SimpleScript(t *testing.T) {
	// Create a temp script that outputs valid PreToolUse JSON
	dir := t.TempDir()
	script := filepath.Join(dir, "test-hook.sh")
	content := `#!/bin/sh
cat <<'EOF'
{"hookSpecificOutput": {"hookEventName": "PreToolUse", "permissionDecision": "allow", "additionalContext": "test context"}}
EOF`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	hook := ResolvedHook{
		Event:       "PreToolUse",
		Command:     script,
		ExpandedCmd: script,
		Timeout:     5,
	}

	result := RunHook(hook, RunOptions{
		Input: BuildInput("PreToolUse", "Bash", nil),
	})

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if result.Validation == nil {
		t.Fatal("expected validation result")
	}
	if result.Validation.Decision != DecisionAllow {
		t.Errorf("expected ALLOW, got %s", result.Validation.Decision)
	}
	if result.Validation.Context != "test context" {
		t.Errorf("expected context 'test context', got '%s'", result.Validation.Context)
	}
}

func TestRunHook_DenyScript(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "deny-hook.sh")
	content := `#!/bin/sh
cat <<'EOF'
{"hookSpecificOutput": {"hookEventName": "PreToolUse", "permissionDecision": "deny", "permissionDecisionReason": "not allowed"}}
EOF`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	hook := ResolvedHook{
		Event:       "PreToolUse",
		Command:     script,
		ExpandedCmd: script,
		Timeout:     5,
	}

	result := RunHook(hook, RunOptions{
		Input: BuildInput("PreToolUse", "Bash", nil),
	})

	if result.Validation.Decision != DecisionDeny {
		t.Errorf("expected DENY, got %s", result.Validation.Decision)
	}
	if result.Validation.Reason != "not allowed" {
		t.Errorf("expected reason 'not allowed', got '%s'", result.Validation.Reason)
	}
}

func TestRunHook_NonZeroExit(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fail-hook.sh")
	content := `#!/bin/sh
exit 1`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	hook := ResolvedHook{
		Event:       "PreToolUse",
		Command:     script,
		ExpandedCmd: script,
		Timeout:     5,
	}

	result := RunHook(hook, RunOptions{
		Input: BuildInput("PreToolUse", "Bash", nil),
	})

	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
}

func TestRunHook_Timeout(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "slow-hook.sh")
	content := `#!/bin/sh
sleep 10`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	hook := ResolvedHook{
		Event:       "PreToolUse",
		Command:     script,
		ExpandedCmd: script,
	}

	result := RunHook(hook, RunOptions{
		Input:   BuildInput("PreToolUse", "Bash", nil),
		Timeout: 100 * time.Millisecond,
	})

	if result.Error == nil {
		t.Error("expected timeout error")
	}
}

func TestRunHook_EnvOverrides(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "env-hook.sh")
	content := `#!/bin/sh
echo "{\"context\": \"$CLAUDE_CONTEXT\"}"
`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	hook := ResolvedHook{
		Event:       "PreToolUse",
		Command:     script,
		ExpandedCmd: script,
		Timeout:     5,
	}

	result := RunHook(hook, RunOptions{
		Input:        BuildInput("PreToolUse", "Bash", nil),
		EnvOverrides: map[string]string{"CLAUDE_CONTEXT": "orchestrator"},
	})

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.Stdout == "" {
		t.Error("expected output with CLAUDE_CONTEXT")
	}
}

func TestRunHookWithLogging_WritesTrace(t *testing.T) {
	dir := t.TempDir()
	tracePath := filepath.Join(dir, "trace.jsonl")

	// Create a simple hook
	script := filepath.Join(dir, "log-test.sh")
	content := `#!/bin/sh
echo '{}'`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	hook := ResolvedHook{
		Event:       "PreToolUse",
		Command:     script,
		ExpandedCmd: script,
		Matcher:     "Bash",
		Timeout:     5,
	}

	result := RunHookWithLogging(hook, RunOptions{
		Input: BuildInput("PreToolUse", "Bash", nil),
	}, "test-session-123", tracePath)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}

	// Verify trace was written
	entries, err := ReadTrace(tracePath, TraceOptions{})
	if err != nil {
		t.Fatalf("failed to read trace: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 trace entry, got %d", len(entries))
	}
	if entries[0].Session != "test-session-123" {
		t.Errorf("expected session 'test-session-123', got '%s'", entries[0].Session)
	}
	if entries[0].Event != "PreToolUse" {
		t.Errorf("expected event 'PreToolUse', got '%s'", entries[0].Event)
	}
}

func TestRunHookWithLogging_DryRunSkipsTrace(t *testing.T) {
	dir := t.TempDir()
	tracePath := filepath.Join(dir, "trace.jsonl")

	hook := ResolvedHook{
		Event:       "PreToolUse",
		Command:     "echo test",
		ExpandedCmd: "echo test",
	}

	RunHookWithLogging(hook, RunOptions{DryRun: true}, "sess", tracePath)

	// Trace file should not exist
	if _, err := os.Stat(tracePath); !os.IsNotExist(err) {
		t.Error("expected no trace file for dry run")
	}
}

func TestCommandBasename(t *testing.T) {
	tests := []struct {
		cmd  string
		want string
	}{
		{"$HOME/.orch/hooks/gate.py", "gate.py"},
		{"/usr/bin/test.sh", "test.sh"},
		{"bd prime", "bd prime"},
	}

	for _, tt := range tests {
		got := CommandBasename(tt.cmd)
		if got != tt.want {
			t.Errorf("CommandBasename(%q) = %q, want %q", tt.cmd, got, tt.want)
		}
	}
}
