package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// RunResult contains the result of executing a hook.
type RunResult struct {
	Hook        ResolvedHook
	ExitCode    int
	Stdout      string
	Stderr      string
	Duration    time.Duration
	Error       error
	Validation  *ValidationResult
}

// RunOptions configures how hooks are executed.
type RunOptions struct {
	// EnvOverrides sets/overrides environment variables for hook execution.
	EnvOverrides map[string]string
	// Input is the JSON input to send via stdin.
	Input map[string]interface{}
	// Timeout overrides the hook's configured timeout.
	Timeout time.Duration
	// DryRun shows what would happen without executing.
	DryRun bool
	// Verbose includes full JSON input/output in results.
	Verbose bool
}

// RunHook executes a single hook with the given input and options.
func RunHook(hook ResolvedHook, opts RunOptions) *RunResult {
	result := &RunResult{Hook: hook}

	if opts.DryRun {
		return result
	}

	// Determine timeout
	timeout := opts.Timeout
	if timeout == 0 && hook.Timeout > 0 {
		timeout = time.Duration(hook.Timeout) * time.Second
	}
	if timeout == 0 {
		timeout = 600 * time.Second // Claude Code default
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Build command
	cmd := exec.CommandContext(ctx, "sh", "-c", hook.ExpandedCmd)
	// Ensure child processes are killed on timeout
	cmd.WaitDelay = time.Second

	// Set environment
	cmd.Env = buildEnv(opts.EnvOverrides)

	// Set working directory
	if dir, ok := opts.EnvOverrides["CLAUDE_PROJECT_DIR"]; ok {
		cmd.Dir = dir
	} else {
		cmd.Dir, _ = os.Getwd()
	}

	// Prepare stdin
	inputJSON, err := json.Marshal(opts.Input)
	if err != nil {
		result.Error = fmt.Errorf("failed to marshal input: %w", err)
		return result
	}
	cmd.Stdin = bytes.NewReader(inputJSON)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	start := time.Now()
	runErr := cmd.Run()
	result.Duration = time.Since(start)
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()

	// Get exit code
	if runErr != nil {
		if ctx.Err() != nil {
			result.Error = fmt.Errorf("hook timed out after %v", timeout)
			return result
		}
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.Error = runErr
			return result
		}
	}

	// Validate output
	result.Validation = ValidateOutput(hook.Event, []byte(result.Stdout), result.ExitCode)

	return result
}

// BuildInput constructs the JSON input for a hook invocation.
// Merges common fields with event-specific fields from userInput.
func BuildInput(event, tool string, userInput map[string]interface{}) map[string]interface{} {
	input := map[string]interface{}{
		"hook_event_name": event,
		"session_id":      "test-session",
		"cwd":             getwd(),
		"permission_mode": "default",
	}

	// Add event-specific fields
	switch event {
	case "PreToolUse":
		input["tool_name"] = tool
		input["tool_input"] = map[string]interface{}{}
	case "PostToolUse":
		input["tool_name"] = tool
		input["tool_input"] = map[string]interface{}{}
		input["tool_response"] = ""
	case "UserPromptSubmit":
		input["prompt"] = ""
	}

	// Merge user-provided input (overrides defaults)
	for k, v := range userInput {
		input[k] = v
	}

	return input
}

// buildEnv constructs the environment for hook execution.
func buildEnv(overrides map[string]string) []string {
	// Start with current environment
	env := os.Environ()

	// Set defaults for Claude Code env vars
	home, _ := os.UserHomeDir()
	defaults := map[string]string{
		"CLAUDE_PROJECT_DIR": getwd(),
		"HOME":               home,
		"PATH":               os.Getenv("PATH"),
	}

	for k, v := range defaults {
		env = setEnv(env, k, v)
	}

	// Apply overrides
	for k, v := range overrides {
		env = setEnv(env, k, v)
	}

	return env
}

// setEnv sets or replaces an environment variable in a slice.
func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

// getwd returns the current working directory or "." if it fails.
func getwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}

// CommandBasename extracts the filename from a hook command path.
func CommandBasename(cmd string) string {
	expanded := expandCommand(cmd)
	return filepath.Base(expanded)
}
