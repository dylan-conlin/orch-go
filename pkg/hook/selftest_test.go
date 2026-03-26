package hook

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSelfTest_AllPass(t *testing.T) {
	dir := t.TempDir()

	// Create a passing hook script
	script := filepath.Join(dir, "pass-hook.sh")
	content := `#!/bin/sh
cat <<'EOF'
{"hookSpecificOutput": {"permissionDecision": "allow"}}
EOF`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	settings := &Settings{
		Hooks: map[string][]HookGroup{
			"PreToolUse": {
				{
					Matcher: "Bash",
					Hooks: []HookConfig{
						{Type: "command", Command: script, Timeout: 5},
					},
				},
			},
		},
	}

	results := SelfTest(settings, SelfTestOptions{Timeout: 5})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Passed {
		t.Errorf("expected pass, got fail: %s", results[0].Summary)
	}
}

func TestSelfTest_DetectsMissingScript(t *testing.T) {
	settings := &Settings{
		Hooks: map[string][]HookGroup{
			"PreToolUse": {
				{
					Matcher: "Bash",
					Hooks: []HookConfig{
						{Type: "command", Command: "/nonexistent/hook.sh", Timeout: 5},
					},
				},
			},
		},
	}

	results := SelfTest(settings, SelfTestOptions{Timeout: 5})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Passed {
		t.Error("expected fail for missing script")
	}
}

func TestSelfTest_DetectsInvalidOutput(t *testing.T) {
	dir := t.TempDir()

	// Create a hook that outputs wrong format for PreToolUse
	script := filepath.Join(dir, "bad-format.sh")
	content := `#!/bin/sh
echo '{"decision": "allow"}'`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	settings := &Settings{
		Hooks: map[string][]HookGroup{
			"PreToolUse": {
				{
					Matcher: "Bash",
					Hooks: []HookConfig{
						{Type: "command", Command: script, Timeout: 5},
					},
				},
			},
		},
	}

	results := SelfTest(settings, SelfTestOptions{Timeout: 5})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// Should warn about format but not necessarily fail (exit 0 = allow)
	if len(results[0].Warnings) == 0 {
		t.Error("expected warnings about output format")
	}
}

func TestSelfTest_MultipleHooks(t *testing.T) {
	dir := t.TempDir()

	script := filepath.Join(dir, "ok-hook.sh")
	content := `#!/bin/sh
echo '{}'`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	settings := &Settings{
		Hooks: map[string][]HookGroup{
			"PreToolUse": {
				{
					Matcher: "Bash",
					Hooks: []HookConfig{
						{Type: "command", Command: script, Timeout: 5},
					},
				},
			},
			"UserPromptSubmit": {
				{
					Hooks: []HookConfig{
						{Type: "command", Command: script, Timeout: 5},
					},
				},
			},
		},
	}

	results := SelfTest(settings, SelfTestOptions{Timeout: 5})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestSelfTest_NonZeroExitFails(t *testing.T) {
	dir := t.TempDir()

	script := filepath.Join(dir, "exit1.sh")
	content := `#!/bin/sh
exit 1`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	settings := &Settings{
		Hooks: map[string][]HookGroup{
			"PreToolUse": {
				{
					Matcher: "Bash",
					Hooks: []HookConfig{
						{Type: "command", Command: script, Timeout: 5},
					},
				},
			},
		},
	}

	results := SelfTest(settings, SelfTestOptions{Timeout: 5})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Passed {
		t.Error("expected fail for non-zero exit")
	}
}

func TestSelfTestSummary(t *testing.T) {
	results := []SelfTestResult{
		{Passed: true, HookName: "hook1"},
		{Passed: true, HookName: "hook2"},
		{Passed: false, HookName: "hook3", Summary: "missing script"},
	}

	summary := FormatSelfTestSummary(results)
	if summary.Total != 3 {
		t.Errorf("expected total 3, got %d", summary.Total)
	}
	if summary.Passed != 2 {
		t.Errorf("expected 2 passed, got %d", summary.Passed)
	}
	if summary.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", summary.Failed)
	}
	if summary.AllPassed {
		t.Error("expected AllPassed false")
	}
}
