package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// hookPath returns the path to enforce-phase-complete.py.
func enforcePhaseCompleteHookPath() string {
	home, _ := os.UserHomeDir()
	return home + "/.orch/hooks/enforce-phase-complete.py"
}

// runEnforcePhaseComplete runs the hook with the given env vars and stdin JSON.
// Returns stdout, stderr, and error.
func runEnforcePhaseComplete(t *testing.T, envOverrides map[string]string, inputJSON map[string]interface{}) (string, string, error) {
	t.Helper()

	hookPath := enforcePhaseCompleteHookPath()
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		t.Skip("enforce-phase-complete.py not found")
	}

	cmd := exec.Command("python3", hookPath)

	// Build env: start from clean slate with only needed vars
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
	}
	for k, v := range envOverrides {
		env = append(env, k+"="+v)
	}
	cmd.Env = env

	// Pipe input JSON to stdin
	inputBytes, err := json.Marshal(inputJSON)
	if err != nil {
		t.Fatalf("failed to marshal input JSON: %v", err)
	}
	cmd.Stdin = strings.NewReader(string(inputBytes))

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	return stdout.String(), stderr.String(), runErr
}

// TestEnforcePhaseCompleteBlocksSpawnedWorker verifies the hook blocks exit
// for a spawned worker that hasn't reported Phase: Complete.
// This is the baseline behavior that should be preserved.
func TestEnforcePhaseCompleteBlocksSpawnedWorker(t *testing.T) {
	input := map[string]interface{}{
		"stop_hook_active":       false,
		"last_assistant_message": "I've finished the implementation.",
	}
	env := map[string]string{
		"ORCH_SPAWNED":    "1",
		"CLAUDE_CONTEXT":  "worker",
		"ORCH_BEADS_ID":   "test-proj-xxxx",
	}

	stdout, _, err := runEnforcePhaseComplete(t, env, input)
	if err != nil {
		t.Fatalf("hook failed: %v", err)
	}

	// Should output a block decision
	if !strings.Contains(stdout, `"decision"`) {
		t.Errorf("expected block decision in stdout, got: %q", stdout)
	}
	if !strings.Contains(stdout, "block") {
		t.Errorf("expected 'block' in stdout, got: %q", stdout)
	}
}

// TestEnforcePhaseCompleteAllowsNonWorker verifies the hook allows exit
// for non-spawned sessions (ORCH_SPAWNED not set).
func TestEnforcePhaseCompleteAllowsNonWorker(t *testing.T) {
	input := map[string]interface{}{
		"stop_hook_active":       false,
		"last_assistant_message": "Hello.",
	}
	// No ORCH_SPAWNED or CLAUDE_CONTEXT
	env := map[string]string{}

	stdout, _, err := runEnforcePhaseComplete(t, env, input)
	if err != nil {
		t.Fatalf("hook failed: %v", err)
	}

	// Should produce no stdout (allow exit)
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("expected empty stdout (allow exit), got: %q", stdout)
	}
}

// TestEnforcePhaseCompleteAllowsPhaseCompleteMessage verifies the hook allows
// exit when the last message contains "Phase: Complete".
func TestEnforcePhaseCompleteAllowsPhaseCompleteMessage(t *testing.T) {
	input := map[string]interface{}{
		"stop_hook_active":       false,
		"last_assistant_message": "Phase: Complete - All tests passing.",
	}
	env := map[string]string{
		"ORCH_SPAWNED":   "1",
		"CLAUDE_CONTEXT": "worker",
		"ORCH_BEADS_ID":  "test-proj-xxxx",
	}

	stdout, _, err := runEnforcePhaseComplete(t, env, input)
	if err != nil {
		t.Fatalf("hook failed: %v", err)
	}

	if strings.TrimSpace(stdout) != "" {
		t.Errorf("expected empty stdout (allow exit), got: %q", stdout)
	}
}

// BUG TEST: This test demonstrates the bug where claude --print output gets
// contaminated by the Stop hook when called from within a spawned worker session.
//
// When a spawned worker runs `claude --print "test prompt"`, the child claude
// process inherits ORCH_SPAWNED=1 and CLAUDE_CONTEXT=worker from the parent.
// The Stop hook fires on the child's exit, blocks it (because no Phase: Complete
// was reported), and the block message creates a 2nd conversation turn that
// contaminates the --print output.
//
// Expected behavior after fix: When ORCH_PRINT_MODE=1 is set, the hook should
// skip enforcement and allow clean exit.
func TestEnforcePhaseCompleteSkipsPrintMode(t *testing.T) {
	input := map[string]interface{}{
		"stop_hook_active":       false,
		"last_assistant_message": "Here is the routing analysis for the prompt.",
	}
	// Simulates a claude --print call from within a spawned worker:
	// inherits ORCH_SPAWNED=1, CLAUDE_CONTEXT=worker, ORCH_BEADS_ID from parent,
	// but also has ORCH_PRINT_MODE=1 set by the caller.
	env := map[string]string{
		"ORCH_SPAWNED":    "1",
		"CLAUDE_CONTEXT":  "worker",
		"ORCH_BEADS_ID":   "test-proj-xxxx",
		"ORCH_PRINT_MODE": "1",
	}

	stdout, _, err := runEnforcePhaseComplete(t, env, input)
	if err != nil {
		t.Fatalf("hook failed: %v", err)
	}

	// After fix: ORCH_PRINT_MODE=1 should cause the hook to skip enforcement.
	// The hook should produce no stdout (allow exit), ensuring clean --print output.
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("BUG: hook blocked exit despite ORCH_PRINT_MODE=1.\n"+
			"This contaminates claude --print output and blocks all skill A/B testing.\n"+
			"Got stdout: %q\n"+
			"Expected: empty (allow exit)\n"+
			"Fix: Add ORCH_PRINT_MODE=1 check to enforce-phase-complete.py skip conditions",
			stdout)
	}
}

// TestEnforcePhaseCompleteStillBlocksWithoutPrintMode verifies that the hook
// still enforces Phase: Complete for real spawned workers (no ORCH_PRINT_MODE).
// This ensures the fix doesn't create a bypass for actual workers.
func TestEnforcePhaseCompleteStillBlocksWithoutPrintMode(t *testing.T) {
	input := map[string]interface{}{
		"stop_hook_active":       false,
		"last_assistant_message": "Done with the task.",
	}
	env := map[string]string{
		"ORCH_SPAWNED":   "1",
		"CLAUDE_CONTEXT": "worker",
		"ORCH_BEADS_ID":  "test-proj-xxxx",
		// No ORCH_PRINT_MODE — real worker session
	}

	stdout, _, err := runEnforcePhaseComplete(t, env, input)
	if err != nil {
		t.Fatalf("hook failed: %v", err)
	}

	// Should still block for real workers
	if !strings.Contains(stdout, "block") {
		t.Errorf("expected hook to block real worker without Phase: Complete, got: %q", stdout)
	}
}
